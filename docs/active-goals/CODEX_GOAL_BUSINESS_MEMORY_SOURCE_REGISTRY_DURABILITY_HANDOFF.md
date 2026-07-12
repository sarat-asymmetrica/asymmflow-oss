# CODEX_GOAL_BUSINESS_MEMORY_SOURCE_REGISTRY_DURABILITY_HANDOFF

Status: Ready for autonomous GPT-5.5 Codex CLI execution
Created: 2026-05-15
Repo: `the AsymmFlow repository`
Mode: implementation + verification + docs + milestone commits

## Goal Command

```text
/goal Run the AsymmFlow Business Memory Source Registry Durability chain.

Do not stop at the first working slice. Complete the mandatory checkpoint ladder below through code, focused tests/builds, docs/status updates, and rollback-safe commits after coherent milestones.

If the Business Memory ladder completes cleanly, promote the first Cashflow Evidence allocation-aware preflight checkpoint in the same run and commit that checkpoint too.
```

## Required First Reads

Read these before editing:

1. `C:\Projects\AGENTS.md`
2. `C:\Projects\ASYMMETRICA_ECOSYSTEM_LOG.md`
3. `C:\Projects\AGENTIC_SWARM_PROTOCOL.md`
4. `C:\Projects\ASYMMETRICA_PRODUCT_UI_AND_ARCHITECTURE_STANDARD.md`
5. `AGENTS.md`
6. `docs/CODEX_REPO_HISTORY_AUDIT_2026_05_15.md`
7. `docs/CODEX_AUDIT_EVIDENCE_INDEX_2026_05_15.md`
8. `docs/CODEX_DEV_ROADMAP_2026_05_15.md`
9. `docs/CODEX_PH_HOLDINGS_SOURCE_TRACK_RECON_2026_05_15.md`
10. `docs/BUSINESS_MEMORY_INTAKE_STATUS.md`
11. `docs/modules/business_memory_intake.manifest.json`
12. `docs/modules/cashflow_evidence.manifest.json`
13. `docs/MODULE_CONTRACT_FOUNDATION.md`

Treat old roadmap/progress docs as evidence, not canon, unless the current code/tests/manifests support them.

## Preamble

The repo already absorbed the rough `ph_holdings` client-feedback track at the file/capability level. The next job is not a blind backport. It is architectural reconciliation: turn the Business Memory source registry into durable, inspectable, replayable provenance while preserving the refactor line's module-contract shape.

Jordan's client-feedback line contributed hard product lessons:

- Offer and costing revision identity is commercial truth.
- Bank matching needs allocation-aware evidence, not one invoice equals one payment thinking.
- Destructive or authoritative actions need approval queues.
- Sync drift, training support, user guides, and UI/backend action inventories are launch-readiness concerns.

This run begins by closing Business Memory source provenance, then promotes one Cashflow allocation-aware preflight checkpoint if time and verification stay green.

## One Principle

Source evidence must survive every transformation.

Business Memory should never produce an attractive candidate that has lost its source identity, privacy class, hash/path/label, processing status, candidate links, or audit trail. Agent and operator surfaces can summarize, but deterministic services must preserve provenance.

## GitNexus Policy

Use `npx gitnexus` only as a discovery and assistance tool:

- OK: `npx gitnexus status` to understand current index state.
- OK: GitNexus query/context/impact-style discovery if available and useful for finding callers, upstream/downstream relationships, or relevant flows.
- OK: use it as a second map while reading actual source files.
- Not required: keeping the index up to date.
- Do not run `npx gitnexus analyze` merely because code changed.
- Do not block commits because the GitNexus index is stale after your own edits.
- Do not treat GitNexus output as proof. Confirm against source, tests, manifests, and build output.

The authoritative verification gates for this run are Go tests, schema checks when relevant, frontend check when UI/bindings change, manifest parsing, build output, git diff review, and runtime code inspection.

## Authority Zones

### Green

You may edit these directly:

- `pkg/documents/intake/**`
- `pkg/adapter/documents/**`
- `internal/viewmodel/documents/**`
- `business_memory_review_runtime.go`
- `service_documents.go`
- `frontend/src/lib/components/documents/BusinessMemoryReviewPanel.svelte`
- `frontend/src/lib/screens/InboxScreen.svelte`
- `docs/BUSINESS_MEMORY_INTAKE_STATUS.md`
- `docs/modules/business_memory_intake.manifest.json`
- `docs/CODEX_GOAL_BUSINESS_MEMORY_SOURCE_REGISTRY_DURABILITY_HANDOFF.md`
- New focused Business Memory tests and fixtures under the same module boundaries

