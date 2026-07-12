<script lang="ts">
  import { run, self } from 'svelte/legacy';

  import { createEventDispatcher } from "svelte";
  import { toast } from "$lib/stores/toasts";
  import {
    createTask,
    listEmployeeProfiles,
    listProjectMembers,
    listProjects,
    type CollaborativeProject,
    type CollaborativeTask,
    type EmployeeProfile,
    type ProjectMember,
  } from "$lib/api/collaboration";

  interface Props {
    open?: boolean;
    title?: string;
    subtitle?: string;
    defaults?: Partial<CollaborativeTask> & { seed_title?: string };
  }

  let {
    open = false,
    title = "Create Task",
    subtitle = "",
    defaults = {}
  }: Props = $props();

  const dispatch = createEventDispatcher();

  let loading = $state(false);
  let saving = $state(false);
  let employees: EmployeeProfile[] = $state([]);
  let projects: CollaborativeProject[] = $state([]);
  // Wave 9.4 B3.1: when a project is in context, the assignee list scopes to
  // that project's members (adding/removing a member visibly changes who can
  // be assigned here). With no project context we fall back to all employees
  // so cross-context task creation (e.g. from a customer with no project)
  // still works.
  let scopedMembers: ProjectMember[] = $state([]);
  let scopedMembersLoadedFor = $state("");

  let taskTitle = $state("");
  let description = $state("");
  let priority = $state("medium");
  let dueDate = $state("");
  let assigneeEmployeeId = $state("");
  let projectId = $state("");

  let assigneeOptions = $derived(
    projectId && scopedMembersLoadedFor === projectId && scopedMembers.length > 0
      ? scopedMembers.map((member) => ({
          id: member.employee_id,
          label: member.employee_name || "Unknown",
          sub: member.role || "",
        }))
      : employees.map((employee) => ({ id: employee.id, label: employee.full_name, sub: employee.department || "" })),
  );

  async function loadScopedMembers(id: string) {
    if (!id) {
      scopedMembers = [];
      scopedMembersLoadedFor = "";
      return;
    }
    try {
      scopedMembers = await listProjectMembers(id);
      scopedMembersLoadedFor = id;
    } catch {
      scopedMembers = [];
      scopedMembersLoadedFor = "";
    }
  }


  async function initializeModal() {
    if (loading || saving) return;
    loading = true;
    try {
      const [employeeRows, projectRows] = await Promise.all([
        listEmployeeProfiles(true),
        listProjects(true),
      ]);
      employees = employeeRows;
      projects = projectRows;

      taskTitle = defaults.seed_title || defaults.title || "";
      description = defaults.description || "";
      priority = defaults.priority || "medium";
      dueDate = defaults.due_date ? new Date(defaults.due_date).toISOString().slice(0, 10) : "";
      assigneeEmployeeId = defaults.assignee_employee_id || "";
      projectId = defaults.project_id || "";
    } catch (err) {
      toast.danger(`Failed to prepare task form: ${String(err)}`);
    } finally {
      loading = false;
    }
  }

  function close() {
    dispatch("close");
  }

  // Switching the project inside the modal re-scopes the assignee list; drop
  // a stale selection that no longer belongs to the newly-scoped set.
  run(() => {
    if (open && projectId !== scopedMembersLoadedFor) {
      void loadScopedMembers(projectId).then(() => {
        if (assigneeEmployeeId && !assigneeOptions.some((option) => option.id === assigneeEmployeeId)) {
          assigneeEmployeeId = "";
        }
      });
    }
  });

  async function save() {
    if (!taskTitle.trim()) {
      toast.warning("Task title is required");
      return;
    }

    saving = true;
    try {
      const created = await createTask({
        title: taskTitle.trim(),
        description: description.trim(),
        priority,
        due_date: dueDate ? new Date(`${dueDate}T09:00:00`).toISOString() : undefined,
        assignee_employee_id: assigneeEmployeeId || undefined,
        project_id: projectId || undefined,
        customer_id: defaults.customer_id,
        opportunity_id: defaults.opportunity_id,
        order_id: defaults.order_id,
      });
      toast.success("Task created");
      dispatch("created", created);
      close();
    } catch (err) {
      toast.danger(`Failed to create task: ${String(err)}`);
    } finally {
      saving = false;
    }
  }

  function handleBackdropKeydown(event: KeyboardEvent) {
    if (event.currentTarget !== event.target) return;
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      close();
    }
  }
  run(() => {
    if (open) {
      initializeModal();
    }
  });
</script>

<svelte:window onkeydown={(event) => event.key === "Escape" && open && close()} />

{#if open}
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    onclick={self(close)}
    onkeydown={handleBackdropKeydown}
  >
    <div
      class="modal-card"
      role="dialog"
      aria-modal="true"
      tabindex="-1"
      aria-labelledby="context-task-modal-title"
    >
      <div class="modal-head">
        <div>
          <h3 id="context-task-modal-title">{title}</h3>
          {#if subtitle}
            <p>{subtitle}</p>
          {/if}
        </div>
        <button class="close-btn" onclick={close} aria-label="Close task modal">Close</button>
      </div>

      {#if loading}
        <div class="empty">Preparing task form...</div>
      {:else}
        <div class="form-grid">
          <input bind:value={taskTitle} placeholder="Task title" />
          <select bind:value={priority}>
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
            <option value="urgent">Urgent</option>
          </select>
          <input bind:value={dueDate} type="date" />
          <select bind:value={projectId}>
            <option value="">No project</option>
            {#each projects as project}
              <option value={project.id}>{project.name}</option>
            {/each}
          </select>
          <select bind:value={assigneeEmployeeId}>
            <option value="">Assign later</option>
            {#each assigneeOptions as option}
              <option value={option.id}>{option.label}{option.sub ? ` • ${option.sub}` : ""}</option>
            {/each}
          </select>
        </div>

        <textarea bind:value={description} rows="4" placeholder="Describe the work, expected outcome, blockers, or context"></textarea>

        <div class="actions">
          <button class="ghost" onclick={close} disabled={saving}>Cancel</button>
          <button class="primary" onclick={save} disabled={saving}>{saving ? "Creating..." : "Create Task"}</button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1200;
    padding: 20px;
  }

  .modal-card {
    width: min(720px, 100%);
    border-radius: 18px;
    border: 1px solid var(--border);
    background: var(--surface);
    padding: 20px;
    display: grid;
    gap: 16px;
  }

  .modal-head,
  .actions {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: center;
  }

  h3,
  p {
    margin: 0;
  }

  p,
  .empty {
    color: var(--text-secondary);
  }

  .form-grid {
    display: grid;
    grid-template-columns: 2fr repeat(4, 1fr);
    gap: 12px;
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

  button {
    border-radius: 10px;
    padding: 10px 14px;
  }

  .close-btn,
  .ghost {
    border: 1px solid var(--border);
    background: white;
  }

  .primary {
    border: none;
    background: var(--onyx);
    color: white;
  }

  @media (max-width: 900px) {
    .form-grid {
      grid-template-columns: 1fr;
    }

    .modal-head,
    .actions {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
