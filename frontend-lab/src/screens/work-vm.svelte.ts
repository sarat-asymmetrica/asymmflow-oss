/* WorkHub viewmodel — L5's reactive half for the tasks/projects operational
 * hub (TabShell: My Work / Team Board / Projects / Approvals). Owns: the
 * workspace load (current employee + my/team tasks + projects + employees +
 * task counts), My Work / Team Board search+filter state, the task composer,
 * the task-detail modal (edit/reassign/due-date/status/comment/delete — task
 * delete UNIFIED onto a ConfirmDialog, replacing the old screen's
 * press-again-to-confirm toggle per Design Constitution III.6), and the
 * Projects master-detail (create/edit, member roster with the Wave-9.8-B3
 * allocation-capacity precheck-before-batch, and archive/shelve/delete which
 * carry a MANDATORY audit reason). No layout — the screen renders this state
 * on primitives only. Snapshot/cache (30s TTL) and the cross-screen
 * "Start project" handoff are DEFERRED (see WorkHub.parity.md). */

import type { Tone } from '$kernel/tones'
import {
  addProjectMember,
  addTaskComment,
  archiveProject,
  createProject,
  createTask,
  deleteProject,
  deleteTask,
  fetchAllocationSummary,
  fetchCurrentEmployee,
  fetchEmployees,
  fetchMyTasks,
  fetchProjectActivity,
  fetchProjectMembers,
  fetchProjectTaskCounts,
  fetchProjectTasks,
  fetchProjects,
  fetchTask,
  fetchTaskActivity,
  fetchTaskComments,
  fetchTeamTasks,
  reassignTask,
  shelveProject,
  updateProject,
  updateTask,
  updateTaskDueDate,
  updateTaskStatus,
  type CurrentEmployeeContext,
  type EmployeeOption,
  type Project,
  type ProjectMember,
  type TaskActivity,
  type TaskComment,
  type TaskItem,
} from '../bridge/work'

export type WorkTab = 'my_work' | 'team_board' | 'projects' | 'approvals'

export const TASK_STATUS_TONES: Record<string, Tone> = {
  todo: 'neutral',
  in_progress: 'info',
  blocked: 'danger',
  completed: 'success',
}

export const PROJECT_STATUS_TONES: Record<string, Tone> = {
  active: 'success',
  shelved: 'warning',
  archived: 'neutral',
  deleted: 'danger',
}

const DAY_MS = 24 * 60 * 60 * 1000

export function isTaskOverdue(task: TaskItem): boolean {
  if (!task.dueDate || task.status === 'completed') return false
  const due = new Date(task.dueDate)
  if (Number.isNaN(due.getTime())) return false
  return due.getTime() < Date.now()
}

/** Danger once overdue, warning inside a 3-day horizon, neutral otherwise
 * (including tasks with no due date at all — never a false-danger read). */
export function taskDueTone(task: TaskItem): Tone {
  if (!task.dueDate || task.status === 'completed') return 'neutral'
  const due = new Date(task.dueDate)
  if (Number.isNaN(due.getTime())) return 'neutral'
  const days = (due.getTime() - Date.now()) / DAY_MS
  if (days < 0) return 'danger'
  if (days <= 3) return 'warning'
  return 'neutral'
}

function taskMatches(task: TaskItem, search: string): boolean {
  if (!search.trim()) return true
  const needle = search.trim().toLowerCase()
  return `${task.title} ${task.assigneeName} ${task.description}`.toLowerCase().includes(needle)
}

interface ProjectDraftState {
  name: string
  projectType: string
  description: string
  customerName: string
  customerPocName: string
  customerPocEmail: string
  customerPocPhone: string
}

function emptyProjectDraft(): ProjectDraftState {
  return { name: '', projectType: 'internal', description: '', customerName: '', customerPocName: '', customerPocEmail: '', customerPocPhone: '' }
}

const TERMINAL_PROJECT_STATUSES = new Set(['archived', 'shelved', 'deleted'])

export class WorkViewModel {
  loading = $state(true)
  error = $state<string | null>(null)
  activeTab = $state<WorkTab>('my_work')

