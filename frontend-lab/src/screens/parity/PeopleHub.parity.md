# PeopleHub — parity notes

**Entity:** `people` · **Group:** People · **Archetype:** bespoke hub (K4-deferred)

Old: `PeopleHub.svelte` (1879 lines) + `collaboration.ts`. New: `bridge/people.ts` +
`people-vm.svelte.ts` + `PeopleHub.svelte`. The HIGHEST-PII screen (gov-ID docs, credentials).
Shape: **TabShell** (Directory / Org / Contributions / Payroll) + a shared "Add Employee" composer
in the TabShell `header`. Directory's detail Card hosts an inner **ViewSwitcher** (Profile / Work /
Access / Compliance) over the selected employee — not a nested TabShell. **First TabShell consumer —
validated the primitive (lazy-mount keep-mounted + visible:false permission gate) with zero gaps.**

## Capability census

| Capability | Verdict | Notes |
|---|---|---|
| Directory list + search + active/archive/all filter | **DONE/EQUIV** | DataTable + FilterChips (built-in All) + SearchInput. |
| Add Employee composer | **DONE** | TabShell `header` snippet, shared across tabs. |
| Profile / Work fields + manager reassignment | **DONE / INTEG** | FormGrid (k-field/k-input); same two-call save (update then reassignManager if changed). Adds a guarded "Reporting Chain" breadcrumb. |
| Employment reactivate | **INTEG** | `SetEmployeeEmploymentState`, cosmetic isAdmin gate. |
| Employee archive (HOT-ZONE) | **DONE/FIXED + INTEG** | **FIXED:** now a `ConfirmDialog` with `reasonLabel`+`requireReason` (was a bare textarea+button — archive can no longer fire without an explicit confirm). Cascade consequence stated in the message. `RequestEmployeeArchive` INTEG-gapped. |
| Access: license links + issue/link/reassign license | **INTEG** | `CreateEmployeeAccessLink`/`ReassignEmployeeLicenseAccess`/`GenerateLicenseKey` (HOT — mints a live credential). |
| Access: bind/create login user | **INTEG (HOT)** | `CreateUser` (login credential + temp password); temp-password field never pre-filled, mock never disguises a realistic key. |
| Compliance docs CRUD + expiry countdown | **INTEG + PII HOT-ZONE** | masked `docNumberMasked` by default via `canViewUnmasked` (default true = byte-parity, same pattern as Payroll salary masking). |
| Org tab (manager-grouped) | **EQUIV** | DataTable per manager group (old screen used raw `<button>` rows → DataTable keeps the screen L1-clean). |
| Contributions tab | **EQUIV (richer)** | StatTileGrid + DistributionWidget (task mix) + RankedBarList (completion-rate) vs the old flat card grid. |
| Payroll tab | **DONE (embedded)** | `<Payroll embedded presetEmployeeID={vm.selectedEmployeeId} />`; tab `visible` only when `canViewPayroll`. Payroll's own division FilterChips replaces the old separate division toggle. |
| Snapshot cache (30s TTL) | **DEFER** | Not ported (per HUB_BUILD_CONTEXT); VM loads fresh. |
| Deep-link routing (`params.tab`/`payrollEmployeeID`) | **DEFER** | No router/params in the lab yet; "Set up payroll →" jumps `activeTab='payroll'` locally (TabShell keeps the selection so `presetEmployeeID` resolves). K5 app-shell concern. |
| `openUsersAndAccess()` cross-screen nav | **DEFER** | No registry-level cross-screen nav in the lab; Access tab's admin composers cover the destination. K5. |

## PII / INTEG ledger
FETCH (real): ListEmployeeProfiles, ListEmployeeAccessLinks, ListLicenseKeys, ListEmployeeContributionSummaries,
ListEmployeeProjectAssignments, GetCurrentEmployeeContext, ListEmployeeDocuments (App); ListUsers/ListRoles (InfraService).
MUTATION (13, INTEG-gapped, named): CreateEmployeeProfile, UpdateEmployeeProfile, SetEmployeeEmploymentState,
RequestEmployeeArchive, ReviewEmployeeArchiveRequest, ReassignEmployeeManager, CreateEmployeeAccessLink,
ReassignEmployeeLicenseAccess, GenerateLicenseKey, CreateUser, CreateEmployeeDocument, UpdateEmployeeDocument,
DeleteEmployeeDocument. Mock branch is functional (interactive lab); real branch honestly refuses.

## Stop-and-asks (flagged, not fixed)
1. **Doc-number unmask on Edit** — `editDocument()` auto-populates the DECRYPTED docNumber (same as old), gated only by `canViewUnmasked`. Should Edit require a fresh unmask action instead? Preserved.
2. **`isAdmin` is cosmetic** — client-side role-string match gating panel visibility only; every mutation is server-INTEG-gapped regardless. Never a security boundary.
3. **Manager-cycle guard** — mock seeds emp-20↔emp-21 mutual managers. Org grouping is a flat reduce (immune); `managerChain()` (Work-tab breadcrumb) traverses with a visited-Set + 25-hop cap, verified to terminate.

## Orchestrator notes
- Form controls use kernel `k-field`/`k-input` classes throughout — no per-screen form CSS (L1/L2 clean). No kernel gaps found.
- Adversarial mock (36 employees): 200-char + RTL + Latin-mixed names, orphan leaf, manager cycle, two-flag inconsistency,
  UNKNOWN status, archived employee, 0 and 5 license links, empty + 200-char license displayName, compliance docs with
  past/null/2099 expiry + empty permit subtype. Synthetic Gulf identity only.
