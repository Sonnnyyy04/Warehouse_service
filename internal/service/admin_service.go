package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidAdminInput      = errors.New("invalid admin input")
	ErrInvalidAdminImport     = errors.New("invalid admin import")
	ErrEmptyAdminImport       = errors.New("empty admin import")
	ErrInvalidAdminReference  = errors.New("invalid admin reference")
	ErrAdminTargetOccupied    = errors.New("admin target occupied")
	ErrConflictingBatchTarget = errors.New("conflicting batch target")
	ErrMixedBoxProducts       = errors.New("mixed box products")
	ErrAdminConflict          = errors.New("admin conflict")
	ErrAdminProductExists     = errors.New("admin product exists")
)

type AdminProductRepository interface {
	List(ctx context.Context, limit int32) ([]models.Product, error)
	GetByID(ctx context.Context, id int64) (models.Product, error)
	GetByName(ctx context.Context, name string) (models.Product, error)
	Create(ctx context.Context, sku, name, unit string) (models.Product, error)
	Update(ctx context.Context, id int64, sku, name, unit string) (models.Product, error)
}

type AdminStorageCellRepository interface {
	List(ctx context.Context, limit int32) ([]models.StorageCell, error)
	GetByID(ctx context.Context, id int64) (models.StorageCell, error)
	GetByCode(ctx context.Context, code string) (models.StorageCell, error)
	Create(ctx context.Context, code, name string, zone *string, status string) (models.StorageCell, error)
	Update(ctx context.Context, id int64, code, name string, zone *string, status string) (models.StorageCell, error)
}

type AdminBoxRepository interface {
	List(ctx context.Context, limit int32) ([]models.Box, error)
	GetByID(ctx context.Context, id int64) (models.Box, error)
	GetByCode(ctx context.Context, code string) (models.Box, error)
	HasAnyInStorageCell(ctx context.Context, storageCellID int64) (bool, error)
	Create(ctx context.Context, code, status string, storageCellID *int64) (models.Box, error)
	Update(ctx context.Context, id int64, code, status string, storageCellID *int64) (models.Box, error)
}

type AdminBatchRepository interface {
	List(ctx context.Context, limit int32) ([]models.Batch, error)
	GetByID(ctx context.Context, id int64) (models.Batch, error)
	HasOtherProductInBox(ctx context.Context, boxID int64, productID int64, excludeBatchID *int64) (bool, error)
	HasAnyInBox(ctx context.Context, boxID int64) (bool, error)
	HasAnyInStorageCell(ctx context.Context, storageCellID int64) (bool, error)
	Create(ctx context.Context, code string, productID int64, quantity int32, status string, boxID *int64, palletID *int64, storageCellID *int64) (models.Batch, error)
	Update(ctx context.Context, id int64, code string, productID int64, quantity int32, status string, boxID *int64, palletID *int64, storageCellID *int64) (models.Batch, error)
}

type AdminMarkerRepository interface {
	Create(ctx context.Context, markerCode, objectType string, objectID int64) (models.Marker, error)
}

type AdminUserRepository interface {
	ListByRole(ctx context.Context, role string, limit int32) ([]models.User, error)
	ListByRoles(ctx context.Context, roles []string, limit int32) ([]models.User, error)
	Create(ctx context.Context, login, email, fullName, role, passwordHash string) (models.User, error)
}

type CreateProductInput struct {
	SKU             string
	Name            string
	Unit            string
	InitialQuantity int32
	BoxCode         string
	StorageCellCode string
}

type UpdateProductInput struct {
	ID   int64
	SKU  string
	Name string
	Unit string
}

type CreateStorageCellInput struct {
	Code string
	Name string
	Zone string
}

type CreateBoxInput struct {
	Code          string
	StorageCellID *int64
}

type UpdateStorageCellInput struct {
	ID   int64
	Code string
	Name string
	Zone string
}

type UpdateBoxInput struct {
	ID            int64
	Code          string
	StorageCellID *int64
}

type CreateBatchInput struct {
	Code          string
	ProductID     int64
	Quantity      int32
	BoxID         *int64
	StorageCellID *int64
}

type UpdateBatchInput struct {
	ID            int64
	Code          string
	ProductID     int64
	Quantity      int32
	BoxID         *int64
	StorageCellID *int64
}

