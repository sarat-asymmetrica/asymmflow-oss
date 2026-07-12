# Wave 14 Progress Report

Date: 2026-05-07  
Project: AsymmFlow (`asymmflow`)  
Scope: Svelte 5 + Go service architecture + Wails v3 validation

## Commit Ledger

| Ticket | Commit | Status | Summary |
|---|---|---|---|
| 1 | `01605f0` | Complete | Validated Wails v3, Svelte 5 migrator, and Pretext availability. |
| 2 | `e2bf1b4` | Complete | Created domain service delegation layer. |
| 3 | `12f5ea6` | Complete | Bound domain services in Wails v2 and regenerated bindings. |
| 4 | `9698f3b` | Complete | Migrated frontend function imports to domain service namespaces. |
| 5 | `b18a684` | Complete | Upgraded Svelte 5 package stack. |
| 6 | `b66283d` | Complete | Ran automated Svelte 5 migration over `frontend/src`. |
| 7 | `e95fd05` | Complete | Fixed Svelte 5 migration edge cases and type errors. |
| 8 | `357bd6f` | Deferred | Documented Wails v3 runtime-switch defer. |
| 9 | `c1cfc4f` | Complete | Ran full build verification and stabilized generated model namespaces. |
| 10 | _this commit_ | Complete | Wrote Wave 14 progress audit and updated master plan. |

## Service Migration Metrics

Created 6 Wails-bound domain service facades, all delegating to `*App` with no behavior changes:

| Service | Delegated Methods |
|---|---:|
| `FinanceService` | 241 |
| `CRMService` | 228 |
| `ButlerService` | 35 |
| `DocumentsService` | 99 |
| `SyncServiceBinding` | 54 |
| `InfraService` | 182 |
| **Total** | **839** |

Generated Wails namespaces now exist under `frontend/wailsjs/go/main/`:
- `FinanceService`
- `CRMService`
- `ButlerService`
- `DocumentsService`
- `SyncServiceBinding`
- `InfraService`

## Frontend Migration Metrics

- Svelte upgraded to `5.55.5`.
- `@sveltejs/vite-plugin-svelte` pinned to `4.0.4` for Vite 5 compatibility.
- `svelte-chartjs` upgraded to Svelte 5-compatible `4.0.1`.
- Automated migrator ran on `frontend/src` only.
- 184 source files were touched by the automated Svelte 5 migration.
- 18 files were manually fixed for Svelte 5 edge cases.
- 59 frontend files had Wails function imports split from `main/App` into domain service imports.
- 63 `main/App` references remain intentionally for retained glue, docs/demo strings, or methods left on `App`.

## Wails v3 Status

Deferred for this wave.

What worked:
- Source-installed `wails3`.
- `wails3 version` reported `v3.0.0-dev`.
- `wails3 doctor` passed on Windows amd64 with WebView2 available.

Why deferred:
- Direct `go install github.com/wailsapp/wails/v3/cmd/wails3@latest` failed because the published module contains `replace` directives.
- Source checkout hit Windows long-path failures inside Wails v3 generator testdata, though the CLI still built.
- The current backend uses Wails v2 runtime APIs for events, dialogs, file pickers, and message dialogs across many root service files.
- Wails v3 uses `pkg/application` and a different runtime/dialog/event surface, so the actual switch needs a dedicated adapter pass.

Wave 14 still delivered the most important prerequisite: domain services are now bound independently and can be registered as Wails v3 services in a later wave.

## Pretext Status

Deferred.

`go list -m github.com/nicholasgasior/pretext@latest` failed because the repository was not found. Existing PDF libraries remain in place.

## Breaking Changes

Expected behavior changes: zero.

This wave intentionally kept `App` bound for backward compatibility while adding domain service bindings and gradually migrating frontend imports.

## Final Gates

Passed:
- `go build ./...`
- `go test ./... -count=1 -timeout 300s`
- `cd frontend; npm run build`
- `cd frontend; npm run check` (0 errors, 13 warnings)
- `wails build`

Output:
- `build/bin/AsymmFlow.exe`

Known warnings:
- Svelte compiler/check warnings remain for a small number of accessibility and state-reference diagnostics. They do not fail the build or type gate.

