package main

import (
	"fmt"
	"log"
	"time"
)

// =============================================================================
// P1 FIX #1: OFFER EXPIRY HANDLING
// =============================================================================

// ValidateOfferExpiry validates that ValidUntil/ValidityDate has not already passed.
func ValidateOfferExpiry(validityDate time.Time) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	validityDay := time.Date(validityDate.Year(), validityDate.Month(), validityDate.Day(), 0, 0, 0, 0, now.Location())
	if validityDay.Before(today) {
		return fmt.Errorf("offer validity date must be today or in the future (received: %s, current: %s)",
			validityDate.Format("2006-01-02"), now.Format("2006-01-02"))
	}
	return nil
}

// AutoExpireOffers scans all Quoted/Proposal stage offers and marks expired ones
// Should be called periodically (e.g., daily cron job or on app startup)
func (a *App) AutoExpireOffers() error {
	if err := a.requirePermission("offers:edit"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Find all active offers whose validity date ended before today.
	result := a.db.Model(&Offer{}).
		Where("stage IN (?, ?) AND validity_date < ?", "Quoted", "RFQ", today).
		Update("stage", "Expired")

	if result.Error != nil {
		log.Printf("❌ Failed to auto-expire offers: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Printf("⏰ Auto-expired %d offers past their validity date", result.RowsAffected)
	}

	return nil
}

// =============================================================================
// P1 FIX #2: RFQ-TO-ORDER TRACEABILITY
// =============================================================================

// ValidateRFQTraceability ensures Order has RFQID when created from offer
// This is a validation function - call in CreateOrderFromOffer or MarkOfferWon
func ValidateRFQTraceability(offer *Offer, order *Order) error {
	// If offer has an RFQ link, order MUST preserve it
	if offer.RFQID != "" && order.RFQID == "" {
		return fmt.Errorf("RFQID missing in order - traceability broken (offer.RFQID=%s)", offer.RFQID)
	}

	// If offer has RFQ link, order's RFQID must match
	if offer.RFQID != "" && order.RFQID != offer.RFQID {
		return fmt.Errorf("RFQID mismatch - order.RFQID (%s) != offer.RFQID (%s)", order.RFQID, offer.RFQID)
	}

	return nil
}

// GetRFQTraceability retrieves full pipeline traceability for an order
func (a *App) GetRFQTraceability(orderID string) (*PipelineTraceability, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var order Order
	if err := a.db.First(&order, "id = ?", orderID).Error; err != nil {
		return nil, fmt.Errorf("order not found: %v", err)
	}

	trace := &PipelineTraceability{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		RFQID:       order.RFQID,
		OfferID:     order.OfferID,
		OfferNumber: order.OfferNumber,
	}

	// Fetch RFQ details if linked
	if order.RFQID != "" {
		var rfq RFQData
		if err := a.db.Where("id = ?", order.RFQID).First(&rfq).Error; err == nil {
			trace.RFQFound = true
			trace.RFQStatus = rfq.Status
			trace.RFQCustomer = rfq.Client
		}
	}

	// Fetch Offer details if linked
	if order.OfferID != "" {
		var offer Offer
		if err := a.db.Where("id = ?", order.OfferID).First(&offer).Error; err == nil {
			trace.OfferFound = true
			trace.OfferStage = offer.Stage
			trace.OfferValue = offer.TotalValueBHD
		}
	}

	return trace, nil
}

type PipelineTraceability struct {
	OrderID     string  `json:"order_id"`
	OrderNumber string  `json:"order_number"`
	RFQID       string  `json:"rfq_id"`
	RFQFound    bool    `json:"rfq_found"`
	RFQStatus   string  `json:"rfq_status"`
	RFQCustomer string  `json:"rfq_customer"`
	OfferID     string  `json:"offer_id"`
	OfferNumber string  `json:"offer_number"`
	OfferFound  bool    `json:"offer_found"`
	OfferStage  string  `json:"offer_stage"`
	OfferValue  float64 `json:"offer_value"`
}

// =============================================================================
// P1 FIX #3: COSTING SHEET VERSIONING
// =============================================================================

// CreateCostingSheetVersion creates a new version when editing finalized sheets
func (a *App) CreateCostingSheetVersion(originalID string, updatedBy string) (*DBCostingSheet, error) {
	if err := a.requirePermission("offers:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var original DBCostingSheet
	if err := a.db.Preload("Items").Preload("AdditionalCosts").First(&original, "id = ?", originalID).Error; err != nil {
		return nil, fmt.Errorf("costing sheet not found: %v", err)
	}

	// Create new version with incremented version number
	newVersion := &DBCostingSheet{
		Base:               Base{ID: "", Version: original.Version + 1}, // ID auto-generated
		CostingNumber:      fmt.Sprintf("%s-v%d", original.CostingNumber, original.Version+1),
		CustomerID:         original.CustomerID,
		CustomerName:       original.CustomerName,
		CostingDate:        time.Now(),
		ValidUntil:         original.ValidUntil,
		SubtotalBHD:        original.SubtotalBHD,
		TotalMarginBHD:     original.TotalMarginBHD,
		ShippingCostBHD:    original.ShippingCostBHD,
		CustomsDutyBHD:     original.CustomsDutyBHD,
		ClearanceCostBHD:   original.ClearanceCostBHD,
		HandlingCostBHD:    original.HandlingCostBHD,
		AdditionalCostsBHD: original.AdditionalCostsBHD,
		GrandTotalBHD:      original.GrandTotalBHD,
		Status:             "Draft", // New version starts as Draft
		ConvertedToOfferID: "",      // Not converted yet
	}

	// Clone items
	for _, item := range original.Items {
		newItem := DBCostingItem{
			Base:          Base{ID: ""}, // Auto-generated
			LineNumber:    item.LineNumber,
			ProductID:     item.ProductID,
			ProductType:   item.ProductType,
			Description:   item.Description,
			Quantity:      item.Quantity,
			UnitCostBHD:   item.UnitCostBHD,
			MarginPercent: item.MarginPercent,
			UnitPriceBHD:  item.UnitPriceBHD,
			LineTotalBHD:  item.LineTotalBHD,
		}
		newVersion.Items = append(newVersion.Items, newItem)
	}

	// Clone additional costs
	for _, cost := range original.AdditionalCosts {
		newCost := DBCostingAdditionalCost{
			Base:        Base{ID: ""}, // Auto-generated
			Description: cost.Description,
			AmountBHD:   cost.AmountBHD,
		}
		newVersion.AdditionalCosts = append(newVersion.AdditionalCosts, newCost)
	}

	// Save new version
	if err := a.db.Create(newVersion).Error; err != nil {
		return nil, fmt.Errorf("failed to create costing sheet version: %v", err)
	}

	log.Printf("✅ Created Costing Sheet version %d from %s (by %s)", newVersion.Version, original.CostingNumber, updatedBy)
	return newVersion, nil
}

// ValidateCostingSheetEditable checks if a costing sheet can be edited
func ValidateCostingSheetEditable(sheet *DBCostingSheet) error {
	// Cannot edit if status is "Approved" or "Converted"
	if sheet.Status == "Approved" {
		return fmt.Errorf("cannot edit approved costing sheet %s - create a new version instead", sheet.CostingNumber)
	}

	if sheet.Status == "Converted" || sheet.ConvertedToOfferID != "" {
		return fmt.Errorf("cannot edit costing sheet %s - already converted to offer %s", sheet.CostingNumber, sheet.ConvertedToOfferID)
	}

	return nil
}

// UpdateCostingSheetWithVersionCheck updates a costing sheet with edit validation
func (a *App) UpdateCostingSheetWithVersionCheck(sheetID string, updates map[string]any, updatedBy string) error {
	if err := a.requirePermission("offers:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var sheet DBCostingSheet
	if err := a.db.First(&sheet, "id = ?", sheetID).Error; err != nil {
		return fmt.Errorf("costing sheet not found: %v", err)
	}

	// Validate editable
	if err := ValidateCostingSheetEditable(&sheet); err != nil {
		return err
	}

	// Mission I (I-12): the client map was applied unfiltered — any column
	// (status, approved_by, version, created_by) was writable, bypassing the
	// approval workflow this very function's editability check protects.
	// Whitelist the fields a costing edit legitimately touches.
	allowedColumns := map[string]bool{
		"customer_id": true, "customer_name": true,
		"costing_date": true, "valid_until": true,
		"subtotal_bhd": true, "total_margin_bhd": true,
		"shipping_cost_bhd": true, "customs_duty_bhd": true,
		"clearance_cost_bhd": true, "handling_cost_bhd": true,
		"additional_costs_bhd": true, "grand_total_bhd": true,
	}
	filtered := make(map[string]any, len(updates))
	for key, value := range updates {
		if allowedColumns[key] {
			filtered[key] = value
		} else {
			log.Printf("⚠️ UpdateCostingSheetWithVersionCheck: dropped non-editable column %q (by %s)", key, updatedBy)
		}
	}
	if len(filtered) == 0 {
		return fmt.Errorf("no editable fields in update payload")
	}

	// Perform update
	if err := a.db.Model(&sheet).Updates(filtered).Error; err != nil {
		return fmt.Errorf("failed to update costing sheet: %v", err)
	}

	log.Printf("✅ Updated Costing Sheet %s (by %s)", sheet.CostingNumber, updatedBy)
	return nil
}

// =============================================================================
// P1 FIX #4: MARGIN THRESHOLD ALERTS
// =============================================================================

// MarginAlert represents a low-margin warning
type MarginAlert struct {
	Severity       string  `json:"severity"` // "warning" or "critical"
	Message        string  `json:"message"`
	MarginPercent  float64 `json:"margin_percent"`
	Threshold      float64 `json:"threshold"`
	Recommendation string  `json:"recommendation"`
}

// CheckMarginThreshold validates margin and generates alerts
func CheckMarginThreshold(margin float64, totalValue float64) *MarginAlert {
	const (
		CriticalThreshold = 5.0  // < 5% = critical
		WarningThreshold  = 10.0 // < 10% = warning
	)

	if margin < CriticalThreshold {
		return &MarginAlert{
			Severity:       "critical",
			Message:        fmt.Sprintf("CRITICAL: Gross margin %.2f%% is below %.0f%% threshold", margin, CriticalThreshold),
			MarginPercent:  margin,
			Threshold:      CriticalThreshold,
			Recommendation: "This deal will likely result in a loss when accounting for overhead. Consider rejecting or renegotiating terms.",
		}
	}

	if margin < WarningThreshold {
		return &MarginAlert{
			Severity:       "warning",
			Message:        fmt.Sprintf("WARNING: Gross margin %.2f%% is below %.0f%% threshold", margin, WarningThreshold),
			MarginPercent:  margin,
			Threshold:      WarningThreshold,
			Recommendation: "Low margin deal - requires management review and approval before proceeding.",
		}
	}

	return nil // No alert
}

// LogMarginAlert records margin alerts to database for management review
func (a *App) LogMarginAlert(entityType string, entityID string, alert *MarginAlert) error {
	if err := a.requirePermission("offers:view"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if alert == nil {
		return nil // No alert to log
	}

	// Create alert record
	alertRecord := &Alert{
		Base:           Base{ID: ""},
		AlertType:      fmt.Sprintf("low_margin_%s", entityType),
		Severity:       alert.Severity,
		Title:          fmt.Sprintf("Low Margin %s", entityType),
		Message:        fmt.Sprintf("%s\n\nEntity: %s\nRecommendation: %s", alert.Message, entityID, alert.Recommendation),
		IsActive:       true,
		IsAcknowledged: false,
	}

	if err := a.db.Create(alertRecord).Error; err != nil {
		log.Printf("❌ Failed to log margin alert: %v", err)
		return err
	}

	log.Printf("⚠️ MARGIN ALERT [%s]: %s for %s %s", alert.Severity, alert.Message, entityType, entityID)
	return nil
}

// GetLowMarginOffers retrieves all offers with margins below threshold
func (a *App) GetLowMarginOffers(threshold float64) ([]Offer, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var offers []Offer
	if err := a.db.Where("estimated_margin < ? AND stage NOT IN (?)", threshold, []string{"Lost", "Expired", "Won"}).
		Order("estimated_margin ASC").
		Find(&offers).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve low margin offers: %v", err)
	}

	log.Printf("📊 Found %d active offers with margin < %.1f%%", len(offers), threshold)
	return offers, nil
}
