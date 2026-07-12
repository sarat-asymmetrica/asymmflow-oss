// Payment deletion: the transactional delete + invoice-balance rollback
// logic, moved from the trading root in Wave 6 (Mission A.2). The host
// keeps the delete guard, RBAC check, and audit sink; the balance
// restoration, status re-derivation, and the delete itself live here.
package payment

import (
	"fmt"
	"log"
	"math"
	"time"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/finance"
	"ph_holdings_app/pkg/kernel/apperr"
)

// bhdPrecisionMultiplier mirrors the host's 3-decimal BHD rounding.
const bhdPrecisionMultiplier = 1000

// DeletePayment deletes a customer payment and restores the invoice's
// outstanding balance and status. audit (may be nil) is called with the
// payment BEFORE deletion — the host logs financial deletions ahead of
// the destructive write.
func DeletePayment(db *gorm.DB, id string, audit func(finance.Payment)) error {
	if db == nil {
		return apperr.New("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get payment details
	var payment finance.Payment
	if err := db.First(&payment, "id = ?", id).Error; err != nil {
		return apperr.New("PAYMENT_NOT_FOUND", "Payment not found", err.Error())
	}

	// Get invoice
	var invoice finance.Invoice
	if err := db.First(&invoice, "id = ?", payment.InvoiceID).Error; err != nil {
		return apperr.New("INVOICE_NOT_FOUND", "Invoice not found", err.Error())
	}

	// Begin transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// SECURITY: Audit log BEFORE deletion (financial transaction)
	if audit != nil {
		audit(payment)
	}

	// Delete payment
	if err := tx.Delete(&payment).Error; err != nil {
		tx.Rollback()
		return apperr.New("DB_DELETE_FAILED", "Failed to delete payment", err.Error())
	}

	// Restore invoice outstanding balance
	newOutstanding := invoice.OutstandingBHD + payment.AmountBHD
	newOutstanding = math.Round(newOutstanding*bhdPrecisionMultiplier) / bhdPrecisionMultiplier
	if newOutstanding < 0 {
		newOutstanding = 0
	}

	// Mission I (I-08): status derived with the same settlement math the host's
	// customer_invoice_payment_policy applies (PH parity). The old
	// remaining-payments heuristic reverted to "Sent" even when a credit note
	// had already reduced the balance below the grand total.
	var newStatus string
	switch {
	case newOutstanding <= floatingPointTolerance:
		newStatus = "Paid"
	case invoice.GrandTotalBHD-newOutstanding > floatingPointTolerance:
		newStatus = "PartiallyPaid"
	case !invoice.DueDate.IsZero() && time.Now().After(invoice.DueDate):
		newStatus = "Overdue"
	default:
		newStatus = "Sent"
	}

	// Update invoice
	if err := tx.Model(&invoice).Updates(map[string]any{
		"outstanding_bhd": newOutstanding,
		"status":          newStatus,
	}).Error; err != nil {
		tx.Rollback()
		return apperr.New("DB_UPDATE_FAILED", "Failed to update invoice", err.Error())
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return apperr.New("DB_COMMIT_FAILED", "Failed to commit payment deletion", err.Error())
	}

	log.Printf("🗑️ Payment deleted: %s BHD for Invoice %s", fmt.Sprintf("%.3f", payment.AmountBHD), payment.InvoiceNumber)
	log.Printf("📊 Invoice %s restored: Outstanding %.3f BHD → %.3f BHD, Status: %s",
		invoice.InvoiceNumber, invoice.OutstandingBHD, newOutstanding, newStatus)

	return nil
}

// floatingPointTolerance mirrors the host's BHD float comparison window.
const floatingPointTolerance = 0.001

// DeleteSupplierPayment deletes a supplier payment inside a transaction
// and re-derives the supplier invoice's payment_status from the payments
// that remain — the same Paid/Partial/Unpaid derivation the update path
// uses. (W6 fix: the historical rollback targeted amount_paid_bhd, a
// column no model or migration ever defined, so the delete always
// failed; Commander-authorized in-wave.)
func DeleteSupplierPayment(db *gorm.DB, id string) error {
	if db == nil {
		return apperr.New("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// P0 FIX: Use transaction to handle invoice balance rollback
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get payment first
	var payment finance.SupplierPayment
	if err := tx.First(&payment, "id = ?", id).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return apperr.New("NOT_FOUND", "Supplier payment not found", fmt.Sprintf("ID: %s", id))
		}
		return apperr.New("DB_QUERY_FAILED", "Failed to retrieve supplier payment", err.Error())
	}

	// Delete payment
	if err := tx.Delete(&payment).Error; err != nil {
		tx.Rollback()
		return apperr.New("DB_DELETE_FAILED", "Failed to delete supplier payment", err.Error())
	}

	// Re-derive the invoice payment status from the payments that remain.
	// An orphaned payment (invoice already gone) stays deletable — the
	// historical rollback UPDATE was a silent no-op in that case.
	var invoice finance.SupplierInvoice
	invoiceErr := tx.First(&invoice, "id = ?", payment.SupplierInvoiceID).Error
	if invoiceErr != nil && invoiceErr != gorm.ErrRecordNotFound {
		tx.Rollback()
		return apperr.New("DB_QUERY_FAILED", "Failed to retrieve supplier invoice", invoiceErr.Error())
	}
	var totalPaid float64
	if err := tx.Model(&finance.SupplierPayment{}).
		Where("supplier_invoice_id = ?", payment.SupplierInvoiceID).
		Select("COALESCE(SUM(amount_bhd), 0)").
		Scan(&totalPaid).Error; err != nil {
		tx.Rollback()
		return apperr.New("DB_QUERY_FAILED", "Failed to recalculate total payments", err.Error())
	}
	if invoiceErr == nil {
		var newPaymentStatus string
		switch {
		case totalPaid >= invoice.TotalBHD-floatingPointTolerance:
			newPaymentStatus = "Paid"
		case totalPaid > 0:
			newPaymentStatus = "Partial"
		default:
			newPaymentStatus = "Unpaid"
		}
		if err := tx.Model(&finance.SupplierInvoice{}).
			Where("id = ?", payment.SupplierInvoiceID).
			Update("payment_status", newPaymentStatus).Error; err != nil {
			tx.Rollback()
			return apperr.New("DB_UPDATE_FAILED", "Failed to update invoice payment status", err.Error())
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return apperr.New("DB_COMMIT_FAILED", "Failed to commit transaction", err.Error())
	}

	log.Printf("🗑️ Deleted Supplier Payment: %s (rolled back %.3f BHD)", id, payment.AmountBHD)
	return nil
}
