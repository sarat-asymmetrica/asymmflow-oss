# Frontend Kernel Campaign — Orchestrator Log

> **SPRINT 1 CLOSED at commit f4cf526 (2026-07-14).** 36 product screens migrated across
> K1–K4-partial; 4 archetypes + full primitive/widget library; every wave green +
> detector-clean + parity-ledgered. Continuation brief for a fresh Opus 4.8:
> **`FABLE_CAMPAIGN_SPRINT2_HANDOFF.md`** (remaining K4 L-monsters + deferred, then K5
> app-shell/INTEG, then K6 flip). This log is the durable running record; the handoff is
> the entry point for Sprint 2.

Durable progress tracker for the K1–K6 full-migration campaign
(`FABLE_CAMPAIGN_FRONTEND_KERNEL.md`). Orchestrator = Opus 4.8; coders = Sonnet 5.
Branch `exp/frontend-kernel` (LOCAL-ONLY). Updated as waves land.

## INTEG EXECUTION (fresh Opus 4.8 orchestrator, from c29e17a-minus-repoint) — `FABLE_CAMPAIGN_INTEG.md`

- **Tooling:** installed the SQLite CLI (winget `SQLite.SQLite` 3.53.3) for scratch-DB
  verification per §3/§4 (Go query snippets remain the primary check).
- **★ Wave I1 — cross-cutting prerequisites DONE (green: check 0/0 348, test 148, build clean, go build ./... clean).**
  - **I1.1 session actor:** the last `actor='lab-user'` placeholder (bank-reconciliation-vm.svelte.ts)
    now reads `actingUserId()` from the session store (a getter, so it reflects the live license
    identity at mutation time). Grep for `lab-user` in `src/` = zero screen hits. Session is already
    populated for real by the shell (App.svelte `setSession` from the license-activation result).
  - **I1.2 divisions registry as the ONE division-vocabulary source:** added `getDivisionOptions()` to
    `stores/divisions.svelte.ts`; routed `bridge/index.ts divisionOptions` (invoice/payment forms) and
    `bridge/costing-sheet.ts costingDivisionOptions`/`defaultCostingDivision` through it; deleted the
    dead static-mock `mock.divisionOptions`. Under real Wails these now reflect `GetDivisionRegistry`
    (loaded during boot, `await initDivisions()` before render); under mock they keep the BUILTIN
    synthetic fallback. Invoice division options made LAZY (`options: async () => divisionOptions()`)
    so they read the post-boot registry rather than the module-eval fallback. Mock DATA seeding keeps
    private synthetic literals (L7 audit-exempt, like every generator).
  - **I1.3 date→time.Time form bridge:** built ONE kernel-level helper `map.goTime(dateStr)` — emits the
    UTC-midnight RFC3339 string Wails marshals into a Go `time.Time` (the generated `time.Time` TS class
    is an empty codegen stub, so we pass the wire string + cast). Wired `SetExchangeRate` as the proof
    consumer (currency-rates mutation row flipped mock→**wired**). VALIDATED end-to-end by
    `integ_date_bridge_test.go` against a scratch SQLite: the exact wire string round-trips into the
    correct `time.Time`, the rate persists, a re-set closes the prior active rate (effective_to), and the
    empty-date guard maps to Go zero time (refused at the seam, never a silent "today").
  - **I1.4 AI-provider-key secrets storage:** parked owner decision — surfaced, NOT improvised (see
    INTEG checkpoint). Only affects the Settings/Butler AI-key path (a DEFER); does not block I2/I3.
- **★ Wave I2 — read swaps (mock→real) DONE (green: check 0/0 348, test 148, build clean, FULL layout sweep 49/49).**
  Committed in 3 batches (`9dd1660` dashboards, `fc970f1` sales+system reads, `7097633` profile-enrich engine).
  - **Batch A — 5 dashboards:** main (3-binding composition GetDashboardStats+pipeline+AR-aging via
    Promise.all; focus/alerts/tasks honest-blank — no roster binding), finance-overview
    (GetFinancialDashboardForYear, ~35 fields 1:1), ahs-finance (GetFinancialDashboardByDivision with
    division resolved from the registry `dashboardVariant==='ahs'` — consumes the I1.2 store, no literal),
    crm-customer/crm-supplier (GetCRM*Dashboard[ByYear]; metric-card share pct DERIVED since the cards
    carry none).
  - **Batch B — sales+system reads:** serial-trace (SearchSerials/GetRecentlyDeliveredSerials);
    opportunities READ (RFQ+pipeline merge mirroring costing-sheet's proven mapping + folder dedup, +
    ListCustomers options — mutations stay I3); audit-trail 3-level chain
    (accounts→statements→GetAuditTrail flattened, amount honest-blank — the log has none); approvals
    fetch (ListDeleteApprovalRequests+ListEmployeeArchiveRequests, status=''=all — reviews stay I3);
    notifications (ListNotificationFeed+MarkNotificationAsRead — the recon's transport uncertainty
    RESOLVED, direct bindings exist; reviewStatus/requester/reason honest-blank as they live on the
    request, not the notification; approve/reject reviews stay I3; live-push DEFER).
  - **Batch C — EntityMaster `profile.enrich` ENGINE (fix-at-the-kernel):** new
    `EntityDescriptor.profile.enrich?(row)` + `LedgerViewModel.enrichSelected()` (idempotent per id,
    reset on reload, non-fatal) + an EntityMaster `$effect` on selection. Wires the secondary-fetch
    depth: GetCustomerFullProfile (customers) + GetSupplierFullProfile (suppliers). GetCashPosition was
    already wired (bank-recon); finance-overview has no separate overlay consumer.
  - **⚠️ customer-360 STOP-AND-SURFACED (not wired):** real `Customer360Data` is NARROWER than the
    view (no contact/TRN/credit-limit/regime; adds aging/history/orders) and `Customer360Graph` is a
    node/edge graph, not the flat connections summary — a genuine SHAPE-DIVERGENCE, not a swap. Kept
    honestly gapped with precise notes; needs an OWNER shape decision (reshape the view to the backend,
    or compose a supplementary customer-detail fetch). Read-only, no persistence risk.
  - **INTEG discipline held:** every mutation on these screens still throws its honest `INTEG gap:` —
    only reads (+ the benign MarkNotificationAsRead) flipped. No silent mock persistence anywhere.
