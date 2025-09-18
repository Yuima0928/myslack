// src/api/client.ts
const API_BASE = import.meta.env.VITE_API_BASE as string;

export type Msg = {
  id: string;
  workspace_id: string;
  channel_id: string;
  user_id: string;
  text: string;
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
  // JSONボディを送る場合は Content-Type を自動付与
  const headers = new Headers(init.headers || {});
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  const res = await authedFetch(apiPath(path), { ...init, headers });
  const body = await parseJsonOrText(res);

  if (!res.ok) {
    const msg =
      typeof body === "object" && body && "error" in body
        ? 
          (body as any).error?.message ?? JSON.stringify(body)
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
};

// ---- WebSocket URL helper ----
export const wsURL = (channelId: string) => {
  const base = import.meta.env.VITE_WS_BASE as string;
  const url = new URL("/ws", base);
  url.searchParams.set("channel_id", channelId);
  return url.toString();
};
