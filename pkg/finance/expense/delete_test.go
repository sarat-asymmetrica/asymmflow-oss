package expense

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/ncruces/go-sqlite3/embed"
	"ph_holdings_app/pkg/finance"
)

func deleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "expense.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&finance.ExpenseCategory{}, &finance.ExpenseVendor{},
		&finance.ExpenseEntry{}, &finance.RecurringExpense{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func TestDeleteCategory_BlockedWhileReferenced(t *testing.T) {
	db := deleteTestDB(t)
	category := finance.ExpenseCategory{Name: "Utilities"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("seed category: %v", err)
	}
	entry := finance.ExpenseEntry{CategoryID: category.ID, ExpenseDate: time.Now(), Amount: 10, TotalAmount: 10}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	err := DeleteCategory(db, category.ID)
	if err == nil || !strings.Contains(err.Error(), `cannot delete expense category "Utilities" because it is used by 1 expense entry`) {
		t.Fatalf("referenced category must be blocked with usage phrasing, got %v", err)
	}

	if err := DeleteEntry(db, entry.ID); err != nil {
		t.Fatalf("delete draft entry: %v", err)
	}
	if err := DeleteCategory(db, category.ID); err != nil {
		t.Fatalf("unreferenced category must delete, got %v", err)
	}
}

func TestDeleteEntry_PostedAndPaidAreImmutable(t *testing.T) {
	db := deleteTestDB(t)
	journalID := "JE-1"
	cases := []finance.ExpenseEntry{
		{EntryNumber: "EXP-1", ExpenseDate: time.Now(), Status: "posted"},
		{EntryNumber: "EXP-2", ExpenseDate: time.Now(), Status: "draft", PaymentStatus: "paid"},
		{EntryNumber: "EXP-3", ExpenseDate: time.Now(), Status: "draft", JournalEntryID: &journalID},
	}
	for i := range cases {
		if err := db.Create(&cases[i]).Error; err != nil {
			t.Fatalf("seed entry %d: %v", i, err)
		}
		if err := DeleteEntry(db, cases[i].ID); err == nil {
			t.Fatalf("entry %d must be immutable", i)
		}
	}

	if err := DeleteEntry(db, "   "); err == nil || !strings.Contains(err.Error(), "id is required") {
		t.Fatalf("blank id must be refused, got %v", err)
	}
}
