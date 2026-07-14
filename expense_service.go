package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	finance "ph_holdings_app/pkg/finance"
	financeexpense "ph_holdings_app/pkg/finance/expense"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

type ExpenseCategory = finance.ExpenseCategory

type ExpenseVendor = finance.ExpenseVendor

type ExpenseEntry = finance.ExpenseEntry

type ExpenseAllocation struct {
	Base
	ExpenseEntryID    string  `gorm:"index;size:36" json:"expense_entry_id"`
	AllocationType    string  `gorm:"size:30" json:"allocation_type"`
	ProjectID         *string `gorm:"size:36" json:"project_id"`
	CustomerID        *string `gorm:"size:36" json:"customer_id"`
	OpportunityID     *string `gorm:"size:36" json:"opportunity_id"`
	OrderID           *string `gorm:"size:36" json:"order_id"`
	AllocationPercent float64 `gorm:"default:100" json:"allocation_percent"`
	AllocatedAmount   float64 `gorm:"type:decimal(15,3)" json:"allocated_amount"`
	Notes             string  `gorm:"type:text" json:"notes"`
}

func (ExpenseAllocation) TableName() string { return "expense_allocations" }

type RecurringExpense = finance.RecurringExpense

type ExpenseAttachment struct {
	Base
	ExpenseEntryID string `gorm:"index;size:36" json:"expense_entry_id"`
	FileName       string `gorm:"size:255" json:"file_name"`
	FilePath       string `gorm:"size:1000" json:"file_path"`
	MimeType       string `gorm:"size:120" json:"mime_type"`
}

func (ExpenseAttachment) TableName() string { return "expense_attachments" }

type ExpenseApproval struct {
	Base
	ExpenseEntryID string    `gorm:"index;size:36" json:"expense_entry_id"`
	Action         string    `gorm:"size:30" json:"action"`
	ActorID        string    `gorm:"size:36" json:"actor_id"`
	Notes          string    `gorm:"type:text" json:"notes"`
	ActionAt       time.Time `gorm:"index" json:"action_at"`
}

func (ExpenseApproval) TableName() string { return "expense_approvals" }

type ExpenseDashboardSummary struct {
	TotalDrafts         int     `json:"total_drafts"`
	TotalSubmitted      int     `json:"total_submitted"`
	TotalApprovedUnpaid int     `json:"total_approved_unpaid"`
	TotalRecurring      int     `json:"total_recurring"`
	MonthToDateSpend    float64 `json:"month_to_date_spend"`
	UpcomingCommitments float64 `json:"upcoming_commitments"`
}

func (a *App) EnsureExpenseFoundation() error {
	return a.expenseService().EnsureFoundation()
}

func ensureExpenseFoundation(a *App) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	models := []any{
		&CompanyBankAccount{},
		&ExpenseCategory{},
		&ExpenseVendor{},
		&ExpenseEntry{},
		&ExpenseAllocation{},
		&RecurringExpense{},
		&ExpenseAttachment{},
		&ExpenseApproval{},
	}
	for _, model := range models {
		if err := a.db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}
	divisionDefaultDDL := "TEXT DEFAULT " + sqlStringLiteral(activeOverlay.DefaultDivision())
	a.addColumnIfNotExists("expense_entries", "payment_method", "TEXT")
	a.addColumnIfNotExists("expense_entries", "division", divisionDefaultDDL)
	a.addColumnIfNotExists("recurring_expenses", "division", divisionDefaultDDL)
	a.addColumnIfNotExists("bank_expense_entries", "division", divisionDefaultDDL)
	if err := a.seedCompanyBankAccountsInternal(); err != nil {
		return fmt.Errorf("failed to seed company bank accounts: %w", err)
	}

	return seedDefaultExpenseCategories(a)
}

func requireExpenseView(a *App) error {
	if err := a.requirePermission("expenses:view"); err != nil {
		return a.requirePermission("finance:view")
	}
	return nil
}

func requireExpenseCreate(a *App) error {
	if err := a.requirePermission("expenses:create"); err != nil {
		return a.requirePermission("finance:create")
	}
	return nil
}

func requireExpenseUpdate(a *App) error {
	if err := a.requirePermission("expenses:update"); err != nil {
		return a.requirePermission("finance:update")
	}
	return nil
}

func (a *App) ListExpenseCategories(activeOnly bool) ([]ExpenseCategory, error) {
	return a.expenseService().ListCategories(activeOnly)
}

func listExpenseCategories(a *App, activeOnly bool) ([]ExpenseCategory, error) {
	if err := requireExpenseView(a); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := a.db.Order("sort_order ASC, name ASC")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	var categories []ExpenseCategory
	if err := query.Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("failed to list expense categories: %w", err)
	}

	accountNames := lookupAccountNames(a, categoryAccountIDs(categories))
	for i := range categories {
		if categories[i].GLAccountID != nil {
			categories[i].GLAccountName = accountNames[*categories[i].GLAccountID]
		}
	}
	return categories, nil
}

func (a *App) CreateExpenseCategory(category ExpenseCategory) (ExpenseCategory, error) {
	return a.expenseService().CreateCategory(category)
}

func createExpenseCategory(a *App, category ExpenseCategory) (ExpenseCategory, error) {
	if err := requireExpenseCreate(a); err != nil {
		return ExpenseCategory{}, err
	}
	if a.db == nil {
		return ExpenseCategory{}, fmt.Errorf("database not initialized")
	}

	category.Name = strings.TrimSpace(category.Name)
	category.Code = strings.ToUpper(strings.TrimSpace(category.Code))
	if category.Name == "" {
		return ExpenseCategory{}, fmt.Errorf("category name is required")
	}
	if category.Code == "" {
		category.Code = strings.ToUpper(strings.ReplaceAll(category.Name, " ", "_"))
	}
	category.IsActive = true
	category.CreatedBy = a.getCurrentUserID()
	if err := a.db.Create(&category).Error; err != nil {
		return ExpenseCategory{}, fmt.Errorf("failed to create expense category: %w", err)
	}
	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "category", "action": "create", "id": category.ID})
	return category, nil
}

