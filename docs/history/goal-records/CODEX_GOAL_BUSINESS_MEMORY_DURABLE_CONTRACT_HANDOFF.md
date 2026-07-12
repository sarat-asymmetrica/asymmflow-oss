# CODEX_GOAL_BUSINESS_MEMORY_DURABLE_CONTRACT_HANDOFF

Status: Ready for Codex
Created: 2026-05-14
Repo: `the AsymmFlow repository`
Starting expectation: clean `master` at or after `8045a7d docs(asymmflow): finalize business memory intake ledger`

## Tiny Goal Prompt

The human prompt should be this small:

```text
/goal In the AsymmFlow repository, read docs\CODEX_GOAL_BUSINESS_MEMORY_DURABLE_CONTRACT_HANDOFF.md and execute the roadmap-chain completion ladder exactly as specified. Do not use elapsed time as completion. Continue until the mandatory 3x ladder and promoted continuation checkpoint are complete, verified, committed, or blocked with concrete evidence.
```

Everything else needed for the run is inside this file.

## Non-Negotiable Goal Contract

This is a roadmap-chain completion-ladder run, not a time-boxed run and not a single-ticket run.

The previous Business Memory Intake run proved that process adherence is not the problem; scope estimation is. The obvious next work was seven items:

1. Cap'n Proto contract upgrade.
2. Generated schema refresh.
3. Go adapters between intake structs and generated schemas.
4. Durable review storage.
5. Deterministic document review service.
6. Wails/ViewModel/UI wiring.
7. Audit/export/replay/docs verification.

Do not treat those seven items as the whole run. They are the baseline. The mandatory scope is the baseline multiplied by 3:

```text
baseline 7 items -> mandatory 21-checkpoint ladder -> promoted next roadmap set if completed
```

After every coherent milestone:

1. Run the checkpoint's verification gate.
2. Commit or explicitly checkpoint the rollback boundary.
3. Continue to the next checkpoint immediately.

If the mandatory ladder completes quickly, do not mark the goal complete. Promote the next roadmap set in this order:

1. Business Memory source asset registry.
2. Inventory + Asset Evidence Ledger preflight and first implementation checkpoint.
3. Agent-safe module surface hardening for Business Memory and Cashflow Evidence.

The goal may stop only when:

- The mandatory 21-checkpoint ladder is complete, verified, committed, and at least one promoted continuation checkpoint is complete.
- A concrete blocker prevents further implementation and is recorded with attempted resolutions.
- A required verification gate fails and needs human review after diagnosis.
- The worktree cannot be made safe without risking unrelated user work.

Do not stop because:

- The original seven items are done.
- A handoff or doc exists.
- A build passed.
- One worker finished.
- The next checkpoint seems broad.
- The repo is pre-release and rollback-safe commits are available.

## Mission

Turn Business Memory Intake from a JSON-facing review proposal into a durable, typed, service-backed module.

The target loop is:

```text
unstructured business source
-> canonical intake candidate
-> Cap'n Proto durable contract
-> review decision storage
-> deterministic review service
-> Wails/ViewModel/UI review commands
-> Butler/agent context pack
-> audit/export/replay evidence
-> next module queue
```

Business Memory must remain safe: agents inspect, explain, draft, recommend, and assemble context. Deterministic services own review decisions, persistence, linking requests, permissions, audit, and future authoritative mutations.

## Current Findings To Preserve

The repo already has a real Cap'n Proto layer:

- `schemas/documents.capnp`
- `schemas/go/documents/documents.capnp.go`
- `frontend/src/lib/types/schemas/index.ts`
- `pkg/adapter/documents/convert.go`

The new Business Memory Intake slice currently lives mostly in Go structs with JSON tags:

- `pkg/documents/intake.Candidate`
- `pkg/documents/intake.ContextPack`
- `pkg/documents/intake.ReviewRecord`

The manifest currently says `schemas/documents.capnp` is a future durable source. This goal should make it current.

Correct boundary after this run:

- Cap'n Proto: durable/cross-module Business Memory contracts, sync-ready records, generated schema surface, adapter tests.
- Go structs: pure kernel and service/domain authority.
- JSON: manifests, Wails/browser payloads, fixtures, third-party interop.
- TOON: Butler/Codex/agent context packs and compact evidence briefings.

## Product Bar

Use the "$1000/mo justification paradigm."

Business Memory earns value by eliminating document hunting, retyping, evidence reconstruction, context loss between operators/accountants, duplicate tool usage, and failed follow-ups caused by fragmented WhatsApp/email/PDF/Excel/folder truth.

Apply the four filters at every checkpoint:

- ROI proof: name the labor, risk, leakage, or uncertainty reduced by the checkpoint.
- Workflow closure: end with durable contract, persisted decision, service command, UI review action, export, audit, or next-module queue.
- Engine leverage: reuse OCR, classifier, Inbox, Cap'n Proto schema, ViewModel, TOON, Butler, Cashflow Evidence, and future Inventory seams.
- Operator trust: expose source provenance, confidence, status, correction/review choice, deterministic service target, audit refs, export, replay, and rollback.

## Required Read Order

Read all of these before editing:

1. `C:\Projects\ASYMMETRICA_ECOSYSTEM_LOG.md`
2. `C:\Projects\AGENTS.md`
3. `C:\Projects\AGENTIC_SWARM_PROTOCOL.md`
4. `C:\Projects\ASYMMETRICA_PRODUCT_UI_AND_ARCHITECTURE_STANDARD.md`
5. `docs\CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md`
6. `docs\MODULE_CONTRACT_FOUNDATION.md`
7. `docs\BUSINESS_MEMORY_INTAKE_STATUS.md`
8. `docs\modules\business_memory_intake.manifest.json`
9. `schemas\documents.capnp`
10. `schemas\generate.ps1`
11. `pkg\documents\intake\model.go`
12. `pkg\documents\intake\normalize.go`
13. `pkg\documents\intake\context_pack.go`
14. `pkg\documents\intake\review_queue.go`
15. `pkg\adapter\documents\convert.go`
16. `pkg\documents\ports.go`
17. `pkg\documents\domain.go`
18. `runtime_handlers.go`
19. `service_documents.go`
20. `app_setup_documents_surface.go`
21. `internal\viewmodel\documents\documents_vm.go`
22. `frontend\src\lib\screens\InboxScreen.svelte`
23. `frontend\src\lib\components\documents\BusinessMemoryReviewPanel.svelte`
24. `pkg\cashflow\evidence`

Treat older umbrella roadmaps as evidence, not canon.

## Architecture Rules

Preserve this authority model:

```text
pure intake kernel
-> Cap'n Proto schema/adapters
-> deterministic review service
-> storage adapter
-> ViewModel
-> UI
-> agent-safe TOON context
```

Rules:

- Backend/domain services own authority, invariants, permissions, persistence, events, idempotency, and audit.
- ViewModels own display-ready state, commands, validation labels, async status, and correction flows.
- Svelte renders state and emits user intent.
- Agents and Butler may inspect, explain, draft, recommend, and assemble context.
- Agents and Butler may not approve, link, post, delete, or create authoritative business records.
- Do not replace the existing Inbox/OCR/classifier stack wholesale.
- Normalize existing outputs before inventing new parsers.
- Do not add cloud dependencies.
- Do not add dependencies unless current stable version, license, maintenance, and runtime impact are checked from authoritative sources and documented.
- If schemas change, regenerate and verify generated Go/TS surfaces.
- If Wails-exposed Go types or methods change, regenerate Wails bindings.
- Preserve baseline Svelte warnings unless the touched files introduce new errors.

## Authority Zones

### Green Zone: Pre-Approved

The orchestrator and workers may edit these freely if the changes serve this goal:

- `docs\CODEX_GOAL_BUSINESS_MEMORY_DURABLE_CONTRACT_HANDOFF.md`
- `docs\BUSINESS_MEMORY_INTAKE_STATUS.md`
- `docs\modules\business_memory_intake.manifest.json`
- `schemas\documents.capnp`
- `schemas\go\documents\documents.capnp.go`
- `frontend\src\lib\types\schemas\index.ts`
- `pkg\documents\intake\**`
- `pkg\adapter\documents\**`
- `internal\viewmodel\documents\**`
- `frontend\src\lib\screens\InboxScreen.svelte`
- `frontend\src\lib\components\documents\**`
- tests and fixtures directly tied to the above

### Amber Zone: Allowed With Rationale

The agent may edit these if implementation needs it. Record why in the run ledger and keep the diff tight:

- `pkg\documents\domain.go`
- `pkg\documents\ports.go`
- `service_documents.go`
- `runtime_handlers.go`
- `app_setup_documents_surface.go`
- `database.go`
- root app setup/service files that already own document Wails methods
- `frontend\wailsjs\**` generated bindings after `wails generate module`
- `pkg\infra\events\**`
- `pkg\cashflow\evidence\**` only for promoted continuation wiring

### Red Zone: Do Not Touch In This Run

