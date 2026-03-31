package models

type Box struct {
	ID            int64
	Code          string
	Status        string
	PalletID      *int64
	StorageCellID *int64
}
