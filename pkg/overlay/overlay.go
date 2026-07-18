// Package overlay provides config-driven company and division profiles.
//
// Instead of hardcoding legal names, addresses, VAT numbers, and bank details
// in Go source, deployers can drop an overlay.json next to the binary (or in
// their data directory) to customise or extend the built-in defaults without
// recompiling.
//
// The built-in defaults reproduce the SYNTHETIC demo values from company_branding.go
// exactly. Behaviour is identical for the two existing divisions — this is a pure
// refactor of where the data lives.
//
// See data/overlay.json at the repo root for an annotated example.
package overlay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DivisionProfile holds all the per-division facts that used to be hardcoded
// in the companyDocumentProfile switch in company_branding.go.
type DivisionProfile struct {
	// Key is the canonical display name of the division (e.g. "Acme Instrumentation").
	Key string `json:"key"`

	// LegalName is the full registered legal name printed on documents.
	LegalName string `json:"legal_name"`

	// VATNumber is the VAT/TRN registration number.
	VATNumber string `json:"vat_number"`

	// City is the primary city (informational, not currently printed separately).
	City string `json:"city"`

	// AddressLines are the postal address lines shown on document headers.
	AddressLines []string `json:"address_lines"`

	// BankDetails are one or more bank account strings shown on documents.
	BankDetails []string `json:"bank_details"`

	// LetterheadAssetName is the internal asset-store key for the letterhead image
	// (e.g. "letterhead" or "letterhead_ahs").
	LetterheadAssetName string `json:"letterhead_asset_name"`

	// LetterheadFile is the filename of the letterhead artwork on disk
	// (e.g. "Acme Instrumentation Letterhead.png").
	LetterheadFile string `json:"letterhead_file"`

	// Aliases are extra lowercase spellings that normalise to this division's Key.
	// E.g. ["beacon controls wll", "beacon controls w.l.l"] all map to "Beacon Controls".
	Aliases []string `json:"aliases"`

	// DocumentDisplayName is the exact string historically printed on exported
	// documents (PDF/costing sheets) for this division. It is deliberately
	// separate from LegalName: LegalName is the full registered legal name
	// (e.g. "BEACON CONTROLS W.L.L."), while documents have historically
	// printed a shorter trading label (e.g. "Beacon Controls WLL") that does
	// not byte-match LegalName's casing/punctuation. Optional: when blank,
	// DivisionDocumentDisplayName falls back to CompanyDisplayName (for the
	// default division) or LegalName, so a partial overlay still renders.
	DocumentDisplayName string `json:"document_display_name"`

	// DashboardVariant is an optional per-division dashboard/audited-data
	// variant key (e.g. "ahs") that selects an alternate dashboard data path
	// for this division. Blank means the standard/default dashboard path.
	DashboardVariant string `json:"dashboard_variant"`
}

