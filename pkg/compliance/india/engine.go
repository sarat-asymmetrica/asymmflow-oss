package india

// This file is the transaction-level GST computation engine (India Spec-01
// §2 B3): pure functions, no DB, no globals, no network. It is additive to
// the legacy IndiaGST (gst.go, bound as a compliance.TaxEngine) — that engine
// is untouched; this one will be wired to documents in a later mission (B4)
// and feeds the GSTR-1 exporter (B5). Rates and cess come ONLY from
// overlay.IndiaCompanyConfig data (§0 G3) — the legacy hsnRates map in
// gst.go is legacy-only and never consulted here.

import (
	"fmt"
	"math"
	"strings"

	"ph_holdings_app/pkg/overlay"
)

// Supplier carries the selling division's GST registration facts that this
// engine needs (everything else on IndiaDivisionProfile — the GSTIN itself,
// for instance — is a B4 rendering concern, not a computation input).
type Supplier struct {
	// StateCode is the division's registered GST state code (2 digits).
	StateCode string
	// Composition marks the supplier as a composition taxable person (§0
	// G6): the result carries NO tax at all, only the Composition marker —
	// this feeds a Bill of Supply in B4, never a tax invoice.
	Composition bool
}

// Supply carries the transaction-level facts that decide intra vs
// inter-state classification and the special-case flags from §0 G2.
type Supply struct {
	// PlaceOfSupplyStateCode is the 2-digit GST state code of the place of
	// supply (generally the delivery destination state for goods).
	PlaceOfSupplyStateCode string
	// SEZ marks the buyer as a Special Economic Zone unit. Per §0 G2 this is
	// ALWAYS inter-state/zero-rated, even when supplier and place-of-supply
	// states are identical (IGST Act s.16) — the LUT workflow itself is out
	// of scope this wave.
	SEZ bool
	// ReverseCharge marks the supply as reverse-charge. Computation is
	// UNCHANGED by this flag — it is carried through to the result purely so
	// B4 can render the mandatory "tax payable on reverse charge basis"
	// statement (§0 G5). Never branch tax math on this field.
	ReverseCharge bool
	// B2B is true when the buyer holds a GSTIN. Feeds the G4 HSN-digit
	// mandate (B2B needs digits at the lower AATO tier; B2C does not).
	B2B bool
	// Export marks an export supply. Per §0 G4, export lines always require
	// 8-digit HSN regardless of AATO tier. Only HSN-digit validation reads
	// this flag this wave (export duty/zero-rating mechanics are out of
	// scope).
	Export bool
}

// Line is one invoice line item feeding the engine. Quantity and UQC pass
// through to LineResult untouched — the engine does not validate them; that
// is a B4 rendering concern.
type Line struct {
	HSN             string
	Description     string
	Quantity        float64
	UQC             string
	TaxableValueINR float64
}

// EngineConfig bundles the config-as-data inputs ComputeInvoiceGST needs.
// Rates must be the RESOLVED config (i.e. the result of
// (*overlay.CompanyOverlay).IndiaConfig(), not a raw nil-defaulted
// overlay.India pointer) — a zero HSNTierThresholdINR is treated as a
// literal ₹0 threshold, not the statutory ₹5cr default, so an unresolved
// config will silently push every AATO into TierAbove5Cr.
type EngineConfig struct {
	Rates overlay.IndiaCompanyConfig
	// AATOINR is the PAN's Aggregate Annual Turnover for the fiscal year
	// (§0 G1: PAN-level, across every sibling GSTIN) — the caller resolves
	// this (via aato.go's AATOSource/ResolveAATO seam); the engine only
	// classifies it against the tier boundary.
	AATOINR float64
}

// Classification is the intra/inter-state character of a supply (§0 G2).
type Classification int

const (
	IntraState Classification = iota
	InterState
)

func (c Classification) String() string {
	if c == InterState {
		return "inter-state"
	}
	return "intra-state"
}

