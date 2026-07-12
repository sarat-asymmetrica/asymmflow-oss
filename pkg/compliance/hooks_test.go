package compliance

import (
	"context"
	"testing"
	"time"

	"ph_holdings_app/pkg/infra/events"
)

type hookTestEngine struct{}

func (hookTestEngine) Jurisdiction() Jurisdiction { return JurisdictionBahrain }
func (hookTestEngine) Name() string               { return "Hook Test VAT" }
func (hookTestEngine) TaxRates() []TaxRate        { return nil }
func (hookTestEngine) CalculateTax(tx TaxableTransaction) (*TaxResult, error) {
	return &TaxResult{BaseAmount: tx.Amount, TotalAmount: tx.Amount, Jurisdiction: JurisdictionBahrain}, nil
}
func (hookTestEngine) ValidateInvoice(inv InvoiceData) (*ValidationResult, error) {
	if inv.SellerTaxID == "" {
		return &ValidationResult{Valid: false, Errors: []string{"seller tax ID required"}}, nil
	}
	return &ValidationResult{Valid: true}, nil
}

type invoiceComplianceEvent struct {
	data map[string]any
}

func (e invoiceComplianceEvent) Name() string { return events.EventInvoiceCreated }
func (e invoiceComplianceEvent) ComplianceData() map[string]any {
	return e.data
}

func TestInvoiceCreatedEventTriggersBahrainValidation(t *testing.T) {
	registry := NewRegistry()
	registry.Register(hookTestEngine{})
	bus := events.NewInMemoryBus()
	hook := NewComplianceHook(registry, bus)

	err := bus.Publish(context.Background(), invoiceComplianceEvent{data: map[string]any{
		"jurisdiction":   "BH",
		"invoice_number": "INV-1",
		"invoice_date":   time.Now(),
		"seller_tax_id":  "BH-VAT-123",
		"buyer_tax_id":   "BH-VAT-456",
		"amount":         1000.0,
		"tax_amount":     100.0,
		"currency":       "BHD",
	}})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	waitForValidation(t, hook, 1)
	entries := hook.RecentValidations(1)
	if len(entries) != 1 || !entries[0].Valid {
		t.Fatalf("validation entry = %+v", entries)
	}
}

func TestInvalidInvoiceValidationErrorsLogged(t *testing.T) {
	registry := NewRegistry()
	registry.Register(hookTestEngine{})
	hook := NewComplianceHook(registry, nil)

	if err := hook.OnInvoiceCreated(map[string]any{
		"jurisdiction":   "BH",
		"invoice_number": "INV-2",
		"amount":         1000.0,
		"tax_amount":     100.0,
		"currency":       "BHD",
	}); err != nil {
		t.Fatalf("OnInvoiceCreated: %v", err)
	}

	entries := hook.RecentValidations(1)
	if len(entries) != 1 || entries[0].Valid || len(entries[0].Errors) == 0 {
		t.Fatalf("expected invalid validation entry, got %+v", entries)
	}
}

func TestUnknownJurisdictionGracefullySkipped(t *testing.T) {
	hook := NewComplianceHook(NewRegistry(), nil)

	if err := hook.OnInvoiceCreated(map[string]any{
		"jurisdiction":   "GH",
		"invoice_number": "INV-3",
		"currency":       "GHS",
	}); err != nil {
		t.Fatalf("OnInvoiceCreated: %v", err)
	}

	entries := hook.RecentValidations(1)
	if len(entries) != 1 || !entries[0].Valid || len(entries[0].Warnings) == 0 {
		t.Fatalf("expected graceful skip warning, got %+v", entries)
	}
}

func waitForValidation(t *testing.T, hook *ComplianceHook, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(hook.RecentValidations(count)) >= count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d validation entries", count)
}
