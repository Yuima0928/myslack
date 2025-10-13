-- +goose Up
-- 0) 既存の email UNIQUE があるなら先に落としておく（任意・安全側）
DROP INDEX IF EXISTS idx_users_email;

-- 1) email / password_hash を NULL 許容に（先に制約を緩める）
ALTER TABLE users
  ALTER COLUMN email DROP NOT NULL,
  ALTER COLUMN password_hash DROP NOT NULL;

-- 2) 空文字を NULL に正規化（ここで NOT NULL 制約に引っかからない）
UPDATE users SET email = NULL WHERE email = '';

-- 3) email が NOT NULL のときだけ UNIQUE にする部分インデックス
CREATE UNIQUE INDEX IF NOT EXISTS uniq_users_email_not_null
  ON users (email) WHERE email IS NOT NULL;

-- 4) external_id は一意化（列が無い環境があるなら先に ADD COLUMN IF NOT EXISTS を入れる）
CREATE UNIQUE INDEX IF NOT EXISTS users_external_id_key
  ON users (external_id);



-- +goose Down
-- 可能な範囲で戻します（データに NULL が残っていると NOT NULL は失敗する点に注意）

DROP INDEX IF EXISTS users_external_id_key;
DROP INDEX IF EXISTS uniq_users_email_not_null;

-- 元の単純 UNIQUE を復活（必要なら）
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- ※ NOT NULL を戻す場合は、先に NULL を解消してから
-- ALTER TABLE users
--   ALTER COLUMN email SET NOT NULL,
--   ALTER COLUMN password_hash SET NOT NULL;
