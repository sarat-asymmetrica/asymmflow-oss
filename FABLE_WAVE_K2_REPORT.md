# FABLE Wave K2 — Entity Blitz — Report

**Branch:** `exp/frontend-kernel` (LOCAL-ONLY) · **Date:** 2026-07-14
**Orchestrator:** Opus 4.8 · **Coders:** 3× Sonnet 5 (1 recon + 2 build), gated.

## 1. What K2 delivered

The entity-master family, on the `EntityMaster` archetype (+ one read-only ledger):

| Screen | Entity | Archetype | Notable |
|---|---|---|---|
| Suppliers | suppliers | EntityMaster | folded SupplierDetailView profile; 2-state status fix; Delete |
| Customers (widened) | customers | EntityMaster | pilot widened to CustomerFullProfile: TRN/industry/credit-blocked/AR-aging/RFQ-winrate + summary + credit-blocked KPI tone |
| Users | users | EntityMaster | read-only (RBAC hot-zone); role distribution w/ Admin flagged; no password anywhere |
| Inventory Fulfillment | inventory-fulfillment | DocumentLedger | read-only report; pending/shortage tone columns + shortage summary |

**Deferred to K4 (bespoke), with honest ledgers — NOT force-fit into the archetype:**
- **PricingScreen** — a margin-strategy simulator whose "customer list" is hardcoded
  fixture data with no backend binding; not an entity master.
- **Customer360** — a relationship-graph + payment-regime-prediction analytics view;
  genuinely different data/purpose; stays a separate bespoke screen (graph = SLOT,
  predictions = DEFER). NOT folded into the customers profile.

Per-screen parity ledgers in `frontend-lab/src/screens/parity/` (Suppliers, Users,
InventoryFulfillment, Customers).

## 2. Fix-don't-preserve (correctness catches)

- **The phantom `status` field.** Both `SupplierMaster` and `User` have NO `status`
  string field — the old screens faked one via `row.status || 'Active'`, showing 3-state
  Active/Inactive/Pending tabs where "Pending" never had backing data. K2 derives an honest
  **2-state** status from the real `is_active` bool and drops the phantom. A status filter
  that silently always reads one value is a trust issue — doubly so next to Users (security).
- **Users search widened** — the old screen searched name+username only; now sweeps
  email+department+role too (every other screen already searches broadly).

## 3. Security / RBAC (Users)

- Users is **read + summary + profile only** in K2. Create/Edit/Deactivate (which set the
  privilege-bearing `role_id` / `is_active`) are ledgered INTEG — they wire at K5 through the
  real server-RBAC-gated bindings, never via optimistic local mutation.
- **Password/password_hash never appears** in any row, column, or profile field (verified in
  the bridge). `mustChangePassword` is a boolean flag only, not a credential.
- Role distribution flags **Administrator** in warning tone — a small RBAC-hygiene signal
  (role sprawl is visible at a glance).

## 4. Engine features added (≥2 screens)

1. **Summary strip on EntityMaster** — was DocumentLedger-only; now Suppliers/Customers/
   Users all get the KPI-strip + distribution bar.
2. **`ProfileKpiSpec.tone`** — profile KPI values can carry a semantic tone (mirrors
   ColumnSpec/SummaryMetric tone). Surfaced by a build agent's finding; used to show a
   **credit-blocked customer's Balance in danger red**.
3. **LedgerSummary legend truncation fix** — long distribution labels (e.g. "Warehouse
   Supervisor") now ellipsis-truncate instead of forcing a 2px card overflow at 420px.
   Caught by exercising profile panels at the tightest width; benefits every summary strip
   in K1+K2.

## 5. Architecture notes

- **Profile-detail fetch not wired in K2.** Rich profile KPIs (supplier purchases, customer
  AR aging) come from a *second* binding (`GetSupplierFullProfile` / `GetCustomerFullProfile`),
  not the list fetch. Mock rows carry full data; real adapters map the list fields and
  INTEG-blank the profile-only ones — ledgered, no engine profile-fetch added (matches the
  Customers-pilot approach). Wires at K5.
- **List columns = the real list SELECT.** `ListSuppliers` prunes tax_id/brands/address/bank
  for list-view performance; those are profile-only. The descriptor's list columns match the
  real query, so nothing renders silently blank against real data.

## 6. Deferred ledger (later waves)

- **ENGINE (a focused mini-wave):** `profile.tabs` (Suppliers+Customers detail views both use
  a 5-tab shape); `profile.slots` nested CRUD sub-ledgers (contacts/notes/issues strips —
  shared by Suppliers+Customers); create-vs-edit field requiredness; **app-shell nav router**
  (every K2 screen drills cross-screen — now well past the threshold, a K5 concern).
- **SLOT:** Customer360 relationship graph; supplier issues/notes panels.
- **INTEG:** all mutations; profile-detail fetch; cross-screen nav.

## 7. Gate results (green)

- `npm run check` — **0 errors, 0 warnings, 230 files**
- `npm run test` — **26 passing**
- `npm run build` — clean
- **Layout-detector — CLEAN on all 18 product screens at 1440 + 420, WITH the detail/profile
  panel open** (profiles exercised, not just list views), against adversarial fixtures.
  (Showcase dev kitchen-sink still exempt.)

## 8. Verdict

K2 is at flip-grade for the entity family: master lists + rich profiles + summaries at parity
or better, honest status, RBAC-safe Users, two bespoke screens correctly deferred rather than
mangled into the archetype. Gates green, detector clean. Ready for review before K3 (Hub
archetype + dashboards — where the anti-card-heavy mandate gets its biggest canvas).
