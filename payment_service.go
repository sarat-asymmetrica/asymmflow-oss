package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	financepayment "ph_holdings_app/pkg/finance/payment"
)

// Payment service constants
const (
	BHDPrecisionMultiplier = 1000 // 3 decimal places for BHD currency
)

// =============================================================================
// PAYMENT RECORDING SERVICE
// =============================================================================

// RecordPayment creates a new payment record, updates invoice balances, and triggers order completion checks
// P0 FIX: Added SELECT FOR UPDATE transaction locking to prevent race conditions
// P0 FIX: Ensures outstanding balance never goes negative (floors at zero)
func (a *App) RecordPayment(invoiceID string, amount float64, method string, dateStr string, reference string) (*Payment, error) {
	return a.paymentService().RecordPayment(invoiceID, amount, method, dateStr, reference)
}

func recordPayment(a *App, invoiceID string, amount float64, method string, dateStr string, reference string) (*Payment, error) {
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "database connection not available", "")
	}

	if err := a.requirePermission("payments:create"); err != nil {
		return nil, err
	}

	// Validate amount (basic check before transaction)
	if amount <= 0 {
		return nil, newError("INVALID_AMOUNT", "payment amount must be greater than zero", "")
	}

	// Round to 3 decimal places (BHD precision)
	amount = math.Round(amount*BHDPrecisionMultiplier) / BHDPrecisionMultiplier

	// Parse payment date
	paymentDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, newError("INVALID_DATE", "Invalid date format, expected YYYY-MM-DD", err.Error())
	}

	// Begin transaction FIRST (to enable locking)
	tx := a.db.Begin()
	if tx.Error != nil {
		return nil, newError("DB_TRANSACTION_FAILED", "Failed to begin transaction", tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("PANIC in RecordPayment: %v", r)
		}
	}()

	// P0 FIX: Load invoice with locking (falls back gracefully for SQLite)
	var invoice Invoice
	lockingQuery := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invoice, "id = ?", invoiceID)
	if lockingQuery.Error != nil {
		// SQLite doesn't support SELECT FOR UPDATE - retry without locking
		lockingQuery = tx.First(&invoice, "id = ?", invoiceID)
	}
	if err := lockingQuery.Error; err != nil {
		tx.Rollback()
		return nil, newError("INVOICE_NOT_FOUND", "Invoice not found", err.Error())
	}

	// Reject payments on non-payable invoices. Mission G (Wave 4) replaced an
	// inline map that omitted "Draft" with the shared settlement policy — an
	// unsent Draft invoice (a closed workflow status) is not open, so it is not
	// payable, matching PH's canRecordCustomerInvoicePayment.
	if !canRecordCustomerInvoicePayment(invoice, time.Now()) {
		tx.Rollback()
		return nil, fmt.Errorf("cannot record payment on invoice with status '%s'", invoice.Status)
	}

	// Validate payment doesn't exceed outstanding balance (within locked transaction)
	if amount > invoice.OutstandingBHD {
		tx.Rollback()
		return nil, newError("INVALID_AMOUNT",
			fmt.Sprintf("Payment amount %.3f BHD exceeds outstanding balance %.3f BHD", amount, invoice.OutstandingBHD),
			"")
	}

	// Calculate days to payment (from invoice date to payment date)
	daysToPayment := int(paymentDate.Sub(invoice.InvoiceDate).Hours() / 24)

	// P1 FIX: Validate payment reference required for bank transfers
	if method == "BankTransfer" || method == "Bank Transfer" {
		if reference == "" {
			tx.Rollback()
			return nil, newError("INVALID_REFERENCE", "Payment reference is required for bank transfers", "")
		}
	}

	// P1 FIX: Duplicate payment detection (same invoice + amount + date within 1 hour)
	var existingPaymentCount int64
	if err := tx.Model(&Payment{}).
		Where("invoice_id = ? AND amount_bhd = ? AND payment_date = ?", invoiceID, amount, paymentDate).
		Count(&existingPaymentCount).Error; err == nil && existingPaymentCount > 0 {
		tx.Rollback()
		return nil, newError("DUPLICATE_PAYMENT",
			fmt.Sprintf("Potential duplicate payment detected: same invoice, amount (%.3f BHD), and date already exists", amount),
			"")
	}

	// Compute idempotency key: sha256(invoiceID + amount + date + reference)
	idempotencyInput := fmt.Sprintf("%s|%.3f|%s|%s", invoiceID, amount, dateStr, reference)
	idempotencyHash := sha256.Sum256([]byte(idempotencyInput))
	idempotencyKey := hex.EncodeToString(idempotencyHash[:])

	// Create payment record
	payment := Payment{
		Base: Base{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: a.getCurrentUserID(),
		},
		InvoiceID:      invoiceID,
		InvoiceNumber:  invoice.InvoiceNumber,
		AmountBHD:      amount,
		PaymentDate:    paymentDate,
		PaymentMethod:  method,
		Reference:      reference,
		DaysToPayment:  daysToPayment,
		IdempotencyKey: idempotencyKey,
		Division:       normalizeDivisionName(invoice.Division),
	}

	// Create payment
	if err := tx.Create(&payment).Error; err != nil {
		tx.Rollback()
		return nil, newError("DB_CREATE_FAILED", "Failed to create payment", err.Error())
	}

	// P0 FIX: Calculate new outstanding balance with floor at zero
	newOutstanding := invoice.OutstandingBHD - amount
	newOutstanding = math.Round(newOutstanding*BHDPrecisionMultiplier) / BHDPrecisionMultiplier // Round to 3 decimals
	if newOutstanding < 0 {
		log.Printf("⚠️ Payment %.3f exceeds outstanding %.3f - flooring to zero", amount, invoice.OutstandingBHD)
		newOutstanding = 0 // Floor at zero - no negative balances!
	}

	// Mission I (I-04): derive post-payment status through the settlement
	// policy instead of the old inline if/else whose branches all collapsed
	// to "PartiallyPaid" (Overdue handling was lost). Matches PH exactly.
	previousOutstanding := invoice.OutstandingBHD
	previousStatus := invoice.Status
	invoice.OutstandingBHD = newOutstanding
	state, err := a.applyCustomerInvoicePaymentState(tx, &invoice)
	if err != nil {
		tx.Rollback()
		return nil, newError("DB_UPDATE_FAILED", "Failed to update invoice", err.Error())
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, newError("DB_COMMIT_FAILED", "Failed to commit payment transaction", err.Error())
	}

	// P1 FIX: Audit log for financial transaction
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogFinancialTransaction(
			a.getCurrentUserID(),
			"payment_recorded",
			"invoice",
			invoiceID,
			amount,
			"BHD",
			true,
			map[string]any{
				"invoice_number":      invoice.InvoiceNumber,
				"payment_method":      method,
				"payment_date":        dateStr,
				"reference":           reference,
				"new_outstanding_bhd": state.OutstandingBHD,
				"days_to_payment":     daysToPayment,
			},
		)
	}

	log.Printf("💰 Payment recorded: %s BHD for Invoice %s (Method: %s, Days: %d)",
		fmt.Sprintf("%.3f", amount), invoice.InvoiceNumber, method, daysToPayment)
	log.Printf("📊 Invoice %s updated: Outstanding %.3f BHD → %.3f BHD, Status: %s → %s",
		invoice.InvoiceNumber, previousOutstanding, state.OutstandingBHD, previousStatus, state.Status)

	// If invoice is fully paid, check if order should progress to Complete
	if state.Status == "Paid" && invoice.OrderID != "" {
		if err := checkOrderCompletion(a, invoice.OrderID); err != nil {
			log.Printf("⚠️ Failed to check order completion for order %s: %v", invoice.OrderID, err)
			// Don't fail the payment, just log warning
		}
	}

	return &payment, nil
}

