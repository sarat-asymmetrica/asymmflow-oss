<script lang="ts">
import { devLog } from "$lib/utils/devLog";
import { onMount, onDestroy } from 'svelte';
import { GetSurvivalMetrics } from '../../../wailsjs/go/main/App';
import type { main } from '../../../wailsjs/go/models';


    interface Props {
        // Props
        loading?: boolean;
        error?: string;
    }

    let { loading = $bindable(false), error = $bindable('') }: Props = $props();

// Survival Metrics State (use backend types)
let metrics: main.SurvivalMetrics | null = $state(null);

// Auto-refresh interval (30 seconds)
let refreshInterval: ReturnType<typeof setInterval> | null = null;
const REFRESH_INTERVAL_MS = 30000;

// Load survival metrics from backend
async function loadMetrics() {
    loading = true;
    error = '';
    try {
        const data = await GetSurvivalMetrics();
        metrics = data;
        devLog.log('Survival metrics loaded:', metrics);
    } catch (err) {
        devLog.error('Failed to load survival metrics:', err);
        error = 'Failed to load survival metrics';
        metrics = null;
    } finally {
        loading = false;
    }
}

// Color coding based on runway status
function getRunwayColor(status: string): string {
    switch (status) {
        case 'critical': return '#dc2626'; // Red
        case 'warning': return '#f59e0b';  // Yellow/Orange
        case 'safe': return '#16a34a';     // Green
        default: return '#57534e';         // Gray
    }
}

// Format currency (BHD)
function formatCurrency(value: number): string {
    return 'BHD ' + (value || 0).toLocaleString(undefined, { minimumFractionDigits: 0, maximumFractionDigits: 0 });
}

// Format percentage
function formatPercent(value: number): string {
    return (value * 100).toFixed(1) + '%';
}

// Format days with appropriate suffix
function formatDays(days: number): string {
    const rounded = Math.round(days);
    return `${rounded} day${rounded !== 1 ? 's' : ''}`;
}

// Format time.Time to string
function formatLastUpdated(timeObj: string | Date | any): string {
    try {
        return new Date(timeObj as any).toLocaleTimeString();
    } catch {
        return 'Unknown';
    }
}

onMount(async () => {
    // Initial load
    await loadMetrics();

    // Set up auto-refresh
    refreshInterval = setInterval(loadMetrics, REFRESH_INTERVAL_MS);

    devLog.log('Auto-refresh enabled (30s interval)');
});

onDestroy(() => {
    // Clean up interval
    if (refreshInterval) {
        clearInterval(refreshInterval);
        devLog.log('Auto-refresh disabled');
    }
});
</script>