- Live WhatsApp API integration.
- New cloud OCR/LLM dependencies.
- Autonomous accounting, CRM, inventory, or procurement mutations.
- Broad redesign of Inbox, Accounting, Finance, or Butler.
- Cross-repo writes outside `the AsymmFlow repository`.
- Real private client data fixtures.

## Subagent Requirement

The orchestrator must use subagents unless the Codex environment makes that impossible. If subagents are unavailable, record that and still execute the full ladder locally.

Tell every worker:

- You are not alone in the codebase.
- Do not revert edits by others.
- Keep ownership disjoint.
- Report changed paths, verification, risks, and suggested next work.

Minimum wave ownership:

| Worker | Ownership | Expected output |
| --- | --- | --- |
| Worker A: schema and adapters | `schemas/documents.capnp`, generated schema files, `pkg/adapter/documents` | Business Memory Cap'n Proto structs/enums and conversion tests. |
| Worker B: durable service/storage | `pkg/documents/intake`, document service/storage seams, tests | Persistent review records, deterministic service, idempotency and permission boundaries. |
| Worker C: ViewModel/UI/Wails | `internal/viewmodel/documents`, Inbox UI, Wails bindings | Real review commands and read surfaces backed by service state. |
| Worker D: verification/docs | manifest/status/spec ledger, command gates | Verification matrix, docs, baseline warning audit, final status. |
| Orchestrator | integration, amber-zone decisions, commits | Vet diffs, run gates, commit milestones, promote next roadmap set. |

## Mandatory 21-Checkpoint Ladder

Complete these in order. Commit after each coherent milestone or checkpoint group when verification passes.

### Milestone 1: Contract Audit And Schema Design

Checkpoint 1. Record run ledger start: start time, starting commit, `git status --short --branch`, optional safety/check-in cap, subagents/ownership, and baseline warnings.

Checkpoint 2. Audit current JSON/Cap'n Proto/TOON boundary in `docs\BUSINESS_MEMORY_INTAKE_STATUS.md` and `docs\modules\business_memory_intake.manifest.json`.

Checkpoint 3. Design additive Cap'n Proto enums/structs in `schemas\documents.capnp` for:

- `BusinessMemorySourceKind`
- `BusinessMemoryReviewStatus`
- `BusinessMemoryFieldStatus`
- `BusinessMemoryReviewDecision`
- `BusinessMemorySourceRef`
- `BusinessMemoryClassification`
- `BusinessMemoryExtractedField`
- `BusinessMemorySuggestedLink`
- `BusinessMemoryAuditRef`
- `BusinessMemoryCandidate`
- `BusinessMemoryContextPack`
- `BusinessMemoryReviewRecord`
- list wrappers or batch/result structs if needed by generation patterns

Verification:

```powershell
powershell -NoProfile -File schemas\generate.ps1 -CheckOnly
git diff --check
```

Commit target:

```text
feat(asymmflow): add business memory schema contracts
```

### Milestone 2: Generated Schemas And Adapter Bridge

Checkpoint 4. Regenerate Go and TypeScript schema surfaces using the repo script.

Checkpoint 5. Add adapter functions between `pkg/documents/intake` and generated Cap'n Proto types. Preserve pure-kernel authority; adapters should not own review decisions.

Checkpoint 6. Add fixture-backed tests proving round-trip conversion for candidate, context pack, and review record.

Verification:

```powershell
powershell -NoProfile -File schemas\generate.ps1
go test ./pkg/adapter/documents ./pkg/documents/... -count=1
go build ./...
git diff --check
```

Commit target:

```text
feat(asymmflow): bridge business memory intake to capnp
```

### Milestone 3: Durable Storage Preparation

Checkpoint 7. Inspect existing persistence patterns for document/inbox/review records and choose the least-invasive storage adapter pattern.

Checkpoint 8. Add durable review record persistence or a migration-backed repository. It must include idempotency by candidate, decision, deterministic service, and correlation ID.

Checkpoint 9. Add tests for save/list/get/idempotency and invalid decision behavior. If the repo lacks a clean migration seam, implement a tested repository interface plus in-memory and GORM-ready adapter plan, but do not stop there; continue to service wiring.

Verification:

```powershell
go test ./pkg/documents/... -count=1
go build ./...
git diff --check
```

Commit target:

```text
feat(asymmflow): add durable business memory review storage
```

### Milestone 4: Deterministic Review Service

Checkpoint 10. Add a deterministic document review service boundary for Business Memory review decisions.

Checkpoint 11. Enforce allowed decisions, idempotency, actor/correlation requirements, source references, and forbidden agent mutations.

Checkpoint 12. Add read/query methods for review queue state and context pack generation. The service may assemble drafts and recommendations, but must not create authoritative accounting/CRM/inventory records.

