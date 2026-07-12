// =============================================================================
// BANK STATEMENT PARSER SERVICE
//
// MISSION: Parse bank statements from PDF (via OCR) and CSV formats
// FORMATS: NBB (National Bank of Bahrain), KFH, Al Salam, generic CSV
// =============================================================================

package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// PARSED STRUCTURES
// =============================================================================

type parsedStatement struct {
	AccountNumber  string
	IBAN           string
	Currency       string
	PeriodStart    time.Time
	PeriodEnd      time.Time
	OpeningBalance float64
	ClosingBalance float64
	TotalDebits    float64
	TotalCredits   float64
	DebitCount     int
	CreditCount    int
	Lines          []parsedLine
}

type parsedLine struct {
	LineNumber  int
	Date        time.Time
	ValueDate   time.Time
	Reference   string
	Description string
	Debit       float64
	Credit      float64
	Balance     float64
}

// =============================================================================
// PDF IMPORT (via OCR + Butler AI)
// =============================================================================

// ImportBankStatementPDF imports a bank statement from PDF using OCR
func (a *App) ImportBankStatementPDF(filePath string, bankAccountID string) (*BankStatement, error) {
	statement, err := a.parseBankStatementPDF(filePath, bankAccountID)
	if err != nil {
		return nil, err
	}

	// Save to database
	if err := a.db.Create(statement).Error; err != nil {
		return nil, fmt.Errorf("failed to save statement: %w", err)
	}

	// Archive the original PDF
	a.ArchiveStatementPDF(statement.ID, filePath)

	log.Printf("📊 Imported PDF statement: %s with %d lines (Opening: %.3f, Closing: %.3f)",
		statement.StatementNumber, len(statement.Lines), statement.OpeningBalance, statement.ClosingBalance)

	return statement, nil
}

// parseBankStatementPDF parses (OCR + format detection) a bank statement PDF
// into an in-memory *BankStatement WITHOUT persisting it. Wave 9.3 B1d: this
// is the shared core behind ImportBankStatementPDF (parse + persist) and
// PreviewBankStatementImportWithDialog (parse only, review before commit).
func (a *App) parseBankStatementPDF(filePath string, bankAccountID string) (*BankStatement, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Verify bank account exists
	var bankAccount CompanyBankAccount
	if err := a.db.First(&bankAccount, "id = ?", bankAccountID).Error; err != nil {
		return nil, fmt.Errorf("bank account not found: %w", err)
	}

	// Step 1: OCR the PDF. Use the direct OCR pipeline so we always receive the
	// extracted text payload instead of the lightweight quick-capture summary map.
	ocrResult, err := a.ProcessDocumentWithOCR(filePath, "bank_statement")
	if err != nil {
		return nil, fmt.Errorf("OCR failed: %w", err)
	}

	if ocrResult == nil {
		return nil, fmt.Errorf("OCR returned empty result")
	}

	extractedText := strings.TrimSpace(ocrResult.Text)

	if extractedText == "" {
		return nil, fmt.Errorf("OCR returned empty text")
	}

	// Step 2: Parse and validate the bank statement.
	parseOutcome, err := a.parseImportedPDFBankStatement(extractedText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse statement: %w", err)
	}
	parsed := parseOutcome.Statement

	// Step 3: Create the BankStatement
	statementNumber := buildBankStatementNumber(bankAccount, parsed.PeriodStart, parsed.PeriodEnd)
	if statementNumber == "" {
		statementNumber = fmt.Sprintf("%s-%s", bankAccount.AccountNumber, parsed.PeriodEnd.Format("200601"))
	}
	statementNotes := strings.TrimSpace(strings.Join(parseOutcome.ValidationNote, "\n"))
	if statementNotes == "" {
		statementNotes = buildBankStatementSummaryNote(bankAccount, parsed.PeriodStart, parsed.PeriodEnd, parsed.OpeningBalance, parsed.ClosingBalance, parsed.TotalDebits, parsed.TotalCredits, parsed.Currency)
	}

	statement := &BankStatement{
		Base:            Base{ID: uuid.New().String()},
		BankAccountID:   bankAccountID,
		Division:        normalizeDivisionName(bankAccount.Division),
		StatementNumber: statementNumber,
		StatementDate:   time.Now(),
		PeriodStart:     parsed.PeriodStart,
		PeriodEnd:       parsed.PeriodEnd,
		OpeningBalance:  parsed.OpeningBalance,
		ClosingBalance:  parsed.ClosingBalance,
		Status:          "Imported",
		ImportedFrom:    filePath,
		ImportMethod:    parseOutcome.ImportMethod,
		OCRConfidence:   parseOutcome.OCRConfidence,
		TotalDebits:     parsed.TotalDebits,
		TotalCredits:    parsed.TotalCredits,
		DebitCount:      parsed.DebitCount,
		CreditCount:     parsed.CreditCount,
		Notes:           statementNotes,
	}

	// Step 4: Create statement lines
	var lines []BankStatementLine
	for _, pl := range parsed.Lines {
		line := BankStatementLine{
			Base:            Base{ID: uuid.New().String()},
			BankStatementID: statement.ID,
			LineNumber:      pl.LineNumber,
			TransactionDate: pl.Date,
			ValueDate:       pl.ValueDate,
			Reference:       pl.Reference,
			Description:     pl.Description,
			Debit:           pl.Debit,
			Credit:          pl.Credit,
			Balance:         pl.Balance,
			IsMatched:       false,
			MatchType:       "Unmatched",
		}
		lines = append(lines, line)
	}
	statement.Lines = lines

	return statement, nil
}

// =============================================================================
// CSV IMPORT
// =============================================================================

// ImportBankStatementCSV imports a bank statement from CSV file
func (a *App) ImportBankStatementCSV(filePath string, bankAccountID string) (*BankStatement, error) {
	statement, err := a.parseBankStatementCSV(filePath, bankAccountID)
	if err != nil {
		return nil, err
	}

	// Save to database
	if err := a.db.Create(statement).Error; err != nil {
		return nil, fmt.Errorf("failed to save statement: %w", err)
	}

	log.Printf("📊 Imported CSV statement: %s with %d lines", statement.StatementNumber, len(statement.Lines))
	return statement, nil
}

// parseBankStatementCSV parses a bank statement CSV file into an in-memory
// *BankStatement WITHOUT persisting it (Wave 9.3 B1d — shared with the
// import preview path).
func (a *App) parseBankStatementCSV(filePath string, bankAccountID string) (*BankStatement, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Verify bank account exists
	var bankAccount CompanyBankAccount
	if err := a.db.First(&bankAccount, "id = ?", bankAccountID).Error; err != nil {
		return nil, fmt.Errorf("bank account not found: %w", err)
	}

	// Open and parse CSV
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields per row (bank CSV metadata)
	reader.LazyQuotes = true    // Handle irregular quoting in bank CSVs
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file has no data rows")
	}

	// Parse the CSV content
	parsed, err := parseCSVRecords(records)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	// Generate statement number
	statementNumber := fmt.Sprintf("%s-%s", bankAccount.AccountNumber, parsed.PeriodEnd.Format("200601"))

	// Create statement
	statement := &BankStatement{
		Base:            Base{ID: uuid.New().String()},
		BankAccountID:   bankAccountID,
		Division:        normalizeDivisionName(bankAccount.Division),
		StatementNumber: statementNumber,
		StatementDate:   time.Now(),
		PeriodStart:     parsed.PeriodStart,
		PeriodEnd:       parsed.PeriodEnd,
		OpeningBalance:  parsed.OpeningBalance,
		ClosingBalance:  parsed.ClosingBalance,
		Status:          "Imported",
		ImportedFrom:    filePath,
		TotalDebits:     parsed.TotalDebits,
		TotalCredits:    parsed.TotalCredits,
		DebitCount:      parsed.DebitCount,
		CreditCount:     parsed.CreditCount,
	}

	// Create lines
	var lines []BankStatementLine
	for _, pl := range parsed.Lines {
		line := BankStatementLine{
			Base:            Base{ID: uuid.New().String()},
			BankStatementID: statement.ID,
			LineNumber:      pl.LineNumber,
			TransactionDate: pl.Date,
			ValueDate:       pl.ValueDate,
			Reference:       pl.Reference,
			Description:     pl.Description,
			Debit:           pl.Debit,
			Credit:          pl.Credit,
			Balance:         pl.Balance,
			IsMatched:       false,
			MatchType:       "Unmatched",
		}
		lines = append(lines, line)
	}
	statement.Lines = lines

	return statement, nil
}

