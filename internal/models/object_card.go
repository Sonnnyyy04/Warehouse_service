package models

type ObjectCard struct {
	MarkerCode   string  `json:"marker_code"`
	ObjectType   string  `json:"object_type"`
	ObjectID     int64   `json:"object_id"`
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	LocationCode *string `json:"location_code,omitempty"`
	ParentCode   *string `json:"parent_code,omitempty"`
	Quantity     *int32  `json:"quantity,omitempty"`
}
