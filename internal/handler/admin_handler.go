package handler

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"
	"Warehouse_service/internal/service"
)

type AdminUseCase interface {
	ListProducts(ctx context.Context, limit int32) ([]models.Product, error)
	ListStorageCells(ctx context.Context, limit int32) ([]models.StorageCell, error)
	ListBoxes(ctx context.Context, limit int32) ([]models.Box, error)
	ListBatches(ctx context.Context, limit int32) ([]models.Batch, error)
	ListWorkers(ctx context.Context, limit int32) ([]models.User, error)
	CreateProduct(ctx context.Context, input service.CreateProductInput) (models.Product, models.Marker, error)
	UpdateProduct(ctx context.Context, input service.UpdateProductInput) (models.Product, error)
	CreateStorageCell(ctx context.Context, input service.CreateStorageCellInput) (models.StorageCell, models.Marker, error)
	CreateBox(ctx context.Context, input service.CreateBoxInput) (models.Box, models.Marker, error)
	CreateBatch(ctx context.Context, input service.CreateBatchInput) (models.Batch, models.Marker, error)
	CreateWorker(ctx context.Context, input service.CreateWorkerInput) (models.User, error)
}

type AdminLabelUseCase interface {
	List(ctx context.Context, objectType string, limit int32) ([]models.Label, error)
}

type AdminHandler struct {
	adminUseCase AdminUseCase
	labelUseCase AdminLabelUseCase
}

func NewAdminHandler(adminUseCase AdminUseCase, labelUseCase AdminLabelUseCase) *AdminHandler {
	return &AdminHandler{
		adminUseCase: adminUseCase,
		labelUseCase: labelUseCase,
	}
}

func (h *AdminHandler) Page(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	objectType := r.URL.Query().Get("object_type")
	if objectType == "" {
		objectType = "box"
	}

	limit := int32(100)
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsedLimit, err := strconv.Atoi(rawLimit); err == nil && parsedLimit > 0 {
			limit = int32(parsedLimit)
		}
	}

	products, err := h.adminUseCase.ListProducts(ctx, 200)
	if err != nil {
		http.Error(w, "failed to load products", http.StatusInternalServerError)
		return
	}

	storageCells, err := h.adminUseCase.ListStorageCells(ctx, 200)
	if err != nil {
		http.Error(w, "failed to load storage cells", http.StatusInternalServerError)
		return
	}

	boxes, err := h.adminUseCase.ListBoxes(ctx, 200)
	if err != nil {
		http.Error(w, "failed to load boxes", http.StatusInternalServerError)
		return
	}

	batches, err := h.adminUseCase.ListBatches(ctx, 100)
	if err != nil {
		http.Error(w, "failed to load batches", http.StatusInternalServerError)
		return
	}

	labels, err := h.labelUseCase.List(ctx, objectType, limit)
	if err != nil {
		http.Error(w, "failed to load labels", http.StatusInternalServerError)
		return
	}

	data := struct {
		Notice       string
		Error        string
		SelectedType string
		Limit        int32
		Products     []models.Product
		StorageCells []models.StorageCell
		Boxes        []models.Box
		Batches      []models.Batch
		Labels       []models.Label
		Types        []adminObjectType
	}{
		Notice:       r.URL.Query().Get("notice"),
		Error:        r.URL.Query().Get("error"),
		SelectedType: objectType,
		Limit:        limit,
		Products:     products,
		StorageCells: storageCells,
		Boxes:        boxes,
		Batches:      batches,
		Labels:       labels,
		Types: []adminObjectType{
			{Value: "box", Label: "Короба"},
			{Value: "storage_cell", Label: "Ячейки"},
			{Value: "batch", Label: "Партии"},
			{Value: "product", Label: "Товары"},
		},
	}

	tpl := template.Must(template.New("admin").Parse(adminTemplate))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.Execute(w, data); err != nil {
		http.Error(w, "failed to render admin page", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.redirectWithError(w, r, "", "invalid product form")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	product, marker, err := h.adminUseCase.CreateProduct(ctx, service.CreateProductInput{
		SKU:  r.FormValue("sku"),
		Name: r.FormValue("name"),
		Unit: r.FormValue("unit"),
	})
	if err != nil {
		h.redirectWithAdminError(w, r, err)
		return
	}

	h.redirectWithNotice(w, r, "product", "Товар "+product.SKU+" создан, QR: "+marker.MarkerCode)
}

func (h *AdminHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.redirectWithError(w, r, "", "invalid product update form")
		return
	}

	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		h.redirectWithError(w, r, "", "invalid product id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	product, err := h.adminUseCase.UpdateProduct(ctx, service.UpdateProductInput{
		ID:   id,
		SKU:  r.FormValue("sku"),
		Name: r.FormValue("name"),
		Unit: r.FormValue("unit"),
	})
	if err != nil {
		h.redirectWithAdminError(w, r, err)
		return
	}

	h.redirectWithNotice(w, r, "product", "Товар "+product.SKU+" обновлён")
}

