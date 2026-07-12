package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUpdateBankStatementLineRecalculatesAndClearsMatch(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&BankStatement{}, &BankStatementLine{}, &BankLinePaymentAllocation{}, &PayrollRun{}, &PayrollPayout{}, &BankReconciliationAuditLog{}))

	statementID := uuid.New().String()
	lineID := uuid.New().String()

	statement := BankStatement{
		Base:            Base{ID: statementID},
		BankAccountID:   uuid.New().String(),
		StatementNumber: "OCR-TEST-001",
		StatementDate:   time.Now(),
		PeriodStart:     time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:       time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC),
		OpeningBalance:  100,
		ClosingBalance:  120,
		Status:          "Reconciled",
		ReconciledBy:    "admin",
		TotalCredits:    20,
		CreditCount:     1,
		BalanceVerified: true,
	}
	require.NoError(t, app.db.Create(&statement).Error)

	line := BankStatementLine{
		Base:              Base{ID: lineID},
		BankStatementID:   statementID,
		LineNumber:        1,
		TransactionDate:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
		ValueDate:         time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
		Description:       "OCR imported receipt",
		Credit:            20,
		Balance:           120,
		IsMatched:         true,
		MatchType:         "Manual",
		MatchConfidence:   1,
		MatchedInvoiceIDs: "invoice-1",
	}
	require.NoError(t, app.db.Create(&line).Error)

	// PC-D4 (deliberate golden change): a Reconciled statement REFUSES line
	// edits instead of silently auto-reverting to InProgress. Reopening is an
	// explicit, audited act — only then does the edit proceed.
	err := app.UpdateBankStatementLine(lineID, map[string]any{
		"debit":  20.0,
		"credit": 0.0,
	})
	require.Error(t, err)

	var untouched BankStatementLine
	require.NoError(t, app.db.First(&untouched, "id = ?", lineID).Error)
	require.Equal(t, 20.0, untouched.Credit, "refused edit must not change the line")
	require.True(t, untouched.IsMatched)

	require.NoError(t, app.ReopenReconciliation(statementID, "admin", "correcting an OCR mis-read"))

	err = app.UpdateBankStatementLine(lineID, map[string]any{
		"debit":  20.0,
		"credit": 0.0,
	})
	require.NoError(t, err)

	var updatedLine BankStatementLine
	require.NoError(t, app.db.First(&updatedLine, "id = ?", lineID).Error)
	require.Equal(t, 20.0, updatedLine.Debit)
	require.Equal(t, 0.0, updatedLine.Credit)
	require.False(t, updatedLine.IsMatched)
	require.Equal(t, "Unmatched", updatedLine.MatchType)
	require.Zero(t, updatedLine.MatchConfidence)
	require.Empty(t, updatedLine.MatchedInvoiceIDs)

	var updatedStatement BankStatement
	require.NoError(t, app.db.First(&updatedStatement, "id = ?", statementID).Error)
	require.Equal(t, 20.0, updatedStatement.TotalDebits)
	require.Equal(t, 0.0, updatedStatement.TotalCredits)
	require.Equal(t, 1, updatedStatement.DebitCount)
	require.Equal(t, 0, updatedStatement.CreditCount)
	require.False(t, updatedStatement.BalanceVerified)
	require.Equal(t, 40.0, updatedStatement.DiscrepancyAmount)
	require.Equal(t, "InProgress", updatedStatement.Status)
	require.Nil(t, updatedStatement.ReconciledAt)
	require.Empty(t, updatedStatement.ReconciledBy)
}

func TestCreateSplitAllocationMatchesBulkReceiptToMultipleCustomerInvoices(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&BankStatement{}, &BankStatementLine{}, &BankLinePaymentAllocation{}, &Invoice{}, &BankReconciliationAuditLog{}))

	statement := BankStatement{
		Base:            Base{ID: uuid.New().String()},
		BankAccountID:   uuid.New().String(),
		StatementNumber: "BULK-RECEIPT-001",
		StatementDate:   time.Now(),
		PeriodStart:     time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:       time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC),
		Division:        "Acme Instrumentation",
		Status:          "Imported",
	}
	require.NoError(t, app.db.Create(&statement).Error)

	line := BankStatementLine{
		Base:            Base{ID: uuid.New().String()},
		BankStatementID: statement.ID,
		LineNumber:      1,
		TransactionDate: time.Now(),
		ValueDate:       time.Now(),
		Description:     "Bulk customer receipt",
		Credit:          150,
		Balance:         150,
	}
	require.NoError(t, app.db.Create(&line).Error)

	invoiceA := seedBankReconInvoice(t, app, "INV-2026-0101", 100)
	invoiceB := seedBankReconInvoice(t, app, "INV-2026-0102", 50)

	require.NoError(t, app.CreateSplitAllocation(line.ID, []AllocationInput{
		{AllocationType: "CUSTOMER_INVOICE", EntityID: invoiceA.ID, AllocatedAmount: 100},
		{AllocationType: "CUSTOMER_INVOICE", EntityID: invoiceB.ID, AllocatedAmount: 50},
	}, "admin"))

	var updatedLine BankStatementLine
	require.NoError(t, app.db.First(&updatedLine, "id = ?", line.ID).Error)
	require.True(t, updatedLine.IsMatched)
	require.Equal(t, "Split", updatedLine.MatchType)
	require.ElementsMatch(t, []string{invoiceA.ID, invoiceB.ID}, parseJSONArray(updatedLine.MatchedInvoiceIDs))

	var allocations []BankLinePaymentAllocation
	require.NoError(t, app.db.Where("bank_statement_line_id = ?", line.ID).Find(&allocations).Error)
	require.Len(t, allocations, 2)
}

