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
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
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

	err := r.pool.QueryRow(ctx, query, id).Scan(
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

	err := r.pool.QueryRow(ctx, query, name).Scan(
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

func (r *ProductRepository) List(ctx context.Context, limit int32) ([]models.Product, error) {
	const query = `
SELECT p.id, p.sku, p.name, p.unit, COALESCE(SUM(b.quantity), 0) AS total_quantity
FROM products p
LEFT JOIN batches b ON b.product_id = p.id
GROUP BY p.id, p.sku, p.name, p.unit
ORDER BY p.id
LIMIT $1
`

	rows, err := r.pool.Query(ctx, query, limit)
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

func (r *ProductRepository) Create(ctx context.Context, sku, name, unit string) (models.Product, error) {
	const query = `
INSERT INTO products (sku, name, unit)
VALUES ($1, $2, $3)
RETURNING id, sku, name, unit, 0
`

	var product models.Product

	if err := r.pool.QueryRow(ctx, query, sku, name, unit).Scan(
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

	if err := r.pool.QueryRow(ctx, query, id, sku, name, unit).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Unit,
		&product.TotalQuantity,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrNotFound
		}
		return models.Product{}, fmt.Errorf("update product: %w", err)
	}

	return product, nil
}
