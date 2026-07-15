/* Deployment Hub bridge module — self-contained: types + mock + real + switch.
 * Internal ops/pilot console (K4 operational hub): pilot-readiness audit,
 * deployment data audit, pilot checklist, and collaborative-sync support
 * tools. Ported from `frontend/src/lib/screens/DeploymentHub.svelte` (1093
 * lines). The old screen's Activity tab (weekly per-employee user-activity /
 * "efficiency" monitor, `UserActivityMonitorPanel.svelte`) is RETIRED
 * entirely — owner-ratified (surveillance-adjacent, out of scope for the OSS
 * kernel tranche); see DeploymentHub.parity.md. Real adapters
 * bind `$wails/go/main/InfraService` (FETCH — confirmed against
 * `frontend-lab/wailsjs/go/main/InfraService.d.ts`); `ExportPilotSupportBundle`
 * is bound on `App` in the old screen (also present on InfraService, but the
 * old screen imports it from App — followed here) and
 * `ReassignEmployeeLicenseAccess` is bound on `SyncServiceBinding`. FETCH
 * bindings are real (each is a single-call, non-aggregating list/get); every
 * mutation is internal-ops-with-real-teeth (sync triggers, bulk retry-storms,
 * license reassignment) and throws an honest INTEG-gap naming the exact real
 * binding — see DeploymentHub.parity.md for the full ledger. Synthetic-only
 * mock data (SYNTHETIC_IDENTITY.md) — invented names, no real employees.
 */
import { pick } from './runtime'
import { goDate, num, str } from './map'
/* Only the FETCH bindings are actually invoked below — mutations are
 * INTEG-gap throws that NAME the real binding without importing it (same
 * convention as bridge/payroll.ts: importing a binding this file never calls
 * would just be an unused-import lint trap). */
import {
  GetDeploymentDataAudit,
  GetPhase7RolloutStatus,
  GetPilotDeploymentChecklist,
  GetPilotReadinessSummary,
  ListCollaborativePendingOperations,
  ListLicenseKeys,
  ListPilotReadinessRows,
} from '$wails/go/main/InfraService'

/* ---- types (camelCase; mirrors main.PilotReadinessRow / main.DeploymentDataAudit / …) ---- */

export interface PilotReadinessRow {
  employeeId: string
  employeeCode: string
  employeeName: string
  department: string
  jobTitle: string
  employmentState: string
  accessStatus: string
  licenseKey: string
  licenseRole: string
  licenseActive: boolean
  licenseAssignedTo: string
  deviceId: string
  deviceName: string
  deviceStatus: string
  lastSeenAt: string
  userId: string
  userName: string
  readyForPilot: boolean
  issues: string[]
}

export interface PilotReadinessSummary {
  generatedAt: string
  totalEmployees: number
  readyEmployees: number
  employeesWithIssues: number
  employeesMissingAccess: number
  activatedLicenses: number
  unlinkedLicenses: number
  pendingDevices: number
  blockedDevices: number
  approvedDevices: number
}

export interface Phase7RolloutStatus {
  followupBackfillCompletedAt: string
  followupBackfillCount: number
  legacyFollowupTasks: number
  migratedLegacyTasks: number
  pendingCollaborativeOps: number
  failedCollaborativeOps: number
  deadLetterCollaborativeOps: number
  payrollPayoutsAwaitingRecon: number
}

export interface PilotChecklistItem {
  id: string
  title: string
  description: string
  completed: boolean
  notes: string
  completedAt: string
}

export interface DeploymentDataAudit {
  generatedAt: string
  databasePath: string
  expectedRuntimeDatabasePath: string
  packagedDatabasePath: string
  missingTables: string[]
  blockingDataIssues: string[]
  warningDataIssues: string[]
  blocking: boolean
  taskItems: number
  expenseEntries: number
  payrollRuns: number
  legacyQuotedOfferShells: number
  legacyRfqOfferShells: number
  legacyFollowupTasks: number
  migratedLegacyTasks: number
}

export interface CollaborativePendingOperation {
  id: string
  createdAt: string
  updatedAt: string
  entityType: string
  entityId: string
  operation: string
  status: string
  attempts: number
  errorMessage: string
}

