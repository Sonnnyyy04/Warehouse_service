package models

type StorageCell struct {
	ID     int64   `json:"id"`
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Zone   *string `json:"zone"`
	Status string  `json:"status"`
}
