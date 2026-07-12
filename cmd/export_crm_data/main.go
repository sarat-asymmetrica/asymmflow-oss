package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/xuri/excelize/v2"
)

const (
	dbPath     = "ph_holdings.db"
	outputPath = "deploy_package/PH_Trading_CRM_Data_Export.xlsx"
)

// Style IDs (set during init)
var (
	headerStyle   int
	dateStyle     int
	currencyStyle int
	normalStyle   int
	instrHeader   int
	instrBody     int
	instrNote     int
)

func main() {
	log.Println("=== Acme Instrumentation CRM Data Export ===")
	log.Printf("Database: %s", dbPath)
	log.Printf("Output:   %s", outputPath)
	log.Println()

	// Open database
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database opened successfully.")

	// Create Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create styles
	initStyles(f)

	// --- Sheet 0: Data Cleanup Instructions ---
	writeInstructionsSheet(f)

	// --- Sheet 1: Customers ---
	custCount := exportCustomers(f, db)

	// --- Sheet 2: Customer Contacts ---
	ccCount := exportCustomerContacts(f, db)

	// --- Sheet 3: Suppliers ---
	suppCount := exportSuppliers(f, db)

	// --- Sheet 4: Supplier Contacts ---
	scCount := exportSupplierContacts(f, db)

	// --- Sheet 5: Orders Summary ---
	ordCount := exportOrders(f, db)

	// --- Sheet 6: Invoices Summary ---
	invCount := exportInvoices(f, db)

	// --- Sheet 7: Purchase Orders Summary ---
	poCount := exportPurchaseOrders(f, db)

	// Remove the default "Sheet1"
	f.DeleteSheet("Sheet1")

	// Set active sheet to Instructions
	idx, _ := f.GetSheetIndex("Data Cleanup Instructions")
	f.SetActiveSheet(idx)

	// Ensure output directory exists
	outDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Save
	if err := f.SaveAs(outputPath); err != nil {
		log.Fatalf("Failed to save Excel file: %v", err)
	}

	log.Println()
	log.Println("=== Export Complete ===")
	log.Printf("File: %s", outputPath)
	log.Println()
	log.Println("Row counts per sheet:")
	log.Printf("  Customers:           %d", custCount)
	log.Printf("  Customer Contacts:   %d", ccCount)
	log.Printf("  Suppliers:           %d", suppCount)
	log.Printf("  Supplier Contacts:   %d", scCount)
	log.Printf("  Orders Summary:      %d", ordCount)
	log.Printf("  Invoices Summary:    %d", invCount)
	log.Printf("  Purchase Orders:     %d", poCount)
	log.Printf("  TOTAL data rows:     %d", custCount+ccCount+suppCount+scCount+ordCount+invCount+poCount)

	// Verify file exists
	info, err := os.Stat(outputPath)
	if err != nil {
		log.Fatalf("Output file verification failed: %v", err)
	}
	log.Printf("  File size:           %.1f KB", float64(info.Size())/1024.0)
}

func initStyles(f *excelize.File) {
	// Header style: bold, light blue background, border bottom
	headerStyle, _ = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  11,
			Color: "#1F2937",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#DBEAFE"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#93C5FD", Style: 2},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
	})

	// Date style
	dateStyle, _ = f.NewStyle(&excelize.Style{
		NumFmt: 22, // m/d/yy h:mm
		Alignment: &excelize.Alignment{
			Horizontal: "left",
		},
	})

	// Currency style (3 decimal places for BHD)
	currencyStyle, _ = f.NewStyle(&excelize.Style{
		CustomNumFmt: func() *string { s := "#,##0.000"; return &s }(),
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
	})

	// Normal style
	normalStyle, _ = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Vertical: "top",
			WrapText: false,
		},
	})

	// Instructions header
	instrHeader, _ = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  16,
			Color: "#1E3A5F",
		},
	})

	// Instructions body
	instrBody, _ = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:  11,
			Color: "#374151",
		},
		Alignment: &excelize.Alignment{
			WrapText: true,
			Vertical: "top",
		},
	})

	// Instructions note (bold)
	instrNote, _ = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  11,
			Color: "#DC2626",
		},
		Alignment: &excelize.Alignment{
			WrapText: true,
		},
	})
}

