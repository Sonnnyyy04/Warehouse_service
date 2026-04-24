package models

type ScanEventFilter struct {
	Limit      int32
	UserID     *int64
	MarkerCode string
}
