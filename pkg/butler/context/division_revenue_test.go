package context

import (
	"testing"

	"ph_holdings_app/pkg/overlay"
)

// TestGetFinancialContext_DivisionRevenue_NDivision proves the Wave 12.5 B4
// fix: division_revenue must contain one entry PER configured division
// (keyed by the division's registry Key), not just the old frozen
// primary/secondary ("ph_trading"/"ahs_trading") two-slot breakdown. A third
// configured division's revenue must no longer be silently dropped.
func TestGetFinancialContext_DivisionRevenue_NDivision(t *testing.T) {
	svc := testService(t)

	// Build a THREE-division overlay on top of the builtin two-division
	// defaults and make it the process-wide active overlay for this test.
	base := overlay.BuiltinDefaults()
	threeDiv := *base
	threeDiv.Divisions = append(append([]overlay.DivisionProfile{}, base.Divisions...), overlay.DivisionProfile{
		Key:       "Gamma Devices",
		LegalName: "GAMMA DEVICES W.L.L.",
	})

	previous := overlay.Active()
	overlay.SetActive(&threeDiv)
	t.Cleanup(func() { overlay.SetActive(previous) })

	seed := []struct {
		invoiceNumber string
		division      string
		total         float64
	}{
		{"INV-A1", "Acme Instrumentation", 100.0},
		{"INV-B1", "Beacon Controls", 250.5},
		{"INV-G1", "Gamma Devices", 75.25},
	}
	for _, s := range seed {
		inv := Invoice{InvoiceNumber: s.invoiceNumber, Division: s.division, GrandTotalBHD: s.total}
		if err := svc.db.Create(&inv).Error; err != nil {
			t.Fatalf("seed invoice %s: %v", s.invoiceNumber, err)
		}
	}

	result := svc.getFinancialContext()
	divRevenue, ok := result["division_revenue"].(map[string]any)
	if !ok {
		t.Fatalf("division_revenue must be a map[string]any, got %T", result["division_revenue"])
	}

	if len(divRevenue) != 3 {
		t.Fatalf("expected 3 division_revenue entries, got %d: %+v", len(divRevenue), divRevenue)
	}
	for _, s := range seed {
		got, ok := divRevenue[s.division]
		if !ok {
			t.Fatalf("division_revenue missing key %q: %+v", s.division, divRevenue)
		}
		if got.(float64) != s.total {
			t.Fatalf("division_revenue[%q] = %v, want %v", s.division, got, s.total)
		}
	}

	// The old frozen two-slot legacy keys must be gone.
	if _, leaked := divRevenue["ph_trading"]; leaked {
		t.Fatal("legacy key ph_trading must not appear in division_revenue")
	}
	if _, leaked := divRevenue["ahs_trading"]; leaked {
		t.Fatal("legacy key ahs_trading must not appear in division_revenue")
	}
}
