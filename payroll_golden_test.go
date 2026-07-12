package main

// Payroll golden tests (Wave 5 A.2). Written and committed green against
// the UNTOUCHED root payroll_service.go BEFORE the peel to pkg — the
// accrual/posting journal is sacred financial arithmetic (invariant 5),
// so the numbers are pinned first and the peel must reproduce them
// exactly. All fixture amounts are integers (exact in float64), so every
// assertion is exact equality.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setupPayrollApp(t *testing.T) *App {
	t.Helper()
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(
		&Employee{},
		&JournalEntry{},
		&JournalLine{},
		&ExpenseCategory{},
		&ExpenseEntry{},
	))
	require.NoError(t, app.EnsurePayrollFoundation())
	return app
}

func seedPayrollEmployee(t *testing.T, app *App, id, name string, profile EmployeeCompensationProfile) {
	t.Helper()
	employee := Employee{
		Base:             Base{ID: id},
		EmployeeCode:     "EMP-" + id,
		FullName:         name,
		JobTitle:         "Technician",
		EmploymentStatus: "active",
		IsActive:         true,
	}
	require.NoError(t, app.db.Create(&employee).Error)
	profile.EmployeeID = id
	profile.IsActive = true
	profile.Division = "Acme Instrumentation"
	profile.Currency = "BHD"
	require.NoError(t, app.db.Create(&profile).Error)
}

func seedPayrollPeriod(t *testing.T, app *App) PayrollPeriod {
	t.Helper()
	payment := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	period := PayrollPeriod{
		Name:        "Jun 2026 Payroll - Golden",
		Division:    "Acme Instrumentation",
		PeriodStart: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		PaymentDate: &payment,
		Status:      "open",
	}
	require.NoError(t, app.db.Create(&period).Error)
	return period
}

func accountBalance(t *testing.T, app *App, code string) float64 {
	t.Helper()
	var account ChartOfAccount
	require.NoError(t, app.db.Where("account_code = ?", code).First(&account).Error)
	return account.Balance
}

// TestPayrollRunGeneration_GoldenTotals pins the run-generation arithmetic:
// allowance folding, deduction folding, gross/net per item, run totals, the
// per-item component decomposition, and the scheduled payout amounts.
func TestPayrollRunGeneration_GoldenTotals(t *testing.T) {
	app := setupPayrollApp(t)
	period := seedPayrollPeriod(t, app)

	seedPayrollEmployee(t, app, "emp-1", "Aisha Rahman", EmployeeCompensationProfile{
		BaseSalary:         1000,
		HousingAllowance:   128,
		TransportAllowance: 64,
		StandardDeduction:  32,
		TaxDeduction:       16,
		EmployerCost:       96,
	})
	seedPayrollEmployee(t, app, "emp-2", "Omar Farouk", EmployeeCompensationProfile{
		BaseSalary:     512,
		OtherAllowance: 256,
	})

	run, err := app.GeneratePayrollRun(period.ID)
	require.NoError(t, err)

	require.Equal(t, 2, run.TotalEmployees)
	require.Equal(t, 1960.0, run.GrossTotal)      // (1000+128+64) + (512+256)
	require.Equal(t, 48.0, run.DeductionsTotal)   // 32+16
	require.Equal(t, 1912.0, run.NetTotal)        // 1960−48
	require.Equal(t, 96.0, run.EmployerCostTotal) // emp-1 only
	require.Equal(t, "draft", run.Status)
	require.Equal(t, "BHD", run.Currency)

	require.Len(t, run.Items, 2)
	byName := map[string]PayrollRunItem{}
	for _, item := range run.Items {
		byName[item.EmployeeNameSnapshot] = item
	}
	aisha := byName["Aisha Rahman"]
	require.Equal(t, 1000.0, aisha.BaseSalary)
	require.Equal(t, 192.0, aisha.AllowancesTotal)
	require.Equal(t, 48.0, aisha.DeductionsTotal)
	require.Equal(t, 96.0, aisha.EmployerCostTotal)
	require.Equal(t, 1192.0, aisha.GrossPay)
	require.Equal(t, 1144.0, aisha.NetPay)
	// BASE + HOUSING + TRANSPORT + STANDARD + TAX + EMPLOYER (zero OTHER omitted)
	require.Len(t, aisha.Components, 6)

	omar := byName["Omar Farouk"]
	require.Equal(t, 768.0, omar.GrossPay)
	require.Equal(t, 768.0, omar.NetPay)
	// BASE + OTHER only
	require.Len(t, omar.Components, 2)

	require.Len(t, run.Payouts, 2)
	var payoutTotal float64
	for _, payout := range run.Payouts {
		require.Equal(t, "scheduled", payout.Status)
		payoutTotal += payout.Amount
	}
	require.Equal(t, 1912.0, payoutTotal)
}

