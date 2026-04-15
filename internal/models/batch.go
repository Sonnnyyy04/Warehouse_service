package models

type Batch struct {
	ID            int64  `json:"id"`
	Code          string `json:"code"`
	ProductID     int64  `json:"product_id"`
	Quantity      int32  `json:"quantity"`
	Status        string `json:"status"`
	BoxID         *int64 `json:"box_id"`
	PalletID      *int64 `json:"pallet_id"`
	StorageCellID *int64 `json:"storage_cell_id"`
}