func writeInstructionsSheet(f *excelize.File) {
	sheet := "Data Cleanup Instructions"
	f.NewSheet(sheet)

	// Set column widths
	f.SetColWidth(sheet, "A", "A", 80)

	now := time.Now().Format("2006-01-02 15:04")

	instructions := []struct {
		style int
		text  string
	}{
		{instrHeader, "Acme Instrumentation - CRM Data Cleanup Export"},
		{instrBody, fmt.Sprintf("Generated: %s", now)},
		{instrBody, ""},
		{instrNote, "IMPORTANT: Do NOT modify the 'id' columns. These are database UUIDs needed to map changes back to the system."},
		{instrBody, ""},
		{instrHeader, "Sheets in this workbook:"},
		{instrBody, "1. Customers - All 348 customer records (including soft-deleted). Review business names, addresses, contact info, payment grades."},
		{instrBody, "2. Customer Contacts - All 535 contact records linked to customers. Review names, phone numbers, emails, primary contact flags."},
		{instrBody, "3. Suppliers - All 34 supplier records. Review names, addresses, bank details, payment terms."},
		{instrBody, "4. Supplier Contacts - Supplier contact persons (may be empty if contacts are stored inline)."},
		{instrBody, "5. Orders Summary - All 175 orders with customer names and totals."},
		{instrBody, "6. Invoices Summary - All 468 invoices with status, amounts, and outstanding balances."},
		{instrBody, "7. Purchase Orders Summary - All 45 purchase orders with supplier names."},
		{instrBody, ""},
		{instrHeader, "How to clean up the data:"},
		{instrBody, "1. Review each sheet for incorrect, incomplete, or duplicate data."},
		{instrBody, "2. Use yellow highlighting on any cell you want to change."},
		{instrBody, "3. Add comments (right-click > Insert Comment) to explain changes."},
		{instrBody, "4. Check the 'deleted_at' column - records with a date were soft-deleted. Mark if they should be restored or permanently removed."},
		{instrBody, "5. Verify customer_type values: Corporate, Government, SME, Individual."},
		{instrBody, "6. Verify payment_grade values: A (excellent), B (good), C (average), D (poor - requires 100% advance)."},
		{instrBody, "7. Check for duplicate customers with slightly different names (e.g., 'NPC' vs 'National Petroleum Co. Company')."},
		{instrBody, "8. Ensure phone numbers follow a consistent format."},
		{instrBody, "9. Verify email addresses are valid."},
		{instrBody, "10. Check that credit_limit_bhd values are appropriate for each customer."},
		{instrBody, ""},
		{instrHeader, "Column format notes:"},
		{instrBody, "- Currency values (BHD) are formatted to 3 decimal places."},
		{instrBody, "- Date columns show date and time. Blank dates mean the field was never set."},
		{instrBody, "- is_credit_blocked: 0 = not blocked, 1 = blocked."},
		{instrBody, "- is_primary: 0 = secondary contact, 1 = primary contact."},
		{instrBody, ""},
		{instrNote, "Return the annotated file to Jordan for import back into AsymmFlow."},
	}

	for i, instr := range instructions {
		cell := fmt.Sprintf("A%d", i+1)
		f.SetCellValue(sheet, cell, instr.text)
		f.SetCellStyle(sheet, cell, cell, instr.style)
	}

	// Set row height for header rows
	f.SetRowHeight(sheet, 1, 30)
	f.SetRowHeight(sheet, 6, 25)
	f.SetRowHeight(sheet, 15, 25)
	f.SetRowHeight(sheet, 27, 25)
}

// writeHeaders writes a header row and applies header style + freeze pane
func writeHeaders(f *excelize.File, sheet string, headers []string) {
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	// Apply header style to entire header row
	startCell, _ := excelize.CoordinatesToCellName(1, 1)
	endCell, _ := excelize.CoordinatesToCellName(len(headers), 1)
	f.SetCellStyle(sheet, startCell, endCell, headerStyle)

	// Freeze header row
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Set header row height
	f.SetRowHeight(sheet, 1, 25)
}

// setColumnWidths sets widths for columns. widths maps column index (0-based) to width.
func setColumnWidths(f *excelize.File, sheet string, widths map[int]float64) {
	for col, w := range widths {
		colName, _ := excelize.ColumnNumberToName(col + 1)
		f.SetColWidth(sheet, colName, colName, w)
	}
}

