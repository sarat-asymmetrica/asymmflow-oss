package main

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"ph_holdings_app/pkg/compliance/india"
	"ph_holdings_app/pkg/documents/numbering"
	"ph_holdings_app/pkg/overlay"
)

// India Spec-01 B4 (document emission) tests. Canon fixtures per
// SYNTHETIC_IDENTITY.md "India demo canon": Meridian Instruments & Controls
// Pvt Ltd (two divisions, same PAN, different GST states — Mumbai/
// Maharashtra state 27, Bengaluru/Karnataka state 29) and Kaveri Trade Links
// (Karnataka state 29, composition). All fictional.

// withIndiaOverlay loads the named india-demo overlay directory as the
// active overlay for the duration of the test (mirrors
// bank_accounts_seed_gate_test.go / offer_signature_blocks_test.go's
// swap-and-restore pattern), keeping both the package-main activeOverlay
// var and the pkg/overlay singleton in sync via setActiveOverlay.
func withIndiaOverlay(t *testing.T, dir string) *overlay.CompanyOverlay {
	t.Helper()
	saved := activeOverlay
	t.Cleanup(func() { setActiveOverlay(saved) })

	ov := overlay.LoadOverlay([]string{dir})
	require.True(t, ov.IndiaMounted(), "fixture overlay %s must mount the India plane", dir)
	setActiveOverlay(ov)
	return ov
}

// synthGSTIN builds a checksum-valid but entirely fictional customer GSTIN
// for the given state code, so buyer-side fixtures pass the same format the
// real GSTN check-digit algorithm expects without inventing a real
// registration (SYNTHETIC_IDENTITY.md public-repo law).
func synthGSTIN(t *testing.T, stateCode string) string {
	t.Helper()
	gstin, err := india.MakeGSTIN(stateCode, "AAAAA0000A", '1')
	require.NoError(t, err)
	return gstin
}

func migrateIndiaDocTables(t *testing.T, app *App) {
	t.Helper()
	require.NoError(t, app.db.AutoMigrate(&CreditNote{}, &CreditNoteItem{}))
}

