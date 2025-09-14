import React, { useState } from 'react';
import { useAuth } from '../auth/useAuth';
import { useNavigate } from 'react-router-dom';

export default function Login() {
  const nav = useNavigate();
  const { login, signup } = useAuth();
  const [mode, setMode] = useState<'login'|'signup'>('login');
  const [email, setEmail] = useState('u@example.com');
  const [password, setPassword] = useState('pass');
  const [display, setDisplay] = useState('U');
  const [err, setErr] = useState<string | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr(null);
    try {
      if (mode === 'login') await login(email, password);
      else await signup(email, password, display);

      // ここを '/' → '/chat' に
      nav('/chat', { replace: true });
    } catch (e:any) {
      setErr(e.message || 'failed');
    }
  };


  return (
    <div style={{maxWidth: 360, margin: '10vh auto', padding: 16}}>
      <h2>{mode==='login' ? 'Login' : 'Sign up'}</h2>
      <form onSubmit={submit} style={{display:'grid', gap:8}}>
        <input value={email} onChange={e=>setEmail(e.target.value)} placeholder="email" />
        <input type="password" value={password} onChange={e=>setPassword(e.target.value)} placeholder="password" />
        {mode==='signup' && (
          <input value={display} onChange={e=>setDisplay(e.target.value)} placeholder="display name (optional)" />
        )}
        {err && <div style={{color:'crimson'}}>{err}</div>}
        <button type="submit">{mode==='login' ? 'Login' : 'Create account'}</button>
      </form>
      <div style={{marginTop:8}}>
        <button onClick={()=>setMode(m=>m==='login'?'signup':'login')}>
          {mode==='login' ? 'Need account? Sign up' : 'Have account? Login'}
        </button>
      </div>
    </div>
  );
}