  currentEmployee = $state<CurrentEmployeeContext | null>(null)
  myTasks = $state<TaskItem[]>([])
  teamTasks = $state<TaskItem[]>([])
  projects = $state<Project[]>([])
  employees = $state<EmployeeOption[]>([])
  projectTaskCounts = $state<Record<string, number>>({})

  // Mock permission flags (server-side RBAC is the real enforcer; every
  // mutation is INTEG-gapped regardless — see BUILD context "Division/
  // permission gating").
  canManageProjects = $state(true)
  canDeleteProject = $state(true)

  /* ---- My Work / Team Board filters ---- */
  myShowCompleted = $state(false)
  mySearch = $state('')
  teamShowCompleted = $state(false)
  teamSearch = $state('')
  teamStatusFilter = $state('')
  teamAssigneeFilter = $state('')

  visibleMyTasks = $derived(
    this.myTasks.filter((t) => (this.myShowCompleted || t.status !== 'completed') && taskMatches(t, this.mySearch)),
  )

  visibleTeamTasks = $derived(
    this.teamTasks
      .filter((t) => this.teamShowCompleted || t.status !== 'completed')
      .filter((t) => taskMatches(t, this.teamSearch))
      .filter((t) => !this.teamStatusFilter || t.status === this.teamStatusFilter)
      .filter((t) => {
        if (!this.teamAssigneeFilter) return true
        if (this.teamAssigneeFilter === 'unassigned') return !t.assigneeEmployeeId
        return t.assigneeEmployeeId === this.teamAssigneeFilter
      }),
  )

  teamStatusOptions = $derived(
    [...new Set(this.teamTasks.map((t) => t.status))].sort().map((s) => ({ value: s, label: s.replace(/_/g, ' ') })),
  )

  teamAssigneeOptions = $derived.by(() => {
    const ids = new Set(this.teamTasks.map((t) => t.assigneeEmployeeId).filter(Boolean))
    const opts = [...ids].map((id) => ({ value: id, label: this.employees.find((e) => e.id === id)?.name || id }))
    const hasUnassigned = this.teamTasks.some((t) => !t.assigneeEmployeeId)
    return hasUnassigned ? [...opts, { value: 'unassigned', label: 'Unassigned' }] : opts
  })

  /** Tasks-by-status distribution — Team Board's visual-diversity widget
   * (the old screen was a flat list with no aggregate view). */
  teamStatusDistribution = $derived.by(() => {
    const counts = new Map<string, number>()
    for (const t of this.visibleTeamTasks) counts.set(t.status, (counts.get(t.status) ?? 0) + 1)
    return [...counts.entries()].map(([status, value]) => ({
      key: status,
      label: status.replace(/_/g, ' ') || 'Unknown',
      value,
      tone: TASK_STATUS_TONES[status] ?? 'neutral',
    }))
  })

  myOpenCount = $derived(this.myTasks.filter((t) => t.status !== 'completed').length)
  teamOpenCount = $derived(this.teamTasks.filter((t) => t.status !== 'completed').length)
  teamBlockedCount = $derived(this.teamTasks.filter((t) => t.status === 'blocked').length)

  projectName(projectId: string): string {
    if (!projectId) return '—'
    return this.projects.find((p) => p.id === projectId)?.name || this.archivedProjects.find((p) => p.id === projectId)?.name || projectId
  }

  /* ---- task composer ---- */
  composerTitle = $state('')
  composerDescription = $state('')
  composerPriority = $state('medium')
  composerDueDate = $state('')
  composerProjectId = $state('')
  composerAssigneeId = $state('')
  creatingTask = $state(false)
  composerError = $state<string | null>(null)

  async submitComposer(): Promise<void> {
    if (!this.composerTitle.trim()) {
      this.composerError = 'Task title is required.'
      return
    }
    this.creatingTask = true
    this.composerError = null
    try {
      await createTask({
        title: this.composerTitle.trim(),
        description: this.composerDescription.trim(),
        priority: this.composerPriority,
        dueDate: this.composerDueDate,
        projectId: this.composerProjectId,
        assigneeEmployeeId: this.composerAssigneeId || this.currentEmployee?.employeeId || '',
      })
      this.composerTitle = ''
      this.composerDescription = ''
      this.composerPriority = 'medium'
      this.composerDueDate = ''
      this.composerAssigneeId = ''
      await this.refreshLists()
    } catch (e) {
      this.composerError = e instanceof Error ? e.message : String(e)
    } finally {
      this.creatingTask = false
    }
  }

  /* ---- task detail modal ---- */
  taskModalOpen = $state(false)
  selectedTaskId = $state('')
  selectedTask = $state<TaskItem | null>(null)
  taskComments = $state<TaskComment[]>([])
  taskActivity = $state<TaskActivity[]>([])
  taskDetailLoading = $state(false)
  taskDetailError = $state<string | null>(null)

  draftTitle = $state('')
  draftDescription = $state('')
  draftPriority = $state('medium')
  draftAssigneeId = $state('')
  draftDueDate = $state('')
  draftBlockedReason = $state('')

  savingTaskDetails = $state(false)
  savingAssignment = $state(false)
  savingDueDate = $state(false)
  savingComment = $state(false)
  deletingTask = $state(false)
  commentDraft = $state('')
  taskDeleteConfirmOpen = $state(false)

  async openTask(taskId: string): Promise<void> {
    this.selectedTaskId = taskId
    this.taskModalOpen = true
    this.taskDeleteConfirmOpen = false
    await this.loadTaskDetail()
  }

  async loadTaskDetail(): Promise<void> {
    if (!this.selectedTaskId) return
    this.taskDetailLoading = true
    this.taskDetailError = null
    try {
      const [task, comments, activity] = await Promise.all([
        fetchTask(this.selectedTaskId),
        fetchTaskComments(this.selectedTaskId),
        fetchTaskActivity(this.selectedTaskId),
      ])
      this.selectedTask = task
      this.taskComments = comments
      this.taskActivity = activity
      this.draftTitle = task.title
      this.draftDescription = task.description
      this.draftPriority = task.priority || 'medium'
      this.draftAssigneeId = task.assigneeEmployeeId
      this.draftDueDate = task.dueDate
      this.draftBlockedReason = task.blockedReason
    } catch (e) {
      this.taskDetailError = e instanceof Error ? e.message : String(e)
      this.selectedTask = null
    } finally {
      this.taskDetailLoading = false
    }
  }

  closeTaskModal(): void {
    this.taskModalOpen = false
    this.taskDeleteConfirmOpen = false
  }

  async saveTaskDetails(): Promise<void> {
    const task = this.selectedTask
    if (!task) return
    if (!this.draftTitle.trim()) {
      this.taskDetailError = 'Task title is required.'
      return
    }
    this.savingTaskDetails = true
    this.taskDetailError = null
    try {
      await updateTask({
        id: task.id,
        title: this.draftTitle.trim(),
        description: this.draftDescription.trim(),
        priority: this.draftPriority,
        taskType: task.taskType,
        projectId: task.projectId,
      })
      await this.loadTaskDetail()
      await this.refreshLists()
    } catch (e) {
      this.taskDetailError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingTaskDetails = false
    }
  }

  async reassign(): Promise<void> {
    const task = this.selectedTask
    if (!task) return
    this.savingAssignment = true
    this.taskDetailError = null
    try {
      await reassignTask(task.id, this.draftAssigneeId)
      await this.loadTaskDetail()
      await this.refreshLists()
    } catch (e) {
      this.taskDetailError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingAssignment = false
    }
  }

  async saveDueDate(): Promise<void> {
    const task = this.selectedTask
    if (!task) return
    this.savingDueDate = true
    this.taskDetailError = null
    try {
      await updateTaskDueDate(task.id, this.draftDueDate)
      await this.loadTaskDetail()
      await this.refreshLists()
    } catch (e) {
      this.taskDetailError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingDueDate = false
    }
  }

  async setStatus(status: string, note = ''): Promise<void> {
    const task = this.selectedTask
    if (!task) return
    this.taskDetailError = null
    try {
      await updateTaskStatus(task.id, status, note)
      await this.loadTaskDetail()
      await this.refreshLists()
    } catch (e) {
      this.taskDetailError = e instanceof Error ? e.message : String(e)
    }
  }