// LineResult is one line's computed GST — feeds the Rule-46 invoice
// renderer (per-line HSN/rate/tax columns, B4) and the GSTR-1 HSN summary
// export (B5).
type LineResult struct {
	HSN             string
	Description     string
	Quantity        float64
	UQC             string
	TaxableValueINR float64
	// RatePct and CessPct are the GST/cess rates actually applied, as
	// percentages (18 = 18%) — zeroed for composition and SEZ zero-rated
	// lines (neither discloses a rate: a Bill of Supply has no rate column,
	// and a zero-rated supply is exactly that, zero).
	RatePct float64
	CessPct float64
	CGST    float64
	SGST    float64
	IGST    float64
	Cess    float64
}

// TaxINR is the total tax collected on this line (CGST+SGST+IGST+Cess) —
// convenience for renderers that want a line-total-with-tax column.
func (l LineResult) TaxINR() float64 {
	return round2dp(l.CGST + l.SGST + l.IGST + l.Cess)
}

// InvoiceTotals sums each tax head across every line: what a Rule-46
// renderer prints as the invoice footer and what GSTR-1 aggregates read.
type InvoiceTotals struct {
	TaxableValueINR float64
	CGST            float64
	SGST            float64
	IGST            float64
	Cess            float64
}

// InvoiceResult is the full output of ComputeInvoiceGST.
type InvoiceResult struct {
	Classification Classification
	Lines          []LineResult
	Totals         InvoiceTotals
	// ZeroRated is true when the SEZ flag forced inter-state/zero-rate
	// treatment (§0 G2) — distinct from a line simply resolving to a
	// configured 0% rate (e.g. HSN 4901 books), which is not zero-rated in
	// this sense at all, just nil-rated.
	ZeroRated bool
	// Composition is true when the supplier is a composition taxable
	// person (§0 G6): every tax field above is zero and this result feeds a
	// Bill of Supply, never a tax invoice.
	Composition bool
	// ReverseCharge is carried through unchanged from Supply for B4's
	// mandatory invoice text — it never altered the math above.
	ReverseCharge bool
	B2B           bool
}

// HSNValidationError is returned when ComputeInvoiceGST refuses to compute
// because a line fails the §0 G4 HSN-digit mandate or resolves to no
// configured GST rate. Refusal is whole-invoice, never partial — this is
// the house "refuse-to-generate" doctrine (CLAUDE.md invariant 5: tax
// behavior is stop-and-ask, never a silent fallback).
type HSNValidationError struct {
	LineIndex int // 0-based index of the offending line
	HSN       string
	Message   string
}

func (e *HSNValidationError) Error() string {
	return fmt.Sprintf("india: GST computation refused at line %d (HSN %q): %s", e.LineIndex+1, e.HSN, e.Message)
}

func newHSNDigitError(index int, hsn string, required int) *HSNValidationError {
	return &HSNValidationError{
		LineIndex: index,
		HSN:       hsn,
		Message: fmt.Sprintf(
			"requires at least %d HSN/SAC digits under N/N 78/2020-CT, got %d",
			required, len(strings.TrimSpace(hsn)),
		),
	}
}

func newHSNRateError(index int, hsn string) *HSNValidationError {
	return &HSNValidationError{
		LineIndex: index,
		HSN:       hsn,
		Message:   "has no configured GST rate — refusing rather than silently applying a default rate",
	}
}

