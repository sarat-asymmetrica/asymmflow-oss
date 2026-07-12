<script lang="ts">
    /**
     * SyncProgress - Real-time database sync progress display
     * Shows percentage bar, table-by-table status, and record counts
     */
    import { onMount, onDestroy } from 'svelte';
    import { EventsOn, EventsOff } from '../../../../wailsjs/runtime/runtime';
    import WabiSpinner from './WabiSpinner.svelte';

    
    interface Props {
        // Props
        show?: boolean;
        title?: string;
    }

    let { show = $bindable(false), title = "Syncing Data" }: Props = $props();

    // State from sync events
    let progress = $state({
        phase: 'checking',
        current_table: '',
        tables_completed: 0,
        tables_total: 0,
        records_synced: 0,
        records_total: 0,
        percentage: 0,
        message: 'Initializing...',
        error: ''
    });

    let completedTables: string[] = [];
    let isComplete = $state(false);
    let hasError = $state(false);

    function handleSyncProgress(data: any) {
        progress = { ...progress, ...data };

        // Track completed tables
        if (data.current_table && data.phase !== 'error') {
            const idx = completedTables.indexOf(data.current_table);
            if (idx === -1 && data.tables_completed > completedTables.length) {
                // Previous table completed
                if (progress.tables_completed > 0) {
                    // Add the table that was just completed
                }
            }
        }

        isComplete = data.phase === 'complete';
        hasError = data.phase === 'error';

        if (isComplete || hasError) {
            // Auto-hide after a delay if complete
            if (isComplete) {
                setTimeout(() => {
                    show = false;
                }, 3000);
            }
        }
    }

    onMount(() => {
        EventsOn('sync:progress', handleSyncProgress);
    });

    onDestroy(() => {
        EventsOff('sync:progress');
    });

    // Computed
    let phaseIcon = $derived({
        'checking': 'Check',
        'uploading': 'Up',
        'downloading': 'Down',
        'complete': 'Done',
        'error': 'Error'
    }[progress.phase] || 'Wait');

    let phaseColor = $derived({
        'checking': 'var(--steel, #86868B)',
        'uploading': 'var(--info, #007AFF)',
        'downloading': 'var(--info, #007AFF)',
        'complete': 'var(--success, #34C759)',
        'error': 'var(--danger, #FF3B30)'
    }[progress.phase] || 'var(--steel)');
</script>

{#if show}
    <div class="sync-overlay">
        <div class="sync-modal">
            <div class="sync-header">
                <span class="phase-icon">{phaseIcon}</span>
                <h2>{title}</h2>
            </div>

            <div class="sync-content">
                <!-- Progress Bar -->
                <div class="progress-container">
                    <div class="progress-bar">
                        <div
                            class="progress-fill"
                            style="width: {progress.percentage}%; background: {phaseColor}"
                        ></div>
                    </div>
                    <span class="progress-percent">{Math.round(progress.percentage)}%</span>
                </div>

                <!-- Status Message -->
                <p class="status-message">{progress.message}</p>

                <!-- Current Table -->
                {#if progress.current_table && !isComplete && !hasError}
                    <div class="current-table">
                        <WabiSpinner size="sm" />
                        <span>{progress.current_table}</span>
                    </div>
                {/if}

                <!-- Stats -->
                <div class="sync-stats">
                    <div class="stat">
                        <span class="stat-value">{progress.tables_completed}</span>
                        <span class="stat-label">of {progress.tables_total} tables</span>
                    </div>
                    <div class="stat">
                        <span class="stat-value">{progress.records_synced.toLocaleString()}</span>
                        <span class="stat-label">records synced</span>
                    </div>
                </div>

                <!-- Error Message -->
                {#if hasError && progress.error}
                    <div class="error-box">
                        <strong>Error:</strong> {progress.error}
                    </div>
                {/if}

                <!-- Complete Message -->
                {#if isComplete}
                    <div class="complete-box">
                        <span class="complete-icon">Done</span>
                        <span>Sync completed successfully!</span>
                    </div>
                {/if}
            </div>
        </div>
    </div>
{/if}

<style>
    .sync-overlay {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 9999;
        backdrop-filter: blur(4px);
    }

    .sync-modal {
        background: var(--surface, #fff);
        border-radius: 16px;
        padding: 32px;
        min-width: 400px;
        max-width: 500px;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
    }

    .sync-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 24px;
    }

    .phase-icon {
        font-size: 28px;
    }

    h2 {
        margin: 0;
        font-size: 20px;
        font-weight: 600;
        color: var(--onyx, #1D1D1F);
    }

    .sync-content {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    .progress-container {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .progress-bar {
        flex: 1;
        height: 8px;
        background: var(--ether, #F5F5F7);
        border-radius: 4px;
        overflow: hidden;
    }

    .progress-fill {
        height: 100%;
        border-radius: 4px;
        transition: width 0.3s ease;
    }

    .progress-percent {
        font-size: 14px;
        font-weight: 600;
        color: var(--onyx, #1D1D1F);
        min-width: 45px;
        text-align: right;
        font-variant-numeric: tabular-nums;
    }

    .status-message {
        margin: 0;
        font-size: 14px;
        color: var(--steel, #86868B);
    }

    .current-table {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        background: var(--ether, #F5F5F7);
        border-radius: 8px;
        font-size: 13px;
        color: var(--onyx, #1D1D1F);
        font-family: monospace;
    }

    .sync-stats {
        display: flex;
        gap: 24px;
        padding-top: 8px;
        border-top: 1px solid var(--border, #E5E5E5);
    }

    .stat {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .stat-value {
        font-size: 24px;
        font-weight: 700;
        color: var(--onyx, #1D1D1F);
        font-variant-numeric: tabular-nums;
    }

    .stat-label {
        font-size: 12px;
        color: var(--steel, #86868B);
    }

    .error-box {
        padding: 12px 16px;
        background: var(--danger-bg, #FFF0F0);
        border: 1px solid var(--danger, #FF3B30);
        border-radius: 8px;
        color: var(--danger, #FF3B30);
        font-size: 13px;
    }

    .complete-box {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        padding: 16px;
        background: var(--success-bg, #F0FFF4);
        border-radius: 8px;
        color: var(--success, #34C759);
        font-size: 15px;
        font-weight: 500;
    }

    .complete-icon {
        font-size: 24px;
    }
</style>
