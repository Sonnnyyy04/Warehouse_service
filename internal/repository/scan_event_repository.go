package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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

func (r *ScanEventRepository) List(
	ctx context.Context,
	filter models.ScanEventFilter,
) ([]models.ScanEvent, error) {
	query := `
SELECT
    se.id,
    se.marker_code,
    se.user_id,
    se.device_info,
    se.success,
    se.scanned_at,
    u.id,
    u.login,
    u.full_name,
    u.role
FROM scan_events se
LEFT JOIN users u ON u.id = se.user_id
`

	args := make([]any, 0, 3)
	conditions := make([]string, 0, 2)

	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		conditions = append(conditions, fmt.Sprintf("se.user_id = $%d", len(args)))
	}

	if filter.MarkerCode != "" {
		args = append(args, filter.MarkerCode)
		conditions = append(conditions, fmt.Sprintf("se.marker_code = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.Limit)
	query += fmt.Sprintf("ORDER BY se.scanned_at DESC, se.id DESC\nLIMIT $%d\n", len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list scan events: %w", err)
	}
	defer rows.Close()

	events := make([]models.ScanEvent, 0)

	for rows.Next() {
		var event models.ScanEvent
		var dbUserID sql.NullInt64
		var dbDeviceInfo sql.NullString
		var actorID sql.NullInt64
		var actorLogin sql.NullString
		var actorFullName sql.NullString
		var actorRole sql.NullString

		if err := rows.Scan(
			&event.ID,
			&event.MarkerCode,
			&dbUserID,
			&dbDeviceInfo,
			&event.Success,
			&event.ScannedAt,
			&actorID,
			&actorLogin,
			&actorFullName,
			&actorRole,
		); err != nil {
			return nil, fmt.Errorf("scan scan event row: %w", err)
		}

		if dbUserID.Valid {
			event.UserID = &dbUserID.Int64
		}
		if dbDeviceInfo.Valid {
			event.DeviceInfo = &dbDeviceInfo.String
		}
		if actorID.Valid {
			event.Actor = &models.UserSummary{
				ID:       actorID.Int64,
				Login:    actorLogin.String,
				FullName: actorFullName.String,
				Role:     actorRole.String,
			}
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scan events: %w", err)
	}

	return events, nil
}
