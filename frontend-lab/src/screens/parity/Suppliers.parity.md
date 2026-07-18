# Parity Ledger — SuppliersScreen (old) vs Suppliers EntityMaster descriptor

Verdicts:

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL entities at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Status tabs (All/Active/Inactive/Pending with counts) | **FIXED, not preserved** | `SupplierMaster` has NO `status` field server-side; the old screen's `Pending` tab has never had backing data (`|| 'Active'` fallback). Descriptor derives a 2-state `Active`/`Inactive` status from `is_active` — recon-K2 verdict #1, an intentional correctness fix, not a regression. |
| 2 | VAT/TRN, brands, address, bank columns on the master list | **INTEG gap (by design)** | `ListSuppliers`'s real `SELECT` deliberately excludes these for list-perf reasons — they only exist after `GetSupplierFullProfile`. The descriptor's master-list `columns` omit them entirely (mock still generates them for the row shape); they surface in `profile.sections` instead, where `mapSupplier` blanks them against real data until K5 wires the second fetch. |
| 3 | Profile KPIs (Total Purchases/Total POs/Avg PO Value/Open Issues, from `SupplierFullProfile`) | **INTEG gap** | Requires `GetSupplierFullProfile(id)`, a second fetch K2 does not wire (per the K2 brief — no engine profile-fetch added). Mock generates full adversarial values; `mapSupplier` zeroes them against real data. Wiring lands at K5. |
| 4 | 5-tab detail view (Overview/POs/Invoices/Issues/Notes) with inline-edit-toggle Overview | **ENGINE gap** | `EntityDescriptor.profile` is flat (kpis + sections), no tabs concept. Collapsed to one scrolling profile page per the K2 brief's scope discipline; a `profile.tabs` extension is an orchestrator-level decision (recon-K2 cross-screen synthesis #1), not built here. |
| 5 | Contacts strip (`SupplierContact` CRUD, `ListSupplierContacts`/`AddSupplierContact`) | **SLOT** | Nested mini-ledger, not a profile field. Same shape as Customers' contacts strip — needs a `profile.slots` extension (recon-K2 synthesis #2). Not built. |
| 6 | Issues sub-ledger (report + resolve-with-reason) | **SLOT** | Same "nested ledger inside a profile" shape as #5, plus a resolve-with-reason escalation (mirrors K1's confirm.askForReason pattern) scoped to the nested collection. Not built. |
| 7 | Notes sub-ledger (typed: general/delivery/quality/pricing) | **SLOT** | Same shape as #5/#6. Not built. |
| 8 | + New Supplier (`CreateSupplier`, `suppliers:create` gated) | **LEDGER** | Header-only create is in scope per the K1/K2 brief, but Suppliers' create form has enough fields (contact/commercial/bank) that it's deferred to the form-archetype wave rather than hand-rolled here. Not built. |
| 9 | Edit (inline-in-tab form on the detail view, not a modal) | **LEDGER** | Depends on #4's tab structure existing first. Not built. |
| 10 | Delete supplier (`DeleteSupplier`, server refuses if POs/invoices/contacts exist) | **DONE** | `ActionSpec.confirm` (danger-style confirm string) + mock mutation (removes the row from `cache`). Real binding throws an honest INTEG-gap error naming `DeleteSupplier` and documenting the referential-integrity refusal in the error text itself — the descriptor does not assume success. |
| 11 | Summary stat cards (Total Suppliers/Active/Active Rate %) + dead `Pending Invoices` count | **DONE, with a fix** | Rebuilt as the declarative `summary` strip (count + active count + active rate, computed weighted like GRNs' Acceptance Rate) plus a supplier-type distribution bar (new — wasn't in the old screen, cheap given `supplier_type` is already on the list row). The old screen's `Pending Invoices` stat was always 0 (dead field, not wired to any real fetch) — dropped, not carried forward as fake data. |
| 12 | Free-text search (name/code/contact/email) | **DONE** | `searchText` sweeps code/name/primaryContact/email. |
| 13 | Cross-screen drill-through (`startNewPO`/`openPO`/`openSupplierInvoice`) | **INTEG** | Needs the app-shell nav concept K1 already flagged (recurring across ~7 screens now per recon-K2 synthesis #3). Not built. |

## Reading

The master list + status/summary/search/delete surface is built and matches
what the real `ListSuppliers` fetch can actually provide today — no column
renders a value the real backend can't fill. The profile intentionally stops
at flat KPIs + 3 sections (Contact/Commercial/Bank Details), all sourced from
`SupplierFullProfile`'s Overview-tab fields; the two biggest deferred pieces
are structural (tabs) and collection-shaped (contacts/issues/notes), both
flagged by recon-K2 as cross-entity engine gaps rather than Suppliers-specific
misses — the same `profile.tabs`/`profile.slots` extensions would unblock
Customers' detail view too. Create/Edit and the nested sub-ledgers are
deliberately ledgered per the K2 brief's scope discipline, not missed.
