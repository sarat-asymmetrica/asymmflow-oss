//go:build manual

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestManualImportOneDrive(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ONEDRIVE_IMPORT_ROOT")) == "" {
		t.Skip("set ONEDRIVE_IMPORT_ROOT to scan/import a OneDrive folder")
	}

	rootPath := defaultOneDriveImportRoot()
	dbPath := resolveOneDriveImportDBPath()
	importYear := resolveOneDriveImportYear(rootPath)
	importSource := oneDriveOpportunitySource(importYear)

	commitImport := os.Getenv("ONEDRIVE_IMPORT_COMMIT") == "1"
	matchThreshold := 0.75

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open database %s: %v", dbPath, err)
	}

	app := &App{
		db:                     db,
		cache:                  NewCache(),
		startupImporting:       true,
		startupImportStartTime: time.Now(),
		currentUserID:          "manual-onedrive-import",
		currentUser:            &User{Base: Base{ID: "manual-onedrive-import"}, Username: "manual-onedrive-import"},
	}
	t.Cleanup(app.cache.Stop)

	scan, err := app.ScanOneDrivePaths([]string{rootPath})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	var confirmed []DiscoveredDeal
	var unresolved []DiscoveredDeal
	for _, deal := range scan.Deals {
		if len(deal.CustomerMatches) == 0 || deal.CustomerMatches[0].Score < matchThreshold {
			unresolved = append(unresolved, deal)
			continue
		}
		if deal.CustomerMatches[0].Score < 0.90 {
			t.Logf("low_confidence folder=%q customer=%q score=%.2f reason=%s",
				deal.FolderName,
				deal.CustomerMatches[0].BusinessName,
				deal.CustomerMatches[0].Score,
				deal.CustomerMatches[0].MatchReason,
			)
		}
		deal.ConfirmedCustomerID = deal.CustomerMatches[0].CustomerID
		confirmed = append(confirmed, deal)
	}

	t.Logf("scan_summary deals=%d files=%d confirmed=%d unresolved=%d errors=%d",
		len(scan.Deals), scan.TotalFiles, len(confirmed), len(unresolved), len(scan.Errors))

	for i, deal := range unresolved {
		if i >= 25 {
			t.Logf("unresolved_truncated remaining=%d", len(unresolved)-i)
			break
		}
		best := ""
		if len(deal.CustomerMatches) > 0 {
			best = fmt.Sprintf("%s score=%.2f", deal.CustomerMatches[0].BusinessName, deal.CustomerMatches[0].Score)
		}
		t.Logf("unresolved folder=%q best=%s", deal.FolderName, best)
	}

	if !commitImport {
		return
	}

	results, err := app.ImportOneDriveDeals(confirmed)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
			continue
		}
		t.Logf("import_error local_id=%s msg=%s", result.DealLocalID, result.Message)
	}

	var oppCount int64
	var offerCount int64
	var orderCount int64
	var invoiceCount int64

	db.Model(&Opportunity{}).Where("year = ? AND source = ?", importYear, importSource).Count(&oppCount)
	db.Model(&Offer{}).Where("offer_number LIKE ? OR quotation_date >= ?", "EH-%", time.Date(importYear, 1, 1, 0, 0, 0, 0, time.Local)).Count(&offerCount)
	db.Model(&Order{}).Where("order_number LIKE ?", "IMP-%").Count(&orderCount)
	db.Model(&Invoice{}).Where("invoice_number LIKE ? OR invoice_number LIKE ?", "PH%", "INV-%").Count(&invoiceCount)

	t.Logf("import_summary year=%d source=%s attempted=%d succeeded=%d imported_opportunities=%d imported_orders=%d imported_invoices=%d offers_seen=%d",
		importYear, importSource, len(results), successCount, oppCount, orderCount, invoiceCount, offerCount)
}
