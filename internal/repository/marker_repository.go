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

func (r *MarkerRepository) List(ctx context.Context, objectType string, limit int32) ([]models.Marker, error) {
	query := `
SELECT id, marker_code, object_type::text, object_id
FROM markers
`

	args := make([]any, 0, 2)

	if objectType != "" {
		query += "WHERE object_type = $1\n"
		args = append(args, objectType)
	}

	query += fmt.Sprintf("ORDER BY id LIMIT $%d", len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list markers: %w", err)
	}
	defer rows.Close()

	markers := make([]models.Marker, 0)

	for rows.Next() {
		var marker models.Marker

		if err := rows.Scan(
			&marker.ID,
			&marker.MarkerCode,
			&marker.ObjectType,
			&marker.ObjectID,
		); err != nil {
			return nil, fmt.Errorf("scan marker row: %w", err)
		}

		markers = append(markers, marker)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate marker rows: %w", err)
	}

	return markers, nil
}