export interface LicenseKey {
  key: string
  role: string
  displayName: string
  activated: boolean
}

export interface Phase7ActionResult {
  status: string
  message: string
  processed: number
}

export interface PilotExportResult {
  path: string
}

export interface PilotSupportBundleResult {
  path: string
  rows: number
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts pattern) ---- */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number): string => String(n).padStart(w, '0')
const todayIso = (): string => new Date().toISOString().slice(0, 10)

const MONSTER_EMPLOYEE_NAME =
  'SENIORFIELDINSTRUMENTATIONANDCALIBRATIONTECHNICIANFORMERLYOPERATIONSDEPARTMENTNOWREASSIGNEDTOTHEPILOTCOHORTPENDINGRECLASSIFICATION'.padEnd(
    200,
    'X',
  )

const EMPLOYEE_NAMES = [
  'Rania Haddad',
  'Yusuf Al-Marzouq',
  'S. Almousawi',
  'Bilal',
  '', // empty — exercises the row's "Unknown Employee" fallback
  'عبدالعزيز بن سالم الكندري', // RTL
  MONSTER_EMPLOYEE_NAME, // unbroken 200-char token
  'Dana Fakhro',
  'Marwan Sabbagh',
  'Huda Al-Rashid',
  'T. Nasser',
  'Zainab Al-Dosari',
  'Karim Chaudhry',
  'Leila Karam',
  'Faisal Bin Rashid',
]

const DEPARTMENTS = ['Field Operations', 'Finance', 'Warehouse & Logistics', 'Sales', 'HR & Admin', 'Engineering', '']
const JOB_TITLES = ['Field Engineer', 'Accountant', 'Warehouse Supervisor', 'Sales Executive', 'HR Coordinator', 'Instrumentation Technician', '']
const EMPLOYMENT_STATES = ['active', 'probation', 'on_leave']
const ACCESS_STATUSES = ['linked', 'partial', 'unlinked']
const LICENSE_ROLES = ['admin', 'user', 'viewer', 'UNKNOWN_ROLE']
const DEVICE_STATUSES = ['approved', 'pending', 'blocked']

const ALL_ISSUES = [
  'no_license',
  'license_inactive',
  'no_device',
  'device_pending',
  'device_blocked',
  'no_user_link',
  'name_mismatch',
  'stale_device',
]

interface Dataset {
  readinessRows: PilotReadinessRow[]
  summary: PilotReadinessSummary
  rollout: Phase7RolloutStatus
  checklist: PilotChecklistItem[]
  audit: DeploymentDataAudit
  licenseKeys: LicenseKey[]
  queueOps: CollaborativePendingOperation[]
}

let cache: Dataset | null = null

