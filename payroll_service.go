package main

// Wave 5 A.2: the payroll domain lives in pkg/finance/payroll — models,
// run-generation arithmetic, the kernel-gated approval, and the accrual/
// payout posting journals (pinned by payroll_golden_test.go BEFORE the
// move). These delegates keep the Wails binding surface and the RBAC
// guards; the host implements the ports (employee directory, identity,
// UI events, and the payroll→expense-ledger bridge below).

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"

	financepayroll "ph_holdings_app/pkg/finance/payroll"
	"ph_holdings_app/pkg/kernel/actor"
)

// Type aliases keep the table shapes, JSON contracts, and model registry
// unchanged by the move.
type EmployeeCompensationProfile = financepayroll.CompensationProfile
type PayrollPeriod = financepayroll.Period
type PayrollRun = financepayroll.Run
type PayrollRunItem = financepayroll.RunItem
type PayrollComponent = financepayroll.Component
type PayrollPayout = financepayroll.Payout
type PayrollDashboardSummary = financepayroll.DashboardSummary

// Mission I (I-11): bound DDL is gated (this was the one payroll method with
// no gate at all); startup uses the internal.
func (a *App) EnsurePayrollFoundation() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	return a.ensurePayrollFoundationInternal()
}

func (a *App) ensurePayrollFoundationInternal() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	models := []any{
		&EmployeeCompensationProfile{},
		&PayrollPeriod{},
		&PayrollRun{},
		&PayrollRunItem{},
		&PayrollComponent{},
		&PayrollPayout{},
	}
	for _, model := range models {
		if !a.db.Migrator().HasTable(model) {
			if err := a.db.AutoMigrate(model); err != nil {
				return fmt.Errorf("failed to migrate %T: %w", model, err)
			}
		}
	}
	payrollDivisionDefaultDDL := "TEXT DEFAULT " + sqlStringLiteral(activeOverlay.DefaultDivision())
	a.addColumnIfNotExists("employee_compensation_profiles", "division", payrollDivisionDefaultDDL)
	a.addColumnIfNotExists("payroll_periods", "division", payrollDivisionDefaultDDL)
	a.addColumnIfNotExists("payroll_runs", "division", payrollDivisionDefaultDDL)
	a.addColumnIfNotExists("payroll_payouts", "division", payrollDivisionDefaultDDL)

	return nil
}

// ── Ports: the host side of pkg/finance/payroll ────────────────────────────

// appPayrollIdentity resolves the acting user for payroll attribution.
type appPayrollIdentity struct{ app *App }

func (p appPayrollIdentity) UserID() string { return p.app.getCurrentUserID() }
func (p appPayrollIdentity) ActorID() string {
	if current, err := p.app.GetCurrentEmployeeContext(); err == nil && current.EmployeeID != "" {
		return current.EmployeeID
	}
	if userID := p.app.getCurrentUserID(); strings.TrimSpace(userID) != "" {
		return userID
	}
	return "system"
}
func (p appPayrollIdentity) DisplayName() string { return p.app.getCurrentUserDisplayName() }

// appPayrollDirectory resolves employees from the collaboration hub.
type appPayrollDirectory struct{ app *App }

func (p appPayrollDirectory) Employees(ids []string) map[string]financepayroll.EmployeeRef {
	lookup := map[string]financepayroll.EmployeeRef{}
	if len(ids) == 0 || p.app.db == nil {
		return lookup
	}
	var employees []Employee
	if err := p.app.db.Where("id IN ?", ids).Find(&employees).Error; err != nil {
		return lookup
	}
	for _, employee := range employees {
		lookup[employee.ID] = financepayroll.EmployeeRef{
			ID:       employee.ID,
			FullName: employee.FullName,
			JobTitle: employee.JobTitle,
			IsActive: employee.IsActive,
		}
	}
	return lookup
}

// appPayrollEvents publishes payroll UI events through the Wails runtime.
type appPayrollEvents struct{ app *App }

func (p appPayrollEvents) Emit(name string, payload map[string]any) {
	if p.app.ctx == nil {
		return
	}
	runtime.EventsEmit(p.app.ctx, name, payload)
}

