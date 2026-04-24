package models

type OperationHistoryFilter struct {
	Limit      int32
	UserID     *int64
	ObjectType string
	ObjectID   *int64
}
