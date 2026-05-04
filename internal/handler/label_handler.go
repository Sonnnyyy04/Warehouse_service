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

type LabelUseCase interface {
	List(ctx context.Context, objectType string, limit int32) ([]models.Label, error)
	ListSelected(ctx context.Context, objectType string, markerCodes []string) ([]models.Label, error)
	GenerateQRCodePNG(markerCode string, size int) ([]byte, error)
	GenerateLabelsPDF(labels []models.Label) ([]byte, error)
}

type LabelHandler struct {
	useCase LabelUseCase
}

func NewLabelHandler(useCase LabelUseCase) *LabelHandler {
	return &LabelHandler{useCase: useCase}
}

// List godoc
// @Summary Ð ÑŸÐ Ñ•Ð Â»Ð¡Ñ“Ð¡â€¡Ð Ñ‘Ð¡â€šÐ¡ÐŠ Ð¡ÐƒÐ Ñ—Ð Ñ‘Ð¡ÐƒÐ Ñ•Ð Ñ” Ð Ð…Ð Â°Ð Ñ”Ð Â»Ð ÂµÐ ÂµÐ Ñ”
// @Description Ð â€™Ð Ñ•Ð Â·Ð Ð†Ð¡Ð‚Ð Â°Ð¡â€°Ð Â°Ð ÂµÐ¡â€š Ð¡ÐƒÐ Ñ—Ð Ñ‘Ð¡ÐƒÐ Ñ•Ð Ñ” warehouse-Ð Ñ•Ð Â±Ð¡Ð‰Ð ÂµÐ Ñ”Ð¡â€šÐ Ñ•Ð Ð† Ð¡Ðƒ marker_code Ð Ò‘Ð Â»Ð¡Ð Ð Ñ–Ð ÂµÐ Ð…Ð ÂµÐ¡Ð‚Ð Â°Ð¡â€ Ð Ñ‘Ð Ñ‘ Ð Ñ‘ Ð Ñ—Ð ÂµÐ¡â€¡Ð Â°Ð¡â€šÐ Ñ‘ QR Ð Ð…Ð Â°Ð Ñ”Ð Â»Ð ÂµÐ ÂµÐ Ñ”.
// @Tags Ð ÑœÐ Â°Ð Ñ”Ð Â»Ð ÂµÐ â„–Ð Ñ”Ð Ñ‘
// @Produce json
// @Param object_type query string false "Object type: rack, storage_cell, box, product, batch"
// @Param limit query int false "Ð ÑšÐ Â°Ð Ñ”Ð¡ÐƒÐ Ñ‘Ð Ñ˜Ð Â°Ð Â»Ð¡ÐŠÐ Ð…Ð Ñ•Ð Âµ Ð¡â€¡Ð Ñ‘Ð¡ÐƒÐ Â»Ð Ñ• Ð Â·Ð Â°Ð Ñ—Ð Ñ‘Ð¡ÐƒÐ ÂµÐ â„–; Ð Ñ—Ð Ñ• Ð¡Ñ“Ð Ñ˜Ð Ñ•Ð Â»Ð¡â€¡Ð Â°Ð Ð…Ð Ñ‘Ð¡Ð‹ 50, Ð Ñ˜Ð Â°Ð Ñ”Ð¡ÐƒÐ Ñ‘Ð Ñ˜Ð¡Ñ“Ð Ñ˜ 200"
// @Success 200 {array} models.Label "Ð ÐŽÐ Ñ—Ð Ñ‘Ð¡ÐƒÐ Ñ•Ð Ñ” Ð Ð…Ð Â°Ð Ñ”Ð Â»Ð ÂµÐ ÂµÐ Ñ”"
// @Failure 400 {object} ErrorResponse "Ð ÑœÐ ÂµÐ Ñ”Ð Ñ•Ð¡Ð‚Ð¡Ð‚Ð ÂµÐ Ñ”Ð¡â€šÐ Ð…Ð¡â€¹Ð â„– Ð Â·Ð Â°Ð Ñ—Ð¡Ð‚Ð Ñ•Ð¡Ðƒ"
// @Failure 500 {object} ErrorResponse "Ð â€™Ð Ð…Ð¡Ñ“Ð¡â€šÐ¡Ð‚Ð ÂµÐ Ð…Ð Ð…Ð¡ÐÐ¡Ð Ð Ñ•Ð¡â‚¬Ð Ñ‘Ð Â±Ð Ñ”Ð Â° Ð¡ÐƒÐ ÂµÐ¡Ð‚Ð Ð†Ð ÂµÐ¡Ð‚Ð Â°"
// @Router /api/v1/labels [get]
func (h *LabelHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	labels, err := h.listLabels(ctx, r)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLabelObjectType):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid object_type"})
		case errors.Is(err, service.ErrInvalidLimit):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, labels)
}

