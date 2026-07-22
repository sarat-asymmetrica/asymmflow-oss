package india

import (
	"strings"
	"testing"

	"ph_holdings_app/pkg/overlay"
)

func TestValidateOverlayIndiaDemoFixturesAreValid(t *testing.T) {
	for _, dir := range []string{
		"../../../overlays/india-demo",
		"../../../overlays/india-demo/composition",
	} {
		ov := overlay.LoadOverlay([]string{dir})
		if !ov.IndiaMounted() {
			t.Fatalf("%s: expected India plane mounted, LoadOverlay may have fallen back to BuiltinDefaults", dir)
		}
		if problems := ValidateOverlayIndia(ov); len(problems) != 0 {
			t.Errorf("%s: expected no problems, got %v", dir, problems)
		}
	}
}

func TestValidateOverlayIndiaUnmountedReturnsNil(t *testing.T) {
	if problems := ValidateOverlayIndia(nil); problems != nil {
		t.Errorf("nil overlay: got %v, want nil", problems)
	}
	if problems := ValidateOverlayIndia(overlay.BuiltinDefaults()); problems != nil {
		t.Errorf("BuiltinDefaults (India unmounted): got %v, want nil", problems)
	}
}

func TestValidateOverlayIndiaCatchesBadCheckDigit(t *testing.T) {
	ov := &overlay.CompanyOverlay{
		India: &overlay.IndiaCompanyConfig{PAN: "AABCM0472E"},
		Divisions: []overlay.DivisionProfile{
			{Key: "Bad Div", India: &overlay.IndiaDivisionProfile{
				GSTIN:     "27AABCM0472E1Z5", // real check digit is T, not 5
				StateCode: "27",
			}},
		},
	}
	problems := ValidateOverlayIndia(ov)
	if !anyContains(problems, "check-digit") {
		t.Errorf("expected a check-digit problem, got %v", problems)
	}
}

func TestValidateOverlayIndiaCatchesStateMismatch(t *testing.T) {
	ov := &overlay.CompanyOverlay{
		India: &overlay.IndiaCompanyConfig{PAN: "AABCM0472E"},
		Divisions: []overlay.DivisionProfile{
			{Key: "Mismatched Div", India: &overlay.IndiaDivisionProfile{
				GSTIN:     "27AABCM0472E1ZT", // valid GSTIN, state prefix 27
				StateCode: "29",              // declared state_code disagrees
			}},
		},
	}
	problems := ValidateOverlayIndia(ov)
	if !anyContains(problems, "does not match state_code") {
		t.Errorf("expected a state-mismatch problem, got %v", problems)
	}
}

func TestValidateOverlayIndiaCatchesUnknownStateCode(t *testing.T) {
	ov := &overlay.CompanyOverlay{
		India: &overlay.IndiaCompanyConfig{PAN: "AABCM0472E"},
		Divisions: []overlay.DivisionProfile{
			{Key: "Unknown State Div", India: &overlay.IndiaDivisionProfile{
				GSTIN:     "27AABCM0472E1ZT",
				StateCode: "99", // not a real GST state code
			}},
		},
	}
	problems := ValidateOverlayIndia(ov)
	if !anyContains(problems, "not a known GST state code") {
		t.Errorf("expected an unknown-state-code problem, got %v", problems)
	}
}

func TestValidateOverlayIndiaCatchesPANMismatch(t *testing.T) {
	ov := &overlay.CompanyOverlay{
		India: &overlay.IndiaCompanyConfig{PAN: "AABCM0472E"},
		Divisions: []overlay.DivisionProfile{
			{Key: "Wrong PAN Div", India: &overlay.IndiaDivisionProfile{
				GSTIN:     "29AAECK3814F1ZM", // valid GSTIN, but embeds a DIFFERENT PAN
				StateCode: "29",
			}},
		},
	}
	problems := ValidateOverlayIndia(ov)
	if !anyContains(problems, "does not embed company PAN") {
		t.Errorf("expected a PAN-embed problem, got %v", problems)
	}
}

func TestValidateOverlayIndiaCatchesInvalidCompanyPAN(t *testing.T) {
	ov := &overlay.CompanyOverlay{
		India: &overlay.IndiaCompanyConfig{PAN: "NOTAPAN"},
		Divisions: []overlay.DivisionProfile{
			{Key: "X", India: &overlay.IndiaDivisionProfile{GSTIN: "27AABCM0472E1ZT", StateCode: "27"}},
		},
	}
	problems := ValidateOverlayIndia(ov)
	if !anyContains(problems, "not a valid PAN") {
		t.Errorf("expected an invalid-company-PAN problem, got %v", problems)
	}
}

func TestGCCOverlaysAreIndiaInert(t *testing.T) {
	if overlay.BuiltinDefaults().IndiaMounted() {
		t.Fatal("BuiltinDefaults() must never mount the India plane")
	}
	for _, dir := range []string{"../../../data", "../../../overlays/hospitality"} {
		ov := overlay.LoadOverlay([]string{dir})
		if ov.India != nil {
			t.Errorf("%s: expected India == nil, got %+v", dir, ov.India)
		}
		if ov.IndiaMounted() {
			t.Errorf("%s: expected India plane unmounted", dir)
		}
	}
}

func anyContains(problems []string, substr string) bool {
	for _, p := range problems {
		if strings.Contains(p, substr) {
			return true
		}
	}
	return false
}
