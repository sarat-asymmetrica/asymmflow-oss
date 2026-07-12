// ═══════════════════════════════════════════════════════════════════════════
// COSTING ENGINE - Quotation Generation with Margins and Risk Assessment
//
// MISSION: Calculate customer quotations with margins and risk adjustments
//
// FEATURES:
//   - Product-specific markup rules (Rhine Flow, Level, Pressure, etc.)
//   - Customer grade-based discounts (A/B/C/D)
//   - Payment terms based on risk assessment
//   - Profit analysis and margin calculations
//   - Risk warnings and approval recommendations
//
// Built with MATHEMATICAL RIGOR × PRODUCTION ROBUSTNESS
// ═══════════════════════════════════════════════════════════════════════════

package engines

import (
	"fmt"
	"math"
	"time"

	prediction "ph_holdings_app/pkg/butler/prediction"
	"ph_holdings_app/pkg/overlay"
)

// Product markup rules, the default product margin, grade discounts, and grade
// payment terms now live in the company overlay (pkg/overlay). They are read at
// runtime via overlay.Active() so a different vertical can ship different policy
// numbers via overlay.json without recompiling. The values are reproduced
// byte-identically in overlay.BuiltinDefaults().
// See pkg/overlay/business_rules.go.

// CustomerGrade represents customer payment risk classification
type CustomerGrade string

const (
	GradeA CustomerGrade = "A" // Best: R3 ≥ 50%, max 7% discount
	GradeB CustomerGrade = "B" // Good: R3 ≥ 35%, max 3% discount
	GradeC CustomerGrade = "C" // Caution: R3 ≥ 20%, no discount, 50% advance
	GradeD CustomerGrade = "D" // High risk: R3 < 20%, decline or 100% advance
)

// CostingEngine calculates customer quotations with margins and risk adjustments
type CostingEngine struct {
	Parser *EHParser
}

// NewCostingEngine creates a new costing engine
func NewCostingEngine() *CostingEngine {
	return &CostingEngine{
		Parser: NewEHParser(),
	}
}

// CostingItem represents a single line item with markup applied
type CostingItem struct {
	OrderCode        string  `json:"order_code"`
	Description      string  `json:"description"`
	Quantity         int     `json:"quantity"`
	ProductType      string  `json:"product_type"`
	ProductionDays   int     `json:"production_days"`
	UnitCostBHD      float64 `json:"unit_cost_bhd"`
	TotalCostBHD     float64 `json:"total_cost_bhd"`
	StandardMargin   float64 `json:"standard_margin"`
	UnitSellBHD      float64 `json:"unit_sell_bhd"`
	TotalSellBHD     float64 `json:"total_sell_bhd"`
	CustomerDiscount float64 `json:"customer_discount"`
	FinalUnitBHD     float64 `json:"final_unit_bhd"`
	FinalTotalBHD    float64 `json:"final_total_bhd"`
	UnitProfitBHD    float64 `json:"unit_profit_bhd"`
	TotalProfitBHD   float64 `json:"total_profit_bhd"`
	ActualMargin     float64 `json:"actual_margin"`
}

// CostingSheet represents complete quotation
type CostingSheet struct {
	CustomerID        string                       `json:"customer_id"`
	CustomerName      string                       `json:"customer_name"`
	CustomerGrade     CustomerGrade                `json:"customer_grade"`
	PaymentPrediction prediction.PaymentPrediction `json:"payment_prediction"`
	QuotationDate     string                       `json:"quotation_date"`
	ValidUntil        string                       `json:"valid_until"`
	SourceBasket      string                       `json:"source_basket"`
	Items             []CostingItem                `json:"items"`
	TotalCostBHD      float64                      `json:"total_cost_bhd"`
	TotalSellBHD      float64                      `json:"total_sell_bhd"`
	TotalDiscountBHD  float64                      `json:"total_discount_bhd"`
	TotalFinalBHD     float64                      `json:"total_final_bhd"`
	TotalProfitBHD    float64                      `json:"total_profit_bhd"`
	VATRate           float64                      `json:"vat_rate"`
	VATAmountBHD      float64                      `json:"vat_amount_bhd"`
	GrandTotalBHD     float64                      `json:"grand_total_bhd"`
	StandardMarginPct float64                      `json:"standard_margin_pct"`
	ActualMarginPct   float64                      `json:"actual_margin_pct"`
	PaymentTerms      string                       `json:"payment_terms"`
	AdvanceRequired   float64                      `json:"advance_required"`
	ApprovalStatus    string                       `json:"approval_status"`
	RiskWarnings      []string                     `json:"risk_warnings"`
	RecommendedAction string                       `json:"recommended_action"`
}

