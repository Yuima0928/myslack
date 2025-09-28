// src/api/client.ts
// src/api/client.ts
const API_BASE = import.meta.env.VITE_API_BASE as string;

export type Msg = {
  id: string;
  workspace_id: string;
  channel_id: string;
  user_id: string;
  text: string;
  // 追加フィールド（サーバの MsgOut に合わせる）
  user_display_name?: string | null;
  user_avatar_file_id?: string | null;
  parent_id?: string | null;
  thread_root_id?: string | null;
  created_at: string; 
};

export type FileRec = {
  id: string;
  workspace_id: string | null; // アバター用途では null の可能性も
  channel_id: string | null;
  uploader_id: string;
  filename: string;
  content_type?: string | null;
  size_bytes?: number | null;
  etag?: string | null;
  sha256_hex?: string | null;
  storage_key: string;
  is_image: boolean;
  created_at: string;
};

type Fetcher = (input: RequestInfo, init?: RequestInit) => Promise<Response>;

// ---- Token Provider wiring ----
let tokenProvider: null | (() => Promise<string>) = null;

export function setTokenProvider(p: () => Promise<string>) {
  tokenProvider = p;
}

// ---- Low-level fetch wrapper ----
export async function authedFetch(
  input: RequestInfo,
  init: RequestInit = {},
  fetcher: Fetcher = fetch
) {
  const headers = new Headers(init.headers || {});
  if (tokenProvider) {
    const token = await tokenProvider();
    headers.set("Authorization", `Bearer ${token}`);
  }
  return fetcher(input, { ...init, headers });
}

// ---- Helpers ----
function apiPath(path: string) {
  return `${API_BASE}${path}`;
}

async function parseJsonOrText(res: Response) {
  const isJSON = res.headers.get("content-type")?.includes("application/json");
  return isJSON ? res.json() : res.text();
}

async function authedJson<T>(path: string, init: RequestInit = {}) {
  const headers = new Headers(init.headers || {});
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  const res = await authedFetch(apiPath(path), { ...init, headers });
  const body = await parseJsonOrText(res);

  if (!res.ok) {
    const msg =
      typeof body === "object" && body
        ? ((body as any).detail || (body as any).error || JSON.stringify(body))
        : String(body);
    throw new Error(msg || `HTTP ${res.status}`);
  }

  return body as T;
}

// ---- Public API ----
export const api = {
  // Authミドルウェアが埋めた user_id を返す軽量エンドポイント
  async authMe() {
    return authedJson<{ id: string; email?: string | null; display_name?: string | null }>(
      "/auth/me"
    );
  },

  // プロフィール（display_name, avatar_url 等を返す）
  async getMe() {
    return authedJson<{
      id: string;
      email?: string | null;
      display_name?: string | null;
      avatar_file_id?: string | null;
      avatar_url?: string | null;
    }>("/users/me");
  },

  async updateMe(payload: {
    display_name?: string | null;
    avatar_file_id_or_null?: string | null;
  }) {
    await authedJson<void>("/users/me", {
      method: "PUT",
      body: JSON.stringify(payload),
    });
    return true;
  },

  async listWorkspaces() {
    return authedJson<Array<{ id: string; name: string }>>("/workspaces");
  },

  async createWorkspace(name: string) {
    return authedJson<{ id: string }>("/workspaces", {
      method: "POST",
      body: JSON.stringify({ name: name.trim() }),
    });
  },

  async listChannels(wsId: string) {
    return authedJson<Array<{ id: string; name: string; is_private: boolean }>>(
      `/workspaces/${wsId}/channels`
    );
  },

  async isChannelMember(channelId: string) {
    return authedJson<{
      is_member: boolean;
      can_read: boolean;
      can_post: boolean;
      role: "owner" | "member" | "none";
      is_private: boolean;
    }>(`/channels/${channelId}/membership`);
  },

  async joinSelf(wsId: string, channelId: string) {
    return authedJson<{ ok: boolean }>(
      `/workspaces/${wsId}/channels/${channelId}/join`,
      { method: "POST" }
    );
  },

  async createChannel(wsId: string, name: string, isPrivate = false) {
    return authedJson<{ id: string }>(`/workspaces/${wsId}/channels`, {
      method: "POST",
      body: JSON.stringify({ name: name.trim(), is_private: isPrivate }),
    });
  },

  async listMessages(channelId: string) {
    return authedJson<Array<Msg>>(`/channels/${channelId}/messages`);
  },

  async postMessage(channelId: string, text: string, parentId?: string) {
    return authedJson<Msg>(`/channels/${channelId}/messages`, {
      method: "POST",
      body: JSON.stringify({ text, parent_id: parentId ?? null }),
    });
  },

  async searchUsers(q: string, limit = 20) {
    const p = new URLSearchParams({ q, limit: String(limit) });
    return authedJson<Array<{ id: string; email: string; display_name?: string }>>(
      `/users/search?${p}`
    );
  },

  async addWorkspaceMember(
    wsId: string,
    userId: string,
    role: "owner" | "member" = "member"
  ) {
    return authedJson<{ ok: boolean }>(`/workspaces/${wsId}/members`, {
      method: "POST",
      body: JSON.stringify({ user_id: userId, role }),
    });
  },

  async searchChannelMemberCandidates(channelId: string, q: string, limit = 20) {
    const p = new URLSearchParams({ q, limit: String(limit) });
    return authedJson<Array<{ id: string; email: string; display_name?: string }>>(
      `/channels/${channelId}/members/search?${p}`
    );
  },

  async addChannelMember(
    channelId: string,
    userId: string,
    role: "owner" | "member" = "member"
  ) {
    return authedJson<{ ok: boolean }>(`/channels/${channelId}/members`, {
      method: "POST",
      body: JSON.stringify({ user_id: userId, role }),
    });
  },

  // アバター用の署名URL発行
  async signUploadAvatar(input: {
    filename: string;
    content_type: string;
    size_bytes: number;
  }) {
    return authedJson<{ upload_url: string; storage_key: string; file_id: string }>(
      "/users/me/avatar/sign-upload",
      {
        method: "POST",
        body: JSON.stringify(input),
      }
    );
  },

  async signUploadMessage(
    wsId: string,
    chId: string,
    input: { filename: string; content_type: string; size_bytes: number }
  ) {
    return authedJson<{ upload_url: string; storage_key: string; file_id: string }>(
      `/workspaces/${wsId}/channels/${chId}/files/sign-upload`,
      {
        method: "POST",
        body: JSON.stringify(input),
      }
    );
  },

  // 完了報告（purpose を渡せるように）
  async completeFile(payload: any) {
    return authedJson<FileRec>("/files/complete", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  // 任意ファイルの署名GET URLを取得
  async getFileURL(fileId: string, disposition: "inline" | "attachment" = "inline") {
    return authedJson<{ url: string; expires_at: string }>(
      `/files/${fileId}/url?disposition=${disposition}`
    );
  },
};

// ---- WebSocket URL helper ----
export const wsURL = (channelId: string) => {
  const base = import.meta.env.VITE_WS_BASE as string;
  const url = new URL("/ws", base);
  url.searchParams.set("channel_id", channelId);
  return url.toString();
};
