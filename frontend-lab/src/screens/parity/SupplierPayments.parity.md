# Parity Ledger — SupplierPaymentsScreen (old) vs Supplier Payments descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Two-source merged ledger (`GetAllSupplierPayments()` real rows + synthetic Expense-settlement rows from `listExpenseEntries`, tagged by `source`) | **DONE (mock)** / **INTEG (real)** | Per the brief, the merge is a bridge-layer concern, not a kernel one: `bridge/supplier-payments.ts`'s mock `generate()` builds and tags both kinds, merges, and sorts newest-first. The real adapter maps ONLY `GetAllSupplierPayments()` — composing the Expense side needs a second, unrelated fetch (`listExpenseEntries` via the Expenses API wrapper) and is left as an honest INTEG gap rather than half-wired. |
| 2 | Source badge (`source` column, Expense=amber/Supplier Invoice=indigo) | **DONE**/EQUIV | No formal status field exists on this ledger (census confirms it), so `source` doubles as the descriptor's `StatusSpec` — gets badge rendering, a filter chip, and the summary distribution bar for free. Tones: Supplier Invoice=info, Expense=warning (kernel palette, not the old screen's literal indigo/amber hexes). |
| 3 | Source filter buttons (all/supplier/expense) | **DONE**/EQUIV | `options:'derive'` chip, count-in-chip is automatic (`deriveFilterOptions`), replacing the old screen's hand-rolled `All (N) / Supplier Payments (N) / Expense Settlements (N)` buttons with the same information. |
| 4 | Summary counts (total paid, outstanding count, overdue count — shown as 3 bare numbers, not styled) | **DONE**/EQUIV, improved | `SummarySpec`: count, Total Paid (BHD), Expense Settlements count (amber if >0) + by-source distribution bar — this screen had the least visual differentiation in the whole cluster per the census; it now has the same KPI-strip treatment as Payments/PurchaseOrders/GRNs. |
| 5 | Record Payment (FX-aware: invoice picker, exchange-rate field only shown when currency≠BHD, derived BHD preview, posts `amount_bhd = amount × rate`) | **SLOT (financial hot-zone)** | Not built. Same territory as PurchaseOrders'/SupplierInvoices' multi-currency create forms — needs the form archetype's derived-field support. Wave 9.3's authorized posting-change (`amount_bhd = amount × rate` in one write) stays untouched/unreimplemented, not silently reproduced. |
| 6 | Edit (invoice linkage LOCKED "for audit safety"; amount/method/date/reference editable; non-expense rows only) | **SLOT + ENGINE candidate** | Not built. Same "descriptive-only edit with locked fields" pattern as SupplierInvoices #11 and Payments' edit-mode — 3 of the K1-B cluster's 7 screens want a declared locked-field concept on `FormSpec`. |
| 7 | Delete (2-click inline confirm, `canDelete`, non-expense rows only) | **DONE**/EQUIV | Built as a row action: `ActionSpec.confirm` + `ConfirmDialog` (the kernel's single confirm pattern, replacing the old screen's inline 2-click toggle — the same "pick ONE confirm UX" cleanup PaymentsScreen's census flagged). `visible: r.source === 'Supplier Invoice'` reproduces the `isExpenseSettlement` guard exactly — expense rows never show a Delete action. Mock mutation removes the row from `cache`; real = INTEG-gap naming `DeleteSupplierPayment`, with the same source-scoping caveat spelled out in the error message. |
| 8 | Client-computed outstanding balance (`invoiceOutstanding()`, summed from prior payments — no dedicated backend endpoint) | **ENGINE candidate** | Not surfaced. Same computation shape as Payments' `outstanding_bhd` (which the backend DOES supply directly there) — a shared "outstanding balance" derivation belongs at the bridge layer once a real endpoint exists uniformly; currently INTEG-adjacent (no backend support), not a UI gap. |
| 9 | Overpay guard (`OVERPAY_TOLERANCE_BHD`, duplicate of Payments' `PAID_TOLERANCE_BHD`, both `0.001`) | **ENGINE** | N/A here since Record Payment isn't built (#5), but flagged for the orchestrator: this is the same duplicated money-tolerance constant K1-A/K1-B's synthesis calls out — belongs in a shared bridge-layer money util, not reintroduced per-screen when #5 eventually gets built. |
| 10 | Open Bank Recon (nav-only, no binding) | **DEFER** | Cross-screen navigation glue; revisits once the app-shell nav model exists. |

## Reading

This is the one screen in the K1-B Finance pair with a real action built:
Delete, scoped exactly to the old screen's `isExpenseSettlement` guard so an
Expense-settlement row can never appear to be deletable from here (matching
the old screen's own "managed from Expenses" boundary). Everything else that
touches money — Record Payment's FX math, the locked-linkage Edit — stays
ledgered, since building either loosely would either drop the FX-derived-BHD
posting guard or the audit-safety linkage lock.

The two-source merge is the interesting engineering call: rather than wait
for a kernel-level "multi-origin row" concept (the census's ENGINE-gap
framing), the brief's own guidance to do it in the bridge's `fetch()` proved
sufficient — mock tags and merges both kinds identically to how the real
`source` discriminator will work once the Expense side is wired at INTEG.
