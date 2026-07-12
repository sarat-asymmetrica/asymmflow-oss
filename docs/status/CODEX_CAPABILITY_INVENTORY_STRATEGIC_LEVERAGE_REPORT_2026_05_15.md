# AsymmFlow Capability Inventory And Strategic Leverage Report - 2026-05-15

Scope: `the AsymmFlow repository`

Primary posture: strategic architecture and product intelligence, not implementation commentary.

Current evidence note: repo HEAD at the start of this report pass was `cfad303` on `master`, after the Business Memory source asset durability commits. The worktree also contained in-flight edits in `pkg/documents/intake/export.go`, `pkg/documents/intake/export_test.go`, `pkg/documents/intake/review_service.go`, and `pkg/documents/intake/review_service_test.go`. This report treats committed files, schemas, tests, manifests, and audited docs as stronger evidence than active worktree edits.

External market note: this report uses current public vendor evidence from Odoo, Frappe/ERPNext, Twenty CRM, monday.com, NetSuite, SAP Business One, and Retool to frame competitive direction. Those sources show a strong market shift toward metadata-driven customization, embedded AI agents, workflow automation, dashboards, document-grounded AI, and developer-extensible platforms.

## Section 1 - Executive Summary

### What AsymmFlow Actually Is

AsymmFlow is no longer best described as "an ERP app" or "a CRM refactor." The repo has evolved into an evidence-centered business operating substrate: a modular monolith with deterministic Go engines, typed schema boundaries, local-first deployment assumptions, Wails/Svelte operator surfaces, Cap'n Proto contracts, JSON/TOON agent boundaries, and emerging module manifests.

The most accurate category is:

```text
local-first evidence-centered ERP/CRM operating layer for owner-led and mid-market businesses
```

It contains the raw material for several overlapping product categories:

| Category | Fit | Why |
|---|---:|---|
| ERP/CRM application | High | Finance, CRM, sales, procurement, fulfillment, inventory, documents, banking, payroll, compliance, and reporting surfaces exist. |
| Modular monolith platform | High | `pkg/*`, `internal/viewmodel/*`, schemas, adapters, manifests, and module contracts define reusable slices. |
| Business runtime | Medium-high | Events, sync, release, health, auth, license, setup, and API/cloud packages point beyond a single app. |
| Workflow operating system | Medium | Workflows are emerging through review queues, action proposals, approvals, and manifests; a first-class workflow DSL is still missing. |
| Evidence and document engine | High | OCR, classification, Business Memory Intake, source asset registry, review/export, and Butler context packs are unusually strategic. |
| AI-native operator layer | Medium-high | Butler, TOON, draft-only agent APIs, evidence packs, and permission boundaries exist; mutation safety must harden before external positioning. |
| App/platform ecosystem substrate | Medium | Module manifests and package boundaries exist, but marketplace, plugin lifecycle, and stable extension APIs are not yet standardized. |

### Architectural Resemblance

AsymmFlow resembles a blend of:

- ERPNext/Frappe's metadata and workflow ambition, but implemented as Go authority modules rather than Python DocTypes.
- Odoo's broad business-suite surface, but with stronger local-first and evidence provenance ambitions.
- SAP Business One's SMB operational breadth, but with a more inspectable, agent-ready, self-hostable path.
- NetSuite/SuiteCloud's platform ambition, but at an earlier stage and with stronger offline/on-prem intent.
- Twenty CRM's modern open-source/custom-object posture, but aimed at operational ERP evidence rather than CRM records alone.
- Retool/Appsmith/internal-tool platforms, but with domain authority already built in rather than requiring every business rule to be rebuilt in low-code screens.
- monday.com's new AI-work-platform narrative, but with deterministic mutation gates as a core principle rather than a pure work-management board model.

### Strategic Interpretation

The strategic center of gravity is not "forms and tables." It is:

```text
messy operational reality -> canonical evidence -> deterministic business state -> operator decisions -> audited actions -> reusable intelligence
```

That makes AsymmFlow potentially differentiated in markets where:

- business records arrive through PDFs, scans, WhatsApp/email, Excel, bank statements, and operator memory;
- compliance is regional, document-heavy, and trust-sensitive;
- cloud-only ERP is not always acceptable;
- owner/operators need proof, not only dashboards;
- AI assistance must be useful without being allowed to silently mutate books, inventory, tax, or customer commitments.

### Core Strengths

| Strength | Evidence | Strategic leverage |
|---|---|---|
| Deterministic Go authority core | `pkg/finance`, `pkg/crm`, `pkg/documents`, `pkg/compliance`, `pkg/cashflow/evidence` | Serious business systems need inspectable engines before AI. |
| Typed contracts | `schemas/*.capnp`, generated Go/TS surfaces | Enables sync, agent packs, storage adapters, module boundaries, and generated UI/API surfaces. |
| Evidence-first modules | Business Memory and Cashflow Evidence manifests | Strong wedge against generic ERP dashboards and chatbots. |
| Finance/accounting spine | `pkg/finance/posting`, banking, allocations, reconciliation structs in `schemas/finance.capnp` | Cashflow, audit, accounting close, collections, and compliance all reuse this. |
| Document/OCR depth | `pkg/ocr`, `pkg/documents/intake`, classifiers, parsers | Converts messy business input into productizable workflow assets. |
| Regional compliance direction | Bahrain VAT, India GST/income tax, i18n/langpack/Arabic shaping | Defensible in India, GCC, Africa, and multilingual trading contexts. |
| Local-first/offline substrate | `pkg/sync`, Turso CDC, Wails desktop, sync normalization | A clear alternative to purely cloud SaaS for regional SMBs. |
| Agent-safety posture | Manifest forbidden operations, review services, TOON context packs | Lets AI become a coworker without becoming untrusted authority. |

### Architectural Risks

| Risk | Severity | Why it matters |
|---|---:|---|
| Partial loops can look complete | High | Kernels and manifests exist before operator-grade closed workflows. |
| Root service coupling remains | High | `App`/Wails/root services still mix queries, permissions, orchestration, and side effects. |
| Sync and events are not yet module-authoritative | High | Local-first platform claims need conflict policy and durable event semantics. |
| AI/Butler breadth can exceed authority model | Medium-high | Agent surfaces must stay inspect/explain/draft/recommend until mutation gates are proven. |
| Compliance freshness | Medium-high | Tax/regulatory engines need jurisdiction versioning and legal update process. |
| UI breadth and baseline warnings | Medium | Many screens exist; launch quality requires fewer, stronger operator loops. |
| Over-platformization before wedge closure | Medium | The repo has enough substrate; the next value is closing product loops. |

### Productization Opportunities

The strongest near-term product is not "full ERP." It is:

```text
Cashflow Evidence + Business Memory for owner-led trading/distribution companies
```

This wedge turns invoices, bank statements, receipts, supplier docs, and document chaos into an operator-facing evidence command center with follow-up, export, and deterministic posting requests.

The best longer-term positioning is:

```text
AsymmFlow is the local-first evidence OS for regional business operations: ERP records, documents, cash, inventory, compliance, and AI assistance under one deterministic trust layer.
```

## Section 2 - Complete Engine Inventory

Maturity scale used below:

