package overlay

import (
	"encoding/json"
	"testing"
)

func TestSupplierAliasVocabulary_NilFallsBackToBuiltin(t *testing.T) {
	o := &CompanyOverlay{} // field absent → built-in vocabulary
	vocab := o.SupplierAliasVocabulary()
	if got := vocab.CanonicalCodes["SVX"]; got != "SRVX" {
		t.Fatalf("nil SupplierAliases must fall back to the built-in default, got %q", got)
	}
}

func TestSupplierAliasVocabulary_ExplicitEmptyClears(t *testing.T) {
	var o CompanyOverlay
	if err := json.Unmarshal([]byte(`{"divisions":[{"key":"X"}],"supplier_aliases":{}}`), &o); err != nil {
		t.Fatal(err)
	}
	vocab := o.SupplierAliasVocabulary()
	if len(vocab.CanonicalCodes) != 0 || len(vocab.BrandAliases) != 0 {
		t.Fatalf("explicit empty object must clear the vocabulary, got %+v", vocab)
	}
}

func TestSupplierAliasVocabulary_NormalisesKeysUppercase(t *testing.T) {
	o := &CompanyOverlay{SupplierAliases: &SupplierAliasConfig{
		CanonicalCodes: map[string]string{" svx ": " srvx "},
		BrandAliases:   map[string][]string{"promag": {"Rhine Instruments"}},
	}}
	vocab := o.SupplierAliasVocabulary()
	if vocab.CanonicalCodes["SVX"] != "SRVX" {
		t.Fatalf("code keys/values must normalise to uppercase trimmed, got %+v", vocab.CanonicalCodes)
	}
	if terms := vocab.BrandAliases["PROMAG"]; len(terms) != 1 || terms[0] != "Rhine Instruments" {
		t.Fatalf("brand tokens must uppercase, terms pass through: %+v", vocab.BrandAliases)
	}
}

func TestLicenseKeyPrefixOrDefault(t *testing.T) {
	cases := []struct {
		configured string
		want       string
	}{
		{"", "PH"},       // absent/blank keeps existing activations valid
		{"  ", "PH"},     // whitespace is blank
		{"ph", "PH"},     // normalised uppercase
		{"ACME", "ACME"}, // sovereign fork prefix
		{" wsl ", "WSL"}, // trimmed
	}
	for _, tc := range cases {
		o := &CompanyOverlay{LicenseKeyPrefix: tc.configured}
		if got := o.LicenseKeyPrefixOrDefault(); got != tc.want {
			t.Fatalf("LicenseKeyPrefixOrDefault(%q) = %q, want %q", tc.configured, got, tc.want)
		}
	}
	if got := BuiltinDefaults().LicenseKeyPrefixOrDefault(); got != "PH" {
		t.Fatalf("built-in default prefix must stay PH (13-char keys), got %q", got)
	}
}
