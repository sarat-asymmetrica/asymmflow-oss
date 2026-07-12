package otel

import (
	"context"
	"testing"
)

func TestStartDomainSpan(t *testing.T) {
	provider, err := New(Config{Enabled: false})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, end := provider.StartDomainSpan(context.Background(), "finance", "CreateInvoice")
	end()
}

func TestRecordRegimeNoOp(t *testing.T) {
	provider, err := New(Config{Enabled: false})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	provider.RecordRegime(context.Background(), "sync", 0.3, 0.2, 0.5)
}

func TestRecordLatency(t *testing.T) {
	provider, err := New(Config{Enabled: false})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	provider.RecordLatency(context.Background(), "crm", "LoadCustomer", 12.5)
}

func TestRecordCount(t *testing.T) {
	provider, err := New(Config{Enabled: false})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	provider.RecordCount(context.Background(), "butler", "Chat", 1)
}
