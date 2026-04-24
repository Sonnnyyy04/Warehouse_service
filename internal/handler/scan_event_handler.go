package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/service"
)

type ScanEventUseCase interface {
	Create(ctx context.Context, input service.CreateScanEventInput) (models.ScanEvent, error)
	List(ctx context.Context, filter models.ScanEventFilter) ([]models.ScanEvent, error)
}

type ScanEventHandler struct {
	useCase ScanEventUseCase
}

func NewScanEventHandler(useCase ScanEventUseCase) *ScanEventHandler {
	return &ScanEventHandler{useCase: useCase}
}

type CreateScanEventRequest struct {
	MarkerCode string  `json:"marker_code"`
	UserID     *int64  `json:"user_id"`
	DeviceInfo *string `json:"device_info"`
	Success    *bool   `json:"success"`
}

// Create godoc
// @Summary Создать событие сканирования
// @Description Сохраняет попытку сканирования по коду маркера.
// @Tags События сканирования
// @Accept json
// @Produce json
// @Param request body CreateScanEventRequest true "Данные события сканирования"
// @Success 201 {object} models.ScanEvent "Событие сканирования сохранено"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/scan-events [post]
func (h *ScanEventHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	authUser, ok := userFromContext(ctx)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	var req CreateScanEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	event, err := h.useCase.Create(ctx, service.CreateScanEventInput{
		MarkerCode: req.MarkerCode,
		UserID:     &authUser.ID,
		DeviceInfo: req.DeviceInfo,
		Success:    req.Success,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidScanEvent):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "marker_code is required",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal server error",
			})
		}
		return
	}

	writeJSON(w, http.StatusCreated, event)
}

// List godoc
// @Summary Получить события сканирования
// @Description Возвращает последние события сканирования маркеров.
// @Tags События сканирования
// @Produce json
// @Param limit query int false "Максимальное число записей; по умолчанию 50, максимум 200"
// @Success 200 {array} models.ScanEvent "Список событий сканирования"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/scan-events [get]
func (h *ScanEventHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	authUser, ok := userFromContext(ctx)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	filter := models.ScanEventFilter{}
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

	filter.MarkerCode = r.URL.Query().Get("marker_code")

	if authUser.Role != "admin" {
		if filter.MarkerCode == "" {
			filter.UserID = &authUser.ID
		} else {
			filter.UserID = nil
		}
	}

	events, err := h.useCase.List(ctx, filter)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLimit):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid limit",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal server error",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, events)
}
