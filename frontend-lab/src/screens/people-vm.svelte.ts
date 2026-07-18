/* PeopleHub viewmodel — L5's reactive half: tab/sub-tab switching, directory
 * search + filter, the selected employee's Profile/Work/Access/Compliance
 * forms, the Org tree grouping (cycle-guarded), and the Contributions
 * rank/distribution math. No rendering/layout — PeopleHub.svelte binds an
 * instance of this and composes primitives only (L1). Named `people-vm` (not
 * `people.svelte.ts`) so its stem never collides with `PeopleHub.svelte`
 * case-insensitively on Windows (same rule as payroll-vm/pricing-vm).
 *
 * PII hot-zone: gov-ID document numbers are field-encrypted server-side.
 * `canViewUnmasked` gates full-number display everywhere (list AND edit) —
 * same pattern Payroll's `canViewUnmasked` established for salary amounts.
 * Defaults true for byte-parity with the old screen's all-or-nothing
 * behavior; see PeopleHub.parity.md for the Edit-form unmask stop-and-ask. */

import type { Tone } from '$kernel/tones'
import type { RankedRow, WidgetSegment } from '$kernel/hub'
import {
  createAccessLink,
  createDocument,
  createEmployee,
  createLoginUser,
  deleteDocument,
  fetchAccessLinks,
  fetchContributions,
  fetchCurrentEmployeeContext,
  fetchDocuments,
  fetchEmployees,
  fetchLicenseKeys,
  fetchLoginRoles,
  fetchLoginUsers,
  fetchProjectAssignments,
  generateLicenseKey,
  issuableLicenseRoles,
  reassignLicense,
  reassignManager,
  requestEmployeeArchive,
  setEmploymentState,
  updateDocument,
  updateEmployee,
  type CurrentEmployeeContext,
  type EmployeeAccessLink,
  type EmployeeComplianceDocument,
  type EmployeeContributionSummary,
  type EmployeeCreateInput,
  type EmployeeProfile,
  type EmployeeProfileDraft,
  type LicenseKeySummary,
  type LoginUserSummary,
  type ProjectAssignment,
  type RoleSummary,
} from '../bridge/people'

export type PeopleTab = 'directory' | 'org' | 'contributions' | 'payroll'
export type DirectoryDetailTab = 'profile' | 'work' | 'access' | 'compliance'
/** '' = All (FilterChips' own built-in "All" chip — mirrors payroll-vm's
 * divisionFilter convention), 'active' / 'archive' are explicit chips. Plain
 * `string` (not a literal union) so it binds cleanly to FilterChips'
 * `selected` prop, same as payroll-vm's divisionFilter. */
export type DirectoryStatusFilter = string

/** Archive is the ONLY deactivation path — `employment_status` (Active/On
 * Leave/Probation/Contract) tracks work state, never "inactive". Mirrors the
 * old screen's `isArchivedEmployee` exactly, including the two-flag
 * inconsistency adversary (is_active:false, employment_status still
 * "active") — both flags are checked, neither is trusted alone. */
export function isArchivedEmployee(e: EmployeeProfile): boolean {
  return e.isActive === false || e.employmentStatus.toLowerCase() === 'archived'
}

export function employeeStatusLabel(e: EmployeeProfile): string {
  if (e.employmentStatus.toLowerCase() === 'archived') return 'Archived'
  if (e.isActive === false) return 'Inactive'
  if (!e.employmentStatus) return 'Active'
  const known: Record<string, string> = { active: 'Active', on_leave: 'On Leave', probation: 'Probation', contract: 'Contract' }
  return known[e.employmentStatus.toLowerCase()] ?? 'Unknown'
}

export const EMPLOYEE_STATUS_TONES: Record<string, Tone> = {
  Active: 'success',
  'On Leave': 'warning',
  Probation: 'warning',
  Contract: 'info',
  Archived: 'neutral',
  Inactive: 'danger',
  Unknown: 'neutral',
}

/** Guarded walk up the manager chain (Work tab's "Reporting chain" strip).
 * A visited-id Set plus a hop cap means the emp-20<->emp-21 manager CYCLE in
 * the adversarial mock terminates instead of looping forever — this is the
 * one place in the screen that actually traverses parent links (the Org
 * tab's grouping below is a flat, cycle-safe reduce and needs no guard). */
