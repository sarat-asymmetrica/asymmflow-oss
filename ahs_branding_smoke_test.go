package main

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAHSDocumentExportsUseAHSTradingBranding(t *testing.T) {
	app := setupTestApp(t)

	require.NoError(t, app.db.AutoMigrate(
		&Offer{},
		&OfferItem{},
		&PurchaseOrder{},
		&PurchaseOrderItem{},
		&SupplierInvoice{},
		&SupplierInvoiceItem{},
		&CreditNote{},
		&CreditNoteItem{},
	))

	now := time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC)

	customer := CustomerMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		BusinessName: "AHS Smoke Customer",
		CustomerCode: "AHS-CUST-001",
		CustomerID:   "AHS-CUST-001",
		AddressLine1: "Manama, Kingdom of Bahrain",
		TRN:          "990000000000002",
		Status:       "Active",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	supplier := SupplierMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		SupplierName: "AHS Smoke Supplier",
		SupplierCode: "AHS-SUP-001",
		Address:      "Sitra Industrial Area",
		Country:      "Kingdom of Bahrain",
		TaxID:        "SUP-TRN-001",
	}
	require.NoError(t, app.db.Create(&supplier).Error)

	order := Order{
		Base:          Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		OrderNumber:   "AHS-ORD-001",
		CustomerID:    customer.ID,
		CustomerName:  customer.BusinessName,
		OrderDate:     now,
		RequiredDate:  now.AddDate(0, 0, 14),
		TotalValueBHD: 100,
		GrandTotalBHD: 110,
		Status:        "Processing",
		PaymentTerms:  "30 Days",
		DeliveryTerms: "DAP Bahrain",
		Division:      "Beacon Controls",
	}
	require.NoError(t, app.db.Create(&order).Error)

	invoice := Invoice{
		Base:           Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		InvoiceNumber:  "AHS-INV-001",
		InvoiceDate:    now,
		CustomerID:     customer.ID,
		CustomerName:   customer.BusinessName,
		OrderID:        order.ID,
		GrandTotalBHD:  110,
		OutstandingBHD: 110,
		SubtotalBHD:    100,
		DueDate:        now.AddDate(0, 0, 30),
		Status:         "Sent",
		Division:       "Beacon Controls",
		VATPercent:     10,
		VATBHD:         10,
		PaymentTerms:   "30 Days",
	}
	require.NoError(t, app.db.Create(&invoice).Error)
	require.NoError(t, app.db.Create(&DBInvoiceItem{
		Base:        Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		InvoiceID:   invoice.ID,
		LineNumber:  1,
		Description: "AHS invoice line",
		Quantity:    1,
		Rate:        100,
		TotalBHD:    100,
	}).Error)

	offer := Offer{
		Base:              Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		OfferNumber:       "AHS-OFF-001",
		RFQID:             "rfq-ahs-001",
		CustomerID:        customer.ID,
		CustomerName:      customer.BusinessName,
		QuotationDate:     now,
		ValidityDate:      now.AddDate(0, 0, 30),
		Stage:             "Quoted",
		PaymentTerms:      "30 days from Date of Delivery",
		DeliveryTerms:     "DAP Bahrain at your store or Beacon Controls",
		DeliveryWeeks:     "5-7 weeks",
		IssuedBy:          "Sales Test",
		CustomerReference: "AL-009",
		AttentionPerson:   "Arif",
		AttentionCompany:  customer.BusinessName,
		QuoteType:         "Quotation",
		VatRate:           10,
		Division:          "Beacon Controls",
	}
	require.NoError(t, app.db.Create(&offer).Error)
	require.NoError(t, app.db.Create(&OfferItem{
		Base:                Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		OfferID:             offer.ID,
		LineNumber:          1,
		Description:         "Flowmeter-water",
		ProductCode:         "00112FNAPER",
		LongCode:            "FMU90-F41230S",
		DetailedDescription: "AHS smoke test specification",
		Quantity:            1,
		UnitPrice:           85,
		TotalPrice:          85,
	}).Error)

	po := PurchaseOrder{
		Base:             Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		OrderID:          order.ID,
		SupplierID:       supplier.ID,
		SupplierName:     supplier.SupplierName,
		PONumber:         "AHS-PO-001",
		PODate:           now,
		ExpectedDelivery: now.AddDate(0, 0, 21),
		Currency:         "BHD",
		ExchangeRate:     1,
		SubtotalBHD:      100,
		VATAmount:        10,
		TotalBHD:         110,
		Status:           "Approved",
		PaymentTerms:     "30 Days",
		Division:         "Beacon Controls",
	}
	require.NoError(t, app.db.Create(&po).Error)
	require.NoError(t, app.db.Create(&PurchaseOrderItem{
		Base:            Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		PurchaseOrderID: po.ID,
		Description:     "AHS PO line",
		Quantity:        1,
		UnitPriceBHD:    100,
		TotalBHD:        100,
	}).Error)

	supplierInvoice := SupplierInvoice{
		Base:            Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		SupplierID:      supplier.ID,
		SupplierName:    supplier.SupplierName,
		PurchaseOrderID: po.ID,
		PONumber:        po.PONumber,
		OrderID:         order.ID,
		InvoiceNumber:   "AHS-SINV-001",
		InvoiceDate:     now,
		DueDate:         now.AddDate(0, 0, 30),
		Currency:        "BHD",
		ExchangeRate:    1,
		SubtotalForeign: 100,
		SubtotalBHD:     100,
		VATForeign:      10,
		VATBHD:          10,
		TotalForeign:    110,
		TotalBHD:        110,
		MatchStatus:     "Matched",
		Status:          "Approved",
		PaymentStatus:   "Unpaid",
		Division:        "Beacon Controls",
	}
	require.NoError(t, app.db.Create(&supplierInvoice).Error)

	creditNote := CreditNote{
		Base:             Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		CreditNoteNumber: "AHS-CN-001",
		CreditNoteDate:   now,
		InvoiceID:        invoice.ID,
		InvoiceNumber:    invoice.InvoiceNumber,
		CustomerID:       customer.ID,
		CustomerName:     customer.BusinessName,
		Reason:           "AHS adjustment",
		SubtotalBHD:      10,
		VATBHD:           1,
		VATPercent:       10,
		GrandTotalBHD:    11,
		Status:           "Issued",
	}
	require.NoError(t, app.db.Create(&creditNote).Error)
	require.NoError(t, app.db.Create(&CreditNoteItem{
		Base:         Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test-user"},
		CreditNoteID: creditNote.ID,
		LineNumber:   1,
		Description:  "AHS credit note line",
		Quantity:     1,
		Rate:         10,
		TotalBHD:     10,
	}).Error)

	require.Equal(t, "Beacon Controls", app.resolvePurchaseOrderDivision(po))
	require.Equal(t, "Beacon Controls", app.resolveCreditNoteDivision(creditNote))
	require.Equal(t, "Beacon Controls", app.resolveSupplierInvoiceDivision(supplierInvoice))
	require.Equal(t, "BEACON CONTROLS W.L.L.", companyDocumentProfile("Beacon Controls").LegalName)
	// In the open-source build no branded artwork is bundled, so the Beacon Controls
	// division resolves its letterhead from the generated placeholder asset
	// (letterhead_ahs) rather than a bundled "Beacon Controls Letterhead" file.
	require.Contains(t, app.letterheadImagePathForDivision("Beacon Controls"), AssetLetterheadAHS)

	exports := map[string]func() (string, error){
		"offer":            func() (string, error) { return app.GenerateOfferPDF(offer.ID) },
		"invoice":          func() (string, error) { return app.GenerateInvoicePDF(invoice.ID) },
		"purchase_order":   func() (string, error) { return app.GeneratePurchaseOrderPDF(po.ID) },
		"supplier_invoice": func() (string, error) { return app.GenerateSupplierInvoicePDF(supplierInvoice.ID) },
		"credit_note":      func() (string, error) { return app.GenerateCreditNotePDF(creditNote.ID) },
	}

	for label, generate := range exports {
		path, err := generate()
		require.NoError(t, err, "%s export should succeed", label)
		t.Cleanup(func() {
			if path != "" {
				_ = os.Remove(path)
			}
		})

		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr, "%s export should be readable", label)
		require.NotEmpty(t, data, "%s export should not be empty", label)
		require.Contains(t, string(data), "/Subtype /Image", "%s export should embed the company letterhead image", label)
	}
}

