// Customer invoice deletion, moved from the trading root in Wave 6
// (Mission A.2). The audit-preservation rules live here: terminal-state
// invoices (Paid/Void/Cancelled) and invoices with payment history are
// never deletable. The quantity-invoiced reversal on order items is
// fulfillment bookkeeping on CRM models — finance does not import crm,
// so the host passes it in as a closure.
package invoice

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/finance"
)

// Delete removes a customer invoice (items cascade). It refuses
// terminal-state invoices and invoices with payment history.
// reverseInvoicedQuantities (may be nil) is called with the loaded
// invoice before deletion so the host can roll back order-item
// quantity_invoiced bookkeeping.
func Delete(db *gorm.DB, id string, reverseInvoicedQuantities func(finance.Invoice)) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Verify invoice exists
	var invoice finance.Invoice
	if err := db.Preload("Items").Where("id = ?", id).First(&invoice).Error; err != nil {
		return fmt.Errorf("invoice not found: %w", err)
	}

	// Block deletion of terminal-state invoices (audit trail preservation)
	undeletableStatuses := map[string]bool{"Paid": true, "Void": true, "Cancelled": true}
	if undeletableStatuses[invoice.Status] {
		return fmt.Errorf("cannot delete %s invoice: %s (terminal state — preserved for audit)", invoice.Status, invoice.InvoiceNumber)
	}

	// SECURITY: Block deletion if any payment history exists (audit requirement)
	var paymentCount int64
	db.Model(&finance.Payment{}).Where("invoice_id = ?", id).Count(&paymentCount)
	if paymentCount > 0 {
		return fmt.Errorf("cannot delete invoice with payment history (%d payment(s) - audit requirement)", paymentCount)
	}

	// Reverse QuantityInvoiced on order items if order exists
	if invoice.OrderID != "" && reverseInvoicedQuantities != nil {
		reverseInvoicedQuantities(invoice)
	}

	// Delete invoice (GORM will cascade delete items)
	if err := db.Delete(&invoice).Error; err != nil {
		return fmt.Errorf("failed to delete invoice: %w", err)
	}

	log.Printf("🗑️ Deleted Invoice: %s", invoice.InvoiceNumber)
	return nil
}

// DeleteSupplier removes a supplier invoice (soft delete). Paid invoices
// are never deletable.
func DeleteSupplier(db *gorm.DB, id string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Verify invoice exists and is deletable (not paid)
	var invoice finance.SupplierInvoice
	if err := db.Where("id = ?", id).First(&invoice).Error; err != nil {
		return fmt.Errorf("supplier invoice not found: %w", err)
	}

	if invoice.PaymentStatus == "Paid" {
		return fmt.Errorf("cannot delete paid invoice: %s", invoice.InvoiceNumber)
	}

	if err := db.Delete(&invoice).Error; err != nil {
		return fmt.Errorf("failed to delete supplier invoice: %w", err)
	}

	log.Printf("🗑️ Deleted Supplier Invoice: %s", invoice.InvoiceNumber)
	return nil
}
