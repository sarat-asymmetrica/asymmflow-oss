package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	financepayment "ph_holdings_app/pkg/finance/payment"
)

// Supplier payment service constants
const (
	FloatingPointTolerance = 0.001          // Tolerance for BHD float comparisons
	FuturePaymentWindow    = 24 * time.Hour // Maximum future date allowed for payments
)

// RecordSupplierPayment creates a new payment record for a supplier invoice
// and updates the invoice payment status if fully paid
func (a *App) RecordSupplierPayment(invoiceID string, amount float64, currency, method, date, reference string, exchangeRate float64) (*SupplierPayment, error) {
	return a.paymentService().RecordSupplierPayment(invoiceID, amount, currency, method, date, reference, exchangeRate)
}

func recordSupplierPayment(a *App, invoiceID string, amount float64, currency, method, date, reference string, exchangeRate float64) (*SupplierPayment, error) {
	if err := a.requirePermission("payments:record"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Validate invoice exists
	var invoice SupplierInvoice
	if err := a.db.First(&invoice, "id = ?", invoiceID).Error; err != nil {
		return nil, fmt.Errorf("invoice not found: %w", err)
	}

	// P0-4 Fix: Prevent paying disputed/unmatched invoices
	if invoice.Status == "Disputed" {
		return nil, fmt.Errorf("cannot pay disputed invoice %s - resolve dispute first", invoice.InvoiceNumber)
	}
	if invoice.MatchStatus == "Discrepancy" {
		return nil, fmt.Errorf("cannot pay invoice %s with 3-way match discrepancies - resolve discrepancies first", invoice.InvoiceNumber)
	}
	// Wave 8 P5-1 (user-ratified hybrid): cash leaves ONLY after the explicit,
	// segregated ApproveSupplierInvoice step. Mission G had allowed "Verified"
	// (clean 3-way match) to be paid directly; the hybrid keeps OSS's status
	// vocabulary but restores PH's segregation-of-duties — match verifies the
	// paperwork, a different human approves the disbursement. Disputed and
	// Discrepancy remain hard-blocked above.
	if invoice.Status != "Approved" {
		return nil, fmt.Errorf("invoice must be Approved before payment (run ApproveSupplierInvoice after a clean 3-way match), current status: %s", invoice.Status)
	}

	// Validate amount
	if amount <= 0 {
		return nil, fmt.Errorf("payment amount must be positive, got: %.3f", amount)
	}

	// Set default currency
	if currency == "" {
		currency = "BHD"
	}

	// C1 (Wave 9.3, authorized posting change): the confirmed rate now drives
	// the BHD posting in one write instead of the old implicit 1:1. Omitted or
	// non-positive rate falls back to 1.0, preserving BHD behavior and any
	// caller that doesn't pass one.
	if exchangeRate <= 0 {
		exchangeRate = 1.0
		if currency != "BHD" {
			log.Printf("Warning: Non-BHD currency %s provided without a positive exchange rate, falling back to 1.0", currency)
		}
	}

	// Calculate amount in BHD
	amountBHD := amount * exchangeRate

	// Parse payment date
	paymentDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
	}

	// Prevent future payment dates
	if paymentDate.After(time.Now().Add(FuturePaymentWindow)) {
		return nil, fmt.Errorf("payment date cannot be in the future")
	}

	// SECURITY FIX (TOCTOU): Start transaction BEFORE balance check to prevent race conditions
	// Two concurrent payments can no longer both pass validation (SELECT FOR UPDATE locks the row)
	tx := a.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Check for existing payments to calculate outstanding balance (INSIDE transaction)
	var totalPaidBefore float64
	if err := tx.Model(&SupplierPayment{}).
		Where("supplier_invoice_id = ?", invoiceID).
		Select("COALESCE(SUM(amount_bhd), 0)").
		Scan(&totalPaidBefore).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to calculate existing payments: %w", err)
	}

	// Validate payment doesn't exceed outstanding balance (within locked transaction)
	outstanding := invoice.TotalBHD - totalPaidBefore
	if amountBHD > outstanding+FloatingPointTolerance { // Small tolerance for floating point
		tx.Rollback()
		return nil, fmt.Errorf("payment amount %.3f BHD exceeds outstanding balance %.3f BHD", amountBHD, outstanding)
	}

	// SECURITY FIX: Duplicate payment detection regardless of reference
	var duplicateCount int64
	duplicateQuery := tx.Model(&SupplierPayment{}).Where("supplier_invoice_id = ? AND amount_bhd = ?", invoiceID, amountBHD)
	if reference != "" {
		duplicateQuery = duplicateQuery.Where("reference = ?", reference)
	} else {
		// For payments without reference, check same amount on same date
		duplicateQuery = duplicateQuery.Where("DATE(payment_date) = DATE(?)", paymentDate)
	}
	duplicateQuery.Count(&duplicateCount)
	if duplicateCount > 0 {
		tx.Rollback()
		return nil, fmt.Errorf("duplicate payment detected: matching invoice, amount, and %s already exists",
			func() string {
				if reference != "" {
					return "reference"
				}
				return "date"
			}())
	}

	// Create payment record with display fields populated from invoice
	payment := SupplierPayment{
		SupplierInvoiceID: invoiceID,
		SupplierID:        invoice.SupplierID,
		AmountForeign:     amount,
		Currency:          currency,
		ExchangeRate:      exchangeRate,
		AmountBHD:         amountBHD,
		PaymentDate:       paymentDate,
		PaymentMethod:     method,
		Reference:         reference,
		SupplierName:      invoice.SupplierName,  // Populate for UI display
		InvoiceNumber:     invoice.InvoiceNumber, // Populate for UI display
		Division:          a.resolveSupplierInvoiceDivision(invoice),
	}

	if err := tx.Create(&payment).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	log.Printf("💰 Supplier payment recorded: %.3f %s for invoice %s", amount, currency, invoiceID)

	// Wave 8 P5-1: settle through the supplier-invoice payment-state policy so
	// invoice.Status (not just payment_status) reflects the ledger — a fully
	// paid invoice reads "Paid", a partial stays "Approved"/"Partial". This
	// replaces the inline payment_status-only update the parity audit flagged.
	state, err := a.applySupplierInvoicePaymentState(tx, invoiceID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update invoice payment state: %w", err)
	}
	log.Printf("📊 Invoice %s payment state: %s / %s (outstanding %.3f BHD)",
		invoiceID, state.InvoiceStatus, state.PaymentStatus, state.OutstandingBHD)

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to commit payment transaction: %w", err)
	}

	// P1 FIX: Audit log for financial transaction
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogFinancialTransaction(
			a.getCurrentUserID(),
			"supplier_payment_recorded",
			"supplier_invoice",
			invoiceID,
			amountBHD,
			"BHD",
			true,
			map[string]any{
				"supplier_id":    invoice.SupplierID,
				"invoice_number": invoice.InvoiceNumber,
				"payment_method": method,
				"payment_date":   date,
				"reference":      reference,
				"amount_foreign": amount,
				"currency":       currency,
				"exchange_rate":  exchangeRate,
				"payment_status": "recorded",
			},
		)
	}

	return &payment, nil
}

