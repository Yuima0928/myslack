// src/hooks/useWS.ts
import { useEffect, useRef } from "react";
import { wsURL } from "../api/client";
import { useAuth } from "../auth/AuthContext";

export function useWS(channelId: string | null, onEvent: (ev: unknown) => void) {
  const wsRef = useRef<WebSocket | null>(null);
  const { getAccessToken } = useAuth();

  useEffect(() => {
    let closed = false;
    if (!channelId) return;

    (async () => {
      const token = await getAccessToken();
      // サブプロトコルに "bearer,<JWT>" を載せる
    const ws = new WebSocket(wsURL(channelId), ["bearer", token]);
      wsRef.current = ws;

      ws.onmessage = (e) => {
        try {
          onEvent(JSON.parse(e.data));
        } catch {
          // no-op
        }
      };
      ws.onerror = () => {};
      ws.onclose = () => {};

      if (closed) ws.close();
    })();

    return () => {
      closed = true;
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [channelId, onEvent, getAccessToken]);
}
