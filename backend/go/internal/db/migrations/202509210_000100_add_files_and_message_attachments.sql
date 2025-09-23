-- +goose Up
-- files: ファイルメタデータ本体
CREATE TABLE IF NOT EXISTS files (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id  uuid NOT NULL,
  channel_id    uuid NOT NULL,
  uploader_id   uuid NOT NULL,
  filename      text NOT NULL,
  content_type  text,
  size_bytes    bigint,
  etag          text,
  sha256_hex    text,
  storage_key   text NOT NULL,
  is_image      boolean NOT NULL DEFAULT false,
  created_at    timestamptz NOT NULL DEFAULT now(),
  deleted_at    timestamptz
);

-- 参照整合性（既に同名制約があればスキップ）
-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'files_workspace_fk') THEN
    ALTER TABLE files
      ADD CONSTRAINT files_workspace_fk
        FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
        ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'files_channel_fk') THEN
    ALTER TABLE files
      ADD CONSTRAINT files_channel_fk
        FOREIGN KEY (channel_id) REFERENCES channels(id)
        ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'files_uploader_fk') THEN
    ALTER TABLE files
      ADD CONSTRAINT files_uploader_fk
        FOREIGN KEY (uploader_id) REFERENCES users(id)
        ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END
$$;
-- +goose StatementEnd

-- パフォーマンス用インデックス
CREATE INDEX IF NOT EXISTS idx_files_channel  ON files(channel_id);
CREATE INDEX IF NOT EXISTS idx_files_uploader ON files(uploader_id);
CREATE INDEX IF NOT EXISTS idx_files_created  ON files(created_at);

-- message_attachments: メッセージとファイルの N:N
CREATE TABLE IF NOT EXISTS message_attachments (
  message_id uuid NOT NULL,
  file_id    uuid NOT NULL,
  PRIMARY KEY (message_id, file_id)
);

-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'msgatt_message_fk') THEN
    ALTER TABLE message_attachments
      ADD CONSTRAINT msgatt_message_fk
        FOREIGN KEY (message_id) REFERENCES messages(id)
        ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'msgatt_file_fk') THEN
    ALTER TABLE message_attachments
      ADD CONSTRAINT msgatt_file_fk
        FOREIGN KEY (file_id) REFERENCES files(id)
        ON UPDATE CASCADE ON DELETE CASCADE;
  END IF;
END
$$;
-- +goose StatementEnd



-- +goose Down
-- 依存順で落とす（存在すれば）
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relname = 'message_attachments') THEN
    DROP TABLE message_attachments;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_class WHERE relname = 'files') THEN
    DROP TABLE files;
  END IF;
END
$$;
-- +goose StatementEnd
