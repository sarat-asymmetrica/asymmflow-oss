<script lang="ts">
  import WabiSpinner from "$lib/components/ui/WabiSpinner.svelte";
  import type { CollaborativeProject, CollaborativeTask, EmployeeProfile, ProjectMember, TaskActivityItem } from "$lib/api/collaboration";
  import { formatDate } from "./workHubHelpers";
  import WorkHubMemberManagementPanel from "./WorkHubMemberManagementPanel.svelte";

  interface MemberDraft {
    role: string;
    allocation: number;
  }

  interface Props {
    projectsShowArchived: boolean;
    visibleProjects: CollaborativeProject[];
    loadingArchivedProjects: boolean;
    onToggleArchivedProjectsView: () => void;
    projectComposerEl: HTMLElement | null;
    projectName: string;
    projectType: string;
    projectDescription: string;
    projectCustomerName: string;
    projectPOCName: string;
    projectPOCEmail: string;
    projectPOCPhone: string;
    savingProject: boolean;
    onCreateProject: () => void;
    selectedProjectId: string;
    projectTaskCounts: Record<string, number>;
    onSelectProject: (projectID: string) => void;
    selectedProject: CollaborativeProject | null;
    editingProject: boolean;
    projectEditName: string;
    projectEditType: string;
    projectEditDescription: string;
    savingProjectAdmin: boolean;
    onStartProjectEdit: () => void;
    onCancelProjectEdit: () => void;
    onSaveProjectEdit: () => void;
    projectStats: { open: number; blocked: number; completed: number; members: number };
    projectMembers: ProjectMember[];
    availableProjectMembers: EmployeeProfile[];
    projectContextLoading: boolean;
    memberSelections: string[];
    newMemberRole: string;
    newMemberAllocation: number;
    savingMember: boolean;
    savingMemberId: string;
    highlightMemberStep: boolean;
    memberStepEl: HTMLElement | null;
    memberDraft: (member: ProjectMember) => MemberDraft;
    onToggleMemberSelection: (employeeID: string) => void;
    onAddMembers: () => void;
    onSetMemberDraftField: (employeeID: string, field: "role" | "allocation", value: string | number) => void;
    onSaveMember: (member: ProjectMember) => void;
    projectTasks: CollaborativeTask[];
    selectedTaskId: string;
    onSelectTask: (taskID: string) => void;
    projectActivity: TaskActivityItem[];
    canManageProjects: boolean;
    canDeleteProject: boolean;
    onRestoreProject: () => void;
    onDeleteProject: () => void;
    onApplyProjectAdminAction: (action: "archive" | "shelve") => void;
  }

  let {
    projectsShowArchived,
    visibleProjects,
    loadingArchivedProjects,
    onToggleArchivedProjectsView,
    projectComposerEl = $bindable(null),
    projectName = $bindable(),
    projectType = $bindable(),
    projectDescription = $bindable(),
    projectCustomerName = $bindable(),
    projectPOCName = $bindable(),
    projectPOCEmail = $bindable(),
    projectPOCPhone = $bindable(),
    savingProject,
    onCreateProject,
    selectedProjectId,
    projectTaskCounts,
    onSelectProject,
    selectedProject,
    editingProject,
    projectEditName = $bindable(),
    projectEditType = $bindable(),
    projectEditDescription = $bindable(),
    savingProjectAdmin,
    onStartProjectEdit,
    onCancelProjectEdit,
    onSaveProjectEdit,
    projectStats,
    projectMembers,
    availableProjectMembers,
    projectContextLoading,
    memberSelections = $bindable(),
    newMemberRole = $bindable(),
    newMemberAllocation = $bindable(),
    savingMember,
    savingMemberId,
    highlightMemberStep,
    memberStepEl = $bindable(null),
    memberDraft,
    onToggleMemberSelection,
    onAddMembers,
    onSetMemberDraftField,
    onSaveMember,
    projectTasks,
    selectedTaskId,
    onSelectTask,
    projectActivity,
    canManageProjects,
    canDeleteProject,
    onRestoreProject,
    onDeleteProject,
    onApplyProjectAdminAction,
  }: Props = $props();
</script>

