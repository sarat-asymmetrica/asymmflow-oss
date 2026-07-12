package main

// B5 checkpoint stage. Folds any pending WAL frames back into the main
// database file and truncates the WAL, so the on-disk .db (the file that
// gets backed up / copied into deploy packages) actually reflects the
// committed B5 repair. This performs no logical data change — it is a
// storage-engine housekeeping step, run last in the runbook.

import (
	"os"
	"testing"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestB2StockCheckpoint runs PRAGMA wal_checkpoint(TRUNCATE) against the
// target DB(s) and force-closes the handle. Run with:
//
//	B2_STOCK_CHECKPOINT=1 B2_STOCK_DB_PATH="ph_holdings.db" go test ./... -run TestB2StockCheckpoint -v
//
// Target DBs default to the project-root working copy and the live AppData
// copy; override with a ';'-separated B2_STOCK_DB_PATH.
func TestB2StockCheckpoint(t *testing.T) {
	if os.Getenv("B2_STOCK_CHECKPOINT") != "1" {
		t.Skip("set B2_STOCK_CHECKPOINT=1 to checkpoint the WAL into the main db file")
	}

	for _, dbPath := range b2StockCandidateDBPaths() {
		if dbPath == "" {
			continue
		}
		if b2StockIsPostgres(dbPath) {
			// WAL checkpointing is a SQLite-file concept; a PostgreSQL
			// target (the cloud-sync layer) has server-side durability and
			// no on-disk file to fold. Skipping is correct, not a failure.
			t.Logf("---- %s : PostgreSQL target — WAL checkpoint not applicable, skipping ----", b2StockTargetLabel(dbPath))
			continue
		}
		if _, err := os.Stat(dbPath); err != nil {
			t.Logf("---- %s : NOT PRESENT, skipping (%v) ----", dbPath, err)
			continue
		}

		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatalf("open %s: %v", dbPath, err)
		}

		type ckpt struct {
			Busy      int
			Log       int
			Checkpkts int
		}
		var r ckpt
		if err := db.Raw(`PRAGMA wal_checkpoint(TRUNCATE)`).Scan(&r).Error; err != nil {
			t.Fatalf("wal_checkpoint %s: %v", dbPath, err)
		}
		t.Logf("%s: wal_checkpoint(TRUNCATE) busy=%d log=%d checkpointed=%d (busy=0 & log=0 => fully merged)",
			dbPath, r.Busy, r.Log, r.Checkpkts)

		if err := closeGormDB(db); err != nil {
			t.Logf("warning: close %s: %v", dbPath, err)
		}

		if r.Busy != 0 {
			t.Errorf("%s: checkpoint reported busy=%d (another connection held the db); WAL not fully merged", dbPath, r.Busy)
		}
	}
}
