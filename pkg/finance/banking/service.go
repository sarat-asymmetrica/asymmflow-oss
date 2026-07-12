// Package banking contains the concrete banking and reconciliation service implementation.
package banking

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"ph_holdings_app/pkg/finance"
	"ph_holdings_app/pkg/overlay"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handlers[BookBankReconciliationModel any, MatchResultModel any, AllocationInputModel any, BankStatementModel any, BankStatementLineModel any, BalanceContinuityReportModel any, StatementHashModel any, AuditLogModel any] struct {
	RequirePermission            func(permission string) error
	GetCashPosition              func() (map[string]any, error)
	GetCashPositionByAccount     func(bankAccountID string) (float64, error)
	CreateBookBankReconciliation func(bankAccountID string, reconciliationDate time.Time, bankStatementBalance, bookBalance, depositsInTransit, outstandingCheques float64) (*BookBankReconciliationModel, error)
}

type Service[BookBankReconciliationModel any, MatchResultModel any, AllocationInputModel any, BankStatementModel any, BankStatementLineModel any, BalanceContinuityReportModel any, StatementHashModel any, AuditLogModel any] struct {
	db       *gorm.DB
	handlers Handlers[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]
	auth     AuthorizationPort
	audit    AuditPort
	division DivisionPort
	deletes  DeleteApprovalPort
}

func New[BookBankReconciliationModel any, MatchResultModel any, AllocationInputModel any, BankStatementModel any, BankStatementLineModel any, BalanceContinuityReportModel any, StatementHashModel any, AuditLogModel any](db *gorm.DB, handlers Handlers[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel], ports Ports) *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel] {
	return &Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]{
		db:       db,
		handlers: handlers,
		auth:     ports.Auth,
		audit:    ports.Audit,
		division: ports.Division,
		deletes:  ports.DeleteApproval,
	}
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) requirePermission(permission string) error {
	if s.auth != nil {
		if s.auth.HasPermission(permission) {
			return nil
		}
		return fmt.Errorf("permission denied: %s", permission)
	}
	if s.handlers.RequirePermission == nil {
		return nil
	}
	return s.handlers.RequirePermission(permission)
}

