// Package phimport is the one-time PH-format → OSS-format SQLite importer
// (PH convergence Mission C). Only CODE lives in this repository; the data it
// moves lives solely in PH's sovereign deployment and never enters this repo.
//
// Hard-won rules baked in (PH 5545793 — "the promote had never actually
// worked"):
//   - PRAGMA foreign_keys is set on a SINGLE PINNED CONNECTION BEFORE the
//     transaction begins. Inside an open transaction the pragma is a silent
//     no-op and FK enforcement stays on.
//   - Copies use INTERSECTED EXPLICIT COLUMN LISTS, never positional
//     SELECT * — the two schemas have drifted per-table.
//   - PRAGMA foreign_key_check runs before commit; violations abort.
//
// Transforms this importer applies (everything else is a straight copy):
//   - invoices.invoice_hash / credit_notes.credit_note_hash are blanked: the
//     HMAC key derives from the destination install's field-crypto salt, so
//     source hashes can never verify; the OSS startup backfill recomputes
//     blank hashes under the destination salt.
//   - settings rows with is_encrypted=1 are not copied: ciphertext is bound
//     to the source machine + salt file and is undecryptable after transfer.
//     Re-enter those secrets (API keys) in the destination app.
//   - customer_receipts are TRANSFORMED into payments (PC-D7, Commander
//     decision 2026-07-06). PH creates a real payments row for every receipt
//     allocation at apply time, and payments is in the copy set — so the
//     applied portion of each receipt already arrives; copying it again would
//     double-count cash. Only the UNAPPLIED on-account remainder becomes a
//     new invoice-less payment row (receipt number + customer id packed into
//     reference so the money stays traceable). Reversed/soft-deleted receipts
//     are void and fully-applied receipts are already represented; both are
//     counted in the report.
//   - Peeled/derived PH tables are skipped and REPORTED, never silently
//     dropped.
package phimport

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"
)

// tableRenames maps PH table names to their OSS spellings (PH overrode
// TableName(); OSS uses GORM's default plural).
var tableRenames = map[string]string{
	"costing_history":             "costing_histories",
	"costing_line_items":          "costing_line_item_data",
	"discount_recommendations":    "discount_recommendation_records",
	"payment_prediction_accuracy": "payment_prediction_accuracies",
}

// columnTransforms rewrites individual column values during the copy, keyed
// "table.column" → SQL expression over the source row. Each entry must be a
// faithful port of the normalisation PH itself applies when READING the row
// (never an invented mapping), so the imported value is what PH's own code
// would have shown the user. Unknown spellings pass through untouched — the
// destination CHECK constraint stays the last line of defence.
var columnTransforms = map[string]string{
	// Port of PH normalizePurchaseOrderStatus (purchase_order_service.go):
	// legacy spellings ("Completed", "partial", …) normalise to the modern
	// vocabulary; PH maps completed/complete to the terminal "Closed".
	"purchase_orders.status": `CASE lower(replace(replace(replace(trim(COALESCE(status,'')),' ',''),'_',''),'-',''))
		WHEN 'draft' THEN 'Draft'
		WHEN 'pendingapproval' THEN 'Pending Approval'
		WHEN 'approved' THEN 'Approved'
		WHEN 'sent' THEN 'Sent'
		WHEN 'acknowledged' THEN 'Acknowledged'
		WHEN 'partiallyreceived' THEN 'Partially Received'
		WHEN 'partialreceived' THEN 'Partially Received'
		WHEN 'partial' THEN 'Partially Received'
		WHEN 'received' THEN 'Received'
		WHEN 'fullyreceived' THEN 'Received'
		WHEN 'fullreceived' THEN 'Received'
		WHEN 'closed' THEN 'Closed'
		WHEN 'completed' THEN 'Closed'
		WHEN 'complete' THEN 'Closed'
		WHEN 'cancelled' THEN 'Cancelled'
		WHEN 'canceled' THEN 'Cancelled'
		ELSE status END`,
}

