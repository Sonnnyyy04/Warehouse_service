package models

type OutboundShipmentBox struct {
	BoxID      int64  `json:"box_id"`
	BoxCode    string `json:"box_code"`
	MarkerCode string `json:"marker_code"`
	Quantity   int32  `json:"quantity"`
}

type OutboundShipmentResult struct {
	Product           Product               `json:"product"`
	RequestedQuantity int32                 `json:"requested_quantity"`
	ShippedQuantity   int32                 `json:"shipped_quantity"`
	Boxes             []OutboundShipmentBox `json:"boxes"`
	Operation         OperationHistory      `json:"operation"`
}
