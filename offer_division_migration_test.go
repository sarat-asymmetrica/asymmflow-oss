package main

import (
	"fmt"
	"strings"
	"testing"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

func TestMigrateOfferDivisionSupportAddsAndBackfillsDivision(t *testing.T) {
	dbName := strings.ReplaceAll(t.Name(), "/", "_")
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	statements := []string{
		`CREATE TABLE offers (
			id TEXT PRIMARY KEY,
			offer_number TEXT,
			customer_name TEXT,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`CREATE TABLE orders (
			id TEXT PRIMARY KEY,
			offer_id TEXT,
			division TEXT,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`CREATE TABLE invoices (
			id TEXT PRIMARY KEY,
			offer_id TEXT,
			division TEXT,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`INSERT INTO offers (id, offer_number, customer_name) VALUES
			('offer-ph', '1-26', 'PH Customer'),
			('offer-ahs', '2-26', 'AHS Customer'),
			('offer-default', '3-26', 'Default Customer')`,
		`INSERT INTO orders (id, offer_id, division) VALUES ('order-1', 'offer-ahs', 'Beacon Controls')`,
		`INSERT INTO invoices (id, offer_id, division) VALUES ('invoice-1', 'offer-ph', 'Acme Instrumentation')`,
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to seed test db: %v", err)
		}
	}

	app := &App{db: db}
	app.migrateOfferDivisionSupport()

	if !app.hasColumn("offers", "division") {
		t.Fatalf("expected offers.division column to be created")
	}

	type resultRow struct {
		ID       string
		Division string
	}
	var rows []resultRow
	if err := db.Raw("SELECT id, division FROM offers ORDER BY id").Scan(&rows).Error; err != nil {
		t.Fatalf("failed to query offers divisions: %v", err)
	}

	got := map[string]string{}
	for _, row := range rows {
		got[row.ID] = row.Division
	}

	if got["offer-ahs"] != "Beacon Controls" {
		t.Fatalf("offer-ahs division = %q, want %q", got["offer-ahs"], "Beacon Controls")
	}
	if got["offer-ph"] != "Acme Instrumentation" {
		t.Fatalf("offer-ph division = %q, want %q", got["offer-ph"], "Acme Instrumentation")
	}
	if got["offer-default"] != "Acme Instrumentation" {
		t.Fatalf("offer-default division = %q, want %q", got["offer-default"], "Acme Instrumentation")
	}
}

func TestEnsureCrossModuleSchemaExtensionsRepairsOfferDivisionForStableSchemas(t *testing.T) {
	dbName := strings.ReplaceAll(t.Name(), "/", "_")
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	statements := []string{
		`CREATE TABLE offers (
			id TEXT PRIMARY KEY,
			offer_number TEXT,
			customer_name TEXT,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`CREATE TABLE orders (
			id TEXT PRIMARY KEY,
			offer_id TEXT,
			division TEXT,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`INSERT INTO offers (id, offer_number, customer_name) VALUES ('offer-ahs', '2-26', 'AHS Customer')`,
		`INSERT INTO orders (id, offer_id, division) VALUES ('order-1', 'offer-ahs', 'Beacon Controls')`,
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to seed test db: %v", err)
		}
	}

	app := &App{db: db}
	app.ensureCrossModuleSchemaExtensions()

	if !app.hasColumn("offers", "division") {
		t.Fatalf("expected offers.division column to be created")
	}

	var division string
	if err := db.Raw("SELECT division FROM offers WHERE id = ?", "offer-ahs").Scan(&division).Error; err != nil {
		t.Fatalf("failed to query offer division: %v", err)
	}
	if division != "Beacon Controls" {
		t.Fatalf("offer-ahs division = %q, want %q", division, "Beacon Controls")
	}
}