// appPayrollExpenseBridge mirrors posted/paid runs into the expense ledger.
type appPayrollExpenseBridge struct{ app *App }

func (p appPayrollExpenseBridge) SyncRunExpense(tx *gorm.DB, run *PayrollRun) error {
	return p.app.syncPayrollRunExpenseEntry(tx, run)
}

func (p appPayrollExpenseBridge) EmitExpenseUpdated(runID string) {
	emitExpenseEvent(p.app, "expenses:updated", map[string]any{"entity": "entry", "action": "payroll_sync", "id": runID})
}

// ── RBAC guards (payroll permissions fall back to finance) ────────────────

func (a *App) requirePayrollView() error {
	if err := a.requirePermission("payroll:view"); err != nil {
		return a.requirePermission("finance:view")
	}
	return nil
}

func (a *App) requirePayrollCreate() error {
	if err := a.requirePermission("payroll:create"); err != nil {
		return a.requirePermission("finance:create")
	}
	return nil
}

func (a *App) requirePayrollUpdate() error {
	if err := a.requirePermission("payroll:update"); err != nil {
		return a.requirePermission("finance:update")
	}
	return nil
}

func (a *App) requirePayrollApprove() error {
	if err := a.requirePermission("payroll:approve"); err != nil {
		return a.requirePermission("finance:update")
	}
	return nil
}

func (a *App) payrollGuarded(check func() error) error {
	if err := check(); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return nil
}

// ── Wails-bound delegates ──────────────────────────────────────────────────

func (a *App) ListEmployeeCompensationProfiles(activeOnly bool) ([]EmployeeCompensationProfile, error) {
	if err := a.payrollGuarded(a.requirePayrollView); err != nil {
		return nil, err
	}
	return a.payrollService().ListProfiles(activeOnly)
}

func (a *App) UpsertEmployeeCompensationProfile(profile EmployeeCompensationProfile) (EmployeeCompensationProfile, error) {
	if strings.TrimSpace(profile.ID) == "" {
		if err := a.requirePayrollCreate(); err != nil {
			return EmployeeCompensationProfile{}, err
		}
	} else {
		if err := a.requirePayrollUpdate(); err != nil {
			return EmployeeCompensationProfile{}, err
		}
	}
	if a.db == nil {
		return EmployeeCompensationProfile{}, fmt.Errorf("database not initialized")
	}
	return a.payrollService().UpsertProfile(profile)
}

func (a *App) ListPayrollPeriods(includeClosed bool) ([]PayrollPeriod, error) {
	if err := a.payrollGuarded(a.requirePayrollView); err != nil {
		return nil, err
	}
	return a.payrollService().ListPeriods(includeClosed)
}

func (a *App) CreatePayrollPeriod(period PayrollPeriod) (PayrollPeriod, error) {
	if err := a.payrollGuarded(a.requirePayrollCreate); err != nil {
		return PayrollPeriod{}, err
	}
	return a.payrollService().CreatePeriod(period)
}

func (a *App) ListPayrollRuns(payrollPeriodID string) ([]PayrollRun, error) {
	if err := a.payrollGuarded(a.requirePayrollView); err != nil {
		return nil, err
	}
	return a.payrollService().ListRuns(payrollPeriodID)
}

func (a *App) GetPayrollRun(runID string) (PayrollRun, error) {
	if err := a.payrollGuarded(a.requirePayrollView); err != nil {
		return PayrollRun{}, err
	}
	return a.payrollService().GetRun(runID)
}

func (a *App) GeneratePayrollRun(payrollPeriodID string) (PayrollRun, error) {
	if err := a.payrollGuarded(a.requirePayrollCreate); err != nil {
		return PayrollRun{}, err
	}
	return a.payrollService().GenerateRun(payrollPeriodID)
}

func (a *App) ApprovePayrollRun(runID, notes string) (PayrollRun, error) {
	if err := a.payrollGuarded(a.requirePayrollApprove); err != nil {
		return PayrollRun{}, err
	}
	return a.payrollService().ApproveRun(runID, notes)
}

