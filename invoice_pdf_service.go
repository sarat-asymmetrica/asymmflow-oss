// ═══════════════════════════════════════════════════════════════════════════
// INVOICE PDF GENERATION SERVICE
//
// MISSION: Generate professional customer and supplier invoices on Acme Instrumentation letterhead
//          matching the company's established format and branding
//
// FORMAT:
//   - A4 Portrait (210x297mm)
//   - Acme Instrumentation letterhead background (full page template)
//   - Invoice header: Invoice No, Date, PO Ref, Payment Terms
//   - Buyer block: Business Name, Address, TRN
//   - Line items table: Sl No | Description | Qty | Rate | VAT 10% | Total
//   - Totals section: Subtotal, VAT, Grand Total
//   - Amount in words (BHD and Fils)
//
// OUTPUTS:
//   - Customer Invoices: exports/invoices/Invoice_<number>.pdf
//   - Supplier Invoices: exports/invoices/SupplierInv_<number>.pdf
//
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// FieldVisibilitySettings controls which fields appear on invoice PDF
type FieldVisibilitySettings struct {
	ShowEquipment     bool `json:"show_equipment"`
	ShowSpecification bool `json:"show_specification"`
	ShowDetailedDesc  bool `json:"show_detailed_desc"`
	ShowFOB           bool `json:"show_fob"`
	ShowFreight       bool `json:"show_freight"`
	ShowCost          bool `json:"show_cost"`
	ShowMargin        bool `json:"show_margin"`
	ShowContact       bool `json:"show_contact"`
	ShowRFQ           bool `json:"show_rfq"`
	ShowCurrency      bool `json:"show_currency"`
	ShowCountryOrigin bool `json:"show_country_origin"`
	ShowDeliveryWeeks bool `json:"show_delivery_weeks"`
}

// parseFieldVisibility parses the JSON field visibility settings
func parseFieldVisibility(jsonStr string) FieldVisibilitySettings {
	// Default settings: show customer-facing fields, hide internal cost data
	defaults := FieldVisibilitySettings{
		ShowEquipment:     true,
		ShowSpecification: true,
		ShowDetailedDesc:  false,
		ShowFOB:           false,
		ShowFreight:       false,
		ShowCost:          false,
		ShowMargin:        false,
		ShowContact:       true,
		ShowRFQ:           true,
		ShowCurrency:      false,
		ShowCountryOrigin: true,
		ShowDeliveryWeeks: true,
	}

	if jsonStr == "" {
		return defaults
	}

	var settings FieldVisibilitySettings
	if err := json.Unmarshal([]byte(jsonStr), &settings); err != nil {
		log.Printf("⚠️ Failed to parse field visibility JSON, using defaults: %v", err)
		return defaults
	}

	return settings
}

// ============================================================================
// CUSTOMER INVOICE PDF GENERATION
// ============================================================================

