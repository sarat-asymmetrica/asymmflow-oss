# CODEX_GOAL_BUSINESS_MEMORY_INTAKE_HANDOFF

Status: Ready for Codex
Revised: 2026-05-14
Repo: `the AsymmFlow repository`
Clean base expected: `aa578b2 docs(asymmflow): add business memory intake handoff`
Depends on:

- `9c88fb8 feat(asymmflow): extract reusable product components`
- `aa578b2 docs(asymmflow): add business memory intake handoff`

## Tiny Goal Prompt

The human prompt should be this small:

```text
/goal In the AsymmFlow repository, read docs\CODEX_GOAL_BUSINESS_MEMORY_INTAKE_HANDOFF.md and execute the full 2-hour roadmap-chain run exactly as specified.
```

Everything else needed for the run is inside this file.

## Non-Negotiable Goal Contract

This is a 2-hour autonomous roadmap-chain run, not a single-ticket run and not a docs-only planning task.

Do not stop after the first verified commit. After every checkpoint:

1. Check elapsed wall-clock time with PowerShell.
2. If more than 20 minutes remain, continue to the next checkpoint immediately.
3. Implement at least one real, tested slice in the next checkpoint before considering the run complete.

Creating or updating a handoff is not sufficient completion unless:

1. Less than 20 minutes remain in the 2-hour window.
2. Implementation is blocked by a concrete missing decision that cannot be resolved from repo context.
3. Verification or repair has consumed the remaining window.
4. All checkpoints A through D below are complete, verified, committed, and documented.

The goal should be marked complete only when the 2-hour window is near expiry, the repo is clean or explicitly checkpointed, verification evidence is recorded, and the next action is unambiguous.

## Why This Spec Is Written This Way

The previous Product Component Library run completed a useful primary slice in about 22 minutes and then stopped because the handoff treated the next document as the early-completion target. That was too small for Codex `/goal`.

This handoff fixes that failure mode:

- The time box is the operating container.
- Milestones are checkpoints, not the definition of done.
- A documentation-only checkpoint does not end the run while implementation time remains.
- The spec gives authority zones so the agent does not pause merely because a useful file is outside a narrow write list.
- The orchestrator must use subagents for coding or verification work when disjoint ownership is available.

Codex `/goal` is a persistent long-running objective, but Codex still decides completion from local acceptance criteria. Therefore this file makes the continuation rule more explicit than any individual checkpoint.

## Mission

Build the first serious Business Memory Intake chain inside AsymmFlow:

```text
unstructured message/document/folder
-> classified intake candidate
-> extracted fields and source evidence
-> review/correction state
-> suggested deterministic link/action
-> Butler-readable context pack
-> audit trail and next module queue
```

This is not "upload a file." This is the beginning of business memory: the system should turn messy source material into inspectable, correctable, linkable, and auditable operating context.

## Product Bar

Use the "$1000/mo justification paradigm."

This capability earns its keep by reducing manual document hunting, data retyping, accountant/operator context loss, follow-up reconstruction, and evidence chaos across WhatsApp exports, email, PDFs, Excel files, screenshots, scans, and folders.

Apply four filters before every checkpoint:

- ROI proof: name the labor, rework, cash leakage, compliance risk, or operator uncertainty reduced by the slice.
- Workflow closure: end with a reviewed candidate, link proposal, context pack, persisted review state, or operator-ready artifact.
- Engine leverage: reuse existing OCR, classifier, Inbox, source-link, TOON, Butler, and ViewModel seams instead of adding a disconnected point tool.
- Operator trust: show source provenance, extracted-field status, confidence, correction path, deterministic service target, audit trail, and repeatability.

## Current Baseline

Recently completed:

- Cashflow Evidence command-center proof module.
- Product Component Library first slice.
- Reusable UI components:
  - `KpiStatusStrip`
  - `EvidenceSourceList`
  - `ActionProposalCard`
- `AccountingScreen.svelte` now consumes those components without changing Cashflow backend behavior.
- `ShowcaseScreen.svelte` includes product operator component fixtures.
- Business Memory Intake handoff exists but was revised by this file to remove early-stop ambiguity.

Known baseline verification context:

- Frontend check/build recently passed with 13 existing Svelte warnings.
- Wails anonymous-struct warning may remain baseline noise.
- Treat baseline warnings as acceptable only if unchanged and reported clearly.

## Required Read Order

