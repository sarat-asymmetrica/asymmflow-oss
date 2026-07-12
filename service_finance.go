package main

import (
	financevm "ph_holdings_app/internal/viewmodel/finance"
	cashflowevidence "ph_holdings_app/pkg/cashflow/evidence"
	"ph_holdings_app/pkg/finance/posting"
	"time"
)

// FinanceService exposes domain-specific Wails bindings by delegating to App.
type FinanceService struct {
	app *App
}

func NewFinanceService(app *App) *FinanceService {
	return &FinanceService{app: app}
}

// --- app_accounting_inventory.go ---

func (s *FinanceService) ApproveStockAdjustment(adjustmentID string) error {
	return s.app.ApproveStockAdjustment(adjustmentID)
}

func (s *FinanceService) CreateAccount(account ChartOfAccount) (*ChartOfAccount, error) {
	return s.app.CreateAccount(account)
}

func (s *FinanceService) CreateInventoryItem(item InventoryItem) (*InventoryItem, error) {
	return s.app.CreateInventoryItem(item)
}

func (s *FinanceService) CreateJournalEntry(entry JournalEntry) (*JournalEntry, error) {
	return s.app.CreateJournalEntry(entry)
}

func (s *FinanceService) CreateDraftJournalFromPosting(sourceType string, sourceID string) (*JournalEntry, error) {
	return s.app.CreateDraftJournalFromPosting(sourceType, sourceID)
}

func (s *FinanceService) CreatePostSaleNote(note PostSaleNote) (*PostSaleNote, error) {
	return s.app.CreatePostSaleNote(note)
}

func (s *FinanceService) CreateStockAdjustment(adjustment StockAdjustment) error {
	return s.app.CreateStockAdjustment(adjustment)
}

func (s *FinanceService) CreateWarehouse(warehouse Warehouse) (*Warehouse, error) {
	return s.app.CreateWarehouse(warehouse)
}

func (s *FinanceService) DeletePostSaleNote(noteID uint) error {
	return s.app.DeletePostSaleNote(noteID)
}

func (s *FinanceService) FileVATReturn(returnID uint) error {
	return s.app.FileVATReturn(returnID)
}

func (s *FinanceService) GenerateVATReturn(periodStart, periodEnd time.Time) (*VATReturn, error) {
	return s.app.GenerateVATReturn(periodStart, periodEnd)
}

func (s *FinanceService) GetAPAgingReport() (*ARAgingReport, error) {
	return s.app.GetAPAgingReport()
}

func (s *FinanceService) GetARAgingReport() (*ARAgingReport, error) {
	return s.app.GetARAgingReport()
}

func (s *FinanceService) GetBalanceSheet(asOfDate time.Time) (*BalanceSheetReport, error) {
	return s.app.GetBalanceSheet(asOfDate)
}

func (s *FinanceService) GetChartOfAccounts(accountType string) ([]ChartOfAccount, error) {
	return s.app.GetChartOfAccounts(accountType)
}

func (s *FinanceService) GetInventoryItem(itemID string) (*InventoryItem, error) {
	return s.app.GetInventoryItem(itemID)
}

func (s *FinanceService) GetInventoryItems(warehouseID *string, stockStatus string, lowStockOnly bool) ([]InventoryItem, error) {
	return s.app.GetInventoryItems(warehouseID, stockStatus, lowStockOnly)
}

func (s *FinanceService) GetInventoryValuation(warehouseID *string) (map[string]any, error) {
	return s.app.GetInventoryValuation(warehouseID)
}

func (s *FinanceService) GetJournalEntries(fiscalYear int, fiscalPeriod int, isPosted *bool, limit int) ([]JournalEntry, error) {
	return s.app.GetJournalEntries(fiscalYear, fiscalPeriod, isPosted, limit)
}

func (s *FinanceService) GetPostingCoverageReport() (posting.CoverageReport, error) {
	return s.app.GetPostingCoverageReport()
}

func (s *FinanceService) GetTrialBalanceGate(fiscalYear int, fiscalPeriod int) (posting.TrialBalanceGate, error) {
	return s.app.GetTrialBalanceGate(fiscalYear, fiscalPeriod)
}

func (s *FinanceService) GetLowStockItems() ([]InventoryItem, error) {
	return s.app.GetLowStockItems()
}

func (s *FinanceService) GetPostSaleNotes(orderID string) ([]PostSaleNote, error) {
	return s.app.GetPostSaleNotes(orderID)
}

func (s *FinanceService) GetProfitLoss(periodStart, periodEnd time.Time) (*ProfitLossReport, error) {
	return s.app.GetProfitLoss(periodStart, periodEnd)
}

func (s *FinanceService) GetSalesPipeline() ([]SalesPipelineData, error) {
	return s.app.GetSalesPipeline()
}

func (s *FinanceService) GetStockMovements(itemID *string, movementType string, startDate, endDate time.Time, limit int) ([]StockMovement, error) {
	return s.app.GetStockMovements(itemID, movementType, startDate, endDate, limit)
}

func (s *FinanceService) GetVATReturns(fiscalYear int, status string) ([]VATReturn, error) {
	return s.app.GetVATReturns(fiscalYear, status)
}

func (s *FinanceService) GetWarehouses() ([]Warehouse, error) {
	return s.app.GetWarehouses()
}

func (s *FinanceService) PostJournalEntry(entryID uint) error {
	return s.app.PostJournalEntry(entryID)
}

func (s *FinanceService) PreviewCustomerInvoicePosting(invoiceID string) (posting.Entry, error) {
	return s.app.PreviewCustomerInvoicePosting(invoiceID)
}

