package main

import (
	"context"
	"fmt"
	"time"
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

// =============================================================================
// DEAL TIMELINE (Wave 10 B3) - single-row assembly of the deal spine
// RFQ -> Costing -> Offer -> Order -> Delivery -> Invoice -> Paid
// =============================================================================

// DealTimelineNode is one stage of the deal spine (Article I: the human
// document serial leads, never a raw id). RecordID/RecordType exist purely
// for the frontend to deep-link into the source record's own screen; nothing
// here is mutated.
type DealTimelineNode struct {
	Stage      string    `json:"stage"`  // RFQ | Costing | Offer | Order | Delivery | Invoice | Paid
	Serial     string    `json:"serial"` // human document serial; empty if the node was never reached
	Date       time.Time `json:"date"`   // document date; zero value if none
	State      string    `json:"state"`  // done | current | pending | na
	RecordID   string    `json:"record_id,omitempty"`
	RecordType string    `json:"record_type,omitempty"` // opportunity | offer | order | delivery_note | invoice
	Count      int       `json:"count,omitempty"`       // >1 when a stage aggregates multiple records (DNs, invoices)
}

// DealTimeline is the ordered, one-call assembly of a single order's full
// document chain, for the deal-spine stepper (Article I - the signature
// timeline). Read-only: no field here is ever written back to the DB.
type DealTimeline struct {
	OrderID string             `json:"order_id"`
	Nodes   []DealTimelineNode `json:"nodes"`
}

// GetDealTimeline assembles the deal spine for one order in a small, fixed
// number of queries (Order, Offer, Opportunity/RFQ, best-effort Costing
// attachment, DeliveryNotes, Invoices - six queries worst case, fewer when
// upstream links are empty). PAID state is derived with the exact same
// policy function the app already uses on every invoice read
// (hydrateCustomerInvoicesPaymentState, customer_invoice_payment_policy.go),
// applied here in-memory only - this function never writes to the database.
func (a *App) GetDealTimeline(orderID string) (DealTimeline, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return DealTimeline{}, err
	}
	if a.db == nil {
		return DealTimeline{}, fmt.Errorf("database not initialized")
	}
	ctx := context.Background()

	// 1. Order - the seed record.
	var order Order
	if err := a.db.WithContext(ctx).First(&order, "id = ?", orderID).Error; err != nil {
		return DealTimeline{OrderID: orderID, Nodes: []DealTimelineNode{}}, fmt.Errorf("order not found: %w", err)
	}

	return a.buildDealTimeline(ctx, order)
}

// GetDealTimelineByOrderNumber is the same read-only assembly as
// GetDealTimeline, keyed by the human order serial instead of the raw id.
// Exists because some list surfaces (e.g. the customer-orders tab, which
// renders OrderSummary rows carrying only order_number/date/status/total,
// no id) have no order id to deep-link with - this lets them resolve the
// serial they already display into the same one-call timeline without a
// wider surface change.
func (a *App) GetDealTimelineByOrderNumber(orderNumber string) (DealTimeline, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return DealTimeline{}, err
	}
	if a.db == nil {
		return DealTimeline{}, fmt.Errorf("database not initialized")
	}
	ctx := context.Background()

	var order Order
	if err := a.db.WithContext(ctx).First(&order, "order_number = ?", orderNumber).Error; err != nil {
		return DealTimeline{Nodes: []DealTimelineNode{}}, fmt.Errorf("order not found: %w", err)
	}

	return a.buildDealTimeline(ctx, order)
}