func (a *App) DeleteExpenseCategory(categoryID string) error {
	if ok, err := a.guardDeleteOrRequest("expenses:update", "expense_category", categoryID, "Expense category"); !ok {
		return err
	}
	if err := requireExpenseUpdate(a); err != nil {
		return err
	}
	categoryID = strings.TrimSpace(categoryID)
	if err := financeexpense.DeleteCategory(a.db, categoryID); err != nil {
		return err
	}
	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "category", "action": "delete", "id": categoryID})
	return nil
}

func (a *App) ListExpenseVendors(activeOnly bool) ([]ExpenseVendor, error) {
	return a.expenseService().ListVendors(activeOnly)
}

func listExpenseVendors(a *App, activeOnly bool) ([]ExpenseVendor, error) {
	if err := requireExpenseView(a); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := a.db.Order("name ASC")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	var vendors []ExpenseVendor
	if err := query.Find(&vendors).Error; err != nil {
		return nil, fmt.Errorf("failed to list expense vendors: %w", err)
	}
	return vendors, nil
}

func (a *App) CreateExpenseVendor(vendor ExpenseVendor) (ExpenseVendor, error) {
	return a.expenseService().CreateVendor(vendor)
}

func createExpenseVendor(a *App, vendor ExpenseVendor) (ExpenseVendor, error) {
	if err := requireExpenseCreate(a); err != nil {
		return ExpenseVendor{}, err
	}
	if a.db == nil {
		return ExpenseVendor{}, fmt.Errorf("database not initialized")
	}
	vendor.Name = strings.TrimSpace(vendor.Name)
	if vendor.Name == "" {
		return ExpenseVendor{}, fmt.Errorf("vendor name is required")
	}
	vendor.IsActive = true
	vendor.CreatedBy = a.getCurrentUserID()
	if err := a.db.Create(&vendor).Error; err != nil {
		return ExpenseVendor{}, fmt.Errorf("failed to create expense vendor: %w", err)
	}
	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "vendor", "action": "create", "id": vendor.ID})
	return vendor, nil
}

func (a *App) DeleteExpenseVendor(vendorID string) error {
	if ok, err := a.guardDeleteOrRequest("expenses:update", "expense_vendor", vendorID, "Expense vendor"); !ok {
		return err
	}
	if err := requireExpenseUpdate(a); err != nil {
		return err
	}
	vendorID = strings.TrimSpace(vendorID)
	if err := financeexpense.DeleteVendor(a.db, vendorID); err != nil {
		return err
	}
	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "vendor", "action": "delete", "id": vendorID})
	return nil
}

func (a *App) CreateExpenseEntry(entry ExpenseEntry) (ExpenseEntry, error) {
	return a.expenseService().CreateEntry(entry)
}

func createExpenseEntry(a *App, entry ExpenseEntry) (ExpenseEntry, error) {
	if err := requireExpenseCreate(a); err != nil {
		return ExpenseEntry{}, err
	}
	if a.db == nil {
		return ExpenseEntry{}, fmt.Errorf("database not initialized")
	}

	prepared, err := prepareExpenseEntry(a, entry, false)
	if err != nil {
		return ExpenseEntry{}, err
	}

	if err := a.db.Create(&prepared).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("failed to create expense entry: %w", err)
	}
	recordExpenseApproval(a, prepared.ID, "created", "")
	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "entry", "action": "create", "id": prepared.ID})
	return decorateExpenseEntry(a, prepared), nil
}

func (a *App) DeleteExpenseEntry(entryID string) error {
	if ok, err := a.guardDeleteOrRequest("expenses:update", "expense_entry", entryID, "Expense entry"); !ok {
		return err
	}
	if err := requireExpenseUpdate(a); err != nil {
		return err
	}
	entryID = strings.TrimSpace(entryID)
	if err := financeexpense.DeleteEntry(a.db, entryID); err != nil {
		return err
	}
	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "entry", "action": "delete", "id": entryID})
	return nil
}

func (a *App) ListExpenseEntries(status string, includePaid bool) ([]ExpenseEntry, error) {
	return a.expenseService().ListEntries(status, includePaid)
}

func listExpenseEntries(a *App, status string, includePaid bool) ([]ExpenseEntry, error) {
	if err := requireExpenseView(a); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := a.db.Order("expense_date DESC, created_at DESC")
	status = strings.TrimSpace(strings.ToLower(status))
	if status != "" {
		query = query.Where("LOWER(status) = ?", status)
	}
	if !includePaid {
		query = query.Where("payment_status != ?", "paid")
	}

	var entries []ExpenseEntry
	if err := query.Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("failed to list expense entries: %w", err)
	}
	return decorateExpenseEntries(a, entries), nil
}

func (a *App) ListExpenseDashboardSummary() (ExpenseDashboardSummary, error) {
	return a.expenseService().ListDashboardSummary()
}

func listExpenseDashboardSummary(a *App) (ExpenseDashboardSummary, error) {
	if err := requireExpenseView(a); err != nil {
		return ExpenseDashboardSummary{}, err
	}
	if a.db == nil {
		return ExpenseDashboardSummary{}, fmt.Errorf("database not initialized")
	}

	summary := ExpenseDashboardSummary{}
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	windowEnd := now.AddDate(0, 0, 30)

	var draftCount, submittedCount, approvedCount, recurringCount int64
	a.db.Model(&ExpenseEntry{}).Where("status = ?", "draft").Count(&draftCount)
	a.db.Model(&ExpenseEntry{}).Where("status = ?", "submitted").Count(&submittedCount)
	a.db.Model(&ExpenseEntry{}).Where("status IN ? AND payment_status != ?", []string{"approved", "posted"}, "paid").Count(&approvedCount)
	a.db.Model(&RecurringExpense{}).Where("is_active = ?", true).Count(&recurringCount)
	summary.TotalDrafts = int(draftCount)
	summary.TotalSubmitted = int(submittedCount)
	summary.TotalApprovedUnpaid = int(approvedCount)
	summary.TotalRecurring = int(recurringCount)

	a.db.Model(&ExpenseEntry{}).
		Where("expense_date >= ? AND status NOT IN ?", monthStart, []string{"draft", "rejected"}).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&summary.MonthToDateSpend)

	a.db.Model(&ExpenseEntry{}).
		Where("(due_date BETWEEN ? AND ? OR (due_date IS NULL AND expense_date BETWEEN ? AND ?)) AND status IN ? AND payment_status != ?",
			now, windowEnd, now, windowEnd, []string{"approved", "posted"}, "paid").
		Select("COALESCE(SUM(total_amount), 0)").Scan(&summary.UpcomingCommitments)

	var recurring []RecurringExpense
	if err := a.db.Where("is_active = ? AND next_run_date <= ?", true, windowEnd).Find(&recurring).Error; err == nil {
		for _, item := range recurring {
			for runAt := item.NextRunDate; !runAt.After(windowEnd); runAt = nextRecurringDate(runAt, item.Frequency, item.IntervalValue) {
				if runAt.Before(now) {
					continue
				}
				summary.UpcomingCommitments += item.DefaultAmount + item.DefaultVATAmount
			}
		}
	}

	return summary, nil
}