func (s *FinanceService) PreviewCustomerPaymentPosting(paymentID string) (posting.Entry, error) {
	return s.app.PreviewCustomerPaymentPosting(paymentID)
}

func (s *FinanceService) PreviewSupplierInvoicePosting(invoiceID string) (posting.Entry, error) {
	return s.app.PreviewSupplierInvoicePosting(invoiceID)
}

func (s *FinanceService) PreviewSupplierPaymentPosting(paymentID string) (posting.Entry, error) {
	return s.app.PreviewSupplierPaymentPosting(paymentID)
}

func (s *FinanceService) RecordStockMovement(movement StockMovement) (*StockMovement, error) {
	return s.app.RecordStockMovement(movement)
}

func (s *FinanceService) ReverseJournalEntry(entryID string, reason string) (*JournalEntry, error) {
	return s.app.ReverseJournalEntry(entryID, reason)
}

func (s *FinanceService) SeedDefaultChartOfAccounts() error {
	return s.app.SeedDefaultChartOfAccounts()
}

func (s *FinanceService) UpdateAccount(accountID string, updates map[string]any) error {
	return s.app.UpdateAccount(accountID, updates)
}

func (s *FinanceService) UpdateInventoryItem(itemID string, updates map[string]any) error {
	return s.app.UpdateInventoryItem(itemID, updates)
}

func (s *FinanceService) UpdatePostSaleNote(note PostSaleNote) (*PostSaleNote, error) {
	return s.app.UpdatePostSaleNote(note)
}

func (s *FinanceService) ValidateJournalEntryBalance(entryID string) (bool, float64, float64, error) {
	return s.app.ValidateJournalEntryBalance(entryID)
}

// --- bank_accounts_service.go ---

func (s *FinanceService) CreateBankAccount(account CompanyBankAccount) (*CompanyBankAccount, error) {
	return s.app.CreateBankAccount(account)
}

func (s *FinanceService) DeleteBankAccount(id string) error {
	return s.app.DeleteBankAccount(id)
}

func (s *FinanceService) GetActiveBankAccounts() ([]CompanyBankAccount, error) {
	return s.app.GetActiveBankAccounts()
}

func (s *FinanceService) GetAllBankAccounts() ([]CompanyBankAccount, error) {
	return s.app.GetAllBankAccounts()
}

func (s *FinanceService) GetBankAccountByID(id string) (*CompanyBankAccount, error) {
	return s.app.GetBankAccountByID(id)
}

func (s *FinanceService) MigrateBankAccountEncryption() {
	s.app.MigrateBankAccountEncryption()
}

func (s *FinanceService) SeedCompanyBankAccounts() error {
	return s.app.SeedCompanyBankAccounts()
}

func (s *FinanceService) UpdateBankAccount(id string, updates map[string]any) (*CompanyBankAccount, error) {
	return s.app.UpdateBankAccount(id, updates)
}

// --- bank_integrity_service.go ---

func (s *FinanceService) ArchiveStatementPDF(statementID string, filePath string) error {
	return s.app.ArchiveStatementPDF(statementID, filePath)
}

func (s *FinanceService) CheckDuplicateStatement(statement *BankStatement, lines []BankStatementLine) (*DuplicateStatementCheck, error) {
	return s.app.CheckDuplicateStatement(statement, lines)
}

func (s *FinanceService) ComputeStatementHash(statement *BankStatement, lines []BankStatementLine) string {
	return s.app.ComputeStatementHash(statement, lines)
}

func (s *FinanceService) ForceReimportStatement(statementID string, user, reason string) error {
	return s.app.ForceReimportStatement(statementID, user, reason)
}

func (s *FinanceService) GenerateAuditReport(bankAccountID string, startDate, endDate time.Time) (map[string]any, error) {
	return s.app.GenerateAuditReport(bankAccountID, startDate, endDate)
}

func (s *FinanceService) GetAuditTrail(statementID string) ([]BankReconciliationAuditLog, error) {
	return s.app.GetAuditTrail(statementID)
}

func (s *FinanceService) GetAuditTrailByDateRange(bankAccountID string, startDate, endDate time.Time) ([]BankReconciliationAuditLog, error) {
	return s.app.GetAuditTrailByDateRange(bankAccountID, startDate, endDate)
}

func (s *FinanceService) GetBalanceContinuityReport(bankAccountID string) (*BalanceContinuityReportData, error) {
	return s.app.GetBalanceContinuityReport(bankAccountID)
}

func (s *FinanceService) GetFileHash(filePath string) (string, error) {
	return s.app.GetFileHash(filePath)
}

func (s *FinanceService) LogReconciliationAction(statementID string, lineID *string, action string, detail any, user string, isAuto bool, confidence float64, reason string) error {
	return s.app.LogReconciliationAction(statementID, lineID, action, detail, user, isAuto, confidence, reason)
}

func (s *FinanceService) RetrieveOriginalPDF(statementID string) (*ArchivedFileResult, error) {
	return s.app.RetrieveOriginalPDF(statementID)
}

func (s *FinanceService) ReverseAction(logID string, user, reason string) error {
	return s.app.ReverseAction(logID, user, reason)
}

func (s *FinanceService) SaveStatementHash(statement *BankStatement, lines []BankStatementLine) error {
	return s.app.SaveStatementHash(statement, lines)
}

func (s *FinanceService) ValidateStatementContinuity(bankAccountID string, newStatement *BankStatement) error {
	return s.app.ValidateStatementContinuity(bankAccountID, newStatement)
}

// --- bank_reconciliation_service.go ---

