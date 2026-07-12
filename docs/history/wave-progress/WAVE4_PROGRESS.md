# Wave 4 Progress

Date: 2026-05-05

## Summary

Wave 4 reduced `app.go` to the lifecycle and migration shell, then extended the banking service boundary over statement CRUD plus reconciliation audit/integrity entry points.

The Wails-facing `*App` method names remain unchanged. The main change is file and boundary ownership: application startup now lives in `app.go`, domain surfaces live in focused root adapter files, and a larger banking slice routes through `pkg/finance/banking`.

## Verification

- `go build ./...` passed at Wave 4 baseline.
- `go build ./...` passed after splitting the remaining `app.go` tail.
- `go test ./... -count=1 -timeout 300s` passed after the app shell split.
- `go build ./...` passed after the banking statement/audit facade extension.
- `go test ./... -count=1 -timeout 300s` passed after the banking facade extension.

## Commits

- `18932e4` - reduce app shell to lifecycle surface
- `097bf6a` - route bank statements through banking service

## App Shell Reduction

Baseline at start of Wave 4:

- `app.go` receiver methods: 194
- `app.go` LOC: 11,533

Current:

- `app.go` receiver methods: 19
- `app.go` LOC: 1,916
- Repo-wide `func (a *App)` methods: 1,190

Receiver method count repo-wide is intentionally unchanged. Wave 4 preserved the Wails API and moved the remaining surfaces out of the startup shell.

## Files Created

- `app_setup_documents_surface.go`
- `app_graph_contract_surface.go`
- `app_auth_rbac.go`
- `app_costing_exports_surface.go`
- `app_accounting_inventory.go`
- `app_dashboard_datafix_surface.go`
- `docs/DOMAIN_MODEL_ALIAS_PLAN.md`
- `docs/WAVE4_PROGRESS.md`

## App Surface Split

The remaining `app.go` tail was split into:

- Setup, OCR, document, and path-related surface.
- Graph and contract generation surface.
- Auth, RBAC, role/user/device surface.
- Costing and export surface.
- Accounting and inventory surface.
- Dashboard and data-fix surface.

The remaining `app.go` methods are now lifecycle, shutdown, startup, path resolution, CSRF, and migration helpers.

## Banking Boundary Extension

The following entry points now route through `pkg/finance/banking`:

- `GetBankStatements`
- `GetBankStatementByID`
- `CreateBankStatement`
- `UpdateBankStatement`
- `DeleteBankStatement`
- `GetBankStatementLines`
- `GetUnmatchedLines`
- `UpdateBankStatementLine`
- `CreateBankStatementLine`
- `DeleteBankStatementLine`
- `ValidateStatementContinuity`
- `GetBalanceContinuityReport`
- `ComputeStatementHash`
- `CheckDuplicateStatement`
- `SaveStatementHash`
- `ForceReimportStatement`
- `LogReconciliationAction`
- `GetAuditTrail`
- `GetAuditTrailByDateRange`
- `ReverseAction`

As with Wave 3, the heavy SQL bodies still live in root helper functions. The value of this step is that the service boundary is now explicit and ready for a model-alias-driven logic move.

## Alias Plan

Created `docs/DOMAIN_MODEL_ALIAS_PLAN.md` to define the bridge from duplicated root structs to package-owned domain models.

The recommended next slice is:

- Alias low-coupling finance models first.
- Alias banking models second.
- Move banking SQL bodies into `pkg/finance/banking` once model ownership is clean.
- Defer invoices, offers, orders, customers, supplier invoices, auth, and sync models until the low-risk batches prove the pattern.

## Remaining Hotspots

- `butler_ai.go` remains the largest high-method service file.
- `app_sales_pipeline.go`, `app_order_customer_surface.go`, and `app_crm_surface.go` are now isolated but still root-package adapters.
- `database.go` still owns canonical GORM models.
- Domain package model duplicates exist and need alias reconciliation before deeper logic extraction.
- Wails v3, Svelte 5, Turso, and ncruces migrations remain later-pass structural upgrades.

## Wave 5 Recommendation

Do the model-alias bridge:

1. Alias Batch 1 finance leaf models.
2. Alias Batch 2 banking models.
3. Move banking statement/reconciliation/matcher SQL bodies from root helpers into `pkg/finance/banking`.
4. Keep every model family in its own commit with full `go test ./... -count=1 -timeout 300s` verification.