export function managerChain(employeeId: string, employees: EmployeeProfile[], maxHops = 25): EmployeeProfile[] {
  const byId = new Map(employees.map((e) => [e.id, e]))
  const chain: EmployeeProfile[] = []
  const visited = new Set<string>([employeeId])
  let currentId = byId.get(employeeId)?.managerEmployeeId
  while (currentId && !visited.has(currentId) && chain.length < maxHops) {
    visited.add(currentId)
    const manager = byId.get(currentId)
    if (!manager) break
    chain.push(manager)
    currentId = manager.managerEmployeeId
  }
  return chain
}

/** Flat grouping by manager name (parity with the old screen's
 * `orgGroups`/`orgGroupEntries`) — a single reduce pass, never a recursive
 * tree walk, so it is structurally immune to the manager-cycle adversary. */
export function groupByManager(employees: EmployeeProfile[]): [string, EmployeeProfile[]][] {
  const groups = new Map<string, EmployeeProfile[]>()
  for (const e of employees) {
    const label = e.managerName || 'Leadership / Unassigned'
    if (!groups.has(label)) groups.set(label, [])
    groups.get(label)!.push(e)
  }
  return [...groups.entries()].sort((a, b) => a[0].localeCompare(b[0]))
}

const CONTRIBUTION_STATUS_TONE: Tone = 'info'

/** Top-N by completion rate for the Contributions RankedBarList — unrecognized
 * / zero-task employees still render (0% rate, not dropped). */
export function contributionRanking(rows: EmployeeContributionSummary[]): RankedRow[] {
  const sorted = [...rows].sort((a, b) => b.completionRate - a.completionRate)
  return sorted.map((r, i) => ({
    rank: i + 1,
    label: r.employeeName || r.employeeCode || 'Unnamed employee',
    value: Math.round(r.completionRate),
    pct: Math.max(0, Math.min(100, r.completionRate)),
    sublabel: `${r.department || 'No department'} · ${r.activeTaskCount} active / ${r.overdueTaskCount} overdue`,
  }))
}

/** Task-mix distribution across the scoped employees — active vs completed
 * vs blocked vs overdue, summed. Unknown/zero rows just don't add volume,
 * never crash the widget. */
export function taskMixDistribution(rows: EmployeeContributionSummary[]): WidgetSegment[] {
  const totals = { active: 0, completed: 0, blocked: 0, overdue: 0 }
  for (const r of rows) {
    totals.active += r.activeTaskCount
    totals.completed += r.completedTaskCount
    totals.blocked += r.blockedTaskCount
    totals.overdue += r.overdueTaskCount
  }
  const tones: Record<keyof typeof totals, Tone> = { active: CONTRIBUTION_STATUS_TONE, completed: 'success', blocked: 'danger', overdue: 'warning' }
  const labels: Record<keyof typeof totals, string> = { active: 'Active', completed: 'Completed', blocked: 'Blocked', overdue: 'Overdue' }
  return (Object.keys(totals) as (keyof typeof totals)[])
    .filter((k) => totals[k] > 0)
    .map((k) => ({ key: k, label: labels[k], value: totals[k], tone: tones[k] }))
}

function blankCreateDraft(): EmployeeCreateInput {
  return { fullName: '', department: '', jobTitle: '', email: '', phone: '', startDate: '', managerEmployeeId: '' }
}

function blankProfileDraft(): EmployeeProfileDraft {
  return {
    id: '',
    employeeCode: '',
    fullName: '',
    preferredName: '',
    email: '',
    phone: '',
    department: '',
    jobTitle: '',
    employmentStatus: 'active',
    managerEmployeeId: '',
    startDate: '',
    emergencyContact: '',
    notes: '',
    isActive: true,
  }
}

function draftFromEmployee(e: EmployeeProfile): EmployeeProfileDraft {
  return {
    id: e.id,
    employeeCode: e.employeeCode,
    fullName: e.fullName,
    preferredName: e.preferredName,
    email: e.email,
    phone: e.phone,
    department: e.department,
    jobTitle: e.jobTitle,
    employmentStatus: e.employmentStatus || 'active',
    managerEmployeeId: e.managerEmployeeId,
    startDate: e.startDate,
    emergencyContact: e.emergencyContact,
    notes: e.notes,
    isActive: e.isActive !== false,
  }
}

export class PeopleHubViewModel {
  activeTab = $state<PeopleTab>('directory')
  detailTab = $state<DirectoryDetailTab>('profile')

