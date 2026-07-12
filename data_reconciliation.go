package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// =============================================================================
// DATA RECONCILIATION SERVICE
// =============================================================================
// Purpose: Match extracted PDF data (offers 101-350) with existing database records
// Author: Research Dyad (the maintainer + Claude)
// Date: February 5, 2026
//
// Features:
// 1. Match offers to RFQData/Offer records
// 2. Match invoices to Invoice records
// 3. Fuzzy match customer names to CustomerMaster
// 4. Link POs to orders and purchase_orders
// 5. Generate reconciliation report with discrepancies
// 6. Apply reconciliation updates to database
// =============================================================================

// ReconciliationResult contains the full reconciliation analysis
type ReconciliationResult struct {
	// Offer Matching
	TotalOffers   int `json:"total_offers"`
	MatchedOffers int `json:"matched_offers"`
	NewOffers     int `json:"new_offers"`
	WonOffers     int `json:"won_offers"`     // Offers with EXECUTION status
	PendingOffers int `json:"pending_offers"` // Offers without EXECUTION

	// Invoice Matching
	TotalInvoices     int `json:"total_invoices"`
	MatchedInvoices   int `json:"matched_invoices"`
	MissingInvoices   int `json:"missing_invoices"`
	InvoicesWithItems int `json:"invoices_with_items"` // Invoices where we filled line items

	// Customer Matching
	CustomerMatches    map[string]string  `json:"customer_matches"` // extracted name -> canonical CustomerID
	UnmatchedCustomers []string           `json:"unmatched_customers"`
	FuzzyMatches       []FuzzyMatchResult `json:"fuzzy_matches"`

	// PO Matching
	TotalPOs           int `json:"total_pos"`
	MatchedCustomerPOs int `json:"matched_customer_pos"`
	MatchedSupplierPOs int `json:"matched_supplier_pos"`

	// Discrepancies
	Discrepancies []DataDiscrepancy `json:"discrepancies"`

	// Summary
	ProcessingTime  time.Duration `json:"processing_time"`
	ConfidenceScore float64       `json:"confidence_score"` // Overall confidence 0-1
	ReadyToApply    bool          `json:"ready_to_apply"`   // Safe to run ApplyReconciliation
}

// DataDiscrepancy represents a mismatch between extracted and database data
type DataDiscrepancy struct {
	Type           string    `json:"type"` // AMOUNT_MISMATCH, DATE_MISMATCH, MISSING_RECORD, NAME_MISMATCH
	OfferNumber    string    `json:"offer_number"`
	InvoiceNumber  string    `json:"invoice_number"`
	Field          string    `json:"field"`
	ExtractedValue string    `json:"extracted_value"`
	DatabaseValue  string    `json:"database_value"`
	Severity       string    `json:"severity"`   // LOW, MEDIUM, HIGH, CRITICAL
	Confidence     float64   `json:"confidence"` // 0-1
	Timestamp      time.Time `json:"timestamp"`
}

// FuzzyMatchResult represents a fuzzy customer name match
type FuzzyMatchResult struct {
	ExtractedName string  `json:"extracted_name"`
	CanonicalName string  `json:"canonical_name"`
	CustomerID    string  `json:"customer_id"`
	MatchScore    float64 `json:"match_score"`  // 0-1 (1 = exact match)
	MatchMethod   string  `json:"match_method"` // "exact", "acronym", "fuzzy", "manual"
	NeedsReview   bool    `json:"needs_review"` // Human review required
}

// ExtractedOfferData represents data extracted from offer PDFs
type ExtractedOfferData struct {
	OfferNumber    string               `json:"offer_number"`
	RevisionNumber int                  `json:"revision_number"`
	CustomerName   string               `json:"customer_name"`
	QuotationDate  time.Time            `json:"quotation_date"`
	ValidityDate   time.Time            `json:"validity_date"`
	TotalValueBHD  float64              `json:"total_value_bhd"`
	HasExecution   bool                 `json:"has_execution"` // True if "EXECUTION" appears in PDF
	Items          []ExtractedOfferItem `json:"items"`
	RawText        string               `json:"raw_text"` // Full extracted text for analysis
}