// GenerateInvoicePDF creates a professional customer invoice on Acme Instrumentation letterhead
// FORMAT: Matches client's Tally-style invoice with full header metadata, bank details,
// VAT breakdown, declaration and signature section
func (a *App) GenerateInvoicePDF(invoiceID string) (string, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return "", err
	}

	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	log.Printf("📄 Generating customer invoice PDF: invoiceID=%s", invoiceID)

	// 1. Fetch invoice with items from database
	var invoice Invoice
	if err := a.db.Preload("Items").First(&invoice, "id = ?", invoiceID).Error; err != nil {
		return "", fmt.Errorf("failed to fetch invoice: %w", err)
	}

	// 2. Fetch customer details for address/TRN
	var customer CustomerMaster
	if err := a.db.First(&customer, "id = ?", invoice.CustomerID).Error; err != nil {
		log.Printf("⚠️ Could not fetch customer details: %v", err)
	}

	profile := companyDocumentProfile(invoice.Division)

	// 3. Fetch bank accounts for bank details section.
	// Use the document-context fetch (unguarded + auto-seeding) so an operator
	// who legitimately passed the invoice document gate but lacks finance:view
	// still gets an invoice with payable bank details rather than an error/blank.
	bankAccounts, err := a.getActiveBankAccountsForDocuments()
	if err != nil {
		log.Printf("⚠️ Could not fetch bank accounts: %v", err)
	}

	// 4. Parse field visibility settings with NULL fallback
	visibilityJSON := invoice.FieldVisibility
	if visibilityJSON == "" {
		visibilityJSON = "{}"
	}
	visibility := parseFieldVisibility(visibilityJSON)
	log.Printf("📄 Field visibility: %+v", visibility)

	// India Spec-01 B4 (R-A3-1): a division carrying an India GST profile
	// renders through the India Rule-46 pipeline (tax invoice, or — enforced
	// regardless of caller intent, G6 — a Bill of Supply for a composition
	// division) instead of the GCC layout below. GCC divisions
	// (profile.India == nil) fall through unchanged: byte-identical output,
	// the hard boundary this wave.
	if profile.India != nil {
		return a.generateIndiaInvoicePDF(invoice, profile, customer, bankAccounts)
	}

	// 5. Create PDF with letterhead
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 30)
	// C-a: trim the empty band below the letterhead (was 50mm). The letterhead
	// header graphic clears well above this, so starting content at 40mm removes
	// ~10mm of dead space without overlapping the template.
	pdf.SetTopMargin(40)
	pdf.SetLeftMargin(10)
	pdf.SetRightMargin(10)

	// Set header function for letterhead on all pages
	pdf.SetHeaderFuncMode(func() {
		a.applyLetterheadForDivision(pdf, profile.Division)
	}, true)

	pdf.AddPage()

	// ========================================================================
	// SECTION 1: INVOICE TITLE (Proforma or Tax Invoice)
	// ========================================================================
	// C-a: start the title at 40mm (was 50mm) to match the reduced top margin.
	pdf.SetY(40)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(29, 29, 31)
	invoiceTitle := "TAX INVOICE"
	if invoice.Status == "Proforma" {
		invoiceTitle = "PROFORMA INVOICE"
	}
	pdf.CellFormat(0, 8, invoiceTitle, "", 0, "C", false, 0, "")
	pdf.Ln(10)

	// ========================================================================
	// SECTION 2: TWO-COLUMN HEADER (Seller Info + Invoice Metadata)
	// ========================================================================
	headerY := pdf.GetY()

	// LEFT COLUMN: Seller Information
	pdf.SetXY(10, headerY)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(90, 4, profile.LegalName)
	pdf.Ln(4)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(60, 60, 60)
	for _, line := range profile.AddressLines {
		pdf.Cell(90, 4, line)
		pdf.Ln(4)
		pdf.SetX(10)
	}
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(90, 4, "TRN: "+profile.VATNumber)

	// RIGHT COLUMN: Invoice Metadata (11 fields in rows)
	rightX := 105.0
	metaY := headerY
	rowH := 4.0
	labelW := 32.0
	valueW := 30.0

	// Row 1: Invoice No / Dated
	pdf.SetXY(rightX, metaY)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(labelW, rowH, "Invoice No.")
	pdf.Cell(valueW, rowH, "Dated")
	pdf.Ln(rowH)
	pdf.SetX(rightX)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(labelW, rowH, sanitizeForPDF(invoice.InvoiceNumber))
	pdf.Cell(valueW, rowH, invoice.InvoiceDate.Format("02-Jan-2006"))
	metaY += rowH * 2

	// Row 2: Delivery Note / Mode of Payment
	pdf.SetXY(rightX, metaY)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(labelW, rowH, "Delivery Note")
	pdf.Cell(valueW, rowH, "Mode/Terms of Payment")
	pdf.Ln(rowH)
	pdf.SetX(rightX)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(29, 29, 31)
	deliveryNote := invoice.DeliveryNoteRef
	if deliveryNote == "" {
		deliveryNote = invoice.DeliveryNoteNumber
	}
	modeOfPayment := invoice.ModeOfPayment
	if modeOfPayment == "" {
		modeOfPayment = invoice.PaymentTerms
	}
	pdf.Cell(labelW, rowH, sanitizeForPDF(deliveryNote))
	pdf.Cell(valueW, rowH, sanitizeForPDF(modeOfPayment))
	metaY += rowH * 2

	// Row 3: Supplier's Ref / Other Reference(s)
	pdf.SetXY(rightX, metaY)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(labelW, rowH, "Supplier's Ref.")
	pdf.Cell(valueW, rowH, "Other Reference(s)")
	pdf.Ln(rowH)
	pdf.SetX(rightX)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(29, 29, 31)
	suppliersRef := invoice.SuppliersRef
	if suppliersRef == "" {
		suppliersRef = invoice.CustomerReference
	}
	// C-c: a long RFQ / reference must stay inside its labelW cell instead of
	// spilling over into the "Other Reference(s)" column. gofpdf's Cell does not
	// clip, so truncate to the column width at the current font.
	pdf.Cell(labelW, rowH, truncatePDFTextToWidth(pdf, sanitizeForPDF(suppliersRef), labelW-1))
	pdf.Cell(valueW, rowH, sanitizeForPDF(invoice.OtherReferences))
	metaY += rowH * 2

	// Row 4: Buyer's Order No. / Dated
	pdf.SetXY(rightX, metaY)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(labelW, rowH, "Buyer's Order No.")
	pdf.Cell(valueW, rowH, "Dated")
	pdf.Ln(rowH)
	pdf.SetX(rightX)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(29, 29, 31)
	buyersOrderNum := invoice.BuyersOrderNumber
	if buyersOrderNum == "" {
		buyersOrderNum = invoice.CustomerPONumber
	}
	buyersOrderDate := ""
	if invoice.BuyersOrderDate != nil {
		buyersOrderDate = invoice.BuyersOrderDate.Format("02-Jan-2006")
	}
	pdf.Cell(labelW, rowH, sanitizeForPDF(buyersOrderNum))
	pdf.Cell(valueW, rowH, buyersOrderDate)
	metaY += rowH * 2

	// Row 5: Despatch Doc No. / Delivery Note Date
	pdf.SetXY(rightX, metaY)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(labelW, rowH, "Despatch Document No.")
	pdf.Cell(valueW, rowH, "Delivery Note Date")
	pdf.Ln(rowH)
	pdf.SetX(rightX)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(29, 29, 31)
	despatchDoc := invoice.DespatchDocumentNo
	deliveryNoteDate := ""
	if invoice.DeliveryNoteDate != nil {
		deliveryNoteDate = invoice.DeliveryNoteDate.Format("02-Jan-2006")
	}
	pdf.Cell(labelW, rowH, sanitizeForPDF(despatchDoc))
	pdf.Cell(valueW, rowH, deliveryNoteDate)
	metaY += rowH * 2

	// Row 6: Despatched through / Destination
	pdf.SetXY(rightX, metaY)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(labelW, rowH, "Despatched through")
	pdf.Cell(valueW, rowH, "Destination")
	pdf.Ln(rowH)
	pdf.SetX(rightX)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(29, 29, 31)
	despatchedThrough := invoice.DespatchedThrough
	if despatchedThrough == "" {
		despatchedThrough = "Direct"
	}
	destination := invoice.Destination
	if destination == "" {
		destination = "Bahrain"
	}
	pdf.Cell(labelW, rowH, sanitizeForPDF(despatchedThrough))
	pdf.Cell(valueW, rowH, sanitizeForPDF(destination))

	// Move below header section
	pdf.SetY(metaY + rowH*2 + 2)
	pdf.Ln(4)

	// ========================================================================
	// SECTION 3: BUYER INFORMATION
	// ========================================================================
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(25, 5, "Buyer:")
	pdf.Ln(5)

	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.Cell(0, 5, sanitizeForPDF(customer.BusinessName))
	pdf.Ln(5)

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(60, 60, 60)
	if customer.AddressLine1 == "" && customer.City == "" && customer.Country == "" {
		// C-b: the customer master has no address — fall back to the address
		// captured on the invoice itself (carried from the order/offer attention
		// block) so the buyer block is not just a bare name.
		if invoice.AttentionCompany != "" {
			pdf.SetX(10)
			pdf.MultiCell(90, 4, sanitizeForPDF(invoice.AttentionCompany), "", "", false)
		}
		if invoice.AttentionAddress != "" {
			pdf.SetX(10)
			pdf.MultiCell(90, 4, sanitizeForPDF(invoice.AttentionAddress), "", "", false)
		}
	} else {
		if customer.AddressLine1 != "" {
			pdf.SetX(10)
			pdf.Cell(0, 4, sanitizeForPDF(customer.AddressLine1))
			pdf.Ln(4)
		}
		if customer.City != "" {
			pdf.SetX(10)
			pdf.Cell(0, 4, sanitizeForPDF(customer.City))
			pdf.Ln(4)
		}
		if customer.Country != "" {
			pdf.SetX(10)
			pdf.Cell(20, 4, "Country:")
			pdf.Cell(0, 4, sanitizeForPDF(customer.Country))
			pdf.Ln(4)
		}
	}
	if customer.TRN != "" {
		pdf.SetX(10)
		pdf.Cell(20, 4, "TRN:")
		pdf.SetFont("Helvetica", "B", 8)
		pdf.Cell(0, 4, sanitizeForPDF(customer.TRN))
		pdf.Ln(4)
		pdf.SetFont("Helvetica", "", 8)
	}
	// Place of Supply
	placeOfSupply := invoice.PlaceOfSupply
	if placeOfSupply == "" {
		placeOfSupply = "Kingdom of Bahrain"
	}
	pdf.SetX(10)
	pdf.Cell(30, 4, "Place of supply:")
	pdf.Cell(0, 4, sanitizeForPDF(placeOfSupply))
	pdf.Ln(6)

	// ========================================================================
	// SECTION 4: LINE ITEMS TABLE (Enhanced 10-column format)
	// ========================================================================
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(29, 29, 31)

	// Column widths for 10-column table (total = 190mm)
	colSl := 8.0
	colDesc := 52.0
	colQty := 12.0
	colRate := 18.0
	colPer := 10.0
	colDisc := 12.0
	colAmtExcl := 20.0
	colVatPct := 10.0
	colTaxVal := 20.0
	colVat := 14.0
	_ = colVat // Used in header rendering

	// Header row
	pdf.SetX(10)
	pdf.CellFormat(colSl, 6, "Sl", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colDesc, 6, "Description of Goods", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colQty, 6, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colRate, 6, "Rate", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colPer, 6, "per", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colDisc, 6, "Disc.%", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colAmtExcl, 6, "Amt Excl.VAT", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colVatPct, 6, "VAT%", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colTaxVal, 6, "Taxable Val", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colVat, 6, "VAT", "1", 0, "C", true, 0, "")
	pdf.Ln(6)

	// Data rows
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(29, 29, 31)

	totalQty := 0.0

	// P1 FIX: Use stored values from invoice instead of recalculating
	// This ensures PDF matches the database values exactly
	totalAmtExclVAT := invoice.SubtotalBHD
	totalVAT := invoice.VATBHD
	// Display the invoice's actual VAT rate, including an explicit 0% for
	// zero-rated/export invoices — a stored 0 (with no VAT amount) is a
	// legitimate rate, not a missing one. Legacy invoices that carry a VATBHD
	// without a stored VATPercent get the rate reconstructed rather than shown
	// as 0%.
	vatPct := effectiveInvoiceVATPercent(invoice)

	for i, item := range invoice.Items {
		lineNum := i + 1

		// Build description
		descParts := []string{}
		if visibility.ShowEquipment && item.Equipment != "" {
			descParts = append(descParts, item.Equipment)
		}
		if item.Model != "" {
			descParts = append(descParts, item.Model)
		} else if item.ProductCode != "" {
			descParts = append(descParts, item.ProductCode)
		}
		if item.Description != "" && len(descParts) == 0 {
			descParts = append(descParts, item.Description)
		}
		description := sanitizeForPDF(strings.Join(descParts, " - "))
		if len(description) > 45 {
			description = description[:42] + "..."
		}

		// Calculations - display line item details
		amtExclVAT := item.Rate * item.Quantity
		discPct := invoice.DiscountPercent
		taxableVal := amtExclVAT * (1 - discPct/100)

		// P1 FIX: Display VAT using stored percentage (for display only, totals come from DB)
		vatAmt := taxableVal * (vatPct / 100)

		totalQty += item.Quantity
		// Note: totalAmtExclVAT uses stored invoice.SubtotalBHD, not accumulated here

		// Render row
		pdf.SetX(10)
		pdf.CellFormat(colSl, 5, fmt.Sprintf("%d", lineNum), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colDesc, 5, description, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colQty, 5, fmt.Sprintf("%.0f", item.Quantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colRate, 5, fmt.Sprintf("%.3f", item.Rate), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colPer, 5, "nos", "1", 0, "C", false, 0, "")
		pdf.CellFormat(colDisc, 5, fmt.Sprintf("%.0f%%", discPct), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colAmtExcl, 5, fmt.Sprintf("%.3f", amtExclVAT), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colVatPct, 5, fmt.Sprintf("%.0f%%", vatPct), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colTaxVal, 5, fmt.Sprintf("%.3f", taxableVal), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colVat, 5, fmt.Sprintf("%.3f", vatAmt), "1", 0, "R", false, 0, "")
		pdf.Ln(5)

		// Phase 23: Render serial numbers below item description (if any)
		if item.ProductID != "" {
			serials, _ := a.GetSerialsForInvoiceItem(invoiceID, item.ProductID)
			if len(serials) > 0 {
				serialNos := make([]string, 0, len(serials))
				for _, s := range serials {
					serialNos = append(serialNos, s.SerialNo)
				}
				pdf.SetX(10 + colSl)
				pdf.SetFont("Helvetica", "I", 6)
				pdf.SetTextColor(100, 100, 100)
				serialLine := "S/N: " + strings.Join(serialNos, ", ")
				if len(serialLine) > 90 {
					serialLine = serialLine[:87] + "..."
				}
				pdf.CellFormat(colDesc+colQty+colRate+colPer+colDisc+colAmtExcl+colVatPct+colTaxVal+colVat, 4, serialLine, "", 0, "L", false, 0, "")
				pdf.Ln(4)
				pdf.SetFont("Helvetica", "", 7)
				pdf.SetTextColor(29, 29, 31)
			}
		}
	}

	// P1 FIX: Use stored grand total from database to ensure exact match
	grandTotal := invoice.GrandTotalBHD
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetX(10)
	pdf.CellFormat(colSl+colDesc, 5, "", "1", 0, "C", false, 0, "")
	pdf.CellFormat(colQty, 5, fmt.Sprintf("%.0f", totalQty), "1", 0, "C", false, 0, "")
	pdf.CellFormat(colRate+colPer+colDisc, 5, "TOTAL", "1", 0, "R", false, 0, "")
	pdf.CellFormat(colAmtExcl, 5, fmt.Sprintf("%.3f", totalAmtExclVAT), "1", 0, "R", false, 0, "")
	pdf.CellFormat(colVatPct, 5, "", "1", 0, "C", false, 0, "")
	pdf.CellFormat(colTaxVal, 5, fmt.Sprintf("%.3f", totalAmtExclVAT), "1", 0, "R", false, 0, "")
	pdf.CellFormat(colVat, 5, fmt.Sprintf("%.3f", totalVAT), "1", 0, "R", false, 0, "")
	pdf.Ln(6)

	// ========================================================================
	// SECTION 5: TOTALS SUMMARY (Right side)
	// ========================================================================
	totalsY := pdf.GetY()
	pdf.SetXY(130, totalsY)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(35, 5, "Total Excl. VAT:")
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(30, 5, fmt.Sprintf("%.3f BHD", totalAmtExclVAT))
	pdf.Ln(5)

	pdf.SetX(130)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(35, 5, "Output VAT:")
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(30, 5, fmt.Sprintf("%.3f BHD", totalVAT))
	pdf.Ln(5)

	pdf.SetX(130)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(35, 6, "Grand Total:")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(30, 6, fmt.Sprintf("%.3f BHD", grandTotal))
	pdf.Ln(8)

	// ========================================================================
	// SECTION 6: AMOUNT IN WORDS
	// ========================================================================
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(45, 5, "Amount Chargeable (in words):")
	pdf.Ln(5)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(60, 60, 60)
	amountWords := amountInWords(grandTotal)
	pdf.MultiCell(120, 4, amountWords, "", "", false)
	pdf.Ln(2)

	// VAT in words
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(35, 5, "VAT Amount (in words):")
	pdf.Ln(5)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(60, 60, 60)
	vatWords := amountInWords(totalVAT)
	pdf.MultiCell(120, 4, vatWords, "", "", false)
	pdf.Ln(4)

	// ========================================================================
	// SECTION 7: VAT BREAKDOWN TABLE (Right side)
	// ========================================================================
	vatTableY := pdf.GetY() - 30
	if vatTableY < totalsY+25 {
		vatTableY = totalsY + 25
	}
	pdf.SetXY(140, vatTableY)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(29, 29, 31)
	pdf.CellFormat(15, 5, "VAT%", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 5, "Assess. Value", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 5, "Tax Amount", "1", 0, "C", true, 0, "")
	pdf.Ln(5)

	pdf.SetX(140)
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(29, 29, 31)
	// P1 FIX: Display stored VAT percentage, not hardcoded 10%
	pdf.CellFormat(15, 5, fmt.Sprintf("%.0f%%", vatPct), "1", 0, "C", false, 0, "")
	pdf.CellFormat(25, 5, fmt.Sprintf("%.3f", totalAmtExclVAT), "1", 0, "R", false, 0, "")
	pdf.CellFormat(20, 5, fmt.Sprintf("%.3f", totalVAT), "1", 0, "R", false, 0, "")
	pdf.Ln(5)

	pdf.SetX(140)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.CellFormat(15, 5, "Total", "1", 0, "C", false, 0, "")
	pdf.CellFormat(25, 5, fmt.Sprintf("%.3f", totalAmtExclVAT), "1", 0, "R", false, 0, "")
	pdf.CellFormat(20, 5, fmt.Sprintf("%.3f", totalVAT), "1", 0, "R", false, 0, "")

	// ========================================================================
	// SECTION 7.5: PROFORMA DISCLAIMER (if applicable)
	// ========================================================================
	if invoice.Status == "Proforma" {
		pdf.SetY(pdf.GetY() + 4)
		pdf.SetX(10)
		pdf.SetFont("Helvetica", "BI", 8)
		pdf.SetTextColor(180, 60, 60)
		pdf.MultiCell(190, 4, "This is a Proforma Invoice and is not a tax document. It is issued for reference purposes only and does not constitute a demand for payment.", "", "C", false)
		pdf.Ln(2)
	}

	// ========================================================================
	// SECTION 8: DECLARATION + SIGNATURE
	// ========================================================================
	pdf.SetY(pdf.GetY() + 10)
	declY := pdf.GetY()

	// Left side: Declaration
	pdf.SetXY(10, declY)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(30, 5, "Declaration:")
	pdf.Ln(5)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(60, 60, 60)
	declaration := "We declare that this invoice shows the actual price of the goods described and that all particulars are true and correct."
	pdf.MultiCell(90, 4, declaration, "", "", false)

	// Right side: Signature box
	pdf.SetXY(130, declY)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(60, 5, "For "+profile.LegalName)
	pdf.Ln(20)
	pdf.SetX(130)
	pdf.Cell(60, 5, "________________________")
	pdf.Ln(4)
	pdf.SetX(130)
	pdf.SetFont("Helvetica", "", 7)
	pdf.Cell(60, 5, "Authorized Signatory")

	// ========================================================================
	// SECTION 9: BANK DETAILS
	// ========================================================================
	pdf.SetY(pdf.GetY() + 8)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 5, "BANK DETAILS:")
	pdf.Ln(5)

	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(60, 60, 60)
	if len(profile.BankDetails) > 0 {
		for _, bankLine := range profile.BankDetails {
			pdf.SetX(10)
			pdf.Cell(0, 4, bankLine)
			pdf.Ln(4)
		}
	} else {
		// C-f: the fallback account list spans BOTH divisions. Only render banks
		// for this invoice's division so a PH invoice never shows AHS banks (and
		// vice-versa). Number the rendered lines, not the source index.
		targetDiv := normalizeDivisionName(invoice.Division)
		n := 0
		for _, bank := range bankAccounts {
			if normalizeDivisionName(bank.Division) != targetDiv {
				continue
			}
			n++
			pdf.SetX(10)
			bankLine := fmt.Sprintf("%d. %s, A/c No: %s, IBAN: %s, BIC: %s",
				n, bank.BankName, bank.AccountNumber, bank.IBAN, bank.SwiftBIC)
			pdf.Cell(0, 4, bankLine)
			pdf.Ln(4)
		}
	}

	// ========================================================================
	// SECTION 10: PAGE NUMBERS
	// ========================================================================
	// Disable auto-break before writing page numbers to prevent blank trailing page
	pdf.SetAutoPageBreak(false, 0)
	totalPages := pdf.PageCount()
	for i := 1; i <= totalPages; i++ {
		pdf.SetPage(i)
		pdf.SetY(265)
		pdf.SetFont("Helvetica", "I", 7)
		pdf.SetTextColor(160, 160, 160)
		pdf.CellFormat(0, 5, fmt.Sprintf("Invoice %s  |  Page %d of %d  |  %s", invoice.InvoiceNumber, i, totalPages, profile.LegalName), "", 0, "C", false, 0, "")
	}

	// ========================================================================
	// SECTION 11: SAVE PDF
	// ========================================================================
	cleanInvoiceNum := filepath.Base(strings.ReplaceAll(invoice.InvoiceNumber, "..", ""))
	cleanInvoiceNum = strings.ReplaceAll(cleanInvoiceNum, "/", "_")
	cleanInvoiceNum = strings.ReplaceAll(cleanInvoiceNum, "\\", "_")
	// Combined filename: {SystemNumber}_{UserReference}.pdf
	userRef := ""
	if invoice.CustomerPONumber != "" {
		userRef = sanitizeFilename(invoice.CustomerPONumber)
	} else if invoice.CustomerReference != "" {
		userRef = sanitizeFilename(invoice.CustomerReference)
	}
	filename := cleanInvoiceNum
	if userRef != "" {
		filename = fmt.Sprintf("%s_%s", cleanInvoiceNum, userRef)
	}
	filename = fmt.Sprintf("Invoice_%s.pdf", filename)

	docYear := invoice.InvoiceDate.Year()
	if docYear <= 0 {
		docYear = time.Now().Year()
	}
	saveDir := a.getExportDir("customer", invoice.CustomerName, "Order", docYear)
	filePath := filepath.Join(saveDir, filename)

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", fmt.Errorf("failed to save customer invoice PDF: %w", err)
	}

	log.Printf("✅ Customer invoice PDF generated: %s", filePath)

	// Phase 23: Generate e-invoice XML alongside the PDF
	if xmlPath, err := a.GenerateEInvoiceXML(invoiceID); err != nil {
		log.Printf("⚠️ E-Invoice XML generation failed (non-blocking): %v", err)
	} else {
		log.Printf("✅ E-Invoice XML generated alongside PDF: %s", xmlPath)
	}

	return filePath, nil
}

