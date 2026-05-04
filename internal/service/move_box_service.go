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
	ErrInvalidMoveBoxPayload        = errors.New("invalid move box payload")
	ErrInvalidBoxMarkerType         = errors.New("invalid box marker type")
	ErrInvalidStorageCellMarkerType = errors.New("invalid storage cell marker type")
	ErrInvalidBoxTargetMarkerType   = errors.New("invalid box target marker type")
	ErrBoxAlreadyInTargetCell       = errors.New("box already in target cell")
)

type MoveBoxMarkerRepository interface {
	GetByCode(ctx context.Context, markerCode string) (models.Marker, error)
}

type MoveBoxRepository interface {
	GetByID(ctx context.Context, id int64) (models.Box, error)
	MoveToStorageCell(ctx context.Context, boxID, storageCellID int64) error
}

type MoveBoxStorageCellRepository interface {
	GetByID(ctx context.Context, id int64) (models.StorageCell, error)
}

type MoveBoxBatchRepository interface {
	ListProductIDsInBox(ctx context.Context, boxID int64) ([]int64, error)
	HasOtherProductInStorageCell(ctx context.Context, storageCellID int64, productID int64, excludeBatchID *int64, excludeBoxID *int64) (bool, error)
}

type MoveBoxOperationWriter interface {
	Create(ctx context.Context, input CreateOperationInput) (models.OperationHistory, error)
}

type MoveBoxInput struct {
	BoxMarkerCode           string
	TargetMarkerCode        string
	ToStorageCellMarkerCode string
	UserID                  *int64
	Actor                   *models.UserSummary
}

type MoveBoxService struct {
	markerRepo      MoveBoxMarkerRepository
	boxRepo         MoveBoxRepository
	storageCellRepo MoveBoxStorageCellRepository
	operationWriter MoveBoxOperationWriter
	txPool          repository.TxBeginner
}

func NewMoveBoxService(
	markerRepo MoveBoxMarkerRepository,
	boxRepo MoveBoxRepository,
	storageCellRepo MoveBoxStorageCellRepository,
	operationWriter MoveBoxOperationWriter,
	txPool repository.TxBeginner,
) *MoveBoxService {
	return &MoveBoxService{
		markerRepo:      markerRepo,
		boxRepo:         boxRepo,
		storageCellRepo: storageCellRepo,
		operationWriter: operationWriter,
		txPool:          txPool,
	}
}

func (s *MoveBoxService) Execute(ctx context.Context, input MoveBoxInput) (models.MoveBoxResult, error) {
	boxMarkerCode := strings.TrimSpace(input.BoxMarkerCode)
	targetMarkerCode := strings.TrimSpace(input.TargetMarkerCode)
	if targetMarkerCode == "" {
		targetMarkerCode = strings.TrimSpace(input.ToStorageCellMarkerCode)
	}

	if boxMarkerCode == "" || targetMarkerCode == "" {
		return models.MoveBoxResult{}, ErrInvalidMoveBoxPayload
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.MoveBoxResult{}, fmt.Errorf("begin move box tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)
	boxRepo := repository.NewBoxRepositoryWithQuerier(tx)
	batchRepo := repository.NewBatchRepositoryWithQuerier(tx)
	storageCellRepo := repository.NewStorageCellRepositoryWithQuerier(tx)
	operationRepo := repository.NewOperationHistoryRepositoryWithQuerier(tx)
	operationWriter := NewOperationHistoryService(operationRepo)

	boxMarker, err := markerRepo.GetByCode(ctx, boxMarkerCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBoxResult{}, ErrObjectNotFound
		}
		return models.MoveBoxResult{}, err
	}

	if boxMarker.ObjectType != "box" {
		return models.MoveBoxResult{}, ErrInvalidBoxMarkerType
	}

	targetMarker, err := markerRepo.GetByCode(ctx, targetMarkerCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBoxResult{}, ErrObjectNotFound
		}
		return models.MoveBoxResult{}, err
	}

	box, err := boxRepo.GetByID(ctx, boxMarker.ObjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBoxResult{}, ErrObjectNotFound
		}
		return models.MoveBoxResult{}, err
	}

	if targetMarker.ObjectType != "storage_cell" {
		return models.MoveBoxResult{}, ErrInvalidBoxTargetMarkerType
	}

	resultCard, details, err := s.moveBoxToStorageCell(
		ctx,
		boxRepo,
		batchRepo,
		storageCellRepo,
		boxMarker,
		targetMarker,
		box,
	)
	if err != nil {
		return models.MoveBoxResult{}, err
	}

	detailsBytes, err := json.Marshal(details)
	if err != nil {
		return models.MoveBoxResult{}, err
	}

	rawDetails := json.RawMessage(detailsBytes)

	operation, err := operationWriter.Create(ctx, CreateOperationInput{
		ObjectType:    "box",
		ObjectID:      box.ID,
		OperationType: "move_box",
		UserID:        input.UserID,
		Actor:         input.Actor,
		Details:       &rawDetails,
	})
	if err != nil {
		return models.MoveBoxResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.MoveBoxResult{}, fmt.Errorf("commit move box tx: %w", err)
	}

	return models.MoveBoxResult{
		Box:       resultCard,
		Operation: operation,
	}, nil
}

