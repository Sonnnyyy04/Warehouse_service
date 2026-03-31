package repository

import (
	"context"
	"errors"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BoxRepository struct {
	pool *pgxpool.Pool
}

func NewBoxRepository(pool *pgxpool.Pool) *BoxRepository {
	return &BoxRepository{pool: pool}
}

func (r *BoxRepository) GetByID(ctx context.Context, id int64) (models.Box, error) {
	const query = `
SELECT id, code, status, pallet_id, storage_cell_id
FROM boxes
WHERE id = $1
`

	var box models.Box

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&box.ID,
		&box.Code,
		&box.Status,
		&box.PalletID,
		&box.StorageCellID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Box{}, ErrNotFound
		}
		return models.Box{}, fmt.Errorf("get box by id: %w", err)
	}

	return box, nil
}

func (r *BoxRepository) MoveToStorageCell(ctx context.Context, boxID, storageCellID int64) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE boxes
		SET pallet_id = NULL,
		    storage_cell_id = $2
		WHERE id = $1
	`, boxID, storageCellID)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
