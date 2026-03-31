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

type ObjectUseCase interface {
	GetByMarkerCode(ctx context.Context, markerCode string) (models.ObjectCard, error)
}

type ObjectHandler struct {
	useCase ObjectUseCase
}

func NewObjectHandler(useCase ObjectUseCase) *ObjectHandler {
	return &ObjectHandler{useCase: useCase}
}

// GetByMarkerCode godoc
// @Summary Получить объект по коду маркера
// @Description Возвращает унифицированную карточку объекта для указанного маркера.
// @Tags Объекты
// @Produce json
// @Param marker_code query string true "Код маркера"
// @Success 200 {object} models.ObjectCard "Карточка объекта"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 404 {object} ErrorResponse "Объект не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/objects/by-marker [get]
func (h *ObjectHandler) GetByMarkerCode(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	markerCode := r.URL.Query().Get("marker_code")

	card, err := h.useCase.GetByMarkerCode(ctx, markerCode)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidMarkerCode):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "marker_code is required",
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

	writeJSON(w, http.StatusOK, card)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
