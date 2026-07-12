package overlay

import (
	"encoding/json"
	"os"
	"testing"
)

// TestNormalizeDivisionName_BeaconSpellings verifies that every known spelling
// of "Beacon Controls" normalises to the canonical key.
func TestNormalizeDivisionName_BeaconSpellings(t *testing.T) {
	o := BuiltinDefaults()

	beaconInputs := []string{
		"Beacon Controls",
		"beacon controls",
		"BEACON CONTROLS",
		"  beacon controls  ",
		"beacon controls wll",
		"beacon controls w.l.l",
		"beacon controls w.l.l.",
		"BEACON CONTROLS WLL",
		"Beacon Controls WLL",
		"Beacon Controls W.L.L",
		"Beacon Controls W.L.L.",
	}

	for _, input := range beaconInputs {
		got := o.NormalizeDivisionName(input)
		if got != "Beacon Controls" {
			t.Errorf("NormalizeDivisionName(%q) = %q, want %q", input, got, "Beacon Controls")
		}
	}
}

// TestNormalizeDivisionName_UnknownFallsToDefault verifies that unknown or
// empty strings fall back to the default division.
func TestNormalizeDivisionName_UnknownFallsToDefault(t *testing.T) {
	o := BuiltinDefaults()

	unknownInputs := []string{
		"",
		"   ",
		"Unknown Division",
		"acme",
		"beacon",
		"random text",
		"Acme Instrumentation Ltd",
	}

	for _, input := range unknownInputs {
		got := o.NormalizeDivisionName(input)
		if got != "Acme Instrumentation" {
			t.Errorf("NormalizeDivisionName(%q) = %q, want %q", input, got, "Acme Instrumentation")
		}
	}
}

// TestNormalizeDivisionName_AcmeVariants verifies that the exact key match works
// for Acme Instrumentation.
func TestNormalizeDivisionName_AcmeVariants(t *testing.T) {
	o := BuiltinDefaults()

	acmeInputs := []string{
		"Acme Instrumentation",
		"acme instrumentation",
		"ACME INSTRUMENTATION",
		"  Acme Instrumentation  ",
	}

	for _, input := range acmeInputs {
		got := o.NormalizeDivisionName(input)
		if got != "Acme Instrumentation" {
			t.Errorf("NormalizeDivisionName(%q) = %q, want %q", input, got, "Acme Instrumentation")
		}
	}
}

// TestProfile_KnownDivision checks that Profile returns the correct values for
// known division keys.
func TestProfile_KnownDivision(t *testing.T) {
	o := BuiltinDefaults()

	beacon := o.Profile("Beacon Controls")
	if beacon.LegalName != "BEACON CONTROLS W.L.L." {
		t.Errorf("beacon LegalName = %q, want %q", beacon.LegalName, "BEACON CONTROLS W.L.L.")
	}
	if beacon.VATNumber != "990000000000001" {
		t.Errorf("beacon VATNumber = %q, want %q", beacon.VATNumber, "990000000000001")
	}
	if len(beacon.AddressLines) != 2 {
		t.Errorf("beacon AddressLines len = %d, want 2", len(beacon.AddressLines))
	}
	if len(beacon.BankDetails) != 1 {
		t.Errorf("beacon BankDetails len = %d, want 1", len(beacon.BankDetails))
	}

	acme := o.Profile("Acme Instrumentation")
	if acme.LegalName != "ACME INSTRUMENTATION W.L.L" {
		t.Errorf("acme LegalName = %q, want %q", acme.LegalName, "ACME INSTRUMENTATION W.L.L")
	}
	if acme.VATNumber != "990000000000000" {
		t.Errorf("acme VATNumber = %q, want %q", acme.VATNumber, "990000000000000")
	}
	if len(acme.AddressLines) != 3 {
		t.Errorf("acme AddressLines len = %d, want 3", len(acme.AddressLines))
	}
}

// TestProfile_Fallback verifies that an unknown key returns the default division.
func TestProfile_Fallback(t *testing.T) {
	o := BuiltinDefaults()

	fallback := o.Profile("nonexistent division")
	if fallback.Key != "Acme Instrumentation" {
		t.Errorf("Profile(unknown).Key = %q, want %q", fallback.Key, "Acme Instrumentation")
	}
}

// TestProfile_EmptyKey verifies that an empty key returns the default division.
func TestProfile_EmptyKey(t *testing.T) {
	o := BuiltinDefaults()

	fallback := o.Profile("")
	if fallback.Key != "Acme Instrumentation" {
		t.Errorf("Profile(\"\").Key = %q, want %q", fallback.Key, "Acme Instrumentation")
	}
}

