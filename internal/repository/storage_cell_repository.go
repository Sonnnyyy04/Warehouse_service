package repository

import (
	"context"
	"errors"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StorageCellRepository struct {
	pool *pgxpool.Pool
}

func NewStorageCellRepository(pool *pgxpool.Pool) *StorageCellRepository {
	return &StorageCellRepository{pool: pool}
}

func (r *StorageCellRepository) GetByID(ctx context.Context, id int64) (models.StorageCell, error) {
	const query = `
SELECT id, code, name, zone, status
FROM storage_cells
WHERE id = $1
`

	var cell models.StorageCell

	err := r.pool.QueryRow(ctx, query, id).Scan(
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