  async block(): Promise<void> {
    if (!this.draftBlockedReason.trim()) {
      this.taskDetailError = 'Add a blocked reason so the team knows what is stuck.'
      return
    }
    await this.setStatus('blocked', this.draftBlockedReason.trim())
  }

  async addComment(): Promise<void> {
    if (!this.selectedTaskId || !this.commentDraft.trim()) return
    this.savingComment = true
    this.taskDetailError = null
    try {
      await addTaskComment(this.selectedTaskId, this.commentDraft.trim())
      this.commentDraft = ''
      await this.loadTaskDetail()
    } catch (e) {
      this.taskDetailError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingComment = false
    }
  }

  /** UNIFIED onto ConfirmDialog (Design Constitution III.6) — the old screen
   * used an inconsistent "press Delete again" toggle; this matches project
   * delete's confirm-modal pattern. Irreversible. */
  requestDeleteTask(): void {
    this.taskDeleteConfirmOpen = true
  }

  cancelDeleteTask(): void {
    this.taskDeleteConfirmOpen = false
  }

  async confirmDeleteTask(): Promise<void> {
    const task = this.selectedTask
    if (!task) return
    this.deletingTask = true
    try {
      await deleteTask(task.id)
      this.closeTaskModal()
      this.selectedTask = null
      this.selectedTaskId = ''
      await this.refreshLists()
    } catch (e) {
      this.taskDetailError = e instanceof Error ? e.message : String(e)
      this.taskDeleteConfirmOpen = false
    } finally {
      this.deletingTask = false
    }
  }

  /* ---- projects tab: master list ---- */
  selectedProjectId = $state('')
  projectMembers = $state<ProjectMember[]>([])
  projectTasks = $state<TaskItem[]>([])
  projectActivity = $state<TaskActivity[]>([])
  projectContextLoading = $state(false)
  projectContextError = $state<string | null>(null)

  showArchivedProjects = $state(false)
  archivedProjects = $state<Project[]>([])
  loadingArchivedProjects = $state(false)

  visibleProjects = $derived(this.showArchivedProjects ? this.archivedProjects : this.projects)
  selectedProject = $derived(
    this.projects.find((p) => p.id === this.selectedProjectId)
    ?? this.archivedProjects.find((p) => p.id === this.selectedProjectId)
    ?? null,
  )

  projectStats = $derived({
    open: this.projectTasks.filter((t) => t.status !== 'completed').length,
    blocked: this.projectTasks.filter((t) => t.status === 'blocked').length,
    completed: this.projectTasks.filter((t) => t.status === 'completed').length,
    members: this.projectMembers.length,
  })

  availableEmployeesForMembers = $derived(this.employees.filter((e) => !this.projectMembers.some((m) => m.employeeId === e.id)))

  async selectProject(projectId: string): Promise<void> {
    this.selectedProjectId = projectId
    this.composerProjectId = projectId
    this.editingProject = false
    this.memberSelections = []
    if (!projectId) {
      this.projectMembers = []
      this.projectTasks = []
      this.projectActivity = []
      return
    }
    await this.loadProjectContext(projectId)
  }

  async loadProjectContext(projectId: string): Promise<void> {
    this.projectContextLoading = true
    this.projectContextError = null
    try {
      const [members, tasks, activity] = await Promise.all([
        fetchProjectMembers(projectId),
        fetchProjectTasks(projectId, true),
        fetchProjectActivity(projectId),
      ])
      this.projectMembers = members
      this.projectTasks = tasks
      this.projectActivity = activity
    } catch (e) {
      this.projectContextError = e instanceof Error ? e.message : String(e)
    } finally {
      this.projectContextLoading = false
    }
  }

  async toggleShowArchived(): Promise<void> {
    this.showArchivedProjects = !this.showArchivedProjects
    if (this.showArchivedProjects && this.archivedProjects.length === 0) {
      await this.loadArchivedProjects()
    }
  }

  async loadArchivedProjects(): Promise<void> {
    this.loadingArchivedProjects = true
    try {
      const rows = await fetchProjects(false)
      this.archivedProjects = rows.filter((p) => TERMINAL_PROJECT_STATUSES.has(String(p.status || '').toLowerCase()))
    } catch (e) {
      this.projectAdminError = e instanceof Error ? e.message : String(e)
    } finally {
      this.loadingArchivedProjects = false
    }
  }

