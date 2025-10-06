// src/auth/AuthContext.tsx
import React, {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type {
  Auth0Client,
  RedirectLoginOptions,
  User,
} from "@auth0/auth0-spa-js";
import { auth0Promise } from "./auth0";
import { setTokenProvider } from "../api/client";

type AuthContextValue = {
  isLoading: boolean;
  isAuthenticated: boolean;
  user?: User | null;
  loginWithRedirect: (opts?: RedirectLoginOptions) => Promise<void>;
  getAccessToken: () => Promise<string>;
};

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export const useAuth = (): AuthContextValue => {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within <AuthProvider>");
  return ctx;
};

export const AuthProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
  const clientRef = useRef<Auth0Client | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const client = await auth0Promise;
        clientRef.current = client;

        const authed = await client.isAuthenticated();
        if (!mounted) return;
        setIsAuthenticated(authed);

        if (!authed) {
          setUser(null);
          return;
        }

        const [u, claims] = await Promise.all([
          client.getUser(),
          client.getIdTokenClaims(),
        ]);
        if (!mounted) return;
        setUser(u ?? null);

        setTokenProvider(() => client.getTokenSilently());

        const sub = (claims?.sub as string | undefined) || (u as any)?.sub || null;
        if (sub) {
          const key = "bootstrapped_sub";
          const last = localStorage.getItem(key);
          if (last !== sub) {
            try {
              const at = await client.getTokenSilently();
              const displayName =
                u?.name ?? (u as any)?.nickname ?? (u?.email?.split("@")[0] ?? null);
              const API_BASE = import.meta.env.VITE_API_BASE;
              await fetch(`${API_BASE}/auth/bootstrap`, {
                method: "POST",
                headers: {
                  "Content-Type": "application/json",
                  Authorization: `Bearer ${at}`,
                },
                body: JSON.stringify({
                  email: u?.email ?? null,
                  display_name: displayName,
                }),
              }).catch(() => {});
              localStorage.setItem(key, sub);
            } catch {}
          }
        }
      } finally {
        if (mounted) setIsLoading(false);
      }
    })();
    return () => {
      mounted = false;
    };
  }, []);

  const value = useMemo<AuthContextValue>(() => {
    return {
      isLoading,
      isAuthenticated,
      user,
      loginWithRedirect: async (opts) => {
        const c = clientRef.current ?? (await auth0Promise);
        await c.loginWithRedirect({
          appState: { returnTo: location.pathname },
          ...opts,
        });
      },
      getAccessToken: async () => {
        const c = clientRef.current ?? (await auth0Promise);
        return c.getTokenSilently();
      },
    };
  }, [isLoading, isAuthenticated, user]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