- Prototype: useful code exists, but product loop is not stable.
- Foundation: tested core primitives exist, but workflow closure is incomplete.
- Emerging product: enough backend/ViewModel/UI exists for operator loop hardening.
- Mature internal: broad repo evidence and tests, but not externally productized.

### Engine Inventory Matrix

| Engine/module | Purpose and responsibilities | Inputs/outputs | Dependencies and primitives | Reusable? | Productizable? | Strategic importance | Complexity | Maturity | Coupling | Archetype |
|---|---|---|---|---:|---:|---:|---|---|---|---|
| Finance domain core | Owns invoices, payments, supplier invoices, expenses, accounts, fiscal periods, bank accounts, reports | Invoices, payments, expenses, accounts -> ledger/reporting state | `pkg/finance/domain.go`, `schemas/finance.capnp`, ViewModels | High | High | Critical | High | Foundation | Medium-high | ERP finance engine |
| Accounting posting spine | Generates balanced posting intent, coverage, trial balance gates | Source docs -> draft journals/coverage/gate status | `pkg/finance/posting`, posting tests, Wave 17 docs | High | High | Critical | High | Foundation | Medium | Compliance/accounting engine |
| Banking and reconciliation | Parses bank statements, matches transactions, supports allocation-aware reconciliation | Statements, bank lines, payments, invoices -> matches, allocations, audit logs | `pkg/finance/banking`, bank service files, finance schema allocations | High | High | Critical | High | Foundation | Medium-high | Reconciliation and cash engine |
| Cashflow Evidence | Composes finance, bank, document, and posting facts into an evidence command center | Orders/invoices/payments/bank/docs -> risk rows, missing evidence, action proposals, exports | `pkg/cashflow/evidence`, `internal/viewmodel/cashflow`, manifest | High | Very high | Critical | Medium-high | Emerging product | Medium | Evidence/reporting/workflow engine |
| CRM/customer/supplier master | Manages parties, contacts, grades, customer/supplier profiles, entity notes | Party records -> profiles, dashboards, pipeline context | `pkg/crm/domain.go`, `schemas/crm.capnp`, CRM ViewModels | High | High | High | Medium | Foundation | Medium | Business graph/CRM engine |
| Sales pipeline and offers | RFQ, opportunity, costing, quote/offer revisions, orders | RFQs/costing/products -> offers, orders, followups | `pkg/crm/pipeline`, `pkg/engines/costing_engine.go`, offer scanner, CRM schema | Medium-high | High | High | High | Foundation | High | Quote-to-cash workflow engine |
| Procurement and receive-to-pay | POs, GRNs, supplier invoices/payments, supplier issues | Supplier demand/docs -> POs, GRNs, supplier invoices/payments | CRM/finance schemas, procurement service, GRN/PO screens | Medium-high | High | High | High | Foundation | High | Procurement engine |
| Fulfillment/delivery/serial traceability | Delivery notes, shipments, serials, stock movements | Orders/stock/serials -> delivery evidence and availability | `pkg/crm/fulfillment`, serial/inventory services, CRM schema | High after extraction | High | High | High | Foundation | High | Inventory/fulfillment engine |
| Inventory and asset evidence ledger | Intended stock/evidence kernel for movements, valuation, serial/lot provenance | POs/GRNs/DNs/serials/source assets -> evidence-linked stock ledger | Current evidence in CRM schema and services; dedicated module still missing | High | Very high | High | High | Planning/foundation | High | Inventory/evidence engine |
| Business Memory Intake | Converts messy docs/messages into canonical reviewable candidates with source provenance | Emails, PDFs, scans, Excel, inbox records -> candidates, fields, links, context packs | `pkg/documents/intake`, `schemas/documents.capnp`, manifest | Very high | Very high | Critical | Medium-high | Emerging product | Low-medium | Document/evidence/memory engine |
| Source asset registry | Tracks source IDs, kind, path, hash, privacy class, status, candidate IDs, audit refs | Source observations -> durable source registry | `source_registry.go`, repository/storage adapter, manifest | Very high | High | Critical | Medium | Foundation to emerging | Low | Provenance registry |
| Document classifier | Classifies document types and produces confidence/explanation | File metadata/text/OCR -> type, confidence, routing | `pkg/documents/classifier`, root classifier | High | High | High | Medium | Foundation | Medium | Document engine |
| OCR and extraction pipeline | Extracts text/fields/tables, routes between local/provider/GPU paths | PDFs/images/scans -> text, pages, fields, metrics | `pkg/ocr`, `pkg/documents/ocr`, orchestrator, ksum, octonion, sparse OCR | High | Very high | High | Very high | Foundation | Medium-high | OCR/data pipeline |
| PDF/document generation | Produces invoices, quotes, reports, document artifacts | Business records/templates -> PDFs/reports | `pkg/engines/pdf_generator.go`, document schemas | Medium-high | High | High | Medium | Foundation | Medium | Document generation/reporting |
| Excel/email/import parsers | Parses costing sheets, MSG/EML, bank/import files | Excel/email/files -> normalized records/evidence | `pkg/documents/excel`, `pkg/documents/email`, import handlers | High | High | High | Medium | Foundation | Medium | Import/integration substrate |
| Butler AI and agent layer | Provides chat, intent routing, TOON context, reports, grounded fastpaths | User intent/business context -> explanations, drafts, reports, proposed actions | `pkg/butler`, `pkg/toon`, service_butler, manifests | Medium-high | High | High | High | Prototype/foundation | High | AI assistant/orchestration layer |
| Prediction/payment intelligence | Payment risk, customer encoding, win probability, discount guidance | Customer/payment/history features -> risk/probability/suggestions | `pkg/butler/prediction`, `pkg/engines/predictor.go`, VQC/math | Medium | Medium-high | Medium-high | High | Prototype/foundation | Medium | Analytics/prediction engine |
| Compliance packs | Jurisdiction rules for VAT/GST/income tax and invoice validation | Transactions/tax profiles -> validations, reports, hooks | `pkg/compliance/bahrain`, `pkg/compliance/india`, hooks, compliance VM | High | Very high | High | Medium-high | Foundation | Low-medium | Compliance/localization engine |
| Localization/i18n | Language packs, Arabic shaping, currency/date/format direction | Locale/text/amounts -> translated/shaped/rendered UI/doc output | `pkg/i18n`, `pkg/engines/langpack.go`, `arabic_shaper.go`, frontend i18n | High | High | High | Medium | Foundation | Medium | Localization engine |
| Sync/local-first/CDC | Replication, collaboration, normalization, Turso CDC | Local records/events -> synced records/CDC/readiness | `pkg/sync`, `pkg/sync/turso`, adapter sync, root sync services | High | Very high | Critical | Very high | Foundation | High | Sync/runtime infrastructure |
| Infra events/health/OTel/release | Event bus, health monitor, observability, release metadata | Domain events/metrics/build data -> readiness and support data | `pkg/infra/events`, `health`, `otel`, `release` | High | Medium-high | High | Medium | Foundation | Low-medium | Runtime/observability layer |
| Auth, license, approvals, security | Permissions, licensing, sovereign auth, delete approvals, access gates | Users/tokens/actions -> allow/deny/audit | `pkg/infra/auth`, `pkg/infra/license`, `pkg/sovereign_auth`, root approval services | Medium-high | High | Critical | High | Foundation | Medium-high | Auth/governance engine |
| ViewModel layer | Converts authority state into UI-ready state and commands | Domain facts -> display models/commands/validation status | `internal/viewmodel/*` | High | Medium | High | Medium | Foundation | Medium | MVVM adapter layer |
| Component/action inventory system | Reusable UI primitives and action audit ledgers | UX actions/components -> consistent operator surfaces and QA | `frontend/src/lib/components/ui`, docs/testing inventories | Medium-high | Medium | Medium-high | Medium | Foundation | Medium | Product UI infrastructure |
| Graph/SSOT builder | Entity graph and single-source-of-truth import support | Entities/import rows -> graph/relationships | `pkg/graph`, `pkg/data/ssot_importer.go` | High | Medium-high | Medium-high | Medium | Foundation | Low-medium | Business graph/data substrate |
| API/cloud/multitenant substrate | REST/API server, Docker/Kubernetes/Prometheus, regime service | Requests/services -> API responses/cloud deployment | `pkg/api`, Dockerfile, k8s, middleware | Medium | High later | Medium-high | High | Prototype/foundation | Medium | Cloud/runtime layer |
| Setup/deployment/training readiness | Setup wizard, release scripts, support docs, user guides | Install/config/training data -> deployable app/support evidence | `pkg/setup`, `pkg/infra/release`, docs/user-guides | Medium | High | High | Medium | Foundation | Medium | Launch operations layer |
| Math/optimization substrate | Williams, quaternion, trident, prism, VQC, compression, UI alchemy | Batch/workflow/context signals -> routing, classification, compression, visual/system primitives | `pkg/math`, `pkg/vqc`, `pkg/compression`, `pkg/ui_alchemy` | High with status labels | Medium-high | Medium-high | High | Prototype/foundation | Medium | Optimization/math substrate |
| Integrations | COM automation, Graph client, OneDrive/Tally docs, external OCR providers | External systems/files -> records/evidence/actions | `pkg/integration`, `pkg/sync/onedrive`, `pkg/sync/tally`, `pkg/ocr/orchestrator` | Medium | High | Medium-high | High | Prototype/foundation | Medium | Integration substrate |

