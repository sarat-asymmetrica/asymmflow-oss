# Parity Ledger — InventoryFulfillmentScreen (old) vs InventoryFulfillment ledger descriptor

Verdicts:

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Read-only report: single fetch, columns, no create/edit/delete, no status transitions | **DONE** | The simplest possible `LedgerDescriptor` — recon-K2 called this "the closest-to-kernel-shape screen in the whole batch." No `actions` array; the row-click deep-link is the only interactive affordance in the old screen, and it's a nav (see #2). |
| 2 | Row-click → Open Order (`navigateToScreen` custom event, `order_id`/`order_number`) | **INTEG** | Same app-shell-nav dependency K1 flagged repeatedly (Invoices' Bank Recon jump, Suppliers' `startNewPO`/`openPO`) — now a whole-app pattern per recon-K2 synthesis #3, not this screen's own gap. Not built; needs a real router. |
| 3 | Order-status tone via substring match on free-text status (not an exact vocabulary) | **FIXED, not preserved** | The old screen's substring matching was itself evidence of a messy vocabulary, not a feature worth reproducing. `StatusSpec.tones` maps the 9 known values (Delivered/Invoiced/Closed/Complete/Pending/Processing/Open/Cancelled/Lost) as exact keys; anything else — including the mock's deliberate `ON_HOLD_CUSTOMS_INSPECTION` adversary row — renders neutral by the existing engine contract. No `StatusSpec` extension needed. |
| 4 | Row count shown as plain text ("N outstanding lines") | **DONE, upgraded** | Rebuilt as a `summary` strip: Total Lines, Total Shortage Qty (sum, danger tone if >0), Lines With Shortage (count, danger tone if >0), plus a Shortage/OK distribution bar — none of which existed as a real strip in the old screen (it was one line of text). |
| 5 | `description`/`invoiced_quantity` fetched but never rendered | **FIXED, partially** | `description` is now a real column (free enhancement — "why is this pending" per recon-K2 #4). `invoiced_quantity` stays on the row type (mapped from the real fetch, present in the mock) but is not rendered as a column — the K2 brief's column list didn't ask for it; the field is there if a future wave wants it. |
| 6 | No filters (flat list) | **FIXED, not preserved** | Two derived filters added: Order Status and a Shortage/OK toggle (`r => r.shortageQuantity > 0 ? 'Shortage' : 'OK'`) — both free given the data already carries `shortage_quantity`. |
| 7 | Pending/Shortage columns colour-coded (amber/red if >0) | **DONE** | `ColumnSpec.tone` on both columns — amber (`warning`) for Pending > 0, red (`danger`) for Shortage > 0, matching the old CSS thresholds exactly. |

## Reading

This screen was already the best-fitting candidate for the ledger archetype
in the whole K1/K2 batch, and the descriptor doesn't add friction: it's a
flat read-only report with a summary strip, two derived filters, and
threshold-coloured quantity columns — genuinely built to spec, not scoped
down. The one real gap is the row-click drill-through to Orders, which is
the same cross-screen-nav dependency that recurs across ~8 screens now and
belongs in the kernel's app-shell router, not a per-screen workaround. The
messy free-text order-status vocabulary was fixed at the mapping boundary
(exact-key tones, unknown → neutral) rather than reproducing the old
screen's substring-matching fragility.
