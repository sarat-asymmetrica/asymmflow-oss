package pipeline

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/ncruces/go-sqlite3/embed"
	"ph_holdings_app/pkg/crm"
)

func TestDeleteOfferNote(t *testing.T) {
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "pipeline.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&crm.OfferNote{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})

	if err := DeleteOfferNote(db, "missing"); err == nil || !strings.Contains(err.Error(), "note not found") {
		t.Fatalf("missing note must be refused, got %v", err)
	}

	note := crm.OfferNote{Content: "Follow up after commissioning"}
	if err := db.Create(&note).Error; err != nil {
		t.Fatalf("seed note: %v", err)
	}
	if err := DeleteOfferNote(db, note.ID); err != nil {
		t.Fatalf("delete note: %v", err)
	}
	var count int64
	db.Model(&crm.OfferNote{}).Count(&count)
	if count != 0 {
		t.Fatal("note must be deleted")
	}
}
