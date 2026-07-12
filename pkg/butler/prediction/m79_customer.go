package prediction

import (
	"math"
)

// Customer represents Acme Instrumentation customer data
type Customer struct {
	ID             string    `json:"id"`
	BusinessName   string    `json:"business_name"`
	OrderValue     float64   `json:"order_value"`     // BHD
	OrderHistory   []float64 `json:"order_history"`   // Past orders
	PaymentHistory []int     `json:"payment_history"` // Past payment days
	RelationYears  int       `json:"relation_years"`  // Tenure
	Industry       string    `json:"industry"`        // Sector
	Country        string    `json:"country"`         // Region
	IsEmergency    int       `json:"is_emergency"`    // Premium flag (0/1)
	HasABB         int       `json:"has_abb"`         // Competition flag (0/1)
	DisputeCount   int       `json:"dispute_count"`   // Disputes
}

// M79_DIM is the dimension of the customer state manifold
const M79_DIM = 79

// EncodeCustomerToM79 maps customer to 79-D state using M⁷⁹ manifold encoding
// This is the MATHEMATICAL CORE - it transforms raw business data into
// a geometric representation on the 79-dimensional Vedic manifold.
//
// The encoding preserves the three-regime dynamics:
// - Component magnitudes determine R1/R2/R3 classification
// - Low magnitude components → R3 (Stable) → Grade A/B
// - High magnitude components → R1 (Risky) → Grade C/D
func EncodeCustomerToM79(c *Customer) [M79_DIM]float64 {
	var state [M79_DIM]float64

	// Component 0-10: Order value (log scale, harmonic)
	// Log scale prevents large orders from dominating the state vector.
	// Harmonic 1/√(i+1) decay creates natural frequency structure.
	// Scale factor 0.1 keeps magnitudes low for stable customers.
	orderLog := math.Log1p(c.OrderValue)
	for i := 0; i < 11; i++ {
		state[i] = (orderLog / math.Sqrt(float64(i+1))) * 0.1
	}

	// Component 11-30: Payment history (temporal pattern)
	// Encodes BOTH average payment time AND variance using sinusoidal basis.
	// Good customers (low avg, low variance) → LOW magnitudes → R3
	// Bad customers (high avg, high variance) → HIGH magnitudes → R1
	if len(c.PaymentHistory) > 0 {
		avg := averageInt(c.PaymentHistory)
		variance := computeVarianceInt(c.PaymentHistory, avg)

		for i := 0; i < 20; i++ {
			freq := float64(i) * 0.1

			// avg/90.0 normalizes to [0,1] assuming 90 days baseline
			// variance/30.0 normalizes volatility
			// Scale 0.5 keeps stable customers in R3 regime
			val := ((avg/90.0)*math.Sin(freq) + (math.Sqrt(variance)/30.0)*math.Cos(freq)) * 0.5
			state[11+i] = val
		}
	}

	// Component 31-40: Relationship tenure (exponential decay)
	// Long relationships → LOWER magnitudes (exponential decay) → R3
	// New customers → HIGHER magnitudes → R1/R2
	// This rewards loyalty with stability classification.
	relationFactor := float64(c.RelationYears) / 20.0
	for i := 0; i < 10; i++ {
		state[31+i] = relationFactor * math.Exp(-float64(i)*0.1)
	}

	// Component 41-50: Order variability (tanh activation)
	// High variance relative to mean → HIGH magnitudes → R1
	// Consistent order sizes → LOW magnitudes → R3
	// Tanh creates smooth nonlinearity around center.
	if len(c.OrderHistory) > 1 {
		mean := averageFloat(c.OrderHistory)
		variance := computeVarianceFloat(c.OrderHistory, mean)

		for i := 0; i < 10; i++ {
			state[41+i] = (math.Sqrt(variance) / (mean + 1.0)) *
				math.Tanh(float64(i)-5.0)
		}
	}

	// Component 51-60: Risk factors (sinusoidal encoding)
	// Binary flags encoded as continuous risk factor
	// Emergency orders (premium) → LOWER risk (−0.2)
	// ABB competition → HIGHER risk (+0.3)
	// Disputes → LINEAR risk increase (+0.05 per dispute)
	riskFactor := 0.5
	if c.IsEmergency == 1 {
		riskFactor -= 0.2 // Emergency = more reliable payment
	}
	if c.HasABB == 1 {
		riskFactor += 0.3 // ABB competition = less likely to pay
	}
	riskFactor += float64(c.DisputeCount) * 0.05

	for i := 0; i < 10; i++ {
		state[51+i] = riskFactor * math.Sin(float64(i)*0.5)
	}

	// Component 61-70: Geographic/industry encoding
	// Hash-based encoding with reduced magnitude (0.1 scale).
	// Prevents random geographic/industry noise from dominating classification.
	// Modest contribution to overall state vector.
	industryHash := hashString(c.Industry)
	geoHash := hashString(c.Country)

	for i := 0; i < 10; i++ {
		state[61+i] = ((float64(industryHash%100)/100.0)*math.Cos(float64(i)) +
			(float64(geoHash%100)/100.0)*math.Sin(float64(i))) * 0.1
	}

	// Component 71-78: Harmonic structure (fractal coupling)
	// Cross-term coupling between order value and relationship.
	// Creates fractal self-similarity in the manifold.
	// Sin/Cos products generate interference patterns that
	// encode nonlinear interactions between features.
	for i := 71; i < 79; i++ {
		freq := float64(i - 70)
		state[i] = math.Sin(freq*orderLog) * math.Cos(freq*relationFactor) * 0.1
	}

	return state
}

// Helper functions for statistical calculations

// averageInt computes mean of integer slice
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

// computeVarianceInt computes variance of integer slice
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

// averageFloat computes mean of float slice
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

// computeVarianceFloat computes variance of float slice
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

// hashString provides simple string hashing for categorical encoding
// Uses polynomial rolling hash with prime modulus
func hashString(s string) int {
	hash := 0
	for _, c := range s {
		hash = (hash*31 + int(c)) % 10000
	}
	return hash
}

// boolToInt converts a boolean to an integer (0 or 1) for SQLite compatibility
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