func (s *MoveBoxService) moveBoxToStorageCell(
	ctx context.Context,
	boxRepo *repository.BoxRepository,
	batchRepo *repository.BatchRepository,
	storageCellRepo *repository.StorageCellRepository,
	boxMarker models.Marker,
	targetMarker models.Marker,
	box models.Box,
) (models.ObjectCard, map[string]any, error) {
	targetCell, err := storageCellRepo.GetByID(ctx, targetMarker.ObjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.ObjectCard{}, nil, ErrObjectNotFound
		}
		return models.ObjectCard{}, nil, err
	}

	if box.StorageCellID != nil && *box.StorageCellID == targetCell.ID {
		return models.ObjectCard{}, nil, ErrBoxAlreadyInTargetCell
	}

	if err := ensureStorageCellCanAcceptBox(ctx, boxRepo, batchRepo, targetCell.ID, box.ID); err != nil {
		return models.ObjectCard{}, nil, err
	}

	if err := boxRepo.MoveToStorageCell(ctx, box.ID, targetCell.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.ObjectCard{}, nil, ErrObjectNotFound
		}
		return models.ObjectCard{}, nil, err
	}

	stats, err := boxRepo.GetContentStats(ctx, box.ID)
	if err != nil {
		return models.ObjectCard{}, nil, err
	}

	locationCode := targetCell.Code
	locationType := "storage_cell"
	contentSummary := buildContentSummary(stats)

	card := models.ObjectCard{
		MarkerCode:     boxMarker.MarkerCode,
		ObjectType:     "box",
		ObjectID:       box.ID,
		Code:           box.Code,
		Name:           firstNonEmptyString(stats.ProductName, box.Code),
		Status:         box.Status,
		LocationCode:   &locationCode,
		LocationType:   &locationType,
		ProductSKU:     stats.ProductSKU,
		ProductName:    stats.ProductName,
		Unit:           stats.ProductUnit,
		ContentSummary: &contentSummary,
		BatchesCount:   int32Ptr(stats.BatchesCount),
		ProductsCount:  int32Ptr(stats.ProductsCount),
		TotalQuantity:  int32Ptr(stats.TotalQuantity),
		Quantity:       int32Ptr(stats.TotalQuantity),
	}

	details := map[string]any{
		"action":                     "move_box",
		"box_marker_code":            boxMarker.MarkerCode,
		"target_marker_code":         targetMarker.MarkerCode,
		"target_type":                "storage_cell",
		"target_storage_cell_marker": targetMarker.MarkerCode,
		"from_storage_cell_id":       box.StorageCellID,
		"to_storage_cell_id":         targetCell.ID,
		"to_storage_cell_code":       targetCell.Code,
	}

	return card, details, nil
}
