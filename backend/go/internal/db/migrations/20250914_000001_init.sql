-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- users
CREATE TABLE IF NOT EXISTS users (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email         varchar(255) NOT NULL UNIQUE,
  password_hash text NOT NULL,
  display_name  varchar(255),
  created_at    timestamptz NOT NULL DEFAULT now()
);

-- workspaces
CREATE TABLE IF NOT EXISTS workspaces (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name       varchar(255) NOT NULL,
  created_at timestamptz   NOT NULL DEFAULT now()
);

-- workspace_members
CREATE TABLE IF NOT EXISTS workspace_members (
  user_id      uuid NOT NULL,
  workspace_id uuid NOT NULL,
  role         varchar(32) NOT NULL DEFAULT 'member',
  created_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, workspace_id),
  CONSTRAINT fk_wm_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_wm_ws   FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_wm_ws ON workspace_members (workspace_id);
CREATE INDEX IF NOT EXISTS idx_wm_user ON workspace_members (user_id);

-- channels
CREATE TABLE IF NOT EXISTS channels (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL,
  name         varchar(255) NOT NULL,
  is_private   boolean NOT NULL DEFAULT false,
  created_by   uuid,
  created_at   timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT fk_ch_ws FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_ch_user FOREIGN KEY (created_by)  REFERENCES users(id)      ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_ch_ws ON channels (workspace_id);

-- channel_members
CREATE TABLE IF NOT EXISTS channel_members (
  user_id    uuid NOT NULL,
  channel_id uuid NOT NULL,
  role       varchar(32) NOT NULL DEFAULT 'member',
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, channel_id),
  CONSTRAINT fk_cm_user FOREIGN KEY (user_id)    REFERENCES users(id)    ON DELETE CASCADE,
  CONSTRAINT fk_cm_ch   FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cm_ch   ON channel_members (channel_id);
CREATE INDEX IF NOT EXISTS idx_cm_user ON channel_members (user_id);

-- messages
CREATE TABLE IF NOT EXISTS messages (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id   uuid NOT NULL,
  channel_id     uuid NOT NULL,
  user_id        uuid,
  text           text,
  parent_id      uuid,
  thread_root_id uuid,
  created_at     timestamptz NOT NULL DEFAULT now(),
  edited_at      timestamptz,
  CONSTRAINT fk_msg_ws   FOREIGN KEY (workspace_id)   REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_msg_ch   FOREIGN KEY (channel_id)     REFERENCES channels(id)   ON DELETE CASCADE,
  CONSTRAINT fk_msg_user FOREIGN KEY (user_id)        REFERENCES users(id)      ON DELETE SET NULL,
  CONSTRAINT fk_msg_parent FOREIGN KEY (parent_id)        REFERENCES messages(id) ON DELETE SET NULL,
  CONSTRAINT fk_msg_root   FOREIGN KEY (thread_root_id)  REFERENCES messages(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_msg_ch_created      ON messages (channel_id, created_at);
CREATE INDEX IF NOT EXISTS idx_msg_ch_root_created ON messages (channel_id, thread_root_id, created_at);
CREATE INDEX IF NOT EXISTS idx_msg_parent          ON messages (parent_id);

-- +goose Down
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS channel_members;
DROP TABLE IF EXISTS channels;
DROP TABLE IF EXISTS workspace_members;
DROP TABLE IF EXISTS workspaces;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS pgcrypto;
