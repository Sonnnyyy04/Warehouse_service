package models

type Batch struct {
	ID            int64
	Code          string
	ProductID     int64
	Quantity      int32
	Status        string
	BoxID         *int64
	PalletID      *int64
	StorageCellID *int64
}
