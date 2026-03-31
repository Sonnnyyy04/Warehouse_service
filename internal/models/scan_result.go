package models

type ScanResult struct {
	Object    ObjectCard `json:"object"`
	ScanEvent ScanEvent  `json:"scan_event"`
}
