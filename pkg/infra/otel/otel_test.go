package otel

import (
	"bytes"
	"context"
	"testing"
)

func TestNewDisabled(t *testing.T) {
	provider, err := New(Config{Enabled: false})
	if err != nil {
		t.Fatalf("New disabled: %v", err)
	}
	if provider.Tracer() == nil {
		t.Fatalf("disabled tracer is nil")
	}
	if provider.Meter() == nil {
		t.Fatalf("disabled meter is nil")
	}
}

func TestNewEnabled(t *testing.T) {
	var traces bytes.Buffer
	var metrics bytes.Buffer
	provider, err := New(Config{
		ServiceName:  "asymmflow-test",
		TraceOutput:  &traces,
		MetricOutput: &metrics,
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("New enabled: %v", err)
	}
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown: %v", err)
		}
	})
	if provider.Tracer() == nil || provider.Meter() == nil {
		t.Fatalf("enabled provider returned nil tracer or meter")
	}
}

func TestShutdown(t *testing.T) {
	provider, err := New(Config{Enabled: false})
	if err != nil {
		t.Fatalf("New disabled: %v", err)
	}
	if err := provider.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestTracerSpan(t *testing.T) {
	provider, err := New(Config{Enabled: false})
	if err != nil {
		t.Fatalf("New disabled: %v", err)
	}
	_, span := provider.Tracer().Start(context.Background(), "test-span")
	span.End()
}
