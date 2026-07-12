package overlay

import "testing"

// TestExchangeRateToBase_BuiltinDefaults pins the canonical FX rates and the
// base/empty/unknown/case-insensitive behaviour. These rates are the single
// source of truth consumed by both the import-time Rhine parser and the live
// costing/posting paths, so this is where their values are asserted. EUR = 0.45
// is the canonical rate that unified the old 0.41-vs-0.45 split.
func TestExchangeRateToBase_BuiltinDefaults(t *testing.T) {
	o := BuiltinDefaults()
	cases := map[string]float64{
		"EUR":   0.45,
		"USD":   0.376,
		"GBP":   0.52,
		"CHF":   0.425,
		"SAR":   0.100,
		"AED":   0.102,
		"eur":   0.45, // case-insensitive
		" EUR ": 0.45, // whitespace-trimmed
		"BHD":   1.0,  // base currency
		"bhd":   1.0,  // base currency, case-insensitive
		"":      1.0,  // empty → base
		"JPY":   1.0,  // unknown → 1.0 (historic default case)
	}
	for cur, want := range cases {
		if got := o.ExchangeRateToBase(cur); got != want {
			t.Errorf("ExchangeRateToBase(%q) = %v, want %v", cur, got, want)
		}
	}
}
