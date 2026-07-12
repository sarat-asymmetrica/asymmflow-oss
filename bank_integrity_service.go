// =============================================================================
// BANK INTEGRITY SERVICE
//
// MISSION: Ensure data integrity for bank reconciliation
// FEATURES: Continuity validation, duplicate detection, audit trail, PDF archival
// =============================================================================

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	finance "ph_holdings_app/pkg/finance"
)

// =============================================================================
// STATEMENT CONTINUITY
// =============================================================================

// ValidateStatementContinuity ensures previous closing = new opening
func (a *App) ValidateStatementContinuity(bankAccountID string, newStatement *BankStatement) error {
	return a.bankingService().ValidateStatementContinuity(bankAccountID, newStatement)
}

// BalanceGap represents a gap in balance continuity
type BalanceGap = finance.BalanceGap

// BalanceContinuityReport provides a full continuity audit
type BalanceContinuityReportData = finance.BalanceContinuityReportData

// GetBalanceContinuityReport generates a continuity report for an account
func (a *App) GetBalanceContinuityReport(bankAccountID string) (*BalanceContinuityReportData, error) {
	return a.bankingService().GetBalanceContinuityReport(bankAccountID)
}

// =============================================================================
// DUPLICATE DETECTION
// =============================================================================

// ComputeStatementHash generates a unique SHA-256 hash for a statement
func (a *App) ComputeStatementHash(statement *BankStatement, lines []BankStatementLine) string {
	return a.bankingService().ComputeStatementHash(statement, lines)
}

// CheckDuplicateStatement checks if a statement already exists
func (a *App) CheckDuplicateStatement(statement *BankStatement, lines []BankStatementLine) (*DuplicateStatementCheck, error) {
	return a.bankingService().CheckDuplicateStatement(statement, lines)
}

// SaveStatementHash saves the hash for a new statement
func (a *App) SaveStatementHash(statement *BankStatement, lines []BankStatementLine) error {
	return a.bankingService().SaveStatementHash(statement, lines)
}

// ForceReimportStatement allows re-import with audit trail
func (a *App) ForceReimportStatement(statementID string, user, reason string) error {
	return a.bankingService().ForceReimportStatement(statementID, user, reason)
}

// =============================================================================
// AUDIT TRAIL
// =============================================================================

// LogReconciliationAction logs any reconciliation action
func (a *App) LogReconciliationAction(
	statementID string,
	lineID *string,
	action string,
	detail any,
	user string,
	isAuto bool,
	confidence float64,
	reason string,
) error {
	return a.bankingService().LogReconciliationAction(statementID, lineID, action, detail, user, isAuto, confidence, reason)
}

// GetAuditTrail retrieves all audit logs for a statement
func (a *App) GetAuditTrail(statementID string) ([]BankReconciliationAuditLog, error) {
	return a.bankingService().GetAuditTrail(statementID)
}

// GetAuditTrailByDateRange retrieves audit logs for a date range
func (a *App) GetAuditTrailByDateRange(bankAccountID string, startDate, endDate time.Time) ([]BankReconciliationAuditLog, error) {
	return a.bankingService().GetAuditTrailByDateRange(bankAccountID, startDate, endDate)
}

// ReverseAction marks an audit log entry as reversed
func (a *App) ReverseAction(logID string, user, reason string) error {
	// Wave 9.3 B2: identity resolved server-side; client value ignored (Article III.4)
	return a.bankingService().ReverseAction(logID, a.getCurrentUserID(), reason)
}

// =============================================================================
// PDF ARCHIVAL
// =============================================================================