func (s *FinanceService) CreateBankStatement(statement *BankStatement) error {
	return s.app.CreateBankStatement(statement)
}

func (s *FinanceService) CreateBankStatementLine(statementID string, line map[string]any) (*BankStatementLine, error) {
	return s.app.CreateBankStatementLine(statementID, line)
}

func (s *FinanceService) DeleteBankStatement(id string) error {
	return s.app.DeleteBankStatement(id)
}

func (s *FinanceService) DeleteBankStatementLine(lineID string) error {
	return s.app.DeleteBankStatementLine(lineID)
}

func (s *FinanceService) FinalizeReconciliation(statementID string, reconciledBy string) error {
	return s.app.FinalizeReconciliation(statementID, reconciledBy)
}

func (s *FinanceService) GetBankStatementByID(id string) (*BankStatement, error) {
	return s.app.GetBankStatementByID(id)
}

func (s *FinanceService) GetBankStatementLines(statementID string) ([]BankStatementLine, error) {
	return s.app.GetBankStatementLines(statementID)
}

func (s *FinanceService) GetBankStatements(bankAccountID string) ([]BankStatement, error) {
	return s.app.GetBankStatements(bankAccountID)
}

func (s *FinanceService) GetCashPosition() (map[string]any, error) {
	return s.app.GetCashPosition()
}

func (s *FinanceService) GetCashPositionByAccount(bankAccountID string) (float64, error) {
	return s.app.GetCashPositionByAccount(bankAccountID)
}

func (s *FinanceService) GetReconciliationStats(bankAccountID string) (map[string]any, error) {
	return s.app.GetReconciliationStats(bankAccountID)
}

func (s *FinanceService) GetReconciliationSummary(statementID string) (map[string]any, error) {
	return s.app.GetReconciliationSummary(statementID)
}

func (s *FinanceService) GetUnmatchedLines(statementID string) ([]BankStatementLine, error) {
	return s.app.GetUnmatchedLines(statementID)
}

func (s *FinanceService) ReopenReconciliation(statementID string, user, reason string) error {
	return s.app.ReopenReconciliation(statementID, user, reason)
}

func (s *FinanceService) UpdateBankStatement(id string, updates map[string]any) error {
	return s.app.UpdateBankStatement(id, updates)
}

func (s *FinanceService) UpdateBankStatementLine(lineID string, updates map[string]any) error {
	return s.app.UpdateBankStatementLine(lineID, updates)
}

func (s *FinanceService) ValidateStatementBalance(statementID string) (*StatementBalanceValidation, error) {
	return s.app.ValidateStatementBalance(statementID)
}

// --- bank_statement_parser.go ---

func (s *FinanceService) ImportBankStatementCSV(filePath string, bankAccountID string) (*BankStatement, error) {
	return s.app.ImportBankStatementCSV(filePath, bankAccountID)
}

func (s *FinanceService) ImportBankStatementPDF(filePath string, bankAccountID string) (*BankStatement, error) {
	return s.app.ImportBankStatementPDF(filePath, bankAccountID)
}

// --- bank_transaction_matcher.go ---

func (s *FinanceService) AutoMatchBankLines(statementID string) (*BankReconciliationMatchResult, error) {
	return s.app.AutoMatchBankLines(statementID)
}

func (s *FinanceService) CategorizeTransactions(statementID string) error {
	return s.app.CategorizeTransactions(statementID)
}

func (s *FinanceService) CreateSplitAllocation(lineID string, allocations []AllocationInput, user string) error {
	return s.app.CreateSplitAllocation(lineID, allocations, user)
}

func (s *FinanceService) ManualMatchLine(lineID, entityType, entityID, user string) error {
	return s.app.ManualMatchLine(lineID, entityType, entityID, user)
}

func (s *FinanceService) UnmatchLine(lineID, user, reason string) error {
	return s.app.UnmatchLine(lineID, user, reason)
}

// --- cheque_register_service.go ---

func (s *FinanceService) CancelCheque(chequeNumber, reason string) error {
	return s.app.CancelCheque(chequeNumber, reason)
}

func (s *FinanceService) CreateChequeRegister(bankAccountID, chequeBookNo string, startNum, endNum int) (*ChequeRegister, error) {
	return s.app.CreateChequeRegister(bankAccountID, chequeBookNo, startNum, endNum)
}

func (s *FinanceService) ExhaustChequeRegister(registerID string) error {
	return s.app.ExhaustChequeRegister(registerID)
}

func (s *FinanceService) GetActiveChequeRegister(bankAccountID string) (*ChequeRegister, error) {
	return s.app.GetActiveChequeRegister(bankAccountID)
}

func (s *FinanceService) GetChequeByNumber(chequeNumber string) (*OutstandingCheque, error) {
	return s.app.GetChequeByNumber(chequeNumber)
}

func (s *FinanceService) GetChequeRegisterReport(bankAccountID string, startDate, endDate time.Time) (map[string]any, error) {
	return s.app.GetChequeRegisterReport(bankAccountID, startDate, endDate)
}

func (s *FinanceService) GetChequeRegisters(bankAccountID string) ([]ChequeRegister, error) {
	return s.app.GetChequeRegisters(bankAccountID)
}

func (s *FinanceService) GetChequesByStatus(bankAccountID, status string) ([]OutstandingCheque, error) {
	return s.app.GetChequesByStatus(bankAccountID, status)
}

func (s *FinanceService) GetNextChequeNumber(bankAccountID string) (string, error) {
	return s.app.GetNextChequeNumber(bankAccountID)
}

func (s *FinanceService) GetOutstandingCheques(bankAccountID string) (*OutstandingChequesResult, error) {
	return s.app.GetOutstandingCheques(bankAccountID)
}