func (h *AdminHandler) CreateStorageCell(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.redirectWithError(w, r, "", "invalid storage cell form")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	cell, marker, err := h.adminUseCase.CreateStorageCell(ctx, service.CreateStorageCellInput{
		Code: r.FormValue("code"),
		Name: r.FormValue("name"),
		Zone: r.FormValue("zone"),
	})
	if err != nil {
		h.redirectWithAdminError(w, r, err)
		return
	}

	h.redirectWithNotice(w, r, "storage_cell", "Ячейка "+cell.Code+" создана, QR: "+marker.MarkerCode)
}

func (h *AdminHandler) CreateBox(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.redirectWithError(w, r, "", "invalid box form")
		return
	}

	storageCellID, err := parseOptionalInt64(r.FormValue("storage_cell_id"))
	if err != nil {
		h.redirectWithError(w, r, "", "invalid storage cell id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	box, marker, err := h.adminUseCase.CreateBox(ctx, service.CreateBoxInput{
		Code:          r.FormValue("code"),
		StorageCellID: storageCellID,
	})
	if err != nil {
		h.redirectWithAdminError(w, r, err)
		return
	}

	h.redirectWithNotice(w, r, "box", "Короб "+box.Code+" создан, QR: "+marker.MarkerCode)
}

func (h *AdminHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.redirectWithError(w, r, "", "invalid batch form")
		return
	}

	productID, err := strconv.ParseInt(r.FormValue("product_id"), 10, 64)
	if err != nil {
		h.redirectWithError(w, r, "", "invalid product id")
		return
	}

	quantity, err := strconv.Atoi(r.FormValue("quantity"))
	if err != nil {
		h.redirectWithError(w, r, "", "invalid quantity")
		return
	}

	boxID, err := parseOptionalInt64(r.FormValue("box_id"))
	if err != nil {
		h.redirectWithError(w, r, "", "invalid box id")
		return
	}

	storageCellID, err := parseOptionalInt64(r.FormValue("storage_cell_id"))
	if err != nil {
		h.redirectWithError(w, r, "", "invalid storage cell id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	batch, marker, err := h.adminUseCase.CreateBatch(ctx, service.CreateBatchInput{
		Code:          r.FormValue("code"),
		ProductID:     productID,
		Quantity:      int32(quantity),
		BoxID:         boxID,
		StorageCellID: storageCellID,
	})
	if err != nil {
		h.redirectWithAdminError(w, r, err)
		return
	}

	h.redirectWithNotice(w, r, "batch", "Партия "+batch.Code+" создана, QR: "+marker.MarkerCode)
}

