package banking

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"
	"time"

	"ph_holdings_app/pkg/finance"
	"ph_holdings_app/pkg/overlay"

	"gorm.io/gorm"
)

const (
	TxTypeCustomerPayment     = "CUSTOMER_PAYMENT"
	TxTypeChequeReceived      = "CHEQUE_RECEIVED"
	TxTypeSupplierPayment     = "SUPPLIER_PAYMENT"
	TxTypeChequeIssued        = "CHEQUE_ISSUED"
	TxTypeBankFee             = "BANK_FEE"
	TxTypeSwiftCharge         = "SWIFT_CHARGE"
	TxTypeVATOnFee            = "VAT_ON_FEE"
	TxTypeBGFee               = "BG_FEE"
	TxTypeCorrespondentCharge = "CORRESPONDENT_CHARGE"
	TxTypeInternalTransfer    = "INTERNAL_TRANSFER"
	TxTypeFXConversion        = "FX_CONVERSION"
	TxTypeSalary              = "SALARY"
	TxTypeBillPayment         = "BILL_PAYMENT"
	TxTypeOther               = "OTHER"
)

type BankReconciliationMatchResult struct {
	MatchedCount     int     `json:"matched_count"`
	UnmatchedCount   int     `json:"unmatched_count"`
	TotalLines       int     `json:"total_lines"`
	MatchedPercent   float64 `json:"matched_percent"`
	AutoMatchedCount int     `json:"auto_matched_count"`
}

type AllocationInput struct {
	AllocationType  string  `json:"allocation_type"`
	EntityID        string  `json:"entity_id"`
	AllocatedAmount float64 `json:"allocated_amount"`
}

type matchCustomer struct {
	ID           string
	BusinessName string
}

func (matchCustomer) TableName() string { return "customers" }

type matchSupplier struct {
	ID           string
	SupplierName string
}

func (matchSupplier) TableName() string { return "suppliers" }

type matchInvoice struct {
	ID             string
	InvoiceNumber  string
	Division       string
	Status         string
	CustomerID     string
	OutstandingBHD float64
	GrandTotalBHD  float64
}

func (matchInvoice) TableName() string { return "invoices" }

type matchSupplierInvoice struct {
	ID              string
	PONumber        string
	Division        string
	Status          string
	SupplierID      string
	OrderID         string
	PurchaseOrderID string
	TotalBHD        float64
}

func (matchSupplierInvoice) TableName() string { return "supplier_invoices" }

type matchSupplierPayment struct {
	ID        string
	Division  string
	AmountBHD float64
}

func (matchSupplierPayment) TableName() string { return "supplier_payments" }

type matchExpenseEntry struct {
	ID            string
	EntryNumber   string
	Division      string
	Status        string
	PaymentStatus string
	TotalAmount   float64
	Amount        float64
	VATAmount     float64
}

func (matchExpenseEntry) TableName() string { return "expense_entries" }

type matchPayrollPayout struct {
	ID                  string
	PayrollRunID        string
	Amount              float64
	BankStatementLineID *string
}

func (matchPayrollPayout) TableName() string { return "payroll_payouts" }

type matchPayrollRun struct {
	ID                   string
	PayoutJournalEntryID *string
}

func (matchPayrollRun) TableName() string { return "payroll_runs" }

type matchOrder struct {
	ID       string
	Division string
}

func (matchOrder) TableName() string { return "orders" }

type matchPurchaseOrder struct {
	ID       string
	OrderID  string
	Division string
}

