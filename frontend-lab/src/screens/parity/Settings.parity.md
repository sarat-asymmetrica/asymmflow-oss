# Parity Ledger — SettingsScreen (old) vs Bank Accounts / Currency Rates / Business Settings (K4 split)

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked
- **RETIRE** — deliberately dropped, not carried forward

The old `SettingsScreen.svelte` (2,497 lines, 10 tabs) was a catch-all —
recon-K4 correctly classified it as "5-6 screens glued into one tabbed
shell," not one screen. This wave splits it into three kernel-sized pieces
and ledgers the rest; it does not port the tab shell.

| # | Old-screen tab / capability | Verdict | Notes |
|---|---|---|---|
| 1 | Bank Account CRUD (list/create/edit/delete) | **DONE (split)** | `bank-accounts.descriptor.ts` — standalone `DocumentLedger`. `GetAllBankAccounts` is real and wired (`bridge/bank-accounts.ts#fetchBankAccounts`); Create/Update/Delete are INTEG-gapped with named reasons (division-scoped record + encrypted IBAN/SWIFT handling this lab doesn't reproduce) rather than a naive pass-through against a FINANCIAL hot-zone record. |
| 2 | FX Rates tab (`GetActiveCurrencyRates`/`SetExchangeRate`) | **DONE (split)** | `currency-rates.descriptor.ts` — standalone ledger. `GetActiveCurrencyRates` real and wired; `SetExchangeRate` INTEG-gapped (needs a real date→`time.Time` bridge at the form layer — see `bridge/currency-rates.ts`). Row shape is "currency + rate + as-of date + source" against BHD, not a two-currency "pair" — `finance.CurrencyExchangeRate` has no pair concept, so this is a corrected mapping, not a literal port of the old field labels. |
| 3 | General app prefs + business rules (default margin %, VAT %, currency, company name) | **DONE (split)** | `BusinessSettings.svelte` + `business-settings-vm.svelte.ts` — bespoke form on primitives (Card + FormGrid). `GetSettings` is real but untyped (`Record<string, any>`); this bridge maps an ASSUMED key vocabulary (`company_name`, `base_currency`, `default_margin_percent`, `vat_rate_percent`, `fiscal_year_start_month`) that was never confirmed against the real Go handler. `UpdateSettings` is INTEG-gapped for exactly that reason — see `bridge/business-settings.ts` header. |
| 4 | Fiscal-year-start | **DONE (split, new field)** | Not confirmed present on the old screen's settings form; added here as part of the "business rules" consolidation the orchestrator asked for. Folded into #3, not a separate binding. |
| 5 | AI provider keys (AIMLAPI/OpenAI/Anthropic) + GPU detection sub-tab | **DEFER** | Credential-handling surface (recon-K4 flagged this as "treat as credential handling, consider deferring out of first-run setup"). Not built this wave — wires at K5 alongside a real secrets-storage decision, not as a plain-text form field. |
| 6 | Workspace folder paths | **DEFER** | Belongs with `SetupWizard`'s folder step (Group A, same bindings: `GetFolderPaths`/`UpdateFolderPaths`/`ValidateFolder`/`BrowseFolder`) — a duplicate folder-picker here would fork one concept into two UIs. Build once, at the Wizard. |
| 7 | Tally import (`ImportAllTallyData`/`ImportTallyInvoices`/`ImportTallyPurchases`/`ImportARDefaulters`) | **DEFER** | A file-import workflow with its own progress/log UI shape (closer to `OneDriveImportScreen`'s pattern than a settings form) — not a form field, needs its own screen at K5/K6. |
| 8 | P&L / Balance Sheet generation (`GenerateProfitAndLoss`/`GenerateBalanceSheet`) | **DEFER** | Belongs on `AccountingScreen`/`ReportsScreen` territory (recon-K4 Group C) — report generation, not a settings concern; duplicated only because the old screen was a catch-all. |
| 9 | Supabase sync config (`TestSupabaseConnection`/`GetSyncHealth`) | **DEFER** | Ops/infra concern — closer to `DeploymentHub` (already DEFER-to-K5/K6 in recon-K4) than a business-settings form. |
| 10 | "Deployment" tab (pilot-readiness stats + nav button) | **RETIRE** | Owner-ratified per the task brief: pure duplicate of `DeploymentHub`'s own summary stats plus a nav shortcut — recon-K4 flagged this exact tab for retirement independent of when DeploymentHub itself gets rebuilt. Not carried into any of the three new pieces; a future nav should route straight to DeploymentHub instead. |

## Reading

Three genuinely kernel-sized pieces came out of one 2,497-line file: two
`DocumentLedger` CRUD tables (Bank Accounts, Currency Rates) and one bespoke
form (Business Settings). All three real list-fetches (`GetAllBankAccounts`,
`GetActiveCurrencyRates`, `GetSettings`) are wired for real today; every
mutation is INTEG-gapped with a *specific* reason — division/encryption
scope on bank accounts, a `time.Time` bridge gap on FX rates, an unverified
key schema on business settings — rather than a blanket "not done yet."
Financial hot-zone data (IBAN/SWIFT, exchange rates, margin/VAT) stays
synthetic-only throughout. The other seven tabs are honestly ledgered
(five DEFER to the screen family they actually belong with, one RETIRE) —
none of them were built or faked this wave.