// CompanyOverlay holds the full company/division configuration.
// A zero SchemaVersion means "unversioned" (treated as version 1).
type CompanyOverlay struct {
	SchemaVersion int `json:"schema_version"`

	// Deployment carries deployment-layout identity — currently the slug that
	// keys the three-plane directory namespace (%APPDATA%\Asymmetrica\<slug>\...).
	// A separate exe-adjacent deployment.json (written by the installer) is the
	// authoritative source; this overlay field is the secondary fallback for
	// single-config / dev deployments. See pkg/infra/deploy.
	Deployment DeploymentConfig `json:"deployment"`

	// DefaultDivisionKey is the Key of the division returned when no match is found.
	DefaultDivisionKey string `json:"default_division_key"`

	// CompanyDisplayName is the human-readable trading name of the holding company.
	CompanyDisplayName string `json:"company_display_name"`

	Industry string `json:"industry"`
	Country  string `json:"country"`
	Currency string `json:"currency"`

	// Jurisdiction is the ISO-3166 alpha-2 tax-jurisdiction code ("BH", "SA",
	// "IN") that routes invoices to the matching pkg/compliance engine. Optional:
	// when blank, JurisdictionCode() derives it from Country by name, and event
	// consumers fall back to currency inference. Explicit config wins.
	Jurisdiction string `json:"jurisdiction"`

	// CurrencyDecimals is the number of decimal places for the currency (BHD = 3).
	CurrencyDecimals int `json:"currency_decimals"`

	// DefaultVATRate is the default VAT/tax rate percentage (e.g. 10.0 = 10%).
	DefaultVATRate float64 `json:"default_vat_rate"`

	// ExchangeRatesToBase maps an uppercase ISO currency code to the rate that
	// converts ONE unit of that currency into the base Currency (e.g. EUR→BHD).
	// The base currency itself is always 1.0 and need not be listed. These rates
	// are the SINGLE source of truth for FX conversion, shared by every code path
	// — import-time parsers AND live costing/posting — so the two can never
	// diverge. They are configuration (overlay.json), not hardcoded constants.
	ExchangeRatesToBase map[string]float64 `json:"exchange_rates_to_base"`

	Divisions []DivisionProfile `json:"divisions"`

	// BusinessRules holds the company's costing/approval policy numbers
	// (margin floors, grade discounts, payment terms). See business_rules.go.
	BusinessRules BusinessRules `json:"business_rules"`

	// ProductMarkupRules maps product types to standard margins (fractions);
	// types not listed fall back to DefaultProductMargin.
	ProductMarkupRules []ProductMarkupRule `json:"product_markup_rules"`

	// DefaultProductMargin is the fallback margin (fraction) for product types
	// without a specific markup rule. (0.12 = 12%)
	DefaultProductMargin float64 `json:"default_product_margin"`

	// SupplierAliases is the commercial supplier vocabulary handed to the
	// pkg/crm/supplierlink resolution engine (variant code spellings, brand
	// names). Company-specific catalogues belong here, never in engine code.
	// Semantics mirror SeedSets: a nil value (field absent from overlay.json)
	// keeps the built-in default vocabulary; an explicit empty object clears
	// it. See SupplierAliasVocabulary.
	SupplierAliases *SupplierAliasConfig `json:"supplier_aliases"`

	// LicenseKeyPrefix is the leading token of license keys
	// ({PREFIX}-{ROLE}-{6-char}). Blank falls back to the built-in "PH" so
	// existing activations keep validating. See LicenseKeyPrefixOrDefault.
	LicenseKeyPrefix string `json:"license_key_prefix"`

	// SeedSets selects which named seed bundles the vertical applies at boot
	// (license keys, RBAC roles, demo rows, …). Semantics via SeedEnabled:
	//   absent/null  → every seed the vertical historically ran (back-compat:
	//                  existing deployments keep today's behavior exactly)
	//   []           → no optional seeding at all
	//   ["a","b"]    → exactly the named bundles ("all" enables everything)
	// Seed-set names are defined by the vertical (see trading_models.go /
	// startup() call sites for the trading names).
	SeedSets []string `json:"seed_sets"`

	// SignatureBlocks are the named "Best Regards" identity blocks printed on
	// outbound documents (quotations/offers, credit notes). The staff identity
	// (name, title, phones, email) is deployment configuration, never source:
	// the repo ships SYNTHETIC identities and a sovereign overlay.json supplies
	// the real ones. Resolution matches a document's prepared-by/issuer string
	// against each block's DisplayName or Aliases; see SignatureBlockFor.
	SignatureBlocks []SignatureBlockProfile `json:"signature_blocks"`

	// SignatureDefault is the fallback signature block used when a prepared-by
	// name matches no SignatureBlocks entry — it carries the company-level
	// contact facts (company line, address, office/fax/email) onto which the
	// unmatched signer's own name is stamped as DisplayName. Optional: a nil
	// value derives a minimal fallback from the default division; see
	// SignatureFallback.
	SignatureDefault *SignatureBlockProfile `json:"signature_default"`
}