func (a *App) SubmitExpenseEntry(entryID string) (ExpenseEntry, error) {
	return a.expenseService().SubmitEntry(entryID)
}

func submitExpenseEntry(a *App, entryID string) (ExpenseEntry, error) {
	return transitionExpenseEntry(a, entryID, "submitted", "", func(entry *ExpenseEntry, actor string, now time.Time) {
		entry.SubmittedAt = &now
		entry.SubmittedBy = actor
	})
}

func (a *App) ApproveExpenseEntry(entryID, notes string) (ExpenseEntry, error) {
	return a.expenseService().ApproveEntry(entryID, notes)
}

func approveExpenseEntry(a *App, entryID, notes string) (ExpenseEntry, error) {
	return transitionExpenseEntry(a, entryID, "approved", notes, func(entry *ExpenseEntry, actor string, now time.Time) {
		entry.ApprovedAt = &now
		entry.ApprovedBy = actor
		entry.RejectedAt = nil
		entry.RejectedBy = ""
		entry.RejectionReason = ""
	})
}

func (a *App) RejectExpenseEntry(entryID, reason string) (ExpenseEntry, error) {
	return a.expenseService().RejectEntry(entryID, reason)
}

func rejectExpenseEntry(a *App, entryID, reason string) (ExpenseEntry, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "Rejected"
	}
	return transitionExpenseEntry(a, entryID, "rejected", reason, func(entry *ExpenseEntry, actor string, now time.Time) {
		entry.RejectedAt = &now
		entry.RejectedBy = actor
		entry.RejectionReason = reason
	})
}

func (a *App) PostExpenseEntry(entryID string) (ExpenseEntry, error) {
	return a.expenseService().PostEntry(entryID)
}

func postExpenseEntry(a *App, entryID string) (ExpenseEntry, error) {
	if err := requireExpenseUpdate(a); err != nil {
		return ExpenseEntry{}, err
	}
	if a.db == nil {
		return ExpenseEntry{}, fmt.Errorf("database not initialized")
	}

	var entry ExpenseEntry
	if err := a.db.First(&entry, "id = ?", strings.TrimSpace(entryID)).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("expense entry not found: %w", err)
	}
	if entry.Status != "approved" && entry.Status != "paid" && entry.Status != "posted" {
		return ExpenseEntry{}, fmt.Errorf("only approved expenses can be posted")
	}
	if entry.JournalEntryID != nil && *entry.JournalEntryID != "" {
		return decorateExpenseEntry(a, entry), nil
	}

	var category ExpenseCategory
	if err := a.db.First(&category, "id = ?", entry.CategoryID).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("expense category not found: %w", err)
	}

	journalID, err := postExpenseJournal(a, &entry, category)
	if err != nil {
		return ExpenseEntry{}, err
	}

	now := time.Now()
	entry.Status = "posted"
	entry.PostedAt = &now
	entry.PostedBy = getExpenseActorID(a)
	entry.JournalEntryID = &journalID
	if err := a.db.Model(&entry).Updates(map[string]any{
		"status":           entry.Status,
		"posted_at":        entry.PostedAt,
		"posted_by":        entry.PostedBy,
		"journal_entry_id": entry.JournalEntryID,
		"updated_at":       now,
	}).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("failed to mark expense as posted: %w", err)
	}

	recordExpenseApproval(a, entry.ID, "posted", "")
	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "entry", "action": "posted", "id": entry.ID})
	return decorateExpenseEntry(a, entry), nil
}

func (a *App) MarkExpenseEntryPaid(entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod string) (ExpenseEntry, error) {
	return a.expenseService().MarkEntryPaid(entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod)
}

