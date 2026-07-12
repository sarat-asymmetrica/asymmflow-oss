# AsymmFlow User Manual Pack

Last updated: 2026-04-25

This documentation pack is written from the current Wails, Go, and Svelte code in this repository. It is intended for Acme Instrumentation / Beacon Controls users, managers, auditors, and support staff.

## Manuals In This Pack

| Document | Audience | Purpose |
|---|---|---|
| `ADMIN_USER_GUIDE.md` | Administrator / developer support | Activation, users, settings, sync, backups, deployment, master controls |
| `MANAGER_USER_GUIDE.md` | Manager / finance leadership | Sales, operations, finance, approvals, reports, reconciliation |
| `SALES_USER_GUIDE.md` | Sales engineers and sales support | RFQ, costing, quotations, customer orders, customer notes, Butler |
| `OPERATIONS_USER_GUIDE.md` | Procurement, logistics, delivery team | Supplier orders, supplier invoices, delivery notes, order fulfillment |
| `STAFF_USER_GUIDE.md` | General staff | Dashboard, work board, notifications, basic collaboration |
| `FIELD_AND_WORKFLOW_REFERENCE.md` | All users and trainers | What to enter in each major field and what each action does |

## Main Application Areas

| Area | Navigation Label | Main Screens | Typical Owners |
|---|---|---|---|
| Executive cockpit | Dashboard | Dashboard cards, alerts, cash/revenue/task summaries | All roles |
| Sales flow | Opportunities | RFQs, Costing, Offers, Customer Orders | Sales, Manager, Admin |
| Operations flow | Operations | Supplier Orders, Supplier Invoices, Delivery Notes | Operations, Manager, Admin |
| Finance flow | Finance | Dashboard, customer invoices, payments received, payments made, expenses, approvals, payroll, bank reconciliation | Manager, Admin |
| Collaboration | Work | My Work, Team Board, Projects | All roles |
| People | People | Employee directory, org, contributions, license access linking | Manager/Admin where enabled |
| Relationships | Relationships | Customers, suppliers, 360 degree detail views | Sales, Operations, Manager, Admin |
| Intelligence | Intelligence | Butler chat, action prompts, conversation history | Sales, Operations, Manager, Admin where licensed |
| Administration | Settings | Company settings, folder paths, AI keys, currency rates, imports, backup, sync, reports, rollout controls | Admin |

## Role Model

Production access is license-key driven. License prefixes map to roles:

| Prefix | Role | Summary |
|---|---|---|
| `PH-ADM-*` | Admin | Full system access |
| `PH-MGR-*` | Manager | Commercial, operations, finance, HR/work visibility, approvals |
| `PH-SLS-*` | Sales | RFQ, costing, offers, orders, customers, supplier lookup, limited PO creation |
| `PH-OPS-*` | Operations | Supplier, PO, supplier invoice, delivery, order fulfillment, OCR |
| `PH-STF-*` | Staff | Dashboard, work, tasks, notifications |
| Developer master key (value redacted) | Developer/admin master key | Controlled developer support key when enabled |

If a user cannot see a screen, the screen is hidden by permission filtering rather than shown as disabled. That is expected behavior.

## Standard Workflow

The core trading workflow is:

1. Create or import an RFQ/opportunity.
2. Prepare costing against the opportunity.
3. Save the costing as an offer and export the quotation PDF.
4. Mark the offer won when the customer PO arrives.
5. Convert or create the customer order.
6. Create supplier orders as needed.
7. Receive and check supplier invoice data.
8. Create delivery note and dispatch/confirm delivery.
9. Create/send customer invoice.
10. Record customer payment and reconcile the bank statement.

## Source Anchors

The manual pack was created from these live code areas:

| Source | What it defines |
|---|---|
| `license_service.go` | License roles and role permissions |
| `frontend/src/App.svelte` | Screen routing and screen-level permissions |
| `frontend/src/lib/components/ui/EnterpriseSidebar.svelte` | Navigation visibility |
| `frontend/src/lib/screens/*.svelte` | User-facing fields, tabs, buttons, and modal workflows |
| `database.go` | Main data models and persisted fields |
| `customer_invoice_service.go`, `payment_service.go`, `supplier_payment_service.go` | Finance rules and transaction behavior |
| `purchase_order_service.go`, `delivery_note_service.go`, `supplier_invoice_service.go` | Procurement, delivery, matching, and status behavior |
| `business_invariants.go` | Business rules such as minimum margin, customer grade payment expectations, and Apex Engineering competition constraints |

