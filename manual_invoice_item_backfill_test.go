//go:build manual

package main

import (
	"os"
	"testing"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

func TestManualBackfillInvoiceItems(t *testing.T) {
	if os.Getenv("INVOICE_ITEM_BACKFILL_COMMIT") != "1" {
		t.Skip("set INVOICE_ITEM_BACKFILL_COMMIT=1 to run manual invoice-item backfill")
	}

	dbPaths := []string{
		"ph_holdings.db",
		"/Users/developer/.local/share/AsymmFlow/ph_holdings.db",
	}

	for _, dbPath := range dbPaths {
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("open %s: %v", dbPath, err)
		}

		app := NewApp()
		app.db = db

		results, err := app.BackfillInvoiceItemsFromOrders()
		if err != nil {
			t.Fatalf("backfill %s: %v", dbPath, err)
		}

		var remaining int64
		err = db.Raw(`
			SELECT COUNT(*) FROM (
				SELECT i.id
				FROM invoices i
				LEFT JOIN invoice_items ii ON ii.invoice_id = i.id AND ii.deleted_at IS NULL
				WHERE i.deleted_at IS NULL
				  AND COALESCE(i.order_id, '') != ''
				GROUP BY i.id
				HAVING COUNT(ii.id) = 0
			)
		`).Scan(&remaining).Error
		if err != nil {
			t.Fatalf("count remaining %s: %v", dbPath, err)
		}

		t.Logf("%s backfill_results=%v remaining_without_items=%d", dbPath, results, remaining)
	}
}