Read all of these before editing:

1. `C:\Projects\ASYMMETRICA_ECOSYSTEM_LOG.md`
2. `C:\Projects\AGENTIC_SWARM_PROTOCOL.md`
3. `C:\Projects\ASYMMETRICA_PRODUCT_UI_AND_ARCHITECTURE_STANDARD.md`
4. `docs\CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md`
5. `docs\MODULE_CONTRACT_FOUNDATION.md`
6. `docs\CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT.md`
7. `docs\PRODUCT_COMPONENT_LIBRARY_INVENTORY.md`
8. `docs\CODEX_GOAL_BUSINESS_MEMORY_INTAKE_HANDOFF.md`
9. `pkg\documents\ports.go`
10. `pkg\documents\domain.go`
11. `pkg\documents\classifier\service.go`
12. `document_classifier.go`
13. `ocr_service_simple.go`
14. `runtime_handlers.go`
15. `internal\viewmodel\documents\documents_vm.go`
16. `frontend\src\lib\screens\InboxScreen.svelte`
17. `frontend\src\lib\components\ui\index.ts`
18. `docs\modules\cashflow_evidence.manifest.json`

Treat older wave docs as evidence, not current canon.

## Architecture Rules

Preserve the AsymmFlow authority model:

```text
pure kernel -> domain/read service -> storage adapter -> ViewModel -> UI -> agent-safe context
```

Rules:

- Backend/domain services own authority, invariants, persistence, and side effects.
- ViewModels own display-ready state, commands, validation labels, async status, and correction flows.
- Svelte renders state and emits user intent.
- Agents and Butler may inspect, explain, classify, draft, recommend, and assemble context packs.
- Agents and Butler may not approve, link, post, delete, or mutate business authority without deterministic service approval.
- Do not replace the existing Inbox/OCR/classifier stack wholesale.
- Normalize existing outputs before inventing new parsers.
- Start with importable files, folder scans, emails, PDFs, Excel, screenshots, and existing inbox records.
- Do not implement live WhatsApp API access in this run.
- Do not add dependencies unless current stable version, license, maintenance, and runtime impact are checked from authoritative sources and documented.
- Follow the AsymmFlow UI standard. No generic colors, no generic dashboards, no janky or one-off review cards.

## Existing Seams

| Area | Current seam | Use in this run |
| --- | --- | --- |
| Document ports | `pkg/documents/ports.go` | Understand existing OCR/classifier/parser boundaries. |
| Document domain | `pkg/documents/domain.go`, `schemas/documents.capnp` | Reuse document/source metadata language where possible. |
| Classifier package | `pkg/documents/classifier/service.go` | Prefer package-level deterministic classifier logic for reusable tests. |
| App classifier facade | `document_classifier.go` | Keep runtime/API behavior stable; use only if integration requires it. |
| OCR/extraction | `ocr_service_simple.go`, `pkg/documents/ocr`, `pkg/documents/excel`, `pkg/documents/email` | Normalize output; do not replace services. |
| Inbox runtime | `runtime_handlers.go` | Existing `InboxProcessResult`, `InboxDocument`, and process/save seams. |
| ViewModel | `internal/viewmodel/documents/documents_vm.go` | Extend with intake review display state. |
| UI surface | `frontend/src/lib/screens/InboxScreen.svelte` | Add bounded review/correction surface without redesigning the whole page. |
| Product UI | `KpiStatusStrip`, `EvidenceSourceList`, `ActionProposalCard` | Reuse for metrics, provenance, and suggested deterministic actions. |
| Cashflow proof | `docs/modules/cashflow_evidence.manifest.json` | Pattern for module manifest structure. |

## Authority Zones

Use these zones instead of stopping for a new handoff.

### Green Zone: Pre-Approved

The orchestrator and workers may edit these freely if the changes serve this goal:

- `docs\CODEX_GOAL_BUSINESS_MEMORY_INTAKE_HANDOFF.md`
- `docs\modules\business_memory_intake.manifest.json`
- `docs\BUSINESS_MEMORY_INTAKE_STATUS.md`
- `pkg\documents\memory\**`
- `pkg\documents\intake\**`
- `internal\viewmodel\documents\**`
- `frontend\src\lib\screens\InboxScreen.svelte`
- `frontend\src\lib\components\documents\**`
- `frontend\src\lib\components\ui\**` only for small reusable component improvements
- `frontend\src\lib\components\ui\index.ts`
- tests and fixtures directly tied to the above

