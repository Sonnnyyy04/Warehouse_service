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

type RackRepository struct {
	db Querier
}

func NewRackRepository(pool *pgxpool.Pool) *RackRepository {
	return NewRackRepositoryWithQuerier(pool)
}

func NewRackRepositoryWithQuerier(db Querier) *RackRepository {
	return &RackRepository{db: db}
}

func (r *RackRepository) List(ctx context.Context, limit int32) ([]models.Rack, error) {
	const query = `
SELECT id, code, name, zone, status
FROM racks
ORDER BY id
LIMIT $1
`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list racks: %w", err)
	}
	defer rows.Close()

	racks := make([]models.Rack, 0)
	for rows.Next() {
		var rack models.Rack
		if err := rows.Scan(&rack.ID, &rack.Code, &rack.Name, &rack.Zone, &rack.Status); err != nil {
			return nil, fmt.Errorf("scan rack row: %w", err)
		}
		racks = append(racks, rack)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rack rows: %w", err)
	}

	return racks, nil
}

func (r *RackRepository) GetByID(ctx context.Context, id int64) (models.Rack, error) {
	const query = `
SELECT id, code, name, zone, status
FROM racks
WHERE id = $1
`

	var rack models.Rack
	if err := r.db.QueryRow(ctx, query, id).Scan(
		&rack.ID,
		&rack.Code,
		&rack.Name,
		&rack.Zone,
		&rack.Status,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Rack{}, ErrNotFound
		}
		return models.Rack{}, fmt.Errorf("get rack by id: %w", err)
	}

	return rack, nil
}

func (r *RackRepository) ListByIDs(ctx context.Context, ids []int64) ([]models.Rack, error) {
	if len(ids) == 0 {
		return []models.Rack{}, nil
	}

	const query = `
SELECT id, code, name, zone, status
FROM racks
WHERE id = ANY($1)
`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("list racks by ids: %w", err)
	}
	defer rows.Close()

	racks := make([]models.Rack, 0, len(ids))
	for rows.Next() {
		var rack models.Rack
		if err := rows.Scan(&rack.ID, &rack.Code, &rack.Name, &rack.Zone, &rack.Status); err != nil {
			return nil, fmt.Errorf("scan rack by id row: %w", err)
		}
		racks = append(racks, rack)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rack by ids rows: %w", err)
	}

	return racks, nil
}

func (r *RackRepository) Create(ctx context.Context, code, name string, zone *string, status string) (models.Rack, error) {
	const query = `
INSERT INTO racks (code, name, zone, status)
VALUES ($1, $2, $3, $4)
RETURNING id, code, name, zone, status
`

	var rack models.Rack
	if err := r.db.QueryRow(ctx, query, code, name, zone, status).Scan(
		&rack.ID,
		&rack.Code,
		&rack.Name,
		&rack.Zone,
		&rack.Status,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Rack{}, ErrConflict
		}
		return models.Rack{}, fmt.Errorf("create rack: %w", err)
	}

	return rack, nil
}

func (r *RackRepository) Update(ctx context.Context, id int64, code, name string, zone *string, status string) (models.Rack, error) {
	const query = `
UPDATE racks
SET code = $2,
    name = $3,
    zone = $4,
    status = $5
WHERE id = $1
RETURNING id, code, name, zone, status
`

	var rack models.Rack
	if err := r.db.QueryRow(ctx, query, id, code, name, zone, status).Scan(
		&rack.ID,
		&rack.Code,
		&rack.Name,
		&rack.Zone,
		&rack.Status,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Rack{}, ErrConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Rack{}, ErrNotFound
		}
		return models.Rack{}, fmt.Errorf("update rack: %w", err)
	}

	return rack, nil
}

func (r *RackRepository) HasAnyStorageCells(ctx context.Context, rackID int64) (bool, error) {
	const query = `
SELECT EXISTS (
    SELECT 1
    FROM storage_cells
    WHERE rack_id = $1
)
`

	var exists bool
	if err := r.db.QueryRow(ctx, query, rackID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check rack storage cells: %w", err)
	}

	return exists, nil
}

func (r *RackRepository) DeleteByID(ctx context.Context, id int64) error {
	const query = `
DELETE FROM racks
WHERE id = $1
`

	commandTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete rack: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RackRepository) GetContentStats(ctx context.Context, rackID int64) (models.ObjectContentStats, error) {
	const query = `
WITH rack_cells AS (
    SELECT id
    FROM storage_cells
    WHERE rack_id = $1
),
rack_boxes AS (
    SELECT id
    FROM boxes
    WHERE storage_cell_id IN (SELECT id FROM rack_cells)
),
rack_batches AS (
    SELECT b.product_id, b.quantity
    FROM batches b
    WHERE b.storage_cell_id IN (SELECT id FROM rack_cells)
       OR b.box_id IN (SELECT id FROM rack_boxes)
),
product_counts AS (
    SELECT COUNT(DISTINCT product_id)::INT AS products_count
    FROM rack_batches
)
SELECT
    (SELECT COUNT(*)::INT FROM rack_cells) AS cells_count,
    (SELECT COUNT(*)::INT FROM rack_boxes) AS boxes_count,
    (SELECT COUNT(*)::INT FROM rack_batches) AS batches_count,
    COALESCE((SELECT products_count FROM product_counts), 0) AS products_count,
    COALESCE((SELECT SUM(quantity)::INT FROM rack_batches), 0) AS total_quantity
`

	var stats models.ObjectContentStats
	if err := r.db.QueryRow(ctx, query, rackID).Scan(
		&stats.CellsCount,
		&stats.BoxesCount,
		&stats.BatchesCount,
		&stats.ProductsCount,
		&stats.TotalQuantity,
	); err != nil {
		return models.ObjectContentStats{}, fmt.Errorf("get rack content stats: %w", err)
	}

	return stats, nil
}
