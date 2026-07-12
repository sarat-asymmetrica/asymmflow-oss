package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"ph_holdings_app/pkg/documents/excel"
)

// TallyImportResult tracks import operation results
type TallyImportResult struct {
	TotalRows    int      `json:"total_rows"`
	Imported     int      `json:"imported"`
	Duplicates   int      `json:"duplicates"`
	Errors       int      `json:"errors"`
	ErrorDetails []string `json:"error_details,omitempty"`
	BatchID      string   `json:"batch_id"`
}

// ImportTallyInvoices imports invoices from Tally Excel export
func (a *App) ImportTallyInvoices(year int) (*TallyImportResult, error) {
	if err := a.requirePermission("data:import"); err != nil {
		return nil, err
	}

	result := &TallyImportResult{
		BatchID: uuid.New().String(),
	}

	// Try multiple filename variations across known data directories
	dataDirs := []string{"Data_for_database/Tally_data", "Data_for_database", "data/ssot"}
	var possibleFiles []string
	for _, baseDir := range dataDirs {
		possibleFiles = append(possibleFiles,
			filepath.Join(baseDir, fmt.Sprintf("invoices %d.xlsx", year)),
			filepath.Join(baseDir, fmt.Sprintf("invoices%d.xlsx", year)),
			filepath.Join(baseDir, fmt.Sprintf("Invoices %d.xlsx", year)),
			filepath.Join(baseDir, fmt.Sprintf("Invoices%d.xlsx", year)),
		)
	}

	var f *excelize.File
	var err error
	var foundFile string

	for _, filePath := range possibleFiles {
		f, err = excelize.OpenFile(filePath)
		if err == nil {
			foundFile = filePath
			break
		}
	}

	if f == nil {
		return nil, fmt.Errorf("could not find invoices file for year %d. Expected one of: %v", year, possibleFiles)
	}
	defer f.Close()

	log.Printf("Importing Tally invoices from: %s", foundFile)

	// Get all rows from Sheet1
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, fmt.Errorf("failed to read Sheet1: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("file has no data rows (only header or empty)")
	}

	// Parse header to find column indices
	header := rows[0]
	colMap := map[string]int(excel.IndexHeader(header))

	// Required columns (flexible matching)
	dateCol := findColumn(colMap, "date", "invoice date", "date of invoice")
	invoiceCol := findColumn(colMap, "invoice no", "invoice number", "invoice #", "inv no")
	customerCol := findColumn(colMap, "customer", "customer name", "party")
	amountCol := findColumn(colMap, "amount", "total", "invoice amount", "value")
	currencyCol := findColumn(colMap, "currency", "curr")

	if dateCol == -1 || invoiceCol == -1 || customerCol == -1 || amountCol == -1 {
		return nil, fmt.Errorf("missing required columns. Found headers: %v", header)
	}

	// Process data rows inside a transaction for atomicity
	txErr := a.db.Transaction(func(tx *gorm.DB) error {
		for i, row := range rows[1:] {
			result.TotalRows++

			if len(row) < len(header) {
				// Pad short rows with empty strings
				padded := make([]string, len(header))
				copy(padded, row)
				row = padded
			}

			invoiceNo := strings.TrimSpace(row[invoiceCol])
			if invoiceNo == "" {
				result.Errors++
				if len(result.ErrorDetails) < 100 {
					result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: Empty invoice number", i+2))
				} else if len(result.ErrorDetails) == 100 {
					result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
				}
				continue
			}

			// Check for duplicate
			var existingCount int64
			checkErr := tx.Model(&TallyInvoiceImport{}).
				Where("invoice_number = ? AND year = ?", invoiceNo, year).
				Count(&existingCount).Error
			if checkErr == nil && existingCount > 0 {
				result.Duplicates++
				continue
			}

			customerName := strings.TrimSpace(row[customerCol])
			customerID := a.matchCustomerByName(customerName)

			dateStr := strings.TrimSpace(row[dateCol])
			invoiceDate, parseErr := a.parseExcelDate(dateStr)
			if parseErr != nil {
				result.Errors++
				if len(result.ErrorDetails) < 100 {
					result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: invalid date '%s'", i+2, dateStr))
				} else if len(result.ErrorDetails) == 100 {
					result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
				}
				continue
			}

			amountStr := strings.TrimSpace(row[amountCol])
			var amount float64
			fmt.Sscanf(strings.ReplaceAll(amountStr, ",", ""), "%f", &amount)

			// Validate amount is reasonable (skip NaN and extreme values)
			if amount < 0 || amount > 100000000 {
				result.Errors++
				if len(result.ErrorDetails) < 100 {
					result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: invalid amount '%s'", i+2, amountStr))
				}
				continue
			}

			currency := "BHD"
			if currencyCol != -1 && currencyCol < len(row) {
				curr := strings.TrimSpace(row[currencyCol])
				if curr != "" {
					currency = strings.ToUpper(curr)
				}
			}

			tallyInvoice := TallyInvoiceImport{
				Year:              year,
				InvoiceNumber:     invoiceNo,
				InvoiceDate:       invoiceDate,
				CustomerName:      customerName,
				MatchedCustomerID: customerID,
				Amount:            amount,
				Currency:          currency,
				ImportBatch:       result.BatchID,
			}

			if err := tx.Create(&tallyInvoice).Error; err != nil {
				result.Errors++
				if len(result.ErrorDetails) < 100 {
					result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: %v", i+2, err))
				} else if len(result.ErrorDetails) == 100 {
					result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
				}
			} else {
				result.Imported++
			}
		}
		return nil // Commit all rows
	})

	if txErr != nil {
		return nil, fmt.Errorf("import transaction failed: %w", txErr)
	}

	log.Printf("Tally invoices import complete: %d imported, %d duplicates, %d errors out of %d rows",
		result.Imported, result.Duplicates, result.Errors, result.TotalRows)

	return result, nil
}