// columnRenames maps "table.source_column" → destination column for columns
// whose NAME drifted between the two schemas while the semantics did not.
// Only add a pair when PH's own model READS the source spelling (faithful
// carry), never to relocate data PH itself no longer reads.
var columnRenames = map[string]string{
	// PH CreditNote.VATBHD persists as vat_bhd; the OSS model (no column tag)
	// persists as vatbhd. Without this pair every carried credit note would
	// silently lose its VAT amount (Mission H drift scan, PC-D17).
	"credit_notes.vat_bhd": "vatbhd",
}

// replaceTables are destination tables whose rows at import time are the
// app's own provisioned skeleton (foundation-ensured accounts, seed bundles),
// not operator data — the import replaces them wholesale with the source
// company's rows, because copied child rows (journal lines, expense entries,
// role assignments) reference the SOURCE row ids. This is only sound under
// the tool's contract that the destination is a freshly provisioned file;
// every replacement is counted in the report, never silent.
var replaceTables = map[string]string{
	"roles":                 "provisioned RBAC seed replaced by the company's imported roles",
	"chart_of_accounts":     "foundation-ensured account skeleton replaced by the company's chart (same codes, source row ids win)",
	"expense_categories":    "foundation-seeded categories replaced by the company's rows (source row ids win)",
	"company_bank_accounts": "foundation-seeded bank account fixtures replaced by the company's rows (seed uses fixed ids, source rows win)",
	"assets":                "default-assets seed (placeholder letterheads) replaced by the company's real artwork (unique on name)",
}

// skippedTables are PH tables deliberately not carried, with the reason the
// report shows. Silent truncation is forbidden — every skip is counted.
var skippedTables = map[string]string{
	"customer_receipt_allocations": "represented: each allocation's money is a copied payments row (PH creates the payment at allocation time)",
	"costing_sheet_attachments":    "PENDING DECISION: OSS has no attachment table; attachment paths/blobs would be lost",
	"data_update_manifests":        "peeled: PH data-update distribution mechanism removed in OSS",
	"data_update_operations":       "peeled: PH data-update distribution mechanism removed in OSS",
	"data_update_receipts":         "peeled: PH data-update distribution mechanism removed in OSS",
	"employee_archive_requests":    "peeled: OSS keeps only the generic delete_approval_requests ledger",
	"data_quality_reviews":         "peeled: removed in OSS",
	"ar_aging":                     "derived: OSS recomputes AR aging (ar_aging_buckets) from invoices",
	"license_keys":                 "device-local: license activations must not move between installs",
	"opportunities_ssot":           "import scratch: SSOT staging table, not business data",
	"payments_ssot":                "import scratch: SSOT staging table, not business data",
	"products_costing_ssot":        "import scratch: SSOT staging table, not business data",
	"customers_shadow":             "import scratch: Sprint-4 shadow table, not business data",
	"user_sessions":                "session state: users re-authenticate on the destination",
	"sync_status":                  "sync bookkeeping: destination starts with a clean sync state",
	"sync_records":                 "sync bookkeeping: destination starts with a clean sync state",
	"file_watch_events":            "machine-local watcher state",
	"sqlite_sequence":              "sqlite internal",
	"sqlite_stat1":                 "sqlite internal: ANALYZE statistics, regenerated by the destination",
	// Mission H full-surface adjudications (PC-D16..): every formerly-UNMAPPED
	// source table now has an explicit carry/skip ruling.
	"customers_backup":                    "backup: point-in-time backup copy, not live business data (the live customers table is carried)",
	"customer_contacts_backup":            "backup: point-in-time backup copy, not live business data (the live customer_contacts table is carried)",
	"intelligence_order_enrichment":       "derived: intelligence enrichment scaffolding, recomputable from carried business rows",
	"intelligence_line_item_enrichment":   "derived: intelligence enrichment scaffolding, recomputable from carried business rows",
	"intelligence_opportunity_enrichment": "derived: intelligence enrichment scaffolding, recomputable from carried business rows",
	"collaborative_pending_operations":    "transient: collaboration op-queue state; destination starts with a clean queue",
	"extracted_documents":                 "dead scan artifact (PC-D16): written by the historical OneDrive extraction sweep, read by nothing in PH (referenced only in sync-coverage exclusions); preserved in the archived source snapshot; a live re-scan at cutover supersedes it",
}

