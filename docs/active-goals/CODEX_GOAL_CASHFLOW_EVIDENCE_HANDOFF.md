# CODEX_GOAL_CASHFLOW_EVIDENCE_HANDOFF

Status: Draft v0.1
Created: 2026-05-14
Scope: `the AsymmFlow repository`

## Goal Command

Run the AsymmFlow roadmap chain for a 2-hour window. Complete the Cashflow + Evidence Command Center proof module through a first implementation checkpoint: manifest/read-model contract, deterministic backend read service or service skeleton, ViewModel contract, focused kernel/read-model tests where code is touched, docs updates, verification, and a rollback-safe commit.

## Current Context

Current canon:

1. `C:\Projects\ASYMMETRICA_ECOSYSTEM_LOG.md`
2. `C:\Projects\AGENTIC_SWARM_PROTOCOL.md`
3. `C:\Projects\ASYMMETRICA_PRODUCT_UI_AND_ARCHITECTURE_STANDARD.md`
4. `docs/CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md`
5. `docs\MODULE_CONTRACT_FOUNDATION.md`
6. `docs\CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT.md`

Historical evidence:

- `docs/V0_1_RELEASE_ROADMAP_2026_05_08.md`
- `docs/WAVE17_PROGRESS.md`

Cashflow Evidence should compose existing deterministic Finance, Documents, Banking, Butler, Compliance, and event/sync seams. It should not become a second accounting authority.

## Product Bar

### $1000/mo Justification

This module targets owner/operator pain that can plausibly replace manual collection spreadsheets, accountant back-and-forth, overdue-payment chasing, document hunting, support-bundle assembly, and fragile cash visibility.

### ROI Proof

- Reduces delayed collection and missed follow-up.
- Reduces manual reconciliation/evidence preparation.
- Reduces accountant/operator time spent proving which invoices/payments/docs are trustworthy.
- Reduces support/debug time by exporting a ready evidence pack.

### Workflow Closure

The user should end with at least one concrete outcome:

- follow-up queue item
- draft journal/posting readiness action
- missing evidence request
- support/evidence pack export
- manager cashflow brief
- approval decision for a deterministic service command

### Engine Leverage

- `pkg/finance/posting` for posting coverage and trial-balance gates.
- `finance_reporting_service.go` for current AR aging/cashflow evidence, to be extracted or wrapped.
- `pkg/finance/banking` for reconciliation status and audit trail.
- `invoice_traceability.go` for source chain.
- `pkg/documents` and OCR/classification seams for source evidence.
- `pkg/butler/chat/toon.go` and grounded fastpath surfaces for explain/draft agent adapters.
- `pkg/infra/events` and `pkg/sync` for future event/subscription and support-bundle contracts.

### Operator Trust

The first screen/read model must expose:

- source provenance
- canonical status by source type
- missing evidence
- posting coverage
- trial-balance status
- reconciliation status
- correction/approval next action
- export/support-bundle path
- audit trail references

## Architecture Constraints

- Backend/domain Go services remain authoritative for persistence, posting, matching, approval, filing, export, and audit.
- Cashflow Evidence is a composition/read-model module first.
- Butler may inspect, explain, summarize, draft, and recommend. It may not post, persist, approve, or reverse.
- UI follows MVVM: backend/read model -> ViewModel -> Svelte surface.
- Use JSON only for browser/read-model payloads and planning manifests. Use Cap'n Proto when records become durable sync/engine contracts. Use TOON for agent/evidence packs.
- Do not mutate ignored local DB/build/runtime state except as required by verification.

## Initial Module Shape

```text
cash exposure kernel
missing evidence kernel
posting readiness kernel
    -> cashflow evidence read service
    -> storage/source adapters over existing finance/document/banking tables
    -> CashflowEvidenceVM
    -> command center UI
    -> Butler TOON evidence/explanation adapter
```

## Suggested First Implementation Slice

Keep the first slice narrow but real:

1. Add a Cashflow Evidence module manifest using `docs/templates/module_manifest.example.json` as the template.
2. Define Go read-model structs for the command center summary in an appropriate package, likely `pkg/finance/cashflow` or a new `pkg/cashflow`.
3. Extract pure scoring helpers:
   - cash exposure bucket
   - missing evidence severity
   - posting readiness status
   - follow-up priority
4. Add a backend read service that composes existing sources without creating new accounting state.
5. Add focused tests for pure helpers and read-model mapping.
6. Add `internal/viewmodel` contract for display-ready command center state.
7. Add a minimal Svelte command center only if the backend/read-model slice is stable inside the time box.
8. Add TOON evidence pack/explanation shape for Butler as draft/inspect only.

## Implementation Checkpoint

Completed in the first advance slice:

- Added `pkg/cashflow/evidence` with deterministic command-center model building.
- Added pure helpers for cash exposure, missing evidence status, posting readiness, overall status, and next deterministic action.
- Added a read-only `SnapshotReader`/`Service` seam so future storage adapters can feed existing deterministic finance/document/banking facts without creating duplicate accounting authority.
- Added `internal/viewmodel/cashflow` with `CommandCenterVM`, evidence rows, posting readiness display, summary cards, and inspect/draft/export action contracts.
- Added `docs/modules/cashflow_evidence.manifest.json` as the module manifest for the current proof slice.
- Added a read-only App/FinanceService adapter that loads AR exposure, posting coverage, trial-balance readiness, and bank-match evidence into the command-center model.
- Added an agent-safe TOON evidence brief with `mutates_state=false`, deterministic command hints, and explicit forbidden operations.
- Added a compact Svelte command-center surface in `AccountingScreen.svelte` that loads the read-only Wails endpoint, exposes source readiness, posting gaps, bank-match gaps, evidence-pack item count, and non-mutating inspect/draft/export affordances.
- Added deterministic JSON/TOON evidence-pack builders and `ExportCashflowEvidencePack(days)` on App/FinanceService. The UI export action now writes a JSON evidence pack through the existing report export directory while preserving deterministic services as the only authority.
- Added `GetCashflowEvidenceAgentBrief(days, maxChars)` on App/ButlerService so root Butler surfaces can retrieve the TOON brief under the same `finance:view` gate without gaining mutation authority.
- Added read-only `ActionProposal` generation to the command-center model, agent brief, and evidence pack. Proposals name the deterministic service required for real execution and carry `mutates_state=false`.
- Added invoice traceability as a storage-backed evidence source over existing invoice links (order/RFQ/quote/offer/delivery/customer PO/despatch references), increasing evidence-pack audit items without introducing new accounting state.
- Added open follow-up task counting from existing follow-up storage for receivables/payment-oriented work, making the command center's follow-up count storage-backed instead of placeholder-only.
- Surfaced read-only action proposals in the Accounting command-center panel with their source, reason, priority cue, and required deterministic service.
- Added a persisted proposal review queue for current Cashflow Evidence action proposals. Queue sync records pending reviews without executing finance actions; operator signoff updates review status only.
- Added Accounting signoff controls for queued proposals (`approved`, `needs_input`, `rejected`) while keeping real finance/task execution routed to deterministic services.

Deferred from this checkpoint:

- No DB/GORM adapter yet.
- Added a read-only App/FinanceService adapter in `cashflow_evidence_service.go` and `service_finance.go`.
- Added package-level Butler-ready TOON brief generation; root Butler wiring remains deferred.
- No durable Cap'n Proto schema yet.

## Worker Wave Plan

Use subagents only where they materially increase throughput and keep ownership disjoint.

| Worker | Ownership | Output |
| --- | --- | --- |
| Orchestrator | Final architecture, write boundaries, integration, verification, commit | Consolidated implementation and checkpoint |
| Finance/read-model worker | Cashflow Evidence structs, scoring kernels, read service, tests | Backend/read-model patch and exact verification |
| ViewModel/UI worker | `internal/viewmodel` contract and optional Svelte command center | Presentation state and UI if time allows |
| Butler/evidence worker | TOON evidence pack shape and agent-safety notes/tests | Inspect/explain adapter without write authority |
| Verification worker | Go/frontend/schema gates and docs/status review | Exact commands, pass/fail, baseline noise |

Workers are not alone in the codebase. They must not revert edits by others, must keep to assigned write scopes, and must report conflicts instead of overwriting.

## Verification Gates

Docs-only prep:

- `git status --short`
- `git status --short --ignored`
- `git diff --check`

If Go code is touched:

- focused package tests for new kernels/read model
- `go test ./... -count=1 -timeout 300s` if the slice touches shared service behavior
- `go build ./...`

If frontend code is touched:

- `npm.cmd --prefix frontend run check`
- `npm.cmd --prefix frontend run build`
- targeted browser/Playwright check for the command center surface if practical

If schemas are touched:

- `powershell -NoProfile -File schemas/generate.ps1 -CheckOnly`

## Verification Log

