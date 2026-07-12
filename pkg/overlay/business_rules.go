package overlay

// This file extends the company overlay with the BUSINESS-POLICY numbers that
// used to be hardcoded constants scattered across the codebase:
//
//   - business_invariants.go            (margin floors, grade discounts, payment terms)
//   - pkg/engines/costing_engine.go     (ProductMarkupRules, GetCustomerDiscount, GetPaymentTerms)
//   - pkg/engines/geometry_bridge.go    (ABB tender margin floor)
//   - app_costing_exports_surface.go    (assessCostingRisk thresholds)
//
// Moving them here makes the company's pricing/approval policy CONFIGURATION,
// not code — a different vertical can ship different numbers via overlay.json
// without recompiling.
//
// FINANCIAL SEMANTICS ARE SACRED. BuiltinDefaults() reproduces every value
// byte-identically; this is a pure relocation of WHERE the numbers live, never
// a change to the numbers themselves.

// GradePolicy holds the per-grade costing/payment policy for a customer grade
// (A/B/C/D). It consolidates the values that used to live in three places:
// costing_engine.go's GetCustomerDiscount + GetPaymentTerms switches and the
// grade invariants in business_invariants.go.
type GradePolicy struct {
	// Terms is the human-readable payment-terms string (e.g. "Net 45 days").
	Terms string `json:"terms"`

	// AdvancePct is the required advance payment as a fraction (0.0 .. 1.0).
	AdvancePct float64 `json:"advance_pct"`

	// MaxDiscount is the maximum allowed customer discount as a fraction.
	MaxDiscount float64 `json:"max_discount"`

	// MaxDays is the predicted-payment-days ceiling used by the payment-term
	// invariants (0 means "no ceiling check for this grade").
	MaxDays int `json:"max_days"`
}

// BusinessRules holds the company's costing/approval policy numbers.
type BusinessRules struct {
	// MinMarginPct is the absolute minimum margin (fraction) for an approved
	// order; below this an order should be declined / flagged. (0.08 = 8%)
	MinMarginPct float64 `json:"min_margin_pct"`

	// ABBCompetitionMinMargin is the margin floor (fraction) required when a
	// named competitor is bidding. (0.15 = 15%)
	ABBCompetitionMinMargin float64 `json:"abb_competition_min_margin"`

	// EmergencyMinMarginPct is the recommended margin (fraction) for emergency
	// orders, which should carry premium pricing. (0.20 = 20%)
	EmergencyMinMarginPct float64 `json:"emergency_min_margin_pct"`

	// ApprovalThresholdMargin is the margin (fraction) below which a costing
	// needs manager approval. (0.20 = 20%)
	ApprovalThresholdMargin float64 `json:"approval_threshold_margin"`

	// LargeOrderThresholdBHD is the order value (BHD) above which a credit
	// check warning is raised. (10000)
	LargeOrderThresholdBHD float64 `json:"large_order_threshold_bhd"`

	// MonthlyOperatingCostBHD is the documented all-in monthly operating cost
	// (BHD), used for runway / sustainability reporting. (15000)
	MonthlyOperatingCostBHD float64 `json:"monthly_operating_cost_bhd"`

	// NamedCompetitors are the competitors the company explicitly prices
	// against. The first entry is used in risk-warning messages. (["ABB"])
	NamedCompetitors []string `json:"named_competitors"`

	// GradePaymentTerms maps a customer grade key ("A".."D") to its policy.
	GradePaymentTerms map[string]GradePolicy `json:"grade_payment_terms"`
}

// ProductMarkupRule maps a product type to its standard margin fraction.
type ProductMarkupRule struct {
	ProductType string  `json:"product_type"`
	Margin      float64 `json:"margin"`
}

// ProductMargin returns the standard margin (fraction) for a product type, or
// DefaultProductMargin when the type has no specific rule. This reproduces the
// old costing_engine.GetProductMargin behaviour exactly.
func (o *CompanyOverlay) ProductMargin(productType string) float64 {
	for _, r := range o.ProductMarkupRules {
		if r.ProductType == productType {
			return r.Margin
		}
	}
	return o.DefaultProductMargin
}

// GradePolicyFor returns the policy for a grade key ("A".."D"). The boolean is
// false when the grade is unknown, letting callers apply a historical fallback.
func (o *CompanyOverlay) GradePolicyFor(grade string) (GradePolicy, bool) {
	p, ok := o.BusinessRules.GradePaymentTerms[grade]
	return p, ok
}

// CustomerDiscount returns the maximum allowed discount fraction for a grade.
// Unknown grades return 0.00, matching the old GetCustomerDiscount default case.
func (o *CompanyOverlay) CustomerDiscount(grade string) float64 {
	if p, ok := o.BusinessRules.GradePaymentTerms[grade]; ok {
		return p.MaxDiscount
	}
	return 0.0
}

// PaymentTerms returns the payment-terms string and advance fraction for a
// grade. Unknown grades fall back to grade B's policy (the historical
// default-case value), then to "Net 90 days"/0 if B is also absent.
func (o *CompanyOverlay) PaymentTerms(grade string) (string, float64) {
	if p, ok := o.BusinessRules.GradePaymentTerms[grade]; ok {
		return p.Terms, p.AdvancePct
	}
	if p, ok := o.BusinessRules.GradePaymentTerms["B"]; ok {
		return p.Terms, p.AdvancePct
	}
	return "Net 90 days", 0.0
}

// CompetitorName returns the primary named competitor for risk-warning display,
// or "competitor" when none is configured.
func (o *CompanyOverlay) CompetitorName() string {
	if len(o.BusinessRules.NamedCompetitors) > 0 {
		return o.BusinessRules.NamedCompetitors[0]
	}
	return "competitor"
}

// active is the process-wide overlay singleton used by packages that cannot
// import package main (notably pkg/engines). It defaults to BuiltinDefaults so
// engines work offline with zero configuration. package main keeps its own
// activeOverlay reference in sync via SetActive (called from setActiveOverlay).
var active = BuiltinDefaults()

// Active returns the process-wide active overlay. It never returns nil.
func Active() *CompanyOverlay {
	if active == nil {
		active = BuiltinDefaults()
	}
	return active
}

// SetActive replaces the process-wide active overlay. A nil argument is a no-op,
// so the singleton always stays non-nil.
func SetActive(o *CompanyOverlay) {
	if o != nil {
		active = o
	}
}
