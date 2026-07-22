package india

import (
	"errors"
	"math"
	"strings"

	"ph_holdings_app/pkg/compliance"
)

var hsnRates = map[string]compliance.TaxRate{
	"8481": {Name: "Valves, taps, cocks", Rate: 0.18, Category: "goods", Description: "Valves, taps, cocks and similar appliances"},
	"9032": {Name: "Automatic regulating instruments", Rate: 0.18, Category: "goods", Description: "Automatic regulating or controlling instruments"},
	"8536": {Name: "Electrical switching apparatus", Rate: 0.18, Category: "goods", Description: "Electrical switching and protection apparatus"},
	"7304": {Name: "Tubes and pipes", Rate: 0.18, Category: "goods", Description: "Iron or steel tubes and pipes"},
	"4901": {Name: "Printed books", Rate: 0, Category: "goods", Description: "Printed books, brochures and similar printed matter"},
	"3004": {Name: "Medicaments", Rate: 0.12, Category: "goods", Description: "Medicaments"},
	"1006": {Name: "Rice", Rate: 0.05, Category: "goods", Description: "Rice"},
	"2710": {Name: "Petroleum oils", Rate: 0.18, Category: "goods", Description: "Petroleum oils and oils from bituminous minerals"},
	"3926": {Name: "Plastic articles", Rate: 0.18, Category: "goods", Description: "Other articles of plastics"},
	"4016": {Name: "Rubber articles", Rate: 0.18, Category: "goods", Description: "Other articles of vulcanised rubber"},
	"7318": {Name: "Fasteners", Rate: 0.18, Category: "goods", Description: "Screws, bolts, nuts and similar articles"},
	"8413": {Name: "Pumps", Rate: 0.18, Category: "goods", Description: "Pumps for liquids"},
	"8414": {Name: "Compressors and fans", Rate: 0.18, Category: "goods", Description: "Air or vacuum pumps and compressors"},
	"8421": {Name: "Filtering machinery", Rate: 0.18, Category: "goods", Description: "Filtering or purifying machinery"},
	"8471": {Name: "Computers", Rate: 0.18, Category: "goods", Description: "Automatic data processing machines"},
	"8504": {Name: "Transformers", Rate: 0.18, Category: "goods", Description: "Electrical transformers and converters"},
	"8507": {Name: "Batteries", Rate: 0.18, Category: "goods", Description: "Electric accumulators"},
	"8544": {Name: "Insulated wire", Rate: 0.18, Category: "goods", Description: "Insulated wire, cable and conductors"},
	"9026": {Name: "Flow instruments", Rate: 0.18, Category: "goods", Description: "Instruments for measuring flow, level or pressure"},
	"9983": {Name: "Professional services", Rate: 0.18, Category: "services", Description: "Other professional, technical and business services"},
}

// IndiaGST implements compliance.TaxEngine for India GST.
type IndiaGST struct{}

func NewGST() *IndiaGST {
	return &IndiaGST{}
}

func (g *IndiaGST) Jurisdiction() compliance.Jurisdiction {
	return compliance.JurisdictionIndia
}

func (g *IndiaGST) Name() string {
	return "India GST"
}

func (g *IndiaGST) CalculateTax(tx compliance.TaxableTransaction) (*compliance.TaxResult, error) {
	if tx.Amount < 0 {
		return nil, errors.New("amount cannot be negative")
	}
	rate := RateForHSN(tx.HSNCode)
	if rate < 0 {
		rate = defaultRateForCategory(tx.Category)
	}

	tax := roundINR(tx.Amount * rate)
	result := &compliance.TaxResult{
		BaseAmount:   roundINR(tx.Amount),
		TaxAmount:    tax,
		TotalAmount:  roundINR(tx.Amount + tax),
		Jurisdiction: compliance.JurisdictionIndia,
	}

	if rate == 0 {
		result.TaxBreakdown = []compliance.TaxComponent{{Name: "GST", Rate: 0, Amount: 0}}
		return result, nil
	}

	if isInterState(tx.PlaceOfSupply) {
		result.TaxBreakdown = []compliance.TaxComponent{{Name: "IGST", Rate: rate, Amount: tax}}
		return result, nil
	}

	halfRate := rate / 2
	halfAmount := roundINR(tax / 2)
	result.TaxBreakdown = []compliance.TaxComponent{
		{Name: "CGST", Rate: halfRate, Amount: halfAmount},
		{Name: "SGST", Rate: halfRate, Amount: roundINR(tax - halfAmount)},
	}
	return result, nil
}

