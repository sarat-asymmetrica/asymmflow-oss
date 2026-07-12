package bahrain

import (
	"testing"
	"time"

	"ph_holdings_app/pkg/compliance"
)

func TestStandardGoodsVAT(t *testing.T) {
	engine := New()

	result, err := engine.CalculateTax(compliance.TaxableTransaction{
		Amount:   1000,
		Currency: "BHD",
		Category: "goods",
	})
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if result.TaxAmount != 100 || result.TotalAmount != 1100 {
		t.Fatalf("result = %+v, want 100 VAT and 1100 total", result)
	}
}

func TestExemptCategoryVAT(t *testing.T) {
	engine := New()

	result, err := engine.CalculateTax(compliance.TaxableTransaction{
		Amount:   1000,
		Currency: "BHD",
		Category: "healthcare",
	})
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if result.TaxAmount != 0 || result.TaxBreakdown[0].Rate != 0 {
		t.Fatalf("result = %+v, want exempt zero VAT", result)
	}
}

func TestZeroRatedExportVAT(t *testing.T) {
	engine := New()

	result, err := engine.CalculateTax(compliance.TaxableTransaction{
		Amount:       1000,
		Currency:     "BHD",
		Category:     "goods",
		CustomerType: "export",
	})
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if result.TaxAmount != 0 || result.TaxBreakdown[0].Rate != 0 {
		t.Fatalf("result = %+v, want zero-rated export", result)
	}
}

func TestValidInvoicePasses(t *testing.T) {
	engine := New()

	result, err := engine.ValidateInvoice(compliance.InvoiceData{
		InvoiceNumber: "INV-1",
		InvoiceDate:   time.Now(),
		SellerTaxID:   "BH-VAT-12345678",
		BuyerTaxID:    "BH-VAT-87654321",
		Amount:        1000,
		TaxAmount:     100,
		Currency:      "BHD",
		LineItems: []compliance.LineItemData{
			{Description: "Valve", Quantity: 1, UnitPrice: 1000, TaxRate: 0.10},
		},
	})
	if err != nil {
		t.Fatalf("ValidateInvoice: %v", err)
	}
	if !result.Valid {
		t.Fatalf("valid invoice failed: %+v", result)
	}
}

func TestMissingTaxIDFails(t *testing.T) {
	engine := New()

	result, err := engine.ValidateInvoice(compliance.InvoiceData{
		InvoiceNumber: "INV-1",
		InvoiceDate:   time.Now(),
		Amount:        1000,
		TaxAmount:     100,
		Currency:      "BHD",
	})
	if err != nil {
		t.Fatalf("ValidateInvoice: %v", err)
	}
	if result.Valid {
		t.Fatalf("invoice without seller tax ID should fail: %+v", result)
	}
}