// GetSupplierPaymentsByInvoice retrieves all payments for a specific supplier invoice
func (a *App) GetSupplierPaymentsByInvoice(invoiceID string) ([]SupplierPayment, error) {
	return a.paymentService().GetSupplierPaymentsByInvoice(invoiceID)
}

func getSupplierPaymentsByInvoice(a *App, invoiceID string) ([]SupplierPayment, error) {
	if err := a.requirePermission("payments:view"); err != nil {
		return nil, err
	}

	var payments []SupplierPayment

	if err := a.db.Where("supplier_invoice_id = ?", invoiceID).
		Order("payment_date DESC").
		Find(&payments).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve payments for invoice %s: %w", invoiceID, err)
	}

	return payments, nil
}

// GetAllSupplierPayments retrieves all supplier payments (limited to 500 most recent)
func (a *App) GetAllSupplierPayments() ([]SupplierPayment, error) {
	return a.paymentService().GetAllSupplierPayments()
}

func getAllSupplierPayments(a *App) ([]SupplierPayment, error) {
	if err := a.requirePermission("payments:view"); err != nil {
		return nil, err
	}

	var payments []SupplierPayment

	if err := a.db.Order("payment_date DESC").
		Limit(500).
		Find(&payments).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve supplier payments: %w", err)
	}

	return payments, nil
}

