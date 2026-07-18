/* Deployment Hub viewmodel — L5's reactive half: three tabs (Audit /
 * Checklist / Support) over one pilot-rollout workspace. Named `deployment-vm`
 * (not `deployment.svelte.ts`) so its stem never differs from
 * `DeploymentHub.svelte` by case only — collides under TypeScript's
 * case-insensitive resolution on Windows (same convention as pricing-vm).
 *
 * The old screen's Activity tab (weekly per-employee user-activity monitor)
 * is RETIRED entirely — owner-ratified, surveillance-adjacent, out of scope
 * for the OSS kernel tranche. No `canViewActivityMonitoring` flag, no
 * activity state, no activity bindings anywhere in this VM. */

import type { Tone } from '$kernel/tones'
import type { WidgetSegment } from '$kernel/hub'
import {
  fetchCollaborativeQueue,
  fetchDeploymentDataAudit,
  fetchLicenseKeys,
  fetchPhase7RolloutStatus,
  fetchPilotChecklist,
  fetchPilotReadinessRows,
  fetchPilotReadinessSummary,
  exportPilotSignoffReport,
  exportPilotSupportBundle,
  reassignEmployeeLicense,
  retryCollaborativeOp,
  retryCollaborativeOpsBulk,
  triggerCollaborativeSync,
  updateLicenseDisplayName,
  updatePilotChecklistItem,
  type CollaborativePendingOperation,
  type DeploymentDataAudit,
  type LicenseKey,
  type Phase7RolloutStatus,
  type PilotChecklistItem,
  type PilotReadinessRow,
  type PilotReadinessSummary,
} from '../bridge/deployment'

export type DeploymentTab = 'audit' | 'checklist' | 'support'

const QUEUE_STATUS_TONE: Record<string, Tone> = {
  pending: 'warning',
  failed: 'danger',
  dead_letter: 'danger',
  synced: 'success',
}

export function queueStatusTone(status: string): Tone {
  return QUEUE_STATUS_TONE[status] ?? 'neutral'
}

export const AUDIT_FILTER_OPTIONS: { value: string; label: string }[] = [{ value: 'issues', label: 'Issues Only' }]

export const QUEUE_FILTER_OPTIONS: { value: string; label: string }[] = [
  { value: 'active', label: 'Active Issues' },
  { value: 'pending', label: 'Pending' },
  { value: 'failed', label: 'Failed' },
  { value: 'dead_letter', label: 'Dead Letter' },
  { value: 'synced', label: 'Recently Synced' },
]

export class DeploymentHubViewModel {
  loading = $state(true)
  error = $state<string | null>(null)
  activeTab = $state<DeploymentTab>('audit')

  summary = $state<PilotReadinessSummary | null>(null)
  rollout = $state<Phase7RolloutStatus | null>(null)
  audit = $state<DeploymentDataAudit | null>(null)
  readinessRows = $state<PilotReadinessRow[]>([])
  checklist = $state<PilotChecklistItem[]>([])
  licenseKeys = $state<LicenseKey[]>([])
  queueOps = $state<CollaborativePendingOperation[]>([])

  // ---- Audit tab ----
  search = $state('')
  /** FilterChips-driven ('' = all, 'issues' = issues only) — defaults to
   * issues-only, matching the old screen's default. */
  auditFilter = $state('issues')
  selectedEmployeeId = $state('')
  selectedLicenseKey = $state('')
  syncLicenseName = $state(true)
  licenseDisplayNameDraft = $state('')
  reassignBusy = $state(false)
  reassignError = $state<string | null>(null)
  licenseNameBusy = $state(false)
  licenseNameError = $state<string | null>(null)

  // ---- Checklist tab ----
  checklistNotesDraft = $state<Record<string, string>>({})
  checklistSavingId = $state('')
  checklistError = $state<string | null>(null)

  // ---- Support tab ----
  queueFilter = $state('active')
  queueLoading = $state(false)
  selectedOpId = $state<string | null>(null)
  syncBusy = $state(false)
  syncError = $state<string | null>(null)
  syncMessage = $state<string | null>(null)
  retryBusy = $state(false)
  retryError = $state<string | null>(null)
  bulkRetryStatus = $state<string | null>(null) // non-null -> ConfirmDialog open
  bulkRetryBusy = $state(false)
  bulkRetryError = $state<string | null>(null)
  bulkRetryMessage = $state<string | null>(null)
  exportingBundle = $state(false)
  exportBundleError = $state<string | null>(null)
  exportBundleMessage = $state<string | null>(null)
  exportingSignoff = $state(false)
  exportSignoffError = $state<string | null>(null)
  exportSignoffMessage = $state<string | null>(null)