// ExtractedOfferItem represents a line item from offer PDF
type ExtractedOfferItem struct {
	LineNumber  int     `json:"line_number"`
	Description string  `json:"description"`
	Model       string  `json:"model"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price_bhd"`
	TotalPrice  float64 `json:"total_price_bhd"`
}

// ExtractedInvoiceData represents data extracted from invoice PDFs
type ExtractedInvoiceData struct {
	InvoiceNumber    string                 `json:"invoice_number"`
	InvoiceDate      time.Time              `json:"invoice_date"`
	CustomerName     string                 `json:"customer_name"`
	CustomerPONumber string                 `json:"customer_po_number"`
	GrandTotalBHD    float64                `json:"grand_total_bhd"`
	Items            []ExtractedInvoiceItem `json:"items"`
}

// ExtractedInvoiceItem represents a line item from invoice PDF
type ExtractedInvoiceItem struct {
	LineNumber  int     `json:"line_number"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Rate        float64 `json:"rate"`
	TotalBHD    float64 `json:"total_bhd"`
}

// CustomerNameMapping stores canonical customer name mappings
type CustomerNameMapping struct {
	ID            string    `gorm:"primaryKey;size:36" json:"id"`
	ExtractedName string    `gorm:"uniqueIndex;size:500" json:"extracted_name"`
	CanonicalName string    `gorm:"size:255" json:"canonical_name"`
	CustomerID    string    `gorm:"index;size:36" json:"customer_id"`
	MatchScore    float64   `json:"match_score"`
	MatchMethod   string    `gorm:"size:50" json:"match_method"`
	Verified      bool      `gorm:"default:false" json:"verified"` // Human verified
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (CustomerNameMapping) TableName() string { return "customer_name_mappings" }

// =============================================================================
// WAILS BINDINGS
// =============================================================================

// ReconcileOfferData performs full reconciliation of extracted PDF data
func (a *App) ReconcileOfferData(basePath string) (*ReconciliationResult, error) {
	// Wave 8 P0: filesystem reconcile over a caller-supplied path; PH gates documents:view.
	if err := a.requirePermission("documents:view"); err != nil {
		return nil, err
	}
	startTime := time.Now()

	log.Printf("🔄 Starting data reconciliation from path: %s", basePath)

	result := &ReconciliationResult{
		CustomerMatches: make(map[string]string),
		Discrepancies:   []DataDiscrepancy{},
		FuzzyMatches:    []FuzzyMatchResult{},
	}

	// 1. Load extracted offer data from PDFs
	offers, err := a.loadExtractedOffers(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load offers: %w", err)
	}
	result.TotalOffers = len(offers)
	log.Printf("📄 Loaded %d extracted offers", len(offers))

	// 2. Match offers to database (RFQData and Offer tables)
	matched, newOffers, won, pending := a.matchOffersToDB(offers, result)
	result.MatchedOffers = matched
	result.NewOffers = newOffers
	result.WonOffers = won
	result.PendingOffers = pending
	log.Printf("✅ Matched %d offers, %d new, %d won, %d pending", matched, newOffers, won, pending)

	// 3. Load extracted invoice data
	invoices, err := a.loadExtractedInvoices(basePath)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to load invoices: %v", err)
	} else {
		result.TotalInvoices = len(invoices)
		log.Printf("📄 Loaded %d extracted invoices", len(invoices))

		// 4. Match invoices to database
		matchedInv, missingInv, withItems := a.matchInvoicesToDB(invoices, result)
		result.MatchedInvoices = matchedInv
		result.MissingInvoices = missingInv
		result.InvoicesWithItems = withItems
		log.Printf("✅ Matched %d invoices, %d missing, %d with items", matchedInv, missingInv, withItems)
	}

	// 5. Fuzzy match customer names
	a.fuzzyMatchCustomers(result)
	log.Printf("🔍 Fuzzy matched %d customers, %d unmatched", len(result.CustomerMatches), len(result.UnmatchedCustomers))

	// 6. Calculate confidence score
	result.ConfidenceScore = a.calculateConfidenceScore(result)
	result.ReadyToApply = result.ConfidenceScore >= 0.70

	result.ProcessingTime = time.Since(startTime)
	log.Printf("✅ Reconciliation complete in %v (confidence: %.1f%%)", result.ProcessingTime, result.ConfidenceScore*100)

	return result, nil
}

// GetCustomerNameMapping returns the current customer name mappings
func (a *App) GetCustomerNameMapping() (map[string]string, error) {
	var mappings []CustomerNameMapping

	err := a.db.Find(&mappings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load mappings: %w", err)
	}

	result := make(map[string]string)
	for _, m := range mappings {
		result[m.ExtractedName] = m.CustomerID
	}

	return result, nil
}

// ApplyReconciliation applies the reconciliation updates to the database
func (a *App) ApplyReconciliation(resultJSON string) error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	log.Printf("🔄 Applying reconciliation updates...")

	var result ReconciliationResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return fmt.Errorf("failed to parse result: %w", err)
	}

	if !result.ReadyToApply {
		return fmt.Errorf("reconciliation not ready to apply (confidence: %.1f%%, need ≥70%%)", result.ConfidenceScore*100)
	}

	// Begin transaction
	tx := a.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Apply customer name mappings
	for extractedName, customerID := range result.CustomerMatches {
		mapping := CustomerNameMapping{
			ExtractedName: extractedName,
			CustomerID:    customerID,
			MatchScore:    1.0,
			MatchMethod:   "reconciliation",
			Verified:      true,
		}

		// Get canonical name from customer table
		var customer CustomerMaster
		if err := tx.Where("id = ?", customerID).First(&customer).Error; err == nil {
			mapping.CanonicalName = customer.BusinessName
		}

		tx.Save(&mapping)
	}

	// 2. Apply fuzzy matches with high confidence
	for _, match := range result.FuzzyMatches {
		if match.MatchScore >= 0.80 && !match.NeedsReview {
			mapping := CustomerNameMapping{
				ExtractedName: match.ExtractedName,
				CanonicalName: match.CanonicalName,
				CustomerID:    match.CustomerID,
				MatchScore:    match.MatchScore,
				MatchMethod:   match.MatchMethod,
				Verified:      false, // Auto-matched, not human verified
			}
			tx.Save(&mapping)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	log.Printf("✅ Reconciliation applied successfully")
	return nil
}

// =============================================================================
// INTERNAL MATCHING LOGIC
// =============================================================================

// loadExtractedOffers loads offer data from extracted JSON or PDFs
func (a *App) loadExtractedOffers(basePath string) ([]ExtractedOfferData, error) {
	var offers []ExtractedOfferData

	// Check if there's a consolidated JSON file
	consolidatedPath := filepath.Join(basePath, "extracted_offers.json")
	if data, err := os.ReadFile(consolidatedPath); err == nil {
		if err := json.Unmarshal(data, &offers); err == nil {
			return offers, nil
		}
	}

	// Otherwise, walk the directory and load individual offer files
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".json") {
			// Try to load as offer data
			data, err := os.ReadFile(path)
			if err != nil {
				return nil // Skip file
			}

			var offer ExtractedOfferData
			if err := json.Unmarshal(data, &offer); err == nil {
				if offer.OfferNumber != "" {
					offers = append(offers, offer)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return offers, nil
}

// loadExtractedInvoices loads invoice data from extracted JSON
func (a *App) loadExtractedInvoices(basePath string) ([]ExtractedInvoiceData, error) {
	var invoices []ExtractedInvoiceData

	// Check for consolidated invoice JSON
	consolidatedPath := filepath.Join(basePath, "extracted_invoices.json")
	if data, err := os.ReadFile(consolidatedPath); err == nil {
		if err := json.Unmarshal(data, &invoices); err == nil {
			return invoices, nil
		}
	}

	// Walk directory for individual invoice files
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.Contains(strings.ToLower(path), "invoice") && strings.HasSuffix(strings.ToLower(path), ".json") {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			var invoice ExtractedInvoiceData
			if err := json.Unmarshal(data, &invoice); err == nil {
				if invoice.InvoiceNumber != "" {
					invoices = append(invoices, invoice)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return invoices, nil
}

// matchOffersToDB matches extracted offers to database records
func (a *App) matchOffersToDB(offers []ExtractedOfferData, result *ReconciliationResult) (matched, newOffers, won, pending int) {
	for _, offer := range offers {
		// Try to find matching offer in Offer table
		var dbOffer Offer
		err := a.db.Where("offer_number = ?", offer.OfferNumber).First(&dbOffer).Error

		if err == gorm.ErrRecordNotFound {
			// New offer - check RFQData table
			var rfq RFQData
			rfqErr := a.db.Where("rfq_number = ?", offer.OfferNumber).First(&rfq).Error

			if rfqErr == gorm.ErrRecordNotFound {
				newOffers++
				result.Discrepancies = append(result.Discrepancies, DataDiscrepancy{
					Type:           "MISSING_RECORD",
					OfferNumber:    offer.OfferNumber,
					Field:          "offer_record",
					ExtractedValue: "Offer found in PDF",
					DatabaseValue:  "Not in database",
					Severity:       "MEDIUM",
					Confidence:     0.90,
					Timestamp:      time.Now(),
				})
			} else {
				// Found in RFQ table
				matched++
			}
		} else {
			// Found in Offer table
			matched++

			// Check for discrepancies
			if math.Abs(offer.TotalValueBHD-dbOffer.TotalValueBHD) > 0.01 {
				result.Discrepancies = append(result.Discrepancies, DataDiscrepancy{
					Type:           "AMOUNT_MISMATCH",
					OfferNumber:    offer.OfferNumber,
					Field:          "total_value_bhd",
					ExtractedValue: fmt.Sprintf("%.3f", offer.TotalValueBHD),
					DatabaseValue:  fmt.Sprintf("%.3f", dbOffer.TotalValueBHD),
					Severity:       "HIGH",
					Confidence:     0.95,
					Timestamp:      time.Now(),
				})
			}

			// Check for date discrepancies
			if !offer.QuotationDate.IsZero() && !dbOffer.QuotationDate.IsZero() {
				daysDiff := math.Abs(offer.QuotationDate.Sub(dbOffer.QuotationDate).Hours() / 24)
				if daysDiff > 1 {
					result.Discrepancies = append(result.Discrepancies, DataDiscrepancy{
						Type:           "DATE_MISMATCH",
						OfferNumber:    offer.OfferNumber,
						Field:          "quotation_date",
						ExtractedValue: offer.QuotationDate.Format("2006-01-02"),
						DatabaseValue:  dbOffer.QuotationDate.Format("2006-01-02"),
						Severity:       "LOW",
						Confidence:     0.85,
						Timestamp:      time.Now(),
					})
				}
			}
		}

		// Track won vs pending
		if offer.HasExecution {
			won++
		} else {
			pending++
		}
	}

	return
}

// matchInvoicesToDB matches extracted invoices to database records
func (a *App) matchInvoicesToDB(invoices []ExtractedInvoiceData, result *ReconciliationResult) (matched, missing, withItems int) {
	for _, invoice := range invoices {
		var dbInvoice Invoice
		err := a.db.Preload("Items").Where("invoice_number = ?", invoice.InvoiceNumber).First(&dbInvoice).Error

		if err == gorm.ErrRecordNotFound {
			missing++
			result.Discrepancies = append(result.Discrepancies, DataDiscrepancy{
				Type:           "MISSING_RECORD",
				InvoiceNumber:  invoice.InvoiceNumber,
				Field:          "invoice_record",
				ExtractedValue: "Invoice found in PDF",
				DatabaseValue:  "Not in database",
				Severity:       "HIGH",
				Confidence:     0.90,
				Timestamp:      time.Now(),
			})
		} else {
			matched++

			// Check if invoice has line items - if not, we can fill them from extraction
			if len(dbInvoice.Items) == 0 && len(invoice.Items) > 0 {
				withItems++
			}

			// Check amount discrepancy
			if math.Abs(invoice.GrandTotalBHD-dbInvoice.GrandTotalBHD) > 0.01 {
				result.Discrepancies = append(result.Discrepancies, DataDiscrepancy{
					Type:           "AMOUNT_MISMATCH",
					InvoiceNumber:  invoice.InvoiceNumber,
					Field:          "grand_total_bhd",
					ExtractedValue: fmt.Sprintf("%.3f", invoice.GrandTotalBHD),
					DatabaseValue:  fmt.Sprintf("%.3f", dbInvoice.GrandTotalBHD),
					Severity:       "CRITICAL",
					Confidence:     0.95,
					Timestamp:      time.Now(),
				})
			}
		}
	}

	return
}

// fuzzyMatchCustomers performs fuzzy matching of customer names
func (a *App) fuzzyMatchCustomers(result *ReconciliationResult) {
	// Get all customers from database
	var customers []CustomerMaster
	a.db.Find(&customers)

	// Build index of customer names
	customerIndex := make(map[string]CustomerMaster)
	for _, c := range customers {
		customerIndex[strings.ToLower(c.BusinessName)] = c
	}

	// Get unique extracted customer names
	extractedNames := make(map[string]bool)

	// Collect from discrepancies and other sources
	for _, disc := range result.Discrepancies {
		if disc.Type == "NAME_MISMATCH" {
			extractedNames[disc.ExtractedValue] = true
		}
	}

	// Try to match each extracted name
	for extractedName := range extractedNames {
		match, found := a.findBestCustomerMatch(extractedName, customers)

		if found {
			result.CustomerMatches[extractedName] = match.CustomerID
			result.FuzzyMatches = append(result.FuzzyMatches, match)
		} else {
			result.UnmatchedCustomers = append(result.UnmatchedCustomers, extractedName)
		}
	}
}

// findBestCustomerMatch finds the best matching customer for a given name
func (a *App) findBestCustomerMatch(extractedName string, customers []CustomerMaster) (FuzzyMatchResult, bool) {
	normalizedExtracted := normalizeCustomerName(extractedName)

	bestMatch := FuzzyMatchResult{
		ExtractedName: extractedName,
		MatchScore:    0.0,
		NeedsReview:   false,
	}

	for _, customer := range customers {
		normalizedDB := normalizeCustomerName(customer.BusinessName)

		// Exact match
		if normalizedExtracted == normalizedDB {
			return FuzzyMatchResult{
				ExtractedName: extractedName,
				CanonicalName: customer.BusinessName,
				CustomerID:    customer.ID,
				MatchScore:    1.0,
				MatchMethod:   "exact",
				NeedsReview:   false,
			}, true
		}

		// Acronym match (e.g., "GSC" -> "GULF SMELTING B.S.C.(C)")
		if isAcronymMatch(normalizedExtracted, normalizedDB) {
			score := 0.95
			if score > bestMatch.MatchScore {
				bestMatch = FuzzyMatchResult{
					ExtractedName: extractedName,
					CanonicalName: customer.BusinessName,
					CustomerID:    customer.ID,
					MatchScore:    score,
					MatchMethod:   "acronym",
					NeedsReview:   false,
				}
			}
		}

		// Fuzzy string match (Levenshtein-based)
		score := fuzzyStringMatch(normalizedExtracted, normalizedDB)
		if score > bestMatch.MatchScore && score >= 0.70 {
			bestMatch = FuzzyMatchResult{
				ExtractedName: extractedName,
				CanonicalName: customer.BusinessName,
				CustomerID:    customer.ID,
				MatchScore:    score,
				MatchMethod:   "fuzzy",
				NeedsReview:   score < 0.85, // Review if confidence is borderline
			}
		}
	}

	if bestMatch.MatchScore >= 0.70 {
		return bestMatch, true
	}

	return bestMatch, false
}

// normalizeCustomerName normalizes a customer name for matching
func normalizeCustomerName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Remove common suffixes and legal entities
	suffixes := []string{
		" b.s.c.(c)", " bsc", " b.s.c", " bsc(c)",
		" ltd", " limited", " llc", " inc", " corp",
		" corporation", " company", " co", " w.l.l", " wll",
	}

	for _, suffix := range suffixes {
		name = strings.TrimSuffix(name, suffix)
	}

	// Remove special characters
	reg := regexp.MustCompile(`[^a-z0-9\s]`)
	name = reg.ReplaceAllString(name, "")

	// Collapse whitespace
	name = strings.Join(strings.Fields(name), " ")

	return strings.TrimSpace(name)
}

// isAcronymMatch checks if extracted name is an acronym of the canonical name
func isAcronymMatch(acronym, fullName string) bool {
	if len(acronym) < 2 || len(acronym) > 10 {
		return false
	}

	// Extract initials from full name
	words := strings.Fields(fullName)
	if len(words) < 2 {
		return false
	}

	initials := ""
	for _, word := range words {
		if len(word) > 0 && word[0] >= 'a' && word[0] <= 'z' {
			initials += string(word[0])
		}
	}

	return acronym == initials
}

// fuzzyStringMatch computes similarity score between two strings (0-1)
func fuzzyStringMatch(s1, s2 string) float64 {
	// Simple Levenshtein-based similarity
	distance := levenshteinDistance(s1, s2)
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - (float64(distance) / maxLen)
}

// levenshteinDistance computes the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = minInt3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// minInt3 returns the minimum of three integers
func minInt3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// calculateConfidenceScore computes overall confidence in reconciliation
func (a *App) calculateConfidenceScore(result *ReconciliationResult) float64 {
	if result.TotalOffers == 0 {
		return 0.0
	}

	// Factors contributing to confidence:
	// 1. Offer match rate (40%)
	offerMatchRate := float64(result.MatchedOffers) / float64(result.TotalOffers)

	// 2. Customer match rate (30%)
	totalCustomers := len(result.CustomerMatches) + len(result.UnmatchedCustomers)
	customerMatchRate := 1.0
	if totalCustomers > 0 {
		customerMatchRate = float64(len(result.CustomerMatches)) / float64(totalCustomers)
	}

	// 3. Low discrepancy rate (20%)
	criticalDiscrepancies := 0
	for _, disc := range result.Discrepancies {
		if disc.Severity == "CRITICAL" || disc.Severity == "HIGH" {
			criticalDiscrepancies++
		}
	}
	discrepancyScore := 1.0
	if result.TotalOffers > 0 {
		discrepancyRate := float64(criticalDiscrepancies) / float64(result.TotalOffers)
		discrepancyScore = math.Max(0.0, 1.0-discrepancyRate)
	}

	// 4. Invoice match rate (10%)
	invoiceMatchRate := 1.0
	if result.TotalInvoices > 0 {
		invoiceMatchRate = float64(result.MatchedInvoices) / float64(result.TotalInvoices)
	}

	// Weighted average
	confidence := (offerMatchRate * 0.40) +
		(customerMatchRate * 0.30) +
		(discrepancyScore * 0.20) +
		(invoiceMatchRate * 0.10)

	return confidence
}

// =============================================================================
// UTILITY FUNCTIONS FOR MANUAL REVIEW
// =============================================================================

// GetDiscrepanciesByType returns discrepancies filtered by type and severity
func (a *App) GetDiscrepanciesByType(resultJSON string, discType string, minSeverity string) ([]DataDiscrepancy, error) {
	var result ReconciliationResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, err
	}

	severityOrder := map[string]int{
		"LOW":      1,
		"MEDIUM":   2,
		"HIGH":     3,
		"CRITICAL": 4,
	}

	minSev := severityOrder[minSeverity]

	var filtered []DataDiscrepancy
	for _, disc := range result.Discrepancies {
		matchType := discType == "" || disc.Type == discType
		matchSeverity := severityOrder[disc.Severity] >= minSev

		if matchType && matchSeverity {
			filtered = append(filtered, disc)
		}
	}

	return filtered, nil
}

// AddManualCustomerMapping adds a manual customer name mapping
func (a *App) AddManualCustomerMapping(extractedName string, customerID string) error {
	if err := a.requirePermission("customers:update"); err != nil {
		return err
	}
	var customer CustomerMaster
	if err := a.db.Where("id = ?", customerID).First(&customer).Error; err != nil {
		return fmt.Errorf("customer not found: %w", err)
	}

	mapping := CustomerNameMapping{
		ExtractedName: extractedName,
		CanonicalName: customer.BusinessName,
		CustomerID:    customerID,
		MatchScore:    1.0,
		MatchMethod:   "manual",
		Verified:      true,
	}

	return a.db.Save(&mapping).Error
}

// ExportReconciliationReport exports reconciliation report as JSON
func (a *App) ExportReconciliationReport(result *ReconciliationResult, outputPath string) error {
	if err := a.requirePermission("reports:export"); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}