Verification:

```powershell
go test ./pkg/documents/... -count=1
go build ./...
git diff --check
```

Commit target:

```text
feat(asymmflow): add business memory review service
```

### Milestone 5: Runtime, Wails, And ViewModel Integration

Checkpoint 13. Wire Wails/runtime methods or document service methods for listing intake candidates, recording review decisions, and generating context packs. Use amber-zone edits only with ledger rationale.

Checkpoint 14. Extend ViewModels so the UI shows persisted review state, command availability, last review action, actor/reason, and deterministic service target.

Checkpoint 15. Regenerate Wails bindings if exposed methods or payloads changed.

Verification:

```powershell
go test ./internal/viewmodel/documents -count=1
go test ./pkg/documents/... -count=1
go build ./...
wails generate module
npm.cmd --prefix frontend run check
git diff --check
```

Commit target:

```text
feat(asymmflow): expose business memory review workflow
```

### Milestone 6: Inbox UI Review Commands

Checkpoint 16. Wire the Inbox Business Memory review panel to real backend/ViewModel state instead of transient proposal-only state where current seams allow it.

Checkpoint 17. Review commands must remain operator-confirmed: accept proposal, needs input, correct field, reject candidate, archive. Labels may be present before full mutation if a backend command is not yet safely exposed, but the limitation must be visible in docs and tests.

Checkpoint 18. Preserve product component usage: `KpiStatusStrip`, `EvidenceSourceList`, `ActionProposalCard`. Do not redesign the whole Inbox.

Verification:

```powershell
npm.cmd --prefix frontend run check
npm.cmd --prefix frontend run build
go test ./internal/viewmodel/documents -count=1
git diff --check
```

Commit target:

```text
feat(asymmflow): back inbox business memory review actions
```

### Milestone 7: Audit, Export, Replay, And Documentation

Checkpoint 19. Add export/replay support for Business Memory candidates, context packs, and review records:

- JSON export for operator/support bundle and browser interop.
- TOON export for Butler/agent context.
- Cap'n Proto adapter test evidence for durable contract readiness.

Checkpoint 20. Update manifest/status docs with the final boundary:

- Cap'n Proto current contract.
- JSON allowed surfaces.
- TOON context surfaces.
- durable storage/service status.
- verification commands and baseline warnings.
- residual risks.

Checkpoint 21. Run final broad gates and leave a final ledger with commits, commands, changed paths, baseline warnings, residual risks, and promoted next checkpoint.

Verification:

```powershell
Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null
powershell -NoProfile -File schemas\generate.ps1 -CheckOnly
go test ./pkg/documents/... -count=1
go test ./pkg/adapter/documents -count=1
go test ./internal/viewmodel/documents -count=1
go build ./...
npm.cmd --prefix frontend run check
npm.cmd --prefix frontend run build
git diff --check
git status --short --branch
```

Optional if the repo is stable enough:

```powershell
go test ./... -count=1 -timeout 300s
```

Commit target:

```text
docs(asymmflow): finalize business memory durable contract ledger
```

## Promoted Continuation Set

If all 21 checkpoints complete and verification is clean enough to continue, the orchestrator must promote the next set before marking complete.

### Promotion A: Business Memory Source Asset Registry

Goal:

Create the first registry layer for sources that Business Memory can track across documents, folders, inbox records, emails, screenshots, and future imports.

Required first checkpoint:

- Add a source asset registry model or spec-backed pure kernel.
- Track source ID, kind, path/label, hash when available, import batch, privacy class, processing status, candidate IDs, and audit refs.
- Add tests for stable IDs and duplicate detection.
- Update Business Memory docs.
- Commit.

Suggested commit:

```text
feat(asymmflow): add business memory source asset registry
```

### Promotion B: Inventory + Asset Evidence Ledger Preflight

Only start this if Promotion A has at least one real tested checkpoint.

Required first checkpoint:

- Create or update `docs\CODEX_GOAL_INVENTORY_ASSET_LEDGER_HANDOFF.md`.
- Include a 3x completion ladder, not a time box.
- Tie Inventory/Asset Evidence to Business Memory source refs, Cap'n Proto contracts, deterministic stock movement authority, product components, and operator trust.
- If implementation is safe, add the first pure stock/evidence kernel checkpoint and tests.

### Promotion C: Agent-Safe Module Surface Hardening

Only start this if Promotion B has a committed checkpoint or is blocked with evidence.

Required first checkpoint:

- Inventory the current agent-safe APIs for Business Memory and Cashflow Evidence.
- Add tests or docs proving inspect/explain/draft/recommend cannot mutate authority.
- Commit a small hardening slice or status doc.

## Run Ledger Requirements

At the start, append a run section to this file:

- Start time.
- Optional safety/check-in cap if the orchestrator chooses one.
- Starting commit.
- Starting `git status --short --branch`.
- Active checkpoint.
- Subagents spawned and ownership.
- Baseline warnings known before work.

After each checkpoint or milestone, append:

- Completed changes.
- Commit hash if committed.
- Verification commands run.
- Baseline warnings/noise.
- Residual risks.
- Next checkpoint selected.

At final stop, append:

- Final HEAD.
- Final status.
- Commands run.
- Changed paths.
- Residual risks.
- Promoted continuation status.
- Next recommended `/goal`.

## Commit Policy

- Commit after coherent milestones so rollback is cheap.
- Stage only goal-owned files.
- Preserve unrelated dirty work.
- Do not commit generated runtime/build artifacts unless they are required source artifacts such as generated schema files or Wails bindings.
- If a checkpoint cannot be completed cleanly, leave a status note with changed files, failing command, and next repair step.

Healthy commit cadence:

1. Schema contracts.
2. Generated schemas/adapters.
3. Durable storage.
4. Review service.
5. Runtime/ViewModel/Wails.
6. Inbox UI review actions.
7. Audit/export/replay/docs.
8. Promoted continuation checkpoint.

## Exit Criteria

Minimum acceptable final state:

- Business Memory Cap'n Proto contracts exist and are generated.
- Pure intake structs convert to/from generated contracts with tests.
- Review records are durable or have a tested repository boundary plus concrete migration blocker evidence.
- Deterministic review service exists.
- UI review commands are backed by service/ViewModel state where safe.
- JSON/Cap'n Proto/TOON boundaries are documented and tested.
- Verification commands are run and recorded.
- Rollback-safe commits exist.
- At least one promoted continuation checkpoint is completed or a concrete blocker is recorded.

Preferred final state:

- All 21 checkpoints complete.
- Business Memory source asset registry first checkpoint complete.
- Inventory + Asset Evidence handoff exists with 3x ladder.
- Final repo status is clean.
- `docs\BUSINESS_MEMORY_INTAKE_STATUS.md` is the live status source for the module.

## Run Ledger

Append run notes below this line during execution.

### Run 2026-05-14T16:57:53+05:30 - Durable Business Memory Contract Ladder

- Start time: 2026-05-14T16:57:53.7488159+05:30.
- Safety/check-in cap: none selected; completion is checkpoint-ladder based, not elapsed-time based.
- Starting commit: `58c8ecc`.
- Starting status: `## master` with no short-status entries.
- Active checkpoint: Checkpoint 1, run ledger start and baseline boundary.
- Subagents/ownership: subagents not spawned because the current Codex tool policy only permits spawning when the user explicitly asks for delegated/parallel agent work. The run will execute locally while preserving the handoff ownership map: schema/adapters, durable service/storage, ViewModel/UI/Wails, verification/docs, and orchestration.
- Baseline warnings/noise:
  - `powershell -NoProfile -File schemas\generate.ps1 -CheckOnly` is blocked by the local Windows script execution policy (`PSSecurityException`). Resolution for this run: use `powershell -ExecutionPolicy Bypass -NoProfile -File ...` for the same repo script, which compiled the current schemas successfully.
  - `powershell -ExecutionPolicy Bypass -NoProfile -File schemas\generate.ps1 -CheckOnly` passed before schema edits.
- Product filters for Checkpoint 1:
  - ROI proof: establish a typed contract before more review/storage/UI work so operators do not re-enter or reinterpret business evidence across tools.
  - Workflow closure: begin from a clean rollback boundary with explicit verification and ownership.
  - Engine leverage: existing Cap'n Proto generator, document intake kernel, Inbox/OCR/classifier seams, ViewModel, and Butler-safe TOON packs.
  - Operator trust: baseline status, verification exception, and subagent limitation are visible before implementation.
- Next checkpoint: Checkpoint 2, audit JSON/Cap'n Proto/TOON boundary in the status doc and manifest.

### Milestone 1 - Checkpoints 2 and 3

- Completed changes:
  - Audited and updated the Business Memory JSON/Cap'n Proto/TOON boundary in `docs\BUSINESS_MEMORY_INTAKE_STATUS.md`.
  - Updated `docs\modules\business_memory_intake.manifest.json` so Cap'n Proto is current durable contract territory, with JSON and TOON retained for their intended surfaces.
  - Added additive Business Memory schema contracts to `schemas\documents.capnp`: source/review/field/decision enums, source/classification/field/link/audit refs, candidate, context pack, review record, and batch wrappers.