  /* ---- project composer (create) ---- */
  projectDraft = $state<ProjectDraftState>(emptyProjectDraft())
  savingProject = $state(false)
  projectComposerError = $state<string | null>(null)

  async submitProjectComposer(): Promise<void> {
    if (!this.projectDraft.name.trim()) {
      this.projectComposerError = 'Project name is required.'
      return
    }
    this.savingProject = true
    this.projectComposerError = null
    try {
      const created = await createProject({ ...this.projectDraft, name: this.projectDraft.name.trim() })
      this.projectDraft = emptyProjectDraft()
      await this.load()
      await this.selectProject(created.id)
    } catch (e) {
      this.projectComposerError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingProject = false
    }
  }

  /* ---- project edit ---- */
  editingProject = $state(false)
  projectEditDraft = $state<ProjectDraftState>(emptyProjectDraft())
  savingProjectAdmin = $state(false)
  projectAdminError = $state<string | null>(null)

  startProjectEdit(): void {
    const p = this.selectedProject
    if (!p) return
    this.projectEditDraft = {
      name: p.name,
      projectType: p.projectType || 'internal',
      description: p.description,
      customerName: p.customerName,
      customerPocName: p.customerPocName,
      customerPocEmail: p.customerPocEmail,
      customerPocPhone: p.customerPocPhone,
    }
    this.projectAdminError = null
    this.editingProject = true
  }

  cancelProjectEdit(): void {
    this.editingProject = false
  }

  async saveProjectEdit(): Promise<void> {
    const p = this.selectedProject
    if (!p) return
    if (!this.projectEditDraft.name.trim()) {
      this.projectAdminError = 'Project name is required.'
      return
    }
    this.savingProjectAdmin = true
    this.projectAdminError = null
    try {
      const updated = await updateProject(p.id, { ...this.projectEditDraft, name: this.projectEditDraft.name.trim() })
      this.projects = this.projects.map((x) => (x.id === updated.id ? updated : x))
      this.editingProject = false
    } catch (e) {
      this.projectAdminError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingProjectAdmin = false
    }
  }

  /* ---- project archive / shelve / delete (HOT-ZONE — mandatory audit reason) ---- */
  projectAdminConfirm = $state<{ action: 'archive' | 'shelve' | 'delete' } | null>(null)

  requestProjectAdmin(action: 'archive' | 'shelve' | 'delete'): void {
    if (!this.selectedProject) return
    this.projectAdminError = null
    this.projectAdminConfirm = { action }
  }

  cancelProjectAdmin(): void {
    this.projectAdminConfirm = null
  }

  async confirmProjectAdmin(reason: string): Promise<void> {
    const p = this.selectedProject
    const action = this.projectAdminConfirm?.action
    if (!p || !action) return
    this.projectAdminConfirm = null
    this.savingProjectAdmin = true
    this.projectAdminError = null
    try {
      if (action === 'archive') await archiveProject(p.id, reason)
      else if (action === 'shelve') await shelveProject(p.id, reason)
      else await deleteProject(p.id, reason)
      await this.finishProjectAdminCleanup()
    } catch (e) {
      this.projectAdminError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingProjectAdmin = false
    }
  }

  private async finishProjectAdminCleanup(): Promise<void> {
    this.editingProject = false
    await this.load()
    const stillActive = this.projects.some((p) => p.id === this.selectedProjectId)
    if (!stillActive) {
      const next = this.projects[0]?.id ?? ''
      if (next) {
        await this.selectProject(next)
      } else {
        this.selectedProjectId = ''
        this.projectMembers = []
        this.projectTasks = []
        this.projectActivity = []
      }
    }
  }

  async restoreProject(): Promise<void> {
    const p = this.selectedProject
    if (!p) return
    this.savingProjectAdmin = true
    this.projectAdminError = null
    try {
      const updated = await updateProject(p.id, { status: 'active' })
      this.archivedProjects = this.archivedProjects.filter((x) => x.id !== updated.id)
      this.showArchivedProjects = false
      await this.load()
      await this.selectProject(updated.id)
    } catch (e) {
      this.projectAdminError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingProjectAdmin = false
    }
  }