// ImportTallyPurchases imports purchases from Tally Excel export
func (a *App) ImportTallyPurchases(year int) (*TallyImportResult, error) {
	if err := a.requirePermission("data:import"); err != nil {
		return nil, err
	}

	result := &TallyImportResult{
		BatchID: uuid.New().String(),
	}

	dataDirs := []string{"Data_for_database/Tally_data", "Data_for_database", "data/ssot"}
	var possibleFiles []string
	for _, baseDir := range dataDirs {
		possibleFiles = append(possibleFiles,
			filepath.Join(baseDir, fmt.Sprintf("purchase %d.xlsx", year)),
			filepath.Join(baseDir, fmt.Sprintf("purchase%d.xlsx", year)),
			filepath.Join(baseDir, fmt.Sprintf("Purchase %d.xlsx", year)),
			filepath.Join(baseDir, fmt.Sprintf("purchases %d.xlsx", year)),
			filepath.Join(baseDir, fmt.Sprintf("Purchases %d.xlsx", year)),
		)
	}

	var f *excelize.File
	var err error
	var foundFile string

	for _, filePath := range possibleFiles {
		f, err = excelize.OpenFile(filePath)
		if err == nil {
			foundFile = filePath
			break
		}
	}

	if f == nil {
		return nil, fmt.Errorf("could not find purchases file for year %d. Expected one of: %v", year, possibleFiles)
	}
	defer f.Close()

	log.Printf("Importing Tally purchases from: %s", foundFile)

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, fmt.Errorf("failed to read Sheet1: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("file has no data rows")
	}

	header := rows[0]
	colMap := map[string]int(excel.IndexHeader(header))

	dateCol := findColumn(colMap, "date", "purchase date", "invoice date")
	invoiceCol := findColumn(colMap, "invoice no", "invoice number", "bill no", "voucher no")
	supplierCol := findColumn(colMap, "supplier", "supplier name", "party", "vendor")
	amountCol := findColumn(colMap, "amount", "total", "invoice amount", "value")
	currencyCol := findColumn(colMap, "currency", "curr")

	if dateCol == -1 || invoiceCol == -1 || supplierCol == -1 || amountCol == -1 {
		return nil, fmt.Errorf("missing required columns. Found headers: %v", header)
	}

	txErr := a.db.Transaction(func(tx *gorm.DB) error {
		for i, row := range rows[1:] {
			result.TotalRows++

			if len(row) < len(header) {
				padded := make([]string, len(header))
				copy(padded, row)
				row = padded
			}

			invoiceNo := strings.TrimSpace(row[invoiceCol])
			if invoiceNo == "" {
				result.Errors++
				if len(result.ErrorDetails) < 100 {
					result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: Empty invoice number", i+2))
				} else if len(result.ErrorDetails) == 100 {
					result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
				}
				continue
			}

			// Check duplicate
			var existingCount int64
			checkErr := tx.Model(&TallyPurchaseImport{}).
				Where("invoice_number = ? AND year = ?", invoiceNo, year).
				Count(&existingCount).Error
			if checkErr == nil && existingCount > 0 {
				result.Duplicates++
				continue
			}

			supplierName := strings.TrimSpace(row[supplierCol])
			supplierID := a.matchSupplierByName(supplierName)

			dateStr := strings.TrimSpace(row[dateCol])
			invoiceDate, parseErr := a.parseExcelDate(dateStr)
			if parseErr != nil {
				result.Errors++
				if len(result.ErrorDetails) < 100 {
					result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: invalid date '%s'", i+2, dateStr))
				} else if len(result.ErrorDetails) == 100 {
					result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
				}
				continue
			}

			amountStr := strings.TrimSpace(row[amountCol])
			var amount float64
			fmt.Sscanf(strings.ReplaceAll(amountStr, ",", ""), "%f", &amount)

			if amount < 0 || amount > 100000000 {
				result.Errors++
				if len(result.ErrorDetails) < 100 {
					result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: invalid amount '%s'", i+2, amountStr))
				}
				continue
			}

			currency := "BHD"
			if currencyCol != -1 && currencyCol < len(row) {
				curr := strings.TrimSpace(row[currencyCol])
				if curr != "" {
					currency = strings.ToUpper(curr)
				}
			}

			tallyPurchase := TallyPurchaseImport{
				Year:              year,
				InvoiceNumber:     invoiceNo,
				InvoiceDate:       invoiceDate,
				SupplierName:      supplierName,
				MatchedSupplierID: supplierID,
				Amount:            amount,
				Currency:          currency,
				ImportBatch:       result.BatchID,
			}

			if err := tx.Create(&tallyPurchase).Error; err != nil {
				result.Errors++
				if len(result.ErrorDetails) < 100 {
					result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: %v", i+2, err))
				} else if len(result.ErrorDetails) == 100 {
					result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
				}
			} else {
				result.Imported++
			}
		}
		return nil
	})

	if txErr != nil {
		return nil, fmt.Errorf("import transaction failed: %w", txErr)
	}

	log.Printf("Tally purchases import complete: %d imported, %d duplicates, %d errors out of %d rows",
		result.Imported, result.Duplicates, result.Errors, result.TotalRows)

	return result, nil
}

