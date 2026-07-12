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
