// ═══════════════════════════════════════════════════════════════════════════
// PREDICTOR - M⁷⁹ Manifold Payment Prediction Engine
//
// MATHEMATICAL FOUNDATION:
//   - Encodes customer to 79-D state vector (M⁷⁹ manifold)
//   - Normalizes to S³ unit sphere (||Φ|| = 1.0)
//   - Classifies into three regimes (R1, R2, R3)
//   - Predicts payment days and assigns customer grade (A/B/C/D)
//
// THREE-REGIME DYNAMICS:
//   R1 (Risky):   High energy outliers
//   R2 (Moderate): Transition zone
//   R3 (Stable):  Low energy, reliable customers
//
// Built with LOVE × SIMPLICITY × TRUTH × JOY 🕉️💎⚡
// ═══════════════════════════════════════════════════════════════════════════

package engines

import (
	"fmt"
	"math"
	"time"
)

// M79_DIM is the dimension of the customer state manifold
const M79_DIM = 79

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
	IsEmergency    bool      `json:"is_emergency"`    // Premium flag
	HasABB         bool      `json:"has_abb"`         // Competition flag
	DisputeCount   int       `json:"dispute_count"`   // Disputes
}

// ThreeRegime represents the three-regime distribution
type ThreeRegime struct {
	R1 float64 `json:"r1"` // Risky regime percentage
	R2 float64 `json:"r2"` // Moderate regime percentage
	R3 float64 `json:"r3"` // Stable regime percentage
}

// PaymentPrediction represents prediction output
type PaymentPrediction struct {
	CustomerID        string      `json:"customer_id"`
	CustomerName      string      `json:"customer_name"`
	Grade             string      `json:"grade"`          // A/B/C/D
	PredictedDays     int         `json:"predicted_days"` // 45-180
	Confidence        float64     `json:"confidence"`     // 0.0-1.0
	ThreeRegimes      ThreeRegime `json:"three_regimes"`  // Legacy-compatible field
	Regimes           ThreeRegime `json:"regimes"`        // Preferred typed field
	RiskFactors       []string    `json:"risk_factors"`
	RecommendedAction string      `json:"recommended_action"`
	Timestamp         string      `json:"timestamp"`
}

// PaymentPredictor predicts customer payment behavior using M⁷⁹ manifold
type PaymentPredictor struct {
	State      [79]float64 // M⁷⁹ customer state
	Energy     float64     // ||Φ|| magnitude before normalization
	R1, R2, R3 float64     // Three-regime distribution
}

// NewPaymentPredictor creates predictor from customer
func NewPaymentPredictor(customer *Customer) *PaymentPredictor {
	pp := &PaymentPredictor{}

	// Stage 1: Encode customer to M⁷⁹ manifold
	pp.State = EncodeCustomerToM79(customer)

	// Stage 2: Compute three-regime distribution BEFORE normalization
	pp.UpdateRegimes()

	// Stage 3: Normalize to S³ unit sphere (geometric projection)
	pp.Normalize()

	return pp
}

// Normalize projects state to S³ unit sphere (||Φ|| = 1.0)
func (pp *PaymentPredictor) Normalize() {
	sum := 0.0
	for i := 0; i < 79; i++ {
		sum += pp.State[i] * pp.State[i]
	}
	pp.Energy = math.Sqrt(sum)

	// Prevent division by zero
	if pp.Energy > 1e-10 {
		for i := 0; i < 79; i++ {
			pp.State[i] /= pp.Energy
		}
		pp.Energy = 1.0 // After normalization, energy = 1
	}
}

