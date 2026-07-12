package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// =============================================================================
// PAYMENT SERVICE TESTS - REAL IMPLEMENTATION
// =============================================================================
// This file contains REAL tests for payment recording functionality.
// Tests verify overpayment prevention, duplicate detection, and transaction safety.
// =============================================================================

// setupPaymentTestApp creates an in-memory database for testing
func setupPaymentTestApp(t *testing.T) *App {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	require.NoError(t, err)

	// AutoMigrate all required tables
	err = db.AutoMigrate(
		&Invoice{},
		&Payment{},
		&Order{},
		&CustomerMaster{},
	)
	require.NoError(t, err)

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
	return app
}

// createTestInvoice creates a test invoice with specified amount
func createTestInvoice(t *testing.T, app *App, amount float64, invoiceNumber string) *Invoice {
	t.Helper()

	invoice := &Invoice{
		Base: Base{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		InvoiceNumber:  invoiceNumber,
		InvoiceDate:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), // Fixed date before all test payment dates
		DueDate:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), // Far future due date
		GrandTotalBHD:  amount,
		OutstandingBHD: amount,
		Status:         "Sent",
	}
	require.NoError(t, app.db.Create(invoice).Error)
	return invoice
}

// TestRecordPayment_Success verifies successful payment recording
func TestRecordPayment_Success(t *testing.T) {
	t.Run("should create payment record successfully", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-001")

		// Execute
		payment, err := app.RecordPayment(invoice.ID, 500.000, "Bank Transfer", "2026-01-27", "REF123")

		// Verify
		require.NoError(t, err)
		require.NotNil(t, payment)
		assert.Equal(t, 500.000, payment.AmountBHD)
		assert.Equal(t, "Bank Transfer", payment.PaymentMethod)
		assert.Equal(t, "REF123", payment.Reference)
		assert.Equal(t, invoice.ID, payment.InvoiceID)

		// Verify invoice updated
		var updatedInvoice Invoice
		err = app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 500.000, updatedInvoice.OutstandingBHD)
		assert.Equal(t, "PartiallyPaid", updatedInvoice.Status)
	})

	t.Run("should handle full payment", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-002")

		// Execute
		payment, err := app.RecordPayment(invoice.ID, 1000.000, "Cash", "2026-01-27", "FULL")

		// Verify
		require.NoError(t, err)
		require.NotNil(t, payment)

		// Verify invoice fully paid
		var updatedInvoice Invoice
		err = app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 0.000, updatedInvoice.OutstandingBHD)
		assert.Equal(t, "Paid", updatedInvoice.Status)
	})

	t.Run("should handle partial payment", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-003")

		// Execute - first payment
		payment1, err := app.RecordPayment(invoice.ID, 300.000, "Bank Transfer", "2026-01-27", "P1")
		require.NoError(t, err)
		require.NotNil(t, payment1)

		// Execute - second payment
		payment2, err := app.RecordPayment(invoice.ID, 700.000, "Bank Transfer", "2026-01-28", "P2")
		require.NoError(t, err)
		require.NotNil(t, payment2)

		// Verify total payments
		var payments []Payment
		err = app.db.Where("invoice_id = ?", invoice.ID).Find(&payments).Error
		require.NoError(t, err)
		assert.Len(t, payments, 2)

		// Verify invoice fully paid
		var updatedInvoice Invoice
		err = app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 0.000, updatedInvoice.OutstandingBHD)
		assert.Equal(t, "Paid", updatedInvoice.Status)
	})

	t.Run("should round to BHD precision (3 decimals)", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 500.000, "INV-TEST-004")

		// Execute - payment with excessive decimal places
		payment, err := app.RecordPayment(invoice.ID, 123.456789, "Cash", "2026-01-27", "REF")

		// Verify - should be rounded to 3 decimals
		require.NoError(t, err)
		require.NotNil(t, payment)
		assert.Equal(t, 123.457, payment.AmountBHD)
	})

	t.Run("should validate payment date format", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 100.000, "INV-TEST-005")

		// Execute - valid date
		payment, err := app.RecordPayment(invoice.ID, 50.000, "Cash", "2026-01-27", "REF")
		require.NoError(t, err)
		require.NotNil(t, payment)

		// Execute - invalid date format (should fail)
		_, err = app.RecordPayment(invoice.ID, 50.000, "Cash", "27-01-2026", "REF2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid date format")
	})

	t.Run("should trigger order completion check", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)

		// Create order
		order := &Order{
			Base: Base{
				ID:        uuid.New().String(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			OrderNumber:   "ORD-TEST-001",
			OrderDate:     time.Now(),
			TotalValueBHD: 1000.000,
			GrandTotalBHD: 1000.000,
			Status:        "Confirmed",
		}
		require.NoError(t, app.db.Create(order).Error)

		// Create invoice linked to order
		invoice := &Invoice{
			Base: Base{
				ID:        uuid.New().String(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			InvoiceNumber:  "INV-TEST-006",
			InvoiceDate:    time.Now(),
			DueDate:        time.Now().AddDate(0, 0, 30),
			OrderID:        order.ID,
			GrandTotalBHD:  1000.000,
			OutstandingBHD: 1000.000,
			Status:         "Sent",
		}
		require.NoError(t, app.db.Create(invoice).Error)

		// Execute - full payment (use today's date so days_to_payment >= 0)
		payment, err := app.RecordPayment(invoice.ID, 1000.000, "Cash", time.Now().Format("2006-01-02"), "REF")
		require.NoError(t, err)
		require.NotNil(t, payment)

		// Verify order status progressed to Complete
		var updatedOrder Order
		err = app.db.First(&updatedOrder, "id = ?", order.ID).Error
		require.NoError(t, err)
		assert.Equal(t, "Complete", updatedOrder.Status)
	})
}

// TestRecordPayment_OverpaymentPrevention verifies overpayment protection
func TestRecordPayment_OverpaymentPrevention(t *testing.T) {
	t.Run("should reject payment exceeding outstanding balance", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-101")

		// Execute - try to pay 1500 BHD on 1000 BHD invoice
		payment, err := app.RecordPayment(invoice.ID, 1500.000, "Cash", "2026-01-27", "REF")

		// Verify error
		require.Error(t, err)
		assert.Nil(t, payment)
		assert.Contains(t, err.Error(), "exceeds outstanding balance")

		// Verify invoice unchanged
		var unchangedInvoice Invoice
		err = app.db.First(&unchangedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 1000.000, unchangedInvoice.OutstandingBHD)
		assert.Equal(t, "Sent", unchangedInvoice.Status)

		// Verify no payment record created
		var count int64
		app.db.Model(&Payment{}).Where("invoice_id = ?", invoice.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("should reject overpayment after partial payments", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-102")

		// First payment - 600 BHD
		_, err := app.RecordPayment(invoice.ID, 600.000, "Cash", "2026-01-26", "REF1")
		require.NoError(t, err)

		// Execute - try to pay 500 BHD when only 400 BHD outstanding
		payment, err := app.RecordPayment(invoice.ID, 500.000, "Cash", "2026-01-27", "REF2")

		// Verify error
		require.Error(t, err)
		assert.Nil(t, payment)
		assert.Contains(t, err.Error(), "exceeds outstanding balance")

		// Verify invoice balance unchanged (still 400 BHD outstanding)
		var updatedInvoice Invoice
		err = app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 400.000, updatedInvoice.OutstandingBHD)
	})

	t.Run("should accept payment exactly matching outstanding balance", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-103")

		// Execute - exact payment
		payment, err := app.RecordPayment(invoice.ID, 1000.000, "Cash", "2026-01-27", "REF")

		// Verify success
		require.NoError(t, err)
		require.NotNil(t, payment)

		// Verify invoice fully paid
		var updatedInvoice Invoice
		err = app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 0.000, updatedInvoice.OutstandingBHD)
		assert.Equal(t, "Paid", updatedInvoice.Status)
	})

	t.Run("should handle precision edge cases", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-104")

		// Execute - overpayment by 0.001 BHD (should be rejected)
		payment, err := app.RecordPayment(invoice.ID, 1000.001, "Cash", "2026-01-27", "REF")

		// Verify error
		require.Error(t, err)
		assert.Nil(t, payment)
		assert.Contains(t, err.Error(), "exceeds outstanding balance")

		// Verify invoice unchanged
		var unchangedInvoice Invoice
		err = app.db.First(&unchangedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 1000.000, unchangedInvoice.OutstandingBHD)
	})
}