- **★ OWNER RULINGS at the I1/I2→I3 checkpoint (2026-07-15):**
  1. **I3 validation = Go-test doctrine (ratified).** I cannot headlessly drive the WebView2 GUI
     (Playwright hits the vite dev server = mock mode, no `window.go`). So each I3 hot-zone is validated
     by: (a) `npm run check` proving the adapter↔binding contract via the generated d.ts, (b) a Go test
     driving the actual App binding against a scratch SQLite asserting persisted state + audit trail +
     the reversal path where one exists (the spec's "Go query snippet against the scratch DB"). The
     owner's smoke checklist remains the human GUI pass. time.Time marshalling already proven (I1.3).
  2. **Customer-360 = reshape the view to the backend.** Drop the mock-invented contact/TRN/credit/regime;
     surface what `Customer360Data` provides (receivables aging, payment history, recent orders/RFQs);
     derive connections from the `Customer360Graph` node/edge data. (Bespoke-screen rework.)
  3. **AI-provider keys = encrypted in-app settings** via the existing FieldCrypto/DPAPI keystore (matches
     the no-secrets-in-source posture); a Settings key field. Resolves the I1.4 parked decision.

## INTEG campaign staged (2026-07-15, post-Sprint-3; Fable + owner)

- **Merged to main `c29e17a`** (pushed) — K1–K6 flip-prep + mesh Wave 0, **minus the
  `wails.json` repoint** (held back as flip step 2; the repoint lives only on this branch —
  do NOT naively `git merge main` here or it reverts). Gates at merge: check 0/0 (348),
  vitest 148/148, go build+test clean, mesh smoke green.
- **★ OWNER RULING (2026-07-15, supersedes the runtime clause of `1779b3c`): SQLite-primary
  is PERMANENT; Postgres RETIRED from the target architecture.** Rationale ratified after
  architectural review: the boot path is deeply SQLite-shaped (PRAGMA/writable-schema
  CHECK-constraint surgery, app.go:1984–2046), mesh peers run SQLite, and DB-row-level sync
  can't express business-invariant conflict semantics — the mesh reducer can. The always-on
  office machine's job changes: **always-on mesh peer** (durability anchor + backup
  custodian), not a database server. Owner also validated the Holepunch stack first-hand
  (keet.io). Consequence: INTEG's Wave I0 (PG-runtime spike) was cut from the spec before
  launch; validation runtime = quarantined scratch SQLite. Verified-then-retired PG artifacts
  (throwaway `asymmflow_integ` DB, `.env.integ.local`) were dropped/deleted.
- **`FABLE_CAMPAIGN_INTEG.md`** = the Task #4 execution spec (I1 prereqs → I2 read swaps →
  I3 financial hot-zones). Runs parallel to sovereign-mesh Wave 1 (`asymmflow-mesh`
  worktree, Fable-driven) — disjoint surfaces by design.

## SPRINT 3 (fresh Opus 4.8 orchestrator, from 5fe30bc)

Continuation of `FABLE_CAMPAIGN_SPRINT3_HANDOFF.md`: finish the K5 tail (tripwires,
known-violation fixes, OneDriveImport), then INTEG (owner-gated) + K6 flip.

- **L1/L2 law tripwires + known-violation fixes (commit 0f94fbf):** the campaign's
  hand-enforced laws are now MECHANICAL. `tests/l1-no-layout-css.test.ts` (scans every
  `src/screens/*.svelte` `<style>`, fails on structural layout props / raw-px sizing /
  min-width≠0 / raw hex; Showcase excluded) + `tests/l2-no-duplication.test.ts` (fails
  on a screen redefining a `.k-*` class or re-implementing formatDate/Money/Number).
  Proven to BITE (injected display/margin/hex/.k-field → both fail) then revert clean.
  Fixed the two audited violations: BusinessSettings `.bs-*` form CSS → kernel
  `k-field/k-field-label/k-input` + `#b3261e` → `var(--k-tone-danger-fg)` + margins →
  Stack; CostingSheet `.cs-textarea{min-height:160px}` → `k-input k-input-area`.
  `tests/node-builtins.d.ts` — tiny ambient shim for the node builtins the harness reads
  with, so `npm run check` stays green WITHOUT adding `@types/node` (which would pollute
  ambient globals across all ~50 screens). check 0/0 (343), test 80→139.
