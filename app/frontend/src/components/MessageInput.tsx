// src/components/MessageInput.tsx
import { useRef, useState } from 'react';
import { api } from '../api/client';

type Props = {
  channelId: string | null;
  workspaceId: string | null;
};

export default function MessageInput({ channelId, workspaceId }: Props) {
  const [text, setText] = useState('');
  const [busy, setBusy] = useState(false);
  const fileRef = useRef<HTMLInputElement | null>(null);

  async function send() {
    if (!channelId || !text.trim()) return;
    setBusy(true);
    try {
      await api.postMessage(channelId, text.trim());
      setText('');
    } finally {
      setBusy(false);
    }
  }

  async function onPickFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = ''; // 同じファイルを再選択しても change させる
    if (!file || !channelId || !workspaceId) return;

    setBusy(true);
    try {
      // 1) 署名URLを取得
      const { upload_url, storage_key /*, file_id*/ } = await api.signUploadMessage(
        workspaceId,
        channelId,
        {
          filename: file.name,
          content_type: file.type || 'application/octet-stream',
          size_bytes: file.size,
        },
      );

      // 2) PUT アップロード
      const putRes = await fetch(upload_url, {
        method: 'PUT',
        headers: { 'Content-Type': file.type || 'application/octet-stream' },
        body: file,
      });
      if (!putRes.ok) throw new Error(`PUT failed: ${putRes.status}`);

      // ETag ヘッダ（MinIO/S3 は引用符付きのことが多い）
      const rawETag =
        putRes.headers.get('ETag') ??
        putRes.headers.get('Etag') ??
        putRes.headers.get('etag') ??
        '';
      const etag = rawETag.replaceAll(`"`, '');

      // 3) 完了報告（DB登録）
      const rec = await api.completeFile({
        storage_key,
        etag,
        sha256_hex: null, // 計算しない場合は null
        filename: file.name,
        content_type: file.type || 'application/octet-stream',
        size_bytes: file.size,
        workspace_id: workspaceId,
        channel_id: channelId,
        purpose: 'message_attachment', // サーバ側が見ていれば指定
      });

      // 4) ダウンロードURL取得（画像は inline、その他は attachment）
      const disp = (file.type || '').startsWith('image/') ? 'inline' : 'attachment';
      const { url } = await api.getFileURL(rec.id, disp as 'inline' | 'attachment');

      // 5) メッセージ投稿（画像は見やすいようにラベル簡素化）
      const pretty = file.type.startsWith('image/') ? `${file.name}` : `(file) ${file.name}`;
      await api.postMessage(channelId, `${pretty}\n${url}`);
    } catch (err) {
      console.error(err);
      alert('Upload failed');
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="message-input">
      <input
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder="Write a message"
        disabled={!channelId || busy}
      />
      <button className="btn" onClick={send} disabled={!channelId || busy || !text.trim()}>
        Send
      </button>

      {/* 添付 */}
      <input
        ref={fileRef}
        type="file"
        style={{ display: 'none' }}
        onChange={onPickFile}
        // accept="image/*" // 画像限定にしたい場合は付ける
      />
      <button
        className="btn"
        onClick={() => fileRef.current?.click()}
        disabled={!channelId || !workspaceId || busy}
        style={{ marginLeft: 8 }}
      >
        Attach
      </button>
    </div>
  );
}