func (s *FinanceService) GetStaleCheques(bankAccountID string) ([]OutstandingCheque, error) {
	return s.app.GetStaleCheques(bankAccountID)
}

func (s *FinanceService) IssueCheque(bankAccountID string, amount float64, payeeName, payeeType string, supplierID *string, purpose string) (*OutstandingCheque, error) {
	return s.app.IssueCheque(bankAccountID, amount, payeeName, payeeType, supplierID, purpose)
}

func (s *FinanceService) MarkChequeBounced(chequeNumber, reason string) error {
	return s.app.MarkChequeBounced(chequeNumber, reason)
}

func (s *FinanceService) MarkChequeCleared(chequeNumber, bankStatementLineID string, clearedDate time.Time) error {
	return s.app.MarkChequeCleared(chequeNumber, bankStatementLineID, clearedDate)
}

func (s *FinanceService) MarkChequePresented(chequeNumber string) error {
	return s.app.MarkChequePresented(chequeNumber)
}

func (s *FinanceService) MarkChequeStale(chequeNumber string) error {
	return s.app.MarkChequeStale(chequeNumber)
}

func (s *FinanceService) ReissueCheque(oldChequeNumber, bankAccountID string) (*OutstandingCheque, error) {
	return s.app.ReissueCheque(oldChequeNumber, bankAccountID)
}

// --- credit_note_service.go ---

func (s *FinanceService) ApplyCreditNote(id string) error {
	return s.app.ApplyCreditNote(id)
}

func (s *FinanceService) CreateCreditNote(invoiceID, reason string, items []CreditNoteItemInput) (CreditNote, error) {
	return s.app.CreateCreditNote(invoiceID, reason, items)
}

func (s *FinanceService) GenerateCreditNoteNumber() (string, error) {
	return s.app.GenerateCreditNoteNumber()
}

func (s *FinanceService) GenerateCreditNotePDF(id string) (string, error) {
	return s.app.GenerateCreditNotePDF(id)
}

func (s *FinanceService) GetCreditNote(id string) (CreditNote, error) {
	return s.app.GetCreditNote(id)
}

func (s *FinanceService) IssueCreditNote(id string) (CreditNote, error) {
	return s.app.IssueCreditNote(id)
}

func (s *FinanceService) ListCreditNotes(limit, offset int) ([]CreditNote, error) {
	return s.app.ListCreditNotes(limit, offset)
}

// --- customer_invoice_service.go ---

func (s *FinanceService) BackfillInvoiceItemsFromOrders() (map[string]any, error) {
	return s.app.BackfillInvoiceItemsFromOrders()
}

func (s *FinanceService) CalculateARAgingBuckets() ([]ARAgingBucket, error) {
	return s.app.CalculateARAgingBuckets()
}

func (s *FinanceService) CreateInvoiceFromDN(deliveryNoteID string) (Invoice, error) {
	return s.app.CreateInvoiceFromDN(deliveryNoteID)
}

func (s *FinanceService) CreateInvoiceFromOrder(orderID string) (Invoice, error) {
	return s.app.CreateInvoiceFromOrder(orderID)
}

func (s *FinanceService) CreateInvoiceFromOrderWithDN(orderID string, deliveryNoteID string) (Invoice, error) {
	return s.app.CreateInvoiceFromOrderWithDN(orderID, deliveryNoteID)
}

func (s *FinanceService) CreateInvoiceWithOptions(orderID string, deliveryNoteID string, fieldVisibilityJSON string) (Invoice, error) {
	return s.app.CreateInvoiceWithOptions(orderID, deliveryNoteID, fieldVisibilityJSON)
}

func (s *FinanceService) CreateInvoiceWithCreditOverride(orderID string, deliveryNoteID string, fieldVisibilityJSON string, reason string) (Invoice, error) {
	return s.app.CreateInvoiceWithCreditOverride(orderID, deliveryNoteID, fieldVisibilityJSON, reason)
}

func (s *FinanceService) CreateProformaInvoice(orderID string) (Invoice, error) {
	return s.app.CreateProformaInvoice(orderID)
}

func (s *FinanceService) DeleteCustomerInvoice(id string) error {
	return s.app.DeleteCustomerInvoice(id)
}

func (s *FinanceService) GenerateInvoiceNumber() (string, error) {
	return s.app.GenerateInvoiceNumber()
}

func (s *FinanceService) GetARAgingByCustomer(customerID string) (ARAgingBucket, error) {
	return s.app.GetARAgingByCustomer(customerID)
}

func (s *FinanceService) GetAvailableDeliveryNotesForOrder(orderID string) ([]DeliveryNote, error) {
	return s.app.GetAvailableDeliveryNotesForOrder(orderID)
}

func (s *FinanceService) GetCustomerInvoiceByID(id string) (Invoice, error) {
	return s.app.GetCustomerInvoiceByID(id)
}

func (s *FinanceService) GetInvoiceRevenueSummary() (float64, float64, float64, int64, error) {
	return s.app.GetInvoiceRevenueSummary()
}

func (s *FinanceService) GetInvoicesByCustomer(customerID string) ([]Invoice, error) {
	return s.app.GetInvoicesByCustomer(customerID)
}

func (s *FinanceService) GetInvoicesByDateRange(startDate, endDate time.Time) ([]Invoice, error) {
	return s.app.GetInvoicesByDateRange(startDate, endDate)
}

func (s *FinanceService) GetInvoicesByStatus(status string) ([]Invoice, error) {
	return s.app.GetInvoicesByStatus(status)
}