### Hidden Gems

| Hidden gem | Why it is underappreciated | Strategic use |
|---|---|---|
| Business Memory source asset registry | It looks like plumbing, but it is the provenance spine for every future document-driven module. | Turns files, scans, folders, emails, and messages into durable evidence objects. |
| Posting coverage and trial-balance gates | These are not only accounting utilities; they are trust gates for any automation that touches books. | Makes AI/draft workflows safe by showing what can and cannot be posted. |
| UI/backend action inventories | Usually ignored after QA, but they can become a product-readiness and permission-surface generator. | Proves visible actions map to deterministic commands, tests, and permissions. |
| TOON context packs | A compact bridge between deterministic state and agent reasoning. | Allows AI to inspect and draft from bounded context instead of scraping the app. |
| Arabic shaping plus i18n | Most early ERPs treat localization as translation only. Script rendering matters for GCC/Africa deployment trust. | Makes regional compliance and operator UX credible. |
| Turso CDC/sync normalization | It is easy to treat sync as infrastructure, but local-first is a strategic product wedge. | Supports branch/offline deployments and later cloud/on-prem hybrid. |
| OCR observability and advanced image processing | These look experimental, but document quality and confidence are central to operator trust. | Gives Business Memory and compliance products explainable ingestion quality. |

## Section 3 - Domain Model Analysis

### Core Business Objects

The domain model spans a full operating company rather than a narrow CRM:

| Domain | Entities visible in schemas/packages | Lifecycle role |
|---|---|---|
| Parties | CustomerMaster, SupplierMaster, contacts, profiles, grades, notes, issues | The business graph root for sales, procurement, credit, service, and evidence. |
| Sales | RFQs, opportunities, costing sheets, offers, offer items, revisions, follow-ups, orders | Quote-to-cash and customer commitment management. |
| Procurement | Purchase orders, PO items, GRNs, supplier invoices, supplier payments | Receive-to-pay and supplier commitment management. |
| Inventory/fulfillment | Products, warehouses, stock movements, serial numbers, shipments, delivery notes | Physical-world operational truth and dispute evidence. |
| Finance | Invoices, credit notes, payments, chart of accounts, journal entries, fiscal periods, expenses, recurring expenses, payroll-adjacent records | Monetary truth, accounting close, cashflow, tax, audit. |
| Banking | Bank accounts, statements, lines, allocations, reconciliation logs, outstanding cheques, deposits in transit, FX revaluation | Cash truth and reconciliation evidence. |
| Documents/evidence | OCR requests/results, classification results, Business Memory candidates, source refs, context packs, review records | Converts external mess into reviewable operational memory. |
| Compliance | VAT/GST/income-tax calculations, invoice validation, event hooks | Jurisdiction trust and readiness. |
| Users/governance | Auth, permissions, licenses, delete approvals, audit/action ledgers | Controls who can mutate business truth. |
| Runtime | Events, sync records, health, release manifests, API responses, setup state | Deployment and operational control plane. |

### Relationship Map

The strongest inferred business graph is:

```text
Customer/Supplier
  -> RFQ/Opportunity
  -> Costing
  -> Offer/Revision
  -> Order
  -> Purchase/GRN/Supplier Invoice
  -> Delivery/Serial/Stock Movement
  -> Customer Invoice/Payment
  -> Bank Allocation/Reconciliation
  -> Posting Coverage/Journal Draft
  -> Compliance Validation
  -> Evidence Pack/Audit Trail
```

Business Memory sits beside the graph, not underneath it:

```text
Source Asset -> Intake Candidate -> Review Decision -> Suggested Link -> Deterministic Service Request -> Business Record
```

This is strategically correct. It prevents OCR/AI from directly becoming business truth.

### Lifecycle Flows

| Flow | Current evidence | State of abstraction |
|---|---|---|
| Lead/RFQ to quote | CRM schema, costing engine, offer scanner, screens | Strong domain breadth, but revision integrity must remain a launch gate. |
| Quote to order to delivery | Offers/orders/shipments/delivery notes/serials | Present, but inventory kernel should be separated from CRM/root services. |
| Purchase to receipt to supplier invoice | PO/GRN/supplier invoice/payment | Present, but receive-to-pay evidence pack is not yet productized. |
| Invoice to payment to bank match | Finance schema, payment, banking/reconciliation, allocations | Strong substrate for Cashflow Evidence. |
| Source document to reviewed candidate | Business Memory Intake, source registry, review service/export | Emerging product loop and likely best wedge with Cashflow. |
| Record to posting intent | Posting spine, trial-balance gate, coverage report | Strategic finance trust layer. |
| Transaction to tax/compliance validation | Compliance packages and hooks | Good kernel foundation; needs filing/export and law-version governance. |
| Local state to sync/cloud/support | Sync, CDC, release/health/OTel, support docs | Foundation exists; module-aware sync semantics are missing. |

### Generalized Workflow Potential

