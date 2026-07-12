# CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT

Status: Draft v0.1
Created: 2026-05-14
Scope: `the AsymmFlow repository`

## Goal Command

Run the AsymmFlow roadmap chain for the 2-hour window that starts with Engine Generalization Inventory. Complete the engine inventory through a reviewable artifact, verification, and a coherent commit/checkpoint; if the inventory reaches an auditable state early, advance to `CODEX_GOAL_CASHFLOW_EVIDENCE_HANDOFF.md`.

## Run Boundary

| Field | Value |
| --- | --- |
| Start time | 2026-05-14T12:44:30.7240875+05:30 |
| Target checkpoint time | 2026-05-14T14:44:30+05:30 |
| Starting commit | `85dcf61` |
| Initial tracked status | `git status --short` returned no tracked changes |
| Ignored local artifacts | `.claude/`, `.gitnexus/`, `build/`, `frontend/dist/`, `frontend/node_modules/`, `ph_holdings.db`, `test_output/`, `wave7_root_test.log`, `wave7_root_test_ticket7.log`, `wave7_target_test.log` |
| Write boundary | Docs only unless verification discovers a blocker |

Ignored runtime/build state is baseline local state and should not be touched unless a later verification gate requires it.

## Context Read

Required context was read in order:

1. `C:\Projects\ASYMMETRICA_ECOSYSTEM_LOG.md`
2. `C:\Projects\AGENTIC_SWARM_PROTOCOL.md`
3. `C:\Projects\ASYMMETRICA_PRODUCT_UI_AND_ARCHITECTURE_STANDARD.md`
4. `docs/CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md`
5. `docs/MODULE_CONTRACT_FOUNDATION.md`
6. `docs/CODEX_GOAL_MODULE_CONTRACT_HANDOFF.md`
7. Historical evidence from `docs/V0_1_RELEASE_ROADMAP_2026_05_08.md` and `docs/WAVE17_PROGRESS.md`

The current canon is the 2026-05-14 master roadmap plus Module Contract Foundation. Older wave docs are evidence of shipped behavior and verification history, not the current plan of record.

## Target Shape

Every engine should move toward this module contract:

```text
pure kernel -> domain service -> storage adapter -> ViewModel adapter -> agent adapter
```

Classifications used in this audit:

- pure kernel: deterministic calculation, parsing, scoring, matching, routing, optimization, or validation with no persistence/UI/LLM side effects.
- domain service: permissions, transactions, persistence orchestration, audit, event emission, idempotency, and irreversible side effects.
- storage adapter: GORM/SQLite/Cap'n Proto/import/export/sync mapping and migration ownership.
- ViewModel adapter: presentation-ready state, commands, validation status, async status, correction options, and trust surfaces.
- agent adapter: inspect, explain, summarize, draft, recommend, or assemble evidence. It must not approve, persist, post, file, delete, reverse, or enforce invariants directly.

## Product Bar

The engine inventory was scored against the "$1000/mo justification" filters:

- ROI proof: replaced labor, subscriptions, cash leakage, compliance risk, support burden, or operational uncertainty.
- Workflow closure: action, artifact, posting, export, claim pack, approval, support bundle, or decision.
- Engine leverage: deterministic engine, invariant, math substrate, OCR/classification, local-first sync, or memory/context advantage.
- Operator trust: provenance, correction, approval, export, audit trail, repeatability, and reviewable evidence.

## Engine Generalization Inventory

