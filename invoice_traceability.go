package main

import (
	"context"
	"fmt"
)

// =============================================================================
// INVOICE TRACEABILITY - OPERATIONS PIPELINE FULL AUDIT TRAIL
// =============================================================================

// InvoiceAuditTrail represents complete traceability from RFQ to Invoice
type InvoiceAuditTrail struct {
	Invoice          Invoice           `json:"invoice"`
	RFQ              *Offer            `json:"rfq,omitempty"`     // Original RFQ
	Quote            *Offer            `json:"quote,omitempty"`   // Accepted quote/offer
	Order            *Order            `json:"order,omitempty"`   // Customer order
	PurchaseOrders   []PurchaseOrder   `json:"purchase_orders"`   // Supplier POs (when entities exist)
	SupplierInvoices []SupplierInvoice `json:"supplier_invoices"` // Supplier invoices (when entities exist)
	DeliveryNotes    []DeliveryNote    `json:"delivery_notes"`    // Delivery notes (when entities exist)
}

// NOTE: All entities used by InvoiceAuditTrail already exist:
// - PurchaseOrder: operations_entities.go
// - SupplierInvoice: database.go (lines 851-898)
// - DeliveryNote: database.go (line 352)

// GetInvoiceAuditTrail retrieves complete audit trail for an invoice
func (a *App) GetInvoiceAuditTrail(invoiceID string) (InvoiceAuditTrail, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return InvoiceAuditTrail{}, err
	}
	if a.db == nil {
		return InvoiceAuditTrail{}, fmt.Errorf("database not initialized")
	}
	ctx := context.Background()
	trail := InvoiceAuditTrail{
		PurchaseOrders:   []PurchaseOrder{},
		SupplierInvoices: []SupplierInvoice{},
		DeliveryNotes:    []DeliveryNote{},
	}

	// 1. Load the invoice with items
	if err := a.db.WithContext(ctx).Preload("Items").First(&trail.Invoice, "id = ?", invoiceID).Error; err != nil {
		return trail, fmt.Errorf("invoice not found: %w", err)
	}

	// 2. Load RFQ if linked
	if trail.Invoice.RfqID != "" {
		var rfq Offer
		if err := a.db.WithContext(ctx).Preload("Items").First(&rfq, "id = ?", trail.Invoice.RfqID).Error; err == nil {
			trail.RFQ = &rfq
		}
	}

	// 3. Load Quote/Offer if linked
	if trail.Invoice.QuoteID != "" {
		var quote Offer
		if err := a.db.WithContext(ctx).Preload("Items").First(&quote, "id = ?", trail.Invoice.QuoteID).Error; err == nil {
			trail.Quote = &quote
		}
	}

	// 4. Load Order if linked
	if trail.Invoice.OrderID != "" {
		var order Order
		if err := a.db.WithContext(ctx).Preload("Items").First(&order, "id = ?", trail.Invoice.OrderID).Error; err == nil {
			trail.Order = &order

			// 5. Load Purchase Orders for this order
			if err := a.db.WithContext(ctx).Where("order_id = ?", trail.Order.ID).Find(&trail.PurchaseOrders).Error; err != nil {
				// Non-fatal: POs might not exist yet
				trail.PurchaseOrders = []PurchaseOrder{}
			}

			// 6. Load Supplier Invoices via Purchase Orders
			if len(trail.PurchaseOrders) > 0 {
				var poIDs []string
				for _, po := range trail.PurchaseOrders {
					poIDs = append(poIDs, po.ID)
				}
				if err := a.db.WithContext(ctx).Where("purchase_order_id IN ?", poIDs).Find(&trail.SupplierInvoices).Error; err != nil {
					// Non-fatal: Supplier invoices might not exist yet
					trail.SupplierInvoices = []SupplierInvoice{}
				}
			}

			// 7. Load Delivery Notes
			if err := a.db.WithContext(ctx).Where("order_id = ?", trail.Order.ID).Find(&trail.DeliveryNotes).Error; err != nil {
				// Non-fatal: Delivery notes might not exist yet
				trail.DeliveryNotes = []DeliveryNote{}
			}
		}
	}

	return trail, nil
}

