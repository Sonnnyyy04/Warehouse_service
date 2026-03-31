package service

import (
	"context"
	"errors"
	"strings"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"
)

var ErrInvalidMarkerCode = errors.New("marker_code is required")
var ErrObjectNotFound = errors.New("object not found")

type MarkerRepository interface {
	GetByCode(ctx context.Context, markerCode string) (models.Marker, error)
}

type StorageCellRepository interface {
	GetByID(ctx context.Context, id int64) (models.StorageCell, error)
}

type PalletRepository interface {
	GetByID(ctx context.Context, id int64) (models.Pallet, error)
}

type BoxRepository interface {
	GetByID(ctx context.Context, id int64) (models.Box, error)
}

type ProductRepository interface {
	GetByID(ctx context.Context, id int64) (models.Product, error)
}

type BatchRepository interface {
	GetByID(ctx context.Context, id int64) (models.Batch, error)
}

type ObjectService struct {
	markerRepo      MarkerRepository
	storageCellRepo StorageCellRepository
	palletRepo      PalletRepository
	boxRepo         BoxRepository
	productRepo     ProductRepository
	batchRepo       BatchRepository
}

func NewObjectService(
	markerRepo MarkerRepository,
	storageCellRepo StorageCellRepository,
	palletRepo PalletRepository,
	boxRepo BoxRepository,
	productRepo ProductRepository,
	batchRepo BatchRepository,
) *ObjectService {
	return &ObjectService{
		markerRepo:      markerRepo,
		storageCellRepo: storageCellRepo,
		palletRepo:      palletRepo,
		boxRepo:         boxRepo,
		productRepo:     productRepo,
		batchRepo:       batchRepo,
	}
}

func (s *ObjectService) GetByMarkerCode(ctx context.Context, markerCode string) (models.ObjectCard, error) {
	markerCode = strings.TrimSpace(markerCode)
	if markerCode == "" {
		return models.ObjectCard{}, ErrInvalidMarkerCode
	}

	marker, err := s.markerRepo.GetByCode(ctx, markerCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.ObjectCard{}, ErrObjectNotFound
		}
		return models.ObjectCard{}, err
	}

	switch marker.ObjectType {
	case "storage_cell":
		return s.buildStorageCellCard(ctx, marker)
	case "pallet":
		return s.buildPalletCard(ctx, marker)
	case "box":
		return s.buildBoxCard(ctx, marker)
	case "product":
		return s.buildProductCard(ctx, marker)
	case "batch":
		return s.buildBatchCard(ctx, marker)
	default:
		return models.ObjectCard{}, ErrObjectNotFound
	}
}

func (s *ObjectService) buildStorageCellCard(ctx context.Context, marker models.Marker) (models.ObjectCard, error) {
	cell, err := s.storageCellRepo.GetByID(ctx, marker.ObjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.ObjectCard{}, ErrObjectNotFound
		}
		return models.ObjectCard{}, err
	}

	locationCode := cell.Code

	return models.ObjectCard{
		MarkerCode:   marker.MarkerCode,
		ObjectType:   marker.ObjectType,
		ObjectID:     cell.ID,
		Code:         cell.Code,
		Name:         cell.Name,
		Status:       cell.Status,
		LocationCode: &locationCode,
	}, nil
}

func (s *ObjectService) buildPalletCard(ctx context.Context, marker models.Marker) (models.ObjectCard, error) {
	pallet, err := s.palletRepo.GetByID(ctx, marker.ObjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.ObjectCard{}, ErrObjectNotFound
		}
		return models.ObjectCard{}, err
	}

	var locationCode *string
	if pallet.StorageCellID != nil {
		cell, err := s.storageCellRepo.GetByID(ctx, *pallet.StorageCellID)
		if err == nil {
			locationCode = &cell.Code
		}
	}

	return models.ObjectCard{
		MarkerCode:   marker.MarkerCode,
		ObjectType:   marker.ObjectType,
		ObjectID:     pallet.ID,
		Code:         pallet.Code,
		Name:         pallet.Code,
		Status:       pallet.Status,
		LocationCode: locationCode,
	}, nil
}

