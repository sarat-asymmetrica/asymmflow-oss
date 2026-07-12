package india

import (
	"fmt"
	"math"
)

const (
	oldStandardDeduction = 50000
	newStandardDeduction = 75000
)

// IndiaIncomeTax provides income tax calculations.
type IndiaIncomeTax struct{}

func NewIncomeTax() *IndiaIncomeTax {
	return &IndiaIncomeTax{}
}

// Deductions holds all claimed deductions.
type Deductions struct {
	Section80C        float64 `json:"section_80c"`
	Section80CCD      float64 `json:"section_80ccd"`
	Section80D        float64 `json:"section_80d"`
	Section80DSenior  float64 `json:"section_80d_senior"`
	Section80E        float64 `json:"section_80e"`
	Section80G        float64 `json:"section_80g"`
	Section24         float64 `json:"section_24"`
	HRAExemption      float64 `json:"hra_exemption"`
	StandardDeduction float64 `json:"standard_deduction"`
}

// IncomeTaxResult holds computed tax.
type IncomeTaxResult struct {
	TaxableIncome float64      `json:"taxable_income"`
	TaxAmount     float64      `json:"tax_amount"`
	Surcharge     float64      `json:"surcharge"`
	Cess          float64      `json:"cess"`
	TotalTax      float64      `json:"total_tax"`
	EffectiveRate float64      `json:"effective_rate"`
	SlabBreakdown []SlabDetail `json:"slab_breakdown"`
}

// SlabDetail shows tax for one income slab.
type SlabDetail struct {
	From float64 `json:"from"`
	To   float64 `json:"to"`
	Rate float64 `json:"rate"`
	Tax  float64 `json:"tax"`
}

// RegimeComparison shows both regimes side by side.
type RegimeComparison struct {
	OldRegime      *IncomeTaxResult `json:"old_regime"`
	NewRegime      *IncomeTaxResult `json:"new_regime"`
	Savings        float64          `json:"savings"`
	Recommendation string           `json:"recommendation"`
}

type taxSlab struct {
	from float64
	to   float64
	rate float64
}

var oldRegimeSlabs = []taxSlab{
	{from: 0, to: 250000, rate: 0},
	{from: 250000, to: 500000, rate: 0.05},
	{from: 500000, to: 1000000, rate: 0.20},
	{from: 1000000, to: 0, rate: 0.30},
}

var newRegimeSlabs = []taxSlab{
	{from: 0, to: 400000, rate: 0},
	{from: 400000, to: 800000, rate: 0.05},
	{from: 800000, to: 1200000, rate: 0.10},
	{from: 1200000, to: 1600000, rate: 0.15},
	{from: 1600000, to: 2000000, rate: 0.20},
	{from: 2000000, to: 2400000, rate: 0.25},
	{from: 2400000, to: 0, rate: 0.30},
}

// CalculateOldRegime computes tax under the old regime with deductions.
func (t *IndiaIncomeTax) CalculateOldRegime(income float64, deductions Deductions) *IncomeTaxResult {
	if income < 0 {
		income = 0
	}
	taxable := math.Max(0, income-oldStandardDeduction-eligibleOldDeductions(deductions))
	result := calculateSlabTax(taxable, income, oldRegimeSlabs)
	if taxable <= 700000 {
		result.TaxAmount = 0
		result.Surcharge = 0
		result.Cess = 0
		result.TotalTax = 0
	}
	result.TaxableIncome = roundRupee(taxable)
	result.EffectiveRate = effectiveRate(result.TotalTax, income)
	return result
}

// CalculateNewRegime computes tax under the new regime.
func (t *IndiaIncomeTax) CalculateNewRegime(income float64) *IncomeTaxResult {
	if income < 0 {
		income = 0
	}
	taxable := math.Max(0, income-newStandardDeduction)
	result := calculateSlabTax(taxable, income, newRegimeSlabs)
	if taxable <= 1200000 {
		result.TaxAmount = 0
		result.Surcharge = 0
		result.Cess = 0
		result.TotalTax = 0
	}
	result.TaxableIncome = roundRupee(taxable)
	result.EffectiveRate = effectiveRate(result.TotalTax, income)
	return result
}

// Compare returns both regimes with a recommendation.
func (t *IndiaIncomeTax) Compare(income float64, deductions Deductions) *RegimeComparison {
	oldRegime := t.CalculateOldRegime(income, deductions)
	newRegime := t.CalculateNewRegime(income)
	savings := roundRupee(newRegime.TotalTax - oldRegime.TotalTax)

	recommendation := "Both regimes are equal"
	if savings > 0 {
		recommendation = fmt.Sprintf("Old regime saves ₹%.0f", savings)
	} else if savings < 0 {
		recommendation = fmt.Sprintf("New regime saves ₹%.0f", math.Abs(savings))
	}

	return &RegimeComparison{
		OldRegime:      oldRegime,
		NewRegime:      newRegime,
		Savings:        savings,
		Recommendation: recommendation,
	}
}

func eligibleOldDeductions(d Deductions) float64 {
	standard := d.StandardDeduction
	if standard <= 0 {
		standard = oldStandardDeduction
	}
	// Standard deduction is applied separately in CalculateOldRegime. This field
	// is retained for API completeness without double-counting it.
	_ = standard

	return min(d.Section80C, 150000) +
		min(d.Section80CCD, 50000) +
		min(d.Section80D, 25000) +
		min(d.Section80DSenior, 50000) +
		d.Section80E +
		d.Section80G +
		min(d.Section24, 200000) +
		d.HRAExemption
}

func calculateSlabTax(taxableIncome, grossIncome float64, slabs []taxSlab) *IncomeTaxResult {
	var tax float64
	breakdown := make([]SlabDetail, 0, len(slabs))
	for _, slab := range slabs {
		upper := slab.to
		if upper == 0 || upper > taxableIncome {
			upper = taxableIncome
		}
		if upper <= slab.from {
			continue
		}
		slabAmount := upper - slab.from
		slabTax := roundRupee(slabAmount * slab.rate)
		tax += slabTax
		breakdown = append(breakdown, SlabDetail{
			From: slab.from,
			To:   slab.to,
			Rate: slab.rate,
			Tax:  slabTax,
		})
		if slab.to == 0 || taxableIncome <= slab.to {
			break
		}
	}

	tax = roundRupee(tax)
	surcharge := roundRupee(tax * surchargeRate(grossIncome))
	cess := roundRupee((tax + surcharge) * 0.04)
	total := roundRupee(tax + surcharge + cess)

	return &IncomeTaxResult{
		TaxableIncome: taxableIncome,
		TaxAmount:     tax,
		Surcharge:     surcharge,
		Cess:          cess,
		TotalTax:      total,
		EffectiveRate: effectiveRate(total, grossIncome),
		SlabBreakdown: breakdown,
	}
}

func surchargeRate(income float64) float64 {
	switch {
	case income > 20000000:
		return 0.25
	case income > 10000000:
		return 0.15
	case income > 5000000:
		return 0.10
	default:
		return 0
	}
}

func effectiveRate(totalTax, income float64) float64 {
	if income <= 0 {
		return 0
	}
	return roundPercent(totalTax / income)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func roundRupee(value float64) float64 {
	return math.Round(value)
}

func roundPercent(value float64) float64 {
	return math.Round(value*10000) / 10000
}
