// ═══════════════════════════════════════════════════════════════════════════
// BUSINESS_INVARIANTS - Acme Instrumentation Business Rules as Executable Assertions
//
// Extracted from:
//   - PH_TRADING_BUSINESS_REALITY_DOC.md
//   - PH_VISION_SSOT.md
//   - Codebase implementations (customer.go, costing_engine.go, predictor.go)
//
// PURPOSE:
//   These invariants serve as:
//   1. Validation rules for production data
//   2. Test assertions for correctness
//   3. Documentation of business constraints
//   4. Auditable business logic
//
// Built with LOVE × SIMPLICITY × TRUTH × JOY 🕉️💎⚡
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"fmt"
	"math"
)

// ═══════════════════════════════════════════════════════════════════════════
// CORE TYPES
// ═══════════════════════════════════════════════════════════════════════════

// BusinessInvariant represents a single business rule that can be validated
type BusinessInvariant struct {
	Name      string              // Human-readable name
	Category  string              // GRADING | PRICING | PAYMENT | RISK | FINANCIAL
	Rule      string              // Natural language description
	Assertion func(Context) error // Executable validation
	Severity  Severity            // CRITICAL | WARNING | INFO
	Source    string              // Which document/file defines this rule
}

// Severity indicates importance of rule violation
type Severity string

const (
	SeverityCritical Severity = "CRITICAL" // Must never be violated
	SeverityWarning  Severity = "WARNING"  // Should not be violated
	SeverityInfo     Severity = "INFO"     // Nice to have
)

// Context provides data for validation
type Context struct {
	Customer     *Customer
	Prediction   *PaymentPrediction
	CostingSheet *CostingSheet
	OrderValue   float64
	ActualMargin float64
}

// ═══════════════════════════════════════════════════════════════════════════
// CUSTOMER GRADING INVARIANTS
// ═══════════════════════════════════════════════════════════════════════════

