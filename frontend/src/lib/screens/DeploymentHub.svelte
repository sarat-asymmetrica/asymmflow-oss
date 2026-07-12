<script lang="ts">
  import { run } from 'svelte/legacy';

  import { onDestroy, onMount } from "svelte";
  import { EventsOff, EventsOn } from "../../../wailsjs/runtime/runtime";
  import { toast } from "../stores/toasts";
  import WabiSpinner from "../components/ui/WabiSpinner.svelte";
  import UserActivityMonitorPanel from "../components/UserActivityMonitorPanel.svelte";
  import {
    ExportPilotSupportBundle } from "../../../wailsjs/go/main/App";
import { CanViewUserActivityMonitoring, ExportPilotSignoffReport, GetDeploymentDataAudit, GetPhase7RolloutStatus, GetPilotDeploymentChecklist, GetPilotReadinessSummary, ListCollaborativePendingOperations, ListLicenseKeys, ListPilotReadinessRows, RetryCollaborativePendingOperation, RetryCollaborativePendingOperations, TriggerCollaborativeSyncNow, UpdateLicenseDisplayName, UpdatePilotDeploymentChecklistItem } from "../../../wailsjs/go/main/InfraService";
import { ReassignEmployeeLicenseAccess } from "../../../wailsjs/go/main/SyncServiceBinding";

  // C5 (Wave 9 hardening): activity monitoring relocated here from
  // SalesAdminTools — cross-department surveillance belongs on the ops
  // surface, not under Sales. Same server-side gate as before the move
  // (CanViewUserActivityMonitoring); DeploymentHub's route additionally
  // requires settings:update, so this panel is now behind both gates.
  type DeploymentTab = "audit" | "checklist" | "support" | "activity";

  let canViewActivityMonitoring = $state(false);

  let loading = $state(true);
  let refreshing = $state(false);
  let activeTab: DeploymentTab = $state("audit");

  let readinessSummary: any = $state(null);
  let rolloutStatus: any = $state(null);
  let deploymentAudit: any = $state(null);
  let readinessRows: any[] = $state([]);
  let checklist: any[] = $state([]);
  let checklistNotesDraft: Record<string, string> = $state({});
  let queueOps: any[] = $state([]);
  let licenseKeys: any[] = $state([]);

  let issuesOnly = $state(true);
  let search = $state("");
  let selectedEmployeeID = $state("");
  let queueFilter = $state("active");
  let selectedLicenseKey = $state("");
  let syncLicenseName = $state(true);
  let licenseDisplayNameDraft = $state("");
  let lastLicenseDraftKey = $state("");

  let loadingQueue = $state(false);
  let actionRunning = $state(false);
  let checklistSavingID = $state("");
  let exportingBundle = $state(false);
  let exportingSignoff = $state(false);

  function syncChecklistDrafts(items: any[]) {
    const nextDrafts: Record<string, string> = {};
    for (const item of items || []) {
      if (item?.id) {
        nextDrafts[item.id] = item.notes || "";
      }
    }
    checklistNotesDraft = nextDrafts;
  }

  async function loadQueue() {
    loadingQueue = true;
    try {
      queueOps = await ListCollaborativePendingOperations(queueFilter, 20) || [];
    } catch {
      queueOps = [];
    } finally {
      loadingQueue = false;
    }
  }

  async function loadAll(showSpinner = true) {
    if (showSpinner) {
      loading = true;
    } else {
      refreshing = true;
    }

    try {
      const [summary, rows, rollout, checklistRows, licenseRows, dataAudit] = await Promise.all([
        GetPilotReadinessSummary(),
        ListPilotReadinessRows(false),
        GetPhase7RolloutStatus(),
        GetPilotDeploymentChecklist(),
        ListLicenseKeys().catch(() => []),
        GetDeploymentDataAudit().catch(() => null),
      ]);

      readinessSummary = summary;
      readinessRows = rows || [];
      rolloutStatus = rollout;
      deploymentAudit = dataAudit;
      checklist = checklistRows || [];
      syncChecklistDrafts(checklist);
      licenseKeys = licenseRows || [];

      const visibleRows = applyReadinessFilters(readinessRows);
      if (!selectedEmployeeID || !visibleRows.some((row) => row.employee_id === selectedEmployeeID)) {
        selectedEmployeeID = visibleRows[0]?.employee_id || readinessRows[0]?.employee_id || "";
      }

      await loadQueue();
    } catch (err) {
      toast.danger(`Failed to load deployment workspace: ${String(err)}`);
    } finally {
      loading = false;
      refreshing = false;
    }
  }

  function applyReadinessFilters(rows: any[]) {
    const term = search.trim().toLowerCase();
    return rows.filter((row) => {
      if (issuesOnly && (!Array.isArray(row.issues) || row.issues.length === 0)) {
        return false;
      }
      if (!term) {
        return true;
      }
      const haystack = [
        row.employee_name,
        row.employee_code,
        row.department,
        row.job_title,
        row.license_key,
        row.device_name,
        row.user_name,
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return haystack.includes(term);
    });
  }

  async function refreshWorkspace() {
    await loadAll(false);
  }

  async function handleChecklistToggle(item: any, nextCompleted: boolean) {
    checklistSavingID = item.id;
    try {
      checklist = await UpdatePilotDeploymentChecklistItem(item.id, nextCompleted, checklistNotesDraft[item.id] || "");
      syncChecklistDrafts(checklist);
      toast.success(nextCompleted ? "Checklist item completed" : "Checklist item reopened");
    } catch (err) {
      toast.danger(`Failed to update checklist item: ${String(err)}`);
    } finally {
      checklistSavingID = "";
    }
  }

  async function handleChecklistNotesSave(item: any) {
    checklistSavingID = item.id;
    try {
      checklist = await UpdatePilotDeploymentChecklistItem(item.id, !!item.completed, checklistNotesDraft[item.id] || "");
      syncChecklistDrafts(checklist);
      toast.success("Checklist notes saved");
    } catch (err) {
      toast.danger(`Failed to save checklist notes: ${String(err)}`);
    } finally {
      checklistSavingID = "";
    }
  }

  async function handleTriggerSync() {
    actionRunning = true;
    try {
      await TriggerCollaborativeSyncNow();
      toast.success("Collaborative sync completed");
      await refreshWorkspace();
    } catch (err) {
      toast.danger(`Collaborative sync failed: ${String(err)}`);
    } finally {
      actionRunning = false;
    }
  }

  async function handleRetryQueue(status: string) {
    actionRunning = true;
    try {
      const result = await RetryCollaborativePendingOperations(status, 100);
      toast.success(result?.message || "Collaborative queue re-queued");
      await refreshWorkspace();
    } catch (err) {
      toast.danger(`Failed to retry queue: ${String(err)}`);
    } finally {
      actionRunning = false;
    }
  }

  async function handleRetrySingle(operationID: string) {
    actionRunning = true;
    try {
      await RetryCollaborativePendingOperation(operationID);
      toast.success("Collaborative operation re-queued");
      await refreshWorkspace();
    } catch (err) {
      toast.danger(`Failed to retry operation: ${String(err)}`);
    } finally {
      actionRunning = false;
    }
  }

  async function handleExportBundle() {
    exportingBundle = true;
    try {
      const result = await ExportPilotSupportBundle();
      toast.success(`Support bundle exported to ${result?.path || "reports directory"}`);
    } catch (err) {
      toast.danger(`Failed to export support bundle: ${String(err)}`);
    } finally {
      exportingBundle = false;
    }
  }

  async function handleExportSignoff() {
    exportingSignoff = true;
    try {
      const result = await ExportPilotSignoffReport();
      toast.success(`Pilot sign-off report exported to ${result?.path || "reports directory"}`);
    } catch (err) {
      toast.danger(`Failed to export sign-off report: ${String(err)}`);
    } finally {
      exportingSignoff = false;
    }
  }

  async function handleLicenseReassign() {
    if (!selectedRow?.employee_id) {
      toast.warning("Select an employee first");
      return;
    }
    if (!selectedLicenseKey) {
      toast.warning("Choose a license key to assign");
      return;
    }

    actionRunning = true;
    try {
      await ReassignEmployeeLicenseAccess(selectedRow.employee_id, selectedLicenseKey, syncLicenseName);
      toast.success("Employee license assignment updated");
      await refreshWorkspace();
    } catch (err) {
      toast.danger(`Failed to reassign employee license: ${String(err)}`);
    } finally {
      actionRunning = false;
    }
  }

  async function handleLicenseNameSave() {
    if (!selectedLicenseKey) {
      toast.warning("Choose a license key first");
      return;
    }
    if (!licenseDisplayNameDraft.trim()) {
      toast.warning("License display name cannot be empty");
      return;
    }

    actionRunning = true;
    try {
      await UpdateLicenseDisplayName(selectedLicenseKey, licenseDisplayNameDraft.trim());
      toast.success("License display name updated");
      await refreshWorkspace();
    } catch (err) {
      toast.danger(`Failed to update license display name: ${String(err)}`);
    } finally {
      actionRunning = false;
    }
  }

  function formatIssue(issue: string): string {
    if (!issue) return "Unknown issue";
    return issue.replace(/_/g, " ").replace(/\b\w/g, (match) => match.toUpperCase());
  }

  function formatDate(value?: string) {
    if (!value) return "—";
    return new Date(value).toLocaleString();
  }

  let filteredRows = $derived(applyReadinessFilters(readinessRows));
  let selectedRow = $derived(filteredRows.find((row) => row.employee_id === selectedEmployeeID)
    || readinessRows.find((row) => row.employee_id === selectedEmployeeID)
    || null);
  let selectedLicenseRecord = $derived(licenseKeys.find((license) => license.key === selectedLicenseKey) || null);
  let completedChecklistCount = $derived(checklist.filter((item) => item.completed).length);
  run(() => {
    if (selectedRow) {
      selectedLicenseKey = selectedRow.license_key || selectedLicenseKey || licenseKeys[0]?.key || "";
    }
  });
  run(() => {
    if (selectedLicenseKey !== lastLicenseDraftKey) {
      licenseDisplayNameDraft = selectedLicenseRecord?.display_name || "";
      lastLicenseDraftKey = selectedLicenseKey;
    }
  });

  onMount(() => {
    loadAll();
    if (window.go) {
      CanViewUserActivityMonitoring().then((allowed) => {
        canViewActivityMonitoring = allowed;
      }).catch(() => {
        canViewActivityMonitoring = false;
      });
    }
    EventsOn("employees:updated", refreshWorkspace);
    EventsOn("tasks:updated", refreshWorkspace);
    EventsOn("notifications:new", refreshWorkspace);
    EventsOn("notifications:updated", refreshWorkspace);
  });

  onDestroy(() => {
    EventsOff("employees:updated");
    EventsOff("tasks:updated");
    EventsOff("notifications:new");
    EventsOff("notifications:updated");
  });
</script>

<div class="page">
  <header class="header">
    <div>
      <h1>Deployment.</h1>
      <p class="subtitle">Pilot rollout audit, employee-license-device readiness, and support controls.</p>
    </div>
    <div class="header-actions">
      <button class="btn-secondary" onclick={handleExportSignoff} disabled={exportingSignoff}>
        {exportingSignoff ? "Exporting..." : "Export Sign-Off"}
      </button>
      <button class="btn-ghost" onclick={refreshWorkspace} disabled={refreshing || loading}>
        {refreshing ? "Refreshing..." : "Refresh"}
      </button>
      <button class="btn-primary" onclick={handleExportBundle} disabled={exportingBundle}>
        {exportingBundle ? "Exporting..." : "Export Bundle"}
      </button>
    </div>
  </header>

  {#if loading}
    <div class="loading-state"><WabiSpinner /></div>
  {:else}
    <section class="summary-grid">
      <article class="summary-card" class:warn={!!deploymentAudit?.blocking}>
        <span class="summary-label">Deployment Audit</span>
        <strong>{deploymentAudit?.blocking ? "Blocked" : "Clear"}</strong>
        <small>{deploymentAudit?.missing_tables?.length ?? 0} missing tables, {deploymentAudit?.blocking_data_issues?.length ?? 0} blocking data issues.</small>
      </article>
      <article class="summary-card">
        <span class="summary-label">Pilot Readiness</span>
        <strong>{readinessSummary?.ready_employees ?? 0} / {readinessSummary?.total_employees ?? 0}</strong>
        <small>Employees fully ready for pilot rollout.</small>
      </article>
      <article class="summary-card warn">
        <span class="summary-label">Needs Attention</span>
        <strong>{readinessSummary?.employees_with_issues ?? 0}</strong>
        <small>Employees with access, device, or user mapping gaps.</small>
      </article>
      <article class="summary-card">
        <span class="summary-label">Checklist Progress</span>
        <strong>{completedChecklistCount} / {checklist.length}</strong>
        <small>Deployment preparation tasks completed on this workspace.</small>
      </article>
      <article class="summary-card">
        <span class="summary-label">Queue Health</span>
        <strong>{rolloutStatus?.pending_collaborative_ops ?? 0} pending</strong>
        <small>{rolloutStatus?.failed_collaborative_ops ?? 0} failed, {rolloutStatus?.dead_letter_collaborative_ops ?? 0} dead letters.</small>
      </article>
    </section>

    {#if deploymentAudit}
      <section class="panel deployment-audit-panel">
        <div class="panel-head">
          <h2>Deployment Data Audit</h2>
          <span class={`state-pill ${deploymentAudit.blocking ? "" : "ready"}`}>
            {deploymentAudit.blocking ? "Blocking" : "Verified"}
          </span>
        </div>

        <div class="detail-meta">
          <div><span>Current DB:</span> <strong>{deploymentAudit.database_path || "—"}</strong></div>
          <div><span>Expected Runtime DB:</span> <strong>{deploymentAudit.expected_runtime_database_path || "—"}</strong></div>
          <div><span>Packaged DB:</span> <strong>{deploymentAudit.packaged_database_path || "—"}</strong></div>
        </div>

        <div class="status-grid audit-counts">
          <article class="detail-card">
            <span class="detail-label">Task Items</span>
            <strong>{deploymentAudit.task_items ?? 0}</strong>
            <small>{deploymentAudit.legacy_followup_tasks ?? 0} legacy follow-ups, {deploymentAudit.migrated_legacy_tasks ?? 0} migrated.</small>
          </article>
          <article class="detail-card">
            <span class="detail-label">Expense Entries</span>
            <strong>{deploymentAudit.expense_entries ?? 0}</strong>
            <small>Schema materialized for expense workflow.</small>
          </article>
          <article class="detail-card">
            <span class="detail-label">Payroll Runs</span>
            <strong>{deploymentAudit.payroll_runs ?? 0}</strong>
            <small>Schema materialized for payroll workflow.</small>
          </article>
          <article class="detail-card">
            <span class="detail-label">Offers Hidden</span>
            <strong>{(deploymentAudit.legacy_quoted_offer_shells ?? 0) + (deploymentAudit.legacy_rfq_offer_shells ?? 0)}</strong>
            <small>Legacy quoted/RFQ shells excluded from the live offer list.</small>
          </article>
        </div>

        <div class="audit-detail-grid">
          <article class="audit-detail-card">
            <h3>Missing Tables</h3>
            {#if deploymentAudit.missing_tables?.length}
              <div class="issue-list">
                {#each deploymentAudit.missing_tables as table}
                  <span class="issue-pill">{table}</span>
                {/each}
              </div>
            {:else}
              <span class="issue-pill success">No missing deployment tables</span>
            {/if}
          </article>

          <article class="audit-detail-card">
            <h3>Blocking Data Issues</h3>
            {#if deploymentAudit.blocking_data_issues?.length}
              <div class="audit-lines">
                {#each deploymentAudit.blocking_data_issues as issue}
                  <div>{issue}</div>
                {/each}
              </div>
            {:else}
              <span class="issue-pill success">No blocking data issues</span>
            {/if}
          </article>

          <article class="audit-detail-card">
            <h3>Warnings</h3>
            {#if deploymentAudit.warning_data_issues?.length}
              <div class="audit-lines">
                {#each deploymentAudit.warning_data_issues as issue}
                  <div>{issue}</div>
                {/each}
              </div>
            {:else}
              <span class="issue-pill success">No deployment warnings</span>
            {/if}
          </article>
        </div>
      </section>
    {/if}

    <section class="tabs">
      <button class:active={activeTab === "audit"} onclick={() => (activeTab = "audit")}>Audit</button>
      <button class:active={activeTab === "checklist"} onclick={() => (activeTab = "checklist")}>Checklist</button>
      <button class:active={activeTab === "support"} onclick={() => (activeTab = "support")}>Support</button>
      {#if canViewActivityMonitoring}
        <button class:active={activeTab === "activity"} onclick={() => (activeTab = "activity")}>Activity</button>
      {/if}
    </section>

    {#if activeTab === "audit"}
      <section class="workspace two-column">
        <div class="panel">
          <div class="panel-head">
            <h2>Employee Audit</h2>
            <span>{filteredRows.length} visible</span>
          </div>
          <div class="toolbar">
            <input aria-label="Search pilot readiness" bind:value={search} placeholder="Search employees, licenses, devices..." />
            <label class="toggle-inline" for="issues-only">
              <input id="issues-only" type="checkbox" bind:checked={issuesOnly} />
              Issues only
            </label>
          </div>

          <div class="audit-list">
            {#if filteredRows.length === 0}
              <div class="empty-state">No rollout records match the current filters.</div>
            {:else}
              {#each filteredRows as row}
                <button
                  class="audit-row"
                  class:selected={selectedEmployeeID === row.employee_id}
                  onclick={() => (selectedEmployeeID = row.employee_id)}
                >
                  <div>
                    <strong>{row.employee_name || "Unknown Employee"}</strong>
                    <small>{row.department || "No department"} • {row.job_title || row.employee_code || "No role title"}</small>
                  </div>
                  <span class:ready={row.ready_for_pilot} class="state-pill">
                    {row.ready_for_pilot ? "Ready" : `${(row.issues || []).length} issues`}
                  </span>
                </button>
              {/each}
            {/if}
          </div>
        </div>

        <div class="panel detail-panel">
          <div class="panel-head">
            <h2>Detail</h2>
            <span>{selectedRow?.employee_code || "No employee selected"}</span>
          </div>
          {#if selectedRow}
            <div class="detail-header">
              <div>
                <h3>{selectedRow.employee_name}</h3>
                <p>{selectedRow.department || "No department"} • {selectedRow.job_title || selectedRow.employment_state || "No title"}</p>
              </div>
              <span class:ready={selectedRow.ready_for_pilot} class="state-pill large">
                {selectedRow.ready_for_pilot ? "Ready for Pilot" : "Needs Attention"}
              </span>
            </div>

            <div class="issue-list">
              {#if selectedRow.issues?.length}
                {#each selectedRow.issues as issue}
                  <span class="issue-pill">{formatIssue(issue)}</span>
                {/each}
              {:else}
                <span class="issue-pill success">No active rollout blockers</span>
              {/if}
            </div>

            <div class="detail-grid">
              <article class="detail-card">
                <span class="detail-label">Access</span>
                <strong>{selectedRow.access_status || "unlinked"}</strong>
                <small>Employment state: {selectedRow.employment_state || "—"}</small>
              </article>
              <article class="detail-card">
                <span class="detail-label">License</span>
                <strong>{selectedRow.license_key || "Unassigned"}</strong>
                <small>{selectedRow.license_role || (selectedRow.license_active ? "Activated" : "Not activated") || "—"}</small>
              </article>
              <article class="detail-card">
                <span class="detail-label">Device</span>
                <strong>{selectedRow.device_name || selectedRow.device_id || "No device"}</strong>
                <small>{selectedRow.device_status || "Unknown status"}</small>
              </article>
              <article class="detail-card">
                <span class="detail-label">User</span>
                <strong>{selectedRow.user_name || "No linked user"}</strong>
                <small>{selectedRow.user_id || "Unlinked"}</small>
              </article>
            </div>

            <div class="detail-meta">
              <div><span>Assigned To:</span> <strong>{selectedRow.license_assigned || "—"}</strong></div>
              <div><span>Last Seen:</span> <strong>{formatDate(selectedRow.last_seen_at)}</strong></div>
            </div>

            <div class="assignment-card">
              <div class="panel-head">
                <h2>Access Reassignment</h2>
                <span>Direct pilot fix</span>
              </div>
              <div class="toolbar">
                <select bind:value={selectedLicenseKey} aria-label="Select license key">
                  {#each licenseKeys as license}
                    <option value={license.key}>
                      {license.key} • {license.display_name || "Unassigned"} • {license.role}
                    </option>
                  {/each}
                </select>
                <label class="toggle-inline" for="sync-license-name">
                  <input id="sync-license-name" type="checkbox" bind:checked={syncLicenseName} />
                  Sync license name to employee
                </label>
              </div>
              <div class="toolbar">
                <input bind:value={licenseDisplayNameDraft} aria-label="License display name" placeholder="License display name" />
                <button class="btn-ghost" onclick={handleLicenseNameSave} disabled={actionRunning || !selectedLicenseKey}>
                  Save License Name
                </button>
                <button class="btn-primary" onclick={handleLicenseReassign} disabled={actionRunning || !selectedLicenseKey}>
                  {actionRunning ? "Updating..." : "Reassign License"}
                </button>
              </div>
            </div>
          {:else}
            <div class="empty-state">Select an employee to inspect rollout readiness.</div>
          {/if}
        </div>
      </section>
    {:else if activeTab === "activity" && canViewActivityMonitoring}
      <section class="workspace">
        <div class="panel">
          <UserActivityMonitorPanel {canViewActivityMonitoring} />
        </div>
      </section>
    {:else if activeTab === "checklist"}
      <section class="workspace">
        <div class="panel">
          <div class="panel-head">
            <h2>Pilot Checklist</h2>
            <div class="toolbar compact">
              <span>{completedChecklistCount} complete</span>
              <button class="btn-secondary" onclick={handleExportSignoff} disabled={exportingSignoff}>
                {exportingSignoff ? "Exporting..." : "Export Sign-Off"}
              </button>
            </div>
          </div>
          <div class="checklist-grid">
            {#each checklist as item}
              <article class="checklist-card">
                <div class="checklist-head">
                  <label class="check-toggle" for={`check-${item.id}`}>
                    <input
                      id={`check-${item.id}`}
                      type="checkbox"
                      checked={!!item.completed}
                      disabled={checklistSavingID === item.id}
                      onchange={(event) => handleChecklistToggle(item, event.currentTarget.checked)}
                    />
                    <div>
                      <strong>{item.title}</strong>
                      <p>{item.description}</p>
                    </div>
                  </label>
                  {#if item.completed_at}
                    <span class="check-date">{new Date(item.completed_at).toLocaleString()}</span>
                  {/if}
                </div>
                <textarea
                  aria-label={`Notes for ${item.title}`}
                  value={checklistNotesDraft[item.id] ?? item.notes ?? ""}
                  rows="3"
                  placeholder="Add rollout notes, blockers, or sign-off context..."
                  oninput={(event) => {
                    checklistNotesDraft = {
                      ...checklistNotesDraft,
                      [item.id]: event.currentTarget.value,
                    };
                  }}
                ></textarea>
                <div class="checklist-actions">
                  <button class="btn-ghost" onclick={() => handleChecklistNotesSave(item)} disabled={checklistSavingID === item.id}>
                    {checklistSavingID === item.id ? "Saving..." : "Save Notes"}
                  </button>
                </div>
              </article>
            {/each}
          </div>
        </div>
      </section>
    {:else}
      <section class="workspace two-column support-layout">
        <div class="panel">
          <div class="panel-head">
            <h2>Support Actions</h2>
            <span>Recovery & diagnostics</span>
          </div>
          <div class="support-actions">
            <button class="btn-primary" onclick={handleTriggerSync} disabled={actionRunning}>
              {actionRunning ? "Working..." : "Run Collaborative Sync"}
            </button>
            <button class="btn-secondary" onclick={() => handleRetryQueue("failed")} disabled={actionRunning}>
              Retry Failed Ops
            </button>
            <button class="btn-secondary" onclick={() => handleRetryQueue("dead_letter")} disabled={actionRunning}>
              Revive Dead Letters
            </button>
            <button class="btn-ghost" onclick={handleExportBundle} disabled={exportingBundle}>
              {exportingBundle ? "Exporting..." : "Export Support Bundle"}
            </button>
          </div>

          <div class="status-grid">
            <article class="detail-card">
              <span class="detail-label">Legacy Follow-Ups</span>
              <strong>{rolloutStatus?.legacy_followup_tasks ?? 0}</strong>
            </article>
            <article class="detail-card">
              <span class="detail-label">Migrated Tasks</span>
              <strong>{rolloutStatus?.migrated_legacy_tasks ?? 0}</strong>
            </article>
            <article class="detail-card">
              <span class="detail-label">Pending Ops</span>
              <strong>{rolloutStatus?.pending_collaborative_ops ?? 0}</strong>
            </article>
            <article class="detail-card">
              <span class="detail-label">Awaiting Payroll Recon</span>
              <strong>{rolloutStatus?.payroll_payouts_awaiting_recon ?? 0}</strong>
            </article>
          </div>
        </div>

        <div class="panel">
          <div class="panel-head">
            <h2>Collaborative Queue</h2>
            <div class="toolbar compact">
              <select bind:value={queueFilter} onchange={loadQueue}>
                <option value="active">Active Issues</option>
                <option value="pending">Pending</option>
                <option value="failed">Failed</option>
                <option value="dead_letter">Dead Letter</option>
                <option value="synced">Recently Synced</option>
              </select>
              <button class="btn-ghost" onclick={loadQueue} disabled={loadingQueue}>
                {loadingQueue ? "Refreshing..." : "Refresh Queue"}
              </button>
            </div>
          </div>

          <div class="queue-table">
            <div class="queue-row head">
              <span>Status</span>
              <span>Entity</span>
              <span>Operation</span>
              <span>Attempts</span>
              <span>Action</span>
            </div>
            {#if queueOps.length === 0}
              <div class="empty-state compact">No collaborative queue items match the current filter.</div>
            {:else}
              {#each queueOps as op}
                <div class="queue-row">
                  <span class={`state-pill ${op.status || ""}`}>{op.status || "unknown"}</span>
                  <span>{op.entity_type}:{op.entity_id?.slice?.(0, 8) || op.entity_id}</span>
                  <span>{op.operation}</span>
                  <span>{op.attempts ?? 0}</span>
                  <span>
                    {#if op.status === "failed" || op.status === "dead_letter"}
                      <button class="btn-ghost" onclick={() => handleRetrySingle(op.id)} disabled={actionRunning}>Retry</button>
                    {:else}
                      <small>{formatDate(op.updated_at)}</small>
                    {/if}
                  </span>
                </div>
                {#if op.error_message}
                  <div class="queue-error">{op.error_message}</div>
                {/if}
              {/each}
            {/if}
          </div>
        </div>
      </section>
    {/if}
  {/if}
</div>

<style>
  .page {
    padding: var(--page-padding);
    display: flex;
    flex-direction: column;
    gap: 18px;
    min-height: 100%;
    background: var(--bg-base, #f5f5f7);
    color: var(--text-primary, #1d1d1f);
  }
  .header,
  .panel-head,
  .toolbar,
  .checklist-actions,
  .checklist-head,
  .detail-header,
  .detail-meta,
  .header-actions,
  .support-actions {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .header-actions,
  .support-actions,
  .toolbar.compact {
    flex-wrap: wrap;
  }
  h1,
  h2,
  h3,
  p {
    margin: 0;
  }
  h1 {
    font-size: 28px;
    font-weight: 300;
  }
  h2 {
    font-size: 16px;
    font-weight: 600;
  }
  h3 {
    font-size: 20px;
    font-weight: 600;
  }
  .subtitle,
  .summary-label,
  .detail-label,
  small,
  .check-date {
    color: var(--text-muted, #6b7280);
  }
  .summary-grid,
  .detail-grid,
  .status-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 14px;
  }
  .summary-card,
  .panel,
  .detail-card,
  .checklist-card,
  .assignment-card {
    background: var(--surface, #fff);
    border: 1px solid var(--border, #e5e7eb);
    border-radius: 16px;
  }
  .summary-card,
  .detail-card {
    padding: 16px;
  }
  .summary-card strong,
  .detail-card strong {
    display: block;
    margin: 8px 0 4px;
    font-size: 24px;
    color: var(--onyx, #111827);
  }
  .summary-card.warn {
    border-color: rgba(180, 35, 24, 0.2);
  }
  .tabs {
    display: flex;
    gap: 10px;
  }
  .tabs button,
  .btn-primary,
  .btn-secondary,
  .btn-ghost,
  .audit-row {
    border: none;
    cursor: pointer;
    transition: 0.2s ease;
    font: inherit;
  }
  .tabs button {
    padding: 10px 16px;
    border-radius: 999px;
    background: rgba(17, 24, 39, 0.06);
    color: var(--text-muted, #6b7280);
  }
  .tabs button.active {
    background: #111827;
    color: #fff;
  }
  .workspace.two-column {
    display: grid;
    grid-template-columns: 420px minmax(0, 1fr);
    gap: 16px;
  }
  .support-layout {
    grid-template-columns: minmax(0, 0.95fr) minmax(0, 1.05fr);
  }
  .panel {
    padding: 18px;
    display: flex;
    flex-direction: column;
    gap: 14px;
    min-height: 0;
  }
  .toolbar {
    flex-wrap: wrap;
  }
  input,
  select,
  textarea {
    width: 100%;
    box-sizing: border-box;
    padding: 10px 12px;
    border-radius: 10px;
    border: 1px solid var(--border, #e5e7eb);
    background: #fff;
    color: inherit;
    font: inherit;
  }
  textarea {
    resize: vertical;
    min-height: 88px;
  }
  .toggle-inline,
  .check-toggle {
    display: inline-flex;
    align-items: flex-start;
    gap: 10px;
  }
  .toggle-inline {
    color: var(--text-muted, #6b7280);
    white-space: nowrap;
  }
  .toggle-inline input,
  .check-toggle input {
    width: auto;
    margin: 2px 0 0;
  }
  .audit-list,
  .queue-table {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .audit-row {
    width: 100%;
    padding: 14px 16px;
    border-radius: 14px;
    background: rgba(17, 24, 39, 0.03);
    display: flex;
    align-items: center;
    justify-content: space-between;
    text-align: left;
  }
  .audit-row.selected,
  .audit-row:hover {
    background: rgba(17, 24, 39, 0.08);
  }
  .state-pill,
  .issue-pill {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 5px 10px;
    border-radius: 999px;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    background: rgba(180, 35, 24, 0.12);
    color: #b42318;
  }
  .state-pill.ready,
  .issue-pill.success,
  .state-pill.synced {
    background: rgba(2, 122, 72, 0.12);
    color: #027a48;
  }
  .state-pill.pending {
    background: rgba(180, 35, 24, 0.08);
    color: #b42318;
  }
  .state-pill.failed,
  .state-pill.dead_letter {
    background: rgba(127, 29, 29, 0.16);
    color: #7f1d1d;
  }
  .state-pill.large {
    padding: 8px 12px;
    font-size: 12px;
  }
  .detail-panel {
    gap: 18px;
  }
  .deployment-audit-panel {
    gap: 16px;
  }
  .audit-counts {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
  .audit-detail-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 14px;
  }
  .audit-detail-card {
    padding: 16px;
    border-radius: 16px;
    background: rgba(17, 24, 39, 0.03);
    border: 1px solid var(--border, #e5e7eb);
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .audit-lines {
    display: flex;
    flex-direction: column;
    gap: 8px;
    color: var(--onyx, #111827);
  }
  .issue-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .detail-meta {
    justify-content: flex-start;
    flex-wrap: wrap;
  }
  .detail-meta span {
    color: var(--text-muted, #6b7280);
  }
  .checklist-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px;
  }
  .checklist-card {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .assignment-card {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .check-toggle p {
    margin-top: 4px;
    color: var(--text-muted, #6b7280);
    line-height: 1.45;
  }
  .btn-primary,
  .btn-secondary,
  .btn-ghost {
    padding: 10px 14px;
    border-radius: 10px;
  }
  .btn-primary {
    background: #111827;
    color: #fff;
  }
  .btn-secondary {
    background: rgba(17, 24, 39, 0.08);
    color: #111827;
  }
  .btn-ghost {
    background: transparent;
    border: 1px solid var(--border, #e5e7eb);
    color: #111827;
  }
  .btn-primary:disabled,
  .btn-secondary:disabled,
  .btn-ghost:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }
  .queue-row {
    display: grid;
    grid-template-columns: 110px 1.3fr 0.8fr 70px 0.9fr;
    gap: 10px;
    align-items: center;
    padding: 12px 0;
    border-bottom: 1px solid var(--border, #e5e7eb);
    font-size: 12px;
  }
  .queue-row.head {
    font-weight: 700;
    text-transform: uppercase;
    color: var(--text-muted, #6b7280);
    letter-spacing: 0.04em;
    border-bottom: none;
    padding-top: 0;
  }
  .queue-error {
    margin-top: -4px;
    margin-bottom: 8px;
    color: #b42318;
    font-size: 12px;
    white-space: pre-wrap;
  }
  .loading-state,
  .empty-state {
    min-height: 220px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-muted, #6b7280);
    border: 1px dashed var(--border, #e5e7eb);
    border-radius: 16px;
    background: rgba(255, 255, 255, 0.7);
  }
  .empty-state.compact {
    min-height: 120px;
  }
  @media (max-width: 1280px) {
    .workspace.two-column,
    .checklist-grid,
    .summary-grid,
    .detail-grid,
    .status-grid,
    .audit-counts,
    .audit-detail-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
