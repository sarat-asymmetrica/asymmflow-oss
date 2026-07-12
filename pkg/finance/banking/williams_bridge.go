package banking

import "ph_holdings_app/pkg/math/vedic"

// WilliamsBatchSize returns the optimal batch size for processing n reconciliation items.
func WilliamsBatchSize(n int) int {
	return vedic.WilliamsBatchSizeInt(n)
}
