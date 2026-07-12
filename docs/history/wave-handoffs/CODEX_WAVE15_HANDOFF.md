# CODEX WAVE 15 HANDOFF — i18n + Compliance

**Project**: AsymmFlow (asymmflow)
**Wave**: 15 of 16
**Depends On**: Wave 14 (complete — commit `11002ac`)
**Module**: `ph_holdings_app`

---

## RULES (READ FIRST)

1. **`go build ./...` and `go test ./... -count=1 -timeout 300s` MUST pass after EVERY Go ticket.**
2. **`cd frontend && npm run build` MUST pass after EVERY frontend ticket.**
3. **Commit after EACH ticket** with message format: `feat(codex): <description> (Wave 15, Ticket N)`
4. **NO behavior changes to existing functionality.** i18n wraps existing strings. Compliance modules are ADDITIVE — they don't modify existing finance/CRM logic.
5. **STOP CONDITIONS**: If `go build` fails after 3 attempts, stop. If a compliance calculation produces incorrect results against known test cases, stop.
6. **ELEVATED READ ACCESS**: You can read files across `C:\Projects\` for reference patterns.
7. **DO NOT modify Wave 1-14 domain packages** (`pkg/finance/`, `pkg/crm/`, etc.) unless a ticket explicitly says to add an interface or hook.

---

## CONTEXT: What Exists After Wave 14

### Architecture Summary

```
main.go               → Wails v2, 7 bound structs (App + 6 domain services)
service_finance.go     → FinanceService (241 methods)
service_crm.go         → CRMService (228 methods)
service_infra.go       → InfraService (182 methods)
service_documents.go   → DocumentsService (99 methods)
service_sync.go        → SyncServiceBinding (54 methods)
service_butler.go      → ButlerService (35 methods)
internal/viewmodel/    → 76 ViewModel types, 7 builders
pkg/math/              → Full substrate (quaternion, vedic, trident, prism, conversation, encoding)
pkg/infra/otel/        → OTel provider with no-op mode
pkg/infra/health/      → Three-regime health monitor
pkg/sync/turso/        → Turso client + CDC logger
frontend/              → Svelte 5.55.5, 186 components, Tailwind v4
```

### Current Language/Locale Support
- **Backend**: English-only strings hardcoded in service files, PDF generators, error messages
- **Frontend**: English-only strings in Svelte components
- **Compliance**: Bahrain VAT (10%) hardcoded in finance calculations. No other tax jurisdictions.

### Target Markets (from MASTER_PLAN)
- **Primary**: Bahrain (current deployment), India (expanding)
- **Secondary**: Nigeria, Ghana, Kenya, Indonesia, Philippines
- **All**: Offline-first, emerging market compatible

---

## WAVE 15 STRATEGY

Two parallel tracks:

**Track A: Internationalization (i18n)** — Make ALL user-facing strings translatable without changing business logic. Start with 5 languages (English, Arabic, Hindi, French, Spanish).

**Track B: Compliance Modules** — Pluggable tax/regulatory engines that hook into existing finance events. Start with 3 jurisdictions (Bahrain VAT, India GST, India Income Tax).

---

## WAVE 15 TICKETS

### Ticket 1: i18n Infrastructure

**Files**: Create `pkg/i18n/` package

```go
package i18n

// Locale represents a supported language.
type Locale string

const (
    EN Locale = "en"  // English (default)
    AR Locale = "ar"  // Arabic
    HI Locale = "hi"  // Hindi
    FR Locale = "fr"  // French
    ES Locale = "es"  // Spanish
)

// Translator provides string localization.
type Translator struct {
    locale     Locale
    messages   map[string]string  // key → translated string
    fallback   *Translator        // fallback to English if key missing
}

// New creates a translator for the given locale.
func New(locale Locale) *Translator

// T returns the translated string for a key.
// Falls back to English if not found in current locale.
// Falls back to the key itself if not found in any locale.
func (t *Translator) T(key string) string

// Tf returns a formatted translated string (like fmt.Sprintf).
func (t *Translator) Tf(key string, args ...any) string

// SetLocale changes the active locale.
func (t *Translator) SetLocale(locale Locale)

// Locale returns the current locale.
func (t *Translator) CurrentLocale() Locale

// AvailableLocales returns all supported locales.
func AvailableLocales() []Locale

// LoadMessages loads translation messages from a JSON file.
func LoadMessages(locale Locale, path string) (map[string]string, error)

