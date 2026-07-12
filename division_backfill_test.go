package main

import (
	"fmt"
	"strings"
	"testing"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

// TestBackfillDivisionAwareFinanceData_RoutesViaOverlayCase proves the division
// backfill queries — now generated from the active overlay via
// normalizeDivisionSQL instead of 15 hardcoded inline IN-lists — still parse and
// route Beacon/Acme exactly. It exercises BOTH converted query shapes: a
// single-source COALESCE (opportunities from offers) and a multi-source COALESCE
// fallback chain (supplier_invoices from orders, then purchase orders).
func TestBackfillDivisionAwareFinanceData_RoutesViaOverlayCase(t *testing.T) {
	dbName := strings.ReplaceAll(t.Name(), "/", "_")
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	statements := []string{
		`CREATE TABLE offers (id TEXT PRIMARY KEY, division TEXT)`,
		`CREATE TABLE opportunities (id TEXT PRIMARY KEY, offer_id TEXT, division TEXT)`,
		`CREATE TABLE orders (id TEXT PRIMARY KEY, division TEXT)`,
		`CREATE TABLE purchase_orders (id TEXT PRIMARY KEY, order_id TEXT, division TEXT)`,
		`CREATE TABLE supplier_invoices (id TEXT PRIMARY KEY, order_id TEXT, purchase_order_id TEXT, division TEXT)`,

		`INSERT INTO offers (id, division) VALUES ('o-beacon', 'Beacon Controls'), ('o-acme', 'Acme Instrumentation')`,
		`INSERT INTO opportunities (id, offer_id, division) VALUES ('opp-beacon', 'o-beacon', ''), ('opp-acme', 'o-acme', '')`,

		`INSERT INTO orders (id, division) VALUES ('ord-beacon', 'Beacon Controls')`,
		`INSERT INTO purchase_orders (id, order_id, division) VALUES ('po-acme', '', 'Acme Instrumentation')`,
		`INSERT INTO supplier_invoices (id, order_id, purchase_order_id, division) VALUES
			('si-from-order', 'ord-beacon', '', ''),
			('si-from-po', '', 'po-acme', '')`,
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to seed test db: %v", err)
		}
	}

	app := &App{db: db}
	app.backfillDivisionAwareFinanceData()

	assertDivision := func(table, id, want string) {
		t.Helper()
		var got string
		if err := db.Raw(fmt.Sprintf("SELECT division FROM %s WHERE id = ?", table), id).Scan(&got).Error; err != nil {
			t.Fatalf("query %s/%s: %v", table, id, err)
		}
		if got != want {
			t.Errorf("%s/%s division = %q, want %q", table, id, got, want)
		}
	}

	// Single-source COALESCE (opportunities from offers).
	assertDivision("opportunities", "opp-beacon", "Beacon Controls")
	assertDivision("opportunities", "opp-acme", "Acme Instrumentation")

	// Multi-source COALESCE fallback chain (supplier_invoices: order first, then PO).
	assertDivision("supplier_invoices", "si-from-order", "Beacon Controls")
	assertDivision("supplier_invoices", "si-from-po", "Acme Instrumentation")
}