func TestAHSCostingExportEmbedsLetterhead(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := setupTestApp(t)

	exportPath, err := app.ExportCostingToPDF(CostingExportData{
		Division:      "Beacon Controls",
		Date:          "2026-04-14",
		PreparedBy:    "Sales Test",
		CustomerName:  "RIVERSIDE POWER OPERATION AND MAINTENANCE COMPANY W.L.L",
		ContactPerson: "Arif",
		RfqReference:  "009",
		CostingId:     "Offer 21-26 AL",
		Subject:       "RIVERSIDE POWER OPERATION AND MAINTENANCE COMPANY W.L.L",
		EstDelivery:   "5-7 weeks",
		DeliveryTerms: "DAP Bahrain at your store or Beacon Controls",
		PaymentTerms:  "30 days from Date of Delivery",
		QuoteType:     "Quotation",
		VatRate:       10,
		LineItems: []CostingExportLineItem{
			{
				SlNo:           1,
				Equipment:      "Flowmeter-water",
				Model:          "00112FNAPER",
				Quantity:       1,
				SuggestedPrice: 85,
				TotalPrice:     85,
			},
		},
		Subtotal:   85,
		VAT:        8.5,
		GrandTotal: 93.5,
		Body:       "We thank you for the opportunity and are pleased to submit our techno-commercial offer for your review.",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if exportPath != "" {
			_ = os.Remove(exportPath)
		}
	})

	data, readErr := os.ReadFile(exportPath)
	require.NoError(t, readErr)
	require.Contains(t, string(data), "/Subtype /Image", "costing export should embed the company letterhead image")
}

