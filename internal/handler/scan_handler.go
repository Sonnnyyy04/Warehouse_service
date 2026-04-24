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

type ScanUseCase interface {
	Execute(ctx context.Context, input service.ScanObjectInput) (models.ScanResult, error)
}

type ScanHandler struct {
	useCase ScanUseCase
}

func NewScanHandler(useCase ScanUseCase) *ScanHandler {
	return &ScanHandler{useCase: useCase}
}

type ScanRequest struct {
	MarkerCode string  `json:"marker_code"`
	UserID     *int64  `json:"user_id"`
	DeviceInfo *string `json:"device_info"`
}

// Execute godoc
// @Summary Сканировать объект по маркеру
// @Description Находит объект по коду маркера и записывает событие сканирования.
// @Tags Сканирование
// @Accept json
// @Produce json
// @Param request body ScanRequest true "Данные сканирования"
// @Success 200 {object} models.ScanResult "Результат сканирования"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 404 {object} ErrorResponse "Объект не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/scan [post]
func (h *ScanHandler) Execute(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	authUser, ok := userFromContext(ctx)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	var req ScanRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	result, err := h.useCase.Execute(ctx, service.ScanObjectInput{
		MarkerCode: req.MarkerCode,
		UserID:     &authUser.ID,
		DeviceInfo: req.DeviceInfo,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidScanPayload):
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

	writeJSON(w, http.StatusOK, result)
}
