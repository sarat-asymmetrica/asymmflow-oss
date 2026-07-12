package engines

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGenerateEWAInvoice_FromRoot creates a sample North Grid invoice PDF (E2E test)
func TestGenerateEWAInvoice_FromRoot(t *testing.T) {
	// Create test output directory
	outputDir := "test_output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Sample North Grid invoice data (from Invoice format.md)
	invoiceData := &InvoiceData{
		// Header
		InvoiceType:   "TAX INVOICE",
		InvoiceNumber: "INV-2025-0106",
		InvoiceDate:   time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
		TRN:           "990000000000000",

		// Buyer (North Grid Authority)
		BuyerName:      "North Grid Authority",
		BuyerBuilding:  "0",
		BuyerRoad:      "1702",
		BuyerTown:      "Manama",
		BuyerBlock:     "317",
		BuyerCountry:   "Kingdom of Bahrain",
		BuyerTRN:       "990000000000002",
		BuyerOrderNo:   "4500157334",
		BuyerOrderDate: time.Date(2025, 5, 28, 0, 0, 0, 0, time.UTC),

		// Delivery
		DeliveryNote:     "EDD/SLA/AIM JUN-JUL 25",
		DeliveryNoteDate: time.Date(2025, 7, 27, 0, 0, 0, 0, time.UTC),
		PaymentTerms:     "30 Days",
		ModeOfPayment:    "Bank Transfer",
		Destination:      "BAHRAIN",
		TermsOfDelivery:  "Direct Delivery to North Grid Metering Section Bahrain",

		// Line items (from the invoice format example)
		Items: []InvoiceItem{
			{
				SlNo:          1,
				Description:   "SLA L&G AIM GRIDSTREAM FOR JULY 2025",
				Quantity:      171425,
				Unit:          "Each",
				Rate:          0.232,
				AnnualTotal:   39770.600,
				MonthlyAmount: 3314.217,
				VATPercent:    10,
				TaxableValue:  3314.217,
				VAT:           331.422,
				Total:         3645.639,
			},
			{
				SlNo:          2,
				Description:   "Support Service 24/7 for July 2025",
				Quantity:      1,
				Unit:          "Service",
				Rate:          1150.000,
				AnnualTotal:   13800.000,
				MonthlyAmount: 1150.000,
				VATPercent:    10,
				TaxableValue:  1150.000,
				VAT:           115.000,
				Total:         1265.000,
			},
		},

		// Totals
		Subtotal:       4464.217,
		TotalVAT:       446.422,
		GrandTotal:     4910.639,
		Currency:       "BHD",
		CurrencySymbol: "B.D",

		// Additional
		SupplierRef:     "EDD/SLA/AIM JUN-JUL 25",
		OtherReferences: "AIM SLA",
		DespatchDoc:     "EDD/SLA/AIM JUN-JUL 25",
		DespatchMethod:  "DIRECT",
		PlaceOfSupply:   "Kingdom of Bahrain",
	}

	// Path to letterhead (if exists)
	letterheadPath := filepath.Join("..", "..", "data/ssot", "Acme Instrumentation Letterhead.png") // asset filename kept (out-of-scope build asset)

	// Create PDF generator
	generator, err := NewPDFGenerator(letterheadPath)
	if err != nil {
		t.Fatalf("Failed to create PDF generator: %v", err)
	}

	// Generate PDF
	outputPath := filepath.Join(outputDir, "NorthGrid_Invoice_INV_2025_0106_Test.pdf")
	err = generator.Generate(invoiceData, outputPath)
	if err != nil {
		t.Fatalf("Failed to generate PDF: %v", err)
	}

	// Validate PDF was created
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("PDF file was not created: %v", err)
	}

	// Validate PDF is non-empty
	if info.Size() == 0 {
		t.Fatalf("Generated PDF is empty (0 bytes)")
	}

	t.Logf("✓ PDF generated successfully: %s (%.2f KB)", outputPath, float64(info.Size())/1024)
}

// TestGenerateMinimalInvoice_FromRoot tests with minimal data
func TestGenerateMinimalInvoice_FromRoot(t *testing.T) {
	outputDir := "test_output"
	os.MkdirAll(outputDir, 0755)

	// Minimal invoice data
	invoiceData := &InvoiceData{
		InvoiceType:   "QUOTATION",
		InvoiceNumber: "INV-2025-TEST-001",
		InvoiceDate:   time.Now(),
		TRN:           "990000000000000",

		BuyerName:    "Test Customer",
		BuyerCountry: "Kingdom of Bahrain",

		Items: []InvoiceItem{
			{
				SlNo:         1,
				Description:  "Test Item",
				Quantity:     1,
				Unit:         "Each",
				Rate:         100.000,
				TaxableValue: 100.000,
				VATPercent:   10,
				VAT:          10.000,
				Total:        110.000,
			},
		},

		Subtotal:       100.000,
		TotalVAT:       10.000,
		GrandTotal:     110.000,
		Currency:       "BHD",
		CurrencySymbol: "B.D",
		PaymentTerms:   "30 Days",
	}

	generator, err := NewPDFGenerator("") // No letterhead
	if err != nil {
		t.Fatalf("Failed to create PDF generator: %v", err)
	}

	outputPath := filepath.Join(outputDir, "Minimal_Test_Invoice.pdf")
	err = generator.Generate(invoiceData, outputPath)
	if err != nil {
		t.Fatalf("Failed to generate minimal PDF: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Minimal PDF file was not created: %v", err)
	}

	if info.Size() == 0 {
		t.Fatalf("Generated minimal PDF is empty")
	}

	t.Logf("✓ Minimal PDF generated successfully: %s (%.2f KB)", outputPath, float64(info.Size())/1024)
}

// TestMultipleInvoices_FromRoot tests generating multiple invoices in sequence
func TestMultipleInvoices_FromRoot(t *testing.T) {
	outputDir := "test_output"
	os.MkdirAll(outputDir, 0755)

	for i := 1; i <= 3; i++ {
		invoiceData := &InvoiceData{
			InvoiceType:   "TAX INVOICE",
			InvoiceNumber: "INV-2025-BATCH-" + string(rune('0'+i)),
			InvoiceDate:   time.Now(),
			TRN:           "990000000000000",
			BuyerName:     "Batch Customer",
			BuyerCountry:  "Kingdom of Bahrain",
			Items: []InvoiceItem{
				{
					SlNo:         1,
					Description:  "Batch Item",
					Quantity:     float64(i),
					Rate:         100.000,
					TaxableValue: 100.000 * float64(i),
					VATPercent:   10,
					VAT:          10.000 * float64(i),
					Total:        110.000 * float64(i),
				},
			},
			Subtotal:       100.000 * float64(i),
			TotalVAT:       10.000 * float64(i),
			GrandTotal:     110.000 * float64(i),
			Currency:       "BHD",
			CurrencySymbol: "B.D",
			PaymentTerms:   "30 Days",
		}

		generator, _ := NewPDFGenerator("")
		outputPath := filepath.Join(outputDir, "Batch_Invoice_"+string(rune('0'+i))+".pdf")
		err := generator.Generate(invoiceData, outputPath)
		if err != nil {
			t.Fatalf("Failed to generate batch invoice %d: %v", i, err)
		}

		t.Logf("✓ Generated batch invoice %d", i)
	}

	t.Logf("✓ All 3 batch invoices generated successfully")
}