type CreateWorkerInput struct {
	Login    string
	FullName string
	Password string
	Email    string
}

type AdminService struct {
	productRepo     AdminProductRepository
	storageCellRepo AdminStorageCellRepository
	boxRepo         AdminBoxRepository
	batchRepo       AdminBatchRepository
	markerRepo      AdminMarkerRepository
	userRepo        AdminUserRepository
	txPool          repository.TxBeginner
}

func NewAdminService(
	productRepo AdminProductRepository,
	storageCellRepo AdminStorageCellRepository,
	boxRepo AdminBoxRepository,
	batchRepo AdminBatchRepository,
	markerRepo AdminMarkerRepository,
	userRepo AdminUserRepository,
	txPool repository.TxBeginner,
) *AdminService {
	return &AdminService{
		productRepo:     productRepo,
		storageCellRepo: storageCellRepo,
		boxRepo:         boxRepo,
		batchRepo:       batchRepo,
		markerRepo:      markerRepo,
		userRepo:        userRepo,
		txPool:          txPool,
	}
}

func (s *AdminService) ListProducts(ctx context.Context, limit int32) ([]models.Product, error) {
	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	return s.productRepo.List(ctx, normalizedLimit)
}

func (s *AdminService) ListStorageCells(ctx context.Context, limit int32) ([]models.StorageCell, error) {
	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	return s.storageCellRepo.List(ctx, normalizedLimit)
}

func (s *AdminService) ListBoxes(ctx context.Context, limit int32) ([]models.Box, error) {
	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	return s.boxRepo.List(ctx, normalizedLimit)
}

func (s *AdminService) ListBatches(ctx context.Context, limit int32) ([]models.Batch, error) {
	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	return s.batchRepo.List(ctx, normalizedLimit)
}

func (s *AdminService) ListWorkers(ctx context.Context, limit int32) ([]models.User, error) {
	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	users, err := s.userRepo.ListByRoles(ctx, []string{"admin", "worker"}, normalizedLimit)
	if err != nil {
		return nil, err
	}

	for i := range users {
		users[i] = sanitizeAdminUser(users[i])
	}

	return users, nil
}

func (s *AdminService) CreateProduct(ctx context.Context, input CreateProductInput) (models.Product, models.Marker, error) {
	sku := strings.TrimSpace(input.SKU)
	name := strings.TrimSpace(input.Name)
	unit := strings.TrimSpace(input.Unit)
	boxCode := strings.TrimSpace(input.BoxCode)
	storageCellCode := strings.TrimSpace(input.StorageCellCode)
	if unit == "" {
		unit = "pcs"
	}

	if sku == "" || name == "" {
		return models.Product{}, models.Marker{}, ErrInvalidAdminInput
	}
	if input.InitialQuantity <= 0 {
		return models.Product{}, models.Marker{}, ErrInvalidAdminInput
	}
	if !hasSingleProductTarget(boxCode, storageCellCode) {
		return models.Product{}, models.Marker{}, ErrConflictingBatchTarget
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.Product{}, models.Marker{}, fmt.Errorf("begin create product tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	productRepo := repository.NewProductRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)
	batchRepo := repository.NewBatchRepositoryWithQuerier(tx)
	boxRepo := repository.NewBoxRepositoryWithQuerier(tx)
	storageCellRepo := repository.NewStorageCellRepositoryWithQuerier(tx)

	var boxID *int64
	if boxCode != "" {
		box, err := boxRepo.GetByCode(ctx, boxCode)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Product{}, models.Marker{}, ErrInvalidAdminReference
			}
			return models.Product{}, models.Marker{}, err
		}

		hasBatches, err := batchRepo.HasAnyInBox(ctx, box.ID)
		if err != nil {
			return models.Product{}, models.Marker{}, err
		}
		if hasBatches {
			return models.Product{}, models.Marker{}, ErrMixedBoxProducts
		}

		boxID = &box.ID
	}

	var storageCellID *int64
	if storageCellCode != "" {
		cell, err := storageCellRepo.GetByCode(ctx, storageCellCode)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Product{}, models.Marker{}, ErrInvalidAdminReference
			}
			return models.Product{}, models.Marker{}, err
		}

		if err := ensureStorageCellIsAvailable(ctx, boxRepo, batchRepo, cell.ID); err != nil {
			return models.Product{}, models.Marker{}, err
		}

		storageCellID = &cell.ID
	}

	existingProduct, err := productRepo.GetByName(ctx, name)
	if err == nil && existingProduct.ID > 0 {
		return models.Product{}, models.Marker{}, ErrAdminProductExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return models.Product{}, models.Marker{}, err
	}

	product, err := productRepo.Create(ctx, sku, name, unit)
	if err != nil {
		return models.Product{}, models.Marker{}, err
	}

	marker, err := markerRepo.Create(ctx, buildMarkerCode("product", product.ID), "product", product.ID)
	if err != nil {
		return models.Product{}, models.Marker{}, err
	}

	if _, err := batchRepo.Create(
		ctx,
		buildInitialBatchCode(product.ID),
		product.ID,
		input.InitialQuantity,
		"active",
		boxID,
		nil,
		storageCellID,
	); err != nil {
		return models.Product{}, models.Marker{}, err
	}

	product, err = productRepo.GetByID(ctx, product.ID)
	if err != nil {
		return models.Product{}, models.Marker{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Product{}, models.Marker{}, fmt.Errorf("commit create product tx: %w", err)
	}

	return product, marker, nil
}

