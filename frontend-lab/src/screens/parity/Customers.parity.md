# Parity Ledger — CustomerDetailView (old) vs Customers EntityMaster descriptor (K2 widen)

The master-list/hold/reactivate pilot (K1-era) already existed and is not
re-litigated here — this doc covers the K2 widen: bringing `customers.descriptor.ts`'s
profile up to `CustomerFullProfile` (`app_crm_surface.go:550`), which the original
pilot only captured a strict subset of (recon-K2 finding).

Verdicts:

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL entities at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | TRN, Industry, Relationship (years), Payment terms (days) fields (`CustomerFullProfile`) | **DONE (mock) / INTEG gap (real)** | Added to `CustomerRow` + Commercial section. `mapCustomer` blanks/zeroes them — `ListCustomers` doesn't return them, only `GetCustomerFullProfile` does, and this bridge doesn't wire that second fetch (mirrors Suppliers' identical gap). |
| 2 | Credit-blocked flag (`is_credit_blocked`) toning the Balance/Credit limit KPI danger | **ENGINE gap** | `ProfileKpiSpec` has no `tone` field (unlike `ColumnSpec`, which does) — there's no way to color a profile KPI by a computed condition today. The raw flag is surfaced honestly instead as a plain Commercial-section field (`Credit blocked: Yes/No`); the tone request itself needs a `ProfileKpiSpec.tone` extension, an orchestrator-level engine decision, not something added ad hoc here. |
| 3 | AR aging (`ar_current`/`ar_30`/`ar_60`/`ar_90`, from `ReceivablesAgingSummary`) | **DONE (mock) / INTEG gap (real)** | New "Receivables Aging" profile section, 4 money fields. Mock splits the row's `balance` across buckets (with a monster row parking 100% in 90+ to exercise a fully-overdue customer); real bridge zeroes all four pending `GetCustomerFullProfile`. |
| 4 | RFQ performance (`rfqs_floated`/`rfqs_won`/`win_rate`) | **DONE (mock) / INTEG gap (real)** | New "RFQ Performance" section (all 3 fields) plus `Win Rate %` promoted to a 4th profile KPI — same house convention as GRNs' Acceptance Rate (percentage points as a plain number, `%` lives in the label). Real bridge zeroes all three. |
| 5 | Nested collections: contacts, recent RFQs, recent orders, recent invoices, payment history | **SLOT** | Not built — same "nested CRUD/read collection inside a profile" shape as Suppliers' Contacts/Issues/Notes (recon-K2 cross-screen synthesis #2), needs a `profile.slots` extension. Out of K2 scope per the brief. |
| 6 | 5-tab detail structure (Overview/Orders/Invoices/RFQs/Notes) | **ENGINE gap** | Same `profile.tabs` gap flagged on Suppliers — collapsed to one flat scrolling profile page. Not built here; an orchestrator-level decision shared across both entities. |
| 7 | Delete customer (soft-delete, 2-step confirm) | **DEFER** | Out of this widen's scope (brief: "KEEP the existing hold/reactivate actions" — delete wasn't asked for and isn't added). |
| 8 | Summary stat strip (count/active/rate/outstanding) | **DONE** | New declarative `summary` (mirrors Suppliers/Invoices/GRNs shape): Customers count, Active count, Active Rate % (weighted, house convention), Total Outstanding (BHD, sum of `balance`), plus a status distribution bar. Wasn't in the pilot before this widen — mandatory per the K2 visual-diversity rule. |
| 9 | Existing hold/reactivate row actions + master-list columns (code/name/city/balance/status) | **DONE, unchanged** | Kept exactly as the K1-era pilot had them — this widen only adds profile depth + summary, per the brief's "KEEP the existing hold/reactivate actions + columns." |

## Reading

The widen brings the profile's KPI/section surface up to `CustomerFullProfile`'s
actual field set — nothing in the new sections renders a value the real Go
struct doesn't carry, and every profile-only field is honestly zeroed/blanked
against the real bridge (not faked) until `GetCustomerFullProfile` is wired at
K5, exactly mirroring the Suppliers profile-KPI gap. The one true engine gap
surfaced by this widen — `ProfileKpiSpec` can't tone a KPI by a computed
condition — is new information for the orchestrator: it wasn't visible from
the original 3-KPI pilot, only showed up once a boolean-gated KPI (credit
block) was in scope. Nested collections (contacts/orders/invoices/RFQs/notes)
and the 5-tab structure remain ledgered exactly as recon-K2 predicted, shared
with Suppliers rather than duplicated per-entity.
