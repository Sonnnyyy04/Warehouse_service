package models

type Label struct {
	MarkerCode string `json:"marker_code"`
	ObjectType string `json:"object_type"`
	ObjectID   int64  `json:"object_id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
}
