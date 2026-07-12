// Serial-number lifecycle: PO Receipt (GRN) → Inventory → Dispatch (DN) →
// Delivery → Invoice. Government clients require serial-level traceability.
//
// Wave 5 A.1: this is a W4-D1 peel — the logic moved inward from the root
// serial_number_service.go. The service needs only the database and the CRM
// models (which already live in pkg/crm), so no host ports are required;
// RBAC guards stay with the host's thin delegates.
package fulfillment

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/kernel/text"
)

// MaxSerialsPerBatch caps every registration/allocation batch (P1-5).
const MaxSerialsPerBatch = 500

// Serials is the serial-number lifecycle service.
type Serials struct {
	db *gorm.DB
}

func NewSerials(db *gorm.DB) *Serials { return &Serials{db: db} }

// validateBatch trims every serial in place and enforces batch/shape limits
// (P1-5 batch cap, P1-8 format validation).
func validateBatch(serialNos []string) error {
	if len(serialNos) > MaxSerialsPerBatch {
		return fmt.Errorf("maximum %d serial numbers per batch (got %d)", MaxSerialsPerBatch, len(serialNos))
	}
	for i, sn := range serialNos {
		sn = strings.TrimSpace(sn)
		serialNos[i] = sn
		if sn == "" {
			return fmt.Errorf("serial number at position %d is empty", i+1)
		}
		if len(sn) > 255 {
			return fmt.Errorf("serial number at position %d exceeds 255 characters", i+1)
		}
	}
	return nil
}

// Register creates serial number records for a product (manual registration).
func (s *Serials) Register(productID string, serialNos []string) ([]crm.SerialNumber, error) {
	if len(serialNos) == 0 {
		return nil, fmt.Errorf("at least one serial number is required")
	}
	if err := validateBatch(serialNos); err != nil {
		return nil, err
	}

	var product crm.ProductMaster
	if err := s.db.Where("id = ?", productID).First(&product).Error; err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	// S5 FIX: Duplicate check + creation inside single transaction to prevent TOCTOU race
	var created []crm.SerialNumber
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, sn := range serialNos {
			var existing crm.SerialNumber
			if err := tx.Where("serial_no = ?", sn).First(&existing).Error; err == nil {
				return fmt.Errorf("serial number %s already exists (product: %s)", sn, existing.ProductCode)
			}

			record := crm.SerialNumber{
				ProductID:   productID,
				ProductCode: product.ProductCode,
				SerialNo:    sn,
				Status:      "Available",
			}
			if err := tx.Create(&record).Error; err != nil {
				return fmt.Errorf("failed to create serial %s: %w", sn, err)
			}
			created = append(created, record)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Printf("✅ Registered %d serial numbers for product %s", len(created), product.ProductCode)
	return created, nil
}

// ByNumber retrieves a serial number record by its serial number string.
func (s *Serials) ByNumber(serialNo string) (crm.SerialNumber, error) {
	var sn crm.SerialNumber
	if err := s.db.Where("serial_no = ?", serialNo).First(&sn).Error; err != nil {
		return crm.SerialNumber{}, fmt.Errorf("serial number not found: %w", err)
	}
	return sn, nil
}

// Search matches serials by partial serial_no, product_code, or customer_name.
func (s *Serials) Search(query string, limit int) ([]crm.SerialNumber, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	searchPattern := "%" + text.EscapeLike(query) + "%"
	var serials []crm.SerialNumber
	if err := s.db.Where("serial_no LIKE ? ESCAPE '\\' OR product_code LIKE ? ESCAPE '\\' OR customer_name LIKE ? ESCAPE '\\'",
		searchPattern, searchPattern, searchPattern).
		Order("created_at DESC").
		Limit(limit).
		Find(&serials).Error; err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	return serials, nil
}

func (s *Serials) capped(label string, out *[]crm.SerialNumber, q *gorm.DB) ([]crm.SerialNumber, error) {
	if err := q.Limit(1000).Find(out).Error; err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	if len(*out) >= 1000 {
		log.Printf("WARNING: %s query hit 1000 limit — results may be truncated", label)
	}
	return *out, nil
}

// ByProduct returns all serial numbers for a specific product.
func (s *Serials) ByProduct(productID string) ([]crm.SerialNumber, error) {
	var serials []crm.SerialNumber
	return s.capped(
		fmt.Sprintf("GetSerialsByProduct (product %s)", productID),
		&serials,
		s.db.Where("product_id = ?", productID).Order("created_at DESC"),
	)
}

// ByCustomer returns all serial numbers delivered to a specific customer.
func (s *Serials) ByCustomer(customerID string) ([]crm.SerialNumber, error) {
	var serials []crm.SerialNumber
	return s.capped(
		fmt.Sprintf("GetSerialsByCustomer (customer %s)", customerID),
		&serials,
		s.db.Where("customer_id = ?", customerID).Order("created_at DESC"),
	)
}

// Available returns serial numbers with status "Available" for a product.
func (s *Serials) Available(productID string) ([]crm.SerialNumber, error) {
	var serials []crm.SerialNumber
	return s.capped(
		fmt.Sprintf("GetAvailableSerials (product %s)", productID),
		&serials,
		s.db.Where("product_id = ? AND status = ?", productID, "Available").Order("created_at ASC"),
	)
}

// ForInvoiceItem returns serials on a specific invoice for a specific product.
func (s *Serials) ForInvoiceItem(invoiceID, productID string) ([]crm.SerialNumber, error) {
	var serials []crm.SerialNumber
	return s.capped(
		fmt.Sprintf("GetSerialsForInvoiceItem (invoice %s, product %s)", invoiceID, productID),
		&serials,
		s.db.Where("invoice_id = ? AND product_id = ?", invoiceID, productID).Order("serial_no ASC"),
	)
}

// UpdateWarranty sets warranty duration and calculates the end date when a
// warranty start date is already known.
func (s *Serials) UpdateWarranty(serialID string, warrantyMonths int) error {
	if warrantyMonths <= 0 {
		return fmt.Errorf("warranty months must be positive")
	}

	var sn crm.SerialNumber
	if err := s.db.Where("id = ?", serialID).First(&sn).Error; err != nil {
		return fmt.Errorf("serial not found: %w", err)
	}

	updates := map[string]any{
		"warranty_months": warrantyMonths,
	}
	if sn.WarrantyStartDate != nil {
		endDate := sn.WarrantyStartDate.AddDate(0, warrantyMonths, 0)
		updates["warranty_end_date"] = endDate
	}

	if err := s.db.Model(&sn).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update warranty: %w", err)
	}

	log.Printf("✅ Updated warranty for serial %s: %d months", sn.SerialNo, warrantyMonths)
	return nil
}

