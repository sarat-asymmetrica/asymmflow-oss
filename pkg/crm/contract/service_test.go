package contract

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testService(t *testing.T) *Service {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "contract.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Template{}, &Clause{}, &Contract{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return New(db)
}

func TestSelectClausesForGrade_Filters(t *testing.T) {
	svc := testService(t)
	if err := svc.SeedContractClauses(); err != nil {
		t.Fatalf("seed clauses: %v", err)
	}

	gradeA, err := svc.SelectClausesForGrade("A", "Services")
	if err != nil {
		t.Fatalf("grade A: %v", err)
	}
	gradeD, err := svc.SelectClausesForGrade("D", "Services")
	if err != nil {
		t.Fatalf("grade D: %v", err)
	}
	if len(gradeA) <= len(gradeD) {
		t.Fatalf("grade A must see more clauses than grade D: %d vs %d", len(gradeA), len(gradeD))
	}
	for _, c := range gradeD {
		if strings.Contains(c.Title, "Grade A") || strings.Contains(c.Title, "Grade C") {
			t.Fatalf("grade D selection leaked a higher-grade clause: %s", c.Title)
		}
	}
}

func TestGenerateContractNumber_SequentialFormat(t *testing.T) {
	svc := testService(t)

	first, err := svc.GenerateContractNumber()
	if err != nil {
		t.Fatalf("number: %v", err)
	}
	year := time.Now().Year() % 100
	want := fmt.Sprintf("CON%d/001", year)
	if first != want {
		t.Fatalf("expected %s, got %s", want, first)
	}

	// Simulate an existing contract and confirm increments.
	if err := svc.db.Create(&Contract{ID: "c-1", ContractNo: first}).Error; err != nil {
		t.Fatalf("seed contract: %v", err)
	}
	second, err := svc.GenerateContractNumber()
	if err != nil {
		t.Fatalf("second number: %v", err)
	}
	if second != fmt.Sprintf("CON%d/002", year) {
		t.Fatalf("expected 002, got %s", second)
	}
}

func TestSeedsAreIdempotent(t *testing.T) {
	svc := testService(t)
	for i := 0; i < 2; i++ {
		if err := svc.SeedContractTemplates(); err != nil {
			t.Fatalf("seed templates: %v", err)
		}
		if err := svc.SeedContractClauses(); err != nil {
			t.Fatalf("seed clauses: %v", err)
		}
	}
	var templates, clauses int64
	svc.db.Model(&Template{}).Count(&templates)
	svc.db.Model(&Clause{}).Count(&clauses)
	if templates != 3 {
		t.Fatalf("expected 3 templates, got %d", templates)
	}
	if clauses != 16 {
		t.Fatalf("expected 16 clauses, got %d", clauses)
	}
}
