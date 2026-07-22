package overlay

import "strings"

// This file mounts India as a jurisdiction plane on the overlay seam,
// alongside the existing GCC (Bahrain/Saudi) identity fields. It follows the
// SupplierAliasConfig precedent: plane-scoped, dependency-free (no import of
// pkg/compliance/india — that package imports overlay, not the other way),
// and INERT unless configured. A nil India field on CompanyOverlay, or a nil
// India field on every DivisionProfile, means the India plane is unmounted —
// GCC deployments observe no change whatsoever.
//
// Every threshold below is config-not-constant (India Spec-01 §0): GST
// thresholds have been renotified five times since 2020, so BuiltinDefaults()
// never sets these fields and IndiaConfig() is the ONE place the 0⇒default
// resolution happens.

// IndiaCompanyConfig carries PAN-level India facts shared by every division
// under one PAN. It is optional (nil ⇒ India plane unmounted).
type IndiaCompanyConfig struct {
	// PAN is the 10-character company Permanent Account Number. Every
	// division's GSTIN (see IndiaDivisionProfile) must embed this PAN.
	PAN string `json:"pan"`

	// AATOOverrideINR is an optional PAN-level Aggregate Annual Turnover
	// figure supplied directly, for deployments whose invoice history isn't
	// yet in-app to compute it from. 0 ⇒ no override (compute from history).
	AATOOverrideINR float64 `json:"aato_override_inr"`

	// HSNTierThresholdINR is the AATO boundary between the 4-digit and
	// 6-digit HSN mandates (N/N 78/2020-CT). 0 ⇒ default ₹5,00,00,000 (₹5cr).
	HSNTierThresholdINR float64 `json:"hsn_tier_threshold_inr"`

	// EInvoiceThresholdAATO is the AATO above which e-invoicing (IRN/QR)
	// applies. 0 ⇒ default ₹5,00,00,000 (₹5cr). E-invoicing itself is out of
	// scope this wave; this config only feeds the applicability indicator.
	EInvoiceThresholdAATO float64 `json:"einvoice_threshold_aato"`

	// CompositionCeilingGoodsINR is the AATO ceiling for the composition
	// scheme on goods. 0 ⇒ default ₹1,50,00,000 (₹1.5cr).
	CompositionCeilingGoodsINR float64 `json:"composition_ceiling_goods_inr"`

	// CompositionCeilingServicesINR is the AATO ceiling for the composition
	// scheme on services. 0 ⇒ default ₹50,00,000 (₹50L).
	CompositionCeilingServicesINR float64 `json:"composition_ceiling_services_inr"`

	// B2CLThresholdINR is the invoice value above which an inter-state B2C
	// supply is reported as B2CL (large) rather than folded into the B2CS
	// consolidated bucket in GSTR-1. 0 ⇒ default ₹1,00,000 (N/N 12/2024-CT,
	// effective 1 Aug 2024).
	B2CLThresholdINR float64 `json:"b2cl_threshold_inr"`

	// GSTR1SchemaVersion identifies the GST-portal offline-tool JSON schema
	// version this deployment targets. "" ⇒ default "GST3.2.4". The literal
	// in-file schema version string used by the portal tool is unresolved as
	// of this wave (config, not a hardcoded constant, precisely because of
	// that uncertainty — see IN Spec-01 §0 G7).
	GSTR1SchemaVersion string `json:"gstr1_schema_version"`

	// TaxCategories is the GST rate schedule as DATA (G3): rates and cess
	// attach to HSN/SAC prefixes here, never hardcoded into engine logic.
	// RateForHSN resolves the longest matching prefix.
	TaxCategories []GSTTaxCategory `json:"tax_categories"`
}

// GSTTaxCategory is one HSN/SAC-prefix → rate record in the India rate
// schedule (config, per G3 — never hardcoded).
type GSTTaxCategory struct {
	// HSNPrefix is the HSN/SAC code or code prefix this rate applies to
	// (e.g. "8481" or the shorter "84"). Longest-prefix match wins.
	HSNPrefix string `json:"hsn_prefix"`

	// RatePct is the GST rate as a percentage (18 = 18%, not 0.18).
	RatePct float64 `json:"rate_pct"`

	// CessPct is any additional compensation cess as a percentage.
	CessPct float64 `json:"cess_pct"`

	Description string `json:"description"`
}

