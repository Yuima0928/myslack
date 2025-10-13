import { useEffect, useRef, useState } from 'react';
import { api } from '../api/client';

type Props = {
  onDone: () => void;
};

export default function ProfileSetupModal({ onDone }: Props) {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [displayName, setDisplayName] = useState<string>('');
  const [avatarUrl, setAvatarUrl] = useState<string | null>(null);
  const pendingAvatarFileId = useRef<string | null>(null); // complete後のfile_id

  useEffect(() => {
    (async () => {
      try {
        const me = await api.getMe(); // { display_name?, avatar_url?, avatar_file_id? }
        setDisplayName(me.display_name ?? '');
        setAvatarUrl(me.avatar_url ?? null);
        pendingAvatarFileId.current = me.avatar_file_id ?? null;
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  async function pickAvatar(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0];
    e.target.value = ''; // 同じファイル再選択でもchangeさせる
    if (!f) return;

    try {
      setSaving(true);
      // 1) 署名URL
      const { upload_url, storage_key } = await api.signUploadAvatar({
        filename: f.name,
        content_type: f.type || 'application/octet-stream',
        size_bytes: f.size,
      });

      // 2) PUT
      const putRes = await fetch(upload_url, {
        method: 'PUT',
        headers: { 'Content-Type': f.type || 'application/octet-stream' },
        body: f,
      });
      if (!putRes.ok) throw new Error(`PUT failed: ${putRes.status}`);
      const rawETag =
        putRes.headers.get('ETag') ??
        putRes.headers.get('Etag') ??
        putRes.headers.get('etag') ??
        '';
      const etag = rawETag.replaceAll(`"`, '');

      // 3) 完了報告（purposeに"avatar"を渡す設計なら追加）
      const rec = await api.completeFile({
        storage_key,
        etag,
        sha256_hex: null,
        filename: f.name,
        content_type: f.type || 'application/octet-stream',
        size_bytes: f.size,
        workspace_id: null, // アバターはWS/CHに属さない想定
        channel_id: null,
        purpose: 'avatar',
      });

      // 最新の署名GET URLをinlineで取得してプレビュー
      const { url } = await api.getFileURL(rec.id, 'inline');
      setAvatarUrl(url);
      pendingAvatarFileId.current = rec.id;
    } catch (e) {
      console.error(e);
      alert('Avatar upload failed');
    } finally {
      setSaving(false);
    }
  }

  async function save() {
    setSaving(true);
    try {
      await api.updateMe({
        display_name: displayName,
        avatar_file_id_or_null: pendingAvatarFileId.current ?? '', // 空文字で解除、今回は設定するのでid
      });
      onDone();
    } catch (e) {
      console.error(e);
      alert('Save failed');
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="modal">
        <div className="card">Loading…</div>
      </div>
    );
  }

  return (
    <div className="modal">
      <div className="card" style={{ minWidth: 360 }}>
        <h3 style={{ marginTop: 0 }}>プロフィールを設定</h3>

        <div style={{ display: 'grid', gap: 12 }}>
          <div>
            <label style={{ display: 'block', fontSize: 12, color: '#6b7280' }}>表示名</label>
            <input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="Your display name"
              disabled={saving}
              style={{ width: '100%' }}
            />
          </div>

          <div>
            <label style={{ display: 'block', fontSize: 12, color: '#6b7280' }}>アバター</label>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <div
                style={{
                  width: 48,
                  height: 48,
                  borderRadius: '9999px',
                  background: '#e5e7eb',
                  overflow: 'hidden',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontWeight: 700,
                }}
              >
                {avatarUrl ? (
                  // eslint-disable-next-line jsx-a11y/alt-text
                  <img
                    src={avatarUrl}
                    style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                  />
                ) : (
                  (displayName || 'U').slice(0, 1).toUpperCase()
                )}
              </div>
              <label className="btn">
                画像を選択
                <input
                  type="file"
                  accept="image/*"
                  onChange={pickAvatar}
                  style={{ display: 'none' }}
                  disabled={saving}
                />
              </label>
              <button
                className="btn"
                onClick={() => {
                  // 解除：プレビュー消し、file_id 空文字を送れるようにする
                  setAvatarUrl(null);
                  pendingAvatarFileId.current = '';
                }}
                disabled={saving}
              >
                画像を外す
              </button>
            </div>
          </div>

          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
            <button className="btn" onClick={onDone} disabled={saving}>
              キャンセル
            </button>
            <button className="btn" onClick={save} disabled={saving}>
              保存
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