// AttachCalibrationCert sets calibration certificate path and date on a serial.
func (s *Serials) AttachCalibrationCert(serialID, certPath string) error {
	// P1-1: Sanitize file path — store only the base filename to prevent path traversal
	certPath = strings.TrimSpace(certPath)
	if certPath == "" {
		return fmt.Errorf("certificate path is required")
	}
	certPath = filepath.Base(certPath)
	if certPath == "." || certPath == "/" || certPath == "\\" {
		return fmt.Errorf("invalid certificate path")
	}

	now := time.Now()
	updates := map[string]any{
		"calibration_cert_path": certPath,
		"calibration_date":      now,
	}
	if err := s.db.Model(&crm.SerialNumber{}).Where("id = ?", serialID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to attach calibration cert: %w", err)
	}

	log.Printf("✅ Attached calibration cert to serial %s", serialID)
	return nil
}

// AssignToGRN creates serial records when goods are received against a PO.
func (s *Serials) AssignToGRN(grnItemID, grnNumber, poID, poNumber, productID, productCode string, serialNos []string, receivedDate time.Time) error {
	if err := validateBatch(serialNos); err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, sn := range serialNos {
			var existing crm.SerialNumber
			if err := tx.Where("serial_no = ?", sn).First(&existing).Error; err == nil {
				return fmt.Errorf("serial number %s already exists (product: %s)", sn, existing.ProductCode)
			}

			record := crm.SerialNumber{
				ProductID:    productID,
				ProductCode:  productCode,
				SerialNo:     sn,
				Status:       "Available",
				POID:         poID,
				PONumber:     poNumber,
				GRNItemID:    grnItemID,
				GRNNumber:    grnNumber,
				ReceivedDate: &receivedDate,
			}
			if err := tx.Create(&record).Error; err != nil {
				return fmt.Errorf("failed to create serial %s: %w", sn, err)
			}
		}
		log.Printf("✅ Assigned %d serials to GRN %s (PO: %s, Product: %s)", len(serialNos), grnNumber, poNumber, productCode)
		return nil
	})
}

