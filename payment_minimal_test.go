package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

// TestMinimalPayment is a minimal test to debug the issue
func TestMinimalPayment(t *testing.T) {
	// Setup
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate
	err = db.AutoMigrate(&Invoice{}, &Payment{}, &Order{}, &Role{}, &Device{}, &DeviceUser{})
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create app
	app := &App{
		db:                     db,
		cache:                  NewCache(),
		startupImporting:       false,
		startupImportStartTime: time.Now(),
		currentUserID:          "test-user",
		currentUser: &User{
			Base:     Base{ID: "test-user"},
			Username: "test-admin",
			RoleName: "admin",
			Role: Role{
				Name:        "admin",
				DisplayName: "Administrator",
				Permissions: `["*"]`,
			},
		},
	}
	t.Cleanup(app.cache.Stop)

	// Create invoice
	invoice := &Invoice{
		Base: Base{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		InvoiceNumber:  "INV-MINIMAL-001",
		InvoiceDate:    time.Now(),
		DueDate:        time.Now().AddDate(0, 0, 30),
		GrandTotalBHD:  1000.000,
		OutstandingBHD: 1000.000,
		Status:         "Sent",
	}

	if err := db.Create(invoice).Error; err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	t.Logf("Invoice created: %s", invoice.ID)

	// Record payment
	t.Log("About to call RecordPayment...")
	payment, err := app.RecordPayment(invoice.ID, 500.000, "Cash", time.Now().Format("2006-01-02"), "TEST-REF")
	t.Logf("RecordPayment returned: payment=%v, err=%v", payment, err)

	// Debug
	if err != nil {
		t.Logf("Error type: %T", err)
		t.Logf("Error: %v", err)
		t.FailNow()
	}

	if payment == nil {
		t.Fatal("Payment is nil even though no error")
	}

	t.Logf("Payment created: ID=%s, Amount=%.3f", payment.ID, payment.AmountBHD)

	// Verify
	var updatedInvoice Invoice
	if err := db.First(&updatedInvoice, "id = ?", invoice.ID).Error; err != nil {
		t.Fatalf("Failed to get updated invoice: %v", err)
	}

	t.Logf("Invoice updated: Outstanding=%.3f, Status=%s", updatedInvoice.OutstandingBHD, updatedInvoice.Status)

	if updatedInvoice.OutstandingBHD != 500.000 {
		t.Errorf("Expected outstanding 500.000, got %.3f", updatedInvoice.OutstandingBHD)
	}

	if updatedInvoice.Status != "PartiallyPaid" {
		t.Errorf("Expected status PartiallyPaid, got %s", updatedInvoice.Status)
	}

	t.Log("SUCCESS!")
}