- **OneDriveImport — THE LAST SCREEN (commit c0dc3b7):** 3-step Wizard (configure paths →
  review deals → run import) on the `Wizard` primitive; closes the K4-deferred screen.
  ONE Sonnet agent, orchestrator-gated+fixed. **DataTable interactive-cell RULING:** the
  per-row include-checkbox + customer-select use the EXISTING `ColumnSpec.cell` L4 ejection
  (two `Component<{row}>` cells mutating `$state`-backed rows directly; VM's derived
  selection recomputes live) — NO new kernel API. The handoff's `ColumnSpec.rowAction`
  candidate stays OPEN (poor fit for stateful checkbox+select; a button-style consumer like
  DeploymentHub-retry may justify it later). All four bindings INTEG-gapped (screen runs on
  an adversarial mock: 200-char/empty/RTL folder names, 0/1/2-3 matches, huge/zero files).
  Fixed in review: scan mock regenerates fresh (was caching → Start-Over showed stale
  selections). **Gate coverage gotcha:** `tests/gate.mjs` only reaches a Wizard's step 0
  (it doesn't click Next) — steps 1/2 layout-verified with a throwaway driver (clean
  @1440+420 incl. the 200-char + RTL rows). check 0/0 (348), test 148.
- **★ K5 MOCK-SAFE TAIL COMPLETE.** Remaining: 2c i18n shell chrome (LOW/optional), 2e
  deferred hub polish (nice-to-have), **2d INTEG (OWNER-GATED — pause for PG env)**, then K6.