// SetVATRate sets the VAT rate for the costing sheet (call before GenerateCostingSheet)
func (sheet *CostingSheet) SetVATRate(rate float64) {
	sheet.VATRate = rate
}

// GetProductMargin returns margin percentage for product type (from overlay).
func (ce *CostingEngine) GetProductMargin(productType string) float64 {
	return overlay.Active().ProductMargin(productType)
}

// GetCustomerDiscount returns maximum allowed discount based on grade (from overlay).
func (ce *CostingEngine) GetCustomerDiscount(grade CustomerGrade) float64 {
	return overlay.Active().CustomerDiscount(string(grade))
}

// GetPaymentTerms returns payment terms and advance requirement (from overlay).
func (ce *CostingEngine) GetPaymentTerms(grade CustomerGrade) (terms string, advance float64) {
	return overlay.Active().PaymentTerms(string(grade))
}

// GenerateCostingFromBasket creates a costing sheet from Rhine basket
func (ce *CostingEngine) GenerateCostingFromBasket(basket *ParsedEHBasket, customerName string, grade CustomerGrade) *CostingSheet {
	sheet := &CostingSheet{
		CustomerName:  customerName,
		CustomerGrade: grade,
		QuotationDate: time.Now().UTC().Format("2006-01-02"),
		ValidUntil:    time.Now().UTC().AddDate(0, 0, 30).Format("2006-01-02"),
		SourceBasket:  basket.SourceFile,
		Items:         make([]CostingItem, 0, len(basket.Items)),
		RiskWarnings:  make([]string, 0),
	}

	customerDiscount := ce.GetCustomerDiscount(grade)
	sheet.PaymentTerms, sheet.AdvanceRequired = ce.GetPaymentTerms(grade)

	for _, item := range basket.Items {
		costingItem := CostingItem{
			OrderCode:      item.OrderCode,
			Description:    item.Description,
			Quantity:       item.Quantity,
			ProductType:    item.ProductType,
			ProductionDays: item.ProductionDays,
			UnitCostBHD:    item.UnitSalesPriceBHD,
			TotalCostBHD:   item.ItemSalesPriceBHD,
		}

		costingItem.StandardMargin = ce.GetProductMargin(item.ProductType)
		costingItem.UnitSellBHD = costingItem.UnitCostBHD * (1.0 + costingItem.StandardMargin)
		costingItem.TotalSellBHD = costingItem.UnitSellBHD * float64(costingItem.Quantity)
		costingItem.CustomerDiscount = customerDiscount
		costingItem.FinalUnitBHD = costingItem.UnitSellBHD * (1.0 - customerDiscount)
		costingItem.FinalTotalBHD = costingItem.FinalUnitBHD * float64(costingItem.Quantity)
		costingItem.UnitProfitBHD = costingItem.FinalUnitBHD - costingItem.UnitCostBHD
		costingItem.TotalProfitBHD = costingItem.UnitProfitBHD * float64(costingItem.Quantity)

		if costingItem.UnitCostBHD > 0 {
			costingItem.ActualMargin = costingItem.UnitProfitBHD / costingItem.UnitCostBHD
		}

		sheet.Items = append(sheet.Items, costingItem)
	}

	// Calculate totals
	for _, item := range sheet.Items {
		sheet.TotalCostBHD += item.TotalCostBHD
		sheet.TotalSellBHD += item.TotalSellBHD
		sheet.TotalFinalBHD += item.FinalTotalBHD
		sheet.TotalProfitBHD += item.TotalProfitBHD
	}

	sheet.TotalDiscountBHD = sheet.TotalSellBHD - sheet.TotalFinalBHD

	if sheet.TotalCostBHD > 0 {
		sheet.StandardMarginPct = (sheet.TotalSellBHD - sheet.TotalCostBHD) / sheet.TotalCostBHD
		sheet.ActualMarginPct = sheet.TotalProfitBHD / sheet.TotalCostBHD
	}

	// Risk assessment
	ce.AssessRisk(sheet)

	return sheet
}

