# Wave 2 Progress

Date: 2026-05-05

## Summary

Wave 2 established concrete service wiring and moved the Wails-facing API surface for several domains behind domain service objects. The extraction was intentionally conservative where root-package models still block a true package move: public `*App` Wails methods remain as thin wrappers, while service packages own the delegation contracts.

## Verification

- `go build ./...` passed after each slice.
- `go test ./... -count=1 -timeout 300s` passed after each committed slice.
- Latest verified commit: `a9bdac3`.

## Commits

- `322a7ea` - payment service infrastructure and payment/supplier-payment delegation
- `4c23126` - expense service delegation and private helper receiver cleanup
- `a5e7a41` - banking cash-position/reconciliation core delegation
- `d147580` - fulfillment delivery/serial delegation
- `9b2530c` - procurement PO/GRN delegation
- `a9bdac3` - license delegation

## Method Counts

- Baseline total `func (a *App)` methods at `HEAD~6`: 1203
- Current total `func (a *App)` methods: 1190
- Net receiver reduction: 13
- `app.go` receiver methods: 331 before, 331 after
- `app.go` LOC: 19,121 before, 19,124 after

The net receiver reduction is smaller than the handoff target because the Wails API methods were kept as wrappers, and this wave added service accessor methods for lazy initialization in tests. The main architectural gain is that 78 Wails-facing methods now route through domain service boundaries.

## Delegated Domains

- Payment: customer payments, supplier payments, and order payment progression wrappers route through `pkg/finance/payment`.
- Expense: expense categories, vendors, entries, recurring expenses, bank candidates, and dashboard summary wrappers route through `pkg/finance/expense`; expense-only private helpers were moved off `*App`.
- Banking: cash position and reconciliation core wrappers route through `pkg/finance/banking`; matcher algorithms were left intact for a later, deeper extraction.
- Fulfillment: delivery note CRUD/workflow and serial registration/availability wrappers route through `pkg/crm/fulfillment`.
- Procurement: purchase order and GRN port methods route through `pkg/crm/procurement`.
- License: public license lifecycle methods route through `pkg/infra/license`.

## Remaining Hotspots

- `app.go`: 331 receiver methods, still the biggest God Object surface.
- `butler_ai.go`: 61 receiver methods, intentionally deferred by the handoff.
- `collaboration_service.go`: 49 receiver methods.
- `payroll_service.go`: 33 receiver methods.
- `customer_invoice_service.go`: 26 receiver methods, intentionally deferred.
- Banking remainder: bank statement CRUD, transaction matcher, book-bank deposit/cheque helpers, and integrity/audit helpers remain on `*App`.

## Wave 3 Recommendation

1. Move root-package finance/CRM model definitions into the existing `pkg/*/domain.go` packages or convert root models into aliases. This is the blocker for moving real business logic into domain packages instead of handler delegation.
2. Extract the remaining banking matcher and integrity methods as one focused wave, preserving the matching algorithms byte-for-byte.
3. Split `app.go` by domain wrappers after Wails binding generation is confirmed unaffected, so `app.go` itself can finally drop below 200 receiver methods.
