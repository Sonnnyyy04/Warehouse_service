package repository

import (
	"context"
	"errors"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
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
