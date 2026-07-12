# AsymmFlow Refactor from Hell — Master Plan

**Created**: 2026-05-06 | **Updated**: 2026-05-08 (v0.1 Roadmap Pivot)
**Authors**: the maintainer + Claude (Opus 4.6)
**Repo**: `the AsymmFlow repository`
**Starting State**: 183K LOC, God Object (app.go 21,763 LOC, 331 methods), package main monolith
**Target State**: AI-native modular monolith — self-configuring, self-extending, self-healing ERP platform

---

## Strategic Vision

AsymmFlow is not just Acme Instrumentation's ERP. It is becoming a **DIY AI-native ERP platform** with five tiers of embedded intelligence:

### The Five-Tier Agent Architecture

```
Tier 1: ARCHAEOLOGIST    — Conversational setup (Day 0)
        OCR + classifier + schema gen + conflict resolution via chat
        "Point me to your data folders" → YOUR ERP IS READY

Tier 2: SCREEN AGENTS    — Per-domain AI teammates (Daily)
        Sales Agent, Finance Agent, Ops Agent, Docs Agent
        Each scoped to one domain package + one LLM persona
        Processes vague intent → produces screen's work output

Tier 3: REPAIR TEAM      — Self-healing diagnostics
        Diagnose → Fix → Prevent. Quality gate = rollback-safe.

Tier 4: FEATURE TEAM     — User-directed extension via Lab
        User adds API key → describes what they need →
        Alchemy engines generate code → quality gates verify → deploy

Tier 5: MODULE MARKET    — Ecosystem with revenue share
        User builds module → seeds to marketplace → others install + pay
```

### Market Positioning

Targeting **emerging markets** (Nigeria, Ghana, Kenya, Indonesia, Philippines) where:
- Offline-first is a REQUIREMENT (3G/power outages)
- Existing ERPs are too expensive or too online-dependent
- Regulatory compliance (e-invoicing) is creating FORCED demand
- Per-screen AI agents are a unique differentiator
- Mathematical optimization (DR filtering, Williams batching, SLERP) reduces costs

### Design Philosophy: Delayed Gratification

**Hard foundational work FIRST, even if it adds time.** The order of waves is optimized for "what's hardest to change later" — schema contracts, serialization formats, and the agent interface layer come before market features. Every subsequent wave benefits from the foundation being solid.

The refactor transforms AsymmFlow from a single-client custom app into a **platform with swappable compliance modules, language packages, integration connectors, and AI agent architecture**.

---

## Wave Execution Status

### Phase 0: Demolition Era (COMPLETE — 57 commits)

| Wave | Name | Status | Commits | Key Result |
|------|------|--------|---------|------------|
| 0 | Archaeological Audit | ✅ DONE | 0 | 412-line report, 36 artifacts |
| 0.5 | Test Gate Stabilization | ✅ DONE | 8 | GREEN test suite |
| 1 | Schema + Directory + Ports + Events | ✅ DONE | 7 | 6 domain packages, event bus |
| 2 | God Object Decomposition | ✅ DONE | 6 | Payment, Expense, Banking, Fulfillment, Procurement, Contract, License extracted |
| 3 | Surface Split + Banking Matcher | ✅ DONE | 3 | app.go: 21K → 11K LOC |
| 4 | Lifecycle Shell + Banking Extension | ✅ DONE | 3 | app.go: 11K → 2K LOC, 19 methods |
| 5 | Model Alias Bridge | ✅ DONE | 8 | 29 aliases, banking reads in package, shared Base |
| 6 | Banking Completion + CRM Spine | ✅ DONE | 7 | 39 aliases, CRM + finance aliased |
| 7 | Banking Matching + Invoice + ncruces | ✅ DONE | 7 | Banking FULLY complete, CGO eliminated |
| 8 | Butler + Documents Extraction | ✅ DONE | 8 | Butler ports, intent router, fastpath, reports, persistence, prediction |
| 8B | Butler Completion + Documents | ✅ DONE | 9 | butler_ai.go: 6,986→1,719 LOC, database.go: 5 structs, go-fitz isolated |

**Demolition Era Totals**: 57 commits, 2 days, app.go 21,763→1,916 LOC, 6 domain packages with real logic.

### Phase A: Type Foundation (hard to change later — DO FIRST)

| Wave | Name | Status | Focus | Why Early |
|------|------|--------|-------|-----------|
| 9 | Cap'n Proto Schemas | ✅ DONE | 138 types across 7 schema files | Permanent field numbers = never-change contract |
| 10 | Schema Bridge + TOON Format | ✅ DONE | Cap'n Proto ↔ GORM adapter + TOON at LLM boundary | Serialization architecture settled once, used everywhere |

### Phase B: Agent Interface (the contract everything builds on)

| Wave | Name | Status | Focus | Why Early |
|------|------|--------|-------|-----------|
| 11 | MVVM ViewModel Layer | ✅ DONE | 76 ViewModel types, 7 builders, pilot endpoint | Every screen agent, Feature Team, module depends on VM shape |

