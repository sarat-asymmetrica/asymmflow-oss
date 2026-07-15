/* People bridge module — self-contained: types + mock + real + switch.
 * HR console (K4 hub tranche). THE HIGHEST-PII screen in the kernel: gov-ID
 * document numbers (CPR/passport/visa/permit) are field-encrypted server-side
 * (same FieldCrypto convention as `collaboration.ts`'s EmployeeComplianceDocument),
 * and license issuance / login-user creation mint live credentials. FETCH
 * bindings are real, single-call adapters (verified against
 * `frontend/wailsjs/go/main/App.d.ts` and `InfraService.d.ts`); every mutation
 * is an honest INTEG-gap throw naming the exact real binding — a PII + identity
 * hot-zone doesn't get a naive pass-through (see PeopleHub.parity.md). Mock
 * mutations DO work (mirrors payroll.ts / book-bank-recon.ts: the mock branch
 * is only reached when `usingWails()` is false, i.e. the lab/dev browser —
 * never inside the real desktop app). Synthetic-only data (SYNTHETIC_IDENTITY.md)
 * — invented Gulf/Latin-mixed names, never real people. */
import { pick } from './runtime'
import { goDate, num, str } from './map'
import {
  GetCurrentEmployeeContext,
  ListEmployeeAccessLinks,
  ListEmployeeContributionSummaries,
  ListEmployeeDocuments,
  ListEmployeeProfiles,
  ListEmployeeProjectAssignments,
  ListLicenseKeys,
} from '$wails/go/main/App'
import { ListRoles, ListUsers } from '$wails/go/main/InfraService'
/* Only the FETCH bindings above are actually invoked below — mutations are
 * INTEG-gap throws that NAME the real binding without importing it (same
 * convention as bridge/payroll.ts / bridge/expenses.ts: importing a binding
 * this file never calls would just be an unused-import lint trap). Real
 * binding names referenced in the throws: CreateEmployeeProfile,
 * UpdateEmployeeProfile, SetEmployeeEmploymentState, RequestEmployeeArchive,
 * ReviewEmployeeArchiveRequest, ReassignEmployeeManager,
 * CreateEmployeeAccessLink, ReassignEmployeeLicenseAccess,
 * GenerateLicenseKey, CreateUser, CreateEmployeeDocument,
 * UpdateEmployeeDocument, DeleteEmployeeDocument. */

/* ---- types (camelCase; mirror frontend/src/lib/api/collaboration.ts's
 * snake_case shapes 1:1) ---- */

export interface EmployeeProfile {
  id: string
  employeeCode: string
  fullName: string
  preferredName: string
  email: string
  phone: string
  department: string
  jobTitle: string
  employmentStatus: string
  managerEmployeeId: string
  managerName: string
  startDate: string
  endDate: string
  emergencyContact: string
  notes: string
  isActive: boolean
  archivedAt: string
  archivedBy: string
  archiveReason: string
  archiveRequestId: string
}

export interface EmployeeProfileDraft {
  id: string
  employeeCode: string
  fullName: string
  preferredName: string
  email: string
  phone: string
  department: string
  jobTitle: string
  employmentStatus: string
  managerEmployeeId: string
  startDate: string
  emergencyContact: string
  notes: string
  isActive: boolean
}

export interface EmployeeCreateInput {
  fullName: string
  department: string
  jobTitle: string
  email: string
  phone: string
  startDate: string
  managerEmployeeId: string
}

export interface EmployeeArchiveApproval {
  id: string
  employeeId: string
  employeeName: string
  requestedBy: string
  reason: string
  status: string
}

export interface EmployeeAccessLink {
  id: string
  employeeId: string
  employeeName: string
  licenseKey: string
  userId: string
  deviceId: string
  deviceName: string
  accessStatus: string
  isPrimary: boolean
}

export interface LicenseKeySummary {
  id: string
  key: string
  role: string
  deviceId: string
  assignedTo: string
  displayName: string
  status: string
}

export interface EmployeeContributionSummary {
  employeeId: string
  employeeCode: string
  employeeName: string
  department: string
  jobTitle: string
  managerEmployeeId: string
  managerName: string
  employmentStatus: string
  isActive: boolean
  activeProjectCount: number
  activeTaskCount: number
  completedTaskCount: number
  blockedTaskCount: number
  overdueTaskCount: number
  completionRate: number
  primaryLicenseKey: string
  primaryDeviceName: string
}

export interface ProjectAssignment {
  id: string
  projectId: string
  employeeId: string
  employeeName: string
  projectName: string
  role: string
  allocationPercent: number
  isActive: boolean
  joinedAt: string
  leftAt: string
}

/** PII hot-zone: `docNumber` is the DECRYPTED value (server never returns raw
 * ciphertext) — `docNumberMasked` is the convenience field every LIST surface
 * should render by default. See PeopleHub.parity.md's unmask stop-and-ask. */
export interface EmployeeComplianceDocument {
  id: string
  employeeId: string
  docType: string
  permitSubtype: string
  docNumber: string
  docNumberMasked: string
  expiresOn: string | null
  notes: string
  notifiedAt: string | null
  createdAt: string
  updatedAt: string
}

export interface EmployeeDocumentDraft {
  docType: string
  permitSubtype: string
  docNumber: string
  expiresOn: string | null
  notes: string
}

export interface LoginUserSummary {
  id: string
  username: string
  email: string
  fullName: string
  department: string
  jobTitle: string
  roleId: string
  roleName: string
  isActive: boolean
}

export interface RoleSummary {
  id: string
  name: string
  displayName: string
}