// UpdateRegimes computes three-regime distribution
func (pp *PaymentPredictor) UpdateRegimes() {
	// Compute mean of absolute component magnitudes
	mean := 0.0
	for i := 0; i < 79; i++ {
		mean += math.Abs(pp.State[i])
	}
	mean /= 79.0

	// Compute variance and standard deviation
	variance := 0.0
	for i := 0; i < 79; i++ {
		diff := math.Abs(pp.State[i]) - mean
		variance += diff * diff
	}
	variance /= 79.0
	stdDev := math.Sqrt(variance)

	// Classify each component into one of three regimes
	highEnergy := 0 // Components with high magnitude (risky)
	medEnergy := 0  // Components with medium magnitude
	lowEnergy := 0  // Components with low magnitude (stable)

	threshold1 := mean + 0.5*stdDev // Upper threshold
	threshold2 := mean - 0.5*stdDev // Lower threshold

	for i := 0; i < 79; i++ {
		abs := math.Abs(pp.State[i])
		if abs > threshold1 {
			highEnergy++ // High magnitude = more risk signal
		} else if abs > threshold2 {
			medEnergy++ // Medium magnitude
		} else {
			lowEnergy++ // Low magnitude = stable signal
		}
	}

	// Score based on key components:
	// - state[11-30]: Payment history (lower = better)
	// - state[31-40]: Relationship tenure (higher = better)
	// - state[51-60]: Risk factors (lower = better)

	paymentScore := 0.0
	for i := 11; i <= 30; i++ {
		paymentScore += math.Abs(pp.State[i])
	}
	paymentScore /= 20.0 // Average magnitude - HIGHER = WORSE

	relationScore := 0.0
	for i := 31; i <= 40; i++ {
		relationScore += math.Abs(pp.State[i])
	}
	relationScore /= 10.0 // Average magnitude - HIGHER = BETTER (longer relationship)

	riskScore := 0.0
	for i := 51; i <= 60; i++ {
		riskScore += math.Abs(pp.State[i])
	}
	riskScore /= 10.0 // Average magnitude - HIGHER = WORSE

	// Composite stability score: high relation, low payment, low risk = stable
	stabilityScore := relationScore*3.0 - paymentScore*2.0 - riskScore*2.0 + 0.5

	// Map stability score to three regimes
	if stabilityScore > 0.2 {
		// Stable customer - Grade A
		pp.R1 = 0.20
		pp.R2 = 0.25
		pp.R3 = 0.55
	} else if stabilityScore > -0.5 {
		// Moderate customer - Grade B
		pp.R1 = 0.30
		pp.R2 = 0.30
		pp.R3 = 0.40
	} else if stabilityScore > -1.05 {
		// Risky customer - Grade C
		pp.R1 = 0.40
		pp.R2 = 0.35
		pp.R3 = 0.25
	} else {
		// High risk customer - Grade D
		pp.R1 = 0.55
		pp.R2 = 0.30
		pp.R3 = 0.15
	}

	// Note: We still compute the original counts for reference
	_ = highEnergy
	_ = medEnergy
	_ = lowEnergy
}

// Predict generates payment prediction from three-regime classification
func (pp *PaymentPredictor) Predict(customer *Customer) PaymentPrediction {
	pred := PaymentPrediction{
		CustomerID:   customer.ID,
		CustomerName: customer.BusinessName,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		RiskFactors:  make([]string, 0),
	}

	pred.ThreeRegimes.R1 = pp.R1
	pred.ThreeRegimes.R2 = pp.R2
	pred.ThreeRegimes.R3 = pp.R3
	pred.Regimes = pred.ThreeRegimes

	// Grade assignment based on R3 (stability regime percentage)
	if pp.R3 >= 0.50 {
		pred.Grade = "A"
		pred.PredictedDays = 45
		pred.Confidence = 0.90
		pred.RecommendedAction = "✓ APPROVE: Max discount 7%, reliable customer"
	} else if pp.R3 >= 0.35 {
		pred.Grade = "B"
		pred.PredictedDays = 90
		pred.Confidence = 0.75
		pred.RecommendedAction = "✓ APPROVE: Max discount 3%, moderate risk"
	} else if pp.R3 >= 0.20 {
		pred.Grade = "C"
		pred.PredictedDays = 120
		pred.Confidence = 0.60
		pred.RecommendedAction = "⚠ CAUTION: No discount, require 50% advance"
	} else {
		pred.Grade = "D"
		pred.PredictedDays = 180
		pred.Confidence = 0.40
		pred.RecommendedAction = "✗ DECLINE: High risk, require 100% advance or decline"
	}

	// Adjustments based on customer features

	// Long relationship bonus
	if customer.RelationYears >= 5 {
		pred.PredictedDays -= 10
		pred.Confidence += 0.05
		pred.RiskFactors = append(pred.RiskFactors, "Long relationship (+trust)")
	}

	// Emergency order adjustment
	if customer.IsEmergency {
		pred.PredictedDays -= 15
		pred.RiskFactors = append(pred.RiskFactors, "Emergency order (premium pricing)")
	}

	// ABB competition warning
	if customer.HasABB {
		pred.RecommendedAction += " | ⚠ ABB COMPETING - Consider declining if margin < 15%"
		pred.RiskFactors = append(pred.RiskFactors, "ABB competition detected")
		pred.Confidence -= 0.10
	}

	// Dispute history penalty
	if customer.DisputeCount > 2 {
		pred.RiskFactors = append(pred.RiskFactors,
			fmt.Sprintf("High dispute history (%d disputes)", customer.DisputeCount))
		pred.Confidence -= 0.05 * float64(customer.DisputeCount)
	}

	// Clamp confidence to [0, 1]
	if pred.Confidence > 1.0 {
		pred.Confidence = 1.0
	}
	if pred.Confidence < 0.0 {
		pred.Confidence = 0.0
	}

	// Ensure predicted days is positive
	if pred.PredictedDays < 15 {
		pred.PredictedDays = 15 // Minimum 15 days
	}

	return pred
}

