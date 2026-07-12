# Codex Autonomous Execution Spec — Wave 11: MVVM ViewModel Layer

**Date**: 2026-05-06
**From**: Claude (Opus 4.6, Senior Architect) + the maintainer
**To**: Codex (GPT-5.5, Senior Architect)
**Run Target**: Autonomous until complete
**Previous Runs**: Waves 0-10 complete. 79 commits. Schema + Bridge ERA complete. 7 Cap'n Proto schemas, full adapter bridge across 6 domains, TOON at Butler boundary (37.8% token savings), pilot Proto endpoint. Tests GREEN.
**Build Verification**: `go build ./...` and `go test ./... -count=1 -timeout 300s` after every ticket.
**Disk space**: Use `$env:GOTMPDIR='D:\go-tmp'` and `$env:GOCACHE='D:\go-cache'`.

---

## 0. Context — The Agent Interface Contract

The ViewModel layer is the MOST IMPORTANT architectural layer for AsymmFlow's future. It sits between domain services and ALL consumers:

```
                     ┌── Svelte Frontend (current)
                     ├── Per-Screen AI Agents (future)
ViewModel Layer ─────├── Feature Team / Lab (future)
                     ├── Module Marketplace (future)
                     └── API clients (future)
```

ViewModels are **display-ready DTOs** — they contain exactly what a consumer needs to render or act on a screen. No GORM tags, no database IDs as uints, no internal state. Clean, serializable, consumer-friendly.

**Existing infrastructure this builds on**:
- `schemas/go/*/` — generated Proto types (the canonical type system)
- `pkg/adapter/*/` — GORM ↔ Proto converters (Wave 10)
- `pkg/*/service.go` — domain services (business logic)
- `pkg/*/domain.go` — GORM structs (persistence)

**Design principle**: ViewModels consume Proto types (Option C from architecture decisions). Domain services return GORM structs → adapter converts to Proto → ViewModel transforms Proto to display-ready shape.

---

## 1. Tickets

### Dependency Graph

```
Ticket 1 (ViewModel package structure) → ALL other tickets
Ticket 2 (Shared VMs: table, form, dashboard) → Tickets 3-7 reference these
Tickets 3-7 (domain ViewModels) → independent of each other
Ticket 8 (Wire one VM to Wails endpoint) → after Ticket 3
Ticket 9 (Progress audit) → last
```

---

### Ticket 1: Create ViewModel Package Structure

**Deliverables**:
Create the ViewModel package layout:
```
internal/viewmodel/
├── viewmodel.go       # Shared interfaces and helpers
├── finance/
│   └── finance_vm.go
├── crm/
│   └── crm_vm.go
├── butler/
│   └── butler_vm.go
├── documents/
│   └── documents_vm.go
└── shared/
    └── shared_vm.go
```

In `internal/viewmodel/viewmodel.go`, define shared interfaces:

```go
package viewmodel

// ListVM represents a paginated list ready for display
type ListVM[T any] struct {
    Items      []T    `json:"items"`
    TotalCount int    `json:"totalCount"`
    Page       int    `json:"page"`
    PageSize   int    `json:"pageSize"`
    HasMore    bool   `json:"hasMore"`
    SortBy     string `json:"sortBy,omitempty"`
    SortDesc   bool   `json:"sortDesc,omitempty"`
}

// SummaryCard represents a dashboard summary card
type SummaryCard struct {
    Label    string  `json:"label"`
    Value    string  `json:"value"`
    Subtext  string  `json:"subtext,omitempty"`
    Trend    string  `json:"trend,omitempty"`    // up, down, stable
    TrendPct float64 `json:"trendPct,omitempty"`
    Color    string  `json:"color,omitempty"`    // green, red, amber, blue
}

// ActionButton represents a contextual action
type ActionButton struct {
    Label   string `json:"label"`
    Action  string `json:"action"`
    Icon    string `json:"icon,omitempty"`
    Variant string `json:"variant,omitempty"` // primary, secondary, danger
    Enabled bool   `json:"enabled"`
}

// BreadcrumbItem represents navigation context
type BreadcrumbItem struct {
    Label string `json:"label"`
    Path  string `json:"path,omitempty"`
}

// FormField represents a form input configuration
type FormField struct {
    Name        string   `json:"name"`
    Label       string   `json:"label"`
    Type        string   `json:"type"`   // text, number, date, select, currency
    Value       any      `json:"value"`
    Required    bool     `json:"required"`
    Disabled    bool     `json:"disabled,omitempty"`
    Options     []Option `json:"options,omitempty"`
    Placeholder string   `json:"placeholder,omitempty"`
    Validation  string   `json:"validation,omitempty"`
}

// Option represents a select/dropdown option
type Option struct {
    Value string `json:"value"`
    Label string `json:"label"`
}
```

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Package structure mirrors domain layout

