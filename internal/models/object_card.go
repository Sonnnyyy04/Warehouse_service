package models

type ObjectCard struct {
	MarkerCode     string  `json:"marker_code"`
	ObjectType     string  `json:"object_type"`
	ObjectID       int64   `json:"object_id"`
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	LocationCode   *string `json:"location_code,omitempty"`
	LocationType   *string `json:"location_type,omitempty"`
	ParentCode     *string `json:"parent_code,omitempty"`
	ParentType     *string `json:"parent_type,omitempty"`
	Quantity       *int32  `json:"quantity,omitempty"`
	Unit           *string `json:"unit,omitempty"`
	ProductSKU     *string `json:"product_sku,omitempty"`
	ProductName    *string `json:"product_name,omitempty"`
	ContentSummary *string `json:"content_summary,omitempty"`
	PalletsCount   *int32  `json:"pallets_count,omitempty"`
	BoxesCount     *int32  `json:"boxes_count,omitempty"`
	BatchesCount   *int32  `json:"batches_count,omitempty"`
	ProductsCount  *int32  `json:"products_count,omitempty"`
	TotalQuantity  *int32  `json:"total_quantity,omitempty"`
}

type ObjectContentStats struct {
	PalletsCount  int32
	BoxesCount    int32
	BatchesCount  int32
	ProductsCount int32
	TotalQuantity int32
	ProductSKU    *string
	ProductName   *string
	ProductUnit   *string
}