func (matchPurchaseOrder) TableName() string { return "purchase_orders" }

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) AutoMatchBankLines(statementID string) (*BankReconciliationMatchResult, error) {
	if err := s.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var statement finance.BankStatement
	if err := s.db.First(&statement, "id = ?", statementID).Error; err != nil {
		return nil, fmt.Errorf("statement not found: %w", err)
	}
	// PC-D4: refuse matching on any finalized statement (Reconciled or
	// Verified); reopen the reconciliation first.
	if IsFinalStatementStatus(statement.Status) {
		return nil, fmt.Errorf("cannot match transactions on a %s statement; reopen the reconciliation first", strings.ToLower(DisplayStatementStatus(statement.Status)))
	}

	var lines []finance.BankStatementLine
	s.db.Where("bank_statement_id = ? AND is_matched = ?", statementID, false).Find(&lines)

	var customerNames []string
	var supplierNames []string
	s.db.Model(&matchCustomer{}).Pluck("business_name", &customerNames)
	s.db.Model(&matchSupplier{}).Pluck("supplier_name", &supplierNames)

	matchedCount := 0
	autoMatchedCount := 0
	statementDivision := normalizeDivisionName(statement.Division)
	if strings.TrimSpace(statementDivision) == "" {
		statementDivision = resolveBankAccountDivisionTx(s.db, statement.BankAccountID)
	}

	for i := range lines {
		line := &lines[i]
		isCredit := line.Credit > 0
		txType := detectTransactionType(line.Description, isCredit)
		invoiceNumbers := extractInvoiceNumbers(line.Description)
		poNumbers := extractPONumbers(line.Description)
		customerName, supplierName := extractBankEntityName(line.Description, customerNames, supplierNames)

		updates := map[string]any{
			"transaction_type":     txType,
			"extracted_invoices":   toJSONArray(invoiceNumbers),
			"extracted_po_numbers": toJSONArray(poNumbers),
			"extracted_customer":   customerName,
			"extracted_supplier":   supplierName,
		}

		var matched bool
		var matchedID string
		var confidence float64

		switch txType {
		case TxTypeCustomerPayment, TxTypeChequeReceived:
			matched, matchedID, confidence = matchToCustomerInvoice(s.db, line, invoiceNumbers, customerName, statementDivision)
		case TxTypeSupplierPayment, TxTypeChequeIssued:
			matched, matchedID, confidence = matchToSupplierPayment(s.db, line, poNumbers, supplierName, statementDivision)
		case TxTypeBankFee, TxTypeSwiftCharge, TxTypeVATOnFee, TxTypeBGFee, TxTypeCorrespondentCharge:
			_ = createExpenseFromBankLine(s.db, line, txType)
			matched = true
			confidence = 1.0
		}

		if matched {
			updates["is_matched"] = true
			updates["match_type"] = "Auto"
			updates["match_confidence"] = confidence
			updates["matched_invoice_ids"] = toJSONArray([]string{matchedID})
			matchedCount++
			autoMatchedCount++

			lineID := line.ID
			_ = s.LogReconciliationAction(statementID, &lineID, "MATCH",
				map[string]any{
					"type":       "Auto",
					"tx_type":    txType,
					"matched_to": matchedID,
					"confidence": confidence,
				},
				"System", true, confidence, "Auto-matched")
		}

		s.db.Model(line).Updates(updates)
	}

	var totalLines, finalMatchedCount int64
	s.db.Model(&finance.BankStatementLine{}).Where("bank_statement_id = ?", statementID).Count(&totalLines)
	s.db.Model(&finance.BankStatementLine{}).Where("bank_statement_id = ? AND is_matched = ?", statementID, true).Count(&finalMatchedCount)

	matchedPercent := 0.0
	if totalLines > 0 {
		matchedPercent = float64(finalMatchedCount) / float64(totalLines) * 100
	}

	newStatus := "InProgress"
	if finalMatchedCount == totalLines {
		newStatus = "InProgress"
	}
	s.db.Model(&statement).Update("status", newStatus)

	log.Printf("Auto-matched %d/%d lines (%.1f%%) for statement %s", matchedCount, totalLines, matchedPercent, statementID)

	return &BankReconciliationMatchResult{
		MatchedCount:     int(finalMatchedCount),
		UnmatchedCount:   int(totalLines - finalMatchedCount),
		TotalLines:       int(totalLines),
		MatchedPercent:   matchedPercent,
		AutoMatchedCount: autoMatchedCount,
	}, nil
}

func detectTransactionType(description string, isCredit bool) string {
	desc := strings.ToUpper(description)

	if isCredit {
		if strings.Contains(desc, "FAWRI") && strings.Contains(desc, "TRANSFER") {
			return TxTypeCustomerPayment
		}
		if strings.Contains(desc, "CHEQUE") || strings.Contains(desc, "CHQ") {
			return TxTypeChequeReceived
		}
		if strings.Contains(desc, "INTERNAL") || strings.Contains(desc, "NBB TRF FROM") {
			return TxTypeInternalTransfer
		}
		return TxTypeCustomerPayment
	}

	if strings.Contains(desc, "SWIFT FEE") || strings.Contains(desc, "SWIFT CHG") {
		return TxTypeSwiftCharge
	}
	if strings.Contains(desc, "VAT") && (strings.Contains(desc, "FEE") || strings.Contains(desc, "CHARGE")) {
		return TxTypeVATOnFee
	}
	if strings.Contains(desc, "EFTS CHARGE") || strings.Contains(desc, "SERVICE CHARGE") {
		return TxTypeBankFee
	}
	if strings.Contains(desc, "AMENDBG") || strings.Contains(desc, "BGG") || strings.Contains(desc, "GUARANTEE") {
		return TxTypeBGFee
	}
	if strings.Contains(desc, "CORR BNK CHG") || strings.Contains(desc, "CORRESPONDENT") {
		return TxTypeCorrespondentCharge
	}
	if strings.Contains(desc, "NBB TRF FROM") && strings.Contains(desc, "TO") {
		return TxTypeInternalTransfer
	}
	if strings.Contains(desc, "FAWATEER") || strings.Contains(desc, "NBL-") {
		return TxTypeBillPayment
	}
	if strings.Contains(desc, "CHQ NO") || strings.Contains(desc, "CHEQUE") {
		return TxTypeChequeIssued
	}
	if strings.Contains(desc, "DGC") || strings.Contains(desc, "FAWRI") || strings.Contains(desc, "REMITTANCE") {
		return TxTypeSupplierPayment
	}
	if strings.Contains(desc, "SALARY") || strings.Contains(desc, "PAYROLL") {
		return TxTypeSalary
	}
	return TxTypeOther
}

