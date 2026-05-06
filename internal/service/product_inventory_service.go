package service

import (
	"context"
	"errors"
	"strings"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"
)

var ErrProductNotFound = errors.New("product not found")

type ProductInventoryRepository interface {
	Search(ctx context.Context, query string, limit int32) ([]models.Product, error)
	GetByID(ctx context.Context, id int64) (models.Product, error)
	ListLocations(ctx context.Context, productID int64) ([]models.ProductLocation, error)
}

type ProductInventoryService struct {
	productRepo ProductInventoryRepository
}

func NewProductInventoryService(productRepo ProductInventoryRepository) *ProductInventoryService {
	return &ProductInventoryService{productRepo: productRepo}
}

func (s *ProductInventoryService) SearchProducts(ctx context.Context, query string, limit int32) ([]models.Product, error) {
	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	return s.productRepo.Search(ctx, strings.TrimSpace(query), normalizedLimit)
}

func (s *ProductInventoryService) GetProductLocations(ctx context.Context, productID int64) (models.ProductLocations, error) {
	if productID <= 0 {
		return models.ProductLocations{}, ErrInvalidAdminInput
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.ProductLocations{}, ErrProductNotFound
		}
		return models.ProductLocations{}, err
	}

	locations, err := s.productRepo.ListLocations(ctx, productID)
	if err != nil {
		return models.ProductLocations{}, err
	}

	return models.ProductLocations{
		Product:   product,
		Locations: locations,
	}, nil
}
