package numbering

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(gormlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&Sequence{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

var testNow = time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)

func TestRenderTemplates(t *testing.T) {
	cases := []struct {
		spec Spec
		seq  int
		want string
	}{
		{Spec{Prefix: "INV", Template: "INV-{date}-{seq}"}, 7, "INV-20260703-0007"},
		{Spec{Prefix: "PO", Template: "PO-{year}-{seq}"}, 12, "PO-2026-0012"},
		{Spec{Prefix: "Q", Template: "{prefix}/{yy}/{seq}", Pad: 3}, 5, "Q/26/005"},
	}
	for _, tc := range cases {
		if got := Render(tc.spec, testNow, tc.seq); got != tc.want {
			t.Errorf("Render(%q, %d) = %q, want %q", tc.spec.Template, tc.seq, got, tc.want)
		}
	}
}

func TestNextSequential(t *testing.T) {
	db := openTestDB(t)
	engine := New(db)
	spec := Spec{Prefix: "INV", Template: "INV-{date}-{seq}"}

	for i := 1; i <= 3; i++ {
		got, err := engine.Next(spec, testNow)
		if err != nil {
			t.Fatalf("Next #%d: %v", i, err)
		}
		want := fmt.Sprintf("INV-20260703-%04d", i)
		if got != want {
			t.Errorf("Next #%d = %q, want %q", i, got, want)
		}
	}
}

func TestPrefixesAreIndependent(t *testing.T) {
	db := openTestDB(t)
	engine := New(db)

	inv, err := engine.Next(Spec{Prefix: "INV", Template: "INV-{date}-{seq}"}, testNow)
	if err != nil {
		t.Fatal(err)
	}
	cn, err := engine.Next(Spec{Prefix: "CN", Template: "CN-{date}-{seq}"}, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if inv != "INV-20260703-0001" || cn != "CN-20260703-0001" {
		t.Errorf("prefixes not independent: inv=%q cn=%q", inv, cn)
	}
}

func TestYearRollover(t *testing.T) {
	db := openTestDB(t)
	engine := New(db)
	spec := Spec{Prefix: "PO", Template: "PO-{year}-{seq}"}

	if _, err := engine.Next(spec, testNow); err != nil {
		t.Fatal(err)
	}
	next, err := engine.Next(spec, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if next != "PO-2027-0001" {
		t.Errorf("year rollover: got %q, want PO-2027-0001", next)
	}
}

func TestSeedRunsOnceForFirstOfYear(t *testing.T) {
	db := openTestDB(t)
	engine := New(db)
	seedCalls := 0
	spec := Spec{
		Prefix:   "DN",
		Template: "DN-{year}-{seq}",
		Seed: func(tx *gorm.DB, year int) (int64, error) {
			seedCalls++
			return 41, nil // e.g. 41 legacy documents already exist
		},
	}

	first, err := engine.Next(spec, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if first != "DN-2026-0042" {
		t.Errorf("seeded first = %q, want DN-2026-0042", first)
	}
	second, err := engine.Next(spec, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if second != "DN-2026-0043" {
		t.Errorf("second = %q, want DN-2026-0043", second)
	}
	if seedCalls != 1 {
		t.Errorf("seed called %d times, want 1", seedCalls)
	}
}

func TestSpecValidation(t *testing.T) {
	db := openTestDB(t)
	engine := New(db)
	if _, err := engine.Next(Spec{Template: "X-{seq}"}, testNow); err == nil {
		t.Error("missing prefix should error")
	}
	if _, err := engine.Next(Spec{Prefix: "X"}, testNow); err == nil {
		t.Error("missing template should error")
	}
}

func TestFiscalYearFor(t *testing.T) {
	cases := []struct {
		name       string
		now        time.Time
		startMonth int
		want       int
	}{
		{"April FY, day before boundary", time.Date(2026, 3, 31, 23, 59, 0, 0, time.UTC), 4, 2025},
		{"April FY, boundary day", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), 4, 2026},
		{"April FY, January (before boundary)", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), 4, 2025},
		{"April FY, December (after boundary)", time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC), 4, 2026},
		{"April FY, month equals start month", time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC), 4, 2026},
		{"startMonth 0 is calendar year", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 0, 2026},
		{"startMonth 1 is calendar year", time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), 1, 2026},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FiscalYearFor(tc.now, tc.startMonth); got != tc.want {
				t.Errorf("FiscalYearFor(%v, %d) = %d, want %d", tc.now, tc.startMonth, got, tc.want)
			}
		})
	}
}

func TestFiscalYearNumberingAprilBoundary(t *testing.T) {
	// Spec AC #5: FY-boundary numbering demo. FYStartMonth 4 (India) resets
	// the counter on April 1, not January 1, and {fy} labels the series by
	// the fiscal year it belongs to.
	db := openTestDB(t)
	engine := New(db)
	spec := Spec{Prefix: "ININV", Template: "INV/{fy}/{seq}", Pad: 3, FYStartMonth: 4}

	mar30, err := engine.Next(spec, time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if mar30 != "INV/25-26/001" {
		t.Errorf("Mar 30 = %q, want INV/25-26/001", mar30)
	}

	mar31, err := engine.Next(spec, time.Date(2026, 3, 31, 23, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if mar31 != "INV/25-26/002" {
		t.Errorf("Mar 31 = %q, want INV/25-26/002 (still FY 2025-26, counter continues)", mar31)
	}

	apr1, err := engine.Next(spec, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if apr1 != "INV/26-27/001" {
		t.Errorf("Apr 1 = %q, want INV/26-27/001 (new FY, counter reset)", apr1)
	}

	apr2, err := engine.Next(spec, time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if apr2 != "INV/26-27/002" {
		t.Errorf("Apr 2 = %q, want INV/26-27/002", apr2)
	}

	for _, n := range []string{mar30, mar31, apr1, apr2} {
		if err := ValidateGSTSeriesNumber(n); err != nil {
			t.Errorf("ValidateGSTSeriesNumber(%q): %v", n, err)
		}
	}

	// The two fiscal years must be backed by separate Sequence rows, not a
	// single counter that happens to have reset.
	var rows []Sequence
	if err := db.Where("prefix = ?", "ININV").Order("year").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 Sequence rows (FY 2025 and FY 2026), got %d: %+v", len(rows), rows)
	}
	if rows[0].Year != 2025 || rows[0].LastNumber != 2 {
		t.Errorf("FY 2025-26 row = %+v, want Year=2025 LastNumber=2", rows[0])
	}
	if rows[1].Year != 2026 || rows[1].LastNumber != 2 {
		t.Errorf("FY 2026-27 row = %+v, want Year=2026 LastNumber=2", rows[1])
	}
}

func TestCalendarYearRolloverUnaffectedByFYStartMonth(t *testing.T) {
	// Regression: existing calendar-year specs (FYStartMonth left at its
	// zero value) must roll over on Jan 1 exactly as before Mission B2.
	db := openTestDB(t)
	engine := New(db)
	spec := Spec{Prefix: "INV", Template: "INV-{date}-{seq}"} // FYStartMonth: 0

	dec31, err := engine.Next(spec, time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if dec31 != "INV-20261231-0001" {
		t.Errorf("Dec 31 = %q, want INV-20261231-0001", dec31)
	}

	jan1, err := engine.Next(spec, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if jan1 != "INV-20270101-0001" {
		t.Errorf("Jan 1 = %q, want INV-20270101-0001 (calendar-year reset, unchanged)", jan1)
	}
}

func TestValidateGSTSeriesNumber(t *testing.T) {
	cases := []struct {
		name    string
		number  string
		wantErr bool
	}{
		{"16 chars ok", "ABCD1234567890AB", false}, // exactly 16
		{"17 chars fails", "ABCD1234567890ABC", true},
		{"typical India series ok", "INV/26-27/0001", false},
		{"space fails", "INV 001", true},
		{"rupee sign fails", "INV₹001", true},
		{"hash fails", "INV#001", true},
		{"empty fails", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateGSTSeriesNumber(tc.number)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateGSTSeriesNumber(%q) error = %v, wantErr %v", tc.number, err, tc.wantErr)
			}
		})
	}
}

func TestConcurrentAllocationUnique(t *testing.T) {
	// File-backed DB so concurrent goroutines share state across connections.
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "numbering.db")) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close() // release file handles so TempDir cleanup works on Windows
		}
	})
	if err := db.AutoMigrate(&Sequence{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	engine := New(db)
	spec := Spec{Prefix: "INV", Template: "INV-{date}-{seq}"}

	const n = 20
	var mu sync.Mutex
	seen := make(map[string]bool, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			num, err := engine.Next(spec, testNow)
			if err != nil {
				t.Errorf("Next: %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if seen[num] {
				t.Errorf("duplicate number issued: %s", num)
			}
			seen[num] = true
		}()
	}
	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != n {
		t.Errorf("issued %d unique numbers, want %d", len(seen), n)
	}
}
