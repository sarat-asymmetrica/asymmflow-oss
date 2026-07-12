package main

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseCostingSheet_CustomerCostingLayout(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "customer_costing.xlsx")
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Customer Costing")

	values := map[string]any{
		"B2":  46096,
		"D2":  "RH 45-26",
		"F2":  "7-9 weeks",
		"B3":  "Riley Shah",
		"D3":  "RH 45-26",
		"F3":  "Delivered Duty Paid (DDP)",
		"B4":  "Gulf Smelting Co.",
		"D4":  "Choose Contact Person",
		"F4":  "General",
		"A5":  "Reference",
		"B5":  "Email",
		"D5":  "60 days from Date of Delivery",
		"F5":  "DE",
		"A7":  "Supplier",
		"A8":  "Equipment",
		"A9":  "Model",
		"A10": "Specification / Order Code",
		"A11": "Currency",
		"B11": "Quantity",
		"A12": "EUR",
		"B12": "FOB",
		"B13": "Total Price",
		"A14": "EUR",
		"B14": "Freight",
		"B17": "Exchange Rate to BHD",
		"A18": "BHD",
		"B18": "FOB",
		"A19": "BHD",
		"B19": "Freight",
		"B20": "C&F",
		"B21": "Insurance",
		"B22": "Customs @ 6%",
		"B23": "Landed Cost",
		"B24": "Handling @ 4%",
		"B25": "Finance Charges @1%",
		"B26": "Other Costs",
		"B27": "Total Cost",
		"B28": "Markup @ 20%",
		"A29": "Selling Price",
		"A30": "Suggested Price per unit",
		"A31": "Total Suggested Price",
		"A34": "Summary",
		"C34": "Total",
		"D34": "VAT",
		"A35": "Total PO value expected from Client",
		"C35": 2300.0,
		"D35": 230.0,
		"A36": "ACME INSTRUMENTATION COST",
		"C36": 2070.071399025,
		"D36": 207.0071399025,
		"C7":  "RH",
		"C8":  "Level Switch LS51B",
		"C9":  "LS51B",
		"C11": 5.0,
		"C12": 788.11,
		"C13": 3940.55,
		"C14": 70.9299,
		"C17": 0.45,
		"C18": 354.6495,
		"C19": 31.918455,
		"C20": 386.567955,
		"C21": 0.0,
		"C22": 19.32839775,
		"C23": 405.89635275,
		"C24": 4.0589635275,
		"C25": 4.0589635275,
		"C26": 0.0,
		"C27": 414.014279805,
		"C28": 45.54157077855,
		"C29": 459.55585058355,
		"C30": 460.0,
		"C31": 2300.0,
		"M8":  "[Product 11]",
	}

	for cell, value := range values {
		if err := f.SetCellValue("Customer Costing", cell, value); err != nil {
			t.Fatalf("failed to set %s: %v", cell, err)
		}
	}

	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("failed to save workbook: %v", err)
	}

	data, err := ParseCostingSheet(filePath)
	if err != nil {
		t.Fatalf("ParseCostingSheet failed: %v", err)
	}

	if data.Metadata.Date != "2026-03-15" {
		t.Fatalf("expected normalized date 2026-03-15, got %q", data.Metadata.Date)
	}
	if len(data.LineItems) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(data.LineItems))
	}

	item := data.LineItems[0]
	if item.Quantity != 5 {
		t.Fatalf("expected quantity 5, got %.3f", item.Quantity)
	}
	if item.SuggestedPriceBHD != 460 {
		t.Fatalf("expected suggested price 460, got %.3f", item.SuggestedPriceBHD)
	}
	if item.TotalSuggestedBHD != 2300 {
		t.Fatalf("expected line total 2300, got %.3f", item.TotalSuggestedBHD)
	}

	if data.Totals.Subtotal != 2300 {
		t.Fatalf("expected subtotal 2300, got %.3f", data.Totals.Subtotal)
	}
	if data.Totals.VatAmount != 230 {
		t.Fatalf("expected vat 230, got %.3f", data.Totals.VatAmount)
	}
	if data.Totals.GrandTotal != 2530 {
		t.Fatalf("expected grand total 2530, got %.3f", data.Totals.GrandTotal)
	}
}

