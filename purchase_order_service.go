package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"gorm.io/gorm/clause"
	crmprocurement "ph_holdings_app/pkg/crm/procurement"
	"ph_holdings_app/pkg/documents/numbering"

	"ph_holdings_app/pkg/crm/supplierlink"
)

func isMalformedDraftPO(po PurchaseOrder) bool {
	return po.Status == "Draft" &&
		strings.TrimSpace(po.SupplierID) == "" &&
		strings.TrimSpace(po.SupplierName) == "" &&
		len(po.Items) == 0
}

func filterMalformedDraftPOs(pos []PurchaseOrder) []PurchaseOrder {
	filtered := make([]PurchaseOrder, 0, len(pos))
	for _, po := range pos {
		if isMalformedDraftPO(po) {
			log.Printf("⚠️ Skipping malformed draft PO %s from listings", po.PONumber)
			continue
		}
		filtered = append(filtered, po)
	}
	return filtered
}

// resolveSupplierForOrderItem resolves ONE order item to its supplier.
// Band-2 rows 15-16: the item resolves through the supplierlink engine's
// fallback chain (product supplier ID → code → canonical code → commercial
// token), with a free-text token fallback for items whose product row is
// missing or unlinkable. The old code read only product.supplier_id, so a
// stale or placeholder link failed the whole PO inference.
func (a *App) resolveSupplierForOrderItem(orderItem OrderItem) *SupplierMaster {
	var supplier *SupplierMaster
	if productID := strings.TrimSpace(orderItem.ProductID); productID != "" {
		var product ProductMaster
		if err := a.db.First(&product, "id = ?", productID).Error; err == nil {
			supplier, _ = supplierlink.ResolveSupplierForProduct(a.db, product, supplierLinkAliases())
		}
	}
	if supplier == nil {
		for _, token := range orderItemSupplierTokens(orderItem) {
			if s, err := supplierlink.FindSupplierByCommercialToken(a.db, token, supplierLinkAliases()); err == nil {
				supplier = s
				break
			}
		}
	}
	return supplier
}

func (a *App) inferSupplierForOrderItems(order Order, itemIDs []string) (SupplierMaster, error) {
	selectedItemIDs := make(map[string]struct{}, len(itemIDs))
	for _, itemID := range itemIDs {
		if trimmed := strings.TrimSpace(itemID); trimmed != "" {
			selectedItemIDs[trimmed] = struct{}{}
		}
	}

	resolved := make(map[string]SupplierMaster)
	for _, orderItem := range order.Items {
		if len(selectedItemIDs) > 0 {
			if _, ok := selectedItemIDs[orderItem.ID]; !ok {
				continue
			}
		}

		if supplier := a.resolveSupplierForOrderItem(orderItem); supplier != nil {
			resolved[supplier.ID] = *supplier
		}
	}

	if len(resolved) == 0 {
		return SupplierMaster{}, newError("SUPPLIER_NOT_FOUND", "Unable to determine supplier from selected order items", "")
	}
	if len(resolved) > 1 {
		return SupplierMaster{}, newError("MULTIPLE_SUPPLIERS", "Selected order items belong to multiple suppliers. Please create supplier POs separately.", "")
	}

	for _, supplier := range resolved {
		return supplier, nil
	}
	return SupplierMaster{}, newError("SUPPLIER_NOT_FOUND", "Unable to determine supplier from selected order items", "")
}

// groupOrderItemsByInferredSupplier resolves every (selected) order line item
// to a supplier and groups the item IDs per supplier — the input for
// CreatePOsFromOrder's one-PO-per-supplier split. Unlike the single-supplier
// inference above, an unresolvable item is a hard error here (a wrong supplier
// on a PO is worse than asking the user). Ported from deployed PH, resolution
// upgraded to the supplierlink engine.
func (a *App) groupOrderItemsByInferredSupplier(order Order, itemIDs []string) (map[string][]string, map[string]SupplierMaster, error) {
	selectedItemIDs := make(map[string]struct{}, len(itemIDs))
	for _, itemID := range itemIDs {
		if trimmed := strings.TrimSpace(itemID); trimmed != "" {
			selectedItemIDs[trimmed] = struct{}{}
		}
	}

	groups := make(map[string][]string)
	suppliers := make(map[string]SupplierMaster)
	matched := 0
	for _, item := range order.Items {
		if len(selectedItemIDs) > 0 {
			if _, ok := selectedItemIDs[item.ID]; !ok {
				continue
			}
		}
		matched++
		if strings.TrimSpace(item.ID) == "" {
			return nil, nil, newError("ORDER_ITEM_ID_MISSING", "Order line item is missing an ID and cannot be routed to a supplier PO", item.Description)
		}
		supplier := a.resolveSupplierForOrderItem(item)
		if supplier == nil {
			label := firstNonEmptyString(item.ProductCode, item.Model, item.Description, item.ID)
			return nil, nil, newError("SUPPLIER_NOT_FOUND", "Unable to determine supplier for order line item", label)
		}
		groups[supplier.ID] = append(groups[supplier.ID], item.ID)
		suppliers[supplier.ID] = *supplier
	}
	if matched == 0 {
		return nil, nil, newError("NO_ITEMS", "No order line items are available for supplier PO creation", "")
	}
	return groups, suppliers, nil
}

