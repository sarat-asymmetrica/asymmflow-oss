# AsymmFlow Development Roadmap - 2026-05-15

Scope: `the AsymmFlow repository`

## 2026-05-22 Sovereign Direction Note

This roadmap remains useful tactical evidence for Business Memory, Cashflow
Evidence, source-track reconciliation, and pilot packaging, but it is no longer
the active top-level north star.

New implementation goals should first consult:

- `docs/SOVEREIGN_SOFTWARE_CONSTITUTION.md`
- `docs/KERNEL_CONSTITUTION.md`
- `docs/OVERLAY_BOUNDARY_GUIDE.md`
- `docs/EDITION_AND_FORK_GOVERNANCE.md`
- `docs/AI_REPAIR_AGENT_WORKFLOW.md`
- `docs/IMPLEMENTATION_FIT_MATRIX.md`
- `docs/CAPABILITY_CATALOG_PLAN.md`
- `docs/SOVEREIGN_SUBSTRATE_6_MONTH_ROADMAP.md`

The current strategic objective is to transform `asymmflow` from a
PH/trading/instrumentation ERP refactor into the canonical sovereign operational
substrate: primitive kernel, reusable engines, domain overlays, sovereign forks,
harness-guided AI repair, and source-owned edition packaging.

This roadmap converts the evidence-first audit into a checkpointed 2-4 week sprint ladder. It assumes the repo remains pre-release and rollback-by-commit is the safety model. It does not authorize app behavior changes by itself; each sprint should become a concrete implementation handoff before coding.

## Operating Principle

Do not add more foundation until a product loop closes.

The next sprint chain should turn the May 14 modules into operator-grade loops:

```text
messy evidence -> canonical state -> deterministic/read-model intelligence -> operator decision -> export/action -> audit trail
```

Agents may inspect, explain, draft, recommend, and assemble evidence. Deterministic services own approve, link, post, persist, delete, reverse, and other authoritative mutations.

## Authority Zones

### Green

- `pkg/documents/intake`
- `pkg/cashflow/evidence`
- `pkg/adapter/documents`
- `internal/viewmodel/documents`
- `internal/viewmodel/cashflow`
- `business_memory_review_runtime.go`
- `cashflow_evidence_service.go`
- `cashflow_evidence_review.go`
- `service_documents.go`
- `service_finance.go`
- `frontend/src/lib/components/documents/BusinessMemoryReviewPanel.svelte`
- `frontend/src/lib/screens/InboxScreen.svelte`
- `frontend/src/lib/screens/AccountingScreen.svelte`
- `frontend/src/lib/components/ui/ActionProposalCard.svelte`
- `frontend/src/lib/components/ui/EvidenceSourceList.svelte`
- `frontend/src/lib/components/ui/KpiStatusStrip.svelte`
- `docs/modules/*.manifest.json`
- `docs/*STATUS*.md`, `docs/*ROADMAP*.md`, and goal handoffs

### Amber

- `schemas/documents.capnp`, `schemas/finance.capnp`, generated `schemas/go/*`, and `frontend/src/lib/types/schemas/index.ts`
- `frontend/wailsjs/**`
- `pkg/infra/events`
- `pkg/sync/**`
- `database.go` and migrations
- Root app/service binding files outside the named Green files

Amber edits require recorded rationale, focused generated-artifact review, and verification gates.

### Red

- Unrelated PH/client production behavior outside the sprint module paths
- Broad UI redesign
- Live credential/config changes
- Destructive data migrations
- Supabase/Fly/VPS deployment changes
- Generated binding churn without a matching runtime/schema change

## Sprint Ladder

### Sprint 0: Acme Instrumentation Source-Track Reconciliation

Goal: keep the rough client-feedback track as an acceptance benchmark while preserving the cleaner refactor architecture.

Current source evidence:

- `C:\Projects\asymmflow\ph_holdings` is up to date with `origin/main` at `88dac32`.
- Latest source-track commits emphasize offer reference/revision correctness, final training deployment, bank reconciliation allocations, expense targets, user activity telemetry, delete approvals, opportunity conflict review, sync normalization, UI/backend action inventories, OCR RBAC routing, and user guides.
- The corresponding artifacts or equivalents are already present in `asymmflow`; this is not a blind backport task.

