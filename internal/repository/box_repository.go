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

type BoxRepository struct {
	db Querier
}

func NewBoxRepository(pool *pgxpool.Pool) *BoxRepository {
	return NewBoxRepositoryWithQuerier(pool)
}

func NewBoxRepositoryWithQuerier(db Querier) *BoxRepository {
	return &BoxRepository{db: db}
}

func (r *BoxRepository) GetByID(ctx context.Context, id int64) (models.Box, error) {
	const query = `
SELECT id, code, status, pallet_id, storage_cell_id
FROM boxes
WHERE id = $1
`

	var box models.Box

	err := r.db.QueryRow(ctx, query, id).Scan(
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

func (r *BoxRepository) GetByCode(ctx context.Context, code string) (models.Box, error) {
	const query = `
SELECT id, code, status, pallet_id, storage_cell_id
FROM boxes
WHERE LOWER(code) = LOWER($1)
LIMIT 1
`

	var box models.Box

	err := r.db.QueryRow(ctx, query, code).Scan(
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
		return models.Box{}, fmt.Errorf("get box by code: %w", err)
	}

	return box, nil
}

func (r *BoxRepository) List(ctx context.Context, limit int32) ([]models.Box, error) {
	const query = `
SELECT id, code, status, pallet_id, storage_cell_id
FROM boxes
ORDER BY id
LIMIT $1
`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list boxes: %w", err)
	}
	defer rows.Close()

	boxes := make([]models.Box, 0)

	for rows.Next() {
		var box models.Box

		if err := rows.Scan(
			&box.ID,
			&box.Code,
			&box.Status,
			&box.PalletID,
			&box.StorageCellID,
		); err != nil {
			return nil, fmt.Errorf("scan box row: %w", err)
		}

		boxes = append(boxes, box)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate box rows: %w", err)
	}

	return boxes, nil
}

func (r *BoxRepository) ListByIDs(ctx context.Context, ids []int64) ([]models.Box, error) {
	if len(ids) == 0 {
		return []models.Box{}, nil
	}

	const query = `
SELECT id, code, status, pallet_id, storage_cell_id
FROM boxes
WHERE id = ANY($1)
`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("list boxes by ids: %w", err)
	}
	defer rows.Close()

	boxes := make([]models.Box, 0, len(ids))

	for rows.Next() {
		var box models.Box

		if err := rows.Scan(
			&box.ID,
			&box.Code,
			&box.Status,
			&box.PalletID,
			&box.StorageCellID,
		); err != nil {
			return nil, fmt.Errorf("scan box by id row: %w", err)
		}

		boxes = append(boxes, box)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate box by ids rows: %w", err)
	}

	return boxes, nil
}

func (r *BoxRepository) Create(ctx context.Context, code, status string, storageCellID *int64) (models.Box, error) {
	const query = `
INSERT INTO boxes (code, status, storage_cell_id)
VALUES ($1, $2, $3)
RETURNING id, code, status, pallet_id, storage_cell_id
`

	var box models.Box

	if err := r.db.QueryRow(ctx, query, code, status, storageCellID).Scan(
		&box.ID,
		&box.Code,
		&box.Status,
		&box.PalletID,
		&box.StorageCellID,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Box{}, ErrConflict
		}
		return models.Box{}, fmt.Errorf("create box: %w", err)
	}

	return box, nil
}

func (r *BoxRepository) Update(ctx context.Context, id int64, code, status string, storageCellID *int64) (models.Box, error) {
	const query = `
UPDATE boxes
SET code = $2,
    status = $3,
    pallet_id = NULL,
    storage_cell_id = $4
WHERE id = $1
RETURNING id, code, status, pallet_id, storage_cell_id
`

	var box models.Box

	if err := r.db.QueryRow(ctx, query, id, code, status, storageCellID).Scan(
		&box.ID,
		&box.Code,
		&box.Status,
		&box.PalletID,
		&box.StorageCellID,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Box{}, ErrConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Box{}, ErrNotFound
		}
		return models.Box{}, fmt.Errorf("update box: %w", err)
	}

	return box, nil
}

func (r *BoxRepository) MoveToStorageCell(ctx context.Context, boxID, storageCellID int64) error {
	cmd, err := r.db.Exec(ctx, `
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

func (r *BoxRepository) HasAnyInStorageCell(ctx context.Context, storageCellID int64) (bool, error) {
	const query = `
SELECT EXISTS (
    SELECT 1
    FROM boxes
    WHERE storage_cell_id = $1
)
`

	var exists bool
	if err := r.db.QueryRow(ctx, query, storageCellID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check storage cell box occupancy: %w", err)
	}

	return exists, nil
}

func (r *BoxRepository) GetContentStats(ctx context.Context, boxID int64) (models.ObjectContentStats, error) {
	const query = `
WITH box_batches AS (
    SELECT b.product_id, b.quantity
    FROM batches b
    WHERE b.box_id = $1
),
product_counts AS (
    SELECT COUNT(DISTINCT product_id)::INT AS products_count
    FROM box_batches
),
single_product AS (
    SELECT p.sku, p.name, p.unit
    FROM box_batches bb
    JOIN products p ON p.id = bb.product_id
    GROUP BY p.id, p.sku, p.name, p.unit
    HAVING (SELECT products_count FROM product_counts) = 1
    LIMIT 1
)
SELECT
    (SELECT COUNT(*)::INT FROM box_batches) AS batches_count,
    COALESCE((SELECT products_count FROM product_counts), 0) AS products_count,
    COALESCE((SELECT SUM(quantity)::INT FROM box_batches), 0) AS total_quantity,
    sp.sku,
    sp.name,
    sp.unit
FROM single_product sp
RIGHT JOIN (SELECT 1) anchor ON TRUE
`

	var stats models.ObjectContentStats
	if err := r.db.QueryRow(ctx, query, boxID).Scan(
		&stats.BatchesCount,
		&stats.ProductsCount,
		&stats.TotalQuantity,
		&stats.ProductSKU,
		&stats.ProductName,
		&stats.ProductUnit,
	); err != nil {
		return models.ObjectContentStats{}, fmt.Errorf("get box content stats: %w", err)
	}

	return stats, nil
}

func (r *BoxRepository) DeleteByID(ctx context.Context, id int64) error {
	const query = `
DELETE FROM boxes
WHERE id = $1
`

	commandTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete box: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
