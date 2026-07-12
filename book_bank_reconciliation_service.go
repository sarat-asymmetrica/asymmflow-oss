// =============================================================================
// BOOK-BANK RECONCILIATION SERVICE
//
// MISSION: Reconcile book balance (GL) with bank statement balance
// FEATURES: Outstanding cheques, deposits in transit, adjustments, variance analysis
// =============================================================================

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// =============================================================================
// DEPOSITS IN TRANSIT
// =============================================================================

// CreateDepositInTransit records a deposit made but not yet on bank statement
func (a *App) CreateDepositInTransit(bankAccountID string, depositDate time.Time, amount float64, slipNo, description, sourceType string, customerID *string, invoiceIDs []string) (*DepositInTransit, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if amount <= 0 {
		return nil, fmt.Errorf("deposit amount must be positive")
	}

	invoiceIDsJSON, _ := json.Marshal(invoiceIDs)

	deposit := &DepositInTransit{
		BankAccountID: bankAccountID,
		DepositDate:   depositDate,
		Amount:        amount,
		Currency:      "BHD",
		DepositSlipNo: slipNo,
		Description:   description,
		SourceType:    sourceType,
		CustomerID:    customerID,
		InvoiceIDs:    string(invoiceIDsJSON),
		Status:        "PENDING",
	}

	if err := a.db.Create(deposit).Error; err != nil {
		return nil, fmt.Errorf("failed to create deposit: %w", err)
	}

	log.Printf("📥 Deposit in transit recorded: %.3f BHD (Slip: %s)", amount, slipNo)
	return deposit, nil
}

// DepositsInTransitResult bundles the deposits list with its total. Wails
// v2's bound-method marshaling only handles OutputCount 1 or 2 (see
// internal/binding/boundMethod.go) — a 3-value Go return silently marshals
// to null on the JS side. Bundling into a struct + error keeps the binding
// a clean 2-value return.
type DepositsInTransitResult struct {
	Deposits []DepositInTransit `json:"deposits"`
	Total    float64            `json:"total"`
}

// GetDepositsInTransit retrieves all pending deposits for an account
func (a *App) GetDepositsInTransit(bankAccountID string) (*DepositsInTransitResult, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var deposits []DepositInTransit
	query := a.db.Where("status = ?", "PENDING")
	if bankAccountID != "" {
		query = query.Where("bank_account_id = ?", bankAccountID)
	}

	if err := query.Order("deposit_date ASC").Find(&deposits).Error; err != nil {
		return nil, err
	}

	var total float64
	for _, d := range deposits {
		total += d.Amount
	}

	return &DepositsInTransitResult{Deposits: deposits, Total: total}, nil
}

