# Parity Ledger — PaymentsScreen (old) vs Payments descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

The old screen is two co-located sub-ledgers (Receipts + Payment History).
Per the build brief's multi-panel rule, this descriptor builds the PRIMARY
ledger — Receipts, the one AR money-in sub-ledger — and ledgers the rest.

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Two co-located ledgers on one screen (Receipts + Payment History) | **ENGINE gap** | `LedgerDescriptor` is single-table (recon K1-B synthesis #1 — 5 of 7 screens in this cluster need it). Payment History (`GetAllPayments`, paged, PAGE_SIZE=50) is not built here; it needs either a screen-level layout composing two descriptors, or a "co-located ledgers" archetype variant. |
| 2 | List receipts (`ListCustomerReceipts(limit, offset)`, paged) | **DONE** | `fetchPage`/`pageSize` (50, matching `RECEIPT_PAGE_SIZE`) — receipts were already paged correctly in the old screen, unlike Credit Notes. |
| 3 | Division scoping (silent `matchesCompany()`) | **DONE** | Division filter chip, derived from data; consumes the divisions registry store at INTEG (L7), same pattern as every other screen in this cluster. |
| 4 | Status filter (OnAccount/PartiallyApplied/Applied/Reversed) | **DONE** | Derived chips; unknown statuses surface honestly (a synthetic `UNKNOWN_STATE` row is woven into the mock). |
| 5 | Unapplied balance highlighted (bold + amber when > 0.001 BHD) | **DONE** | `ColumnSpec.tone` on the Unapplied column, same threshold (0.001) as the old screen's `PAID_TOLERANCE_BHD`. Carried into the summary strip too (Unapplied metric turns amber when the visible-row total exceeds the tolerance). |
| 6 | Record Receipt (2-mode create: apply-now / on-account) | **SLOT (financial hot-zone)** | Not built. The ONE AR money-in creation path (recon K1-B financial-hot-zone rollup) — a complex form (customer/invoice picker, apply-now vs on-account mode) squarely in "form archetype not yet built for this shape" territory, same class as Invoices #7-#10. |
| 7 | Apply Unapplied Balance (invoice picker, `ApplyCustomerReceiptToInvoice`) | **SLOT** | Not built. Needs a pre-scoped invoice picker; ejection-component territory, not a generic form field list. |
| 8 | Reverse Receipt (confirm + reason, zero-application-only) | **DONE** | Row-aware `form` action (reason textarea) — the guard `applied_amount_bhd<=0.001 && status!=='Reversed'` is preserved verbatim as `ActionSpec.visible`. Mock mutation zeroes both amounts and flips status; real mutation throws an honest INTEG-gap naming `ReverseCustomerReceipt`. |
| 9 | Edit / Delete legacy Payment (Payment History rows) | **DEFER** | Lives on the ledgered Payment History panel (#1); no row shape to act on until that panel exists. |
| 10 | Open Bank Recon (nav) | **DEFER** | Cross-screen navigation glue; revisits once the app-shell nav model exists (same as Orders/Invoices bank-recon/nav findings). |
| 11 | KPI strip (Total Collected YTD, This Month, Avg Days, Unapplied) | **DONE**-shape | `summary`: Receipts count, Total Received (BHD), Applied (BHD), Unapplied (BHD, tone-driven) + a 4-state status distribution bar. Computed over visible rows, not YTD/this-month buckets — a straight port of the money totals, not the time-windowed variants; those would need the date-range `FilterSpec` primitive recon K1-B flagged for SupplierInvoices. |
| 12 | Client-side settlement-tolerance guard (`PAID_TOLERANCE_BHD = 0.001`) | **ENGINE** | Duplicated verbatim in SupplierPayments as `OVERPAY_TOLERANCE_BHD` (recon K1-B synthesis #4) — both this descriptor's `visible`/`tone` predicates and SupplierPayments' equivalent should eventually import one shared bridge-layer constant instead of each screen hardcoding `0.001`. |

## Reading

The Receipts sub-ledger — the actual AR money-in ledger — is fully built:
list, paging (fixing nothing, since receipts were already paged), filters,
summary, and the one row action safe enough to build under K1 scope
discipline (Reverse, zero-application-only, reason-captured). Everything
that creates or moves money against an invoice (Record Receipt, Apply
Unapplied Balance) stays ledgered with its guard rails named, not
approximated. Payment History is entirely deferred to the multi-panel
ENGINE gap — building a second, unrelated `GetAllPayments`-backed table here
would just be a second unreviewed ledger, not a step toward the real fix
(a screen-level composition primitive that 5 of 7 screens in this cluster
need).
