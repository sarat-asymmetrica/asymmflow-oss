package phimport

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/finance"
)

// costingSheetDataFixture mirrors the enriched columns of the real
// main.CostingSheetData model (which lives in package main and cannot be
// imported here). It carries the PC-D22 provenance columns so the importer has
// destination columns to pair the source ones against, exactly as the app's
// provisioned schema would.
type costingSheetDataFixture struct {
	ID             uint `gorm:"primaryKey"`
	RFQID          uint
	OfferNumber    string
	CustomerName   string
	ProductType    string
	TotalValueBHD  float64 `gorm:"column:total_value_bhd"`
	LineItemCount  int
	SourceFilePath string
	ExtractedAt    *time.Time
}

func (costingSheetDataFixture) TableName() string { return "costing_sheet_data" }

// buildEnrichSource fabricates a synthetic PH-format source carrying, for each
// of the 7 enriched tables, BOTH the OSS-spelled column and the PH legacy/extra
// column with DIFFERENT values — so a passing assertion proves the enriched
// columns are carried as distinct data, not folded onto their OSS siblings.
func buildEnrichSource(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmts := []string{
		// customers: tax_code + address/phone/email carried alongside the OSS
		// address_line1/primary_phone/primary_email (distinct values).
		`CREATE TABLE customers (id TEXT PRIMARY KEY, customer_id TEXT, business_name TEXT,
			tax_code TEXT, trn TEXT, address TEXT, address_line1 TEXT,
			phone TEXT, primary_phone TEXT, email TEXT, primary_email TEXT, deleted_at DATETIME)`,
		`INSERT INTO customers (id, customer_id, business_name, tax_code, trn, address, address_line1, phone, primary_phone, email, primary_email) VALUES
			('c1', 'CUST-NIMB1', 'Nimbus Controls', 'TC-100', 'TRN-900', 'Old Souq Road', 'New Tower 5',
				'+973-1111', '+973-9999', 'legacy@nimbus.test', 'primary@nimbus.test'),
			('c2', 'CUST-ATLA1', 'Atlas Traders', '', '', '', '', '', '', '', '')`,

		// suppliers: is_active flag.
		`CREATE TABLE suppliers (id TEXT PRIMARY KEY, supplier_code TEXT, supplier_name TEXT, is_active INTEGER, deleted_at DATETIME)`,
		`INSERT INTO suppliers (id, supplier_code, supplier_name, is_active) VALUES
			('s1', 'SUP-RHINE', 'Rhine Instruments', 1),
			('s2', 'SUP-DORMANT', 'Dormant Supply Co', 0)`,

		// customer_contacts: is_primary + salutation carried alongside the OSS
		// is_primary_contact (distinct value).
		`CREATE TABLE customer_contacts (id TEXT PRIMARY KEY, customer_id TEXT, contact_name TEXT,
			is_primary NUMERIC, is_primary_contact NUMERIC, salutation TEXT, deleted_at DATETIME)`,
		`INSERT INTO customer_contacts (id, customer_id, contact_name, is_primary, is_primary_contact, salutation) VALUES
			('cc1', 'c1', 'Dana Fields', 1, 0, 'Dr.'),
			('cc2', 'c1', 'Sam Reed', 0, 0, '')`,

		// costing_sheet_data: the 7 extraction/provenance columns.
		`CREATE TABLE costing_sheet_data (id INTEGER PRIMARY KEY, rfq_id INTEGER,
			offer_number TEXT, customer_name TEXT, product_type TEXT, total_value_bhd REAL,
			line_item_count INTEGER, source_file_path TEXT, extracted_at DATETIME)`,
		`INSERT INTO costing_sheet_data (id, rfq_id, offer_number, customer_name, product_type, total_value_bhd, line_item_count, source_file_path, extracted_at) VALUES
			(1, 10, 'OFF-26-0001', 'Nimbus Controls', 'Flow Meter', 1234.567, 3, 'C:\inbox\offer-a.pdf', '2026-05-01 10:00:00'),
			(2, 11, '', '', '', 0, 0, '', NULL)`,

		// order_items: unit_price_bhd carried alongside the OSS unit_price
		// (distinct value); brand + token identify the instrument.
		`CREATE TABLE order_items (id TEXT PRIMARY KEY, order_id TEXT, product_id TEXT, line_number INTEGER,
			unit_price REAL, unit_price_bhd DECIMAL(15,3), brand TEXT, token TEXT, deleted_at DATETIME)`,
		`INSERT INTO order_items (id, order_id, product_id, line_number, unit_price, unit_price_bhd, brand, token) VALUES
			('oi1', 'o1', 'p1', 1, 10.000, 3.750, 'Acme', 'TKN-XYZ'),
			('oi2', 'o1', 'p2', 2, 0.0, 0.0, '', '')`,

		// invoices: notes.
		`CREATE TABLE invoices (id TEXT PRIMARY KEY, invoice_number TEXT, customer_id TEXT, grand_total_bhd REAL, notes TEXT, invoice_hash TEXT, deleted_at DATETIME)`,
		`INSERT INTO invoices (id, invoice_number, customer_id, grand_total_bhd, notes, invoice_hash) VALUES
			('i1', 'INV-26-0001', 'c1', 525.500, 'Deliver to gate 4', ''),
			('i2', 'INV-26-0002', 'c2', 105.000, '', '')`,

		// supplier_payments: payment_number.
		`CREATE TABLE supplier_payments (id TEXT PRIMARY KEY, supplier_id TEXT, amount_bhd REAL, payment_method TEXT, payment_number TEXT, deleted_at DATETIME)`,
		`INSERT INTO supplier_payments (id, supplier_id, amount_bhd, payment_method, payment_number) VALUES
			('sp1', 's1', 500.000, 'Cash', 'SP-26-0007'),
			('sp2', 's1', 250.000, 'Cheque', '')`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("source DDL: %v\n%s", err, stmt)
		}
	}
}

