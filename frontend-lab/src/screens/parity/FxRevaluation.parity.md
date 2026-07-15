# Parity Ledger — FXRevaluationScreen (old) vs FX Revaluation descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Three tabs (Exposure / Revaluations / Rates), each a SEPARATE fetch (`GetFXExposureReport`, `GetFXRevaluations(accountId)`, `GetLatestFXRate`/`GetAllFXRates`) | **ENGINE gap** | K4 builds the PRIMARY ledger only — Revaluations, merged across every active foreign-currency account (same account-merge trade-off as `cheque-register.ts`'s `realFetchAll`). Exposure is a genuinely different aggregate shape (per-currency rollup, not per-run history) and Rates is a genuinely different entity (rate history, not revaluation runs) — neither is a client-side filter of the Revaluations feed. Needs a multi-panel screen layout composing descriptors, same class of gap as ChequeRegister #1. |
| 2 | Account selector driving the Revaluations tab | **EQUIV** | The old screen required picking one foreign account before it would show revaluations. K4's ledger shows every foreign account's revaluation history at once (Account column + a Currency filter chip), same "merge instead of gate" call `cheque-register.descriptor.ts` made for Outstanding cheques — arguably better since nothing is hidden by default. |
| 3 | Revaluation history list (`GetFXRevaluations`, per account) | **DONE** | `fetch()` — merged across all active non-BHD accounts, matching the real binding's per-account shape. |
| 4 | Calculate for Selected Account / Revalue All (`CalculateFXRevaluation`, `RevalueAllForeignAccounts`) | **SLOT (financial hot-zone)** | Not built. These CREATE new Draft revaluation rows (a batch mutation with an as-of-date input), not a status transition on an existing row — genuinely a screen-level create action needing its own date-picker form, ledgered here rather than faked as a plain confirm. |
| 5 | Post Revaluation (row, gated on not-yet-posted, requires authenticated operator) | **DONE** | Row action, `confirm` (mirrors the old screen's `confirmAndPost` message including the signed amount/account/date). Mock mutation; real = INTEG-gap naming `PostFXRevaluation`. Note: reading `fx_revaluation_service.go`, the server actually re-resolves identity itself and ignores the client-supplied user param (Wave 9.3 B2) — technically wireable without a lab identity layer — but K4 stays consistent with the campaign-wide "K4 lab doesn't wire real mutations" convention (ChequeRegister #5), so it's INTEG-gapped anyway. |
| 6 | Reverse (binding exists — `ReverseRevaluation` — but the old screen has **no UI** for it) | **DONE, new** | Row-aware reason `form`, same ROW-AWARE FORMS pattern as ChequeRegister's Cancel. This is a genuine addition beyond the old screen's UI, not a straight port — the binding was real and unused. Mock mutation; real = INTEG-gap naming `ReverseRevaluation`. |
| 7 | "Reversed" as a visual status | **CORRECTED, not ported** | The old screen's `FXRevaluation` interface has no `is_reversed`/status-enum field, and reading `pkg/finance/fx/fx.go`'s `Reverse()` directly confirms why: reversing a POSTED row does not flip that row's own status — it INSERTS a new Posted row with negated rates/gain-loss (an accounting reversing entry, not a state change), and reversing a DRAFT row just deletes it. So this descriptor models status as two-state (Draft/Posted), not three — inventing a "Reversed" badge would show something the schema can't actually represent. `bridge/fx-revaluation.ts`'s mock `mockReverse` mirrors the real insert/delete behavior exactly so a reversed-and-reloaded ledger looks the same shape it would against the real backend. |
| 8 | Update Rate modal (`GetLatestFXRate`/`CreateFXRate`, per currency, live current-rate preview) | **SLOT** | Belongs to the Rates tab (#1), not the Revaluations ledger — ledgered together rather than built as a bare create form disconnected from its tab. |
| 9 | KPI strip (FX Exposure, YTD Gain, YTD Loss, Net Unposted) | **DONE**, redesigned | `SummarySpec`: Total Exposure (BHD, latest-run-per-account, not a naive sum — see the descriptor's `latestPerAccount` comment), Total Unrealized G/L (BHD, tone-flipped success/danger), Revaluation Runs (count) + a Draft/Posted status distribution bar. YTD Gain/Loss split (`GetFXGainLossSummary`, calendar-year scoped) isn't reproduced — it's a server-computed aggregate over ALL history, not something the visible-rows reduction this pilot's summary strip does can approximate honestly; ledgered as an ENGINE gap alongside #1 if wanted later. |
| 10 | Row-click opens a read-only details modal; Post lives inside it, not inline (B3a design intent: "row-click = read-only view, Post is a separate explicit action") | **EQUIV** | The DocumentLedger archetype's row-select detail panel (all columns + gated row actions) already satisfies this shape generically — clicking a row shows detail, Post/Reverse render as explicit buttons below it, never fired by the click itself. |
| 11 | As-of date picker (governs Calculate/Revalue All) | **DEFER** | Belongs to the ledgered create actions (#4), not the ledger itself. |

## Reading

The old screen's three-tab shape is the same "genuinely separate fetches,
not one dataset filtered three ways" pattern recon K1-B flagged repeatedly
(ChequeRegister #1, this screen's own census entry) — Revaluations is the
one sub-view K4 builds as the PRIMARY ledger, merged across every active
foreign-currency account the same way `cheque-register.ts` merges
Outstanding cheques across bank accounts.

The one finding worth flagging beyond the brief: the old screen's own
`FXRevaluation` TypeScript interface (and the real Go struct behind it)
never had a three-state Draft/Posted/Reversed status — `pkg/finance/fx/fx.go`
confirms `Reverse()` inserts a new row rather than relabeling the old one.
Building a fictional "Reversed" badge for the original row would be
prettier than the old screen but wrong. This descriptor's two-state
`FX_STATUS_TRANSITIONS` (`Draft: ['Posted']`, `Posted: []`) and the mock's
insert/delete `mockReverse` both stay honest to that reality — the same
"fix, don't blindly preserve" call `credit-notes.descriptor.ts` made when
it added a confirm the old screen's Apply lacked.
