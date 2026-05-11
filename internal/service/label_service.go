package service

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"Warehouse_service/internal/models"

	"github.com/phpdave11/gofpdf"
	"github.com/skip2/go-qrcode"
)

var (
	ErrInvalidLabelObjectType = errors.New("invalid label object type")
	ErrInvalidLabelMarkerCode = errors.New("invalid label marker code")
)

//go:embed fonts/DejaVuSansCondensed.ttf
var labelFontRegular []byte

//go:embed fonts/DejaVuSansCondensed-Bold.ttf
var labelFontBold []byte

type LabelMarkerRepository interface {
	List(ctx context.Context, objectType string, limit int32) ([]models.Marker, error)
	ListByCodes(ctx context.Context, objectType string, markerCodes []string) ([]models.Marker, error)
}

type LabelStorageCellRepository interface {
	GetByID(ctx context.Context, id int64) (models.StorageCell, error)
	ListByIDs(ctx context.Context, ids []int64) ([]models.StorageCell, error)
}

type LabelRackRepository interface {
	GetByID(ctx context.Context, id int64) (models.Rack, error)
	ListByIDs(ctx context.Context, ids []int64) ([]models.Rack, error)
}

type LabelBoxRepository interface {
	GetByID(ctx context.Context, id int64) (models.Box, error)
	ListByIDs(ctx context.Context, ids []int64) ([]models.Box, error)
}

type LabelProductRepository interface {
	GetByID(ctx context.Context, id int64) (models.Product, error)
	ListByIDs(ctx context.Context, ids []int64) ([]models.Product, error)
}

type LabelBatchRepository interface {
	GetByID(ctx context.Context, id int64) (models.Batch, error)
	ListByIDs(ctx context.Context, ids []int64) ([]models.Batch, error)
	ListActiveByBoxIDs(ctx context.Context, boxIDs []int64) ([]models.Batch, error)
}

type LabelService struct {
	markerRepo      LabelMarkerRepository
	rackRepo        LabelRackRepository
	storageCellRepo LabelStorageCellRepository
	boxRepo         LabelBoxRepository
	productRepo     LabelProductRepository
	batchRepo       LabelBatchRepository
}

func NewLabelService(
	markerRepo LabelMarkerRepository,
	rackRepo LabelRackRepository,
	storageCellRepo LabelStorageCellRepository,
	boxRepo LabelBoxRepository,
	productRepo LabelProductRepository,
	batchRepo LabelBatchRepository,
) *LabelService {
	return &LabelService{
		markerRepo:      markerRepo,
		rackRepo:        rackRepo,
		storageCellRepo: storageCellRepo,
		boxRepo:         boxRepo,
		productRepo:     productRepo,
		batchRepo:       batchRepo,
	}
}

func (s *LabelService) List(ctx context.Context, objectType string, limit int32) ([]models.Label, error) {
	objectType = strings.TrimSpace(objectType)
	if objectType != "" && !isSupportedLabelObjectType(objectType) {
		return nil, ErrInvalidLabelObjectType
	}

	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	markers, err := s.markerRepo.List(ctx, objectType, normalizedLimit)
	if err != nil {
		return nil, err
	}

	return s.buildLabels(ctx, markers)
}

func (s *LabelService) ListSelected(ctx context.Context, objectType string, markerCodes []string) ([]models.Label, error) {
	objectType = strings.TrimSpace(objectType)
	if objectType != "" && !isSupportedLabelObjectType(objectType) {
		return nil, ErrInvalidLabelObjectType
	}

	if len(markerCodes) == 0 {
		return []models.Label{}, nil
	}

	markers, err := s.markerRepo.ListByCodes(ctx, objectType, markerCodes)
	if err != nil {
		return nil, err
	}

	return s.buildLabels(ctx, markers)
}

