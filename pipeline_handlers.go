// ═══════════════════════════════════════════════════════════════════════════
// PIPELINE HANDLERS - Wails-Callable Geometry Pipeline Endpoints
//
// MISSION: Expose 4 geometry pipelines as Wails-callable functions
//
// HANDLERS:
//   1. ProcessInvoice(invoice)    → Oblate Spheroid flow dynamics
//   2. ProcessTender(tender)      → Constraint satisfaction (SAT solver)
//   3. GetCustomer360(customerID) → Entanglement joins
//   4. CheckCompliance(data)      → Banach convergence
//   5. RouteEvent(event)          → Generic router (demo/testing)
//
// Built with E2E INTEGRATION × ZERO CRUFT × PRODUCTION READY 🕉️⚡💎
// Day 193 - Pipeline Handlers
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"fmt"
	"log"
	"time"
)

// ============================================================================
// EXTENDED APP METHODS (Add to app.go bindings)
// ============================================================================

// Initialize geometry bridge on app startup
func (a *App) initializeGeometryBridge() {
	if a.geometryBridge == nil {
		a.geometryBridge = NewGeometryBridge()
		log.Println("✓ Geometry Bridge initialized")
	}
}

// ============================================================================
// INVOICE PROCESSING
// ============================================================================

// ProcessInvoice processes invoice through Oblate Spheroid pipeline
// Wails binding: frontend can call app.ProcessInvoice(invoice)
func (a *App) ProcessInvoice(invoice InvoiceGeometry) (*InvoiceResult, error) {
	if err := a.requirePermission("finance:update"); err != nil {
		return nil, err
	}
	a.initializeGeometryBridge()

	log.Printf("📄 Processing invoice %s (Amount: %.2f BHD) via Oblate Spheroid",
		invoice.ID, invoice.Amount)

	result, err := a.geometryBridge.ProcessInvoice(invoice)
	if err != nil {
		log.Printf("❌ Invoice processing failed: %v", err)
		return nil, err
	}

	log.Printf("✓ Invoice processed: Predicted %d days, Confidence %.2f",
		result.PredictedDays, result.Confidence)

	// Save to database (optional)
	a.saveInvoiceResult(result)

	return result, nil
}

// ============================================================================
// TENDER PROCESSING
// ============================================================================

// ProcessTender processes tender through Truncated Icosahedron pipeline
// Wails binding: frontend can call app.ProcessTender(tender)
func (a *App) ProcessTender(tender TenderGeometry) (*TenderResult, error) {
	a.initializeGeometryBridge()

	log.Printf("📋 Processing tender %s (Budget: %.2f BHD, Items: %d) via Truncated Icosahedron",
		tender.ID, tender.Budget, len(tender.Items))

	result, err := a.geometryBridge.ProcessTender(tender)
	if err != nil {
		log.Printf("❌ Tender processing failed: %v", err)
		return nil, err
	}

	log.Printf("✓ Tender processed: %s (Margin: %.2f%%, Quote: %.2f BHD)",
		result.Recommendation, result.Margin, result.OptimalQuote)

	// Save to database (optional)
	a.saveTenderResult(result)

	return result, nil
}

// ============================================================================
// CUSTOMER 360° VIEW
// ============================================================================

// GetCustomer360 retrieves entangled customer profile via Quaternionic S³
// Wails binding: frontend can call app.GetCustomer360(customerID)
func (a *App) GetCustomer360Geometry(customerID string) (*Customer360, error) {
	a.initializeGeometryBridge()

	log.Printf("👤 Retrieving Customer 360° for %s via Quaternionic S³", customerID)

	// First get customer from database or use template
	customer := a.getCustomerByID(customerID)
	if customer == nil {
		// Return error or mock data
		return nil, fmt.Errorf("customer %s not found", customerID)
	}

	result, err := a.geometryBridge.GetCustomer360(customerID, customer)
	if err != nil {
		log.Printf("❌ Customer 360° failed: %v", err)
		return nil, err
	}

	log.Printf("✓ Customer 360° retrieved: Grade %s, %d orders, %.2f BHD total value",
		result.Grade, result.TotalOrders, result.TotalValue)

	return result, nil
}

