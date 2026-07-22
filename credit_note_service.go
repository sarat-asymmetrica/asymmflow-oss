package main

import (
	"fmt"
	"html"
	"log"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"

	"gorm.io/gorm/clause"
	"ph_holdings_app/pkg/documents/numbering"
)

// =============================================================================
// CREDIT NOTE SERVICE (Phase 23 - E-Invoicing)
//
// Credit notes are issued against invoices for returns, pricing corrections,
// or other adjustments. They reduce the outstanding balance on the original invoice.
//
// Workflow: Draft → Issued → Applied
// =============================================================================

// CreditNoteItemInput is the frontend-facing input for creating CN items
type CreditNoteItemInput struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Rate        float64 `json:"rate"`
	// HSNCode and UQC are additive India Rule-46 fields (India Spec-01 B4(c)),
	// blank/inert for a GCC credit note.
	HSNCode string `json:"hsn_code"`
	UQC     string `json:"uqc"`
}

// effectiveInvoiceVATPercent returns the VAT rate to apply to an invoice's
// derived documents (credit note, e-invoice, PDF). It preserves a genuine 0%
// (zero-rated/export — VATBHD == 0) while reconstructing the rate for legacy
// invoices that stored a VATBHD without a VATPercent (column default 0). This
// mirrors the recovery logic in UpdateCustomerInvoice and never silently forces
// a stored 0 up to the 10% default.
func effectiveInvoiceVATPercent(inv Invoice) float64 {
	if inv.VATPercent > 0 {
		return inv.VATPercent
	}
	if inv.SubtotalBHD > 0 && inv.VATBHD > 0 {
		return inv.VATBHD / inv.SubtotalBHD * 100.0 // legacy: reconstruct the applied rate
	}
	return 0 // VATBHD == 0 → genuinely zero-rated, nothing to tax
}

