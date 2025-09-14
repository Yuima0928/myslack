export type TokenOut = { access_token: string; token_type: string };
export type Me = { id: string; email: string; display_name?: string | null };
export type Message = {
  id: string;
  workspace_id: string;
  channel_id: string;
  user_id: string;
  text: string;
};