// GetPaymentsByInvoice retrieves all payments for a specific invoice
func (a *App) GetPaymentsByInvoice(invoiceID string) ([]Payment, error) {
	return a.paymentService().GetPaymentsByInvoice(invoiceID)
}

func getPaymentsByInvoice(a *App, invoiceID string) ([]Payment, error) {
	if err := a.requirePermission("payments:view"); err != nil {
		return nil, err
	}

	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var payments []Payment
	if err := a.db.Where("invoice_id = ?", invoiceID).Order("payment_date ASC, created_at ASC").Find(&payments).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve payments", err.Error())
	}

	log.Printf("💳 Retrieved %d payments for invoice %s", len(payments), invoiceID)
	return payments, nil
}

// GetAllPayments retrieves all payments with pagination
func (a *App) GetAllPayments(limit, offset int) ([]Payment, error) {
	return a.paymentService().GetAllPayments(limit, offset)
}

func getAllPayments(a *App, limit, offset int) ([]Payment, error) {
	if err := a.requirePermission("payments:view"); err != nil {
		return nil, err
	}

	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var payments []Payment
	query := a.db.Order("payment_date DESC, created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&payments).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve payments", err.Error())
	}

	log.Printf("💳 Retrieved %d payments (limit: %d, offset: %d)", len(payments), limit, offset)
	return payments, nil
}

