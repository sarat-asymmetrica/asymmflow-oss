package data

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"ph_holdings_app/pkg/overlay"
)

// =============================================================================
// OPPORTUNITY STAGE VOCABULARY (package-local copy)
// =============================================================================
//
// This package cannot depend on the main package's canonical stage helpers
// (pkg/data is imported BY main, not the other way around), so it carries a
// small self-contained copy of the canonical Opportunity/RFQ stage enum and
// the owner-ratified legacy migration map. Keep in sync with
// canonicalOpportunityStages / legacyOpportunityStageMap in
// stage_vocabulary.go at the repo root if that vocabulary ever changes.

var canonicalOpportunityStagesSSOT = []string{
	"New", "Qualified", "Proposal", "Quoted", "Won", "Lost", "Expired", "On Hold",
}

var legacyOpportunityStageMapSSOT = map[string]string{
	"RFQ Received":     "New",
	"Costing":          "Proposal",
	"Tender":           "Proposal",
	"Offer Sent":       "Quoted",
	"Follow-up/Eval":   "Quoted",
	"PO/LOI Received":  "Won",
	"Order Placed":     "Won",
	"In Process":       "Won",
	"Delivered":        "Won",
	"Closed (Payment)": "Won",
	"Closed (Lost)":    "Lost",
	"In Progress":      "Proposal",
}

func isCanonicalOpportunityStageSSOT(s string) bool {
	for _, c := range canonicalOpportunityStagesSSOT {
		if c == s {
			return true
		}
	}
	return false
}

// canonicalizeOpportunityStageSSOT mirrors canonicalizeOpportunityStage in
// the main package: trims, applies the ratified legacy map, and never
// guesses at genuinely unrecognized values (caller decides). The SSOT
// importer is lenient — it always coerces unrecognized/unmapped values to
// "New" and logs a warning rather than aborting the import.
func canonicalizeOpportunityStageSSOT(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "New"
	}
	if isCanonicalOpportunityStageSSOT(trimmed) {
		return trimmed
	}
	if target, ok := legacyOpportunityStageMapSSOT[trimmed]; ok {
		return target
	}
	log.Printf("⚠️ SSOT import: unrecognized opportunity stage %q, coercing to \"New\"", trimmed)
	return "New"
}

// =============================================================================
// SSOT IMPORTER - Single Source of Truth Data Import
// =============================================================================
//
// PURPOSE:
// Import critical business data from data/ssot folder:
// 1. Bahrain_Customer_Database_Clean.csv → CustomerMaster table
// 2. opportunities created 2025.xlsx → Opportunity entities
// 3. Payments to suppliers.xlsx → Payment tracking
// 4. <default division> Costing MasterFile.xlsx → Product costing
//
// PHILOSOPHY:
// - Batch inserts for performance (Williams batching!)
// - Comprehensive error handling and reporting
// - Idempotent (can run multiple times safely)
// - Detailed statistics for observability
//
// USAGE:
// ```go
// result, err := ImportAllSSOT(db, "./data/ssot")
// if err != nil {
//     log.Fatalf("Import failed: %v", err)
// }
// fmt.Printf("Imported: %+v\n", result)
// ```