// DeploymentConfig holds deployment-layout identity read from the overlay.
type DeploymentConfig struct {
	// Slug keys the three-plane directory layout; blank falls back to the
	// built-in "AsymmFlow-Dev" via DeploymentSlug().
	Slug string `json:"slug"`
}

// SignatureBlockProfile is one "Best Regards" identity printed on documents.
// It mirrors the layout fields the PDF pipelines render, in order: a bold
// DisplayName, then Title, Company, each AddressLine, then Mobile/Office/Fax
// (each prefixed) and Email. Empty fields are skipped by the renderers.
type SignatureBlockProfile struct {
	DisplayName  string   `json:"display_name"`
	Title        string   `json:"title"`
	Company      string   `json:"company"`
	AddressLines []string `json:"address_lines"`
	Mobile       string   `json:"mobile"`
	Office       string   `json:"office"`
	Fax          string   `json:"fax"`
	Email        string   `json:"email"`
	// Aliases are extra spellings (nicknames, full names, role words) that
	// resolve to this block. Matching is case-insensitive and ignores all
	// non-alphanumeric characters (see normalizeSignatureKey).
	Aliases []string `json:"aliases"`
}

// normalizeSignatureKey lowercases value and strips every non-alphanumeric
// rune, so "V.M. Sundar", "vm sundar", and "VM  Sundar" all collapse to one
// key. Mirrors the deployed PH matcher exactly.
func normalizeSignatureKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SignatureBlockFor returns the configured signature block whose DisplayName
// or one of whose Aliases matches name (normalised), and true. An empty name
// or no match returns the zero block and false.
func (o *CompanyOverlay) SignatureBlockFor(name string) (SignatureBlockProfile, bool) {
	key := normalizeSignatureKey(name)
	if key == "" {
		return SignatureBlockProfile{}, false
	}
	for _, block := range o.SignatureBlocks {
		if normalizeSignatureKey(block.DisplayName) == key {
			return block, true
		}
		for _, alias := range block.Aliases {
			if normalizeSignatureKey(alias) == key {
				return block, true
			}
		}
	}
	return SignatureBlockProfile{}, false
}

// SignatureNames returns the DisplayName of every configured signature block,
// in declaration order. Used to seed collaboration/assignment pickers.
func (o *CompanyOverlay) SignatureNames() []string {
	names := make([]string, 0, len(o.SignatureBlocks))
	for _, block := range o.SignatureBlocks {
		names = append(names, block.DisplayName)
	}
	return names
}

// SignatureFallback returns the company-level fallback block for prepared-by
// names that match no SignatureBlocks entry. It returns SignatureDefault when
// configured; otherwise it derives a minimal block (company line + address)
// from the default division so a partial overlay.json still renders something
// coherent. The returned block's DisplayName is left blank for the caller to
// stamp with the actual signer's name.
func (o *CompanyOverlay) SignatureFallback() SignatureBlockProfile {
	if o.SignatureDefault != nil {
		return *o.SignatureDefault
	}
	div := o.Profile(o.DefaultDivisionKey)
	company := div.LegalName
	if strings.TrimSpace(company) == "" {
		company = o.CompanyDisplayName
	}
	return SignatureBlockProfile{
		Company:      company,
		AddressLines: div.AddressLines,
	}
}

// SupplierAliasConfig is the supplier commercial vocabulary. It deliberately
// mirrors pkg/crm/supplierlink.AliasConfig without importing it — overlay
// stays dependency-free and the vertical converts at the seam.
type SupplierAliasConfig struct {
	// CanonicalCodes maps a variant supplier-code spelling (uppercase) to the
	// canonical code stored on the supplier row, e.g. "SVX" -> "SRVX".
	CanonicalCodes map[string]string `json:"canonical_codes"`
	// BrandAliases maps an uppercased commercial token (a brand name) to the
	// supplier search terms it implies.
	BrandAliases map[string][]string `json:"brand_aliases"`
}

