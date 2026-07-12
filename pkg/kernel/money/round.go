package money

import "math"

// RoundFloat64 rounds a float64 to the given number of decimal places.
// This is the canonical replacement for all duplicated round() functions
// in the codebase that do: math.Round(v * multiplier) / multiplier
//
// For common scales (2 and 3), a pre-computed multiplier avoids the
// math.Pow10 call entirely.
//
//	RoundFloat64(125.5556, 3) == 125.556
//	RoundFloat64(125.555,  2) == 125.56
func RoundFloat64(v float64, scale int) float64 {
	switch scale {
	case 2:
		return math.Round(v*100) / 100
	case 3:
		return math.Round(v*1000) / 1000
	default:
		m := math.Pow10(scale)
		return math.Round(v*m) / m
	}
}
