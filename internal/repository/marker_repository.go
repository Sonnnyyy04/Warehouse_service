package repository

import (
	"context"
	"errors"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type MarkerRepository struct {
	pool *pgxpool.Pool
}

func NewMarkerRepository(pool *pgxpool.Pool) *MarkerRepository {
	return &MarkerRepository{pool: pool}
}

func (r *MarkerRepository) GetByCode(ctx context.Context, markerCode string) (models.Marker, error) {
	const query = `
SELECT id, marker_code, object_type::text, object_id
FROM markers
WHERE marker_code = $1
`

	var marker models.Marker

	err := r.pool.QueryRow(ctx, query, markerCode).Scan(
		&marker.ID,
		&marker.MarkerCode,
		&marker.ObjectType,
		&marker.ObjectID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Marker{}, ErrNotFound
		}
		return models.Marker{}, fmt.Errorf("get marker by code: %w", err)
	}

	return marker, nil
}