// SupplierAliasVocabulary returns the effective supplier vocabulary with all
// keys normalised to uppercase. A nil SupplierAliases falls back to the
// built-in default (the one synthetic alias in the seed canon); an explicit
// empty object in overlay.json yields an empty vocabulary.
func (o *CompanyOverlay) SupplierAliasVocabulary() SupplierAliasConfig {
	src := o.SupplierAliases
	if src == nil {
		src = BuiltinDefaults().SupplierAliases
	}
	out := SupplierAliasConfig{
		CanonicalCodes: map[string]string{},
		BrandAliases:   map[string][]string{},
	}
	for code, canonical := range src.CanonicalCodes {
		out.CanonicalCodes[strings.ToUpper(strings.TrimSpace(code))] = strings.ToUpper(strings.TrimSpace(canonical))
	}
	for token, terms := range src.BrandAliases {
		out.BrandAliases[strings.ToUpper(strings.TrimSpace(token))] = terms
	}
	return out
}

// BuiltinDefaults returns the overlay that exactly reproduces the hardcoded
// switch statements in company_branding.go. All synthetic demo values are
// reproduced verbatim.
func BuiltinDefaults() *CompanyOverlay {
	return &CompanyOverlay{
		SchemaVersion:      1,
		DefaultDivisionKey: "Acme Instrumentation",
		CompanyDisplayName: "Acme Instrumentation WLL",
		Industry:           "Process instrumentation",
		Country:            "Bahrain",
		Currency:           "BHD",
		CurrencyDecimals:   3,
		DefaultVATRate:     10.0,
		// FX rates → BHD. The canonical live values (app_sales_pipeline.go's
		// defaultExchangeRateToBHD switch). EUR = 0.45 is canonical: the old
		// eh_parser EUR_TO_BHD = 0.41 was a stale duplicate that the live path
		// already blacklisted/overwrote; config-driving collapses both to 0.45.
		ExchangeRatesToBase: map[string]float64{
			"EUR": 0.45,
			"USD": 0.376,
			"GBP": 0.52,
			"CHF": 0.425,
			"SAR": 0.100,
			"AED": 0.102,
		},
		Divisions: []DivisionProfile{
			{
				Key:                 "Acme Instrumentation",
				LegalName:           "ACME INSTRUMENTATION W.L.L",
				VATNumber:           "990000000000000",
				City:                "Manama",
				AddressLines:        []string{"PO Box 0000, Building 198", "Road 2803, Block 428", "Kingdom of Bahrain"},
				BankDetails:         []string{},
				LetterheadAssetName: "letterhead",
				LetterheadFile:      "Acme Instrumentation Letterhead.png",
				Aliases:             []string{},
				DocumentDisplayName: "Acme Instrumentation WLL",
				DashboardVariant:    "",
			},
			{
				Key:                 "Beacon Controls",
				LegalName:           "BEACON CONTROLS W.L.L.",
				VATNumber:           "990000000000001",
				City:                "Manama",
				AddressLines:        []string{"PO Box 0000, Manama", "Kingdom of Bahrain"},
				BankDetails:         []string{"1. Demo Bank B, A/c No: 00DEMO0000000002, IBAN: BH29BECN00000000000000, SWIFT: BECNBHBMXXX"},
				LetterheadAssetName: "letterhead_ahs",
				LetterheadFile:      "Beacon Controls Letterhead.jpg",
				Aliases: []string{
					"beacon controls wll",
					"beacon controls w.l.l",
					"beacon controls w.l.l.",
				},
				DocumentDisplayName: "Beacon Controls WLL",
				DashboardVariant:    "ahs",
			},
		},
		// BusinessRules reproduce the hardcoded thresholds from
		// business_invariants.go / costing_engine.go / app_costing_exports_surface.go
		// / geometry_bridge.go byte-identically.
		BusinessRules: BusinessRules{
			MinMarginPct:            0.08,  // business_invariants.go:424,726; costing_engine.go:236,405; app_costing_exports_surface.go:155
			ABBCompetitionMinMargin: 0.15,  // business_invariants.go:445,741; costing_engine.go:415; app_costing_exports_surface.go:143,166; geometry_bridge.go:753
			EmergencyMinMarginPct:   0.20,  // business_invariants.go:466
			ApprovalThresholdMargin: 0.20,  // app_costing_exports_surface.go:132
			LargeOrderThresholdBHD:  10000, // app_costing_exports_surface.go:172
			MonthlyOperatingCostBHD: 15000, // business_invariants.go:488
			NamedCompetitors:        []string{"ABB"},
			GradePaymentTerms: map[string]GradePolicy{
				// Terms/AdvancePct from costing_engine.GetPaymentTerms; MaxDiscount
				// from GetCustomerDiscount; MaxDays from the business_invariants
				// payment-term ceilings (GradeX_PaymentTerms).
				"A": {Terms: "Net 45 days", AdvancePct: 0.0, MaxDiscount: 0.07, MaxDays: 55},
				"B": {Terms: "Net 90 days", AdvancePct: 0.0, MaxDiscount: 0.03, MaxDays: 100},
				"C": {Terms: "Net 120 days with 50% advance", AdvancePct: 0.50, MaxDiscount: 0.00, MaxDays: 130},
				"D": {Terms: "100% advance or DECLINE", AdvancePct: 1.00, MaxDiscount: 0.00, MaxDays: 0},
			},
		},
		// ProductMarkupRules reproduce the costing_engine.ProductMarkupRules map.
		ProductMarkupRules: []ProductMarkupRule{
			{ProductType: "Rhine Flow", Margin: 0.15},
			{ProductType: "Rhine Level", Margin: 0.18},
			{ProductType: "Rhine Instruments Pressure", Margin: 0.18},
			{ProductType: "Rhine Instruments Temperature", Margin: 0.15},
			{ProductType: "Rhine Analytics", Margin: 0.20},
			{ProductType: "Rhine Instruments General", Margin: 0.12},
			{ProductType: "Oxan Analytics", Margin: 0.25},
			{ProductType: "GIC", Margin: 0.10},
		},
		DefaultProductMargin: 0.12, // costing_engine.go:116
		// SupplierAliases carries only the one alias in the synthetic seed
		// canon (product seeds use code SVX, the seeded supplier row SRVX).
		// Real principal alias catalogues are sovereign-overlay facts.
		SupplierAliases: &SupplierAliasConfig{
			CanonicalCodes: map[string]string{"SVX": "SRVX"},
		},
		// LicenseKeyPrefix reproduces the hardcoded "PH-{ROLE}-{6}" key format
		// byte-identically; existing activations must keep validating.
		LicenseKeyPrefix: "PH",
		// SignatureBlocks ship SYNTHETIC identities only (SYNTHETIC_IDENTITY.md:
		// first-name people, obviously-fake +973-1700 phones, *.example emails,
		// the Acme legal name and PO Box 0000 address). Real staff blocks are a
		// sovereign-overlay fact supplied at deploy time. One block (Casey Quinn)
		// deliberately omits a Title to exercise the skip-empty-title layout path.
		SignatureBlocks: []SignatureBlockProfile{
			{
				DisplayName:  "Jordan Avery",
				Title:        "Technical Sales Engineer",
				Company:      "ACME INSTRUMENTATION W.L.L",
				AddressLines: []string{"PO Box 0000, Manama,", "Kingdom of Bahrain"},
				Mobile:       "+973-1700-0010",
				Office:       "+973-1700-0000",
				Fax:          "+973-1700-0001",
				Email:        "jordan@acme-instrumentation.example",
				Aliases:      []string{"Jordan", "Jordan Avery"},
			},
			{
				DisplayName:  "Alex Morgan",
				Title:        "Assistant Sales Manager",
				Company:      "ACME INSTRUMENTATION W.L.L",
				AddressLines: []string{"PO Box 0000, Manama,", "Kingdom of Bahrain"},
				Mobile:       "+973-1700-0011",
				Office:       "+973-1700-0000",
				Fax:          "+973-1700-0001",
				Email:        "alex@acme-instrumentation.example",
				Aliases:      []string{"Alex", "Alex Morgan"},
			},
			{
				DisplayName:  "Sam Rivera",
				Title:        "Business Development Manager",
				Company:      "ACME INSTRUMENTATION W.L.L",
				AddressLines: []string{"PO Box 0000, Manama,", "Kingdom of Bahrain"},
				Mobile:       "+973-1700-0012",
				Office:       "+973-1700-0000",
				Fax:          "+973-1700-0001",
				Email:        "sam@acme-instrumentation.example",
				Aliases:      []string{"Sam", "Sam Rivera"},
			},
			{
				DisplayName:  "Casey Quinn",
				Company:      "ACME INSTRUMENTATION W.L.L",
				AddressLines: []string{"PO Box 0000, Manama,", "Kingdom of Bahrain"},
				Mobile:       "+973-1700-0013",
				Office:       "+973-1700-0000",
				Fax:          "+973-1700-0001",
				Email:        "support@acme-instrumentation.example",
				Aliases:      []string{"Casey", "Casey Quinn", "Support"},
			},
			{
				DisplayName:  "Taylor Brooks",
				Title:        "Chief Operating Officer",
				Company:      "ACME INSTRUMENTATION W.L.L",
				AddressLines: []string{"PO Box 0000, Manama,", "Kingdom of Bahrain"},
				Mobile:       "+973-1700-0014",
				Office:       "+973-1700-0000",
				Fax:          "+973-1700-0001",
				Email:        "taylor@acme-instrumentation.example",
				Aliases:      []string{"Taylor", "Taylor Brooks"},
			},
			{
				DisplayName:  "Jamie Ellis",
				Title:        "General Manager",
				Company:      "ACME INSTRUMENTATION W.L.L",
				AddressLines: []string{"PO Box 0000, Manama,", "Kingdom of Bahrain"},
				Mobile:       "+973-1700-0015",
				Office:       "+973-1700-0000",
				Fax:          "+973-1700-0001",
				Email:        "jamie@acme-instrumentation.example",
				Aliases:      []string{"Jamie", "Jamie Ellis"},
			},
		},
		// SignatureDefault is the synthetic company-level fallback for unknown
		// signers; the unmatched signer's own name is stamped onto DisplayName.
		SignatureDefault: &SignatureBlockProfile{
			Company:      "ACME INSTRUMENTATION W.L.L",
			AddressLines: []string{"PO Box 0000, Manama,", "Kingdom of Bahrain"},
			Office:       "+973-1700-0000",
			Fax:          "+973-1700-0001",
			Email:        "sales@acme-instrumentation.example",
		},
	}
}

