package india

import (
	"fmt"
	"strings"

	"ph_holdings_app/pkg/overlay"
)

// ValidateOverlayIndia checks an overlay's India plane for internal
// consistency and returns human-readable problems. An empty slice means the
// plane is either unmounted (nothing to check) or fully valid. This package
// (pkg/compliance/india) is the one direction allowed to import pkg/overlay
// — overlay itself stays dependency-free (mirrors the SupplierAliasConfig
// precedent) so this validator lives on the compliance side of the seam.
func ValidateOverlayIndia(ov *overlay.CompanyOverlay) []string {
	if ov == nil || !ov.IndiaMounted() {
		return nil
	}

	var problems []string

	pan := strings.ToUpper(strings.TrimSpace(ov.India.PAN))
	if !ValidPANFormat(pan) {
		problems = append(problems, fmt.Sprintf("company PAN %q is not a valid PAN (expected AAAAA9999A)", ov.India.PAN))
	}

	fyMonth := ov.FYStartMonthOrDefault()
	if fyMonth < 1 || fyMonth > 12 {
		problems = append(problems, fmt.Sprintf("fiscal_year_start_month %d is out of range (must be 1-12)", fyMonth))
	}

	for _, div := range ov.Divisions {
		if div.India == nil {
			continue
		}
		gstin := strings.ToUpper(strings.TrimSpace(div.India.GSTIN))
		stateCode := strings.TrimSpace(div.India.StateCode)

		if !ValidStateCode(stateCode) {
			problems = append(problems, fmt.Sprintf("division %q: state_code %q is not a known GST state code", div.Key, div.India.StateCode))
		}

		if !ValidGSTINFormat(gstin) {
			problems = append(problems, fmt.Sprintf("division %q: GSTIN %q has an invalid format", div.Key, div.India.GSTIN))
		} else {
			if !ValidStateCode(gstin[0:2]) {
				problems = append(problems, fmt.Sprintf("division %q: GSTIN %q starts with an unknown state code", div.Key, div.India.GSTIN))
			}
			if want, err := GSTINCheckDigit(gstin[0:14]); err != nil || gstin[14] != want {
				problems = append(problems, fmt.Sprintf("division %q: GSTIN %q fails the check-digit algorithm", div.Key, div.India.GSTIN))
			}
			if stateCode != "" && gstin[0:2] != stateCode {
				problems = append(problems, fmt.Sprintf("division %q: GSTIN %q state prefix %q does not match state_code %q", div.Key, div.India.GSTIN, gstin[0:2], stateCode))
			}
			if pan != "" && ValidPANFormat(pan) && PANFromGSTIN(gstin) != pan {
				problems = append(problems, fmt.Sprintf("division %q: GSTIN %q does not embed company PAN %q", div.Key, div.India.GSTIN, ov.India.PAN))
			}
		}
	}

	return problems
}