func extractInvoiceNumbers(description string) []string {
	var invoices []string
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)/INV/([A-Z0-9/-]+)`),
		regexp.MustCompile(`(?i)PH(\d{2})[/-]?(\d{3,4})`),
		regexp.MustCompile(`(?i)INV[/-]?(\d{4})[/-](\d{3,4})`),
	}
	for _, pattern := range patterns {
		matches := pattern.FindAllString(description, -1)
		invoices = append(invoices, matches...)
	}

	seen := make(map[string]bool)
	unique := []string{}
	for _, inv := range invoices {
		if !seen[inv] {
			seen[inv] = true
			unique = append(unique, inv)
		}
	}
	return unique
}

func extractPONumbers(description string) []string {
	var pos []string
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)PO[/-]?(\d{4})[/-](\d{3,4})`),
		regexp.MustCompile(`(?i)PUR[/-]?(\d+)`),
	}
	for _, pattern := range patterns {
		matches := pattern.FindAllString(description, -1)
		pos = append(pos, matches...)
	}
	return pos
}

func extractBankEntityName(description string, knownCustomers, knownSuppliers []string) (customer, supplier string) {
	desc := strings.ToUpper(description)
	for _, cust := range knownCustomers {
		if strings.Contains(desc, strings.ToUpper(cust)) {
			return cust, ""
		}
	}
	for _, supp := range knownSuppliers {
		if strings.Contains(desc, strings.ToUpper(supp)) {
			return "", supp
		}
	}
	if strings.Contains(desc, "TRANSFER") {
		parts := strings.Split(desc, "TRANSFER")
		if len(parts) > 1 {
			name := strings.TrimSpace(parts[1])
			name = strings.Split(name, " FTRF")[0]
			name = strings.Split(name, " W.L.L")[0]
			name = strings.Split(name, " WLL")[0]
			name = strings.Split(name, " B.S.C")[0]
			if len(name) > 3 {
				return strings.TrimSpace(name), ""
			}
		}
	}
	return "", ""
}

func matchToCustomerInvoice(db *gorm.DB, line *finance.BankStatementLine, invoiceNumbers []string, customerName string, division string) (bool, string, float64) {
	division = normalizeDivisionName(division)
	if line.Credit <= 0 {
		return false, "", 0
	}
	openStatuses := []string{"Sent", "Overdue", "PartiallyPaid"}

	for _, invNum := range invoiceNumbers {
		var invoice matchInvoice
		err := db.Where("invoice_number LIKE ? ESCAPE '\\\\' AND division = ? AND status IN ?", "%"+escapeLikeWildcards(invNum)+"%", division, openStatuses).First(&invoice).Error
		if err == nil {
			payable := payableCustomerInvoiceAmount(invoice)
			if line.Credit <= payable+bankMatchTolerance(payable) || bankAmountsClose(line.Credit, payable) {
				return true, invoice.ID, 0.95
			}
		}
	}

	if customerName != "" {
		var customer matchCustomer
		err := db.Where("business_name LIKE ? ESCAPE '\\\\'", "%"+escapeLikeWildcards(customerName)+"%").First(&customer).Error
		if err == nil {
			var invoices []matchInvoice
			err = db.Where("customer_id = ? AND division = ? AND status IN ?", customer.ID, division, openStatuses).Find(&invoices).Error
			if err == nil {
				for _, invoice := range invoices {
					if bankAmountsClose(line.Credit, payableCustomerInvoiceAmount(invoice)) {
						return true, invoice.ID, 0.85
					}
				}
			}
		}
	}

	var invoices []matchInvoice
	err := db.Where("division = ? AND status IN ?", division, openStatuses).Limit(100).Find(&invoices).Error
	if err == nil {
		for _, invoice := range invoices {
			if bankAmountsClose(line.Credit, payableCustomerInvoiceAmount(invoice)) {
				return true, invoice.ID, 0.60
			}
		}
	}
	return false, "", 0
}

