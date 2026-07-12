# AsymmFlow Module Contract Foundation

Status: Draft v0.1
Created: 2026-05-14
Scope: `the AsymmFlow repository`

## Purpose

This document defines the standard shape of an AsymmFlow module.

The target architecture is:

```text
Core Kernel -> Domain Module -> ViewModel -> UI Component -> Agent Surface
```

A module is not just a screen, service file, or package. A module is a closed business capability with durable data contracts, deterministic authority, user-facing workflow state, and constrained agent-safe APIs.

Every major module must justify itself with the Asymmetrica product filters:

- ROI proof: name the labor, leakage, subscription cost, compliance risk, or operational uncertainty it removes.
- Workflow closure: end with an action, artifact, posting, export, approval, claim pack, or decision.
- Engine leverage: identify the deterministic engine, invariant, memory/OCR/classification flow, optimization routine, or local-first runtime advantage.
- Operator trust: expose inspectability, correction, approval, export, audit trail, and repeatability.

## Contract Layers

### 1. Schemas And Data Contracts

Required:

- Durable domain schema in Cap'n Proto when the record crosses engine, sync, local-first, or generated-binding boundaries.
- Go domain structs for backend authority and persistence mapping.
- TypeScript/browser payloads generated from schemas or mapped from backend bindings, not hand-invented per screen.
- Explicit JSON use for browser payloads, third-party APIs, simple configs, and manifest files.
- TOON use for agent/context transfer when compact human-readable structure is more valuable than binary schema stability.

Rules:

- Cap'n Proto is the default for versioned module records, engine contracts, sync envelopes, and cross-runtime durability.
- JSON is acceptable for module manifests, Wails/browser interop, user-editable config, and third-party APIs.
- TOON is the default for Butler/Codex/agent briefings, source summaries, evidence packs, and context-kit style handoffs.
- Schema changes must name migration impact, generated binding impact, and test impact.

Current matching seams:

- `schemas/*.capnp` and `schemas/go/*/*.capnp.go`
- `frontend/src/lib/types/schemas/index.ts`
- `schemas/generate.ps1`
- `pkg/finance/domain.go`, `pkg/crm/domain.go`, `pkg/documents/domain.go`, `pkg/butler/domain.go`

Refactor need:

- New modules should not rely only on GORM structs plus generated Wails bindings. Durable records should have Cap'n Proto first-class ownership when the module is intended to be reusable.

### 2. Pure Kernels

Required:

- Pure calculation, invariant, matching, scoring, parsing, validation, or transformation functions.
- Inputs and outputs must be domain values, not database handles or UI state.
- No persistence, logging side effects, permissions, Wails calls, timers, or network access.
- Unit tests with edge cases and failure cases.

Current matching seams:

- `pkg/finance/posting` for balanced posting previews and trial-balance/coverage calculation.
- `pkg/compliance/bahrain` and `pkg/compliance/india` for deterministic VAT/GST/income-tax calculations and invoice validation.
- `bank_statement_parser.go`, `bank_transaction_matcher.go`, `costing_engine.go`, `eh_parser.go`, and OCR/classification helpers are candidate kernels but are not all separated consistently.

Refactor need:

- Inventory stock status and stock movement arithmetic currently lives inside app/service methods and should be extracted into a pure stock ledger kernel before the inventory module becomes a foundation module.
- Cashflow evidence should compose finance posting, AR aging, bank reconciliation, invoice traceability, and document evidence kernels instead of adding another screen-local calculation layer.

### 3. Domain Services

Required:

- Own permissions, transactions, persistence orchestration, events, audit logs, idempotency, and irreversible side effects.
- Keep deterministic backend authority in Go.
- Approve, persist, post, file, export, or mutate state only through deterministic services.
- Return domain results or ViewModel-ready adapters, not raw UI decisions.

Current matching seams:

- `accounting_posting_service.go`
- `customer_invoice_service.go`
- `payment_service.go`
- `supplier_invoice_service.go`
- `supplier_payment_service.go`
- `bank_reconciliation_service.go`
- `pkg/finance/banking/service.go`
- `document_classifier.go`
- `ocr_service_simple.go`
- `pkg/documents/classifier/service.go`
- `pkg/documents/excel/costing_parser.go`
- `butler_ai.go`, `butler_grounded_fastpath.go`, `butler_intent_router.go`
- `pkg/butler/fastpath/grounded.go`, `pkg/butler/intent/router.go`, `pkg/butler/chat/toon.go`
- `pkg/compliance/hooks.go`

