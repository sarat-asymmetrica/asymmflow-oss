# UI Issue Log - 2026-04-17

Purpose: append-only tracker for every issue found while testing page buttons, backend actions, and modal workflows.

## Status Values

- `Open`
- `Investigating`
- `Fixed`
- `Verified`
- `Deferred`
- `Duplicate`
- `Won't Fix`

## Severity Values

- `P0 - Crash/Data Loss/Security`
- `P1 - Broken Critical Workflow`
- `P2 - Incorrect Business Output`
- `P3 - UX/Copy/Polish`

## Issue Template

Copy this block for each new issue.

```markdown
## UI-000 - Short Title

- Status: Open
- Severity: P2 - Incorrect Business Output
- Screen/Page:
- Button/Action:
- Modal/Popup:
- Role/User:
- Backend Test Result:
- App Test Result:
- Expected:
- Actual:
- Reproduction Steps:
- Data Used:
- Files/Methods Suspected:
- Notes:
- Fixed In:
- Verified By:
```

## Issues

## UI-001 - Accounting Report Generate Buttons Are Inert

- Status: Open
- Severity: P2 - Incorrect Business Output
- Screen/Page: Accounting
- Button/Action: Profit & Loss `Generate`, Balance Sheet `Generate`, VAT Return `Generate`
- Modal/Popup: None
- Role/User: Finance/Admin
- Backend Test Result: Not backend-testable from current UI wiring
- App Test Result: Untested
- Expected: Each button should call a report/export backend method or open a report configuration modal.
- Actual: Buttons have no `on:click` handler.
- Reproduction Steps: Open Accounting screen and inspect report card buttons.
- Data Used: Static source audit
- Files/Methods Suspected: `frontend/src/lib/screens/AccountingScreen.svelte:462`, `frontend/src/lib/screens/AccountingScreen.svelte:467`, `frontend/src/lib/screens/AccountingScreen.svelte:472`
- Notes: Screen appears mock-first; real backend loading branch is not implemented in the inspected code.
- Fixed In:
- Verified By:

## UI-002 - User Management Buttons Are Inert

- Status: Open
- Severity: P1 - Broken Critical Workflow
- Screen/Page: User Management
- Button/Action: `+ Add User`, row `Edit`, `View Permissions`
- Modal/Popup: Expected user/permission modals are absent or unwired
- Role/User: Admin
- Backend Test Result: Not backend-testable from current UI wiring
- App Test Result: Untested
- Expected: Add User should open/create a user workflow; Edit should edit selected user; View Permissions should show role permissions.
- Actual: Buttons have no `on:click` handler.
- Reproduction Steps: Open User Management screen and inspect/click the listed buttons.
- Data Used: Static source audit
- Files/Methods Suspected: `frontend/src/lib/screens/UserManagementScreen.svelte:113`, `frontend/src/lib/screens/UserManagementScreen.svelte:186`, `frontend/src/lib/screens/UserManagementScreen.svelte:199`
- Notes: `CreateUser` is imported but not called.
- Fixed In:
- Verified By:

## UI-003 - Root Go Test Fails On Deployment Reconciliation Count

- Status: Open
- Severity: P2 - Incorrect Business Output
- Screen/Page: Deployment / backend deployment audit
- Button/Action: Backend verification suite
- Modal/Popup: None
- Role/User: Admin/Deployment
- Backend Test Result: Fail
- App Test Result: N/A - backend only
- Expected: `go test ./...` should pass.
- Actual: `TestDeploymentDBCopyReconciliationAndPackaging` expected `471`, actual `470`.
- Reproduction Steps: Run `GOCACHE=/tmp/asymmflow-gocache go test ./...`.
- Data Used: Local test DB/runtime state
- Files/Methods Suspected: `deployment_audit_test.go:266`
- Notes: `go build ./...` passes; failure appears to be a reconciliation/count assertion rather than a compile failure.
- Fixed In:
- Verified By:

## UI-004 - Create PO From Customer Order Needs Mixed-Supplier Stress Test

- Status: Investigating
- Severity: P2 - Incorrect Business Output
- Screen/Page: Opportunities > Customer Orders
- Button/Action: `Create PO`
- Modal/Popup: Order detail modal
- Role/User: Sales/Admin/Ops
- Backend Test Result: Untested
- App Test Result: Untested
- Expected: PO creation from an order should correctly group/select suppliers for all selected order items.
- Actual: Frontend calls `CreatePOFromOrder(order.id, '', itemIDs)` with an empty supplier ID.
- Reproduction Steps: Create or find a customer order with items from mixed suppliers; click `Create PO`; inspect generated PO(s).
- Data Used: Static source audit
- Files/Methods Suspected: `frontend/src/lib/screens/OrdersScreen.svelte:728`
- Notes: Backend may infer supplier correctly for single-supplier orders; mixed-supplier behavior must be tested.
- Fixed In:
- Verified By:

## UI-005 - New Order Modal Did Not Persist Line Items

- Status: Fixed
- Severity: P1 - Broken Critical Workflow
- Screen/Page: Opportunities > Customer Orders
- Button/Action: `+ New Order` / `Create Order`
- Modal/Popup: New Order modal
- Role/User: Sales/Admin
- Backend Test Result: `npm run build` passed; `go build ./...` passed
- App Test Result: Pending manual app test
- Expected: Creating an order should save the header and all order line items so delivery notes, invoices, and POs can be generated from real items.
- Actual: The create modal hid the line-item grid and called `CreateOrder(...)`, which only persisted the order header.
- Reproduction Steps: Open Customer Orders, create a new order, then reopen it and inspect line items.
- Data Used: Static modal audit
- Files/Methods Suspected: `frontend/src/lib/screens/OrdersScreen.svelte`, `CreateOrder`, `UpdateOrder`
- Notes: Fixed by showing the line-item grid for create mode, requiring at least one priced item, computing total from rows, and saving the created order through `UpdateOrder` with items.
- Fixed In: `frontend/src/lib/screens/OrdersScreen.svelte`
- Verified By: Codex build verification

## UI-006 - Accounting Voucher And Account Modals Were Frontend-Only

- Status: Fixed
- Severity: P1 - Broken Critical Workflow
- Screen/Page: Finance > Accounting
- Button/Action: `+ New Entry`, `+ New Account`, account `Edit`
- Modal/Popup: New Accounting Entry, Add/Edit Account
- Role/User: Finance/Admin
- Backend Test Result: `npm run build` passed; `go build ./...` passed
- App Test Result: Pending manual app test
- Expected: Accounting modal actions should persist to the real chart of accounts and journal tables.
- Actual: `createVoucher()` and `saveAccount()` only mutated local Svelte arrays; data vanished on reload.
- Reproduction Steps: Open Accounting, create a voucher or account, reload the screen/app, and inspect persistence.
- Data Used: Static modal audit
- Files/Methods Suspected: `frontend/src/lib/screens/AccountingScreen.svelte`, `CreateJournalEntry`, `CreateAccount`, `UpdateAccount`
- Notes: Fixed by wiring backend load/create/update paths. Also corrected `UpdateAccount` to accept string UUID account IDs.
- Fixed In: `frontend/src/lib/screens/AccountingScreen.svelte`, `app.go`, `frontend/wailsjs/go/main/App.d.ts`
- Verified By: Codex build verification

## UI-007 - Supplier Payment Edit Modal Could Not Correct Exchange Rate

