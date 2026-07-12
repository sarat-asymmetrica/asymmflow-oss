// =============================================================================
// BANK RECONCILIATION SERVICE
//
// MISSION: Core service for bank statement reconciliation as SSOT
// FEATURES: Statement management, matching, cash position, validation
// =============================================================================

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

// =============================================================================
// STATEMENT MANAGEMENT
// =============================================================================

// GetBankStatements retrieves all statements for a bank account
func (a *App) GetBankStatements(bankAccountID string) ([]BankStatement, error) {
	return a.bankingService().GetBankStatements(bankAccountID)
}

// GetBankStatementByID retrieves a single statement with all lines
func (a *App) GetBankStatementByID(id string) (*BankStatement, error) {
	return a.bankingService().GetBankStatementByID(id)
}

// CreateBankStatement creates a new bank statement
func (a *App) CreateBankStatement(statement *BankStatement) error {
	return a.bankingService().CreateBankStatement(statement)
}

// UpdateBankStatement updates statement fields
func (a *App) UpdateBankStatement(id string, updates map[string]any) error {
	return a.bankingService().UpdateBankStatement(id, updates)
}

// DeleteBankStatement soft-deletes a statement
func (a *App) DeleteBankStatement(id string) error {
	return a.bankingService().DeleteBankStatement(id)
}

// =============================================================================
// STATEMENT LINES
// =============================================================================

// GetBankStatementLines retrieves all lines for a statement
func (a *App) GetBankStatementLines(statementID string) ([]BankStatementLine, error) {
	return a.bankingService().GetBankStatementLines(statementID)
}

// GetUnmatchedLines retrieves only unmatched lines for a statement
func (a *App) GetUnmatchedLines(statementID string) ([]BankStatementLine, error) {
	return a.bankingService().GetUnmatchedLines(statementID)
}

// UpdateBankStatementLine updates a statement line
func (a *App) UpdateBankStatementLine(lineID string, updates map[string]any) error {
	return a.bankingService().UpdateBankStatementLine(lineID, updates)
}

// CreateBankStatementLine adds a new line to a statement
func (a *App) CreateBankStatementLine(statementID string, line map[string]any) (*BankStatementLine, error) {
	return a.bankingService().CreateBankStatementLine(statementID, line)
}

// DeleteBankStatementLine soft-deletes a statement line
// SECURITY: Audit log before deletion (financial transaction)
func (a *App) DeleteBankStatementLine(lineID string) error {
	return a.bankingService().DeleteBankStatementLine(lineID)
}

// =============================================================================
// CASH POSITION SSOT
// =============================================================================

type CashPositionAccount struct {
	BankAccountID        string    `json:"bank_account_id"`
	Division             string    `json:"division"`
	BankName             string    `json:"bank_name"`
	AccountName          string    `json:"account_name"`
	AccountNumber        string    `json:"account_number"`
	Currency             string    `json:"currency"`
	CurrentBalance       float64   `json:"current_balance"`
	CurrentBalanceBHD    float64   `json:"current_balance_bhd"`
	LastStatementDate    time.Time `json:"last_statement_date"`
	LastStatementNumber  string    `json:"last_statement_number"`
	HasStatement         bool      `json:"has_statement"`
	IsCurrentStatement   bool      `json:"is_current_statement"`
	StatementPeriodLabel string    `json:"statement_period_label"`
	Notice               string    `json:"notice"`
}

type CashPositionSnapshot struct {
	Accounts              []CashPositionAccount `json:"accounts"`
	TotalBHD              float64               `json:"total_bhd"`
	TotalEUR              float64               `json:"total_eur"`
	CashBalanceBHD        float64               `json:"cash_balance_bhd"`
	LatestStatementPeriod string                `json:"latest_statement_period"`
	Notices               []string              `json:"notices"`
	StaleAccounts         []string              `json:"stale_accounts"`
	AsOf                  time.Time             `json:"as_of"`
}

// GetCashPosition returns real-time cash position for all bank accounts
func (a *App) GetCashPosition() (map[string]any, error) {
	return a.bankingService().GetCashPosition()
}

