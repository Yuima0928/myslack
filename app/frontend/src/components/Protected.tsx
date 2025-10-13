// src/components/Protected.tsx
import { type PropsWithChildren, useEffect, useRef } from 'react';
import { useAuth } from '../auth/AuthContext';
import { Outlet } from 'react-router-dom';

export function Protected({ children }: PropsWithChildren) {
  const { isLoading, isAuthenticated, loginWithRedirect } = useAuth();
  const kickedRef = useRef(false);

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated && !kickedRef.current) {
      kickedRef.current = true; // ← 二重キック防止（React 18 StrictMode対応）
      loginWithRedirect({ appState: { returnTo: window.location.pathname } }).catch(() => {
        kickedRef.current = false;
      });
    }
  }, [isLoading, isAuthenticated, loginWithRedirect]);

  // ローディング中/未認証中は何も描画しない（Auth0画面へ遷移中）
  if (isLoading || !isAuthenticated) return null;

  // 子要素 or ネストされたルートを表示
  return children ? <>{children}</> : <Outlet />;
}
