# A3 Recon — Deal Spine Data Reality (Wave 10, feeds B3/B4/B5a)

Read-only recon. Repo root: `c:\Projects\asymmflow\asymmflow-oss`.

## 1. FK chain — stage → stage → FK field

Models live in `pkg/crm/domain.go` (Offer, Order, DeliveryNote, PurchaseOrder,
SerialNumber) and `pkg/finance/domain.go` (Invoice, Payment). Registration
order (not schema) is in `trading_models.go:16-179`. `CustomerReceipt` /
`CustomerReceiptAllocation` are defined locally in `receipt_service.go:24-56`.

```
Opportunity (RFQ)  --Opportunity.OfferID-->            Offer   [pkg/crm/domain.go:308, :212]
Offer               --Offer.RFQID (opportunity link)--> (informal, string RFQ id)
Offer               --Order.OfferID / Order.OfferNumber--> Order   [domain.go:390-391]
                        (Order.RFQID also carried through, domain.go:392)
Order                --DeliveryNote.OrderID-->          DeliveryNote  [domain.go:479]
DeliveryNote(Item)   --DeliveryNoteItem.OrderItemID-->  OrderItem      [domain.go:521]
Order                --Invoice.OrderID-->                Invoice  [finance/domain.go:54]
Offer                --Invoice.OfferID / Invoice.QuoteID / Invoice.RfqID--> Invoice [finance/domain.go:73-76]
DeliveryNote          --Invoice.DeliveryNoteID-->        Invoice  [finance/domain.go:77]
Invoice               --Payment.InvoiceID-->              Payment  [finance/domain.go, Payment struct]
CustomerReceipt        --CustomerReceiptAllocation.ReceiptID
                        --CustomerReceiptAllocation.InvoiceID / .PaymentID--> Payment/Invoice
                        [receipt_service.go:46-53]
Order/PO (supplier side): PurchaseOrder links to Order indirectly via product/GRN,
  not a direct OrderID FK — procurement chain (PO -> GRN -> SupplierInvoice) is
  parallel to the sales chain, joined mainly through shared ProductID / OrderItem
  fulfillment counters (OrderItem.QuantityShipped/QuantityInvoiced, domain.go:424-425).
SerialNumber: linked at GRN receipt (assignSerialsToGRN), DN dispatch
  (allocateSerialsToDN/markSerialsShipped), and invoice (linkSerialsToInvoice) —
  see serial_number_service.go:129-149. Confirms serials are the cross-thread
  that ties supplier PO receipt to customer DN/Invoice for one physical unit.
```

