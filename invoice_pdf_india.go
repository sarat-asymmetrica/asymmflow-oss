// ═══════════════════════════════════════════════════════════════════════════
// INDIA GST DOCUMENT EMISSION (India Spec-01 B4)
//
// Renders the two India document kinds through the same gofpdf pipeline
// GenerateInvoicePDF/GenerateCreditNotePDF already use (letterhead, bank
// details, signature blocks) — only the invoice body differs from the GCC
// layout:
//
//   - TAX INVOICE:     all 16 Rule-46 fields (§0 G5), per-line HSN/UQC, a
//     CGST/SGST/IGST/Cess column split, reverse-charge statement.
//   - BILL OF SUPPLY:  composition dealers (§0 G6) — no tax lines at all,
//     mandatory legend, own numbering series.
//
// Both branches compute their GST split via pkg/compliance/india's engine
// (B3) and refuse to generate — returning an error, never a bad PDF — on an
// HSNValidationError or a Rule-46 numbering violation. GCC divisions never
// reach this file: GenerateInvoicePDF/GenerateCreditNotePDF branch to it only
// when companyDocumentProfile(...).India is non-nil.
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"

	"ph_holdings_app/pkg/compliance/india"
)

// indiaLineDisplay is the rendering-agnostic shape renderIndiaLineItemsTable
// needs from either a DBInvoiceItem or a CreditNoteItem, so one table
// renderer serves both documents.
type indiaLineDisplay struct {
	HSN         string
	Description string
	Quantity    float64
	UQC         string
	Rate        float64
}

// computeIndiaGSTLines is the shared entry point into the B3 engine for both
// the tax invoice and the credit note: build the supplier/supply facts from
// the division's India profile and the transaction's place-of-supply/B2B/
// reverse-charge flags, then compute. AATOINR uses
// india.ResolveAATO(0, cfg.AATOOverrideINR) — this wave has no invoice-history
// AATO summer yet (a document store would be needed to sum PAN-level
// turnover), so only an explicit overlay override feeds the HSN-tier
// boundary; a future mission wires the real roll-up.
func computeIndiaGSTLines(profile CompanyDocumentProfile, posStateCode string, b2b, reverseCharge bool, lines []india.Line) (*india.InvoiceResult, error) {
	cfg := activeOverlay.IndiaConfig()
	aato := india.ResolveAATO(0, cfg.AATOOverrideINR)
	supplier := india.Supplier{StateCode: profile.India.StateCode, Composition: profile.India.Composition}
	supply := india.Supply{PlaceOfSupplyStateCode: posStateCode, B2B: b2b, ReverseCharge: reverseCharge}
	return india.ComputeInvoiceGST(supplier, supply, lines, india.EngineConfig{Rates: cfg, AATOINR: aato})
}

// resolveIndiaGSTForInvoice builds the engine's per-line input from an
// invoice's items (taxable value = rate × quantity, less the invoice-level
// discount — the same taxable-value derivation the GCC line table already
// uses for its own display recompute).
func resolveIndiaGSTForInvoice(invoice Invoice, profile CompanyDocumentProfile) (*india.InvoiceResult, error) {
	lines := make([]india.Line, 0, len(invoice.Items))
	for _, item := range invoice.Items {
		taxable := item.Rate * item.Quantity
		if invoice.DiscountPercent > 0 {
			taxable *= 1 - invoice.DiscountPercent/100
		}
		lines = append(lines, india.Line{
			HSN:             item.HSNCode,
			Description:     item.Description,
			Quantity:        item.Quantity,
			UQC:             item.UQC,
			TaxableValueINR: taxable,
		})
	}
	return computeIndiaGSTLines(profile, invoice.PlaceOfSupplyStateCode, invoice.BuyerGSTIN != "", invoice.ReverseCharge, lines)
}

// resolveIndiaGSTForCreditNote is resolveIndiaGSTForInvoice's twin for a
// credit note's items, using the ORIGINAL invoice's place-of-supply/B2B/
// reverse-charge facts (a credit note is not itself a fresh transaction — it
// adjusts the one it references).
func resolveIndiaGSTForCreditNote(items []CreditNoteItem, profile CompanyDocumentProfile, posStateCode string, b2b, reverseCharge bool) (*india.InvoiceResult, error) {
	lines := make([]india.Line, 0, len(items))
	for _, item := range items {
		lines = append(lines, india.Line{
			HSN:             item.HSNCode,
			Description:     item.Description,
			Quantity:        item.Quantity,
			UQC:             item.UQC,
			TaxableValueINR: item.Rate * item.Quantity,
		})
	}
	return computeIndiaGSTLines(profile, posStateCode, b2b, reverseCharge, lines)
}

