package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// =============================================================================
// E-INVOICE SERVICE (Phase 23)
//
// Generates UBL 2.1 compliant XML for each invoice (GCC e-invoicing aligned).
// Also provides VAT return data export for NBR compliance.
//
// Bahrain NBR mandating e-invoicing 2026-2027.
// =============================================================================

// phTradingCountry is the ISO 3166-1 alpha-2 country code used in UBL XML.
// The supplier name, TRN, address, and city are now read from the overlay
// via companyDocumentProfile so that per-division identity is correct.
const phTradingCountry = "BH"

// GenerateEInvoiceXML generates a UBL 2.1 compliant XML for a customer invoice
func (a *App) GenerateEInvoiceXML(invoiceID string) (string, error) {
	if err := a.requirePermission("invoices:view"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	// Load invoice with items
	var invoice Invoice
	if err := a.db.Preload("Items").Where("id = ?", invoiceID).First(&invoice).Error; err != nil {
		return "", fmt.Errorf("invoice not found: %w", err)
	}

	// Load customer for TRN and address
	var customer CustomerMaster
	if err := a.db.Where("id = ?", invoice.CustomerID).First(&customer).Error; err != nil {
		log.Printf("⚠️ Could not fetch customer for e-invoice: %v", err)
	}

	// Resolve supplier identity from the invoice's division — this is the key
	// fix: a Beacon Controls invoice must emit Beacon's TRN/name/address, not
	// Acme Instrumentation's hardcoded constants.
	supplierProfile := companyDocumentProfile(invoice.Division)
	supplierTRN := supplierProfile.VATNumber
	supplierName := supplierProfile.LegalName
	supplierCity := supplierProfile.City
	if supplierCity == "" {
		supplierCity = "Manama"
	}
	// Concatenate address lines into a single street string for UBL StreetName.
	supplierAddress := ""
	if len(supplierProfile.AddressLines) > 0 {
		supplierAddress = supplierProfile.AddressLines[0]
		for _, line := range supplierProfile.AddressLines[1:] {
			supplierAddress += ", " + line
		}
	}

	// Compute document hash (HMAC-SHA256 with salt, P1-4 fix)
	invoiceHash := computeDocumentHMAC(invoice.InvoiceNumber, invoice.InvoiceDate.Format("2006-01-02"), invoice.GrandTotalBHD, invoice.VATBHD)

	// Determine invoice type code
	typeCode := "388" // Tax Invoice
	if invoice.Status == "Proforma" {
		typeCode = "325" // Proforma
	}

	// ZATCA must report the invoice's true rate — including an explicit 0%
	// for zero-rated/export invoices. Legacy invoices with a VATBHD but no
	// stored VATPercent get the rate reconstructed rather than reported as 0%.
	vatPercent := effectiveInvoiceVATPercent(invoice)

	// Build XML
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"` + "\n")
	sb.WriteString(`         xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"` + "\n")
	sb.WriteString(`         xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"` + "\n")
	sb.WriteString(`         xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2">` + "\n")

	// UBL Extensions - Hash
	sb.WriteString("  <ext:UBLExtensions>\n")
	sb.WriteString("    <ext:UBLExtension>\n")
	sb.WriteString("      <ext:ExtensionContent>\n")
	sb.WriteString(fmt.Sprintf("        <InvoiceHash>%s</InvoiceHash>\n", invoiceHash))
	sb.WriteString("      </ext:ExtensionContent>\n")
	sb.WriteString("    </ext:UBLExtension>\n")
	sb.WriteString("  </ext:UBLExtensions>\n")

	// Header
	sb.WriteString(fmt.Sprintf("  <cbc:ID>%s</cbc:ID>\n", xmlEscape(invoice.InvoiceNumber)))
	sb.WriteString(fmt.Sprintf("  <cbc:IssueDate>%s</cbc:IssueDate>\n", invoice.InvoiceDate.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("  <cbc:DueDate>%s</cbc:DueDate>\n", invoice.DueDate.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("  <cbc:InvoiceTypeCode>%s</cbc:InvoiceTypeCode>\n", typeCode))
	sb.WriteString("  <cbc:DocumentCurrencyCode>BHD</cbc:DocumentCurrencyCode>\n")

	// Supplier Party — identity dispatched from invoice.Division via the overlay
	sb.WriteString("  <cac:AccountingSupplierParty>\n")
	sb.WriteString("    <cac:Party>\n")
	sb.WriteString("      <cac:PartyIdentification>\n")
	sb.WriteString(fmt.Sprintf("        <cbc:ID schemeID=\"TRN\">%s</cbc:ID>\n", xmlEscape(supplierTRN)))
	sb.WriteString("      </cac:PartyIdentification>\n")
	sb.WriteString("      <cac:PartyName>\n")
	sb.WriteString(fmt.Sprintf("        <cbc:Name>%s</cbc:Name>\n", xmlEscape(supplierName)))
	sb.WriteString("      </cac:PartyName>\n")
	sb.WriteString("      <cac:PostalAddress>\n")
	sb.WriteString(fmt.Sprintf("        <cbc:StreetName>%s</cbc:StreetName>\n", xmlEscape(supplierAddress)))
	sb.WriteString(fmt.Sprintf("        <cbc:CityName>%s</cbc:CityName>\n", xmlEscape(supplierCity)))
	sb.WriteString("        <cac:Country>\n")
	sb.WriteString(fmt.Sprintf("          <cbc:IdentificationCode>%s</cbc:IdentificationCode>\n", phTradingCountry))
	sb.WriteString("        </cac:Country>\n")
	sb.WriteString("      </cac:PostalAddress>\n")
	sb.WriteString("      <cac:PartyTaxScheme>\n")
	sb.WriteString(fmt.Sprintf("        <cbc:CompanyID>%s</cbc:CompanyID>\n", xmlEscape(supplierTRN)))
	sb.WriteString("        <cac:TaxScheme>\n")
	sb.WriteString("          <cbc:ID>VAT</cbc:ID>\n")
	sb.WriteString("        </cac:TaxScheme>\n")
	sb.WriteString("      </cac:PartyTaxScheme>\n")
	sb.WriteString("    </cac:Party>\n")
	sb.WriteString("  </cac:AccountingSupplierParty>\n")

	// Customer Party
	customerTRN := customer.TRN
	customerAddress := customer.AddressLine1
	customerCity := customer.City
	if customerCity == "" {
		customerCity = "Bahrain"
	}
	customerCountry := customer.Country
	if customerCountry == "" {
		customerCountry = "BH"
	}

	sb.WriteString("  <cac:AccountingCustomerParty>\n")
	sb.WriteString("    <cac:Party>\n")
	if customerTRN != "" {
		sb.WriteString("      <cac:PartyIdentification>\n")
		sb.WriteString(fmt.Sprintf("        <cbc:ID schemeID=\"TRN\">%s</cbc:ID>\n", xmlEscape(customerTRN)))
		sb.WriteString("      </cac:PartyIdentification>\n")
	}
	sb.WriteString("      <cac:PartyName>\n")
	sb.WriteString(fmt.Sprintf("        <cbc:Name>%s</cbc:Name>\n", xmlEscape(invoice.CustomerName)))
	sb.WriteString("      </cac:PartyName>\n")
	sb.WriteString("      <cac:PostalAddress>\n")
	if customerAddress != "" {
		sb.WriteString(fmt.Sprintf("        <cbc:StreetName>%s</cbc:StreetName>\n", xmlEscape(customerAddress)))
	}
	sb.WriteString(fmt.Sprintf("        <cbc:CityName>%s</cbc:CityName>\n", xmlEscape(customerCity)))
	sb.WriteString("        <cac:Country>\n")
	sb.WriteString(fmt.Sprintf("          <cbc:IdentificationCode>%s</cbc:IdentificationCode>\n", xmlEscape(customerCountry)))
	sb.WriteString("        </cac:Country>\n")
	sb.WriteString("      </cac:PostalAddress>\n")
	if customerTRN != "" {
		sb.WriteString("      <cac:PartyTaxScheme>\n")
		sb.WriteString(fmt.Sprintf("        <cbc:CompanyID>%s</cbc:CompanyID>\n", xmlEscape(customerTRN)))
		sb.WriteString("        <cac:TaxScheme>\n")
		sb.WriteString("          <cbc:ID>VAT</cbc:ID>\n")
		sb.WriteString("        </cac:TaxScheme>\n")
		sb.WriteString("      </cac:PartyTaxScheme>\n")
	}
	sb.WriteString("    </cac:Party>\n")
	sb.WriteString("  </cac:AccountingCustomerParty>\n")

	// Tax Total
	sb.WriteString("  <cac:TaxTotal>\n")
	sb.WriteString(fmt.Sprintf("    <cbc:TaxAmount currencyID=\"BHD\">%.3f</cbc:TaxAmount>\n", invoice.VATBHD))
	sb.WriteString("    <cac:TaxSubtotal>\n")
	sb.WriteString(fmt.Sprintf("      <cbc:TaxableAmount currencyID=\"BHD\">%.3f</cbc:TaxableAmount>\n", invoice.SubtotalBHD))
	sb.WriteString(fmt.Sprintf("      <cbc:TaxAmount currencyID=\"BHD\">%.3f</cbc:TaxAmount>\n", invoice.VATBHD))
	sb.WriteString("      <cac:TaxCategory>\n")
	sb.WriteString(fmt.Sprintf("        <cbc:Percent>%.1f</cbc:Percent>\n", vatPercent))
	sb.WriteString("        <cac:TaxScheme>\n")
	sb.WriteString("          <cbc:ID>VAT</cbc:ID>\n")
	sb.WriteString("        </cac:TaxScheme>\n")
	sb.WriteString("      </cac:TaxCategory>\n")
	sb.WriteString("    </cac:TaxSubtotal>\n")
	sb.WriteString("  </cac:TaxTotal>\n")

	// Legal Monetary Total
	sb.WriteString("  <cac:LegalMonetaryTotal>\n")
	sb.WriteString(fmt.Sprintf("    <cbc:LineExtensionAmount currencyID=\"BHD\">%.3f</cbc:LineExtensionAmount>\n", invoice.SubtotalBHD))
	sb.WriteString(fmt.Sprintf("    <cbc:TaxExclusiveAmount currencyID=\"BHD\">%.3f</cbc:TaxExclusiveAmount>\n", invoice.SubtotalBHD))
	sb.WriteString(fmt.Sprintf("    <cbc:TaxInclusiveAmount currencyID=\"BHD\">%.3f</cbc:TaxInclusiveAmount>\n", invoice.GrandTotalBHD))
	sb.WriteString(fmt.Sprintf("    <cbc:PayableAmount currencyID=\"BHD\">%.3f</cbc:PayableAmount>\n", invoice.GrandTotalBHD))
	sb.WriteString("  </cac:LegalMonetaryTotal>\n")

	// Invoice Lines
	for _, item := range invoice.Items {
		lineTotal := item.Quantity * item.Rate
		lineTax := lineTotal * (vatPercent / 100.0)

		sb.WriteString("  <cac:InvoiceLine>\n")
		sb.WriteString(fmt.Sprintf("    <cbc:ID>%d</cbc:ID>\n", item.LineNumber))
		sb.WriteString(fmt.Sprintf("    <cbc:InvoicedQuantity unitCode=\"EA\">%.2f</cbc:InvoicedQuantity>\n", item.Quantity))
		sb.WriteString(fmt.Sprintf("    <cbc:LineExtensionAmount currencyID=\"BHD\">%.3f</cbc:LineExtensionAmount>\n", lineTotal))
		sb.WriteString("    <cac:Item>\n")
		sb.WriteString(fmt.Sprintf("      <cbc:Description>%s</cbc:Description>\n", xmlEscape(item.Description)))
		if item.ProductCode != "" {
			sb.WriteString("      <cac:SellersItemIdentification>\n")
			sb.WriteString(fmt.Sprintf("        <cbc:ID>%s</cbc:ID>\n", xmlEscape(item.ProductCode)))
			sb.WriteString("      </cac:SellersItemIdentification>\n")
		}
		sb.WriteString("      <cac:ClassifiedTaxCategory>\n")
		sb.WriteString(fmt.Sprintf("        <cbc:Percent>%.1f</cbc:Percent>\n", vatPercent))
		sb.WriteString("        <cac:TaxScheme>\n")
		sb.WriteString("          <cbc:ID>VAT</cbc:ID>\n")
		sb.WriteString("        </cac:TaxScheme>\n")
		sb.WriteString("      </cac:ClassifiedTaxCategory>\n")
		sb.WriteString("    </cac:Item>\n")
		sb.WriteString("    <cac:Price>\n")
		sb.WriteString(fmt.Sprintf("      <cbc:PriceAmount currencyID=\"BHD\">%.3f</cbc:PriceAmount>\n", item.Rate))
		sb.WriteString("    </cac:Price>\n")
		sb.WriteString("    <cac:TaxTotal>\n")
		sb.WriteString(fmt.Sprintf("      <cbc:TaxAmount currencyID=\"BHD\">%.3f</cbc:TaxAmount>\n", lineTax))
		sb.WriteString("    </cac:TaxTotal>\n")
		sb.WriteString("  </cac:InvoiceLine>\n")
	}

	sb.WriteString("</Invoice>\n")

	// Save XML file alongside PDFs
	paths := a.getAppPaths()
	if paths == nil {
		return "", fmt.Errorf("application paths not available")
	}
	cleanNum := filepath.Base(strings.ReplaceAll(invoice.InvoiceNumber, "..", ""))
	cleanNum = strings.ReplaceAll(cleanNum, "/", "_")
	cleanNum = strings.ReplaceAll(cleanNum, "\\", "_")
	xmlFilename := fmt.Sprintf("%s.xml", cleanNum)

	docYear := invoice.InvoiceDate.Year()
	if docYear <= 0 {
		docYear = time.Now().Year()
	}
	exportDir := a.getExportDir("customer", customer.BusinessName, "MISC", docYear)
	xmlPath := filepath.Join(exportDir, xmlFilename)

	if err := os.WriteFile(xmlPath, []byte(sb.String()), 0640); err != nil {
		return "", fmt.Errorf("failed to write e-invoice XML: %w", err)
	}

	log.Printf("✅ E-Invoice XML generated: %s", xmlPath)
	return xmlPath, nil
}

// vatReturnBucket accumulates one division's (one TRN's) VAT-return figures.
type vatReturnBucket struct {
	standardRatedSupplies, vatCollected, zeroRated, exempt float64
	cnDeductions, cnVATDeductions                         float64
	invoiceCount, creditNoteCount                         int
}

// ExportVATReturnData exports VAT return data for a specific quarter as CSV,
// ONE CSV per configured division (Wave 12.5, owner-sanctioned). Each division
// is a distinct VAT-registered legal entity with its own TRN, so it files its
// own return covering only its own supplies — filing a division's sales under
// another division's TRN would mis-report to the NBR. Every invoice/credit note
// is attributed to exactly one division (empty/unknown division falls back to
// the default, so nothing is silently dropped). For a single-division
// deployment this emits exactly one CSV with all supplies under that TRN —
// byte-identical in content to the pre-Wave-12.5 aggregate. Returns the export
// directory holding the per-division CSVs.
func (a *App) ExportVATReturnData(year, quarter int) (string, error) {
	if err := a.requirePermission("finance:read"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}
	if quarter < 1 || quarter > 4 {
		return "", fmt.Errorf("quarter must be between 1 and 4")
	}
	if year < 2020 || year > 2100 {
		return "", fmt.Errorf("year must be between 2020 and 2100")
	}

	// Calculate period dates
	startMonth := time.Month((quarter-1)*3 + 1)
	startDate := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 3, 0)

	// Query invoices for the period
	var invoices []Invoice
	if err := a.db.Where("invoice_date >= ? AND invoice_date < ? AND status NOT IN ?",
		startDate, endDate, []string{"Cancelled", "Void", "Proforma", "Draft"}).
		Find(&invoices).Error; err != nil {
		return "", fmt.Errorf("failed to query invoices: %w", err)
	}

	// Query credit notes for the period
	// Only Applied credit notes should reduce VAT liability (not Draft or Issued-but-unapplied)
	var creditNotes []CreditNote
	if err := a.db.Where("applied_at >= ? AND applied_at < ? AND status = ?",
		startDate, endDate, "Applied").
		Find(&creditNotes).Error; err != nil {
		return "", fmt.Errorf("failed to query credit notes: %w", err)
	}

	// Bucket every invoice/credit note by its (normalized) division so each
	// division's return reflects only its own TRN's supplies. normalizeDivisionName
	// maps empty/unknown values to the default division, so no invoice is dropped.
	buckets := map[string]*vatReturnBucket{}
	bucketFor := func(key string) *vatReturnBucket {
		b := buckets[key]
		if b == nil {
			b = &vatReturnBucket{}
			buckets[key] = b
		}
		return b
	}

	for _, inv := range invoices {
		b := bucketFor(normalizeDivisionName(inv.Division))
		if inv.VATPercent > 0 {
			b.standardRatedSupplies += inv.SubtotalBHD
			b.vatCollected += inv.VATBHD
		} else {
			b.zeroRated += inv.SubtotalBHD
		}
		b.invoiceCount++
	}

	for _, cn := range creditNotes {
		// Credit notes carry no Division of their own — resolve it from the
		// linked invoice's division (chain lookup) so the deduction lands on
		// the same TRN that reported the original supply.
		b := bucketFor(a.resolveCreditNoteDivision(cn))
		b.cnDeductions += cn.SubtotalBHD
		b.cnVATDeductions += cn.VATBHD
		b.creditNoteCount++
	}

	paths := a.getAppPaths()
	if paths == nil {
		return "", fmt.Errorf("application paths not available")
	}
	exportDir := a.getExportDir("report", "", "", year)

	// Emit one CSV per configured division (a registered TRN files a return even
	// in a nil quarter). Iterating activeOverlay.Divisions keeps output
	// deterministic and covers every registration.
	for _, division := range activeOverlay.Divisions {
		b := buckets[division.Key]
		if b == nil {
			b = &vatReturnBucket{}
		}
		if err := a.writeVATReturnCSV(exportDir, division.Key, year, quarter, startDate, endDate, b); err != nil {
			return "", err
		}
	}

	log.Printf("✅ VAT Return CSVs exported: %s (Q%d %d: %d invoices, %d credit notes across %d division TRNs)",
		exportDir, quarter, year, len(invoices), len(creditNotes), len(activeOverlay.Divisions))
	return exportDir, nil
}

// writeVATReturnCSV writes one division's VAT-return CSV, stamped with that
// division's TRN and legal name.
func (a *App) writeVATReturnCSV(exportDir, division string, year, quarter int, startDate, endDate time.Time, b *vatReturnBucket) error {
	profile := companyDocumentProfile(division)

	// Net figures
	netSupplies := b.standardRatedSupplies - b.cnDeductions
	netVAT := b.vatCollected - b.cnVATDeductions

	filename := fmt.Sprintf("VAT_Return_Q%d_%d_%s.csv", quarter, year, sanitizeFilename(profile.Division))
	csvPath := filepath.Join(exportDir, filename)

	file, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	// Header
	writer.Write([]string{"Category", "Amount (BHD)", "VAT (BHD)", "Notes"})

	// Data rows
	// Label carries the overlay VAT rate (byte-identical "(10%)" for the
	// built-in default) so a different-rate deployment's return is truthful.
	writer.Write([]string{fmt.Sprintf("Standard-Rated Supplies (%g%%)", activeOverlay.DefaultVATRate), fmt.Sprintf("%.3f", b.standardRatedSupplies), fmt.Sprintf("%.3f", b.vatCollected), fmt.Sprintf("%d invoices", b.invoiceCount)})
	writer.Write([]string{"Credit Notes", fmt.Sprintf("-%.3f", b.cnDeductions), fmt.Sprintf("-%.3f", b.cnVATDeductions), fmt.Sprintf("%d credit notes", b.creditNoteCount)})
	writer.Write([]string{"Net Standard-Rated Supplies", fmt.Sprintf("%.3f", netSupplies), fmt.Sprintf("%.3f", netVAT), ""})
	writer.Write([]string{"Zero-Rated Supplies", fmt.Sprintf("%.3f", b.zeroRated), "0.000", ""})
	writer.Write([]string{"Exempt Supplies", fmt.Sprintf("%.3f", b.exempt), "0.000", ""})
	writer.Write([]string{"", "", "", ""})
	writer.Write([]string{"TOTAL OUTPUT VAT", "", fmt.Sprintf("%.3f", netVAT), fmt.Sprintf("Period: Q%d %d", quarter, year)})
	writer.Write([]string{"", "", "", ""})
	writer.Write([]string{"Division", profile.Division, "", ""})
	writer.Write([]string{"TRN", profile.VATNumber, "", profile.LegalName})
	writer.Write([]string{"Period", fmt.Sprintf("Q%d %d", quarter, year), fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.AddDate(0, 0, -1).Format("2006-01-02")), ""})

	// Flush CSV writer and check for write errors
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("CSV write error: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync CSV file: %w", err)
	}

	log.Printf("✅ VAT Return CSV exported: %s (%s TRN %s, Q%d %d: %d invoices, %d credit notes)",
		csvPath, profile.Division, profile.VATNumber, quarter, year, b.invoiceCount, b.creditNoteCount)
	return nil
}

// xmlEscape escapes special XML characters
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