// ImportARDefaulters imports AR payment defaulters and updates customer records
func (a *App) ImportARDefaulters() (*TallyImportResult, error) {
	if err := a.requirePermission("data:import"); err != nil {
		return nil, err
	}

	result := &TallyImportResult{
		BatchID: uuid.New().String(),
	}

	arPossibleFiles := []string{
		filepath.Join("Data_for_database", "Tally_data", "AR payment defaulters.xlsx"),
		filepath.Join("Data_for_database", "AR payment defaulters.xlsx"),
		filepath.Join("data/ssot", "AR payment defaulters.xlsx"),
	}

	var f *excelize.File
	var err error
	var filePath string
	for _, fp := range arPossibleFiles {
		f, err = excelize.OpenFile(fp)
		if err == nil {
			filePath = fp
			break
		}
	}
	if f == nil {
		return nil, fmt.Errorf("could not find AR defaulters file. Tried: %v", arPossibleFiles)
	}
	defer f.Close()

	log.Printf("Importing AR defaulters from: %s", filePath)

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, fmt.Errorf("failed to read Sheet1: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("file has no data rows")
	}

	header := rows[0]
	colMap := map[string]int(excel.IndexHeader(header))

	customerCol := findColumn(colMap, "customer", "customer name", "party")
	outstandingCol := findColumn(colMap, "outstanding", "outstanding amount", "balance", "amount due")
	daysCol := findColumn(colMap, "days", "days overdue", "overdue days", "aging")

	if customerCol == -1 || outstandingCol == -1 {
		return nil, fmt.Errorf("missing required columns. Found headers: %v", header)
	}

	for i, row := range rows[1:] {
		result.TotalRows++

		if len(row) < len(header) {
			padded := make([]string, len(header))
			copy(padded, row)
			row = padded
		}

		customerName := strings.TrimSpace(row[customerCol])
		if customerName == "" {
			result.Errors++
			if len(result.ErrorDetails) < 100 {
				result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: Empty customer name", i+2))
			} else if len(result.ErrorDetails) == 100 {
				result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
			}
			continue
		}

		customerID := a.matchCustomerByName(customerName)
		if customerID == "" {
			result.Errors++
			if len(result.ErrorDetails) < 100 {
				result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: Customer not found: %s", i+2, customerName))
			} else if len(result.ErrorDetails) == 100 {
				result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
			}
			continue
		}

		outstandingStr := strings.TrimSpace(row[outstandingCol])
		var outstanding float64
		fmt.Sscanf(strings.ReplaceAll(outstandingStr, ",", ""), "%f", &outstanding)

		var daysOverdue int
		if daysCol != -1 && daysCol < len(row) {
			daysStr := strings.TrimSpace(row[daysCol])
			fmt.Sscanf(daysStr, "%d", &daysOverdue)
		}

		// Update customer record
		err := a.db.Model(&CustomerMaster{}).
			Where("id = ?", customerID).
			Updates(map[string]any{
				"outstanding_bhd": outstanding,
				"overdue_days":    daysOverdue,
			}).Error

		if err != nil {
			result.Errors++
			if len(result.ErrorDetails) < 100 {
				result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: Failed to update customer %s: %v", i+2, customerName, err))
			} else if len(result.ErrorDetails) == 100 {
				result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
			}
		} else {
			result.Imported++
		}
	}

	log.Printf("AR defaulters import complete: %d updated, %d errors out of %d rows",
		result.Imported, result.Errors, result.TotalRows)

	return result, nil
}

