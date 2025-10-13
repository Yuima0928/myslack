// src/pages/Chat.tsx
import { useEffect, useMemo, useState } from 'react';
import { api } from '../api/client';
import { useAuth } from '../auth/AuthContext';
import ChannelList from '../components/ChannelList';
import MessageList from '../components/MessageList';
import MessageInput from '../components/MessageInput';
import ChannelCreateForm from '../components/ChannelCreateForm';
import ProfileSetupModal from '../components/ProfileSetupModal';

type Workspace = { id: string; name: string };
type Channel = { id: string; name: string; is_private?: boolean };

export default function Chat() {
  const { isLoading, isAuthenticated } = useAuth(); // ← ここだけでOK

  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [activeWs, setActiveWs] = useState<string | null>(null);
  const [activeCh, setActiveCh] = useState<string | null>(null);
  const [isMember, setIsMember] = useState<boolean>(false);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [needsProfile, setNeedsProfile] = useState(false);

  // Protected が未認証を /login に飛ばすので、ここではローディングだけ見る
  if (isLoading) return <div style={{ padding: 20 }}>Loading…</div>;
  // 念のための保険（Protectedのはずで来ない）
  if (!isAuthenticated) return null;

  // 認証済みになったら WS を読む
  useEffect(() => {
    if (!isAuthenticated) return;
    (async () => {
      const res = await api.listWorkspaces();
      setWorkspaces(res);
      if (res.length > 0) setActiveWs(res[0].id);
    })();
  }, [isAuthenticated]);

  // WS 選択時に CH を読む
  useEffect(() => {
    if (!isAuthenticated) return;
    (async () => {
      if (!activeWs) {
        setChannels([]);
        setActiveCh(null);
        return;
      }
      const chs = await api.listChannels(activeWs);
      setChannels(chs);
      if (chs.length > 0) setActiveCh(chs[0].id);
    })();
  }, [isAuthenticated, activeWs]);

  useEffect(() => {
    (async () => {
      if (!isAuthenticated || !activeCh) {
        setIsMember(false);
        return;
      }
      try {
        const m = await api.isChannelMember(activeCh);
        setIsMember(m.is_member);
      } catch {
        setIsMember(false);
      }
    })();
  }, [isAuthenticated, activeWs, activeCh]);

  useEffect(() => {
    if (!isAuthenticated) return;
    (async () => {
      try {
        const me = await api.getMe(); // { display_name?, avatar_url?, ... }
        if (!me.display_name || me.display_name.trim() === '') {
          setNeedsProfile(true);
        }
      } catch {
        // 無視（失敗しても致命ではない）
      }
    })();
  }, [isAuthenticated]);

  const currentCh = useMemo(
    () => channels.find((c) => c.id === activeCh) || null,
    [channels, activeCh],
  );

  // 以降はそのまま UI
  async function handleAddWorkspaceMember() {
    if (!activeWs) {
      alert('Select a workspace first');
      return;
    }
    const q = prompt('Search user (email or display name contains):');
    if (!q) return;
    const cand = await api.searchUsers(q, 10);
    if (cand.length === 0) {
      alert('No users found');
      return;
    }
    const list = cand
      .map((u: any, i: number) => `${i}: ${u.email}${u.display_name ? ` (${u.display_name})` : ''}`)
      .join('\n');
    const ans = prompt(`Pick index to add:\n${list}\n\nEnter number (0-${cand.length - 1})`);
    if (ans == null) return;
    const idx = Number(ans);
    if (Number.isNaN(idx) || idx < 0 || idx >= cand.length) {
      alert('Invalid index');
      return;
    }
    await api.addWorkspaceMember(activeWs, cand[idx].id, 'member');
    alert(`Added ${cand[idx].email} to workspace`);
  }

  async function handleAddChannelMember() {
    if (!activeCh) {
      alert('Select a channel first');
      return;
    }
    const q = prompt('Search workspace member (email or display name contains):');
    if (!q) return;
    const cand = await api.searchChannelMemberCandidates(activeCh, q, 10);
    if (cand.length === 0) {
      alert('No candidates (maybe already joined or not in workspace)');
      return;
    }
    const list = cand
      .map((u: any, i: number) => `${i}: ${u.email}${u.display_name ? ` (${u.display_name})` : ''}`)
      .join('\n');
    const ans = prompt(`Pick index to add:\n${list}\n\nEnter number (0-${cand.length - 1})`);
    if (ans == null) return;
    const idx = Number(ans);
    if (Number.isNaN(idx) || idx < 0 || idx >= cand.length) {
      alert('Invalid index');
      return;
    }
    await api.addChannelMember(activeCh, cand[idx].id, 'member');
    alert(`Added ${cand[idx].email} to channel`);
  }

  async function handleJoinSelf() {
    if (!activeWs || !activeCh) return;
    try {
      await api.joinSelf(activeWs, activeCh); // POST /join
      setIsMember(true); // 参加できたので入力欄を解放
      // 必要ならチャンネル一覧を更新
      const chs = await api.listChannels(activeWs);
      setChannels(chs);
    } catch (e: any) {
      alert(`参加に失敗しました: ${e?.message ?? e}`);
    }
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <button
          className="btn"
          onClick={async () => {
            const name = prompt('Workspace name');
            if (!name) return;
            const ws = await api.createWorkspace(name);
            setWorkspaces([...(workspaces || []), { id: ws.id, name }]);
            setActiveWs(ws.id);
          }}
        >
          + Create Workspace
        </button>

        <button className="btn" onClick={handleAddWorkspaceMember}>
          + Add WS Member
        </button>

        <div style={{ color: '#9ca3af', fontWeight: 700, marginTop: 4 }}>Workspaces</div>
        <div className="list">
          {workspaces.map((w, i) => {
            const shortId = (w?.id ?? '').slice(0, 6);
            return (
              <div
                key={w.id ?? `ws-${i}`}
                className={`list-item ${activeWs === w.id ? 'active' : ''}`}
                onClick={() => w.id && setActiveWs(w.id)}
                title={w.id ?? ''}
              >
                {w.name ?? '(no name)'}{' '}
                <span style={{ color: '#64748b' }}>（{shortId || '------'}）</span>
              </div>
            );
          })}
        </div>

        <div style={{ display: 'grid', gap: 8 }}>
          <button className="btn" onClick={() => setShowCreateForm((v) => !v)}>
            {showCreateForm ? 'Close' : '+ Create Channel'}
          </button>

          {showCreateForm && (
            <ChannelCreateForm
              onCancel={() => setShowCreateForm(false)}
              onCreate={async (name, isPrivate) => {
                if (!activeWs) {
                  alert('Select a workspace first');
                  return;
                }
                const res = await api.createChannel(activeWs, name, isPrivate);
                const newCh = { id: res.id, name, is_private: isPrivate };
                setChannels([newCh, ...channels]);
                setActiveCh(res.id);
                setShowCreateForm(false);
              }}
            />
          )}
        </div>

        <button className="btn" onClick={handleAddChannelMember}>
          + Add Channel Member
        </button>

        <div style={{ color: '#9ca3af', fontWeight: 700, marginTop: 4 }}>Channels</div>
        <ChannelList channels={channels} activeId={activeCh} onPick={setActiveCh} />
      </aside>

      <section className="content">
        {/* ヘッダー例 */}
        <div className="header">
          <div className="channel-pill">Channel: {currentCh ? currentCh.name : '—'}</div>
          <div style={{ flex: 1 }} />
          <button className="btn" onClick={() => setNeedsProfile(true)}>
            プロフィール編集
          </button>
        </div>
        {needsProfile && <ProfileSetupModal onDone={() => setNeedsProfile(false)} />}

        <MessageList channelId={activeCh} />
        {activeWs && activeCh ? (
          isMember ? (
            <MessageInput channelId={activeCh} workspaceId={activeWs} />
          ) : (
            <div style={{ padding: 12, borderTop: '1px solid #eee' }}>
              <p style={{ marginBottom: 8 }}>
                このチャンネルに参加するとメッセージを投稿できます。
              </p>
              <button className="btn" onClick={handleJoinSelf}>
                チャンネルに参加する
              </button>
            </div>
          )
        ) : null}
      </section>
    </div>
  );
}