Checkpoints:

1. Use `docs/CODEX_PH_HOLDINGS_SOURCE_TRACK_RECON_2026_05_15.md` as the current source-track reconciliation artifact.
2. Convert client-feedback features into acceptance scenarios for module work.
3. Add Sales Revision Integrity as a required launch gate for quote-to-cash.
4. Add allocation-aware evidence requirements to Cashflow Evidence.
5. Treat delete approvals and opportunity conflict review as reusable review/request patterns for agent-safe module surfaces.
6. Fold UI/backend action inventories and user guides into launch-readiness gates.

ROI proof: protects the refactor from losing hard-won client reality while avoiding chaotic file-level merging.

Workflow closure: each rough-track lesson becomes a testable roadmap constraint or acceptance scenario.

Engine leverage: existing sales pipeline revision logic, bank allocation records, delete approval service, activity telemetry, sync normalization, UI/backend audit scripts, and module manifests.

Operator trust: preserves client-observed correctness around offers, payments, destructive actions, training, and deployment support.

Verification gates:

- No product code changes for this checkpoint.
- `git status` in both repos.
- Source-track pull/fetch evidence.
- Roadmap/doc diff review.

Needs human steering:

- Whether `ph_holdings` should remain a permanent rough-source branch or become read-only after one more reconciliation pass.
- Which client feedback items should become blocking pilot gates versus backlog hardening.

### Sprint 1: Business Memory Durable Source Registry

Goal: make the source asset registry durable, reviewable, exportable, and event-aware enough to serve as the intake provenance spine.

Checkpoints:

1. Add storage boundary for `SourceAssetRegistry` with idempotent upsert/list/get semantics.
2. Bridge registry records to Business Memory review queue state and export bundles.
3. Publish or record module events for candidate review, link request, context assembly, and source registry upsert.
4. Add operator-visible source registry summary in the Inbox Business Memory surface.
5. Update manifest/status docs and produce a rollback-safe commit.

ROI proof: reduces document hunting and duplicate intake confusion; supports future import folders and WhatsApp/email exports.

Workflow closure: operator can see what source generated a candidate, what happened to it, and export a bundle that preserves source provenance.

Engine leverage: `pkg/documents/intake`, Cap'n Proto document contract, TOON context pack, local-first repository adapters.

Operator trust: source ID, path/label/hash, privacy class, processing status, candidate IDs, audit refs, and export replay evidence.

Verification gates:

- `go test ./pkg/documents/... -count=1`
- `go test ./pkg/adapter/documents -count=1`
- `go test ./internal/viewmodel/documents -count=1`
- `go build ./...`
- `npm.cmd --prefix frontend run check`
- Manifest JSON parse
- Schema check-only if schema changes

Needs human steering:

- Privacy classes and retention expectations.
- Whether source hashes should be mandatory for imported files before pilot.
- Naming of operator-visible "Business Memory" versus "Inbox Evidence" surfaces.

### Sprint 2: Cashflow Evidence Operator Loop

Goal: move Cashflow Evidence from command-center proof into a usable operator loop with review state, evidence exports, and subscriptions to finance/document facts.

Checkpoints:

1. Add drill-down lists from risk/coverage rows to source documents or finance records.
2. Add allocation-aware evidence for customer invoices, supplier invoices, expenses, and partial/mixed bank matches. Initial pure preflight is implemented in `pkg/cashflow/evidence` and `internal/viewmodel/cashflow`; future work should wire real snapshot sources and operator review state.
3. Add review states for action proposals: draft, needs input, approved for deterministic action, rejected, exported.
4. Add export support bundle that includes cash exposure, missing evidence, allocation state, proposal decisions, posting coverage, and trial-balance state.
5. Wire read-only event subscriptions or snapshot refresh points for invoice, payment, bank statement, allocation, and document classification events.
6. Update Accounting surface with compact operator actions without auto-posting.

ROI proof: reduces receivables chasing, accountant back-and-forth, and time spent reconstructing missing payment evidence.