Evidence supports generalized workflows in an emerging form:

- review queues in Business Memory;
- action proposals in Cashflow Evidence;
- delete approvals and permission gates;
- UI/backend action inventories;
- manifests describing inputs, outputs, agent-safe APIs, permissions, and tests;
- ViewModels exposing commands rather than raw tables.

However, it is not yet a low-code/no-code platform in the Frappe sense. Frappe's central primitive is metadata-driven DocType, where a record model can become a table, form, API, list, calendar, kanban view, and workflow configuration. AsymmFlow's current central primitive is closer to:

```text
typed deterministic module + evidence inputs + review/action gates + ViewModel surface
```

That is less flexible today but more trustworthy for compliance-heavy operations. The opportunity is not to copy DocType. The opportunity is to create a stricter module primitive:

```text
Module = schema + pure kernel + repository + events + permissions + ViewModel + evidence pack + agent-safe API + verification gate
```

### Event Sourcing, Sync, And Deterministic State

Evidence shows event and sync primitives, but not full event sourcing:

| Pattern | Evidence | Assessment |
|---|---|---|
| Event bus | `pkg/infra/events` | Useful infra, but domain event ownership needs standardization. |
| CDC | `pkg/sync/turso/cdc.go` | Important for local-first/cloud sync, still table/adapter oriented. |
| Audit logs | banking audit, Business Memory audit refs, docs/compliance dossiers | Present in slices, needs common envelope. |
| Deterministic state handling | posting gates, review services, agent forbidden operations | Strong principle, not yet universal enforcement. |
| Offline conflict handling | sync normalization, local-first posture | Foundation exists, but module-level conflict policies are not yet canonical. |

Strategic inference: AsymmFlow should avoid claiming event sourcing. It should claim deterministic module state with evidence logs, then build toward evented module envelopes.

## Section 4 - Document And Data Pipeline Analysis

### Pipeline Inventory

| Pipeline | Inputs | Transformations | Outputs | Strategic meaning |
|---|---|---|---|---|
| OCR pipeline | PDFs, images, scans | local/provider OCR, GPU preprocessing, table detection, image enhancement, metrics | OCRResult, pages, fields, confidence, errors | Converts paper/scans into usable business evidence. |
| Business Memory Intake | inbox docs, OCR maps, source assets, messages/emails | normalization, classification mapping, field status, review queue, source registry | candidates, context packs, review records, exports | General document-to-business-memory engine. |
| Finance documents | invoices, credit notes, POs, supplier invoices, bank files | parsing, matching, posting coverage, allocations | records, draft journals, reconciliation evidence | Compliance and accounting proof. |
| Sales documents | RFQs, costing sheets, offers, PDFs | costing parser, offer scanner, PDF generation | quotes, orders, revision artifacts | Quote-to-cash execution and customer trust. |
| Export/replay | Business Memory bundles, Cashflow Evidence packs | JSON/TOON assembly, source/audit inclusion | support bundle, agent brief, replayable review state | Operator support and audit moat. |
| Sync/import | SSOT importer, table normalization, Turso CDC | normalization, upsert/CDC | replicated state/support traces | Local-first and cloud/on-prem bridge. |

### Serialization Boundaries

| Boundary | Role | Current strategic assessment |
|---|---|---|
| Cap'n Proto | Durable/cross-module schema contracts for finance, CRM, documents, butler, infra, sync | Strong choice for stable contracts and generated surfaces. Must be guarded from churn. |
| JSON | Manifest, exports, frontend payloads, support bundles | Good for interoperability and operator-readable exports. |
| TOON | Compact agent context and evidence summaries | Strong AI boundary because it encourages bounded, structured context. |
| GORM/SQLite | Local persistence and adapters | Practical for desktop/local-first. Needs module-aware migration and conflict policy. |
| Wails generated bindings | Desktop UI runtime bridge | Useful but easy to churn; should stay downstream of domain contracts. |

### Is Document Architecture Differentiated?

Yes, if the team completes the provenance loop.

Most SMB ERP products support attachments. Some support OCR. Fewer treat source evidence as a first-class durable object that can be:

- hashed or identified;
- reviewed;
- linked to candidate business records;
- included in exports;
- cited by an AI assistant;
- replayed or audited later;
- governed by privacy class and permissions.

The repo is moving toward that architecture through Business Memory Intake, source registry, context packs, and review/export services. That is strategically stronger than "upload attachments to an invoice."

### Compliance-Heavy Workflow Fit

The document pipeline is particularly suited for compliance-heavy markets because it supports:

- invoices and supplier invoices as evidence;
- bank statements and allocations;
- source refs and audit refs;
- jurisdiction validation hooks;
- Business Memory candidate review before mutation;
- posting coverage and trial-balance checks;
- export packs for accountant/support review.

Missing pieces before strong external claims:

- common audit envelope;
- module-aware event persistence;
- immutable evidence pack signatures or hashes;
- document retention policy;
- legal version metadata for compliance rules;
- operator-visible correction trail across modules.

## Section 5 - Localization And Compliance Analysis

### Localization Evidence

| Capability | Evidence | Depth assessment |
|---|---|---|
| Language packs | `pkg/i18n/messages/ar.json`, `en.json`, `es.json`, `fr.json`, `hi.json` | Foundational, broad language intent. |
| Arabic script shaping | `pkg/engines/arabic_shaper.go` | Deeper than translation; important for GCC document/UI quality. |
| Langpack engine | `pkg/engines/langpack.go` | Useful abstraction for regional packs. |
| Currency and FX | finance schemas include exchange rates, FX revaluation, BHD context in historical docs | Practical finance localization. |
| Bahrain VAT | `pkg/compliance/bahrain/vat.go` | Strong GCC wedge. |
| India GST/income tax | `pkg/compliance/india/gst.go`, `income_tax.go` | Strong India wedge, needs law-update process. |
| Tax/invoice validation hooks | `pkg/compliance/hooks.go` | Good direction for evented validation. |
| User guides and compliance dossiers | `docs/compliance/*`, `docs/user-guides/*` | Useful operator trust layer. |

### Superficial Or Architectural?

Localization is architectural in finance/compliance and partially architectural in UI/document rendering.

It is not just translation because:

- tax engines exist as code;
- finance schemas include bank, VAT, FX, and allocation concepts;
- Arabic shaping is a rendering primitive;
- compliance hooks connect validation to business events;
- regional deployment/on-prem/offline assumptions influence sync/runtime strategy.

It remains incomplete because:

- legal rule versioning is not fully formalized;
- filing/export workflows are not yet first-class products;
- locale-specific document templates and e-invoice formats need hardening;
- UI translations and right-to-left behavior need launch-grade verification;
- Africa/GCC/India specific workflows need customer acceptance scenarios.

### Market Suitability

| Region | Suitability | Why | Missing |
|---|---:|---|---|
| India | High potential | GST/income tax kernels, multilingual intent, offline/on-prem relevance, document-heavy SMBs | GST filing/export maturity, e-invoice/e-way bill decisions, Hindi/regional-language UX depth |
| GCC/Gulf | High potential | Bahrain VAT, Arabic shaping, BHD/FX/banking relevance, trading/distribution workflows | GCC country packs, Arabic RTL QA, e-invoice/ZATCA-style abstractions if targeting KSA |
| Africa | Medium-high potential | Offline-first, document evidence, multilingual deployment, owner-led SMB fit | Country-specific tax packs, mobile-first/offline workflow proof, payment integrations |
| Multilingual deployments | Medium-high potential | i18n packages and script shaping | Full UI coverage, locale QA, translation governance |