// =============================================================================
// NBB FORMAT PARSER (National Bank of Bahrain)
// =============================================================================

// parseNBBFormat parses NBB bank statement format from OCR text
func parseNBBFormat(text string) (*parsedStatement, error) {
	result := &parsedStatement{
		Currency: "BHD",
	}

	lowerText := strings.ToLower(text)
	hasNBBSignature := strings.Contains(lowerText, "national bank of bahrain") ||
		strings.Contains(lowerText, " nbb ") ||
		strings.Contains(lowerText, "nbobbhbm") ||
		strings.Contains(lowerText, "account statement - tax invoice")
	if !hasNBBSignature {
		return nil, fmt.Errorf("text does not appear to be an NBB statement")
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r", "\n"), "\n")

	// Extract header information
	for _, rawLine := range lines {
		line := strings.Join(strings.Fields(strings.TrimSpace(rawLine)), " ")
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)

		// Account Number
		if strings.Contains(lower, "account number") || strings.Contains(lower, "account no") {
			result.AccountNumber = extractNumber(line)
		}

		if strings.Contains(lower, "currency") {
			if currency := extractCurrencyCode(line); currency != "" {
				result.Currency = currency
			}
		}

		// IBAN
		if strings.Contains(line, "IBAN") {
			ibanMatch := regexp.MustCompile(`BH\d{2}[A-Z]{4}\d{14}`).FindString(line)
			if ibanMatch != "" {
				result.IBAN = ibanMatch
			}
		}

		// Opening Balance
		if strings.Contains(lower, "opening balance") {
			result.OpeningBalance = extractAmount(line)
		}

		// Closing Balance
		if strings.Contains(lower, "closing balance") {
			result.ClosingBalance = extractAmount(line)
		}

		// Statement summary totals
		if strings.Contains(lower, "total debits") {
			result.DebitCount, result.TotalDebits = extractSummaryCountAndAmount(line)
		}
		if strings.Contains(lower, "total credits") {
			result.CreditCount, result.TotalCredits = extractSummaryCountAndAmount(line)
		}

		// Statement Period
		if (strings.Contains(lower, "statement") || strings.Contains(lower, "period")) && strings.Contains(lower, "to") {
			result.PeriodStart, result.PeriodEnd = extractPeriod(line)
		}
	}

	entries := collectOCRBankEntries(lines)
	previousBalance := result.OpeningBalance
	previousBalanceKnown := result.OpeningBalance != 0

	for _, entry := range entries {
		pl, ok := parseNBBTransactionEntry(entry, previousBalance, previousBalanceKnown)
		if !ok {
			continue
		}
		pl.LineNumber = len(result.Lines) + 1
		result.Lines = append(result.Lines, pl)
		if pl.Balance != 0 {
			previousBalance = pl.Balance
			previousBalanceKnown = true
		}
	}

	// If no transactions found, return error
	if len(result.Lines) == 0 {
		return nil, fmt.Errorf("no transactions found in NBB format")
	}

	// Infer period from transactions if not found in header
	if result.PeriodStart.IsZero() && len(result.Lines) > 0 {
		result.PeriodStart = result.Lines[0].Date
		result.PeriodEnd = result.Lines[len(result.Lines)-1].Date
	}

	repairParsedStatementPolarity(result)
	if result.OpeningBalance == 0 && len(result.Lines) > 0 {
		first := result.Lines[0]
		if strings.Contains(strings.ToLower(first.Description), "opening balance") {
			result.OpeningBalance = first.Balance
		} else if first.Balance != 0 {
			result.OpeningBalance = first.Balance + first.Debit - first.Credit
		}
	}
	if result.ClosingBalance == 0 && len(result.Lines) > 0 {
		result.ClosingBalance = result.Lines[len(result.Lines)-1].Balance
	}

	if columnar := parseNBBColumnarLayout(lines, result); shouldPreferColumnarNBBParse(result, columnar) {
		return columnar, nil
	}

	return result, nil
}

func shouldPreferColumnarNBBParse(standard, columnar *parsedStatement) bool {
	if columnar == nil || len(columnar.Lines) == 0 {
		return false
	}
	if standard == nil || len(standard.Lines) == 0 {
		return true
	}
	if len(columnar.Lines) >= len(standard.Lines)+3 {
		return true
	}

	first := standard.Lines[0]
	firstLower := strings.ToLower(first.Description)
	headerKeywords := []string{
		"account number",
		"date printed",
		"amount type",
		"bank bic",
		"available balance",
		"current balance",
		"account type",
	}
	for _, keyword := range headerKeywords {
		if strings.Contains(firstLower, keyword) {
			return true
		}
	}
	if first.Debit == 0 && first.Credit == 0 {
		return true
	}
	if first.Reference == "" && columnar.Lines[0].Reference != "" {
		return true
	}
	if !standard.PeriodEnd.IsZero() && first.Date.After(standard.PeriodEnd) {
		return true
	}
	if !standard.PeriodStart.IsZero() && first.Date.Before(standard.PeriodStart.AddDate(0, 0, -1)) {
		return true
	}
	return false
}