// EncodeCustomerToM79 maps customer to 79-D state using M⁷⁹ manifold encoding
func EncodeCustomerToM79(c *Customer) [M79_DIM]float64 {
	var state [M79_DIM]float64

	// Component 0-10: Order value (log scale, harmonic)
	orderLog := math.Log1p(c.OrderValue)
	for i := 0; i < 11; i++ {
		state[i] = (orderLog / math.Sqrt(float64(i+1))) * 0.1
	}

	// Component 11-30: Payment history (temporal pattern)
	if len(c.PaymentHistory) > 0 {
		avg := averageInt(c.PaymentHistory)
		variance := computeVarianceInt(c.PaymentHistory, avg)

		for i := 0; i < 20; i++ {
			freq := float64(i) * 0.1
			val := ((avg/90.0)*math.Sin(freq) + (math.Sqrt(variance)/30.0)*math.Cos(freq)) * 0.5
			state[11+i] = val
		}
	}

	// Component 31-40: Relationship tenure (exponential decay)
	relationFactor := float64(c.RelationYears) / 20.0
	for i := 0; i < 10; i++ {
		state[31+i] = relationFactor * math.Exp(-float64(i)*0.1)
	}

	// Component 41-50: Order variability (tanh activation)
	if len(c.OrderHistory) > 1 {
		mean := averageFloat(c.OrderHistory)
		variance := computeVarianceFloat(c.OrderHistory, mean)

		for i := 0; i < 10; i++ {
			state[41+i] = (math.Sqrt(variance) / (mean + 1.0)) *
				math.Tanh(float64(i)-5.0)
		}
	}

	// Component 51-60: Risk factors (sinusoidal encoding)
	riskFactor := 0.5
	if c.IsEmergency {
		riskFactor -= 0.2 // Emergency = more reliable payment
	}
	if c.HasABB {
		riskFactor += 0.3 // ABB competition = less likely to pay
	}
	riskFactor += float64(c.DisputeCount) * 0.05

	for i := 0; i < 10; i++ {
		state[51+i] = riskFactor * math.Sin(float64(i)*0.5)
	}

	// Component 61-70: Geographic/industry encoding
	industryHash := hashString(c.Industry)
	geoHash := hashString(c.Country)

	for i := 0; i < 10; i++ {
		state[61+i] = ((float64(industryHash%100)/100.0)*math.Cos(float64(i)) +
			(float64(geoHash%100)/100.0)*math.Sin(float64(i))) * 0.1
	}

	// Component 71-78: Harmonic structure (fractal coupling)
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
func hashString(s string) int {
	hash := 0
	for _, c := range s {
		hash = (hash*31 + int(c)) % 10000
	}
	return hash
}

// sumFloat computes sum of float slice
func sumFloat(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum
}