func (s *AdminService) UpdateProduct(ctx context.Context, input UpdateProductInput) (models.Product, error) {
	sku := strings.TrimSpace(input.SKU)
	name := strings.TrimSpace(input.Name)
	unit := strings.TrimSpace(input.Unit)
	if unit == "" {
		unit = "pcs"
	}

	if input.ID <= 0 || sku == "" || name == "" {
		return models.Product{}, ErrInvalidAdminInput
	}

	existingProduct, err := s.productRepo.GetByName(ctx, name)
	if err == nil && existingProduct.ID != input.ID {
		return models.Product{}, ErrAdminProductExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return models.Product{}, err
	}

	product, err := s.productRepo.Update(ctx, input.ID, sku, name, unit)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Product{}, ErrInvalidAdminReference
		}
		return models.Product{}, err
	}

	return product, nil
}

func (s *AdminService) CreateStorageCell(ctx context.Context, input CreateStorageCellInput) (models.StorageCell, models.Marker, error) {
	code := strings.TrimSpace(input.Code)
	name := strings.TrimSpace(input.Name)
	zone := strings.TrimSpace(input.Zone)
	if name == "" {
		name = code
	}

	if code == "" {
		return models.StorageCell{}, models.Marker{}, ErrInvalidAdminInput
	}

	var zoneValue *string
	if zone != "" {
		zoneValue = &zone
	}

	cell, err := s.storageCellRepo.Create(ctx, code, name, zoneValue, "active")
	if err != nil {
		return models.StorageCell{}, models.Marker{}, err
	}

	marker, err := s.markerRepo.Create(ctx, buildMarkerCode("storage_cell", cell.ID), "storage_cell", cell.ID)
	if err != nil {
		return models.StorageCell{}, models.Marker{}, err
	}

	return cell, marker, nil
}

func (s *AdminService) UpdateStorageCell(ctx context.Context, input UpdateStorageCellInput) (models.StorageCell, error) {
	code := strings.TrimSpace(input.Code)
	name := strings.TrimSpace(input.Name)
	zone := strings.TrimSpace(input.Zone)
	if name == "" {
		name = code
	}

	if input.ID <= 0 || code == "" {
		return models.StorageCell{}, ErrInvalidAdminInput
	}

	var zoneValue *string
	if zone != "" {
		zoneValue = &zone
	}

	cell, err := s.storageCellRepo.Update(ctx, input.ID, code, name, zoneValue, "active")
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.StorageCell{}, ErrInvalidAdminReference
		}
		return models.StorageCell{}, err
	}

	return cell, nil
}

func (s *AdminService) CreateBox(ctx context.Context, input CreateBoxInput) (models.Box, models.Marker, error) {
	code := strings.TrimSpace(input.Code)
	if code == "" {
		return models.Box{}, models.Marker{}, ErrInvalidAdminInput
	}

	if input.StorageCellID != nil {
		if _, err := s.storageCellRepo.GetByID(ctx, *input.StorageCellID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Box{}, models.Marker{}, ErrInvalidAdminReference
			}
			return models.Box{}, models.Marker{}, err
		}

		if err := ensureStorageCellIsAvailable(ctx, s.boxRepo, s.batchRepo, *input.StorageCellID); err != nil {
			return models.Box{}, models.Marker{}, err
		}
	}

	box, err := s.boxRepo.Create(ctx, code, "active", input.StorageCellID)
	if err != nil {
		return models.Box{}, models.Marker{}, err
	}

	marker, err := s.markerRepo.Create(ctx, buildMarkerCode("box", box.ID), "box", box.ID)
	if err != nil {
		return models.Box{}, models.Marker{}, err
	}

	return box, marker, nil
}