func (s *LabelService) GenerateQRCodePNG(markerCode string, size int) ([]byte, error) {
	markerCode = strings.TrimSpace(markerCode)
	if markerCode == "" {
		return nil, ErrInvalidLabelMarkerCode
	}

	if size <= 0 {
		size = 256
	}

	if size > 1024 {
		size = 1024
	}

	return qrcode.Encode(markerCode, qrcode.Medium, size)
}

func (s *LabelService) GenerateLabelsPDF(labels []models.Label) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Warehouse Labels", false)
	pdf.SetAuthor("Warehouse Service", false)
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(false, 10)
	pdf.AddUTF8FontFromBytes("dejavu", "", bytes.Clone(labelFontRegular))
	pdf.AddUTF8FontFromBytes("dejavu", "B", bytes.Clone(labelFontBold))

	const (
		columns  = 3
		rows     = 3
		marginX  = 10.0
		marginY  = 10.0
		gapX     = 4.0
		gapY     = 4.0
		qrSize   = 38.0
		typeY    = 5.0
		qrY      = 12.0
		codeY    = 53.0
		detailsY = 65.0
		markerY  = 84.0
	)

	pageW, pageH := 210.0, 297.0
	cardW := (pageW - (2 * marginX) - (gapX * float64(columns-1))) / float64(columns)
	cardH := (pageH - (2 * marginY) - (gapY * float64(rows-1))) / float64(rows)

	for index, label := range labels {
		if index%(columns*rows) == 0 {
			pdf.AddPage()
		}

		position := index % (columns * rows)
		col := position % columns
		row := position / columns

		x := marginX + float64(col)*(cardW+gapX)
		y := marginY + float64(row)*(cardH+gapY)

		pdf.SetDrawColor(209, 213, 219)
		pdf.SetFillColor(255, 255, 255)
		pdf.RoundedRect(x, y, cardW, cardH, 4, "1234", "DF")

		pdf.SetXY(x+6, y+typeY)
		pdf.SetFont("dejavu", "B", 8)
		pdf.SetTextColor(107, 114, 128)
		pdf.CellFormat(cardW-12, 4, labelObjectTypeTitle(label.ObjectType), "", 0, "C", false, 0, "")

		qrBytes, err := s.GenerateQRCodePNG(label.MarkerCode, 256)
		if err != nil {
			return nil, err
		}

		imageID := fmt.Sprintf("label-qr-%d", index)
		options := gofpdf.ImageOptions{
			ImageType: "PNG",
			ReadDpi:   true,
		}
		pdf.RegisterImageOptionsReader(imageID, options, bytes.NewReader(qrBytes))
		pdf.ImageOptions(imageID, x+(cardW-qrSize)/2, y+qrY, qrSize, qrSize, false, options, 0, "")

		drawLabelCode(pdf, label.Code, x+6, y+codeY, cardW-12)

		pdf.SetXY(x+7, y+detailsY)
		pdf.SetFont("dejavu", "", 8.5)
		pdf.SetTextColor(55, 65, 81)
		pdf.MultiCell(cardW-14, 4.2, strings.Join(compactLabelDetails(label), "\n"), "", "L", false)

		pdf.SetXY(x+7, y+markerY)
		pdf.SetFont("dejavu", "", 7.5)
		pdf.SetTextColor(107, 114, 128)
		pdf.CellFormat(cardW-14, 4, label.MarkerCode, "", 0, "C", false, 0, "")
	}

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

