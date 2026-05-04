package service

import (
	"context"
	"errors"
	"fmt"
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
	GetContentStats(ctx context.Context, storageCellID int64) (models.ObjectContentStats, error)
}

type RackRepository interface {
	GetByID(ctx context.Context, id int64) (models.Rack, error)
	GetContentStats(ctx context.Context, rackID int64) (models.ObjectContentStats, error)
}

type BoxRepository interface {
	GetByID(ctx context.Context, id int64) (models.Box, error)
	GetContentStats(ctx context.Context, boxID int64) (models.ObjectContentStats, error)
}

type ProductRepository interface {
	GetByID(ctx context.Context, id int64) (models.Product, error)
}

type BatchRepository interface {
	GetByID(ctx context.Context, id int64) (models.Batch, error)
}

type ObjectService struct {
	markerRepo      MarkerRepository
	rackRepo        RackRepository
	storageCellRepo StorageCellRepository
	boxRepo         BoxRepository
	productRepo     ProductRepository
	batchRepo       BatchRepository
}

func NewObjectService(
	markerRepo MarkerRepository,
	rackRepo RackRepository,
	storageCellRepo StorageCellRepository,
	boxRepo BoxRepository,
	productRepo ProductRepository,
	batchRepo BatchRepository,
) *ObjectService {
	return &ObjectService{
		markerRepo:      markerRepo,
		rackRepo:        rackRepo,
		storageCellRepo: storageCellRepo,
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
	case "rack":
		return s.buildRackCard(ctx, marker)
	case "storage_cell":
		return s.buildStorageCellCard(ctx, marker)
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

func (s *ObjectService) buildRackCard(ctx context.Context, marker models.Marker) (models.ObjectCard, error) {
	rack, err := s.rackRepo.GetByID(ctx, marker.ObjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.ObjectCard{}, ErrObjectNotFound
		}
		return models.ObjectCard{}, err
	}

	stats, err := s.rackRepo.GetContentStats(ctx, rack.ID)
	if err != nil {
		return models.ObjectCard{}, err
	}
	contentSummary := buildContentSummary(stats)

	return models.ObjectCard{
		MarkerCode:     marker.MarkerCode,
		ObjectType:     marker.ObjectType,
		ObjectID:       rack.ID,
		Code:           rack.Code,
		Name:           rack.Name,
		Status:         rack.Status,
		LocationCode:   &rack.Code,
		LocationType:   stringPtr("rack"),
		ContentSummary: &contentSummary,
		CellsCount:     int32Ptr(stats.CellsCount),
		BoxesCount:     int32Ptr(stats.BoxesCount),
		BatchesCount:   int32Ptr(stats.BatchesCount),
		ProductsCount:  int32Ptr(stats.ProductsCount),
		TotalQuantity:  int32Ptr(stats.TotalQuantity),
	}, nil
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
	locationType := "storage_cell"
	var parentCode *string
	var parentType *string
	if cell.RackCode != nil {
		parentCode = cell.RackCode
		parentType = stringPtr("rack")
	}
	stats, err := s.storageCellRepo.GetContentStats(ctx, cell.ID)
	if err != nil {
		return models.ObjectCard{}, err
	}
	contentSummary := buildContentSummary(stats)

	return models.ObjectCard{
		MarkerCode:     marker.MarkerCode,
		ObjectType:     marker.ObjectType,
		ObjectID:       cell.ID,
		Code:           cell.Code,
		Name:           cell.Name,
		Status:         cell.Status,
		LocationCode:   &locationCode,
		LocationType:   &locationType,
		ParentCode:     parentCode,
		ParentType:     parentType,
		ContentSummary: &contentSummary,
		BoxesCount:     int32Ptr(stats.BoxesCount),
		BatchesCount:   int32Ptr(stats.BatchesCount),
		ProductsCount:  int32Ptr(stats.ProductsCount),
		TotalQuantity:  int32Ptr(stats.TotalQuantity),
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
	var locationType *string
	var parentCode *string
	var parentType *string

	if box.StorageCellID != nil {
		cell, err := s.storageCellRepo.GetByID(ctx, *box.StorageCellID)
		if err == nil {
			locationCode = &cell.Code
			locationType = stringPtr("storage_cell")
		}
	}

	stats, err := s.boxRepo.GetContentStats(ctx, box.ID)
	if err != nil {
		return models.ObjectCard{}, err
	}
	contentSummary := buildContentSummary(stats)

	return models.ObjectCard{
		MarkerCode:     marker.MarkerCode,
		ObjectType:     marker.ObjectType,
		ObjectID:       box.ID,
		Code:           box.Code,
		Name:           firstNonEmptyString(stats.ProductName, box.Code),
		Status:         box.Status,
		LocationCode:   locationCode,
		LocationType:   locationType,
		ParentCode:     parentCode,
		ParentType:     parentType,
		ProductSKU:     stats.ProductSKU,
		ProductName:    stats.ProductName,
		Unit:           stats.ProductUnit,
		ContentSummary: &contentSummary,
		BatchesCount:   int32Ptr(stats.BatchesCount),
		ProductsCount:  int32Ptr(stats.ProductsCount),
		TotalQuantity:  int32Ptr(stats.TotalQuantity),
		Quantity:       int32Ptr(stats.TotalQuantity),
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
		MarkerCode:    marker.MarkerCode,
		ObjectType:    marker.ObjectType,
		ObjectID:      product.ID,
		Code:          product.SKU,
		Name:          product.Name,
		Status:        "active",
		Quantity:      int32Ptr(product.TotalQuantity),
		Unit:          &product.Unit,
		ProductSKU:    &product.SKU,
		ProductName:   &product.Name,
		TotalQuantity: int32Ptr(product.TotalQuantity),
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
	var locationType *string
	var parentCode *string
	var parentType *string

	if batch.StorageCellID != nil {
		cell, err := s.storageCellRepo.GetByID(ctx, *batch.StorageCellID)
		if err == nil {
			locationCode = &cell.Code
			locationType = stringPtr("storage_cell")
		}
	}

	if batch.BoxID != nil {
		box, err := s.boxRepo.GetByID(ctx, *batch.BoxID)
		if err == nil {
			parentCode = &box.Code
			parentType = stringPtr("box")
			if locationCode == nil && box.StorageCellID != nil {
				cell, err := s.storageCellRepo.GetByID(ctx, *box.StorageCellID)
				if err == nil {
					locationCode = &cell.Code
					locationType = stringPtr("storage_cell")
				}
			}
		}
	}

	quantity := batch.Quantity
	product, err := s.productRepo.GetByID(ctx, batch.ProductID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.ObjectCard{}, ErrObjectNotFound
		}
		return models.ObjectCard{}, err
	}
	contentSummary := fmt.Sprintf("%s, %d %s", product.Name, quantity, product.Unit)

	return models.ObjectCard{
		MarkerCode:     marker.MarkerCode,
		ObjectType:     marker.ObjectType,
		ObjectID:       batch.ID,
		Code:           batch.Code,
		Name:           product.Name,
		Status:         batch.Status,
		LocationCode:   locationCode,
		LocationType:   locationType,
		ParentCode:     parentCode,
		ParentType:     parentType,
		Quantity:       &quantity,
		Unit:           &product.Unit,
		ProductSKU:     &product.SKU,
		ProductName:    &product.Name,
		ContentSummary: &contentSummary,
		ProductsCount:  int32Ptr(1),
		TotalQuantity:  &quantity,
	}, nil
}

func buildContentSummary(stats models.ObjectContentStats) string {
	if stats.CellsCount == 0 &&
		stats.BoxesCount == 0 &&
		stats.BatchesCount == 0 &&
		stats.ProductsCount == 0 &&
		stats.TotalQuantity == 0 {
		return "пусто"
	}

	parts := make([]string, 0, 5)
	if stats.CellsCount > 0 {
		parts = append(parts, fmt.Sprintf("ячеек: %d", stats.CellsCount))
	}
	if stats.BoxesCount > 0 {
		parts = append(parts, fmt.Sprintf("коробов: %d", stats.BoxesCount))
	}
	if stats.BatchesCount > 0 {
		parts = append(parts, fmt.Sprintf("партий: %d", stats.BatchesCount))
	}
	if stats.ProductsCount > 0 {
		parts = append(parts, fmt.Sprintf("товаров: %d", stats.ProductsCount))
	}
	if stats.TotalQuantity > 0 {
		parts = append(parts, fmt.Sprintf("единиц: %d", stats.TotalQuantity))
	}

	return strings.Join(parts, ", ")
}

func int32Ptr(value int32) *int32 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func firstNonEmptyString(value *string, fallback string) string {
	if value != nil && strings.TrimSpace(*value) != "" {
		return *value
	}
	return fallback
}
