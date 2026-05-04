package models

type Box struct {
	ID            int64  `json:"id"`
	Code          string `json:"code"`
	Status        string `json:"status"`
	StorageCellID *int64 `json:"storage_cell_id"`
}
