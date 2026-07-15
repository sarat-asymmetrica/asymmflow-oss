# Parity Ledger — ExpensesScreen (old) vs Expenses descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Card/list layout (`.list-row` divs, no `<DataTable>`) | **EQUIV, fixed** | Rebuilt as a real table. The old layout was the one screen in this cluster without a `DataTable` — the census itself calls this "legacy, not intentional design." Data is tabular (entry #, description, category/vendor, date, amount, dual status); the table is a straightforward improvement, not a compromise. |
| 2 | Multi-mode hub (entries / recurring / approvals / workspace) | **ENGINE gap** | K1 builds the PRIMARY panel only — Entries. Recurring schedules, the Approvals queue (a saved-filter view), and the bank-candidate Workspace are all separate panels needing screen-level composition — same finding as K1-B synthesis #1 (3 of 7 screens in this cluster need it: Payments, ChequeRegister, Expenses). |
| 3 | Entries list (`ListExpenseEntries(status, includePaid)`) | **DONE** | `fetch()` — unpaged, `status=''` (all), `includePaid=true`, mirrors the old screen's mount call exactly. |
| 4 | Dual-status row (`status` + `payment_status`, two independent dimensions) | **ENGINE gap** | Same finding as `supplier-invoices.descriptor.ts` (`match_status`/`payment_status`): `StatusSpec` is single-field, so `status` drives the real badge column and `payment_status` renders as toned text instead of a second badge — `DataTable` only badges the column bound to the descriptor's primary `StatusSpec` (`kernel/primitives/DataTable.svelte`'s `content === 'status' && status` branch), not any column declaring `content: 'status'`. Getting this wrong would have colored the Payment column by the row's *primary* status, not its own value — verified against the actual render logic, not assumed. |
| 5 | Lowercase status vocabulary (`draft/submitted/approved/rejected/posted`) vs. TitleCase elsewhere in the cluster | **DONE**, handled correctly | `StatusSpec.tones` keys are lowercase, verified against `pkg/finance/domain.go`'s `ExpenseEntry.status` and the Wails-generated model, not assumed from pattern-matching other screens (census's own explicit warning). |
| 6 | 5-stage approval workflow (draft→submitted→approved→posted→paid) | **DONE** | Declared as `StatusSpec.transitions` (draft→submitted→{approved,rejected}→posted) and consumed via `nextStates()`, same pattern as PurchaseOrders/ChequeRegister. `payment_status` (unpaid→paid) is a separate axis, not part of this graph — see #10. |
| 7 | Submit (draft→submitted) | **DONE** | Row action, confirm + mock mutation; real = INTEG-gap naming `SubmitExpenseEntry`. |
| 8 | Approve (submitted→approved) — **fired with NO confirmation on the old screen** | **DONE, FIXED** | Every comparable approval action in this cluster (PO Approve, SupplierInvoice Approve, Cheque Cancel/Stale) gates through `confirm`. Adding it here closes the inconsistency the census flagged (gap #3) rather than reproducing a real UX/audit regression. |
| 9 | Reject (submitted→rejected) — **hardcoded reason `"Rejected from approvals queue"`, never operator-supplied** | **DONE, FIXED** | The single worst gap the census found in this cluster ("GAP — worse than DEFER," #4): the backend has a real `rejection_reason` field the old frontend never asked the operator to fill. Rebuilt as a ROW-AWARE reason `form` (`FormSpec.submit(draft, row)` receives the clicked entry) — the same shape Cancel PO / Reverse Receipt / Cancel Cheque use elsewhere in this cluster. Mock mutation; real = INTEG-gap naming `RejectExpenseEntry`. |
| 10 | Post (approved→posted, posts to GL) — **fired with NO confirmation on the old screen** | **DONE, FIXED** | Same fix as #8 — posting to the GL is exactly the kind of financial state transition the brief's anti-jank mandate calls out. |
| 11 | Delete entry (`canDeleteEntry`: not posted, `payment_status!=='paid'`) | **DONE** | Row action, confirm + mock mutation; real = INTEG-gap naming `DeleteExpenseEntry`. Visibility mirrors the old screen's guard exactly (`status !== 'posted' && paymentStatus !== 'paid'`). |
| 12 | Create Draft (`createExpenseEntry`, description + category required, plus amount/currency/VAT/cost-center/notes) | **SLOT** | Not built. Unlike Invoices' header-only create, this form has real reference-data dependencies (category/vendor pickers) and derived VAT math not yet modeled by the form archetype — same territory as Invoices #7–#10 and PurchaseOrders #9. |
| 13 | Record Payment (posted→paid, form: date/method/bank account/reference, all required) | **SLOT (financial hot-zone)** | Not built. Settles AP — genuinely comparable to SupplierInvoices' Mark Paid and SupplierPayments' Record Payment, both ledgered in their own parity docs. Needs a bank-account picker (another reference-data dependency) the form archetype doesn't model yet. |
| 14 | Import bank candidate (`createExpenseFromBankCandidate`) | **SLOT** | Belongs to the deferred Workspace panel (#2), pulls from a third data source (`listBankExpenseCandidates`). |
| 15 | Category/Vendor master-data CRUD | **SLOT** | Master data, not money — belongs to the deferred Quick-Entry disclosure panel, not the Entries ledger. |
| 16 | Generate Due Items (`generateRecurringExpenses`, bulk-creates drafts) | **SLOT** | Belongs to the deferred Recurring panel (#2). |
| 17 | Dashboard summary strip, server-backed (`getExpenseDashboardSummary()`) | **EQUIV, INTEG later** | The one screen in this cluster whose KPIs come from the server, not a client `.reduce()`. K1's `SummarySpec` computes client-side (MTD Spend, Submitted count, Approved count + status distribution) over the visible rows — matches every other ledger's summary mechanism. Swapping to the server-backed endpoint is a real INTEG enhancement once the summary engine supports a backend-binding option (K1-B synthesis #3 flags this explicitly), not a K1 blocker. |

## Reading

This screen carried the cluster's two worst "preserve the bug" traps — bare
`run` on Approve/Post and a hardcoded Reject reason — and both are fixed
here per the brief's explicit mandate, not preserved. The Reject fix is the
more consequential of the two: it turns a real audit-trail gap (a
`rejection_reason` field the backend has but the old frontend never
populated with operator input) into a working reason-capture form, using
the same ROW-AWARE FORMS mechanism `ChequeRegister.parity.md`'s Cancel
action uses. The dual-status handling (`status` badge + `payment_status`
toned text) was verified against `DataTable.svelte`'s actual render branch,
not assumed — the same care `supplier-invoices.descriptor.ts` took. Every
deep feature (Create, Record Payment, Recurring, Approvals-as-queue,
Workspace, category/vendor CRUD) stays ledgered, unbuilt, with its guard
rails intact in the old screen and undisturbed here.
