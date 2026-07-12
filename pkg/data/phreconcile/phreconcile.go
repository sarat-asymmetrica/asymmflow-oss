// Package phreconcile is the PH-format → OSS-format migration reconciliation
// harness (PH convergence Mission H). It re-runs the Mission E check set —
// extended to the full banking/FX/VAT surface — as CODE, so the cutover
// runbook has a deterministic, re-runnable acceptance gate instead of a
// one-off scratchpad session. Only code lives in this repository; the data it
// checks lives solely in the operator's out-of-repo working directory.
//
// Every check runs an aggregate query against BOTH files and compares the
// rendered results exactly; money renders to 3 decimals (the fils). A check
// that does not match is a live bug in the source data or the import mapping
// — stop and ask, never fudge (wave invariant).
//
// Checks come in two flavours:
//   - carry checks: COUNT(*) over ALL rows (soft-deleted included) — the
//     import carries everything, so totals must match row-for-row;
//   - money checks: sums over LIVE rows (deleted_at IS NULL), matching how
//     Mission E reconciled the books.
//
// Where the two schemas genuinely diverge the destination query is overridden
// (costing_line_items → costing_line_item_data, credit-note vat_bhd → vatbhd,
// PC-D7 payments = source payments + unapplied on-account receipts, encrypted
// settings not carried).
package phreconcile

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"
)

// Options names the two database files. Both are opened read-only.
type Options struct {
	SourcePath string
	DestPath   string
}

// CheckResult is one check's outcome, with both rendered values so a mismatch
// is diagnosable from the report alone.
type CheckResult struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Dest   string `json:"dest"`
	Match  bool   `json:"match"`
}

// Report is the full harness outcome.
type Report struct {
	Checks  []CheckResult `json:"checks"`
	Matched int           `json:"matched"`
	Failed  int           `json:"failed"`
	Pass    bool          `json:"pass"`
}

type check struct {
	name    string
	srcSQL  string
	destSQL string // empty → srcSQL runs on both sides
}

// PC-D7 receipt liveness — must mirror pkg/data/phimport exactly, or the
// payments check would disagree with the importer about what money moved.
const receiptLive = `r.deleted_at IS NULL AND COALESCE(r.status, '') != 'Reversed'`
const receiptOnAccount = receiptLive + ` AND COALESCE(r.unapplied_amount_bhd, 0) > 0.0005`