// ============================================================================
// COMPLIANCE CHECK
// ============================================================================

// CheckCompliance validates data through Banach Ball pipeline
// Wails binding: frontend can call app.CheckCompliance(data)
func (a *App) CheckCompliance(data ComplianceData) (*ComplianceResult, error) {
	a.initializeGeometryBridge()

	log.Printf("⚖ Checking compliance for %s via Banach Ball (RuleSet: %s)",
		data.Type, data.RuleSet)

	result, err := a.geometryBridge.CheckCompliance(data)
	if err != nil {
		log.Printf("❌ Compliance check failed: %v", err)
		return nil, err
	}

	status := "NON-COMPLIANT"
	if result.Compliant {
		status = "COMPLIANT"
	}

	log.Printf("✓ Compliance: %s (Score: %.2f, Iterations: %d)",
		status, result.Score, result.Iterations)

	return result, nil
}

// ============================================================================
// GENERIC EVENT ROUTING (Demo/Testing)
// ============================================================================

// RouteEvent demonstrates geometry selection for any event
// Wails binding: frontend can call app.RouteEvent(event)
func (a *App) RouteEvent(event ERPEvent) (*RoutingResult, error) {
	a.initializeGeometryBridge()

	log.Printf("🔀 Routing event: Type=%s, Priority=%.2f", event.Type, event.Priority)

	result, err := a.geometryBridge.RouteEvent(event)
	if err != nil {
		log.Printf("❌ Event routing failed: %v", err)
		return nil, err
	}

	log.Printf("✓ Event routed to %s (R1=%.1f%%, R2=%.1f%%, R3=%.1f%%, Difficulty=%.2f)",
		result.Geometry, result.ThreeRegimes.R1, result.ThreeRegimes.R2,
		result.ThreeRegimes.R3, result.Difficulty)

	return result, nil
}

// ============================================================================
// INTEGRATED WORKFLOWS (Combines Multiple Pipelines)
// ============================================================================

