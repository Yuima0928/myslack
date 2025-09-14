// src/hooks/useWS.ts
import { useEffect, useRef } from "react";
import { wsURL } from "../api/client";

/**
 * 指定channelにWS接続し、受信したメッセージを onEvent で通知する。
 * サーバは {type:"message_created", message:{...}} をブロードキャストする前提。
 */
export function useWS(
  channelId: string | null,
  onEvent: (ev: unknown) => void
) {
  const wsRef = useRef<WebSocket | null>(null);
  const retryRef = useRef(0);
  const timerRef = useRef<number | null>(null);

  useEffect(() => {
    // チャンネル未選択なら閉じておく
    if (!channelId) {
      if (wsRef.current && wsRef.current.readyState <= 1) wsRef.current.close();
      wsRef.current = null;
      return;
    }

    let cancelled = false;

    const connect = () => {
      if (cancelled) return;
      const url = wsURL(channelId);
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        retryRef.current = 0;
        // 必要なら「入室」メッセージ等を送る場合はここ
      };

      ws.onmessage = (e) => {
        try {
          const data = JSON.parse(e.data);
          onEvent(data);
        } catch {
          // プレーンテキストなどの場合
          onEvent(e.data);
        }
      };

      ws.onclose = () => {
        // 自動再接続（指数バックオフ 0.5s,1s,2s,4s… 最大10s）
        if (!cancelled) {
          const delay = Math.min(10000, 500 * 2 ** retryRef.current);
          retryRef.current += 1;
          timerRef.current = window.setTimeout(connect, delay) as unknown as number;
        }
      };

      ws.onerror = () => {
        // onclose で再接続するので特に何もしない
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
      if (wsRef.current && wsRef.current.readyState <= 1) {
        wsRef.current.close();
      }
      wsRef.current = null;
    };
  }, [channelId, onEvent]);
}
