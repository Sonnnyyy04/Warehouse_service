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

type StorageCellRepository struct {
	db Querier
}

func NewStorageCellRepository(pool *pgxpool.Pool) *StorageCellRepository {
	return NewStorageCellRepositoryWithQuerier(pool)
}

func NewStorageCellRepositoryWithQuerier(db Querier) *StorageCellRepository {
	return &StorageCellRepository{db: db}
}

func (r *StorageCellRepository) GetByID(ctx context.Context, id int64) (models.StorageCell, error) {
	const query = `
SELECT id, code, name, zone, status
FROM storage_cells
WHERE id = $1
`

	var cell models.StorageCell

	err := r.db.QueryRow(ctx, query, id).Scan(
		&cell.ID,
		&cell.Code,
		&cell.Name,
		&cell.Zone,
		&cell.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.StorageCell{}, ErrNotFound
		}
		return models.StorageCell{}, fmt.Errorf("get storage cell by id: %w", err)
	}

	return cell, nil
}

func (r *StorageCellRepository) GetByCode(ctx context.Context, code string) (models.StorageCell, error) {
	const query = `
SELECT id, code, name, zone, status
FROM storage_cells
WHERE LOWER(code) = LOWER($1)
LIMIT 1
`

	var cell models.StorageCell

	err := r.db.QueryRow(ctx, query, code).Scan(
		&cell.ID,
		&cell.Code,
		&cell.Name,
		&cell.Zone,
		&cell.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.StorageCell{}, ErrNotFound
		}
		return models.StorageCell{}, fmt.Errorf("get storage cell by code: %w", err)
	}

	return cell, nil
}

func (r *StorageCellRepository) List(ctx context.Context, limit int32) ([]models.StorageCell, error) {
	const query = `
SELECT id, code, name, zone, status
FROM storage_cells
ORDER BY id
LIMIT $1
`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list storage cells: %w", err)
	}
	defer rows.Close()

	cells := make([]models.StorageCell, 0)

	for rows.Next() {
		var cell models.StorageCell

		if err := rows.Scan(
			&cell.ID,
			&cell.Code,
			&cell.Name,
			&cell.Zone,
			&cell.Status,
		); err != nil {
			return nil, fmt.Errorf("scan storage cell row: %w", err)
		}

		cells = append(cells, cell)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate storage cell rows: %w", err)
	}

	return cells, nil
}

func (r *StorageCellRepository) Create(ctx context.Context, code, name string, zone *string, status string) (models.StorageCell, error) {
	const query = `
INSERT INTO storage_cells (code, name, zone, status)
VALUES ($1, $2, $3, $4)
RETURNING id, code, name, zone, status
`

	var cell models.StorageCell

	if err := r.db.QueryRow(ctx, query, code, name, zone, status).Scan(
		&cell.ID,
		&cell.Code,
		&cell.Name,
		&cell.Zone,
		&cell.Status,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.StorageCell{}, ErrConflict
		}
		return models.StorageCell{}, fmt.Errorf("create storage cell: %w", err)
	}

	return cell, nil
}

func (r *StorageCellRepository) Update(ctx context.Context, id int64, code, name string, zone *string, status string) (models.StorageCell, error) {
	const query = `
UPDATE storage_cells
SET code = $2,
    name = $3,
    zone = $4,
    status = $5
WHERE id = $1
RETURNING id, code, name, zone, status
`

	var cell models.StorageCell

	if err := r.db.QueryRow(ctx, query, id, code, name, zone, status).Scan(
		&cell.ID,
		&cell.Code,
		&cell.Name,
		&cell.Zone,
		&cell.Status,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.StorageCell{}, ErrConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return models.StorageCell{}, ErrNotFound
		}
		return models.StorageCell{}, fmt.Errorf("update storage cell: %w", err)
	}

	return cell, nil
}

func (r *StorageCellRepository) DeleteByID(ctx context.Context, id int64) error {
	const query = `
DELETE FROM storage_cells
WHERE id = $1
`

	commandTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete storage cell: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
