# Parity Ledger — OrdersScreen (old) vs Orders descriptor

Verdicts:

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Paged list + Load More (`ListOrders(limit, offset)`) | **DONE** | Same `fetchPage`/`pageSize` shape as the invoices pilot. |
| 2 | Status filter tabs | **DONE** | Derived chip, unknown statuses render neutral (old screen's `normalizeOrderStatusKey` messy-string coercion is not reproduced — flagged, not silently dropped). |
| 3 | Year / Customer filters | **DONE** | Derived from `order_date`/`customer_name`. |
| 4 | Free-text search (order#/customer/customer PO#) | **DONE** | One `searchText` (L2). |
| 5 | Summary strip (Total Orders, Total Value, status mix) | **DONE** | New — old screen's stats row rebuilt as the declarative `SummarySpec`. |
| 6 | Delivery-status batch fetch (`GetOrderDeliveryStatusBatch`) | **ENGINE/SLOT** | Two-phase load a single `fetchPage()` row can't carry. K1 **mocks** `deliveryPercent` on the row (deterministic, not real data) so the column renders; real wiring needs either a second VM-level fetch phase or a backend join. Not built here. |
| 7 | Create Order (line-items editor, `CreateOrderWithItems`) | **LEDGER / SLOT** | Full line-item form is out of K1 scope by design (brief: "don't build"). No screen action exists for it yet. |
| 8 | Cascade-preview Delete (`PreviewOrderDeleteCascade` → `DeleteOrder`) | **LEDGER / ENGINE** | Financial hot-zone — blocks on payment/invoice dependents. `ActionSpec.confirm` today is a plain string; needs a `{message, lines[], blocked}` variant before this can be built honestly. Not built here. |
| 9 | Create Delivery Note / Create Supplier Order / Start Project handoffs | **LEDGER / INTEG** | Cross-screen `pending*` store + `navigateToScreen` event handoffs, not simple bindings. Needs a real navigation primitive at the engine level. Not built here. |
| 10 | Create Invoice / Proforma from Order | **LEDGER / INTEG** | Financial hot-zone (creates AR documents). Ledgered per the brief, same family as #9. Not built here. |
| 11 | Mark as Delivered (`QuickMarkOrderDelivered`) | **DONE** (mock) / **INTEG** | Fulfillment-only, not financial — the one safe row action built in K1. Real binding throws an honest INTEG-gap error naming `QuickMarkOrderDelivered`. |
| 12 | Traceability chain (RFQ→Offer→Order breadcrumb) | **SLOT** | Detail-panel visual; conceptually spans the whole K1-A cluster, not Orders alone. Not built here (no detail panel in K1). |
| 13 | No-items / zero-value data-quality banners | **SLOT** | Screen-level warning banner; not built here. |
| 14 | Row → Detail modal | **SLOT** | Detail panel (`slots.detail`) not built in K1 — list + row actions only. |

## Reading

K1 builds the Orders spine honestly: paged list, filters, search, and a new
summary strip all land at parity or better. Every deep feature — line-item
creation, cascade-delete, the four cross-screen financial handoffs, and the
delivery-status batch join — is deliberately ledgered per the K1 brief's
scope discipline, not silently dropped. The one row action built
(`Mark as Delivered`) was chosen specifically because it's the only
non-financial, non-handoff mutation on this screen. `deliveryPercent` is
mocked data in K1's mock bridge — real integration needs either a second
fetch phase in the viewmodel or a backend change that returns it inline.
