<script lang="ts">
import { devLog } from "$lib/utils/devLog";
import { onMount, onDestroy } from 'svelte';
import { toast } from '$lib/stores/toasts';
import { GetAlertSummary } from '../../../wailsjs/go/main/App';
import { AcknowledgeAlert, DismissAlert } from '../../../wailsjs/go/main/ButlerService';
import type { main, infra } from '../../../wailsjs/go/models';


    interface Props {
        // Props
        loading?: boolean;
        error?: string;
    }

    let { loading = $bindable(false), error = $bindable('') }: Props = $props();

// Extended alert type with optional customer_name
type ExtendedAlert = infra.Alert & { customer_name?: string };
type ExtendedAlertSummary = Omit<main.AlertSummary, 'top_alerts'> & { top_alerts: ExtendedAlert[] };

// Alert Summary State (use backend types)
let summary: ExtendedAlertSummary | null = $state(null);

// Auto-refresh interval (30 seconds)
let refreshInterval: ReturnType<typeof setInterval> | null = null;
const REFRESH_INTERVAL_MS = 30000;

// Load alert summary from backend
async function loadAlertSummary() {
    loading = true;
    error = '';
    try {
        const data = await GetAlertSummary();
        summary = data;
        devLog.log('Alert summary loaded:', summary);
    } catch (err) {
        devLog.error('Failed to load alert summary:', err);
        error = 'Failed to load alert summary';
        summary = null;
    } finally {
        loading = false;
    }
}

// Acknowledge alert
async function acknowledgeAlert(alertId: string | number) {
    try {
        const id = typeof alertId === 'string' ? parseInt(alertId) : alertId;
        await AcknowledgeAlert(id);
        toast.success('Alert acknowledged');
        await loadAlertSummary(); // Refresh
    } catch (err) {
        devLog.error('Failed to acknowledge alert:', err);
        toast.danger('Failed to acknowledge alert');
    }
}

// Dismiss alert
async function dismissAlert(alertId: string | number) {
    try {
        const id = typeof alertId === 'string' ? parseInt(alertId) : alertId;
        await DismissAlert(id);
        toast.success('Alert dismissed');
        await loadAlertSummary(); // Refresh
    } catch (err) {
        devLog.error('Failed to dismiss alert:', err);
        toast.danger('Failed to dismiss alert');
    }
}

// Get severity color
function getSeverityColor(severity: string): string {
    switch (severity) {
        case 'critical': return '#dc2626';   // Red
        case 'warning': return '#f59e0b';    // Orange
        case 'opportunity': return '#16a34a'; // Green
        case 'info': return '#3b82f6';       // Blue
        default: return '#57534e';           // Gray
    }
}

// Get severity icon
function getSeverityIcon(severity: string): string {
    switch (severity) {
        case 'critical': return '!';
        case 'warning': return '!';
        case 'opportunity': return '*';
        case 'info': return 'i';
        default: return '*';
    }
}