// LoadEmbedded loads translation messages from embedded resources.
func LoadEmbedded(locale Locale) (map[string]string, error)
```

**Message storage**: JSON files in `pkg/i18n/messages/`
```
pkg/i18n/messages/
  en.json
  ar.json
  hi.json
  fr.json
  es.json
```

**Message file format** (en.json):
```json
{
  "app.title": "AsymmFlow",
  "nav.dashboard": "Dashboard",
  "nav.finance": "Finance",
  "nav.crm": "CRM",
  "nav.documents": "Documents",
  "nav.butler": "Butler AI",
  "nav.settings": "Settings",
  "finance.invoice.title": "Invoices",
  "finance.invoice.create": "Create Invoice",
  "finance.invoice.total": "Total",
  "finance.payment.title": "Payments",
  "finance.payment.record": "Record Payment",
  "common.save": "Save",
  "common.cancel": "Cancel",
  "common.delete": "Delete",
  "common.search": "Search",
  "common.loading": "Loading...",
  "common.error": "An error occurred",
  "common.success": "Success",
  "compliance.vat": "VAT",
  "compliance.gst": "GST",
  "compliance.tax_invoice": "Tax Invoice"
}
```

**For Arabic (ar.json)**: Include RTL indicator. Key translations for finance/CRM terms.

**For Hindi (hi.json)**: Devanagari translations for all UI strings.

Create at least 50 message keys covering: navigation, finance terms, CRM terms, common actions, compliance terms, error messages.

**Embed the JSON files** using `//go:embed messages/*.json` so they're bundled in the binary.

**Test file**: `pkg/i18n/i18n_test.go`
Tests (minimum 5):
1. English translation returns correct string
2. Missing key falls back to English
3. Missing key in all locales returns the key itself
4. Tf formats correctly with arguments
5. SetLocale changes active locale

---

### Ticket 2: Frontend i18n Integration

**Files**: Create `frontend/src/lib/i18n/` directory

Create a Svelte 5 i18n store using runes:

```typescript
// frontend/src/lib/i18n/index.ts

export type Locale = 'en' | 'ar' | 'hi' | 'fr' | 'es';

// Messages loaded from backend
let messages = $state<Record<string, string>>({});
let currentLocale = $state<Locale>('en');

// Translation function
export function t(key: string, ...args: any[]): string {
    const template = messages[key] || key;
    // Simple sprintf-style replacement: {0}, {1}, etc.
    return template.replace(/\{(\d+)\}/g, (_, i) => args[parseInt(i)] ?? '');
}

// Set locale (fetches messages from backend)
export async function setLocale(locale: Locale): Promise<void> {
    const response = await GetTranslations(locale);
    messages = response;
    currentLocale = locale;
    // Set dir="rtl" for Arabic
    document.documentElement.dir = locale === 'ar' ? 'rtl' : 'ltr';
}
```

**Backend endpoint**: Add `GetTranslations(locale string) map[string]string` method on InfraService that returns the message map for the requested locale.

**RTL support**: When Arabic is selected, set `dir="rtl"` on the root HTML element. Tailwind v4 has built-in RTL utilities.

**DO NOT convert all 186 components to use i18n in this ticket.** Just set up the infrastructure and convert 5-10 key components as proof:
- Header/navigation
- Dashboard title
- One finance screen (invoice list)
- Settings page (locale selector)

---

### Ticket 3: Compliance Module Interface

**File**: Create `pkg/compliance/` package

