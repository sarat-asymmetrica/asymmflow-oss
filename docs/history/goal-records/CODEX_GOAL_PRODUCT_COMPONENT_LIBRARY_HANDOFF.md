# CODEX_GOAL_PRODUCT_COMPONENT_LIBRARY_HANDOFF

Status: Ready for Codex
Created: 2026-05-14
Repo: `the AsymmFlow repository`
Time box: 2 hours
Depends on: `2d933aa docs(asymmflow): refresh goal completion audit`

## Goal Command

```text
/goal Run the AsymmFlow roadmap chain for 2 hours.

Complete the Reusable Product Component Library first implementation slice. Extract the reusable UI/ViewModel patterns proven by Cashflow Evidence into documented, reusable product components without changing backend business behavior. If this completes early, checkpoint it and advance to Business Memory Intake planning.
```

## Read First

Read these before editing:

1. `C:\Projects\ASYMMETRICA_ECOSYSTEM_LOG.md`
2. `C:\Projects\AGENTIC_SWARM_PROTOCOL.md`
3. `C:\Projects\ASYMMETRICA_PRODUCT_UI_AND_ARCHITECTURE_STANDARD.md`
4. `docs\CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md`
5. `docs\MODULE_CONTRACT_FOUNDATION.md`
6. `docs\CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT.md`
7. `docs\CODEX_GOAL_CASHFLOW_EVIDENCE_HANDOFF.md`
8. `docs\CODEX_GOAL_COMPLETION_AUDIT_2026_05_14.md`
9. `frontend\src\lib\components\ui\DESIGN_SYSTEM_COMPONENTS.md`
10. `frontend\src\lib\components\ui\index.ts`
11. `internal\viewmodel\shared`

Treat older roadmaps and historical wave docs as evidence, not canon.

## Current Baseline

The previous 2-hour AsymmFlow chain completed:

- Engine Generalization audit.
- Cashflow Evidence backend/read-model proof.
- Deterministic JSON/TOON evidence-pack export.
- Butler-facing brief.
- Action proposals.
- Invoice/follow-up evidence sources.
- Persisted proposal review queue.
- Accounting signoff controls.
- Final `go test ./... -count=1 -timeout 300s`, frontend check/build, Wails generation, and clean status.

Current clean base:

```text
2d933aa docs(asymmflow): refresh goal completion audit
6ae5f40 test(asymmflow): cover cashflow review status aliases
e6fe0f0 docs(asymmflow): add goal completion audit
```

## Product Bar

Use the "$1000/mo justification paradigm."

This wave's ROI proof is product velocity and UX trust. Cashflow Evidence proved that AsymmFlow modules need repeated operator surfaces: KPI/status strips, evidence source rows, action proposal cards, review/signoff queues, audit trails, source links, and export actions. If each module hand-rolls these, UI drift and hidden behavior debt will compound.

Apply four filters:

- ROI proof: reduce repeated UI work, lower future module implementation time, and prevent janky bespoke module screens.
- Workflow closure: end with reusable components, docs, a fixture/showcase surface, and at least one real screen consuming the library.
- Engine leverage: map backend/ViewModel authority into stable MVVM UI contracts that future modules can reuse.
- Operator trust: every component should make state, provenance, correction, approval, export, and audit semantics inspectable.

## Architecture Constraints

- Preserve MVVM. Backend/ViewModel contracts own business state; Svelte components render and emit intent.
- Do not change Cashflow Evidence backend behavior unless a compile/type issue forces a tiny adapter.
- Do not add dependencies unless the current stable version, license, and maintenance status are checked from authoritative sources and documented.
- Use the AsymmFlow product UI standard: warm surfaces, forest primary, gold accent, restrained shadows, stable dimensions, no generic palette drift.
- Avoid nested cards and marketing-style composition. These are operational components, not landing-page blocks.
- Do not use visible in-app instructional text to explain the design system.
- Favor icons already available in the repo; do not introduce a new icon library during this wave.
- Text must fit on 375px mobile width and desktop. Prefer stable grids, responsive constraints, and compact labels.

## Primary Goal: Reusable Product Component Library

Create the first reusable product component layer for AsymmFlow-class modules.

The first slice should focus on components extracted from Cashflow Evidence because they now have real behavior and operator semantics.

Candidate components:

- `KpiStatusStrip` or equivalent: compact status/metric cards for command centers.
- `EvidenceSourceList` or equivalent: source type, completeness, confidence, status, and last-updated rows.
- `ActionProposalCard` or equivalent: advisory action, priority, required deterministic service, reason, and review status.
- `ReviewQueuePanel` or equivalent: pending/approved/rejected/needs-input/superseded proposal states with signoff controls.
- `AuditTrailList` or `SourceLinkList` if the current screen has enough repeated markup to extract safely.
- `EmptyState`, `LoadingState`, and `ErrorState` wrappers only if existing components do not already serve the need.

