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
SELECT c.id, c.code, c.name, c.zone, c.status, c.rack_id, r.code, r.name
FROM storage_cells c
LEFT JOIN racks r ON r.id = c.rack_id
WHERE c.id = $1
`

	var cell models.StorageCell

	err := r.db.QueryRow(ctx, query, id).Scan(
		&cell.ID,
		&cell.Code,
		&cell.Name,
		&cell.Zone,
		&cell.Status,
		&cell.RackID,
		&cell.RackCode,
		&cell.RackName,
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
SELECT c.id, c.code, c.name, c.zone, c.status, c.rack_id, r.code, r.name
FROM storage_cells c
LEFT JOIN racks r ON r.id = c.rack_id
WHERE LOWER(c.code) = LOWER($1)
LIMIT 1
`

	var cell models.StorageCell

	err := r.db.QueryRow(ctx, query, code).Scan(
		&cell.ID,
		&cell.Code,
		&cell.Name,
		&cell.Zone,
		&cell.Status,
		&cell.RackID,
		&cell.RackCode,
		&cell.RackName,
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
SELECT c.id, c.code, c.name, c.zone, c.status, c.rack_id, r.code, r.name
FROM storage_cells c
LEFT JOIN racks r ON r.id = c.rack_id
ORDER BY c.id
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
			&cell.RackID,
			&cell.RackCode,
			&cell.RackName,
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

func (r *StorageCellRepository) ListByIDs(ctx context.Context, ids []int64) ([]models.StorageCell, error) {
	if len(ids) == 0 {
		return []models.StorageCell{}, nil
	}

	const query = `
SELECT c.id, c.code, c.name, c.zone, c.status, c.rack_id, r.code, r.name
FROM storage_cells c
LEFT JOIN racks r ON r.id = c.rack_id
WHERE c.id = ANY($1)
`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("list storage cells by ids: %w", err)
	}
	defer rows.Close()

	cells := make([]models.StorageCell, 0, len(ids))

	for rows.Next() {
		var cell models.StorageCell

		if err := rows.Scan(
			&cell.ID,
			&cell.Code,
			&cell.Name,
			&cell.Zone,
			&cell.Status,
			&cell.RackID,
			&cell.RackCode,
			&cell.RackName,
		); err != nil {
			return nil, fmt.Errorf("scan storage cell by id row: %w", err)
		}

		cells = append(cells, cell)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate storage cell by ids rows: %w", err)
	}

	return cells, nil
}

func (r *StorageCellRepository) GetContentStats(ctx context.Context, storageCellID int64) (models.ObjectContentStats, error) {
	const query = `
WITH cell_boxes AS (
    SELECT id
    FROM boxes
    WHERE storage_cell_id = $1
),
cell_batches AS (
    SELECT b.product_id, b.quantity
    FROM batches b
    WHERE b.storage_cell_id = $1
       OR b.box_id IN (SELECT id FROM cell_boxes)
),
product_counts AS (
    SELECT COUNT(DISTINCT product_id)::INT AS products_count
    FROM cell_batches
)
SELECT
    (SELECT COUNT(*)::INT FROM cell_boxes) AS boxes_count,
    (SELECT COUNT(*)::INT FROM cell_batches) AS batches_count,
    COALESCE((SELECT products_count FROM product_counts), 0) AS products_count,
    COALESCE((SELECT SUM(quantity)::INT FROM cell_batches), 0) AS total_quantity
`

	var stats models.ObjectContentStats
	if err := r.db.QueryRow(ctx, query, storageCellID).Scan(
		&stats.BoxesCount,
		&stats.BatchesCount,
		&stats.ProductsCount,
		&stats.TotalQuantity,
	); err != nil {
		return models.ObjectContentStats{}, fmt.Errorf("get storage cell content stats: %w", err)
	}

	return stats, nil
}

func (r *StorageCellRepository) Create(ctx context.Context, code, name string, zone *string, status string, rackID *int64) (models.StorageCell, error) {
	const query = `
WITH inserted AS (
    INSERT INTO storage_cells (code, name, zone, status, rack_id)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, code, name, zone, status, rack_id
)
SELECT i.id, i.code, i.name, i.zone, i.status, i.rack_id, r.code, r.name
FROM inserted i
LEFT JOIN racks r ON r.id = i.rack_id
`

	var cell models.StorageCell

	if err := r.db.QueryRow(ctx, query, code, name, zone, status, rackID).Scan(
		&cell.ID,
		&cell.Code,
		&cell.Name,
		&cell.Zone,
		&cell.Status,
		&cell.RackID,
		&cell.RackCode,
		&cell.RackName,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.StorageCell{}, ErrConflict
		}
		return models.StorageCell{}, fmt.Errorf("create storage cell: %w", err)
	}

	return cell, nil
}

func (r *StorageCellRepository) Update(ctx context.Context, id int64, code, name string, zone *string, status string, rackID *int64) (models.StorageCell, error) {
	const query = `
WITH updated AS (
    UPDATE storage_cells
    SET code = $2,
        name = $3,
        zone = $4,
        status = $5,
        rack_id = $6
    WHERE id = $1
    RETURNING id, code, name, zone, status, rack_id
)
SELECT u.id, u.code, u.name, u.zone, u.status, u.rack_id, r.code, r.name
FROM updated u
LEFT JOIN racks r ON r.id = u.rack_id
`

	var cell models.StorageCell

	if err := r.db.QueryRow(ctx, query, id, code, name, zone, status, rackID).Scan(
		&cell.ID,
		&cell.Code,
		&cell.Name,
		&cell.Zone,
		&cell.Status,
		&cell.RackID,
		&cell.RackCode,
		&cell.RackName,
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
