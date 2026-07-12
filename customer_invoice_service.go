package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"

	"ph_holdings_app/pkg/approvals"
	"ph_holdings_app/pkg/documents/numbering"
	financeinvoice "ph_holdings_app/pkg/finance/invoice"
	"ph_holdings_app/pkg/kernel/approval"

	"gorm.io/gorm/clause"

	"ph_holdings_app/pkg/infra/audit"
)

// hmacSaltWarningOnce ensures the missing salt file warning is logged only once
var hmacSaltWarningOnce sync.Once

// computeDocumentHMAC generates an HMAC-SHA256 hash for invoice integrity verification
// Uses globalFieldCrypto.salt as the secret key (avoids disk read + path mismatch issues)
// Falls back to plain SHA-256 if FieldCrypto is not initialized (first-run scenario)
// I1 FIX: Missing salt warning logged only once via sync.Once
func computeDocumentHMAC(invoiceNumber, date string, grandTotal, vat float64) string {
	hashInput := invoiceNumber + "|" + date + "|" + fmt.Sprintf("%.3f", grandTotal) + "|" + fmt.Sprintf("%.3f", vat)

	// Use globalFieldCrypto.salt directly — avoids EvalSymlinks/path mismatch issues
	if globalFieldCrypto != nil {
		globalFieldCrypto.mu.RLock()
		salt := globalFieldCrypto.salt
		globalFieldCrypto.mu.RUnlock()
		if len(salt) > 0 {
			mac := hmac.New(sha256.New, salt)
			mac.Write([]byte(hashInput))
			return hex.EncodeToString(mac.Sum(nil))
		}
	}

	// Fallback to plain SHA-256 if FieldCrypto not available (first-run scenario)
	hmacSaltWarningOnce.Do(func() {
		log.Printf("WARNING: globalFieldCrypto not available — using plain SHA-256 for document hash. Document integrity verification is degraded.")
	})
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

func roundInvoiceMoney(value float64) float64 {
	return math.Round(value*1000) / 1000
}

// InvoiceHashVerification is the result of recomputing an invoice's integrity hash.
type InvoiceHashVerification struct {
	InvoiceID     string `json:"invoice_id"`
	InvoiceNumber string `json:"invoice_number"`
	StoredHash    string `json:"stored_hash"`
	ComputedHash  string `json:"computed_hash"`
	HasHash       bool   `json:"has_hash"`
	Valid         bool   `json:"valid"`
}

// backfillInvoiceHashesInternal (MON-003) fills invoice_hash for invoices that
// have none — e.g. bulk-imported invoices created before hashing existed. It
// runs at startup so it reaches mature client DBs. It MUST only persist a hash
// when the HMAC salt is available; otherwise computeDocumentHMAC would fall
// back to plain SHA-256 and we would poison rows with a hash that later fails
// HMAC verification. No permission check (startup-internal); the exported
// BackfillInvoiceHashes wraps it for the UI.
func (a *App) backfillInvoiceHashesInternal() (int, error) {
	if a.db == nil {
		return 0, nil
	}
	if globalFieldCrypto == nil {
		return 0, nil
	}
	globalFieldCrypto.mu.RLock()
	haveSalt := len(globalFieldCrypto.salt) > 0
	globalFieldCrypto.mu.RUnlock()
	if !haveSalt {
		return 0, nil
	}

	var invoices []Invoice
	if err := a.db.Where("invoice_hash IS NULL OR invoice_hash = ''").Find(&invoices).Error; err != nil {
		return 0, err
	}
	filled := 0
	for i := range invoices {
		inv := &invoices[i]
		hash := computeDocumentHMAC(inv.InvoiceNumber, inv.InvoiceDate.Format("2006-01-02"), inv.GrandTotalBHD, inv.VATBHD)
		if hash == "" {
			continue
		}
		if err := a.db.Model(&Invoice{}).Where("id = ?", inv.ID).Update("invoice_hash", hash).Error; err != nil {
			log.Printf("⚠️ MON-003: failed to backfill hash for invoice %s: %v", inv.InvoiceNumber, err)
			continue
		}
		filled++
	}
	if filled > 0 {
		log.Printf("✅ MON-003: backfilled %d blank invoice hashes (HMAC-SHA256)", filled)
	}
	return filled, nil
}

// BackfillInvoiceHashes is the RBAC-guarded entry point so an admin can fill any
// remaining blank invoice hashes on demand. Returns the number of rows updated.
func (a *App) BackfillInvoiceHashes() (int, error) {
	if err := a.requirePermission("finance:update"); err != nil {
		return 0, err
	}
	return a.backfillInvoiceHashesInternal()
}

// VerifyInvoiceHash (MON-003) recomputes an invoice's integrity hash and reports
// whether the stored hash still matches — surfacing tampering or salt drift.
func (a *App) VerifyInvoiceHash(invoiceID string) (*InvoiceHashVerification, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var inv Invoice
	if err := a.db.First(&inv, "id = ?", strings.TrimSpace(invoiceID)).Error; err != nil {
		return nil, fmt.Errorf("invoice not found: %w", err)
	}
	computed := computeDocumentHMAC(inv.InvoiceNumber, inv.InvoiceDate.Format("2006-01-02"), inv.GrandTotalBHD, inv.VATBHD)
	stored := strings.TrimSpace(inv.InvoiceHash)
	return &InvoiceHashVerification{
		InvoiceID:     inv.ID,
		InvoiceNumber: inv.InvoiceNumber,
		StoredHash:    stored,
		ComputedHash:  computed,
		HasHash:       stored != "",
		Valid:         stored != "" && hmac.Equal([]byte(stored), []byte(computed)),
	}, nil
}

func normalizeOrderItemForInvoice(item OrderItem) (OrderItem, bool) {
	if isSyntheticCommercialSummary(item.Description, item.ProductCode, item.Model, item.Equipment) {
		return item, false
	}

	item.Quantity = roundInvoiceMoney(item.Quantity)
	item.UnitPrice = roundInvoiceMoney(item.UnitPrice)
	item.TotalPrice = roundInvoiceMoney(item.TotalPrice)

	if item.UnitPrice <= 0 && item.Quantity > 0 && item.TotalPrice > 0 {
		item.UnitPrice = roundInvoiceMoney(item.TotalPrice / item.Quantity)
	}
	if item.TotalPrice <= 0 && item.Quantity > 0 && item.UnitPrice > 0 {
		item.TotalPrice = roundInvoiceMoney(item.Quantity * item.UnitPrice)
	}

	item.Description = strings.TrimSpace(item.Description)
	if item.Description == "" {
		switch {
		case strings.TrimSpace(item.Equipment) != "" && strings.TrimSpace(item.Model) != "":
			item.Description = fmt.Sprintf("%s - %s", strings.TrimSpace(item.Equipment), strings.TrimSpace(item.Model))
		case strings.TrimSpace(item.Equipment) != "":
			item.Description = strings.TrimSpace(item.Equipment)
		case strings.TrimSpace(item.Model) != "":
			item.Description = strings.TrimSpace(item.Model)
		case strings.TrimSpace(item.ProductCode) != "":
			item.Description = strings.TrimSpace(item.ProductCode)
		}
	}

	hasCommercialIdentity := item.Description != "" ||
		strings.TrimSpace(item.ProductCode) != "" ||
		strings.TrimSpace(item.Equipment) != "" ||
		strings.TrimSpace(item.Model) != ""

	if !hasCommercialIdentity {
		return item, false
	}
	if item.Quantity <= 0 {
		return item, false
	}
	if item.UnitPrice <= 0 && item.TotalPrice <= 0 {
		return item, false
	}

	return item, true
}

func invoiceOrderItemsFromOrder(order Order) []OrderItem {
	targetTotal := roundInvoiceMoney(order.TotalValueBHD)
	if targetTotal <= 0 {
		targetTotal = roundInvoiceMoney(order.GrandTotalBHD)
	}

	normalized := make([]OrderItem, 0, len(order.Items))
	seen := make(map[string]struct{})
	for _, item := range order.Items {
		clean, ok := normalizeOrderItemForInvoice(item)
		if !ok {
			continue
		}
		signature := strings.Join([]string{
			strings.ToLower(strings.TrimSpace(clean.ProductCode)),
			strings.ToLower(strings.TrimSpace(clean.Description)),
			fmt.Sprintf("%.3f", clean.Quantity),
			fmt.Sprintf("%.3f", clean.UnitPrice),
			fmt.Sprintf("%.3f", clean.TotalPrice),
		}, "|")
		if _, exists := seen[signature]; exists {
			continue
		}
		seen[signature] = struct{}{}
		normalized = append(normalized, clean)
	}

	if len(normalized) == 0 {
		return nil
	}

	if targetTotal > 0 {
		for _, item := range normalized {
			if math.Abs(roundInvoiceMoney(item.TotalPrice)-targetTotal) <= 0.01 {
				return []OrderItem{item}
			}
		}
	}

	sum := 0.0
	for _, item := range normalized {
		sum += item.TotalPrice
	}
	if targetTotal > 0 && math.Abs(roundInvoiceMoney(sum)-targetTotal) <= 0.01 {
		return normalized
	}

	return normalized
}

func defaultInvoiceFieldVisibilityJSON() string {
	return `{"show_fob":false,"show_freight":false,"show_margin":false,"show_cost":false,"show_contact":true,"show_rfq":true,"show_equipment":true,"show_specification":true,"show_detailed_desc":true,"show_country_origin":true,"show_delivery_weeks":true}`
}

// Bound entry point. Mission I (I-11): gated — startup uses the internal.
func (a *App) BackfillInvoiceItemsFromOrders() (map[string]any, error) {
	if err := a.requirePermission("invoices:update"); err != nil {
		return nil, err
	}
	return a.backfillInvoiceItemsFromOrdersInternal()
}

func (a *App) backfillInvoiceItemsFromOrdersInternal() (map[string]any, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	type invoiceShell struct {
		ID string
	}

	var targets []invoiceShell
	if err := a.db.Raw(`
			SELECT i.id
			FROM invoices i
			LEFT JOIN invoice_items ii ON ii.invoice_id = i.id AND ii.deleted_at IS NULL
			WHERE i.deleted_at IS NULL
			  AND COALESCE(i.order_id, '') != ''
			GROUP BY i.id
			HAVING COUNT(ii.id) = 0
		`).Scan(&targets).Error; err != nil {
		return nil, fmt.Errorf("failed to find invoices missing items: %w", err)
	}

	repaired := 0
	skipped := 0
	examples := make([]string, 0, 5)

	for _, shell := range targets {
		didRepair, invoiceNumber, err := a.repairInvoiceItemsFromOrder(shell.ID)
		if err != nil {
			return nil, err
		}
		if !didRepair {
			skipped++
			continue
		}
		repaired++
		if len(examples) < 5 {
			examples = append(examples, invoiceNumber)
		}
	}

	return map[string]any{
		"attempted": len(targets),
		"repaired":  repaired,
		"skipped":   skipped,
		"examples":  examples,
	}, nil
}

// repairInvoiceItemsFromOrder rebuilds one invoice's line items from its linked
// order (PH convergence B1, PH 3c5127b). Shared by the startup backfill and the
// won-import path so an invoice created mid-session is populated inline instead
// of staying hollow until the next restart. Returns didRepair=false (no error)
// when the invoice has no usable order/amounts to repair from.
func (a *App) repairInvoiceItemsFromOrder(invoiceID string) (bool, string, error) {
	var invoice Invoice
	if err := a.db.First(&invoice, "id = ?", invoiceID).Error; err != nil {
		return false, "", fmt.Errorf("failed to load invoice %s: %w", invoiceID, err)
	}

	var order Order
	if err := a.db.Preload("Items").First(&order, "id = ?", invoice.OrderID).Error; err != nil {
		return false, invoice.InvoiceNumber, nil
	}

	items := invoiceOrderItemsFromOrder(order)
	if len(items) == 0 {
		fallbackAmount := roundInvoiceMoney(invoice.SubtotalBHD)
		if fallbackAmount <= 0 {
			fallbackAmount = roundInvoiceMoney(order.TotalValueBHD)
		}
		if fallbackAmount <= 0 {
			fallbackAmount = roundInvoiceMoney(invoice.GrandTotalBHD)
		}
		if fallbackAmount <= 0 {
			return false, invoice.InvoiceNumber, nil
		}
		items = []OrderItem{{
			Base:        Base{ID: uuid.New().String()},
			OrderID:     order.ID,
			LineNumber:  1,
			Description: fmt.Sprintf("Per Order %s", order.OrderNumber),
			Quantity:    1,
			UnitPrice:   fallbackAmount,
			TotalPrice:  fallbackAmount,
		}}
	}

	invoiceItems := make([]DBInvoiceItem, 0, len(items))
	totalSupplierCost := 0.0
	subtotal := 0.0
	for index, item := range items {
		lineTotal := roundInvoiceMoney(item.TotalPrice)
		if lineTotal <= 0 && item.Quantity > 0 && item.UnitPrice > 0 {
			lineTotal = roundInvoiceMoney(item.Quantity * item.UnitPrice)
		}
		unitPrice := roundInvoiceMoney(item.UnitPrice)
		if unitPrice <= 0 && item.Quantity > 0 && lineTotal > 0 {
			unitPrice = roundInvoiceMoney(lineTotal / item.Quantity)
		}
		if lineTotal <= 0 {
			continue
		}
		supplierCost := roundInvoiceMoney(item.TotalCost)
		totalSupplierCost += supplierCost
		subtotal += lineTotal
		invoiceItems = append(invoiceItems, DBInvoiceItem{
			Base:                Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			InvoiceID:           invoice.ID,
			LineNumber:          index + 1,
			Description:         item.Description,
			Quantity:            item.Quantity,
			Rate:                unitPrice,
			TotalBHD:            lineTotal,
			ProductID:           item.ProductID,
			ProductCode:         item.ProductCode,
			Equipment:           item.Equipment,
			Model:               item.Model,
			Specification:       item.Specification,
			DetailedDescription: item.DetailedDescription,
			Currency:            item.Currency,
			FOB:                 item.FOB,
			Freight:             item.Freight,
			TotalCost:           item.TotalCost,
			MarginPercent:       item.MarginPercent,
			TotalPrice:          lineTotal,
		})
	}

	if len(invoiceItems) == 0 {
		return false, invoice.InvoiceNumber, nil
	}

	currentSubtotal := roundInvoiceMoney(invoice.SubtotalBHD)
	if currentSubtotal <= 0 {
		currentSubtotal = roundInvoiceMoney(subtotal)
	}

	updates := map[string]any{
		"rfq_id":                  order.RFQID,
		"offer_id":                order.OfferID,
		"offer_number":            order.OfferNumber,
		"customer_reference":      order.CustomerReference,
		"attention_person":        order.AttentionPerson,
		"attention_company":       order.AttentionCompany,
		"attention_phone":         order.AttentionPhone,
		"attention_address":       order.AttentionAddress,
		"delivery_weeks":          order.DeliveryWeeks,
		"country_of_origin":       order.CountryOfOrigin,
		"issued_by":               order.IssuedBy,
		"contact_phone":           order.ContactPhone,
		"discount_percent":        order.DiscountPercent,
		"payment_terms":           order.PaymentTerms,
		"delivery_terms":          order.DeliveryTerms,
		"total_supplier_cost_bhd": roundInvoiceMoney(totalSupplierCost),
		"gross_margin_bhd":        roundInvoiceMoney(currentSubtotal - totalSupplierCost),
	}
	if invoice.SubtotalBHD <= 0 {
		updates["subtotal_bhd"] = currentSubtotal
	}
	if currentSubtotal > 0 {
		updates["gross_margin_percent"] = roundInvoiceMoney(((currentSubtotal - totalSupplierCost) / currentSubtotal) * 100)
	}
	if strings.TrimSpace(invoice.FieldVisibility) == "" {
		updates["field_visibility"] = defaultInvoiceFieldVisibilityJSON()
	}

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("invoice_id = ?", invoice.ID).Delete(&DBInvoiceItem{}).Error; err != nil {
			return fmt.Errorf("failed to clear stale invoice items for %s: %w", invoice.InvoiceNumber, err)
		}
		if err := tx.Create(&invoiceItems).Error; err != nil {
			return fmt.Errorf("failed to create invoice items for %s: %w", invoice.InvoiceNumber, err)
		}
		if err := tx.Model(&Invoice{}).Where("id = ?", invoice.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update invoice header for %s: %w", invoice.InvoiceNumber, err)
		}
		return nil
	}); err != nil {
		return false, invoice.InvoiceNumber, err
	}

	return true, invoice.InvoiceNumber, nil
}

