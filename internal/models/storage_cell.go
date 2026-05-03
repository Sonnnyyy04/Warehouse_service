package models

type StorageCell struct {
	ID       int64   `json:"id"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Zone     *string `json:"zone"`
	Status   string  `json:"status"`
	RackID   *int64  `json:"rack_id,omitempty"`
	RackCode *string `json:"rack_code,omitempty"`
	RackName *string `json:"rack_name,omitempty"`
}
