package main

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"ph_holdings_app/pkg/documents/excel"
)

// Note: EHBasket, EHHeader, EHItem, EHProduct, etc. are already defined in eh_parser.go
// We reuse those existing types

// Customer name mapping from short codes to full business names
var customerNameMap = map[string]string{
	"DPC":                 "Delta Petrochemicals",
	"GSC":                 "Gulf Smelting Co.",
	"NPC":                 "National Petroleum Co.",
	"NGA":                 "North Grid Authority",
	"CJV":                 "Coastal JV W.L.L.",
	"ICG":                 "Intercon Group",
	"MDY":                 "Meadow Dairy",
	"VWT":                 "AquaPure Technologies",
	"AQUAPURE":            "AquaPure Energy",
	"AQUAPURE ENERGY":     "AquaPure Energy",
	"SHI":                 "Sandhill Industrial",
	"PNM":                 "Pinnacle O&M",
	"XNT":                 "Xenon Tech",
	"BLUEWAVE":            "BlueWave Marine",
	"AQUA TECH":           "Aqua Tech Demo",
	"CSW":                 "Cascade Water",
	"ALW":                 "Alder Works",
	"CRT":                 "Crescent Trading",
	"ZNT":                 "Zenith Trading",
	"VERTEX":              "Vertex Energy",
	"AIC":                 "Axis Controls",
	"LGS":                 "Logica Systems",
	"BEACON CONTROLS WLL": "Beacon Controls W.L.L.",
	"AHS":                 "Beacon Controls W.L.L.",
}

// Product type code mapping
var productTypeMap = map[string]string{
	"TW":  "Thermowell",
	"FIT": "Flow Instrument",
	"LIT": "Level Instrument",
	"TIT": "Temperature Instrument",
	"PIT": "Pressure Instrument",
	"AIT": "Analytical Instrument",
	"SP":  "Spare Parts",
	"DSP": "Display",
	"BTU": "BTU Meter",
}

// Import2026BusinessData imports business data from 2026 offers folder and bank statements
func (a *App) Import2026BusinessData(offersPath string) map[string]any {
	// Check permission
	if err := a.requirePermission("*"); err != nil {
		return map[string]any{
			"success": false,
			"error":   "Admin permission required",
		}
	}

	log.Printf("Starting 2026 business data import from: %s", offersPath)

	stats := map[string]any{
		"offers_created":      0,
		"offer_items_created": 0,
		"customers_created":   0,
		"customers_matched":   0,
		"bank_statements":     0,
		"bank_transactions":   0,
		"bank_accounts_added": 0,
		"errors":              []string{},
	}

	// 1. Import offers from folder structure
	if err := a.importOffersFromFolders(offersPath, stats); err != nil {
		stats["errors"] = append(stats["errors"].([]string), fmt.Sprintf("Offers import failed: %v", err))
	}

	// 2. Import bank statement
	bankStatementPath := `C:\Users\developer\Documents\Demo_Business_Docs\Bank_Statements\DEMO BANK 1-1-26 TO 31-1-26.csv`
	if err := a.importBankStatement(bankStatementPath, stats); err != nil {
		stats["errors"] = append(stats["errors"].([]string), fmt.Sprintf("Bank statement import failed: %v", err))
	}

	// 3. Seed additional bank accounts
	if err := a.SeedAdditionalBankAccounts(); err != nil {
		stats["errors"] = append(stats["errors"].([]string), fmt.Sprintf("Bank accounts seed failed: %v", err))
	} else {
		stats["bank_accounts_added"] = 2 // Demo Bank C + Demo Bank A Euro
	}

	stats["success"] = len(stats["errors"].([]string)) == 0
	log.Printf("2026 data import completed: %+v", stats)

	return stats
}