Strategic conclusion: localization is a defensibility vector, not a checklist. The emerging-market wedge is credible if it is paired with evidence, offline operation, and compliance packs.

## Section 6 - Platformization Potential

### What Already Exists

| Platform primitive | Evidence | Strategic value |
|---|---|---|
| Typed module schemas | `schemas/*.capnp` | Stable contracts across Go, TS, sync, agent context, adapters. |
| Pure kernels | posting, cashflow evidence, compliance, math, document normalizers | Testable reusable engines. |
| Storage adapters | `pkg/adapter/*`, Business Memory storage | Separation between contracts and persistence. |
| ViewModel adapters | `internal/viewmodel/*` | UI surfaces can become predictable and testable. |
| Manifests | `docs/modules/*.manifest.json` | Early module metadata and launch-readiness contract. |
| Event bus and CDC | `pkg/infra/events`, `pkg/sync/turso` | Runtime coordination and local-first support. |
| Agent context format | `pkg/toon`, Business Memory/Cashflow agent APIs | AI can consume bounded evidence. |
| UI component library | reusable Svelte components and action inventories | Consistent operator surfaces and QA automation. |
| Release/support tooling | release manifests, scripts, docs/testing ledgers | Pilot readiness and supportability. |

### What Is Missing For Platform Status

| Missing abstraction | Why it matters | Recommended form |
|---|---|---|
| Module runtime lifecycle | Install/enable/disable/version/migrate modules | `module.json` plus Go registry and migration contract. |
| Durable event envelope | Cross-module subscriptions and sync need reliable event semantics | Typed event envelope with actor, source, correlation, version, replay policy. |
| Module-aware sync policy | Offline/cloud conflicts cannot be table-only | Per-module conflict rules, merge strategies, and operator conflict surfaces. |
| Workflow DSL or state-machine primitive | Review/action flows repeat across modules | Conservative state-machine spec for review/approval/action proposals. |
| Permission namespace standard | Agent and human authority must be uniform | `module:resource:action` with actor class and mutation tier. |
| Extension API | Marketplace/app ecosystem needs stable seams | Schema, repository, event, ViewModel, UI slot, and agent-safe API contracts. |
| Tenant boundary | Cloud/SaaS requires isolation | Tenant ID propagation through schemas, storage, events, sync, audit. |
| Data model customization layer | Low-code competitors win on custom objects | Start with custom fields/evidence tags, not arbitrary table mutation. |
| Productized deployment plane | On-prem/cloud needs upgrades/backups/support | Admin readiness dashboard, support bundle, update policy. |

### Can It Become Low-Code/No-Code?

Yes, but it should not start by copying generic low-code platforms.

Frappe/ERPNext's market advantage is metadata-driven customization: DocTypes, forms, reports, workflows, permissions, APIs, and multi-site deployment are all platform primitives. Twenty CRM is pushing custom data models, custom fields, object relationships, workflow triggers, AI agents, dashboards, RBAC, and GraphQL/REST APIs as core CRM capabilities. Odoo is adding AI over documents, server actions, import templates, livechat lead generation, and dashboards.

AsymmFlow's differentiated platform path should be:

```text
evidence-aware modules first, custom metadata second
```

Do not let users create arbitrary tables before the core evidence and authority model is stable. Instead, expose:

- custom evidence tags;
- custom review states;
- custom source categories;
- custom dashboard cards over approved read models;
- custom export templates;
- custom workflow thresholds for approvals and follow-ups.

That gives 80 percent of perceived flexibility without weakening deterministic authority.

## Section 7 - Productization Opportunities

### Product Opportunity Matrix

| Product | Target customer | Deployment | Pricing potential | Difficulty | Strategic fit | Moat potential |
|---|---|---|---:|---|---:|---:|
| Cashflow Evidence Command Center | Owner-led distributors, trading firms, service SMEs with messy receivables | Desktop local-first plus optional cloud sync | $99-$499/mo SMB; $1k+/mo assisted ops | Medium | Very high | High |
| Business Memory / Evidence Inbox | SMEs drowning in PDFs, scans, emails, WhatsApp docs | Desktop/on-prem/cloud hybrid | $49-$299/mo standalone; bundled in ERP | Medium | Very high | High |
| Bank Reconciliation and Allocation Workbench | Accountants, trading firms, finance teams | Local-first with bank import | $99-$399/mo; accountant seats | Medium | Very high | Medium-high |
| GCC/India Compliance Evidence Pack | Bahrain/India-first businesses, regional accountants | Self-hosted or managed cloud | $199-$999/mo plus filing support | Medium-high | High | High if rules stay fresh |
| Offline-first Trading ERP | Import/export, distribution, industrial suppliers | Desktop/on-prem with optional sync | $299-$2k/mo by branch/users | High | High | High |
| Inventory and Asset Evidence Ledger | Distributors, equipment/service businesses, warehouses | Local-first/on-prem | $199-$999/mo | High | High | High |
| Quote Revision Integrity Pack | Sales teams where quote references/revisions cause disputes | Add-on to CRM/ERP | $49-$199/mo | Low-medium | High | Medium |
| AI-safe ERP Butler | Existing AsymmFlow users, accountants, managers | Embedded assistant | Add-on $49-$299/user/mo | Medium-high | High | Medium-high |
| OCR/Document Intelligence API | Regional ERP implementers, accountants, vertical SaaS | API/self-hosted worker | usage-based or $500+/mo | High | Medium-high | Medium |
| Support Bundle and Readiness Dashboard | ERP implementers, on-prem customers | Admin module | $99-$499/mo or support-plan differentiator | Medium | Medium-high | Medium-high |
| Compliance-ready Document Vault | SMEs needing source-proof retention | Local-first/cloud backup | $49-$299/mo | Medium | High | Medium-high |
| Agent-safe Internal Tool Runtime | Technical teams needing deterministic business tools | Self-hosted platform | $20-$100/user/mo or enterprise | Very high | Medium later | Medium-high |
| Regional ERP Implementation Kit | Implementation partners in India/GCC/Africa | Toolkit plus managed services | Partner revenue; $5k-$50k/project | Medium | High | High through services |
| Collections and Follow-up Copilot | Sales/finance teams with overdue AR | SaaS/local hybrid | $49-$199/user/mo | Medium | Very high | Medium |
| Accountant Evidence Export Portal | Businesses and external accountants | Cloud portal or export bundle | $99-$499/mo | Medium | High | Medium-high |

### Strongest GTM Wedges

#### 1. Cashflow Evidence + Business Memory

Target: owner-led distributors/trading firms where receivables, bank statements, supplier docs, and customer commitments are fragmented.

Why it wins:

- Immediate ROI: cash collection, reconciliation time, accountant back-and-forth.
- Closed workflow: ingest docs -> review evidence -> identify missing proof -> draft follow-up/export -> request deterministic posting.
- Differentiation: evidence provenance plus AI-safe explanation.
- Existing repo fit: strongest current path with committed code and manifests.