func normalizeDivisionName(division string) string {
	// Consolidated into the overlay's normaliser (Mission D): division names
	// AND their legacy spellings are config — a deployment whose historic
	// data carries variant division strings declares them under its
	// division's "aliases" in overlay.json; unknown strings fall back to the
	// overlay's default division. No division vocabulary lives in engine code.
	return overlay.Active().NormalizeDivisionName(division)
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) currentUserID() string {
	if s.auth == nil {
		return "system"
	}
	userID := strings.TrimSpace(s.auth.CurrentUserID())
	if userID == "" {
		return "system"
	}
	return userID
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) logAction(entityType, entityID, action, detail string) {
	if s.audit == nil {
		return
	}
	if err := s.audit.LogAction(entityType, entityID, action, detail, s.currentUserID()); err != nil {
		log.Printf("banking audit log error: %v", err)
	}
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) logFinancialAction(action, entityType, entityID string, amount float64, currency string, details map[string]any) {
	if financialAudit, ok := s.audit.(FinancialAuditPort); ok {
		if err := financialAudit.LogFinancialTransaction(s.currentUserID(), action, entityType, entityID, amount, currency, details); err != nil {
			log.Printf("banking financial audit log error: %v", err)
		}
		return
	}

	s.logAction(entityType, entityID, action, fmt.Sprintf("amount=%.3f details=%v", amount, details))
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) GetBankStatements(bankAccountID string) ([]finance.BankStatement, error) {
	if err := s.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var statements []finance.BankStatement
	query := s.db.Order("statement_date DESC")
	if bankAccountID != "" {
		query = query.Where("bank_account_id = ?", bankAccountID)
	}

	if err := query.Find(&statements).Error; err != nil {
		log.Printf("GetBankStatements error: %v", err)
		return nil, fmt.Errorf("operation failed. Please try again or contact support")
	}

	return statements, nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) GetBankStatementByID(id string) (*finance.BankStatement, error) {
	if err := s.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var statement finance.BankStatement
	if err := s.db.Preload("Lines").First(&statement, "id = ?", id).Error; err != nil {
		log.Printf("GetBankStatementByID error: %v", err)
		return nil, fmt.Errorf("operation failed. Please try again or contact support")
	}

	return &statement, nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) CreateBankStatement(statement *finance.BankStatement) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if statement == nil {
		return fmt.Errorf("bank statement is required")
	}

	if statement.Status == "" {
		statement.Status = "Imported"
	}
	if strings.TrimSpace(statement.Division) == "" {
		if s.division != nil {
			division, err := s.division.ResolveDivision(statement.BankAccountID)
			if err != nil {
				return err
			}
			statement.Division = normalizeDivisionName(division)
		} else {
			statement.Division = normalizeDivisionName("")
		}
	} else {
		statement.Division = normalizeDivisionName(statement.Division)
	}

	if err := s.db.Create(statement).Error; err != nil {
		log.Printf("CreateBankStatement error: %v", err)
		return fmt.Errorf("operation failed. Please try again or contact support")
	}

	log.Printf("Bank statement created: %s, period %s to %s",
		statement.StatementNumber,
		statement.PeriodStart.Format("2006-01-02"),
		statement.PeriodEnd.Format("2006-01-02"),
	)
	s.logAction("bank_statement", statement.ID, "create", fmt.Sprintf("Created bank statement %s", statement.StatementNumber))
	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) UpdateBankStatement(id string, updates map[string]any) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("bank statement ID is required")
	}

	// PC-D4: refuse edits on a finalized statement, and refuse setting a final
	// status directly — final states are only reachable through the workflow.
	if err := ensureStatementMutableTx(s.db, id, "edit statement"); err != nil {
		return err
	}
	if err := normalizeStatementStatusUpdate(updates); err != nil {
		return err
	}

	// Mission I (I-12): whitelist — the unfiltered map let a client rewrite
	// integrity columns (hashes, IDs, audit fields) on non-finalized statements.
	allowedColumns := map[string]bool{
		"bank_account_id": true, "division": true, "statement_number": true,
		"statement_date": true, "period_start": true, "period_end": true,
		"opening_balance": true, "closing_balance": true, "currency": true,
		"status": true, "notes": true,
	}
	for key := range updates {
		if !allowedColumns[key] {
			log.Printf("UpdateBankStatement: dropped non-editable column %q", key)
			delete(updates, key)
		}
	}
	if len(updates) == 0 {
		return fmt.Errorf("no editable fields in update payload")
	}

	if bankAccountID, ok := updates["bank_account_id"].(string); ok && strings.TrimSpace(bankAccountID) != "" && s.division != nil {
		division, err := s.division.ResolveDivision(bankAccountID)
		if err != nil {
			return err
		}
		updates["division"] = normalizeDivisionName(division)
	} else if division, ok := updates["division"].(string); ok && strings.TrimSpace(division) != "" {
		updates["division"] = normalizeDivisionName(division)
	}

	result := s.db.Model(&finance.BankStatement{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		log.Printf("UpdateBankStatement error: %v", result.Error)
		return fmt.Errorf("operation failed. Please try again or contact support")
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("statement not found")
	}

	log.Printf("Bank statement updated: %s", id)
	s.logAction("bank_statement", id, "update", "Updated bank statement")
	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) DeleteBankStatement(id string) error {
	if s.deletes != nil {
		ok, err := s.deletes.GuardDeleteOrRequest("finance:delete", "bank_statement", id, "Bank statement")
		if !ok {
			return err
		}
	}
	if err := s.requirePermission("finance:delete"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("bank statement ID is required")
	}
	// PC-D4: a finalized statement must be reopened before it can be deleted.
	if err := ensureStatementMutableTx(s.db, id, "delete statement"); err != nil {
		return err
	}

	result := s.db.Delete(&finance.BankStatement{}, "id = ?", id)
	if result.Error != nil {
		log.Printf("DeleteBankStatement error: %v", result.Error)
		return fmt.Errorf("operation failed. Please try again or contact support")
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("statement not found")
	}

	log.Printf("Bank statement deleted: %s", id)
	s.logAction("bank_statement", id, "delete", "Deleted bank statement")
	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) GetBankStatementLines(statementID string) ([]finance.BankStatementLine, error) {
	if err := s.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var lines []finance.BankStatementLine
	if err := s.db.Where("bank_statement_id = ?", statementID).
		Order("line_number ASC").
		Find(&lines).Error; err != nil {
		log.Printf("GetBankStatementLines error: %v", err)
		return nil, fmt.Errorf("operation failed. Please try again or contact support")
	}

	return lines, nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) GetUnmatchedLines(statementID string) ([]finance.BankStatementLine, error) {
	if err := s.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var lines []finance.BankStatementLine
	if err := s.db.Where("bank_statement_id = ? AND is_matched = ?", statementID, false).
		Order("line_number ASC").
		Find(&lines).Error; err != nil {
		log.Printf("GetUnmatchedLines error: %v", err)
		return nil, fmt.Errorf("operation failed. Please try again or contact support")
	}

	return lines, nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) UpdateBankStatementLine(lineID string, updates map[string]any) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var originalLine finance.BankStatementLine
	editedMatchedLine := false

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&originalLine, "id = ?", lineID).Error; err != nil {
			return fmt.Errorf("line not found: %w", err)
		}
		// PC-D4: refuse line edits on a finalized statement.
		if err := ensureStatementMutableTx(tx, originalLine.BankStatementID, "edit statement lines"); err != nil {
			return err
		}

		sanitizedUpdates, err := sanitizeBankStatementLineUpdates(originalLine, updates)
		if err != nil {
			return err
		}
		if len(sanitizedUpdates) == 0 {
			return nil
		}

		if originalLine.IsMatched {
			if err := clearBankStatementLineMatchTx(tx, lineID); err != nil {
				return err
			}
			editedMatchedLine = true
		}

		if err := tx.Model(&finance.BankStatementLine{}).Where("id = ?", lineID).Updates(sanitizedUpdates).Error; err != nil {
			return fmt.Errorf("failed to update line: %w", err)
		}

		return refreshBankStatementRollupTx(tx, originalLine.BankStatementID)
	})
	if err != nil {
		log.Printf("UpdateBankStatementLine error: %v", err)
		return fmt.Errorf("operation failed. Please try again or contact support")
	}

	if editedMatchedLine {
		previousMatch := originalLine.MatchedInvoiceIDs
		if previousMatch == "" {
			previousMatch = originalLine.MatchedPaymentID
		}
		s.LogReconciliationAction(originalLine.BankStatementID, &lineID, "UNMATCH",
			map[string]any{
				"previous_match": previousMatch,
				"reason":         "Line edited during OCR review",
			},
			s.currentUserID(), false, 1.0, "Line edited during OCR review")
	}

	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) CreateBankStatementLine(statementID string, line map[string]any) (*finance.BankStatementLine, error) {
	if err := s.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var newLine finance.BankStatementLine
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// PC-D4: refuse adding lines to a finalized statement.
		if err := ensureStatementMutableTx(tx, statementID, "add statement lines"); err != nil {
			return err
		}

		var maxLineNum int
		if err := tx.Model(&finance.BankStatementLine{}).Where("bank_statement_id = ?", statementID).
			Select("COALESCE(MAX(line_number), 0)").Scan(&maxLineNum).Error; err != nil {
			return fmt.Errorf("failed to determine next line number: %w", err)
		}

		debit, err := parseBankStatementAmountValue(line["debit"])
		if err != nil {
			return err
		}
		credit, err := parseBankStatementAmountValue(line["credit"])
		if err != nil {
			return err
		}
		balance, err := parseBankStatementAmountValue(line["balance"])
		if err != nil {
			return err
		}
		if err := validateBankStatementAmounts(debit, credit); err != nil {
			return err
		}

		transactionDate, err := parseBankStatementDateValue(line["transaction_date"])
		if err != nil {
			return err
		}
		description := strings.TrimSpace(fmt.Sprintf("%v", line["description"]))
		if description == "" {
			return fmt.Errorf("description is required")
		}

		newLine = finance.BankStatementLine{
			Base:            finance.Base{ID: uuid.New().String()},
			BankStatementID: statementID,
			LineNumber:      maxLineNum + 1,
			TransactionDate: transactionDate,
			ValueDate:       transactionDate,
			Description:     description,
			Reference:       strings.TrimSpace(fmt.Sprintf("%v", line["reference"])),
			Debit:           debit,
			Credit:          credit,
			Balance:         balance,
			MatchType:       "Unmatched",
		}

		if err := tx.Create(&newLine).Error; err != nil {
			return fmt.Errorf("failed to create line: %w", err)
		}

		return refreshBankStatementRollupTx(tx, statementID)
	})
	if err != nil {
		return nil, err
	}

	log.Printf("Bank statement line created: %s in statement %s", newLine.ID, statementID)
	return &newLine, nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) DeleteBankStatementLine(lineID string) error {
	if s.deletes != nil {
		ok, err := s.deletes.GuardDeleteOrRequest("finance:delete", "bank_statement_line", lineID, "Bank statement line")
		if !ok {
			return err
		}
	}
	if err := s.requirePermission("finance:delete"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var line finance.BankStatementLine
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&line, "id = ?", lineID).Error; err != nil {
			return fmt.Errorf("line not found: %w", err)
		}
		// PC-D4: refuse deleting lines from a finalized statement.
		if err := ensureStatementMutableTx(tx, line.BankStatementID, "delete statement lines"); err != nil {
			return err
		}

		if err := unlinkPayrollPayoutsFromLineTx(tx, lineID); err != nil {
			return fmt.Errorf("failed to unlink payroll payout from deleted line: %w", err)
		}

		if err := tx.Delete(&finance.BankStatementLine{}, "id = ?", lineID).Error; err != nil {
			return fmt.Errorf("failed to delete line: %w", err)
		}

		return refreshBankStatementRollupTx(tx, line.BankStatementID)
	})
	if err != nil {
		return err
	}

	log.Printf("AUDIT: Bank statement line %s deleted, amount %.3f BHD, description %s, user %s",
		lineID, line.Debit+line.Credit, line.Description, s.currentUserID())
	s.logFinancialAction(
		"bank_line_deleted",
		"bank_statement_line",
		lineID,
		line.Debit+line.Credit,
		"BHD",
		map[string]any{
			"bank_statement_id": line.BankStatementID,
			"description":       line.Description,
			"reference":         line.Reference,
			"debit":             line.Debit,
			"credit":            line.Credit,
		},
	)

	log.Printf("Bank statement line deleted: %s", lineID)
	return nil
}