Refactor need:

- Large root-level app/service files should progressively move behind package ports and module service boundaries.
- Agent-originated suggestions must remain draft-only until a domain service validates permissions and invariants.

### 4. Storage Adapters

Required:

- Storage adapter per module or submodule, even if the first implementation uses GORM/SQLite directly.
- Explicit migration ownership, transaction boundaries, indexes, idempotency keys, and sync/CDC impact.
- Import/export adapters separate from core service logic.

Current matching seams:

- SQLite/GORM domain tags in `pkg/*/domain.go`.
- `database.go` migration/autoload surface.
- `pkg/finance/ports.go`, `pkg/documents/ports.go`, `pkg/butler/ports.go`, `pkg/crm/ports.go`.
- `db_sync_service.go`, `sync_service.go`, `pkg/sync`.

Refactor need:

- Many root services still operate directly on `a.db`; future modules should depend on repositories/ports so kernels and services can be tested without Wails/App wiring.

### 5. ViewModels

Required:

- ViewModels own presentation-ready state, commands, validation state, async status, correction options, and operator trust surfaces.
- Views render ViewModels and emit intent only.
- ViewModels must not own database logic, posting logic, tax logic, OCR logic, or LLM orchestration.

Current matching seams:

- `internal/viewmodel/finance`
- `internal/viewmodel/butler`
- `internal/viewmodel/documents`
- `internal/viewmodel/crm`
- `internal/viewmodel/compliance_vm.go`
- `invoice_list_vm_endpoint.go`

Refactor need:

- Existing Svelte screens still contain substantial local workflow logic. Future module work should create backend/ViewModel builders first, then simplify screens around command/state contracts.

### 6. UI Surfaces

Required:

- Svelte UI surfaces follow the Asymmetrica product UI standard.
- No generic SaaS defaults when building new surfaces.
- Each module screen must expose source/provenance, canonical state, confidence/status, correction, approval/action, export or output, and audit trail where applicable.
- UI is not allowed to become the business engine.

Current matching seams:

- `frontend/src/lib/screens/AccountingScreen.svelte`
- `frontend/src/lib/screens/FinanceHub.svelte`
- `frontend/src/lib/screens/InboxScreen.svelte`
- `frontend/src/lib/screens/ButlerScreen.svelte`
- `frontend/src/lib/screens/IntelligenceHub.svelte`
- `frontend/src/lib/screens/GRNScreen.svelte`
- `frontend/src/lib/screens/DeliveryNotesScreen.svelte`
- `frontend/src/lib/screens/PurchaseOrdersScreen.svelte`
- `frontend/src/lib/screens/AuditTrailViewer.svelte`

Refactor need:

- New modules should expose compact operational command centers rather than adding another screen pile. Tables, evidence timelines, approval cards, audit trails, source-link chips, and readiness indicators should become reusable components.

### 7. Events

Required:

- Every state-changing module command should define emitted events, subscribers, and event payload contract.
- Events must include correlation ID where cross-module action is expected.
- Events are facts about deterministic state changes, not agent thoughts.

Current matching seams:

- `pkg/infra/events/events.go`
- `pkg/infra/events/bus.go`
- `pkg/compliance/hooks.go`

Refactor need:

- Event payloads should move from minimal IDs to typed module events with optional Cap'n Proto schema ownership when they become sync/audit boundaries.
- Cashflow Evidence should subscribe to invoice/payment/document/bank/posting events rather than polling every source independently.

### 8. Permissions

Required:

- Module manifest lists all read, create, update, approve, post, export, administer, and agent permissions.
- Backend service methods enforce permissions before data access or mutation.
- Agent APIs must use the same permission model and cannot bypass approvals.

Current matching seams:

- `requirePermission()` guards across App endpoints.
- Existing permission namespaces such as `finance:view`, `finance:create`, `inventory:view`, `inventory:create`, `invoices:view`, `dashboard:view`.
- `pkg/butler/ports.go` includes user context/permission ports.

Refactor need:

- Permission namespaces should be normalized per module and included in module manifests before new modules ship.

### 9. Audit Trails

Required:

- Module commands must document which durable audit records they write.
- Irreversible actions need actor, timestamp, source record, before/after or reason, and correlation ID where applicable.
- Agent suggestions need audit records linking suggestion, evidence, user approval, and deterministic service action.

