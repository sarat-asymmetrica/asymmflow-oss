# AsymmFlow Master Goal Roadmap

Status: Superseded planning spine; see sovereign substrate roadmap
Created: 2026-05-14
Scope: `the AsymmFlow repository`

## 2026-05-22 Supersession Note

This document remains useful historical planning evidence, but the active north
star is now sovereign operational software, not modular ERP/SaaS productization.

Use these current planning docs before deriving new implementation goals:

- `docs/SOVEREIGN_SOFTWARE_CONSTITUTION.md`
- `docs/KERNEL_CONSTITUTION.md`
- `docs/OVERLAY_BOUNDARY_GUIDE.md`
- `docs/EDITION_AND_FORK_GOVERNANCE.md`
- `docs/AI_REPAIR_AGENT_WORKFLOW.md`
- `docs/IMPLEMENTATION_FIT_MATRIX.md`
- `docs/CAPABILITY_CATALOG_PLAN.md`
- `docs/SOVEREIGN_SUBSTRATE_6_MONTH_ROADMAP.md`

The new roadmap deliberately treats the current PH/trading/instrumentation
system as the first serious overlay proof, while extracting a stable primitive
kernel, reusable operational engines, harness-guided AI repair workflows, and
edition/fork governance for source-owned deployments.

## Purpose

This document is the planning spine for decomposing the next six months of AsymmFlow work into long-horizon Codex `/goal` runs.

Existing roadmaps, wave logs, and handoff files are evidence, not canon. The product vision has evolved. Future work should preserve the hard-won deterministic ERP nucleus while reorienting the system toward reusable engines, modular product loops, and $1000/mo-value business capabilities.

## Operating Thesis

AsymmFlow should become a modular, local-first business operating system for owner-led and mid-market companies in India, GCC, Africa, South Asia, and similar markets.

The system should grow toward Zoho / ERPNext breadth without copying their architecture. The AsymmFlow route is:

```text
Core Kernel -> Domain Module -> ViewModel -> UI Component -> Agent Surface
```

The inspiration from Convex is conceptual: backend capability should be as composable as frontend components. A module should package data contracts, deterministic functions, persistence rules, events, permissions, viewmodels, UI surfaces, and agent-safe APIs behind explicit interfaces.

Do not build a pile of screens. Build composable business capability.

## Product Bar

Use the "$1000/mo justification paradigm" for major modules. Each major capability should be deep enough to plausibly justify $1000/mo in real customer value, even if final pricing is lower or owner-shaped.

Each module should close a real business loop:

```text
messy input -> canonical state -> intelligence -> user decision/action -> output -> audit trail -> measurable saved time/money
```

Apply four filters:

- ROI proof: what labor, subscription cost, cash leakage, compliance risk, or operational uncertainty does this remove?
- Workflow closure: does the user finish with an action, artifact, posting, export, reminder, claim pack, or decision?
- Engine leverage: which Asymmetrica engine, invariant, memory, optimization, OCR/classification, accounting, or local-first substrate makes this unfairly capable?
- Operator trust: can the user inspect, correct, approve, export, audit, and repeat the workflow?

## Architectural Defaults

Default architecture:

- Modular monolith, vertical slices by business domain.
- Go for backend authority and deterministic engines.
- Svelte/SvelteKit for apps.
- Astro for websites.
- MVVM by default.
- Cap'n Proto for durable schemas, local-first records, and engine/module contracts.
- TOON for compact agent/context transfer.
- JSON for browser payloads, third-party interop, simple configs, and cases where ecosystem tooling makes JSON the right choice.
- Strict compiler languages for core logic unless another language has a durable library advantage.
- Zero-warning policy: warnings are either fixed or documented as pre-existing baseline noise.
- Dependency additions require current authoritative version/license/maintenance checks before adoption.

Default product UI:

- Use `C:\Projects\ASYMMETRICA_PRODUCT_UI_AND_ARCHITECTURE_STANDARD.md`.
- Use the AsymmFlow workspace palette and surface language unless a product has a stronger justified brand.
- No generic UI colors, generic components, or visible jank.
- Use precision text/layout ideas from `C:\Projects\pretext` when text geometry affects correctness or polish.

## Engine Separation Rule

Every significant engine should be separable into:

- Pure kernel: calculation, invariant, scoring, optimization, matching, parsing, or transformation.
- Domain service: workflow-specific use, persistence, permissions, events, and audit.
- Storage adapter: SQLite/GORM/Cap'n Proto/import/export mapping.
- ViewModel adapter: presentation-ready state, commands, validation, async status, and correction flows.
- Agent adapter: constrained inspect/explain/draft/recommend operations that cannot bypass deterministic authority.