  issuesOnly = $derived(this.auditFilter === 'issues')

  filteredRows = $derived.by(() => {
    const term = this.search.trim().toLowerCase()
    return this.readinessRows.filter((row) => {
      if (this.issuesOnly && row.issues.length === 0) return false
      if (!term) return true
      const haystack = [row.employeeName, row.employeeCode, row.department, row.jobTitle, row.licenseKey, row.deviceName, row.userName]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
      return haystack.includes(term)
    })
  })

  selectedRow = $derived(
    this.readinessRows.find((r) => r.employeeId === this.selectedEmployeeId) ?? null,
  )

  selectedLicenseRecord = $derived(this.licenseKeys.find((l) => l.key === this.selectedLicenseKey) ?? null)

  completedChecklistCount = $derived(this.checklist.filter((c) => c.completed).length)

  queueDistribution = $derived.by((): WidgetSegment[] => {
    const counts = new Map<string, number>()
    for (const op of this.queueOps) counts.set(op.status, (counts.get(op.status) ?? 0) + 1)
    return [...counts.entries()].map(([status, value]) => ({
      key: status,
      label: status.replace(/_/g, ' '),
      value,
      tone: queueStatusTone(status),
    }))
  })

  selectedOp = $derived(this.queueOps.find((o) => o.id === this.selectedOpId) ?? null)

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      const [summary, rows, rollout, checklist, licenseKeys, audit] = await Promise.all([
        fetchPilotReadinessSummary(),
        fetchPilotReadinessRows(false),
        fetchPhase7RolloutStatus(),
        fetchPilotChecklist(),
        fetchLicenseKeys(),
        fetchDeploymentDataAudit(),
      ])
      this.summary = summary
      this.readinessRows = rows
      this.rollout = rollout
      this.checklist = checklist
      this.syncChecklistDrafts()
      this.licenseKeys = licenseKeys
      this.audit = audit

      if (!this.selectedEmployeeId || !rows.some((r) => r.employeeId === this.selectedEmployeeId)) {
        const first = rows.find((r) => r.issues.length > 0) ?? rows[0]
        if (first) this.selectEmployee(first)
      }

      await this.loadQueue()
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  async refresh(): Promise<void> {
    await this.load()
  }

  setTab(tab: DeploymentTab): void {
    this.activeTab = tab
  }

  // ---- Audit ----

  selectEmployee(row: PilotReadinessRow): void {
    this.selectedEmployeeId = row.employeeId
    this.reassignError = null
    this.licenseNameError = null
    this.selectedLicenseKey = row.licenseKey || this.licenseKeys[0]?.key || ''
    this.licenseDisplayNameDraft = this.licenseKeys.find((l) => l.key === this.selectedLicenseKey)?.displayName ?? ''
  }

  selectLicenseKey(key: string): void {
    this.selectedLicenseKey = key
    this.licenseDisplayNameDraft = this.licenseKeys.find((l) => l.key === key)?.displayName ?? ''
  }

  async reassignLicense(): Promise<void> {
    if (!this.selectedRow || !this.selectedLicenseKey) return
    this.reassignBusy = true
    this.reassignError = null
    try {
      await reassignEmployeeLicense(this.selectedRow.employeeId, this.selectedLicenseKey, this.syncLicenseName)
      await this.load()
    } catch (e) {
      this.reassignError = e instanceof Error ? e.message : String(e)
    } finally {
      this.reassignBusy = false
    }
  }

  async saveLicenseDisplayName(): Promise<void> {
    if (!this.selectedLicenseKey || !this.licenseDisplayNameDraft.trim()) return
    this.licenseNameBusy = true
    this.licenseNameError = null
    try {
      await updateLicenseDisplayName(this.selectedLicenseKey, this.licenseDisplayNameDraft.trim())
      await this.load()
    } catch (e) {
      this.licenseNameError = e instanceof Error ? e.message : String(e)
    } finally {
      this.licenseNameBusy = false
    }
  }

  // ---- Checklist ----

  private syncChecklistDrafts(): void {
    const drafts: Record<string, string> = {}
    for (const item of this.checklist) drafts[item.id] = item.notes
    this.checklistNotesDraft = drafts
  }

