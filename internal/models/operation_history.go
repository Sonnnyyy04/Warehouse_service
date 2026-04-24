package models

import (
	"encoding/json"
	"time"
)

type OperationHistory struct {
	ID            int64            `json:"id"`
	ObjectType    string           `json:"object_type"`
	ObjectID      int64            `json:"object_id"`
	OperationType string           `json:"operation_type"`
	UserID        *int64           `json:"user_id,omitempty"`
	Actor         *UserSummary     `json:"actor,omitempty"`
	Details       *json.RawMessage `json:"details,omitempty" swaggertype:"object"`
	CreatedAt     time.Time        `json:"created_at"`
}
