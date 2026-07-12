package overlay

import "testing"

// TestBuiltinBusinessRulesByteIdentical pins every business-policy number to the
// exact value it had as a hardcoded constant before Phase 2c. If any of these
// fail, a financial-semantics value has drifted — STOP and investigate.
func TestBuiltinBusinessRulesByteIdentical(t *testing.T) {
	o := BuiltinDefaults()
	br := o.BusinessRules

	checks := []struct {
		name string
		got  float64
		want float64
	}{
		{"MinMarginPct", br.MinMarginPct, 0.08},
		{"ABBCompetitionMinMargin", br.ABBCompetitionMinMargin, 0.15},
		{"EmergencyMinMarginPct", br.EmergencyMinMarginPct, 0.20},
		{"ApprovalThresholdMargin", br.ApprovalThresholdMargin, 0.20},
		{"LargeOrderThresholdBHD", br.LargeOrderThresholdBHD, 10000},
		{"MonthlyOperatingCostBHD", br.MonthlyOperatingCostBHD, 15000},
		{"DefaultProductMargin", o.DefaultProductMargin, 0.12},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}

	// Product markup rules — byte-identical to the old ProductMarkupRules map.
	wantMargins := map[string]float64{
		"Rhine Flow":                    0.15,
		"Rhine Level":                   0.18,
		"Rhine Instruments Pressure":    0.18,
		"Rhine Instruments Temperature": 0.15,
		"Rhine Analytics":               0.20,
		"Rhine Instruments General":     0.12,
		"Oxan Analytics":                0.25,
		"GIC":                           0.10,
		"Unknown":                       0.12, // falls back to DefaultProductMargin
	}
	for pt, want := range wantMargins {
		if got := o.ProductMargin(pt); got != want {
			t.Errorf("ProductMargin(%q) = %v, want %v", pt, got, want)
		}
	}

	// Grade policies — discount, advance, terms, day-ceiling per grade.
	type gp struct {
		discount float64
		advance  float64
		terms    string
		maxDays  int
	}
	wantGrades := map[string]gp{
		"A": {0.07, 0.0, "Net 45 days", 55},
		"B": {0.03, 0.0, "Net 90 days", 100},
		"C": {0.00, 0.50, "Net 120 days with 50% advance", 130},
		"D": {0.00, 1.00, "100% advance or DECLINE", 0},
	}
	for grade, want := range wantGrades {
		p, ok := o.GradePolicyFor(grade)
		if !ok {
			t.Errorf("GradePolicyFor(%q) missing", grade)
			continue
		}
		if p.MaxDiscount != want.discount {
			t.Errorf("grade %s MaxDiscount = %v, want %v", grade, p.MaxDiscount, want.discount)
		}
		if p.AdvancePct != want.advance {
			t.Errorf("grade %s AdvancePct = %v, want %v", grade, p.AdvancePct, want.advance)
		}
		if p.Terms != want.terms {
			t.Errorf("grade %s Terms = %q, want %q", grade, p.Terms, want.terms)
		}
		if p.MaxDays != want.maxDays {
			t.Errorf("grade %s MaxDays = %v, want %v", grade, p.MaxDays, want.maxDays)
		}
		// Accessor parity.
		if d := o.CustomerDiscount(grade); d != want.discount {
			t.Errorf("CustomerDiscount(%q) = %v, want %v", grade, d, want.discount)
		}
		terms, adv := o.PaymentTerms(grade)
		if terms != want.terms || adv != want.advance {
			t.Errorf("PaymentTerms(%q) = (%q,%v), want (%q,%v)", grade, terms, adv, want.terms, want.advance)
		}
	}

	// Unknown grade: discount 0, terms fall back to grade B (historical default case).
	if d := o.CustomerDiscount("Z"); d != 0.0 {
		t.Errorf("CustomerDiscount(unknown) = %v, want 0", d)
	}
	if terms, adv := o.PaymentTerms("Z"); terms != "Net 90 days" || adv != 0.0 {
		t.Errorf("PaymentTerms(unknown) = (%q,%v), want (Net 90 days,0)", terms, adv)
	}

	if got := o.CompetitorName(); got != "ABB" {
		t.Errorf("CompetitorName() = %q, want ABB", got)
	}
}

// TestABBThresholdPercentExact guards the geometry_bridge.go repoint: the tender
// pipeline compares an achieved margin expressed in PERCENT against the ABB
// floor, so it uses ABBCompetitionMinMargin*100. That multiply must equal 15.0
// exactly or a financial boundary could flip.
func TestABBThresholdPercentExact(t *testing.T) {
	if got := BuiltinDefaults().BusinessRules.ABBCompetitionMinMargin * 100.0; got != 15.0 {
		t.Fatalf("ABBCompetitionMinMargin*100 = %v, want exactly 15.0", got)
	}
}

// TestActiveSingleton verifies the process-wide accessor defaults to builtin
// values and honours SetActive (and that nil is a no-op).
func TestActiveSingleton(t *testing.T) {
	if Active() == nil {
		t.Fatal("Active() returned nil")
	}
	if Active().BusinessRules.MinMarginPct != 0.08 {
		t.Errorf("Active() default MinMarginPct = %v, want 0.08", Active().BusinessRules.MinMarginPct)
	}
	orig := Active()
	SetActive(nil) // no-op
	if Active() != orig {
		t.Error("SetActive(nil) must be a no-op")
	}
}