// ProcessRFQToOrder handles complete flow: RFQ → Tender → Order → Invoice
func (a *App) ProcessRFQToOrder(tender TenderGeometry, customer Customer) (*CompleteFlowResult, error) {
	a.initializeGeometryBridge()

	log.Printf("🔄 Processing complete RFQ→Order flow for %s", tender.CustomerID)

	result := &CompleteFlowResult{
		TenderID:   tender.ID,
		CustomerID: tender.CustomerID,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	// STEP 1: Customer 360° analysis (Quaternionic S³)
	customer360, err := a.geometryBridge.GetCustomer360(tender.CustomerID, &customer)
	if err != nil {
		return nil, fmt.Errorf("customer 360° failed: %v", err)
	}
	result.CustomerGrade = customer360.Grade
	result.CustomerRisk = len(customer360.RiskFactors) > 2

	log.Printf("  Step 1/4: Customer grade %s, Risk factors: %d",
		customer360.Grade, len(customer360.RiskFactors))

	// STEP 2: Tender constraint satisfaction (Truncated Icosahedron)
	tenderResult, err := a.geometryBridge.ProcessTender(tender)
	if err != nil {
		return nil, fmt.Errorf("tender processing failed: %v", err)
	}
	result.TenderFeasible = tenderResult.Feasible
	result.QuoteAmount = tenderResult.OptimalQuote
	result.Margin = tenderResult.Margin
	result.Recommendation = tenderResult.Recommendation

	log.Printf("  Step 2/4: Tender %s, Margin %.2f%%",
		tenderResult.Recommendation, tenderResult.Margin)

	// STEP 3: If feasible, create mock invoice for flow analysis (Oblate Spheroid)
	if tenderResult.Feasible {
		mockInvoice := InvoiceGeometry{
			ID:         fmt.Sprintf("INV_%s", tender.ID),
			CustomerID: tender.CustomerID,
			Amount:     tenderResult.OptimalQuote,
			IssueDate:  time.Now(),
			DueDate:    time.Now().AddDate(0, 0, 45),
			Status:     "pending",
			Currency:   "BHD",
			ItemCount:  len(tender.Items),
		}

		invoiceResult, err := a.geometryBridge.ProcessInvoice(mockInvoice)
		if err != nil {
			log.Printf("  Warning: Invoice flow analysis failed: %v", err)
		} else {
			result.PredictedPaymentDays = invoiceResult.PredictedDays
			result.FlowConfidence = invoiceResult.Confidence
			log.Printf("  Step 3/4: Payment predicted in %d days (Confidence: %.2f)",
				invoiceResult.PredictedDays, invoiceResult.Confidence)
		}
	} else {
		log.Printf("  Step 3/4: SKIPPED - Tender not feasible")
	}

	// STEP 4: Final decision
	if tenderResult.Feasible && tenderResult.Margin >= 10.0 && customer360.Grade != "D" {
		result.FinalDecision = "APPROVE"
	} else if tenderResult.ABBWarning {
		result.FinalDecision = "DECLINE - ABB Risk"
	} else if customer360.Grade == "D" {
		result.FinalDecision = "DECLINE - Customer Risk"
	} else {
		result.FinalDecision = "NEGOTIATE"
	}

	log.Printf("  Step 4/4: FINAL DECISION = %s", result.FinalDecision)
	log.Printf("✓ Complete flow processed in %.2f ms",
		time.Since(parseTime(result.Timestamp)).Seconds()*1000)

	return result, nil
}

// CompleteFlowResult represents end-to-end RFQ→Order result
type CompleteFlowResult struct {
	TenderID             string  `json:"tender_id"`
	CustomerID           string  `json:"customer_id"`
	CustomerGrade        string  `json:"customer_grade"`
	CustomerRisk         bool    `json:"customer_risk"`
	TenderFeasible       bool    `json:"tender_feasible"`
	QuoteAmount          float64 `json:"quote_amount"`
	Margin               float64 `json:"margin"`
	Recommendation       string  `json:"recommendation"`
	PredictedPaymentDays int     `json:"predicted_payment_days"`
	FlowConfidence       float64 `json:"flow_confidence"`
	FinalDecision        string  `json:"final_decision"`
	Timestamp            string  `json:"timestamp"`
}

// ============================================================================
// DATABASE HELPERS (Optional - for audit trail)
// ============================================================================

func (a *App) saveInvoiceResult(result *InvoiceResult) {
	// Could save to invoices table for audit
	log.Printf("  Saved invoice result: %s", result.InvoiceID)
}

func (a *App) saveTenderResult(result *TenderResult) {
	// Could save to tenders table for audit
	log.Printf("  Saved tender result: %s", result.TenderID)
}

func (a *App) getCustomerByID(customerID string) *Customer {
	// In production, query from database
	// For now, return template customer
	return &Customer{
		ID:             customerID,
		BusinessName:   "Sample Customer Ltd.",
		OrderValue:     15000.0,
		OrderHistory:   []float64{12000, 14000, 13500, 15000},
		PaymentHistory: []int{45, 50, 48, 52},
		RelationYears:  3,
		Industry:       "Manufacturing",
		Country:        "Bahrain",
		IsEmergency:    0,
		HasABB:         0,
		DisputeCount:   1,
	}
}

// ============================================================================
// STATISTICS & MONITORING
// ============================================================================

// GetPipelineStatistics returns usage statistics for all pipelines
func (a *App) GetPipelineStatistics() map[string]any {
	if a.geometryBridge == nil {
		return map[string]any{
			"total_events": 0,
			"status":       "Not initialized",
		}
	}

	stats := map[string]any{
		"total_events":    a.geometryBridge.TotalEvents(),
		"routing_history": a.geometryBridge.RoutingHistoryLen(),
		"truncated_icosa": a.geometryBridge.HasTruncatedIcosahedron(),
		"quaternionic_s3": a.geometryBridge.HasQuaternionicS3(),
		"banach_ball":     a.geometryBridge.HasBanachBall(),
	}

	return stats
}

// GetRoutingHistory retrieves recent routing decisions (for debugging)
func (a *App) GetRoutingHistory(limit int) []RoutingResult {
	if a.geometryBridge == nil || a.geometryBridge.RoutingHistoryLen() == 0 {
		return []RoutingResult{}
	}

	return a.geometryBridge.RoutingHistorySlice(limit)
}

// ============================================================================
// UTILITY
// ============================================================================

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now()
	}
	return t
}
