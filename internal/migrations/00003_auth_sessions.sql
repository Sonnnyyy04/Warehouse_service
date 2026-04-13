-- +goose Up

CREATE TABLE user_sessions (
id BIGSERIAL PRIMARY KEY,
token TEXT NOT NULL UNIQUE,
user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
expires_at TIMESTAMPTZ NOT NULL,
last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_sessions_token ON user_sessions(token);
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);

UPDATE users
SET password_hash = '$2a$10$3E8HDB76jhPcfWKCangnTeK/6I5AAxne6eBOdjw7jTTwd2gAwxh1.',
    role = 'worker'
WHERE email = 'demo.worker@example.com';

INSERT INTO users (email, full_name, role, password_hash)
VALUES (
  'demo.admin@example.com',
  'Demo Admin',
  'admin',
  '$2a$10$hrV7NkgYaskY1.2LeSn1M.2VRZ469gKGeRd5QkjSVjgsOHRqcuGAq'
)
ON CONFLICT (email) DO UPDATE
SET full_name = EXCLUDED.full_name,
    role = EXCLUDED.role,
    password_hash = EXCLUDED.password_hash;

-- +goose Down

DELETE FROM users
WHERE email = 'demo.admin@example.com';

DROP TABLE IF EXISTS user_sessions;
