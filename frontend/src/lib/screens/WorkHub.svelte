<script lang="ts">
  import { run } from 'svelte/legacy';

  import { onMount, onDestroy, tick } from "svelte";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
  import { toast } from "../stores/toasts";
  import { confirm } from "$lib/stores/confirm";
  import { pendingProjectHandoff, type PendingProjectHandoff } from "$lib/stores/navigation";
  import { can, permissions } from "$lib/stores/authContext";
  import WabiSpinner from "$lib/components/ui/WabiSpinner.svelte";
  import ApprovalsQueueScreen from "$lib/screens/ApprovalsQueueScreen.svelte";
  import WorkHubTaskComposer from "$lib/components/workhub/WorkHubTaskComposer.svelte";
  import WorkHubMyWorkPanel from "$lib/components/workhub/WorkHubMyWorkPanel.svelte";
  import WorkHubTeamBoardPanel from "$lib/components/workhub/WorkHubTeamBoardPanel.svelte";
  import WorkHubTaskDetailModal from "$lib/components/workhub/WorkHubTaskDetailModal.svelte";
  import WorkHubProjectsPanel from "$lib/components/workhub/WorkHubProjectsPanel.svelte";
  import {
    addProjectMember,
    addTaskComment,
    archiveProject,
    createProject,
    createTask,
    deleteProject,
    deleteTask,
    getCurrentEmployeeContext,
    getEmployeeAllocationSummary,
    getProjectTaskCounts,
    getTask,
    listEmployeeProfiles,
    listProjectActivity,
    listMyTasks,
    listProjectMembers,
    listProjectTasks,
    listProjects,
    listTaskActivity,
    listTaskComments,
    listTeamTasks,
    refreshCollaborativeWorkspace,
    reassignTask,
    shelveProject,
    updateProject,
    updateTask,
    updateTaskDueDate,
    updateTaskStatus,
    type CollaborativeProject,
    type CollaborativeTask,
    type CurrentEmployeeContext,
    type EmployeeProfile,
    type ProjectMember,
    type TaskActivityItem,
    type TaskCommentItem,
  } from "$lib/api/collaboration";
  import {
    formatDate,
    projectNameFor,
    normalizedBoardStatus,
    dueLabel,
    isTaskOverdue,
    upsertTaskRow,
    upsertProjectRow,
    assigneeOptionsFor,
  } from "$lib/components/workhub/workHubHelpers";

  type WorkTab = "my_work" | "team_board" | "projects" | "approvals";
  const pendingTaskStorageKey = "asymmflow.pendingCollaborativeTaskId";
  const WORK_HUB_CACHE_TTL_MS = 30_000;

  type WorkHubSnapshot = {
    currentEmployee: CurrentEmployeeContext | null;
    myTasks: CollaborativeTask[];
    teamTasks: CollaborativeTask[];
    projects: CollaborativeProject[];
    employees: EmployeeProfile[];
    selectedProjectId: string;
    selectedProjectForTask: string;
    selectedAssignee: string;
    selectedTaskId: string;
    activeTab: WorkTab;
    initializedPreferredTab: boolean;
    savedAt: number;
  };

  let workHubSnapshot: WorkHubSnapshot | null = null;

  let loading = $state(true);
  let savingTask = $state(false);
  let savingProject = $state(false);
  let savingComment = $state(false);
  let savingMember = $state(false);
  let savingAssignment = $state(false);
  let savingDueDate = $state(false);
  let savingTaskDetails = $state(false);
  let deletingTask = $state(false);
  let confirmDeleteTask = $state(false);
  let deletingProject = $state(false);

  let activeTab: WorkTab = $state("team_board");
  let currentEmployee: CurrentEmployeeContext | null = $state(null);
  let myTasks: CollaborativeTask[] = $state([]);
  let teamTasks: CollaborativeTask[] = $state([]);
  let projects: CollaborativeProject[] = $state([]);
  let employees: EmployeeProfile[] = $state([]);
  let projectMembers: ProjectMember[] = $state([]);
  let projectTasks: CollaborativeTask[] = $state([]);
  let projectActivity: TaskActivityItem[] = $state([]);
  let selectedTask: CollaborativeTask | null = $state(null);
  let taskComments: TaskCommentItem[] = $state([]);
  let taskActivity: TaskActivityItem[] = $state([]);
  let taskModalOpen = $state(false);

  let selectedTaskId = $state("");
  let selectedProjectId = $state("");

  let taskTitle = $state("");
  let taskDescription = $state("");
  let taskPriority = $state("medium");
  let taskDueDate = $state("");
  let selectedProjectForTask = $state("");
  let selectedAssignee = $state("");

  let projectName = $state("");
  let projectType = $state("internal");
  let projectDescription = $state("");
  // B4.2: customer/POC lineage block, only shown+sent when projectType === "customer".
  let projectCustomerName = $state("");
  let projectPOCName = $state("");
  let projectPOCEmail = $state("");
  let projectPOCPhone = $state("");
  let projectLineageCustomerId = $state("");
  let projectLineageOpportunityId = $state("");
  let projectLineageOrderId = $state("");
  let projectComposerEl: HTMLElement | null = $state(null);
  let memberStepEl: HTMLElement | null = $state(null);
  let highlightMemberStep = $state(false);

  let editingProject = $state(false);
  let projectEditName = $state("");
  let projectEditType = $state("internal");
  let projectEditDescription = $state("");
  let savingProjectAdmin = $state(false);
  // (e) FIX: Archive/Shelve/Restore require projects:update, Delete requires
  // projects:delete server-side already — gate button VISIBILITY to match
  // (referencing $permissions keeps these reactive as roles/permissions load).
  let canManageProjects = $derived($permissions ? can("projects:update") : false);
  let canDeleteProject = $derived($permissions ? can("projects:delete") : false);

  // B5d: archived/shelved/deleted projects are hidden from the default active
  // list; this toggle fetches them separately (listProjects(false), filtered
  // to terminal statuses) so they stay findable + restorable.
  let projectsShowArchived = $state(false);
  let archivedProjects: CollaborativeProject[] = $state([]);
  let loadingArchivedProjects = $state(false);
  // B3.3: per-project total task counts for list-row badges (own count, not 0).
  let projectTaskCounts: Record<string, number> = $state({});

  let memberSelections: string[] = $state([]);
  // B3.2: add-time defaults (applied per-employee on add); each member then
  // gets its own editable role + allocation via memberEditDrafts below —
  // "add-then-edit-per-member" so a batch add never forces one shared role.
  let newMemberRole = $state("Member");
  let newMemberAllocation = $state(100);
  let memberEditDrafts: Record<string, { role: string; allocation: number }> = $state({});
  let savingMemberId = $state("");
  let commentDraft = $state("");
  let selectedTaskAssigneeDraft = $state("");
  let selectedTaskDueDateDraft = $state("");
  let selectedTaskBlockerDraft = $state("");
  let selectedTaskTitleDraft = $state("");
  let selectedTaskDescriptionDraft = $state("");
  let selectedTaskPriorityDraft = $state("medium");
  let teamFocusFilter = $state("all");
  let teamAssigneeFilter = $state("all");
  let initializedPreferredTab = false;
  let loadRequestSeq = 0;
  let projectContextRequestSeq = 0;
  let taskDetailRequestSeq = 0;
  let pendingOpenTaskId = $state("");
  let openingPendingTask = $state(false);
  let lastLoadFailed = $state(false);
  let lastLoadError = $state("");
  let boardRecoveryAttempts = 0;
  let projectContextLoading = $state(false);
  let projectContextLoadedFor = $state("");
  let projectContextFailedFor = $state("");
  let taskDetailLoading = $state(false);

  // B3.1: membership-scoped assignee lists for project-scoped work. Falls
  // back to all `employees` when there's no project context (see the
  // `assigneeOptionsFor` helper below).
  let taskComposerMembers: ProjectMember[] = $state([]);
  let taskComposerMembersLoadedFor = $state("");
  let taskDetailMembers: ProjectMember[] = $state([]);
  let taskDetailMembersLoadedFor = $state("");

  function defaultWorkTabForRole(role?: string): WorkTab {
    const normalized = (role || "").toLowerCase();
    if (["admin", "manager", "developer"].includes(normalized)) {
      return "team_board";
    }
    return "my_work";
  }

  function hasLoadedWorkspaceData() {
    return Boolean(
      currentEmployee
      || myTasks.length
      || teamTasks.length
      || projects.length
      || employees.length,
    );
  }

  function restoreSnapshot() {
    if (!workHubSnapshot) return false;
    currentEmployee = workHubSnapshot.currentEmployee;
    myTasks = workHubSnapshot.myTasks;
    teamTasks = workHubSnapshot.teamTasks;
    projects = workHubSnapshot.projects;
    employees = workHubSnapshot.employees;
    selectedProjectId = workHubSnapshot.selectedProjectId;
    selectedProjectForTask = workHubSnapshot.selectedProjectForTask;
    selectedAssignee = workHubSnapshot.selectedAssignee;
    selectedTaskId = workHubSnapshot.selectedTaskId;
    activeTab = workHubSnapshot.activeTab;
    initializedPreferredTab = workHubSnapshot.initializedPreferredTab;
    loading = false;
    return true;
  }

  function saveSnapshot() {
    workHubSnapshot = {
      currentEmployee,
      myTasks: [...myTasks],
      teamTasks: [...teamTasks],
      projects: [...projects],
      employees: [...employees],
      selectedProjectId,
      selectedProjectForTask,
      selectedAssignee,
      selectedTaskId,
      activeTab,
      initializedPreferredTab,
      savedAt: Date.now(),
    };
  }

  function isSnapshotStale() {
    return !workHubSnapshot || Date.now() - workHubSnapshot.savedAt > WORK_HUB_CACHE_TTL_MS;
  }

  async function load(options: { refreshRemote?: boolean; forceRemote?: boolean; silent?: boolean } = {}) {
    const { refreshRemote = false, forceRemote = false, silent = false } = options;
    const requestSeq = ++loadRequestSeq;
    const shouldShowLoading = !silent || !hasLoadedWorkspaceData();
    if (shouldShowLoading) {
      loading = true;
    }
    try {
      if (refreshRemote) {
        await refreshCollaborativeWorkspace({ force: forceRemote }).catch(() => undefined);
      }
      const [employee, myTaskRows, teamTaskRows, projectRows, employeeRows, taskCounts] = await Promise.all([
        getCurrentEmployeeContext(),
        // B5a: fetch all of "my" tasks (including completed) once, then filter
        // client-side via the completed-work toggle — mirrors how Team Board
        // fetches everything and filters in the UI.
        listMyTasks(true),
        listTeamTasks(true),
        listProjects(true),
        listEmployeeProfiles(true),
        getProjectTaskCounts().catch(() => ({}) as Record<string, number>),
      ]);

      if (requestSeq !== loadRequestSeq) {
        return;
      }

      currentEmployee = employee;
      myTasks = myTaskRows || [];
      teamTasks = teamTaskRows || [];
      projects = projectRows || [];
      employees = employeeRows || [];
      projectTaskCounts = taskCounts || {};
      lastLoadFailed = false;
      lastLoadError = "";
      boardRecoveryAttempts = 0;

      if (!selectedAssignee && currentEmployee?.employee_id) {
        selectedAssignee = currentEmployee.employee_id;
      }
      if (!selectedProjectId && projects.length > 0) {
        selectedProjectId = projects[0].id;
      }
      if (!selectedProjectForTask && selectedProjectId) {
        selectedProjectForTask = selectedProjectId;
      }
      if (!selectedTaskId) {
        selectedTaskId = myTasks[0]?.id || teamTasks[0]?.id || "";
      }
      if (!initializedPreferredTab && currentEmployee) {
        activeTab = defaultWorkTabForRole(currentEmployee.license_role);
        initializedPreferredTab = true;
      }
      saveSnapshot();
    } catch (err) {
      if (requestSeq !== loadRequestSeq) {
        return;
      }
      lastLoadFailed = true;
      lastLoadError = String(err);
      if (!silent || !hasLoadedWorkspaceData()) {
        toast.danger(`Failed to load work hub: ${String(err)}`);
      }
    } finally {
      if (requestSeq === loadRequestSeq && shouldShowLoading) {
        loading = false;
      }
    }
  }

  async function hydrateWorkHub() {
    const restored = restoreSnapshot();
    if (!restored) {
      await load();
      await load({ refreshRemote: true, silent: true });
      return;
    }
    if (hasLoadedWorkspaceData()) {
      await load({ refreshRemote: isSnapshotStale(), silent: true });
      return;
    }
    await load({ refreshRemote: true, silent: true });
  }

  async function recoverBoardIfNeeded() {
    if (loading) return;
    if (boardRecoveryAttempts >= 2) return;
    if (!lastLoadFailed && (teamTasks.length > 0 || employees.length > 0 || projects.length > 0)) {
      return;
    }
    boardRecoveryAttempts += 1;
    await load({ refreshRemote: true, forceRemote: true });
  }

  async function loadProjectContext(projectID: string) {
    const requestSeq = ++projectContextRequestSeq;
    if (!projectID) {
      projectMembers = [];
      projectTasks = [];
      projectActivity = [];
      projectContextLoading = false;
      projectContextLoadedFor = "";
      projectContextFailedFor = "";
      return;
    }

    projectContextLoading = true;
    projectContextFailedFor = "";
    try {
      const [memberRows, taskRows, activityRows] = await Promise.all([
        listProjectMembers(projectID),
        listProjectTasks(projectID, true),
        listProjectActivity(projectID),
      ]);
      if (requestSeq !== projectContextRequestSeq) {
        return;
      }
      projectMembers = memberRows;
      projectTasks = taskRows;
      projectActivity = activityRows;
      projectContextLoadedFor = projectID;
    } catch (err) {
      if (requestSeq === projectContextRequestSeq) {
        projectContextFailedFor = projectID;
        toast.danger(`Failed to load project context: ${String(err)}`);
      }
    } finally {
      if (requestSeq === projectContextRequestSeq) {
        projectContextLoading = false;
      }
    }
  }

  async function loadTaskDetail(taskID: string, options: { suppressErrorToast?: boolean } = {}): Promise<boolean> {
    const { suppressErrorToast = false } = options;
    const requestSeq = ++taskDetailRequestSeq;
    if (!taskID) {
      selectedTask = null;
      taskComments = [];
      taskActivity = [];
      taskDetailLoading = false;
      return false;
    }

    taskDetailLoading = true;
    try {
      const [task, comments, activity] = await Promise.all([
        getTask(taskID),
        listTaskComments(taskID),
        listTaskActivity(taskID),
      ]);
      if (requestSeq !== taskDetailRequestSeq) {
        return;
      }
      selectedTask = task;
      taskComments = comments;
      taskActivity = activity;
      selectedTaskTitleDraft = task?.title || "";
      selectedTaskDescriptionDraft = task?.description || "";
      selectedTaskPriorityDraft = task?.priority || "medium";
      selectedTaskAssigneeDraft = task?.assignee_employee_id || "";
      selectedTaskDueDateDraft = task?.due_date ? new Date(task.due_date).toISOString().slice(0, 10) : "";
      selectedTaskBlockerDraft = task?.blocked_reason || "";
      confirmDeleteTask = false;
      return true;
    } catch (err) {
      if (requestSeq === taskDetailRequestSeq) {
        selectedTask = null;
        taskComments = [];
        taskActivity = [];
      }
      if (!suppressErrorToast) {
        toast.danger(`Failed to load task detail: ${String(err)}`);
      }
      return false;
    } finally {
      if (requestSeq === taskDetailRequestSeq) {
        taskDetailLoading = false;
      }
    }
  }

  async function openTaskAfterRefresh(taskID: string) {
    if (!taskID || openingPendingTask) return;

    openingPendingTask = true;
    taskModalOpen = true;
    selectedTaskId = taskID;

    try {
      let opened = false;
      for (let attempt = 0; attempt < 3 && !opened; attempt += 1) {
        if (attempt > 0 && !loading) {
          await load({ refreshRemote: true, forceRemote: true });
        }
        opened = await loadTaskDetail(taskID, { suppressErrorToast: true });
      }

      if (!opened) {
        if (pendingOpenTaskId === taskID) {
          pendingOpenTaskId = "";
        }
        if (selectedTaskId === taskID) {
          selectedTaskId = "";
        }
        taskModalOpen = false;
        toast.danger("Failed to load task detail: task not found. The task may still be syncing to this device.");
        return;
      }

      if (pendingOpenTaskId === taskID) {
        pendingOpenTaskId = "";
      }
    } finally {
      openingPendingTask = false;
    }
  }

  async function handleCreateTask() {
    if (!taskTitle.trim()) {
      toast.warning("Task title is required");
      return;
    }

    savingTask = true;
    try {
      const created = await createTask({
        title: taskTitle.trim(),
        description: taskDescription.trim(),
        priority: taskPriority,
        due_date: taskDueDate ? new Date(`${taskDueDate}T09:00:00`).toISOString() : undefined,
        project_id: selectedProjectForTask || undefined,
        assignee_employee_id: selectedAssignee || currentEmployee?.employee_id || undefined,
      });
      taskTitle = "";
      taskDescription = "";
      taskPriority = "medium";
      taskDueDate = "";
      const createdAssignee = created.assignee_employee_id || selectedAssignee || currentEmployee?.employee_id || "";
      const assignedToCurrent = !!currentEmployee?.employee_id && createdAssignee === currentEmployee.employee_id;
      myTasks = assignedToCurrent ? upsertTaskRow(myTasks, created) : myTasks.filter((task) => task.id !== created.id);
      teamTasks = upsertTaskRow(teamTasks, created);
      if (selectedProjectForTask && created.project_id === selectedProjectForTask) {
        projectTasks = upsertTaskRow(projectTasks, created);
      }
      activeTab = assignedToCurrent ? "my_work" : "team_board";
      if (!assignedToCurrent) {
        teamFocusFilter = "all";
        teamAssigneeFilter = "all";
      }
      queuePendingTaskOpen(created.id);
      primeTaskDetail(created);
      pendingOpenTaskId = "";
      void load({ refreshRemote: true, forceRemote: true, silent: true });
      void loadTaskDetail(created.id, { suppressErrorToast: true });
      toast.success("Task created");
    } catch (err) {
      toast.danger(`Failed to create task: ${String(err)}`);
    } finally {
      savingTask = false;
    }
  }

  const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

  function resetProjectComposer() {
    projectName = "";
    projectDescription = "";
    projectType = "internal";
    projectCustomerName = "";
    projectPOCName = "";
    projectPOCEmail = "";
    projectPOCPhone = "";
    projectLineageCustomerId = "";
    projectLineageOpportunityId = "";
    projectLineageOrderId = "";
  }

  async function handleCreateProject() {
    if (!projectName.trim()) {
      toast.warning("Project name is required");
      return;
    }
    // B4.2: customer/POC block only applies (and is only sent) when this is a
    // customer project — internal/admin projects never carry customer fields.
    if (projectType === "customer" && projectPOCEmail.trim() && !EMAIL_PATTERN.test(projectPOCEmail.trim())) {
      toast.warning("POC email looks invalid — use name@example.com format");
      return;
    }

    savingProject = true;
    try {
      const payload: Record<string, any> = {
        name: projectName.trim(),
        project_type: projectType,
        description: projectDescription.trim(),
      };
      if (projectType === "customer") {
        if (projectLineageCustomerId) payload.customer_id = projectLineageCustomerId;
        if (projectLineageOpportunityId) payload.opportunity_id = projectLineageOpportunityId;
        if (projectLineageOrderId) payload.order_id = projectLineageOrderId;
        payload.customer_name = projectCustomerName.trim();
        payload.customer_poc_name = projectPOCName.trim();
        payload.customer_poc_email = projectPOCEmail.trim();
        payload.customer_poc_phone = projectPOCPhone.trim();
      }
      const created = await createProject(payload);
      resetProjectComposer();
      projects = upsertProjectRow(projects, created);
      selectedProjectId = created.id;
      selectedProjectForTask = created.id;
      projectContextRequestSeq += 1;
      projectMembers = [];
      projectTasks = [];
      projectActivity = [];
      memberSelections = [];
      projectContextLoading = false;
      projectContextLoadedFor = created.id;
      projectContextFailedFor = "";
      if (employees.length === 0) {
        employees = await listEmployeeProfiles(true).catch((err) => {
          toast.danger(`Project created, but employee directory could not load: ${String(err)}`);
          return [];
        });
      }
      saveSnapshot();
      void load({ refreshRemote: true, silent: true });
      toast.success("Project created — add members below to get moving");
      // B4.3: land on the member step, not a dead-end — scroll it into view
      // and pulse it briefly so "what's next" is obvious.
      highlightMemberStep = true;
      void tick().then(() => memberStepEl?.scrollIntoView({ behavior: "smooth", block: "center" }));
      setTimeout(() => { highlightMemberStep = false; }, 4000);
    } catch (err) {
      toast.danger(`Failed to create project: ${String(err)}`);
    } finally {
      savingProject = false;
    }
  }

  // B4.1: consume a "Start project" handoff from Opportunity/Order screens —
  // preseed the composer and jump to the Projects tab with it open.
  function applyPendingProjectHandoff(payload: PendingProjectHandoff) {
    activeTab = "projects";
    projectsShowArchived = false;
    projectType = payload.customerId || payload.customerName ? "customer" : "internal";
    projectName = payload.suggestedName || "";
    projectLineageCustomerId = payload.customerId || "";
    projectLineageOpportunityId = payload.opportunityId || "";
    projectLineageOrderId = payload.orderId || "";
    projectCustomerName = payload.customerName || "";
    projectPOCName = payload.pocName || "";
    projectPOCEmail = payload.pocEmail || "";
    projectPOCPhone = payload.pocPhone || "";
    void tick().then(() => projectComposerEl?.scrollIntoView({ behavior: "smooth", block: "start" }));
  }

  function startProjectEdit() {
    if (!selectedProject) return;
    projectEditName = selectedProject.name || "";
    projectEditType = selectedProject.project_type || "internal";
    projectEditDescription = selectedProject.description || "";
    editingProject = true;
  }

  function cancelProjectEdit() {
    editingProject = false;
  }

  async function saveProjectEdit() {
    if (!selectedProject) return;
    if (!projectEditName.trim()) {
      toast.warning("Project name is required");
      return;
    }
    savingProjectAdmin = true;
    try {
      const updated = await updateProject(selectedProject.id, {
        name: projectEditName.trim(),
        project_type: projectEditType || "internal",
        description: projectEditDescription.trim(),
      });
      projects = upsertProjectRow(projects, updated);
      editingProject = false;
      saveSnapshot();
      toast.success("Project updated");
    } catch (err) {
      toast.danger(`Failed to update project: ${String(err)}`);
    } finally {
      savingProjectAdmin = false;
    }
  }

  // Shared post-action cleanup for archive/shelve/delete — all three are
  // terminal statuses that remove the project from the active list.
  function finishProjectAdminCleanup() {
    projects = projects.filter((project) => project.id !== selectedProject.id);
    archivedProjects = archivedProjects.filter((project) => project.id !== selectedProject.id);
    selectedProjectId = projects[0]?.id || "";
    selectedProjectForTask = selectedProjectId;
    projectMembers = [];
    projectTasks = [];
    projectActivity = [];
    editingProject = false;
    saveSnapshot();
    void load({ refreshRemote: true, silent: true });
  }

  // B5d: archive/shelve require a canonical, REQUIRED audit reason (same bar
  // as delete) — no more free-text-optional input on the panel.
  async function applyProjectAdminAction(action: "archive" | "shelve") {
    if (!selectedProject) return;
    const label = action === "archive" ? "Archive" : "Shelve";
    const explain = action === "archive"
      ? "This closes a finished project."
      : "This pauses the project; it stays findable under Archived and can be restored later.";
    const r = await confirm.askForReason({
      title: `${label} Project`,
      message: `${label} "${selectedProject.name}"? ${explain}`,
      confirmLabel: label,
      variant: "warning",
      reasonLabel: "Reason (recorded in the audit trail)",
      reasonRequired: true,
    });
    if (!r.confirmed) return;

    savingProjectAdmin = true;
    try {
      if (action === "archive") {
        await archiveProject(selectedProject.id, r.reason);
      } else {
        await shelveProject(selectedProject.id, r.reason);
      }
      finishProjectAdminCleanup();
      toast.success(`Project ${action}d`);
    } catch (err) {
      toast.danger(`Failed to ${action} project: ${String(err)}`);
    } finally {
      savingProjectAdmin = false;
    }
  }

  // B5d: restore an archived/shelved project back to active — normal
  // projects:update permission suffices (only moving INTO a terminal status
  // escalates to projects:delete).
  async function handleRestoreProject() {
    if (!selectedProject) return;
    savingProjectAdmin = true;
    try {
      const updated = await updateProject(selectedProject.id, { status: "active" });
      archivedProjects = archivedProjects.filter((project) => project.id !== updated.id);
      projects = upsertProjectRow(projects, updated);
      selectedProjectId = updated.id;
      selectedProjectForTask = updated.id;
      projectsShowArchived = false;
      saveSnapshot();
      void load({ refreshRemote: true, silent: true });
      toast.success("Project restored to active");
    } catch (err) {
      toast.danger(`Failed to restore project: ${String(err)}`);
    } finally {
      savingProjectAdmin = false;
    }
  }

  async function loadArchivedProjects() {
    loadingArchivedProjects = true;
    try {
      const rows = await listProjects(false);
      archivedProjects = rows.filter((project) =>
        ["archived", "shelved", "deleted"].includes(String(project.status || "").toLowerCase()),
      );
    } catch (err) {
      toast.danger(`Failed to load archived projects: ${String(err)}`);
    } finally {
      loadingArchivedProjects = false;
    }
  }

  async function toggleArchivedProjectsView() {
    projectsShowArchived = !projectsShowArchived;
    if (projectsShowArchived && archivedProjects.length === 0) {
      await loadArchivedProjects();
    }
  }

  // Wave 9.3 B3d: project delete must be at least as strong a guard as task
  // delete (Design Constitution Article III.6) — routed through the
  // canonical confirm.askForReason with a REQUIRED reason and the cascade
  // consequence stated up front. No one-click / canned-reason path remains.
  async function handleDeleteProject() {
    if (!selectedProject) return;
    const taskCount = projectTasks.length;
    const memberCount = projectMembers.length;
    const cascadeNote =
      taskCount > 0 || memberCount > 0
        ? ` It has ${taskCount} task${taskCount === 1 ? "" : "s"} and ${memberCount} member${memberCount === 1 ? "" : "s"} attached — they will stay linked to the deleted project's history.`
        : "";
    const r = await confirm.askForReason({
      title: "Delete Project",
      message: `Delete "${selectedProject.name}"? This cannot be undone.${cascadeNote}`,
      confirmLabel: "Delete Project",
      variant: "danger",
      reasonLabel: "Reason for deletion",
      reasonRequired: true,
    });
    if (!r.confirmed) return;

    deletingProject = true;
    savingProjectAdmin = true;
    try {
      await deleteProject(selectedProject.id, r.reason);
      finishProjectAdminCleanup();
      toast.success("Project deleted");
    } catch (err) {
      toast.danger(`Failed to delete project: ${String(err)}`);
    } finally {
      deletingProject = false;
      savingProjectAdmin = false;
    }
  }

  async function handleAddProjectMembers() {
    if (!selectedProjectId) {
      toast.warning("Choose a project first");
      return;
    }
    if (memberSelections.length === 0) {
      toast.warning("Choose at least one employee to add");
      return;
    }

    const role = newMemberRole.trim() || "Member";
    const allocation = Number.isFinite(newMemberAllocation) && newMemberAllocation > 0
      ? Math.min(newMemberAllocation, 100)
      : 100;

    // Wave 9.8 B3: precheck every selected employee before adding anyone —
    // one aggregated WARN for the whole batch (server totals, never a
    // client-side sum), so a cancel aborts cleanly with nothing half-added.
    const overAllocated: string[] = [];
    for (const employeeID of memberSelections) {
      const summary = await getEmployeeAllocationSummary(employeeID, selectedProjectId);
      const resultingTotal = summary.other_projects_total + allocation;
      if (resultingTotal > 100) {
        const name = employees.find((e) => e.id === employeeID)?.full_name || employeeID;
        overAllocated.push(`${name} (would reach ${resultingTotal}%)`);
      }
    }
    if (overAllocated.length > 0) {
      const proceed = await confirm.ask({
        title: "Over 100% Allocation",
        message: `Adding ${allocation}% here would push these members over 100% total allocation: ${overAllocated.join(", ")}. Save anyway?`,
        confirmLabel: "Save Anyway",
        variant: "warning",
      });
      if (!proceed) return;
    }

    savingMember = true;
    try {
      await Promise.all(
        memberSelections.map((employeeID) => addProjectMember(selectedProjectId, employeeID, role, allocation)),
      );
      const addedCount = memberSelections.length;
      memberSelections = [];
      newMemberRole = "Member";
      newMemberAllocation = 100;
      await loadProjectContext(selectedProjectId);
      toast.success(`${addedCount} project ${addedCount === 1 ? "member" : "members"} added`);
    } catch (err) {
      toast.danger(`Failed to add project members: ${String(err)}`);
    } finally {
      savingMember = false;
    }
  }

  function toggleProjectMemberSelection(employeeID: string) {
    memberSelections = memberSelections.includes(employeeID)
      ? memberSelections.filter((id) => id !== employeeID)
      : [...memberSelections, employeeID];
  }

  // B3.2: per-member role + allocation editing ("add-then-edit-per-member").
  // AddCollaborativeProjectMember upserts on the existing (project_id, employee_id)
  // pair, so re-calling it with new role/allocation updates that member in place.
  function memberDraft(member: ProjectMember) {
    return memberEditDrafts[member.employee_id] || { role: member.role || "Member", allocation: member.allocation_percent ?? 100 };
  }

  function setMemberDraftField(employeeID: string, field: "role" | "allocation", value: string | number) {
    const current = memberEditDrafts[employeeID] || { role: "Member", allocation: 100 };
    memberEditDrafts = { ...memberEditDrafts, [employeeID]: { ...current, [field]: value } };
  }

  // Wave 9.8 B3: allocation capacity is a WARN, never a hard block. The
  // server (GetEmployeeAllocationSummary) computes what this employee is
  // already committed to on OTHER active projects — we only ever display
  // that number, never re-derive it client-side.
  async function confirmOverAllocation(employeeID: string, employeeName: string, projectID: string, newAllocation: number): Promise<boolean> {
    let summary;
    try {
      summary = await getEmployeeAllocationSummary(employeeID, projectID);
    } catch (err) {
      // If the precheck itself fails, don't block the save on it — just skip the warning.
      console.error("Allocation summary precheck failed", err);
      return true;
    }
    const resultingTotal = summary.other_projects_total + newAllocation;
    if (resultingTotal <= 100) return true;

    const otherLines = summary.projects
      .map((p) => `${p.project_name || p.project_id}: ${p.allocation_percent}%`)
      .join(", ");
    return confirm.ask({
      title: "Over 100% Allocation",
      message: `${employeeName || "This person"} is already committed to ${summary.other_projects_total}% across other active projects${otherLines ? ` (${otherLines})` : ""}. Adding ${newAllocation}% here brings their total to ${resultingTotal}%. Save anyway?`,
      confirmLabel: "Save Anyway",
      variant: "warning",
    });
  }

  async function handleSaveMember(member: ProjectMember) {
    const draft = memberDraft(member);
    const role = String(draft.role || "Member").trim() || "Member";
    const allocation = Number(draft.allocation);
    const clampedAllocation = Number.isFinite(allocation) && allocation > 0 ? Math.min(allocation, 100) : 100;

    const proceed = await confirmOverAllocation(member.employee_id, member.employee_name || "", selectedProjectId, clampedAllocation);
    if (!proceed) return;

    savingMemberId = member.employee_id;
    try {
      await addProjectMember(selectedProjectId, member.employee_id, role, clampedAllocation);
      const next = { ...memberEditDrafts };
      delete next[member.employee_id];
      memberEditDrafts = next;
      await loadProjectContext(selectedProjectId);
      toast.success(`${member.employee_name || "Member"} updated`);
    } catch (err) {
      toast.danger(`Failed to update member: ${String(err)}`);
    } finally {
      savingMemberId = "";
    }
  }

  async function changeStatus(task: CollaborativeTask, status: string, note = "") {
    try {
      await updateTaskStatus(task.id, status, note);
      await load();
      if (selectedTaskId === task.id) {
        await loadTaskDetail(task.id);
      }
      toast.success(`Task marked ${status.replace("_", " ")}`);
    } catch (err) {
      toast.danger(`Failed to update task: ${String(err)}`);
    }
  }

  async function handleAddComment() {
    if (!selectedTaskId || !commentDraft.trim()) {
      return;
    }

    savingComment = true;
    try {
      await addTaskComment(selectedTaskId, commentDraft.trim());
      commentDraft = "";
      await loadTaskDetail(selectedTaskId);
      toast.success("Comment added");
    } catch (err) {
      toast.danger(`Failed to add comment: ${String(err)}`);
    } finally {
      savingComment = false;
    }
  }

  async function handleReassignSelectedTask() {
    if (!selectedTask) return;
    savingAssignment = true;
    try {
      await reassignTask(selectedTask.id, selectedTaskAssigneeDraft);
      await load();
      await loadTaskDetail(selectedTask.id);
      toast.success("Task assignee updated");
    } catch (err) {
      toast.danger(`Failed to reassign task: ${String(err)}`);
    } finally {
      savingAssignment = false;
    }
  }

  async function handleUpdateDueDate() {
    if (!selectedTask) return;
    savingDueDate = true;
    try {
      const dueDateISO = selectedTaskDueDateDraft ? new Date(`${selectedTaskDueDateDraft}T09:00:00`).toISOString() : "";
      await updateTaskDueDate(selectedTask.id, dueDateISO);
      await load();
      await loadTaskDetail(selectedTask.id);
      toast.success("Task due date updated");
    } catch (err) {
      toast.danger(`Failed to update due date: ${String(err)}`);
    } finally {
      savingDueDate = false;
    }
  }

  async function selectTask(taskID: string) {
    if (!taskID) return;
    pendingOpenTaskId = "";
    taskModalOpen = true;
    const cachedTask = [...myTasks, ...teamTasks, ...projectTasks].find((task) => task.id === taskID);
    if (cachedTask) {
      primeTaskDetail(cachedTask);
    } else {
      queuePendingTaskOpen(taskID);
    }
    const opened = await loadTaskDetail(taskID, { suppressErrorToast: Boolean(cachedTask) });
    if (!opened && cachedTask) {
      primeTaskDetail(cachedTask);
      toast.warning("Showing cached task details while sync catches up.");
    }
  }

  function closeTaskModal() {
    taskModalOpen = false;
    confirmDeleteTask = false;
  }

  async function handleBlockSelectedTask() {
    if (!selectedTask) return;
    if (!selectedTaskBlockerDraft.trim()) {
      toast.warning("Add a blocked reason so the team knows what is stuck");
      return;
    }
    await changeStatus(selectedTask, "blocked", selectedTaskBlockerDraft.trim());
  }

  async function handleSaveTaskDetails() {
    if (!selectedTask) return;
    if (!selectedTaskTitleDraft.trim()) {
      toast.warning("Task title is required");
      return;
    }
    savingTaskDetails = true;
    try {
      await updateTask({
        id: selectedTask.id,
        title: selectedTaskTitleDraft.trim(),
        description: selectedTaskDescriptionDraft.trim(),
        priority: selectedTaskPriorityDraft,
        task_type: selectedTask.task_type,
        project_id: selectedTask.project_id,
      });
      await load();
      await loadTaskDetail(selectedTask.id);
      toast.success("Task details updated");
    } catch (err) {
      toast.danger(`Failed to update task: ${String(err)}`);
    } finally {
      savingTaskDetails = false;
    }
  }

  async function handleDeleteSelectedTask() {
    if (!selectedTask) return;
    if (!confirmDeleteTask) {
      confirmDeleteTask = true;
      toast.warning("Press Delete again to remove this task from the work board");
      return;
    }
    deletingTask = true;
    try {
      const taskID = selectedTask.id;
      await deleteTask(taskID);
      closeTaskModal();
      selectedTask = null;
      selectedTaskId = "";
      await load();
      toast.success("Task deleted");
    } catch (err) {
      toast.danger(`Failed to delete task: ${String(err)}`);
    } finally {
      deletingTask = false;
    }
  }

  function queuePendingTaskOpen(taskID: string) {
    if (!taskID) return;
    pendingOpenTaskId = taskID;
    selectedTaskId = taskID;
    taskModalOpen = true;
    taskDetailLoading = true;
    selectedTask = null;
    taskComments = [];
    taskActivity = [];
  }

  function primeTaskDetail(task: CollaborativeTask) {
    selectedTask = task;
    selectedTaskId = task.id;
    taskComments = [];
    taskActivity = [];
    taskDetailLoading = false;
    selectedTaskTitleDraft = task.title || "";
    selectedTaskDescriptionDraft = task.description || "";
    selectedTaskPriorityDraft = task.priority || "medium";
    selectedTaskAssigneeDraft = task.assignee_employee_id || "";
    selectedTaskDueDateDraft = task.due_date ? new Date(task.due_date).toISOString().slice(0, 10) : "";
    selectedTaskBlockerDraft = task.blocked_reason || "";
  }

  function consumePendingTaskOpenFromStorage(): string {
    const taskID = sessionStorage.getItem(pendingTaskStorageKey)?.trim() || "";
    if (taskID) {
      sessionStorage.removeItem(pendingTaskStorageKey);
    }
    return taskID;
  }

  async function selectProject(projectID: string) {
    selectedProjectId = projectID;
    selectedProjectForTask = projectID;
    editingProject = false;
    projectContextLoadedFor = "";
    projectContextFailedFor = "";
    await loadProjectContext(projectID);
  }

  function handleOpenTaskEvent(event: Event) {
    const taskID = (event as CustomEvent<{ taskID?: string }>).detail?.taskID || "";
    if (!taskID) return;
    activeTab = defaultWorkTabForRole(currentEmployee?.license_role);
    queuePendingTaskOpen(taskID);
    void openTaskAfterRefresh(taskID);
  }

  async function loadTaskComposerMembers(projectID: string) {
    if (!projectID) {
      taskComposerMembers = [];
      taskComposerMembersLoadedFor = "";
      return;
    }
    try {
      taskComposerMembers = await listProjectMembers(projectID);
      taskComposerMembersLoadedFor = projectID;
    } catch {
      taskComposerMembers = [];
      taskComposerMembersLoadedFor = "";
    }
  }

  async function loadTaskDetailMembers(projectID: string) {
    if (!projectID) {
      taskDetailMembers = [];
      taskDetailMembersLoadedFor = "";
      return;
    }
    try {
      taskDetailMembers = await listProjectMembers(projectID);
      taskDetailMembersLoadedFor = projectID;
    } catch {
      taskDetailMembers = [];
      taskDetailMembersLoadedFor = "";
    }
  }



  let unsubscribePendingProjectHandoff: (() => void) | null = null;

  onMount(() => {
    void hydrateWorkHub();
    EventsOn("tasks:updated", () => void load({ silent: true }));
    EventsOn("projects:updated", () => {
      void load({ silent: true });
      if (selectedProjectId) {
        projectContextLoadedFor = "";
        projectContextFailedFor = "";
        void loadProjectContext(selectedProjectId);
      }
    });
    EventsOn("employees:updated", () => void load({ silent: true }));
    window.addEventListener("openCollaborativeTask", handleOpenTaskEvent);
    const pendingTaskID = consumePendingTaskOpenFromStorage();
    if (pendingTaskID) {
      queuePendingTaskOpen(pendingTaskID);
    }
    // B4.1: "Start project" handoff from Opportunity/Order screens.
    unsubscribePendingProjectHandoff = pendingProjectHandoff.subscribe((payload) => {
      if (payload) {
        applyPendingProjectHandoff(payload);
        pendingProjectHandoff.clear();
      }
    });
  });

  onDestroy(() => {
    EventsOff("tasks:updated", "projects:updated", "employees:updated");
    window.removeEventListener("openCollaborativeTask", handleOpenTaskEvent);
    unsubscribePendingProjectHandoff?.();
  });
  // B5d: selected project may live in the active list or the archived list,
  // depending on which view is showing.
  let selectedProject = $derived(
    projects.find((project) => project.id === selectedProjectId)
    || archivedProjects.find((project) => project.id === selectedProjectId)
    || null,
  );
  let visibleProjects = $derived(projectsShowArchived ? archivedProjects : projects);
  let taskComposerAssigneeOptions = $derived(assigneeOptionsFor(taskComposerMembers, taskComposerMembersLoadedFor, selectedProjectForTask, employees));
  run(() => {
    if (selectedProjectForTask !== taskComposerMembersLoadedFor) {
      void loadTaskComposerMembers(selectedProjectForTask);
    }
  });
  let projectStats = $derived({
    open: projectTasks.filter((task) => !["completed", "archived"].includes(task.status || "")).length,
    blocked: projectTasks.filter((task) => task.status === "blocked").length,
    completed: projectTasks.filter((task) => task.status === "completed").length,
    members: projectMembers.length,
  });
  let availableProjectMembers = $derived(employees.filter((employee) => !projectMembers.some((member) => member.employee_id === employee.id)));
  run(() => {
    memberSelections = memberSelections.filter((employeeID) => availableProjectMembers.some((employee) => employee.id === employeeID));
  });
  let modalTask = $derived(selectedTask || [...myTasks, ...teamTasks, ...projectTasks].find((task) => task.id === selectedTaskId) || null);
  let taskDetailAssigneeOptions = $derived(assigneeOptionsFor(taskDetailMembers, taskDetailMembersLoadedFor, modalTask?.project_id || "", employees));
  run(() => {
    const taskProjectId = modalTask?.project_id || "";
    if (taskModalOpen && taskProjectId !== taskDetailMembersLoadedFor) {
      void loadTaskDetailMembers(taskProjectId);
    }
  });
  run(() => {
    if (
      activeTab === "projects"
      && selectedProjectId
      && projectContextLoadedFor !== selectedProjectId
      && projectContextFailedFor !== selectedProjectId
      && !projectContextLoading
      && !loading
    ) {
      void loadProjectContext(selectedProjectId);
    }
  });
  run(() => {
    if (pendingOpenTaskId && !loading && activeTab !== "projects") {
      void openTaskAfterRefresh(pendingOpenTaskId);
    }
  });
  run(() => {
    if (selectedTaskId && activeTab !== "projects" && !pendingOpenTaskId && !openingPendingTask && (!selectedTask || selectedTask.id !== selectedTaskId) && !loading) {
      void loadTaskDetail(selectedTaskId);
    }
  });
  run(() => {
    if (activeTab === "team_board" && !loading && (lastLoadFailed || (teamTasks.length === 0 && employees.length === 0 && projects.length === 0))) {
      void recoverBoardIfNeeded();
    }
  });
