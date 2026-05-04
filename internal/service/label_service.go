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
		columns     = 3
		rows        = 3
		marginX     = 10.0
		marginY     = 10.0
		gapX        = 4.0
		gapY        = 4.0
		qrSize      = 28.0
		topOffset   = 10.0
		titleOffset = 44.0
		nameOffset  = 56.0
		markerY     = 74.0
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

		pdf.SetXY(x+8, y+topOffset)
		pdf.SetFont("dejavu", "B", 8)
		pdf.SetTextColor(107, 114, 128)
		pdf.CellFormat(cardW-16, 4, strings.ToUpper(label.ObjectType), "", 0, "L", false, 0, "")

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
		pdf.ImageOptions(imageID, x+(cardW-qrSize)/2, y+16, qrSize, qrSize, false, options, 0, "")

		pdf.SetXY(x+8, y+titleOffset)
		pdf.SetFont("dejavu", "B", 16)
		pdf.SetTextColor(31, 41, 55)
		pdf.CellFormat(cardW-16, 8, label.Code, "", 0, "L", false, 0, "")

		pdf.SetXY(x+8, y+nameOffset)
		pdf.SetFont("dejavu", "", 11)
		pdf.SetTextColor(55, 65, 81)
		pdf.MultiCell(cardW-16, 5, label.Name, "", "L", false)

		pdf.SetXY(x+8, y+markerY)
		pdf.SetFont("dejavu", "", 9)
		pdf.SetTextColor(107, 114, 128)
		pdf.MultiCell(cardW-16, 4.5, label.MarkerCode, "", "L", false)
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
		})
	}

	return labels, nil
}

type labelObjectData struct {
	Code string
	Name string
}

func (s *LabelService) preloadLabelResolver(ctx context.Context, markers []models.Marker) (map[string]map[int64]labelObjectData, error) {
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

	resolver := map[string]map[int64]labelObjectData{
		"rack":         {},
		"storage_cell": {},
		"box":          {},
		"product":      {},
		"batch":        {},
	}

	if len(idsByType["rack"]) > 0 {
		racks, err := s.rackRepo.ListByIDs(ctx, idsByType["rack"])
		if err != nil {
			return nil, err
		}
		for _, rack := range racks {
			resolver["rack"][rack.ID] = labelObjectData{Code: rack.Code, Name: rack.Name}
		}
	}

	if len(idsByType["storage_cell"]) > 0 {
		cells, err := s.storageCellRepo.ListByIDs(ctx, idsByType["storage_cell"])
		if err != nil {
			return nil, err
		}
		for _, cell := range cells {
			resolver["storage_cell"][cell.ID] = labelObjectData{Code: cell.Code, Name: cell.Name}
		}
	}

	if len(idsByType["box"]) > 0 {
		boxes, err := s.boxRepo.ListByIDs(ctx, idsByType["box"])
		if err != nil {
			return nil, err
		}
		for _, box := range boxes {
			resolver["box"][box.ID] = labelObjectData{Code: box.Code, Name: box.Code}
		}
	}

	if len(idsByType["product"]) > 0 {
		products, err := s.productRepo.ListByIDs(ctx, idsByType["product"])
		if err != nil {
			return nil, err
		}
		for _, product := range products {
			resolver["product"][product.ID] = labelObjectData{Code: product.SKU, Name: product.Name}
		}
	}

	if len(idsByType["batch"]) > 0 {
		batches, err := s.batchRepo.ListByIDs(ctx, idsByType["batch"])
		if err != nil {
			return nil, err
		}
		for _, batch := range batches {
			resolver["batch"][batch.ID] = labelObjectData{Code: batch.Code, Name: batch.Code}
		}
	}

	return resolver, nil
}

func isSupportedLabelObjectType(value string) bool {
	switch value {
	case "rack", "storage_cell", "box", "product", "batch":
		return true
	default:
		return false
	}
}