func sanitizeBankStatementLineUpdates(existing finance.BankStatementLine, updates map[string]any) (map[string]any, error) {
	sanitized := make(map[string]any)

	for key, value := range updates {
		switch key {
		case "transaction_date":
			transactionDate, err := parseBankStatementDateValue(value)
			if err != nil {
				return nil, err
			}
			sanitized["transaction_date"] = transactionDate
			sanitized["value_date"] = transactionDate
		case "description":
			description := strings.TrimSpace(fmt.Sprintf("%v", value))
			if description == "" {
				return nil, fmt.Errorf("description is required")
			}
			sanitized["description"] = description
		case "reference":
			sanitized["reference"] = strings.TrimSpace(fmt.Sprintf("%v", value))
		case "debit", "credit", "balance":
			amount, err := parseBankStatementAmountValue(value)
			if err != nil {
				return nil, err
			}
			sanitized[key] = amount
		}
	}

	debit := existing.Debit
	if value, ok := sanitized["debit"].(float64); ok {
		debit = value
	}
	credit := existing.Credit
	if value, ok := sanitized["credit"].(float64); ok {
		credit = value
	}

	if err := validateBankStatementAmounts(debit, credit); err != nil {
		return nil, err
	}

	return sanitized, nil
}

func parseBankStatementAmountValue(value any) (float64, error) {
	switch val := value.(type) {
	case nil:
		return 0, nil
	case float64:
		return roundBHD(val), nil
	case float32:
		return roundBHD(float64(val)), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		trimmed := strings.TrimSpace(val)
		if trimmed == "" {
			return 0, nil
		}
		parsed, err := strconv.ParseFloat(strings.ReplaceAll(trimmed, ",", ""), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid amount: %q", val)
		}
		return roundBHD(parsed), nil
	default:
		return 0, fmt.Errorf("invalid amount value")
	}
}

