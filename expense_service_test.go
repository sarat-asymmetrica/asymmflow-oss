package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeleteExpenseCategoryDeletesUnusedCategory(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureExpenseFoundation())

	category, err := app.CreateExpenseCategory(ExpenseCategory{
		Name: "Temporary Cleanup Category",
		Code: "TEMP_DELETE",
	})
	require.NoError(t, err)

	require.NoError(t, app.DeleteExpenseCategory(category.ID))

	var activeCount int64
	require.NoError(t, app.db.Model(&ExpenseCategory{}).Where("id = ?", category.ID).Count(&activeCount).Error)
	require.Zero(t, activeCount)

	var deletedCount int64
	require.NoError(t, app.db.Unscoped().Model(&ExpenseCategory{}).Where("id = ?", category.ID).Count(&deletedCount).Error)
	require.Equal(t, int64(1), deletedCount)
}

func TestDeleteExpenseCategoryBlocksUsedCategory(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureExpenseFoundation())

	category, err := app.CreateExpenseCategory(ExpenseCategory{
		Name: "Protected Operating Expense",
		Code: "PROTECTED_OPERATING",
	})
	require.NoError(t, err)

	_, err = app.CreateExpenseEntry(ExpenseEntry{
		Description: "Expense already posted against this category",
		CategoryID:  category.ID,
		Amount:      125,
		VATAmount:   12.5,
		ExpenseDate: time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	_, err = app.CreateRecurringExpense(RecurringExpense{
		Name:        "Monthly protected schedule",
		CategoryID:  category.ID,
		NextRunDate: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	err = app.DeleteExpenseCategory(category.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "1 expense entry")
	require.Contains(t, err.Error(), "1 recurring schedule")

	var activeCount int64
	require.NoError(t, app.db.Model(&ExpenseCategory{}).Where("id = ?", category.ID).Count(&activeCount).Error)
	require.Equal(t, int64(1), activeCount)
}

func TestDeleteExpenseVendorDeletesUnusedVendor(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureExpenseFoundation())

	vendor, err := app.CreateExpenseVendor(ExpenseVendor{
		Name: "Temporary Cleanup Vendor",
	})
	require.NoError(t, err)

	require.NoError(t, app.DeleteExpenseVendor(vendor.ID))

	var activeCount int64
	require.NoError(t, app.db.Model(&ExpenseVendor{}).Where("id = ?", vendor.ID).Count(&activeCount).Error)
	require.Zero(t, activeCount)

	var deletedCount int64
	require.NoError(t, app.db.Unscoped().Model(&ExpenseVendor{}).Where("id = ?", vendor.ID).Count(&deletedCount).Error)
	require.Equal(t, int64(1), deletedCount)
}

func TestDeleteExpenseVendorBlocksUsedVendor(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureExpenseFoundation())

	category, err := app.CreateExpenseCategory(ExpenseCategory{
		Name: "Vendor Protection Category",
		Code: "VENDOR_PROTECT",
	})
	require.NoError(t, err)

	vendor, err := app.CreateExpenseVendor(ExpenseVendor{
		Name: "Protected Vendor",
	})
	require.NoError(t, err)

	_, err = app.CreateExpenseEntry(ExpenseEntry{
		Description: "Expense linked to vendor",
		CategoryID:  category.ID,
		VendorID:    &vendor.ID,
		Amount:      90,
		VATAmount:   9,
		ExpenseDate: time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	_, err = app.CreateRecurringExpense(RecurringExpense{
		Name:        "Recurring vendor-linked expense",
		CategoryID:  category.ID,
		VendorID:    &vendor.ID,
		NextRunDate: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	err = app.DeleteExpenseVendor(vendor.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "1 expense entry")
	require.Contains(t, err.Error(), "1 recurring schedule")

	var activeCount int64
	require.NoError(t, app.db.Model(&ExpenseVendor{}).Where("id = ?", vendor.ID).Count(&activeCount).Error)
	require.Equal(t, int64(1), activeCount)
}

func TestDeleteRecurringExpenseDeletesSchedule(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureExpenseFoundation())

	category, err := app.CreateExpenseCategory(ExpenseCategory{
		Name: "Recurring Cleanup Category",
		Code: "RECURRING_DELETE",
	})
	require.NoError(t, err)

	recurring, err := app.CreateRecurringExpense(RecurringExpense{
		Name:        "Delete Me Monthly",
		CategoryID:  category.ID,
		NextRunDate: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	require.NoError(t, app.DeleteRecurringExpense(recurring.ID))

	var activeCount int64
	require.NoError(t, app.db.Model(&RecurringExpense{}).Where("id = ?", recurring.ID).Count(&activeCount).Error)
	require.Zero(t, activeCount)

	var deletedCount int64
	require.NoError(t, app.db.Unscoped().Model(&RecurringExpense{}).Where("id = ?", recurring.ID).Count(&deletedCount).Error)
	require.Equal(t, int64(1), deletedCount)
}

func TestDeleteExpenseVendorSucceedsAfterRecurringScheduleRemoval(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureExpenseFoundation())

	category, err := app.CreateExpenseCategory(ExpenseCategory{
		Name: "Recurring Vendor Cleanup Category",
		Code: "RECUR_VENDOR_DELETE",
	})
	require.NoError(t, err)

	vendor, err := app.CreateExpenseVendor(ExpenseVendor{
		Name: "Recurring Cleanup Vendor",
	})
	require.NoError(t, err)

	recurring, err := app.CreateRecurringExpense(RecurringExpense{
		Name:        "Vendor-linked recurring cleanup",
		CategoryID:  category.ID,
		VendorID:    &vendor.ID,
		NextRunDate: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	err = app.DeleteExpenseVendor(vendor.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "1 recurring schedule")

	require.NoError(t, app.DeleteRecurringExpense(recurring.ID))
	require.NoError(t, app.DeleteExpenseVendor(vendor.ID))

	var activeCount int64
	require.NoError(t, app.db.Model(&ExpenseVendor{}).Where("id = ?", vendor.ID).Count(&activeCount).Error)
	require.Zero(t, activeCount)
}

func TestDeleteExpenseEntryDeletesUnpostedExpense(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureExpenseFoundation())

	category, err := app.CreateExpenseCategory(ExpenseCategory{
		Name: "Delete Entry Category",
		Code: "DELETE_ENTRY_CAT",
	})
	require.NoError(t, err)

	entry, err := app.CreateExpenseEntry(ExpenseEntry{
		Description: "Unposted deletable expense",
		CategoryID:  category.ID,
		Amount:      145.5,
		VATAmount:   14.5,
		ExpenseDate: time.Date(2026, time.January, 12, 0, 0, 0, 0, time.UTC),
		Status:      "draft",
	})
	require.NoError(t, err)

	require.NoError(t, app.DeleteExpenseEntry(entry.ID))

	var activeCount int64
	require.NoError(t, app.db.Model(&ExpenseEntry{}).Where("id = ?", entry.ID).Count(&activeCount).Error)
	require.Zero(t, activeCount)
}

func TestDeleteExpenseEntryBlocksPaidExpense(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.EnsureExpenseFoundation())

	category, err := app.CreateExpenseCategory(ExpenseCategory{
		Name: "Protected Paid Expense",
		Code: "PROTECTED_PAID_EXPENSE",
	})
	require.NoError(t, err)

	entry, err := app.CreateExpenseEntry(ExpenseEntry{
		Description:   "Paid expense cannot be deleted",
		CategoryID:    category.ID,
		Amount:        980,
		VATAmount:     98,
		ExpenseDate:   time.Date(2026, time.January, 20, 0, 0, 0, 0, time.UTC),
		Status:        "paid",
		PaymentStatus: "paid",
	})
	require.NoError(t, err)

	err = app.DeleteExpenseEntry(entry.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "posted or paid expenses cannot be deleted")

	var activeCount int64
	require.NoError(t, app.db.Model(&ExpenseEntry{}).Where("id = ?", entry.ID).Count(&activeCount).Error)
	require.Equal(t, int64(1), activeCount)
}
