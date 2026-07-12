// Package audit is the ONE engine-backed audit-recording path (Wave 3 B.2).
//
// History: the app carried two audit systems. The live one (App.logAudit →
// infra.AuditLog) silently DROPPED the resourceID and description arguments
// it was handed — a bug wearing a "simplified schema" comment. The second
// (security_enhancements.go AuditLogger) constructed rich AuditEvent values
// and then discarded them (`_ = AuditEvent{…}`), logging to the structured
// logger only — nothing ever reached the database. Both now converge here:
// one Entry shape, one table, persistence plus whatever logging the caller
// keeps doing itself.
package audit

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/infra"
)

// Entry is one auditable action.
type Entry struct {
	UserID      string // empty = system action
	Action      string // e.g. "CREATE", "payment_recorded"
	Resource    string // e.g. "users", "invoice"
	ResourceID  string // identity of the touched row, when known
	Description string // human-readable context
}

// Recorder persists audit entries to the audit_logs table.
type Recorder struct {
	db *gorm.DB
}

// NewRecorder returns a Recorder writing through the given DB handle.
func NewRecorder(db *gorm.DB) *Recorder {
	return &Recorder{db: db}
}

// Record persists the entry synchronously. Action and Resource are required —
// an audit row that doesn't say what happened to what is noise.
func (r *Recorder) Record(e Entry) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("audit: recorder has no database")
	}
	if strings.TrimSpace(e.Action) == "" || strings.TrimSpace(e.Resource) == "" {
		return fmt.Errorf("audit: entry requires an action and a resource (got action=%q resource=%q)", e.Action, e.Resource)
	}
	row := infra.AuditLog{
		UserID:      e.UserID,
		Action:      e.Action,
		Resource:    e.Resource,
		ResourceID:  e.ResourceID,
		Description: e.Description,
	}
	return r.db.Create(&row).Error
}

// RecordAsync persists the entry on a background goroutine (the historical
// logAudit behavior: the write must survive request cancellation and must
// never block the caller). Errors are reported to onErr when non-nil.
func (r *Recorder) RecordAsync(e Entry, onErr func(error)) {
	go func() {
		if err := r.Record(e); err != nil && onErr != nil {
			onErr(err)
		}
	}()
}
