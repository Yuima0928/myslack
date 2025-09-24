// src/pages/Callback.tsx
import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { auth0Promise } from '../auth/auth0';

export default function Callback() {
  const nav = useNavigate();

  useEffect(() => {
    (async () => {
      const params = new URLSearchParams(window.location.search);
      const hasCode = params.has('code');
      const hasState = params.has('state');

      // 直アクセス（/callback に直接来た）→ ルートへ戻す（Protected が再度リダイレクト処理）
      if (!hasCode || !hasState) {
        nav('/', { replace: true });
        return;
      }

      const auth0 = await auth0Promise;
      try {
        const { appState } = await auth0.handleRedirectCallback();
        const returnTo =
          (appState as { returnTo?: string } | undefined)?.returnTo ?? '/chat';
        nav(returnTo, { replace: true });
      } catch (e) {
        // state 不一致など失敗→ いったんルートへ（Protected が処理）
        console.error('handleRedirectCallback failed:', e);
        nav('/', { replace: true });
      }
    })();
  }, [nav]);

  return <div>Signing you in…</div>;
}