func parseNBBColumnarLayout(lines []string, base *parsedStatement) *parsedStatement {
	if base == nil {
		return nil
	}

	result := &parsedStatement{
		AccountNumber:  base.AccountNumber,
		IBAN:           base.IBAN,
		Currency:       base.Currency,
		PeriodStart:    base.PeriodStart,
		PeriodEnd:      base.PeriodEnd,
		OpeningBalance: base.OpeningBalance,
		ClosingBalance: base.ClosingBalance,
		TotalDebits:    base.TotalDebits,
		TotalCredits:   base.TotalCredits,
		DebitCount:     base.DebitCount,
		CreditCount:    base.CreditCount,
	}

	normalized := make([]string, 0, len(lines))
	for _, raw := range lines {
		line := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
		if line == "" {
			continue
		}
		normalized = append(normalized, line)
	}

	start := -1
	for idx, line := range normalized {
		lower := strings.ToLower(line)
		if lower == "reference number" || (strings.Contains(lower, "reference number") && strings.Contains(lower, "debit amount")) {
			start = idx
			break
		}
	}
	if start < 0 {
		return nil
	}

	datePattern := regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`)
	refPattern := regexp.MustCompile(`^\d{6,}$`)
	amountPattern := regexp.MustCompile(`^[\d,]+\.\d{2,3}$`)
	currencyPattern := regexp.MustCompile(`^[A-Z]{3}$`)
	previousBalance := result.OpeningBalance
	previousBalanceKnown := previousBalance != 0

	for idx := start + 1; idx < len(normalized); {
		line := normalized[idx]
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "date and time:") || strings.HasPrefix(lower, "page ") || strings.HasPrefix(lower, "--- page") {
			idx++
			continue
		}
		if strings.HasPrefix(lower, "for the national bank") || strings.Contains(lower, "system generated statement") {
			idx++
			continue
		}
		if line == "Opening Balance" {
			idx++
			continue
		}
		if !datePattern.MatchString(line) {
			idx++
			continue
		}

		date := parseFlexibleDate(line)
		valueDate := date
		idx++

		reference := ""
		if idx < len(normalized) && refPattern.MatchString(normalized[idx]) {
			reference = normalized[idx]
			idx++
		}

		descriptionParts := make([]string, 0, 2)
		for idx < len(normalized) {
			next := normalized[idx]
			nextLower := strings.ToLower(next)
			if amountPattern.MatchString(next) {
				break
			}
			if datePattern.MatchString(next) {
				break
			}
			if next == "Opening Balance" || nextLower == "currency" || nextLower == "balance" || nextLower == "debit amount" || nextLower == "credit amount" {
				idx++
				continue
			}
			if strings.HasPrefix(nextLower, "date and time:") || strings.HasPrefix(nextLower, "page ") || strings.HasPrefix(nextLower, "--- page") {
				break
			}
			descriptionParts = append(descriptionParts, next)
			idx++
		}
		if idx >= len(normalized) || !amountPattern.MatchString(normalized[idx]) {
			continue
		}

		balance := parseAmount(normalized[idx])
		idx++
		if idx < len(normalized) && currencyPattern.MatchString(normalized[idx]) {
			idx++
		}
		if idx >= len(normalized) || !amountPattern.MatchString(normalized[idx]) {
			continue
		}
		amount := parseAmount(normalized[idx])
		idx++

		description := strings.TrimSpace(strings.Join(descriptionParts, " "))
		if description == "" {
			continue
		}
		debit, credit := inferAmountByDescriptionOrBalance(description, amount, balance, previousBalance, previousBalanceKnown)
		result.Lines = append(result.Lines, parsedLine{
			LineNumber:  len(result.Lines) + 1,
			Date:        date,
			ValueDate:   valueDate,
			Reference:   reference,
			Description: description,
			Debit:       debit,
			Credit:      credit,
			Balance:     balance,
		})
		if balance != 0 {
			previousBalance = balance
			previousBalanceKnown = true
		}
	}

	if len(result.Lines) == 0 {
		return nil
	}
	repairParsedStatementPolarity(result)
	result.ClosingBalance = result.Lines[len(result.Lines)-1].Balance
	if result.OpeningBalance == 0 && len(result.Lines) > 0 {
		first := result.Lines[0]
		result.OpeningBalance = first.Balance + first.Debit - first.Credit
	}
	return result
}

// =============================================================================
// GENERIC FORMAT PARSER
// =============================================================================

// parseGenericBankFormat attempts to parse any bank statement format
func parseGenericBankFormat(text string) (*parsedStatement, error) {
	result := &parsedStatement{
		Currency: "BHD",
	}

	lines := strings.Split(text, "\n")

	// Try to find dates, amounts, and descriptions
	datePattern := regexp.MustCompile(`\d{1,2}[/-]\d{1,2}[/-]\d{2,4}`)
	amountPattern := regexp.MustCompile(`[\d,]+\.\d{2,3}`)

	lineNum := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 10 {
			continue
		}

		dates := datePattern.FindAllString(line, -1)
		amounts := amountPattern.FindAllString(line, -1)

		if len(dates) > 0 && len(amounts) > 0 {
			lineNum++
			date := parseFlexibleDate(dates[0])

			var debit, credit, balance float64
			switch len(amounts) {
			case 1:
				balance = parseAmount(amounts[0])
			case 2:
				credit = parseAmount(amounts[0])
				balance = parseAmount(amounts[1])
			case 3:
				debit = parseAmount(amounts[0])
				credit = parseAmount(amounts[1])
				balance = parseAmount(amounts[2])
			}

			// Extract description (text between date and amounts)
			desc := line
			for _, d := range dates {
				desc = strings.Replace(desc, d, "", 1)
			}
			for _, a := range amounts {
				desc = strings.Replace(desc, a, "", 1)
			}
			desc = strings.TrimSpace(desc)

			pl := parsedLine{
				LineNumber:  lineNum,
				Date:        date,
				ValueDate:   date,
				Description: desc,
				Debit:       debit,
				Credit:      credit,
				Balance:     balance,
			}

			if pl.Debit > 0 {
				result.DebitCount++
				result.TotalDebits += pl.Debit
			}
			if pl.Credit > 0 {
				result.CreditCount++
				result.TotalCredits += pl.Credit
			}

			result.Lines = append(result.Lines, pl)
		}
	}

	if len(result.Lines) == 0 {
		return nil, fmt.Errorf("no transactions found in generic format")
	}

	repairParsedStatementPolarity(result)

	// Infer balances and period from lines
	result.PeriodStart = result.Lines[0].Date
	result.PeriodEnd = result.Lines[len(result.Lines)-1].Date
	result.OpeningBalance = result.Lines[0].Balance + result.Lines[0].Debit - result.Lines[0].Credit
	result.ClosingBalance = result.Lines[len(result.Lines)-1].Balance

	return result, nil
}

func parseKFHFormat(text string) (*parsedStatement, error) {
	result := &parsedStatement{
		Currency: "BHD",
	}

	normalizedText := strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(normalizedText, "\n")
	collapsed := strings.Join(strings.Fields(normalizedText), " ")

	if match := regexp.MustCompile(`Account Number:\s*([A-Z0-9]+)`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.AccountNumber = strings.TrimSpace(match[1])
	}
	if match := regexp.MustCompile(`IBAN:\s*(BH\d{2}[A-Z]{4}[A-Z0-9]+)`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.IBAN = strings.TrimSpace(match[1])
	}
	if match := regexp.MustCompile(`Currency:\s*([A-Z]{3})`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.Currency = strings.TrimSpace(match[1])
	}
	if match := regexp.MustCompile(`From\s*-\s*To:\s*(\d{2}/\d{2}/\d{4})(?:\s*-\s*|\s+)(\d{2}/\d{2}/\d{4})`).FindStringSubmatch(collapsed); len(match) == 3 {
		result.PeriodStart = parseFlexibleDate(match[1])
		result.PeriodEnd = parseFlexibleDate(match[2])
	}
	if match := regexp.MustCompile(`Opening Balance:\s*([\d,]+\.\d{3})`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.OpeningBalance = parseAmount(match[1])
	}
	if match := regexp.MustCompile(`Closing Balance:\s*([\d,]+\.\d{3})`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.ClosingBalance = parseAmount(match[1])
	}

	for _, rawLine := range lines {
		line := strings.Join(strings.Fields(strings.TrimSpace(rawLine)), " ")
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)

		if strings.Contains(lower, "opening balance") {
			if amount := extractAmount(line); amount != 0 {
				result.OpeningBalance = amount
			}
		}
		if strings.Contains(lower, "closing balance") {
			if amount := extractAmount(line); amount != 0 {
				result.ClosingBalance = amount
			}
		}
	}

	entries := collectOCRBankEntriesWithStart(lines, regexp.MustCompile(`^\s*\d{2}/\d{2}/\d{4}\s+\d{2}/\d{2}/\d{4}\b`))
	previousBalance := result.OpeningBalance
	previousBalanceKnown := result.OpeningBalance != 0

	for _, entry := range entries {
		pl, ok := parseKFHTransactionEntry(entry, previousBalance, previousBalanceKnown)
		if !ok {
			continue
		}
		pl.LineNumber = len(result.Lines) + 1
		result.Lines = append(result.Lines, pl)
		if pl.Debit > 0 {
			result.TotalDebits += pl.Debit
			result.DebitCount++
		}
		if pl.Credit > 0 {
			result.TotalCredits += pl.Credit
			result.CreditCount++
		}
		if pl.Balance != 0 {
			previousBalance = pl.Balance
			previousBalanceKnown = true
		}
	}

	if len(result.Lines) == 0 {
		columnarLines := parseKFHColumnarLayout(lines, result.OpeningBalance, result.OpeningBalance != 0)
		for _, pl := range columnarLines {
			pl.LineNumber = len(result.Lines) + 1
			result.Lines = append(result.Lines, pl)
			if pl.Debit > 0 {
				result.TotalDebits += pl.Debit
				result.DebitCount++
			}
			if pl.Credit > 0 {
				result.TotalCredits += pl.Credit
				result.CreditCount++
			}
		}
	}

	if len(result.Lines) == 0 {
		return nil, fmt.Errorf("no transactions found in KFH format")
	}
	repairParsedStatementPolarity(result)
	if result.PeriodStart.IsZero() {
		result.PeriodStart = result.Lines[0].Date
	}
	if result.PeriodEnd.IsZero() {
		result.PeriodEnd = result.Lines[len(result.Lines)-1].Date
	}
	if result.ClosingBalance == 0 {
		result.ClosingBalance = result.Lines[len(result.Lines)-1].Balance
	}

	return result, nil
}

func parseAlSalamOCRFormat(text string) (*parsedStatement, error) {
	result := &parsedStatement{
		Currency: "BHD",
	}

	normalizedText := strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(normalizedText, "\n")
	collapsed := strings.Join(strings.Fields(normalizedText), " ")

	if match := regexp.MustCompile(`IBAN:\s*(BH\d{2}[A-Z]{4}[A-Z0-9]+)`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.IBAN = strings.TrimSpace(match[1])
	}
	if match := regexp.MustCompile(`Account Number:\s*([A-Z0-9]+)`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.AccountNumber = strings.TrimSpace(match[1])
	}
	if match := regexp.MustCompile(`Account Currency:\s*([A-Z]{3})`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.Currency = strings.TrimSpace(match[1])
	}
	if match := regexp.MustCompile(`From:\s*(\d{2}\s+[A-Za-z]{3}\s+\d{4})`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.PeriodStart = parseFlexibleDate(match[1])
	}
	if match := regexp.MustCompile(`To:\s*(\d{2}\s+[A-Za-z]{3}\s+\d{4})`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.PeriodEnd = parseFlexibleDate(match[1])
	}
	if match := regexp.MustCompile(`Closing Balance:\s*[A-Z]{3}\s*([\d,]+\.\d{3})`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.ClosingBalance = parseAmount(match[1])
	}
	if match := regexp.MustCompile(`Total Credits:\s*[A-Z]{3}\s*([\d,]+\.\d{3})`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.TotalCredits = parseAmount(match[1])
	}
	if match := regexp.MustCompile(`Total Debits:\s*[A-Z]{3}\s*([\d,]+\.\d{3})`).FindStringSubmatch(collapsed); len(match) == 2 {
		result.TotalDebits = parseAmount(match[1])
	}

	entries := collectOCRBankEntriesWithStart(lines, regexp.MustCompile(`^\s*\d{2}\s+[A-Za-z]{3}\s+\d{4}\b`))
	previousBalance := result.OpeningBalance
	previousBalanceKnown := result.OpeningBalance != 0

	for _, entry := range entries {
		pl, ok := parseAlSalamOCREntry(entry, previousBalance, previousBalanceKnown)
		if !ok {
			continue
		}
		pl.LineNumber = len(result.Lines) + 1
		result.Lines = append(result.Lines, pl)
		if pl.Debit > 0 {
			result.TotalDebits += pl.Debit
			result.DebitCount++
		}
		if pl.Credit > 0 {
			result.TotalCredits += pl.Credit
			result.CreditCount++
		}
		if strings.Contains(strings.ToLower(pl.Description), "opening balance") && pl.Balance != 0 {
			result.OpeningBalance = pl.Balance
		}
		if pl.Balance != 0 {
			previousBalance = pl.Balance
			previousBalanceKnown = true
		}
	}

	if len(result.Lines) == 0 {
		return nil, fmt.Errorf("no transactions found in Al Salam OCR format")
	}
	repairParsedStatementPolarity(result)
	if result.OpeningBalance == 0 {
		first := result.Lines[0]
		if first.Balance != 0 {
			result.OpeningBalance = first.Balance + first.Debit - first.Credit
		}
	}
	if result.ClosingBalance == 0 {
		result.ClosingBalance = result.Lines[len(result.Lines)-1].Balance
	}
	if result.PeriodStart.IsZero() {
		result.PeriodStart = result.Lines[0].Date
	}
	if result.PeriodEnd.IsZero() {
		result.PeriodEnd = result.Lines[len(result.Lines)-1].Date
	}

	return result, nil
}

func collectOCRBankEntries(lines []string) []string {
	return collectOCRBankEntriesWithStart(lines, regexp.MustCompile(`^\s*\d{1,2}[/-]\d{1,2}[/-]\d{2,4}\b`))
}

func collectOCRBankEntriesWithStart(lines []string, dateStartPattern *regexp.Regexp) []string {
	var entries []string
	current := ""

	for _, raw := range lines {
		line := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
		if line == "" {
			continue
		}
		if dateStartPattern.MatchString(line) {
			if current != "" {
				entries = append(entries, current)
			}
			current = line
			continue
		}
		if current == "" || shouldSkipOCRContinuation(line) {
			continue
		}
		current = strings.TrimSpace(current + " " + line)
	}

	if current != "" {
		entries = append(entries, current)
	}
	return entries
}

func shouldSkipOCRContinuation(line string) bool {
	lower := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(line)), " "))
	if lower == "" {
		return true
	}

	headerLikePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^page\s+\d+`),
		regexp.MustCompile(`^you acknowledge`),
		regexp.MustCompile(`^generated on:`),
		regexp.MustCompile(`^national bank of bahrain\b`),
		regexp.MustCompile(`^account statement\b`),
		regexp.MustCompile(`^statement summary\b`),
		regexp.MustCompile(`^transaction date(?:\s+value date)?(?:\s+description)?(?:\s+reference)?(?:\s+debit)?(?:\s+credit)?(?:\s+balance)?$`),
		regexp.MustCompile(`^value date(?:\s+description)?(?:\s+reference)?(?:\s+debit)?(?:\s+credit)?(?:\s+balance)?$`),
		regexp.MustCompile(`^description(?:\s+reference)?(?:\s+debit)?(?:\s+credit)?(?:\s+balance)?$`),
		regexp.MustCompile(`^reference(?:\s+debit)?(?:\s+credit)?(?:\s+balance)?$`),
		regexp.MustCompile(`^debit(?:\s+credit)?(?:\s+balance)?$`),
		regexp.MustCompile(`^credit(?:\s+balance)?$`),
		regexp.MustCompile(`^balance$`),
		regexp.MustCompile(`^opening balance(?:\s+[\d,]+\.\d{3})?$`),
		regexp.MustCompile(`^closing balance(?:\s+[\d,]+\.\d{3})?$`),
		regexp.MustCompile(`^total debit(?:s)?(?:\s+[\d,]+\.\d{3})?$`),
		regexp.MustCompile(`^total credit(?:s)?(?:\s+[\d,]+\.\d{3})?$`),
	}
	for _, pattern := range headerLikePatterns {
		if pattern.MatchString(lower) {
			return true
		}
	}
	return false
}