### Amber

Allowed with a short rationale in the commit message or status doc:

- `schemas/documents.capnp`
- `schemas/go/documents/documents.capnp.go`
- `frontend/src/lib/types/schemas/index.ts`
- `frontend/wailsjs/**`
- `pkg/infra/events/**`
- `pkg/sync/**`
- `database.go`
- root service or app files needed only to wire the Business Memory runtime boundary
- `docs/CODEX_DEV_ROADMAP_2026_05_15.md`

If touching generated files, run the matching schema or Wails generation/check commands and inspect generated churn before committing.

### Red

Do not touch in this run:

- `C:\Projects\asymmflow\ph_holdings` except read-only comparison commands.
- Live credentials, `.env`, Supabase/Fly/VPS deployment configuration.
- Unrelated sales, finance, inventory, or UI redesign code.
- Existing generated bindings without a matching source/schema/runtime reason.
- Destructive migrations or broad client-data assumptions.

## Mandatory Checkpoint Ladder

### Checkpoint 0: Start Ledger

Record in the final response and, if helpful, in `docs/BUSINESS_MEMORY_INTAKE_STATUS.md`:

- Start commit.
- `git status --short --branch`.
- Whether `ph_holdings` source-track evidence is being used only through existing reconciliation docs.
- Which checkpoint you are starting.

Commands:

```powershell
git status --short --branch
git rev-parse --short HEAD
npx gitnexus status
```

Remember: GitNexus status is informational only.

### Checkpoint 1: Source Registry Repository Boundary

Problem:

`pkg/documents/intake/source_registry.go` currently provides a pure in-memory source asset registry. That is a good kernel, but source provenance is not launch-ready until there is a repository boundary with idempotent persistence semantics.

Implement:

- Define a `SourceAssetRepository` interface in `pkg/documents/intake`.
- Add an in-memory implementation for deterministic tests.
- Add idempotent save/upsert semantics keyed by stable source ID.
- Add list/get methods that support candidate/source review workflows.
- Preserve duplicate merge behavior from `SourceAssetRegistry.Upsert`.
- Add tests covering:
  - stable ID persistence,
  - duplicate merge,
  - candidate ID merge,
  - audit ref merge,
  - first/last seen range preservation,
  - invalid/empty inputs rejected.

Verification:

```powershell
go test ./pkg/documents/... -count=1
```

Commit:

```text
feat(asymmflow): add business memory source registry repository
```

### Checkpoint 2: Durable Adapter

Problem:

Business Memory review records already have durable GORM storage. Source assets need the same adapter seam without assuming mature client databases will auto-migrate every table.

Implement:

- Add a GORM source asset repository in `pkg/adapter/documents`.
- Use an explicit `Migrate(ctx)` method, mirroring the review storage pattern.
- Store source ID, kind, path, label, hash, import batch, privacy class, processing status, candidate IDs, audit refs, first seen, last seen.
- Serialize slices predictably as JSON text or another existing local pattern.
- Preserve idempotent duplicate upsert semantics.
- Add focused adapter tests with SQLite temp DB.

Verification:

```powershell
go test ./pkg/documents/... -count=1
go test ./pkg/adapter/documents -count=1
go build ./...
```

Commit:

```text
feat(asymmflow): persist business memory source assets
```

### Checkpoint 3: Review Queue And Export Integration

Problem:

The review queue and export bundle should expose source registry provenance, not only candidate-local source fields.

Implement:

- Extend queue-state or export construction so a selected candidate can include registry-derived source metadata.
- Preserve JSON export/replay behavior.
- Preserve TOON export source citation.
- Add tests for source registry provenance appearing in review/export output.
- Do not create authoritative accounting/CRM/inventory records.

Verification:

```powershell
go test ./pkg/documents/... -count=1
go test ./pkg/adapter/documents -count=1
```

Commit:

```text
feat(asymmflow): include source registry provenance in business memory exports
```

### Checkpoint 4: Runtime And ViewModel Surface

Problem:

Operators need to see whether a Business Memory candidate came from a durable source asset and what state that asset is in.

Implement:

- Wire source registry repository construction through the Business Memory runtime/service path.
- Expose source registry summary fields through `internal/viewmodel/documents`.
- Update the Inbox Business Memory surface with a compact provenance summary:
  - source ID,
  - kind,
  - label/path,
  - privacy class,
  - processing status,
  - candidate count or current candidate link,
  - audit ref count.