// TestIndiaTaxInvoiceIntraStateCGSTSGST covers the happy path for an
// intra-state supply: Meridian Mumbai (state 27) selling to a Maharashtra
// B2B buyer (also state 27) splits tax CGST+SGST, never IGST.
func TestIndiaTaxInvoiceIntraStateCGSTSGST(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	customer := CustomerMaster{
		Base: Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		BusinessName: "Sahyadri Process Equipment Pvt Ltd", CustomerCode: "SPE-01", CustomerID: "SPE-01",
		AddressLine1: "MIDC Industrial Area", City: "Pune", Country: "India", Status: "Active",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	invoice := Invoice{
		Base:                   Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceNumber:          "INV/26-27/001",
		InvoiceDate:            now,
		DueDate:                now.AddDate(0, 0, 30),
		CustomerID:             customer.ID,
		CustomerName:           customer.BusinessName,
		Status:                 "Sent",
		Division:               "Meridian Mumbai",
		BuyerGSTIN:             synthGSTIN(t, "27"),
		PlaceOfSupplyStateCode: "27",
		SubtotalBHD:            10000,
		VATPercent:             18,
		VATBHD:                 1800,
		GrandTotalBHD:          11800,
		OutstandingBHD:         11800,
	}
	require.NoError(t, app.db.Create(&invoice).Error)
	require.NoError(t, app.db.Create(&DBInvoiceItem{
		Base: Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceID: invoice.ID, LineNumber: 1,
		Description: "Flow transmitter", Quantity: 2, Rate: 5000, TotalBHD: 10000,
		HSNCode: "9026", UQC: "NOS",
	}).Error)

	path, err := app.GenerateInvoicePDF(invoice.ID)
	require.NoError(t, err)
	require.NotEmpty(t, path)
	t.Cleanup(func() { _ = os.Remove(path) })

	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.NotEmpty(t, data)

	text, ok := pdfText(t, data)
	if !ok {
		t.Skip("pdftotext not available for content assertions")
	}
	require.Contains(t, text, "TAX INVOICE")
	require.Contains(t, text, "GSTIN")
	require.Contains(t, text, "27AABCM0472E1ZT") // Meridian Mumbai's own GSTIN
	require.Contains(t, text, "CGST")
	require.Contains(t, text, "SGST")
}

// TestIndiaTaxInvoiceInterStateIGST covers the inter-state twin: Meridian
// Mumbai (state 27) selling to Charminar Engineering Co, Telangana (state
// 36) splits tax as IGST only.
func TestIndiaTaxInvoiceInterStateIGST(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	customer := CustomerMaster{
		Base: Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		BusinessName: "Charminar Engineering Co", CustomerCode: "CEC-01", CustomerID: "CEC-01",
		AddressLine1: "Hitech City", City: "Hyderabad", Country: "India", Status: "Active",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	invoice := Invoice{
		Base:                   Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceNumber:          "INV/26-27/002",
		InvoiceDate:            now,
		DueDate:                now.AddDate(0, 0, 30),
		CustomerID:             customer.ID,
		CustomerName:           customer.BusinessName,
		Status:                 "Sent",
		Division:               "Meridian Mumbai",
		BuyerGSTIN:             synthGSTIN(t, "36"),
		PlaceOfSupplyStateCode: "36",
		SubtotalBHD:            10000,
		VATPercent:             18,
		VATBHD:                 1800,
		GrandTotalBHD:          11800,
		OutstandingBHD:         11800,
	}
	require.NoError(t, app.db.Create(&invoice).Error)
	require.NoError(t, app.db.Create(&DBInvoiceItem{
		Base: Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceID: invoice.ID, LineNumber: 1,
		Description: "Gate valve", Quantity: 1, Rate: 10000, TotalBHD: 10000,
		HSNCode: "8481", UQC: "NOS",
	}).Error)

	path, err := app.GenerateInvoicePDF(invoice.ID)
	require.NoError(t, err)
	require.NotEmpty(t, path)
	t.Cleanup(func() { _ = os.Remove(path) })

	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)

	text, ok := pdfText(t, data)
	if !ok {
		t.Skip("pdftotext not available for content assertions")
	}
	require.Contains(t, text, "TAX INVOICE")
	require.Contains(t, text, "IGST")
}

// TestIndiaBillOfSupplyEnforcedForComposition covers G6: Kaveri Trade Links
// is a composition taxable person, so it ALWAYS emits a Bill of Supply —
// enforced from profile.India.Composition, never trusting invoice.DocKind.
func TestIndiaBillOfSupplyEnforcedForComposition(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo/composition")

	now := time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC)
	customer := CustomerMaster{
		Base: Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		BusinessName: "Local Retail Buyer", CustomerCode: "LRB-01", CustomerID: "LRB-01",
		Status: "Active",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	// DocKind deliberately left "" (an ordinary caller's intent) — composition
	// must still force a Bill of Supply. This is the "enforce, don't trust"
	// assertion.
	invoice := Invoice{
		Base:                   Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceNumber:          "BOS/26-27/001",
		InvoiceDate:            now,
		DueDate:                now.AddDate(0, 0, 30),
		CustomerID:             customer.ID,
		CustomerName:           customer.BusinessName,
		Status:                 "Sent",
		Division:               "Kaveri Trade Links",
		PlaceOfSupplyStateCode: "29",
		SubtotalBHD:            5000,
		GrandTotalBHD:          5000,
		OutstandingBHD:         5000,
	}
	require.NoError(t, app.db.Create(&invoice).Error)
	require.NoError(t, app.db.Create(&DBInvoiceItem{
		Base: Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceID: invoice.ID, LineNumber: 1,
		Description: "Gate valve (resale)", Quantity: 5, Rate: 1000, TotalBHD: 5000,
		HSNCode: "8481", UQC: "NOS",
	}).Error)

	path, err := app.GenerateInvoicePDF(invoice.ID)
	require.NoError(t, err)
	require.NotEmpty(t, path)
	t.Cleanup(func() { _ = os.Remove(path) })

	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)

	text, ok := pdfText(t, data)
	if !ok {
		t.Skip("pdftotext not available for content assertions")
	}
	require.Contains(t, text, "BILL OF SUPPLY")
	require.NotContains(t, text, "TAX INVOICE")
	require.Contains(t, text, "not eligible to collect tax")
	require.NotContains(t, text, "CGST")
}