func parseNBBTransactionEntry(entry string, previousBalance float64, previousBalanceKnown bool) (parsedLine, bool) {
	dateMatch := regexp.MustCompile(`^\s*(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})\b`).FindStringSubmatch(entry)
	if len(dateMatch) < 2 {
		return parsedLine{}, false
	}

	date := parseFlexibleDate(dateMatch[1])
	body := strings.TrimSpace(strings.TrimPrefix(entry, dateMatch[0]))
	amountPattern := regexp.MustCompile(`(?:\(-\)|-|\()?\s*[\d,]+\.\d{2,3}\)?`)
	amountSource := body
	descriptionPart := body

	if currencyMatches := regexp.MustCompile(`\b(?:BHD|USD|EUR|GBP|SAR)\b`).FindAllStringIndex(body, -1); len(currencyMatches) > 0 {
		lastCurrency := currencyMatches[len(currencyMatches)-1]
		descriptionPart = strings.TrimSpace(body[:lastCurrency[0]])
		amountSource = strings.TrimSpace(body[lastCurrency[1]:])
	}

	rawAmounts := amountPattern.FindAllString(amountSource, -1)
	amountIndexes := amountPattern.FindAllStringIndex(amountSource, -1)
	if len(rawAmounts) == 0 || len(amountIndexes) == 0 {
		return parsedLine{}, false
	}

	if descriptionPart == body {
		descriptionPart = strings.TrimSpace(body[:amountIndexes[0][0]])
	}
	descriptionPart = regexp.MustCompile(`\s+(?:BHD|USD|EUR|GBP|SAR)$`).ReplaceAllString(descriptionPart, "")
	reference := ""
	if refMatch := regexp.MustCompile(`\b\d{6,}\b`).FindString(descriptionPart); refMatch != "" {
		reference = refMatch
		descriptionPart = strings.TrimSpace(strings.Replace(descriptionPart, refMatch, "", 1))
	}
	descriptionPart = strings.Trim(descriptionPart, "- ")

	balance, _ := parseSignedAmountString(rawAmounts[len(rawAmounts)-1])
	var debit, credit float64

	switch len(rawAmounts) {
	case 1:
		// Balance-only rows, such as opening balance, are preserved with zero amounts.
	case 2:
		amount, negative := parseSignedAmountString(rawAmounts[0])
		if negative {
			debit = amount
		} else if previousBalanceKnown && balance != 0 {
			delta := balance - previousBalance
			if math.Abs(math.Abs(delta)-amount) <= 0.051 {
				if delta >= 0 {
					credit = amount
				} else {
					debit = amount
				}
			}
		}
		if debit == 0 && credit == 0 {
			switch inferBankAmountDirection(descriptionPart) {
			case 1:
				credit = amount
			case -1:
				debit = amount
			}
		}
	case 3:
		debit, _ = parseSignedAmountString(rawAmounts[0])
		credit, _ = parseSignedAmountString(rawAmounts[1])
	default:
		debit, _ = parseSignedAmountString(rawAmounts[0])
		credit, _ = parseSignedAmountString(rawAmounts[1])
	}

	pl := parsedLine{
		Date:        date,
		ValueDate:   date,
		Reference:   reference,
		Description: descriptionPart,
		Debit:       debit,
		Credit:      credit,
		Balance:     balance,
	}

	return pl, pl.Description != "" || pl.Balance != 0 || pl.Debit != 0 || pl.Credit != 0
}

func parseSignedAmountString(s string) (float64, bool) {
	trimmed := strings.TrimSpace(s)
	negative := strings.Contains(trimmed, "(-)") || strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "(")
	trimmed = strings.ReplaceAll(trimmed, "(-)", "")
	trimmed = strings.TrimPrefix(trimmed, "-")
	trimmed = strings.Trim(trimmed, "() ")
	return parseAmount(trimmed), negative
}

func parseKFHTransactionEntry(entry string, previousBalance float64, previousBalanceKnown bool) (parsedLine, bool) {
	match := regexp.MustCompile(`^\s*(\d{2}/\d{2}/\d{4})\s+(\d{2}/\d{2}/\d{4})\s+(.*?)\s+([\d,]+\.\d{3})\s+([\d,]+\.\d{3})\s*$`).FindStringSubmatch(entry)
	if len(match) != 6 {
		return parsedLine{}, false
	}

	date := parseFlexibleDate(match[1])
	valueDate := parseFlexibleDate(match[2])
	description := strings.TrimSpace(match[3])
	amount := parseAmount(match[4])
	balance := parseAmount(match[5])
	if description == "" {
		return parsedLine{}, false
	}

	debit, credit := inferAmountByDescriptionOrBalance(description, amount, balance, previousBalance, previousBalanceKnown)
	reference := extractKFHReference(description)

	return parsedLine{
		Date:        date,
		ValueDate:   valueDate,
		Reference:   reference,
		Description: description,
		Debit:       debit,
		Credit:      credit,
		Balance:     balance,
	}, true
}

type kfhColumnarDescriptor struct {
	Date        time.Time
	ValueDate   time.Time
	Description string
	Reference   string
}

func parseKFHColumnarLayout(lines []string, openingBalance float64, openingBalanceKnown bool) []parsedLine {
	normalizedLines := make([]string, 0, len(lines))
	for _, rawLine := range lines {
		line := strings.Join(strings.Fields(strings.TrimSpace(rawLine)), " ")
		if line == "" {
			continue
		}
		normalizedLines = append(normalizedLines, line)
	}

	descriptors := extractKFHColumnarDescriptors(normalizedLines)
	amountPairs := extractKFHAmountBalancePairs(normalizedLines)
	limit := len(descriptors)
	if len(amountPairs) < limit {
		limit = len(amountPairs)
	}

	previousBalance := openingBalance
	previousBalanceKnown := openingBalanceKnown
	parsedLines := make([]parsedLine, 0, limit)
	for idx := 0; idx < limit; idx++ {
		descriptor := descriptors[idx]
		amount := amountPairs[idx][0]
		balance := amountPairs[idx][1]
		debit, credit := inferAmountByDescriptionOrBalance(descriptor.Description, amount, balance, previousBalance, previousBalanceKnown)
		parsedLines = append(parsedLines, parsedLine{
			Date:        descriptor.Date,
			ValueDate:   descriptor.ValueDate,
			Reference:   descriptor.Reference,
			Description: descriptor.Description,
			Debit:       debit,
			Credit:      credit,
			Balance:     balance,
		})
		if balance != 0 {
			previousBalance = balance
			previousBalanceKnown = true
		}
	}

	return parsedLines
}

