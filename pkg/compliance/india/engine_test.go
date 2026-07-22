package india

import (
	"errors"
	"testing"

	"ph_holdings_app/pkg/overlay"
)

// Canon fixtures (SYNTHETIC_IDENTITY.md India demo canon): Meridian
// Instruments & Controls Pvt Ltd, PAN AABCM0472E, GSTIN 27AABCM0472E1ZT
// (Mumbai, Maharashtra state 27) and 29AABCM0472E1ZP (Bengaluru, Karnataka
// state 29). Kaveri Trade Links, PAN AAECK3814F, GSTIN 29AAECK3814F1ZM
// (Karnataka), composition. Customers: Sahyadri Process Equipment Pvt Ltd
// (Maharashtra, intra-state B2B), Charminar Engineering Co (Telangana state
// 36, inter-state B2B), Konark Exports (SEZ Unit, Maharashtra). All
// fictional — no real Indian business, PAN, or GSTIN.

// standardConfig is the demo GST rate schedule as DATA (§0 G3) — never the
// legacy hardcoded hsnRates map in gst.go.
func standardConfig(aatoINR float64) EngineConfig {
	return EngineConfig{
		AATOINR: aatoINR,
		Rates: overlay.IndiaCompanyConfig{
			HSNTierThresholdINR: 50000000, // ₹5cr, resolved (mirrors IndiaConfig() default)
			TaxCategories: []overlay.GSTTaxCategory{
				{HSNPrefix: "8481", RatePct: 18, Description: "Valves, taps, cocks"},
				{HSNPrefix: "4901", RatePct: 0, Description: "Printed books"},
				{HSNPrefix: "2202", RatePct: 28, CessPct: 12, Description: "Aerated waters"},
				{HSNPrefix: "1006", RatePct: 5, Description: "Rice"},
			},
		},
	}
}

func TestComputeInvoiceGSTIntraStateSplit(t *testing.T) {
	result, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"}, // Meridian Mumbai, Maharashtra
		Supply{PlaceOfSupplyStateCode: "27", B2B: true},
		[]Line{{HSN: "8481", Description: "Gate valve", Quantity: 10, UQC: "NOS", TaxableValueINR: 1000}},
		standardConfig(10000000),
	)
	if err != nil {
		t.Fatalf("ComputeInvoiceGST: %v", err)
	}
	if result.Classification != IntraState {
		t.Fatalf("Classification = %v, want IntraState", result.Classification)
	}
	if result.ZeroRated || result.Composition {
		t.Fatalf("unexpected markers: %+v", result)
	}
	line := result.Lines[0]
	if line.CGST != 90 || line.SGST != 90 || line.IGST != 0 {
		t.Fatalf("line = %+v, want CGST=90 SGST=90 IGST=0", line)
	}
	if result.Totals.CGST != 90 || result.Totals.SGST != 90 || result.Totals.IGST != 0 {
		t.Fatalf("totals = %+v", result.Totals)
	}
}

func TestComputeInvoiceGSTInterStateIGST(t *testing.T) {
	result, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},                                  // Meridian Mumbai, Maharashtra
		Supply{PlaceOfSupplyStateCode: "36", B2B: true},            // Charminar Engineering Co, Telangana
		[]Line{{HSN: "8481", Quantity: 1, TaxableValueINR: 1000}},
		standardConfig(10000000),
	)
	if err != nil {
		t.Fatalf("ComputeInvoiceGST: %v", err)
	}
	if result.Classification != InterState {
		t.Fatalf("Classification = %v, want InterState", result.Classification)
	}
	if result.ZeroRated {
		t.Fatalf("plain inter-state must not be marked ZeroRated")
	}
	line := result.Lines[0]
	if line.IGST != 180 || line.CGST != 0 || line.SGST != 0 {
		t.Fatalf("line = %+v, want IGST=180", line)
	}
}

func TestComputeInvoiceGSTSEZSameStateStillZeroRatedInterState(t *testing.T) {
	// Konark Exports (SEZ Unit) is in Maharashtra, same state as Meridian
	// Mumbai — §0 G2 says SEZ forces inter-state/zero-rated regardless.
	result, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", SEZ: true, B2B: true},
		[]Line{{HSN: "8481", Quantity: 1, TaxableValueINR: 1000}},
		standardConfig(10000000),
	)
	if err != nil {
		t.Fatalf("ComputeInvoiceGST: %v", err)
	}
	if result.Classification != InterState {
		t.Fatalf("Classification = %v, want InterState (SEZ forces inter-state)", result.Classification)
	}
	if !result.ZeroRated {
		t.Fatalf("expected ZeroRated marker for SEZ supply")
	}
	line := result.Lines[0]
	if line.IGST != 0 || line.CGST != 0 || line.SGST != 0 || line.RatePct != 0 {
		t.Fatalf("line = %+v, want all-zero tax for SEZ zero-rated supply", line)
	}
}