| Command | Result |
| --- | --- |
| `go test ./pkg/cashflow/evidence -count=1` | Initial sandbox run failed on Go build-cache access; rerun with approved Go-test access found one service-window bug; after fix passed; post-adapter rerun passed |
| `go test ./internal/viewmodel/cashflow -count=1` | Initial sandbox run failed on Go build-cache access; rerun with approved Go-test access passed; post-fix and post-adapter reruns passed |
| `Get-Content docs\modules\cashflow_evidence.manifest.json \| ConvertFrom-Json \| Out-Null; Write-Output "JSON OK"` | Passed |
| `go build ./...` | Initial sandbox run failed on Go build-cache access; rerun with approved Go-build access passed; post-adapter rerun passed |
| `go test ./pkg/cashflow/evidence -count=1` after agent brief | Passed |
| `go build ./...` after agent brief | Passed |
| `wails generate module` | Passed with the known generated-struct warning for an unrelated anonymous struct |
| `npm.cmd --prefix frontend run check` after Wails bindings | Passed with 0 errors and 13 baseline warnings |
| `npm.cmd --prefix frontend run build` after Wails bindings | Passed with baseline Svelte warnings |
| `npm.cmd --prefix frontend run check` after command-center UI | Passed with 0 errors and 13 baseline warnings |
| `npm.cmd --prefix frontend run build` after command-center UI | Passed with baseline Svelte warnings |
| `go test ./pkg/cashflow/evidence -count=1` after evidence-pack export | Passed |
| `go build ./...` after evidence-pack export | Passed |
| `npm.cmd --prefix frontend run check` after export binding | Passed with 0 errors and 13 baseline warnings |
| `npm.cmd --prefix frontend run build` after export binding | Passed with baseline Svelte warnings |
| `wails generate module` after Butler agent-brief binding | Passed with the known generated-struct warning for an unrelated anonymous struct |
| `go test ./pkg/cashflow/evidence -count=1` after action proposals | Passed |
| `go test ./internal/viewmodel/cashflow -count=1` after action proposals | Passed |
| `go build ./...` after invoice traceability source adapter | Passed |
| `go build ./...` after follow-up task source adapter | Passed |
| `npm.cmd --prefix frontend run check` after action-proposal UI | Passed with 0 errors and baseline warnings |
| `npm.cmd --prefix frontend run build` after action-proposal UI | Passed with baseline Svelte warnings |
| `go test ./pkg/cashflow/evidence -count=1` after persisted proposal review key | Passed |
| `go test . -run TestNonExistent -count=0` after persisted proposal review endpoints | Passed |
| `wails generate module` after proposal review endpoints | Passed with known anonymous-struct warning |
| `go build ./...` after proposal review endpoints and bindings | Passed |
| `npm.cmd --prefix frontend run check` after proposal review UI | Passed with 0 errors and baseline warnings |
| `npm.cmd --prefix frontend run build` after proposal review UI | Passed with baseline Svelte warnings |
| `npm.cmd --prefix frontend run check` after proposal signoff controls | Passed with 0 errors and baseline warnings |
| `npm.cmd --prefix frontend run build` after proposal signoff controls | Passed with baseline Svelte warnings |
| `go test ./... -count=1 -timeout 300s` final checkpoint audit | Passed |

## Commit Policy

- Start from a clean tracked worktree or explicitly record unrelated dirty files.
- Stage only goal-owned paths.
- Keep ignored runtime/build state untouched unless a verification gate writes it.
- Commit one coherent checkpoint after verification.
- If the slice cannot complete inside the time box, leave a documented checkpoint with changed paths, commands run, risks, and next action.

## Exit Criteria

- Cashflow Evidence manifest/read-model contract exists. Completed in `docs/modules/cashflow_evidence.manifest.json`.
- First read-model/service or service skeleton composes existing deterministic sources without new accounting authority. Completed in `pkg/cashflow/evidence`.
- Pure scoring/status helpers have focused tests if implemented. Completed in `pkg/cashflow/evidence/model_test.go`.
- ViewModel contract exists before any full UI surface. Completed in `internal/viewmodel/cashflow/evidence_vm.go`.
- Butler adapter is inspect/explain/draft/recommend only. Package-level brief completed in `pkg/cashflow/evidence/agent.go`; root Butler wiring is deferred.
- Minimal Svelte command center exists. Completed in `frontend/src/lib/screens/AccountingScreen.svelte`.
- Evidence-pack export exists. Completed with `pkg/cashflow/evidence/export.go`, `App.ExportCashflowEvidencePack`, FinanceService binding, and Accounting export action.
- Butler-facing TOON brief binding exists. Completed with `App.GetCashflowEvidenceAgentBrief`, ButlerService binding, and regenerated Wails methods.
- Read-only action proposals exist. Completed with `pkg/cashflow/evidence/proposal.go` and Wails `ActionProposal` model generation.
- Invoice traceability evidence source exists. Completed in `appCashflowEvidenceReader.invoiceTraceabilityEvidence`.
- Open follow-up task source exists. Completed in `appCashflowEvidenceReader.openCashflowFollowUpTasks`.
- Action proposals are operator-visible. Completed in `frontend/src/lib/screens/AccountingScreen.svelte`.
- Persisted proposal review/signoff queue exists. Completed with `cashflow_evidence_review.go`, FinanceService/App Wails bindings, and Accounting queue sync/status display.
- Operator can mark queued proposals approved, rejected, or needing input from the Accounting panel without triggering deterministic execution.
- Verification commands are run and recorded.
- Roadmap/progress docs are updated if the chain changes.
- A rollback-safe commit/checkpoint exists.

## Residual Risks To Watch

- Avoid building a screen-local aggregator with no backend authority boundary.
- Avoid creating duplicate receivables/accounting state.
- Avoid letting Butler mutate finance/doc state directly.
- Avoid using cash projections without exposing assumptions and missing evidence.
- Avoid broad inventory/sync refactors inside the first Cashflow Evidence slice unless they block the module.