- Verification commands run:
  - `powershell -ExecutionPolicy Bypass -NoProfile -File schemas\generate.ps1 -CheckOnly` passed.
  - `Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null` passed.
  - `git diff --check` passed with LF-to-CRLF working-copy warnings only.
  - `npx.cmd gitnexus analyze` refreshed a stale index from `aa578b2` to `58c8ecc`; generated `AGENTS.md`/`CLAUDE.md` metric side effects were removed from the checkpoint.
- Baseline warnings/noise:
  - Windows script execution policy blocks the handoff's exact `powershell -NoProfile -File ...` form.
  - Git reports LF-to-CRLF warnings on touched text files; no whitespace errors were reported.
- Residual risks:
  - Generated Go/TypeScript surfaces and adapter bridge are not complete until Milestone 2.
  - Schema is additive and compiled, but round-trip tests do not exist yet for the new Business Memory types.
- Next checkpoint: Checkpoint 4, regenerate Go and TypeScript schema surfaces.

Milestone 1 rollback commit: `c303508 feat(asymmflow): add business memory schema contracts`.

### Milestone 2 - Checkpoints 4, 5, and 6

- Completed changes:
  - Regenerated Go and TypeScript schema surfaces from `schemas\documents.capnp`.
  - Added `pkg\adapter\documents\business_memory.go` as the adapter bridge between `pkg/documents/intake` structs and generated Business Memory Cap'n Proto types.
  - Added `pkg\adapter\documents\business_memory_test.go` with round-trip tests for candidate, context pack, and review record.
  - Updated `docs\BUSINESS_MEMORY_INTAKE_STATUS.md` and `docs\modules\business_memory_intake.manifest.json` with generated surface and adapter test evidence.
- Verification commands run:
  - `powershell -ExecutionPolicy Bypass -NoProfile -File schemas\generate.ps1 -TypeScript` passed.
  - `go test ./pkg/adapter/documents ./pkg/documents/... -count=1` passed.
  - `go build ./...` passed after generation settled.
  - `git diff --check` passed with LF-to-CRLF working-copy warnings only.
- Baseline warnings/noise:
  - LF-to-CRLF warnings on generated schema files remain baseline working-copy noise.
  - The first combined generation/build batch was rerun in a settled order so the final build evidence is clean.
- Residual risks:
  - Durable repository/service wiring is not complete until Milestones 3 and 4.
  - Generated schema refresh rewrote all Go schema packages because `schemas\generate.ps1` regenerates the full schema order.
- Next checkpoint: Checkpoint 7, inspect persistence patterns and select the least-invasive durable review storage adapter.

Milestone 2 rollback commit: `3347dad feat(asymmflow): bridge business memory intake to capnp`.

### Milestone 3 - Checkpoints 7, 8, and 9

- Completed changes:
  - Inspected the existing startup migration pattern in `app.go`; mature databases skip broad `AutoMigrate` after table count exceeds 50, so implicit startup migration is not a safe assumption for this slice.
  - Added `pkg\documents\intake\review_repository.go` with the review repository contract, validation, and an in-memory idempotent adapter.
  - Added `pkg\adapter\documents\business_memory_storage.go` with `BusinessMemoryReviewRecordModel`, explicit `Migrate(ctx)`, save/get/list methods, and idempotency by candidate ID + decision + proposed deterministic service + correlation ID.
  - Added focused storage tests in `pkg\documents\intake\review_repository_test.go` and `pkg\adapter\documents\business_memory_storage_test.go`.
  - Updated Business Memory status and manifest with the durable storage boundary and migration note.
