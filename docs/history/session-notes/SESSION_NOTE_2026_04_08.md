# Session Note — April 8, 2026

This note captures the application, data, and workflow changes completed in this session for AsymmFlow / Acme Instrumentation ERP.

## Main Outcomes

### Work Hub
- Reworked the `Work` experience around compact task cards and a task detail modal instead of the old long-scroll layout.
- Improved the Team Board for admin visibility into active work, blocked work, and employee assignment.
- Added project multi-member behavior so projects can hold several employees at once.
- Added richer assignee notifications and made notification access broadly available to signed-in users.
- Added blocker-reason support for blocked tasks.
- Added task CRUD support:
  - edit title
  - edit description
  - edit priority
  - unblock task
  - delete task
- Updated the task modal to use the modern button system and an in-app delete confirmation flow.
- Added `tasks:delete` to seeded non-admin role permissions so task delete is not silently blocked by RBAC.

### Finance / Expenses / Payroll
- Merged payout tracking into the Payroll workspace instead of keeping it as a separate Finance tab.
- Merged recurring expense management into the Expenses workspace.
- Merged compensation and payroll runs into a more unified payroll flow.
- Removed Supplier Invoices from Finance because that workflow already exists in Operations.
- Changed finance KPI language and calculations toward YTD framing where requested.
- Enforced better expense payment behavior:
  - posted expenses do not become paid without payment details
  - expense payment requires method, reference, and bank account
  - paid expenses flow into `Payments Made`
- Improved payroll UX:
  - reduced field overlap on the compensation form
  - added clearer structure for salary/payment logging
  - added bank account + paid date + payment reference capture before logging salary payment
  - clarified what “Employer Cost” means in the UI

### Opportunities / Offers / Orders / Invoices
- Continued improving seeded 2026 commercial data from the shared drive (`C:\Data\Acme Instrumentation`).
- Repaired won offers that were missing line items by backfilling from verified opportunity product details.
- Narrowed no-item warnings on offers so they focus on truly actionable missing-item cases.
- Fixed duplicate and zero-value legacy customer order records in list behavior.
- Improved order modal actions:
  - create PO
  - create delivery note
  - create invoice
  - create proforma
  - edit order
  - delete order
- Order actions now route users into the right destination workspace so the result is visible immediately.
- Improved order-line sanitization to handle duplicate imported rows and derive missing unit prices from totals.
- Strengthened invoice creation from orders:
  - normalize messy imported order lines
  - derive unit prices from totals when needed
  - carry costing metadata through to invoice items
- Improved invoice detail presentation:
  - richer line-item descriptions
  - stronger linkage to offer / RFQ / costing metadata
  - customer reference, attention, origin, delivery, and terms displayed in the invoice detail modal
- Backfilled existing invoices that had no `invoice_items` rows but were linked to orders.

### Customer / Relationship Data
- Backfilled business-facing `customer_id` values for customers that had ended up with UUIDs in the visible customer ID field.
- Updated startup to keep that customer ID repair in place.

### Licensing / App Package / Runtime
- Continued refreshing the built app into `/Applications/AsymmFlow.app` after verified changes.
- Prior path and runtime DB alignment work remained part of this session’s continuity so the live app kept using the intended database copies.

## Data / DB Repairs Completed

### Live Invoice Repair
- Added a targeted backfill path for invoices linked to orders but missing invoice items.
- Repaired known broken invoice shells in both:
  - repo DB: `/Users/developer/House_of_Projects/ph_holdings/ph_holdings.db`
  - runtime DB: `/Users/developer/.local/share/AsymmFlow/ph_holdings.db`

Verified repaired examples:
- `9917`
- `INV-2026-0610`

### Won Offer Repair
- Verified won offers with missing items were repaired from verified opportunity product details.

### Customer ID Repair
- Verified customer rows using UUID-like visible IDs were backfilled to business-facing IDs.

## Verification Completed

- `go test ./...`
- `npm run build`
- `wails build`
- targeted manual repair runners for:
  - won-offer item backfill
  - invoice-item backfill

## Follow-Up Notes

### Deployment Check — Red Team Pass
Run a full red-team pass across the application workflows before deployment.

This should include at minimum:
- Opportunities: RFQ → Costing → Offer → Won/Lost → Customer Order
- Operations: PO → GRN → Delivery Note → Delivery Tracking
- Finance: Customer Invoice → Payments Received → Expenses → Payments Made → Payroll
- Work / People / Notifications: task creation, assignment, blocker handling, comments, notifications, RBAC visibility
- Relationships / CRM views
- Deployment page, support actions, and rollout checklist
- License / startup / runtime DB path behavior

Goal:
- verify all buttons
- verify navigation handoffs
- verify modal actions
- verify record creation and downstream propagation
- verify RBAC behavior role by role
- catch broken but visible UI actions before rollout

### Database Schema Review
Run a schema and workflow coverage review to verify the database fully encompasses:
- all live workflows
- all user-facing data capture points
- all hidden operational side effects
- all approval / posting / payment state changes
- all traceability links between pipeline, operations, finance, and collaboration

Specific check:
- every data entry point in the UI should map cleanly to durable storage
- every downstream document flow should have supporting child tables and linkage fields
- every workflow state transition should be represented in schema and query surfaces
- no new screen should rely on transient frontend-only state for business-critical data

### Recommended Next Session Priorities
1. Run the red-team workflow pass end to end and capture findings in a dedicated audit note.
2. Run the schema/data-entry coverage review and list any missing tables, columns, or relationships.
3. Continue refining finance document richness so invoice and payroll records feel fully operational, not just technically present.
4. Continue validating seeded/imported historical data against source documents wherever the UI still feels suspicious.