// ComputeInvoiceGST computes per-line and invoice-level GST for one
// transaction. It validates every line's HSN digit count and rate
// configuration BEFORE computing anything (§0 G4) — a single bad line
// refuses the whole invoice, no partial result.
func ComputeInvoiceGST(supplier Supplier, supply Supply, lines []Line, cfg EngineConfig) (*InvoiceResult, error) {
	tier := AATOTier(cfg.AATOINR, cfg.Rates.HSNTierThresholdINR)

	// Pass 1: refuse-to-generate validation, whole-invoice, before any math.
	ratePcts := make([]float64, len(lines))
	cessPcts := make([]float64, len(lines))
	for i, line := range lines {
		hsn := strings.TrimSpace(line.HSN)

		required := RequiredHSNDigits(tier, supply.B2B)
		if supply.Export {
			required = HSNDigitsForExport()
		}
		if required > 0 && len(hsn) < required {
			return nil, newHSNDigitError(i, line.HSN, required)
		}

		ratePct, cessPct, ok := cfg.Rates.RateForHSN(hsn)
		if !ok {
			return nil, newHSNRateError(i, line.HSN)
		}
		ratePcts[i] = ratePct
		cessPcts[i] = cessPct
	}

	// Pass 2: classification and per-line computation.
	supplierState := strings.TrimSpace(supplier.StateCode)
	posState := strings.TrimSpace(supply.PlaceOfSupplyStateCode)
	interState := supply.SEZ || supplierState != posState

	classification := IntraState
	if interState {
		classification = InterState
	}

	result := &InvoiceResult{
		Classification: classification,
		Lines:          make([]LineResult, 0, len(lines)),
		ZeroRated:      supply.SEZ,
		Composition:    supplier.Composition,
		ReverseCharge:  supply.ReverseCharge,
		B2B:            supply.B2B,
	}

	for i, line := range lines {
		taxableValue := round2dp(line.TaxableValueINR)
		lr := LineResult{
			HSN:             strings.TrimSpace(line.HSN),
			Description:     line.Description,
			Quantity:        line.Quantity,
			UQC:             line.UQC,
			TaxableValueINR: taxableValue,
		}

		switch {
		case supplier.Composition:
			// §0 G6: no tax lines at all — Bill of Supply territory.

		case supply.SEZ:
			// §0 G2: always inter-state/zero-rated, IGST head at 0.
			lr.IGST = 0

		case interState:
			lr.RatePct = ratePcts[i]
			lr.CessPct = cessPcts[i]
			lr.IGST = round2dp(taxableValue * ratePcts[i] / 100)
			lr.Cess = round2dp(taxableValue * cessPcts[i] / 100)

		default: // intra-state
			lr.RatePct = ratePcts[i]
			lr.CessPct = cessPcts[i]
			tax := round2dp(taxableValue * ratePcts[i] / 100)
			cgst := round2dp(tax / 2)
			lr.CGST = cgst
			lr.SGST = round2dp(tax - cgst) // heads sum exactly to tax, even on odd paise
			lr.Cess = round2dp(taxableValue * cessPcts[i] / 100)
		}

		result.Lines = append(result.Lines, lr)
		result.Totals.TaxableValueINR = round2dp(result.Totals.TaxableValueINR + lr.TaxableValueINR)
		result.Totals.CGST = round2dp(result.Totals.CGST + lr.CGST)
		result.Totals.SGST = round2dp(result.Totals.SGST + lr.SGST)
		result.Totals.IGST = round2dp(result.Totals.IGST + lr.IGST)
		result.Totals.Cess = round2dp(result.Totals.Cess + lr.Cess)
	}

	return result, nil
}

// RoundInvoiceHeadsToRupee applies Section 170 CGST Act statutory rounding:
// each tax head total (CGST, SGST, IGST, Cess) is rounded to the nearest
// whole rupee. Taxable value is untouched — the statute rounds "tax,
// interest, penalty, fine or any other sum", not the value of supply.
//
// This is NOT applied by ComputeInvoiceGST automatically. Whether it should
// become the default is an orchestrator ruling pending schema research (the
// GSTR-1 JSON export in a later mission may expect paise-precision totals
// that reconcile line-by-line with the invoice, in which case rupee-rounding
// the invoice footer independently of the export would create a mismatch to
// solve first). Callers who want statutory rounding call this explicitly.
func RoundInvoiceHeadsToRupee(totals InvoiceTotals) InvoiceTotals {
	totals.CGST = math.Round(totals.CGST)
	totals.SGST = math.Round(totals.SGST)
	totals.IGST = math.Round(totals.IGST)
	totals.Cess = math.Round(totals.Cess)
	return totals
}

// round2dp rounds to 2 decimal places (paise), half away from zero — the
// same rule as gst.go's roundINR, just under a different name since that one
// is already declared in this package.
func round2dp(value float64) float64 {
	return math.Round(value*100) / 100
}
