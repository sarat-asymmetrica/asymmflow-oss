// Procurement deletions (purchase orders, GRNs), moved from the trading
// root in Wave 6 (Mission A.2). The host keeps the delete guard and RBAC
// check; the status rules, serial-number resets, and cascades live here.
package procurement

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/kernel/apperr"
)

// DeletePurchaseOrder soft-deletes a purchase order. Only Draft and
// Cancelled POs are deletable.
func DeletePurchaseOrder(db *gorm.DB, id string) error {
	if db == nil {
		return apperr.New("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Verify PO exists
	var po crm.PurchaseOrder
	if err := db.First(&po, "id = ?", id).Error; err != nil {
		return apperr.New("PO_NOT_FOUND", "Purchase order not found", err.Error())
	}

	// P2 FIX: Only Draft and Cancelled POs can be deleted
	deletableStatuses := map[string]bool{"Draft": true, "Cancelled": true}
	if !deletableStatuses[po.Status] {
		return apperr.New("PO_INVALID_STATUS",
			fmt.Sprintf("Cannot delete PO %s: status is '%s' (only Draft or Cancelled POs can be deleted)", po.PONumber, po.Status), "")
	}

	// Soft delete (GORM handles DeletedAt automatically)
	if err := db.Delete(&po).Error; err != nil {
		log.Printf("❌ Failed to delete purchase order: %v", err)
		return apperr.New("DB_DELETE_FAILED", "Failed to delete purchase order", err.Error())
	}

	log.Printf("✅ Deleted Purchase Order %s", po.PONumber)
	return nil
}

// DeleteGRN deletes a goods received note with its items, resetting any
// serial numbers the GRN had claimed back to Available.
func DeleteGRN(db *gorm.DB, id string) error {
	if db == nil {
		return apperr.New("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Verify GRN exists
	var grn crm.GoodsReceivedNote
	if err := db.First(&grn, "id = ?", id).Error; err != nil {
		return apperr.New("GRN_NOT_FOUND", "GRN not found", err.Error())
	}

	// Reset serial numbers linked to this GRN before deleting items
	if err := db.Model(&crm.SerialNumber{}).
		Where("grn_number = ?", grn.GRNNumber).
		Updates(map[string]any{
			"status":      "Available",
			"grn_item_id": "",
			"grn_number":  "",
		}).Error; err != nil {
		log.Printf("⚠️ Failed to reset serial numbers for GRN %s: %v", grn.GRNNumber, err)
	}

	// Delete GRN items first (cascade)
	if err := db.Where("grn_id = ?", id).Delete(&crm.GRNItem{}).Error; err != nil {
		log.Printf("❌ Failed to delete GRN items: %v", err)
		return apperr.New("DB_DELETE_FAILED", "Failed to delete GRN items", err.Error())
	}

	// Delete GRN
	if err := db.Delete(&grn).Error; err != nil {
		log.Printf("❌ Failed to delete GRN: %v", err)
		return apperr.New("DB_DELETE_FAILED", "Failed to delete GRN", err.Error())
	}

	log.Printf("✅ Deleted GRN %s", grn.GRNNumber)
	return nil
}
