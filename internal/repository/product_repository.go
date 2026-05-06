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

type ProductRepository struct {
	db Querier
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return NewProductRepositoryWithQuerier(pool)
}

func NewProductRepositoryWithQuerier(db Querier) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) GetByID(ctx context.Context, id int64) (models.Product, error) {
	const query = `
SELECT p.id, p.sku, p.name, p.unit, COALESCE(SUM(b.quantity), 0) AS total_quantity
FROM products p
LEFT JOIN batches b ON b.product_id = p.id
WHERE p.id = $1
GROUP BY p.id, p.sku, p.name, p.unit
`

	var product models.Product

	err := r.db.QueryRow(ctx, query, id).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Unit,
		&product.TotalQuantity,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrNotFound
		}
		return models.Product{}, fmt.Errorf("get product by id: %w", err)
	}

	return product, nil
}

func (r *ProductRepository) GetByName(ctx context.Context, name string) (models.Product, error) {
	const query = `
SELECT p.id, p.sku, p.name, p.unit, COALESCE(SUM(b.quantity), 0) AS total_quantity
FROM products p
LEFT JOIN batches b ON b.product_id = p.id
WHERE LOWER(p.name) = LOWER($1)
GROUP BY p.id, p.sku, p.name, p.unit
LIMIT 1
`

	var product models.Product

	err := r.db.QueryRow(ctx, query, name).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Unit,
		&product.TotalQuantity,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrNotFound
		}
		return models.Product{}, fmt.Errorf("get product by name: %w", err)
	}

	return product, nil
}

func (r *ProductRepository) GetBySKU(ctx context.Context, sku string) (models.Product, error) {
	const query = `
SELECT p.id, p.sku, p.name, p.unit, COALESCE(SUM(b.quantity), 0) AS total_quantity
FROM products p
LEFT JOIN batches b ON b.product_id = p.id
WHERE LOWER(p.sku) = LOWER($1)
GROUP BY p.id, p.sku, p.name, p.unit
LIMIT 1
`

	var product models.Product

	err := r.db.QueryRow(ctx, query, sku).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Unit,
		&product.TotalQuantity,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrNotFound
		}
		return models.Product{}, fmt.Errorf("get product by sku: %w", err)
	}

	return product, nil
}