// LicenseKeyPrefixOrDefault returns the license-key prefix ("PH" in the
// built-in format PH-{ROLE}-{6}). A blank value in overlay.json falls back to
// the built-in default so existing activations never break on a partial file.
func (o *CompanyOverlay) LicenseKeyPrefixOrDefault() string {
	if p := strings.ToUpper(strings.TrimSpace(o.LicenseKeyPrefix)); p != "" {
		return p
	}
	return "PH"
}

// DeploymentSlug returns the deployment slug that keys the three-plane
// directory layout. A blank slug in overlay.json falls back to the built-in
// "AsymmFlow-Dev" default so a partial or absent overlay never yields an empty
// namespace. Note: an exe-adjacent deployment.json (installer-written) takes
// precedence over this value — see pkg/infra/deploy.DeploymentSlug.
func (o *CompanyOverlay) DeploymentSlug() string {
	if s := strings.TrimSpace(o.Deployment.Slug); s != "" {
		return s
	}
	return "AsymmFlow-Dev"
}

// NormalizeDivisionName maps a raw division string (as stored in the DB or
// passed by the user) to the canonical Key of a known division. Matching is
// case-insensitive and whitespace-trimmed. Unknown strings fall back to the
// DefaultDivisionKey.
//
// This reproduces the original normalizeDivisionName switch exactly:
//
//	"beacon controls" / "beacon controls wll" / "beacon controls w.l.l" /
//	"beacon controls w.l.l." → "Beacon Controls"
//	anything else            → "Acme Instrumentation"
func (o *CompanyOverlay) NormalizeDivisionName(raw string) string {
	needle := strings.TrimSpace(strings.ToLower(raw))
	for _, div := range o.Divisions {
		// Match against the division's own Key (lowercased).
		if strings.ToLower(div.Key) == needle {
			return div.Key
		}
		// Match against any declared alias.
		for _, alias := range div.Aliases {
			if alias == needle {
				return div.Key
			}
		}
	}
	return o.DefaultDivisionKey
}