// ArchiveStatementPDF stores the original PDF for audit trail
func (a *App) ArchiveStatementPDF(statementID string, filePath string) error {
	if err := a.requirePermission("finance:create"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file info: %w", err)
	}

	// Calculate file hash
	fileHash, err := a.GetFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Create archive directory
	archiveDir := filepath.Join("data", "bank_statements", statementID)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		log.Printf("⚠️ Failed to create archive directory: %v", err)
		// Continue even if directory creation fails - we'll store the path
	}

	// Copy file to archive
	destPath := filepath.Join(archiveDir, filepath.Base(filePath))
	if err := copyFile(filePath, destPath); err != nil {
		log.Printf("⚠️ Failed to archive file: %v", err)
		destPath = filePath // Keep original path if copy fails
	}

	// Determine file type
	ext := filepath.Ext(filePath)
	fileType := "PDF"
	if ext == ".csv" {
		fileType = "CSV"
	} else if ext == ".xls" || ext == ".xlsx" {
		fileType = "XLS"
	}

	// Create archive record
	archive := BankStatementFile{
		BankStatementID: statementID,
		FileName:        filepath.Base(filePath),
		FileType:        fileType,
		FileSize:        fileInfo.Size(),
		FileHash:        fileHash,
		StoragePath:     destPath,
		IsStored:        destPath != filePath,
	}

	if err := a.db.Create(&archive).Error; err != nil {
		return fmt.Errorf("failed to create archive record: %w", err)
	}

	log.Printf("📁 Archived statement file: %s (%d bytes)", filepath.Base(filePath), fileInfo.Size())
	return nil
}

// ArchivedFileResult bundles archived file bytes with its original file
// name. Wails v2's bound-method marshaling only handles OutputCount 1 or 2
// (see internal/binding/boundMethod.go) — a 3-value Go return silently
// marshals to null on the JS side. Bundling into a struct + error keeps
// the binding a clean 2-value return.
type ArchivedFileResult struct {
	Data     []byte `json:"data"`
	FileName string `json:"file_name"`
}

// RetrieveOriginalPDF returns the archived PDF content
func (a *App) RetrieveOriginalPDF(statementID string) (*ArchivedFileResult, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var archive BankStatementFile
	if err := a.db.Where("bank_statement_id = ?", statementID).First(&archive).Error; err != nil {
		return nil, fmt.Errorf("archived file not found: %w", err)
	}

	data, err := os.ReadFile(archive.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read archived file: %w", err)
	}

	return &ArchivedFileResult{Data: data, FileName: archive.FileName}, nil
}

// GetFileHash calculates SHA-256 hash of a file
func (a *App) GetFileHash(filePath string) (string, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return "", err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// =============================================================================
// COMPLIANCE REPORTS
// =============================================================================

// GenerateAuditReport creates a comprehensive audit report
func (a *App) GenerateAuditReport(bankAccountID string, startDate, endDate time.Time) (map[string]any, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Get account info
	var bankAccount CompanyBankAccount
	if err := a.db.First(&bankAccount, "id = ?", bankAccountID).Error; err != nil {
		return nil, fmt.Errorf("bank account not found: %w", err)
	}

	// Get statements in period
	var statements []BankStatement
	a.db.Where("bank_account_id = ? AND period_start >= ? AND period_end <= ?",
		bankAccountID, startDate, endDate).
		Order("period_start ASC").
		Find(&statements)

	// Get audit logs
	logs, _ := a.GetAuditTrailByDateRange(bankAccountID, startDate, endDate)

	// Get continuity report
	continuity, _ := a.GetBalanceContinuityReport(bankAccountID)

	// Count by action type
	actionCounts := make(map[string]int)
	for _, log := range logs {
		actionCounts[log.Action]++
	}

	return map[string]any{
		"bank_account":     bankAccount.BankName,
		"account_number":   bankAccount.AccountNumber,
		"period_start":     startDate,
		"period_end":       endDate,
		"statements_count": len(statements),
		"statements":       statements,
		"audit_logs_count": len(logs),
		"action_counts":    actionCounts,
		"is_continuous":    continuity.IsContinuous,
		"gaps":             continuity.Gaps,
		"generated_at":     time.Now(),
	}, nil
}

// =============================================================================
// HELPERS
// =============================================================================

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
