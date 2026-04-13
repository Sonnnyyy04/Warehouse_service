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
	ListSelected(ctx context.Context, objectType string, markerCodes []string) ([]models.Label, error)
	GenerateQRCodePNG(markerCode string, size int) ([]byte, error)
	GenerateLabelsPDF(labels []models.Label) ([]byte, error)
}

type LabelHandler struct {
	useCase LabelUseCase
}

type labelAdminOption struct {
	Value string
	Label string
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

	objectType := r.URL.Query().Get("object_type")
	labels, err := h.listLabels(ctx, r)
	if err != nil {
		http.Error(w, "failed to load labels", http.StatusInternalServerError)
		return
	}

	data := struct {
		ObjectType string
		Labels     []models.Label
		MarkerCodes []string
		AccessToken string
	}{
		ObjectType: objectType,
		Labels:     labels,
		MarkerCodes: r.URL.Query()["marker_code"],
		AccessToken: r.URL.Query().Get("access_token"),
	}

	tpl := template.Must(template.New("labels").Parse(printLabelsTemplate))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.Execute(w, data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *LabelHandler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
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

func (h *LabelHandler) AdminPage(w http.ResponseWriter, r *http.Request) {
	objectType := r.URL.Query().Get("object_type")
	if objectType == "" {
		objectType = "box"
	}

	limit := 100
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsedLimit, err := strconv.Atoi(rawLimit); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	data := struct {
		SelectedType string
		Limit        int
		Options      []labelAdminOption
	}{
		SelectedType: objectType,
		Limit:        limit,
		Options: []labelAdminOption{
			{Value: "box", Label: "Короба"},
			{Value: "storage_cell", Label: "Ячейки"},
			{Value: "pallet", Label: "Паллеты"},
			{Value: "batch", Label: "Партии"},
			{Value: "product", Label: "Товары"},
		},
	}

	tpl := template.Must(template.New("labels-admin").Parse(adminLabelsTemplate))

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
      text-decoration: none;
      display: inline-flex;
      align-items: center;
      justify-content: center;
    }
    .secondary-button {
      background: #e7f3f1;
      color: var(--accent);
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
      <div style="display:flex; gap:12px; flex-wrap:wrap;">
        <button class="print-button" onclick="window.print()">Печать</button>
        <a class="print-button secondary-button" href="/labels/pdf?object_type={{.ObjectType}}&limit={{len .Labels}}">Скачать PDF</a>
      </div>
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

const adminLabelsTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Warehouse Admin Labels</title>
  <style>
    :root {
      color-scheme: light;
      --ink: #172033;
      --muted: #637083;
      --paper: #ffffff;
      --line: #d8dde6;
      --accent: #0f766e;
      --accent-dark: #115e59;
      --bg: linear-gradient(180deg, #f2f0ea 0%, #eef6f4 100%);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: var(--bg);
      color: var(--ink);
      font-family: Arial, sans-serif;
    }
    .page {
      max-width: 980px;
      margin: 0 auto;
      padding: 32px 20px 48px;
    }
    .hero {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 28px;
      padding: 28px;
      box-shadow: 0 18px 40px rgba(23, 32, 51, 0.08);
    }
    h1 {
      margin: 0 0 10px;
      font-size: 32px;
    }
    .subtitle {
      margin: 0;
      color: var(--muted);
      line-height: 1.5;
      max-width: 720px;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 18px;
      margin-top: 22px;
    }
    .panel {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: 24px;
    }
    .panel h2 {
      margin: 0 0 14px;
      font-size: 22px;
    }
    label {
      display: block;
      margin-bottom: 8px;
      font-size: 14px;
      font-weight: 700;
    }
    select, input {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 14px;
      padding: 12px 14px;
      font: inherit;
      margin-bottom: 14px;
      background: white;
    }
    .actions {
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
      margin-top: 8px;
    }
    .button, .button-secondary {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-height: 46px;
      padding: 0 18px;
      border-radius: 14px;
      text-decoration: none;
      font-weight: 700;
      cursor: pointer;
    }
    .button {
      border: none;
      background: var(--accent);
      color: white;
    }
    .button:hover {
      background: var(--accent-dark);
    }
    .button-secondary {
      border: 1px solid var(--line);
      color: var(--ink);
      background: #f8fafc;
    }
    .button-secondary:hover {
      background: #eef2f7;
    }
    .links {
      display: grid;
      gap: 10px;
    }
    .links a {
      color: var(--accent-dark);
      text-decoration: none;
      font-weight: 700;
    }
    .links a:hover {
      text-decoration: underline;
    }
    .hint {
      margin-top: 14px;
      color: var(--muted);
      font-size: 14px;
      line-height: 1.5;
    }
  </style>
</head>
<body>
  <main class="page">
    <section class="hero">
      <h1>Печать QR-наклеек</h1>
      <p class="subtitle">Администратор открывает эту страницу, выбирает тип складских объектов и получает готовую страницу печати с QR-кодами для коробов, ячеек, паллет и других сущностей.</p>
    </section>

    <section class="grid">
      <article class="panel">
        <h2>Открыть печать</h2>
        <form action="/labels/print" method="get" target="_blank">
          <label for="object_type">Тип объекта</label>
          <select id="object_type" name="object_type">
            {{range .Options}}
            <option value="{{.Value}}" {{if eq $.SelectedType .Value}}selected{{end}}>{{.Label}}</option>
            {{end}}
          </select>

          <label for="limit">Сколько наклеек загрузить</label>
          <input id="limit" type="number" min="1" max="200" name="limit" value="{{.Limit}}" />

          <div class="actions">
            <button class="button" type="submit">Открыть страницу печати</button>
            <a class="button-secondary" href="/labels/pdf?object_type={{.SelectedType}}&limit={{.Limit}}" target="_blank" rel="noreferrer">Скачать PDF</a>
            <a class="button-secondary" href="/swagger/" target="_blank" rel="noreferrer">Swagger</a>
          </div>
        </form>
        <p class="hint">После открытия страницы нажмите «Печать» в самом интерфейсе печати или используйте печать браузера.</p>
      </article>

      <article class="panel">
        <h2>Быстрые ссылки</h2>
        <div class="links">
          <a href="/labels/print?object_type=box&limit=100" target="_blank" rel="noreferrer">Печать коробов</a>
          <a href="/labels/pdf?object_type=box&limit=100" target="_blank" rel="noreferrer">PDF коробов</a>
          <a href="/labels/print?object_type=storage_cell&limit=100" target="_blank" rel="noreferrer">Печать ячеек</a>
          <a href="/labels/pdf?object_type=storage_cell&limit=100" target="_blank" rel="noreferrer">PDF ячеек</a>
          <a href="/labels/print?object_type=pallet&limit=100" target="_blank" rel="noreferrer">Печать паллет</a>
          <a href="/labels/pdf?object_type=pallet&limit=100" target="_blank" rel="noreferrer">PDF паллет</a>
          <a href="/labels/print?object_type=batch&limit=100" target="_blank" rel="noreferrer">Печать партий</a>
          <a href="/labels/pdf?object_type=batch&limit=100" target="_blank" rel="noreferrer">PDF партий</a>
          <a href="/labels/print?object_type=product&limit=100" target="_blank" rel="noreferrer">Печать товаров</a>
          <a href="/labels/pdf?object_type=product&limit=100" target="_blank" rel="noreferrer">PDF товаров</a>
        </div>
        <p class="hint">Эту страницу удобно сохранить в закладки и использовать как простую админку для расклейки QR-кодов на складе.</p>
      </article>
    </section>
  </main>
</body>
</html>`