// renderIndiaLineItemsTable draws SECTION 4 for either document kind. A
// composition division (Bill of Supply, G6) gets a wider table with NO tax
// columns at all — the legend below it, not this table, carries the "why".
// Otherwise every line carries HSN, UQC, taxable value, and all four tax
// heads (CGST/SGST/IGST/Cess) — the non-applicable heads for a given
// classification render as "-" rather than a bare 0.00.
func renderIndiaLineItemsTable(pdf *gofpdf.Fpdf, rows []indiaLineDisplay, gst *india.InvoiceResult, isBoS bool) {
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(29, 29, 31)

	if isBoS {
		const colSl, colHSN, colDesc, colQty, colUQC, colRate, colTaxable, colTotal = 8.0, 18.0, 74.0, 14.0, 12.0, 20.0, 24.0, 20.0
		pdf.SetFont("Helvetica", "B", 7)
		pdf.SetX(10)
		pdf.CellFormat(colSl, 6, "Sl", "1", 0, "C", true, 0, "")
		pdf.CellFormat(colHSN, 6, "HSN/SAC", "1", 0, "C", true, 0, "")
		pdf.CellFormat(colDesc, 6, "Description of Goods", "1", 0, "C", true, 0, "")
		pdf.CellFormat(colQty, 6, "Qty", "1", 0, "C", true, 0, "")
		pdf.CellFormat(colUQC, 6, "UQC", "1", 0, "C", true, 0, "")
		pdf.CellFormat(colRate, 6, "Rate", "1", 0, "C", true, 0, "")
		pdf.CellFormat(colTaxable, 6, "Value", "1", 0, "C", true, 0, "")
		pdf.CellFormat(colTotal, 6, "Total", "1", 0, "C", true, 0, "")
		pdf.Ln(6)

		pdf.SetFont("Helvetica", "", 7)
		pdf.SetTextColor(29, 29, 31)
		for i, row := range rows {
			line := gst.Lines[i]
			desc := sanitizeForPDF(row.Description)
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			pdf.SetX(10)
			pdf.CellFormat(colSl, 5, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
			pdf.CellFormat(colHSN, 5, sanitizeForPDF(row.HSN), "1", 0, "C", false, 0, "")
			pdf.CellFormat(colDesc, 5, desc, "1", 0, "L", false, 0, "")
			pdf.CellFormat(colQty, 5, fmt.Sprintf("%.2f", row.Quantity), "1", 0, "C", false, 0, "")
			pdf.CellFormat(colUQC, 5, sanitizeForPDF(row.UQC), "1", 0, "C", false, 0, "")
			pdf.CellFormat(colRate, 5, indianDigitGrouping(row.Rate), "1", 0, "R", false, 0, "")
			pdf.CellFormat(colTaxable, 5, indianDigitGrouping(line.TaxableValueINR), "1", 0, "R", false, 0, "")
			pdf.CellFormat(colTotal, 5, indianDigitGrouping(line.TaxableValueINR), "1", 0, "R", false, 0, "")
			pdf.Ln(5)
		}
		return
	}

	const colSl, colHSN, colDesc, colQty, colUQC, colRate, colTaxable = 7.0, 14.0, 48.0, 9.0, 8.0, 14.0, 18.0
	const colCGST, colSGST, colIGST, colCess, colTotal = 15.0, 15.0, 15.0, 12.0, 15.0

	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetX(10)
	pdf.CellFormat(colSl, 6, "Sl", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colHSN, 6, "HSN", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colDesc, 6, "Description", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colQty, 6, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colUQC, 6, "UQC", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colRate, 6, "Rate", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colTaxable, 6, "Taxable Val", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colCGST, 6, "CGST", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colSGST, 6, "SGST", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colIGST, 6, "IGST", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colCess, 6, "Cess", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colTotal, 6, "Total", "1", 0, "C", true, 0, "")
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 6)
	pdf.SetTextColor(29, 29, 31)
	for i, row := range rows {
		line := gst.Lines[i]
		desc := sanitizeForPDF(row.Description)
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		total := line.TaxableValueINR + line.CGST + line.SGST + line.IGST + line.Cess
		pdf.SetX(10)
		pdf.CellFormat(colSl, 5, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colHSN, 5, sanitizeForPDF(row.HSN), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colDesc, 5, desc, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colQty, 5, fmt.Sprintf("%.2f", row.Quantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colUQC, 5, sanitizeForPDF(row.UQC), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colRate, 5, fmt.Sprintf("%.2f", row.Rate), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colTaxable, 5, indianDigitGrouping(line.TaxableValueINR), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colCGST, 5, formatIndiaTaxCell(line.CGST), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colSGST, 5, formatIndiaTaxCell(line.SGST), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colIGST, 5, formatIndiaTaxCell(line.IGST), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colCess, 5, formatIndiaTaxCell(line.Cess), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colTotal, 5, indianDigitGrouping(total), "1", 0, "R", false, 0, "")
		pdf.Ln(5)
	}
}

// formatIndiaTaxCell renders a tax-head cell as "-" when it doesn't apply to
// this line (CGST/SGST are always zero on an inter-state line and vice
// versa) rather than a bare "0.00", so the eye isn't drawn to a column that
// was never meant to carry a value on this row.
func formatIndiaTaxCell(amount float64) string {
	if amount == 0 {
		return "-"
	}
	return indianDigitGrouping(amount)
}

// renderIndiaTaxSummary draws the right-side totals block shared by both
// documents: taxable value, then each non-zero tax head, then the grand
// total. A Bill of Supply skips every tax head — G6, no tax lines at all —
// leaving only the taxable value and the (identical) total. isBoS is passed
// explicitly (rather than reading gst.Composition) so the summary can never
// disagree with the line table's own document-kind decision — a defense
// against a mis-stamped DocKind producing a hybrid document (gate fix).
func renderIndiaTaxSummary(pdf *gofpdf.Fpdf, gst *india.InvoiceResult, isBoS bool, totalLabel string, totalColor [3]int) float64 {
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(35, 5, "Taxable Value:")
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(30, 5, "Rs. "+indianDigitGrouping(gst.Totals.TaxableValueINR))
	pdf.Ln(5)

	if !isBoS && !gst.Composition {
		heads := []struct {
			label string
			amt   float64
		}{
			{"CGST:", gst.Totals.CGST},
			{"SGST:", gst.Totals.SGST},
			{"IGST:", gst.Totals.IGST},
			{"Cess:", gst.Totals.Cess},
		}
		for _, h := range heads {
			if h.amt == 0 {
				continue
			}
			pdf.SetX(pdf.GetX())
			pdf.SetFont("Helvetica", "", 8)
			pdf.SetTextColor(80, 80, 80)
			pdf.Cell(35, 5, h.label)
			pdf.SetFont("Helvetica", "B", 8)
			pdf.SetTextColor(29, 29, 31)
			pdf.Cell(30, 5, "Rs. "+indianDigitGrouping(h.amt))
			pdf.Ln(5)
		}
	}

	grandTotal := gst.Totals.TaxableValueINR + gst.Totals.CGST + gst.Totals.SGST + gst.Totals.IGST + gst.Totals.Cess
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(totalColor[0], totalColor[1], totalColor[2])
	pdf.Cell(35, 6, totalLabel)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(30, 6, "Rs. "+indianDigitGrouping(grandTotal))
	pdf.Ln(8)
	return grandTotal
}

// generateIndiaInvoicePDF renders the India Rule-46 tax invoice or, for a
// composition division (enforced regardless of caller intent, G6), the Bill
// of Supply. It shares GenerateInvoicePDF's letterhead, bank-details, and
// export-path conventions — only the body differs.
func (a *App) generateIndiaInvoicePDF(invoice Invoice, profile CompanyDocumentProfile, customer CustomerMaster, bankAccounts []CompanyBankAccount) (string, error) {
	isBoS := profile.India.Composition || invoice.DocKind == "bill_of_supply"

	gst, err := resolveIndiaGSTForInvoice(invoice, profile)
	if err != nil {
		return "", err // refuse-to-generate (HSNValidationError) — never a silently wrong PDF
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 30)
	pdf.SetTopMargin(40)
	pdf.SetLeftMargin(10)
	pdf.SetRightMargin(10)
	pdf.SetHeaderFuncMode(func() {
		a.applyLetterheadForDivision(pdf, profile.Division)
	}, true)
	pdf.AddPage()

	// SECTION 1: TITLE
	pdf.SetY(40)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(29, 29, 31)
	title := "TAX INVOICE"
	switch {
	case invoice.Status == "Proforma":
		title = "PROFORMA INVOICE"
	case isBoS:
		title = "BILL OF SUPPLY"
	}
	pdf.CellFormat(0, 8, title, "", 0, "C", false, 0, "")
	pdf.Ln(10)

	// SECTION 2: SELLER (GSTIN/PAN/state, not TRN) + INVOICE METADATA
	headerY := pdf.GetY()
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
	pdf.Cell(90, 4, "GSTIN: "+profile.India.GSTIN)
	pdf.Ln(4)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(60, 60, 60)
	if profile.India.PAN != "" {
		pdf.Cell(90, 4, "PAN: "+profile.India.PAN)
		pdf.Ln(4)
		pdf.SetX(10)
	}
	if profile.India.StateName != "" {
		pdf.Cell(90, 4, fmt.Sprintf("State: %s (%s)", profile.India.StateName, profile.India.StateCode))
	}

	rightX := 105.0
	metaY := headerY
	rowH := 4.0
	labelW := 32.0
	valueW := 40.0
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

	pdf.SetXY(rightX, metaY)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(labelW, rowH, "Place of Supply")
	pdf.Cell(valueW, rowH, "Reverse Charge")
	pdf.Ln(rowH)
	pdf.SetX(rightX)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(29, 29, 31)
	posLabel := invoice.PlaceOfSupply
	if name, ok := india.StateName(invoice.PlaceOfSupplyStateCode); ok {
		posLabel = fmt.Sprintf("%s (%s)", name, invoice.PlaceOfSupplyStateCode)
	}
	rcLabel := "No"
	if invoice.ReverseCharge {
		rcLabel = "Yes"
	}
	pdf.Cell(labelW, rowH, sanitizeForPDF(posLabel))
	pdf.Cell(valueW, rowH, rcLabel)

	pdf.SetY(metaY + rowH*2 + 4)

	// SECTION 3: BUYER
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(25, 5, "Buyer:")
	pdf.Ln(5)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 9)
	buyerName := customer.BusinessName
	if buyerName == "" {
		buyerName = invoice.AttentionCompany
	}
	pdf.Cell(0, 5, sanitizeForPDF(buyerName))
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(60, 60, 60)
	if customer.AddressLine1 != "" {
		pdf.SetX(10)
		pdf.Cell(0, 4, sanitizeForPDF(customer.AddressLine1))
		pdf.Ln(4)
	} else if invoice.AttentionAddress != "" {
		pdf.SetX(10)
		pdf.MultiCell(0, 4, sanitizeForPDF(invoice.AttentionAddress), "", "", false)
	}
	if customer.City != "" {
		pdf.SetX(10)
		pdf.Cell(0, 4, sanitizeForPDF(customer.City))
		pdf.Ln(4)
	}
	if invoice.BuyerGSTIN != "" {
		pdf.SetX(10)
		pdf.Cell(20, 4, "GSTIN:")
		pdf.SetFont("Helvetica", "B", 8)
		pdf.Cell(0, 4, sanitizeForPDF(invoice.BuyerGSTIN))
		pdf.Ln(4)
		pdf.SetFont("Helvetica", "", 8)
	} else {
		pdf.SetX(10)
		pdf.Cell(0, 4, "Unregistered buyer (B2C)")
		pdf.Ln(4)
	}
	if invoice.ShipToGSTIN != "" && invoice.ShipToGSTIN != invoice.BuyerGSTIN {
		pdf.SetX(10)
		pdf.Cell(30, 4, "Ship To GSTIN:")
		pdf.SetFont("Helvetica", "B", 8)
		pdf.Cell(0, 4, sanitizeForPDF(invoice.ShipToGSTIN))
		pdf.Ln(4)
		pdf.SetFont("Helvetica", "", 8)
	}
	pdf.Ln(2)

	// SECTION 4: LINE ITEMS
	rows := make([]indiaLineDisplay, len(invoice.Items))
	for i, item := range invoice.Items {
		descParts := make([]string, 0, 2)
		if item.Equipment != "" {
			descParts = append(descParts, item.Equipment)
		}
		if item.Model != "" {
			descParts = append(descParts, item.Model)
		} else if item.ProductCode != "" {
			descParts = append(descParts, item.ProductCode)
		}
		desc := strings.Join(descParts, " - ")
		if desc == "" {
			desc = item.Description
		}
		rows[i] = indiaLineDisplay{HSN: item.HSNCode, Description: desc, Quantity: item.Quantity, UQC: item.UQC, Rate: item.Rate}
	}
	renderIndiaLineItemsTable(pdf, rows, gst, isBoS)

	// SECTION 5: TOTALS
	pdf.Ln(4)
	pdf.SetXY(130, pdf.GetY())
	grandTotal := renderIndiaTaxSummary(pdf, gst, isBoS, "Grand Total:", [3]int{29, 29, 31})

	// SECTION 6: AMOUNT IN WORDS
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(45, 5, "Amount Chargeable (in words):")
	pdf.Ln(5)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(60, 60, 60)
	pdf.MultiCell(150, 4, amountInWordsIndian(grandTotal), "", "", false)
	pdf.Ln(2)

	// SECTION 7: LEGEND / REVERSE-CHARGE / PROFORMA DISCLAIMER
	if isBoS {
		pdf.SetX(10)
		pdf.SetFont("Helvetica", "BI", 8)
		pdf.SetTextColor(29, 29, 31)
		pdf.MultiCell(190, 4, "Composition taxable person, not eligible to collect tax on supplies.", "", "C", false)
		pdf.Ln(2)
	} else if invoice.ReverseCharge {
		pdf.SetX(10)
		pdf.SetFont("Helvetica", "BI", 8)
		pdf.SetTextColor(29, 29, 31)
		pdf.MultiCell(190, 4, "Tax payable on reverse charge basis: Yes", "", "L", false)
		pdf.Ln(2)
	}
	if invoice.Status == "Proforma" {
		pdf.SetX(10)
		pdf.SetFont("Helvetica", "BI", 8)
		pdf.SetTextColor(180, 60, 60)
		pdf.MultiCell(190, 4, "This is a Proforma Invoice and is not a tax document. It is issued for reference purposes only and does not constitute a demand for payment.", "", "C", false)
		pdf.Ln(2)
	}

	// SECTION 8: DECLARATION + SIGNATURE
	pdf.SetY(pdf.GetY() + 6)
	declY := pdf.GetY()
	pdf.SetXY(10, declY)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(30, 5, "Declaration:")
	pdf.Ln(5)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(60, 60, 60)
	pdf.MultiCell(90, 4, "We declare that this invoice shows the actual price of the goods/services described and that all particulars are true and correct.", "", "", false)

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

	// SECTION 9: BANK DETAILS
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
		targetDiv := normalizeDivisionName(invoice.Division)
		n := 0
		for _, bank := range bankAccounts {
			if normalizeDivisionName(bank.Division) != targetDiv {
				continue
			}
			n++
			pdf.SetX(10)
			bankLine := fmt.Sprintf("%d. %s, A/c No: %s, IBAN: %s, BIC: %s", n, bank.BankName, bank.AccountNumber, bank.IBAN, bank.SwiftBIC)
			pdf.Cell(0, 4, bankLine)
			pdf.Ln(4)
		}
	}

	// SECTION 10: PAGE NUMBERS
	pdf.SetAutoPageBreak(false, 0)
	totalPages := pdf.PageCount()
	for i := 1; i <= totalPages; i++ {
		pdf.SetPage(i)
		pdf.SetY(265)
		pdf.SetFont("Helvetica", "I", 7)
		pdf.SetTextColor(160, 160, 160)
		pdf.CellFormat(0, 5, fmt.Sprintf("Invoice %s  |  Page %d of %d  |  %s", invoice.InvoiceNumber, i, totalPages, profile.LegalName), "", 0, "C", false, 0, "")
	}

	// SECTION 11: SAVE
	// Substitute the Rule-46 "/" separators BEFORE filepath.Base — Base first
	// would strip everything up to the last separator and collapse
	// "INV/25-26/001" and "INV/26-27/001" onto the same "Invoice_001.pdf"
	// (two FY series can share a calendar-year export dir), silently
	// overwriting one with the other. (Gate fix; CN section below already
	// had this order right.)
	cleanInvoiceNum := strings.ReplaceAll(invoice.InvoiceNumber, "..", "")
	cleanInvoiceNum = strings.ReplaceAll(cleanInvoiceNum, "/", "_")
	cleanInvoiceNum = strings.ReplaceAll(cleanInvoiceNum, "\\", "_")
	cleanInvoiceNum = filepath.Base(cleanInvoiceNum)
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
		return "", fmt.Errorf("failed to save India invoice PDF: %w", err)
	}

	log.Printf("✅ India invoice PDF generated: %s (docKind=%q)", filePath, invoice.DocKind)
	return filePath, nil
}

