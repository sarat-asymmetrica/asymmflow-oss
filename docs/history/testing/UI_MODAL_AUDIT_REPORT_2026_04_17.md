# UI Modal Audit Report - 2026-04-17

Purpose: second-pass review of popup/modal surfaces after the button/action inventory. This report checks whether modal content is complete enough for the workflow and whether submit actions call the right backend functions.

Generated alongside:

- `docs/testing/UI_MODAL_INVENTORY_2026_04_17.md`
- `docs/testing/UI_MODAL_INVENTORY_2026_04_17.json`
- `docs/testing/UI_ISSUE_LOG_2026_04_17.md`

## Summary

- Static modal surfaces detected: 63 screen modals plus 3 reusable modal components.
- Backend-mapped modal actions after fixes: 49.
- Static notes remaining: 13. Most remaining notes are parser limitations around form submit handlers, wrapper close buttons, `ContextTaskModal`, and demo components.
- Build verification after fixes: `npm run build` passed, `go build ./...` passed.

## Fixes Applied In This Pass

| Area | Modal | Problem | Fix |
| --- | --- | --- | --- |
| Customer Orders | New/Edit Order | Create mode hid line items and only called header-only `CreateOrder(...)`. | Create mode now shows line items, requires at least one priced item, computes totals from rows, creates the header, then persists rows through `UpdateOrder`. |
| Customer Orders | Order Detail | Traceability showed Customer PO as `RFQ` and hardcoded Offer as blank. | Detail modal now labels RFQ/enquiry separately and shows `offer_number` when available. |
| Accounting | New Accounting Entry | Voucher creation only mutated local frontend state. | Modal now calls `CreateJournalEntry` and reloads backend journal data. |
| Accounting | Add/Edit Account | Account creation/edit only mutated local frontend state. | Modal now calls `CreateAccount` / `UpdateAccount` and reloads backend chart of accounts. |
| Accounting backend | Update account API | `UpdateAccount` accepted a numeric id while accounts use UUID string ids. | API signature and Wails TypeScript declaration now use string ids. |
| Supplier Payments | Edit Supplier Payment | Foreign-currency edit hid the exchange rate used for `amount_bhd`. | Edit modal now exposes `Exchange Rate to BHD` for non-BHD payments and validates it before `UpdateSupplierPayment`. |

## High-Confidence Modal Findings

| Screen | Modal | Status | Backend Methods |
| --- | --- | --- | --- |
| Delivery Notes | New/Edit Delivery Note | Wired, but workflow status selection is too permissive. | `CreateDeliveryNote`, `CreateDNWithSerials`, `UpdateDeliveryNote` |
| Delivery Notes | Detail | Wired. | `GenerateDeliveryNotePDF`, `DispatchDeliveryNote`, `ConfirmDeliveryNote`, `DeleteDeliveryNote`, `UpdateDeliveryNote` |
| Customer Invoices | Create Invoice | Wired and field set matches backend options. | `CreateInvoiceWithOptions` |
| Customer Invoices | Edit/Delete/Credit Note | Wired. | `UpdateCustomerInvoice`, `DeleteCustomerInvoice`, `CreateCreditNote`, `ApplyCreditNote`, `GenerateCreditNotePDF` |
| Offers | Re-quote/Create Offer | Reached through re-quote; main `+ New Offer` intentionally routes to costing. Minimal modal is acceptable only as a re-quote shortcut. | `SaveCostingAsOffer` |
| Offers | View/Edit Details | Backend preserves items when `items: []`; current header save path is acceptable but should be manually tested on a real offer. | `UpdateOfferFull` |
| Purchase Orders | Create/Edit/View | Wired for create/update/status/PDF. Delete backend is imported but not exposed. | `CreatePurchaseOrder`, `UpdatePurchaseOrder`, `UpdatePOStatus`, `GeneratePurchaseOrderPDF` |
| Supplier Invoices | Create/Edit/Payment/Match | Wired. Delete backend is imported but not exposed. Approval attribution still uses `System Admin`. | `CreateSupplierInvoice`, `UpdateSupplierInvoice`, `UpdateSupplierInvoiceWithPayment`, `PerformThreeWayMatch`, `MarkSupplierInvoicePaid` |
| Payments Received | Record/Edit Payment | Wired. | `RecordPayment`, `UpdatePayment`, `DeletePayment` |
| Bank Reconciliation | Import/Match/Edit/Add/Accounts | Wired. | `ImportBankStatementWithDialog`, `ManualMatchLine`, `UpdateBankStatement`, `CreateBankStatementLine`, `UpdateBankStatementLine`, `CreateBankAccount`, `UpdateBankAccount`, `DeleteBankAccount` |
| Expenses | Payment Modal | Wired. | `MarkExpenseEntryPaid` |

## Open Modal Issues

| Issue | Severity | Modal | Notes |
| --- | --- | --- | --- |
| UI-008 | P2 | Reports export | Date range is visually selectable but backend load/export uses a hardcoded month bucket. PDF/Excel options are shown even though backend export supports CSV only. |
| UI-009 | P2 | Delivery note create/edit | Status dropdown can bypass dispatch/confirm workflow buttons. |
| UI-010 | P3 | PO/Supplier invoice delete | Delete backend functions are imported but no guarded delete modal/action exists. |
| UI-002 | P1 | User management | Add/Edit/Permissions controls are inert and no real modals exist yet. |

## Manual App Test Checklist

1. Create a new customer order with two line items. Reopen it and confirm both items persist.
2. From that order, create a DN and invoice; confirm they inherit the real items.
3. Open Accounting > Chart of Accounts, add a test account, reload, and confirm it persists. Then edit the account name and confirm it persists.
4. Open Accounting > Journal Entries, create a balanced two-line manual voucher, reload, and confirm it persists.
5. Edit a non-BHD supplier payment and verify changing the exchange rate updates the BHD amount correctly.
6. Try Reports export after changing the date range and confirm UI-008 before fixing it.
7. Try setting a Delivery Note directly to a later status from edit mode and confirm UI-009 before deciding the final workflow rule.