// =============================================================================
// CUSTOMER INVOICE CRUD - Complete Invoice Lifecycle Management
// =============================================================================
//
// FEATURES:
//   - Invoice generation from customer orders
//   - Automatic invoice numbering (INV-YYYYMMDD-NNNN)
//   - Payment terms parsing and due date calculation
//   - Status workflow (Draft → Sent → Paid → Overdue)
//   - Order-to-Invoice traceability with quantity tracking
//   - Multi-item invoice support with margin tracking
//   - Outstanding balance management
//   - VAT calculation (10% Bahrain rate)
//   - BHD precision (3 decimal places)
//
// Built with MATHEMATICAL RIGOR × PRODUCTION ROBUSTNESS
// Day 196+ - Finance Hub Phase 2
// =============================================================================

// CreateInvoiceFromOrder creates a customer invoice from an existing order
// Maps all order fields, generates invoice number, calculates due date, creates line items
// Updates OrderItem.QuantityInvoiced for fulfillment tracking
func (a *App) CreateInvoiceFromOrder(orderID string) (Invoice, error) {
	log.Printf("📋 CreateInvoiceFromOrder: Starting for orderID=%s", orderID)

	if err := a.requirePermission("invoices:create"); err != nil {
		log.Printf("❌ CreateInvoiceFromOrder: Permission denied: %v", err)
		return Invoice{}, err
	}
	log.Printf("✅ CreateInvoiceFromOrder: Permission granted, calling CreateInvoiceWithOptions")

	invoice, err := a.CreateInvoiceWithOptions(orderID, "", "")
	if err != nil {
		log.Printf("❌ CreateInvoiceFromOrder: CreateInvoiceWithOptions failed: %v", err)
		return Invoice{}, err
	}
	log.Printf("✅ CreateInvoiceFromOrder: Invoice created successfully: %s", invoice.InvoiceNumber)
	return invoice, nil
}

// CreateProformaInvoice creates a proforma invoice from an order
// Proforma invoices are not tax documents - they serve as reference/quotation documents
// Status is set to "Proforma" and the invoice is NOT counted toward order fulfillment
func (a *App) CreateProformaInvoice(orderID string) (Invoice, error) {
	log.Printf("📋 CreateProformaInvoice: Starting for orderID=%s", orderID)

	if err := a.requirePermission("invoices:create"); err != nil {
		return Invoice{}, err
	}
	if a.db == nil {
		return Invoice{}, fmt.Errorf("database not initialized")
	}

	// Load order with items
	var order Order
	if err := a.db.Preload("Items").Where("id = ?", orderID).First(&order).Error; err != nil {
		return Invoice{}, fmt.Errorf("order not found: %w", err)
	}

	// C4: dedicated PF- sequence (own counter, own prefix) so a proforma NEVER
	// consumes an INV- fiscal number. See proformaNumberSpec below.
	invoiceNumber, err := numbering.New(a.db).Next(proformaNumberSpec(), time.Now())
	if err != nil {
		return Invoice{}, fmt.Errorf("failed to generate proforma number: %w", err)
	}

	// Build proforma invoice directly (skip fulfillment tracking)
	invoice := Invoice{
		InvoiceNumber:  invoiceNumber,
		OrderID:        order.ID,
		CustomerID:     order.CustomerID,
		CustomerName:   order.CustomerName,
		InvoiceDate:    time.Now(),
		DueDate:        time.Now().AddDate(0, 0, 30),
		Status:         "Proforma",
		SubtotalBHD:    order.TotalValueBHD,
		VATPercent:     10.0,
		VATBHD:         order.TotalValueBHD * 0.10,
		GrandTotalBHD:  order.TotalValueBHD * 1.10,
		OutstandingBHD: 0.0, // Proforma has no outstanding balance
		PaymentTerms:   order.PaymentTerms,
		Division:       normalizeDivisionName(order.Division),
	}

	// Copy line items from order, including the same costing/specification fields used by tax invoices.
	for index, orderItem := range order.Items {
		lineNumber := orderItem.LineNumber
		if lineNumber == 0 {
			lineNumber = index + 1
		}
		invoice.Items = append(invoice.Items, DBInvoiceItem{
			LineNumber:          lineNumber,
			Description:         orderItem.Description,
			Quantity:            orderItem.Quantity,
			Rate:                orderItem.UnitPrice,
			TotalBHD:            orderItem.TotalPrice,
			ProductID:           orderItem.ProductID,
			ProductCode:         orderItem.ProductCode,
			Equipment:           orderItem.Equipment,
			Model:               orderItem.Model,
			Specification:       orderItem.Specification,
			DetailedDescription: orderItem.DetailedDescription,
			Currency:            orderItem.Currency,
			FOB:                 orderItem.FOB,
			Freight:             orderItem.Freight,
			TotalCost:           orderItem.TotalCost,
			MarginPercent:       orderItem.MarginPercent,
			TotalPrice:          orderItem.TotalPrice,
		})
	}

	if err := a.db.Create(&invoice).Error; err != nil {
		return Invoice{}, fmt.Errorf("failed to create proforma invoice: %w", err)
	}

	// NOTE: No QuantityInvoiced update, no order status progression
	// Proforma is a reference document only

	log.Printf("✅ Created Proforma Invoice: %s (Customer: %s, Order: %s, Total: %.3f BHD)",
		invoice.InvoiceNumber, invoice.CustomerName, orderID, invoice.GrandTotalBHD)

	return invoice, nil
}

// proformaNumberSpec mints PF-{date}-{seq} numbers from their own counter
// (prefix "PF", independent of "INV" — see pkg/documents/numbering), so
// proforma creation never touches the INV- fiscal sequence. Mirrors
// GenerateCreditNoteNumber's CN- spec (credit_note_service.go); no Seed
// callback needed since PF- is a brand-new scheme with nothing to migrate.
func proformaNumberSpec() numbering.Spec {
	return numbering.Spec{
		Prefix:   "PF",
		Template: "PF-{date}-{seq}",
		Pad:      4,
	}
}

// ProformaInvoiceItemInput is the frontend-facing input for orderless
// proforma line items — mirrors CreditNoteItemInput's minimal shape
// (description/quantity/rate) since no order backs the line.
type ProformaInvoiceItemInput struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Rate        float64 `json:"rate"`
}

// CreateProformaInvoiceManual creates a proforma invoice directly against a
// customer, with no backing order (C4: orderless proforma). Like
// CreateProformaInvoice, it posts nothing — Status is "Proforma" and
// OutstandingBHD is 0, so the invoice is excluded from AR aging/VAT/revenue
// everywhere via customerInvoiceClosedWorkflowStatuses
// (customer_invoice_payment_policy.go) until it is converted. VAT is
// computed with the same 10% Bahrain rate and BHD rounding
// (roundInvoiceMoney) the order-based invoice paths use.
func (a *App) CreateProformaInvoiceManual(customerID string, customerName string, lineItems []ProformaInvoiceItemInput, notes string) (Invoice, error) {
	if err := a.requirePermission("invoices:create"); err != nil {
		return Invoice{}, err
	}
	if a.db == nil {
		return Invoice{}, fmt.Errorf("database not initialized")
	}

	customerID = strings.TrimSpace(customerID)
	customerName = strings.TrimSpace(customerName)
	if customerID == "" && customerName == "" {
		return Invoice{}, fmt.Errorf("a customer is required")
	}
	if len(lineItems) == 0 {
		return Invoice{}, fmt.Errorf("at least one line item is required")
	}

	now := time.Now()
	invoiceItems := make([]DBInvoiceItem, 0, len(lineItems))
	var subtotal float64
	for i, li := range lineItems {
		desc := strings.TrimSpace(li.Description)
		if desc == "" {
			return Invoice{}, fmt.Errorf("item %d: description is required", i+1)
		}
		if li.Quantity <= 0 || li.Rate <= 0 {
			return Invoice{}, fmt.Errorf("item %d: quantity and rate must be positive", i+1)
		}
		lineTotal := roundInvoiceMoney(li.Quantity * li.Rate)
		subtotal += lineTotal
		invoiceItems = append(invoiceItems, DBInvoiceItem{
			Base:        Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
			LineNumber:  i + 1,
			Description: desc,
			Quantity:    li.Quantity,
			Rate:        li.Rate,
			TotalBHD:    lineTotal,
			TotalPrice:  lineTotal,
		})
	}
	subtotal = roundInvoiceMoney(subtotal)
	vatPercent := 10.0 // Bahrain rate, same as the order-based invoice paths
	vatBHD := roundInvoiceMoney(subtotal * (vatPercent / 100.0))
	grandTotal := roundInvoiceMoney(subtotal + vatBHD)

	// Resolve a display name from the customer master if the caller only
	// passed an ID. Tries both the primary key and the business customer_id
	// column since callers of this bound method may hold either.
	if customerName == "" && customerID != "" {
		var customer CustomerMaster
		if err := a.db.Where("id = ? OR customer_id = ?", customerID, customerID).First(&customer).Error; err == nil {
			customerName = customer.BusinessName
		}
	}

	invoice := Invoice{
		Base:           Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		CustomerID:     customerID,
		CustomerName:   customerName,
		InvoiceDate:    now,
		DueDate:        now.AddDate(0, 0, 30),
		Status:         "Proforma",
		SubtotalBHD:    subtotal,
		VATPercent:     vatPercent,
		VATBHD:         vatBHD,
		GrandTotalBHD:  grandTotal,
		OutstandingBHD: 0.0, // Proforma has no outstanding balance — posts nothing
		Notes:          strings.TrimSpace(notes),
		Items:          invoiceItems,
	}

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		pfNumber, err := numbering.NextInTx(tx, proformaNumberSpec(), now)
		if err != nil {
			return fmt.Errorf("failed to generate proforma number: %w", err)
		}
		invoice.InvoiceNumber = pfNumber
		if err := tx.Create(&invoice).Error; err != nil {
			return fmt.Errorf("failed to create proforma invoice: %w", err)
		}
		return nil
	}); err != nil {
		return Invoice{}, err
	}

	log.Printf("✅ Created Proforma Invoice (manual): %s (Customer: %s, Total: %.3f BHD)",
		invoice.InvoiceNumber, invoice.CustomerName, invoice.GrandTotalBHD)

	return invoice, nil
}