func TestCreateSplitAllocationAllowsOneInvoiceAcrossMultipleReceipts(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&BankStatement{}, &BankStatementLine{}, &BankLinePaymentAllocation{}, &Invoice{}, &BankReconciliationAuditLog{}))

	statement := BankStatement{
		Base:            Base{ID: uuid.New().String()},
		BankAccountID:   uuid.New().String(),
		StatementNumber: "PARTIAL-RECEIPT-001",
		StatementDate:   time.Now(),
		PeriodStart:     time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:       time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
		Division:        "Acme Instrumentation",
		Status:          "Imported",
	}
	require.NoError(t, app.db.Create(&statement).Error)

	invoice := seedBankReconInvoice(t, app, "INV-2026-0201", 150)
	lineA := seedBankReconCreditLine(t, app, statement.ID, 1, 100)
	lineB := seedBankReconCreditLine(t, app, statement.ID, 2, 50)

	require.NoError(t, app.CreateSplitAllocation(lineA.ID, []AllocationInput{
		{AllocationType: "CUSTOMER_INVOICE", EntityID: invoice.ID, AllocatedAmount: 100},
	}, "admin"))
	require.NoError(t, app.CreateSplitAllocation(lineB.ID, []AllocationInput{
		{AllocationType: "CUSTOMER_INVOICE", EntityID: invoice.ID, AllocatedAmount: 50},
	}, "admin"))

	var totalAllocated float64
	require.NoError(t, app.db.Model(&BankLinePaymentAllocation{}).
		Where("customer_invoice_id = ?", invoice.ID).
		Select("COALESCE(SUM(allocated_amount), 0)").
		Scan(&totalAllocated).Error)
	require.Equal(t, 150.0, totalAllocated)

	lineC := seedBankReconCreditLine(t, app, statement.ID, 3, 1)
	err := app.CreateSplitAllocation(lineC.ID, []AllocationInput{
		{AllocationType: "CUSTOMER_INVOICE", EntityID: invoice.ID, AllocatedAmount: 1},
	}, "admin")
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds remaining invoice balance")
}

func TestUnmatchLineClearsSplitAllocations(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&BankStatement{}, &BankStatementLine{}, &BankLinePaymentAllocation{}, &Invoice{}, &BankReconciliationAuditLog{}))

	statement := BankStatement{
		Base:            Base{ID: uuid.New().String()},
		BankAccountID:   uuid.New().String(),
		StatementNumber: "UNMATCH-SPLIT-001",
		StatementDate:   time.Now(),
		PeriodStart:     time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:       time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC),
		Division:        "Acme Instrumentation",
		Status:          "Imported",
	}
	require.NoError(t, app.db.Create(&statement).Error)

	invoice := seedBankReconInvoice(t, app, "INV-2026-0301", 75)
	line := seedBankReconCreditLine(t, app, statement.ID, 1, 75)
	require.NoError(t, app.CreateSplitAllocation(line.ID, []AllocationInput{
		{AllocationType: "CUSTOMER_INVOICE", EntityID: invoice.ID, AllocatedAmount: 75},
	}, "admin"))
	require.NoError(t, app.UnmatchLine(line.ID, "admin", "test unmatch"))

	var updatedLine BankStatementLine
	require.NoError(t, app.db.First(&updatedLine, "id = ?", line.ID).Error)
	require.False(t, updatedLine.IsMatched)
	require.Equal(t, "Unmatched", updatedLine.MatchType)

	var allocationCount int64
	require.NoError(t, app.db.Model(&BankLinePaymentAllocation{}).Where("bank_statement_line_id = ?", line.ID).Count(&allocationCount).Error)
	require.Zero(t, allocationCount)
}