func extractKFHColumnarDescriptors(lines []string) []kfhColumnarDescriptor {
	descriptors := []kfhColumnarDescriptor{}
	datePattern := regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`)

	for idx := 0; idx+1 < len(lines); {
		if !datePattern.MatchString(lines[idx]) || !datePattern.MatchString(lines[idx+1]) {
			idx++
			continue
		}

		date := parseFlexibleDate(lines[idx])
		valueDate := parseFlexibleDate(lines[idx+1])
		idx += 2

		descriptionParts := make([]string, 0, 2)
		for idx < len(lines) {
			line := lines[idx]
			if idx+1 < len(lines) && datePattern.MatchString(line) && datePattern.MatchString(lines[idx+1]) {
				break
			}
			if isStandaloneAmountValue(line) || shouldSkipKFHColumnarLine(line) {
				idx++
				continue
			}
			descriptionParts = append(descriptionParts, line)
			idx++
		}

		description := strings.TrimSpace(strings.Join(descriptionParts, " "))
		description = regexp.MustCompile(`\s+`).ReplaceAllString(description, " ")
		if description == "" {
			continue
		}

		descriptors = append(descriptors, kfhColumnarDescriptor{
			Date:        date,
			ValueDate:   valueDate,
			Description: description,
			Reference:   extractKFHReference(description),
		})
	}

	return descriptors
}

func extractKFHAmountBalancePairs(lines []string) [][2]float64 {
	headerIndex := -1
	for idx, line := range lines {
		lower := strings.ToLower(line)
		if lower == "amount" || lower == "balance" || (strings.Contains(lower, "amount") && strings.Contains(lower, "balance")) {
			headerIndex = idx
			break
		}
	}
	if headerIndex < 0 {
		return nil
	}

	values := make([]float64, 0, 8)
	for _, line := range lines[headerIndex+1:] {
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "you acknowledge") || strings.HasPrefix(lower, "generated on") || strings.HasPrefix(lower, "page ") {
			break
		}
		if isStandaloneAmountValue(line) {
			values = append(values, parseAmount(line))
		}
	}

	pairs := make([][2]float64, 0, len(values)/2)
	for idx := 0; idx+1 < len(values); idx += 2 {
		pairs = append(pairs, [2]float64{values[idx], values[idx+1]})
	}
	return pairs
}

func shouldSkipKFHColumnarLine(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	if lower == "" {
		return true
	}
	if !regexp.MustCompile(`[A-Za-z0-9]`).MatchString(line) {
		return true
	}

	skipPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^account history$`),
		regexp.MustCompile(`^p h trading w\.l\.l$`),
		regexp.MustCompile(`^acme instrumentation spc$`),
		regexp.MustCompile(`^account number:?$`),
		regexp.MustCompile(`^account type:?$`),
		regexp.MustCompile(`^currency:?$`),
		regexp.MustCompile(`^from - to:?$`),
		regexp.MustCompile(`^opening balance:?$`),
		regexp.MustCompile(`^closing balance:?$`),
		regexp.MustCompile(`^current balance:?$`),
		regexp.MustCompile(`^blocked amount:?$`),
		regexp.MustCompile(`^uncleared cheque:?$`),
		regexp.MustCompile(`^available balance:?$`),
		regexp.MustCompile(`^date$`),
		regexp.MustCompile(`^value date$`),
		regexp.MustCompile(`^description$`),
		regexp.MustCompile(`^amount$`),
		regexp.MustCompile(`^balance$`),
		regexp.MustCompile(`^iban:?$`),
		regexp.MustCompile(`^page \d+ of \d+$`),
		regexp.MustCompile(`^generated on:`),
		regexp.MustCompile(`^you acknowledge`),
	}
	for _, pattern := range skipPatterns {
		if pattern.MatchString(lower) {
			return true
		}
	}

	return strings.Contains(lower, "all rights reserved")
}

func isStandaloneAmountValue(line string) bool {
	return regexp.MustCompile(`^[\d,]+\.\d{3}$`).MatchString(strings.TrimSpace(line))
}

