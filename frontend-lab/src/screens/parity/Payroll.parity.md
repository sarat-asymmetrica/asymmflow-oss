# Payroll — parity notes

**Entity:** `payroll` · **Group:** People · **Archetype:** bespoke (K4 L-monster)

## Old screen

`PayrollScreen` (1167 lines) — a Finance-Hub-embedded workspace with
compensation/runs/payouts/workspace modes, gated by an `embedded`/`mode` prop
and a `company` (division) prop. **Transport confirmed: Wails IPC only**
(`window.go.main.{App,FinanceService}`) — verified against
`frontend/wailsjs/go/main/{App,FinanceService}.d.ts` and the `payroll` namespace
in `models.ts`. **This resolves Sprint-1 owner open-question #4: payroll is never
HTTP.** PII hot-zone: salary/allowance/deduction amounts + employee names (there
is NO employee IBAN — the bank picker is the company's own disbursing account).

## This build

Three `ViewSwitcher` modes (horizontal) over one division-scoped dataset:
- **Compensation**: upsert form (kernel `k-field`/`k-input` controls) + profiles `DataTable`.
- **Runs**: period-create form, a `Grid` split of Periods/Runs `DataTable`s, then the
  selected run's approve→post→pay lifecycle on the new **`Stepper`** primitive; its
  `detail` slot holds run-total `StatTileGrid` + per-employee items `DataTable` +
  approval-reason field + Mark-Paid sub-form.
- **Payouts**: a flat `DataTable`.
Always-on: `StatTileGrid` (Active Profiles / Open Periods / Approved-or-Posted /
Upcoming Liability) + `DistributionWidget` of run counts by state. All
arithmetic/derivation lives in `payroll-vm.svelte.ts` (L5).

## Capability census

| # | Old capability | Verdict | Notes |
|---|---|---|---|
| 1 | Compensation/Runs/Payouts/Workspace mode prop | **EQUIV** | `ViewSwitcher` (new primitive) replaces the hand-rolled mode switch; dropped the redundant `workspace` "show-everything" mode. |
| 2 | 5 list fetches + `GetActiveBankAccounts` | **DONE** | All 6 are real single-call FETCH adapters (no INTEG gap). |
| 3 | `matchesCompany` client-side division scoping | **DONE/EQUIV** | Replicated as `matchesDivision()`; division options are a fixed mock vocabulary until the K5 divisions store (INTEG-noted). |
| 4 | Upsert compensation profile | **INTEG** | `UpsertEmployeeCompensationProfile` — financial + PII hot-zone. |
| 5 | Create payroll period | **INTEG** | `CreatePayrollPeriod`. |
| 6 | Generate payroll run | **INTEG** | `GeneratePayrollRun`. |
| 7 | Approve run | **DONE/FIXED + INTEG** | **FIXED (per Expenses precedent):** added a `ConfirmDialog` (was a bare click) AND replaced the hardcoded note `"Approved from Finance Hub"` with an operator-supplied reason feeding `ApprovePayrollRun`'s notes arg. Mutation INTEG-gapped. |
| 8 | Post run | **DONE/FIXED + INTEG** | **FIXED:** added a `ConfirmDialog` (was a bare click). `PostPayrollRun` INTEG-gapped. |
| 9 | Mark Paid (required ref + bank account) | **DONE/PRESERVED + INTEG** | Same required-field guards; `MarkPayrollRunPaid` INTEG-gapped. |
| 10 | Mark Paid enabled from `approved` OR `posted` | **PRESERVED + FLAGGED** | Kept the old state-machine quirk (Post not strictly enforced before Pay), not silently "fixed" — see Owner Question #2. |
| 11 | Dashboard summary (`ListPayrollDashboardSummary`) | **EQUIV (dead fetch retired)** | The old server fetch was immediately overwritten by a client `.reduce()` — dead code. Summary computed client-side over scoped rows, matching every other ledger and what the old screen actually did. |
| 12 | Employee picker (`collaboration.listEmployeeProfiles`) | **INTEG** | Cross-domain binding outside this bridge's collision-free file; mocked as a synthetic roster, real throws an honest gap rather than cross-wiring another bridge. |
| 13 | `RunItem.components` earning/deduction/employer-cost chips | **SLOT (deferred)** | Needs a `ColumnSpec.cell` ejection for a multi-badge breakdown; data present in the mock, not rendered. |
| 14 | Field-level masking of salary/name | **NET-NEW (not parity)** | Old screen had all-or-nothing screen-level RBAC only. `canViewUnmasked` (defaults **true** = byte-parity) routes every per-employee money/name column through mask helpers; aggregate run totals stay unmasked. A toolbar toggle exercises it. See Owner Question #1. |

## Orchestrator notes
- **Form controls** use the kernel-owned `k-field`/`k-field-label`/`k-input` classes
  (single-source in `styles/kernel.css`), NOT per-screen CSS — the tech lead added
  these to kill the `.bs-input`/`.pr-input` duplication the build surfaced (L1/L2).
  `BusinessSettings.svelte` should migrate to them in a later cleanup pass.
- **Division scoping is client-side** — the mock deliberately includes a legacy-cased
  division string that matches neither canonical chip, keeping the K5 normalization gap
  visible rather than papered over.
- Mock is adversarial (200-char/RTL/empty names, zero/negative/huge/tiny salaries,
  UNKNOWN status → Stepper all-pending, 3+ divisions, posted-with-paid_at contradiction,
  empty & 240-item runs, dangling-FK payout guarded on row-click, invalid effective range).

## Owner questions (ratified defaults applied; surface at review)
1. **Field-masking policy** — `canViewUnmasked` is plumbed but no granular permission
   flips it yet (default true = today's behavior). Confirm the flag shape before K5 wires a real permission.
2. **Post-before-pay** — Mark Paid stays enabled from `approved` OR `posted` (preserved).
   Intentional, or should K5 tighten to `posted`-only?
3. **Approve reason** — added as a free-text field, not required. Should K5 make it mandatory?

## Kernel gap surfaced (resolved by the tech lead)
- No shared form-control primitive → **resolved**: kernel-owned `k-field`/`k-input`
  global classes now exist (`styles/kernel.css`). `ColumnSpec` still has no multi-badge
  cell type (item #13 SLOT), deferred.