// DivisionDocumentDisplayName returns the document-display string for the
// normalized division key, falling back to CompanyDisplayName (default
// division) or LegalName when the profile leaves it blank, so partial
// overlays still render.
func (o *CompanyOverlay) DivisionDocumentDisplayName(key string) string {
	div := o.Profile(key)
	if strings.TrimSpace(div.DocumentDisplayName) != "" {
		return div.DocumentDisplayName
	}
	if div.Key == o.DefaultDivisionKey && strings.TrimSpace(o.CompanyDisplayName) != "" {
		return o.CompanyDisplayName
	}
	return div.LegalName
}

// IsKnownDivision reports whether raw matches a division Key or alias
// (case-insensitive, trimmed). Unlike NormalizeDivisionName it does NOT
// fall back to the default — it returns false for genuinely unknown values.
func (o *CompanyOverlay) IsKnownDivision(raw string) bool {
	needle := strings.TrimSpace(strings.ToLower(raw))
	if needle == "" {
		return false
	}
	for _, div := range o.Divisions {
		if strings.ToLower(div.Key) == needle {
			return true
		}
		for _, alias := range div.Aliases {
			if alias == needle {
				return true
			}
		}
	}
	return false
}

// Profile returns the DivisionProfile for the given key (already normalised).
// Falls back to the default division if no match is found.
func (o *CompanyOverlay) Profile(key string) DivisionProfile {
	for _, div := range o.Divisions {
		if div.Key == key {
			return div
		}
	}
	// Fallback: return the default division.
	for _, div := range o.Divisions {
		if div.Key == o.DefaultDivisionKey {
			return div
		}
	}
	// Should never happen with well-formed defaults; return zero value.
	return DivisionProfile{}
}

