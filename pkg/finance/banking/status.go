package banking

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/finance"
)

// Bank statement status vocabulary (PC-D4, ported from PH's reconciliation
// policy seam). Reconciled/Verified are FINAL: an audited final state must not
// silently un-finalize — edits refuse until the statement is explicitly
// reopened (ReopenReconciliation).
const (
	StatementStatusImported   = "Imported"
	StatementStatusInProgress = "InProgress"
	StatementStatusReconciled = "Reconciled"
	StatementStatusVerified   = "Verified"
	StatementStatusCancelled  = "Cancelled"
)

// NormalizeStatementStatus canonicalizes a raw status string ("in progress",
// "RECONCILED") onto the vocabulary, rejecting blanks and unknown values.
func NormalizeStatementStatus(raw string) (string, error) {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), " ", ""))
	switch normalized {
	case strings.ToLower(StatementStatusImported):
		return StatementStatusImported, nil
	case strings.ToLower(StatementStatusInProgress):
		return StatementStatusInProgress, nil
	case strings.ToLower(StatementStatusReconciled):
		return StatementStatusReconciled, nil
	case strings.ToLower(StatementStatusVerified):
		return StatementStatusVerified, nil
	case strings.ToLower(StatementStatusCancelled):
		return StatementStatusCancelled, nil
	case "":
		return "", fmt.Errorf("statement status cannot be blank")
	default:
		return "", fmt.Errorf("invalid bank statement status: %s", raw)
	}
}

// DisplayStatementStatus renders a status for operator-facing messages.
func DisplayStatementStatus(status string) string {
	normalized, err := NormalizeStatementStatus(status)
	if err != nil {
		return strings.TrimSpace(status)
	}
	if normalized == StatementStatusInProgress {
		return "In Progress"
	}
	return normalized
}

// IsFinalStatementStatus reports whether the status is a terminal,
// audit-relevant state.
func IsFinalStatementStatus(status string) bool {
	switch status {
	case StatementStatusReconciled, StatementStatusVerified:
		return true
	default:
		return false
	}
}

// IsEditableStatementStatus reports whether the status may be set directly by
// an edit; final states are only reachable through the reconciliation
// workflow (FinalizeReconciliation) and left through ReopenReconciliation.
func IsEditableStatementStatus(status string) bool {
	switch status {
	case StatementStatusImported, StatementStatusInProgress:
		return true
	default:
		return false
	}
}

// normalizeStatementStatusUpdate canonicalizes a status value inside an
// updates map and refuses direct writes into final states.
func normalizeStatementStatusUpdate(updates map[string]any) error {
	rawStatus, ok := updates["status"]
	if !ok {
		return nil
	}
	status, err := NormalizeStatementStatus(fmt.Sprintf("%v", rawStatus))
	if err != nil {
		return err
	}
	if !IsEditableStatementStatus(status) {
		return fmt.Errorf("cannot set statement status to %s directly; use the reconciliation workflow", DisplayStatementStatus(status))
	}
	updates["status"] = status
	return nil
}

// ensureStatementMutableTx loads the statement's status and refuses the
// action when the statement is in a final state.
func ensureStatementMutableTx(tx *gorm.DB, statementID, action string) error {
	var status string
	if err := tx.Model(&finance.BankStatement{}).Where("id = ?", statementID).
		Select("status").Scan(&status).Error; err != nil {
		return fmt.Errorf("statement not found: %w", err)
	}
	if !IsFinalStatementStatus(status) {
		return nil
	}
	return fmt.Errorf("cannot %s on a %s statement; reopen the reconciliation first", action, strings.ToLower(DisplayStatementStatus(status)))
}