func (s *LabelService) buildLabel(ctx context.Context, marker models.Marker) (models.Label, error) {
	switch marker.ObjectType {
	case "rack":
		rack, err := s.rackRepo.GetByID(ctx, marker.ObjectID)
		if err != nil {
			return models.Label{}, err
		}

		return models.Label{
			MarkerCode: marker.MarkerCode,
			ObjectType: marker.ObjectType,
			ObjectID:   marker.ObjectID,
			Code:       rack.Code,
			Name:       rack.Name,
			Details:    rackLabelDetails(rack),
		}, nil
	case "storage_cell":
		cell, err := s.storageCellRepo.GetByID(ctx, marker.ObjectID)
		if err != nil {
			return models.Label{}, err
		}

		return models.Label{
			MarkerCode: marker.MarkerCode,
			ObjectType: marker.ObjectType,
			ObjectID:   marker.ObjectID,
			Code:       cell.Code,
			Name:       cell.Name,
			Details:    storageCellLabelDetails(cell),
		}, nil
	case "box":
		box, err := s.boxRepo.GetByID(ctx, marker.ObjectID)
		if err != nil {
			return models.Label{}, err
		}

		return models.Label{
			MarkerCode: marker.MarkerCode,
			ObjectType: marker.ObjectType,
			ObjectID:   marker.ObjectID,
			Code:       box.Code,
			Name:       box.Code,
			Details:    boxLabelDetails(box, nil, nil, nil),
		}, nil
	case "product":
		product, err := s.productRepo.GetByID(ctx, marker.ObjectID)
		if err != nil {
			return models.Label{}, err
		}

		return models.Label{
			MarkerCode: marker.MarkerCode,
			ObjectType: marker.ObjectType,
			ObjectID:   marker.ObjectID,
			Code:       product.SKU,
			Name:       product.Name,
			Details:    productLabelDetails(product),
		}, nil
	case "batch":
		batch, err := s.batchRepo.GetByID(ctx, marker.ObjectID)
		if err != nil {
			return models.Label{}, err
		}

		return models.Label{
			MarkerCode: marker.MarkerCode,
			ObjectType: marker.ObjectType,
			ObjectID:   marker.ObjectID,
			Code:       batch.Code,
			Name:       batch.Code,
			Details:    batchLabelDetails(batch, nil, nil),
		}, nil
	default:
		return models.Label{}, ErrInvalidLabelObjectType
	}
}

func (s *LabelService) buildLabels(ctx context.Context, markers []models.Marker) ([]models.Label, error) {
	resolver, err := s.preloadLabelResolver(ctx, markers)
	if err != nil {
		return nil, err
	}

	labels := make([]models.Label, 0, len(markers))

	for _, marker := range markers {
		label, ok := resolver[marker.ObjectType][marker.ObjectID]
		if !ok {
			continue
		}

		labels = append(labels, models.Label{
			MarkerCode: marker.MarkerCode,
			ObjectType: marker.ObjectType,
			ObjectID:   marker.ObjectID,
			Code:       label.Code,
			Name:       label.Name,
			Details:    label.Details,
		})
	}

	return labels, nil
}

type labelObjectData struct {
	Code    string
	Name    string
	Details []string
}