- **K5 polish — owner chose "polish first" (commit 59ae4ef):** wired **hub tab deep-linking**,
  finishing the nav store's `Route.tab` contract (defined in Sprint 2, previously unwired).
  New `routeTabOr(validKeys, fallback)` store helper + all 4 tab-navigator hubs init their
  active tab from `currentRoute().tab` and switch via `$effect` on in-place navigate. So
  `navigate('finance-hub', { tab: 'payroll' })` deep-links to a hub tab (fresh-mount + in-place,
  both PROVEN with a throwaway Playwright probe driving the app's own nav singleton). Drill-downs
  deliberately UNCHANGED (they route to standalone filtered screens — the hub-embedded tabs don't
  thread `initialQuery`, so staying in-hub would drop the drill filter → P2 "in-hub drill" was
  analyzed and REJECTED as counterproductive).
- **K5 polish DEFERRED (honest rationale — none are safe mock-only "polish"):**
  - *FinanceHub division selector* — needs a division-filter prop contract the embedded child
    screens don't expose; threading it touches many screens → INTEG-adjacent, not polish.
  - *Hub per-tab badge counts* (Operations/Finance) — TabShell's `badge` prop is ready, but real
    counts want "open PO / pending fulfillment" FILTERED semantics, not raw row counts → INTEG.
  - *SalesHub conditional admin tab* — `SalesAdminTools` has NO kernel equivalent (net-new screen)
    and `CanResolveOpportunityConflicts` is unwired anywhere in the lab → net-new build, not polish.
  - *Nav curation* (sidebar shows hubs AND their child screens) — a genuine UX/design decision the
    handoff reserves for "K6/polish"; owner-reserved, better decided holistically at the flip.
  - *Butler fill-page-height chain* — a real visible defect but a multi-primitive kernel change
    (PageShell scroll region → Grid rows → Card heights) with regression risk across 48 screens
    right before the flip; catalogued as a KERNEL GAP (§5), not a 2e polish item → deferred.
  - *i18n shell chrome* — LOW value for an English pilot; screen-level i18n is a separate wave. Skipped.
- **K6 flip-prep — owner chose "parity table" (commit 6240340):** consolidated the ~42
  per-screen parity ledgers (`src/screens/parity/*.md` + `PARITY_INVOICES.md`) + K3/K5
  composition notes into ONE sign-off doc **`FABLE_WAVE_K6_PARITY.md`** (repo root). Per-screen
  table for all 49 product screens (grouped by nav group): archetype, old→new, read-data status
  (`real`/`real*`/`mock-INTEG`), mutation status, INTEG-pending real bindings, owner ☐. Plus:
  deliberately-retired ledger (IntelligenceHub, Settings Deployment tab, activity monitor, VAT
  card, phantom Pending, EcosystemDashboard); a **consolidated INTEG roster grouped by risk**
  (feeds owner-gated Task #4, incl. cross-cutting prereqs = session store / divisions registry /
  date→time.Time bridge); open kernel gaps; an **owner smoke checklist** for a real Wails build;
  and the flip procedure DOCUMENTED-NOT-RUN (no `frontend/` delete, no `wails build`, no push —
  owner go required). Two roll-up corrections caught vs the mocks during authoring: main-dashboard
  fetch = `GetDashboardStats`+pipeline+AR-aging (not a single binding), and finance-overview read
  is `mock (INTEG)` (`GetFinancialDashboardForYear`), not real. Docs only, reversible.
- **★ MOCK-SAFE CAMPAIGN COMPLETE.** All that remains is owner-gated: **2d INTEG** (needs PG env
  confirmation) and **K6 flip execution** (repoint build → wails smoke → delete `frontend/` →
  full gates → owner graduation). Both PAUSE for the owner.

## SPRINT 2 (fresh Opus 4.8 orchestrator, from 9011bdd)

> **SPRINT 2 CLOSED at commit d335716 (2026-07-15).** K4 COMPLETE (all ~60 screens rebuilt-or-retired);
> K5 ~70% (real app shell + auth gate + session/divisions/navigation stores + 4 tab-navigator hubs). 48
> product screens on the kernel through the REAL shell; full-app layout-detector 48/48 clean; check 0/0
> (340 files), test 80, build clean. 13 commits (8363172→d335716). **Continuation brief for a fresh Opus
> 4.8: `FABLE_CAMPAIGN_SPRINT3_HANDOFF.md`** — remaining = K5 tail (L1/L2 tripwire tests + BusinessSettings
> L1 migration + OneDriveImport on Wizard + INTEG toward sovereign-mesh/local-Postgres, owner-gated), then K6 flip.

- **K4-L engine spine (commit 8363172):** 5 tech-lead primitives, all tested (55 tests):
  LineItemsEditor(+line-items.ts) · ViewSwitcher · AllocationMatchPanel(+allocation.ts) ·
  Stepper · ChatTranscript(+markdown.ts, escape-first XSS-safe). ViewSwitcher was a real
  kernel gap (no left-nav/tab primitive existed → screens couldn't switch views L1-clean).
- **K4 L-monsters DONE (commit 86b86d2):** Accounting + CostingSheet + BankRecon + Payroll
  rebuilt (4 parallel Sonnet agents, orchestrator-gated+fixed). Gate green: check 0/0 (314),
  test 80/80 (+25 costing sacred-math), build clean, layout-detector 0 @1440+420 (row-sel).
  - Accounting: ViewSwitcher console; VAT string-match heuristic DROPPED (owner-ratified);
    LineItemsEditor voucher (balanced badge display-only); 10 mutations INTEG-gapped.
  - CostingSheet: 25-col LineItemsEditor waterfall; sacred math VERBATIM + 25 unit tests
    (Math.ceil; freight/margin fallback=0; profit/cost asymmetry); 2 live bugs caught.
  - BankRecon: old allocation UI → AllocationMatchPanel with ZERO mods (headline win);
    edit-clears-match preserved; audit-trail ActivityFeed added; 13 mutations gapped.
  - Payroll: Owner Q#4 RESOLVED (Wails IPC only). ViewSwitcher+Stepper; FIXED approve/post
    confirm+operator-reason; field-mask net-new (canViewUnmasked default true); 6 gapped.
  - Kernel: single-source form controls (k-field/k-field-label/k-input/k-input-area/
    k-field-wide) in styles/kernel.css — killed .bs-/.pr-/.acc-/.br-/.cs- duplication (L1/L2).
  - Harness: tests/gate.mjs (reusable Playwright layout-detector, each screen @1440/420).
- **ORCHESTRATION LESSON:** a build agent edited registry.ts (shared merge point) to run its
  own Playwright gate, then reverted — collision-free contract needs "do not touch registry
  EVER, not even transiently; gate via a throwaway entry file or ask the orchestrator to wire."
- **Butler DONE (commit 721d0ec):** AI-chat console on ChatTranscript; arm/confirm/6s hot-zone
  preserved+verified-live; AI-authority boundary structurally enforced (23 bindings → 1 INTEG seam);
  refuse-over-guess guards preserved; 3-tier fallback collapsed; insights feed dropped; RETIRES
  IntelligenceHub. Kernel refinements from its gaps (all 41 screens re-gated clean): .k-grow utility;
  Button min-width:0; Row shrink={false} prop. Deferred to K5: fill-page-height chain + bespoke navigate hook.
- **ALL 5 K4 L-MONSTERS COMPLETE** (Accounting/CostingSheet/BankRecon/Payroll/Butler). Full-app gate 41/41
  clean @1440+420.
- **K4-deferred spine (commit 449006f):** TabShell primitive (lazy-mount keep-mounted, permission-gated,
  header slot; also serves K5 tab-navigators) + embedding convention (PageShell/DocumentLedger/Payroll
  `embedded`; Payroll `presetEmployeeID`) + ConfirmDialog reason variant (reasonLabel/requireReason).
- **K4-deferred hubs DONE (commit 12961c0):** PeopleHub + WorkHub + DeploymentHub on TabShell (3 parallel
  agents). Full-app gate 44/44 clean. PeopleHub (gov-ID masking, archive confirm+reason, manager-cycle guard,
  embeds real Payroll), WorkHub (allocation precheck preserved, task-delete unified, project delete/archive/
  shelve with reason, embeds real Approvals), DeploymentHub (**Activity/surveillance monitor RETIRED per owner**;
  bulk-retry confirm added). OneDriveImport + the Wizard primitive it needs → DEFERRED to K5.
- **★ K4 COMPLETE — every one of the ~60 old screens is now REBUILT or owner-RETIRED.** ~46 screens on the
  kernel + 7 retired. Next: K5 (app shell + auth chrome on a new Wizard primitive + OneDriveImport + close all
  INTEG gaps + L1/L2 tripwire harness), then K6 (the flip).
- **★ K5 AUTH — OWNER RULING (2026-07-15): LICENSE-ONLY, NO CEREMONY.** Build only the live license gate
  (`checking → license_needed → LicenseActivation(PH-XXX-YYYYYY) → approved`). SKIP ArrivalCeremony. PARK
  (documented backlog, not built) the orphaned device-registration flow: Login / PendingApproval / SetupAdmin /
  SetupWizard. Consequence: no PasswordField control needed for K5 (license key input, not a password field).
  SetupWizard's Wizard-primitive consumer is parked too → the `Wizard` primitive's live K5 consumer is
  OneDriveImportScreen only.
- **K5 app-shell recon done:** old App.svelte = currentScreen string + screenLoaders lazy map + persistentScreenIDs
  (keep-mounted) + hash deep-link + permission gate; nav = EnterpriseSidebar (NAV_ITEMS filtered by hasPermission).
  Tab-navigator hubs (FinanceHub 13 tabs / SalesHub / CRMHub drill-in / OperationsHub badge-counts) = thin TabShell
  wrappers over ALREADY-BUILT kernel screens. Infra to port: session.svelte.ts (currentUser/permissions/hasPermission
  — NEW, unblocks BankRecon/Deployment/People acting-user), divisions.svelte.ts (near-zero-risk straight port,
  GetDivisionRegistry), i18n (shell chrome only, defer screen-level), navigate(intent)+pending-handoff stores,
  $wails repoint → ./wailsjs (frontend-lab/wailsjs ALREADY EXISTS with all bindings). Net-new screens: SupplierDetailView,
  Expenses-approvals-mode tab. SalesHub opportunities-tab collision → orchestrator resolves (maps to registry `opportunities`).
- **★ K5 INTEG DIRECTION — OWNER RULING (Sprint 2, supersedes the handoff's quarantine-SQLite plan):** close the
  INTEG gaps toward the **SOVEREIGN MESH** vision, NOT the old remote-Postgres sync. Concretely: (1) do NOT wire
  or enable the legacy DuckDNS-exposed-Postgres remote sync (Era-1, retired); (2) wire the frontend `real*`
  adapters to the Wails bindings and VALIDATE the INTEG surface against the OWNER'S LOCAL PostgreSQL server
  (owner has PG tooling installed) — not the real PH SQLite/`%APPDATA%\Roaming\AsymmFlow`; (3) the sync/replication
  layer becomes a **Holesail (holesail.io) P2P sidecar** — a SEPARATE future build, out of scope for closing these
  gaps. This is the Era-3 architecture (DuckDNS→Holesail P2P mesh) applied to the kernel INTEG layer. NEVER touch
  the live PH-adjacent DB with enabled remote sync.
- **Kernel gap for K5 (non-blocking):** DataTable has no declarative lightweight per-row action (needs a `cell`
  override today) — candidate `ColumnSpec.rowAction` (label + predicate + onClick). Recurs (DeploymentHub queue,
  OneDriveImport per-deal). Also: no session/currentUser store yet (BankRecon/DeploymentHub); no fill-page-height
  chain (Butler chat); no multi-select control (WorkHub add-members).
- **Owner questions parked for review** (all non-blocking — mutations INTEG-gapped): payroll
  field-mask policy / post-before-pay / approve-reason-required; costing freight-margin +
  profit/cost asymmetries + save-as-offer overwrite-guard; bankrecon K5 session primitive +
  audit-drawer + book-vs-bank cross-nav; accounting VAT-heuristic-dropped confirmation.

## Architecture decisions (orchestrator, binding for all waves)

- **Per-entity bridge modules** (`bridge/<entity>.ts`): each ledger/entity owns a
  self-contained module (types + mock + real + bridge-switched public fns) using the
  shared `bridge/runtime.ts` (`usingWails`/`pick`). Kills the god-mock and lets N
  agents build in parallel with ZERO shared-file collisions. The two pilots
  (invoices/customers) keep the legacy central `bridge/index.ts`; index.ts re-exports
  runtime + barrels the new modules.
- **Screen registry** (`screens/registry.ts`): one typed list mapping key → {label,
  group, archetype, descriptor|component}. App.svelte renders from it. This is the
  K5 nav backbone, grown wave by wave. Registry edits are orchestrator-owned (merge
  point) so parallel agents never touch it.
- **Agent output contract per screen** (collision-free — agents write only NEW files):
  1. `bridge/<entity>.ts`
  2. `screens/<entity>.descriptor.ts`
  3. `screens/parity/<Entity>.parity.md` (the PARITY_INVOICES.md method)
  Orchestrator wires the registry entry + gates + fixes.
- **Visual-diversity mandate** (owner): don't inherit the card-heavy jank. Ledger
  engine gains a declarative **summary strip** (count + money totals + status
  distribution mini-bar) when ≥2 screens want it — consistent data-viz, cheap.

## Gate bar (every wave end)
`npm run check` 0/0 · `npm run test` all green · `npm run build` clean ·
layout-detector zero-violation at 1440/900/420 · per-screen parity docs honest.

## Wave status

| Wave | Scope | Status |
|---|---|---|
| K0 | Baseline verified + scaffolding + engine spine | ✅ done |
| K1 | Ledger blitz — 12 ledgers built, gated, detector-clean; report written | ✅ done (awaiting review) |
| K2 | Entity blitz — Suppliers/Users/widen-Customers/Inventory DONE; Pricing+Cust360→K4 | ✅ done (awaiting review) |
| K3 | Hub archetype + 4 dashboards DONE (donut/comparison/ranked/stat-grid/etc) | ✅ done (awaiting review) |
| K4 | Bespoke — 15 built (t1:6 + t2:4 + t3:5) + 6 retired; L-monsters + deferred remain | 🔨 ~65% |
| K3 | Hub archetype + dashboards | ⏳ |
| K4 | Bespoke screens on primitives | ⏳ |
| K5 | App shell + INTEG completion + harness | ⏳ |
| K6 | The flip | ⏳ |

## K1 screen → binding map (to be confirmed by recon)
| Screen | Likely fetch binding | Service |
|---|---|---|
| Orders | ListOrders(limit,offset) / FilterOrders | CRM |
| PurchaseOrders | ListPurchaseOrdersPaginated(page,size,status) | Infra |
| Quotations | (recon) | ? |
| RFQs | (recon) | CRM |
| Offers | GetAllOffers / ListOffersPaginated | CRM/Infra |
| DeliveryNotes | (recon — ListShipments?) | CRM |
| GRNs | ListGRNs(limit,offset,qcStatus) | CRM |
| ChequeRegister | (recon) | Finance |
| Expenses | ListExpenseEntries(status,includePaid) | Finance |
| SupplierInvoices | (recon) | Finance |
| SupplierPayments | GetAllSupplierPayments() | Finance |
| Payments | GetAllPayments(limit,offset) | Finance |
| CreditNotes | ListCreditNotes(limit,offset) | Finance |

## Log
- **K0 (2026-07-14):** Read authority chain (KERNEL/PARITY_INVOICES/spec). Verified
  baseline green. Census: 62 old screen files (~65k LOC). Established per-entity
  bridge + registry architecture. Scaffolding green.
- **K1 recon (2026-07-14):** Two Sonnet agents censused all 13 ledgers → recon-K1-A.md
  (sales) + recon-K1-B.md (finance) in scratchpad. Rulings: QuotationScreen is NOT a
  ledger (Excel→PDF tool → K4). ~10/13 screens want a summary strip. Finance cluster is
  the hot zone (multi-panel, dual-status, transition-gated). "Fix don't preserve" gaps:
  Expenses approve/reject/post no-confirm + hardcoded reason; CreditNotes apply no-confirm.
- **K1 engine spine (2026-07-14, commit b885736):** Built + tested + Playwright-verified
  the shared engine features (≥2 screens each): summary strip (LedgerDescriptor.summary +
  LedgerSummary primitive + distribution bar), ColumnSpec.tone, FilterChips counts,
  StatusSpec.transitions + nextStates(), single-source tone palette (tones.ts + --k-tone-*).
  Invoices pilot upgraded with a summary as the reference. bridge/map.ts extracted (goDate/
  str/num). Gates green (check 0/0, test 26, build), detector 0 violations @1440/420.
- **K1 build batch 1 (2026-07-14, commit 8103fbe):** 6 ledgers built+gated+committed —
  Orders, RFQs, Offers, PurchaseOrders, DeliveryNotes, GRNs. All reviewed by orchestrator
  (types, parity honesty, mock adversarial-ness, real mapping, INTEG-gap discipline).
  Highlights: PO 9-state transition machine w/ approval-threshold guard; GRN weighted
  acceptance-rate tone; Offers two-signal validity tone + computed Expired. Registry wired.
  Full gate green (check 0/0 212 files, test 26, build). Playwright detector CLEAN on all
  8 product screens @1440+420. Showcase (dev kitchen-sink, intentional 3000px overflow demo)
  = detector-EXEMPT, not a product screen, will not ship.
- **K1 build batch 2 (2026-07-14, done):** 6 finance ledgers built+gated — Payments
  (Receipts), CreditNotes, SupplierInvoices, SupplierPayments, ChequeRegister, Expenses.
  All reviewed by orchestrator. Highlights: Payments row-aware Reverse form (validates the
  new engine feature from an agent's hands); SupplierInvoices dual-status (match badge +
  payment toned column); SupplierPayments 2-source bridge merge; ChequeRegister cheque
  status machine + row-aware Cancel; Expenses = card-list→TABLE + all 3 "fix don't preserve"
  gaps FIXED (approve/post confirms + operator reject-reason form). All wired.
- **K1 COMPLETE (2026-07-14):** full gate green (check 0/0 224 files, test 26, build).
  Layout-detector CLEAN on all 14 product screens @1440+420. FABLE_WAVE_K1_REPORT.md written
  (per-screen parity + engine features + consolidated deferred ledger + hot-zone preservation).
  Awaiting Fable review before K2. QuotationScreen reclassified (not a ledger) → K4.

## Engine additions mid-K1 (orchestrator-built on agent findings)
- **Row-aware forms** (2026-07-14): bldPipeline (STOP-and-reported per brief) found
  FormModal never received the clicked row, so row-scoped input-capture actions
  (Cancel/Reject/Reverse with a reason, edit-prefill) couldn't use `action.form`.
  Fix: `FormSpec.initial(row?)` + `submit(draft, row?)`, threaded ActionHost→FormModal→
  FormViewModel. Row typed `unknown` (cast at descriptor; ActionHost is the `any` seam).
  Backward-compat (existing 0-arg forms still valid). Green (check 0/0, test 26). This
  unblocks batch-2 reason-on-row actions (PO Cancel, Cheque Cancel/Stale, Expenses
  Reject, Payments Reverse). RFQ's 4-button stage workaround can later fold to 1 form.

## K4 plan (orchestrator, from recon-K4.md) — the bespoke grab-bag, run in tranches
33 screens triaged: 5 auth-chrome, 7 archetype-fit, 11 bespoke-on-primitives, 4 defer-K5/K6,
6 RETIRE. Nearly all already on real Wails bindings.
- **Tranche 1 (building):** archetype-fit fast wins as descriptors — Reports+AHS (Hubs),
  Opportunities/ApprovalsQueue/DataQuality/AuditTrail (ledgers). Proves archetypes cover
  far past "ledgers". bldK4a + bldK4b.
- **Tranche 1 DONE (commit 814dd4f):** Reports+AHS hubs + Opportunities/Approvals/
  DataQuality/AuditTrail ledgers. 28 screens total on the kernel. Archetype polish: Hub
  hides empty widgets + period wraps; PageShell actions cap to header width.
- **REFINED PLAN — auth chrome moved to K5** (the auth FLOW lives in the app shell; building
  isolated login/activation forms now has little demonstrable value). Stepper primitive also
  → K5 (SetupWizard is part of first-run flow).
- **Tranche 2 DONE (commit 87217d6):** FX Revaluation (ledger, 2-state per fx.go),
  Serial Trace (bespoke, first ColumnSpec.cell L4 use), Notifications (bespoke feed),
  Pricing (bespoke + new RangeSlider control). 32 screens on the kernel.
- **Tranche 3-light DONE (commit 4d03d9d):** Customer360 (bespoke detail, 3 tabs) +
  BookBankReconciliation (bespoke) + NEW BalanceComparisonPanel primitive. 34 screens on
  the kernel. Detector CLEAN @1440+420. Mock-data polish note: a BookBankRecon row shows
  status "Reconciled" with a non-zero variance (list/panel data pairing quirk — not a bug).
- **REMAINING K4 (the hard L-monsters + deferred — genuinely multi-session):**
  Settings (SPLIT into general/bank-accounts/currency/business-rules + retire Deployment tab),
  Accounting (GL/journal/CoA console, no new primitive), CostingSheet (line-items editor +
  doc workspace), BankReconciliation (+AllocationMatchPanel primitive), Butler (+chat-transcript
  primitive), Payroll (PII; confirm $lib/api/payroll transport). Deferred: DeploymentHub,
  PeopleHub, WorkHub, OneDriveImport (+operational-hub tabbed-console primitive).
- **Windows gotcha for agents:** `entity.svelte.ts` VM collides case-insensitively with
  `Entity.svelte` component (svelte-check ambiguous-import). Use a distinct stem (kebab vs
  Pascal differ) or a `-vm.svelte.ts` suffix.
- **New primitives still to build:** BalanceComparisonPanel (BookBankRecon), AllocationMatchPanel
  (BankRecon — reusable AR/AP), chat-transcript (Butler), operational-hub tabbed console
  (PeopleHub/WorkHub/maybe Settings/Accounting left-nav).
- **Tranche 3 (L-monsters, sequenced by risk — the hard remainder):** Settings (SPLIT + retire its Deployment
  tab), FXRevaluation, Accounting, CostingSheet, BankRecon (build AllocationMatchPanel),
  BookBankRecon (BalanceComparisonPanel), Payroll (PII — confirm $lib/api/payroll transport
  first), Butler (chat-transcript primitive; subsumes IntelligenceHub). NotificationsScreen.
- **Defer to K5/K6:** DeploymentHub, PeopleHub, WorkHub, OneDriveImport (need a shared
  operational-hub tabbed-console primitive; OneDrive is currently disabled/unrouted).

## K4 RETIRE — ✅ OWNER-RATIFIED 2026-07-14 ("let's retire them, brother")
All 6 below are RETIRED with owner sign-off. Not rebuilt in the kernel. The old
`frontend/` files stay as reference until the K6 flip deletes the whole tree. For
CashPositionWidget, its 4 Go bindings are KEPT (re-plumb into a future cash-position
tile); only the .svelte file retires. Recorded here + in the K4 report as the sign-off.
1. **IntelligenceHub** — 58-line zero-logic wrapper around ButlerScreen (route Intelligence
   → rebuilt Butler instead).
2. **EntityDiscoveryScreen** — dead/unreferenced D3 graph explorer (internal debug tool).
3. **ArchaeologistScreen** — dead/unreferenced file-scanner dev-tool; arbitrary server-side
   path scan is a security smell for an ERP screen.
4. **ArrivalCeremony** — unreachable (DEMO_SETUP_SCREEN=false), demo scaffold, legacy branding.
5. **EcosystemDashboard** — non-Wails local-runtime dev/research tool (Edge-tab scraping),
   unreferenced, out of ERP scope.
6. **CashPositionWidget** — orphaned, redundant with StatTileGrid+ListWidget; KEEP its 4 Go
   bindings, drop the file (re-plumb into a future cash-position tile).
All 6 are dead/unreferenced/dev-tooling/demo. Owner ratifies during review.

### K4-deferred RETIRE — ✅ OWNER-RATIFIED (Sprint 2)
7. **DeploymentHub → Activity tab / user-activity monitor** (weekly per-employee productivity/
   surveillance report; `UserActivityMonitorPanel` + `CanViewUserActivityMonitoring` +
   `GetWeeklyUserActivityReport`) — RETIRED ENTIRELY, owner-ratified. Surveillance-adjacent,
   out of scope for the OSS kernel. The rebuilt DeploymentHub ships only Audit/Checklist/Support;
   the two activity bindings are dropped from the kernel bridge entirely (no adapter, no mock).

## K3 COMPLETE (2026-07-14, commits d7d1531 + 0206ed9)
4th archetype `Hub` + 8-widget data-viz library (donut/distribution h+v/ranked/stat-grid/
list/activity/callout/comparison, no chart lib) + 4 dashboards (main Dashboard, Finance
Overview, CRM Customer, CRM Supplier). Categorical palette --k-series-* CVD-validated.
Drill-downs proven live (AR KPI → Invoices Overdue). Responsive harness nav (off-canvas
≤720px) so screens get full width. Gate green (check 0/0 249 files, test 26, build).
Detector CLEAN on all 21 product screens @1440+420 w/ widgets. FABLE_WAVE_K3_REPORT.md.
Campaign: 3/6 waves done; 22 screens on the kernel; 4 archetypes built.

## K3 rulings (orchestrator, from recon-K3a + recon-K3b)
- **Real Hub targets = 4 dashboards:** DashboardScreen (main), Finance Overview
  (FinancialDashboard), CRM Customer Overview, CRM Supplier Overview. AHSDashboard =
  a division-subset variant of Finance Overview (ledger/consolidate). Sales Hub dashboard
  = net-new (optional).
- **NOT Hubs (→ K5 nav / K4):** FinanceHub/SalesHub/CRMHub/OperationsHub = tab-shell
  navigators → become a K5 `TabShell`. IntelligenceHub = Butler chat wrapper → K4 bespoke.
  PeopleHub = directory+payroll (PII-sensitive) → K4/EntityMaster later. WorkHub = kanban
  workspace → K4 bespoke.
- **Chart lib: NONE** — all dashboards hand-roll div/CSS bars; kernel widgets are SVG/CSS
  on the palette. No dependency added. Sparkline/time-series NEVER built anywhere → no
  line-chart widget in v1 (KPI delta text covers trend).
- **Widget library v1:** KPI tile (+delta+tone+nav), distribution-bar (h/v, tone),
  ranked-bar-list (Top-N), stat-tile-grid (tone thresholds), list, activity-feed, callout,
  **donut** (NEW anti-card win for grade/type mix — owner-mandate-driven, no old precedent),
  comparison-bars (YoY), bespoke slot. Types in kernel/hub.ts.
- **Categorical palette** --k-series-1..6 (blue/aqua/yellow/green/violet/red) added to
  kernel.css; CVD-VALIDATED on white surface (worst adjacent ΔE 24.2; aqua/yellow contrast
  WARN satisfied by the relief rule — every widget ships labels/legend). Status stays with
  --k-tone-* (reserved, never a series slot).
- **Drill-downs** wired: HubDescriptor KPIs/widgets carry `nav: NavIntent{key,query}`;
  App.navigate() switches screen + seeds initialQuery into the target ledger (parity #4).
  initialQuery added to EntityMaster too.
- **Build split:** orchestrator built the engine (hub.ts types, HubViewModel, Hub.svelte
  archetype, DonutWidget, nav); bldWidgets agent built the 7 mechanical widgets; dashboard
  descriptors → agents next. Per-widget independent async (backend resilience) = ledgered
  ENGINE for K5 (K3 mock = one fetch).

## K2 COMPLETE (2026-07-14)
Built+gated: Suppliers (EntityMaster), Users (read-only EntityMaster, RBAC-safe),
Customers widened to CustomerFullProfile, Inventory Fulfillment (read-only ledger).
Engine adds: summary on EntityMaster, ProfileKpiSpec.tone (credit-blocked balance red),
LedgerSummary legend-truncation fix (benefits all summary strips). Fix-don't-preserve:
Suppliers+Users phantom-status → honest 2-state from is_active; Users search widened.
Deferred to K4: PricingScreen (bespoke simulator, mock data) + Customer360 (graph SLOT).
Gate green (check 0/0 230 files, test 26, build). Detector CLEAN on all 18 product
screens @1440+420 WITH profile panels open. FABLE_WAVE_K2_REPORT.md written.

## K2 rulings (orchestrator, from recon-K2.md)
- **Classifications:** Suppliers→EntityMaster (fold SupplierDetailView); Users→EntityMaster
  (thin profile, RBAC hot-zone, read-only in K2); Customers pilot→WIDEN to CustomerFullProfile
  (trn/industry/credit-blocked/AR-aging/RFQ-winrate — pilot was incomplete); Inventory
  Fulfillment→read-only DocumentLedger (K1-shaped). **DEFER to K4:** PricingScreen (bespoke
  margin simulator, hardcoded mock data — not an entity) + Customer360 (graph SLOT +
  regime-prediction DEFER; stays a separate bespoke screen, NOT folded).
- **Fix-don't-preserve:** Suppliers & Users both fake a `status` field via `|| 'Active'`
  (real field is `is_active` bool) → derive honest 2-state, drop the phantom Pending.
  Widen Users searchText (was name+username only).
- **Deferred ENGINE (later mini-wave):** `profile.tabs`, `profile.slots` nested CRUD
  sub-ledgers (contacts/notes/issues — shared by Suppliers+Customers), entity-graph slot,
  create-vs-edit field requiredness, app-shell nav router (every K2 screen drills cross-screen).
- **Engine added:** summary strip now renders on EntityMaster too (was DocumentLedger-only).
- **Profile-detail fetch (GetXFullProfile) NOT wired in K2:** mock rows carry full profile
  data; real maps list-fetch fields + INTEG-blanks profile-only KPIs. Ledgered, no engine
  profile-fetch added (matches the Customers-pilot approach).

## K1 ruling: scope = the ledger spine
K1 delivers list+paging+status+filters+search+summary+simple actions to parity, and
HONESTLY LEDGERS the deep features (multi-panel composition, dual-status rows, FX/line-item/
receive forms, cross-screen handoffs, real mutations) as SLOT/INTEG/ENGINE for K5. Real
bridge = fetch wired, mutations INTEG-gap throw (proven pilot pattern). 12 real K1 ledgers
(Quotations dropped).