// TestPayrollRunGeneration_NegativeNetRefused pins the Wave 6 refusal:
// when an item's deductions exceed its gross, the WHOLE run is refused
// with an error naming the employee and the amounts. This deliberately
// replaces the Wave 5 clamp golden (net clamped to 0 while the accrual
// journal still debited full gross — an unbalanced journal), a
// Commander-authorized semantics change: refuse-to-generate, no partial
// generation, no silent skipping.
func TestPayrollRunGeneration_NegativeNetRefused(t *testing.T) {
	app := setupPayrollApp(t)
	period := seedPayrollPeriod(t, app)

	seedPayrollEmployee(t, app, "emp-3", "Clamp Case", EmployeeCompensationProfile{
		BaseSalary:        100,
		StandardDeduction: 150,
	})
	// A healthy employee in the same run proves the refusal is whole-run.
	seedPayrollEmployee(t, app, "emp-4", "Healthy Case", EmployeeCompensationProfile{
		BaseSalary: 500,
	})

	_, err := app.GeneratePayrollRun(period.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Clamp Case")
	require.Contains(t, err.Error(), "150.000")
	require.Contains(t, err.Error(), "100.000")

	// Nothing was generated: no run rows persisted for the period.
	var count int64
	require.NoError(t, app.db.Model(&PayrollRun{}).Where("payroll_period_id = ?", period.ID).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

// TestPayrollPostingJournal_GoldenNumbers pins the accrual journal and the
// payout journal: exact line amounts, entry totals, account balances, and
// the synced expense-ledger entry.
func TestPayrollPostingJournal_GoldenNumbers(t *testing.T) {
	app := setupPayrollApp(t)
	period := seedPayrollPeriod(t, app)

	seedPayrollEmployee(t, app, "emp-1", "Aisha Rahman", EmployeeCompensationProfile{
		BaseSalary:         1000,
		HousingAllowance:   128,
		TransportAllowance: 64,
		StandardDeduction:  32,
		TaxDeduction:       16,
		EmployerCost:       96,
	})
	seedPayrollEmployee(t, app, "emp-2", "Omar Farouk", EmployeeCompensationProfile{
		BaseSalary:     512,
		OtherAllowance: 256,
	})

	run, err := app.GeneratePayrollRun(period.ID)
	require.NoError(t, err)
	run, err = app.ApprovePayrollRun(run.ID, "golden approval")
	require.NoError(t, err)
	require.Equal(t, "approved", run.Status)

	run, err = app.PostPayrollRun(run.ID)
	require.NoError(t, err)
	require.Equal(t, "posted", run.Status)
	require.NotNil(t, run.JournalEntryID)

	// Accrual journal: balanced at 2056 both sides.
	var journal JournalEntry
	require.NoError(t, app.db.First(&journal, "id = ?", *run.JournalEntryID).Error)
	require.Equal(t, 2056.0, journal.DebitTotal)  // gross 1960 + employer 96
	require.Equal(t, 2056.0, journal.CreditTotal) // net 1912 + deductions 48 + employer 96
	require.True(t, journal.IsPosted)
	require.Equal(t, "payroll_run", journal.SourceType)
	require.Equal(t, period.PeriodEnd.Format("2006-01-02"), journal.EntryDate.Format("2006-01-02"))

	var lines []JournalLine
	require.NoError(t, app.db.Where("entry_id = ?", journal.ID).Find(&lines).Error)
	require.Len(t, lines, 5)
	amounts := map[string][2]float64{}
	for _, line := range lines {
		var account ChartOfAccount
		require.NoError(t, app.db.First(&account, "id = ?", line.AccountID).Error)
		amounts[account.AccountCode] = [2]float64{line.Debit, line.Credit}
	}
	require.Equal(t, [2]float64{1960, 0}, amounts["6000"]) // Salaries & Wages
	require.Equal(t, [2]float64{96, 0}, amounts["6050"])   // Payroll Overheads
	require.Equal(t, [2]float64{0, 1912}, amounts["2210"]) // Payroll Payable
	require.Equal(t, [2]float64{0, 48}, amounts["2211"])   // Deductions Payable
	require.Equal(t, [2]float64{0, 96}, amounts["2212"])   // Employer Liabilities

	// Account balances after the accrual.
	require.Equal(t, 1960.0, accountBalance(t, app, "6000"))
	require.Equal(t, 96.0, accountBalance(t, app, "6050"))
	require.Equal(t, 1912.0, accountBalance(t, app, "2210"))
	require.Equal(t, 48.0, accountBalance(t, app, "2211"))
	require.Equal(t, 96.0, accountBalance(t, app, "2212"))

	// Expense ledger sync: net payout amount, posted, unpaid.
	var expense ExpenseEntry
	require.NoError(t, app.db.Where("source_type = ? AND source_ref_id = ?", "payroll", run.ID).First(&expense).Error)
	require.Equal(t, 1912.0, expense.Amount)
	require.Equal(t, 1912.0, expense.TotalAmount)
	require.Equal(t, 0.0, expense.VATAmount)
	require.Equal(t, "posted", expense.Status)
	require.Equal(t, "unpaid", expense.PaymentStatus)

	// Mark paid: payout journal clears the payable against cash.
	run, err = app.MarkPayrollRunPaid(run.ID, "2026-06-30", "TRF-GOLDEN-1", "")
	require.NoError(t, err)
	require.Equal(t, "paid", run.Status)
	require.NotNil(t, run.PayoutJournalEntryID)

	var payoutJournal JournalEntry
	require.NoError(t, app.db.First(&payoutJournal, "id = ?", *run.PayoutJournalEntryID).Error)
	require.Equal(t, 1912.0, payoutJournal.DebitTotal)
	require.Equal(t, 1912.0, payoutJournal.CreditTotal)
	require.Equal(t, "payroll_payout", payoutJournal.SourceType)

	require.Equal(t, 0.0, accountBalance(t, app, "2210"))     // payable cleared
	require.Equal(t, -1912.0, accountBalance(t, app, "1000")) // cash disbursed

	for _, payout := range run.Payouts {
		require.Equal(t, "paid", payout.Status)
		require.Equal(t, "TRF-GOLDEN-1", payout.PaymentReference)
	}

	require.NoError(t, app.db.Where("source_type = ? AND source_ref_id = ?", "payroll", run.ID).First(&expense).Error)
	require.Equal(t, "paid", expense.Status)
	require.Equal(t, "paid", expense.PaymentStatus)
}
