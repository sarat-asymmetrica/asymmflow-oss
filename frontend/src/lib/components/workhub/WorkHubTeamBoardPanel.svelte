<script lang="ts">
  import { toast } from "$lib/stores/toasts";
  import { reassignTask, type CollaborativeProject, type CollaborativeTask, type EmployeeProfile } from "$lib/api/collaboration";
  import { projectNameFor, normalizedBoardStatus, dueLabel } from "./workHubHelpers";
  import CardGridSkeleton from "$lib/components/ui/CardGridSkeleton.svelte";

  interface Props {
    teamTasks: CollaborativeTask[];
    employees: EmployeeProfile[];
    projects: CollaborativeProject[];
    loading: boolean;
    lastLoadFailed: boolean;
    lastLoadError: string;
    selectedTaskId: string;
    teamFocusFilter: string;
    teamAssigneeFilter: string;
    onSelectTask: (taskID: string) => void;
    load: (options?: { refreshRemote?: boolean; forceRemote?: boolean; silent?: boolean }) => Promise<void>;
    loadTaskDetail: (taskID: string, options?: { suppressErrorToast?: boolean }) => Promise<boolean>;
  }

  let {
    teamTasks,
    employees,
    projects,
    loading,
    lastLoadFailed,
    lastLoadError,
    selectedTaskId,
    teamFocusFilter = $bindable(),
    teamAssigneeFilter = $bindable(),
    onSelectTask,
    load,
    loadTaskDetail,
  }: Props = $props();

  let teamSearch = $state("");
  let draggedEmployeeId = $state("");
  let draggedOverTaskId = $state("");

  const boardColumns = [
    { key: "open", label: "Open" },
    { key: "in_progress", label: "In Progress" },
    { key: "blocked", label: "Blocked" },
    { key: "completed", label: "Completed" },
  ];

  let filteredTeamTasks = $derived(teamTasks.filter((task) => {
    const matchesSearch = !teamSearch.trim() || [task.title, task.description, task.assignee_name, task.creator_name]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(teamSearch.trim().toLowerCase()));
    const matchesAssignee = teamAssigneeFilter === "all"
      || (teamAssigneeFilter === "unassigned" && !task.assignee_employee_id)
      || task.assignee_employee_id === teamAssigneeFilter;
    const isOverdue = Boolean(task.due_date && new Date(task.due_date) < new Date() && !["completed", "archived"].includes(task.status || ""));
    const matchesFocus = teamFocusFilter === "all"
      || (teamFocusFilter === "active" && !["completed", "archived"].includes(task.status || ""))
      || (teamFocusFilter === "assigned" && !!task.assignee_employee_id)
      || (teamFocusFilter === "unassigned" && !task.assignee_employee_id)
      || (teamFocusFilter === "overdue" && isOverdue)
      || (teamFocusFilter === "blocked" && task.status === "blocked");
    return matchesSearch && matchesAssignee && matchesFocus;
  }));
  let boardTaskGroups = $derived(filteredTeamTasks.reduce((groups, task) => {
    const lane = normalizedBoardStatus(task);
    const targetLane = boardColumns.some((column) => column.key === lane) ? lane : "open";
    groups[targetLane] = [...groups[targetLane], task];
    return groups;
  }, {
    open: [] as CollaborativeTask[],
    in_progress: [] as CollaborativeTask[],
    blocked: [] as CollaborativeTask[],
    completed: [] as CollaborativeTask[],
  }));
  let workload = $derived(employees
    .map((employee) => ({
      id: employee.id,
      name: employee.full_name,
      department: employee.department,
      openCount: teamTasks.filter((task) => task.assignee_employee_id === employee.id && !["completed", "archived"].includes(task.status || "")).length,
      blockedCount: teamTasks.filter((task) => task.assignee_employee_id === employee.id && task.status === "blocked").length,
      urgentCount: teamTasks.filter((task) => task.assignee_employee_id === employee.id && ["high", "urgent"].includes(task.priority || "") && !["completed", "archived"].includes(task.status || "")).length,
    }))
    .sort((a, b) => b.openCount - a.openCount || b.blockedCount - a.blockedCount));
  let activeWorkload = $derived(workload.filter((item) => item.openCount > 0 || item.blockedCount > 0));

  function startEmployeeDrag(employeeID: string, event: DragEvent) {
    draggedEmployeeId = employeeID;
    event.dataTransfer?.setData("text/plain", employeeID);
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = "move";
    }
  }

  function allowTaskDrop(event: DragEvent, taskID: string) {
    if (!draggedEmployeeId) return;
    event.preventDefault();
    draggedOverTaskId = taskID;
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "move";
    }
  }

  function clearDragState() {
    draggedEmployeeId = "";
    draggedOverTaskId = "";
  }

  async function handleTaskDrop(task: CollaborativeTask, event: DragEvent) {
    if (!draggedEmployeeId) return;
    event.preventDefault();
    const employeeID = draggedEmployeeId;
    clearDragState();
    try {
      await reassignTask(task.id, employeeID);
      await load();
      if (selectedTaskId === task.id) {
        await loadTaskDetail(task.id);
      }
      const employeeName = employees.find((employee) => employee.id === employeeID)?.full_name || "Employee";
      toast.success(`${employeeName} assigned to ${task.title}`);
    } catch (err) {
      toast.danger(`Failed to assign task: ${String(err)}`);
    }
  }