// ConvertProformaToInvoice converts an existing Proforma into a real,
// fiscally-numbered invoice (C4 guarded conversion — mirrors MarkOfferWon's
// permission-gate + status-check + transaction pattern, app_sales_pipeline.go).
// It mints a fresh INV- number (the PF- number stays on the audit trail via
// logs; the row itself is updated in place rather than duplicated), sets
// OutstandingBHD to the full total, and moves Status to "Sent" — Draft would
// still be excluded from AR aging/VAT (customerInvoiceClosedWorkflowStatuses,
// customer_invoice_payment_policy.go), and a document the customer already
// holds as a proforma should not sit in an internal-only Draft state after
// conversion. customerPO is optional; when supplied it is stamped onto the
// invoice's PO fields.
func (a *App) ConvertProformaToInvoice(proformaID string, customerPO string) (Invoice, error) {
	if err := a.requirePermission("invoices:create"); err != nil {
		return Invoice{}, err
	}
	if a.db == nil {
		return Invoice{}, fmt.Errorf("database not initialized")
	}
	proformaID = strings.TrimSpace(proformaID)
	if proformaID == "" {
		return Invoice{}, fmt.Errorf("proforma invoice ID is required")
	}
	customerPO = strings.TrimSpace(customerPO)

	var invoice Invoice
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Items").First(&invoice, "id = ?", proformaID).Error; err != nil {
			return fmt.Errorf("proforma invoice not found: %w", err)
		}
		if invoice.Status != "Proforma" {
			return fmt.Errorf("cannot convert invoice %s: status is %s (only Proforma invoices can be converted)", invoice.InvoiceNumber, invoice.Status)
		}
		if len(invoice.Items) == 0 {
			return fmt.Errorf("cannot convert proforma %s: it has no line items", invoice.InvoiceNumber)
		}

		invoiceNumber, err := a.generateInvoiceNumberWithTx(tx)
		if err != nil {
			return fmt.Errorf("failed to generate invoice number: %w", err)
		}

		updates := map[string]any{
			"invoice_number":  invoiceNumber,
			"status":          "Sent",
			"outstanding_bhd": invoice.GrandTotalBHD,
			"updated_by":      a.getCurrentUserID(),
			"updated_at":      time.Now(),
		}
		if customerPO != "" {
			updates["customer_po_number"] = customerPO
			updates["buyers_order_number"] = customerPO
		}
		if err := tx.Model(&Invoice{}).Where("id = ?", proformaID).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to convert proforma invoice: %w", err)
		}

		invoice.InvoiceNumber = invoiceNumber
		invoice.Status = "Sent"
		invoice.OutstandingBHD = invoice.GrandTotalBHD
		if customerPO != "" {
			invoice.CustomerPONumber = customerPO
			invoice.BuyersOrderNumber = customerPO
		}

		invoice.InvoiceHash = computeDocumentHMAC(invoice.InvoiceNumber, invoice.InvoiceDate.Format("2006-01-02"), invoice.GrandTotalBHD, invoice.VATBHD)
		if err := tx.Model(&invoice).Update("invoice_hash", invoice.InvoiceHash).Error; err != nil {
			log.Printf("⚠️ Failed to save invoice hash after proforma conversion: %v", err)
		}

		return nil
	}); err != nil {
		return Invoice{}, err
	}

	log.Printf("✅ Converted Proforma to Invoice: %s (Customer: %s, Total: %.3f BHD)",
		invoice.InvoiceNumber, invoice.CustomerName, invoice.GrandTotalBHD)

	return invoice, nil
}

// CreateInvoiceFromOrderWithDN creates invoice with optional DN linkage
func (a *App) CreateInvoiceFromOrderWithDN(orderID string, deliveryNoteID string) (Invoice, error) {
	if err := a.requirePermission("invoices:create"); err != nil {
		return Invoice{}, err
	}
	return a.CreateInvoiceWithOptions(orderID, deliveryNoteID, "")
}

// CreateInvoiceFromDN creates an invoice directly from a delivery note
// Resolves the order from the DN and passes through to CreateInvoiceWithOptions
func (a *App) CreateInvoiceFromDN(deliveryNoteID string) (Invoice, error) {
	if err := a.requirePermission("invoices:create"); err != nil {
		return Invoice{}, err
	}
	if a.db == nil {
		return Invoice{}, fmt.Errorf("database not initialized")
	}

	// Load DN to get the order ID
	var dn DeliveryNote
	if err := a.db.Where("id = ?", deliveryNoteID).First(&dn).Error; err != nil {
		return Invoice{}, fmt.Errorf("delivery note not found: %w", err)
	}
	if dn.OrderID == "" {
		return Invoice{}, fmt.Errorf("delivery note %s is not linked to an order", dn.DNNumber)
	}

	log.Printf("📋 CreateInvoiceFromDN: DN=%s -> Order=%s", dn.DNNumber, dn.OrderID)
	return a.CreateInvoiceWithOptions(dn.OrderID, deliveryNoteID, "")
}

// CreateInvoiceWithOptions creates invoice with DN linkage and field visibility settings
// fieldVisibility is a JSON string like: {"show_fob":false,"show_freight":false,...}
func (a *App) CreateInvoiceWithOptions(orderID string, deliveryNoteID string, fieldVisibilityJSON string) (Invoice, error) {
	return a.createInvoiceWithOptionsEx(orderID, deliveryNoteID, fieldVisibilityJSON, "")
}

// CreateInvoiceFromOrderWithCreditOverride lets a management session create an
// invoice that exceeds the customer credit limit, recording an audited reason
// (PH SPOC #9). The override rides the kernel approval seam: the session is
// mapped to a kernel actor and the DecisionPending→DecisionApproved transition
// is validated by pkg/approvals, so the AI-authority boundary applies — agent
// actors can never satisfy it.
func (a *App) CreateInvoiceFromOrderWithCreditOverride(orderID string, reason string) (Invoice, error) {
	return a.CreateInvoiceWithCreditOverride(orderID, "", "", reason)
}

// CreateInvoiceWithCreditOverride is the override twin of
// CreateInvoiceWithOptions: same order/DN/field-visibility selection, plus the
// audited override reason. The frontend replays a credit-blocked create
// through this path.
func (a *App) CreateInvoiceWithCreditOverride(orderID string, deliveryNoteID string, fieldVisibilityJSON string, reason string) (Invoice, error) {
	if err := a.requirePermission("invoices:create"); err != nil {
		return Invoice{}, err
	}
	// Fast-fail before loading the order. The same actor gate is enforced again
	// at the chokepoint in createInvoiceWithOptionsEx, so every override path is
	// covered.
	if by, err := a.creditOverrideActor(); err != nil || !by.CanApprove() {
		return Invoice{}, newError("CREDIT_OVERRIDE_DENIED", "Only an Admin or Manager can override the credit limit", "")
	}
	if strings.TrimSpace(reason) == "" {
		return Invoice{}, newError("CREDIT_OVERRIDE_REASON_REQUIRED", "A reason is required to override the credit limit", "")
	}
	return a.createInvoiceWithOptionsEx(orderID, deliveryNoteID, fieldVisibilityJSON, reason)
}

