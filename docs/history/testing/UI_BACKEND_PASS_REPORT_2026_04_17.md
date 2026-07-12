# UI Backend Pass Report - 2026-04-17

Purpose: record the first backend-first pass over the 515-button/action inventory.

## Result Summary

- Screen-level actions audited: 515
- Backend-backed actions statically mapped to Wails methods: 195
- UI/event-only controls: 207
- Delegated DataTable row actions: 22
- Handler-present but runtime-only/no direct Wails call detected: 68
- Form-submit actions: 5
- Unknown/static-only classification: 6
- Suspicious/unwired actions after filtering delegated actions: 12

## Verification Commands

```bash
node scripts/button_inventory.mjs
node scripts/button_backend_audit.mjs
node --check scripts/button_backend_audit.mjs
npm run build
GOCACHE=/tmp/asymmflow-gocache go build ./...
GOCACHE=/tmp/asymmflow-gocache go test ./...
```

## Verification Results

- `node scripts/button_inventory.mjs`: Pass. Generated markdown and JSON inventory.
- `node scripts/button_backend_audit.mjs`: Pass. Generated backend classification report.
- `node --check scripts/button_backend_audit.mjs`: Pass.
- `npm run build` from `frontend/`: Pass. Svelte/Vite production build completed.
- `go build ./...`: Pass. Backend compiles.
- `go test ./...`: Fail. First root-package failure is `TestDeploymentDBCopyReconciliationAndPackaging`: expected `471`, actual `470` at `deployment_audit_test.go:266`.

## What This Proves

- The app frontend currently compiles with the generated Wails bindings.
- The backend currently compiles.
- 195 screen-level actions have a visible path from button handler to Wails backend method.
- No missing backend function was found in the reviewed high-risk sales/order slice: customer orders, delivery notes, invoices, offers, and costing.

## What This Does Not Prove Yet

- It does not prove all 515 actions work after a real click.
- It does not prove runtime-only controls such as tabs, modal open/close, row selection, drag/drop, or router events behave correctly in the built app.
- It does not prove DataTable HTML action buttons fire correctly in the app, although delegated handlers exist for the major DataTable screens inspected.
- It does not prove modals are client-ready; that is the next explicit audit phase.

## Confirmed / Likely UI Issues

### UI-001 - Accounting Report Generate Buttons Are Inert

File: `frontend/src/lib/screens/AccountingScreen.svelte`

The three `Generate` buttons under the accounting report cards have no click handlers:

- Profit & Loss: line 462
- Balance Sheet: line 467
- VAT Return: line 472

The screen also appears mock-first, with real backend loading not implemented in the inspected branch.

### UI-002 - User Management Action Buttons Are Inert

File: `frontend/src/lib/screens/UserManagementScreen.svelte`

These controls have no handlers:

- `+ Add User`: line 113
- row `Edit`: line 186
- `View Permissions`: line 199

`CreateUser` is imported in the screen but is not called.

### UI-003 - Deployment Reconciliation Test Count Mismatch

File: `deployment_audit_test.go`

`go test ./...` fails at `TestDeploymentDBCopyReconciliationAndPackaging`, where expected count is `471` and actual is `470`.

### UI-004 - Create PO From Customer Order Needs Mixed-Supplier Stress Test

File: `frontend/src/lib/screens/OrdersScreen.svelte`

The `Create PO` action calls `CreatePOFromOrder(order.id, '', itemIDs)`. The backend can infer supplier details, but this needs a deliberate mixed-supplier order test because the frontend passes an empty supplier ID.

## Not Logged As Issues

- `ShowcaseScreen` buttons are design-system examples and intentionally have no business action.
- `WorkHub` employee chips are draggable controls, not click buttons.
- DataTable action buttons in Offers, Purchase Orders, Supplier Invoices, RFQ, Suppliers, and GRN render as HTML strings with `data-action`; those are handled by delegated row/table click logic and require runtime verification rather than immediate bug logging.

## Recommended Next Backend Pass

1. Build a disposable SQLite test DB for click-action workflows.
2. Start with Customer Orders:
   - Create PO
   - Create Delivery Note
   - Mark Delivered
   - Create Invoice
   - Create Proforma
   - Edit Order
   - Delete Order
3. Then test Delivery Notes and Invoices from the generated objects.
4. Mark `Backend Test Status` in `UI_BUTTON_INVENTORY_2026_04_17.md`.
5. Add every mismatch to `UI_ISSUE_LOG_2026_04_17.md`.
