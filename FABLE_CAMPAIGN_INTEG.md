# Campaign — INTEG Execution (Task #4): wire the kernel frontend to real bindings

**You are the incoming orchestrator + technical lead** for INTEG — closing every
`INTEG gap: <Binding>` throw in `frontend-lab/` against real Wails bindings, validated on a
throwaway runtime. The kernel migration (K1–K6 flip-prep) is DONE and merged to main; this campaign
is the last owner-gated work before the K6 flip (Task #5, **NOT in this campaign's scope**).

Same operating model as Sprints 1–3: **Sonnet 5 agents code, you gate every wave and fix what they
miss.** Enforce the kernel laws (L1–L7). The INTEG discipline is already in the codebase — respect it:
a mutation either persists for real or throws an honest `INTEG gap:` — it must NEVER silently claim
to have persisted.

**Read FIRST, in order:** `CLAUDE.md` → `frontend-lab/KERNEL.md` → `FABLE_WAVE_K6_PARITY.md`
(the INTEG roster in §"Consolidated INTEG roster" is your work-list; the sign-off table is your
scoreboard) → `FABLE_KERNEL_CAMPAIGN_LOG.md` (every ruling/gotcha) → `FABLE_CAMPAIGN_SPRINT3_HANDOFF.md`
§2d + §4 (parked owner rulings) → this file.

**Branch:** `exp/frontend-kernel` (worktree `asymmflow-lab`). The branch was merged to main at
`c29e17a` **minus the `wails.json` repoint** (that's flip step 2). On this branch `wails.json`
correctly points at `frontend-lab` for dev.
⚠️ **Do NOT `git merge main` into this branch casually** — git will resolve `wails.json` back to
`frontend` and break lab `wails dev`. If you must sync main, re-assert the repoint in the same commit.
**Never push this branch.** Merges to main go through the owner's review gate.

---

## 0. The owner ruling that shapes this campaign (Sprint 2, commit `1779b3c`)

Close INTEG toward the **sovereign mesh** (Era-3): validate the real adapters against the
**owner's local PostgreSQL** — NOT the legacy DuckDNS remote-Postgres sync (Era-1, retired; never
enable it), NOT the live PH SQLite at `%APPDATA%\Roaming\AsymmFlow` (remote sync is ENABLED there —
never touch it). The sync layer becomes a Holesail P2P sidecar — a separate campaign running in
parallel (worktree `asymmflow-mesh`, `mesh/` dir — **hands off `mesh/`**).

**Verified 2026-07-15:** PostgreSQL 16.14 running (Windows service `postgresql-x64-16`), psql at
`C:\Program Files\PostgreSQL\16\bin\psql.exe`, throwaway DB **`asymmflow_integ`** created and
connectable. Credentials live in **`.env.integ.local`** (repo root of this worktree, gitignored).
**NEVER commit, log, echo, or paste the DSN/password into any committed file, report, or campaign log.**

---

## 1. Wave I0 — PG runtime spike (TIMEBOXED, decision-gated)

The app's primary DB is hard-wired SQLite: single open site
`pkg/runtime/composition/composition.go:59` (`OpenSQLite`, gormlite/ncruces). The
`gorm.io/driver/postgres` dep already exists (used by the retired remote-sync layer in
`db_manager.go`). But the boot path is SQLite-shaped: DSN PRAGMAs + manual PRAGMAs
(`app.go:480–539`), `sqlite_master` table counts (`app.go:592,744`), and — the big one — the
CHECK-constraint migration machinery that does `PRAGMA writable_schema` surgery on `sqlite_master`
(`app.go:1984–2046`). ~52 sqlite-isms repo-wide, though many are in tooling/backup, not boot.

**The spike:** add a dev-only escape hatch — if env `INTEG_PG_DSN` is set (read it from
`.env.integ.local`), the composition seam opens `postgres.Open(dsn)` instead of gormlite, and every
SQLite-specific boot step is gated behind a dialect check (`db.Dialector.Name() == "sqlite"`):
PRAGMAs skipped, `sqlite_master` queries get an `information_schema` equivalent or are skipped with a
log line, the writable-schema CHECK-constraint migrations are **skipped on PG** (AutoMigrate creates
fresh-correct tables on an empty DB — the surgery only exists to mutate *existing* SQLite schemas).
Target: **app boots green on `asymmflow_integ`, AutoMigrate lands the full model set, license keys
seed, `cmd/devkey` (add a PG mode) prints an admin key, you activate and see the Dashboard.**

**Decision gate (report to owner before proceeding past it):**
- **Outcome A** — boot green with modest gating (roughly: composition seam + app.go boot path only):
  PG is the validation runtime for I1–I3. Proceed.
- **Outcome B** — it's a rat's nest (migration machinery, dialect-incompatible raw SQL in binding
  paths, >~a day of yak-shaving): **STOP. Default ruling (owner pre-ratified):** fall back to a
  quarantined **scratch SQLite** as the validation runtime for I1–I3 — the frontend seam talks to Go
  bindings, never SQL, so the wiring work is identical — and "primary-on-Postgres" graduates to its
  own backend campaign aligned with the mesh. Record findings in the campaign log either way.

Backend edits in this wave stay **minimal and additive** (composition seam + boot gating + devkey).
`go build ./...` + `go test ./...` stay green with the env var UNSET (pure SQLite path untouched).

## 2. Wave I1 — Cross-cutting prerequisites (build once, unblocks many)

From the K6 parity roster:
1. **Session/currentUser real wiring** — the store exists (`src/stores/session.svelte.ts`); replace
   every placeholder actor (`actor='lab-user'` in BankRecon etc.) with the real session identity from
   the license-activation flow. Grep for `lab-user` — zero hits when done.
2. **Divisions registry real wiring** — `src/stores/divisions.svelte.ts` exists with synthetic
   fallback; wire `GetDivisionRegistry` for real (AHS + payment/invoice division scoping).
3. **date → `time.Time` form bridge** — one kernel-level bridge in the form layer (the
   `SetExchangeRate` blocker); NOT per-screen conversions (L2).
4. **Secrets storage for AI provider keys** — still a parked owner decision (Settings DEFER).
   Surface it in your wave report; do not improvise a storage scheme.

## 3. Wave I2 — Read swaps (mock → real; low risk, straight `pick()` swaps)

The roster, verbatim from `FABLE_WAVE_K6_PARITY.md` — accuracy notes included (they were verified
against the actual bridges, trust them over older roll-ups):
- **Dashboards:** main = a **3-binding composition** (`GetDashboardStats` + pipeline + AR-aging YTD),
  NOT a single `GetDashboardData`; CRM customer/supplier; AHS-by-division; finance-overview's primary
  read is `GetFinancialDashboardForYear` (currently mock).
- Opportunities 2-source fetch; `GetCustomer360`; Serial Trace searches; Audit Trail chain;
  Approvals/Notifications fetches.
- **Secondary-fetch depth (blank-till-wired):** `GetCustomerFullProfile`, `GetSupplierFullProfile`,
  `GetCashPosition` (live-cash overlay).

Seed the runtime DB with **synthetic canon data only** (adversarial: RTL, long strings, zero/negative
amounts) — repo law #2, no real client data, ever. Per screen: swap → exercise in `wails dev` →
tick the parity table row → gate (`node tests/gate.mjs "<labels>"`).

## 4. Wave I3 — 🔥 Financial hot-zones (wire LAST, each with tests)

Irreversible/money-moving mutations, from the roster: invoice send/PDF + edit/proforma/credit-override;
`ReverseCustomerReceipt`; `ApplyCreditNote`; supplier-invoice 3-way-match/approve/pay;
`DeleteSupplierPayment`; `PostFXRevaluation`/`ReverseRevaluation`; `FinalizeBookBankReconciliation`;
`FinalizeReconciliation`/`DeleteBankStatement`; `CreateJournalEntry`; PO Receive Items; GRN
Receive/Complete; payroll generate/approve/post; `SaveCostingAsOffer`; `DeleteRFQWithCascade`;
`ImportOneDriveDeals`; delete-approval reviews.

Per flow, the bar is higher than I2: wire → drive the flow end-to-end in the running app → **verify
persisted state AND the audit trail in the DB directly** (psql or sqlite3 per the I0 outcome) →
verify the reversal path where one exists (reverse-receipt, un-finalize, credit-note chain) → tick
the parity row. Batch by domain (AR, AP, recon/FX, inventory-docs, payroll, costing/CRM) — one wave
report each.

## 5. Gates, safety rails, and coordination

- **Per-wave gates:** `npm run check` 0/0 · `npm run test` all green · `npm run build` clean ·
  `node tests/gate.mjs` (subset per touched screens; full sweep at campaign end) · `go build ./...` +
  `go test ./...` (I0 and any backend touch).
- **Quarantine, ALWAYS (even with PG primary):** `export PH_DB_PATH=<scratch>` +
  `export APPDATA=<scratch>` before any `wails dev`. ⚠️ The "PowerShell" tool executes via bash —
  `$env:` assignments silently no-op; use `export`. The live `%APPDATA%\Roaming\AsymmFlow` has
  remote sync ENABLED — touching it is the one unrecoverable mistake available to you.
- **INTEG-throw discipline:** any binding you do NOT wire keeps its honest throw. No silent mocks.
- **Scoreboard:** every wired screen flips its row in `FABLE_WAVE_K6_PARITY.md` (`mock-INTEG` →
  `real`, mutation ☐ → status) and gets a line in `FABLE_KERNEL_CAMPAIGN_LOG.md`.
- **Parallel campaign:** sovereign-mesh Wave 1 runs concurrently in the `asymmflow-mesh` worktree.
  Disjoint by design — you never touch `mesh/`; it never touches `frontend-lab/`. Your only backend
  surface is the I0 seam; keep it that way.
- **Out of scope:** the K6 flip (repoint/embed/delete `frontend/`) — owner-gated Task #5; the
  Holesail sync sidecar; pushing anything.

## 6. The prize

Every row of the K6 parity table honestly `real`, every hot-zone mutation proven against a throwaway
DB with its audit trail intact — so the only thing left between the kernel and production is the
owner's smoke checklist and the flip itself.
