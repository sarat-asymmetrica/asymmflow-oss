/* Work bridge module — self-contained: types + mock + real + switch (K4
 * operational hub). Old screen: WorkHub.svelte (1445 lines) + 5 child
 * components + api/collaboration.ts — a tasks/projects collaboration
 * workspace over the `main.TaskItem` / `main.Project` / `main.ProjectMember`
 * tables. Transport confirmed: Wails IPC only, split across two services —
 * `SyncServiceBinding` (tasks/projects/members/activity/comments/employees/
 * current-employee-context) and `App` (`GetEmployeeAllocationSummary`) —
 * verified against `frontend/wailsjs/go/main/{SyncServiceBinding,App}.d.ts`.
 *
 * FETCH bindings are real (single-call, non-aggregating list/get calls), and
 * (R3) every MUTATION is now wired too — all 14 real* adapters call their
 * named SyncServiceBinding function directly; every actor (creator/CreatedBy)
 * is derived server-side from GetCurrentEmployeeContext/getCurrentUserID, so
 * none of these bindings takes a client-supplied actor arg. Project
 * archive/shelve/delete carry a mandatory audit reason (HOT-ZONE, threaded
 * from the caller); task delete is likewise irreversible (no reason param on
 * that one binding). Synthetic-only mock data (SYNTHETIC_IDENTITY.md) —
 * invented Gulf names, no real people/companies. */

import { pick } from './runtime'
import { goDate, goTime, num, str } from './map'
import type { main } from '$wails/go/models'
import {
  AddCollaborativeProjectMember,
  AddCollaborativeTaskComment,
  ArchiveCollaborativeProject,
  CreateCollaborativeProject,
  CreateCollaborativeTask,
  DeleteCollaborativeProject,
  DeleteCollaborativeTask,
  GetCollaborativeTask,
  GetCurrentEmployeeContext,
  GetProjectTaskCounts,
  ListCollaborativeProjectActivity,
  ListCollaborativeProjectMembers,
  ListCollaborativeProjectTasks,
  ListCollaborativeProjects,
  ListCollaborativeTaskActivity,
  ListCollaborativeTaskComments,
  ListCollaborativeTeamTasks,
  ListEmployeeProfiles,
  ListMyCollaborativeTasks,
  ReassignCollaborativeTask,
  RefreshCollaborativeWorkspace,
  ShelveCollaborativeProject,
  UpdateCollaborativeProject,
  UpdateCollaborativeTask,
  UpdateCollaborativeTaskDueDate,
  UpdateCollaborativeTaskStatus,
} from '$wails/go/main/SyncServiceBinding'
import { GetEmployeeAllocationSummary } from '$wails/go/main/App'

/* ---- types (camelCase; mirrors main.TaskItem / main.Project / main.ProjectMember / main.AllocationSummary) ---- */

export interface TaskItem {
  id: string
  title: string
  description: string
  taskType: string
  status: string
  blockedReason: string
  priority: string
  dueDate: string
  customerId: string
  opportunityId: string
  orderId: string
  projectId: string
  creatorEmployeeId: string
  assigneeEmployeeId: string
  creatorName: string
  assigneeName: string
  startedAt: string
  completedAt: string
  lastCommentAt: string
}

export interface TaskDraft {
  title: string
  description: string
  priority: string
  dueDate: string
  projectId: string
  assigneeEmployeeId: string
}

export interface Project {
  id: string
  name: string
  projectType: string
  description: string
  status: string
  customerId: string
  opportunityId: string
  orderId: string
  customerName: string
  endUserName: string
  opportunityKey: string
  customerPocName: string
  customerPocEmail: string
  customerPocPhone: string
  startsOn: string
  endsOn: string
}

export interface ProjectDraft {
  name: string
  projectType: string
  description: string
  customerName: string
  customerPocName: string
  customerPocEmail: string
  customerPocPhone: string
}

/** Update patch — the create-time draft fields plus `status` (restore-from-
 * archived is a status-only patch; UpdateCollaborativeProject's real arg2 is
 * an untyped `Record<string, any>`, so this stays permissive on purpose). */
export type ProjectPatch = Partial<ProjectDraft> & { status?: string }

