package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidAdminInput          = errors.New("invalid admin input")
	ErrInvalidAdminLogin          = errors.New("invalid admin login")
	ErrInvalidAdminPassword       = errors.New("invalid admin password")
	ErrAdminPermissionDenied      = errors.New("admin permission denied")
	ErrAdminSelfDelete            = errors.New("admin self delete")
	ErrInvalidAdminImport         = errors.New("invalid admin import")
	ErrEmptyAdminImport           = errors.New("empty admin import")
	ErrInvalidAdminReference      = errors.New("invalid admin reference")
	ErrAdminTargetOccupied        = errors.New("admin target occupied")
	ErrConflictingBatchTarget     = errors.New("conflicting batch target")
	ErrMixedBoxProducts           = errors.New("mixed box products")
	ErrStorageCellProductConflict = errors.New("storage cell product conflict")
	ErrAdminConflict              = errors.New("admin conflict")
	ErrAdminProductExists         = errors.New("admin product exists")
	ErrAdminRackRequired          = errors.New("rack_id is required")
	ErrAdminBoxRequired           = errors.New("box is required for product placement")
	ErrAdminProductHasBatches     = errors.New("admin product has batches")
	ErrAdminBoxNotEmpty           = errors.New("admin box not empty")
	ErrAdminStorageCellBusy       = errors.New("admin storage cell busy")
	ErrAdminRackBusy              = errors.New("admin rack busy")
)

var latinLoginPattern = regexp.MustCompile(`^[a-z]+$`)

type AdminProductRepository interface {
	List(ctx context.Context, limit int32) ([]models.Product, error)
	GetByID(ctx context.Context, id int64) (models.Product, error)
	GetByName(ctx context.Context, name string) (models.Product, error)
	Create(ctx context.Context, sku, name, unit string) (models.Product, error)
	Update(ctx context.Context, id int64, sku, name, unit string) (models.Product, error)
	DeleteByID(ctx context.Context, id int64) error
}

type AdminStorageCellRepository interface {
	List(ctx context.Context, limit int32) ([]models.StorageCell, error)
	GetByID(ctx context.Context, id int64) (models.StorageCell, error)
	GetByCode(ctx context.Context, code string) (models.StorageCell, error)
	Create(ctx context.Context, code, name string, zone *string, status string, rackID *int64) (models.StorageCell, error)
	Update(ctx context.Context, id int64, code, name string, zone *string, status string, rackID *int64) (models.StorageCell, error)
	DeleteByID(ctx context.Context, id int64) error
}

type AdminRackRepository interface {
	List(ctx context.Context, limit int32) ([]models.Rack, error)
	GetByID(ctx context.Context, id int64) (models.Rack, error)
	Create(ctx context.Context, code, name string, zone *string, status string) (models.Rack, error)
	Update(ctx context.Context, id int64, code, name string, zone *string, status string) (models.Rack, error)
	HasAnyStorageCells(ctx context.Context, rackID int64) (bool, error)
	DeleteByID(ctx context.Context, id int64) error
}

type AdminBoxRepository interface {
	List(ctx context.Context, limit int32) ([]models.Box, error)
	GetByID(ctx context.Context, id int64) (models.Box, error)
	GetByCode(ctx context.Context, code string) (models.Box, error)
	HasAnyInStorageCell(ctx context.Context, storageCellID int64) (bool, error)
	Create(ctx context.Context, code, status string, storageCellID *int64) (models.Box, error)
	Update(ctx context.Context, id int64, code, status string, storageCellID *int64) (models.Box, error)
	DeleteByID(ctx context.Context, id int64) error
}

type AdminBatchRepository interface {
	List(ctx context.Context, limit int32) ([]models.Batch, error)
	GetByID(ctx context.Context, id int64) (models.Batch, error)
	HasOtherProductInBox(ctx context.Context, boxID int64, productID int64, excludeBatchID *int64) (bool, error)
	ListProductIDsInBox(ctx context.Context, boxID int64) ([]int64, error)
	HasOtherProductInStorageCell(ctx context.Context, storageCellID int64, productID int64, excludeBatchID *int64, excludeBoxID *int64) (bool, error)
	HasAnyInBox(ctx context.Context, boxID int64) (bool, error)
	HasAnyInStorageCell(ctx context.Context, storageCellID int64) (bool, error)
	HasAnyForProduct(ctx context.Context, productID int64) (bool, error)
	Create(ctx context.Context, code string, productID int64, quantity int32, status string, boxID *int64, storageCellID *int64) (models.Batch, error)
	Update(ctx context.Context, id int64, code string, productID int64, quantity int32, status string, boxID *int64, storageCellID *int64) (models.Batch, error)
	DeleteByID(ctx context.Context, id int64) error
}