  loading = $state(true)
  error = $state<string | null>(null)

  employees = $state<EmployeeProfile[]>([])
  accessLinks = $state<EmployeeAccessLink[]>([])
  licenseKeys = $state<LicenseKeySummary[]>([])
  contributions = $state<EmployeeContributionSummary[]>([])
  loginUsers = $state<LoginUserSummary[]>([])
  loginRoles = $state<RoleSummary[]>([])
  context = $state<CurrentEmployeeContext | null>(null)

  selectedEmployeeId = $state('')
  projectAssignments = $state<ProjectAssignment[]>([])
  complianceDocuments = $state<EmployeeComplianceDocument[]>([])

  directorySearch = $state('')
  directoryStatusFilter = $state<DirectoryStatusFilter>('active')

  /** Mock permission flags — client-side gating only, mirrors the real
   * server RBAC (payroll:view / users:*) it will eventually read; every
   * mutation is INTEG-gapped regardless of what these hide/show. */
  canViewPayroll = $state(true)
  canViewUnmasked = $state(true)

  // ---- Add Employee composer (TabShell header, shared across tabs) ----
  createDraft = $state<EmployeeCreateInput>(blankCreateDraft())
  creatingEmployee = $state(false)
  createError = $state<string | null>(null)

  // ---- Profile / Work form ----
  profileDraft = $state<EmployeeProfileDraft>(blankProfileDraft())
  savingProfile = $state(false)
  profileError = $state<string | null>(null)

  // ---- Archive (HOT-ZONE: cascades access + project revocation, routes to
  // Approvals) ----
  archiveConfirmOpen = $state(false)
  archiving = $state(false)
  archiveError = $state<string | null>(null)

  // ---- Access tab ----
  selectedLicenseKey = $state('')
  linkingAccess = $state(false)
  accessError = $state<string | null>(null)
  reassignLicenseKey = $state('')
  bindUserId = $state('')
  bindingUser = $state(false)
  showNewLoginForm = $state(false)
  newLoginUsername = $state('')
  newLoginPassword = $state('')
  newLoginRoleId = $state('')
  issueLicenseRole = $state('staff')
  issueLicenseNotes = $state('')
  issuingLicense = $state(false)

  // ---- Compliance documents ----
  editingDocumentId = $state('')
  docType = $state('cpr')
  docPermitSubtype = $state('')
  docNumber = $state('')
  docExpiresOn = $state('')
  docNotes = $state('')
  savingDocument = $state(false)
  documentError = $state<string | null>(null)

  // ---- derived ----
  issuableRoles = issuableLicenseRoles()

  /** Cosmetic only (client-side role-string matching) — never a security
   * boundary; every mutation this gates is INTEG-gapped server-side
   * regardless. See PeopleHub.parity.md's isAdmin stop-and-ask. */
  isAdmin = $derived(['admin', 'administrator', 'developer'].includes((this.context?.licenseRole || '').toLowerCase()))

  activeEmployeeCount = $derived(this.employees.filter((e) => !isArchivedEmployee(e)).length)
  archivedEmployeeCount = $derived(this.employees.filter(isArchivedEmployee).length)

  filteredEmployees = $derived(
    this.employees.filter((e) => {
      if (this.directoryStatusFilter === 'active' && isArchivedEmployee(e)) return false
      if (this.directoryStatusFilter === 'archive' && !isArchivedEmployee(e)) return false
      // '' (All) falls through to the search-only check below.
      const q = this.directorySearch.trim().toLowerCase()
      if (!q) return true
      const haystack = [e.fullName, e.employeeCode, e.department, e.jobTitle, e.managerName].filter(Boolean).join(' ').toLowerCase()
      return haystack.includes(q)
    }),
  )

  selectedEmployee = $derived(this.employees.find((e) => e.id === this.selectedEmployeeId) || null)

  selectedAccessLinks = $derived(this.accessLinks.filter((l) => l.employeeId === this.selectedEmployeeId))
  linkedLicenseKeys = $derived(new Set(this.accessLinks.map((l) => l.licenseKey)))
  availableLicenseKeys = $derived(this.licenseKeys.filter((k) => !this.linkedLicenseKeys.has(k.key)))
  reassignableLicenseKeys = $derived(
    this.licenseKeys.filter((k) => this.linkedLicenseKeys.has(k.key) && !this.selectedAccessLinks.some((l) => l.licenseKey === k.key)),
  )

