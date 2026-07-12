package health

import "ph_holdings_app/pkg/math/vedic"

// SystemDigitalRoot computes the DR signature of a system metric value.
func SystemDigitalRoot(value int64) int64 {
	return vedic.DigitalRoot(value)
}
