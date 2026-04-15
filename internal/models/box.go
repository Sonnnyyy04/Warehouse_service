package models

type Box struct {
	ID            int64  `json:"id"`
	Code          string `json:"code"`
	Status        string `json:"status"`
	PalletID      *int64 `json:"pallet_id"`
	StorageCellID *int64 `json:"storage_cell_id"`
}
