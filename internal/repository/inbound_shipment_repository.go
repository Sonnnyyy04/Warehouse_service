package repository

import (
	"context"
	"errors"
	"fmt"

	"Warehouse_service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InboundShipmentRepository struct {
	db Querier
}

func NewInboundShipmentRepository(pool *pgxpool.Pool) *InboundShipmentRepository {
	return NewInboundShipmentRepositoryWithQuerier(pool)
}

func NewInboundShipmentRepositoryWithQuerier(db Querier) *InboundShipmentRepository {
	return &InboundShipmentRepository{db: db}
}

func (r *InboundShipmentRepository) Create(ctx context.Context, code, supplierName string) (models.InboundShipment, error) {
	const query = `
INSERT INTO inbound_shipments (code, supplier_name, status)
VALUES ($1, $2, 'draft')
RETURNING id, code, supplier_name, status, created_at
`

	var shipment models.InboundShipment
	if err := r.db.QueryRow(ctx, query, code, supplierName).Scan(
		&shipment.ID,
		&shipment.Code,
		&shipment.SupplierName,
		&shipment.Status,
		&shipment.CreatedAt,
	); err != nil {
		return models.InboundShipment{}, fmt.Errorf("create inbound shipment: %w", err)
	}

	return shipment, nil
}

func (r *InboundShipmentRepository) List(ctx context.Context, limit int32) ([]models.InboundShipment, error) {
	const query = `
SELECT
    s.id,
    s.code,
    s.supplier_name,
    s.status,
    s.created_at,
    COUNT(i.id)::int,
    COUNT(i.id) FILTER (WHERE i.status = 'matched')::int,
    COUNT(i.id) FILTER (WHERE i.status = 'unresolved')::int,
    COALESCE(SUM(i.boxes_count), 0)::int,
    COALESCE(SUM(i.total_quantity), 0)::int
FROM inbound_shipments s
LEFT JOIN inbound_shipment_items i ON i.shipment_id = s.id
GROUP BY s.id, s.code, s.supplier_name, s.status, s.created_at
ORDER BY s.id DESC
LIMIT $1
`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list inbound shipments: %w", err)
	}
	defer rows.Close()

	shipments := make([]models.InboundShipment, 0)
	for rows.Next() {
		var shipment models.InboundShipment
		if err := rows.Scan(
			&shipment.ID,
			&shipment.Code,
			&shipment.SupplierName,
			&shipment.Status,
			&shipment.CreatedAt,
			&shipment.TotalItems,
			&shipment.MatchedItems,
			&shipment.UnresolvedItems,
			&shipment.BoxesCount,
			&shipment.TotalQuantity,
		); err != nil {
			return nil, fmt.Errorf("scan inbound shipment: %w", err)
		}
		shipments = append(shipments, shipment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inbound shipments: %w", err)
	}

	return shipments, nil
}

func (r *InboundShipmentRepository) GetByID(ctx context.Context, id int64) (models.InboundShipment, error) {
	const query = `
SELECT
    s.id,
    s.code,
    s.supplier_name,
    s.status,
    s.created_at,
    COUNT(i.id)::int,
    COUNT(i.id) FILTER (WHERE i.status = 'matched')::int,
    COUNT(i.id) FILTER (WHERE i.status = 'unresolved')::int,
    COALESCE(SUM(i.boxes_count), 0)::int,
    COALESCE(SUM(i.total_quantity), 0)::int
FROM inbound_shipments s
LEFT JOIN inbound_shipment_items i ON i.shipment_id = s.id
WHERE s.id = $1
GROUP BY s.id, s.code, s.supplier_name, s.status, s.created_at
`

	var shipment models.InboundShipment
	if err := r.db.QueryRow(ctx, query, id).Scan(
		&shipment.ID,
		&shipment.Code,
		&shipment.SupplierName,
		&shipment.Status,
		&shipment.CreatedAt,
		&shipment.TotalItems,
		&shipment.MatchedItems,
		&shipment.UnresolvedItems,
		&shipment.BoxesCount,
		&shipment.TotalQuantity,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.InboundShipment{}, ErrNotFound
		}
		return models.InboundShipment{}, fmt.Errorf("get inbound shipment: %w", err)
	}

	return shipment, nil
}

