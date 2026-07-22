package overlay

import "testing"

func TestIndiaMountedRequiresBothCompanyAndDivision(t *testing.T) {
	// Neither set: unmounted.
	o := &CompanyOverlay{}
	if o.IndiaMounted() {
		t.Error("no India config and no division profile: should be unmounted")
	}

	// Company config only, no division carries a profile: still unmounted.
	o = &CompanyOverlay{
		India:     &IndiaCompanyConfig{PAN: "AABCM0472E"},
		Divisions: []DivisionProfile{{Key: "X"}},
	}
	if o.IndiaMounted() {
		t.Error("company India config with no division profile should stay unmounted")
	}

	// Both set: mounted.
	o = &CompanyOverlay{
		India: &IndiaCompanyConfig{PAN: "AABCM0472E"},
		Divisions: []DivisionProfile{
			{Key: "X", India: &IndiaDivisionProfile{GSTIN: "27AABCM0472E1ZT", StateCode: "27"}},
		},
	}
	if !o.IndiaMounted() {
		t.Error("company India config + one division profile should be mounted")
	}
}

func TestBuiltinDefaultsHaveNoIndiaPlane(t *testing.T) {
	if BuiltinDefaults().IndiaMounted() {
		t.Error("BuiltinDefaults() must never mount the India plane (GCC inertness)")
	}
	if BuiltinDefaults().India != nil {
		t.Error("BuiltinDefaults() must leave India nil")
	}
	if BuiltinDefaults().FiscalYearStartMonth != 0 {
		t.Error("BuiltinDefaults() must leave FiscalYearStartMonth unset (calendar year)")
	}
}

func TestIndiaConfigResolvesZeroValueDefaults(t *testing.T) {
	o := &CompanyOverlay{India: &IndiaCompanyConfig{PAN: "AABCM0472E"}}
	cfg := o.IndiaConfig()
	if cfg.HSNTierThresholdINR != 50000000 {
		t.Errorf("HSNTierThresholdINR default = %v, want 50000000", cfg.HSNTierThresholdINR)
	}
	if cfg.EInvoiceThresholdAATO != 50000000 {
		t.Errorf("EInvoiceThresholdAATO default = %v, want 50000000", cfg.EInvoiceThresholdAATO)
	}
	if cfg.CompositionCeilingGoodsINR != 15000000 {
		t.Errorf("CompositionCeilingGoodsINR default = %v, want 15000000", cfg.CompositionCeilingGoodsINR)
	}
	if cfg.CompositionCeilingServicesINR != 5000000 {
		t.Errorf("CompositionCeilingServicesINR default = %v, want 5000000", cfg.CompositionCeilingServicesINR)
	}
	if cfg.B2CLThresholdINR != 100000 {
		t.Errorf("B2CLThresholdINR default = %v, want 100000", cfg.B2CLThresholdINR)
	}
	if cfg.GSTR1SchemaVersion != "GST3.2.4" {
		t.Errorf("GSTR1SchemaVersion default = %q, want GST3.2.4", cfg.GSTR1SchemaVersion)
	}
}

func TestIndiaConfigExplicitValuesOverrideDefaults(t *testing.T) {
	o := &CompanyOverlay{India: &IndiaCompanyConfig{
		PAN:                 "AABCM0472E",
		HSNTierThresholdINR: 60000000,
		GSTR1SchemaVersion:  "GST4.0.0",
	}}
	cfg := o.IndiaConfig()
	if cfg.HSNTierThresholdINR != 60000000 {
		t.Errorf("explicit HSNTierThresholdINR overridden, got %v", cfg.HSNTierThresholdINR)
	}
	if cfg.GSTR1SchemaVersion != "GST4.0.0" {
		t.Errorf("explicit GSTR1SchemaVersion overridden, got %q", cfg.GSTR1SchemaVersion)
	}
	// Untouched fields still resolve to their defaults.
	if cfg.B2CLThresholdINR != 100000 {
		t.Errorf("B2CLThresholdINR default = %v, want 100000", cfg.B2CLThresholdINR)
	}
}

func TestIndiaConfigNilIndiaStillResolvesDefaults(t *testing.T) {
	o := &CompanyOverlay{}
	cfg := o.IndiaConfig()
	if cfg.HSNTierThresholdINR != 50000000 {
		t.Errorf("nil India should still resolve defaults, got %v", cfg.HSNTierThresholdINR)
	}
}

func TestRateForHSNLongestPrefixWins(t *testing.T) {
	cfg := IndiaCompanyConfig{TaxCategories: []GSTTaxCategory{
		{HSNPrefix: "84", RatePct: 12, Description: "generic machinery"},
		{HSNPrefix: "8481", RatePct: 18, Description: "valves"},
	}}
	rate, cess, ok := cfg.RateForHSN("8481")
	if !ok || rate != 18 {
		t.Errorf("RateForHSN(8481) = (%v, %v, %v), want (18, _, true)", rate, cess, ok)
	}
	rate, _, ok = cfg.RateForHSN("8413") // matches only the "84" prefix
	if !ok || rate != 12 {
		t.Errorf("RateForHSN(8413) = (%v, _, %v), want (12, true)", rate, ok)
	}
	if _, _, ok = cfg.RateForHSN("9999"); ok {
		t.Error("RateForHSN(9999) should not match any configured prefix")
	}
	if _, _, ok = cfg.RateForHSN(""); ok {
		t.Error("RateForHSN(\"\") should not match")
	}
}

func TestFYStartMonthOrDefault(t *testing.T) {
	// Calendar year: no India plane, no explicit month.
	if got := (&CompanyOverlay{}).FYStartMonthOrDefault(); got != 1 {
		t.Errorf("no India, no month: got %d, want 1", got)
	}
	// Explicit month always wins, India or not.
	o := &CompanyOverlay{FiscalYearStartMonth: 7}
	if got := o.FYStartMonthOrDefault(); got != 7 {
		t.Errorf("explicit month 7: got %d, want 7", got)
	}
	// India mounted, no explicit month: defaults to April (G9).
	o = &CompanyOverlay{
		India: &IndiaCompanyConfig{PAN: "AABCM0472E"},
		Divisions: []DivisionProfile{
			{Key: "X", India: &IndiaDivisionProfile{GSTIN: "27AABCM0472E1ZT", StateCode: "27"}},
		},
	}
	if got := o.FYStartMonthOrDefault(); got != 4 {
		t.Errorf("India mounted, no explicit month: got %d, want 4", got)
	}
	// India mounted but explicit month still wins.
	o.FiscalYearStartMonth = 1
	if got := o.FYStartMonthOrDefault(); got != 1 {
		t.Errorf("India mounted with explicit calendar-year month: got %d, want 1", got)
	}
}