func (s *LabelService) preloadLabelResolver(ctx context.Context, markers []models.Marker) (map[string]map[int64]labelObjectData, error) {
	resolver := map[string]map[int64]labelObjectData{
		"rack":         {},
		"storage_cell": {},
		"box":          {},
		"product":      {},
		"batch":        {},
	}

	idsByType := labelIDsByType(markers)
	productIDs := make([]int64, 0)
	cellIDs := make([]int64, 0)
	boxIDs := make([]int64, 0)

	rackByID := make(map[int64]models.Rack)
	cellByID := make(map[int64]models.StorageCell)
	boxByID := make(map[int64]models.Box)
	batchByID := make(map[int64]models.Batch)
	productByID := make(map[int64]models.Product)
	batchesByBoxID := make(map[int64][]models.Batch)

	if len(idsByType["rack"]) > 0 {
		racks, err := s.rackRepo.ListByIDs(ctx, idsByType["rack"])
		if err != nil {
			return nil, err
		}
		for _, rack := range racks {
			rackByID[rack.ID] = rack
		}
	}

	if len(idsByType["storage_cell"]) > 0 {
		cells, err := s.storageCellRepo.ListByIDs(ctx, idsByType["storage_cell"])
		if err != nil {
			return nil, err
		}
		for _, cell := range cells {
			cellByID[cell.ID] = cell
		}
	}

	if len(idsByType["box"]) > 0 {
		boxes, err := s.boxRepo.ListByIDs(ctx, idsByType["box"])
		if err != nil {
			return nil, err
		}
		for _, box := range boxes {
			if box.Status != "active" {
				continue
			}
			boxByID[box.ID] = box
			boxIDs = appendUniqueInt64(boxIDs, box.ID)
			if box.StorageCellID != nil {
				cellIDs = appendUniqueInt64(cellIDs, *box.StorageCellID)
			}
		}
	}

	if len(idsByType["product"]) > 0 {
		productIDs = appendUniqueInt64s(productIDs, idsByType["product"])
	}

	if len(idsByType["batch"]) > 0 {
		batches, err := s.batchRepo.ListByIDs(ctx, idsByType["batch"])
		if err != nil {
			return nil, err
		}
		for _, batch := range batches {
			batchByID[batch.ID] = batch
			productIDs = appendUniqueInt64(productIDs, batch.ProductID)
			if batch.BoxID != nil {
				boxIDs = appendUniqueInt64(boxIDs, *batch.BoxID)
			}
			if batch.StorageCellID != nil {
				cellIDs = appendUniqueInt64(cellIDs, *batch.StorageCellID)
			}
		}
	}

	if len(boxIDs) > 0 {
		for _, boxID := range boxIDs {
			if _, ok := boxByID[boxID]; ok {
				continue
			}
			boxes, err := s.boxRepo.ListByIDs(ctx, boxIDs)
			if err != nil {
				return nil, err
			}
			for _, box := range boxes {
				if box.Status != "active" {
					continue
				}
				boxByID[box.ID] = box
				if box.StorageCellID != nil {
					cellIDs = appendUniqueInt64(cellIDs, *box.StorageCellID)
				}
			}
			break
		}

		batches, err := s.batchRepo.ListActiveByBoxIDs(ctx, boxIDs)
		if err != nil {
			return nil, err
		}
		for _, batch := range batches {
			if batch.BoxID == nil {
				continue
			}
			batchesByBoxID[*batch.BoxID] = append(batchesByBoxID[*batch.BoxID], batch)
			productIDs = appendUniqueInt64(productIDs, batch.ProductID)
		}
	}

	if len(cellIDs) > 0 {
		for _, cellID := range cellIDs {
			if _, ok := cellByID[cellID]; ok {
				continue
			}
			cells, err := s.storageCellRepo.ListByIDs(ctx, cellIDs)
			if err != nil {
				return nil, err
			}
			for _, cell := range cells {
				cellByID[cell.ID] = cell
			}
			break
		}
	}

	if len(productIDs) > 0 {
		products, err := s.productRepo.ListByIDs(ctx, productIDs)
		if err != nil {
			return nil, err
		}
		for _, product := range products {
			productByID[product.ID] = product
		}
	}

	for _, rack := range rackByID {
		resolver["rack"][rack.ID] = labelObjectData{
			Code:    rack.Code,
			Name:    rack.Name,
			Details: rackLabelDetails(rack),
		}
	}

	for _, cell := range cellByID {
		if !containsInt64(idsByType["storage_cell"], cell.ID) {
			continue
		}
		resolver["storage_cell"][cell.ID] = labelObjectData{
			Code:    cell.Code,
			Name:    cell.Name,
			Details: storageCellLabelDetails(cell),
		}
	}

	for _, box := range boxByID {
		if !containsInt64(idsByType["box"], box.ID) {
			continue
		}
		resolver["box"][box.ID] = labelObjectData{
			Code:    box.Code,
			Name:    box.Code,
			Details: boxLabelDetails(box, batchesByBoxID[box.ID], productByID, cellByID),
		}
	}

	for _, product := range productByID {
		if !containsInt64(idsByType["product"], product.ID) {
			continue
		}
		resolver["product"][product.ID] = labelObjectData{
			Code:    product.SKU,
			Name:    product.Name,
			Details: productLabelDetails(product),
		}
	}

	for _, batch := range batchByID {
		var box *models.Box
		if batch.BoxID != nil {
			if value, ok := boxByID[*batch.BoxID]; ok {
				box = &value
			}
		}
		var product *models.Product
		if value, ok := productByID[batch.ProductID]; ok {
			product = &value
		}
		resolver["batch"][batch.ID] = labelObjectData{
			Code:    batch.Code,
			Name:    batch.Code,
			Details: batchLabelDetails(batch, product, box),
		}
	}

	return resolver, nil
}

