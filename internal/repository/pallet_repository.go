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

func (r *PalletRepository) GetContentStats(ctx context.Context, palletID int64) (models.ObjectContentStats, error) {
	const query = `
WITH pallet_boxes AS (
    SELECT id
    FROM boxes
    WHERE pallet_id = $1
),
pallet_batches AS (
    SELECT b.product_id, b.quantity
    FROM batches b
    WHERE b.pallet_id = $1
       OR b.box_id IN (SELECT id FROM pallet_boxes)
),
product_counts AS (
    SELECT COUNT(DISTINCT product_id)::INT AS products_count
    FROM pallet_batches
),
single_product AS (
    SELECT p.sku, p.name, p.unit
    FROM pallet_batches pb
    JOIN products p ON p.id = pb.product_id
    GROUP BY p.id, p.sku, p.name, p.unit
    HAVING (SELECT products_count FROM product_counts) = 1
    LIMIT 1
)
SELECT
    (SELECT COUNT(*)::INT FROM pallet_boxes) AS boxes_count,
    (SELECT COUNT(*)::INT FROM pallet_batches) AS batches_count,
    COALESCE((SELECT products_count FROM product_counts), 0) AS products_count,
    COALESCE((SELECT SUM(quantity)::INT FROM pallet_batches), 0) AS total_quantity,
    sp.sku,
    sp.name,
    sp.unit
FROM single_product sp
RIGHT JOIN (SELECT 1) anchor ON TRUE
`

	var stats models.ObjectContentStats
	if err := r.db.QueryRow(ctx, query, palletID).Scan(
		&stats.BoxesCount,
		&stats.BatchesCount,
		&stats.ProductsCount,
		&stats.TotalQuantity,
		&stats.ProductSKU,
		&stats.ProductName,
		&stats.ProductUnit,
	); err != nil {
		return models.ObjectContentStats{}, fmt.Errorf("get pallet content stats: %w", err)
	}

	return stats, nil
}
