package service

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"

	"github.com/xuri/excelize/v2"
)

type productImportRow struct {
	Row             int
	SKU             string
	Name            string
	Unit            string
	Quantity        int32
	BoxCode         string
	StorageCellCode string
	ValidationError string
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
		if row.ValidationError != "" {
			result.SkippedCount++
			result.Errors = append(result.Errors, models.ProductImportRowError{
				Row:   row.Row,
				SKU:   row.SKU,
				Name:  row.Name,
				Error: row.ValidationError,
			})
			continue
		}

		_, _, createErr := s.CreateProduct(ctx, CreateProductInput{
			SKU:             row.SKU,
			Name:            row.Name,
			Unit:            row.Unit,
			InitialQuantity: row.Quantity,
			BoxCode:         row.BoxCode,
			StorageCellCode: row.StorageCellCode,
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

	indexes, err := resolveProductImportHeaderIndexes(rawRows[0])
	if err != nil {
		return nil, err
	}

	rows := make([]productImportRow, 0, len(rawRows)-1)
	for index, rawRow := range rawRows[1:] {
		row := productImportRow{
			Row:             index + 2,
			SKU:             strings.TrimSpace(cellValue(rawRow, indexes.sku)),
			Name:            strings.TrimSpace(cellValue(rawRow, indexes.name)),
			Unit:            strings.TrimSpace(cellValue(rawRow, indexes.unit)),
			BoxCode:         strings.TrimSpace(cellValue(rawRow, indexes.boxCode)),
			StorageCellCode: strings.TrimSpace(cellValue(rawRow, indexes.storageCellCode)),
		}

		quantityValue := strings.TrimSpace(cellValue(rawRow, indexes.quantity))
		if row.SKU == "" && row.Name == "" && row.Unit == "" && quantityValue == "" && row.BoxCode == "" && row.StorageCellCode == "" {
			continue
		}

		if quantityValue == "" {
			row.ValidationError = "Количество обязательно и должно быть больше 0"
		} else {
			quantity, parseErr := strconv.Atoi(quantityValue)
			if parseErr != nil || quantity <= 0 {
				row.ValidationError = "Количество обязательно и должно быть больше 0"
			} else {
				row.Quantity = int32(quantity)
			}
		}

		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return nil, ErrEmptyAdminImport
	}

	return rows, nil
}

type productImportHeaderIndexes struct {
	sku             int
	name            int
	unit            int
	quantity        int
	boxCode         int
	storageCellCode int
}

func resolveProductImportHeaderIndexes(header []string) (productImportHeaderIndexes, error) {
	indexes := productImportHeaderIndexes{
		sku:             -1,
		name:            -1,
		unit:            -1,
		quantity:        -1,
		boxCode:         -1,
		storageCellCode: -1,
	}

	for index, rawValue := range header {
		switch normalizeProductImportHeader(rawValue) {
		case "sku", "артикул", "кодтовара":
			if indexes.sku < 0 {
				indexes.sku = index
			}
		case "name", "название", "наименование", "товар":
			if indexes.name < 0 {
				indexes.name = index
			}
		case "unit", "ед", "едизм", "единица", "единицаизмерения":
			if indexes.unit < 0 {
				indexes.unit = index
			}
		case "quantity", "количество", "колво", "остаток":
			if indexes.quantity < 0 {
				indexes.quantity = index
			}
		case "box", "короб", "кодкороба", "boxcode":
			if indexes.boxCode < 0 {
				indexes.boxCode = index
			}
		case "storagecell", "ячейка", "кодячейки", "cell", "cellcode":
			if indexes.storageCellCode < 0 {
				indexes.storageCellCode = index
			}
		}
	}

	if indexes.sku < 0 || indexes.name < 0 || indexes.quantity < 0 {
		return productImportHeaderIndexes{}, ErrInvalidAdminImport
	}
	if indexes.boxCode < 0 {
		return productImportHeaderIndexes{}, ErrInvalidAdminImport
	}

	return indexes, nil
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
		return "Проверьте SKU, название и начальное количество"
	case errors.Is(err, ErrAdminBoxRequired), errors.Is(err, ErrConflictingBatchTarget):
		return "Для товара укажите короб. Размещение напрямую в ячейку запрещено"
	case errors.Is(err, ErrInvalidAdminReference):
		return "Указанный короб или ячейка не найдены"
	case errors.Is(err, ErrAdminTargetOccupied):
		return "Выбранная ячейка уже занята"
	case errors.Is(err, ErrMixedBoxProducts):
		return "Короб для нового товара должен быть пустым"
	case errors.Is(err, ErrStorageCellProductConflict):
		return "В выбранной ячейке уже лежит другой товар"
	case errors.Is(err, ErrAdminProductExists):
		return "Товар с таким названием уже существует"
	case errors.Is(err, repository.ErrConflict), errors.Is(err, ErrAdminConflict):
		return "Товар с таким SKU уже существует"
	default:
		return "Не удалось импортировать строку"
	}
}