// ImportResult contains statistics from the import operation
type ImportResult struct {
	// Customers
	CustomersTotal    int      `json:"customers_total"`
	CustomersImported int      `json:"customers_imported"`
	CustomersSkipped  int      `json:"customers_skipped"`
	CustomerErrors    []string `json:"customer_errors,omitempty"`

	// Opportunities
	OpportunitiesTotal    int      `json:"opportunities_total"`
	OpportunitiesImported int      `json:"opportunities_imported"`
	OpportunitiesSkipped  int      `json:"opportunities_skipped"`
	OpportunityErrors     []string `json:"opportunity_errors,omitempty"`

	// Payments
	PaymentsTotal    int      `json:"payments_total"`
	PaymentsImported int      `json:"payments_imported"`
	PaymentsSkipped  int      `json:"payments_skipped"`
	PaymentErrors    []string `json:"payment_errors,omitempty"`

	// Products
	ProductsTotal    int      `json:"products_total"`
	ProductsImported int      `json:"products_imported"`
	ProductsSkipped  int      `json:"products_skipped"`
	ProductErrors    []string `json:"product_errors,omitempty"`

	// Overall
	TotalRecords  int           `json:"total_records"`
	TotalImported int           `json:"total_imported"`
	Duration      time.Duration `json:"duration"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
}

// CustomerMaster is the database model (from database.go)
type CustomerMaster struct {
	ID           uint   `gorm:"primaryKey"`
	CustomerID   string `gorm:"uniqueIndex;size:50"`
	CustomerType string `gorm:"index;size:50"`
	BusinessName string `gorm:"index;size:255"`
	ShortCode    string `gorm:"size:10"`

	// Contact Information
	AddressLine1 string `gorm:"size:255"`
	AddressLine2 string `gorm:"size:255"`
	City         string `gorm:"index;size:100"`
	Country      string `gorm:"index;size:100"`
	PostalCode   string `gorm:"size:50"`
	TRN          string `gorm:"size:100"`

	// Business Details
	Industry      string `gorm:"index;size:100"`
	RelationYears int

	// Payment Profile
	PaymentGrade     string `gorm:"index;size:10"`
	PaymentTermsDays int
	AvgPaymentDays   float64
	DisputeCount     int

	// Financial Metrics
	TotalOrdersValue float64
	TotalOrdersCount int
	AvgOrderValue    float64
	LastOrderDate    *time.Time

	// Risk Flags
	HasABBCompetition  bool
	IsEmergencyOnly    bool
	IsCreditBlocked    bool
	RequiresPrepayment bool

	// Audit
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy string     `gorm:"size:100"`
	UpdatedBy string     `gorm:"size:100"`
	DeletedAt *time.Time `gorm:"index"`
}

func (CustomerMaster) TableName() string {
	return "customers"
}

// OpportunitySSOT represents opportunities from Excel
type OpportunitySSOT struct {
	ID            uint   `gorm:"primaryKey"`
	OpportunityID string `gorm:"uniqueIndex;size:100"`
	CustomerName  string `gorm:"index;size:255"`
	CustomerID    uint   `gorm:"index"`

	// Opportunity Details
	Title       string `gorm:"size:500"`
	Description string `gorm:"type:text"`
	ValueBHD    float64
	Stage       string  `gorm:"index;size:50"`
	Probability float64 // 0.0 - 1.0

	// Dates
	CreatedDate   time.Time `gorm:"index"`
	ExpectedClose time.Time `gorm:"index"`
	ActualClose   *time.Time

	// Product/Industry
	ProductType string `gorm:"index;size:100"`
	Industry    string `gorm:"index;size:100"`

	// Competition
	HasCompetition bool
	CompetitorName string `gorm:"size:255"`

	// Source tracking
	SourceFile string    `gorm:"size:500"`
	ImportedAt time.Time `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (OpportunitySSOT) TableName() string {
	return "opportunities_ssot"
}

// PaymentSSOT represents supplier payments from Excel
type PaymentSSOT struct {
	ID        uint   `gorm:"primaryKey"`
	PaymentID string `gorm:"uniqueIndex;size:100"`

	// Supplier Details
	SupplierName string `gorm:"index;size:255"`
	SupplierCode string `gorm:"index;size:50"`

	// Payment Details
	PaymentDate  time.Time `gorm:"index"`
	AmountBHD    float64
	Currency     string `gorm:"size:10"`
	ExchangeRate float64

	// Reference
	InvoiceNumber   string `gorm:"index;size:100"`
	ReferenceNumber string `gorm:"size:255"`
	PaymentMethod   string `gorm:"size:100"`

	// Categories
	Category string `gorm:"index;size:100"`
	Notes    string `gorm:"type:text"`

	// Source tracking
	SourceFile string    `gorm:"size:500"`
	ImportedAt time.Time `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (PaymentSSOT) TableName() string {
	return "payments_ssot"
}

// ProductCostingSSOT represents product costing from Excel
type ProductCostingSSOT struct {
	ID          uint   `gorm:"primaryKey"`
	ProductCode string `gorm:"index;size:100"`
	ProductName string `gorm:"index;size:255"`

	// Supplier
	SupplierCode string `gorm:"index;size:50"`
	SupplierName string `gorm:"size:255"`

	// Costing
	CostBHD       float64
	PriceBHD      float64
	MarginPercent float64

	// Product Details
	Category     string `gorm:"index;size:100"`
	Description  string `gorm:"type:text"`
	LeadTimeDays int

	// Competitive
	ABBEquivalent    string `gorm:"size:100"`
	CompetitiveNotes string `gorm:"type:text"`

	// Source tracking
	SourceFile string    `gorm:"size:500"`
	ImportedAt time.Time `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ProductCostingSSOT) TableName() string {
	return "products_costing_ssot"
}

// =============================================================================
// MAIN IMPORT FUNCTION
// =============================================================================

// ImportAllSSOT imports all SSOT data from the specified directory
func ImportAllSSOT(db *gorm.DB, dataDir string) (*ImportResult, error) {
	result := &ImportResult{
		StartTime: time.Now(),
	}

	// Auto-migrate all SSOT tables
	if err := db.AutoMigrate(
		&CustomerMaster{},
		&OpportunitySSOT{},
		&PaymentSSOT{},
		&ProductCostingSSOT{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	// Import customers from CSV
	customerFile := filepath.Join(dataDir, "Bahrain_Customer_Database_Clean.csv")
	if err := importCustomersFromCSV(db, customerFile, result); err != nil {
		return result, fmt.Errorf("customer import failed: %w", err)
	}

	// Import opportunities from Excel
	opportunityFile := filepath.Join(dataDir, "opportunities created 2025.xlsx")
	if _, err := os.Stat(opportunityFile); err == nil {
		if err := importOpportunitiesFromExcel(db, opportunityFile, result); err != nil {
			result.OpportunityErrors = append(result.OpportunityErrors, err.Error())
		}
	}

	// Import payments from Excel
	paymentFile := filepath.Join(dataDir, "Payments to suppliers.xlsx")
	if _, err := os.Stat(paymentFile); err == nil {
		if err := importPaymentsFromExcel(db, paymentFile, result); err != nil {
			result.PaymentErrors = append(result.PaymentErrors, err.Error())
		}
	}

	// Import product costing from Excel. The masterfile is named after the
	// deployment's default division (Wave 12: registry-driven, not a literal),
	// so an overlay with a different default division finds its own file.
	costingFile := filepath.Join(dataDir, overlay.Active().DefaultDivision()+" Costing MasterFile.xlsx")
	if _, err := os.Stat(costingFile); err == nil {
		if err := importProductCostingFromExcel(db, costingFile, result); err != nil {
			result.ProductErrors = append(result.ProductErrors, err.Error())
		}
	}

	// Calculate totals
	result.TotalRecords = result.CustomersTotal + result.OpportunitiesTotal +
		result.PaymentsTotal + result.ProductsTotal
	result.TotalImported = result.CustomersImported + result.OpportunitiesImported +
		result.PaymentsImported + result.ProductsImported

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// =============================================================================
// CUSTOMER IMPORT FROM CSV
// =============================================================================

func importCustomersFromCSV(db *gorm.DB, filename string, result *ImportResult) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Map column indices
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[col] = i
	}

	// Batch preparation (Williams batching: sqrt(n) * log2(n))
	// For ~311 customers: sqrt(311) * log2(311) ≈ 17.6 * 8.3 ≈ 146
	// Use batch size of 50 for safety
	const batchSize = 50
	var batch []CustomerMaster

	lineNum := 1 // Header is line 1

	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.CustomerErrors = append(result.CustomerErrors,
				fmt.Sprintf("Line %d: %v", lineNum, err))
			continue
		}

		result.CustomersTotal++

		// Parse customer record
		customer := CustomerMaster{
			BusinessName: getField(record, colMap, "business_account_name"),
			CustomerType: getField(record, colMap, "customer_type"),
			Industry:     getField(record, colMap, "industry"),
			City:         getField(record, colMap, "city"),
			Country:      getField(record, colMap, "country_iso"),
			PostalCode:   getField(record, colMap, "postal_code"),
			CreatedBy:    "SSOT_IMPORT",
			UpdatedBy:    "SSOT_IMPORT",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Generate customer ID from business name (simplified)
		customer.CustomerID = generateCustomerID(customer.BusinessName, customer.CustomerType)
		customer.ShortCode = extractShortCode(customer.CustomerType)

		// Skip empty business names
		if customer.BusinessName == "" {
			result.CustomersSkipped++
			continue
		}

		// Add to batch
		batch = append(batch, customer)

		// Insert batch when full
		if len(batch) >= batchSize {
			if err := insertCustomerBatch(db, batch, result); err != nil {
				result.CustomerErrors = append(result.CustomerErrors,
					fmt.Sprintf("Batch insert failed: %v", err))
			}
			batch = batch[:0] // Clear batch
		}
	}

	// Insert remaining records
	if len(batch) > 0 {
		if err := insertCustomerBatch(db, batch, result); err != nil {
			result.CustomerErrors = append(result.CustomerErrors,
				fmt.Sprintf("Final batch insert failed: %v", err))
		}
	}

	return nil
}

func insertCustomerBatch(db *gorm.DB, batch []CustomerMaster, result *ImportResult) error {
	// Use GORM's CreateInBatches for optimal batch insertion
	if err := db.CreateInBatches(batch, len(batch)).Error; err != nil {
		// If batch fails, try individual inserts (may be duplicates)
		for _, customer := range batch {
			if err := db.Create(&customer).Error; err != nil {
				result.CustomersSkipped++
			} else {
				result.CustomersImported++
			}
		}
		return err
	}

	result.CustomersImported += len(batch)
	return nil
}

// =============================================================================
// OPPORTUNITY IMPORT FROM EXCEL
// =============================================================================

func importOpportunitiesFromExcel(db *gorm.DB, filename string, result *ImportResult) error {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return fmt.Errorf("failed to open Excel: %w", err)
	}
	defer f.Close()

	// Get first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return fmt.Errorf("failed to read rows: %w", err)
	}

	if len(rows) < 2 {
		return fmt.Errorf("Excel file has no data rows")
	}

	// Assume first row is header
	header := rows[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	const batchSize = 50
	var batch []OpportunitySSOT

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		result.OpportunitiesTotal++

		// Parse opportunity
		opp := OpportunitySSOT{
			CustomerName: getFieldFromRow(row, colMap, "customer"),
			Title:        getFieldFromRow(row, colMap, "title"),
			Description:  getFieldFromRow(row, colMap, "description"),
			ValueBHD:     parseFloat(getFieldFromRow(row, colMap, "value")),
			Stage:        canonicalizeOpportunityStageSSOT(getFieldFromRow(row, colMap, "stage")),
			ProductType:  getFieldFromRow(row, colMap, "product"),
			Industry:     getFieldFromRow(row, colMap, "industry"),
			SourceFile:   filepath.Base(filename),
			ImportedAt:   time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Generate opportunity ID
		opp.OpportunityID = fmt.Sprintf("OPP2025-%04d", i)

		// Parse dates
		if dateStr := getFieldFromRow(row, colMap, "created"); dateStr != "" {
			if t, err := parseDate(dateStr); err == nil {
				opp.CreatedDate = t
			}
		}

		if dateStr := getFieldFromRow(row, colMap, "expected_close"); dateStr != "" {
			if t, err := parseDate(dateStr); err == nil {
				opp.ExpectedClose = t
			}
		}

		// Competition detection
		competitorName := getFieldFromRow(row, colMap, "competitor")
		if competitorName != "" {
			opp.HasCompetition = true
			opp.CompetitorName = competitorName
		}

		batch = append(batch, opp)

		if len(batch) >= batchSize {
			if err := db.CreateInBatches(batch, len(batch)).Error; err != nil {
				result.OpportunityErrors = append(result.OpportunityErrors, err.Error())
			} else {
				result.OpportunitiesImported += len(batch)
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := db.CreateInBatches(batch, len(batch)).Error; err != nil {
			result.OpportunityErrors = append(result.OpportunityErrors, err.Error())
		} else {
			result.OpportunitiesImported += len(batch)
		}
	}

	return nil
}

// =============================================================================
// PAYMENT IMPORT FROM EXCEL
// =============================================================================

func importPaymentsFromExcel(db *gorm.DB, filename string, result *ImportResult) error {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return fmt.Errorf("failed to open Excel: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("no sheets found")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return fmt.Errorf("failed to read rows: %w", err)
	}

	if len(rows) < 2 {
		return fmt.Errorf("no data rows")
	}

	header := rows[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	const batchSize = 50
	var batch []PaymentSSOT

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		result.PaymentsTotal++

		payment := PaymentSSOT{
			PaymentID:       fmt.Sprintf("PAY2025-%04d", i),
			SupplierName:    getFieldFromRow(row, colMap, "supplier"),
			SupplierCode:    getFieldFromRow(row, colMap, "supplier_code"),
			AmountBHD:       parseFloat(getFieldFromRow(row, colMap, "amount")),
			Currency:        getFieldFromRow(row, colMap, "currency"),
			InvoiceNumber:   getFieldFromRow(row, colMap, "invoice"),
			ReferenceNumber: getFieldFromRow(row, colMap, "reference"),
			PaymentMethod:   getFieldFromRow(row, colMap, "method"),
			Category:        getFieldFromRow(row, colMap, "category"),
			Notes:           getFieldFromRow(row, colMap, "notes"),
			SourceFile:      filepath.Base(filename),
			ImportedAt:      time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if dateStr := getFieldFromRow(row, colMap, "date"); dateStr != "" {
			if t, err := parseDate(dateStr); err == nil {
				payment.PaymentDate = t
			}
		}

		batch = append(batch, payment)

		if len(batch) >= batchSize {
			if err := db.CreateInBatches(batch, len(batch)).Error; err != nil {
				result.PaymentErrors = append(result.PaymentErrors, err.Error())
			} else {
				result.PaymentsImported += len(batch)
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := db.CreateInBatches(batch, len(batch)).Error; err != nil {
			result.PaymentErrors = append(result.PaymentErrors, err.Error())
		} else {
			result.PaymentsImported += len(batch)
		}
	}

	return nil
}

// =============================================================================
// PRODUCT COSTING IMPORT FROM EXCEL
// =============================================================================

func importProductCostingFromExcel(db *gorm.DB, filename string, result *ImportResult) error {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return fmt.Errorf("failed to open Excel: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("no sheets found")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return fmt.Errorf("failed to read rows: %w", err)
	}

	if len(rows) < 2 {
		return fmt.Errorf("no data rows")
	}

	header := rows[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	const batchSize = 50
	var batch []ProductCostingSSOT

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		result.ProductsTotal++

		product := ProductCostingSSOT{
			ProductCode:      getFieldFromRow(row, colMap, "product_code"),
			ProductName:      getFieldFromRow(row, colMap, "product_name"),
			SupplierCode:     getFieldFromRow(row, colMap, "supplier_code"),
			SupplierName:     getFieldFromRow(row, colMap, "supplier"),
			CostBHD:          parseFloat(getFieldFromRow(row, colMap, "cost")),
			PriceBHD:         parseFloat(getFieldFromRow(row, colMap, "price")),
			MarginPercent:    parseFloat(getFieldFromRow(row, colMap, "margin")),
			Category:         getFieldFromRow(row, colMap, "category"),
			Description:      getFieldFromRow(row, colMap, "description"),
			LeadTimeDays:     parseInt(getFieldFromRow(row, colMap, "lead_time")),
			ABBEquivalent:    getFieldFromRow(row, colMap, "abb_equivalent"),
			CompetitiveNotes: getFieldFromRow(row, colMap, "competitive_notes"),
			SourceFile:       filepath.Base(filename),
			ImportedAt:       time.Now(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		batch = append(batch, product)

		if len(batch) >= batchSize {
			if err := db.CreateInBatches(batch, len(batch)).Error; err != nil {
				result.ProductErrors = append(result.ProductErrors, err.Error())
			} else {
				result.ProductsImported += len(batch)
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := db.CreateInBatches(batch, len(batch)).Error; err != nil {
			result.ProductErrors = append(result.ProductErrors, err.Error())
		} else {
			result.ProductsImported += len(batch)
		}
	}

	return nil
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

func getField(record []string, colMap map[string]int, fieldName string) string {
	if idx, ok := colMap[fieldName]; ok && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}

func getFieldFromRow(row []string, colMap map[string]int, fieldName string) string {
	if idx, ok := colMap[fieldName]; ok && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	val, _ := strconv.Atoi(s)
	return val
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	// Try common date formats
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
		"2006/01/02",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

func generateCustomerID(businessName, customerType string) string {
	// Extract prefix from customer type
	prefix := extractShortCode(customerType)

	// Generate hash from business name
	hash := 0
	for _, c := range businessName {
		hash = (hash*31 + int(c)) % 1000
	}

	return fmt.Sprintf("%s%03d", prefix, hash)
}

func extractShortCode(customerType string) string {
	customerType = strings.ToUpper(strings.TrimSpace(customerType))

	switch {
	case strings.Contains(customerType, "END CUSTOMER"):
		return "EC"
	case strings.Contains(customerType, "ENGINEERING"):
		return "EG"
	case strings.Contains(customerType, "SYSTEM INTEGRATOR"):
		return "SI"
	case strings.Contains(customerType, "INTERNATIONAL RESELLER"):
		return "IR"
	case strings.Contains(customerType, "NATIONAL RESELLER"):
		return "NR"
	case strings.Contains(customerType, "PLANT BUILDER"):
		return "PB"
	case strings.Contains(customerType, "SERVICE PROVIDER"):
		return "SP"
	case strings.Contains(customerType, "CONSULTANT"):
		return "CO"
	case strings.Contains(customerType, "OEM"):
		return "OE"
	default:
		return "GC" // General Customer
	}
}
