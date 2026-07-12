package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/xuri/excelize/v2"
)

// GenerateDataImportTemplate creates a multi-sheet Excel template for client data onboarding.
// The client fills this out and returns it for import into AsymmFlow.
func (a *App) GenerateDataImportTemplate() (string, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return "", err
	}

	f := excelize.NewFile()
	defer f.Close()

	// =========================================================================
	// STYLES
	// =========================================================================

	// Title style - dark blue background, white bold text, large font
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "FFFFFF", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1B2A4A"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "bottom", Color: "E8A838", Style: 2},
		},
	})

	// Section header style - gold accent
	sectionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "1B2A4A", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8A838"}},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	})

	// Column header style - dark header with white text
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"2C3E50"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "95A5A6", Style: 1},
			{Type: "right", Color: "95A5A6", Style: 1},
			{Type: "top", Color: "95A5A6", Style: 1},
			{Type: "bottom", Color: "95A5A6", Style: 1},
		},
	})

	// Required column header - red accent to show mandatory
	requiredHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"C0392B"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "95A5A6", Style: 1},
			{Type: "right", Color: "95A5A6", Style: 1},
			{Type: "top", Color: "95A5A6", Style: 1},
			{Type: "bottom", Color: "95A5A6", Style: 1},
		},
	})

	// Example data style - light blue background, italic
	exampleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Size: 10, Color: "7F8C8D", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"EBF5FB"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "D5DBDB", Style: 1},
			{Type: "right", Color: "D5DBDB", Style: 1},
			{Type: "top", Color: "D5DBDB", Style: 1},
			{Type: "bottom", Color: "D5DBDB", Style: 1},
		},
	})

	// Normal data cell style
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "Calibri"},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "D5DBDB", Style: 1},
			{Type: "right", Color: "D5DBDB", Style: 1},
			{Type: "top", Color: "D5DBDB", Style: 1},
			{Type: "bottom", Color: "D5DBDB", Style: 1},
		},
	})

	// Note/instruction style - yellow background
	noteStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "856404", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFF3CD"}},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "FFEEBA", Style: 1},
			{Type: "right", Color: "FFEEBA", Style: 1},
			{Type: "top", Color: "FFEEBA", Style: 1},
			{Type: "bottom", Color: "FFEEBA", Style: 1},
		},
	})

	// =========================================================================
	// SHEET 1: INSTRUCTIONS
	// =========================================================================
	instrSheet := "Instructions"
	f.SetSheetName("Sheet1", instrSheet)

	f.SetColWidth(instrSheet, "A", "A", 80)
	f.SetRowHeight(instrSheet, 1, 45)

	f.MergeCell(instrSheet, "A1", "A1")
	f.SetCellValue(instrSheet, "A1", "AsymmFlow - Data Import Template")
	f.SetCellStyle(instrSheet, "A1", "A1", titleStyle)

	instructions := []struct {
		row   int
		style int
		text  string
	}{
		{3, sectionStyle, "OVERVIEW"},
		{4, noteStyle, "This workbook contains sheets for importing master data and transactions into AsymmFlow ERP."},
		{5, noteStyle, "Please fill in the data sheets following the guidelines below, then return this file for import."},
		{7, sectionStyle, "IMPORT ORDER (IMPORTANT)"},
		{8, noteStyle, "1. Customers  - Fill first (other sheets reference customers)"},
		{9, noteStyle, "2. Customer Contacts  - References customer_code from Customers sheet"},
		{10, noteStyle, "3. Suppliers  - Fill second (products and POs reference suppliers)"},
		{11, noteStyle, "4. Supplier Contacts  - References supplier_code from Suppliers sheet"},
		{12, noteStyle, "5. Products  - References supplier_code from Suppliers sheet"},
		{13, noteStyle, "6. Orders  - References customer_code from Customers sheet"},
		{14, noteStyle, "7. Invoices  - References customer_code and order_number"},
		{15, noteStyle, "8. Payments  - References invoice_number from Invoices sheet"},
		{17, sectionStyle, "COLUMN GUIDE"},
		{18, noteStyle, "RED header columns are REQUIRED - these must be filled for every row."},
		{19, noteStyle, "DARK header columns are OPTIONAL - fill if data is available."},
		{20, noteStyle, "Row 2 (light blue italic) is an EXAMPLE row - delete it before submitting."},
		{22, sectionStyle, "DATA FORMATTING RULES"},
		{23, noteStyle, "Dates: Use DD/MM/YYYY format (e.g., 15/03/2025)"},
		{24, noteStyle, "Amounts: Use numbers without commas (e.g., 15250.500, not 15,250.500)"},
		{25, noteStyle, "Currency: BHD uses 3 decimal places (e.g., 1234.567)"},
		{26, noteStyle, "Codes: Use SHORT UPPERCASE codes (e.g., CUST001, SUP-RI, PROD-FT10)"},
		{27, noteStyle, "Booleans: Use TRUE or FALSE (not Yes/No or 1/0)"},
		{29, sectionStyle, "DROPDOWN FIELDS"},
		{30, noteStyle, "Some columns have dropdown validation (click the cell to see options)."},
		{31, noteStyle, "If your value isn't in the dropdown, type it manually - we'll review during import."},
		{33, sectionStyle, "TIPS"},
		{34, noteStyle, "- Keep customer_code / supplier_code consistent across all sheets"},
		{35, noteStyle, "- Do NOT change column headers or sheet names"},
		{36, noteStyle, "- You can add as many rows as needed below the example row"},
		{37, noteStyle, "- Leave optional fields blank if data is not available (do NOT put N/A or -)"},
		{38, noteStyle, fmt.Sprintf("- Template generated: %s", time.Now().Format("02 Jan 2006 15:04"))},
		{40, sectionStyle, "SUPPORT"},
		{41, noteStyle, "For questions about this template, contact: admin@asymmetrica.co"},
	}

	for _, instr := range instructions {
		f.SetCellValue(instrSheet, fmt.Sprintf("A%d", instr.row), instr.text)
		f.SetCellStyle(instrSheet, fmt.Sprintf("A%d", instr.row), fmt.Sprintf("A%d", instr.row), instr.style)
		f.SetRowHeight(instrSheet, instr.row, 22)
	}

	// =========================================================================
	// SHEET 2: CUSTOMERS
	// =========================================================================
	custSheet := "Customers"
	f.NewSheet(custSheet)

	custHeaders := []struct {
		col      string
		name     string
		width    float64
		required bool
	}{
		{"A", "customer_code", 18, true},
		{"B", "business_name", 35, true},
		{"C", "customer_type", 18, false},
		{"D", "address_line1", 35, false},
		{"E", "city", 15, false},
		{"F", "country", 15, false},
		{"G", "trn (Tax ID)", 18, false},
		{"H", "industry", 20, false},
		{"I", "payment_terms_days", 18, false},
		{"J", "credit_limit_bhd", 18, false},
		{"K", "payment_grade", 15, false},
		{"L", "customer_grade", 15, false},
	}

	writeSheetHeaders(f, custSheet, custHeaders, headerStyle, requiredHeaderStyle, titleStyle)

	// Example row
	custExample := []string{
		"NPC-001", "National Petroleum Co.", "Corporate",
		"PO Box 0000, Manama", "Manama", "Bahrain",
		"123456789", "Oil & Gas", "30", "50000.000", "A", "A",
	}
	writeExampleRow(f, custSheet, custExample, exampleStyle, len(custHeaders))

	// Apply data style to rows 3-100
	applyDataStyle(f, custSheet, dataStyle, len(custHeaders), 3, 100)

	// Dropdowns
	addDropdown(f, custSheet, "C3:C500", `"Corporate,Government,SME,Individual,Joint Venture"`)
	addDropdown(f, custSheet, "K3:K500", `"A,B,C,D"`)
	addDropdown(f, custSheet, "L3:L500", `"A,B,C,D"`)

	// =========================================================================
	// SHEET 3: CUSTOMER CONTACTS
	// =========================================================================
	ccSheet := "Customer Contacts"
	f.NewSheet(ccSheet)

	ccHeaders := []struct {
		col      string
		name     string
		width    float64
		required bool
	}{
		{"A", "customer_code", 18, true},
		{"B", "contact_name", 30, true},
		{"C", "job_title", 25, false},
		{"D", "email", 30, false},
		{"E", "phone", 20, false},
		{"F", "address", 40, false},
		{"G", "is_primary_contact", 18, false},
	}

	writeSheetHeaders(f, ccSheet, ccHeaders, headerStyle, requiredHeaderStyle, titleStyle)

	ccExample := []string{
		"NPC-001", "Pat Morgan", "Procurement Manager",
		"pat.morgan@nationalpetroleum.example", "+973-1700-0000", "PO Box 0000, Manama", "TRUE",
	}
	writeExampleRow(f, ccSheet, ccExample, exampleStyle, len(ccHeaders))
	applyDataStyle(f, ccSheet, dataStyle, len(ccHeaders), 3, 100)
	addDropdown(f, ccSheet, "G3:G500", `"TRUE,FALSE"`)

	// =========================================================================
	// SHEET 4: SUPPLIERS
	// =========================================================================
	supSheet := "Suppliers"
	f.NewSheet(supSheet)

	supHeaders := []struct {
		col      string
		name     string
		width    float64
		required bool
	}{
		{"A", "supplier_code", 18, true},
		{"B", "supplier_name", 35, true},
		{"C", "country", 15, false},
		{"D", "supplier_type", 18, false},
		{"E", "brands_handled", 30, false},
		{"F", "primary_contact", 25, false},
		{"G", "email", 30, false},
		{"H", "phone", 20, false},
		{"I", "address", 40, false},
		{"J", "payment_terms", 18, false},
		{"K", "lead_time_days", 15, false},
		{"L", "bank_name", 25, false},
		{"M", "iban", 30, false},
		{"N", "swift_code", 15, false},
		{"O", "tax_id", 18, false},
	}

	writeSheetHeaders(f, supSheet, supHeaders, headerStyle, requiredHeaderStyle, titleStyle)

	supExample := []string{
		"SUP-RI", "Rhine Instruments AG", "Switzerland", "Manufacturer",
		"Rhine Instruments", "Hans Mueller", "hans.mueller@rhineinstruments.example", "+41-00-000-0000",
		"1 Rhine Strasse, Basel", "Net 60", "45",
		"Demo Bank D", "CH00DEMO00000000000000", "DEMOCHZZXXX", "CHE-000.000.000",
	}
	writeExampleRow(f, supSheet, supExample, exampleStyle, len(supHeaders))
	applyDataStyle(f, supSheet, dataStyle, len(supHeaders), 3, 100)
	addDropdown(f, supSheet, "D3:D500", `"Manufacturer,Distributor,Agent,Service Provider"`)

	// =========================================================================
	// SHEET 5: SUPPLIER CONTACTS
	// =========================================================================
	scSheet := "Supplier Contacts"
	f.NewSheet(scSheet)

	scHeaders := []struct {
		col      string
		name     string
		width    float64
		required bool
	}{
		{"A", "supplier_code", 18, true},
		{"B", "contact_name", 30, true},
		{"C", "job_title", 25, false},
		{"D", "email", 30, false},
		{"E", "phone", 20, false},
		{"F", "address", 40, false},
		{"G", "is_primary_contact", 18, false},
	}

	writeSheetHeaders(f, scSheet, scHeaders, headerStyle, requiredHeaderStyle, titleStyle)

	scExample := []string{
		"SUP-RI", "Hans Mueller", "Regional Sales Manager",
		"hans.mueller@rhineinstruments.example", "+41-00-000-0000", "1 Rhine Strasse, Basel", "TRUE",
	}
	writeExampleRow(f, scSheet, scExample, exampleStyle, len(scHeaders))
	applyDataStyle(f, scSheet, dataStyle, len(scHeaders), 3, 100)
	addDropdown(f, scSheet, "G3:G500", `"TRUE,FALSE"`)

	// =========================================================================
	// SHEET 6: PRODUCTS
	// =========================================================================
	prodSheet := "Products"
	f.NewSheet(prodSheet)

	prodHeaders := []struct {
		col      string
		name     string
		width    float64
		required bool
	}{
		{"A", "product_code", 20, true},
		{"B", "product_name", 35, true},
		{"C", "product_category", 20, false},
		{"D", "supplier_code", 18, true},
		{"E", "part_number", 20, false},
		{"F", "description", 40, false},
		{"G", "standard_cost_bhd", 18, false},
		{"H", "standard_price_bhd", 18, false},
		{"I", "unit_of_measure", 15, false},
		{"J", "hs_code", 15, false},
	}

	writeSheetHeaders(f, prodSheet, prodHeaders, headerStyle, requiredHeaderStyle, titleStyle)

	prodExample := []string{
		"RI-RLT60", "RadarLine RLT60 Radar Level", "Level Measurement",
		"SUP-RI", "RLT60-AAACCAAPK2", "80 GHz radar level transmitter for liquids",
		"2500.000", "4200.000", "Each", "9026.10",
	}
	writeExampleRow(f, prodSheet, prodExample, exampleStyle, len(prodHeaders))
	applyDataStyle(f, prodSheet, dataStyle, len(prodHeaders), 3, 100)
	addDropdown(f, prodSheet, "C3:C500", `"Level Measurement,Flow Measurement,Pressure Measurement,Temperature Measurement,Analytics,Gas Analysis,Electrical,Services,Spare Parts,Other"`)
	addDropdown(f, prodSheet, "I3:I500", `"Each,Set,Meter,Kg,Lot,Box"`)

	// =========================================================================
	// SHEET 7: ORDERS
	// =========================================================================
	orderSheet := "Orders"
	f.NewSheet(orderSheet)

	orderHeaders := []struct {
		col      string
		name     string
		width    float64
		required bool
	}{
		{"A", "order_number", 18, true},
		{"B", "customer_code", 18, true},
		{"C", "customer_po_number", 20, false},
		{"D", "order_date (DD/MM/YYYY)", 22, true},
		{"E", "required_date (DD/MM/YYYY)", 22, false},
		{"F", "status", 18, false},
		{"G", "total_value_bhd", 18, false},
		{"H", "payment_terms", 25, false},
		{"I", "delivery_terms", 25, false},
		{"J", "customer_reference", 20, false},
		{"K", "attention_person", 25, false},
	}

	writeSheetHeaders(f, orderSheet, orderHeaders, headerStyle, requiredHeaderStyle, titleStyle)

	orderExample := []string{
		"ORD-2025-001", "NPC-001", "NPC-PO-44521",
		"15/03/2025", "30/04/2025", "Processing",
		"42500.000", "30 days from delivery", "DAP Bahrain",
		"RFQ-2025-188", "Pat Morgan",
	}
	writeExampleRow(f, orderSheet, orderExample, exampleStyle, len(orderHeaders))
	applyDataStyle(f, orderSheet, dataStyle, len(orderHeaders), 3, 100)
	addDropdown(f, orderSheet, "F3:F500", `"Processing,Confirmed,Shipped,Delivered,Completed,Cancelled"`)

	// =========================================================================
	// SHEET 8: INVOICES
	// =========================================================================
	invSheet := "Invoices"
	f.NewSheet(invSheet)

	invHeaders := []struct {
		col      string
		name     string
		width    float64
		required bool
	}{
		{"A", "invoice_number", 20, true},
		{"B", "invoice_date (DD/MM/YYYY)", 22, true},
		{"C", "customer_code", 18, true},
		{"D", "order_number", 18, false},
		{"E", "customer_po_number", 20, false},
		{"F", "subtotal_bhd", 15, false},
		{"G", "vat_percent", 12, false},
		{"H", "vat_bhd", 12, false},
		{"I", "grand_total_bhd", 18, true},
		{"J", "status", 15, false},
		{"K", "outstanding_bhd", 18, false},
		{"L", "due_date (DD/MM/YYYY)", 22, false},
		{"M", "payment_terms", 25, false},
		{"N", "mode_of_payment", 20, false},
	}

	writeSheetHeaders(f, invSheet, invHeaders, headerStyle, requiredHeaderStyle, titleStyle)

	invExample := []string{
		"INV-2025-001", "20/03/2025", "NPC-001",
		"ORD-2025-001", "NPC-PO-44521",
		"38636.364", "10", "3863.636", "42500.000",
		"Sent", "42500.000", "19/04/2025",
		"30 days from delivery", "Bank Transfer",
	}
	writeExampleRow(f, invSheet, invExample, exampleStyle, len(invHeaders))
	applyDataStyle(f, invSheet, dataStyle, len(invHeaders), 3, 100)
	addDropdown(f, invSheet, "J3:J500", `"Draft,Sent,Paid,PartiallyPaid,Overdue,Cancelled"`)
	addDropdown(f, invSheet, "N3:N500", `"Bank Transfer,Cheque,Cash,LC,Credit Card,PDC"`)

	// =========================================================================
	// SHEET 9: PAYMENTS
	// =========================================================================
	paySheet := "Payments"
	f.NewSheet(paySheet)

	payHeaders := []struct {
		col      string
		name     string
		width    float64
		required bool
	}{
		{"A", "invoice_number", 20, true},
		{"B", "payment_date (DD/MM/YYYY)", 22, true},
		{"C", "amount_bhd", 18, true},
		{"D", "payment_method", 18, false},
		{"E", "reference", 25, false},
	}

	writeSheetHeaders(f, paySheet, payHeaders, headerStyle, requiredHeaderStyle, titleStyle)

	payExample := []string{
		"INV-2025-001", "15/04/2025", "42500.000",
		"Bank Transfer", "TRF-2025-04-15-NPC",
	}
	writeExampleRow(f, paySheet, payExample, exampleStyle, len(payHeaders))
	applyDataStyle(f, paySheet, dataStyle, len(payHeaders), 3, 100)
	addDropdown(f, paySheet, "D3:D500", `"Cash,Cheque,Bank Transfer,Credit Card,LC,PDC,Other"`)

	// =========================================================================
	// SHEET 10: PURCHASE ORDERS (Bonus - useful for ops data)
	// =========================================================================
	poSheet := "Purchase Orders"
	f.NewSheet(poSheet)

	poHeaders := []struct {
		col      string
		name     string
		width    float64
		required bool
	}{
		{"A", "po_number", 18, true},
		{"B", "supplier_code", 18, true},
		{"C", "order_number (internal)", 22, false},
		{"D", "po_date (DD/MM/YYYY)", 22, true},
		{"E", "expected_delivery (DD/MM/YYYY)", 25, false},
		{"F", "currency", 12, false},
		{"G", "subtotal_bhd", 15, false},
		{"H", "vat_amount_bhd", 15, false},
		{"I", "total_bhd", 15, true},
		{"J", "status", 18, false},
		{"K", "payment_terms", 25, false},
	}

	writeSheetHeaders(f, poSheet, poHeaders, headerStyle, requiredHeaderStyle, titleStyle)

	poExample := []string{
		"PO-2025-001", "SUP-RI", "ORD-2025-001",
		"18/03/2025", "30/04/2025", "EUR",
		"22727.273", "2272.727", "25000.000",
		"Sent", "Net 60",
	}
	writeExampleRow(f, poSheet, poExample, exampleStyle, len(poHeaders))
	applyDataStyle(f, poSheet, dataStyle, len(poHeaders), 3, 100)
	addDropdown(f, poSheet, "F3:F500", `"BHD,USD,EUR,GBP,CHF,SAR,AED"`)
	addDropdown(f, poSheet, "J3:J500", `"Draft,Approved,Sent,Acknowledged,Partially Received,Received,Cancelled"`)

	// =========================================================================
	// SHEET 11: SUPPLIER INVOICES
	// =========================================================================
	siSheet := "Supplier Invoices"
	f.NewSheet(siSheet)

	siHeaders := []struct {
		col      string
		name     string
		width    float64
		required bool
	}{
		{"A", "invoice_number", 20, true},
		{"B", "supplier_code", 18, true},
		{"C", "po_number", 18, false},
		{"D", "invoice_date (DD/MM/YYYY)", 22, true},
		{"E", "due_date (DD/MM/YYYY)", 22, false},
		{"F", "currency", 12, false},
		{"G", "total_bhd", 15, true},
		{"H", "status", 15, false},
		{"I", "payment_status", 15, false},
	}

	writeSheetHeaders(f, siSheet, siHeaders, headerStyle, requiredHeaderStyle, titleStyle)

	siExample := []string{
		"RI-INV-2025-4421", "SUP-RI", "PO-2025-001",
		"25/04/2025", "24/06/2025", "EUR",
		"25000.000", "Approved", "Unpaid",
	}
	writeExampleRow(f, siSheet, siExample, exampleStyle, len(siHeaders))
	applyDataStyle(f, siSheet, dataStyle, len(siHeaders), 3, 100)
	addDropdown(f, siSheet, "F3:F500", `"BHD,USD,EUR,GBP,CHF,SAR,AED"`)
	addDropdown(f, siSheet, "H3:H500", `"Pending,Approved,Rejected,Paid,Dispute"`)
	addDropdown(f, siSheet, "I3:I500", `"Unpaid,Scheduled,Paid"`)

	// =========================================================================
	// SHEET 12: SUPPLIER PAYMENTS
	// =========================================================================
	spSheet := "Supplier Payments"
	f.NewSheet(spSheet)

	spHeaders := []struct {
		col      string
		name     string
		width    float64
		required bool
	}{
		{"A", "supplier_invoice_number", 22, true},
		{"B", "supplier_code", 18, true},
		{"C", "payment_date (DD/MM/YYYY)", 22, true},
		{"D", "amount_bhd", 18, true},
		{"E", "currency", 12, false},
		{"F", "payment_method", 18, false},
		{"G", "reference", 25, false},
		{"H", "notes", 35, false},
	}

	writeSheetHeaders(f, spSheet, spHeaders, headerStyle, requiredHeaderStyle, titleStyle)

	spExample := []string{
		"RI-INV-2025-4421", "SUP-RI", "20/06/2025",
		"25000.000", "EUR", "Bank Transfer",
		"TT-2025-06-20-RI", "Payment for RLT60 order",
	}
	writeExampleRow(f, spSheet, spExample, exampleStyle, len(spHeaders))
	applyDataStyle(f, spSheet, dataStyle, len(spHeaders), 3, 100)
	addDropdown(f, spSheet, "E3:E500", `"BHD,USD,EUR,GBP,CHF,SAR,AED"`)
	addDropdown(f, spSheet, "F3:F500", `"Bank Transfer,Cheque,LC,Cash,Wire Transfer,PDC,Other"`)

	// =========================================================================
	// Set active sheet to Instructions
	// =========================================================================
	instrIdx, _ := f.GetSheetIndex(instrSheet)
	f.SetActiveSheet(instrIdx)

	// =========================================================================
	// SAVE FILE
	// =========================================================================

	// Determine output path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	desktopDir := filepath.Join(homeDir, "Desktop")
	if _, err := os.Stat(desktopDir); os.IsNotExist(err) {
		desktopDir = homeDir
	}

	timestamp := time.Now().Format("2006_01_02")
	fileName := fmt.Sprintf("AsymmFlow_Data_Import_Template_%s.xlsx", timestamp)
	outputPath := filepath.Join(desktopDir, fileName)

	if err := f.SaveAs(outputPath); err != nil {
		return "", fmt.Errorf("failed to save template: %w", err)
	}

	log.Printf("Data import template generated: %s", outputPath)
	return outputPath, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