<section class="workspace">
  <article class="panel">
    <div class="section-head">
      <div>
        <h2>Projects</h2>
        <p>{projectsShowArchived ? "Archived, shelved, and deleted projects." : "Customer, internal, and admin work containers."}</p>
      </div>
      <span class="count">{visibleProjects.length}</span>
    </div>

    <div class="actions archived-toggle-row">
      <button onclick={onToggleArchivedProjectsView} disabled={loadingArchivedProjects}>
        {projectsShowArchived ? "Show Active Projects" : "Show Archived"}
      </button>
    </div>

    {#if !projectsShowArchived}
      <div class="project-composer" bind:this={projectComposerEl}>
        <input bind:value={projectName} placeholder="Project name" />
        <select bind:value={projectType}>
          <option value="internal">Internal</option>
          <option value="customer">Customer</option>
          <option value="admin">Admin</option>
        </select>
        <textarea bind:value={projectDescription} rows="2" placeholder="What is this project for?"></textarea>
        {#if projectType === "customer"}
          <div class="composer-grid">
            <input bind:value={projectCustomerName} placeholder="Customer name" />
            <input bind:value={projectPOCName} placeholder="POC name" />
            <input bind:value={projectPOCEmail} placeholder="POC email" type="email" />
            <input bind:value={projectPOCPhone} placeholder="POC phone" />
          </div>
        {/if}
        <div class="actions">
          <button class="primary" onclick={onCreateProject} disabled={savingProject}>Create Project</button>
        </div>
      </div>
    {/if}

    <div class="project-list">
      {#if loadingArchivedProjects}
        <div class="empty compact">Loading archived projects...</div>
      {:else if visibleProjects.length === 0}
        <div class="empty compact">{projectsShowArchived ? "No archived projects." : "No projects yet."}</div>
      {:else}
        {#each visibleProjects as project}
          <button class="project-row selectable" class:selected={selectedProjectId === project.id} onclick={() => onSelectProject(project.id)}>
            <div>
              <strong>{project.name}</strong>
              <div class="meta">{project.project_type || "internal"} • {project.status || "active"}</div>
            </div>
            <span class="count">{projectTaskCounts[project.id] ?? 0}</span>
          </button>
        {/each}
      {/if}
    </div>
  </article>

  <article class="panel detail-panel">
    <div class="section-head">
      <h2>{selectedProject?.name || "Project Detail"}</h2>
      {#if selectedProject}
        <span class="count">{selectedProject.project_type || "internal"}</span>
      {/if}
    </div>

    {#if !selectedProject}
      <div class="empty">Choose a project to view members and project-linked tasks.</div>
    {:else}
      <div class="detail-stack">
      <div class="detail-hero">
        {#if editingProject}
          <div class="project-edit-form">
            <input bind:value={projectEditName} placeholder="Project name" />
            <select bind:value={projectEditType}>
              <option value="internal">Internal</option>
              <option value="customer">Customer</option>
              <option value="admin">Admin</option>
            </select>
            <textarea bind:value={projectEditDescription} rows="2" placeholder="What is this project for?"></textarea>
            <div class="actions">
              <button class="primary" onclick={onSaveProjectEdit} disabled={savingProjectAdmin}>Save Changes</button>
              <button onclick={onCancelProjectEdit} disabled={savingProjectAdmin}>Cancel</button>
            </div>
          </div>
        {:else}
          <div>
            <h3>{selectedProject.name}</h3>
            <p>{selectedProject.description || "No project description provided."}</p>
            {#if selectedProject.project_type === "customer" && (selectedProject.customer_name || selectedProject.customer_poc_name)}
              <div class="task-meta chips">
                {#if selectedProject.customer_name}<span>Customer: {selectedProject.customer_name}</span>{/if}
                {#if selectedProject.customer_poc_name}<span>POC: {selectedProject.customer_poc_name}</span>{/if}
                {#if selectedProject.customer_poc_email}<span>{selectedProject.customer_poc_email}</span>{/if}
                {#if selectedProject.customer_poc_phone}<span>{selectedProject.customer_poc_phone}</span>{/if}
              </div>
            {/if}
          </div>
          <button class="edit-project" onclick={onStartProjectEdit}>Edit</button>
        {/if}
      </div>
      <div class="metrics-grid compact">
        <article class="metric-card">
          <span class="label">Open</span>
          <strong>{projectStats.open}</strong>
        </article>
        <article class="metric-card">
          <span class="label">Blocked</span>
          <strong>{projectStats.blocked}</strong>
        </article>
        <article class="metric-card">
          <span class="label">Completed</span>
          <strong>{projectStats.completed}</strong>
        </article>
        <article class="metric-card">
          <span class="label">Members</span>
          <strong>{projectStats.members}</strong>
        </article>
      </div>

        <WorkHubMemberManagementPanel
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
          onToggleMemberSelection={onToggleMemberSelection}
          onAddMembers={onAddMembers}
          onSetMemberDraftField={onSetMemberDraftField}
          onSaveMember={onSaveMember}
        />

        <div class="subpanel">
          <div class="section-head">
            <h3>Project Tasks</h3>
            <span class="count">{projectTasks.length}</span>
          </div>
          {#if projectContextLoading}
            <div class="section-loading">
              <WabiSpinner size="md" tempo="calm" />
              <p>Loading project tasks...</p>
            </div>
          {:else if projectTasks.length === 0}
            <div class="empty compact">No tasks attached to this project yet.</div>
          {:else}
            <div class="timeline">
              {#each projectTasks as task}
                <button class="timeline-item selectable" class:selected={selectedTaskId === task.id} onclick={() => onSelectTask(task.id)}>
                  <strong>{task.title}</strong>
                  <p>{task.assignee_name || "Unassigned"} • {task.status || "open"}</p>
                </button>
              {/each}
            </div>
          {/if}
        </div>

        <div class="subpanel">
          <div class="section-head">
            <h3>Project Activity</h3>
            <span class="count">{projectActivity.length}</span>
          </div>
          {#if projectContextLoading}
            <div class="section-loading">
              <WabiSpinner size="md" tempo="calm" />
              <p>Loading project activity...</p>
            </div>
          {:else if projectActivity.length === 0}
            <div class="empty compact">No project activity yet.</div>
          {:else}
            <div class="timeline">
              {#each projectActivity as item}
                <div class="timeline-item">
                  <strong>{item.employee_name || "System"}</strong>
                  <p>{item.detail}</p>
                  <span class="meta">{formatDate(item.created_at)}</span>
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <div class="subpanel admin-panel">
          <div class="section-head">
            <h3>Project Administration</h3>
            <span class="count">{selectedProject.status || "active"}</span>
          </div>
          {#if canManageProjects || canDeleteProject}
            {#if ["archived", "shelved", "deleted"].includes(String(selectedProject.status || "").toLowerCase())}
              <p class="admin-hint">
                This project is {selectedProject.status}. Restoring reopens it as active with all tasks and history intact.
              </p>
              <div class="actions">
                {#if canManageProjects}
                  <button class="primary" onclick={onRestoreProject} disabled={savingProjectAdmin}>Restore to Active</button>
                {/if}
                {#if canDeleteProject}
                  <button class="danger" onclick={onDeleteProject} disabled={savingProjectAdmin}>Delete</button>
                {/if}
              </div>
            {:else}
              <p class="admin-hint">
                Archive closes a finished project, shelve pauses it, delete retires it — all three
                keep tasks and history attached, require admin permission, and prompt for a reason.
              </p>
              <div class="actions">
                {#if canManageProjects}
                  <button onclick={() => onApplyProjectAdminAction("archive")} disabled={savingProjectAdmin}>Archive</button>
                  <button onclick={() => onApplyProjectAdminAction("shelve")} disabled={savingProjectAdmin}>Shelve</button>
                {/if}
                {#if canDeleteProject}
                  <button class="danger" onclick={onDeleteProject} disabled={savingProjectAdmin}>Delete</button>
                {/if}
              </div>
            {/if}
          {:else}
            <p class="admin-hint">
              You don't have permission to archive, shelve, restore, or delete projects.
            </p>
          {/if}
        </div>
      </div>
    {/if}
  </article>
</section>

<style>
  h2,
  h3,
  p {
    margin: 0;
  }

  .section-head,
  .detail-hero {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: center;
  }

  .count,
  .meta,
  .label,
  .section-head p {
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

  .composer-grid,
  .workspace,
  .metrics-grid {
    display: grid;
    gap: 12px;
  }

  .composer-grid {
    grid-template-columns: 2fr repeat(4, 1fr);
    margin-top: 12px;
    margin-bottom: 12px;
  }

  .workspace {
    grid-template-columns: minmax(0, 1.1fr) minmax(320px, 0.9fr);
    align-items: start;
  }

  .metrics-grid.compact {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  input,
  select,
  textarea,
  button {
    font: inherit;
  }

  input,
  select,
  textarea {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 10px 12px;
    background: var(--bg-base);
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 12px;
  }

  .primary {
    border: none;
    border-radius: 10px;
    padding: 10px 14px;
    background: var(--onyx);
    color: white;
  }

  .danger {
    background: #fff1f2;
    color: #9f1239;
  }

  .detail-stack,
  .timeline,
  .project-list {
    display: grid;
    gap: 12px;
  }

  .timeline-item,
  .project-row {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 14px;
    background: var(--bg-base);
    text-align: left;
  }

  .timeline-item.selectable,
  .project-row.selectable {
    cursor: pointer;
  }

  .selected {
    border-color: var(--onyx);
    box-shadow: 0 0 0 1px var(--onyx);
  }

  .timeline-item p {
    margin-top: 6px;
    color: var(--text-secondary);
  }

  .timeline-item strong,
  .project-row strong {
    display: block;
  }

  .task-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    color: var(--text-secondary);
    font-size: 12px;
  }

  .task-meta.chips span {
    padding: 6px 10px;
    border-radius: 999px;
    background: color-mix(in srgb, var(--bg-base) 85%, #e2e8f0 15%);
  }

  .subpanel {
    border-top: 1px solid var(--border);
    padding-top: 16px;
    display: grid;
    gap: 12px;
    align-content: start;
    grid-auto-rows: max-content;
  }

  .project-edit-form {
    flex: 1;
    display: grid;
    gap: 10px;
  }

  .edit-project {
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 8px 12px;
    background: transparent;
    color: var(--text-secondary);
    flex-shrink: 0;
  }

  .admin-hint {
    margin: 0;
    color: var(--text-secondary);
    font-size: 0.88rem;
    line-height: 1.5;
  }

  .project-composer {
    display: grid;
    gap: 10px;
  }

  .archived-toggle-row {
    margin-bottom: 4px;
  }

  .section-loading {
    display: grid;
    place-items: center;
    gap: 10px;
    padding: 24px;
    color: var(--text-secondary);
    text-align: center;
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
    .workspace,
    .composer-grid,
    .metrics-grid.compact {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 900px) {
    .detail-hero,
    .section-head {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
