package models

type MoveBoxResult struct {
	Box       ObjectCard       `json:"box"`
	Operation OperationHistory `json:"operation"`
}
