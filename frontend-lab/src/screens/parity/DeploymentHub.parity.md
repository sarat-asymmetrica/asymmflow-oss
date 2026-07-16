# DeploymentHub — parity notes

**Entity:** `deployment` · **Group:** System · **Archetype:** bespoke hub (K4-deferred)

Old: `DeploymentHub.svelte` (1093 lines). New: `bridge/deployment.ts` + `deployment-vm.svelte.ts` +
`DeploymentHub.svelte`, on the `TabShell` primitive (Audit / Checklist / Support).

## OWNER-RATIFIED RETIREMENT (Sprint 2)
**Activity / weekly per-employee user-activity monitor — RETIRED ENTIRELY.** The old screen's 4th tab
wrapped `UserActivityMonitorPanel` (weekly per-employee productivity/"efficiency" report — active
hours, meaningful hours, search/create/update/export counts, gated on `CanViewUserActivityMonitoring`).
Owner ruled full retirement mid-build (surveillance-adjacent, out of scope for the OSS kernel). NO
`canViewActivityMonitoring` flag, NO activity bindings (`CanViewUserActivityMonitoring`,
`GetWeeklyUserActivityReport`), NO activity types/UI/mock anywhere. Ships with exactly 3 tabs.

## Capability census

| Capability | Verdict | Notes |
|---|---|---|
| Summary strip (Deployment Audit / Pilot Readiness / Needs Attention / Checklist Progress / Queue Health) | **DONE** | `StatTileGrid` in TabShell's `header` snippet. |
| Refresh | **DONE** | Reloads the workspace. |
| Export Sign-Off / Bundle | **EQUIV (dedup)** | Consolidated to the Support tab (old screen duplicated them in header + tabs). |
| Deployment Data Audit panel (blocking/missing_tables/counts) | **DONE** | Placed in the Audit tab. |
| Readiness list (search + issues-only filter + select) | **DONE** | DataTable + SearchInput + FilterChips. |
| Readiness detail (issue pills, access/license/device/user, last-seen) | **DONE** | StatTileGrid + Badge pills. |
| Access reassignment fix-it (license select, sync-name, display-name save, reassign) | **INTEG** | FormGrid (k-field/k-input); `ReassignEmployeeLicenseAccess`/`UpdateLicenseDisplayName` gapped. |
| Pilot checklist (toggle + notes) | **INTEG** | Card grid; `UpdatePilotDeploymentChecklistItem` gapped. |
| Support actions (sync/retry/export) | **INTEG** | All gapped; mock performs interactively. |
| **Bulk retry (failed/dead-letter) — no confirm in old screen** | **FIXED** | Added a mandatory danger `ConfirmDialog` before a resync-storm. |
| Collaborative queue table (status filter + refresh) | **DONE** | Native k-input select + DataTable. |
| Per-row conditional retry + inline error | **SIMPLIFIED** | select-row + contextual "Retry This Operation" panel + `CalloutWidget` error (a true inline per-row button needs a DataTable `cell` override = a 4th file, outside the collision-free contract — see kernel gap). |
| Queue-by-status visual | **ADDED** | `DistributionWidget` above the queue (visual-diversity vehicle). |
| Snapshot cache (30s TTL) | **DEFER** | Not ported; loads fresh. |

## INTEG / hot-zone ledger (8 mutations, all named)
UpdatePilotDeploymentChecklistItem · TriggerCollaborativeSyncNow · **RetryCollaborativePendingOperations** (HOT bulk,
now confirm-gated) · RetryCollaborativePendingOperation · ExportPilotSupportBundle · ExportPilotSignoffReport ·
ReassignEmployeeLicenseAccess · UpdateLicenseDisplayName — all WIRED (R3; the two pilot exports wired + artifact-proven in G4, returning the on-disk path); mock remains as the lab feature.
FETCH wired real (7): GetPilotReadinessSummary, ListPilotReadinessRows, GetPhase7RolloutStatus, GetPilotDeploymentChecklist,
ListLicenseKeys, GetDeploymentDataAudit, ListCollaborativePendingOperations.

## Orchestrator notes
- Form controls use kernel `k-field`/`k-input` classes (L1/L2 clean). Adversarial mock: 26 readiness rows (0→6+ issues,
  a fully-unlinked employee, 200-char + RTL + empty names), a blocking DeploymentDataAudit with long missingTables, 9
  checklist items (600-char note, 2019 completedAt), 12 license keys (200-char name, several unassigned), 45 queue ops
  across every status (400-char error, non-UUID entityId). Synthetic identity only.

## Kernel gap (K5 enhancement candidate)
- **DataTable has no declarative lightweight per-row action** short of a `cell` component override. Recurs (DeploymentHub
  queue retry; OneDriveImport per-deal actions). Candidate: a `ColumnSpec.rowAction` (label + visibility predicate + onClick)
  rendering inline without ejecting to a component. Flagged for K5; not blocking (contextual-panel workaround used).
