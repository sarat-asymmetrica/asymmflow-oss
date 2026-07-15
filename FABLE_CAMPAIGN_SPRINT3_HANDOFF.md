# Campaign — Frontend Kernel: Sprint 3 Handoff (for a fresh Opus 4.8 orchestrator)

**You are the incoming orchestrator + technical lead** for the final stretch of the
frontend-kernel full migration. Sprint 1 built the kernel + migrated 36 screens (K1–K4-partial).
Sprint 2 (this handoff's author) **completed K4** (all ~60 old screens rebuilt-or-retired) and built
**most of K5** (real app shell + auth gate + stores + the 4 tab-navigator hubs). Your job: finish
K5's tail (OneDriveImport, L1/L2 tripwire tests, one L1 cleanup, INTEG), then **K6 (the flip)**.

Same operating model: **Sonnet 5 agents code, you gate every wave and fix what they miss.** Enforce
the kernel laws; the architecture has held beautifully across ~50 screens — keep it that way.

**Read FIRST, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` → `frontend-lab/KERNEL.md` (5 pillars,
laws L1–L7) → `frontend-lab/PARITY_INVOICES.md` (parity method) → `FABLE_KERNEL_CAMPAIGN_LOG.md` (the
durable running log — EVERY ruling/gap/gotcha, Sprint 1 + Sprint 2, is there) → the Sprint 1/2 wave
reports → this file. Then skim the per-screen parity docs in `frontend-lab/src/screens/parity/`.

**Branch:** `exp/frontend-kernel` (worktree `asymmflow-lab`, LOCAL-ONLY — never pushed).
**HEAD at handoff: `d335716`.** `wails.json` points at `frontend-lab`. Old `frontend/` stays
read-only reference until the K6 flip.

---

## 1. State of the world (verified green at HEAD d335716)

- **K4 COMPLETE** — every one of the ~60 old screens is REBUILT (~50 on the kernel) or owner-RETIRED (7).
- **K5 ~70% done** — real app shell + auth + stores + 4 tab-navigator hubs all landed.
- Gates green: `npm run check` **0/0 (340 files)** · `npm run test` **80 passing** · `npm run build`
  clean · **full-app layout-detector 48/48 CLEAN @1440+420** (via `frontend-lab/tests/gate.mjs`).
- **48 product screens** in the registry, rendering from the kernel through the REAL app shell.

### Kernel inventory (all built + tested where they carry logic)
- **Archetypes** (`kernel/archetypes/`): DocumentLedger, EntityMaster, Hub, FormModal, ActionHost.
  DocumentLedger + Hub now accept `embedded` (drop header + own scroll — for hub-tab hosting).
- **Primitives** (`kernel/primitives/`): PageShell (+`embedded`), Stack, Row (+`shrink`), Grid, Card,
  Scroll, Toolbar, FormGrid, DataTable, Modal, LedgerSummary, BalanceComparisonPanel, **ViewSwitcher**,
  **AllocationMatchPanel**, **Stepper**, **ChatTranscript**, **TabShell**, **Wizard**.
- **Widgets** (`kernel/widgets/`): Distribution, RankedBarList, StatTileGrid, List, ActivityFeed,
  Callout, ComparisonBars, Donut, **LineItemsEditor**.
- **Controls** (`kernel/controls/`): Button (+`min-width:0`), SearchInput, FilterChips, Badge,
  EmptyState, ConfirmDialog (+`reasonLabel`/`requireReason`), RangeSlider.
- **Kernel modules**: descriptor.ts, form.ts, format.ts, tones.ts, content.ts, hub.ts, ledger-core.ts,
  line-items.ts, allocation.ts, **markdown.ts** (escape-first, XSS-safe). Tested: format, form-core,
  ledger-core, line-items, allocation, markdown (80 tests).
- **Kernel form controls** (`styles/kernel.css`, global, L2 single-source): `k-field`, `k-field-label`,
  `k-input`, `k-input-area`, `k-field-wide`, `k-field-row`, `k-grow`. Screens use these for native
  form controls — NEVER per-screen form CSS.
- **Stores** (`src/stores/`): `session.svelte.ts` (currentUser/permissions/`hasPermission`/`actingUserId`),
  `divisions.svelte.ts` (GetDivisionRegistry + synthetic fallback), `navigation.svelte.ts` (route +
  navigate + one-shot cross-screen handoffs).
- **App shell** (`src/App.svelte` + `src/app/LicenseActivation.svelte` + `src/bridge/auth.ts`): real
  license gate (mock boots straight in) + permission-filtered sidebar + navigation-store routing.
- `$wails` alias repointed to `./wailsjs` (frontend-lab's own generated tree — has every binding).

### The gate harness — USE IT (`frontend-lab/tests/gate.mjs`)
`node tests/gate.mjs` gates ALL product screens; `node tests/gate.mjs "Label1,Label2"` gates a subset.
It drives a running dev server (`npm run dev -- --port 5175 --strictPort`), clicks each screen at 1440
then 420, selects a row, and runs `detectLayoutViolations`. Playwright is resolved GLOBALLY (see the
`createRequire` line). **Gotcha:** build agents spawn their own dev servers + sometimes edit `registry.ts`
transiently — before gating, kill stray servers on 5175-5182 (`netstat -ano | grep :PORT | taskkill //F //PID`)
and start ONE clean, and re-verify `registry.ts` has no leftover TEMP lines.

---

## 2. Remaining K5 work (finish these, gate each, commit)

### 2a. L1/L2 tripwire tests (HIGH VALUE — do this first; it locks in the whole campaign)
The handoff's harness requirement. Build vitest tests that mechanically enforce the laws I hand-enforced:
- **L1 tripwire** (`tests/l1-no-layout-css.test.ts`): scan every `src/screens/*.svelte` `<style>` block
  and FAIL on raw layout CSS — `display:`, `margin`, `float:`, `flex-direction/justify-content/align-items`,
  `grid-template/grid-column` (except via the kernel `k-field-wide`), `gap:`, non-font `padding:`/`min-height`/
  `width`/`height`/`top`/`left` in raw `px`, and raw `#rrggbb` hex. ALLOW: font-*, color:var(), letter-spacing,
  text-*, overflow*, white-space, overflow-wrap, `calc(NNpx * var(--ui-font-scale))` for font-size, min-width:0.
  **EXCLUDE**: `Showcase.svelte` (intentional dev kitchen-sink w/ a 3000px overflow demo — detector-exempt),
  and `App.svelte`/`app/*` (app CHROME, exempt like the old lab-shell).
- **L2 tripwire** (`tests/l2-no-duplication.test.ts`): FAIL if any screen redefines a kernel utility —
  a local `.k-field`/`.k-input`/`.k-field-label` style def, or a re-implemented `formatDate`/`formatMoney`/
  `formatNumber` (screens must import from `$kernel/format`). Grep-style assertions over the screen files.
- **Known violations to FIX before the tripwire lands green** (audited at handoff):
  - **`BusinessSettings.svelte`** (Sprint-1 Settings screen) still hand-rolls `.bs-field/.bs-label/.bs-input`
    form CSS + a raw hex `#b3261e` (line ~122). MIGRATE it to the kernel `k-field`/`k-field-label`/`k-input`
    classes (same mechanical swap I did for Payroll/Accounting/BankRecon/CostingSheet: `sed` the class names,
    delete the local style defs, hex → `var(--k-tone-danger-fg)`). Parity docs already flagged this migration.
  - **`CostingSheet.svelte`** `.cs-textarea { min-height: 160px }` (line ~529): swap the T&C textarea to
    `class="k-input k-input-area"` and delete `.cs-textarea` (accept the 72px min; it's resizable).

### 2b. OneDriveImportScreen (the last screen — on the `Wizard` primitive)
Deferred from K4 (disabled/unrouted originally). The `Wizard` primitive is built (`kernel/primitives/Wizard.svelte`:
`steps`, `currentIndex`, `content` snippet, `onBack`/`onNext`, `canAdvance`, `nextLabel`, `busy`). Recon is in
`FABLE_KERNEL_CAMPAIGN_LOG.md` + the K4-deferred recon: 3 steps (configure paths → review scanned deals → run
import). Bindings: `ValidateOneDrivePath` (FETCH-ish), `ScanOneDrivePaths`/`ConfirmOneDriveDeal`(dead)/
`ImportOneDriveDeals` (INTEG-gap, filesystem + real offer creation). Step 2's deals table wants an interactive
checkbox + customer-select per row — check whether to add a `ColumnSpec.rowAction`/interactive-cell to DataTable
(see kernel gaps) or eject to a `cell` component. Dispatch ONE agent (avoid concurrency thrash for a single screen).

### 2c. i18n shell chrome (LOW priority — optional)
Port `frontend/src/lib/i18n/index.svelte.ts` (thin runes store + `GetTranslations` binding) for the SHELL
CHROME ONLY (nav labels, header). DEFER pervasive screen-level i18n (retrofitting 50 screens' English literals
is a separate wave — do NOT block K5 on it). Judgment call; skippable if time-pressed.

### 2d. INTEG completion — ⚠️ OWNER-GATED, sovereign-mesh direction
**Owner ruling (do NOT deviate):** close the bridge `INTEG gap:` throws toward the SOVEREIGN MESH, NOT the old
remote-Postgres sync. (1) Do NOT wire/enable the legacy DuckDNS-exposed-Postgres remote sync (Era-1, retired).
(2) Wire the frontend `real*` adapters to the Wails bindings and VALIDATE against the OWNER'S LOCAL PostgreSQL
(owner has PG tooling) — never the real PH SQLite / `%APPDATA%\Roaming\AsymmFlow` (live DB, remote sync ENABLED).
(3) The sync/replication layer = a Holesail (holesail.io) P2P sidecar — a SEPARATE future build, out of scope here.
**Before touching the real-binding/DB layer, PAUSE and confirm the Postgres/runtime env with the owner** (the
owner asked to be looped in on the environment setup). `grep -rn "INTEG gap:" src/bridge/` lists every gap; each
names its exact binding. This is the ONE genuinely risky step — do the mock-safe K5 tail first.

### 2e. Deferred K5 polish (nice-to-have, not blocking)
- FinanceHub division selector; OperationsHub per-tab badge counts; SalesHub conditional SalesAdminTools tab
  (`CanResolveOpportunityConflicts` gate). All flagged in the hub files/parity docs.
- Nav curation: the sidebar currently shows BOTH the hubs AND the individual screens (over-complete). K6/polish
  should decide the canonical top-level nav (hub-level vs flat) — the registry keeps all screens as routable
  targets regardless.
- Cross-screen deep-links (Butler navigate actions, WorkHub `openTask`, PeopleHub "set up payroll") — the
  `navigation.svelte.ts` store now provides `navigate()` + `setHandoff`/`consumeHandoff`; wire the deferred hooks.

---

## 3. K6 — The flip
Per-screen parity sign-off table (old vs new, every screen, verdict + evidence — the parity docs are your
source). Owner smoke checklist. Delete `frontend/` (one commit). `wails build` packaged smoke. Full repo gates
(go test ALONE, svelte-check, vite, all audits). Flip commit carries the parity table. **NO PUSH** — owner
decides when the branch graduates. NOTE: repointing `$wails` at K5 means the old `frontend/wailsjs` is no longer
consumed by the lab; the flip deletes the whole old tree.

---

## 4. Parked owner rulings/questions (fold in at INTEG/K6; all non-blocking — mutations are INTEG-gapped)
- **Payroll:** field-masking is net-new (`canViewUnmasked` defaults true) — confirm the flag shape before a real
  permission wires in · Mark-Paid enabled from approved OR posted (preserved) — tighten to posted-only? ·
  Approve-reason optional — make mandatory?
- **CostingSheet:** freight/margin calc-time fallback=0 (vs the 9/20 seed) & the profit/cost asymmetry
  (hiddenCharges cost-only; discount/VAT revenue-only) — both preserved verbatim + unit-tested; correct them? ·
  Save-as-Offer overwrite-guard rebased on `linkedOfferNumber`.
- **Accounting:** the fragile VAT string-match heuristic was DROPPED (owner-ratified) — confirm.
- **BankRecon:** book-vs-bank cross-nav not ported (restore?); audit-trail drawer added (keep?).
- **PeopleHub:** doc-number auto-unmask on Edit (require a fresh unmask action?).
- **WorkHub:** default tab fixed to my_work; assignee lists simplified to full roster (restore project-scoping?);
  blocked-reason inline-refusal vs requireReason dialog.

## 5. Kernel gaps noted (K5/polish candidates — none blocking)
- **`DataTable`/`ColumnSpec` has no declarative lightweight per-row action** (needs a `cell` override today).
  Recurs (DeploymentHub queue retry, OneDriveImport per-deal). Candidate: `ColumnSpec.rowAction` (label +
  visibility predicate + onClick) rendering inline. Worth building for 2b.
- **No "fill remaining page height" chain** (Butler chat scrolls the page vs an internal feed) — a viewport-height
  chain the real shell could thread (add a `fill` mode to Card/Grid + PageShell content). Butler is functional meanwhile.
- **No multi-select control** (WorkHub's add-members checklist is hand-rolled).
- **ChatTranscript type exports** live in the instance `<script>`, not `<script module>` — a `kernel/chat.ts`
  types extraction (like line-items.ts/allocation.ts) would let consumers import cleanly.

## 6. Operating playbook (what worked in Sprint 2)
- **Recon → build shared primitives yourself (tech-lead, with tests) → dispatch Sonnet agents per screen →
  gate → fix → wire registry → commit.** Build a primitive BEFORE its consumer agent dispatches.
- **Collision-free agent contract:** agents write ONLY their new files (bridge/vm/screen + return the parity
  doc content in their final message — YOU write `parity/*.md`, blocked for subagents). Registry is
  ORCHESTRATOR-OWNED — collect their registry lines, wire them yourself. **Tighten the contract: agents must
  NOT touch registry.ts EVER, not even transiently to run their own gate** (Sprint 2 had repeated transient
  edits + stray dev servers; have them gate via a throwaway or ask you to wire, then you gate).
- **Fix drift at the KERNEL, not per-screen.** When agents hand-rolled form CSS / hit a flex-overflow, the fix
  was a kernel-level addition (form-control classes, `Row shrink`, `Button min-width:0`, `.k-grow`) that paid
  out across all screens. This is the campaign's whole thesis — honor it.
- **Adversarial mock is doctrine** (seeded LCG: 200-char/RTL/empty names, huge/negative/tiny amounts, UNKNOWN
  statuses, empties, dangling FKs). Synthetic identity ONLY (SYNTHETIC_IDENTITY.md) — never real names/PII.
- **Windows gotcha:** `Entity.svelte.ts` collides case-insensitively with `Entity.svelte` — use a `-vm.svelte.ts`
  suffix. `pnpm-workspace.yaml` auto-touches — `git checkout` it before commits.
- **Commit in small coherent steps** with gate results in the message. Sprint 2 = 13 commits, `8363172`→`d335716`.

## 7. The prize (unchanged)
All 60 old screens rebuilt-or-retired ✅ (done). K5 tail + K6 flip remain. One kernel, ~10 primitives + ~40
descriptors + a handful of bespoke screens, and "add a screen" is a config change forever after.
