# Evidence-First AsymmFlow Repo History Audit - 2026-05-15

Scope: `the AsymmFlow repository`

## Executive Readout

This audit treats commits, current code, tests, manifests, schemas, and generated surfaces as primary evidence. Planning docs, handoffs, progress logs, and old repo-local instructions are useful historical artifacts, but they are not canon unless backed by shipped code or tests.

Current HEAD is `6d8a69d14b8c74c1b9d0c11b1b58763f72d803ea` on `master`. The repo was clean before these audit docs were added. GitNexus was up to date at `6d8a69d`.

The May 2026 work is materially real. It moved the repo from a large inherited ERP snapshot into a more modular architecture with package boundaries, schema contracts, adapter bridges, ViewModels, release tooling, an accounting posting spine, Cashflow Evidence, reusable UI components, and Business Memory Intake. The main risk is not that nothing shipped. The risk is that the repo now contains a large amount of foundation and partial loop work that can look more complete than it is unless every claim is anchored to code, tests, and operator closure.

## Current Verification

| Gate | Result | Interpretation |
|---|---|---|
| `go test ./pkg/documents/... -count=1` | Passed | Business Memory/intake package tests are green |
| `go test ./pkg/cashflow/evidence -count=1` | Passed | Cashflow Evidence pure package is green |
| `go test ./internal/viewmodel/... -count=1` | Passed | ViewModel packages compile and focused tests pass |
| `go build ./...` | Passed | Current Go tree builds |
| `npm.cmd --prefix frontend run check` | Passed with warnings | 0 errors, 13 baseline Svelte warnings |
| Manifest JSON parse | Passed | `cashflow_evidence` and `business_memory_intake` manifests parse |
| `schemas\generate.ps1 -CheckOnly` | Passed | Cap'n Proto schema compilation check passes |

Baseline warnings are the 13 Svelte warnings previously recorded in status ledgers. They remain product debt, but they are not introduced by this audit.

## Timeline Reconstruction

### Era 1: May 5 stabilization and extraction baseline

Evidence:

- `ac97535 feat: initial snapshot from ph_holdings (refactor baseline)` imported the large Acme Instrumentation/AsymmFlow baseline.
- Manual and data-dependent tests were quarantined or guarded by commits such as `4c80c9e`, `6f1920b`, `2a78729`, and `b94588a`.
- Demo/dead code was removed in `dae65e0`.
- Domain package structure and ports began in `d732cdb`, `06139ab`, `7a7b272`, `4db2747`, and `7fe3145`.
- Service extraction and aliasing continued through Waves 2-7.

State assessment:

- Shipped: package structure, event bus, domain types, finance/CRM/document/butler/sync/infra ports, service extractions, model aliasing, and a smaller app shell.
- Partially wired: extracted services exist, but not all domain packages are equally authoritative. Several root-level Wails methods and old service files remain as integration surfaces.
- Planning-only/stale: older phase claims inside `AGENTS.md` predate the May waves and should be treated as history, not present-state truth.

Confidence: high for structural extraction because commit stats and files are direct evidence. Medium for behavioral completeness of every extracted domain because this audit did not run the full broad test suite or manual runtime flows.

### Era 2: May 6-8 schemas, adapters, compliance, release, and Wave 17 accounting

Evidence:

- `schemas/*.capnp`, generated Go schemas, and generated TypeScript schema interfaces landed on May 6.
- `pkg/adapter/{finance,crm,butler,documents,infra,sync}` landed with roundtrip tests for several packages.
- `pkg/math/*` and `pkg/compliance/*` landed with focused tests.
- `pkg/sync/turso`, `pkg/infra/otel`, and `pkg/infra/health` landed as additive platform slices.
- Wave 16 release tooling landed in `pkg/infra/release`, `scripts/build_release_windows.ps1`, `scripts/verify_release.ps1`, and `scripts/preflight_backup_restore.ps1`.
- Wave 17 accounting landed in `pkg/finance/posting`, `accounting_posting_service.go`, generated Wails bindings, and Accounting UI readiness changes.

State assessment:

- Shipped: Cap'n Proto schema family, generated schema surfaces, adapter packages, compliance engines, math packages, release metadata/tooling, and posting preview/trial-balance/coverage foundations.
- Partially wired: Wave 17 deliberately stops short of auto-posting. `docs/WAVE17_PROGRESS.md` says the slice creates balanced posting intent, draft journal entries, source links, and ledger gates, but does not auto-post generated drafts or mutate account balances for invoices/payments.
- Planning-only/stale: v0.1 inventory ledger, setup/import wizard, hardening, pilot feedback, and expansion modules remain roadmap items.

Confidence: high for package existence and focused tests. Medium for release installability because this audit ran `go build ./...`, not the full `scripts/verify_release.ps1` or a Wails packaged app run. Medium for accounting loop closure because the posting spine is intentionally additive and review-first.

### Era 3: May 14 module contract, Cashflow Evidence, component library, Business Memory, durable contract, and source registry

Evidence:

- `c1cf4e8` added module contract foundation docs and manifest template.
- `b57c66c` added engine generalization audit and the first Cashflow Evidence handoff.
- `4cb6279` added `pkg/cashflow/evidence`, manifest, ViewModel, and tests.
- `8a1dce0`, `e5a67d8`, `9be9565`, `e04a8fb`, `e7224f5`, and related commits added exports, agent briefs, action proposals, review persistence, bindings, and Accounting UI surfaces.
- `9c88fb8` extracted `ActionProposalCard`, `EvidenceSourceList`, `KpiStatusStrip`, component docs, and a showcase entry.
- `7fb962f` added the Business Memory intake kernel, normalizers, fixture tests, and manifest.
- `3347dad` bridged Business Memory to Cap'n Proto through generated schema surfaces and adapter tests.
- `d528999`, `c399d63`, `c6c26fb`, `202ced0`, and `9baf305` added durable review storage/service, Wails/UI review workflow, export/replay, and final ledger updates.
- `6d8a69d` added the Business Memory source asset registry and tests.

State assessment:

- Shipped: module contract docs/template, Cashflow Evidence pure kernel/read-model package, Business Memory intake pure kernel, Cap'n Proto Business Memory adapter bridge, durable review repository/service, review UI/runtime methods, export/replay, first reusable UI components, and source asset registry.
- Partially wired: Cashflow Evidence is still a command-center proof, not a complete collections/reconciliation operating loop. Business Memory Source Asset Registry is a pure tested registry; durable storage, event publication, sync envelopes, and bridge into Inventory/Asset Evidence remain future work.
- Planning-only/stale: the 21-checkpoint durable-contract ladder was partially advanced, but not all continuation items are complete. Manifests still contain future/module-aware sync language.

Confidence: high for the implemented kernels and adapter tests because focused verification passed. Medium for UI/operator closure because this audit ran `svelte-check` but did not launch the app or perform browser interaction. Medium for durable cross-module readiness because the Cap'n Proto check passes, but module-aware sync/event contracts are not completed.

## Capability Classification

| Capability | Current classification | Evidence | Confidence |
|---|---|---|---|
| Test-gate stabilization | Shipped | Manual-test quarantine, fixture skips, demo deletion, green focused gates | High |
| Domain extraction and ports | Shipped foundation | `pkg/*` packages, root app split files, ports, service extractions | High |
| Cap'n Proto schema family | Shipped foundation | `schemas/*.capnp`, generated Go/TS files, check-only schema compile | High |
| Adapter roundtrips | Shipped in slices | `pkg/adapter/*` tests and Business Memory adapter tests | High for tested adapters |
| Math/compliance engines | Shipped foundation | `pkg/math/*`, `pkg/compliance/*`, focused package tests in history | Medium-high |
| Release tooling | Shipped tooling | release manifest, build script, verify script, preflight docs | Medium |
| Wave 17 posting spine | Shipped foundation, partial operator loop | `pkg/finance/posting`, service endpoints, coverage report, trial-balance gate | High for preview/gate, medium for complete accounting workflow |
| Cashflow Evidence | Partial operator loop | Pure package, ViewModel, exports, review surface, action proposals | High for kernel, medium for product loop |
| Product component library | Shipped first extraction | `ActionProposalCard`, `EvidenceSourceList`, `KpiStatusStrip`, docs/showcase | Medium-high |
| Business Memory Intake | Partial but substantial | `pkg/documents/intake`, normalizers, review service/storage, runtime/UI, export/replay | High for kernel/review/export, medium for end-to-end intake product |
| Business Memory source registry | Shipped pure checkpoint | `source_registry.go` and tests | High for pure registry, low-medium for durability |
| Inventory + Asset Evidence Ledger | Planning/preflight only | Roadmap references, no dedicated current module evidence found | Low |
| Agent-safe module surfaces | Partially established | manifests forbid agent mutations; review service rejects agent decision recording | Medium |
| Launch-ready pilot OS | Planning only | v0.1 roadmap and release tooling exist, but no complete pilot surface verified here | Low-medium |