Names may change if the existing design system suggests better names.

## Required Work

1. Inventory the existing component surface:
   - `frontend/src/lib/components/ui`
   - `frontend/src/lib/components/layout`
   - relevant screen-local Cashflow Evidence markup in `AccountingScreen.svelte`
   - existing ViewModel shared primitives under `internal/viewmodel/shared`
2. Write `docs/PRODUCT_COMPONENT_LIBRARY_INVENTORY.md`:
   - current reusable components
   - duplicated screen-local patterns
   - recommended component families
   - first extraction targets
   - components deferred to later waves
3. Add or extend shared ViewModel primitives if needed:
   - status badge
   - KPI summary card
   - evidence source row
   - action proposal display/review state
   - audit/source link item
4. Extract at least two real reusable Svelte components from Cashflow Evidence UI.
5. Refactor `AccountingScreen.svelte` to consume the new components without changing behavior.
6. Export the new components from `frontend/src/lib/components/ui/index.ts` or a clearly named product-component index.
7. Add a fixture/demo surface using an existing showcase route/component if practical. Prefer enhancing existing `ShowcaseScreen.svelte`, `DesignSystemShowcase.svelte`, or component demo files instead of inventing a new app shell.
8. Update docs with usage examples and module-fit guidance.

## Secondary Goal: Business Memory Intake Spec

If the product component slice completes and is committed before the 2-hour window ends, begin the next roadmap goal by writing:

```text
docs/CODEX_GOAL_BUSINESS_MEMORY_INTAKE_HANDOFF.md
```

Do not implement Business Memory Intake unless the component slice is already complete, verified, and committed.

The Business Memory Intake handoff should target:

```text
unstructured message/document/folder -> classified object -> extracted fields -> review/correction -> linked business record -> audit trail
```

## Non-Goals

- Do not redesign the entire app.
- Do not replace the existing design system wholesale.
- Do not introduce Storybook or a new UI framework.
- Do not change backend Cashflow Evidence semantics.
- Do not build Business Memory Intake in this wave unless the component library slice completes early and there is verified time remaining.
- Do not create new accounting authority or agent mutation paths.

## Suggested Worker Wave Plan

Use subagents only where they materially increase throughput and keep ownership disjoint.

| Worker | Ownership | Output |
| --- | --- | --- |
| Orchestrator | Architecture, write boundary, integration, final commit | Consolidated component slice |
| UI inventory worker | Component inventory and duplicated-pattern map | `docs/PRODUCT_COMPONENT_LIBRARY_INVENTORY.md` |
| Component worker | Svelte reusable components and exports | Reusable components plus demo fixture |
| Integration worker | `AccountingScreen.svelte` refactor only | Same behavior through reusable components |
| Verification worker | Frontend checks, focused Go tests if ViewModels touched, final status | Exact command evidence |

Workers are not alone in the codebase. They must not revert edits by others and must keep to assigned write scopes.

## Verification Gates

Minimum:

```powershell
git status --short --branch
npm.cmd --prefix frontend run check
npm.cmd --prefix frontend run build
git diff --check
git status --short --branch
```

If Go ViewModel files are touched:

```powershell
go test ./internal/viewmodel/... -count=1
go test ./pkg/cashflow/evidence -count=1
go build ./...
```

If Wails bindings are touched:

```powershell
wails generate module
npm.cmd --prefix frontend run check
npm.cmd --prefix frontend run build
```

If a live UI check is practical inside the time box, run a focused browser/Playwright smoke at desktop and 375px width. If not practical, state why.

Known baseline noise:

- Wails anonymous-struct warning may remain.
- Existing Svelte warnings may remain if count/location is unchanged.

## Commit Policy

Make one coherent rollback-safe commit after verification.

Suggested message:

```text
feat(asymmflow): extract reusable product components
```

If the result is docs-only, use:

```text
docs(asymmflow): specify product component library
```

Stage only goal-owned files. Do not commit generated runtime/build artifacts.

## Exit Criteria

- `docs/PRODUCT_COMPONENT_LIBRARY_INVENTORY.md` exists.
- At least two reusable components exist and are exported.
- `AccountingScreen.svelte` consumes the new components without behavior changes.
- Demo/fixture/showcase coverage exists or is clearly deferred with rationale.
- Frontend check/build pass or baseline warnings are recorded.
- Go checks pass if ViewModel code changes.
- A rollback-safe commit exists.
- Next goal is clear: Business Memory Intake handoff.