func markExpenseEntryPaid(a *App, entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod string) (ExpenseEntry, error) {
	if err := requireExpenseUpdate(a); err != nil {
		return ExpenseEntry{}, err
	}
	if a.db == nil {
		return ExpenseEntry{}, fmt.Errorf("database not initialized")
	}

	var entry ExpenseEntry
	if err := a.db.First(&entry, "id = ?", strings.TrimSpace(entryID)).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("expense entry not found: %w", err)
	}
	if entry.PaymentStatus == "paid" || entry.Status == "paid" {
		return ExpenseEntry{}, fmt.Errorf("expense is already paid")
	}
	if entry.Status != "posted" {
		return ExpenseEntry{}, fmt.Errorf("expense must be posted before it can be marked paid")
	}

	if strings.TrimSpace(paidAtISO) == "" {
		return ExpenseEntry{}, fmt.Errorf("payment date is required")
	}
	paidAt, err := time.Parse(time.RFC3339, strings.TrimSpace(paidAtISO))
	if err != nil {
		return ExpenseEntry{}, fmt.Errorf("invalid payment date: %w", err)
	}
	paymentReference = strings.TrimSpace(paymentReference)
	if paymentReference == "" {
		return ExpenseEntry{}, fmt.Errorf("payment reference is required")
	}
	bankAccountID = strings.TrimSpace(bankAccountID)
	if bankAccountID == "" {
		return ExpenseEntry{}, fmt.Errorf("bank account is required")
	}
	paymentMethod = strings.TrimSpace(paymentMethod)
	if paymentMethod == "" {
		return ExpenseEntry{}, fmt.Errorf("payment method is required")
	}

	var bankAccount CompanyBankAccount
	if err := a.db.First(&bankAccount, "id = ? AND is_active = ?", bankAccountID, true).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("bank account not found: %w", err)
	}
	bankDivision := normalizeDivisionName(bankAccount.Division)
	entryDivision := normalizeDivisionName(entry.Division)
	if entryDivision != bankDivision {
		return ExpenseEntry{}, fmt.Errorf("expense belongs to %s but bank account belongs to %s", entryDivision, bankDivision)
	}

	updates := map[string]any{
		"status":            "paid",
		"payment_status":    "paid",
		"paid_at":           &paidAt,
		"payment_method":    paymentMethod,
		"payment_reference": paymentReference,
		"bank_account_id":   bankAccountID,
		"division":          bankDivision,
		"updated_at":        time.Now(),
	}
	if err := a.db.Model(&entry).Updates(updates).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("failed to mark expense as paid: %w", err)
	}

	entry.Status = "paid"
	entry.PaymentStatus = "paid"
	entry.PaidAt = &paidAt
	entry.PaymentMethod = paymentMethod
	entry.PaymentReference = paymentReference
	entry.BankAccountID = &bankAccountID
	entry.Division = bankDivision
	recordExpenseApproval(a, entry.ID, "paid", paymentReference)
	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "entry", "action": "paid", "id": entry.ID})
	return decorateExpenseEntry(a, entry), nil
}

func (a *App) ListRecurringExpenses(activeOnly bool) ([]RecurringExpense, error) {
	return a.expenseService().ListRecurring(activeOnly)
}

func listRecurringExpenses(a *App, activeOnly bool) ([]RecurringExpense, error) {
	if err := requireExpenseView(a); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	query := a.db.Order("next_run_date ASC, name ASC")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	var items []RecurringExpense
	if err := query.Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list recurring expenses: %w", err)
	}
	return decorateRecurringExpenses(a, items), nil
}

func (a *App) CreateRecurringExpense(item RecurringExpense) (RecurringExpense, error) {
	return a.expenseService().CreateRecurring(item)
}

func createRecurringExpense(a *App, item RecurringExpense) (RecurringExpense, error) {
	if err := requireExpenseCreate(a); err != nil {
		return RecurringExpense{}, err
	}
	if a.db == nil {
		return RecurringExpense{}, fmt.Errorf("database not initialized")
	}

	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return RecurringExpense{}, fmt.Errorf("recurring expense name is required")
	}
	if strings.TrimSpace(item.CategoryID) == "" {
		return RecurringExpense{}, fmt.Errorf("category is required")
	}
	if item.Currency == "" {
		item.Currency = "BHD"
	}
	if item.IntervalValue <= 0 {
		item.IntervalValue = 1
	}
	if item.Frequency == "" {
		item.Frequency = "monthly"
	}
	if item.NextRunDate.IsZero() {
		item.NextRunDate = time.Now()
	}
	item.Division = normalizeDivisionName(item.Division)
	item.IsActive = true
	item.CreatedBy = a.getCurrentUserID()

	if err := a.db.Create(&item).Error; err != nil {
		return RecurringExpense{}, fmt.Errorf("failed to create recurring expense: %w", err)
	}
	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "recurring", "action": "create", "id": item.ID})
	return decorateRecurringExpense(a, item), nil
}

func (a *App) DeleteRecurringExpense(recurringID string) error {
	return a.expenseService().DeleteRecurring(recurringID)
}

func deleteRecurringExpense(a *App, recurringID string) error {
	if ok, err := a.guardDeleteOrRequest("expenses:update", "recurring_expense", recurringID, "Recurring expense"); !ok {
		return err
	}
	if err := requireExpenseUpdate(a); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	recurringID = strings.TrimSpace(recurringID)
	if recurringID == "" {
		return fmt.Errorf("recurring expense id is required")
	}

	var item RecurringExpense
	if err := a.db.First(&item, "id = ?", recurringID).Error; err != nil {
		return fmt.Errorf("recurring expense not found: %w", err)
	}

	if err := a.db.Delete(&item).Error; err != nil {
		return fmt.Errorf("failed to delete recurring expense: %w", err)
	}

	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "recurring", "action": "delete", "id": recurringID})
	return nil
}

func (a *App) GenerateRecurringExpenses(cutoffISO string) ([]ExpenseEntry, error) {
	return a.expenseService().GenerateRecurring(cutoffISO)
}

func generateRecurringExpenses(a *App, cutoffISO string) ([]ExpenseEntry, error) {
	if err := requireExpenseUpdate(a); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	cutoff := time.Now().AddDate(0, 1, 0)
	if strings.TrimSpace(cutoffISO) != "" {
		parsed, err := time.Parse(time.RFC3339, cutoffISO)
		if err != nil {
			return nil, fmt.Errorf("invalid cutoff date: %w", err)
		}
		cutoff = parsed
	}

	var recurring []RecurringExpense
	if err := a.db.Where("is_active = ? AND next_run_date <= ?", true, cutoff).Order("next_run_date ASC").Find(&recurring).Error; err != nil {
		return nil, fmt.Errorf("failed to load recurring expenses: %w", err)
	}

	created := make([]ExpenseEntry, 0)
	for _, item := range recurring {
		runAt := item.NextRunDate
		for !runAt.After(cutoff) {
			entry, err := createExpenseEntryFromRecurring(a, item, runAt)
			if err != nil {
				return nil, err
			}
			created = append(created, entry)
			lastGenerated := runAt
			nextRun := nextRecurringDate(runAt, item.Frequency, item.IntervalValue)
			if err := a.db.Model(&item).Updates(map[string]any{
				"last_generated_at": &lastGenerated,
				"next_run_date":     nextRun,
				"updated_at":        time.Now(),
			}).Error; err != nil {
				return nil, fmt.Errorf("failed to update recurring expense: %w", err)
			}
			item.LastGeneratedAt = &lastGenerated
			item.NextRunDate = nextRun
			runAt = nextRun
		}
	}

	if len(created) > 0 {
		emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "recurring", "action": "generated", "count": len(created)})
	}
	return created, nil
}

func (a *App) ListBankExpenseCandidates(includeLinked bool) ([]BankExpenseEntry, error) {
	return a.expenseService().ListBankCandidates(includeLinked)
}

func listBankExpenseCandidates(a *App, includeLinked bool) ([]BankExpenseEntry, error) {
	if err := requireExpenseView(a); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := a.db.Order("expense_date DESC")
	if !includeLinked {
		query = query.Where("id NOT IN (?)",
			a.db.Model(&ExpenseEntry{}).Select("bank_expense_entry_id").Where("bank_expense_entry_id IS NOT NULL"))
	}

	var items []BankExpenseEntry
	if err := query.Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list bank expense candidates: %w", err)
	}
	return items, nil
}

func (a *App) CreateExpenseFromBankCandidate(bankExpenseID, categoryID string) (ExpenseEntry, error) {
	return a.expenseService().CreateEntryFromBankCandidate(bankExpenseID, categoryID)
}

func createExpenseFromBankCandidate(a *App, bankExpenseID, categoryID string) (ExpenseEntry, error) {
	if err := requireExpenseCreate(a); err != nil {
		return ExpenseEntry{}, err
	}
	if a.db == nil {
		return ExpenseEntry{}, fmt.Errorf("database not initialized")
	}

	var bankExpense BankExpenseEntry
	if err := a.db.First(&bankExpense, "id = ?", strings.TrimSpace(bankExpenseID)).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("bank expense candidate not found: %w", err)
	}

	var category ExpenseCategory
	if strings.TrimSpace(categoryID) != "" {
		if err := a.db.First(&category, "id = ?", strings.TrimSpace(categoryID)).Error; err != nil {
			return ExpenseEntry{}, fmt.Errorf("expense category not found: %w", err)
		}
	} else {
		if err := a.db.Where("LOWER(code) = ?", strings.ToLower(defaultCategoryCodeForBank(bankExpense.Category))).First(&category).Error; err != nil {
			return ExpenseEntry{}, fmt.Errorf("default category not found for bank expense: %w", err)
		}
	}

	entry, err := a.CreateExpenseEntry(ExpenseEntry{
		ExpenseDate:        bankExpense.ExpenseDate,
		DueDate:            &bankExpense.ExpenseDate,
		Description:        bankExpense.Description,
		CategoryID:         category.ID,
		SourceType:         "bank_import",
		BankExpenseEntryID: &bankExpense.ID,
		Currency:           bankExpense.Currency,
		Amount:             bankExpense.Amount,
		VATAmount:          bankExpense.VATAmount,
		Division:           normalizeDivisionName(bankExpense.Division),
		Notes:              fmt.Sprintf("Imported from bank expense candidate (%s)", bankExpense.Category),
	})
	if err != nil {
		return ExpenseEntry{}, err
	}

	if bankExpense.BankStatementLineID != "" {
		// Mission I (I-16): a lost link here silently orphans the bank line from
		// its expense — log instead of discarding the error.
		if err := a.db.Model(&BankStatementLine{}).Where("id = ?", bankExpense.BankStatementLineID).
			Update("matched_expense_id", entry.ID).Error; err != nil {
			log.Printf("⚠️ Failed to link bank statement line %s to expense %s: %v", bankExpense.BankStatementLineID, entry.ID, err)
		}
	}

	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "bank_candidate", "action": "import", "id": bankExpense.ID})
	return entry, nil
}

func prepareExpenseEntry(a *App, entry ExpenseEntry, preserveIdentity bool) (ExpenseEntry, error) {
	entry.Description = strings.TrimSpace(entry.Description)
	if entry.Description == "" {
		return ExpenseEntry{}, fmt.Errorf("expense description is required")
	}
	entry.CategoryID = strings.TrimSpace(entry.CategoryID)
	if entry.CategoryID == "" {
		return ExpenseEntry{}, fmt.Errorf("expense category is required")
	}
	if entry.Amount < 0 || entry.VATAmount < 0 {
		return ExpenseEntry{}, fmt.Errorf("expense amounts cannot be negative")
	}
	if entry.Currency == "" {
		entry.Currency = "BHD"
	}
	if entry.ExpenseDate.IsZero() {
		entry.ExpenseDate = time.Now()
	}
	if entry.DueDate == nil {
		entry.DueDate = &entry.ExpenseDate
	}
	if entry.OrderID != nil && strings.TrimSpace(*entry.OrderID) != "" && strings.TrimSpace(entry.Division) == "" {
		entry.Division = a.resolveOrderDivision(*entry.OrderID)
	}
	if entry.BankAccountID != nil && strings.TrimSpace(*entry.BankAccountID) != "" && strings.TrimSpace(entry.Division) == "" {
		entry.Division = a.resolveBankAccountDivision(*entry.BankAccountID)
	}
	entry.Division = normalizeDivisionName(entry.Division)
	if entry.Status == "" {
		entry.Status = "draft"
	}
	if entry.PaymentStatus == "" {
		entry.PaymentStatus = "unpaid"
	}
	entry.TotalAmount = entry.Amount + entry.VATAmount
	if !preserveIdentity && entry.EntryNumber == "" {
		number, err := generateExpenseEntryNumber(a)
		if err != nil {
			return ExpenseEntry{}, err
		}
		entry.EntryNumber = number
	}
	if !preserveIdentity {
		entry.CreatedBy = a.getCurrentUserID()
	}

	var category ExpenseCategory
	if err := a.db.First(&category, "id = ?", entry.CategoryID).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("expense category not found: %w", err)
	}
	if entry.VATAmount == 0 && category.DefaultTaxRate > 0 {
		entry.VATAmount = entry.Amount * (category.DefaultTaxRate / 100)
		entry.TotalAmount = entry.Amount + entry.VATAmount
	}
	return entry, nil
}

func transitionExpenseEntry(a *App, entryID, nextStatus, notes string, mutate func(entry *ExpenseEntry, actor string, now time.Time)) (ExpenseEntry, error) {
	if err := requireExpenseUpdate(a); err != nil {
		return ExpenseEntry{}, err
	}
	if a.db == nil {
		return ExpenseEntry{}, fmt.Errorf("database not initialized")
	}

	var entry ExpenseEntry
	if err := a.db.First(&entry, "id = ?", strings.TrimSpace(entryID)).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("expense entry not found: %w", err)
	}

	actor := getExpenseActorID(a)

	// Segregation of duties: an expense creator may not approve their own
	// expense. Mirrors ApproveSupplierInvoice's SoD guard exactly — keyed
	// strictly on the approve transition so submit and reject stay unblocked.
	if nextStatus == "approved" {
		if entry.CreatedBy == "" {
			return ExpenseEntry{}, fmt.Errorf("segregation of duties: expense entry %s has no creator recorded — set CreatedBy before approving", entry.ID)
		}
		if entry.CreatedBy == actor {
			return ExpenseEntry{}, fmt.Errorf("segregation of duties: expense creator %s cannot approve their own expense", actor)
		}
	}

	now := time.Now()
	mutate(&entry, actor, now)
	entry.Status = nextStatus

	updates := map[string]any{
		"status":           entry.Status,
		"submitted_at":     entry.SubmittedAt,
		"submitted_by":     entry.SubmittedBy,
		"approved_at":      entry.ApprovedAt,
		"approved_by":      entry.ApprovedBy,
		"rejected_at":      entry.RejectedAt,
		"rejected_by":      entry.RejectedBy,
		"rejection_reason": entry.RejectionReason,
		"updated_at":       time.Now(),
	}
	if err := a.db.Model(&entry).Updates(updates).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("failed to update expense entry: %w", err)
	}

	recordExpenseApproval(a, entry.ID, nextStatus, notes)
	emitExpenseEvent(a, "expenses:updated", map[string]any{"entity": "entry", "action": nextStatus, "id": entry.ID})
	return decorateExpenseEntry(a, entry), nil
}

func recordExpenseApproval(a *App, entryID, action, notes string) {
	if a.db == nil || strings.TrimSpace(entryID) == "" {
		return
	}
	record := ExpenseApproval{
		Base:           Base{CreatedBy: a.getCurrentUserID()},
		ExpenseEntryID: entryID,
		Action:         action,
		ActorID:        getExpenseActorID(a),
		Notes:          strings.TrimSpace(notes),
		ActionAt:       time.Now(),
	}
	// Mission I (I-16): the approval audit row must not vanish silently.
	if err := a.db.Create(&record).Error; err != nil {
		log.Printf("⚠️ Failed to record expense approval audit row (entry %s, action %s): %v", entryID, action, err)
	}
}

func createExpenseEntryFromRecurring(a *App, item RecurringExpense, runAt time.Time) (ExpenseEntry, error) {
	entry, err := prepareExpenseEntry(a, ExpenseEntry{
		ExpenseDate:   runAt,
		DueDate:       &runAt,
		Description:   item.Name,
		CategoryID:    item.CategoryID,
		VendorID:      item.VendorID,
		SourceType:    "recurring",
		SourceRefID:   &item.ID,
		ProjectID:     item.ProjectID,
		CostCenter:    item.CostCenter,
		Currency:      item.Currency,
		Division:      normalizeDivisionName(item.Division),
		Amount:        item.DefaultAmount,
		VATAmount:     item.DefaultVATAmount,
		Status:        "draft",
		PaymentStatus: "unpaid",
		Notes:         item.Description,
	}, false)
	if err != nil {
		return ExpenseEntry{}, err
	}

	if item.AutoSubmit {
		now := time.Now()
		entry.Status = "submitted"
		entry.SubmittedAt = &now
		entry.SubmittedBy = getExpenseActorID(a)
	}

	if err := a.db.Create(&entry).Error; err != nil {
		return ExpenseEntry{}, fmt.Errorf("failed to generate recurring expense entry: %w", err)
	}
	recordExpenseApproval(a, entry.ID, "generated", item.Name)
	return decorateExpenseEntry(a, entry), nil
}