func (s *AdminService) UpdateBox(ctx context.Context, input UpdateBoxInput) (models.Box, error) {
	code := strings.TrimSpace(input.Code)
	if input.ID <= 0 || code == "" {
		return models.Box{}, ErrInvalidAdminInput
	}

	currentBox, err := s.boxRepo.GetByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Box{}, ErrInvalidAdminReference
		}
		return models.Box{}, err
	}

	if input.StorageCellID != nil {
		if _, err := s.storageCellRepo.GetByID(ctx, *input.StorageCellID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Box{}, ErrInvalidAdminReference
			}
			return models.Box{}, err
		}

		if currentBox.StorageCellID == nil || *currentBox.StorageCellID != *input.StorageCellID {
			if err := ensureStorageCellIsAvailable(ctx, s.boxRepo, s.batchRepo, *input.StorageCellID); err != nil {
				return models.Box{}, err
			}
		}
	}

	box, err := s.boxRepo.Update(ctx, input.ID, code, "active", input.StorageCellID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Box{}, ErrInvalidAdminReference
		}
		return models.Box{}, err
	}

	return box, nil
}

func (s *AdminService) CreateBatch(ctx context.Context, input CreateBatchInput) (models.Batch, models.Marker, error) {
	code := strings.TrimSpace(input.Code)
	if code == "" || input.ProductID <= 0 || input.Quantity <= 0 {
		return models.Batch{}, models.Marker{}, ErrInvalidAdminInput
	}

	if !hasSingleBatchTarget(input.BoxID, input.StorageCellID) {
		return models.Batch{}, models.Marker{}, ErrConflictingBatchTarget
	}

	if _, err := s.productRepo.GetByID(ctx, input.ProductID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Batch{}, models.Marker{}, ErrInvalidAdminReference
		}
		return models.Batch{}, models.Marker{}, err
	}

	if input.BoxID != nil {
		if _, err := s.boxRepo.GetByID(ctx, *input.BoxID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Batch{}, models.Marker{}, ErrInvalidAdminReference
			}
			return models.Batch{}, models.Marker{}, err
		}

		hasMixedProducts, err := s.batchRepo.HasOtherProductInBox(ctx, *input.BoxID, input.ProductID, nil)
		if err != nil {
			return models.Batch{}, models.Marker{}, err
		}
		if hasMixedProducts {
			return models.Batch{}, models.Marker{}, ErrMixedBoxProducts
		}
	}

	if input.StorageCellID != nil {
		if _, err := s.storageCellRepo.GetByID(ctx, *input.StorageCellID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Batch{}, models.Marker{}, ErrInvalidAdminReference
			}
			return models.Batch{}, models.Marker{}, err
		}

		if err := ensureStorageCellIsAvailable(ctx, s.boxRepo, s.batchRepo, *input.StorageCellID); err != nil {
			return models.Batch{}, models.Marker{}, err
		}
	}

	batch, err := s.batchRepo.Create(
		ctx,
		code,
		input.ProductID,
		input.Quantity,
		"active",
		input.BoxID,
		nil,
		input.StorageCellID,
	)
	if err != nil {
		return models.Batch{}, models.Marker{}, err
	}

	marker, err := s.markerRepo.Create(ctx, buildMarkerCode("batch", batch.ID), "batch", batch.ID)
	if err != nil {
		return models.Batch{}, models.Marker{}, err
	}

	return batch, marker, nil
}

