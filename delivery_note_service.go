package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"

	"gorm.io/gorm/clause"
	crmfulfillment "ph_holdings_app/pkg/crm/fulfillment"
	"ph_holdings_app/pkg/documents/numbering"
)

// =============================================================================
// DELIVERY NOTE CRUD - OPERATIONS PIPELINE
// =============================================================================

// CreateDeliveryNote creates a new delivery note
func (a *App) CreateDeliveryNote(dn DeliveryNote) (DeliveryNote, error) {
	return a.fulfillmentService().CreateDeliveryNote(dn)
}

func createDeliveryNote(a *App, dn DeliveryNote) (DeliveryNote, error) {
	if err := a.requirePermission("delivery_notes:create"); err != nil {
		return DeliveryNote{}, err
	}
	if a.db == nil {
		return DeliveryNote{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Validate order exists
	var order Order
	if err := a.db.First(&order, "id = ?", dn.OrderID).Error; err != nil {
		return DeliveryNote{}, newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	// Auto-generate DN number if not provided
	if dn.DNNumber == "" {
		dnNumber, err := a.GenerateDNNumber()
		if err != nil {
			return DeliveryNote{}, err
		}
		dn.DNNumber = dnNumber
	}

	// Validate positive quantities first (before DB-dependent checks)
	for _, item := range dn.Items {
		if item.QuantityDelivered <= 0 {
			return DeliveryNote{}, newError("INVALID_QUANTITY",
				fmt.Sprintf("Delivery quantity must be positive, got: %.2f", item.QuantityDelivered), "")
		}
	}

	// P3 DN-A6 FIX: Force initial status to "Prepared" regardless of input
	dn.Status = "Prepared"
	if dn.CustomerID == "" {
		dn.CustomerID = order.CustomerID
	}
	if dn.DeliveryDate.IsZero() {
		dn.DeliveryDate = time.Now()
	}
	dn.CreatedBy = a.getCurrentUserID()

	// P1 TOCTOU FIX: Wrap quantity validation + DN creation in a single transaction
	// with order row lock to prevent concurrent over-delivery
	err := a.db.Transaction(func(tx *gorm.DB) error {
		// Lock order row to serialize concurrent DN creation for same order
		var lockedOrder Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedOrder, "id = ?", dn.OrderID).Error; err != nil {
			return newError("ORDER_LOCK_FAILED", "Failed to lock order for delivery validation", err.Error())
		}

		// Validate delivery item quantities won't go negative (inside transaction)
		if len(dn.Items) > 0 && dn.OrderID != "" {
			// Use direct query inside tx (not the exported method which has its own RBAC + limit)
			var existingDNs []DeliveryNote
			if err := tx.Where("order_id = ?", dn.OrderID).Preload("Items").Find(&existingDNs).Error; err != nil {
				return newError("DELIVERY_STATUS_CHECK_FAILED", "Failed to validate delivery quantities", err.Error())
			}

			// Calculate remaining quantities per order item
			deliveredQty := make(map[string]float64)
			for _, existingDN := range existingDNs {
				for _, item := range existingDN.Items {
					if item.OrderItemID != "" {
						deliveredQty[item.OrderItemID] += item.QuantityDelivered
					}
				}
			}

			// Get order items to know total ordered quantities
			var orderItems []OrderItem
			if err := tx.Where("order_id = ?", dn.OrderID).Find(&orderItems).Error; err != nil {
				return newError("ORDER_ITEMS_QUERY_FAILED", "Failed to retrieve order items", err.Error())
			}
			remainingQty := make(map[string]float64)
			for _, oi := range orderItems {
				remainingQty[oi.ID] = float64(oi.Quantity) - deliveredQty[oi.ID]
			}

			for _, item := range dn.Items {
				if item.OrderItemID != "" {
					remaining, ok := remainingQty[item.OrderItemID]
					if !ok {
						return newError("INVALID_ORDER_ITEM",
							fmt.Sprintf("Order item %s not found in order %s", item.OrderItemID, dn.OrderID), "")
					}
					if item.QuantityDelivered > remaining {
						return newError("QUANTITY_EXCEEDED",
							fmt.Sprintf("Cannot deliver %.2f units for %s - only %.2f remaining",
								item.QuantityDelivered, item.ProductCode, remaining), "")
					}
				}
			}
		}

		// Create delivery note inside the same transaction
		if err := tx.Create(&dn).Error; err != nil {
			return newError("DB_CREATE_FAILED", "Failed to create delivery note", err.Error())
		}
		return nil
	})
	if err != nil {
		return DeliveryNote{}, err
	}

	log.Printf("✅ Created DeliveryNote: %s for Order %s", dn.DNNumber, order.OrderNumber)
	return dn, nil
}

// GetDeliveryNotes retrieves all delivery notes
func (a *App) GetDeliveryNotes() ([]DeliveryNote, error) {
	return a.fulfillmentService().GetDeliveryNotes()
}

func getDeliveryNotes(a *App) ([]DeliveryNote, error) {
	if err := a.requirePermission("delivery_notes:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// S1 FIX: Cap results to prevent memory exhaustion (Preload("Items") is expensive)
	var deliveryNotes []DeliveryNote
	if err := a.db.Preload("Items").Order("delivery_date DESC, created_at DESC").Limit(200).Find(&deliveryNotes).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve delivery notes", err.Error())
	}

	log.Printf("📦 Retrieved %d delivery notes", len(deliveryNotes))
	return deliveryNotes, nil
}

// GetDeliveryNoteByID retrieves a single delivery note by ID
func (a *App) GetDeliveryNoteByID(id string) (DeliveryNote, error) {
	return a.fulfillmentService().GetDeliveryNoteByID(id)
}

func getDeliveryNoteByID(a *App, id string) (DeliveryNote, error) {
	if err := a.requirePermission("delivery_notes:view"); err != nil {
		return DeliveryNote{}, err
	}
	if a.db == nil {
		return DeliveryNote{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var dn DeliveryNote
	if err := a.db.Preload("Items").First(&dn, "id = ?", id).Error; err != nil {
		return DeliveryNote{}, newError("DN_NOT_FOUND", "Delivery note not found", err.Error())
	}

	log.Printf("📄 Retrieved DeliveryNote: %s", dn.DNNumber)
	return dn, nil
}

// GenerateDeliveryNotePDF creates a customer-facing delivery note that follows
// the Acme Instrumentation delivery note structure shared by the client.
func (a *App) GenerateDeliveryNotePDF(id string) (string, error) {
	if err := a.requirePermission("delivery_notes:view"); err != nil {
		return "", err
	}
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("delivery note ID is required")
	}
	if a.db == nil {
		return "", newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var dn DeliveryNote
	if err := a.db.Preload("Items").First(&dn, "id = ?", id).Error; err != nil {
		return "", newError("DN_NOT_FOUND", "Delivery note not found", err.Error())
	}
	if dn.DeliveryDate.IsZero() {
		dn.DeliveryDate = time.Now()
	}

	var order Order
	if strings.TrimSpace(dn.OrderID) != "" {
		_ = a.db.First(&order, "id = ?", dn.OrderID).Error
	}

	var customer CustomerMaster
	if strings.TrimSpace(dn.CustomerID) != "" {
		_ = a.db.First(&customer, "id = ?", dn.CustomerID).Error
	}

	division := normalizeDivisionName(order.Division)
	profile := companyDocumentProfile(division)
	customerName := firstNonEmptyString(customer.BusinessName, order.CustomerName, dn.CustomerID)
	deliveryAddress := firstNonEmptyString(dn.DeliveryAddress, order.AttentionAddress, customer.AddressLine1, customer.City, customer.Country)
	billToLines := compactStrings([]string{
		customerName,
		customer.AddressLine1,
		customer.City,
		customer.Country,
		customerTRNLine(customer.TRN),
	})
	if len(billToLines) == 1 && strings.TrimSpace(order.AttentionAddress) != "" {
		billToLines = append(billToLines, splitNonEmptyLines(order.AttentionAddress)...)
	}
	deliveryLines := compactStrings(append([]string{customerName}, splitNonEmptyLines(deliveryAddress)...))
	if len(deliveryLines) == 1 {
		deliveryLines = append(deliveryLines, compactStrings([]string{customer.AddressLine1, customer.City, customer.Country})...)
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()
	a.applyLetterheadForDivision(pdf, profile.Division)

	pdf.SetTextColor(20, 20, 20)
	pdf.SetFont("Times", "B", 17)
	pdf.SetXY(0, 38)
	pdf.CellFormat(210, 8, "Delivery Note", "", 0, "C", false, 0, "")
	pdf.Line(89, 47, 121, 47)

	pdf.SetFont("Times", "B", 12)
	pdf.SetXY(35, 66)
	pdf.Cell(20, 6, "Ref:")
	pdf.SetFont("Times", "", 12)
	pdf.Cell(58, 6, sanitizeForPDF(firstNonEmptyString(dn.DNNumber, "-")))

	pdf.SetFont("Times", "B", 12)
	pdf.SetXY(126, 66)
	pdf.Cell(18, 6, "Dated:")
	pdf.SetFont("Times", "B", 12)
	pdf.Cell(40, 6, dn.DeliveryDate.Format("02-01-2006"))

	y := 82.0
	pdf.SetFont("Times", "B", 11)
	pdf.SetXY(35, y)
	pdf.Cell(0, 5, "Bill to,")
	y += 6
	pdf.SetFont("Times", "", 11)
	for _, line := range billToLines {
		pdf.SetXY(35, y)
		pdf.MultiCell(95, 5, sanitizeForPDF(line), "", "L", false)
		y = pdf.GetY()
	}

	y += 5
	pdf.SetFont("Times", "B", 11)
	pdf.SetXY(35, y)
	pdf.Cell(0, 5, "Delivery Location:")
	y += 6
	pdf.SetFont("Times", "", 11)
	for _, line := range deliveryLines {
		pdf.SetXY(35, y)
		pdf.MultiCell(110, 5, sanitizeForPDF(line), "", "L", false)
		y = pdf.GetY()
	}

	y += 8
	pdf.SetFont("Times", "", 11)
	pdf.SetXY(35, y)
	pdf.Cell(42, 6, "PURCHASE ORDER NO.:")
	pdf.SetFont("Times", "B", 11)
	pdf.Cell(80, 6, sanitizeForPDF(firstNonEmptyString(order.CustomerPONumber, order.OrderNumber, "-")))
	y += 7
	pdf.SetFont("Times", "", 11)
	pdf.SetXY(35, y)
	pdf.Cell(18, 6, "DATE :")
	orderDate := order.OrderDate
	if orderDate.IsZero() {
		orderDate = dn.CreatedAt
	}
	pdf.Cell(60, 6, orderDate.Format("02 Jan, 2006"))

	tableY := y + 16
	tableEndY := a.drawDeliveryNoteTable(pdf, tableY, dn.Items, profile.Division)

	const signatureBlockHeight = 22.0
	const signatureDefaultY = 222.0
	const deliveryNoteFooterSafeY = 268.0
	signatureY := mathMax(tableEndY+10, signatureDefaultY)
	if signatureY+signatureBlockHeight > deliveryNoteFooterSafeY {
		pdf.AddPage()
		a.applyLetterheadForDivision(pdf, profile.Division)
		signatureY = signatureDefaultY
	}
	pdf.SetFont("Times", "B", 10)
	pdf.SetXY(35, signatureY)
	pdf.Cell(70, 5, "Supplied in Good Order and condition.")
	pdf.SetXY(112, signatureY)
	pdf.Cell(78, 5, "Received in Good order and condition")

	pdf.SetFont("Times", "", 10)
	pdf.SetXY(35, signatureY+12)
	pdf.Cell(70, 5, "For "+sanitizeForPDF(profile.LegalName))
	pdf.SetXY(112, signatureY+12)
	pdf.Cell(78, 5, "For "+sanitizeForPDF(customerName))

	docYear := dn.DeliveryDate.Year()
	if docYear <= 0 {
		docYear = time.Now().Year()
	}
	outputDir := a.getExportDir("customer", customerName, "Delivery Notes", docYear)
	fileName := fmt.Sprintf("Delivery_Note_%s.pdf", sanitizeFilename(firstNonEmptyString(dn.DNNumber, id)))
	outputPath := filepath.Join(outputDir, fileName)
	if err := pdf.OutputFileAndClose(outputPath); err != nil {
		return "", fmt.Errorf("failed to save delivery note PDF: %w", err)
	}

	log.Printf("✅ Delivery note PDF generated: %s", outputPath)
	return outputPath, nil
}

func (a *App) drawDeliveryNoteTable(pdf *gofpdf.Fpdf, y float64, items []DeliveryNoteItem, division string) float64 {
	x := 35.0
	widths := []float64{13, 89, 25, 25}
	headers := []string{"Item\nNo", "Order Code Description", "Qty ordered", "Qty Delivered"}
	rowHeight := 10.0
	pageBottomY := 252.0

	drawHeader := func(headerY float64) {
		pdf.SetLineWidth(0.2)
		pdf.SetFont("Times", "B", 10)
		pdf.SetXY(x, headerY)
		for i, header := range headers {
			cellX := pdf.GetX()
			cellY := pdf.GetY()
			pdf.Rect(cellX, cellY, widths[i], rowHeight, "D")
			pdf.MultiCell(widths[i], 5, sanitizeForPDF(header), "", "C", false)
			pdf.SetXY(cellX+widths[i], cellY)
		}
		pdf.SetXY(x, headerY+rowHeight)
		pdf.SetFont("Times", "", 10)
	}

	drawHeader(y)
	for i, item := range items {
		description := strings.TrimSpace(item.Description)
		if strings.TrimSpace(item.ProductCode) != "" {
			description = strings.TrimSpace(description + "\n" + item.ProductCode)
		}
		if description == "" {
			description = "-"
		}
		lines := pdf.SplitText(sanitizeForPDF(description), widths[1]-4)
		height := mathMax(10, float64(len(lines))*5+2)
		if pdf.GetY()+height > pageBottomY && pdf.GetY() > y+rowHeight {
			pdf.AddPage()
			if a != nil {
				a.applyLetterheadForDivision(pdf, division)
			}
			drawHeader(45)
		}

		startX := x
		startY := pdf.GetY()
		pdf.Rect(startX, startY, widths[0], height, "D")
		pdf.SetXY(startX+2, startY+2)
		pdf.CellFormat(widths[0]-4, 5, fmt.Sprintf("%d", i+1), "", 0, "L", false, 0, "")

		descX := startX + widths[0]
		pdf.Rect(descX, startY, widths[1], height, "D")
		pdf.SetXY(descX+2, startY+2)
		pdf.MultiCell(widths[1]-4, 5, sanitizeForPDF(description), "", "L", false)

		qtyOrderedX := descX + widths[1]
		pdf.Rect(qtyOrderedX, startY, widths[2], height, "D")
		pdf.SetXY(qtyOrderedX, startY+2)
		pdf.CellFormat(widths[2], 5, formatQty(item.QuantityOrdered), "", 0, "C", false, 0, "")

		qtyDeliveredX := qtyOrderedX + widths[2]
		pdf.Rect(qtyDeliveredX, startY, widths[3], height, "D")
		pdf.SetXY(qtyDeliveredX, startY+2)
		pdf.CellFormat(widths[3], 5, formatQty(item.QuantityDelivered), "", 0, "C", false, 0, "")

		pdf.SetXY(x, startY+height)
	}

	return pdf.GetY()
}

func customerTRNLine(trn string) string {
	if strings.TrimSpace(trn) == "" {
		return ""
	}
	return "VAT Reg. No. " + strings.TrimSpace(trn)
}

func splitNonEmptyLines(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	return compactStrings(parts)
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func formatQty(value float64) string {
	if mathAbs(value-mathRound(value)) < 0.001 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.2f", value)
}

func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func mathRound(value float64) float64 {
	if value >= 0 {
		return float64(int(value + 0.5))
	}
	return float64(int(value - 0.5))
}

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

// GetDeliveryNotesByOrder retrieves all delivery notes for an order
func (a *App) GetDeliveryNotesByOrder(orderID string) ([]DeliveryNote, error) {
	if err := a.requirePermission("delivery_notes:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var deliveryNotes []DeliveryNote
	if err := a.db.Preload("Items").Where("order_id = ?", orderID).Order("delivery_sequence ASC").Limit(200).Find(&deliveryNotes).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve delivery notes", err.Error())
	}

	log.Printf("📦 Retrieved %d delivery notes for order %s", len(deliveryNotes), orderID)
	return deliveryNotes, nil
}

// GetDeliveryNotesByCustomer retrieves all delivery notes for a customer
func (a *App) GetDeliveryNotesByCustomer(customerID string) ([]DeliveryNote, error) {
	if err := a.requirePermission("delivery_notes:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var deliveryNotes []DeliveryNote
	if err := a.db.Preload("Items").Where("customer_id = ?", customerID).Order("delivery_date DESC").Limit(200).Find(&deliveryNotes).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve delivery notes", err.Error())
	}

	log.Printf("📦 Retrieved %d delivery notes for customer %s", len(deliveryNotes), customerID)
	return deliveryNotes, nil
}

// UpdateDeliveryNote updates an existing delivery note
func (a *App) UpdateDeliveryNote(dn DeliveryNote) (DeliveryNote, error) {
	return a.fulfillmentService().UpdateDeliveryNote(dn)
}

func updateDeliveryNote(a *App, dn DeliveryNote) (DeliveryNote, error) {
	if err := a.requirePermission("delivery_notes:update"); err != nil {
		return DeliveryNote{}, err
	}
	if a.db == nil {
		return DeliveryNote{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Verify exists
	var existing DeliveryNote
	if err := a.db.First(&existing, "id = ?", dn.ID).Error; err != nil {
		return DeliveryNote{}, newError("DN_NOT_FOUND", "Delivery note not found", err.Error())
	}

	// Status guard: only Prepared DNs can be updated (allowlist — blocks InTransit, Signed, Cancelled, Dispatched, Delivered)
	if existing.Status != "Prepared" {
		return DeliveryNote{}, newError("DN_INVALID_STATUS",
			fmt.Sprintf("Cannot update DN %s: status is %s (only Prepared delivery notes can be updated)", existing.DNNumber, existing.Status), "")
	}

	// Strip Status from input — status changes must go through DispatchDeliveryNote/ConfirmDeliveryNote
	dn.Status = ""

	// Allow DN number edit in Prepared status with uniqueness check
	if dn.DNNumber != "" && dn.DNNumber != existing.DNNumber {
		var count int64
		a.db.Model(&DeliveryNote{}).Where("dn_number = ? AND id != ?", dn.DNNumber, dn.ID).Count(&count)
		if count > 0 {
			return DeliveryNote{}, newError("DN_NUMBER_EXISTS",
				fmt.Sprintf("Delivery note number %s already exists", dn.DNNumber), "")
		}
	}

	// Update with transaction to handle items
	tx := a.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update delivery note. Mission I (I-12): proof-of-delivery fields are set
	// by the dispatch/confirm workflow — a pre-dispatch edit payload must not
	// be able to forge them.
	if err := tx.Model(&existing).
		Omit("SignedBy", "SignedAt", "SignatureImage", "CreatedBy", "CreatedAt").
		Updates(dn).Error; err != nil {
		tx.Rollback()
		return DeliveryNote{}, newError("DB_UPDATE_FAILED", "Failed to update delivery note", err.Error())
	}

	// Only replace items if new items are explicitly provided
	if len(dn.Items) > 0 {
		// Delete existing items ONLY when we have replacements
		if err := tx.Where("delivery_note_id = ?", dn.ID).Delete(&DeliveryNoteItem{}).Error; err != nil {
			tx.Rollback()
			return DeliveryNote{}, newError("DB_UPDATE_FAILED", "Failed to delete old items", err.Error())
		}

		// Create new items
		for i := range dn.Items {
			dn.Items[i].DeliveryNoteID = dn.ID
			if err := tx.Create(&dn.Items[i]).Error; err != nil {
				tx.Rollback()
				return DeliveryNote{}, newError("DB_CREATE_FAILED", "Failed to create item", err.Error())
			}
		}
	}
	// If no items provided, keep existing items (don't delete)

	if err := tx.Commit().Error; err != nil {
		return DeliveryNote{}, newError("DB_COMMIT_FAILED", "Failed to commit update", err.Error())
	}

	// Reload with items
	var updated DeliveryNote
	if err := a.db.Preload("Items").First(&updated, "id = ?", dn.ID).Error; err != nil {
		return DeliveryNote{}, newError("DB_QUERY_FAILED", "Failed to reload delivery note", err.Error())
	}

	log.Printf("✅ Updated DeliveryNote: %s", updated.DNNumber)
	return updated, nil
}

// DeleteDeliveryNote deletes a delivery note (soft delete)
func (a *App) DeleteDeliveryNote(id string) error {
	if ok, err := a.guardDeleteOrRequest("delivery_notes:delete", "delivery_note", id, "Delivery note"); !ok {
		return err
	}
	if err := a.requirePermission("delivery_notes:delete"); err != nil {
		return err
	}
	return crmfulfillment.DeleteDeliveryNote(a.db, id)
}

// GenerateDNNumber generates a new DN number in format DN-2026-0001
// S4 FIX: Rewritten to use InvoiceSequence table (same pattern as GenerateInvoiceNumber/GenerateCreditNoteNumber)
// Eliminates nested BEGIN EXCLUSIVE bug and COUNT-based fragility
func (a *App) GenerateDNNumber() (string, error) {
	if a.db == nil {
		return "", newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Delegates to the promoted pkg/documents/numbering engine (Wave 2
	// Mission A). Format and first-of-year seeding are byte-identical to the
	// old inline implementation.
	dnNumber, err := numbering.New(a.db).Next(numbering.Spec{
		Prefix:   "DN",
		Template: "DN-{year}-{seq}",
		Seed: func(tx *gorm.DB, year int) (int64, error) {
			// First DN of the year — seed from existing DN count for migration safety
			var maxExisting int64
			tx.Model(&DeliveryNote{}).
				Where("dn_number LIKE ?", fmt.Sprintf("DN-%d-%%", year)).
				Count(&maxExisting)
			return maxExisting, nil
		},
	}, time.Now())

	if err != nil {
		return "", err
	}
	log.Printf("🔢 Generated DN Number: %s", dnNumber)
	return dnNumber, nil
}

// =============================================================================
// DELIVERY NOTE STATUS UPDATES
// =============================================================================

// DispatchDeliveryNote marks a delivery note as dispatched
func (a *App) DispatchDeliveryNote(id string, driverName string, vehicleNumber string) error {
	return a.fulfillmentService().DispatchDeliveryNote(id, driverName, vehicleNumber)
}

func dispatchDeliveryNote(a *App, id string, driverName string, vehicleNumber string) error {
	if err := a.requirePermission("delivery_notes:dispatch"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var dn DeliveryNote
	if err := a.db.First(&dn, "id = ?", id).Error; err != nil {
		return newError("DN_NOT_FOUND", "Delivery note not found", err.Error())
	}

	// P3 DN-A3 FIX: Reject empty status — cannot dispatch a DN with no status set
	if dn.Status == "" {
		return newError("DN_INVALID_STATUS",
			fmt.Sprintf("Delivery note %s has no status set — cannot dispatch", dn.DNNumber), "")
	}

	// S2 FIX: Only Prepared DNs can be dispatched — prevent status regression
	if dn.Status != "Prepared" {
		return newError("DN_INVALID_STATUS",
			fmt.Sprintf("Cannot dispatch DN %s: status is %s (only Prepared delivery notes can be dispatched)", dn.DNNumber, dn.Status), "")
	}

	// P2 FIX: Wrap DN status + serial status updates in a single transaction for atomicity
	now := time.Now()
	err := a.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"status":         "Dispatched",
			"driver_name":    driverName,
			"vehicle_number": vehicleNumber,
		}
		if err := tx.Model(&dn).Updates(updates).Error; err != nil {
			return newError("DB_UPDATE_FAILED", "Failed to dispatch delivery note", err.Error())
		}

		// Update serial statuses to "Shipped" + set shipped_date — inside same transaction
		if err := tx.Model(&SerialNumber{}).
			Where("dn_number = ? AND UPPER(status) IN ?", dn.DNNumber, []string{"RESERVED"}).
			Updates(map[string]any{
				"status":       "Shipped",
				"shipped_date": now,
			}).Error; err != nil {
			return fmt.Errorf("failed to update serial status for dispatch: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("🚚 Dispatched DN %s with driver %s, vehicle %s", dn.DNNumber, driverName, vehicleNumber)
	return nil
}

// ConfirmDeliveryNote marks a delivery note as delivered and signed.
// The returned string is a non-fatal warning (empty when clean): the DN
// confirmation itself is transactional and never rolled back by a warning,
// but the downstream order-progression steps below can fail independently
// (Inv4) and are surfaced here instead of being silently swallowed.
func (a *App) ConfirmDeliveryNote(id string, signedBy string) (string, error) {
	return a.fulfillmentService().ConfirmDeliveryNote(id, signedBy)
}

func confirmDeliveryNote(a *App, id string, signedBy string) (string, error) {
	if err := a.requirePermission("delivery_notes:confirm"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var dn DeliveryNote
	if err := a.db.First(&dn, "id = ?", id).Error; err != nil {
		return "", newError("DN_NOT_FOUND", "Delivery note not found", err.Error())
	}

	// S3 FIX: Only Dispatched DNs can be confirmed — prevent re-confirmation and status regression
	if dn.Status != "Dispatched" {
		return "", newError("DN_INVALID_STATUS",
			fmt.Sprintf("Cannot confirm DN %s: status is %s (only Dispatched delivery notes can be confirmed)", dn.DNNumber, dn.Status), "")
	}

	// P2 FIX: Wrap DN status + serial status updates in a single transaction for atomicity
	now := time.Now()
	err := a.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"status":    "Delivered",
			"signed_by": signedBy,
			"signed_at": now,
		}
		if err := tx.Model(&dn).Updates(updates).Error; err != nil {
			return newError("DB_UPDATE_FAILED", "Failed to confirm delivery", err.Error())
		}

		// Mark serials as Delivered, set warranty start + end dates — inside same transaction
		// Must iterate per serial to calculate per-product warranty_end_date (varies by WarrantyMonths)
		var serials []SerialNumber
		if err := tx.Where("dn_number = ? AND UPPER(status) IN ?", dn.DNNumber, []string{"RESERVED", "SHIPPED"}).
			Find(&serials).Error; err != nil {
			return fmt.Errorf("failed to find serials for delivery: %w", err)
		}
		for _, serial := range serials {
			serialUpdates := map[string]any{
				"status":              "Delivered",
				"warranty_start_date": now,
			}
			if serial.WarrantyMonths > 0 {
				serialUpdates["warranty_end_date"] = now.AddDate(0, serial.WarrantyMonths, 0)
			}
			if err := tx.Model(&serial).Updates(serialUpdates).Error; err != nil {
				return fmt.Errorf("failed to update serial %s: %w", serial.SerialNo, err)
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	// Inv4: the DN is now durably confirmed above and is NOT rolled back by
	// anything below. The three steps here are order-progression side effects;
	// if they fail we log (as before) AND accumulate a warning to return to the
	// caller instead of swallowing it — the frontend surfaces this as a
	// non-blocking toast so the user knows to verify the order.
	var postConfirmIssues []string

	// E1 Enhancement: Update QuantityShipped on order items and propagate order status
	if err := a.confirmDeliveryAndUpdateOrder(dn); err != nil {
		log.Printf("Warning: Failed to update order items after delivery confirmation: %v", err)
		postConfirmIssues = append(postConfirmIssues, "order item quantities")
	}

	// Also run the existing order status checks for backward compatibility
	if err := a.updateOrderDeliveryStatus(dn.OrderID); err != nil {
		log.Printf("Warning: Failed to update order status: %v", err)
		postConfirmIssues = append(postConfirmIssues, "order delivery status")
	}

	if err := a.ProgressOrderOnDelivery(dn.OrderID); err != nil {
		log.Printf("Warning: Failed to progress order on delivery: %v", err)
		postConfirmIssues = append(postConfirmIssues, "order stage progression")
	}

	log.Printf("Confirmed delivery of DN %s, signed by %s", dn.DNNumber, signedBy)

	if len(postConfirmIssues) > 0 {
		warning := fmt.Sprintf(
			"Delivery confirmed, but the order could not be fully updated (%s) — please refresh and verify the order.",
			strings.Join(postConfirmIssues, ", "),
		)
		return warning, nil
	}
	return "", nil
}

// updateOrderDeliveryStatus checks if all order items are delivered and updates order status
func (a *App) updateOrderDeliveryStatus(orderID string) error {
	// Get all delivery notes for this order
	deliveryNotes, err := a.GetDeliveryNotesByOrder(orderID)
	if err != nil {
		return err
	}

	// Check if all are delivered
	allDelivered := true
	for _, dn := range deliveryNotes {
		if dn.Status != "Delivered" {
			allDelivered = false
			break
		}
	}

	// Update order status if all delivered
	if allDelivered {
		var order Order
		if err := a.db.First(&order, "id = ?", orderID).Error; err != nil {
			return newError("ORDER_NOT_FOUND", "Order not found", err.Error())
		}

		if err := a.db.Model(&order).Update("status", "Delivered").Error; err != nil {
			return newError("DB_UPDATE_FAILED", "Failed to update order status", err.Error())
		}

		log.Printf("✅ Order %s marked as Delivered (all DNs delivered)", order.OrderNumber)
	}

	return nil
}

// =============================================================================
// DELIVERY NOTE CREATION FROM ORDER
// =============================================================================

// CreateDNFromOrder creates a delivery note from an order with specified items
func (a *App) CreateDNFromOrder(orderID string, items []DeliveryNoteItem) (DeliveryNote, error) {
	if err := a.requirePermission("delivery_notes:create"); err != nil {
		return DeliveryNote{}, err
	}
	if a.db == nil {
		return DeliveryNote{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get order with items
	var order Order
	if err := a.db.Preload("Items").First(&order, "id = ?", orderID).Error; err != nil {
		return DeliveryNote{}, newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	// Get existing delivery notes to calculate sequence
	existingDNs, err := a.GetDeliveryNotesByOrder(orderID)
	if err != nil {
		return DeliveryNote{}, err
	}

	// Calculate delivery status to validate quantities
	remainingQty, err := a.GetOrderDeliveryStatus(orderID)
	if err != nil {
		return DeliveryNote{}, err
	}

	// P1 FIX: Validate delivery quantities don't exceed order quantities
	orderItemMap := make(map[string]OrderItem)
	for _, orderItem := range order.Items {
		orderItemMap[orderItem.ID] = orderItem
	}

	for _, item := range items {
		// Check remaining quantity
		remaining, ok := remainingQty[item.OrderItemID]
		if !ok {
			return DeliveryNote{}, newError("INVALID_ITEM", fmt.Sprintf("Order item %s not found", item.OrderItemID), "")
		}

		// P1 FIX: Validate delivery quantity doesn't exceed remaining
		if item.QuantityDelivered > remaining {
			orderItem := orderItemMap[item.OrderItemID]
			return DeliveryNote{}, newError("QUANTITY_EXCEEDED",
				fmt.Sprintf("Cannot deliver %.2f units for %s - only %.2f remaining (ordered: %.2f)",
					item.QuantityDelivered, orderItem.ProductCode, remaining, orderItem.Quantity), "")
		}

		// Additional validation: delivery quantity must be positive
		if item.QuantityDelivered <= 0 {
			return DeliveryNote{}, newError("INVALID_QUANTITY",
				fmt.Sprintf("Delivery quantity must be positive, got: %.2f", item.QuantityDelivered), "")
		}
	}

	// Determine if this is a partial delivery
	isPartial := false
	allItemsFullyDelivered := true
	for _, item := range items {
		remaining := remainingQty[item.OrderItemID]
		afterThisDelivery := remaining - item.QuantityDelivered
		if afterThisDelivery > 0.001 { // tolerance for floating point
			allItemsFullyDelivered = false
			isPartial = true
			break
		}
	}

	// Create delivery note
	dn := DeliveryNote{
		OrderID:           orderID,
		CustomerID:        order.CustomerID,
		DeliveryDate:      time.Now(),
		Status:            "Prepared",
		IsPartialDelivery: isPartial,
		DeliverySequence:  len(existingDNs) + 1,
	}

	// Calculate total deliveries (estimated)
	if isPartial {
		dn.TotalDeliveries = 0 // Unknown until all delivered
	} else {
		dn.TotalDeliveries = len(existingDNs) + 1
	}

	// Populate items with calculated remaining quantities
	dn.Items = make([]DeliveryNoteItem, len(items))
	for i, item := range items {
		remaining := remainingQty[item.OrderItemID]
		dn.Items[i] = DeliveryNoteItem{
			OrderItemID:       item.OrderItemID,
			ProductID:         item.ProductID,
			ProductCode:       item.ProductCode,
			Description:       item.Description,
			QuantityOrdered:   item.QuantityOrdered,
			QuantityDelivered: item.QuantityDelivered,
			QuantityRemaining: remaining - item.QuantityDelivered,
		}
	}

	// Create the delivery note
	createdDN, err := a.CreateDeliveryNote(dn)
	if err != nil {
		return DeliveryNote{}, err
	}

	// P1 FIX: Update order status based on delivery completeness
	newOrderStatus := "Shipping"
	if isPartial {
		newOrderStatus = "Partially Delivered"
		log.Printf("📦 Partial delivery for Order %s - not all items delivered", order.OrderNumber)
	} else if allItemsFullyDelivered {
		// Check if this was the final delivery
		allDelivered := true
		for _, remaining := range remainingQty {
			if remaining > 0.001 {
				allDelivered = false
				break
			}
		}
		if allDelivered {
			newOrderStatus = "Delivered"
			log.Printf("✅ Full delivery complete for Order %s", order.OrderNumber)
		}
	}

	if err := a.db.Model(&order).Update("status", newOrderStatus).Error; err != nil {
		log.Printf("⚠️ Failed to update order status: %v", err)
		// Don't fail the creation, just log
	}

	log.Printf("✅ Created DN %s from Order %s (%d items, partial=%v, status=%s)",
		createdDN.DNNumber, order.OrderNumber, len(items), isPartial, newOrderStatus)
	return createdDN, nil
}

// GetOrderDeliveryStatus calculates remaining quantities for each order item
func (a *App) GetOrderDeliveryStatus(orderID string) (map[string]float64, error) {
	if err := a.requirePermission("delivery_notes:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get order items
	var order Order
	if err := a.db.Preload("Items").First(&order, "id = ?", orderID).Error; err != nil {
		return nil, newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	// Initialize remaining quantities from order
	remainingQty := make(map[string]float64)
	for _, item := range order.Items {
		remainingQty[item.ID] = item.Quantity
	}

	// Get all delivery notes for this order
	deliveryNotes, err := a.GetDeliveryNotesByOrder(orderID)
	if err != nil {
		return nil, err
	}

	// Subtract delivered quantities
	for _, dn := range deliveryNotes {
		for _, item := range dn.Items {
			if current, ok := remainingQty[item.OrderItemID]; ok {
				remainingQty[item.OrderItemID] = current - item.QuantityDelivered
			}
		}
	}

	log.Printf("📊 Order %s delivery status: %d items tracked", orderID, len(remainingQty))
	return remainingQty, nil
}

// GetOrderDeliveryStatusBatch returns per-order remaining-quantity maps (same
// shape as GetOrderDeliveryStatus, keyed by order ID) for multiple orders in
// ONE query set — not a loop over the singular method. B10-1: eliminates the
// Orders-list N+1 (was one GetOrderDeliveryStatus round-trip per order, each
// doing 2 DB queries).
func (a *App) GetOrderDeliveryStatusBatch(orderIDs []string) (map[string]map[string]float64, error) {
	if err := a.requirePermission("delivery_notes:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}
	if len(orderIDs) == 0 {
		return map[string]map[string]float64{}, nil
	}

	// One query: all order items across all requested orders.
	var orderItems []OrderItem
	if err := a.db.Where("order_id IN ?", orderIDs).Find(&orderItems).Error; err != nil {
		return nil, newError("ORDER_ITEMS_QUERY_FAILED", "Failed to retrieve order items", err.Error())
	}

	result := make(map[string]map[string]float64, len(orderIDs))
	for _, oid := range orderIDs {
		result[oid] = make(map[string]float64)
	}
	for _, item := range orderItems {
		orderRemaining, ok := result[item.OrderID]
		if !ok {
			orderRemaining = make(map[string]float64)
			result[item.OrderID] = orderRemaining
		}
		orderRemaining[item.ID] = item.Quantity
	}

	// One query: all delivery notes (with items) across all requested orders.
	var deliveryNotes []DeliveryNote
	if err := a.db.Preload("Items").Where("order_id IN ?", orderIDs).Find(&deliveryNotes).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve delivery notes", err.Error())
	}
	for _, dn := range deliveryNotes {
		orderRemaining, ok := result[dn.OrderID]
		if !ok {
			continue
		}
		for _, item := range dn.Items {
			if current, ok := orderRemaining[item.OrderItemID]; ok {
				orderRemaining[item.OrderItemID] = current - item.QuantityDelivered
			}
		}
	}

	log.Printf("📊 Batch order delivery status: %d orders", len(orderIDs))
	return result, nil
}

// =============================================================================
// P2 FIX: DELIVERY ROUTE OPTIMIZATION & PLANNING
// =============================================================================

// DeliveryPlanningItem represents a delivery with urgency and location info
type DeliveryPlanningItem struct {
	DeliveryNote
	CustomerArea   string  `json:"customer_area"`
	UrgencyScore   float64 `json:"urgency_score"` // 0-1, higher = more urgent
	DaysUntilDue   int     `json:"days_until_due"`
	OrderValue     float64 `json:"order_value"`
	IsHighPriority bool    `json:"is_high_priority"`
}

// GetDeliveriesByArea groups pending deliveries by customer area for route optimization
func (a *App) GetDeliveriesByArea(area string) ([]DeliveryPlanningItem, error) {
	if err := a.requirePermission("delivery_notes:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get pending/dispatched deliveries
	var deliveryNotes []DeliveryNote
	query := a.db.Preload("Items").Where("status IN (?)", []string{"Prepared", "Dispatched"})

	if area != "" && area != "All" {
		// Join with customers to filter by city/area
		query = query.Joins("JOIN customers ON delivery_notes.customer_id = customers.id").
			Where("customers.city = ?", area)
	}

	if err := query.Order("delivery_date ASC").Limit(500).Find(&deliveryNotes).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve deliveries", err.Error())
	}

	// Enrich with planning data
	planningItems := make([]DeliveryPlanningItem, len(deliveryNotes))
	for i, dn := range deliveryNotes {
		// Get customer area
		var customer CustomerMaster
		customerArea := "Unknown"
		if err := a.db.First(&customer, "id = ?", dn.CustomerID).Error; err == nil {
			customerArea = customer.City
		}

		// Get order value for prioritization
		var order Order
		orderValue := 0.0
		if err := a.db.First(&order, "id = ?", dn.OrderID).Error; err == nil {
			orderValue = order.TotalValueBHD
		}

		// Calculate urgency (based on delivery date)
		daysUntilDue := int(time.Until(dn.DeliveryDate).Hours() / 24)
		urgencyScore := 1.0
		if daysUntilDue <= 0 {
			urgencyScore = 1.0 // Overdue
		} else if daysUntilDue <= 2 {
			urgencyScore = 0.9 // Very urgent
		} else if daysUntilDue <= 5 {
			urgencyScore = 0.7 // Urgent
		} else {
			urgencyScore = 0.5 // Normal
		}

		// High priority customers
		isHighPriority := customer.CustomerGrade == "A" || orderValue > 10000.0

		planningItems[i] = DeliveryPlanningItem{
			DeliveryNote:   dn,
			CustomerArea:   customerArea,
			UrgencyScore:   urgencyScore,
			DaysUntilDue:   daysUntilDue,
			OrderValue:     orderValue,
			IsHighPriority: isHighPriority,
		}
	}

	log.Printf("📍 Retrieved %d deliveries for area: %s", len(planningItems), area)
	return planningItems, nil
}

// GetPendingDeliveries retrieves all pending deliveries sorted by urgency
func (a *App) GetPendingDeliveries() ([]DeliveryPlanningItem, error) {
	if err := a.requirePermission("delivery_notes:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Get all pending deliveries
	deliveries, err := a.GetDeliveriesByArea("All")
	if err != nil {
		return nil, err
	}

	// Sort by urgency score (descending) then by order value (descending)
	// Most urgent and highest value first
	for i := 0; i < len(deliveries)-1; i++ {
		for j := i + 1; j < len(deliveries); j++ {
			// Primary sort: urgency
			if deliveries[j].UrgencyScore > deliveries[i].UrgencyScore {
				deliveries[i], deliveries[j] = deliveries[j], deliveries[i]
			} else if deliveries[j].UrgencyScore == deliveries[i].UrgencyScore {
				// Secondary sort: order value
				if deliveries[j].OrderValue > deliveries[i].OrderValue {
					deliveries[i], deliveries[j] = deliveries[j], deliveries[i]
				}
			}
		}
	}

	log.Printf("📋 Retrieved %d pending deliveries sorted by urgency", len(deliveries))
	return deliveries, nil
}

// GetDeliveryAreaSummary returns delivery counts grouped by area
func (a *App) GetDeliveryAreaSummary() (map[string]int, error) {
	if err := a.requirePermission("delivery_notes:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Query delivery counts by customer city
	type AreaCount struct {
		Area  string
		Count int
	}

	var results []AreaCount
	err := a.db.Raw(`
		SELECT c.city AS area, COUNT(dn.id) AS count
		FROM delivery_notes dn
		JOIN customers c ON dn.customer_id = c.id
		WHERE dn.status IN ('Prepared', 'Dispatched')
		GROUP BY c.city
		ORDER BY count DESC
	`).Scan(&results).Error

	if err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to get area summary", err.Error())
	}

	// Convert to map
	summary := make(map[string]int)
	for _, r := range results {
		summary[r.Area] = r.Count
	}

	log.Printf("📊 Delivery area summary: %d areas with pending deliveries", len(summary))
	return summary, nil
}

// =============================================================================
// E1: PARTIAL DELIVERY ENHANCEMENT - FULFILLMENT DETAIL & ITEM-LEVEL CREATION
// =============================================================================

// OrderFulfillmentItem represents per-item delivery breakdown for an order
type OrderFulfillmentItem struct {
	OrderItemID            string  `json:"order_item_id"`
	ProductID              string  `json:"product_id"`
	ProductCode            string  `json:"product_code"`
	Description            string  `json:"description"`
	OrderedQty             float64 `json:"ordered_qty"`
	ShippedQty             float64 `json:"shipped_qty"`
	DeliveredQty           float64 `json:"delivered_qty"`
	InvoicedQty            float64 `json:"invoiced_qty"`
	RemainingQty           float64 `json:"remaining_qty"`
	RequiresSerialTracking bool    `json:"requires_serial_tracking"`
}

// OrderFulfillment represents the full delivery fulfillment status for an order
type OrderFulfillment struct {
	OrderID        string                 `json:"order_id"`
	OrderNumber    string                 `json:"order_number"`
	CustomerName   string                 `json:"customer_name"`
	Status         string                 `json:"status"`
	Items          []OrderFulfillmentItem `json:"items"`
	FullyDelivered bool                   `json:"fully_delivered"`
	FullyInvoiced  bool                   `json:"fully_invoiced"`
}

// GetOrderFulfillmentDetail returns per-item delivery breakdown for an order
func (a *App) GetOrderFulfillmentDetail(orderID string) (OrderFulfillment, error) {
	if err := a.requirePermission("orders:view"); err != nil {
		return OrderFulfillment{}, err
	}
	if a.db == nil {
		return OrderFulfillment{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Load order with items
	var order Order
	if err := a.db.Preload("Items").First(&order, "id = ?", orderID).Error; err != nil {
		return OrderFulfillment{}, newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	// Get all delivery notes for this order to calculate delivered quantities
	var deliveryNotes []DeliveryNote
	if err := a.db.Preload("Items").Where("order_id = ?", orderID).Find(&deliveryNotes).Error; err != nil {
		return OrderFulfillment{}, newError("DB_QUERY_FAILED", "Failed to retrieve delivery notes", err.Error())
	}

	// Calculate delivered quantities per order item from all DNs
	deliveredQtyMap := make(map[string]float64)
	for _, dn := range deliveryNotes {
		if strings.EqualFold(strings.TrimSpace(dn.Status), "Cancelled") {
			continue
		}
		for _, item := range dn.Items {
			deliveredQtyMap[item.OrderItemID] += item.QuantityDelivered
		}
	}

	productIDs := make([]string, 0, len(order.Items))
	for _, item := range order.Items {
		if strings.TrimSpace(item.ProductID) != "" {
			productIDs = append(productIDs, item.ProductID)
		}
	}
	serialTrackingByProduct := make(map[string]bool)
	if len(productIDs) > 0 {
		var products []ProductMaster
		if err := a.db.Where("id IN ?", productIDs).Find(&products).Error; err == nil {
			for _, product := range products {
				serialTrackingByProduct[product.ID] = product.RequiresSerialTracking
			}
		}
	}

	// Build fulfillment items
	fulfillment := OrderFulfillment{
		OrderID:      orderID,
		OrderNumber:  order.OrderNumber,
		CustomerName: order.CustomerName,
		Status:       order.Status,
	}

	fullyDelivered := true
	fullyInvoiced := true

	for _, item := range order.Items {
		deliveredQty := deliveredQtyMap[item.ID]
		remainingQty := item.Quantity - deliveredQty
		if remainingQty < 0.001 {
			remainingQty = 0
		}

		fi := OrderFulfillmentItem{
			OrderItemID:            item.ID,
			ProductID:              item.ProductID,
			ProductCode:            item.ProductCode,
			Description:            item.Description,
			OrderedQty:             item.Quantity,
			ShippedQty:             item.QuantityShipped,
			DeliveredQty:           deliveredQty,
			InvoicedQty:            item.QuantityInvoiced,
			RemainingQty:           remainingQty,
			RequiresSerialTracking: serialTrackingByProduct[item.ProductID],
		}
		fulfillment.Items = append(fulfillment.Items, fi)

		if remainingQty > 0.001 {
			fullyDelivered = false
		}
		if item.QuantityInvoiced < item.Quantity-0.001 {
			fullyInvoiced = false
		}
	}

	fulfillment.FullyDelivered = fullyDelivered
	fulfillment.FullyInvoiced = fullyInvoiced

	log.Printf("📊 Order %s fulfillment: %d items, fully_delivered=%v, fully_invoiced=%v",
		order.OrderNumber, len(fulfillment.Items), fullyDelivered, fullyInvoiced)
	return fulfillment, nil
}

// DeliveryNoteItemInput is a simplified input for creating delivery notes with per-item quantities
type DeliveryNoteItemInput struct {
	OrderItemID string  `json:"order_item_id"`
	ShipQty     float64 `json:"ship_qty"`
}

// CreateDeliveryNoteWithItems creates a delivery note with per-item ship quantities
// This is the preferred way to create a DN from an order with partial delivery support
func (a *App) CreateDeliveryNoteWithItems(orderID string, items []DeliveryNoteItemInput) (DeliveryNote, error) {
	if err := a.requirePermission("delivery_notes:create"); err != nil {
		return DeliveryNote{}, err
	}
	if a.db == nil {
		return DeliveryNote{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	if len(items) == 0 {
		return DeliveryNote{}, newError("NO_ITEMS", "At least one item is required", "")
	}

	// Load order with items
	var order Order
	if err := a.db.Preload("Items").First(&order, "id = ?", orderID).Error; err != nil {
		return DeliveryNote{}, newError("ORDER_NOT_FOUND", "Order not found", err.Error())
	}

	// Build order item lookup
	orderItemMap := make(map[string]OrderItem)
	for _, oi := range order.Items {
		orderItemMap[oi.ID] = oi
	}

	// Get remaining quantities
	remainingQty, err := a.GetOrderDeliveryStatus(orderID)
	if err != nil {
		return DeliveryNote{}, err
	}

	// Get existing DN count for sequence
	var existingDNCount int64
	a.db.Model(&DeliveryNote{}).Where("order_id = ?", orderID).Count(&existingDNCount)

	// Validate and build DN items
	var dnItems []DeliveryNoteItem
	allFullyDelivered := true

	for _, input := range items {
		if input.ShipQty <= 0 {
			return DeliveryNote{}, newError("INVALID_QUANTITY",
				fmt.Sprintf("Ship quantity must be positive, got: %.2f", input.ShipQty), "")
		}

		orderItem, ok := orderItemMap[input.OrderItemID]
		if !ok {
			return DeliveryNote{}, newError("INVALID_ITEM",
				fmt.Sprintf("Order item %s not found in order", input.OrderItemID), "")
		}

		remaining, ok := remainingQty[input.OrderItemID]
		if !ok {
			remaining = orderItem.Quantity
		}

		if input.ShipQty > remaining+0.001 {
			return DeliveryNote{}, newError("QUANTITY_EXCEEDED",
				fmt.Sprintf("Cannot ship %.2f units for %s - only %.2f remaining",
					input.ShipQty, orderItem.ProductCode, remaining), "")
		}

		afterDelivery := remaining - input.ShipQty
		if afterDelivery > 0.001 {
			allFullyDelivered = false
		}

		dnItems = append(dnItems, DeliveryNoteItem{
			OrderItemID:       input.OrderItemID,
			ProductID:         orderItem.ProductID,
			ProductCode:       orderItem.ProductCode,
			Description:       orderItem.Description,
			QuantityOrdered:   orderItem.Quantity,
			QuantityDelivered: input.ShipQty,
			QuantityRemaining: afterDelivery,
		})
	}

	// Check if ALL order items will be fully delivered (not just the items in this DN)
	// We need to check items NOT included in this DN too
	for _, orderItem := range order.Items {
		found := false
		for _, input := range items {
			if input.OrderItemID == orderItem.ID {
				found = true
				break
			}
		}
		if !found {
			// This item is not in the DN - check if it's already fully delivered
			remaining, ok := remainingQty[orderItem.ID]
			if !ok || remaining > 0.001 {
				allFullyDelivered = false
			}
		}
	}

	isPartial := !allFullyDelivered

	// Generate DN number
	dnNumber, err := a.GenerateDNNumber()
	if err != nil {
		return DeliveryNote{}, err
	}

	// Create delivery note
	dn := DeliveryNote{
		OrderID:           orderID,
		CustomerID:        order.CustomerID,
		DNNumber:          dnNumber,
		DeliveryDate:      time.Now(),
		Status:            "Prepared",
		IsPartialDelivery: isPartial,
		DeliverySequence:  int(existingDNCount) + 1,
		TotalDeliveries:   0, // Unknown until all delivered
		Items:             dnItems,
	}
	dn.CreatedBy = a.getCurrentUserID()

	if !isPartial {
		dn.TotalDeliveries = int(existingDNCount) + 1
	}

	// TOCTOU FIX: Wrap DN creation + quantity re-validation in a single transaction
	// to prevent concurrent over-delivery between GetOrderDeliveryStatus and Create
	err = a.db.Transaction(func(tx *gorm.DB) error {
		// Re-check remaining quantities inside transaction to prevent TOCTOU
		var existingDNs []DeliveryNote
		if err := tx.Where("order_id = ?", orderID).Preload("Items").Find(&existingDNs).Error; err != nil {
			return newError("DELIVERY_STATUS_CHECK_FAILED", "Failed to re-validate delivery quantities", err.Error())
		}
		deliveredQty := make(map[string]float64)
		for _, existingDN := range existingDNs {
			for _, item := range existingDN.Items {
				if item.OrderItemID != "" {
					deliveredQty[item.OrderItemID] += item.QuantityDelivered
				}
			}
		}
		for _, input := range items {
			oi, ok := orderItemMap[input.OrderItemID]
			if !ok {
				continue
			}
			currentRemaining := oi.Quantity - deliveredQty[input.OrderItemID]
			if input.ShipQty > currentRemaining+0.001 {
				return newError("QUANTITY_EXCEEDED",
					fmt.Sprintf("Concurrent delivery detected: cannot ship %.2f units for %s - only %.2f remaining",
						input.ShipQty, oi.ProductCode, currentRemaining), "")
			}
		}

		// Create DN inside transaction
		if err := tx.Create(&dn).Error; err != nil {
			return newError("DB_CREATE_FAILED", "Failed to create delivery note", err.Error())
		}

		// Update order status inside same transaction
		newStatus := "Shipping"
		if isPartial {
			newStatus = "Partially Delivered"
		} else {
			newStatus = "Shipped"
		}
		if err := tx.Model(&order).Update("status", newStatus).Error; err != nil {
			log.Printf("Warning: Failed to update order status: %v", err)
		}

		return nil
	})
	if err != nil {
		return DeliveryNote{}, err
	}

	log.Printf("Created DN %s from Order %s (%d items, partial=%v)",
		dn.DNNumber, order.OrderNumber, len(items), isPartial)
	return dn, nil
}

// =============================================================================
// CREATE DN WITH SERIAL ALLOCATION (Phase 23)
// =============================================================================

// DNItemInputWithSerials wraps DN item input with optional serial number allocation
type DNItemInputWithSerials struct {
	OrderItemID string   `json:"order_item_id"`
	ShipQty     float64  `json:"ship_qty"`
	SerialNos   []string `json:"serial_nos"`
}

// DeliveryNoteHeaderInput carries the DN header fields (delivery address,
// contact, transport) that the non-serial CreateDeliveryNote/
// CreateDeliveryNoteWithItems path already threads at creation time. B7c:
// CreateDNWithSerials accepts this so the frontend makes ONE create call
// instead of create-then-UpdateDeliveryNote patch (whose failure previously
// only console.warned).
type DeliveryNoteHeaderInput struct {
	DeliveryDate    time.Time `json:"delivery_date"`
	DeliveryAddress string    `json:"delivery_address"`
	ContactPerson   string    `json:"contact_person"`
	ContactPhone    string    `json:"contact_phone"`
	DriverName      string    `json:"driver_name"`
	VehicleNumber   string    `json:"vehicle_number"`
	TransportMethod string    `json:"transport_method"`
}

// CreateDNWithSerials creates a delivery note with serial number allocation
func (a *App) CreateDNWithSerials(orderID string, items []DNItemInputWithSerials, header DeliveryNoteHeaderInput) (DeliveryNote, error) {
	if err := a.requirePermission("delivery_notes:create"); err != nil {
		return DeliveryNote{}, err
	}
	if a.db == nil {
		return DeliveryNote{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Pre-validate serial count matches ship qty (availability is checked atomically in allocateSerialsToDN)
	// S6 FIX: Removed TOCTOU-prone availability pre-check — allocateSerialsToDN uses atomic UPDATE WHERE
	for _, item := range items {
		if len(item.SerialNos) > 0 {
			if float64(len(item.SerialNos)) != item.ShipQty {
				return DeliveryNote{}, newError("SERIAL_COUNT_MISMATCH",
					fmt.Sprintf("Serial count (%d) must match ship qty (%.0f) for order item %s",
						len(item.SerialNos), item.ShipQty, item.OrderItemID), "")
			}
		}
	}

	// Convert to plain DN items for existing logic
	var plainItems []DeliveryNoteItemInput
	for _, item := range items {
		plainItems = append(plainItems, DeliveryNoteItemInput{
			OrderItemID: item.OrderItemID,
			ShipQty:     item.ShipQty,
		})
	}

	// Create DN using existing logic
	dn, err := a.CreateDeliveryNoteWithItems(orderID, plainItems)
	if err != nil {
		return DeliveryNote{}, err
	}

	// P2 FIX: If post-creation steps fail, clean up the orphaned DN
	// Helper to soft-delete DN and release any allocated serials on failure
	cleanupDN := func(reason string) {
		log.Printf("Warning: Cleaning up orphaned DN %s: %s", dn.DNNumber, reason)
		// Release any serials that may have been allocated
		a.db.Model(&SerialNumber{}).
			Where("dn_number = ? AND status = ?", dn.DNNumber, "Reserved").
			Updates(map[string]any{
				"status":        "Available",
				"dn_number":     "",
				"dn_item_id":    "",
				"customer_id":   "",
				"customer_name": "",
			})
		// F FIX: also soft-delete the DeliveryNoteItem rows created by
		// CreateDeliveryNoteWithItems above — deleting only the DN header left
		// these orphaned (no parent, never cleaned up). Run both deletes even
		// if one fails, and log failures rather than losing them silently.
		if err := a.db.Where("delivery_note_id = ?", dn.ID).Delete(&DeliveryNoteItem{}).Error; err != nil {
			log.Printf("Warning: Failed to clean up DeliveryNoteItem rows for orphaned DN %s: %v", dn.DNNumber, err)
		}
		// Soft-delete the orphaned DN
		if err := a.db.Delete(&dn).Error; err != nil {
			log.Printf("Warning: Failed to soft-delete orphaned DN %s: %v", dn.DNNumber, err)
		}
	}

	// B7c: apply the DN header fields (delivery address/contact/transport) in
	// this same create call — no follow-up UpdateDeliveryNote patch, and a
	// failure here now surfaces as a real error (with cleanup) instead of a
	// silent console.warn.
	headerUpdates := map[string]any{}
	if !header.DeliveryDate.IsZero() {
		headerUpdates["delivery_date"] = header.DeliveryDate
	}
	if strings.TrimSpace(header.DeliveryAddress) != "" {
		headerUpdates["delivery_address"] = header.DeliveryAddress
	}
	if strings.TrimSpace(header.ContactPerson) != "" {
		headerUpdates["contact_person"] = header.ContactPerson
	}
	if strings.TrimSpace(header.ContactPhone) != "" {
		headerUpdates["contact_phone"] = header.ContactPhone
	}
	if strings.TrimSpace(header.DriverName) != "" {
		headerUpdates["driver_name"] = header.DriverName
	}
	if strings.TrimSpace(header.VehicleNumber) != "" {
		headerUpdates["vehicle_number"] = header.VehicleNumber
	}
	if strings.TrimSpace(header.TransportMethod) != "" {
		headerUpdates["transport_method"] = header.TransportMethod
	}
	if len(headerUpdates) > 0 {
		if err := a.db.Model(&dn).Updates(headerUpdates).Error; err != nil {
			cleanupDN("header field update failed")
			return DeliveryNote{}, newError("DB_UPDATE_FAILED", "Failed to set delivery note header fields", err.Error())
		}
		if v, ok := headerUpdates["delivery_date"]; ok {
			dn.DeliveryDate = v.(time.Time)
		}
		if v, ok := headerUpdates["delivery_address"]; ok {
			dn.DeliveryAddress = v.(string)
		}
		if v, ok := headerUpdates["contact_person"]; ok {
			dn.ContactPerson = v.(string)
		}
		if v, ok := headerUpdates["contact_phone"]; ok {
			dn.ContactPhone = v.(string)
		}
		if v, ok := headerUpdates["driver_name"]; ok {
			dn.DriverName = v.(string)
		}
		if v, ok := headerUpdates["vehicle_number"]; ok {
			dn.VehicleNumber = v.(string)
		}
		if v, ok := headerUpdates["transport_method"]; ok {
			dn.TransportMethod = v.(string)
		}
	}

	// Load customer for serial allocation
	var order Order
	if err := a.db.Where("id = ?", orderID).First(&order).Error; err != nil {
		cleanupDN("order lookup failed")
		return DeliveryNote{}, newError("ORDER_NOT_FOUND", "Order not found for serial allocation", err.Error())
	}
	var customer CustomerMaster
	if err := a.db.Where("id = ?", order.CustomerID).First(&customer).Error; err != nil {
		cleanupDN("customer lookup failed")
		return DeliveryNote{}, newError("CUSTOMER_NOT_FOUND", "Customer not found for serial allocation", err.Error())
	}

	// Allocate serials to DN items
	for i, item := range items {
		if len(item.SerialNos) == 0 {
			continue
		}
		if i >= len(dn.Items) {
			break
		}
		dnItem := dn.Items[i]

		if err := a.allocateSerialsToDN(
			dnItem.ID, dn.DNNumber,
			customer.ID, customer.BusinessName,
			item.SerialNos,
		); err != nil {
			cleanupDN(fmt.Sprintf("serial allocation failed for item %s: %v", dnItem.ID, err))
			return DeliveryNote{}, newError("SERIAL_ALLOCATION_FAILED",
				fmt.Sprintf("Failed to allocate serials to DN item %s: %v", dnItem.ID, err), "")
		}
	}

	log.Printf("Created DN %s with serial allocation from Order %s", dn.DNNumber, order.OrderNumber)
	return dn, nil
}

// confirmDeliveryAndUpdateOrder is called by ConfirmDeliveryNote to update QuantityShipped on order items
func (a *App) confirmDeliveryAndUpdateOrder(dn DeliveryNote) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		// Load the full DN with items if not already loaded
		var fullDN DeliveryNote
		if err := tx.Preload("Items").First(&fullDN, "id = ?", dn.ID).Error; err != nil {
			return newError("DN_NOT_FOUND", "Delivery note not found for order update", err.Error())
		}

		// Update QuantityShipped on each order item based on DN items
		for _, dnItem := range fullDN.Items {
			if dnItem.OrderItemID == "" {
				continue
			}

			var orderItem OrderItem
			if err := tx.First(&orderItem, "id = ?", dnItem.OrderItemID).Error; err != nil {
				log.Printf("Warning: Could not find order item %s for DN update: %v", dnItem.OrderItemID, err)
				continue
			}

			// Recalculate total shipped from ALL confirmed DNs for this order item
			var totalDelivered float64
			err := tx.Model(&DeliveryNoteItem{}).
				Joins("JOIN delivery_notes ON delivery_note_items.delivery_note_id = delivery_notes.id").
				Where("delivery_note_items.order_item_id = ? AND delivery_notes.status = ?", dnItem.OrderItemID, "Delivered").
				Select("COALESCE(SUM(delivery_note_items.quantity_delivered), 0)").
				Row().Scan(&totalDelivered)
			if err != nil {
				log.Printf("Warning: Could not calculate total delivered for order item %s: %v", dnItem.OrderItemID, err)
				continue
			}

			// Update QuantityShipped to match total confirmed deliveries
			if err := tx.Model(&orderItem).Update("quantity_shipped", totalDelivered).Error; err != nil {
				log.Printf("Warning: Failed to update QuantityShipped for order item %s: %v", dnItem.OrderItemID, err)
			}
		}

		// Now update order status based on delivery completeness
		var order Order
		if err := tx.Preload("Items").First(&order, "id = ?", fullDN.OrderID).Error; err != nil {
			return newError("ORDER_NOT_FOUND", "Order not found", err.Error())
		}

		allDelivered := true
		someDelivered := false
		for _, item := range order.Items {
			if item.Quantity > 0 {
				// Reload the item to get updated QuantityShipped
				var freshItem OrderItem
				if err := tx.First(&freshItem, "id = ?", item.ID).Error; err == nil {
					if freshItem.QuantityShipped >= freshItem.Quantity-0.001 {
						someDelivered = true
					} else {
						allDelivered = false
						if freshItem.QuantityShipped > 0.001 {
							someDelivered = true
						}
					}
				}
			}
		}

		newStatus := order.Status
		if allDelivered {
			newStatus = "Delivered"
		} else if someDelivered {
			newStatus = "Partially Delivered"
		}

		if newStatus != order.Status {
			if err := tx.Model(&order).Update("status", newStatus).Error; err != nil {
				log.Printf("Warning: Failed to update order status to %s: %v", newStatus, err)
			} else {
				log.Printf("Order %s status updated to %s after delivery confirmation", order.OrderNumber, newStatus)
			}
		}

		return nil
	})
}
