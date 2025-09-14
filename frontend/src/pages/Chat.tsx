import { useEffect, useMemo, useState } from "react";
import { useAuth } from "../auth/useAuth";
import { api } from "../api/client";
import ChannelList from "../components/ChannelList";
import MessageList from "../components/MessageList";
import MessageInput from "../components/MessageInput";

type Workspace = { id: string; name: string };
type Channel = { id: string; name: string; is_private?: boolean };

export default function Chat() {
  const { token } = useAuth();
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [activeWs, setActiveWs] = useState<string | null>(null);
  const [activeCh, setActiveCh] = useState<string | null>(null);

  // 初回：所属WSを取る
  useEffect(() => {
    (async () => {
      const res = await api.listWorkspaces();
      setWorkspaces(res);
      if (res.length > 0) setActiveWs(res[0].id);
    })();
  }, [token]);

  // WS選択時：そのWSで見えるチャンネルを取得
  useEffect(() => {
    (async () => {
      if (!activeWs) { setChannels([]); setActiveCh(null); return; }
      try {
        const chs = await api.listChannels(activeWs);
        setChannels(chs);
        if (chs.length > 0) setActiveCh(chs[0].id);
      } catch (e) {
        console.error(e);
        setChannels([]);
        setActiveCh(null);
      }
    })();
  }, [activeWs]);


  const currentWs = useMemo(() => workspaces.find(w => w.id === activeWs) || null, [workspaces, activeWs]);
  const currentCh = useMemo(() => channels.find(c => c.id === activeCh) || null, [channels, activeCh]);

  return (
    <div className="app">
      <aside className="sidebar">
        <button className="btn" onClick={async () => {
          const name = prompt("Workspace name");
          if (!name) return;
          const ws = await api.createWorkspace(name)
          setWorkspaces([...(workspaces||[]), { id: ws.id, name }]);
          setActiveWs(ws.id);
        }}>+ Create Workspace</button>

        <div style={{color:"#9ca3af", fontWeight:700, marginTop:4}}>Workspaces</div>
        <div className="list">
          {workspaces.map((w, i) => {
            const shortId = (w?.id ?? '').slice(0, 6);
            return (
              <div
                key={w.id ?? `ws-${i}`}  // ← ランダム禁止。最悪 index で固定化
                className={`list-item ${activeWs === w.id ? "active" : ""}`}
                onClick={() => w.id && setActiveWs(w.id)}
                title={w.id ?? ''}
              >
                {w.name ?? '(no name)'} <span style={{color:"#64748b"}}>（{shortId || '------'}）</span>
              </div>
            );
          })}
        </div>

        <button className="btn" onClick={async () => {
          if (!activeWs) return alert("Select a workspace first");
          const name = prompt("Channel name");
          if (!name) return;
          const res = await api.createChannel(activeWs, name)
          const newCh = { id: res.id, name };
          setChannels([newCh, ...channels]);
          setActiveCh(res.id);
        }}>+ Create Channel</button>

        <div style={{color:"#9ca3af", fontWeight:700, marginTop:4}}>Channels</div>
        <ChannelList
          channels={channels}
          activeId={activeCh}
          onPick={setActiveCh}
        />
      </aside>

      <section className="content">
        <div className="header">
          <div className="channel-pill">Channel: {currentCh ? currentCh.name : "—"}</div>
          {currentWs && (
            <div style={{color:"#9ca3af"}}>in <code>{currentWs.name}</code></div>
          )}
        </div>

        <MessageList channelId={activeCh} />
        <MessageInput channelId={activeCh} />
      </section>
    </div>
  );
}
