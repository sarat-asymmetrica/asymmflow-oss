<script lang="ts">
  import type { CollaborativeProject } from "$lib/api/collaboration";

  interface AssigneeOption {
    id: string;
    label: string;
    sub: string;
  }

  interface Props {
    projects: CollaborativeProject[];
    assigneeOptions: AssigneeOption[];
    savingTask: boolean;
    taskTitle: string;
    taskDescription: string;
    taskPriority: string;
    taskDueDate: string;
    selectedProjectForTask: string;
    selectedAssignee: string;
    onCreate: () => void;
  }

  let {
    projects,
    assigneeOptions,
    savingTask,
    taskTitle = $bindable(),
    taskDescription = $bindable(),
    taskPriority = $bindable(),
    taskDueDate = $bindable(),
    selectedProjectForTask = $bindable(),
    selectedAssignee = $bindable(),
    onCreate,
  }: Props = $props();
</script>

<section class="composer">
  <div class="section-head">
    <div>
      <h2>Create Task</h2>
      <p>Assign work to a person, attach it to a project, and make it visible across devices.</p>
    </div>
  </div>
  <div class="composer-grid">
    <input bind:value={taskTitle} placeholder="Task title" />
    <select bind:value={taskPriority}>
      <option value="low">Low</option>
      <option value="medium">Medium</option>
      <option value="high">High</option>
      <option value="urgent">Urgent</option>
    </select>
    <input bind:value={taskDueDate} type="date" />
    <select bind:value={selectedProjectForTask}>
      <option value="">No project</option>
      {#each projects as project}
        <option value={project.id}>{project.name}</option>
      {/each}
    </select>
    <select bind:value={selectedAssignee}>
      <option value="">Assign to me</option>
      {#each assigneeOptions as option}
        <option value={option.id}>{option.label}{option.sub ? ` • ${option.sub}` : ""}</option>
      {/each}
    </select>
  </div>
  <textarea bind:value={taskDescription} rows="3" placeholder="Context, blockers, or expected outcome"></textarea>
  <div class="actions">
    <button class="primary" onclick={onCreate} disabled={savingTask}>Create Task</button>
  </div>
</section>

<style>
  .composer {
    border: 1px solid var(--border);
    border-radius: 16px;
    background: var(--surface);
    padding: 18px;
  }

  h2,
  p {
    margin: 0;
  }

  .section-head {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: center;
  }

  .section-head p {
    color: var(--text-secondary);
  }

  .composer-grid {
    display: grid;
    gap: 12px;
    grid-template-columns: 2fr repeat(4, 1fr);
    margin-top: 12px;
    margin-bottom: 12px;
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

  @media (max-width: 1100px) {
    .composer-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
