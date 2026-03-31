package models

import "time"

type ScanEvent struct {
	ID         int64     `json:"id"`
	MarkerCode string    `json:"marker_code"`
	UserID     *int64    `json:"user_id,omitempty"`
	DeviceInfo *string   `json:"device_info,omitempty"`
	Success    bool      `json:"success"`
	ScannedAt  time.Time `json:"scanned_at"`
}