- Verification commands run:
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./pkg/adapter/documents -count=1` passed.
  - `go build ./...` passed.
  - `Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null` passed.
  - `git diff --check` passed with LF-to-CRLF working-copy warnings only.
- Baseline warnings/noise:
  - None from the focused storage tests.
- Residual risks:
  - The GORM adapter is tested and migration-ready, but it is not wired into root app startup migrations because that path intentionally skips migration for established client databases.
  - Deterministic review service command methods are still pending in Milestone 4.
- Next checkpoint: Checkpoint 10, add a deterministic document review service boundary.

Milestone 3 rollback commit: `d528999 feat(asymmflow): add durable business memory review storage`.

### Milestone 4 - Checkpoints 10, 11, and 12

- Completed changes:
  - Added `pkg\documents\intake\review_service.go` as the deterministic Business Memory review service boundary.
  - Enforced supported decisions, operator actor, correlation ID, candidate/source references, and repository idempotency.
  - Rejected agent-originated review mutations while preserving context-pack assembly as a non-mutating operation.
  - Added read/query methods for get/list review records and `BuildQueueState`, which returns the selected candidate, persisted records, last review, and Butler-safe context pack.
  - Added service tests covering operator decision persistence, idempotency, missing actor/correlation/source validation, agent mutation rejection, and queue/context-pack state.
  - Updated Business Memory status and manifest with the deterministic service boundary.
- Verification commands run:
  - `go test ./pkg/documents/... -count=1` passed.
  - `go build ./...` passed.
- Baseline warnings/noise:
  - None from service tests or build.
- Residual risks:
  - Runtime/Wails/ViewModel methods still need to expose service state and commands safely in Milestone 5.
  - Permission enforcement is represented by actor/agent boundary in the pure service; root app/Wails permission guards are still pending for exposed methods.
- Next checkpoint: Checkpoint 13, wire runtime/document service methods for listing candidates, recording review decisions, and generating context packs.

### Milestone 5 - Checkpoints 13, 14, and 15

- Completed changes:
  - Added `business_memory_review_runtime.go` with Wails/App methods for `GetBusinessMemoryReviewQueue`, `RecordBusinessMemoryReviewDecision`, and `GenerateBusinessMemoryContextPack`.
  - Added `DocumentsService` wrappers so the frontend can call the document-domain binding instead of reaching only through `App`.
  - Wired runtime methods to the durable GORM review repository and deterministic review service, including explicit repository migration, idempotent correlation IDs, candidate lookup from stored inbox rows, and context-pack TOON output.
  - Used existing seeded permissions: `documents:view` for queue/context reads and `documents:classify` for operator review decisions. This is an amber-zone rationale to avoid adding a new unseeded permission that would strand current roles.
  - Extended document ViewModels with persisted last-review state, actor/reason, command availability, decision status overlay, and deterministic service target.
  - Updated Inbox UI and `BusinessMemoryReviewPanel` so review choices call durable Wails methods and display the last persisted review instead of relying only on transient proposal labels.
  - Regenerated `frontend\wailsjs` bindings with `wails generate module` and mechanically removed generated trailing whitespace from `models.ts` so `git diff --check` stays green.
  - Updated Business Memory status and manifest with runtime/Wails/ViewModel wiring.
- Verification commands run:
  - `go test ./internal/viewmodel/documents -count=1` passed.
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./pkg/adapter/documents -count=1` passed.
  - `go test . -run "TestParseBusinessMemoryReviewDecision|TestInboxDocumentToBusinessMemoryCandidate" -count=1` passed.
  - `go build ./...` passed.
  - `wails generate module` passed.
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.
  - `git diff --check` passed with LF-to-CRLF working-copy warnings only.
- Baseline warnings/noise:
  - `wails generate module` still emits the existing `Not found: struct { R1 float64 ... }` generator warning.
  - `npm.cmd --prefix frontend run check` still reports the known 13 Svelte warnings outside this slice.
  - LF-to-CRLF warnings remain working-copy normalization noise.
- Residual risks:
  - Context pack generation is exposed and returned but not yet surfaced as a copied/exported artifact in the UI.
  - Review decisions persist but do not yet publish events or write export/replay evidence; those remain in Milestones 6 and 7.
- Next checkpoint: Checkpoint 16, complete the Inbox UI review command surface and operator feedback loop.

### Milestone 6 - Checkpoints 16, 17, and 18

- Completed changes:
  - Tightened the Inbox Business Memory panel so all five durable operator decisions are reachable: accept proposal, needs input, correct field, reject candidate, and archive review.
  - Extended the ViewModel `ReviewCommands` list with `correct_field` and `archive` actions and added a focused assertion that they remain enabled.
  - Preserved the existing product components and layout: `KpiStatusStrip`, `EvidenceSourceList`, and `ActionProposalCard` remain the primary review composition.
  - Kept document archival separate from durable review archival by labeling the existing inbox status action as `Archive document`.
  - Updated Business Memory status and manifest with the operator command set.
- Verification commands run:
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.
  - `npm.cmd --prefix frontend run build` passed with baseline Svelte warnings.
  - `go test ./internal/viewmodel/documents -count=1` passed.
- Baseline warnings/noise:
  - The frontend check/build warning set remains the known Svelte baseline outside this slice.
- Residual risks:
  - Review commands are immediate operator button actions; a richer confirmation modal/correction editor can be promoted later if needed.
  - Export/replay evidence and final ledger are still pending in Milestone 7.