// TestRecordPayment_DuplicateDetection verifies duplicate payment prevention
func TestRecordPayment_DuplicateDetection(t *testing.T) {
	t.Run("should detect duplicate payment by amount and date", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-201")

		// Execute - first payment
		payment1, err1 := app.RecordPayment(invoice.ID, 500.000, "Bank Transfer", "2026-01-27", "REF123")
		require.NoError(t, err1)
		require.NotNil(t, payment1)

		// Execute - duplicate payment (same invoice, amount, and date)
		payment2, err2 := app.RecordPayment(invoice.ID, 500.000, "Bank Transfer", "2026-01-27", "REF456")

		// Verify duplicate detected
		require.Error(t, err2)
		assert.Nil(t, payment2)
		assert.Contains(t, err2.Error(), "duplicate")

		// Verify only one payment in database
		var count int64
		app.db.Model(&Payment{}).Where("invoice_id = ?", invoice.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("should allow different dates with same amount", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-202")

		// Execute - payments on different dates (should be allowed)
		payment1, err1 := app.RecordPayment(invoice.ID, 500.000, "Bank Transfer", "2026-01-26", "REF123")
		require.NoError(t, err1)
		require.NotNil(t, payment1)

		payment2, err2 := app.RecordPayment(invoice.ID, 500.000, "Bank Transfer", "2026-01-27", "REF456")
		require.NoError(t, err2)
		require.NotNil(t, payment2)

		// Verify both payments exist
		var count int64
		app.db.Model(&Payment{}).Where("invoice_id = ?", invoice.ID).Count(&count)
		assert.Equal(t, int64(2), count)

		// Verify invoice fully paid
		var updatedInvoice Invoice
		err := app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 0.000, updatedInvoice.OutstandingBHD)
	})

	t.Run("should allow different amounts on same date", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-203")

		// Execute - payments with different amounts (should be allowed)
		payment1, err1 := app.RecordPayment(invoice.ID, 300.000, "Bank Transfer", "2026-01-27", "REF123")
		require.NoError(t, err1)
		require.NotNil(t, payment1)

		payment2, err2 := app.RecordPayment(invoice.ID, 700.000, "Bank Transfer", "2026-01-27", "REF456")
		require.NoError(t, err2)
		require.NotNil(t, payment2)

		// Verify both payments exist
		var count int64
		app.db.Model(&Payment{}).Where("invoice_id = ?", invoice.ID).Count(&count)
		assert.Equal(t, int64(2), count)
	})

	t.Run("should allow same amount and date for different invoices", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice1 := createTestInvoice(t, app, 1000.000, "INV-TEST-204")
		invoice2 := createTestInvoice(t, app, 2000.000, "INV-TEST-205")

		// Execute - same amount and date but different invoices
		payment1, err1 := app.RecordPayment(invoice1.ID, 500.000, "Bank Transfer", "2026-01-27", "REF123")
		require.NoError(t, err1)
		require.NotNil(t, payment1)

		payment2, err2 := app.RecordPayment(invoice2.ID, 500.000, "Bank Transfer", "2026-01-27", "REF456")
		require.NoError(t, err2)
		require.NotNil(t, payment2)

		// Verify both payments succeed (different invoices = OK)
		assert.NotEqual(t, payment1.ID, payment2.ID)
	})
}

