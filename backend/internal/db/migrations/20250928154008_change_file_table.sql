-- +goose Up
ALTER TABLE files
    ALTER COLUMN workspace_id DROP NOT NULL,
    ALTER COLUMN channel_id DROP NOT NULL;

ALTER TABLE files
    ADD COLUMN purpose text NOT NULL DEFAULT 'message_attachment',
    ADD COLUMN owner_user_id uuid NULL;

CREATE INDEX IF NOT EXISTS idx_files_owner_user ON files(owner_user_id);

-- 可能ならチェック制約（Postgres）
-- ALTER TABLE files ADD CONSTRAINT chk_files_purpose
-- CHECK (
--   (purpose = 'message_attachment' AND workspace_id IS NOT NULL AND channel_id IS NOT NULL) OR
--   (purpose = 'avatar' AND owner_user_id IS NOT NULL)
-- );

-- +goose Down
ALTER TABLE files DROP COLUMN IF EXISTS owner_user_id;
ALTER TABLE files DROP COLUMN IF EXISTS purpose;
-- （必要なら NOT NULL を戻す）