func (g *IndiaGST) ValidateInvoice(inv compliance.InvoiceData) (*compliance.ValidationResult, error) {
	result := &compliance.ValidationResult{Valid: true}

	if strings.TrimSpace(inv.InvoiceNumber) == "" {
		result.Errors = append(result.Errors, "invoice number is required")
	}
	if !ValidGSTIN(inv.SellerTaxID) {
		result.Errors = append(result.Errors, "seller GSTIN is invalid")
	}
	if strings.TrimSpace(inv.BuyerTaxID) != "" && !ValidGSTIN(inv.BuyerTaxID) {
		result.Errors = append(result.Errors, "buyer GSTIN is invalid")
	}
	if inv.Amount < 0 {
		result.Errors = append(result.Errors, "invoice amount cannot be negative")
	}
	if strings.ToUpper(inv.Currency) != "" && strings.ToUpper(inv.Currency) != "INR" {
		result.Warnings = append(result.Warnings, "India GST invoices are expected in INR or need FX disclosure")
	}
	for i, item := range inv.LineItems {
		if strings.TrimSpace(item.HSNCode) == "" {
			result.Errors = append(result.Errors, lineError(i, "HSN/SAC code is required"))
		}
		if item.Quantity <= 0 {
			result.Errors = append(result.Errors, lineError(i, "quantity must be greater than zero"))
		}
		if item.UnitPrice < 0 {
			result.Errors = append(result.Errors, lineError(i, "unit price cannot be negative"))
		}
	}

	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result, nil
}

func (g *IndiaGST) TaxRates() []compliance.TaxRate {
	return []compliance.TaxRate{
		{Name: "Nil GST", Rate: 0, Category: "exempt,zero-rated", Description: "Nil-rated or exempt goods"},
		{Name: "Lower GST", Rate: 0.05, Category: "goods", Description: "5% GST slab"},
		{Name: "Merit GST", Rate: 0.12, Category: "goods", Description: "12% GST slab"},
		{Name: "Standard GST", Rate: 0.18, Category: "goods,services", Description: "18% GST slab"},
		{Name: "Demerit GST", Rate: 0.28, Category: "goods", Description: "28% GST slab"},
	}
}

func RateForHSN(hsn string) float64 {
	hsn = strings.TrimSpace(hsn)
	if hsn == "" {
		return -1
	}
	if rate, ok := hsnRates[hsn]; ok {
		return rate.Rate
	}
	if len(hsn) > 4 {
		if rate, ok := hsnRates[hsn[:4]]; ok {
			return rate.Rate
		}
	}
	return -1
}

func defaultRateForCategory(category string) float64 {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "exempt", "zero-rated", "zero rated":
		return 0
	case "basic_goods":
		return 0.05
	case "medicaments":
		return 0.12
	case "luxury":
		return 0.28
	default:
		return 0.18
	}
}

func isInterState(placeOfSupply string) bool {
	placeOfSupply = strings.ToUpper(strings.TrimSpace(placeOfSupply))
	if strings.Contains(placeOfSupply, ":") {
		parts := strings.SplitN(placeOfSupply, ":", 2)
		return strings.TrimSpace(parts[0]) != strings.TrimSpace(parts[1])
	}
	if strings.Contains(placeOfSupply, "-") {
		parts := strings.SplitN(placeOfSupply, "-", 2)
		return strings.TrimSpace(parts[0]) != strings.TrimSpace(parts[1])
	}
	return false
}

func roundINR(value float64) float64 {
	return math.Round(value*100) / 100
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
