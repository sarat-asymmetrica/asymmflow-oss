package fulfillment

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/pkg/crm"
)

func serialsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "serials.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&crm.ProductMaster{}, &crm.DeliveryNote{}, &crm.SerialNumber{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func seedProduct(t *testing.T, db *gorm.DB) crm.ProductMaster {
	t.Helper()
	product := crm.ProductMaster{ProductCode: "PT-100", Description: "Pressure transmitter"}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("seed product: %v", err)
	}
	return product
}

func TestRegister_CreatesAndRefusesDuplicates(t *testing.T) {
	db := serialsTestDB(t)
	product := seedProduct(t, db)
	svc := NewSerials(db)

	created, err := svc.Register(product.ID, []string{" SN-001 ", "SN-002"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("expected 2 serials, got %d", len(created))
	}
	if created[0].SerialNo != "SN-001" {
		t.Fatalf("expected trimmed serial SN-001, got %q", created[0].SerialNo)
	}
	if created[0].Status != "Available" || created[0].ProductCode != "PT-100" {
		t.Fatalf("unexpected serial: %+v", created[0])
	}

	// A duplicate anywhere in the batch fails the whole batch atomically.
	if _, err := svc.Register(product.ID, []string{"SN-003", "SN-002"}); err == nil {
		t.Fatal("expected duplicate serial to be refused")
	}
	var count int64
	db.Model(&crm.SerialNumber{}).Count(&count)
	if count != 2 {
		t.Fatalf("failed batch must leave no rows behind: got %d serials", count)
	}
}

func TestRegister_ValidatesBatch(t *testing.T) {
	db := serialsTestDB(t)
	product := seedProduct(t, db)
	svc := NewSerials(db)

	if _, err := svc.Register(product.ID, nil); err == nil {
		t.Fatal("expected empty batch to be refused")
	}
	if _, err := svc.Register(product.ID, []string{"SN-1", "  "}); err == nil {
		t.Fatal("expected blank serial to be refused")
	}
	big := make([]string, MaxSerialsPerBatch+1)
	for i := range big {
		big[i] = "SN"
	}
	if _, err := svc.Register(product.ID, big); err == nil {
		t.Fatal("expected oversized batch to be refused")
	}
	if _, err := svc.Register(product.ID, []string{strings.Repeat("x", 256)}); err == nil {
		t.Fatal("expected over-long serial to be refused")
	}
	if _, err := svc.Register("no-such-product", []string{"SN-1"}); err == nil {
		t.Fatal("expected unknown product to be refused")
	}
}

func TestAllocateToDN_AtomicAndRefusesUnavailable(t *testing.T) {
	db := serialsTestDB(t)
	product := seedProduct(t, db)
	svc := NewSerials(db)

	if _, err := svc.Register(product.ID, []string{"SN-A", "SN-B"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := svc.AllocateToDN("item-1", "DN-0001", "cust-1", "Al Manar Trading", []string{"SN-A"}); err != nil {
		t.Fatalf("allocate: %v", err)
	}

	got, err := svc.ByNumber("SN-A")
	if err != nil {
		t.Fatalf("by number: %v", err)
	}
	if got.Status != "Reserved" || got.DNNumber != "DN-0001" || got.CustomerName != "Al Manar Trading" {
		t.Fatalf("unexpected allocation state: %+v", got)
	}

	// Already reserved → refused; and the refusal must not disturb SN-B.
	if err := svc.AllocateToDN("item-2", "DN-0002", "cust-2", "Other", []string{"SN-B", "SN-A"}); err == nil {
		t.Fatal("expected reserved serial to refuse re-allocation")
	}
	b, _ := svc.ByNumber("SN-B")
	if b.Status != "Available" {
		t.Fatalf("failed allocation must roll back the batch: SN-B is %q", b.Status)
	}
}

func TestLifecycle_ShippedDeliveredInvoice(t *testing.T) {
	db := serialsTestDB(t)
	product := seedProduct(t, db)
	svc := NewSerials(db)

	dn := crm.DeliveryNote{DNNumber: "DN-0100"}
	if err := db.Create(&dn).Error; err != nil {
		t.Fatalf("seed dn: %v", err)
	}

	received := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	if err := svc.AssignToGRN("grn-item-1", "GRN-26-0001", "po-1", "PO-26-0001", product.ID, product.ProductCode, []string{"SN-L1"}, received); err != nil {
		t.Fatalf("assign to grn: %v", err)
	}
	if err := svc.UpdateWarranty(mustSerial(t, svc, "SN-L1").ID, 12); err != nil {
		t.Fatalf("warranty: %v", err)
	}
	if err := svc.AllocateToDN("dn-item-1", dn.DNNumber, "cust-1", "Al Manar Trading", []string{"SN-L1"}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if err := svc.MarkShipped(dn.DNNumber); err != nil {
		t.Fatalf("shipped: %v", err)
	}
	if got := mustSerial(t, svc, "SN-L1"); got.Status != "Shipped" || got.ShippedDate == nil {
		t.Fatalf("expected shipped with date, got %+v", got)
	}
	if err := svc.MarkDelivered(dn.ID); err != nil {
		t.Fatalf("delivered: %v", err)
	}
	got := mustSerial(t, svc, "SN-L1")
	if got.Status != "Delivered" || got.WarrantyStartDate == nil || got.WarrantyEndDate == nil {
		t.Fatalf("expected delivered with warranty window, got %+v", got)
	}
	wantEnd := got.WarrantyStartDate.AddDate(0, 12, 0)
	if !got.WarrantyEndDate.Equal(wantEnd) {
		t.Fatalf("warranty end: want %v, got %v", wantEnd, got.WarrantyEndDate)
	}

	if err := svc.LinkToInvoice("inv-1", "INV-26-0001", dn.ID); err != nil {
		t.Fatalf("link invoice: %v", err)
	}
	got = mustSerial(t, svc, "SN-L1")
	if got.InvoiceNumber != "INV-26-0001" {
		t.Fatalf("expected invoice link, got %+v", got)
	}

	// Race guard: a second invoice may not re-stamp an already-claimed serial.
	if err := svc.LinkToInvoice("inv-2", "INV-26-0002", dn.ID); err != nil {
		t.Fatalf("second link call errored: %v", err)
	}
	got = mustSerial(t, svc, "SN-L1")
	if got.InvoiceNumber != "INV-26-0001" {
		t.Fatalf("claimed serial was re-stamped: %+v", got)
	}
}

func TestSearch_EscapesLikeWildcards(t *testing.T) {
	db := serialsTestDB(t)
	product := seedProduct(t, db)
	svc := NewSerials(db)

	if _, err := svc.Register(product.ID, []string{"SN%1", "SNX1"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	found, err := svc.Search("SN%", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found) != 1 || found[0].SerialNo != "SN%1" {
		t.Fatalf("wildcard must match literally: %+v", found)
	}
}

func mustSerial(t *testing.T, svc *Serials, serialNo string) crm.SerialNumber {
	t.Helper()
	sn, err := svc.ByNumber(serialNo)
	if err != nil {
		t.Fatalf("serial %s: %v", serialNo, err)
	}
	return sn
}