| Domain | Engine / Current Seams | Current Classification | Reusable Kernel Candidates | Product Loops Powered | Coupling Risks | Recommended Future Shape |
| --- | --- | --- | --- | --- | --- | --- |
| Finance | Posting previews and ledger gates: `pkg/finance/posting`, `accounting_posting_service.go`, `pkg/finance/domain.go`, `internal/viewmodel/finance`, `AccountingScreen.svelte` | Strong pure kernel plus root-level domain service and UI adapter | Balanced posting preview, trial-balance gate, posting coverage report, source-to-journal link resolution | Cashflow Evidence, accounting close, support bundle, historical posting backfill, audit-ready finance desk | `CreateDraftJournalFromPosting` still lives in root `App`; account resolution mixes mapping, creation, and source linking; period-lock checks are still a next ticket | Keep `pkg/finance/posting` as pure kernel; extract a `finance/posting` domain service with account mapping, idempotency, period locks, event emission, and repository ports; expose coverage via a Cashflow Evidence VM and Butler explanation adapter |
| Finance | Banking/reconciliation: `pkg/finance/banking`, `bank_reconciliation_service.go`, `bank_statement_parser.go`, `bank_transaction_matcher.go`, `pkg/finance/banking/williams_bridge.go` | Domain service with embedded matching/parsing kernels and storage side effects | Bank statement parser, transaction classifier, invoice/PO matcher, amount tolerance, duplicate statement hash, Williams batch sizing | Bank reconciliation, cash position, payment evidence, receivables follow-up, cash leakage detection | Parser remains root-level and bank-format-specific; matcher touches GORM directly; reconciliation audit exists but is not a module-wide event contract | Split bank import parsing and match scoring into pure kernels; keep manual match/unmatch/finalize in a banking service; emit typed reconciliation events; surface unmatched and duplicate-risk rows in Cashflow Evidence |
| Finance | Reporting and cash projections: `finance_reporting_service.go`, `internal/viewmodel/finance/finance_vm.go` | Root-level domain/read-model service with ViewModel support | AR aging buckets, cashflow projection, VAT reconciliation, margin analysis | Receivables command center, owner cash forecast, supplier payment planning, compliance readiness | Cash projection queries invoices, supplier invoices, expenses, payroll, and recurring expenses directly from root service; forecast assumptions are not yet explainable/correctable as a module contract | Extract AR aging and cash projection kernels over normalized inputs; make data collection a service/read-model adapter; expose assumptions and missing evidence to a Cashflow Evidence VM |
| Documents | Classification and filesystem routing: `document_classifier.go`, `pkg/documents/classifier/service.go`, `pkg/documents/domain.go`, `pkg/documents/ports.go`, `InboxScreen.svelte` | Mixed pure classifier, domain service, and UI workflow | Regex/heuristic document type classification, filesystem metadata extraction, routing suggestion, confidence/explanation model | Business Memory Intake, evidence inbox, invoice/PO/contract review queue, Butler source-grounding | Root classifier duplicates package seam; AI fallback and deterministic fallback share result shape but not a formal permission/audit contract; source links are not yet a canonical evidence record | Keep deterministic classifier as pure kernel; introduce `documents/evidence` domain service for review/correction/linking; emit TOON evidence packs for agents; keep AI classification as draft-only enrichment |
| Documents | OCR and extraction: `ocr_service_simple.go`, `pkg/documents/ocr`, `pkg/ocr/orchestrator`, `pkg/ocr/ksum`, `pkg/ocr/octonion`, `pkg/ocr/observability.go` | Domain service around several pure/heuristic OCR kernels and external adapters | Vector PDF extraction, document type detection from text, structured field regex extraction, k-sum table detection, octonion color enhancement, OCR metrics | Business Memory Intake, expense/invoice ingestion, bank statement import, support bundle evidence, document quality scoring | Runtime service mixes local extraction, Mistral/Fly.io calls, file-type branching, logging, and field extraction; external OCR adapters are not isolated behind module ports | Separate extraction kernels from OCR provider adapters; add an evidence intake service with retry/status/audit; use observability metrics as operator-visible trust signals |
| Documents | Excel/email/PDF import: `pkg/documents/excel/costing_parser.go`, `excel_costing_parser.go`, `msg_parser.go`, `runtime_handlers.go`, `pkg/documents/email/msg_parser.go` | Parser kernels plus root runtime handlers | Costing parser, bank statement parser, MSG/EML text extraction, field normalization | Sales costing, tender intake, bank reconciliation, business memory, source-backed offer generation | Format-specific parsers are not consistently package-owned; root handlers can become workflow glue instead of adapters | Move format parsers behind `documents.Parser` ports; normalize outputs into canonical evidence records; bind to review queue before domain writes |
| Butler | Intent, grounded fastpath, TOON chat, report generation: `butler_ai.go`, `pkg/butler/intent`, `pkg/butler/fastpath`, `pkg/butler/chat/toon.go`, `pkg/butler/reports`, `internal/viewmodel/butler`, `ButlerScreen.svelte` | Agent adapter with deterministic fastpath kernels and report service | Intent classifier, AR projection scope parser, capability explainer, TOON compaction, action parser, report data formatter | Manager finance briefs, AR/cash explanations, task/follow-up drafts, capability discovery, evidence pack narration | Root `butler_ai.go` remains large and mixes LLM calls, deterministic routes, action parsing, report dispatch, and permissions; action tiers are not enforced as a first-class module permission contract | Treat Butler as an agent adapter over modules; keep intent parsing and TOON encoding pure; route all mutations through deterministic domain services; formalize action tiers as inspect/explain/draft/recommend vs approve/persist/post |
| Butler | Prediction/payment intelligence: `pkg/butler/prediction`, `payment_intelligence.go`, `pkg/engines/predictor.go` | Pure-ish math scoring plus DB-backed service | M79 customer encoder, three-regime payment risk, win probability, discount recommendation, business value summary | Credit policy, collections prioritization, offer discount control, counterparty risk | Some predictors read GORM directly or exist in both `pkg/butler/prediction` and `pkg/engines`; epistemic status is heuristic/empirical, not proven | Consolidate prediction kernels in one package; define feature snapshots as inputs; make database-backed intelligence a service; expose confidence, inputs, and override path in Cashflow Evidence |
| Compliance | Jurisdiction tax engines and registry: `pkg/compliance`, `pkg/compliance/bahrain`, `pkg/compliance/india`, `pkg/compliance/hooks.go`, `internal/viewmodel/compliance_vm.go`, `compliance_bindings.go` | Strong pure kernel plus event hook and minimal VM | Bahrain VAT, India GST, India income tax, GSTIN validation, slab tax, invoice validation, compliance registry | Jurisdiction packs, filing/export readiness, invoice compliance, owner tax planning | No `pkg/compliance/ports.go`; validation storage is in-memory; event payloads are loosely mapped from maps; no filing/export service yet | Create compliance pack interface with ports, durable validation storage, typed event payloads, Cap'n Proto schema if pack records cross sync boundaries, and filing/export ViewModel |
| Inventory | Stock, serial, procurement, fulfillment: `pkg/crm/domain.go`, `pkg/crm/ports.go`, `pkg/crm/procurement`, `pkg/crm/fulfillment`, `inventory_service.go`, `serial_number_service.go`, `grn_service.go`, `delivery_note_service.go`, `purchase_order_service.go`, operations screens | Medium fit: data model and service ports exist, but kernels are coupled to CRM/root app | Stock movement arithmetic, available/reserved/on-hand calculation, serial availability, reorder suggestion, GRN-to-stock update, delivery fulfillment status, valuation baseline | Inventory Asset Evidence Ledger, receive-to-pay, delivery evidence, warranty/serial traceability, stock risk | Inventory ownership is split across CRM domain, root services, GRN/PO/DN screens, and finance inventory facade; reorder days use placeholders; no independent `pkg/inventory` module yet | Extract `pkg/inventory` for stock ledger kernel and inventory service; keep procurement/fulfillment as CRM-adjacent domain services; emit typed stock/serial/GRN/DN events; create inventory ViewModel and evidence timeline |
| Sync/CDC/Observability | DB sync, file sync, CDC, event bus, health/OTel: `db_sync_service.go`, `sync_service_impl.go`, `sync_record_normalization.go`, `pkg/sync`, `pkg/sync/turso`, `pkg/adapter/sync`, `pkg/infra/events`, `pkg/infra/health`, `pkg/infra/otel`, `pkg/ocr/observability.go` | Storage adapter and infrastructure domain service with pure normalization/health kernels | Boolean/schema normalization, natural-key upsert, CDC log/retrieve, event bus, health regime classifier, OTel domain spans, OCR metrics | Local-first module state, support bundles, audit trail, sync evidence, operator readiness dashboard | Two sync services coexist; conflict policy is table/record-oriented rather than module-contract-oriented; root services emit Wails events but module event ownership is thin | Make `pkg/sync` the module-state contract; keep normalization as pure adapter kernel; add module-aware sync envelope and conflict policies; wire typed domain events into CDC/support-bundle evidence |
| Math/Optimization | Williams, quaternion, trident, prism, conversation chain, VQC, OCR math: `pkg/math`, `pkg/engines/williams_optimizer.go`, `pkg/vqc`, `geometry_bridge.go`, `pkg/butler/chat/optimizer_bridge.go`, `pkg/finance/banking/williams_bridge.go`, `pkg/ocr/ksum`, `pkg/ocr/octonion` | Mostly pure kernels, with some demo/bridge/domain coupling | Williams batching, digital root filtering, quaternion distance/SLERP, three-regime classification, conversation coherence, k-sum table detection, octonion color processing | Batch OCR/import, bank reconciliation batching, Butler prompt routing, customer risk scoring, document quality routing, support diagnostics | Some math packages carry research/demo language and duplicate older root-level `pkg/engines`; proof/empirical status varies; domain bridges may overclaim if not labeled | Promote `pkg/math` kernels as reusable substrate with explicit epistemic status; keep root `pkg/engines` bridges as adapters or retire duplicates; document when a kernel is Lean-proven, empirically tested, assumption-scaffolded, or heuristic |
| Cashflow Evidence | Composition from finance, documents, Butler, compliance, banking, events: `accounting_posting_service.go`, `finance_reporting_service.go`, `invoice_traceability.go`, `pkg/finance/posting`, `pkg/finance/banking`, `pkg/butler/fastpath`, `pkg/infra/events`, `AccountingScreen.svelte` | Emerging composition module, not yet first-class | AR aging, cash projection, posting coverage, trial-balance gate, bank match status, invoice traceability, evidence completeness scoring, follow-up priority | Cashflow + Evidence Command Center, collections queue, posting backfill, evidence pack export, manager brief | No read model/module manifest; no canonical evidence schema; no event subscription spine; no command center VM; Butler can explain pieces but does not own deterministic state | Make this the next proof module: read model over invoice/payment/bank/document/posting events, evidence pack schema, command center VM, export/support bundle, Butler inspect/explain adapter, deterministic service commands for draft postings and follow-ups |