// AssessRisk performs risk analysis and sets approval status
func (ce *CostingEngine) AssessRisk(sheet *CostingSheet) {
	switch sheet.CustomerGrade {
	case GradeA:
		sheet.ApprovalStatus = "✓ APPROVE"
		sheet.RecommendedAction = "Proceed with order. Reliable customer."
	case GradeB:
		sheet.ApprovalStatus = "✓ APPROVE"
		sheet.RecommendedAction = "Proceed with caution. Monitor payment."
		sheet.RiskWarnings = append(sheet.RiskWarnings, "Moderate risk - monitor payment")
	case GradeC:
		sheet.ApprovalStatus = "⚠ CAUTION"
		sheet.RecommendedAction = "Require 50% advance payment."
		sheet.RiskWarnings = append(sheet.RiskWarnings, "Higher risk - require 50% advance")
	case GradeD:
		sheet.ApprovalStatus = "✗ DECLINE"
		sheet.RecommendedAction = "Decline or require 100% advance."
		sheet.RiskWarnings = append(sheet.RiskWarnings, "HIGH RISK - decline or full advance")
	}

	if sheet.ActualMarginPct < overlay.Active().BusinessRules.MinMarginPct {
		sheet.RiskWarnings = append(sheet.RiskWarnings,
			fmt.Sprintf("Low margin (%.1f%%) - consider rejecting", sheet.ActualMarginPct*100))
	}
}

// CalculateROI calculates 3-year ROI for the costing engine
func (ce *CostingEngine) CalculateROI() map[string]any {
	quotationsPerMonth := 40.0
	manualMinutesPerQuote := 45.0
	automatedMinutesPerQuote := 3.0

	manualErrorRate := 0.12
	automatedErrorRate := 0.02
	avgQuoteValue := 3000.0
	avgMargin := 0.15

	manualMonthlyHours := (quotationsPerMonth * manualMinutesPerQuote) / 60.0
	automatedMonthlyHours := (quotationsPerMonth * automatedMinutesPerQuote) / 60.0
	savedHours := manualMonthlyHours - automatedMonthlyHours

	manualErrorCost := quotationsPerMonth * manualErrorRate * avgQuoteValue * avgMargin * 0.5
	automatedErrorCost := quotationsPerMonth * automatedErrorRate * avgQuoteValue * avgMargin * 0.5
	errorSavings := manualErrorCost - automatedErrorCost

	additionalCapacity := savedHours / (automatedMinutesPerQuote / 60.0)
	additionalRevenue := additionalCapacity * avgQuoteValue * avgMargin * 0.3

	monthlyBenefit := errorSavings + additionalRevenue
	yearlyBenefit := monthlyBenefit * 12
	threeYearBenefit := yearlyBenefit * 3
	devCost := 2000.0

	return map[string]any{
		"time_saved_hours_per_month":       math.Round(savedHours*10) / 10,
		"error_savings_bhd_per_month":      math.Round(errorSavings*100) / 100,
		"additional_revenue_bhd_per_month": math.Round(additionalRevenue*100) / 100,
		"total_monthly_benefit_bhd":        math.Round(monthlyBenefit*100) / 100,
		"yearly_benefit_bhd":               math.Round(yearlyBenefit*100) / 100,
		"three_year_benefit_bhd":           math.Round(threeYearBenefit*100) / 100,
		"development_cost_bhd":             devCost,
		"roi_percent":                      math.Round((threeYearBenefit/devCost-1.0)*10000) / 100,
		"payback_months":                   math.Round((devCost/monthlyBenefit)*10) / 10,
	}
}

