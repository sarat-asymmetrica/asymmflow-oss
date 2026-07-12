<script lang="ts">
  import type { CollaborativeProject, CollaborativeTask } from "$lib/api/collaboration";
  import { projectNameFor, dueLabel, isTaskOverdue } from "./workHubHelpers";

  interface Props {
    myTasks: CollaborativeTask[];
    projects: CollaborativeProject[];
    loading: boolean;
    selectedTaskId: string;
    onSelectTask: (taskID: string) => void;
  }

  let { myTasks, projects, loading, selectedTaskId, onSelectTask }: Props = $props();

  // B5a: My Work filters, mirroring the Team Board's filter panel.
  let myWorkSearch = $state("");
  let myWorkFocusFilter = $state("all");
  let myWorkIncludeCompleted = $state(false);

  // B5a: My Work — overdue-first, with the same filter/completed-toggle
  // language as Team Board so the worker's surface is at least as capable.
  let filteredMyTasks = $derived(myTasks.filter((task) => {
    const matchesSearch = !myWorkSearch.trim() || [task.title, task.description, projectNameFor(task, projects)]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(myWorkSearch.trim().toLowerCase()));
    const overdue = isTaskOverdue(task);
    const matchesFocus = myWorkFocusFilter === "all"
      || (myWorkFocusFilter === "active" && !["completed", "archived"].includes(task.status || ""))
      || (myWorkFocusFilter === "overdue" && overdue)
      || (myWorkFocusFilter === "blocked" && task.status === "blocked");
    const matchesCompleted = myWorkIncludeCompleted || !["completed", "archived"].includes(task.status || "");
    return matchesSearch && matchesFocus && matchesCompleted;
  }).sort((a, b) => {
    const overdueRank = Number(isTaskOverdue(b)) - Number(isTaskOverdue(a));
    if (overdueRank !== 0) return overdueRank;
    const aDue = a.due_date ? new Date(a.due_date).getTime() : Number.POSITIVE_INFINITY;
    const bDue = b.due_date ? new Date(b.due_date).getTime() : Number.POSITIVE_INFINITY;
    return aDue - bDue;
  }));
  let myWorkOverdueTasks = $derived(filteredMyTasks.filter((task) => isTaskOverdue(task)));
  let myWorkOtherTasks = $derived(filteredMyTasks.filter((task) => !isTaskOverdue(task)));
</script>

<section class="metrics-grid">
  <article class="metric-card">
    <span class="label">Overdue</span>
    <strong>{myWorkOverdueTasks.length}</strong>
  </article>
  <article class="metric-card">
    <span class="label">Visible</span>
    <strong>{filteredMyTasks.length}</strong>
  </article>
  <article class="metric-card">
    <span class="label">Total Assigned</span>
    <strong>{myTasks.length}</strong>
  </article>
</section>

<section class="panel board-filters">
  <div class="filter-grid">
    <input bind:value={myWorkSearch} placeholder="Search your tasks" />
    <select bind:value={myWorkFocusFilter}>
      <option value="all">All focus</option>
      <option value="active">Active only</option>
      <option value="overdue">Overdue</option>
      <option value="blocked">Blocked</option>
    </select>
    <label class="toggle-field">
      <input type="checkbox" bind:checked={myWorkIncludeCompleted} />
      <span>Show completed</span>
    </label>
  </div>
</section>

<section class="panel task-gallery-panel">
  <div class="section-head">
    <div>
      <h2>My Work</h2>
      <p>Overdue work first. Open any card to inspect comments, assignee, and history.</p>
    </div>
    <span class="count">{filteredMyTasks.length} of {myTasks.length}</span>
  </div>
  {#if loading}
    <div class="empty">Loading your work queue...</div>
  {:else if filteredMyTasks.length === 0}
    <div class="empty">No tasks match these filters.</div>
  {:else}
    {#if myWorkOverdueTasks.length > 0}
      <p class="task-bucket-head">Overdue ({myWorkOverdueTasks.length})</p>
      <div class="task-gallery two-up">
        {#each myWorkOverdueTasks as task (task.id)}
          {@render myWorkCard(task)}
        {/each}
      </div>
    {/if}
    {#if myWorkOtherTasks.length > 0}
      {#if myWorkOverdueTasks.length > 0}
        <p class="task-bucket-head">Everything else</p>
      {/if}
      <div class="task-gallery two-up">
        {#each myWorkOtherTasks as task (task.id)}
          {@render myWorkCard(task)}
        {/each}
      </div>
    {/if}
  {/if}
</section>

{#snippet myWorkCard(task: CollaborativeTask)}
  <button class="task-shell selectable" class:selected={selectedTaskId === task.id} onclick={() => onSelectTask(task.id)}>
    <div class="task-shell-top">
      <span class="task-status">{(task.status || "open").replaceAll("_", " ")}</span>
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
      <span>{projectNameFor(task, projects)}</span>
      <span>{dueLabel(task)}</span>
      <span>{task.assignee_name || "Unassigned"}</span>
    </div>
  </button>
{/snippet}

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
  .task-gallery {
    display: grid;
    gap: 12px;
  }

  .metrics-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
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

  .toggle-field {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text-secondary);
    font-size: 0.9rem;
  }

  .task-gallery-panel {
    display: grid;
    gap: 16px;
  }

  .task-bucket-head {
    margin: 4px 0 0;
    font-weight: 600;
    font-size: 0.85rem;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .task-gallery.two-up {
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  }

  .task-shell {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 14px;
    background: var(--bg-base);
    text-align: left;
    display: grid;
    gap: 12px;
    min-height: 176px;
    transition: transform 140ms ease, box-shadow 140ms ease, border-color 140ms ease;
    cursor: pointer;
  }

  .task-shell:hover {
    transform: translateY(-1px);
    box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
  }

  .selected {
    border-color: var(--onyx);
    box-shadow: 0 0 0 1px var(--onyx);
  }

  .task-shell-top,
  .task-shell-meta {
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

  .empty {
    padding: 24px 8px;
    color: var(--text-secondary);
    text-align: center;
  }

  @media (max-width: 1100px) {
    .metrics-grid,
    .filter-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
