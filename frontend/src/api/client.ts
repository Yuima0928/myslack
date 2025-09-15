// frontend/src/api/client.ts
const API_BASE = import.meta.env.VITE_API_BASE as string;

export type Workspace = { id: string; name: string };
export type Channel   = { id: string; name: string; is_private?: boolean };

export class ApiClient {
  private get token(): string | null {
    return localStorage.getItem("token");
  }

  private headers(json = true): HeadersInit {
    const h: Record<string, string> = {};
    if (json) h["Content-Type"] = "application/json";
    if (this.token) h["Authorization"] = `Bearer ${this.token}`;
    return h;
  }

  private async json<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetch(`${API_BASE}${path}`, init);
    const isJSON = res.headers.get("content-type")?.includes("application/json");
    const body = isJSON ? await res.json() : await res.text();
    if (!res.ok) {
      const msg = isJSON ? (body?.error?.message || JSON.stringify(body)) : String(body);
      throw new Error(msg || `HTTP ${res.status}`);
    }
    return body as T;
  }

  // ---- Auth ----
  signup(email: string, password: string, displayName?: string) {
    return this.json<{ access_token: string; token_type: string }>("/auth/signup", {
      method: "POST",
      headers: this.headers(),
      body: JSON.stringify({ email, password, display_name: displayName ?? "" }),
    });
  }
  login(email: string, password: string) {
    return this.json<{ access_token: string; token_type: string }>("/auth/login", {
      method: "POST",
      headers: this.headers(),
      body: JSON.stringify({ email, password }),
    });
  }
  me() {
    return this.json<{ id: string; email: string; display_name?: string }>("/auth/me", {
      headers: this.headers(false),
    });
  }

  // ---- Workspaces / Channels ----
  listWorkspaces() {
    // GET /workspaces （自分が属するWS一覧）
    return this.json<Workspace[]>("/workspaces", { headers: this.headers(false) });
  }

  createWorkspace(name: string) {
    return this.json<{ id: string }>("/workspaces", {
      method: "POST",
      headers: this.headers(),
      body: JSON.stringify({ name }),
    });
  }

  listChannels(wsId: string) {
    return this.json<Array<{id:string; name:string; is_private:boolean}>>(
      `/workspaces/${wsId}/channels`,
      { headers: this.headers(false) }
    );
  }


  createChannel(wsId: string, name: string, isPrivate = false) {
    return this.json<{ id: string }>(`/workspaces/${wsId}/channels`, {
      method: "POST",
      headers: this.headers(),
      body: JSON.stringify({ name, is_private: isPrivate }),
    });
  }

  // ---- Messages ----
  listMessages(channelId: string) {
    return this.json<
      Array<{ id: string; workspace_id: string; channel_id: string; user_id: string; text: string }>
    >(`/channels/${channelId}/messages`, { headers: this.headers(false) });
  }

  postMessage(channelId: string, text: string, parentId?: string) {
    return this.json<{ id: string; workspace_id: string; channel_id: string; user_id: string; text: string }>(
      `/channels/${channelId}/messages`,
      { method: "POST", headers: this.headers(), body: JSON.stringify({ text, parent_id: parentId ?? null }) }
    );
  }
  
  // ユーザー検索
  searchUsers(q: string, limit = 20) {
    const p = new URLSearchParams({ q, limit: String(limit) });
    return this.json<Array<{id:string;email:string;display_name?:string}>>(
      `/users/search?${p}`,
      { headers: this.headers(false) }   // ← Authorization を付ける（Content-Typeは不要）
    );
  }

  // WSにメンバー追加
  addWorkspaceMember(wsId: string, userId: string, role: 'owner'|'member' = 'member') {
    return this.json<{ok:boolean}>(`/workspaces/${wsId}/members`, {
      method: 'POST',
      headers: this.headers(),
      body: JSON.stringify({ user_id: userId, role }),
    });
  }

  // チャンネル候補検索（WSメンバーから、未参加のみ）
  searchChannelMemberCandidates(channelId: string, q: string, limit = 20) {
    const p = new URLSearchParams({ q, limit: String(limit) });
    return this.json<Array<{id:string;email:string;display_name?:string}>>(
      `/channels/${channelId}/members/search?${p}`,
      { headers: this.headers(false) }   // ← ここも同様に付与
    );
  }

  // チャンネルにメンバー追加
  addChannelMember(channelId: string, userId: string, role: 'owner'|'member' = 'member') {
    return this.json<{ok:boolean}>(`/channels/${channelId}/members`, {
      method: 'POST',
      headers: this.headers(),
      body: JSON.stringify({ user_id: userId, role }),
    });
  }
}

export const api = new ApiClient();

// WebSocket URL helper
export const wsURL = (channelId: string) => {
  const base = import.meta.env.VITE_WS_BASE as string;
  const url = new URL("/ws", base);
  url.searchParams.set("channel_id", channelId);
  return url.toString();
};