// applyColumnStyles applies date or currency styles to entire data columns
func applyColumnStyles(f *excelize.File, sheet string, rowCount int, dateCols []int, currCols []int, totalCols int) {
	for row := 2; row <= rowCount+1; row++ {
		for _, col := range dateCols {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellStyle(sheet, cell, cell, dateStyle)
		}
		for _, col := range currCols {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellStyle(sheet, cell, cell, currencyStyle)
		}
	}
}

// nullStr handles sql.NullString
func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// nullFloat handles sql.NullFloat64
func nullFloat(nf sql.NullFloat64) any {
	if nf.Valid {
		return nf.Float64
	}
	return ""
}

// nullInt handles sql.NullInt64
func nullInt(ni sql.NullInt64) any {
	if ni.Valid {
		return ni.Int64
	}
	return ""
}

// nullTime handles sql.NullString for datetime columns - returns formatted string
func nullTime(ns sql.NullString) string {
	if !ns.Valid || ns.String == "" {
		return ""
	}
	// Try parsing various datetime formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05.000",
		"2006-01-02",
	}
	for _, fmt := range formats {
		if t, err := time.Parse(fmt, strings.TrimSpace(ns.String)); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	return ns.String
}

// ============================================================
// CUSTOMERS
// ============================================================
func exportCustomers(f *excelize.File, db *sql.DB) int {
	sheet := "Customers"
	f.NewSheet(sheet)

	headers := []string{
		"id", "customer_id", "customer_code", "short_code",
		"business_name", "customer_type", "customer_grade", "payment_grade",
		"address_line1", "address", "city", "country",
		"trn", "tax_code", "vat_number", "industry",
		"phone", "email",
		"credit_limit_bhd", "is_credit_blocked", "requires_prepayment",
		"relation_years", "payment_terms_days", "avg_payment_days",
		"total_orders_count", "total_orders_value", "avg_order_value", "last_order_date",
		"outstanding_bhd", "overdue_days", "ar_risk_tier", "dispute_count",
		"has_abb_competition", "is_emergency_only",
		"created_at", "updated_at", "deleted_at",
	}
	writeHeaders(f, sheet, headers)

	// Column widths
	setColumnWidths(f, sheet, map[int]float64{
		0: 38, 1: 38, 2: 16, 3: 12,
		4: 35, 5: 14, 6: 14, 7: 14,
		8: 30, 9: 30, 10: 15, 11: 15,
		12: 18, 13: 14, 14: 18, 15: 20,
		16: 18, 17: 30,
		18: 16, 19: 16, 20: 16,
		21: 14, 22: 16, 23: 16,
		24: 14, 25: 16, 26: 16, 27: 20,
		28: 16, 29: 14, 30: 14, 31: 14,
		32: 16, 33: 16,
		34: 20, 35: 20, 36: 20,
	})

	query := `SELECT
		id, customer_id, customer_code, short_code,
		business_name, customer_type, customer_grade, payment_grade,
		address_line1, address, city, country,
		trn, tax_code, vat_number, industry,
		phone, email,
		credit_limit_bhd, is_credit_blocked, requires_prepayment,
		relation_years, payment_terms_days, avg_payment_days,
		total_orders_count, total_orders_value, avg_order_value, last_order_date,
		outstanding_bhd, overdue_days, ar_risk_tier, dispute_count,
		has_abb_competition, is_emergency_only,
		created_at, updated_at, deleted_at
	FROM customers
	ORDER BY business_name`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query customers: %v", err)
	}
	defer rows.Close()

	rowNum := 0
	for rows.Next() {
		var (
			id, custID, custCode, shortCode                sql.NullString
			bizName, custType, custGrade, payGrade         sql.NullString
			addrLine1, addr, city, country                 sql.NullString
			trn, taxCode, vatNum, industry                 sql.NullString
			phone, email                                   sql.NullString
			arRiskTier                                     sql.NullString
			lastOrderDate, createdAt, updatedAt, deletedAt sql.NullString
			creditLimit, avgPayDays, totalOrdersValue      sql.NullFloat64
			avgOrderValue, outstanding                     sql.NullFloat64
			isCreditBlocked, reqPrepay, relationYears      sql.NullInt64
			payTermsDays, totalOrdersCount, ovDays         sql.NullInt64
			disputeCount, hasABB, isEmergency              sql.NullInt64
		)

		err := rows.Scan(
			&id, &custID, &custCode, &shortCode,
			&bizName, &custType, &custGrade, &payGrade,
			&addrLine1, &addr, &city, &country,
			&trn, &taxCode, &vatNum, &industry,
			&phone, &email,
			&creditLimit, &isCreditBlocked, &reqPrepay,
			&relationYears, &payTermsDays, &avgPayDays,
			&totalOrdersCount, &totalOrdersValue, &avgOrderValue, &lastOrderDate,
			&outstanding, &ovDays, &arRiskTier, &disputeCount,
			&hasABB, &isEmergency,
			&createdAt, &updatedAt, &deletedAt,
		)
		if err != nil {
			log.Fatalf("Failed to scan customer row: %v", err)
		}

		rowNum++
		r := rowNum + 1 // 1-indexed, header is row 1

		vals := []any{
			nullStr(id), nullStr(custID), nullStr(custCode), nullStr(shortCode),
			nullStr(bizName), nullStr(custType), nullStr(custGrade), nullStr(payGrade),
			nullStr(addrLine1), nullStr(addr), nullStr(city), nullStr(country),
			nullStr(trn), nullStr(taxCode), nullStr(vatNum), nullStr(industry),
			nullStr(phone), nullStr(email),
			nullFloat(creditLimit), nullInt(isCreditBlocked), nullInt(reqPrepay),
			nullInt(relationYears), nullInt(payTermsDays), nullFloat(avgPayDays),
			nullInt(totalOrdersCount), nullFloat(totalOrdersValue), nullFloat(avgOrderValue), nullTime(lastOrderDate),
			nullFloat(outstanding), nullInt(ovDays), nullStr(arRiskTier), nullInt(disputeCount),
			nullInt(hasABB), nullInt(isEmergency),
			nullTime(createdAt), nullTime(updatedAt), nullTime(deletedAt),
		}

		for c, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, r)
			f.SetCellValue(sheet, cell, val)
		}
	}

	// Apply column styles
	// date columns: last_order_date(27), created_at(34), updated_at(35), deleted_at(36)
	dateCols := []int{27, 34, 35, 36}
	// currency columns: credit_limit_bhd(18), total_orders_value(25), avg_order_value(26), outstanding_bhd(28)
	currCols := []int{18, 25, 26, 28}
	applyColumnStyles(f, sheet, rowNum, dateCols, currCols, len(headers))

	log.Printf("Customers: %d rows exported", rowNum)
	return rowNum
}

