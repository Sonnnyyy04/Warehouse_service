package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/service"
)

type ProductInventoryUseCase interface {
	SearchProducts(ctx context.Context, query string, limit int32) ([]models.Product, error)
	GetProductLocations(ctx context.Context, productID int64) (models.ProductLocations, error)
}

type ProductInventoryHandler struct {
	useCase ProductInventoryUseCase
}

func NewProductInventoryHandler(useCase ProductInventoryUseCase) *ProductInventoryHandler {
	return &ProductInventoryHandler{useCase: useCase}
}

func (h *ProductInventoryHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	limit := int32(50)
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		limit = int32(parsedLimit)
	}

	products, err := h.useCase.SearchProducts(ctx, r.URL.Query().Get("q"), limit)
	if err != nil {
		if errors.Is(err, service.ErrInvalidLimit) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, products)
}

func (h *ProductInventoryHandler) GetProductLocations(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	productID, err := strconv.ParseInt(r.URL.Query().Get("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product_id"})
		return
	}

	locations, err := h.useCase.GetProductLocations(ctx, productID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAdminInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product_id"})
		case errors.Is(err, service.ErrProductNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, locations)
}
