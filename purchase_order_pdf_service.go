// ═══════════════════════════════════════════════════════════════════════════
// PURCHASE ORDER PDF GENERATION SERVICE
//
// MISSION: Generate professional purchase orders on Acme Instrumentation letterhead
//          matching the company's exact format standards
//
// FORMAT:
//   PAGE 1 - PURCHASE ORDER:
//     - Acme Instrumentation letterhead background (full page template)
//     - Header: "PURCHASE ORDER" title (top right)
//     - Left block: Supplier details ("TO:")
//     - Right block: Acme Instrumentation delivery address ("DELIVER TO:")
//     - PO metadata: PO Number, Date, Supplier Ref, Buyer Order No, Payment Terms
//     - Line items table: # | Description | Qty | Unit Price | Amount (BHD)
//     - Totals: SUBTOTAL, VAT 10%, TOTAL (BHD)
//     - If multi-currency: show foreign currency amounts + exchange rate
//
//   PAGE 2 - TERMS AND CONDITIONS:
//     - Same letterhead on every page
//     - "Terms and Conditions" header
//     - Standard procurement terms
//     - Bank details for supplier payments (4 banks)
//     - Authorized signature line
//     - Company stamp area
//
// OUTPUTS:
//   - PO PDFs: exports/purchase_orders/PO_<number>.pdf
//
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// ============================================================================
// PURCHASE ORDER PDF GENERATION
// ============================================================================

// GeneratePurchaseOrderPDF creates a professional purchase order on Acme Instrumentation letterhead
func (a *App) GeneratePurchaseOrderPDF(poID string) (string, error) {
	if err := a.requirePermission("po:view"); err != nil {
		return "", err
	}

	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	if poID == "" {
		return "", fmt.Errorf("purchase order ID is required")
	}

	log.Printf("📄 Generating purchase order PDF: poID=%s", poID)

	// 1. Fetch PO with items from database
	var po PurchaseOrder
	if err := a.db.Preload("Items").First(&po, "id = ?", poID).Error; err != nil {
		return "", fmt.Errorf("failed to fetch purchase order: %w", err)
	}

	// Mission G (Wave 4, Commander-approved): mark an unapproved PO so it cannot
	// be mistaken for a supplier-ready document. A Draft / Pending Approval PO
	// carries a red "NOT VALID FOR SUPPLIER ISSUE" banner (parity with PH).
	approvalPending := po.Status == "Draft" || po.Status == "Pending Approval"

	// Validate PO has items
	if len(po.Items) == 0 {
		return "", fmt.Errorf("This purchase order has no line items. Please add items before generating a PDF")
	}

	// 2. Fetch supplier details
	var supplier SupplierMaster
	if err := a.db.First(&supplier, "id = ?", po.SupplierID).Error; err != nil {
		log.Printf("⚠️ Could not fetch supplier details: %v", err)
		// Continue with minimal supplier info from PO
		supplier.SupplierName = po.SupplierName
		if supplier.SupplierName == "" {
			supplier.SupplierName = "Unknown Supplier"
		}
	}

	// 3. Create PDF with letterhead
	division := a.resolvePurchaseOrderDivision(po)
	profile := companyDocumentProfile(division)
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 35) // Bottom margin: 35mm to avoid letterhead footer
	pdf.SetTopMargin(50)           // Start content below letterhead header
	pdf.SetLeftMargin(18)
	pdf.SetRightMargin(18)

	// Set header function so letterhead appears on ALL pages
	pdf.SetHeaderFuncMode(func() {
		a.applyLetterheadForDivision(pdf, profile.Division)
	}, true) // true = allow content to overlap the header area

	// ========================================================================
	// PAGE 1 - PURCHASE ORDER
	// ========================================================================
	pdf.AddPage()

	// I-27: an unapproved PO (Draft / Pending Approval) carries a large faint
	// diagonal DRAFT watermark across page 1 in addition to the red banner below,
	// so a printed copy can never be mistaken for a supplier-ready document.
	if approvalPending {
		drawDraftWatermark(pdf)
	}

	// Position for content start (below letterhead logo area)
	pdf.SetY(52)

	// Centered document title
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetTextColor(29, 29, 31)
	pdf.CellFormat(0, 10, "PURCHASE ORDER", "", 0, "C", false, 0, "")
	pdf.Ln(12)

	if approvalPending {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(180, 40, 40)
		pdf.CellFormat(0, 5, "DRAFT / PENDING APPROVAL - NOT VALID FOR SUPPLIER ISSUE", "", 0, "C", false, 0, "")
		pdf.Ln(8)
		pdf.SetTextColor(29, 29, 31)
	}

	// ========================================================================
	// SUPPLIER AND DELIVERY BLOCKS (2 columns)
	// ========================================================================
	y := pdf.GetY()

	// LEFT BLOCK: SUPPLIER ("TO:")
	pdf.SetXY(18, y)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, "TO:")
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(29, 29, 31)
	pdf.MultiCell(85, 5, sanitizeForPDF(supplier.SupplierName), "", "", false)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(60, 60, 60)

	// Supplier address
	if supplier.Address != "" {
		pdf.MultiCell(85, 5, sanitizeForPDF(supplier.Address), "", "", false)
	}
	if supplier.Country != "" {
		pdf.MultiCell(85, 5, sanitizeForPDF(supplier.Country), "", "", false)
	}

	// Supplier TaxID (TRN equivalent)
	if supplier.TaxID != "" {
		pdf.Ln(2)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.Cell(20, 5, "TRN:")
		pdf.SetFont("Helvetica", "", 9)
		pdf.Cell(0, 5, sanitizeForPDF(supplier.TaxID))
		pdf.Ln(5)
	}

	// RIGHT BLOCK: DELIVER TO
	rightX := 110.0
	pdf.SetXY(rightX, y)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, "DELIVER TO:")
	pdf.Ln(6)

	pdf.SetXY(rightX, pdf.GetY())
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 5, sanitizeForPDF(profile.LegalName))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(60, 60, 60)
	for _, line := range profile.AddressLines {
		pdf.SetXY(rightX, pdf.GetY())
		pdf.Cell(82, 5, sanitizeForPDF(line))
		pdf.Ln(5)
	}

	pdf.SetXY(rightX, pdf.GetY()+2)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.Cell(20, 5, "TRN:")
	pdf.SetFont("Helvetica", "", 9)
	pdf.Cell(0, 5, sanitizeForPDF(profile.VATNumber))
	pdf.Ln(8)

	// ========================================================================
	// PO METADATA (Right-aligned table)
	// ========================================================================
	metadataY := pdf.GetY()
	metaX := 105.0

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)

	// PO Number
	pdf.SetXY(metaX, metadataY)
	pdf.Cell(45, 5, "PO Number:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 5, sanitizeForPDF(po.PONumber))

	// PO Date
	pdf.SetXY(metaX, metadataY+6)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 5, "Date:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 5, po.PODate.Format("02-Jan-2006"))

	// Supplier Reference (use RFQ ID if available, otherwise blank)
	pdf.SetXY(metaX, metadataY+12)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 5, "Supplier Ref:")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(29, 29, 31)
	supplierRef := po.RfqID
	if supplierRef == "" {
		supplierRef = "_______________"
	}
	pdf.Cell(0, 5, sanitizeForPDF(supplierRef))

	// Buyer Order Number
	buyerOrderNo := "_______________"
	if po.OrderID != "" {
		var order Order
		if err := a.db.First(&order, "id = ?", po.OrderID).Error; err == nil {
			buyerOrderNo = order.OrderNumber
		}
	}
	pdf.SetXY(metaX, metadataY+18)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 5, "Buyer Order No:")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 5, sanitizeForPDF(buyerOrderNo))

	// Payment Terms
	pdf.SetXY(metaX, metadataY+24)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(45, 5, "Payment Terms:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	paymentTerms := po.PaymentTerms
	if paymentTerms == "" {
		paymentTerms = "Net 30 Days"
	}
	pdf.Cell(0, 5, sanitizeForPDF(paymentTerms))

	pdf.SetY(metadataY + 36)
	pdf.Ln(6)

	// ========================================================================
	// LINE ITEMS TABLE
	// ========================================================================
	tableX := 18.0
	tableColWidths := []float64{10, 86, 16, 34, 28}
	tableWidth := 0.0
	for _, width := range tableColWidths {
		tableWidth += width
	}
	tableLineHeight := 4.4
	minRowHeight := 7.8
	tableHeaderHeight := 8.0
	tableContinuationY := 52.0

	drawTableHeader := func() {
		pdf.SetX(tableX)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFillColor(29, 29, 31)
		pdf.CellFormat(tableColWidths[0], tableHeaderHeight, "#", "1", 0, "C", true, 0, "")
		pdf.CellFormat(tableColWidths[1], tableHeaderHeight, "Description", "1", 0, "C", true, 0, "")
		pdf.CellFormat(tableColWidths[2], tableHeaderHeight, "Qty", "1", 0, "C", true, 0, "")
		pdf.CellFormat(tableColWidths[3], tableHeaderHeight, "Unit Price", "1", 0, "C", true, 0, "")
		pdf.CellFormat(tableColWidths[4], tableHeaderHeight, "Amount", "1", 0, "C", true, 0, "")
		pdf.Ln(tableHeaderHeight)
		pdf.SetFont("Helvetica", "", 8.5)
		pdf.SetTextColor(29, 29, 31)
	}

	wrappedLineCount := func(text string, width float64) int {
		total := 0
		for _, paragraph := range strings.Split(text, "\n") {
			lines := pdf.SplitLines([]byte(paragraph), width)
			if len(lines) == 0 {
				total++
				continue
			}
			total += len(lines)
		}
		if total == 0 {
			return 1
		}
		return total
	}

	drawTableHeader()

	// Table rows
	// Determine if we're showing foreign currency
	showForeignCurrency := (po.Currency != "BHD")
	currencyLabel := po.Currency
	if currencyLabel == "" {
		currencyLabel = "BHD"
	}

	subtotal := 0.0
	for i, item := range po.Items {
		lineNum := i + 1
		description := sanitizeForPDF(item.Description)
		if item.ProductCode != "" {
			description = description + "\n" + sanitizeForPDF(item.ProductCode)
		}

		// Use foreign currency prices if available, otherwise BHD
		unitPrice := item.UnitPriceForeign
		lineTotal := item.TotalForeign
		if !showForeignCurrency {
			unitPrice = item.UnitPriceBHD
			lineTotal = item.TotalBHD
		}

		subtotal += lineTotal

		lineCount := wrappedLineCount(description, tableColWidths[1]-4)
		rowHeight := float64(lineCount)*tableLineHeight + 1.8
		if rowHeight < minRowHeight {
			rowHeight = minRowHeight
		}

		if pdf.GetY()+rowHeight > 235 {
			pdf.AddPage()
			pdf.SetY(tableContinuationY)
			drawTableHeader()
		}

		rowX := tableX
		rowY := pdf.GetY()
		pdf.SetDrawColor(175, 175, 175)
		pdf.SetLineWidth(0.18)
		if i%2 == 1 {
			pdf.SetFillColor(248, 250, 252)
			pdf.Rect(rowX, rowY, tableWidth, rowHeight, "FD")
		} else {
			pdf.Rect(rowX, rowY, tableWidth, rowHeight, "D")
		}
		gridX := rowX
		for colIdx := 0; colIdx < len(tableColWidths)-1; colIdx++ {
			gridX += tableColWidths[colIdx]
			pdf.Line(gridX, rowY, gridX, rowY+rowHeight)
		}

		pdf.SetFont("Helvetica", "", 8.7)
		pdf.SetTextColor(29, 29, 31)
		pdf.SetXY(rowX, rowY+(rowHeight-4.6)/2)
		pdf.CellFormat(tableColWidths[0], 4.6, fmt.Sprintf("%d", lineNum), "", 0, "C", false, 0, "")

		pdf.SetXY(rowX+tableColWidths[0]+2, rowY+1.3)
		pdf.MultiCell(tableColWidths[1]-4, tableLineHeight, description, "", "L", false)

		numericY := rowY + (rowHeight-4.6)/2
		qtyX := rowX + tableColWidths[0] + tableColWidths[1]
		unitX := qtyX + tableColWidths[2]
		amountX := unitX + tableColWidths[3]
		pdf.SetXY(qtyX, numericY)
		pdf.CellFormat(tableColWidths[2], 4.6, fmt.Sprintf("%.0f", item.Quantity), "", 0, "C", false, 0, "")
		pdf.SetXY(unitX+1, numericY)
		pdf.CellFormat(tableColWidths[3]-2, 4.6, fmt.Sprintf("%.3f %s", unitPrice, currencyLabel), "", 0, "R", false, 0, "")
		pdf.SetXY(amountX+1, numericY)
		pdf.CellFormat(tableColWidths[4]-2, 4.6, fmt.Sprintf("%.3f", lineTotal), "", 0, "R", false, 0, "")
		pdf.SetY(rowY + rowHeight)
	}

	pdf.Ln(3)

	// ========================================================================
	// TOTALS SECTION (right-aligned)
	// ========================================================================
	vatAmount := po.VATAmount
	if showForeignCurrency {
		vatAmount = subtotal * 0.10
	}
	grandTotal := subtotal + vatAmount

	totalsX := 110.0
	labelWidth := 34.0
	valueWidth := 48.0
	totalRowHeight := 7.0
	totalBlockHeight := totalRowHeight * 3
	if showForeignCurrency && po.ExchangeRate > 0 {
		totalBlockHeight += totalRowHeight + 5
	}
	if pdf.GetY()+totalBlockHeight > 238 {
		pdf.AddPage()
		pdf.SetY(tableContinuationY)
	}

	drawTotalRow := func(label, value string, bold bool, green bool) {
		rowY := pdf.GetY()
		pdf.SetDrawColor(190, 190, 190)
		if bold {
			pdf.SetFillColor(242, 248, 245)
			pdf.Rect(totalsX, rowY, labelWidth+valueWidth, totalRowHeight, "FD")
		} else {
			pdf.SetFillColor(255, 255, 255)
			pdf.Rect(totalsX, rowY, labelWidth+valueWidth, totalRowHeight, "D")
		}
		pdf.Line(totalsX+labelWidth, rowY, totalsX+labelWidth, rowY+totalRowHeight)

		if bold {
			pdf.SetFont("Helvetica", "B", 9)
		} else {
			pdf.SetFont("Helvetica", "", 8.5)
		}
		pdf.SetTextColor(80, 80, 80)
		pdf.SetXY(totalsX+2, rowY+1.5)
		pdf.CellFormat(labelWidth-4, 4.2, label, "", 0, "L", false, 0, "")

		if bold {
			pdf.SetFont("Helvetica", "B", 9)
		} else {
			pdf.SetFont("Helvetica", "", 8.5)
		}
		if green {
			pdf.SetTextColor(0, 120, 80)
		} else {
			pdf.SetTextColor(29, 29, 31)
		}
		pdf.SetXY(totalsX+labelWidth+2, rowY+1.5)
		pdf.CellFormat(valueWidth-4, 4.2, value, "", 0, "R", false, 0, "")
		pdf.SetY(rowY + totalRowHeight)
	}

	drawTotalRow("SUBTOTAL", fmt.Sprintf("%.3f %s", subtotal, currencyLabel), false, false)
	drawTotalRow("VAT 10%", fmt.Sprintf("%.3f %s", vatAmount, currencyLabel), false, false)
	drawTotalRow(fmt.Sprintf("TOTAL (%s)", currencyLabel), fmt.Sprintf("%.3f %s", grandTotal, currencyLabel), true, false)

	if showForeignCurrency && po.ExchangeRate > 0 {
		totalBHD := po.TotalBHD
		drawTotalRow("TOTAL (BHD)", fmt.Sprintf("%.3f BHD", totalBHD), true, true)
		pdf.SetFont("Helvetica", "I", 7.5)
		pdf.SetTextColor(120, 120, 120)
		pdf.SetX(totalsX)
		pdf.CellFormat(labelWidth+valueWidth, 5, fmt.Sprintf("Exchange Rate: 1 %s = %.4f BHD", currencyLabel, po.ExchangeRate), "", 0, "R", false, 0, "")
		pdf.Ln(6)
	}

	// ========================================================================
	// PAGE 2 - TERMS AND CONDITIONS
	// ========================================================================
	pdf.AddPage()

	pdf.SetY(52)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(0, 120, 80) // Green header
	pdf.Cell(0, 8, "Terms and Conditions")
	pdf.Ln(10)

	// Terms bullets
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(40, 40, 40)

	terms := []string{
		fmt.Sprintf("Payment Terms: %s", paymentTerms),
		"Delivery Terms: CIF Bahrain (unless otherwise specified)",
		fmt.Sprintf("Expected Delivery: %s", po.ExpectedDelivery.Format("02-Jan-2006")),
		"Warranty: As per manufacturer's standard warranty terms",
		"Order Cancellation: Subject to supplier approval and restocking charges",
		"Inspection: Goods subject to inspection upon receipt",
		"Quality Standards: All items must meet specified quality standards",
		"Packaging: All items to be properly packaged for international shipping",
		"Documentation: Invoice, packing list, and certificates to accompany shipment",
		"Late Delivery: Supplier to notify buyer immediately of any delays",
		fmt.Sprintf("Partial Shipments: Subject to prior approval from %s", profile.Division),
		"Dispute Resolution: Any disputes to be resolved per Bahrain Commercial Law",
	}

	termNumberWidth := 8.0
	termTextWidth := 160.0
	termLineHeight := 5.5
	for i, term := range terms {
		if pdf.GetY()+termLineHeight > 238 {
			pdf.AddPage()
			pdf.SetY(52)
			pdf.SetFont("Helvetica", "B", 14)
			pdf.SetTextColor(0, 120, 80)
			pdf.Cell(0, 8, "Terms and Conditions (Continued)")
			pdf.Ln(10)
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetTextColor(40, 40, 40)
		}

		termY := pdf.GetY()
		pdf.SetXY(20, termY)
		pdf.CellFormat(termNumberWidth, termLineHeight, fmt.Sprintf("%d.", i+1), "", 0, "R", false, 0, "")
		pdf.SetXY(20+termNumberWidth+2, termY)
		pdf.MultiCell(termTextWidth, termLineHeight, sanitizeForPDF(term), "", "L", false)
		pdf.Ln(1.5)
	}

	pdf.Ln(6)

	// ========================================================================
	// BANK DETAILS FOR PAYMENT
	// ========================================================================
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 8, "Bank Details for Payment")
	pdf.Ln(8)

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(60, 60, 60)

	// Bank 1: Demo Bank D
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(0, 5, "Demo Bank D")
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(40, 4, "Account Name:")
	pdf.Cell(0, 4, "ACME INSTRUMENTATION WLL")
	pdf.Ln(4)
	pdf.Cell(40, 4, "Account Number:")
	pdf.Cell(0, 4, "10000000004")
	pdf.Ln(4)
	pdf.Cell(40, 4, "IBAN:")
	pdf.Cell(0, 4, "BH29DMOD10000000000004")
	pdf.Ln(4)
	pdf.Cell(40, 4, "SWIFT:")
	pdf.Cell(0, 4, "DMODBHBM")
	pdf.Ln(6)

	// Bank 2: Demo Bank A
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(0, 5, "Demo Bank A")
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(40, 4, "Account Name:")
	pdf.Cell(0, 4, "ACME INSTRUMENTATION WLL")
	pdf.Ln(4)
	pdf.Cell(40, 4, "Account Number:")
	pdf.Cell(0, 4, "10000000001")
	pdf.Ln(4)
	pdf.Cell(40, 4, "IBAN:")
	pdf.Cell(0, 4, "BH29DMOA10000000000001")
	pdf.Ln(4)
	pdf.Cell(40, 4, "SWIFT:")
	pdf.Cell(0, 4, "DMOABHBM")
	pdf.Ln(6)

	// Bank 3: Demo Bank B
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(0, 5, "Demo Bank B")
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(40, 4, "Account Name:")
	pdf.Cell(0, 4, "ACME INSTRUMENTATION WLL")
	pdf.Ln(4)
	pdf.Cell(40, 4, "Account Number:")
	pdf.Cell(0, 4, "10000000002")
	pdf.Ln(4)
	pdf.Cell(40, 4, "IBAN:")
	pdf.Cell(0, 4, "BH29DMOB10000000000002")
	pdf.Ln(4)
	pdf.Cell(40, 4, "SWIFT:")
	pdf.Cell(0, 4, "DMOBBHBM")
	pdf.Ln(6)

	// Bank 4: Demo Bank C
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(0, 5, "Demo Bank C")
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "", 8)
	if len(profile.BankDetails) > 0 {
		for _, bankLine := range profile.BankDetails {
			pdf.MultiCell(0, 4, sanitizeForPDF(bankLine), "", "L", false)
			pdf.Ln(1)
		}
		pdf.Ln(6)
	} else {
		pdf.Cell(40, 4, "Account Name:")
		pdf.Cell(0, 4, "ACME INSTRUMENTATION WLL")
		pdf.Ln(4)
		pdf.Cell(40, 4, "Account Number:")
		pdf.Cell(0, 4, "10000000003")
		pdf.Ln(4)
		pdf.Cell(40, 4, "IBAN:")
		pdf.Cell(0, 4, "BH29DMOC10000000000003")
		pdf.Ln(4)
		pdf.Cell(40, 4, "SWIFT:")
		pdf.Cell(0, 4, "DMOCBHBM")
		pdf.Ln(10)
	}

	// ========================================================================
	// SIGNATURE SECTION
	// ========================================================================
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(0, 6, fmt.Sprintf("For %s", sanitizeForPDF(profile.LegalName)))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(60, 60, 60)
	pdf.Cell(0, 5, "Authorized Signature: _____________________")
	pdf.Ln(6)
	pdf.Cell(0, 5, "Name: _____________________")
	pdf.Ln(6)
	pdf.Cell(0, 5, "Date: _____________________")
	pdf.Ln(8)

	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(120, 120, 120)
	pdf.Cell(0, 4, "[Company Stamp]")
	pdf.Ln(10)

	// ========================================================================
	// PAGE NUMBERS (on all pages, positioned above letterhead footer)
	// ========================================================================
	// Disable auto-break before writing page numbers to prevent blank trailing page
	pdf.SetAutoPageBreak(false, 0)
	totalPages := pdf.PageCount()
	for i := 1; i <= totalPages; i++ {
		pdf.SetPage(i)
		pdf.SetY(270) // Just above the letterhead footer area
		pdf.SetFont("Helvetica", "I", 7)
		pdf.SetTextColor(160, 160, 160)
		pdf.CellFormat(0, 5, fmt.Sprintf("Purchase Order %s  |  Page %d of %d  |  %s", po.PONumber, i, totalPages, sanitizeForPDF(profile.LegalName)), "", 0, "C", false, 0, "")
	}

	// ========================================================================
	// SAVE PDF
	// ========================================================================
	// Combined filename: {SystemNumber}_{UserReference}.pdf
	cleanPONum := sanitizeFilename(po.PONumber)
	userRef := ""
	if po.RfqID != "" {
		userRef = sanitizeFilename(po.RfqID)
	}
	filename := cleanPONum
	if userRef != "" {
		filename = fmt.Sprintf("%s_%s", cleanPONum, userRef)
	}
	filename = fmt.Sprintf("PO_%s.pdf", filename)

	docYear := po.PODate.Year()
	if docYear <= 0 {
		docYear = time.Now().Year()
	}
	saveDir := a.getExportDir("supplier", po.SupplierName, "Orders", docYear)
	filePath := filepath.Join(saveDir, filename)

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", fmt.Errorf("failed to save purchase order PDF: %w", err)
	}

	log.Printf("✅ Purchase Order PDF generated: %s", filePath)
	return filePath, nil
}

// drawDraftWatermark renders a large faint diagonal "DRAFT" watermark across the
// current page. Drawn for purchase orders that are not yet approved so a printed
// copy is visually unmistakable as non-final — a companion to the red
// "NOT VALID FOR SUPPLIER ISSUE" banner. (I-27)
func drawDraftWatermark(pdf *gofpdf.Fpdf) {
	pdf.SetAlpha(0.10, "Normal")
	pdf.SetFont("Helvetica", "B", 90)
	pdf.SetTextColor(200, 40, 40)
	// Rotate ~55° about the A4 page centre (105mm, 148mm) and draw the text there.
	pdf.TransformBegin()
	pdf.TransformRotate(55, 105, 148)
	pdf.SetXY(20, 138)
	pdf.CellFormat(170, 20, "DRAFT", "", 0, "C", false, 0, "")
	pdf.TransformEnd()
	// Restore opaque drawing and the default body text colour for the rest of the
	// document (the caller resets font/position immediately after).
	pdf.SetAlpha(1.0, "Normal")
	pdf.SetTextColor(29, 29, 31)
}