func getCashPosition(a *App) (map[string]any, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	snapshot, err := computeCashPositionSnapshot(a)
	if err != nil {
		return nil, fmt.Errorf("failed to get cash position: %w", err)
	}

	return map[string]any{
		"accounts":                snapshot.Accounts,
		"total_bhd":               snapshot.TotalBHD,
		"total_eur":               snapshot.TotalEUR,
		"cash_balance_bhd":        snapshot.CashBalanceBHD,
		"latest_statement_period": snapshot.LatestStatementPeriod,
		"notices":                 snapshot.Notices,
		"stale_accounts":          snapshot.StaleAccounts,
		"as_of":                   snapshot.AsOf,
	}, nil
}

func computeCashPositionSnapshot(a *App) (*CashPositionSnapshot, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var accounts []CompanyBankAccount
	if err := a.db.Where("is_active = ?", true).
		Order("division ASC, display_order ASC, bank_name ASC").
		Find(&accounts).Error; err != nil {
		return nil, err
	}

	statuses := []string{"Imported", "InProgress", "In Progress", "Reconciled", "Verified"}
	positions := make([]CashPositionAccount, 0, len(accounts))
	latestByDivision := make(map[string]time.Time)
	var latestOverall time.Time

	for _, account := range accounts {
		division := normalizeDivisionName(account.Division)
		if division == "" {
			division = activeOverlay.DefaultDivision()
		}

		position := CashPositionAccount{
			BankAccountID: account.ID,
			Division:      division,
			BankName:      account.BankName,
			AccountName:   account.AccountName,
			AccountNumber: account.AccountNumber,
			Currency:      strings.ToUpper(strings.TrimSpace(account.Currency)),
		}
		if position.Currency == "" {
			position.Currency = "BHD"
		}

		var statement BankStatement
		err := a.db.Where("bank_account_id = ? AND status IN ?", account.ID, statuses).
			Order("period_end DESC, updated_at DESC, created_at DESC").
			First(&statement).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			position.Notice = fmt.Sprintf("No statement imported for %s.", formatCashAccountName(position))
			positions = append(positions, position)
			continue
		}

		rate := normalizeExchangeRateToBHD(position.Currency, account.BookingRate)
		position.CurrentBalance = roundBHD(statement.ClosingBalance)
		position.CurrentBalanceBHD = roundBHD(statement.ClosingBalance * rate)
		position.LastStatementDate = statement.PeriodEnd
		position.LastStatementNumber = statement.StatementNumber
		position.HasStatement = true
		position.StatementPeriodLabel = formatCashStatementMonth(statement.PeriodEnd)
		positions = append(positions, position)

		if statement.PeriodEnd.After(latestByDivision[division]) {
			latestByDivision[division] = statement.PeriodEnd
		}
		if statement.PeriodEnd.After(latestOverall) {
			latestOverall = statement.PeriodEnd
		}
	}

	snapshot := &CashPositionSnapshot{
		Accounts:              positions,
		LatestStatementPeriod: formatCashStatementMonth(latestOverall),
		AsOf:                  time.Now(),
	}

	for i := range snapshot.Accounts {
		account := &snapshot.Accounts[i]
		if !account.HasStatement {
			snapshot.Notices = append(snapshot.Notices, account.Notice)
			snapshot.StaleAccounts = append(snapshot.StaleAccounts, formatCashAccountName(*account))
			continue
		}

		latestForDivision := latestByDivision[account.Division]
		account.IsCurrentStatement = sameCashStatementMonth(account.LastStatementDate, latestForDivision)
		if !account.IsCurrentStatement {
			account.Notice = fmt.Sprintf("%s latest statement is %s; latest imported month for %s is %s.",
				formatCashAccountName(*account),
				formatCashStatementMonth(account.LastStatementDate),
				account.Division,
				formatCashStatementMonth(latestForDivision),
			)
			snapshot.Notices = append(snapshot.Notices, account.Notice)
			snapshot.StaleAccounts = append(snapshot.StaleAccounts, formatCashAccountName(*account))
		}

		if account.Currency == "EUR" {
			snapshot.TotalEUR = roundBHD(snapshot.TotalEUR + account.CurrentBalance)
		}
		snapshot.TotalBHD = roundBHD(snapshot.TotalBHD + account.CurrentBalanceBHD)
	}
	snapshot.CashBalanceBHD = snapshot.TotalBHD

	return snapshot, nil
}