## Top Reusable Kernels

| Rank | Kernel | Evidence | Epistemic Status | Why It Matters |
| --- | --- | --- | --- | --- |
| 1 | Posting preview + trial-balance/coverage kernel | `pkg/finance/posting` | Verified by local tests and Wave 17 progress evidence | Turns invoices/payments/supplier docs into auditable accounting intent and gives operators coverage gates before automation |
| 2 | OCR/classification/extraction kernels | `document_classifier.go`, `pkg/documents/classifier`, `pkg/ocr/ksum`, `pkg/ocr/octonion`, `ocr_service_simple.go` | Mixed: deterministic regex/heuristic plus empirical OCR/math tests | Converts messy business input into canonical evidence and review queues |
| 3 | Bank matching and reconciliation scoring | `pkg/finance/banking`, `bank_statement_parser.go`, `bank_transaction_matcher.go` | Empirical/service-tested | Converts statement chaos into cash evidence, payment links, and leakage controls |
| 4 | Jurisdiction tax engines | `pkg/compliance/bahrain`, `pkg/compliance/india`, `pkg/compliance/hooks.go` | Deterministic unit-tested rules, law/version assumptions must be maintained | Powers installable compliance packs and filing/export readiness |
| 5 | Stock/serial movement ledger kernel | `pkg/crm/domain.go`, `inventory_service.go`, `serial_number_service.go`, `grn_service.go`, `delivery_note_service.go` | Assumption-scaffolded from current services; extraction still needed | Generalizes industrial traceability into inventory/assets/evidence ledgers |
| 6 | Williams batching and three-regime/math routing | `pkg/math/vedic`, `pkg/engines/williams_optimizer.go`, `pkg/infra/health`, `pkg/butler/chat/optimizer_bridge.go` | Williams has a Lean proof pointer in code; local Go tests cover implementation; other routing is heuristic/empirical | Provides reusable batching, health, prompt/context, and import/OCR scaling substrate |

