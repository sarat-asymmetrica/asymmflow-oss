package phimport

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/finance"
	infradomain "ph_holdings_app/pkg/infra"
)

// buildSourceDB fabricates a small SYNTHETIC PH-format database: drifted
// extra columns, populated HMAC hashes, encrypted settings, a renamed table,
// a pending-decision table, and an unmapped legacy table. No real data.
func buildSourceDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmts := []string{
		// customers: PH-side drift column ph_only_flag must not break the copy.
		`CREATE TABLE customers (id TEXT PRIMARY KEY, customer_id TEXT, customer_code TEXT, business_name TEXT, ph_only_flag INTEGER, deleted_at DATETIME)`,
		`INSERT INTO customers (id, customer_id, customer_code, business_name, ph_only_flag) VALUES
			('c1', 'CUST-NIMB1', 'CUST-NIMB1', 'Nimbus Controls', 1),
			('c2', 'CUST-ATLA1', 'CUST-ATLA1', 'Atlas Traders', 0)`,
		`CREATE TABLE invoices (id TEXT PRIMARY KEY, invoice_number TEXT, customer_id TEXT, grand_total_bhd REAL, invoice_hash TEXT, deleted_at DATETIME)`,
		`INSERT INTO invoices (id, invoice_number, customer_id, grand_total_bhd, invoice_hash) VALUES
			('i1', 'INV-26-0001', 'c1', 525.500, 'deadbeef'),
			('i2', 'INV-26-0002', 'c2', 105.000, 'cafebabe')`,
		`CREATE TABLE settings (id TEXT PRIMARY KEY, key TEXT, value TEXT, category TEXT, description TEXT, is_encrypted INTEGER, deleted_at DATETIME)`,
		`INSERT INTO settings (id, key, value, is_encrypted) VALUES
			('s1', 'company.display_name', 'Wasela Trading', 0),
			('s2', 'api.some_key', 'ENCRYPTEDBLOB', 1)`,
		`CREATE TABLE costing_history (id TEXT PRIMARY KEY, product_id TEXT, cost_bhd REAL, deleted_at DATETIME)`,
		`INSERT INTO costing_history (id, product_id, cost_bhd) VALUES ('h1', 'p1', 42.125)`,
		// customer_receipts in the real PH shape: fully applied, partially
		// applied (odd method spelling), reversed, and soft-deleted rows.
		`CREATE TABLE customer_receipts (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, version INTEGER, created_by TEXT,
			receipt_number TEXT, customer_id TEXT, customer_name TEXT, division TEXT, receipt_date DATETIME,
			amount_bhd REAL, applied_amount_bhd REAL, unapplied_amount_bhd REAL,
			payment_method TEXT, reference TEXT, status TEXT, notes TEXT, updated_by TEXT)`,
		`INSERT INTO customer_receipts (id, receipt_number, customer_id, customer_name, receipt_date, amount_bhd, applied_amount_bhd, unapplied_amount_bhd, payment_method, status, deleted_at) VALUES
			('r1', 'RCPT-26-0001', 'c1', 'Nimbus Controls', '2026-05-01', 250.0, 250.0, 0.0, 'Cash', 'Applied', NULL),
			('r2', 'RCPT-26-0002', 'c2', 'Atlas Traders', '2026-05-02', 100.0, 59.5, 40.500, 'Wire Transfer', 'PartiallyApplied', NULL),
			('r3', 'RCPT-26-0003', 'c1', 'Nimbus Controls', '2026-05-03', 25.0, 0.0, 25.0, 'Cash', 'Reversed', NULL),
			('r4', 'RCPT-26-0004', 'c2', 'Atlas Traders', '2026-05-04', 30.0, 0.0, 30.0, 'Cash', 'OnAccount', '2026-05-05')`,
		`CREATE TABLE customer_receipt_allocations (id TEXT PRIMARY KEY, receipt_id TEXT, invoice_id TEXT, payment_id TEXT, allocated_amount_bhd REAL)`,
		`INSERT INTO customer_receipt_allocations (id, receipt_id, invoice_id, payment_id, allocated_amount_bhd) VALUES ('a1', 'r1', 'i1', 'p1', 250.0)`,
		// chart_of_accounts: the provisioned destination carries the same
		// foundation-ensured skeleton (same codes, different ids); imported
		// journal/expense rows reference the SOURCE ids, so source must win.
		`CREATE TABLE chart_of_accounts (id TEXT PRIMARY KEY, account_code TEXT, account_name TEXT, account_type TEXT, deleted_at DATETIME)`,
		`INSERT INTO chart_of_accounts (id, account_code, account_name, account_type) VALUES
			('src-acct-6100', '6100', 'Rent Expense', 'Expense')`,
		// purchase_orders: PH data carries the legacy 'Completed' status that
		// PH itself normalises to 'Closed' on read; the OSS CHECK constraint
		// rejects it, so the copy must normalise in transit.
		`CREATE TABLE purchase_orders (id TEXT PRIMARY KEY, po_number TEXT, supplier_id TEXT, status TEXT, deleted_at DATETIME)`,
		`INSERT INTO purchase_orders (id, po_number, supplier_id, status) VALUES
			('po1', 'PO-26-0001', 'sup1', 'Completed'),
			('po2', 'PO-26-0002', 'sup1', 'Draft')`,
		`CREATE TABLE weird_legacy (id INTEGER PRIMARY KEY, note TEXT)`,
		`INSERT INTO weird_legacy (note) VALUES ('forgotten table')`,
		// Mission H banking suite: statements + lines must carry once the
		// destination provisions the tables (Mission G).
		`CREATE TABLE bank_statements (id TEXT PRIMARY KEY, bank_account_id TEXT, statement_number TEXT, opening_balance REAL, closing_balance REAL, deleted_at DATETIME)`,
		`INSERT INTO bank_statements (id, bank_account_id, statement_number, opening_balance, closing_balance) VALUES
			('bs1', 'ba1', 'STMT-26-01', 1000.000, 1250.500)`,
		`CREATE TABLE bank_statement_lines (id TEXT PRIMARY KEY, bank_statement_id TEXT, line_number INTEGER, debit REAL, credit REAL, balance REAL, deleted_at DATETIME)`,
		`INSERT INTO bank_statement_lines (id, bank_statement_id, line_number, debit, credit, balance) VALUES
			('bl1', 'bs1', 1, 0, 300.500, 1300.500),
			('bl2', 'bs1', 2, 50.000, 0, 1250.500)`,
		// Mission H adjudicated skips (PC-D16): dead scan artifact, enrichment
		// scaffolding, point-in-time backups — all counted, never silent.
		`CREATE TABLE extracted_documents (id INTEGER PRIMARY KEY AUTOINCREMENT, source_path TEXT, document_type TEXT, total_value REAL)`,
		`INSERT INTO extracted_documents (source_path, document_type, total_value) VALUES ('a.pdf', 'offer', 10.0), ('b.pdf', 'invoice', 20.0)`,
		`CREATE TABLE intelligence_order_enrichment (id TEXT PRIMARY KEY, order_id TEXT)`,
		`INSERT INTO intelligence_order_enrichment (id, order_id) VALUES ('en1', 'o1')`,
		`CREATE TABLE customers_backup (id TEXT, business_name TEXT)`,
		`INSERT INTO customers_backup (id, business_name) VALUES ('c1', 'Nimbus Controls')`,
		// A copy-set table with rows whose destination is NOT provisioned in
		// this fixture: the skip must still carry the honest row count.
		`CREATE TABLE vat_returns (id TEXT PRIMARY KEY, period TEXT, total_vat_due REAL, deleted_at DATETIME)`,
		`INSERT INTO vat_returns (id, period, total_vat_due) VALUES ('v1', '2026-Q1', 123.456)`,
		// credit_notes in the real PH spelling: VAT lives in vat_bhd; the OSS
		// model persists the same field as vatbhd (PC-D17 rename pair).
		`CREATE TABLE credit_notes (id TEXT PRIMARY KEY, credit_note_number TEXT, subtotal_bhd REAL, vat_bhd REAL, grand_total_bhd REAL, credit_note_hash TEXT, deleted_at DATETIME)`,
		`INSERT INTO credit_notes (id, credit_note_number, subtotal_bhd, vat_bhd, grand_total_bhd, credit_note_hash) VALUES
			('cn1', 'CN-26-0001', 100.000, 10.500, 110.500, 'feedf00d')`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("source DDL: %v\n%s", err, stmt)
		}
	}
}