</script>

<div class="page">
  <header class="header">
    <div>
      <h1>Work.</h1>
      <p class="subtitle">Projects, team workload, and task history in one collaborative workspace.</p>
    </div>
    {#if currentEmployee}
      <div class="identity-card">
        <span class="label">Active Employee</span>
        <strong>{currentEmployee.employee_name}</strong>
        <span class="meta">{currentEmployee.license_role || currentEmployee.resolved_by}</span>
      </div>
    {/if}
  </header>

  <section class="tabs">
    <button class:active={activeTab === "my_work"} onclick={() => activeTab = "my_work"}>My Work</button>
    <button class:active={activeTab === "team_board"} onclick={() => activeTab = "team_board"}>Team Board</button>
    <button class:active={activeTab === "projects"} onclick={() => activeTab = "projects"}>Projects</button>
    <button class:active={activeTab === "approvals"} onclick={() => activeTab = "approvals"}>Approvals</button>
  </section>

  {#if activeTab === "my_work" || activeTab === "projects"}
    <WorkHubTaskComposer
      {projects}
      assigneeOptions={taskComposerAssigneeOptions}
      {savingTask}
      bind:taskTitle
      bind:taskDescription
      bind:taskPriority
      bind:taskDueDate
      bind:selectedProjectForTask
      bind:selectedAssignee
      onCreate={handleCreateTask}
    />
  {/if}

  {#if activeTab === "my_work"}
    <WorkHubMyWorkPanel {myTasks} {projects} {loading} {selectedTaskId} onSelectTask={selectTask} />
  {/if}

  {#if activeTab === "team_board"}
    <WorkHubTeamBoardPanel
      {teamTasks}
      {employees}
      {projects}
      {loading}
      {lastLoadFailed}
      {lastLoadError}
      {selectedTaskId}
      bind:teamFocusFilter
      bind:teamAssigneeFilter
      onSelectTask={selectTask}
      {load}
      {loadTaskDetail}
    />
  {/if}

  {#if activeTab === "projects"}
    <WorkHubProjectsPanel
      {projectsShowArchived}
      {visibleProjects}
      {loadingArchivedProjects}
      onToggleArchivedProjectsView={toggleArchivedProjectsView}
      bind:projectComposerEl
      bind:projectName
      bind:projectType
      bind:projectDescription
      bind:projectCustomerName
      bind:projectPOCName
      bind:projectPOCEmail
      bind:projectPOCPhone
      {savingProject}
      onCreateProject={handleCreateProject}
      {selectedProjectId}
      {projectTaskCounts}
      onSelectProject={selectProject}
      {selectedProject}
      {editingProject}
      bind:projectEditName
      bind:projectEditType
      bind:projectEditDescription
      {savingProjectAdmin}
      onStartProjectEdit={startProjectEdit}
      onCancelProjectEdit={cancelProjectEdit}
      onSaveProjectEdit={saveProjectEdit}
      {projectStats}
      {projectMembers}
      {availableProjectMembers}
      {projectContextLoading}
      bind:memberSelections
      bind:newMemberRole
      bind:newMemberAllocation
      {savingMember}
      {savingMemberId}
      {highlightMemberStep}
      bind:memberStepEl
      {memberDraft}
      onToggleMemberSelection={toggleProjectMemberSelection}
      onAddMembers={handleAddProjectMembers}
      onSetMemberDraftField={setMemberDraftField}
      onSaveMember={handleSaveMember}
      {projectTasks}
      {selectedTaskId}
      onSelectTask={selectTask}
      {projectActivity}
      {canManageProjects}
      {canDeleteProject}
      onRestoreProject={handleRestoreProject}
      onDeleteProject={handleDeleteProject}
      onApplyProjectAdminAction={applyProjectAdminAction}
    />
  {/if}

  {#if activeTab === "approvals"}
    <ApprovalsQueueScreen embedded={true} />
  {/if}
</div>

<WorkHubTaskDetailModal
  bind:open={taskModalOpen}
  {taskDetailLoading}
  {selectedTask}
  {selectedTaskId}
  {myTasks}
  {teamTasks}
  {projectTasks}
  {projects}
  {taskComments}
  {taskActivity}
  {taskDetailAssigneeOptions}
  bind:selectedTaskTitleDraft
  bind:selectedTaskDescriptionDraft
  bind:selectedTaskPriorityDraft
  bind:selectedTaskBlockerDraft
  bind:selectedTaskAssigneeDraft
  bind:selectedTaskDueDateDraft
  bind:commentDraft
  {confirmDeleteTask}
  {savingTaskDetails}
  {savingAssignment}
  {savingDueDate}
  {savingComment}
  {deletingTask}
  onSaveTaskDetails={handleSaveTaskDetails}
  onReassign={handleReassignSelectedTask}
  onUpdateDueDate={handleUpdateDueDate}
  onBlock={handleBlockSelectedTask}
  onChangeStatus={changeStatus}
  onDeleteTask={handleDeleteSelectedTask}
  onAddComment={handleAddComment}
  onClose={closeTaskModal}
/>

<style>
  .page {
    padding: 24px;
    display: grid;
    gap: 20px;
  }

  .header {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: flex-start;
  }

  h1,
  p {
    margin: 0;
  }

  .subtitle,
  .meta,
  .label {
    color: var(--text-secondary);
  }

  .identity-card,
  .tabs {
    border: 1px solid var(--border);
    border-radius: 16px;
    background: var(--surface);
  }

  .identity-card {
    padding: 14px 16px;
    min-width: 220px;
    display: grid;
    gap: 4px;
  }

  .tabs {
    display: flex;
    gap: 8px;
    padding: 8px;
    align-items: center;
  }

  .tabs button {
    font: inherit;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: none;
    border-radius: 999px;
    min-height: 40px;
    padding: 10px 16px;
    background: transparent;
    color: var(--text-secondary);
    line-height: 1;
    white-space: nowrap;
    transition: background 140ms ease, color 140ms ease, box-shadow 140ms ease;
  }

  .tabs button.active {
    background: var(--onyx);
    color: white;
    box-shadow: 0 8px 18px rgba(15, 23, 42, 0.12);
  }

  @media (max-width: 900px) {
    .header {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
