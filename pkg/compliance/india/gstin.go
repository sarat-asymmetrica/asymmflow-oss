package india

import (
	"fmt"
	"regexp"
	"strings"
)

// gstinPattern matches the 15-character GSTIN shape: 2-digit state code +
// 10-char PAN + 1 entity code (1-9,A-Z) + literal "Z" + 1 check digit.
// Moved here from gst.go (formerly the sole validation — format only).
var gstinPattern = regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$`)

// panPattern matches the 10-character PAN shape: 5 letters, 4 digits, 1 letter.
var panPattern = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`)

// gstinCharset is the base-36 alphabet used by the official GSTIN check-digit
// algorithm: digits 0-9 then A-Z, each character's index is its value.
const gstinCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// ValidGSTINFormat reports whether gstin matches the 15-character GSTIN shape
// (state + PAN + entity code + "Z" + check digit). It does not verify the
// check digit or the state code — see ValidGSTIN for the full check.
func ValidGSTINFormat(gstin string) bool {
	return gstinPattern.MatchString(strings.ToUpper(strings.TrimSpace(gstin)))
}

// ValidPANFormat reports whether pan matches the 10-character PAN shape
// ([A-Z]{5}[0-9]{4}[A-Z]).
func ValidPANFormat(pan string) bool {
	return panPattern.MatchString(strings.ToUpper(strings.TrimSpace(pan)))
}

// PANFromGSTIN extracts the 10-character PAN embedded in a GSTIN (characters
// 3-12, 0-indexed [2:12]). It does not validate either string — callers that
// need a guarantee should check ValidGSTIN/ValidPANFormat first.
func PANFromGSTIN(gstin string) string {
	gstin = strings.ToUpper(strings.TrimSpace(gstin))
	if len(gstin) < 12 {
		return ""
	}
	return gstin[2:12]
}

// gstinCharValue returns the base-36 value (0-35) of a GSTIN alphabet
// character and whether it was recognised.
func gstinCharValue(c byte) (int, bool) {
	idx := strings.IndexByte(gstinCharset, c)
	if idx < 0 {
		return 0, false
	}
	return idx, true
}

// GSTINCheckDigit computes the official GSTN mod-36 check digit for the
// first 14 characters of a GSTIN (2-digit state + 10-char PAN + entity code +
// literal "Z"). Weights alternate 1,2 left-to-right; each weighted value is
// reduced to (quotient + remainder) of its division by 36 before summing;
// the check digit is (36 - sum mod 36) mod 36, mapped back through the
// base-36 alphabet.
func GSTINCheckDigit(first14 string) (byte, error) {
	first14 = strings.ToUpper(strings.TrimSpace(first14))
	if len(first14) != 14 {
		return 0, fmt.Errorf("india: GSTIN check-digit input must be 14 characters, got %d", len(first14))
	}
	sum := 0
	for i := 0; i < 14; i++ {
		value, ok := gstinCharValue(first14[i])
		if !ok {
			return 0, fmt.Errorf("india: invalid GSTIN character %q at position %d", first14[i], i+1)
		}
		factor := 1
		if (i+1)%2 == 0 { // positions 1,3,5,...=1; positions 2,4,6,...=2 (left→right)
			factor = 2
		}
		digit := value * factor
		sum += digit/36 + digit%36
	}
	checkValue := (36 - sum%36) % 36
	return gstinCharset[checkValue], nil
}

// ValidGSTIN reports whether gstin is a fully valid GSTIN: correct 15-char
// format, a known GST state code, AND a check digit that matches the
// official algorithm. This supersedes the earlier format-only check (moved
// from gst.go) — existing callers keep the same name and signature.
func ValidGSTIN(gstin string) bool {
	gstin = strings.ToUpper(strings.TrimSpace(gstin))
	if !ValidGSTINFormat(gstin) {
		return false
	}
	if !ValidStateCode(gstin[0:2]) {
		return false
	}
	want, err := GSTINCheckDigit(gstin[0:14])
	if err != nil {
		return false
	}
	return gstin[14] == want
}

// MakeGSTIN constructs a checksum-valid GSTIN from a 2-digit state code, a
// 10-character PAN, and a single entity code character (1-9 or A-Z,
// distinguishing multiple registrations of the same PAN within one state).
// Used by fixtures and tests to build synthetic-but-valid GSTINs instead of
// hand-computing check digits.
func MakeGSTIN(stateCode, pan string, entityCode byte) (string, error) {
	stateCode = strings.TrimSpace(stateCode)
	pan = strings.ToUpper(strings.TrimSpace(pan))
	if !ValidStateCode(stateCode) {
		return "", fmt.Errorf("india: unknown GST state code %q", stateCode)
	}
	if !ValidPANFormat(pan) {
		return "", fmt.Errorf("india: invalid PAN format %q", pan)
	}
	entityCode = byte(strings.ToUpper(string(entityCode))[0])
	if _, ok := gstinCharValue(entityCode); !ok || entityCode == '0' {
		return "", fmt.Errorf("india: entity code must be 1-9 or A-Z, got %q", entityCode)
	}
	base := stateCode + pan + string(entityCode) + "Z"
	check, err := GSTINCheckDigit(base)
	if err != nil {
		return "", err
	}
	return base + string(check), nil
}