- Keep UI consistent with existing AsymmFlow operational style. Do not redesign the screen.
- Refresh Wails bindings only if method signatures change.

Verification:

```powershell
go test ./internal/viewmodel/documents -count=1
go test ./pkg/documents/... -count=1
go test ./pkg/adapter/documents -count=1
go build ./...
npm.cmd --prefix frontend run check
```

Expected frontend baseline:

- `npm.cmd --prefix frontend run check` may report 0 errors and the known baseline warnings. Do not claim warnings are fixed unless you actually fix them.

Commit:

```text
feat(asymmflow): surface business memory source provenance
```

### Checkpoint 5: Manifest, Status, And Action Inventory Update

Problem:

The manifests and status docs must reflect what is now current versus future. The previous audit noted that `next_goal.primary_gap` is stale after the durable-contract bridge.

Implement:

- Update `docs/modules/business_memory_intake.manifest.json`:
  - source registry repository/adapter status,
  - durable storage status,
  - tests,
  - launch readiness,
  - next gaps.
- Update `docs/BUSINESS_MEMORY_INTAKE_STATUS.md`:
  - new checkpoint ledger entries,
  - commands run,
  - baseline warnings,
  - remaining gaps.
- Add a short note that PH source-track lessons are carried as acceptance constraints:
  - revision provenance,
  - allocation-aware evidence,
  - approval queues,
  - sync/conflict policy,
  - UI/backend action inventories.

Verification:

```powershell
Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null
git diff --check
```

Commit:

```text
docs(asymmflow): record business memory source registry durability
```

### Checkpoint 6: Cashflow Allocation-Aware Preflight Promotion

Only start this after Checkpoints 1-5 are committed and verification is clean.

Problem:

Jordan's source track proved bank reconciliation requires allocation-aware evidence: customer invoices, supplier invoices, expenses, partial matches, and mixed matches. Cashflow Evidence should model this before the operator loop grows.

Implement a pure preflight only:

- Add allocation-aware input/read-model types in `pkg/cashflow/evidence`.
- Map or represent:
  - allocation ID,
  - bank statement line ID,
  - source type,
  - source ID,
  - amount,
  - allocation type,
  - confidence/status if available.
- Extend evidence pack or command center model to carry allocation state without mutating finance records.
- Add pure tests.
- Update `docs/modules/cashflow_evidence.manifest.json` and `docs/CODEX_DEV_ROADMAP_2026_05_15.md` only if needed.

Do not touch bank reconciliation mutation logic in this checkpoint.

Verification:

```powershell
go test ./pkg/cashflow/evidence -count=1
go test ./internal/viewmodel/cashflow -count=1
go build ./...
Get-Content docs\modules\cashflow_evidence.manifest.json | ConvertFrom-Json | Out-Null
```

Commit:

```text
feat(asymmflow): add allocation-aware cashflow evidence preflight
```

## Dependency Graph

```text
Checkpoint 0
  -> Checkpoint 1 source registry repository
    -> Checkpoint 2 durable adapter
      -> Checkpoint 3 export/review provenance
        -> Checkpoint 4 runtime/viewmodel/UI provenance
          -> Checkpoint 5 docs/manifest ledger
            -> Checkpoint 6 cashflow allocation preflight promotion
```

Checkpoint 6 is a promotion checkpoint. It should not begin until the Business Memory chain is committed and verified.

## Verification Matrix

Run focused gates after the checkpoint that needs them:

```powershell
go test ./pkg/documents/... -count=1
go test ./pkg/adapter/documents -count=1
go test ./internal/viewmodel/documents -count=1
go test ./pkg/cashflow/evidence -count=1
go test ./internal/viewmodel/cashflow -count=1
go build ./...
npm.cmd --prefix frontend run check
Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null
Get-Content docs\modules\cashflow_evidence.manifest.json | ConvertFrom-Json | Out-Null
git diff --check
```

If schema files change:

```powershell
powershell -ExecutionPolicy Bypass -NoProfile -File schemas\generate.ps1 -CheckOnly
```

If Wails-bound method signatures change:

```powershell
wails generate module
npm.cmd --prefix frontend run check
```

Do not run broad `go test ./...` unless a change touches broad root behavior or you have time after all required focused gates pass.

## Commit Contract

- Commit after every coherent verified checkpoint.
- Use conventional commit messages like the examples above.
- Do not commit unrelated dirty files.
- Generated files must be committed with the source/schema/runtime change that required them.
- Before each commit:

```powershell
git status --short
git diff --check
```

- Include exact verification commands in the commit body or status doc when useful.

