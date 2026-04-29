package repository

import (
	"context"
	"errors"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
)

type PalletRepository struct {
	db Querier
}

func NewPalletRepository(pool Querier) *PalletRepository {
	return NewPalletRepositoryWithQuerier(pool)
}

func NewPalletRepositoryWithQuerier(db Querier) *PalletRepository {
	return &PalletRepository{db: db}
}

func (r *PalletRepository) GetByID(ctx context.Context, id int64) (models.Pallet, error) {
	const query = `
SELECT id, code, status, storage_cell_id
FROM pallets
WHERE id = $1
`

	var pallet models.Pallet

	err := r.db.QueryRow(ctx, query, id).Scan(
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

func (r *PalletRepository) ListByIDs(ctx context.Context, ids []int64) ([]models.Pallet, error) {
	if len(ids) == 0 {
		return []models.Pallet{}, nil
	}

	const query = `
SELECT id, code, status, storage_cell_id
FROM pallets
WHERE id = ANY($1)
`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("list pallets by ids: %w", err)
	}
	defer rows.Close()

	pallets := make([]models.Pallet, 0, len(ids))

	for rows.Next() {
		var pallet models.Pallet

		if err := rows.Scan(
			&pallet.ID,
			&pallet.Code,
			&pallet.Status,
			&pallet.StorageCellID,
		); err != nil {
			return nil, fmt.Errorf("scan pallet by id row: %w", err)
		}

		pallets = append(pallets, pallet)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pallet by ids rows: %w", err)
	}

	return pallets, nil
}