Workflow closure: operator ends with a review state, follow-up/export, or deterministic posting request, not just a dashboard.

Engine leverage: `pkg/cashflow/evidence`, `pkg/finance/posting`, banking matching, document classification, Butler TOON brief.

Operator trust: source provenance, trial-balance status, posting coverage, proposal review history, export bundle, no hidden auto-mutation.

Current preflight status:

- `CommandCenterInput` and `CommandCenter` can carry bank allocation evidence with allocation ID, bank statement line ID, source type/ID, amount, allocation type, confidence, and allocation status.
- `EvidencePack` exports allocation summary/detail as read-model evidence.
- `CommandCenterVM` exposes allocation rows for operator inspection.
- No bank reconciliation mutation path is touched by this preflight.

Verification gates:

- `go test ./pkg/cashflow/evidence -count=1`
- `go test ./internal/viewmodel/cashflow -count=1`
- Focused root tests for cashflow review/runtime methods if changed
- `go build ./...`
- `npm.cmd --prefix frontend run check`
- Manifest JSON parse

Needs human steering:

- Which cashflow actions are allowed to become deterministic requests in the pilot.
- Export format priority: JSON/TOON first, CSV/PDF later, or immediate operator PDF.
- Whether action approvals should use finance permissions or new cashflow permissions.

### Sprint 3: Inventory + Asset Evidence Ledger Preflight

Goal: start the first pure stock/evidence kernel checkpoint without disturbing existing inventory screens or serial-number behavior.

Checkpoints:

1. Write an inventory/asset evidence handoff from current repo evidence.
2. Implement pure stock movement and evidence link types in a new package.
3. Add fixture-backed tests for receipt, reservation, delivery, serial/lot evidence, and valuation baseline.
4. Add a manifest draft mapping sources from purchase orders, GRNs, delivery notes, serial numbers, and Business Memory source assets.
5. Stop before runtime mutation unless the pure kernel and handoff are verified.

ROI proof: reduces stock ambiguity, delivery disputes, serial evidence hunting, and valuation uncertainty.

Workflow closure: first checkpoint produces a tested kernel and manifest that can later power a readiness UI and audit/export surface.

Engine leverage: existing procurement/fulfillment/serial traceability, Business Memory source registry, Cap'n Proto/JSON manifest split.

Operator trust: evidence-linked movements, no mutation of current stock state until kernel behavior is inspectable.

Verification gates:

- New package tests
- Existing fulfillment/CRM focused tests if touched
- `go build ./...`
- Manifest JSON parse
- No generated binding changes unless explicitly authorized

Needs human steering:

- Pilot domain priority: Acme Instrumentation stock, generic asset ledger, or broader inventory module.
- Valuation method for first checkpoint.
- Whether inventory belongs in the v0.1 release path before or after Cashflow closure.

### Sprint 4: Agent-Safe Module Surface Hardening

Goal: prove that inspect/explain/draft/recommend surfaces cannot mutate deterministic authority.

Checkpoints:

1. Create shared agent-surface rules and tests for Business Memory and Cashflow Evidence.
2. Add denial tests for agent actors attempting approve/link/post/delete/create-authoritative-record operations.
3. Reuse the delete-approval pattern for irreversible agent-suggested actions: request first, operator/admin approval second, deterministic service mutation last.
4. Add audit references or event records whenever an agent-generated suggestion is accepted by an operator.
5. Define TOON context shape requirements for source citation and forbidden operations.
6. Update manifests and module contract docs with reusable agent-surface checklist.

ROI proof: prevents failed-agent cleanup and makes agent assistance safe enough for serious operators.

Workflow closure: every agent suggestion is inspectable, citeable, and operator-approved before deterministic services act.

Engine leverage: TOON context packs, deterministic review services, module manifests, event/audit bus.

Operator trust: explicit forbidden operations, actor typing, audit references, and mutation rejection tests.

Verification gates:

- `go test ./pkg/documents/... -count=1`
- `go test ./pkg/cashflow/evidence -count=1`
- Relevant adapter/ViewModel tests
- `go build ./...`
- Manifest JSON parse

