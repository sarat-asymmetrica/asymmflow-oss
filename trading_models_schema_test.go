package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/pkg/runtime/composition"
)

var updateSchemaGolden = flag.Bool("update-schema-golden", false, "regenerate testdata/trading_schema.golden from tradingModels()")

// normalizeConstraintOrder sorts the trailing CONSTRAINT clauses of a CREATE
// TABLE statement: GORM emits them from a map, so their order varies run to
// run while the schema is semantically identical. Column order stays as-is.
func normalizeConstraintOrder(sql string) string {
	idx := strings.Index(sql, ",CONSTRAINT `")
	if idx < 0 || !strings.HasSuffix(sql, ")") {
		return sql
	}
	head := sql[:idx]
	tail := strings.TrimSuffix(sql[idx+1:], ")") // "CONSTRAINT `a` …,CONSTRAINT `b` …"
	parts := strings.Split(tail, ",CONSTRAINT `")
	constraints := []string{parts[0]}
	for _, p := range parts[1:] {
		constraints = append(constraints, "CONSTRAINT `"+p)
	}
	sort.Strings(constraints)
	return head + "," + strings.Join(constraints, ",") + ")"
}

// TestTradingModels_SchemaGolden pins the exact SQLite schema the trading
// model-set produces (Wave 3 A.2: the AutoMigrate list moved from startup()
// into tradingModels() + composition.MigrateModels — this test is the
// byte-identical guarantee, and from now on any model reshape must arrive
// with a deliberate golden regeneration:
//
//	go test -run TestTradingModels_SchemaGolden -update-schema-golden .
func TestTradingModels_SchemaGolden(t *testing.T) {
	root := composition.NewRoot()
	dsn := composition.SQLiteDSN(filepath.Join(t.TempDir(), "schema.db"), "busy_timeout(5000)", "journal_mode(WAL)")
	db, err := root.OpenSQLite(dsn, &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true, // same as startup()
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})

	migrated, skipped := root.MigrateModels(tradingModels(), func(i, n int, name string, err error) {
		if err != nil {
			t.Errorf("model %s failed to migrate on a FRESH database: %v", name, err)
		}
	})
	if skipped != 0 {
		t.Fatalf("fresh-database migration skipped %d models (migrated %d) — the registered set must migrate cleanly", skipped, migrated)
	}

	type row struct {
		Type string
		Name string
		SQL  string
	}
	var rows []row
	if err := db.Raw(`SELECT type, name, COALESCE(sql, '') AS sql FROM sqlite_master
		WHERE name NOT LIKE 'sqlite_%' ORDER BY type, name`).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "-- %s %s\n%s\n\n", r.Type, r.Name, normalizeConstraintOrder(strings.ReplaceAll(r.SQL, "\r\n", "\n")))
	}
	got := b.String()

	goldenPath := filepath.Join("testdata", "trading_schema.golden")
	if *updateSchemaGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("golden regenerated: %s (%d schema objects)", goldenPath, len(rows))
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (regenerate with -update-schema-golden): %v", err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("trading schema drifted from golden.\nIf the change is intentional, regenerate with:\n  go test -run TestTradingModels_SchemaGolden -update-schema-golden .\nand review the golden diff in the same commit.")
	}
}
