package models

type Product struct {
	ID            int64  `json:"id"`
	SKU           string `json:"sku"`
	Name          string `json:"name"`
	Unit          string `json:"unit"`
	TotalQuantity int32  `json:"total_quantity"`
}