Example:

```text
posting kernel -> receivables service -> journal/source-link storage -> CashflowEvidenceVM -> Accounting/Cashflow UI -> Butler explanation surface
```

## Six-Month Goal Map

The map below is deliberately broad. Each goal should become one or more detailed handoff specs before implementation.

### Goal 1: Module Contract Foundation

Define the standard contract every AsymmFlow module must expose.

Target outcome:

- A repo-local module contract doc.
- A skeleton/example module manifest.
- Required sections for schemas, kernels, services, ViewModels, UI surfaces, events, permissions, audit, tests, and agent APIs.
- Decision rules for Cap'n Proto / TOON / JSON.
- A launch-readiness checklist for modules.

Why first:

This prevents future waves from creating one-off modules with incompatible shapes.

Candidate detailed spec:

- `CODEX_GOAL_MODULE_CONTRACT_HANDOFF.md`

Goal 1 artifacts:

- `docs/MODULE_CONTRACT_FOUNDATION.md`
- `docs/templates/module_manifest.example.json`
- `docs/CODEX_GOAL_MODULE_CONTRACT_HANDOFF.md`

Exit criteria:

- Contract doc exists.
- At least Finance, Documents, Butler, Compliance, Inventory, and Cashflow Evidence can be mapped onto it.
- Verification commands and future migration steps are listed.

### Goal 2: Engine Generalization Inventory

Audit existing engines and classify them into pure kernels, domain services, adapters, ViewModels, and agent surfaces.

Target engines:

- Finance posting, banking, payment, reporting.
- Document classification, OCR, Excel/email/PDF.
- Butler fastpath, prediction, reports, TOON chat.
- Compliance Bahrain VAT, India GST, India income tax.
- Sync/CDC/observability.
- Math and optimization packages.
- Inventory/procurement/fulfillment/serial traceability.

Target outcome:

- Engine inventory matrix.
- Reuse opportunities beyond industrial instrumentation.
- Generalization targets and naming.
- Risks where current code is too use-case-coupled.

Candidate detailed spec:

- `CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT.md`

Exit criteria:

- Every major existing engine has a recommended future shape.
- Top five reusable kernels are named.
- Top five product loops they can power are named.

### Goal 3: Cashflow + Evidence Command Center

First proof module under the new architecture.

Closed loop:

```text
orders/invoices/payments/docs/messages -> canonical evidence -> receivables risk -> follow-up/action -> draft posting/report/export -> audit trail
```

Why this module:

It directly matches real market pain: delayed payments, working capital stress, document evidence chaos, manual reconciliation, and owner uncertainty. It also reuses the strongest existing AsymmFlow assets.

Likely components:

- Receivables aging and cash exposure.
- Missing evidence detector.
- Invoice/payment/document source links.
- Posting coverage and trial-balance status.
- Follow-up queue.
- Counterparty risk/payment intelligence.
- Butler-grounded explanations.
- Exportable evidence pack.

Candidate detailed spec:

- `CODEX_GOAL_CASHFLOW_EVIDENCE_HANDOFF.md`

Exit criteria:

- Backend read model/service for cashflow evidence.
- ViewModel for the command center.
- UI surface with AsymmFlow visual standard.
- Tests for core calculations and service mapping.
- Build/check gates run.
- Docs updated with module status.

### Goal 4: Reusable Product Component Library

Create reusable UI components for AsymmFlow-class apps.

Component inventory should take inspiration from Chakra-style breadth but obey AsymmFlow visual and MVVM standards.

Initial component families:

- Tables and ledgers.
- Evidence timelines.
- Approval cards.
- Audit trails.
- KPI strips.
- Risk/status badges.
- Document preview panes.
- Reconciliation panels.
- Source-link chips.
- Empty/loading/error/correction states.
- Precision text cells and stable virtual rows.

Candidate detailed spec:

- `CODEX_GOAL_PRODUCT_COMPONENT_LIBRARY.md`

Exit criteria:

- Component inventory doc.
- First reusable Svelte components.
- Fixture page or story route.
- Visual checks at desktop/tablet/375px.
- Palette/token adherence.

### Goal 5: Business Memory Intake

Generalize document/communication intake into business memory.

Problem class:

Real business activity lives in WhatsApp, email, PDFs, Excel, scans, screenshots, voice notes, and folders. Operators reconstruct truth manually later.

Closed loop:

```text
unstructured message/document/folder -> classified object -> extracted fields -> review/correction -> linked business record -> audit trail
```

Initial scope should not depend on WhatsApp API access. Start with importable exports, screenshots, PDFs, emails, and folders. Add live connectors later.

Candidate detailed spec:

- `CODEX_GOAL_BUSINESS_MEMORY_INTAKE.md`

Exit criteria:

- Canonical intake model.
- Classifier/extraction review queue.
- Source-link persistence.
- Agent-readable context pack for Butler.
- UI review surface.

### Goal 6: Inventory + Asset Evidence Ledger

Reframe the old inventory roadmap into a generalized asset/evidence ledger.

Closed loop:

```text
purchase/receipt/reservation/delivery/serial evidence -> stock truth -> availability/valuation/risk -> action/export/audit
```

Why:

Existing industrial instrumentation traceability can generalize into inventory, assets, clinic stock, school supplies, construction materials, agriculture produce lots, and compliance evidence.

Candidate detailed spec:

- `CODEX_GOAL_INVENTORY_ASSET_LEDGER.md`

Exit criteria:

- Stock movement kernel.
- Warehouse/bin/reservation model.
- Serial/lot evidence links.
- Valuation baseline.
- ViewModel and readiness UI.

### Goal 7: Compliance Pack Architecture

Generalize Bahrain/India compliance engines into installable jurisdiction packs.

Closed loop:

```text
business events -> jurisdiction rules -> compliance calculations -> filing/export evidence -> audit trail
```

Candidate packs:

- Bahrain VAT.
- India GST.
- India income tax.
- GCC e-invoicing readiness.
- Future Africa/South Asia packs after market selection.

Candidate detailed spec:

- `CODEX_GOAL_COMPLIANCE_PACKS.md`

Exit criteria:

- Compliance pack interface.
- Existing engines adapted to pack shape.
- Event hooks documented.
- UI/readiness surfaces defined.

### Goal 8: Agent-Safe Module Surfaces

Define how Butler, Codex, and future agents interact with modules safely.

Principle:

Agents inspect, explain, draft, recommend, and assemble evidence. Deterministic services approve, persist, post, and enforce invariants.

Candidate detailed spec:

- `CODEX_GOAL_AGENT_SAFE_MODULE_SURFACES.md`

Exit criteria:

- Agent API rules.
- Read/write/approval permission levels.
- TOON context shapes.
- Butler command registry.
- Audit logging for agent-suggested actions.

### Goal 9: Local-First Sync And Module State

Make module state replication and audit explicit.

Closed loop:

```text
local action -> event/CDC -> sync envelope -> conflict policy -> remote/device state -> audit evidence
```

Candidate detailed spec:

- `CODEX_GOAL_LOCAL_FIRST_MODULE_SYNC.md`

Exit criteria:

- Module-aware sync contract.
- Conflict policy by record class.
- Offline readiness indicators.
- Support bundle for sync evidence.

### Goal 10: Launch Packaging And Pilot Operating System

Turn AsymmFlow from a powerful repo into a pilotable product.

Target outcome:

- Installable builds.
- Demo tenant/sample data.
- Setup/import wizard.
- Support bundle export.
- Known issues.
- Smoke checklist.
- Module readiness dashboard.
- Operator guide.

Candidate detailed spec:

- `CODEX_GOAL_PILOT_OPERATING_SYSTEM.md`

Exit criteria:

- A non-developer operator can install, configure, import data, inspect module readiness, and run the core loops.

## Suggested Goal Order

Recommended first sequence:

1. Module Contract Foundation.
2. Engine Generalization Inventory.
3. Cashflow + Evidence Command Center.
4. Reusable Product Component Library.
5. Business Memory Intake.
6. Inventory + Asset Evidence Ledger.

Rationale:

- Goals 1 and 2 create the map.
- Goal 3 proves the new architecture against a high-value market pain.
- Goal 4 prevents UI drift as more modules are added.
- Goal 5 unlocks messy real-world data ingestion.
- Goal 6 generalizes the repo's industrial inventory strengths.

## Completion-Ladder Execution Model

AsymmFlow goal runs should use roadmap-chain completion ladders, not wall-clock time boxes, as the primary completion model.

The orchestrator should not stop merely because one goal is complete. If a goal reaches an auditable checkpoint, the orchestrator must commit or checkpoint it, then move to the next queued checkpoint or roadmap set.

Required loop:

1. Record start time, optional safety/check-in cap, current commit, and dirty worktree boundary.
2. Read required ecosystem/repo contracts.
3. Define the obvious baseline work and triple it into the mandatory ladder because agent estimates have been low by about 3-4x.
4. Execute each checkpoint through artifacts, verification, docs, and commit/checkpoint.
5. After each rollback-safe commit, advance to the next checkpoint.
6. If the mandatory ladder finishes quickly, auto-promote the next queued roadmap set and complete at least one real, tested checkpoint there before stopping.
7. On human check-in, report current checkpoint, changed paths, verification state, risks, and next intended move.
8. Stop only when the ladder plus promoted continuation checkpoint is complete, a concrete blocker is recorded, or verification failure needs human review.

This preserves long autonomous momentum while making progress inspectable through commits rather than relying on poor wall-clock estimates.

## Goal Spec Template

Each broad goal should be expanded into a detailed handoff spec with this shape:

```text
# CODEX_GOAL_<NAME>_HANDOFF

## Goal Command

/goal Run the [goal chain/window] for 2 hours. Complete [goal name] through [verifiable end state]; if it finishes early, checkpoint it and advance to the next queued roadmap goal.

## Current Context

- Existing repo evidence.
- Relevant commits/docs.
- Relevant engines/modules.
- Dirty worktree boundaries.

## Product Bar

- $1000/mo justification.
- ROI proof.
- Workflow closure.
- Engine leverage.
- Operator trust.

## Architecture Constraints

- Kernel/domain/ViewModel/UI/agent split.
- MVVM rules.
- Data contract choices.
- Dependency policy.
- UI standard.

## Worker Wave Plan

- Orchestrator duties.
- Start time, stop/checkpoint time, and check-in cadence.
- Worker 1 ownership.
- Worker 2 ownership.
- Worker 3 ownership.
- Verification lead ownership.

## Verification Gates

- Go tests/build.
- Frontend check/build.
- Focused UI/browser checks.
- Docs/work-log update.

## Commit Policy

- Preserve unrelated dirty work.
- Consolidated rollback-safe commit.
- Record baseline warnings separately.

## Exit Criteria

- Concrete artifacts.
- Commands run.
- Residual risks.
- Next goal queue.
- Time-box checkpoint state.
```

## Meta-Framework For Other Repos

Use the same pattern across Asymmetrica repos:

1. Read ecosystem and repo-local contracts.
2. Treat old roadmaps as evidence, not canon.
3. Inventory committed capabilities from history and code.
4. Identify reusable kernels and product loops.
5. Apply the $1000/mo justification filters.
6. Decide module contract and UI/architecture constraints.
7. Write one broad master roadmap if missing.
8. Write one detailed `/goal` handoff for the next run.
9. Dispatch long-horizon orchestrator with subagents.
10. Verify, document, commit, and queue the next goal.

## Current Next Spec

`CODEX_GOAL_MODULE_CONTRACT_HANDOFF.md` was completed in commit `c1cf4e8` on 2026-05-14.

`CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT.md` was completed as the Goal 2 inventory artifact on 2026-05-14 from starting commit `85dcf61`.

`CODEX_GOAL_CASHFLOW_EVIDENCE_HANDOFF.md` now has an initial backend-to-UI proof checkpoint: module manifest, pure/read-model package, service seam, invoice traceability and follow-up task source adapters, ViewModel contract, Wails endpoint/bindings, Butler-facing agent-safe TOON brief, operator-visible read-only action proposals, persisted proposal review/signoff queue with Accounting signoff controls, deterministic JSON/TOON evidence-pack export, Accounting dashboard command-center surface, focused tests, `go build ./...`, and frontend check/build gates.

`CODEX_GOAL_PRODUCT_COMPONENT_LIBRARY_HANDOFF.md` completed in `9c88fb8 feat(asymmflow): extract reusable product components`, with follow-up handoff `aa578b2 docs(asymmflow): add business memory intake handoff`.

`CODEX_GOAL_BUSINESS_MEMORY_INTAKE_HANDOFF.md` completed through Checkpoints A-D in commits `7fb962f`, `29ebe21`, `a57836f`, `cb4421e`, and final ledger commit `8045a7d`.

The next recommended goal is now `CODEX_GOAL_BUSINESS_MEMORY_DURABLE_CONTRACT_HANDOFF.md`.

Important execution correction: future handoffs should use a completion ladder with a 3x ambition floor, not a 2-hour time box. The orchestrator must not stop merely because the obvious checklist is complete. After each verified commit, it should continue to the next checkpoint; if the mandatory ladder finishes quickly, it should promote the next roadmap set into the same run.