func postExpenseJournal(a *App, expense *ExpenseEntry, category ExpenseCategory) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	expenseAccount, err := ensureExpensePostingAccount(a, category)
	if err != nil {
		return "", err
	}
	creditAccount, err := a.ensureSupportingAccount(func() (string, string, string) {
		if expense.PaymentStatus == "paid" {
			return "1000", "Cash", "Asset"
		}
		return "2200", "Accrued Expenses", "Liability"
	}())
	if err != nil {
		return "", err
	}

	now := time.Now()
	journal := JournalEntry{
		Base:            Base{ID: uuid.New().String(), CreatedBy: a.getCurrentUserID(), CreatedAt: now, UpdatedAt: now},
		EntryNumber:     fmt.Sprintf("EXP-JE-%d-%04d", now.Year(), time.Now().UnixNano()%10000),
		EntryDate:       expense.ExpenseDate,
		Description:     fmt.Sprintf("Expense posting: %s", expense.Description),
		DebitTotal:      expense.TotalAmount,
		CreditTotal:     expense.TotalAmount,
		IsPosted:        true,
		PostedAt:        &now,
		PostedBy:        getExpenseActorID(a),
		FiscalYear:      expense.ExpenseDate.Year(),
		FiscalPeriod:    int(expense.ExpenseDate.Month()),
		SourceType:      "expense_entry",
		SourceID:        expense.ID,
		IsAutoGenerated: true,
	}

	lines := []JournalLine{
		{
			Base:        Base{ID: uuid.New().String(), CreatedBy: a.getCurrentUserID(), CreatedAt: now, UpdatedAt: now},
			EntryID:     journal.ID,
			AccountID:   expenseAccount.ID,
			AccountName: expenseAccount.AccountName,
			Debit:       expense.TotalAmount,
			Credit:      0,
			Description: expense.Description,
		},
		{
			Base:        Base{ID: uuid.New().String(), CreatedBy: a.getCurrentUserID(), CreatedAt: now, UpdatedAt: now},
			EntryID:     journal.ID,
			AccountID:   creditAccount.ID,
			AccountName: creditAccount.AccountName,
			Debit:       0,
			Credit:      expense.TotalAmount,
			Description: expense.Description,
		},
	}

	tx := a.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&journal).Error; err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to create expense journal entry: %w", err)
	}
	if err := tx.Create(&lines).Error; err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to create expense journal lines: %w", err)
	}

	for _, line := range lines {
		var account ChartOfAccount
		if err := tx.First(&account, "id = ?", line.AccountID).Error; err != nil {
			tx.Rollback()
			return "", fmt.Errorf("failed to load posting account: %w", err)
		}
		change := line.Credit - line.Debit
		if account.AccountType == "Asset" || account.AccountType == "Expense" {
			change = line.Debit - line.Credit
		}
		if err := tx.Model(&account).Update("balance", gorm.Expr("balance + ?", change)).Error; err != nil {
			tx.Rollback()
			return "", fmt.Errorf("failed to update account balance: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return "", fmt.Errorf("failed to commit expense posting: %w", err)
	}
	return journal.ID, nil
}

func ensureExpensePostingAccount(a *App, category ExpenseCategory) (ChartOfAccount, error) {
	if category.GLAccountID != nil && *category.GLAccountID != "" {
		var account ChartOfAccount
		if err := a.db.First(&account, "id = ?", *category.GLAccountID).Error; err == nil {
			return account, nil
		}
	}

	code, name := defaultExpenseAccountForCategory(category.Code, category.Name)
	return a.ensureSupportingAccount(code, name, "Expense")
}

func (a *App) ensureSupportingAccount(code, name, accountType string) (ChartOfAccount, error) {
	var account ChartOfAccount
	if err := a.db.Where("account_code = ?", code).First(&account).Error; err == nil {
		return account, nil
	}
	account = ChartOfAccount{
		Base:        Base{CreatedBy: "system"},
		AccountCode: code,
		AccountName: name,
		AccountType: accountType,
		IsActive:    true,
	}
	if err := a.db.Create(&account).Error; err != nil {
		return ChartOfAccount{}, fmt.Errorf("failed to ensure account %s: %w", code, err)
	}
	return account, nil
}

func generateExpenseEntryNumber(a *App) (string, error) {
	var count int64
	if err := a.db.Model(&ExpenseEntry{}).Count(&count).Error; err != nil {
		return "", fmt.Errorf("failed to generate expense entry number: %w", err)
	}
	return fmt.Sprintf("EXP-%d-%04d", time.Now().Year(), count+1), nil
}

func seedDefaultExpenseCategories(a *App) error {
	var count int64
	if err := a.db.Model(&ExpenseCategory{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count expense categories: %w", err)
	}
	if count > 0 {
		return nil
	}

	defaults := []struct {
		Name        string
		Code        string
		Description string
		AccountCode string
		SortOrder   int
	}{
		{"Office Rent", "RENT", "Office rent and lease costs", "6100", 10},
		{"Fleet Costs", "FLEET", "Vehicle, fuel, and delivery-related costs", "6500", 20},
		{"Employee Salaries", "PAYROLL", "Salaries, wages, and payroll overhead", "6000", 30},
		{"Utilities", "UTILITIES", "Electricity, water, telecom, and internet", "6200", 40},
		{"Office Supplies", "OFFICE", "Stationery, printing, and office consumables", "6300", 50},
		{"Software & Antivirus", "SOFTWARE", "Licensing, subscriptions, and antivirus", "6900", 60},
		{"Professional Fees", "PROFESSIONAL", "Auditors, consultants, and legal fees", "6600", 70},
		{"Bank Charges", "BANK", "Bank fees, swift charges, and card charges", "6800", 80},
		{"Miscellaneous", "MISC", "Other operating expenses", "6900", 90},
	}

	for _, item := range defaults {
		account, err := a.ensureSupportingAccount(item.AccountCode, accountNameForCode(item.AccountCode), "Expense")
		if err != nil {
			return err
		}
		accountID := account.ID
		category := ExpenseCategory{
			Base:        Base{CreatedBy: "system"},
			Name:        item.Name,
			Code:        item.Code,
			Description: item.Description,
			GLAccountID: &accountID,
			IsActive:    true,
			SortOrder:   item.SortOrder,
		}
		if err := a.db.Create(&category).Error; err != nil {
			return fmt.Errorf("failed to seed expense category %s: %w", item.Name, err)
		}
	}
	return nil
}

func decorateExpenseEntries(a *App, entries []ExpenseEntry) []ExpenseEntry {
	if len(entries) == 0 || a.db == nil {
		return entries
	}
	categoryIDs := make([]string, 0, len(entries))
	vendorIDs := make([]string, 0, len(entries))
	seenCategories := map[string]struct{}{}
	seenVendors := map[string]struct{}{}
	for _, entry := range entries {
		if entry.CategoryID != "" {
			if _, ok := seenCategories[entry.CategoryID]; !ok {
				seenCategories[entry.CategoryID] = struct{}{}
				categoryIDs = append(categoryIDs, entry.CategoryID)
			}
		}
		if entry.VendorID != nil && *entry.VendorID != "" {
			if _, ok := seenVendors[*entry.VendorID]; !ok {
				seenVendors[*entry.VendorID] = struct{}{}
				vendorIDs = append(vendorIDs, *entry.VendorID)
			}
		}
	}

	categoryNames := map[string]string{}
	vendorNames := map[string]string{}

	var categories []ExpenseCategory
	if len(categoryIDs) > 0 && a.db.Select("id", "name").Where("id IN ?", categoryIDs).Find(&categories).Error == nil {
		for _, category := range categories {
			categoryNames[category.ID] = category.Name
		}
	}
	var vendors []ExpenseVendor
	if len(vendorIDs) > 0 && a.db.Select("id", "name").Where("id IN ?", vendorIDs).Find(&vendors).Error == nil {
		for _, vendor := range vendors {
			vendorNames[vendor.ID] = vendor.Name
		}
	}

	for i := range entries {
		entries[i].CategoryName = categoryNames[entries[i].CategoryID]
		if entries[i].VendorID != nil {
			entries[i].VendorName = vendorNames[*entries[i].VendorID]
		}
	}
	return entries
}

func decorateExpenseEntry(a *App, entry ExpenseEntry) ExpenseEntry {
	rows := decorateExpenseEntries(a, []ExpenseEntry{entry})
	if len(rows) == 0 {
		return entry
	}
	return rows[0]
}

func decorateRecurringExpenses(a *App, items []RecurringExpense) []RecurringExpense {
	if len(items) == 0 || a.db == nil {
		return items
	}
	categoryIDs := make([]string, 0, len(items))
	vendorIDs := make([]string, 0, len(items))
	for _, item := range items {
		if item.CategoryID != "" {
			categoryIDs = append(categoryIDs, item.CategoryID)
		}
		if item.VendorID != nil && *item.VendorID != "" {
			vendorIDs = append(vendorIDs, *item.VendorID)
		}
	}
	categoryNames := map[string]string{}
	vendorNames := map[string]string{}

	var categories []ExpenseCategory
	if len(categoryIDs) > 0 && a.db.Select("id", "name").Where("id IN ?", categoryIDs).Find(&categories).Error == nil {
		for _, category := range categories {
			categoryNames[category.ID] = category.Name
		}
	}
	var vendors []ExpenseVendor
	if len(vendorIDs) > 0 && a.db.Select("id", "name").Where("id IN ?", vendorIDs).Find(&vendors).Error == nil {
		for _, vendor := range vendors {
			vendorNames[vendor.ID] = vendor.Name
		}
	}

	for i := range items {
		items[i].CategoryName = categoryNames[items[i].CategoryID]
		if items[i].VendorID != nil {
			items[i].VendorName = vendorNames[*items[i].VendorID]
		}
	}
	return items
}

func decorateRecurringExpense(a *App, item RecurringExpense) RecurringExpense {
	items := decorateRecurringExpenses(a, []RecurringExpense{item})
	if len(items) == 0 {
		return item
	}
	return items[0]
}

func lookupAccountNames(a *App, ids []string) map[string]string {
	names := map[string]string{}
	if len(ids) == 0 || a.db == nil {
		return names
	}
	var accounts []ChartOfAccount
	if err := a.db.Select("id", "account_name").Where("id IN ?", ids).Find(&accounts).Error; err != nil {
		return names
	}
	for _, account := range accounts {
		names[account.ID] = account.AccountName
	}
	return names
}

func categoryAccountIDs(categories []ExpenseCategory) []string {
	ids := make([]string, 0, len(categories))
	seen := map[string]struct{}{}
	for _, category := range categories {
		if category.GLAccountID == nil || *category.GLAccountID == "" {
			continue
		}
		if _, ok := seen[*category.GLAccountID]; ok {
			continue
		}
		seen[*category.GLAccountID] = struct{}{}
		ids = append(ids, *category.GLAccountID)
	}
	return ids
}

func defaultExpenseAccountForCategory(code, name string) (string, string) {
	key := strings.ToUpper(strings.TrimSpace(code))
	switch key {
	case "RENT":
		return "6100", "Rent Expense"
	case "PAYROLL":
		return "6000", "Salaries & Wages"
	case "UTILITIES":
		return "6200", "Utilities"
	case "OFFICE":
		return "6300", "Office Supplies"
	case "BANK":
		return "6800", "Bank Charges"
	case "PROFESSIONAL":
		return "6600", "Professional Fees"
	case "FLEET":
		return "6500", "Travel & Entertainment"
	default:
		lowered := strings.ToLower(name)
		if strings.Contains(lowered, "rent") {
			return "6100", "Rent Expense"
		}
		if strings.Contains(lowered, "salary") || strings.Contains(lowered, "payroll") {
			return "6000", "Salaries & Wages"
		}
		if strings.Contains(lowered, "bank") || strings.Contains(lowered, "charge") {
			return "6800", "Bank Charges"
		}
		return "6900", "Miscellaneous Expense"
	}
}

func accountNameForCode(code string) string {
	switch code {
	case "1000":
		return "Cash"
	case "2200":
		return "Accrued Expenses"
	case "6000":
		return "Salaries & Wages"
	case "6100":
		return "Rent Expense"
	case "6200":
		return "Utilities"
	case "6300":
		return "Office Supplies"
	case "6500":
		return "Travel & Entertainment"
	case "6600":
		return "Professional Fees"
	case "6800":
		return "Bank Charges"
	default:
		return "Miscellaneous Expense"
	}
}

func defaultCategoryCodeForBank(category string) string {
	switch strings.ToUpper(strings.TrimSpace(category)) {
	case "BANK_FEE", "SWIFT_CHARGE", "BG_FEE":
		return "BANK"
	case "VAT":
		return "BANK"
	default:
		return "MISC"
	}
}

func nextRecurringDate(from time.Time, frequency string, interval int) time.Time {
	if interval <= 0 {
		interval = 1
	}
	switch strings.ToLower(strings.TrimSpace(frequency)) {
	case "weekly":
		return from.AddDate(0, 0, 7*interval)
	case "quarterly":
		return from.AddDate(0, 3*interval, 0)
	case "yearly":
		return from.AddDate(interval, 0, 0)
	default:
		return from.AddDate(0, interval, 0)
	}
}

func getExpenseActorID(a *App) string {
	if current, err := a.GetCurrentEmployeeContext(); err == nil && current.EmployeeID != "" {
		return current.EmployeeID
	}
	if userID := a.getCurrentUserID(); strings.TrimSpace(userID) != "" {
		return userID
	}
	return "system"
}

func emitExpenseEvent(a *App, name string, payload any) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, payload)
}