func parseBankStatementDateValue(value any) (time.Time, error) {
	switch val := value.(type) {
	case time.Time:
		if val.IsZero() {
			return time.Time{}, fmt.Errorf("transaction date is required")
		}
		return val, nil
	case string:
		trimmed := strings.TrimSpace(val)
		if trimmed == "" {
			return time.Time{}, fmt.Errorf("transaction date is required")
		}
		parsed := parseDate(trimmed)
		if parsed.IsZero() {
			return time.Time{}, fmt.Errorf("invalid transaction date")
		}
		return parsed, nil
	default:
		return time.Time{}, fmt.Errorf("invalid transaction date")
	}
}

func validateBankStatementAmounts(debit, credit float64) error {
	if debit < 0 || credit < 0 {
		return fmt.Errorf("debit and credit must be non-negative")
	}
	if debit == 0 && credit == 0 {
		return fmt.Errorf("either debit or credit must be non-zero")
	}
	if debit > 0 && credit > 0 {
		return fmt.Errorf("cannot have both debit and credit on the same line")
	}
	return nil
}

func clearBankStatementLineMatchTx(tx *gorm.DB, lineID string) error {
	if err := tx.Where("bank_statement_line_id = ?", lineID).Delete(&finance.BankLinePaymentAllocation{}).Error; err != nil {
		return fmt.Errorf("failed to clear bank line allocations: %w", err)
	}

	if err := unlinkPayrollPayoutsFromLineTx(tx, lineID); err != nil {
		return fmt.Errorf("failed to unlink payroll payout from bank line: %w", err)
	}

	return tx.Model(&finance.BankStatementLine{}).Where("id = ?", lineID).Updates(map[string]any{
		"is_matched":          false,
		"match_type":          "Unmatched",
		"match_confidence":    0,
		"matched_invoice_ids": "",
		"matched_payment_id":  "",
		"matched_expense_id":  nil,
		"matched_journal_id":  "",
	}).Error
}