// createInvoiceWithOptionsEx is the single invoice-creation sink.
// creditOverrideReason, when non-empty, allows the invoice to proceed even if
// it would exceed the customer's credit limit — gated on a kernel actor with
// approve authority (management roles) and recorded to the audit trail. Only
// CreateInvoiceFromOrderWithCreditOverride threads a non-empty reason here.
func (a *App) createInvoiceWithOptionsEx(orderID string, deliveryNoteID string, fieldVisibilityJSON string, creditOverrideReason string) (Invoice, error) {
	log.Printf("📋 CreateInvoiceWithOptions: orderID=%s, dnID=%s", orderID, deliveryNoteID)

	if err := a.requirePermission("invoices:create"); err != nil {
		log.Printf("❌ CreateInvoiceWithOptions: Permission denied: %v", err)
		return Invoice{}, err
	}

	// SPOC #9 chokepoint: a non-empty creditOverrideReason means the caller
	// intends to bypass the credit limit — re-verify the actor's authority here
	// so no other path can thread a reason through.
	overrideActor, actorErr := a.creditOverrideActor()
	if strings.TrimSpace(creditOverrideReason) != "" {
		if actorErr != nil || !overrideActor.CanApprove() {
			return Invoice{}, newError("CREDIT_OVERRIDE_DENIED", "Only an Admin or Manager can override the credit limit", "")
		}
	}
	if a.db == nil {
		log.Printf("❌ CreateInvoiceWithOptions: Database not initialized")
		return Invoice{}, fmt.Errorf("database not initialized")
	}

	// Load order with items preloaded
	log.Printf("📋 CreateInvoiceWithOptions: Loading order from database...")
	var order Order
	if err := a.db.Preload("Items").Where("id = ?", orderID).First(&order).Error; err != nil {
		log.Printf("❌ CreateInvoiceWithOptions: Order not found: %v", err)
		return Invoice{}, fmt.Errorf("order not found: %w", err)
	}
	log.Printf("✅ CreateInvoiceWithOptions: Loaded order %s with %d items, Total: %.3f BHD", order.OrderNumber, len(order.Items), order.GrandTotalBHD)

	order.Items = invoiceOrderItemsFromOrder(order)

	// FIXED: Handle orders without items (e.g., Tally imports) by creating a single line item
	// B1 FIX: Synthetic item is added to memory here but persisted inside the transaction below
	var syntheticItem *OrderItem
	if len(order.Items) == 0 {
		log.Printf("⚠️ Order %s has no items - creating single line item from order total", order.OrderNumber)
		// Create synthetic order item from order total (in-memory only; DB persist inside tx)
		item := OrderItem{
			Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			OrderID:     order.ID,
			LineNumber:  1,
			Description: fmt.Sprintf("Per Order %s", order.OrderNumber),
			Quantity:    1,
			UnitPrice:   order.GrandTotalBHD,
			TotalPrice:  order.GrandTotalBHD,
		}
		order.Items = []OrderItem{item}
		syntheticItem = &item
	}

	// Check if order has delivery notes (determines invoicing mode)
	var dnCount int64
	a.db.Model(&DeliveryNote{}).Where("order_id = ?", orderID).Count(&dnCount)
	hasDNs := dnCount > 0
	log.Printf("📋 CreateInvoiceWithOptions: Order has %d delivery notes (DN-based=%v)", dnCount, hasDNs)

	// Duplicate/invoiceable check depends on whether order uses delivery notes
	log.Printf("📋 CreateInvoiceWithOptions: Checking invoiceability...")
	var existingCount int64
	if err := a.db.Model(&Invoice{}).Where("order_id = ?", orderID).Count(&existingCount).Error; err != nil {
		log.Printf("❌ CreateInvoiceWithOptions: Failed to check existing invoices: %v", err)
		return Invoice{}, fmt.Errorf("failed to check existing invoices: %w", err)
	}

	if hasDNs {
		// DN-based flow: multiple invoices per order allowed (one per delivery)
		// Check that there are delivered-but-uninvoiced quantities remaining
		log.Printf("📦 DN-based invoicing: checking for uninvoiced delivered quantities")
		hasInvoiceable := false
		for _, item := range order.Items {
			uninvoiced := item.Quantity - item.QuantityInvoiced
			if uninvoiced > 0.001 {
				hasInvoiceable = true
				break
			}
		}
		if !hasInvoiceable {
			log.Printf("❌ CreateInvoiceWithOptions: All order items are fully invoiced")
			return Invoice{}, fmt.Errorf("all items in this order are already fully invoiced")
		}
		log.Printf("✅ DN-based invoicing: uninvoiced quantities found, proceeding")
	} else {
		// Legacy flow (no DNs): only one invoice per order allowed
		if existingCount > 0 {
			log.Printf("❌ CreateInvoiceWithOptions: Order already has %d invoice(s) (no DNs)", existingCount)
			return Invoice{}, fmt.Errorf("order already has %d invoice(s) - cannot create duplicate (use delivery notes for partial invoicing)", existingCount)
		}
		log.Printf("✅ CreateInvoiceWithOptions: No duplicate invoices found (legacy mode)")
	}

	// P0 FIX: Credit limit enforcement — check happens atomically inside the invoice creation
	// transaction below (Fix 1: TOCTOU gap closed). Preparation: just log intent here.
	log.Printf("📋 CreateInvoiceWithOptions: Credit check will run atomically with invoice creation for customer_id=%s", order.CustomerID)

	// Load delivery note if provided
	var deliveryNote DeliveryNote
	var dnItems []DeliveryNoteItem
	var dnNumber string
	var deliveryNoteDate *time.Time
	if deliveryNoteID != "" {
		if err := a.db.Preload("Items").Where("id = ?", deliveryNoteID).First(&deliveryNote).Error; err != nil {
			log.Printf("❌ Delivery note %s not found for invoice creation: %v", deliveryNoteID, err)
			return Invoice{}, fmt.Errorf("delivery note not found: %w", err)
		}
		if deliveryNote.OrderID != orderID {
			return Invoice{}, fmt.Errorf("delivery note %s does not belong to the selected order", deliveryNote.DNNumber)
		}
		var linkedInvoiceCount int64
		if err := a.db.Model(&Invoice{}).
			Where("delivery_note_id = ? AND status NOT IN ?", deliveryNoteID, []string{"Cancelled", "Void"}).
			Count(&linkedInvoiceCount).Error; err != nil {
			return Invoice{}, fmt.Errorf("failed to check delivery note invoice linkage: %w", err)
		}
		if linkedInvoiceCount > 0 {
			return Invoice{}, fmt.Errorf("delivery note %s has already been linked to an invoice", deliveryNote.DNNumber)
		}
		dnNumber = deliveryNote.DNNumber
		dnItems = deliveryNote.Items
		if !deliveryNote.DeliveryDate.IsZero() {
			dt := deliveryNote.DeliveryDate
			deliveryNoteDate = &dt
		}
		log.Printf("📦 Linking invoice to DN: %s (%d items)", dnNumber, len(dnItems))
	}

	// Invoice number is generated INSIDE the transaction below (PH parity) so a
	// rollback releases the reserved number instead of leaving a sequence gap.

	// Parse payment terms to calculate due date (e.g., "Net 30" → 30 days)
	dueDate := calculateDueDate(time.Now(), order.PaymentTerms)

	// Create invoice header - the costing sheet becomes the invoice
	now := time.Now()
	invoice := Invoice{
		Base: Base{
			ID:        uuid.New().String(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		// InvoiceNumber assigned inside the transaction (see generateInvoiceNumberWithTx)
		InvoiceDate:        now,
		CustomerID:         order.CustomerID,
		CustomerName:       order.CustomerName,
		OrderID:            order.ID,
		CustomerPONumber:   order.CustomerPONumber,
		GrandTotalBHD:      order.GrandTotalBHD,
		SubtotalBHD:        order.TotalValueBHD,
		Status:             "Draft",
		OutstandingBHD:     order.GrandTotalBHD, // Full amount outstanding initially
		DueDate:            dueDate,
		DeliveryNoteID:     deliveryNoteID,
		DeliveryNoteNumber: dnNumber,
		DeliveryNoteRef:    dnNumber,
		DeliveryNoteDate:   deliveryNoteDate,
		DespatchDocumentNo: dnNumber,
		ModeOfPayment:      firstNonEmptyString(order.PaymentTerms, "Direct Bank Transfer"),
		BuyersOrderNumber:  order.CustomerPONumber,
		TermsOfDelivery:    firstNonEmptyString(order.DeliveryTerms, "Direct Bank Transfer"),
		// Traceability links
		RfqID:       order.RFQID,
		OfferID:     order.OfferID,
		OfferNumber: order.OfferNumber,
		// Contact & RFQ details (copied from Order, originally from Offer/Costing Sheet)
		CustomerReference: order.CustomerReference,
		AttentionPerson:   order.AttentionPerson,
		AttentionCompany:  order.AttentionCompany,
		AttentionPhone:    order.AttentionPhone,
		AttentionAddress:  order.AttentionAddress,
		DeliveryWeeks:     order.DeliveryWeeks,
		CountryOfOrigin:   order.CountryOfOrigin,
		IssuedBy:          order.IssuedBy,
		ContactPhone:      order.ContactPhone,
		DiscountPercent:   order.DiscountPercent,
		PaymentTerms:      order.PaymentTerms,
		DeliveryTerms:     order.DeliveryTerms,
		Division:          normalizeDivisionName(order.Division),
		// Margin fields - will be calculated from items
		TotalSupplierCostBHD: 0.0,
		GrossMarginBHD:       0.0,
		GrossMarginPercent:   0.0,
		// Field visibility - use provided or default
		FieldVisibility: func() string {
			if fieldVisibilityJSON != "" {
				return fieldVisibilityJSON
			}
			return defaultInvoiceFieldVisibilityJSON()
		}(),
		Items: []DBInvoiceItem{},
	}

	// Create invoice items - logic depends on whether order has delivery notes
	// DN-based: use delivered quantities (from DN items or uninvoiced remainder)
	// Legacy: use full order quantities
	var totalSupplierCost float64
	var invoiceSubtotal float64
	lineNum := 0

	// Track actual invoiced quantity per order item ID for QuantityInvoiced update
	invoicedQtyByOrderItemID := make(map[string]float64)

	// Build a map of DN item quantities keyed by order_item_id for DN-based invoicing
	dnQtyByOrderItem := make(map[string]float64)
	if hasDNs && len(dnItems) > 0 {
		for _, di := range dnItems {
			dnQtyByOrderItem[di.OrderItemID] += di.QuantityDelivered
		}
	}

	for _, orderItem := range order.Items {
		// Determine the quantity to invoice for this item
		var invoiceQty float64
		if hasDNs {
			if len(dnItems) > 0 {
				// Specific DN provided: use that DN's delivered quantity for this item
				invoiceQty = dnQtyByOrderItem[orderItem.ID]
			} else {
				// No specific DN but order has DNs: invoice remaining uninvoiced quantity
				invoiceQty = orderItem.Quantity - orderItem.QuantityInvoiced
			}
			if invoiceQty <= 0.001 {
				continue // Skip fully invoiced items
			}
		} else {
			// Legacy (no DNs): invoice full order quantity
			invoiceQty = orderItem.Quantity
		}

		// Track invoiced quantity for this order item
		if strings.TrimSpace(orderItem.ID) != "" {
			invoicedQtyByOrderItemID[orderItem.ID] = invoiceQty
		}
		lineNum++

		// Use TotalCost from costing sheet if available, otherwise estimate
		var supplierCost float64
		if orderItem.TotalCost > 0 {
			supplierCost = orderItem.TotalCost * invoiceQty
		} else {
			// Fallback: try to get product cost
			var product ProductMaster
			if err := a.db.Where("id = ?", orderItem.ProductID).First(&product).Error; err == nil {
				supplierCost = product.StandardCostBHD * invoiceQty
			} else {
				// Estimate as 70% of selling price if no cost data
				supplierCost = orderItem.UnitPrice * invoiceQty * 0.70
			}
		}
		totalSupplierCost += supplierCost

		unitPrice := roundInvoiceMoney(orderItem.UnitPrice)
		fullLineTotal := roundInvoiceMoney(orderItem.TotalPrice)
		if unitPrice <= 0 && orderItem.Quantity > 0 && fullLineTotal > 0 {
			unitPrice = roundInvoiceMoney(fullLineTotal / orderItem.Quantity)
		}
		lineTotal := roundInvoiceMoney(invoiceQty * unitPrice)
		if lineTotal <= 0 && !hasDNs {
			lineTotal = fullLineTotal
		}
		if lineTotal <= 0 && invoiceQty > 0 && fullLineTotal > 0 && orderItem.Quantity <= 0 {
			lineTotal = fullLineTotal
		}
		if unitPrice <= 0 && invoiceQty > 0 && lineTotal > 0 {
			unitPrice = roundInvoiceMoney(lineTotal / invoiceQty)
		}
		invoiceSubtotal += lineTotal

		// Invoice item contains FULL costing sheet data (the costing sheet becomes the invoice)
		invoiceItem := DBInvoiceItem{
			Base: Base{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			InvoiceID:   invoice.ID,
			LineNumber:  lineNum,
			Description: orderItem.Description,
			Quantity:    invoiceQty,
			Rate:        unitPrice,
			TotalBHD:    lineTotal,
			// Full costing data from OrderItem (originally from OfferItem/CostingSheet)
			ProductID:           orderItem.ProductID,
			ProductCode:         orderItem.ProductCode,
			Equipment:           orderItem.Equipment,
			Model:               orderItem.Model,
			Specification:       orderItem.Specification,
			DetailedDescription: orderItem.DetailedDescription,
			Currency:            orderItem.Currency,
			FOB:                 orderItem.FOB,
			Freight:             orderItem.Freight,
			TotalCost:           orderItem.TotalCost,
			MarginPercent:       orderItem.MarginPercent,
			TotalPrice:          lineTotal,
		}
		invoice.Items = append(invoice.Items, invoiceItem)
	}

	if len(invoice.Items) == 0 {
		if deliveryNoteID != "" {
			return Invoice{}, fmt.Errorf("delivery note %s has no remaining invoiceable items for this order", dnNumber)
		}
		return Invoice{}, fmt.Errorf("order %s has no invoiceable items", order.OrderNumber)
	}

	// For partial invoices (DN-based), recalculate totals from actual invoice items
	if hasDNs {
		invoice.SubtotalBHD = roundInvoiceMoney(invoiceSubtotal)
	}

	// Calculate margin metrics (MON-005: round every BHD money write to 3dp so
	// the subtotal×0.10 VAT and subtotal+VAT additions below can't drift)
	invoice.TotalSupplierCostBHD = roundInvoiceMoney(totalSupplierCost)
	invoice.GrossMarginBHD = roundInvoiceMoney(invoice.SubtotalBHD - totalSupplierCost)
	if invoice.SubtotalBHD > 0 {
		invoice.GrossMarginPercent = (invoice.GrossMarginBHD / invoice.SubtotalBHD) * 100.0
	}

	// P0-4 FIX: Alert on negative margin (don't block, but warn management)
	if invoice.GrossMarginBHD < 0 {
		log.Printf("⚠️ ALERT: Negative margin detected on invoice %s: %.3f BHD (%.2f%%)",
			invoice.InvoiceNumber, invoice.GrossMarginBHD, invoice.GrossMarginPercent)
		// Don't block invoice creation, but log for management review
	}

	// Calculate VAT and grand total
	if hasDNs {
		// Partial invoices: calculate VAT from invoice subtotal (not order total)
		invoice.VATPercent = 10.0 // Bahrain rate
		invoice.VATBHD = roundInvoiceMoney(invoice.SubtotalBHD * 0.10)
		invoice.GrandTotalBHD = roundInvoiceMoney(invoice.SubtotalBHD + invoice.VATBHD)
		invoice.OutstandingBHD = invoice.GrandTotalBHD
	} else if order.GrandTotalBHD > order.TotalValueBHD {
		// Legacy: order already has VAT baked in, extract the VAT amount
		invoice.VATBHD = roundInvoiceMoney(order.GrandTotalBHD - order.TotalValueBHD)
		if order.TotalValueBHD > 0 {
			invoice.VATPercent = (invoice.VATBHD / order.TotalValueBHD) * 100.0
		} else {
			invoice.VATPercent = 10.0 // Default Bahrain rate
		}
		// Use order's grand total (already includes VAT)
		invoice.GrandTotalBHD = roundInvoiceMoney(order.GrandTotalBHD)
		invoice.OutstandingBHD = invoice.GrandTotalBHD
	} else {
		// Legacy: order doesn't include VAT, calculate it (10% Bahrain rate)
		invoice.VATPercent = 10.0
		invoice.VATBHD = roundInvoiceMoney(invoice.SubtotalBHD * 0.10)
		invoice.GrandTotalBHD = roundInvoiceMoney(invoice.SubtotalBHD + invoice.VATBHD)
		invoice.OutstandingBHD = invoice.GrandTotalBHD
	}

	// P0 FIX (TOCTOU closed): Credit check + invoice creation + QuantityInvoiced update
	// all happen inside a SINGLE transaction. No gap between credit check and DB write.
	log.Printf("📋 CreateInvoiceWithOptions: Beginning atomic transaction (credit check + create + qty update)...")
	var overrideAuditDescription string
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		// Step 0: Duplicate check inside transaction to prevent race conditions (Fix 6)
		if !hasDNs {
			var existingCount int64
			if err := tx.Model(&Invoice{}).Where("order_id = ?", orderID).Count(&existingCount).Error; err != nil {
				return fmt.Errorf("failed to check existing invoices: %w", err)
			}
			if existingCount > 0 {
				return fmt.Errorf("order already has %d invoice(s) — cannot create duplicate", existingCount)
			}
		}

		// Step 1: Credit limit check with row lock (SELECT FOR UPDATE)
		var customer CustomerMaster
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("customer_id = ?", order.CustomerID).First(&customer).Error; err == nil {
			// Customer found and locked — check credit limit atomically
			var totalOutstanding float64
			if err := tx.Model(&Invoice{}).
				Where("customer_id = ? AND status NOT IN ?", order.CustomerID, []string{"Paid", "Cancelled", "Void", "Proforma"}).
				Select("COALESCE(SUM(outstanding_bhd), 0)").
				Scan(&totalOutstanding).Error; err != nil {
				log.Printf("⚠️ Failed to calculate customer outstanding AR: %v", err)
				// Don't block invoice — log warning and continue
			} else {
				newTotalOutstanding := totalOutstanding + invoice.GrandTotalBHD
				creditLimit := customer.CreditLimitBHD
				if creditLimit == 0 {
					creditLimit = 50000.0
				}
				if newTotalOutstanding > creditLimit*0.80 && newTotalOutstanding <= creditLimit {
					log.Printf("⚠️ WARNING: Customer %s approaching credit limit (%.1f%% utilized: %.3f / %.3f BHD)",
						customer.BusinessName,
						(newTotalOutstanding/creditLimit)*100,
						newTotalOutstanding,
						creditLimit)
				}

				if customer.IsCreditBlocked {
					log.Printf("BLOCKED: Customer %s is credit blocked - no new invoices allowed", customer.BusinessName)
					return fmt.Errorf("customer %s is credit blocked - no new invoices allowed", customer.BusinessName)
				}
				if newTotalOutstanding > creditLimit {
					if strings.TrimSpace(creditOverrideReason) != "" {
						// SPOC #9: management credit-limit override with an audited reason.
						// Route the decision through the kernel approval seam: an assessment
						// finding demands approval, and the Pending→Approved transition is
						// only legal for an actor with approve authority (never an agent).
						assessment := approvals.NewAssessment(orderID, "credit_override")
						assessment.Add(approvals.Finding{
							Code:             "credit_limit_exceeded",
							Message:          fmt.Sprintf("new outstanding %.3f exceeds limit %.3f for %s", newTotalOutstanding, creditLimit, customer.BusinessName),
							RequiresApproval: true,
						})
						record, trErr := approvals.Transition(orderID, "credit_override",
							assessment.Decision(), approval.DecisionApproved,
							overrideActor, creditOverrideReason, time.Now())
						if trErr != nil {
							return fmt.Errorf("credit limit override refused: %w", trErr)
						}
						log.Printf("⚠️ CREDIT LIMIT OVERRIDE by %s: customer %s new outstanding %.3f > limit %.3f — reason: %s",
							a.GetCurrentUserRole(), customer.BusinessName, newTotalOutstanding, creditLimit, creditOverrideReason)
						// Audit row is written after the tx commits (a.db is a separate
						// connection; writing inside the tx deadlocks SQLite) — and only a
						// created invoice should carry an override audit row anyway.
						overrideAuditDescription = fmt.Sprintf("Credit limit override by %s for %s: new outstanding %.3f > limit %.3f — reason: %s (approval %s)",
							a.GetCurrentUserRole(), customer.BusinessName, newTotalOutstanding, creditLimit, creditOverrideReason, record.CorrelationID)
					} else {
						log.Printf("BLOCKED: Customer %s exceeds credit limit (%.3f BHD > %.3f BHD)",
							customer.BusinessName, newTotalOutstanding, creditLimit)
						return fmt.Errorf(
							"credit limit exceeded for %s: new outstanding %.3f exceeds limit %.3f",
							customer.BusinessName, newTotalOutstanding, creditLimit)
					}
				}
			}
		}
		// Customer not found — skip credit check (not an error)

		// B1 FIX: Persist synthetic order item inside transaction (was previously outside)
		if syntheticItem != nil {
			if err := tx.Create(syntheticItem).Error; err != nil {
				log.Printf("⚠️ Could not persist synthetic order item: %v", err)
			} else {
				log.Printf("✅ Created synthetic order item for order %s (inside tx)", order.OrderNumber)
			}
		}

		// Step 1.5: Reserve the invoice number on THIS transaction so it rolls back
		// with the invoice on any downstream failure (no sequence gaps). Assigned
		// before Create and before the HMAC below, which hashes InvoiceNumber.
		invoiceNumber, numErr := a.generateInvoiceNumberWithTx(tx)
		if numErr != nil {
			return fmt.Errorf("failed to generate invoice number: %w", numErr)
		}
		invoice.InvoiceNumber = invoiceNumber

		// Step 2: Create invoice in database (GORM will cascade create items)
		if err := tx.Create(&invoice).Error; err != nil {
			log.Printf("❌ CreateInvoiceWithOptions: Failed to create invoice in DB: %v", err)
			return fmt.Errorf("failed to create invoice: %w", err)
		}
		log.Printf("✅ CreateInvoiceWithOptions: Invoice saved to database: ID=%s", invoice.ID)

		// Step 3: Update OrderItem.QuantityInvoiced using actual invoiced quantities
		for i, orderItem := range order.Items {
			invoicedQty, found := invoicedQtyByOrderItemID[orderItem.ID]
			if !found || invoicedQty <= 0.001 {
				continue // This order item was not invoiced (skipped in DN-based flow)
			}
			// Use atomic SQL increment to prevent race conditions
			if err := tx.Model(&OrderItem{}).
				Where("id = ?", orderItem.ID).
				Update("quantity_invoiced", gorm.Expr("quantity_invoiced + ?", invoicedQty)).Error; err != nil {
				log.Printf("⚠️ Failed to update quantity_invoiced for order item %s: %v", orderItem.ID, err)
				// Continue processing - this is not critical
			} else {
				log.Printf("✅ Updated OrderItem[%d]: quantity_invoiced += %.3f", i, invoicedQty)
			}
		}

		// Step 4: Compute and save invoice integrity hash (HMAC-SHA256 with server secret — P1-4)
		invoice.InvoiceHash = computeDocumentHMAC(invoice.InvoiceNumber, invoice.InvoiceDate.Format("2006-01-02"), invoice.GrandTotalBHD, invoice.VATBHD)
		if err := tx.Model(&invoice).Update("invoice_hash", invoice.InvoiceHash).Error; err != nil {
			log.Printf("⚠️ Failed to save invoice hash: %v", err)
		} else {
			log.Printf("✅ Invoice hash computed: %s...%s", invoice.InvoiceHash[:8], invoice.InvoiceHash[56:])
		}

		return nil
	}); err != nil {
		return Invoice{}, err
	}

	if overrideAuditDescription != "" {
		var actorID *string
		if uid := strings.TrimSpace(a.getCurrentUserID()); uid != "" {
			actorID = &uid
		}
		// Synchronous on purpose (unlike logAudit's RecordAsync): this row is
		// the audit evidence for a credit-authority override — it must be
		// durable before the call returns, not racing a shutdown.
		if rec := a.auditRecorder(); rec != nil {
			var uid string
			if actorID != nil {
				uid = *actorID
			}
			if err := rec.Record(audit.Entry{
				UserID:      uid,
				Action:      "CREDIT_LIMIT_OVERRIDE",
				Resource:    "invoices",
				ResourceID:  orderID,
				Description: overrideAuditDescription,
			}); err != nil {
				log.Printf("⚠️ audit write failed (CREDIT_LIMIT_OVERRIDE invoices): %v", err)
			}
		}
	}

	log.Printf("✅ Created Invoice: %s (Customer: %s, Order: %s, Total: %.3f BHD, Margin: %.3f BHD [%.1f%%])",
		invoice.InvoiceNumber, invoice.CustomerName, order.OrderNumber,
		invoice.GrandTotalBHD, invoice.GrossMarginBHD, invoice.GrossMarginPercent)

	// Emit event to notify frontend to refresh dashboard
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "data:refresh", map[string]any{
			"source":         "invoice-created",
			"invoice_id":     invoice.ID,
			"invoice_number": invoice.InvoiceNumber,
			"order_id":       order.ID,
		})
	}

	// Phase 23: Link serial numbers from DN to this invoice
	if deliveryNoteID != "" {
		if err := a.linkSerialsToInvoice(invoice.ID, invoice.InvoiceNumber, deliveryNoteID); err != nil {
			log.Printf("⚠️ Failed to link serials to invoice: %v (non-blocking)", err)
		}
	}

	// Auto-progress order status to "Invoiced"
	if err := a.ProgressOrderOnInvoice(order.ID); err != nil {
		log.Printf("⚠️ Failed to progress order %s to Invoiced: %v", order.OrderNumber, err)
		// Don't fail the invoice creation, just log the warning
	}

	return invoice, nil
}