export interface CurrentEmployeeContext {
  employeeId: string
  employeeName: string
  licenseKey: string
  licenseRole: string
  deviceId: string
  userId: string
  resolvedBy: string
  permissions: string[]
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

const MONSTER_NAME =
  'ABDULRAHMANMOHAMMEDALSHAMSISENIORHUMANRESOURCESANDPAYROLLCOMPLIANCEOFFICERFORMERLYREGIONALOPERATIONSDIRECTORATEOFADMINISTRATIVEAFFAIRSDEPARTMENT'.padEnd(
    210,
    'Z',
  )

// 200-char + RTL + Latin-mixed adversarial name pool (SYNTHETIC_IDENTITY.md —
// invented names only, never real people).
const NAMES = [
  'Ahmed Al-Khalifa',
  'Fatima Mohammed Al-Sayed',
  'Yusuf Kanoo',
  'Al', // 2-char
  '', // empty — exercises directory/detail blank-name fallbacks
  'محمد عبدالله الأنصاري الكوري', // RTL
  MONSTER_NAME, // unbroken 200+ char token
  'Layla van der Berg-Hassan', // Latin-mixed, hyphen
  "Sarah O'Connor-Al Mannai", // apostrophe + hyphen
  'José García Al-Rashid', // diacritics + Latin-mixed
  'Karim Nasser',
  'X',
  'Noor Al-Zayani',
  'Hessa Al-Dosari',
  'Omar Fakhro',
  'Bashir Al-Ansari',
  'Priya Al-Balushi Nair', // Latin-mixed
  'Chen Wei Al-Rumaihi', // Latin-mixed
]

const DEPARTMENTS = ['Sales', 'Finance', 'Warehouse', 'Procurement', 'Operations', 'IT', 'Human Resources', '']
const JOB_TITLES = [
  'Field Engineer',
  'Sales Executive',
  'Finance Officer',
  'HR Coordinator',
  'Warehouse Supervisor',
  'Procurement Lead',
  'Operations Coordinator',
  'IT Administrator',
  '',
]
const EMPLOYMENT_STATUSES = ['active', 'on_leave', 'probation', 'contract']
const ROLE_KEYS = ['admin', 'manager', 'sales', 'operations', 'staff']
const DOC_TYPES = ['cpr', 'passport', 'visa', 'permit']

interface Dataset {
  employees: EmployeeProfile[]
  accessLinks: EmployeeAccessLink[]
  licenseKeys: LicenseKeySummary[]
  contributions: EmployeeContributionSummary[]
  projectAssignments: ProjectAssignment[]
  complianceDocuments: EmployeeComplianceDocument[]
  loginUsers: LoginUserSummary[]
  loginRoles: RoleSummary[]
}

let cache: Dataset | null = null
let archiveApprovalSeq = 0
let documentSeq = 0

function generate(): Dataset {
  const rand = lcg(20260714 ^ 0x9e0 /* people seed tweak, matches sibling bridges' xor convention */)

  // ---- employees: 36 total. Woven-in adversarial cases (indices are
  // 1-based `i`, matching id `emp-${i}`):
  //  - i=2:  orphan leaf, no manager, not a "leadership root"
  //  - i=20/21: MANAGER CYCLE (A -> B -> A) — org-tree grouping and
  //    managerChain() below must both guard against this, never loop.
  //  - i=15: two-flag inconsistency — is_active:false but
  //    employment_status still "active"
  //  - i=27: UNKNOWN employment_status (tone degrades neutral)
  //  - i=30: archived (is_active:false, employment_status:"archived")
  const EMP_COUNT = 36
  const employees: EmployeeProfile[] = []
  for (let i = 1; i <= EMP_COUNT; i++) {
    const id = `emp-${i}`
    const fullName = i % 31 === 0 ? NAMES[6]! /* monster */ : i % 23 === 0 ? NAMES[5]! /* RTL */ : NAMES[(i - 1) % NAMES.length]!
    const department = DEPARTMENTS[(i - 1) % DEPARTMENTS.length]!
    const jobTitle = JOB_TITLES[(i - 1) % JOB_TITLES.length]!

    let managerEmployeeId = i <= 1 ? '' : `emp-${1 + ((i - 2) % 6)}`
    if (i === 2) managerEmployeeId = '' // orphan leaf
    if (i === 20) managerEmployeeId = 'emp-21' // cycle
    if (i === 21) managerEmployeeId = 'emp-20' // cycle
    if (managerEmployeeId === id) managerEmployeeId = '' // never self-manage

    let isActive = i % 9 !== 0
    let employmentStatus = EMPLOYMENT_STATUSES[i % EMPLOYMENT_STATUSES.length]!
    let archivedAt = ''
    let archivedBy = ''
    let archiveReason = ''
    let archiveRequestId = ''

    if (i === 15) {
      isActive = false
      employmentStatus = 'active' // two-flag inconsistency
    }
    if (i === 27) {
      employmentStatus = 'UNKNOWN_STATUS'
    }
    if (i === 30) {
      isActive = false
      employmentStatus = 'archived'
      archivedAt = '2026-03-01'
      archivedBy = 'F. Al Khalifa'
      archiveReason = 'Contract concluded, not renewed.'
      archiveRequestId = 'arch-req-1'
    }

    const yearsAgo = 1 + (i % 7)
    const startDate = `${2026 - yearsAgo}-${pad(1 + (i % 12), 2)}-${pad(1 + (i % 27), 2)}`

    employees.push({
      id,
      employeeCode: `EMP-${pad(i, 4)}`,
      fullName,
      preferredName: i % 5 === 0 ? '' : fullName.split(' ')[0] || '',
      email: fullName ? `${id}@example.bh` : '',
      phone: i % 13 === 0 ? '' : `+973 3${pad(1000000 + i, 7)}`,
      department,
      jobTitle,
      employmentStatus,
      managerEmployeeId,
      managerName: '', // resolved in a second pass below
      startDate,
      endDate: '',
      emergencyContact: i % 7 === 0 ? '' : `+973 3${pad(2000000 + i, 7)}`,
      notes: i % 4 === 0 ? '' : 'Synthetic employee record.',
      isActive,
      archivedAt,
      archivedBy,
      archiveReason,
      archiveRequestId,
    })
  }
  // Second pass: resolve managerName now that every employee exists (handles
  // the cycle at 20/21 trivially — each just reads the other's CURRENT name,
  // no traversal involved here).
  const byId = new Map(employees.map((e) => [e.id, e]))
  for (const e of employees) {
    if (e.managerEmployeeId) e.managerName = byId.get(e.managerEmployeeId)?.fullName || ''
  }

  // ---- license keys: 22 total. Adversarial: empty display_name vs a very
  // long one; some unassigned (device_id empty).
  const LICENSE_COUNT = 22
  const licenseKeys: LicenseKeySummary[] = []
  for (let k = 1; k <= LICENSE_COUNT; k++) {
    const role = ROLE_KEYS[(k - 1) % ROLE_KEYS.length]!
    let displayName = `${role[0]!.toUpperCase()}${role.slice(1)} seat ${k}`
    if (k === 3) displayName = '' // empty display name adversary
    if (k === 9) displayName = 'Regional Operations And Field Services Coordination Department Shared License Seat Reserved For The Deputy Director Of Administrative Affairs'.padEnd(220, ' seat') // very long
    licenseKeys.push({
      id: `lic-${k}`,
      key: `LIC-${pad(k, 4)}`,
      role,
      deviceId: k % 4 === 0 ? '' : `device-${pad(k, 3)}`,
      assignedTo: '',
      displayName,
      status: k % 4 === 0 ? 'Available' : 'Activated',
    })
  }

  // ---- access links. Adversarial: emp-5 has ZERO links; emp-10 has 5+
  // links (bulk-license monster). Others get 0-2 links, some bound to a
  // login user, most not.
  const accessLinks: EmployeeAccessLink[] = []
  let licenseCursor = 0
  const nextLicense = () => licenseKeys[licenseCursor++ % licenseKeys.length]!
  for (let i = 1; i <= EMP_COUNT; i++) {
    if (i === 5) continue // zero-license adversary
    const emp = employees[i - 1]!
    const linkCount = i === 10 ? 5 : rand() < 0.6 ? 1 + Math.floor(rand() * 2) : 0
    for (let j = 0; j < linkCount; j++) {
      const lic = nextLicense()
      accessLinks.push({
        id: `link-${i}-${j}`,
        employeeId: emp.id,
        employeeName: emp.fullName,
        licenseKey: lic.key,
        userId: '', // bound in the loginUsers pass below for a subset
        deviceId: lic.deviceId,
        deviceName: lic.deviceId ? `Device ${lic.deviceId}` : '',
        accessStatus: 'active',
        isPrimary: j === 0,
      })
    }
  }

  // ---- login users + roles (small pool; Access tab bind/create surface).
  const loginRoles: RoleSummary[] = ROLE_KEYS.map((key, idx) => ({
    id: `role-${idx + 1}`,
    name: key,
    displayName: key.charAt(0).toUpperCase() + key.slice(1),
  }))
  const LOGIN_USER_COUNT = 14
  const loginUsers: LoginUserSummary[] = []
  for (let u = 1; u <= LOGIN_USER_COUNT; u++) {
    const role = loginRoles[(u - 1) % loginRoles.length]!
    loginUsers.push({
      id: `login-${u}`,
      username: `user.${pad(u, 3)}`,
      email: `user.${pad(u, 3)}@example.bh`,
      fullName: NAMES[(u * 3) % NAMES.length]! || `User ${u}`,
      department: DEPARTMENTS[(u - 1) % DEPARTMENTS.length]!,
      jobTitle: JOB_TITLES[(u - 1) % JOB_TITLES.length]!,
      roleId: role.id,
      roleName: role.displayName,
      isActive: u % 11 !== 0,
    })
  }
  // Bind roughly a third of the primary access links to a login user.
  accessLinks.forEach((link, idx) => {
    if (link.isPrimary && idx % 3 === 0) {
      link.userId = loginUsers[idx % loginUsers.length]!.id
    }
  })

  // ---- contributions: one row per employee.
  const contributions: EmployeeContributionSummary[] = employees.map((emp, idx) => {
    const active = Math.floor(rand() * 12)
    const completed = Math.floor(rand() * 40)
    const blocked = idx % 9 === 0 ? Math.floor(rand() * 4) : 0
    const overdue = idx % 6 === 0 ? Math.floor(rand() * 3) : 0
    const total = active + completed
    const primaryLink = accessLinks.find((l) => l.employeeId === emp.id && l.isPrimary)
    return {
      employeeId: emp.id,
      employeeCode: emp.employeeCode,
      employeeName: emp.fullName,
      department: emp.department,
      jobTitle: emp.jobTitle,
      managerEmployeeId: emp.managerEmployeeId,
      managerName: emp.managerName,
      employmentStatus: emp.employmentStatus,
      isActive: emp.isActive,
      activeProjectCount: idx % 5 === 0 ? 0 : 1 + Math.floor(rand() * 3),
      activeTaskCount: active,
      completedTaskCount: completed,
      blockedTaskCount: blocked,
      overdueTaskCount: overdue,
      completionRate: total > 0 ? Math.round((completed / total) * 1000) / 10 : 0,
      primaryLicenseKey: primaryLink?.licenseKey || '',
      primaryDeviceName: primaryLink?.deviceName || '',
    }
  })

  // ---- project assignments: a small synthetic project pool, 0-4
  // assignments per employee.
  const PROJECTS = [
    'Refinery Instrumentation Retrofit',
    'Port Calibration Services Contract',
    'Regional Warehouse Rollout',
    'ERP Data Migration',
    'Field Service Mobility Pilot',
    '',
  ]
  const projectAssignments: ProjectAssignment[] = []
  let assignmentSeq = 0
  for (const emp of employees) {
    const count = Math.floor(rand() * 5)
    for (let a = 0; a < count; a++) {
      assignmentSeq += 1
      const project = PROJECTS[assignmentSeq % PROJECTS.length]!
      projectAssignments.push({
        id: `assign-${assignmentSeq}`,
        projectId: `proj-${(assignmentSeq % PROJECTS.length) + 1}`,
        employeeId: emp.id,
        employeeName: emp.fullName,
        projectName: project || 'Untitled project',
        role: assignmentSeq % 4 === 0 ? 'Lead' : 'Member',
        allocationPercent: 10 * (1 + (assignmentSeq % 10)),
        isActive: assignmentSeq % 6 !== 0,
        joinedAt: `2025-${pad(1 + (assignmentSeq % 12), 2)}-01`,
        leftAt: assignmentSeq % 6 === 0 ? '2026-02-01' : '',
      })
    }
  }

  // ---- compliance documents: PII hot-zone. Adversarial: expiry in the
  // past (negative days), null expiry, ~2099 (effectively never), and one
  // permit with an EMPTY subtype.
  const complianceDocuments: EmployeeComplianceDocument[] = []
  let docSeqLocal = 0
  for (const emp of employees) {
    const count = emp.id === 'emp-3' ? 3 : Math.floor(rand() * 3)
    for (let d = 0; d < count; d++) {
      docSeqLocal += 1
      const docType = DOC_TYPES[docSeqLocal % DOC_TYPES.length]!
      const raw = `${docType.toUpperCase()}-${pad(docSeqLocal, 6)}`
      let expiresOn: string | null = `2026-${pad(1 + (docSeqLocal % 12), 2)}-${pad(1 + (docSeqLocal % 27), 2)}`
      if (docSeqLocal % 7 === 0) expiresOn = '2024-01-15' // already expired (negative days)
      if (docSeqLocal % 11 === 0) expiresOn = null // no expiry on file
      if (docSeqLocal % 13 === 0) expiresOn = '2099-12-31' // effectively never
      complianceDocuments.push({
        id: `doc-${docSeqLocal}`,
        employeeId: emp.id,
        docType,
        permitSubtype: docType === 'permit' ? (docSeqLocal % 5 === 0 ? '' : 'Work permit') : '',
        docNumber: raw,
        docNumberMasked: `••••${raw.slice(-4)}`,
        expiresOn,
        notes: docSeqLocal % 4 === 0 ? '' : 'Synthetic compliance document.',
        notifiedAt: null,
        createdAt: '2025-06-01',
        updatedAt: '2025-06-01',
      })
    }
  }
  documentSeq = docSeqLocal

  return {
    employees,
    accessLinks,
    licenseKeys,
    contributions,
    projectAssignments,
    complianceDocuments,
    loginUsers,
    loginRoles,
  }
}

function ensureCache(): Dataset {
  cache ??= generate()
  return cache
}

async function mockFetchEmployees(activeOnly: boolean): Promise<EmployeeProfile[]> {
  const ds = ensureCache()
  await sleep(220)
  return ds.employees.filter((e) => !activeOnly || e.isActive).map((e) => ({ ...e }))
}

async function mockFetchAccessLinks(): Promise<EmployeeAccessLink[]> {
  const ds = ensureCache()
  await sleep(180)
  return ds.accessLinks.map((l) => ({ ...l }))
}

async function mockFetchLicenseKeys(): Promise<LicenseKeySummary[]> {
  const ds = ensureCache()
  await sleep(150)
  return ds.licenseKeys.map((k) => ({ ...k }))
}

async function mockFetchContributions(): Promise<EmployeeContributionSummary[]> {
  const ds = ensureCache()
  await sleep(200)
  return ds.contributions.map((c) => ({ ...c }))
}

async function mockFetchProjectAssignments(employeeId: string): Promise<ProjectAssignment[]> {
  const ds = ensureCache()
  await sleep(140)
  return ds.projectAssignments.filter((a) => a.employeeId === employeeId).map((a) => ({ ...a }))
}

async function mockFetchDocuments(employeeId: string): Promise<EmployeeComplianceDocument[]> {
  const ds = ensureCache()
  await sleep(140)
  return ds.complianceDocuments.filter((d) => d.employeeId === employeeId).map((d) => ({ ...d }))
}

async function mockFetchLoginUsers(): Promise<LoginUserSummary[]> {
  const ds = ensureCache()
  await sleep(150)
  return ds.loginUsers.map((u) => ({ ...u }))
}

async function mockFetchLoginRoles(): Promise<RoleSummary[]> {
  const ds = ensureCache()
  await sleep(100)
  return ds.loginRoles.map((r) => ({ ...r }))
}

async function mockFetchContext(): Promise<CurrentEmployeeContext> {
  await sleep(80)
  // Fixed "admin" context so the Access tab's admin-only composers (Issue
  // License / Login User & Role) exercise their full render path in the lab.
  return {
    employeeId: 'emp-1',
    employeeName: NAMES[0]!,
    licenseKey: 'LIC-0001',
    licenseRole: 'admin',
    deviceId: 'device-001',
    userId: 'login-1',
    resolvedBy: 'license',
    permissions: ['*'],
  }
}

/* ---- mock mutations (only reached when `usingWails()` is false — see file
 * header) ---- */

async function mockCreateEmployee(input: EmployeeCreateInput): Promise<EmployeeProfile> {
  const ds = ensureCache()
  await sleep(200)
  const id = `emp-new-${ds.employees.length + 1}`
  const manager = ds.employees.find((e) => e.id === input.managerEmployeeId)
  const created: EmployeeProfile = {
    id,
    employeeCode: `EMP-${pad(ds.employees.length + 1, 4)}`,
    fullName: input.fullName,
    preferredName: '',
    email: input.email,
    phone: input.phone,
    department: input.department,
    jobTitle: input.jobTitle,
    employmentStatus: 'active',
    managerEmployeeId: input.managerEmployeeId,
    managerName: manager?.fullName || '',
    startDate: input.startDate,
    endDate: '',
    emergencyContact: '',
    notes: '',
    isActive: true,
    archivedAt: '',
    archivedBy: '',
    archiveReason: '',
    archiveRequestId: '',
  }
  ds.employees.unshift(created)
  return { ...created }
}

function findEmployeeOrThrow(id: string): EmployeeProfile {
  const ds = ensureCache()
  const e = ds.employees.find((x) => x.id === id)
  if (!e) throw new Error(`Employee ${id} not found`)
  return e
}

async function mockUpdateEmployee(draft: EmployeeProfileDraft): Promise<EmployeeProfile> {
  await sleep(180)
  const e = findEmployeeOrThrow(draft.id)
  e.fullName = draft.fullName
  e.preferredName = draft.preferredName
  e.email = draft.email
  e.phone = draft.phone
  e.department = draft.department
  e.jobTitle = draft.jobTitle
  e.employmentStatus = draft.employmentStatus
  e.startDate = draft.startDate
  e.emergencyContact = draft.emergencyContact
  e.notes = draft.notes
  return { ...e }
}

async function mockSetEmploymentState(employeeId: string, isActive: boolean, employmentStatus: string): Promise<EmployeeProfile> {
  await sleep(150)
  const e = findEmployeeOrThrow(employeeId)
  e.isActive = isActive
  e.employmentStatus = employmentStatus
  return { ...e }
}

async function mockRequestArchive(employeeId: string, reason: string): Promise<EmployeeArchiveApproval> {
  await sleep(180)
  const e = findEmployeeOrThrow(employeeId)
  e.archiveReason = reason
  archiveApprovalSeq += 1
  return {
    id: `arch-req-${archiveApprovalSeq}`,
    employeeId,
    employeeName: e.fullName,
    requestedBy: 'you (mock)',
    reason,
    status: 'pending',
  }
}

async function mockReviewArchive(requestId: string, decision: 'approve' | 'reject', notes: string): Promise<EmployeeArchiveApproval> {
  await sleep(120)
  return {
    id: requestId,
    employeeId: '',
    employeeName: '',
    requestedBy: 'you (mock)',
    reason: notes,
    status: decision === 'approve' ? 'approved' : 'rejected',
  }
}

async function mockReassignManager(employeeId: string, managerEmployeeId: string): Promise<EmployeeProfile> {
  await sleep(150)
  const ds = ensureCache()
  const e = findEmployeeOrThrow(employeeId)
  e.managerEmployeeId = managerEmployeeId
  e.managerName = ds.employees.find((x) => x.id === managerEmployeeId)?.fullName || ''
  return { ...e }
}

async function mockCreateAccessLink(link: {
  employeeId: string
  licenseKey: string
  userId?: string
  accessStatus?: string
}): Promise<EmployeeAccessLink> {
  const ds = ensureCache()
  await sleep(160)
  const emp = ds.employees.find((e) => e.id === link.employeeId)
  const existing = ds.accessLinks.find((l) => l.employeeId === link.employeeId && l.licenseKey === link.licenseKey)
  if (existing) {
    if (link.userId) existing.userId = link.userId
    existing.accessStatus = link.accessStatus || existing.accessStatus
    return { ...existing }
  }
  const created: EmployeeAccessLink = {
    id: `link-new-${ds.accessLinks.length + 1}`,
    employeeId: link.employeeId,
    employeeName: emp?.fullName || '',
    licenseKey: link.licenseKey,
    userId: link.userId || '',
    deviceId: '',
    deviceName: '',
    accessStatus: link.accessStatus || 'active',
    isPrimary: !ds.accessLinks.some((l) => l.employeeId === link.employeeId),
  }
  ds.accessLinks.push(created)
  return { ...created }
}

async function mockReassignLicense(employeeId: string, licenseKey: string): Promise<EmployeeAccessLink> {
  return mockCreateAccessLink({ employeeId, licenseKey })
}

async function mockGenerateLicense(role: string, notes: string, createdBy: string): Promise<string> {
  const ds = ensureCache()
  await sleep(160)
  const key = `LIC-${pad(ds.licenseKeys.length + 1, 4)}`
  ds.licenseKeys.push({
    id: `lic-new-${ds.licenseKeys.length + 1}`,
    key,
    role,
    deviceId: '',
    assignedTo: '',
    displayName: notes || `${role} seat`,
    status: 'Available',
  })
  void createdBy
  return key
}

async function mockCreateLoginUser(input: {
  username: string
  email: string
  fullName: string
  department: string
  jobTitle: string
  roleId: string
}): Promise<LoginUserSummary> {
  const ds = ensureCache()
  await sleep(200)
  const role = ds.loginRoles.find((r) => r.id === input.roleId)
  const created: LoginUserSummary = {
    id: `login-new-${ds.loginUsers.length + 1}`,
    username: input.username,
    email: input.email,
    fullName: input.fullName,
    department: input.department,
    jobTitle: input.jobTitle,
    roleId: input.roleId,
    roleName: role?.displayName || '',
    isActive: true,
  }
  ds.loginUsers.push(created)
  return { ...created }
}

async function mockCreateDocument(draft: EmployeeDocumentDraft & { employeeId: string }): Promise<EmployeeComplianceDocument> {
  const ds = ensureCache()
  await sleep(180)
  documentSeq += 1
  const created: EmployeeComplianceDocument = {
    id: `doc-new-${documentSeq}`,
    employeeId: draft.employeeId,
    docType: draft.docType,
    permitSubtype: draft.permitSubtype,
    docNumber: draft.docNumber,
    docNumberMasked: `••••${draft.docNumber.slice(-4)}`,
    expiresOn: draft.expiresOn,
    notes: draft.notes,
    notifiedAt: null,
    createdAt: todayIso(),
    updatedAt: todayIso(),
  }
  ds.complianceDocuments.push(created)
  return { ...created }
}

async function mockUpdateDocument(documentId: string, draft: EmployeeDocumentDraft): Promise<EmployeeComplianceDocument> {
  const ds = ensureCache()
  await sleep(160)
  const doc = ds.complianceDocuments.find((d) => d.id === documentId)
  if (!doc) throw new Error(`Compliance document ${documentId} not found`)
  doc.docType = draft.docType
  doc.permitSubtype = draft.permitSubtype
  if (draft.docNumber) {
    doc.docNumber = draft.docNumber
    doc.docNumberMasked = `••••${draft.docNumber.slice(-4)}`
  }
  doc.expiresOn = draft.expiresOn
  doc.notes = draft.notes
  doc.updatedAt = todayIso()
  return { ...doc }
}

async function mockDeleteDocument(documentId: string): Promise<void> {
  const ds = ensureCache()
  await sleep(140)
  ds.complianceDocuments = ds.complianceDocuments.filter((d) => d.id !== documentId)
}

/* ---- real: FETCH is wired (single-call, non-aggregating, confirmed against
 * frontend/wailsjs/go/main/App.d.ts + InfraService.d.ts); every mutation is
 * an honest INTEG-gap throw naming the exact real binding. ---- */

function mapEmployee(e: Record<string, unknown>): EmployeeProfile {
  return {
    id: str(e.id),
    employeeCode: str(e.employee_code),
    fullName: str(e.full_name),
    preferredName: str(e.preferred_name),
    email: str(e.email),
    phone: str(e.phone),
    department: str(e.department),
    jobTitle: str(e.job_title),
    employmentStatus: str(e.employment_status),
    managerEmployeeId: str(e.manager_employee_id),
    managerName: str(e.manager_name),
    startDate: goDate(e.start_date),
    endDate: goDate(e.end_date),
    emergencyContact: str(e.emergency_contact),
    notes: str(e.notes),
    isActive: Boolean(e.is_active),
    archivedAt: goDate(e.archived_at),
    archivedBy: str(e.archived_by),
    archiveReason: str(e.archive_reason),
    archiveRequestId: str(e.archive_request_id),
  }
}

function mapAccessLink(l: Record<string, unknown>): EmployeeAccessLink {
  return {
    id: str(l.id),
    employeeId: str(l.employee_id),
    employeeName: str(l.employee_name),
    licenseKey: str(l.license_key),
    userId: str(l.user_id),
    deviceId: str(l.device_id),
    deviceName: str(l.device_name),
    accessStatus: str(l.access_status),
    isPrimary: Boolean(l.is_primary),
  }
}

function mapLicenseKey(k: Record<string, unknown>): LicenseKeySummary {
  return {
    id: str(k.id),
    key: str(k.key),
    role: str(k.role),
    deviceId: str(k.device_hash),
    assignedTo: str(k.display_name),
    displayName: str(k.display_name),
    status: k.activated ? 'Activated' : 'Available',
  }
}

function mapContribution(c: Record<string, unknown>): EmployeeContributionSummary {
  return {
    employeeId: str(c.employee_id),
    employeeCode: str(c.employee_code),
    employeeName: str(c.employee_name),
    department: str(c.department),
    jobTitle: str(c.job_title),
    managerEmployeeId: str(c.manager_employee_id),
    managerName: str(c.manager_name),
    employmentStatus: str(c.employment_status),
    isActive: Boolean(c.is_active),
    activeProjectCount: num(c.active_project_count),
    activeTaskCount: num(c.active_task_count),
    completedTaskCount: num(c.completed_task_count),
    blockedTaskCount: num(c.blocked_task_count),
    overdueTaskCount: num(c.overdue_task_count),
    completionRate: num(c.completion_rate),
    primaryLicenseKey: str(c.primary_license_key),
    primaryDeviceName: str(c.primary_device_name),
  }
}

function mapProjectAssignment(m: Record<string, unknown>): ProjectAssignment {
  return {
    id: str(m.id),
    projectId: str(m.project_id),
    employeeId: str(m.employee_id),
    employeeName: str(m.employee_name),
    projectName: str(m.project_name),
    role: str(m.role),
    allocationPercent: num(m.allocation_percent),
    isActive: Boolean(m.is_active),
    joinedAt: goDate(m.joined_at),
    leftAt: goDate(m.left_at),
  }
}

function mapDocument(d: Record<string, unknown>): EmployeeComplianceDocument {
  return {
    id: str(d.id),
    employeeId: str(d.employee_id),
    docType: str(d.doc_type),
    permitSubtype: str(d.permit_subtype),
    docNumber: str(d.doc_number),
    docNumberMasked: str(d.doc_number_masked),
    expiresOn: goDate(d.expires_on) || null,
    notes: str(d.notes),
    notifiedAt: goDate(d.notified_at) || null,
    createdAt: goDate(d.created_at),
    updatedAt: goDate(d.updated_at),
  }
}

function mapLoginUser(u: Record<string, unknown>): LoginUserSummary {
  const role = u.role as Record<string, unknown> | undefined
  return {
    id: str(u.id),
    username: str(u.username),
    email: str(u.email),
    fullName: str(u.full_name) || str(u.display_name),
    department: str(u.department),
    jobTitle: str(u.job_title),
    roleId: str(u.role_id),
    roleName: str(u.role_name) || (role ? str(role.display_name) || str(role.name) : ''),
    isActive: Boolean(u.is_active),
  }
}

function mapRole(r: Record<string, unknown>): RoleSummary {
  return {
    id: str(r.id),
    name: str(r.name),
    displayName: str(r.display_name),
  }
}

function mapContext(c: Record<string, unknown>): CurrentEmployeeContext {
  return {
    employeeId: str(c.employee_id),
    employeeName: str(c.employee_name),
    licenseKey: str(c.license_key),
    licenseRole: str(c.license_role),
    deviceId: str(c.device_id),
    userId: str(c.user_id),
    resolvedBy: str(c.resolved_by),
    permissions: Array.isArray(c.permissions) ? (c.permissions as string[]) : [],
  }
}

async function realFetchEmployees(activeOnly: boolean): Promise<EmployeeProfile[]> {
  const rows = await ListEmployeeProfiles(activeOnly)
  return (rows ?? []).map((r) => mapEmployee(r as unknown as Record<string, unknown>))
}

async function realFetchAccessLinks(): Promise<EmployeeAccessLink[]> {
  const rows = await ListEmployeeAccessLinks()
  return (rows ?? []).map((r) => mapAccessLink(r as unknown as Record<string, unknown>))
}

async function realFetchLicenseKeys(): Promise<LicenseKeySummary[]> {
  const rows = await ListLicenseKeys()
  return (rows ?? []).map((r) => mapLicenseKey(r as unknown as Record<string, unknown>))
}

async function realFetchContributions(): Promise<EmployeeContributionSummary[]> {
  const rows = await ListEmployeeContributionSummaries()
  return (rows ?? []).map((r) => mapContribution(r as unknown as Record<string, unknown>))
}

async function realFetchProjectAssignments(employeeId: string): Promise<ProjectAssignment[]> {
  const rows = await ListEmployeeProjectAssignments(employeeId)
  return (rows ?? []).map((r) => mapProjectAssignment(r as unknown as Record<string, unknown>))
}

async function realFetchDocuments(employeeId: string): Promise<EmployeeComplianceDocument[]> {
  const rows = await ListEmployeeDocuments(employeeId)
  return (rows ?? []).map((r) => mapDocument(r as unknown as Record<string, unknown>))
}

async function realFetchLoginUsers(): Promise<LoginUserSummary[]> {
  const rows = await ListUsers()
  return (rows ?? []).map((r) => mapLoginUser(r as unknown as Record<string, unknown>))
}

async function realFetchLoginRoles(): Promise<RoleSummary[]> {
  const rows = await ListRoles()
  return (rows ?? []).map((r) => mapRole(r as unknown as Record<string, unknown>))
}

async function realFetchContext(): Promise<CurrentEmployeeContext | null> {
  try {
    const ctx = await GetCurrentEmployeeContext()
    return mapContext(ctx as unknown as Record<string, unknown>)
  } catch {
    return null
  }
}

async function realCreateEmployee(_input: EmployeeCreateInput): Promise<EmployeeProfile> {
  throw new Error('INTEG gap: CreateEmployeeProfile — wires at K5')
}
async function realUpdateEmployee(_draft: EmployeeProfileDraft): Promise<EmployeeProfile> {
  throw new Error('INTEG gap: UpdateEmployeeProfile — wires at K5')
}
async function realSetEmploymentState(_employeeId: string, _isActive: boolean, _employmentStatus: string): Promise<EmployeeProfile> {
  throw new Error('INTEG gap: SetEmployeeEmploymentState — wires at K5')
}
async function realRequestArchive(_employeeId: string, _reason: string): Promise<EmployeeArchiveApproval> {
  throw new Error('INTEG gap: RequestEmployeeArchive — HOT-ZONE (cascades access + project revocation, routes to Approvals), wires at K5')
}
async function realReviewArchive(_requestId: string, _decision: 'approve' | 'reject', _notes: string): Promise<EmployeeArchiveApproval> {
  throw new Error('INTEG gap: ReviewEmployeeArchiveRequest — wires at K5')
}
async function realReassignManager(_employeeId: string, _managerEmployeeId: string): Promise<EmployeeProfile> {
  throw new Error('INTEG gap: ReassignEmployeeManager — wires at K5')
}
async function realCreateAccessLink(_link: { employeeId: string; licenseKey: string; userId?: string; accessStatus?: string }): Promise<EmployeeAccessLink> {
  throw new Error('INTEG gap: CreateEmployeeAccessLink — wires at K5')
}
async function realReassignLicense(_employeeId: string, _licenseKey: string): Promise<EmployeeAccessLink> {
  throw new Error('INTEG gap: ReassignEmployeeLicenseAccess — wires at K5')
}
async function realGenerateLicense(_role: string, _notes: string, _createdBy: string): Promise<string> {
  throw new Error('INTEG gap: GenerateLicenseKey — mints a live credential, wires at K5')
}
async function realCreateLoginUser(_input: {
  username: string
  email: string
  fullName: string
  department: string
  jobTitle: string
  roleId: string
}): Promise<LoginUserSummary> {
  throw new Error('INTEG gap: CreateUser — mints a live login credential + temp password, wires at K5')
}
async function realCreateDocument(_draft: EmployeeDocumentDraft & { employeeId: string }): Promise<EmployeeComplianceDocument> {
  throw new Error('INTEG gap: CreateEmployeeDocument — PII hot-zone (field-encrypted doc number), wires at K5')
}
async function realUpdateDocument(_documentId: string, _draft: EmployeeDocumentDraft): Promise<EmployeeComplianceDocument> {
  throw new Error('INTEG gap: UpdateEmployeeDocument — PII hot-zone, wires at K5')
}
async function realDeleteDocument(_documentId: string): Promise<void> {
  throw new Error('INTEG gap: DeleteEmployeeDocument — wires at K5')
}

/* ---- public switched API (viewmodel imports THESE) ---- */
export const fetchEmployees = (activeOnly = false): Promise<EmployeeProfile[]> => pick(realFetchEmployees, mockFetchEmployees)(activeOnly)
export const fetchAccessLinks = (): Promise<EmployeeAccessLink[]> => pick(realFetchAccessLinks, mockFetchAccessLinks)()
export const fetchLicenseKeys = (): Promise<LicenseKeySummary[]> => pick(realFetchLicenseKeys, mockFetchLicenseKeys)()
export const fetchContributions = (): Promise<EmployeeContributionSummary[]> => pick(realFetchContributions, mockFetchContributions)()
export const fetchProjectAssignments = (employeeId: string): Promise<ProjectAssignment[]> =>
  pick(realFetchProjectAssignments, mockFetchProjectAssignments)(employeeId)
export const fetchDocuments = (employeeId: string): Promise<EmployeeComplianceDocument[]> =>
  pick(realFetchDocuments, mockFetchDocuments)(employeeId)
export const fetchLoginUsers = (): Promise<LoginUserSummary[]> => pick(realFetchLoginUsers, mockFetchLoginUsers)()
export const fetchLoginRoles = (): Promise<RoleSummary[]> => pick(realFetchLoginRoles, mockFetchLoginRoles)()
export const fetchCurrentEmployeeContext = (): Promise<CurrentEmployeeContext | null> => pick(realFetchContext, mockFetchContext)()

export const createEmployee = (input: EmployeeCreateInput): Promise<EmployeeProfile> => pick(realCreateEmployee, mockCreateEmployee)(input)
export const updateEmployee = (draft: EmployeeProfileDraft): Promise<EmployeeProfile> => pick(realUpdateEmployee, mockUpdateEmployee)(draft)
export const setEmploymentState = (employeeId: string, isActive: boolean, employmentStatus: string): Promise<EmployeeProfile> =>
  pick(realSetEmploymentState, mockSetEmploymentState)(employeeId, isActive, employmentStatus)
export const requestEmployeeArchive = (employeeId: string, reason: string): Promise<EmployeeArchiveApproval> =>
  pick(realRequestArchive, mockRequestArchive)(employeeId, reason)
export const reviewEmployeeArchive = (requestId: string, decision: 'approve' | 'reject', notes = ''): Promise<EmployeeArchiveApproval> =>
  pick(realReviewArchive, mockReviewArchive)(requestId, decision, notes)
export const reassignManager = (employeeId: string, managerEmployeeId: string): Promise<EmployeeProfile> =>
  pick(realReassignManager, mockReassignManager)(employeeId, managerEmployeeId)
export const createAccessLink = (link: { employeeId: string; licenseKey: string; userId?: string; accessStatus?: string }): Promise<EmployeeAccessLink> =>
  pick(realCreateAccessLink, mockCreateAccessLink)(link)
export const reassignLicense = (employeeId: string, licenseKey: string): Promise<EmployeeAccessLink> =>
  pick(realReassignLicense, mockReassignLicense)(employeeId, licenseKey)
export const generateLicenseKey = (role: string, notes: string, createdBy: string): Promise<string> =>
  pick(realGenerateLicense, mockGenerateLicense)(role, notes, createdBy)
export const createLoginUser = (input: {
  username: string
  email: string
  password: string
  fullName: string
  department: string
  jobTitle: string
  roleId: string
}): Promise<LoginUserSummary> =>
  pick(realCreateLoginUser, mockCreateLoginUser)(input)
export const createDocument = (draft: EmployeeDocumentDraft & { employeeId: string }): Promise<EmployeeComplianceDocument> =>
  pick(realCreateDocument, mockCreateDocument)(draft)
export const updateDocument = (documentId: string, draft: EmployeeDocumentDraft): Promise<EmployeeComplianceDocument> =>
  pick(realUpdateDocument, mockUpdateDocument)(documentId, draft)
export const deleteDocument = (documentId: string): Promise<void> => pick(realDeleteDocument, mockDeleteDocument)(documentId)

/** Client-side role vocabulary for the Access tab's Issue License picker —
 * mirrors payroll.ts's payrollDivisionOptions() convention (a fixed lab-only
 * set until a real roles-for-licensing binding lands at K5). */
export const issuableLicenseRoles = (): string[] => [...ROLE_KEYS]
