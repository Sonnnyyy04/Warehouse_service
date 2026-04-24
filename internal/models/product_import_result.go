package models

type ProductImportResult struct {
	TotalRows    int                     `json:"total_rows"`
	CreatedCount int                     `json:"created_count"`
	SkippedCount int                     `json:"skipped_count"`
	Errors       []ProductImportRowError `json:"errors"`
}

type ProductImportRowError struct {
	Row   int    `json:"row"`
	SKU   string `json:"sku,omitempty"`
	Name  string `json:"name,omitempty"`
	Error string `json:"error"`
}
