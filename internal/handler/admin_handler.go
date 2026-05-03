package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"
	"Warehouse_service/internal/service"
)

type AdminUseCase interface {
	ListProducts(ctx context.Context, limit int32) ([]models.Product, error)
	ListRacks(ctx context.Context, limit int32) ([]models.Rack, error)
	ListStorageCells(ctx context.Context, limit int32) ([]models.StorageCell, error)
	ListBoxes(ctx context.Context, limit int32) ([]models.Box, error)
	ListBatches(ctx context.Context, limit int32) ([]models.Batch, error)
	ListWorkers(ctx context.Context, limit int32) ([]models.User, error)
	CreateProduct(ctx context.Context, input service.CreateProductInput) (models.Product, models.Marker, error)
	ImportProducts(ctx context.Context, reader io.Reader) (models.ProductImportResult, error)
	UpdateProduct(ctx context.Context, input service.UpdateProductInput) (models.Product, error)
	DeleteProduct(ctx context.Context, id int64) error
	CreateRack(ctx context.Context, input service.CreateRackInput) (models.Rack, models.Marker, error)
	UpdateRack(ctx context.Context, input service.UpdateRackInput) (models.Rack, error)
	DeleteRack(ctx context.Context, id int64) error
	CreateStorageCell(ctx context.Context, input service.CreateStorageCellInput) (models.StorageCell, models.Marker, error)
	UpdateStorageCell(ctx context.Context, input service.UpdateStorageCellInput) (models.StorageCell, error)
	DeleteStorageCell(ctx context.Context, id int64) error
	CreateBox(ctx context.Context, input service.CreateBoxInput) (models.Box, models.Marker, error)
	UpdateBox(ctx context.Context, input service.UpdateBoxInput) (models.Box, error)
	DeleteBox(ctx context.Context, id int64) error
	CreateBatch(ctx context.Context, input service.CreateBatchInput) (models.Batch, models.Marker, error)
	UpdateBatch(ctx context.Context, input service.UpdateBatchInput) (models.Batch, error)
	DeleteBatch(ctx context.Context, id int64) error
	CreateWorker(ctx context.Context, input service.CreateWorkerInput) (models.User, error)
	DeleteWorker(ctx context.Context, actor models.User, userID int64) error
}

type AdminHandler struct {
	adminUseCase AdminUseCase
}

func NewAdminHandler(adminUseCase AdminUseCase) *AdminHandler {
	return &AdminHandler{adminUseCase: adminUseCase}
}

type createWorkerRequest struct {
	Login    string `json:"login"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type createProductRequest struct {
	SKU             string `json:"sku"`
	Name            string `json:"name"`
	Unit            string `json:"unit"`
	InitialQuantity int32  `json:"initial_quantity"`
	BoxCode         string `json:"box_code"`
	StorageCellCode string `json:"storage_cell_code"`
}

type createProductResponse struct {
	Product    models.Product `json:"product"`
	MarkerCode string         `json:"marker_code"`
}

type createStorageCellRequest struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Zone   string `json:"zone"`
	RackID *int64 `json:"rack_id"`
}

type createRackRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Zone string `json:"zone"`
}

type createRackResponse struct {
	Rack       models.Rack `json:"rack"`
	MarkerCode string      `json:"marker_code"`
}

type createStorageCellResponse struct {
	StorageCell models.StorageCell `json:"storage_cell"`
	MarkerCode  string             `json:"marker_code"`
}

type updateProductRequest struct {
	ID   int64  `json:"id"`
	SKU  string `json:"sku"`
	Name string `json:"name"`
	Unit string `json:"unit"`
}

type updateStorageCellRequest struct {
	ID     int64  `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Zone   string `json:"zone"`
	RackID *int64 `json:"rack_id"`
}

type updateRackRequest struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Zone string `json:"zone"`
}

type createBoxRequest struct {
	Code          string `json:"code"`
	StorageCellID *int64 `json:"storage_cell_id"`
}

type createBoxResponse struct {
	Box        models.Box `json:"box"`
	MarkerCode string     `json:"marker_code"`
}

type updateBoxRequest struct {
	ID            int64  `json:"id"`
	Code          string `json:"code"`
	StorageCellID *int64 `json:"storage_cell_id"`
}

type createBatchRequest struct {
	Code          string `json:"code"`
	ProductID     int64  `json:"product_id"`
	Quantity      int32  `json:"quantity"`
	BoxID         *int64 `json:"box_id"`
	StorageCellID *int64 `json:"storage_cell_id"`
}

type createBatchResponse struct {
	Batch      models.Batch `json:"batch"`
	MarkerCode string       `json:"marker_code"`
}

type updateBatchRequest struct {
	ID            int64  `json:"id"`
	Code          string `json:"code"`
	ProductID     int64  `json:"product_id"`
	Quantity      int32  `json:"quantity"`
	BoxID         *int64 `json:"box_id"`
	StorageCellID *int64 `json:"storage_cell_id"`
}

type deleteWorkerRequest struct {
	ID int64 `json:"id"`
}

type deleteEntityRequest struct {
	ID int64 `json:"id"`
}

