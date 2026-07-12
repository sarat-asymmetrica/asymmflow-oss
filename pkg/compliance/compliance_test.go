package compliance

import (
	"encoding/json"
	"testing"
	"time"
)

type mockEngine struct {
	jurisdiction Jurisdiction
	name         string
}

func (m mockEngine) Jurisdiction() Jurisdiction { return m.jurisdiction }
func (m mockEngine) Name() string               { return m.name }
func (m mockEngine) TaxRates() []TaxRate        { return nil }
func (m mockEngine) CalculateTax(tx TaxableTransaction) (*TaxResult, error) {
	tax := tx.Amount * 0.1
	return &TaxResult{
		BaseAmount:   tx.Amount,
		TaxAmount:    tax,
		TotalAmount:  tx.Amount + tax,
		Jurisdiction: m.jurisdiction,
	}, nil
}
func (m mockEngine) ValidateInvoice(inv InvoiceData) (*ValidationResult, error) {
	return &ValidationResult{Valid: true}, nil
}

func TestRegistryStoresAndRetrievesEngines(t *testing.T) {
	registry := NewRegistry()
	engine := mockEngine{jurisdiction: JurisdictionBahrain, name: "Bahrain VAT"}

	registry.Register(engine)

	got, ok := registry.Get(JurisdictionBahrain)
	if !ok {
		t.Fatal("expected registered engine")
	}
	if got.Name() != "Bahrain VAT" {
		t.Fatalf("engine name = %q", got.Name())
	}
}

func TestTaxableTransactionSerializesCorrectly(t *testing.T) {
	tx := TaxableTransaction{
		Amount:        1000,
		Currency:      "INR",
		Date:          time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC),
		Category:      "goods",
		CustomerType:  "registered",
		SupplierType:  "registered",
		HSNCode:       "8481",
		PlaceOfSupply: "KA",
	}

	payload, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("marshal transaction: %v", err)
	}
	var decoded TaxableTransaction
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal transaction: %v", err)
	}
	if decoded.HSNCode != "8481" || decoded.PlaceOfSupply != "KA" {
		t.Fatalf("decoded transaction mismatch: %+v", decoded)
	}
}

func TestTaxResultComputesTotalCorrectly(t *testing.T) {
	engine := mockEngine{jurisdiction: JurisdictionBahrain, name: "Bahrain VAT"}

	result, err := engine.CalculateTax(TaxableTransaction{Amount: 1000})
	if err != nil {
		t.Fatalf("calculate tax: %v", err)
	}
	if result.TaxAmount != 100 || result.TotalAmount != 1100 {
		t.Fatalf("tax result = %+v, want tax 100 total 1100", result)
	}
}
