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
	ErrInvalidMoveBoxPayload        = errors.New("invalid move box payload")
	ErrInvalidBoxMarkerType         = errors.New("invalid box marker type")
	ErrInvalidStorageCellMarkerType = errors.New("invalid storage cell marker type")
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

type MoveBoxOperationWriter interface {
	Create(ctx context.Context, input CreateOperationInput) (models.OperationHistory, error)
}

type MoveBoxInput struct {
	BoxMarkerCode           string
	ToStorageCellMarkerCode string
	UserID                  *int64
}

type MoveBoxService struct {
	markerRepo      MoveBoxMarkerRepository
	boxRepo         MoveBoxRepository
	storageCellRepo MoveBoxStorageCellRepository
	operationWriter MoveBoxOperationWriter
}

func NewMoveBoxService(
	markerRepo MoveBoxMarkerRepository,
	boxRepo MoveBoxRepository,
	storageCellRepo MoveBoxStorageCellRepository,
	operationWriter MoveBoxOperationWriter,
) *MoveBoxService {
	return &MoveBoxService{
		markerRepo:      markerRepo,
		boxRepo:         boxRepo,
		storageCellRepo: storageCellRepo,
		operationWriter: operationWriter,
	}
}

func (s *MoveBoxService) Execute(ctx context.Context, input MoveBoxInput) (models.MoveBoxResult, error) {
	boxMarkerCode := strings.TrimSpace(input.BoxMarkerCode)
	targetCellMarkerCode := strings.TrimSpace(input.ToStorageCellMarkerCode)

	if boxMarkerCode == "" || targetCellMarkerCode == "" {
		return models.MoveBoxResult{}, ErrInvalidMoveBoxPayload
	}

	boxMarker, err := s.markerRepo.GetByCode(ctx, boxMarkerCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBoxResult{}, ErrObjectNotFound
		}
		return models.MoveBoxResult{}, err
	}

	if boxMarker.ObjectType != "box" {
		return models.MoveBoxResult{}, ErrInvalidBoxMarkerType
	}

	targetCellMarker, err := s.markerRepo.GetByCode(ctx, targetCellMarkerCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBoxResult{}, ErrObjectNotFound
		}
		return models.MoveBoxResult{}, err
	}

	if targetCellMarker.ObjectType != "storage_cell" {
		return models.MoveBoxResult{}, ErrInvalidStorageCellMarkerType
	}

	box, err := s.boxRepo.GetByID(ctx, boxMarker.ObjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBoxResult{}, ErrObjectNotFound
		}
		return models.MoveBoxResult{}, err
	}

	targetCell, err := s.storageCellRepo.GetByID(ctx, targetCellMarker.ObjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBoxResult{}, ErrObjectNotFound
		}
		return models.MoveBoxResult{}, err
	}

	if box.PalletID == nil && box.StorageCellID != nil && *box.StorageCellID == targetCell.ID {
		return models.MoveBoxResult{}, ErrBoxAlreadyInTargetCell
	}

	if err := s.boxRepo.MoveToStorageCell(ctx, box.ID, targetCell.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.MoveBoxResult{}, ErrObjectNotFound
		}
		return models.MoveBoxResult{}, err
	}

	detailsBytes, err := json.Marshal(map[string]any{
		"action":                     "move_box",
		"box_marker_code":            boxMarker.MarkerCode,
		"target_storage_cell_marker": targetCellMarker.MarkerCode,
		"from_pallet_id":             box.PalletID,
		"from_storage_cell_id":       box.StorageCellID,
		"to_storage_cell_id":         targetCell.ID,
		"to_storage_cell_code":       targetCell.Code,
	})
	if err != nil {
		return models.MoveBoxResult{}, err
	}

	rawDetails := json.RawMessage(detailsBytes)

	operation, err := s.operationWriter.Create(ctx, CreateOperationInput{
		ObjectType:    "box",
		ObjectID:      box.ID,
		OperationType: "move_box",
		UserID:        input.UserID,
		Details:       &rawDetails,
	})
	if err != nil {
		return models.MoveBoxResult{}, err
	}

	locationCode := targetCell.Code

	return models.MoveBoxResult{
		Box: models.ObjectCard{
			MarkerCode:   boxMarker.MarkerCode,
			ObjectType:   "box",
			ObjectID:     box.ID,
			Code:         box.Code,
			Name:         box.Code,
			Status:       box.Status,
			LocationCode: &locationCode,
			ParentCode:   nil,
		},
		Operation: operation,
	}, nil
}
