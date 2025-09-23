// src/components/MessageInput.tsx
import { useRef, useState } from "react";
import { api } from "../api/client";

type Props = {
  channelId: string | null;
  workspaceId: string | null;    // ★ 追加
};

export default function MessageInput({ channelId, workspaceId }: Props) {
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const fileRef = useRef<HTMLInputElement | null>(null);

  async function send() {
    if (!channelId || !text.trim()) return;
    setBusy(true);
    try {
      await api.postMessage(channelId, text); // ←既存の投稿API
      setText("");
    } finally {
      setBusy(false);
    }
  }

  async function onPickFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = ""; // 同じファイルを選んだときも change させる
    if (!file || !channelId || !workspaceId) return;

    setBusy(true);
    try {
      // 1) 署名URLを貰う
      const { upload_url, storage_key, file_id } = await api.signUpload(
        workspaceId, channelId, {
          filename: file.name,
          content_type: file.type || "application/octet-stream",
          size_bytes: file.size,
        }
      );

      // 2) 直PUT
      const putRes = await fetch(upload_url, {
        method: "PUT",
        headers: { "Content-Type": file.type || "application/octet-stream" },
        body: file,
      });
      if (!putRes.ok) throw new Error(`PUT failed: ${putRes.status}`);
      // S3/MinIO は ETag ヘッダを返す（例: "5d41402abc4b2a76b9719d911017c592"）
      const rawETag = putRes.headers.get("ETag") || putRes.headers.get("Etag") || "";
      const etag = rawETag.replaceAll(`"`, "");

      // 3) 完了報告（DB登録）
      const rec = await api.completeFile({
        storage_key: storage_key,
        etag,
        sha256_hex: null, // 計算しない場合は null
        filename: file.name,
        content_type: file.type || "application/octet-stream",
        size_bytes: file.size,
        workspace_id: workspaceId,
        channel_id: channelId,
      });

      // 4) ダウンロードURLを取得（リンクとしてメッセージに貼る）
      const { url } = await api.getFileURL(rec.id, "attachment");

      // 5) ファイルのリンクを本文にしてメッセージ投稿
      const pretty = file.type.startsWith("image/")
        ? `(image) ${file.name}`
        : `(file) ${file.name}`;
      await api.postMessage(channelId, `${pretty}\n${url}`);
    } catch (err) {
      console.error(err);
      alert("Upload failed");
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

      {/* ★ 添付 */}
      <input
        ref={fileRef}
        type="file"
        style={{ display: "none" }}
        onChange={onPickFile}
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