// importOffersFromFolders scans the offers folder and creates Offer records
func (a *App) importOffersFromFolders(offersPath string, stats map[string]any) error {
	entries, err := os.ReadDir(offersPath)
	if err != nil {
		return fmt.Errorf("failed to read offers directory: %w", err)
	}

	// Regex patterns for folder name parsing
	// Format: "NN-26 CUSTOMER PRODUCT" or "NN - 26 CUSTOMER PRODUCT"
	offerPattern := regexp.MustCompile(`^(\d+)\s*-?\s*26\s+(.+)$`)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		folderName := entry.Name()
		matches := offerPattern.FindStringSubmatch(folderName)
		if len(matches) < 3 {
			log.Printf("Skipping folder (doesn't match offer pattern): %s", folderName)
			continue
		}

		offerNum := matches[1]
		customerProduct := strings.TrimSpace(matches[2])

		// Parse customer and product from the rest of the name
		customerName, productType := a.parseCustomerAndProduct(customerProduct)
		if customerName == "" {
			log.Printf("Skipping folder (couldn't extract customer): %s", folderName)
			continue
		}

		// Map short code to full business name
		fullCustomerName := a.mapCustomerName(customerName)

		// Check for OFFER or RFQ subfolder to determine stage
		folderPath := filepath.Join(offersPath, folderName)
		stage := a.determineOfferStage(folderPath)

		// Get folder modification time for quotation date
		info, err := entry.Info()
		quotationDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC) // Default to Jan 15, 2026
		if err == nil {
			quotationDate = info.ModTime()
		}

		// Create or find customer
		customer, err := a.findOrCreateCustomer(fullCustomerName, customerName)
		if err != nil {
			log.Printf("Failed to find/create customer %s: %v", fullCustomerName, err)
			continue
		}

		if customer.CreatedAt.After(time.Now().Add(-1 * time.Second)) {
			stats["customers_created"] = stats["customers_created"].(int) + 1
		} else {
			stats["customers_matched"] = stats["customers_matched"].(int) + 1
		}

		// Create offer
		offerNumber := fmt.Sprintf("%s-26", offerNum)
		offer := Offer{}
		result := a.db.Where("offer_number = ?", offerNumber).FirstOrCreate(&offer, Offer{
			Base: Base{
				ID:        uuid.New().String(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			OfferNumber:     offerNumber,
			RevisionNumber:  0,
			CustomerID:      customer.ID,
			CustomerName:    fullCustomerName,
			QuotationDate:   quotationDate,
			ValidityDate:    quotationDate.AddDate(0, 1, 0), // 1 month validity
			TotalValueBHD:   0,                              // Will be calculated from items
			EstimatedMargin: 0,
			Stage:           stage,
		})

		if result.Error != nil {
			log.Printf("Failed to create offer %s: %v", offerNumber, result.Error)
			stats["errors"] = append(stats["errors"].([]string), fmt.Sprintf("Offer %s: %v", offerNumber, result.Error))
			continue
		}

		if result.RowsAffected > 0 {
			stats["offers_created"] = stats["offers_created"].(int) + 1
			log.Printf("Created offer: %s for customer %s (stage: %s, product: %s)", offerNumber, fullCustomerName, stage, productType)
		}

		// Parse Rhine XML files if present
		itemsCount := a.parseEHXMLFiles(folderPath, offer.ID, stats)

		// Update offer total if items were added
		if itemsCount > 0 {
			a.updateOfferTotal(offer.ID)
		}
	}

	return nil
}

// parseCustomerAndProduct extracts customer name and product type from folder name remainder
func (a *App) parseCustomerAndProduct(customerProduct string) (string, string) {
	parts := strings.Fields(customerProduct)
	if len(parts) == 0 {
		return "", ""
	}

	// Last part is often the product code (TW, FIT, etc.)
	productCode := ""
	customerParts := parts

	if len(parts) > 1 {
		lastPart := strings.ToUpper(parts[len(parts)-1])
		if _, exists := productTypeMap[lastPart]; exists {
			productCode = lastPart
			customerParts = parts[:len(parts)-1]
		}
	}

	customerName := strings.Join(customerParts, " ")
	return customerName, productCode
}