// copyOrder is PH's dependency-ordered sync-table list (parents before
// children, PH db_sync_service.go dbSyncTables) plus the non-synced tables
// worth carrying (settings, audit trail, document inboxes, graph memory).
var copyOrder = []string{
	// Access and master data
	"roles", "users", "devices", "device_users",
	"customers", "suppliers", "products",
	"customer_contacts", "supplier_contacts", "customer_name_mappings",
	"supplier_issues", "entity_notes",
	// Commercial pipeline
	"opportunities", "opportunity_comments", "opportunity_edit_conflicts",
	"rfq_data", "rfq_comments",
	"costing_sheet_data", "costing_line_items", "costing_history",
	"offers", "offer_items", "offer_data", "offer_notes", "offer_follow_ups",
	"orders", "order_items", "shipments", "post_sale_notes",
	// Customer finance
	"invoices", "invoice_items", "invoice_sequences",
	"credit_notes", "credit_note_items", "payments",
	// Supplier, procurement, and delivery
	"purchase_orders", "purchase_order_items",
	"supplier_invoices", "supplier_invoice_items", "supplier_payments",
	"goods_received_notes", "grn_items", "serial_numbers",
	"delivery_notes", "delivery_note_items",
	// Operational accounting
	"chart_of_accounts", "account_mappings", "fiscal_periods",
	"journal_entries", "journal_lines", "vat_returns",
	"currency_exchange_rates", "company_bank_accounts", "bank_accounts",
	"bank_statements", "bank_statement_lines", "bank_line_payment_allocations",
	"bank_cash_balances", "bank_expense_entries", "statement_hashes",
	"book_bank_reconciliations", "outstanding_cheques", "deposits_in_transit",
	"cheque_registers", "fx_rates", "fx_revaluations",
	"bank_statement_files", "bank_reconciliation_audit_logs",
	// Expenses and payroll
	"expense_categories", "expense_vendors", "expense_entries",
	"expense_allocations", "recurring_expenses", "expense_attachments",
	"expense_approvals",
	"employee_compensation_profiles", "payroll_periods", "payroll_runs",
	"payroll_run_items", "payroll_components", "payroll_payouts",
	// Inventory visibility
	"warehouses", "inventory_items", "stock_movements", "stock_adjustments",
	// People, work, and collaboration
	"employees", "employee_access_links", "projects", "project_members",
	"task_items", "task_comments", "task_activity",
	"notifications", "notification_receipts", "delete_approval_requests",
	"user_activity_sessions", "user_activity_events", "user_activity_weekly_summaries",
	// Intelligence, reporting, and imports
	"conversations", "chat_messages", "prediction_records",
	"win_probability_predictions", "discount_recommendations",
	"customer_snapshots", "actual_outcomes", "grade_changes",
	"payment_prediction_accuracy",
	"tally_invoice_imports", "tally_purchase_imports",
	"alerts", "followup_tasks", "assets",
	"contract_templates", "contract_clauses", "contracts",
	// Non-synced but worth carrying
	"settings", "audit_logs", "inbox_documents", "quick_captures",
	"ocr_documents", "graph_nodes", "graph_edges", "jobs",
}

// Options configures a run. DestPath must already carry the OSS schema —
// run the OSS app once against a fresh database file to provision it (the
// app's migration gate never re-migrates a mature imported DB, so the schema
// must exist before rows arrive).
type Options struct {
	SourcePath string
	DestPath   string
}

// TableCopy is one table's outcome.
type TableCopy struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
	Rows   int64  `json:"rows"`
	Reason string `json:"reason,omitempty"`
}

// ColumnDrop records a source column that the copy could not land anywhere
// (no destination column, no rename, no transform) and that actually held
// data. PH's own app reads none of these (that is why the OSS schema lacks
// them), so dropping is faithful — but it must never be silent: these counts
// are Mission I's (data-quality wave) work-list.
type ColumnDrop struct {
	Table        string `json:"table"`
	Column       string `json:"column"`
	NonEmptyRows int64  `json:"non_empty_rows"`
}

