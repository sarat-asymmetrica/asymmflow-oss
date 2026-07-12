package audit

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/pkg/infra"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "audit.db")) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&infra.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	return db
}

// The B.2 regression test: resourceID and description MUST be persisted.
// The pre-engine logAudit accepted them and silently dropped them.
func TestRecord_PersistsResourceIDAndDescription(t *testing.T) {
	db := testDB(t)
	rec := NewRecorder(db)
	if err := rec.Record(Entry{
		UserID:      "u-1",
		Action:      "DELETE",
		Resource:    "customer",
		ResourceID:  "cust-42",
		Description: "removed duplicate record",
	}); err != nil {
		t.Fatal(err)
	}
	var row infra.AuditLog
	if err := db.First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.ResourceID != "cust-42" || row.Description != "removed duplicate record" ||
		row.UserID != "u-1" || row.Action != "DELETE" || row.Resource != "customer" {
		t.Fatalf("audit row dropped fields: %+v", row)
	}
}

func TestRecord_RequiresActionAndResource(t *testing.T) {
	rec := NewRecorder(testDB(t))
	if err := rec.Record(Entry{Action: "", Resource: "customer"}); err == nil {
		t.Fatal("actionless entry accepted")
	}
	if err := rec.Record(Entry{Action: "DELETE", Resource: " "}); err == nil {
		t.Fatal("resourceless entry accepted")
	}
	var nilRec *Recorder
	if err := nilRec.Record(Entry{Action: "A", Resource: "R"}); err == nil {
		t.Fatal("nil recorder accepted a write")
	}
}

func TestRecordAsync_WritesInBackground(t *testing.T) {
	db := testDB(t)
	rec := NewRecorder(db)
	rec.RecordAsync(Entry{UserID: "u-2", Action: "UPDATE", Resource: "offer", ResourceID: "off-7"}, nil)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var n int64
		db.Model(&infra.AuditLog{}).Count(&n)
		if n == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("async audit write never landed")
}