// Format time ago
function formatTimeAgo(dateTime: string | Date | any): string {
    // Handle time.Time objects from backend
    const date = new Date(dateTime as any);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;

    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays}d ago`;
}

onMount(async () => {
    // Initial load
    await loadAlertSummary();

    // Set up auto-refresh
    refreshInterval = setInterval(loadAlertSummary, REFRESH_INTERVAL_MS);

    devLog.log('Alert auto-refresh enabled (30s interval)');
});

onDestroy(() => {
    // Clean up interval
    if (refreshInterval) {
        clearInterval(refreshInterval);
        devLog.log('Alert auto-refresh disabled');
    }
});
</script>

<div class="alert-panel" role="region" aria-labelledby="alert-heading">
    <header class="panel-header">
        <h2 id="alert-heading" class="panel-title">CRITICAL ALERTS</h2>
        <div class="alert-counts" role="status">
            {#if summary && summary.active_critical > 0}
                <span class="count-badge critical" aria-label="{summary.active_critical} critical alerts">
                    {summary.active_critical} Critical
                </span>
            {/if}
            {#if summary && summary.active_warning > 0}
                <span class="count-badge warning" aria-label="{summary.active_warning} warning alerts">
                    {summary.active_warning} Warning
                </span>
            {/if}
            {#if summary && summary.active_critical === 0 && summary.active_warning === 0}
                <span class="count-badge safe">All Clear</span>
            {/if}
        </div>
    </header>

    {#if loading}
        <div class="loading-state" role="status">
            <div class="spinner" aria-hidden="true"></div>
            <p>Loading alerts...</p>
        </div>
    {:else if error}
        <div class="error-state" role="alert">
            <p>Error: {error}</p>
        </div>
    {:else if !summary || summary.top_alerts.length === 0}
        <div class="empty-state">
            <div class="empty-icon" aria-hidden="true">OK</div>
            <p class="empty-message">No active alerts</p>
            <p class="empty-subtitle">All systems operating normally</p>
        </div>
    {:else}
        <div class="alerts-list">
            {#each summary.top_alerts as alert (alert.id)}
                <div
                    class="alert-card {alert.severity}"
                    class:acknowledged={alert.is_acknowledged}
                    style="--severity-color: {getSeverityColor(alert.severity)}"
                    role="article"
                    aria-labelledby="alert-title-{alert.id}"
                >
                    <div class="alert-header">
                        <div class="alert-icon" aria-hidden="true">
                            {getSeverityIcon(alert.severity)}
                        </div>
                        <div class="alert-meta">
                            <span class="alert-severity-badge {alert.severity}">
                                {alert.severity.toUpperCase()}
                            </span>
                            <span class="alert-time">{formatTimeAgo(alert.created_at)}</span>
                        </div>
                    </div>

                    <h3 id="alert-title-{alert.id}" class="alert-title">
                        {alert.title}
                    </h3>

                    <p class="alert-message">
                        {alert.message}
                    </p>

                    {#if alert.customer_name}
                        <div class="alert-customer">
                            <strong>Customer:</strong> {alert.customer_name}
                        </div>
                    {/if}

                    <div class="alert-actions">
                        {#if !alert.is_acknowledged}
                            <button
                                class="action-btn acknowledge-btn"
                                onclick={() => acknowledgeAlert(alert.id)}
                                aria-label="Acknowledge alert"
                            >
                                Acknowledge
                            </button>
                        {/if}
                        <button
                            class="action-btn dismiss-btn"
                            onclick={() => dismissAlert(alert.id)}
                            aria-label="Dismiss alert"
                        >
                            &times; Dismiss
                        </button>
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>

<style>
    /* Panel Container */
    .alert-panel {
        background: rgba(255, 255, 255, 0.7);
        border: 1px solid rgba(0, 0, 0, 0.08);
        border-radius: var(--fib-1);
        padding: var(--fib-3);
        margin-bottom: var(--fib-4);
        animation: fadeIn 0.5s ease-out;
    }

    @keyframes fadeIn {
        from { opacity: 0; transform: translateY(-10px); }
        to { opacity: 1; transform: translateY(0); }
    }

    /* Header */
    .panel-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: var(--fib-3);
        padding-bottom: var(--fib-2);
        border-bottom: 1px solid rgba(0, 0, 0, 0.1);
    }

    .panel-title {
        font-family: 'Courier Prime', monospace;
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 2px;
        color: #1c1c1c;
        font-weight: 600;
        margin: 0;
    }

    .alert-counts {
        display: flex;
        gap: var(--fib-1);
    }

    .count-badge {
        display: inline-block;
        padding: 4px var(--fib-2);
        border-radius: 12px;
        font-size: 11px;
        font-family: 'Courier Prime', monospace;
        font-weight: 600;
    }

    .count-badge.critical {
        background: rgba(220, 38, 38, 0.15);
        color: #dc2626;
    }

    .count-badge.warning {
        background: rgba(245, 158, 11, 0.15);
        color: #f59e0b;
    }

    .count-badge.safe {
        background: rgba(22, 163, 74, 0.15);
        color: #16a34a;
    }

    /* Alerts List */
    .alerts-list {
        display: flex;
        flex-direction: column;
        gap: var(--fib-3);
    }

    /* Alert Card */
    .alert-card {
        background: white;
        border-left: 4px solid var(--severity-color);
        border-radius: var(--fib-1);
        padding: var(--fib-3);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        animation: slideIn 0.4s ease-out;
    }

    @keyframes slideIn {
        from {
            opacity: 0;
            transform: translateX(-20px);
        }
        to {
            opacity: 1;
            transform: translateX(0);
        }
    }

    .alert-card:hover {
        box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
        transform: translateX(4px);
    }

    .alert-card.acknowledged {
        opacity: 0.6;
        background: rgba(255, 255, 255, 0.8);
    }

    /* Alert Header */
    .alert-header {
        display: flex;
        align-items: center;
        gap: var(--fib-2);
        margin-bottom: var(--fib-2);
    }

    .alert-icon {
        font-size: 24px;
        flex-shrink: 0;
    }

    .alert-meta {
        display: flex;
        align-items: center;
        gap: var(--fib-1);
        flex: 1;
    }

    .alert-severity-badge {
        display: inline-block;
        padding: 3px var(--fib-1);
        border-radius: 4px;
        font-size: 9px;
        font-family: 'Courier Prime', monospace;
        font-weight: 600;
        letter-spacing: 1px;
    }

    .alert-severity-badge.critical {
        background: rgba(220, 38, 38, 0.15);
        color: #dc2626;
    }

    .alert-severity-badge.warning {
        background: rgba(245, 158, 11, 0.15);
        color: #f59e0b;
    }

    .alert-severity-badge.opportunity {
        background: rgba(22, 163, 74, 0.15);
        color: #16a34a;
    }

    .alert-time {
        font-size: 11px;
        color: #57534e;
        font-family: 'Courier Prime', monospace;
    }

    /* Alert Content */
    .alert-title {
        font-family: Georgia, serif;
        font-size: 16px;
        font-weight: normal;
        color: #1c1c1c;
        margin: 0 0 var(--fib-2);
        line-height: 1.4;
    }

    .alert-message {
        font-size: 13px;
        color: #57534e;
        line-height: 1.6;
        margin: 0 0 var(--fib-2);
    }

    .alert-customer {
        font-size: 12px;
        color: #57534e;
        margin-bottom: var(--fib-2);
    }

    .alert-customer strong {
        color: #1c1c1c;
    }

    /* Alert Actions */
    .alert-actions {
        display: flex;
        gap: var(--fib-1);
        margin-top: var(--fib-2);
        padding-top: var(--fib-2);
        border-top: 1px solid rgba(0, 0, 0, 0.05);
    }

    .action-btn {
        padding: 6px var(--fib-2);
        border: 1px solid rgba(0, 0, 0, 0.1);
        border-radius: 6px;
        background: white;
        font-size: 11px;
        font-family: 'Courier Prime', monospace;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s ease;
    }

    .action-btn:hover {
        transform: translateY(-1px);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }

    .action-btn:active {
        transform: translateY(0);
    }

    .acknowledge-btn {
        color: #16a34a;
        border-color: #16a34a;
    }

    .acknowledge-btn:hover {
        background: rgba(22, 163, 74, 0.1);
    }

    .dismiss-btn {
        color: #57534e;
        border-color: #57534e;
    }

    .dismiss-btn:hover {
        background: rgba(87, 83, 78, 0.1);
    }

    /* Empty State */
    .empty-state {
        text-align: center;
        padding: var(--fib-4) 0;
    }

    .empty-icon {
        font-size: 48px;
        margin-bottom: var(--fib-2);
    }

    .empty-message {
        font-family: Georgia, serif;
        font-size: 18px;
        color: #1c1c1c;
        margin: 0 0 var(--fib-1);
    }

    .empty-subtitle {
        font-size: 13px;
        color: #57534e;
        margin: 0;
    }

    /* Loading State */
    .loading-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: var(--fib-4);
        color: #57534e;
    }

    .spinner {
        width: 32px;
        height: 32px;
        border: 3px solid rgba(0, 0, 0, 0.1);
        border-top-color: #1c1c1c;
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin-bottom: var(--fib-2);
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    /* Error State */
    .error-state {
        background: rgba(220, 38, 38, 0.1);
        border: 1px solid rgba(220, 38, 38, 0.3);
        border-radius: var(--fib-1);
        padding: var(--fib-3);
        color: #dc2626;
        text-align: center;
    }

    /* Responsive */
    @media (max-width: 768px) {
        .alert-actions {
            flex-direction: column;
        }

        .action-btn {
            width: 100%;
        }
    }
</style>
