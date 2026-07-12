package main

import butlerprediction "ph_holdings_app/pkg/butler/prediction"

// Customer represents Acme Instrumentation customer data for M79 payment prediction.
type Customer = butlerprediction.Customer

// M79_DIM is the dimension of the customer state manifold.
const M79_DIM = butlerprediction.M79_DIM

func EncodeCustomerToM79(c *Customer) [M79_DIM]float64 {
	return butlerprediction.EncodeCustomerToM79(c)
}

// averageInt computes mean of integer slice.
func averageInt(values []int) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += float64(v)
	}
	return sum / float64(len(values))
}

// computeVarianceInt computes variance of integer slice.
func computeVarianceInt(values []int, mean float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	variance := 0.0
	for _, v := range values {
		diff := float64(v) - mean
		variance += diff * diff
	}
	return variance / float64(len(values))
}

// averageFloat computes mean of float slice.
func averageFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// computeVarianceFloat computes variance of float slice.
func computeVarianceFloat(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	return variance / float64(len(values))
}

// hashString provides simple string hashing for categorical encoding.
func hashString(s string) int {
	hash := 0
	for _, c := range s {
		hash = (hash*31 + int(c)) % 10000
	}
	return hash
}

// boolToInt converts a boolean to an integer (0 or 1) for SQLite compatibility.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
