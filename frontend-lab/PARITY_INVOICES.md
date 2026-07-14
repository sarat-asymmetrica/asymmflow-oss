# Parity Ledger — InvoicesScreen (old, 2,930 lines) vs Invoices descriptor

L6 requires judging every rebuilt screen against the old one. This is the
honest capability census. Verdicts:

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | List invoices (`ListCustomerInvoices(PAGE_SIZE, offset)` + `loadMore`) | **ENGINE** | VM fetch is load-all today (fine ≤ ~2k rows). Paged/virtualized fetch is one engine feature; every ledger inherits it. |
| 2 | Division scoping (`matchesCompany(invoice.division)`) | **DONE** | Division filter chip, derived from data; consumes the divisions registry store at INTEG (L7). |
| 3 | Status filter (`selectedFilter: All/Draft/…`) | **DONE** | Derived chips; unknown statuses surface honestly. |
| 4 | Aging-bucket pre-filter (dashboard drill prop `agingBucket`) | **ENGINE** | Descriptor/VM needs "initial query" seeding from navigation. Small: `LedgerViewModel` accepts an initial `LedgerQuery`. |
| 5 | Debounced search (`debouncedQuery`) | **EQUIV** | Instant search is fine at mock scale; debounce is a one-line VM change when real data demands it. |
| 6 | Detail via modal (`openDetailModal`) | **EQUIV** | Kernel uses a side panel, not a modal — selection stays in list context. Taste call ratified by the archetype. |
| 7 | Create invoice from order (+ delivery-note link, `GetAvailableDeliveryNotesForOrder`) | **SLOT** + form archetype | Needs the FormModal/Wizard archetype (next kernel wave), then a `slots.createForm` ejection. |
| 8 | Credit override flow (`openCreditOverride`/`submitCreditOverride`) | **SLOT** | Rides on #7's form. |
| 9 | Proforma modal + convert (`handleCreateProforma`/`handleConvertProforma`) | **SLOT** | Screen-specific flow; ejection component on the form archetype. |
| 10 | Edit invoice modal (`UpdateCustomerInvoice`) | **SLOT** + form archetype | Same dependency as #7. |
| 11 | Delete with confirm modal (`DeleteCustomerInvoice`) | **ENGINE** | Kernel needs ONE `ConfirmDialog` control; then it's a row action with `confirm:` metadata. |
| 12 | Send invoice (`SendCustomerInvoice`) | **DONE**/INTEG | Row-action pattern proven by Mark Paid; swap mock for binding. |
| 13 | Generate PDF (`GenerateInvoicePDF`) | **DONE**/INTEG | Row action calling a binding; no UI machinery needed. |
| 14 | Credit notes: list/create/apply/PDF (`ListCreditNotes`…) | **EQUIV** | That's a second ledger squatting inside the old screen. Kernel way: `credit-notes.descriptor.ts` — its own ~80-line screen, linked by action. |
| 15 | Field-visibility toggles (`fieldVisibility`) | **ENGINE** | Column show/hide belongs in the VM once, for every ledger. |
| 16 | Bank-reconciliation jump (`openBankReconciliation`) | **INTEG** | Row action that navigates; needs the app-shell nav (flip-time concern). |
| 17 | Overdue derivation (`isDateOverdue`, `parseGoDate`) | **DONE** | `format.ts`/status derivation; Go-date parsing goes in ONE bridge adapter at INTEG. |
| 18 | Error toasts (`toast.danger`) | **EQUIV** | Kernel renders inline error + Retry (Wave-10 zero-announce-toast law; old screen predates it). |
| 19 | Load-more button + `totalLoaded` counter | **ENGINE** | Same feature as #1. |
| 20 | Dev logging (`devLog`, stray `console.log`s) | **DEFER** | Kernel gets one instrumented bridge adapter at INTEG; no per-screen logging. |

## Reading

Nothing in the old screen requires abandoning the descriptor model: every gap
is either an **engine feature that pays out across all ~15 ledgers** (paging,
initial-query seeding, confirm dialog, column visibility) or an **ejection
slot** exactly where L4 predicted (create/edit/proforma forms). The riskiest
dependency is the form archetype — it is the next kernel mountain after
EntityMaster.

Flip criterion for THIS screen: #1, #4, #11, #15 landed as engine features;
form archetype shipped; slots #7–#10 built; INTEG swap. Everything else is
already at or above the old screen's behavior.
