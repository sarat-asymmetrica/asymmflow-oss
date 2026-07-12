package assets

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	dsn := "file:" + filepath.ToSlash(filepath.Join(dir, "assets.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	svc := New(db, filepath.Join(dir, "cache"))
	if err := svc.EnsureTable(); err != nil {
		t.Fatalf("ensure table: %v", err)
	}
	return svc
}

func TestUpsertGetRoundtrip(t *testing.T) {
	svc := testService(t)
	payload := []byte{0x89, 'P', 'N', 'G', 0x00, 0x01}

	if err := svc.UpsertBytes("company_logo", "Logo", "image/png", payload); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := svc.Get("company_logo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("roundtrip mismatch: %v vs %v", got, payload)
	}

	// Second upsert with the same name updates in place.
	if err := svc.UpsertBytes("company_logo", "Logo v2", "image/png", []byte{0x01}); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	infos, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(infos) != 1 || infos[0].Size != 1 || infos[0].Description != "Logo v2" {
		t.Fatalf("expected single updated asset, got %+v", infos)
	}
}

func TestGetToFileAndDeleteCleansCache(t *testing.T) {
	svc := testService(t)
	if err := svc.UpsertBytes("letterhead", "LH", "image/png", []byte{1, 2, 3}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	path, err := svc.GetToFile("letterhead")
	if err != nil {
		t.Fatalf("get to file: %v", err)
	}
	if filepath.Ext(path) != ".png" {
		t.Fatalf("expected .png extension, got %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file missing: %v", err)
	}

	if err := svc.Delete("letterhead"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cache file must be removed on delete")
	}
	if svc.Has("letterhead") {
		t.Fatal("asset must be gone")
	}
	if err := svc.Delete("letterhead"); err == nil {
		t.Fatal("second delete must report not found")
	}
}

func TestEnsureDefaultLetterheadSeedsPlaceholder(t *testing.T) {
	svc := testService(t)

	// No candidate path exists → generated placeholder is seeded.
	svc.EnsureDefaultLetterhead("letterhead", "Missing.png", "test", []string{filepath.Join(t.TempDir(), "nope.png")})
	if !svc.Has("letterhead") {
		t.Fatal("placeholder letterhead must be seeded")
	}
	data, err := svc.Get("letterhead")
	if err != nil || len(data) == 0 {
		t.Fatalf("placeholder bytes: %v (%d bytes)", err, len(data))
	}

	// A real candidate file wins for a different asset name.
	artwork := filepath.Join(t.TempDir(), "real.png")
	if err := os.WriteFile(artwork, []byte{0xAA, 0xBB}, 0644); err != nil {
		t.Fatalf("write artwork: %v", err)
	}
	svc.EnsureDefaultLetterhead("letterhead_ahs", "real.png", "test", []string{artwork})
	got, err := svc.Get("letterhead_ahs")
	if err != nil || !bytes.Equal(got, []byte{0xAA, 0xBB}) {
		t.Fatalf("expected artwork bytes, got %v err %v", got, err)
	}
}
