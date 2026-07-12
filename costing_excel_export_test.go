package main

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestStandaloneCostingWorkbookIncludesDetailedCostBreakdown(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "costing.xlsx")
	data := CostingExportData{
		Division:           "Acme Instrumentation",
		OfferNumber:        "OFFER-1",
		Date:               "2026-04-27",
		PreparedBy:         "Jordan",
		CustomerID:         "CUST-1",
		CustomerName:       "Example Customer WLL",
		ContactPerson:      "Buyer",
		RfqReference:       "RFQ-1",
		FolderNumber:       "10-26",
		CostingId:          "COST-1",
		Subject:            "Sub: Example",
		EstDelivery:        "5-7 weeks",
		DeliveryTerms:      "DAP Bahrain",
		PaymentTerms:       "30 days",
		OrderType:          "General",
		CountryOfOrigin:    "DE",
		CocCoo:             "Yes",
		TestCertificate:    "Included",
		Installation:       "No",
		Commissioning:      "No",
		Testing:            "No",
		QuoteType:          "Quotation",
		VatRate:            10,
		HiddenCharges:      25.500,
		PlaceOfSupply:      "Kingdom of Bahrain",
		TaxCategory:        "Standard",
		CustomerTRN:        "TRN-123",
		Body:               "Customer editable PDF body.",
		Subtotal:           1200,
		Discount:           50,
		NetAmount:          1150,
		VAT:                115,
		GrandTotal:         1265,
		TotalCost:          800,
		Profit:             350,
		ProfitPercent:      30.43,
		ProjectName:        "Example Project",
		TermsAndConditions: "Terms stay attached.",
		LineItems: []CostingExportLineItem{
			{
				SlNo:                1,
				Supplier:            "EH",
				Equipment:           "Pressure Transmitter",
				Model:               "PT-100",
				LongCode:            "LC-123",
				Specification:       "Range 0-10 bar",
				DetailedDescription: "Detailed multiline technical description",
				Currency:            "EUR",
				Quantity:            2,
				FOB:                 100,
				Freight:             9,
				FreightPercent:      9,
				TotalCost:           180,
				MarkupPercent:       20,
				SuggestedPrice:      220,
				TotalPrice:          440,
				ExchangeRate:        0.45,
				FobBHD:              45,
				FreightBHD:          4.05,
				Insurance:           2,
				CustomsPercent:      5,
				CustomsBHD:          2.25,
				HandlingPercent:     4,
				HandlingBHD:         1.8,
				FinancePercent:      1,
				FinanceBHD:          0.45,
				OtherCosts:          10,
				UserPrice:           230,
				UserPriceSet:        true,
			},
		},
	}

	if err := writeStandaloneCostingWorkbook(data, outputPath); err != nil {
		t.Fatalf("writeStandaloneCostingWorkbook() error = %v", err)
	}

	workbook, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer workbook.Close()

	expectedCells := map[string]string{
		"A3":   "Header and Commercial Terms",
		"A16":  "Sl No",
		"M16":  "Freight %",
		"W16":  "Extra Cost",
		"X16":  "Unit PH Cost BHD",
		"Y16":  "Total PH Cost BHD",
		"AB16": "Manual Unit Price",
		"AC16": "Manual Price Used",
		"C17":  "Pressure Transmitter",
		"G17":  "Detailed multiline technical description",
		"AC17": "Yes",
	}
	for cell, expected := range expectedCells {
		actual, err := workbook.GetCellValue("Detailed Costing", cell)
		if err != nil {
			t.Fatalf("GetCellValue(%s) error = %v", cell, err)
		}
		if actual != expected {
			t.Fatalf("Detailed Costing!%s = %q, want %q", cell, actual, expected)
		}
	}
}