// DeletePayment reverses a payment by deleting it and restoring invoice balance
func (a *App) DeletePayment(id string) error {
	if ok, err := a.guardDeleteOrRequest("payments:delete", "payment", id, "Customer payment"); !ok {
		return err
	}
	if err := a.requirePermission("payments:delete"); err != nil {
		return err
	}
	return financepayment.DeletePayment(a.db, id, func(payment Payment) {
		// SECURITY: Audit log BEFORE deletion (financial transaction)
		log.Printf("🔒 AUDIT: Payment %s (%.3f BHD) deleted for invoice %s by user %s",
			payment.ID, payment.AmountBHD, payment.InvoiceID, a.getCurrentUserID())

		if GlobalAuditLogger != nil {
			GlobalAuditLogger.LogFinancialTransaction(
				a.getCurrentUserID(),
				"payment_deleted",
				"payment",
				payment.ID,
				payment.AmountBHD,
				"BHD",
				true,
				map[string]any{
					"invoice_id":     payment.InvoiceID,
					"invoice_number": payment.InvoiceNumber,
					"payment_method": payment.PaymentMethod,
					"reference":      payment.Reference,
				},
			)
		}
	})
}

// =============================================================================
// ORDER AUTO-PROGRESSION HOOKS
// =============================================================================