// generateIndiaCreditNotePDF renders a credit note against an India invoice:
// the same GSTIN-labeled seller block, the referenced original invoice's
// number/date, and the same HSN/UQC/tax-split line table as the tax invoice.
func (a *App) generateIndiaCreditNotePDF(cn CreditNote, customer CustomerMaster, profile CompanyDocumentProfile) (string, error) {
	// Best-effort: a credit note whose original invoice was since deleted
	// should still render (mirrors GenerateCreditNotePDF's own best-effort
	// customer fetch) — place-of-supply/B2B/reverse-charge simply degrade to
	// their zero values in that edge case.
	var originalInvoice Invoice
	_ = a.db.Where("id = ?", cn.InvoiceID).First(&originalInvoice).Error

	rows := make([]indiaLineDisplay, len(cn.Items))
	for i, item := range cn.Items {
		rows[i] = indiaLineDisplay{HSN: item.HSNCode, Description: item.Description, Quantity: item.Quantity, UQC: item.UQC, Rate: item.Rate}
	}
	gst, err := resolveIndiaGSTForCreditNote(cn.Items, profile, originalInvoice.PlaceOfSupplyStateCode, originalInvoice.BuyerGSTIN != "", originalInvoice.ReverseCharge)
	if err != nil {
		return "", err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 30)
	pdf.SetTopMargin(50)
	pdf.SetLeftMargin(10)
	pdf.SetRightMargin(10)
	pdf.SetHeaderFuncMode(func() {
		a.applyLetterheadForDivision(pdf, profile.Division)
	}, true)
	pdf.AddPage()

	pdf.SetY(50)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(180, 40, 40)
	pdf.CellFormat(0, 8, "CREDIT NOTE", "", 0, "C", false, 0, "")
	pdf.Ln(10)

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
	pdf.Cell(90, 4, "GSTIN: "+sanitizeForPDF(profile.India.GSTIN))

	rightX := 120.0
	pdf.SetXY(rightX, headerY)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(35, 4, "CN Number:")
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(45, 4, sanitizeForPDF(cn.CreditNoteNumber))
	pdf.SetXY(rightX, headerY+5)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(35, 4, "CN Date:")
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(45, 4, cn.CreditNoteDate.Format("02-Jan-2006"))
	pdf.SetXY(rightX, headerY+10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(35, 4, "Orig. Invoice No.:")
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(45, 4, sanitizeForPDF(cn.InvoiceNumber))
	pdf.SetXY(rightX, headerY+15)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(35, 4, "Orig. Invoice Date:")
	pdf.SetFont("Helvetica", "", 8)
	origDate := ""
	if !originalInvoice.InvoiceDate.IsZero() {
		origDate = originalInvoice.InvoiceDate.Format("02-Jan-2006")
	}
	pdf.Cell(45, 4, origDate)
	pdf.SetXY(rightX, headerY+20)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.Cell(35, 4, "Status:")
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(45, 4, cn.Status)

	pdf.SetY(headerY + 28)
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
	if originalInvoice.BuyerGSTIN != "" {
		pdf.SetX(10)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.Cell(0, 4, "GSTIN: "+sanitizeForPDF(originalInvoice.BuyerGSTIN))
		pdf.Ln(4)
	}

	pdf.Ln(4)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(20, 4, "Reason:")
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(60, 60, 60)
	pdf.MultiCell(170, 4, sanitizeForPDF(cn.Reason), "", "", false)
	pdf.Ln(4)

	renderIndiaLineItemsTable(pdf, rows, gst, profile.India.Composition)

	pdf.Ln(4)
	pdf.SetXY(130, pdf.GetY())
	renderIndiaTaxSummary(pdf, gst, profile.India.Composition, "Credit Total:", [3]int{180, 40, 40})

	if profile.India.Composition {
		pdf.SetX(10)
		pdf.SetFont("Helvetica", "BI", 8)
		pdf.SetTextColor(29, 29, 31)
		pdf.MultiCell(190, 4, "Composition taxable person, not eligible to collect tax on supplies.", "", "L", false)
		pdf.Ln(2)
	}

	pdf.SetX(10)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(29, 29, 31)
	pdf.Cell(30, 5, "Declaration:")
	pdf.Ln(5)
	pdf.SetX(10)
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(60, 60, 60)
	pdf.MultiCell(90, 4, "We declare that this credit note shows the actual adjustment against the referenced invoice and that all particulars are true and correct.", "", "", false)

	pdf.SetXY(130, pdf.GetY()-15)
	pdf.SetTextColor(29, 29, 31)
	signerName := a.resolveDocumentSignerName(a.getCurrentUserDisplayName())
	signatureBlock := a.resolvePreparedBySignatureBlock(signerName)
	drawSignaturePDFLines(pdf, 130, pdf.GetY(), 60, 3.5, 6.6, signatureBlock, false)

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
		return "", fmt.Errorf("failed to save India credit note PDF: %w", err)
	}

	log.Printf("✅ India credit note PDF generated: %s", filePath)
	return filePath, nil
}