func TestCostingPDFWrapsLongCommercialDescription(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not available for PDF text-order regression")
	}
	t.Setenv("HOME", t.TempDir())

	app := setupTestApp(t)

	exportPath, err := app.ExportCostingToPDF(CostingExportData{
		Division:      "Acme Instrumentation",
		Date:          "2026-04-27",
		PreparedBy:    "Sales Test",
		CustomerName:  "BlueWave Marine Operations LLC",
		Subject:       "Unit Price",
		EstDelivery:   "5-7 weeks",
		DeliveryTerms: "DAP Bahrain at your store or Acme Instrumentation",
		PaymentTerms:  "30 days from Date of Delivery",
		QuoteType:     "Quotation",
		VatRate:       10,
		LineItems: []CostingExportLineItem{
			{
				SlNo:           1,
				Equipment:      "Differential Pressure Indicator Transmitter Deltabar FMD78, Rhine Instruments, NEMA4X/6P IP66/IP67, with SW Filter-A DP",
				Model:          "FMD78-AEA7HE3AFMBW+PDSCZ1",
				Quantity:       1,
				SuggestedPrice: 540,
				TotalPrice:     540,
			},
		},
		Subtotal:   540,
		VAT:        54,
		GrandTotal: 594,
		Body:       "Please find our pricing and scope below.",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if exportPath != "" {
			_ = os.Remove(exportPath)
		}
	})

	output, err := exec.Command("pdftotext", exportPath, "-").Output()
	require.NoError(t, err)
	text := string(output)
	require.NotContains(t, text, "540.000IP66", "description text must not run into the unit price column")
	require.Contains(t, text, "FMD78-AEA7HE3AFMBW+PDSCZ1")
}