## Top Product Loops Powered

| Rank | Product Loop | Engines Used | Closed Outcome |
| --- | --- | --- | --- |
| 1 | Cashflow + Evidence Command Center | AR aging, cash projection, posting coverage, bank matching, invoice traceability, document evidence, Butler explanation | Follow-up queue, draft posting/report/export, support bundle, audit trail |
| 2 | Business Memory Intake | OCR/classification, Excel/email/PDF parsers, evidence linking, TOON context packs | Reviewed/corrected canonical business records with source provenance |
| 3 | Inventory + Asset Evidence Ledger | Stock movement, serial traceability, GRN/DN/procurement/fulfillment, valuation | Availability/valuation/risk decisions plus serial/lot evidence export |
| 4 | Compliance Pack Architecture | Bahrain VAT, India GST/income tax, event hooks, validation registry | Filing/export evidence, readiness dashboard, jurisdiction-specific audit trail |
| 5 | Local-First Module State + Support Bundle | Sync/CDC, event bus, OTel/health, audit logs, module manifests | Conflict-aware replication, operator support evidence, deploy/pilot readiness |
| 6 | Butler Manager Briefs | TOON compaction, AR scope parser, grounded fastpath, prediction, report generation | Inspectable manager explanations, draft actions, and cited next deterministic commands |

