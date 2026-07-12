package context

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeHost struct{}

func (fakeHost) WorkContext(intent Intent) map[string]any {
	return map[string]any{"stub": true}
}
func (fakeHost) ResolveEmployeeReference(reference string) *ButlerResolvedEntity { return nil }
func (fakeHost) EmployeeContext(resolution *ButlerResolvedEntity) map[string]any {
	return nil
}
func (fakeHost) RecentOpenQuickCaptures() []map[string]any { return nil }
func (fakeHost) CashflowProjectionContext() map[string]any { return nil }
func (fakeHost) OpenDedupedOpportunities() []Opportunity   { return nil }

func testService(t *testing.T) *Service {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "context.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&CustomerMaster{}, &SupplierMaster{}, &Invoice{}, &Order{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return New(db, fakeHost{})
}

func TestBusinessSummary_CountsAndRevenue(t *testing.T) {
	svc := testService(t)
	for _, c := range []CustomerMaster{
		{CustomerID: "C-1", CustomerCode: "NC1", BusinessName: "Nimbus Controls", CustomerGrade: "A"},
		{CustomerID: "C-2", CustomerCode: "ZM1", BusinessName: "Zephyr Marine", CustomerGrade: "B"},
	} {
		if err := svc.db.Create(&c).Error; err != nil {
			t.Fatalf("seed customer: %v", err)
		}
	}
	inv := Invoice{InvoiceNumber: "INV-1", GrandTotalBHD: 512.0, OutstandingBHD: 512.0, Status: "Sent", InvoiceDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)}
	if err := svc.db.Create(&inv).Error; err != nil {
		t.Fatalf("seed invoice: %v", err)
	}

	summary := svc.BusinessSummary()
	if summary["total_customers"].(int64) != 2 {
		t.Fatalf("expected 2 customers, got %v", summary["total_customers"])
	}
	if summary["total_revenue_bhd"].(float64) != 512.0 {
		t.Fatalf("expected revenue 512, got %v", summary["total_revenue_bhd"])
	}
	if summary["total_outstanding_bhd"].(float64) != 512.0 {
		t.Fatalf("expected outstanding 512, got %v", summary["total_outstanding_bhd"])
	}
	grades := summary["grade_distribution"].(map[string]int64)
	if grades["A"] != 1 || grades["B"] != 1 {
		t.Fatalf("unexpected grade distribution: %v", grades)
	}
}

func TestResolveCustomerReference_ExactFuzzyAmbiguous(t *testing.T) {
	svc := testService(t)
	for _, c := range []CustomerMaster{
		{CustomerID: "C-1", CustomerCode: "NC1", BusinessName: "Nimbus Controls", ShortCode: "NIM"},
		{CustomerID: "C-2", CustomerCode: "NO1", BusinessName: "Nimbus Offshore"},
		{CustomerID: "C-3", CustomerCode: "ZM1", BusinessName: "Zephyr Marine"},
	} {
		if err := svc.db.Create(&c).Error; err != nil {
			t.Fatalf("seed customer: %v", err)
		}
	}

	exact := svc.ResolveCustomerReference("Nimbus Controls")
	if exact == nil || exact.EntityID != "C-1" || exact.MatchReason != "exact customer match" {
		t.Fatalf("exact match failed: %+v", exact)
	}

	fuzzy := svc.ResolveCustomerReference("Zeph")
	if fuzzy == nil || fuzzy.EntityID != "C-3" || fuzzy.Ambiguous {
		t.Fatalf("fuzzy match failed: %+v", fuzzy)
	}

	ambiguous := svc.ResolveCustomerReference("Nimbus")
	if ambiguous == nil || !ambiguous.Ambiguous || len(ambiguous.Alternatives) != 2 {
		t.Fatalf("ambiguous match failed: %+v", ambiguous)
	}

	if svc.ResolveCustomerReference("") != nil {
		t.Fatal("empty reference must resolve to nil")
	}
}

func TestBuildIntentContext_UsesHostWorkContextAndRedacts(t *testing.T) {
	svc := testService(t)
	ctx := svc.BuildIntentContext(Intent{Domain: "general", RawQuery: "how are things"}, false)
	work, ok := ctx["work_data"].(map[string]any)
	if !ok || work["stub"] != true {
		t.Fatalf("work_data must come from HostPort, got %v", ctx["work_data"])
	}
	summary := ctx["business_summary"].(map[string]any)
	if _, leaked := summary["total_revenue_bhd"]; leaked {
		t.Fatal("revenue must be redacted without finance access")
	}
	if _, leaked := ctx["financial_data"]; leaked {
		t.Fatal("financial_data must be absent without finance access")
	}
}

func TestParseWindows_PureParsers(t *testing.T) {
	start, end, year, label, ok := ParseYearWindowFromQuery("show me revenue for 2025")
	if !ok || year != 2025 || label == "" {
		t.Fatalf("year window parse failed: %v %v %v %v %v", start, end, year, label, ok)
	}
	if start.Year() != 2025 || end.Year() != 2025 {
		t.Fatalf("window must span 2025: %v..%v", start, end)
	}

	qs, qe, qlabel, qok := ParseQuarterWindowFromQuery("invoices in Q2 2025")
	if !qok || qlabel == "" || qs.Month() != time.April || qe.Month() != time.June {
		t.Fatalf("quarter window parse failed: %v..%v %q %v", qs, qe, qlabel, qok)
	}

	if FirstNonEmpty("", "  ", "keep me", "later") != "keep me" {
		t.Fatal("FirstNonEmpty must return first non-blank value")
	}
	if Round3(1.23456) != 1.235 {
		t.Fatal("Round3 must round to 3 decimals")
	}
}