## Multi-Codex Coordination

You are not alone in the broader roadmap. Other instances may work on different slices later.

During this run:

- Own only the Business Memory Source Registry durability chain and optional Cashflow allocation preflight.
- Do not edit Inventory/Asset Ledger code.
- Do not edit launch packaging unless documenting direct verification results.
- Do not revert files you did not change.
- If unrelated dirty files appear, ignore them unless they block your work.
- If another instance changes one of your Green files mid-run, stop and report the conflict instead of overwriting.

## Stop Conditions

Stop and leave a clear checkpoint if:

- A migration decision would affect mature client databases destructively.
- Source registry durability requires a schema/contract decision that is not obvious from current patterns.
- UI changes require a redesign beyond compact provenance display.
- A verification gate fails for reasons that require product/authority decisions rather than mechanical repair.
- You detect source-track divergence requiring a fresh `ph_holdings` comparison beyond the existing reconciliation doc.

Do not stop merely because the first commit passed. Continue down the ladder.

## Final Ledger Required

Final response must include:

- Starting commit and final commit(s).
- Changed paths.
- Checkpoints completed.
- Commands run and exact pass/fail status.
- Baseline warnings separated from new regressions.
- Any files intentionally left dirty/uncommitted.
- Residual risks.
- Next recommended checkpoint.

## What Comes After

If this run completes through Checkpoint 6, the next independent Codex instance can take:

```text
docs/CODEX_GOAL_CASHFLOW_EVIDENCE_OPERATOR_LOOP_HANDOFF.md
```

That future handoff should own the operator loop: drill-downs, allocation review state, export bundles, and event refresh points. This run should give it durable provenance and allocation-aware model foundations without trampling the UI or accounting mutation path.

## Run Ledger

### Run 2026-05-15 - Checkpoints 0 and 1

- Start commit: `00ec6cf`.
- Starting status: clean `master`.
- `ph_holdings` source-track evidence: read-only through `docs/CODEX_PH_HOLDINGS_SOURCE_TRACK_RECON_2026_05_15.md`; no source-track repo writes or blind backport.
- Active checkpoint: Checkpoint 1, Source Registry Repository Boundary.
- GitNexus status: informational; reported stale at indexed commit `6d8a69d` while current commit was `00ec6cf` because a docs-only audit commit followed the indexed source registry commit.
- GitNexus impact:
  - `SourceAssetRegistry`: LOW risk, one direct same-package impact.
  - `SourceAsset`: LOW risk, three direct same-package impacts.
- Completed changes:
  - Added `pkg/documents/intake/source_repository.go`.
  - Added `pkg/documents/intake/source_repository_test.go`.
  - Defined `SourceAssetRepository`, `SourceAssetListFilter`, and `MemorySourceAssetRepository`.
  - Added idempotent upsert, get, list, and list-by-candidate methods keyed by stable source ID.
  - Preserved duplicate merge semantics from `SourceAssetRegistry.Upsert`.
  - Added source asset validation and tests for stable ID persistence, duplicate merge, candidate/audit reference merge, seen-time range preservation, filters, and invalid input rejection.
- Verification:
  - `go test ./pkg/documents/... -count=1` passed.
- Baseline warnings/noise:
  - None for the Checkpoint 1 Go package gate.
- Next checkpoint:
  - Checkpoint 2, Durable Adapter.

### Checkpoint 2

- Completed changes:
  - Added `pkg/adapter/documents/business_memory_source_storage.go`.
  - Added `pkg/adapter/documents/business_memory_source_storage_test.go`.
  - Added `BusinessMemorySourceAssetModel` for the `business_memory_source_assets` table.
  - Added `GORMBusinessMemorySourceAssetRepository` with explicit `Migrate(ctx)`, idempotent duplicate upsert, get, list, and list-by-candidate behavior.
  - Stored source asset candidate IDs and audit refs as JSON text.
  - Reused the intake merge helper so durable duplicate upserts preserve source registry semantics.
- Verification:
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./pkg/adapter/documents -count=1` passed.
  - `go build ./...` passed.
- Baseline warnings/noise:
  - None for the Checkpoint 2 Go gates.
- Next checkpoint:
  - Checkpoint 3, Review Queue And Export Integration.

### Checkpoint 3

- Completed changes:
  - Extended `pkg/documents/intake.ReviewQueueState` with `SourceAssets`.
  - Added `ReviewService.BuildQueueStateWithSources` to enrich review queue state from a `SourceAssetRepository`.
  - Extended `ReviewExportBundle` with optional source registry assets.
  - Added `NewReviewExportBundleWithSources`.
  - Updated TOON export so source asset ID, kind, label/path/hash, privacy class, processing status, candidate IDs, and audit ref count are cited alongside candidate-local source refs.
  - Added tests for JSON replay and TOON source registry provenance.
- Verification:
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./pkg/adapter/documents -count=1` passed.
- Baseline warnings/noise:
  - None for the Checkpoint 3 Go gates.