  reportingChain = $derived(this.selectedEmployeeId ? managerChain(this.selectedEmployeeId, this.employees) : [])

  orgGroups = $derived(groupByManager(this.employees))

  contributionsRanked = $derived(contributionRanking(this.contributions))
  contributionsDistribution = $derived(taskMixDistribution(this.contributions))
  contributionsSummary = $derived.by(() => {
    const rows = this.contributions
    const tracked = rows.length
    const avgCompletion = tracked > 0 ? Math.round((rows.reduce((s, r) => s + r.completionRate, 0) / tracked) * 10) / 10 : 0
    const totalActive = rows.reduce((s, r) => s + r.activeTaskCount, 0)
    const totalOverdue = rows.reduce((s, r) => s + r.overdueTaskCount, 0)
    return { tracked, avgCompletion, totalActive, totalOverdue }
  })

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      const [employees, accessLinks, licenseKeys, contributions, loginUsers, loginRoles, context] = await Promise.all([
        fetchEmployees(false),
        fetchAccessLinks(),
        fetchLicenseKeys(),
        fetchContributions(),
        fetchLoginUsers().catch(() => []),
        fetchLoginRoles().catch(() => []),
        fetchCurrentEmployeeContext().catch(() => null),
      ])
      this.employees = employees
      this.accessLinks = accessLinks
      this.licenseKeys = licenseKeys
      this.contributions = contributions
      this.loginUsers = loginUsers
      this.loginRoles = loginRoles
      this.context = context
      if (!this.newLoginRoleId && loginRoles.length > 0) this.newLoginRoleId = loginRoles[0]!.id
      if (!this.selectedEmployeeId && employees.length > 0) this.selectedEmployeeId = employees[0]!.id
      else if (this.selectedEmployeeId && !employees.some((e) => e.id === this.selectedEmployeeId)) {
        this.selectedEmployeeId = employees[0]?.id || ''
      }
      if (!this.selectedLicenseKey && this.availableLicenseKeys.length > 0) this.selectedLicenseKey = this.availableLicenseKeys[0]!.key
      await this.loadSelectedEmployeeData()
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  setTab(tab: PeopleTab): void {
    this.activeTab = tab
  }

  setDetailTab(tab: DirectoryDetailTab): void {
    this.detailTab = tab
  }

  async loadSelectedEmployeeData(): Promise<void> {
    const selected = this.selectedEmployee
    this.profileDraft = selected ? draftFromEmployee(selected) : blankProfileDraft()
    this.resetDocumentForm()
    if (!selected?.id) {
      this.projectAssignments = []
      this.complianceDocuments = []
      return
    }
    try {
      const [assignments, documents] = await Promise.all([fetchProjectAssignments(selected.id), fetchDocuments(selected.id)])
      this.projectAssignments = assignments
      this.complianceDocuments = documents
    } catch (e) {
      this.projectAssignments = []
      this.complianceDocuments = []
      this.profileError = e instanceof Error ? e.message : String(e)
    }
  }

  async selectEmployee(employeeId: string): Promise<void> {
    this.selectedEmployeeId = employeeId
    this.detailTab = 'profile'
    this.profileError = null
    this.archiveError = null
    await this.loadSelectedEmployeeData()
  }

  // ---- Add Employee composer ----
  async createEmployeeFromDraft(): Promise<void> {
    if (!this.createDraft.fullName.trim()) {
      this.createError = 'Employee name is required.'
      return
    }
    this.creatingEmployee = true
    this.createError = null
    try {
      const created = await createEmployee({ ...this.createDraft, fullName: this.createDraft.fullName.trim() })
      this.createDraft = blankCreateDraft()
      await this.load()
      await this.selectEmployee(created.id)
      this.activeTab = 'directory'
      this.detailTab = 'profile'
    } catch (e) {
      this.createError = e instanceof Error ? e.message : String(e)
    } finally {
      this.creatingEmployee = false
    }
  }

  // ---- Profile / Work ----
  async saveProfile(): Promise<void> {
    if (!this.selectedEmployee?.id) {
      this.profileError = 'Choose an employee first.'
      return
    }
    if (!this.profileDraft.fullName.trim()) {
      this.profileError = 'Employee name is required.'
      return
    }
    this.savingProfile = true
    this.profileError = null
    try {
      await updateEmployee(this.profileDraft)
      if ((this.selectedEmployee.managerEmployeeId || '') !== this.profileDraft.managerEmployeeId) {
        await reassignManager(this.selectedEmployee.id, this.profileDraft.managerEmployeeId)
      }
      await this.load()
    } catch (e) {
      this.profileError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingProfile = false
    }
  }