Needs human steering:

- Whether Butler and Codex share the same permission model or need distinct actor classes.
- How much agent activity should appear in the normal audit trail versus a separate suggestion log.

### Sprint 5: Pilot Packaging And Launch Readiness

Goal: package the strongest closed loops into a pilotable operator surface instead of a repo-only demo.

Checkpoints:

1. Define module readiness dashboard fields: Cashflow, Business Memory, Posting, Release, Inventory preflight.
2. Add support bundle export index that references Business Memory and Cashflow evidence bundles.
3. Add Sales Revision Integrity to the quote-to-cash launch gate: active costing revision, offer reference numbering, PDF reference stability, and revision regression tests.
4. Add UI/backend action inventory review as a launch gate so visible actions map to backend commands, permissions, and test coverage.
5. Update release checklist with module readiness gates and baseline warnings.
6. Run `scripts/verify_release.ps1` when the preceding loops are stable enough for a release candidate.
7. Create a short operator smoke path for install, open, review Business Memory, inspect Cashflow Evidence, export bundle, and confirm quote revision behavior.

ROI proof: turns implementation into something a non-developer can trial and report on.

Workflow closure: operator can install, inspect readiness, run a core loop, export evidence, and send support data.

Engine leverage: release tooling, module manifests, evidence exports, deterministic posting gates, local-first app paths.

Operator trust: known issues, readiness states, support bundles, explicit baseline warnings, rollback-safe release artifacts.

Verification gates:

- `go test ./pkg/documents/... -count=1`
- `go test ./pkg/cashflow/evidence -count=1`
- `go test ./internal/viewmodel/... -count=1`
- `go build ./...`
- `npm.cmd --prefix frontend run check`
- `scripts/verify_release.ps1` when release packaging is in scope
- Manual app smoke or browser/app walkthrough if UI changed

Needs human steering:

- Pilot user persona and dataset.
- Whether the first pilot should emphasize Cashflow Evidence, Business Memory, or the older trading/distribution loop.
- Known-issues threshold for a beta candidate.

## Commit Cadence

Commit after each coherent checkpoint that passes its verification gate:

- Business Memory registry durability.
- Cashflow Evidence operator loop.
- Inventory/Asset pure preflight.
- Agent-safe surface hardening.
- Launch-readiness packaging.

Do not batch unrelated module work into one commit. Generated artifacts should be in the same commit as the schema/runtime change that required them.

## Decision Queue

1. Product naming: Business Memory, Inbox Evidence, or another operator-facing term.
2. First pilot wedge: Cashflow collections, Business Memory intake, or trading/distribution release hardening.
3. Event bus policy: in-memory module events only for now, or durable event log before launch.
4. Export priority: JSON/TOON support bundles first, or human-readable PDF/CSV in the same sprint.
5. Inventory scope: pure stock ledger first, serial/lot evidence first, or asset evidence ledger first.
6. Source-track policy: keep `ph_holdings` as an active rough client-feedback track, or freeze it as a comparison baseline once the refactor line owns future development.
7. Pilot gate policy: decide which source-track lessons are launch blockers: offer revisions, bank allocations, delete approvals, activity/training telemetry, UI/backend action inventory, or all of them.

## Stop Conditions

- A verification gate fails in a way that requires behavior decisions rather than mechanical repair.
- Schema/generated churn exceeds the planned module boundary.
- A migration would touch client data destructively or broadly.
- The work needs live credentials, deployment access, or real customer data.
- Human steering is needed for naming, pilot wedge, privacy/retention, or authoritative mutation policy.

## Next Recommended Handoff

Use the next implementation handoff:

```text
docs/CODEX_GOAL_BUSINESS_MEMORY_SOURCE_REGISTRY_DURABILITY_HANDOFF.md
```

Minimum expected end state:

- Source asset registry has durable storage or an explicit repository adapter.
- Inbox review/export surfaces include source registry provenance.
- Business Memory status and manifest are synchronized.
- Focused tests, `go build ./...`, `npm.cmd --prefix frontend run check`, manifest parse, and schema check-only if needed have run.
- A rollback-safe commit exists.
