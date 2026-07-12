# Deployment & Red-Team Checklist

Date: April 8, 2026
Project: AsymmFlow / Acme Instrumentation ERP
Purpose: Final pre-deployment workflow validation, adversarial testing, and schema coverage review.

## How To Use This Document

- Run this checklist on the actual deployment candidate build.
- Prefer testing against a copy of the real seeded database, not only a blank/dev DB.
- Record pass/fail, screenshots, and exact repro steps for anything unstable.
- Treat any silent failure, wrong navigation, missing child rows, stale KPIs, or RBAC leak as a release blocker until triaged.

---

## 1. Deployment Readiness Gate

### Build & Runtime
- Confirm `go test ./...` passes.
- Confirm `npm run build` passes.
- Confirm `wails build` passes.
- Confirm `/Applications/AsymmFlow.app` launches successfully.
- Confirm the app opens the intended runtime DB and does not create a fresh unintended DB.
- Confirm the app does not re-prompt for license on the same Mac unexpectedly.
- Confirm startup completes without blocking the UI indefinitely.
- Confirm seeded DB loads without schema errors or missing-column errors.

### DB & Package Integrity
- Confirm runtime DB path is the intended one.
- Confirm repo DB, runtime DB, and packaged app DB copies are understood and not accidentally diverging.
- Confirm migrations/backfills do not corrupt seeded data on startup.
- Confirm the app can reopen cleanly after force quit and normal quit.

---

## 2. Core Workflow Checks

### Opportunities Workflow
- Create RFQ.
- Create costing sheet from RFQ.
- Add multiple line items.
- Verify pricing, totals, VAT, and margin calculations.
- Save as offer.
- Open offer and verify line items carried through.
- Mark offer won.
- Confirm order is created with correct totals and line items.
- Confirm opportunity stage updates correctly.
- Confirm traceability chain is preserved.

### Orders Workflow
- Open order modal from list.
- Verify all modal buttons work:
  - `Create PO`
  - `Create Delivery Note`
  - `Mark as Delivered`
  - `Create Invoice`
  - `Proforma`
  - `Create Task`
  - `Edit Order`
  - `Delete Order`
- Confirm each action routes to the correct downstream workspace when applicable.
- Confirm zero-value and duplicate orders are not shown as active operational records unless intentional.
- Confirm order items match commercial totals.

### Invoice Workflow
- Create invoice from Finance Hub.
- Create invoice from Order modal.
- Confirm invoice items are created in DB.
- Confirm invoice detail modal shows:
  - line items
  - offer/order traceability
  - customer reference
  - attention/contact fields
  - country of origin
  - delivery weeks/terms
  - payment terms
- Generate PDF and inspect layout/content.
- Confirm invoice appears in Customer Invoices immediately.
- Confirm sent/edit/delete actions work as intended.

### Operations Workflow
- Create supplier PO from order.
- Create GRN from PO.
- Create delivery note from order.
- Mark delivery/dispatched/delivered.
- Confirm serial and quantity tracking behaves correctly.
- Confirm downstream invoiceability reflects delivery/invoicing state.

### Expenses Workflow
- Create draft expense.
- Submit/approve/post expense.
- Confirm posting does not mark it paid.
- Pay expense with:
  - payment method
  - payment reference
  - bank account
  - paid date
- Confirm paid expense appears in `Payments Made`.
- Confirm recurring expense flow works inside Expenses.

### Payroll Workflow
- Create compensation profile.
- Verify compensation form layout does not overlap.
- Generate payroll period.
- Generate payroll run.
- Approve payroll run.
- Post payroll run.
- Mark payroll run paid with:
  - paid date
  - bank account
  - payment reference
- Confirm payout trail appears inside Payroll.
- Confirm employer cost appears consistently in profile and run summaries.

### Work / Collaboration Workflow
- Create task.
- Assign task.
- Reassign task.
- Add comments.
- Change due date.
- Block task with blocker reason.
- Unblock task.
- Start task.
- Complete task.
- Edit task details.
- Delete task.
- Confirm task modal buttons all work.
- Confirm project can have multiple team members.
- Confirm same employee can hold multiple tasks.
- Confirm notifications are created for assignees.

### Notifications Workflow
- Confirm notifications visible for all intended roles.
- Confirm notification feed shows:
  - sender
  - task/event context
  - date grouping / historical trail
- Confirm `Open task` or equivalent action routes correctly.
- Confirm read/unread state updates.

### CRM / Relationships Workflow
- Open customer detail page.
- Confirm customer business ID is business-facing, not UUID-like.
- Confirm orders/invoices/RFQs/contacts render correctly.
- Confirm no obvious master-data mismatches remain for seeded customers.

### Deployment Page Workflow
- Open readiness audit.
- Open rollout checklist.
- Open support/export actions.
- Confirm no broken buttons or nil states.
- Confirm deployment page copy and actions are understandable to operators.

---

## 3. Adversarial Red-Team Pass

This section is mandatory before deployment.

### Navigation Adversarial Tests
- Click the same action button rapidly 3-5 times and verify no duplicate records are created.
- Open and close modals repeatedly and verify state does not corrupt.
- Trigger cross-screen navigation from orders, notifications, and finance repeatedly and confirm target tabs stay correct.
- Refresh/reopen the app after partially completed actions and confirm state is durable.

