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

type MoveBoxUseCase interface {
	Execute(ctx context.Context, input service.MoveBoxInput) (models.MoveBoxResult, error)
}

type MoveBoxHandler struct {
	useCase MoveBoxUseCase
}

func NewMoveBoxHandler(useCase MoveBoxUseCase) *MoveBoxHandler {
	return &MoveBoxHandler{useCase: useCase}
}

type MoveBoxRequest struct {
	BoxMarkerCode           string `json:"box_marker_code"`
	ToStorageCellMarkerCode string `json:"to_storage_cell_marker_code"`
	UserID                  *int64 `json:"user_id"`
}

// Execute godoc
// @Summary Переместить короб в ячейку
// @Description Перемещает короб по коду маркера в целевую ячейку хранения и записывает операцию в историю.
// @Tags Короба
// @Accept json
// @Produce json
// @Param request body MoveBoxRequest true "Данные для перемещения короба"
// @Success 200 {object} models.MoveBoxResult "Короб успешно перемещён"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 404 {object} ErrorResponse "Объект не найден"
// @Failure 409 {object} ErrorResponse "Конфликт состояния"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/boxes/move [post]
func (h *MoveBoxHandler) Execute(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req MoveBoxRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	result, err := h.useCase.Execute(ctx, service.MoveBoxInput{
		BoxMarkerCode:           req.BoxMarkerCode,
		ToStorageCellMarkerCode: req.ToStorageCellMarkerCode,
		UserID:                  req.UserID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidMoveBoxPayload):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "box_marker_code and to_storage_cell_marker_code are required",
			})
		case errors.Is(err, service.ErrInvalidBoxMarkerType):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "box_marker_code must point to box",
			})
		case errors.Is(err, service.ErrInvalidStorageCellMarkerType):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "to_storage_cell_marker_code must point to storage_cell",
			})
		case errors.Is(err, service.ErrBoxAlreadyInTargetCell):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "box already in target cell",
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
