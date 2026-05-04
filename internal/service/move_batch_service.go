package service

import (
	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	ListProductIDsInBox(ctx context.Context, boxID int64) ([]int64, error)
	HasOtherProductInStorageCell(ctx context.Context, storageCellID int64, productID int64, excludeBatchID *int64, excludeBoxID *int64) (bool, error)
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
	Actor            *models.UserSummary
}

type MoveBatchService struct {
	markerRepo      MoveBatchMarkerRepository
	batchRepo       MoveBatchRepository
	boxRepo         MoveBatchBoxRepository
	storageCellRepo MoveBatchStorageCellRepository
	operationWriter MoveBatchOperationWriter
	txPool          repository.TxBeginner
}

func NewMoveBatchService(
	markerRepo MoveBatchMarkerRepository,
	batchRepo MoveBatchRepository,
	boxRepo MoveBatchBoxRepository,
	storageCellRepo MoveBatchStorageCellRepository,
	operationWriter MoveBatchOperationWriter,
	txPool repository.TxBeginner,
) *MoveBatchService {
	return &MoveBatchService{
		markerRepo:      markerRepo,
		batchRepo:       batchRepo,
		boxRepo:         boxRepo,
		storageCellRepo: storageCellRepo,
		operationWriter: operationWriter,
		txPool:          txPool,
	}
}

func (s *MoveBatchService) Execute(ctx context.Context, input MoveBatchInput) (models.MoveBatchResult, error) {
	batchMarkerCode := strings.TrimSpace(input.BatchMarkerCode)
	targetMarkerCode := strings.TrimSpace(input.TargetMarkerCode)

	if batchMarkerCode == "" || targetMarkerCode == "" {
		return models.MoveBatchResult{}, ErrInvalidMoveBatchPayload
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.MoveBatchResult{}, fmt.Errorf("begin move batch tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)
	batchRepo := repository.NewBatchRepositoryWithQuerier(tx)
	boxRepo := repository.NewBoxRepositoryWithQuerier(tx)
	storageCellRepo := repository.NewStorageCellRepositoryWithQuerier(tx)
	operationRepo := repository.NewOperationHistoryRepositoryWithQuerier(tx)
	operationWriter := NewOperationHistoryService(operationRepo)

	batchMarker, err := markerRepo.GetByCode(ctx, batchMarkerCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBatchResult{}, ErrObjectNotFound
		}
		return models.MoveBatchResult{}, err
	}

	if batchMarker.ObjectType != "batch" {
		return models.MoveBatchResult{}, ErrInvalidBatchMarkerType
	}

	targetMarker, err := markerRepo.GetByCode(ctx, targetMarkerCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBatchResult{}, ErrObjectNotFound
		}
		return models.MoveBatchResult{}, err
	}

	if targetMarker.ObjectType != "box" {
		return models.MoveBatchResult{}, ErrInvalidBatchTargetMarkerType
	}

	batch, err := batchRepo.GetByID(ctx, batchMarker.ObjectID)
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
		box, err := boxRepo.GetByID(ctx, targetMarker.ObjectID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.MoveBatchResult{}, ErrObjectNotFound
			}
			return models.MoveBatchResult{}, err
		}

		if batch.BoxID != nil && *batch.BoxID == box.ID {
			return models.MoveBatchResult{}, ErrBatchAlreadyInTargetBox
		}

		hasMixedProducts, err := batchRepo.HasOtherProductInBox(ctx, box.ID, batch.ProductID, &batch.ID)
		if err != nil {
			return models.MoveBatchResult{}, err
		}
		if hasMixedProducts {
			return models.MoveBatchResult{}, ErrMixedBoxProducts
		}

		if box.StorageCellID != nil {
			if err := ensureStorageCellCanAcceptProduct(ctx, batchRepo, *box.StorageCellID, batch.ProductID, &batch.ID, &box.ID); err != nil {
				return models.MoveBatchResult{}, err
			}
		}

		if err := batchRepo.MoveToBox(ctx, batch.ID, box.ID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.MoveBatchResult{}, ErrObjectNotFound
			}
			return models.MoveBatchResult{}, err
		}

		parentCode = &box.Code
		if box.StorageCellID != nil {
			cell, err := storageCellRepo.GetByID(ctx, *box.StorageCellID)
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
		cell, err := storageCellRepo.GetByID(ctx, targetMarker.ObjectID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.MoveBatchResult{}, ErrObjectNotFound
			}
			return models.MoveBatchResult{}, err
		}

		if batch.BoxID == nil && batch.StorageCellID != nil && *batch.StorageCellID == cell.ID {
			return models.MoveBatchResult{}, ErrBatchAlreadyInTargetCell
		}

		if err := ensureStorageCellCanAcceptProduct(ctx, batchRepo, cell.ID, batch.ProductID, &batch.ID, nil); err != nil {
			return models.MoveBatchResult{}, err
		}

		if err := batchRepo.MoveToStorageCell(ctx, batch.ID, cell.ID); err != nil {
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

	operation, err := operationWriter.Create(ctx, CreateOperationInput{
		ObjectType:    "batch",
		ObjectID:      batch.ID,
		OperationType: "move_batch",
		UserID:        input.UserID,
		Actor:         input.Actor,
		Details:       &rawDetails,
	})
	if err != nil {
		return models.MoveBatchResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.MoveBatchResult{}, fmt.Errorf("commit move batch tx: %w", err)
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
