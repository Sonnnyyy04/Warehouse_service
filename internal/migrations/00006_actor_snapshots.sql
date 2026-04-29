-- +goose Up

ALTER TABLE scan_events
ADD COLUMN actor_user_id BIGINT,
ADD COLUMN actor_login TEXT,
ADD COLUMN actor_full_name TEXT,
ADD COLUMN actor_role TEXT,
ADD COLUMN actor_is_super_admin BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE scan_events se
SET actor_user_id = u.id,
    actor_login = u.login,
    actor_full_name = u.full_name,
    actor_role = u.role,
    actor_is_super_admin = u.is_super_admin
FROM users u
WHERE se.user_id = u.id;

ALTER TABLE operation_history
ADD COLUMN actor_user_id BIGINT,
ADD COLUMN actor_login TEXT,
ADD COLUMN actor_full_name TEXT,
ADD COLUMN actor_role TEXT,
ADD COLUMN actor_is_super_admin BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE operation_history oh
SET actor_user_id = u.id,
    actor_login = u.login,
    actor_full_name = u.full_name,
    actor_role = u.role,
    actor_is_super_admin = u.is_super_admin
FROM users u
WHERE oh.user_id = u.id;

-- +goose Down

ALTER TABLE operation_history
DROP COLUMN IF EXISTS actor_is_super_admin,
DROP COLUMN IF EXISTS actor_role,
DROP COLUMN IF EXISTS actor_full_name,
DROP COLUMN IF EXISTS actor_login,
DROP COLUMN IF EXISTS actor_user_id;

ALTER TABLE scan_events
DROP COLUMN IF EXISTS actor_is_super_admin,
DROP COLUMN IF EXISTS actor_role,
DROP COLUMN IF EXISTS actor_full_name,
DROP COLUMN IF EXISTS actor_login,
DROP COLUMN IF EXISTS actor_user_id;