// ImportSupplierPaymentsFromFile imports supplier payments from Excel
func (a *App) ImportSupplierPaymentsFromFile() (*TallyImportResult, error) {
	if err := a.requirePermission("data:import"); err != nil {
		return nil, err
	}

	result := &TallyImportResult{
		BatchID: uuid.New().String(),
	}

	spPossibleFiles := []string{
		filepath.Join("Data_for_database", "Tally_data", "Payments to suppliers.xlsx"),
		filepath.Join("Data_for_database", "Payments to suppliers.xlsx"),
		filepath.Join("data/ssot", "Payments to suppliers.xlsx"),
	}

	var f *excelize.File
	var err error
	var filePath string
	for _, fp := range spPossibleFiles {
		f, err = excelize.OpenFile(fp)
		if err == nil {
			filePath = fp
			break
		}
	}
	if f == nil {
		return nil, fmt.Errorf("could not find supplier payments file. Tried: %v", spPossibleFiles)
	}
	defer f.Close()

	log.Printf("Importing supplier payments from: %s", filePath)

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, fmt.Errorf("failed to read Sheet1: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("file has no data rows")
	}

	header := rows[0]
	colMap := map[string]int(excel.IndexHeader(header))

	dateCol := findColumn(colMap, "date", "payment date", "paid date")
	supplierCol := findColumn(colMap, "supplier", "supplier name", "party", "vendor")
	amountCol := findColumn(colMap, "amount", "paid amount", "payment amount", "value")
	currencyCol := findColumn(colMap, "currency", "curr")
	referenceCol := findColumn(colMap, "reference", "ref", "payment reference", "cheque no", "transaction ref")

	if dateCol == -1 || supplierCol == -1 || amountCol == -1 {
		return nil, fmt.Errorf("missing required columns. Found headers: %v", header)
	}

	for i, row := range rows[1:] {
		result.TotalRows++

		if len(row) < len(header) {
			padded := make([]string, len(header))
			copy(padded, row)
			row = padded
		}

		supplierName := strings.TrimSpace(row[supplierCol])
		if supplierName == "" {
			result.Errors++
			if len(result.ErrorDetails) < 100 {
				result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: Empty supplier name", i+2))
			} else if len(result.ErrorDetails) == 100 {
				result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
			}
			continue
		}

		supplierID := a.matchSupplierByName(supplierName)
		if supplierID == "" {
			result.Errors++
			if len(result.ErrorDetails) < 100 {
				result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: Supplier not found: %s", i+2, supplierName))
			} else if len(result.ErrorDetails) == 100 {
				result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
			}
			continue
		}

		dateStr := strings.TrimSpace(row[dateCol])
		paymentDate, err := a.parseExcelDate(dateStr)
		if err != nil {
			result.Errors++
			if len(result.ErrorDetails) < 100 {
				result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: invalid date '%s'", i+2, dateStr))
			} else if len(result.ErrorDetails) == 100 {
				result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
			}
			continue
		}

		amountStr := strings.TrimSpace(row[amountCol])
		var amount float64
		fmt.Sscanf(strings.ReplaceAll(amountStr, ",", ""), "%f", &amount)

		currency := "BHD"
		if currencyCol != -1 && currencyCol < len(row) {
			curr := strings.TrimSpace(row[currencyCol])
			if curr != "" {
				currency = strings.ToUpper(curr)
			}
		}

		reference := ""
		if referenceCol != -1 && referenceCol < len(row) {
			reference = strings.TrimSpace(row[referenceCol])
		}

		payment := SupplierPayment{
			SupplierID:    supplierID,
			PaymentDate:   paymentDate,
			AmountBHD:     amount,
			Currency:      currency,
			PaymentMethod: "Bank Transfer",
			Reference:     reference,
			Notes:         fmt.Sprintf("Imported from Tally (Batch: %s)", result.BatchID),
		}

		if err := a.db.Create(&payment).Error; err != nil {
			result.Errors++
			if len(result.ErrorDetails) < 100 {
				result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("Row %d: %v", i+2, err))
			} else if len(result.ErrorDetails) == 100 {
				result.ErrorDetails = append(result.ErrorDetails, "... (additional errors truncated)")
			}
		} else {
			result.Imported++
		}
	}

	log.Printf("Supplier payments import complete: %d imported, %d errors out of %d rows",
		result.Imported, result.Errors, result.TotalRows)

	return result, nil
}