// TestIndiaInvoiceRefusesUnderHSNTierBoundary pins the refuse-to-generate
// doctrine (house law, CLAUDE.md invariant 5): an HSN that fails the B3
// engine's digit-count validation must surface as an error, never a
// silently wrong PDF.
func TestIndiaInvoiceRefusesUnderHSNTierBoundary(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	customer := CustomerMaster{
		Base: Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		BusinessName: "Sahyadri Process Equipment Pvt Ltd", CustomerCode: "SPE-01", CustomerID: "SPE-01",
		Status: "Active",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	invoice := Invoice{
		Base:                   Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceNumber:          "INV/26-27/003",
		InvoiceDate:            now,
		DueDate:                now.AddDate(0, 0, 30),
		CustomerID:             customer.ID,
		CustomerName:           customer.BusinessName,
		Status:                 "Sent",
		Division:               "Meridian Mumbai",
		BuyerGSTIN:             synthGSTIN(t, "27"),
		PlaceOfSupplyStateCode: "27",
		SubtotalBHD:            1000,
		GrandTotalBHD:          1000,
		OutstandingBHD:         1000,
	}
	require.NoError(t, app.db.Create(&invoice).Error)
	require.NoError(t, app.db.Create(&DBInvoiceItem{
		Base: Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceID: invoice.ID, LineNumber: 1,
		Description: "Under-declared item", Quantity: 1, Rate: 1000, TotalBHD: 1000,
		HSNCode: "84", UQC: "NOS", // 2-digit HSN on a B2B invoice: refused, needs 4
	}).Error)

	_, err := app.GenerateInvoicePDF(invoice.ID)
	require.Error(t, err)
	var hsnErr *india.HSNValidationError
	require.True(t, errors.As(err, &hsnErr), "expected *india.HSNValidationError, got %v", err)
}

// TestIndiaCreditNoteReferencesOriginalInvoice covers B4(c): a credit note
// against an India invoice renders the original invoice's number/date and
// the same HSN/tax-split columns.
func TestIndiaCreditNoteReferencesOriginalInvoice(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)
	customer := CustomerMaster{
		Base: Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		BusinessName: "Sahyadri Process Equipment Pvt Ltd", CustomerCode: "SPE-02", CustomerID: "SPE-02",
		Status: "Active",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	invoice := Invoice{
		Base:                   Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceNumber:          "INV/26-27/004",
		InvoiceDate:            now,
		DueDate:                now.AddDate(0, 0, 30),
		CustomerID:             customer.ID,
		CustomerName:           customer.BusinessName,
		Status:                 "Sent",
		Division:               "Meridian Mumbai",
		BuyerGSTIN:             synthGSTIN(t, "27"),
		PlaceOfSupplyStateCode: "27",
		SubtotalBHD:            10000,
		VATPercent:             18,
		VATBHD:                 1800,
		GrandTotalBHD:          11800,
		OutstandingBHD:         11800,
	}
	require.NoError(t, app.db.Create(&invoice).Error)
	require.NoError(t, app.db.Create(&DBInvoiceItem{
		Base: Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		InvoiceID: invoice.ID, LineNumber: 1,
		Description: "Flow transmitter", Quantity: 2, Rate: 5000, TotalBHD: 10000,
		HSNCode: "9026", UQC: "NOS",
	}).Error)

	cn := CreditNote{
		Base:             Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		CreditNoteNumber: "CN/26-27/001",
		CreditNoteDate:   now,
		InvoiceID:        invoice.ID,
		InvoiceNumber:    invoice.InvoiceNumber,
		CustomerID:       customer.ID,
		CustomerName:     customer.BusinessName,
		Reason:           "Partial return",
		SubtotalBHD:      5000,
		VATBHD:           900,
		VATPercent:       18,
		GrandTotalBHD:    5900,
		Status:           "Issued",
		Division:         invoice.Division,
	}
	require.NoError(t, app.db.Create(&cn).Error)
	require.NoError(t, app.db.Create(&CreditNoteItem{
		Base: Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		CreditNoteID: cn.ID, LineNumber: 1,
		Description: "Flow transmitter (returned unit)", Quantity: 1, Rate: 5000, TotalBHD: 5000,
		HSNCode: "9026", UQC: "NOS",
	}).Error)

	path, err := app.GenerateCreditNotePDF(cn.ID)
	require.NoError(t, err)
	require.NotEmpty(t, path)
	t.Cleanup(func() { _ = os.Remove(path) })

	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)

	text, ok := pdfText(t, data)
	if !ok {
		t.Skip("pdftotext not available for content assertions")
	}
	require.Contains(t, text, "CREDIT NOTE")
	require.Contains(t, text, invoice.InvoiceNumber)
	require.Contains(t, text, "GSTIN")
}

