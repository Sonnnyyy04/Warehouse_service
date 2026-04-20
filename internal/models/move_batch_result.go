package models

type MoveBatchResult struct {
	Batch     ObjectCard       `json:"batch"`
	Operation OperationHistory `json:"operation"`
}