### Amber Zone: Allowed With Rationale

The agent may edit these if implementation needs it. Record why in the checkpoint notes and keep the diff tight:

- `runtime_handlers.go`
- `document_classifier.go`
- `ocr_service_simple.go`
- `pkg\documents\ports.go`
- `pkg\documents\domain.go`
- `pkg\documents\classifier\**`
- `schemas\documents.capnp`
- `wailsjs\**` generated bindings, only after `wails generate module`

### Red Zone: Do Not Touch In This Run

- Live WhatsApp API integration.
- New cloud dependencies.
- Autonomous accounting, CRM, inventory, or procurement mutations.
- Broad redesign of Inbox or Accounting.
- Cross-repo writes outside `the AsymmFlow repository`.
- OpenSwarm fork or `C:\Projects\openswarm-asymmetrica`.
- Real private client data fixtures.

## Run Ledger Requirements

At the start, record in this handoff under "Run Ledger":

- Start time from PowerShell.
- Planned 2-hour stop time.
- Starting commit.
- Starting `git status --short --branch`.
- Active checkpoint.
- Subagents spawned and ownership.

After each checkpoint, append:

- Completed changes.
- Commit hash if committed.
- Verification commands run.
- Baseline warnings/noise.
- Elapsed time.
- Next checkpoint selected.

At final stop, append:

- Final HEAD.
- Final status.
- Commands run.
- Residual risks.
- Next recommended `/goal`.

## Subagent Requirement

The orchestrator must use subagents unless the Codex environment makes that impossible.

Minimum delegation:

| Worker | Ownership | Expected output |
| --- | --- | --- |
| Worker A: intake kernel | `pkg/documents/memory` or `pkg/documents/intake`, fixtures, pure tests | Canonical intake model and normalization helpers. |
| Worker B: ViewModel/UI | `internal/viewmodel/documents`, `InboxScreen.svelte`, optional `components/documents` | Review/correction display state and product component reuse. |
| Worker C: context/docs/verification | Butler TOON/context pack, module manifest/status doc, verification | Draft-only context contract, docs, command evidence. |
| Orchestrator | Architecture, integration, amber-zone decisions, commits | Integrated checkpoint chain and final clean state. |

Tell every worker:

- You are not alone in the codebase.
- Do not revert edits by others.
- Keep ownership disjoint.
- Report changed paths, verification, risks, and next suggested work.

If subagents are unavailable, the orchestrator must state that in the run ledger and still execute the same checkpoint chain locally.

## Checkpoint Chain

Complete these in order. Commit after each checkpoint if it is coherent and verification passes.

### Checkpoint A: Canonical Intake Kernel And Module Manifest

Goal:

Create the deterministic, testable model for Business Memory Intake.

Required outputs:

1. Add `docs\modules\business_memory_intake.manifest.json`.
2. Add a pure package, preferably one of:
   - `pkg\documents\memory`
   - `pkg\documents\intake`
3. Define a canonical intake candidate contract with fields equivalent to:
   - `id`
   - `source`
   - `source_kind`
   - `business_object_type`
   - `classification`
   - `extracted_fields`
   - `suggested_links`
   - `review_status`
   - `audit_refs`
   - `confidence`
   - `warnings`
4. Define source kinds:
   - `message`
   - `email`
   - `pdf`
   - `scan`
   - `screenshot`
   - `excel`
   - `folder`
   - `inbox_record`
   - `other`
5. Define review statuses:
   - `new`
   - `needs_review`
   - `corrected`
   - `linked`
   - `rejected`
   - `archived`
6. Define extracted-field status:
   - `extracted`
   - `missing`
   - `inferred`
   - `needs_confirmation`
   - `corrected`
7. Normalize at least these source shapes into intake candidates:
   - Existing `InboxProcessResult`.
   - Existing `InboxDocument`.
   - OCR/extraction map shape from current services, if it can be done without invasive runtime edits.
8. Add fixture-backed Go tests for normalization and status/confidence mapping.
9. Update or create `docs\BUSINESS_MEMORY_INTAKE_STATUS.md` with the contract and current checkpoint state.

Verification after Checkpoint A:

```powershell
go test ./pkg/documents/... -count=1
git diff --check
```

Commit if coherent:

```text
feat(asymmflow): add business memory intake kernel
```

