# Wave 3 Progress

Date: 2026-05-05

## Summary

Wave 3 completed the main `app.go` surface split and extended the banking service boundary over the bank transaction matcher entry points. The Wails-facing `*App` method names were preserved, but large domain surfaces now live in focused root-package files instead of the startup/app infrastructure file.

## Verification

- `go build ./...` passed after the app surface split.
- `go test ./... -count=1 -timeout 300s` passed after the app surface split.
- `go build ./...` passed after banking matcher delegation.
- `go test ./... -count=1 -timeout 300s` passed after banking matcher delegation.
- Latest verified code commit: `e195fff`.

## Commits

- `775b5a1` - split app surface into domain files
- `e195fff` - route banking matcher through banking service

## Method Counts

- Baseline `app.go` receiver methods at `8097467`: 331
- Current `app.go` receiver methods: 194
- `app.go` receiver reduction: 137
- Baseline `app.go` LOC: 19,124
- Current `app.go` LOC: 11,533
- `app.go` LOC reduction: 7,591
- Total `func (a *App)` methods repo-wide: 1190 before, 1190 after

The total receiver count is intentionally unchanged because Wave 3 preserved Wails bindings and moved methods to root-package adapter files. The architectural win is that `app.go` is now mostly application lifecycle, infrastructure setup, auth/RBAC, OCR/setup, accounting/inventory tail code, and remaining backend surfaces.

## Files Created

- `app_prediction_dashboard.go`
- `app_crm_surface.go`
- `app_sales_pipeline.go`
- `app_order_customer_surface.go`

## Banking Matcher Boundary

The following matcher entry points now route through `pkg/finance/banking`:

- `AutoMatchBankLines`
- `ManualMatchLine`
- `UnmatchLine`
- `CreateSplitAllocation`
- `CategorizeTransactions`

The matcher logic itself was preserved in place in `bank_transaction_matcher.go`.

## Remaining Hotspots

- `app.go`: 194 receiver methods, now below the Wave 3 target but still the largest file.
- `butler_ai.go`: 61 receiver methods, still intentionally deferred.
- `app_sales_pipeline.go`: 57 receiver methods, now isolated and ready for package-level service extraction.
- `app_order_customer_surface.go`: 50 receiver methods.
- `collaboration_service.go`: 49 receiver methods.
- `payroll_service.go`: 33 receiver methods.
- `customer_invoice_service.go`: 26 receiver methods, still deferred due to coupling.

## Wave 4 Recommendation

1. Split the remaining `app.go` tail by infrastructure domain: auth/RBAC, setup/OCR routing, graph/SSOT, accounting/inventory, and data fixes.
2. Convert root-domain models to aliases of `pkg/*/domain.go` models so Wave 2/3 delegation can become real logic movement into packages.
3. Extract `app_sales_pipeline.go` into pipeline/offer/order service packages once model aliases are in place.
