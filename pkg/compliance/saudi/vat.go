// Package saudi implements compliance.TaxEngine for Saudi Arabia VAT
// (ZATCA jurisdiction). Tax math follows the KSA VAT Law: 15% standard rate
// (since 2020-07-01), zero-rated (Z), exempt (E) and out-of-scope (O)
// categories per ZATCA's UBL tax category codes, and the reverse-charge
// mechanism for imported services. Amounts are SAR with 2-decimal (halala)
// rounding, matching ZATCA's BR-KSA arithmetic validation.
//
// ZATCA Phase 2 e-invoicing artifacts (UBL 2.1 XML, invoice hash, ECDSA
// signature, QR TLV) live in this package's sibling files; this file is the
// pure tax-calculation and invoice-validation engine.
package saudi

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"ph_holdings_app/pkg/compliance"
)

// StandardVATRate is the KSA standard VAT rate (15% since 2020-07-01).
const StandardVATRate = 0.15

// ZATCA UBL tax category codes (UNCL5305 subset used by BR-KSA rules).
const (
	CategoryStandard   = "S" // standard rate 15%
	CategoryZeroRated  = "Z" // zero-rated supplies
	CategoryExempt     = "E" // exempt supplies
	CategoryOutOfScope = "O" // services outside scope of tax / not subject
)

// vatNumberPattern: KSA VAT registration numbers are 15 digits that start
// and end with '3' (ZATCA onboarding rule).
var vatNumberPattern = regexp.MustCompile(`^3[0-9]{13}3$`)

// zeroRatedCategories per KSA VAT Law art. 32-36 (exports outside GCC,
// international transport, qualifying medicines/medical goods, investment
// metals >= 99% purity).
var zeroRatedCategories = map[string]bool{
	"export":                  true,
	"exports":                 true,
	"international_transport": true,
	"international transport": true,
	"qualifying_medicines":    true,
	"qualifying medicines":    true,
	"medical_goods":           true,
	"medical goods":           true,
	"investment_metals":       true,
	"investment metals":       true,
}

// exemptCategories per KSA VAT Law (margin-based financial services, life
// insurance, residential real estate lease/sale).
var exemptCategories = map[string]bool{
	"financial_services":      true,
	"financial services":      true,
	"life_insurance":          true,
	"life insurance":          true,
	"residential_real_estate": true,
	"residential real estate": true,
	"residential_rent":        true,
	"residential rent":        true,
}

// outOfScopeCategories — supplies outside the scope of KSA VAT.
var outOfScopeCategories = map[string]bool{
	"out_of_scope": true,
	"out of scope": true,
	"not_subject":  true,
	"not subject":  true,
}

// reverseChargeCategories — imported services where the KSA-registered buyer
// self-accounts for output VAT (and reclaims it as input tax if entitled).
var reverseChargeCategories = map[string]bool{
	"imported_services":  true,
	"imported services":  true,
	"import_of_services": true,
	"import of services": true,
}

// SaudiVAT implements compliance.TaxEngine for Saudi Arabia.
type SaudiVAT struct{}

func New() *SaudiVAT {
	return &SaudiVAT{}
}

func (s *SaudiVAT) Jurisdiction() compliance.Jurisdiction {
	return compliance.JurisdictionSaudi
}

func (s *SaudiVAT) Name() string {
	return "Saudi Arabia VAT (ZATCA)"
}

// CalculateTax computes KSA VAT for a transaction. For reverse-charge
// imported services the 15% VAT is computed and reported in the breakdown
// (the buyer must self-account for it), but TotalAmount stays equal to the
// base amount — the foreign supplier does not charge the VAT.
func (s *SaudiVAT) CalculateTax(tx compliance.TaxableTransaction) (*compliance.TaxResult, error) {
	if tx.Amount < 0 {
		return nil, errors.New("amount cannot be negative")
	}

	base := roundSAR(tx.Amount)
	result := &compliance.TaxResult{
		BaseAmount:   base,
		Jurisdiction: compliance.JurisdictionSaudi,
	}

	if isReverseCharge(tx) {
		tax := roundSAR(base * StandardVATRate)
		result.TaxAmount = tax
		// Payable to supplier excludes the self-accounted VAT.
		result.TotalAmount = base
		result.TaxBreakdown = []compliance.TaxComponent{
			{Name: "VAT (reverse charge)", Rate: StandardVATRate, Amount: tax},
		}
		return result, nil
	}

	rate := s.rateFor(tx)
	tax := roundSAR(base * rate)
	result.TaxAmount = tax
	result.TotalAmount = roundSAR(base + tax)
	result.TaxBreakdown = []compliance.TaxComponent{
		{Name: "VAT", Rate: rate, Amount: tax},
	}
	return result, nil
}