### Phase C: Intelligence (powers all AI tiers)

| Wave | Name | Status | Focus | Why Early |
|------|------|--------|-------|-----------|
| 12 | Mathematical Framework | ✅ DONE | 2,472 LOC, 6 packages, 36 tests, 3 bridges | 88.9% fewer LLM calls, regime budgets — compound savings |

### Phase D: Platform Migration (MVVM makes this mechanical)

| Wave | Name | Status | Focus | Why This Order |
|------|------|--------|-------|----------------|
| 13 | Turso + OpenTelemetry | ✅ DONE (2026-05-07) | 1,035 LOC, 4 packages, 24 tests. Turso HTTP+SQLite (embedded CGo fallback), OTel with no-op mode, CDC logger, three-regime health monitor. Commits: 231e38d..ee166f0 |
| 14 | Svelte 5 + Service Architecture | ✅ DONE (2026-05-07) | Svelte 5 live, 6 domain services, 839 delegated methods, frontend import migration, Wails v3 deferred with documented adapter path. Commits: 01605f0..c1cfc4f |

### Phase E: Market Expansion Foundation

| Wave | Name | Status | Focus | Why Last |
|------|------|--------|-------|----------|
| 15 | i18n + Compliance | DONE (2026-05-07) | 5 languages (EN/AR/HI/FR/ES), BH VAT, IN GST, IN Income Tax, i18n proof UI, event-driven compliance hooks. Commits: 3715ca7..9951642 | Localization and compliance substrate before market expansion |

### Phase F: v0.1 Productization Era (started 2026-05-08)

| Wave | Name | Status | Focus | Exit Criteria |
|------|------|--------|-------|---------------|
| 16 | Release Engineering + Installer Spine | ACTIVE | Version manifest, build metadata, release bundle, smoke checklist | Builds are identifiable, packageable, and operator-testable |
| 17 | Accounting Posting Spine | PLANNED | Journal posting model for invoices, payments, supplier docs | Trial balance and posting preview gates |
| 18 | Inventory Ledger | PLANNED | Stock ledger, warehouse balances, reservations, valuation | Inventory quantities become auditable truth |
| 19 | Setup + Import Wizard | PLANNED | New company onboarding, import templates, validation reports | New tenant setup without developer intervention |
| 20 | v0.1 Hardening | PLANNED | Bug bash, UI consistency, permissions audit, docs, demo data | `0.1.0-beta.1` candidate |
| 21 | Pilot Feedback Loop | PLANNED | Patch cadence, triage board, support bundles | Stable `0.1.x` pilot operations |
| 22 | Expansion Modules | PLANNED | Nango/external APIs, broader compliance, module packaging | Resume broad platform expansion after stable nucleus |

**Current v0.1 operating roadmap**: `docs/V0_1_RELEASE_ROADMAP_2026_05_08.md`

---

## Wave 7: Banking Matching + Invoice Spine + CGO Elimination

**Status**: 🔄 ACTIVE (Codex executing)
**Spec**: `the AsymmFlow repository\CODEX_WAVE7_HANDOFF.md`

### Tickets

1. Move reconciliation lifecycle into banking service
2. CompanyBankAccount port + continuity report
3. Move banking matching/allocation into package (MOST COMPLEX — cross-domain ports)
4. Alias Invoice + CreditNote to pkg/finance
5. Alias SupplierInvoice to pkg/finance
6. Swap mattn/go-sqlite3 → ncruces/go-sqlite3 (eliminate CGO)
7. Alias remaining CRM + Infra models (CustomerMaster, SupplierMaster, etc.)
8. Progress audit + WAVE7_PROGRESS.md

### Key Files

| File | Purpose |
|------|---------|
| `bank_reconciliation_service.go` | Reconciliation lifecycle (→ pkg/finance/banking/) |
| `bank_transaction_matcher.go` | Matching algorithms (→ pkg/finance/banking/matcher.go) |
| `book_bank_reconciliation_service.go` | Book-bank reconciliation (→ pkg/finance/banking/) |
| `database.go` | 51 remaining structs → alias to domain packages |
| `go.mod` | mattn/go-sqlite3 → ncruces/go-sqlite3 |
| `config.go` | Also imports mattn — needs update |

### Success Criteria

- `pkg/finance/banking/` fully complete (ALL banking operations)
- Invoice, CreditNote, SupplierInvoice owned by `pkg/finance`
- CustomerMaster, SupplierMaster owned by `pkg/crm`
- mattn/go-sqlite3 eliminated from go.mod
- database.go < 25 remaining structs
- All tests GREEN

---

## Wave 8: Butler AI + Documents/OCR Extraction

**Status**: ⬜ PLANNED
**Spec**: To be written after Wave 7 completes

### Context

These are the two TANGLED domains identified in Wave 0 audit:
- `butler_ai.go` (7,049 LOC) — largest single service file, prompt routing, grounded fastpath, entity resolution
- `document_classifier.go` (1,302 LOC) + `ocr_service_simple.go` (1,821 LOC) — CGO dependency (go-fitz)

