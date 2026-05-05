package models

import "time"

type ProductAlias struct {
	ID           int64     `json:"id"`
	ProductID    int64     `json:"product_id"`
	SupplierName string    `json:"supplier_name"`
	AliasType    string    `json:"alias_type"`
	AliasValue   string    `json:"alias_value"`
	CreatedAt    time.Time `json:"created_at"`
}

type InboundShipment struct {
	ID              int64                 `json:"id"`
	Code            string                `json:"code"`
	SupplierName    string                `json:"supplier_name"`
	Status          string                `json:"status"`
	CreatedAt       time.Time             `json:"created_at"`
	Items           []InboundShipmentItem `json:"items,omitempty"`
	TotalItems      int32                 `json:"total_items"`
	MatchedItems    int32                 `json:"matched_items"`
	UnresolvedItems int32                 `json:"unresolved_items"`
	BoxesCount      int32                 `json:"boxes_count"`
	TotalQuantity   int32                 `json:"total_quantity"`
}

type InboundShipmentItem struct {
	ID              int64     `json:"id"`
	ShipmentID      int64     `json:"shipment_id"`
	ProductID       *int64    `json:"product_id,omitempty"`
	ProductSKU      *string   `json:"product_sku,omitempty"`
	SupplierArticle string    `json:"supplier_article"`
	ProductName     string    `json:"product_name"`
	Unit            string    `json:"unit"`
	TotalQuantity   int32     `json:"total_quantity"`
	BoxesCount      int32     `json:"boxes_count"`
	QuantityPerBox  int32     `json:"quantity_per_box"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type InboundShipmentBox struct {
	ID              int64   `json:"id"`
	ShipmentItemID  int64   `json:"shipment_item_id"`
	BoxID           *int64  `json:"box_id,omitempty"`
	BatchID         *int64  `json:"batch_id,omitempty"`
	BoxCode         *string `json:"box_code,omitempty"`
	BatchCode       *string `json:"batch_code,omitempty"`
	BoxMarkerCode   *string `json:"box_marker_code,omitempty"`
	BatchMarkerCode *string `json:"batch_marker_code,omitempty"`
	PlannedQuantity int32   `json:"planned_quantity"`
	Status          string  `json:"status"`
}

type InboundShipmentImportResult struct {
	Shipment InboundShipment `json:"shipment"`
}

type InboundShipmentGenerateResult struct {
	Shipment InboundShipment      `json:"shipment"`
	Boxes    []InboundShipmentBox `json:"boxes"`
}
