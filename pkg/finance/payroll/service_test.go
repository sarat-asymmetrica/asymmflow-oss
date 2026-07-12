package payroll

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/pkg/finance"
)

type fakeIdentity struct{}

func (fakeIdentity) UserID() string      { return "user-1" }
func (fakeIdentity) ActorID() string     { return "emp-actor" }
func (fakeIdentity) DisplayName() string { return "Test Operator" }

type fakeDirectory struct{ refs map[string]EmployeeRef }

func (f fakeDirectory) Employees(ids []string) map[string]EmployeeRef {
	out := map[string]EmployeeRef{}
	for _, id := range ids {
		if ref, ok := f.refs[id]; ok {
			out[id] = ref
		}
	}
	return out
}

type fakeEvents struct{ names []string }

func (f *fakeEvents) Emit(name string, payload map[string]any) {
	f.names = append(f.names, name)
}

type fakeExpenses struct{ synced int }

func (f *fakeExpenses) SyncRunExpense(tx *gorm.DB, run *Run) error { f.synced++; return nil }
func (f *fakeExpenses) EmitExpenseUpdated(runID string)            {}

func testService(t *testing.T, refs map[string]EmployeeRef) (*Service, *fakeExpenses) {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "payroll.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&CompensationProfile{}, &Period{}, &Run{}, &RunItem{}, &Component{}, &Payout{},
		&finance.ChartOfAccount{}, &finance.JournalEntry{}, &finance.JournalLine{},
		&finance.CompanyBankAccount{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	expenses := &fakeExpenses{}
	return New(db, fakeIdentity{}, fakeDirectory{refs: refs}, &fakeEvents{}, expenses), expenses
}

func seedPeriod(t *testing.T, svc *Service) Period {
	t.Helper()
	period, err := svc.CreatePeriod(Period{
		PeriodStart: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create period: %v", err)
	}
	return period
}

func TestGenerateRun_SkipsInactiveEmployees(t *testing.T) {
	svc, _ := testService(t, map[string]EmployeeRef{
		"emp-1": {ID: "emp-1", FullName: "Active", IsActive: true},
		"emp-2": {ID: "emp-2", FullName: "Gone", IsActive: false},
	})
	period := seedPeriod(t, svc)

	for _, id := range []string{"emp-1", "emp-2"} {
		if _, err := svc.UpsertProfile(CompensationProfile{EmployeeID: id, BaseSalary: 500}); err != nil {
			t.Fatalf("profile %s: %v", id, err)
		}
	}

	run, err := svc.GenerateRun(period.ID)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if run.TotalEmployees != 1 || run.GrossTotal != 500 {
		t.Fatalf("inactive employee must be skipped: %+v", run)
	}
}

func TestGenerateRun_RefusesDeductionsExceedingGross(t *testing.T) {
	svc, _ := testService(t, map[string]EmployeeRef{
		"emp-1": {ID: "emp-1", FullName: "Over Deducted", IsActive: true},
		"emp-2": {ID: "emp-2", FullName: "Healthy", IsActive: true},
	})
	period := seedPeriod(t, svc)

	if _, err := svc.UpsertProfile(CompensationProfile{EmployeeID: "emp-1", BaseSalary: 100, StandardDeduction: 120, TaxDeduction: 30}); err != nil {
		t.Fatalf("profile emp-1: %v", err)
	}
	if _, err := svc.UpsertProfile(CompensationProfile{EmployeeID: "emp-2", BaseSalary: 500}); err != nil {
		t.Fatalf("profile emp-2: %v", err)
	}

	_, err := svc.GenerateRun(period.ID)
	if err == nil {
		t.Fatal("deductions exceeding gross must refuse the whole run")
	}
	for _, want := range []string{"Over Deducted", "150.000", "100.000"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal error must contain %q, got: %v", want, err)
		}
	}

	var count int64
	if e := svc.db.Model(&Run{}).Where("payroll_period_id = ?", period.ID).Count(&count).Error; e != nil {
		t.Fatalf("count runs: %v", e)
	}
	if count != 0 {
		t.Fatalf("refused generation must persist nothing, found %d runs", count)
	}
}

func TestUpsertProfile_UnknownEmployeeRefused(t *testing.T) {
	svc, _ := testService(t, map[string]EmployeeRef{})
	if _, err := svc.UpsertProfile(CompensationProfile{EmployeeID: "ghost", BaseSalary: 1}); err == nil {
		t.Fatal("unknown employee must be refused")
	}
}

func TestLifecycle_PostRefusesUnapprovedAndSyncsExpense(t *testing.T) {
	svc, expenses := testService(t, map[string]EmployeeRef{
		"emp-1": {ID: "emp-1", FullName: "Aisha", IsActive: true},
	})
	period := seedPeriod(t, svc)
	if _, err := svc.UpsertProfile(CompensationProfile{EmployeeID: "emp-1", BaseSalary: 1000, StandardDeduction: 100, EmployerCost: 50}); err != nil {
		t.Fatalf("profile: %v", err)
	}

	run, err := svc.GenerateRun(period.ID)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := svc.PostRun(run.ID); err == nil {
		t.Fatal("posting a draft run must be refused")
	}

	run, err = svc.ApproveRun(run.ID, "ok")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if run.Status != "approved" {
		t.Fatalf("expected approved, got %s", run.Status)
	}
	// Second approval must be refused (only draft runs approve).
	if _, err := svc.ApproveRun(run.ID, "again"); err == nil {
		t.Fatal("double approval must be refused")
	}

	run, err = svc.PostRun(run.ID)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if run.Status != "posted" || run.JournalEntryID == nil {
		t.Fatalf("expected posted run with journal: %+v", run)
	}
	if expenses.synced != 1 {
		t.Fatalf("expense bridge must sync once on post, got %d", expenses.synced)
	}

	run, err = svc.MarkPaid(run.ID, "2026-06-30", "TRF-1", "")
	if err != nil {
		t.Fatalf("mark paid: %v", err)
	}
	if run.Status != "paid" || run.PayoutJournalEntryID == nil {
		t.Fatalf("expected paid run with payout journal: %+v", run)
	}
	if expenses.synced != 2 {
		t.Fatalf("expense bridge must sync again on paid, got %d", expenses.synced)
	}
}