// ClearDepositInTransit marks a deposit as cleared when it appears on statement
func (a *App) ClearDepositInTransit(depositID, matchedLineID string, clearedDate time.Time) error {
	if err := a.requirePermission("finance:create"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	result := a.db.Model(&DepositInTransit{}).
		Where("id = ? AND status = ?", depositID, "PENDING").
		Updates(map[string]any{
			"status":          "CLEARED",
			"cleared_date":    clearedDate,
			"matched_line_id": matchedLineID,
		})

	if result.RowsAffected == 0 {
		return fmt.Errorf("deposit not found or not pending")
	}

	log.Printf("✅ Deposit cleared: %s", depositID)
	return nil
}

// ReturnDeposit marks a deposit as returned (bounced)
func (a *App) ReturnDeposit(depositID, reason string) error {
	if err := a.requirePermission("finance:create"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	result := a.db.Model(&DepositInTransit{}).
		Where("id = ? AND status = ?", depositID, "PENDING").
		Updates(map[string]any{
			"status":      "RETURNED",
			"description": reason,
		})

	if result.RowsAffected == 0 {
		return fmt.Errorf("deposit not found or not pending")
	}

	log.Printf("❌ Deposit returned: %s - %s", depositID, reason)
	return nil
}

// =============================================================================
// BOOK-BANK RECONCILIATION
// =============================================================================

// CreateBookBankReconciliation creates a new book-bank reconciliation.
// Wave 9.3 B1b: depositsInTransit/outstandingCheques let the "prove the
// balance" step persist the user-confirmed totals (pre-populated from the
// register, editable) instead of always trusting a silent server recompute.
// Pass -1 for either to fall back to the auto-computed total (preserves
// older callers).
func (a *App) CreateBookBankReconciliation(bankAccountID string, reconciliationDate time.Time, bankStatementBalance, bookBalance, depositsInTransit, outstandingCheques float64) (*BookBankReconciliation, error) {
	return a.bankingService().CreateBookBankReconciliation(bankAccountID, reconciliationDate, bankStatementBalance, bookBalance, depositsInTransit, outstandingCheques)
}

func createBookBankReconciliation(a *App, bankAccountID string, reconciliationDate time.Time, bankStatementBalance, bookBalance, depositsInTransit, outstandingCheques float64) (*BookBankReconciliation, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Deposits in transit: caller-confirmed total wins; negative sentinel
	// falls back to the register auto-compute.
	depositsTotal := depositsInTransit
	if depositsTotal < 0 {
		if res, err := a.GetDepositsInTransit(bankAccountID); err == nil && res != nil {
			depositsTotal = res.Total
		}
	}

	// Outstanding cheques: same caller-confirmed-or-auto-compute rule.
	chequesTotal := outstandingCheques
	if chequesTotal < 0 {
		if res, err := a.GetOutstandingCheques(bankAccountID); err == nil && res != nil {
			chequesTotal = res.Total
		}
	}

	// Calculate adjusted bank balance
	adjustedBankBalance := bankStatementBalance + depositsTotal - chequesTotal

	// Calculate adjusted book balance (placeholder - bank charges and interest would come from user input)
	adjustedBookBalance := bookBalance

	// Calculate difference
	difference := adjustedBankBalance - adjustedBookBalance

	recon := &BookBankReconciliation{
		BankAccountID:        bankAccountID,
		ReconciliationDate:   reconciliationDate,
		Currency:             "BHD",
		BankStatementBalance: bankStatementBalance,
		DepositsInTransit:    depositsTotal,
		OutstandingCheques:   chequesTotal,
		AdjustedBankBalance:  adjustedBankBalance,
		BookBalance:          bookBalance,
		AdjustedBookBalance:  adjustedBookBalance,
		Difference:           difference,
		IsReconciled:         false,
	}

	if err := a.db.Create(recon).Error; err != nil {
		return nil, fmt.Errorf("failed to create reconciliation: %w", err)
	}

	log.Printf("📊 Book-Bank reconciliation created: Bank %.3f, Book %.3f, Diff %.3f BHD",
		adjustedBankBalance, adjustedBookBalance, difference)
	return recon, nil
}

// UpdateBookBankReconciliationAdjustments persists user-confirmed
// deposits-in-transit / outstanding-cheque totals on a reconciliation.
// Wave 9.3 B1b: the "prove the balance" step pre-populates these from the
// register but lets the user review/adjust them; this is the save path for
// that confirmation. Self-contained — talks directly to a.db, independent of
// the generic banking dispatcher (app_services.go / service_infra.go), so it
// doesn't ripple into CreateBookBankReconciliation's signature or any other
// coder's files. Proof record only — still no GL journal.
func (a *App) UpdateBookBankReconciliationAdjustments(reconID string, depositsInTransit, outstandingCheques float64) error {
	if err := a.requirePermission("finance:create"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var recon BookBankReconciliation
	if err := a.db.First(&recon, "id = ?", reconID).Error; err != nil {
		return fmt.Errorf("reconciliation not found: %w", err)
	}
	if recon.IsReconciled {
		return fmt.Errorf("cannot modify a finalized reconciliation")
	}

	recon.DepositsInTransit = depositsInTransit
	recon.OutstandingCheques = outstandingCheques
	recon.AdjustedBankBalance = recon.BankStatementBalance + depositsInTransit - outstandingCheques + recon.BankErrors
	recon.Difference = recon.AdjustedBankBalance - recon.AdjustedBookBalance

	if err := a.db.Save(&recon).Error; err != nil {
		return fmt.Errorf("failed to update reconciliation adjustments: %w", err)
	}

	log.Printf("📊 Book-Bank reconciliation %s adjustments: DIT %.3f, Outstanding Cheques %.3f, Difference now %.3f BHD",
		reconID, depositsInTransit, outstandingCheques, recon.Difference)
	return nil
}

// UpdateBookBankReconciliation updates adjustments on a reconciliation
func (a *App) UpdateBookBankReconciliation(reconID string, bankCharges, interest, nsfCheques, bankErrors, bookErrors float64, notes string) (*BookBankReconciliation, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var recon BookBankReconciliation
	if err := a.db.First(&recon, "id = ?", reconID).Error; err != nil {
		return nil, fmt.Errorf("reconciliation not found: %w", err)
	}

	if recon.IsReconciled {
		return nil, fmt.Errorf("cannot modify a finalized reconciliation")
	}

	// Recalculate adjusted bank balance
	recon.BankErrors = bankErrors
	recon.AdjustedBankBalance = recon.BankStatementBalance + recon.DepositsInTransit - recon.OutstandingCheques + bankErrors

	// Recalculate adjusted book balance
	recon.BankChargesNotRecorded = bankCharges
	recon.InterestNotRecorded = interest
	recon.NSFCheques = nsfCheques
	recon.BookErrors = bookErrors
	recon.AdjustedBookBalance = recon.BookBalance - bankCharges + interest - nsfCheques + bookErrors

	// Recalculate difference
	recon.Difference = recon.AdjustedBankBalance - recon.AdjustedBookBalance
	recon.Notes = notes

	if err := a.db.Save(&recon).Error; err != nil {
		return nil, fmt.Errorf("failed to update reconciliation: %w", err)
	}

	log.Printf("📊 Reconciliation updated: Difference now %.3f BHD", recon.Difference)
	return &recon, nil
}

// FinalizeReconciliation marks reconciliation as complete
func (a *App) FinalizeBookBankReconciliation(reconID, user string) error {
	if err := a.requirePermission("finance:create"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	// Wave 9.3 B2: identity resolved server-side; client value ignored (Article III.4)
	user = a.getCurrentUserID()

	var recon BookBankReconciliation
	if err := a.db.First(&recon, "id = ?", reconID).Error; err != nil {
		return fmt.Errorf("reconciliation not found: %w", err)
	}

	if recon.IsReconciled {
		return fmt.Errorf("already reconciled")
	}

	// Check if difference is acceptable (within 0.001 for BHD)
	if recon.Difference > 0.001 || recon.Difference < -0.001 {
		return fmt.Errorf("cannot finalize with difference of %.3f BHD - must be zero", recon.Difference)
	}

	now := time.Now()
	recon.IsReconciled = true
	recon.ReconciledBy = user
	recon.ReconciledAt = &now

	if err := a.db.Save(&recon).Error; err != nil {
		return fmt.Errorf("failed to finalize: %w", err)
	}

	log.Printf("✅ Book-Bank reconciliation finalized by %s", user)
	return nil
}

// GetBookBankReconciliation retrieves a specific reconciliation
func (a *App) GetBookBankReconciliation(reconID string) (*BookBankReconciliation, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var recon BookBankReconciliation
	if err := a.db.First(&recon, "id = ?", reconID).Error; err != nil {
		return nil, fmt.Errorf("reconciliation not found: %w", err)
	}

	return &recon, nil
}

// GetBookBankReconciliations retrieves reconciliation history for an account
func (a *App) GetBookBankReconciliations(bankAccountID string) ([]BookBankReconciliation, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var recons []BookBankReconciliation
	err := a.db.Where("bank_account_id = ?", bankAccountID).
		Order("reconciliation_date DESC").
		Find(&recons).Error

	return recons, err
}

// GetLatestBookBankReconciliation gets the most recent reconciliation for an account
func (a *App) GetLatestBookBankReconciliation(bankAccountID string) (*BookBankReconciliation, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var recon BookBankReconciliation
	err := a.db.Where("bank_account_id = ?", bankAccountID).
		Order("reconciliation_date DESC").
		First(&recon).Error

	if err != nil {
		return nil, fmt.Errorf("no reconciliation found for account")
	}

	return &recon, nil
}

// =============================================================================
// RECONCILIATION REPORTS
// =============================================================================

// BookBankReconciliationReport provides a detailed reconciliation view
type BookBankReconciliationReport struct {
	Reconciliation     BookBankReconciliation `json:"reconciliation"`
	DepositsInTransit  []DepositInTransit     `json:"deposits_in_transit"`
	OutstandingCheques []OutstandingCheque    `json:"outstanding_cheques"`
	BankAccount        CompanyBankAccount     `json:"bank_account"`
}

// GetBookBankReconciliationReport generates a detailed report
func (a *App) GetBookBankReconciliationReport(reconID string) (*BookBankReconciliationReport, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	recon, err := a.GetBookBankReconciliation(reconID)
	if err != nil {
		return nil, err
	}

	var deposits []DepositInTransit
	if res, err := a.GetDepositsInTransit(recon.BankAccountID); err == nil && res != nil {
		deposits = res.Deposits
	}
	var cheques []OutstandingCheque
	if res, err := a.GetOutstandingCheques(recon.BankAccountID); err == nil && res != nil {
		cheques = res.Cheques
	}

	var account CompanyBankAccount
	a.db.First(&account, "id = ?", recon.BankAccountID)

	return &BookBankReconciliationReport{
		Reconciliation:     *recon,
		DepositsInTransit:  deposits,
		OutstandingCheques: cheques,
		BankAccount:        account,
	}, nil
}

// ReconciliationStatusSummary provides an overview of all accounts
type ReconciliationStatusSummary struct {
	BankAccountID       string     `json:"bank_account_id"`
	BankName            string     `json:"bank_name"`
	AccountNumber       string     `json:"account_number"`
	Currency            string     `json:"currency"`
	LastReconciled      *time.Time `json:"last_reconciled"`
	LastDifference      float64    `json:"last_difference"`
	IsReconciled        bool       `json:"is_reconciled"`
	DaysSinceReconciled int        `json:"days_since_reconciled"`
	Status              string     `json:"status"` // CURRENT, OVERDUE, NEVER_RECONCILED
}

// GetReconciliationStatusSummary provides reconciliation status for all accounts
func (a *App) GetReconciliationStatusSummary() ([]ReconciliationStatusSummary, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var accounts []CompanyBankAccount
	a.db.Where("is_active = ?", true).Find(&accounts)

	var summaries []ReconciliationStatusSummary

	for _, account := range accounts {
		summary := ReconciliationStatusSummary{
			BankAccountID: account.ID,
			BankName:      account.BankName,
			AccountNumber: account.AccountNumber,
			Currency:      account.Currency,
			Status:        "NEVER_RECONCILED",
		}

		latestRecon, err := a.GetLatestBookBankReconciliation(account.ID)
		if err == nil {
			summary.LastReconciled = latestRecon.ReconciledAt
			summary.LastDifference = latestRecon.Difference
			summary.IsReconciled = latestRecon.IsReconciled

			if latestRecon.ReconciledAt != nil {
				daysSince := int(time.Since(*latestRecon.ReconciledAt).Hours() / 24)
				summary.DaysSinceReconciled = daysSince

				if daysSince <= 30 {
					summary.Status = "CURRENT"
				} else {
					summary.Status = "OVERDUE"
				}
			}
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// =============================================================================
// VARIANCE ANALYSIS
// =============================================================================

// VarianceItem represents a single variance in reconciliation
type VarianceItem struct {
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"` // BANK_ADJUSTMENT, BOOK_ADJUSTMENT
}

// GetReconciliationVariances breaks down the variance items
func (a *App) GetReconciliationVariances(reconID string) ([]VarianceItem, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	recon, err := a.GetBookBankReconciliation(reconID)
	if err != nil {
		return nil, err
	}

	var variances []VarianceItem

	// Bank side adjustments
	if recon.DepositsInTransit != 0 {
		variances = append(variances, VarianceItem{
			Category:    "Deposits in Transit",
			Description: "Deposits recorded in books but not yet on bank statement",
			Amount:      recon.DepositsInTransit,
			Type:        "BANK_ADJUSTMENT",
		})
	}

	if recon.OutstandingCheques != 0 {
		variances = append(variances, VarianceItem{
			Category:    "Outstanding Cheques",
			Description: "Cheques issued but not yet cleared by bank",
			Amount:      -recon.OutstandingCheques,
			Type:        "BANK_ADJUSTMENT",
		})
	}

	if recon.BankErrors != 0 {
		variances = append(variances, VarianceItem{
			Category:    "Bank Errors",
			Description: "Errors by the bank requiring correction",
			Amount:      recon.BankErrors,
			Type:        "BANK_ADJUSTMENT",
		})
	}

	// Book side adjustments
	if recon.BankChargesNotRecorded != 0 {
		variances = append(variances, VarianceItem{
			Category:    "Bank Charges",
			Description: "Bank fees not yet recorded in books",
			Amount:      -recon.BankChargesNotRecorded,
			Type:        "BOOK_ADJUSTMENT",
		})
	}

	if recon.InterestNotRecorded != 0 {
		variances = append(variances, VarianceItem{
			Category:    "Interest Income",
			Description: "Interest earned not yet recorded in books",
			Amount:      recon.InterestNotRecorded,
			Type:        "BOOK_ADJUSTMENT",
		})
	}

	if recon.NSFCheques != 0 {
		variances = append(variances, VarianceItem{
			Category:    "NSF Cheques",
			Description: "Customer cheques returned for insufficient funds",
			Amount:      -recon.NSFCheques,
			Type:        "BOOK_ADJUSTMENT",
		})
	}

	if recon.BookErrors != 0 {
		variances = append(variances, VarianceItem{
			Category:    "Book Errors",
			Description: "Recording errors in books requiring correction",
			Amount:      recon.BookErrors,
			Type:        "BOOK_ADJUSTMENT",
		})
	}

	return variances, nil
}

// =============================================================================
// AUTO-RECONCILIATION HELPERS
// =============================================================================

// AutoMatchDepositsToStatement attempts to match deposits in transit to bank statement lines
func (a *App) AutoMatchDepositsToStatement(bankAccountID, statementID string) (int, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return 0, err
	}
	if a.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	// Get pending deposits
	var deposits []DepositInTransit
	if res, err := a.GetDepositsInTransit(bankAccountID); err == nil && res != nil {
		deposits = res.Deposits
	}

	// Get unmatched credit lines from statement (credits have Credit > 0)
	var lines []BankStatementLine
	a.db.Where("bank_statement_id = ? AND credit > 0 AND is_matched = ?", statementID, false).Find(&lines)

	matchCount := 0

	for _, deposit := range deposits {
		for _, line := range lines {
			// Match by credit amount (within 0.001 tolerance)
			if line.Credit > deposit.Amount-0.001 && line.Credit < deposit.Amount+0.001 {
				// Clear the deposit
				a.ClearDepositInTransit(deposit.ID, line.ID, line.TransactionDate)

				// Mark line as matched
				a.db.Model(&line).Update("is_matched", true)

				matchCount++
				break
			}
		}
	}

	if matchCount > 0 {
		log.Printf("✅ Auto-matched %d deposits to statement lines", matchCount)
	}

	return matchCount, nil
}

// AutoMatchChequesToStatement attempts to match outstanding cheques to bank statement lines
func (a *App) AutoMatchChequesToStatement(bankAccountID, statementID string) (int, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return 0, err
	}
	if a.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	// Get outstanding cheques
	var cheques []OutstandingCheque
	if res, err := a.GetOutstandingCheques(bankAccountID); err == nil && res != nil {
		cheques = res.Cheques
	}

	// Get unmatched debit lines from statement (debits have Debit > 0)
	var lines []BankStatementLine
	a.db.Where("bank_statement_id = ? AND debit > 0 AND is_matched = ?", statementID, false).Find(&lines)

	matchCount := 0

	for _, cheque := range cheques {
		for _, line := range lines {
			// Match by debit amount (within 0.001 tolerance)
			if line.Debit > cheque.Amount-0.001 && line.Debit < cheque.Amount+0.001 {
				// Clear the cheque
				a.MarkChequeCleared(cheque.ChequeNumber, line.ID, line.TransactionDate)

				// Mark line as matched
				a.db.Model(&line).Update("is_matched", true)

				matchCount++
				break
			}
		}
	}

	if matchCount > 0 {
		log.Printf("✅ Auto-matched %d cheques to statement lines", matchCount)
	}

	return matchCount, nil
}
