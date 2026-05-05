package repository

import (
	"context"
	"errors"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductAliasRepository struct {
	db Querier
}

func NewProductAliasRepository(pool *pgxpool.Pool) *ProductAliasRepository {
	return NewProductAliasRepositoryWithQuerier(pool)
}

func NewProductAliasRepositoryWithQuerier(db Querier) *ProductAliasRepository {
	return &ProductAliasRepository{db: db}
}

func (r *ProductAliasRepository) GetBySupplierArticle(ctx context.Context, supplierName, article string) (models.ProductAlias, error) {
	const query = `
SELECT id, product_id, supplier_name, alias_type, alias_value, created_at
FROM product_aliases
WHERE LOWER(supplier_name) = LOWER($1)
  AND alias_type = 'supplier_article'
  AND LOWER(alias_value) = LOWER($2)
LIMIT 1
`

	var alias models.ProductAlias
	if err := r.db.QueryRow(ctx, query, supplierName, article).Scan(
		&alias.ID,
		&alias.ProductID,
		&alias.SupplierName,
		&alias.AliasType,
		&alias.AliasValue,
		&alias.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ProductAlias{}, ErrNotFound
		}
		return models.ProductAlias{}, fmt.Errorf("get product alias: %w", err)
	}

	return alias, nil
}

func (r *ProductAliasRepository) UpsertSupplierArticle(ctx context.Context, productID int64, supplierName, article string) (models.ProductAlias, error) {
	const query = `
INSERT INTO product_aliases (product_id, supplier_name, alias_type, alias_value)
VALUES ($1, $2, 'supplier_article', $3)
ON CONFLICT (supplier_name, alias_type, alias_value)
DO UPDATE SET product_id = EXCLUDED.product_id
RETURNING id, product_id, supplier_name, alias_type, alias_value, created_at
`

	var alias models.ProductAlias
	if err := r.db.QueryRow(ctx, query, productID, supplierName, article).Scan(
		&alias.ID,
		&alias.ProductID,
		&alias.SupplierName,
		&alias.AliasType,
		&alias.AliasValue,
		&alias.CreatedAt,
	); err != nil {
		return models.ProductAlias{}, fmt.Errorf("upsert product alias: %w", err)
	}

	return alias, nil
}