Key exact FK field names (for B3's join logic):
- `Order.OfferID`, `Order.OfferNumber`, `Order.RFQID` (Order → Offer/RFQ)
- `DeliveryNote.OrderID` (DN → Order)
- `DeliveryNoteItem.OrderItemID` (DN line → Order line)
- `Invoice.OrderID`, `Invoice.OfferID`, `Invoice.QuoteID`, `Invoice.RfqID`, `Invoice.DeliveryNoteID` (Invoice → everything upstream — Invoice is the most heavily cross-linked node)
- `Payment.InvoiceID` (Payment → Invoice)
- `CustomerReceiptAllocation.ReceiptID` / `.InvoiceID` / `.PaymentID` (Receipt → Payment/Invoice)
- `SerialNumber` rows carry GRN/PO number, DN number, and Invoice number stamped in as the physical unit moves (see serial_number_service.go funcs above) — a serial's own row is effectively a mini per-unit timeline.

## 2. What Wave 9.x already built for assembly — NO single full-chain-in-one-call method exists for one order/deal.

Closest existing assemblers:
- **`GetInvoiceAuditTrail(invoiceID)`** — `invoice_traceability.go:29`. Returns `InvoiceAuditTrail{Invoice, RFQ *Offer, Quote *Offer, Order *Order, PurchaseOrders []PurchaseOrder, SupplierInvoices []SupplierInvoice, DeliveryNotes []DeliveryNote}` in one call. **This is the template to clone/extend** — but it's keyed by `invoiceID` (not orderID), and it does NOT include Payments, CustomerReceipts, or SerialNumbers.
- `GetInvoicesByOrder(orderID)` / `GetInvoicesByRFQ(rfqID)` — `invoice_traceability.go:222,245` — simple `WHERE order_id = ?` / `WHERE rfq_id = ?` lookups, no nesting.
- `LinkInvoiceToOrder` / `LinkInvoiceToRFQ` — `invoice_traceability.go:159,189` — mutators, not readers.
- `GetCustomer360(customerID)` and `GetCustomer360Graph(customerID)` — `app_order_customer_surface.go:2686` and `:2540` — customer-level rollups (all orders/offers/invoices for a customer), not single-deal chain assembly. Built on `customer_linkage_service.go`'s `customerLinkIndex`, which already resolves Order/Offer/RFQ/Invoice/Payment-history by customer via `linkedOrdersForCustomer`, `linkedOffersForCustomer`, `linkedInvoicesForCustomer`, `linkedPaymentHistoryForCustomer` (`customer_linkage_service.go:383-556`).
- No `SerialNumber` join is included in either assembler today; serial deep-links exist (`GetSerialsForInvoiceItem`, `GetRecentlyDeliveredSerials`, `serial_number_service.go:96,106`) but are queried separately from the order/invoice chain, not folded in.

**Recommended one-call assembly point for B3's `GetDealTimeline(orderID)`:** add a new function alongside `GetInvoiceAuditTrail` in `invoice_traceability.go` (or a new `deal_timeline_service.go` next to it), keyed by `orderID`:
1. Load `Order` (+ Items) by id.
2. Load `Offer` via `Order.OfferID` (+ Items) — gives RFQ/Costing/Offer-rev stage.
3. Load `[]DeliveryNote` via `WHERE order_id = ?` (+ Items) — reuse the query shape from `GetInvoicesByOrder`.
4. Load `[]Invoice` via `WHERE order_id = ?` (reuse `GetInvoicesByOrder` directly).
5. For each Invoice, load `[]Payment` via `WHERE invoice_id = ?` and `[]CustomerReceiptAllocation` via `WHERE invoice_id = ?`.
6. Optionally fold in `[]SerialNumber` via `GetSerialsForInvoiceItem`-style query per invoice item, if B3 wants per-unit granularity.

This costs ~5-6 sequential queries (Order, Offer, DNs, Invoices, then Payments/Allocations per invoice — small N, deals rarely have >5 invoices) — cheap enough for a single Wails round trip, same cost class as `GetInvoiceAuditTrail` already pays today.

## 3. The PAID transition — status is COMPUTED, not directly set.

There is no single line that writes `Status = "Paid"` as an imperative action. Instead:
- **`customerInvoiceSettlementStatus(invoice Invoice, outstanding float64, asOf time.Time) string`** — `customer_invoice_payment_policy.go:48-67` — is the pure function that derives status. The transition to Paid is: `if outstanding <= FloatingPointTolerance { return "Paid" }` (line 54-56), evaluated AFTER workflow-closed-state checks (Draft/Cancelled/Void/Proforma short-circuit first).
- This is invoked from **`applyCustomerInvoicePaymentState(tx *gorm.DB, invoice *Invoice)`** — `customer_invoice_payment_policy.go:123` — the single server-side function that owns writing the derived status+outstanding back to the DB row inside a transaction.
- `applyCustomerInvoicePaymentState` is called from **`RecordPartialPayment(id, paymentAmount, paymentDate, paymentRef)`** — `customer_invoice_service.go:1974` (the App-level Wails-bound entry point; reduces `invoice.OutstandingBHD`, then calls `applyCustomerInvoicePaymentState`, then creates the `Payment` row, all inside one `tx`).
- `MarkCustomerInvoicePaid(id, paymentDate, paymentRef)` — `customer_invoice_service.go:1809` — the "mark fully paid" convenience wrapper — does NOT set status directly either; it computes `state.OutstandingBHD` via `customerInvoicePaymentStateFromInvoice` and delegates to `RecordPartialPayment` for the full outstanding amount.
- Status is ALSO recomputed lazily on read via `hydrateCustomerInvoicePaymentState` / `hydrateCustomerInvoicesPaymentState` (`customer_invoice_payment_policy.go:89-103`), called wherever invoices are listed — so a past-due invoice flips to Overdue on read even without a mutator running.

**B4's sound trigger / B5a's "set complete" moment:** fire on the return of `applyCustomerInvoicePaymentState` when `state.Status == "Paid"` AND previous status was open (`state.IsOpen` was true pre-call) — i.e. instrument the write path inside `RecordPartialPayment` (`customer_invoice_service.go:1974`, right after `applyCustomerInvoicePaymentState` returns at line 2050), not the read-side hydrate (that would false-trigger on every list view).

## 4. Wails bindings — where a new `GetDealTimeline` would live, and regen command

- Existing invoice/order/CRM Wails-bound App methods are spread across `app_order_customer_surface.go` (Order/Customer/Supplier CRUD, `GetCustomer360*`), `app_crm_surface.go` (RFQData struct, CRM-facing methods), and plain top-level files like `invoice_traceability.go` and `customer_invoice_service.go` — Wails binds ANY exported `func (a *App) ...` method regardless of which file it's in (all `package main`), so file placement is organizational, not functional.
- **Recommendation:** put `GetDealTimeline(orderID string) (DealTimeline, error)` in `invoice_traceability.go` right next to `GetInvoiceAuditTrail` — same pattern, same package, keeps the "traceability" methods co-located.
- **Binding regeneration:** there is no dedicated Task in `Taskfile.yml` for this (checked — only `build`, `test`, `lint`, `clean`, `frontend:build`, `frontend:dev`, `audit`). `wails.json` (project root) has no explicit `wailsjsdir` override, so bindings default to `frontend/wailsjs/`. Bindings are regenerated by the **standard Wails CLI**: `wails generate module` (bindings-only) or implicitly on `wails dev` / `wails build`. Confirmed existing generated files at `frontend/wailsjs/go/main/App.js` / `App.d.ts` and `frontend/wailsjs/go/main/FinanceService.js` — e.g. `RecordPartialPayment` already appears in both `App.d.ts` and `FinanceService.js`, showing methods get bound wherever the receiver type lives (App vs a service struct) and both surface bindings need regeneration together.

## 5. Frontend mount points (Svelte)

No SvelteKit file-based `routes/` — this app is a screens/SPA router; `frontend/src/routes` has no order/customer/deal/timeline matches at all. Actual UI lives under `frontend/src/lib`:
- `frontend/src/lib/components/OrderDetail.svelte` — **already has a `.timeline` section** (line 485, styled at line 769) — this is the natural mount point for B3's deal-timeline component; it currently renders order-level info only, no cross-stage chain.
- `frontend/src/lib/components/OrderCard.svelte` — order summary card (likely the list-row context, not detail).
- `frontend/src/lib/screens/Customer360.svelte` — imports `GetCustomer360` from `wailsjs/go/main/App` (line 7) and `GetCustomer360Graph` from `wailsjs/go/main/CRMService` (line 8), calls them at load (lines 66-70). This is the customer-level 360 view (deal ROWS per customer) — B3's per-deal timeline would be a drill-down FROM here into `OrderDetail.svelte`, not embedded directly in Customer360 itself.
- `frontend/src/lib/components/customer/CustomerOrdersTab.svelte` and `CustomerInvoicesTab.svelte` — the per-customer tabs listing orders/invoices — these are the "deal rows" B3 needs to link out from into the timeline.
- `frontend/src/lib/screens/CustomerDetailView.svelte` and `CRMCustomerDashboard.svelte` — other customer-level screens, likely composing the Customer360 tabs.

## Summary of open items / risks for B3/B4/B5a
- Need a NEW backend method (`GetDealTimeline`), no existing one-call assembler for a single order's full chain including receipts/payments/serials.
- PAID transition is a derived/computed value, not a direct write — hook the sound trigger into `RecordPartialPayment`'s post-`applyCustomerInvoicePaymentState` return, not a naive `Status == "Paid"` field-watch (read-side hydrate will false-fire).
- Procurement-side (PO→GRN→SupplierInvoice) is NOT directly FK-linked to the sales Order — only loosely joined via shared Product/OrderItem fulfillment counters and SerialNumber stamps. If B3 wants to show supplier-side procurement stages in the same timeline, that join is weaker/optional (best-effort via SerialNumber or shared OrderItem/ProductID), unlike the tight sales-side FK chain.
- Bindings regen has no Taskfile shortcut — must run `wails generate module` (or `wails dev`/`wails build`) manually after adding the method.
