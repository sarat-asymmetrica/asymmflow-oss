# Wave 13: Turso + OpenTelemetry

**Date**: 2026-05-07
**Author**: Claude (Opus 4.6) for Codex (GPT-5.5) autonomous execution
**Depends on**: Wave 12 (Mathematical Framework complete)
**Quality gate**: `go build -tags='' ./...` + `go test ./... -count=1 -timeout 300s` after every ticket

---

## Mission

Settle the sync architecture and observability stack before multi-market expansion. Two pillars:

1. **Turso (libsql)**: Replace manual Supabase sync with embedded SQLite replicas. Reads stay instant (local). Writes replicate to cloud. CDC audit trail for every change. This means AsymmFlow works offline-first and syncs when connected — critical for emerging markets (3G, power outages).

2. **OpenTelemetry**: Add structured observability. Traces for request flows, metrics for system health, three-regime classification for operational intelligence. This gives visibility into what's happening across the ERP without drowning in unstructured logs.

Both are ADDITIVE. Existing SQLite via ncruces/go-sqlite3 + gormlite continues working. Turso wraps it. OTel instruments it.

---

## Environment

```powershell
$env:GOTMPDIR='D:\go-tmp'
$env:GOCACHE='D:\go-cache'
New-Item -ItemType Directory -Force -Path $env:GOTMPDIR,$env:GOCACHE | Out-Null
```

---

## Current State

### Database
- `go.mod` has `github.com/ncruces/go-sqlite3 v0.34.0` + `gormlite v0.24.0`
- GORM is the ORM layer (used by all domain packages)
- `db_manager.go` (949 LOC) has `SyncToRemote`, `SyncFromRemote`, `runCollaborativeSync`
- `db_sync_service.go` (817 LOC) has current sync implementation
- Both files are in root `package main` — NOT yet extracted to `pkg/sync/`

### Sync Package (stub)
- `pkg/sync/domain.go` — has `SyncStatus`, `SyncRecord`, `TallyInvoiceImport`, `TallyPurchaseImport` types
- `pkg/sync/ports.go` — has `SyncEngine`, `CloudStorage`, `CollaborationService` interfaces
- `pkg/sync/turso/doc.go` — empty stub (package declaration only)
- `pkg/sync/engine/doc.go` — empty stub
- `pkg/sync/onedrive/doc.go` — empty stub
- `pkg/sync/collaboration/doc.go` — empty stub
- `pkg/sync/tally/doc.go` — empty stub
- `pkg/sync/etl/doc.go` — empty stub