func TestParseCostingSheet_LegacyCostingSheetLayout(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "legacy_costing.xlsx")
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Costing Sheet")

	values := map[string]any{
		"A7":  "Supplier",
		"A8":  "Equipment",
		"A9":  "Model",
		"A10": "Specification / Order Code",
		"A11": "Currency",
		"B11": "Quantity",
		"A12": "EUR",
		"B12": "FOB",
		"A13": "EUR",
		"B13": "Freight",
		"B14": "Exchange Rate to BHD",
		"A15": "BHD",
		"B15": "FOB",
		"A16": "BHD",
		"B16": "Freight",
		"B17": "C&F",
		"B18": "Insurance",
		"B19": "Customs @ 6%",
		"B20": "Landed Cost",
		"B21": "Handling @ 4%",
		"B22": "Finance Charges @1%",
		"B23": "Other Costs",
		"B24": "Total Cost",
		"B25": "Markup @ 20%",
		"A27": "Suggested Price per unit",
		"A28": "Total Suggested Price",
		"A31": "Summary",
		"C31": "Total",
		"D31": "VAT",
		"A32": "Total PO value expected from Client",
		"C32": 2614.0,
		"D32": 261.4,
		"C7":  "RH",
		"C8":  "REPLACEMENT DATALOGGER ELECTRONIC",
		"C9":  "6026604",
		"C11": 1.0,
		"C12": 4522.0,
		"C13": 406.98,
		"C14": 0.45,
		"C15": 2034.9,
		"C16": 183.141,
		"C17": 2218.041,
		"C18": 0.0,
		"C19": 110.90205,
		"C20": 2328.94305,
		"C21": 23.2894305,
		"C22": 23.2894305,
		"C23": 0.0,
		"C24": 2375.521911,
		"C25": 237.5521911,
		"C27": 2614.0,
		"C28": 2614.0,
		"M7":  "Total for Order",
		"M29": 261.4,
		"M30": 2875.4,
	}

	for cell, value := range values {
		if err := f.SetCellValue("Costing Sheet", cell, value); err != nil {
			t.Fatalf("failed to set %s: %v", cell, err)
		}
	}

	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("failed to save workbook: %v", err)
	}

	data, err := ParseCostingSheet(filePath)
	if err != nil {
		t.Fatalf("ParseCostingSheet failed: %v", err)
	}

	if len(data.LineItems) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(data.LineItems))
	}

	item := data.LineItems[0]
	if item.Quantity != 1 {
		t.Fatalf("expected quantity 1, got %.3f", item.Quantity)
	}
	if item.SuggestedPriceBHD != 2614 {
		t.Fatalf("expected suggested price 2614, got %.3f", item.SuggestedPriceBHD)
	}
	if item.TotalSuggestedBHD != 2614 {
		t.Fatalf("expected line total 2614, got %.3f", item.TotalSuggestedBHD)
	}

	if data.Totals.Subtotal != 2614 {
		t.Fatalf("expected subtotal 2614, got %.3f", data.Totals.Subtotal)
	}
	if data.Totals.VatAmount != 261.4 {
		t.Fatalf("expected vat 261.4, got %.3f", data.Totals.VatAmount)
	}
	if data.Totals.GrandTotal != 2875.4 {
		t.Fatalf("expected grand total 2875.4, got %.3f", data.Totals.GrandTotal)
	}
}

func TestParseCostingSheet_UsesModelWhenEquipmentPlaceholder(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "model_fallback.xlsx")
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Customer Costing")

	values := map[string]any{
		"A7":  "Supplier",
		"A8":  "Equipment",
		"A9":  "Model",
		"A10": "Specification / Order Code",
		"A11": "Currency",
		"B11": "Quantity",
		"A12": "EUR",
		"B12": "FOB",
		"B13": "Total Price",
		"A14": "EUR",
		"B14": "Freight",
		"B17": "Exchange Rate to BHD",
		"A18": "BHD",
		"B18": "FOB",
		"A19": "BHD",
		"B19": "Freight",
		"B20": "C&F",
		"B21": "Insurance",
		"B22": "Customs @ 6%",
		"B23": "Landed Cost",
		"B24": "Handling @ 4%",
		"B25": "Finance Charges @1%",
		"B26": "Other Costs",
		"B27": "Total Cost",
		"B28": "Markup @ 20%",
		"A29": "Selling Price",
		"A30": "Suggested Price per unit",
		"A31": "Total Suggested Price",
		"A34": "Summary",
		"C34": "Total",
		"D34": "VAT",
		"A35": "Total PO value expected from Client",
		"C35": 9000.0,
		"D35": 900.0,
		"C7":  "RH",
		"C8":  "[Product 1]",
		"C9":  "FlowMag W 400",
		"C10": "Electromagnetic flowmeter",
		"C11": 1.0,
		"C12": 13384.58,
		"C13": 13384.58,
		"C14": 1204.6122,
		"C17": 0.45,
		"C18": 6023.061,
		"C19": 542.07549,
		"C20": 6565.13649,
		"C22": 328.2568245,
		"C23": 6893.3933145,
		"C24": 137.86786629,
		"C25": 68.933933145,
		"C27": 7100.195113935,
		"C28": 1881.5517051927752,
		"C29": 8981.746819127775,
		"C30": 9000.0,
		"C31": 9000.0,
	}

	for cell, value := range values {
		if err := f.SetCellValue("Customer Costing", cell, value); err != nil {
			t.Fatalf("failed to set %s: %v", cell, err)
		}
	}

	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("failed to save workbook: %v", err)
	}

	data, err := ParseCostingSheet(filePath)
	if err != nil {
		t.Fatalf("ParseCostingSheet failed: %v", err)
	}

	if len(data.LineItems) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(data.LineItems))
	}
	if data.LineItems[0].Equipment != "FlowMag W 400" {
		t.Fatalf("expected equipment fallback to model, got %q", data.LineItems[0].Equipment)
	}
	if data.LineItems[0].SuggestedPriceBHD != 9000 {
		t.Fatalf("expected suggested price 9000, got %.3f", data.LineItems[0].SuggestedPriceBHD)
	}
}