// ListCustomerInvoices retrieves all customer invoices with pagination
// Preloads items for complete data
func (a *App) ListCustomerInvoices(limit, offset int) ([]Invoice, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// B8 FIX: Enforce pagination bounds to prevent memory exhaustion
	if limit <= 0 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	var invoices []Invoice
	query := a.db.Preload("Items").Order("invoice_date DESC").Limit(limit).Offset(offset)

	if err := query.Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve invoices: %w", err)
	}

	// Mission G (Wave 4): recompute settlement status/outstanding on read so a
	// past-due invoice surfaces as "Overdue" without waiting for a mutator,
	// matching PH's read-path hydration. Display-only; the store is unchanged.
	hydrateCustomerInvoicesPaymentState(invoices)

	// Debug: Log items loaded per invoice
	for i, inv := range invoices {
		log.Printf("🔍 Invoice[%d] %s: %d items loaded", i, inv.InvoiceNumber, len(inv.Items))
	}

	log.Printf("📊 Retrieved %d customer invoices (limit: %d, offset: %d)", len(invoices), limit, offset)
	return invoices, nil
}

// GetCustomerInvoiceByID retrieves a single customer invoice by ID
// Preloads items for complete invoice data
func (a *App) GetCustomerInvoiceByID(id string) (Invoice, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return Invoice{}, err
	}
	if a.db == nil {
		return Invoice{}, fmt.Errorf("database not initialized")
	}

	var invoice Invoice
	if err := a.db.Preload("Items").Where("id = ?", id).First(&invoice).Error; err != nil {
		return Invoice{}, fmt.Errorf("invoice not found: %w", err)
	}

	// Mission G (Wave 4): recompute settlement status/outstanding on read (parity
	// with PH). Display-only; the store is unchanged.
	hydrateCustomerInvoicePaymentState(&invoice)

	return invoice, nil
}