func (s *ObjectService) buildBoxCard(ctx context.Context, marker models.Marker) (models.ObjectCard, error) {
	box, err := s.boxRepo.GetByID(ctx, marker.ObjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.ObjectCard{}, ErrObjectNotFound
		}
		return models.ObjectCard{}, err
	}

	var locationCode *string
	var parentCode *string

	if box.StorageCellID != nil {
		cell, err := s.storageCellRepo.GetByID(ctx, *box.StorageCellID)
		if err == nil {
			locationCode = &cell.Code
		}
	}

	if box.PalletID != nil {
		pallet, err := s.palletRepo.GetByID(ctx, *box.PalletID)
		if err == nil {
			parentCode = &pallet.Code
			if locationCode == nil && pallet.StorageCellID != nil {
				cell, err := s.storageCellRepo.GetByID(ctx, *pallet.StorageCellID)
				if err == nil {
					locationCode = &cell.Code
				}
			}
		}
	}

	return models.ObjectCard{
		MarkerCode:   marker.MarkerCode,
		ObjectType:   marker.ObjectType,
		ObjectID:     box.ID,
		Code:         box.Code,
		Name:         box.Code,
		Status:       box.Status,
		LocationCode: locationCode,
		ParentCode:   parentCode,
	}, nil
}

func (s *ObjectService) buildProductCard(ctx context.Context, marker models.Marker) (models.ObjectCard, error) {
	product, err := s.productRepo.GetByID(ctx, marker.ObjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.ObjectCard{}, ErrObjectNotFound
		}
		return models.ObjectCard{}, err
	}

	return models.ObjectCard{
		MarkerCode: marker.MarkerCode,
		ObjectType: marker.ObjectType,
		ObjectID:   product.ID,
		Code:       product.SKU,
		Name:       product.Name,
		Status:     "active",
	}, nil
}

func (s *ObjectService) buildBatchCard(ctx context.Context, marker models.Marker) (models.ObjectCard, error) {
	batch, err := s.batchRepo.GetByID(ctx, marker.ObjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.ObjectCard{}, ErrObjectNotFound
		}
		return models.ObjectCard{}, err
	}

	var locationCode *string
	var parentCode *string

	if batch.StorageCellID != nil {
		cell, err := s.storageCellRepo.GetByID(ctx, *batch.StorageCellID)
		if err == nil {
			locationCode = &cell.Code
		}
	}

	if batch.BoxID != nil {
		box, err := s.boxRepo.GetByID(ctx, *batch.BoxID)
		if err == nil {
			parentCode = &box.Code
			if locationCode == nil && box.StorageCellID != nil {
				cell, err := s.storageCellRepo.GetByID(ctx, *box.StorageCellID)
				if err == nil {
					locationCode = &cell.Code
				}
			}
		}
	}

	if parentCode == nil && batch.PalletID != nil {
		pallet, err := s.palletRepo.GetByID(ctx, *batch.PalletID)
		if err == nil {
			parentCode = &pallet.Code
			if locationCode == nil && pallet.StorageCellID != nil {
				cell, err := s.storageCellRepo.GetByID(ctx, *pallet.StorageCellID)
				if err == nil {
					locationCode = &cell.Code
				}
			}
		}
	}

	quantity := batch.Quantity

	return models.ObjectCard{
		MarkerCode:   marker.MarkerCode,
		ObjectType:   marker.ObjectType,
		ObjectID:     batch.ID,
		Code:         batch.Code,
		Name:         batch.Code,
		Status:       batch.Status,
		LocationCode: locationCode,
		ParentCode:   parentCode,
		Quantity:     &quantity,
	}, nil
}