  async reactivateEmployee(): Promise<void> {
    if (!this.selectedEmployee?.id) return
    this.savingProfile = true
    this.profileError = null
    try {
      await setEmploymentState(this.selectedEmployee.id, true, 'active')
      await this.load()
    } catch (e) {
      this.profileError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingProfile = false
    }
  }

  /** HOT-ZONE: cascades access + project revocation and routes to the
   * Approvals queue — gated behind a mandatory-reason ConfirmDialog, never a
   * bare click. */
  requestArchive(): void {
    if (!this.selectedEmployee || isArchivedEmployee(this.selectedEmployee)) return
    this.archiveConfirmOpen = true
  }

  cancelArchive(): void {
    this.archiveConfirmOpen = false
  }

  async confirmArchive(reason: string): Promise<void> {
    if (!this.selectedEmployee) return
    this.archiveConfirmOpen = false
    this.archiving = true
    this.archiveError = null
    try {
      await requestEmployeeArchive(this.selectedEmployee.id, reason)
      await this.load()
    } catch (e) {
      this.archiveError = e instanceof Error ? e.message : String(e)
    } finally {
      this.archiving = false
    }
  }

  // ---- Access ----
  async linkAccess(): Promise<void> {
    if (!this.selectedEmployeeId || !this.selectedLicenseKey) {
      this.accessError = 'Choose a license key to link.'
      return
    }
    this.linkingAccess = true
    this.accessError = null
    try {
      await createAccessLink({ employeeId: this.selectedEmployeeId, licenseKey: this.selectedLicenseKey, accessStatus: 'active' })
      await this.load()
      this.selectedLicenseKey = this.availableLicenseKeys[0]?.key || ''
    } catch (e) {
      this.accessError = e instanceof Error ? e.message : String(e)
    } finally {
      this.linkingAccess = false
    }
  }

  async reassignLicenseToSelected(licenseKey: string): Promise<void> {
    if (!this.selectedEmployeeId || !licenseKey) return
    this.linkingAccess = true
    this.accessError = null
    try {
      await reassignLicense(this.selectedEmployeeId, licenseKey)
      this.reassignLicenseKey = ''
      await this.load()
    } catch (e) {
      this.accessError = e instanceof Error ? e.message : String(e)
    } finally {
      this.linkingAccess = false
    }
  }

  async bindUser(): Promise<void> {
    if (!this.selectedEmployeeId) {
      this.accessError = 'Choose an employee first.'
      return
    }
    const primary = this.selectedAccessLinks.find((l) => l.isPrimary) || this.selectedAccessLinks[0]
    if (!primary) {
      this.accessError = 'Link a license before binding a login user.'
      return
    }
    if (!this.bindUserId) {
      this.accessError = 'Choose a user to bind.'
      return
    }
    this.bindingUser = true
    this.accessError = null
    try {
      await createAccessLink({ employeeId: this.selectedEmployeeId, licenseKey: primary.licenseKey, userId: this.bindUserId, accessStatus: primary.accessStatus || 'active' })
      await this.load()
      this.bindUserId = ''
    } catch (e) {
      this.accessError = e instanceof Error ? e.message : String(e)
    } finally {
      this.bindingUser = false
    }
  }

  /** Mints a live login credential (temp password) — never pre-populated
   * with a realistic-looking value; the operator must type one. */
  async createAndBindLoginUser(): Promise<void> {
    if (!this.selectedEmployee?.id) {
      this.accessError = 'Choose an employee first.'
      return
    }
    if (!this.newLoginUsername.trim() || !this.newLoginPassword.trim() || !this.newLoginRoleId) {
      this.accessError = 'Username, password, and role are required.'
      return
    }
    this.bindingUser = true
    this.accessError = null
    try {
      const user = await createLoginUser({
        username: this.newLoginUsername.trim(),
        email: this.selectedEmployee.email || '',
        password: this.newLoginPassword,
        fullName: this.selectedEmployee.fullName,
        department: this.selectedEmployee.department || '',
        jobTitle: this.selectedEmployee.jobTitle || '',
        roleId: this.newLoginRoleId,
      })
      this.newLoginUsername = ''
      this.newLoginPassword = ''
      this.showNewLoginForm = false
      const primary = this.selectedAccessLinks.find((l) => l.isPrimary) || this.selectedAccessLinks[0]
      if (primary) {
        await createAccessLink({ employeeId: this.selectedEmployee.id, licenseKey: primary.licenseKey, userId: user.id, accessStatus: primary.accessStatus || 'active' })
      }
      await this.load()
    } catch (e) {
      this.accessError = e instanceof Error ? e.message : String(e)
    } finally {
      this.bindingUser = false
    }
  }