// ============================================================================
// SUPPLIER INVOICE PDF GENERATION
// ============================================================================

// GenerateSupplierInvoicePDF creates a supplier invoice PDF (simpler internal format)
func (a *App) GenerateSupplierInvoicePDF(invoiceID string) (string, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return "", err
	}

	log.Printf("📄 Generating supplier invoice PDF: invoiceID=%s", invoiceID)

	// 1. Fetch supplier invoice from database
	var invoice SupplierInvoice
	if err := a.db.First(&invoice, "id = ?", invoiceID).Error; err != nil {
		return "", fmt.Errorf("failed to fetch supplier invoice: %w", err)
	}

	// 2. Fetch supplier details
	var supplier SupplierMaster
	if err := a.db.First(&supplier, "id = ?", invoice.SupplierID).Error; err != nil {
		log.Printf("⚠️ Could not fetch supplier details: %v", err)
	}

	// 3. Fetch PO details if linked
	var po PurchaseOrder
	if invoice.PurchaseOrderID != "" {
		a.db.First(&po, "id = ?", invoice.PurchaseOrderID)
	}
	profile := companyDocumentProfile(a.resolveSupplierInvoiceDivision(invoice))

	// 4. Create PDF with letterhead
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 35)
	pdf.SetTopMargin(50)
	pdf.SetLeftMargin(18)
	pdf.SetRightMargin(18)

	// Set header function so letterhead appears on ALL pages
	pdf.SetHeaderFuncMode(func() {
		a.applyLetterheadForDivision(pdf, profile.Division)
	}, true)

	pdf.AddPage()

	// Header
	pdf.SetY(52)
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 10, "SUPPLIER INVOICE RECORD")
	pdf.Ln(12)

	// Invoice metadata
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)

	pdf.Cell(45, 6, "Supplier Invoice No:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, sanitizeForPDF(invoice.InvoiceNumber))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 6, "Invoice Date:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, invoice.InvoiceDate.Format("02 Jan 2006"))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 6, "Due Date:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, invoice.DueDate.Format("02 Jan 2006"))
	pdf.Ln(6)

	if po.PONumber != "" {
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(80, 80, 80)
		pdf.Cell(45, 6, "Our PO Number:")
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(29, 29, 31)
		pdf.Cell(0, 6, sanitizeForPDF(po.PONumber))
		pdf.Ln(6)
	}

	pdf.Ln(4)

	// Supplier details
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, "SUPPLIER:")
	pdf.Ln(7)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(0, 6, sanitizeForPDF(supplier.SupplierName))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(60, 60, 60)
	if supplier.Country != "" {
		pdf.Cell(0, 5, sanitizeForPDF(supplier.Country))
		pdf.Ln(5)
	}
	if supplier.TaxID != "" {
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(80, 80, 80)
		pdf.Cell(25, 5, "Tax ID:")
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(29, 29, 31)
		pdf.Cell(0, 5, sanitizeForPDF(supplier.TaxID))
		pdf.Ln(5)
	}

	pdf.Ln(6)

	// Financial details
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, "AMOUNTS:")
	pdf.Ln(7)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)

	pdf.Cell(45, 6, "Currency:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, sanitizeForPDF(invoice.Currency))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 6, "Subtotal (Foreign):")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, fmt.Sprintf("%.2f %s", invoice.SubtotalForeign, invoice.Currency))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 6, "VAT (Foreign):")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, fmt.Sprintf("%.2f %s", invoice.VATForeign, invoice.Currency))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 6, "Total (Foreign):")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, fmt.Sprintf("%.2f %s", invoice.TotalForeign, invoice.Currency))
	pdf.Ln(8)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 6, "Exchange Rate:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, fmt.Sprintf("%.4f", invoice.ExchangeRate))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(45, 7, "Total (BHD):")
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 7, fmt.Sprintf("%.3f BHD", invoice.TotalBHD))
	pdf.Ln(10)

	// Status section
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, "STATUS:")
	pdf.Ln(7)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 6, "Match Status:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, sanitizeForPDF(invoice.MatchStatus))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 6, "Approval Status:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, sanitizeForPDF(invoice.Status))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 6, "Payment Status:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, sanitizeForPDF(invoice.PaymentStatus))
	pdf.Ln(6)

	if invoice.PaymentDate != nil {
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(80, 80, 80)
		pdf.Cell(45, 6, "Payment Date:")
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(29, 29, 31)
		pdf.Cell(0, 6, invoice.PaymentDate.Format("02 Jan 2006"))
		pdf.Ln(6)
	}

	// Page footer
	pdf.SetY(270)
	pdf.SetFont("Helvetica", "I", 7)
	pdf.SetTextColor(160, 160, 160)
	pdf.Cell(0, 5, fmt.Sprintf("Supplier Invoice Record  |  %s  |  Generated %s", sanitizeForPDF(profile.LegalName), time.Now().Format("02 Jan 2006")))

	// Save PDF
	cleanInvoiceNum := filepath.Base(strings.ReplaceAll(invoice.InvoiceNumber, "..", ""))
	cleanInvoiceNum = strings.ReplaceAll(cleanInvoiceNum, "/", "_")
	cleanInvoiceNum = strings.ReplaceAll(cleanInvoiceNum, "\\", "_")
	userRef := ""
	if invoice.PONumber != "" {
		userRef = sanitizeFilename(invoice.PONumber)
	}
	filename := cleanInvoiceNum
	if userRef != "" {
		filename = fmt.Sprintf("%s_%s", cleanInvoiceNum, userRef)
	}
	filename = fmt.Sprintf("SupplierInv_%s.pdf", filename)

	docYear := invoice.InvoiceDate.Year()
	if docYear <= 0 {
		docYear = time.Now().Year()
	}
	saveDir := a.getExportDir("supplier", invoice.SupplierName, "MISC", docYear)
	filePath := filepath.Join(saveDir, filename)

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", fmt.Errorf("failed to save supplier invoice PDF: %w", err)
	}

	log.Printf("✅ Supplier invoice PDF generated: %s", filePath)
	return filePath, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// amountInWords converts a BHD amount to words
