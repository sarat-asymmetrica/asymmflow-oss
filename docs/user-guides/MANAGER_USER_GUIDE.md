# Manager User Guide

Last updated: 2026-04-25

## Role Purpose

Managers use AsymmFlow to supervise the full commercial and operational cycle: pipeline, costing, offers, customer orders, supplier orders, finance, payments, expenses, payroll approvals, bank reconciliation, and team work.

Manager access is broad, but admin-only functions such as license generation, deep deployment controls, and destructive system maintenance should remain with Admin.

## Manager Home Routine

1. Open Dashboard and scan revenue, receivables, overdue tasks, and operational alerts.
2. Open Opportunities to review RFQs, costing readiness, offer stage, and won/lost pipeline.
3. Open Operations to review supplier orders, supplier invoices, and delivery notes.
4. Open Finance to review customer invoices, payments received, payments made, expenses, payroll, and bank reconciliation.
5. Open Work for assigned tasks, blocked items, and team board status.
6. Open Notifications for approvals, mentions, and pending actions.

## Sales Supervision

### RFQs And Opportunities

Use Opportunities -> RFQs.

Manager checks:

| Area | What to verify |
|---|---|
| Customer | Correct customer selected or typed |
| Project | Project name clearly identifies the enquiry |
| Value | Expected BHD value is realistic |
| Priority | High or urgent only for time-sensitive or strategic RFQs |
| Notes | Customer requirement, scope, exclusions, and next action are captured |
| Stage | Qualified, Proposal, Quoted, Won, or Lost is current |

Managers can use comments/tasks to push follow-up ownership to Sales.

### Costing Review

Use Opportunities -> Costing.

Manager checks:

| Field group | What to verify |
|---|---|
| Header | Customer, contact, RFQ reference, division, prepared by, quote type |
| Commercial terms | Payment terms, delivery terms, estimated delivery, country of origin |
| Line items | Equipment, model, quantity, currency, exchange rate, FOB, freight, margin |
| Risk controls | Minimum margin should be at least 8%; ABB competition requires stronger margin |
| Customer grade | Grade C should require at least 50% advance; Grade D should require 100% advance or decline |
| VAT | VAT rate should match the transaction type; exports may be 0% only when explicitly valid |
| Hidden charges | Internal profit adjustment only; should not appear on customer-facing exports |

Use export PDF/Excel for review packs. Use `Save as Offer` only when the costing is commercially ready.

### Offers

Use Opportunities -> Offers.

Manager actions:

| Action | Meaning |
|---|---|
| Generate PDF | Creates customer quotation PDF |
| Mark Won | Captures customer PO number and moves offer toward order flow |
| Mark Lost | Records loss reason for pipeline analysis |
| Requote | Starts a follow-up/revised quotation path |
| Follow-up | Schedules customer follow-up |
| Notes | Stores sales context on the offer |

When marking won, enter the exact customer PO reference. This reference flows into order/invoice traceability.

## Operations Supervision

Use Operations Hub.

| Tab | Manager focus |
|---|---|
| Supplier Orders | Ensure supplier, linked customer order, currency, exchange rate, terms, and items are correct |
| Supplier Invoices | Review 3-way match status, approve only after matching PO/GRN/invoice |
| Delivery Notes | Confirm dispatch/delivery status and partial delivery quantities |

Purchase orders over the configured 5,000 BHD threshold should go through approval before being sent.

## Finance Supervision

Use Finance Hub. Select company first: `Acme Instrumentation` or `Beacon Controls`.

| Tab | Manager use |
|---|---|
| Dashboard | Review P&L, balance sheet, ratios, and yearly trend |
| Customer Invoices | Create invoices from orders/DNs, send invoices, generate PDFs, credit notes |
| Payments Received | Record customer receipts against invoices |
| Payments Made | Record supplier payments against supplier invoices |
| Expenses | Create, submit, approve, post, and pay expense records |
| Approvals | Review expense items requiring approval |
| Payroll | Approve payroll runs and record payouts |
| Bank Recon | Import statements, auto-match, manually match, finalize reconciliation |

## Bank Reconciliation

Manager procedure:

1. Select the bank account.
2. Import statement or select an existing statement.
3. Click auto-match.
4. Review unmatched lines.
5. Manually match lines to customer payments, supplier payments, payroll payouts, expenses, journal entries, or split allocations.
6. Add missing statement lines only when the bank import is incomplete.
7. Finalize only when unmatched count is zero and opening/closing balances are correct.

## Approval Discipline

Managers should not approve:

| Item | Reason |
|---|---|
| Costing below 8% actual margin | Violates minimum commercial rule |
| Grade C without sufficient advance | Payment-risk violation |
| Grade D without full advance or explicit decline decision | Payment-risk violation |
| Supplier invoice with unresolved match discrepancy | AP control issue |
| Expense without vendor/category/description/date | Audit trail issue |
| Payroll without employee compensation profile validation | Payroll control issue |

## Butler For Managers

Use Intelligence to ask operational questions such as:

| Prompt | Expected use |
|---|---|
| `Show overdue customer invoices by risk.` | Collections review |
| `Which offers need follow-up this week?` | Sales review |
| `Summarize supplier invoices pending approval.` | AP review |
| `Create a task for Riley to follow up on the Gulf Smelting quotation.` | Work delegation |

Butler actions should be reviewed before execution. Treat Butler as an assistant, not an approval authority.

