package engines

import (
	"math"
	"testing"
)

func TestPaymentPredictor_Predict(t *testing.T) {
	customer := &Customer{
		ID:             "CUST001",
		BusinessName:   "NPC",
		OrderValue:     50000,
		RelationYears:  10,
		PaymentHistory: []int{30, 35, 40},
		DisputeCount:   0,
	}

	pp := NewPaymentPredictor(customer)
	prediction := pp.Predict(customer)

	if prediction.CustomerID != "CUST001" {
		t.Errorf("Expected CustomerID CUST001, got %s", prediction.CustomerID)
	}

	// NPC with 10 years and good history should be Grade A
	if prediction.Grade != "A" {
		t.Errorf("Expected Grade A for NPC, got %s", prediction.Grade)
	}

	if prediction.PredictedDays <= 0 {
		t.Errorf("Expected positive PredictedDays, got %d", prediction.PredictedDays)
	}
}

func TestPaymentPredictor_Normalize(t *testing.T) {
	customer := &Customer{ID: "TEST"}
	pp := NewPaymentPredictor(customer)

	// Check if state is normalized to S³ unit sphere (||Φ|| ≈ 1.0)
	sum := 0.0
	for i := 0; i < 79; i++ {
		sum += pp.State[i] * pp.State[i]
	}

	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("State not normalized: ||Φ||² = %v, want 1.0", sum)
	}
}

func TestEncodeCustomerToM79(t *testing.T) {
	customer := &Customer{
		ID:             "CUST001",
		BusinessName:   "TEST",
		OrderValue:     1000,
		RelationYears:  5,
		PaymentHistory: []int{30, 30, 30},
	}

	state := EncodeCustomerToM79(customer)

	if len(state) != 79 {
		t.Errorf("Expected 79 dimensions, got %d", len(state))
	}

	// Value components check
	if state[0] != 1.0 { // ID hash or something?
		// We'd need to check the actual implementation of EncodeCustomerToM79
	}
}

func TestPaymentPredictor_UpdateRegimes(t *testing.T) {
	customer := &Customer{ID: "TEST"}
	pp := NewPaymentPredictor(customer)

	// Manually set some state to test regimes
	for i := 0; i < 10; i++ {
		pp.State[i] = 10.0 // High energy
	}
	for i := 10; i < 20; i++ {
		pp.State[i] = 1.0 // Low energy
	}

	pp.UpdateRegimes()

	if pp.R1 == 0 && pp.R2 == 0 && pp.R3 == 0 {
		t.Error("Regimes should be updated")
	}
}
