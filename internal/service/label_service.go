package service

import (
	"context"
	"errors"
	"strings"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"

	"github.com/skip2/go-qrcode"
)

var (
	ErrInvalidLabelObjectType = errors.New("invalid label object type")
	ErrInvalidLabelMarkerCode = errors.New("invalid label marker code")
)

type LabelMarkerRepository interface {
	List(ctx context.Context, objectType string, limit int32) ([]models.Marker, error)
}

type LabelStorageCellRepository interface {
	GetByID(ctx context.Context, id int64) (models.StorageCell, error)
}

type LabelPalletRepository interface {
	GetByID(ctx context.Context, id int64) (models.Pallet, error)
}

type LabelBoxRepository interface {
	GetByID(ctx context.Context, id int64) (models.Box, error)
}

type LabelProductRepository interface {
	GetByID(ctx context.Context, id int64) (models.Product, error)
}

type LabelBatchRepository interface {
	GetByID(ctx context.Context, id int64) (models.Batch, error)
}

type LabelService struct {
	markerRepo      LabelMarkerRepository
	storageCellRepo LabelStorageCellRepository
	palletRepo      LabelPalletRepository
	boxRepo         LabelBoxRepository
	productRepo     LabelProductRepository
	batchRepo       LabelBatchRepository
}

func NewLabelService(
	markerRepo LabelMarkerRepository,
	storageCellRepo LabelStorageCellRepository,
	palletRepo LabelPalletRepository,
	boxRepo LabelBoxRepository,
	productRepo LabelProductRepository,
	batchRepo LabelBatchRepository,
) *LabelService {
	return &LabelService{
		markerRepo:      markerRepo,
		storageCellRepo: storageCellRepo,
		palletRepo:      palletRepo,
		boxRepo:         boxRepo,
		productRepo:     productRepo,
		batchRepo:       batchRepo,
	}
}

func (s *LabelService) List(ctx context.Context, objectType string, limit int32) ([]models.Label, error) {
	objectType = strings.TrimSpace(objectType)
	if objectType != "" && !isSupportedLabelObjectType(objectType) {
		return nil, ErrInvalidLabelObjectType
	}

	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	markers, err := s.markerRepo.List(ctx, objectType, normalizedLimit)
	if err != nil {
		return nil, err
	}

	labels := make([]models.Label, 0, len(markers))

	for _, marker := range markers {
		label, err := s.buildLabel(ctx, marker)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				continue
			}
			return nil, err
		}

		labels = append(labels, label)
	}

	return labels, nil
}

func (s *LabelService) GenerateQRCodePNG(markerCode string, size int) ([]byte, error) {
	markerCode = strings.TrimSpace(markerCode)
	if markerCode == "" {
		return nil, ErrInvalidLabelMarkerCode
	}

	if size <= 0 {
		size = 256
	}

	if size > 1024 {
		size = 1024
	}

	return qrcode.Encode(markerCode, qrcode.Medium, size)
}

func (s *LabelService) buildLabel(ctx context.Context, marker models.Marker) (models.Label, error) {
	switch marker.ObjectType {
	case "storage_cell":
		cell, err := s.storageCellRepo.GetByID(ctx, marker.ObjectID)
		if err != nil {
			return models.Label{}, err
		}

		return models.Label{
			MarkerCode: marker.MarkerCode,
			ObjectType: marker.ObjectType,
			ObjectID:   marker.ObjectID,
			Code:       cell.Code,
			Name:       cell.Name,
		}, nil
	case "pallet":
		pallet, err := s.palletRepo.GetByID(ctx, marker.ObjectID)
		if err != nil {
			return models.Label{}, err
		}

		return models.Label{
			MarkerCode: marker.MarkerCode,
			ObjectType: marker.ObjectType,
			ObjectID:   marker.ObjectID,
			Code:       pallet.Code,
			Name:       pallet.Code,
		}, nil
	case "box":
		box, err := s.boxRepo.GetByID(ctx, marker.ObjectID)
		if err != nil {
			return models.Label{}, err
		}

		return models.Label{
			MarkerCode: marker.MarkerCode,
			ObjectType: marker.ObjectType,
			ObjectID:   marker.ObjectID,
			Code:       box.Code,
			Name:       box.Code,
		}, nil
	case "product":
		product, err := s.productRepo.GetByID(ctx, marker.ObjectID)
		if err != nil {
			return models.Label{}, err
		}

		return models.Label{
			MarkerCode: marker.MarkerCode,
			ObjectType: marker.ObjectType,
			ObjectID:   marker.ObjectID,
			Code:       product.SKU,
			Name:       product.Name,
		}, nil
	case "batch":
		batch, err := s.batchRepo.GetByID(ctx, marker.ObjectID)
		if err != nil {
			return models.Label{}, err
		}

		return models.Label{
			MarkerCode: marker.MarkerCode,
			ObjectType: marker.ObjectType,
			ObjectID:   marker.ObjectID,
			Code:       batch.Code,
			Name:       batch.Code,
		}, nil
	default:
		return models.Label{}, ErrInvalidLabelObjectType
	}
}

func isSupportedLabelObjectType(value string) bool {
	switch value {
	case "storage_cell", "pallet", "box", "product", "batch":
		return true
	default:
		return false
	}
}
