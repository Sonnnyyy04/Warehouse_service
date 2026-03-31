package repository

import (
	"context"
	"errors"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PalletRepository struct {
	pool *pgxpool.Pool
}

func NewPalletRepository(pool *pgxpool.Pool) *PalletRepository {
	return &PalletRepository{pool: pool}
}

func (r *PalletRepository) GetByID(ctx context.Context, id int64) (models.Pallet, error) {
	const query = `
SELECT id, code, status, storage_cell_id
FROM pallets
WHERE id = $1
`

	var pallet models.Pallet

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&pallet.ID,
		&pallet.Code,
		&pallet.Status,
		&pallet.StorageCellID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Pallet{}, ErrNotFound
		}
		return models.Pallet{}, fmt.Errorf("get pallet by id: %w", err)
	}

	return pallet, nil
}