func (h *AdminHandler) ListProductsAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var limit int32
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		limit = int32(parsedLimit)
	}

	products, err := h.adminUseCase.ListProducts(ctx, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLimit):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, products)
}

func (h *AdminHandler) CreateProductAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req createProductRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	product, marker, err := h.adminUseCase.CreateProduct(ctx, service.CreateProductInput{
		SKU:             req.SKU,
		Name:            req.Name,
		Unit:            req.Unit,
		InitialQuantity: req.InitialQuantity,
		BoxCode:         req.BoxCode,
		StorageCellCode: req.StorageCellCode,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sku, name and initial quantity must be valid"})
		case errors.Is(err, service.ErrConflictingBatchTarget):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "initial quantity requires exactly one target: box_code or storage_cell_code"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "box or storage cell not found"})
		case errors.Is(err, service.ErrAdminTargetOccupied):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target storage cell must be empty for a new product"})
		case errors.Is(err, service.ErrStorageCellProductConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "storage cell can store only one product"})
		case errors.Is(err, service.ErrMixedBoxProducts):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target box must be empty for a new product"})
		case errors.Is(err, service.ErrAdminProductExists):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "product name already exists"})
		case errors.Is(err, repository.ErrConflict), errors.Is(err, service.ErrAdminConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "product sku already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, createProductResponse{
		Product:    product,
		MarkerCode: marker.MarkerCode,
	})
}

func (h *AdminHandler) ImportProductsAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()

	result, err := h.adminUseCase.ImportProducts(ctx, file)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminImport):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid excel file"})
		case errors.Is(err, service.ErrEmptyAdminImport):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "excel file contains no products"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *AdminHandler) UpdateProductAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req updateProductRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	product, err := h.adminUseCase.UpdateProduct(ctx, service.UpdateProductInput{
		ID:   req.ID,
		SKU:  req.SKU,
		Name: req.Name,
		Unit: req.Unit,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sku and name are required"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		case errors.Is(err, service.ErrAdminProductExists):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "product name already exists"})
		case errors.Is(err, repository.ErrConflict), errors.Is(err, service.ErrAdminConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "product sku already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, product)
}

func (h *AdminHandler) DeleteProductAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req deleteEntityRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.adminUseCase.DeleteProduct(ctx, req.ID); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product_id"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		case errors.Is(err, service.ErrAdminProductHasBatches):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "product cannot be deleted while batches exist"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) ListRacksAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var limit int32
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		limit = int32(parsedLimit)
	}

	racks, err := h.adminUseCase.ListRacks(ctx, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLimit):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, racks)
}

func (h *AdminHandler) CreateRackAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req createRackRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	rack, marker, err := h.adminUseCase.CreateRack(ctx, service.CreateRackInput{
		Code: req.Code,
		Name: req.Name,
		Zone: req.Zone,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code is required"})
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "rack code already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, createRackResponse{
		Rack:       rack,
		MarkerCode: marker.MarkerCode,
	})
}

func (h *AdminHandler) UpdateRackAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req updateRackRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	rack, err := h.adminUseCase.UpdateRack(ctx, service.UpdateRackInput{
		ID:   req.ID,
		Code: req.Code,
		Name: req.Name,
		Zone: req.Zone,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code is required"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "rack not found"})
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "rack code already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, rack)
}

func (h *AdminHandler) DeleteRackAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req deleteEntityRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.adminUseCase.DeleteRack(ctx, req.ID); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid rack_id"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "rack not found"})
		case errors.Is(err, service.ErrAdminRackBusy):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "rack cannot be deleted while it contains storage cells"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) ListStorageCellsAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var limit int32
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		limit = int32(parsedLimit)
	}

	storageCells, err := h.adminUseCase.ListStorageCells(ctx, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLimit):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, storageCells)
}

func (h *AdminHandler) CreateStorageCellAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req createStorageCellRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	storageCell, marker, err := h.adminUseCase.CreateStorageCell(ctx, service.CreateStorageCellInput{
		Code:   req.Code,
		Name:   req.Name,
		Zone:   req.Zone,
		RackID: req.RackID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code is required"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "rack not found"})
		case errors.Is(err, service.ErrAdminTargetOccupied):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target storage cell must be empty"})
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "storage cell code already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, createStorageCellResponse{
		StorageCell: storageCell,
		MarkerCode:  marker.MarkerCode,
	})
}

func (h *AdminHandler) UpdateStorageCellAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req updateStorageCellRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	storageCell, err := h.adminUseCase.UpdateStorageCell(ctx, service.UpdateStorageCellInput{
		ID:     req.ID,
		Code:   req.Code,
		Name:   req.Name,
		Zone:   req.Zone,
		RackID: req.RackID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code is required"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "storage cell or rack not found"})
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "storage cell code already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, storageCell)
}

func (h *AdminHandler) DeleteStorageCellAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req deleteEntityRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.adminUseCase.DeleteStorageCell(ctx, req.ID); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid storage_cell_id"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "storage cell not found"})
		case errors.Is(err, service.ErrAdminStorageCellBusy):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "storage cell cannot be deleted while it contains boxes or batches"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) ListBoxesAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var limit int32
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		limit = int32(parsedLimit)
	}

	boxes, err := h.adminUseCase.ListBoxes(ctx, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLimit):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, boxes)
}

