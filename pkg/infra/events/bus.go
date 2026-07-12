package events

import (
	"context"
	"sync"
)

// Event is the minimum contract shared by all domain events.
type Event interface {
	Name() string
}

// Handler reacts to a published event.
type Handler func(ctx context.Context, event Event) error

// Bus publishes events to subscribed handlers.
type Bus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventName string, handler Handler)
}

// InMemoryBus dispatches events synchronously in subscription order.
type InMemoryBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewInMemoryBus creates an empty in-process event bus.
func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{handlers: make(map[string][]Handler)}
}

// Subscribe registers a handler for an event name.
func (b *InMemoryBus) Subscribe(eventName string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.handlers == nil {
		b.handlers = make(map[string][]Handler)
	}
	b.handlers[eventName] = append(b.handlers[eventName], handler)
}

// Publish dispatches an event to all subscribers for the event name.
func (b *InMemoryBus) Publish(ctx context.Context, event Event) error {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[event.Name()]...)
	b.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