</script>

<section class="metrics-grid">
  <article class="metric-card">
    <span class="label">Open Team Tasks</span>
    <strong>{teamTasks.filter((task) => !["completed", "archived"].includes(task.status || "")).length}</strong>
  </article>
  <article class="metric-card">
    <span class="label">Blocked</span>
    <strong>{teamTasks.filter((task) => task.status === "blocked").length}</strong>
  </article>
  <article class="metric-card">
    <span class="label">Overdue</span>
    <strong>{teamTasks.filter((task) => task.due_date && new Date(task.due_date) < new Date() && !["completed", "archived"].includes(task.status || "")).length}</strong>
  </article>
  <article class="metric-card">
    <span class="label">Active Employees</span>
    <strong>{workload.length}</strong>
  </article>
</section>

<section class="panel board-filters">
  <div class="filter-grid">
    <input bind:value={teamSearch} placeholder="Search tasks, assignees, or descriptions" />
    <select bind:value={teamFocusFilter}>
      <option value="all">All focus</option>
      <option value="active">Active only</option>
      <option value="assigned">Assigned only</option>
      <option value="unassigned">Unassigned</option>
      <option value="overdue">Overdue</option>
      <option value="blocked">Blocked</option>
    </select>
    <select bind:value={teamAssigneeFilter}>
      <option value="all">All assignees</option>
      <option value="unassigned">Unassigned only</option>
      {#each employees as employee}
        <option value={employee.id}>{employee.full_name}</option>
      {/each}
    </select>
  </div>
</section>

<section class="team-overview-grid">
  <article class="panel overview-panel">
    <div class="section-head">
      <h2>Admin Snapshot</h2>
      <span class="count">{projects.length} projects</span>
    </div>
    <div class="insight-grid">
      <div class="insight-row">
        <strong>{teamTasks.filter((task) => !task.assignee_employee_id && !["completed", "archived"].includes(task.status || "")).length}</strong>
        <span>unassigned tasks waiting for an owner</span>
      </div>
      <div class="insight-row">
        <strong>{teamTasks.filter((task) => ["high", "urgent"].includes(task.priority || "") && !["completed", "archived"].includes(task.status || "")).length}</strong>
        <span>high-priority tasks still in flight</span>
      </div>
      <div class="insight-row">
        <strong>{activeWorkload.filter((person) => person.openCount >= 3).length}</strong>
        <span>employees carrying three or more active tasks</span>
      </div>
    </div>
  </article>

  <article class="panel roster-panel">
    <div class="section-head">
      <div>
        <h2>Assignment Roster</h2>
        <p>Drag an employee card onto any task card to assign it. One employee can hold multiple tasks.</p>
      </div>
      <span class="count">{employees.length} people</span>
    </div>
    {#if employees.length === 0}
      <div class="empty">No employees available for assignment yet.</div>
    {:else}
      <div class="roster-strip">
        {#each workload as person}
          <button
            class="employee-chip"
            class:dragging={draggedEmployeeId === person.id}
            draggable="true"
            ondragstart={(event) => startEmployeeDrag(person.id, event)}
            ondragend={clearDragState}
            onclick={() => (teamAssigneeFilter = person.id)}
          >
            <strong>{person.name}</strong>
            <span>{person.department || "Team"}</span>
            <span>{person.openCount} active • {person.blockedCount} blocked</span>
          </button>
        {/each}
      </div>
    {/if}
  </article>
</section>

<section class="panel board-panel">
    <div class="section-head">
      <div>
        <h2>Team Board</h2>
        <p>Small task cards grouped by status. Click a card to open the full task modal.</p>
      </div>
      <span class="count">{filteredTeamTasks.length} visible</span>
    </div>
    {#if loading}
      <div class="board-sections-loading">
        <CardGridSkeleton statCards={0} panels={4} panelCols={4} panelRows={2} />
      </div>
    {:else if lastLoadFailed}
      <div class="empty">
        Team board is retrying connection to the collaborative workspace.
        {#if lastLoadError}
          <span class="load-error-detail">{lastLoadError}</span>
        {/if}
      </div>
    {:else}
    <div class="board-sections">
      {#each boardColumns as column}
        <section class="board-lane">
          <div class="board-lane-head">
            <h3>{column.label}</h3>
            <span>{boardTaskGroups[column.key]?.length || 0}</span>
          </div>
          {#if (boardTaskGroups[column.key]?.length || 0) === 0}
            <div class="empty compact lane-empty">No tasks in this lane.</div>
          {:else}
            <div class="task-gallery lane-gallery">
              {#each boardTaskGroups[column.key] as task}
                <button
                  class="task-shell board-card selectable"
                  class:selected={selectedTaskId === task.id}
                  class:drop-target={draggedOverTaskId === task.id}
                  onclick={() => onSelectTask(task.id)}
                  ondragover={(event) => allowTaskDrop(event, task.id)}
                  ondragleave={() => draggedOverTaskId = ""}
                  ondrop={(event) => handleTaskDrop(task, event)}
                >
                  <div class="task-shell-top">
                    <span class="task-status">{column.label}</span>
                    <span class="pill {task.priority || 'medium'}">{task.priority || "medium"}</span>
                  </div>
                  <div class="task-shell-body">
                    <h3>{task.title}</h3>
                    <p>{task.description || "No task details provided."}</p>
                    {#if task.status === "blocked" && task.blocked_reason}
                      <p class="blocker-copy">Blocked because: {task.blocked_reason}</p>
                    {/if}
                  </div>
                  <div class="task-shell-meta">
                    <span>{task.assignee_name || "Unassigned"}</span>
                    <span>{projectNameFor(task, projects)}</span>
                    <span>{dueLabel(task)}</span>
                  </div>
                </button>
              {/each}
            </div>
          {/if}
        </section>
      {/each}
    </div>
    {/if}
</section>

<style>
  h2,
  h3,
  p {
    margin: 0;
  }

  .section-head {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: center;
  }

  .section-head p,
  .label,
  .count {
    color: var(--text-secondary);
  }

  .panel,
  .metric-card {
    border: 1px solid var(--border);
    border-radius: 16px;
    background: var(--surface);
  }

  .metric-card {
    padding: 14px 16px;
  }

  .panel {
    padding: 18px;
  }

  .metrics-grid,
  .filter-grid,
  .team-overview-grid,
  .task-gallery,
  .insight-grid {
    display: grid;
    gap: 12px;
  }

  .metrics-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .team-overview-grid {
    grid-template-columns: minmax(320px, 0.7fr) minmax(0, 1.3fr);
  }

  .filter-grid {
    grid-template-columns: 2fr 1fr 1fr;
  }

  input,
  select,
  button {
    font: inherit;
  }

  input,
  select {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 10px 12px;
    background: var(--bg-base);
  }

  .board-filters {
    padding: 14px 18px;
  }

  .task-gallery {
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  }

  .task-shell,
  .employee-chip {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 14px;
    background: var(--bg-base);
    text-align: left;
  }

  .employee-chip {
    cursor: pointer;
  }

  .selected {
    border-color: var(--onyx);
    box-shadow: 0 0 0 1px var(--onyx);
  }

  .task-shell {
    display: grid;
    gap: 12px;
    min-height: 176px;
    transition: transform 140ms ease, box-shadow 140ms ease, border-color 140ms ease;
  }

  .task-shell:hover {
    transform: translateY(-1px);
    box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
  }

  .task-shell-top,
  .task-shell-meta,
  .board-lane-head {
    display: flex;
    justify-content: space-between;
    gap: 10px;
    align-items: center;
  }

  .task-shell-body {
    display: grid;
    gap: 8px;
    align-content: start;
  }

  .task-shell-body h3 {
    font-size: 16px;
    line-height: 1.3;
  }

  .task-shell-body p {
    color: var(--text-secondary);
    font-size: 13px;
    line-height: 1.45;
    display: -webkit-box;
    line-clamp: 3;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .task-shell-body .blocker-copy {
    color: #9f1239;
    font-weight: 500;
  }

  .task-shell-meta {
    flex-wrap: wrap;
    align-items: flex-start;
    color: var(--text-secondary);
    font-size: 12px;
  }

  .task-shell-meta span {
    padding: 4px 8px;
    border-radius: 999px;
    background: color-mix(in srgb, var(--bg-base) 82%, #e2e8f0 18%);
  }

  .task-status {
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--text-secondary);
  }

  .employee-chip span {
    margin-top: 6px;
    color: var(--text-secondary);
  }

  .pill {
    border-radius: 999px;
    padding: 6px 10px;
    font-size: 12px;
    text-transform: capitalize;
    background: #eef2ff;
    color: #312e81;
  }

  .pill.high,
  .pill.urgent {
    background: #fff1f2;
    color: #9f1239;
  }

  .board-sections {
    display: grid;
    gap: 18px;
    margin-top: 12px;
  }

  .board-sections-loading {
    margin-top: 12px;
  }

  .board-lane {
    border: 1px solid var(--border);
    border-radius: 14px;
    background: var(--bg-base);
    padding: 14px;
    display: grid;
    gap: 12px;
  }

  .lane-gallery {
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  }

  .lane-empty {
    border: 1px dashed var(--border);
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.45);
  }

  .drop-target {
    border-color: var(--onyx);
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--onyx) 20%, transparent);
    background: color-mix(in srgb, var(--bg-base) 88%, var(--onyx) 12%);
  }

  .employee-chip {
    display: grid;
    gap: 2px;
    background: color-mix(in srgb, var(--bg-base) 88%, #dbeafe 12%);
    min-width: 210px;
    flex: 0 0 210px;
  }

  .employee-chip.dragging {
    opacity: 0.55;
  }

  .roster-strip {
    display: flex;
    gap: 12px;
    overflow-x: auto;
    padding-bottom: 4px;
  }

  .overview-panel {
    background: color-mix(in srgb, var(--surface) 88%, #ecfccb 12%);
  }

  .insight-row {
    display: grid;
    gap: 4px;
    padding: 12px;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--bg-base);
  }

  .insight-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .employee-chip strong {
    display: block;
  }

  .empty {
    padding: 24px 8px;
    color: var(--text-secondary);
    text-align: center;
  }

  .empty.compact {
    padding: 12px 8px;
  }

  @media (max-width: 1100px) {
    .metrics-grid,
    .filter-grid,
    .team-overview-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
