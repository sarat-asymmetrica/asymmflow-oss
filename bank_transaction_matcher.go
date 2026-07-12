// =============================================================================
// BANK TRANSACTION MATCHER SERVICE
//
// MISSION: Wails-facing wrappers for banking package matching and allocation.
// =============================================================================

package main

import financebanking "ph_holdings_app/pkg/finance/banking"

const (
	TxTypeCustomerPayment     = financebanking.TxTypeCustomerPayment
	TxTypeChequeReceived      = financebanking.TxTypeChequeReceived
	TxTypeSupplierPayment     = financebanking.TxTypeSupplierPayment
	TxTypeChequeIssued        = financebanking.TxTypeChequeIssued
	TxTypeBankFee             = financebanking.TxTypeBankFee
	TxTypeSwiftCharge         = financebanking.TxTypeSwiftCharge
	TxTypeVATOnFee            = financebanking.TxTypeVATOnFee
	TxTypeBGFee               = financebanking.TxTypeBGFee
	TxTypeCorrespondentCharge = financebanking.TxTypeCorrespondentCharge
	TxTypeInternalTransfer    = financebanking.TxTypeInternalTransfer
	TxTypeFXConversion        = financebanking.TxTypeFXConversion
	TxTypeSalary              = financebanking.TxTypeSalary
	TxTypeBillPayment         = financebanking.TxTypeBillPayment
	TxTypeOther               = financebanking.TxTypeOther
)

type BankReconciliationMatchResult = financebanking.BankReconciliationMatchResult
type AllocationInput = financebanking.AllocationInput

func (a *App) AutoMatchBankLines(statementID string) (*BankReconciliationMatchResult, error) {
	return a.bankingService().AutoMatchBankLines(statementID)
}

func (a *App) ManualMatchLine(lineID, entityType, entityID, user string) error {
	// Wave 9.3 B2: identity resolved server-side; client value ignored (Article III.4)
	return a.bankingService().ManualMatchLine(lineID, entityType, entityID, a.getCurrentUserID())
}

func (a *App) UnmatchLine(lineID, user, reason string) error {
	// Wave 9.3 B2: identity resolved server-side; client value ignored (Article III.4)
	return a.bankingService().UnmatchLine(lineID, a.getCurrentUserID(), reason)
}

func (a *App) CreateSplitAllocation(lineID string, allocations []AllocationInput, user string) error {
	// Wave 9.3 B2: identity resolved server-side; client value ignored (Article III.4)
	return a.bankingService().CreateSplitAllocation(lineID, allocations, a.getCurrentUserID())
}

func (a *App) CategorizeTransactions(statementID string) error {
	return a.bankingService().CategorizeTransactions(statementID)
}