// ImportAllTallyData orchestrates all Tally imports
func (a *App) ImportAllTallyData() (*TallyImportResult, error) {
	if err := a.requirePermission("data:import"); err != nil {
		return nil, err
	}

	years := []int{2023, 2024, 2025}
	aggregateResult := &TallyImportResult{
		BatchID: uuid.New().String(),
	}

	log.Println("Starting comprehensive Tally data import (years 2023-2025)...")

	// Import invoices and purchases for each year
	for _, year := range years {
		invoiceResult, err := a.ImportTallyInvoices(year)
		if err != nil {
			log.Printf("Invoice import for %d failed: %v", year, err)
			aggregateResult.Errors++
			aggregateResult.ErrorDetails = append(aggregateResult.ErrorDetails, fmt.Sprintf("Invoices %d: %v", year, err))
		} else {
			aggregateResult.TotalRows += invoiceResult.TotalRows
			aggregateResult.Imported += invoiceResult.Imported
			aggregateResult.Duplicates += invoiceResult.Duplicates
			aggregateResult.Errors += invoiceResult.Errors
			aggregateResult.ErrorDetails = append(aggregateResult.ErrorDetails, invoiceResult.ErrorDetails...)
		}

		purchaseResult, err := a.ImportTallyPurchases(year)
		if err != nil {
			log.Printf("Purchase import for %d failed: %v", year, err)
			aggregateResult.Errors++
			aggregateResult.ErrorDetails = append(aggregateResult.ErrorDetails, fmt.Sprintf("Purchases %d: %v", year, err))
		} else {
			aggregateResult.TotalRows += purchaseResult.TotalRows
			aggregateResult.Imported += purchaseResult.Imported
			aggregateResult.Duplicates += purchaseResult.Duplicates
			aggregateResult.Errors += purchaseResult.Errors
			aggregateResult.ErrorDetails = append(aggregateResult.ErrorDetails, purchaseResult.ErrorDetails...)
		}
	}

	// Import AR defaulters
	arResult, err := a.ImportARDefaulters()
	if err != nil {
		log.Printf("AR defaulters import failed: %v", err)
		aggregateResult.Errors++
		aggregateResult.ErrorDetails = append(aggregateResult.ErrorDetails, fmt.Sprintf("AR Defaulters: %v", err))
	} else {
		aggregateResult.TotalRows += arResult.TotalRows
		aggregateResult.Imported += arResult.Imported
		aggregateResult.Errors += arResult.Errors
		aggregateResult.ErrorDetails = append(aggregateResult.ErrorDetails, arResult.ErrorDetails...)
	}

	// Import supplier payments
	paymentResult, err := a.ImportSupplierPaymentsFromFile()
	if err != nil {
		log.Printf("Supplier payments import failed: %v", err)
		aggregateResult.Errors++
		aggregateResult.ErrorDetails = append(aggregateResult.ErrorDetails, fmt.Sprintf("Supplier Payments: %v", err))
	} else {
		aggregateResult.TotalRows += paymentResult.TotalRows
		aggregateResult.Imported += paymentResult.Imported
		aggregateResult.Errors += paymentResult.Errors
		aggregateResult.ErrorDetails = append(aggregateResult.ErrorDetails, paymentResult.ErrorDetails...)
	}

	log.Printf("Comprehensive Tally import complete: %d total rows, %d imported, %d duplicates, %d errors",
		aggregateResult.TotalRows, aggregateResult.Imported, aggregateResult.Duplicates, aggregateResult.Errors)

	return aggregateResult, nil
}

