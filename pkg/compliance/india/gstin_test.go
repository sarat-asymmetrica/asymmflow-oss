package india

import (
	"strings"
	"testing"
)

func TestGSTINCheckDigitKnownVectors(t *testing.T) {
	// Hand-computed via the official mod-36 algorithm (see GSTINCheckDigit
	// doc comment); each is re-derived here independently of MakeGSTIN so the
	// two implementations cross-check each other.
	cases := []struct {
		base14 string
		want   byte
	}{
		{"29ABCDE1234F1Z", 'W'},
		{"27AABCM0472E1Z", 'T'},
		{"29AABCM0472E1Z", 'P'},
		{"29AAECK3814F1Z", 'M'},
	}
	for _, tc := range cases {
		got, err := GSTINCheckDigit(tc.base14)
		if err != nil {
			t.Fatalf("GSTINCheckDigit(%q): %v", tc.base14, err)
		}
		if got != tc.want {
			t.Errorf("GSTINCheckDigit(%q) = %q, want %q", tc.base14, got, tc.want)
		}
	}
}

func TestMakeGSTINRoundTripsThroughValidGSTIN(t *testing.T) {
	gstin, err := MakeGSTIN("29", "ABCDE1234F", '1')
	if err != nil {
		t.Fatalf("MakeGSTIN: %v", err)
	}
	if gstin != "29ABCDE1234F1ZW" {
		t.Errorf("MakeGSTIN = %q, want 29ABCDE1234F1ZW", gstin)
	}
	if !ValidGSTIN(gstin) {
		t.Errorf("MakeGSTIN output %q must pass ValidGSTIN", gstin)
	}
}

func TestMakeGSTINRejectsBadInputs(t *testing.T) {
	if _, err := MakeGSTIN("99", "ABCDE1234F", '1'); err == nil {
		t.Error("unknown state code should error")
	}
	if _, err := MakeGSTIN("29", "NOTAPAN", '1'); err == nil {
		t.Error("invalid PAN format should error")
	}
	if _, err := MakeGSTIN("29", "ABCDE1234F", '0'); err == nil {
		t.Error("entity code '0' should error (must be 1-9 or A-Z)")
	}
}

func TestValidGSTINAcceptsConstructedFixtures(t *testing.T) {
	for _, gstin := range []string{
		"29ABCDE1234F1ZW",
		"27AABCM0472E1ZT",
		"29AABCM0472E1ZP",
		"29AAECK3814F1ZM",
	} {
		if !ValidGSTIN(gstin) {
			t.Errorf("ValidGSTIN(%q) = false, want true", gstin)
		}
		if !ValidGSTIN(strings.ToLower(gstin)) {
			t.Errorf("ValidGSTIN must normalise case, lowercase %q failed", gstin)
		}
	}
}

func TestValidGSTINRejectsBadChecksum(t *testing.T) {
	// Flip the check digit of a known-good GSTIN.
	bad := "29ABCDE1234F1Z5" // real check digit is W, not 5
	if ValidGSTIN(bad) {
		t.Errorf("ValidGSTIN(%q) = true, want false (wrong check digit)", bad)
	}
}

func TestValidGSTINRejectsUnknownState(t *testing.T) {
	// State "99" does not exist in the GST state registry.
	if ValidGSTIN("99ABCDE1234F1Z5") {
		t.Error("ValidGSTIN with unknown state code should fail")
	}
}

func TestValidGSTINRejectsBadPANEmbed(t *testing.T) {
	// Malformed PAN segment ("12345" is not [A-Z]{5}) fails format outright.
	if ValidGSTIN("2912345ABCDF1Z5") {
		t.Error("ValidGSTIN with malformed PAN segment should fail format check")
	}
}

func TestValidGSTINRejectsWrongFormat(t *testing.T) {
	for _, bad := range []string{
		"", "short", "29ABCDE1234F1Z", "29ABCDE1234F1ZWW",
	} {
		if ValidGSTIN(bad) {
			t.Errorf("ValidGSTIN(%q) = true, want false", bad)
		}
	}
}

func TestPANFromGSTIN(t *testing.T) {
	if got := PANFromGSTIN("29ABCDE1234F1ZW"); got != "ABCDE1234F" {
		t.Errorf("PANFromGSTIN = %q, want ABCDE1234F", got)
	}
	if got := PANFromGSTIN("short"); got != "" {
		t.Errorf("PANFromGSTIN(short) = %q, want empty", got)
	}
}

func TestValidPANFormat(t *testing.T) {
	cases := map[string]bool{
		"ABCDE1234F": true,
		"AABCM0472E": true,
		"abcde1234f": true, // normalises case
		"ABCDE1234":  false,
		"12345ABCDE": false,
		"":           false,
	}
	for pan, want := range cases {
		if got := ValidPANFormat(pan); got != want {
			t.Errorf("ValidPANFormat(%q) = %v, want %v", pan, got, want)
		}
	}
}