// TestRecordPayment_ValidationErrors verifies input validation
func TestRecordPayment_ValidationErrors(t *testing.T) {
	t.Run("should reject negative amount", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 100.000, "INV-TEST-301")

		// Execute - negative amount
		payment, err := app.RecordPayment(invoice.ID, -100.000, "Cash", "2026-01-27", "REF")

		// Verify error
		require.Error(t, err)
		assert.Nil(t, payment)
		assert.Contains(t, err.Error(), "must be greater than zero")
	})

	t.Run("should reject zero amount", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 100.000, "INV-TEST-302")

		// Execute - zero amount
		payment, err := app.RecordPayment(invoice.ID, 0.000, "Cash", "2026-01-27", "REF")

		// Verify error
		require.Error(t, err)
		assert.Nil(t, payment)
		assert.Contains(t, err.Error(), "must be greater than zero")
	})

	t.Run("should reject invalid date format", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 100.000, "INV-TEST-303")

		// Execute - invalid date format
		payment, err := app.RecordPayment(invoice.ID, 100.000, "Cash", "27-01-2026", "REF")

		// Verify error
		require.Error(t, err)
		assert.Nil(t, payment)
		assert.Contains(t, err.Error(), "Invalid date format")
	})

	t.Run("should reject non-existent invoice", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)

		// Execute - non-existent invoice ID
		payment, err := app.RecordPayment("non-existent-uuid", 100.000, "Cash", "2026-01-27", "REF")

		// Verify error
		require.Error(t, err)
		assert.Nil(t, payment)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("should handle nil database connection", func(t *testing.T) {
		// Setup - app with nil database (bypass RBAC to test DB nil check)
		app := &App{db: nil, startupImporting: true}

		// Execute
		payment, err := app.RecordPayment("some-id", 100.000, "Cash", "2026-01-27", "REF")

		// Verify error
		require.Error(t, err)
		assert.Nil(t, payment)
		assert.Contains(t, err.Error(), "connection not available")
	})
}