// buildDestDB provisions the destination with (a subset of) the real OSS
// schema via the actual GORM models, exactly as the app would.
func buildDestDB(t *testing.T, path string) {
	t.Helper()
	db, err := gorm.Open(gormlite.Open("file:"+filepath.ToSlash(path)), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&crm.CustomerMaster{}, &finance.Invoice{}, &finance.Payment{}, &infradomain.Setting{}, &crm.CostingHistory{}, &crm.PurchaseOrder{}, &finance.ChartOfAccount{}, &finance.BankStatement{}, &finance.BankStatementLine{}, &finance.CreditNote{}); err != nil {
		t.Fatal(err)
	}
	// The app's expense foundation ensures a skeleton account on provision —
	// same code as the source's row but a destination-local id.
	if err := db.Exec(`INSERT INTO chart_of_accounts (id, account_code, account_name, account_type) VALUES ('dest-acct-6100', '6100', 'Rent Expense', 'Expense')`).Error; err != nil {
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

func TestRun_CopiesTransformsAndReports(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "ph_source.db")
	destPath := filepath.Join(dir, "oss_dest.db")
	buildSourceDB(t, sourcePath)
	buildDestDB(t, destPath)

	report, err := Run(context.Background(), Options{SourcePath: sourcePath, DestPath: destPath})
	if err != nil {
		t.Fatal(err)
	}

	copied := map[string]TableCopy{}
	for _, c := range report.Copied {
		copied[c.Source] = c
	}
	if copied["customers"].Rows != 2 {
		t.Fatalf("customers copy: %+v", copied["customers"])
	}
	if copied["invoices"].Rows != 2 {
		t.Fatalf("invoices copy: %+v", copied["invoices"])
	}
	if c := copied["costing_history"]; c.Dest != "costing_histories" || c.Rows != 1 {
		t.Fatalf("rename copy: %+v", c)
	}
	if c := copied["settings"]; c.Rows != 1 {
		t.Fatalf("settings should copy only the plaintext row, got %+v", c)
	}
	if report.EncryptedSettingsSkipped != 1 {
		t.Fatalf("encrypted settings skipped: %d", report.EncryptedSettingsSkipped)
	}
	if report.InvoiceHashesBlanked != 2 {
		t.Fatalf("invoice hashes blanked: %d", report.InvoiceHashesBlanked)
	}

	// PC-D7: only r2's unapplied remainder becomes a payment. r1 is fully
	// applied (its money is the copied allocation payment), r3 is reversed,
	// r4 is soft-deleted.
	if report.ReceiptsTransformed != 1 {
		t.Fatalf("receipts transformed: %d", report.ReceiptsTransformed)
	}
	if report.ReceiptsOnAccountBHD != 40.5 {
		t.Fatalf("on-account BHD must copy exactly: %v", report.ReceiptsOnAccountBHD)
	}
	if report.ReceiptsFullyApplied != 1 || report.ReceiptsReversedOrVoided != 2 {
		t.Fatalf("receipt accounting: fully-applied=%d reversed/voided=%d",
			report.ReceiptsFullyApplied, report.ReceiptsReversedOrVoided)
	}
	if c, ok := copied["customer_receipts"]; !ok || c.Dest != "payments" || c.Rows != 1 || c.Reason == "" {
		t.Fatalf("customer_receipts must report as a transform into payments: %+v", c)
	}
	foundAllocations := false
	for _, s := range report.Skipped {
		if s.Source == "customer_receipt_allocations" {
			foundAllocations = true
			if s.Rows != 1 || s.Reason == "" {
				t.Fatalf("allocation skip must carry count+reason: %+v", s)
			}
		}
	}
	if !foundAllocations {
		t.Fatal("customer_receipt_allocations must be reported as skipped (represented by payments)")
	}

	foundLegacy := false
	for _, u := range report.Unmapped {
		if u.Source == "weird_legacy" && u.Rows == 1 {
			foundLegacy = true
		}
	}
	if !foundLegacy {
		t.Fatalf("unknown source tables must be reported unmapped, got %+v", report.Unmapped)
	}
	if len(report.Unmapped) != 1 {
		t.Fatalf("Mission H: every known PH table must be adjudicated (carry/transform/skip); unexpected unmapped set: %+v", report.Unmapped)
	}

	// Mission H: banking suite carries once the destination provisions it.
	if c := copied["bank_statements"]; c.Rows != 1 {
		t.Fatalf("bank_statements copy: %+v", c)
	}
	if c := copied["bank_statement_lines"]; c.Rows != 2 {
		t.Fatalf("bank_statement_lines copy: %+v", c)
	}

	// Mission H adjudicated skips carry honest counts + reasons.
	skips := map[string]TableCopy{}
	for _, s := range report.Skipped {
		skips[s.Source] = s
	}
	if s := skips["extracted_documents"]; s.Rows != 2 || !strings.Contains(s.Reason, "PC-D16") {
		t.Fatalf("extracted_documents skip must carry count+decision: %+v", s)
	}
	if s := skips["intelligence_order_enrichment"]; s.Rows != 1 || s.Reason == "" {
		t.Fatalf("intelligence enrichment skip: %+v", s)
	}
	if s := skips["customers_backup"]; s.Rows != 1 || !strings.Contains(s.Reason, "backup") {
		t.Fatalf("customers_backup skip: %+v", s)
	}
	// The no-destination-table skip path must also count source rows — a skip
	// entry showing 0 rows for a populated table is a silent drop in disguise.
	if s := skips["vat_returns"]; s.Rows != 1 || !strings.Contains(s.Reason, "no destination table") {
		t.Fatalf("vat_returns (unprovisioned in this fixture) must skip with the honest count: %+v", s)
	}

	// PC-D17 rename pair: credit-note VAT must land under the OSS spelling,
	// and the copy entry must say so.
	if c := copied["credit_notes"]; c.Rows != 1 || !strings.Contains(c.Reason, "vat_bhd→vatbhd") {
		t.Fatalf("credit_notes copy must report the vat_bhd→vatbhd rename: %+v", c)
	}

	// Faithful column drops (columns PH's own app no longer reads) are
	// reported with non-empty counts, never silent.
	foundDrop := false
	for _, d := range report.ColumnDrops {
		if d.Table == "customers" && d.Column == "ph_only_flag" {
			foundDrop = true
			if d.NonEmptyRows != 1 {
				t.Fatalf("ph_only_flag drop must count the single non-empty value: %+v", d)
			}
		}
	}
	if !foundDrop {
		t.Fatalf("dropped source columns holding data must be reported, got %+v", report.ColumnDrops)
	}

	// Ground-truth the destination file.
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(destPath)+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var businessName string
	if err := db.QueryRow(`SELECT business_name FROM customers WHERE id = 'c1'`).Scan(&businessName); err != nil || businessName != "Nimbus Controls" {
		t.Fatalf("customer row: %q %v", businessName, err)
	}
	var hash string
	var total float64
	if err := db.QueryRow(`SELECT COALESCE(invoice_hash, ''), grand_total_bhd FROM invoices WHERE id = 'i1'`).Scan(&hash, &total); err != nil {
		t.Fatal(err)
	}
	if hash != "" {
		t.Fatalf("invoice hash must be blanked for destination-salt recompute, got %q", hash)
	}
	if total != 525.5 {
		t.Fatalf("money must copy exactly, got %v", total)
	}
	var settingsCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM settings`).Scan(&settingsCount); err != nil || settingsCount != 1 {
		t.Fatalf("settings count: %d %v", settingsCount, err)
	}
	var costBHD float64
	if err := db.QueryRow(`SELECT cost_bhd FROM costing_histories WHERE id = 'h1'`).Scan(&costBHD); err != nil || costBHD != 42.125 {
		t.Fatalf("renamed table row: %v %v", costBHD, err)
	}
	var cnVAT float64
	var cnHash string
	if err := db.QueryRow(`SELECT vatbhd, COALESCE(credit_note_hash,'') FROM credit_notes WHERE id = 'cn1'`).Scan(&cnVAT, &cnHash); err != nil || cnVAT != 10.5 {
		t.Fatalf("credit-note VAT must carry under the OSS spelling: %v %v", cnVAT, err)
	}
	if cnHash != "" {
		t.Fatalf("credit-note hash must be blanked for destination-salt recompute, got %q", cnHash)
	}
	var closing, lineSum float64
	if err := db.QueryRow(`SELECT closing_balance FROM bank_statements WHERE id = 'bs1'`).Scan(&closing); err != nil || closing != 1250.5 {
		t.Fatalf("bank statement closing balance must copy exactly: %v %v", closing, err)
	}
	if err := db.QueryRow(`SELECT SUM(credit) - SUM(debit) FROM bank_statement_lines`).Scan(&lineSum); err != nil || lineSum != 250.5 {
		t.Fatalf("bank line sums must copy exactly: %v %v", lineSum, err)
	}

	// The on-account payment: exact amount, normalized method, blank invoice,
	// deterministic idempotency key, traceability packed into reference.
	var payAmount float64
	var payMethod, payInvoiceID, payKey, payRef string
	err = db.QueryRow(`SELECT amount_bhd, payment_method, COALESCE(invoice_id,''), idempotency_key, reference FROM payments WHERE id = 'r2'`).
		Scan(&payAmount, &payMethod, &payInvoiceID, &payKey, &payRef)
	if err != nil {
		t.Fatalf("on-account payment row: %v", err)
	}
	if payAmount != 40.5 {
		t.Fatalf("on-account amount must copy exactly, got %v", payAmount)
	}
	if payMethod != "Bank Transfer" {
		t.Fatalf("'Wire Transfer' must normalize into the payments check constraint, got %q", payMethod)
	}
	if payInvoiceID != "" {
		t.Fatalf("on-account payment must be invoice-less, got %q", payInvoiceID)
	}
	if payKey != "phimport-receipt-r2" {
		t.Fatalf("idempotency key must be deterministic on the receipt id, got %q", payKey)
	}
	if !strings.Contains(payRef, "RCPT-26-0002") || !strings.Contains(payRef, "c2") {
		t.Fatalf("reference must carry receipt number + customer id, got %q", payRef)
	}
	var payCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM payments`).Scan(&payCount); err != nil || payCount != 1 {
		t.Fatalf("exactly the one on-account payment must exist (no double-counted applied money): %d %v", payCount, err)
	}

	// The provisioned account skeleton must be replaced by the source chart —
	// same code, but the SOURCE row id survives (children reference it).
	if c := copied["chart_of_accounts"]; c.Rows != 1 || !strings.Contains(c.Reason, "replaced") {
		t.Fatalf("chart_of_accounts must report the baseline replacement: %+v", c)
	}
	var acctID string
	var acctCount int
	if err := db.QueryRow(`SELECT id FROM chart_of_accounts WHERE account_code = '6100'`).Scan(&acctID); err != nil || acctID != "src-acct-6100" {
		t.Fatalf("source chart row must win the baseline replacement, got %q %v", acctID, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM chart_of_accounts`).Scan(&acctCount); err != nil || acctCount != 1 {
		t.Fatalf("no duplicate accounts after replacement: %d %v", acctCount, err)
	}

	// Legacy PO status normalises in transit (PH's own read-side mapping:
	// completed → Closed); modern spellings pass through untouched.
	if c := copied["purchase_orders"]; c.Rows != 2 || !strings.Contains(c.Reason, "status") {
		t.Fatalf("purchase_orders copy must report the in-transit status normalisation: %+v", c)
	}
	var poStatus string
	if err := db.QueryRow(`SELECT status FROM purchase_orders WHERE id = 'po1'`).Scan(&poStatus); err != nil || poStatus != "Closed" {
		t.Fatalf("legacy 'Completed' PO must import as 'Closed', got %q %v", poStatus, err)
	}
	if err := db.QueryRow(`SELECT status FROM purchase_orders WHERE id = 'po2'`).Scan(&poStatus); err != nil || poStatus != "Draft" {
		t.Fatalf("modern PO status must pass through, got %q %v", poStatus, err)
	}
}

func TestRun_RefusesUnprovisionedDestination(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "ph_source.db")
	destPath := filepath.Join(dir, "empty_dest.db")
	buildSourceDB(t, sourcePath)

	// Destination file exists but carries no OSS schema.
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(destPath))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE placeholder_only (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	db.Close()

	if _, err := Run(context.Background(), Options{SourcePath: sourcePath, DestPath: destPath}); err == nil {
		t.Fatal("must refuse a destination without the OSS schema")
	}
}
