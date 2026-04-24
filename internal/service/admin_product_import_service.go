package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"unicode"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"

	"github.com/xuri/excelize/v2"
)

type productImportRow struct {
	Row  int
	SKU  string
	Name string
	Unit string
}

func (s *AdminService) ImportProducts(ctx context.Context, reader io.Reader) (models.ProductImportResult, error) {
	rows, err := parseProductImportRows(reader)
	if err != nil {
		return models.ProductImportResult{}, err
	}

	result := models.ProductImportResult{
		TotalRows: len(rows),
		Errors:    make([]models.ProductImportRowError, 0),
	}

	for _, row := range rows {
		_, _, createErr := s.CreateProduct(ctx, CreateProductInput{
			SKU:  row.SKU,
			Name: row.Name,
			Unit: row.Unit,
		})
		if createErr != nil {
			result.SkippedCount++
			result.Errors = append(result.Errors, models.ProductImportRowError{
				Row:   row.Row,
				SKU:   row.SKU,
				Name:  row.Name,
				Error: mapProductImportRowError(createErr),
			})
			continue
		}

		result.CreatedCount++
	}

	return result, nil
}

func parseProductImportRows(reader io.Reader) ([]productImportRow, error) {
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, ErrInvalidAdminImport
	}
	defer func() {
		_ = file.Close()
	}()

	sheetName := file.GetSheetName(0)
	if strings.TrimSpace(sheetName) == "" {
		return nil, ErrInvalidAdminImport
	}

	rawRows, err := file.GetRows(sheetName)
	if err != nil {
		return nil, ErrInvalidAdminImport
	}
	if len(rawRows) == 0 {
		return nil, ErrEmptyAdminImport
	}

	skuIndex, nameIndex, unitIndex, err := resolveProductImportHeaderIndexes(rawRows[0])
	if err != nil {
		return nil, err
	}

	rows := make([]productImportRow, 0, len(rawRows)-1)
	for index, rawRow := range rawRows[1:] {
		row := productImportRow{
			Row:  index + 2,
			SKU:  strings.TrimSpace(cellValue(rawRow, skuIndex)),
			Name: strings.TrimSpace(cellValue(rawRow, nameIndex)),
			Unit: strings.TrimSpace(cellValue(rawRow, unitIndex)),
		}

		if row.SKU == "" && row.Name == "" && row.Unit == "" {
			continue
		}

		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return nil, ErrEmptyAdminImport
	}

	return rows, nil
}

func resolveProductImportHeaderIndexes(header []string) (int, int, int, error) {
	skuIndex := -1
	nameIndex := -1
	unitIndex := -1

	for index, rawValue := range header {
		switch normalizeProductImportHeader(rawValue) {
		case "sku", "артикул", "кодтовара":
			if skuIndex < 0 {
				skuIndex = index
			}
		case "name", "название", "наименование", "товар":
			if nameIndex < 0 {
				nameIndex = index
			}
		case "unit", "ед", "едизм", "единица", "единицаизмерения":
			if unitIndex < 0 {
				unitIndex = index
			}
		}
	}

	if skuIndex < 0 || nameIndex < 0 {
		return 0, 0, 0, ErrInvalidAdminImport
	}

	return skuIndex, nameIndex, unitIndex, nil
}

func normalizeProductImportHeader(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return ""
	}

	var normalized strings.Builder
	normalized.Grow(len(trimmed))

	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			normalized.WriteRune(r)
		}
	}

	return normalized.String()
}

func cellValue(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}

	return row[index]
}

func mapProductImportRowError(err error) string {
	switch {
	case errors.Is(err, ErrInvalidAdminInput):
		return "Укажите SKU и название"
	case errors.Is(err, ErrAdminProductExists):
		return "Товар с таким названием уже существует"
	case errors.Is(err, repository.ErrConflict), errors.Is(err, ErrAdminConflict):
		return "Товар с таким SKU уже существует"
	default:
		return "Не удалось импортировать строку"
	}
}