// IndiaDivisionProfile carries one division's GST registration identity. It
// is optional (nil ⇒ this division has no India plane, even if the company's
// India config is set — mixed-jurisdiction holding companies are possible).
type IndiaDivisionProfile struct {
	// GSTIN is the division's 15-character GST registration number.
	GSTIN string `json:"gstin"`

	// StateCode is the 2-digit GST state code the division is registered in.
	// Must equal GSTIN[0:2] — ValidateOverlayIndia checks this.
	StateCode string `json:"state_code"`

	// Composition marks the division as a composition taxable person: it
	// issues a Bill of Supply (no tax lines, no ITC) instead of a tax
	// invoice, per G6.
	Composition bool `json:"composition"`
}

// IndiaMounted reports whether the India jurisdiction plane is active for
// this overlay: the company-level India config is present AND at least one
// division carries an India profile. Both conditions are required — a
// company-level config with no division GSTINs has nothing to route.
func (o *CompanyOverlay) IndiaMounted() bool {
	if o == nil || o.India == nil {
		return false
	}
	for _, div := range o.Divisions {
		if div.India != nil {
			return true
		}
	}
	return false
}

// IndiaConfig returns the effective India company config with every
// 0-valued threshold resolved to its statutory default. This is the ONE
// place the 0⇒default logic lives (India Spec-01 §0: config-not-constant,
// but callers still need a usable number). Safe to call even when India is
// nil (unmounted) — it returns an all-defaults config in that case.
func (o *CompanyOverlay) IndiaConfig() IndiaCompanyConfig {
	var cfg IndiaCompanyConfig
	if o != nil && o.India != nil {
		cfg = *o.India
	}
	if cfg.HSNTierThresholdINR == 0 {
		cfg.HSNTierThresholdINR = 50000000 // ₹5,00,00,000 (₹5cr), N/N 78/2020-CT
	}
	if cfg.EInvoiceThresholdAATO == 0 {
		cfg.EInvoiceThresholdAATO = 50000000 // ₹5cr — moved 5x historically; config, never constant
	}
	if cfg.CompositionCeilingGoodsINR == 0 {
		cfg.CompositionCeilingGoodsINR = 15000000 // ₹1,50,00,000 (₹1.5cr)
	}
	if cfg.CompositionCeilingServicesINR == 0 {
		cfg.CompositionCeilingServicesINR = 5000000 // ₹50,00,000 (₹50L)
	}
	if cfg.B2CLThresholdINR == 0 {
		cfg.B2CLThresholdINR = 100000 // ₹1,00,000, N/N 12/2024-CT eff. 1 Aug 2024
	}
	if strings.TrimSpace(cfg.GSTR1SchemaVersion) == "" {
		cfg.GSTR1SchemaVersion = "GST3.2.4" // offline-tool version; see IN Spec-01 §0 G7
	}
	return cfg
}

// RateForHSN resolves the GST rate and cess for an HSN/SAC code against this
// config's TaxCategories, matching the longest configured prefix (e.g. an
// exact "8481" entry wins over a shorter "84" entry). ok is false when no
// prefix matches.
func (c IndiaCompanyConfig) RateForHSN(hsn string) (ratePct, cessPct float64, ok bool) {
	hsn = strings.TrimSpace(hsn)
	if hsn == "" {
		return 0, 0, false
	}
	bestLen := -1
	for _, cat := range c.TaxCategories {
		prefix := strings.TrimSpace(cat.HSNPrefix)
		if prefix == "" || !strings.HasPrefix(hsn, prefix) {
			continue
		}
		if len(prefix) > bestLen {
			bestLen = len(prefix)
			ratePct, cessPct, ok = cat.RatePct, cat.CessPct, true
		}
	}
	return ratePct, cessPct, ok
}

// FYStartMonthOrDefault returns the configured fiscal-year start month
// (1-12). A configured 0 (absent from overlay.json) means calendar year for
// every existing GCC book — EXCEPT when the India plane is mounted, where
// G9 mandates April (month 4) as the statutory default so a fresh India
// overlay doesn't need to state the obvious.
func (o *CompanyOverlay) FYStartMonthOrDefault() int {
	if o == nil {
		return 1
	}
	if o.FiscalYearStartMonth != 0 {
		return o.FiscalYearStartMonth
	}
	if o.IndiaMounted() {
		return 4 // April—March, G9
	}
	return 1
}