// GradeA_R3_Threshold - Grade A customers MUST have R3 ≥ 50%
var GradeA_R3_Threshold = BusinessInvariant{
	Name:     "GradeA_R3_Threshold",
	Category: "GRADING",
	Rule:     "Customer grade A must have R3 (stability regime) ≥ 50%",
	Assertion: func(ctx Context) error {
		if ctx.Prediction == nil {
			return nil // No prediction to validate
		}
		if ctx.Prediction.Grade == "A" && ctx.Prediction.ThreeRegimes.R3 < 0.50 {
			return fmt.Errorf("Grade A customer %s has R3=%.1f%% (expected ≥50%%)",
				ctx.Prediction.CustomerID, ctx.Prediction.ThreeRegimes.R3*100)
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "predictor.go:258, PH_TRADING_BUSINESS_REALITY_DOC.md:112",
}

// GradeB_R3_Threshold - Grade B customers MUST have R3 ≥ 35%
var GradeB_R3_Threshold = BusinessInvariant{
	Name:     "GradeB_R3_Threshold",
	Category: "GRADING",
	Rule:     "Customer grade B must have R3 (stability regime) ≥ 35%",
	Assertion: func(ctx Context) error {
		if ctx.Prediction == nil {
			return nil
		}
		if ctx.Prediction.Grade == "B" && ctx.Prediction.ThreeRegimes.R3 < 0.35 {
			return fmt.Errorf("Grade B customer %s has R3=%.1f%% (expected ≥35%%)",
				ctx.Prediction.CustomerID, ctx.Prediction.ThreeRegimes.R3*100)
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "predictor.go:264, PH_TRADING_BUSINESS_REALITY_DOC.md:113",
}

// GradeC_R3_Threshold - Grade C customers MUST have R3 ≥ 20%
var GradeC_R3_Threshold = BusinessInvariant{
	Name:     "GradeC_R3_Threshold",
	Category: "GRADING",
	Rule:     "Customer grade C must have R3 (stability regime) ≥ 20%",
	Assertion: func(ctx Context) error {
		if ctx.Prediction == nil {
			return nil
		}
		if ctx.Prediction.Grade == "C" && ctx.Prediction.ThreeRegimes.R3 < 0.20 {
			return fmt.Errorf("Grade C customer %s has R3=%.1f%% (expected ≥20%%)",
				ctx.Prediction.CustomerID, ctx.Prediction.ThreeRegimes.R3*100)
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "predictor.go:269, PH_TRADING_BUSINESS_REALITY_DOC.md:114",
}

// GradeD_R3_Threshold - Grade D customers MUST have R3 < 20%
var GradeD_R3_Threshold = BusinessInvariant{
	Name:     "GradeD_R3_Threshold",
	Category: "GRADING",
	Rule:     "Customer grade D must have R3 (stability regime) < 20%",
	Assertion: func(ctx Context) error {
		if ctx.Prediction == nil {
			return nil
		}
		if ctx.Prediction.Grade == "D" && ctx.Prediction.ThreeRegimes.R3 >= 0.20 {
			return fmt.Errorf("Grade D customer %s has R3=%.1f%% (expected <20%%)",
				ctx.Prediction.CustomerID, ctx.Prediction.ThreeRegimes.R3*100)
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "predictor.go:274, PH_TRADING_BUSINESS_REALITY_DOC.md:115",
}

// ThreeRegimes_Sum_Unity - R1 + R2 + R3 MUST equal 100%
var ThreeRegimes_Sum_Unity = BusinessInvariant{
	Name:     "ThreeRegimes_Sum_Unity",
	Category: "GRADING",
	Rule:     "R1 + R2 + R3 must equal 1.0 (100%)",
	Assertion: func(ctx Context) error {
		if ctx.Prediction == nil {
			return nil
		}
		sum := ctx.Prediction.ThreeRegimes.R1 +
			ctx.Prediction.ThreeRegimes.R2 +
			ctx.Prediction.ThreeRegimes.R3
		tolerance := 0.01 // Allow 1% tolerance for floating-point errors
		if math.Abs(sum-1.0) > tolerance {
			return fmt.Errorf("Three regimes sum to %.3f (expected 1.0 ± %.3f)",
				sum, tolerance)
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "predictor.go:105-225 (three-regime calculation)",
}

// ═══════════════════════════════════════════════════════════════════════════
// PAYMENT TERM INVARIANTS
// ═══════════════════════════════════════════════════════════════════════════

// GradeA_PaymentTerms - Grade A customers get Net 45 days, no advance
var GradeA_PaymentTerms = BusinessInvariant{
	Name:     "GradeA_PaymentTerms",
	Category: "PAYMENT",
	Rule:     "Grade A customers: Net 45 days, 0% advance, predicted payment ≤ 55 days",
	Assertion: func(ctx Context) error {
		if ctx.Prediction == nil || ctx.Prediction.Grade != "A" {
			return nil
		}
		if ctx.Prediction.PredictedDays > activeOverlay.BusinessRules.GradePaymentTerms["A"].MaxDays {
			return fmt.Errorf("Grade A customer %s predicted %d days (expected ≤55)",
				ctx.Prediction.CustomerID, ctx.Prediction.PredictedDays)
		}
		if ctx.CostingSheet != nil && ctx.CostingSheet.AdvanceRequired > 0 {
			return fmt.Errorf("Grade A customer %s requires %.0f%% advance (expected 0%%)",
				ctx.Customer.ID, ctx.CostingSheet.AdvanceRequired*100)
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "costing_engine.go:226, predictor.go:260, PH_TRADING_BUSINESS_REALITY_DOC.md:112",
}

// GradeB_PaymentTerms - Grade B customers get Net 90 days, no advance
var GradeB_PaymentTerms = BusinessInvariant{
	Name:     "GradeB_PaymentTerms",
	Category: "PAYMENT",
	Rule:     "Grade B customers: Net 90 days, 0% advance, predicted payment ≤ 100 days",
	Assertion: func(ctx Context) error {
		if ctx.Prediction == nil || ctx.Prediction.Grade != "B" {
			return nil
		}
		if ctx.Prediction.PredictedDays > activeOverlay.BusinessRules.GradePaymentTerms["B"].MaxDays {
			return fmt.Errorf("Grade B customer %s predicted %d days (expected ≤100)",
				ctx.Prediction.CustomerID, ctx.Prediction.PredictedDays)
		}
		if ctx.CostingSheet != nil && ctx.CostingSheet.AdvanceRequired > 0 {
			return fmt.Errorf("Grade B customer %s requires %.0f%% advance (expected 0%%)",
				ctx.Customer.ID, ctx.CostingSheet.AdvanceRequired*100)
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "costing_engine.go:228, predictor.go:265, PH_TRADING_BUSINESS_REALITY_DOC.md:113",
}

// GradeC_PaymentTerms - Grade C customers get Net 120 days, 50% advance REQUIRED
var GradeC_PaymentTerms = BusinessInvariant{
	Name:     "GradeC_PaymentTerms",
	Category: "PAYMENT",
	Rule:     "Grade C customers: Net 120 days, 50% advance REQUIRED, predicted payment ≤ 130 days",
	Assertion: func(ctx Context) error {
		if ctx.Prediction == nil || ctx.Prediction.Grade != "C" {
			return nil
		}
		if ctx.Prediction.PredictedDays > activeOverlay.BusinessRules.GradePaymentTerms["C"].MaxDays {
			return fmt.Errorf("Grade C customer %s predicted %d days (expected ≤130)",
				ctx.Prediction.CustomerID, ctx.Prediction.PredictedDays)
		}
		if ctx.CostingSheet != nil && ctx.CostingSheet.AdvanceRequired < activeOverlay.BusinessRules.GradePaymentTerms["C"].AdvancePct {
			return fmt.Errorf("Grade C customer %s requires only %.0f%% advance (expected ≥50%%)",
				ctx.Customer.ID, ctx.CostingSheet.AdvanceRequired*100)
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "costing_engine.go:230, predictor.go:270, PH_TRADING_BUSINESS_REALITY_DOC.md:114",
}

// GradeD_PaymentTerms - Grade D customers MUST pay 100% advance OR be declined
var GradeD_PaymentTerms = BusinessInvariant{
	Name:     "GradeD_PaymentTerms",
	Category: "PAYMENT",
	Rule:     "Grade D customers: 100% advance REQUIRED or DECLINE order",
	Assertion: func(ctx Context) error {
		if ctx.Prediction == nil || ctx.Prediction.Grade != "D" {
			return nil
		}
		if ctx.CostingSheet != nil {
			if ctx.CostingSheet.ApprovalStatus == "✓ APPROVE" &&
				ctx.CostingSheet.AdvanceRequired < activeOverlay.BusinessRules.GradePaymentTerms["D"].AdvancePct {
				return fmt.Errorf("Grade D customer %s APPROVED with only %.0f%% advance (expected 100%% or DECLINE)",
					ctx.Customer.ID, ctx.CostingSheet.AdvanceRequired*100)
			}
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "costing_engine.go:232, predictor.go:274, PH_TRADING_BUSINESS_REALITY_DOC.md:115",
}

// ═══════════════════════════════════════════════════════════════════════════
// DISCOUNT INVARIANTS
// ═══════════════════════════════════════════════════════════════════════════

// GradeA_MaxDiscount - Grade A customers can receive up to 7% discount
var GradeA_MaxDiscount = BusinessInvariant{
	Name:     "GradeA_MaxDiscount",
	Category: "PRICING",
	Rule:     "Grade A customers: Maximum 7% discount allowed",
	Assertion: func(ctx Context) error {
		if ctx.CostingSheet == nil || ctx.CostingSheet.CustomerGrade != "A" {
			return nil
		}
		for _, item := range ctx.CostingSheet.Items {
			if item.CustomerDiscount > activeOverlay.CustomerDiscount("A") {
				return fmt.Errorf("Grade A customer %s item %s has %.1f%% discount (max 7%%)",
					ctx.Customer.ID, item.OrderCode, item.CustomerDiscount*100)
			}
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "costing_engine.go:209, PH_TRADING_BUSINESS_REALITY_DOC.md:158",
}

// GradeB_MaxDiscount - Grade B customers can receive up to 3% discount
var GradeB_MaxDiscount = BusinessInvariant{
	Name:     "GradeB_MaxDiscount",
	Category: "PRICING",
	Rule:     "Grade B customers: Maximum 3% discount allowed",
	Assertion: func(ctx Context) error {
		if ctx.CostingSheet == nil || ctx.CostingSheet.CustomerGrade != "B" {
			return nil
		}
		for _, item := range ctx.CostingSheet.Items {
			if item.CustomerDiscount > activeOverlay.CustomerDiscount("B") {
				return fmt.Errorf("Grade B customer %s item %s has %.1f%% discount (max 3%%)",
					ctx.Customer.ID, item.OrderCode, item.CustomerDiscount*100)
			}
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "costing_engine.go:211, PH_TRADING_BUSINESS_REALITY_DOC.md:159",
}

// GradeC_NoDiscount - Grade C customers CANNOT receive discounts
var GradeC_NoDiscount = BusinessInvariant{
	Name:     "GradeC_NoDiscount",
	Category: "PRICING",
	Rule:     "Grade C customers: NO discount allowed (0%)",
	Assertion: func(ctx Context) error {
		if ctx.CostingSheet == nil || ctx.CostingSheet.CustomerGrade != "C" {
			return nil
		}
		for _, item := range ctx.CostingSheet.Items {
			if item.CustomerDiscount > activeOverlay.CustomerDiscount("C") {
				return fmt.Errorf("Grade C customer %s item %s has %.1f%% discount (expected 0%%)",
					ctx.Customer.ID, item.OrderCode, item.CustomerDiscount*100)
			}
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "costing_engine.go:213, PH_TRADING_BUSINESS_REALITY_DOC.md:163",
}

// GradeD_NoDiscount - Grade D customers CANNOT receive discounts
var GradeD_NoDiscount = BusinessInvariant{
	Name:     "GradeD_NoDiscount",
	Category: "PRICING",
	Rule:     "Grade D customers: NO discount allowed (0%)",
	Assertion: func(ctx Context) error {
		if ctx.CostingSheet == nil || ctx.CostingSheet.CustomerGrade != "D" {
			return nil
		}
		for _, item := range ctx.CostingSheet.Items {
			if item.CustomerDiscount > activeOverlay.CustomerDiscount("D") {
				return fmt.Errorf("Grade D customer %s item %s has %.1f%% discount (expected 0%%)",
					ctx.Customer.ID, item.OrderCode, item.CustomerDiscount*100)
			}
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "costing_engine.go:215, PH_TRADING_BUSINESS_REALITY_DOC.md:167",
}

// ═══════════════════════════════════════════════════════════════════════════
// PRODUCT MARGIN INVARIANTS
// ═══════════════════════════════════════════════════════════════════════════

// RhineFlow_Margin - Rhine Instruments flow meters MUST have 15% standard margin
var RhineFlow_Margin = BusinessInvariant{
	Name:     "RhineFlow_Margin",
	Category: "PRICING",
	Rule:     "Rhine Instruments flow meters: 15% standard margin",
	Assertion: func(ctx Context) error {
		if ctx.CostingSheet == nil {
			return nil
		}
		for _, item := range ctx.CostingSheet.Items {
			if item.ProductType == "Rhine Flow" && math.Abs(item.StandardMargin-activeOverlay.ProductMargin("Rhine Flow")) > 0.001 {
				return fmt.Errorf("Rhine Flow item %s has %.1f%% margin (expected 15%%)",
					item.OrderCode, item.StandardMargin*100)
			}
		}
		return nil
	},
	Severity: SeverityWarning,
	Source:   "costing_engine.go:12, PH_TRADING_BUSINESS_REALITY_DOC.md:147",
}

// GasAnalyzer_Margin - gas analyzers MUST have 25% standard margin
var GasAnalyzer_Margin = BusinessInvariant{
	Name:     "GasAnalyzer_Margin",
	Category: "PRICING",
	Rule:     "gas analyzers: 25% standard margin (specialty premium)",
	Assertion: func(ctx Context) error {
		if ctx.CostingSheet == nil {
			return nil
		}
		for _, item := range ctx.CostingSheet.Items {
			if item.ProductType == "Oxan Analytics" && math.Abs(item.StandardMargin-activeOverlay.ProductMargin("Oxan Analytics")) > 0.001 {
				return fmt.Errorf("Oxan Analytics item %s has %.1f%% margin (expected 25%%)",
					item.OrderCode, item.StandardMargin*100)
			}
		}
		return nil
	},
	Severity: SeverityWarning,
	Source:   "costing_engine.go:18, PH_TRADING_BUSINESS_REALITY_DOC.md:151",
}

// GIC_Margin - GIC instruments MUST have 10% standard margin
var GIC_Margin = BusinessInvariant{
	Name:     "GIC_Margin",
	Category: "PRICING",
	Rule:     "GIC instruments: 10% standard margin (commodity pricing)",
	Assertion: func(ctx Context) error {
		if ctx.CostingSheet == nil {
			return nil
		}
		for _, item := range ctx.CostingSheet.Items {
			if item.ProductType == "GIC" && math.Abs(item.StandardMargin-activeOverlay.ProductMargin("GIC")) > 0.001 {
				return fmt.Errorf("GIC item %s has %.1f%% margin (expected 10%%)",
					item.OrderCode, item.StandardMargin*100)
			}
		}
		return nil
	},
	Severity: SeverityWarning,
	Source:   "costing_engine.go:19, PH_TRADING_BUSINESS_REALITY_DOC.md:152",
}

// ═══════════════════════════════════════════════════════════════════════════
// RISK & APPROVAL INVARIANTS
// ═══════════════════════════════════════════════════════════════════════════

// Minimum_Margin_Threshold - NEVER accept orders with margin < 8%
var Minimum_Margin_Threshold = BusinessInvariant{
	Name:     "Minimum_Margin_Threshold",
	Category: "RISK",
	Rule:     "Actual margin after discount MUST be ≥ 8% or order should be declined",
	Assertion: func(ctx Context) error {
		if ctx.CostingSheet == nil {
			return nil
		}
		if ctx.CostingSheet.ActualMarginPct < activeOverlay.BusinessRules.MinMarginPct &&
			ctx.CostingSheet.ApprovalStatus == "✓ APPROVE" {
			return fmt.Errorf("Order APPROVED with %.1f%% margin (minimum 8%% required)",
				ctx.CostingSheet.ActualMarginPct*100)
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "costing_engine.go:265, PH_TRADING_BUSINESS_REALITY_DOC.md:169",
}

// ABB_Competition_Margin - If ABB competing, margin MUST be ≥ 15% to proceed
var ABB_Competition_Margin = BusinessInvariant{
	Name:     "ABB_Competition_Margin",
	Category: "RISK",
	Rule:     "If ABB is competing, margin MUST be ≥ 15% or DECLINE order",
	Assertion: func(ctx Context) error {
		if ctx.Customer == nil || ctx.Customer.HasABB == 0 {
			return nil
		}
		if ctx.CostingSheet != nil &&
			ctx.CostingSheet.ActualMarginPct < activeOverlay.BusinessRules.ABBCompetitionMinMargin &&
			ctx.CostingSheet.ApprovalStatus != "✗ DECLINE" {
			return fmt.Errorf("ABB competing, margin %.1f%% < 15%%, but order not DECLINED",
				ctx.CostingSheet.ActualMarginPct*100)
		}
		return nil
	},
	Severity: SeverityCritical,
	Source:   "costing_engine.go:273-278, PH_TRADING_BUSINESS_REALITY_DOC.md:168-173",
}

// Emergency_Order_Premium - Emergency orders should have premium pricing
var Emergency_Order_Premium = BusinessInvariant{
	Name:     "Emergency_Order_Premium",
	Category: "PRICING",
	Rule:     "Emergency orders should carry premium pricing (+10-20%)",
	Assertion: func(ctx Context) error {
		if ctx.Customer == nil || ctx.Customer.IsEmergency == 0 {
			return nil
		}
		// INFO level - this is a recommendation, not a hard rule
		if ctx.CostingSheet != nil && ctx.CostingSheet.ActualMarginPct < activeOverlay.BusinessRules.EmergencyMinMarginPct {
			return fmt.Errorf("Emergency order has only %.1f%% margin (consider +10-20%% premium)",
				ctx.CostingSheet.ActualMarginPct*100)
		}
		return nil
	},
	Severity: SeverityInfo,
	Source:   "costing_engine.go:291-294, PH_TRADING_BUSINESS_REALITY_DOC.md:143",
}

// ═══════════════════════════════════════════════════════════════════════════
// FINANCIAL INVARIANTS
// ═══════════════════════════════════════════════════════════════════════════

// Monthly_Burn_Rate - Monthly costs MUST be tracked accurately
var Monthly_Burn_Rate = BusinessInvariant{
	Name:     "Monthly_Burn_Rate",
	Category: "FINANCIAL",
	Rule:     "Monthly costs documented as 15,000 BHD all-in",
	Assertion: func(ctx Context) error {
		// This is a constant, mainly for documentation
		// Would be used in actual financial reporting validation
		documentedMonthlyCosts := activeOverlay.BusinessRules.MonthlyOperatingCostBHD // BHD
		_ = documentedMonthlyCosts
		return nil
	},
	Severity: SeverityInfo,
	Source:   "PH_TRADING_BUSINESS_REALITY_DOC.md:11",
}

// Cash_Runway_Critical - Cash runway < 60 days is CRITICAL alert
var Cash_Runway_Critical = BusinessInvariant{
	Name:     "Cash_Runway_Critical",
	Category: "FINANCIAL",
	Rule:     "Cash runway < 60 days triggers CRITICAL alert",
	Assertion: func(ctx Context) error {
		// Would be implemented when cash position tracking is added
		// daysOfRunway := currentCash / (monthlyCosts - monthlyRevenue)
		// if daysOfRunway < 60 { CRITICAL }
		return nil
	},
	Severity: SeverityCritical,
	Source:   "PH_TRADING_BUSINESS_REALITY_DOC.md:258",
}

// MRR_Sustainability_Ratio - Monthly recurring revenue divided by costs
var MRR_Sustainability_Ratio = BusinessInvariant{
	Name:     "MRR_Sustainability_Ratio",
	Category: "FINANCIAL",
	Rule:     "MRR / Monthly Costs: Target 1.00 for break-even, current ~0.27",
	Assertion: func(ctx Context) error {
		// Would be implemented when MRR tracking is added
		// sustainabilityRatio := monthlyRecurringRevenue / monthlyCosts
		// Target Month 6: 1.00 (15K MRR / 15K costs)
		return nil
	},
	Severity: SeverityInfo,
	Source:   "PH_TRADING_BUSINESS_REALITY_DOC.md:348-352",
}

// ═══════════════════════════════════════════════════════════════════════════
// CUSTOMER DISTRIBUTION INVARIANTS
// ═══════════════════════════════════════════════════════════════════════════

// Customer_Grade_Distribution - Target 50% A/B grade customers
var Customer_Grade_Distribution = BusinessInvariant{
	Name:     "Customer_Grade_Distribution",
	Category: "RISK",
	Rule:     "Target: 50%+ of customers should be Grade A or B",
	Assertion: func(ctx Context) error {
		// Would be implemented when portfolio analysis is added
		// Would check: (countGradeA + countGradeB) / totalCustomers >= 0.50
		return nil
	},
	Severity: SeverityWarning,
	Source:   "PH_TRADING_BUSINESS_REALITY_DOC.md:344",
}

// ═══════════════════════════════════════════════════════════════════════════
// INVARIANT REGISTRY
// ═══════════════════════════════════════════════════════════════════════════

// AllInvariants is the complete registry of business rules
var AllInvariants = []BusinessInvariant{
	// Customer Grading (5)
	GradeA_R3_Threshold,
	GradeB_R3_Threshold,
	GradeC_R3_Threshold,
	GradeD_R3_Threshold,
	ThreeRegimes_Sum_Unity,

	// Payment Terms (4)
	GradeA_PaymentTerms,
	GradeB_PaymentTerms,
	GradeC_PaymentTerms,
	GradeD_PaymentTerms,

	// Discounts (4)
	GradeA_MaxDiscount,
	GradeB_MaxDiscount,
	GradeC_NoDiscount,
	GradeD_NoDiscount,

	// Product Margins (3)
	RhineFlow_Margin,
	GasAnalyzer_Margin,
	GIC_Margin,

	// Risk & Approval (3)
	Minimum_Margin_Threshold,
	ABB_Competition_Margin,
	Emergency_Order_Premium,

	// Financial (3)
	Monthly_Burn_Rate,
	Cash_Runway_Critical,
	MRR_Sustainability_Ratio,

	// Portfolio (1)
	Customer_Grade_Distribution,
}

// ═══════════════════════════════════════════════════════════════════════════
// VALIDATION FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

// InvariantValidationResult represents the outcome of validating invariants
type InvariantValidationResult struct {
	InvariantName string
	Category      string
	Severity      Severity
	Passed        bool
	Error         error
}

// ValidateAll runs all business invariants against provided context
func ValidateAll(ctx Context) []InvariantValidationResult {
	results := make([]InvariantValidationResult, 0, len(AllInvariants))

	for _, inv := range AllInvariants {
		err := inv.Assertion(ctx)
		results = append(results, InvariantValidationResult{
			InvariantName: inv.Name,
			Category:      inv.Category,
			Severity:      inv.Severity,
			Passed:        err == nil,
			Error:         err,
		})
	}

	return results
}

// ValidateCategory runs invariants for a specific category
func ValidateCategory(ctx Context, category string) []InvariantValidationResult {
	results := make([]InvariantValidationResult, 0)

	for _, inv := range AllInvariants {
		if inv.Category == category {
			err := inv.Assertion(ctx)
			results = append(results, InvariantValidationResult{
				InvariantName: inv.Name,
				Category:      inv.Category,
				Severity:      inv.Severity,
				Passed:        err == nil,
				Error:         err,
			})
		}
	}

	return results
}

// ValidateBySeverity runs only invariants of specified severity
func ValidateBySeverity(ctx Context, severity Severity) []InvariantValidationResult {
	results := make([]InvariantValidationResult, 0)

	for _, inv := range AllInvariants {
		if inv.Severity == severity {
			err := inv.Assertion(ctx)
			results = append(results, InvariantValidationResult{
				InvariantName: inv.Name,
				Category:      inv.Category,
				Severity:      inv.Severity,
				Passed:        err == nil,
				Error:         err,
			})
		}
	}

	return results
}

// PrintValidationResults displays results in human-readable format
func PrintValidationResults(results []InvariantValidationResult) {
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║ BUSINESS INVARIANTS VALIDATION REPORT                                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	critical := 0
	warning := 0
	info := 0
	passed := 0
	failed := 0

	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
			switch r.Severity {
			case SeverityCritical:
				critical++
			case SeverityWarning:
				warning++
			case SeverityInfo:
				info++
			}
		}
	}

	fmt.Printf("Total Invariants Checked: %d\n", len(results))
	fmt.Printf("✓ Passed:  %d\n", passed)
	fmt.Printf("✗ Failed:  %d\n", failed)
	if failed > 0 {
		fmt.Printf("  - CRITICAL: %d\n", critical)
		fmt.Printf("  - WARNING:  %d\n", warning)
		fmt.Printf("  - INFO:     %d\n", info)
	}
	fmt.Println()

	if failed > 0 {
		fmt.Println("FAILED INVARIANTS:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for _, r := range results {
			if !r.Passed {
				symbol := "⚠"
				if r.Severity == SeverityCritical {
					symbol = "✗"
				} else if r.Severity == SeverityInfo {
					symbol = "ℹ"
				}
				fmt.Printf("%s [%s] %s\n", symbol, r.Severity, r.InvariantName)
				fmt.Printf("  Category: %s\n", r.Category)
				fmt.Printf("  Error:    %s\n", r.Error)
				fmt.Println()
			}
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// STATISTICS & REPORTING
// ═══════════════════════════════════════════════════════════════════════════

// ValidateCostingApproval validates key business rules for costing/order approval
// without requiring the full Context struct. Returns nil if all checks pass.
func ValidateCostingApproval(customerGrade string, actualMarginPct float64, advanceRequired float64, hasABBCompetition bool) error {
	br := activeOverlay.BusinessRules
	// 8% minimum margin for approved orders
	if actualMarginPct < br.MinMarginPct {
		return fmt.Errorf("actual margin %.1f%% is below minimum 8%% threshold", actualMarginPct*100)
	}

	// Grade D requires 100% advance
	if customerGrade == "D" && advanceRequired < br.GradePaymentTerms["D"].AdvancePct {
		return fmt.Errorf("grade D customer requires 100%% advance (got %.0f%%)", advanceRequired*100)
	}

	// Grade C requires 50% advance
	if customerGrade == "C" && advanceRequired < br.GradePaymentTerms["C"].AdvancePct {
		return fmt.Errorf("grade C customer requires at least 50%% advance (got %.0f%%)", advanceRequired*100)
	}

	// ABB competition requires 15% margin
	if hasABBCompetition && actualMarginPct < br.ABBCompetitionMinMargin {
		return fmt.Errorf("ABB competing: margin %.1f%% is below 15%% threshold", actualMarginPct*100)
	}

	return nil
}

// InvariantStatistics provides overview of registered invariants
func InvariantStatistics() map[string]int {
	stats := make(map[string]int)
	stats["total"] = len(AllInvariants)

	for _, inv := range AllInvariants {
		stats["category_"+inv.Category]++
		stats["severity_"+string(inv.Severity)]++
	}

	return stats
}

// ListInvariantsByCategory returns all invariants grouped by category
func ListInvariantsByCategory() map[string][]BusinessInvariant {
	byCategory := make(map[string][]BusinessInvariant)

	for _, inv := range AllInvariants {
		byCategory[inv.Category] = append(byCategory[inv.Category], inv)
	}

	return byCategory
}

// PrintInvariantIndex displays all registered invariants
func PrintInvariantIndex() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║ BUSINESS INVARIANTS INDEX                                                 ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	byCategory := ListInvariantsByCategory()

	for category, invs := range byCategory {
		fmt.Printf("┌─ %s (%d rules)\n", category, len(invs))
		for _, inv := range invs {
			fmt.Printf("│  • %s [%s]\n", inv.Name, inv.Severity)
			fmt.Printf("│    %s\n", inv.Rule)
		}
		fmt.Println("└───")
		fmt.Println()
	}

	stats := InvariantStatistics()
	fmt.Printf("Total: %d invariants\n", stats["total"])
	fmt.Printf("  CRITICAL: %d\n", stats["severity_CRITICAL"])
	fmt.Printf("  WARNING:  %d\n", stats["severity_WARNING"])
	fmt.Printf("  INFO:     %d\n", stats["severity_INFO"])
}