// ValidateInvoice checks KSA invoice fields: VAT registration number format
// (15 digits, starts and ends with 3), non-negative amounts, SAR currency
// expectation (VAT must be expressed in SAR even on FX invoices), line-item
// sanity, and 2-decimal tax arithmetic consistency.
func (s *SaudiVAT) ValidateInvoice(inv compliance.InvoiceData) (*compliance.ValidationResult, error) {
	result := &compliance.ValidationResult{Valid: true}

	if strings.TrimSpace(inv.InvoiceNumber) == "" {
		result.Errors = append(result.Errors, "invoice number is required")
	}
	if !ValidVATNumber(inv.SellerTaxID) {
		result.Errors = append(result.Errors, "seller VAT registration number must be 15 digits starting and ending with 3")
	}
	if strings.TrimSpace(inv.BuyerTaxID) != "" && !ValidVATNumber(inv.BuyerTaxID) {
		result.Errors = append(result.Errors, "buyer VAT registration number must be 15 digits starting and ending with 3")
	}
	if inv.Amount < 0 {
		result.Errors = append(result.Errors, "invoice amount cannot be negative")
	}
	if inv.TaxAmount < 0 {
		result.Errors = append(result.Errors, "tax amount cannot be negative")
	}
	if inv.Currency != "" && strings.ToUpper(strings.TrimSpace(inv.Currency)) != "SAR" {
		result.Warnings = append(result.Warnings, "ZATCA requires the VAT amount expressed in SAR; non-SAR invoices need a SAR tax total")
	}

	for i, item := range inv.LineItems {
		if item.Quantity <= 0 {
			result.Errors = append(result.Errors, lineError(i, "quantity must be greater than zero"))
		}
		if item.UnitPrice < 0 {
			result.Errors = append(result.Errors, lineError(i, "unit price cannot be negative"))
		}
		if item.TaxRate != 0 && math.Abs(item.TaxRate-StandardVATRate) > 0.000001 {
			result.Errors = append(result.Errors, lineError(i, "tax rate must be 0% or 15%"))
		}
	}

	// BR-KSA arithmetic check: when the invoice carries a standard-rated tax
	// amount, it must equal round(base × 15%, 2) within a halala.
	if inv.TaxAmount > 0 && inv.Amount > 0 {
		expected := roundSAR(inv.Amount * StandardVATRate)
		if math.Abs(inv.TaxAmount-expected) > 0.01 {
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"tax amount %.2f does not match 15%% of base %.2f (expected %.2f) — verify mixed-category lines",
				inv.TaxAmount, inv.Amount, expected))
		}
	}

	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result, nil
}

func (s *SaudiVAT) TaxRates() []compliance.TaxRate {
	return []compliance.TaxRate{
		{Name: "Standard VAT", Rate: StandardVATRate, Category: "goods,services", Description: "Standard 15% Saudi VAT"},
		{Name: "Zero-rated", Rate: 0, Category: "exports,international_transport,qualifying_medicines,investment_metals", Description: "Zero-rated supplies (ZATCA category Z)"},
		{Name: "Exempt", Rate: 0, Category: "financial_services,life_insurance,residential_real_estate", Description: "Exempt supplies (ZATCA category E)"},
		{Name: "Out of scope", Rate: 0, Category: "out_of_scope", Description: "Supplies outside the scope of KSA VAT (ZATCA category O)"},
		{Name: "Reverse charge", Rate: StandardVATRate, Category: "imported_services", Description: "Buyer self-accounts 15% VAT on imported services"},
	}
}

// CategoryCode maps a transaction category to its ZATCA UBL tax category
// code (S/Z/E/O) — used by the e-invoice XML generator.
func CategoryCode(category string) string {
	c := normalize(category)
	switch {
	case zeroRatedCategories[c]:
		return CategoryZeroRated
	case exemptCategories[c]:
		return CategoryExempt
	case outOfScopeCategories[c]:
		return CategoryOutOfScope
	default:
		return CategoryStandard
	}
}

// ValidVATNumber reports whether the value is a well-formed KSA VAT
// registration number (15 digits, starts and ends with 3).
func ValidVATNumber(vat string) bool {
	return vatNumberPattern.MatchString(strings.TrimSpace(vat))
}

func (s *SaudiVAT) rateFor(tx compliance.TaxableTransaction) float64 {
	category := normalize(tx.Category)
	customerType := normalize(tx.CustomerType)
	if zeroRatedCategories[category] || customerType == "export" {
		return 0
	}
	if exemptCategories[category] || outOfScopeCategories[category] {
		return 0
	}
	return StandardVATRate
}

func isReverseCharge(tx compliance.TaxableTransaction) bool {
	if reverseChargeCategories[normalize(tx.Category)] {
		return true
	}
	supplier := normalize(tx.SupplierType)
	return (supplier == "non_resident" || supplier == "non resident" || supplier == "foreign") &&
		normalize(tx.Category) == "services"
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// roundSAR rounds to 2 decimals (halalas), half away from zero — matching
// ZATCA's BR-KSA 2-decimal arithmetic validation.
func roundSAR(value float64) float64 {
	return math.Round(value*100) / 100
}

func lineError(index int, message string) string {
	return fmt.Sprintf("line %d: %s", index+1, message)
}
