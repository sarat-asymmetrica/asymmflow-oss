// Package money provides a precise monetary value type for use across the
// ph_holdings_app codebase. It is the canonical replacement for all duplicated
// round() helpers (pkg/cashflow/evidence/model.go, pkg/finance/posting/posting.go).
//
// Design principles:
//   - Amount is an immutable value type; fields are unexported.
//   - Arithmetic is performed on integer minor units to avoid float drift.
//   - Zero stdlib dependencies beyond "math" and "fmt".
package money

import (
	"fmt"
	"math"
)

// Amount represents a precise monetary value stored as integer minor units.
//
// Examples:
//
//	BHD 125.500  → value=125500, currency="BHD", scale=3
//	USD 10.99    → value=1099,   currency="USD", scale=2
type Amount struct {
	value    int64
	currency string
	scale    int // decimal places (3 for BHD, 2 for USD)
}

// BHD constructs an Amount for Bahraini Dinar (scale=3).
// The float64 is rounded to 3 decimal places before conversion to minor units.
//
//	BHD(125.5556) → minor units = 125556, Float64() = 125.556
func BHD(v float64) Amount {
	return Amount{
		value:    int64(math.Round(v * 1000)),
		currency: "BHD",
		scale:    3,
	}
}

// FromMinor constructs an Amount directly from minor units.
// Use this when you already have an integer minor-unit value (e.g. from a DB column).
//
//	FromMinor(125500, "BHD", 3).Float64() == 125.5
func FromMinor(minor int64, currency string, scale int) Amount {
	return Amount{value: minor, currency: currency, scale: scale}
}

// Float64 converts the Amount to a float64.
// Loss of precision is possible for very large values; prefer Minor() for exact arithmetic.
func (a Amount) Float64() float64 {
	return float64(a.value) / math.Pow10(a.scale)
}

// Minor returns the raw integer minor-unit value.
//
//	BHD(125.5).Minor() == 125500
func (a Amount) Minor() int64 { return a.value }

// Currency returns the ISO 4217 currency code (e.g. "BHD", "USD").
func (a Amount) Currency() string { return a.currency }

// Scale returns the number of decimal places used for this currency.
//
//	BHD → 3, USD → 2
func (a Amount) Scale() int { return a.scale }

// Add returns a + b. Returns an error if currencies differ.
func (a Amount) Add(b Amount) (Amount, error) {
	if a.currency != b.currency {
		return Amount{}, fmt.Errorf("money: cannot add %s to %s", b.currency, a.currency)
	}
	return Amount{value: a.value + b.value, currency: a.currency, scale: a.scale}, nil
}

// Sub returns a - b. Returns an error if currencies differ.
func (a Amount) Sub(b Amount) (Amount, error) {
	if a.currency != b.currency {
		return Amount{}, fmt.Errorf("money: cannot subtract %s from %s", b.currency, a.currency)
	}
	return Amount{value: a.value - b.value, currency: a.currency, scale: a.scale}, nil
}

// IsZero reports whether the amount is exactly zero.
func (a Amount) IsZero() bool { return a.value == 0 }

// IsPositive reports whether the amount is strictly greater than zero.
func (a Amount) IsPositive() bool { return a.value > 0 }

// IsNegative reports whether the amount is strictly less than zero.
func (a Amount) IsNegative() bool { return a.value < 0 }

// Negate returns the additive inverse of the amount.
//
//	BHD(100).Negate() == BHD(-100)
func (a Amount) Negate() Amount {
	return Amount{value: -a.value, currency: a.currency, scale: a.scale}
}

// Format returns a human-readable string such as "BHD 125.500" or "BHD -50.123".
// The value is formatted to exactly Scale() decimal places.
//
//	BHD(125.5).Format()   == "BHD 125.500"
//	BHD(0).Format()       == "BHD 0.000"
//	BHD(-50.123).Format() == "BHD -50.123"
func (a Amount) Format() string {
	format := fmt.Sprintf("%%s %%.%df", a.scale)
	return fmt.Sprintf(format, a.currency, a.Float64())
}
