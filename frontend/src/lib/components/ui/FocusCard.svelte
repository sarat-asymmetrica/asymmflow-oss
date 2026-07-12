<script lang="ts">
  import { run, stopPropagation } from 'svelte/legacy';

  import { createEventDispatcher, onMount } from 'svelte';
  import type { UITask, TaskPriority } from '$lib/types';
  import { ChatWithButler } from '../../../../wailsjs/go/main/App';
import { CreateQuickCapture, GetQuickCaptures, UpdateQuickCapture, DeleteQuickCapture } from '../../../../wailsjs/go/main/DocumentsService';

  interface Props {
    tasks?: UITask[];
  }

  let { tasks = [] }: Props = $props();

  const dispatch = createEventDispatcher();

  interface InternalTask {
      id: string | number;
      title: string;
      priority: UITask['priority'];
      completed: boolean;
  }

  let internalTasks: InternalTask[] = $state([]);
  let showModal = $state(false);
  let newTaskTitle = $state("");
  let selectedPriority: TaskPriority = $state("Medium");

  // Butler AI suggestions
  let suggestions: string[] = $state([]);
  let loadingSuggestions = $state(false);

  // Merge prop tasks into internal state
  run(() => {
    if (tasks.length > 0) {
        const propMapped: InternalTask[] = tasks.map(t => ({
            id: t.id,
            title: t.title,
            priority: t.priority || "Medium",
            completed: t.done || t.status === 'completed'
        }));
        // Only update if prop tasks actually changed
        const propIds = propMapped.map(t => String(t.id)).join(',');
        const currentPropIds = internalTasks.filter(t => tasks.some(pt => String(pt.id) === String(t.id))).map(t => String(t.id)).join(',');
        if (propIds !== currentPropIds) {
            const localOnly = internalTasks.filter(t => !tasks.some(pt => String(pt.id) === String(t.id)));
            internalTasks = [...propMapped, ...localOnly];
        }
    }
  });

  // Filter out error messages that were accidentally saved as tasks
  const errorPhrases = ['financial data access', 'requires manager', 'admin privileges', 'you can still ask'];

  function isErrorTask(title: string): boolean {
      const lower = title.toLowerCase();
      return errorPhrases.some(phrase => lower.includes(phrase));
  }

  // Reactive sorting: incomplete first, then by priority (filtering error tasks)
  let sortedTasks = $derived([...internalTasks]
      .filter(t => !isErrorTask(t.title))
      .sort((a, b) => {
          if (a.completed === b.completed) {
              const order: Record<string, number> = { "High": 1, "Medium": 2, "Low": 3 };
              return (order[a.priority] || 2) - (order[b.priority] || 2);
          }
          return a.completed ? 1 : -1;
      }));

  let filteredTasks = $derived(internalTasks.filter(t => !isErrorTask(t.title)));
  let completedCount = $derived(filteredTasks.filter(t => t.completed).length);
  let totalCount = $derived(filteredTasks.length);

  function openModal() {
      showModal = true;
      newTaskTitle = "";
      selectedPriority = "Medium";
  }

  function closeModal() {
      showModal = false;
  }

  async function addTask() {
      if (!newTaskTitle.trim()) return;
      const title = newTaskTitle.trim();
      const priority = selectedPriority;

      // Optimistic local update
      const newTask: InternalTask = {
          id: Date.now(),
          title,
          priority,
          completed: false
      };
      internalTasks = [...internalTasks, newTask];
      closeModal();

      // Persist to backend
      try {
          const savedId = await CreateQuickCapture(title, "", "", priority);
          // Update local ID with persisted ID
          internalTasks = internalTasks.map(t =>
              t.id === newTask.id ? { ...t, id: savedId } : t
          );
      } catch (err) {
          console.error("Failed to save task:", err);
      }

      dispatch('addTask', newTask);
  }

  function toggleTask(id: string | number) {
      const task = internalTasks.find(t => t.id === id);
      if (!task) return;

      const newCompleted = !task.completed;
      internalTasks = internalTasks.map(t =>
          t.id === id ? { ...t, completed: newCompleted } : t
      );
      dispatch('toggleTask', { id });

      // Persist status change
      const numId = Number(id);
      if (!isNaN(numId) && numId > 0) {
          UpdateQuickCapture(numId, task.title, "", "", task.priority, newCompleted ? "Done" : "Open")
              .catch(err => console.error("Failed to update task status:", err));
      }
  }

  function deleteTask(id: string | number) {
      internalTasks = internalTasks.filter(t => t.id !== id);
      dispatch('deleteTask', { id });

      // Persist deletion
      DeleteQuickCapture(String(id))
          .catch(err => console.error("Failed to delete task:", err));
  }

  async function addSuggestion(text: string) {
      const newTask: InternalTask = {
          id: Date.now(),
          title: text,
          priority: "Medium",
          completed: false
      };
      internalTasks = [...internalTasks, newTask];
      suggestions = suggestions.filter(s => s !== text);

      try {
          const savedId = await CreateQuickCapture(text, "", "butler-suggested", "Medium");
          internalTasks = internalTasks.map(t =>
              t.id === newTask.id ? { ...t, id: savedId } : t
          );
      } catch (err) {
          console.error("Failed to save suggestion:", err);
      }

      dispatch('addTask', newTask);
  }

  async function fetchSuggestions() {
      loadingSuggestions = true;
      try {
          // Avoid triggering financial RBAC by asking about operations/sales instead
          const response = await ChatWithButler(
              "Based on current pending RFQs, follow-ups needed, and customer opportunities, suggest 3 brief actionable tasks for today. Reply with just the task titles, one per line, no numbering or bullets."
          );
          if (response?.message) {
              // Filter out RBAC error messages and system text
              const blockedPhrases = [
                  'financial data access',
                  'requires manager',
                  'admin privileges',
                  'you can still ask',
                  'please contact',
                  'for financial reports',
                  '•' // Filter bullet points from error messages
              ];

              suggestions = response.message
                  .split('\n')
                  .map(s => s.trim())
                  .filter(s => {
                      if (s.length === 0 || s.length > 100) return false;
                      const lower = s.toLowerCase();
                      return !blockedPhrases.some(phrase => lower.includes(phrase));
                  })
                  .slice(0, 3);
          }
      } catch (err) {
          console.error("Butler suggestions failed:", err);
      } finally {
          loadingSuggestions = false;
      }
  }

  onMount(async () => {
      await loadPersistedTasks();
      fetchSuggestions();
  });

  async function loadPersistedTasks() {
      try {
          const captures = await GetQuickCaptures(20);
          if (captures && captures.length > 0) {
              const loaded: InternalTask[] = captures.map(c => ({
                  id: c.id,
                  title: c.title,
                  priority: (c.priority as UITask['priority']) || "Medium",
                  completed: c.status === "Done"
              }));
              // Merge with prop tasks (prop tasks take precedence by being first)
              if (tasks.length > 0) {
                  const propIds = new Set(tasks.map(t => String(t.id)));
                  const uniqueCaptures = loaded.filter(c => !propIds.has(String(c.id)));
                  internalTasks = [...internalTasks, ...uniqueCaptures];
              } else {
                  internalTasks = loaded;
              }
          }
      } catch (err) {
          console.error("Failed to load tasks:", err);
      }
  }

  // Priority weight classes
  function priorityWeight(p: string): string {
      if (p === "High") return "weight-high";
      if (p === "Low") return "weight-low";
      return "weight-medium";
  }