// AllocateToDN reserves serials for a delivery note. Uses an atomic UPDATE
// with WHERE status='Available' to prevent double-allocation (safe on SQLite).
func (s *Serials) AllocateToDN(dnItemID, dnNumber, customerID, customerName string, serialNos []string) error {
	// P3 S7 FIX: Batch size validation to prevent unbounded allocations
	if len(serialNos) > MaxSerialsPerBatch {
		return fmt.Errorf("batch size %d exceeds maximum of %d serials per allocation", len(serialNos), MaxSerialsPerBatch)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, sn := range serialNos {
			// Atomic check-and-update: only updates if status is still Available
			result := tx.Model(&crm.SerialNumber{}).
				Where("serial_no = ? AND status = ?", sn, "Available").
				Updates(map[string]any{
					"status":        "Reserved",
					"dn_item_id":    dnItemID,
					"dn_number":     dnNumber,
					"customer_id":   customerID,
					"customer_name": customerName,
				})
			if result.Error != nil {
				return fmt.Errorf("failed to allocate serial %s: %w", sn, result.Error)
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("serial %s is not available for allocation (already reserved or does not exist)", sn)
			}
		}
		log.Printf("✅ Allocated %d serials to DN %s (Customer: %s)", len(serialNos), dnNumber, customerName)
		return nil
	})
}

// MarkShipped updates serials on a DN to "Shipped" status.
func (s *Serials) MarkShipped(dnNumber string) error {
	now := time.Now()
	result := s.db.Model(&crm.SerialNumber{}).
		Where("dn_number = ? AND status = ?", dnNumber, "Reserved").
		Updates(map[string]any{
			"status":       "Shipped",
			"shipped_date": now,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to mark serials shipped: %w", result.Error)
	}
	log.Printf("✅ Marked %d serials as Shipped for DN %s", result.RowsAffected, dnNumber)
	return nil
}

// MarkDelivered updates serials on a DN to "Delivered" and sets warranty start.
func (s *Serials) MarkDelivered(dnID string) error {
	var dn crm.DeliveryNote
	if err := s.db.Where("id = ?", dnID).First(&dn).Error; err != nil {
		return fmt.Errorf("DN not found: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		var serials []crm.SerialNumber
		if err := tx.Where("dn_number = ? AND status IN ?", dn.DNNumber, []string{"Reserved", "Shipped"}).Find(&serials).Error; err != nil {
			return fmt.Errorf("failed to find serials: %w", err)
		}

		for _, serial := range serials {
			updates := map[string]any{
				"status":              "Delivered",
				"warranty_start_date": now,
			}
			if serial.WarrantyMonths > 0 {
				endDate := now.AddDate(0, serial.WarrantyMonths, 0)
				updates["warranty_end_date"] = endDate
			}
			if err := tx.Model(&serial).Updates(updates).Error; err != nil {
				return fmt.Errorf("failed to update serial %s: %w", serial.SerialNo, err)
			}
		}

		log.Printf("✅ Marked %d serials as Delivered for DN %s", len(serials), dn.DNNumber)
		return nil
	})
}

// LinkToInvoice stamps invoice ID on all serials from a delivery note.
func (s *Serials) LinkToInvoice(invoiceID, invoiceNumber, dnID string) error {
	var dn crm.DeliveryNote
	if err := s.db.Where("id = ?", dnID).First(&dn).Error; err != nil {
		return fmt.Errorf("DN not found: %w", err)
	}

	// P2 FIX: Add race condition guard — only stamp serials that haven't been claimed by another invoice
	// P3 S1 FIX: Case-insensitive status comparison to handle data inconsistencies
	result := s.db.Model(&crm.SerialNumber{}).
		Where("dn_number = ? AND UPPER(status) IN ? AND (invoice_id = '' OR invoice_id IS NULL)", dn.DNNumber, []string{"RESERVED", "SHIPPED", "DELIVERED"}).
		Updates(map[string]any{
			"invoice_id":     invoiceID,
			"invoice_number": invoiceNumber,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to link serials to invoice: %w", result.Error)
	}

	log.Printf("✅ Linked %d serials to invoice %s from DN %s", result.RowsAffected, invoiceNumber, dn.DNNumber)
	return nil
}