<div class="survival-panel" role="region" aria-labelledby="survival-heading">
    <header class="panel-header">
        <h2 id="survival-heading" class="panel-title">SURVIVAL METRICS</h2>
        <span class="last-updated" role="status">
            {#if metrics && metrics.last_updated}
                Updated {formatLastUpdated(metrics.last_updated)}
            {:else}
                Loading...
            {/if}
        </span>
    </header>

    {#if loading}
        <div class="loading-state" role="status">
            <div class="spinner" aria-hidden="true"></div>
            <p>Loading survival metrics...</p>
        </div>
    {:else if error}
        <div class="error-state" role="alert">
            <p>Error: {error}</p>
        </div>
    {:else if !metrics}
        <div class="empty-state">
            <p>No survival data available</p>
        </div>
    {:else}
        <div class="metrics-grid">
            <!-- Cash Runway (Most Critical) -->
            <div
                class="metric-card runway-card"
                style="--status-color: {getRunwayColor(metrics.runway_status)}"
                role="group"
                aria-labelledby="runway-label"
            >
                <div class="metric-icon" aria-hidden="true">Time</div>
                <div class="metric-content">
                    <h3 id="runway-label" class="metric-label">Cash Runway</h3>
                    <div class="metric-value-large" style="color: {getRunwayColor(metrics.runway_status)}">
                        {formatDays(metrics.days_of_runway)}
                    </div>
                    <div class="metric-status" aria-live="polite">
                        <span class="status-badge {metrics.runway_status}">
                            {metrics.runway_status.toUpperCase()}
                        </span>
                    </div>
                    <div class="metric-details">
                        <div class="detail-row">
                            <span>Cash:</span>
                            <strong>{formatCurrency(metrics.cash_balance)}</strong>
                        </div>
                        <div class="detail-row">
                            <span>Burn:</span>
                            <strong>{formatCurrency(metrics.monthly_burn)}/mo</strong>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Collections This Week -->
            <div
                class="metric-card collections-card"
                role="group"
                aria-labelledby="collections-label"
            >
                <div class="metric-icon" aria-hidden="true">Cash</div>
                <div class="metric-content">
                    <h3 id="collections-label" class="metric-label">Collections This Week</h3>
                    <div class="metric-value">
                        {formatCurrency(metrics.week_collections_actual)}
                    </div>
                    <div class="metric-subtitle">
                        of {formatCurrency(metrics.week_collections_target)} target
                    </div>
                    <div class="progress-bar" role="progressbar"
                         aria-valuenow={metrics.collection_efficiency * 100}
                         aria-valuemin="0"
                         aria-valuemax="100">
                        <div
                            class="progress-fill"
                            style="width: {Math.min(metrics.collection_efficiency * 100, 100)}%"
                        ></div>
                    </div>
                    <div class="metric-percentage">
                        {formatPercent(metrics.collection_efficiency)} collected
                    </div>
                </div>
            </div>

            <!-- Overdue Summary -->
            <div
                class="metric-card overdue-card"
                role="group"
                aria-labelledby="overdue-label"
            >
                <div class="metric-icon" aria-hidden="true">Overdue</div>
                <div class="metric-content">
                    <h3 id="overdue-label" class="metric-label">Overdue Invoices</h3>
                    {#if metrics.overdue_by_grade && Object.keys(metrics.overdue_by_grade).length > 0}
                        <div class="overdue-list">
                            {#each Object.entries(metrics.overdue_by_grade) as [grade, amount]}
                                <div class="overdue-item">
                                    <span class="grade-badge grade-{grade}">{grade}</span>
                                    <span class="overdue-amount">{formatCurrency(amount)}</span>
                                </div>
                            {/each}
                        </div>
                    {:else}
                        <div class="empty-state">
                            <p>OK - No overdue invoices</p>
                        </div>
                    {/if}
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    /* Panel Container */
    .survival-panel {
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

    .last-updated {
        font-family: 'Courier Prime', monospace;
        font-size: 10px;
        color: #57534e;
        font-style: italic;
    }

    /* Metrics Grid */
    .metrics-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
        gap: var(--fib-3);
    }

    /* Metric Card */
    .metric-card {
        background: rgba(255, 255, 255, 0.9);
        border: 1px solid rgba(0, 0, 0, 0.06);
        border-radius: var(--fib-1);
        padding: var(--fib-3);
        display: flex;
        gap: var(--fib-2);
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    }

    .metric-card:hover {
        background: white;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
        transform: translateY(-2px);
    }

    .metric-icon {
        font-size: 32px;
        flex-shrink: 0;
    }

    .metric-content {
        flex: 1;
    }

    .metric-label {
        font-family: 'Courier Prime', monospace;
        font-size: 10px;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: #57534e;
        margin: 0 0 var(--fib-1);
    }

    .metric-value-large {
        font-family: Georgia, serif;
        font-size: 36px;
        font-weight: normal;
        line-height: 1.2;
        margin: var(--fib-1) 0;
    }

    .metric-value {
        font-family: 'Courier Prime', monospace;
        font-size: 20px;
        font-weight: 600;
        color: #1c1c1c;
        margin: var(--fib-1) 0;
    }

    .metric-subtitle {
        font-size: 12px;
        color: #57534e;
        margin-bottom: var(--fib-2);
    }

    /* Status Badge */
    .status-badge {
        display: inline-block;
        padding: 3px var(--fib-1);
        border-radius: 4px;
        font-size: 10px;
        font-family: 'Courier Prime', monospace;
        font-weight: 600;
        letter-spacing: 1px;
    }

    .status-badge.critical {
        background: rgba(220, 38, 38, 0.15);
        color: #dc2626;
    }

    .status-badge.warning {
        background: rgba(245, 158, 11, 0.15);
        color: #f59e0b;
    }

    .status-badge.safe {
        background: rgba(22, 163, 74, 0.15);
        color: #16a34a;
    }

    /* Metric Details */
    .metric-details {
        margin-top: var(--fib-2);
        padding-top: var(--fib-2);
        border-top: 1px solid rgba(0, 0, 0, 0.05);
    }

    .detail-row {
        display: flex;
        justify-content: space-between;
        font-size: 12px;
        margin-bottom: 4px;
        color: #57534e;
    }

    .detail-row strong {
        color: #1c1c1c;
    }

    /* Progress Bar */
    .progress-bar {
        width: 100%;
        height: 8px;
        background: rgba(0, 0, 0, 0.05);
        border-radius: 4px;
        overflow: hidden;
        margin: var(--fib-2) 0;
    }

    .progress-fill {
        height: 100%;
        background: linear-gradient(90deg, #16a34a, #22c55e);
        transition: width 0.5s ease-out;
    }

    .metric-percentage {
        font-size: 11px;
        color: #57534e;
        text-align: center;
    }

    /* Overdue List */
    .overdue-list {
        display: flex;
        flex-direction: column;
        gap: var(--fib-1);
        margin-top: var(--fib-2);
    }

    .overdue-item {
        display: flex;
        align-items: center;
        gap: var(--fib-1);
        font-size: 12px;
    }

    .grade-badge {
        display: inline-block;
        width: 24px;
        height: 24px;
        border-radius: 50%;
        text-align: center;
        line-height: 24px;
        font-weight: 600;
        font-size: 11px;
    }

    .grade-badge.grade-A {
        background: rgba(22, 163, 74, 0.15);
        color: #16a34a;
    }

    .grade-badge.grade-B {
        background: rgba(59, 130, 246, 0.15);
        color: #3b82f6;
    }

    .grade-badge.grade-C {
        background: rgba(245, 158, 11, 0.15);
        color: #f59e0b;
    }

    .grade-badge.grade-D {
        background: rgba(220, 38, 38, 0.15);
        color: #dc2626;
    }

    .overdue-amount {
        font-weight: 600;
        color: #1c1c1c;
        flex: 1;
    }

    .overdue-count {
        color: #57534e;
        font-size: 11px;
    }

    /* Empty State */
    .empty-state {
        text-align: center;
        padding: var(--fib-3) 0;
        color: #57534e;
        font-style: italic;
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
        .metrics-grid {
            grid-template-columns: 1fr;
        }

        .metric-value-large {
            font-size: 28px;
        }
    }
</style>