func (s *FinanceService) GetLatePaymentInvoices() ([]Invoice, error) {
	return s.app.GetLatePaymentInvoices()
}

func (s *FinanceService) GetOverdueInvoices() ([]Invoice, error) {
	return s.app.GetOverdueInvoices()
}

func (s *FinanceService) GetUnpaidInvoices() ([]Invoice, error) {
	return s.app.GetUnpaidInvoices()
}

func (s *FinanceService) ListCustomerInvoices(limit, offset int) ([]Invoice, error) {
	return s.app.ListCustomerInvoices(limit, offset)
}

func (s *FinanceService) MarkCustomerInvoiceOverdue(id string) error {
	return s.app.MarkCustomerInvoiceOverdue(id)
}

func (s *FinanceService) MarkCustomerInvoicePaid(id string, paymentDate time.Time, paymentRef string) error {
	return s.app.MarkCustomerInvoicePaid(id, paymentDate, paymentRef)
}

func (s *FinanceService) RecordPartialPayment(id string, paymentAmount float64, paymentDate time.Time, paymentRef string) error {
	return s.app.RecordPartialPayment(id, paymentAmount, paymentDate, paymentRef)
}

func (s *FinanceService) SendCustomerInvoice(id string) error {
	return s.app.SendCustomerInvoice(id)
}

func (s *FinanceService) TrackLatePaymentHistory(customerID string) (int64, int64, float64, int, error) {
	return s.app.TrackLatePaymentHistory(customerID)
}

func (s *FinanceService) UpdateCustomerInvoice(inv Invoice) (Invoice, error) {
	return s.app.UpdateCustomerInvoice(inv)
}

// --- einvoice_service.go ---

func (s *FinanceService) ExportVATReturnData(year, quarter int) (string, error) {
	return s.app.ExportVATReturnData(year, quarter)
}

func (s *FinanceService) GenerateEInvoiceXML(invoiceID string) (string, error) {
	return s.app.GenerateEInvoiceXML(invoiceID)
}

// --- expense_service.go ---

func (s *FinanceService) ApproveExpenseEntry(entryID, notes string) (ExpenseEntry, error) {
	return s.app.ApproveExpenseEntry(entryID, notes)
}

func (s *FinanceService) CreateExpenseCategory(category ExpenseCategory) (ExpenseCategory, error) {
	return s.app.CreateExpenseCategory(category)
}

func (s *FinanceService) CreateExpenseEntry(entry ExpenseEntry) (ExpenseEntry, error) {
	return s.app.CreateExpenseEntry(entry)
}

func (s *FinanceService) CreateExpenseFromBankCandidate(bankExpenseID, categoryID string) (ExpenseEntry, error) {
	return s.app.CreateExpenseFromBankCandidate(bankExpenseID, categoryID)
}

func (s *FinanceService) CreateExpenseVendor(vendor ExpenseVendor) (ExpenseVendor, error) {
	return s.app.CreateExpenseVendor(vendor)
}

func (s *FinanceService) CreateRecurringExpense(item RecurringExpense) (RecurringExpense, error) {
	return s.app.CreateRecurringExpense(item)
}

func (s *FinanceService) DeleteExpenseCategory(categoryID string) error {
	return s.app.DeleteExpenseCategory(categoryID)
}

func (s *FinanceService) DeleteExpenseEntry(entryID string) error {
	return s.app.DeleteExpenseEntry(entryID)
}

func (s *FinanceService) DeleteExpenseVendor(vendorID string) error {
	return s.app.DeleteExpenseVendor(vendorID)
}

func (s *FinanceService) DeleteRecurringExpense(recurringID string) error {
	return s.app.DeleteRecurringExpense(recurringID)
}

func (s *FinanceService) EnsureExpenseFoundation() error {
	return s.app.EnsureExpenseFoundation()
}

func (s *FinanceService) GenerateRecurringExpenses(cutoffISO string) ([]ExpenseEntry, error) {
	return s.app.GenerateRecurringExpenses(cutoffISO)
}

func (s *FinanceService) ListBankExpenseCandidates(includeLinked bool) ([]BankExpenseEntry, error) {
	return s.app.ListBankExpenseCandidates(includeLinked)
}

func (s *FinanceService) ListExpenseCategories(activeOnly bool) ([]ExpenseCategory, error) {
	return s.app.ListExpenseCategories(activeOnly)
}

func (s *FinanceService) ListExpenseDashboardSummary() (ExpenseDashboardSummary, error) {
	return s.app.ListExpenseDashboardSummary()
}

func (s *FinanceService) ListExpenseEntries(status string, includePaid bool) ([]ExpenseEntry, error) {
	return s.app.ListExpenseEntries(status, includePaid)
}

func (s *FinanceService) ListExpenseVendors(activeOnly bool) ([]ExpenseVendor, error) {
	return s.app.ListExpenseVendors(activeOnly)
}

func (s *FinanceService) ListRecurringExpenses(activeOnly bool) ([]RecurringExpense, error) {
	return s.app.ListRecurringExpenses(activeOnly)
}

func (s *FinanceService) MarkExpenseEntryPaid(entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod string) (ExpenseEntry, error) {
	return s.app.MarkExpenseEntryPaid(entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod)
}

func (s *FinanceService) PostExpenseEntry(entryID string) (ExpenseEntry, error) {
	return s.app.PostExpenseEntry(entryID)
}

func (s *FinanceService) RejectExpenseEntry(entryID, reason string) (ExpenseEntry, error) {
	return s.app.RejectExpenseEntry(entryID, reason)
}

