# Parity Ledger — OffersScreen (old) vs Offers descriptor

L6 requires judging every rebuilt screen against the old one. Verdicts:

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Create/Edit offer (`+ New Offer`, row `Edit`) | **LEDGER / INTEG-DEFER (major)** | Both eject entirely to `CostingSheetScreen` (~2,000+ line pricing/costing tool, out of this recon batch) — Offers is not self-contained in the old UI. Per the build brief this is out of K1 scope: either port CostingSheet as the form archetype's real create/edit surface, or keep Offers read-mostly, which is what's built here. Not attempted. |
| 2 | List (`GetAllOffers()`, no pagination) | **DONE** | Unpaged `fetch()`, matches the old screen's own behavior (no Load-More existed to replicate). Flagged by recon as a candidate for real server-side paging once customer data volume demands it — a backend change, not a frontend one. |
| 3 | Division scoping | **DONE** | Filter derived from loaded rows, same as the old screen's already-L7-clean `getDefaultDivisionKey`/`isKnownDivision` sourcing — no hardcoded division literal here either. |
| 4 | Stage tabs incl. computed Expired (`isOfferExpired()`) | **DONE** | `effectiveStage()` in the bridge: raw `stage` unless the offer is still RFQ/Quoted and `validity_date` has passed, then `'Expired'` — same one-truth treatment of the client signal + the backend `AutoExpireOffers` job the old screen relies on. Status filter tabs + summary distribution both read this effective value. |
| 5 | Won (capture Customer PO #, creates an Order) / Lost (capture reason, terminal) | **LEDGER (financial hot-zone + ENGINE gap)** | Both need to capture free-text input scoped to one specific row. `FormModal` has no row-context binding today (`ActionHost` keeps the clicked row in local `$state`, never passes it into the form spec) — see `RFQs.parity.md` #3 for the same finding on that screen. Even setting the engine gap aside, these create/terminate real documents (an Order on Won; a permanent lock on Lost) — the build brief calls for preferring LEDGER here rather than a mock-only approximation that could read as "this works," so neither is built. |
| 6 | Notes thread (`GetOfferNotes`/`AddOfferNote`/`DeleteOfferNote`) | **SLOT** | Small sub-feature inside the old detail modal; needs its own panel component (`slots.detail` ejection) — not built, no functional loss since the default detail panel already shows every column. |
| 7 | PDF generation (`GenerateOfferPDF` + `OpenExportedFile`) | **LEDGER / INTEG** | Bindings-only row action, same shape as invoices' PDF (#13 in `PARITY_INVOICES.md`) — cheap to add once a row action is wired to a real binding, but per the build brief no real mutations get wired in K1. Not built. |
| 8 | Re-quote (opens create form pre-filled via `SaveCostingAsOffer`, only visible when `stage==='Lost'`) | **DEFER** | Rides on #1's CostingSheet dependency; also flagged by recon as an inconsistency in the OLD screen (the primary "+ New Offer" button doesn't use this direct-create path even though it's technically available) — not something to reproduce as-is. |
| 9 | "Valid Until" cell color-codes independent of the stage badge (expired = red, expiring soon = amber) | **DONE** | `ColumnSpec.tone` on the `validityDate` column, computed from the same `effectiveStage`/`isExpiringSoon` derivation used for the badge — a genuine second signal on one fact, per recon's read of this as the richest 2-signal cell in the whole K1-A cluster. |
| 10 | Stats via stage-tab counts | **DONE** | Kernel `summary` strip (Offers / Total Value (BHD) / Won count) + a by-stage distribution bar; filter chip counts (`deriveFilterOptions`) cover the old screen's per-tab counts too. |
| 11 | Free-text search box | **DONE (added)** | Recon flagged this as unverified in the old screen's script section. The kernel `DocumentLedger` archetype always renders a search box driven by `descriptor.searchText`, so one exists regardless — sweeps offer number + customer name. |

## Reading

Every mutating capability on this screen — create, edit, Won, Lost, notes,
PDF — is explicitly LEDGER per the build brief, so what's built here is
deliberately the read/list/filter/summary spine: paging-equivalent fetch,
division + stage filters, search, the effective-stage-with-Expired
derivation, and the two-signal "Valid Until" tone. That's a real and honest
reflection of the old screen too — its own primary create/edit buttons don't
work without a ~2,000-line pricing tool this recon batch didn't cover, and
its two terminal actions (#5) hit the same row-context engine gap flagged in
`RFQs.parity.md`. Nothing here is a shortfall from scope discipline; it's the
CostingSheet dependency and the FormModal row-binding gap surfacing honestly
rather than being papered over with a mock-only stand-in.