  setChecklistNote(id: string, value: string): void {
    this.checklistNotesDraft = { ...this.checklistNotesDraft, [id]: value }
  }

  async toggleChecklistItem(item: PilotChecklistItem, completed: boolean): Promise<void> {
    this.checklistSavingId = item.id
    this.checklistError = null
    try {
      this.checklist = await updatePilotChecklistItem(item.id, completed, this.checklistNotesDraft[item.id] ?? item.notes)
      this.syncChecklistDrafts()
    } catch (e) {
      this.checklistError = e instanceof Error ? e.message : String(e)
    } finally {
      this.checklistSavingId = ''
    }
  }

  async saveChecklistNotes(item: PilotChecklistItem): Promise<void> {
    this.checklistSavingId = item.id
    this.checklistError = null
    try {
      this.checklist = await updatePilotChecklistItem(item.id, item.completed, this.checklistNotesDraft[item.id] ?? '')
      this.syncChecklistDrafts()
    } catch (e) {
      this.checklistError = e instanceof Error ? e.message : String(e)
    } finally {
      this.checklistSavingId = ''
    }
  }

  // ---- Support ----

  async loadQueue(): Promise<void> {
    this.queueLoading = true
    try {
      this.queueOps = await fetchCollaborativeQueue(this.queueFilter, 100)
    } catch {
      this.queueOps = []
    } finally {
      this.queueLoading = false
    }
  }

  setQueueFilter(status: string): void {
    this.queueFilter = status
    void this.loadQueue()
  }

  selectOp(op: CollaborativePendingOperation): void {
    this.selectedOpId = op.id
    this.retryError = null
  }

  async triggerSync(): Promise<void> {
    this.syncBusy = true
    this.syncError = null
    this.syncMessage = null
    try {
      await triggerCollaborativeSync()
      this.syncMessage = 'Collaborative sync completed.'
      await this.load()
    } catch (e) {
      this.syncError = e instanceof Error ? e.message : String(e)
    } finally {
      this.syncBusy = false
    }
  }

  /** HOT bulk mutation (retry-storm) — gated behind an explicit ConfirmDialog;
   * the old screen fired this with no confirm at all (parity fix). */
  requestBulkRetry(status: string): void {
    this.bulkRetryStatus = status
    this.bulkRetryError = null
  }

  cancelBulkRetry(): void {
    this.bulkRetryStatus = null
  }

  async confirmBulkRetry(): Promise<void> {
    const status = this.bulkRetryStatus
    if (!status) return
    this.bulkRetryStatus = null
    this.bulkRetryBusy = true
    this.bulkRetryError = null
    this.bulkRetryMessage = null
    try {
      const result = await retryCollaborativeOpsBulk(status, 100)
      this.bulkRetryMessage = result.message
      await this.loadQueue()
      this.rollout = await fetchPhase7RolloutStatus()
    } catch (e) {
      this.bulkRetryError = e instanceof Error ? e.message : String(e)
    } finally {
      this.bulkRetryBusy = false
    }
  }

  async retrySelectedOp(): Promise<void> {
    if (!this.selectedOp) return
    this.retryBusy = true
    this.retryError = null
    try {
      await retryCollaborativeOp(this.selectedOp.id)
      await this.loadQueue()
    } catch (e) {
      this.retryError = e instanceof Error ? e.message : String(e)
    } finally {
      this.retryBusy = false
    }
  }

  async exportSupportBundle(): Promise<void> {
    this.exportingBundle = true
    this.exportBundleError = null
    this.exportBundleMessage = null
    try {
      const result = await exportPilotSupportBundle()
      this.exportBundleMessage = `Support bundle exported to ${result.path} (${result.rows} rows).`
    } catch (e) {
      this.exportBundleError = e instanceof Error ? e.message : String(e)
    } finally {
      this.exportingBundle = false
    }
  }

  async exportSignoffReport(): Promise<void> {
    this.exportingSignoff = true
    this.exportSignoffError = null
    this.exportSignoffMessage = null
    try {
      const result = await exportPilotSignoffReport()
      this.exportSignoffMessage = `Sign-off report exported to ${result.path}.`
    } catch (e) {
      this.exportSignoffError = e instanceof Error ? e.message : String(e)
    } finally {
      this.exportingSignoff = false
    }
  }
}

export function formatIssue(issue: string): string {
  if (!issue) return 'Unknown issue'
  return issue.replace(/_/g, ' ').replace(/\b\w/g, (m) => m.toUpperCase())
}
