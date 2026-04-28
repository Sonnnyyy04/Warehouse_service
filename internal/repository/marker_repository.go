package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type MarkerRepository struct {
	db Querier
}

func NewMarkerRepository(pool *pgxpool.Pool) *MarkerRepository {
	return NewMarkerRepositoryWithQuerier(pool)
}

func NewMarkerRepositoryWithQuerier(db Querier) *MarkerRepository {
	return &MarkerRepository{db: db}
}

func (r *MarkerRepository) GetByCode(ctx context.Context, markerCode string) (models.Marker, error) {
	const query = `
SELECT id, marker_code, object_type::text, object_id
FROM markers
WHERE marker_code = $1
`

	var marker models.Marker

	err := r.db.QueryRow(ctx, query, markerCode).Scan(
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

	rows, err := r.db.Query(ctx, query, args...)
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

func (r *MarkerRepository) ListByCodes(ctx context.Context, objectType string, markerCodes []string) ([]models.Marker, error) {
	query := `
SELECT id, marker_code, object_type::text, object_id
FROM markers
WHERE marker_code = ANY($1)
`

	args := []any{markerCodes}

	if objectType != "" {
		query += "AND object_type = $2\n"
		args = append(args, objectType)
	}

	query += "ORDER BY id"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list markers by codes: %w", err)
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
			return nil, fmt.Errorf("scan marker by code row: %w", err)
		}

		markers = append(markers, marker)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate marker by code rows: %w", err)
	}

	return markers, nil
}

func (r *MarkerRepository) Create(ctx context.Context, markerCode, objectType string, objectID int64) (models.Marker, error) {
	const query = `
INSERT INTO markers (marker_code, object_type, object_id)
VALUES ($1, $2::object_type, $3)
RETURNING id, marker_code, object_type::text, object_id
`

	var marker models.Marker

	if err := r.db.QueryRow(
		ctx,
		query,
		strings.TrimSpace(markerCode),
		strings.TrimSpace(objectType),
		objectID,
	).Scan(
		&marker.ID,
		&marker.MarkerCode,
		&marker.ObjectType,
		&marker.ObjectID,
	); err != nil {
		return models.Marker{}, fmt.Errorf("create marker: %w", err)
	}

	return marker, nil
}

func (r *MarkerRepository) DeleteByObject(ctx context.Context, objectType string, objectID int64) error {
	const query = `
DELETE FROM markers
WHERE object_type = $1::object_type
  AND object_id = $2
`

	if _, err := r.db.Exec(ctx, query, strings.TrimSpace(objectType), objectID); err != nil {
		return fmt.Errorf("delete marker by object: %w", err)
	}

	return nil
}
