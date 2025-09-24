import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { useWS } from "../hooks/useWS";

type Msg = {
  id: string;
  workspace_id: string;
  channel_id: string;
  user_id: string;
  text: string;
};

export default function MessageList({ channelId }: { channelId: string | null }) {
  const { isAuthenticated, isLoading } = useAuth();
  const [msgs, setMsgs] = useState<Msg[]>([]);
  const scRef = useRef<HTMLDivElement>(null);

  // 初回ロード
  useEffect(() => {
    if (!channelId) { setMsgs([]); return; }
    if (isLoading || !isAuthenticated) return;

    (async () => {
      const res = await api.listMessages(channelId); // ← 中で自動的にAT付与
      setMsgs(res);
      requestAnimationFrame(() => scRef.current?.scrollTo({ top: 1e9 }));
    })();
  }, [channelId, isAuthenticated, isLoading]); // ← token 依存は不要

  // WSイベント処理（新着が来たら末尾に追加）
  const onWsEvent = useCallback((ev: unknown) => {
    if (typeof ev === "object" && ev !== null && "type" in (ev as any)) {
      const e = ev as { type: string; message?: Msg };
      if (e.type === "message_created" && e.message) {
        if (!channelId || e.message.channel_id !== channelId) return;
        setMsgs((prev) => {
          if (prev.some(m => m.id === e.message!.id)) return prev;
          const next = [...prev, e.message!];
          requestAnimationFrame(() => scRef.current?.scrollTo({ top: 1e9 }));
          return next;
        });
      }
    }
  }, [channelId]);

  useWS(channelId, onWsEvent);

  return (
    <div className="messages" ref={scRef}>
      {msgs.length === 0 && <div style={{color:"#9ca3af"}}>No messages</div>}
      {msgs.map(m => (
        <div key={m.id} className="msg">
          <div className="avatar">{(m.user_id || "u").slice(0, 1).toUpperCase()}</div>
          <div className="bubble">
            <div className="meta">
              <span>User {(m.user_id ?? "unknown").slice(0, 6)}</span>
              <span className="dot" />
              <span>{(m.id ?? "").slice(0, 8)}</span>
            </div>
            <div>{m.text}</div>
          </div>
        </div>
      ))}
    </div>
  );
}