Current matching seams:

- `pkg/finance/domain.go` audit fields and `BankReconciliationAuditLog`.
- `pkg/infra/events`.
- `invoice_traceability.go`.
- `AuditTrailViewer.svelte`.
- `pkg/butler/domain.go` action status fields on chat messages.

Refactor need:

- Audit trails exist but are not yet expressed as a consistent module contract. The module manifest should make audit obligations explicit before implementation.

### 10. Tests

Required:

- Kernel unit tests.
- Service tests for permissions, transactions, idempotency, and persistence edge cases.
- Storage adapter tests for migrations and import/export mapping.
- ViewModel builder tests for display-ready state and command visibility.
- UI smoke/contract tests where the module exposes a new or changed surface.
- Agent API tests proving draft/recommend actions cannot mutate state without deterministic approval.

Current matching seams:

- `pkg/finance/posting` tests.
- `pkg/compliance` tests.
- Root service tests for invoices, payments, banking, Butler, OCR, sync, and business invariants.
- `internal/viewmodel/*/*_test.go`.
- Frontend contract/e2e scripts in `frontend/package.json`.

Refactor need:

- Future modules should name focused test gates in the manifest and handoff before coding begins.

### 11. Agent-Safe APIs

Required:

- Agents may inspect, explain, summarize, classify, draft, recommend, and assemble evidence.
- Agents may not approve, persist, post, file, delete, reverse, or enforce invariants directly.
- Agent responses should cite source records, confidence/status, missing evidence, and next deterministic command.
- TOON is preferred for compact context packs and agent handoffs.

Current matching seams:

- `pkg/butler/ports.go`
- `butler_grounded_fastpath.go`
- `butler_intent_router.go`
- `app_butler_context.go`
- `app_butler_fastpath.go`
- `schemas/butler.capnp`
- `pkg/butler/chat/toon.go`

Refactor need:

- Butler actions should be classified into inspect/explain/draft/recommend versus approve/persist/post. Only the first group belongs in agent surfaces.

## Module Manifest

The skeleton manifest lives at:

```text
docs/templates/module_manifest.example.json
```

Use JSON because the manifest is a human-editable configuration and planning artifact. When a module moves from planning to generated/runtime consumption, durable module records should move into Cap'n Proto or a generated contract, while compact agent summaries should be emitted as TOON.

Minimum manifest fields:

- module identity and owner
- closed product loop
- ROI/workflow/engine/operator-trust filters
- schemas and data contracts
- kernels
- domain services
- storage adapters
- ViewModels
- UI surfaces
- events
- permissions
- audit trails
- tests
- agent-safe APIs
- launch readiness checklist

## Domain Mapping

