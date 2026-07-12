package turso

import (
	"testing"
	"time"
)

func TestCDCLogAndRetrieve(t *testing.T) {
	logger := newCDCLogger(t)

	if err := logger.LogChange("customers", "C-001", ChangeInsert, "tester", "", `{"name":"Acme"}`); err != nil {
		t.Fatalf("LogChange: %v", err)
	}

	changes, err := logger.Unsynced()
	if err != nil {
		t.Fatalf("Unsynced: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("changes len = %d, want 1", len(changes))
	}
	got := changes[0]
	if got.Table != "customers" || got.RecordID != "C-001" || got.ChangeType != ChangeInsert || got.ChangedBy != "tester" {
		t.Fatalf("unexpected change: %+v", got)
	}
	if got.Synced {
		t.Fatalf("new change should be unsynced")
	}
}

func TestCDCUnsynced(t *testing.T) {
	logger := newCDCLogger(t)
	for _, id := range []string{"1", "2", "3"} {
		if err := logger.LogChange("orders", id, ChangeUpdate, "tester", `{}`, `{}`); err != nil {
			t.Fatalf("LogChange: %v", err)
		}
	}

	changes, err := logger.Unsynced()
	if err != nil {
		t.Fatalf("Unsynced: %v", err)
	}
	if err := logger.MarkSynced([]int64{changes[0].ID}); err != nil {
		t.Fatalf("MarkSynced: %v", err)
	}

	changes, err = logger.Unsynced()
	if err != nil {
		t.Fatalf("Unsynced after mark: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("unsynced len = %d, want 2", len(changes))
	}
}

func TestCDCSince(t *testing.T) {
	logger := newCDCLogger(t)
	oldTime := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339Nano)
	if _, err := logger.db.Exec(`
INSERT INTO cdc_log (table_name, record_id, change_type, changed_at, changed_by, old_data, new_data, synced)
VALUES (?, ?, ?, ?, ?, ?, ?, 0)`, "items", "old", string(ChangeInsert), oldTime, "tester", "", "{}"); err != nil {
		t.Fatalf("insert old row: %v", err)
	}
	cutoff := time.Now().UTC().Add(-1 * time.Hour)
	if err := logger.LogChange("items", "new", ChangeUpdate, "tester", "{}", `{"ok":true}`); err != nil {
		t.Fatalf("LogChange: %v", err)
	}

	changes, err := logger.Since(cutoff)
	if err != nil {
		t.Fatalf("Since: %v", err)
	}
	if len(changes) != 1 || changes[0].RecordID != "new" {
		t.Fatalf("Since returned %+v, want only new change", changes)
	}
}

func TestCDCTableCreation(t *testing.T) {
	logger := newCDCLogger(t)

	var name string
	err := logger.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='cdc_log'").Scan(&name)
	if err != nil {
		t.Fatalf("cdc_log table not found: %v", err)
	}
	if name != "cdc_log" {
		t.Fatalf("table name = %q, want cdc_log", name)
	}
}

func newCDCLogger(t *testing.T) *CDCLogger {
	t.Helper()

	client := newLocalClient(t)
	logger, err := NewCDCLogger(client.DB())
	if err != nil {
		t.Fatalf("NewCDCLogger: %v", err)
	}
	return logger
}