// GenerateCostingSheet creates a complete quotation from Rhine basket using full prediction.Customer.
// This is the richer API that accepts a prediction.Customer (with int HasABB/IsEmergency flags)
// and performs full payment prediction, VAT calculation, and risk assessment.
func (ce *CostingEngine) GenerateCostingSheet(basket *ParsedEHBasket, customer *prediction.Customer) (*CostingSheet, error) {
	// Step 1: Predict customer payment behavior
	predictor := prediction.NewPaymentPredictor(customer)
	pred := predictor.Predict(customer)

	// Step 2: Initialize costing sheet
	sheet := &CostingSheet{
		CustomerID:        customer.ID,
		CustomerName:      customer.BusinessName,
		CustomerGrade:     CustomerGrade(pred.Grade),
		PaymentPrediction: pred,
		QuotationDate:     time.Now().UTC().Format("2006-01-02"),
		ValidUntil:        time.Now().UTC().AddDate(0, 0, 30).Format("2006-01-02"), // 30 days validity
		SourceBasket:      basket.SourceFile,
		Items:             make([]CostingItem, 0, len(basket.Items)),
		RiskWarnings:      make([]string, 0),
	}

	// Step 3: Determine customer discount and payment terms based on grade
	customerDiscount := ce.GetCustomerDiscount(CustomerGrade(pred.Grade))
	sheet.PaymentTerms, sheet.AdvanceRequired = ce.GetPaymentTerms(CustomerGrade(pred.Grade))

	// Step 4: Process each item
	for _, item := range basket.Items {
		costingItem := CostingItem{
			OrderCode:      item.OrderCode,
			Description:    item.Description,
			Quantity:       item.Quantity,
			ProductType:    item.ProductType,
			ProductionDays: item.ProductionDays,
			UnitCostBHD:    item.UnitSalesPriceBHD, // What PH pays Rhine Instruments
			TotalCostBHD:   item.ItemSalesPriceBHD,
		}

		// Get standard margin for this product type
		costingItem.StandardMargin = ce.GetProductMargin(item.ProductType)

		// Calculate sell price before customer discount
		costingItem.UnitSellBHD = costingItem.UnitCostBHD * (1.0 + costingItem.StandardMargin)
		costingItem.TotalSellBHD = costingItem.UnitSellBHD * float64(costingItem.Quantity)

		// Apply customer discount
		costingItem.CustomerDiscount = customerDiscount
		costingItem.FinalUnitBHD = costingItem.UnitSellBHD * (1.0 - customerDiscount)
		costingItem.FinalTotalBHD = costingItem.FinalUnitBHD * float64(costingItem.Quantity)

		// Calculate profit
		costingItem.UnitProfitBHD = costingItem.FinalUnitBHD - costingItem.UnitCostBHD
		costingItem.TotalProfitBHD = costingItem.UnitProfitBHD * float64(costingItem.Quantity)

		// Calculate actual margin
		if costingItem.UnitCostBHD > 0 {
			costingItem.ActualMargin = costingItem.UnitProfitBHD / costingItem.UnitCostBHD
		}

		sheet.Items = append(sheet.Items, costingItem)
	}

	// Step 5: Calculate totals
	sheet.TotalCostBHD = 0.0
	sheet.TotalSellBHD = 0.0
	sheet.TotalFinalBHD = 0.0
	sheet.TotalProfitBHD = 0.0

	for _, item := range sheet.Items {
		sheet.TotalCostBHD += item.TotalCostBHD
		sheet.TotalSellBHD += item.TotalSellBHD
		sheet.TotalFinalBHD += item.FinalTotalBHD
		sheet.TotalProfitBHD += item.TotalProfitBHD
	}

	sheet.TotalDiscountBHD = sheet.TotalSellBHD - sheet.TotalFinalBHD

	// Calculate margins
	if sheet.TotalCostBHD > 0 {
		sheet.StandardMarginPct = (sheet.TotalSellBHD - sheet.TotalCostBHD) / sheet.TotalCostBHD
		sheet.ActualMarginPct = sheet.TotalProfitBHD / sheet.TotalCostBHD
	}

	// Step 6: Calculate VAT (rate injected via SetVATRate, defaults to 10%)
	if sheet.VATRate <= 0 {
		sheet.VATRate = 10.0 // Default VAT rate if not set
	}
	sheet.VATAmountBHD = sheet.TotalFinalBHD * (sheet.VATRate / 100.0)
	sheet.GrandTotalBHD = sheet.TotalFinalBHD + sheet.VATAmountBHD

	// Step 7: Risk assessment and approval decision
	ce.AssessRiskFull(sheet, customer)

	return sheet, nil
}