function generate(): Dataset {
  const rand = lcg(20260715 ^ 0x0deb)

  // ---- readiness rows: 26 employees, issue counts spread 0 -> 6, one fully
  // unlinked employee (no license/device/user), monster name/RTL/empty rows.
  const ROW_COUNT = 26
  const readinessRows: PilotReadinessRow[] = []
  for (let i = 1; i <= ROW_COUNT; i++) {
    const name = EMPLOYEE_NAMES[(i - 1) % EMPLOYEE_NAMES.length]!
    const department = DEPARTMENTS[i % DEPARTMENTS.length]!
    const jobTitle = JOB_TITLES[i % JOB_TITLES.length]!

    // Monster: employee-14 is entirely unlinked — no license, no device, no
    // user mapping — the worst-case rollout blocker.
    const fullyUnlinked = i === 14
    let issues: string[] = []
    if (fullyUnlinked) {
      issues = [...ALL_ISSUES]
    } else {
      const issueCount = i % 6 // 0..5, spreads "ready" through "5+ issues"
      for (let j = 0; j < issueCount; j++) issues.push(ALL_ISSUES[(i + j) % ALL_ISSUES.length]!)
    }
    const readyForPilot = issues.length === 0

    const licenseKey = fullyUnlinked || i % 9 === 0 ? '' : `LIC-${pad(((i * 3) % 40) + 1, 4)}`
    const deviceId = fullyUnlinked || i % 11 === 0 ? '' : `dev-${pad(i, 3)}`
    const userId = fullyUnlinked || i % 7 === 0 ? '' : `user-${pad(i, 3)}`

    readinessRows.push({
      employeeId: `emp-${i}`,
      employeeCode: `EMP-${pad(i, 4)}`,
      employeeName: name,
      department,
      jobTitle,
      employmentState: EMPLOYMENT_STATES[i % EMPLOYMENT_STATES.length]!,
      accessStatus: fullyUnlinked ? 'unlinked' : ACCESS_STATUSES[i % ACCESS_STATUSES.length]!,
      licenseKey,
      licenseRole: licenseKey ? LICENSE_ROLES[i % LICENSE_ROLES.length]! : '',
      licenseActive: !!licenseKey && i % 3 !== 0,
      licenseAssignedTo: licenseKey ? name || `Employee ${i}` : '',
      deviceId,
      deviceName: deviceId ? `Workstation-${pad(i, 2)}` : '',
      deviceStatus: deviceId ? DEVICE_STATUSES[i % DEVICE_STATUSES.length]! : '',
      lastSeenAt: fullyUnlinked ? '' : i % 5 === 0 ? '2019-02-11T08:00:00Z' : `2026-07-${pad(1 + (i % 14), 2)}T09:15:00Z`,
      userId,
      userName: userId ? name || `Employee ${i}` : '',
      readyForPilot,
      issues,
    })
  }

  const readyCount = readinessRows.filter((r) => r.readyForPilot).length
  const withIssues = readinessRows.length - readyCount
  const missingAccess = readinessRows.filter((r) => r.accessStatus === 'unlinked').length
  const activatedLicenses = readinessRows.filter((r) => r.licenseActive).length
  const unlinkedLicenses = readinessRows.filter((r) => !r.licenseKey).length
  const pendingDevices = readinessRows.filter((r) => r.deviceStatus === 'pending').length
  const blockedDevices = readinessRows.filter((r) => r.deviceStatus === 'blocked').length
  const approvedDevices = readinessRows.filter((r) => r.deviceStatus === 'approved').length

  const summary: PilotReadinessSummary = {
    generatedAt: todayIso(),
    totalEmployees: readinessRows.length,
    readyEmployees: readyCount,
    employeesWithIssues: withIssues,
    employeesMissingAccess: missingAccess,
    activatedLicenses,
    unlinkedLicenses,
    pendingDevices,
    blockedDevices,
    approvedDevices,
  }

  const rollout: Phase7RolloutStatus = {
    followupBackfillCompletedAt: '2026-06-30',
    followupBackfillCount: 42,
    legacyFollowupTasks: 6,
    migratedLegacyTasks: 118,
    pendingCollaborativeOps: 0, // recomputed below from queueOps
    failedCollaborativeOps: 0,
    deadLetterCollaborativeOps: 0,
    payrollPayoutsAwaitingRecon: 3,
  }

  // ---- checklist: 9 items, one with a very long notes string + a far-past
  // completed_at (regression: "far past" completion dates must still render).
  const CHECKLIST_SEED: { title: string; description: string }[] = [
    { title: 'Verify production build artifact', description: 'Confirm the packaged installer matches the tagged release.' },
    { title: 'Confirm database backup routine', description: 'Automated nightly backup job is enabled and tested.' },
    { title: 'Validate employee license assignments', description: 'Every active employee has a linked, activated license.' },
    { title: 'Test collaborative sync end-to-end', description: 'Push and pull a round-trip change across two devices.' },
    { title: 'Review deployment data audit for blockers', description: 'No missing tables or blocking data issues remain.' },
    { title: 'Confirm device enrollment for pilot cohort', description: 'All pilot devices are approved, none pending or blocked.' },
    { title: 'Walk through support bundle export', description: 'Export a support bundle and confirm it opens cleanly.' },
    { title: 'Sign off on sign-off report contents', description: 'Stakeholder review of the generated pilot sign-off report.' },
    { title: 'Confirm rollback plan documented', description: 'A written rollback procedure exists if the pilot needs reverting.' },
  ]
  const LONG_NOTE =
    'Escalated during the pilot readiness review: three employees in the Field Operations department reported ' +
    'intermittent license activation failures when switching between the pilot device and their prior workstation. ' +
    'Root cause traced to a stale device fingerprint cached from the previous rollout attempt. Workaround applied ' +
    'by clearing the cached fingerprint and re-issuing the license key; a permanent fix is tracked separately. ' +
    'Re-tested on 2019-03-14 with all three employees present — confirmed resolved, but leaving this note attached ' +
    'to the checklist item for audit-trail purposes in case the issue recurs during the next pilot wave.'
  const checklist: PilotChecklistItem[] = CHECKLIST_SEED.map((seed, idx) => {
    const i = idx + 1
    const longNoteItem = i === 3
    const completed = i % 4 !== 0
    return {
      id: `check-${i}`,
      title: seed.title,
      description: seed.description,
      completed,
      notes: longNoteItem ? LONG_NOTE : i % 3 === 0 ? '' : `Reviewed on rollout pass ${i}; no further action needed.`,
      completedAt: completed ? (longNoteItem ? '2019-03-14T10:00:00Z' : `2026-07-${pad(1 + i, 2)}T12:00:00Z`) : '',
    }
  })

  // ---- deployment data audit: singleton — the "Blocking" render path (long
  // missingTables + populated blocking/warning issues) is the demo-worthy
  // state for this frozen seed; see DeploymentHub.parity.md for the
  // "Verified" all-clear path note.
  const audit: DeploymentDataAudit = {
    generatedAt: todayIso(),
    databasePath: 'C:\\Users\\pilot\\AppData\\Roaming\\AsymmFlow\\asymmflow.db',
    expectedRuntimeDatabasePath: 'C:\\Users\\pilot\\AppData\\Roaming\\AsymmFlow\\asymmflow.db',
    packagedDatabasePath: 'C:\\Program Files\\AsymmFlow\\resources\\asymmflow.db',
    missingTables: [
      'payroll_run_items',
      'employee_compensation_profiles',
      'collaborative_pending_operations',
      'pilot_deployment_checklist',
      'license_keys',
      'device_registrations',
      'expense_entries',
      'audit_trail_entries',
    ],
    blockingDataIssues: [
      '3 offers reference a customer_id that no longer exists.',
      'Order ORD-2026-0417 has zero line items but a nonzero total.',
    ],
    warningDataIssues: ['12 legacy quoted-offer shells are hidden from the live offer list.'],
    blocking: true,
    taskItems: 214,
    expenseEntries: 58,
    payrollRuns: 12,
    legacyQuotedOfferShells: 9,
    legacyRfqOfferShells: 3,
    legacyFollowupTasks: rollout.legacyFollowupTasks,
    migratedLegacyTasks: rollout.migratedLegacyTasks,
  }

  // ---- license keys: 12, some unassigned, one 200-char display name.
  const LICENSE_COUNT = 12
  const licenseKeys: LicenseKey[] = []
  for (let i = 1; i <= LICENSE_COUNT; i++) {
    const monsterName = i === 6
    licenseKeys.push({
      key: `LIC-${pad(i, 4)}`,
      role: LICENSE_ROLES[i % LICENSE_ROLES.length]!,
      displayName: monsterName ? 'UNASSIGNEDPOOLLICENSERESERVEDFORNEXTPILOTWAVECOHORTBATCH'.padEnd(200, 'Y') : i % 4 === 0 ? '' : EMPLOYEE_NAMES[i % EMPLOYEE_NAMES.length]!,
      activated: i % 5 !== 0,
    })
  }

  // ---- collaborative queue: 45 ops across every status, a very long
  // error_message (wrap test), and a non-UUID entity_id.
  const ENTITY_TYPES = ['customer', 'offer', 'order', 'invoice', 'payment', 'expense']
  const OPERATIONS = ['create', 'update', 'delete', 'sync']
  const STATUS_CYCLE = ['pending', 'failed', 'dead_letter', 'synced']
  const LONG_ERROR =
    'sync conflict: remote version 14 does not match local base version 11 for entity offer/off-2201-fallback-reconstructed-from-legacy-shell; ' +
    'automatic three-way merge failed because both sides modified the line_items array — manual reconciliation required before this operation ' +
    'can be safely retried; see collaborative_conflict_log for the full diff payload (payload omitted here for brevity, 4.2kb JSON).'
  const QUEUE_COUNT = 45
  const queueOps: CollaborativePendingOperation[] = []
  for (let i = 1; i <= QUEUE_COUNT; i++) {
    const status = STATUS_CYCLE[i % STATUS_CYCLE.length]!
    const nonUuidId = i === 30
    const longError = i === 9
    const attempts =
      status === 'dead_letter' ? 8 + Math.floor(rand() * 8) : status === 'failed' ? 1 + Math.floor(rand() * 4) : 0
    queueOps.push({
      id: `op-${pad(i, 4)}`,
      createdAt: `2026-07-${pad(1 + (i % 14), 2)}T${pad(i % 24, 2)}:00:00Z`,
      updatedAt: `2026-07-${pad(1 + ((i + 2) % 14), 2)}T${pad((i + 3) % 24, 2)}:00:00Z`,
      entityType: ENTITY_TYPES[i % ENTITY_TYPES.length]!,
      entityId: nonUuidId ? 'legacy-import-row-42' : `${ENTITY_TYPES[i % ENTITY_TYPES.length]}-${1000 + i}`,
      operation: OPERATIONS[i % OPERATIONS.length]!,
      status,
      attempts,
      errorMessage: longError ? LONG_ERROR : status === 'failed' || status === 'dead_letter' ? `Sync failed: connection reset (attempt ${attempts}).` : '',
    })
  }
  rollout.pendingCollaborativeOps = queueOps.filter((o) => o.status === 'pending').length
  rollout.failedCollaborativeOps = queueOps.filter((o) => o.status === 'failed').length
  rollout.deadLetterCollaborativeOps = queueOps.filter((o) => o.status === 'dead_letter').length

  return { readinessRows, summary, rollout, checklist, audit, licenseKeys, queueOps }
}

