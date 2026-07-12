# Codex Audit Evidence Index - 2026-05-15

Scope: `the AsymmFlow repository`

This appendix records the evidence sources used for the 2026-05-15 repository audit. It is intentionally compact; the larger interpretation lives in `docs/CODEX_REPO_HISTORY_AUDIT_2026_05_15.md` and the forward plan lives in `docs/CODEX_DEV_ROADMAP_2026_05_15.md`.

## Snapshot

| Item | Evidence |
|---|---|
| Current branch | `master` |
| Current HEAD | `6d8a69d14b8c74c1b9d0c11b1b58763f72d803ea` |
| Short HEAD | `6d8a69d` |
| Dirty status before audit docs | `git status --short --branch` returned only `## master` |
| GitNexus | `npx gitnexus status` reported index up to date at commit `6d8a69d` |
| Audit posture | Non-behavior-changing; docs only |

## Primary Commands

```powershell
git status --short --branch
git rev-parse --short HEAD
git rev-parse HEAD
npx gitnexus status
git log --since="2026-05-01" --date=short --pretty=format:"%h %ad %s" --shortstat
git diff --stat ac97535..6d8a69d
git diff --stat ac97535..0122f7c
git diff --stat 0122f7c..b0e4a70
git diff --stat b0e4a70..6d8a69d
git show --stat --oneline c1cf4e8 b57c66c 4cb6279 e7224f5 9c88fb8 7fb962f 3347dad 9baf305 6d8a69d
rg --files docs
rg --files -g "*.manifest.json" -g "*.schema.json" -g "*.capnp" -g "*STATUS*.md" -g "*ROADMAP*.md" -g "*HANDOFF*.md" -g "WORK_LOG.md"
rg -n "TODO|planned|future|not yet|placeholder|mock|FIXME|stub|baseline warnings|13 baseline" docs\CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md docs\BUSINESS_MEMORY_INTAKE_STATUS.md docs\modules docs\WAVE17_PROGRESS.md docs\CODEX_GOAL_COMPLETION_AUDIT_2026_05_14.md
```

## Verification Commands

```powershell
go test ./pkg/documents/... -count=1
go test ./pkg/cashflow/evidence -count=1
go test ./internal/viewmodel/... -count=1
go build ./...
npm.cmd --prefix frontend run check
Get-Content docs\modules\cashflow_evidence.manifest.json | ConvertFrom-Json | Out-Null
Get-Content docs\modules\business_memory_intake.manifest.json | ConvertFrom-Json | Out-Null
powershell -ExecutionPolicy Bypass -NoProfile -File schemas\generate.ps1 -CheckOnly
```

## Verification Results

| Gate | Result | Notes |
|---|---|---|
| `go test ./pkg/documents/... -count=1` | Passed | `pkg/documents/intake` tests ran; other document packages had no test files |
| `go test ./pkg/cashflow/evidence -count=1` | Passed | Cashflow evidence kernel/package green |
| `go test ./internal/viewmodel/... -count=1` | Passed | Cashflow, CRM, documents, and finance viewmodel tests passed |
| `go build ./...` | Passed | No output |
| `npm.cmd --prefix frontend run check` | Passed with warnings | 0 errors, 13 Svelte warnings in 10 files |
| Manifest JSON parse | Passed | Both module manifests parsed with `ConvertFrom-Json` |
| `schemas\generate.ps1 -CheckOnly` | Passed | Cap'n Proto schemas compile successfully |

## Baseline Frontend Warnings

`npm.cmd --prefix frontend run check` reported 0 errors and 13 warnings. They are treated as baseline because prior ledgers record the same count.

Warning files:

- `frontend/src/lib/screens/DashboardScreen.svelte`
- `frontend/src/lib/components/ui/WabiModal.svelte`
- `frontend/src/lib/screens/CostingSheetScreen.svelte`
- `frontend/src/lib/screens/ButlerScreen.svelte`
- `frontend/src/lib/asyl/components/QuaternionScenePlayer.svelte`
- `frontend/src/lib/asyl/components/Modal.svelte`
- `frontend/src/lib/screens/CustomersScreen.svelte`
- `frontend/src/lib/components/ui/OrigamiNav.svelte`
- `frontend/src/lib/asyl/qgif/components/QGIFLoader.svelte`
- `frontend/src/lib/components/ui/E8TopologyBadge.svelte`

## Commit Evidence

### Era 1: May 5 stabilization and extraction baseline

Representative commits:

- `ac97535 feat: initial snapshot from ph_holdings (refactor baseline)`
- `4c80c9e refactor(codex): quarantine manual tests`
- `dae65e0 refactor(codex): delete dead demo code`
- `06139ab refactor(codex): extract domain type definitions to pkg packages`
- `7fe3145 refactor(codex): implement in-process event bus`
- `18932e4 refactor(codex): reduce app shell to lifecycle surface`
- `7962c69 refactor(codex): write wave 5 progress report`

Interpretation: this era established a testable baseline, deleted or quarantined known non-runtime fixtures, created package structure, and began moving large root services into domain packages.

### Era 2: May 6-8 schemas, adapters, compliance, release, and accounting

Representative commits:

- `987f20c` through `d213a1d`: Cap'n Proto schema family, generated Go/TypeScript surfaces, adapters, and dashboard stats proto pilot.
- `5366113` through `5459165`: math packages and bridges.
- `231e38d` through `8209dd7`: Turso, CDC, OpenTelemetry, and three-regime health.
- `3715ca7` through `775f9b8`: i18n, compliance engines, hooks, and compliance ViewModel.
- `ea067e2` through `4246f1b`: release build metadata, bundle script, verification script, backup/restore preflight, and Wave 16 progress.
- `a9ff523` through `ef180b0`: Wave 17 posting preview spine, trial-balance gate, idempotent draft journal creation, coverage report, and Accounting UI readiness.

Interpretation: this era is broad and real, but several slices are foundations or readiness surfaces rather than complete operator loops.

### Era 3: May 14 module contract, Cashflow Evidence, component library, and Business Memory

Representative commits:

- `c1cf4e8 docs(asymmflow): add module contract foundation`
- `b57c66c docs(asymmflow): add engine generalization audit`
- `4cb6279 feat(asymmflow): add cashflow evidence proof slice`
- `e7224f5 feat(asymmflow): persist cashflow evidence reviews`
- `9c88fb8 feat(asymmflow): extract reusable product components`
- `7fb962f feat(asymmflow): add business memory intake kernel`
- `3347dad feat(asymmflow): bridge business memory intake to capnp`
- `9baf305 docs(asymmflow): finalize business memory durable contract ledger`
- `6d8a69d feat(asymmflow): add business memory source asset registry`

Interpretation: this era created the new module/productization shape. Cashflow Evidence and Business Memory both have tested kernels and partial UI/runtime surfaces, but their operator loops are not yet product-complete.

## Diff Scale

`git diff --stat ac97535..6d8a69d` reports:

- 650 files changed.
- 158,662 insertions.
- 47,124 deletions.

Boundary stats:

- `ac97535..0122f7c`: 509 files changed, 115,473 insertions, 47,012 deletions.
- `0122f7c..b0e4a70`: 31 files changed, 2,587 insertions, 7 deletions.
- `b0e4a70..6d8a69d`: 147 files changed, 44,076 insertions, 3,579 deletions.

## High-Signal Current Artifacts

- `docs/CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md`
- `docs/MODULE_CONTRACT_FOUNDATION.md`
- `docs/CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT.md`
- `docs/CODEX_GOAL_CASHFLOW_EVIDENCE_HANDOFF.md`
- `docs/CODEX_GOAL_PRODUCT_COMPONENT_LIBRARY_HANDOFF.md`
- `docs/CODEX_GOAL_BUSINESS_MEMORY_INTAKE_HANDOFF.md`
- `docs/CODEX_GOAL_BUSINESS_MEMORY_DURABLE_CONTRACT_HANDOFF.md`
- `docs/BUSINESS_MEMORY_INTAKE_STATUS.md`
- `docs/WAVE17_PROGRESS.md`
- `docs/modules/cashflow_evidence.manifest.json`
- `docs/modules/business_memory_intake.manifest.json`
- `schemas/documents.capnp`
- `pkg/cashflow/evidence`
- `pkg/documents/intake`
- `pkg/adapter/documents/business_memory.go`
- `pkg/adapter/documents/business_memory_storage.go`
- `business_memory_review_runtime.go`
- `cashflow_evidence_service.go`
- `cashflow_evidence_review.go`
- `accounting_posting_service.go`
- `internal/viewmodel/cashflow/evidence_vm.go`
- `internal/viewmodel/documents/documents_vm.go`
- `frontend/src/lib/components/ui/ActionProposalCard.svelte`
- `frontend/src/lib/components/ui/EvidenceSourceList.svelte`
- `frontend/src/lib/components/ui/KpiStatusStrip.svelte`
- `frontend/src/lib/components/documents/BusinessMemoryReviewPanel.svelte`