### Infra Package (stub)
- `pkg/infra/otel/doc.go` — empty stub
- `pkg/infra/health/regime.go` — Wave 12 bridge (SystemDigitalRoot function)
- `pkg/infra/events/bus.go` — event bus with tests (WORKING, don't touch)

### Math Package (complete from Wave 12)
- `pkg/math/vedic/` — Digital Root, Williams batching
- `pkg/math/trident/` — Three-Regime types (`Regime`, `RegimeExploration/Optimization/Stabilization`)

---

## Tickets

### Ticket 1: Add Turso and OTel dependencies

Add to `go.mod`:

```
github.com/tursodatabase/go-libsql v0.0.0-20250413163942-3b27d3bfeb1b
go.opentelemetry.io/otel v1.35.0
go.opentelemetry.io/otel/trace v1.35.0
go.opentelemetry.io/otel/metric v1.35.0
go.opentelemetry.io/otel/sdk v1.35.0
go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.35.0
go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.35.0
```

Run `go mod tidy` to resolve transitive deps.

**IMPORTANT**: If `go-libsql` has CGO requirements for embedded mode, use the HTTP-only client mode instead:
```
github.com/tursodatabase/libsql-client-go v0.0.0-20250414055720-a6ac64d8f6a5
```

The HTTP client does NOT require CGO and works with remote Turso databases. Embedded mode (local SQLite + sync) requires CGO via `github.com/tursodatabase/go-libsql`. Try embedded first. If CGO causes build failures (we eliminated CGO in Wave 7 for good reasons), fall back to HTTP-only and document the tradeoff.

**Gate**: `go build -tags='' ./...` passes with new deps. If embedded Turso fails CGO, switch to HTTP client and note it in the progress doc.

---

### Ticket 2: Turso client package

**Target**: `pkg/sync/turso/client.go`

Create a Turso client wrapper that:
1. Opens a local SQLite database with optional remote sync URL
2. Provides a `*sql.DB` compatible interface
3. Handles auth token management
4. Supports both embedded (local+sync) and HTTP-only (remote-only) modes

```go
package turso

import (
    "database/sql"
    "fmt"
)

// Config holds Turso connection settings.
type Config struct {
    LocalPath  string // Path to local SQLite file (empty = remote-only)
    RemoteURL  string // Turso database URL (empty = local-only)
    AuthToken  string // Turso auth token
    SyncPeriod int    // Seconds between sync intervals (0 = manual)
}

// Client wraps a Turso/libsql connection.
type Client struct {
    db     *sql.DB
    config Config
    mode   string // "embedded", "remote", "local"
}

// New creates a Turso client based on config.
// If both LocalPath and RemoteURL are set: embedded mode (local + sync).
// If only RemoteURL: HTTP remote mode.
// If only LocalPath: plain local SQLite (fallback).
func New(cfg Config) (*Client, error) {
    // Implementation depends on whether go-libsql (embedded) or
    // libsql-client-go (HTTP) was resolved in Ticket 1.
    // See the library docs for the correct Open() call.
}

// DB returns the underlying *sql.DB for use with GORM or raw queries.
func (c *Client) DB() *sql.DB {
    return c.db
}

// Sync triggers a manual sync (embedded mode only).
func (c *Client) Sync() error {
    // Embedded: call the sync function from go-libsql
    // Remote/local: no-op
}

// Mode returns "embedded", "remote", or "local".
func (c *Client) Mode() string {
    return c.mode
}

// Close closes the database connection.
func (c *Client) Close() error {
    return c.db.Close()
}
```

**Test**: `pkg/sync/turso/turso_test.go`:
- `TestNewLocalOnly` — Config with only LocalPath creates a "local" mode client with valid *sql.DB
- `TestLocalDBReadWrite` — INSERT and SELECT on the local DB
- `TestModeDetection` — both paths → "embedded", URL only → "remote", path only → "local"

**Gate**: `go build -tags='' ./...` + `go test ./pkg/sync/turso/ -count=1` pass.

---

### Ticket 3: CDC audit log

**Target**: `pkg/sync/turso/cdc.go`

Create a Change Data Capture layer that logs every data mutation:

```go
package turso

import (
    "database/sql"
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
    OldData    string     `json:"old_data,omitempty"` // JSON snapshot before change
    NewData    string     `json:"new_data,omitempty"` // JSON snapshot after change
    Synced     bool       `json:"synced"`
}

// CDCLogger records data changes for audit and sync tracking.
type CDCLogger struct {
    db *sql.DB
}

// NewCDCLogger creates a CDC logger. It creates the cdc_log table if not exists.
func NewCDCLogger(db *sql.DB) (*CDCLogger, error) {
    // CREATE TABLE IF NOT EXISTS cdc_log (...)
}

// LogChange records a change.
func (c *CDCLogger) LogChange(table, recordID string, changeType ChangeType, changedBy, oldData, newData string) error {}

// Unsynced returns all change records not yet synced.
func (c *CDCLogger) Unsynced() ([]ChangeRecord, error) {}

// MarkSynced marks records as synced.
func (c *CDCLogger) MarkSynced(ids []int64) error {}

// Since returns changes after a given timestamp.
func (c *CDCLogger) Since(t time.Time) ([]ChangeRecord, error) {}

// Count returns total CDC entries.
func (c *CDCLogger) Count() (int64, error) {}
```

**Test**: `pkg/sync/turso/cdc_test.go`:
- `TestCDCLogAndRetrieve` — log a change, retrieve it, verify fields
- `TestCDCUnsynced` — log 3 changes, mark 1 synced, Unsynced returns 2
- `TestCDCSince` — log changes at different times, Since filters correctly
- `TestCDCTableCreation` — NewCDCLogger auto-creates cdc_log table

**Gate**: `go build -tags='' ./...` + `go test ./pkg/sync/turso/ -count=1` pass.

---

### Ticket 4: OpenTelemetry provider setup

**Target**: `pkg/infra/otel/provider.go`

Create the OTel initialization that the app calls at startup:

```go
package otel

import (
    "context"
    "io"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/trace"
    stdouttrace "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
    stdoutmetric "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
)

// Config controls OTel behavior.
type Config struct {
    ServiceName    string // e.g., "asymmflow"
    ServiceVersion string // e.g., "2.0.0"
    TraceOutput    io.Writer // Where traces go (os.Stdout, file, nil=noop)
    MetricOutput   io.Writer // Where metrics go
    Enabled        bool      // Master switch
}

// Provider holds initialized OTel resources.
type Provider struct {
    tracer   trace.Tracer
    meter    metric.Meter
    shutdown func(context.Context) error
    config   Config
}

// New creates and registers an OTel provider.
// If config.Enabled is false, returns a no-op provider.
func New(cfg Config) (*Provider, error) {}

// Tracer returns the configured tracer.
func (p *Provider) Tracer() trace.Tracer { return p.tracer }

// Meter returns the configured meter.
func (p *Provider) Meter() metric.Meter { return p.meter }

// Shutdown gracefully flushes and closes exporters.
func (p *Provider) Shutdown(ctx context.Context) error {}
```

**Key design**: When `Enabled=false`, return a provider that wraps OTel's no-op tracer/meter. This means ALL instrumentation code compiles and runs without branching — it just doesn't export anything. Zero overhead in production until you flip the switch.

**Test**: `pkg/infra/otel/otel_test.go`:
- `TestNewDisabled` — Enabled=false creates provider, Tracer() is non-nil (no-op)
- `TestNewEnabled` — Enabled=true with stdout writers creates working provider
- `TestShutdown` — Shutdown returns nil error
- `TestTracerSpan` — Start and End a span without panic

**Gate**: `go build -tags='' ./...` + `go test ./pkg/infra/otel/ -count=1` pass.

---

### Ticket 5: OTel instrumentation helpers

**Target**: `pkg/infra/otel/instrument.go`

Create domain-aware instrumentation helpers:

```go
package otel

import (
    "context"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// StartDomainSpan begins a span with domain context.
// Usage: ctx, end := otelProvider.StartDomainSpan(ctx, "finance", "CreateInvoice")
//        defer end()
func (p *Provider) StartDomainSpan(ctx context.Context, domain, operation string) (context.Context, func()) {
    ctx, span := p.tracer.Start(ctx, domain+"."+operation,
        trace.WithAttributes(
            attribute.String("domain", domain),
            attribute.String("operation", operation),
        ))
    return ctx, func() { span.End() }
}

// RecordRegime records the current system regime as a metric.
// Uses the three-regime classification from pkg/math/trident.
func (p *Provider) RecordRegime(ctx context.Context, domain string, r1, r2, r3 float64) {
    // Record as gauge metrics:
    // asymmflow.regime.exploration = r1
    // asymmflow.regime.optimization = r2
    // asymmflow.regime.stabilization = r3
    // Tagged with domain
}

// RecordLatency records an operation's duration.
func (p *Provider) RecordLatency(ctx context.Context, domain, operation string, durationMs float64) {}

// RecordCount increments a counter for an operation.
func (p *Provider) RecordCount(ctx context.Context, domain, operation string, count int64) {}
```

**Test**: `pkg/infra/otel/instrument_test.go`:
- `TestStartDomainSpan` — creates and ends a span without error
- `TestRecordRegimeNoOp` — calling RecordRegime with disabled provider doesn't panic
- `TestRecordLatency` — records a latency value
- `TestRecordCount` — increments a counter

**Gate**: `go build -tags='' ./...` + `go test ./pkg/infra/otel/ -count=1` pass.

---

### Ticket 6: Three-regime health monitor

**Target**: `pkg/infra/health/monitor.go`

Extend the existing `pkg/infra/health/` (which has `regime.go` from Wave 12) with a health monitor that classifies system state using three-regime dynamics:

```go
package health

import (
    "sync"
    "time"

    "ph_holdings_app/pkg/math/trident"
)

// SystemHealth represents the current system health assessment.
type SystemHealth struct {
    Regime       trident.Regime `json:"regime"`
    RegimeName   string         `json:"regime_name"`
    Score        float64        `json:"score"`       // 0.0 (critical) to 1.0 (excellent)
    ActiveUsers  int            `json:"active_users"`
    ErrorRate    float64        `json:"error_rate"`   // errors per minute
    AvgLatencyMs float64        `json:"avg_latency_ms"`
    Uptime       time.Duration  `json:"uptime"`
    CheckedAt    time.Time      `json:"checked_at"`
}

// Monitor tracks system health metrics and classifies them into regimes.
type Monitor struct {
    mu           sync.RWMutex
    errorCount   int64
    requestCount int64
    latencySum   float64
    startTime    time.Time
}

// NewMonitor creates a health monitor.
func NewMonitor() *Monitor {
    return &Monitor{startTime: time.Now()}
}

// RecordRequest records a completed request.
func (m *Monitor) RecordRequest(latencyMs float64, isError bool) {}

// Health returns the current health assessment with regime classification.
// Classification:
//   High error rate OR high latency → Exploration (system is unstable, exploring failure modes)
//   Moderate metrics, improving → Optimization (system is tuning)
//   Low error rate AND low latency → Stabilization (system is healthy and stable)
func (m *Monitor) Health() SystemHealth {}

// Reset clears accumulated metrics (for testing or periodic reset).
func (m *Monitor) Reset() {}
```

**Test**: `pkg/infra/health/health_test.go`:
- `TestNewMonitor` — creates without error
- `TestHealthySystemIsStabilization` — 100 low-latency, no-error requests → Stabilization regime
- `TestHighErrorRateIsExploration` — 50% error rate → Exploration regime
- `TestHealthScore` — healthy system score > 0.8, unhealthy < 0.5
- `TestReset` — after Reset, health returns to initial state

**Gate**: `go build -tags='' ./...` + `go test ./pkg/infra/health/ -count=1` pass.

---

### Ticket 7: Sync engine implementation

**Target**: `pkg/sync/engine/sync_engine.go`

Implement the `SyncEngine` interface from `pkg/sync/ports.go` using the Turso client + CDC logger:

```go
package engine

import (
    "context"
    "time"

    "ph_holdings_app/pkg/sync"
    "ph_holdings_app/pkg/sync/turso"
)

// TursoSyncEngine implements sync.SyncEngine using Turso + CDC.
type TursoSyncEngine struct {
    client    *turso.Client
    cdc       *turso.CDCLogger
    tables    []string  // Tables to sync
    interval  time.Duration
    running   bool
    stopCh    chan struct{}
}

// New creates a sync engine.
func New(client *turso.Client, cdc *turso.CDCLogger, tables []string) *TursoSyncEngine {}

// Start begins periodic sync in a goroutine.
func (e *TursoSyncEngine) Start(ctx context.Context, interval time.Duration) error {}

// Stop halts periodic sync.
func (e *TursoSyncEngine) Stop(ctx context.Context) error {}

// Push sends local changes to remote (using CDC unsynced records).
func (e *TursoSyncEngine) Push(ctx context.Context, table string) error {}

// Pull fetches remote changes (embedded mode: Turso handles this; HTTP mode: fetch and apply).
func (e *TursoSyncEngine) Pull(ctx context.Context, table string) error {}

// SyncNow triggers an immediate full sync cycle.
func (e *TursoSyncEngine) SyncNow(ctx context.Context) error {}

// Health returns sync health metrics.
func (e *TursoSyncEngine) Health(ctx context.Context) (map[string]interface{}, error) {}
```

**Important**: This engine is ADDITIVE. The existing `db_sync_service.go` and `db_manager.go` in root package continue working. This engine is the REPLACEMENT that will be wired in once validated. Don't modify root files.

**Test**: `pkg/sync/engine/engine_test.go`:
- `TestNewEngine` — creates with local Turso client
- `TestStartStop` — Start and Stop without error
- `TestHealthReturnsMetrics` — Health returns a map with expected keys
- `TestPushLogsToCSC` — Push records sync attempt in CDC

**Gate**: `go build -tags='' ./...` + `go test ./pkg/sync/engine/ -count=1` pass.

---

### Ticket 8: Progress audit

Write `docs/WAVE13_PROGRESS.md` with:

1. Commit table
2. Package inventory (files, functions, types, tests)
3. Dependency changes to go.mod (what was added)
4. Turso mode achieved (embedded vs HTTP — document which and why)
5. OTel provider status
6. Test counts
7. Any CGO issues encountered and how resolved

---

## Rules

### DO

- Run `go mod tidy` after adding dependencies
- Use `go build -tags='' ./...` + `go test ./... -count=1 -timeout 300s` after every ticket
- Commit after each ticket: `feat(codex): <description>`
- If CGO is required for embedded Turso and causes build failures, fall back to HTTP client mode and document the tradeoff
- Preserve the existing event bus in `pkg/infra/events/` — don't touch it

### DO NOT

- Do NOT modify `db_manager.go`, `db_sync_service.go`, or any root package files
- Do NOT modify any file from Waves 9-12 (schemas, adapters, ViewModels, math)
- Do NOT touch Svelte, Wails bindings, or frontend files
- Do NOT add CGO dependencies if they break the existing build (we eliminated CGO in Wave 7)
- Do NOT remove the existing ncruces/go-sqlite3 dependency — Turso is additive

### STOP CONDITIONS

- If embedded Turso requires CGO and the build fails, STOP Ticket 1. Switch to HTTP-only client. Document the CGO issue. Continue with HTTP mode for remaining tickets.
- If OTel SDK version conflicts with existing deps, STOP and document the conflict.
- If `go test ./... -timeout 300s` fails on existing tests after dep additions, STOP and investigate.

---

## Commit Convention

```
feat(codex): add Turso and OTel dependencies (Wave 13, Ticket 1)
feat(codex): create Turso client package (Wave 13, Ticket 2)
feat(codex): create CDC audit logger (Wave 13, Ticket 3)
feat(codex): create OTel provider (Wave 13, Ticket 4)
feat(codex): create OTel instrumentation helpers (Wave 13, Ticket 5)
feat(codex): create three-regime health monitor (Wave 13, Ticket 6)
feat(codex): create Turso sync engine (Wave 13, Ticket 7)
docs(codex): write wave 13 progress report (Wave 13, Ticket 8)
```

---

Built with Love x Simplicity x Truth x Joy.
Om Lokah Samastah Sukhino Bhavantu