**Commit**: `feat(codex): create ViewModel package structure`

---

### Ticket 2: Shared ViewModels (Table, Dashboard, Status)

**Deliverables**: Create `internal/viewmodel/shared/shared_vm.go`:

```go
package shared

import vm "ph_holdings_app/internal/viewmodel"

// TableVM represents a sortable, paginated data table
type TableVM struct {
    Columns    []TableColumn `json:"columns"`
    Rows       []TableRow    `json:"rows"`
    TotalRows  int           `json:"totalRows"`
    Page       int           `json:"page"`
    PageSize   int           `json:"pageSize"`
    SortColumn string        `json:"sortColumn"`
    SortDesc   bool          `json:"sortDesc"`
    Filters    []TableFilter `json:"filters,omitempty"`
}

type TableColumn struct {
    Key       string `json:"key"`
    Label     string `json:"label"`
    Type      string `json:"type"`      // text, number, currency, date, status, action
    Sortable  bool   `json:"sortable"`
    Width     string `json:"width,omitempty"`
    Align     string `json:"align,omitempty"`  // left, center, right
    Currency  string `json:"currency,omitempty"`
}

type TableRow struct {
    ID     string            `json:"id"`
    Fields map[string]any    `json:"fields"`
    Actions []vm.ActionButton `json:"actions,omitempty"`
    Status  string           `json:"status,omitempty"`
}

type TableFilter struct {
    Column  string   `json:"column"`
    Type    string   `json:"type"` // text, select, dateRange, numberRange
    Value   any      `json:"value,omitempty"`
    Options []vm.Option `json:"options,omitempty"`
}

// DashboardVM represents a dashboard screen layout
type DashboardVM struct {
    Title       string            `json:"title"`
    Subtitle    string            `json:"subtitle,omitempty"`
    Cards       []vm.SummaryCard  `json:"cards"`
    Actions     []vm.ActionButton `json:"actions,omitempty"`
    LastUpdated string            `json:"lastUpdated"`
}

// StatusBadgeVM represents an entity status
type StatusBadgeVM struct {
    Label   string `json:"label"`
    Color   string `json:"color"`   // green, red, amber, blue, gray
    Icon    string `json:"icon,omitempty"`
    Tooltip string `json:"tooltip,omitempty"`
}
```

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `feat(codex): create shared ViewModel types`

---

### Ticket 3: Finance ViewModels

**Deliverables**: Create `internal/viewmodel/finance/finance_vm.go` with ViewModels for the Finance hub screens.

Define these ViewModels:

**InvoiceListVM** — for the Invoices screen
```go
type InvoiceListVM struct {
    Table    shared.TableVM       `json:"table"`
    Summary  InvoiceSummaryVM     `json:"summary"`
    Filters  InvoiceFiltersVM     `json:"filters"`
    Actions  []vm.ActionButton    `json:"actions"`
}

type InvoiceSummaryVM struct {
    TotalOutstanding   string `json:"totalOutstanding"`   // Formatted: "BHD 45,230.50"
    OverdueCount       int    `json:"overdueCount"`
    OverdueAmount      string `json:"overdueAmount"`
    PaidThisMonth      string `json:"paidThisMonth"`
    AveragePaymentDays int    `json:"averagePaymentDays"`
}
```