- Status: Fixed
- Severity: P2 - Incorrect Business Output
- Screen/Page: Finance > Payments Made / Supplier Payments
- Button/Action: Edit selected supplier payment
- Modal/Popup: Edit Supplier Payment
- Role/User: Finance/Admin
- Backend Test Result: `npm run build` passed
- App Test Result: Pending manual app test
- Expected: Foreign-currency supplier payment edits should expose and persist the exchange rate used to calculate BHD value.
- Actual: The edit modal backfilled the old exchange rate silently, so a bad rate could not be corrected from the UI.
- Reproduction Steps: Select a non-BHD supplier payment, open edit, inspect available fields.
- Data Used: Static modal audit
- Files/Methods Suspected: `frontend/src/lib/screens/SupplierPaymentsScreen.svelte`, `UpdateSupplierPayment`
- Notes: Fixed by adding an exchange-rate field in edit mode for non-BHD payments and validating it before update.
- Fixed In: `frontend/src/lib/screens/SupplierPaymentsScreen.svelte`
- Verified By: Codex build verification

## UI-008 - Reports Export Modal Does Not Honor Date Range

- Status: Open
- Severity: P2 - Incorrect Business Output
- Screen/Page: Reports
- Button/Action: `Export Report`
- Modal/Popup: Export Report
- Role/User: Admin/Manager
- Backend Test Result: Static source audit only
- App Test Result: Untested
- Expected: Export should use the visible selected start/end dates or clearly expose the date bucket being exported.
- Actual: `loadReportData()` hardcodes `month`, and `ExportReport` receives only category, format, and current data JSON. PDF/Excel options are also shown even though backend returns not implemented for those formats.
- Reproduction Steps: Change the date range, refresh/export, and compare backend request parameters/export contents.
- Data Used: Static modal audit
- Files/Methods Suspected: `frontend/src/lib/screens/ReportsScreen.svelte`, `reports.go:GetReportData`, `reports.go:ExportReport`
- Notes:
- Fixed In:
- Verified By:

## UI-009 - Delivery Note Modal Allows Manual Workflow Status Selection

- Status: Open
- Severity: P2 - Incorrect Business Output
- Screen/Page: Operations > Delivery Notes
- Button/Action: `+ New Delivery Note`, `Edit`
- Modal/Popup: New/Edit Delivery Note
- Role/User: Operations/Admin
- Backend Test Result: Static source audit only
- App Test Result: Untested
- Expected: DN status should move through guarded actions such as dispatch and confirm delivery.
- Actual: The create/edit modal exposes a free status dropdown, allowing users to save later workflow states directly.
- Reproduction Steps: Create or edit a DN and inspect the Status dropdown.
- Data Used: Static modal audit
- Files/Methods Suspected: `frontend/src/lib/screens/DeliveryNotesScreen.svelte`, `CreateDeliveryNote`, `UpdateDeliveryNote`, `DispatchDeliveryNote`, `ConfirmDeliveryNote`
- Notes: The DN modal otherwise has the right fields and backend wiring.
- Fixed In:
- Verified By:

## UI-010 - Purchase Order And Supplier Invoice Delete Backends Are Unreachable

- Status: Open
- Severity: P3 - UX/Copy/Polish
- Screen/Page: Operations > Purchase Orders; Finance > Supplier Invoices
- Button/Action: Delete PO / Delete Supplier Invoice
- Modal/Popup: Expected delete confirmation modal is absent
- Role/User: Admin/Ops/Finance
- Backend Test Result: Static source audit only
- App Test Result: Untested
- Expected: If delete is supported, the UI should expose a guarded confirmation flow; if not supported, imports should be removed and policy made explicit.
- Actual: `DeletePurchaseOrder` and `DeleteSupplierInvoice` are imported but not used by the modal flows.
- Reproduction Steps: Open PO or Supplier Invoice detail/edit modals and look for delete actions.
- Data Used: Static modal audit
- Files/Methods Suspected: `frontend/src/lib/screens/PurchaseOrdersScreen.svelte`, `frontend/src/lib/screens/SupplierInvoicesScreen.svelte`
- Notes:
- Fixed In:
- Verified By:
