package events

import (
	"context"
	"sync"
)

// The process-wide default bus lets domain-layer code (e.g. a GORM AfterCreate
// hook in pkg/finance) publish events without threading an *App or a Bus through
// every call site. The application wires it once at startup with SetDefault; in
// tests and headless contexts it stays nil and PublishDefault is a no-op.
var (
	defaultMu  sync.RWMutex
	defaultBus Bus
)

// SetDefault installs the process-wide default bus. Passing nil disables
// default publishing (useful for test isolation).
func SetDefault(bus Bus) {
	defaultMu.Lock()
	defaultBus = bus
	defaultMu.Unlock()
}

// Default returns the current process-wide default bus (may be nil).
func Default() Bus {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultBus
}

// PublishDefault publishes to the default bus if one is installed. It is
// nil-safe and swallows any handler error: publishing a domain event must never
// break the operation that produced it (e.g. an invoice insert must commit even
// if a compliance subscriber errors).
func PublishDefault(ctx context.Context, event Event) {
	defaultMu.RLock()
	bus := defaultBus
	defaultMu.RUnlock()
	if bus == nil {
		return
	}
	_ = bus.Publish(ctx, event)
}
