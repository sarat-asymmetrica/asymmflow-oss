// Delivery note deletion, moved from the trading root in Wave 6
// (Mission A.2). The host keeps the delete guard and RBAC check; the
// status rule and serial release live here with the rest of the
// fulfillment serial logic.
package fulfillment

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/kernel/apperr"
)

// DeleteDeliveryNote soft-deletes a delivery note. Only Prepared DNs are
// deletable (serial integrity); reserved serials are released back to
// Available best-effort before the delete.
func DeleteDeliveryNote(db *gorm.DB, id string) error {
	if db == nil {
		return apperr.New("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var dn crm.DeliveryNote
	if err := db.First(&dn, "id = ?", id).Error; err != nil {
		return apperr.New("DN_NOT_FOUND", "Delivery note not found", err.Error())
	}

	// Status guard: only Prepared DNs can be deleted (serial integrity)
	if dn.Status != "Prepared" {
		return apperr.New("DN_INVALID_STATUS",
			fmt.Sprintf("Cannot delete DN %s: status is %s (only Prepared delivery notes can be deleted)", dn.DNNumber, dn.Status), "")
	}

	// P2 FIX: Release allocated serials before soft-deleting DN
	// Serials pointing to this DN should return to Available status
	if err := db.Model(&crm.SerialNumber{}).
		Where("dn_number = ? AND status = ?", dn.DNNumber, "Reserved").
		Updates(map[string]any{
			"status":        "Available",
			"dn_number":     "",
			"dn_item_id":    "",
			"customer_id":   "",
			"customer_name": "",
		}).Error; err != nil {
		log.Printf("Warning: Failed to release serials for DN %s: %v", dn.DNNumber, err)
		// Continue with deletion — serial cleanup is best-effort
	}

	// Soft delete (GORM will handle DeletedAt)
	if err := db.Delete(&dn).Error; err != nil {
		return apperr.New("DB_DELETE_FAILED", "Failed to delete delivery note", err.Error())
	}

	log.Printf("🗑️ Deleted DeliveryNote: %s", dn.DNNumber)
	return nil
}