// TestBuiltinDefaults_JSONRoundTrip verifies that BuiltinDefaults can be
// marshalled to JSON and unmarshalled back without loss of data.
func TestBuiltinDefaults_JSONRoundTrip(t *testing.T) {
	original := BuiltinDefaults()

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(BuiltinDefaults()) error: %v", err)
	}

	var restored CompanyOverlay
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if restored.DefaultDivisionKey != original.DefaultDivisionKey {
		t.Errorf("DefaultDivisionKey mismatch: got %q, want %q", restored.DefaultDivisionKey, original.DefaultDivisionKey)
	}
	if restored.Currency != original.Currency {
		t.Errorf("Currency mismatch: got %q, want %q", restored.Currency, original.Currency)
	}
	if restored.CurrencyDecimals != original.CurrencyDecimals {
		t.Errorf("CurrencyDecimals mismatch: got %d, want %d", restored.CurrencyDecimals, original.CurrencyDecimals)
	}
	if restored.DefaultVATRate != original.DefaultVATRate {
		t.Errorf("DefaultVATRate mismatch: got %f, want %f", restored.DefaultVATRate, original.DefaultVATRate)
	}
	if len(restored.Divisions) != len(original.Divisions) {
		t.Fatalf("Divisions len mismatch: got %d, want %d", len(restored.Divisions), len(original.Divisions))
	}

	// Verify the restored overlay normalises identically.
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"Beacon Controls", "Beacon Controls"},
		{"beacon controls wll", "Beacon Controls"},
		{"beacon controls w.l.l.", "Beacon Controls"},
		{"anything else", "Acme Instrumentation"},
	} {
		got := restored.NormalizeDivisionName(tc.input)
		if got != tc.want {
			t.Errorf("after round-trip NormalizeDivisionName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestDefaultDivision confirms the DefaultDivision() helper works.
func TestDefaultDivision(t *testing.T) {
	o := BuiltinDefaults()
	if o.DefaultDivision() != "Acme Instrumentation" {
		t.Errorf("DefaultDivision() = %q, want %q", o.DefaultDivision(), "Acme Instrumentation")
	}
}

// TestLoadOverlay_EmptyDirs verifies that LoadOverlay with no valid dirs
// returns BuiltinDefaults (never nil).
func TestLoadOverlay_EmptyDirs(t *testing.T) {
	got := LoadOverlay([]string{})
	if got == nil {
		t.Fatal("LoadOverlay(empty) returned nil")
	}
	if got.DefaultDivisionKey != "Acme Instrumentation" {
		t.Errorf("DefaultDivisionKey = %q, want %q", got.DefaultDivisionKey, "Acme Instrumentation")
	}
}

// TestLoadOverlay_NonExistentDir verifies that a missing dir doesn't panic.
func TestLoadOverlay_NonExistentDir(t *testing.T) {
	got := LoadOverlay([]string{"/nonexistent/path/that/does/not/exist"})
	if got == nil {
		t.Fatal("LoadOverlay(nonexistent dir) returned nil")
	}
}

// TestLoadOverlay_ValidFile verifies that a valid overlay.json is loaded.
func TestLoadOverlay_ValidFile(t *testing.T) {
	dir := t.TempDir()

	overlay := CompanyOverlay{
		SchemaVersion:      1,
		DefaultDivisionKey: "MyDivision",
		CompanyDisplayName: "My Test Company",
		Currency:           "USD",
		CurrencyDecimals:   2,
		DefaultVATRate:     5.0,
		Divisions: []DivisionProfile{
			{
				Key:       "MyDivision",
				LegalName: "My Test Division LLC",
				Aliases:   []string{"my division"},
			},
		},
	}
	data, _ := json.Marshal(overlay)
	if err := writeFile(dir+"/overlay.json", data); err != nil {
		t.Fatalf("could not write test overlay.json: %v", err)
	}

	got := LoadOverlay([]string{dir})
	if got == nil {
		t.Fatal("LoadOverlay returned nil")
	}
	if got.DefaultDivisionKey != "MyDivision" {
		t.Errorf("DefaultDivisionKey = %q, want MyDivision", got.DefaultDivisionKey)
	}
	if got.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", got.Currency)
	}
}

// writeFile is a minimal helper to write bytes to a file path.
func writeFile(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	f.Close()
	return err
}
