package india

import (
	"testing"
	"time"

	"ph_holdings_app/pkg/compliance"
)

func TestGSTIntraStateSplitsCGSTAndSGST(t *testing.T) {
	engine := NewGST()

	result, err := engine.CalculateTax(compliance.TaxableTransaction{
		Amount:        1000,
		Currency:      "INR",
		Category:      "goods",
		HSNCode:       "8481",
		PlaceOfSupply: "KA:KA",
	})
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if result.TaxAmount != 180 || len(result.TaxBreakdown) != 2 {
		t.Fatalf("result = %+v, want split GST of 180", result)
	}
	if result.TaxBreakdown[0].Name != "CGST" || result.TaxBreakdown[0].Amount != 90 {
		t.Fatalf("CGST = %+v, want 90", result.TaxBreakdown[0])
	}
	if result.TaxBreakdown[1].Name != "SGST" || result.TaxBreakdown[1].Amount != 90 {
		t.Fatalf("SGST = %+v, want 90", result.TaxBreakdown[1])
	}
}

func TestGSTInterStateUsesIGST(t *testing.T) {
	engine := NewGST()

	result, err := engine.CalculateTax(compliance.TaxableTransaction{
		Amount:        1000,
		Currency:      "INR",
		Category:      "goods",
		HSNCode:       "8481",
		PlaceOfSupply: "KA:MH",
	})
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if result.TaxAmount != 180 || len(result.TaxBreakdown) != 1 {
		t.Fatalf("result = %+v, want IGST of 180", result)
	}
	if result.TaxBreakdown[0].Name != "IGST" || result.TaxBreakdown[0].Amount != 180 {
		t.Fatalf("IGST = %+v, want 180", result.TaxBreakdown[0])
	}
}

func TestGSTZeroRatedBooks(t *testing.T) {
	engine := NewGST()

	result, err := engine.CalculateTax(compliance.TaxableTransaction{
		Amount:   1000,
		Currency: "INR",
		Category: "goods",
		HSNCode:  "4901",
	})
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if result.TaxAmount != 0 || result.TotalAmount != 1000 {
		t.Fatalf("result = %+v, want zero GST", result)
	}
}

func TestGSTHSNLookup(t *testing.T) {
	if got := RateForHSN("3004"); got != 0.12 {
		t.Fatalf("RateForHSN(3004) = %v, want 0.12", got)
	}
}

func TestGSTINValidationPasses(t *testing.T) {
	if !ValidGSTIN("29ABCDE1234F1Z5") {
		t.Fatal("expected valid GSTIN")
	}
}

func TestGSTINValidationFails(t *testing.T) {
	engine := NewGST()

	result, err := engine.ValidateInvoice(compliance.InvoiceData{
		InvoiceNumber: "INV-1",
		InvoiceDate:   time.Now(),
		SellerTaxID:   "BAD-GSTIN",
		Amount:        1000,
		Currency:      "INR",
		LineItems: []compliance.LineItemData{
			{Description: "Valve", Quantity: 1, UnitPrice: 1000, HSNCode: "8481", TaxRate: 0.18},
		},
	})
	if err != nil {
		t.Fatalf("ValidateInvoice: %v", err)
	}
	if result.Valid {
		t.Fatalf("invalid GSTIN should fail: %+v", result)
	}
}
