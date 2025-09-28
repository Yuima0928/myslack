// src/components/MessageList.tsx
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api, type Msg} from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { useWS } from "../hooks/useWS";
import Avatar from "./Avatar"; // ★ 追加

export default function MessageList({ channelId }: { channelId: string | null }) {
  const { isAuthenticated, isLoading } = useAuth();
  const [msgs, setMsgs] = useState<Msg[]>([]);
  const scRef = useRef<HTMLDivElement>(null);

  // file_id -> url の簡易キャッシュ
  const [avatarURLCache] = useState(() => new Map<string, string>());

  const resolveAvatarURL = useCallback(
    async (fileId?: string | null): Promise<string | null> => {
      if (!fileId) return null;
      const cached = avatarURLCache.get(fileId);
      if (cached) return cached;
      try {
        const { url } = await api.getFileURL(fileId, "inline");
        avatarURLCache.set(fileId, url);
        return url;
      } catch {
        return null;
      }
    },
    [avatarURLCache]
  );

  // 初回ロード
  useEffect(() => {
    if (!channelId) { setMsgs([]); return; }
    if (isLoading || !isAuthenticated) return;

    (async () => {
      const res = await api.listMessages(channelId);
      setMsgs(res);
      requestAnimationFrame(() => scRef.current?.scrollTo({ top: 1e9 }));
    })();
  }, [channelId, isAuthenticated, isLoading]);

  // WSイベント
  const onWsEvent = useCallback((ev: unknown) => {
    if (typeof ev === "object" && ev !== null && "type" in (ev as any)) {
      const e = ev as { type: string; message?: Msg };
      if (e.type === "message_created" && e.message) {
        if (!channelId || e.message.channel_id !== channelId) return;
        // ★ created_at が無ければ今の時刻で補完
        const normalized = {
          ...e.message,
          created_at: e.message.created_at ?? new Date().toISOString(),
        };
        setMsgs(prev => {
          if (prev.some(m => m.id === normalized.id)) return prev;
          const next = [...prev, normalized];
          requestAnimationFrame(() => scRef.current?.scrollTo({ top: 1e9 }));
          return next;
        });
      }
    }
  }, [channelId]);

  useWS(channelId, onWsEvent);

  return (
    <div className="messages" ref={scRef}>
      {msgs.length === 0 && <div style={{ color: "#9ca3af" }}>No messages</div>}
      {msgs.map(m => (
        <MessageRow
          key={m.id}
          msg={m}
          resolveAvatarURL={resolveAvatarURL}
        />
      ))}
    </div>
  );
}

function MessageRow({
  msg,
  resolveAvatarURL,
}: {
  msg: Msg;
  resolveAvatarURL: (fid?: string | null) => Promise<string | null>;
}) {
  const [avatarURL, setAvatarURL] = useState<string | null>(null);

  // 1) サーバが直接 URL を返してくれていればそれを使う
  // 2) 無ければ file_id から署名URLを解決
  useEffect(() => {
    let mounted = true;
    (async () => {
      const url = await resolveAvatarURL(msg.user_avatar_file_id); // ← ここだけ
      if (mounted) setAvatarURL(url);
    })();
    return () => { mounted = false; };
  }, [msg.user_avatar_file_id, resolveAvatarURL]);


  const display = useMemo(
    () => msg.user_display_name || `User ${(msg.user_id ?? "").slice(0, 6)}`,
    [msg.user_display_name, msg.user_id]
  );

  return (
    <div className="msg">
      <Avatar
        src={avatarURL}
        name={msg.user_display_name || null}
        size={32}
        mode="contain"  // ← 切り取らず縮小
        shape="circle"
      />
      <div className="bubble">
        <div className="meta">
          <span>{display}</span>
          <span className="dot" />
          <span title={msg.created_at}>{formatTs(msg.created_at)}</span>
        </div>
        <div>{msg.text}</div>
      </div>
    </div>
  );
}

function formatTs(s: string) {
  const d = new Date(s);
  const today = new Date();
  const sameDay =
    d.getFullYear() === today.getFullYear() &&
    d.getMonth() === today.getMonth() &&
    d.getDate() === today.getDate();

  const time = d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  return sameDay ? time : `${d.toLocaleDateString()} ${time}`;
}
