package service

import (
	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"strings"
)

var (
	ErrInvalidMoveBatchPayload      = errors.New("invalid move batch payload")
	ErrInvalidBatchMarkerType       = errors.New("invalid batch marker type")
	ErrInvalidBatchTargetMarkerType = errors.New("invalid batch target marker type")
	ErrBatchAlreadyInTargetBox      = errors.New("batch already in target box")
	ErrBatchAlreadyInTargetCell     = errors.New("batch already in target cell")
)

type MoveBatchMarkerRepository interface {
	GetByCode(ctx context.Context, markerCode string) (models.Marker, error)
}

type MoveBatchRepository interface {
	GetByID(ctx context.Context, id int64) (models.Batch, error)
	HasOtherProductInBox(ctx context.Context, boxID int64, productID int64, excludeBatchID *int64) (bool, error)
	MoveToBox(ctx context.Context, batchID, boxID int64) error
	MoveToStorageCell(ctx context.Context, batchID, storageCellID int64) error
}

type MoveBatchBoxRepository interface {
	GetByID(ctx context.Context, id int64) (models.Box, error)
}

type MoveBatchStorageCellRepository interface {
	GetByID(ctx context.Context, id int64) (models.StorageCell, error)
}

type MoveBatchOperationWriter interface {
	Create(ctx context.Context, input CreateOperationInput) (models.OperationHistory, error)
}

type MoveBatchInput struct {
	BatchMarkerCode  string
	TargetMarkerCode string
	UserID           *int64
}

type MoveBatchService struct {
	markerRepo      MoveBatchMarkerRepository
	batchRepo       MoveBatchRepository
	boxRepo         MoveBatchBoxRepository
	storageCellRepo MoveBatchStorageCellRepository
	operationWriter MoveBatchOperationWriter
}

func NewMoveBatchService(
	markerRepo MoveBatchMarkerRepository,
	batchRepo MoveBatchRepository,
	boxRepo MoveBatchBoxRepository,
	storageCellRepo MoveBatchStorageCellRepository,
	operationWriter MoveBatchOperationWriter,
) *MoveBatchService {
	return &MoveBatchService{
		markerRepo:      markerRepo,
		batchRepo:       batchRepo,
		boxRepo:         boxRepo,
		storageCellRepo: storageCellRepo,
		operationWriter: operationWriter,
	}
}

func (s *MoveBatchService) Execute(ctx context.Context, input MoveBatchInput) (models.MoveBatchResult, error) {
	batchMarkerCode := strings.TrimSpace(input.BatchMarkerCode)
	targetMarkerCode := strings.TrimSpace(input.TargetMarkerCode)

	if batchMarkerCode == "" || targetMarkerCode == "" {
		return models.MoveBatchResult{}, ErrInvalidMoveBatchPayload
	}

	batchMarker, err := s.markerRepo.GetByCode(ctx, batchMarkerCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBatchResult{}, ErrObjectNotFound
		}
		return models.MoveBatchResult{}, err
	}

	if batchMarker.ObjectType != "batch" {
		return models.MoveBatchResult{}, ErrInvalidBatchMarkerType
	}

	targetMarker, err := s.markerRepo.GetByCode(ctx, targetMarkerCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBatchResult{}, ErrObjectNotFound
		}
		return models.MoveBatchResult{}, err
	}

	if targetMarker.ObjectType != "box" && targetMarker.ObjectType != "storage_cell" {
		return models.MoveBatchResult{}, ErrInvalidBatchTargetMarkerType
	}

	batch, err := s.batchRepo.GetByID(ctx, batchMarker.ObjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBatchResult{}, ErrObjectNotFound
		}
		return models.MoveBatchResult{}, err
	}

	var (
		locationCode *string
		parentCode   *string
		details      map[string]any
	)

	switch targetMarker.ObjectType {
	case "box":
		box, err := s.boxRepo.GetByID(ctx, targetMarker.ObjectID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.MoveBatchResult{}, ErrObjectNotFound
			}
			return models.MoveBatchResult{}, err
		}

		if batch.BoxID != nil && *batch.BoxID == box.ID {
			return models.MoveBatchResult{}, ErrBatchAlreadyInTargetBox
		}

		hasMixedProducts, err := s.batchRepo.HasOtherProductInBox(ctx, box.ID, batch.ProductID, &batch.ID)
		if err != nil {
			return models.MoveBatchResult{}, err
		}
		if hasMixedProducts {
			return models.MoveBatchResult{}, ErrMixedBoxProducts
		}

		if err := s.batchRepo.MoveToBox(ctx, batch.ID, box.ID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.MoveBatchResult{}, ErrObjectNotFound
			}
			return models.MoveBatchResult{}, err
		}

		parentCode = &box.Code
		if box.StorageCellID != nil {
			cell, err := s.storageCellRepo.GetByID(ctx, *box.StorageCellID)
			if err == nil {
				locationCode = &cell.Code
			}
		}

		details = map[string]any{
			"action":               "move_batch",
			"batch_marker_code":    batchMarker.MarkerCode,
			"target_marker_code":   targetMarker.MarkerCode,
			"target_type":          "box",
			"from_box_id":          batch.BoxID,
			"from_storage_cell_id": batch.StorageCellID,
			"to_box_id":            box.ID,
			"to_box_code":          box.Code,
		}
		if box.StorageCellID != nil {
			details["to_storage_cell_id"] = box.StorageCellID
		}
		if locationCode != nil {
			details["to_storage_cell_code"] = *locationCode
		}

	case "storage_cell":
		cell, err := s.storageCellRepo.GetByID(ctx, targetMarker.ObjectID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.MoveBatchResult{}, ErrObjectNotFound
			}
			return models.MoveBatchResult{}, err
		}

		if batch.BoxID == nil && batch.StorageCellID != nil && *batch.StorageCellID == cell.ID {
			return models.MoveBatchResult{}, ErrBatchAlreadyInTargetCell
		}

		if err := s.batchRepo.MoveToStorageCell(ctx, batch.ID, cell.ID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.MoveBatchResult{}, ErrObjectNotFound
			}
			return models.MoveBatchResult{}, err
		}

		locationCode = &cell.Code
		details = map[string]any{
			"action":               "move_batch",
			"batch_marker_code":    batchMarker.MarkerCode,
			"target_marker_code":   targetMarker.MarkerCode,
			"target_type":          "storage_cell",
			"from_box_id":          batch.BoxID,
			"from_storage_cell_id": batch.StorageCellID,
			"to_storage_cell_id":   cell.ID,
			"to_storage_cell_code": cell.Code,
		}
	}

	detailsBytes, err := json.Marshal(details)
	if err != nil {
		return models.MoveBatchResult{}, err
	}

	rawDetails := json.RawMessage(detailsBytes)

	operation, err := s.operationWriter.Create(ctx, CreateOperationInput{
		ObjectType:    "batch",
		ObjectID:      batch.ID,
		OperationType: "move_batch",
		UserID:        input.UserID,
		Details:       &rawDetails,
	})
	if err != nil {
		return models.MoveBatchResult{}, err
	}

	quantity := batch.Quantity

	return models.MoveBatchResult{
		Batch: models.ObjectCard{
			MarkerCode:   batchMarker.MarkerCode,
			ObjectType:   "batch",
			ObjectID:     batch.ID,
			Code:         batch.Code,
			Name:         batch.Code,
			Status:       batch.Status,
			LocationCode: locationCode,
			ParentCode:   parentCode,
			Quantity:     &quantity,
		},
		Operation: operation,
	}, nil
}
