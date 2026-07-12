//go:build manual

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestApplyCustomerReferenceSeedToLiveDB(t *testing.T) {
	if os.Getenv("CUSTOMER_REFERENCE_SEED_COMMIT") != "1" {
		t.Skip("set CUSTOMER_REFERENCE_SEED_COMMIT=1 to apply the customer reference seed to the live database")
	}

	dbPath := filepath.Join(".", "ph_holdings.db")
	seedPath := filepath.Join("data", "customer_reference_seed.tsv")

	stamp := time.Now().Format("2006_01_02_150405")
	backupDir := filepath.Join("backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}
	backupPath := filepath.Join(backupDir, "ph_holdings_before_customer_cid_reseed_"+stamp+".db")
	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("failed to read live db: %v", err)
	}
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open live database: %v", err)
	}

	result, err := applyCustomerReferenceSeed(db, seedPath)
	if err != nil {
		t.Fatalf("failed to apply customer reference seed: %v", err)
	}

	t.Logf("backup=%s seed_rows=%d matched_existing=%d updated_existing=%d inserted_new=%d skipped_ambiguous=%d unmatched_seed=%d",
		backupPath,
		result.SeedRows,
		result.MatchedExisting,
		result.UpdatedExisting,
		result.InsertedNew,
		result.SkippedAmbiguous,
		result.UnmatchedSeed,
	)
}