// UpdateCustomerInvoice updates an existing customer invoice
// Recalculates margin if costs changed, updates timestamp
// FIX: Uses Save() instead of Updates() to preserve ALL fields including zero-value strings
// SECURITY: Paid invoices are immutable to prevent audit trail tampering
func (a *App) UpdateCustomerInvoice(inv Invoice) (Invoice, error) {
	if err := a.requirePermission("invoices:update"); err != nil {
		return Invoice{}, err
	}
	if a.db == nil {
		return Invoice{}, fmt.Errorf("database not initialized")
	}

	// Verify invoice exists and load ALL fields
	var existing Invoice
	if err := a.db.Where("id = ?", inv.ID).First(&existing).Error; err != nil {
		return Invoice{}, fmt.Errorf("invoice not found: %w", err)
	}
	hydrateCustomerInvoicePaymentState(&existing)

	// SECURITY: Terminal states (Paid, Cancelled, Void) are immutable for audit compliance
	terminalStatuses := map[string]bool{"Paid": true, "Cancelled": true, "Void": true}
	if terminalStatuses[existing.Status] {
		return Invoice{}, fmt.Errorf("cannot modify invoice with status '%s' (terminal state — immutable for audit compliance)", existing.Status)
	}

	// CRITICAL FIX: Preserve ALL fields that weren't sent from frontend
	// The edit modal only sends 5 fields, so we must preserve the other 40+ fields

	// Fix 3: InvoiceNumber locked once past Draft — part of the audit trail + HMAC hash
	if inv.InvoiceNumber != "" && inv.InvoiceNumber != existing.InvoiceNumber {
		if existing.Status != "Draft" {
			return Invoice{}, fmt.Errorf("cannot change invoice number on %s invoice (locked for audit compliance)", existing.Status)
		}
		// Uniqueness check: verify no other invoice has the same number
		var count int64
		a.db.Model(&Invoice{}).Where("invoice_number = ? AND id != ?", inv.InvoiceNumber, inv.ID).Count(&count)
		if count > 0 {
			return Invoice{}, fmt.Errorf("invoice number %s already exists", inv.InvoiceNumber)
		}
		existing.InvoiceNumber = inv.InvoiceNumber
	}

	// Wave 9.6 AR1: capture the original status before any reassignment below,
	// so the item-replacement guard further down can check what the invoice
	// ACTUALLY was on load, not a status the request may have overwritten it to.
	originalStatus := existing.Status

	// Fix 1: Validate status allowlist before accepting (rejects empty string + adds Void/Proforma)
	validStatuses := map[string]bool{"Draft": true, "Sent": true, "Overdue": true, "PartiallyPaid": true, "Paid": true, "Cancelled": true, "Void": true, "Proforma": true}
	if !validStatuses[inv.Status] {
		return Invoice{}, fmt.Errorf("invalid invoice status: '%s'", inv.Status)
	}

	// Mission I (I-08): Outstanding is payment-driven — this guard runs BEFORE
	// any recompute so a client payload can never overwrite the balance. Closed
	// workflow statuses (Draft/Proforma) are exempt: they carry no payments and
	// their outstanding is server-derived from line items on save regardless of
	// what the client sends.
	if !customerInvoiceClosedWorkflowStatuses[existing.Status] &&
		math.Abs(inv.OutstandingBHD-existing.OutstandingBHD) > FloatingPointTolerance {
		return Invoice{}, fmt.Errorf("outstanding balance is payment-driven and cannot be edited directly")
	}

	// Mission I (I-08): settlement statuses (Paid/PartiallyPaid/Overdue) are
	// derived from payments + due date, never set by hand (PH parity; subsumes
	// the old Paid-with-outstanding check).
	if inv.Status != existing.Status && (isCustomerInvoiceSettlementStatus(inv.Status) || isCustomerInvoiceSettlementStatus(existing.Status)) {
		return Invoice{}, fmt.Errorf("invoice settlement status is payment-driven; record a payment or let the due-date workflow derive overdue state")
	}

	// Wave 9.6 AR1: a posted invoice (Sent or beyond) may not be reverted to a
	// pre-posting EDITABLE state (Draft/Proforma) — that would pull a numbered,
	// aged/VAT-scoped invoice out of the books and re-open its line items. Mirrors
	// the supplier-side lifecycle protection (Wave 9.2). Draft→Sent stays allowed
	// (existing is not yet posted); Sent→Cancelled/Void (terminal closures that do
	// NOT re-open items) are intentionally NOT blocked here.
	if inv.Status != existing.Status &&
		isCustomerInvoicePostedStatus(existing.Status) &&
		(inv.Status == "Draft" || inv.Status == "Proforma") {
		return Invoice{}, fmt.Errorf("a posted invoice (%s) cannot be reverted to %s", existing.Status, inv.Status)
	}

	existing.Status = inv.Status

	// Fix 2: GrandTotalBHD is a computed field (SubtotalBHD + VATBHD) — NOT directly editable
	// OutstandingBHD intentionally NOT editable — must change via payments/credit notes only
	existing.CustomerPONumber = inv.CustomerPONumber

	// Update timestamp
	existing.UpdatedAt = time.Now()

	// Recalculate margin metrics if values changed
	if existing.SubtotalBHD > 0 && existing.TotalSupplierCostBHD > 0 {
		existing.GrossMarginBHD = existing.SubtotalBHD - existing.TotalSupplierCostBHD
		existing.GrossMarginPercent = (existing.GrossMarginBHD / existing.SubtotalBHD) * 100.0
	}

	// Use Save() to update ALL fields (preserves empty strings, zero values)
	if err := a.db.Save(&existing).Error; err != nil {
		return Invoice{}, fmt.Errorf("failed to update invoice: %w", err)
	}

	// If items were updated, handle them separately
	if len(inv.Items) > 0 {
		// Fix 4: Block item replacement on non-Draft invoices — items are part of the financial record
		// Wave 9.6 AR1: check originalStatus (as loaded), not existing.Status (which may
		// have just been reassigned above) — defense in depth so a genuinely-posted
		// invoice's items can never be replaced even if a future path lets status through.
		if originalStatus != "Draft" {
			log.Printf("⚠️ Ignoring item replacement on %s invoice %s (only Draft invoices can have items modified)", existing.Status, existing.InvoiceNumber)
		} else {
			if err := a.db.Where("invoice_id = ?", inv.ID).Delete(&DBInvoiceItem{}).Error; err != nil {
				log.Printf("⚠️ Failed to delete old invoice items: %v", err)
			}
			var newSubtotal float64
			for i := range inv.Items {
				item := &inv.Items[i]
				item.InvoiceID = inv.ID
				item.LineNumber = i + 1
				item.Quantity = roundInvoiceMoney(item.Quantity)
				item.Rate = roundInvoiceMoney(item.Rate)
				lineTotal := roundInvoiceMoney(item.Quantity * item.Rate)
				if lineTotal <= 0 && item.TotalBHD > 0 {
					lineTotal = roundInvoiceMoney(item.TotalBHD) // tolerate client-sent total
				}
				item.TotalBHD = lineTotal
				item.TotalPrice = lineTotal
				newSubtotal += lineTotal
				item.UpdatedAt = time.Now()
				if item.ID == "" {
					item.ID = uuid.New().String()
					item.CreatedAt = time.Now()
				}
				if err := a.db.Create(item).Error; err != nil {
					log.Printf("⚠️ Failed to create invoice item: %v", err)
				}
			}
			// Recompute invoice money from the new lines (Draft only; Drafts are unpaid).
			// Recover the EFFECTIVE VAT rate before defaulting so a genuine 0%
			// (zero-rated/export) invoice is not silently forced to 10% on edit — a
			// won-import Draft can carry VATPercent=0 when OCR found no rate.
			vatPercent := existing.VATPercent
			if vatPercent <= 0 {
				if existing.SubtotalBHD > 0 && existing.VATBHD > 0 {
					vatPercent = existing.VATBHD / existing.SubtotalBHD * 100.0
				} else if existing.VATBHD == 0 && existing.GrandTotalBHD > 0 &&
					math.Abs(existing.GrandTotalBHD-existing.SubtotalBHD) <= FloatingPointTolerance {
					vatPercent = 0 // genuinely zero-rated
				} else {
					vatPercent = 10.0 // Bahrain default
				}
			}
			subtotal := roundInvoiceMoney(newSubtotal)
			vat := roundInvoiceMoney(subtotal * vatPercent / 100.0)
			grand := roundInvoiceMoney(subtotal + vat)
			existing.SubtotalBHD = subtotal
			existing.VATPercent = vatPercent
			existing.VATBHD = vat
			existing.GrandTotalBHD = grand
			existing.OutstandingBHD = grand // Draft = fully outstanding/unpaid
			if existing.TotalSupplierCostBHD > 0 {
				existing.GrossMarginBHD = roundInvoiceMoney(subtotal - existing.TotalSupplierCostBHD)
				if subtotal > 0 {
					existing.GrossMarginPercent = (existing.GrossMarginBHD / subtotal) * 100.0
				}
			}
			if err := a.db.Save(&existing).Error; err != nil {
				return Invoice{}, fmt.Errorf("failed to persist recomputed invoice totals: %w", err)
			}
		}
	}

	log.Printf("✅ Updated Invoice: %s (preserved all 44+ fields)", existing.InvoiceNumber)

	// Reload with items
	return a.GetCustomerInvoiceByID(inv.ID)
}

// DeleteCustomerInvoice deletes a customer invoice
// Only allows deletion if not paid (to prevent accounting issues)
// Reverses QuantityInvoiced updates on order items
// SECURITY: Blocks deletion if any payment history exists (audit requirement)
func (a *App) DeleteCustomerInvoice(id string) error {
	if ok, err := a.guardDeleteOrRequest("invoices:delete", "customer_invoice", id, "Customer invoice"); !ok {
		return err
	}
	if err := a.requirePermission("invoices:delete"); err != nil {
		return err
	}
	return financeinvoice.Delete(a.db, id, a.reverseInvoicedQuantities)
}

// reverseInvoicedQuantities rolls back order-item quantity_invoiced
// bookkeeping when an invoice is deleted (fulfillment concern — stays
// with the host's CRM surface).
func (a *App) reverseInvoicedQuantities(invoice Invoice) {
	var order Order
	if err := a.db.Preload("Items").Where("id = ?", invoice.OrderID).First(&order).Error; err == nil {
		// Match invoice items to order items and reverse quantities
		for _, invItem := range invoice.Items {
			for _, orderItem := range order.Items {
				// Match by description (simplified - could be more robust)
				if orderItem.Description == invItem.Description {
					newQuantityInvoiced := orderItem.QuantityInvoiced - invItem.Quantity
					if newQuantityInvoiced < 0 {
						newQuantityInvoiced = 0
					}
					if err := a.db.Model(&OrderItem{}).
						Where("id = ?", orderItem.ID).
						Update("quantity_invoiced", newQuantityInvoiced).Error; err != nil {
						log.Printf("⚠️ Failed to reverse quantity_invoiced for order item %s: %v", orderItem.ID, err)
					} else {
						log.Printf("✅ Reversed OrderItem quantity_invoiced: %.3f → %.3f",
							orderItem.QuantityInvoiced, newQuantityInvoiced)
					}
					break
				}
			}
		}
	}
}

// SendCustomerInvoice marks an invoice as sent to customer
// Updates status from Draft → Sent
func (a *App) SendCustomerInvoice(id string) error {
	if err := a.requirePermission("invoices:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Verify invoice exists
	var invoice Invoice
	if err := a.db.Where("id = ?", id).First(&invoice).Error; err != nil {
		return fmt.Errorf("invoice not found: %w", err)
	}

	// B9 FIX: Only Draft invoices can be sent — reject all other statuses
	if invoice.Status != "Draft" {
		return fmt.Errorf("cannot send invoice %s: status is %s (only Draft invoices can be sent)", invoice.InvoiceNumber, invoice.Status)
	}

	// MON-007: refuse to send a hollow invoice (no line items) — sending it would
	// render a blank line-item table against a non-zero total.
	var itemCount int64
	a.db.Table("invoice_items").Where("invoice_id = ?", id).Count(&itemCount)
	if itemCount == 0 {
		return fmt.Errorf("cannot send invoice %s: it has no line items", invoice.InvoiceNumber)
	}

	// Update status to Sent
	now := time.Now()
	if err := a.db.Model(&Invoice{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     "Sent",
			"updated_at": now,
		}).Error; err != nil {
		return fmt.Errorf("failed to send invoice: %w", err)
	}

	log.Printf("✅ Sent Invoice: %s to customer %s", invoice.InvoiceNumber, invoice.CustomerName)
	return nil
}

// GetInvoicesByCustomer retrieves all invoices for a specific customer
// Ordered by invoice date descending (most recent first)
func (a *App) GetInvoicesByCustomer(customerID string) ([]Invoice, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var invoices []Invoice
	if err := a.db.Preload("Items").
		Where("customer_id = ?", customerID).
		Order("invoice_date DESC").
		Limit(500).
		Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve customer invoices: %w", err)
	}

	hydrateCustomerInvoicesPaymentState(invoices)

	log.Printf("📊 Retrieved %d invoices for customer %s", len(invoices), customerID)
	return invoices, nil
}

