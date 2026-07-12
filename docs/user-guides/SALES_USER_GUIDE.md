# Sales User Guide

Last updated: 2026-04-25

## Role Purpose

Sales users own the customer-facing flow: RFQ/opportunity capture, costing preparation, offer generation, follow-up, won/lost updates, and customer order creation. Sales also maintains customer relationship context and can raise supplier orders where tied to commercial flow.

## Daily Sales Routine

1. Open Opportunities.
2. Review RFQs by year, stage, and search.
3. Update notes and create tasks for follow-up.
4. Prepare costing for qualified opportunities.
5. Save approved costing as an offer.
6. Export/send quotation PDF.
7. Schedule follow-ups.
8. Mark offers won/lost.
9. Create or review customer orders after a win.

## RFQs And Opportunities

Open Opportunities -> RFQs.

To create a new opportunity:

| Field | What to enter |
|---|---|
| Customer | Existing customer name from search list, or the customer name from the enquiry |
| Project | Short project/enquiry title, e.g. `Flowmeter replacement for Gulf Smelting line 3` |
| Value (BHD) | Expected revenue value in BHD; use `0` if unknown |
| Priority | Low, Medium, High, or Urgent based on customer deadline and strategic value |
| Notes | Scope, model numbers, customer contact, exclusions, delivery expectation, next action |

How it functions:

| Action | Result |
|---|---|
| Create | Creates RFQ/opportunity record |
| Search | Filters by customer/project text |
| Year filter | Shows pipeline for selected year |
| Sort | Sorts by selected metric |
| Open card | Shows opportunity details and comments/tasks |
| Delete | Deletes RFQ; cascade delete removes linked records where allowed |

## Costing

Open Opportunities -> Costing.

Workflow:

1. Select an opportunity.
2. Confirm or select customer.
3. Fill header fields.
4. Add or review line items.
5. Confirm margin, VAT, discount, and hidden charges.
6. Export PDF/Excel for review if needed.
7. Save as Offer.

Header fields:

| Field | What to enter |
|---|---|
| Customer | Customer receiving the quote |
| Contact Person | Buyer or technical contact name |
| RFQ Reference | Customer enquiry/reference number |
| Division | `Acme Instrumentation` or `Beacon Controls` |
| Date | Quotation/costing date |
| Prepared By | Salesperson/support person preparing the quote |
| Quote Type | Quotation, Budgetary Quote, Estimate, Technical, or Commercial |
| Folder Number | Physical/electronic folder number, e.g. `42-26` |
| Costing ID | Leave auto unless correcting a known reference |
| Payment Terms | Agreed customer payment terms |
| Delivery Terms | Delivery basis, e.g. DAP Bahrain |
| Estimated Delivery | Weeks or delivery promise |
| Subject | Customer-facing quotation subject |
| Opening Body | Cover note shown before line items |

Line item fields:

| Field | What to enter |
|---|---|
| Equipment | Product/equipment name |
| Model | Supplier model number |
| Quantity | Quantity required |
| Currency | Supplier currency, e.g. EUR, USD, BHD |
| Exchange Rate | BHD conversion rate |
| FOB | Supplier FOB/unit cost in source currency |
| Freight % | Freight percentage applied |
| Other Costs | Internal miscellaneous cost |
| Suggested Price | System-calculated suggested selling price |
| User Price | Manual selling price override if needed |
| Long Code | Full supplier ordering code |
| Detailed Description | Specs, approvals, HS codes, technical detail for PDF annexure |
| Customs % | Customs duty percentage |
| Handling % | Handling cost percentage |
| Finance % | Finance cost percentage |
| Margin % | Target margin percentage |

Summary fields:

| Field | What to enter |
|---|---|
| Discount | Customer-visible discount amount |
| Hidden Charges | Internal cost/profit adjustment; not printed to customer |
| VAT Rate | VAT percentage, 0 to 100 |
| Terms and Conditions | Commercial terms printed on quote |

Important sales rules:

| Rule | Meaning |
|---|---|
| Minimum margin | Do not submit below 8% actual margin without management decision |
| Grade C | Expect advance payment requirement |
| Grade D | Full advance or decline path |
| ABB competition | Use stronger margin discipline |
| VAT 0% | Use only where transaction is genuinely VAT exempt/export eligible |

## Offers

Open Opportunities -> Offers.

Main actions:

| Action | What to do |
|---|---|
| Create Offer | Enter customer, project, quote dates, and line items manually |
| Edit Offer | Update equipment, model, currency, costs, margin, price, and header fields |
| Generate PDF | Export quotation PDF for customer |
| Mark Won | Enter exact customer PO number |
| Mark Lost | Select loss reason |
| Schedule Follow-Up | Pick future date and notes |
| Add Note | Capture sales context or customer call notes |

Offer fields:

| Field | What to enter |
|---|---|
| Customer | Customer receiving quote |
| Project Name | Project/reference title |
| Quotation Date | Offer date |
| Validity Date | Expiry date; must be on/after quote date |
| Equipment | Product name |
| Model | Model number |
| Currency | Source currency |
| Specification | Short technical description |
| Detailed Description | Extended specs and approvals |
| Quantity | Quoted quantity |
| FOB | Supplier cost |
| Freight | Freight amount |
| Margin % | Commercial margin |
| Unit Price | Customer unit sell price |
| Customer PO Number | Required when marking won |
| Lost Reason | Required when marking lost |
| Follow-Up Date | Future follow-up date |
| Follow-Up Notes | What to discuss/check |

## Customer Orders

Open Opportunities -> Customer Orders.

Order fields:

| Field | What to enter |
|---|---|
| Order Number | Leave generated value or enter official order reference |
| Customer PO Number | Customer's purchase order reference |
| Customer | Customer name |
| Order Date | Date order was received |
| Required Date | Required delivery date |
| Total Value | Order value in BHD |
| Status | Processing, Delivered, Cancelled, or current status |
| Payment Terms | Customer payment terms |
| Delivery Terms | Delivery basis |
| Line Code | Product/model code |
| Description | Product/service description |
| Quantity | Ordered quantity |
| Unit Price | BHD unit price |

Order actions:

| Action | Result |
|---|---|
| Create Delivery Note | Starts delivery workflow |
| Create Purchase Order | Starts supplier order procurement |
| Create Invoice | Creates customer invoice from order |
| Create Proforma | Creates proforma invoice |
| Quick Mark Delivered | Updates order delivery status quickly |
| Edit | Updates order data |

## Relationship Management

Open Relationships -> Customers.

Use customer detail views to review orders, invoices, contacts, notes, and risk information. Keep contact names, emails, mobile numbers, and notes current because they flow into follow-ups and customer-facing documents.

## Butler For Sales

Use Intelligence for:

| Prompt | Use |
|---|---|
| `Draft an offer for this opportunity.` | Create offer draft action |
| `Which quotations are overdue for follow-up?` | Follow-up planning |
| `Show won offers missing orders.` | Pipeline hygiene |
| `Create a task to follow up with Gulf Smelting next Tuesday.` | Task creation |

Review every Butler action before accepting it.