</script>

<div class="focus-card">
    <!-- Header -->
    <div class="card-header">
        <div class="header-left">
            <h3 class="card-title">My Tasks</h3>
            <span class="task-count">{completedCount}/{totalCount}</span>
        </div>
        <div class="progress-track">
            <div class="progress-fill" style="width: {totalCount > 0 ? (completedCount / totalCount) * 100 : 0}%"></div>
        </div>
    </div>

    <!-- Task List -->
    <div class="task-list">
        {#each sortedTasks as task (task.id)}
            <div class="task-row" class:completed={task.completed}>
                <button
                    class="checkbox"
                    class:checked={task.completed}
                    onclick={() => toggleTask(task.id)}
                    aria-label={task.completed ? "Mark incomplete" : "Mark complete"}
                >
                    {#if task.completed}
                        <svg viewBox="0 0 20 20" fill="currentColor" class="check-icon">
                            <path d="M0 11l2-2 5 5L18 3l2 2L7 18z"/>
                        </svg>
                    {/if}
                </button>

                <span class="task-title {priorityWeight(task.priority)}" class:struck={task.completed}>
                    {task.title}
                </span>

                <button
                    class="delete-btn"
                    onclick={stopPropagation(() => deleteTask(task.id))}
                    aria-label="Delete task"
                >
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M18 6L6 18M6 6l12 12"/>
                    </svg>
                </button>
            </div>
        {/each}

        {#if sortedTasks.length === 0}
            <div class="empty-state">No tasks yet</div>
        {/if}
    </div>

    <!-- Butler Suggestions -->
    {#if suggestions.length > 0}
        <div class="suggestions">
            <span class="suggestions-label">Butler suggests</span>
            <div class="suggestion-chips">
                {#each suggestions as suggestion}
                    <button class="suggestion-chip" onclick={() => addSuggestion(suggestion)}>
                        <span class="chip-plus">+</span>
                        <span class="chip-text">{suggestion}</span>
                    </button>
                {/each}
            </div>
        </div>
    {:else if loadingSuggestions}
        <div class="suggestions">
            <span class="suggestions-label">Butler thinking...</span>
        </div>
    {/if}

    <!-- FAB -->
    <button class="fab" onclick={openModal} aria-label="Add task">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <path d="M12 4v16m8-8H4"/>
        </svg>
    </button>

    <!-- Add Task Modal -->
    {#if showModal}
        <div class="modal-overlay">
            <div class="modal-content">
                <h3 class="modal-title">Add Task</h3>

                <input
                    type="text"
                    bind:value={newTaskTitle}
                    placeholder="What needs to be done?"
                    class="modal-input"
                    onkeydown={(e) => e.key === 'Enter' && addTask()}
                />

                <div class="priority-selector">
                    <span class="priority-label">Priority</span>
                    <div class="priority-options">
                        <button
                            class="priority-opt"
                            class:active={selectedPriority === 'High'}
                            onclick={() => selectedPriority = 'High'}
                        >High</button>
                        <button
                            class="priority-opt"
                            class:active={selectedPriority === 'Medium'}
                            onclick={() => selectedPriority = 'Medium'}
                        >Medium</button>
                        <button
                            class="priority-opt"
                            class:active={selectedPriority === 'Low'}
                            onclick={() => selectedPriority = 'Low'}
                        >Low</button>
                    </div>
                </div>

                <div class="modal-actions">
                    <button class="btn-add" onclick={addTask}>Add</button>
                    <button class="btn-cancel" onclick={closeModal}>Cancel</button>
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .focus-card {
        background: var(--canvas, #fff);
        border: 1px solid var(--border, #e5e5e5);
        border-radius: var(--border-radius, 12px);
        display: flex;
        flex-direction: column;
        position: relative;
        overflow: hidden;
        min-height: 400px;
    }

    /* Header */
    .card-header {
        padding: 20px 20px 16px;
        border-bottom: 1px solid var(--border, #e5e5e5);
    }

    .header-left {
        display: flex;
        align-items: baseline;
        gap: 10px;
        margin-bottom: 10px;
    }

    .card-title {
        font-size: 16px;
        font-weight: 600;
        color: var(--onyx, #1D1D1F);
        margin: 0;
        letter-spacing: -0.01em;
    }

    .task-count {
        font-size: 12px;
        color: var(--steel, #86868B);
        font-variant-numeric: tabular-nums;
    }

    .progress-track {
        height: 3px;
        background: var(--bg-base, #f5f5f7);
        border-radius: 2px;
        overflow: hidden;
    }

    .progress-fill {
        height: 100%;
        background: var(--carbon, #000);
        border-radius: 2px;
        transition: width 0.3s ease;
    }

    /* Task List */
    .task-list {
        flex: 1;
        overflow-y: auto;
        padding: 8px 12px;
    }

    .task-list::-webkit-scrollbar {
        width: 4px;
    }
    .task-list::-webkit-scrollbar-track {
        background: transparent;
    }
    .task-list::-webkit-scrollbar-thumb {
        background: var(--border, #e5e5e5);
        border-radius: 2px;
    }

    .task-row {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px 8px;
        border-radius: var(--border-radius-sm, 8px);
        transition: background var(--transition-fast, 0.12s);
    }

    .task-row:hover {
        background: var(--bg-base, #f5f5f7);
    }

    .task-row.completed {
        opacity: 0.5;
    }

    /* Checkbox */
    .checkbox {
        width: 20px;
        height: 20px;
        border-radius: 50%;
        border: 2px solid var(--border, #e5e5e5);
        background: transparent;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
        transition: all var(--transition-fast, 0.12s);
        padding: 0;
    }

    .checkbox:hover {
        border-color: var(--steel, #86868B);
    }

    .checkbox.checked {
        background: var(--carbon, #000);
        border-color: var(--carbon, #000);
        color: white;
    }

    .check-icon {
        width: 10px;
        height: 10px;
    }

    /* Task Title with priority weight */
    .task-title {
        flex: 1;
        font-size: 13px;
        color: var(--onyx, #1D1D1F);
        line-height: 1.3;
    }

    .task-title.weight-high {
        font-weight: 700;
    }

    .task-title.weight-medium {
        font-weight: 500;
    }

    .task-title.weight-low {
        font-weight: 400;
        color: var(--steel, #86868B);
    }

    .task-title.struck {
        text-decoration: line-through;
        color: var(--steel, #86868B);
    }

    /* Delete button */
    .delete-btn {
        width: 20px;
        height: 20px;
        border: none;
        background: transparent;
        color: var(--steel, #86868B);
        cursor: pointer;
        opacity: 0;
        transition: opacity var(--transition-fast, 0.12s), color var(--transition-fast, 0.12s);
        padding: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }

    .task-row:hover .delete-btn {
        opacity: 1;
    }

    .delete-btn:hover {
        color: var(--onyx, #1D1D1F);
    }

    .delete-btn svg {
        width: 14px;
        height: 14px;
    }

    .empty-state {
        text-align: center;
        padding: 32px 16px;
        color: var(--steel, #86868B);
        font-size: 13px;
    }

    /* Butler Suggestions */
    .suggestions {
        padding: 12px 16px;
        border-top: 1px solid var(--border, #e5e5e5);
    }

    .suggestions-label {
        font-size: 10px;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: var(--steel, #86868B);
        display: block;
        margin-bottom: 8px;
    }

    .suggestion-chips {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .suggestion-chip {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        background: var(--bg-base, #f5f5f7);
        border: 1px solid var(--border, #e5e5e5);
        border-radius: var(--border-radius-sm, 8px);
        cursor: pointer;
        transition: all var(--transition-fast, 0.12s);
        text-align: left;
        width: 100%;
    }

    .suggestion-chip:hover {
        background: var(--carbon, #000);
        color: white;
        border-color: var(--carbon, #000);
    }

    .chip-plus {
        font-weight: 700;
        font-size: 14px;
        flex-shrink: 0;
    }

    .chip-text {
        font-size: 12px;
        line-height: 1.3;
    }

    /* FAB */
    .fab {
        position: absolute;
        bottom: 16px;
        right: 16px;
        width: 44px;
        height: 44px;
        border-radius: 50%;
        background: var(--carbon, #000);
        color: white;
        border: none;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        transition: transform var(--transition-fast, 0.12s), box-shadow var(--transition-fast, 0.12s);
    }

    .fab:hover {
        transform: scale(1.08);
        box-shadow: 0 6px 20px rgba(0, 0, 0, 0.2);
    }

    .fab:active {
        transform: scale(0.95);
    }

    .fab svg {
        width: 20px;
        height: 20px;
    }

    /* Modal */
    .modal-overlay {
        position: absolute;
        inset: 0;
        background: rgba(255, 255, 255, 0.96);
        backdrop-filter: blur(8px);
        z-index: 10;
        display: flex;
        flex-direction: column;
        justify-content: center;
        padding: 24px;
        border-radius: var(--border-radius, 12px);
    }

    .modal-content {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    .modal-title {
        font-size: 18px;
        font-weight: 600;
        color: var(--onyx, #1D1D1F);
        margin: 0;
    }

    .modal-input {
        width: 100%;
        padding: 12px 0;
        border: none;
        border-bottom: 2px solid var(--border, #e5e5e5);
        font-size: 15px;
        color: var(--onyx, #1D1D1F);
        background: transparent;
        outline: none;
        transition: border-color var(--transition-fast, 0.12s);
    }

    .modal-input:focus {
        border-color: var(--carbon, #000);
    }

    .modal-input::placeholder {
        color: var(--steel, #86868B);
    }

    .priority-selector {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .priority-label {
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: var(--steel, #86868B);
        font-weight: 600;
    }

    .priority-options {
        display: flex;
        gap: 8px;
    }

    .priority-opt {
        flex: 1;
        padding: 8px 12px;
        border: 1px solid var(--border, #e5e5e5);
        border-radius: var(--border-radius-sm, 8px);
        background: transparent;
        cursor: pointer;
        font-size: 13px;
        font-weight: 500;
        color: var(--steel, #86868B);
        transition: all var(--transition-fast, 0.12s);
    }

    .priority-opt:hover {
        border-color: var(--onyx, #1D1D1F);
        color: var(--onyx, #1D1D1F);
    }

    .priority-opt.active {
        background: var(--carbon, #000);
        color: white;
        border-color: var(--carbon, #000);
    }

    .modal-actions {
        display: flex;
        gap: 12px;
        margin-top: 8px;
    }

    .btn-add {
        flex: 1;
        padding: 12px;
        background: var(--carbon, #000);
        color: white;
        border: none;
        border-radius: var(--border-radius-sm, 8px);
        font-size: 14px;
        font-weight: 600;
        cursor: pointer;
        transition: opacity var(--transition-fast, 0.12s);
    }

    .btn-add:hover {
        opacity: 0.85;
    }

    .btn-cancel {
        padding: 12px 16px;
        background: transparent;
        color: var(--steel, #86868B);
        border: none;
        font-size: 14px;
        font-weight: 600;
        cursor: pointer;
        transition: color var(--transition-fast, 0.12s);
    }

    .btn-cancel:hover {
        color: var(--onyx, #1D1D1F);
    }
</style>
