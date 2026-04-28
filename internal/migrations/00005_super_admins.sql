-- +goose Up

ALTER TABLE users
ADD COLUMN is_super_admin BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users
SET is_super_admin = TRUE
WHERE role = 'admin';

-- +goose Down

ALTER TABLE users
DROP COLUMN IF EXISTS is_super_admin;