func (s *FinanceService) SubmitExpenseEntry(entryID string) (ExpenseEntry, error) {
	return s.app.SubmitExpenseEntry(entryID)
}

// --- finance_reporting_service.go ---

func (s *FinanceService) ClosePeriod(periodID string) error {
	return s.app.ClosePeriod(periodID)
}

func (s *FinanceService) GetCashFlowProjection(days int) (CashFlowProjection, error) {
	return s.app.GetCashFlowProjection(days)
}

func (s *FinanceService) GetCashflowEvidenceCommandCenter(days int) (cashflowevidence.CommandCenter, error) {
	return s.app.GetCashflowEvidenceCommandCenter(days)
}

func (s *FinanceService) ExportCashflowEvidencePack(days int) (string, error) {
	return s.app.ExportCashflowEvidencePack(days)
}

func (s *FinanceService) ListCashflowEvidenceProposalReviews(days int, includeResolved bool) ([]CashflowEvidenceProposalReview, error) {
	return s.app.ListCashflowEvidenceProposalReviews(days, includeResolved)
}

func (s *FinanceService) ReviewCashflowEvidenceProposal(proposalReviewID string, status string, note string) (CashflowEvidenceProposalReview, error) {
	return s.app.ReviewCashflowEvidenceProposal(proposalReviewID, status, note)
}

func (s *FinanceService) SyncCashflowEvidenceProposalReviews(days int) ([]CashflowEvidenceProposalReview, error) {
	return s.app.SyncCashflowEvidenceProposalReviews(days)
}

func (s *FinanceService) GetMarginAnalysisByCustomer() ([]MarginAnalysisByCustomer, error) {
	return s.app.GetMarginAnalysisByCustomer()
}

func (s *FinanceService) GetMarginAnalysisByProduct() ([]MarginAnalysisByProduct, error) {
	return s.app.GetMarginAnalysisByProduct()
}

func (s *FinanceService) GetPaymentAgingReport() (PaymentAgingReport, error) {
	return s.app.GetPaymentAgingReport()
}

func (s *FinanceService) GetVATReconciliation(startDateStr, endDateStr string) (VATReconciliation, error) {
	return s.app.GetVATReconciliation(startDateStr, endDateStr)
}

func (s *FinanceService) IsPeriodClosed(dateStr string) (bool, error) {
	return s.app.IsPeriodClosed(dateStr)
}

// --- financial_year_service.go ---

func (s *FinanceService) GetAvailableFinancialYears() ([]int, error) {
	return s.app.GetAvailableFinancialYears()
}

func (s *FinanceService) GetDynamicFinancialDashboard(year int) (FinancialDashboard, error) {
	return s.app.GetDynamicFinancialDashboard(year)
}

func (s *FinanceService) GetFinancialDashboardByDivision(year int, division string) (DivisionFinancialSummary, error) {
	return s.app.GetFinancialDashboardByDivision(year, division)
}

func (s *FinanceService) GetFinancialYearData(year int) (*FinancialYearSummary, error) {
	return s.app.GetFinancialYearData(year)
}

// --- fx_revaluation_service.go ---

func (s *FinanceService) CalculateFXRevaluation(bankAccountID string, revaluationDate time.Time) (*FXRevaluation, error) {
	return s.app.CalculateFXRevaluation(bankAccountID, revaluationDate)
}

func (s *FinanceService) CreateFXRate(fromCurrency, toCurrency string, rate float64, rateDate time.Time, source string) (*FXRate, error) {
	return s.app.CreateFXRate(fromCurrency, toCurrency, rate, rateDate, source)
}

func (s *FinanceService) DeleteFXRate(rateID string) error {
	return s.app.DeleteFXRate(rateID)
}

func (s *FinanceService) GetAllFXRates(baseCurrency string, date time.Time) ([]FXRate, error) {
	return s.app.GetAllFXRates(baseCurrency, date)
}

func (s *FinanceService) GetFXExposureReport() (*FXExposureResult, error) {
	return s.app.GetFXExposureReport()
}

func (s *FinanceService) GetFXGainLossSummary(year int) (map[string]any, error) {
	return s.app.GetFXGainLossSummary(year)
}

func (s *FinanceService) GetFXRate(fromCurrency, toCurrency string, date time.Time) (*FXRate, error) {
	return s.app.GetFXRate(fromCurrency, toCurrency, date)
}

func (s *FinanceService) GetFXRateHistory(fromCurrency, toCurrency string, startDate, endDate time.Time) ([]FXRate, error) {
	return s.app.GetFXRateHistory(fromCurrency, toCurrency, startDate, endDate)
}

func (s *FinanceService) GetFXRevaluations(bankAccountID string) ([]FXRevaluation, error) {
	return s.app.GetFXRevaluations(bankAccountID)
}

func (s *FinanceService) GetLatestFXRate(fromCurrency, toCurrency string) (*FXRate, error) {
	return s.app.GetLatestFXRate(fromCurrency, toCurrency)
}

func (s *FinanceService) GetUnpostedRevaluations() (*FXRevaluationBatchResult, error) {
	return s.app.GetUnpostedRevaluations()
}

func (s *FinanceService) PostFXRevaluation(revaluationID, user string) error {
	return s.app.PostFXRevaluation(revaluationID, user)
}

func (s *FinanceService) RevalueAllForeignAccounts(revaluationDate time.Time) (*FXRevaluationBatchResult, error) {
	return s.app.RevalueAllForeignAccounts(revaluationDate)
}

func (s *FinanceService) ReverseRevaluation(revaluationID, user, reason string) error {
	return s.app.ReverseRevaluation(revaluationID, user, reason)
}

