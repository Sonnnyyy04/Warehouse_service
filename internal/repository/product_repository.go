package repository

import (
	"context"
	"errors"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
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
SELECT id, sku, name, unit
FROM products
WHERE id = $1
`

	var product models.Product

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Unit,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrNotFound
		}
		return models.Product{}, fmt.Errorf("get product by id: %w", err)
	}

	return product, nil
}

func (r *ProductRepository) List(ctx context.Context, limit int32) ([]models.Product, error) {
	const query = `
SELECT id, sku, name, unit
FROM products
ORDER BY id
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
RETURNING id, sku, name, unit
`

	var product models.Product

	if err := r.pool.QueryRow(ctx, query, sku, name, unit).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Unit,
	); err != nil {
		return models.Product{}, fmt.Errorf("create product: %w", err)
	}

	return product, nil
}

func (r *ProductRepository) Update(ctx context.Context, id int64, sku, name, unit string) (models.Product, error) {
	const query = `
UPDATE products
SET sku = $2,
    name = $3,
    unit = $4
WHERE id = $1
RETURNING id, sku, name, unit
`

	var product models.Product

	if err := r.pool.QueryRow(ctx, query, id, sku, name, unit).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Unit,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrNotFound
		}
		return models.Product{}, fmt.Errorf("update product: %w", err)
	}

	return product, nil
}