func unlinkPayrollPayoutsFromLineTx(tx *gorm.DB, lineID string) error {
	return tx.Table("payroll_payouts").
		Where("bank_statement_line_id = ?", lineID).
		Updates(map[string]any{
			"bank_statement_line_id": nil,
			"updated_at":             time.Now(),
		}).Error
}

func refreshBankStatementRollupTx(tx *gorm.DB, statementID string) error {
	var statement finance.BankStatement
	if err := tx.First(&statement, "id = ?", statementID).Error; err != nil {
		return fmt.Errorf("statement not found: %w", err)
	}

	var rollup struct {
		TotalDebits  float64
		TotalCredits float64
		DebitCount   int64
		CreditCount  int64
	}

	if err := tx.Model(&finance.BankStatementLine{}).
		Select(`
			COALESCE(SUM(debit), 0) as total_debits,
			COALESCE(SUM(credit), 0) as total_credits,
			COALESCE(SUM(CASE WHEN debit > 0 THEN 1 ELSE 0 END), 0) as debit_count,
			COALESCE(SUM(CASE WHEN credit > 0 THEN 1 ELSE 0 END), 0) as credit_count
		`).
		Where("bank_statement_id = ?", statementID).
		Scan(&rollup).Error; err != nil {
		return fmt.Errorf("failed to calculate statement totals: %w", err)
	}

	expectedClosing := roundBHD(statement.OpeningBalance + rollup.TotalCredits - rollup.TotalDebits)
	discrepancy := roundBHD(math.Abs(expectedClosing - statement.ClosingBalance))
	isValid := discrepancy <= 0.001

	updateMap := map[string]any{
		"total_debits":       roundBHD(rollup.TotalDebits),
		"total_credits":      roundBHD(rollup.TotalCredits),
		"debit_count":        int(rollup.DebitCount),
		"credit_count":       int(rollup.CreditCount),
		"balance_verified":   isValid,
		"discrepancy_amount": discrepancy,
	}
	// PC-D4: finalized statements no longer silently auto-revert to InProgress
	// here — every mutation path refuses on a final status before reaching this
	// rollup, and reopening is an explicit, audited act (ReopenReconciliation).

	if err := tx.Model(&finance.BankStatement{}).Where("id = ?", statementID).Updates(updateMap).Error; err != nil {
		return fmt.Errorf("failed to update statement totals: %w", err)
	}

	return nil
}

func parseDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t
		}
	}

	return time.Time{}
}