type AdminMarkerRepository interface {
	Create(ctx context.Context, markerCode, objectType string, objectID int64) (models.Marker, error)
	DeleteByObject(ctx context.Context, objectType string, objectID int64) error
}

type AdminUserRepository interface {
	ListByRole(ctx context.Context, role string, limit int32) ([]models.User, error)
	ListByRoles(ctx context.Context, roles []string, limit int32) ([]models.User, error)
	GetByID(ctx context.Context, id int64) (models.User, error)
	Create(ctx context.Context, login, email, fullName, role, passwordHash string) (models.User, error)
	DeleteByID(ctx context.Context, id int64) error
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
	Code   string
	Name   string
	Zone   string
	RackID *int64
}

type CreateRackInput struct {
	Code string
	Name string
	Zone string
}

type CreateBoxInput struct {
	Code          string
	StorageCellID *int64
}

type UpdateStorageCellInput struct {
	ID     int64
	Code   string
	Name   string
	Zone   string
	RackID *int64
}

type UpdateRackInput struct {
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
	Actor    models.User
	Login    string
	FullName string
	Password string
	Email    string
	Role     string
}

type AdminService struct {
	productRepo     AdminProductRepository
	rackRepo        AdminRackRepository
	storageCellRepo AdminStorageCellRepository
	boxRepo         AdminBoxRepository
	batchRepo       AdminBatchRepository
	markerRepo      AdminMarkerRepository
	userRepo        AdminUserRepository
	txPool          repository.TxBeginner
}

func NewAdminService(
	productRepo AdminProductRepository,
	rackRepo AdminRackRepository,
	storageCellRepo AdminStorageCellRepository,
	boxRepo AdminBoxRepository,
	batchRepo AdminBatchRepository,
	markerRepo AdminMarkerRepository,
	userRepo AdminUserRepository,
	txPool repository.TxBeginner,
) *AdminService {
	return &AdminService{
		productRepo:     productRepo,
		rackRepo:        rackRepo,
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

func (s *AdminService) ListRacks(ctx context.Context, limit int32) ([]models.Rack, error) {
	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	return s.rackRepo.List(ctx, normalizedLimit)
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
	if boxCode == "" || storageCellCode != "" {
		return models.Product{}, models.Marker{}, ErrAdminBoxRequired
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

	var (
		boxID            *int64
		boxStorageCellID *int64
	)
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
		boxStorageCellID = box.StorageCellID
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

	if boxStorageCellID != nil {
		if err := ensureStorageCellCanAcceptProduct(ctx, batchRepo, *boxStorageCellID, product.ID, nil, boxID); err != nil {
			return models.Product{}, models.Marker{}, err
		}
	}

	if _, err := batchRepo.Create(
		ctx,
		buildInitialBatchCode(product.ID),
		product.ID,
		input.InitialQuantity,
		"active",
		boxID,
		nil,
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

func (s *AdminService) DeleteProduct(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidAdminInput
	}

	hasBatches, err := s.batchRepo.HasAnyForProduct(ctx, id)
	if err != nil {
		return err
	}
	if hasBatches {
		return ErrAdminProductHasBatches
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete product tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	productRepo := repository.NewProductRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)

	if err := productRepo.DeleteByID(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidAdminReference
		}
		return err
	}

	if err := markerRepo.DeleteByObject(ctx, "product", id); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete product tx: %w", err)
	}

	return nil
}

func (s *AdminService) CreateRack(ctx context.Context, input CreateRackInput) (models.Rack, models.Marker, error) {
	code := strings.TrimSpace(input.Code)
	name := strings.TrimSpace(input.Name)
	zone := strings.TrimSpace(input.Zone)
	if name == "" {
		name = code
	}

	if code == "" {
		return models.Rack{}, models.Marker{}, ErrInvalidAdminInput
	}

	var zoneValue *string
	if zone != "" {
		zoneValue = &zone
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.Rack{}, models.Marker{}, fmt.Errorf("begin create rack tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	rackRepo := repository.NewRackRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)

	rack, err := rackRepo.Create(ctx, code, name, zoneValue, "active")
	if err != nil {
		return models.Rack{}, models.Marker{}, err
	}

	marker, err := markerRepo.Create(ctx, buildMarkerCode("rack", rack.ID), "rack", rack.ID)
	if err != nil {
		return models.Rack{}, models.Marker{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Rack{}, models.Marker{}, fmt.Errorf("commit create rack tx: %w", err)
	}

	return rack, marker, nil
}

func (s *AdminService) UpdateRack(ctx context.Context, input UpdateRackInput) (models.Rack, error) {
	code := strings.TrimSpace(input.Code)
	name := strings.TrimSpace(input.Name)
	zone := strings.TrimSpace(input.Zone)
	if name == "" {
		name = code
	}

	if input.ID <= 0 || code == "" {
		return models.Rack{}, ErrInvalidAdminInput
	}

	var zoneValue *string
	if zone != "" {
		zoneValue = &zone
	}

	rack, err := s.rackRepo.Update(ctx, input.ID, code, name, zoneValue, "active")
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Rack{}, ErrInvalidAdminReference
		}
		return models.Rack{}, err
	}

	return rack, nil
}

func (s *AdminService) DeleteRack(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidAdminInput
	}

	hasCells, err := s.rackRepo.HasAnyStorageCells(ctx, id)
	if err != nil {
		return err
	}
	if hasCells {
		return ErrAdminRackBusy
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete rack tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	rackRepo := repository.NewRackRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)

	if err := rackRepo.DeleteByID(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidAdminReference
		}
		return err
	}

	if err := markerRepo.DeleteByObject(ctx, "rack", id); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete rack tx: %w", err)
	}

	return nil
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
	if input.RackID == nil {
		return models.StorageCell{}, models.Marker{}, ErrAdminRackRequired
	}

	var zoneValue *string
	if zone != "" {
		zoneValue = &zone
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.StorageCell{}, models.Marker{}, fmt.Errorf("begin create storage cell tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	storageCellRepo := repository.NewStorageCellRepositoryWithQuerier(tx)
	rackRepo := repository.NewRackRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)

	if _, err := rackRepo.GetByID(ctx, *input.RackID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.StorageCell{}, models.Marker{}, ErrInvalidAdminReference
		}
		return models.StorageCell{}, models.Marker{}, err
	}

	cell, err := storageCellRepo.Create(ctx, code, name, zoneValue, "active", input.RackID)
	if err != nil {
		return models.StorageCell{}, models.Marker{}, err
	}

	marker, err := markerRepo.Create(ctx, buildMarkerCode("storage_cell", cell.ID), "storage_cell", cell.ID)
	if err != nil {
		return models.StorageCell{}, models.Marker{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.StorageCell{}, models.Marker{}, fmt.Errorf("commit create storage cell tx: %w", err)
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
	if input.RackID == nil {
		return models.StorageCell{}, ErrAdminRackRequired
	}

	var zoneValue *string
	if zone != "" {
		zoneValue = &zone
	}

	if _, err := s.rackRepo.GetByID(ctx, *input.RackID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.StorageCell{}, ErrInvalidAdminReference
		}
		return models.StorageCell{}, err
	}

	cell, err := s.storageCellRepo.Update(ctx, input.ID, code, name, zoneValue, "active", input.RackID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.StorageCell{}, ErrInvalidAdminReference
		}
		return models.StorageCell{}, err
	}

	return cell, nil
}

