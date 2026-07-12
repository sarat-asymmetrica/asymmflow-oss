<script lang="ts">
  import Modal from "$lib/components/layout/Modal.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import WabiSpinner from "$lib/components/ui/WabiSpinner.svelte";
  import type { CollaborativeProject, CollaborativeTask, TaskActivityItem, TaskCommentItem } from "$lib/api/collaboration";
  import { formatDate, projectNameFor, dueLabel } from "./workHubHelpers";

  interface AssigneeOption {
    id: string;
    label: string;
    sub: string;
  }

  interface Props {
    open: boolean;
    taskDetailLoading: boolean;
    selectedTask: CollaborativeTask | null;
    selectedTaskId: string;
    myTasks: CollaborativeTask[];
    teamTasks: CollaborativeTask[];
    projectTasks: CollaborativeTask[];
    projects: CollaborativeProject[];
    taskComments: TaskCommentItem[];
    taskActivity: TaskActivityItem[];
    taskDetailAssigneeOptions: AssigneeOption[];
    selectedTaskTitleDraft: string;
    selectedTaskDescriptionDraft: string;
    selectedTaskPriorityDraft: string;
    selectedTaskBlockerDraft: string;
    selectedTaskAssigneeDraft: string;
    selectedTaskDueDateDraft: string;
    commentDraft: string;
    confirmDeleteTask: boolean;
    savingTaskDetails: boolean;
    savingAssignment: boolean;
    savingDueDate: boolean;
    savingComment: boolean;
    deletingTask: boolean;
    onSaveTaskDetails: () => void;
    onReassign: () => void;
    onUpdateDueDate: () => void;
    onBlock: () => void;
    onChangeStatus: (task: CollaborativeTask, status: string) => void;
    onDeleteTask: () => void;
    onAddComment: () => void;
    onClose: () => void;
  }

  let {
    open = $bindable(),
    taskDetailLoading,
    selectedTask,
    selectedTaskId,
    myTasks,
    teamTasks,
    projectTasks,
    projects,
    taskComments,
    taskActivity,
    taskDetailAssigneeOptions,
    selectedTaskTitleDraft = $bindable(),
    selectedTaskDescriptionDraft = $bindable(),
    selectedTaskPriorityDraft = $bindable(),
    selectedTaskBlockerDraft = $bindable(),
    selectedTaskAssigneeDraft = $bindable(),
    selectedTaskDueDateDraft = $bindable(),
    commentDraft = $bindable(),
    confirmDeleteTask,
    savingTaskDetails,
    savingAssignment,
    savingDueDate,
    savingComment,
    deletingTask,
    onSaveTaskDetails,
    onReassign,
    onUpdateDueDate,
    onBlock,
    onChangeStatus,
    onDeleteTask,
    onAddComment,
    onClose,
  }: Props = $props();

  let modalTask = $derived(selectedTask || [...myTasks, ...teamTasks, ...projectTasks].find((task) => task.id === selectedTaskId) || null);
</script>