// CreateCreditNote creates a credit note against an existing invoice
func (a *App) CreateCreditNote(invoiceID, reason string, items []CreditNoteItemInput) (CreditNote, error) {
	if err := a.requirePermission("invoices:create"); err != nil {
		return CreditNote{}, err
	}
	if a.db == nil {
		return CreditNote{}, fmt.Errorf("database not initialized")
	}
	if len(items) == 0 {
		return CreditNote{}, fmt.Errorf("at least one item is required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 1000 {
		return CreditNote{}, fmt.Errorf("reason is required and must not exceed 1000 characters")
	}
	reason = html.EscapeString(reason)

	// Generate CN number (outside tx — sequence has its own locking). Peek
	// the invoice's division (plain read, no lock — the locked flow below
	// re-validates the invoice for the credit-note business rules) so India
	// divisions route through the per-GSTIN/FY series (India Spec-01 B4)
	// instead of the GCC CN- scheme.
	var divisionPeek struct{ Division string }
	if err := a.db.Model(&Invoice{}).Select("division").Where("id = ?", invoiceID).First(&divisionPeek).Error; err != nil {
		return CreditNote{}, fmt.Errorf("invoice not found: %w", err)
	}
	cnNumber, err := a.generateCreditNoteNumberForDivision(divisionPeek.Division)
	if err != nil {
		return CreditNote{}, fmt.Errorf("failed to generate CN number: %w", err)
	}

	// Build credit note items and calculate totals (pure computation, no DB)
	now := time.Now()
	var subtotal float64
	var cnItems []CreditNoteItem
	for i, input := range items {
		if input.Quantity <= 0 || input.Rate <= 0 {
			return CreditNote{}, fmt.Errorf("item %d: quantity and rate must be positive", i+1)
		}
		if utf8.RuneCountInString(input.Description) > 500 {
			return CreditNote{}, fmt.Errorf("credit note item description exceeds 500 character limit")
		}
		lineTotal := input.Quantity * input.Rate
		subtotal += lineTotal
		cnItems = append(cnItems, CreditNoteItem{
			Base:        Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
			LineNumber:  i + 1,
			Description: input.Description,
			Quantity:    input.Quantity,
			Rate:        input.Rate,
			TotalBHD:    lineTotal,
			HSNCode:     input.HSNCode,
			UQC:         input.UQC,
		})
	}

	// B10 FIX: Wrap invoice load + over-crediting check + CN insert in a single transaction
	// with row locking to prevent concurrent CreateCreditNote race conditions
	var cn CreditNote
	err = a.db.Transaction(func(tx *gorm.DB) error {
		// Lock the invoice row to serialize concurrent CN creation
		var invoice Invoice
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", invoiceID).First(&invoice).Error; err != nil {
			return fmt.Errorf("invoice not found: %w", err)
		}
		if invoice.Status == "Cancelled" || invoice.Status == "Void" {
			return fmt.Errorf("cannot create credit note for %s invoice", invoice.Status)
		}

		// Calculate VAT (match invoice rate). A posted invoice's rate is
		// authoritative — including an explicit 0% for zero-rated/export
		// invoices, which must not be coerced up to the 10% default. Legacy
		// invoices that stored a VATBHD without a VATPercent get the rate
		// reconstructed rather than shown as 0%.
		vatPercent := effectiveInvoiceVATPercent(invoice)
		vatBHD := subtotal * (vatPercent / 100.0)
		grandTotal := subtotal + vatBHD

		// P0-4 FIX: Account for ALL existing credit notes to prevent over-crediting
		var existingCNTotal float64
		if err := tx.Model(&CreditNote{}).
			Where("invoice_id = ? AND status IN ?", invoiceID, []string{"Draft", "Issued", "Applied"}).
			Select("COALESCE(SUM(grand_total_bhd), 0)").
			Scan(&existingCNTotal).Error; err != nil {
			return fmt.Errorf("failed to check existing credit notes: %w", err)
		}

		// Validate: total of all CNs (existing + new) cannot exceed invoice grand total
		if existingCNTotal+grandTotal > invoice.GrandTotalBHD+0.001 {
			return fmt.Errorf("total credit notes (%.3f existing + %.3f new = %.3f) would exceed invoice total (%.3f BHD)",
				existingCNTotal, grandTotal, existingCNTotal+grandTotal, invoice.GrandTotalBHD)
		}

		cn = CreditNote{
			Base:             Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
			CreditNoteNumber: cnNumber,
			CreditNoteDate:   now,
			InvoiceID:        invoiceID,
			InvoiceNumber:    invoice.InvoiceNumber,
			CustomerID:       invoice.CustomerID,
			CustomerName:     invoice.CustomerName,
			Reason:           reason,
			SubtotalBHD:      subtotal,
			VATBHD:           vatBHD,
			VATPercent:       vatPercent,
			GrandTotalBHD:    grandTotal,
			Status:           "Draft",
			Division:         normalizeDivisionName(invoice.Division),
			Items:            cnItems,
		}

		// Compute integrity hash (HMAC-SHA256 with salt, P1-4 fix)
		cn.CreditNoteHash = computeDocumentHMAC(cn.CreditNoteNumber, cn.CreditNoteDate.Format("2006-01-02"), cn.GrandTotalBHD, cn.VATBHD)

		// Set CN ID on items
		for i := range cn.Items {
			cn.Items[i].CreditNoteID = cn.ID
		}

		if err := tx.Create(&cn).Error; err != nil {
			return fmt.Errorf("failed to create credit note: %w", err)
		}

		log.Printf("✅ Created Credit Note %s against invoice %s (%.3f BHD)", cn.CreditNoteNumber, invoice.InvoiceNumber, cn.GrandTotalBHD)
		return nil
	})
	if err != nil {
		return CreditNote{}, err
	}
	return cn, nil
}

// ListCreditNotes retrieves credit notes with pagination
func (a *App) ListCreditNotes(limit, offset int) ([]CreditNote, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// P1-6: Enforce pagination bounds
	if limit <= 0 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	var creditNotes []CreditNote
	query := a.db.Preload("Items").Order("credit_note_date DESC").Limit(limit).Offset(offset)
	if err := query.Find(&creditNotes).Error; err != nil {
		return nil, fmt.Errorf("failed to list credit notes: %w", err)
	}
	return creditNotes, nil
}

// GetCreditNote retrieves a single credit note by ID
func (a *App) GetCreditNote(id string) (CreditNote, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return CreditNote{}, err
	}
	if a.db == nil {
		return CreditNote{}, fmt.Errorf("database not initialized")
	}

	var cn CreditNote
	if err := a.db.Preload("Items").Where("id = ?", id).First(&cn).Error; err != nil {
		return CreditNote{}, fmt.Errorf("credit note not found: %w", err)
	}
	return cn, nil
}