export interface ProjectMember {
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

export interface TaskComment {
  id: string
  taskId: string
  employeeId: string
  employeeName: string
  body: string
  createdAt: string
}

export interface TaskActivity {
  id: string
  taskId: string
  employeeId: string
  employeeName: string
  activityType: string
  detail: string
  metadataJson: string
  createdAt: string
}

export interface AllocationProjectLine {
  projectId: string
  projectName: string
  allocationPercent: number
}

/** Wave 9.8 B3 pattern: read-only precheck for the allocation-capacity WARN.
 * The server (real: GetEmployeeAllocationSummary) computes otherProjectsTotal
 * — the UI only ever displays what comes back here, never re-derives it. */
export interface AllocationSummary {
  employeeId: string
  otherProjectsTotal: number
  projects: AllocationProjectLine[]
}

export interface EmployeeOption {
  id: string
  name: string
  email: string
  department: string
  jobTitle: string
  isActive: boolean
}

export interface CurrentEmployeeContext {
  employeeId: string
  employeeName: string
  licenseRole: string
  resolvedBy: string
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
const daysAgoIso = (days: number): string => {
  const d = new Date()
  d.setDate(d.getDate() - days)
  return d.toISOString().slice(0, 10)
}

const MONSTER_EMPLOYEE_NAME =
  'MOHAMMEDABDULRAHMANALSHAMSISENIORFIELDINSTRUMENTATIONANDCALIBRATIONTECHNICIANFORMERLYGULFTECHNICALSERVICESDEPARTMENTOFPROCESSCONTROLENGINEERING'.padEnd(
    200,
    'X',
  )

// index 0 (emp-1) is the mock "current employee" — everything else is
// adversarial seasoning: 2-char, empty, RTL, 200-char monster, single-char.
// 60 total: 20 hand-authored (adversarial seasoning) + 40 synthetic filler —
// proj-mega's 55 members each need a DISTINCT employee (a real project can
// never carry two active memberships for the same employee, per
// AddCollaborativeProjectMember's upsert-on-(project,employee) semantics), so
// the roster must outnumber the perf-monster's member count.
const EMPLOYEE_NAMES = [
  'Ahmed Al-Khalifa',
  'Fatima Mohammed Al-Sayed',
  'Yusuf Kanoo',
  'Al',
  '',
  'محمد عبدالله الأنصاري الكوري',
  MONSTER_EMPLOYEE_NAME,
  'Layla Haidar',
  'Karim Nasser',
  'Bashir Al-Ansari',
  'X',
  'Noor Al-Zayani',
  'Hessa Al-Dosari',
  'Omar Fakhro',
  'Sara Buali',
  'Rashid Al-Marzooqi',
  'Dana Khalil',
  'Ibrahim Youssef',
  'Maya Haddad',
  'Talal Jassim',
  ...Array.from({ length: 40 }, (_, i) => `Field Associate ${21 + i}`),
]
const DEPARTMENTS = ['Field Services', 'Finance', 'Sales', 'Operations', 'Engineering', 'HR']
const JOB_TITLES = ['Field Engineer', 'Calibration Technician', 'Finance Officer', 'Sales Executive', 'Operations Coordinator', 'HR Specialist', '']
const CUSTOMERS = [
  'Gulf Fabrication W.L.L.',
  'Manama Process Systems',
  'Al Dana Engineering Co.',
  'المؤسسة الدولية لخدمات الأجهزة الصناعية والبتروكيماوية والتجارة العامة ذ.م.م',
  'Sitra Contracting',
]
const TASK_TITLES = [
  'Draft scope of work',
  'Follow up with site contact',
  'Prepare cost estimate',
  'Review vendor quotation',
  'Schedule site survey',
  'Update project timeline',
  'Coordinate delivery logistics',
  'Reconcile purchase order',
  'Escalate blocked shipment',
  'Prepare handover documentation',
  '', // adversarial: blank title
]
const MONSTER_TASK_TITLE =
  'RECONCILE THE THIRD-PARTY VENDOR QUOTATION AGAINST THE ORIGINAL PURCHASE ORDER LINE ITEMS INCLUDING FREIGHT INSURANCE AND CUSTOMS CLEARANCE CHARGES BEFORE ESCALATING TO PROCUREMENT'.padEnd(
    200,
    '.',
  )
const STATUSES = ['todo', 'in_progress', 'blocked', 'completed']
const PRIORITIES = ['low', 'medium', 'high', 'urgent']

function emailFor(name: string, idx: number): string {
  if (!name) return ''
  if (idx === 10) return 'x@x' // adversarial: invalid-looking (no TLD), round-trips unchanged
  const slug = name
    .toLowerCase()
    .replace(/[^a-z\s]/g, '')
    .trim()
    .split(/\s+/)
    .join('.')
  return slug ? `${slug}@example.test` : ''
}

interface Dataset {
  employees: EmployeeOption[]
  projects: Project[]
  members: ProjectMember[]
  tasks: TaskItem[]
  comments: TaskComment[]
  activities: TaskActivity[]
}

let cache: Dataset | null = null
let taskSeq = 0
let commentSeq = 0
let projectSeq = 0
let memberSeq = 0

function generate(): Dataset {
  const rand = lcg(20260714 ^ 0x574f524b /* 'WORK' */)

  // ---- employees ----
  const employees: EmployeeOption[] = EMPLOYEE_NAMES.map((name, i) => ({
    id: `emp-${i + 1}`,
    name,
    email: emailFor(name, i),
    department: DEPARTMENTS[i % DEPARTMENTS.length]!,
    jobTitle: JOB_TITLES[i % JOB_TITLES.length]!,
    isActive: i !== 12, // emp-13 (Hessa) is inactive — exercises "no longer active" assignee rendering
  }))
  const empId = (i: number) => employees[i]!.id
  const empName = (i: number) => employees[i]!.name

  // ---- projects ----
  const projects: Project[] = [
    { id: 'proj-1', name: 'Flow Metering Skid Replacement', projectType: 'customer', description: 'Replace aging flow metering skids across two process trains.', status: 'active', customerId: 'cust-1', opportunityId: '', orderId: '', customerName: CUSTOMERS[0]!, endUserName: CUSTOMERS[0]!, opportunityKey: 'OPP-2026-0041', customerPocName: 'Yusuf Kanoo', customerPocEmail: 'yusuf.kanoo@example.test', customerPocPhone: '+973 3900 1122', startsOn: '2026-02-01', endsOn: '' },
    { id: 'proj-2', name: 'Internal Systems Upgrade', projectType: 'internal', description: 'ERP and ticketing tooling refresh.', status: 'active', customerId: '', opportunityId: '', orderId: '', customerName: '', endUserName: '', opportunityKey: '', customerPocName: '', customerPocEmail: '', customerPocPhone: '', startsOn: '2026-01-15', endsOn: '' },
    { id: 'proj-3', name: 'DCS Migration — Phase 2', projectType: 'customer', description: 'Distributed control system migration, phase 2 of 3.', status: 'active', customerId: 'cust-2', opportunityId: '', orderId: '', customerName: CUSTOMERS[1]!, endUserName: CUSTOMERS[1]!, opportunityKey: 'OPP-2026-0058', customerPocName: 'Layla Haidar', customerPocEmail: 'layla.haidar@example.test', customerPocPhone: '+973 3900 3344', startsOn: '2026-03-01', endsOn: '' },
    { id: 'proj-4', name: 'Warehouse Reorganization', projectType: 'internal', description: 'Consolidate two warehouse sites into one.', status: 'active', customerId: '', opportunityId: '', orderId: '', customerName: '', endUserName: '', opportunityKey: '', customerPocName: '', customerPocEmail: '', customerPocPhone: '', startsOn: '', endsOn: '' },
    { id: 'proj-5', name: 'Tank Farm Level Instrumentation', projectType: 'customer', description: 'Radar-level transmitter retrofit across 12 tanks.', status: 'active', customerId: 'cust-3', opportunityId: '', orderId: '', customerName: CUSTOMERS[2]!, endUserName: CUSTOMERS[2]!, opportunityKey: 'OPP-2026-0063', customerPocName: 'Karim Nasser', customerPocEmail: 'karim.nasser@example.test', customerPocPhone: '+973 3900 5566', startsOn: '2026-04-10', endsOn: '' },
    { id: 'proj-6', name: 'Turbine Control Retrofit', projectType: 'customer', description: 'Governor and control retrofit on two gas turbines.', status: 'active', customerId: 'cust-1', opportunityId: '', orderId: '', customerName: CUSTOMERS[0]!, endUserName: CUSTOMERS[0]!, opportunityKey: '', customerPocName: 'Bashir Al-Ansari', customerPocEmail: 'bashir.alansari@example.test', customerPocPhone: '', startsOn: '2026-05-01', endsOn: '' },
    { id: 'proj-archived', name: 'Legacy ERP Rollout', projectType: 'internal', description: 'Superseded by the Internal Systems Upgrade.', status: 'archived', customerId: '', opportunityId: '', orderId: '', customerName: '', endUserName: '', opportunityKey: '', customerPocName: '', customerPocEmail: '', customerPocPhone: '', startsOn: '2024-01-01', endsOn: '2025-06-30' },
    { id: 'proj-shelved', name: 'Paused Automation Pilot', projectType: 'internal', description: 'On hold pending budget approval.', status: 'shelved', customerId: '', opportunityId: '', orderId: '', customerName: '', endUserName: '', opportunityKey: '', customerPocName: '', customerPocEmail: '', customerPocPhone: '', startsOn: '2025-09-01', endsOn: '' },
    // Adversarial: unrecognized status — must render (neutral badge), never crash, and stay visible in the default active list.
    { id: 'proj-unknown', name: 'Vendor Consolidation Review', projectType: 'internal', description: 'Status migrated from a retired workflow state.', status: 'UNKNOWN_STATUS', customerId: '', opportunityId: '', orderId: '', customerName: '', endUserName: '', opportunityKey: '', customerPocName: '', customerPocEmail: '', customerPocPhone: '', startsOn: '', endsOn: '' },
    // Adversarial: empty-state project — 0 members, 0 tasks.
    { id: 'proj-empty', name: 'New Business Unit Setup', projectType: 'internal', description: 'Not yet staffed.', status: 'active', customerId: '', opportunityId: '', orderId: '', customerName: '', endUserName: '', opportunityKey: '', customerPocName: '', customerPocEmail: '', customerPocPhone: '', startsOn: '', endsOn: '' },
    // Adversarial: perf monster — 55 members / 220 tasks (seeded below).
    { id: 'proj-mega', name: 'Enterprise Rollout — All Sites', projectType: 'customer', description: 'Company-wide rollout spanning every division.', status: 'active', customerId: 'cust-4', opportunityId: '', orderId: '', customerName: 'Bahrain Water Authority — Directorate of Operations & Maintenance, Section 7', endUserName: '', opportunityKey: 'OPP-2026-0002', customerPocName: 'Noor Al-Zayani', customerPocEmail: 'noor.alzayani@example.test', customerPocPhone: '', startsOn: '2025-11-01', endsOn: '' },
    // Adversarial: RTL customer_name + an invalid-looking POC email that round-trips (no domain).
    { id: 'proj-rtl', name: 'مشروع تحديث الأنظمة الصناعية', projectType: 'customer', description: 'ترقية أنظمة التحكم الصناعية', status: 'active', customerId: 'cust-5', opportunityId: '', orderId: '', customerName: 'المؤسسة الدولية لخدمات الأجهزة الصناعية والبتروكيماوية والتجارة العامة ذ.م.م', endUserName: '', opportunityKey: '', customerPocName: 'محمد عبدالله الأنصاري الكوري', customerPocEmail: 'contact@', customerPocPhone: '', startsOn: '', endsOn: '' },
  ]

  // ---- project members ----
  const members: ProjectMember[] = []
  const ROLES = ['Lead', 'Member', 'Contributor', 'Reviewer']
  function addMember(projectId: string, employeeIdx: number, role: string, allocation: number, active = true) {
    memberSeq += 1
    members.push({
      id: `member-${memberSeq}`,
      projectId,
      employeeId: empId(employeeIdx),
      employeeName: empName(employeeIdx),
      projectName: projects.find((p) => p.id === projectId)?.name ?? '',
      role,
      allocationPercent: allocation,
      isActive: active,
      joinedAt: '2026-01-15',
      leftAt: active ? '' : '2026-05-01',
    })
  }
  addMember('proj-1', 0, 'Lead', 60)
  addMember('proj-1', 2, 'Member', 40)
  addMember('proj-1', 7, 'Contributor', 20)
  addMember('proj-2', 3, 'Lead', 30)
  addMember('proj-2', 8, 'Member', 100) // boundary: exactly 100%
  addMember('proj-3', 0, 'Reviewer', 15)
  addMember('proj-3', 9, 'Lead', 70)
  addMember('proj-3', 15, 'Member', 25)
  addMember('proj-4', 11, 'Lead', 50)
  addMember('proj-4', 13, 'Contributor', 10, false) // inactive membership
  addMember('proj-5', 2, 'Lead', 50)
  addMember('proj-5', 16, 'Member', 340) // already over 100% before any new addition
  addMember('proj-6', 9, 'Lead', 45)
  addMember('proj-6', 17, 'Member', 35)
  addMember('proj-archived', 0, 'Lead', 20, false)
  addMember('proj-shelved', 3, 'Member', 40)
  addMember('proj-unknown', 8, 'Lead', 30)
  addMember('proj-rtl', 5, 'Lead', 55)
  addMember('proj-rtl', 18, 'Member', 20)
  // proj-empty: deliberately 0 members.
  // proj-mega: 55 members, each a DISTINCT employee (i is unique and the
  // roster is 60-strong — see the EMPLOYEE_NAMES comment), modest allocations
  // so the cross-project allocation precheck stays meaningful elsewhere.
  for (let i = 0; i < 55; i++) {
    addMember('proj-mega', i, ROLES[i % ROLES.length]!, 5 + (i % 15))
  }

  // ---- tasks ----
  const tasks: TaskItem[] = []
  const comments: TaskComment[] = []
  const activities: TaskActivity[] = []

  function addTask(opts: {
    projectId: string
    titleIdx: number
    assigneeIdx: number | null
    status: string
    priority: string
    dueDate: string
    blockedReason?: string
    completedAt?: string
  }) {
    taskSeq += 1
    const id = `task-${taskSeq}`
    const assignee = opts.assigneeIdx == null ? null : employees[opts.assigneeIdx]!
    tasks.push({
      id,
      title: TASK_TITLES[opts.titleIdx % TASK_TITLES.length] || `Task ${taskSeq}`,
      description: taskSeq % 4 === 0 ? '' : `Synthetic task description #${taskSeq}.`,
      taskType: taskSeq % 6 === 0 ? 'follow_up' : 'action',
      status: opts.status,
      blockedReason: opts.blockedReason ?? '',
      priority: opts.priority,
      dueDate: opts.dueDate,
      customerId: '',
      opportunityId: '',
      orderId: '',
      projectId: opts.projectId,
      creatorEmployeeId: empId(0),
      assigneeEmployeeId: assignee?.id ?? '',
      creatorName: empName(0),
      assigneeName: assignee?.name ?? '',
      startedAt: opts.status === 'todo' ? '' : daysAgoIso(10),
      completedAt: opts.completedAt ?? '',
      lastCommentAt: '',
    })
    // Sparse comments/activity keep the dataset perf-realistic while still
    // giving the mega project (220 tasks) a meaningful activity feed.
    if (taskSeq % 6 === 0) {
      commentSeq += 1
      comments.push({
        id: `comment-${commentSeq}`,
        taskId: id,
        employeeId: empId((taskSeq + 1) % employees.length),
        employeeName: empName((taskSeq + 1) % employees.length),
        body: `Update on task ${taskSeq}: progressing as planned.`,
        createdAt: daysAgoIso(taskSeq % 20),
      })
    }
    if (taskSeq % 9 === 0) {
      activities.push({
        id: `activity-${taskSeq}`,
        taskId: id,
        employeeId: empId(0),
        employeeName: empName(0),
        activityType: 'status_changed',
        detail: `Status set to ${opts.status}.`,
        metadataJson: '',
        createdAt: daysAgoIso(taskSeq % 15),
      })
    }
  }

  const NORMAL_PROJECT_IDS = ['proj-1', 'proj-2', 'proj-3', 'proj-4', 'proj-5', 'proj-6', 'proj-archived', 'proj-shelved', 'proj-unknown', 'proj-rtl']
  let cursor = 0
  for (const projectId of NORMAL_PROJECT_IDS) {
    const count = 3 + (cursor % 6)
    for (let j = 0; j < count; j++) {
      cursor += 1
      const r = rand()
      const status = STATUSES[Math.floor(r * STATUSES.length)]!
      const priority = PRIORITIES[Math.floor(rand() * PRIORITIES.length)]!
      const assigneeIdx = cursor % employees.length
      addTask({
        projectId,
        titleIdx: cursor % TASK_TITLES.length,
        assigneeIdx,
        status,
        priority,
        dueDate: `2026-${pad(1 + (cursor % 12), 2)}-${pad(1 + (cursor % 27), 2)}`,
      })
    }
  }

  // Unassigned-project task bucket (~15 rows) — exercises "no project" filtering.
  for (let j = 0; j < 15; j++) {
    cursor += 1
    addTask({
      projectId: '',
      titleIdx: cursor % TASK_TITLES.length,
      assigneeIdx: cursor % 3 === 0 ? null : cursor % employees.length,
      status: STATUSES[cursor % STATUSES.length]!,
      priority: PRIORITIES[cursor % PRIORITIES.length]!,
      dueDate: cursor % 5 === 0 ? '' : `2026-${pad(1 + (cursor % 12), 2)}-15`,
    })
  }

  // proj-mega: 220 tasks, mostly assigned to the current employee (emp-1) or
  // cycling the roster, so Team Board's perf case has real weight.
  for (let j = 0; j < 220; j++) {
    cursor += 1
    const assigneeIdx = j % 4 === 0 ? 0 : cursor % employees.length
    addTask({
      projectId: 'proj-mega',
      titleIdx: cursor % TASK_TITLES.length,
      assigneeIdx,
      status: STATUSES[cursor % STATUSES.length]!,
      priority: PRIORITIES[cursor % PRIORITIES.length]!,
      dueDate: `2026-${pad(1 + (cursor % 12), 2)}-${pad(1 + (cursor % 27), 2)}`,
      completedAt: STATUSES[cursor % STATUSES.length] === 'completed' ? daysAgoIso(5) : '',
    })
  }

  // ---- deliberate monsters appended last (deterministic, easy to find) ----
  addTask({ projectId: 'proj-1', titleIdx: 10, assigneeIdx: null, status: 'todo', priority: 'medium', dueDate: '' }) // no assignee, no due date
  addTask({ projectId: '', titleIdx: 0, assigneeIdx: 0, status: 'in_progress', priority: 'high', dueDate: '' }) // no project, no due date
  // Overdue 400+ days (relative to "today"): still open, exercises the
  // overdue-tone column without a due-date-format crash.
  addTask({ projectId: 'proj-3', titleIdx: 3, assigneeIdx: 0, status: 'in_progress', priority: 'urgent', dueDate: daysAgoIso(430) })
  // Blocked with an EMPTY blocked_reason (legacy row predating the required-reason UI).
  addTask({ projectId: 'proj-5', titleIdx: 8, assigneeIdx: 16, status: 'blocked', priority: 'high', dueDate: daysAgoIso(20), blockedReason: '' })
  // Unrecognized status — must render (neutral badge), never crash.
  addTask({ projectId: 'proj-6', titleIdx: 5, assigneeIdx: 9, status: 'UNKNOWN_STATUS', priority: 'medium', dueDate: '2026-09-01' })
  // Unbroken 200-char title.
  taskSeq += 1
  tasks.push({
    id: `task-${taskSeq}`,
    title: MONSTER_TASK_TITLE,
    description: 'Monster-title task — unbroken 200-char token.',
    taskType: 'action',
    status: 'todo',
    blockedReason: '',
    priority: 'high',
    dueDate: '2026-08-01',
    customerId: '',
    opportunityId: '',
    orderId: '',
    projectId: 'proj-mega',
    creatorEmployeeId: empId(0),
    assigneeEmployeeId: empId(6), // the 200-char monster employee name
    creatorName: empName(0),
    assigneeName: empName(6),
    startedAt: '',
    completedAt: '',
    lastCommentAt: '',
  })

  return { employees, projects, members, tasks, comments, activities }
}

function findTaskOrThrow(id: string): TaskItem {
  cache ??= generate()
  const task = cache.tasks.find((t) => t.id === id)
  if (!task) throw new Error(`Task ${id} not found`)
  return task
}

function findProjectOrThrow(id: string): Project {
  cache ??= generate()
  const project = cache.projects.find((p) => p.id === id)
  if (!project) throw new Error(`Project ${id} not found`)
  return project
}

const TERMINAL_PROJECT_STATUSES = new Set(['archived', 'shelved', 'deleted'])

async function mockFetchMyTasks(includeDone: boolean): Promise<TaskItem[]> {
  cache ??= generate()
  await sleep(200)
  const me = cache.employees[0]!.id
  return cache.tasks
    .filter((t) => t.assigneeEmployeeId === me)
    .filter((t) => includeDone || t.status !== 'completed')
    .map((t) => ({ ...t }))
}

async function mockFetchTeamTasks(includeDone: boolean): Promise<TaskItem[]> {
  cache ??= generate()
  await sleep(220)
  return cache.tasks.filter((t) => includeDone || t.status !== 'completed').map((t) => ({ ...t }))
}

async function mockFetchProjects(activeOnly: boolean): Promise<Project[]> {
  cache ??= generate()
  await sleep(160)
  const rows = activeOnly ? cache.projects.filter((p) => !TERMINAL_PROJECT_STATUSES.has(p.status)) : cache.projects
  return rows.map((p) => ({ ...p }))
}

async function mockFetchProjectMembers(projectId: string): Promise<ProjectMember[]> {
  cache ??= generate()
  await sleep(140)
  return cache.members.filter((m) => m.projectId === projectId && m.isActive).map((m) => ({ ...m }))
}

async function mockFetchProjectTasks(projectId: string, includeDone: boolean): Promise<TaskItem[]> {
  cache ??= generate()
  await sleep(180)
  return cache.tasks
    .filter((t) => t.projectId === projectId)
    .filter((t) => includeDone || t.status !== 'completed')
    .map((t) => ({ ...t }))
}

async function mockFetchProjectActivity(projectId: string): Promise<TaskActivity[]> {
  cache ??= generate()
  await sleep(150)
  const taskIds = new Set(cache.tasks.filter((t) => t.projectId === projectId).map((t) => t.id))
  return cache.activities.filter((a) => taskIds.has(a.taskId)).map((a) => ({ ...a }))
}

async function mockFetchTask(id: string): Promise<TaskItem> {
  await sleep(120)
  return { ...findTaskOrThrow(id) }
}

async function mockFetchTaskComments(taskId: string): Promise<TaskComment[]> {
  cache ??= generate()
  await sleep(120)
  return cache.comments.filter((c) => c.taskId === taskId).map((c) => ({ ...c }))
}

async function mockFetchTaskActivity(taskId: string): Promise<TaskActivity[]> {
  cache ??= generate()
  await sleep(120)
  return cache.activities.filter((a) => a.taskId === taskId).map((a) => ({ ...a }))
}

async function mockFetchProjectTaskCounts(): Promise<Record<string, number>> {
  cache ??= generate()
  await sleep(100)
  const counts: Record<string, number> = {}
  for (const t of cache.tasks) {
    if (!t.projectId) continue
    counts[t.projectId] = (counts[t.projectId] ?? 0) + 1
  }
  return counts
}

async function mockFetchEmployees(activeOnly: boolean): Promise<EmployeeOption[]> {
  cache ??= generate()
  await sleep(140)
  return cache.employees.filter((e) => !activeOnly || e.isActive).map((e) => ({ ...e }))
}

async function mockFetchCurrentEmployee(): Promise<CurrentEmployeeContext> {
  cache ??= generate()
  await sleep(100)
  const me = cache.employees[0]!
  return { employeeId: me.id, employeeName: me.name, licenseRole: 'manager', resolvedBy: 'license' }
}

async function mockFetchAllocationSummary(employeeId: string, excludeProjectId: string): Promise<AllocationSummary> {
  cache ??= generate()
  await sleep(140)
  const lines = cache.members
    .filter((m) => m.employeeId === employeeId && m.isActive && m.projectId !== excludeProjectId)
    .map((m) => ({ projectId: m.projectId, projectName: m.projectName, allocationPercent: m.allocationPercent }))
  const otherProjectsTotal = lines.reduce((sum, l) => sum + l.allocationPercent, 0)
  return { employeeId, otherProjectsTotal, projects: lines }
}

async function mockCreateProject(draft: ProjectDraft): Promise<Project> {
  cache ??= generate()
  await sleep(180)
  projectSeq += 1
  const created: Project = {
    id: `proj-new-${projectSeq}`,
    name: draft.name,
    projectType: draft.projectType || 'internal',
    description: draft.description,
    status: 'active',
    customerId: '',
    opportunityId: '',
    orderId: '',
    customerName: draft.projectType === 'customer' ? draft.customerName : '',
    endUserName: '',
    opportunityKey: '',
    customerPocName: draft.projectType === 'customer' ? draft.customerPocName : '',
    customerPocEmail: draft.projectType === 'customer' ? draft.customerPocEmail : '',
    customerPocPhone: draft.projectType === 'customer' ? draft.customerPocPhone : '',
    startsOn: todayIso(),
    endsOn: '',
  }
  cache.projects.unshift(created)
  return { ...created }
}

async function mockUpdateProject(id: string, patch: ProjectPatch): Promise<Project> {
  cache ??= generate()
  await sleep(160)
  const project = findProjectOrThrow(id)
  Object.assign(project, patch)
  return { ...project }
}

async function mockArchiveProject(id: string, reason: string): Promise<Project> {
  void reason // captured for the (mocked) audit trail, not replayed here
  cache ??= generate()
  await sleep(180)
  const project = findProjectOrThrow(id)
  project.status = 'archived'
  return { ...project }
}

async function mockShelveProject(id: string, reason: string): Promise<Project> {
  void reason
  cache ??= generate()
  await sleep(180)
  const project = findProjectOrThrow(id)
  project.status = 'shelved'
  return { ...project }
}

async function mockDeleteProject(id: string, reason: string): Promise<void> {
  void reason
  cache ??= generate()
  await sleep(180)
  const project = findProjectOrThrow(id)
  project.status = 'deleted'
}

async function mockAddProjectMember(projectId: string, employeeId: string, role: string, allocationPercent: number): Promise<ProjectMember> {
  cache ??= generate()
  await sleep(160)
  const project = findProjectOrThrow(projectId)
  const employee = cache.employees.find((e) => e.id === employeeId)
  // AddCollaborativeProjectMember upserts on (project_id, employee_id) —
  // re-calling it updates that member's role/allocation in place.
  const existing = cache.members.find((m) => m.projectId === projectId && m.employeeId === employeeId)
  if (existing) {
    existing.role = role
    existing.allocationPercent = allocationPercent
    existing.isActive = true
    return { ...existing }
  }
  memberSeq += 1
  const created: ProjectMember = {
    id: `member-new-${memberSeq}`,
    projectId,
    employeeId,
    employeeName: employee?.name ?? '',
    projectName: project.name,
    role,
    allocationPercent,
    isActive: true,
    joinedAt: todayIso(),
    leftAt: '',
  }
  cache.members.push(created)
  return { ...created }
}

async function mockCreateTask(draft: TaskDraft): Promise<TaskItem> {
  cache ??= generate()
  await sleep(160)
  taskSeq += 1
  const assignee = cache.employees.find((e) => e.id === draft.assigneeEmployeeId)
  const created: TaskItem = {
    id: `task-new-${taskSeq}`,
    title: draft.title,
    description: draft.description,
    taskType: 'action',
    status: 'todo',
    blockedReason: '',
    priority: draft.priority || 'medium',
    dueDate: draft.dueDate,
    customerId: '',
    opportunityId: '',
    orderId: '',
    projectId: draft.projectId,
    creatorEmployeeId: cache.employees[0]!.id,
    assigneeEmployeeId: draft.assigneeEmployeeId,
    creatorName: cache.employees[0]!.name,
    assigneeName: assignee?.name ?? '',
    startedAt: '',
    completedAt: '',
    lastCommentAt: '',
  }
  cache.tasks.unshift(created)
  return { ...created }
}

async function mockUpdateTask(task: Partial<TaskItem> & { id: string }): Promise<TaskItem> {
  cache ??= generate()
  await sleep(160)
  const existing = findTaskOrThrow(task.id)
  Object.assign(existing, task)
  return { ...existing }
}

async function mockUpdateTaskStatus(id: string, status: string, note: string): Promise<void> {
  cache ??= generate()
  await sleep(140)
  const task = findTaskOrThrow(id)
  task.status = status
  if (status === 'blocked') task.blockedReason = note
  if (status === 'completed') task.completedAt = todayIso()
}

async function mockReassignTask(id: string, assigneeEmployeeId: string): Promise<void> {
  cache ??= generate()
  await sleep(140)
  const task = findTaskOrThrow(id)
  const assignee = cache.employees.find((e) => e.id === assigneeEmployeeId)
  task.assigneeEmployeeId = assigneeEmployeeId
  task.assigneeName = assignee?.name ?? ''
}

async function mockUpdateTaskDueDate(id: string, dueDateIso: string): Promise<void> {
  cache ??= generate()
  await sleep(120)
  const task = findTaskOrThrow(id)
  task.dueDate = dueDateIso
}

async function mockAddTaskComment(taskId: string, body: string): Promise<TaskComment> {
  cache ??= generate()
  await sleep(140)
  const task = findTaskOrThrow(taskId)
  commentSeq += 1
  const created: TaskComment = {
    id: `comment-new-${commentSeq}`,
    taskId,
    employeeId: cache.employees[0]!.id,
    employeeName: cache.employees[0]!.name,
    body,
    createdAt: todayIso(),
  }
  cache.comments.push(created)
  task.lastCommentAt = created.createdAt
  return { ...created }
}

async function mockDeleteTask(id: string): Promise<void> {
  cache ??= generate()
  await sleep(140)
  cache.tasks = cache.tasks.filter((t) => t.id !== id)
}

async function mockRefreshWorkspace(): Promise<void> {
  await sleep(80)
}

/* ---- real: FETCH and MUTATION are both wired (R3) — project delete/
 * archive/shelve carry a mandatory audit reason (financial/PII-adjacent hot
 * zone), task delete is irreversible. ---- */

function mapEmployee(e: Record<string, unknown>): EmployeeOption {
  return {
    id: str(e.id),
    name: str(e.full_name),
    email: str(e.email),
    department: str(e.department),
    jobTitle: str(e.job_title),
    isActive: Boolean(e.is_active),
  }
}

function mapCurrentEmployee(c: Record<string, unknown>): CurrentEmployeeContext {
  return {
    employeeId: str(c.employee_id),
    employeeName: str(c.employee_name),
    licenseRole: str(c.license_role),
    resolvedBy: str(c.resolved_by),
  }
}

function mapProject(p: Record<string, unknown>): Project {
  return {
    id: str(p.id),
    name: str(p.name),
    projectType: str(p.project_type),
    description: str(p.description),
    status: str(p.status) || 'active',
    customerId: str(p.customer_id),
    opportunityId: str(p.opportunity_id),
    orderId: str(p.order_id),
    customerName: str(p.customer_name),
    endUserName: str(p.end_user_name),
    opportunityKey: str(p.opportunity_key),
    customerPocName: str(p.customer_poc_name),
    customerPocEmail: str(p.customer_poc_email),
    customerPocPhone: str(p.customer_poc_phone),
    startsOn: goDate(p.starts_on),
    endsOn: goDate(p.ends_on),
  }
}

function mapMember(m: Record<string, unknown>): ProjectMember {
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

function mapTask(t: Record<string, unknown>): TaskItem {
  return {
    id: str(t.id),
    title: str(t.title),
    description: str(t.description),
    taskType: str(t.task_type),
    status: str(t.status) || 'todo',
    blockedReason: str(t.blocked_reason),
    priority: str(t.priority) || 'medium',
    dueDate: goDate(t.due_date),
    customerId: str(t.customer_id),
    opportunityId: str(t.opportunity_id),
    orderId: str(t.order_id),
    projectId: str(t.project_id),
    creatorEmployeeId: str(t.creator_employee_id),
    assigneeEmployeeId: str(t.assignee_employee_id),
    creatorName: str(t.creator_name),
    assigneeName: str(t.assignee_name),
    startedAt: goDate(t.started_at),
    completedAt: goDate(t.completed_at),
    lastCommentAt: goDate(t.last_comment_at),
  }
}

function mapComment(c: Record<string, unknown>): TaskComment {
  return {
    id: str(c.id),
    taskId: str(c.task_id),
    employeeId: str(c.employee_id),
    employeeName: str(c.employee_name),
    body: str(c.body),
    createdAt: goDate(c.created_at),
  }
}

function mapActivity(a: Record<string, unknown>): TaskActivity {
  return {
    id: str(a.id),
    taskId: str(a.task_id),
    employeeId: str(a.employee_id),
    employeeName: str(a.employee_name),
    activityType: str(a.activity_type),
    detail: str(a.detail),
    metadataJson: str(a.metadata_json),
    createdAt: goDate(a.created_at),
  }
}

function mapAllocationSummary(a: Record<string, unknown>): AllocationSummary {
  const lines = ((a.projects as unknown as Record<string, unknown>[] | undefined) ?? []).map((l) => ({
    projectId: str(l.project_id),
    projectName: str(l.project_name),
    allocationPercent: num(l.allocation_percent),
  }))
  return { employeeId: str(a.employee_id), otherProjectsTotal: num(a.other_projects_total), projects: lines }
}

async function realFetchMyTasks(includeDone: boolean): Promise<TaskItem[]> {
  const rows = await ListMyCollaborativeTasks(includeDone)
  return (rows ?? []).map((r) => mapTask(r as unknown as Record<string, unknown>))
}

async function realFetchTeamTasks(includeDone: boolean): Promise<TaskItem[]> {
  const rows = await ListCollaborativeTeamTasks(includeDone)
  return (rows ?? []).map((r) => mapTask(r as unknown as Record<string, unknown>))
}

async function realFetchProjects(activeOnly: boolean): Promise<Project[]> {
  const rows = await ListCollaborativeProjects(activeOnly)
  return (rows ?? []).map((r) => mapProject(r as unknown as Record<string, unknown>))
}

async function realFetchProjectMembers(projectId: string): Promise<ProjectMember[]> {
  const rows = await ListCollaborativeProjectMembers(projectId)
  return (rows ?? []).map((r) => mapMember(r as unknown as Record<string, unknown>))
}

async function realFetchProjectTasks(projectId: string, includeDone: boolean): Promise<TaskItem[]> {
  const rows = await ListCollaborativeProjectTasks(projectId, includeDone)
  return (rows ?? []).map((r) => mapTask(r as unknown as Record<string, unknown>))
}

async function realFetchProjectActivity(projectId: string): Promise<TaskActivity[]> {
  const rows = await ListCollaborativeProjectActivity(projectId)
  return (rows ?? []).map((r) => mapActivity(r as unknown as Record<string, unknown>))
}

async function realFetchTask(id: string): Promise<TaskItem> {
  const t = await GetCollaborativeTask(id)
  return mapTask(t as unknown as Record<string, unknown>)
}

async function realFetchTaskComments(taskId: string): Promise<TaskComment[]> {
  const rows = await ListCollaborativeTaskComments(taskId)
  return (rows ?? []).map((r) => mapComment(r as unknown as Record<string, unknown>))
}

async function realFetchTaskActivity(taskId: string): Promise<TaskActivity[]> {
  const rows = await ListCollaborativeTaskActivity(taskId)
  return (rows ?? []).map((r) => mapActivity(r as unknown as Record<string, unknown>))
}

async function realFetchProjectTaskCounts(): Promise<Record<string, number>> {
  return (await GetProjectTaskCounts()) ?? {}
}

async function realFetchEmployees(activeOnly: boolean): Promise<EmployeeOption[]> {
  const rows = await ListEmployeeProfiles(activeOnly)
  return (rows ?? []).map((r) => mapEmployee(r as unknown as Record<string, unknown>))
}

async function realFetchCurrentEmployee(): Promise<CurrentEmployeeContext> {
  const c = await GetCurrentEmployeeContext()
  return mapCurrentEmployee(c as unknown as Record<string, unknown>)
}

async function realFetchAllocationSummary(employeeId: string, excludeProjectId: string): Promise<AllocationSummary> {
  const a = await GetEmployeeAllocationSummary(employeeId, excludeProjectId)
  return mapAllocationSummary(a as unknown as Record<string, unknown>)
}

/** UpdateCollaborativeTaskDueDate's arg2 is a raw string, not a time.Time —
 * Go treats '' as "clear the due date" (nil) and anything else must be
 * strict RFC3339. goTime() doesn't fit here: its empty-input branch emits
 * Go's ZERO time literal ('0001-01-01T00:00:00Z'), which Go would happily
 * time.Parse as a REAL date and write, instead of clearing — clear and
 * zero-date are different outcomes for this specific binding, so empty must
 * stay empty. */
function dueDateArg(dueDateStr: string): string {
  const s = (dueDateStr ?? '').trim()
  if (!s) return ''
  return s.includes('T') ? s : `${s}T00:00:00Z`
}

async function realCreateProject(draft: ProjectDraft): Promise<Project> {
  // CreateCollaborativeProject(main.Project) → main.Project. Only the
  // user-authored fields are sent; the server assigns id/timestamps and
  // defaults project_type/status when blank (a.getCurrentUserID() stamps
  // created_by — no actor arg on this binding).
  const arg = {
    name: draft.name,
    project_type: draft.projectType,
    description: draft.description,
    customer_name: draft.customerName,
    customer_poc_name: draft.customerPocName,
    customer_poc_email: draft.customerPocEmail,
    customer_poc_phone: draft.customerPocPhone,
  } as unknown as main.Project
  const created = await CreateCollaborativeProject(arg)
  return mapProject(created as unknown as Record<string, unknown>)
}
async function realUpdateProject(id: string, patch: ProjectPatch): Promise<Project> {
  // UpdateCollaborativeProject(id, Record<string,any>) — server whitelists the
  // key set (name/project_type/description/status/customer_*/starts_on/
  // ends_on) and escalates to projects:delete when status enters a terminal
  // state; only the keys the caller actually supplied are sent.
  const updates: Record<string, unknown> = {}
  if (patch.name !== undefined) updates.name = patch.name
  if (patch.projectType !== undefined) updates.project_type = patch.projectType
  if (patch.description !== undefined) updates.description = patch.description
  if (patch.customerName !== undefined) updates.customer_name = patch.customerName
  if (patch.customerPocName !== undefined) updates.customer_poc_name = patch.customerPocName
  if (patch.customerPocEmail !== undefined) updates.customer_poc_email = patch.customerPocEmail
  if (patch.customerPocPhone !== undefined) updates.customer_poc_phone = patch.customerPocPhone
  if (patch.status !== undefined) updates.status = patch.status
  const updated = await UpdateCollaborativeProject(id, updates)
  return mapProject(updated as unknown as Record<string, unknown>)
}
async function realArchiveProject(id: string, reason: string): Promise<Project> {
  // ArchiveCollaborativeProject(id, reason) — HOT-ZONE: flips status to
  // 'archived' and writes `reason` to the (server-side) audit log.
  const archived = await ArchiveCollaborativeProject(id, reason)
  return mapProject(archived as unknown as Record<string, unknown>)
}
async function realShelveProject(id: string, reason: string): Promise<Project> {
  // ShelveCollaborativeProject(id, reason) — HOT-ZONE, same shape as archive.
  const shelved = await ShelveCollaborativeProject(id, reason)
  return mapProject(shelved as unknown as Record<string, unknown>)
}
async function realDeleteProject(id: string, reason: string): Promise<void> {
  // DeleteCollaborativeProject(id, reason) — HOT-ZONE, irreversible: flips
  // status to 'deleted' and writes `reason` to the audit log.
  await DeleteCollaborativeProject(id, reason)
}
async function realAddProjectMember(projectId: string, employeeId: string, role: string, allocationPercent: number): Promise<ProjectMember> {
  // AddCollaborativeProjectMember upserts on (project_id, employee_id) —
  // same semantics as the mock's upsert-on-re-add.
  const member = await AddCollaborativeProjectMember(projectId, employeeId, role, allocationPercent)
  return mapMember(member as unknown as Record<string, unknown>)
}
async function realCreateTask(draft: TaskDraft): Promise<TaskItem> {
  // CreateCollaborativeTask(main.TaskItem) → main.TaskItem. Creator/status
  // default server-side (creator from GetCurrentEmployeeContext, status
  // 'open'). due_date/project_id/assignee_employee_id are OMITTED (not '')
  // when blank — they're pointer fields server-side, and an empty-string
  // due_date fails Go's time.Time JSON unmarshal outright.
  const arg: Record<string, unknown> = {
    title: draft.title,
    description: draft.description,
    priority: draft.priority,
  }
  if (draft.projectId) arg.project_id = draft.projectId
  if (draft.assigneeEmployeeId) arg.assignee_employee_id = draft.assigneeEmployeeId
  if (draft.dueDate) arg.due_date = goTime(draft.dueDate)
  const created = await CreateCollaborativeTask(arg as unknown as main.TaskItem)
  return mapTask(created as unknown as Record<string, unknown>)
}
async function realUpdateTask(task: Partial<TaskItem> & { id: string }): Promise<TaskItem> {
  // UpdateCollaborativeTask(main.TaskItem) — the server only applies
  // title/description/priority/task_type/project_id from the payload (status,
  // due date, and assignee route through their own dedicated bindings below);
  // title is required non-empty server-side, matching the caller
  // (work-vm.svelte.ts saveTaskDetails) which always sends the full set.
  const arg = {
    id: task.id,
    title: task.title ?? '',
    description: task.description ?? '',
    priority: task.priority ?? '',
    task_type: task.taskType ?? '',
    project_id: task.projectId || undefined,
  } as unknown as main.TaskItem
  const updated = await UpdateCollaborativeTask(arg)
  return mapTask(updated as unknown as Record<string, unknown>)
}
async function realUpdateTaskStatus(id: string, status: string, note: string): Promise<void> {
  // UpdateCollaborativeTaskStatus(id, status, note) — server requires a
  // non-empty note when status === 'blocked'; surfaces honestly otherwise.
  await UpdateCollaborativeTaskStatus(id, status, note)
}
async function realReassignTask(id: string, assigneeEmployeeId: string): Promise<void> {
  // ReassignCollaborativeTask(id, assigneeEmployeeId) — '' clears the
  // assignee server-side (assigneeEmployeeID == "" ⇒ nil).
  await ReassignCollaborativeTask(id, assigneeEmployeeId)
}
async function realUpdateTaskDueDate(id: string, dueDateIso: string): Promise<void> {
  await UpdateCollaborativeTaskDueDate(id, dueDateArg(dueDateIso))
}
async function realAddTaskComment(taskId: string, body: string): Promise<TaskComment> {
  const created = await AddCollaborativeTaskComment(taskId, body)
  return mapComment(created as unknown as Record<string, unknown>)
}
async function realDeleteTask(id: string): Promise<void> {
  // DeleteCollaborativeTask(id) — HOT-ZONE, irreversible. No reason param on
  // this binding (unlike the project deletes); the server's own
  // guardDeleteOrRequest + audit logging cover it.
  await DeleteCollaborativeTask(id)
}
async function realRefreshWorkspace(): Promise<void> {
  await RefreshCollaborativeWorkspace()
}

/* ---- public switched API (viewmodel imports THESE) ---- */

export const fetchMyTasks = (includeDone = false): Promise<TaskItem[]> => pick(realFetchMyTasks, mockFetchMyTasks)(includeDone)
export const fetchTeamTasks = (includeDone = false): Promise<TaskItem[]> => pick(realFetchTeamTasks, mockFetchTeamTasks)(includeDone)
export const fetchProjects = (activeOnly = true): Promise<Project[]> => pick(realFetchProjects, mockFetchProjects)(activeOnly)
export const fetchProjectMembers = (projectId: string): Promise<ProjectMember[]> => pick(realFetchProjectMembers, mockFetchProjectMembers)(projectId)
export const fetchProjectTasks = (projectId: string, includeDone = true): Promise<TaskItem[]> =>
  pick(realFetchProjectTasks, mockFetchProjectTasks)(projectId, includeDone)
export const fetchProjectActivity = (projectId: string): Promise<TaskActivity[]> => pick(realFetchProjectActivity, mockFetchProjectActivity)(projectId)
export const fetchTask = (id: string): Promise<TaskItem> => pick(realFetchTask, mockFetchTask)(id)
export const fetchTaskComments = (taskId: string): Promise<TaskComment[]> => pick(realFetchTaskComments, mockFetchTaskComments)(taskId)
export const fetchTaskActivity = (taskId: string): Promise<TaskActivity[]> => pick(realFetchTaskActivity, mockFetchTaskActivity)(taskId)
export const fetchProjectTaskCounts = (): Promise<Record<string, number>> => pick(realFetchProjectTaskCounts, mockFetchProjectTaskCounts)()
export const fetchEmployees = (activeOnly = false): Promise<EmployeeOption[]> => pick(realFetchEmployees, mockFetchEmployees)(activeOnly)
export const fetchCurrentEmployee = (): Promise<CurrentEmployeeContext> => pick(realFetchCurrentEmployee, mockFetchCurrentEmployee)()
export const fetchAllocationSummary = (employeeId: string, excludeProjectId = ''): Promise<AllocationSummary> =>
  pick(realFetchAllocationSummary, mockFetchAllocationSummary)(employeeId, excludeProjectId)

export const createProject = (draft: ProjectDraft): Promise<Project> => pick(realCreateProject, mockCreateProject)(draft)
export const updateProject = (id: string, patch: ProjectPatch): Promise<Project> => pick(realUpdateProject, mockUpdateProject)(id, patch)
export const archiveProject = (id: string, reason: string): Promise<Project> => pick(realArchiveProject, mockArchiveProject)(id, reason)
export const shelveProject = (id: string, reason: string): Promise<Project> => pick(realShelveProject, mockShelveProject)(id, reason)
export const deleteProject = (id: string, reason: string): Promise<void> => pick(realDeleteProject, mockDeleteProject)(id, reason)
export const addProjectMember = (projectId: string, employeeId: string, role: string, allocationPercent: number): Promise<ProjectMember> =>
  pick(realAddProjectMember, mockAddProjectMember)(projectId, employeeId, role, allocationPercent)

export const createTask = (draft: TaskDraft): Promise<TaskItem> => pick(realCreateTask, mockCreateTask)(draft)
export const updateTask = (task: Partial<TaskItem> & { id: string }): Promise<TaskItem> => pick(realUpdateTask, mockUpdateTask)(task)
export const updateTaskStatus = (id: string, status: string, note = ''): Promise<void> => pick(realUpdateTaskStatus, mockUpdateTaskStatus)(id, status, note)
export const reassignTask = (id: string, assigneeEmployeeId: string): Promise<void> => pick(realReassignTask, mockReassignTask)(id, assigneeEmployeeId)
export const updateTaskDueDate = (id: string, dueDateIso: string): Promise<void> => pick(realUpdateTaskDueDate, mockUpdateTaskDueDate)(id, dueDateIso)
export const addTaskComment = (taskId: string, body: string): Promise<TaskComment> => pick(realAddTaskComment, mockAddTaskComment)(taskId, body)
export const deleteTask = (id: string): Promise<void> => pick(realDeleteTask, mockDeleteTask)(id)
export const refreshWorkspace = (): Promise<void> => pick(realRefreshWorkspace, mockRefreshWorkspace)()