func TestParseCostingSheet_SkipsNumericOnlyTemplateColumn(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "numeric_only.xlsx")
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Customer Costing")

	values := map[string]any{
		"A7":  "Supplier",
		"A8":  "Equipment",
		"A9":  "Model",
		"A10": "Specification / Order Code",
		"A11": "Currency",
		"B11": "Quantity",
		"A12": "EUR",
		"B12": "FOB",
		"B13": "Total Price",
		"A14": "EUR",
		"B14": "Freight",
		"B17": "Exchange Rate to BHD",
		"A18": "BHD",
		"B18": "FOB",
		"A19": "BHD",
		"B19": "Freight",
		"B20": "C&F",
		"B21": "Insurance",
		"B22": "Customs @ 6%",
		"B23": "Landed Cost",
		"B24": "Handling @ 4%",
		"B25": "Finance Charges @1%",
		"B26": "Other Costs",
		"B27": "Total Cost",
		"B28": "Markup @ 20%",
		"A29": "Selling Price",
		"A30": "Suggested Price per unit",
		"A31": "Total Suggested Price",
		"A34": "Summary",
		"C34": "Total",
		"D34": "VAT",
		"A35": "Total PO value expected from Client",
		"C35": 48403.3,
		"D35": 4840.33,
		"C7":  "RH",
		"C8":  "[Product 1]",
		"C9":  "",
		"C10": "[Specification 1]",
		"C11": 1.0,
		"C12": 73630.0,
		"C13": 73630.0,
		"C17": 0.45,
		"C18": 33133.5,
		"C20": 33133.5,
		"C22": 1656.675,
		"C23": 34790.175,
		"C24": 347.90175,
		"C25": 347.90175,
		"C27": 35485.9785,
		"C28": 8516.63484,
		"C29": 44002.61334,
		"C30": 44003.0,
		"C31": 44003.0,
		"O7":  "Total for Order",
	}

	for cell, value := range values {
		if err := f.SetCellValue("Customer Costing", cell, value); err != nil {
			t.Fatalf("failed to set %s: %v", cell, err)
		}
	}

	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("failed to save workbook: %v", err)
	}

	data, err := ParseCostingSheet(filePath)
	if err != nil {
		t.Fatalf("ParseCostingSheet failed: %v", err)
	}

	if len(data.LineItems) != 0 {
		t.Fatalf("expected numeric-only template column to be skipped, got %d line items", len(data.LineItems))
	}
	if data.Totals.Subtotal != 48403.3 {
		t.Fatalf("expected subtotal to remain available, got %.3f", data.Totals.Subtotal)
	}
}

func TestExtractTotals_PrefersWideSummaryBandOverLegacyFallbackCells(t *testing.T) {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Customer Costing")

	values := map[string]any{
		"A31": "Total Suggested Price",
		"A35": "Total PO value expected from Client",
		"C31": 1437.0,
		"D31": 244.0,
		"M30": 118.0,
		"C35": 17300.0,
		"D35": 1730.0,
		"E35": 19030.0,
		"G35": 411.0,
		"K35": 6.89643,
	}

	for cell, value := range values {
		if err := f.SetCellValue("Customer Costing", cell, value); err != nil {
			t.Fatalf("failed to set %s: %v", cell, err)
		}
	}

	totals := extractTotals(f, "Customer Costing", []ExcelCostingLineItem{
		{TotalSuggestedBHD: 1437},
		{TotalSuggestedBHD: 244},
	})

	if totals.Subtotal != 17300 {
		t.Fatalf("expected subtotal 17300, got %.3f", totals.Subtotal)
	}
	if totals.VatAmount != 1730 {
		t.Fatalf("expected vat 1730, got %.3f", totals.VatAmount)
	}
	if totals.GrandTotal != 19030 {
		t.Fatalf("expected grand total 19030, got %.3f", totals.GrandTotal)
	}
}