| Domain | Closed Loop | Existing Seams | Contract Fit | Refactor Needs |
| --- | --- | --- | --- | --- |
| Finance | invoices/payments/bank/expenses -> postings/reconciliation/reports -> approval/export/audit | `pkg/finance/domain.go`, `pkg/finance/ports.go`, `pkg/finance/posting`, `pkg/finance/banking`, `schemas/finance.capnp`, `accounting_posting_service.go`, `finance_reporting_service.go`, `internal/viewmodel/finance`, `AccountingScreen.svelte` | Strongest current fit. Has schemas, GORM models, ports, pure posting kernel, banking service/audit shape, services, ViewModels, UI, permissions, tests, and audit fields. | Split more root services behind package adapters; finish controlled posting action, period-lock checks, and support-bundle evidence export. Cap'n Proto exists but is not yet the active source of truth for all service payloads. |
| Documents | files/PDF/Excel/email/OCR -> classification/extraction -> review/correction -> linked business record -> evidence/audit | `pkg/documents/domain.go`, `pkg/documents/ports.go`, `pkg/documents/classifier/service.go`, `pkg/documents/ocr`, `pkg/documents/excel/costing_parser.go`, `schemas/documents.capnp`, `document_classifier.go`, `ocr_service_simple.go`, `runtime_handlers.go`, `InboxScreen.svelte`, `internal/viewmodel/documents` | Medium fit. Has schemas, ports, classifier/OCR/Excel package seams, and intake services, but workflow boundaries are still spread across root files and screens. | Add canonical evidence record, review queue ViewModel, source-link audit, retry/status contracts, and agent-readable TOON evidence pack. |
| Butler | operator query/context -> grounded answer/draft/recommendation -> deterministic service approval -> audit | `pkg/butler/domain.go`, `pkg/butler/ports.go`, `pkg/butler/intent/router.go`, `pkg/butler/fastpath/grounded.go`, `pkg/butler/chat/toon.go`, `schemas/butler.capnp`, `app_butler_context.go`, `internal/viewmodel/butler`, `ButlerScreen.svelte` | Medium-strong fit. Has domain records, ports, schema, TOON boundary, ViewModels, screens, conversation persistence, permission-aware finance access, and action metadata. | Formalize action permission tiers; prevent agent APIs from owning writes; emit TOON module context packs with source citations. |
| Compliance | business events -> jurisdiction engine -> validation/calculation -> filing/export evidence -> audit trail | `pkg/compliance`, `pkg/compliance/bahrain`, `pkg/compliance/india`, `pkg/compliance/hooks.go`, `internal/viewmodel/compliance_vm.go`, `compliance_bindings.go`, `docs/compliance/*` | Strong kernel/registry fit. Engines are clean and tested; hooks already connect events to validation. | Add explicit `pkg/compliance/ports.go`, Cap'n Proto schema, manifest, durable validation storage, filing/export surfaces, jurisdiction pack readiness checklist, typed event payloads, and UI command center. |
| Inventory | receipt/reservation/delivery/serial evidence -> stock truth -> valuation/risk/action/export/audit | `pkg/crm/domain.go` inventory/serial structs, `pkg/crm/ports.go` fulfillment/procurement ports, `pkg/crm/procurement/service.go`, `pkg/crm/fulfillment/service.go`, `app_accounting_inventory.go`, `inventory_service.go`, `serial_number_service.go`, `schemas/crm.capnp`, operations screens | Medium fit. Durable data and UI exist, serial traceability is valuable, but ownership is split across CRM, finance facade, GRN, serial, delivery note, and root app services. | Extract stock movement/reservation/valuation kernels; define `pkg/inventory` ownership or clear submodule boundary; add inventory ViewModels and explicit events. |
| Cashflow Evidence | orders/invoices/payments/docs/bank/postings -> canonical evidence -> receivables risk/follow-up/posting/export -> audit | Finance posting coverage, trial-balance gate, AR/cashflow reporting, `pkg/finance/banking` reconciliation/audit, `invoice_traceability.go`, document/OCR, Butler grounded context | Emerging composition module. It should compose existing Finance, Documents, Butler, and Compliance seams rather than become a standalone data silo. | Define first-class module manifest, read model, event subscriptions, evidence pack schema, command center ViewModel, and support-bundle export. |

## Launch Readiness Checklist

A module is not launch-ready until all required items are explicitly answered.

### Product Closure

- Closed loop is documented from messy input to audit trail.
- ROI proof names replacement value or risk reduction.
- Workflow closure ends with an action/artifact/posting/export/approval/decision.
- Engine leverage is named.
- Operator trust surface is visible.

### Architecture

- Cap'n Proto / TOON / JSON choices are documented.
- Pure kernel exists or the reason it is unnecessary is documented.
- Domain service owns permissions, persistence, audit, events, and side effects.
- Storage adapter and migration ownership are explicit.
- ViewModel contract exists before or with UI work.
- UI follows the Asymmetrica product UI standard.
- Agent API tier is inspect/explain/draft/recommend only unless routed through deterministic approval.

### Data And Safety

- Permissions are listed and enforced.
- Audit records are durable and queryable.
- Event payloads and subscribers are named.
- Idempotency and rollback behavior are documented for mutations.
- Import/export paths include validation and error reporting.
- Local-first/sync impact is documented.

### Verification

- Kernel tests pass.
- Service tests cover permissions and invariant failures.
- Storage/migration tests or checks are run when schema changes.
- ViewModel tests cover display state and command availability.
- Frontend check/build runs for UI changes.
- `go build ./...` and focused Go tests run for code changes.
- Baseline warnings are recorded separately from new regressions.

## Epistemic Status

- Existing repo seams listed here are verified by current file inspection during the 2026-05-14 Module Contract Foundation goal.
- The module contract is a design standard and should be treated as an assumption-scaffolded architecture rule until a full module is implemented under it.
- The Cap'n Proto / TOON / JSON split is a standing Asymmetrica architecture standard and is applied here as a pragmatic engineering rule.
- Cashflow Evidence is a proposed composition module. It is not yet implemented as a first-class module.
