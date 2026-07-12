package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// P2 UX ENHANCEMENTS (2026-01-27)
// Sales Pipeline UX Improvements
// ============================================================================

// OfferRevision represents a historical revision of an offer
type OfferRevision struct {
	ID             string    `json:"id"`
	OfferID        string    `json:"offer_id"`
	RevisionNumber int       `json:"revision_number"`
	RevisedBy      string    `json:"revised_by"`
	RevisionDate   time.Time `json:"revision_date"`
	RevisionNotes  string    `json:"revision_notes"`
	TotalValueBHD  float64   `json:"total_value_bhd"`
	Stage          string    `json:"stage"`
	ItemsJSON      string    `json:"items_json"` // Snapshot of items at this revision
}

// GetOfferRevisionHistory returns all revisions of an offer
func (a *App) GetOfferRevisionHistory(offerID string) ([]OfferRevision, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var offer Offer
	if err := a.db.Where("id = ?", offerID).First(&offer).Error; err != nil {
		return nil, fmt.Errorf("offer not found: %v", err)
	}

	// Build revision history from UpdatedAt changes
	// For now, create a simple revision log from the current offer state
	revisions := []OfferRevision{
		{
			ID:             uuid.New().String(),
			OfferID:        offer.ID,
			RevisionNumber: offer.RevisionNumber,
			RevisedBy:      offer.CreatedBy,
			RevisionDate:   offer.UpdatedAt,
			RevisionNotes:  fmt.Sprintf("Revision %d", offer.RevisionNumber),
			TotalValueBHD:  offer.TotalValueBHD,
			Stage:          offer.Stage,
			ItemsJSON:      "", // Would serialize offer.Items to JSON
		},
	}

	log.Printf("📜 Retrieved %d revisions for offer %s", len(revisions), offerID)
	return revisions, nil
}

// OpportunityDueData represents an Opportunity/RFQ with due date information
type OpportunityDueData struct {
	ID             string     `json:"id"`
	FolderNumber   string     `json:"folder_number"`
	CustomerID     string     `json:"customer_id"`
	CustomerName   string     `json:"customer_name"`
	ExpectedDate   *time.Time `json:"expected_date"`
	Stage          string     `json:"stage"`
	DaysOverdue    int        `json:"days_overdue"`
	PrimaryContact string     `json:"primary_contact"`
	PrimaryEmail   string     `json:"primary_email"`
	PrimaryPhone   string     `json:"primary_phone"`
	EstimatedValue float64    `json:"estimated_value"`
}

// GetOverdueRFQs returns Opportunities past their expected date
func (a *App) GetOverdueRFQs() ([]OpportunityDueData, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var opportunities []Opportunity
	now := time.Now()

	// Find opportunities with expected_date < now and stage still in progress
	if err := a.db.Where("expected_date < ? AND stage IN (?, ?, ?, ?)", now, "Lead", "Qualified", "Proposal", "Negotiation").
		Order("expected_date ASC").
		Find(&opportunities).Error; err != nil {
		return nil, fmt.Errorf("failed to get overdue opportunities: %v", err)
	}

	result := make([]OpportunityDueData, 0, len(opportunities))
	for _, opp := range opportunities {
		// Get primary contact for this customer
		var contact CustomerContact
		a.db.Where("customer_id = ? AND is_primary_contact = ?", opp.CustomerID, true).
			First(&contact)

		var daysOverdue int
		if opp.ExpectedDate != nil {
			daysOverdue = int(now.Sub(*opp.ExpectedDate).Hours() / 24)
		}

		result = append(result, OpportunityDueData{
			ID:             opp.ID,
			FolderNumber:   opp.FolderNumber,
			CustomerID:     opp.CustomerID,
			CustomerName:   opp.CustomerName,
			ExpectedDate:   opp.ExpectedDate,
			Stage:          opp.Stage,
			DaysOverdue:    daysOverdue,
			PrimaryContact: contact.ContactName,
			PrimaryEmail:   contact.Email,
			PrimaryPhone:   contact.Phone,
			EstimatedValue: opp.RevenueBHD,
		})
	}

	log.Printf("⚠️ Found %d overdue opportunities", len(result))
	return result, nil
}

