// App.tsx
import { Routes, Route, Navigate } from 'react-router-dom';
import { Protected } from './components/Protected';
import Chat from './pages/Chat';
import Callback from './pages/Callback'; // ← 追加

export default function App() {
  return (
    <Routes>
      {/* 認証不要ルート */}
      <Route path="/callback" element={<Callback />} /> {/* ← これが必須 */}
      {/* 認証が必要な領域 */}
      <Route element={<Protected />}>
        <Route path="/" element={<Navigate to="/chat" replace />} />
        <Route path="/chat" element={<Chat />} />
      </Route>
      {/* 任意: 404 対応 */}
      <Route path="*" element={<Navigate to="/chat" replace />} />
    </Routes>
  );
}