func (a *App) PostPayrollRun(runID string) (PayrollRun, error) {
	if err := a.payrollGuarded(a.requirePayrollUpdate); err != nil {
		return PayrollRun{}, err
	}
	return a.payrollService().PostRun(runID)
}

func (a *App) MarkPayrollRunPaid(runID, paidAtISO, paymentReference, bankAccountID string) (PayrollRun, error) {
	if err := a.payrollGuarded(a.requirePayrollUpdate); err != nil {
		return PayrollRun{}, err
	}
	return a.payrollService().MarkPaid(runID, paidAtISO, paymentReference, bankAccountID)
}

func (a *App) ListPayrollPayouts(payrollRunID string) ([]PayrollPayout, error) {
	if err := a.payrollGuarded(a.requirePayrollView); err != nil {
		return nil, err
	}
	return a.payrollService().ListPayouts(payrollRunID)
}

func (a *App) ListUnreconciledPayrollPayouts() ([]PayrollPayout, error) {
	if err := a.payrollGuarded(a.requirePayrollView); err != nil {
		return nil, err
	}
	return a.payrollService().ListUnreconciledPayouts()
}

func (a *App) ListPayrollDashboardSummary() (PayrollDashboardSummary, error) {
	if err := a.payrollGuarded(a.requirePayrollView); err != nil {
		return PayrollDashboardSummary{}, err
	}
	return a.payrollService().Dashboard()
}

// gatePayrollRunApproval keeps the historical root entry point for the
// kernel approval gate (used by approval_routing_test.go); the gate lives
// in pkg/finance/payroll.
func (a *App) gatePayrollRunApproval(run PayrollRun, notes string, by actor.Actor) error {
	return financepayroll.GateRunApproval(run, notes, by)
}

// ── Payroll → expense-ledger bridge (host side; the expense service owns
//    numbering, approvals, and its own events) ────────────────────────────

func (a *App) syncPayrollRunExpenseEntry(tx *gorm.DB, run *PayrollRun) error {
	if tx == nil {
		tx = a.db
	}
	if tx == nil || run == nil || strings.TrimSpace(run.ID) == "" || run.NetTotal <= 0 {
		return nil
	}

	category, err := a.ensurePayrollExpenseCategory(tx)
	if err != nil {
		return fmt.Errorf("failed to ensure payroll expense category: %w", err)
	}

	expenseDate, dueDate := a.resolvePayrollExpenseDates(tx, run)
	status := "posted"
	paymentStatus := "unpaid"
	if strings.EqualFold(run.Status, "paid") {
		status = "paid"
		paymentStatus = "paid"
	}

	description := fmt.Sprintf("Salary payment for %s", strings.TrimSpace(run.RunNumber))
	notes := fmt.Sprintf(
		"Linked from payroll run %s. Gross %.3f BHD, deductions %.3f BHD, employer cost %.3f BHD. Expense ledger tracks the net salary payout for operational payment visibility.",
		strings.TrimSpace(run.RunNumber),
		run.GrossTotal,
		run.DeductionsTotal,
		run.EmployerCostTotal,
	)

	var existing ExpenseEntry
	err = tx.Where("source_type = ? AND source_ref_id = ?", "payroll", run.ID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to look up payroll expense entry: %w", err)
	}

	paymentReference := strings.TrimSpace(run.PaymentReference)
	now := time.Now()
	if err == gorm.ErrRecordNotFound {
		entryNumber, genErr := generateExpenseEntryNumber(a)
		if genErr != nil {
			return genErr
		}
		sourceRefID := run.ID
		entry := ExpenseEntry{
			Base:             Base{ID: uuid.New().String(), CreatedBy: a.getCurrentUserID(), CreatedAt: now, UpdatedAt: now},
			EntryNumber:      entryNumber,
			ExpenseDate:      expenseDate,
			DueDate:          &dueDate,
			Description:      description,
			CategoryID:       category.ID,
			SourceType:       "payroll",
			SourceRefID:      &sourceRefID,
			CostCenter:       "Payroll",
			Currency:         firstPopulatedString(run.Currency, "BHD"),
			Division:         normalizeDivisionName(run.Division),
			Amount:           payrollClampAmount(run.NetTotal),
			VATAmount:        0,
			TotalAmount:      payrollClampAmount(run.NetTotal),
			Status:           status,
			PaymentStatus:    paymentStatus,
			PostedAt:         run.PostedAt,
			PostedBy:         run.PostedBy,
			PaidAt:           run.PaidAt,
			PaymentMethod:    "Payroll Transfer",
			PaymentReference: paymentReference,
			BankAccountID:    run.BankAccountID,
			JournalEntryID:   run.JournalEntryID,
			Notes:            notes,
		}
		if paymentStatus != "paid" {
			entry.PaymentMethod = ""
		}
		if err := tx.Create(&entry).Error; err != nil {
			return fmt.Errorf("failed to create payroll expense entry: %w", err)
		}
		recordExpenseApproval(a, entry.ID, "payroll_sync", run.RunNumber)
		return nil
	}

	updates := map[string]any{
		"expense_date":      expenseDate,
		"due_date":          &dueDate,
		"description":       description,
		"category_id":       category.ID,
		"cost_center":       "Payroll",
		"currency":          firstPopulatedString(run.Currency, "BHD"),
		"division":          normalizeDivisionName(run.Division),
		"amount":            payrollClampAmount(run.NetTotal),
		"vat_amount":        0,
		"total_amount":      payrollClampAmount(run.NetTotal),
		"status":            status,
		"payment_status":    paymentStatus,
		"posted_at":         run.PostedAt,
		"posted_by":         run.PostedBy,
		"paid_at":           run.PaidAt,
		"payment_reference": paymentReference,
		"bank_account_id":   run.BankAccountID,
		"journal_entry_id":  run.JournalEntryID,
		"notes":             notes,
		"updated_at":        now,
	}
	if paymentStatus == "paid" {
		updates["payment_method"] = "Payroll Transfer"
	} else {
		updates["payment_method"] = ""
	}
	if err := tx.Model(&existing).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update payroll expense entry: %w", err)
	}
	return nil
}