async function mockFetchSummary(): Promise<PilotReadinessSummary> {
  cache ??= generate()
  await sleep(180)
  return { ...cache.summary }
}

async function mockFetchReadinessRows(issuesOnly: boolean): Promise<PilotReadinessRow[]> {
  cache ??= generate()
  await sleep(220)
  return cache.readinessRows.filter((r) => !issuesOnly || r.issues.length > 0).map((r) => ({ ...r, issues: [...r.issues] }))
}

async function mockFetchRollout(): Promise<Phase7RolloutStatus> {
  cache ??= generate()
  await sleep(150)
  return { ...cache.rollout }
}

async function mockFetchChecklist(): Promise<PilotChecklistItem[]> {
  cache ??= generate()
  await sleep(160)
  return cache.checklist.map((c) => ({ ...c }))
}

async function mockFetchAudit(): Promise<DeploymentDataAudit> {
  cache ??= generate()
  await sleep(180)
  return { ...cache.audit, missingTables: [...cache.audit.missingTables], blockingDataIssues: [...cache.audit.blockingDataIssues], warningDataIssues: [...cache.audit.warningDataIssues] }
}

async function mockFetchLicenseKeys(): Promise<LicenseKey[]> {
  cache ??= generate()
  await sleep(150)
  return cache.licenseKeys.map((l) => ({ ...l }))
}

