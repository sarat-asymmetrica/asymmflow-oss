# Parity Ledger — RFQScreen (old) vs RFQs descriptor

L6 requires judging every rebuilt screen against the old one. Verdicts:

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | List `GetRFQs(100, 0)`, no Load-More + stage tabs + search | **DONE** | Unpaged `fetch()`; real bridge calls `GetRFQs(200, 0)` internally (matches invoices' `fetchInvoices→fetchInvoicesPage(200,0)` pattern). Search sweeps number/client/project/notes. |
| 2 | Edit stage (dropdown restricted to Pending/Qualified/Proposal/Negotiation, Won/Lost read-only) | **EQUIV** | `FormModal` has no row-context binding today — `ActionHost.run(action, row)` keeps the clicked row in its own `$state`, but `<FormModal spec={...}>` never receives it, so a static `ActionSpec.form.submit(draft)` can't know which RFQ it's editing. Built instead as four gated `confirm`-only row actions ("Set Pending" / "Set Qualified" / "Set Proposal" / "Set Negotiation"), each visible only when the row's current stage is one of the four and isn't already the target — same net capability (any-of-4 reassignment, Won/Lost untouchable from this screen), no engine change needed. |
| 3 | (derived from #2) Row-scoped form actions in general | **ENGINE** | The real fix is giving `FormModal`/`ActionHost` a row-aware submit path (`submit: (draft, row) => …` or similar). This blocks the same shape everywhere a "capture input, mutate this one row" action is needed — Offers' Won (capture PO#)/Lost (capture reason), and per recon's cross-screen synthesis #3, Orders' cascade-delete preview and DeliveryNotes/GRN's recoverable-confirm flows. Flagging for the orchestrator; not something a single screen's descriptor can route around cleanly. |
| 4 | Create RFQ (customer select + multi-product line items + notes) | **SLOT** | Multi-line create is out of K1 scope per the build brief — needs the form archetype's line-item repeater (not shipped yet, same gap `invoices.descriptor.ts`'s create form doesn't hit because invoices only needs header fields). Not built. |
| 5 | Due Date column + create-form field | **INTEG (backend gap, pre-existing)** | `due_date` is rendered by the old screen but isn't a real column — `RFQData` has no `DueDate` field and `CreateRFQ(client, project, value, notes, productDetails)` takes no due-date param. Column is kept (shows the gap honestly) — real bridge always maps it to `''`; mock fills plausible dates so the column reads normally in the lab. Not a kernel-side decision: either drop the column or land a real backend field. |
| 6 | Delete with confirm (`DeleteRFQ`) | **DONE** | Same `ConfirmDialog`/`ActionSpec.confirm` pattern as invoices' Delete Draft. |
| 7 | Product count column | **INTEG** | Not a real `RFQData` field. Real bridge best-effort parses `product_details` as JSON and counts array entries — unverified against `GetRFQs`' actual query (out of this recon's scope, per census). Mock generates a plausible count directly. |
| 8 | `status` vs `stage` field ambiguity | **INTEG** | Old screen reads `row.status` for the badge and writes via `UpdateRFQStage` (which — per its name — sounds like it should target `stage`, a separate 9-value pipeline field the screen never surfaces). The bridge/mock here follow the screen's own behavior (`status` is the display+mutate field); which column real `UpdateRFQStage` actually writes needs an INTEG-time DB check. |
| 9 | Win-rate stat with 56.8% target line | **DEFER** | Visual-diversity opportunity noted in recon, not required by the build brief's mandatory summary (Total RFQs / Total Value / Won count / stage distribution — all built). |
| 10 | Stage summary strip (Total RFQs, Win Rate, Total Value, Avg Deal Size) | **DONE**/EQUIV | Kernel's `summary` strip: RFQ count, Total Value (BHD), Won count (tone), + a by-stage distribution bar — denser than the old card grid, same information class. |
| 11 | `UpdateRFQStatus(id, status)` binding | **DEFER** | Exists server-side (`app_sales_pipeline.go:1212`) but this screen never calls it (only `UpdateRFQStage` is used) — dead from this screen's perspective; not wired here either. Flagged by recon as worth confirming with another caller before assuming unused system-wide. |

## Reading

The list/search/filter/delete surface is at parity (#1, #6). The one real
mutation this screen owns beyond delete — stage editing — hits a genuine
engine limit (#2/#3): `FormModal` can't be scoped to a specific row today, so
the "one dropdown, four options" old-screen mechanism became "four gated
buttons." Functionally equivalent, but it's the tell that the next kernel
mountain isn't another archetype, it's giving the form path row context.
Multi-product create (#4) and the due-date/product-count backend gaps
(#5/#7/#8) are pre-existing debt this screen surfaces rather than causes —
none of them are things the kernel side should paper over.
