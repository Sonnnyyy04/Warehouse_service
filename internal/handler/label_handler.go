package handler

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/service"
)

type LabelUseCase interface {
	List(ctx context.Context, objectType string, limit int32) ([]models.Label, error)
	GenerateQRCodePNG(markerCode string, size int) ([]byte, error)
}

type LabelHandler struct {
	useCase LabelUseCase
}

func NewLabelHandler(useCase LabelUseCase) *LabelHandler {
	return &LabelHandler{useCase: useCase}
}

// List godoc
// @Summary Получить список наклеек
// @Description Возвращает список warehouse-объектов с marker_code для генерации и печати QR наклеек.
// @Tags Наклейки
// @Produce json
// @Param object_type query string false "Тип объекта: storage_cell, pallet, box, product, batch"
// @Param limit query int false "Максимальное число записей; по умолчанию 50, максимум 200"
// @Success 200 {array} models.Label "Список наклеек"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/labels [get]
func (h *LabelHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var limit int32
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		limit = int32(parsedLimit)
	}

	labels, err := h.useCase.List(ctx, r.URL.Query().Get("object_type"), limit)
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
// @Summary Получить QR-код маркера
// @Description Генерирует PNG QR-код для указанного marker_code.
// @Tags Наклейки
// @Produce png
// @Param marker_code query string true "Код маркера"
// @Param size query int false "Размер PNG; по умолчанию 256"
// @Success 200 {file} binary "PNG QR-код"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
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

func (h *LabelHandler) PrintPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var limit int32
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err == nil {
			limit = int32(parsedLimit)
		}
	}

	objectType := r.URL.Query().Get("object_type")
	labels, err := h.useCase.List(ctx, objectType, limit)
	if err != nil {
		http.Error(w, "failed to load labels", http.StatusInternalServerError)
		return
	}

	data := struct {
		ObjectType string
		Labels     []models.Label
	}{
		ObjectType: objectType,
		Labels:     labels,
	}

	tpl := template.Must(template.New("labels").Parse(printLabelsTemplate))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.Execute(w, data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

const printLabelsTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Warehouse Labels</title>
  <style>
    :root {
      color-scheme: light;
      --ink: #1f2937;
      --muted: #6b7280;
      --paper: #ffffff;
      --line: #d1d5db;
      --accent: #0f766e;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: #f5f3ef;
      color: var(--ink);
      font-family: Arial, sans-serif;
    }
    .page {
      max-width: 1200px;
      margin: 0 auto;
      padding: 24px;
    }
    .toolbar {
      display: flex;
      gap: 12px;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 20px;
      flex-wrap: wrap;
    }
    .toolbar h1 {
      margin: 0;
      font-size: 28px;
    }
    .toolbar p {
      margin: 6px 0 0;
      color: var(--muted);
    }
    .print-button {
      border: none;
      border-radius: 12px;
      background: var(--accent);
      color: white;
      padding: 12px 18px;
      font-weight: 700;
      cursor: pointer;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
      gap: 16px;
    }
    .label {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: 18px;
      display: flex;
      flex-direction: column;
      gap: 12px;
      page-break-inside: avoid;
      break-inside: avoid;
    }
    .label img {
      width: 100%;
      max-width: 180px;
      align-self: center;
    }
    .eyebrow {
      font-size: 12px;
      font-weight: 700;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }
    .code {
      font-size: 22px;
      font-weight: 700;
      margin: 0;
    }
    .name, .marker {
      margin: 0;
      word-break: break-word;
    }
    .marker {
      font-size: 13px;
      color: var(--muted);
    }
    @media print {
      body { background: white; }
      .page { max-width: none; padding: 0; }
      .toolbar { display: none; }
      .grid { grid-template-columns: repeat(3, 1fr); gap: 10px; }
      .label { border-radius: 0; }
    }
  </style>
</head>
<body>
  <div class="page">
    <div class="toolbar">
      <div>
        <h1>Печать QR-наклеек</h1>
        <p>Тип: {{if .ObjectType}}{{.ObjectType}}{{else}}all{{end}} | Количество: {{len .Labels}}</p>
      </div>
      <button class="print-button" onclick="window.print()">Печать</button>
    </div>
    <div class="grid">
      {{range .Labels}}
      <article class="label">
        <div class="eyebrow">{{.ObjectType}}</div>
        <img src="/api/v1/labels/qr?marker_code={{.MarkerCode}}&size=256" alt="{{.MarkerCode}}" />
        <p class="code">{{.Code}}</p>
        <p class="name">{{.Name}}</p>
        <p class="marker">{{.MarkerCode}}</p>
      </article>
      {{end}}
    </div>
  </div>
</body>
</html>`