func TestCreateSplitAllocationMatchesSupplierInvoiceAndExpense(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&BankStatement{}, &BankStatementLine{}, &BankLinePaymentAllocation{}, &SupplierInvoice{}, &ExpenseEntry{}, &BankReconciliationAuditLog{}))

	statement := BankStatement{
		Base:            Base{ID: uuid.New().String()},
		BankAccountID:   uuid.New().String(),
		StatementNumber: "PAYMENT-SPLIT-001",
		StatementDate:   time.Now(),
		PeriodStart:     time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:       time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC),
		Division:        "Acme Instrumentation",
		Status:          "Imported",
	}
	require.NoError(t, app.db.Create(&statement).Error)

	line := BankStatementLine{
		Base:            Base{ID: uuid.New().String()},
		BankStatementID: statement.ID,
		LineNumber:      1,
		TransactionDate: time.Now(),
		ValueDate:       time.Now(),
		Description:     "Supplier invoice and office expense payment",
		Debit:           175,
		Balance:         -175,
	}
	require.NoError(t, app.db.Create(&line).Error)

	supplierInvoice := SupplierInvoice{
		Base:          Base{ID: uuid.New().String()},
		SupplierID:    uuid.New().String(),
		SupplierName:  "Test Supplier",
		InvoiceNumber: "SI-001",
		InvoiceDate:   time.Now(),
		DueDate:       time.Now().AddDate(0, 0, 30),
		Currency:      "BHD",
		ExchangeRate:  1,
		TotalBHD:      125,
		Status:        "Approved",
		PaymentStatus: "Unpaid",
		MatchStatus:   "Matched",
		Division:      "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&supplierInvoice).Error)

	expense := ExpenseEntry{
		Base:          Base{ID: uuid.New().String()},
		EntryNumber:   "EXP-001",
		Division:      "Acme Instrumentation",
		ExpenseDate:   time.Now(),
		Description:   "Office expense",
		CategoryID:    uuid.New().String(),
		Currency:      "BHD",
		Amount:        50,
		TotalAmount:   50,
		Status:        "approved",
		PaymentStatus: "unpaid",
	}
	require.NoError(t, app.db.Create(&expense).Error)

	require.NoError(t, app.CreateSplitAllocation(line.ID, []AllocationInput{
		{AllocationType: "SUPPLIER_INVOICE", EntityID: supplierInvoice.ID, AllocatedAmount: 125},
		{AllocationType: "EXPENSE", EntityID: expense.ID, AllocatedAmount: 50},
	}, "admin"))

	var allocations []BankLinePaymentAllocation
	require.NoError(t, app.db.Where("bank_statement_line_id = ?", line.ID).Find(&allocations).Error)
	require.Len(t, allocations, 2)

	var expenseAllocation BankLinePaymentAllocation
	require.NoError(t, app.db.Where("expense_entry_id = ?", expense.ID).First(&expenseAllocation).Error)
	require.Equal(t, 50.0, expenseAllocation.AllocatedAmount)

	overpayLine := BankStatementLine{
		Base:            Base{ID: uuid.New().String()},
		BankStatementID: statement.ID,
		LineNumber:      2,
		TransactionDate: time.Now(),
		ValueDate:       time.Now(),
		Description:     "Duplicate expense payment",
		Debit:           1,
		Balance:         -176,
	}
	require.NoError(t, app.db.Create(&overpayLine).Error)

	err := app.CreateSplitAllocation(overpayLine.ID, []AllocationInput{
		{AllocationType: "EXPENSE", EntityID: expense.ID, AllocatedAmount: 1},
	}, "admin")
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds remaining expense balance")
}

func seedBankReconInvoice(t *testing.T, app *App, number string, outstanding float64) Invoice {
	t.Helper()
	invoice := Invoice{
		Base:           Base{ID: uuid.New().String()},
		InvoiceNumber:  number,
		InvoiceDate:    time.Now(),
		CustomerID:     uuid.New().String(),
		CustomerName:   "Bulk Customer",
		GrandTotalBHD:  outstanding,
		OutstandingBHD: outstanding,
		Status:         "Sent",
		Division:       "Acme Instrumentation",
		DueDate:        time.Now().AddDate(0, 0, 30),
	}
	require.NoError(t, app.db.Create(&invoice).Error)
	return invoice
}

func seedBankReconCreditLine(t *testing.T, app *App, statementID string, lineNumber int, amount float64) BankStatementLine {
	t.Helper()
	line := BankStatementLine{
		Base:            Base{ID: uuid.New().String()},
		BankStatementID: statementID,
		LineNumber:      lineNumber,
		TransactionDate: time.Now(),
		ValueDate:       time.Now(),
		Description:     "Customer receipt",
		Credit:          amount,
		Balance:         amount,
	}
	require.NoError(t, app.db.Create(&line).Error)
	return line
}
