-- +goose Up

ALTER TABLE users
ADD COLUMN login TEXT;

UPDATE users
SET login = 'worker'
WHERE email = 'demo.worker@example.com';

UPDATE users
SET login = 'admin'
WHERE email = 'demo.admin@example.com';

UPDATE users
SET login = 'user-' || id::text
WHERE login IS NULL OR btrim(login) = '';

ALTER TABLE users
ALTER COLUMN login SET NOT NULL;

ALTER TABLE users
ADD CONSTRAINT users_login_unique UNIQUE (login);

-- +goose Down

ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_login_unique;

ALTER TABLE users
DROP COLUMN IF EXISTS login;
