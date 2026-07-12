package sync

import (
	"testing"
	"time"

	gormsync "ph_holdings_app/pkg/sync"
)

func TestSyncRecordRoundtrip(t *testing.T) {
	now := time.Date(2026, 5, 6, 7, 40, 0, 0, time.UTC)
	record := gormsync.SyncRecord{
		Base: gormsync.Base{
			ID:        "sync-1",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   3,
			CreatedBy: "codex",
		},
		SyncTable:     "invoices",
		RecordID:      "inv-1",
		SyncedAt:      now,
		Direction:     "pull",
		RemoteVersion: 4,
		LocalVersion:  3,
		ConflictState: "remote_wins",
	}

	proto, err := SyncRecordToProto(record)
	if err != nil {
		t.Fatalf("SyncRecordToProto: %v", err)
	}
	got, err := SyncRecordFromProto(*proto)
	if err != nil {
		t.Fatalf("SyncRecordFromProto: %v", err)
	}

	if got.ID != record.ID || got.SyncTable != record.SyncTable || got.RecordID != record.RecordID {
		t.Fatalf("identity mismatch: got %#v want %#v", got, record)
	}
	if got.Direction != record.Direction || got.ConflictState != record.ConflictState {
		t.Fatalf("enum mismatch: got direction=%q conflict=%q", got.Direction, got.ConflictState)
	}
	if !got.SyncedAt.Equal(record.SyncedAt) {
		t.Fatalf("synced_at mismatch: got %s want %s", got.SyncedAt, record.SyncedAt)
	}
}

func TestTallyInvoiceImportRoundtrip(t *testing.T) {
	now := time.Date(2026, 5, 6, 7, 45, 0, 0, time.UTC)
	row := gormsync.TallyInvoiceImport{
		Base: gormsync.Base{
			ID:        "tally-inv-1",
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "codex",
		},
		ImportBatch:       "batch-1",
		Year:              2026,
		InvoiceNumber:     "INV-001",
		CustomerName:      "Asymmetrica",
		MatchedCustomerID: "cust-1",
		InvoiceDate:       now,
		Amount:            120.5,
		Currency:          "BHD",
		Status:            "matched",
		RawData:           `{"source":"tally"}`,
	}

	proto, err := TallyInvoiceImportToProto(row)
	if err != nil {
		t.Fatalf("TallyInvoiceImportToProto: %v", err)
	}
	got, err := TallyInvoiceImportFromProto(*proto)
	if err != nil {
		t.Fatalf("TallyInvoiceImportFromProto: %v", err)
	}

	if got.ID != row.ID || got.InvoiceNumber != row.InvoiceNumber || got.MatchedCustomerID != row.MatchedCustomerID {
		t.Fatalf("identity mismatch: got %#v want %#v", got, row)
	}
	if got.Currency != row.Currency || got.Status != row.Status || got.Amount != row.Amount {
		t.Fatalf("value mismatch: got currency=%q status=%q amount=%f", got.Currency, got.Status, got.Amount)
	}
}