## Coupling Risks

1. Root-level `App` services still own too much business workflow. Finance, documents, inventory, Butler, sync, and OCR all have root files that combine permissions, queries, transformations, Wails endpoint behavior, logs, and side effects.
2. Inventory is hidden inside CRM plus root services. This blocks a clean Inventory Asset Evidence Ledger and makes stock movement hard to test independently.
3. Documents lack a canonical evidence record. OCR/classification can produce outputs, but source provenance, correction, linking, retry status, and agent context are not yet one module contract.
4. Butler is powerful but too broad. It mixes LLM calls, deterministic fastpaths, action parsing, report generation, and permissions in root code. Without action tiers, agent surfaces can drift toward command authority.
5. Sync is table-centric, not module-centric. Current sync records and normalization are useful, but module contracts need typed conflict policy, support evidence, and event/CDC linkage.
6. Math substrate needs clearer status labels. Some pieces are proven or test-backed; others are heuristic or experimental. Future product specs should state the epistemic status next to every mathematical claim.
7. ViewModels are uneven. Finance/CRM/compliance/documents/butler have seams, but many Svelte screens still contain workflow logic that should move into ViewModel adapters.

## Refactor Recommendations

### Immediate

- Build Cashflow Evidence as the first proof module rather than refactoring everything at once.
- Define a Cashflow Evidence read model that composes invoices, payments, supplier invoices/payments, bank statements, posting coverage, trial-balance gate, documents, and invoice traceability.
- Extract only the minimum pure kernels needed for Cashflow Evidence: AR aging normalization, cash exposure scoring, evidence completeness scoring, and follow-up priority.
- Add a `docs/CODEX_GOAL_CASHFLOW_EVIDENCE_HANDOFF.md` handoff before implementation so the next wave has a concrete write boundary.

### Near-Term

- Create `pkg/inventory` as the owner of stock movement, reservation/availability, valuation baseline, and serial evidence kernels.
- Add `pkg/compliance/ports.go` and durable validation storage before turning jurisdiction engines into installable packs.
- Create a `documents/evidence` service and canonical evidence record before Business Memory Intake expands.
- Formalize Butler action tiers in the module manifest and tests: inspect, explain, summarize, draft, recommend are agent-safe; approve, post, persist, delete, reverse, file are deterministic-service-only.
- Make sync envelopes module-aware and connect typed domain events to CDC/support bundle output.

### Later

- Retire or adapter-wrap duplicated research/demo engine packages once `pkg/math` is the source of reusable math kernels.
- Move Cap'n Proto ownership forward for module records that cross sync, generated-binding, or engine boundaries.
- Convert repeated screen-local workflow state into reusable ViewModel builders and AsymmFlow component-library surfaces.

## Recommended Next Goal

The next roadmap goal should be:

```text
CODEX_GOAL_CASHFLOW_EVIDENCE_HANDOFF.md
```

Reason: Cashflow Evidence is the highest-leverage proof module because it reuses the strongest existing engines and closes a painful business loop:

```text
orders/invoices/payments/docs/messages -> canonical evidence -> receivables risk -> follow-up/action -> draft posting/report/export -> audit trail
```

Target first implementation slice:

- Manifest and read-model contract for Cashflow Evidence.
- Pure kernels for cash exposure, missing evidence, and posting readiness scoring.
- Backend read service over existing deterministic finance/document/banking/posting seams.
- ViewModel for command center state and actions.
- Butler inspect/explain adapter that cites evidence and points to deterministic commands.
- Focused tests for the new kernels/read model.

## Verification Plan

Docs-only verification for this audit:

- `git status --short` before edits and after edits.
- `git status --short --ignored` to keep ignored runtime/build state separate.
- Structure/content checks for required sections and domains.
- JSON validation if manifest or handoff references are changed.
- `git diff --check`.

No code/schema/frontend files are intentionally changed by this audit. If later implementation touches code, use:

- `go build ./...`
- `go test ./... -count=1 -timeout 300s`
- `npm.cmd --prefix frontend run check`
- `npm.cmd --prefix frontend run build`
- `powershell -NoProfile -File schemas/generate.ps1 -CheckOnly` for schema changes

## Command Log