Continuation rule:

- If more than 20 minutes remain, continue to Checkpoint B.
- Do not stop after Checkpoint A unless blocked or near time expiry.

### Checkpoint B: ViewModel And Inbox Review Surface

Goal:

Make the intake model visible and reviewable in the existing Inbox flow without redesigning the whole screen.

Required outputs:

1. Extend `internal\viewmodel\documents` with display-ready intake review state, such as:
   - queue metrics
   - selected candidate
   - extracted-field rows
   - source/provenance rows
   - action proposals
   - review commands or command labels
2. Add mapping helpers from canonical intake candidates into ViewModel state.
3. Add Go tests for ViewModel mapping if Go ViewModel logic changes.
4. Update `InboxScreen.svelte` or add a bounded child component under `frontend\src\lib\components\documents`.
5. Reuse product components:
   - `KpiStatusStrip` for intake queue/readiness metrics.
   - `EvidenceSourceList` for source/provenance completeness.
   - `ActionProposalCard` for suggested deterministic links/actions.
6. The UI must present actions as proposals or review choices, not as autonomous business mutations.
7. Preserve existing Process/Review/Archive behavior unless a tiny adapter is required.

Design constraints:

- Use the AsymmFlow product palette and density.
- Avoid nested cards.
- Keep text within containers at desktop and narrow widths.
- Do not add explanatory feature text in the app surface.
- The UI should feel like an operator review workbench, not a generic dashboard.

Verification after Checkpoint B:

```powershell
go test ./internal/viewmodel/documents -count=1
npm.cmd --prefix frontend run check
npm.cmd --prefix frontend run build
git diff --check
```

If Wails bindings changed:

```powershell
wails generate module
npm.cmd --prefix frontend run check
npm.cmd --prefix frontend run build
```

Commit if coherent:

```text
feat(asymmflow): add business memory intake review surface
```

Continuation rule:

- If more than 20 minutes remain, continue to Checkpoint C.
- Do not stop after Checkpoint B merely because the UI builds.

### Checkpoint C: Butler Context Pack And Agent-Safe Boundary

Goal:

Make reviewed/intake candidates consumable by Butler or future agents without giving agents mutation authority.

Required outputs:

1. Add a draft-only context pack shape for intake candidates.
2. Prefer a small pure Go helper that can emit compact text/TOON-like context from an intake candidate or queue.
3. The context pack must include:
   - candidate id
   - source summary
   - source kind
   - business object type
   - extracted fields and statuses
   - missing fields
   - suggested deterministic service target
   - review status
   - warnings
   - audit/source references
4. Explicitly label allowed agent actions:
   - inspect
   - explain
   - draft
   - recommend
   - assemble context
5. Explicitly label forbidden agent actions:
   - approve
   - link
   - post
   - delete
   - create authoritative business records
6. Update `docs\BUSINESS_MEMORY_INTAKE_STATUS.md` with the agent-safe contract.

Verification after Checkpoint C:

```powershell
go test ./pkg/documents/... -count=1
go build ./...
git diff --check
```

Commit if coherent:

```text
feat(asymmflow): add business memory context pack
```

Continuation rule:

- If more than 20 minutes remain, continue to Checkpoint D.
- A context pack alone is not final completion while time remains.

### Checkpoint D: Durable Review Queue Or Inventory Preflight

Choose D1 unless the implementation clearly shows persistence would be premature.

#### D1: Durable Review Queue Preparation Or Slice

Goal:

Move the review workflow toward persistence and repeatability.

Preferred outputs:

1. Inspect existing persistence boundaries for inbox review state.
2. If safe, add a small durable review record or adapter for intake review decisions.
3. If a schema/migration is needed and too risky for the remaining time, write the concrete migration plan and add tests around the non-persistent adapter instead.
4. Add status doc notes on what is durable now and what remains transient.

Verification:

```powershell
go test ./pkg/documents/... -count=1
go build ./...
npm.cmd --prefix frontend run check
git diff --check
```

Commit if coherent:

```text
feat(asymmflow): prepare durable business memory reviews
```

#### D2: Inventory And Asset Evidence Ledger Preflight

Use this only if D1 is complete, blocked, or clearly unsuitable for the remaining window.

Goal:

Prepare the next roadmap module using what Business Memory Intake just proved.

Allowed outputs:

1. Create `docs\CODEX_GOAL_INVENTORY_ASSET_LEDGER_HANDOFF.md`.
2. Include the same 2-hour roadmap-chain anti-stop contract.
3. Tie Inventory/Asset Evidence to intake source references, product component reuse, and deterministic stock movement authority.
4. Do not implement Inventory unless D1 is committed and more than 30 minutes remain.

Commit if coherent:

```text
docs(asymmflow): add inventory asset ledger handoff
```

## Required Final Verification

Run the broadest practical set before final stop. Minimum if Go and frontend changed:

```powershell
git status --short --branch
go test ./pkg/documents/... -count=1
go test ./internal/viewmodel/documents -count=1
go build ./...
npm.cmd --prefix frontend run check
npm.cmd --prefix frontend run build
git diff --check
git status --short --branch
```

If generated bindings changed:

```powershell
wails generate module
npm.cmd --prefix frontend run check
npm.cmd --prefix frontend run build
```

If schemas changed:

```powershell
powershell -NoProfile -File schemas/generate.ps1 -CheckOnly
```

Optional but valuable if time remains:

```powershell
go test ./... -count=1 -timeout 300s
```

Report exact commands and outcomes. Do not claim a gate passed unless it actually ran.

## Commit Policy

- Make rollback-safe commits after coherent checkpoints.
- Stage only goal-owned files.
- Preserve unrelated dirty work.
- Do not commit generated runtime/build artifacts unless they are required source artifacts such as Wails bindings.
- If a checkpoint cannot be completed cleanly, leave a status note with changed files, failing command, and next repair step.

Expected commit cadence in a healthy run:

1. Kernel and manifest commit.
2. ViewModel/UI commit.
3. Butler/context commit.
4. Durable review or next-handoff commit.

A single final commit is acceptable only if integration requires it, but the run must still continue through the checkpoint chain.

## Stop Conditions

The agent may stop only when one of these is true:

1. The 2-hour window is within 20 minutes of expiry and the current checkpoint is committed or explicitly documented.
2. A concrete blocker prevents further implementation and is recorded with attempted resolution.
3. Required verification is failing and the remaining time is needed to diagnose or document the failure.
4. Checkpoints A, B, C, and D are all complete, verified, committed, and the repo is clean.

Do not stop because:

- The first implementation slice passed.
- A handoff exists.
- A doc was updated.
- The frontend check passed.
- The agent feels the next step might be "too broad."

This repo is a pre-release workspace. Prefer controlled aggression plus rollback-safe commits over nervous under-scoping.

## Exit Criteria

Minimum acceptable final state:

- `docs\modules\business_memory_intake.manifest.json` exists.
- Canonical Business Memory Intake candidate/read-model contract exists in Go.
- Existing Inbox/OCR/classifier behavior is preserved.
- At least one normalization path from existing inbox or OCR/classifier output is tested.
- Review/correction/linking state is display-ready through ViewModel or a documented adapter.
- Inbox or a bounded child component consumes product components if UI is touched.
- Butler/context pack remains inspect/explain/draft/recommend only.
- Verification commands are run and recorded.
- A rollback-safe commit/checkpoint exists.

Preferred final state:

- Checkpoints A through C are implemented and committed.
- D1 has at least a persistence plan or small durable review slice.
- The final repo status is clean.
- `docs\BUSINESS_MEMORY_INTAKE_STATUS.md` records what shipped and the next module queue.

## Run Ledger

Append run notes below this line during execution.

### Run 2026-05-14

- Start time: `2026-05-14T16:12:28.9650613+05:30`.
- Planned 2-hour stop time: `2026-05-14T18:12:28.9650613+05:30`.
- Starting commit: `63b7a36`.
- Starting status: `## master`.
- Active checkpoint: A.
- Subagents spawned and ownership:
  - Worker A: intake kernel, `pkg/documents/intake`, pure tests, and fixture-level normalization advice.
  - Worker B: ViewModel/UI, `internal/viewmodel/documents`, `frontend/src/lib/screens/InboxScreen.svelte`, and optional `frontend/src/lib/components/documents`.
  - Worker C: Butler context/docs/verification, context pack contract, module manifest/status doc, and command evidence.
- Authority stance: Checkpoints A through D are in scope under the green/amber zones; amber edits require a status note rationale.

### Checkpoint A - Canonical Intake Kernel

