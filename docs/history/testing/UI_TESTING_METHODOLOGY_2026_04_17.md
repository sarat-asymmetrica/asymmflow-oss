# UI Testing Methodology - 2026-04-17

Purpose: make every page action testable through two independent paths: backend-first verification by Codex, then manual app verification by the client. When both agree, we can trust the workflow. When they disagree, the mismatch becomes an issue.

## Working Documents

- Button inventory: `docs/testing/UI_BUTTON_INVENTORY_2026_04_17.md`
- Issue log: `docs/testing/UI_ISSUE_LOG_2026_04_17.md`
- Extractor script: `scripts/button_inventory.mjs`

Regenerate the inventory after frontend edits:

```bash
node scripts/button_inventory.mjs
```

## Scope

The source currently has 58 Svelte screen files, not exactly 52. Some are shell-mounted pages, some are hub tabs, some are entry/admin states, and some are inactive or older page-like screens. The audit keeps all of them in scope until we explicitly decide to remove/deprecate a screen.

Primary navigation surfaces:

- Dashboard
- Opportunities: RFQs, Costing, Offers, Customer Orders
- Operations: Supplier POs, Supplier Invoices, Delivery Notes
- Finance: Dashboard, Customer Invoices, Payments Received, Payments Made, Expenses, Approvals, Payroll, Bank Recon
- Relationships: Customers, Suppliers, Customer Detail, Supplier Detail
- Intelligence: Butler, Archaeologist, Entity Discovery
- Work, People, Notifications, Settings, Deployment, User Management
- Entry states: Login, License Activation, Pending Approval, Setup/Admin screens

## Test Phases

### Phase 1 - Static Inventory

Goal: know every visible action before clicking anything.

Steps:

1. Generate `UI_BUTTON_INVENTORY_2026_04_17.md`.
2. Review each screen section for obvious duplicate, dead, or unclear buttons.
3. Mark pure UI controls separately from backend-backed actions.
4. Add missing runtime-only buttons discovered manually.

Static extraction captures literal `<button>`, design-system `<Button>`, `<WabiButton>`, and selected `role="button"` actions. It cannot fully resolve DataTable row renderers, dynamic Butler actions, permission-gated buttons, or modal-only buttons without runtime testing.

### Phase 2 - Backend-First Verification

Goal: prove the underlying method works independently of the UI.

For each backend-backed button:

1. Identify the click handler in the inventory.
2. Map handler to Wails method(s) imported from `frontend/wailsjs/go/main/App`.
3. Run the backend method against a disposable DB copy when the action creates, updates, deletes, exports, approves, or posts data.
4. Record result in `Backend Test Status`:
   - `Pass`
   - `Fail`
   - `Blocked`
   - `N/A - UI only`
5. If failed or suspicious, create an issue in `UI_ISSUE_LOG_2026_04_17.md`.

Safety rule: destructive or financial tests must use a copied SQLite database unless the user explicitly approves touching live data.

### Phase 3 - Manual App Verification

Goal: verify the actual user experience matches the backend result.

For each button:

1. Open the page in the built app.
2. Click the button as the intended role/user.
3. Confirm the expected toast, modal, file, state change, navigation, or persisted database result.
4. Record result in `App Test Status`:
   - `Pass`
   - `Fail`
   - `Blocked`
   - `Not visible`
   - `N/A - backend only`
5. Link any issue ID from the issue log.

### Phase 4 - Modal And Popup Audit

Goal: make every popup make sense for the client.

For each button that opens a modal:

1. Capture modal title and purpose.
2. List fields, defaults, required fields, validation, and footer actions.
3. Check whether it has a clear cancel/close path.
4. Check whether the save/submit button calls a backend method and handles errors honestly.
5. Log confusing copy, missing fields, invalid defaults, broken validation, or risky destructive flows.

### Phase 5 - Reconciliation

Goal: identify mismatches between backend and app behavior.

Mismatch examples:

- Backend passes, app fails: frontend wiring, stale state, validation, permissions, or modal data problem.
- Backend fails, app appears to pass: UI may be showing optimistic success without persistence.
- Backend and app both fail: service/data/model bug.
- Button is present but not useful: UX/design issue, dead action, or deprecated workflow.

## Status Conventions

Use these exact values in the inventory:

- `Untested`
- `Pass`
- `Fail`
- `Blocked`
- `N/A - UI only`
- `N/A - backend only`
- `Not visible`

Use issue IDs like:

- `UI-001`
- `UI-002`
- `UI-003`

## Backend Mapping Notes

Generated bindings live in:

- `frontend/wailsjs/go/main/App.d.ts`
- `frontend/wailsjs/go/main/App.js`
- `frontend/wailsjs/go/models.ts`

Important domains to prioritize first:

- Customer Orders: create PO, create DN, mark delivered, create invoice, proforma, edit, delete
- Delivery Notes: create, edit, dispatch, confirm, PDF
- Customer Invoices: create, send, delete, PDF, credit notes
- Offers/Costing: save offer, edit offer, won/lost, PDF, Excel export
- Finance: expenses, payments, bank recon, dashboard year/company selection
- Intelligence: chat, report generation, workflow action buttons

## Automation Path

The existing Playwright helper `frontend/tests/e2e/helpers/mockWailsBridge.ts` can be extended to intercept `window.go.main.App` calls. That gives us a safe UI-level test where button clicks assert the intended Wails method and arguments without touching the real DB.

For persistence-sensitive workflows, use a copied SQLite DB and compare before/after rows.