- Next checkpoint:
  - Checkpoint 4, Runtime And ViewModel Surface.

### Checkpoint 4

- Completed changes:
  - Wired `GORMBusinessMemorySourceAssetRepository` construction and explicit migration through the Business Memory runtime queue, review-decision refresh, and export paths.
  - Upserted inbox-derived source assets before queue state and export bundle construction so durable source provenance follows the candidate.
  - Extended `internal/viewmodel/documents` with source registry summaries and selected-candidate `sourceRegistry` output.
  - Added selected-candidate ViewModel coverage for source registry candidate/audit counts and current-candidate linkage.
  - Added a compact source registry provenance block to `BusinessMemoryReviewPanel.svelte` without redesigning the Inbox surface.
  - Refreshed Wails models for `documents.SourceRegistryItemVM`, `intake.SourceAsset`, and `ReviewExportBundle.source_assets`.
- Verification:
  - `go test ./internal/viewmodel/documents -count=1` passed.
  - `go test ./pkg/documents/... -count=1` passed.
  - `go test ./pkg/adapter/documents -count=1` passed.
  - `go test . -run "TestParseBusinessMemoryReviewDecision|TestInboxDocumentToBusinessMemoryCandidate|TestCandidateToBusinessMemorySourceAsset" -count=1` passed.
  - `go build ./...` passed.
  - `wails generate module` passed.
  - `npm.cmd --prefix frontend run check` passed with 0 errors and 13 baseline warnings.
- Baseline warnings/noise:
  - `wails generate module` still reports the known anonymous `r1/r2/r3` prediction struct warning.
  - `npm.cmd --prefix frontend run check` still reports 13 baseline warnings in unrelated Svelte files.
- Next checkpoint:
  - Checkpoint 5, Manifest, Status, And Action Inventory Update.

### Checkpoint 5

- Completed changes:
  - Updated `docs/modules/business_memory_intake.manifest.json` so source registry durability, runtime surfacing, tests, launch readiness, and the next implementation gap are current.
  - Updated `docs/BUSINESS_MEMORY_INTAKE_STATUS.md` with the Checkpoint 5 ledger, remaining gaps, and current next architecture target.
  - Added PH source-track acceptance constraints for revision provenance, allocation-aware evidence, approval queues, sync/conflict policy, and UI/backend action inventories.
  - Added a Business Memory action inventory separating operator-visible commands from agent-safe APIs.
- Verification:
  - `Get-Content docs/modules/business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null` passed.
  - `git diff --check` passed with LF/CRLF normalization warnings only.
- Baseline warnings/noise:
  - LF/CRLF normalization warnings only.
- Next checkpoint:
  - Checkpoint 6, Cashflow Allocation-Aware Preflight Promotion.

### Checkpoint 6

- Completed changes:
  - Added allocation-aware cashflow evidence inputs and read models carrying allocation ID, bank statement line ID, source type/ID, amount, allocation type, confidence, and allocation status.
  - Extended `CommandCenter` with bank allocations plus allocation summary counts for matched, partial, mixed, conflict, and unresolved states.
  - Added read-only allocation inspection proposals without touching bank reconciliation mutation logic.
  - Extended agent briefs and JSON/TOON evidence packs with allocation summary/detail while preserving `MutatesState: false`.
  - Exposed allocation rows through `internal/viewmodel/cashflow.CommandCenterVM`.
  - Updated `docs/modules/cashflow_evidence.manifest.json` and `docs/CODEX_DEV_ROADMAP_2026_05_15.md` to mark the pure allocation-aware preflight as implemented.
- Verification:
  - `go test ./pkg/cashflow/evidence -count=1` passed.
  - `go test ./internal/viewmodel/cashflow -count=1` passed.
  - `go build ./...` passed.
  - `Get-Content docs/modules/cashflow_evidence.manifest.json | ConvertFrom-Json | Out-Null` passed.
- Baseline warnings/noise:
  - None for the Checkpoint 6 gates.
- Next checkpoint:
  - The mandatory ladder in this handoff is complete after final diff hygiene and commit.
