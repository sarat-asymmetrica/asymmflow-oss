package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	crmprocurement "ph_holdings_app/pkg/crm/procurement"
	"ph_holdings_app/pkg/documents/numbering"
	"ph_holdings_app/pkg/inventory"
)

// =============================================================================
// GRN DISPLAY TYPES (FRONTEND-FRIENDLY)
// =============================================================================

// GRNResponse adds computed fields for frontend display
type GRNResponse struct {
	GoodsReceivedNote
	SupplierName   string  `json:"supplier_name"`
	PONumber       string  `json:"po_number"`
	ItemsCount     int     `json:"items_count"`
	TotalReceived  float64 `json:"total_received"`
	TotalAccepted  float64 `json:"total_accepted"`
	TotalRejected  float64 `json:"total_rejected"`
	AcceptanceRate float64 `json:"acceptance_rate"`
	// B3: true "already completed" signal for the frontend Complete-button
	// gate, derived from GoodsReceivedNote.CompletedAt (see that field's
	// doc comment). Superseded the B6 movement-ledger derivation below,
	// which had a blind spot for all-rejected GRNs.
	IsCompleted bool `json:"is_completed"`
}

// grnHasPostedMovement reports whether GRN quantities/inventory were already
// posted for grnID — a persisted, authoritative "GRN Receipt" stock movement
// referencing this GRN (reconcileInventoryReceipt in
// procurement_inventory_policy.go writes exactly one such row per accepted
// line, inside the same transaction as the PO quantity update in
// CompleteGRN). Still used as the belt-and-suspenders half of CompleteGRN's
// idempotency guard (B6), alongside the B3 CompletedAt flag which is now the
// source of truth for GRNResponse.IsCompleted.
//
// QCStatus is NOT a safe substitute for either signal: the intended workflow
// requires QC to be resolved (Passed/Partial) BEFORE Complete is even offered
// (GRNScreen.svelte gating), so "QCStatus != Pending" is the ORDINARY
// pre-completion state, not evidence of a prior posting.
//
// Known residual gap (closed by B3's CompletedAt flag, kept here as
// historical context): a GRN whose lines were ALL rejected (zero accepted
// quantity on every item) never creates a stock movement even on a
// legitimate first completion, so this signal alone cannot distinguish
// "never completed" from "completed with nothing accepted" for that edge
// case.
func grnHasPostedMovement(db *gorm.DB, grnID string) (bool, error) {
	var count int64
	if err := db.Model(&StockMovement{}).
		Where("reference_type = ? AND reference_id = ?", "goods_received_note", grnID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// =============================================================================
// GRN CRUD OPERATIONS
// =============================================================================

// CreateGRN creates a new Goods Received Note
func (a *App) CreateGRN(grn GoodsReceivedNote) (GoodsReceivedNote, error) {
	return a.procurementService().CreateGRN(grn)
}

func createGRN(a *App, grn GoodsReceivedNote) (GoodsReceivedNote, error) {
	if err := a.requirePermission("grn:create"); err != nil {
		return GoodsReceivedNote{}, err
	}
	if a.db == nil {
		return GoodsReceivedNote{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate PO exists
	var po PurchaseOrder
	if err := a.db.First(&po, "id = ?", grn.PurchaseOrderID).Error; err != nil {
		return GoodsReceivedNote{}, newError("PO_NOT_FOUND", "Purchase Order not found", err.Error())
	}

	// Validate PO status - cannot create GRN against Draft or Cancelled POs
	if po.Status == "Cancelled" || po.Status == "Draft" {
		return GoodsReceivedNote{}, newError("INVALID_PO_STATUS", fmt.Sprintf("cannot create GRN against %s purchase order", po.Status), "")
	}

	// Generate GRN number if empty
	if grn.GRNNumber == "" {
		var err error
		grn.GRNNumber, err = a.GenerateGRNNumber()
		if err != nil {
			return GoodsReceivedNote{}, err
		}
	}

	// Set defaults
	if grn.ReceivedDate.IsZero() {
		grn.ReceivedDate = time.Now()
	}
	if grn.QCStatus == "" {
		grn.QCStatus = "Pending"
	}

	// Wave 9.3 B2: identity server-resolved (Article III.4) — ignore any
	// client-supplied ReceivedBy and stamp the authenticated operator instead.
	grn.ReceivedBy = a.getCurrentUserID()

	if err := a.db.Create(&grn).Error; err != nil {
		log.Printf("❌ Failed to create GRN: %v", err)
		return GoodsReceivedNote{}, newError("DB_CREATE_FAILED", "Failed to create GRN", err.Error())
	}

	log.Printf("✅ Created GRN %s for PO %s", grn.GRNNumber, po.PONumber)
	return grn, nil
}

// GetGRNs retrieves all GRNs with optional pagination (basic version)
func (a *App) GetGRNs() ([]GoodsReceivedNote, error) {
	if err := a.requirePermission("grn:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var grns []GoodsReceivedNote
	if err := a.db.Preload("Items").Order("received_date DESC").Find(&grns).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to list GRNs", err.Error())
	}

	log.Printf("Retrieved %d GRNs", len(grns))
	return grns, nil
}

// ListGRNs retrieves all GRNs with computed fields for frontend (PRODUCTION API)
func (a *App) ListGRNs(limit int, offset int, qcStatus string) ([]GRNResponse, error) {
	if err := a.requirePermission("grn:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Default limit: 100, max: 1000
	if limit <= 0 {
		limit = 100
	} else if limit > 1000 {
		limit = 1000
	}

	if offset < 0 {
		offset = 0
	}

	var grns []GoodsReceivedNote
	query := a.db.Preload("Items").Order("received_date DESC")

	// Filter by QC status if provided
	if qcStatus != "" && qcStatus != "All" {
		query = query.Where("qc_status = ?", qcStatus)
	}

	query = query.Limit(limit).Offset(offset)
	if err := query.Find(&grns).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to list GRNs", err.Error())
	}

	// B3: completion is now read directly off each row's persisted flag —
	// the authoritative signal, including for all-rejected GRNs that never
	// post a StockMovement — so no separate batch lookup is needed.

	// Enrich with computed fields
	responses := make([]GRNResponse, len(grns))
	for i, grn := range grns {
		resp := GRNResponse{
			GoodsReceivedNote: grn,
			ItemsCount:        len(grn.Items),
			IsCompleted:       grn.CompletedAt != nil,
		}

		// Get PO info and supplier name
		var po PurchaseOrder
		if err := a.db.First(&po, "id = ?", grn.PurchaseOrderID).Error; err == nil {
			resp.PONumber = po.PONumber

			// Get supplier info
			var supplier SupplierMaster
			if err := a.db.First(&supplier, "id = ?", po.SupplierID).Error; err == nil {
				resp.SupplierName = supplier.SupplierName
			}
		}

		// Calculate totals
		for _, item := range grn.Items {
			resp.TotalReceived += item.QuantityReceived
			acceptedQty := item.QuantityReceived - item.QuantityRejected
			resp.TotalAccepted += acceptedQty
			resp.TotalRejected += item.QuantityRejected
		}

		// Calculate acceptance rate
		if resp.TotalReceived > 0 {
			resp.AcceptanceRate = resp.TotalAccepted / resp.TotalReceived
		}

		responses[i] = resp
	}

	log.Printf("✅ Retrieved %d GRNs (limit=%d, offset=%d, qcStatus=%s)", len(responses), limit, offset, qcStatus)
	return responses, nil
}

// GetGRN retrieves a single GRN by ID with full details (PRODUCTION API)
func (a *App) GetGRN(grnID string) (*GRNResponse, error) {
	if err := a.requirePermission("grn:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var grn GoodsReceivedNote
	if err := a.db.Preload("Items").First(&grn, "id = ?", grnID).Error; err != nil {
		return nil, newError("GRN_NOT_FOUND", "GRN not found", err.Error())
	}

	// B3: derive from the dedicated completion flag — the authoritative
	// signal, including for all-rejected GRNs that never post a StockMovement.
	resp := &GRNResponse{
		GoodsReceivedNote: grn,
		ItemsCount:        len(grn.Items),
		IsCompleted:       grn.CompletedAt != nil,
	}

	// Get PO info and supplier name
	var po PurchaseOrder
	if err := a.db.First(&po, "id = ?", grn.PurchaseOrderID).Error; err == nil {
		resp.PONumber = po.PONumber

		// Get supplier info
		var supplier SupplierMaster
		if err := a.db.First(&supplier, "id = ?", po.SupplierID).Error; err == nil {
			resp.SupplierName = supplier.SupplierName
		}
	}

	// Calculate totals
	for _, item := range grn.Items {
		resp.TotalReceived += item.QuantityReceived
		acceptedQty := item.QuantityReceived - item.QuantityRejected
		resp.TotalAccepted += acceptedQty
		resp.TotalRejected += item.QuantityRejected
	}

	if resp.TotalReceived > 0 {
		resp.AcceptanceRate = resp.TotalAccepted / resp.TotalReceived
	}

	log.Printf("✅ Retrieved GRN #%s (%d items)", grnID, resp.ItemsCount)
	return resp, nil
}

// GetGRNByID retrieves a single GRN by ID with items preloaded
func (a *App) GetGRNByID(id string) (GoodsReceivedNote, error) {
	if err := a.requirePermission("grn:view"); err != nil {
		return GoodsReceivedNote{}, err
	}
	if a.db == nil {
		return GoodsReceivedNote{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var grn GoodsReceivedNote
	if err := a.db.Preload("Items").First(&grn, "id = ?", id).Error; err != nil {
		return GoodsReceivedNote{}, newError("GRN_NOT_FOUND", "GRN not found", err.Error())
	}

	return grn, nil
}

// GetGRNsByPO retrieves all GRNs for a specific Purchase Order
func (a *App) GetGRNsByPO(purchaseOrderID string) ([]GoodsReceivedNote, error) {
	if err := a.requirePermission("grn:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var grns []GoodsReceivedNote
	if err := a.db.Preload("Items").Where("purchase_order_id = ?", purchaseOrderID).
		Order("received_date DESC").Find(&grns).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to get GRNs for PO", err.Error())
	}

	log.Printf("Retrieved %d GRNs for PO %s", len(grns), purchaseOrderID)
	return grns, nil
}

// UpdateGRN updates an existing GRN
func (a *App) UpdateGRN(grn GoodsReceivedNote) (GoodsReceivedNote, error) {
	if err := a.requirePermission("grn:update"); err != nil {
		return GoodsReceivedNote{}, err
	}
	if a.db == nil {
		return GoodsReceivedNote{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Verify GRN exists
	var existing GoodsReceivedNote
	if err := a.db.First(&existing, "id = ?", grn.ID).Error; err != nil {
		return GoodsReceivedNote{}, newError("GRN_NOT_FOUND", "GRN not found", err.Error())
	}

	// Allow GRN number edit only if QC is still Pending (not yet completed)
	if grn.GRNNumber != "" && grn.GRNNumber != existing.GRNNumber {
		if existing.QCStatus != "Pending" {
			return GoodsReceivedNote{}, newError("GRN_NUMBER_LOCKED",
				fmt.Sprintf("Cannot change GRN number: QC status is %s (only Pending GRNs can have number changed)", existing.QCStatus), "")
		}
		// Uniqueness check: verify no other GRN has the same number
		var count int64
		a.db.Model(&GoodsReceivedNote{}).Where("grn_number = ? AND id != ?", grn.GRNNumber, grn.ID).Count(&count)
		if count > 0 {
			return GoodsReceivedNote{}, newError("GRN_NUMBER_EXISTS",
				fmt.Sprintf("GRN number %s already exists", grn.GRNNumber), "")
		}
	}

	// INT-001: field-mask — QC fields are owned by the dedicated QC workflow
	// (grn:qc / grn:complete), not this header edit. Restore them and the audit
	// metadata from the loaded row so a partial payload can't wipe QC
	// status/notes/date/inspector.
	grn.QCStatus = existing.QCStatus
	grn.QCNotes = existing.QCNotes
	grn.QCDate = existing.QCDate
	grn.QCBy = existing.QCBy
	grn.CreatedAt = existing.CreatedAt
	grn.CreatedBy = existing.CreatedBy
	grn.DeletedAt = existing.DeletedAt
	grn.Version = existing.Version
	if grn.UpdatedBy == "" {
		grn.UpdatedBy = existing.UpdatedBy
	}
	// Mission I (I-12): Save() writes every column — a partial payload that
	// omitted the PO/warehouse linkage or receiving metadata wiped it, breaking
	// 3-way-match traceability. Restore from the loaded row when unset.
	if grn.PurchaseOrderID == "" {
		grn.PurchaseOrderID = existing.PurchaseOrderID
	}
	if grn.WarehouseID == "" {
		grn.WarehouseID = existing.WarehouseID
	}
	if grn.SupplierDNNumber == "" {
		grn.SupplierDNNumber = existing.SupplierDNNumber
	}
	if grn.ReceivedBy == "" {
		grn.ReceivedBy = existing.ReceivedBy
	}
	if grn.ReceivedDate.IsZero() {
		grn.ReceivedDate = existing.ReceivedDate
	}

	if err := a.db.Save(&grn).Error; err != nil {
		log.Printf("❌ Failed to update GRN: %v", err)
		return GoodsReceivedNote{}, newError("DB_UPDATE_FAILED", "Failed to update GRN", err.Error())
	}

	log.Printf("✅ Updated GRN %s", grn.GRNNumber)
	return grn, nil
}

// DeleteGRN deletes a GRN by ID
func (a *App) DeleteGRN(id string) error {
	if ok, err := a.guardDeleteOrRequest("grn:delete", "grn", id, "GRN"); !ok {
		return err
	}
	if err := a.requirePermission("grn:delete"); err != nil {
		return err
	}
	return crmprocurement.DeleteGRN(a.db, id)
}

// =============================================================================
// GRN NUMBER GENERATION
// =============================================================================

// GenerateGRNNumber generates a unique GRN number in format GRN-2026-0001
// W4 A.2: GRN was the last document type still allocating via raw
// BEGIN EXCLUSIVE + max-scan (the pattern the S4 fixes replaced everywhere
// else). It now delegates to pkg/documents/numbering like INV/CN/PO/DN.
// Format is unchanged; the first allocation of a year continues from the
// highest existing GRN-{year}-NNNN (max-parse, not COUNT, so deleted GRNs
// can never cause a number to be reissued).
func (a *App) GenerateGRNNumber() (string, error) {
	// Wave 8 P0: gate the sequence read for parity. Internal callers (CreateGRN,
	// ReceiveAgainstPO) already hold grn:create, so this re-check is a no-op for them.
	if err := a.requirePermission("grn:create"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	grnNumber, err := numbering.New(a.db).Next(numbering.Spec{
		Prefix:   "GRN",
		Template: "GRN-{year}-{seq}",
		Seed: func(tx *gorm.DB, year int) (int64, error) {
			prefix := fmt.Sprintf("GRN-%d-", year)
			var lastGRN GoodsReceivedNote
			err := tx.Where("grn_number LIKE ?", prefix+"%").
				Order("grn_number DESC").
				First(&lastGRN).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, nil
			}
			if err != nil {
				return 0, err
			}
			var lastNum int64
			if _, parseErr := fmt.Sscanf(lastGRN.GRNNumber, prefix+"%d", &lastNum); parseErr != nil {
				return 0, newError("NUMBER_PARSE_FAILED", "Failed to parse last GRN number", parseErr.Error())
			}
			return lastNum, nil
		},
	}, time.Now())
	if err != nil {
		return "", err
	}

	log.Printf("Generated GRN number: %s", grnNumber)
	return grnNumber, nil
}

// =============================================================================
// GRN COMPLETION & STATUS UPDATES
// =============================================================================

// CompleteGRN marks a GRN as complete and updates PO quantities
func (a *App) CompleteGRN(id string) error {
	if err := a.requirePermission("grn:complete"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get GRN with items
	var grn GoodsReceivedNote
	if err := a.db.Preload("Items").First(&grn, "id = ?", id).Error; err != nil {
		return newError("GRN_NOT_FOUND", "GRN not found", err.Error())
	}

	// P1 FIX: Block completing GRN with QC failures
	if grn.QCStatus == "Failed" {
		return newError("QC_FAILED",
			fmt.Sprintf("Cannot complete GRN %s - Quality Control failed. Resolve QC issues first.", grn.GRNNumber),
			grn.QCNotes)
	}

	// Begin transaction
	tx := a.db.Begin()
	if tx.Error != nil {
		return newError("DB_TX_FAILED", "Failed to begin GRN completion transaction", tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// B6 idempotency fix: row-lock the GRN inside the transaction so two
	// concurrent Complete calls for the same GRN can't both pass the
	// already-applied check together, then re-verify the true completion
	// signal under that lock. See grnHasPostedMovement for why QCStatus
	// (the old guard's `{Approved,Rejected,Completed,Passed}` allowlist,
	// which missed "Partial" — the reported double-count leak) is not a
	// safe substitute for it.
	var lockedGRN GoodsReceivedNote
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedGRN, "id = ?", grn.ID).Error; err != nil {
		tx.Rollback()
		return newError("GRN_NOT_FOUND", "GRN not found", err.Error())
	}

	alreadyApplied, err := grnHasPostedMovement(tx, grn.ID)
	if err != nil {
		tx.Rollback()
		return newError("DB_QUERY_FAILED", "Failed to verify GRN completion state", err.Error())
	}
	// B3: also treat a previously-set completion flag as "already applied".
	// This closes the all-rejected edge case: such a GRN posts no
	// StockMovement even on a legitimate first completion (see
	// grnHasPostedMovement), so grnHasPostedMovement alone would let a
	// second CompleteGRN call re-run the PO-quantity update. The flag,
	// set below under this same row lock, is the belt to that guard's
	// suspenders.
	if alreadyApplied || lockedGRN.CompletedAt != nil {
		tx.Rollback()
		log.Printf("⚠️ GRN %s already applied (stock movement or completion flag already on record), skipping PO quantity update to prevent double-counting", grn.GRNNumber)
		return nil
	}

	// Update PO item quantities received
	for _, item := range grn.Items {
		if item.POItemID == "" {
			continue
		}

		// Get current PO item
		var poItem PurchaseOrderItem
		if err := tx.First(&poItem, "id = ?", item.POItemID).Error; err != nil {
			tx.Rollback()
			return newError("PO_ITEM_NOT_FOUND", "PO item not found", err.Error())
		}

		// Update quantity received
		newQuantityReceived := poItem.QuantityReceived + item.QuantityReceived
		if err := tx.Model(&poItem).Update("quantity_received", newQuantityReceived).Error; err != nil {
			tx.Rollback()
			return newError("DB_UPDATE_FAILED", "Failed to update PO item quantity", err.Error())
		}

		// Band-2 (PH 5937958): accepted goods become stock — inventory item
		// upsert + valued GRN Receipt movement, inside this same transaction.
		if err := a.reconcileInventoryReceipt(tx, grn, poItem, item); err != nil {
			tx.Rollback()
			return newError("DB_UPDATE_FAILED", "Failed to reconcile inventory receipt", err.Error())
		}

		log.Printf("Updated PO item %s: received %.2f (total now: %.2f/%.2f)",
			item.POItemID, item.QuantityReceived, newQuantityReceived, poItem.Quantity)
	}

	// Update PO status based on completion
	if err := a.updatePOStatus(tx, grn.PurchaseOrderID); err != nil {
		tx.Rollback()
		return err
	}

	// P1 FIX: Mark GRN QC as Passed on completion (if still Pending)
	if grn.QCStatus == "Pending" {
		if err := tx.Model(&grn).Update("qc_status", "Passed").Error; err != nil {
			tx.Rollback()
			return newError("DB_UPDATE_FAILED", "Failed to update GRN QC status", err.Error())
		}
	}

	// B3: stamp the dedicated completion flag under the same row lock/
	// transaction as the PO-quantity update above, so it's the authoritative
	// "already completed" signal even for all-rejected GRNs that post no
	// StockMovement.
	completedAt := time.Now()
	if err := tx.Model(&GoodsReceivedNote{}).Where("id = ?", grn.ID).Update("completed_at", completedAt).Error; err != nil {
		tx.Rollback()
		return newError("DB_UPDATE_FAILED", "Failed to mark GRN completed", err.Error())
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return newError("DB_COMMIT_FAILED", "Failed to commit GRN completion", err.Error())
	}

	log.Printf("✅ Completed GRN %s and updated PO quantities", grn.GRNNumber)
	return nil
}

// updatePOStatus updates PO status based on received quantities
func (a *App) updatePOStatus(tx *gorm.DB, poID string) error {
	// Get PO with items
	var po PurchaseOrder
	if err := tx.Preload("Items").First(&po, "id = ?", poID).Error; err != nil {
		return newError("PO_NOT_FOUND", "Purchase Order not found", err.Error())
	}

	// Check if all items are fully received
	fullyReceived := true
	partiallyReceived := false

	for _, item := range po.Items {
		// P0-3 Fix: Add over-receipt detection
		if item.QuantityReceived > item.Quantity {
			log.Printf("⚠️ WARNING: Over-receipt detected for PO %s item %s: received %.2f > ordered %.2f",
				po.PONumber, item.ProductCode, item.QuantityReceived, item.Quantity)
			// Don't block - log warning for audit trail
		}

		if item.QuantityReceived < item.Quantity {
			fullyReceived = false
		}
		if item.QuantityReceived > 0 {
			partiallyReceived = true
		}
	}

	// Update PO status
	var newStatus string
	if fullyReceived && len(po.Items) > 0 {
		newStatus = "Received"
	} else if partiallyReceived {
		newStatus = "Partially Received"
	} else {
		newStatus = po.Status // Keep existing status
	}

	if newStatus != po.Status {
		if err := tx.Model(&po).Update("status", newStatus).Error; err != nil {
			return newError("DB_UPDATE_FAILED", "Failed to update PO status", err.Error())
		}
		log.Printf("Updated PO %s status: %s → %s", po.PONumber, po.Status, newStatus)
	}

	return nil
}

// UpdateGRNQCStatus updates the QC status of a GRN
func (a *App) UpdateGRNQCStatus(id string, status string, notes string, qcBy string) error {
	if err := a.requirePermission("grn:qc"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Verify GRN exists
	var grn GoodsReceivedNote
	if err := a.db.First(&grn, "id = ?", id).Error; err != nil {
		return newError("GRN_NOT_FOUND", "GRN not found", err.Error())
	}

	// Wave 9.3 B2: identity server-resolved (Article III.4) — ignore the
	// client-supplied qcBy (kept as a param for binding stability) and stamp
	// the authenticated operator instead.
	qcBy = a.getCurrentUserID()

	// Update QC fields
	now := time.Now()
	updates := map[string]any{
		"qc_status": status,
		"qc_notes":  notes,
		"qc_by":     qcBy,
		"qc_date":   now,
	}

	if err := a.db.Model(&grn).Updates(updates).Error; err != nil {
		log.Printf("❌ Failed to update GRN QC status: %v", err)
		return newError("DB_UPDATE_FAILED", "Failed to update QC status", err.Error())
	}

	log.Printf("✅ Updated GRN %s QC status to: %s (by %s)", grn.GRNNumber, status, qcBy)
	return nil
}

// =============================================================================
// RECEIVE AGAINST PO - AUTO-CREATE GRN
// =============================================================================

// ReceiveAgainstPO creates a GRN from a PO with specified items
func (a *App) ReceiveAgainstPO(poID string, items []GRNItem) (GoodsReceivedNote, error) {
	return a.procurementService().ReceiveAgainstPO(poID, items)
}

// rejectSerializedItemsWithoutSerials is Wave 9.8 B1 defense-in-depth for the
// plain (non-serial) receive path: GRNItem carries no SerialNumbers field, so
// there is no way for this path to capture per-unit serials at all. If any
// item's product requires serial tracking, refuse the whole receive rather
// than silently posting it unserialized — the caller must route serialized
// lines through ReceiveAgainstPOWithSerials / ReceiveAndCompletePOWithSerials
// instead, where the len(serials)==quantity check is the enforcement of
// record. Fails open (returns nil) on a lookup error so a transient DB issue
// here doesn't block an otherwise-valid non-serialized receive.
func (a *App) rejectSerializedItemsWithoutSerials(items []GRNItem) error {
	productIDs := make([]string, 0, len(items))
	seen := make(map[string]bool)
	for _, item := range items {
		if item.ProductID != "" && !seen[item.ProductID] {
			seen[item.ProductID] = true
			productIDs = append(productIDs, item.ProductID)
		}
	}
	if len(productIDs) == 0 {
		return nil
	}

	var products []ProductMaster
	if err := a.db.Where("id IN ?", productIDs).Find(&products).Error; err != nil {
		return nil
	}

	serialRequired := make(map[string]bool, len(products))
	productCode := make(map[string]string, len(products))
	for _, p := range products {
		serialRequired[p.ID] = p.RequiresSerialTracking
		productCode[p.ID] = p.ProductCode
	}

	for _, item := range items {
		if serialRequired[item.ProductID] {
			return newError("SERIAL_REQUIRED",
				fmt.Sprintf("Product %s requires serial tracking — use the serialized receive flow and supply one serial number per unit", productCode[item.ProductID]), "")
		}
	}
	return nil
}

func receiveAgainstPO(a *App, poID string, items []GRNItem) (GoodsReceivedNote, error) {
	if err := a.requirePermission("grn:create"); err != nil {
		return GoodsReceivedNote{}, err
	}
	if a.db == nil {
		return GoodsReceivedNote{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Verify PO exists
	var po PurchaseOrder
	if err := a.db.Preload("Items").First(&po, "id = ?", poID).Error; err != nil {
		return GoodsReceivedNote{}, newError("PO_NOT_FOUND", "Purchase Order not found", err.Error())
	}

	// Generate GRN number
	grnNumber, err := a.GenerateGRNNumber()
	if err != nil {
		return GoodsReceivedNote{}, err
	}

	// Create GRN
	grn := GoodsReceivedNote{
		PurchaseOrderID: poID,
		GRNNumber:       grnNumber,
		ReceivedDate:    time.Now(),
		ReceivedBy:      a.getCurrentUserID(), // Wave 9.3 B2: identity server-resolved (Article III.4)
		QCStatus:        "Pending",
		Items:           items,
	}

	// Validate items against PO
	poItemMap := make(map[string]PurchaseOrderItem)
	for _, poItem := range po.Items {
		poItemMap[poItem.ID] = poItem
	}

	for i, item := range grn.Items {
		poItem, exists := poItemMap[item.POItemID]
		if !exists {
			return GoodsReceivedNote{}, newError("INVALID_PO_ITEM", fmt.Sprintf("PO item %s not found", item.POItemID), "")
		}

		// Set quantity ordered from PO
		grn.Items[i].QuantityOrdered = poItem.Quantity
		grn.Items[i].ProductID = poItem.ProductID

		// P0-1 Fix: Calculate QuantityAccepted = QuantityReceived - QuantityRejected
		grn.Items[i].QuantityAccepted = item.QuantityReceived - item.QuantityRejected

		// Validate received quantity
		totalReceived := poItem.QuantityReceived + item.QuantityReceived
		if totalReceived > poItem.Quantity {
			return GoodsReceivedNote{}, newError("QUANTITY_EXCEEDED",
				fmt.Sprintf("Received quantity (%.2f) exceeds ordered quantity (%.2f) for product %s",
					totalReceived, poItem.Quantity, poItem.ProductCode), "")
		}
	}

	// Create GRN in database
	if err := a.db.Create(&grn).Error; err != nil {
		log.Printf("❌ Failed to create GRN: %v", err)
		return GoodsReceivedNote{}, newError("DB_CREATE_FAILED", "Failed to create GRN", err.Error())
	}

	log.Printf("✅ Created GRN %s from PO %s with %d items", grn.GRNNumber, po.PONumber, len(items))
	return grn, nil
}

// =============================================================================
// RECEIVE AGAINST PO WITH SERIAL NUMBERS (Phase 23)
// =============================================================================

// GRNItemWithSerials wraps a GRN item with serial number entries for serialized products
type GRNItemWithSerials struct {
	GRNItem
	SerialNumbers []string `json:"serial_numbers"`
}

// ReceiveAgainstPOWithSerials creates a GRN with serial number tracking
func (a *App) ReceiveAgainstPOWithSerials(poID string, items []GRNItemWithSerials) (GoodsReceivedNote, error) {
	if err := a.requirePermission("grn:create"); err != nil {
		return GoodsReceivedNote{}, err
	}
	if a.db == nil {
		return GoodsReceivedNote{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Verify PO exists
	var po PurchaseOrder
	if err := a.db.Preload("Items").First(&po, "id = ?", poID).Error; err != nil {
		return GoodsReceivedNote{}, newError("PO_NOT_FOUND", "Purchase Order not found", err.Error())
	}

	// Build PO item map for validation
	poItemMap := make(map[string]PurchaseOrderItem)
	for _, poItem := range po.Items {
		poItemMap[poItem.ID] = poItem
	}

	// Wave 9.8 B1: batch-fetch RequiresSerialTracking for the products on this
	// receive so the empty-serials gap below can be closed (previously
	// SERIAL_COUNT_MISMATCH only fired when serials were present at all — a
	// serialized line submitted with zero serials sailed through).
	serialRequiredByProduct := make(map[string]bool)
	{
		productIDs := make([]string, 0, len(items))
		seenProduct := make(map[string]bool)
		for _, item := range items {
			poItem := poItemMap[item.POItemID]
			if poItem.ProductID != "" && !seenProduct[poItem.ProductID] {
				seenProduct[poItem.ProductID] = true
				productIDs = append(productIDs, poItem.ProductID)
			}
		}
		if len(productIDs) > 0 {
			var products []ProductMaster
			if err := a.db.Where("id IN ?", productIDs).Find(&products).Error; err == nil {
				for _, p := range products {
					serialRequiredByProduct[p.ID] = p.RequiresSerialTracking
				}
			}
		}
	}

	// Pre-validate serial numbers
	for _, item := range items {
		poItem := poItemMap[item.POItemID]

		if len(item.SerialNumbers) == 0 {
			if serialRequiredByProduct[poItem.ProductID] {
				return GoodsReceivedNote{}, newError("SERIAL_REQUIRED",
					fmt.Sprintf("Product %s requires serial tracking — supply one serial number per unit received", poItem.ProductCode), "")
			}
			continue
		}

		// Validate serial count matches received quantity
		if float64(len(item.SerialNumbers)) != item.QuantityReceived {
			return GoodsReceivedNote{}, newError("SERIAL_COUNT_MISMATCH",
				fmt.Sprintf("Serial count (%d) must match received quantity (%.0f) for %s",
					len(item.SerialNumbers), item.QuantityReceived, poItem.ProductCode), "")
		}
		// Validate no duplicate serials in DB
		for _, sn := range item.SerialNumbers {
			var existing SerialNumber
			if err := a.db.Where("serial_no = ?", sn).First(&existing).Error; err == nil {
				return GoodsReceivedNote{}, newError("DUPLICATE_SERIAL",
					fmt.Sprintf("Serial number %s already exists (product: %s)", sn, existing.ProductCode), "")
			}
		}
	}

	// Extract plain GRN items for existing ReceiveAgainstPO logic
	var grnItems []GRNItem
	for _, item := range items {
		grnItems = append(grnItems, item.GRNItem)
	}

	// Create GRN using existing logic
	grn, err := a.ReceiveAgainstPO(poID, grnItems)
	if err != nil {
		return GoodsReceivedNote{}, err
	}

	// Now assign serial numbers to the created GRN items
	for i, item := range items {
		if len(item.SerialNumbers) == 0 {
			continue
		}
		// Find the matching created GRN item
		if i >= len(grn.Items) {
			break
		}
		grnItem := grn.Items[i]
		poItem := poItemMap[item.POItemID]

		if err := a.assignSerialsToGRN(
			grnItem.ID, grn.GRNNumber,
			poID, po.PONumber,
			poItem.ProductID, poItem.ProductCode,
			item.SerialNumbers,
			grn.ReceivedDate,
		); err != nil {
			log.Printf("⚠️ Failed to assign serials to GRN item %s: %v", grnItem.ID, err)
			// Don't fail the GRN - serials can be added later
		}
	}

	log.Printf("✅ Created GRN %s with serial tracking from PO %s", grn.GRNNumber, po.PONumber)
	return grn, nil
}

// =============================================================================
// RECEIVE + COMPLETE ATOMIC WRAPPERS (Wave 9.7 tight-ship-2)
// =============================================================================
//
// ReceiveAgainstPO / ReceiveAgainstPOWithSerials only create a PENDING GRN —
// no stock posts and PO status doesn't move (that's CompleteGRN's job, which
// owns the FOR UPDATE row-lock + CompletedAt/grnHasPostedMovement idempotency
// guard above). A caller that invokes create-only reproduces the cosmetic
// PO-status-flip bug this wave fixes: the PO can look "Received" with no
// stock movement behind it. These wrappers chain create -> complete so a
// single frontend action either fully posts (GRN completed, stock moved via
// reconcileInventoryReceipt, PO status advanced via updatePOStatus) or fails
// loudly — never a silently stranded Pending GRN.

// ReceiveAndCompletePO creates a GRN against poID and immediately completes
// it in the same call, so stock posts and the PO's real status (Received /
// Partially Received) advances atomically from the caller's point of view.
// If CompleteGRN fails after the GRN was created, the just-created Pending
// GRN is rolled back (soft-deleted, which also releases any claimed serials)
// so it isn't left stranded; the completion error is returned either way.
func (a *App) ReceiveAndCompletePO(poID string, items []GRNItem) (GRNResponse, error) {
	// Wave 9.8 B1 defense-in-depth: this plain (non-serial) receive path has
	// no way to carry per-unit serials — GRNItem has no SerialNumbers field.
	// If any line's product requires serial tracking, refuse here instead of
	// silently receiving it unserialized; the caller must route that line
	// through ReceiveAndCompletePOWithSerials, where the len(serials)==qty
	// check is the enforcement of record.
	if err := a.rejectSerializedItemsWithoutSerials(items); err != nil {
		return GRNResponse{}, err
	}

	grn, err := a.ReceiveAgainstPO(poID, items)
	if err != nil {
		return GRNResponse{}, err
	}

	if completeErr := a.CompleteGRN(grn.ID); completeErr != nil {
		log.Printf("⚠️ ReceiveAndCompletePO: CompleteGRN failed for GRN %s (PO %s), rolling back the pending GRN it created: %v",
			grn.GRNNumber, poID, completeErr)
		if delErr := crmprocurement.DeleteGRN(a.db, grn.ID); delErr != nil {
			log.Printf("❌ ReceiveAndCompletePO: FAILED to roll back stranded pending GRN %s after completion failure — manual cleanup required: %v",
				grn.GRNNumber, delErr)
		}
		return GRNResponse{}, completeErr
	}

	resp, err := a.GetGRN(grn.ID)
	if err != nil {
		return GRNResponse{}, err
	}
	return *resp, nil
}

// ReceiveAndCompletePOWithSerials is the serial-tracked counterpart of
// ReceiveAndCompletePO: creates a GRN (minting serial numbers via
// assignSerialsToGRN) then immediately completes it. Same rollback-on-
// failure behavior — DeleteGRN resets any minted serials back to Available.
func (a *App) ReceiveAndCompletePOWithSerials(poID string, items []GRNItemWithSerials) (GRNResponse, error) {
	grn, err := a.ReceiveAgainstPOWithSerials(poID, items)
	if err != nil {
		return GRNResponse{}, err
	}

	if completeErr := a.CompleteGRN(grn.ID); completeErr != nil {
		log.Printf("⚠️ ReceiveAndCompletePOWithSerials: CompleteGRN failed for GRN %s (PO %s), rolling back the pending GRN it created: %v",
			grn.GRNNumber, poID, completeErr)
		if delErr := crmprocurement.DeleteGRN(a.db, grn.ID); delErr != nil {
			log.Printf("❌ ReceiveAndCompletePOWithSerials: FAILED to roll back stranded pending GRN %s after completion failure — manual cleanup required: %v",
				grn.GRNNumber, delErr)
		}
		return GRNResponse{}, completeErr
	}

	resp, err := a.GetGRN(grn.ID)
	if err != nil {
		return GRNResponse{}, err
	}
	return *resp, nil
}

// =============================================================================
// P2 FIX: GRN DISCREPANCY WORKFLOW
// =============================================================================

// GRNDiscrepancy tracks quality/quantity issues for supplier follow-up
type GRNDiscrepancy struct {
	ID              string     `json:"id"`
	GRNID           string     `json:"grn_id"`
	GRNItemID       string     `json:"grn_item_id"`
	ProductCode     string     `json:"product_code"`
	DiscrepancyType string     `json:"discrepancy_type"` // "quantity_short", "quality_issue", "damaged", "wrong_item"
	Reason          string     `json:"reason"`
	RejectedQty     float64    `json:"rejected_qty"`
	RaisedBy        string     `json:"raised_by"`
	RaisedAt        time.Time  `json:"raised_at"`
	Status          string     `json:"status"` // "open", "supplier_contacted", "resolved", "credited"
	Resolution      string     `json:"resolution"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	CostImpactBHD   float64    `json:"cost_impact_bhd"`
}

// RaiseGRNDiscrepancy creates a discrepancy record for supplier follow-up
func (a *App) RaiseGRNDiscrepancy(grnID string, itemID string, reason string, discrepancyType string, rejectedQty float64) error {
	if err := a.requirePermission("grn:qc"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get GRN and item details
	var grn GoodsReceivedNote
	if err := a.db.Preload("Items").First(&grn, "id = ?", grnID).Error; err != nil {
		return newError("GRN_NOT_FOUND", "GRN not found", err.Error())
	}

	// Find the specific item
	var grnItem *GRNItem
	for i := range grn.Items {
		if grn.Items[i].ID == itemID {
			grnItem = &grn.Items[i]
			break
		}
	}

	if grnItem == nil {
		return newError("GRN_ITEM_NOT_FOUND", "GRN item not found", "")
	}

	// Get product code and reference cost from the linked PO item (Band-2):
	// the cost impact comes from the PO/product contract, not a placeholder.
	var po PurchaseOrder
	productCode := "Unknown"
	unitCost := 0.0
	if err := a.db.Preload("Items").First(&po, "id = ?", grn.PurchaseOrderID).Error; err == nil {
		for _, poItem := range po.Items {
			if poItem.ID == grnItem.POItemID {
				productCode = poItem.ProductCode
				unitCost = inventory.ResolvePurchaseOrderItemReferenceCost(a.db, poItem).UnitCostBHD
				break
			}
		}
	}

	costImpact := rejectedQty * unitCost

	log.Printf("⚠️ GRN Discrepancy raised for %s: %s - %s (Qty: %.2f, Type: %s)",
		grn.GRNNumber, productCode, reason, rejectedQty, discrepancyType)

	// P2 FIX: Link to supplier performance tracking
	// Get supplier from PO
	if err := a.db.First(&po, "id = ?", grn.PurchaseOrderID).Error; err == nil {
		// Create supplier issue for tracking
		issue := SupplierIssue{
			SupplierID:  po.SupplierID,
			OrderRef:    grn.GRNNumber,
			Description: fmt.Sprintf("GRN Discrepancy: %s - %s (Qty: %.2f)", discrepancyType, reason, rejectedQty),
			Status:      "open",
			CostBHD:     costImpact,
		}
		if err := a.db.Create(&issue).Error; err != nil {
			log.Printf("⚠️ Failed to create supplier issue: %v", err)
		} else {
			log.Printf("📝 Linked discrepancy to supplier performance tracking (Issue ID: %s)", issue.ID)
		}
	}

	// The persisted discrepancy record IS the SupplierIssue created above —
	// surfaced on the Supplier detail "Issues" tab. There is intentionally
	// no separate grn_discrepancies row: SupplierIssue is the single source
	// of truth for GRN discrepancies, deliberately.

	return nil
}
