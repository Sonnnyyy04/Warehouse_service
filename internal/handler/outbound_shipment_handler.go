package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/service"
)

type OutboundShipmentUseCase interface {
	Complete(ctx context.Context, input service.OutboundShipmentInput) (models.OutboundShipmentResult, error)
}

type OutboundShipmentHandler struct {
	useCase OutboundShipmentUseCase
}

func NewOutboundShipmentHandler(useCase OutboundShipmentUseCase) *OutboundShipmentHandler {
	return &OutboundShipmentHandler{useCase: useCase}
}

type CompleteOutboundShipmentRequest struct {
	ProductID         int64    `json:"product_id"`
	RequestedQuantity int32    `json:"requested_quantity"`
	BoxMarkerCodes    []string `json:"box_marker_codes"`
}

func (h *OutboundShipmentHandler) Complete(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	authUser, ok := userFromContext(ctx)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req CompleteOutboundShipmentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	result, err := h.useCase.Complete(ctx, service.OutboundShipmentInput{
		ProductID:         req.ProductID,
		RequestedQuantity: req.RequestedQuantity,
		BoxMarkerCodes:    req.BoxMarkerCodes,
		UserID:            &authUser.ID,
		Actor:             service.UserSummaryFromUser(authUser),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidOutboundShipmentPayload):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "product_id, requested_quantity and box_marker_codes are required"})
		case errors.Is(err, service.ErrInvalidBoxMarkerType):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "box_marker_codes must point to boxes"})
		case errors.Is(err, service.ErrOutboundShipmentBoxNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "selected box not found or does not contain product"})
		case errors.Is(err, service.ErrOutboundShipmentNotEnoughStock):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "selected boxes do not cover requested quantity"})
		case errors.Is(err, service.ErrObjectNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}