func TestComputeInvoiceGSTCompositionNoTax(t *testing.T) {
	// Kaveri Trade Links, Karnataka, composition taxable person.
	result, err := ComputeInvoiceGST(
		Supplier{StateCode: "29", Composition: true},
		Supply{PlaceOfSupplyStateCode: "29", B2B: false},
		[]Line{{HSN: "8481", Quantity: 5, TaxableValueINR: 5000}},
		standardConfig(10000000),
	)
	if err != nil {
		t.Fatalf("ComputeInvoiceGST: %v", err)
	}
	if !result.Composition {
		t.Fatalf("expected Composition marker")
	}
	if result.ZeroRated {
		t.Fatalf("composition is a distinct marker from zero-rated — must not also set ZeroRated")
	}
	line := result.Lines[0]
	if line.CGST != 0 || line.SGST != 0 || line.IGST != 0 || line.Cess != 0 || line.RatePct != 0 {
		t.Fatalf("line = %+v, want zero tax across every head", line)
	}
	if line.TaxableValueINR != 5000 {
		t.Fatalf("TaxableValueINR = %v, want 5000 (Bill of Supply still states the value)", line.TaxableValueINR)
	}
}

func TestComputeInvoiceGSTCessOwnComponentPerLine(t *testing.T) {
	// Demo canon: HSN 2202 (aerated waters) = 28% GST + 12% cess.
	result, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: true},
		[]Line{{HSN: "2202", Quantity: 1, TaxableValueINR: 1000}},
		standardConfig(10000000),
	)
	if err != nil {
		t.Fatalf("ComputeInvoiceGST: %v", err)
	}
	line := result.Lines[0]
	if line.CGST != 140 || line.SGST != 140 {
		t.Fatalf("line = %+v, want CGST=140 SGST=140 (28%% intra-state split)", line)
	}
	if line.Cess != 120 {
		t.Fatalf("line = %+v, want Cess=120 (12%% of 1000, its own component)", line)
	}
	if result.Totals.Cess != 120 {
		t.Fatalf("Totals.Cess = %v, want 120", result.Totals.Cess)
	}
}

func TestComputeInvoiceGSTRoundingOddPaiseHeadsSumExactly(t *testing.T) {
	result, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: true},
		[]Line{{HSN: "8481", Quantity: 1, TaxableValueINR: 41.83}}, // 41.83 * 18% = 7.5294 -> 7.53 (odd paise)
		standardConfig(10000000),
	)
	if err != nil {
		t.Fatalf("ComputeInvoiceGST: %v", err)
	}
	line := result.Lines[0]
	// round2dp guards against float64 representation drift on the raw sum
	// (3.77+3.76 alone can render as 7.529999999999999) — the same rounding
	// every accumulation into Totals already goes through.
	if round2dp(line.CGST+line.SGST) != 7.53 {
		t.Fatalf("CGST+SGST = %v, want 7.53 exactly (line = %+v)", line.CGST+line.SGST, line)
	}
	if line.CGST == line.SGST {
		t.Fatalf("expected an uneven split to exercise the odd-paise path, got CGST=SGST=%v", line.CGST)
	}
}

func TestComputeInvoiceGSTHSNRefusalUnderTierBoundary(t *testing.T) {
	cfg := standardConfig(10000000) // ₹1cr AATO: TierUpTo5Cr

	// 3-digit HSN on a B2B invoice under the ₹5cr tier: refused (needs 4).
	_, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: true},
		[]Line{{HSN: "848", TaxableValueINR: 1000}},
		cfg,
	)
	var hsnErr *HSNValidationError
	if !errors.As(err, &hsnErr) {
		t.Fatalf("err = %v, want *HSNValidationError for 3-digit HSN under 5cr B2B", err)
	}

	// 4-digit HSN on the same invoice passes the digit check (and 8481 is
	// configured, so it resolves cleanly).
	result, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: true},
		[]Line{{HSN: "8481", TaxableValueINR: 1000}},
		cfg,
	)
	if err != nil {
		t.Fatalf("ComputeInvoiceGST with 4-digit HSN under 5cr B2B: %v", err)
	}
	if result == nil {
		t.Fatalf("expected a result")
	}
}

func TestComputeInvoiceGSTHSNRefusalAboveTierNeedsSixOnB2CToo(t *testing.T) {
	cfg := standardConfig(60000000) // ₹6cr AATO: TierAbove5Cr

	// 4-digit HSN, B2C, above the 5cr tier: refused (needs 6 on ALL invoices).
	_, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: false},
		[]Line{{HSN: "8481", TaxableValueINR: 1000}},
		cfg,
	)
	var hsnErr *HSNValidationError
	if !errors.As(err, &hsnErr) {
		t.Fatalf("err = %v, want *HSNValidationError for 4-digit HSN above 5cr B2C", err)
	}
}