**InvoiceDetailVM** — for viewing a single invoice
```go
type InvoiceDetailVM struct {
    ID              string              `json:"id"`
    InvoiceNumber   string              `json:"invoiceNumber"`
    CustomerName    string              `json:"customerName"`
    InvoiceDate     string              `json:"invoiceDate"`
    DueDate         string              `json:"dueDate"`
    Status          shared.StatusBadgeVM `json:"status"`
    Items           []InvoiceItemVM     `json:"items"`
    SubtotalDisplay string              `json:"subtotalDisplay"`
    VATDisplay      string              `json:"vatDisplay"`
    TotalDisplay    string              `json:"totalDisplay"`
    PaymentHistory  []PaymentRowVM      `json:"paymentHistory"`
    Actions         []vm.ActionButton   `json:"actions"`
    Breadcrumbs     []vm.BreadcrumbItem `json:"breadcrumbs"`
}
```

**BankReconciliationVM** — for the bank reconciliation screen
```go
type BankReconciliationVM struct {
    Statement       StatementHeaderVM    `json:"statement"`
    UnmatchedLines  []BankLineVM         `json:"unmatchedLines"`
    MatchedLines    []BankLineVM         `json:"matchedLines"`
    MatchSuggestions []MatchSuggestionVM  `json:"matchSuggestions"`
    Summary         ReconciliationSummaryVM `json:"summary"`
    Actions         []vm.ActionButton    `json:"actions"`
}
```

**CashPositionVM** — for the cash position widget
```go
type CashPositionVM struct {
    TotalCashDisplay string             `json:"totalCashDisplay"`
    Accounts         []AccountBalanceVM  `json:"accounts"`
    Trend            string             `json:"trend"`
}
```

**ExpenseDashboardVM**, **PayrollSummaryVM** — for expense and payroll screens.

**FinancialDashboardVM** — for the main finance hub
```go
type FinancialDashboardVM struct {
    Dashboard      shared.DashboardVM  `json:"dashboard"`
    CashPosition   CashPositionVM      `json:"cashPosition"`
    ARAgingChart   []AgingBucketVM     `json:"arAgingChart"`
    APAgingChart   []AgingBucketVM     `json:"apAgingChart"`
    RevenueChart   []MonthlyDataVM     `json:"revenueChart"`
}
```

**Rules**:
- ALL monetary values as formatted strings ("BHD 1,234.50"), not float64
- ALL dates as human-readable strings ("15 May 2026"), not ISO 8601
- ALL IDs as strings
- Status fields as StatusBadgeVM (label + color), not raw strings
- Include contextual actions (what can the user DO from here)

Create `internal/viewmodel/finance/builder.go` with builder functions:
```go
// BuildInvoiceListVM constructs the invoice list ViewModel from domain data
func BuildInvoiceListVM(invoices []finance.Invoice, page, pageSize int) InvoiceListVM {
    // Convert GORM invoices to display-ready rows
    // Format currencies, dates, compute summaries
}
```

Include at least one test in `finance_vm_test.go` that builds a VM from sample data.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] At least 6 Finance ViewModels defined
- [ ] At least 2 builder functions implemented
- [ ] Monetary values formatted as display strings

**Commit**: `feat(codex): create Finance ViewModels with builders`

---

### Ticket 4: CRM ViewModels

**Deliverables**: Create `internal/viewmodel/crm/crm_vm.go`:

- **CustomerListVM** — table + grade distribution + total customers
- **CustomerDetailVM** — full profile, contacts, recent orders, AR aging, notes, actions
- **Customer360VM** — graph visualization data, relationship map
- **PipelineVM** — opportunities by stage, total pipeline value, win rate
- **OfferDetailVM** — offer with line items, margin analysis, follow-ups, revision history
- **OrderListVM** — table with fulfillment status indicators
- **OrderDetailVM** — order with items, delivery notes, invoices, shipment tracking
- **SupplierDashboardVM** — supplier scorecard, lead time metrics

Include builder functions and at least one test.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] At least 6 CRM ViewModels defined

