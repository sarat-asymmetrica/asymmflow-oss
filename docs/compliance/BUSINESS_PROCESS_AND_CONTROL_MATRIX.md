# AsymmFlow Business Process And Control Matrix

Last updated: 2026-04-25

This document maps the application's major business flows to controls, validations, and evidence points.

## Core Process Flow

| Step | Module | Primary records | Control objective |
|---:|---|---|---|
| 1 | Opportunities | RFQ / Opportunity | Capture enquiry and customer context |
| 2 | Costing | Costing sheet, line items | Validate cost, margin, VAT, terms |
| 3 | Offers | Offer, offer items, follow-ups | Issue controlled quotation and track outcome |
| 4 | Customer Orders | Order, order items | Convert won offer/customer PO into fulfillment record |
| 5 | Supplier Orders | Purchase order, PO items | Procure goods/services with approval threshold |
| 6 | Supplier Invoices | Supplier invoice, invoice items | Match supplier invoice to PO/receipt evidence |
| 7 | Delivery Notes | DN, DN items, serials | Dispatch and confirm customer delivery |
| 8 | Customer Invoices | Invoice, invoice items, credit notes | Bill customer with traceability and VAT controls |
| 9 | Payments | Customer/supplier payments | Record cash movement without overpayment |
| 10 | Bank Reconciliation | Bank statements/lines/matches | Reconcile bank to system records |

## Business Rule Controls

| Control | Rule | Evidence in code |
|---|---|---|
| Minimum margin | Approved commercial work should not fall below 8% actual margin | `business_invariants.go` |
| Grade C payment risk | Grade C customers require at least 50% advance | `business_invariants.go` |
| Grade D payment risk | Grade D customers require 100% advance or decline | `business_invariants.go` |
| ABB competition | ABB competition requires at least 15% margin to proceed | `business_invariants.go` |
| VAT bounds | VAT rate should stay between 0 and 100 | Costing/offer backend validations |
| VAT 0% | Explicit 0% is allowed for valid exempt/export cases | Offer PDF and costing flow |
| PO approval threshold | POs above 5,000 BHD enter approval control | `purchase_order_service.go` |
| Customer credit limit | Invoice creation checks credit blocked and credit limit state | `customer_invoice_service.go` |
| Payment overrun prevention | Customer payment cannot exceed outstanding invoice balance | `payment_service.go` |
| Supplier payment overrun prevention | Supplier payment cannot exceed outstanding supplier invoice balance | `supplier_payment_service.go` |
| Delivery quantity control | DN quantity is rechecked in transaction | `delivery_note_service.go` |
| Serial allocation control | Serial assignment uses atomic availability update | `delivery_note_service.go`, `serial_number_service.go` |
| Supplier invoice matching | Three-way match compares invoice against PO/receipt context | `supplier_invoice_service.go` |
| Document integrity | Invoice hashes/HMAC support document integrity | `customer_invoice_service.go`, `field_crypto.go` |

## Data Integrity Controls

| Area | Control |
|---|---|
| SQLite | WAL mode, busy timeout, synchronous NORMAL, cache and mmap tuning |
| Transactions | Payments, invoice creation, delivery, supplier payments, PO updates use DB transactions |
| Backups | Atomic `VACUUM INTO` backup, permissions, retention |
| Integrity check | Startup database integrity check |
| Sync | Merge-only sync, canonical dependency-ordered table list |
| Audit | Created/updated metadata and financial audit logging paths |
| Soft delete | Base model supports soft delete for recoverable records |
| Validation | Server-side validators protect customer/user/document input |

## Process Evidence Checklist

Use this checklist during client sign-off:

| Process | Evidence to capture |
|---|---|
| RFQ to offer | Screenshot/export of RFQ, costing, offer PDF |
| Offer won to order | Offer marked won with customer PO number, resulting order |
| Order to PO | Customer order linked to supplier PO |
| PO approval | PO above threshold shows approval path |
| Supplier invoice match | Match result and approval state |
| Delivery | DN PDF and status transition to delivered |
| Customer invoice | Invoice PDF with correct visibility controls |
| Payment | Payment record and invoice outstanding reduction |
| Bank reconciliation | Imported statement, matched lines, finalized status |
| Backup/recovery | Backup path, timestamp, and integrity check result |

## Exception Handling

| Exception | Required handling |
|---|---|
| Below-minimum margin | Manager/admin decision and documented reason |
| Credit-blocked customer | Do not create invoice/order without management override path |
| Supplier invoice discrepancy | Keep in Discrepancy/Dispute until resolved |
| Partial delivery | Use partial delivery checkbox and sequence fields |
| Missing bank line | Add manual statement line with note |
| Imported OCR uncertainty | Human review before save |
| Cloud sync unavailable | Continue local work and sync later |

