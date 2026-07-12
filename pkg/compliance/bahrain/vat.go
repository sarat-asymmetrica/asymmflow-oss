package bahrain

import (
	"errors"
	"math"
	"strings"

	"ph_holdings_app/pkg/compliance"
)

const (
	standardVATRate = 0.10
)

var exemptCategories = map[string]bool{
	"basic_food":              true,
	"basic food":              true,
	"healthcare":              true,
	"education":               true,
	"financial_services":      true,
	"financial services":      true,
	"residential_real_estate": true,
	"residential real estate": true,
}

// BahrainVAT implements compliance.TaxEngine for Bahrain VAT.
type BahrainVAT struct{}

func New() *BahrainVAT {
	return &BahrainVAT{}
}

func (b *BahrainVAT) Jurisdiction() compliance.Jurisdiction {
	return compliance.JurisdictionBahrain
}

func (b *BahrainVAT) Name() string {
	return "Bahrain VAT"
}

func (b *BahrainVAT) CalculateTax(tx compliance.TaxableTransaction) (*compliance.TaxResult, error) {
	if tx.Amount < 0 {
		return nil, errors.New("amount cannot be negative")
	}

	rate := b.rateFor(tx)
	tax := roundBHD(tx.Amount * rate)
	return &compliance.TaxResult{
		BaseAmount:  roundBHD(tx.Amount),
		TaxAmount:   tax,
		TotalAmount: roundBHD(tx.Amount + tax),
		TaxBreakdown: []compliance.TaxComponent{
			{Name: "VAT", Rate: rate, Amount: tax},
		},
		Jurisdiction: compliance.JurisdictionBahrain,
	}, nil
}

func (b *BahrainVAT) ValidateInvoice(inv compliance.InvoiceData) (*compliance.ValidationResult, error) {
	result := &compliance.ValidationResult{Valid: true}

	if strings.TrimSpace(inv.InvoiceNumber) == "" {
		result.Errors = append(result.Errors, "invoice number is required")
	}
	if strings.TrimSpace(inv.SellerTaxID) == "" {
		result.Errors = append(result.Errors, "seller VAT registration number is required")
	}
	if inv.Amount < 0 {
		result.Errors = append(result.Errors, "invoice amount cannot be negative")
	}
	if inv.TaxAmount < 0 {
		result.Errors = append(result.Errors, "tax amount cannot be negative")
	}
	if inv.Currency != "" && strings.ToUpper(inv.Currency) != "BHD" {
		result.Warnings = append(result.Warnings, "Bahrain VAT invoices are expected in BHD or need FX disclosure")
	}

	for i, item := range inv.LineItems {
		if item.Quantity <= 0 {
			result.Errors = append(result.Errors, lineError(i, "quantity must be greater than zero"))
		}
		if item.UnitPrice < 0 {
			result.Errors = append(result.Errors, lineError(i, "unit price cannot be negative"))
		}
		if item.TaxRate != 0 && math.Abs(item.TaxRate-standardVATRate) > 0.000001 {
			result.Errors = append(result.Errors, lineError(i, "tax rate must be 0% or 10%"))
		}
	}

	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result, nil
}

func (b *BahrainVAT) TaxRates() []compliance.TaxRate {
	return []compliance.TaxRate{
		{Name: "Standard VAT", Rate: standardVATRate, Category: "goods,services", Description: "Standard 10% Bahrain VAT"},
		{Name: "Zero-rated", Rate: 0, Category: "exports,international_transport", Description: "Zero-rated supplies"},
		{Name: "Exempt", Rate: 0, Category: "basic_food,healthcare,education,financial_services,residential_real_estate", Description: "Exempt supplies"},
	}
}

func (b *BahrainVAT) rateFor(tx compliance.TaxableTransaction) float64 {
	category := normalize(tx.Category)
	customerType := normalize(tx.CustomerType)
	if category == "export" || category == "exports" || category == "international_transport" || customerType == "export" {
		return 0
	}
	if exemptCategories[category] {
		return 0
	}
	return standardVATRate
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func roundBHD(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func lineError(index int, message string) string {
	return "line " + strconvItoa(index+1) + ": " + message
}

func strconvItoa(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = digits[value%10]
		value /= 10
	}
	return string(buf[i:])
}