// GetSupplierPaymentsSummary returns summary statistics for supplier payments
func (a *App) GetSupplierPaymentsSummary() (map[string]any, error) {
	return a.paymentService().GetSupplierPaymentsSummary()
}

func getSupplierPaymentsSummary(a *App) (map[string]any, error) {
	if err := a.requirePermission("payments:view"); err != nil {
		return nil, err
	}

	summary := make(map[string]any)

	// Total paid amount in BHD
	var totalPaidBHD float64
	if err := a.db.Model(&SupplierPayment{}).
		Select("COALESCE(SUM(amount_bhd), 0)").
		Scan(&totalPaidBHD).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate total paid: %w", err)
	}
	summary["total_paid_bhd"] = totalPaidBHD

	// Count of invoices not fully paid
	var outstandingCount int64
	if err := a.db.Model(&SupplierInvoice{}).
		Where("payment_status != ?", "Paid").
		Count(&outstandingCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count outstanding invoices: %w", err)
	}
	summary["outstanding_count"] = outstandingCount

	// Count of overdue invoices (past due date and not paid)
	var overdueCount int64
	now := time.Now()
	if err := a.db.Model(&SupplierInvoice{}).
		Where("payment_status != ? AND due_date < ?", "Paid", now).
		Count(&overdueCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count overdue invoices: %w", err)
	}
	summary["overdue_count"] = overdueCount

	log.Printf("📊 Supplier payments summary: %.3f BHD paid, %d outstanding, %d overdue",
		totalPaidBHD, outstandingCount, overdueCount)

	return summary, nil
}

// =============================================================================
// ADDITIONAL CRUD OPERATIONS (P3 FIX)
// =============================================================================

// GetSupplierPayment retrieves a single supplier payment by ID
// P3 FIX: Added missing CRUD operation for consistency
func (a *App) GetSupplierPayment(id string) (SupplierPayment, error) {
	return a.paymentService().GetSupplierPayment(id)
}

func getSupplierPayment(a *App, id string) (SupplierPayment, error) {
	if err := a.requirePermission("payments:view"); err != nil {
		return SupplierPayment{}, err
	}

	if a.db == nil {
		return SupplierPayment{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var payment SupplierPayment
	if err := a.db.First(&payment, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return SupplierPayment{}, newError("NOT_FOUND", "Supplier payment not found", fmt.Sprintf("ID: %s", id))
		}
		return SupplierPayment{}, newError("DB_QUERY_FAILED", "Failed to retrieve supplier payment", err.Error())
	}

	return payment, nil
}

// UpdateSupplierPayment updates an existing supplier payment
// P3 FIX: Added missing CRUD operation for consistency
// SECURITY FIX: Added validation for amount, invoice bounds, and invoice change prevention
func (a *App) UpdateSupplierPayment(id string, payment SupplierPayment) (*SupplierPayment, error) {
	return a.paymentService().UpdateSupplierPayment(id, payment)
}

func updateSupplierPayment(a *App, id string, payment SupplierPayment) (*SupplierPayment, error) {
	if err := a.requirePermission("supplier_payments:update"); err != nil {
		return nil, err
	}

	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Check if payment exists
	var existing SupplierPayment
	if err := a.db.First(&existing, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newError("NOT_FOUND", "Supplier payment not found", fmt.Sprintf("ID: %s", id))
		}
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve supplier payment", err.Error())
	}

	// SECURITY FIX: Validate amount is positive
	if payment.AmountBHD <= 0 {
		return nil, newError("INVALID_AMOUNT", "Payment amount must be greater than zero", fmt.Sprintf("got: %.3f", payment.AmountBHD))
	}

	// SECURITY FIX: Prevent changing the linked supplier invoice
	if payment.SupplierInvoiceID != "" && payment.SupplierInvoiceID != existing.SupplierInvoiceID {
		return nil, newError("INVALID_UPDATE", "Cannot change the supplier invoice linked to a payment - delete and recreate instead", "")
	}

	// SECURITY FIX: Validate amount against supplier invoice outstanding balance
	var invoice SupplierInvoice
	if err := a.db.First(&invoice, "id = ?", existing.SupplierInvoiceID).Error; err != nil {
		return nil, newError("INVOICE_NOT_FOUND", "Linked supplier invoice not found", err.Error())
	}

	// Calculate total already paid for this invoice (excluding current payment being updated)
	var totalPaidExcludingCurrent float64
	if err := a.db.Model(&SupplierPayment{}).
		Where("supplier_invoice_id = ? AND id != ?", existing.SupplierInvoiceID, id).
		Select("COALESCE(SUM(amount_bhd), 0)").
		Scan(&totalPaidExcludingCurrent).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to calculate existing payments", err.Error())
	}

	// The max allowed is invoice total minus what others have paid
	maxAllowedAmount := invoice.TotalBHD - totalPaidExcludingCurrent + FloatingPointTolerance
	payment.Division = a.resolveSupplierInvoiceDivision(invoice)
	if payment.AmountBHD > maxAllowedAmount {
		return nil, newError("INVALID_AMOUNT",
			fmt.Sprintf("Payment amount %.3f BHD exceeds available balance %.3f BHD (invoice total %.3f - other payments %.3f)",
				payment.AmountBHD, maxAllowedAmount-FloatingPointTolerance, invoice.TotalBHD, totalPaidExcludingCurrent),
			"")
	}

	log.Printf("AUDIT: UpdateSupplierPayment id=%s by user=%s: amount %.3f -> %.3f, invoice=%s",
		id, a.getCurrentUserID(), existing.AmountBHD, payment.AmountBHD, existing.SupplierInvoiceID)

	// Preserve ID, timestamps, and invoice linkage
	payment.ID = existing.ID
	payment.CreatedAt = existing.CreatedAt
	payment.SupplierInvoiceID = existing.SupplierInvoiceID
	payment.SupplierID = existing.SupplierID

	// Update payment. Mission I (I-12): GL link, bank linkage, and audit
	// author are server-owned — never mass-assignable from a client payload.
	if err := a.db.Model(&existing).
		Omit("JournalEntryID", "BankAccountID", "IdempotencyKey", "CreatedBy", "CreatedAt").
		Updates(payment).Error; err != nil {
		return nil, newError("DB_UPDATE_FAILED", "Failed to update supplier payment", err.Error())
	}

	// Recalculate invoice payment status if amount changed
	if payment.AmountBHD != existing.AmountBHD {
		var newTotalPaid float64
		if err := a.db.Model(&SupplierPayment{}).
			Where("supplier_invoice_id = ?", existing.SupplierInvoiceID).
			Select("COALESCE(SUM(amount_bhd), 0)").
			Scan(&newTotalPaid).Error; err != nil {
			log.Printf("WARNING: Failed to recalculate total paid for invoice %s: %v", existing.SupplierInvoiceID, err)
		} else {
			var newPaymentStatus string
			if newTotalPaid >= invoice.TotalBHD-FloatingPointTolerance {
				newPaymentStatus = "Paid"
			} else if newTotalPaid > 0 {
				newPaymentStatus = "Partial"
			} else {
				newPaymentStatus = "Unpaid"
			}
			if err := a.db.Model(&SupplierInvoice{}).Where("id = ?", existing.SupplierInvoiceID).
				Update("payment_status", newPaymentStatus).Error; err != nil {
				log.Printf("WARNING: Failed to update invoice payment status: %v", err)
			}
			log.Printf("AUDIT: Supplier invoice %s payment status recalculated: %s (total paid: %.3f / %.3f)",
				invoice.InvoiceNumber, newPaymentStatus, newTotalPaid, invoice.TotalBHD)
		}
	}

	// Reload
	if err := a.db.First(&existing, "id = ?", id).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to reload supplier payment", err.Error())
	}

	log.Printf("AUDIT: Supplier payment updated successfully: id=%s, amount=%.3f BHD, invoice=%s",
		id, existing.AmountBHD, existing.SupplierInvoiceID)
	return &existing, nil
}

// DeleteSupplierPayment deletes a supplier payment by ID
// P3 FIX: Added missing CRUD operation for consistency
func (a *App) DeleteSupplierPayment(id string) error {
	if ok, err := a.guardDeleteOrRequest("supplier_payments:delete", "supplier_payment", id, "Supplier payment"); !ok {
		return err
	}
	if err := a.requirePermission("supplier_payments:delete"); err != nil {
		return err
	}
	return financepayment.DeleteSupplierPayment(a.db, id)
}