// DefaultDivision returns the DefaultDivisionKey.
func (o *CompanyOverlay) DefaultDivision() string {
	return o.DefaultDivisionKey
}

// ExchangeRateToBase returns the rate that converts one unit of the given
// currency into the company's base Currency (e.g. EUR→BHD). The base currency
// and the empty string return 1.0; unknown currencies also return 1.0 (matching
// the historic defaultExchangeRateToBHD default case). Rates come from
// ExchangeRatesToBase, so import-time and live conversion paths share one
// configurable source of truth.
func (o *CompanyOverlay) ExchangeRateToBase(currency string) float64 {
	c := strings.ToUpper(strings.TrimSpace(currency))
	if c == "" || c == strings.ToUpper(strings.TrimSpace(o.Currency)) {
		return 1.0
	}
	if r, ok := o.ExchangeRatesToBase[c]; ok {
		return r
	}
	return 1.0
}

// JurisdictionCode returns the ISO-3166 alpha-2 tax-jurisdiction code for
// this deployment, used to route invoices to a pkg/compliance engine.
// Precedence: explicit Jurisdiction config → country-name lookup → "" (callers
// fall back to currency inference, preserving pre-overlay behaviour).
func (o *CompanyOverlay) JurisdictionCode() string {
	if j := strings.ToUpper(strings.TrimSpace(o.Jurisdiction)); j != "" {
		return j
	}
	switch strings.ToLower(strings.TrimSpace(o.Country)) {
	case "bahrain", "kingdom of bahrain":
		return "BH"
	case "saudi arabia", "kingdom of saudi arabia", "ksa":
		return "SA"
	case "india":
		return "IN"
	case "united arab emirates", "uae":
		return "AE"
	case "oman", "sultanate of oman":
		return "OM"
	case "qatar":
		return "QA"
	case "kuwait":
		return "KW"
	default:
		return ""
	}
}