- Next checkpoint: Checkpoint 19, add export/replay support for Business Memory candidates, context packs, and review records.

### Milestone 7 - Checkpoints 19, 20, and 21

- Completed changes:
  - Added `pkg\documents\intake\export.go` with `ReviewExportBundle`, JSON export, JSON replay validation, and TOON export.
  - Added export tests for JSON replay, TOON agent-boundary evidence, and schema mismatch rejection.
  - Added `ExportBusinessMemoryReviewBundle` on `App` and `DocumentsService`, returning the typed bundle plus JSON and TOON strings for a selected inbox candidate.
  - Regenerated `frontend\wailsjs` bindings for the export payloads and cleaned generated `models.ts` trailing whitespace.
  - Updated Business Memory status and manifest with the final JSON/TOON/Cap'n Proto boundary and export/replay verification evidence.
- Verification commands run:
  - `Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null` passed.
  - `powershell -ExecutionPolicy Bypass -NoProfile -File schemas\generate.ps1 -CheckOnly` passed. The exact `powershell -NoProfile -File` form remains blocked by local execution policy, so the verified gate uses `-ExecutionPolicy Bypass`.
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./pkg/adapter/documents -count=1` passed.
  - `go test ./internal/viewmodel/documents -count=1` passed.
  - `go build ./...` passed.
  - `wails generate module` passed.
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.
  - `npm.cmd --prefix frontend run build` passed with baseline Svelte warnings.
  - `git diff --check` passed with LF-to-CRLF working-copy warnings only.
  - Optional `go test ./... -count=1 -timeout 300s` passed.
- Baseline warnings/noise:
  - `wails generate module` still emits the existing `Not found: struct { R1 float64 ... }` warning.
  - Frontend check/build still report the known Svelte warning set outside the Business Memory files.
  - LF-to-CRLF warnings remain working-copy normalization noise.
- Residual risks:
  - The Wails export method returns JSON/TOON strings but the UI does not yet provide copy/download buttons.
  - Event publishing for `documents.intake.*` remains manifest-level future work.
  - No known test failure remains after the optional broad Go suite; remaining risks are product-surface/export ergonomics rather than failing verification.
- Completion ledger before promoted continuation:
  - Milestone 1 commit: `c303508 feat(asymmflow): add business memory schema contracts`.
  - Milestone 2 commit: `3347dad feat(asymmflow): bridge business memory intake to capnp`.
  - Milestone 3 commit: `d528999 feat(asymmflow): add durable business memory review storage`.
  - Milestone 4 commit: `c399d63 feat(asymmflow): add business memory review service`.
  - Milestone 5 commit: `c6c26fb feat(asymmflow): expose business memory review workflow`.
  - Milestone 6 commit: `202ced0 feat(asymmflow): back inbox business memory review actions`.
- Milestone 7 commit: `9baf305 docs(asymmflow): finalize business memory durable contract ledger`.

### Promotion A.1 - Business Memory Source Asset Registry

- Completed changes:
  - Added `pkg\documents\intake\source_registry.go` with a pure source asset registry for document, folder, inbox, email, screenshot, and future import sources.
  - Added stable source IDs that prefer source kind plus content hash when available, then path, then label.
  - Added duplicate merge behavior for candidate IDs, audit refs, import batch, privacy/status metadata, and seen-time range.
  - Added tests for hash-backed stable IDs, duplicate detection/merge behavior, and default privacy/status/source-kind inference.
  - Updated Business Memory status and manifest with the registry contract and first promoted checkpoint evidence.
- Verification commands run:
  - `go test ./pkg/documents/... -count=1` passed.
  - `Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null` passed.
  - `git diff --check` passed with LF-to-CRLF working-copy warnings only.
- Residual risks:
  - Registry is a pure in-memory/spec kernel; durable DB/sync wiring is a future checkpoint.
- Completion note:
  - This satisfies the promoted continuation requirement's first tested checkpoint.

Promoted continuation rollback commit: `6d8a69d feat(asymmflow): add business memory source asset registry`.

### Final Stop

- Final HEAD before follow-on audit docs: `6d8a69d`.
- Final status before follow-on audit docs: clean `master`.
- Completion state: original Business Memory Intake checkpoints A-D, the durable contract 21-checkpoint ladder, and promoted continuation A.1 are implemented, verified, and committed.
- Next recommended `/goal`: run `docs\CODEX_GOAL_BUSINESS_MEMORY_SOURCE_REGISTRY_DURABILITY_HANDOFF.md` to make the source asset registry durable, reviewable, exportable, and visible in the Inbox review surface.
