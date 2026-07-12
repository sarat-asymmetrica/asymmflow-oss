package finance_test

import (
	"context"
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"

	"ph_holdings_app/pkg/finance"
	"ph_holdings_app/pkg/infra/events"
)

// TestInvoiceAfterCreate_PublishesEventWithOverlaySellerTRN proves the Phase 3
// publisher wiring end to end at the storage layer: inserting any Invoice fires
// the AfterCreate hook, which publishes a finance.invoice.created event on the
// default bus, with the seller TRN resolved from the active company overlay by
// division (tying Phase 2's overlay identity to Phase 3's event bus).
func TestInvoiceAfterCreate_PublishesEventWithOverlaySellerTRN(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&finance.Invoice{}, &finance.DBInvoiceItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	bus := events.NewInMemoryBus()
	var captured events.InvoiceCreated
	var received bool
	bus.Subscribe((events.InvoiceCreated{}).Name(), func(ctx context.Context, e events.Event) error {
		if ic, ok := e.(events.InvoiceCreated); ok {
			captured = ic
			received = true
		}
		return nil
	})
	events.SetDefault(bus)
	defer events.SetDefault(nil)

	inv := finance.Invoice{
		InvoiceNumber: "INV-100",
		InvoiceDate:   time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC),
		Division:      "Acme Instrumentation",
		SubtotalBHD:   100.0,
		VATBHD:        10.0,
	}
	if err := db.Create(&inv).Error; err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	if !received {
		t.Fatal("AfterCreate did not publish an InvoiceCreated event")
	}
	if captured.InvoiceID != inv.ID {
		t.Errorf("event InvoiceID = %q, want %q", captured.InvoiceID, inv.ID)
	}
	if captured.InvoiceNumber != "INV-100" {
		t.Errorf("event InvoiceNumber = %q, want INV-100", captured.InvoiceNumber)
	}
	// Seller TRN comes from the overlay built-in default for Acme Instrumentation.
	if captured.SellerTaxID != "990000000000000" {
		t.Errorf("event SellerTaxID = %q, want the Acme overlay TRN 990000000000000", captured.SellerTaxID)
	}
	if captured.Currency != "BHD" {
		t.Errorf("event Currency = %q, want BHD (overlay default)", captured.Currency)
	}
	if captured.Amount != 100.0 || captured.TaxAmount != 10.0 {
		t.Errorf("event amounts = (%v, %v), want (100, 10)", captured.Amount, captured.TaxAmount)
	}
	if captured.CorrelationID == "" {
		t.Error("event must carry a correlation id")
	}
}

// TestInvoiceAfterCreate_BeaconDivisionGetsBeaconTRN proves the per-division
// dispatch: a Beacon Controls invoice publishes Beacon's TRN, not Acme's.
func TestInvoiceAfterCreate_BeaconDivisionGetsBeaconTRN(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&finance.Invoice{}, &finance.DBInvoiceItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	bus := events.NewInMemoryBus()
	var captured events.InvoiceCreated
	bus.Subscribe((events.InvoiceCreated{}).Name(), func(ctx context.Context, e events.Event) error {
		captured, _ = e.(events.InvoiceCreated)
		return nil
	})
	events.SetDefault(bus)
	defer events.SetDefault(nil)

	inv := finance.Invoice{InvoiceNumber: "INV-200", Division: "Beacon Controls", SubtotalBHD: 50}
	if err := db.Create(&inv).Error; err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	if captured.SellerTaxID != "990000000000001" {
		t.Errorf("Beacon invoice SellerTaxID = %q, want Beacon overlay TRN 990000000000001", captured.SellerTaxID)
	}
}
