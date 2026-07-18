# Costing Sheet — parity notes

**Entity:** `costing-sheet` · **Group:** Sales · **Archetype:** bespoke (K4 L-monster, largest old screen at 3026 lines)

New: `bridge/costing-sheet.ts` + `costing-sheet-vm.svelte.ts` + `CostingSheet.svelte` +
`tests/costing-sheet.test.ts` (25 tests pinning the sacred math). Full-page cost/quote
workspace: opportunity picker → costing form with the pricing waterfall (25-column
`LineItemsEditor`), revisions, save/export. `LineItemsEditor` handled the whole waterfall
(incl. two `wide` fields + readonly computed cells) with **zero new primitive work**.

## Capability census (condensed)

| Old capability | Verdict | Notes |
|---|---|---|
| Opportunity picker (merged RFQ + pipeline) + preview + Start Blank | **DONE** | Real merge; value-then-recency preview sort ported verbatim. |
| Header form + collapsible advanced/compliance block | **DONE** | `FormGrid` (kernel `k-input`/`k-field-wide` controls). |
| Line-items pricing waterfall (`calculateLineItem`) | **DONE (VERBATIM)** | `calcLine` reproduces the old math field-for-field. See Sacred Math. |
| Sheet totals (subtotal/discount/hidden/VAT/grand/cost/profit) | **DONE (VERBATIM)** | Profit/cost asymmetry preserved byte-for-byte. See below. |
| Currency FX table | **DONE** | Hardcoded `{BHD:1, EUR:0.45, USD:0.376, GBP:0.52, CHF:0.43}` — no live binding exists on the old screen either. |
| Revisions (RFQ-scoped): list, auto-select active, +New, Make Current | **DONE** | Auto-select-by-flag (not list order) verified live. |
| Save Costing / +New Revision | **INTEG** | `CreateCostingSheet`/`UpdateCostingSheet`/`CloneCostingAsNewRevision`. |
| Set active revision | **INTEG** | `SetActiveCostingRevision`. |
| Save as Offer (+ confirm-before-overwrite) | **INTEG (HOT-ZONE)** | `SaveCostingAsOffer`; confirm gate re-based on `linkedOfferNumber` (see stop-and-ask). |
| Export PDF / Excel / Open file | **INTEG** | `ExportCostingToPDF`/`ExportCostingToExcel`/`OpenExportedFile` (side-effecting → gapped per brief). |
| Terms & Conditions textarea | **DONE (simplified)** | VAT rate stamped at sheet-reset, not live-regex-patched on every VAT keystroke. Flagged. |
| Recent Sheets card | **DONE** | `GetCostingSheets`. |
| Copy-first-item-costs-to-all | **DROP** | Not in the column/action spec; flagged, not silently lost. |
| localStorage draft autosave + beforeunload guard | **DEFER** | Per explicit ruling. |
| Session-storage cross-screen handoff (pending-opportunity/offer) | **DROP** | Screen always entered fresh (picker → form). |
| "Start Fresh"/"Disconnect" wipe-confirm | **SIMPLIFIED** | Single "Change Opportunity" button back to picker (no draft to lose since autosave deferred). |
| Supplier line field | **DROP** | Dead in old screen, per ruling. |
| Customer fuzzy-match (`namesRepresentSameParty`) | **DONE (condensed)** | ~15-line port of the ~40-line engine; same technique (strip WLL/LLC, collapse whitespace, substring ≥8 chars); unit-tested against the seeded near-duplicate pair. |

## Sacred math — ported verbatim (pinned by unit tests; do NOT "fix")
`calcLine` reproduces `calculateLineItem` (old ~lines 525–582): fobBHD → freight → C&F → customs →
landed → handling → finance → totalCost → sellingPrice → **`Math.ceil` suggested price (the ONLY
rounding, never `Math.round`)** → effective price → line total.
- **customs/handling/finance** fall back to **5/4/1** at calc-time (explicit 2nd arg).
- **freight%/margin%** fall back to **0** at calc-time (NO rescue arg) — the 9/20 only seed a fresh blank
  line. This is a genuine old-app inconsistency, **preserved verbatim + unit-tested both ways** so no future
  edit silently changes it. Stop-and-ask #1.
Sheet totals: `netAmount = max(0, subtotal − max(0,discount))`; `vat = netAmount × clamp(vatRate)/100`;
`totalCost = Σ(line.totalCost × max(1,qty)) + max(0,hiddenCharges)`; `profit = netAmount − totalCost`.

## Profit/cost asymmetry — preserved byte-for-byte (do NOT "fix")
`hiddenCharges` → **cost only**; `discount`/`VAT` → **revenue only**. Pinned by unit tests asserting
hiddenCharges never moves netAmount/grandTotal and discount/VAT never move totalCost. Stop-and-ask #2.

## INTEG ledger
FETCH wired real: GetRFQs+GetPipelineOpportunities (merged), GetOpportunityLineItems, GetCostingsByRFQ,
ListCustomers, GetPreparedByOptions, GetSettings, GetCostingSheets.
MUTATION INTEG-gapped: CreateCostingSheet, UpdateCostingSheet, CloneCostingAsNewRevision,
SetActiveCostingRevision, SaveCostingAsOffer (HOT), ExportCostingToPDF, ExportCostingToExcel, OpenExportedFile.
Division vocabulary via `costingDivisionOptions()` helper (mirrors mock.ts) — never a hardcoded literal (L7).

## Live-verification bugs caught + fixed (by the build agent)
1. Mock RFQ ids were prefixed strings (`rfq-6`) but the VM does `Number(opp.id)` for `GetCostingsByRFQ`/
   `CreateCostingSheet` → `Number('rfq-6')=NaN` silently broke all revisions in mock/demo. Fixed to plain numeric strings.
2. The 100-line stress opportunity's id was unreachable in the generator; retargeted.

## Orchestrator notes
- Form controls refactored to kernel `k-field`/`k-input`/`k-field-wide` classes (L1/L2), consistent with the batch.
- Adversarial mock verified LIVE (not just present): 60 opportunities (value=0, 200-char/empty/RTL names, out-of-range year),
  a 100-line stress sheet hitting the maxRows cap with no NaN, revisions (0-rev, active-not-first auto-select, malformed-JSON
  caught, offer-linked → overwrite-confirm), always-rejecting Settings → VAT 10%/margin 20% fallback, near-duplicate customers.

## Stop-and-asks (flagged, not fixed)
1. Freight%/margin% calc-time fallback = 0 (not 9/20) — genuine old-app inconsistency, ported + tested. Correct for consistency?
2. Profit/cost asymmetry — preserved per instruction; policy question surfaced.
3. Save-as-Offer overwrite guard re-based on `linkedOfferNumber` (dropped the session-storage `sourceOfferId` cross-screen restore). Acceptable, or restore the original flow at K5?
4. Customer near-duplicate matcher is a condensed (not byte-identical) port.

## Kernel notes (no primitive fixes needed)
- `LineColumn.content` has no `'percent'` variant (percent columns are editable numeric inputs here, so unaffected).
- `ColumnSpec` (DataTable) has no `align` override — worked around on the opportunities "Value" column with a formatted string. Cosmetic.