// TestRecordPayment_TransactionSafety verifies transaction rollback on errors
func TestRecordPayment_TransactionSafety(t *testing.T) {
	t.Run("should rollback on overpayment error", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-401")

		// Execute - attempt overpayment (should trigger rollback)
		payment, err := app.RecordPayment(invoice.ID, 1500.000, "Cash", "2026-01-27", "REF")

		// Verify error and no payment created
		require.Error(t, err)
		assert.Nil(t, payment)

		// Verify invoice unchanged (rollback successful)
		var unchangedInvoice Invoice
		err = app.db.First(&unchangedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 1000.000, unchangedInvoice.OutstandingBHD)
		assert.Equal(t, "Sent", unchangedInvoice.Status)

		// Verify no payment record created
		var count int64
		app.db.Model(&Payment{}).Where("invoice_id = ?", invoice.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("should maintain consistency after partial payment failure", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-402")

		// First successful payment
		payment1, err := app.RecordPayment(invoice.ID, 600.000, "Cash", "2026-01-26", "REF1")
		require.NoError(t, err)
		require.NotNil(t, payment1)

		// Second payment attempt fails (overpayment)
		payment2, err := app.RecordPayment(invoice.ID, 500.000, "Cash", "2026-01-27", "REF2")
		require.Error(t, err)
		assert.Nil(t, payment2)

		// Verify invoice still has correct outstanding balance
		var updatedInvoice Invoice
		err = app.db.First(&updatedInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 400.000, updatedInvoice.OutstandingBHD)

		// Verify only one payment exists
		var count int64
		app.db.Model(&Payment{}).Where("invoice_id = ?", invoice.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("should maintain data integrity across multiple operations", func(t *testing.T) {
		// Setup
		app := setupPaymentTestApp(t)
		invoice := createTestInvoice(t, app, 1000.000, "INV-TEST-403")

		// Multiple successful payments
		_, err := app.RecordPayment(invoice.ID, 200.000, "Cash", "2026-01-25", "REF1")
		require.NoError(t, err)

		_, err = app.RecordPayment(invoice.ID, 300.000, "Bank Transfer", "2026-01-26", "REF2")
		require.NoError(t, err)

		_, err = app.RecordPayment(invoice.ID, 500.000, "Cheque", "2026-01-27", "REF3")
		require.NoError(t, err)

		// Verify final state
		var finalInvoice Invoice
		err = app.db.First(&finalInvoice, "id = ?", invoice.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 0.000, finalInvoice.OutstandingBHD)
		assert.Equal(t, "Paid", finalInvoice.Status)

		// Verify all 3 payments exist
		var payments []Payment
		err = app.db.Where("invoice_id = ?", invoice.ID).Find(&payments).Error
		require.NoError(t, err)
		assert.Len(t, payments, 3)

		// Verify total payments sum correctly
		totalPaid := payments[0].AmountBHD + payments[1].AmountBHD + payments[2].AmountBHD
		assert.Equal(t, 1000.000, totalPaid)
	})
}