// AssessRiskFull performs risk analysis using a full prediction.Customer (int flags HasABB/IsEmergency).
// This ports the root AssessRisk(sheet, customer) body exactly, preserving all thresholds.
func (ce *CostingEngine) AssessRiskFull(sheet *CostingSheet, customer *prediction.Customer) {
	warnings := make([]string, 0)

	// Check customer grade
	switch sheet.CustomerGrade {
	case GradeA:
		sheet.ApprovalStatus = "✓ APPROVE"
		sheet.RecommendedAction = "Proceed with order. Reliable customer."

	case GradeB:
		sheet.ApprovalStatus = "✓ APPROVE"
		sheet.RecommendedAction = "Proceed with caution. Monitor payment."
		warnings = append(warnings, "Moderate risk customer - monitor payment closely")

	case GradeC:
		sheet.ApprovalStatus = "⚠ CAUTION"
		sheet.RecommendedAction = "Require 50% advance payment before production."
		warnings = append(warnings, "Higher risk - require 50% advance")

	case GradeD:
		sheet.ApprovalStatus = "✗ DECLINE"
		sheet.RecommendedAction = "Decline order OR require 100% advance payment."
		warnings = append(warnings, "HIGH RISK - decline or require full advance")
	}

	o := overlay.Active()

	// Check margin adequacy (overlay-configured minimum, default 8%)
	if sheet.ActualMarginPct < o.BusinessRules.MinMarginPct {
		warnings = append(warnings, fmt.Sprintf("Low margin (%.1f%%) - consider rejecting", sheet.ActualMarginPct*100))
		if sheet.ApprovalStatus == "✓ APPROVE" {
			sheet.ApprovalStatus = "⚠ CAUTION"
		}
	}

	// Check named-competitor competition (default competitor: ABB)
	if customer.HasABB == 1 {
		competitor := o.CompetitorName()
		abbMin := o.BusinessRules.ABBCompetitionMinMargin
		warnings = append(warnings, fmt.Sprintf("%s COMPETING - only proceed if margin ≥ %.0f%%", competitor, abbMin*100))
		if sheet.ActualMarginPct < abbMin {
			sheet.RecommendedAction += fmt.Sprintf(" | ⚠ REJECT - %s competition with low margin", competitor)
			sheet.ApprovalStatus = "✗ DECLINE"
		}
	}

	// Check order size relative to customer history
	if len(customer.OrderHistory) > 0 {
		avgOrder := averageFloat(customer.OrderHistory)
		if sheet.TotalFinalBHD > avgOrder*3.0 {
			warnings = append(warnings, fmt.Sprintf("Unusually large order (%.0f%% above average)",
				(sheet.TotalFinalBHD/avgOrder-1.0)*100))
		}
	}

	// Check emergency pricing opportunity
	if customer.IsEmergency == 1 {
		warnings = append(warnings, "Emergency order - premium pricing justified (+10-20%)")
		sheet.RecommendedAction += " | Consider additional emergency surcharge."
	}

	// Check long lead times
	maxLeadTime := 0
	for _, item := range sheet.Items {
		if item.ProductionDays > maxLeadTime {
			maxLeadTime = item.ProductionDays
		}
	}
	if maxLeadTime > 14 {
		warnings = append(warnings, fmt.Sprintf("Long lead time (%d days) - manage customer expectations", maxLeadTime))
	}

	sheet.RiskWarnings = warnings
}