async function mockFetchQueue(status: string, limit: number): Promise<CollaborativePendingOperation[]> {
  cache ??= generate()
  await sleep(200)
  const activeSet = new Set(['pending', 'failed', 'dead_letter'])
  const filtered = cache.queueOps.filter((o) => (status === 'active' ? activeSet.has(o.status) : status ? o.status === status : true))
  return filtered.slice(0, limit).map((o) => ({ ...o }))
}

async function mockUpdateChecklistItem(id: string, completed: boolean, notes: string): Promise<PilotChecklistItem[]> {
  cache ??= generate()
  await sleep(160)
  const item = cache.checklist.find((c) => c.id === id)
  if (!item) throw new Error(`Checklist item ${id} not found`)
  item.completed = completed
  item.notes = notes
  item.completedAt = completed ? new Date().toISOString() : ''
  return cache.checklist.map((c) => ({ ...c }))
}

async function mockTriggerSync(): Promise<void> {
  cache ??= generate()
  await sleep(400)
  for (const op of cache.queueOps) {
    if (op.status === 'pending') op.status = 'synced'
  }
  cache.rollout.pendingCollaborativeOps = cache.queueOps.filter((o) => o.status === 'pending').length
}

async function mockRetryBulk(status: string, limit: number): Promise<Phase7ActionResult> {
  cache ??= generate()
  await sleep(350)
  let processed = 0
  for (const op of cache.queueOps) {
    if (processed >= limit) break
    if (op.status === status) {
      op.status = 'pending'
      op.attempts = 0
      op.errorMessage = ''
      processed++
    }
  }
  cache.rollout.pendingCollaborativeOps = cache.queueOps.filter((o) => o.status === 'pending').length
  cache.rollout.failedCollaborativeOps = cache.queueOps.filter((o) => o.status === 'failed').length
  cache.rollout.deadLetterCollaborativeOps = cache.queueOps.filter((o) => o.status === 'dead_letter').length
  return { status: 'ok', message: `${processed} operation(s) re-queued for retry.`, processed }
}

