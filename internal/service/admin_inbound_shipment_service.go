package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"

	"github.com/xuri/excelize/v2"
)

type inboundShipmentImportRow struct {
	Row             int
	SupplierName    string
	SupplierArticle string
	ProductName     string
	Unit            string
	TotalQuantity   int32
	BoxesCount      int32
	QuantityPerBox  int32
	ValidationError string
}

func (s *AdminService) ListInboundShipments(ctx context.Context, limit int32) ([]models.InboundShipment, error) {
	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	return s.shipmentRepo.List(ctx, normalizedLimit)
}

func (s *AdminService) GetInboundShipment(ctx context.Context, id int64) (models.InboundShipment, []models.InboundShipmentBox, error) {
	if id <= 0 {
		return models.InboundShipment{}, nil, ErrInvalidAdminInput
	}

	shipment, err := s.shipmentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.InboundShipment{}, nil, ErrInvalidAdminReference
		}
		return models.InboundShipment{}, nil, err
	}

	items, err := s.shipmentRepo.ListItems(ctx, id)
	if err != nil {
		return models.InboundShipment{}, nil, err
	}
	boxes, err := s.shipmentRepo.ListBoxes(ctx, id)
	if err != nil {
		return models.InboundShipment{}, nil, err
	}

	shipment.Items = items
	return shipment, boxes, nil
}