func labelIDsByType(markers []models.Marker) map[string][]int64 {
	idsByType := make(map[string][]int64)
	seen := make(map[string]map[int64]struct{})

	for _, marker := range markers {
		if !isSupportedLabelObjectType(marker.ObjectType) {
			continue
		}
		if _, ok := seen[marker.ObjectType]; !ok {
			seen[marker.ObjectType] = make(map[int64]struct{})
		}
		if _, ok := seen[marker.ObjectType][marker.ObjectID]; ok {
			continue
		}
		seen[marker.ObjectType][marker.ObjectID] = struct{}{}
		idsByType[marker.ObjectType] = append(idsByType[marker.ObjectType], marker.ObjectID)
	}

	return idsByType
}

func labelObjectTypeTitle(objectType string) string {
	switch objectType {
	case "rack":
		return "СТЕЛЛАЖ"
	case "storage_cell":
		return "ЯЧЕЙКА"
	case "box":
		return "КОРОБ"
	case "product":
		return "ТОВАР"
	case "batch":
		return "ПАРТИЯ"
	default:
		return strings.ToUpper(objectType)
	}
}

func compactLabelDetails(label models.Label) []string {
	details := make([]string, 0, 3)
	for _, detail := range label.Details {
		detail = strings.TrimSpace(detail)
		if detail == "" {
			continue
		}
		details = append(details, detail)
		if len(details) == 3 {
			return details
		}
	}
	if len(details) == 0 && strings.TrimSpace(label.Name) != "" && label.Name != label.Code {
		details = append(details, label.Name)
	}
	return details
}

func drawLabelCode(pdf *gofpdf.Fpdf, code string, x, y, width float64) {
	code = strings.TrimSpace(code)
	if code == "" {
		return
	}

	pdf.SetTextColor(31, 41, 55)
	for _, fontSize := range []float64{15, 13, 11, 9} {
		pdf.SetFont("dejavu", "B", fontSize)
		if pdf.GetStringWidth(code) <= width {
			pdf.SetXY(x, y)
			pdf.CellFormat(width, 8, code, "", 0, "C", false, 0, "")
			return
		}
	}

	lines := splitLabelCode(code, 2)
	pdf.SetFont("dejavu", "B", 8.5)
	pdf.SetXY(x, y-1)
	pdf.MultiCell(width, 4.2, strings.Join(lines, "\n"), "", "C", false)
}

func splitLabelCode(code string, maxLines int) []string {
	if maxLines <= 1 {
		return []string{code}
	}

	parts := strings.Split(code, "-")
	if len(parts) <= 1 {
		return splitStringEvenly(code, maxLines)
	}

	lines := make([]string, 0, maxLines)
	current := parts[0]
	for _, part := range parts[1:] {
		candidate := current + "-" + part
		if len(lines) < maxLines-1 && len(candidate) > 18 {
			lines = append(lines, current+"-")
			current = part
			continue
		}
		current = candidate
	}
	lines = append(lines, current)

	if len(lines) > maxLines {
		return splitStringEvenly(code, maxLines)
	}
	return lines
}

func splitStringEvenly(value string, maxLines int) []string {
	if maxLines <= 1 || len(value) <= maxLines {
		return []string{value}
	}

	chunkSize := (len(value) + maxLines - 1) / maxLines
	lines := make([]string, 0, maxLines)
	for len(value) > 0 && len(lines) < maxLines {
		if len(value) <= chunkSize || len(lines) == maxLines-1 {
			lines = append(lines, value)
			break
		}
		lines = append(lines, value[:chunkSize])
		value = value[chunkSize:]
	}
	return lines
}