## Stale Or Contradictory Documentation

- `AGENTS.md` still carries March-era phase/status claims and old "STILL PENDING" text. It also contains GitNexus guidance that remains useful, but its product-state claims should not drive current planning without commit/code checks.
- `docs/MASTER_PLAN.md` is broad historical context. Current wave state should prefer dated roadmaps and current manifests.
- `docs/V0_1_RELEASE_ROADMAP_2026_05_08.md` is accurate as a roadmap/log through Wave 17 but is not updated with the May 14 productization wave status.
- `docs/CODEX_MASTER_GOAL_ROADMAP_2026_05_14.md` is the strongest current planning spine, but it is still a roadmap, not proof. Its current-next-goal section lags behind `6d8a69d` because the source registry has already landed.
- `docs/modules/business_memory_intake.manifest.json` still says the primary gap is making Cap'n Proto the current durable/cross-module contract, while later status evidence says the Cap'n Proto bridge is now current. The manifest is not entirely wrong, but its `next_goal.primary_gap` is stale after the durable-contract commits.
- Module manifests use future-oriented event/sync language. Treat `publishes`, `subscribes`, and `sync_impact` as desired contracts unless backed by event-bus wiring and tests.

## Architectural Debt And Product Risk

1. The repo has a strong deterministic nucleus, but old root-level services, Wails surfaces, generated bindings, and newer domain packages coexist. This is manageable, but every new module must state its authority boundary.
2. Several May 14 modules have good pure kernels and manifests but incomplete closed operator loops. The next sprint should favor workflow closure over adding more foundations.
3. Event publication and module-aware sync are repeatedly named but not yet proved as durable cross-module infrastructure.
4. Generated schema/bindings are large and easy to churn. Schema changes need check-only generation plus a deliberate generated-artifact review.
5. Frontend `svelte-check` is clean for errors but carries 13 baseline warnings. The warnings are not blocking this audit, but they should be burned down before launch-hardening claims.
6. `AGENTS.md` and older phase docs can mislead agents into believing March-era statuses or old pending lists are current. A later docs hygiene pass should mark historical files clearly.
7. Business Memory and Cashflow Evidence both need event/audit consistency before agent surfaces should be expanded.

## Epistemic Status

- Proven by current verification: focused document, cashflow, ViewModel tests; Go build; manifest parse; Cap'n Proto schema check; Svelte check with baseline warnings.
- Proven by commit/file evidence: package existence, manifest contents, generated schema surfaces, Wails binding changes, component extraction, roadmap/status docs.
- Assumption-scaffolded: how complete the UI feels to an operator, because no browser/manual app smoke was run in this audit.
- Heuristic: 2-4 week roadmap ordering and sprint sizing.
- Not claimed: full release readiness, broad `go test ./...`, Wails packaged build, live app browser walkthrough, or real client data import validation.

## Audit Conclusion

AsymmFlow is now more than a refactor experiment. It has a usable deterministic substrate, a module contract language, schema infrastructure, accounting gates, and two serious emerging product loops: Cashflow Evidence and Business Memory Intake. The next work should resist the temptation to add another abstract framework. The highest-value path is to close the partially wired loops: durable source registry, evented reviews, export/replay ergonomics, Cashflow operator state, and the first Inventory/Asset Evidence bridge.