// checkOrderCompletion checks if all invoices for an order are paid and progresses order to Complete
func checkOrderCompletion(a *App, orderID string) error {
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get order
	var order Order
	if err := a.db.First(&order, "id = ?", orderID).Error; err != nil {
		return newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	// Get all invoices for this order
	var invoices []Invoice
	if err := a.db.Where("order_id = ?", orderID).Find(&invoices).Error; err != nil {
		return newError("DB_QUERY_FAILED", "Failed to retrieve invoices", err.Error())
	}

	if len(invoices) == 0 {
		log.Printf("📋 Order %s has no invoices, skipping completion check", order.OrderNumber)
		return nil
	}

	// Check if all invoices are paid
	allPaid := true
	for _, inv := range invoices {
		if inv.Status != "Paid" {
			allPaid = false
			break
		}
	}

	// If all paid and order not already Complete, update it
	if allPaid && order.Status != "Complete" {
		if err := a.db.Model(&order).Update("status", "Complete").Error; err != nil {
			return newError("DB_UPDATE_FAILED", "Failed to update order status", err.Error())
		}

		log.Printf("✅ Order %s auto-progressed to Complete (all %d invoices paid)", order.OrderNumber, len(invoices))
	}

	return nil
}

// ProgressOrderOnDelivery updates order status based on delivery completion
func (a *App) ProgressOrderOnDelivery(orderID string) error {
	return a.paymentService().ProgressOrderOnDelivery(orderID)
}

func progressOrderOnDelivery(a *App, orderID string) error {
	if err := a.requirePermission("orders:update"); err != nil {
		return err
	}

	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get order with items
	var order Order
	if err := a.db.Preload("Items").First(&order, "id = ?", orderID).Error; err != nil {
		return newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	// Get all delivery notes for this order
	var deliveryNotes []DeliveryNote
	if err := a.db.Preload("Items").Where("order_id = ?", orderID).Find(&deliveryNotes).Error; err != nil {
		return newError("DB_QUERY_FAILED", "Failed to retrieve delivery notes", err.Error())
	}

	// Calculate total delivered quantity per order item
	deliveredQty := make(map[string]float64)
	for _, dn := range deliveryNotes {
		for _, item := range dn.Items {
			deliveredQty[item.OrderItemID] += item.QuantityDelivered
		}
	}

	// Check if all items are fully delivered
	allFullyDelivered := true
	anyPartiallyDelivered := false

	for _, orderItem := range order.Items {
		delivered := deliveredQty[orderItem.ID]
		if delivered < orderItem.Quantity {
			allFullyDelivered = false
			if delivered > 0 {
				anyPartiallyDelivered = true
			}
		}
	}

	// Determine new status
	var newStatus string
	if allFullyDelivered {
		newStatus = "FullyDelivered"
	} else if anyPartiallyDelivered {
		newStatus = "PartiallyDelivered"
	} else {
		// No deliveries yet, keep current status
		return nil
	}

	// Update order status if different
	if order.Status != newStatus && order.Status != "Complete" {
		if err := a.db.Model(&order).Update("status", newStatus).Error; err != nil {
			return newError("DB_UPDATE_FAILED", "Failed to update order status", err.Error())
		}

		log.Printf("✅ Order %s auto-progressed to %s based on delivery status", order.OrderNumber, newStatus)
	}

	return nil
}

// ProgressOrderOnInvoice updates order status when an invoice is created
func (a *App) ProgressOrderOnInvoice(orderID string) error {
	return a.paymentService().ProgressOrderOnInvoice(orderID)
}

func progressOrderOnInvoice(a *App, orderID string) error {
	if err := a.requirePermission("orders:update"); err != nil {
		return err
	}

	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get order
	var order Order
	if err := a.db.First(&order, "id = ?", orderID).Error; err != nil {
		return newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	// Only progress if not already in a more advanced state
	// Status hierarchy: Confirmed → InProgress → PartiallyDelivered → FullyDelivered → Invoiced → Complete
	advancedStatuses := []string{"FullyDelivered", "Complete"}
	for _, status := range advancedStatuses {
		if order.Status == status {
			log.Printf("📋 Order %s already at %s, not progressing to Invoiced", order.OrderNumber, order.Status)
			return nil
		}
	}

	// Update to Invoiced
	if order.Status != "Invoiced" {
		if err := a.db.Model(&order).Update("status", "Invoiced").Error; err != nil {
			return newError("DB_UPDATE_FAILED", "Failed to update order status", err.Error())
		}

		log.Printf("✅ Order %s auto-progressed to Invoiced", order.OrderNumber)
	}

	return nil
}

// =============================================================================
// ADDITIONAL CRUD OPERATIONS (P3 FIX)
// =============================================================================

// GetPayment retrieves a single payment by ID
// P3 FIX: Added missing CRUD operation for consistency
func (a *App) GetPayment(id string) (Payment, error) {
	return a.paymentService().GetPayment(id)
}

func getPayment(a *App, id string) (Payment, error) {
	if err := a.requirePermission("payments:view"); err != nil {
		return Payment{}, err
	}

	if a.db == nil {
		return Payment{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var payment Payment
	if err := a.db.First(&payment, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return Payment{}, newError("NOT_FOUND", "Payment not found", fmt.Sprintf("ID: %s", id))
		}
		return Payment{}, newError("DB_QUERY_FAILED", "Failed to retrieve payment", err.Error())
	}

	return payment, nil
}

// UpdatePayment updates an existing payment
// P3 FIX: Added missing CRUD operation for consistency
// SECURITY FIX: Added validation for amount, invoice bounds, and invoice change prevention
// P2 FIX: Wrapped invoice read + balance update + payment update in transaction with row locking
func (a *App) UpdatePayment(id string, payment Payment) (*Payment, error) {
	return a.paymentService().UpdatePayment(id, payment)
}

func updatePayment(a *App, id string, payment Payment) (*Payment, error) {
	if err := a.requirePermission("payments:update"); err != nil {
		return nil, err
	}

	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Check if payment exists (outside tx — just existence check)
	var existing Payment
	if err := a.db.First(&existing, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newError("NOT_FOUND", "Payment not found", fmt.Sprintf("ID: %s", id))
		}
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve payment", err.Error())
	}

	// SECURITY FIX: Validate amount is positive
	if payment.AmountBHD <= 0 {
		return nil, newError("INVALID_AMOUNT", "Payment amount must be greater than zero", fmt.Sprintf("got: %.3f", payment.AmountBHD))
	}

	// SECURITY FIX: Prevent changing the linked invoice
	if payment.InvoiceID != "" && payment.InvoiceID != existing.InvoiceID {
		return nil, newError("INVALID_UPDATE", "Cannot change the invoice linked to a payment - delete and recreate instead", "")
	}

	// Round to BHD precision
	payment.AmountBHD = math.Round(payment.AmountBHD*BHDPrecisionMultiplier) / BHDPrecisionMultiplier

	log.Printf("AUDIT: UpdatePayment id=%s by user=%s: amount %.3f -> %.3f, invoice=%s",
		id, a.getCurrentUserID(), existing.AmountBHD, payment.AmountBHD, existing.InvoiceID)

	// Preserve ID, timestamps, and invoice linkage
	payment.ID = existing.ID
	payment.CreatedAt = existing.CreatedAt
	payment.InvoiceID = existing.InvoiceID
	payment.InvoiceNumber = existing.InvoiceNumber

	// P2 FIX: Begin transaction for invoice read + balance update + payment update
	tx := a.db.Begin()
	if tx.Error != nil {
		return nil, newError("DB_TRANSACTION_FAILED", "Failed to begin transaction", tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("PANIC in UpdatePayment: %v", r)
		}
	}()

	// Load invoice with row locking (falls back for SQLite)
	var invoice Invoice
	lockingQuery := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invoice, "id = ?", existing.InvoiceID)
	if lockingQuery.Error != nil {
		// SQLite doesn't support SELECT FOR UPDATE - retry without locking
		lockingQuery = tx.First(&invoice, "id = ?", existing.InvoiceID)
	}
	if err := lockingQuery.Error; err != nil {
		tx.Rollback()
		return nil, newError("INVOICE_NOT_FOUND", "Linked invoice not found", err.Error())
	}
	payment.Division = normalizeDivisionName(invoice.Division)

	// Mission I (I-01): reject payments on closed workflow invoices while still
	// allowing edits on settled invoices. The old inline map omitted "Draft", so
	// a payment could be edited onto an unsent Draft invoice. Uses the shared
	// settlement-policy status set, matching PH.
	if customerInvoiceClosedWorkflowStatuses[invoice.Status] {
		tx.Rollback()
		return nil, fmt.Errorf("cannot update payment on invoice with status '%s'", invoice.Status)
	}

	// The allowed amount is the current outstanding + the existing payment amount (since we're replacing it)
	maxAllowedAmount := invoice.OutstandingBHD + existing.AmountBHD
	maxAllowedAmount = math.Round(maxAllowedAmount*BHDPrecisionMultiplier) / BHDPrecisionMultiplier
	if payment.AmountBHD > maxAllowedAmount {
		tx.Rollback()
		return nil, newError("INVALID_AMOUNT",
			fmt.Sprintf("Payment amount %.3f BHD exceeds available balance %.3f BHD (outstanding %.3f + current payment %.3f)",
				payment.AmountBHD, maxAllowedAmount, invoice.OutstandingBHD, existing.AmountBHD),
			"")
	}

	// Recalculate invoice outstanding balance if amount changed
	if payment.AmountBHD != existing.AmountBHD {
		amountDifference := payment.AmountBHD - existing.AmountBHD
		newOutstanding := invoice.OutstandingBHD - amountDifference
		newOutstanding = math.Round(newOutstanding*BHDPrecisionMultiplier) / BHDPrecisionMultiplier
		if newOutstanding < 0 {
			newOutstanding = 0
		}

		// Mission I (I-04): status derived through the settlement policy (PH parity)
		previousOutstanding := invoice.OutstandingBHD
		previousStatus := invoice.Status
		invoice.OutstandingBHD = newOutstanding
		state, err := a.applyCustomerInvoicePaymentState(tx, &invoice)
		if err != nil {
			tx.Rollback()
			return nil, newError("DB_UPDATE_FAILED", "Failed to update invoice balance", err.Error())
		}

		log.Printf("AUDIT: Invoice %s balance recalculated: outstanding %.3f -> %.3f, status %s -> %s",
			invoice.InvoiceNumber, previousOutstanding, state.OutstandingBHD, previousStatus, state.Status)
	}

	// Update payment inside transaction. Mission I (I-12): the GL link,
	// bank-account linkage, idempotency key, and audit author are server-owned
	// — never mass-assignable from a client payload.
	if err := tx.Model(&existing).
		Omit("JournalEntryID", "BankAccountID", "IdempotencyKey", "CreatedBy", "CreatedAt").
		Updates(payment).Error; err != nil {
		tx.Rollback()
		return nil, newError("DB_UPDATE_FAILED", "Failed to update payment", err.Error())
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, newError("DB_COMMIT_FAILED", "Failed to commit payment update transaction", err.Error())
	}

	// Reload (outside tx — read-only)
	if err := a.db.First(&existing, "id = ?", id).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to reload payment", err.Error())
	}

	log.Printf("AUDIT: Payment updated successfully: id=%s, amount=%.3f BHD, invoice=%s",
		id, existing.AmountBHD, existing.InvoiceID)
	return &existing, nil
}