// GetRFQsDueSoon returns Opportunities due within N days
func (a *App) GetRFQsDueSoon(days int) ([]OpportunityDueData, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var opportunities []Opportunity
	now := time.Now()
	futureDate := now.AddDate(0, 0, days)

	// Find opportunities with expected_date between now and futureDate
	if err := a.db.Where("expected_date >= ? AND expected_date <= ? AND stage IN (?, ?, ?, ?)", now, futureDate, "Lead", "Qualified", "Proposal", "Negotiation").
		Order("expected_date ASC").
		Find(&opportunities).Error; err != nil {
		return nil, fmt.Errorf("failed to get opportunities due soon: %v", err)
	}

	result := make([]OpportunityDueData, 0, len(opportunities))
	for _, opp := range opportunities {
		// Get primary contact for this customer
		var contact CustomerContact
		a.db.Where("customer_id = ? AND is_primary_contact = ?", opp.CustomerID, true).
			First(&contact)

		var daysUntilDue int
		if opp.ExpectedDate != nil {
			daysUntilDue = int(opp.ExpectedDate.Sub(now).Hours() / 24)
		}

		result = append(result, OpportunityDueData{
			ID:             opp.ID,
			FolderNumber:   opp.FolderNumber,
			CustomerID:     opp.CustomerID,
			CustomerName:   opp.CustomerName,
			ExpectedDate:   opp.ExpectedDate,
			Stage:          opp.Stage,
			DaysOverdue:    -daysUntilDue, // Negative means days remaining
			PrimaryContact: contact.ContactName,
			PrimaryEmail:   contact.Email,
			PrimaryPhone:   contact.Phone,
			EstimatedValue: opp.RevenueBHD,
		})
	}

	log.Printf("📅 Found %d opportunities due within %d days", len(result), days)
	return result, nil
}