// Helper: Match customer by name (fuzzy)
func (a *App) matchCustomerByName(name string) string {
	if name == "" {
		return ""
	}

	var customer CustomerMaster
	escapedName := escapeLikeWildcards(strings.ToLower(name))
	err := a.db.Where("LOWER(name) LIKE ? ESCAPE '\\'", "%"+escapedName+"%").First(&customer).Error
	if err != nil {
		return ""
	}
	return customer.ID
}

// Helper: Match supplier by name (fuzzy)
func (a *App) matchSupplierByName(name string) string {
	if name == "" {
		return ""
	}

	var supplier SupplierMaster
	escapedName := escapeLikeWildcards(strings.ToLower(name))
	err := a.db.Where("LOWER(name) LIKE ? ESCAPE '\\'", "%"+escapedName+"%").First(&supplier).Error
	if err != nil {
		return ""
	}
	return supplier.ID
}

// Helper: Parse Excel date (handles various formats)
func (a *App) parseExcelDate(cell string) (time.Time, error) {
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// Try common formats
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
		"2-Jan-2006",
		"02-Jan-06",
		"2006/01/02",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, cell); err == nil {
			return t, nil
		}
	}

	// Try Excel numeric date (days since 1900-01-01)
	var excelDate float64
	if _, err := fmt.Sscanf(cell, "%f", &excelDate); err == nil && excelDate > 0 {
		// Excel epoch: December 30, 1899 (accounting for Excel's 1900 leap year bug)
		excelEpoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		return excelEpoch.Add(time.Duration(excelDate * 24 * float64(time.Hour))), nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", cell)
}

// Helper: Find column index by flexible matching. Delegates to the
// pkg/documents/excel engine's VARIANT-priority lookup (Wave 3 B.3).
func findColumn(colMap map[string]int, variants ...string) int {
	return excel.HeaderIndex(colMap).Find(variants...)
}

// =============================================================================
// P&L AND BALANCE SHEET REPORT GENERATION (TALLY-BASED)
// =============================================================================

// TallyPLReport represents a Profit & Loss statement for a given year from Tally imports
type TallyPLReport struct {
	Year        int       `json:"year"`
	GeneratedAt time.Time `json:"generated_at"`

	// REVENUE SECTION
	SalesRevenue float64 `json:"sales_revenue"`
	OtherIncome  float64 `json:"other_income"`
	TotalRevenue float64 `json:"total_revenue"`

	// COST OF GOODS SOLD
	Purchases         float64 `json:"purchases"`
	CostOfGoodsSold   float64 `json:"cost_of_goods_sold"`
	GrossProfit       float64 `json:"gross_profit"`
	GrossProfitMargin float64 `json:"gross_profit_margin"` // percentage

	// OPERATING EXPENSES (categorized if possible)
	OperatingExpenses float64 `json:"operating_expenses"`

	// NET PROFIT
	NetProfit       float64 `json:"net_profit"`
	NetProfitMargin float64 `json:"net_profit_margin"` // percentage

	// Supporting metrics
	InvoiceCount  int    `json:"invoice_count"`
	PurchaseCount int    `json:"purchase_count"`
	Currency      string `json:"currency"`

	// Breakdown details (optional, for drill-down)
	MonthlyBreakdown []MonthlyPLBreakdown `json:"monthly_breakdown,omitempty"`
}

