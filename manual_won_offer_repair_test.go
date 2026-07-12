//go:build manual

package main

import (
	"os"
	"testing"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

func TestManualRepairWonOfferItems(t *testing.T) {
	if os.Getenv("WON_OFFER_REPAIR_COMMIT") != "1" {
		t.Skip("set WON_OFFER_REPAIR_COMMIT=1 to run manual won-offer repair")
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

		results, err := app.BackfillWonOfferItemsFromOpportunityProductDetails()
		if err != nil {
			t.Fatalf("repair %s: %v", dbPath, err)
		}

		var remaining int64
		err = db.Raw(`
			SELECT COUNT(*) FROM (
				SELECT o.id
				FROM offers o
				LEFT JOIN offer_items oi ON oi.offer_id = o.id AND oi.deleted_at IS NULL
				WHERE o.deleted_at IS NULL
				  AND LOWER(COALESCE(o.stage, '')) = 'won'
				GROUP BY o.id
				HAVING COUNT(oi.id) = 0
			)
		`).Scan(&remaining).Error
		if err != nil {
			t.Fatalf("count remaining %s: %v", dbPath, err)
		}

		t.Logf("%s repair_results=%v remaining_won_without_items=%d", dbPath, results, remaining)
	}
}
