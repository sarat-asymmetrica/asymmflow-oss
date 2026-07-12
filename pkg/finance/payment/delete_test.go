package payment

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/ncruces/go-sqlite3/embed"
	"ph_holdings_app/pkg/finance"
)

func deleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "payment.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&finance.Invoice{}, &finance.Payment{},
		&finance.SupplierInvoice{}, &finance.SupplierPayment{},
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

func TestDeletePayment_RestoresInvoiceBalanceAndStatus(t *testing.T) {
	db := deleteTestDB(t)
	invoice := finance.Invoice{
		InvoiceNumber:  "INV-100",
		GrandTotalBHD:  500,
		OutstandingBHD: 0,
		Status:         "Paid",
		InvoiceDate:    time.Now().AddDate(0, -1, 0),
		DueDate:        time.Now().AddDate(0, 1, 0),
	}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	payment := finance.Payment{
		InvoiceID:     invoice.ID,
		InvoiceNumber: invoice.InvoiceNumber,
		AmountBHD:     500,
		PaymentDate:   time.Now(),
		PaymentMethod: "Cash",
	}
	if err := db.Create(&payment).Error; err != nil {
		t.Fatalf("seed payment: %v", err)
	}

	var audited *finance.Payment
	if err := DeletePayment(db, payment.ID, func(p finance.Payment) { audited = &p }); err != nil {
		t.Fatalf("delete payment: %v", err)
	}
	if audited == nil || audited.ID != payment.ID {
		t.Fatalf("audit hook must fire with the payment before deletion, got %+v", audited)
	}

	var reloaded finance.Invoice
	if err := db.First(&reloaded, "id = ?", invoice.ID).Error; err != nil {
		t.Fatalf("reload invoice: %v", err)
	}
	if reloaded.OutstandingBHD != 500 {
		t.Fatalf("outstanding must be restored to 500, got %.3f", reloaded.OutstandingBHD)
	}
	if reloaded.Status != "Sent" {
		t.Fatalf("status must revert to Sent (not yet due), got %q", reloaded.Status)
	}
	var count int64
	db.Model(&finance.Payment{}).Where("id = ?", payment.ID).Count(&count)
	if count != 0 {
		t.Fatal("payment row must be deleted")
	}
}

func TestDeletePayment_NotFound(t *testing.T) {
	db := deleteTestDB(t)
	err := DeletePayment(db, "missing-id", nil)
	if err == nil || !strings.Contains(err.Error(), "[PAYMENT_NOT_FOUND]") {
		t.Fatalf("expected coded PAYMENT_NOT_FOUND error, got %v", err)
	}
}

// W6 fix golden: the historical rollback targeted amount_paid_bhd, a
// column no model or migration ever defined, so the delete always
// failed. The Commander-authorized fix re-derives payment_status from
// the payments that remain (the update path's own derivation).
func TestDeleteSupplierPayment_RederivesPaymentStatus(t *testing.T) {
	db := deleteTestDB(t)
	invoice := finance.SupplierInvoice{
		InvoiceNumber: "SINV-7",
		ExchangeRate:  1,
		TotalBHD:      500,
		PaymentStatus: "Paid",
	}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatalf("seed supplier invoice: %v", err)
	}
	partial := finance.SupplierPayment{
		SupplierInvoiceID: invoice.ID,
		AmountBHD:         200,
		ExchangeRate:      1,
		PaymentDate:       time.Now(),
		PaymentMethod:     "Bank Transfer",
	}
	final := finance.SupplierPayment{
		SupplierInvoiceID: invoice.ID,
		AmountBHD:         300,
		ExchangeRate:      1,
		PaymentDate:       time.Now(),
		PaymentMethod:     "Bank Transfer",
	}
	for _, p := range []*finance.SupplierPayment{&partial, &final} {
		if err := db.Create(p).Error; err != nil {
			t.Fatalf("seed supplier payment: %v", err)
		}
	}

	if err := DeleteSupplierPayment(db, final.ID); err != nil {
		t.Fatalf("delete supplier payment: %v", err)
	}
	var reloaded finance.SupplierInvoice
	if err := db.First(&reloaded, "id = ?", invoice.ID).Error; err != nil {
		t.Fatalf("reload supplier invoice: %v", err)
	}
	if reloaded.PaymentStatus != "Partial" {
		t.Fatalf("payment status must re-derive to Partial (200/500 paid), got %q", reloaded.PaymentStatus)
	}

	if err := DeleteSupplierPayment(db, partial.ID); err != nil {
		t.Fatalf("delete remaining supplier payment: %v", err)
	}
	if err := db.First(&reloaded, "id = ?", invoice.ID).Error; err != nil {
		t.Fatalf("reload supplier invoice: %v", err)
	}
	if reloaded.PaymentStatus != "Unpaid" {
		t.Fatalf("payment status must re-derive to Unpaid, got %q", reloaded.PaymentStatus)
	}

	err := DeleteSupplierPayment(db, final.ID)
	if err == nil || !strings.Contains(err.Error(), "[NOT_FOUND]") {
		t.Fatalf("expected coded NOT_FOUND on re-delete, got %v", err)
	}
}

func TestDeleteSupplierPayment_OrphanPaymentStillDeletable(t *testing.T) {
	db := deleteTestDB(t)
	orphan := finance.SupplierPayment{
		SupplierInvoiceID: "gone-invoice",
		AmountBHD:         50,
		ExchangeRate:      1,
		PaymentDate:       time.Now(),
		PaymentMethod:     "Cash",
	}
	if err := db.Create(&orphan).Error; err != nil {
		t.Fatalf("seed orphan payment: %v", err)
	}
	if err := DeleteSupplierPayment(db, orphan.ID); err != nil {
		t.Fatalf("orphan payment must stay deletable, got %v", err)
	}
}