// Report is the full, honest accounting of a run.
type Report struct {
	Copied                   []TableCopy  `json:"copied"`
	Skipped                  []TableCopy  `json:"skipped"`
	Unmapped                 []TableCopy  `json:"unmapped"`
	ColumnDrops              []ColumnDrop `json:"column_drops,omitempty"`
	InvoiceHashesBlanked     int64        `json:"invoice_hashes_blanked"`
	CreditNoteHashesBlanked  int64        `json:"credit_note_hashes_blanked"`
	EncryptedSettingsSkipped int64        `json:"encrypted_settings_skipped"`
	// PC-D7 receipt transform accounting.
	ReceiptsTransformed      int64   `json:"receipts_transformed"`        // on-account remainders carried as payments
	ReceiptsOnAccountBHD     float64 `json:"receipts_on_account_bhd"`     // total unapplied money carried
	ReceiptsFullyApplied     int64   `json:"receipts_fully_applied"`      // already represented by copied payments rows
	ReceiptsReversedOrVoided int64   `json:"receipts_reversed_or_voided"` // reversed / soft-deleted, carry nothing
}

var identRef = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func quoteIdent(name string) (string, error) {
	if !identRef.MatchString(name) {
		return "", fmt.Errorf("invalid SQL identifier %q", name)
	}
	return `"` + name + `"`, nil
}

func tableColumns(ctx context.Context, conn *sql.Conn, schema, table string) ([]string, error) {
	qSchema, err := quoteIdent(schema)
	if err != nil {
		return nil, err
	}
	qTable, err := quoteIdent(table)
	if err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx, fmt.Sprintf("PRAGMA %s.table_info(%s)", qSchema, qTable))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}

// srcCount reports a source table's row count for the report. Every skip path
// must carry the honest count — a skip entry showing 0 rows for a populated
// table is how a migration lies about what it left behind.
func srcCount(ctx context.Context, tx *sql.Tx, table string) int64 {
	var n int64
	if qt, err := quoteIdent(table); err == nil {
		_ = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM src."+qt).Scan(&n)
	}
	return n
}

func contains(list []string, name string) bool {
	for _, c := range list {
		if strings.EqualFold(c, name) {
			return true
		}
	}
	return false
}