func matchToSupplierPayment(db *gorm.DB, line *finance.BankStatementLine, poNumbers []string, supplierName string, division string) (bool, string, float64) {
	division = normalizeDivisionName(division)
	if line.Debit <= 0 {
		return false, "", 0
	}
	for _, poNum := range poNumbers {
		var invoice matchSupplierInvoice
		err := db.Where("po_number LIKE ? ESCAPE '\\\\' AND division = ? AND status NOT IN ?",
			"%"+escapeLikeWildcards(poNum)+"%", division, []string{"Paid", "Rejected", "Cancelled", "Void"}).First(&invoice).Error
		if err == nil && bankAmountsClose(line.Debit, invoice.TotalBHD) {
			return true, invoice.ID, 0.90
		}
	}

	if supplierName != "" {
		var supplier matchSupplier
		err := db.Where("supplier_name LIKE ? ESCAPE '\\\\'", "%"+escapeLikeWildcards(supplierName)+"%").First(&supplier).Error
		if err == nil {
			var invoice matchSupplierInvoice
			err = db.Where("supplier_id = ? AND total_bhd = ? AND division = ? AND status != ?",
				supplier.ID, line.Debit, division, "Paid").First(&invoice).Error
			if err == nil {
				return true, invoice.ID, 0.80
			}
		}
	}
	return false, "", 0
}

