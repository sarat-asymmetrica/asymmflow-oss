<script lang="ts">
    // C1 (Spec-04 gate ruling): opportunity edit-conflict resolution is
    // SALES admin tooling, not user/access administration — moved here out
    // of UserManagementScreen so that screen stays only about user/role/
    // access. Gated on the exact same server-side permission it used before
    // the move (CanResolveOpportunityConflicts) — no widening.
    // C5 (Wave 9 hardening): the Efficiency/activity-monitoring subtab that
    // used to live here has moved to DeploymentHub (the ops surface) — this
    // component is now single-purpose (Conflicts only).
    import { fade } from "svelte/transition";
    import WabiSpinner from "./ui/WabiSpinner.svelte";
    import { ListOpportunityEditConflicts, ResolveOpportunityEditConflict } from "../../../wailsjs/go/main/CRMService";
    import { toast } from "../stores/toasts";

    interface Props {
        canResolveOpportunityConflicts?: boolean;
    }

    let { canResolveOpportunityConflicts = false }: Props = $props();

    let loadingConflicts = $state(false);
    let opportunityConflicts: any[] = $state([]);
    let conflictStatusFilter = $state("pending");

    async function fetchOpportunityConflicts(force = false) {
        if (!canResolveOpportunityConflicts) return;
        if (opportunityConflicts.length > 0 && !force) return;
        loadingConflicts = true;
        try {
            if (window.go) {
                opportunityConflicts = await ListOpportunityEditConflicts(conflictStatusFilter, 100);
            }
        } catch (e) {
            toast.danger("Failed to load opportunity conflicts");
            opportunityConflicts = [];
        } finally {
            loadingConflicts = false;
        }
    }

    async function resolveOpportunityConflict(conflict: any, action: "apply" | "reject") {
        if (!conflict?.id) return;
        loadingConflicts = true;
        try {
            await ResolveOpportunityEditConflict(String(conflict.id), action, action === "apply" ? "Applied from admin review" : "Rejected from admin review");
            toast.success(action === "apply" ? "Conflict applied" : "Conflict rejected");
            await fetchOpportunityConflicts(true);
        } catch (e) {
            toast.danger(`Conflict resolution failed: ${e?.message || e}`);
        } finally {
            loadingConflicts = false;
        }
    }

    function conflictSummary(raw: string) {
        try {
            const data = JSON.parse(raw || "{}");
            return Object.entries(data)
                .filter(([, value]) => String(value || "").trim())
                .map(([key, value]) => `${key.replaceAll("_", " ")}: ${value}`)
                .join(" | ") || "No proposed changes";
        } catch {
            return raw || "No proposed changes";
        }
    }

    $effect(() => {
        if (canResolveOpportunityConflicts) fetchOpportunityConflicts();
    });
</script>

<div class="admin-tools">
    {#if canResolveOpportunityConflicts}
        <div class="activity-panel" in:fade>
            <div class="activity-toolbar">
                <div>
                    <h2>Opportunity Conflicts</h2>
                    <p>Concurrent opportunity edits flagged for admin decision.</p>
                </div>
                <div class="activity-controls">
                    <select bind:value={conflictStatusFilter} onchange={() => fetchOpportunityConflicts(true)}>
                        <option value="pending">Pending</option>
                        <option value="applied">Applied</option>
                        <option value="rejected">Rejected</option>
                        <option value="all">All</option>
                    </select>
                    <button class="btn-sm" onclick={() => fetchOpportunityConflicts(true)} disabled={loadingConflicts}>Refresh</button>
                </div>
            </div>
            {#if loadingConflicts}
                <div class="loading"><WabiSpinner size="lg" /></div>
            {:else if opportunityConflicts.length === 0}
                <div class="empty-state">No opportunity edit conflicts match this filter.</div>
            {:else}
                <div class="table-card compact">
                    <table>
                        <thead>
                            <tr><th>Opportunity</th><th>By</th><th>Operation</th><th>Versions</th><th>Proposed</th><th>Status</th><th class="right">Decision</th></tr>
                        </thead>
                        <tbody>
                            {#each opportunityConflicts as conflict}
                                <tr>
                                    <td class="bold">{conflict.folder_number || conflict.opportunity_id}</td>
                                    <td>{conflict.attempted_by || "User"} <span class="dim">({conflict.attempted_role || "role"})</span></td>
                                    <td>{conflict.operation}</td>
                                    <td>v{conflict.expected_version} → v{conflict.current_version}</td>
                                    <td class="dim">{conflictSummary(conflict.proposed_changes_json)}</td>
                                    <td><span class="badge neutral">{conflict.status}</span></td>
                                    <td class="right">
                                        {#if conflict.status === "pending"}
                                            <button class="btn-sm" onclick={() => resolveOpportunityConflict(conflict, "apply")}>Apply</button>
                                            <button class="btn-sm" onclick={() => resolveOpportunityConflict(conflict, "reject")}>Reject</button>
                                        {:else}
                                            <span class="dim">{conflict.resolution_action || "closed"}</span>
                                        {/if}
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
    {/if}
</div>

<style>
    .admin-tools {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

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
    .activity-controls select {
        border: 1px solid var(--border, #e5e5e5);
        border-radius: 6px;
        padding: 7px 9px;
        background: var(--surface, #fff);
        color: var(--text-primary, #111827);
        font-size: 12px;
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
    .right {
        text-align: right;
    }
    .bold {
        font-weight: 500;
    }
    .dim {
        color: var(--text-secondary, #6b7280);
    }

    .badge {
        padding: 2px 8px;
        border-radius: 4px;
        border: 1px solid var(--border, #e5e5e5);
        font-size: 10px;
        text-transform: uppercase;
    }
    .badge.neutral {
        background: rgba(0, 0, 0, 0.05);
        border: none;
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