  /* ---- member roster: batch add with allocation-capacity precheck-before-
   * batch (Wave 9.8 B3 pattern) — every selected employee is checked via
   * GetEmployeeAllocationSummary BEFORE any write; a cancel on the WARN
   * aborts with NOTHING added (all-or-nothing), never a partial batch. Over
   * 100% is a WARN, never a hard block — the server total is always what's
   * displayed, never a client-side re-derivation. ---- */
  memberSelections = $state<string[]>([])
  newMemberRole = $state('Member')
  newMemberAllocation = $state(100)
  savingMembers = $state(false)
  memberAddError = $state<string | null>(null)
  memberAddWarning = $state<string[] | null>(null)
  private pendingMemberAdd: { role: string; allocation: number; employeeIds: string[] } | null = null

  toggleMemberSelection(employeeId: string): void {
    this.memberSelections = this.memberSelections.includes(employeeId)
      ? this.memberSelections.filter((id) => id !== employeeId)
      : [...this.memberSelections, employeeId]
  }

  private clampAllocation(value: number): number {
    return Number.isFinite(value) && value > 0 ? Math.min(value, 1000) : 100
  }

  async requestAddMembers(): Promise<void> {
    if (!this.selectedProjectId) {
      this.memberAddError = 'Choose a project first.'
      return
    }
    if (this.memberSelections.length === 0) {
      this.memberAddError = 'Choose at least one employee to add.'
      return
    }
    const role = this.newMemberRole.trim() || 'Member'
    const allocation = this.clampAllocation(this.newMemberAllocation)
    this.savingMembers = true
    this.memberAddError = null
    try {
      const overAllocated: string[] = []
      for (const employeeId of this.memberSelections) {
        const summary = await fetchAllocationSummary(employeeId, this.selectedProjectId)
        const resultingTotal = summary.otherProjectsTotal + allocation
        if (resultingTotal > 100) {
          const name = this.employees.find((e) => e.id === employeeId)?.name || employeeId
          overAllocated.push(`${name} (would reach ${resultingTotal}%)`)
        }
      }
      if (overAllocated.length > 0) {
        this.pendingMemberAdd = { role, allocation, employeeIds: [...this.memberSelections] }
        this.memberAddWarning = overAllocated
        this.savingMembers = false
        return
      }
      await this.commitAddMembers(role, allocation, this.memberSelections)
    } catch (e) {
      this.memberAddError = e instanceof Error ? e.message : String(e)
      this.savingMembers = false
    }
  }

  /** Confirming the WARN proceeds with the exact batch that was precheck'd. */
  async confirmMemberAddOverAllocation(): Promise<void> {
    const pending = this.pendingMemberAdd
    this.pendingMemberAdd = null
    this.memberAddWarning = null
    if (!pending) return
    await this.commitAddMembers(pending.role, pending.allocation, pending.employeeIds)
  }

  /** Cancelling aborts cleanly — nothing was written. */
  cancelMemberAddOverAllocation(): void {
    this.pendingMemberAdd = null
    this.memberAddWarning = null
    this.savingMembers = false
  }

  private async commitAddMembers(role: string, allocation: number, employeeIds: string[]): Promise<void> {
    this.savingMembers = true
    try {
      await Promise.all(employeeIds.map((id) => addProjectMember(this.selectedProjectId, id, role, allocation)))
      this.memberSelections = []
      this.newMemberRole = 'Member'
      this.newMemberAllocation = 100
      await this.loadProjectContext(this.selectedProjectId)
    } catch (e) {
      this.memberAddError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingMembers = false
    }
  }

  /* ---- member roster: per-member role/allocation edit (select-a-row-to-edit,
   * mirrors Payroll's compensation-profile edit-via-select convention — avoids
   * rendering 50+ simultaneous inline edit forms for the perf-monster project). ---- */
  editingMemberId = $state('')
  memberEditDrafts = $state<Record<string, { role: string; allocation: number }>>({})
  savingMemberId = $state('')
  memberEditError = $state<string | null>(null)
  memberEditWarning = $state<string | null>(null)
  private pendingMemberEdit: { member: ProjectMember; role: string; allocation: number } | null = null