```go
package compliance

import "time"

// Jurisdiction represents a tax/regulatory jurisdiction.
type Jurisdiction string

const (
    JurisdictionBahrain Jurisdiction = "BH"
    JurisdictionIndia   Jurisdiction = "IN"
)

// TaxEngine is the interface all compliance modules implement.
type TaxEngine interface {
    // Jurisdiction returns the jurisdiction code.
    Jurisdiction() Jurisdiction

    // CalculateTax computes tax on a transaction.
    CalculateTax(tx TaxableTransaction) (*TaxResult, error)

    // ValidateInvoice checks if an invoice meets jurisdiction requirements.
    ValidateInvoice(inv InvoiceData) (*ValidationResult, error)

    // TaxRates returns current tax rates for the jurisdiction.
    TaxRates() []TaxRate

    // Name returns the display name.
    Name() string
}

// TaxableTransaction represents a transaction that may be taxed.
type TaxableTransaction struct {
    Amount       float64
    Currency     string
    Date         time.Time
    Category     string    // goods, services, exempt
    CustomerType string    // registered, unregistered, export
    SupplierType string    // registered, unregistered, import
    HSNCode      string    // Harmonized System Nomenclature (India GST)
    PlaceOfSupply string   // State code (India GST: inter vs intra state)
}

// TaxResult holds computed tax amounts.
type TaxResult struct {
    BaseAmount   float64
    TaxAmount    float64
    TotalAmount  float64
    TaxBreakdown []TaxComponent
    Jurisdiction Jurisdiction
}

// TaxComponent is one line of tax (e.g., CGST 9%, SGST 9%).
type TaxComponent struct {
    Name    string  // "VAT", "CGST", "SGST", "IGST"
    Rate    float64 // 0.10 for 10%
    Amount  float64
}

// TaxRate represents a configured tax rate.
type TaxRate struct {
    Name        string
    Rate        float64
    Category    string // Which categories it applies to
    Description string
}

// InvoiceData holds invoice fields for validation.
type InvoiceData struct {
    InvoiceNumber string
    InvoiceDate   time.Time
    SellerTaxID   string
    BuyerTaxID    string
    Amount        float64
    TaxAmount     float64
    Currency      string
    LineItems     []LineItemData
}

// LineItemData holds one invoice line item.
type LineItemData struct {
    Description string
    Quantity    float64
    UnitPrice   float64
    TaxRate     float64
    HSNCode     string
}

// ValidationResult holds invoice validation outcome.
type ValidationResult struct {
    Valid    bool
    Errors   []string
    Warnings []string
}

// Registry holds all registered tax engines.
type Registry struct {
    engines map[Jurisdiction]TaxEngine
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry

// Register adds a tax engine.
func (r *Registry) Register(engine TaxEngine)

// Get returns the engine for a jurisdiction.
func (r *Registry) Get(j Jurisdiction) (TaxEngine, bool)

// All returns all registered engines.
func (r *Registry) All() []TaxEngine
```

**Test file**: `pkg/compliance/compliance_test.go`
Tests (minimum 3):
1. Registry stores and retrieves engines
2. TaxableTransaction serializes correctly
3. TaxResult computes total correctly

---

### Ticket 4: Bahrain VAT Module

**File**: `pkg/compliance/bahrain/vat.go`

```go
package bahrain

// BahrainVAT implements compliance.TaxEngine for Bahrain VAT.
type BahrainVAT struct{}

func New() *BahrainVAT

// Current Bahrain VAT: 10% standard rate (raised from 5% in Jan 2022)
// Exempt: basic food items, healthcare, education, financial services, real estate
```

**Implementation:**
- Standard rate: 10%
- Zero-rated: exports, international transport
- Exempt: basic food, healthcare, education, financial services (first supply), residential real estate
- Invoice validation: must have VAT registration number, tax amount, correct rate
- Currency: BHD (3 decimal places)

**Test file**: `pkg/compliance/bahrain/vat_test.go`
Tests (minimum 5):
1. Standard goods → 10% VAT
2. Exempt category → 0% VAT
3. Zero-rated export → 0% VAT with explicit zero-rate
4. Invoice validation: valid invoice passes
5. Invoice validation: missing tax ID fails

---

### Ticket 5: India GST Module

**File**: `pkg/compliance/india/gst.go`

```go
package india

// IndiaGST implements compliance.TaxEngine for India GST.
type IndiaGST struct{}

func NewGST() *IndiaGST
```

**Implementation:**
- 5 rate slabs: 0%, 5%, 12%, 18%, 28%
- Intra-state: CGST (half) + SGST (half)
- Inter-state: IGST (full rate)
- HSN code → rate mapping (simplified: provide top 20 common HSN codes)
- Threshold: ₹20L for goods, ₹20L for services (₹10L for special category states)
- Invoice validation: GSTIN format (15 chars), invoice series, HSN code presence

**Key HSN codes to include:**
| HSN | Description | Rate |
|-----|------------|------|
| 8481 | Valves, taps, cocks | 18% |
| 9032 | Automatic regulating instruments | 18% |
| 8536 | Electrical switching apparatus | 18% |
| 7304 | Tubes, pipes (iron/steel) | 18% |
| 4901 | Printed books, newspapers | 0% |
| 3004 | Medicaments | 12% |
| 1006 | Rice | 5% |