// --- invoice_list_vm_endpoint.go ---

func (s *FinanceService) GetInvoiceListVM(page, pageSize int) (financevm.InvoiceListVM, error) {
	return s.app.GetInvoiceListVM(page, pageSize)
}

// --- invoice_traceability.go ---

func (s *FinanceService) CalculateInvoiceMargin(invoiceID string) error {
	return s.app.CalculateInvoiceMargin(invoiceID)
}

func (s *FinanceService) GetInvoiceAuditTrail(invoiceID string) (InvoiceAuditTrail, error) {
	return s.app.GetInvoiceAuditTrail(invoiceID)
}

func (s *FinanceService) GetInvoicesByOrder(orderID string) ([]Invoice, error) {
	return s.app.GetInvoicesByOrder(orderID)
}

func (s *FinanceService) GetInvoicesByRFQ(rfqID string) ([]Invoice, error) {
	return s.app.GetInvoicesByRFQ(rfqID)
}

func (s *FinanceService) LinkInvoiceToOrder(invoiceID string, orderID string) error {
	return s.app.LinkInvoiceToOrder(invoiceID, orderID)
}

func (s *FinanceService) LinkInvoiceToRFQ(invoiceID string, rfqID string) error {
	return s.app.LinkInvoiceToRFQ(invoiceID, rfqID)
}

func (s *FinanceService) RecalculateAllInvoiceMargins() (int, error) {
	return s.app.RecalculateAllInvoiceMargins()
}

// --- payment_service.go ---

func (s *FinanceService) DeletePayment(id string) error {
	return s.app.DeletePayment(id)
}

func (s *FinanceService) GetAllPayments(limit, offset int) ([]Payment, error) {
	return s.app.GetAllPayments(limit, offset)
}

func (s *FinanceService) GetPayment(id string) (Payment, error) {
	return s.app.GetPayment(id)
}

func (s *FinanceService) GetPaymentsByInvoice(invoiceID string) ([]Payment, error) {
	return s.app.GetPaymentsByInvoice(invoiceID)
}

func (s *FinanceService) ProgressOrderOnDelivery(orderID string) error {
	return s.app.ProgressOrderOnDelivery(orderID)
}

func (s *FinanceService) ProgressOrderOnInvoice(orderID string) error {
	return s.app.ProgressOrderOnInvoice(orderID)
}

func (s *FinanceService) RecordPayment(invoiceID string, amount float64, method string, dateStr string, reference string) (*Payment, error) {
	return s.app.RecordPayment(invoiceID, amount, method, dateStr, reference)
}

func (s *FinanceService) UpdatePayment(id string, payment Payment) (*Payment, error) {
	return s.app.UpdatePayment(id, payment)
}

// --- payroll_service.go ---

func (s *FinanceService) ApprovePayrollRun(runID, notes string) (PayrollRun, error) {
	return s.app.ApprovePayrollRun(runID, notes)
}

func (s *FinanceService) CreatePayrollPeriod(period PayrollPeriod) (PayrollPeriod, error) {
	return s.app.CreatePayrollPeriod(period)
}

func (s *FinanceService) EnsurePayrollFoundation() error {
	return s.app.EnsurePayrollFoundation()
}

func (s *FinanceService) GeneratePayrollRun(payrollPeriodID string) (PayrollRun, error) {
	return s.app.GeneratePayrollRun(payrollPeriodID)
}

func (s *FinanceService) GetPayrollRun(runID string) (PayrollRun, error) {
	return s.app.GetPayrollRun(runID)
}

func (s *FinanceService) ListEmployeeCompensationProfiles(activeOnly bool) ([]EmployeeCompensationProfile, error) {
	return s.app.ListEmployeeCompensationProfiles(activeOnly)
}

func (s *FinanceService) ListPayrollDashboardSummary() (PayrollDashboardSummary, error) {
	return s.app.ListPayrollDashboardSummary()
}

func (s *FinanceService) ListPayrollPayouts(payrollRunID string) ([]PayrollPayout, error) {
	return s.app.ListPayrollPayouts(payrollRunID)
}

func (s *FinanceService) ListPayrollPeriods(includeClosed bool) ([]PayrollPeriod, error) {
	return s.app.ListPayrollPeriods(includeClosed)
}

func (s *FinanceService) ListPayrollRuns(payrollPeriodID string) ([]PayrollRun, error) {
	return s.app.ListPayrollRuns(payrollPeriodID)
}

func (s *FinanceService) ListUnreconciledPayrollPayouts() ([]PayrollPayout, error) {
	return s.app.ListUnreconciledPayrollPayouts()
}

func (s *FinanceService) MarkPayrollRunPaid(runID, paidAtISO, paymentReference, bankAccountID string) (PayrollRun, error) {
	return s.app.MarkPayrollRunPaid(runID, paidAtISO, paymentReference, bankAccountID)
}

func (s *FinanceService) PostPayrollRun(runID string) (PayrollRun, error) {
	return s.app.PostPayrollRun(runID)
}

func (s *FinanceService) UpsertEmployeeCompensationProfile(profile EmployeeCompensationProfile) (EmployeeCompensationProfile, error) {
	return s.app.UpsertEmployeeCompensationProfile(profile)
}

// --- statement_export_service.go ---

func (s *FinanceService) ExportBalanceSheetCSV(year int) (string, error) {
	return s.app.ExportBalanceSheetCSV(year)
}

func (s *FinanceService) ExportGeneralLedgerCSV(year int) (string, error) {
	return s.app.ExportGeneralLedgerCSV(year)
}

func (s *FinanceService) ExportJournalCSV(year int) (string, error) {
	return s.app.ExportJournalCSV(year)
}