// TestIndiaInvoiceNumberingPerGSTINPerFY pins R-A3-3: two divisions sharing
// one PAN (Meridian Mumbai/Bengaluru) get INDEPENDENT number sequences
// keyed by GSTIN, and a composition division is forced onto the Bill-of-
// Supply series regardless of caller intent.
func TestIndiaInvoiceNumberingPerGSTINPerFY(t *testing.T) {
	app := setupTestApp(t)
	withIndiaOverlay(t, "overlays/india-demo")

	mumbai1, mumbaiKind, err := app.generateInvoiceNumberWithTx(app.db, "Meridian Mumbai")
	require.NoError(t, err)
	require.Equal(t, "", mumbaiKind)
	require.Equal(t, "INV/26-27/001", mumbai1)

	// Bengaluru shares the PAN but has its own GSTIN: independent counter,
	// so it also starts at 001 rather than continuing Mumbai's sequence.
	bengaluru1, _, err := app.generateInvoiceNumberWithTx(app.db, "Meridian Bengaluru")
	require.NoError(t, err)
	require.Equal(t, "INV/26-27/001", bengaluru1)

	mumbai2, _, err := app.generateInvoiceNumberWithTx(app.db, "Meridian Mumbai")
	require.NoError(t, err)
	require.Equal(t, "INV/26-27/002", mumbai2)

	for _, n := range []string{mumbai1, bengaluru1, mumbai2} {
		require.NoError(t, numbering.ValidateGSTSeriesNumber(n))
	}
}

// TestIndiaCompositionNumberingForcesBillOfSupply pins the G6 numbering
// side: a composition division's generated number rides the BOS series and
// the returned docKind is stamped "bill_of_supply".
func TestIndiaCompositionNumberingForcesBillOfSupply(t *testing.T) {
	app := setupTestApp(t)
	withIndiaOverlay(t, "overlays/india-demo/composition")

	number, docKind, err := app.generateInvoiceNumberWithTx(app.db, "Kaveri Trade Links")
	require.NoError(t, err)
	require.Equal(t, "bill_of_supply", docKind)
	require.Equal(t, "BOS/26-27/001", number)
}

// TestIndiaCreditNoteNumberingPerGSTIN pins the credit-note twin of
// TestIndiaInvoiceNumberingPerGSTINPerFY.
func TestIndiaCreditNoteNumberingPerGSTIN(t *testing.T) {
	app := setupTestApp(t)
	withIndiaOverlay(t, "overlays/india-demo")

	first, err := app.generateCreditNoteNumberForDivision("Meridian Mumbai")
	require.NoError(t, err)
	require.Equal(t, "CN/26-27/001", first)

	second, err := app.generateCreditNoteNumberForDivision("Meridian Mumbai")
	require.NoError(t, err)
	require.Equal(t, "CN/26-27/002", second)
}

// TestGCCInvoiceNumberingUntouchedByIndiaRouting pins byte-identity: a GCC
// division (India == nil) still gets the unchanged INV-YYYYMMDD-NNNN scheme
// and an always-blank docKind.
func TestGCCInvoiceNumberingUntouchedByIndiaRouting(t *testing.T) {
	app := setupTestApp(t)
	// activeOverlay is BuiltinDefaults() by default (no India plane) —
	// exercised here explicitly for clarity, no swap needed.
	number, docKind, err := app.generateInvoiceNumberWithTx(app.db, "Acme Instrumentation")
	require.NoError(t, err)
	require.Equal(t, "", docKind)
	require.Regexp(t, `^INV-\d{8}-0001$`, number)
}

func TestIndianDigitGrouping(t *testing.T) {
	cases := map[float64]string{
		0:         "0.00",
		567:       "567.00",
		1234567:   "12,34,567.00",
		1234567.89: "12,34,567.89",
		100000:    "1,00,000.00",
	}
	for amount, want := range cases {
		require.Equal(t, want, indianDigitGrouping(amount), "amount=%v", amount)
	}
}

func TestAmountInWordsIndian(t *testing.T) {
	require.Equal(t, "Rupees Zero Only", amountInWordsIndian(0))
	require.Equal(t,
		"Rupees Twelve Lakh Thirty Four Thousand Five Hundred Sixty Seven Only",
		amountInWordsIndian(1234567))
	require.Equal(t,
		"Rupees One Hundred and Fifty Paise Only",
		amountInWordsIndian(100.50))
}