  beginEditMember(member: ProjectMember): void {
    this.editingMemberId = member.employeeId
    this.memberEditError = null
    this.memberEditWarning = null
  }

  cancelEditMember(): void {
    this.editingMemberId = ''
  }

  memberDraft(member: ProjectMember): { role: string; allocation: number } {
    return this.memberEditDrafts[member.employeeId] || { role: member.role || 'Member', allocation: member.allocationPercent ?? 100 }
  }

  setMemberDraftField(employeeId: string, field: 'role' | 'allocation', value: string | number): void {
    const current = this.memberEditDrafts[employeeId] || { role: 'Member', allocation: 100 }
    this.memberEditDrafts = { ...this.memberEditDrafts, [employeeId]: { ...current, [field]: value } }
  }

  async requestSaveMember(member: ProjectMember): Promise<void> {
    const draft = this.memberDraft(member)
    const role = String(draft.role || 'Member').trim() || 'Member'
    const allocation = this.clampAllocation(Number(draft.allocation))
    this.savingMemberId = member.employeeId
    this.memberEditError = null
    try {
      const summary = await fetchAllocationSummary(member.employeeId, this.selectedProjectId)
      const resultingTotal = summary.otherProjectsTotal + allocation
      if (resultingTotal > 100) {
        this.pendingMemberEdit = { member, role, allocation }
        this.memberEditWarning = `${member.employeeName || 'This person'} is already committed to ${summary.otherProjectsTotal}% across other active projects. Saving ${allocation}% here brings their total to ${resultingTotal}%.`
        this.savingMemberId = ''
        return
      }
      await this.commitSaveMember(member, role, allocation)
    } catch (e) {
      this.memberEditError = e instanceof Error ? e.message : String(e)
      this.savingMemberId = ''
    }
  }

  async confirmMemberEditOverAllocation(): Promise<void> {
    const pending = this.pendingMemberEdit
    this.pendingMemberEdit = null
    this.memberEditWarning = null
    if (!pending) return
    await this.commitSaveMember(pending.member, pending.role, pending.allocation)
  }

  cancelMemberEditOverAllocation(): void {
    this.pendingMemberEdit = null
    this.memberEditWarning = null
    this.savingMemberId = ''
  }

  private async commitSaveMember(member: ProjectMember, role: string, allocation: number): Promise<void> {
    this.savingMemberId = member.employeeId
    try {
      await addProjectMember(this.selectedProjectId, member.employeeId, role, allocation)
      const next = { ...this.memberEditDrafts }
      delete next[member.employeeId]
      this.memberEditDrafts = next
      this.editingMemberId = ''
      await this.loadProjectContext(this.selectedProjectId)
    } catch (e) {
      this.memberEditError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingMemberId = ''
    }
  }

  /* ---- load / refresh ---- */

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      const [me, mine, team, projects, employees, counts] = await Promise.all([
        fetchCurrentEmployee(),
        fetchMyTasks(true),
        fetchTeamTasks(true),
        fetchProjects(true),
        fetchEmployees(false),
        fetchProjectTaskCounts(),
      ])
      this.currentEmployee = me
      this.myTasks = mine
      this.teamTasks = team
      this.projects = projects
      this.employees = employees
      this.projectTaskCounts = counts
      if (!this.selectedProjectId && projects.length > 0) {
        await this.selectProject(projects[0]!.id)
      }
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  /** Re-fetch the two task lists (+ project context/counts if applicable)
   * after any task mutation — mirrors the old screen's `await load()`. */
  async refreshLists(): Promise<void> {
    try {
      const [mine, team, counts] = await Promise.all([fetchMyTasks(true), fetchTeamTasks(true), fetchProjectTaskCounts()])
      this.myTasks = mine
      this.teamTasks = team
      this.projectTaskCounts = counts
      if (this.selectedProjectId) await this.loadProjectContext(this.selectedProjectId)
    } catch {
      // best-effort refresh — the mutation itself already surfaced its own error
    }
  }
}
