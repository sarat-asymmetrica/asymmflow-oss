# Frontend Kernel Campaign — Orchestrator Log

Durable progress tracker for the K1–K6 full-migration campaign
(`FABLE_CAMPAIGN_FRONTEND_KERNEL.md`). Orchestrator = Opus 4.8; coders = Sonnet 5.
Branch `exp/frontend-kernel` (LOCAL-ONLY). Updated as waves land.

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
| K4 | Bespoke screens on primitives | ⏳ next |
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