// MonthlyPLBreakdown provides monthly granularity
type MonthlyPLBreakdown struct {
	Month       int     `json:"month"`
	MonthName   string  `json:"month_name"`
	Revenue     float64 `json:"revenue"`
	Purchases   float64 `json:"purchases"`
	GrossProfit float64 `json:"gross_profit"`
	NetProfit   float64 `json:"net_profit"`
}

// TallyBalanceSheet represents a Balance Sheet as of a given date from Tally data
type TallyBalanceSheet struct {
	AsOfDate    time.Time `json:"as_of_date"`
	Year        int       `json:"year"`
	GeneratedAt time.Time `json:"generated_at"`

	// ASSETS
	Cash               float64 `json:"cash"`
	AccountsReceivable float64 `json:"accounts_receivable"`
	Inventory          float64 `json:"inventory"`
	TotalCurrentAssets float64 `json:"total_current_assets"`
	TotalAssets        float64 `json:"total_assets"`

	// LIABILITIES
	AccountsPayable         float64 `json:"accounts_payable"`
	TotalCurrentLiabilities float64 `json:"total_current_liabilities"`
	TotalLiabilities        float64 `json:"total_liabilities"`

	// EQUITY
	RetainedEarnings float64 `json:"retained_earnings"`
	TotalEquity      float64 `json:"total_equity"`

	Currency string `json:"currency"`
}

// GenerateProfitAndLoss generates a P&L report for the given year from imported Tally data
func (a *App) GenerateProfitAndLoss(year int) (*TallyPLReport, error) {
	if err := a.requirePermission("reports:view"); err != nil {
		return nil, err
	}

	log.Printf("Generating P&L report for year %d", year)

	report := &TallyPLReport{
		Year:        year,
		GeneratedAt: time.Now(),
		Currency:    "BHD",
	}

	// Calculate Sales Revenue from TallyInvoiceImport
	var invoices []TallyInvoiceImport
	err := a.db.Where("year = ? AND currency = ?", year, "BHD").Find(&invoices).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invoices: %w", err)
	}

	report.InvoiceCount = len(invoices)
	for _, inv := range invoices {
		report.SalesRevenue += inv.Amount
	}

	// Calculate Purchases (COGS) from TallyPurchaseImport
	var purchases []TallyPurchaseImport
	err = a.db.Where("year = ? AND currency = ?", year, "BHD").Find(&purchases).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch purchases: %w", err)
	}

	report.PurchaseCount = len(purchases)
	for _, pur := range purchases {
		report.Purchases += pur.Amount
	}

	// P&L Calculations
	report.CostOfGoodsSold = report.Purchases // Simplification: purchases = COGS
	report.TotalRevenue = report.SalesRevenue + report.OtherIncome
	report.GrossProfit = report.TotalRevenue - report.CostOfGoodsSold

	if report.TotalRevenue > 0 {
		report.GrossProfitMargin = (report.GrossProfit / report.TotalRevenue) * 100
	}

	// Operating expenses (set to 0 for now unless we have data source)
	report.OperatingExpenses = 0

	// Net Profit
	report.NetProfit = report.GrossProfit - report.OperatingExpenses

	if report.TotalRevenue > 0 {
		report.NetProfitMargin = (report.NetProfit / report.TotalRevenue) * 100
	}

	// Generate monthly breakdown
	monthlyMap := make(map[int]*MonthlyPLBreakdown)
	for i := 1; i <= 12; i++ {
		monthlyMap[i] = &MonthlyPLBreakdown{
			Month:     i,
			MonthName: time.Month(i).String(),
		}
	}

	// Aggregate invoices by month
	for _, inv := range invoices {
		month := int(inv.InvoiceDate.Month())
		if mb, ok := monthlyMap[month]; ok {
			mb.Revenue += inv.Amount
		}
	}

	// Aggregate purchases by month
	for _, pur := range purchases {
		month := int(pur.InvoiceDate.Month())
		if mb, ok := monthlyMap[month]; ok {
			mb.Purchases += pur.Amount
		}
	}

	// Calculate monthly P&L
	for _, mb := range monthlyMap {
		mb.GrossProfit = mb.Revenue - mb.Purchases
		mb.NetProfit = mb.GrossProfit // No operating expenses breakdown
	}

	// Convert map to sorted slice
	report.MonthlyBreakdown = make([]MonthlyPLBreakdown, 0, 12)
	for i := 1; i <= 12; i++ {
		report.MonthlyBreakdown = append(report.MonthlyBreakdown, *monthlyMap[i])
	}

	log.Printf("P&L Report generated: Revenue=%.3f BHD, COGS=%.3f BHD, Gross Profit=%.3f BHD (%.1f%%), Net Profit=%.3f BHD (%.1f%%)",
		report.TotalRevenue, report.CostOfGoodsSold, report.GrossProfit, report.GrossProfitMargin, report.NetProfit, report.NetProfitMargin)

	return report, nil
}

