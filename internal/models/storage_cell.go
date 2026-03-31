package models

type StorageCell struct {
	ID     int64
	Code   string
	Name   string
	Zone   *string
	Status string
}