<Modal bind:open title={modalTask ? modalTask.title : "Task Detail"} size="lg" on:close={onClose}>
  {#if taskDetailLoading && !modalTask}
    <div class="modal-loading-state">
      <WabiSpinner size="lg" tempo="calm" />
      <p>Loading task history...</p>
    </div>
  {:else if !modalTask}
    <div class="empty compact">Choose a task to inspect comments and history.</div>
  {:else}
    <div class="task-modal-body">
      <div class="detail-hero">
        <div>
          <p class="eyebrow">{(modalTask.status || "open").replaceAll("_", " ")}</p>
          <h3>{modalTask.title}</h3>
          <p>{modalTask.description || "No task details provided."}</p>
        </div>
        <span class="pill {modalTask.priority || 'medium'}">{modalTask.priority || "medium"}</span>
      </div>

      <div class="task-meta chips">
        <span>Assignee: {modalTask.assignee_name || "Unassigned"}</span>
        <span>Project: {projectNameFor(modalTask, projects)}</span>
        <span>{dueLabel(modalTask)}</span>
        <span>Created by: {modalTask.creator_name || "System"}</span>
      </div>

      <div class="subpanel modal-subpanel">
        <div class="section-head">
          <h3>Task Details</h3>
          <Button variant="secondary" size="sm" on:click={onSaveTaskDetails} disabled={savingTaskDetails}>
            {savingTaskDetails ? "Saving..." : "Save Details"}
          </Button>
        </div>
        <label class="full-width">
          <span>Title</span>
          <input bind:value={selectedTaskTitleDraft} placeholder="Task title" />
        </label>
        <div class="form-grid compact-grid">
          <label>
            <span>Priority</span>
            <select bind:value={selectedTaskPriorityDraft}>
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
              <option value="urgent">Urgent</option>
            </select>
          </label>
        </div>
        <label class="full-width">
          <span>Description</span>
          <textarea bind:value={selectedTaskDescriptionDraft} rows="3" placeholder="Context, outcome, or working notes"></textarea>
        </label>
      </div>

      <div class="subpanel modal-subpanel">
        <div class="section-head">
          <h3>Blocker</h3>
          <span class="count">{modalTask.status === "blocked" ? "Active" : "Optional"}</span>
        </div>
        <textarea
          bind:value={selectedTaskBlockerDraft}
          rows="3"
          placeholder="What is blocking this task? Include the dependency, owner, or next step."
        ></textarea>
      </div>

      <div class="editor-grid">
        <select bind:value={selectedTaskAssigneeDraft}>
          <option value="">Unassigned</option>
          {#each taskDetailAssigneeOptions as option}
            <option value={option.id}>{option.label}{option.sub ? ` • ${option.sub}` : ""}</option>
          {/each}
        </select>
        <Button variant="secondary" size="sm" on:click={onReassign} disabled={savingAssignment}>
          {savingAssignment ? "Saving..." : "Save Assignee"}
        </Button>
        <input bind:value={selectedTaskDueDateDraft} type="date" />
        <Button variant="secondary" size="sm" on:click={onUpdateDueDate} disabled={savingDueDate}>
          {savingDueDate ? "Saving..." : "Save Due Date"}
        </Button>
      </div>

      <div class="task-actions">
        {#if modalTask.status === "blocked"}
          <Button variant="success" size="sm" on:click={() => onChangeStatus(modalTask, "open")}>Unblock</Button>
        {/if}
        {#if modalTask.status !== "in_progress"}
          <Button variant="secondary" size="sm" on:click={() => onChangeStatus(modalTask, "in_progress")}>Start</Button>
        {/if}
        {#if modalTask.status !== "completed"}
          <Button variant="primary" size="sm" on:click={() => onChangeStatus(modalTask, "completed")}>Complete</Button>
        {/if}
        {#if modalTask.status !== "blocked"}
          <Button variant="warning" size="sm" on:click={onBlock}>Block</Button>
        {/if}
        <Button variant={confirmDeleteTask ? "danger" : "ghost"} size="sm" on:click={onDeleteTask} disabled={deletingTask}>
          {deletingTask ? "Deleting..." : confirmDeleteTask ? "Confirm Delete" : "Delete"}
        </Button>
      </div>

      <div class="task-modal-grid">
        <div class="subpanel modal-subpanel">
          <div class="section-head">
            <h3>Comments</h3>
            <span class="count">{taskComments.length}</span>
          </div>
          {#if taskDetailLoading}
            <div class="section-loading">
              <WabiSpinner size="md" tempo="calm" />
              <p>Loading comments...</p>
            </div>
          {:else if taskComments.length === 0}
            <div class="empty compact">No comments yet.</div>
          {:else}
            <div class="timeline">
              {#each taskComments as comment}
                <div class="timeline-item">
                  <strong>{comment.employee_name || "Unknown employee"}</strong>
                  <p>{comment.body}</p>
                  <span class="meta">{formatDate(comment.created_at)}</span>
                </div>
              {/each}
            </div>
          {/if}
          <div class="comment-composer">
            <textarea bind:value={commentDraft} rows="2" placeholder="Add a progress note or blocker update"></textarea>
            <Button variant="primary" size="sm" on:click={onAddComment} disabled={savingComment || taskDetailLoading}>Comment</Button>
          </div>
        </div>

        <div class="subpanel modal-subpanel">
          <div class="section-head">
            <h3>Activity Timeline</h3>
            <span class="count">{taskActivity.length}</span>
          </div>
          {#if taskDetailLoading}
            <div class="section-loading">
              <WabiSpinner size="md" tempo="calm" />
              <p>Loading activity...</p>
            </div>
          {:else if taskActivity.length === 0}
            <div class="empty compact">No activity yet.</div>
          {:else}
            <div class="timeline">
              {#each taskActivity as item}
                <div class="timeline-item">
                  <strong>{item.employee_name || "System"}</strong>
                  <p>{item.detail}</p>
                  <span class="meta">{formatDate(item.created_at)}</span>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</Modal>

<style>
  h3,
  p {
    margin: 0;
  }

  .detail-hero,
  .section-head,
  .task-actions {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: center;
  }

  .count,
  .meta {
    color: var(--text-secondary);
  }

  input,
  select,
  textarea {
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

  .task-modal-body {
    display: grid;
    gap: 18px;
  }

  .eyebrow {
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--text-secondary);
    margin-bottom: 6px;
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

  .modal-subpanel {
    border-top: none;
    padding-top: 0;
    align-self: start;
    height: auto;
    min-height: 0;
  }

  .compact-grid {
    grid-template-columns: minmax(0, 220px);
  }

  .editor-grid {
    display: grid;
    gap: 12px;
    grid-template-columns: 1.5fr auto 1fr auto;
  }

  .task-modal-grid {
    display: grid;
    gap: 12px;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    align-items: start;
  }

  .task-modal-grid .timeline {
    align-content: start;
    grid-auto-rows: max-content;
  }

  .task-modal-grid .timeline-item {
    align-self: start;
    height: auto;
    min-height: 0;
  }

  .task-modal-grid .comment-composer,
  .task-modal-grid .section-loading,
  .task-modal-grid .empty {
    align-self: start;
  }

  .comment-composer {
    display: grid;
    gap: 10px;
  }

  .timeline {
    display: grid;
    gap: 12px;
  }

  .timeline-item {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 14px;
    background: var(--bg-base);
    text-align: left;
  }

  .timeline-item strong {
    display: block;
  }

  .timeline-item p {
    margin-top: 6px;
    color: var(--text-secondary);
  }

  .section-loading,
  .modal-loading-state {
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
    .editor-grid,
    .task-modal-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 900px) {
    .detail-hero,
    .task-actions,
    .section-head {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