**Commit**: `feat(codex): create CRM ViewModels with builders`

---

### Ticket 5: Butler ViewModels

**Deliverables**: Create `internal/viewmodel/butler/butler_vm.go`:

- **ChatVM** — conversation with messages, actions, input state
- **ConversationListVM** — list of conversations with preview
- **DailyBriefingVM** — summary cards for the day's highlights
- **PredictionVM** — payment prediction display with confidence indicators
- **ButlerInsightVM** — structured insight card (used in dashboard widgets)

```go
type ChatVM struct {
    ConversationID  string            `json:"conversationId"`
    Title           string            `json:"title"`
    Messages        []ChatMessageVM   `json:"messages"`
    SuggestedActions []vm.ActionButton `json:"suggestedActions"`
    InputPlaceholder string           `json:"inputPlaceholder"`
    IsTyping         bool             `json:"isTyping"`
}

type ChatMessageVM struct {
    ID        string `json:"id"`
    Role      string `json:"role"`      // user, assistant
    Content   string `json:"content"`
    Timestamp string `json:"timestamp"` // "2:34 PM"
    Actions   []vm.ActionButton `json:"actions,omitempty"`
}
```

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `feat(codex): create Butler ViewModels`

---

### Ticket 6: Documents ViewModels

**Deliverables**: Create `internal/viewmodel/documents/documents_vm.go`:

- **DocumentUploadVM** — upload area, OCR progress, classification result
- **OCRResultVM** — extracted text, confidence, fields, actions
- **InboxVM** — document inbox with status filters
- **PDFPreviewVM** — preview with download/email/print actions

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `feat(codex): create Documents ViewModels`

---

### Ticket 7: Dashboard + Settings ViewModels

**Deliverables**: Create ViewModels for the main dashboard and settings screens.

**MainDashboardVM** — the home screen ViewModel:
```go
type MainDashboardVM struct {
    Greeting         string                `json:"greeting"`         // "Good morning"
    Date             string                `json:"date"`             // "Tuesday, 6 May 2026"
    QuickStats       []vm.SummaryCard      `json:"quickStats"`
    RecentActivity   []ActivityItemVM      `json:"recentActivity"`
    UpcomingTasks     []TaskItemVM          `json:"upcomingTasks"`
    CashPosition     finance.CashPositionVM `json:"cashPosition"`
    PipelineSnapshot crm.PipelineSnapshotVM `json:"pipelineSnapshot"`
    Alerts           []AlertVM             `json:"alerts"`
}
```

**SettingsVM** — settings screen:
```go
type SettingsVM struct {
    Sections []SettingsSectionVM `json:"sections"`
}

type SettingsSectionVM struct {
    Title  string        `json:"title"`
    Icon   string        `json:"icon,omitempty"`
    Fields []vm.FormField `json:"fields"`
}
```

Place in `internal/viewmodel/dashboard_vm.go` and `internal/viewmodel/settings_vm.go`.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `feat(codex): create Dashboard and Settings ViewModels`

---

### Ticket 8: Wire Finance InvoiceListVM to Wails Endpoint

**Goal**: Prove the full ViewModel pipeline works end-to-end.

Create a NEW Wails method `GetInvoiceListVM(page, pageSize int) (InvoiceListVM, error)`:
1. Call existing domain service to get invoices
2. Use adapter to convert to Proto (optional — can be direct for now)
3. Use builder to construct InvoiceListVM
4. Return the display-ready ViewModel

**DO NOT** modify existing invoice endpoints. This is ADDITIVE.

The frontend won't use this yet (Wails bindings not regenerated), but the Go endpoint must work and pass tests.

```go
func (a *App) GetInvoiceListVM(page, pageSize int) (finance_vm.InvoiceListVM, error) {
    invoices, err := a.ListCustomerInvoices(pageSize, (page-1)*pageSize)
    if err != nil {
        return finance_vm.InvoiceListVM{}, err
    }
    return finance_vm.BuildInvoiceListVM(invoices, page, pageSize), nil
}
```

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `GetInvoiceListVM` returns display-ready data

