-- +goose Up
ALTER TABLE users
    ADD COLUMN avatar_file_id uuid NULL;

ALTER TABLE users
    ADD CONSTRAINT fk_users_avatar_file
    FOREIGN KEY (avatar_file_id) REFERENCES files(id) ON DELETE SET NULL;

CREATE INDEX idx_users_avatar_file_id ON users(avatar_file_id);

-- +goose Down
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_avatar_file;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_file_id;
DROP INDEX IF EXISTS idx_users_avatar_file_id;
