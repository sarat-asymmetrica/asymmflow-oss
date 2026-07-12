<script lang="ts">
    // C5 (Wave 9 hardening): relocated from SalesAdminTools' Efficiency
    // subtab to the Deployment/ops surface — cross-department activity
    // surveillance belongs with ops, not sales. Logic, data fetch, and the
    // CanViewUserActivityMonitoring gate are unchanged from the original;
    // only the mount point moved.
    import { fade } from "svelte/transition";
    import WabiSpinner from "./ui/WabiSpinner.svelte";
    import { GetWeeklyUserActivityReport } from "../../../wailsjs/go/main/InfraService";

    interface Props {
        canViewActivityMonitoring?: boolean;
    }

    let { canViewActivityMonitoring = false }: Props = $props();

    function currentWeekStart() {
        const d = new Date();
        const day = d.getDay() || 7;
        d.setDate(d.getDate() - day + 1);
        d.setHours(0, 0, 0, 0);
        return d.toISOString().slice(0, 10);
    }

    let loadingActivity = $state(false);
    let activityReport: any = $state(null);
    let activityError = $state("");
    let activityWeekStart = $state(currentWeekStart());

    async function fetchActivityReport(force = false) {
        if (!canViewActivityMonitoring) return;
        if (activityReport && !force) return;
        loadingActivity = true;
        activityError = "";
        try {
            if (window.go) {
                activityReport = await GetWeeklyUserActivityReport(activityWeekStart);
            }
        } catch (e) {
            activityReport = null;
            activityError = String(e);
        } finally {
            loadingActivity = false;
        }
    }

    function maxMeaningfulHours(rows: any[]) {
        return Math.max(1, ...((rows || []).map((row) => Number(row.meaningful_hours || 0))));
    }

    function barWidth(row: any, rows: any[]) {
        return `${Math.max(4, (Number(row.meaningful_hours || 0) / maxMeaningfulHours(rows)) * 100)}%`;
    }

    $effect(() => {
        if (canViewActivityMonitoring) fetchActivityReport();
    });
</script>

