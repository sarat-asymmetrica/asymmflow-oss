package cheque

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/pkg/finance"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "cheque.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&finance.ChequeRegister{}, &finance.OutstandingCheque{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func TestCreateRegister_RefusesOverlapAndBadRange(t *testing.T) {
	svc := New(testDB(t))

	if _, err := svc.CreateRegister("acc-1", "BOOK-1", 100, 100); err == nil {
		t.Fatal("expected start >= end to be refused")
	}
	if _, err := svc.CreateRegister("acc-1", "BOOK-1", 100, 150); err != nil {
		t.Fatalf("create register: %v", err)
	}
	if _, err := svc.CreateRegister("acc-1", "BOOK-2", 140, 200); err == nil {
		t.Fatal("expected overlapping range to be refused")
	}
	// Different account may reuse the range.
	if _, err := svc.CreateRegister("acc-2", "BOOK-3", 100, 150); err != nil {
		t.Fatalf("other account register: %v", err)
	}
}

func TestIssue_AllocatesSequentiallyAndExhausts(t *testing.T) {
	svc := New(testDB(t))
	if _, err := svc.CreateRegister("acc-1", "BOOK-1", 1, 2); err != nil {
		t.Fatalf("create register: %v", err)
	}

	next, err := svc.NextNumber("acc-1")
	if err != nil || next != "000001" {
		t.Fatalf("next number: %q %v", next, err)
	}

	first, err := svc.Issue("acc-1", 100.500, "Al Manar Trading", "SUPPLIER", nil, "Materials")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if first.ChequeNumber != "000001" || first.Status != "ISSUED" || first.Currency != "BHD" {
		t.Fatalf("unexpected cheque: %+v", first)
	}

	second, err := svc.Issue("acc-1", 25.000, "Utility Co", "VENDOR", nil, "Utilities")
	if err != nil {
		t.Fatalf("issue second: %v", err)
	}
	if second.ChequeNumber != "000002" {
		t.Fatalf("expected sequential number 000002, got %s", second.ChequeNumber)
	}

	// Book of 2 is now exhausted — the register flips and further issuance fails.
	if _, err := svc.Issue("acc-1", 1, "X", "VENDOR", nil, "y"); err == nil {
		t.Fatal("expected exhausted book to refuse issuance")
	}
	regs, _ := svc.Registers("acc-1")
	if len(regs) != 1 || regs[0].Status != "EXHAUSTED" || regs[0].ExhaustedDate == nil {
		t.Fatalf("expected exhausted register, got %+v", regs)
	}
}

func TestLifecycle_Transitions(t *testing.T) {
	svc := New(testDB(t))
	if _, err := svc.CreateRegister("acc-1", "BOOK-1", 1, 100); err != nil {
		t.Fatalf("create register: %v", err)
	}
	c, err := svc.Issue("acc-1", 50, "Payee", "SUPPLIER", nil, "Goods")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if err := svc.MarkPresented(c.ChequeNumber); err != nil {
		t.Fatalf("presented: %v", err)
	}
	if err := svc.MarkPresented(c.ChequeNumber); err == nil {
		t.Fatal("expected double-present to be refused")
	}
	if err := svc.MarkCleared(c.ChequeNumber, "line-1", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("cleared: %v", err)
	}
	if err := svc.MarkCleared(c.ChequeNumber, "line-2", time.Now()); err == nil {
		t.Fatal("expected already-cleared to be refused")
	}
	got, err := svc.ByNumber(c.ChequeNumber)
	if err != nil || got.Status != "CLEARED" || got.MatchedLineID == nil {
		t.Fatalf("unexpected cleared state: %+v err %v", got, err)
	}

	// A cleared cheque cannot be cancelled, staled, or bounced.
	if err := svc.Cancel(c.ChequeNumber, "nope"); err == nil {
		t.Fatal("expected cancel of cleared cheque to be refused")
	}
	if err := svc.MarkStale(c.ChequeNumber); err == nil {
		t.Fatal("expected stale of cleared cheque to be refused")
	}
	if err := svc.MarkBounced(c.ChequeNumber, "nope"); err == nil {
		t.Fatal("expected bounce of cleared cheque to be refused")
	}
}

func TestReissue_OnlyStaleOrCancelled(t *testing.T) {
	svc := New(testDB(t))
	if _, err := svc.CreateRegister("acc-1", "BOOK-1", 1, 100); err != nil {
		t.Fatalf("create register: %v", err)
	}
	c, err := svc.Issue("acc-1", 75.250, "Payee", "SUPPLIER", nil, "Goods")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := svc.Reissue(c.ChequeNumber, "acc-1"); err == nil {
		t.Fatal("expected reissue of ISSUED cheque to be refused")
	}
	if err := svc.Cancel(c.ChequeNumber, "signature error"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	replacement, err := svc.Reissue(c.ChequeNumber, "acc-1")
	if err != nil {
		t.Fatalf("reissue: %v", err)
	}
	if replacement.Amount != 75.250 || replacement.PayeeName != "Payee" {
		t.Fatalf("reissue must carry amount and payee: %+v", replacement)
	}
	old, _ := svc.ByNumber(c.ChequeNumber)
	if old.ReissuedAs == nil || *old.ReissuedAs != replacement.ChequeNumber {
		t.Fatalf("old cheque must reference replacement: %+v", old)
	}
}

func TestOutstanding_TotalsIssuedAndPresented(t *testing.T) {
	svc := New(testDB(t))
	if _, err := svc.CreateRegister("acc-1", "BOOK-1", 1, 100); err != nil {
		t.Fatalf("create register: %v", err)
	}
	a, _ := svc.Issue("acc-1", 10.000, "A", "SUPPLIER", nil, "x")
	b, _ := svc.Issue("acc-1", 20.000, "B", "SUPPLIER", nil, "y")
	c, _ := svc.Issue("acc-1", 40.000, "C", "SUPPLIER", nil, "z")
	if err := svc.MarkPresented(b.ChequeNumber); err != nil {
		t.Fatalf("presented: %v", err)
	}
	if err := svc.MarkCleared(c.ChequeNumber, "line-9", time.Now()); err != nil {
		t.Fatalf("cleared: %v", err)
	}

	result, err := svc.Outstanding("acc-1")
	if err != nil {
		t.Fatalf("outstanding: %v", err)
	}
	if len(result.Cheques) != 2 || result.Total != 30.000 {
		t.Fatalf("expected 2 outstanding totalling 30.000, got %d totalling %v", len(result.Cheques), result.Total)
	}
	_ = a

	report, err := svc.Report("acc-1", time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if report["total_cheques"].(int) != 3 || report["outstanding_total"].(float64) != 30.000 {
		t.Fatalf("unexpected report: %+v", report)
	}
}
