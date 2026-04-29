package handler

import (
	"Warehouse_service/internal/models"
	"Warehouse_service/internal/service"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type MoveBatchUseCase interface {
	Execute(ctx context.Context, input service.MoveBatchInput) (models.MoveBatchResult, error)
}

type MoveBatchHandler struct {
	useCase MoveBatchUseCase
}

func NewMoveBatchHandler(useCase MoveBatchUseCase) *MoveBatchHandler {
	return &MoveBatchHandler{useCase: useCase}
}

type MoveBatchRequest struct {
	BatchMarkerCode  string `json:"batch_marker_code"`
	TargetMarkerCode string `json:"target_marker_code"`
	UserID           *int64 `json:"user_id"`
}

func (h *MoveBatchHandler) Execute(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	authUser, ok := userFromContext(ctx)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	var req MoveBatchRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	result, err := h.useCase.Execute(ctx, service.MoveBatchInput{
		BatchMarkerCode:  req.BatchMarkerCode,
		TargetMarkerCode: req.TargetMarkerCode,
		UserID:           &authUser.ID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidMoveBatchPayload):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "batch_marker_code and target_marker_code are required",
			})
		case errors.Is(err, service.ErrInvalidBatchMarkerType):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "batch_marker_code must point to batch",
			})
		case errors.Is(err, service.ErrInvalidBatchTargetMarkerType):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "target_marker_code must point to box or storage_cell",
			})
		case errors.Is(err, service.ErrBatchAlreadyInTargetBox):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "batch already in target box",
			})
		case errors.Is(err, service.ErrBatchAlreadyInTargetCell):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "batch already in target cell",
			})
		case errors.Is(err, service.ErrMixedBoxProducts):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "box can store only one product",
			})
		case errors.Is(err, service.ErrAdminTargetOccupied):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "storage cell is occupied",
			})
		case errors.Is(err, service.ErrObjectNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "object not found",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal server error",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}
