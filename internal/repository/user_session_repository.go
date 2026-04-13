package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserSessionRepository struct {
	pool *pgxpool.Pool
}

func NewUserSessionRepository(pool *pgxpool.Pool) *UserSessionRepository {
	return &UserSessionRepository{pool: pool}
}

func (r *UserSessionRepository) Create(ctx context.Context, token string, userID int64, expiresAt time.Time) (models.UserSession, error) {
	const query = `
INSERT INTO user_sessions (token, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING id, token, user_id, created_at, expires_at, last_seen_at
`

	var session models.UserSession

	if err := r.pool.QueryRow(ctx, query, token, userID, expiresAt).Scan(
		&session.ID,
		&session.Token,
		&session.UserID,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LastSeenAt,
	); err != nil {
		return models.UserSession{}, fmt.Errorf("create user session: %w", err)
	}

	return session, nil
}

func (r *UserSessionRepository) GetByToken(ctx context.Context, token string) (models.UserSession, error) {
	const query = `
SELECT id, token, user_id, created_at, expires_at, last_seen_at
FROM user_sessions
WHERE token = $1
`

	var session models.UserSession

	if err := r.pool.QueryRow(ctx, query, token).Scan(
		&session.ID,
		&session.Token,
		&session.UserID,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LastSeenAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.UserSession{}, ErrNotFound
		}
		return models.UserSession{}, fmt.Errorf("get user session by token: %w", err)
	}

	return session, nil
}

func (r *UserSessionRepository) Touch(ctx context.Context, token string, lastSeenAt time.Time) error {
	cmd, err := r.pool.Exec(ctx, `
UPDATE user_sessions
SET last_seen_at = $2
WHERE token = $1
`, token, lastSeenAt)
	if err != nil {
		return fmt.Errorf("touch user session: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *UserSessionRepository) DeleteByToken(ctx context.Context, token string) error {
	cmd, err := r.pool.Exec(ctx, `
DELETE FROM user_sessions
WHERE token = $1
`, token)
	if err != nil {
		return fmt.Errorf("delete user session: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