func (r *ProductRepository) List(ctx context.Context, limit int32) ([]models.Product, error) {
	const query = `
SELECT p.id, p.sku, p.name, p.unit, COALESCE(SUM(b.quantity), 0) AS total_quantity
FROM products p
LEFT JOIN batches b ON b.product_id = p.id
GROUP BY p.id, p.sku, p.name, p.unit
ORDER BY p.id
LIMIT $1
`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products := make([]models.Product, 0)

	for rows.Next() {
		var product models.Product

		if err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Unit,
			&product.TotalQuantity,
		); err != nil {
			return nil, fmt.Errorf("scan product row: %w", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product rows: %w", err)
	}

	return products, nil
}

func (r *ProductRepository) Search(ctx context.Context, query string, limit int32) ([]models.Product, error) {
	const sql = `
SELECT p.id, p.sku, p.name, p.unit, COALESCE(SUM(b.quantity), 0) AS total_quantity
FROM products p
LEFT JOIN batches b ON b.product_id = p.id
WHERE $1 = ''
   OR LOWER(p.sku) LIKE '%' || LOWER($1) || '%'
   OR LOWER(p.name) LIKE '%' || LOWER($1) || '%'
GROUP BY p.id, p.sku, p.name, p.unit
ORDER BY p.name, p.sku
LIMIT $2
`

	rows, err := r.db.Query(ctx, sql, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search products: %w", err)
	}
	defer rows.Close()

	products := make([]models.Product, 0)

	for rows.Next() {
		var product models.Product

		if err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Unit,
			&product.TotalQuantity,
		); err != nil {
			return nil, fmt.Errorf("scan product search row: %w", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product search rows: %w", err)
	}

	return products, nil
}

func (r *ProductRepository) ListByIDs(ctx context.Context, ids []int64) ([]models.Product, error) {
	if len(ids) == 0 {
		return []models.Product{}, nil
	}

	const query = `
SELECT p.id, p.sku, p.name, p.unit, COALESCE(SUM(b.quantity), 0) AS total_quantity
FROM products p
LEFT JOIN batches b ON b.product_id = p.id
WHERE p.id = ANY($1)
GROUP BY p.id, p.sku, p.name, p.unit
`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("list products by ids: %w", err)
	}
	defer rows.Close()

	products := make([]models.Product, 0, len(ids))

	for rows.Next() {
		var product models.Product

		if err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.Unit,
			&product.TotalQuantity,
		); err != nil {
			return nil, fmt.Errorf("scan product by id row: %w", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product by ids rows: %w", err)
	}

	return products, nil
}

func (r *ProductRepository) ListLocations(ctx context.Context, productID int64) ([]models.ProductLocation, error) {
	const query = `
SELECT
    bx.id,
    bx.code,
    box_marker.marker_code,
    bx.status,
    b.id,
    b.code,
    batch_marker.marker_code,
    b.quantity,
    sc.id,
    sc.code,
    sc.name,
    rck.id,
    rck.code,
    rck.name
FROM batches b
JOIN boxes bx ON bx.id = b.box_id
LEFT JOIN storage_cells sc ON sc.id = bx.storage_cell_id
LEFT JOIN racks rck ON rck.id = sc.rack_id
LEFT JOIN markers box_marker ON box_marker.object_type::text = 'box' AND box_marker.object_id = bx.id
LEFT JOIN markers batch_marker ON batch_marker.object_type::text = 'batch' AND batch_marker.object_id = b.id
WHERE b.product_id = $1
  AND b.quantity > 0
  AND b.status = 'active'
  AND bx.status = 'active'
ORDER BY rck.code NULLS LAST, sc.code NULLS LAST, bx.code, b.code
`

	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("list product locations: %w", err)
	}
	defer rows.Close()

	locations := make([]models.ProductLocation, 0)

	for rows.Next() {
		var location models.ProductLocation

		if err := rows.Scan(
			&location.BoxID,
			&location.BoxCode,
			&location.BoxMarkerCode,
			&location.BoxStatus,
			&location.BatchID,
			&location.BatchCode,
			&location.BatchMarkerCode,
			&location.Quantity,
			&location.StorageCellID,
			&location.StorageCellCode,
			&location.StorageCellName,
			&location.RackID,
			&location.RackCode,
			&location.RackName,
		); err != nil {
			return nil, fmt.Errorf("scan product location row: %w", err)
		}

		locations = append(locations, location)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product location rows: %w", err)
	}

	return locations, nil
}

func (r *ProductRepository) Create(ctx context.Context, sku, name, unit string) (models.Product, error) {
	const query = `
INSERT INTO products (sku, name, unit)
VALUES ($1, $2, $3)
RETURNING id, sku, name, unit, 0
`

	var product models.Product

	if err := r.db.QueryRow(ctx, query, sku, name, unit).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Unit,
		&product.TotalQuantity,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Product{}, ErrConflict
		}
		return models.Product{}, fmt.Errorf("create product: %w", err)
	}

	return product, nil
}

func (r *ProductRepository) Update(ctx context.Context, id int64, sku, name, unit string) (models.Product, error) {
	const query = `
WITH updated AS (
	UPDATE products
	SET sku = $2,
	    name = $3,
	    unit = $4
	WHERE id = $1
	RETURNING id, sku, name, unit
)
SELECT u.id, u.sku, u.name, u.unit, COALESCE(SUM(b.quantity), 0) AS total_quantity
FROM updated u
LEFT JOIN batches b ON b.product_id = u.id
GROUP BY u.id, u.sku, u.name, u.unit
`

	var product models.Product

	if err := r.db.QueryRow(ctx, query, id, sku, name, unit).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Unit,
		&product.TotalQuantity,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Product{}, ErrConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrNotFound
		}
		return models.Product{}, fmt.Errorf("update product: %w", err)
	}

	return product, nil
}

func (r *ProductRepository) DeleteByID(ctx context.Context, id int64) error {
	const query = `
DELETE FROM products
WHERE id = $1
`

	commandTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