  /** Mints a live license credential — never mock a realistic-looking key
   * client-side; the bridge/server owns key generation. */
  async issueLicense(): Promise<void> {
    this.issuingLicense = true
    this.accessError = null
    try {
      const createdBy = this.context?.employeeName || 'admin'
      const key = await generateLicenseKey(this.issueLicenseRole, this.issueLicenseNotes.trim(), createdBy)
      this.issueLicenseNotes = ''
      await this.load()
      this.selectedLicenseKey = key
    } catch (e) {
      this.accessError = e instanceof Error ? e.message : String(e)
    } finally {
      this.issuingLicense = false
    }
  }

  // ---- Compliance documents ----
  resetDocumentForm(): void {
    this.editingDocumentId = ''
    this.docType = 'cpr'
    this.docPermitSubtype = ''
    this.docNumber = ''
    this.docExpiresOn = ''
    this.docNotes = ''
    this.documentError = null
  }

  editDocument(doc: EmployeeComplianceDocument): void {
    this.editingDocumentId = doc.id
    this.docType = doc.docType
    this.docPermitSubtype = doc.permitSubtype || ''
    // STOP-AND-ASK (see PeopleHub.parity.md): this auto-populates the
    // DECRYPTED doc number into the edit form exactly like the old screen
    // did — should Edit instead require a fresh unmask action? Preserved,
    // not fixed, and gated by canViewUnmasked same as the list.
    this.docNumber = this.canViewUnmasked ? doc.docNumber || '' : ''
    this.docExpiresOn = doc.expiresOn ? doc.expiresOn.slice(0, 10) : ''
    this.docNotes = doc.notes || ''
    this.documentError = null
  }

  async saveDocument(): Promise<void> {
    if (!this.selectedEmployeeId) return
    if (!this.docNumber.trim()) {
      this.documentError = 'Document number is required.'
      return
    }
    this.savingDocument = true
    this.documentError = null
    try {
      const draft = {
        docType: this.docType,
        permitSubtype: this.docType === 'permit' ? this.docPermitSubtype.trim() : '',
        docNumber: this.docNumber.trim(),
        expiresOn: this.docExpiresOn || null,
        notes: this.docNotes.trim(),
      }
      if (this.editingDocumentId) {
        await updateDocument(this.editingDocumentId, draft)
      } else {
        await createDocument({ ...draft, employeeId: this.selectedEmployeeId })
      }
      this.resetDocumentForm()
      this.complianceDocuments = await fetchDocuments(this.selectedEmployeeId)
    } catch (e) {
      this.documentError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingDocument = false
    }
  }

  async removeDocument(documentId: string): Promise<void> {
    try {
      await deleteDocument(documentId)
      if (this.editingDocumentId === documentId) this.resetDocumentForm()
      if (this.selectedEmployeeId) this.complianceDocuments = await fetchDocuments(this.selectedEmployeeId)
    } catch (e) {
      this.documentError = e instanceof Error ? e.message : String(e)
    }
  }
}

/** Days until expiry (negative = already expired); null for no expiry on
 * file or an unparseable date — the Compliance list must render both
 * without crashing (adversarial mock carries both). */
export function daysUntilExpiry(value: string | null): number | null {
  if (!value) return null
  const target = new Date(value)
  if (Number.isNaN(target.getTime())) return null
  return Math.ceil((target.getTime() - Date.now()) / (1000 * 60 * 60 * 24))
}

export function expiryTone(days: number | null): Tone {
  if (days === null) return 'neutral'
  if (days < 0) return 'danger'
  if (days <= 60) return 'warning'
  return 'success'
}

export const DOC_TYPE_LABELS: Record<string, string> = {
  cpr: 'CPR',
  passport: 'Passport',
  visa: 'Visa',
  permit: 'Permit',
}