// ============================================================
// CUSTOMER CONTACTS
// ============================================================
func exportCustomerContacts(f *excelize.File, db *sql.DB) int {
	sheet := "Customer Contacts"
	f.NewSheet(sheet)

	headers := []string{
		"id", "customer_id", "contact_name", "contact_role", "job_title",
		"salutation", "contact_email", "email", "contact_phone", "phone",
		"is_primary", "is_primary_contact", "address",
		"created_at", "updated_at", "deleted_at",
	}
	writeHeaders(f, sheet, headers)

	setColumnWidths(f, sheet, map[int]float64{
		0: 38, 1: 38, 2: 25, 3: 20, 4: 20,
		5: 12, 6: 30, 7: 30, 8: 18, 9: 18,
		10: 12, 11: 16, 12: 30,
		13: 20, 14: 20, 15: 20,
	})

	query := `SELECT
		cc.id, cc.customer_id, cc.contact_name, cc.contact_role, cc.job_title,
		cc.salutation, cc.contact_email, cc.email, cc.contact_phone, cc.phone,
		cc.is_primary, cc.is_primary_contact, cc.address,
		cc.created_at, cc.updated_at, cc.deleted_at
	FROM customer_contacts cc
	ORDER BY cc.customer_id, cc.is_primary DESC, cc.contact_name`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query customer_contacts: %v", err)
	}
	defer rows.Close()

	rowNum := 0
	for rows.Next() {
		var (
			id, custID, name, role, jobTitle         sql.NullString
			salutation, cEmail, email, cPhone, phone sql.NullString
			addr                                     sql.NullString
			createdAt, updatedAt, deletedAt          sql.NullString
			isPrimary, isPrimaryContact              sql.NullInt64
		)

		err := rows.Scan(
			&id, &custID, &name, &role, &jobTitle,
			&salutation, &cEmail, &email, &cPhone, &phone,
			&isPrimary, &isPrimaryContact, &addr,
			&createdAt, &updatedAt, &deletedAt,
		)
		if err != nil {
			log.Fatalf("Failed to scan customer_contacts row: %v", err)
		}

		rowNum++
		r := rowNum + 1

		vals := []any{
			nullStr(id), nullStr(custID), nullStr(name), nullStr(role), nullStr(jobTitle),
			nullStr(salutation), nullStr(cEmail), nullStr(email), nullStr(cPhone), nullStr(phone),
			nullInt(isPrimary), nullInt(isPrimaryContact), nullStr(addr),
			nullTime(createdAt), nullTime(updatedAt), nullTime(deletedAt),
		}

		for c, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, r)
			f.SetCellValue(sheet, cell, val)
		}
	}

	dateCols := []int{13, 14, 15}
	applyColumnStyles(f, sheet, rowNum, dateCols, nil, len(headers))

	log.Printf("Customer Contacts: %d rows exported", rowNum)
	return rowNum
}

