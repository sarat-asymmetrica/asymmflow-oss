package phreconcile

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// commonDDL is the minimal shape both sides share. SYNTHETIC data only.
var commonDDL = []string{
	// Enriched columns (PC-D22, Mission I D-I-5) are present on both sides so
	// the per-column non-empty checks read the same shape source and dest.
	`CREATE TABLE customers (id TEXT, deleted_at DATETIME, tax_code TEXT, address TEXT, phone TEXT, email TEXT)`,
	`CREATE TABLE suppliers (id TEXT, deleted_at DATETIME, is_active INTEGER)`,
	`CREATE TABLE products (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE users (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE roles (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE employees (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE customer_contacts (id TEXT, deleted_at DATETIME, is_primary NUMERIC, salutation TEXT)`,
	`CREATE TABLE opportunities (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE rfq_data (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE costing_sheet_data (id TEXT, offer_number TEXT, customer_name TEXT, product_type TEXT,
		total_value_bhd REAL, line_item_count INTEGER, source_file_path TEXT, extracted_at DATETIME)`,
	`CREATE TABLE offers (id TEXT, total_value_bhd REAL, deleted_at DATETIME)`,
	`CREATE TABLE offer_items (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE orders (id TEXT, total_value_bhd REAL, grand_total_bhd REAL, deleted_at DATETIME)`,
	`CREATE TABLE order_items (id TEXT, deleted_at DATETIME, unit_price_bhd DECIMAL(15,3), brand TEXT, token TEXT)`,
	`CREATE TABLE invoices (id TEXT, invoice_date DATETIME, status TEXT, subtotal_bhd REAL, vatbhd REAL, grand_total_bhd REAL, outstanding_bhd REAL, notes TEXT, deleted_at DATETIME)`,
	`CREATE TABLE invoice_items (id TEXT, total_bhd REAL, deleted_at DATETIME)`,
	`CREATE TABLE credit_note_items (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE payments (id TEXT, amount_bhd REAL, deleted_at DATETIME)`,
	`CREATE TABLE purchase_orders (id TEXT, subtotal_bhd REAL, vat_amount REAL, total_bhd REAL, total_foreign REAL, deleted_at DATETIME)`,
	`CREATE TABLE purchase_order_items (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE supplier_invoices (id TEXT, subtotal_bhd REAL, vatbhd REAL, total_bhd REAL, deleted_at DATETIME)`,
	`CREATE TABLE supplier_invoice_items (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE supplier_payments (id TEXT, amount_bhd REAL, amount_foreign REAL, payment_number TEXT, deleted_at DATETIME)`,
	`CREATE TABLE goods_received_notes (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE grn_items (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE delivery_notes (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE delivery_note_items (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE chart_of_accounts (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE company_bank_accounts (id TEXT, deleted_at DATETIME)`,
	`CREATE TABLE currency_exchange_rates (id TEXT, rate REAL, deleted_at DATETIME)`,
	`CREATE TABLE journal_entries (id TEXT)`,
	`CREATE TABLE journal_lines (id TEXT, debit REAL, credit REAL)`,
	`CREATE TABLE bank_accounts (id TEXT, current_balance REAL)`,
	`CREATE TABLE bank_statements (id TEXT, opening_balance REAL, closing_balance REAL, total_debits REAL, total_credits REAL)`,
	`CREATE TABLE bank_statement_lines (id TEXT, bank_statement_id TEXT, debit REAL, credit REAL, balance REAL)`,
	`CREATE TABLE bank_line_payment_allocations (id TEXT)`,
	`CREATE TABLE cheque_registers (id TEXT)`,
	`CREATE TABLE outstanding_cheques (id TEXT, amount REAL)`,
	`CREATE TABLE deposits_in_transit (id TEXT, amount REAL)`,
	`CREATE TABLE bank_cash_balances (id TEXT, statement_balance REAL, computed_balance REAL)`,
	`CREATE TABLE bank_expense_entries (id TEXT, amount REAL, vat_amount REAL)`,
	`CREATE TABLE statement_hashes (id TEXT)`,
	`CREATE TABLE book_bank_reconciliations (id TEXT)`,
	`CREATE TABLE bank_reconciliation_audit_logs (id TEXT)`,
	`CREATE TABLE bank_statement_files (id TEXT)`,
	`CREATE TABLE fx_rates (id TEXT, rate REAL)`,
	`CREATE TABLE fx_revaluations (id TEXT, gain_loss_bhd REAL)`,
	`CREATE TABLE vat_returns (id TEXT, net_vat REAL)`,
}

var commonInserts = []string{
	// Column-explicit so the enriched columns (unpopulated except email/notes,
	// which exercise the non-empty path identically on both sides) are free to
	// be added to the DDL without disturbing these positional loads.
	`INSERT INTO customers (id, deleted_at, email) VALUES ('c1', NULL, 'a@x.test'), ('c2', '2026-01-01', '')`,
	`INSERT INTO invoices (id, invoice_date, status, subtotal_bhd, vatbhd, grand_total_bhd, outstanding_bhd, notes, deleted_at) VALUES
		('i1', '2026-03-01', 'Paid', 100.0, 10.5, 110.5, 0, 'gate 4', NULL),
		('i2', '2025-06-01', 'Sent', 200.0, 20.0, 220.0, 220.0, '', NULL)`,
	`INSERT INTO invoice_items VALUES ('ii1', 100.0, NULL), ('ii2', 200.0, NULL)`,
	`INSERT INTO bank_statements VALUES ('bs1', 1000.0, 1250.5, 50.0, 300.5)`,
	`INSERT INTO bank_statement_lines VALUES ('bl1', 'bs1', 0, 300.5, 1300.5), ('bl2', 'bs1', 50.0, 0, 1250.5)`,
	`INSERT INTO bank_accounts VALUES ('ba1', 1250.5)`,
	`INSERT INTO fx_rates VALUES ('fx1', 0.376)`,
	`INSERT INTO vat_returns VALUES ('v1', 123.456)`,
}