### Data Integrity Adversarial Tests
- Create records with minimal required data only and confirm they still behave correctly.
- Create records with long descriptions, long references, and many line items.
- Use zero/blank optional fields and confirm layout does not break.
- Use duplicate-looking commercial rows and verify totals do not double-count.
- Verify no invoice/order/offer shows hollow child detail when the parent looks complete.
- Verify subtotal, VAT, and grand total reconcile for commercial documents.
- Verify line-item sums do not exceed or fall below document totals unexpectedly.

### RBAC Adversarial Tests
- Log in or switch context for each role:
  - Admin
  - Manager
  - Sales
  - Operations
  - Staff
- Confirm each role can only see intended screens/tabs/actions.
- Confirm forbidden actions fail safely with a message, not silent breakage.
- Confirm new Work / Notifications / Payroll changes respect RBAC.
- Confirm delete/update actions are not accidentally exposed to the wrong roles.

### Concurrency / Re-entry Tests
- Click create/send/post/pay actions twice quickly.
- Open the same record from two different screens if possible and edit sequentially.
- Re-run startup after a prior backfill/migration and confirm idempotence.
- Confirm repeated app launches do not duplicate seed or backfill records.

### Database Stress / Consistency Tests
- Verify no missing child rows for recently created:
  - offers
  - orders
  - invoices
  - payroll runs
  - expense payments
  - tasks/comments/activity
- Verify soft-deleted duplicates are not still showing in active lists.
- Verify list views are not pulling legacy placeholder shells as live operational records.

### UX Failure Tests
- Confirm button disabled/loading states are visible during long actions.
- Confirm failed actions show meaningful error messages.
- Confirm no “nothing happened” buttons remain.
- Confirm no clipped fields, overlapping controls, or hidden action labels remain on major screens.

---

## 4. Database Schema Coverage Review

Goal: confirm the schema actually covers all workflows and every data entry point.

### Master Review Questions
- Does every user-entered field in the UI persist to the database?
- Does every workflow state transition have a durable DB representation?
- Does every downstream document flow preserve its source traceability?
- Are there any business-critical fields living only in frontend state?
- Are any screens depending on legacy tables while new flows use different tables?

### Coverage Areas To Audit

#### Sales / Commercial
- RFQs
- Costing sheets
- Costing line items
- Offers
- Offer items
- Opportunities
- Orders
- Order items
- Cross-links between all of the above

#### Finance
- Invoices
- Invoice items
- Payments received
- Expense entries
- Recurring expense schedules
- Supplier payments / payments made
- Journal entries / journal lines
- Bank account linkage
- VAT and posting fields

#### Payroll
- Compensation profiles
- Payroll periods
- Payroll runs
- Payroll run items
- Payroll components
- Payroll payouts
- Bank/payment reference linkage

#### Collaboration
- Projects
- Project members
- Tasks
- Task comments
- Task activity
- Notifications
- Notification receipts
- Any legacy task/follow-up tables still surfaced anywhere

#### CRM / Master Data
- Customers
- Customer contacts
- Customer-facing business IDs
- Suppliers
- Supplier contacts
- Products

#### Deployment / Support
- Readiness rows
- Checklist records
- Support/export artifacts
- Any deployment-only audit/support fields

### Specific Schema Review Actions
- Compare every major screen form against its backing DB struct/table.
- Compare every modal action against the backend method and persisted side effects.
- List any fields shown in UI but not saved.
- List any DB columns that are no longer used by real workflows.
- Identify duplicate workflow tables or stale legacy tables that can confuse future changes.
- Confirm indexes exist for major listing and lookup paths.

---

## 5. Data Audit Checks

### Commercial Data
- Check for orders with zero total.
- Check for offers with no line items.
- Check for invoices with no invoice items.
- Check for duplicate order/item rows created by old imports.
- Spot-check imported 2025/2026 commercial records against source files.

### Finance Data
- Check that paid expenses appear in `Payments Made`.
- Check that posted-but-unpaid expenses remain unpaid until payment details are entered.
- Check that payroll paid status corresponds to payout records.
- Check that invoice and payment summaries are YTD where intended.

### Collaboration Data
- Check that task create/update/delete status changes persist.
- Check that comments and activity log correctly.
- Check that blocker reason persists and clears correctly.
- Check that notifications are generated and readable.

---

## 6. Release Blockers

Do not deploy if any of the following remain unresolved:

- Any major workflow button does nothing or silently fails.
- Any invoice/order/offer/payroll record can be created without its required child detail.
- Any RBAC leak exposes finance/admin actions to the wrong role.
- Any startup/license/runtime DB path issue can strand the user on launch.
- Any list is still showing placeholder, duplicate, or zero-value records as active truth.
- Any seeded/imported commercial totals materially disagree with source documents.

---

## 7. Final Sign-Off

Before deployment, confirm all of the below:
- Build candidate tested
- Runtime DB verified
- Red-team pass completed
- Schema coverage review completed
- Data audit completed
- Release blockers resolved
- Deployment package refreshed
- Installed app copy refreshed

Sign-off fields:
- Tester:
- Date:
- Build path:
- Runtime DB path:
- Result:
- Outstanding risks:

