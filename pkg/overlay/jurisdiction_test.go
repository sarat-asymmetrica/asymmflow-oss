package overlay

import "testing"

func TestJurisdictionCodeExplicitWins(t *testing.T) {
	o := &CompanyOverlay{Jurisdiction: " sa ", Country: "Bahrain"}
	if got := o.JurisdictionCode(); got != "SA" {
		t.Errorf("explicit jurisdiction: got %q, want SA", got)
	}
}

func TestJurisdictionCodeFromCountryName(t *testing.T) {
	cases := map[string]string{
		"Bahrain":                 "BH",
		"Kingdom of Bahrain":      "BH",
		"Saudi Arabia":            "SA",
		"Kingdom of Saudi Arabia": "SA",
		"KSA":                     "SA",
		"India":                   "IN",
		"United Arab Emirates":    "AE",
		"Oman":                    "OM",
		"Qatar":                   "QA",
		"Kuwait":                  "KW",
		"Atlantis":                "",
		"":                        "",
	}
	for country, want := range cases {
		o := &CompanyOverlay{Country: country}
		if got := o.JurisdictionCode(); got != want {
			t.Errorf("JurisdictionCode(country=%q) = %q, want %q", country, got, want)
		}
	}
}

func TestBuiltinDefaultsJurisdictionIsBahrain(t *testing.T) {
	// The reference deployment must keep routing to the Bahrain engine.
	if got := BuiltinDefaults().JurisdictionCode(); got != "BH" {
		t.Errorf("BuiltinDefaults jurisdiction = %q, want BH", got)
	}
}
