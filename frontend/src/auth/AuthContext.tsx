import React, { createContext, useEffect, useMemo, useState } from 'react';
import { api } from '../api/client';
import type { Me } from '../api/types';

type AuthState = {
  token: string | null;
  me: Me | null;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string, display?: string) => Promise<void>;
  logout: () => void;
  ready: boolean;
};

export const AuthContext = createContext<AuthState>({
  token: null, me: null, ready: false,
  async login(){}, async signup(){}, logout(){},
});

export const AuthProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('token'));
  const [me, setMe] = useState<Me | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        if (token) {
          const profile = await api.me();
          setMe(profile);
        }
      } catch {
        localStorage.removeItem('token');
        setToken(null);
        setMe(null);
      } finally {
        setReady(true);
      }
    })();
  }, [token]);

  const value = useMemo(() => ({
    token, me, ready,
    login: async (email: string, password: string) => {
      const t = await api.login(email, password);
      localStorage.setItem('token', t.access_token);
      setToken(t.access_token);
      const profile = await api.me();
      setMe(profile);
    },
    signup: async (email: string, password: string, display?: string) => {
      const t = await api.signup(email, password, display);
      localStorage.setItem('token', t.access_token);
      setToken(t.access_token);
      const profile = await api.me();
      setMe(profile);
    },
    logout: () => {
      localStorage.removeItem('token');
      setToken(null);
      setMe(null);
    },
  }), [token, me, ready]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