async function mockRetrySingle(id: string): Promise<void> {
  cache ??= generate()
  await sleep(200)
  const op = cache.queueOps.find((o) => o.id === id)
  if (!op) throw new Error(`Collaborative operation ${id} not found`)
  op.status = 'pending'
  op.attempts = 0
  op.errorMessage = ''
}

async function mockExportSupportBundle(): Promise<PilotSupportBundleResult> {
  cache ??= generate()
  await sleep(300)
  return { path: 'C:\\Users\\pilot\\Documents\\AsymmFlow\\reports\\pilot-support-bundle.zip', rows: cache.queueOps.length }
}

async function mockExportSignoff(): Promise<PilotExportResult> {
  await sleep(300)
  return { path: 'C:\\Users\\pilot\\Documents\\AsymmFlow\\reports\\pilot-signoff-report.pdf' }
}

async function mockReassignLicense(employeeId: string, licenseKey: string, _syncName: boolean): Promise<void> {
  cache ??= generate()
  await sleep(220)
  const row = cache.readinessRows.find((r) => r.employeeId === employeeId)
  if (!row) throw new Error(`Employee ${employeeId} not found`)
  row.licenseKey = licenseKey
  row.licenseActive = true
  row.accessStatus = row.deviceId && row.userId ? 'linked' : 'partial'
}

async function mockUpdateLicenseDisplayName(key: string, displayName: string): Promise<LicenseKey> {
  cache ??= generate()
  await sleep(160)
  const license = cache.licenseKeys.find((l) => l.key === key)
  if (!license) throw new Error(`License ${key} not found`)
  license.displayName = displayName
  return { ...license }
}

/* ---- real: FETCH is wired (single-call, non-aggregating); every mutation
 * is an internal-ops hot-zone (sync trigger, bulk retry-storm, license
 * reassignment) and throws an honest INTEG-gap naming the exact real
 * binding. ---- */

function mapReadinessRow(r: Record<string, unknown>): PilotReadinessRow {
  return {
    employeeId: str(r.employee_id),
    employeeCode: str(r.employee_code),
    employeeName: str(r.employee_name),
    department: str(r.department),
    jobTitle: str(r.job_title),
    employmentState: str(r.employment_state),
    accessStatus: str(r.access_status),
    licenseKey: str(r.license_key),
    licenseRole: str(r.license_role),
    licenseActive: Boolean(r.license_active),
    licenseAssignedTo: str(r.license_assigned_to),
    deviceId: str(r.device_id),
    deviceName: str(r.device_name),
    deviceStatus: str(r.device_status),
    lastSeenAt: goDate(r.last_seen_at),
    userId: str(r.user_id),
    userName: str(r.user_name),
    readyForPilot: Boolean(r.ready_for_pilot),
    issues: ((r.issues as unknown as string[] | undefined) ?? []).map((x) => String(x)),
  }
}

function mapSummary(s: Record<string, unknown>): PilotReadinessSummary {
  return {
    generatedAt: str(s.generated_at),
    totalEmployees: num(s.total_employees),
    readyEmployees: num(s.ready_employees),
    employeesWithIssues: num(s.employees_with_issues),
    employeesMissingAccess: num(s.employees_missing_access),
    activatedLicenses: num(s.activated_licenses),
    unlinkedLicenses: num(s.unlinked_licenses),
    pendingDevices: num(s.pending_devices),
    blockedDevices: num(s.blocked_devices),
    approvedDevices: num(s.approved_devices),
  }
}

