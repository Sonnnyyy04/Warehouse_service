package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"Warehouse_service/internal/models"
)

type OperationHistoryRepository struct {
	db Querier
}

func NewOperationHistoryRepository(pool Querier) *OperationHistoryRepository {
	return NewOperationHistoryRepositoryWithQuerier(pool)
}

func NewOperationHistoryRepositoryWithQuerier(db Querier) *OperationHistoryRepository {
	return &OperationHistoryRepository{db: db}
}

func (r *OperationHistoryRepository) Create(
	ctx context.Context,
	objectType string,
	objectID int64,
	operationType string,
	userID *int64,
	actor *models.UserSummary,
	details []byte,
) (models.OperationHistory, error) {
	const query = `
INSERT INTO operation_history (
    object_type,
    object_id,
    operation_type,
    user_id,
    actor_user_id,
    actor_login,
    actor_full_name,
    actor_role,
    actor_is_super_admin,
    details
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, object_type::text, object_id, operation_type, user_id, actor_user_id, actor_login, actor_full_name, actor_role, actor_is_super_admin, details, created_at
`

	var operation models.OperationHistory
	var dbUserID sql.NullInt64
	var actorID sql.NullInt64
	var actorLogin sql.NullString
	var actorFullName sql.NullString
	var actorRole sql.NullString
	var actorIsSuperAdmin bool
	var dbDetails []byte

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

	err := r.db.QueryRow(ctx, query, objectType, objectID, operationType, userID, actorUserID, actorLoginValue, actorFullNameValue, actorRoleValue, actorIsSuperAdminValue, details).Scan(
		&operation.ID,
		&operation.ObjectType,
		&operation.ObjectID,
		&operation.OperationType,
		&dbUserID,
		&actorID,
		&actorLogin,
		&actorFullName,
		&actorRole,
		&actorIsSuperAdmin,
		&dbDetails,
		&operation.CreatedAt,
	)
	if err != nil {
		return models.OperationHistory{}, fmt.Errorf("create operation history: %w", err)
	}

	if dbUserID.Valid {
		operation.UserID = &dbUserID.Int64
	}
	if dbDetails != nil {
		raw := json.RawMessage(dbDetails)
		operation.Details = &raw
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
		operation.Actor = &summary
	}

	return operation, nil
}

func (r *OperationHistoryRepository) List(
	ctx context.Context,
	filter models.OperationHistoryFilter,
) ([]models.OperationHistory, error) {
	query := `
SELECT
    oh.id,
    oh.object_type::text,
    oh.object_id,
    oh.operation_type,
    oh.user_id,
    oh.actor_user_id,
    oh.actor_login,
    oh.actor_full_name,
    oh.actor_role,
    oh.actor_is_super_admin,
    oh.details,
    oh.created_at,
    u.id,
    u.login,
    u.full_name,
    u.role,
    COALESCE(u.is_super_admin, FALSE)
FROM operation_history oh
LEFT JOIN users u ON u.id = oh.user_id
`

	args := make([]any, 0, 4)
	conditions := make([]string, 0, 3)

	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		conditions = append(conditions, fmt.Sprintf("oh.user_id = $%d", len(args)))
	}

	if filter.ObjectID != nil {
		args = append(args, filter.ObjectType)
		conditions = append(conditions, fmt.Sprintf("oh.object_type = $%d", len(args)))
		args = append(args, *filter.ObjectID)
		conditions = append(conditions, fmt.Sprintf("oh.object_id = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}

	args = append(args, filter.Limit)
	query += fmt.Sprintf("ORDER BY oh.created_at DESC, oh.id DESC\nLIMIT $%d\n", len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list operation history: %w", err)
	}
	defer rows.Close()

	operations := make([]models.OperationHistory, 0)

	for rows.Next() {
		var operation models.OperationHistory
		var dbUserID sql.NullInt64
		var snapshotActorID sql.NullInt64
		var snapshotActorLogin sql.NullString
		var snapshotActorFullName sql.NullString
		var snapshotActorRole sql.NullString
		var snapshotActorIsSuperAdmin bool
		var dbDetails []byte
		var actorID sql.NullInt64
		var actorLogin sql.NullString
		var actorFullName sql.NullString
		var actorRole sql.NullString
		var actorIsSuperAdmin bool

		if err := rows.Scan(
			&operation.ID,
			&operation.ObjectType,
			&operation.ObjectID,
			&operation.OperationType,
			&dbUserID,
			&snapshotActorID,
			&snapshotActorLogin,
			&snapshotActorFullName,
			&snapshotActorRole,
			&snapshotActorIsSuperAdmin,
			&dbDetails,
			&operation.CreatedAt,
			&actorID,
			&actorLogin,
			&actorFullName,
			&actorRole,
			&actorIsSuperAdmin,
		); err != nil {
			return nil, fmt.Errorf("scan operation history row: %w", err)
		}

		if dbUserID.Valid {
			operation.UserID = &dbUserID.Int64
		}
		if dbDetails != nil {
			raw := json.RawMessage(dbDetails)
			operation.Details = &raw
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
			operation.Actor = &summary
		}

		operations = append(operations, operation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate operation history: %w", err)
	}

	return operations, nil
}