// Example: 1234.567 -> "One Thousand Two Hundred Thirty Four Bahraini Dinars and Five Hundred Sixty Seven Fils"
func amountInWords(amount float64) string {
	// Split into dinars (integer part) and fils (fractional part with 3 decimals)
	dinars := int(amount)
	fils := int((amount - float64(dinars)) * 1000)

	// Convert dinars to words
	dinarsWords := intToWords(dinars)
	if dinarsWords == "" {
		dinarsWords = "Zero"
	}

	// Build the result
	result := dinarsWords + " Bahraini Dinar"
	if dinars != 1 {
		result += "s"
	}

	// Add fils if non-zero
	if fils > 0 {
		filsWords := intToWords(fils)
		result += " and " + filsWords + " Fils"
	}

	return result
}

// intToWords converts an integer to English words (0-999999)
func intToWords(n int) string {
	if n == 0 {
		return ""
	}

	// Word arrays
	ones := []string{"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine"}
	teens := []string{"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen", "Seventeen", "Eighteen", "Nineteen"}
	tens := []string{"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety"}

	var words []string

	// Handle thousands (1000-999999)
	if n >= 1000 {
		thousands := n / 1000
		words = append(words, intToWords(thousands), "Thousand")
		n = n % 1000
	}

	// Handle hundreds (100-999)
	if n >= 100 {
		hundreds := n / 100
		words = append(words, ones[hundreds], "Hundred")
		n = n % 100
	}

	// Handle 10-19
	if n >= 10 && n < 20 {
		words = append(words, teens[n-10])
		return strings.Join(words, " ")
	}

	// Handle tens (20-99)
	if n >= 20 {
		words = append(words, tens[n/10])
		n = n % 10
	}

	// Handle ones (1-9)
	if n > 0 {
		words = append(words, ones[n])
	}

	return strings.Join(words, " ")
}

// truncatePDFTextToWidth shortens text with a trailing ellipsis so it fits
// within maxWidth (mm) at the PDF's current font. gofpdf's Cell does not clip,
// so this is used to keep long reference numbers inside their header cell
// instead of overrunning neighbouring columns.
func truncatePDFTextToWidth(pdf *gofpdf.Fpdf, text string, maxWidth float64) string {
	if text == "" || maxWidth <= 0 || pdf.GetStringWidth(text) <= maxWidth {
		return text
	}
	const ellipsis = "..."
	runes := []rune(text)
	for len(runes) > 0 && pdf.GetStringWidth(string(runes)+ellipsis) > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return strings.TrimRight(string(runes), " ") + ellipsis
}