// ============================================================================
// INDIAN NUMBER FORMATTING (§0 G10)
// ============================================================================

// indianDigitGrouping formats amount with Indian digit grouping (lakh/crore:
// last 3 digits, then groups of 2) and 2-decimal paise precision, e.g.
// 1234567.89 -> "12,34,567.89". Used on every India-plane document; the BHD
// GCC formatting (fmt.Sprintf("%.3f", ...)) is untouched.
func indianDigitGrouping(amount float64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	whole := int64(amount)
	frac := int64(math.Round((amount - float64(whole)) * 100))
	if frac >= 100 { // rounding carried into the next rupee
		whole++
		frac -= 100
	}
	grouped := groupIndianDigits(fmt.Sprintf("%d", whole))
	result := fmt.Sprintf("%s.%02d", grouped, frac)
	if negative {
		result = "-" + result
	}
	return result
}

// groupIndianDigits inserts commas into a digit string using the Indian
// convention: the last 3 digits form one group, every group before that is 2
// digits (1234567 -> "12,34,567").
func groupIndianDigits(s string) string {
	if len(s) <= 3 {
		return s
	}
	last3 := s[len(s)-3:]
	rest := s[:len(s)-3]
	var parts []string
	for len(rest) > 2 {
		parts = append([]string{rest[len(rest)-2:]}, parts...)
		rest = rest[:len(rest)-2]
	}
	if rest != "" {
		parts = append([]string{rest}, parts...)
	}
	parts = append(parts, last3)
	return strings.Join(parts, ",")
}

