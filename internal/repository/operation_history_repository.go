package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

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

func (r *OperationHistoryRepository) List(ctx context.Context, limit int32) ([]models.OperationHistory, error) {
	const query = `
SELECT id, object_type::text, object_id, operation_type, user_id, details, created_at
FROM operation_history
ORDER BY created_at DESC, id DESC
LIMIT $1
`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list operation history: %w", err)
	}
	defer rows.Close()

	operations := make([]models.OperationHistory, 0)

	for rows.Next() {
		var operation models.OperationHistory
		var dbUserID sql.NullInt64
		var dbDetails []byte

		if err := rows.Scan(
			&operation.ID,
			&operation.ObjectType,
			&operation.ObjectID,
			&operation.OperationType,
			&dbUserID,
			&dbDetails,
			&operation.CreatedAt,
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

		operations = append(operations, operation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate operation history: %w", err)
	}

	return operations, nil
}