// GenerateInvoiceNumber generates a sequential invoice number using a sequence table
// Format: INV-YYYYMMDD-NNNN (e.g., INV-20260123-0001)
// P0 FIX: Uses InvoiceSequence table with row-level locking to prevent race conditions
// The SELECT FOR UPDATE ensures atomic read-modify-write even under concurrent calls
func (a *App) GenerateInvoiceNumber() (string, error) {
	// Mission G (Wave 4): restore the invoices:create RBAC guard PH carries
	// (it was dropped when numbering moved to the engine). Reserving an invoice
	// sequence number is a create-side side-effect and must be gated.
	if err := a.requirePermission("invoices:create"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	// Delegates to the promoted pkg/documents/numbering engine (Wave 2
	// Mission A). Format and first-of-year seeding are byte-identical to the
	// old inline implementation.
	invoiceNumber, err := numbering.New(a.db).Next(invoiceNumberSpec(), time.Now())

	if err != nil {
		log.Printf("❌ Invoice number generation failed: %v", err)
		return "", fmt.Errorf("failed to generate invoice number: %w", err)
	}

	log.Printf("🔢 Generated invoice number: %s (sequence-locked)", invoiceNumber)
	return invoiceNumber, nil
}

// invoiceNumberSpec is the single source of truth for the INV-YYYYMMDD-NNNN
// format and first-of-year seeding, shared by the standalone GenerateInvoiceNumber
// and the transaction-scoped generateInvoiceNumberWithTx so they can never drift.
func invoiceNumberSpec() numbering.Spec {
	return numbering.Spec{
		Prefix:   "INV",
		Template: "INV-{date}-{seq}",
		Seed: func(tx *gorm.DB, year int) (int64, error) {
			// Initialize from existing invoices (migration safety).
			var maxExisting int64
			tx.Model(&Invoice{}).
				Where("invoice_number LIKE ? ESCAPE '\\'", fmt.Sprintf("INV-%d%%", year%100)).
				Count(&maxExisting)
			return maxExisting, nil
		},
	}
}

// generateInvoiceNumberWithTx reserves the next invoice number using the caller's
// transaction (PH parity), so a rollback releases the number instead of leaving a
// sequence gap. No RBAC gate here — callers are already inside a gated create flow;
// the public GenerateInvoiceNumber wraps the same spec with the invoices:create guard.
func (a *App) generateInvoiceNumberWithTx(tx *gorm.DB) (string, error) {
	invoiceNumber, err := numbering.NextInTx(tx, invoiceNumberSpec(), time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to generate invoice number: %w", err)
	}
	log.Printf("🔢 Generated invoice number: %s (sequence-locked, in-tx)", invoiceNumber)
	return invoiceNumber, nil
}

// =============================================================================
// PAYMENT TRACKING & STATUS MANAGEMENT
// =============================================================================

// MarkCustomerInvoicePaid marks an invoice as paid
// Updates outstanding balance to 0 and status to Paid
// P1 FIX: Race condition protection via database transaction
func (a *App) MarkCustomerInvoicePaid(id string, paymentDate time.Time, paymentRef string) error {
	if err := a.requirePermission("payments:create"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var invoice Invoice
	if err := a.db.Where("id = ?", id).First(&invoice).Error; err != nil {
		return fmt.Errorf("invoice not found: %w", err)
	}

	// Mission I (I-04/I-08): mark-paid is a payment event, not a status edit.
	// The old inline version zeroed the balance with NO payment record — money
	// disappeared from the audit trail. PH routes through RecordPartialPayment
	// for the full outstanding, which creates the Payment row and derives the
	// status via the settlement policy.
	state := customerInvoicePaymentStateFromInvoice(invoice, time.Now())
	if !state.IsOpen {
		return fmt.Errorf("cannot mark invoice as paid: current status is '%s'", state.Status)
	}

	if err := a.RecordPartialPayment(id, state.OutstandingBHD, paymentDate, paymentRef); err != nil {
		return fmt.Errorf("failed to mark invoice as paid: %w", err)
	}

	log.Printf("💰 Marked Invoice as Paid: %s (Customer: %s, Amount: %.3f BHD, Ref: %s)",
		invoice.InvoiceNumber, invoice.CustomerName, state.OutstandingBHD, paymentRef)

	return nil
}

// MarkCustomerInvoiceOverdue marks an invoice as overdue
// Called by background job checking due dates
func (a *App) MarkCustomerInvoiceOverdue(id string) error {
	if err := a.requirePermission("invoices:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Verify invoice exists and is not paid
	var invoice Invoice
	if err := a.db.Where("id = ?", id).First(&invoice).Error; err != nil {
		return fmt.Errorf("invoice not found: %w", err)
	}

	// Mission I (I-02/I-04): overdue is DERIVED from the settlement policy, never
	// asserted — the old inline version stamped "Overdue" even when the due date
	// had not passed. PH parity: refuse unless the policy itself says Overdue.
	state := customerInvoicePaymentStateFromInvoice(invoice, time.Now())
	if !state.IsCollectible {
		return fmt.Errorf("cannot mark %s invoice as overdue: %s", state.Status, invoice.InvoiceNumber)
	}
	if state.Status != "Overdue" {
		return fmt.Errorf("invoice settlement status remains %s: %s", state.Status, invoice.InvoiceNumber)
	}

	tx := a.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	invoice.OutstandingBHD = state.OutstandingBHD
	if _, err := a.applyCustomerInvoicePaymentState(tx, &invoice); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to mark invoice as overdue: %w", err)
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit overdue update: %w", err)
	}

	log.Printf("⚠️ Marked Invoice as Overdue: %s (Customer: %s, Due: %s, Outstanding: %.3f BHD)",
		invoice.InvoiceNumber, invoice.CustomerName,
		invoice.DueDate.Format("2006-01-02"), invoice.OutstandingBHD)

	return nil
}

// GetOverdueInvoices retrieves all overdue invoices
// Invoices where due_date < now AND status != 'Paid'
func (a *App) GetOverdueInvoices() ([]Invoice, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	now := time.Now()
	var invoices []Invoice
	if err := a.db.Preload("Items").
		Where("due_date < ? AND status != ?", now, "Paid").
		Order("due_date ASC").
		Limit(500).
		Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve overdue invoices: %w", err)
	}

	hydrateCustomerInvoicesPaymentState(invoices)

	log.Printf("⚠️ Retrieved %d overdue invoices", len(invoices))
	return invoices, nil
}

// GetUnpaidInvoices retrieves all unpaid invoices
// Invoices where status IN ('Draft', 'Sent', 'Overdue')
func (a *App) GetUnpaidInvoices() ([]Invoice, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var invoices []Invoice
	if err := a.db.Preload("Items").
		Where("status IN ?", []string{"Draft", "Sent", "Overdue"}).
		Order("invoice_date DESC").
		Limit(500).
		Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve unpaid invoices: %w", err)
	}

	hydrateCustomerInvoicesPaymentState(invoices)

	log.Printf("📊 Retrieved %d unpaid invoices", len(invoices))
	return invoices, nil
}

// GetInvoicesByStatus retrieves invoices filtered by status
func (a *App) GetInvoicesByStatus(status string) ([]Invoice, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var invoices []Invoice
	if err := a.db.Preload("Items").
		Where("status = ?", status).
		Order("invoice_date DESC").
		Limit(500).
		Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve invoices by status: %w", err)
	}

	hydrateCustomerInvoicesPaymentState(invoices)

	log.Printf("📊 Retrieved %d invoices with status: %s", len(invoices), status)
	return invoices, nil
}

// RecordPartialPayment records a partial payment against an invoice
// Reduces outstanding balance, keeps status as Sent/Overdue until fully paid
// P1 FIX: Race condition protection via database transaction with row-level locking
// BE-8 FIX: Idempotency key prevents duplicate payments from network retries
func (a *App) RecordPartialPayment(id string, paymentAmount float64, paymentDate time.Time, paymentRef string) error {
	if err := a.requirePermission("payments:create"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// BE-8 FIX: Generate idempotency key from payment parameters
	// Same invoice + amount + date + ref = same key = same payment (idempotent)
	idempotencyData := fmt.Sprintf("%s|%.3f|%s|%s", id, paymentAmount, paymentDate.Format("2006-01-02"), paymentRef)
	idempotencyKey := fmt.Sprintf("%x", sha256.Sum256([]byte(idempotencyData)))

	// BE-8 FIX: Check if this payment was already processed (idempotency check)
	var existingPayment Payment
	if err := a.db.Where("idempotency_key = ?", idempotencyKey).First(&existingPayment).Error; err == nil {
		log.Printf("💰 Idempotent payment request - already processed: %s (Invoice: %s, Amount: %.3f)",
			idempotencyKey[:16]+"...", id, paymentAmount)
		return nil // Idempotent - return success without creating duplicate
	}

	// P1 FIX: Use database transaction with row-level locking to prevent race conditions
	// Multiple concurrent payments cannot create negative balances because:
	// 1. BEGIN IMMEDIATE acquires write lock on database (SQLite-specific)
	// 2. Read outstanding balance with SELECT ... FOR UPDATE (row lock in PostgreSQL)
	// 3. Validate payment <= outstanding atomically
	// 4. Update outstanding balance and create payment record atomically
	// 5. COMMIT releases lock - concurrent payments are serialized
	tx := a.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("❌ Transaction rolled back due to panic: %v", r)
		}
	}()

	// P0 FIX: Read invoice with SELECT FOR UPDATE row lock
	// This prevents concurrent payments from reading the same outstanding balance
	// Works with both SQLite (via transaction) and PostgreSQL (true row lock)
	var invoice Invoice
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&invoice).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("invoice not found: %w", err)
	}

	// Fix 5 + Mission I (I-01): Reject payments on non-payable invoices via the
	// settlement policy (the old inline map omitted "Draft").
	if !canRecordCustomerInvoicePayment(invoice, time.Now()) {
		tx.Rollback()
		return fmt.Errorf("cannot record payment on invoice with status '%s'", invoice.Status)
	}

	// Verify payment amount is valid
	if paymentAmount <= 0 {
		tx.Rollback()
		return fmt.Errorf("payment amount must be positive: %.3f", paymentAmount)
	}

	// CRITICAL: Validate payment against LOCKED outstanding balance
	// This is atomic - no other payment can modify outstanding between check and update
	if paymentAmount > invoice.OutstandingBHD+0.001 {
		tx.Rollback()
		return fmt.Errorf("payment amount (%.3f) exceeds outstanding balance (%.3f)",
			paymentAmount, invoice.OutstandingBHD)
	}

	// Calculate new outstanding balance
	newOutstanding := invoice.OutstandingBHD - paymentAmount

	// Mission I (I-04): status derived through the settlement policy (PH parity)
	previousOutstanding := invoice.OutstandingBHD
	invoice.OutstandingBHD = newOutstanding
	state, err := a.applyCustomerInvoicePaymentState(tx, &invoice)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record partial payment: %w", err)
	}

	// Create Payment record with idempotency key (I12 fix: was missing, broke idempotency check)
	payment := Payment{
		InvoiceID:      id,
		InvoiceNumber:  invoice.InvoiceNumber,
		AmountBHD:      paymentAmount,
		PaymentDate:    paymentDate,
		PaymentMethod:  "Bank Transfer",
		Reference:      paymentRef,
		IdempotencyKey: idempotencyKey,
		DaysToPayment:  int(time.Since(invoice.InvoiceDate).Hours() / 24),
	}
	payment.ID = uuid.New().String()
	if err := tx.Create(&payment).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create payment record: %w", err)
	}
	log.Printf("✅ Payment record created with idempotency key: %s...", idempotencyKey[:16])

	// Commit transaction - releases lock, makes changes visible
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit payment transaction: %w", err)
	}

	log.Printf("💰 Recorded Partial Payment: %.3f BHD for Invoice %s (Ref: %s, Outstanding: %.3f → %.3f BHD)",
		paymentAmount, invoice.InvoiceNumber, paymentRef, previousOutstanding, state.OutstandingBHD)

	return nil
}

// =============================================================================
// ANALYTICS & REPORTING
// =============================================================================