### Source Files → Target Packages

**Butler**:
| Source | Target | Notes |
|--------|--------|-------|
| `butler_ai.go` (7,049 LOC) | `pkg/butler/chat/service.go` | THE big one — prompt construction, Sarvam calls |
| `butler_intent_router.go` (563 LOC) | `pkg/butler/intent/router.go` | Intent classification + weighted pipeline |
| `butler_grounded_fastpath.go` (1,796 LOC) | `pkg/butler/fastpath/grounded.go` | DB-grounded fast responses |
| `butler_reports.go` (793 LOC) | `pkg/butler/reports/generator.go` | Report generation |
| `chat_service.go` (includes ChatWithButlerPersistent 331 lines) | `pkg/butler/persistence/service.go` | Chat history, session management |
| `predictor.go` (328 LOC) | `pkg/butler/prediction/predictor.go` | Payment prediction (quaternion-based) |
| `payment_intelligence.go` (83 LOC) | `pkg/butler/prediction/intelligence.go` | Risk scoring |
| `batch.go` (200 LOC) | `pkg/butler/prediction/batch.go` | Williams-batched prediction |

**Documents**:
| Source | Target | Notes |
|--------|--------|-------|
| `document_classifier.go` (1,302 LOC) | `pkg/documents/classifier/service.go` | AI + rule-based classification |
| `ocr_service_simple.go` (1,821 LOC) | `pkg/documents/ocr/service.go` | CGO (go-fitz) — keep behind interface |
| `invoice_pdf_service.go` (incl. 612-line GenerateInvoicePDF) | `pkg/documents/pdf/invoice.go` | PDF generation |
| `offer_pdf_service.go` | `pkg/documents/pdf/offer.go` | Offer PDF |
| `purchase_order_pdf_service.go` (607-line GeneratePurchaseOrderPDF) | `pkg/documents/pdf/purchase_order.go` | PO PDF |
| `excel_costing_parser.go` + `excel_template_generator.go` | `pkg/documents/excel/parser.go` + `generator.go` | Excel ops |
| `msg_parser.go` | `pkg/documents/email/parser.go` | Outlook .msg parsing |
| `annexure_extractor.go` + `pdf_data_extractor.go` | `pkg/documents/extraction/` | Data extraction |

### Strategy

