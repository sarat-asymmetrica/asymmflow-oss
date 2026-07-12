// =============================================================================
// CHEQUE REGISTER SERVICE
//
// MISSION: Manage cheque lifecycle from issuance to clearance
// FEATURES: Cheque books, issuance, clearance tracking, stale cheque handling
//
// Wave 5 A.1: the lifecycle logic lives in pkg/finance/cheque. These
// delegates keep the Wails binding surface and the RBAC guards.
// =============================================================================

package main

import (
	"fmt"
	"time"

	financecheque "ph_holdings_app/pkg/finance/cheque"
)

// OutstandingChequesResult mirrors financecheque.OutstandingResult at the
// Wails binding boundary (keeps the JSON contract stable at root — same
// pattern as FXExposureReport in fx_revaluation_service.go).
type OutstandingChequesResult = financecheque.OutstandingResult

func (a *App) chequesGuarded(permission string) error {
	if err := a.requirePermission(permission); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return nil
}

// CreateChequeRegister creates a new cheque book register
func (a *App) CreateChequeRegister(bankAccountID, chequeBookNo string, startNum, endNum int) (*ChequeRegister, error) {
	if err := a.chequesGuarded("finance:create"); err != nil {
		return nil, err
	}
	return a.chequeService().CreateRegister(bankAccountID, chequeBookNo, startNum, endNum)
}

// GetChequeRegisters retrieves all registers for a bank account
func (a *App) GetChequeRegisters(bankAccountID string) ([]ChequeRegister, error) {
	if err := a.chequesGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.chequeService().Registers(bankAccountID)
}

// GetActiveChequeRegister gets the active register for a bank account
func (a *App) GetActiveChequeRegister(bankAccountID string) (*ChequeRegister, error) {
	if err := a.chequesGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.chequeService().ActiveRegister(bankAccountID)
}

// GetNextChequeNumber returns the next available cheque number
func (a *App) GetNextChequeNumber(bankAccountID string) (string, error) {
	if err := a.chequesGuarded("finance:view"); err != nil {
		return "", err
	}
	return a.chequeService().NextNumber(bankAccountID)
}

// ExhaustChequeRegister marks a register as exhausted
func (a *App) ExhaustChequeRegister(registerID string) error {
	if err := a.chequesGuarded("finance:create"); err != nil {
		return err
	}
	return a.chequeService().Exhaust(registerID)
}

// IssueCheque issues a new cheque and records it as outstanding
func (a *App) IssueCheque(bankAccountID string, amount float64, payeeName, payeeType string, supplierID *string, purpose string) (*OutstandingCheque, error) {
	if err := a.chequesGuarded("finance:create"); err != nil {
		return nil, err
	}
	return a.chequeService().Issue(bankAccountID, amount, payeeName, payeeType, supplierID, purpose)
}

// MarkChequePresented marks a cheque as presented
func (a *App) MarkChequePresented(chequeNumber string) error {
	if err := a.chequesGuarded("finance:create"); err != nil {
		return err
	}
	return a.chequeService().MarkPresented(chequeNumber)
}

// MarkChequeCleared marks a cheque as cleared and links to bank statement line
func (a *App) MarkChequeCleared(chequeNumber, bankStatementLineID string, clearedDate time.Time) error {
	if err := a.chequesGuarded("finance:create"); err != nil {
		return err
	}
	return a.chequeService().MarkCleared(chequeNumber, bankStatementLineID, clearedDate)
}

// MarkChequeStale marks a cheque as stale (>6 months)
func (a *App) MarkChequeStale(chequeNumber string) error {
	if err := a.chequesGuarded("finance:create"); err != nil {
		return err
	}
	return a.chequeService().MarkStale(chequeNumber)
}

// MarkChequeBounced marks a cheque as bounced
func (a *App) MarkChequeBounced(chequeNumber, reason string) error {
	if err := a.chequesGuarded("finance:create"); err != nil {
		return err
	}
	return a.chequeService().MarkBounced(chequeNumber, reason)
}

// CancelCheque cancels a cheque
func (a *App) CancelCheque(chequeNumber, reason string) error {
	if err := a.chequesGuarded("finance:create"); err != nil {
		return err
	}
	return a.chequeService().Cancel(chequeNumber, reason)
}

// ReissueCheque issues a new cheque to replace a stale/cancelled one
func (a *App) ReissueCheque(oldChequeNumber, bankAccountID string) (*OutstandingCheque, error) {
	if err := a.chequesGuarded("finance:create"); err != nil {
		return nil, err
	}
	return a.chequeService().Reissue(oldChequeNumber, bankAccountID)
}

// GetOutstandingCheques returns all outstanding (not cleared) cheques
func (a *App) GetOutstandingCheques(bankAccountID string) (*OutstandingChequesResult, error) {
	if err := a.chequesGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.chequeService().Outstanding(bankAccountID)
}

// GetChequeByNumber retrieves a cheque by its number
func (a *App) GetChequeByNumber(chequeNumber string) (*OutstandingCheque, error) {
	if err := a.chequesGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.chequeService().ByNumber(chequeNumber)
}

// GetChequesByStatus retrieves cheques by status
func (a *App) GetChequesByStatus(bankAccountID, status string) ([]OutstandingCheque, error) {
	if err := a.chequesGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.chequeService().ByStatus(bankAccountID, status)
}

// GetStaleCheques returns cheques that are >6 months old and not cleared
func (a *App) GetStaleCheques(bankAccountID string) ([]OutstandingCheque, error) {
	if err := a.chequesGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.chequeService().StaleCheques(bankAccountID)
}

// GetChequeRegisterReport generates a report for a date range
func (a *App) GetChequeRegisterReport(bankAccountID string, startDate, endDate time.Time) (map[string]any, error) {
	if err := a.chequesGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.chequeService().Report(bankAccountID, startDate, endDate)
}