func sameCashStatementMonth(a, b time.Time) bool {
	if a.IsZero() || b.IsZero() {
		return false
	}
	return a.Year() == b.Year() && a.Month() == b.Month()
}

func formatCashStatementMonth(value time.Time) string {
	if value.IsZero() {
		return "no statement"
	}
	return value.Format("Jan 2006")
}

func formatCashAccountName(account CashPositionAccount) string {
	parts := []string{strings.TrimSpace(account.BankName)}
	if strings.TrimSpace(account.AccountName) != "" {
		parts = append(parts, strings.TrimSpace(account.AccountName))
	}
	if strings.TrimSpace(account.AccountNumber) != "" {
		parts = append(parts, strings.TrimSpace(account.AccountNumber))
	}
	name := strings.Join(parts, " ")
	if name == "" {
		return "Bank account"
	}
	return name
}

// GetCashPositionByAccount returns cash position for a specific account
func (a *App) GetCashPositionByAccount(bankAccountID string) (float64, error) {
	return a.bankingService().GetCashPositionByAccount(bankAccountID)
}

func getCashPositionByAccount(a *App, bankAccountID string) (float64, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return 0, err
	}
	if a.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var balance float64
	err := a.db.Raw(`
		SELECT COALESCE(closing_balance, 0)
		FROM bank_statements
		WHERE bank_account_id = ? AND status IN ('Imported', 'InProgress', 'In Progress', 'Reconciled', 'Verified')
		ORDER BY period_end DESC
		LIMIT 1
	`, bankAccountID).Scan(&balance).Error

	if err != nil {
		return 0, fmt.Errorf("failed to get account balance: %w", err)
	}

	return balance, nil
}

// =============================================================================
// RECONCILIATION WORKFLOW
// =============================================================================

// ValidateStatementBalance checks if Opening + Credits - Debits = Closing
func (a *App) ValidateStatementBalance(statementID string) (*StatementBalanceValidation, error) {
	return a.bankingService().ValidateStatementBalance(statementID)
}

// FinalizeReconciliation marks a statement as reconciled
func (a *App) FinalizeReconciliation(statementID string, reconciledBy string) error {
	// Wave 9.3 B2: identity resolved server-side; client value ignored (Article III.4)
	return a.bankingService().FinalizeReconciliation(statementID, a.getCurrentUserID())
}

// ReopenReconciliation reverts a reconciled statement to InProgress
func (a *App) ReopenReconciliation(statementID string, user, reason string) error {
	return a.bankingService().ReopenReconciliation(statementID, user, reason)
}

// =============================================================================
// SUMMARY & STATS
// =============================================================================

// GetReconciliationSummary returns summary stats for a statement
func (a *App) GetReconciliationSummary(statementID string) (map[string]any, error) {
	return a.bankingService().GetReconciliationSummary(statementID)
}

// GetReconciliationStats returns account-level reconciliation statistics
func (a *App) GetReconciliationStats(bankAccountID string) (map[string]any, error) {
	return a.bankingService().GetReconciliationStats(bankAccountID)
}

// =============================================================================
// HELPERS
// =============================================================================

// roundBHD rounds a float to 3 decimal places for BHD precision
func roundBHD(amount float64) float64 {
	return math.Round(amount*1000) / 1000
}

// formatBHD formats a float to BHD with 3 decimal places
func formatBHD(amount float64) string {
	return fmt.Sprintf("%.3f", roundBHD(amount))
}

// parseJSONArray parses a JSON array string to []string
func parseJSONArray(jsonStr string) []string {
	if jsonStr == "" {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return []string{}
	}
	return result
}

// toJSONArray converts []string to JSON array string
func toJSONArray(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(arr)
	return string(data)
}
