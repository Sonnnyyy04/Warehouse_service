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

type BatchRepository struct {
	db Querier
}

func NewBatchRepository(pool *pgxpool.Pool) *BatchRepository {
	return NewBatchRepositoryWithQuerier(pool)
}

func NewBatchRepositoryWithQuerier(db Querier) *BatchRepository {
	return &BatchRepository{db: db}
}

func (r *BatchRepository) GetByID(ctx context.Context, id int64) (models.Batch, error) {
	const query = `
SELECT id, code, product_id, quantity, status, box_id, pallet_id, storage_cell_id
FROM batches
WHERE id = $1
`

	var batch models.Batch

	err := r.db.QueryRow(ctx, query, id).Scan(
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

func (r *BatchRepository) List(ctx context.Context, limit int32) ([]models.Batch, error) {
	const query = `
SELECT id, code, product_id, quantity, status, box_id, pallet_id, storage_cell_id
FROM batches
ORDER BY id
LIMIT $1
`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list batches: %w", err)
	}
	defer rows.Close()

	batches := make([]models.Batch, 0)

	for rows.Next() {
		var batch models.Batch

		if err := rows.Scan(
			&batch.ID,
			&batch.Code,
			&batch.ProductID,
			&batch.Quantity,
			&batch.Status,
			&batch.BoxID,
			&batch.PalletID,
			&batch.StorageCellID,
		); err != nil {
			return nil, fmt.Errorf("scan batch row: %w", err)
		}

		batches = append(batches, batch)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate batch rows: %w", err)
	}

	return batches, nil
}

func (r *BatchRepository) HasOtherProductInBox(
	ctx context.Context,
	boxID int64,
	productID int64,
	excludeBatchID *int64,
) (bool, error) {
	const query = `
SELECT EXISTS (
    SELECT 1
    FROM batches
    WHERE box_id = $1
      AND product_id <> $2
      AND ($3::BIGINT IS NULL OR id <> $3)
)
`

	var exists bool
	if err := r.db.QueryRow(ctx, query, boxID, productID, excludeBatchID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check box product compatibility: %w", err)
	}

	return exists, nil
}

func (r *BatchRepository) HasAnyInBox(ctx context.Context, boxID int64) (bool, error) {
	const query = `
SELECT EXISTS (
    SELECT 1
    FROM batches
    WHERE box_id = $1
)
`

	var exists bool
	if err := r.db.QueryRow(ctx, query, boxID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check box occupancy: %w", err)
	}

	return exists, nil
}

func (r *BatchRepository) HasAnyInStorageCell(ctx context.Context, storageCellID int64) (bool, error) {
	const query = `
SELECT EXISTS (
    SELECT 1
    FROM batches
    WHERE storage_cell_id = $1
)
`

	var exists bool
	if err := r.db.QueryRow(ctx, query, storageCellID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check storage cell occupancy: %w", err)
	}

	return exists, nil
}

func (r *BatchRepository) HasAnyForProduct(ctx context.Context, productID int64) (bool, error) {
	const query = `
SELECT EXISTS (
    SELECT 1
    FROM batches
    WHERE product_id = $1
)
`

	var exists bool
	if err := r.db.QueryRow(ctx, query, productID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check product batch existence: %w", err)
	}

	return exists, nil
}

func (r *BatchRepository) MoveToBox(ctx context.Context, batchID, boxID int64) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE batches
		SET box_id = $2,
		    pallet_id = NULL,
		    storage_cell_id = NULL
		WHERE id = $1
	`, batchID, boxID)
	if err != nil {
		return fmt.Errorf("move batch to box: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *BatchRepository) MoveToStorageCell(ctx context.Context, batchID, storageCellID int64) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE batches
		SET box_id = NULL,
		    pallet_id = NULL,
		    storage_cell_id = $2
		WHERE id = $1
	`, batchID, storageCellID)
	if err != nil {
		return fmt.Errorf("move batch to storage cell: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *BatchRepository) Create(
	ctx context.Context,
	code string,
	productID int64,
	quantity int32,
	status string,
	boxID *int64,
	palletID *int64,
	storageCellID *int64,
) (models.Batch, error) {
	const query = `
INSERT INTO batches (code, product_id, quantity, status, box_id, pallet_id, storage_cell_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, code, product_id, quantity, status, box_id, pallet_id, storage_cell_id
`

	var batch models.Batch

	if err := r.db.QueryRow(
		ctx,
		query,
		code,
		productID,
		quantity,
		status,
		boxID,
		palletID,
		storageCellID,
	).Scan(
		&batch.ID,
		&batch.Code,
		&batch.ProductID,
		&batch.Quantity,
		&batch.Status,
		&batch.BoxID,
		&batch.PalletID,
		&batch.StorageCellID,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Batch{}, ErrConflict
		}
		return models.Batch{}, fmt.Errorf("create batch: %w", err)
	}

	return batch, nil
}

func (r *BatchRepository) Update(
	ctx context.Context,
	id int64,
	code string,
	productID int64,
	quantity int32,
	status string,
	boxID *int64,
	palletID *int64,
	storageCellID *int64,
) (models.Batch, error) {
	const query = `
UPDATE batches
SET code = $2,
    product_id = $3,
    quantity = $4,
    status = $5,
    box_id = $6,
    pallet_id = $7,
    storage_cell_id = $8
WHERE id = $1
RETURNING id, code, product_id, quantity, status, box_id, pallet_id, storage_cell_id
`

	var batch models.Batch

	if err := r.db.QueryRow(
		ctx,
		query,
		id,
		code,
		productID,
		quantity,
		status,
		boxID,
		palletID,
		storageCellID,
	).Scan(
		&batch.ID,
		&batch.Code,
		&batch.ProductID,
		&batch.Quantity,
		&batch.Status,
		&batch.BoxID,
		&batch.PalletID,
		&batch.StorageCellID,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Batch{}, ErrConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Batch{}, ErrNotFound
		}
		return models.Batch{}, fmt.Errorf("update batch: %w", err)
	}

	return batch, nil
}

func (r *BatchRepository) DeleteByID(ctx context.Context, id int64) error {
	const query = `
DELETE FROM batches
WHERE id = $1
`

	commandTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete batch: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
