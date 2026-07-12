// Package finance defines the finance domain ports.
package finance

import "time"

type InvoiceRepository interface {
	CreateInvoice(invoice Invoice) (Invoice, error)
	GetInvoiceByID(id string) (Invoice, error)
	ListInvoices(limit, offset int) ([]Invoice, error)
	UpdateInvoice(invoice Invoice) (Invoice, error)
	DeleteInvoice(id string) error
	ListInvoiceItems(invoiceID string) ([]DBInvoiceItem, error)
}

type PaymentRepository interface {
	CreatePayment(payment Payment) (Payment, error)
	GetPayment(id string) (Payment, error)
	ListPayments(limit, offset int) ([]Payment, error)
	ListPaymentsByInvoice(invoiceID string) ([]Payment, error)
	UpdatePayment(id string, payment Payment) (*Payment, error)
	DeletePayment(id string) error
}

type BankingRepository interface {
	CreateBankStatement(statement *BankStatement) error
	GetBankStatementByID(id string) (*BankStatement, error)
	ListBankStatements(bankAccountID string) ([]BankStatement, error)
	ListBankStatementLines(statementID string) ([]BankStatementLine, error)
	CreateBankStatementLine(statementID string, line map[string]any) (*BankStatementLine, error)
	UpdateBankStatementLine(lineID string, updates map[string]any) error
	DeleteBankStatementLine(lineID string) error
}

type InvoiceService interface {
	CreateInvoiceFromOrder(orderID string) (Invoice, error)
	CreateInvoiceFromDN(deliveryNoteID string) (Invoice, error)
	CreateInvoiceWithOptions(orderID, deliveryNoteID, fieldVisibilityJSON string) (Invoice, error)
	SendCustomerInvoice(id string) error
	MarkCustomerInvoicePaid(id string, paymentDate time.Time, paymentRef string) error
	RecordPartialPayment(id string, paymentAmount float64, paymentDate time.Time, paymentRef string) error
	CreateCreditNote(invoiceID, reason string, items []CreditNoteItem) (CreditNote, error)
}

type PaymentService interface {
	RecordPayment(invoiceID string, amount float64, method, dateStr, reference string) (*Payment, error)
	GetPaymentsByInvoice(invoiceID string) ([]Payment, error)
	GetAllPayments(limit, offset int) ([]Payment, error)
	GetPayment(id string) (Payment, error)
	UpdatePayment(id string, payment Payment) (*Payment, error)
	DeletePayment(id string) error
	RecordSupplierPayment(invoiceID string, amount float64, currency, method, date, reference string, exchangeRate float64) (*SupplierPayment, error)
}

type BankingService interface {
	GetCashPosition() (map[string]any, error)
	GetCashPositionByAccount(bankAccountID string) (float64, error)
	ValidateStatementBalance(statementID string) (*StatementBalanceValidation, error)
	FinalizeReconciliation(statementID, reconciledBy string) error
	ReopenReconciliation(statementID, user, reason string) error
	GetReconciliationSummary(statementID string) (map[string]any, error)
	GetReconciliationStats(bankAccountID string) (map[string]any, error)
	CreateBookBankReconciliation(bankAccountID string, reconciliationDate time.Time, bankStatementBalance, bookBalance, depositsInTransit, outstandingCheques float64) (*BookBankReconciliation, error)
}

type ExpenseService interface {
	ListExpenseCategories(activeOnly bool) ([]ExpenseCategory, error)
	CreateExpenseEntry(entry ExpenseEntry) (ExpenseEntry, error)
	DeleteExpenseEntry(entryID string) error
	ListExpenseEntries(status string, includePaid bool) ([]ExpenseEntry, error)
	ListExpenseDashboardSummary() (ExpenseDashboardSummary, error)
	SubmitExpenseEntry(entryID string) (ExpenseEntry, error)
	ApproveExpenseEntry(entryID, notes string) (ExpenseEntry, error)
	RejectExpenseEntry(entryID, reason string) (ExpenseEntry, error)
	MarkExpenseEntryPaid(entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod string) (ExpenseEntry, error)
}