// PrintCostingSheet prints a formatted quotation
func (ce *CostingEngine) PrintCostingSheet(sheet *CostingSheet) {
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║ ACME INSTRUMENTATION W.L.L - QUOTATION\n")
	fmt.Println("╠═══════════════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Customer:        %s (Grade %s)\n", sheet.CustomerName, sheet.CustomerGrade)
	fmt.Printf("║ Customer ID:     %s\n", sheet.CustomerID)
	fmt.Printf("║ Quotation Date:  %s\n", sheet.QuotationDate)
	fmt.Printf("║ Valid Until:     %s\n", sheet.ValidUntil)
	fmt.Printf("║ Payment Terms:   %s\n", sheet.PaymentTerms)
	if sheet.AdvanceRequired > 0 {
		fmt.Printf("║ Advance Required: %.0f%% (%.2f BHD)\n",
			sheet.AdvanceRequired*100, sheet.TotalFinalBHD*sheet.AdvanceRequired)
	}
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Print payment prediction
	fmt.Println("PAYMENT RISK ASSESSMENT:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Grade:            %s\n", sheet.PaymentPrediction.Grade)
	fmt.Printf("Predicted Days:   %d days\n", sheet.PaymentPrediction.PredictedDays)
	fmt.Printf("Confidence:       %.0f%%\n", sheet.PaymentPrediction.Confidence*100)
	fmt.Printf("Three Regimes:    R1=%.1f%%, R2=%.1f%%, R3=%.1f%%\n",
		sheet.PaymentPrediction.ThreeRegimes.R1*100,
		sheet.PaymentPrediction.ThreeRegimes.R2*100,
		sheet.PaymentPrediction.ThreeRegimes.R3*100)
	if len(sheet.PaymentPrediction.RiskFactors) > 0 {
		fmt.Println("Risk Factors:")
		for _, rf := range sheet.PaymentPrediction.RiskFactors {
			fmt.Printf("  • %s\n", rf)
		}
	}
	fmt.Println()

	// Print line items
	fmt.Println("LINE ITEMS:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("%-40s %5s %12s %12s %12s %10s\n",
		"Description", "Qty", "Unit Cost", "Unit Sell", "Total", "Profit")
	fmt.Println("───────────────────────────────────────────────────────────────────────────────")

	for _, item := range sheet.Items {
		fmt.Printf("%-40s %5d %12.2f %12.2f %12.2f %10.2f\n",
			truncateText(item.Description, 40),
			item.Quantity,
			item.UnitCostBHD,
			item.FinalUnitBHD,
			item.FinalTotalBHD,
			item.TotalProfitBHD)
		fmt.Printf("  %s | %s | Lead: %d days | Margin: %.1f%%\n",
			item.OrderCode, item.ProductType, item.ProductionDays, item.ActualMargin*100)
		fmt.Println()
	}

	// Print totals
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("%-58s %12.2f\n", "Subtotal (before discount):", sheet.TotalSellBHD)
	if sheet.TotalDiscountBHD > 0 {
		fmt.Printf("%-58s %12.2f (%.1f%%)\n", "Customer Discount:",
			-sheet.TotalDiscountBHD, (sheet.TotalDiscountBHD/sheet.TotalSellBHD)*100)
	}
	fmt.Printf("%-58s %12.2f BHD\n", "Subtotal (after discount):", sheet.TotalFinalBHD)
	fmt.Printf("%-58s %12.2f BHD (%.1f%%)\n", "VAT:", sheet.VATAmountBHD, sheet.VATRate)
	fmt.Printf("%-58s %12.2f BHD\n", "GRAND TOTAL:", sheet.GrandTotalBHD)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Print profit analysis
	fmt.Println("PROFIT ANALYSIS:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total Cost:        %12.2f BHD\n", sheet.TotalCostBHD)
	fmt.Printf("Total Revenue:           %12.2f BHD\n", sheet.TotalFinalBHD)
	fmt.Printf("Total Profit:            %12.2f BHD\n", sheet.TotalProfitBHD)
	fmt.Printf("Standard Margin:         %12.1f%%\n", sheet.StandardMarginPct*100)
	fmt.Printf("Actual Margin:           %12.1f%%\n", sheet.ActualMarginPct*100)
	fmt.Println()

	// Print risk warnings
	if len(sheet.RiskWarnings) > 0 {
		fmt.Println("⚠ RISK WARNINGS:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for _, warning := range sheet.RiskWarnings {
			fmt.Printf("  ⚠ %s\n", warning)
		}
		fmt.Println()
	}

	// Print approval decision
	fmt.Println("APPROVAL DECISION:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Status: %s\n", sheet.ApprovalStatus)
	fmt.Printf("Action: %s\n", sheet.RecommendedAction)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
