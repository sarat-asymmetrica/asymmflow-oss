// Offer note deletion — the one genuinely self-contained leaf the Wave 6
// sales-pipeline measurement surfaced (Mission A.3 allows exactly that:
// a leaf with db-only coupling on pkg-owned models). The pipeline's
// heavy clusters (RFQ/costing lifecycles, offer numbering) stay at the
// host; their models have not migrated.
package pipeline

import (
	"fmt"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/crm"
)

// DeleteOfferNote removes an offer note after verifying it exists.
func DeleteOfferNote(db *gorm.DB, noteID string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	// Verify note exists before deleting
	var note crm.OfferNote
	if err := db.Where("id = ?", noteID).First(&note).Error; err != nil {
		return fmt.Errorf("note not found")
	}
	result := db.Where("id = ?", noteID).Delete(&crm.OfferNote{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete note: %w", result.Error)
	}
	return nil
}
