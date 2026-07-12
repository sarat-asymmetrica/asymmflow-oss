package composition

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/pkg/compliance"
	"ph_holdings_app/pkg/compliance/bahrain"
	"ph_holdings_app/pkg/compliance/saudi"
	"ph_holdings_app/pkg/infra/events"
)

func TestSQLiteDSN_Shape(t *testing.T) {
	got := SQLiteDSN("C:\\data\\pos.db", "busy_timeout(5000)", "journal_mode(WAL)")
	want := "file:C:/data/pos.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	if got != want {
		t.Fatalf("SQLiteDSN = %q, want %q", got, want)
	}
	if got := SQLiteDSN("plain.db"); got != "file:plain.db" {
		t.Fatalf("SQLiteDSN with no pragmas = %q", got)
	}
}

// Pins the Wave 3 finding that motivated this package's DSN builder: the
// ncruces driver honors ONLY ?_pragma=name(value) params. The trading app's
// previous mattn-style DSN (?_journal_mode=WAL&_busy_timeout=5000) was
// silently ignored — the pilot ran journal_mode=DELETE. If this test fails,
// the driver's DSN contract changed; re-verify every vertical's pragmas.
func TestSQLiteDSN_PragmasAreHonored(t *testing.T) {
	root := NewRoot()
	dsn := SQLiteDSN(filepath.Join(t.TempDir(), "probe.db"), DefaultPragmas...)
	db, err := root.OpenSQLite(dsn, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})

	pragma := func(name string) string {
		var v string
		if err := db.Raw("PRAGMA " + name).Scan(&v).Error; err != nil {
			t.Fatalf("pragma %s: %v", name, err)
		}
		return v
	}
	if got := pragma("journal_mode"); !strings.EqualFold(got, "wal") {
		t.Errorf("journal_mode = %q, want wal", got)
	}
	if got := pragma("busy_timeout"); got != "5000" {
		t.Errorf("busy_timeout = %q, want 5000", got)
	}
	if got := pragma("synchronous"); got != "1" { // 1 = NORMAL
		t.Errorf("synchronous = %q, want 1 (NORMAL)", got)
	}
	if got := pragma("foreign_keys"); got != "1" {
		t.Errorf("foreign_keys = %q, want 1", got)
	}
	if got := pragma("cache_size"); got != "-20000" {
		t.Errorf("cache_size = %q, want -20000", got)
	}
}

func TestWireCompliance_RegistersEnginesAndRoutesEvents(t *testing.T) {
	root := NewRoot()
	hook := root.WireCompliance(bahrain.New(), saudi.New())
	if root.Bus == nil || root.Registry == nil || hook == nil {
		t.Fatal("WireCompliance left the root partially wired")
	}
	if _, ok := root.Registry.Get(compliance.JurisdictionBahrain); !ok {
		t.Fatal("Bahrain engine not registered")
	}
	if _, ok := root.Registry.Get(compliance.JurisdictionSaudi); !ok {
		t.Fatal("Saudi engine not registered")
	}

	// An invoice event published on the root's bus must reach the hook.
	root.Bus.Publish(context.Background(), events.InvoiceCreated{
		BaseEvent:     events.BaseEvent{Timestamp: time.Now().UTC()},
		InvoiceNumber: "TEST-001",
		InvoiceDate:   time.Now().UTC(),
		SellerTaxID:   "310122393500003",
		Currency:      "SAR",
		Amount:        100.0,
		TaxAmount:     15.0,
		Jurisdiction:  "SA",
	})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if entries := hook.RecentValidations(1); len(entries) == 1 {
			if entries[0].Jurisdiction != "SA" {
				t.Fatalf("routed to %s, want SA", entries[0].Jurisdiction)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("compliance hook never recorded the validation (async hook timeout)")
}

func TestWireCompliance_ReusesExistingBus(t *testing.T) {
	bus := events.NewInMemoryBus()
	root := &Root{Bus: bus}
	root.WireCompliance(bahrain.New())
	if root.Bus != bus {
		t.Fatal("WireCompliance replaced a bus the caller already owned")
	}
}

func TestStandardOverlayDirs_CascadeOrder(t *testing.T) {
	dirs := StandardOverlayDirs("AsymmFlow")
	if len(dirs) == 0 {
		t.Fatal("no search dirs")
	}
	// CWD/data must precede CWD; both must be present.
	var dataIdx, cwdIdx = -1, -1
	for i, d := range dirs {
		if strings.HasSuffix(d, "data") && dataIdx == -1 {
			dataIdx = i
		}
	}
	cwd, _ := filepath.Abs(".")
	for i, d := range dirs {
		if filepath.Clean(d) == filepath.Clean(cwd) {
			cwdIdx = i
			break
		}
	}
	if dataIdx == -1 || cwdIdx == -1 || dataIdx > cwdIdx {
		t.Fatalf("cascade order wrong: dirs=%v (dataIdx=%d cwdIdx=%d)", dirs, dataIdx, cwdIdx)
	}
}