// CalculateInvoiceMargin calculates and updates gross margin for an invoice
func (a *App) CalculateInvoiceMargin(invoiceID string) error {
	if err := a.requirePermission("invoices:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx := context.Background()

	// 1. Load invoice
	var invoice Invoice
	if err := a.db.WithContext(ctx).First(&invoice, "id = ?", invoiceID).Error; err != nil {
		return fmt.Errorf("invoice not found: %w", err)
	}

	// 2. Calculate total supplier cost from SupplierInvoices
	var totalSupplierCost float64

	if invoice.OrderID != "" {
		// Query: SUM(supplier_invoices.total_bhd) via purchase_orders
		err := a.db.WithContext(ctx).
			Table("supplier_invoices").
			Select("COALESCE(SUM(total_bhd), 0)").
			Where("purchase_order_id IN (?)",
				a.db.Table("purchase_orders").Select("id").Where("order_id = ?", invoice.OrderID),
			).
			Scan(&totalSupplierCost).Error

		if err != nil {
			// If query fails (e.g., no POs exist), default to 0
			totalSupplierCost = 0.0
		}
	} else {
		totalSupplierCost = 0.0
	}

	// 3. Calculate margins
	invoice.TotalSupplierCostBHD = totalSupplierCost
	invoice.GrossMarginBHD = invoice.GrandTotalBHD - totalSupplierCost

	if invoice.GrandTotalBHD > 0 {
		invoice.GrossMarginPercent = (invoice.GrossMarginBHD / invoice.GrandTotalBHD) * 100.0
	} else {
		invoice.GrossMarginPercent = 0.0
	}

	// 4. Update invoice
	if err := a.db.WithContext(ctx).Model(&invoice).Updates(map[string]any{
		"total_supplier_cost_bhd": invoice.TotalSupplierCostBHD,
		"gross_margin_bhd":        invoice.GrossMarginBHD,
		"gross_margin_percent":    invoice.GrossMarginPercent,
	}).Error; err != nil {
		return fmt.Errorf("failed to update invoice margins: %w", err)
	}

	return nil
}

// LinkInvoiceToOrder links an invoice to an order
func (a *App) LinkInvoiceToOrder(invoiceID string, orderID string) error {
	if err := a.requirePermission("invoices:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx := context.Background()

	// 1. Verify order exists
	var order Order
	if err := a.db.WithContext(ctx).First(&order, "id = ?", orderID).Error; err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	// 2. Update invoice
	if err := a.db.WithContext(ctx).Model(&Invoice{}).Where("id = ?", invoiceID).Update("order_id", orderID).Error; err != nil {
		return fmt.Errorf("failed to link invoice to order: %w", err)
	}

	// 3. Recalculate margin (now that we have order link)
	if err := a.CalculateInvoiceMargin(invoiceID); err != nil {
		// Log warning but don't fail - margin calculation is non-critical
		fmt.Printf("Warning: Failed to recalculate invoice margin: %v\n", err)
	}

	return nil
}

// LinkInvoiceToRFQ links an invoice to an RFQ/Offer
func (a *App) LinkInvoiceToRFQ(invoiceID string, rfqID string) error {
	if err := a.requirePermission("invoices:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx := context.Background()

	// 1. Verify RFQ exists
	var rfq Offer
	if err := a.db.WithContext(ctx).First(&rfq, "id = ?", rfqID).Error; err != nil {
		return fmt.Errorf("RFQ not found: %w", err)
	}

	// 2. Update invoice
	updates := map[string]any{
		"rfq_id": rfqID,
	}

	// If RFQ has stage "Won" and offer_id, also link as quote
	if rfq.Stage == "Won" {
		updates["quote_id"] = rfqID
	}

	if err := a.db.WithContext(ctx).Model(&Invoice{}).Where("id = ?", invoiceID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to link invoice to RFQ: %w", err)
	}

	return nil
}

// GetInvoicesByOrder retrieves all invoices for an order
func (a *App) GetInvoicesByOrder(orderID string) ([]Invoice, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx := context.Background()
	var invoices []Invoice

	if err := a.db.WithContext(ctx).
		Preload("Items").
		Where("order_id = ?", orderID).
		Order("invoice_date DESC").
		Limit(500).
		Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve invoices: %w", err)
	}

	return invoices, nil
}

// GetInvoicesByRFQ retrieves all invoices linked to an RFQ
func (a *App) GetInvoicesByRFQ(rfqID string) ([]Invoice, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx := context.Background()
	var invoices []Invoice

	if err := a.db.WithContext(ctx).
		Preload("Items").
		Where("rfq_id = ?", rfqID).
		Order("invoice_date DESC").
		Limit(500).
		Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve invoices: %w", err)
	}

	return invoices, nil
}

// RecalculateAllInvoiceMargins recalculates margins for all invoices
// Useful after bulk supplier invoice imports
func (a *App) RecalculateAllInvoiceMargins() (int, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return 0, err
	}
	if a.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	ctx := context.Background()

	var invoices []Invoice
	if err := a.db.WithContext(ctx).Find(&invoices).Error; err != nil {
		return 0, fmt.Errorf("failed to load invoices: %w", err)
	}

	successCount := 0
	for _, invoice := range invoices {
		if err := a.CalculateInvoiceMargin(invoice.ID); err != nil {
			fmt.Printf("Warning: Failed to recalculate margin for invoice %s: %v\n", invoice.InvoiceNumber, err)
			continue
		}
		successCount++
	}

	return successCount, nil
}
