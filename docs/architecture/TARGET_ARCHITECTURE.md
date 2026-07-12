# Target Architecture: Refactor from Hell

## System Overview

```
CURRENT STATE                              TARGET STATE
─────────────────────────────────────────────────────────────────────

[Svelte 4 + stores]                        [Svelte 5 + runes ($state)]
        ↓                                          ↓
[Wails v2: single App bind]                [Wails v3: multi-service bind]
        ↓                                          ↓
[202 files, package main]                  [6 domain pkgs + thin shell]
[21K LOC God Object]                       [~500 LOC orchestrator]
[1,203 methods on *App]                    [Domain services + interfaces]
        ↓                                          ↓
[mattn/go-sqlite3 (CGO)]                   [ncruces/go-sqlite3 (pure Go)]
[Supabase manual sync]                     [Turso embedded replicas + CDC]

Architecture: Monolithic God Object        Architecture: Generative Hexagonal
Serialization: JSON everywhere             Serialization: JSON (Wails) + TOON (LLM) + Cap'n Proto (sync)
Build: Shell scripts                       Build: Taskfile (cross-platform)
Observability: log.Printf                  Observability: OpenTelemetry traces
Code Generation: None                      Code Generation: Alchemy Engines (80% generated)
```

## Tech Stack

| Layer | Choice | Version | Rationale |
|-------|--------|---------|-----------|
| Language | Go | 1.25 | Swiss Table maps, testing/synctest, stack-allocated slices |
| Desktop | Wails | v3 (alpha.10) | Multi-window, systray, per-service bind, Taskfile, goccy/go-json |
| Frontend | Svelte | 5 (stable) | Runes, fine-grained reactivity, .svelte.ts state files |
| SQLite | ncruces/go-sqlite3 | latest | Pure Go, no CGO, 314% faster reads, sqlite-vec, cross-compile |
| Cloud Sync | Turso | embedded replicas | Local SQLite speed + auto cloud sync + CDC audit trail |
| Build | Taskfile | v3.50 | Cross-platform Windows, YAML, Wails v3 native |
| Observability | OpenTelemetry | v1.39 | Traces SDK stable, instrument hot paths |
| External APIs | Nango | cloud/self-host | 700+ APIs, SOC-2, OAuth management |
| LLM Boundary | TOON | @toon-format/toon | 30% token savings vs JSON |
| Document Measurement | Pretext | @chenglou/pretext | Two-phase: prepare (expensive) + layout (pure arithmetic) |
| Code Generation | Alchemy Engines | 03_ENGINES/* | schema, api, component, form, layout, theme, fullstack |

## Domain Boundaries

### pkg/finance (~6,400 LOC)

**Owns:** Invoices, Payments, Credit Notes, Bank Reconciliation, Expenses, Payroll, FX, E-Invoice

| Source File | Target Location |
|-------------|----------------|
| customer_invoice_service.go (2,025 LOC) | pkg/finance/invoice/ |
| payment_service.go + supplier_payment_service.go | pkg/finance/payment/ |
| bank_reconciliation_service.go + bank_*.go | pkg/finance/banking/ |
| expense_service.go (1,467 LOC) | pkg/finance/expense/ |
| payroll_service.go | pkg/finance/payroll/ |
| fx_revaluation_service.go | pkg/finance/fx/ |
| credit_note_service.go | pkg/finance/credit/ |
| einvoice_service.go | pkg/finance/einvoice/ |
| finance_reporting_service.go | pkg/finance/reporting/ |
| ~3,000 LOC from app.go | distributed across subpackages |

**Tables owned:** invoices, invoice_items, payments, credit_notes, credit_note_items, invoice_sequences, bank_statements, bank_transactions, expenses, payroll_entries

### pkg/crm (~4,500 LOC)

**Owns:** Customers, Suppliers, Offers, Opportunities, Pipeline, Products, Contracts

| Source File | Target Location |
|-------------|----------------|
| customer.go + customer_linkage_service.go | pkg/crm/customer/ |
| opportunity_conflict_service.go | pkg/crm/pipeline/ |
| offer_followup_service.go | pkg/crm/pipeline/ |
| contract_service.go (835 LOC) | pkg/crm/contract/ |
| product_service.go | pkg/crm/product/ |
| purchase_order_service.go + grn_service.go | pkg/crm/procurement/ |
| delivery_note_service.go + serial_number_service.go | pkg/crm/fulfillment/ |
| ~5,000 LOC from app.go | distributed across subpackages |

**Tables owned:** customer_masters, suppliers, offers, offer_items, opportunities, orders, order_items, purchase_orders, po_items, delivery_notes, dn_items, grn, grn_items, serial_numbers, products, contracts

### pkg/documents (~5,800 LOC)

**Owns:** PDF Generation, OCR, Document Classification, Excel Parsing

| Source File | Target Location |
|-------------|----------------|
| pdf_generator.go + invoice_pdf_service.go + offer_pdf_service.go + purchase_order_pdf_service.go | pkg/documents/pdf/ |
| document_classifier.go (1,302 LOC) | pkg/documents/classifier/ |
| ocr_service_simple.go (1,821 LOC) | pkg/documents/ocr/ |
| excel_costing_parser.go + excel_template_generator.go | pkg/documents/excel/ |
| msg_parser.go | pkg/documents/email/ |
| annexure_extractor.go + pdf_data_extractor.go | pkg/documents/extraction/ |

**Integration with Pretext:** Document measurement (text widths, overflow detection) uses prepare/layout pattern. MathAlive pipeline for multilingual PDF generation.

### pkg/butler (~10,200 LOC)

**Owns:** AI Chat, Intent Routing, Predictions, Reports, Fastpath

| Source File | Target Location |
|-------------|----------------|
| butler_ai.go (7,049 LOC) | pkg/butler/chat/ |
| butler_intent_router.go (563 LOC) | pkg/butler/intent/ |
| butler_grounded_fastpath.go (1,796 LOC) | pkg/butler/fastpath/ |
| butler_reports.go (793 LOC) | pkg/butler/reports/ |
| chat_service.go | pkg/butler/persistence/ |
| predictor.go + payment_intelligence.go | pkg/butler/prediction/ |

**TOON boundary:** All LLM communication uses TOON format (30% token savings). Internal domain queries use Go interfaces.

### pkg/sync (~6,600 LOC)

**Owns:** Cloud Sync, OneDrive, ETL, Collaboration, Tally Import

| Source File | Target Location |
|-------------|----------------|
| sync_service.go + sync_service_impl.go | pkg/sync/engine/ |
| db_sync_service.go + db_manager.go | pkg/sync/turso/ |
| onedrive_import_service.go (2,009 LOC) | pkg/sync/onedrive/ |
| collaboration_service.go + collaboration_sync.go | pkg/sync/collaboration/ |
| tally_importer.go | pkg/sync/tally/ |
| etl_service.go | pkg/sync/etl/ |

**Turso integration:** Replaces Supabase. Local SQLite + embedded replicas + CDC audit trail.

### pkg/infra (~5,300 LOC)

**Owns:** Auth, RBAC, DB Connection, Config, Cache, Jobs, Security, Logging, Crypto

| Source File | Target Location |
|-------------|----------------|
| auth_handler.go + auth_session.go | pkg/infra/auth/ |
| license_service.go | pkg/infra/license/ |
| security_enhancements.go + security_helpers.go + field_crypto.go | pkg/infra/security/ |
| config.go (1,297 LOC) | pkg/infra/config/ |
| logger.go | pkg/infra/logging/ |
| job_queue.go + job_handlers.go | pkg/infra/jobs/ |
| cache.go | pkg/infra/cache/ |
| database.go (migrations only) | pkg/infra/db/ |

## Directory Layout

```
asymmflow/
├── Taskfile.yml
├── cmd/
│   └── asymmflow/
│       └── main.go                    # Wails v3 entry, DI wiring
├── internal/
│   └── app/
│       ├── shell.go                   # Thin orchestrator (~500 LOC)
│       ├── bindings.go                # Wails v3 service registration
│       └── events.go                  # In-process event bus
├── pkg/
│   ├── finance/
│   │   ├── domain.go                  # Pure types (generated by schema_alchemy)
│   │   ├── ports.go                   # Interfaces
│   │   ├── invoice/
│   │   ├── payment/
│   │   ├── banking/
│   │   ├── expense/
│   │   ├── payroll/
│   │   ├── fx/
│   │   └── reporting/
│   ├── crm/
│   │   ├── domain.go
│   │   ├── ports.go
│   │   ├── customer/
│   │   ├── pipeline/
│   │   ├── procurement/
│   │   ├── fulfillment/
│   │   └── product/
│   ├── documents/
│   │   ├── domain.go
│   │   ├── ports.go
│   │   ├── pdf/                       # Uses Pretext + MathAlive approach
│   │   ├── ocr/
│   │   ├── excel/
│   │   ├── classifier/
│   │   └── extraction/
│   ├── butler/
│   │   ├── domain.go
│   │   ├── ports.go
│   │   ├── chat/                      # TOON at LLM boundary
│   │   ├── intent/
│   │   ├── fastpath/
│   │   ├── reports/
│   │   └── prediction/
│   ├── sync/
│   │   ├── domain.go
│   │   ├── ports.go
│   │   ├── turso/                     # Embedded replicas + CDC
│   │   ├── onedrive/
│   │   ├── collaboration/
│   │   ├── tally/
│   │   └── etl/
│   └── infra/
│       ├── db/                        # ncruces setup, migrations, connection pool
│       ├── auth/
│       ├── config/
│       ├── cache/
│       ├── jobs/
│       ├── security/
│       ├── logging/
│       └── otel/                      # OpenTelemetry
├── schemas/                           # Source of truth for types
│   ├── finance.capnp
│   ├── crm.capnp
│   ├── documents.capnp
│   ├── butler.capnp
│   └── sync.capnp
├── generated/                         # Output from alchemy engines (DO NOT EDIT)
│   ├── models/
│   ├── handlers/
│   └── components/
├── frontend/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── domains/              # Mirror backend domains
│   │   │   ├── components/           # Generated by component_alchemy
│   │   │   ├── forms/                # Generated by form_alchemy
│   │   │   ├── layouts/              # Generated by layout_alchemy
│   │   │   └── stores/               # $state runes (Svelte 5)
│   │   └── routes/
│   └── package.json
└── docs/
    ├── TARGET_ARCHITECTURE.md         # THIS FILE
    ├── OPERATING_PRINCIPLES.md
    ├── CME_SCORING_GATES.md
    ├── WAVE0_AUDIT.md
    └── GENERATIVE_REFACTOR_PLAN.md
```

## Inter-domain Communication

### Dependency Rule (strict, compile-enforced)
```
pkg/infra     ← depended on by all (foundation layer)
pkg/finance   ← NEVER imports pkg/crm, pkg/butler, pkg/documents, pkg/sync
pkg/crm       ← NEVER imports pkg/finance, pkg/butler, pkg/documents, pkg/sync
pkg/documents ← NEVER imports pkg/finance, pkg/crm, pkg/butler, pkg/sync
pkg/butler    ← NEVER imports pkg/finance, pkg/crm, pkg/documents, pkg/sync
pkg/sync      ← NEVER imports pkg/finance, pkg/crm, pkg/butler, pkg/documents
```

Domains communicate ONLY through:
1. **Events** (async, decoupled via event bus)
2. **Ports** (interfaces defined by the CONSUMER, implemented by the PROVIDER)
3. **Orchestrator** (thin shell in internal/app/ for cross-domain workflows)

### Event Bus
```go
// pkg/infra/events/bus.go
type Event interface{ Name() string }
type Handler func(ctx context.Context, event Event) error
type Bus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(eventName string, handler Handler)
}
```

Events: InvoiceCreated, PaymentRecorded, OfferWon, DocumentClassified, BankStatementImported, etc.

### Cross-domain Workflow Example
"Convert won offer to order and generate invoice":
```go
// internal/app/workflows/offer_to_invoice.go
func (w *Workflows) ConvertOfferToInvoice(ctx context.Context, offerID string) error {
    offer, _ := w.crm.GetOffer(ctx, offerID)
    order, _ := w.crm.CreateOrderFromOffer(ctx, offer)
    invoice, _ := w.finance.CreateInvoiceFromOrder(ctx, order)
    w.events.Publish(ctx, InvoiceCreatedEvent{Invoice: invoice})
    return nil
}
```

## Data Flow (Two-Phase: Pretext-Inspired)

### Write Path (Command / "Prepare" Phase)
```
Svelte UI → form submit
    → Wails JSON-RPC → Go service binding
    → internal/app/shell.go → routes to domain service
    → pkg/finance/invoice/service.go → validates, computes, persists
    → pkg/infra/db/ → ncruces SQL execution
    → pkg/infra/events/ → publishes InvoiceCreated
    → Turso CDC → replicates to cloud
    → Returns success
```

### Read Path (Query / "Layout" Phase)
```
Svelte UI → $derived state triggers fetch
    → Wails JSON-RPC → Go service binding
    → internal/app/shell.go → routes to domain query
    → pkg/finance/reporting/ → reads pre-computed aggregates (FAST)
    → Returns cached/computed data
    → Svelte $state updates → fine-grained reactivity → minimal DOM updates
```

### Serialization Boundaries

| Boundary | Format | Why |
|----------|--------|-----|
| Svelte ↔ Go (Wails IPC) | JSON (auto-generated) | Wails v3 requirement |
| Domain ↔ SQLite | Raw SQL via ncruces | No ORM overhead |
| Go ↔ Turso Cloud | Cap'n Proto | 60% smaller than JSON |
| Butler ↔ LLM API | TOON | 30% token savings |
| Disk cache/reports | Cap'n Proto | Schema evolution, compact |

## Quality Standard

"Would this make a senior Google engineer weep with elegance?"
If no → iterate. If yes → ship it.