// mapCustomerName converts short codes to full business names
func (a *App) mapCustomerName(shortCode string) string {
	upperCode := strings.ToUpper(strings.TrimSpace(shortCode))

	// Try exact match first
	if fullName, exists := customerNameMap[upperCode]; exists {
		return fullName
	}

	// Try partial match (e.g., "AQUAPURE SOMETHING" → "AquaPure Energy")
	for code, fullName := range customerNameMap {
		if strings.Contains(upperCode, code) {
			return fullName
		}
	}

	// Return original if no mapping found
	return shortCode
}

// determineOfferStage checks for EXECUTION, OFFER, or RFQ subfolder
func (a *App) determineOfferStage(folderPath string) string {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return "RFQ" // Default to RFQ if can't read
	}

	hasExecution := false
	hasOffer := false

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.ToUpper(entry.Name())
		if strings.Contains(name, "EXECUTION") {
			hasExecution = true
		}
		if strings.Contains(name, "OFFER") {
			hasOffer = true
		}
	}

	if hasExecution {
		return "Won" // Confirmed order with PO/DN/Invoice
	}
	if hasOffer {
		return "Quoted"
	}
	return "RFQ"
}

// findOrCreateCustomer finds existing customer or creates new one
func (a *App) findOrCreateCustomer(businessName, shortCode string) (*CustomerMaster, error) {
	customer := &CustomerMaster{}

	// Try to find existing customer by business name (case-insensitive LIKE)
	result := a.db.Where("LOWER(business_name) LIKE ?", "%"+strings.ToLower(businessName)+"%").First(customer)
	if result.Error == nil {
		return customer, nil
	}

	// Create new customer
	customer = &CustomerMaster{
		Base: Base{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		CustomerID:   fmt.Sprintf("CUST-%s", strings.ToUpper(strings.ReplaceAll(shortCode, " ", "-"))),
		BusinessName: businessName,
		ShortCode:    shortCode,
		Industry:     "Process Instrumentation", // Default industry
		Country:      "Bahrain",                 // Default country
		City:         "Manama",                  // Default city
	}

	if err := a.db.Create(customer).Error; err != nil {
		return nil, err
	}

	log.Printf("Created new customer: %s (%s)", businessName, customer.CustomerID)
	return customer, nil
}

// parseEHXMLFiles looks for Rhine XML files in offer subfolders and creates OfferItems
func (a *App) parseEHXMLFiles(offerFolderPath string, offerID string, stats map[string]any) int {
	itemsCount := 0

	// Walk through subfolders looking for XML files
	err := filepath.Walk(offerFolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(strings.ToLower(info.Name()), ".xml") {
			return nil
		}

		// Try to parse as Rhine basket XML
		count, parseErr := a.parseEHBasketXML(path, offerID)
		if parseErr != nil {
			log.Printf("Failed to parse XML %s: %v", path, parseErr)
			return nil
		}

		itemsCount += count
		stats["offer_items_created"] = stats["offer_items_created"].(int) + count
		return nil
	})

	if err != nil {
		log.Printf("Error walking offer folder %s: %v", offerFolderPath, err)
	}

	return itemsCount
}

