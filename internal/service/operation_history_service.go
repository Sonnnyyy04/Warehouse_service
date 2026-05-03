package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"Warehouse_service/internal/models"
)

var ErrInvalidOperation = errors.New("invalid operation")
var ErrInvalidOperationHistoryFilter = errors.New("invalid operation history filter")

type OperationHistoryRepository interface {
	Create(
		ctx context.Context,
		objectType string,
		objectID int64,
		operationType string,
		userID *int64,
		actor *models.UserSummary,
		details []byte,
	) (models.OperationHistory, error)

	List(ctx context.Context, filter models.OperationHistoryFilter) ([]models.OperationHistory, error)
}

type CreateOperationInput struct {
	ObjectType    string
	ObjectID      int64
	OperationType string
	UserID        *int64
	Actor         *models.UserSummary
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

	return s.repo.Create(ctx, objectType, input.ObjectID, operationType, input.UserID, input.Actor, details)
}

func (s *OperationHistoryService) List(ctx context.Context, filter models.OperationHistoryFilter) ([]models.OperationHistory, error) {
	normalizedLimit, err := normalizeLimit(filter.Limit)
	if err != nil {
		return nil, err
	}

	filter.Limit = normalizedLimit
	filter.ObjectType = strings.TrimSpace(filter.ObjectType)

	if (filter.ObjectID == nil) != (filter.ObjectType == "") {
		return nil, ErrInvalidOperationHistoryFilter
	}

	if filter.ObjectType != "" && !isValidObjectType(filter.ObjectType) {
		return nil, ErrInvalidOperationHistoryFilter
	}

	return s.repo.List(ctx, filter)
}

func isValidObjectType(objectType string) bool {
	switch objectType {
	case "rack", "storage_cell", "pallet", "box", "product", "batch":
		return true
	default:
		return false
	}
}
