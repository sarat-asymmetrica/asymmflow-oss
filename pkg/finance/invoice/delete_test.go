package invoice

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
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "invoice.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&finance.Invoice{}, &finance.DBInvoiceItem{}, &finance.Payment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func seedInvoice(t *testing.T, db *gorm.DB, status, orderID string) finance.Invoice {
	t.Helper()
	inv := finance.Invoice{
		InvoiceNumber: "INV-" + status,
		Status:        status,
		OrderID:       orderID,
		InvoiceDate:   time.Now(),
		DueDate:       time.Now().AddDate(0, 1, 0),
	}
	if err := db.Create(&inv).Error; err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	return inv
}

func TestDelete_RefusesTerminalStates(t *testing.T) {
	db := deleteTestDB(t)
	for _, status := range []string{"Paid", "Void", "Cancelled"} {
		inv := seedInvoice(t, db, status, "")
		err := Delete(db, inv.ID, nil)
		if err == nil || !strings.Contains(err.Error(), "terminal state") {
			t.Fatalf("%s invoice must be refused as terminal, got %v", status, err)
		}
	}
}

func TestDelete_RefusesPaymentHistory(t *testing.T) {
	db := deleteTestDB(t)
	inv := seedInvoice(t, db, "Sent", "")
	payment := finance.Payment{InvoiceID: inv.ID, AmountBHD: 10, PaymentDate: time.Now(), PaymentMethod: "Cash"}
	if err := db.Create(&payment).Error; err != nil {
		t.Fatalf("seed payment: %v", err)
	}
	err := Delete(db, inv.ID, nil)
	if err == nil || !strings.Contains(err.Error(), "payment history") {
		t.Fatalf("invoice with payments must be refused, got %v", err)
	}
}

func TestDelete_DeletesAndCallsReversalOnlyWithOrder(t *testing.T) {
	db := deleteTestDB(t)

	noOrder := seedInvoice(t, db, "Draft", "")
	called := false
	if err := Delete(db, noOrder.ID, func(finance.Invoice) { called = true }); err != nil {
		t.Fatalf("delete draft invoice: %v", err)
	}
	if called {
		t.Fatal("reversal must not fire for an invoice without an order")
	}

	withOrder := seedInvoice(t, db, "Sent", "ORD-1")
	if err := Delete(db, withOrder.ID, func(inv finance.Invoice) { called = inv.ID == withOrder.ID }); err != nil {
		t.Fatalf("delete invoice with order: %v", err)
	}
	if !called {
		t.Fatal("reversal must fire with the loaded invoice when an order is linked")
	}

	var count int64
	db.Model(&finance.Invoice{}).Count(&count)
	if count != 0 {
		t.Fatalf("both invoices must be gone, %d remain", count)
	}
}

func TestDeleteSupplier_RefusesPaidDeletesUnpaid(t *testing.T) {
	db := deleteTestDB(t)
	if err := db.AutoMigrate(&finance.SupplierInvoice{}); err != nil {
		t.Fatalf("migrate supplier invoices: %v", err)
	}

	paid := finance.SupplierInvoice{InvoiceNumber: "SINV-P", ExchangeRate: 1, PaymentStatus: "Paid"}
	unpaid := finance.SupplierInvoice{InvoiceNumber: "SINV-U", ExchangeRate: 1, PaymentStatus: "Unpaid"}
	for _, inv := range []*finance.SupplierInvoice{&paid, &unpaid} {
		if err := db.Create(inv).Error; err != nil {
			t.Fatalf("seed supplier invoice: %v", err)
		}
	}

	err := DeleteSupplier(db, paid.ID)
	if err == nil || !strings.Contains(err.Error(), "cannot delete paid invoice") {
		t.Fatalf("paid supplier invoice must be refused, got %v", err)
	}
	if err := DeleteSupplier(db, unpaid.ID); err != nil {
		t.Fatalf("unpaid supplier invoice must delete, got %v", err)
	}
}