func rackLabelDetails(rack models.Rack) []string {
	details := make([]string, 0, 2)
	if strings.TrimSpace(rack.Name) != "" && rack.Name != rack.Code {
		details = append(details, rack.Name)
	}
	if rack.Zone != nil && strings.TrimSpace(*rack.Zone) != "" {
		details = append(details, "Зона: "+strings.TrimSpace(*rack.Zone))
	}
	return details
}

func storageCellLabelDetails(cell models.StorageCell) []string {
	details := make([]string, 0, 3)
	if cell.RackCode != nil && strings.TrimSpace(*cell.RackCode) != "" {
		details = append(details, "Стеллаж: "+strings.TrimSpace(*cell.RackCode))
	}
	if strings.TrimSpace(cell.Name) != "" && cell.Name != cell.Code {
		details = append(details, cell.Name)
	}
	if cell.Zone != nil && strings.TrimSpace(*cell.Zone) != "" {
		details = append(details, "Зона: "+strings.TrimSpace(*cell.Zone))
	}
	return details
}

func boxLabelDetails(
	box models.Box,
	batches []models.Batch,
	products map[int64]models.Product,
	cells map[int64]models.StorageCell,
) []string {
	details := make([]string, 0, 3)
	if len(batches) == 0 {
		if box.StorageCellID != nil {
			if cell, ok := cells[*box.StorageCellID]; ok {
				details = append(details, "Ячейка: "+cell.Code)
			}
		}
		return details
	}

	quantityByProduct := make(map[int64]int32)
	for _, batch := range batches {
		quantityByProduct[batch.ProductID] += batch.Quantity
	}

	if len(quantityByProduct) == 1 {
		for productID, quantity := range quantityByProduct {
			product := products[productID]
			if product.Name != "" {
				details = append(details, product.Name)
			}
			unit := product.Unit
			if unit == "" {
				unit = "pcs"
			}
			details = append(details, fmt.Sprintf("Кол-во: %d %s", quantity, unit))
		}
	} else {
		var total int32
		for _, quantity := range quantityByProduct {
			total += quantity
		}
		details = append(details, fmt.Sprintf("Товаров: %d, ед.: %d", len(quantityByProduct), total))
	}

	if box.StorageCellID != nil {
		if cell, ok := cells[*box.StorageCellID]; ok {
			details = append(details, "Ячейка: "+cell.Code)
		}
	}

	return details
}

func productLabelDetails(product models.Product) []string {
	details := make([]string, 0, 2)
	if strings.TrimSpace(product.Name) != "" {
		details = append(details, product.Name)
	}
	if strings.TrimSpace(product.Unit) != "" {
		details = append(details, "Ед.: "+product.Unit)
	}
	return details
}

func batchLabelDetails(batch models.Batch, product *models.Product, box *models.Box) []string {
	details := make([]string, 0, 3)
	unit := "pcs"
	if product != nil {
		if product.Name != "" {
			details = append(details, product.Name)
		}
		if product.Unit != "" {
			unit = product.Unit
		}
	}
	details = append(details, fmt.Sprintf("Кол-во: %d %s", batch.Quantity, unit))
	if box != nil {
		details = append(details, "Короб: "+box.Code)
	}
	return details
}

func appendUniqueInt64(values []int64, value int64) []int64 {
	if containsInt64(values, value) {
		return values
	}
	return append(values, value)
}

func appendUniqueInt64s(values []int64, extra []int64) []int64 {
	for _, value := range extra {
		values = appendUniqueInt64(values, value)
	}
	return values
}

func containsInt64(values []int64, target int64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func isSupportedLabelObjectType(value string) bool {
	switch value {
	case "rack", "storage_cell", "box", "product", "batch":
		return true
	default:
		return false
	}
}