func buildDB(t *testing.T, path string, extra ...string) {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, stmt := range append(append([]string{}, commonDDL...), commonInserts...) {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%v\n%s", err, stmt)
		}
	}
	for _, stmt := range extra {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%v\n%s", err, stmt)
		}
	}
}

// Source-flavoured divergences: PH spellings + receipt ledger.
var sourceExtra = []string{
	`CREATE TABLE costing_line_items (id TEXT)`,
	`INSERT INTO costing_line_items VALUES ('cl1')`,
	`CREATE TABLE credit_notes (id TEXT, subtotal_bhd REAL, vat_bhd REAL, grand_total_bhd REAL, deleted_at DATETIME)`,
	`INSERT INTO credit_notes VALUES ('cn1', 100.0, 10.5, 110.5, NULL)`,
	`CREATE TABLE settings (id TEXT, key TEXT, is_encrypted INTEGER)`,
	`INSERT INTO settings VALUES ('s1', 'company.display_name', 0), ('s2', 'api.some_key', 1)`,
	`CREATE TABLE customer_receipts (id TEXT, status TEXT, unapplied_amount_bhd REAL, deleted_at DATETIME)`,
	`INSERT INTO customer_receipts VALUES ('r1', 'PartiallyApplied', 40.5, NULL), ('r2', 'Reversed', 25.0, NULL)`,
	`INSERT INTO payments VALUES ('p1', 110.5, NULL)`,
}

// Destination-flavoured: OSS spellings; the PC-D7 remainder is a payment row.
var destExtra = []string{
	`CREATE TABLE costing_line_item_data (id TEXT)`,
	`INSERT INTO costing_line_item_data VALUES ('cl1')`,
	`CREATE TABLE credit_notes (id TEXT, subtotal_bhd REAL, vatbhd REAL, grand_total_bhd REAL, deleted_at DATETIME)`,
	`INSERT INTO credit_notes VALUES ('cn1', 100.0, 10.5, 110.5, NULL)`,
	`CREATE TABLE settings (id TEXT, key TEXT)`,
	// The provisioning boot writes its own backup bookkeeping — the harness
	// must tolerate exactly that and nothing else.
	`INSERT INTO settings VALUES ('s1', 'company.display_name'), ('sb', 'backup_last_at')`,
	`INSERT INTO payments VALUES ('p1', 110.5, NULL), ('r1', 40.5, NULL)`,
}

func TestRun_MatchedMigrationReconciles(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.db")
	destPath := filepath.Join(dir, "dest.db")
	buildDB(t, srcPath, sourceExtra...)
	buildDB(t, destPath, destExtra...)

	report, err := Run(context.Background(), Options{SourcePath: srcPath, DestPath: destPath})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Pass {
		for _, c := range report.Checks {
			if !c.Match {
				t.Errorf("check %q: source=%q dest=%q", c.Name, c.Source, c.Dest)
			}
		}
		t.Fatalf("matched fixture must reconcile: %d/%d", report.Matched, report.Matched+report.Failed)
	}
	if len(report.Checks) <= 25 {
		t.Fatalf("Mission H must extend the Mission E core set (N>25), got %d checks", len(report.Checks))
	}
}

func TestRun_MoneyMismatchFails(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.db")
	destPath := filepath.Join(dir, "dest.db")
	buildDB(t, srcPath, sourceExtra...)
	buildDB(t, destPath, destExtra...)

	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(destPath))
	if err != nil {
		t.Fatal(err)
	}
	// One fils of drift on one invoice must fail the gate.
	if _, err := db.Exec(`UPDATE invoices SET grand_total_bhd = grand_total_bhd + 0.001 WHERE id = 'i1'`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	report, err := Run(context.Background(), Options{SourcePath: srcPath, DestPath: destPath})
	if err != nil {
		t.Fatal(err)
	}
	if report.Pass {
		t.Fatal("a one-fils drift must fail reconciliation")
	}
	found := false
	for _, c := range report.Checks {
		if !c.Match && strings.Contains(c.Name, "invoices live") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the invoice money check must be the one that fails: %+v", report.Checks)
	}
}

func TestRun_MissingTableIsAFailureNotAPass(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.db")
	destPath := filepath.Join(dir, "dest.db")
	buildDB(t, srcPath, sourceExtra...)
	buildDB(t, destPath, destExtra...)

	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(destPath))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE vat_returns`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	report, err := Run(context.Background(), Options{SourcePath: srcPath, DestPath: destPath})
	if err != nil {
		t.Fatal(err)
	}
	if report.Pass {
		t.Fatal("an unreadable destination table must fail the gate, not silently pass")
	}
}