func roundBHD(amount float64) float64 {
	return math.Round(amount*1000) / 1000
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) GetCashPosition() (map[string]any, error) {
	return s.handlers.GetCashPosition()
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) GetCashPositionByAccount(bankAccountID string) (float64, error) {
	return s.handlers.GetCashPositionByAccount(bankAccountID)
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) ValidateStatementBalance(statementID string) (*finance.StatementBalanceValidation, error) {
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

	var totals struct {
		TotalDebits  float64
		TotalCredits float64
	}

	s.db.Model(&finance.BankStatementLine{}).
		Select("COALESCE(SUM(debit), 0) as total_debits, COALESCE(SUM(credit), 0) as total_credits").
		Where("bank_statement_id = ?", statementID).
		Scan(&totals)

	expectedClosing := roundBHD(statement.OpeningBalance + totals.TotalCredits - totals.TotalDebits)
	discrepancy := roundBHD(math.Abs(expectedClosing - statement.ClosingBalance))
	isValid := discrepancy <= 0.001

	s.db.Model(&statement).Updates(map[string]any{
		"balance_verified":   isValid,
		"discrepancy_amount": discrepancy,
		"total_debits":       totals.TotalDebits,
		"total_credits":      totals.TotalCredits,
		"debit_count":        0,
		"credit_count":       0,
	})

	if isValid {
		log.Printf("Statement %s balance verified: Opening %.3f + Credits %.3f - Debits %.3f = Closing %.3f",
			statementID, statement.OpeningBalance, totals.TotalCredits, totals.TotalDebits, statement.ClosingBalance)
	} else {
		log.Printf("Statement %s balance mismatch: Expected %.3f, Got %.3f (Discrepancy: %.3f)",
			statementID, expectedClosing, statement.ClosingBalance, discrepancy)
	}

	return &finance.StatementBalanceValidation{IsValid: isValid, Discrepancy: discrepancy}, nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) FinalizeReconciliation(statementID, reconciledBy string) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var statement finance.BankStatement
	if err := s.db.First(&statement, "id = ?", statementID).Error; err != nil {
		return fmt.Errorf("statement not found: %w", err)
	}

	if statement.Status == "Reconciled" {
		return fmt.Errorf("statement already reconciled")
	}

	validation, err := s.ValidateStatementBalance(statementID)
	if err != nil {
		return fmt.Errorf("balance validation failed: %w", err)
	}

	if !validation.IsValid {
		return fmt.Errorf("cannot finalize: balance discrepancy of %.3f BHD", validation.Discrepancy)
	}

	var unmatchedCount int64
	s.db.Model(&finance.BankStatementLine{}).Where("bank_statement_id = ? AND is_matched = ?", statementID, false).Count(&unmatchedCount)

	if unmatchedCount > 0 {
		return fmt.Errorf("cannot finalize: %d unmatched transactions", unmatchedCount)
	}

	now := time.Now()
	if err := s.db.Model(&statement).Updates(map[string]any{
		"status":        "Reconciled",
		"reconciled_at": now,
		"reconciled_by": reconciledBy,
	}).Error; err != nil {
		return fmt.Errorf("failed to finalize reconciliation: %w", err)
	}

	if err := s.LogReconciliationAction(statementID, nil, "RECONCILE",
		map[string]any{"status": "Reconciled", "unmatched": unmatchedCount},
		reconciledBy, false, 1.0, "Statement finalized"); err != nil {
		return err
	}

	log.Printf("Bank statement %s reconciled by %s", statementID, reconciledBy)
	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) ReopenReconciliation(statementID string, user string, reason string) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var statement finance.BankStatement
	if err := s.db.First(&statement, "id = ?", statementID).Error; err != nil {
		return fmt.Errorf("statement not found: %w", err)
	}

	if statement.Status != "Reconciled" {
		return fmt.Errorf("statement is not reconciled")
	}

	if err := s.db.Model(&statement).Updates(map[string]any{
		"status":        "InProgress",
		"reconciled_at": nil,
		"reconciled_by": "",
	}).Error; err != nil {
		return fmt.Errorf("failed to reopen reconciliation: %w", err)
	}

	if err := s.LogReconciliationAction(statementID, nil, "REOPEN",
		map[string]any{"previous_status": "Reconciled", "reason": reason},
		user, false, 1.0, reason); err != nil {
		return err
	}

	log.Printf("Bank statement %s reopened by %s: %s", statementID, user, reason)
	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) GetReconciliationSummary(statementID string) (map[string]any, error) {
	if err := s.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var statement finance.BankStatement
	if err := s.db.First(&statement, "id = ?", statementID).Error; err != nil {
		return nil, fmt.Errorf("statement not found: %w", err)
	}

	var matchedCount, unmatchedCount int64
	s.db.Model(&finance.BankStatementLine{}).Where("bank_statement_id = ? AND is_matched = ?", statementID, true).Count(&matchedCount)
	s.db.Model(&finance.BankStatementLine{}).Where("bank_statement_id = ? AND is_matched = ?", statementID, false).Count(&unmatchedCount)

	totalCount := matchedCount + unmatchedCount
	matchedPercent := 0.0
	if totalCount > 0 {
		matchedPercent = float64(matchedCount) / float64(totalCount) * 100
	}

	var amounts struct {
		TotalDebits   float64
		TotalCredits  float64
		MatchedAmount float64
	}
	s.db.Model(&finance.BankStatementLine{}).
		Select("COALESCE(SUM(debit), 0) as total_debits, COALESCE(SUM(credit), 0) as total_credits, COALESCE(SUM(CASE WHEN is_matched THEN debit + credit ELSE 0 END), 0) as matched_amount").
		Where("bank_statement_id = ?", statementID).
		Scan(&amounts)

	return map[string]any{
		"statement_id":     statementID,
		"statement_number": statement.StatementNumber,
		"status":           statement.Status,
		"period_start":     statement.PeriodStart,
		"period_end":       statement.PeriodEnd,
		"opening_balance":  statement.OpeningBalance,
		"closing_balance":  statement.ClosingBalance,
		"total_debits":     amounts.TotalDebits,
		"total_credits":    amounts.TotalCredits,
		"total_lines":      totalCount,
		"matched_count":    matchedCount,
		"unmatched_count":  unmatchedCount,
		"matched_percent":  matchedPercent,
		"matched_amount":   amounts.MatchedAmount,
		"balance_verified": statement.BalanceVerified,
		"reconciled_at":    statement.ReconciledAt,
		"reconciled_by":    statement.ReconciledBy,
	}, nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) GetReconciliationStats(bankAccountID string) (map[string]any, error) {
	if err := s.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var stats struct {
		TotalStatements      int64
		ReconciledStatements int64
		InProgressStatements int64
		ImportedStatements   int64
	}

	query := s.db.Model(&finance.BankStatement{})
	if bankAccountID != "" {
		query = query.Where("bank_account_id = ?", bankAccountID)
	}

	query.Count(&stats.TotalStatements)
	query.Where("status = ?", "Reconciled").Count(&stats.ReconciledStatements)
	s.db.Model(&finance.BankStatement{}).Where("status = ?", "InProgress").Count(&stats.InProgressStatements)
	s.db.Model(&finance.BankStatement{}).Where("status = ?", "Imported").Count(&stats.ImportedStatements)

	var latestDate time.Time
	s.db.Model(&finance.BankStatement{}).
		Select("MAX(period_end)").
		Where("bank_account_id = ?", bankAccountID).
		Scan(&latestDate)

	return map[string]any{
		"bank_account_id":        bankAccountID,
		"total_statements":       stats.TotalStatements,
		"reconciled_statements":  stats.ReconciledStatements,
		"in_progress_statements": stats.InProgressStatements,
		"imported_statements":    stats.ImportedStatements,
		"latest_statement_date":  latestDate,
	}, nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) CreateBookBankReconciliation(bankAccountID string, reconciliationDate time.Time, bankStatementBalance, bookBalance, depositsInTransit, outstandingCheques float64) (*BookBankReconciliationModel, error) {
	return s.handlers.CreateBookBankReconciliation(bankAccountID, reconciliationDate, bankStatementBalance, bookBalance, depositsInTransit, outstandingCheques)
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) ValidateStatementContinuity(bankAccountID string, newStatement *finance.BankStatement) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if newStatement == nil {
		return fmt.Errorf("bank statement is required")
	}

	var lastStatement finance.BankStatement
	err := s.db.Where("bank_account_id = ? AND status IN ('Reconciled', 'Verified')", bankAccountID).
		Order("period_end DESC").
		First(&lastStatement).Error

	if err == nil {
		gap := math.Abs(lastStatement.ClosingBalance - newStatement.OpeningBalance)
		if gap > 0.001 {
			return fmt.Errorf(
				"BALANCE GAP DETECTED: Previous closing (%.3f) != New opening (%.3f). Gap: %.3f %s",
				lastStatement.ClosingBalance,
				newStatement.OpeningBalance,
				gap,
				newStatement.Currency,
			)
		}
	}

	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) GetBalanceContinuityReport(bankAccountID string) (*finance.BalanceContinuityReportData, error) {
	if err := s.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var bankAccount finance.CompanyBankAccount
	if err := s.db.First(&bankAccount, "id = ?", bankAccountID).Error; err != nil {
		return nil, fmt.Errorf("bank account not found: %w", err)
	}

	var statements []finance.BankStatement
	s.db.Where("bank_account_id = ?", bankAccountID).
		Order("period_end ASC").
		Find(&statements)

	report := &finance.BalanceContinuityReportData{
		BankAccountID:     bankAccountID,
		BankName:          bankAccount.BankName,
		IsContinuous:      true,
		StatementsCovered: len(statements),
	}

	for i := 1; i < len(statements); i++ {
		prev := statements[i-1]
		curr := statements[i]

		gap := curr.OpeningBalance - prev.ClosingBalance
		if math.Abs(gap) > 0.001 {
			report.Gaps = append(report.Gaps, finance.BalanceGap{
				FromStatementID: prev.ID,
				ToStatementID:   curr.ID,
				FromDate:        prev.PeriodEnd,
				ToDate:          curr.PeriodStart,
				ClosingBalance:  prev.ClosingBalance,
				OpeningBalance:  curr.OpeningBalance,
				GapAmount:       gap,
			})
			report.TotalGapAmount += math.Abs(gap)
			report.IsContinuous = false
		}
	}

	return report, nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) ComputeStatementHash(statement *finance.BankStatement, lines []finance.BankStatementLine) string {
	if err := s.requirePermission("finance:view"); err != nil {
		return ""
	}
	if statement == nil {
		return ""
	}

	firstRef := ""
	lastRef := ""
	if len(lines) > 0 {
		firstRef = lines[0].Reference
		lastRef = lines[len(lines)-1].Reference
	}

	data := fmt.Sprintf("%s|%s|%s|%d|%.3f|%s|%s",
		statement.BankAccountID,
		statement.PeriodStart.Format("2006-01-02"),
		statement.PeriodEnd.Format("2006-01-02"),
		len(lines),
		statement.ClosingBalance,
		firstRef,
		lastRef,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) CheckDuplicateStatement(statement *finance.BankStatement, lines []finance.BankStatementLine) (*finance.DuplicateStatementCheck, error) {
	if err := s.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	hash := s.ComputeStatementHash(statement, lines)

	var existing finance.StatementHash
	err := s.db.Where("statement_hash = ?", hash).First(&existing).Error
	if err == nil {
		return &finance.DuplicateStatementCheck{IsDuplicate: true, Existing: &existing}, fmt.Errorf(
			"DUPLICATE STATEMENT: This statement was already imported on %s (Statement ID: %s)",
			existing.ImportedAt.Format("2006-01-02 15:04"),
			existing.BankStatementID,
		)
	}

	return &finance.DuplicateStatementCheck{IsDuplicate: false}, nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) SaveStatementHash(statement *finance.BankStatement, lines []finance.BankStatementLine) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if statement == nil {
		return fmt.Errorf("bank statement is required")
	}

	hash := s.ComputeStatementHash(statement, lines)

	stmtHash := finance.StatementHash{
		BankAccountID:    statement.BankAccountID,
		StatementHash:    hash,
		PeriodStart:      statement.PeriodStart,
		PeriodEnd:        statement.PeriodEnd,
		TransactionCount: len(lines),
		ClosingBalance:   statement.ClosingBalance,
		ImportedAt:       time.Now(),
		BankStatementID:  statement.ID,
	}

	return s.db.Create(&stmtHash).Error
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) ForceReimportStatement(statementID string, user string, reason string) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	s.db.Where("bank_statement_id = ?", statementID).Delete(&finance.StatementHash{})
	s.db.Where("bank_statement_id = ?", statementID).Delete(&finance.BankStatementLine{})
	s.db.Delete(&finance.BankStatement{}, "id = ?", statementID)

	if err := s.LogReconciliationAction(statementID, nil, "FORCE_REIMPORT",
		map[string]any{"reason": reason},
		user, false, 1.0, reason); err != nil {
		return err
	}

	log.Printf("Statement %s force reimported by %s: %s", statementID, user, reason)
	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) LogReconciliationAction(statementID string, lineID *string, action string, detail any, user string, isAuto bool, confidence float64, reason string) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	detailJSON, _ := json.Marshal(detail)

	logEntry := finance.BankReconciliationAuditLog{
		BankStatementID:     statementID,
		BankStatementLineID: lineID,
		Action:              action,
		ActionDetail:        string(detailJSON),
		PerformedBy:         user,
		PerformedAt:         time.Now(),
		IsAutomatic:         isAuto,
		ConfidenceScore:     confidence,
		Reason:              reason,
	}

	if err := s.db.Create(&logEntry).Error; err != nil {
		log.Printf("failed to log reconciliation action: %v", err)
		return err
	}

	return nil
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) GetAuditTrail(statementID string) ([]finance.BankReconciliationAuditLog, error) {
	if err := s.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var logs []finance.BankReconciliationAuditLog
	err := s.db.Where("bank_statement_id = ?", statementID).
		Order("performed_at ASC").
		Find(&logs).Error

	return logs, err
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) GetAuditTrailByDateRange(bankAccountID string, startDate time.Time, endDate time.Time) ([]finance.BankReconciliationAuditLog, error) {
	if err := s.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var logs []finance.BankReconciliationAuditLog
	query := s.db.Where("performed_at BETWEEN ? AND ?", startDate, endDate)

	if bankAccountID != "" {
		query = query.Joins("JOIN bank_statements bs ON bs.id = bank_reconciliation_audit_logs.bank_statement_id").
			Where("bs.bank_account_id = ?", bankAccountID)
	}

	err := query.Order("performed_at ASC").Find(&logs).Error
	return logs, err
}

func (s *Service[BookBankReconciliationModel, MatchResultModel, AllocationInputModel, BankStatementModel, BankStatementLineModel, BalanceContinuityReportModel, StatementHashModel, AuditLogModel]) ReverseAction(logID string, user string, reason string) error {
	if err := s.requirePermission("finance:create"); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now()
	return s.db.Model(&finance.BankReconciliationAuditLog{}).
		Where("id = ?", logID).
		Updates(map[string]any{
			"is_reversed":     true,
			"reversed_by":     user,
			"reversed_at":     now,
			"reversal_reason": reason,
		}).Error
}