func createExpenseFromBankLine(db *gorm.DB, line *finance.BankStatementLine, txType string) error {
	expense := finance.BankExpenseEntry{
		BankStatementLineID: line.ID,
		Division:            resolveBankStatementDivisionTx(db, line.BankStatementID),
		ExpenseDate:         line.TransactionDate,
		Description:         line.Description,
		Category:            txType,
		Amount:              line.Debit,
		Currency:            "BHD",
	}
	if txType == TxTypeVATOnFee {
		expense.VATAmount = line.Debit
	}
	return db.Create(&expense).Error
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) ManualMatchLine(lineID, entityType, entityID, user string) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var line finance.BankStatementLine
	if err := s.db.First(&line, "id = ?", lineID).Error; err != nil {
		return fmt.Errorf("line not found: %w", err)
	}
	// PC-D4: refuse manual matching on a finalized statement.
	if err := ensureStatementMutableTx(s.db, line.BankStatementID, "match transactions"); err != nil {
		return err
	}
	statementDivision := resolveBankStatementDivisionTx(s.db, line.BankStatementID)
	if line.IsMatched {
		return fmt.Errorf("line already matched")
	}

	var entityAmount float64
	lineAmt := line.Credit
	if lineAmt == 0 {
		lineAmt = line.Debit
	}
	switch entityType {
	case "CUSTOMER_INVOICE":
		if line.Credit <= 0 {
			return fmt.Errorf("customer invoices must be matched to credit transactions")
		}
		var inv matchInvoice
		if err := s.db.First(&inv, "id = ?", entityID).Error; err == nil {
			if normalizeDivisionName(inv.Division) != statementDivision {
				return fmt.Errorf("cannot match %s bank line to %s invoice", statementDivision, normalizeDivisionName(inv.Division))
			}
			entityAmount = payableCustomerInvoiceAmount(inv)
		} else {
			return fmt.Errorf("customer invoice not found: %w", err)
		}
	case "SUPPLIER_INVOICE":
		if line.Debit <= 0 {
			return fmt.Errorf("supplier invoices must be matched to debit transactions")
		}
		var sinv matchSupplierInvoice
		if err := s.db.First(&sinv, "id = ?", entityID).Error; err == nil {
			invoiceDivision := resolveSupplierInvoiceDivisionTx(s.db, sinv)
			if invoiceDivision != statementDivision {
				return fmt.Errorf("cannot match %s bank line to %s supplier invoice", statementDivision, invoiceDivision)
			}
			entityAmount = sinv.TotalBHD
		} else {
			return fmt.Errorf("supplier invoice not found: %w", err)
		}
	case "SUPPLIER_PAYMENT":
		if line.Debit <= 0 {
			return fmt.Errorf("supplier payments must be matched to debit transactions")
		}
		var payment matchSupplierPayment
		if err := s.db.First(&payment, "id = ?", entityID).Error; err == nil {
			if normalizeDivisionName(payment.Division) != statementDivision {
				return fmt.Errorf("cannot match %s bank line to %s supplier payment", statementDivision, normalizeDivisionName(payment.Division))
			}
			entityAmount = payment.AmountBHD
		} else {
			return fmt.Errorf("supplier payment not found: %w", err)
		}
	case "PAYROLL_PAYOUT":
		if line.Debit <= 0 {
			return fmt.Errorf("payroll payouts must be matched to debit transactions")
		}
		var payout matchPayrollPayout
		if err := s.db.First(&payout, "id = ?", entityID).Error; err == nil {
			entityAmount = payout.Amount
		} else {
			return fmt.Errorf("payroll payout not found: %w", err)
		}
	case "EXPENSE":
		if line.Debit <= 0 {
			return fmt.Errorf("expenses must be matched to debit transactions")
		}
		var expense matchExpenseEntry
		if err := s.db.First(&expense, "id = ?", entityID).Error; err == nil {
			if normalizeDivisionName(expense.Division) != statementDivision {
				return fmt.Errorf("cannot match %s bank line to %s expense", statementDivision, normalizeDivisionName(expense.Division))
			}
			if !isExpenseReconcilable(expense) {
				return fmt.Errorf("expense %s is not open for bank reconciliation", firstNonEmptyString(expense.EntryNumber, expense.ID))
			}
			entityAmount = payableExpenseEntryAmount(expense)
		} else {
			return fmt.Errorf("expense not found: %w", err)
		}
	default:
		return fmt.Errorf("unsupported match type: %s", entityType)
	}

	if entityAmount <= 0 {
		return fmt.Errorf("cannot match: entity amount is %.3f BHD - only positive-value invoices can be reconciled", entityAmount)
	}
	if lineAmt > 0 {
		tolerance := bankMatchTolerance(entityAmount)
		if entityType == "CUSTOMER_INVOICE" {
			if lineAmt > entityAmount+tolerance {
				return fmt.Errorf("amount mismatch: bank receipt %.3f BHD exceeds invoice outstanding %.3f BHD by more than %.3f BHD", lineAmt, entityAmount, tolerance)
			}
		} else if !bankAmountsClose(lineAmt, entityAmount) {
			ratio := lineAmt / entityAmount
			return fmt.Errorf("amount mismatch: bank line %.3f BHD vs entity %.3f BHD (%.1f%% difference, tolerance +/-5%%)", lineAmt, entityAmount, (ratio-1)*100)
		}
	}

	updates := map[string]any{
		"is_matched":       true,
		"match_type":       "Manual",
		"match_confidence": 1.0,
	}
	switch entityType {
	case "CUSTOMER_INVOICE", "SUPPLIER_INVOICE":
		updates["matched_invoice_ids"] = toJSONArray([]string{entityID})
	case "SUPPLIER_PAYMENT":
		updates["matched_payment_id"] = entityID
	case "EXPENSE":
		updates["matched_expense_id"] = entityID
	case "PAYROLL_PAYOUT":
		var payout matchPayrollPayout
		if err := s.db.First(&payout, "id = ?", entityID).Error; err != nil {
			return fmt.Errorf("payroll payout not found: %w", err)
		}
		var run matchPayrollRun
		if err := s.db.First(&run, "id = ?", payout.PayrollRunID).Error; err == nil && run.PayoutJournalEntryID != nil {
			updates["matched_journal_id"] = *run.PayoutJournalEntryID
		}
		lineIDRef := line.ID
		if err := s.db.Model(&payout).Updates(map[string]any{
			"bank_statement_line_id": &lineIDRef,
			"updated_at":             time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("failed to link payroll payout to bank line: %w", err)
		}
	}

	if err := s.db.Model(&line).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update line: %w", err)
	}
	_ = s.LogReconciliationAction(line.BankStatementID, &lineID, "MATCH",
		map[string]any{"type": "Manual", "entity_type": entityType, "entity_id": entityID},
		user, false, 1.0, "Manual match")

	log.Printf("Manual match: Line %s -> %s %s", lineID, entityType, entityID)
	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) UnmatchLine(lineID, user, reason string) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var line finance.BankStatementLine
	if err := s.db.First(&line, "id = ?", lineID).Error; err != nil {
		return fmt.Errorf("line not found: %w", err)
	}
	// PC-D4: refuse unmatching on a finalized statement.
	if err := ensureStatementMutableTx(s.db, line.BankStatementID, "unmatch transactions"); err != nil {
		return err
	}
	previousMatch := line.MatchedInvoiceIDs
	if previousMatch == "" {
		previousMatch = line.MatchedPaymentID
	}

	updates := map[string]any{
		"is_matched":          false,
		"match_type":          "Unmatched",
		"match_confidence":    0,
		"matched_invoice_ids": "",
		"matched_payment_id":  "",
		"matched_expense_id":  nil,
		"matched_journal_id":  "",
	}
	if err := s.db.Where("bank_statement_line_id = ?", lineID).Delete(&finance.BankLinePaymentAllocation{}).Error; err != nil {
		return fmt.Errorf("failed to clear split allocations: %w", err)
	}
	var payout matchPayrollPayout
	if err := s.db.Where("bank_statement_line_id = ?", lineID).First(&payout).Error; err == nil {
		if updateErr := s.db.Model(&payout).Updates(map[string]any{
			"bank_statement_line_id": nil,
			"updated_at":             time.Now(),
		}).Error; updateErr != nil {
			return fmt.Errorf("failed to unlink payroll payout from bank line: %w", updateErr)
		}
	}
	if err := s.db.Model(&line).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to unmatch line: %w", err)
	}
	_ = s.LogReconciliationAction(line.BankStatementID, &lineID, "UNMATCH",
		map[string]any{"previous_match": previousMatch, "reason": reason},
		user, false, 1.0, reason)

	log.Printf("Unmatched: Line %s by %s: %s", lineID, user, reason)
	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) CreateSplitAllocation(lineID string, allocations []AllocationInput, user string) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if len(allocations) == 0 {
		return fmt.Errorf("at least one allocation is required")
	}

	var line finance.BankStatementLine
	var allocationIDs []string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&line, "id = ?", lineID).Error; err != nil {
			return fmt.Errorf("line not found: %w", err)
		}
		// PC-D4: refuse split allocations on a finalized statement.
		if err := ensureStatementMutableTx(tx, line.BankStatementID, "match transactions"); err != nil {
			return err
		}
		if line.IsMatched {
			return fmt.Errorf("line already matched")
		}

		var statement finance.BankStatement
		if err := tx.First(&statement, "id = ?", line.BankStatementID).Error; err != nil {
			return fmt.Errorf("statement not found: %w", err)
		}
		statementDivision := normalizeDivisionName(statement.Division)
		if statementDivision == "" {
			statementDivision = resolveBankAccountDivisionTx(tx, statement.BankAccountID)
		}
		if statementDivision == "" {
			statementDivision = overlay.Active().DefaultDivision()
		}

		lineAmount := line.Credit
		if lineAmount == 0 {
			lineAmount = line.Debit
		}
		if lineAmount <= 0 {
			return fmt.Errorf("line amount must be positive")
		}
		const bhdRoundingTolerance = 0.001

		var totalAllocated float64
		seen := make(map[string]bool)
		allocationRecords := make([]finance.BankLinePaymentAllocation, 0, len(allocations))
		allocationIDs = make([]string, 0, len(allocations))

		for _, alloc := range allocations {
			allocationType := strings.ToUpper(strings.TrimSpace(alloc.AllocationType))
			entityID := strings.TrimSpace(alloc.EntityID)
			allocatedAmount := roundBHD(alloc.AllocatedAmount)
			if entityID == "" {
				return fmt.Errorf("allocation target is required")
			}
			if allocatedAmount <= 0 {
				return fmt.Errorf("allocation amount for %s must be positive", entityID)
			}
			key := allocationType + ":" + entityID
			if seen[key] {
				return fmt.Errorf("duplicate allocation target: %s", entityID)
			}
			seen[key] = true

			allocation := finance.BankLinePaymentAllocation{
				BankStatementLineID: lineID,
				AllocationType:      allocationType,
				AllocatedAmount:     allocatedAmount,
				Currency:            "BHD",
				Status:              "Allocated",
			}

			switch allocationType {
			case "CUSTOMER_INVOICE":
				if line.Credit <= 0 {
					return fmt.Errorf("customer invoices must be allocated to credit transactions")
				}
				var invoice matchInvoice
				if err := tx.First(&invoice, "id = ?", entityID).Error; err != nil {
					return fmt.Errorf("customer invoice not found: %w", err)
				}
				if normalizeDivisionName(invoice.Division) != statementDivision {
					return fmt.Errorf("cannot allocate %s bank line to %s invoice", statementDivision, normalizeDivisionName(invoice.Division))
				}
				payable := payableCustomerInvoiceAmount(invoice)
				var previouslyAllocated float64
				if err := tx.Model(&finance.BankLinePaymentAllocation{}).
					Where("customer_invoice_id = ? AND status <> ? AND bank_statement_line_id <> ?", entityID, "Disputed", lineID).
					Select("COALESCE(SUM(allocated_amount), 0)").
					Scan(&previouslyAllocated).Error; err != nil {
					return fmt.Errorf("failed to check existing customer allocations: %w", err)
				}
				remaining := roundBHD(payable - previouslyAllocated)
				if allocatedAmount > remaining+bhdRoundingTolerance {
					return fmt.Errorf("allocation %.3f BHD exceeds remaining invoice balance %.3f BHD", allocatedAmount, remaining)
				}
				idCopy := entityID
				allocation.CustomerInvoiceID = &idCopy
			case "SUPPLIER_INVOICE":
				if line.Debit <= 0 {
					return fmt.Errorf("supplier invoices must be allocated to debit transactions")
				}
				var invoice matchSupplierInvoice
				if err := tx.First(&invoice, "id = ?", entityID).Error; err != nil {
					return fmt.Errorf("supplier invoice not found: %w", err)
				}
				invoiceDivision := resolveSupplierInvoiceDivisionTx(tx, invoice)
				if invoiceDivision != statementDivision {
					return fmt.Errorf("cannot allocate %s bank line to %s supplier invoice", statementDivision, invoiceDivision)
				}
				var previouslyAllocated float64
				if err := tx.Model(&finance.BankLinePaymentAllocation{}).
					Where("supplier_invoice_id = ? AND status <> ? AND bank_statement_line_id <> ?", entityID, "Disputed", lineID).
					Select("COALESCE(SUM(allocated_amount), 0)").
					Scan(&previouslyAllocated).Error; err != nil {
					return fmt.Errorf("failed to check existing supplier allocations: %w", err)
				}
				remaining := roundBHD(invoice.TotalBHD - previouslyAllocated)
				if allocatedAmount > remaining+bhdRoundingTolerance {
					return fmt.Errorf("allocation %.3f BHD exceeds remaining supplier invoice balance %.3f BHD", allocatedAmount, remaining)
				}
				idCopy := entityID
				allocation.SupplierInvoiceID = &idCopy
			case "EXPENSE":
				if line.Debit <= 0 {
					return fmt.Errorf("expenses must be allocated to debit transactions")
				}
				var expense matchExpenseEntry
				if err := tx.First(&expense, "id = ?", entityID).Error; err != nil {
					return fmt.Errorf("expense not found: %w", err)
				}
				if normalizeDivisionName(expense.Division) != statementDivision {
					return fmt.Errorf("cannot allocate %s bank line to %s expense", statementDivision, normalizeDivisionName(expense.Division))
				}
				if !isExpenseReconcilable(expense) {
					return fmt.Errorf("expense %s is not open for bank reconciliation", firstNonEmptyString(expense.EntryNumber, expense.ID))
				}
				var previouslyAllocated float64
				if err := tx.Model(&finance.BankLinePaymentAllocation{}).
					Where("expense_entry_id = ? AND status <> ? AND bank_statement_line_id <> ?", entityID, "Disputed", lineID).
					Select("COALESCE(SUM(allocated_amount), 0)").
					Scan(&previouslyAllocated).Error; err != nil {
					return fmt.Errorf("failed to check existing expense allocations: %w", err)
				}
				remaining := roundBHD(payableExpenseEntryAmount(expense) - previouslyAllocated)
				if allocatedAmount > remaining+bhdRoundingTolerance {
					return fmt.Errorf("allocation %.3f BHD exceeds remaining expense balance %.3f BHD", allocatedAmount, remaining)
				}
				idCopy := entityID
				allocation.ExpenseEntryID = &idCopy
			default:
				return fmt.Errorf("unsupported allocation type: %s", allocationType)
			}

			totalAllocated = roundBHD(totalAllocated + allocatedAmount)
			allocationRecords = append(allocationRecords, allocation)
			allocationIDs = append(allocationIDs, entityID)
		}

		diff := math.Abs(totalAllocated - lineAmount)
		if diff > bhdRoundingTolerance {
			return fmt.Errorf("allocation total (%.3f) does not match line amount (%.3f) - difference %.3f exceeds 1 fils tolerance", totalAllocated, lineAmount, diff)
		}
		if err := tx.Create(&allocationRecords).Error; err != nil {
			return fmt.Errorf("failed to create allocations: %w", err)
		}
		return tx.Model(&line).Updates(map[string]any{
			"is_matched":          true,
			"match_type":          "Split",
			"match_confidence":    1.0,
			"matched_invoice_ids": toJSONArray(allocationIDs),
		}).Error
	})
	if err != nil {
		return err
	}

	allocJSON, _ := json.Marshal(allocations)
	_ = s.LogReconciliationAction(line.BankStatementID, &lineID, "SPLIT",
		map[string]any{"allocations": string(allocJSON)},
		user, false, 1.0, "Split allocation")

	log.Printf("Split allocation created for line %s: %d allocations", lineID, len(allocations))
	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) CategorizeTransactions(statementID string) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var lines []finance.BankStatementLine
	s.db.Where("bank_statement_id = ?", statementID).Find(&lines)

	var customerNames []string
	var supplierNames []string
	s.db.Model(&matchCustomer{}).Pluck("business_name", &customerNames)
	s.db.Model(&matchSupplier{}).Pluck("supplier_name", &supplierNames)

	categorizedCount := 0
	for _, line := range lines {
		isCredit := line.Credit > 0
		txType := detectTransactionType(line.Description, isCredit)
		invoices := extractInvoiceNumbers(line.Description)
		pos := extractPONumbers(line.Description)
		customer, supplier := extractBankEntityName(line.Description, customerNames, supplierNames)

		s.db.Model(&line).Updates(map[string]any{
			"transaction_type":     txType,
			"extracted_invoices":   toJSONArray(invoices),
			"extracted_po_numbers": toJSONArray(pos),
			"extracted_customer":   customer,
			"extracted_supplier":   supplier,
		})
		categorizedCount++
	}

	log.Printf("Categorized %d transactions for statement %s", categorizedCount, statementID)
	return nil
}

