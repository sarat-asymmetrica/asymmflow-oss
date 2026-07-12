// ═══════════════════════════════════════════════════════════════════════════
// BUSINESS_INVARIANTS_TEST - Validation Tests for Business Rules
//
// Demonstrates how to use business invariants for:
//   1. Unit testing business logic
//   2. Integration testing with real data
//   3. Production validation
//   4. Audit compliance
//
// Built with LOVE × SIMPLICITY × TRUTH × JOY 🕉️💎⚡
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"fmt"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// TEST: Customer Grading Invariants
// ═══════════════════════════════════════════════════════════════════════════

func TestGradeA_Customer_MeetsInvariants(t *testing.T) {
	// Create Grade A customer (excellent payment history, long relationship)
	customer := &Customer{
		ID:             "CUST-001",
		BusinessName:   "Reliable Industries LLC",
		OrderValue:     5000.0,
		OrderHistory:   []float64{4500, 4800, 5200, 4900},
		PaymentHistory: []int{35, 40, 38, 42}, // Pays within 35-42 days
		RelationYears:  8,
		Industry:       "Manufacturing",
		Country:        "Bahrain",
		IsEmergency:    0,
		HasABB:         0,
		DisputeCount:   0,
	}

	// Generate prediction
	predictor := NewPaymentPredictor(customer)
	prediction := predictor.Predict(customer)

	// Create validation context
	ctx := Context{
		Customer:   customer,
		Prediction: &prediction,
	}

	// Validate all invariants
	results := ValidateAll(ctx)

	// Check results
	for _, r := range results {
		if !r.Passed && r.Severity == SeverityCritical {
			t.Errorf("CRITICAL invariant failed: %s - %v", r.InvariantName, r.Error)
		}
	}

	// Specific checks for Grade A
	if prediction.Grade != "A" {
		t.Errorf("Expected Grade A, got Grade %s", prediction.Grade)
	}

	if prediction.ThreeRegimes.R3 < 0.50 {
		t.Errorf("Grade A should have R3 ≥ 50%%, got %.1f%%", prediction.ThreeRegimes.R3*100)
	}
}