// BulkUpdateOfferStage updates multiple offers to a new stage
func (a *App) BulkUpdateOfferStage(offerIDs []string, stage string) error {
	if err := a.requirePermission("offers:edit"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Validate stage transitions
	validStages := map[string]bool{
		"RFQ":    true,
		"Quoted": true,
		"Won":    true,
		"Lost":   true,
	}

	if !validStages[stage] {
		return fmt.Errorf("invalid stage: %s (must be RFQ, Quoted, Won, or Lost)", stage)
	}

	// Validate no invalid transitions (e.g., Won → RFQ is not allowed)
	var offers []Offer
	if err := a.db.Where("id IN ?", offerIDs).Find(&offers).Error; err != nil {
		return fmt.Errorf("failed to fetch offers: %v", err)
	}

	for _, offer := range offers {
		// Prevent backwards transitions
		if offer.Stage == "Won" && stage != "Won" {
			return fmt.Errorf("cannot change offer %s from Won to %s", offer.OfferNumber, stage)
		}
		if offer.Stage == "Lost" && stage == "RFQ" {
			return fmt.Errorf("cannot change offer %s from Lost back to RFQ", offer.OfferNumber)
		}
	}

	// Perform bulk update
	result := a.db.Model(&Offer{}).
		Where("id IN ?", offerIDs).
		Updates(map[string]any{
			"stage":      stage,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to bulk update offers: %v", result.Error)
	}

	log.Printf("✅ Bulk updated %d offers to stage: %s", result.RowsAffected, stage)
	return nil
}

// CustomerOrderHistorySummary extends Customer360Data with order analytics
type CustomerOrderHistorySummary struct {
	AvgOrderValue     float64          `json:"avg_order_value"`
	OrderFrequency    float64          `json:"order_frequency"` // Orders per year
	LastOrderDate     *time.Time       `json:"last_order_date"`
	PreferredProducts []ProductSummary `json:"preferred_products"` // Top 5
}

// ProductSummary represents a product with order frequency
type ProductSummary struct {
	ProductID     string  `json:"product_id"`
	ProductCode   string  `json:"product_code"`
	ProductName   string  `json:"product_name"`
	OrderCount    int     `json:"order_count"`
	TotalQuantity float64 `json:"total_quantity"`
	TotalValueBHD float64 `json:"total_value_bhd"`
}

// GetCustomerOrderHistorySummary calculates detailed order analytics
func (a *App) GetCustomerOrderHistorySummary(customerID string) (CustomerOrderHistorySummary, error) {
	if a.db == nil {
		return CustomerOrderHistorySummary{}, fmt.Errorf("database not initialized")
	}

	var summary CustomerOrderHistorySummary

	// 1. Calculate average order value
	var stats struct {
		AvgValue   float64
		OrderCount int
		MinDate    *time.Time
		MaxDate    *time.Time
	}

	a.db.Model(&Order{}).
		Where("customer_id = ?", customerID).
		Select("AVG(grand_total_bhd) as avg_value, COUNT(*) as order_count, MIN(order_date) as min_date, MAX(order_date) as max_date").
		Scan(&stats)

	summary.AvgOrderValue = stats.AvgValue
	summary.LastOrderDate = stats.MaxDate

	// 2. Calculate order frequency (orders per year)
	if stats.MinDate != nil && stats.MaxDate != nil && stats.OrderCount > 0 {
		daysBetween := stats.MaxDate.Sub(*stats.MinDate).Hours() / 24
		if daysBetween > 0 {
			summary.OrderFrequency = float64(stats.OrderCount) / (daysBetween / 365.25)
		}
	}

	// 3. Get preferred products (top 5 by order count)
	type ProductAggregate struct {
		ProductID     string
		ProductCode   string
		OrderCount    int
		TotalQuantity float64
		TotalValue    float64
	}

	var productAggs []ProductAggregate
	a.db.Model(&OrderItem{}).
		Select("product_id, product_code, COUNT(DISTINCT order_id) as order_count, SUM(quantity) as total_quantity, SUM(total_price) as total_value").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.customer_id = ?", customerID).
		Group("product_id, product_code").
		Order("order_count DESC").
		Limit(5).
		Scan(&productAggs)

	// Convert to ProductSummary and fetch product names
	summary.PreferredProducts = make([]ProductSummary, 0, len(productAggs))
	for _, agg := range productAggs {
		var product ProductMaster
		a.db.Where("id = ? OR product_code = ?", agg.ProductID, agg.ProductCode).First(&product)

		summary.PreferredProducts = append(summary.PreferredProducts, ProductSummary{
			ProductID:     agg.ProductID,
			ProductCode:   agg.ProductCode,
			ProductName:   product.ProductName,
			OrderCount:    agg.OrderCount,
			TotalQuantity: agg.TotalQuantity,
			TotalValueBHD: agg.TotalValue,
		})
	}

	log.Printf("📊 Customer order summary for %s: %.2f BHD avg, %.1f orders/year, %d preferred products",
		customerID, summary.AvgOrderValue, summary.OrderFrequency, len(summary.PreferredProducts))

	return summary, nil
}

// OrderSearchResult represents a search result for orders
type OrderSearchResult struct {
	ID               string    `json:"id"`
	OrderNumber      string    `json:"order_number"`
	CustomerPONumber string    `json:"customer_po_number"`
	CustomerID       string    `json:"customer_id"`
	CustomerName     string    `json:"customer_name"`
	OrderDate        time.Time `json:"order_date"`
	TotalValueBHD    float64   `json:"total_value_bhd"`
	Status           string    `json:"status"`
	MatchScore       float64   `json:"match_score"` // Relevance score
}

// SearchOrders performs fuzzy search on orders
func (a *App) SearchOrders(query string) ([]OrderSearchResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if query == "" {
		return []OrderSearchResult{}, nil
	}

	// Sanitize user input to prevent SQL injection via LIKE wildcards
	sanitized := sanitizeSearchQuery(query)
	if len(sanitized) < 2 {
		log.Printf("⚠️ Order search query too short after sanitization: '%s' -> '%s'", query, sanitized)
		return []OrderSearchResult{}, nil
	}

	// Fuzzy matching: LIKE with wildcards (SQLite compatible)
	searchPattern := "%" + sanitized + "%"

	var orders []Order
	err := a.db.Where("order_number LIKE ? OR customer_name LIKE ? OR customer_po_number LIKE ?",
		searchPattern, searchPattern, searchPattern).
		Order("order_date DESC").
		Limit(100). // Prevent DOS
		Find(&orders).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search orders: %v", err)
	}

	results := make([]OrderSearchResult, 0, len(orders))
	for _, order := range orders {
		// Calculate simple match score (higher if multiple fields match)
		score := 0.0
		queryLower := strings.ToLower(query)
		if strings.Contains(strings.ToLower(order.OrderNumber), queryLower) {
			score += 1.0
		}
		if strings.Contains(strings.ToLower(order.CustomerName), queryLower) {
			score += 0.8
		}
		if strings.Contains(strings.ToLower(order.CustomerPONumber), queryLower) {
			score += 0.6
		}

		results = append(results, OrderSearchResult{
			ID:               order.ID,
			OrderNumber:      order.OrderNumber,
			CustomerPONumber: order.CustomerPONumber,
			CustomerID:       order.CustomerID,
			CustomerName:     order.CustomerName,
			OrderDate:        order.OrderDate,
			TotalValueBHD:    order.TotalValueBHD,
			Status:           order.Status,
			MatchScore:       score,
		})
	}

	// Sort by match score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].MatchScore > results[j].MatchScore
	})

	log.Printf("🔍 Search orders '%s': found %d results", query, len(results))
	return results, nil
}