#### 2. Regional Compliance Evidence Pack

Target: Bahrain/GCC and India SMEs with tax anxiety and document-heavy workflows.

Why it wins:

- Compliance is pain with willingness to pay.
- Current repo already contains Bahrain and India compliance kernels.
- Evidence packs can become accountant-facing exports.
- Localization and offline-first matter in these markets.

Risk: law freshness and filing correctness require human/legal process.

#### 3. Offline-first Trading ERP

Target: businesses like Acme Instrumentation where sales, procurement, stock, finance, bank, and documents all meet.

Why it wins:

- The existing repo was born from real client feedback.
- SAP Business One/ERPNext/Odoo alternatives often become implementation-heavy.
- AsymmFlow can specialize in evidence, regional compliance, and local-first operations.

Risk: too broad for first external launch. Use as long-term bundle, not first wedge.

### Fastest Monetizable Products

| Rank | Product | Why fast |
|---:|---|---|
| 1 | Business Memory Evidence Inbox | It can start as review/export without mutating ERP truth. |
| 2 | Cashflow Evidence Command Center | Existing finance/bank/document primitives are already available. |
| 3 | Quote Revision Integrity Pack | Narrow, client-validated, high trust value. |
| 4 | Bank Reconciliation Workbench | Existing schema and matching infrastructure provide leverage. |
| 5 | Compliance Evidence Export | High willingness to pay, but needs careful legal/versioning discipline. |

### Highest Strategic Leverage Products

| Rank | Product | Why leverage is high |
|---:|---|---|
| 1 | Evidence-centered ERP runtime | Every module benefits from source provenance, review, audit, and exports. |
| 2 | Module-aware local-first sync | Hard to copy; supports on-prem, cloud, branch, and offline. |
| 3 | Agent-safe business operating layer | AI becomes trusted because deterministic services retain authority. |
| 4 | Regional compliance platform | Differentiates against generic global SaaS in India/GCC/Africa. |
| 5 | Inventory/asset evidence ledger | Connects physical operations to finance, compliance, and documents. |

## Section 8 - Architectural Assessment

### Modular Monolith Quality

Assessment: appropriately ambitious, still uneven.

The repo is making the right architectural move: modular monolith first, distributed system later. For ERP/CRM, this is preferable to premature microservices because transactions, local deployment, auditability, and operator support benefit from one authoritative runtime.

Strengths:

- package-level domain separation is visible;
- schemas and adapters reduce implicit coupling;
- ViewModels enforce UI separation in newer modules;
- tests exist for pure kernels and adapters;
- manifests are beginning to describe module authority.

Weaknesses:

- root services and Wails bindings still hold workflow orchestration;
- inventory is split across CRM/root services rather than owning its own kernel;
- sync/events are not yet first-class module contracts;
- UI breadth exceeds launch-grade loop depth;
- generated surfaces can mask drift.

### Separation Of Concerns

| Layer | Current state | Assessment |
|---|---|---|
| Pure kernels | Strong in finance posting, cashflow evidence, compliance, intake normalizers | Good and growing. |
| Domain services | Mixed: some packages, some root services | Needs extraction by product loop, not broad refactor. |
| Storage adapters | Improving through `pkg/adapter/*` and Business Memory storage | Good direction. |
| ViewModels | Present for major domains | Should become mandatory for new operator surfaces. |
| UI | Broad, functional, but too many screens | Focus on strongest loops. |
| Agent adapters | Present but broad | Needs strict mutation denial tests and permission model. |
| Sync/runtime | Useful primitives | Needs module semantics. |

### Scalability And Deployment

AsymmFlow is not optimized like a hyperscale SaaS backend. That is fine. Its stronger deployment posture is:

- desktop/local-first for trust and offline use;
- on-prem or branch deployments;
- optional cloud/API surfaces;
- sync/CDC for replication;
- support bundle and release tooling for practical operations.

This is strategically more useful for India/GCC/Africa SMEs than pure multi-tenant SaaS alone. Multi-tenant cloud can come later, but it must not erase the local-first advantage.

### Overengineered, Underengineered, Or Appropriate?

The architecture is appropriately ambitious for the intended category, but it can become overengineered if it keeps adding substrate before closing loops.

The codebase has enough platform foundation for the next cycle. The next value is:

```text
close Business Memory -> close Cashflow Evidence -> preflight Inventory Evidence -> harden agent-safe surfaces -> package pilot
```

### Runtime Consistency

Current runtime consistency is strongest where pure kernels and review services exist. It is weakest where old root services still directly mutate data and where sync/event semantics are table-oriented.

Recommendation: enforce the "authority ladder" for every new module:

```text
agent/UI intent -> ViewModel command -> domain service -> pure kernel -> repository -> event/audit/export
```

## Section 9 - Competitive Positioning

### Current Market Signals