func (s *AdminService) DeleteStorageCell(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidAdminInput
	}

	if err := ensureStorageCellIsAvailable(ctx, s.boxRepo, s.batchRepo, id); err != nil {
		if errors.Is(err, ErrAdminTargetOccupied) {
			return ErrAdminStorageCellBusy
		}
		return err
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete storage cell tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	storageCellRepo := repository.NewStorageCellRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)

	if err := storageCellRepo.DeleteByID(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidAdminReference
		}
		return err
	}

	if err := markerRepo.DeleteByObject(ctx, "storage_cell", id); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete storage cell tx: %w", err)
	}

	return nil
}

func (s *AdminService) CreateBox(ctx context.Context, input CreateBoxInput) (models.Box, models.Marker, error) {
	code := strings.TrimSpace(input.Code)
	if code == "" {
		return models.Box{}, models.Marker{}, ErrInvalidAdminInput
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.Box{}, models.Marker{}, fmt.Errorf("begin create box tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	storageCellRepo := repository.NewStorageCellRepositoryWithQuerier(tx)
	boxRepo := repository.NewBoxRepositoryWithQuerier(tx)
	batchRepo := repository.NewBatchRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)

	if input.StorageCellID != nil {
		if _, err := storageCellRepo.GetByID(ctx, *input.StorageCellID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Box{}, models.Marker{}, ErrInvalidAdminReference
			}
			return models.Box{}, models.Marker{}, err
		}

		if err := ensureStorageCellIsAvailable(ctx, boxRepo, batchRepo, *input.StorageCellID); err != nil {
			return models.Box{}, models.Marker{}, err
		}
	}

	box, err := boxRepo.Create(ctx, code, "active", input.StorageCellID)
	if err != nil {
		return models.Box{}, models.Marker{}, err
	}

	marker, err := markerRepo.Create(ctx, buildMarkerCode("box", box.ID), "box", box.ID)
	if err != nil {
		return models.Box{}, models.Marker{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Box{}, models.Marker{}, fmt.Errorf("commit create box tx: %w", err)
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

		if err := ensureStorageCellCanAcceptBox(ctx, s.boxRepo, s.batchRepo, *input.StorageCellID, currentBox.ID); err != nil {
			return models.Box{}, err
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

func (s *AdminService) DeleteBox(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidAdminInput
	}

	hasBatches, err := s.batchRepo.HasAnyInBox(ctx, id)
	if err != nil {
		return err
	}
	if hasBatches {
		return ErrAdminBoxNotEmpty
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete box tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	boxRepo := repository.NewBoxRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)

	if err := boxRepo.DeleteByID(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidAdminReference
		}
		return err
	}

	if err := markerRepo.DeleteByObject(ctx, "box", id); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete box tx: %w", err)
	}

	return nil
}

func (s *AdminService) CreateBatch(ctx context.Context, input CreateBatchInput) (models.Batch, models.Marker, error) {
	code := strings.TrimSpace(input.Code)
	if code == "" || input.ProductID <= 0 || input.Quantity <= 0 {
		return models.Batch{}, models.Marker{}, ErrInvalidAdminInput
	}

	if input.BoxID == nil || input.StorageCellID != nil {
		return models.Batch{}, models.Marker{}, ErrAdminBoxRequired
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.Batch{}, models.Marker{}, fmt.Errorf("begin create batch tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	productRepo := repository.NewProductRepositoryWithQuerier(tx)
	boxRepo := repository.NewBoxRepositoryWithQuerier(tx)
	batchRepo := repository.NewBatchRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)

	if _, err := productRepo.GetByID(ctx, input.ProductID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Batch{}, models.Marker{}, ErrInvalidAdminReference
		}
		return models.Batch{}, models.Marker{}, err
	}

	if input.BoxID != nil {
		if _, err := boxRepo.GetByID(ctx, *input.BoxID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Batch{}, models.Marker{}, ErrInvalidAdminReference
			}
			return models.Batch{}, models.Marker{}, err
		}

		hasMixedProducts, err := batchRepo.HasOtherProductInBox(ctx, *input.BoxID, input.ProductID, nil)
		if err != nil {
			return models.Batch{}, models.Marker{}, err
		}
		if hasMixedProducts {
			return models.Batch{}, models.Marker{}, ErrMixedBoxProducts
		}

		box, err := boxRepo.GetByID(ctx, *input.BoxID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Batch{}, models.Marker{}, ErrInvalidAdminReference
			}
			return models.Batch{}, models.Marker{}, err
		}
		if box.StorageCellID != nil {
			if err := ensureStorageCellCanAcceptProduct(ctx, batchRepo, *box.StorageCellID, input.ProductID, nil, input.BoxID); err != nil {
				return models.Batch{}, models.Marker{}, err
			}
		}
	}

	batch, err := batchRepo.Create(
		ctx,
		code,
		input.ProductID,
		input.Quantity,
		"active",
		input.BoxID,
		nil,
	)
	if err != nil {
		return models.Batch{}, models.Marker{}, err
	}

	marker, err := markerRepo.Create(ctx, buildMarkerCode("batch", batch.ID), "batch", batch.ID)
	if err != nil {
		return models.Batch{}, models.Marker{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Batch{}, models.Marker{}, fmt.Errorf("commit create batch tx: %w", err)
	}

	return batch, marker, nil
}