// SeedEnabled reports whether the named seed bundle should run for this
// deployment. A nil SeedSets (field absent from overlay.json) enables every
// bundle — the pre-overlay behavior — so existing deployments are unaffected;
// an explicit empty array disables all optional seeding; otherwise only the
// listed bundles (or the sentinel "all") run.
func (o *CompanyOverlay) SeedEnabled(name string) bool {
	if o.SeedSets == nil {
		return true
	}
	for _, s := range o.SeedSets {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "all" || s == strings.ToLower(name) {
			return true
		}
	}
	return false
}

// sqlQuote returns s as a single-quoted, escaped SQL string literal
// (e.g. Acme's → 'Acme''s').
func sqlQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// DivisionNormalizationCase returns a SQL CASE expression that maps the value
// of columnExpr (e.g. "orders.division") to its canonical division Key, built
// from the overlay's divisions and their aliases. Each non-default division
// gets a WHEN that matches LOWER(TRIM(COALESCE(col, ''))) against its
// lowercased Key plus declared aliases; everything else falls through to the
// DefaultDivisionKey (the ELSE branch).
//
// This makes the migration/backfill DDL config-driven: a different vertical's
// divisions and aliases flow into the generated SQL automatically instead of
// being hardcoded in 15+ inline IN-lists. The matching mirrors
// NormalizeDivisionName (exact, case-insensitive Key/alias match), so the SQL
// backfills and the Go normaliser agree by construction.
func (o *CompanyOverlay) DivisionNormalizationCase(columnExpr string) string {
	var b strings.Builder
	b.WriteString("CASE\n")
	for _, div := range o.Divisions {
		if div.Key == o.DefaultDivisionKey {
			continue // the default division is the ELSE branch
		}
		// IN-list = the division's lowercased Key followed by its aliases.
		needles := make([]string, 0, len(div.Aliases)+1)
		needles = append(needles, sqlQuote(strings.ToLower(div.Key)))
		for _, alias := range div.Aliases {
			needles = append(needles, sqlQuote(alias))
		}
		fmt.Fprintf(&b, "\t\t\tWHEN LOWER(TRIM(COALESCE(%s, ''))) IN (%s) THEN %s\n",
			columnExpr, strings.Join(needles, ", "), sqlQuote(div.Key))
	}
	fmt.Fprintf(&b, "\t\t\tELSE %s\n\t\tEND", sqlQuote(o.DefaultDivisionKey))
	return b.String()
}

// LoadOverlay tries to read overlay.json from each directory in searchDirs in
// order. The first valid parse wins. If none is found or all parse attempts
// fail, BuiltinDefaults() is returned. LoadOverlay never returns nil.
//
// Merge semantics: the file is parsed directly. A future version may merge
// over BuiltinDefaults for partial overrides; for now the file must be
// complete if provided.
func LoadOverlay(searchDirs []string) *CompanyOverlay {
	for _, dir := range searchDirs {
		candidate := filepath.Join(dir, "overlay.json")
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue // Not found in this dir — try next.
		}
		var o CompanyOverlay
		if err := json.Unmarshal(data, &o); err != nil {
			fmt.Printf("[overlay] WARNING: found %s but could not parse it (%v) — using built-in defaults\n", candidate, err)
			continue
		}
		if len(o.Divisions) == 0 {
			fmt.Printf("[overlay] WARNING: %s has no divisions — using built-in defaults\n", candidate)
			continue
		}
		fmt.Printf("[overlay] Loaded company overlay from: %s\n", candidate)
		return &o
	}
	fmt.Println("[overlay] No overlay.json found — using built-in defaults")
	return BuiltinDefaults()
}