function mapRollout(r: Record<string, unknown>): Phase7RolloutStatus {
  return {
    followupBackfillCompletedAt: str(r.followup_backfill_completed_at),
    followupBackfillCount: num(r.followup_backfill_count),
    legacyFollowupTasks: num(r.legacy_followup_tasks),
    migratedLegacyTasks: num(r.migrated_legacy_tasks),
    pendingCollaborativeOps: num(r.pending_collaborative_ops),
    failedCollaborativeOps: num(r.failed_collaborative_ops),
    deadLetterCollaborativeOps: num(r.dead_letter_collaborative_ops),
    payrollPayoutsAwaitingRecon: num(r.payroll_payouts_awaiting_recon),
  }
}

function mapChecklistItem(c: Record<string, unknown>): PilotChecklistItem {
  return {
    id: str(c.id),
    title: str(c.title),
    description: str(c.description),
    completed: Boolean(c.completed),
    notes: str(c.notes),
    completedAt: str(c.completed_at),
  }
}

function mapAudit(a: Record<string, unknown>): DeploymentDataAudit {
  return {
    generatedAt: str(a.generated_at),
    databasePath: str(a.database_path),
    expectedRuntimeDatabasePath: str(a.expected_runtime_database_path),
    packagedDatabasePath: str(a.packaged_database_path),
    missingTables: ((a.missing_tables as unknown as string[] | undefined) ?? []).map((x) => String(x)),
    blockingDataIssues: ((a.blocking_data_issues as unknown as string[] | undefined) ?? []).map((x) => String(x)),
    warningDataIssues: ((a.warning_data_issues as unknown as string[] | undefined) ?? []).map((x) => String(x)),
    blocking: Boolean(a.blocking),
    taskItems: num(a.task_items),
    expenseEntries: num(a.expense_entries),
    payrollRuns: num(a.payroll_runs),
    legacyQuotedOfferShells: num(a.legacy_quoted_offer_shells),
    legacyRfqOfferShells: num(a.legacy_rfq_offer_shells),
    legacyFollowupTasks: num(a.legacy_followup_tasks),
    migratedLegacyTasks: num(a.migrated_legacy_tasks),
  }
}

function mapQueueOp(o: Record<string, unknown>): CollaborativePendingOperation {
  return {
    id: str(o.id),
    createdAt: goDate(o.created_at),
    updatedAt: goDate(o.updated_at),
    entityType: str(o.entity_type),
    entityId: str(o.entity_id),
    operation: str(o.operation),
    status: str(o.status),
    attempts: num(o.attempts),
    errorMessage: str(o.error_message),
  }
}

function mapLicense(l: Record<string, unknown>): LicenseKey {
  return {
    key: str(l.key),
    role: str(l.role),
    displayName: str(l.display_name),
    activated: Boolean(l.activated),
  }
}

async function realFetchSummary(): Promise<PilotReadinessSummary> {
  return mapSummary((await GetPilotReadinessSummary()) as unknown as Record<string, unknown>)
}

async function realFetchReadinessRows(issuesOnly: boolean): Promise<PilotReadinessRow[]> {
  const rows = await ListPilotReadinessRows(issuesOnly)
  return (rows ?? []).map((r) => mapReadinessRow(r as unknown as Record<string, unknown>))
}

async function realFetchRollout(): Promise<Phase7RolloutStatus> {
  return mapRollout((await GetPhase7RolloutStatus()) as unknown as Record<string, unknown>)
}

async function realFetchChecklist(): Promise<PilotChecklistItem[]> {
  const rows = await GetPilotDeploymentChecklist()
  return (rows ?? []).map((r) => mapChecklistItem(r as unknown as Record<string, unknown>))
}

async function realFetchAudit(): Promise<DeploymentDataAudit> {
  return mapAudit((await GetDeploymentDataAudit()) as unknown as Record<string, unknown>)
}

async function realFetchLicenseKeys(): Promise<LicenseKey[]> {
  const rows = await ListLicenseKeys()
  return (rows ?? []).map((r) => mapLicense(r as unknown as Record<string, unknown>))
}

