package turso

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ChangeType represents the kind of data change.
type ChangeType string

const (
	ChangeInsert ChangeType = "INSERT"
	ChangeUpdate ChangeType = "UPDATE"
	ChangeDelete ChangeType = "DELETE"
)

// ChangeRecord represents one CDC entry.
type ChangeRecord struct {
	ID         int64      `json:"id"`
	Table      string     `json:"table"`
	RecordID   string     `json:"record_id"`
	ChangeType ChangeType `json:"change_type"`
	ChangedAt  time.Time  `json:"changed_at"`
	ChangedBy  string     `json:"changed_by"`
	OldData    string     `json:"old_data,omitempty"`
	NewData    string     `json:"new_data,omitempty"`
	Synced     bool       `json:"synced"`
}

// CDCLogger records data changes for audit and sync tracking.
type CDCLogger struct {
	db *sql.DB
}

// NewCDCLogger creates a CDC logger and ensures the cdc_log table exists.
func NewCDCLogger(db *sql.DB) (*CDCLogger, error) {
	if db == nil {
		return nil, fmt.Errorf("turso cdc: db is nil")
	}
	logger := &CDCLogger{db: db}
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS cdc_log (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	table_name TEXT NOT NULL,
	record_id TEXT NOT NULL,
	change_type TEXT NOT NULL,
	changed_at TEXT NOT NULL,
	changed_by TEXT NOT NULL,
	old_data TEXT,
	new_data TEXT,
	synced INTEGER NOT NULL DEFAULT 0
)`)
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// LogChange records a change.
func (c *CDCLogger) LogChange(table, recordID string, changeType ChangeType, changedBy, oldData, newData string) error {
	_, err := c.db.Exec(`
INSERT INTO cdc_log (table_name, record_id, change_type, changed_at, changed_by, old_data, new_data, synced)
VALUES (?, ?, ?, ?, ?, ?, ?, 0)`,
		table,
		recordID,
		string(changeType),
		time.Now().UTC().Format(time.RFC3339Nano),
		changedBy,
		oldData,
		newData,
	)
	return err
}

// Unsynced returns all change records not yet synced.
func (c *CDCLogger) Unsynced() ([]ChangeRecord, error) {
	rows, err := c.db.Query(`
SELECT id, table_name, record_id, change_type, changed_at, changed_by, old_data, new_data, synced
FROM cdc_log
WHERE synced = 0
ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChanges(rows)
}

// MarkSynced marks records as synced.
func (c *CDCLogger) MarkSynced(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	_, err := c.db.Exec(
		"UPDATE cdc_log SET synced = 1 WHERE id IN ("+strings.Join(placeholders, ",")+")",
		args...,
	)
	return err
}

// Since returns changes after a given timestamp.
func (c *CDCLogger) Since(t time.Time) ([]ChangeRecord, error) {
	rows, err := c.db.Query(`
SELECT id, table_name, record_id, change_type, changed_at, changed_by, old_data, new_data, synced
FROM cdc_log
WHERE changed_at > ?
ORDER BY id`, t.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChanges(rows)
}

// Count returns total CDC entries.
func (c *CDCLogger) Count() (int64, error) {
	var count int64
	if err := c.db.QueryRow("SELECT COUNT(*) FROM cdc_log").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func scanChanges(rows *sql.Rows) ([]ChangeRecord, error) {
	var records []ChangeRecord
	for rows.Next() {
		var record ChangeRecord
		var changedAt string
		var synced int
		if err := rows.Scan(
			&record.ID,
			&record.Table,
			&record.RecordID,
			&record.ChangeType,
			&changedAt,
			&record.ChangedBy,
			&record.OldData,
			&record.NewData,
			&synced,
		); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339Nano, changedAt)
		if err != nil {
			return nil, err
		}
		record.ChangedAt = parsed
		record.Synced = synced != 0
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}
