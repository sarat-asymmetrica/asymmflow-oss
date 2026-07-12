<script lang="ts">

    interface Props {
        tasks?: any;
        loading?: boolean;
        error?: string;
        onCapture: any;
    }

    let {
        tasks = [],
        loading = false,
        error = "",
        onCapture
    }: Props = $props();

    let quickCapture = $state("");
    let submitting = $state(false);

    async function handleCapture() {
        if (!quickCapture.trim() || !onCapture) return;
        submitting = true;
        await onCapture(quickCapture.trim());
        quickCapture = "";
        submitting = false;
    }
    // Ensure tasks is always an array
    let safeTasks = $derived(Array.isArray(tasks) ? tasks : []);
</script>

<div class="flow-state">
    {#if loading}
        <p class="legend">Loading tasks...</p>
    {:else if error}
        <p class="legend danger">{error}</p>
    {:else}
        <div class="task-list">
            {#if safeTasks.length === 0}
                <p class="legend">No tasks yet. Capture the first one.</p>
            {:else}
                {#each safeTasks as task (task.id)}
                    <div class="task-item">
                        <div class="task-content">
                            <h4>{task.text || task.title}</h4>
                            <p class="task-meta">
                                <span class="task-type"
                                    >{task.status || "open"}</span
                                >
                                <span class="dot">•</span>
                                <span
                                    >{new Date(
                                        task.createdAt,
                                    ).toLocaleDateString?.() || ""}</span
                                >
                            </p>
                        </div>
                        <div
                            class="status-dot {task.status === 'open'
                                ? 'safe'
                                : 'warning'}"
                        ></div>
                    </div>
                {/each}
            {/if}
        </div>
    {/if}

    <div class="quick-capture">
        <input
            type="text"
            bind:value={quickCapture}
            placeholder="Quick Capture: [Action] [Person] [Context]"
            onkeydown={(e) => e.key === "Enter" && handleCapture()}
            disabled={submitting}
        />
    </div>
</div>

<style>
    .flow-state {
        display: flex;
        flex-direction: column;
        height: 100%;
    }

    .legend {
        font-family: var(--font-mono, "Courier New", monospace);
        font-size: 0.8rem;
        color: var(--color-ink-light, #57534e);
        margin: 0;
    }

    .legend.danger {
        color: var(--color-danger, #ef4444);
    }

    .task-list {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 1.25rem;
    }

    .task-item {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        padding-bottom: 1.25rem;
        border-bottom: 1px solid rgba(0, 0, 0, 0.06);
    }

    .task-item:last-child {
        border-bottom: none;
    }

    .task-content h4 {
        font-family: var(--font-serif, Georgia, serif);
        font-size: 1rem;
        font-weight: normal;
        margin: 0 0 0.35rem;
        color: var(--color-ink, #1c1c1c);
    }

    .task-meta {
        font-family: var(--font-mono, "Courier New", monospace);
        font-size: 0.65rem;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--color-ink-light, #57534e);
        margin: 0;
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }

    .task-type {
        opacity: 0.7;
    }

    .dot {
        opacity: 0.3;
    }

    .status-dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
        flex-shrink: 0;
        margin-top: 0.35rem;
    }

    .status-dot.safe {
        background-color: var(--color-safe, #15803d);
    }

    .status-dot.warning {
        background-color: var(--color-gold, #fbbf24);
    }

    .quick-capture {
        margin-top: auto;
        padding-top: 1.5rem;
        border-top: 1px solid rgba(0, 0, 0, 0.06);
    }

    .quick-capture input {
        width: 100%;
        padding: 0.75rem;
        font-family: var(--font-mono, "Courier New", monospace);
        font-size: 0.75rem;
        color: var(--color-ink-light, #57534e);
        background: transparent;
        border: 1px dashed rgba(0, 0, 0, 0.15);
        outline: none;
        transition: all 0.2s ease;
    }

    .quick-capture input::placeholder {
        color: var(--color-ink-light, #57534e);
        opacity: 0.5;
    }

    .quick-capture input:focus {
        border-color: var(--color-ink, #1c1c1c);
        border-style: solid;
    }
</style>
