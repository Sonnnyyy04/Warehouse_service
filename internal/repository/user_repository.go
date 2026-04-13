package repository

import (
	"context"
	"errors"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (models.User, error) {
	const query = `
SELECT id, login, email, full_name, role, password_hash
FROM users
WHERE login = $1
`

	var user models.User

	if err := r.pool.QueryRow(ctx, query, login).Scan(
		&user.ID,
		&user.Login,
		&user.Email,
		&user.FullName,
		&user.Role,
		&user.PasswordHash,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, fmt.Errorf("get user by login: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (models.User, error) {
	const query = `
SELECT id, login, email, full_name, role, password_hash
FROM users
WHERE id = $1
`

	var user models.User

	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Login,
		&user.Email,
		&user.FullName,
		&user.Role,
		&user.PasswordHash,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (r *UserRepository) ListByRole(ctx context.Context, role string, limit int32) ([]models.User, error) {
	const query = `
SELECT id, login, email, full_name, role, password_hash
FROM users
WHERE role = $1
ORDER BY id
LIMIT $2
`

	rows, err := r.pool.Query(ctx, query, role, limit)
	if err != nil {
		return nil, fmt.Errorf("list users by role: %w", err)
	}
	defer rows.Close()

	users := make([]models.User, 0)

	for rows.Next() {
		var user models.User

		if err := rows.Scan(
			&user.ID,
			&user.Login,
			&user.Email,
			&user.FullName,
			&user.Role,
			&user.PasswordHash,
		); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user rows: %w", err)
	}

	return users, nil
}

func (r *UserRepository) Create(ctx context.Context, login, email, fullName, role, passwordHash string) (models.User, error) {
	const query = `
INSERT INTO users (login, email, full_name, role, password_hash)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, login, email, full_name, role, password_hash
`

	var user models.User

	if err := r.pool.QueryRow(ctx, query, login, email, fullName, role, passwordHash).Scan(
		&user.ID,
		&user.Login,
		&user.Email,
		&user.FullName,
		&user.Role,
		&user.PasswordHash,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.User{}, ErrConflict
		}
		return models.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