type createWorkerRequest struct {
	Login    string `json:"login"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type createProductRequest struct {
	SKU  string `json:"sku"`
	Name string `json:"name"`
	Unit string `json:"unit"`
}

type createProductResponse struct {
	Product    models.Product `json:"product"`
	MarkerCode string         `json:"marker_code"`
}

type createStorageCellRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Zone string `json:"zone"`
}

type createStorageCellResponse struct {
	StorageCell models.StorageCell `json:"storage_cell"`
	MarkerCode  string             `json:"marker_code"`
}

type createBoxRequest struct {
	Code          string `json:"code"`
	StorageCellID *int64 `json:"storage_cell_id"`
}

type createBoxResponse struct {
	Box        models.Box `json:"box"`
	MarkerCode string     `json:"marker_code"`
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	product, marker, err := h.adminUseCase.CreateProduct(ctx, service.CreateProductInput{
		SKU:  req.SKU,
		Name: req.Name,
		Unit: req.Unit,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sku and name are required"})
		case errors.Is(err, service.ErrAdminProductExists):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "такой товар уже существует"})
		case errors.Is(err, repository.ErrConflict), errors.Is(err, service.ErrAdminConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "товар с таким SKU уже существует"})
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	storageCell, marker, err := h.adminUseCase.CreateStorageCell(ctx, service.CreateStorageCellInput{
		Code: req.Code,
		Name: req.Name,
		Zone: req.Zone,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code is required"})
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "ячейка с таким кодом уже существует"})
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "короб с таким кодом уже существует"})
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "выберите либо короб, либо ячейку"})
		case errors.Is(err, service.ErrInvalidAdminReference):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "связанный объект не найден"})
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "партия с таким кодом уже существует"})
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

	var req createWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	worker, err := h.adminUseCase.CreateWorker(ctx, service.CreateWorkerInput{
		Login:    req.Login,
		FullName: req.FullName,
		Password: req.Password,
		Email:    req.Email,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "login, full_name and password are required"})
		case errors.Is(err, service.ErrAdminConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "worker login already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, worker)
}

func (h *AdminHandler) redirectWithAdminError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidAdminInput):
		h.redirectWithError(w, r, "", "заполните обязательные поля")
	case errors.Is(err, service.ErrInvalidAdminReference):
		h.redirectWithError(w, r, "", "ссылка на объект не найдена")
	case errors.Is(err, service.ErrConflictingBatchTarget):
		h.redirectWithError(w, r, "", "укажите либо короб, либо ячейку для партии")
	default:
		h.redirectWithError(w, r, "", "операция не выполнена")
	}
}

func (h *AdminHandler) redirectWithNotice(w http.ResponseWriter, r *http.Request, objectType, message string) {
	values := url.Values{}
	if objectType != "" {
		values.Set("object_type", objectType)
	}
	if message != "" {
		values.Set("notice", message)
	}

	location := "/admin"
	if encoded := values.Encode(); encoded != "" {
		location += "?" + encoded
	}

	http.Redirect(w, r, location, http.StatusSeeOther)
}

func (h *AdminHandler) redirectWithError(w http.ResponseWriter, r *http.Request, objectType, message string) {
	values := url.Values{}
	if objectType != "" {
		values.Set("object_type", objectType)
	}
	if message != "" {
		values.Set("error", message)
	}

	location := "/admin"
	if encoded := values.Encode(); encoded != "" {
		location += "?" + encoded
	}

	http.Redirect(w, r, location, http.StatusSeeOther)
}

func parseOptionalInt64(value string) (*int64, error) {
	if value == "" {
		return nil, nil
	}

	parsedValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, err
	}

	return &parsedValue, nil
}

type adminObjectType struct {
	Value string
	Label string
}

const adminTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Warehouse Admin</title>
  <style>
    :root {
      color-scheme: light;
      --ink: #172033;
      --muted: #637083;
      --paper: #ffffff;
      --line: #d8dde6;
      --accent: #0f766e;
      --accent-dark: #115e59;
      --warn: #f59e0b;
      --danger: #b42318;
      --bg: linear-gradient(180deg, #f2f0ea 0%, #eef6f4 100%);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: var(--bg);
      color: var(--ink);
      font-family: Arial, sans-serif;
    }
    .page {
      max-width: 1320px;
      margin: 0 auto;
      padding: 28px 20px 40px;
    }
    .hero, .panel {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 24px;
      box-shadow: 0 18px 40px rgba(23, 32, 51, 0.06);
    }
    .hero {
      padding: 28px;
      margin-bottom: 18px;
    }
    h1, h2, h3 { margin: 0; }
    h1 { font-size: 34px; }
    .subtitle {
      margin-top: 10px;
      max-width: 840px;
      color: var(--muted);
      line-height: 1.6;
    }
    .flash {
      padding: 14px 16px;
      border-radius: 16px;
      margin-bottom: 14px;
      font-weight: 700;
    }
    .flash.notice {
      background: #ecfdf5;
      color: var(--accent-dark);
      border: 1px solid #a7f3d0;
    }
    .flash.error {
      background: #fff1f2;
      color: var(--danger);
      border: 1px solid #fecdd3;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(12, 1fr);
      gap: 18px;
      margin-top: 18px;
    }
    .panel {
      padding: 22px;
    }
    .span-4 { grid-column: span 4; }
    .span-6 { grid-column: span 6; }
    .span-8 { grid-column: span 8; }
    .span-12 { grid-column: span 12; }
    .muted {
      color: var(--muted);
      line-height: 1.5;
    }
    label {
      display: block;
      margin: 14px 0 8px;
      font-size: 14px;
      font-weight: 700;
    }
    input, select {
      width: 100%;
      min-height: 46px;
      border: 1px solid var(--line);
      border-radius: 14px;
      padding: 0 14px;
      font: inherit;
      background: white;
    }
    .button, .button-secondary {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-height: 46px;
      padding: 0 18px;
      border-radius: 14px;
      font-weight: 700;
      text-decoration: none;
      cursor: pointer;
    }
    .button {
      border: none;
      background: var(--accent);
      color: white;
    }
    .button-secondary {
      border: 1px solid var(--line);
      background: #f8fafc;
      color: var(--ink);
    }
    .actions {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      margin-top: 18px;
    }
    .table {
      width: 100%;
      border-collapse: collapse;
      margin-top: 14px;
    }
    .table th, .table td {
      padding: 10px 8px;
      border-bottom: 1px solid var(--line);
      vertical-align: top;
      text-align: left;
    }
    .table th {
      font-size: 13px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.04em;
    }
    .inline-form {
      display: grid;
      grid-template-columns: 110px 1fr 110px auto;
      gap: 8px;
      align-items: center;
    }
    .labels-list {
      display: grid;
      gap: 10px;
      margin-top: 18px;
      max-height: 460px;
      overflow: auto;
      padding-right: 6px;
    }
    .label-row {
      display: grid;
      grid-template-columns: auto 1fr auto;
      gap: 14px;
      align-items: center;
      border: 1px solid var(--line);
      border-radius: 16px;
      padding: 12px 14px;
      background: #fffdf9;
    }
    .label-meta small {
      display: block;
      color: var(--muted);
      margin-top: 4px;
    }
    .hint {
      margin-top: 12px;
      color: var(--muted);
      font-size: 14px;
      line-height: 1.5;
    }
    @media (max-width: 1080px) {
      .span-4, .span-6, .span-8 { grid-column: span 12; }
      .inline-form { grid-template-columns: 1fr; }
      .label-row { grid-template-columns: auto 1fr; }
    }
  </style>
</head>
<body>
  <main class="page">
    <section class="hero">
      <h1>Админ-панель склада</h1>
      <p class="subtitle">Здесь администратор вручную заводит товары и складские объекты, получает marker_code для новых записей и печатает QR-коды только для выбранных объектов.</p>
    </section>

    {{if .Notice}}<div class="flash notice">{{.Notice}}</div>{{end}}
    {{if .Error}}<div class="flash error">{{.Error}}</div>{{end}}

    <section class="grid">
      <article class="panel span-4">
        <h2>Новый товар</h2>
        <p class="muted">Товар можно завести в систему вручную до поступления на склад. Для новой записи сразу создаётся marker_code.</p>
        <form action="/admin/products" method="post">
          <label for="product-sku">SKU</label>
          <input id="product-sku" name="sku" placeholder="SKU-001" required />
          <label for="product-name">Название</label>
          <input id="product-name" name="name" placeholder="Например, Ноутбук 14" required />
          <label for="product-unit">Единица</label>
          <input id="product-unit" name="unit" value="pcs" />
          <div class="actions">
            <button class="button" type="submit">Создать товар</button>
          </div>
        </form>
      </article>

      <article class="panel span-4">
        <h2>Новая ячейка</h2>
        <p class="muted">Ячейка создаётся вручную, после чего ей сразу присваивается QR-маркер для расклейки на стеллаж или место хранения.</p>
        <form action="/admin/storage-cells" method="post">
          <label for="cell-code">Код ячейки</label>
          <input id="cell-code" name="code" placeholder="A-01-01" required />
          <label for="cell-name">Название</label>
          <input id="cell-name" name="name" placeholder="Стеллаж A / Полка 1 / Ячейка 1" />
          <label for="cell-zone">Зона</label>
          <input id="cell-zone" name="zone" placeholder="A" />
          <div class="actions">
            <button class="button" type="submit">Создать ячейку</button>
          </div>
        </form>
      </article>

      <article class="panel span-4">
        <h2>Новый короб</h2>
        <p class="muted">Короб создаётся как физическая единица хранения. Можно сразу привязать его к ячейке.</p>
        <form action="/admin/boxes" method="post">
          <label for="box-code">Код короба</label>
          <input id="box-code" name="code" placeholder="BOX-101" required />
          <label for="box-cell">Ячейка</label>
          <select id="box-cell" name="storage_cell_id">
            <option value="">Без ячейки</option>
            {{range .StorageCells}}
            <option value="{{.ID}}">{{.Code}} {{if .Name}}- {{.Name}}{{end}}</option>
            {{end}}
          </select>
          <div class="actions">
            <button class="button" type="submit">Создать короб</button>
          </div>
        </form>
      </article>

      <article class="panel span-6">
        <h2>Новая партия</h2>
        <p class="muted">Партия привязывается к товару. При необходимости можно сразу поместить её в короб или в ячейку.</p>
        <form action="/admin/batches" method="post">
          <label for="batch-code">Код партии</label>
          <input id="batch-code" name="code" placeholder="BATCH-2026-001" required />
          <label for="batch-product">Товар</label>
          <select id="batch-product" name="product_id" required>
            <option value="">Выберите товар</option>
            {{range .Products}}
            <option value="{{.ID}}">{{.SKU}} - {{.Name}}</option>
            {{end}}
          </select>
          <label for="batch-quantity">Количество</label>
          <input id="batch-quantity" type="number" min="1" name="quantity" value="1" required />
          <label for="batch-box">Короб</label>
          <select id="batch-box" name="box_id">
            <option value="">Без короба</option>
            {{range .Boxes}}
            <option value="{{.ID}}">{{.Code}}</option>
            {{end}}
          </select>
          <label for="batch-cell">Ячейка</label>
          <select id="batch-cell" name="storage_cell_id">
            <option value="">Без ячейки</option>
            {{range .StorageCells}}
            <option value="{{.ID}}">{{.Code}}</option>
            {{end}}
          </select>
          <p class="hint">Для партии укажите либо короб, либо ячейку, либо оставьте оба поля пустыми.</p>
          <div class="actions">
            <button class="button" type="submit">Создать партию</button>
          </div>
        </form>
      </article>

      <article class="panel span-6">
        <h2>Товары в системе</h2>
        <p class="muted">Редактирование каталога доступно прямо на этой странице. Marker_code товара остаётся прежним.</p>
        <table class="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>SKU / Название / Unit</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {{range .Products}}
            <tr>
              <td>{{.ID}}</td>
              <td colspan="2">
                <form class="inline-form" action="/admin/products/update" method="post">
                  <input type="hidden" name="id" value="{{.ID}}" />
                  <input name="sku" value="{{.SKU}}" required />
                  <input name="name" value="{{.Name}}" required />
                  <input name="unit" value="{{.Unit}}" required />
                  <button class="button-secondary" type="submit">Сохранить</button>
                </form>
              </td>
            </tr>
            {{end}}
          </tbody>
        </table>
      </article>

      <article class="panel span-12">
        <h2>Печать конкретных объектов</h2>
        <p class="muted">Выберите тип, отметьте нужные объекты и сформируйте HTML-печать или PDF только для выбранных marker_code.</p>

        <form action="/admin" method="get">
          <div class="actions">
            <select name="object_type" style="max-width: 240px;">
              {{range .Types}}
              <option value="{{.Value}}" {{if eq $.SelectedType .Value}}selected{{end}}>{{.Label}}</option>
              {{end}}
            </select>
            <input type="number" name="limit" min="1" max="200" value="{{.Limit}}" style="max-width: 160px;" />
            <button class="button-secondary" type="submit">Обновить список</button>
          </div>
        </form>

        <form method="get" target="_blank">
          <input type="hidden" name="object_type" value="{{.SelectedType}}" />
          <input type="hidden" name="limit" value="{{.Limit}}" />

          <div class="actions">
            <button class="button" type="submit" formaction="/labels/print">Открыть HTML-печать</button>
            <button class="button-secondary" type="submit" formaction="/labels/pdf">Скачать PDF</button>
          </div>

          <div class="labels-list">
            {{range .Labels}}
            <label class="label-row">
              <input type="checkbox" name="marker_code" value="{{.MarkerCode}}" />
              <div class="label-meta">
                <strong>{{.Code}}</strong>
                <small>{{.Name}}</small>
                <small>{{.MarkerCode}}</small>
              </div>
              <div>{{.ObjectType}}</div>
            </label>
            {{end}}
          </div>

          <p class="hint">Если ничего не отмечать, печать и PDF будут сформированы для всего текущего списка по выбранному типу.</p>
        </form>
      </article>
    </section>
  </main>
</body>
</html>`
