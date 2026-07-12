//go:build manual

// ═══════════════════════════════════════════════════════════════════════════
// E2E DEMO TEST - Automated Testing for Complete Demo Flow
//
// TESTS:
//   1. Full pipeline: Customer → Prediction → Quotation → Decision
//   2. All 4 geometry pipelines (Oblate, Truncated, S³, Banach)
//   3. Real data processing (if available)
//   4. Performance benchmarks
//
// Built with MATHEMATICAL RIGOR × PRODUCTION EXCELLENCE × DEMO READY 🕉️⚡💎
// Day 193 - E2E Integration Mission
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// TEST 1: Full Pipeline E2E
// ═══════════════════════════════════════════════════════════════════════════

func TestE2E_FullPipeline(t *testing.T) {
	t.Log("Testing complete E2E pipeline: Customer → Prediction → Quotation → Decision")

	// Create test customer
	customer := Customer{
		ID:             "TEST-001",
		BusinessName:   "Test Corp",
		OrderValue:     10000.0,
		OrderHistory:   []float64{8000, 9000, 10500},
		PaymentHistory: []int{42, 45, 40},
		RelationYears:  5,
		Industry:       "Manufacturing",
		Country:        "Bahrain",
		IsEmergency:    0,
		HasABB:         0,
		DisputeCount:   0,
	}

	// Step 1: Payment prediction
	predictor := NewPaymentPredictor(&customer)
	if predictor == nil {
		t.Fatal("Failed to create payment predictor")
	}

	prediction := predictor.Predict(&customer)

	// Validate prediction
	if prediction.Grade == "" {
		t.Error("Prediction grade is empty")
	}
	if prediction.PredictedDays < 15 || prediction.PredictedDays > 180 {
		t.Errorf("Predicted days %d out of valid range [15, 180]", prediction.PredictedDays)
	}
	if prediction.Confidence < 0.0 || prediction.Confidence > 1.0 {
		t.Errorf("Confidence %.2f out of valid range [0.0, 1.0]", prediction.Confidence)
	}

	// Validate three-regime distribution
	regimeSum := prediction.ThreeRegimes.R1 + prediction.ThreeRegimes.R2 + prediction.ThreeRegimes.R3
	if regimeSum < 0.99 || regimeSum > 1.01 {
		t.Errorf("Three-regime sum %.2f should be ~1.0", regimeSum)
	}

	t.Logf("✓ Prediction: Grade=%s, Days=%d, Confidence=%.0f%%",
		prediction.Grade, prediction.PredictedDays, prediction.Confidence*100)

	// Step 2: Costing engine
	costingEngine := NewCostingEngine()
	if costingEngine == nil {
		t.Fatal("Failed to create costing engine")
	}

	// Create mock basket
	basket := &ParsedEHBasket{
		CustomerNumber: customer.ID,
		CustomerName:   customer.BusinessName,
		Items: []ParsedEHItem{
			{
				OrderCode:         "RH-2025-001",
				Description:       "Flow Meter",
				Quantity:          2,
				ProductType:       "Rhine Flow",
				UnitSalesPriceBHD: 2500.0,
				ItemSalesPriceBHD: 5000.0,
				ProductionDays:    14,
			},
			{
				OrderCode:         "RH-2025-002",
				Description:       "Level Transmitter",
				Quantity:          3,
				ProductType:       "Rhine Level",
				UnitSalesPriceBHD: 1666.67,
				ItemSalesPriceBHD: 5000.0,
				ProductionDays:    10,
			},
		},
		TotalNetBHD:   10000.0,
		TotalGrossBHD: 10000.0,
		ItemCount:     2,
		SourceFile:    "test_basket.xml",
	}

	sheet, err := costingEngine.GenerateCostingSheet(basket, &customer)
	if err != nil {
		t.Fatalf("Failed to generate costing sheet: %v", err)
	}

	// Validate costing sheet
	if sheet.CustomerGrade == "" {
		t.Error("Costing sheet grade is empty")
	}
	if sheet.TotalFinalBHD <= 0 {
		t.Error("Total final amount should be positive")
	}
	if sheet.ActualMarginPct < 0 || sheet.ActualMarginPct > 1.0 {
		t.Errorf("Actual margin %.2f%% out of reasonable range", sheet.ActualMarginPct*100)
	}

	t.Logf("✓ Quotation: Total=%.2f BHD, Profit=%.2f BHD, Margin=%.1f%%",
		sheet.TotalFinalBHD, sheet.TotalProfitBHD, sheet.ActualMarginPct*100)

	// Step 3: Decision validation
	if sheet.ApprovalStatus == "" {
		t.Error("Approval status is empty")
	}
	if sheet.RecommendedAction == "" {
		t.Error("Recommended action is empty")
	}

	t.Logf("✓ Decision: %s - %s", sheet.ApprovalStatus, sheet.RecommendedAction)
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST 2: All Grade Classifications
// ═══════════════════════════════════════════════════════════════════════════

func TestE2E_AllGrades(t *testing.T) {
	t.Log("Testing all grade classifications (A, B, C, D)")

	testCases := []struct {
		name           string
		paymentHistory []int
		relationYears  int
		disputes       int
		expectedGrade  string
	}{
		{
			name:           "Grade A Customer",
			paymentHistory: []int{30, 32, 28, 35, 33},
			relationYears:  10,
			disputes:       0,
			expectedGrade:  "A",
		},
		{
			name:           "Grade B Customer",
			paymentHistory: []int{50, 48, 52, 45, 55},
			relationYears:  5,
			disputes:       1,
			expectedGrade:  "B",
		},
		{
			name:           "Grade C Customer",
			paymentHistory: []int{70, 65, 72, 68, 75},
			relationYears:  2,
			disputes:       2,
			expectedGrade:  "C",
		},
		{
			name:           "Grade D Customer",
			paymentHistory: []int{95, 100, 92, 105, 98},
			relationYears:  1,
			disputes:       4,
			expectedGrade:  "D",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			customer := Customer{
				ID:             "TEST-" + tc.name,
				BusinessName:   tc.name,
				OrderValue:     10000.0,
				OrderHistory:   []float64{9000, 10000, 11000},
				PaymentHistory: tc.paymentHistory,
				RelationYears:  tc.relationYears,
				Industry:       "Test",
				Country:        "Bahrain",
				DisputeCount:   tc.disputes,
			}

			predictor := NewPaymentPredictor(&customer)
			prediction := predictor.Predict(&customer)

			if prediction.Grade != tc.expectedGrade {
				t.Errorf("Expected grade %s, got %s", tc.expectedGrade, prediction.Grade)
			}

			t.Logf("✓ %s: Grade=%s, Days=%d, R3=%.0f%%",
				tc.name, prediction.Grade, prediction.PredictedDays, prediction.ThreeRegimes.R3*100)
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST 3: Geometry Pipelines
// ═══════════════════════════════════════════════════════════════════════════

func TestE2E_GeometryPipelines(t *testing.T) {
	t.Log("Testing all 4 geometry pipelines")

	bridge := NewGeometryBridge()
	if bridge == nil {
		t.Fatal("Failed to create geometry bridge")
	}

	// Test 1: Invoice Processing (Oblate Spheroid)
	t.Run("Oblate Spheroid Pipeline", func(t *testing.T) {
		invoice := InvoiceGeometry{
			ID:         "INV-001",
			CustomerID: "CUST-001",
			Amount:     10000.0,
			IssueDate:  time.Now().AddDate(0, 0, -30),
			DueDate:    time.Now().AddDate(0, 0, 30),
			Status:     "pending",
			Currency:   "BHD",
		}

		result, err := bridge.ProcessInvoice(invoice)
		if err != nil {
			t.Errorf("Invoice processing failed: %v", err)
		}

		if result == nil {
			t.Fatal("Invoice result is nil")
		}

		if result.PredictedDays <= 0 {
			t.Error("Predicted days should be positive")
		}

		t.Logf("✓ Invoice processed: Days=%d, Confidence=%.0f%%",
			result.PredictedDays, result.Confidence*100)
	})

	// Test 2: Tender Processing (Truncated Icosahedron)
	t.Run("Truncated Icosahedron Pipeline", func(t *testing.T) {
		tender := TenderGeometry{
			ID:         "TEN-001",
			CustomerID: "CUST-001",
			Items: []TenderItem{
				{
					ProductID:   "PROD-001",
					ProductName: "Flow Meter",
					Quantity:    2,
					UnitPrice:   2500.0,
					Margin:      0.15,
				},
			},
			Budget:       6000.0,
			Deadline:     time.Now().AddDate(0, 0, 14),
			RequiredDate: time.Now().AddDate(0, 0, 30),
		}

		result, err := bridge.ProcessTender(tender)
		if err != nil {
			t.Errorf("Tender processing failed: %v", err)
		}

		if result == nil {
			t.Fatal("Tender result is nil")
		}

		if result.OptimalQuote <= 0 {
			t.Error("Optimal quote should be positive")
		}

		t.Logf("✓ Tender processed: Quote=%.2f BHD, Margin=%.1f%%, Feasible=%v",
			result.OptimalQuote, result.Margin, result.Feasible)
	})

	// Test 3: Customer 360 (Quaternionic S³)
	t.Run("Quaternionic S³ Pipeline", func(t *testing.T) {
		customer := Customer{
			ID:             "CUST-001",
			BusinessName:   "Test Customer",
			OrderValue:     10000.0,
			OrderHistory:   []float64{9000, 10000, 11000},
			PaymentHistory: []int{42, 45, 40},
			RelationYears:  5,
			Industry:       "Manufacturing",
			Country:        "Bahrain",
		}

		result, err := bridge.GetCustomer360("CUST-001", &customer)
		if err != nil {
			t.Errorf("Customer 360 failed: %v", err)
		}

		if result == nil {
			t.Fatal("Customer 360 result is nil")
		}

		if result.Grade == "" {
			t.Error("Grade should not be empty")
		}

		t.Logf("✓ Customer 360: Grade=%s, Total Value=%.0f BHD, Avg Days=%.0f",
			result.Grade, result.TotalValue, result.AvgPaymentDays)
	})

	// Test 4: Compliance Check (Banach Ball)
	t.Run("Banach Ball Pipeline", func(t *testing.T) {
		compliance := ComplianceData{
			Type: "contract",
			Data: map[string]any{
				"value":     10000.0,
				"terms":     90,
				"margin":    0.15,
				"advance":   0.0,
				"compliant": true,
			},
			RuleSet:   "bahrain",
			Threshold: 0.01,
		}

		result, err := bridge.CheckCompliance(compliance)
		if err != nil {
			t.Errorf("Compliance check failed: %v", err)
		}

		if result == nil {
			t.Fatal("Compliance result is nil")
		}

		if result.Score < 0 || result.Score > 1 {
			t.Errorf("Compliance score %.2f out of range [0, 1]", result.Score)
		}

		t.Logf("✓ Compliance: Score=%.0f%%, Compliant=%v, Convergence=%v",
			result.Score*100, result.Compliant, result.Convergence)
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST 4: Portfolio Analytics
// ═══════════════════════════════════════════════════════════════════════════

func TestE2E_PortfolioAnalytics(t *testing.T) {
	t.Log("Testing portfolio analytics")

	demo := NewUnifiedDemo()
	demo.Customers = demo.GenerateSyntheticCustomers(20)

	if len(demo.Customers) != 20 {
		t.Errorf("Expected 20 customers, got %d", len(demo.Customers))
	}

	demo.RunPredictions()

	if len(demo.Predictions) != 20 {
		t.Errorf("Expected 20 predictions, got %d", len(demo.Predictions))
	}

	// Validate portfolio metrics
	if demo.Portfolio.TotalCustomers != 20 {
		t.Errorf("Expected 20 total customers, got %d", demo.Portfolio.TotalCustomers)
	}

	if demo.Portfolio.TotalPortfolio <= 0 {
		t.Error("Total portfolio should be positive")
	}

	if demo.Portfolio.RiskScore < 0 || demo.Portfolio.RiskScore > 1 {
		t.Errorf("Risk score %.2f out of range [0, 1]", demo.Portfolio.RiskScore)
	}

	// Check grade distribution
	totalGraded := demo.Portfolio.GradeDistribution["A"] +
		demo.Portfolio.GradeDistribution["B"] +
		demo.Portfolio.GradeDistribution["C"] +
		demo.Portfolio.GradeDistribution["D"]

	if totalGraded != 20 {
		t.Errorf("Grade distribution sum %d != 20", totalGraded)
	}

	t.Logf("✓ Portfolio: Total=%.0f BHD, Risk=%.0f%%, Grades: A:%d B:%d C:%d D:%d",
		demo.Portfolio.TotalPortfolio,
		demo.Portfolio.RiskScore*100,
		demo.Portfolio.GradeDistribution["A"],
		demo.Portfolio.GradeDistribution["B"],
		demo.Portfolio.GradeDistribution["C"],
		demo.Portfolio.GradeDistribution["D"])
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST 5: Performance Benchmarks
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkE2E_Prediction(b *testing.B) {
	customer := Customer{
		ID:             "BENCH-001",
		BusinessName:   "Benchmark Corp",
		OrderValue:     10000.0,
		OrderHistory:   []float64{9000, 10000, 11000},
		PaymentHistory: []int{42, 45, 40},
		RelationYears:  5,
		Industry:       "Manufacturing",
		Country:        "Bahrain",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		predictor := NewPaymentPredictor(&customer)
		_ = predictor.Predict(&customer)
	}
}

func BenchmarkE2E_CostingSheet(b *testing.B) {
	customer := Customer{
		ID:             "BENCH-001",
		BusinessName:   "Benchmark Corp",
		OrderValue:     10000.0,
		OrderHistory:   []float64{9000, 10000, 11000},
		PaymentHistory: []int{42, 45, 40},
		RelationYears:  5,
		Industry:       "Manufacturing",
		Country:        "Bahrain",
	}

	basket := &ParsedEHBasket{
		CustomerNumber: customer.ID,
		CustomerName:   customer.BusinessName,
		Items: []ParsedEHItem{
			{
				OrderCode:         "RH-2025-001",
				Description:       "Flow Meter",
				Quantity:          2,
				ProductType:       "Rhine Flow",
				UnitSalesPriceBHD: 2500.0,
				ItemSalesPriceBHD: 5000.0,
			},
		},
		TotalNetBHD:   10000.0,
		TotalGrossBHD: 10000.0,
		ItemCount:     1,
	}

	costingEngine := NewCostingEngine()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = costingEngine.GenerateCostingSheet(basket, &customer)
	}
}

func BenchmarkE2E_PortfolioAnalysis(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		demo := NewUnifiedDemo()
		demo.Customers = demo.GenerateSyntheticCustomers(50)
		demo.RunPredictions()
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TEST 6: Scenario Execution
// ═══════════════════════════════════════════════════════════════════════════

func TestE2E_AllScenarios(t *testing.T) {
	t.Log("Testing all business scenarios")

	runner := NewScenarioRunner()

	scenarios := []*DemoScenario{
		runner.Scenario1_NewRFQ(),
		runner.Scenario2_GradeDCustomer(),
		runner.Scenario3_ABBCompetition(),
		runner.Scenario4_Customer360(),
		runner.Scenario5_EmergencyOrder(),
	}

	for i, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := scenario.Execute(scenario)
			if err != nil {
				t.Errorf("Scenario %d failed: %v", i+1, err)
			}
			t.Logf("✓ %s completed", scenario.Name)
		})
	}
}
