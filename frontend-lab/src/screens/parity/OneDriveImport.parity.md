# Parity Ledger — OneDriveImportService (Go) vs OneDriveImport bespoke view

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

There is no old frontend screen for this capability — `onedrive_import_service.go`
was unrouted. This ledger is a capability census against the Go service (structs
+ `DetectOneDrivePath` / `ValidateOneDrivePath` / `ScanOneDrivePaths` /
`ImportOneDriveDeals`) rather than a before/after screen diff.

| # | Go capability | Verdict | Notes |
|---|---|---|---|
| 1 | `DetectOneDrivePath()` — suggests a root path | **DONE** | `OneDriveImportViewModel.detectInitialPath()` runs once on mount (view's `$effect`, same pattern as `SerialTrace.loadRecent()`), prefills path 0 if empty, then auto-validates it. Detection is advisory-only: a failure is swallowed, never surfaced as a blocking error — the user can always type a path by hand. |
| 2 | `ValidateOneDrivePath(path)` — `{valid, estimated_deals?, path?, error?}` | **DONE** | Step 0: one row per path (`k-input` + Validate button), a `Badge` shows `~N deal folders detected` (success) or the error text (danger). `canAdvance` on step 0 = at least one path with `valid === true`. |
| 3 | Multiple root paths | **DONE** | "Add another path" / per-row "Remove" (min 1 row). Only rows with `valid === true` feed the scan (`vm.validPaths`). |
| 4 | `ScanOneDrivePaths(paths)` → `OneDriveScanResult{deals, total_folders, total_files, errors}` | **DONE** | Fires on the 0→1 `Wizard` transition (`goNext()`), not on every render — mirrors the brief's "on entering, call ScanOneDrivePaths". `errors` (non-empty) surfaces via `CalloutWidget` (tone `warning`) above the deals table. `total_folders`/`total_files` aren't separately displayed (redundant with the table + per-row file counts) — **DEFER**, cosmetic only. |
| 5 | `DiscoveredDeal` fields: folder name, instrument type, year hint, file count | **DONE** | `DataTable` columns: Deal Folder (name, grows), Instrument (`instrumentType \|\| 'Unknown'` — covers the UNKNOWN/empty-keyword adversary), Year (`yearHint \|\| '—'`), Files (`files.length`, quantity-formatted). |
| 6 | Per-deal include/exclude decision | **SLOT** | `OneDriveIncludeCell.svelte` — a checkbox `ColumnSpec.cell` ejection (L4) that mutates `row.selected` directly. `vm.deals` is `$state`-backed so the mutation is reactive; `vm.includedDeals`/`canAdvance`/`nextLabel` all recompute live. Default: included iff the deal has ≥1 `customer_matches` entry. |
| 7 | Per-deal customer confirmation from `customer_matches` | **SLOT** | `OneDriveCustomerCell.svelte` — a `<select class="k-input">` `ColumnSpec.cell` ejection, options sourced from **that deal's own** `customerMatches` (never a shared/global customer list) plus a leading "— skip / unmatched —" (`''`). Label format: `"{business_name} ({score.toFixed(2)})"` e.g. `Gulf Fabrication W.L.L. (0.82)`. Mutates `row.confirmedCustomerId` directly, same reactive contract as #6. Default: the highest-score match (mock pre-sorts `customerMatches` descending, mirroring the Go service's `sort.Slice`). |
| 8 | Why `ColumnSpec.cell`, not a new kernel API | **DONE (design decision)** | `DataTable`'s built-in interactivity is zero — every non-`cell` column is inert display. Two independently-stateful, per-row interactive controls (checkbox + select, each mutating a different field) is exactly the "cell → column → panel → whole screen bespoke" ejection ladder `descriptor.ts` documents (KERNEL L4). No prop was added to `DataTable`/`ColumnSpec` — both cells are ordinary `Component<{ row }>`s living flat in `src/screens/`, scanned by the L1/L2 tripwires like `SerialWarrantyBadge.svelte`, and both stay CSS-clean (`OneDriveIncludeCell` only centers itself via a wrapper `text-align: center`; `OneDriveCustomerCell` has no `<style>` block at all, just the kernel `k-input` class). A `ColumnSpec.rowAction` (button + onClick) candidate stays OPEN for a future button-style consumer (DeploymentHub queue retry) — it is a poor fit for stateful checkbox+select, so it was not built for one screen. |
| 9 | `canAdvance` on step 1 (review) | **DONE** | At least one deal with `selected && confirmedCustomerId` truthy. `nextLabel` = `Import N Deal(s)` where N = that count, live. |
| 10 | `ImportOneDriveDeals(deals)` → `OneDriveImportResult[]` | **DONE** | Fires on the 1→2 `Wizard` transition. Only `vm.includedDeals` (selected + confirmed) are sent — mirrors the Go function's own defensive skip of unconfirmed deals (`deal.ConfirmedCustomerID == ""` → `success:false, message:"skipped: no customer confirmed"`), reproduced in the mock too for a deal that somehow reaches import without a confirmed customer. |
| 11 | Per-deal result: success/fail, offer id, costing sheets imported, PDFs queued, message | **DONE** | Step-2 `DataTable`: Deal (joined back from `vm.deals` by `dealLocalId`, since `OneDriveImportResult` doesn't carry the folder name), Result (`StatusSpec` Badge, success=success/failed=danger tone), Offer, Costing Sheets, PDFs Queued, Message. Headline: `"X of Y deals imported successfully."` |
| 12 | "Start Over" | **DONE** | A plain `Button` in the step-2 content (not `Wizard`'s own Back — that would just page to step 1, not reset). Resets to step 0, drops scan/import state, **keeps** validated paths (no need to retype). |
| 13 | Next hidden/disabled + busy on the final step | **DONE** | `canAdvance` is hard-`false` whenever `currentIndex === 2` (terminal), so `Wizard`'s Next stays disabled regardless of `busy`. `vm.busy` (`detecting \|\| scanning \|\| importing`) drives `Wizard`'s `busy` prop, disabling both Back/Next during any bridge call — including the initial mount-time auto-detect, closing a click-through-before-ready race. |
| 14 | `ConfirmOneDriveDeal(localID, customerID)` | **DEFER, dead per brief** | Server-side validation call the Go comment marks as legacy ("the frontend holds all scan state"); the brief calls it dead. Not modeled — the frontend already validates customer selection client-side against `customerMatches`. |
| 15 | `DetectOneDrivePath` / `ValidateOneDrivePath` / `ScanOneDrivePaths` / `ImportOneDriveDeals` real bindings | **INTEG gap, by design (all four)** | Per the K5 brief, this screen runs entirely on the mock — no `wailsjs` imports anywhere in `bridge/onedrive-import.ts`. Each `real*` function throws `new Error('INTEG gap: <BindingName> — wires at INTEG')`; the field-mapping comment at the top of the bridge file documents the exact snake_case→camelCase translation so wiring later is a straight `pick()` swap, no reshaping. Note: Validate/Scan are read-only (filesystem stat/walk); Import creates offers in the DB — the DB-touching one is the reason the whole screen defers to the owner-gated INTEG pass. |

## Adversarial mock coverage (`bridge/onedrive-import.ts`)

Seeded LCG (`lcg(20260715)`), 18 deterministic deal specs (`DEAL_SPECS`), covering:
- 0-match deals (5), a single clear high-confidence match (7), a 3-way near-tie
  ambiguous match (1) and 2-way near-ties (3)
- an exact 200-char unbroken folder name (verified `.length === 200`)
- an empty/whitespace-only folder name (`'   '`)
- RTL text (`مشروع الغاز الطبيعي 2024`) and mixed RTL+LTR
  (`GULF INTERNATIONAL PROJECT مشروع 2025`)
- a huge file count (480) and a zero-file deal
- an UNKNOWN/empty instrument type (folder name with no keyword match)
- several empty year hints (no `20XX` substring in the folder name)
- a deal with a costing sheet but zero customer matches (can never be
  auto-included — demonstrates `canAdvance`'s "confirmed customer" gate)

The scan mock regenerates deals fresh each call (deterministic seed → identical
data) rather than caching, so a Start-Over → re-scan restores the default
selections instead of reusing the in-place mutations the cells made on the prior
pass. `ValidateOneDrivePath` mock: empty/whitespace → `"path is empty"`; a string
with no path separator or drive letter → `"path is not a directory"`; otherwise
valid with `estimatedDeals = 1 + (path.length % 24)`. `ImportOneDriveDeals` mock:
two deterministic, `localId`-keyed failures (`deal-6`, `deal-16` — independent of
which subset a user chooses to import) with realistic synthetic failure messages;
everything else succeeds with `costingSheetsImported`/`pdfsQueued` computed from
that deal's own `files` array. All customer names are invented (`Gulf Fabrication
W.L.L.`, `Northgrid Industrial Holdings`, `National Petroleum Upstream Ltd.`, one
Arabic synthetic name, …) — SYNTHETIC_IDENTITY-compliant, no real company/TRN/
person data.

## Gate note

The automated Playwright gate (`tests/gate.mjs`) drives each screen to its
default view and checks layout at 1440/420. For a Wizard, that is **step 0
(configure paths) only** — the gate does not click Next through the scan/import
steps. Steps 1 (deals `DataTable`, incl. the 200-char folder name + RTL rows) and
2 (results `DataTable`) were layout-verified with a one-off driver script that
pages the wizard forward and runs `detectLayoutViolations` at 1440 and 420 —
clean at both, both steps.

## Kernel gaps / gotchas noted

- `DataTable` applies `style:text-align` only to its non-`cell` branch; a
  `cell`-ejected `<td>` gets no alignment, so a centered control must center
  itself inside the cell component (`OneDriveIncludeCell` wraps its checkbox in a
  `text-align:center` div). Candidate: pass `col.content` alignment through to the
  `cell` branch too — low priority, one consumer.
- The `ColumnSpec.rowAction` candidate (button + predicate + onClick) remains
  unbuilt — see row #8. Two consumers now want *some* per-row interactivity
  (OneDrive here via `cell`, DeploymentHub queue retry) but they want different
  shapes (stateful controls vs a button), so no single API is obviously right yet.
