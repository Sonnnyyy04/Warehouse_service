package repository

import (
	"context"
	"errors"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BatchRepository struct {
	pool *pgxpool.Pool
}

func NewBatchRepository(pool *pgxpool.Pool) *BatchRepository {
	return &BatchRepository{pool: pool}
}

func (r *BatchRepository) GetByID(ctx context.Context, id int64) (models.Batch, error) {
	const query = `
SELECT id, code, product_id, quantity, status, box_id, pallet_id, storage_cell_id
FROM batches
WHERE id = $1
`

	var batch models.Batch

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&batch.ID,
		&batch.Code,
		&batch.ProductID,
		&batch.Quantity,
		&batch.Status,
		&batch.BoxID,
		&batch.PalletID,
		&batch.StorageCellID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Batch{}, ErrNotFound
		}
		return models.Batch{}, fmt.Errorf("get batch by id: %w", err)
	}

	return batch, nil
}