| Command | Result |
| --- | --- |
| `git status --short` | Passed; no tracked changes |
| `Get-Content C:\Projects\ASYMMETRICA_ECOSYSTEM_LOG.md` | Passed |
| `Get-Content C:\Projects\AGENTIC_SWARM_PROTOCOL.md` | Passed |
| `Get-Content C:\Projects\ASYMMETRICA_PRODUCT_UI_AND_ARCHITECTURE_STANDARD.md` | Passed |
| `Get-Content docs\CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md` | Passed |
| `Get-Content docs\MODULE_CONTRACT_FOUNDATION.md` | Passed |
| `Get-Content docs\CODEX_GOAL_MODULE_CONTRACT_HANDOFF.md` | Passed |
| `Get-Date -Format o` | Passed; recorded start time |
| `git rev-parse --short HEAD` | Passed; `85dcf61` |
| `git status --short --ignored` | Passed; ignored artifacts recorded separately |
| `Get-Content docs\V0_1_RELEASE_ROADMAP_2026_05_08.md` | Passed |
| `Get-Content docs\WAVE17_PROGRESS.md` | Passed |
| `rg --files docs` | Passed |
| `rg --files pkg internal frontend\src\lib\screens schemas \| rg ...` | Passed; mapped package/screen/schema seams |
| `Test-Path docs\CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT.md` | Passed; returned `False` before creation |
| Finance, document, Butler, compliance, inventory, sync, math, and cashflow `rg -n` inspections | Passed; evidence paths reflected in inventory |
| Read-only explorer for Finance/Documents/Butler/Cashflow | Completed; findings folded into inventory and refactor priorities |
| Read-only explorer for Compliance/Inventory/Sync/Math | Completed; findings folded into inventory and refactor priorities |
| `git status --short --branch` | Passed after edits; staged/unstaged set contained only goal-owned docs before staging |
| `Get-Date -Format o` | Passed; elapsed check at 2026-05-14T12:50:52.8296456+05:30 |
| Required-section `rg` checks for the audit and Cashflow Evidence handoff | Passed |
| Conflict-marker search across touched docs | Passed; no matches |
| `git diff --check` | Passed; only LF/CRLF normalization warnings |
| `git add docs\CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT.md docs\CODEX_GOAL_CASHFLOW_EVIDENCE_HANDOFF.md docs\CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md` | Passed; staged only goal-owned docs |
| `git diff --cached --name-status` | Passed; staged the two new handoff/audit docs and roadmap update |
| `git diff --cached --check` | Passed |
| `git diff --cached --stat` | Passed; three docs changed |

## Subagent Evidence Incorporated

- Finance/Documents/Butler/Cashflow explorer confirmed Finance is the strongest current fit, but root services still load DB records and enforce permissions; banking detection should be pure while match/finalize stays in permissioned Go services. It also confirmed Cashflow Evidence has no first-class package and should become a read model, not a new authority layer.
- Compliance/Inventory/Sync/Math explorer confirmed compliance engines have a strong kernel shape but no `pkg/compliance/ports.go`; inventory arithmetic and serial lifecycle are split across root App, CRM, and Svelte surfaces; sync has multiple partial stacks and should move toward module-aware envelopes; math kernels need explicit proof/test/heuristic labels.
- Both explorers independently recommended keeping Butler as inspect/explain/draft/recommend and routing create/persist/post/file behavior through deterministic domain services.

## Epistemic Status

- File/path evidence is verified by current repo inspection during this run.
- Prior Wave 17 verification is historical evidence from `docs/WAVE17_PROGRESS.md` and memory notes; it is not rerun in this docs-only audit.
- Module contract recommendations are assumption-scaffolded architecture guidance based on current code seams.
- Deterministic tax/posting/math claims are limited to the current implementation and tests; jurisdiction law freshness and mathematical proof status must be checked before external release claims.
- Williams batching has a proof pointer in code and local implementation tests; broader VQC/quaternion/three-regime product uses are empirical or heuristic unless tied to proof/test evidence in the relevant package.

## Exit Criteria

- `docs/CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT.md` exists.
- Engine inventory matrix covers Finance, Documents, Butler, Compliance, Inventory, Sync/CDC/Observability, Math/Optimization, and Cashflow Evidence.
- Top reusable kernels are ranked and mapped to product loops.
- Coupling risks and refactor recommendations are documented.
- Next handoff target is named as `docs/CODEX_GOAL_CASHFLOW_EVIDENCE_HANDOFF.md`.
- Docs-only verification passes.
- A coherent goal-owned commit/checkpoint captures the audit and next handoff.