**Test file**: `pkg/compliance/india/gst_test.go`
Tests (minimum 6):
1. Intra-state: ₹1000 at 18% → CGST ₹90 + SGST ₹90
2. Inter-state: ₹1000 at 18% → IGST ₹180
3. Zero-rated goods (books) → 0%
4. HSN code lookup → correct rate
5. GSTIN validation: valid 15-char format passes
6. GSTIN validation: invalid format fails

---

### Ticket 6: India Income Tax Module

**File**: `pkg/compliance/india/income_tax.go`

```go
package india

// IndiaIncomeTax provides income tax calculations.
type IndiaIncomeTax struct{}

func NewIncomeTax() *IndiaIncomeTax

// CalculateOldRegime computes tax under the old regime with deductions.
func (t *IndiaIncomeTax) CalculateOldRegime(income float64, deductions Deductions) *IncomeTaxResult

// CalculateNewRegime computes tax under the new regime (no deductions, lower slabs).
func (t *IndiaIncomeTax) CalculateNewRegime(income float64) *IncomeTaxResult

// Compare returns both regimes with a recommendation.
func (t *IndiaIncomeTax) Compare(income float64, deductions Deductions) *RegimeComparison

// Deductions holds all claimed deductions.
type Deductions struct {
    Section80C    float64 // Limit: 1,50,000
    Section80CCD  float64 // NPS additional: 50,000
    Section80D    float64 // Health insurance
    Section80DSenior float64 // Senior citizen parents: 50,000
    Section80E    float64 // Education loan interest (no limit)
    Section80G    float64 // Donations
    Section24     float64 // Home loan interest: 2,00,000 (self-occupied)
    HRAExemption  float64 // Computed from rent paid, salary, metro/non-metro
    StandardDeduction float64 // 75,000 (FY 2025-26)
}

// IncomeTaxResult holds computed tax.
type IncomeTaxResult struct {
    TaxableIncome float64
    TaxAmount     float64
    Surcharge     float64 // If income > 50L
    Cess          float64 // 4% health & education cess
    TotalTax      float64
    EffectiveRate float64
    SlabBreakdown []SlabDetail
}

// SlabDetail shows tax for one income slab.
type SlabDetail struct {
    From   float64
    To     float64 // 0 = unlimited
    Rate   float64
    Tax    float64
}

// RegimeComparison shows both regimes side by side.
type RegimeComparison struct {
    OldRegime     *IncomeTaxResult
    NewRegime     *IncomeTaxResult
    Savings       float64 // Positive = old regime saves, negative = new regime saves
    Recommendation string  // "Old regime saves ₹X" or "New regime saves ₹X"
}
```

**Old regime slabs (FY 2025-26):**
| Income Range | Rate |
|-------------|------|
| 0 - 2.5L | 0% |
| 2.5L - 5L | 5% |
| 5L - 10L | 20% |
| Above 10L | 30% |

**New regime slabs (FY 2025-26):**
| Income Range | Rate |
|-------------|------|
| 0 - 4L | 0% |
| 4L - 8L | 5% |
| 8L - 12L | 10% |
| 12L - 16L | 15% |
| 16L - 20L | 20% |
| 20L - 24L | 25% |
| Above 24L | 30% |

**Additional:**
- Standard deduction: ₹75,000 (new regime), ₹50,000 (old regime)
- Section 87A rebate: if taxable income ≤ 7L (old) or ≤ 12L (new), tax = 0
- Health & education cess: 4% on tax + surcharge
- Surcharge: 10% if income > 50L, 15% if > 1Cr, 25% if > 2Cr

**Test file**: `pkg/compliance/india/income_tax_test.go`
Tests (minimum 6):
1. Income 5L, no deductions → 0 tax (both regimes, rebate applies)
2. Income 10L, old regime with 1.5L 80C + 50K 80D → compare with new regime
3. Income 20L, full deductions → old regime likely better
4. Income 8L, no deductions → new regime better
5. Surcharge kicks in above 50L
6. Cess calculated correctly (4%)

---

### Ticket 7: Compliance Hooks into Finance Events

**File**: Modify `pkg/infra/events/events.go` (add new event types) and create `pkg/compliance/hooks.go`

Add compliance event types:
```go
// New event types for compliance
const (
    EventInvoiceCreated   = "finance.invoice.created"
    EventInvoiceUpdated   = "finance.invoice.updated"
    EventPaymentRecorded  = "finance.payment.recorded"
    EventExpenseCreated   = "finance.expense.created"
)
```