type sheetHeader struct {
	col      string
	name     string
	width    float64
	required bool
}

func writeSheetHeaders(f *excelize.File, sheet string, headers []struct {
	col      string
	name     string
	width    float64
	required bool
}, headerStyle, requiredHeaderStyle, titleStyle int) {

	// Title row
	lastCol := headers[len(headers)-1].col
	f.MergeCell(sheet, "A1", lastCol+"1")
	f.SetCellValue(sheet, "A1", fmt.Sprintf("  %s", sheet))
	f.SetCellStyle(sheet, "A1", lastCol+"1", titleStyle)
	f.SetRowHeight(sheet, 1, 35)

	// Header row
	f.SetRowHeight(sheet, 2, 28)
	for _, h := range headers {
		cell := h.col + "2"
		f.SetCellValue(sheet, cell, h.name)
		f.SetColWidth(sheet, h.col, h.col, h.width)
		if h.required {
			f.SetCellStyle(sheet, cell, cell, requiredHeaderStyle)
		} else {
			f.SetCellStyle(sheet, cell, cell, headerStyle)
		}
	}
}

func writeExampleRow(f *excelize.File, sheet string, values []string, style int, colCount int) {
	f.SetRowHeight(sheet, 3, 22)
	for i, val := range values {
		if i >= colCount {
			break
		}
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		cell := colLetter + "3"
		f.SetCellValue(sheet, cell, val)
		f.SetCellStyle(sheet, cell, cell, style)
	}
}

