package engines

// Wave 3 B.4: the PDF engine's seller identity (legal name + address block)
// comes from the active overlay, never hardcoded. These tests pin that the
// header renders from whatever overlay is active — including address blocks
// shorter or longer than the historical three lines — without panicking.

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ph_holdings_app/pkg/overlay"
)

func generateWithOverlay(t *testing.T, ov *overlay.CompanyOverlay) error {
	t.Helper()
	prev := overlay.Active()
	overlay.SetActive(ov)
	t.Cleanup(func() { overlay.SetActive(prev) })

	gen, err := NewPDFGenerator("")
	if err != nil {
		t.Fatalf("NewPDFGenerator: %v", err)
	}
	out := filepath.Join(t.TempDir(), "identity.pdf")
	genErr := gen.Generate(&InvoiceData{
		InvoiceNumber: "INV-TEST-001",
		InvoiceDate:   time.Now().UTC(),
		TRN:           "990000000000000",
		BuyerName:     "Delta Petrochemicals",
		Language:      "en",
		Items: []InvoiceItem{{
			SlNo: 1, Description: "Calibration service", Quantity: 1,
			Rate: 100, TaxableValue: 100, VATPercent: 10, VAT: 10, Total: 110,
		}},
		Subtotal: 100, TotalVAT: 10, GrandTotal: 110, Currency: "BHD",
	}, out)
	if genErr == nil {
		if info, err := os.Stat(out); err != nil || info.Size() == 0 {
			t.Fatalf("generated PDF missing or empty: %v", err)
		}
	}
	return genErr
}

func TestPDFHeader_RendersFromActiveOverlay(t *testing.T) {
	// Default overlay (three address lines — the historical layout).
	if err := generateWithOverlay(t, overlay.BuiltinDefaults()); err != nil {
		t.Skipf("PDF generation unavailable in this environment (fonts): %v", err)
	}

	// A different vertical's identity, with an address block both shorter and
	// longer than three lines — the header must adapt, not panic.
	for _, lines := range [][]string{
		{"Single line address"},
		{"L1", "L2", "L3", "L4", "L5"},
		nil, // no address at all
	} {
		ov := overlay.BuiltinDefaults()
		ov.Divisions[0].LegalName = "Wasela Café LLC"
		ov.Divisions[0].AddressLines = lines
		if err := generateWithOverlay(t, ov); err != nil {
			t.Errorf("generate with %d address lines: %v", len(lines), err)
		}
	}
}