async function realFetchQueue(status: string, limit: number): Promise<CollaborativePendingOperation[]> {
  const rows = await ListCollaborativePendingOperations(status, limit)
  return (rows ?? []).map((r) => mapQueueOp(r as unknown as Record<string, unknown>))
}

async function realUpdateChecklistItem(_id: string, _completed: boolean, _notes: string): Promise<PilotChecklistItem[]> {
  throw new Error('INTEG gap: UpdatePilotDeploymentChecklistItem — wires at K5')
}

async function realTriggerSync(): Promise<void> {
  throw new Error('INTEG gap: TriggerCollaborativeSyncNow — wires at K5')
}

async function realRetryBulk(_status: string, _limit: number): Promise<Phase7ActionResult> {
  throw new Error('INTEG gap: RetryCollaborativePendingOperations — HOT bulk resync-storm, wires at K5')
}

async function realRetrySingle(_id: string): Promise<void> {
  throw new Error('INTEG gap: RetryCollaborativePendingOperation — wires at K5')
}

async function realExportSupportBundle(): Promise<PilotSupportBundleResult> {
  throw new Error('INTEG gap: ExportPilotSupportBundle (App) — wires at K5')
}

async function realExportSignoff(): Promise<PilotExportResult> {
  throw new Error('INTEG gap: ExportPilotSignoffReport — wires at K5')
}

async function realReassignLicense(_employeeId: string, _licenseKey: string, _syncName: boolean): Promise<void> {
  throw new Error('INTEG gap: ReassignEmployeeLicenseAccess (SyncServiceBinding) — wires at K5')
}

async function realUpdateLicenseDisplayName(_key: string, _displayName: string): Promise<LicenseKey> {
  throw new Error('INTEG gap: UpdateLicenseDisplayName — wires at K5')
}

/* ---- public switched API (viewmodel imports THESE) ---- */
export const fetchPilotReadinessSummary = (): Promise<PilotReadinessSummary> => pick(realFetchSummary, mockFetchSummary)()
export const fetchPilotReadinessRows = (issuesOnly: boolean): Promise<PilotReadinessRow[]> =>
  pick(realFetchReadinessRows, mockFetchReadinessRows)(issuesOnly)
export const fetchPhase7RolloutStatus = (): Promise<Phase7RolloutStatus> => pick(realFetchRollout, mockFetchRollout)()
export const fetchPilotChecklist = (): Promise<PilotChecklistItem[]> => pick(realFetchChecklist, mockFetchChecklist)()
export const fetchDeploymentDataAudit = (): Promise<DeploymentDataAudit> => pick(realFetchAudit, mockFetchAudit)()
export const fetchLicenseKeys = (): Promise<LicenseKey[]> => pick(realFetchLicenseKeys, mockFetchLicenseKeys)()
export const fetchCollaborativeQueue = (status: string, limit: number): Promise<CollaborativePendingOperation[]> =>
  pick(realFetchQueue, mockFetchQueue)(status, limit)

export const updatePilotChecklistItem = (id: string, completed: boolean, notes: string): Promise<PilotChecklistItem[]> =>
  pick(realUpdateChecklistItem, mockUpdateChecklistItem)(id, completed, notes)
export const triggerCollaborativeSync = (): Promise<void> => pick(realTriggerSync, mockTriggerSync)()
export const retryCollaborativeOpsBulk = (status: string, limit: number): Promise<Phase7ActionResult> =>
  pick(realRetryBulk, mockRetryBulk)(status, limit)
export const retryCollaborativeOp = (id: string): Promise<void> => pick(realRetrySingle, mockRetrySingle)(id)
export const exportPilotSupportBundle = (): Promise<PilotSupportBundleResult> =>
  pick(realExportSupportBundle, mockExportSupportBundle)()
export const exportPilotSignoffReport = (): Promise<PilotExportResult> => pick(realExportSignoff, mockExportSignoff)()
export const reassignEmployeeLicense = (employeeId: string, licenseKey: string, syncName: boolean): Promise<void> =>
  pick(realReassignLicense, mockReassignLicense)(employeeId, licenseKey, syncName)
export const updateLicenseDisplayName = (key: string, displayName: string): Promise<LicenseKey> =>
  pick(realUpdateLicenseDisplayName, mockUpdateLicenseDisplayName)(key, displayName)
