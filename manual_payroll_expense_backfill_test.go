//go:build manual

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestManualBackfillPayrollExpenseEntries(t *testing.T) {
	if os.Getenv("PAYROLL_EXPENSE_BACKFILL_COMMIT") != "1" {
		t.Skip("set PAYROLL_EXPENSE_BACKFILL_COMMIT=1 to backfill payroll expense entries in the live runtime DB")
	}

	dbPath := filepath.Join(os.Getenv("HOME"), ".local", "share", "AsymmFlow", "ph_holdings.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open runtime database: %v", err)
	}

	app := &App{
		db:                     db,
		cache:                  NewCache(),
		startupImporting:       true,
		startupImportStartTime: time.Now(),
		currentUserID:          "manual-payroll-backfill",
		currentUser:            &User{Base: Base{ID: "manual-payroll-backfill"}, Username: "manual-payroll-backfill"},
	}
	t.Cleanup(app.cache.Stop)

	if err := app.EnsureExpenseFoundation(); err != nil {
		t.Fatalf("failed to ensure expense foundation: %v", err)
	}
	if err := app.ensurePayrollFoundationInternal(); err != nil {
		t.Fatalf("failed to ensure payroll foundation: %v", err)
	}

	var runs []PayrollRun
	if err := db.Where("status IN ?", []string{"posted", "paid"}).Order("updated_at ASC").Find(&runs).Error; err != nil {
		t.Fatalf("failed to load payroll runs: %v", err)
	}

	for _, run := range runs {
		tx := db.Begin()
		if err := app.syncPayrollRunExpenseEntry(tx, &run); err != nil {
			tx.Rollback()
			t.Fatalf("failed to sync payroll expense for %s: %v", run.RunNumber, err)
		}
		if err := tx.Commit().Error; err != nil {
			t.Fatalf("failed to commit payroll expense sync for %s: %v", run.RunNumber, err)
		}
	}

	t.Logf("backfilled payroll expense entries for %d posted/paid payroll runs", len(runs))
}