{#if canViewActivityMonitoring}
    <div class="activity-panel" in:fade>
        <div class="activity-toolbar">
            <div>
                <h2>Weekly Efficiency</h2>
                <p>{activityReport?.confidentiality_notice || "Confidential internal activity report."}</p>
            </div>
            <div class="activity-controls">
                <input type="date" bind:value={activityWeekStart} />
                <button class="btn-sm" onclick={() => fetchActivityReport(true)} disabled={loadingActivity}>Refresh</button>
            </div>
        </div>
        {#if loadingActivity}
            <div class="loading"><WabiSpinner size="lg" /></div>
        {:else if activityError}
            <div class="empty-state">{activityError}</div>
        {:else if !activityReport || (activityReport.users || []).length === 0}
            <div class="empty-state">No activity has been recorded for this week yet.</div>
        {:else}
            <div class="summary-grid">
                <div class="summary-tile">
                    <span>Active Hours</span>
                    <strong>{activityReport.total_active_hours}</strong>
                </div>
                <div class="summary-tile">
                    <span>Meaningful Hours</span>
                    <strong>{activityReport.total_meaningful_hours}</strong>
                </div>
                <div class="summary-tile">
                    <span>Efficiency</span>
                    <strong>{activityReport.average_efficiency}%</strong>
                </div>
                <div class="summary-tile">
                    <span>Users</span>
                    <strong>{activityReport.user_count}</strong>
                </div>
            </div>
            <div class="chart-list">
                {#each activityReport.chart_rows || [] as row}
                    <div class="chart-row">
                        <div class="chart-label">{row.label}</div>
                        <div class="chart-track">
                            <div class="chart-bar" style={`width: ${barWidth(row, activityReport.chart_rows || [])}`}></div>
                        </div>
                        <div class="chart-value">{row.meaningful_hours}h</div>
                        <div class="chart-eff">{row.efficiency_score}%</div>
                    </div>
                {/each}
            </div>
            <div class="table-card compact">
                <table>
                    <thead>
                        <tr><th>User</th><th>Role</th><th>Active</th><th>Meaningful</th><th>Searches</th><th>Creates</th><th>Updates</th><th>Exports</th></tr>
                    </thead>
                    <tbody>
                        {#each activityReport.users || [] as user}
                            <tr>
                                <td class="bold">{user.employee_name}</td>
                                <td><span class="badge role">{user.license_role || "user"}</span></td>
                                <td>{user.active_hours}h</td>
                                <td>{user.meaningful_hours}h</td>
                                <td>{user.search_count}</td>
                                <td>{user.create_count}</td>
                                <td>{user.update_count}</td>
                                <td>{user.export_count}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </div>
{/if}

<style>
    .loading {
        display: flex;
        justify-content: center;
        padding: 40px;
    }

    .activity-panel {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }
    .activity-toolbar {
        display: flex;
        align-items: flex-end;
        justify-content: space-between;
        gap: 16px;
        padding-bottom: 4px;
    }
    .activity-toolbar h2 {
        margin: 0 0 4px;
        font-size: 22px;
        font-weight: 500;
    }
    .activity-toolbar p {
        margin: 0;
        color: var(--text-secondary, #6b7280);
        font-size: 12px;
    }
    .activity-controls {
        display: flex;
        gap: 8px;
        align-items: center;
    }
    .activity-controls input {
        border: 1px solid var(--border, #e5e5e5);
        border-radius: 6px;
        padding: 7px 9px;
        background: var(--surface, #fff);
        color: var(--text-primary, #111827);
        font-size: 12px;
    }
    .summary-grid {
        display: grid;
        grid-template-columns: repeat(4, minmax(0, 1fr));
        gap: 12px;
    }
    .summary-tile {
        border: 1px solid var(--border, #e5e5e5);
        background: var(--surface, #fff);
        border-radius: 8px;
        padding: 14px;
    }
    .summary-tile span {
        display: block;
        font-size: 11px;
        color: var(--text-secondary, #6b7280);
        text-transform: uppercase;
        margin-bottom: 8px;
    }
    .summary-tile strong {
        font-size: 24px;
        font-weight: 500;
    }
    .chart-list {
        display: flex;
        flex-direction: column;
        gap: 10px;
        border: 1px solid var(--border, #e5e5e5);
        background: var(--surface, #fff);
        border-radius: 8px;
        padding: 14px;
    }
    .chart-row {
        display: grid;
        grid-template-columns: minmax(120px, 180px) 1fr 56px 56px;
        gap: 12px;
        align-items: center;
        font-size: 12px;
    }
    .chart-label {
        font-weight: 500;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .chart-track {
        height: 10px;
        border-radius: 999px;
        background: rgba(15, 23, 42, 0.08);
        overflow: hidden;
    }
    .chart-bar {
        height: 100%;
        border-radius: 999px;
        background: #0f766e;
    }
    .chart-value,
    .chart-eff {
        color: var(--text-secondary, #6b7280);
        text-align: right;
        font-variant-numeric: tabular-nums;
    }

    .table-card {
        background: var(--surface, #fff);
        border-radius: 12px;
        border: 1px solid var(--border, #e5e5e5);
        overflow: hidden;
    }
    table {
        width: 100%;
        border-collapse: collapse;
        font-size: 13px;
    }
    .compact table {
        font-size: 12px;
    }
    th {
        text-align: left;
        padding: 12px 16px;
        border-bottom: 1px solid var(--border, #e5e5e5);
        font-size: 11px;
        text-transform: uppercase;
        color: var(--text-secondary, #6b7280);
        font-weight: 500;
    }
    td {
        padding: 12px 16px;
        border-bottom: 1px solid var(--border, #e5e5e5);
        vertical-align: middle;
    }
    .bold {
        font-weight: 500;
    }

    .badge {
        padding: 2px 8px;
        border-radius: 4px;
        border: 1px solid var(--border, #e5e5e5);
        font-size: 10px;
        text-transform: uppercase;
    }
    .badge.role {
        background: var(--surface, #fff);
    }

    .btn-sm {
        padding: 4px 12px;
        border: 1px solid var(--border, #e5e5e5);
        background: var(--surface, #fff);
        border-radius: 4px;
        cursor: pointer;
        font-size: 11px;
    }

    .empty-state {
        border: 1px solid var(--border, #e5e5e5);
        background: var(--surface, #fff);
        border-radius: 8px;
        padding: 24px;
        color: var(--text-secondary, #6b7280);
        text-align: center;
    }
</style>