// buildDealTimeline assembles the deal spine for an already-loaded order.
func (a *App) buildDealTimeline(ctx context.Context, order Order) (DealTimeline, error) {
	timeline := DealTimeline{OrderID: order.ID, Nodes: []DealTimelineNode{}}

	// 2. Offer, via Order.OfferID.
	var offer *Offer
	if order.OfferID != "" {
		var o Offer
		if err := a.db.WithContext(ctx).First(&o, "id = ?", order.OfferID).Error; err == nil {
			offer = &o
		}
	}

	// 3. Opportunity/RFQ, via Order.RFQID (falls back to Offer.RFQID when the
	// order itself doesn't carry the link).
	var opportunity *Opportunity
	rfqID := order.RFQID
	if rfqID == "" && offer != nil {
		rfqID = offer.RFQID
	}
	if rfqID != "" {
		var opp Opportunity
		if err := a.db.WithContext(ctx).First(&opp, "id = ?", rfqID).Error; err == nil {
			opportunity = &opp
		}
	}

	// 4. Costing - best-effort. There is no first-class per-deal costing
	// entity; CostingSheetAttachment is scoped by a free-form ScopeID that
	// callers set to the opportunity or offer id. We look for either, purely
	// to say "was a costing artifact ever attached to this deal" - never
	// invented, never guessed beyond an exact scope-id match.
	var costingAttachment *CostingSheetAttachment
	{
		scopeIDs := []string{}
		if opportunity != nil {
			scopeIDs = append(scopeIDs, opportunity.ID)
		}
		if offer != nil {
			scopeIDs = append(scopeIDs, offer.ID)
		}
		if len(scopeIDs) > 0 {
			var att CostingSheetAttachment
			if err := a.db.WithContext(ctx).
				Where("scope_id IN ?", scopeIDs).
				Order("created_at DESC").
				First(&att).Error; err == nil {
				costingAttachment = &att
			}
		}
	}

	// 5. Delivery Notes for this order.
	var deliveryNotes []DeliveryNote
	if err := a.db.WithContext(ctx).
		Where("order_id = ?", order.ID).
		Order("delivery_date ASC").
		Find(&deliveryNotes).Error; err != nil {
		deliveryNotes = []DeliveryNote{}
	}

	// 6. Invoices for this order (same WHERE shape as GetInvoicesByOrder,
	// inlined here so this assembly stays under the single "finance:view"
	// permission gate rather than layering invoices:view on top).
	var invoices []Invoice
	if err := a.db.WithContext(ctx).
		Where("order_id = ?", order.ID).
		Order("invoice_date ASC").
		Find(&invoices).Error; err != nil {
		invoices = []Invoice{}
	}

	// Derive settlement status honestly with the exact function every invoice
	// list/detail view already uses on read. This mutates only the in-memory
	// slice - no DB write happens here.
	hydrateCustomerInvoicesPaymentState(invoices)

	// --- Assemble nodes in spine order ---

	// RFQ
	rfqNode := DealTimelineNode{Stage: "RFQ", State: "na"}
	if opportunity != nil {
		serial := opportunity.EHRef
		if serial == "" {
			serial = opportunity.FolderNumber
		}
		rfqNode.Serial = serial
		rfqNode.Date = opportunity.OfferDate
		rfqNode.State = "done"
		rfqNode.RecordID = opportunity.ID
		rfqNode.RecordType = "opportunity"
	}
	timeline.Nodes = append(timeline.Nodes, rfqNode)

	// Costing
	costingNode := DealTimelineNode{Stage: "Costing", State: "na"}
	if costingAttachment != nil {
		serial := costingAttachment.CostingNumber
		if serial == "" {
			serial = costingAttachment.FileName
		}
		costingNode.Serial = serial
		costingNode.Date = costingAttachment.CreatedAt
		costingNode.State = "done"
		costingNode.RecordID = costingAttachment.ID
		costingNode.RecordType = "costing_attachment"
	}
	timeline.Nodes = append(timeline.Nodes, costingNode)

	// Offer
	offerNode := DealTimelineNode{Stage: "Offer", State: "na"}
	if offer != nil {
		offerNode.Serial = offer.OfferNumber
		offerNode.Date = offer.QuotationDate
		offerNode.State = "done"
		offerNode.RecordID = offer.ID
		offerNode.RecordType = "offer"
	}
	timeline.Nodes = append(timeline.Nodes, offerNode)

	// Order - always the seed record, always done.
	timeline.Nodes = append(timeline.Nodes, DealTimelineNode{
		Stage:      "Order",
		Serial:     order.OrderNumber,
		Date:       order.OrderDate,
		State:      "done",
		RecordID:   order.ID,
		RecordType: "order",
	})

	// Delivery - a real future stage: "pending" (not "na") when absent,
	// because an order without a DN yet genuinely has delivery ahead of it.
	deliveryNode := DealTimelineNode{Stage: "Delivery", State: "pending"}
	if len(deliveryNotes) > 0 {
		latest := deliveryNotes[len(deliveryNotes)-1]
		deliveryNode.Serial = latest.DNNumber
		deliveryNode.Date = latest.DeliveryDate
		deliveryNode.State = "done"
		deliveryNode.RecordID = latest.ID
		deliveryNode.RecordType = "delivery_note"
		deliveryNode.Count = len(deliveryNotes)
	}
	timeline.Nodes = append(timeline.Nodes, deliveryNode)

	// Invoice - same "pending, not na" reasoning as Delivery.
	invoiceNode := DealTimelineNode{Stage: "Invoice", State: "pending"}
	allPaid := false
	if len(invoices) > 0 {
		latest := invoices[len(invoices)-1]
		invoiceNode.Serial = latest.InvoiceNumber
		invoiceNode.Date = latest.InvoiceDate
		invoiceNode.State = "done"
		invoiceNode.RecordID = latest.ID
		invoiceNode.RecordType = "invoice"
		invoiceNode.Count = len(invoices)

		allPaid = true
		for _, inv := range invoices {
			if inv.Status != "Paid" {
				allPaid = false
				break
			}
		}
	}
	timeline.Nodes = append(timeline.Nodes, invoiceNode)

	// Paid - the closing state of the spine. "current" marks the deal as
	// actively in flight (invoiced but not yet fully settled); "pending"
	// means invoicing hasn't happened yet, so payment can't be in flight.
	paidNode := DealTimelineNode{Stage: "Paid", State: "pending"}
	if len(invoices) > 0 {
		if allPaid {
			latest := invoices[len(invoices)-1]
			paidNode.State = "done"
			paidNode.Serial = latest.InvoiceNumber
			paidNode.Date = latest.InvoiceDate
			paidNode.RecordID = latest.ID
			paidNode.RecordType = "invoice"
		} else {
			paidNode.State = "current"
		}
	}
	timeline.Nodes = append(timeline.Nodes, paidNode)

	return timeline, nil
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
