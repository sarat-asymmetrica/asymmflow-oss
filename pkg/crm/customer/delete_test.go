package customer

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
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "customer.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&crm.CustomerMaster{}, &crm.CustomerContact{},
		&crm.SupplierMaster{}, &crm.SupplierContact{},
		&crm.Order{}, &crm.Offer{}, &crm.Opportunity{}, &crm.PurchaseOrder{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Finance-owned tables are counted by name in the child guard; give the
	// test DB minimal stand-ins with the columns the guard reads.
	for _, ddl := range []string{
		"CREATE TABLE invoices (id TEXT PRIMARY KEY, customer_id TEXT, deleted_at DATETIME)",
		"CREATE TABLE supplier_invoices (id TEXT PRIMARY KEY, supplier_id TEXT, deleted_at DATETIME)",
		"CREATE TABLE supplier_payments (id TEXT PRIMARY KEY, supplier_id TEXT, deleted_at DATETIME)",
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create finance stand-in: %v", err)
		}
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func TestDeleteCustomer_VerifiesExistence(t *testing.T) {
	db := deleteTestDB(t)

	err := DeleteCustomer(db, "missing")
	if err == nil || !strings.Contains(err.Error(), "[CUSTOMER_NOT_FOUND]") {
		t.Fatalf("missing customer must be refused, got %v", err)
	}

	seeded := crm.CustomerMaster{CustomerID: "C-1", CustomerCode: "NC1", BusinessName: "Nimbus Controls"}
	if err := db.Create(&seeded).Error; err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	if err := DeleteCustomer(db, seeded.ID); err != nil {
		t.Fatalf("delete customer: %v", err)
	}
	var count int64
	db.Model(&crm.CustomerMaster{}).Count(&count)
	if count != 0 {
		t.Fatal("customer must be soft-deleted out of default scope")
	}
}

func TestDeleteSupplierAndContacts(t *testing.T) {
	db := deleteTestDB(t)

	if err := DeleteSupplier(db, "missing"); err == nil || !strings.Contains(err.Error(), "[SUPPLIER_NOT_FOUND]") {
		t.Fatalf("missing supplier must be refused, got %v", err)
	}

	supplier := crm.SupplierMaster{SupplierName: "Zephyr Marine Supplies"}
	if err := db.Create(&supplier).Error; err != nil {
		t.Fatalf("seed supplier: %v", err)
	}
	contact := crm.SupplierContact{SupplierID: supplier.ID, ContactName: "Layla"}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("seed contact: %v", err)
	}

	if err := DeleteSupplierContact(db, contact.ID); err != nil {
		t.Fatalf("delete supplier contact: %v", err)
	}
	if err := DeleteSupplier(db, supplier.ID); err != nil {
		t.Fatalf("delete supplier: %v", err)
	}
}

// PC-D1 / PH SPOC #1: parties with transactional children are not deletable —
// the guard lives in the engine so approved delete requests hit it too.
func TestDeleteCustomer_RefusedWhenChildrenExist(t *testing.T) {
	db := deleteTestDB(t)

	customer := crm.CustomerMaster{CustomerID: "C-2", CustomerCode: "AT2", BusinessName: "Atlas Traders"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	if err := db.Create(&crm.Order{CustomerID: customer.ID, OrderNumber: "ORD-26-001"}).Error; err != nil {
		t.Fatalf("seed order: %v", err)
	}
	if err := db.Exec("INSERT INTO invoices (id, customer_id) VALUES ('inv-1', ?)", customer.ID).Error; err != nil {
		t.Fatalf("seed invoice: %v", err)
	}

	err := DeleteCustomer(db, customer.ID)
	if err == nil || !strings.Contains(err.Error(), "[CUSTOMER_HAS_LINKED_RECORDS]") {
		t.Fatalf("linked customer must be refused, got %v", err)
	}
	if !strings.Contains(err.Error(), "1 order(s)") || !strings.Contains(err.Error(), "1 invoice(s)") {
		t.Fatalf("refusal must carry exact counts, got %v", err)
	}

	// Soft-deleted children do not block: contacts-only / emptied parties stay deletable.
	if err := db.Delete(&crm.Order{}, "customer_id = ?", customer.ID).Error; err != nil {
		t.Fatalf("soft-delete order: %v", err)
	}
	if err := db.Exec("UPDATE invoices SET deleted_at = CURRENT_TIMESTAMP WHERE customer_id = ?", customer.ID).Error; err != nil {
		t.Fatalf("soft-delete invoice: %v", err)
	}
	if err := DeleteCustomer(db, customer.ID); err != nil {
		t.Fatalf("emptied customer must be deletable, got %v", err)
	}
}

func TestDeleteSupplier_RefusedWhenChildrenExist(t *testing.T) {
	db := deleteTestDB(t)

	supplier := crm.SupplierMaster{SupplierName: "Meridian Instruments GmbH"}
	if err := db.Create(&supplier).Error; err != nil {
		t.Fatalf("seed supplier: %v", err)
	}
	if err := db.Exec("INSERT INTO supplier_invoices (id, supplier_id) VALUES ('sinv-1', ?)", supplier.ID).Error; err != nil {
		t.Fatalf("seed supplier invoice: %v", err)
	}
	if err := db.Exec("INSERT INTO supplier_payments (id, supplier_id) VALUES ('spay-1', ?)", supplier.ID).Error; err != nil {
		t.Fatalf("seed supplier payment: %v", err)
	}

	err := DeleteSupplier(db, supplier.ID)
	if err == nil || !strings.Contains(err.Error(), "[SUPPLIER_HAS_LINKED_RECORDS]") {
		t.Fatalf("linked supplier must be refused, got %v", err)
	}
	if !strings.Contains(err.Error(), "1 supplier invoice(s)") || !strings.Contains(err.Error(), "1 payment(s)") {
		t.Fatalf("refusal must carry exact counts, got %v", err)
	}
}
