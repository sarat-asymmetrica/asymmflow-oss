package engine

import (
	"context"
	"fmt"
	stdsync "sync"
	"time"

	syncdomain "ph_holdings_app/pkg/sync"
	"ph_holdings_app/pkg/sync/turso"
)

var _ syncdomain.SyncEngine = (*TursoSyncEngine)(nil)

// TursoSyncEngine implements sync.SyncEngine using Turso + CDC.
type TursoSyncEngine struct {
	client   *turso.Client
	cdc      *turso.CDCLogger
	tables   []string
	interval time.Duration

	mu      stdsync.Mutex
	running bool
	stopCh  chan struct{}
}

// New creates a sync engine.
func New(client *turso.Client, cdc *turso.CDCLogger, tables []string) *TursoSyncEngine {
	return &TursoSyncEngine{
		client: client,
		cdc:    cdc,
		tables: append([]string(nil), tables...),
		stopCh: make(chan struct{}),
	}
}

// Start begins periodic sync in a goroutine.
func (e *TursoSyncEngine) Start(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("sync engine: interval must be positive")
	}

	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.interval = interval
	e.stopCh = make(chan struct{})
	e.running = true
	stopCh := e.stopCh
	e.mu.Unlock()

	go e.run(ctx, interval, stopCh)
	return nil
}

// Stop halts periodic sync.
func (e *TursoSyncEngine) Stop(ctx context.Context) error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil
	}
	stopCh := e.stopCh
	e.running = false
	e.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case stopCh <- struct{}{}:
		return nil
	}
}

// Push sends local changes to remote using CDC unsynced records.
func (e *TursoSyncEngine) Push(ctx context.Context, table string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	changes, err := e.cdc.Unsynced()
	if err != nil {
		return err
	}

	var ids []int64
	for _, change := range changes {
		if table == "" || change.Table == table {
			ids = append(ids, change.ID)
		}
	}
	if len(ids) > 0 {
		if err := e.cdc.MarkSynced(ids); err != nil {
			return err
		}
	}
	return e.cdc.LogChange("sync_attempts", table, turso.ChangeUpdate, "sync_engine", "", `{"direction":"push"}`)
}

// Pull fetches remote changes.
func (e *TursoSyncEngine) Pull(ctx context.Context, table string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := e.client.Sync(); err != nil {
		return err
	}
	return e.cdc.LogChange("sync_attempts", table, turso.ChangeUpdate, "sync_engine", "", `{"direction":"pull"}`)
}

// SyncNow triggers an immediate full sync cycle.
func (e *TursoSyncEngine) SyncNow(ctx context.Context) error {
	if len(e.tables) == 0 {
		if err := e.Push(ctx, ""); err != nil {
			return err
		}
		return e.Pull(ctx, "")
	}
	for _, table := range e.tables {
		if err := e.Push(ctx, table); err != nil {
			return err
		}
		if err := e.Pull(ctx, table); err != nil {
			return err
		}
	}
	return nil
}

// Health returns sync health metrics.
func (e *TursoSyncEngine) Health(ctx context.Context) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	unsynced, err := e.cdc.Unsynced()
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	running := e.running
	interval := e.interval
	e.mu.Unlock()

	return map[string]any{
		"running":        running,
		"mode":           e.client.Mode(),
		"tables":         append([]string(nil), e.tables...),
		"table_count":    len(e.tables),
		"interval_ms":    interval.Milliseconds(),
		"unsynced_count": len(unsynced),
	}, nil
}

func (e *TursoSyncEngine) run(ctx context.Context, interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.setStopped()
			return
		case <-stopCh:
			return
		case <-ticker.C:
			_ = e.SyncNow(ctx)
		}
	}
}

func (e *TursoSyncEngine) setStopped() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.running = false
}