func (a *App) ensurePayrollExpenseCategory(tx *gorm.DB) (ExpenseCategory, error) {
	if tx == nil {
		tx = a.db
	}
	var category ExpenseCategory
	if err := tx.Where("UPPER(code) = ?", "PAYROLL").First(&category).Error; err == nil {
		return category, nil
	} else if err != gorm.ErrRecordNotFound {
		return ExpenseCategory{}, err
	}

	account, err := a.ensureSupportingAccount("6000", accountNameForCode("6000"), "Expense")
	if err != nil {
		return ExpenseCategory{}, err
	}
	accountID := account.ID
	category = ExpenseCategory{
		Base:        Base{CreatedBy: a.getCurrentUserID()},
		Name:        "Employee Salaries",
		Code:        "PAYROLL",
		Description: "Salaries, wages, and payroll overhead",
		GLAccountID: &accountID,
		IsActive:    true,
		SortOrder:   30,
	}
	if err := tx.Create(&category).Error; err != nil {
		return ExpenseCategory{}, err
	}
	return category, nil
}

func (a *App) resolvePayrollExpenseDates(tx *gorm.DB, run *PayrollRun) (time.Time, time.Time) {
	if run == nil {
		now := time.Now()
		return now, now
	}
	if run.PaidAt != nil {
		return *run.PaidAt, *run.PaidAt
	}
	if tx != nil {
		var period PayrollPeriod
		if err := tx.First(&period, "id = ?", run.PayrollPeriodID).Error; err == nil {
			if period.PaymentDate != nil {
				return *period.PaymentDate, *period.PaymentDate
			}
			return period.PeriodEnd, period.PeriodEnd
		}
	}
	if run.PostedAt != nil {
		return *run.PostedAt, *run.PostedAt
	}
	if run.GeneratedAt != nil {
		return *run.GeneratedAt, *run.GeneratedAt
	}
	now := time.Now()
	return now, now
}

func firstPopulatedString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func payrollClampAmount(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}
