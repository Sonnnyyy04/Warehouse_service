package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"Warehouse_service/internal/models"
)

type ScanEventRepository struct {
	db Querier
}

func NewScanEventRepository(pool Querier) *ScanEventRepository {
	return NewScanEventRepositoryWithQuerier(pool)
}

func NewScanEventRepositoryWithQuerier(db Querier) *ScanEventRepository {
	return &ScanEventRepository{db: db}
}

func (r *ScanEventRepository) Create(
	ctx context.Context,
	markerCode string,
	userID *int64,
	actor *models.UserSummary,
	deviceInfo *string,
	success bool,
) (models.ScanEvent, error) {
	const query = `
INSERT INTO scan_events (
    marker_code,
    user_id,
    actor_user_id,
    actor_login,
    actor_full_name,
    actor_role,
    actor_is_super_admin,
    device_info,
    success
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, marker_code, user_id, actor_user_id, actor_login, actor_full_name, actor_role, actor_is_super_admin, device_info, success, scanned_at
`

	var event models.ScanEvent
	var dbUserID sql.NullInt64
	var actorID sql.NullInt64
	var actorLogin sql.NullString
	var actorFullName sql.NullString
	var actorRole sql.NullString
	var actorIsSuperAdmin bool
	var dbDeviceInfo sql.NullString

	var actorUserID any
	var actorLoginValue any
	var actorFullNameValue any
	var actorRoleValue any
	actorIsSuperAdminValue := false
	if actor != nil {
		actorUserID = actor.ID
		actorLoginValue = actor.Login
		actorFullNameValue = actor.FullName
		actorRoleValue = actor.Role
		actorIsSuperAdminValue = actor.IsSuperAdmin
	}

	err := r.db.QueryRow(ctx, query, markerCode, userID, actorUserID, actorLoginValue, actorFullNameValue, actorRoleValue, actorIsSuperAdminValue, deviceInfo, success).Scan(
		&event.ID,
		&event.MarkerCode,
		&dbUserID,
		&actorID,
		&actorLogin,
		&actorFullName,
		&actorRole,
		&actorIsSuperAdmin,
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
	if actorLogin.Valid || actorFullName.Valid || actorRole.Valid || actorID.Valid {
		summary := models.UserSummary{
			Login:        actorLogin.String,
			FullName:     actorFullName.String,
			Role:         actorRole.String,
			IsSuperAdmin: actorIsSuperAdmin,
		}
		if actorID.Valid {
			summary.ID = actorID.Int64
		}
		event.Actor = &summary
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
    se.actor_user_id,
    se.actor_login,
    se.actor_full_name,
    se.actor_role,
    se.actor_is_super_admin,
    se.device_info,
    se.success,
    se.scanned_at,
    u.id,
    u.login,
    u.full_name,
    u.role,
    COALESCE(u.is_super_admin, FALSE)
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

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list scan events: %w", err)
	}
	defer rows.Close()

	events := make([]models.ScanEvent, 0)

	for rows.Next() {
		var event models.ScanEvent
		var dbUserID sql.NullInt64
		var snapshotActorID sql.NullInt64
		var snapshotActorLogin sql.NullString
		var snapshotActorFullName sql.NullString
		var snapshotActorRole sql.NullString
		var snapshotActorIsSuperAdmin bool
		var dbDeviceInfo sql.NullString
		var actorID sql.NullInt64
		var actorLogin sql.NullString
		var actorFullName sql.NullString
		var actorRole sql.NullString
		var actorIsSuperAdmin bool

		if err := rows.Scan(
			&event.ID,
			&event.MarkerCode,
			&dbUserID,
			&snapshotActorID,
			&snapshotActorLogin,
			&snapshotActorFullName,
			&snapshotActorRole,
			&snapshotActorIsSuperAdmin,
			&dbDeviceInfo,
			&event.Success,
			&event.ScannedAt,
			&actorID,
			&actorLogin,
			&actorFullName,
			&actorRole,
			&actorIsSuperAdmin,
		); err != nil {
			return nil, fmt.Errorf("scan scan event row: %w", err)
		}

		if dbUserID.Valid {
			event.UserID = &dbUserID.Int64
		}
		if dbDeviceInfo.Valid {
			event.DeviceInfo = &dbDeviceInfo.String
		}
		if actorID.Valid || snapshotActorID.Valid || actorLogin.Valid || snapshotActorLogin.Valid || actorFullName.Valid || snapshotActorFullName.Valid || actorRole.Valid || snapshotActorRole.Valid {
			summary := models.UserSummary{
				IsSuperAdmin: snapshotActorIsSuperAdmin,
			}
			if actorID.Valid {
				summary.ID = actorID.Int64
			} else if snapshotActorID.Valid {
				summary.ID = snapshotActorID.Int64
			}
			if actorLogin.Valid {
				summary.Login = actorLogin.String
			} else {
				summary.Login = snapshotActorLogin.String
			}
			if actorFullName.Valid {
				summary.FullName = actorFullName.String
			} else {
				summary.FullName = snapshotActorFullName.String
			}
			if actorRole.Valid {
				summary.Role = actorRole.String
			} else {
				summary.Role = snapshotActorRole.String
			}
			if actorID.Valid {
				summary.IsSuperAdmin = actorIsSuperAdmin
			}
			event.Actor = &summary
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scan events: %w", err)
	}

	return events, nil
}