- Completed changes: added the pure `pkg/documents/intake` candidate contract, source/review/field status enums, normalizers for Runtime inbox process results, stored inbox documents, and OCR/extraction maps, fixture-backed tests, module manifest, and Business Memory Intake status doc.
- Commit hash: `7fb962f`.
- Verification commands run:
  - `Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null` passed.
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./internal/viewmodel/documents -count=1` passed for the concurrent Checkpoint B ViewModel work.
  - `git diff --check` passed with LF/CRLF normalization warnings only.
- Baseline warnings/noise: Git reports LF will be replaced by CRLF for touched text files; no whitespace errors.
- Elapsed time: about 12 minutes at `2026-05-14T16:24:13.8692717+05:30`.
- Next checkpoint selected: B, ViewModel and Inbox review surface.

### Checkpoint B - ViewModel And Inbox Review Surface

- Completed changes: extended `internal/viewmodel/documents` with intake review state, queue metrics, selected candidate rows, extracted-field display rows, provenance rows, and action proposal labels; added focused ViewModel tests; replaced the old Inbox detail card with a bounded `BusinessMemoryReviewPanel` that reuses `KpiStatusStrip`, `EvidenceSourceList`, and `ActionProposalCard`.
- Commit hash: `29ebe21`.
- Verification commands run:
  - `go test ./internal/viewmodel/documents -count=1` passed.
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.
  - `npm.cmd --prefix frontend run build` passed with baseline Svelte warnings.
  - `git diff --check` passed with LF/CRLF normalization warnings only.
- Baseline warnings/noise: 13 existing `svelte-check` warnings remain outside the Inbox/Business Memory files; Vite build repeats a subset of those existing warnings.
- Elapsed time: about 16 minutes at `2026-05-14T16:28:49.3846449+05:30`.
- Next checkpoint selected: C, Butler context pack and agent-safe boundary.

### Checkpoint C - Butler Context Pack And Agent-Safe Boundary

- Completed changes: added pure `ContextPack` and TOON-like formatter helpers in `pkg/documents/intake`, including candidate id, source summary, source kind, business object type, extracted fields/statuses, missing fields, suggested deterministic service targets, review status, warnings, audit refs, and explicit allowed/forbidden agent actions.
- Commit hash: `a57836f`.
- Verification commands run:
  - `go test ./pkg/documents/... -count=1` passed.
  - `go build ./...` passed.
  - `git diff --check` passed.
- Baseline warnings/noise: none in the C gate.
- Elapsed time: about 19 minutes at `2026-05-14T16:31:49.4397749+05:30`.
- Next checkpoint selected: D1, durable review queue preparation.

### Checkpoint D1 - Durable Review Queue Preparation

- Completed changes: added `ReviewRecord`, review decisions, decision-to-review-status mapping, deterministic correlation/idempotency behavior, and a non-persistent `ReviewQueue` adapter with tests. No schema migration was added; the storage boundary is documented for a future deterministic document review service.
- Commit hash: `cb4421e`.
- Verification commands run:
  - `go test ./pkg/documents/... -count=1` passed.
  - `go build ./...` passed.
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.
  - `git diff --check` passed.
- Baseline warnings/noise: 13 existing frontend warnings remain outside this checkpoint.
- Elapsed time: about 23 minutes at `2026-05-14T16:35:07.0328388+05:30`.
- Next checkpoint selected: final broad verification and completion audit, because A through D are now implemented and the repo should be cleaned through a final checkpoint.

### Final Stop

- Final HEAD before final ledger commit: `cb4421e`.
- Final status before final ledger commit: `## master`.
- Commands run across the chain:
  - `Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null`
  - `go test ./pkg/documents/... -count=1`
  - `go test ./internal/viewmodel/documents -count=1`
  - `go build ./...`
  - `npm.cmd --prefix frontend run check`
  - `npm.cmd --prefix frontend run build`
  - `git diff --check`
  - `git diff --cached --check`
  - `git status --short --branch`
- Residual risks:
  - Frontend proposal review choices remain transient until a deterministic review service and database adapter are added.
  - The review queue currently has a pure in-memory adapter and durable record contract, not a database migration.
  - `npm.cmd --prefix frontend run check` continues to report 13 baseline warnings outside the touched Inbox/business-memory files.
- Next recommended `/goal`: add the deterministic Business Memory Intake review service and storage adapter, then expose canonical intake candidates through a Wails/ViewModel read endpoint.
