# Parity Ledger — OpportunitiesScreen (old) vs Opportunities descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | List merges `GetRFQs` + `GetPipelineOpportunities` client-side | **DONE (fetch INTEG-gapped)** | Unlike the K1 ledgers (rfqs/purchase-orders/cheque-register), where `fetch()` is wired to the real binding, this screen's real `fetch` throws naming both bindings — the two-source merge plus row-level source-tagging (drives which delete binding applies, #6) is judged out of K4 scope per the orchestrator brief. Mock generates a plausible ~2:1 RFQ:pipeline mix with the RFQ six-stage vocabulary. |
| 2 | Detail drawer on row click | **SLOT** | Not built — K4 scope is the ledger spine; `slots.detail` is the documented ejection point for a future bespoke panel. |
| 3 | Create (customer select + project + value + notes) | **DONE**, header-only | `FormModal` screen action, mock-backed. Real `createOpportunity` throws naming `CreateRFQWithReference` (confirmed signature: `client, project, reference, value, notes` in `app_sales_pipeline.go:26`) — multi-line/product detail entry is out of scope, same call the RFQs pilot made for its own create form. |
| 4 | Plain delete (any row) | **DONE** | Confirm + mock mutation. Real throws naming `DeleteRFQ` (rfq-sourced rows) or `DeleteOpportunity` (pipeline-sourced rows) — confirmed both exist as real bindings (`app_sales_pipeline.go`). |
| 5 | Cascade delete with typed reason (destroys linked costing sheets/offers) | **DONE**, RFQ-sourced only | Row-aware reason `form` (same pattern as cheque-register's `cancelChequeForm`), gated `visible: r.source === 'rfq'`. Real throws naming `DeleteRFQWithCascade(id, boolean)` — confirmed real (`app_sales_pipeline.go`). |
| 6 | Cascade delete for pipeline-sourced Opportunities | **INTEG (backend gap)** | No `DeleteOpportunityWithCascade` (or equivalent) binding exists — confirmed by grepping `App.d.ts` for every `Opportunity`-named export. Rather than fake a cascade path or silently downgrade to plain delete, the cascade action is simply not offered for `source: 'pipeline'` rows (#5's `visible` gate) and this gap is ledgered here. A real fix needs either a new backend binding or a documented policy that pipeline Opportunities cascade-delete via a different route (e.g. deleting the linked Offer first). |
| 7 | "Start Project" handoff into WorkHub | **ENGINE gap (cross-screen nav)** | The descriptor/registry system has no navigate-to-another-screen-with-context primitive today (`ActionSpec.run` only gets `{ row, reload }`) — this is the same class of gap as RFQs' row-aware-form finding, but for navigation instead of forms. Not built; flagged for the orchestrator rather than faked with a `window.location` hack. |
| 8 | Stage summary strip (count / pipeline value / win rate) | **DONE** | `SummarySpec`: Opportunities count, Pipeline Value (BHD), Win Rate % (Won / (Won+Lost)), plus a by-stage distribution bar — same shape as `rfqs.descriptor.ts`. |
| 9 | Stage/status vocabulary across both sources | **INTEG (unverified)** | RFQ's `status` field is fully confirmed (six values, `RFQs.parity.md`). Pipeline `crm.Opportunity.stage` is confirmed to include `"Won"`/`"Lost"` (`app_sales_pipeline.go:1292,1298`) but its full enumeration wasn't traced in this recon — mock reuses the RFQ vocabulary as a stand-in. Real fetch is INTEG-gapped anyway (#1), so this doesn't block K4; flag for whoever wires K5. |

## Reading

This screen is the first ledger in the batch built on a genuinely two-source
real backend (RFQs + pipeline Opportunities), and the K4 brief's answer is to
gate the whole real side — fetch included, not just mutations — behind an
honest INTEG throw rather than half-wire one source. The two-tier delete
(#4/#5/#6) is where that source-duality actually bites: RFQ rows get a real
cascade-delete binding, pipeline rows don't, so the cascade button is gated
`visible: r.source === 'rfq'` instead of either faking a cascade for pipeline
rows or hiding cascade delete for everyone. The "Start Project" handoff (#7)
is the same row-context-across-screens gap RFQs' `FormModal` finding
surfaced for forms — worth designing once, not per-screen.
