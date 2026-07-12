package engine

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"ph_holdings_app/pkg/sync/turso"
)

func TestNewEngine(t *testing.T) {
	engine := newTestEngine(t)
	if engine == nil {
		t.Fatalf("New returned nil")
	}
}

func TestStartStop(t *testing.T) {
	engine := newTestEngine(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := engine.Start(ctx, 10*time.Millisecond); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := engine.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestHealthReturnsMetrics(t *testing.T) {
	engine := newTestEngine(t)
	health, err := engine.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}

	for _, key := range []string{"running", "mode", "tables", "table_count", "interval_ms", "unsynced_count"} {
		if _, ok := health[key]; !ok {
			t.Fatalf("Health missing key %q: %+v", key, health)
		}
	}
}

func TestPushLogsToCDC(t *testing.T) {
	engine := newTestEngine(t)
	if err := engine.cdc.LogChange("customers", "C-001", turso.ChangeInsert, "tester", "", "{}"); err != nil {
		t.Fatalf("LogChange: %v", err)
	}
	if err := engine.Push(context.Background(), "customers"); err != nil {
		t.Fatalf("Push: %v", err)
	}

	count, err := engine.cdc.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 2 {
		t.Fatalf("CDC count = %d, want original change plus sync attempt", count)
	}
}

func newTestEngine(t *testing.T) *TursoSyncEngine {
	t.Helper()

	client, err := turso.New(turso.Config{LocalPath: filepath.Join(t.TempDir(), "sync.db")})
	if err != nil {
		t.Fatalf("turso.New: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("client.Close: %v", err)
		}
	})
	cdc, err := turso.NewCDCLogger(client.DB())
	if err != nil {
		t.Fatalf("NewCDCLogger: %v", err)
	}
	return New(client, cdc, []string{"customers", "orders"})
}