Create compliance hooks:
```go
// pkg/compliance/hooks.go

// ComplianceHook listens for finance events and validates compliance.
type ComplianceHook struct {
    registry *Registry
    bus      *events.EventBus
}

// NewComplianceHook creates and registers compliance event listeners.
func NewComplianceHook(registry *Registry, bus *events.EventBus) *ComplianceHook

// OnInvoiceCreated validates invoice tax compliance.
func (h *ComplianceHook) OnInvoiceCreated(data map[string]any) error
```

**Implementation:**
- Listen for invoice/payment/expense events on the event bus
- Look up the jurisdiction from the transaction currency or company settings
- Run the appropriate TaxEngine.ValidateInvoice
- Log warnings/errors to OTel (if enabled)
- DO NOT block the event — validate asynchronously, report issues

**Test file**: `pkg/compliance/hooks_test.go`
Tests (minimum 3):
1. Invoice created event → Bahrain VAT validation triggered
2. Invalid invoice → validation errors logged
3. Unknown jurisdiction → gracefully skipped

---

### Ticket 8: Compliance Dashboard ViewModel

**File**: `internal/viewmodel/compliance_vm.go`

```go
// ComplianceDashboardVM shows compliance status for the active jurisdiction.
type ComplianceDashboardVM struct {
    Jurisdiction   string              `json:"jurisdiction"`
    TaxRates       []compliance.TaxRate `json:"tax_rates"`
    RecentValidations []ValidationEntry `json:"recent_validations"`
    ComplianceScore   float64          `json:"compliance_score"` // 0-100%
    Issues         []ComplianceIssue   `json:"issues"`
}

// TaxCalculatorVM for the tax calculation screen.
type TaxCalculatorVM struct {
    Jurisdictions  []string `json:"jurisdictions"`
    ActiveJurisdiction string `json:"active_jurisdiction"`
}
```

**Wire into InfraService**: Add `GetComplianceDashboard(jurisdiction string) ComplianceDashboardVM` method.

---

### Ticket 9: Full Build + Integration Verification

Verify the entire stack:
```powershell
go build ./...
go test ./... -count=1 -timeout 300s
cd frontend && npm run build && npm run check
wails build
```

Also verify:
- i18n: `GetTranslations("hi")` returns Hindi translations
- Bahrain VAT: `CalculateTax(1000 BHD goods)` → 100 BHD VAT
- India GST: `CalculateTax(1000 INR intra-state 18%)` → CGST 90 + SGST 90
- India IT: `Compare(10L income, 1.5L 80C)` → recommendation

---

### Ticket 10: Progress Audit

**File**: `docs/WAVE15_PROGRESS.md`

Write the progress audit with:
1. **Commit table**
2. **i18n metrics**: languages, message count, components converted
3. **Compliance modules**: jurisdictions, tax rate coverage, test cases
4. **Integration test results**
5. **Issues and deviations**
6. **Final gate**

Update `docs/MASTER_PLAN.md`: Mark Wave 15 as `✅ DONE`.

---

## DEPENDENCY GRAPH

```
Ticket 1 (i18n infra)          ─→ Ticket 2 (frontend i18n)
Ticket 3 (compliance interface) ─→ Ticket 4 (Bahrain VAT) ─→ Ticket 7 (event hooks)
Ticket 3                        ─→ Ticket 5 (India GST)   ─→ Ticket 7
Ticket 3                        ─→ Ticket 6 (India IT)
Ticket 3                        ─→ Ticket 8 (compliance VM)
All                             ─→ Ticket 9 (verification)
All                             ─→ Ticket 10 (audit)
```

**Recommended order**: 1 → 3 → 4 → 5 → 6 → 2 → 7 → 8 → 9 → 10

---

## QUALITY GATES

```powershell
go build ./...
go test ./... -count=1 -timeout 300s
cd frontend && npm run build && npm run check
```

---

## WHAT SUCCESS LOOKS LIKE

After Wave 15:
- **5 languages**: EN, AR, HI, FR, ES with infrastructure for more
- **3 tax jurisdictions**: Bahrain VAT (10%), India GST (5 slabs), India Income Tax (old + new regime)
- **Event-driven compliance**: invoices auto-validated against jurisdiction rules
- **India tax calculator**: old vs new regime comparison with ₹ savings recommendation
- **Zero breaking changes**: all compliance is additive, all i18n wraps existing strings