// IssueCreditNote transitions a Draft credit note to Issued status (required before Apply)
// P2 FIX: Atomic status transition prevents double-issue TOCTOU race condition
func (a *App) IssueCreditNote(id string) (CreditNote, error) {
	if err := a.requirePermission("invoices:create"); err != nil {
		return CreditNote{}, err
	}
	if a.db == nil {
		return CreditNote{}, fmt.Errorf("database not initialized")
	}

	// Atomic status transition: Draft → Issued (prevents double-issue race condition)
	result := a.db.Model(&CreditNote{}).
		Where("id = ? AND status = ?", id, "Draft").
		Updates(map[string]any{
			"status":           "Issued",
			"credit_note_date": time.Now(),
			"updated_at":       time.Now(),
		})
	if result.Error != nil {
		return CreditNote{}, fmt.Errorf("failed to issue credit note: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return CreditNote{}, fmt.Errorf("credit note not found or not in Draft status (may have been already issued)")
	}

	// Reload the credit note to return updated state
	var cn CreditNote
	if err := a.db.Preload("Items").Where("id = ?", id).First(&cn).Error; err != nil {
		return CreditNote{}, fmt.Errorf("failed to reload credit note after issuing: %w", err)
	}

	log.Printf("✅ Credit Note %s issued", cn.CreditNoteNumber)
	return cn, nil
}

// ApplyCreditNote applies a credit note to reduce the original invoice's outstanding balance
func (a *App) ApplyCreditNote(id string) error {
	if err := a.requirePermission("invoices:create"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	return a.db.Transaction(func(tx *gorm.DB) error {
		var cn CreditNote
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", id).First(&cn).Error; err != nil {
			return fmt.Errorf("credit note not found: %w", err)
		}
		if cn.Status == "Applied" {
			return fmt.Errorf("credit note %s is already applied", cn.CreditNoteNumber)
		}
		if cn.Status == "Draft" {
			return fmt.Errorf("credit note %s must be issued before it can be applied", cn.CreditNoteNumber)
		}

		// Load original invoice
		var invoice Invoice
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", cn.InvoiceID).First(&invoice).Error; err != nil {
			return fmt.Errorf("invoice not found: %w", err)
		}

		// P0-4 + P1 FIX: Block applying to non-actionable invoice statuses
		// Cancelled/Void = closed; Draft = not yet issued; Proforma = not a tax document
		blockedStatuses := map[string]bool{"Cancelled": true, "Void": true, "Draft": true, "Proforma": true}
		if blockedStatuses[invoice.Status] {
			return fmt.Errorf("cannot apply credit note to %s invoice %s", invoice.Status, invoice.InvoiceNumber)
		}

		// Reduce outstanding — error if it would go significantly negative
		newOutstanding := invoice.OutstandingBHD - cn.GrandTotalBHD
		if newOutstanding < -0.001 {
			return fmt.Errorf("credit note (%.3f) would reduce outstanding (%.3f) below zero", cn.GrandTotalBHD, invoice.OutstandingBHD)
		}
		if newOutstanding < 0 {
			newOutstanding = 0 // Tolerance for rounding
		}

		invoiceUpdates := map[string]any{
			"outstanding_bhd": newOutstanding,
		}
		if newOutstanding <= 0.001 {
			invoiceUpdates["status"] = "Paid"
		}

		if err := tx.Model(&invoice).Updates(invoiceUpdates).Error; err != nil {
			return fmt.Errorf("failed to update invoice: %w", err)
		}

		// Mark CN as applied
		now := time.Now()
		if err := tx.Model(&cn).Updates(map[string]any{
			"status":     "Applied",
			"applied_at": now,
		}).Error; err != nil {
			return fmt.Errorf("failed to update credit note: %w", err)
		}

		log.Printf("✅ Applied Credit Note %s: Invoice %s outstanding reduced to %.3f BHD",
			cn.CreditNoteNumber, invoice.InvoiceNumber, newOutstanding)
		return nil
	})
}

