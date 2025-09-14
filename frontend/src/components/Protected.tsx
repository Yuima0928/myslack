// components/Protected.tsx
import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '../auth/useAuth';

export default function Protected() {
  const { token, ready } = useAuth();
  if (!ready) return <div style={{padding:16}}>Loading…</div>;
  if (!token) return <Navigate to="/login" replace />;
  return <Outlet />;
}