func (h *AdminHandler) CreateBoxAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req createBoxRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	box, marker, err := h.adminUseCase.CreateBox(ctx, service.CreateBoxInput{
		Code:          req.Code,
		StorageCellID: req.StorageCellID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code is required"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "storage cell not found"})
		case errors.Is(err, service.ErrAdminTargetOccupied):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target storage cell must be empty"})
		case errors.Is(err, service.ErrStorageCellProductConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "storage cell can store only one product"})
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "box code already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, createBoxResponse{
		Box:        box,
		MarkerCode: marker.MarkerCode,
	})
}

func (h *AdminHandler) UpdateBoxAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req updateBoxRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	box, err := h.adminUseCase.UpdateBox(ctx, service.UpdateBoxInput{
		ID:            req.ID,
		Code:          req.Code,
		StorageCellID: req.StorageCellID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code is required"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "box or storage cell not found"})
		case errors.Is(err, service.ErrAdminTargetOccupied):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target storage cell must be empty"})
		case errors.Is(err, service.ErrStorageCellProductConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "storage cell can store only one product"})
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "box code already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, box)
}

func (h *AdminHandler) DeleteBoxAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req deleteEntityRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.adminUseCase.DeleteBox(ctx, req.ID); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid box_id"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "box not found"})
		case errors.Is(err, service.ErrAdminBoxNotEmpty):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "box cannot be deleted while batches exist"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) ListBatchesAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var limit int32
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		limit = int32(parsedLimit)
	}

	batches, err := h.adminUseCase.ListBatches(ctx, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLimit):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, batches)
}

func (h *AdminHandler) CreateBatchAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req createBatchRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	batch, marker, err := h.adminUseCase.CreateBatch(ctx, service.CreateBatchInput{
		Code:          req.Code,
		ProductID:     req.ProductID,
		Quantity:      req.Quantity,
		BoxID:         req.BoxID,
		StorageCellID: req.StorageCellID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code, product_id and quantity are required"})
		case errors.Is(err, service.ErrConflictingBatchTarget):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "choose either box or storage cell"})
		case errors.Is(err, service.ErrAdminTargetOccupied):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target storage cell must be empty"})
		case errors.Is(err, service.ErrMixedBoxProducts):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "box can store only one product"})
		case errors.Is(err, service.ErrStorageCellProductConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "storage cell can store only one product"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product, box or storage cell not found"})
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "batch code already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, createBatchResponse{
		Batch:      batch,
		MarkerCode: marker.MarkerCode,
	})
}

func (h *AdminHandler) UpdateBatchAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req updateBatchRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	batch, err := h.adminUseCase.UpdateBatch(ctx, service.UpdateBatchInput{
		ID:            req.ID,
		Code:          req.Code,
		ProductID:     req.ProductID,
		Quantity:      req.Quantity,
		BoxID:         req.BoxID,
		StorageCellID: req.StorageCellID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code, product_id and quantity are required"})
		case errors.Is(err, service.ErrConflictingBatchTarget):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "choose either box or storage cell"})
		case errors.Is(err, service.ErrAdminTargetOccupied):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "target storage cell must be empty"})
		case errors.Is(err, service.ErrMixedBoxProducts):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "box can store only one product"})
		case errors.Is(err, service.ErrStorageCellProductConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "storage cell can store only one product"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product, box or storage cell not found"})
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "batch code already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, batch)
}

func (h *AdminHandler) DeleteBatchAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req deleteEntityRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.adminUseCase.DeleteBatch(ctx, req.ID); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid batch_id"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "batch not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) ListWorkersAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var limit int32
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		limit = int32(parsedLimit)
	}

	workers, err := h.adminUseCase.ListWorkers(ctx, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLimit):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, workers)
}

func (h *AdminHandler) CreateWorkerAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	authUser, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req createWorkerRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	worker, err := h.adminUseCase.CreateWorker(ctx, service.CreateWorkerInput{
		Actor:    authUser,
		Login:    req.Login,
		FullName: req.FullName,
		Password: req.Password,
		Email:    req.Email,
		Role:     req.Role,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "login, full_name and password are required; role must be admin or worker"})
		case errors.Is(err, service.ErrInvalidAdminLogin):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "login must contain only latin letters"})
		case errors.Is(err, service.ErrInvalidAdminPassword):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be between 6 and 20 characters"})
		case errors.Is(err, service.ErrAdminPermissionDenied):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only super admin can manage admin accounts"})
		case errors.Is(err, service.ErrAdminConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "user login already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, worker)
}

func (h *AdminHandler) DeleteWorkerAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	authUser, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req deleteWorkerRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.adminUseCase.DeleteWorker(ctx, authUser, req.ID); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user_id"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		case errors.Is(err, service.ErrAdminSelfDelete):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot delete current user"})
		case errors.Is(err, service.ErrAdminPermissionDenied):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only super admin can manage admin accounts"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