func (s *AdminService) UpdateBatch(ctx context.Context, input UpdateBatchInput) (models.Batch, error) {
	code := strings.TrimSpace(input.Code)
	if input.ID <= 0 || code == "" || input.ProductID <= 0 || input.Quantity <= 0 {
		return models.Batch{}, ErrInvalidAdminInput
	}

	if !hasSingleBatchTarget(input.BoxID, input.StorageCellID) {
		return models.Batch{}, ErrConflictingBatchTarget
	}

	currentBatch, err := s.batchRepo.GetByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Batch{}, ErrInvalidAdminReference
		}
		return models.Batch{}, err
	}

	if _, err := s.productRepo.GetByID(ctx, input.ProductID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Batch{}, ErrInvalidAdminReference
		}
		return models.Batch{}, err
	}

	if input.BoxID != nil {
		if _, err := s.boxRepo.GetByID(ctx, *input.BoxID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Batch{}, ErrInvalidAdminReference
			}
			return models.Batch{}, err
		}

		hasMixedProducts, err := s.batchRepo.HasOtherProductInBox(ctx, *input.BoxID, input.ProductID, &currentBatch.ID)
		if err != nil {
			return models.Batch{}, err
		}
		if hasMixedProducts {
			return models.Batch{}, ErrMixedBoxProducts
		}
	}

	if input.StorageCellID != nil {
		if _, err := s.storageCellRepo.GetByID(ctx, *input.StorageCellID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Batch{}, ErrInvalidAdminReference
			}
			return models.Batch{}, err
		}

		if currentBatch.StorageCellID == nil || *currentBatch.StorageCellID != *input.StorageCellID {
			if err := ensureStorageCellIsAvailable(ctx, s.boxRepo, s.batchRepo, *input.StorageCellID); err != nil {
				return models.Batch{}, err
			}
		}
	}

	batch, err := s.batchRepo.Update(
		ctx,
		input.ID,
		code,
		input.ProductID,
		input.Quantity,
		"active",
		input.BoxID,
		nil,
		input.StorageCellID,
	)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Batch{}, ErrInvalidAdminReference
		}
		return models.Batch{}, err
	}

	return batch, nil
}

func (s *AdminService) CreateWorker(ctx context.Context, input CreateWorkerInput) (models.User, error) {
	login := strings.TrimSpace(strings.ToLower(input.Login))
	fullName := strings.TrimSpace(input.FullName)
	password := strings.TrimSpace(input.Password)
	email := strings.TrimSpace(strings.ToLower(input.Email))

	if login == "" || fullName == "" || password == "" || len(password) < 6 {
		return models.User{}, ErrInvalidAdminInput
	}

	if email == "" {
		email = login + "@warehouse.local"
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}

	user, err := s.userRepo.Create(ctx, login, email, fullName, "worker", string(passwordHash))
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return models.User{}, ErrAdminConflict
		}
		return models.User{}, err
	}

	return sanitizeAdminUser(user), nil
}

func buildMarkerCode(objectType string, objectID int64) string {
	switch objectType {
	case "storage_cell":
		return fmt.Sprintf("MRK-CELL-%03d", objectID)
	case "pallet":
		return fmt.Sprintf("MRK-PALLET-%03d", objectID)
	case "box":
		return fmt.Sprintf("MRK-BOX-%03d", objectID)
	case "product":
		return fmt.Sprintf("MRK-PRODUCT-%03d", objectID)
	case "batch":
		return fmt.Sprintf("MRK-BATCH-%03d", objectID)
	default:
		return fmt.Sprintf("MRK-OBJECT-%03d", objectID)
	}
}

func sanitizeAdminUser(user models.User) models.User {
	user.PasswordHash = ""
	return user
}

func hasSingleBatchTarget(boxID *int64, storageCellID *int64) bool {
	return (boxID == nil) != (storageCellID == nil)
}

func hasSingleProductTarget(boxCode string, storageCellCode string) bool {
	return (boxCode == "") != (storageCellCode == "")
}

func buildInitialBatchCode(productID int64) string {
	return fmt.Sprintf("BAT-INIT-%06d", productID)
}

func ensureStorageCellIsAvailable(
	ctx context.Context,
	boxRepo AdminBoxRepository,
	batchRepo AdminBatchRepository,
	storageCellID int64,
) error {
	hasBoxes, err := boxRepo.HasAnyInStorageCell(ctx, storageCellID)
	if err != nil {
		return err
	}
	if hasBoxes {
		return ErrAdminTargetOccupied
	}

	hasBatches, err := batchRepo.HasAnyInStorageCell(ctx, storageCellID)
	if err != nil {
		return err
	}
	if hasBatches {
		return ErrAdminTargetOccupied
	}

	return nil
}
