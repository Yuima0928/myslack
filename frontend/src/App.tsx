// App.tsx（例）
import { Routes, Route, Navigate } from 'react-router-dom';
import Protected from './components/Protected';
import Login from './pages/Login';
import Chat from './pages/Chat';

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route element={<Protected />}>
        <Route path="/" element={<Navigate to="/chat" replace />} />
        <Route path="/chat" element={<Chat />} />
      </Route>
    </Routes>
  );
}
