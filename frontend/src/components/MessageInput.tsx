import { useState } from "react";
import { api } from "../api/client";

export default function MessageInput({ channelId }: { channelId: string | null }) {
  const [text, setText] = useState("");
  const [err, setErr] = useState<string | null>(null);

  async function send() {
    if (!channelId) return;
    try{
      await api.postMessage(channelId, text);
      setText("");
      setErr(null);
      // 別でWSを聴いていれば自動でリストに出る。今は手動で読み直してもOK
    }catch(e:any){
      setErr(e?.message ?? "Failed to send");
    }
  }

  return (
    <div className="composer">
      <div className="composer-box">
        <textarea
          className="textarea"
          placeholder="Type message…"
          value={text}
          onChange={e=>setText(e.target.value)}
          onKeyDown={(e)=>{ if(e.key==="Enter" && !e.shiftKey){ e.preventDefault(); send(); }}}
        />
        <button className="send" disabled={!text.trim() || !channelId} onClick={send}>Send</button>
      </div>
      <div className="helper">
        {err ? <span className="error">{err}</span> : <span>Enter to send • Shift+Enter for newline</span>}
      </div>
    </div>
  );
}
