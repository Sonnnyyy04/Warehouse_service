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

type OperationHistoryUseCase interface {
	Create(ctx context.Context, input service.CreateOperationInput) (models.OperationHistory, error)
	List(ctx context.Context, filter models.OperationHistoryFilter) ([]models.OperationHistory, error)
}

type OperationHistoryHandler struct {
	useCase OperationHistoryUseCase
}

func NewOperationHistoryHandler(useCase OperationHistoryUseCase) *OperationHistoryHandler {
	return &OperationHistoryHandler{useCase: useCase}
}

type CreateOperationRequest struct {
	ObjectType    string           `json:"object_type"`
	ObjectID      int64            `json:"object_id"`
	OperationType string           `json:"operation_type"`
	UserID        *int64           `json:"user_id"`
	Details       *json.RawMessage `json:"details" swaggertype:"object"`
}

// Create godoc
// @Summary Создать запись истории операций
// @Description Сохраняет информацию об операции, выполненной над складским объектом.
// @Tags Операции
// @Accept json
// @Produce json
// @Param request body CreateOperationRequest true "Данные операции"
// @Success 201 {object} models.OperationHistory "Операция сохранена"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/operations [post]
func (h *OperationHistoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req CreateOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	operation, err := h.useCase.Create(ctx, service.CreateOperationInput{
		ObjectType:    req.ObjectType,
		ObjectID:      req.ObjectID,
		OperationType: req.OperationType,
		UserID:        req.UserID,
		Details:       req.Details,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidOperation):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid operation payload",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal server error",
			})
		}
		return
	}

	writeJSON(w, http.StatusCreated, operation)
}

// List godoc
// @Summary Получить историю операций
// @Description Возвращает последние складские операции.
// @Tags Операции
// @Produce json
// @Param limit query int false "Максимальное число записей; по умолчанию 50, максимум 200"
// @Success 200 {array} models.OperationHistory "Список операций"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
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
		case errors.Is(err, service.ErrInvalidLimit):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid limit",
			})
		case errors.Is(err, service.ErrInvalidOperationHistoryFilter):
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
