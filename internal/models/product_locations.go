package models

type ProductLocation struct {
	BoxID           int64   `json:"box_id"`
	BoxCode         string  `json:"box_code"`
	BoxMarkerCode   *string `json:"box_marker_code,omitempty"`
	BoxStatus       string  `json:"box_status"`
	BatchID         int64   `json:"batch_id"`
	BatchCode       string  `json:"batch_code"`
	BatchMarkerCode *string `json:"batch_marker_code,omitempty"`
	Quantity        int32   `json:"quantity"`
	StorageCellID   *int64  `json:"storage_cell_id,omitempty"`
	StorageCellCode *string `json:"storage_cell_code,omitempty"`
	StorageCellName *string `json:"storage_cell_name,omitempty"`
	RackID          *int64  `json:"rack_id,omitempty"`
	RackCode        *string `json:"rack_code,omitempty"`
	RackName        *string `json:"rack_name,omitempty"`
}

type ProductLocations struct {
	Product   Product           `json:"product"`
	Locations []ProductLocation `json:"locations"`
}
