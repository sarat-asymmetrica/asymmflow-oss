package procurement

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

func deleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "procurement.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&crm.PurchaseOrder{}, &crm.GoodsReceivedNote{}, &crm.GRNItem{}, &crm.SerialNumber{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func TestDeletePurchaseOrder_StatusGuard(t *testing.T) {
	db := deleteTestDB(t)

	sent := crm.PurchaseOrder{PONumber: "PO-2026-0001", Status: "Sent"}
	draft := crm.PurchaseOrder{PONumber: "PO-2026-0002", Status: "Draft"}
	for _, po := range []*crm.PurchaseOrder{&sent, &draft} {
		if err := db.Create(po).Error; err != nil {
			t.Fatalf("seed PO: %v", err)
		}
	}

	err := DeletePurchaseOrder(db, sent.ID)
	if err == nil || !strings.Contains(err.Error(), "[PO_INVALID_STATUS]") {
		t.Fatalf("sent PO must be refused, got %v", err)
	}
	if err := DeletePurchaseOrder(db, draft.ID); err != nil {
		t.Fatalf("draft PO must delete, got %v", err)
	}
}

func TestDeleteGRN_ResetsSerialsAndCascadesItems(t *testing.T) {
	db := deleteTestDB(t)

	grn := crm.GoodsReceivedNote{GRNNumber: "GRN-2026-0001"}
	if err := db.Create(&grn).Error; err != nil {
		t.Fatalf("seed GRN: %v", err)
	}
	item := crm.GRNItem{GRNID: grn.ID}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("seed GRN item: %v", err)
	}
	serial := crm.SerialNumber{SerialNo: "SER-9", Status: "Reserved", GRNNumber: grn.GRNNumber, GRNItemID: item.ID}
	if err := db.Create(&serial).Error; err != nil {
		t.Fatalf("seed serial: %v", err)
	}

	if err := DeleteGRN(db, grn.ID); err != nil {
		t.Fatalf("delete GRN: %v", err)
	}

	var reloaded crm.SerialNumber
	if err := db.First(&reloaded, "id = ?", serial.ID).Error; err != nil {
		t.Fatalf("reload serial: %v", err)
	}
	if reloaded.Status != "Available" || reloaded.GRNNumber != "" || reloaded.GRNItemID != "" {
		t.Fatalf("serial must reset to Available with GRN links cleared, got %+v", reloaded)
	}

	var itemCount int64
	db.Model(&crm.GRNItem{}).Where("grn_id = ?", grn.ID).Count(&itemCount)
	if itemCount != 0 {
		t.Fatal("GRN items must cascade-delete")
	}

	err := DeleteGRN(db, grn.ID)
	if err == nil || !strings.Contains(err.Error(), "[GRN_NOT_FOUND]") {
		t.Fatalf("expected GRN_NOT_FOUND on re-delete, got %v", err)
	}
}