func (s *AdminService) ImportInboundShipment(ctx context.Context, reader io.Reader) (models.InboundShipmentImportResult, error) {
	rows, err := parseInboundShipmentImportRows(reader)
	if err != nil {
		return models.InboundShipmentImportResult{}, err
	}

	supplierName := rows[0].SupplierName
	shipmentCode := fmt.Sprintf("SHIP-%s", time.Now().Format("20060102-150405"))

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.InboundShipmentImportResult{}, fmt.Errorf("begin import shipment tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	aliasRepo := repository.NewProductAliasRepositoryWithQuerier(tx)
	shipmentRepo := repository.NewInboundShipmentRepositoryWithQuerier(tx)

	shipment, err := shipmentRepo.Create(ctx, shipmentCode, supplierName)
	if err != nil {
		return models.InboundShipmentImportResult{}, err
	}

	for _, row := range rows {
		if row.ValidationError != "" {
			return models.InboundShipmentImportResult{}, ErrInvalidAdminImport
		}

		var productID *int64
		status := "unresolved"
		alias, err := aliasRepo.GetBySupplierArticle(ctx, row.SupplierName, row.SupplierArticle)
		if err == nil {
			productID = &alias.ProductID
			status = "matched"
		} else if !errors.Is(err, repository.ErrNotFound) {
			return models.InboundShipmentImportResult{}, err
		}

		item, err := shipmentRepo.CreateItem(ctx, models.InboundShipmentItem{
			ShipmentID:      shipment.ID,
			ProductID:       productID,
			SupplierArticle: row.SupplierArticle,
			ProductName:     row.ProductName,
			Unit:            row.Unit,
			TotalQuantity:   row.TotalQuantity,
			BoxesCount:      row.BoxesCount,
			QuantityPerBox:  row.QuantityPerBox,
			Status:          status,
		})
		if err != nil {
			return models.InboundShipmentImportResult{}, err
		}

		for _, plannedQuantity := range splitShipmentQuantities(row.TotalQuantity, row.BoxesCount, row.QuantityPerBox) {
			if _, err := shipmentRepo.CreatePlannedBox(ctx, item.ID, plannedQuantity); err != nil {
				return models.InboundShipmentImportResult{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return models.InboundShipmentImportResult{}, fmt.Errorf("commit import shipment tx: %w", err)
	}

	shipment, boxes, err := s.GetInboundShipment(ctx, shipment.ID)
	if err != nil {
		return models.InboundShipmentImportResult{}, err
	}
	_ = boxes

	return models.InboundShipmentImportResult{Shipment: shipment}, nil
}

func (s *AdminService) LinkInboundShipmentItem(ctx context.Context, input LinkShipmentItemInput) (models.InboundShipmentItem, error) {
	if input.ItemID <= 0 || input.ProductID <= 0 {
		return models.InboundShipmentItem{}, ErrInvalidAdminInput
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.InboundShipmentItem{}, fmt.Errorf("begin link shipment item tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	productRepo := repository.NewProductRepositoryWithQuerier(tx)
	aliasRepo := repository.NewProductAliasRepositoryWithQuerier(tx)
	shipmentRepo := repository.NewInboundShipmentRepositoryWithQuerier(tx)

	item, err := shipmentRepo.GetItemByID(ctx, input.ItemID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.InboundShipmentItem{}, ErrInvalidAdminReference
		}
		return models.InboundShipmentItem{}, err
	}
	shipment, err := shipmentRepo.GetByID(ctx, item.ShipmentID)
	if err != nil {
		return models.InboundShipmentItem{}, err
	}
	if _, err := productRepo.GetByID(ctx, input.ProductID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.InboundShipmentItem{}, ErrInvalidAdminReference
		}
		return models.InboundShipmentItem{}, err
	}

	if _, err := aliasRepo.UpsertSupplierArticle(ctx, input.ProductID, shipment.SupplierName, item.SupplierArticle); err != nil {
		return models.InboundShipmentItem{}, err
	}

	updated, err := shipmentRepo.UpdateItemProduct(ctx, input.ItemID, input.ProductID)
	if err != nil {
		return models.InboundShipmentItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.InboundShipmentItem{}, fmt.Errorf("commit link shipment item tx: %w", err)
	}

	return updated, nil
}

func (s *AdminService) CreateProductForInboundShipmentItem(ctx context.Context, input CreateProductForShipmentItemInput) (models.InboundShipmentItem, error) {
	sku := strings.TrimSpace(input.SKU)
	name := strings.TrimSpace(input.Name)
	unit := strings.TrimSpace(input.Unit)
	if unit == "" {
		unit = "pcs"
	}
	if input.ItemID <= 0 || sku == "" || name == "" {
		return models.InboundShipmentItem{}, ErrInvalidAdminInput
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.InboundShipmentItem{}, fmt.Errorf("begin create product for shipment item tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	productRepo := repository.NewProductRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)
	aliasRepo := repository.NewProductAliasRepositoryWithQuerier(tx)
	shipmentRepo := repository.NewInboundShipmentRepositoryWithQuerier(tx)

	item, err := shipmentRepo.GetItemByID(ctx, input.ItemID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.InboundShipmentItem{}, ErrInvalidAdminReference
		}
		return models.InboundShipmentItem{}, err
	}
	shipment, err := shipmentRepo.GetByID(ctx, item.ShipmentID)
	if err != nil {
		return models.InboundShipmentItem{}, err
	}

	product, err := productRepo.Create(ctx, sku, name, unit)
	if err != nil {
		return models.InboundShipmentItem{}, err
	}
	if _, err := markerRepo.Create(ctx, buildMarkerCode("product", product.ID), "product", product.ID); err != nil {
		return models.InboundShipmentItem{}, err
	}
	if _, err := aliasRepo.UpsertSupplierArticle(ctx, product.ID, shipment.SupplierName, item.SupplierArticle); err != nil {
		return models.InboundShipmentItem{}, err
	}

	updated, err := shipmentRepo.UpdateItemProduct(ctx, input.ItemID, product.ID)
	if err != nil {
		return models.InboundShipmentItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.InboundShipmentItem{}, fmt.Errorf("commit create product for shipment item tx: %w", err)
	}

	return updated, nil
}

func (s *AdminService) GenerateInboundShipment(ctx context.Context, shipmentID int64) (models.InboundShipmentGenerateResult, error) {
	if shipmentID <= 0 {
		return models.InboundShipmentGenerateResult{}, ErrInvalidAdminInput
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.InboundShipmentGenerateResult{}, fmt.Errorf("begin generate shipment tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	shipmentRepo := repository.NewInboundShipmentRepositoryWithQuerier(tx)
	boxRepo := repository.NewBoxRepositoryWithQuerier(tx)
	batchRepo := repository.NewBatchRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)

	shipment, err := shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.InboundShipmentGenerateResult{}, ErrInvalidAdminReference
		}
		return models.InboundShipmentGenerateResult{}, err
	}
	if shipment.Status == "received" {
		return models.InboundShipmentGenerateResult{}, ErrInboundShipmentGenerated
	}
	if shipment.UnresolvedItems > 0 {
		return models.InboundShipmentGenerateResult{}, ErrInboundShipmentUnresolved
	}

	items, err := shipmentRepo.ListItems(ctx, shipmentID)
	if err != nil {
		return models.InboundShipmentGenerateResult{}, err
	}
	boxes, err := shipmentRepo.ListBoxes(ctx, shipmentID)
	if err != nil {
		return models.InboundShipmentGenerateResult{}, err
	}

	itemByID := make(map[int64]models.InboundShipmentItem, len(items))
	for _, item := range items {
		if item.ProductID == nil {
			return models.InboundShipmentGenerateResult{}, ErrInboundShipmentUnresolved
		}
		itemByID[item.ID] = item
	}

	for index, plannedBox := range boxes {
		if plannedBox.BoxID != nil || plannedBox.BatchID != nil {
			continue
		}
		item := itemByID[plannedBox.ShipmentItemID]
		if item.ProductID == nil {
			return models.InboundShipmentGenerateResult{}, ErrInboundShipmentUnresolved
		}

		boxCode := fmt.Sprintf("%s-BOX-%03d", shipment.Code, index+1)
		box, err := boxRepo.Create(ctx, boxCode, "active", nil)
		if err != nil {
			return models.InboundShipmentGenerateResult{}, err
		}
		if _, err := markerRepo.Create(ctx, buildMarkerCode("box", box.ID), "box", box.ID); err != nil {
			return models.InboundShipmentGenerateResult{}, err
		}

		batchCode := fmt.Sprintf("%s-BAT-%03d", shipment.Code, index+1)
		batch, err := batchRepo.Create(ctx, batchCode, *item.ProductID, plannedBox.PlannedQuantity, "active", &box.ID, nil)
		if err != nil {
			return models.InboundShipmentGenerateResult{}, err
		}
		if _, err := markerRepo.Create(ctx, buildMarkerCode("batch", batch.ID), "batch", batch.ID); err != nil {
			return models.InboundShipmentGenerateResult{}, err
		}

		if err := shipmentRepo.AssignBoxBatch(ctx, plannedBox.ID, box.ID, batch.ID); err != nil {
			return models.InboundShipmentGenerateResult{}, err
		}
	}

	if err := shipmentRepo.UpdateStatus(ctx, shipmentID, "received"); err != nil {
		return models.InboundShipmentGenerateResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.InboundShipmentGenerateResult{}, fmt.Errorf("commit generate shipment tx: %w", err)
	}

	updatedShipment, generatedBoxes, err := s.GetInboundShipment(ctx, shipmentID)
	if err != nil {
		return models.InboundShipmentGenerateResult{}, err
	}

	return models.InboundShipmentGenerateResult{
		Shipment: updatedShipment,
		Boxes:    generatedBoxes,
	}, nil
}

func parseInboundShipmentImportRows(reader io.Reader) ([]inboundShipmentImportRow, error) {
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

	indexes, err := resolveInboundShipmentHeaderIndexes(rawRows[0])
	if err != nil {
		return nil, err
	}

	rows := make([]inboundShipmentImportRow, 0, len(rawRows)-1)
	for index, rawRow := range rawRows[1:] {
		row := inboundShipmentImportRow{
			Row:             index + 2,
			SupplierName:    strings.TrimSpace(cellValue(rawRow, indexes.supplierName)),
			SupplierArticle: strings.TrimSpace(cellValue(rawRow, indexes.supplierArticle)),
			ProductName:     strings.TrimSpace(cellValue(rawRow, indexes.productName)),
			Unit:            strings.TrimSpace(cellValue(rawRow, indexes.unit)),
		}
		if row.Unit == "" {
			row.Unit = "pcs"
		}
		if row.SupplierName == "" && row.SupplierArticle == "" && row.ProductName == "" {
			continue
		}

		totalQuantity, totalErr := parsePositiveInt32(cellValue(rawRow, indexes.totalQuantity))
		boxesCount, boxesErr := parsePositiveInt32(cellValue(rawRow, indexes.boxesCount))
		quantityPerBox, perBoxErr := parsePositiveInt32(cellValue(rawRow, indexes.quantityPerBox))
		row.TotalQuantity = totalQuantity
		row.BoxesCount = boxesCount
		row.QuantityPerBox = quantityPerBox

		if row.SupplierName == "" || row.SupplierArticle == "" || row.ProductName == "" ||
			totalErr != nil || boxesErr != nil || perBoxErr != nil {
			row.ValidationError = "invalid shipment row"
		}

		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return nil, ErrEmptyAdminImport
	}

	return rows, nil
}

type inboundShipmentHeaderIndexes struct {
	supplierName    int
	supplierArticle int
	productName     int
	unit            int
	totalQuantity   int
	boxesCount      int
	quantityPerBox  int
}

func resolveInboundShipmentHeaderIndexes(header []string) (inboundShipmentHeaderIndexes, error) {
	indexes := inboundShipmentHeaderIndexes{
		supplierName:    -1,
		supplierArticle: -1,
		productName:     -1,
		unit:            -1,
		totalQuantity:   -1,
		boxesCount:      -1,
		quantityPerBox:  -1,
	}

	for index, rawValue := range header {
		switch normalizeProductImportHeader(rawValue) {
		case "suppliername", "supplier", "поставщик":
			indexes.supplierName = index
		case "supplierarticle", "article", "артикул", "артикулпоставщика":
			indexes.supplierArticle = index
		case "productname", "name", "товар", "название", "наименование":
			indexes.productName = index
		case "unit", "ед", "единица":
			indexes.unit = index
		case "totalquantity", "quantity", "количество", "всего":
			indexes.totalQuantity = index
		case "boxescount", "boxes", "коробов", "короба":
			indexes.boxesCount = index
		case "quantityperbox", "perbox", "вкоробе", "колвокоробе":
			indexes.quantityPerBox = index
		}
	}

	if indexes.supplierName < 0 ||
		indexes.supplierArticle < 0 ||
		indexes.productName < 0 ||
		indexes.totalQuantity < 0 ||
		indexes.boxesCount < 0 ||
		indexes.quantityPerBox < 0 {
		return inboundShipmentHeaderIndexes{}, ErrInvalidAdminImport
	}

	return indexes, nil
}

func parsePositiveInt32(value string) (int32, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, ErrInvalidAdminImport
	}

	return int32(parsed), nil
}

func splitShipmentQuantities(totalQuantity int32, boxesCount int32, quantityPerBox int32) []int32 {
	quantities := make([]int32, 0, boxesCount)
	remaining := totalQuantity
	for i := int32(0); i < boxesCount; i++ {
		quantity := quantityPerBox
		if remaining < quantity {
			quantity = remaining
		}
		if i == boxesCount-1 {
			quantity = remaining
		}
		if quantity <= 0 {
			quantity = quantityPerBox
		}
		quantities = append(quantities, quantity)
		remaining -= quantity
	}

	return quantities
}