func (s *AdminService) UpdateBatch(ctx context.Context, input UpdateBatchInput) (models.Batch, error) {
	code := strings.TrimSpace(input.Code)
	if input.ID <= 0 || code == "" || input.ProductID <= 0 || input.Quantity <= 0 {
		return models.Batch{}, ErrInvalidAdminInput
	}

	if input.BoxID == nil || input.StorageCellID != nil {
		return models.Batch{}, ErrAdminBoxRequired
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

		box, err := s.boxRepo.GetByID(ctx, *input.BoxID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return models.Batch{}, ErrInvalidAdminReference
			}
			return models.Batch{}, err
		}
		if box.StorageCellID != nil {
			if err := ensureStorageCellCanAcceptProduct(ctx, s.batchRepo, *box.StorageCellID, input.ProductID, &currentBatch.ID, input.BoxID); err != nil {
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
	)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Batch{}, ErrInvalidAdminReference
		}
		return models.Batch{}, err
	}

	return batch, nil
}

func (s *AdminService) DeleteBatch(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidAdminInput
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete batch tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	batchRepo := repository.NewBatchRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)

	if err := batchRepo.DeleteByID(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidAdminReference
		}
		return err
	}

	if err := markerRepo.DeleteByObject(ctx, "batch", id); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete batch tx: %w", err)
	}

	return nil
}