// --- supplier_invoice_service.go ---

func (s *FinanceService) ApproveSupplierInvoice(id string, approvedBy string) error {
	return s.app.ApproveSupplierInvoice(id, approvedBy)
}

func (s *FinanceService) CalculateSupplierLeadTime(supplierID string) (SupplierLeadTimeMetrics, error) {
	return s.app.CalculateSupplierLeadTime(supplierID)
}

func (s *FinanceService) CreateSupplierInvoice(inv SupplierInvoice) (SupplierInvoice, error) {
	return s.app.CreateSupplierInvoice(inv)
}

func (s *FinanceService) CreateSupplierInvoiceFromOCR(ocrDocID string, supplierID string, poID string) (SupplierInvoice, error) {
	return s.app.CreateSupplierInvoiceFromOCR(ocrDocID, supplierID, poID)
}

func (s *FinanceService) DeleteSupplierInvoice(id string) error {
	return s.app.DeleteSupplierInvoice(id)
}

func (s *FinanceService) DisputeSupplierInvoice(id string, reason string) error {
	return s.app.DisputeSupplierInvoice(id, reason)
}

func (s *FinanceService) GetLateSuppliers() ([]SupplierLeadTimeMetrics, error) {
	return s.app.GetLateSuppliers()
}

func (s *FinanceService) GetOverdueSupplierInvoices() ([]SupplierInvoice, error) {
	return s.app.GetOverdueSupplierInvoices()
}

func (s *FinanceService) GetSupplierInvoiceByID(id string) (SupplierInvoice, error) {
	return s.app.GetSupplierInvoiceByID(id)
}

func (s *FinanceService) GetSupplierInvoices() ([]SupplierInvoice, error) {
	return s.app.GetSupplierInvoices()
}

func (s *FinanceService) GetSupplierInvoicesByPO(poID string) ([]SupplierInvoice, error) {
	return s.app.GetSupplierInvoicesByPO(poID)
}

func (s *FinanceService) GetSupplierInvoicesBySupplier(supplierID string) ([]SupplierInvoice, error) {
	return s.app.GetSupplierInvoicesBySupplier(supplierID)
}

func (s *FinanceService) GetSupplierLeadTimeReport() ([]SupplierLeadTimeMetrics, error) {
	return s.app.GetSupplierLeadTimeReport()
}

func (s *FinanceService) GetUnpaidSupplierInvoices() ([]SupplierInvoice, error) {
	return s.app.GetUnpaidSupplierInvoices()
}

func (s *FinanceService) MarkSupplierInvoicePaid(id string, paymentRef string, paymentMethod string) error {
	return s.app.MarkSupplierInvoicePaid(id, paymentRef, paymentMethod)
}

func (s *FinanceService) PerformThreeWayMatch(invoiceID string) (ThreeWayMatchResult, error) {
	return s.app.PerformThreeWayMatch(invoiceID)
}

func (s *FinanceService) UpdateSupplierInvoice(inv SupplierInvoice) (SupplierInvoice, error) {
	return s.app.UpdateSupplierInvoice(inv)
}

// --- supplier_payment_service.go ---

func (s *FinanceService) DeleteSupplierPayment(id string) error {
	return s.app.DeleteSupplierPayment(id)
}

func (s *FinanceService) GetAllSupplierPayments() ([]SupplierPayment, error) {
	return s.app.GetAllSupplierPayments()
}

func (s *FinanceService) GetSupplierPayment(id string) (SupplierPayment, error) {
	return s.app.GetSupplierPayment(id)
}

func (s *FinanceService) GetSupplierPaymentsByInvoice(invoiceID string) ([]SupplierPayment, error) {
	return s.app.GetSupplierPaymentsByInvoice(invoiceID)
}

func (s *FinanceService) GetSupplierPaymentsSummary() (map[string]any, error) {
	return s.app.GetSupplierPaymentsSummary()
}

func (s *FinanceService) RecordSupplierPayment(invoiceID string, amount float64, currency, method, date, reference string, exchangeRate float64) (*SupplierPayment, error) {
	return s.app.RecordSupplierPayment(invoiceID, amount, currency, method, date, reference, exchangeRate)
}

func (s *FinanceService) UpdateSupplierPayment(id string, payment SupplierPayment) (*SupplierPayment, error) {
	return s.app.UpdateSupplierPayment(id, payment)
}

// --- tally_importer.go ---

func (s *FinanceService) GenerateBalanceSheet(year int) (*TallyBalanceSheet, error) {
	return s.app.GenerateBalanceSheet(year)
}

func (s *FinanceService) GenerateProfitAndLoss(year int) (*TallyPLReport, error) {
	return s.app.GenerateProfitAndLoss(year)
}

func (s *FinanceService) GetFinancialReportYears() ([]int, error) {
	return s.app.GetFinancialReportYears()
}

func (s *FinanceService) ImportARDefaulters() (*TallyImportResult, error) {
	return s.app.ImportARDefaulters()
}

func (s *FinanceService) ImportAllTallyData() (*TallyImportResult, error) {
	return s.app.ImportAllTallyData()
}

func (s *FinanceService) ImportSupplierPaymentsFromFile() (*TallyImportResult, error) {
	return s.app.ImportSupplierPaymentsFromFile()
}

func (s *FinanceService) ImportTallyInvoices(year int) (*TallyImportResult, error) {
	return s.app.ImportTallyInvoices(year)
}

func (s *FinanceService) ImportTallyPurchases(year int) (*TallyImportResult, error) {
	return s.app.ImportTallyPurchases(year)
}