func applyDataStyle(f *excelize.File, sheet string, style int, colCount int, startRow, endRow int) {
	firstCol := "A"
	lastCol, _ := excelize.ColumnNumberToName(colCount)
	for row := startRow; row <= endRow; row++ {
		f.SetCellStyle(sheet, fmt.Sprintf("%s%d", firstCol, row), fmt.Sprintf("%s%d", lastCol, row), style)
	}
}

func addDropdown(f *excelize.File, sheet, sqref, formula string) {
	dv := excelize.NewDataValidation(true)
	dv.Sqref = sqref
	// Parse the formula string like `"A,B,C"` into a slice
	trimmed := formula
	if len(trimmed) > 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
		trimmed = trimmed[1 : len(trimmed)-1]
	}
	items := splitDropdownItems(trimmed)
	dv.SetDropList(items)
	dv.SetError(excelize.DataValidationErrorStyleWarning, "Invalid Value", "Please select from the dropdown or type a valid value.")
	dv.SetInput("Select Value", "Choose from the dropdown or type a custom value.")
	f.AddDataValidation(sheet, dv)
}

func splitDropdownItems(s string) []string {
	parts := []string{}
	for _, p := range splitOnComma(s) {
		trimmed := trimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitOnComma(s string) []string {
	result := []string{}
	current := ""
	for _, ch := range s {
		if ch == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	result = append(result, current)
	return result
}

func trimSpace(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}
