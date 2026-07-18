# Parity Ledger — GRNScreen (old) vs GRNs descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | List GRNs (`ListGRNs(limit,offset,qcStatus)`) | **DONE** | Flat `fetch()` (1000, 0, '') — matches the old screen's actual behavior: it calls the paged/filtered binding but only ever once, on mount, then filters client-side. No `fetchPage`/Load More for K1 (census: "effectively a flat load-1000, not real server-side filtering in practice"). |
| 2 | QC-status filter tabs | **DONE**/EQUIV | `options: 'derive'` chip, not static tabs — same job, and (bonus) surfaces any status value actually present in the data, including the old screen's `UNKNOWN_STATE`-class surprises, instead of a fixed 5-tab list. |
| 3 | Acceptance-rate colour-coded % cell (≥95 green / ≥80 amber / <80 red) | **DONE** | `ColumnSpec.tone`, thresholds match the old screen exactly; `acceptance_rate` is read as a 0–1 fraction and ×100'd for both display and threshold, same convention as `row.acceptance_rate * 100`. |
| 4 | Stats strip (Total GRNs, Items Received/Accepted/Rejected, Acceptance Rate, Pending QC) | **DONE**/EQUIV | `SummarySpec`: GRN count, items accepted, items rejected (amber when >0), and a *weighted* acceptance rate (Σaccepted/Σreceived, not an average-of-row-percentages) + status distribution bar. "Pending QC" isn't a separate metric — it's directly readable off the distribution bar's Pending segment. |
| 5 | "+ Receive from PO" (PO picker → per-line receive sub-form with serials; this call itself CREATES the GRN) | **SLOT** | Not built. There is no separate blank-GRN create path in the real system — the "New" action IS a receive-form, structurally identical to DeliveryNotes' fulfillment sub-form (K1-A synthesis #2). Needs a shared `FulfillmentLineEditor` ejection component; out of K1 scope by design. |
| 6 | QC Review (status select + notes, auth-gated — blocked with no logged-in user) | **SLOT + ENGINE** | Not built. Needs both an ejection form (status+notes) and an engine-level "no ghost actor" attribution guard (recon flags this as a candidate for ALL actions needing attribution, not per-screen). |
| 7 | Complete (idempotency-guarded: posts an inventory stock movement + updates PO received quantities) | **SLOT (financial hot-zone)** | Not built — deliberately. This is the highest-blast-radius action on this screen (`is_completed` gate must stay server-authoritative, not an optimistic client flip). Building even a mock version risks teaching the wrong mental model of what "Complete" is allowed to do; ledgered honestly instead of reimplemented loosely. |
| 8 | View detail modal | **EQUIV** | Kernel's default column-list side panel (no `slots.detail` override needed for K1 — GRN's fields are already flat enough that the generic detail view reads fine). |

## Reading

GRN is deliberately the thinnest ledger in this wave: every mutating capability
the old screen has (Receive, QC Review, Complete) touches either inventory
posting or an auth-gated attribution rule, and the brief is explicit that
those stay ledgered rather than rebuilt as loose `run()` calls that would
silently drop their guard rails. What K1 delivers is the full read surface —
list, search, QC-status filter, threshold-coloured acceptance cell, and a
five-signal summary strip — at or above the old screen's presentation
quality, with zero actions wired (real or mock) so there is nothing here that
could be mistaken for "GRNs can be completed from this screen today."

The mock bridge (`src/bridge/grns.ts`) is read-only for the same reason: no
`realFetch`/`mockFetch` mutation pair exists to switch, since there is
nothing to mutate yet.
