package device

import (
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/pkg/infra"
)

func testService(t *testing.T) *Service {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "device.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&infra.Device{}, &infra.DeviceUser{}, &infra.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return New(db)
}

func TestHashIdentifiersDeterministic(t *testing.T) {
	a := HashIdentifiers([]string{"host", "aa:bb"})
	b := HashIdentifiers([]string{"host", "aa:bb"})
	if a != b || len(a) != 64 {
		t.Fatalf("hash must be deterministic 64-hex: %q vs %q", a, b)
	}
	if HashIdentifiers([]string{"other"}) == a {
		t.Fatal("different identifiers must hash differently")
	}
}

func TestParsePermissions(t *testing.T) {
	got := ParsePermissions(`["a:view", "b:create", ""]`)
	if len(got) != 2 || got[0] != "a:view" || got[1] != "b:create" {
		t.Fatalf("unexpected permissions: %v", got)
	}
	if ParsePermissions("") != nil {
		t.Fatal("empty input must yield nil")
	}
}

func TestRegister_FirstSetupThenKnown(t *testing.T) {
	svc := testService(t)

	first, err := svc.Register()
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if first.Status != "first_setup" || !first.IsFirstSetup {
		t.Fatalf("first device must be first_setup: %+v", first)
	}

	// Same machine re-registers as the same device.
	again, err := svc.Register()
	if err != nil {
		t.Fatalf("re-register: %v", err)
	}
	if again.DeviceID != first.DeviceID {
		t.Fatalf("same machine must map to same device: %s vs %s", again.DeviceID, first.DeviceID)
	}

	current, err := svc.Current()
	if err != nil || current.ID != first.DeviceID {
		t.Fatalf("current device: %+v err %v", current, err)
	}
	if current.LastSeenAt == nil {
		t.Fatal("re-registration must refresh last_seen_at")
	}
}

func TestBlockRefusesAdminDeviceAndUnblockRestores(t *testing.T) {
	svc := testService(t)

	reg, err := svc.Register() // becomes the admin device
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.Block(reg.DeviceID); err == nil {
		t.Fatal("admin device must refuse blocking")
	}

	// Seed a second, non-admin device.
	other := infra.Device{MachineID: "other-machine", DeviceName: "kiosk", Status: "approved"}
	if err := svc.db.Create(&other).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}
	if _, err := svc.Block(other.ID); err != nil {
		t.Fatalf("block: %v", err)
	}
	if err := svc.Unblock(other.ID); err != nil {
		t.Fatalf("unblock: %v", err)
	}
	if err := svc.Unblock(other.ID); err == nil {
		t.Fatal("unblocking a non-blocked device must fail")
	}

	pending, err := svc.ListPending()
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("no pending devices expected, got %d", len(pending))
	}
	all, err := svc.ListAll()
	if err != nil || len(all) != 2 {
		t.Fatalf("expected 2 devices, got %d err %v", len(all), err)
	}
}