func (r *InboundShipmentRepository) CreateItem(ctx context.Context, item models.InboundShipmentItem) (models.InboundShipmentItem, error) {
	const query = `
INSERT INTO inbound_shipment_items (
    shipment_id,
    product_id,
    supplier_article,
    product_name,
    unit,
    total_quantity,
    boxes_count,
    quantity_per_box,
    status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, shipment_id, product_id, supplier_article, product_name, unit, total_quantity, boxes_count, quantity_per_box, status, created_at
`

	var created models.InboundShipmentItem
	if err := r.db.QueryRow(
		ctx,
		query,
		item.ShipmentID,
		item.ProductID,
		item.SupplierArticle,
		item.ProductName,
		item.Unit,
		item.TotalQuantity,
		item.BoxesCount,
		item.QuantityPerBox,
		item.Status,
	).Scan(
		&created.ID,
		&created.ShipmentID,
		&created.ProductID,
		&created.SupplierArticle,
		&created.ProductName,
		&created.Unit,
		&created.TotalQuantity,
		&created.BoxesCount,
		&created.QuantityPerBox,
		&created.Status,
		&created.CreatedAt,
	); err != nil {
		return models.InboundShipmentItem{}, fmt.Errorf("create inbound shipment item: %w", err)
	}

	return created, nil
}

func (r *InboundShipmentRepository) ListItems(ctx context.Context, shipmentID int64) ([]models.InboundShipmentItem, error) {
	const query = `
SELECT
    i.id,
    i.shipment_id,
    i.product_id,
    p.sku,
    i.supplier_article,
    i.product_name,
    i.unit,
    i.total_quantity,
    i.boxes_count,
    i.quantity_per_box,
    i.status,
    i.created_at
FROM inbound_shipment_items i
LEFT JOIN products p ON p.id = i.product_id
WHERE i.shipment_id = $1
ORDER BY i.id
`

	rows, err := r.db.Query(ctx, query, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("list inbound shipment items: %w", err)
	}
	defer rows.Close()

	items := make([]models.InboundShipmentItem, 0)
	for rows.Next() {
		var item models.InboundShipmentItem
		if err := rows.Scan(
			&item.ID,
			&item.ShipmentID,
			&item.ProductID,
			&item.ProductSKU,
			&item.SupplierArticle,
			&item.ProductName,
			&item.Unit,
			&item.TotalQuantity,
			&item.BoxesCount,
			&item.QuantityPerBox,
			&item.Status,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan inbound shipment item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inbound shipment items: %w", err)
	}

	return items, nil
}

func (r *InboundShipmentRepository) GetItemByID(ctx context.Context, id int64) (models.InboundShipmentItem, error) {
	const query = `
SELECT
    i.id,
    i.shipment_id,
    i.product_id,
    p.sku,
    i.supplier_article,
    i.product_name,
    i.unit,
    i.total_quantity,
    i.boxes_count,
    i.quantity_per_box,
    i.status,
    i.created_at
FROM inbound_shipment_items i
LEFT JOIN products p ON p.id = i.product_id
WHERE i.id = $1
`

	var item models.InboundShipmentItem
	if err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.ShipmentID,
		&item.ProductID,
		&item.ProductSKU,
		&item.SupplierArticle,
		&item.ProductName,
		&item.Unit,
		&item.TotalQuantity,
		&item.BoxesCount,
		&item.QuantityPerBox,
		&item.Status,
		&item.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.InboundShipmentItem{}, ErrNotFound
		}
		return models.InboundShipmentItem{}, fmt.Errorf("get inbound shipment item: %w", err)
	}

	return item, nil
}