func payableCustomerInvoiceAmount(invoice matchInvoice) float64 {
	if invoice.OutstandingBHD > 0 {
		return invoice.OutstandingBHD
	}
	return invoice.GrandTotalBHD
}

func payableExpenseEntryAmount(expense matchExpenseEntry) float64 {
	if expense.TotalAmount > 0 {
		return expense.TotalAmount
	}
	if expense.Amount+expense.VATAmount > 0 {
		return expense.Amount + expense.VATAmount
	}
	return expense.Amount
}

func isExpenseReconcilable(expense matchExpenseEntry) bool {
	status := strings.ToLower(strings.TrimSpace(expense.Status))
	paymentStatus := strings.ToLower(strings.TrimSpace(expense.PaymentStatus))
	if paymentStatus == "paid" {
		return false
	}
	switch status {
	case "rejected", "cancelled", "canceled", "void":
		return false
	default:
		return true
	}
}

func bankMatchTolerance(amount float64) float64 {
	return math.Max(0.001, math.Abs(amount)*0.05)
}

func bankAmountsClose(a, b float64) bool {
	return math.Abs(a-b) <= bankMatchTolerance(math.Max(math.Abs(a), math.Abs(b)))
}

func escapeLikeWildcards(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

func toJSONArray(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(arr)
	return string(data)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func resolveBankAccountDivisionTx(db *gorm.DB, bankAccountID string) string {
	if db == nil || strings.TrimSpace(bankAccountID) == "" {
		return overlay.Active().DefaultDivision()
	}
	var account finance.CompanyBankAccount
	if err := db.Select("division").First(&account, "id = ?", bankAccountID).Error; err == nil {
		return normalizeDivisionName(account.Division)
	}
	return overlay.Active().DefaultDivision()
}

func resolveBankStatementDivisionTx(db *gorm.DB, statementID string) string {
	if db == nil || strings.TrimSpace(statementID) == "" {
		return overlay.Active().DefaultDivision()
	}
	var statement finance.BankStatement
	if err := db.Select("division", "bank_account_id").First(&statement, "id = ?", statementID).Error; err == nil {
		if strings.TrimSpace(statement.Division) != "" {
			return normalizeDivisionName(statement.Division)
		}
		return resolveBankAccountDivisionTx(db, statement.BankAccountID)
	}
	return overlay.Active().DefaultDivision()
}

func resolveSupplierInvoiceDivisionTx(db *gorm.DB, invoice matchSupplierInvoice) string {
	if strings.TrimSpace(invoice.Division) != "" {
		return normalizeDivisionName(invoice.Division)
	}
	if strings.TrimSpace(invoice.OrderID) != "" {
		return resolveOrderDivisionTx(db, invoice.OrderID)
	}
	if db == nil || strings.TrimSpace(invoice.PurchaseOrderID) == "" {
		return overlay.Active().DefaultDivision()
	}
	var po matchPurchaseOrder
	if err := db.Select("order_id", "division").First(&po, "id = ?", invoice.PurchaseOrderID).Error; err == nil {
		if strings.TrimSpace(po.Division) != "" {
			return normalizeDivisionName(po.Division)
		}
		return resolveOrderDivisionTx(db, po.OrderID)
	}
	return overlay.Active().DefaultDivision()
}

func resolveOrderDivisionTx(db *gorm.DB, orderID string) string {
	if db == nil || strings.TrimSpace(orderID) == "" {
		return overlay.Active().DefaultDivision()
	}
	var order matchOrder
	if err := db.Select("division").First(&order, "id = ?", orderID).Error; err == nil {
		return normalizeDivisionName(order.Division)
	}
	return overlay.Active().DefaultDivision()
}
