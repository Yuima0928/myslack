-- +goose Up
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS external_id varchar(255) UNIQUE;

-- +goose Down
ALTER TABLE users
  DROP COLUMN IF EXISTS external_id;
