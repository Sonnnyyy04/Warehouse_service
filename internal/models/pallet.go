package models

type Pallet struct {
	ID            int64
	Code          string
	Status        string
	StorageCellID *int64
}