// ============================================================
// SUPPLIERS
// ============================================================
func exportSuppliers(f *excelize.File, db *sql.DB) int {
	sheet := "Suppliers"
	f.NewSheet(sheet)

	headers := []string{
		"id", "supplier_code", "supplier_name", "supplier_type", "country",
		"primary_contact", "phone", "email", "address",
		"payment_terms", "lead_time_days", "rating",
		"bank_name", "bank_account", "swift_code", "account_number", "iban",
		"tax_id", "brands_handled", "product_types",
		"on_time_delivery_pct", "is_active", "notes",
		"created_at", "updated_at", "deleted_at",
	}
	writeHeaders(f, sheet, headers)

	setColumnWidths(f, sheet, map[int]float64{
		0: 38, 1: 16, 2: 35, 3: 16, 4: 15,
		5: 22, 6: 18, 7: 30, 8: 35,
		9: 15, 10: 14, 11: 10,
		12: 20, 13: 22, 14: 16, 15: 18, 16: 28,
		17: 16, 18: 30, 19: 30,
		20: 18, 21: 10, 22: 30,
		23: 20, 24: 20, 25: 20,
	})

	query := `SELECT
		id, supplier_code, supplier_name, supplier_type, country,
		primary_contact, phone, email, address,
		payment_terms, lead_time_days, rating,
		bank_name, bank_account, swift_code, account_number, iban,
		tax_id, brands_handled, product_types,
		on_time_delivery_pct, is_active, notes,
		created_at, updated_at, deleted_at
	FROM suppliers
	ORDER BY supplier_name`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query suppliers: %v", err)
	}
	defer rows.Close()

	rowNum := 0
	for rows.Next() {
		var (
			id, code, name, sType, country           sql.NullString
			contact, phone, email, addr              sql.NullString
			payTerms                                 sql.NullString
			bankName, bankAcct, swift, acctNum, iban sql.NullString
			taxID, brands, prodTypes                 sql.NullString
			notes                                    sql.NullString
			createdAt, updatedAt, deletedAt          sql.NullString
			leadTimeDays, rating, isActive           sql.NullInt64
			onTimePct                                sql.NullFloat64
		)

		err := rows.Scan(
			&id, &code, &name, &sType, &country,
			&contact, &phone, &email, &addr,
			&payTerms, &leadTimeDays, &rating,
			&bankName, &bankAcct, &swift, &acctNum, &iban,
			&taxID, &brands, &prodTypes,
			&onTimePct, &isActive, &notes,
			&createdAt, &updatedAt, &deletedAt,
		)
		if err != nil {
			log.Fatalf("Failed to scan supplier row: %v", err)
		}

		rowNum++
		r := rowNum + 1

		vals := []any{
			nullStr(id), nullStr(code), nullStr(name), nullStr(sType), nullStr(country),
			nullStr(contact), nullStr(phone), nullStr(email), nullStr(addr),
			nullStr(payTerms), nullInt(leadTimeDays), nullInt(rating),
			nullStr(bankName), nullStr(bankAcct), nullStr(swift), nullStr(acctNum), nullStr(iban),
			nullStr(taxID), nullStr(brands), nullStr(prodTypes),
			nullFloat(onTimePct), nullInt(isActive), nullStr(notes),
			nullTime(createdAt), nullTime(updatedAt), nullTime(deletedAt),
		}

		for c, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, r)
			f.SetCellValue(sheet, cell, val)
		}
	}

	dateCols := []int{23, 24, 25}
	applyColumnStyles(f, sheet, rowNum, dateCols, nil, len(headers))

	log.Printf("Suppliers: %d rows exported", rowNum)
	return rowNum
}