// GetInvoiceRevenueSummary calculates total revenue metrics
// Returns: (total_revenue, paid_revenue, outstanding_revenue, invoice_count)
func (a *App) GetInvoiceRevenueSummary() (float64, float64, float64, int64, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return 0, 0, 0, 0, err
	}
	if a.db == nil {
		return 0, 0, 0, 0, fmt.Errorf("database not initialized")
	}

	var totalRevenue, paidRevenue, outstandingRevenue float64
	var invoiceCount int64

	// Total revenue (all invoices)
	if err := a.db.Model(&Invoice{}).
		Select("COALESCE(SUM(grand_total_bhd), 0)").
		Scan(&totalRevenue).Error; err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to calculate total revenue: %w", err)
	}

	// Paid revenue
	if err := a.db.Model(&Invoice{}).
		Where("status = ?", "Paid").
		Select("COALESCE(SUM(grand_total_bhd), 0)").
		Scan(&paidRevenue).Error; err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to calculate paid revenue: %w", err)
	}

	// Outstanding revenue
	if err := a.db.Model(&Invoice{}).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&outstandingRevenue).Error; err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to calculate outstanding revenue: %w", err)
	}

	// Invoice count
	if err := a.db.Model(&Invoice{}).Count(&invoiceCount).Error; err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to count invoices: %w", err)
	}

	log.Printf("📊 Revenue Summary: Total=%.3f BHD, Paid=%.3f BHD, Outstanding=%.3f BHD, Count=%d",
		totalRevenue, paidRevenue, outstandingRevenue, invoiceCount)

	return totalRevenue, paidRevenue, outstandingRevenue, invoiceCount, nil
}

// GetInvoicesByDateRange retrieves invoices within a date range
func (a *App) GetInvoicesByDateRange(startDate, endDate time.Time) ([]Invoice, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var invoices []Invoice
	if err := a.db.Preload("Items").
		Where("invoice_date BETWEEN ? AND ?", startDate, endDate).
		Order("invoice_date DESC").
		Limit(500).
		Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve invoices by date range: %w", err)
	}

	log.Printf("📊 Retrieved %d invoices between %s and %s",
		len(invoices), startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	return invoices, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// calculateDueDate calculates invoice due date from invoice date and payment terms
// Supports formats: "Net 30", "Net 45", "Net 60", "COD", "Advance", etc.
func calculateDueDate(invoiceDate time.Time, paymentTerms string) time.Time {
	// Default to Net 30 if empty
	if paymentTerms == "" {
		return invoiceDate.AddDate(0, 0, 30)
	}

	// Normalize payment terms
	terms := strings.ToUpper(strings.TrimSpace(paymentTerms))

	// Parse different formats
	if strings.Contains(terms, "NET") {
		// Extract number from "Net 30", "NET 45", etc.
		parts := strings.Fields(terms)
		for _, part := range parts {
			if days, err := strconv.Atoi(part); err == nil {
				return invoiceDate.AddDate(0, 0, days)
			}
		}
	}

	// Special cases
	if strings.Contains(terms, "COD") || strings.Contains(terms, "CASH") {
		return invoiceDate // Due immediately
	}

	if strings.Contains(terms, "ADVANCE") || strings.Contains(terms, "PREPAY") {
		return invoiceDate.AddDate(0, 0, -1) // Due before invoice date
	}

	if strings.Contains(terms, "EOM") || strings.Contains(terms, "END OF MONTH") {
		// Due at end of month
		year, month, _ := invoiceDate.Date()
		lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, invoiceDate.Location())
		return lastDay
	}

	// Default to Net 30 if can't parse
	log.Printf("⚠️ Could not parse payment terms '%s', defaulting to Net 30", paymentTerms)
	return invoiceDate.AddDate(0, 0, 30)
}

// =============================================================================
// AR AGING & LATE PAYMENT TRACKING (P1 FINANCE FIXES)
// =============================================================================

// CalculateARAgingBuckets recalculates AR aging for all customers on-demand
// Returns aging buckets: Current (0-30), 30-60, 60-90, 90-120, 120+
func (a *App) CalculateARAgingBuckets() ([]ARAgingBucket, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	now := time.Now()
	var customers []CustomerMaster
	if err := a.db.Limit(1000).Find(&customers).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve customers: %w", err)
	}

	var agingBuckets []ARAgingBucket

	for _, customer := range customers {
		// Get all unpaid invoices for this customer
		var invoices []Invoice
		if err := a.db.Where("customer_id = ? AND status != ?", customer.CustomerID, "Paid").
			Find(&invoices).Error; err != nil {
			log.Printf("⚠️ Failed to get invoices for customer %s: %v", customer.CustomerID, err)
			continue
		}

		if len(invoices) == 0 {
			continue // Skip customers with no outstanding invoices
		}

		// Initialize aging bucket
		bucket := ARAgingBucket{
			Base: Base{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			CustomerID:   customer.CustomerID,
			CustomerName: customer.BusinessName,
			SnapshotDate: now,
		}

		var oldestOverdueDays int

		// Categorize each invoice into aging buckets
		for _, inv := range invoices {
			daysOverdue := int(now.Sub(inv.DueDate).Hours() / 24)
			outstanding := inv.OutstandingBHD

			// Categorize by age
			if daysOverdue <= 0 {
				// Current (not yet due)
				bucket.Less15Days += outstanding
			} else if daysOverdue <= 30 {
				bucket.Days16_30 += outstanding
			} else if daysOverdue <= 60 {
				bucket.Days31_60 += outstanding
			} else if daysOverdue <= 90 {
				bucket.Days61_90 += outstanding
			} else {
				bucket.Over90Days += outstanding
			}

			// Track oldest overdue
			if daysOverdue > oldestOverdueDays {
				oldestOverdueDays = daysOverdue
			}

			bucket.TotalOutstanding += outstanding
		}

		// Calculate total overdue (everything except current)
		bucket.TotalOverdue = bucket.Days16_30 + bucket.Days31_60 + bucket.Days61_90 + bucket.Over90Days
		bucket.OverdueDays = oldestOverdueDays

		// Calculate risk tier based on aging distribution
		if bucket.Over90Days > bucket.TotalOutstanding*0.5 {
			bucket.RiskTier = "Critical"
			bucket.RiskScore = 1.0
		} else if bucket.Days61_90+bucket.Over90Days > bucket.TotalOutstanding*0.3 {
			bucket.RiskTier = "High"
			bucket.RiskScore = 0.75
		} else if bucket.Days31_60+bucket.Days61_90+bucket.Over90Days > bucket.TotalOutstanding*0.2 {
			bucket.RiskTier = "Medium"
			bucket.RiskScore = 0.5
		} else {
			bucket.RiskTier = "Low"
			bucket.RiskScore = 0.25
		}

		agingBuckets = append(agingBuckets, bucket)

		// Update customer's AR fields
		if err := a.db.Model(&customer).Updates(map[string]any{
			"outstanding_bhd": bucket.TotalOutstanding,
			"overdue_days":    bucket.OverdueDays,
			"ar_risk_tier":    bucket.RiskTier,
		}).Error; err != nil {
			log.Printf("⚠️ Failed to update customer AR fields: %v", err)
		}
	}

	// Save aging buckets to database
	for _, bucket := range agingBuckets {
		// Clean up old aging snapshots (keep only the latest)
		a.db.Where("customer_id = ?", bucket.CustomerID).Delete(&ARAgingBucket{})
		if err := a.db.Create(&bucket).Error; err != nil {
			log.Printf("⚠️ Failed to save AR aging bucket for customer %s: %v", bucket.CustomerID, err)
		}
	}

	log.Printf("✅ Calculated AR aging for %d customers", len(agingBuckets))
	return agingBuckets, nil
}

// GetARAgingByCustomer retrieves the latest AR aging bucket for a customer
func (a *App) GetARAgingByCustomer(customerID string) (ARAgingBucket, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return ARAgingBucket{}, err
	}
	if a.db == nil {
		return ARAgingBucket{}, fmt.Errorf("database not initialized")
	}

	var bucket ARAgingBucket
	if err := a.db.Where("customer_id = ?", customerID).
		Order("snapshot_date DESC").
		First(&bucket).Error; err != nil {
		return ARAgingBucket{}, fmt.Errorf("AR aging not found for customer: %w", err)
	}

	return bucket, nil
}

// GetLatePaymentInvoices returns all invoices past due date
// Flags invoices > 30 days overdue as critical
func (a *App) GetLatePaymentInvoices() ([]Invoice, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	now := time.Now()
	var invoices []Invoice
	if err := a.db.Preload("Items").
		Where("due_date < ? AND status != ?", now, "Paid").
		Order("due_date ASC").
		Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve late payment invoices: %w", err)
	}

	// Calculate days past due for each invoice
	for i := range invoices {
		daysPastDue := int(now.Sub(invoices[i].DueDate).Hours() / 24)
		invoices[i].UpdatedAt = time.Now() // Use UpdatedAt to store calculated value temporarily

		if daysPastDue > 30 {
			log.Printf("🚨 CRITICAL: Invoice %s is %d days overdue (Customer: %s, Amount: %.3f BHD)",
				invoices[i].InvoiceNumber, daysPastDue, invoices[i].CustomerName, invoices[i].OutstandingBHD)
		}
	}

	log.Printf("⚠️ Retrieved %d late payment invoices", len(invoices))
	return invoices, nil
}

// TrackLatePaymentHistory calculates late payment metrics for a customer
// Returns: (total_invoices, late_invoices, avg_days_late, max_days_late)
func (a *App) TrackLatePaymentHistory(customerID string) (int64, int64, float64, int, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return 0, 0, 0, 0, err
	}
	if a.db == nil {
		return 0, 0, 0, 0, fmt.Errorf("database not initialized")
	}

	// Get all paid invoices for customer
	var payments []Payment
	if err := a.db.Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("invoices.customer_id = ?", customerID).
		Find(&payments).Error; err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to retrieve payment history: %w", err)
	}

	if len(payments) == 0 {
		return 0, 0, 0, 0, nil
	}

	var totalInvoices int64 = int64(len(payments))
	var lateInvoices int64 = 0
	var totalDaysLate int = 0
	var maxDaysLate int = 0

	for _, payment := range payments {
		// Get invoice to check due date
		var invoice Invoice
		if err := a.db.Where("id = ?", payment.InvoiceID).First(&invoice).Error; err != nil {
			continue
		}

		// Calculate days past due (payment_date - due_date)
		daysPastDue := int(payment.PaymentDate.Sub(invoice.DueDate).Hours() / 24)

		if daysPastDue > 0 {
			lateInvoices++
			totalDaysLate += daysPastDue

			if daysPastDue > maxDaysLate {
				maxDaysLate = daysPastDue
			}
		}
	}

	var avgDaysLate float64 = 0
	if lateInvoices > 0 {
		avgDaysLate = float64(totalDaysLate) / float64(lateInvoices)
	}

	log.Printf("📊 Late payment history for customer %s: %d/%d invoices late (avg: %.1f days, max: %d days)",
		customerID, lateInvoices, totalInvoices, avgDaysLate, maxDaysLate)

	return totalInvoices, lateInvoices, avgDaysLate, maxDaysLate, nil
}

// =============================================================================
// DELIVERY NOTE LINKAGE
// =============================================================================

// GetAvailableDeliveryNotesForOrder retrieves delivery notes that can be linked to an invoice
// Returns DNs for the specified order that are not already linked to another invoice
func (a *App) GetAvailableDeliveryNotesForOrder(orderID string) ([]DeliveryNote, error) {
	if err := a.requirePermission("delivery_notes:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Get all DNs for this order
	var allDNs []DeliveryNote
	if err := a.db.Preload("Items").Where("order_id = ?", orderID).Order("delivery_date DESC").Find(&allDNs).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve delivery notes: %w", err)
	}

	// Get DN IDs already linked to invoices
	var linkedDNIDs []string
	if err := a.db.Model(&Invoice{}).
		Where("delivery_note_id IS NOT NULL AND delivery_note_id != ''").
		Pluck("delivery_note_id", &linkedDNIDs).Error; err != nil {
		log.Printf("⚠️ Failed to get linked DN IDs: %v", err)
		// Continue with all DNs if this query fails
		return allDNs, nil
	}

	// Create a set for quick lookup
	linkedSet := make(map[string]bool)
	for _, id := range linkedDNIDs {
		linkedSet[id] = true
	}

	// Filter out linked DNs
	var availableDNs []DeliveryNote
	for _, dn := range allDNs {
		if !linkedSet[dn.ID] {
			availableDNs = append(availableDNs, dn)
		}
	}

	log.Printf("📦 Found %d available DNs for order %s (total: %d, linked: %d)",
		len(availableDNs), orderID, len(allDNs), len(linkedDNIDs))
	return availableDNs, nil
}

// countFutureDatedInvoices counts invoices dated after today. The startup
// audit uses this instead of a fixed year cutoff: a hardcoded ">= 2026"
// flags every live-FY2026 invoice forever once 2026 is the current year.
// Future-dated relative to now is the durable definition of "wrong date".
func (a *App) countFutureDatedInvoices() int64 {
	if a.db == nil {
		return 0
	}
	var count int64
	a.db.Model(&Invoice{}).
		Where("date(invoice_date) > date(?)", time.Now().Format("2006-01-02")).
		Count(&count)
	return count
}