// RenderQR godoc
// @Summary Ð ÑŸÐ Ñ•Ð Â»Ð¡Ñ“Ð¡â€¡Ð Ñ‘Ð¡â€šÐ¡ÐŠ QR-Ð Ñ”Ð Ñ•Ð Ò‘ Ð Ñ˜Ð Â°Ð¡Ð‚Ð Ñ”Ð ÂµÐ¡Ð‚Ð Â°
// @Description Ð â€œÐ ÂµÐ Ð…Ð ÂµÐ¡Ð‚Ð Ñ‘Ð¡Ð‚Ð¡Ñ“Ð ÂµÐ¡â€š PNG QR-Ð Ñ”Ð Ñ•Ð Ò‘ Ð Ò‘Ð Â»Ð¡Ð Ð¡Ñ“Ð Ñ”Ð Â°Ð Â·Ð Â°Ð Ð…Ð Ð…Ð Ñ•Ð Ñ–Ð Ñ• marker_code.
// @Tags Ð ÑœÐ Â°Ð Ñ”Ð Â»Ð ÂµÐ â„–Ð Ñ”Ð Ñ‘
// @Produce png
// @Param marker_code query string true "Ð Ñ™Ð Ñ•Ð Ò‘ Ð Ñ˜Ð Â°Ð¡Ð‚Ð Ñ”Ð ÂµÐ¡Ð‚Ð Â°"
// @Param size query int false "Ð Â Ð Â°Ð Â·Ð Ñ˜Ð ÂµÐ¡Ð‚ PNG; Ð Ñ—Ð Ñ• Ð¡Ñ“Ð Ñ˜Ð Ñ•Ð Â»Ð¡â€¡Ð Â°Ð Ð…Ð Ñ‘Ð¡Ð‹ 256"
// @Success 200 {file} binary "PNG QR-Ð Ñ”Ð Ñ•Ð Ò‘"
// @Failure 400 {object} ErrorResponse "Ð ÑœÐ ÂµÐ Ñ”Ð Ñ•Ð¡Ð‚Ð¡Ð‚Ð ÂµÐ Ñ”Ð¡â€šÐ Ð…Ð¡â€¹Ð â„– Ð Â·Ð Â°Ð Ñ—Ð¡Ð‚Ð Ñ•Ð¡Ðƒ"
// @Failure 500 {object} ErrorResponse "Ð â€™Ð Ð…Ð¡Ñ“Ð¡â€šÐ¡Ð‚Ð ÂµÐ Ð…Ð Ð…Ð¡ÐÐ¡Ð Ð Ñ•Ð¡â‚¬Ð Ñ‘Ð Â±Ð Ñ”Ð Â° Ð¡ÐƒÐ ÂµÐ¡Ð‚Ð Ð†Ð ÂµÐ¡Ð‚Ð Â°"
// @Router /api/v1/labels/qr [get]
func (h *LabelHandler) RenderQR(w http.ResponseWriter, r *http.Request) {
	size := 256
	if rawSize := r.URL.Query().Get("size"); rawSize != "" {
		parsedSize, err := strconv.Atoi(rawSize)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid size"})
			return
		}
		size = parsedSize
	}

	pngBytes, err := h.useCase.GenerateQRCodePNG(r.URL.Query().Get("marker_code"), size)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLabelMarkerCode):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "marker_code is required"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pngBytes)
}

func (h *LabelHandler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	objectType := r.URL.Query().Get("object_type")
	labels, err := h.listLabels(ctx, r)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLabelObjectType):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid object_type"})
		case errors.Is(err, service.ErrInvalidLimit):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	pdfBytes, err := h.useCase.GenerateLabelsPDF(labels)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate pdf"})
		return
	}

	fileName := "warehouse-labels.pdf"
	if objectType != "" {
		fileName = "warehouse-labels-" + objectType + ".pdf"
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

func (h *LabelHandler) listLabels(ctx context.Context, r *http.Request) ([]models.Label, error) {
	markerCodes := r.URL.Query()["marker_code"]
	objectType := r.URL.Query().Get("object_type")

	if len(markerCodes) > 0 {
		return h.useCase.ListSelected(ctx, objectType, markerCodes)
	}

	var limit int32
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			return nil, service.ErrInvalidLimit
		}
		limit = int32(parsedLimit)
	}

	return h.useCase.List(ctx, objectType, limit)
}
