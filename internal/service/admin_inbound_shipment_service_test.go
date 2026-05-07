package service

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseInboundShipmentImportRowsWithMultipleSuppliers(t *testing.T) {
	t.Parallel()

	var file bytes.Buffer
	xlsx := excelize.NewFile()
	sheet := xlsx.GetSheetName(0)
	rows := [][]any{
		{"supplier_name", "supplier_article", "product_name", "unit", "total_quantity", "boxes_count", "quantity_per_box"},
		{"Supplier A", "A-100", "Hammer 500g", "pcs", 24, 4, 6},
		{"Supplier B", "B-200", "Boots", "pcs", 20, 2, 10},
	}
	for rowIndex, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, rowIndex+1)
		if err != nil {
			t.Fatalf("build cell name: %v", err)
		}
		if err := xlsx.SetSheetRow(sheet, cell, &row); err != nil {
			t.Fatalf("set sheet row: %v", err)
		}
	}
	if err := xlsx.Write(&file); err != nil {
		t.Fatalf("write xlsx: %v", err)
	}

	parsedRows, err := parseInboundShipmentImportRows(&file)
	if err != nil {
		t.Fatalf("parse inbound shipment rows: %v", err)
	}
	if len(parsedRows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(parsedRows))
	}
	if parsedRows[0].SupplierName != "Supplier A" {
		t.Fatalf("expected first row supplier Supplier A, got %q", parsedRows[0].SupplierName)
	}
	if parsedRows[1].SupplierName != "Supplier B" {
		t.Fatalf("expected second row supplier Supplier B, got %q", parsedRows[1].SupplierName)
	}
	if got := shipmentSupplierName(parsedRows); got != "multiple suppliers" {
		t.Fatalf("expected multiple suppliers shipment label, got %q", got)
	}
}

func TestSplitShipmentQuantitiesUsesLastBoxRemainder(t *testing.T) {
	t.Parallel()

	quantities := splitShipmentQuantities(23, 4, 6)
	expected := []int32{6, 6, 6, 5}
	if len(quantities) != len(expected) {
		t.Fatalf("expected %d quantities, got %d", len(expected), len(quantities))
	}
	for index := range expected {
		if quantities[index] != expected[index] {
			t.Fatalf("expected quantity %d at index %d, got %d", expected[index], index, quantities[index])
		}
	}
}
