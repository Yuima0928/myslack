-- +goose Up
BEGIN;

ALTER TABLE users
    DROP COLUMN IF EXISTS password_hash;

COMMIT;

-- +goose Down
BEGIN;

-- もとに戻す（値は失われているため空文字で復元）
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_hash text NOT NULL DEFAULT '';

-- 元スキーマに近づけるため DEFAULT は外しておく
ALTER TABLE users
    ALTER COLUMN password_hash DROP DEFAULT;

COMMIT;