func TestComputeInvoiceGSTHSNRefusalExportNeedsEight(t *testing.T) {
	cfg := standardConfig(10000000) // TierUpTo5Cr would only need 4 for B2B

	_, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: true, Export: true},
		[]Line{{HSN: "84811090", TaxableValueINR: 1000}}, // 8-digit — must pass
		cfg,
	)
	if err != nil {
		t.Fatalf("8-digit export HSN should pass: %v", err)
	}

	_, err = ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: true, Export: true},
		[]Line{{HSN: "848110", TaxableValueINR: 1000}}, // 6-digit — export always needs 8
		cfg,
	)
	var hsnErr *HSNValidationError
	if !errors.As(err, &hsnErr) {
		t.Fatalf("err = %v, want *HSNValidationError for 6-digit export HSN (needs 8)", err)
	}
}

func TestComputeInvoiceGSTUnknownHSNRefuses(t *testing.T) {
	_, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: true},
		[]Line{{HSN: "9999", TaxableValueINR: 1000}}, // not in standardConfig's TaxCategories
		standardConfig(10000000),
	)
	var hsnErr *HSNValidationError
	if !errors.As(err, &hsnErr) {
		t.Fatalf("err = %v, want *HSNValidationError for an unconfigured HSN (no silent fallback rate)", err)
	}
}

func TestComputeInvoiceGSTZeroRateConfiguredHSNIsNotSEZZeroRated(t *testing.T) {
	// HSN 4901 (printed books) is configured at 0% — a genuine nil rate,
	// distinct from the SEZ ZeroRated marker.
	result, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: true},
		[]Line{{HSN: "4901", TaxableValueINR: 1000}},
		standardConfig(10000000),
	)
	if err != nil {
		t.Fatalf("ComputeInvoiceGST: %v", err)
	}
	if result.ZeroRated {
		t.Fatalf("a configured 0%% rate must not set the SEZ ZeroRated marker")
	}
	line := result.Lines[0]
	if line.CGST != 0 || line.SGST != 0 || line.RatePct != 0 {
		t.Fatalf("line = %+v, want computed zero tax at a real (non-SEZ) 0%% rate", line)
	}
}

func TestRoundInvoiceHeadsToRupeeNotAppliedByDefault(t *testing.T) {
	result, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: true},
		[]Line{{HSN: "8481", TaxableValueINR: 41.83}}, // produces CGST=3.77, SGST=3.76 (paise precision)
		standardConfig(10000000),
	)
	if err != nil {
		t.Fatalf("ComputeInvoiceGST: %v", err)
	}
	if result.Totals.CGST == float64(int64(result.Totals.CGST)) {
		t.Fatalf("fixture should carry paise precision before rounding, got CGST=%v", result.Totals.CGST)
	}

	rounded := RoundInvoiceHeadsToRupee(result.Totals)
	if rounded.CGST != 4 || rounded.SGST != 4 {
		t.Fatalf("rounded = %+v, want CGST=4 SGST=4 (nearest rupee)", rounded)
	}
	if rounded.TaxableValueINR != result.Totals.TaxableValueINR {
		t.Fatalf("RoundInvoiceHeadsToRupee must not touch taxable value: got %v, want %v",
			rounded.TaxableValueINR, result.Totals.TaxableValueINR)
	}
}

func TestComputeInvoiceGSTReverseChargeCarriedThroughUnchanged(t *testing.T) {
	withRC, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: true, ReverseCharge: true},
		[]Line{{HSN: "8481", TaxableValueINR: 1000}},
		standardConfig(10000000),
	)
	if err != nil {
		t.Fatalf("ComputeInvoiceGST: %v", err)
	}
	withoutRC, err := ComputeInvoiceGST(
		Supplier{StateCode: "27"},
		Supply{PlaceOfSupplyStateCode: "27", B2B: true, ReverseCharge: false},
		[]Line{{HSN: "8481", TaxableValueINR: 1000}},
		standardConfig(10000000),
	)
	if err != nil {
		t.Fatalf("ComputeInvoiceGST: %v", err)
	}
	if !withRC.ReverseCharge {
		t.Fatalf("expected ReverseCharge marker to be carried through")
	}
	if withoutRC.ReverseCharge {
		t.Fatalf("expected ReverseCharge marker false when flag unset")
	}
	if withRC.Totals != withoutRC.Totals {
		t.Fatalf("reverse charge must not change computation: with=%+v without=%+v", withRC.Totals, withoutRC.Totals)
	}
}