// parseEHBasketXML parses Rhine basket XML format and creates OfferItems
func (a *App) parseEHBasketXML(xmlPath string, offerID string) (int, error) {
	file, err := os.Open(xmlPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return 0, err
	}

	// Try to parse with namespace handling (Rhine Instruments uses bas:basket, bas:item etc)
	// First attempt: direct parsing (Go XML handles namespaces automatically in many cases)
	var basket EHBasket
	if err := xml.Unmarshal(data, &basket); err != nil {
		// If that fails, it might not be an Rhine basket XML
		return 0, fmt.Errorf("not a valid Rhine basket XML: %w", err)
	}

	if len(basket.Items) == 0 {
		log.Printf("No items found in XML: %s", xmlPath)
		return 0, nil
	}

	log.Printf("Parsing Rhine XML: %s (%d items)", xmlPath, len(basket.Items))

	// Create OfferItems from XML data
	// EHItem structure: Product (OrderCode, Quantity, Texts.ShortDescription), ItemPricing (UnitSalesPrice)
	for i, item := range basket.Items {
		// Extract quantity (convert int to float64)
		quantity := float64(item.Product.Quantity.Value)

		// Extract unit price in EUR, convert to BHD using the overlay's
		// single-source-of-truth FX rate (same rate the live costing path uses).
		unitPriceEUR := item.ItemPricing.UnitSalesPrice.Value
		unitPriceBHD := unitPriceEUR * activeOverlay.ExchangeRateToBase("EUR")

		// Extract description
		description := item.Product.Texts.ShortDescription

		offerItem := OfferItem{
			Base: Base{
				ID:        uuid.New().String(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			OfferID:     offerID,
			LineNumber:  i + 1,
			ProductCode: item.Product.OrderCode,
			Model:       item.Product.OrderCodeLong,
			Description: description,
			Quantity:    quantity,
			UnitPrice:   unitPriceBHD,
			Equipment:   description,
			Currency:    "EUR", // Source currency
			FOB:         unitPriceEUR * quantity,
		}

		if err := a.db.Create(&offerItem).Error; err != nil {
			log.Printf("Failed to create offer item from XML: %v", err)
			continue
		}
	}

	return len(basket.Items), nil
}

// updateOfferTotal recalculates and updates the offer total from its items
func (a *App) updateOfferTotal(offerID string) {
	var total float64
	a.db.Model(&OfferItem{}).Where("offer_id = ?", offerID).
		Select("COALESCE(SUM(quantity * unit_price_bhd), 0)").Scan(&total)

	a.db.Model(&Offer{}).Where("id = ?", offerID).Update("total_value_bhd", total)
}

// importBankStatement imports the demo bank statement CSV
// Demo CSV format: metadata rows (Client, IBAN, etc.), then "Posting Date,Value Date,FT Reference,Description,Amount,Balance,"
// Amount uses "(-) " prefix for debits
func (a *App) importBankStatement(csvPath string, stats map[string]any) error {
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("failed to open CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable field count
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	log.Printf("Importing bank statement from CSV (%d rows)", len(records))

	// Find or create demo bank account (try both table types)
	var bankAccountID string
	bankAccount := &BankAccount{}
	if err := a.db.Where("id = ?", "bank-gamma").First(bankAccount).Error; err == nil {
		bankAccountID = bankAccount.ID
	} else {
		// Try CompanyBankAccount table
		companyAccount := &CompanyBankAccount{}
		if err := a.db.Where("id = ?", "bank-gamma").First(companyAccount).Error; err == nil {
			bankAccountID = companyAccount.ID
		} else {
			// Create the bank account
			newAccount := BankAccount{
				Base: Base{
					ID:        "bank-gamma",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				BankName:      "Demo Bank A",
				AccountNumber: "00DEMO0000000001",
				AccountName:   "Acme Instrumentation W.L.L.",
				Currency:      "BHD",
				IsActive:      true,
			}
			a.db.Create(&newAccount)
			bankAccountID = "bank-gamma"
		}
	}

	// Parse the demo bank CSV format:
	// Rows 1-14: metadata (Client Name, IBAN, From/To dates, totals)
	// Row 15: Column headers "Posting Date,Value Date,FT Reference,Description,Amount,Balance,"
	// Row 16: Opening Balance line
	// Rows 17+: Transaction rows
	// Last row: Closing Balance line

	// Extract metadata from header rows
	var openingBalance, closingBalance float64
	var periodStart, periodEnd time.Time

	// Find the header row and extract metadata
	headerRowIdx := -1
	for i, record := range records {
		if len(record) == 0 {
			continue
		}
		firstCol := strings.TrimSpace(record[0])

		// Extract metadata
		if firstCol == "From" && len(record) > 1 {
			if t, err := a.parseBankDate(strings.TrimSpace(record[1])); err == nil {
				periodStart = t
			}
		}
		if firstCol == "To" && len(record) > 1 {
			if t, err := a.parseBankDate(strings.TrimSpace(record[1])); err == nil {
				periodEnd = t
			}
		}

		// Find the header row (Posting Date, Value Date, ...)
		if strings.Contains(firstCol, "Posting Date") || strings.Contains(firstCol, "posting date") {
			headerRowIdx = i
			break
		}
	}

	if headerRowIdx == -1 {
		return fmt.Errorf("could not find header row in CSV")
	}

	// Check for existing statement
	var existingCount int64
	a.db.Model(&BankStatement{}).Where("bank_account_id = ? AND period_start = ?", bankAccountID, periodStart).Count(&existingCount)
	if existingCount > 0 {
		log.Printf("Bank statement already exists for this period, skipping")
		return nil
	}

	// Create bank statement
	statementNumber := fmt.Sprintf("ALSALAM-%s", periodStart.Format("JAN-2006"))
	statement := BankStatement{
		Base: Base{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		BankAccountID:   bankAccountID,
		StatementNumber: statementNumber,
		StatementDate:   periodEnd,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		Currency:        "BHD",
		Status:          "Imported",
		ImportedFrom:    csvPath,
		ImportMethod:    "CSV",
		Division:        activeOverlay.DefaultDivision(),
	}

	var totalDebits, totalCredits float64
	var debitCount, creditCount int
	var lines []BankStatementLine
	lineNum := 0

	// Parse transaction rows after header
	for i := headerRowIdx + 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 5 {
			continue
		}

		dateStr := strings.TrimSpace(record[0])
		description := ""
		if len(record) > 3 {
			description = strings.TrimSpace(record[3])
		}

		// Handle Opening/Closing Balance rows (no date, just description + balance)
		if dateStr == "" && strings.Contains(description, "Opening Balance") {
			if len(record) > 5 {
				openingBalance = a.parseAmount(record[5])
			}
			continue
		}
		if dateStr == "" && strings.Contains(description, "Closing Balance") {
			if len(record) > 5 {
				closingBalance = a.parseAmount(record[5])
			}
			continue
		}

		// Skip empty rows
		if dateStr == "" {
			continue
		}

		// Parse transaction date
		txnDate, err := a.parseBankDate(dateStr)
		if err != nil {
			continue
		}

		// Parse value date
		valueDate := txnDate
		if len(record) > 1 {
			if vd, err := a.parseBankDate(strings.TrimSpace(record[1])); err == nil {
				valueDate = vd
			}
		}

		// FT Reference
		reference := ""
		if len(record) > 2 {
			reference = strings.TrimSpace(record[2])
		}

		// Parse amount - demo bank uses "(-) 20.000" for debits, plain number for credits
		var debit, credit float64
		if len(record) > 4 {
			amountStr := strings.TrimSpace(record[4])
			if strings.HasPrefix(amountStr, "(-)") || strings.HasPrefix(amountStr, "(-) ") {
				// Debit
				amountStr = strings.TrimPrefix(amountStr, "(-)")
				amountStr = strings.TrimSpace(amountStr)
				debit = a.parseAmount(amountStr)
			} else if amountStr != "" {
				credit = a.parseAmount(amountStr)
			}
		}

		// Parse balance
		var balance float64
		if len(record) > 5 {
			balance = a.parseAmount(record[5])
		}

		lineNum++
		line := BankStatementLine{
			Base: Base{
				ID:        uuid.New().String(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			BankStatementID: statement.ID,
			LineNumber:      lineNum,
			TransactionDate: txnDate,
			ValueDate:       valueDate,
			Description:     description,
			Reference:       reference,
			Debit:           debit,
			Credit:          credit,
			Balance:         balance,
		}
		lines = append(lines, line)

		if debit > 0 {
			totalDebits += debit
			debitCount++
		}
		if credit > 0 {
			totalCredits += credit
			creditCount++
		}
	}

	statement.OpeningBalance = openingBalance
	statement.ClosingBalance = closingBalance
	statement.TotalDebits = totalDebits
	statement.TotalCredits = totalCredits
	statement.DebitCount = debitCount
	statement.CreditCount = creditCount

	// Save statement
	if err := a.db.Create(&statement).Error; err != nil {
		return fmt.Errorf("failed to create bank statement: %w", err)
	}

	// Save lines
	for _, line := range lines {
		if err := a.db.Create(&line).Error; err != nil {
			log.Printf("Failed to create bank statement line: %v", err)
		}
	}

	stats["bank_statements"] = stats["bank_statements"].(int) + 1
	stats["bank_transactions"] = len(lines)
	log.Printf("Imported bank statement: %s (opening: %.3f, closing: %.3f, %d debits, %d credits)",
		statementNumber, openingBalance, closingBalance, debitCount, creditCount)

	return nil
}

// parseBankDate parses dates in bank statement format like "25 JAN 2026", "01 JAN 2026"
func (a *App) parseBankDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	formats := []string{
		"2 Jan 2006",
		"02 Jan 2006",
		"2 JAN 2006",
		"02 JAN 2006",
		"2 January 2006",
		"02 January 2006",
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", dateStr)
}

// findColumnIndex finds column index by name (case-insensitive, tries multiple
// variations). Delegates to the pkg/documents/excel engine's COLUMN-priority
// scan — the semantics this caller always had (Wave 3 B.3).
func (a *App) findColumnIndex(header []string, possibleNames []string) int {
	return excel.FindInHeader(header, possibleNames...)
}

// parseDate parses various date formats commonly found in bank CSVs
func (a *App) parseDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}

	// Try common formats
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
		"2-Jan-2006",
		"02-Jan-2006",
		"2-January-2006",
		"02-January-2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"2006/01/02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date format: %s", dateStr)
}

// parseAmount parses amount string (handles commas, spaces, currency symbols)
func (a *App) parseAmount(amountStr string) float64 {
	// Clean string
	amountStr = strings.TrimSpace(amountStr)
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amountStr = strings.ReplaceAll(amountStr, " ", "")
	amountStr = strings.ReplaceAll(amountStr, "BHD", "")
	amountStr = strings.ReplaceAll(amountStr, "BD", "")

	if amountStr == "" || amountStr == "-" {
		return 0
	}

	// Parse float
	val, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0
	}

	return val
}

// SeedAdditionalBankAccounts adds demo BHD and EUR accounts
func (a *App) SeedAdditionalBankAccounts() error {
	// Mission I (I-11): bound mutator writes bank-account master rows — gated.
	if err := a.requirePermission("finance:create"); err != nil {
		return err
	}
	log.Printf("Seeding additional bank accounts (Demo Bank C, Demo Bank A Euro)")

	accounts := []BankAccount{
		{
			Base: Base{
				ID:        "bank-kfh",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			BankName:       "Demo Bank C",
			AccountNumber:  "00DEMO0000000003", // Placeholder demo account
			AccountName:    "Acme Instrumentation W.L.L.",
			Currency:       "BHD",
			CurrentBalance: 0,
			IsActive:       true,
		},
		{
			Base: Base{
				ID:        "bank-alpha-euro",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			BankName:       "Demo Bank A - Euro Call Account",
			AccountNumber:  "00DEMO0000000004", // Placeholder demo account
			AccountName:    "Acme Instrumentation W.L.L.",
			Currency:       "EUR",
			CurrentBalance: 0,
			IsActive:       true,
		},
	}

	for _, account := range accounts {
		// Use FirstOrCreate to avoid duplicates
		existingAccount := BankAccount{}
		result := a.db.Where("id = ?", account.ID).FirstOrCreate(&existingAccount, account)
		if result.Error != nil {
			log.Printf("Failed to create bank account %s: %v", account.BankName, result.Error)
			continue
		}

		if result.RowsAffected > 0 {
			log.Printf("Created bank account: %s (%s)", account.BankName, account.ID)
		} else {
			log.Printf("Bank account already exists: %s", account.BankName)
		}
	}

	return nil
}