// intToWordsIndian converts a non-negative integer to English words using
// Indian lakh/crore grouping, reusing the ones/teens/tens vocabulary from
// intToWords (amountInWords' BHD converter, invoice_pdf_service.go) so the
// two number-to-words paths never drift on digit naming.
func intToWordsIndian(n int64) string {
	if n == 0 {
		return ""
	}
	crore := n / 10000000
	n %= 10000000
	lakh := n / 100000
	n %= 100000
	thousand := n / 1000
	n %= 1000
	hundred := n

	var parts []string
	if crore > 0 {
		parts = append(parts, intToWords(int(crore)), "Crore")
	}
	if lakh > 0 {
		parts = append(parts, intToWords(int(lakh)), "Lakh")
	}
	if thousand > 0 {
		parts = append(parts, intToWords(int(thousand)), "Thousand")
	}
	if hundred > 0 {
		parts = append(parts, intToWords(int(hundred)))
	}
	return strings.Join(parts, " ")
}

// amountInWordsIndian converts an INR amount to words in Indian convention:
// "Rupees <words> and <words> Paise Only" (§0 G10). Mirrors amountInWords'
// BHD/Fils shape but with lakh/crore grouping and the "Rupees ... Only"
// phrasing customary on Indian invoices. Selected only at India-branch call
// sites — amountInWords (BHD) is byte-untouched.
func amountInWordsIndian(amount float64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	rupees := int64(amount)
	paise := int64(math.Round((amount - float64(rupees)) * 100))
	if paise >= 100 {
		rupees++
		paise -= 100
	}

	rupeeWords := intToWordsIndian(rupees)
	if rupeeWords == "" {
		rupeeWords = "Zero"
	}

	result := "Rupees " + rupeeWords
	if paise > 0 {
		result += " and " + intToWordsIndian(paise) + " Paise"
	}
	result += " Only"
	if negative {
		result = "Minus " + result
	}
	return result
}