// OfferSearchResult represents a search result for offers
type OfferSearchResult struct {
	ID             string    `json:"id"`
	OfferNumber    string    `json:"offer_number"`
	CustomerID     string    `json:"customer_id"`
	CustomerName   string    `json:"customer_name"`
	QuotationDate  time.Time `json:"quotation_date"`
	TotalValueBHD  float64   `json:"total_value_bhd"`
	Stage          string    `json:"stage"`
	RevisionNumber int       `json:"revision_number"`
	MatchScore     float64   `json:"match_score"`
}

// SearchOffers performs fuzzy search on offers
func (a *App) SearchOffers(query string) ([]OfferSearchResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if query == "" {
		return []OfferSearchResult{}, nil
	}

	// Sanitize user input to prevent SQL injection via LIKE wildcards
	sanitized := sanitizeSearchQuery(query)
	if len(sanitized) < 2 {
		log.Printf("⚠️ Offer search query too short after sanitization: '%s' -> '%s'", query, sanitized)
		return []OfferSearchResult{}, nil
	}

	// Fuzzy matching: LIKE with wildcards (SQLite compatible)
	searchPattern := "%" + sanitized + "%"

	var offers []Offer
	err := a.db.Where("offer_number LIKE ? OR customer_name LIKE ? OR customer_reference LIKE ?",
		searchPattern, searchPattern, searchPattern).
		Order("quotation_date DESC").
		Limit(100). // Prevent DOS
		Find(&offers).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search offers: %v", err)
	}

	results := make([]OfferSearchResult, 0, len(offers))
	for _, offer := range offers {
		// Calculate simple match score
		score := 0.0
		queryLower := strings.ToLower(query)
		if strings.Contains(strings.ToLower(offer.OfferNumber), queryLower) {
			score += 1.0
		}
		if strings.Contains(strings.ToLower(offer.CustomerName), queryLower) {
			score += 0.8
		}
		if strings.Contains(strings.ToLower(offer.CustomerReference), queryLower) {
			score += 0.6
		}

		results = append(results, OfferSearchResult{
			ID:             offer.ID,
			OfferNumber:    offer.OfferNumber,
			CustomerID:     offer.CustomerID,
			CustomerName:   offer.CustomerName,
			QuotationDate:  offer.QuotationDate,
			TotalValueBHD:  offer.TotalValueBHD,
			Stage:          offer.Stage,
			RevisionNumber: offer.RevisionNumber,
			MatchScore:     score,
		})
	}

	// Sort by match score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].MatchScore > results[j].MatchScore
	})

	log.Printf("🔍 Search offers '%s': found %d results", query, len(results))
	return results, nil
}
