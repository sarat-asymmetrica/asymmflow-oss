package events

import (
	"context"
	"testing"
	"time"
)

func TestPublishDefault_DeliversToSubscriber(t *testing.T) {
	bus := NewInMemoryBus()
	var got Event
	bus.Subscribe((InvoiceCreated{}).Name(), func(ctx context.Context, e Event) error {
		got = e
		return nil
	})
	SetDefault(bus)
	defer SetDefault(nil)

	PublishDefault(context.Background(), InvoiceCreated{InvoiceID: "INV-9"})
	if got == nil {
		t.Fatal("default publish did not reach the subscriber")
	}
	if got.Name() != EventInvoiceCreated {
		t.Errorf("unexpected event name %q", got.Name())
	}
	if Default() != bus {
		t.Error("Default() should return the installed bus")
	}
}

func TestPublishDefault_NilBusIsNoop(t *testing.T) {
	SetDefault(nil)
	// Must not panic and must be a no-op.
	PublishDefault(context.Background(), InvoiceCreated{InvoiceID: "x"})
	if Default() != nil {
		t.Error("Default() should be nil after SetDefault(nil)")
	}
}

func TestInvoiceCreated_ComplianceDataCarrier(t *testing.T) {
	when := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	e := InvoiceCreated{
		InvoiceNumber: "INV-1",
		InvoiceDate:   when,
		SellerTaxID:   "990000000000000",
		BuyerTaxID:    "990000000000999",
		Amount:        100.0,
		TaxAmount:     10.0,
		Currency:      "BHD",
		Jurisdiction:  "BAHRAIN",
	}

	// The compliance hook discovers payloads via this exact interface.
	carrier, ok := any(e).(interface{ ComplianceData() map[string]any })
	if !ok {
		t.Fatal("InvoiceCreated must satisfy the ComplianceData carrier interface")
	}
	data := carrier.ComplianceData()
	if data["invoice_number"] != "INV-1" || data["seller_tax_id"] != "990000000000000" ||
		data["buyer_tax_id"] != "990000000000999" || data["currency"] != "BHD" ||
		data["jurisdiction"] != "BAHRAIN" {
		t.Errorf("unexpected compliance data: %+v", data)
	}
	if data["amount"].(float64) != 100.0 || data["tax_amount"].(float64) != 10.0 {
		t.Errorf("amount/tax not carried: %+v", data)
	}
	if !data["invoice_date"].(time.Time).Equal(when) {
		t.Error("invoice_date not carried")
	}
}
