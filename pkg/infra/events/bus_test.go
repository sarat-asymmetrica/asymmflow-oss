package events

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestInMemoryBusPublishesToSubscribersInOrder(t *testing.T) {
	bus := NewInMemoryBus()
	var seen []string

	bus.Subscribe((InvoiceCreated{}).Name(), func(ctx context.Context, event Event) error {
		seen = append(seen, "first:"+event.Name())
		return nil
	})
	bus.Subscribe((InvoiceCreated{}).Name(), func(ctx context.Context, event Event) error {
		seen = append(seen, "second:"+event.Name())
		return nil
	})

	err := bus.Publish(context.Background(), InvoiceCreated{InvoiceID: "INV-1"})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	want := []string{"first:finance.invoice.created", "second:finance.invoice.created"}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("handlers ran in unexpected order: got %v want %v", seen, want)
	}
}

func TestInMemoryBusStopsOnHandlerError(t *testing.T) {
	bus := NewInMemoryBus()
	wantErr := errors.New("stop")
	calledAfterError := false

	bus.Subscribe((PaymentRecorded{}).Name(), func(ctx context.Context, event Event) error {
		return wantErr
	})
	bus.Subscribe((PaymentRecorded{}).Name(), func(ctx context.Context, event Event) error {
		calledAfterError = true
		return nil
	})

	err := bus.Publish(context.Background(), PaymentRecorded{PaymentID: "PAY-1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Publish error = %v, want %v", err, wantErr)
	}
	if calledAfterError {
		t.Fatal("handler after error should not be called")
	}
}