// ============================================================
// SUPPLIER CONTACTS
// ============================================================
func exportSupplierContacts(f *excelize.File, db *sql.DB) int {
	sheet := "Supplier Contacts"
	f.NewSheet(sheet)

	headers := []string{
		"id", "supplier_id", "contact_name", "job_title",
		"email", "phone", "address", "is_primary_contact",
		"created_at", "updated_at", "deleted_at",
	}
	writeHeaders(f, sheet, headers)

	setColumnWidths(f, sheet, map[int]float64{
		0: 38, 1: 38, 2: 25, 3: 20,
		4: 30, 5: 18, 6: 30, 7: 16,
		8: 20, 9: 20, 10: 20,
	})

	query := `SELECT
		id, supplier_id, contact_name, job_title,
		email, phone, address, is_primary_contact,
		created_at, updated_at, deleted_at
	FROM supplier_contacts
	ORDER BY supplier_id, contact_name`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query supplier_contacts: %v", err)
	}
	defer rows.Close()

	rowNum := 0
	for rows.Next() {
		var (
			id, suppID, name, jobTitle      sql.NullString
			email, phone, addr              sql.NullString
			createdAt, updatedAt, deletedAt sql.NullString
			isPrimary                       sql.NullInt64
		)

		err := rows.Scan(
			&id, &suppID, &name, &jobTitle,
			&email, &phone, &addr, &isPrimary,
			&createdAt, &updatedAt, &deletedAt,
		)
		if err != nil {
			log.Fatalf("Failed to scan supplier_contacts row: %v", err)
		}

		rowNum++
		r := rowNum + 1

		vals := []any{
			nullStr(id), nullStr(suppID), nullStr(name), nullStr(jobTitle),
			nullStr(email), nullStr(phone), nullStr(addr), nullInt(isPrimary),
			nullTime(createdAt), nullTime(updatedAt), nullTime(deletedAt),
		}

		for c, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, r)
			f.SetCellValue(sheet, cell, val)
		}
	}

	dateCols := []int{8, 9, 10}
	applyColumnStyles(f, sheet, rowNum, dateCols, nil, len(headers))

	log.Printf("Supplier Contacts: %d rows exported", rowNum)
	return rowNum
}

// ============================================================
// ORDERS SUMMARY
// ============================================================
func exportOrders(f *excelize.File, db *sql.DB) int {
	sheet := "Orders Summary"
	f.NewSheet(sheet)

	headers := []string{
		"id", "order_number", "customer_id", "customer_name",
		"customer_po_number", "order_date", "required_date",
		"status", "total_value_bhd", "grand_total_bhd",
		"payment_terms", "delivery_terms",
		"offer_id", "offer_number",
		"created_at", "updated_at", "deleted_at",
	}
	writeHeaders(f, sheet, headers)

	setColumnWidths(f, sheet, map[int]float64{
		0: 38, 1: 18, 2: 38, 3: 35,
		4: 20, 5: 20, 6: 20,
		7: 14, 8: 16, 9: 16,
		10: 20, 11: 20,
		12: 38, 13: 18,
		14: 20, 15: 20, 16: 20,
	})

	query := `SELECT
		o.id, o.order_number, o.customer_id, o.customer_name,
		o.customer_po_number, o.order_date, o.required_date,
		o.status, o.total_value_bhd, o.grand_total_bhd,
		o.payment_terms, o.delivery_terms,
		o.offer_id, o.offer_number,
		o.created_at, o.updated_at, o.deleted_at
	FROM orders o
	ORDER BY o.order_date DESC, o.order_number`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query orders: %v", err)
	}
	defer rows.Close()

	rowNum := 0
	for rows.Next() {
		var (
			id, orderNum, custID, custName  sql.NullString
			custPO, orderDate, reqDate      sql.NullString
			status, payTerms, delTerms      sql.NullString
			offerID, offerNum               sql.NullString
			createdAt, updatedAt, deletedAt sql.NullString
			totalVal, grandTotal            sql.NullFloat64
		)

		err := rows.Scan(
			&id, &orderNum, &custID, &custName,
			&custPO, &orderDate, &reqDate,
			&status, &totalVal, &grandTotal,
			&payTerms, &delTerms,
			&offerID, &offerNum,
			&createdAt, &updatedAt, &deletedAt,
		)
		if err != nil {
			log.Fatalf("Failed to scan order row: %v", err)
		}

		rowNum++
		r := rowNum + 1

		vals := []any{
			nullStr(id), nullStr(orderNum), nullStr(custID), nullStr(custName),
			nullStr(custPO), nullTime(orderDate), nullTime(reqDate),
			nullStr(status), nullFloat(totalVal), nullFloat(grandTotal),
			nullStr(payTerms), nullStr(delTerms),
			nullStr(offerID), nullStr(offerNum),
			nullTime(createdAt), nullTime(updatedAt), nullTime(deletedAt),
		}

		for c, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, r)
			f.SetCellValue(sheet, cell, val)
		}
	}

	dateCols := []int{5, 6, 14, 15, 16}
	currCols := []int{8, 9}
	applyColumnStyles(f, sheet, rowNum, dateCols, currCols, len(headers))

	log.Printf("Orders Summary: %d rows exported", rowNum)
	return rowNum
}