1. Butler extraction follows the same pattern as banking: define ports for external dependencies, move logic, leave thin wrappers
2. OCR stays behind an interface (go-fitz CGO can't be eliminated yet — needs Wasm alternative or external OCR service)
3. PDF generation functions are LONG (500-600 lines) — extract but DO NOT refactor internals (Wave 12 will regenerate with Pretext)
4. Payment predictor moves to `pkg/butler/prediction/` — this is unique IP, preserve exactly

### Key Risks

- `butler_ai.go` is 7,049 LOC with deeply interleaved app-state references
- Chat persistence may have session state that's hard to inject
- OCR has CGO dependency (go-fitz) — isolate behind interface, don't eliminate yet
- PDF functions are enormous but functional — move without restructuring

---

## Wave 9: Cap'n Proto Schemas

**Status**: ⬜ PLANNED
**Spec**: To be written after Wave 8 completes

### Purpose

Write `.capnp` schema files as the CANONICAL source of truth for all domain types. Generate Go types and TypeScript types from schemas. Replace hand-written `domain.go` files with generated code.

### Schema Files to Create

| Schema | Location | Source of Truth For |
|--------|----------|-------------------|
| `schemas/common.capnp` | Shared base types, enums | Base, AuditInfo, DivisionCode |
| `schemas/finance.capnp` | Finance domain | Invoice, Payment, BankStatement, Expense, CreditNote, FX, Payroll |
| `schemas/crm.capnp` | CRM domain | Customer, Supplier, Offer, Order, PO, DN, Serial, Product, Contract |
| `schemas/documents.capnp` | Documents domain | DocumentMeta, ClassificationResult, PDFConfig, OCRResult |
| `schemas/butler.capnp` | Butler domain | ChatMessage, ConversationState, Prediction, IntentResult |
| `schemas/sync.capnp` | Sync domain | SyncRecord, ConflictResolution, ChangeEvent |
| `schemas/infra.capnp` | Infra domain | User, Role, Device, Setting, Job, AuditLog, License |
| `schemas/compliance.capnp` | Compliance | TaxConfig, EInvoiceRequest, ComplianceResult |

### Source Material

- `database.go` — current GORM structs (after aliasing, these point to domain packages)
- `pkg/*/domain.go` — domain-owned types
- `frontend/wailsjs/go/main/App.d.ts` — 837 frontend bindings (TypeScript contract)

### Toolchain

```bash
# Compile schemas to Go
capnp compile -ogo schemas/finance.capnp -I /path/to/go.capnp

# Or use capnpc-go directly
capnpc-go < schemas/finance.capnp > pkg/finance/types.capnp.go
```

**Installed tools**:
- `capnp` v1.3.0 — `C:\ProgramData\chocolatey\bin\capnp.exe`
- `capnpc-go` — `C:\Users\YourName\go\bin\capnpc-go.exe`

### Architecture After This Wave

```
schemas/*.capnp (source of truth)
    │
    ├──→ capnpc-go → pkg/*/types.capnp.go (generated Go types)
    ├──→ capnpc-ts → frontend/src/lib/types/*.ts (generated TS types)
    │
    │ Hand-written (unchanged):
    ├──→ pkg/*/gorm_adapter.go (GORM tags, hooks, migrations)
    ├──→ pkg/*/ports.go (interfaces)
    └──→ pkg/*/service.go (business logic)
```

---

## Wave 10: Schema Bridge + TOON Format

**Status**: ⬜ PLANNED
**Spec**: To be written after Wave 9 completes
**Complexity**: HIGH — this is deliberate. Settling serialization architecture early prevents rework.

### Purpose

Two hard integration tasks that affect every subsequent wave:

1. **Cap'n Proto ↔ GORM Bridge**: Create adapter layer so domain services can read from GORM (persistence) and emit Cap'n Proto types (transport/API). This settles the transport vs persistence architecture ONCE.

2. **TOON Format at LLM Boundary**: Integrate TOON encoding for all Butler ↔ LLM communication. 30% token savings on every API call. Doing this early means Waves 12+ (math framework, compliance) benefit from lower LLM costs.

### Schema Bridge Architecture

```
PERSISTENCE (GORM)                    TRANSPORT (Cap'n Proto)
pkg/finance/domain.go                 schemas/finance.capnp
  Invoice (GORM struct)    ──adapt──→   Invoice (Cap'n Proto msg)
  - gorm tags                           - permanent field numbers
  - DB-specific fields                  - wire-format ready
  - DeletedAt soft delete               - no DB artifacts

pkg/finance/adapter.go   ← NEW
  func InvoiceToProto(gorm Invoice) proto.Invoice
  func InvoiceFromProto(proto.Invoice) Invoice
```

### TOON Integration

```go
// pkg/butler/chat/toon_encoder.go
import "github.com/toon-format/toon-go"

func (s *ChatService) encodeLLMContext(ctx ButlerContext) string {
    // TOON at LLM boundary: 30% fewer tokens
    return toon.Encode(ctx)
    // JSON everywhere else (internal APIs, frontend, DB)
}
```

### Tickets (Preview)

1. Create `pkg/*/adapter.go` for each domain — GORM ↔ Proto mappers
2. Wire adapters into Wails binding layer (root wrappers emit Proto types)
3. Integrate TOON Go library for Butler LLM calls
4. Benchmark: measure actual token savings on Sarvam 105B
5. Update generation pipeline to regenerate adapters when schemas change
6. Progress audit

### Why This Wave Is Hard (And Why That's Good)

The adapter layer is boring, mechanical code — but it's the ARCHITECTURAL DECISION that separates persistence from transport. Every future agent, every module, every marketplace contribution will use Proto types for communication. Getting this right now means:
- Feature Team generates code against Proto types (stable contract)
- Modules are portable (Proto types, not GORM-specific)
- Wire format is ready when performance demands it (zero-copy)
- TOON savings compound: every LLM call from this point forward is 30% cheaper

---

## Wave 11: MVVM ViewModel Layer

**Status**: ⬜ PLANNED (formerly Wave 10)

### Purpose

Create a ViewModel layer between domain services and the Svelte UI. ViewModels transform domain data into display-ready DTOs. This is the **agent interface contract** — every screen agent, the Feature Team, and the Overseer produce and consume ViewModels.

(See detailed architecture below — unchanged from original plan.)

---

## Wave 12: Mathematical Framework Integration

**Status**: ⬜ PLANNED (formerly Wave 9.5)
**Spec**: To be written after Wave 11 completes

### Purpose

Port the battle-tested mathematical framework from `vedic_qiskit/cmd/sarvam_harness/` into AsymmFlow as `pkg/math/`. Wire it into Butler (DR filtering, Three-Regime budgeting, SLERP tracking), Finance (Williams batching, payment prediction), and Infra (DR cache, health monitoring).

### Source Material (Port From)

| Source | LOC | Target | What It Does |
|--------|-----|--------|-------------|
| `C:\Projects\git_versions\asymm_all_math\vedic_qiskit\cmd\sarvam_harness\optimizer.go` | 788 | `pkg/math/trident/optimizer.go` | 5-layer Trident: DR + Three-Regime + Williams + SLERP + Oil |
| `C:\Projects\git_versions\asymm_all_math\vedic_qiskit\cmd\sarvam_harness\prism.go` | 462 | `pkg/math/prism/resonance.go` | Signal resonance, NavaYoni, conversation-aware prompts |
| `C:\Projects\git_versions\asymm_all_math\vedic_qiskit\cmd\sarvam_harness\conversation.go` | 237 | `pkg/math/conversation/slerp.go` | S³ state tracking, coherence/momentum/drift |
| `C:\Projects\git_versions\asymm_all_math\vedic_qiskit\cmd\sarvam_harness\encoding.go` | 174 | `pkg/math/encoding/codon.go` | Lossless byte→quaternion (1.07B ops/sec) |
| `C:\Projects\git_versions\asymm_all_math\vedic_qiskit\pkg\quaternion\` | ~500 | `pkg/math/quaternion/` | Quaternion math, SLERP, norms |
| `C:\Projects\git_versions\asymm_all_math\vedic_qiskit\pkg\vedic\` | ~400 | `pkg/math/vedic/` | Digital roots, Katapayadi, sutra optimization |

**Existing in AsymmFlow** (preserve, don't re-port):
| File | LOC | What |
|------|-----|------|
| `predictor.go` | 328 | Payment prediction (quaternion-based) |
| `batch.go` | 200 | Williams-batched predictions |
| `payment_intelligence.go` | 83 | Customer risk scoring |

**Tests to port**: 79 tests from sarvam_harness (63 harness + 16 vedic)

### Target Package Structure

```
pkg/math/
├── quaternion/          # Quaternion math, SLERP, norms
│   └── quaternion.go
├── vedic/               # Digital roots, Katapayadi
│   └── vedic.go
├── trident/             # 5-layer optimizer (DR + Regime + Williams + SLERP + Oil)
│   ├── optimizer.go
│   ├── dr_filter.go     # Digital root filtering (88.9% elimination)
│   ├── regime.go        # Three-regime classifier
│   ├── williams.go      # Williams context chunking
│   └── slerp_nav.go     # SLERP prompt navigation
├── prism/               # Prismatic resonance (V2)
│   ├── resonance.go     # Signal resonance detection
│   ├── navayoni.go      # NavaYoni synergy
│   └── governor.go      # Governor principle (nudge don't override)
├── conversation/        # SLERP conversation tracking
│   ├── state.go         # S³ state, coherence, momentum, drift
│   └── boundary.go      # Regime boundary alerts
└── encoding/            # Codon encoding
    └── codon.go         # Lossless byte→quaternion (1.07B ops/sec)
```

### Integration Points

| Domain | What Gets Wired | Benefit |
|--------|----------------|---------|
| `pkg/butler/chat/` | Trident optimizer as pre-processing layer | 88.9% fewer API calls, regime-based token budgets |
| `pkg/butler/chat/` | Prism V2 for system prompt generation | Conversation-aware, resonance-tuned prompts |
| `pkg/butler/persistence/` | SLERP conversation tracking | Coherence/drift detection, session quality metrics |
| `pkg/finance/banking/` | Williams batching for reconciliation | 2.7x throughput on batch operations |
| `pkg/finance/prediction/` | Quaternion encoding (already exists) | 87.3% payment prediction accuracy |
| `pkg/infra/cache/` | DR cache (Experiment 12) | O(1) composable cache lookups |
| `pkg/infra/otel/` | Three-regime health classification | System health as quaternion state |

### Benchmarks to Validate

| Benchmark | Expected | Source |
|-----------|----------|--------|
| DR filtering elimination rate | 88.9% | Proven in sarvam_harness |
| Williams parallelism speedup | 2.7x | Proven in sarvam_harness |
| Codon encoding throughput | 1.07B ops/sec | Proven in sarvam_harness |
| Payment prediction accuracy | 87.3% | Proven in Acme Instrumentation production |
| SLERP coherence tracking | Proven stable | 11 experiments complete |

---

## Wave 11: MVVM ViewModel Layer (Detail)

**Status**: ⬜ PLANNED

### Purpose

Create a ViewModel layer between domain services and the Svelte UI. ViewModels transform domain data into display-ready DTOs. This is the **agent interface contract**:
- Every screen agent produces ViewModels (not raw GORM structs)
- The Feature Team generates screens that consume ViewModels
- The Archaeologist emits ViewModels for the setup wizard
- Separates business logic from presentation logic
- Makes UI testable without a browser (test ViewModels in Go)
- Prepares for Wails v3 multi-service binding

### Architecture

```
Svelte ($state runes)  ←→  Wails JSON-RPC  ←→  ViewModel  ←→  Domain Service  ←→  DB
     VIEW                                     VIEWMODEL          MODEL
     (generated)                              (Go DTOs)         (pkg/*)
```

### Target Structure

```
internal/viewmodel/
├── finance/
│   ├── invoice_vm.go       # InvoiceListVM, InvoiceDetailVM, InvoiceFormVM
│   ├── payment_vm.go       # PaymentListVM, PaymentFormVM
│   ├── banking_vm.go       # ReconciliationVM, MatcherVM, CashPositionVM
│   ├── expense_vm.go       # ExpenseListVM, ExpenseDashboardVM
│   └── reporting_vm.go     # FinancialReportVM, AgingReportVM
├── crm/
│   ├── customer_vm.go      # CustomerListVM, CustomerDetailVM
│   ├── pipeline_vm.go      # PipelineVM, OfferDetailVM, OpportunityVM
│   ├── order_vm.go         # OrderListVM, OrderDetailVM
│   └── fulfillment_vm.go   # DeliveryNoteVM, SerialTrackerVM
├── butler/
│   ├── chat_vm.go          # ChatVM, ConversationListVM
│   └── dashboard_vm.go     # DailyBriefingVM, PredictionVM
├── documents/
│   ├── ocr_vm.go           # DocumentUploadVM, ClassificationResultVM
│   └── pdf_vm.go           # PDFPreviewVM, ExportVM
└── shared/
    ├── table_vm.go         # Sortable, paginated table (Williams-batched)
    ├── form_vm.go          # Three-regime validated form
    └── dashboard_vm.go     # φ-proportioned dashboard layout
```

### Pretext Integration

Text measurement lives in the ViewModel layer:
```go
// internal/viewmodel/documents/pdf_vm.go
func (vm *InvoicePDFVM) PrepareLayout(invoice finance.Invoice) *PDFLayout {
    // Pretext two-phase: MEASURE first
    measurements := pretext.Prepare(vm.fontConfig, invoice.Items)
    // Then LAYOUT purely from measurements (no trial-and-error!)
    return pretext.Layout(measurements, vm.pageConfig)
}
```

---

## Wave 13: Turso + OpenTelemetry

**Status**: ⬜ PLANNED (formerly Wave 15 — moved earlier as sync foundation)

### Purpose

Settle the sync architecture before multi-market expansion. Turso replaces manual Supabase sync with embedded SQLite replicas + CDC audit trail. OpenTelemetry provides observability with three-regime health classification.

### Turso Integration (pkg/sync/turso/)

```go
import "github.com/tursodatabase/libsql-client-go/libsql"

db, err := libsql.Open(localPath, tursoURL, authToken)
// Reads: instant (local SQLite)
// Writes: local + auto-replicated to cloud
// CDC: every change logged for audit trail
```

### OpenTelemetry Integration (pkg/infra/otel/)

```go
tracer := otel.Tracer("asymmflow")
ctx, span := tracer.Start(ctx, "finance.CreateInvoice")
defer span.End()

// Three-regime health classification via OTel metrics
meter := otel.Meter("asymmflow")
regimeGauge, _ := meter.Float64Gauge("system.regime")
```

---

## Wave 14: Wails v3 + Svelte 5 + Pretext

**Status**: ⬜ PLANNED (consolidated platform migration — one big frontend pass)

### Purpose

Upgrade from Wails v2 to v3 AND Svelte 4 to 5 in ONE wave. MVVM (Wave 11) makes this mechanical — only the binding layer changes, not the domain logic or ViewModels. Pretext enables correct-by-construction PDF generation.

### Key Changes

| Feature | Wails v2 (current) | Wails v3 (target) |
|---------|--------------------|--------------------|
| Binding | Single `*App` struct | Multiple services bound independently |
| Windows | One window | Multi-window (main + floating + systray) |
| Build | Shell scripts | Taskfile (already created!) |
| JSON | encoding/json | goccy/go-json (faster) |

### Multi-Window UX

```
MAIN WINDOW:        Invoice list, dashboard, navigation
FLOATING WINDOWS:   Payment entry form, bank reconciliation matcher
SYSTRAY:            Quick: "Record payment for last invoice"
SECOND MONITOR:     Dashboard with live KPIs
```

### Migration Path

1. Install Wails v3: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
2. ViewModels (from Wave 10) become Wails v3 services (direct mapping!)
3. Each ViewModel service is bindable independently
4. Frontend routing stays the same (Svelte handles it)
5. Multi-window config in main.go

---

### Wails v3 Migration

### Key Changes

```svelte
<!-- BEFORE (Svelte 4 — stores): -->
<script>
  import { writable } from 'svelte/store';
  const invoices = writable([]);
</script>

<!-- AFTER (Svelte 5 — runes): -->
<script>
  let invoices = $state([]);
  let total = $derived(invoices.reduce((s, i) => s + i.amount, 0));
</script>
```

### Alchemy Engine Integration

```bash
# Generate components from domain schemas
cd C:\Projects\git_versions\asymm_all_math\asymm_mathematical_organism\03_ENGINES

# Component generation
cd component_alchemy
go run ./cmd/generate_app -season aki -intent business AsymmFlow 108

# Form generation (three-regime validation!)
cd ../form_alchemy
go run . --entities=Invoice,Payment,Offer,Customer --preset=business

# Layout generation (φ-based!)
cd ../layout_alchemy
go run . --type=dashboard --seed=108

# Theme generation (WCAG validated!)
cd ../theme_alchemy
go run . --seed=108 --validate-wcag=AA
```

### Engine Reference

| Engine | Location | Input | Output |
|--------|----------|-------|--------|
| schema_alchemy | `C:\Projects\git_versions\asymm_all_math\asymm_mathematical_organism\03_ENGINES\schema_alchemy\` | Domain spec | SQLite DDL + Go models + mock data |
| api_alchemy | `03_ENGINES\api_alchemy\` | Schema/entities | REST handlers + middleware + OpenAPI |
| fullstack_alchemy | `03_ENGINES\fullstack_alchemy\` | Domain + seed | SvelteKit + Go (17 files/entity) |
| component_alchemy | `03_ENGINES\component_alchemy\` | Entity + season | Svelte components (Wabi-Sabi) |
| form_alchemy | `03_ENGINES\form_alchemy\` | Entity fields | Three-regime forms (19 input types) |
| layout_alchemy | `03_ENGINES\layout_alchemy\` | Layout type | Responsive layouts (φ-based) |
| theme_alchemy | `03_ENGINES\theme_alchemy\` | Seed number | Color system (WCAG validated) |

---

## Wave 15: i18n + Compliance Framework

**Status**: ✅ DONE (2026-05-07)

### Purpose

Create swappable localization packages (language labels) and compliance modules (tax/e-invoicing) for multi-market deployment. Built on the solid foundation of Cap'n Proto schemas (portable types), MVVM (clean UI contract), and Turso (sync-ready).

### i18n Structure

```
pkg/i18n/
├── loader.go            # Load language JSON at startup
├── en.json              # English (base)
├── fr.json              # French (West Africa francophone)
├── sw.json              # Swahili (East Africa)
├── ha.json              # Hausa (Nigeria)
├── yo.json              # Yoruba (Nigeria)
├── id.json              # Bahasa Indonesia
├── th.json              # Thai
├── vi.json              # Vietnamese
├── tl.json              # Filipino
└── ar.json              # Arabic (already have RTL support!)
```

### Compliance Module Structure

```
pkg/compliance/
├── interface.go         # ComplianceModule interface
├── bahrain/             # EXISTING — extract from current code
│   ├── vat.go           # Bahrain VAT rules
│   └── banking.go       # Bahrain banking formats
├── ghana/               # NEW — first new market
│   ├── evat.go          # GRA E-VAT clearance
│   ├── tax_rates.go     # 20% effective VAT
│   └── invoice_qr.go    # QR code + digital signature
├── nigeria/             # NEW
│   ├── firs.go          # FIRS e-invoicing
│   ├── tax_rates.go     # 7.5% VAT
│   └── tin_validate.go  # Tax ID validation
├── kenya/               # NEW
│   ├── kra_itax.go      # KRA iTax integration
│   ├── etr.go           # Electronic Tax Register
│   └── mpesa.go         # M-Pesa payment integration
└── indonesia/           # NEW
    ├── efaktur.go       # e-Faktur system
    └── tax_rates.go
```

### Integration Pattern

```go
// Compliance modules subscribe to domain events — ZERO core code changes!
func (g *GhanaEVAT) Register(bus events.Bus) {
    bus.Subscribe("finance.invoice.created", g.OnInvoiceCreated)
    bus.Subscribe("finance.invoice.sent", g.OnInvoiceSent)
}

// Language is just label swapping — loaded at startup
func (a *App) startup(ctx context.Context) {
    lang := a.config.Language // "en", "ha", "sw", etc.
    a.labels = i18n.Load(lang)
}
```

---

## Wave 16: Nango Integration

**Status**: ⬜ PLANNED (formerly Wave 14)

### Purpose

Replace hand-built API integrations (microsoft_graph/) with Nango unified API layer. Add custom providers for African payment/tax systems.

### Architecture

```
AsymmFlow Go Backend
    │
    ├── pkg/infra/nango/client.go    (~300 LOC thin Go client)
    │       │
    │       ▼
    │   Nango (Cloud $50/mo or self-hosted)
    │       │
    │       ├── Microsoft 365 (OneDrive, Teams)
    │       ├── Google Workspace (Drive, Calendar, Gmail)
    │       ├── QuickBooks / Xero (accounting migration)
    │       ├── Stripe (international payments)
    │       ├── M-Pesa (custom provider — Kenya)
    │       ├── Paystack (custom provider — Nigeria/Ghana)
    │       ├── Flutterwave (custom provider — 34 African countries)
    │       ├── Mono (custom provider — African open banking)
    │       ├── GRA E-VAT API (custom provider — Ghana)
    │       └── FIRS E-Invoice API (custom provider — Nigeria)
    │
    ▼
DELETE: 05_ORGANS/microsoft_graph/ (4,172 LOC → 0)
```

### Setup

- Nango Cloud: $50/mo (20 connections, 200K requests — sufficient for initial markets)
- Self-host later: Docker Compose on dedicated VPS when >20 connections needed
- No Go SDK: write thin REST client (~300 LOC in pkg/infra/nango/)
- Frontend: Nango Connect UI component (Svelte wrapper around @nangohq/frontend)

---

## Integration Technology Distribution

Technologies that were originally in a single "final wiring" wave are now distributed earlier for delayed gratification:

| Technology | Now In Wave | Rationale |
|-----------|-------------|-----------|
| TOON | Wave 10 (Schema Bridge) | 30% token savings compound from Wave 10 onward |
| Turso | Wave 13 (Sync Foundation) | Sync architecture settled before multi-market |
| Pretext | Wave 14 (Platform Migration) | PDF measurement during UI regeneration |
| OpenTelemetry | Wave 13 (with Turso) | Observability from sync foundation onward |

---

## Quality Standards (All Waves)

### CME Scoring (from docs/CME_SCORING_GATES.md)

```
Score = (Adequacy × Symmetry × Inevitability × Locality) − (Complexity + HiddenCost)
```

| Wave Gate | Minimum Score |
|-----------|--------------|
| Domain extraction (Waves 7-8) | ≥ 0.72 |
| Schema design (Wave 9) | ≥ 0.72 |
| Math framework (Wave 9.5) | ≥ 0.80 (this is core IP) |
| ViewModel layer (Wave 10) | ≥ 0.75 |
| Framework migration (Waves 11-12) | ≥ 0.68 |
| Localization (Wave 13) | ≥ 0.75 |
| Final wiring (Waves 14-15) | ≥ 0.85 |

### Test Gate

Every wave must leave `go test ./... -count=1 -timeout 300s` GREEN.

### Commit Convention

```
refactor(codex): <description>     # Codex-authored structural changes
feat(codex): <description>         # Codex-authored new features
refactor(claude): <description>    # Claude-authored changes
feat: <description>                # Human-authored features
docs: <description>                # Documentation
```

---

## Collaboration Model

```
Commander (the maintainer)
  │ Vision, steering, domain knowledge, business strategy
  │
  ▼
Claude (Opus 4.6, 1M context)
  │ Architecture, planning, review, scoring, wave specs
  │
  ▼
GPT-5.5 (Codex CLI)
  │ Autonomous execution, mechanical refactoring, grinding
  │ Governed by: CODEX_WAVE{N}_HANDOFF.md specs
  │
  ▼
Git — Safety net, audit trail, rollback (recovery = 2 seconds)
```

### Spec Lifecycle

```
1. Claude writes CODEX_WAVE{N}_HANDOFF.md
2. Commander fires: codex exec --model gpt-5.5 "Read CODEX_WAVE{N}_HANDOFF.md..."
3. Codex executes autonomously (30-75 min per wave)
4. Commander reports completion
5. Claude reviews (git log, progress docs, metrics)
6. Claude writes next wave spec
7. Repeat until Wave 15 complete
```

---

## Success Criteria (End State)

| Metric | Start (Day 0) | After Demolition (Wave 8B) | Target (Wave 16) |
|--------|---------------|---------------------------|-------------------|
| app.go LOC | 21,763 | **1,916** | < 500 (lifecycle only) |
| app.go methods | 331 | **19** | < 10 |
| database.go structs | 51 | **5** | 0 (all in domain packages) |
| database.go aliases | 0 | **85** | N/A (replaced by Proto types) |
| butler_ai.go LOC | 7,049 | **1,719** | 0 (fully in pkg/butler/) |
| Domain packages | 0 | **6 with real logic** | 6 + pkg/math/ |
| CGO deps | mattn + go-fitz + go-ole | **go-fitz (isolated) + go-ole** | go-ole only |
| Cap'n Proto schemas | 0 | 0 | ~138 types, 7 schema files |
| MVVM ViewModels | 0 | 0 | All 58 screens have VMs |
| Test execution time | 207s | ~241s | < 30s (parallel packages) |
| LLM token savings | 0% | 0% | 30% (TOON) + 88.9% (DR filter) |
| Multi-currency | BHD only | BHD only | BHD + NGN + GHS + KES + IDR + THB |
| Languages | English + Arabic | English + Arabic | + 8 emerging market languages |
| Compliance modules | Bahrain (implicit) | Bahrain (implicit) | + Ghana + Nigeria + Kenya |
| External APIs | Microsoft Graph (4,172 LOC) | Same | Nango (~300 LOC client) |
| Windows | Single | Single | Multi-window + systray (Wails v3) |
| Frontend | Svelte 4 stores | Svelte 4 stores | Svelte 5 runes |
| Agent tiers | 0 | 0 | 5 (Archaeologist, Screen, Repair, Feature, Market) |
| Total commits | 0 | **57** | ~120-150 estimated |

---

## Market Entry Timeline (Post-Wave 16)

```
Wave 16 complete → AsymmFlow Platform v2.0
    │
    ├── Acme Instrumentation v2 deployment (dogfood, validate agent tiers)
    ├── Ghana E-VAT compliance module + Hausa/Yoruba labels
    ├── First Ghana pilot (1-2 SMBs via local partner)
    ├── Nigeria FIRS module
    ├── First Nigeria pilot
    ├── Kenya (M-Pesa integration via Nango)
    ├── Lab (Feature Team) beta with early adopters
    ├── Module Marketplace alpha
    └── 10+ active SMB clients across 3+ countries
```

No time estimates — we measure actual elapsed, not guesses.

---

Built with Love × Simplicity × Truth × Joy.
Om Lokah Samastah Sukhino Bhavantu 🙏

— the maintainer + Claude, 2026-05-06