func extractKFHReference(description string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b[A-Z]{2,}[A-Z0-9/-]{4,}\b`),
		regexp.MustCompile(`\b[A-Z0-9/-]{8,}\b`),
	}
	for _, pattern := range patterns {
		if ref := pattern.FindString(description); ref != "" {
			return ref
		}
	}
	return ""
}

func parseAlSalamOCREntry(entry string, previousBalance float64, previousBalanceKnown bool) (parsedLine, bool) {
	match := regexp.MustCompile(`^\s*(\d{2}\s+[A-Za-z]{3}\s+\d{4})\s+(.*?)\s+(\d{2}\s+[A-Za-z]{3}\s+\d{4})\s+(\(?-?[\d,]+\.\d{3}\)?)\s+([\d,]+\.\d{3})\s*$`).FindStringSubmatch(entry)
	if len(match) != 6 {
		return parsedLine{}, false
	}

	date := parseFlexibleDate(match[1])
	valueDate := parseFlexibleDate(match[3])
	description := strings.TrimSpace(match[2])
	amountRaw := strings.TrimSpace(match[4])
	balance := parseAmount(match[5])

	var debit, credit float64
	switch {
	case strings.HasPrefix(amountRaw, "(-)"):
		debit = parseAmount(strings.TrimPrefix(amountRaw, "(-)"))
	case strings.HasPrefix(amountRaw, "(") && strings.HasSuffix(amountRaw, ")"):
		debit = parseAmount(strings.Trim(amountRaw, "()"))
	case strings.HasPrefix(amountRaw, "-"):
		debit = parseAmount(strings.TrimPrefix(amountRaw, "-"))
	default:
		amount := parseAmount(strings.Trim(amountRaw, "()"))
		debit, credit = inferAmountByDescriptionOrBalance(description, amount, balance, previousBalance, previousBalanceKnown)
	}

	reference := ""
	if ref := regexp.MustCompile(`\b[A-Z0-9-]{6,}\b`).FindString(description); ref != "" && !strings.Contains(strings.ToLower(description), "opening balance") && !strings.Contains(strings.ToLower(description), "closing balance") {
		reference = ref
	}

	return parsedLine{
		Date:        date,
		ValueDate:   valueDate,
		Reference:   reference,
		Description: description,
		Debit:       debit,
		Credit:      credit,
		Balance:     balance,
	}, true
}

func inferAmountByDescriptionOrBalance(description string, amount float64, balance float64, previousBalance float64, previousBalanceKnown bool) (float64, float64) {
	if amount == 0 {
		return 0, 0
	}
	if previousBalanceKnown && balance != 0 {
		delta := balance - previousBalance
		if math.Abs(math.Abs(delta)-amount) <= 0.051 {
			if delta >= 0 {
				return 0, amount
			}
			return amount, 0
		}
	}
	switch inferBankAmountDirection(description) {
	case 1:
		return 0, amount
	case -1:
		return amount, 0
	default:
		return 0, amount
	}
}

func inferBankAmountDirection(description string) int {
	lower := strings.ToLower(description)

	creditHints := []string{
		"trf in", "transfer in", "transfer from", "receipt", "received from", "deposit", "cash deposit",
		"fawri ordinary transfer", "inwd", "inward", "credit advice", "salary credit",
		"sundry credit", "payment | from", "payment from", "cheque collected", "cheque collection",
	}
	for _, hint := range creditHints {
		if strings.Contains(lower, hint) {
			return 1
		}
	}

	debitHints := []string{
		"trf out", "transfer to", "payment to", "withdrawal", "bank fee", "charges", "charge", "vat",
		"swift", "corr bnk chg", "commission", "cheque", "chq", "dd", "atm", "dgc", "onus",
		"fawri to", "trf from", "debit(tf)", "debit",
	}
	for _, hint := range debitHints {
		if strings.Contains(lower, hint) {
			return -1
		}
	}

	return 0
}

func normalizeParsedStatementPolarity(result *parsedStatement) {
	if result == nil || len(result.Lines) == 0 {
		return
	}

	previousBalance := result.OpeningBalance
	previousBalanceKnown := result.OpeningBalance != 0

	for idx := range result.Lines {
		line := &result.Lines[idx]
		if line.Balance == 0 {
			if line.Credit > 0 || line.Debit > 0 {
				if previousBalanceKnown {
					line.Balance = previousBalance + line.Credit - line.Debit
				}
			}
		}

		if line.Balance != 0 && previousBalanceKnown {
			delta := line.Balance - previousBalance
			expectedDebit := math.Abs(delta)

			switch {
			case line.Debit > 0 && line.Credit > 0:
				if math.Abs(delta) > 0.051 {
					if delta >= 0 && math.Abs(line.Credit-expectedDebit) <= 0.051 {
						line.Debit = 0
					} else if delta < 0 && math.Abs(line.Debit-expectedDebit) <= 0.051 {
						line.Credit = 0
					}
				}
			case line.Debit == 0 && line.Credit == 0:
				if math.Abs(delta) > 0.051 {
					if delta >= 0 {
						line.Credit = expectedDebit
					} else {
						line.Debit = expectedDebit
					}
				}
			default:
				amount := line.Credit
				if amount == 0 {
					amount = line.Debit
				}
				if math.Abs(delta) > 0.051 && math.Abs(expectedDebit-amount) <= 0.051 {
					if delta >= 0 {
						line.Credit = amount
						line.Debit = 0
					} else {
						line.Debit = amount
						line.Credit = 0
					}
				}
			}
		} else if line.Debit == 0 && line.Credit > 0 && inferBankAmountDirection(line.Description) == -1 {
			line.Debit = line.Credit
			line.Credit = 0
		} else if line.Credit == 0 && line.Debit > 0 && inferBankAmountDirection(line.Description) == 1 {
			line.Credit = line.Debit
			line.Debit = 0
		}

		if line.Balance != 0 {
			previousBalance = line.Balance
			previousBalanceKnown = true
		}
	}

	result.TotalDebits = 0
	result.TotalCredits = 0
	result.DebitCount = 0
	result.CreditCount = 0
	for _, line := range result.Lines {
		if line.Debit > 0 {
			result.TotalDebits += line.Debit
			result.DebitCount++
		}
		if line.Credit > 0 {
			result.TotalCredits += line.Credit
			result.CreditCount++
		}
	}
}

func repairParsedStatementPolarity(result *parsedStatement) {
	if result == nil || len(result.Lines) == 0 {
		return
	}

	headerDebitTotal := result.TotalDebits
	headerCreditTotal := result.TotalCredits
	headerDebitCount := result.DebitCount
	headerCreditCount := result.CreditCount
	original := cloneParsedStatement(result)
	normalizeParsedStatementPolarity(result)

	swapped := cloneParsedStatement(original)
	swapParsedStatementAmounts(swapped)
	normalizeParsedStatementPolarity(swapped)

	currentScore := scoreParsedStatementPolarity(result, headerDebitTotal, headerCreditTotal, headerDebitCount, headerCreditCount)
	swappedScore := scoreParsedStatementPolarity(swapped, headerDebitTotal, headerCreditTotal, headerDebitCount, headerCreditCount)
	if swappedScore+1 < currentScore {
		*result = *swapped
	}
}

func cloneParsedStatement(result *parsedStatement) *parsedStatement {
	if result == nil {
		return nil
	}

	clone := *result
	if len(result.Lines) > 0 {
		clone.Lines = make([]parsedLine, len(result.Lines))
		copy(clone.Lines, result.Lines)
	}
	return &clone
}

func swapParsedStatementAmounts(result *parsedStatement) {
	if result == nil {
		return
	}
	for idx := range result.Lines {
		result.Lines[idx].Debit, result.Lines[idx].Credit = result.Lines[idx].Credit, result.Lines[idx].Debit
	}
}

func scoreParsedStatementPolarity(result *parsedStatement, headerDebitTotal float64, headerCreditTotal float64, headerDebitCount int, headerCreditCount int) float64 {
	if result == nil || len(result.Lines) == 0 {
		return math.MaxFloat64
	}

	candidate := cloneParsedStatement(result)
	candidate.TotalDebits = headerDebitTotal
	candidate.TotalCredits = headerCreditTotal
	candidate.DebitCount = headerDebitCount
	candidate.CreditCount = headerCreditCount
	validation := validateParsedStatement(candidate)

	score := 0.0
	if validation.Blocking {
		score += 1000
	}
	score += float64(len(validation.BlockingIssues)) * 250
	score += float64(validation.BalanceMismatchCount) * 100
	score += float64(validation.BothSidesCount) * 150
	score += float64(validation.AmbiguousAmountCount) * 40
	score += float64(validation.ZeroDateCount) * 25

	if candidate.OpeningBalance != 0 && candidate.ClosingBalance != 0 {
		expectedClosing := candidate.OpeningBalance + candidate.TotalCredits - candidate.TotalDebits
		score += absFloat(expectedClosing-candidate.ClosingBalance) * 10
	}

	return score
}

// =============================================================================
// CSV PARSER
// =============================================================================

// parseCSVRecords parses CSV records into a statement structure
// Supports multiple formats:
// 1. Al Salam format: Header metadata, then Posting Date, Value Date, FT Reference, Description, Amount, Balance
// 2. Generic format: Date, Description, Debit, Credit, Reference, Balance
func parseCSVRecords(records [][]string) (*parsedStatement, error) {
	result := &parsedStatement{
		Currency: "BHD",
	}

	// Try to detect Al Salam format (has metadata rows at top)
	if isAlSalamFormat(records) {
		return parseAlSalamCSV(records)
	}

	// Generic CSV format: Date, Description, Debit, Credit, Reference (optional), Balance (optional)
	var openingBalance, runningBalance float64

	for i, record := range records[1:] { // Skip header row
		if len(record) < 4 {
			continue
		}

		date := parseFlexibleDate(strings.TrimSpace(record[0]))
		description := strings.TrimSpace(record[1])
		debit := parseAmount(record[2])
		credit := parseAmount(record[3])

		reference := ""
		if len(record) > 4 {
			reference = strings.TrimSpace(record[4])
		}

		var balance float64
		if len(record) > 5 {
			balance = parseAmount(record[5])
		} else {
			runningBalance = runningBalance + credit - debit
			balance = runningBalance
		}

		if i == 0 {
			openingBalance = balance - credit + debit
		}

		pl := parsedLine{
			LineNumber:  i + 1,
			Date:        date,
			ValueDate:   date,
			Reference:   reference,
			Description: description,
			Debit:       debit,
			Credit:      credit,
			Balance:     balance,
		}

		if pl.Debit > 0 {
			result.DebitCount++
			result.TotalDebits += pl.Debit
		}
		if pl.Credit > 0 {
			result.CreditCount++
			result.TotalCredits += pl.Credit
		}

		result.Lines = append(result.Lines, pl)
	}

	if len(result.Lines) == 0 {
		return nil, fmt.Errorf("no valid transaction rows in CSV")
	}

	result.PeriodStart = result.Lines[0].Date
	result.PeriodEnd = result.Lines[len(result.Lines)-1].Date
	result.OpeningBalance = openingBalance
	result.ClosingBalance = result.Lines[len(result.Lines)-1].Balance

	return result, nil
}

// isAlSalamFormat detects Al Salam Bank CSV format
func isAlSalamFormat(records [][]string) bool {
	// Check for metadata rows like "Statement of Account", "Client Name", "IBAN"
	for i := 0; i < len(records) && i < 10; i++ {
		if len(records[i]) > 0 {
			firstCell := strings.TrimSpace(records[i][0])
			if strings.Contains(firstCell, "Statement of Account") ||
				strings.Contains(firstCell, "Client Name") ||
				strings.Contains(firstCell, "IBAN") {
				return true
			}
		}
	}
	return false
}

// parseAlSalamCSV parses Al Salam Bank CSV format
// Format: Posting Date, Value Date, FT Reference, Description, Amount, Balance
// Amount uses (-) prefix for debits
func parseAlSalamCSV(records [][]string) (*parsedStatement, error) {
	result := &parsedStatement{
		Currency: "BHD",
	}

	// Extract metadata from header rows
	for _, record := range records {
		if len(record) < 2 {
			continue
		}
		key := strings.TrimSpace(record[0])
		value := strings.TrimSpace(record[1])

		switch key {
		case "From":
			result.PeriodStart = parseFlexibleDate(value)
		case "To":
			result.PeriodEnd = parseFlexibleDate(value)
		case "IBAN":
			result.IBAN = value
		case "Account Currency":
			if strings.Contains(strings.ToLower(value), "euro") {
				result.Currency = "EUR"
			} else if strings.Contains(strings.ToLower(value), "usd") || strings.Contains(strings.ToLower(value), "dollar") {
				result.Currency = "USD"
			}
		case "Total Credits":
			result.TotalCredits = parseAmount(value)
		case "Total Debits":
			result.TotalDebits = parseAmount(value)
		}
	}

	// Find the header row (contains "Posting Date")
	headerRowIdx := -1
	for i, record := range records {
		if len(record) > 0 && strings.TrimSpace(record[0]) == "Posting Date" {
			headerRowIdx = i
			break
		}
	}

	if headerRowIdx == -1 {
		return nil, fmt.Errorf("could not find header row in Al Salam CSV")
	}

	// Parse transaction rows
	lineNum := 0
	for i := headerRowIdx + 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 5 {
			continue
		}

		// Skip empty rows
		postingDate := strings.TrimSpace(record[0])
		if postingDate == "" {
			// Check for Opening/Closing Balance rows
			desc := ""
			if len(record) > 3 {
				desc = strings.TrimSpace(record[3])
			}
			if strings.Contains(desc, "Opening Balance") {
				if len(record) > 5 {
					result.OpeningBalance = parseAmount(record[5])
				}
			} else if strings.Contains(desc, "Closing Balance") {
				if len(record) > 5 {
					result.ClosingBalance = parseAmount(record[5])
				}
			}
			continue
		}

		lineNum++
		date := parseFlexibleDate(postingDate)
		valueDate := parseFlexibleDate(strings.TrimSpace(record[1]))
		reference := strings.TrimSpace(record[2])
		description := strings.TrimSpace(record[3])
		amountStr := strings.TrimSpace(record[4])
		balanceStr := ""
		if len(record) > 5 {
			balanceStr = strings.TrimSpace(record[5])
		}

		// Parse amount - (-) prefix means debit
		var debit, credit float64
		if strings.HasPrefix(amountStr, "(-)") {
			debit = parseAmount(strings.TrimPrefix(amountStr, "(-)"))
			result.DebitCount++
		} else if strings.HasPrefix(amountStr, "-") {
			debit = parseAmount(strings.TrimPrefix(amountStr, "-"))
			result.DebitCount++
		} else if amountStr != "" {
			credit = parseAmount(amountStr)
			if credit > 0 {
				result.CreditCount++
			}
		}

		balance := parseAmount(balanceStr)

		pl := parsedLine{
			LineNumber:  lineNum,
			Date:        date,
			ValueDate:   valueDate,
			Reference:   reference,
			Description: description,
			Debit:       debit,
			Credit:      credit,
			Balance:     balance,
		}

		result.Lines = append(result.Lines, pl)
	}

	if len(result.Lines) == 0 {
		return nil, fmt.Errorf("no transaction rows found in Al Salam CSV")
	}

	// Infer opening/closing if not found
	if result.OpeningBalance == 0 && len(result.Lines) > 0 {
		first := result.Lines[0]
		result.OpeningBalance = first.Balance + first.Debit - first.Credit
	}
	if result.ClosingBalance == 0 && len(result.Lines) > 0 {
		result.ClosingBalance = result.Lines[len(result.Lines)-1].Balance
	}

	log.Printf("📊 Parsed Al Salam CSV: %d lines, Opening: %.3f, Closing: %.3f",
		len(result.Lines), result.OpeningBalance, result.ClosingBalance)

	return result, nil
}

// =============================================================================
// EXTRACTION HELPERS
// =============================================================================

// extractNumber extracts a numeric string from a line
func extractNumber(line string) string {
	re := regexp.MustCompile(`\d+`)
	return re.FindString(line)
}

// extractAmount extracts a BHD amount (3 decimal places) from a line
func extractAmount(line string) float64 {
	re := regexp.MustCompile(`[\d,]+\.\d{2,3}`)
	match := re.FindString(line)
	return parseAmount(match)
}

func extractSummaryCountAndAmount(line string) (int, float64) {
	count := 0
	amount := extractAmount(line)

	if match := regexp.MustCompile(`(?i)total\s+(?:debits|credits)\s+(\d+)\s+for\s+([\d,]+\.\d{2,3})`).FindStringSubmatch(line); len(match) == 3 {
		count, _ = strconv.Atoi(match[1])
		amount = parseAmount(match[2])
		return count, amount
	}

	if match := regexp.MustCompile(`(?i)total\s+(?:debits|credits)\s+(\d+)`).FindStringSubmatch(line); len(match) == 2 {
		count, _ = strconv.Atoi(match[1])
	}

	return count, amount
}

// parseAmount converts a string amount to float64
func parseAmount(s string) float64 {
	if s == "" {
		return 0
	}
	s = strings.ReplaceAll(s, ",", "")
	amount, _ := strconv.ParseFloat(s, 64)
	return amount
}

// extractPeriod extracts start and end dates from a period string
func extractPeriod(line string) (start, end time.Time) {
	datePattern := regexp.MustCompile(`\d{1,2}[/-]\d{1,2}[/-]\d{2,4}`)
	dates := datePattern.FindAllString(line, -1)

	if len(dates) >= 2 {
		start = parseFlexibleDate(dates[0])
		end = parseFlexibleDate(dates[1])
	} else if len(dates) == 1 {
		end = parseFlexibleDate(dates[0])
		start = time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	return
}

// parseFlexibleDate parses dates in various formats
func parseFlexibleDate(s string) time.Time {
	formats := []string{
		"02/01/2006", "2/1/2006",
		"02-01-2006", "2-1-2006",
		"02 Jan 2006", "2 Jan 2006",
		"2006-01-02", "2006/01/02",
		"01/02/2006", "1/2/2006",
		"02/01/06", "2/1/06",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			// Handle 2-digit years
			if t.Year() < 100 {
				t = t.AddDate(2000, 0, 0)
			}
			return t
		}
	}

	return time.Time{}
}

func extractCurrencyCode(line string) string {
	if match := regexp.MustCompile(`\b(BHD|USD|EUR|GBP|SAR|AED|KWD|QAR|OMR)\b`).FindStringSubmatch(strings.ToUpper(line)); len(match) == 2 {
		return match[1]
	}
	return ""
}
