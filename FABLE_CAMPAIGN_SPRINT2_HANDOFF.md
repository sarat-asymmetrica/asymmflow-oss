# Campaign — Frontend Kernel: Sprint 2 Handoff (for a fresh Opus 4.8 orchestrator)

**You are the incoming orchestrator + technical lead** for the second sprint of the
frontend-kernel full migration. Sprint 1 (this handoff's author) built the kernel and
migrated 36 screens across waves K1–K4-partial. Your job: finish K4's L-monsters +
deferred screens, then K5 (app shell + INTEG) and K6 (the flip). Same operating model:
**Sonnet 5 agents code, you gate every wave and fix what they miss.** Enforce visual
consistency; don't inherit the old card-heavy jank; inject tasteful data-viz diversity.

**Read FIRST, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` (repo root) →
`frontend-lab/KERNEL.md` (the constitution — 5 pillars, laws L1–L7) →
`frontend-lab/PARITY_INVOICES.md` (parity-ledger method) → the ORIGINAL campaign spec
`FABLE_CAMPAIGN_FRONTEND_KERNEL.md` → `FABLE_KERNEL_CAMPAIGN_LOG.md` (the durable
running log — every ruling, gap, and gotcha is there) → the K1/K2/K3 wave reports
(`FABLE_WAVE_K{1,2,3}_REPORT.md`) → this file.

**Branch:** `exp/frontend-kernel` (worktree `asymmflow-lab`, LOCAL-ONLY — never pushed).
`wails.json` already points at `frontend-lab`. Old `frontend/` stays read-only reference
until the K6 flip. HEAD at handoff: `f4cf526`.

---

## 1. State of the world (verified green at handoff, commit f4cf526)

- **36 product screens migrated** (+ Showcase dev sink), rendering from the kernel.
  Gates green: `npm run check` 0/0 (289 files) · `npm run test` 26 passing · `npm run
  build` clean · **layout-detector CLEAN on every product screen at 1440 + 420**.
- **4 archetypes:** `DocumentLedger`, `EntityMaster`, `Hub`, + `FormModal`/`ActionHost`
  (one action path). All in `frontend-lab/src/kernel/archetypes/`.
- **Primitives** (`kernel/primitives/`): PageShell, Stack, Row, Grid, Card, Scroll,
  Toolbar, FormGrid, DataTable, Modal, **LedgerSummary**, **BalanceComparisonPanel**.
- **Data-viz widget library** (`kernel/widgets/`): distribution (h/v bar), ranked-bar-list,
  stat-tile-grid, list, activity-feed, callout, comparison-bars, **donut** (SVG). No chart
  dependency. Categorical palette `--k-series-1..6` (CVD-validated). Status tones
  `--k-tone-*` (reserved — never a series slot).
- **Controls** (`kernel/controls/`): Button, SearchInput, FilterChips (with counts),
  Badge, EmptyState, ConfirmDialog, **RangeSlider**.
- **Engine features** (all descriptor-declared): `summary` strip, `ColumnSpec.tone`,
  `ColumnSpec.cell` (L4 cell ejection), `StatusSpec.transitions` + `nextStates()`,
  **row-aware forms** (`FormSpec.initial(row?)`/`submit(draft,row?)`),
  `ProfileKpiSpec.tone`, Hub `hasContent` (hides empty widgets), Hub `period` selector,
  drill-down nav (`NavIntent{key,query}` seeds `initialQuery` into ledgers — PROVEN live).
- **Bridge architecture:** per-entity `bridge/<entity>.ts` (types + mock + real + `pick`
  switch). `bridge/runtime.ts` (`usingWails`/`pick`), `bridge/map.ts` (goDate/str/num).
  MOCK is adversarial-by-doctrine (200-char names, 12-digit + 0.001 amounts, RTL, empties,
  UNKNOWN statuses, 200–500 rows, deterministic LCG). REAL adapters: fetch wired where the
  binding is a simple single call; **all mutations + complex/multi-source fetches throw an
  honest `INTEG gap: <BindingName> — wires at K5` error** naming the real binding.
- **Registry** (`screens/registry.ts`): the ONE `screens[]` list → grouped sidebar.
  ORCHESTRATOR-OWNED merge point — do NOT let build agents edit it; collect their registry
  lines and wire them yourself (Sprint 1 had two concurrent-edit collisions this way).
- **Harness nav** collapses to an off-canvas overlay ≤720px so product screens get full
  viewport width (the detector measures true screen width). App.svelte renders by archetype.

## 2. The parity-ledger method (non-negotiable, the campaign centerpiece)

Every rebuilt screen gets `screens/parity/<Screen>.parity.md` — an honest capability
census vs the old screen (verdicts DONE/EQUIV/ENGINE/SLOT/INTEG/DEFER). 32 exist. **The
harness blocks SUBAGENTS from writing `*.parity.md` files** (pattern-matched as reports) —
inconsistently. When an agent is blocked, have it return the doc content in its final
message and YOU write the file. Financial-semantics changes are stop-and-ask, never a
judgment call; hot-zone actions are LEDGERED (real = INTEG-gap), never loosely rebuilt.

## 3. Gate bar (every wave end, enforce ALL)

`npm run check` 0/0 · `npm run test` all green · `npm run build` clean · **layout-detector
zero-violation at 1440 AND 420** (drive it via Playwright: click each screen, select a row
to open detail panels, run the `detectLayoutViolations` logic from
`frontend-lab/tests/layout-detector.mjs`) · per-screen parity doc honest · visually eyeball
new screens (screenshot at 1440) · verify screens render DATA, not error/empty states.

## 4. Remaining work

### K4 — the L-monsters (build order by tractability; each is bespoke-on-primitives)
1. **AccountingScreen** (2098 lines) — GL / chart-of-accounts / journal-entry / financial-
   reports console. No fundamentally new primitive (left-nav view switcher on PageShell +
   DataTable + FormModal voucher entry with repeating debit/credit FormGrid rows +
   StatTileGrid). HIGHEST financial hot-zone (GL, journal, VAT, trial-balance gate) — all
   mutations INTEG-gap. Bindings: CreateAccount/CreateJournalEntry/GetChartOfAccounts/
   GetJournalEntries/GenerateBalanceSheet/GenerateProfitAndLoss/GetCashflowEvidence*.
2. **CostingSheetScreen** (3026 — largest) — build/quote a costing sheet; header + per-line
   pricing math (FX/margin/freight/VAT) + revisions + export + save-as-offer. Needs a
   **LineItemsEditor** widget (DataTable-based repeating editable rows — also reused by
   Accounting's voucher entry and any invoice/PO create). Full-page workflow, not a modal.
3. **BankReconciliationScreen** (2140) — statement-line import + match against invoices/
   payments/expenses/payroll. Needs a new **AllocationMatchPanel** primitive (search
   candidates → multi-add to an allocation plan → running remaining-balance footer;
   reusable by AR/AP allocation). NOT the same as BookBankRecon (already built).
4. **PayrollScreen** (1167) — PII HOT-ZONE (salaries, bank details). **CONFIRM the
   `$lib/api/payroll` transport (Wails vs HTTP) before building.** Mode switcher + profile/
   period/run CRUD + approve→post→pay state machine. Keep synthetic; RBAC/field-masking.
5. **ButlerScreen** (2960) — AI assistant chat. Needs a new **chat-transcript primitive**
   (role bubbles, markdown+table render, arm→confirm action-chip states). Preserve the
   arm→confirm→timeout gating (consequential-action hot-zone). Rebuilding this RETIRES
   IntelligenceHub (route "Intelligence" straight here).

### K4 — deferred (need a shared **operational-hub tabbed-console primitive** — design once)
- **PeopleHub** (1879) — HR console. THE most sensitive screen (employee PII, gov-ID docs
  CPR/passport/visa/permit, payroll, archive/termination). Embeds Payroll.
- **WorkHub** (1445) — task/project kanban; embeds ApprovalsQueue (already built, has an
  `embedded` prop — preserve it).
- **DeploymentHub** (1093) — internal ops/pilot console (embeds a surveillance-adjacent
  activity monitor, double-gated).
- **OneDriveImportScreen** (1720) — 3-step wizard; needs the **Stepper primitive** (also
  SetupWizard). Currently DISABLED/unrouted in old App.svelte — low urgency.

### K4 RETIRED (owner-ratified 2026-07-14 — do NOT build): IntelligenceHub,
EntityDiscoveryScreen, ArchaeologistScreen, ArrivalCeremony, EcosystemDashboard,
CashPositionWidget (keep its 4 Go bindings for a future cash tile; drop the .svelte).

### K5 — App shell + INTEG completion + harness
- **Real app shell:** sidebar nav from the registry, routing, a `TabShell` primitive for
  the old tab-navigators (FinanceHub/SalesHub/CRMHub/OperationsHub — they were NOT
  dashboards, just tab shells hosting child screens). Auth flow: build the **auth chrome**
  here (Login/LicenseActivation/PendingApproval/SetupAdmin + SetupWizard on a new **Stepper
  primitive**), wired as a real flow (activation → device login → RBAC-gated nav). Extract
  a shared `PasswordField` control. i18n (`initI18n` pattern), divisions store fed by
  `GetDivisionRegistry`, `$wails` alias repointed to `frontend-lab/wailsjs`.
- **Close every INTEG gap** in the per-entity `bridge/<entity>.ts` real adapters (grep for
  `INTEG gap:` — every one names its binding). Requires the **quarantine env** (see
  KERNEL.md lesson 3 / original spec §1.3): `export PH_DB_PATH=<scratch>\integ.db` +
  `export APPDATA=<scratch>\appdata` (BASH syntax — the PowerShell tool runs bash;
  `$env:` no-ops). NEVER touch `%APPDATA%\Roaming\AsymmFlow` (real PH-adjacent DB with an
  ENABLED remote sync). Verify the scratch DB file appears before interacting.
  `cmd/devkey <db>` prints a seeded admin key; activate via `ActivateLicense(key)`.
- **Harness:** Playwright sweep running `layout-detector.mjs` on EVERY screen at 3 widths;
  L1 tripwire test (no raw layout CSS/hex/px in screens); L2 duplication tripwire.

### K6 — The flip
Per-screen parity sign-off table (old vs new, every screen, verdict + evidence). Owner
smoke checklist. Delete `frontend/` (one commit). `wails build` packaged smoke. Full repo
gates (go test ALONE, svelte-check, vite, all audits). Flip commit carries the parity
table. NO PUSH — owner decides when the branch graduates.

## 5. Orchestration playbook (what worked in Sprint 1)

- **Recon → decide engine features → build → gate → report → commit**, per wave. Dispatch
  read-only recon agents first (collision-free); they census old screens → real Go
  bindings → descriptor shapes + honest parity verdicts + visual-diversity flags.
- **Collision-free agent contract:** each build agent writes ONLY new files (its bridge +
  descriptor/screen + parity doc). Shared files (registry, engine, App, kernel) are YOURS.
- **Engine features only when ≥2 screens need them.** Build them yourself (tech-lead) with
  tests — they propagate to all screens. Agents fill descriptors/bespoke views.
- **Agents surface real gaps** (Sprint 1: row-aware forms, ProfileKpiSpec.tone, FX
  two-state, empty-widget hiding). When they STOP-and-report, build the fix mid-wave.
- **Batch** big waves (K1 ran 6 build agents in 2 batches). Gate each batch before the next.
- **Windows gotcha:** `entity.svelte.ts` VM collides case-insensitively with `Entity.svelte`
  → use a `-vm.svelte.ts` suffix or a distinct stem.
- **pnpm-workspace.yaml** gets auto-touched by tooling — `git checkout` it before commits.
- **Commit** in small coherent steps (local-only): engine spine, then per-batch, with the
  parity/gate results in the message. Sprint 1 = 20 commits, `b885736`→`f4cf526`.

## 6. Open owner questions (from Sprint 1, surface these at INTEG/K5)
1. Settings `GetSettings`/`UpdateSettings` real key schema (untyped `Record<string,any>` —
   Sprint 1 guessed snake_case, INTEG-gapped the write to avoid corrupting real settings).
2. Currency Rates model has no "pair" concept — currency-vs-BHD only (corrected).
3. Bank Account CRUD mutations INTEG-gapped (division-scoped + encrypted IBAN/SWIFT).
4. PayrollScreen `$lib/api/payroll` transport (Wails vs HTTP) — confirm before rebuild.
5. Minor mock polish: a BookBankRecon row shows status "Reconciled" with a non-zero
   variance (list/panel data pairing quirk — not a logic bug).

## 7. The prize (definition of done, unchanged)
All 60 old screens rebuilt-or-retired (retirements owner-signed in a wave report); K6 flip
executed; gates green; parity table complete; every deferred item named with its ledger
entry. One kernel, ~40 descriptors + a handful of bespoke screens, and "add a screen"
becomes a config change forever after.