// CreatePOsFromOrder splits a customer order into one draft purchase order per
// inferred supplier (deterministic supplier-name ordering) — the enforced,
// deliberate PO-creation flow that replaces MarkOfferWon's auto-Draft-PO.
func (a *App) CreatePOsFromOrder(orderID string, itemIDs []string) ([]PurchaseOrder, error) {
	if err := a.requirePermission("po:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var order Order
	if err := a.db.Preload("Items").First(&order, "id = ?", strings.TrimSpace(orderID)).Error; err != nil {
		return nil, newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	groups, suppliers, err := a.groupOrderItemsByInferredSupplier(order, itemIDs)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, newError("NO_ITEMS", "No items found for supplier PO creation", "")
	}

	supplierIDs := make([]string, 0, len(groups))
	for supplierID := range groups {
		supplierIDs = append(supplierIDs, supplierID)
	}
	sort.Slice(supplierIDs, func(i, j int) bool {
		left := suppliers[supplierIDs[i]].SupplierName
		right := suppliers[supplierIDs[j]].SupplierName
		if strings.EqualFold(left, right) {
			return supplierIDs[i] < supplierIDs[j]
		}
		return strings.ToLower(left) < strings.ToLower(right)
	})

	created := make([]PurchaseOrder, 0, len(supplierIDs))
	for _, supplierID := range supplierIDs {
		po, err := a.CreatePOFromOrder(orderID, supplierID, groups[supplierID])
		if err != nil {
			return created, err
		}
		created = append(created, po)
	}
	return created, nil
}

// orderItemSupplierTokens extracts free-text commercial tokens from an order
// item for last-resort supplier inference: the leading segment of the
// brand-bearing fields (equipment, product code) before the first commercial
// delimiter (e.g. "SVX-2200" → "SVX"). Descriptions are deliberately not
// mined — their leading words are product nouns, not brands, and a wrong
// supplier on a PO is worse than asking the user.
func orderItemSupplierTokens(item OrderItem) []string {
	tokens := make([]string, 0, 2)
	for _, source := range []string{item.Equipment, item.ProductCode} {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		if idx := strings.IndexAny(source, "-|/,: "); idx > 0 {
			source = source[:idx]
		}
		tokens = append(tokens, source)
	}
	return tokens
}

func (a *App) normalizePurchaseOrder(po *PurchaseOrder) error {
	if strings.TrimSpace(po.SupplierID) == "" {
		return newError("SUPPLIER_REQUIRED", "Supplier is required", "")
	}

	var supplier SupplierMaster
	if err := a.db.First(&supplier, "id = ?", po.SupplierID).Error; err != nil {
		return newError("SUPPLIER_NOT_FOUND", "Supplier not found", err.Error())
	}
	po.SupplierName = supplier.SupplierName

	po.Currency = strings.ToUpper(strings.TrimSpace(po.Currency))
	if po.Currency == "" {
		po.Currency = "BHD"
	}
	po.ExchangeRate = normalizeExchangeRateToBHD(po.Currency, po.ExchangeRate)
	if strings.TrimSpace(po.Division) == "" {
		po.Division = a.resolveOrderDivision(po.OrderID)
	} else {
		po.Division = normalizeDivisionName(po.Division)
	}

	if len(po.Items) == 0 {
		return newError("ITEMS_REQUIRED", "At least one purchase order line item is required", "")
	}

	normalizedItems := make([]PurchaseOrderItem, 0, len(po.Items))
	subtotalForeign := 0.0
	for i := range po.Items {
		item := po.Items[i]
		item.Description = strings.TrimSpace(item.Description)
		if item.Description == "" {
			return newError("INVALID_ITEM", fmt.Sprintf("Line item %d is missing a description", i+1), "")
		}
		if item.Quantity <= 0 {
			return newError("INVALID_ITEM", fmt.Sprintf("Line item %d must have quantity greater than zero", i+1), "")
		}
		if item.UnitPriceForeign <= 0 {
			if item.UnitPriceBHD > 0 && po.ExchangeRate > 0 {
				item.UnitPriceForeign = item.UnitPriceBHD / po.ExchangeRate
			} else {
				return newError("INVALID_ITEM", fmt.Sprintf("Line item %d must have a unit price greater than zero", i+1), "")
			}
		}

		item.UnitPriceForeign = roundTo3(item.UnitPriceForeign)
		item.TotalForeign = roundTo3(item.Quantity * item.UnitPriceForeign)
		item.UnitPriceBHD = roundTo3(item.UnitPriceForeign * po.ExchangeRate)
		item.TotalBHD = roundTo3(item.TotalForeign * po.ExchangeRate)

		normalizedItems = append(normalizedItems, item)
		subtotalForeign += item.TotalForeign
	}

	po.Items = normalizedItems
	po.SubtotalForeign = roundTo3(subtotalForeign)
	po.SubtotalBHD = roundTo3(po.SubtotalForeign * po.ExchangeRate)
	// VATAmount is a BHD figure (it feeds TotalBHD and any input-VAT read of
	// the field), so it is computed on the BHD subtotal — not the foreign one.
	po.VATAmount = roundTo3(po.SubtotalBHD * 0.10)
	po.TotalForeign = roundTo3(po.SubtotalForeign * 1.10)
	po.TotalBHD = roundTo3(po.SubtotalBHD + po.VATAmount)

	if po.PaymentDueDate.IsZero() && po.PaymentTerms != "" {
		po.PaymentDueDate = calculatePaymentDueDate(po.PODate, po.PaymentTerms)
	}

	return nil
}

// =============================================================================
// PURCHASE ORDER CRUD OPERATIONS
// =============================================================================

// CreatePurchaseOrder creates a new purchase order
func (a *App) CreatePurchaseOrder(po PurchaseOrder) (PurchaseOrder, error) {
	return a.procurementService().CreatePurchaseOrder(po)
}

func createPurchaseOrder(a *App, po PurchaseOrder) (PurchaseOrder, error) {
	if err := a.requirePermission("po:create"); err != nil {
		return PurchaseOrder{}, err
	}
	if a.db == nil {
		return PurchaseOrder{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Auto-generate PO number if not provided
	if po.PONumber == "" {
		poNumber, err := a.GeneratePONumber()
		if err != nil {
			return PurchaseOrder{}, err
		}
		po.PONumber = poNumber
	}

	// Set default dates if not provided
	if po.PODate.IsZero() {
		po.PODate = time.Now()
	}

	if err := a.normalizePurchaseOrder(&po); err != nil {
		return PurchaseOrder{}, err
	}

	// Store items separately and create them explicitly so malformed associations
	// cannot leave behind half-baked draft POs.
	items := po.Items
	po.Items = nil

	tx := a.db.Begin()
	if tx.Error != nil {
		return PurchaseOrder{}, newError("DB_TX_FAILED", "Failed to begin purchase order transaction", tx.Error.Error())
	}

	// Force initial status — caller cannot bypass workflow
	po.Status = "Draft"

	// P1 FIX: Set CreatedBy for segregation-of-duties check in ApprovePurchaseOrder
	po.CreatedBy = a.getCurrentUserID()

	// P3 FIX (PO-26): Default currency fallback for log messages
	currency := po.Currency
	if currency == "" {
		currency = "BHD"
	}

	// P1 FIX: Check if PO requires approval based on amount threshold
	approvalThreshold := 5000.0 // BHD - configurable threshold
	if po.TotalBHD > approvalThreshold && po.Status == "Draft" {
		po.Status = "Pending Approval"
		log.Printf("⚠️ PO %s requires approval (%.2f %s > %.2f %s threshold)", po.PONumber, po.TotalBHD, currency, approvalThreshold, currency)
	}

	if err := tx.Create(&po).Error; err != nil {
		tx.Rollback()
		log.Printf("❌ Failed to create purchase order: %v", err)
		return PurchaseOrder{}, newError("DB_CREATE_FAILED", "Failed to create purchase order", err.Error())
	}

	for i := range items {
		items[i].PurchaseOrderID = po.ID
		if err := tx.Create(&items[i]).Error; err != nil {
			tx.Rollback()
			log.Printf("❌ Failed to create purchase order item: %v", err)
			return PurchaseOrder{}, newError("DB_CREATE_FAILED", "Failed to create purchase order item", err.Error())
		}
	}
	po.Items = items

	if err := tx.Commit().Error; err != nil {
		return PurchaseOrder{}, newError("DB_COMMIT_FAILED", "Failed to commit purchase order", err.Error())
	}

	log.Printf("✅ Created Purchase Order %s for Supplier %s (%.2f %s)", po.PONumber, po.SupplierID, po.TotalBHD, currency)
	return po, nil
}

// GetPurchaseOrders retrieves all purchase orders with items preloaded and supplier names enriched
func (a *App) GetPurchaseOrders() ([]PurchaseOrder, error) {
	return a.procurementService().GetPurchaseOrders()
}

func getPurchaseOrders(a *App) ([]PurchaseOrder, error) {
	if err := a.requirePermission("po:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var pos []PurchaseOrder
	if err := a.db.Preload("Items").Order("po_date DESC").Limit(500).Find(&pos).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve purchase orders", err.Error())
	}

	// Enrich with supplier names
	a.enrichPOsWithSupplierNames(pos)

	// Wave 9.8 B1: overlay per-line RequiresSerialTracking from ProductMaster
	// (query-time only, gorm:"-" — mirrors delivery_note_service.go's
	// OrderFulfillmentItem enrichment) so the PO receive panel can enforce
	// per-line serial capture.
	a.enrichPOItemsWithSerialTracking(pos)

	return filterMalformedDraftPOs(pos), nil
}

// enrichPOItemsWithSerialTracking batch-fetches ProductMaster.RequiresSerialTracking
// for every distinct ProductID across the given POs' items and stamps it onto
// each PurchaseOrderItem (query-time overlay field, not persisted).
func (a *App) enrichPOItemsWithSerialTracking(pos []PurchaseOrder) {
	productIDs := make(map[string]bool)
	for _, po := range pos {
		for _, item := range po.Items {
			if strings.TrimSpace(item.ProductID) != "" {
				productIDs[item.ProductID] = true
			}
		}
	}
	if len(productIDs) == 0 {
		return
	}

	ids := make([]string, 0, len(productIDs))
	for id := range productIDs {
		ids = append(ids, id)
	}

	var products []ProductMaster
	if err := a.db.Where("id IN ?", ids).Find(&products).Error; err != nil {
		log.Printf("Warning: Failed to fetch products for PO serial-tracking enrichment: %v", err)
		return
	}

	serialByProduct := make(map[string]bool, len(products))
	for _, p := range products {
		serialByProduct[p.ID] = p.RequiresSerialTracking
	}

	for i := range pos {
		for j := range pos[i].Items {
			pos[i].Items[j].RequiresSerialTracking = serialByProduct[pos[i].Items[j].ProductID]
		}
	}
}

// enrichPOsWithSupplierNames populates SupplierName for POs missing it
func (a *App) enrichPOsWithSupplierNames(pos []PurchaseOrder) {
	// Collect unique supplier IDs that need lookup
	supplierIDs := make(map[string]bool)
	for _, po := range pos {
		if po.SupplierID != "" && po.SupplierName == "" {
			supplierIDs[po.SupplierID] = true
		}
	}

	if len(supplierIDs) == 0 {
		return
	}

	// Batch lookup suppliers
	ids := make([]string, 0, len(supplierIDs))
	for id := range supplierIDs {
		ids = append(ids, id)
	}

	var suppliers []SupplierMaster
	if err := a.db.Where("id IN ?", ids).Find(&suppliers).Error; err != nil {
		log.Printf("Warning: Failed to fetch suppliers for PO enrichment: %v", err)
		return
	}

	// Build lookup map
	supplierMap := make(map[string]string)
	for _, s := range suppliers {
		supplierMap[s.ID] = s.SupplierName
	}

	// Enrich POs
	for i := range pos {
		if pos[i].SupplierName == "" && pos[i].SupplierID != "" {
			if name, ok := supplierMap[pos[i].SupplierID]; ok {
				pos[i].SupplierName = name
			} else {
				pos[i].SupplierName = "Unknown Supplier"
			}
		}
	}
}

// GetPurchaseOrderByID retrieves a single purchase order by ID with items preloaded
func (a *App) GetPurchaseOrderByID(id string) (PurchaseOrder, error) {
	return a.procurementService().GetPurchaseOrderByID(id)
}

func getPurchaseOrderByID(a *App, id string) (PurchaseOrder, error) {
	if err := a.requirePermission("po:view"); err != nil {
		return PurchaseOrder{}, err
	}
	if a.db == nil {
		return PurchaseOrder{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var po PurchaseOrder
	if err := a.db.Preload("Items").First(&po, "id = ?", id).Error; err != nil {
		return PurchaseOrder{}, newError("PO_NOT_FOUND", "Purchase order not found", err.Error())
	}
	if isMalformedDraftPO(po) {
		return PurchaseOrder{}, newError("PO_NOT_FOUND", "Purchase order not found", "malformed draft purchase order")
	}

	// Enrich with supplier name
	if po.SupplierName == "" && po.SupplierID != "" {
		var supplier SupplierMaster
		if err := a.db.First(&supplier, "id = ?", po.SupplierID).Error; err == nil {
			po.SupplierName = supplier.SupplierName
		}
	}

	// Wave 9.8 B1: overlay per-line RequiresSerialTracking (see
	// enrichPOItemsWithSerialTracking) so the receive panel can enforce
	// per-line serial capture for this single PO too.
	a.enrichPOItemsWithSerialTracking([]PurchaseOrder{po})

	return po, nil
}

// GetPurchaseOrdersByOrder retrieves all purchase orders for a given customer order
func (a *App) GetPurchaseOrdersByOrder(orderID string) ([]PurchaseOrder, error) {
	if err := a.requirePermission("po:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var pos []PurchaseOrder
	if err := a.db.Preload("Items").Where("order_id = ?", orderID).Order("po_date DESC").Find(&pos).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve purchase orders for order", err.Error())
	}

	// Enrich with supplier names
	a.enrichPOsWithSupplierNames(pos)

	return filterMalformedDraftPOs(pos), nil
}

// GetPurchaseOrdersBySupplier retrieves all purchase orders for a given supplier
func (a *App) GetPurchaseOrdersBySupplier(supplierID string) ([]PurchaseOrder, error) {
	if err := a.requirePermission("po:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var pos []PurchaseOrder
	if err := a.db.Preload("Items").Where("supplier_id = ?", supplierID).Order("po_date DESC").Find(&pos).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve purchase orders for supplier", err.Error())
	}

	a.enrichPOsWithSupplierNames(pos)
	return filterMalformedDraftPOs(pos), nil
}

// UpdatePurchaseOrder updates an existing purchase order
func (a *App) UpdatePurchaseOrder(po PurchaseOrder) (PurchaseOrder, error) {
	return a.procurementService().UpdatePurchaseOrder(po)
}

func updatePurchaseOrder(a *App, po PurchaseOrder) (PurchaseOrder, error) {
	if err := a.requirePermission("po:update"); err != nil {
		return PurchaseOrder{}, err
	}
	if a.db == nil {
		return PurchaseOrder{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Verify PO exists
	var existing PurchaseOrder
	if err := a.db.First(&existing, "id = ?", po.ID).Error; err != nil {
		return PurchaseOrder{}, newError("PO_NOT_FOUND", "Purchase order not found", err.Error())
	}

	// Block financial field changes on post-approval POs (P2 Fix: expanded to cover all post-approval statuses)
	protectedStatuses := map[string]bool{
		"Pending Approval":   true,
		"Approved":           true,
		"Sent":               true,
		"Acknowledged":       true,
		"Partially Received": true,
		"Received":           true,
		"Closed":             true,
	}
	if protectedStatuses[existing.Status] {
		if po.SubtotalForeign != existing.SubtotalForeign || po.SubtotalBHD != existing.SubtotalBHD ||
			po.TotalForeign != existing.TotalForeign || po.TotalBHD != existing.TotalBHD ||
			po.ExchangeRate != existing.ExchangeRate || po.VATAmount != existing.VATAmount {
			return PurchaseOrder{}, fmt.Errorf("cannot modify financial fields on PO with status '%s' - requires new PO or status reset to Draft", existing.Status)
		}
	}

	if err := a.normalizePurchaseOrder(&po); err != nil {
		return PurchaseOrder{}, err
	}

	// Update only the fields that should be editable
	// This preserves: status, approved_by, approved_at, rfq_id, created_by, version
	updates := map[string]any{
		"supplier_id":       po.SupplierID,
		"supplier_name":     po.SupplierName,
		"po_date":           po.PODate,
		"expected_delivery": po.ExpectedDelivery,
		"currency":          po.Currency,
		"exchange_rate":     po.ExchangeRate,
		"subtotal_foreign":  po.SubtotalForeign,
		"subtotal_bhd":      po.SubtotalBHD,
		"vat_amount":        po.VATAmount,
		"total_foreign":     po.TotalForeign,
		"total_bhd":         po.TotalBHD,
		"payment_terms":     po.PaymentTerms,
		"division":          po.Division,
		"updated_at":        time.Now(),
	}

	// Allow PO number edit only in Draft status
	if existing.Status == "Draft" && po.PONumber != "" && po.PONumber != existing.PONumber {
		// Uniqueness check: verify no other PO has the same number
		var count int64
		a.db.Model(&PurchaseOrder{}).Where("po_number = ? AND id != ?", po.PONumber, po.ID).Count(&count)
		if count > 0 {
			return PurchaseOrder{}, fmt.Errorf("PO number %s already exists", po.PONumber)
		}
		updates["po_number"] = po.PONumber
	}

	if err := a.db.Model(&PurchaseOrder{}).Where("id = ?", po.ID).Updates(updates).Error; err != nil {
		log.Printf("❌ Failed to update purchase order: %v", err)
		return PurchaseOrder{}, newError("DB_UPDATE_FAILED", "Failed to update purchase order", err.Error())
	}

	// If items were updated, handle them atomically (delete + create in transaction)
	if len(po.Items) > 0 {
		tx := a.db.Begin()
		if tx.Error != nil {
			return PurchaseOrder{}, newError("DB_TX_FAILED", "Failed to begin transaction for items", tx.Error.Error())
		}

		// Delete existing items
		if err := tx.Where("purchase_order_id = ?", po.ID).Delete(&PurchaseOrderItem{}).Error; err != nil {
			tx.Rollback()
			return PurchaseOrder{}, newError("DB_DELETE_FAILED", "Failed to delete existing PO items", err.Error())
		}

		// Create new items
		for i := range po.Items {
			item := &po.Items[i]
			item.PurchaseOrderID = po.ID
			if po.ExchangeRate > 0 {
				item.UnitPriceBHD = item.UnitPriceForeign * po.ExchangeRate
				item.TotalBHD = item.TotalForeign * po.ExchangeRate
			}
			if err := tx.Create(item).Error; err != nil {
				tx.Rollback()
				return PurchaseOrder{}, newError("DB_CREATE_FAILED", "Failed to create PO item", err.Error())
			}
		}

		if err := tx.Commit().Error; err != nil {
			return PurchaseOrder{}, newError("DB_COMMIT_FAILED", "Failed to commit PO items update", err.Error())
		}
	}

	log.Printf("✅ Updated Purchase Order %s", po.PONumber)

	// Reload with items
	return a.GetPurchaseOrderByID(po.ID)
}

// UpdatePOStatus updates only the status of a purchase order
func (a *App) UpdatePOStatus(id string, status string) error {
	if err := a.requirePermission("po:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate status transition
	validTransitions := map[string][]string{
		"Draft":              {"Pending Approval", "Approved", "Sent", "Cancelled"},
		"Pending Approval":   {"Approved", "Draft", "Cancelled"},
		"Approved":           {"Sent", "Cancelled"},
		"Sent":               {"Acknowledged", "Partially Received", "Received", "Cancelled"},
		"Acknowledged":       {"Partially Received", "Received", "Cancelled"},
		"Partially Received": {"Received", "Cancelled"},
		"Received":           {}, // Terminal state
		"Closed":             {}, // Terminal state
		"Cancelled":          {}, // Terminal state
	}

	// P2 FIX: Wrap SELECT + validation + UPDATE in a transaction with row lock
	return a.db.Transaction(func(tx *gorm.DB) error {
		var current PurchaseOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("status, total_bhd").First(&current, "id = ?", id).Error; err != nil {
			return newError("PO_NOT_FOUND", "Purchase order not found", err.Error())
		}

		allowed, knownStatus := validTransitions[current.Status]
		if !knownStatus {
			return fmt.Errorf("PO has unrecognized current status '%s'", current.Status)
		}

		isValid := false
		for _, s := range allowed {
			if s == status {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid status transition from '%s' to '%s'", current.Status, status)
		}

		// P1 FIX: Block Draft→Sent transition for POs above approval threshold
		if current.Status == "Draft" && status == "Sent" {
			approvalThreshold := 5000.0
			if current.TotalBHD > approvalThreshold {
				return fmt.Errorf("cannot send PO directly from Draft: total %.3f BHD exceeds approval threshold %.3f BHD — must be approved first", current.TotalBHD, approvalThreshold)
			}
		}

		// P1 FIX: Block Draft→Approved for POs above approval threshold
		// Must go through Pending Approval → Approved (via ApprovePurchaseOrder)
		if current.Status == "Draft" && status == "Approved" {
			approvalThreshold := 5000.0
			if current.TotalBHD > approvalThreshold {
				return fmt.Errorf("cannot approve PO directly from Draft: total %.3f BHD exceeds approval threshold %.3f BHD — must go through Pending Approval first", current.TotalBHD, approvalThreshold)
			}
		}

		result := tx.Model(&PurchaseOrder{}).Where("id = ?", id).Update("status", status)
		if result.Error != nil {
			return newError("DB_UPDATE_FAILED", "Failed to update PO status", result.Error.Error())
		}

		if result.RowsAffected == 0 {
			return newError("PO_NOT_FOUND", "Purchase order not found", "")
		}

		log.Printf("✅ Updated PO #%s status to: %s", id, status)
		return nil
	})
}

// =============================================================================
// P1 FIX: PURCHASE ORDER APPROVAL WORKFLOW
// =============================================================================

// ApprovePurchaseOrder approves a purchase order requiring approval
func (a *App) ApprovePurchaseOrder(id string, approvedBy string) error {
	return a.procurementService().ApprovePurchaseOrder(id, approvedBy)
}

func approvePurchaseOrder(a *App, id string, approvedBy string) error {
	if err := a.requirePermission("po:approve"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get PO
	var po PurchaseOrder
	if err := a.db.First(&po, "id = ?", id).Error; err != nil {
		return newError("PO_NOT_FOUND", "Purchase order not found", err.Error())
	}

	// Verify status allows approval
	if po.Status != "Pending Approval" {
		return newError("INVALID_STATUS", fmt.Sprintf("Cannot approve PO with status: %s", po.Status), "")
	}

	// P2 FIX: Segregation of duties — approver must not be the creator
	if po.CreatedBy != "" && po.CreatedBy == approvedBy {
		return newError("SEGREGATION_VIOLATION",
			fmt.Sprintf("Cannot approve PO %s: approver '%s' is the same as creator (segregation of duties)", po.PONumber, approvedBy), "")
	}

	// P1 FIX: Update status and approval tracking with proper fields
	now := time.Now()
	updates := map[string]any{
		"status":      "Approved",
		"approved_by": approvedBy,
		"approved_at": now,
		"updated_by":  approvedBy,
	}

	if err := a.db.Model(&po).Updates(updates).Error; err != nil {
		return newError("DB_UPDATE_FAILED", "Failed to approve purchase order", err.Error())
	}

	// P3 FIX (PO-26): Default currency fallback for log messages
	currency := po.Currency
	if currency == "" {
		currency = "BHD"
	}
	log.Printf("✅ Approved Purchase Order %s by %s (%.2f %s)", po.PONumber, approvedBy, po.TotalBHD, currency)
	return nil
}

// SendPurchaseOrder sends an approved PO to supplier (blocks if not approved)
func (a *App) SendPurchaseOrder(id string) error {
	if err := a.requirePermission("po:send"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get PO
	var po PurchaseOrder
	if err := a.db.First(&po, "id = ?", id).Error; err != nil {
		return newError("PO_NOT_FOUND", "Purchase order not found", err.Error())
	}

	// Status validation: only Draft or Approved POs can be sent
	validSendStatuses := map[string]bool{"Draft": true, "Approved": true}
	if !validSendStatuses[po.Status] {
		return newError("PO_INVALID_STATUS",
			fmt.Sprintf("Cannot send PO %s: status is '%s' (only Draft or Approved POs can be sent)",
				po.PONumber, po.Status), "")
	}

	// P1 FIX: Block sending unapproved POs above threshold
	// P3 FIX (PO-26): Default currency fallback for error messages
	currency := po.Currency
	if currency == "" {
		currency = "BHD"
	}
	approvalThreshold := 5000.0 // BHD
	if po.TotalBHD > approvalThreshold && po.Status != "Approved" {
		return newError("APPROVAL_REQUIRED",
			fmt.Sprintf("Cannot send PO %s (%.2f %s) - requires manager approval (threshold: %.2f %s)",
				po.PONumber, po.TotalBHD, currency, approvalThreshold, currency), "")
	}

	// Update status to Sent
	if err := a.db.Model(&po).Update("status", "Sent").Error; err != nil {
		return newError("DB_UPDATE_FAILED", "Failed to send purchase order", err.Error())
	}

	log.Printf("📧 Sent Purchase Order %s to Supplier %s (%.2f %s)", po.PONumber, po.SupplierID, po.TotalBHD, currency)
	return nil
}

// DeletePurchaseOrder soft-deletes a purchase order
func (a *App) DeletePurchaseOrder(id string) error {
	if ok, err := a.guardDeleteOrRequest("po:delete", "purchase_order", id, "Purchase order"); !ok {
		return err
	}
	if err := a.requirePermission("po:delete"); err != nil {
		return err
	}
	return crmprocurement.DeletePurchaseOrder(a.db, id)
}

// =============================================================================
// PURCHASE ORDER UTILITIES
// =============================================================================

// GeneratePONumber generates a new PO number in format PO-YYYY-NNNN
// Uses InvoiceSequence table with row locking (same pattern as GenerateInvoiceNumber/GenerateCreditNoteNumber)
func (a *App) GeneratePONumber() (string, error) {
	// Wave 8 P0: gate the sequence read for parity. Internal callers (CreatePurchaseOrder)
	// already hold po:create, so this re-check is a no-op for legitimate flows.
	if err := a.requirePermission("po:create"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Delegates to the promoted pkg/documents/numbering engine (Wave 2
	// Mission A). Format and first-of-year seeding are byte-identical to the
	// old inline implementation.
	poNumber, err := numbering.New(a.db).Next(numbering.Spec{
		Prefix:   "PO",
		Template: "PO-{year}-{seq}",
		Seed: func(tx *gorm.DB, year int) (int64, error) {
			// First PO of this year — initialize sequence from existing POs
			var maxExisting int64
			tx.Model(&PurchaseOrder{}).
				Where("po_number LIKE ? ESCAPE '\\'", fmt.Sprintf("PO-%d-%%", year)).
				Count(&maxExisting)
			return maxExisting, nil
		},
	}, time.Now())

	if err != nil {
		return "", fmt.Errorf("PO number generation failed: %w", err)
	}

	log.Printf("Generated PO Number: %s", poNumber)
	return poNumber, nil
}

// CreatePOFromOrder creates a purchase order from a customer order for a specific supplier
// Splits order items by supplier and creates a PO with the matching items
func (a *App) CreatePOFromOrder(orderID string, supplierID string, itemIDs []string) (PurchaseOrder, error) {
	return a.procurementService().CreatePOFromOrder(orderID, supplierID, itemIDs)
}

func createPOFromOrder(a *App, orderID string, supplierID string, itemIDs []string) (PurchaseOrder, error) {
	if err := a.requirePermission("po:create"); err != nil {
		return PurchaseOrder{}, err
	}
	if a.db == nil {
		return PurchaseOrder{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Load the order with items
	var order Order
	if err := a.db.Preload("Items").First(&order, "id = ?", orderID).Error; err != nil {
		return PurchaseOrder{}, newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	// Load supplier to get default currency and payment terms
	var supplier SupplierMaster
	if strings.TrimSpace(supplierID) == "" {
		inferredSupplier, err := a.inferSupplierForOrderItems(order, itemIDs)
		if err != nil {
			return PurchaseOrder{}, err
		}
		supplier = inferredSupplier
		supplierID = inferredSupplier.ID
	} else if err := a.db.First(&supplier, "id = ?", supplierID).Error; err != nil {
		return PurchaseOrder{}, newError("SUPPLIER_NOT_FOUND", "Supplier not found", err.Error())
	}

	// Create PO header
	po := PurchaseOrder{
		OrderID:          orderID,
		SupplierID:       supplierID,
		SupplierName:     supplier.SupplierName,
		PODate:           time.Now(),
		ExpectedDelivery: order.RequiredDate, // Use order's required date as expected delivery
		Currency:         "BHD",              // Default to BHD, can be changed based on supplier
		ExchangeRate:     1.0,                // 1:1 for BHD
		PaymentTerms:     "Net 30",           // Default, override from supplier if available
		Status:           "Draft",
		Division:         normalizeDivisionName(order.Division),
		Items:            []PurchaseOrderItem{},
	}

	// If supplier has default currency, use it
	// Note: SupplierMaster needs DefaultCurrency field added in future
	// For now, we'll assume BHD

	// Add items
	var subtotal float64
	for _, orderItem := range order.Items {
		// Skip zero or negative quantity items
		if orderItem.Quantity <= 0 {
			log.Printf("⚠️ Skipping item %s with invalid quantity %.0f", orderItem.ProductCode, orderItem.Quantity)
			continue
		}

		// Check if this item should be included
		includeItem := len(itemIDs) == 0 // If no specific items, include all
		if !includeItem {
			for _, itemID := range itemIDs {
				if orderItem.ID == itemID {
					includeItem = true
					break
				}
			}
		}

		if !includeItem {
			continue
		}

		// Check if product belongs to this supplier
		var product ProductMaster
		if err := a.db.First(&product, "id = ?", orderItem.ProductID).Error; err == nil {
			if product.SupplierID != supplierID {
				log.Printf("⚠️ Skipping item %s - belongs to different supplier", orderItem.ProductCode)
				continue
			}
		}

		// Determine purchase price using best available source:
		// 1. OrderItem.FOB (from costing sheet via offer — actual supplier cost)
		// 2. ProductMaster.StandardCostBHD (catalog cost)
		// 3. Fallback: 70% of sell price (rough estimate)
		unitForeign := orderItem.UnitPrice * 0.7 // Fallback estimate
		unitBHD := orderItem.UnitPrice * 0.7

		if orderItem.FOB > 0 {
			// Best source: actual FOB from costing sheet (supplier's price in their currency)
			unitForeign = orderItem.FOB
			if orderItem.TotalCost > 0 && orderItem.Quantity > 0 {
				// Use calculated landed cost from costing sheet (already in BHD)
				unitBHD = orderItem.TotalCost / orderItem.Quantity
			} else if orderItem.Currency == "BHD" || orderItem.Currency == "" {
				// FOB is already in BHD
				unitBHD = orderItem.FOB
			} else {
				// Foreign currency FOB without TotalCost — fall back to 70% estimate
				// rather than using unconverted foreign value as BHD
				unitBHD = orderItem.UnitPrice * 0.7
				log.Printf("⚠️ PO item %s: FOB in %s but no TotalCost — using 70%% estimate for BHD", orderItem.ProductCode, orderItem.Currency)
			}
			log.Printf("📊 PO item %s: using costing sheet FOB (%.3f %s, %.3f BHD)", orderItem.ProductCode, unitForeign, orderItem.Currency, unitBHD)
		} else if err := a.db.First(&product, "id = ?", orderItem.ProductID).Error; err == nil && product.StandardCostBHD > 0 {
			// Second source: product master standard cost
			unitForeign = product.StandardCostBHD
			unitBHD = product.StandardCostBHD
			log.Printf("📊 PO item %s: using ProductMaster StandardCost (%.3f BHD)", orderItem.ProductCode, unitBHD)
		} else {
			log.Printf("⚠️ PO item %s: no costing data — using 70%% sell price estimate (%.3f BHD)", orderItem.ProductCode, unitBHD)
		}

		poItem := PurchaseOrderItem{
			OrderItemID:      orderItem.ID,
			ProductID:        orderItem.ProductID,
			ProductCode:      orderItem.ProductCode,
			Description:      orderItem.Description,
			Quantity:         orderItem.Quantity,
			UnitPriceForeign: unitForeign,
			UnitPriceBHD:     unitBHD,
			TotalForeign:     orderItem.Quantity * unitForeign,
			TotalBHD:         orderItem.Quantity * unitBHD,
			QuantityReceived: 0,
		}

		po.Items = append(po.Items, poItem)
		subtotal += poItem.TotalBHD
	}

	if len(po.Items) == 0 {
		return PurchaseOrder{}, newError("NO_ITEMS", "No items found for this supplier in the order", "")
	}

	// Calculate totals (assuming 10% VAT)
	po.SubtotalForeign = subtotal
	po.SubtotalBHD = subtotal
	po.VATAmount = subtotal * 0.10
	po.TotalForeign = subtotal + po.VATAmount
	po.TotalBHD = subtotal + po.VATAmount

	// P1 FIX: Set CreatedBy for segregation-of-duties check in ApprovePurchaseOrder
	po.CreatedBy = a.getCurrentUserID()

	// Create the PO
	createdPO, err := a.CreatePurchaseOrder(po)
	if err != nil {
		return PurchaseOrder{}, err
	}

	log.Printf("✅ Created PO %s from Order %s for Supplier %s (%d items, %.2f BHD)",
		createdPO.PONumber, order.OrderNumber, supplierID, len(createdPO.Items), createdPO.TotalBHD)

	return createdPO, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// calculatePaymentDueDate calculates payment due date from PO date and payment terms
func calculatePaymentDueDate(poDate time.Time, paymentTerms string) time.Time {
	// Parse payment terms (e.g., "Net 30", "Net 45", "Net 60")
	var days int

	terms := strings.ToUpper(paymentTerms)

	if strings.Contains(terms, "NET") {
		// Extract number from "Net 30", "Net 45", etc.
		fmt.Sscanf(terms, "NET %d", &days)
	} else if strings.Contains(terms, "ADVANCE") || strings.Contains(terms, "PREPAYMENT") {
		// Advance payment - due immediately
		days = 0
	} else if strings.Contains(terms, "COD") {
		// Cash on delivery - same day
		days = 0
	} else {
		// Default to Net 30
		days = 30
	}

	return poDate.AddDate(0, 0, days)
}

// =============================================================================
// P2 FIX: PO AMENDMENT TRACKING
// =============================================================================

// POAmendment tracks changes to purchase orders after creation
type POAmendment struct {
	ID                 string     `json:"id"`
	PurchaseOrderID    string     `json:"purchase_order_id"`
	AmendmentNumber    int        `json:"amendment_number"`
	AmendedBy          string     `json:"amended_by"`
	AmendedAt          time.Time  `json:"amended_at"`
	ChangeType         string     `json:"change_type"` // "quantity", "price", "date", "items", "terms"
	OldValue           string     `json:"old_value"`   // JSON of old values
	NewValue           string     `json:"new_value"`   // JSON of new values
	Reason             string     `json:"reason"`
	RequiresReapproval bool       `json:"requires_reapproval"`
	ReapprovedBy       string     `json:"reapproved_by"`
	ReapprovedAt       *time.Time `json:"reapproved_at"`
}

// AmendPurchaseOrder creates an amendment record and optionally requires re-approval
func (a *App) AmendPurchaseOrder(poID string, amendments POAmendment) error {
	if err := a.requirePermission("po:update"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get existing PO
	var po PurchaseOrder
	if err := a.db.Preload("Items").First(&po, "id = ?", poID).Error; err != nil {
		return newError("PO_NOT_FOUND", "Purchase order not found", err.Error())
	}

	// P2 FIX: Only Draft and Pending Approval POs can be amended
	amendableStatuses := map[string]bool{"Draft": true, "Pending Approval": true}
	if !amendableStatuses[po.Status] {
		return newError("PO_INVALID_STATUS",
			fmt.Sprintf("Cannot amend PO %s: status is '%s' (only Draft or Pending Approval POs can be amended)", po.PONumber, po.Status), "")
	}

	// Generate amendment number
	var amendmentCount int64
	a.db.Raw("SELECT COUNT(*) FROM po_amendments WHERE purchase_order_id = ?", poID).Scan(&amendmentCount)
	amendments.AmendmentNumber = int(amendmentCount) + 1
	amendments.PurchaseOrderID = poID
	amendments.AmendedAt = time.Now()

	// Create amendment record (store in a dedicated table via raw SQL for now)
	// In production, add POAmendment to database.go models
	log.Printf("📝 PO Amendment #%d created for %s by %s: %s (Reason: %s)",
		amendments.AmendmentNumber, po.PONumber, amendments.AmendedBy, amendments.ChangeType, amendments.Reason)

	// NOTE: Amendment persistence deferred — po_amendments table not yet created
	// When implemented, persist the amendment record and evaluate re-approval based on value change

	return nil
}

// GetPOAmendmentHistory retrieves all amendments for a purchase order
func (a *App) GetPOAmendmentHistory(poID string) ([]POAmendment, error) {
	if err := a.requirePermission("po:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Placeholder - in production, query from po_amendments table
	// For now, return empty slice
	log.Printf("📋 Retrieving amendment history for PO %s", poID)
	return []POAmendment{}, nil
}