| Competitor/source | Current direction | Implication for AsymmFlow |
|---|---|---|
| Odoo | Odoo 19 adds document-grounded AI, AI server actions/field updates, livechat AI lead generation, import templates, and dashboard improvements. Source: [Odoo 19 release notes](https://www.odoo.com/odoo-19-release-notes). | AI over business records is becoming table stakes. AsymmFlow should differentiate through evidence provenance and deterministic authority gates. |
| ERPNext/Frappe | Frappe emphasizes metadata-driven low-code DocTypes, forms, views, workflows, permissions, APIs, multi-site hosting, and marketplace/cloud. Sources: [ERPNext/Frappe framework](https://frappe.io/erpnext/framework), [Frappe Framework](https://frappe.io/framework). | Platformization requires metadata, workflows, permissions, and hosting story. AsymmFlow should choose stricter evidence-aware modules, not generic DocType cloning. |
| Twenty CRM | Twenty emphasizes custom objects/fields/relationships, table/kanban/calendar views, no-code workflows, email/calendar sync, permissioned AI agents, dashboards, RBAC/audit logs, GraphQL/REST. Source: [Twenty key features](https://docs.twenty.com/getting-started/key-features). | Modern CRM buyers expect custom data models and AI agents. AsymmFlow can win by extending that to ERP evidence, not CRM only. |
| monday.com | In May 2026 monday positioned itself as an AI Work Platform where people and agents operate across work management, CRM, service, and dev under permissions/governance. Source: [monday.com AI Work Platform announcement](https://ir.monday.com/news-and-events/news-releases/news-details/2026/monday-com-Goes-All-In-on-AI-From-Work-Management-Platform-to-AI-Work-Platform/default.aspx). | "Agents in workflows" is becoming mainstream. AsymmFlow needs a sharper claim: agents can draft and explain, deterministic modules approve and mutate. |
| NetSuite | NetSuite/SuiteCloud is adding AI-assisted development/customization with agent skills. Source: [Oracle NetSuite SuiteCloud AI announcement](https://www.oracle.com/latam/news/announcement/netsuite-brings-ai-powered-speed-accuracy-development-on-suitecloud-2026-04-28/). | Enterprise ERP platforms are making customization faster through AI. AsymmFlow should support Codex/agent-safe module development with specs and verification gates. |
| SAP Business One | SAP positions Business One as SMB ERP across accounting, purchasing, inventory, sales/CRM, reporting, analytics, and mobile/customizable deployment. Source: [SAP Business One](https://www.sap.com/products/erp/business-one.html). | AsymmFlow overlaps SMB operational breadth, but can differentiate with local-first evidence and regional compliance. |
| Retool | Retool emphasizes internal apps, workflows, AI agents, monitoring/evaluation, self-hosting, and enterprise operations. Source: [Retool agents and workflows](https://retool.com/resources/collections/agents-and-workflows). | Internal-tool builders own speed; AsymmFlow should own domain correctness and evidence-backed mutation. |

### Conceptual Comparison

| Platform | Overlap | Divergence | Where AsymmFlow can be uniquely powerful |
|---|---|---|---|
| Zoho | CRM, automation, analytics, suite breadth, AI assistant | Zoho is broad SaaS suite; AsymmFlow is local-first/evidence-centric | Regional on-prem evidence workflows and deterministic AI gates. |
| ERPNext | Open-source ERP, workflows, customization, accounting/inventory/CRM | ERPNext is DocType/metadata-first; AsymmFlow is engine/evidence-first | Compliance-heavy document-to-record workflows with stronger Go kernels. |
| Odoo | Broad ERP apps, import templates, AI actions, dashboards | Odoo is app marketplace/cloud/service ecosystem; AsymmFlow can specialize | Local-first regional ERP with evidence provenance and agent-safe authority. |
| SAP Business One | SMB ERP operational breadth | SAP B1 is established, partner-driven, less AI-native/local evidence focused | Faster vertical specialization and source-backed workflows. |
| NetSuite | SuiteCloud platform, ERP, customization, analytics | NetSuite is cloud enterprise suite | On-prem/local-first for smaller/regional firms and evidence-driven workflows. |
| Refrens | Invoicing, accounting-adjacent SMB workflows | Refrens is narrower and cloud-first | Broader operational graph: docs, inventory, banking, compliance, AI. |
| monday.com | Workflows, CRM, AI agents, collaboration | monday is board/work-management-first | Deterministic ERP authority and financial/inventory evidence. |
| Retool/Appsmith | Internal tools, workflow automation, integrations | They provide builders, not native ERP truth | AsymmFlow has domain engines and can expose safe customization later. |
| Notion/Airtable | Flexible records, views, lightweight workflows | They are flexible knowledge/data workspaces | AsymmFlow can handle accounting, compliance, and physical operations. |
| Linear | Workflow quality, speed, opinionated UX | Linear is dev-workflow focused | AsymmFlow can learn from opinionated UX: fewer stronger loops. |

### Competitive White Space

The white space is:

```text
AI-assisted, evidence-centered, local-first ERP for regional businesses where documents, cash, inventory, and compliance must be inspectable.
```

This is not crowded. Existing incumbents have breadth; newer AI work platforms have flexibility; internal tool builders have speed. Few combine:

- on-prem/local-first operation;
- regional tax/compliance;
- document provenance;
- cash/bank/accounting gates;
- inventory/serial evidence;
- AI assistance with deterministic mutation boundaries.

## Section 10 - Strategic Recommendations

### Immediate Recommendations

1. Make Cashflow Evidence + Business Memory the first public narrative.
2. Finish durable source registry exports/replay and operator provenance surfacing.
3. Add allocation-aware Cashflow Evidence as the next high-value loop.
4. Preserve the Acme Instrumentation rough track as acceptance-scenario evidence, not a file-level source of truth.
5. Convert agent-safety rules into tests across Business Memory and Cashflow.
6. Build a module readiness dashboard before adding more screens.
7. Treat Inventory/Asset Evidence Ledger as the next pure-kernel preflight, not immediate UI mutation.
8. Standardize module manifests into a real module contract only after two loops close.

### What Not To Do

| Do not | Reason |
|---|---|
| Do not market as "full ERP replacement" first | The repo has breadth, but closed-loop depth is still emerging. |
| Do not copy Odoo/ERPNext breadth feature-for-feature | That path burns time and hides differentiation. |
| Do not let Butler mutate accounting, inventory, tax, or records directly | Agent trust is the product. Preserve deterministic authority. |
| Do not build generic low-code too early | Evidence-aware modules are the defensible core. |
| Do not over-index on mathematical language in product copy | Keep math substrate internal unless a user benefit is concrete. |
| Do not claim compliance readiness without law-version process | Compliance products require maintenance discipline. |
| Do not broaden UI before closing operator loops | Fewer strong workflows will beat many partial screens. |

### What To Simplify

- Collapse roadmap language around two flagship loops: Business Memory and Cashflow Evidence.
- Make every module explain its closed loop in one line.
- Reduce agent vocabulary to four safe verbs: inspect, explain, draft, recommend.
- Use one common evidence object shape across modules.
- Use one readiness card pattern for module state, warnings, exports, and next action.

### What To Expose Publicly

Expose:

- evidence inbox;
- cashflow evidence command center;
- local-first/on-prem deployment;
- regional compliance packs;
- source-backed AI assistant;
- accountant/support export bundles;
- deterministic posting and review gates;
- offline/branch deployment readiness.

Hide or keep internal:

- raw module manifest complexity;
- math substrate details unless tied to measurable routing/optimization;
- generated schema/binding internals;
- broad platform/marketplace ambitions until extension boundaries stabilize;
- experimental OCR/math packages that are not tied to an operator outcome.

### Best Wedge Product

Best wedge:

```text
Cashflow Evidence for trading/distribution companies, powered by Business Memory.
```

The first customer story:

```text
Drop in bank statements, invoices, supplier docs, receipts, and messy files.
AsymmFlow shows what cash is exposed, what proof is missing, what can be followed up, what can be exported to the accountant, and what deterministic posting requests are safe to create.
```

This is concrete, painful, valuable, and aligned with the repo.

### Best Onboarding Path

1. Import company/customer/supplier basics.
2. Drop documents and bank statements into Business Memory.
3. Review candidates and source provenance.
4. Open Cashflow Evidence.
5. Inspect missing evidence and bank allocation gaps.
6. Draft follow-ups or accountant exports.
7. Request deterministic draft postings only after review.
8. Export support/accountant evidence bundle.

### Best Long-Term Positioning

AsymmFlow should become:

```text
the evidence OS for local-first business operations
```

Expanded positioning:

```text
AsymmFlow is a modular, local-first business operating platform for regional companies that need ERP-grade records, document evidence, cashflow clarity, inventory traceability, compliance readiness, and AI assistance without surrendering deterministic control.
```

## Hidden Strengths

1. The repo has a real acceptance benchmark from `ph_holdings`, not only abstract architecture.
2. Business Memory is a platform primitive disguised as a document module.
3. Cashflow Evidence can unify finance, documents, bank reconciliation, posting, and Butler into one monetizable loop.
4. The source registry can become the foundation for audit, compliance, sync, support, and agent citations.
5. The Cap'n Proto schemas create a better long-term substrate than ad hoc JSON-only payloads.
6. The ViewModel layer is a path to front-end discipline without freezing product iteration.
7. The compliance/localization work is strategically aligned with underserved markets.
8. The UI/backend action inventories can become a systematic launch-readiness mechanism.
9. The local-first/offline posture is not a technical leftover; it is a market wedge.
10. The agent-safety principle is stronger than generic "AI in ERP" messaging.

## Biggest Risks

1. Breadth overwhelms loop closure.
2. Root service coupling makes module guarantees hard to enforce.
3. Sync/event infrastructure remains table-centric while the product narrative becomes module-centric.
4. AI/Butler surfaces outpace permission and audit hardening.
5. Compliance claims become stale or legally risky.
6. Frontend screens multiply without a coherent operator journey.
7. The math substrate gets overexplained in product-facing contexts.
8. Generated bindings and schemas churn without stable module ownership.
9. Inventory remains split and never becomes a clean evidence ledger.
10. The platform story launches before one wedge product is unmistakably useful.

## Most Underestimated Assets

| Asset | Why it matters |
|---|---|
| `docs/modules/*.manifest.json` | They can evolve into installable module contracts. |
| Business Memory review/export/replay | This is the seed of trustable document automation. |
| Source asset registry | Provenance is the moat for AI and compliance. |
| Posting coverage | Converts accounting automation from risky to inspectable. |
| Bank allocations | Makes Cashflow Evidence real for messy partial payments. |
| Compliance hooks | Let jurisdiction packs attach to business events. |
| UI/backend action audit docs | Can prevent feature drift and support agent-safe development. |
| TOON encoder/context packs | Compact structured context is ideal for local/agent workflows. |
| Sync normalization | Crucial for offline/on-prem credibility. |
| User guides/compliance dossiers | Early operator trust assets, not just docs. |

## Five-Year Evolution Prediction

If AsymmFlow keeps its current discipline, the likely five-year arc is:

### Year 1

AsymmFlow becomes a pilotable evidence-centered finance/docs tool for a handful of owner-led companies. Business Memory and Cashflow Evidence close first. Inventory Evidence enters pure-kernel/preflight maturity. The product is sold through founder-led implementation and support.

### Year 2

The platform becomes a specialized local-first ERP for trading/distribution companies in India/GCC-like contexts. Compliance packs, bank reconciliation, quote revision integrity, and accountant exports become reliable revenue features. AI Butler is trusted because it is bounded.

### Year 3

Module manifests become real installable contracts. Partners can build or configure vertical modules without touching core authority. Sync becomes module-aware. On-prem/cloud hybrid deployments become a differentiator.

### Year 4

AsymmFlow evolves into a regional business runtime: evidence inbox, cash, inventory, compliance, procurement, sales, and assistant workflows become composable. The moat is not UI polish alone; it is the accumulated evidence graph and deterministic trust layer.

### Year 5

The strongest version of AsymmFlow is an ecosystem substrate: a local-first, agent-safe ERP operating layer where companies can run operational modules, compliance packs, AI-assisted workflows, and partner-built extensions under one evidence and authority model.

The weaker failure mode is also clear: if the repo keeps adding foundations without closing loops, it becomes an impressive internal architecture with no wedge strong enough to sell. The next 2-4 weeks should therefore be ruthless: close Business Memory provenance, close Cashflow Evidence, and turn those into the first undeniable operator story.

## Evidence Appendix

### Repo Evidence Used

| Evidence type | Paths |
|---|---|
| Audit and roadmap | `docs/CODEX_REPO_HISTORY_AUDIT_2026_05_15.md`, `docs/CODEX_DEV_ROADMAP_2026_05_15.md`, `docs/CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT.md` |
| Module manifests | `docs/modules/business_memory_intake.manifest.json`, `docs/modules/cashflow_evidence.manifest.json` |
| Finance schemas | `schemas/finance.capnp` |
| CRM schemas | `schemas/crm.capnp` |
| Document schemas | `schemas/documents.capnp` |
| Business Memory | `pkg/documents/intake/*`, `pkg/adapter/documents/business_memory*.go`, `internal/viewmodel/documents/documents_vm.go` |
| Cashflow Evidence | `pkg/cashflow/evidence/*`, `internal/viewmodel/cashflow/evidence_vm.go` |
| Finance/accounting | `pkg/finance/*`, `pkg/finance/posting/*`, `pkg/finance/banking/*` |
| Compliance/localization | `pkg/compliance/*`, `pkg/i18n/*`, `pkg/engines/langpack.go`, `pkg/engines/arabic_shaper.go` |
| OCR/documents | `pkg/ocr/*`, `pkg/documents/*`, `pkg/engines/pdf_generator.go` |
| Sync/runtime | `pkg/sync/*`, `pkg/infra/*`, `pkg/api/*` |
| UI/ViewModel | `internal/viewmodel/*`, `frontend/src/lib/screens/*`, `frontend/src/lib/components/ui/*` |
| QA/support | `docs/testing/*`, `docs/compliance/*`, `docs/user-guides/*` |

### External Competitive Sources Used

| Source | Strategic signal |
|---|---|
| [Odoo 19 release notes](https://www.odoo.com/odoo-19-release-notes) | AI over documents/server actions, import templates, dashboards, livechat lead generation. |
| [ERPNext/Frappe framework page](https://frappe.io/erpnext/framework) | ERPNext customization, API-first, visual forms, custom reports/dashboards, workflows, permissions. |
| [Frappe Framework page](https://frappe.io/framework) | Metadata-driven DocType model, forms/views/kanban/calendar, marketplace/cloud hosting. |
| [Twenty key features](https://docs.twenty.com/getting-started/key-features) | Custom objects, custom fields, workflows, email/calendar sync, permissioned AI agents, dashboards, RBAC/audit logs, GraphQL/REST. |
| [monday.com AI Work Platform announcement](https://ir.monday.com/news-and-events/news-releases/news-details/2026/monday-com-Goes-All-In-on-AI-From-Work-Management-Platform-to-AI-Work-Platform/default.aspx) | Market shift from work management to human+agent execution under governance. |
| [Oracle NetSuite SuiteCloud AI announcement](https://www.oracle.com/latam/news/announcement/netsuite-brings-ai-powered-speed-accuracy-development-on-suitecloud-2026-04-28/) | Enterprise ERP customization moving toward AI-assisted agent skills. |
| [SAP Business One](https://www.sap.com/products/erp/business-one.html) | SMB ERP breadth across financials, purchasing, inventory, CRM, reporting, analytics, deployment flexibility. |
| [Retool agents and workflows](https://retool.com/resources/collections/agents-and-workflows) | Internal-tool platforms are moving toward AI agents, workflows, monitoring, and self-hosting. |

### Epistemic Status

| Claim type | Status |
|---|---|
| File/module existence | Evidence from current repo inspection. |
| Business Memory source registry durability | Committed evidence through `cfad303`, plus current manifest. |
| In-flight Business Memory export/review changes | Active worktree evidence only, not treated as stable. |
| Market direction | Current public vendor sources checked on 2026-05-15. |
| Product opportunity rankings | Strategic inference from repo evidence plus market direction. |
| Compliance readiness | Foundation only; legal freshness and filing workflows are not claimed. |
| Platformization readiness | Inferred from primitives; installable marketplace/runtime not yet shipped. |