// GenerateBalanceSheet generates a Balance Sheet report as of the end of the given year
func (a *App) GenerateBalanceSheet(year int) (*TallyBalanceSheet, error) {
	if err := a.requirePermission("reports:view"); err != nil {
		return nil, err
	}

	log.Printf("Generating Balance Sheet for year %d", year)

	asOfDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
	report := &TallyBalanceSheet{
		AsOfDate:    asOfDate,
		Year:        year,
		GeneratedAt: time.Now(),
		Currency:    "BHD",
	}

	// ASSETS: Calculate Accounts Receivable from CustomerMaster
	var customers []CustomerMaster
	err := a.db.Find(&customers).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch customers: %w", err)
	}

	for _, cust := range customers {
		report.AccountsReceivable += cust.OutstandingBHD
	}

	// ASSETS: Calculate Inventory from InventoryItem
	var inventoryItems []InventoryItem
	err = a.db.Where("is_active = ?", true).Find(&inventoryItems).Error
	if err != nil {
		log.Printf("Warning: failed to fetch inventory items: %v", err)
	}

	for _, item := range inventoryItems {
		report.Inventory += item.TotalValue
	}

	// ASSETS: Cash (placeholder - would come from Chart of Accounts in real system)
	// For now, calculate as: Revenue - Purchases - Operating Expenses (accumulated profit)
	pl, err := a.GenerateProfitAndLoss(year)
	if err != nil {
		return nil, fmt.Errorf("failed to generate P&L for balance sheet: %w", err)
	}
	report.Cash = pl.NetProfit // Simplification: net profit = cash (assumes no accruals)

	// Total Current Assets
	report.TotalCurrentAssets = report.Cash + report.AccountsReceivable + report.Inventory
	report.TotalAssets = report.TotalCurrentAssets

	// LIABILITIES: Calculate Accounts Payable from SupplierInvoice
	var supplierInvoices []SupplierInvoice
	err = a.db.Where("payment_status = ? OR payment_status = ?", "Unpaid", "Scheduled").Find(&supplierInvoices).Error
	if err != nil {
		log.Printf("Warning: failed to fetch supplier invoices: %v", err)
	}

	for _, sinv := range supplierInvoices {
		report.AccountsPayable += sinv.TotalBHD
	}

	// Total Current Liabilities
	report.TotalCurrentLiabilities = report.AccountsPayable
	report.TotalLiabilities = report.TotalCurrentLiabilities

	// EQUITY: Retained Earnings = Assets - Liabilities
	report.RetainedEarnings = report.TotalAssets - report.TotalLiabilities
	report.TotalEquity = report.RetainedEarnings

	log.Printf("Balance Sheet generated: Assets=%.3f BHD, Liabilities=%.3f BHD, Equity=%.3f BHD",
		report.TotalAssets, report.TotalLiabilities, report.TotalEquity)

	return report, nil
}

// GetFinancialReportYears returns available years for financial reports
// Now includes audited years (2023, 2024) plus any Tally import years
func (a *App) GetFinancialReportYears() ([]int, error) {
	if err := a.requirePermission("reports:view"); err != nil {
		return nil, err
	}

	// Use the new financial year service which includes:
	// - Audited years (2023, 2024 from FS2024)
	// - Any additional years from Tally imports
	return a.GetAvailableFinancialYears()
}