var checks = []check{
	// ── Carry checks: every row accounted, soft-deleted included ──────────
	{name: "customers rows", srcSQL: `SELECT COUNT(*) FROM customers`},
	{name: "suppliers rows", srcSQL: `SELECT COUNT(*) FROM suppliers`},
	{name: "products rows", srcSQL: `SELECT COUNT(*) FROM products`},
	{name: "users rows", srcSQL: `SELECT COUNT(*) FROM users`},
	{name: "roles rows", srcSQL: `SELECT COUNT(*) FROM roles`},
	{name: "employees rows", srcSQL: `SELECT COUNT(*) FROM employees`},
	{name: "customer_contacts rows", srcSQL: `SELECT COUNT(*) FROM customer_contacts`},
	{name: "opportunities rows", srcSQL: `SELECT COUNT(*) FROM opportunities`},
	{name: "rfq_data rows", srcSQL: `SELECT COUNT(*) FROM rfq_data`},
	{name: "costing sheets rows", srcSQL: `SELECT COUNT(*) FROM costing_sheet_data`},
	{name: "costing line items rows", srcSQL: `SELECT COUNT(*) FROM costing_line_items`,
		destSQL: `SELECT COUNT(*) FROM costing_line_item_data`},
	{name: "offers rows", srcSQL: `SELECT COUNT(*) FROM offers`},
	{name: "offer_items rows", srcSQL: `SELECT COUNT(*) FROM offer_items`},
	{name: "orders rows", srcSQL: `SELECT COUNT(*) FROM orders`},
	{name: "order_items rows", srcSQL: `SELECT COUNT(*) FROM order_items`},
	{name: "invoices rows", srcSQL: `SELECT COUNT(*) FROM invoices`},
	{name: "invoice_items rows", srcSQL: `SELECT COUNT(*) FROM invoice_items`},
	{name: "credit_notes rows", srcSQL: `SELECT COUNT(*) FROM credit_notes`},
	{name: "credit_note_items rows", srcSQL: `SELECT COUNT(*) FROM credit_note_items`},
	{name: "payments rows (source + PC-D7 on-account receipts)",
		srcSQL: `SELECT (SELECT COUNT(*) FROM payments) +
			(SELECT COUNT(*) FROM customer_receipts r WHERE ` + receiptOnAccount + `)`,
		destSQL: `SELECT COUNT(*) FROM payments`},
	{name: "purchase_orders rows", srcSQL: `SELECT COUNT(*) FROM purchase_orders`},
	{name: "purchase_order_items rows", srcSQL: `SELECT COUNT(*) FROM purchase_order_items`},
	{name: "supplier_invoices rows", srcSQL: `SELECT COUNT(*) FROM supplier_invoices`},
	{name: "supplier_invoice_items rows", srcSQL: `SELECT COUNT(*) FROM supplier_invoice_items`},
	{name: "supplier_payments rows", srcSQL: `SELECT COUNT(*) FROM supplier_payments`},
	{name: "goods_received_notes rows", srcSQL: `SELECT COUNT(*) FROM goods_received_notes`},
	{name: "grn_items rows", srcSQL: `SELECT COUNT(*) FROM grn_items`},
	{name: "delivery_notes rows", srcSQL: `SELECT COUNT(*) FROM delivery_notes`},
	{name: "delivery_note_items rows", srcSQL: `SELECT COUNT(*) FROM delivery_note_items`},
	{name: "chart_of_accounts rows", srcSQL: `SELECT COUNT(*) FROM chart_of_accounts`},
	{name: "company_bank_accounts rows", srcSQL: `SELECT COUNT(*) FROM company_bank_accounts`},
	{name: "currency_exchange_rates rows + rate sum",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(rate), 0), 6) FROM currency_exchange_rates`},
	// Key-list compare (stronger than a count): every non-encrypted source
	// setting must arrive. The destination side excludes the app's OWN
	// install-local bookkeeping (the backup_* keys the provisioning boot
	// writes) — any other destination-only key fails the gate and gets
	// investigated.
	{name: "settings keys (encrypted rows not carried; install-local backup_* bookkeeping excluded)",
		srcSQL:  `SELECT key FROM settings WHERE COALESCE(is_encrypted, 0) = 0 ORDER BY key`,
		destSQL: `SELECT key FROM settings WHERE key NOT LIKE 'backup\_%' ESCAPE '\' ORDER BY key`},

	// ── Money checks: reconciled to the fils on live rows ─────────────────
	{name: "invoices live count/subtotal/VAT/grand/outstanding BHD",
		srcSQL: `SELECT COUNT(*), ROUND(SUM(subtotal_bhd), 3), ROUND(SUM(COALESCE(vatbhd, 0)), 3),
			ROUND(SUM(grand_total_bhd), 3), ROUND(SUM(COALESCE(outstanding_bhd, 0)), 3)
			FROM invoices WHERE deleted_at IS NULL`},
	{name: "invoices grand BHD per year (live)",
		srcSQL: `SELECT strftime('%Y', invoice_date), COUNT(*), ROUND(SUM(grand_total_bhd), 3)
			FROM invoices WHERE deleted_at IS NULL GROUP BY 1 ORDER BY 1`},
	{name: "invoices count per status (live)",
		srcSQL: `SELECT COALESCE(status, ''), COUNT(*) FROM invoices WHERE deleted_at IS NULL GROUP BY 1 ORDER BY 1`},
	{name: "invoice_items live count + line total BHD",
		srcSQL: `SELECT COUNT(*), ROUND(SUM(COALESCE(total_bhd, 0)), 3) FROM invoice_items WHERE deleted_at IS NULL`},
	{name: "payments live amount BHD (source + PC-D7 on-account remainders)",
		srcSQL: `SELECT ROUND((SELECT COALESCE(SUM(amount_bhd), 0) FROM payments WHERE deleted_at IS NULL) +
			(SELECT COALESCE(SUM(r.unapplied_amount_bhd), 0) FROM customer_receipts r WHERE ` + receiptOnAccount + `), 3)`,
		destSQL: `SELECT ROUND(COALESCE(SUM(amount_bhd), 0), 3) FROM payments WHERE deleted_at IS NULL`},
	{name: "credit_notes live subtotal/VAT/grand BHD",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(subtotal_bhd), 0), 3), ROUND(COALESCE(SUM(vat_bhd), 0), 3),
			ROUND(COALESCE(SUM(grand_total_bhd), 0), 3) FROM credit_notes WHERE deleted_at IS NULL`,
		destSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(subtotal_bhd), 0), 3), ROUND(COALESCE(SUM(vatbhd), 0), 3),
			ROUND(COALESCE(SUM(grand_total_bhd), 0), 3) FROM credit_notes WHERE deleted_at IS NULL`},
	{name: "supplier_invoices live subtotal/VAT/total BHD",
		srcSQL: `SELECT COUNT(*), ROUND(SUM(COALESCE(subtotal_bhd, 0)), 3), ROUND(SUM(COALESCE(vatbhd, 0)), 3),
			ROUND(SUM(COALESCE(total_bhd, 0)), 3) FROM supplier_invoices WHERE deleted_at IS NULL`},
	{name: "supplier_payments live amount BHD + foreign",
		srcSQL: `SELECT COUNT(*), ROUND(SUM(COALESCE(amount_bhd, 0)), 3), ROUND(SUM(COALESCE(amount_foreign, 0)), 3)
			FROM supplier_payments WHERE deleted_at IS NULL`},
	{name: "orders live value/grand BHD",
		srcSQL: `SELECT COUNT(*), ROUND(SUM(COALESCE(total_value_bhd, 0)), 3), ROUND(SUM(COALESCE(grand_total_bhd, 0)), 3)
			FROM orders WHERE deleted_at IS NULL`},
	{name: "offers live total value BHD",
		srcSQL: `SELECT COUNT(*), ROUND(SUM(COALESCE(total_value_bhd, 0)), 3) FROM offers WHERE deleted_at IS NULL`},
	{name: "purchase_orders live subtotal/VAT/total BHD + foreign",
		srcSQL: `SELECT COUNT(*), ROUND(SUM(COALESCE(subtotal_bhd, 0)), 3), ROUND(SUM(COALESCE(vat_amount, 0)), 3),
			ROUND(SUM(COALESCE(total_bhd, 0)), 3), ROUND(SUM(COALESCE(total_foreign, 0)), 3)
			FROM purchase_orders WHERE deleted_at IS NULL`},
	{name: "journal entries + lines, debit/credit sums (live)",
		srcSQL: `SELECT (SELECT COUNT(*) FROM journal_entries), (SELECT COUNT(*) FROM journal_lines),
			(SELECT ROUND(COALESCE(SUM(debit), 0), 3) FROM journal_lines),
			(SELECT ROUND(COALESCE(SUM(credit), 0), 3) FROM journal_lines)`},

	// ── Mission H banking / FX / VAT extension ─────────────────────────────
	{name: "bank_accounts rows + current balance",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(current_balance), 0), 3) FROM bank_accounts`},
	{name: "bank_statements rows + opening/closing/debits/credits",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(opening_balance), 0), 3), ROUND(COALESCE(SUM(closing_balance), 0), 3),
			ROUND(COALESCE(SUM(total_debits), 0), 3), ROUND(COALESCE(SUM(total_credits), 0), 3) FROM bank_statements`},
	{name: "bank_statement_lines rows + debit/credit sums",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(debit), 0), 3), ROUND(COALESCE(SUM(credit), 0), 3) FROM bank_statement_lines`},
	{name: "bank_statement_lines per statement (count, debit, credit, running balance)",
		srcSQL: `SELECT bank_statement_id, COUNT(*), ROUND(COALESCE(SUM(debit), 0), 3),
			ROUND(COALESCE(SUM(credit), 0), 3), ROUND(COALESCE(SUM(balance), 0), 3)
			FROM bank_statement_lines GROUP BY 1 ORDER BY 1`},
	{name: "bank_line_payment_allocations rows", srcSQL: `SELECT COUNT(*) FROM bank_line_payment_allocations`},
	{name: "cheque_registers rows", srcSQL: `SELECT COUNT(*) FROM cheque_registers`},
	{name: "outstanding_cheques rows + amount",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(amount), 0), 3) FROM outstanding_cheques`},
	{name: "deposits_in_transit rows + amount",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(amount), 0), 3) FROM deposits_in_transit`},
	{name: "bank_cash_balances rows + statement/computed balances",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(statement_balance), 0), 3),
			ROUND(COALESCE(SUM(computed_balance), 0), 3) FROM bank_cash_balances`},
	{name: "bank_expense_entries rows + amount + VAT",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(amount), 0), 3), ROUND(COALESCE(SUM(vat_amount), 0), 3) FROM bank_expense_entries`},
	{name: "statement_hashes rows", srcSQL: `SELECT COUNT(*) FROM statement_hashes`},
	{name: "book_bank_reconciliations rows", srcSQL: `SELECT COUNT(*) FROM book_bank_reconciliations`},
	{name: "bank_reconciliation_audit_logs rows", srcSQL: `SELECT COUNT(*) FROM bank_reconciliation_audit_logs`},
	{name: "bank_statement_files rows", srcSQL: `SELECT COUNT(*) FROM bank_statement_files`},
	{name: "fx_rates rows + rate sum",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(rate), 0), 6) FROM fx_rates`},
	{name: "fx_revaluations rows + gain/loss BHD",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(gain_loss_bhd), 0), 3) FROM fx_revaluations`},
	{name: "vat_returns rows + net VAT",
		srcSQL: `SELECT COUNT(*), ROUND(COALESCE(SUM(net_vat), 0), 3) FROM vat_returns`},

	// ── PC-D22 (Mission I D-I-5) column-enrichment checks ─────────────────
	// The 19 formerly-dropped source columns now carry into matching dest
	// columns. Each check counts NON-EMPTY values with the SAME predicate the
	// importer uses to decide a column "holds data" (non-null and not in
	// ''/0/0.0) so a mismatch proves the enrichment did not round-trip. The
	// dest column name equals the source name for every enriched column, so
	// one query runs on both sides.
	nonEmptyCheck("customers.tax_code"),
	nonEmptyCheck("customers.address"),
	nonEmptyCheck("customers.phone"),
	nonEmptyCheck("customers.email"),
	nonEmptyCheck("suppliers.is_active"),
	nonEmptyCheck("customer_contacts.is_primary"),
	nonEmptyCheck("customer_contacts.salutation"),
	nonEmptyCheck("costing_sheet_data.offer_number"),
	nonEmptyCheck("costing_sheet_data.customer_name"),
	nonEmptyCheck("costing_sheet_data.product_type"),
	nonEmptyCheck("costing_sheet_data.total_value_bhd"),
	nonEmptyCheck("costing_sheet_data.line_item_count"),
	nonEmptyCheck("costing_sheet_data.source_file_path"),
	nonEmptyCheck("costing_sheet_data.extracted_at"),
	nonEmptyCheck("order_items.unit_price_bhd"),
	nonEmptyCheck("order_items.brand"),
	nonEmptyCheck("order_items.token"),
	nonEmptyCheck("invoices.notes"),
	nonEmptyCheck("supplier_payments.payment_number"),
}

// nonEmptyCheck builds a reconciliation check counting rows whose "table.column"
// holds data — non-null and, cast to text and trimmed, not '', '0', or '0.0'.
// The predicate mirrors pkg/data/phimport's ColumnDrop non-empty test exactly,
// so a per-column count that matches source-to-dest proves the enriched column
// round-tripped every value the importer would otherwise have reported dropped.
func nonEmptyCheck(tableDotColumn string) check {
	parts := strings.SplitN(tableDotColumn, ".", 2)
	table, col := parts[0], parts[1]
	sql := fmt.Sprintf(
		`SELECT COUNT(*) FROM %s WHERE %s IS NOT NULL AND TRIM(CAST(%s AS TEXT)) NOT IN ('', '0', '0.0')`,
		table, col, col)
	return check{name: "enriched non-empty: " + tableDotColumn, srcSQL: sql}
}

func render(vals []any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		switch t := v.(type) {
		case nil:
			parts[i] = "NULL"
		case []byte:
			parts[i] = string(t)
		case float64:
			parts[i] = strconv.FormatFloat(t, 'f', 3, 64)
		default:
			parts[i] = fmt.Sprint(t)
		}
	}
	return strings.Join(parts, "|")
}

func runQuery(ctx context.Context, db *sql.DB, query string) (string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	var lines []string
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return "", err
		}
		lines = append(lines, render(vals))
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return strings.Join(lines, "; "), nil
}

func openRO(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// Run executes every check against both files. A query error on either side
// renders as ERROR(...) and counts as a failure — a table the harness cannot
// read is not a pass.
func Run(ctx context.Context, opts Options) (*Report, error) {
	src, err := openRO(opts.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("open source: %w", err)
	}
	defer src.Close()
	dest, err := openRO(opts.DestPath)
	if err != nil {
		return nil, fmt.Errorf("open destination: %w", err)
	}
	defer dest.Close()

	report := &Report{}
	for _, c := range checks {
		srcVal, err := runQuery(ctx, src, c.srcSQL)
		if err != nil {
			srcVal = "ERROR(" + err.Error() + ")"
		}
		destSQL := c.destSQL
		if destSQL == "" {
			destSQL = c.srcSQL
		}
		destVal, err := runQuery(ctx, dest, destSQL)
		if err != nil {
			destVal = "ERROR(" + err.Error() + ")"
		}
		match := srcVal == destVal && !strings.HasPrefix(srcVal, "ERROR(") && !strings.HasPrefix(destVal, "ERROR(")
		report.Checks = append(report.Checks, CheckResult{Name: c.name, Source: srcVal, Dest: destVal, Match: match})
		if match {
			report.Matched++
		} else {
			report.Failed++
		}
	}
	report.Pass = report.Failed == 0
	return report, nil
}