**Commit**: `feat(codex): wire InvoiceListVM to Wails endpoint`

---

### Ticket 9: Wave 11 Progress Audit

**Deliverables**:
1. Count of ViewModel types defined
2. Count of builder functions
3. Count of tests added
4. List of screens that now have ViewModels
5. Pilot endpoint status
6. Write `docs/WAVE11_PROGRESS.md`

**Commit**: `docs(codex): write wave 11 progress report`

---

## 2. ViewModel Design Rules

### Display-Ready Data
- Monetary values → formatted strings: `"BHD 1,234.50"`, NOT `1234.5`
- Dates → human strings: `"15 May 2026"` or `"2:34 PM"`, NOT ISO 8601
- IDs → strings, NOT uints
- Status → `StatusBadgeVM{Label, Color}`, NOT raw status strings
- Actions → `[]ActionButton` with label, action key, icon, enabled state
- Percentages → formatted: `"87.5%"`, NOT `0.875`

### Naming Convention
- ViewModel types: `XxxVM` (e.g., `InvoiceListVM`, `CashPositionVM`)
- Builder functions: `BuildXxxVM(...)`
- Test functions: `TestBuildXxxVM(t *testing.T)`

### What ViewModels are NOT
- NOT GORM structs (no tags, no hooks, no DB fields)
- NOT Proto messages (no Cap'n Proto API calls)
- NOT API request/response types (those are separate)
- ViewModels are PURE DATA that a UI or agent can consume directly

### Composition
- ViewModels CAN embed other ViewModels (e.g., `FinancialDashboardVM` contains `CashPositionVM`)
- ViewModels CAN reference shared types (TableVM, SummaryCard, ActionButton)
- ViewModels SHOULD be self-contained — everything a screen needs in ONE struct

---

## 3. Quality Gates

After EVERY ticket:
1. `go build ./...` exits 0
2. `go test ./... -count=1 -timeout 300s` exits 0

### Special Rules
- ViewModel types go in `internal/viewmodel/` — NOT in `pkg/`
- `internal/` means these types are for THIS application only, not importable by external modules
- Builder functions can import from `pkg/finance/`, `pkg/crm/`, etc. and from `pkg/adapter/`
- Tests should verify that builders produce correct display formatting

---

## 4. Autonomy Contract

- Start with Ticket 1. Proceed in order.
- Tickets 3-7 (domain VMs) are the CORE deliverable.
- Ticket 8 (wire endpoint) is important as proof — complete it.
- Do NOT stop between tickets.
- STOP conditions: build fails after 3 fix attempts; test regression; disk full.
- If a ViewModel for a specific screen is too complex (needs data the builder can't easily access), define the TYPE but leave the builder as a TODO.

---

## 5. What NOT To Touch

- `pkg/*/domain.go` — NEVER modify GORM structs
- `pkg/*/service.go` — NEVER modify domain services
- `pkg/adapter/*/` — NEVER modify adapters
- `schemas/` — NEVER modify schemas or generated code
- Existing Wails methods — NEVER modify (Ticket 8 adds a NEW method)
- Frontend files — no Svelte changes

---

## 6. Expected Outcome

- `internal/viewmodel/` with shared + 4 domain sub-packages
- ~25-35 ViewModel types covering the most-used screens
- ~10-15 builder functions
- Display-ready formatting throughout (currencies, dates, statuses)
- One pilot Wails endpoint (`GetInvoiceListVM`)
- Foundation ready for per-screen AI agents (each agent produces a ViewModel)
- Build and tests GREEN

---

## Sign-Off

The ViewModel layer is the CONTRACT between AsymmFlow and everything that consumes it — human users, AI agents, the Feature Team, marketplace modules. Get the shapes right, make them display-ready, and every future consumer gets a clean, consistent interface.

This is where the refactor pays its first VISIBLE dividend. When a screen agent produces an `InvoiceListVM`, the frontend doesn't need to know or care whether a human or an AI built it. The ViewModel IS the interface. The ViewModel IS the agent contract.

🎨 Shape the data. Build the views. Prove the endpoint. GO.