// ============================================================
// INVOICES SUMMARY
// ============================================================
func exportInvoices(f *excelize.File, db *sql.DB) int {
	sheet := "Invoices Summary"
	f.NewSheet(sheet)

	headers := []string{
		"id", "invoice_number", "order_id", "customer_id", "customer_name",
		"customer_po_number", "invoice_date", "due_date",
		"status", "subtotal_bhd", "vatbhd", "vat_percent", "grand_total_bhd", "outstanding_bhd",
		"payment_terms", "delivery_terms",
		"offer_id", "offer_number", "delivery_note_id", "delivery_note_number",
		"notes",
		"created_at", "updated_at", "deleted_at",
	}
	writeHeaders(f, sheet, headers)

	setColumnWidths(f, sheet, map[int]float64{
		0: 38, 1: 18, 2: 38, 3: 38, 4: 35,
		5: 20, 6: 20, 7: 20,
		8: 14, 9: 14, 10: 12, 11: 12, 12: 14, 13: 16,
		14: 20, 15: 20,
		16: 38, 17: 18, 18: 38, 19: 20,
		20: 30,
		21: 20, 22: 20, 23: 20,
	})

	query := `SELECT
		i.id, i.invoice_number, i.order_id, i.customer_id, i.customer_name,
		i.customer_po_number, i.invoice_date, i.due_date,
		i.status, i.subtotal_bhd, i.vatbhd, i.vat_percent, i.grand_total_bhd, i.outstanding_bhd,
		i.payment_terms, i.delivery_terms,
		i.offer_id, i.offer_number, i.delivery_note_id, i.delivery_note_number,
		i.notes,
		i.created_at, i.updated_at, i.deleted_at
	FROM invoices i
	ORDER BY i.invoice_date DESC, i.invoice_number`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query invoices: %v", err)
	}
	defer rows.Close()

	rowNum := 0
	for rows.Next() {
		var (
			id, invNum, orderID, custID, custName          sql.NullString
			custPO, invDate, dueDate                       sql.NullString
			status, payTerms, delTerms                     sql.NullString
			offerID, offerNum, dnID, dnNum                 sql.NullString
			notes                                          sql.NullString
			createdAt, updatedAt, deletedAt                sql.NullString
			subtotal, vat, vatPct, grandTotal, outstanding sql.NullFloat64
		)

		err := rows.Scan(
			&id, &invNum, &orderID, &custID, &custName,
			&custPO, &invDate, &dueDate,
			&status, &subtotal, &vat, &vatPct, &grandTotal, &outstanding,
			&payTerms, &delTerms,
			&offerID, &offerNum, &dnID, &dnNum,
			&notes,
			&createdAt, &updatedAt, &deletedAt,
		)
		if err != nil {
			log.Fatalf("Failed to scan invoice row: %v", err)
		}

		rowNum++
		r := rowNum + 1

		vals := []any{
			nullStr(id), nullStr(invNum), nullStr(orderID), nullStr(custID), nullStr(custName),
			nullStr(custPO), nullTime(invDate), nullTime(dueDate),
			nullStr(status), nullFloat(subtotal), nullFloat(vat), nullFloat(vatPct), nullFloat(grandTotal), nullFloat(outstanding),
			nullStr(payTerms), nullStr(delTerms),
			nullStr(offerID), nullStr(offerNum), nullStr(dnID), nullStr(dnNum),
			nullStr(notes),
			nullTime(createdAt), nullTime(updatedAt), nullTime(deletedAt),
		}

		for c, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, r)
			f.SetCellValue(sheet, cell, val)
		}
	}

	dateCols := []int{6, 7, 21, 22, 23}
	currCols := []int{9, 10, 12, 13}
	applyColumnStyles(f, sheet, rowNum, dateCols, currCols, len(headers))

	log.Printf("Invoices Summary: %d rows exported", rowNum)
	return rowNum
}