func (s *AdminService) CreateWorker(ctx context.Context, input CreateWorkerInput) (models.User, error) {
	login := strings.TrimSpace(strings.ToLower(input.Login))
	fullName := strings.TrimSpace(input.FullName)
	password := strings.TrimSpace(input.Password)
	email := strings.TrimSpace(strings.ToLower(input.Email))
	role := strings.TrimSpace(strings.ToLower(input.Role))
	if role == "" {
		role = "worker"
	}

	if login == "" || fullName == "" || password == "" {
		return models.User{}, ErrInvalidAdminInput
	}
	if !latinLoginPattern.MatchString(login) {
		return models.User{}, ErrInvalidAdminLogin
	}
	if len(password) < 6 || len(password) > 20 {
		return models.User{}, ErrInvalidAdminPassword
	}
	if role != "worker" && role != "admin" {
		return models.User{}, ErrInvalidAdminInput
	}
	if role == "admin" && !input.Actor.IsSuperAdmin {
		return models.User{}, ErrAdminPermissionDenied
	}

	if email == "" {
		email = login + "@warehouse.local"
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}

	user, err := s.userRepo.Create(ctx, login, email, fullName, role, string(passwordHash))
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return models.User{}, ErrAdminConflict
		}
		return models.User{}, err
	}

	return sanitizeAdminUser(user), nil
}

func (s *AdminService) DeleteWorker(ctx context.Context, actor models.User, userID int64) error {
	if userID <= 0 {
		return ErrInvalidAdminInput
	}

	targetUser, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidAdminReference
		}
		return err
	}

	if actor.ID == targetUser.ID {
		return ErrAdminSelfDelete
	}

	if targetUser.Role == "admin" && !actor.IsSuperAdmin {
		return ErrAdminPermissionDenied
	}

	if err := s.userRepo.DeleteByID(ctx, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidAdminReference
		}
		return err
	}

	return nil
}

func buildMarkerCode(objectType string, objectID int64) string {
	switch objectType {
	case "rack":
		return fmt.Sprintf("MRK-RACK-%03d", objectID)
	case "storage_cell":
		return fmt.Sprintf("MRK-CELL-%03d", objectID)
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

type storageCellOccupancyChecker interface {
	HasAnyInStorageCell(ctx context.Context, storageCellID int64) (bool, error)
}

func ensureStorageCellIsAvailable(
	ctx context.Context,
	boxRepo storageCellOccupancyChecker,
	batchRepo storageCellOccupancyChecker,
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

type storageCellProductChecker interface {
	ListProductIDsInBox(ctx context.Context, boxID int64) ([]int64, error)
	HasAnyInStorageCell(ctx context.Context, storageCellID int64) (bool, error)
	HasOtherProductInStorageCell(ctx context.Context, storageCellID int64, productID int64, excludeBatchID *int64, excludeBoxID *int64) (bool, error)
}

func ensureStorageCellCanAcceptBox(
	ctx context.Context,
	boxRepo AdminBoxRepository,
	batchRepo storageCellProductChecker,
	storageCellID int64,
	boxID int64,
) error {
	productIDs, err := batchRepo.ListProductIDsInBox(ctx, boxID)
	if err != nil {
		return err
	}
	if len(productIDs) == 0 {
		return ensureStorageCellIsAvailable(ctx, boxRepo, batchRepo, storageCellID)
	}
	if len(productIDs) > 1 {
		return ErrMixedBoxProducts
	}

	excludeBoxID := boxID
	return ensureStorageCellCanAcceptProduct(ctx, batchRepo, storageCellID, productIDs[0], nil, &excludeBoxID)
}

func ensureStorageCellCanAcceptProduct(
	ctx context.Context,
	batchRepo storageCellProductChecker,
	storageCellID int64,
	productID int64,
	excludeBatchID *int64,
	excludeBoxID *int64,
) error {
	hasOtherProduct, err := batchRepo.HasOtherProductInStorageCell(
		ctx,
		storageCellID,
		productID,
		excludeBatchID,
		excludeBoxID,
	)
	if err != nil {
		return err
	}
	if hasOtherProduct {
		return ErrStorageCellProductConflict
	}

	return nil
}
