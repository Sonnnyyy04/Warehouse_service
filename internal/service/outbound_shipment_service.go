package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"Warehouse_service/internal/models"
	"Warehouse_service/internal/repository"
)

var (
	ErrInvalidOutboundShipmentPayload = errors.New("invalid outbound shipment payload")
	ErrOutboundShipmentBoxNotFound    = errors.New("outbound shipment box not found")
	ErrOutboundShipmentNotEnoughStock = errors.New("outbound shipment not enough stock")
)

type OutboundShipmentInput struct {
	ProductID         int64
	RequestedQuantity int32
	BoxMarkerCodes    []string
	UserID            *int64
	Actor             *models.UserSummary
}

type OutboundShipmentService struct {
	txPool repository.TxBeginner
}

func NewOutboundShipmentService(txPool repository.TxBeginner) *OutboundShipmentService {
	return &OutboundShipmentService{txPool: txPool}
}

func (s *OutboundShipmentService) Complete(ctx context.Context, input OutboundShipmentInput) (models.OutboundShipmentResult, error) {
	markerCodes := normalizeMarkerCodes(input.BoxMarkerCodes)
	if input.ProductID <= 0 || input.RequestedQuantity <= 0 || len(markerCodes) == 0 {
		return models.OutboundShipmentResult{}, ErrInvalidOutboundShipmentPayload
	}

	tx, err := s.txPool.Begin(ctx)
	if err != nil {
		return models.OutboundShipmentResult{}, fmt.Errorf("begin outbound shipment tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	productRepo := repository.NewProductRepositoryWithQuerier(tx)
	markerRepo := repository.NewMarkerRepositoryWithQuerier(tx)
	boxRepo := repository.NewBoxRepositoryWithQuerier(tx)
	batchRepo := repository.NewBatchRepositoryWithQuerier(tx)
	operationRepo := repository.NewOperationHistoryRepositoryWithQuerier(tx)
	operationWriter := NewOperationHistoryService(operationRepo)

	product, err := productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.OutboundShipmentResult{}, ErrObjectNotFound
		}
		return models.OutboundShipmentResult{}, err
	}

	markers, err := markerRepo.ListByCodes(ctx, "box", markerCodes)
	if err != nil {
		return models.OutboundShipmentResult{}, err
	}
	if len(markers) != len(markerCodes) {
		return models.OutboundShipmentResult{}, ErrOutboundShipmentBoxNotFound
	}

	boxIDs := make([]int64, 0, len(markers))
	markerByBoxID := make(map[int64]string, len(markers))
	for _, marker := range markers {
		if marker.ObjectType != "box" {
			return models.OutboundShipmentResult{}, ErrInvalidBoxMarkerType
		}
		boxIDs = append(boxIDs, marker.ObjectID)
		markerByBoxID[marker.ObjectID] = marker.MarkerCode
	}

	boxes, err := boxRepo.ListByIDs(ctx, boxIDs)
	if err != nil {
		return models.OutboundShipmentResult{}, err
	}
	if len(boxes) != len(boxIDs) {
		return models.OutboundShipmentResult{}, ErrOutboundShipmentBoxNotFound
	}

	boxByID := make(map[int64]models.Box, len(boxes))
	for _, box := range boxes {
		if box.Status != "active" {
			return models.OutboundShipmentResult{}, ErrOutboundShipmentBoxNotFound
		}
		boxByID[box.ID] = box
	}

	batches, err := batchRepo.ListByBoxIDsAndProductID(ctx, boxIDs, input.ProductID)
	if err != nil {
		return models.OutboundShipmentResult{}, err
	}

	batchIDs := make([]int64, 0, len(batches))
	batchCodes := make([]string, 0, len(batches))
	boxQuantity := make(map[int64]int32, len(boxIDs))
	for _, batch := range batches {
		if batch.BoxID == nil {
			continue
		}
		boxQuantity[*batch.BoxID] += batch.Quantity
		batchIDs = append(batchIDs, batch.ID)
		batchCodes = append(batchCodes, batch.Code)
	}

	shipmentBoxes := make([]models.OutboundShipmentBox, 0, len(boxIDs))
	var shippedQuantity int32
	for _, boxID := range boxIDs {
		quantity := boxQuantity[boxID]
		if quantity <= 0 {
			return models.OutboundShipmentResult{}, ErrOutboundShipmentBoxNotFound
		}

		box := boxByID[boxID]
		shippedQuantity += quantity
		shipmentBoxes = append(shipmentBoxes, models.OutboundShipmentBox{
			BoxID:      box.ID,
			BoxCode:    box.Code,
			MarkerCode: markerByBoxID[boxID],
			Quantity:   quantity,
		})
	}

	if shippedQuantity < input.RequestedQuantity {
		return models.OutboundShipmentResult{}, ErrOutboundShipmentNotEnoughStock
	}

	if err := markerRepo.DeleteByObjectIDs(ctx, "batch", batchIDs); err != nil {
		return models.OutboundShipmentResult{}, err
	}
	if err := batchRepo.DeleteByIDs(ctx, batchIDs); err != nil {
		return models.OutboundShipmentResult{}, err
	}
	if err := boxRepo.MarkShipped(ctx, boxIDs); err != nil {
		return models.OutboundShipmentResult{}, err
	}

	detailsBytes, err := json.Marshal(map[string]any{
		"action":             "outbound_shipment",
		"product_id":         product.ID,
		"product_sku":        product.SKU,
		"product_name":       product.Name,
		"requested_quantity": input.RequestedQuantity,
		"shipped_quantity":   shippedQuantity,
		"box_marker_codes":   markerCodes,
		"batch_codes":        batchCodes,
	})
	if err != nil {
		return models.OutboundShipmentResult{}, err
	}
	rawDetails := json.RawMessage(detailsBytes)

	operation, err := operationWriter.Create(ctx, CreateOperationInput{
		ObjectType:    "product",
		ObjectID:      product.ID,
		OperationType: "outbound_shipment",
		UserID:        input.UserID,
		Actor:         input.Actor,
		Details:       &rawDetails,
	})
	if err != nil {
		return models.OutboundShipmentResult{}, err
	}

	updatedProduct, err := productRepo.GetByID(ctx, product.ID)
	if err == nil {
		product = updatedProduct
	}

	if err := tx.Commit(ctx); err != nil {
		return models.OutboundShipmentResult{}, fmt.Errorf("commit outbound shipment tx: %w", err)
	}

	return models.OutboundShipmentResult{
		Product:           product,
		RequestedQuantity: input.RequestedQuantity,
		ShippedQuantity:   shippedQuantity,
		Boxes:             shipmentBoxes,
		Operation:         operation,
	}, nil
}

func normalizeMarkerCodes(rawCodes []string) []string {
	seen := make(map[string]struct{}, len(rawCodes))
	codes := make([]string, 0, len(rawCodes))

	for _, rawCode := range rawCodes {
		code := strings.TrimSpace(rawCode)
		if code == "" {
			continue
		}
		normalized := strings.ToLower(code)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		codes = append(codes, code)
	}

	return codes
}