// ============================================================
// PURCHASE ORDERS SUMMARY
// ============================================================
func exportPurchaseOrders(f *excelize.File, db *sql.DB) int {
	sheet := "Purchase Orders Summary"
	f.NewSheet(sheet)

	headers := []string{
		"id", "po_number", "supplier_id", "supplier_name",
		"order_id", "po_date", "expected_delivery",
		"status", "currency", "exchange_rate",
		"subtotal_foreign", "subtotal_bhd", "vat_amount", "total_foreign", "total_bhd",
		"payment_terms", "payment_due_date",
		"approved_by", "approved_at",
		"created_at", "updated_at", "deleted_at",
	}
	writeHeaders(f, sheet, headers)

	setColumnWidths(f, sheet, map[int]float64{
		0: 38, 1: 18, 2: 38, 3: 30,
		4: 38, 5: 20, 6: 20,
		7: 14, 8: 10, 9: 14,
		10: 16, 11: 14, 12: 12, 13: 16, 14: 14,
		15: 20, 16: 20,
		17: 18, 18: 20,
		19: 20, 20: 20, 21: 20,
	})

	query := `SELECT
		po.id, po.po_number, po.supplier_id, po.supplier_name,
		po.order_id, po.po_date, po.expected_delivery,
		po.status, po.currency, po.exchange_rate,
		po.subtotal_foreign, po.subtotal_bhd, po.vat_amount, po.total_foreign, po.total_bhd,
		po.payment_terms, po.payment_due_date,
		po.approved_by, po.approved_at,
		po.created_at, po.updated_at, po.deleted_at
	FROM purchase_orders po
	ORDER BY po.po_date DESC, po.po_number`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query purchase_orders: %v", err)
	}
	defer rows.Close()

	rowNum := 0
	for rows.Next() {
		var (
			id, poNum, suppID, suppName     sql.NullString
			orderID, poDate, expDel         sql.NullString
			status, currency                sql.NullString
			payTerms, payDue                sql.NullString
			approvedBy, approvedAt          sql.NullString
			createdAt, updatedAt, deletedAt sql.NullString
			exchRate, subForeign, subBHD    sql.NullFloat64
			vatAmt, totalForeign, totalBHD  sql.NullFloat64
		)

		err := rows.Scan(
			&id, &poNum, &suppID, &suppName,
			&orderID, &poDate, &expDel,
			&status, &currency, &exchRate,
			&subForeign, &subBHD, &vatAmt, &totalForeign, &totalBHD,
			&payTerms, &payDue,
			&approvedBy, &approvedAt,
			&createdAt, &updatedAt, &deletedAt,
		)
		if err != nil {
			log.Fatalf("Failed to scan purchase_order row: %v", err)
		}

		rowNum++
		r := rowNum + 1

		vals := []any{
			nullStr(id), nullStr(poNum), nullStr(suppID), nullStr(suppName),
			nullStr(orderID), nullTime(poDate), nullTime(expDel),
			nullStr(status), nullStr(currency), nullFloat(exchRate),
			nullFloat(subForeign), nullFloat(subBHD), nullFloat(vatAmt), nullFloat(totalForeign), nullFloat(totalBHD),
			nullStr(payTerms), nullTime(payDue),
			nullStr(approvedBy), nullTime(approvedAt),
			nullTime(createdAt), nullTime(updatedAt), nullTime(deletedAt),
		}

		for c, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, r)
			f.SetCellValue(sheet, cell, val)
		}
	}

	dateCols := []int{5, 6, 16, 18, 19, 20, 21}
	currCols := []int{10, 11, 12, 13, 14}
	applyColumnStyles(f, sheet, rowNum, dateCols, currCols, len(headers))

	log.Printf("Purchase Orders Summary: %d rows exported", rowNum)
	return rowNum
}
