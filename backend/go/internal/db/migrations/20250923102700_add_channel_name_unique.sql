-- +goose Up
BEGIN;

-- 1) 前後の空白を正規化
UPDATE channels
SET name = trim(both from name)
WHERE name <> trim(both from name);

-- 2) 大小無視で重複している行にサフィックスを付けてリネーム
WITH ranked AS (
    SELECT
    id,
    workspace_id,
    name,
    ROW_NUMBER() OVER (
        PARTITION BY workspace_id, lower(name)
        ORDER BY created_at, id
    ) AS rn
    FROM channels
)
UPDATE channels c
SET name = c.name || '-' || SUBSTRING(c.id::text, 1, 8)
FROM ranked r
WHERE c.id = r.id
    AND r.rn > 1;

-- 3) 一意インデックス（大小無視）
CREATE UNIQUE INDEX IF NOT EXISTS uq_channel_ws_name
    ON channels (workspace_id, lower(name));

COMMIT;

-- +goose Down
BEGIN;
DROP INDEX IF EXISTS uq_channel_ws_name;
COMMIT;