// GenerateCreditNoteNumber generates a sequential credit note number (CN-YYYYMMDD-NNNN)
func (a *App) GenerateCreditNoteNumber() (string, error) {
	if err := a.requirePermission("invoices:create"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	// Delegates to the promoted pkg/documents/numbering engine (Wave 2
	// Mission A). Format is byte-identical to the old inline implementation.
	return numbering.New(a.db).Next(numbering.Spec{
		Prefix:   "CN",
		Template: "CN-{date}-{seq}",
	}, time.Now())
}

// generateCreditNoteNumberForDivision routes credit-note numbering (India
// Spec-01 B4): a division carrying an India GST profile gets the per-GSTIN
// per-FY "CN/{fy}/{seq}" series, validated against Rule 46 before it is
// returned; every other (GCC) division keeps the unchanged CN-YYYYMMDD-NNNN
// scheme. No RBAC gate here — CreateCreditNote's own invoices:create guard
// already covers this call.
func (a *App) generateCreditNoteNumberForDivision(division string) (string, error) {
	profile := activeOverlay.Profile(activeOverlay.NormalizeDivisionName(division))
	if profile.India == nil {
		return numbering.New(a.db).Next(numbering.Spec{
			Prefix:   "CN",
			Template: "CN-{date}-{seq}",
		}, time.Now())
	}
	spec := indiaCreditNoteNumberSpec(profile.India.GSTIN, activeOverlay.FYStartMonthOrDefault())
	number, err := numbering.New(a.db).Next(spec, time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to generate India credit note number: %w", err)
	}
	if err := numbering.ValidateGSTSeriesNumber(number); err != nil {
		return "", fmt.Errorf("India credit note number failed Rule 46 validation: %w", err)
	}
	return number, nil
}

// GenerateCreditNotePDF creates a PDF for a credit note on company letterhead
func (a *App) GenerateCreditNotePDF(id string) (string, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	var cn CreditNote
	if err := a.db.Preload("Items").Where("id = ?", id).First(&cn).Error; err != nil {
		return "", fmt.Errorf("credit note not found: %w", err)
	}

	// Fetch customer details
	var customer CustomerMaster
	if err := a.db.First(&customer, "id = ?", cn.CustomerID).Error; err != nil {
		log.Printf("⚠️ Could not fetch customer details: %v", err)
	}

	division := a.resolveCreditNoteDivision(cn)
	profile := companyDocumentProfile(division)

	// India Spec-01 B4(c): a credit note against an India-mounted division
	// renders through the India GST layout (GSTIN seller block, referenced
	// original invoice, HSN/UQC/tax-split table). GCC divisions fall through
	// unchanged below.
	if profile.India != nil {
		return a.generateIndiaCreditNotePDF(cn, customer, profile)
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 30)
	pdf.SetTopMargin(50)
	pdf.SetLeftMargin(10)
	pdf.SetRightMargin(10)

	// Set header with letterhead
	pdf.SetHeaderFuncMode(func() {
		a.applyLetterheadForDivision(pdf, profile.Division)
	}, true)

	pdf.AddPage()

	// Title
	pdf.SetY(50)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(180, 40, 40)
	pdf.CellFormat(0, 8, "CREDIT NOTE", "", 0, "C", false, 0, "")
	pdf.Ln(10)

	// Seller info (left)
	headerY := pdf.GetY()
	pdf.SetXY(10, headerY)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(90, 4, sanitizeForPDF(profile.LegalName))
	pdf.Ln(4)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(60, 60, 60)
	for _, line := range profile.AddressLines {
		pdf.SetX(10)
		pdf.Cell(90, 4, sanitizeForPDF(line))
		pdf.Ln(4)
	}
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(90, 4, "TRN: "+sanitizeForPDF(profile.VATNumber))

	// CN metadata (right)
	rightX := 120.0
	pdf.SetXY(rightX, headerY)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(30, 4, "CN Number:")
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(50, 4, cn.CreditNoteNumber)
	pdf.SetXY(rightX, headerY+5)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(30, 4, "CN Date:")
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(50, 4, cn.CreditNoteDate.Format("02-Jan-2006"))
	pdf.SetXY(rightX, headerY+10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(30, 4, "Invoice Ref:")
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(50, 4, cn.InvoiceNumber)
	pdf.SetXY(rightX, headerY+15)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(30, 4, "Status:")
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(50, 4, cn.Status)

	// Buyer info
	pdf.SetY(headerY + 25)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(20, 4, "Buyer:")
	pdf.Ln(4)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(60, 60, 60)
	pdf.Cell(0, 4, sanitizeForPDF(cn.CustomerName))
	pdf.Ln(4)
	if customer.AddressLine1 != "" {
		pdf.SetX(10)
		pdf.Cell(0, 4, sanitizeForPDF(customer.AddressLine1))
		pdf.Ln(4)
	}
	if customer.TRN != "" {
		pdf.SetX(10)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.Cell(0, 4, "TRN: "+customer.TRN)
		pdf.Ln(4)
	}

	// Reason
	pdf.Ln(4)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(20, 4, "Reason:")
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(60, 60, 60)
	pdf.MultiCell(170, 4, sanitizeForPDF(cn.Reason), "", "", false)
	pdf.Ln(4)

	// Line items table
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(29, 29, 31)
	pdf.SetX(10)
	pdf.CellFormat(10, 6, "Sl", "1", 0, "C", true, 0, "")
	pdf.CellFormat(90, 6, "Description", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 6, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 6, "Rate (BHD)", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 6, "Total (BHD)", "1", 0, "C", true, 0, "")
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(29, 29, 31)
	for _, item := range cn.Items {
		pdf.SetX(10)
		pdf.CellFormat(10, 5, fmt.Sprintf("%d", item.LineNumber), "1", 0, "C", false, 0, "")
		desc := sanitizeForPDF(item.Description)
		descRunes := []rune(desc)
		if len(descRunes) > 60 {
			desc = string(descRunes[:57]) + "..."
		}
		pdf.CellFormat(90, 5, desc, "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 5, fmt.Sprintf("%.0f", item.Quantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 5, fmt.Sprintf("%.3f", item.Rate), "1", 0, "R", false, 0, "")
		pdf.CellFormat(40, 5, fmt.Sprintf("%.3f", item.TotalBHD), "1", 0, "R", false, 0, "")
		pdf.Ln(5)
	}

	// Totals
	pdf.Ln(4)
	pdf.SetX(130)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(35, 5, "Subtotal:")
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(30, 5, fmt.Sprintf("%.3f BHD", cn.SubtotalBHD))
	pdf.Ln(5)

	pdf.SetX(130)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(35, 5, fmt.Sprintf("VAT (%.0f%%):", cn.VATPercent))
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(30, 5, fmt.Sprintf("%.3f BHD", cn.VATBHD))
	pdf.Ln(5)

	pdf.SetX(130)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(180, 40, 40)
	pdf.Cell(35, 6, "Credit Total:")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(30, 6, fmt.Sprintf("%.3f BHD", cn.GrandTotalBHD))
	pdf.Ln(8)

	// Declaration
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(30, 5, "Declaration:")
	pdf.Ln(5)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(60, 60, 60)
	pdf.MultiCell(90, 4, "We declare that this credit note shows the actual adjustment against the referenced invoice and that all particulars are true and correct.", "", "", false)

	// Prepared-by signature block from the configured signature list. The
	// credit note carries no explicit signer field, so the current session's
	// display name is resolved and matched against the overlay signature blocks
	// (falling back to the company-level block stamped with that name).
	pdf.SetXY(130, pdf.GetY()-15)
	pdf.SetTextColor(29, 29, 31)
	signerName := a.resolveDocumentSignerName(a.getCurrentUserDisplayName())
	signatureBlock := a.resolvePreparedBySignatureBlock(signerName)
	drawSignaturePDFLines(pdf, 130, pdf.GetY(), 60, 3.5, 6.6, signatureBlock, false)

	// Save
	paths := a.getAppPaths()
	if paths == nil {
		return "", fmt.Errorf("application paths not available")
	}
	cleanNum := strings.ReplaceAll(cn.CreditNoteNumber, "/", "_")
	cleanNum = strings.ReplaceAll(cleanNum, "\\", "_")
	filename := fmt.Sprintf("CreditNote_%s.pdf", filepath.Base(cleanNum))

	docYear := cn.CreditNoteDate.Year()
	if docYear <= 0 {
		docYear = time.Now().Year()
	}
	exportDir := a.getExportDir("customer", customer.BusinessName, "MISC", docYear)
	filePath := filepath.Join(exportDir, filename)

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", fmt.Errorf("failed to save credit note PDF: %w", err)
	}

	log.Printf("✅ Credit Note PDF generated: %s", filePath)
	return filePath, nil
}