// Run executes the import. It refuses to run against a destination that has
// not been provisioned with the OSS schema.
func Run(ctx context.Context, opts Options) (*Report, error) {
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(opts.DestPath))
	if err != nil {
		return nil, fmt.Errorf("open destination: %w", err)
	}
	defer db.Close()
	// One pinned connection: pragmas apply per-connection, and PRAGMA
	// foreign_keys inside a transaction is a silent no-op (PH 5545793).
	db.SetMaxOpenConns(1)

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("pin destination connection: %w", err)
	}
	defer conn.Close()

	// Destination must already carry the OSS schema.
	destCore, err := tableColumns(ctx, conn, "main", "customers")
	if err != nil || len(destCore) == 0 {
		return nil, fmt.Errorf("destination %s has no OSS schema (run the OSS app once against a fresh database file to provision it, then re-run)", opts.DestPath)
	}

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return nil, fmt.Errorf("disable foreign keys: %w", err)
	}
	defer conn.ExecContext(ctx, "PRAGMA foreign_keys = ON")

	srcDSN := "file:" + filepath.ToSlash(opts.SourcePath) + "?mode=ro"
	if _, err := conn.ExecContext(ctx, "ATTACH DATABASE ? AS src", srcDSN); err != nil {
		return nil, fmt.Errorf("attach source read-only: %w", err)
	}
	defer conn.ExecContext(ctx, "DETACH DATABASE src")

	// Enumerate source tables so nothing can vanish unreported.
	sourceTables := map[string]bool{}
	rows, err := conn.QueryContext(ctx, "SELECT name FROM src.sqlite_master WHERE type = 'table'")
	if err != nil {
		return nil, fmt.Errorf("list source tables: %w", err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		sourceTables[name] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	report := &Report{}
	covered := map[string]bool{}

	for _, table := range copyOrder {
		covered[table] = true
		if !sourceTables[table] {
			continue // absent in this source file; nothing to report
		}
		if reason, skip := skippedTables[table]; skip {
			report.Skipped = append(report.Skipped, TableCopy{Source: table, Rows: srcCount(ctx, tx, table), Reason: reason})
			continue
		}

		dest := table
		if renamed, ok := tableRenames[table]; ok {
			dest = renamed
		}

		sourceCols, err := tableColumns(ctx, conn, "src", table)
		if err != nil {
			return nil, fmt.Errorf("read source columns of %s: %w", table, err)
		}
		destCols, err := tableColumns(ctx, conn, "main", dest)
		if err != nil {
			return nil, fmt.Errorf("read destination columns of %s: %w", dest, err)
		}
		if len(destCols) == 0 {
			report.Skipped = append(report.Skipped, TableCopy{Source: table, Dest: dest, Rows: srcCount(ctx, tx, table), Reason: "no destination table — schema not provisioned or table peeled without a mapping"})
			continue
		}

		// Pair every source column with a destination column: direct match,
		// documented rename, or (recorded) drop. Transforms rewrite the value
		// in transit; renames redirect drifted spellings (credit-note VAT).
		destCols64 := make(map[string]string, len(destCols)) // lower → exact
		for _, c := range destCols {
			destCols64[strings.ToLower(c)] = c
		}
		var insertCols, selectExprs, transformed, renamed []string
		var dropped []string
		for _, c := range sourceCols {
			destCol, direct := destCols64[strings.ToLower(c)]
			if !direct {
				if target, ok := columnRenames[table+"."+strings.ToLower(c)]; ok {
					if exact, has := destCols64[strings.ToLower(target)]; has {
						destCol = exact
						renamed = append(renamed, c+"→"+exact)
					}
				}
			}
			if destCol == "" {
				dropped = append(dropped, c)
				continue
			}
			qd, err := quoteIdent(destCol)
			if err != nil {
				return nil, err
			}
			insertCols = append(insertCols, qd)
			if expr, ok := columnTransforms[table+"."+c]; ok {
				selectExprs = append(selectExprs, expr)
				transformed = append(transformed, c)
				continue
			}
			qc, err := quoteIdent(c)
			if err != nil {
				return nil, err
			}
			selectExprs = append(selectExprs, qc)
		}
		if len(insertCols) == 0 {
			report.Skipped = append(report.Skipped, TableCopy{Source: table, Dest: dest, Rows: srcCount(ctx, tx, table), Reason: "no shared columns"})
			continue
		}
		cols := strings.Join(insertCols, ", ")
		qSrc, _ := quoteIdent(table)
		qDest, _ := quoteIdent(dest)

		// Dropped source columns that actually hold data are reported — they
		// are faithful drops (PH's own app reads none of them) but never
		// silent ones.
		for _, c := range dropped {
			qc, err := quoteIdent(c)
			if err != nil {
				return nil, err
			}
			var nonEmpty int64
			_ = tx.QueryRowContext(ctx, fmt.Sprintf(
				"SELECT COUNT(*) FROM src.%s WHERE %s IS NOT NULL AND TRIM(CAST(%s AS TEXT)) NOT IN ('', '0', '0.0')",
				qSrc, qc, qc)).Scan(&nonEmpty)
			if nonEmpty > 0 {
				report.ColumnDrops = append(report.ColumnDrops, ColumnDrop{Table: table, Column: c, NonEmptyRows: nonEmpty})
			}
		}

		var replaced int64
		if _, ok := replaceTables[table]; ok {
			_ = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM main."+qDest).Scan(&replaced)
			if replaced > 0 {
				if _, err := tx.ExecContext(ctx, "DELETE FROM main."+qDest); err != nil {
					return nil, fmt.Errorf("replace baseline of %s: %w", dest, err)
				}
			}
		}

		where := ""
		if table == "settings" && contains(sourceCols, "is_encrypted") {
			// Ciphertext is bound to the source machine's salt; do not carry it.
			where = " WHERE COALESCE(is_encrypted, 0) = 0"
			var encrypted int64
			_ = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM src."+qSrc+" WHERE COALESCE(is_encrypted, 0) != 0").Scan(&encrypted)
			report.EncryptedSettingsSkipped = encrypted
		}

		result, err := tx.ExecContext(ctx, fmt.Sprintf(
			"INSERT INTO main.%s (%s) SELECT %s FROM src.%s%s", qDest, cols, strings.Join(selectExprs, ", "), qSrc, where))
		if err != nil {
			return nil, fmt.Errorf("copy %s → %s: %w", table, dest, err)
		}
		n, _ := result.RowsAffected()
		entry := TableCopy{Source: table, Dest: dest, Rows: n}
		notes := []string{}
		if replaced > 0 {
			notes = append(notes, fmt.Sprintf("%d provisioned baseline row(s) replaced: %s", replaced, replaceTables[table]))
		}
		if len(transformed) > 0 {
			notes = append(notes, fmt.Sprintf("column(s) normalised in transit (PH's own read-side normalisation): %s", strings.Join(transformed, ", ")))
		}
		if len(renamed) > 0 {
			notes = append(notes, fmt.Sprintf("column(s) carried under the OSS spelling: %s", strings.Join(renamed, ", ")))
		}
		entry.Reason = strings.Join(notes, "; ")
		report.Copied = append(report.Copied, entry)
	}

	// PC-D7: customer_receipts are transformed into payments, not copied.
	covered["customer_receipts"] = true
	if sourceTables["customer_receipts"] {
		if err := transformCustomerReceipts(ctx, tx, report); err != nil {
			return nil, fmt.Errorf("transform customer_receipts: %w", err)
		}
	}

	// Anything in the source we neither copied nor knowingly skipped is
	// reported as unmapped — silent drops are how migrations lie.
	for name := range sourceTables {
		if covered[name] {
			continue
		}
		if reason, skip := skippedTables[name]; skip {
			report.Skipped = append(report.Skipped, TableCopy{Source: name, Rows: srcCount(ctx, tx, name), Reason: reason})
			continue
		}
		report.Unmapped = append(report.Unmapped, TableCopy{Source: name, Rows: srcCount(ctx, tx, name), Reason: "source table not in the copy set — decide carry/skip explicitly"})
	}

	// HMAC hashes verify only under the destination salt; blank them so the
	// startup backfill recomputes them there.
	if destInv, err := tableColumns(ctx, conn, "main", "invoices"); err == nil && contains(destInv, "invoice_hash") {
		if res, err := tx.ExecContext(ctx, `UPDATE main.invoices SET invoice_hash = '' WHERE COALESCE(invoice_hash, '') != ''`); err == nil {
			report.InvoiceHashesBlanked, _ = res.RowsAffected()
		}
	}
	if destCN, err := tableColumns(ctx, conn, "main", "credit_notes"); err == nil && contains(destCN, "credit_note_hash") {
		if res, err := tx.ExecContext(ctx, `UPDATE main.credit_notes SET credit_note_hash = '' WHERE COALESCE(credit_note_hash, '') != ''`); err == nil {
			report.CreditNoteHashesBlanked, _ = res.RowsAffected()
		}
	}

	// Referential integrity gate before anything becomes durable.
	violations, err := foreignKeyViolations(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("foreign_key_check: %w", err)
	}
	if len(violations) > 0 {
		preview := violations
		if len(preview) > 20 {
			preview = preview[:20]
		}
		return nil, fmt.Errorf("import aborted: %d foreign-key violations after copy (first %d: %s)",
			len(violations), len(preview), strings.Join(preview, "; "))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	committed = true

	return report, nil
}

// transformCustomerReceipts carries the money PH's receipt ledger holds that
// the payments copy does NOT already carry (PC-D7). PH's receipt_service
// creates a real payments row for every allocation at apply time, so the
// applied portion of each receipt is already in main.payments after the copy
// loop — inserting it again would double-count cash. What has no payments row
// is the UNAPPLIED on-account remainder; each live receipt with one becomes an
// invoice-less payment: id and idempotency key deterministic on the receipt id
// (safe to re-run against a fresh destination), method normalized into the
// payments check constraint's vocabulary (PH normalizeCustomerReceiptMethod),
// receipt number + customer id packed into reference (payments has no
// customer column — this keeps on-account money traceable to its customer).
// Amounts copy exactly from unapplied_amount_bhd; no arithmetic is performed.
func transformCustomerReceipts(ctx context.Context, tx *sql.Tx, report *Report) error {
	const live = `r.deleted_at IS NULL AND COALESCE(r.status, '') != 'Reversed'`
	const onAccount = live + ` AND COALESCE(r.unapplied_amount_bhd, 0) > 0.0005`

	result, err := tx.ExecContext(ctx, `
		INSERT INTO main.payments (
			id, created_at, updated_at, deleted_at, version, created_by,
			invoice_id, invoice_number, amount_bhd, payment_date,
			payment_method, days_to_payment, idempotency_key,
			reference, division, updated_by
		)
		SELECT
			r.id, r.created_at, r.updated_at, NULL, COALESCE(r.version, 1), r.created_by,
			'', '', r.unapplied_amount_bhd, r.receipt_date,
			CASE TRIM(COALESCE(r.payment_method, ''))
				WHEN 'Cash' THEN 'Cash'
				WHEN 'Cheque' THEN 'Cheque'
				WHEN 'Bank Transfer' THEN 'Bank Transfer'
				WHEN 'Credit Card' THEN 'Credit Card'
				WHEN 'LC' THEN 'LC'
				WHEN 'PDC' THEN 'PDC'
				WHEN 'Wire Transfer' THEN 'Bank Transfer'
				WHEN 'NEFT' THEN 'Bank Transfer'
				WHEN 'Online' THEN 'Bank Transfer'
				WHEN 'Card' THEN 'Credit Card'
				ELSE 'Other'
			END,
			0, 'phimport-receipt-' || r.id,
			substr('ON-ACCOUNT ' || COALESCE(r.receipt_number, '') || ' cust:' || COALESCE(r.customer_id, ''), 1, 100),
			COALESCE(r.division, ''), COALESCE(r.updated_by, '')
		FROM src.customer_receipts r
		WHERE `+onAccount)
	if err != nil {
		return fmt.Errorf("insert on-account payments: %w", err)
	}
	report.ReceiptsTransformed, _ = result.RowsAffected()

	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(r.unapplied_amount_bhd), 0) FROM src.customer_receipts r WHERE `+onAccount).
		Scan(&report.ReceiptsOnAccountBHD); err != nil {
		return fmt.Errorf("sum on-account remainders: %w", err)
	}
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM src.customer_receipts r WHERE `+live+
			` AND COALESCE(r.unapplied_amount_bhd, 0) <= 0.0005`).
		Scan(&report.ReceiptsFullyApplied); err != nil {
		return fmt.Errorf("count fully-applied receipts: %w", err)
	}
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM src.customer_receipts r WHERE NOT (`+live+`)`).
		Scan(&report.ReceiptsReversedOrVoided); err != nil {
		return fmt.Errorf("count reversed/voided receipts: %w", err)
	}

	report.Copied = append(report.Copied, TableCopy{
		Source: "customer_receipts",
		Dest:   "payments",
		Rows:   report.ReceiptsTransformed,
		Reason: fmt.Sprintf(
			"PC-D7 transform: %d on-account remainders carried as invoice-less payments (%.3f BHD); %d fully-applied receipts already represented by copied payments rows; %d reversed/voided carried nothing",
			report.ReceiptsTransformed, report.ReceiptsOnAccountBHD,
			report.ReceiptsFullyApplied, report.ReceiptsReversedOrVoided),
	})
	return nil
}

func foreignKeyViolations(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx, "PRAGMA main.foreign_key_check")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var violations []string
	for rows.Next() {
		var table string
		var rowid sql.NullInt64
		var parent string
		var fkid int
		if err := rows.Scan(&table, &rowid, &parent, &fkid); err != nil {
			return nil, err
		}
		violations = append(violations, fmt.Sprintf("%s(rowid=%v)→%s", table, rowid.Int64, parent))
	}
	return violations, rows.Err()
}
