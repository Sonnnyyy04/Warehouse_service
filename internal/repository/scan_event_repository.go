package repository

import (
	"context"
	"database/sql"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ScanEventRepository struct {
	pool *pgxpool.Pool
}

func NewScanEventRepository(pool *pgxpool.Pool) *ScanEventRepository {
	return &ScanEventRepository{pool: pool}
}

func (r *ScanEventRepository) Create(
	ctx context.Context,
	markerCode string,
	userID *int64,
	deviceInfo *string,
	success bool,
) (models.ScanEvent, error) {
	const query = `
INSERT INTO scan_events (marker_code, user_id, device_info, success)
VALUES ($1, $2, $3, $4)
RETURNING id, marker_code, user_id, device_info, success, scanned_at
`

	var event models.ScanEvent
	var dbUserID sql.NullInt64
	var dbDeviceInfo sql.NullString

	err := r.pool.QueryRow(ctx, query, markerCode, userID, deviceInfo, success).Scan(
		&event.ID,
		&event.MarkerCode,
		&dbUserID,
		&dbDeviceInfo,
		&event.Success,
		&event.ScannedAt,
	)
	if err != nil {
		return models.ScanEvent{}, fmt.Errorf("create scan event: %w", err)
	}

	if dbUserID.Valid {
		event.UserID = &dbUserID.Int64
	}
	if dbDeviceInfo.Valid {
		event.DeviceInfo = &dbDeviceInfo.String
	}

	return event, nil
}

func (r *ScanEventRepository) List(ctx context.Context, limit int32) ([]models.ScanEvent, error) {
	const query = `
SELECT id, marker_code, user_id, device_info, success, scanned_at
FROM scan_events
ORDER BY scanned_at DESC, id DESC
LIMIT $1
`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list scan events: %w", err)
	}
	defer rows.Close()

	events := make([]models.ScanEvent, 0)

	for rows.Next() {
		var event models.ScanEvent
		var dbUserID sql.NullInt64
		var dbDeviceInfo sql.NullString

		if err := rows.Scan(
			&event.ID,
			&event.MarkerCode,
			&dbUserID,
			&dbDeviceInfo,
			&event.Success,
			&event.ScannedAt,
		); err != nil {
			return nil, fmt.Errorf("scan scan event row: %w", err)
		}

		if dbUserID.Valid {
			event.UserID = &dbUserID.Int64
		}
		if dbDeviceInfo.Valid {
			event.DeviceInfo = &dbDeviceInfo.String
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scan events: %w", err)
	}

	return events, nil
}
