// Expense deletions (categories, vendors, entries), moved from the
// trading root in Wave 6 (Mission A.2). The host keeps the delete guard,
// RBAC check, and UI event emission; the usage checks and the
// posted/paid immutability rules live here.
package expense

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/finance"
)

// DeleteCategory removes an expense category unless entries or recurring
// schedules still reference it.
func DeleteCategory(db *gorm.DB, categoryID string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	categoryID = strings.TrimSpace(categoryID)
	if categoryID == "" {
		return fmt.Errorf("expense category id is required")
	}

	var category finance.ExpenseCategory
	if err := db.First(&category, "id = ?", categoryID).Error; err != nil {
		return fmt.Errorf("expense category not found: %w", err)
	}

	var entryCount int64
	if err := db.Model(&finance.ExpenseEntry{}).Where("category_id = ?", categoryID).Count(&entryCount).Error; err != nil {
		return fmt.Errorf("failed to check expense category usage: %w", err)
	}

	var recurringCount int64
	if err := db.Model(&finance.RecurringExpense{}).Where("category_id = ?", categoryID).Count(&recurringCount).Error; err != nil {
		return fmt.Errorf("failed to check recurring expense usage: %w", err)
	}

	if entryCount > 0 || recurringCount > 0 {
		return fmt.Errorf("cannot delete expense category %q because it is used by %s", category.Name, usageDescription(entryCount, recurringCount))
	}

	if err := db.Delete(&finance.ExpenseCategory{}, "id = ?", categoryID).Error; err != nil {
		return fmt.Errorf("failed to delete expense category: %w", err)
	}
	return nil
}

// DeleteVendor removes an expense vendor unless entries or recurring
// schedules still reference it.
func DeleteVendor(db *gorm.DB, vendorID string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	vendorID = strings.TrimSpace(vendorID)
	if vendorID == "" {
		return fmt.Errorf("expense vendor id is required")
	}

	var vendor finance.ExpenseVendor
	if err := db.First(&vendor, "id = ?", vendorID).Error; err != nil {
		return fmt.Errorf("expense vendor not found: %w", err)
	}

	var entryCount int64
	if err := db.Model(&finance.ExpenseEntry{}).Where("vendor_id = ?", vendorID).Count(&entryCount).Error; err != nil {
		return fmt.Errorf("failed to check expense vendor usage: %w", err)
	}

	var recurringCount int64
	if err := db.Model(&finance.RecurringExpense{}).Where("vendor_id = ?", vendorID).Count(&recurringCount).Error; err != nil {
		return fmt.Errorf("failed to check recurring vendor usage: %w", err)
	}

	if entryCount > 0 || recurringCount > 0 {
		return fmt.Errorf("cannot delete expense vendor %q because it is used by %s", vendor.Name, usageDescription(entryCount, recurringCount))
	}

	if err := db.Delete(&finance.ExpenseVendor{}, "id = ?", vendorID).Error; err != nil {
		return fmt.Errorf("failed to delete expense vendor: %w", err)
	}
	return nil
}

// DeleteEntry removes an expense entry. Posted or paid entries — and
// anything already carrying a journal entry — are immutable.
func DeleteEntry(db *gorm.DB, entryID string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	entryID = strings.TrimSpace(entryID)
	if entryID == "" {
		return fmt.Errorf("expense entry id is required")
	}

	var entry finance.ExpenseEntry
	if err := db.First(&entry, "id = ?", entryID).Error; err != nil {
		return fmt.Errorf("expense entry not found: %w", err)
	}

	if entry.Status == "posted" || entry.Status == "paid" || entry.PaymentStatus == "paid" {
		return fmt.Errorf("posted or paid expenses cannot be deleted")
	}
	if entry.JournalEntryID != nil && strings.TrimSpace(*entry.JournalEntryID) != "" {
		return fmt.Errorf("posted expenses with journal entries cannot be deleted")
	}

	if err := db.Delete(&entry).Error; err != nil {
		return fmt.Errorf("failed to delete expense entry: %w", err)
	}
	return nil
}

// usageDescription phrases the blocking references exactly as the host
// historically did ("2 expense entries and 1 recurring schedule").
func usageDescription(entryCount, recurringCount int64) string {
	usageParts := make([]string, 0, 2)
	if entryCount > 0 {
		label := "expense entries"
		if entryCount == 1 {
			label = "expense entry"
		}
		usageParts = append(usageParts, fmt.Sprintf("%d %s", entryCount, label))
	}
	if recurringCount > 0 {
		label := "recurring schedules"
		if recurringCount == 1 {
			label = "recurring schedule"
		}
		usageParts = append(usageParts, fmt.Sprintf("%d %s", recurringCount, label))
	}
	return strings.Join(usageParts, " and ")
}
