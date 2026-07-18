# Parity Ledger — SerialTraceScreen (old) vs SerialTrace bespoke view

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Search box + Search button (`SearchSerials(query, 200)`) | **DONE** | `SearchInput` + `Button` in a `Toolbar`; `SerialTraceViewModel.search()` (`serial-trace.svelte.ts`) drives it. |
| 2 | Enter-to-search (`Input`'s `on:keydown` → submit) | **DEFER** | `kernel/controls/SearchInput.svelte` doesn't forward extra event props today (no `...rest` spread on its `<input>`) — out of this screen's file-ownership scope to change. Click-only Search is the honest fallback; flag for whoever owns `SearchInput.svelte` next. |
| 3 | "Recently delivered" default greeting (`GetRecentlyDeliveredSerials(25)`, B10-4) | **DONE** | Same shape, `n=20`. Shown until a search runs; `vm.rows` is a `$derived` switch between `recent` and `results`, same pattern the old screen's `hasSearched` flag drove. |
| 4 | Both `SearchSerials` and `GetRecentlyDeliveredSerials` are real, working, single-call `App` bindings | **INTEG gap, by design** | Unlike `cheque-register.ts`/`data-quality.ts` (fetch wired, only mutations gapped), BOTH reads are INTEG-gapped here per the K4 brief for this screen — no account-merge complexity forced it, it's a deliberate scope call to keep bespoke-track reads landing at K5 alongside the ledger-archetype ones rather than special-casing one bespoke view early. Field mapping is fully documented in `bridge/serial-trace.ts`'s real-section header comment, so wiring at K5 is a straight `pick()` swap. |
| 5 | Columns: Serial #, Product, Status (dot + label), PO, GRN, Delivery Note, Invoice, Customer, Warranty Start, Warranty End | **DONE**, consolidated | `DataTable` columns: Serial #, Product, Stage (Badge via `StatusSpec`), Customer, **Warranty** (single Badge — see #6), PO, GRN, Delivery Note, Invoice. Two separate warranty-date columns collapse into one Badge; everything else ports 1:1. |
| 6 | Warranty-end coloring (green = valid, grey = expired) via inline `<span style>` | **DONE, upgraded (SLOT)** | `DataTable`'s built-in Badge rendering is tied to ONE `StatusSpec` per table — already spent on the Stage column here. Warranty needed an independent Badge dimension, so the Warranty column ejects its cell (`ColumnSpec.cell`, L4) to a small new `SerialWarrantyBadge.svelte`, sharing pure `warrantyLabel`/`warrantyTone` helpers with the column's declared `value` (both live in `serial-trace.svelte.ts`, ONE definition per KERNEL L2). Tone mapping preserved exactly: valid = success (was green), expired = neutral (was grey), no warranty = neutral (new — the old screen just showed a blank cell for `warranty_end_date: null`; the badge now says "No warranty" instead of nothing). |
| 7 | Deep-link buttons on PO/DN/Invoice ref columns (`navigateToDoc`, dispatches a `navigateToScreen` CustomEvent the old shell listened for) | **DEFER** | Cross-screen navigation wiring is an app-shell/router concern this kernel-lab registry (`screens/registry.ts`) doesn't have yet — plain code cells for now, same as every other ledger's reference columns (e.g. `credit-notes.descriptor.ts`'s `invoiceNumber`). Revisit once the registry grows real routing. |
| 8 | `embedded` prop (mounts standalone or inside `OperationsHub.svelte`) | **DEFER** | No Hub/embedding composition exists in this pilot yet (see the K4 census's PeopleHub/WorkHub operational-hub-primitive gap) — `SerialTrace.svelte` renders full-page only for now; adding an `embedded` prop is straightforward once a host exists. |
| 9 | Read-only, no mutations anywhere | **DONE** | Preserved exactly — `bridge/serial-trace.ts` exports only the two fetch functions, no create/update/delete of any kind. |

## Reading

This is a small, clean bespoke-on-primitives build — no new heavy primitive
needed, per the census's own verdict. The one interesting design decision is
Warranty: the old screen carried two raw date columns with inline hex
colors; collapsing them into one Badge is a genuine improvement (matches
the brief's "warranty-status(Badge, tone by warranty date)" ask), but
`DataTable`'s Badge machinery only has room for one `StatusSpec` per table,
already claimed by the lifecycle Stage column. `ColumnSpec.cell` — the L4
ejection seam the descriptor schema explicitly reserves for exactly this
("cell → panel → whole screen bespoke") — is the sanctioned way out, and
this is the first descriptor/table in the pilot to use it; `serial-trace.ts`
and `SerialWarrantyBadge.svelte` share their tone/label logic through one
pure-function pair rather than duplicating the expired-date check.
