package banking

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/ncruces/go-sqlite3/embed"
	"ph_holdings_app/pkg/finance"
)

func TestNormalizeStatementStatus(t *testing.T) {
	for raw, want := range map[string]string{
		"imported":    StatementStatusImported,
		"In Progress": StatementStatusInProgress,
		"inprogress":  StatementStatusInProgress,
		"RECONCILED":  StatementStatusReconciled,
		" Verified ":  StatementStatusVerified,
		"cancelled":   StatementStatusCancelled,
	} {
		got, err := NormalizeStatementStatus(raw)
		if err != nil || got != want {
			t.Fatalf("normalize %q: got %q, %v", raw, got, err)
		}
	}
	if _, err := NormalizeStatementStatus(""); err == nil {
		t.Fatal("blank status must be rejected")
	}
	if _, err := NormalizeStatementStatus("Done"); err == nil {
		t.Fatal("unknown status must be rejected")
	}
}

func TestNormalizeStatementStatusUpdate_RefusesDirectFinal(t *testing.T) {
	updates := map[string]any{"status": "reconciled"}
	if err := normalizeStatementStatusUpdate(updates); err == nil || !strings.Contains(err.Error(), "reconciliation workflow") {
		t.Fatalf("direct final status must be refused, got %v", err)
	}
	updates = map[string]any{"status": "in progress"}
	if err := normalizeStatementStatusUpdate(updates); err != nil || updates["status"] != StatementStatusInProgress {
		t.Fatalf("editable status must canonicalize, got %v / %v", updates["status"], err)
	}
}

// PC-D4: a finalized statement refuses mutation until reopened — it no longer
// silently auto-reverts to InProgress.
func TestEnsureStatementMutableTx(t *testing.T) {
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "banking.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&finance.BankStatement{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})

	open := finance.BankStatement{StatementNumber: "ST-OPEN-1", Status: StatementStatusInProgress}
	final := finance.BankStatement{StatementNumber: "ST-FINAL-1", Status: StatementStatusReconciled}
	if err := db.Create(&open).Error; err != nil {
		t.Fatalf("seed open: %v", err)
	}
	if err := db.Create(&final).Error; err != nil {
		t.Fatalf("seed final: %v", err)
	}

	if err := ensureStatementMutableTx(db, open.ID, "edit statement"); err != nil {
		t.Fatalf("open statement must be mutable, got %v", err)
	}
	err = ensureStatementMutableTx(db, final.ID, "edit statement lines")
	if err == nil || !strings.Contains(err.Error(), "reopen the reconciliation first") {
		t.Fatalf("finalized statement must refuse, got %v", err)
	}
}
