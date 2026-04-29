package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/service"
)

type OperationHistoryUseCase interface {
	List(ctx context.Context, filter models.OperationHistoryFilter) ([]models.OperationHistory, error)
}

type OperationHistoryHandler struct {
	useCase OperationHistoryUseCase
}

func NewOperationHistoryHandler(useCase OperationHistoryUseCase) *OperationHistoryHandler {
	return &OperationHistoryHandler{useCase: useCase}
}

// List godoc
// @Summary Ð ÑŸÐ Ñ•Ð Â»Ð¡Ñ“Ð¡â€¡Ð Ñ‘Ð¡â€šÐ¡ÐŠ Ð Ñ‘Ð¡ÐƒÐ¡â€šÐ Ñ•Ð¡Ð‚Ð Ñ‘Ð¡Ð‹ Ð Ñ•Ð Ñ—Ð ÂµÐ¡Ð‚Ð Â°Ð¡â€ Ð Ñ‘Ð â„–
// @Description Ð â€™Ð Ñ•Ð Â·Ð Ð†Ð¡Ð‚Ð Â°Ð¡â€°Ð Â°Ð ÂµÐ¡â€š Ð Ñ—Ð Ñ•Ð¡ÐƒÐ Â»Ð ÂµÐ Ò‘Ð Ð…Ð Ñ‘Ð Âµ Ð¡ÐƒÐ Ñ”Ð Â»Ð Â°Ð Ò‘Ð¡ÐƒÐ Ñ”Ð Ñ‘Ð Âµ Ð Ñ•Ð Ñ—Ð ÂµÐ¡Ð‚Ð Â°Ð¡â€ Ð Ñ‘Ð Ñ‘.
// @Tags Ð Ñ›Ð Ñ—Ð ÂµÐ¡Ð‚Ð Â°Ð¡â€ Ð Ñ‘Ð Ñ‘
// @Produce json
// @Param limit query int false "Ð ÑšÐ Â°Ð Ñ”Ð¡ÐƒÐ Ñ‘Ð Ñ˜Ð Â°Ð Â»Ð¡ÐŠÐ Ð…Ð Ñ•Ð Âµ Ð¡â€¡Ð Ñ‘Ð¡ÐƒÐ Â»Ð Ñ• Ð Â·Ð Â°Ð Ñ—Ð Ñ‘Ð¡ÐƒÐ ÂµÐ â„–; Ð Ñ—Ð Ñ• Ð¡Ñ“Ð Ñ˜Ð Ñ•Ð Â»Ð¡â€¡Ð Â°Ð Ð…Ð Ñ‘Ð¡Ð‹ 50, Ð Ñ˜Ð Â°Ð Ñ”Ð¡ÐƒÐ Ñ‘Ð Ñ˜Ð¡Ñ“Ð Ñ˜ 200"
// @Success 200 {array} models.OperationHistory "Ð ÐŽÐ Ñ—Ð Ñ‘Ð¡ÐƒÐ Ñ•Ð Ñ” Ð Ñ•Ð Ñ—Ð ÂµÐ¡Ð‚Ð Â°Ð¡â€ Ð Ñ‘Ð â„–"
// @Failure 400 {object} ErrorResponse "Ð ÑœÐ ÂµÐ Ñ”Ð Ñ•Ð¡Ð‚Ð¡Ð‚Ð ÂµÐ Ñ”Ð¡â€šÐ Ð…Ð¡â€¹Ð â„– Ð Â·Ð Â°Ð Ñ—Ð¡Ð‚Ð Ñ•Ð¡Ðƒ"
// @Failure 500 {object} ErrorResponse "Ð â€™Ð Ð…Ð¡Ñ“Ð¡â€šÐ¡Ð‚Ð ÂµÐ Ð…Ð Ð…Ð¡ÐÐ¡Ð Ð Ñ•Ð¡â‚¬Ð Ñ‘Ð Â±Ð Ñ”Ð Â° Ð¡ÐƒÐ ÂµÐ¡Ð‚Ð Ð†Ð ÂµÐ¡Ð‚Ð Â°"
// @Router /api/v1/operations [get]
func (h *OperationHistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	authUser, ok := userFromContext(ctx)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	filter := models.OperationHistoryFilter{}
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid limit",
			})
			return
		}
		filter.Limit = int32(parsedLimit)
	}

	if rawUserID := r.URL.Query().Get("user_id"); rawUserID != "" {
		parsedUserID, err := strconv.ParseInt(rawUserID, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid user_id",
			})
			return
		}
		filter.UserID = &parsedUserID
	}

	filter.ObjectType = r.URL.Query().Get("object_type")

	if rawObjectID := r.URL.Query().Get("object_id"); rawObjectID != "" {
		parsedObjectID, err := strconv.ParseInt(rawObjectID, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid object_id",
			})
			return
		}
		filter.ObjectID = &parsedObjectID
	}

	if authUser.Role != "admin" {
		if filter.ObjectType == "" && filter.ObjectID == nil {
			filter.UserID = &authUser.ID
		} else {
			filter.UserID = nil
		}
	}

	operations, err := h.useCase.List(ctx, filter)
	if err != nil {
		switch {
		case err == service.ErrInvalidLimit:
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid limit",
			})
		case err == service.ErrInvalidOperationHistoryFilter:
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid operation history filter",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal server error",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, operations)
}
