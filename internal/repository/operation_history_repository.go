package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OperationHistoryRepository struct {
	pool *pgxpool.Pool
}

func NewOperationHistoryRepository(pool *pgxpool.Pool) *OperationHistoryRepository {
	return &OperationHistoryRepository{pool: pool}
}

func (r *OperationHistoryRepository) Create(
	ctx context.Context,
	objectType string,
	objectID int64,
	operationType string,
	userID *int64,
	details []byte,
) (models.OperationHistory, error) {
	const query = `
INSERT INTO operation_history (object_type, object_id, operation_type, user_id, details)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, object_type::text, object_id, operation_type, user_id, details, created_at
`

	var operation models.OperationHistory
	var dbUserID sql.NullInt64
	var dbDetails []byte

	err := r.pool.QueryRow(ctx, query, objectType, objectID, operationType, userID, details).Scan(
		&operation.ID,
		&operation.ObjectType,
		&operation.ObjectID,
		&operation.OperationType,
		&dbUserID,
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
    oh.details,
    oh.created_at,
    u.id,
    u.login,
    u.full_name,
    u.role
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

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list operation history: %w", err)
	}
	defer rows.Close()

	operations := make([]models.OperationHistory, 0)

	for rows.Next() {
		var operation models.OperationHistory
		var dbUserID sql.NullInt64
		var dbDetails []byte
		var actorID sql.NullInt64
		var actorLogin sql.NullString
		var actorFullName sql.NullString
		var actorRole sql.NullString

		if err := rows.Scan(
			&operation.ID,
			&operation.ObjectType,
			&operation.ObjectID,
			&operation.OperationType,
			&dbUserID,
			&dbDetails,
			&operation.CreatedAt,
			&actorID,
			&actorLogin,
			&actorFullName,
			&actorRole,
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
		if actorID.Valid {
			operation.Actor = &models.UserSummary{
				ID:       actorID.Int64,
				Login:    actorLogin.String,
				FullName: actorFullName.String,
				Role:     actorRole.String,
			}
		}

		operations = append(operations, operation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate operation history: %w", err)
	}

	return operations, nil
}
