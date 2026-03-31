package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"Warehouse_service/internal/models"
)

var ErrInvalidOperation = errors.New("invalid operation")

type OperationHistoryRepository interface {
	Create(
		ctx context.Context,
		objectType string,
		objectID int64,
		operationType string,
		userID *int64,
		details []byte,
	) (models.OperationHistory, error)

	List(ctx context.Context, limit int32) ([]models.OperationHistory, error)
}

type CreateOperationInput struct {
	ObjectType    string
	ObjectID      int64
	OperationType string
	UserID        *int64
	Details       *json.RawMessage
}

type OperationHistoryService struct {
	repo OperationHistoryRepository
}

func NewOperationHistoryService(repo OperationHistoryRepository) *OperationHistoryService {
	return &OperationHistoryService{repo: repo}
}

func (s *OperationHistoryService) Create(ctx context.Context, input CreateOperationInput) (models.OperationHistory, error) {
	objectType := strings.TrimSpace(input.ObjectType)
	operationType := strings.TrimSpace(input.OperationType)

	if !isValidObjectType(objectType) {
		return models.OperationHistory{}, ErrInvalidOperation
	}
	if input.ObjectID <= 0 {
		return models.OperationHistory{}, ErrInvalidOperation
	}
	if operationType == "" {
		return models.OperationHistory{}, ErrInvalidOperation
	}

	var details []byte
	if input.Details != nil {
		details = *input.Details
	}

	return s.repo.Create(ctx, objectType, input.ObjectID, operationType, input.UserID, details)
}

func (s *OperationHistoryService) List(ctx context.Context, limit int32) ([]models.OperationHistory, error) {
	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	return s.repo.List(ctx, normalizedLimit)
}

func isValidObjectType(objectType string) bool {
	switch objectType {
	case "storage_cell", "pallet", "box", "product", "batch":
		return true
	default:
		return false
	}
}