func (r *InboundShipmentRepository) UpdateItemProduct(ctx context.Context, itemID int64, productID int64) (models.InboundShipmentItem, error) {
	const query = `
WITH updated AS (
    UPDATE inbound_shipment_items
    SET product_id = $2,
        status = 'matched'
    WHERE id = $1
    RETURNING id, shipment_id, product_id, supplier_article, product_name, unit, total_quantity, boxes_count, quantity_per_box, status, created_at
)
SELECT u.id, u.shipment_id, u.product_id, p.sku, u.supplier_article, u.product_name, u.unit, u.total_quantity, u.boxes_count, u.quantity_per_box, u.status, u.created_at
FROM updated u
LEFT JOIN products p ON p.id = u.product_id
`

	var item models.InboundShipmentItem
	if err := r.db.QueryRow(ctx, query, itemID, productID).Scan(
		&item.ID,
		&item.ShipmentID,
		&item.ProductID,
		&item.ProductSKU,
		&item.SupplierArticle,
		&item.ProductName,
		&item.Unit,
		&item.TotalQuantity,
		&item.BoxesCount,
		&item.QuantityPerBox,
		&item.Status,
		&item.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.InboundShipmentItem{}, ErrNotFound
		}
		return models.InboundShipmentItem{}, fmt.Errorf("update shipment item product: %w", err)
	}

	return item, nil
}

func (r *InboundShipmentRepository) CreatePlannedBox(ctx context.Context, itemID int64, plannedQuantity int32) (models.InboundShipmentBox, error) {
	const query = `
INSERT INTO inbound_shipment_boxes (shipment_item_id, planned_quantity, status)
VALUES ($1, $2, 'planned')
RETURNING id, shipment_item_id, box_id, batch_id, planned_quantity, status
`

	var box models.InboundShipmentBox
	if err := r.db.QueryRow(ctx, query, itemID, plannedQuantity).Scan(
		&box.ID,
		&box.ShipmentItemID,
		&box.BoxID,
		&box.BatchID,
		&box.PlannedQuantity,
		&box.Status,
	); err != nil {
		return models.InboundShipmentBox{}, fmt.Errorf("create planned shipment box: %w", err)
	}

	return box, nil
}

func (r *InboundShipmentRepository) ListBoxes(ctx context.Context, shipmentID int64) ([]models.InboundShipmentBox, error) {
	const query = `
SELECT
    sb.id,
    sb.shipment_item_id,
    sb.box_id,
    sb.batch_id,
    bx.code,
    bt.code,
    bm.marker_code,
    btm.marker_code,
    sb.planned_quantity,
    sb.status
FROM inbound_shipment_boxes sb
JOIN inbound_shipment_items si ON si.id = sb.shipment_item_id
LEFT JOIN boxes bx ON bx.id = sb.box_id
LEFT JOIN batches bt ON bt.id = sb.batch_id
LEFT JOIN markers bm ON bm.object_type = 'box'::object_type AND bm.object_id = sb.box_id
LEFT JOIN markers btm ON btm.object_type = 'batch'::object_type AND btm.object_id = sb.batch_id
WHERE si.shipment_id = $1
ORDER BY sb.id
`

	rows, err := r.db.Query(ctx, query, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("list shipment boxes: %w", err)
	}
	defer rows.Close()

	boxes := make([]models.InboundShipmentBox, 0)
	for rows.Next() {
		var box models.InboundShipmentBox
		if err := rows.Scan(
			&box.ID,
			&box.ShipmentItemID,
			&box.BoxID,
			&box.BatchID,
			&box.BoxCode,
			&box.BatchCode,
			&box.BoxMarkerCode,
			&box.BatchMarkerCode,
			&box.PlannedQuantity,
			&box.Status,
		); err != nil {
			return nil, fmt.Errorf("scan shipment box: %w", err)
		}
		boxes = append(boxes, box)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shipment boxes: %w", err)
	}

	return boxes, nil
}

func (r *InboundShipmentRepository) AssignBoxBatch(ctx context.Context, shipmentBoxID int64, boxID int64, batchID int64) error {
	const query = `
UPDATE inbound_shipment_boxes
SET box_id = $2,
    batch_id = $3,
    status = 'received'
WHERE id = $1
`

	tag, err := r.db.Exec(ctx, query, shipmentBoxID, boxID, batchID)
	if err != nil {
		return fmt.Errorf("assign shipment box batch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *InboundShipmentRepository) UpdateStatus(ctx context.Context, shipmentID int64, status string) error {
	const query = `
UPDATE inbound_shipments
SET status = $2
WHERE id = $1
`

	tag, err := r.db.Exec(ctx, query, shipmentID, status)
	if err != nil {
		return fmt.Errorf("update shipment status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
