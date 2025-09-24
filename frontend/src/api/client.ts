// src/api/client.ts
const API_BASE = import.meta.env.VITE_API_BASE as string;

export type Msg = {
  id: string;
  workspace_id: string;
  channel_id: string;
  user_id: string;
  text: string;
};

// （任意）サーバが返す File レコードの型
export type FileRec = {
  id: string;
  workspace_id: string;
  channel_id: string;
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
      typeof body === "object" && body && "error" in body
        ? (body as any).error?.message ?? JSON.stringify(body)
        : String(body);
    throw new Error(msg || `HTTP ${res.status}`);
  }

  return body as T;
}

// ---- Public API ----
export const api = {
  async me() {
    return authedJson<{ id: string; email: string; display_name?: string }>(
      "/auth/me"
    );
  },

  async listWorkspaces() {
    return authedJson<Array<{ id: string; name: string }>>("/workspaces");
  },

  async createWorkspace(name: string) {
    return authedJson<{ id: string }>("/workspaces", {
      method: "POST",
      body: JSON.stringify({ name }),
    });
  },

  async listChannels(wsId: string) {
    return authedJson<Array<{ id: string; name: string; is_private: boolean }>>(
      `/workspaces/${wsId}/channels`
    );
  },

  async isChannelMember(wsId: string, channelId: string) {
    return authedJson<{ is_member: boolean; is_private: boolean }>(
      `/workspaces/${wsId}/channels/${channelId}/membership`
    );
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
      body: JSON.stringify({ name, is_private: isPrivate }),
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

  async addWorkspaceMember(wsId: string, userId: string, role: "owner" | "member" = "member") {
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

  async addChannelMember(channelId: string, userId: string, role: "owner" | "member" = "member") {
    return authedJson<{ ok: boolean }>(`/channels/${channelId}/members`, {
      method: "POST",
      body: JSON.stringify({ user_id: userId, role }),
    });
  },

  // ====== ここからファイル系 ======
  async signUpload(
    wsId: string,
    chId: string,
    p: { filename: string; content_type: string; size_bytes: number }
  ) {
    return authedJson<{ upload_url: string; storage_key: string; file_id: string }>(
      `/workspaces/${wsId}/channels/${chId}/files/sign-upload`,
      {
        method: "POST",
        body: JSON.stringify(p),
      }
    );
  },

  async completeFile(p: {
    storage_key: string;
    etag: string;
    sha256_hex?: string | null;
    filename: string;
    content_type: string;
    size_bytes: number;
    workspace_id: string;
    channel_id: string;
  }) {
    return authedJson<FileRec>("/files/complete", {
      method: "POST",
      body: JSON.stringify(p),
    });
  },

  async getFileURL(fileId: string, disposition: "inline" | "attachment" = "attachment") {
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
