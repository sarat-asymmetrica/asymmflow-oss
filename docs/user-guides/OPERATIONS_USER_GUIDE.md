# Operations User Guide

Last updated: 2026-04-25

## Role Purpose

Operations users manage the supplier-side and delivery-side flow after or alongside the sales order: supplier orders, supplier invoices, matching, delivery notes, dispatch, delivery confirmation, and operational document capture.

## Daily Operations Routine

1. Open Operations Hub.
2. Review Supplier Orders for drafts, pending approval, sent, acknowledged, received, or closed status.
3. Review Supplier Invoices for pending match, discrepancy, dispute, approved, or paid status.
4. Review Delivery Notes for prepared, dispatched, delivered, signed, or cancelled status.
5. Open Notifications for pending approvals or operational comments.
6. Open Work for assigned delivery/procurement tasks.

## Supplier Orders

Open Operations -> Supplier Orders.

Create/edit fields:

| Field | What to enter |
|---|---|
| Supplier | Supplier from master list |
| Linked Customer Order | Customer order this PO supports, if any |
| PO Date | Date supplier order is raised |
| Expected Delivery | Expected arrival/delivery date |
| Currency | Supplier currency, e.g. EUR, USD, BHD |
| Exchange Rate | Conversion rate to BHD |
| Payment Terms | Supplier payment terms |
| Item Description | Product/service being purchased |
| Quantity | Quantity ordered |
| Unit Price Foreign | Supplier unit price in selected currency |

Status behavior:

| Status | Meaning |
|---|---|
| Draft | PO is being prepared |
| Pending Approval | Value or workflow requires approval |
| Approved | Approved for issue |
| Sent | Sent to supplier |
| Acknowledged | Supplier has acknowledged |
| Partially Received | Some quantity received |
| Received | Fully received |
| Closed | Finished |
| Cancelled | Cancelled and should not be used for receiving/payment |

Controls:

| Control | Behavior |
|---|---|
| 5,000 BHD threshold | High-value POs require approval workflow |
| PDF generation | Creates supplier PO PDF |
| Update status | Moves PO through lifecycle |
| Linked order | Maintains customer-to-supplier traceability |

## Supplier Invoices

Open Operations -> Supplier Invoices.

Create fields:

| Field | What to enter |
|---|---|
| Supplier | Supplier issuing invoice |
| Invoice Number | Supplier invoice number |
| Customer Order | Internal order supported by invoice |
| PO Reference | Related purchase order |
| Invoice Date | Supplier invoice date |
| Due Date | Payment due date |
| Currency | Supplier invoice currency |
| Exchange Rate | Currency to BHD |
| Item Description | Invoice line description |
| Quantity | Invoice quantity |
| Unit Price | Invoice unit price |
| Subtotal | Net amount before VAT |
| VAT | VAT amount |

Actions:

| Action | Result |
|---|---|
| Three-Way Match | Compares supplier invoice with PO and GRN/receipt evidence |
| Approve | Approves invoice after validation |
| Mark Paid | Updates payment status and payment reference |
| Edit | Updates invoice data |
| Open Bank Recon | Moves to finance bank matching workflow |

Do not approve invoices with unresolved discrepancy or dispute unless management has recorded the exception.

## Delivery Notes

Open Operations -> Delivery Notes.

Create/edit fields:

| Field | What to enter |
|---|---|
| DN Number | Leave generated value or enter official DN reference |
| Delivery Date | Planned or actual delivery date |
| Customer Order | Order being delivered |
| Ship Quantity | Quantity delivered for each order line |
| Serial Selection | Choose available serial numbers where serial tracking applies |
| Delivery Address | Full customer delivery address |
| Contact Person | Receiving contact |
| Contact Phone | Receiver phone |
| Driver Name | Driver/courier name |
| Vehicle Number | Vehicle registration or courier reference |
| Transport Method | Own Vehicle, Courier, Customer Pickup, etc. |
| Status | Prepared, Dispatched, Delivered, Signed, Cancelled |
| Partial Delivery | Check when this is one delivery in a sequence |
| Delivery Sequence | Current partial delivery number |
| Total Deliveries | Expected total partial deliveries |

Actions:

| Action | Result |
|---|---|
| Dispatch | Moves DN to dispatched and updates serials to shipped |
| Confirm Delivery | Marks delivered and updates serial/warranty state |
| Generate PDF | Creates delivery note PDF |
| Delete | Removes DN where allowed |

Operational rule: delivery quantity cannot exceed remaining order quantity. Serial numbers are allocated atomically so the same serial cannot be dispatched twice.

## OCR And Document Capture

Operations can use file drop / OCR for supplier invoices, POs, delivery notes, and other operational documents.

When OCR proposes extracted data:

1. Review supplier/customer identity.
2. Check invoice number, dates, amounts, currency, and VAT.
3. Link to PO/order where possible.
4. Save only after human validation.

## Work And Notifications

Use Work for:

| Task type | Example |
|---|---|
| Procurement follow-up | Supplier acknowledgment pending |
| Delivery blocker | Awaiting vehicle, serial, customer gate pass |
| Invoice discrepancy | Supplier invoice mismatch |
| Internal handoff | Sales needs delivery ETA |

Use comments to document operational facts rather than keeping them in chat messages outside the system.