func TestGradeD_Customer_MeetsInvariants(t *testing.T) {
	// Create Grade D customer (poor payment history, disputes)
	customer := &Customer{
		ID:             "CUST-999",
		BusinessName:   "Risky Ventures Ltd",
		OrderValue:     3000.0,
		OrderHistory:   []float64{2500, 3500, 2800},
		PaymentHistory: []int{150, 180, 165, 175, 190}, // Pays in 150-190 days
		RelationYears:  1,
		Industry:       "Construction",
		Country:        "Bahrain",
		IsEmergency:    0,
		HasABB:         0,
		DisputeCount:   0,
	}

	// Generate prediction
	predictor := NewPaymentPredictor(customer)
	prediction := predictor.Predict(customer)

	// Create validation context
	ctx := Context{
		Customer:   customer,
		Prediction: &prediction,
	}

	// Validate all invariants
	results := ValidateAll(ctx)

	// Check results
	for _, r := range results {
		if !r.Passed && r.Severity == SeverityCritical {
			t.Errorf("CRITICAL invariant failed: %s - %v", r.InvariantName, r.Error)
		}
	}

	// Specific checks for Grade D
	if prediction.Grade != "D" {
		t.Errorf("Expected Grade D, got Grade %s", prediction.Grade)
	}

	if prediction.ThreeRegimes.R3 >= 0.20 {
		t.Errorf("Grade D should have R3 < 20%%, got %.1f%%", prediction.ThreeRegimes.R3*100)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST: Three Regimes Sum to Unity
// ═══════════════════════════════════════════════════════════════════════════

func TestThreeRegimes_SumToUnity(t *testing.T) {
	testCases := []struct {
		name          string
		paymentDays   []int
		relationYears int
	}{
		{"Excellent Customer", []int{30, 35, 32, 38}, 10},
		{"Good Customer", []int{60, 65, 70, 68}, 5},
		{"Average Customer", []int{90, 95, 100, 88}, 3},
		{"Poor Customer", []int{120, 130, 125, 135}, 2},
		{"Very Poor Customer", []int{160, 170, 180, 175}, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			customer := &Customer{
				ID:             "TEST-" + tc.name,
				BusinessName:   tc.name,
				OrderValue:     5000.0,
				PaymentHistory: tc.paymentDays,
				RelationYears:  tc.relationYears,
			}

			predictor := NewPaymentPredictor(customer)
			prediction := predictor.Predict(customer)

			ctx := Context{
				Customer:   customer,
				Prediction: &prediction,
			}

			// Validate unity invariant
			err := ThreeRegimes_Sum_Unity.Assertion(ctx)
			if err != nil {
				t.Errorf("Three regimes do not sum to unity: %v", err)
			}

			// Log the distribution for visibility
			t.Logf("%s: R1=%.1f%%, R2=%.1f%%, R3=%.1f%%, Grade=%s",
				tc.name,
				prediction.ThreeRegimes.R1*100,
				prediction.ThreeRegimes.R2*100,
				prediction.ThreeRegimes.R3*100,
				prediction.Grade)
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST: Discount Invariants
// ═══════════════════════════════════════════════════════════════════════════

func TestDiscount_RulesEnforced(t *testing.T) {
	testCases := []struct {
		grade           CustomerGrade
		appliedDiscount float64
		shouldPass      bool
	}{
		{GradeA, 0.05, true},  // 5% for Grade A (max 7%) - OK
		{GradeA, 0.07, true},  // 7% for Grade A (max 7%) - OK
		{GradeA, 0.10, false}, // 10% for Grade A (max 7%) - FAIL
		{GradeB, 0.03, true},  // 3% for Grade B (max 3%) - OK
		{GradeB, 0.05, false}, // 5% for Grade B (max 3%) - FAIL
		{GradeC, 0.00, true},  // 0% for Grade C (max 0%) - OK
		{GradeC, 0.02, false}, // 2% for Grade C (max 0%) - FAIL
		{GradeD, 0.00, true},  // 0% for Grade D (max 0%) - OK
		{GradeD, 0.01, false}, // 1% for Grade D (max 0%) - FAIL
	}

	for _, tc := range testCases {
		t.Run(string(tc.grade)+"_"+formatPercent(tc.appliedDiscount), func(t *testing.T) {
			customer := &Customer{
				ID:           "TEST-DISCOUNT",
				BusinessName: "Test Customer",
			}

			sheet := &CostingSheet{
				CustomerGrade: tc.grade,
				Items: []CostingItem{
					{
						OrderCode:        "TEST-001",
						CustomerDiscount: tc.appliedDiscount,
					},
				},
			}

			ctx := Context{
				Customer:     customer,
				CostingSheet: sheet,
			}

			// Select appropriate invariant based on grade
			var invariant BusinessInvariant
			switch tc.grade {
			case GradeA:
				invariant = GradeA_MaxDiscount
			case GradeB:
				invariant = GradeB_MaxDiscount
			case GradeC:
				invariant = GradeC_NoDiscount
			case GradeD:
				invariant = GradeD_NoDiscount
			}

			err := invariant.Assertion(ctx)

			if tc.shouldPass && err != nil {
				t.Errorf("Expected discount %.1f%% for %s to PASS, but got error: %v",
					tc.appliedDiscount*100, tc.grade, err)
			}

			if !tc.shouldPass && err == nil {
				t.Errorf("Expected discount %.1f%% for %s to FAIL, but passed",
					tc.appliedDiscount*100, tc.grade)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST: Margin Invariants
// ═══════════════════════════════════════════════════════════════════════════

func TestProductMargins_Enforced(t *testing.T) {
	testCases := []struct {
		productType    string
		expectedMargin float64
		appliedMargin  float64
		shouldPass     bool
	}{
		{"Rhine Flow", 0.15, 0.15, true},      // Correct margin
		{"Rhine Flow", 0.15, 0.12, false},     // Wrong margin
		{"Oxan Analytics", 0.25, 0.25, true},  // Correct margin
		{"Oxan Analytics", 0.25, 0.20, false}, // Wrong margin
		{"GIC", 0.10, 0.10, true},             // Correct margin
		{"GIC", 0.10, 0.15, false},            // Wrong margin
		{"Rhine Level", 0.18, 0.18, true},     // Correct margin
		{"Rhine Analytics", 0.20, 0.20, true}, // Correct margin
	}

	for _, tc := range testCases {
		t.Run(tc.productType+"_"+formatPercent(tc.appliedMargin), func(t *testing.T) {
			sheet := &CostingSheet{
				Items: []CostingItem{
					{
						OrderCode:      "TEST-001",
						ProductType:    tc.productType,
						StandardMargin: tc.appliedMargin,
					},
				},
			}

			ctx := Context{
				CostingSheet: sheet,
			}

			// Select appropriate invariant
			var invariant BusinessInvariant
			switch tc.productType {
			case "Rhine Flow":
				invariant = RhineFlow_Margin
			case "Oxan Analytics":
				invariant = GasAnalyzer_Margin
			case "GIC":
				invariant = GIC_Margin
			default:
				t.Skip("No specific margin rule for", tc.productType)
				return
			}

			err := invariant.Assertion(ctx)

			if tc.shouldPass && err != nil {
				t.Errorf("Expected margin %.1f%% for %s to PASS, but got error: %v",
					tc.appliedMargin*100, tc.productType, err)
			}

			if !tc.shouldPass && err == nil {
				t.Errorf("Expected margin %.1f%% for %s to FAIL, but passed",
					tc.appliedMargin*100, tc.productType)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST: Risk Invariants - ABB Competition
// ═══════════════════════════════════════════════════════════════════════════

func TestABB_Competition_EnforcesMinimumMargin(t *testing.T) {
	testCases := []struct {
		margin         float64
		approvalStatus string
		shouldPass     bool
	}{
		{0.20, "✓ APPROVE", true},  // 20% margin with ABB - OK
		{0.15, "✓ APPROVE", true},  // 15% margin with ABB - OK (minimum)
		{0.12, "✗ DECLINE", true},  // 12% margin with ABB, DECLINED - OK
		{0.12, "✓ APPROVE", false}, // 12% margin with ABB, APPROVED - FAIL
		{0.08, "✗ DECLINE", true},  // 8% margin with ABB, DECLINED - OK
		{0.08, "⚠ CAUTION", false}, // 8% margin with ABB, CAUTION - FAIL
	}

	for _, tc := range testCases {
		t.Run(formatPercent(tc.margin)+"_"+tc.approvalStatus, func(t *testing.T) {
			customer := &Customer{
				ID:           "TEST-ABB",
				BusinessName: "ABB Competitor Customer",
				HasABB:       1, // ABB is competing
			}

			sheet := &CostingSheet{
				ActualMarginPct: tc.margin,
				ApprovalStatus:  tc.approvalStatus,
			}

			ctx := Context{
				Customer:     customer,
				CostingSheet: sheet,
			}

			err := ABB_Competition_Margin.Assertion(ctx)

			if tc.shouldPass && err != nil {
				t.Errorf("Expected margin %.1f%% with ABB + %s to PASS, but got error: %v",
					tc.margin*100, tc.approvalStatus, err)
			}

			if !tc.shouldPass && err == nil {
				t.Errorf("Expected margin %.1f%% with ABB + %s to FAIL, but passed",
					tc.margin*100, tc.approvalStatus)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST: Minimum Margin Threshold
// ═══════════════════════════════════════════════════════════════════════════

func TestMinimum_Margin_Threshold(t *testing.T) {
	testCases := []struct {
		margin         float64
		approvalStatus string
		shouldPass     bool
	}{
		{0.15, "✓ APPROVE", true},  // 15% margin - OK
		{0.08, "✓ APPROVE", true},  // 8% margin (minimum) - OK
		{0.07, "✗ DECLINE", true},  // 7% margin, DECLINED - OK
		{0.07, "✓ APPROVE", false}, // 7% margin, APPROVED - FAIL
		{0.05, "⚠ CAUTION", true},  // 5% margin, CAUTION - OK
		{0.05, "✓ APPROVE", false}, // 5% margin, APPROVED - FAIL
	}

	for _, tc := range testCases {
		t.Run(formatPercent(tc.margin)+"_"+tc.approvalStatus, func(t *testing.T) {
			sheet := &CostingSheet{
				ActualMarginPct: tc.margin,
				ApprovalStatus:  tc.approvalStatus,
			}

			ctx := Context{
				CostingSheet: sheet,
			}

			err := Minimum_Margin_Threshold.Assertion(ctx)

			if tc.shouldPass && err != nil {
				t.Errorf("Expected margin %.1f%% + %s to PASS, but got error: %v",
					tc.margin*100, tc.approvalStatus, err)
			}

			if !tc.shouldPass && err == nil {
				t.Errorf("Expected margin %.1f%% + %s to FAIL, but passed",
					tc.margin*100, tc.approvalStatus)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST: End-to-End Validation
// ═══════════════════════════════════════════════════════════════════════════

func TestE2E_CompleteQuotation_ValidatesAllInvariants(t *testing.T) {
	// Create a realistic Grade A customer
	customer := &Customer{
		ID:             "CUST-A001",
		BusinessName:   "Premier Oil & Gas LLC",
		OrderValue:     8000.0,
		OrderHistory:   []float64{7500, 8200, 7800, 8100},
		PaymentHistory: []int{38, 42, 40, 35, 37},
		RelationYears:  7,
		Industry:       "Oil & Gas",
		Country:        "Bahrain",
		IsEmergency:    0,
		HasABB:         0,
		DisputeCount:   0,
	}

	// Generate prediction
	predictor := NewPaymentPredictor(customer)
	prediction := predictor.Predict(customer)

	// Create mock costing sheet
	sheet := &CostingSheet{
		CustomerID:        customer.ID,
		CustomerName:      customer.BusinessName,
		CustomerGrade:     CustomerGrade(prediction.Grade),
		PaymentPrediction: prediction,
		Items: []CostingItem{
			{
				OrderCode:        "71126072",
				Description:      "Promag 50W DN50",
				ProductType:      "Rhine Flow",
				Quantity:         2,
				UnitCostBHD:      1500.0,
				TotalCostBHD:     3000.0,
				StandardMargin:   0.15, // 15% for Rhine Flow
				UnitSellBHD:      1725.0,
				TotalSellBHD:     3450.0,
				CustomerDiscount: 0.05, // 5% discount for Grade A
				FinalUnitBHD:     1638.75,
				FinalTotalBHD:    3277.50,
				UnitProfitBHD:    138.75,
				TotalProfitBHD:   277.50,
				ActualMargin:     0.0925, // 9.25% after discount
			},
			{
				OrderCode:        "SRV-GA-001",
				Description:      "Oxan Analytics Gas Analyzer",
				ProductType:      "Oxan Analytics",
				Quantity:         1,
				UnitCostBHD:      4000.0,
				TotalCostBHD:     4000.0,
				StandardMargin:   0.25, // 25% for Oxan Analytics
				UnitSellBHD:      5000.0,
				TotalSellBHD:     5000.0,
				CustomerDiscount: 0.05, // 5% discount for Grade A
				FinalUnitBHD:     4750.0,
				FinalTotalBHD:    4750.0,
				UnitProfitBHD:    750.0,
				TotalProfitBHD:   750.0,
				ActualMargin:     0.1875, // 18.75% after discount
			},
		},
		TotalCostBHD:      7000.0,
		TotalSellBHD:      8450.0,
		TotalDiscountBHD:  422.50,
		TotalFinalBHD:     8027.50,
		TotalProfitBHD:    1027.50,
		StandardMarginPct: 0.207, // Weighted average
		ActualMarginPct:   0.147, // 14.7% after discount
		PaymentTerms:      "Net 45 days",
		AdvanceRequired:   0.0,
		ApprovalStatus:    "✓ APPROVE",
	}

	// Create context
	ctx := Context{
		Customer:     customer,
		Prediction:   &prediction,
		CostingSheet: sheet,
	}

	// Validate ALL invariants
	results := ValidateAll(ctx)

	// Print results
	t.Log("═════════════════════════════════════════════════════════════")
	t.Log("E2E VALIDATION RESULTS")
	t.Log("═════════════════════════════════════════════════════════════")

	criticalFailures := 0
	warningFailures := 0

	for _, r := range results {
		if !r.Passed {
			if r.Severity == SeverityCritical {
				criticalFailures++
				t.Errorf("✗ CRITICAL: %s - %v", r.InvariantName, r.Error)
			} else if r.Severity == SeverityWarning {
				warningFailures++
				t.Logf("⚠ WARNING: %s - %v", r.InvariantName, r.Error)
			} else {
				t.Logf("ℹ INFO: %s - %v", r.InvariantName, r.Error)
			}
		}
	}

	if criticalFailures == 0 {
		t.Log("✓ All CRITICAL invariants passed!")
	}

	if warningFailures == 0 {
		t.Log("✓ All WARNING invariants passed!")
	}

	// Log summary
	t.Logf("Customer: %s (Grade %s)", customer.BusinessName, prediction.Grade)
	t.Logf("Total Value: %.2f BHD", sheet.TotalFinalBHD)
	t.Logf("Actual Margin: %.1f%%", sheet.ActualMarginPct*100)
	t.Logf("Payment Terms: %s", sheet.PaymentTerms)
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST: Invariant Registry Statistics
// ═══════════════════════════════════════════════════════════════════════════

func TestInvariant_Registry_Statistics(t *testing.T) {
	stats := InvariantStatistics()

	t.Logf("Total Invariants: %d", stats["total"])
	t.Logf("CRITICAL: %d", stats["severity_CRITICAL"])
	t.Logf("WARNING: %d", stats["severity_WARNING"])
	t.Logf("INFO: %d", stats["severity_INFO"])

	t.Logf("GRADING: %d", stats["category_GRADING"])
	t.Logf("PAYMENT: %d", stats["category_PAYMENT"])
	t.Logf("PRICING: %d", stats["category_PRICING"])
	t.Logf("RISK: %d", stats["category_RISK"])
	t.Logf("FINANCIAL: %d", stats["category_FINANCIAL"])

	// Ensure we have reasonable coverage
	if stats["total"] < 10 {
		t.Errorf("Expected at least 10 invariants, got %d", stats["total"])
	}

	if stats["severity_CRITICAL"] < 5 {
		t.Errorf("Expected at least 5 CRITICAL invariants, got %d", stats["severity_CRITICAL"])
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// HELPER FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

func formatPercent(val float64) string {
	return fmt.Sprintf("%.0f%%", val*100)
}