// buildEnrichDest provisions the destination from the REAL enriched GORM models
// (crm + finance) plus a fixture mirror of costing_sheet_data — so the test
// exercises the actual struct fields added in PC-D22. FK constraints are
// disabled at migration (as startup() does) so the fixture needs no parent
// tables and the importer's foreign_key_check gate has nothing to trip on.
func buildEnrichDest(t *testing.T, path string) {
	t.Helper()
	db, err := gorm.Open(gormlite.Open("file:"+filepath.ToSlash(path)), &gorm.Config{
		Logger:                                   logger.Discard,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&crm.CustomerMaster{}, &crm.SupplierMaster{}, &crm.CustomerContact{}, &crm.OrderItem{},
		&finance.Invoice{}, &finance.SupplierPayment{}, &costingSheetDataFixture{},
	); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestRun_EnrichesNineteenColumns proves the PC-D22 (Mission I D-I-5)
// enrichment: all 19 formerly-dropped source columns now round-trip into
// matching destination columns, distinct from their OSS-spelled siblings, and
// none of the 19 is reported as a column drop any more.
func TestRun_EnrichesNineteenColumns(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "ph_source.db")
	destPath := filepath.Join(dir, "oss_dest.db")
	buildEnrichSource(t, sourcePath)
	buildEnrichDest(t, destPath)

	report, err := Run(context.Background(), Options{SourcePath: sourcePath, DestPath: destPath})
	if err != nil {
		t.Fatal(err)
	}

	// None of the 19 enriched columns may appear in the drop ledger any more.
	enriched := map[string]bool{
		"customers.tax_code": true, "customers.address": true, "customers.phone": true, "customers.email": true,
		"suppliers.is_active":          true,
		"customer_contacts.is_primary": true, "customer_contacts.salutation": true,
		"costing_sheet_data.offer_number": true, "costing_sheet_data.customer_name": true,
		"costing_sheet_data.product_type": true, "costing_sheet_data.total_value_bhd": true,
		"costing_sheet_data.line_item_count": true, "costing_sheet_data.source_file_path": true,
		"costing_sheet_data.extracted_at": true,
		"order_items.unit_price_bhd":      true, "order_items.brand": true, "order_items.token": true,
		"invoices.notes":                   true,
		"supplier_payments.payment_number": true,
	}
	for _, d := range report.ColumnDrops {
		if enriched[d.Table+"."+d.Column] {
			t.Fatalf("enriched column still reported as dropped: %s.%s (%d rows)", d.Table, d.Column, d.NonEmptyRows)
		}
	}

	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(destPath)+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// customers: 4 enriched columns carried DISTINCT from OSS siblings.
	var taxCode, address, addressLine1, phone, primaryPhone, email, primaryEmail string
	if err := db.QueryRow(`SELECT tax_code, address, address_line1, phone, primary_phone, email, primary_email
		FROM customers WHERE id = 'c1'`).
		Scan(&taxCode, &address, &addressLine1, &phone, &primaryPhone, &email, &primaryEmail); err != nil {
		t.Fatal(err)
	}
	if taxCode != "TC-100" {
		t.Fatalf("customers.tax_code: %q", taxCode)
	}
	if address != "Old Souq Road" || addressLine1 != "New Tower 5" {
		t.Fatalf("customers.address must carry distinct from address_line1: %q / %q", address, addressLine1)
	}
	if phone != "+973-1111" || primaryPhone != "+973-9999" {
		t.Fatalf("customers.phone must carry distinct from primary_phone: %q / %q", phone, primaryPhone)
	}
	if email != "legacy@nimbus.test" || primaryEmail != "primary@nimbus.test" {
		t.Fatalf("customers.email must carry distinct from primary_email: %q / %q", email, primaryEmail)
	}

	// suppliers.is_active
	var isActive bool
	if err := db.QueryRow(`SELECT is_active FROM suppliers WHERE id = 's1'`).Scan(&isActive); err != nil || !isActive {
		t.Fatalf("suppliers.is_active: %v %v", isActive, err)
	}

	// customer_contacts: is_primary distinct from is_primary_contact + salutation
	var isPrimary, isPrimaryContact bool
	var salutation string
	if err := db.QueryRow(`SELECT is_primary, is_primary_contact, salutation FROM customer_contacts WHERE id = 'cc1'`).
		Scan(&isPrimary, &isPrimaryContact, &salutation); err != nil {
		t.Fatal(err)
	}
	if !isPrimary || isPrimaryContact {
		t.Fatalf("customer_contacts.is_primary must carry distinct from is_primary_contact: %v / %v", isPrimary, isPrimaryContact)
	}
	if salutation != "Dr." {
		t.Fatalf("customer_contacts.salutation: %q", salutation)
	}

	// costing_sheet_data: all 7 provenance columns
	var offerNumber, custName, productType, sourceFilePath string
	var totalValueBHD float64
	var lineItemCount int
	var extractedAt sql.NullString
	if err := db.QueryRow(`SELECT offer_number, customer_name, product_type, total_value_bhd, line_item_count, source_file_path, extracted_at
		FROM costing_sheet_data WHERE id = 1`).
		Scan(&offerNumber, &custName, &productType, &totalValueBHD, &lineItemCount, &sourceFilePath, &extractedAt); err != nil {
		t.Fatal(err)
	}
	if offerNumber != "OFF-26-0001" || custName != "Nimbus Controls" || productType != "Flow Meter" {
		t.Fatalf("costing_sheet_data text columns: %q %q %q", offerNumber, custName, productType)
	}
	if totalValueBHD != 1234.567 {
		t.Fatalf("costing_sheet_data.total_value_bhd must copy exactly: %v", totalValueBHD)
	}
	if lineItemCount != 3 {
		t.Fatalf("costing_sheet_data.line_item_count: %d", lineItemCount)
	}
	if sourceFilePath != `C:\inbox\offer-a.pdf` {
		t.Fatalf("costing_sheet_data.source_file_path: %q", sourceFilePath)
	}
	if !extractedAt.Valid || extractedAt.String == "" {
		t.Fatalf("costing_sheet_data.extracted_at must carry a value: %+v", extractedAt)
	}

	// order_items: unit_price_bhd distinct from unit_price + brand + token
	var unitPrice, unitPriceBHD float64
	var brand, token string
	if err := db.QueryRow(`SELECT unit_price, unit_price_bhd, brand, token FROM order_items WHERE id = 'oi1'`).
		Scan(&unitPrice, &unitPriceBHD, &brand, &token); err != nil {
		t.Fatal(err)
	}
	if unitPrice != 10.0 || unitPriceBHD != 3.75 {
		t.Fatalf("order_items.unit_price_bhd must carry distinct from unit_price: %v / %v", unitPrice, unitPriceBHD)
	}
	if brand != "Acme" || token != "TKN-XYZ" {
		t.Fatalf("order_items brand/token: %q / %q", brand, token)
	}

	// invoices.notes
	var notes string
	if err := db.QueryRow(`SELECT notes FROM invoices WHERE id = 'i1'`).Scan(&notes); err != nil || notes != "Deliver to gate 4" {
		t.Fatalf("invoices.notes: %q %v", notes, err)
	}

	// supplier_payments.payment_number
	var paymentNumber string
	if err := db.QueryRow(`SELECT payment_number FROM supplier_payments WHERE id = 'sp1'`).Scan(&paymentNumber); err != nil || paymentNumber != "SP-26-0007" {
		t.Fatalf("supplier_payments.payment_number: %q %v", paymentNumber, err)
	}
}
