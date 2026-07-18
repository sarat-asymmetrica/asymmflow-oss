<script lang="ts">
  /* Deployment Hub — internal ops/pilot console (K4 operational hub).
   * TabShell over three independent surfaces sharing one pilot-readiness
   * workspace: Audit (readiness list + license-reassignment fix-it form +
   * read-only Deployment Data Audit panel), Checklist (pilot deployment
   * checklist, toggle + notes per item), Support (sync/retry/export controls
   * + collaborative queue). All state/derivation/mutation-calls live in
   * deployment-vm.svelte.ts (L5); this file only composes primitives and
   * renders (L1). See screens/parity/DeploymentHub.parity.md.
   *
   * The old screen's Activity tab (weekly per-employee user-activity/
   * "efficiency" monitor) is RETIRED entirely — owner-ratified,
   * surveillance-adjacent, out of scope for the OSS kernel tranche. */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import TabShell from '$kernel/primitives/TabShell.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Toolbar from '$kernel/primitives/Toolbar.svelte'
  import FormGrid from '$kernel/primitives/FormGrid.svelte'
  import DataTable from '$kernel/primitives/DataTable.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import ConfirmDialog from '$kernel/controls/ConfirmDialog.svelte'
  import SearchInput from '$kernel/controls/SearchInput.svelte'
  import FilterChips from '$kernel/controls/FilterChips.svelte'
  import StatTileGrid from '$kernel/widgets/StatTileGrid.svelte'
  import DistributionWidget from '$kernel/widgets/DistributionWidget.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import type { ColumnSpec, StatusSpec } from '$kernel/descriptor'
  import type { Tone } from '$kernel/tones'
  import { formatDate } from '$kernel/format'
  import {
    AUDIT_FILTER_OPTIONS,
    DeploymentHubViewModel,
    QUEUE_FILTER_OPTIONS,
    formatIssue,
    queueStatusTone,
    type DeploymentTab,
  } from './deployment-vm.svelte'
  import type { CollaborativePendingOperation, PilotChecklistItem, PilotReadinessRow } from '../bridge/deployment'

  const vm = new DeploymentHubViewModel()
  onMount(() => void vm.load())

  const READINESS_STATUS: StatusSpec<PilotReadinessRow> = {
    value: (r) => (r.readyForPilot ? 'Ready' : 'Needs Attention'),
    tones: { Ready: 'success', 'Needs Attention': 'danger' },
  }

  const readinessColumns: ColumnSpec<PilotReadinessRow>[] = [
    { key: 'employeeName', label: 'Employee', content: 'name', value: (r) => r.employeeName || 'Unknown Employee', grow: true, minWidth: 180 },
    { key: 'employeeCode', label: 'Code', content: 'code', value: (r) => r.employeeCode, minWidth: 100 },
    { key: 'department', label: 'Department', content: 'text', value: (r) => r.department, minWidth: 150 },
    { key: 'jobTitle', label: 'Job Title', content: 'text', value: (r) => r.jobTitle, minWidth: 150 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => (r.readyForPilot ? 'Ready' : 'Needs Attention'), minWidth: 130 },
  ]

  const QUEUE_STATUS: StatusSpec<CollaborativePendingOperation> = {
    value: (o) => o.status,
    tones: { pending: 'warning', failed: 'danger', dead_letter: 'danger', synced: 'success' },
  }

  const queueColumns: ColumnSpec<CollaborativePendingOperation>[] = [
    { key: 'status', label: 'Status', content: 'status', value: (o) => o.status, minWidth: 110 },
    { key: 'entity', label: 'Entity', content: 'code', value: (o) => `${o.entityType}:${o.entityId}`, grow: true, minWidth: 200 },
    { key: 'operation', label: 'Operation', content: 'text', value: (o) => o.operation, minWidth: 100 },
    { key: 'attempts', label: 'Attempts', content: 'quantity', value: (o) => o.attempts, minWidth: 90 },
    { key: 'updatedAt', label: 'Updated', content: 'date', value: (o) => o.updatedAt, minWidth: 110 },
  ]

  const auditToneOf = (blocking: boolean | undefined): Tone => (blocking ? 'danger' : 'success')

  const bulkRetryLabel = $derived(vm.bulkRetryStatus === 'dead_letter' ? 'dead-letter' : 'failed')

  function checklistNotesValue(item: PilotChecklistItem): string {
    return vm.checklistNotesDraft[item.id] ?? item.notes
  }
</script>

{#snippet auditTab()}
  <Stack gap="lg">
    {#if vm.audit}
      <Card>
        <Stack gap="md">
          <Row justify="between" wrap>
            <span class="dh-section-label">Deployment Data Audit</span>
            <Badge tone={auditToneOf(vm.audit.blocking)} label={vm.audit.blocking ? 'Blocking' : 'Verified'} />
          </Row>

          <Stack gap="xs">
            <span class="dh-meta">Current DB: {vm.audit.databasePath || '—'}</span>
            <span class="dh-meta">Expected Runtime DB: {vm.audit.expectedRuntimeDatabasePath || '—'}</span>
            <span class="dh-meta">Packaged DB: {vm.audit.packagedDatabasePath || '—'}</span>
          </Stack>

          <StatTileGrid
            sections={[
              {
                items: [
                  { label: 'Task Items', value: vm.audit.taskItems, content: 'quantity' },
                  { label: 'Expense Entries', value: vm.audit.expenseEntries, content: 'quantity' },
                  { label: 'Payroll Runs', value: vm.audit.payrollRuns, content: 'quantity' },
                  { label: 'Offers Hidden', value: vm.audit.legacyQuotedOfferShells + vm.audit.legacyRfqOfferShells, content: 'quantity' },
                ],
              },
            ]}
          />

          <Grid min="260px" gap="md">
            <Stack gap="sm">
              <span class="dh-subsection-label">Missing Tables</span>
              {#if vm.audit.missingTables.length === 0}
                <Badge tone="success" label="No missing deployment tables" />
              {:else}
                <Row gap="xs" wrap>
                  {#each vm.audit.missingTables as table (table)}
                    <Badge tone="danger" label={table} />
                  {/each}
                </Row>
              {/if}
            </Stack>

            <Stack gap="sm">
              <span class="dh-subsection-label">Blocking Data Issues</span>
              {#if vm.audit.blockingDataIssues.length === 0}
                <CalloutWidget items={[{ label: 'Clear', text: 'No blocking data issues.', tone: 'success' }]} />
              {:else}
                <CalloutWidget items={vm.audit.blockingDataIssues.map((issue) => ({ label: 'Blocking', text: issue, tone: 'danger' as Tone }))} />
              {/if}
            </Stack>

            <Stack gap="sm">
              <span class="dh-subsection-label">Warnings</span>
              {#if vm.audit.warningDataIssues.length === 0}
                <CalloutWidget items={[{ label: 'Clear', text: 'No deployment warnings.', tone: 'success' }]} />
              {:else}
                <CalloutWidget items={vm.audit.warningDataIssues.map((issue) => ({ label: 'Warning', text: issue, tone: 'warning' as Tone }))} />
              {/if}
            </Stack>
          </Grid>
        </Stack>
      </Card>
    {/if}

    <Grid min="380px" gap="lg">
      <Stack gap="md">
        <Card>
          <Toolbar>
            <SearchInput bind:value={vm.search} placeholder="Search employees, licenses, devices…" />
            {#snippet trailing()}
              <FilterChips label="Filter" options={AUDIT_FILTER_OPTIONS} bind:selected={vm.auditFilter} />
            {/snippet}
          </Toolbar>
        </Card>
        <Card padding="none">
          {#if vm.filteredRows.length === 0}
            <EmptyState message="No rollout records match the current filters." />
          {:else}
            <DataTable
              columns={readinessColumns}
              rows={vm.filteredRows}
              id={(r) => r.employeeId}
              status={READINESS_STATUS}
              selectedId={vm.selectedEmployeeId}
              onSelect={(r) => vm.selectEmployee(r)}
            />
          {/if}
        </Card>
      </Stack>

      <Card>
        {#if !vm.selectedRow}
          <EmptyState message="Select an employee to inspect rollout readiness." />
        {:else}
          {@const row = vm.selectedRow}
          <Stack gap="lg">
            <Row justify="between" wrap>
              <Stack gap="xs">
                <span class="dh-employee-name">{row.employeeName || 'Unknown Employee'}</span>
                <span class="dh-meta">{row.department || 'No department'} · {row.jobTitle || row.employeeCode || 'No role title'}</span>
              </Stack>
              <Badge tone={row.readyForPilot ? 'success' : 'danger'} label={row.readyForPilot ? 'Ready for Pilot' : 'Needs Attention'} />
            </Row>

            {#if row.issues.length === 0}
              <Badge tone="success" label="No active rollout blockers" />
            {:else}
              <Row gap="xs" wrap>
                {#each row.issues as issue (issue)}
                  <Badge tone="danger" label={formatIssue(issue)} />
                {/each}
              </Row>
            {/if}

            <StatTileGrid
              sections={[
                {
                  items: [
                    { label: 'Access', value: row.accessStatus || 'unlinked' },
                    { label: 'License', value: row.licenseKey || 'Unassigned' },
                    { label: 'Device', value: row.deviceName || row.deviceId || 'No device' },
                    { label: 'User', value: row.userName || 'No linked user' },
                  ],
                },
              ]}
            />

            <span class="dh-meta">Last seen: {formatDate(row.lastSeenAt)}</span>

            <Card>
              <Stack gap="md">
                <span class="dh-section-label">Access Reassignment — direct pilot fix</span>

                <FormGrid columns={2}>
                  <label class="k-field">
                    <span class="k-field-label">License Key</span>
                    <select
                      class="k-input"
                      value={vm.selectedLicenseKey}
                      onchange={(e) => vm.selectLicenseKey(e.currentTarget.value)}
                    >
                      <option value="">Select license key</option>
                      {#each vm.licenseKeys as license (license.key)}
                        <option value={license.key}>{license.key} · {license.displayName || 'Unassigned'} · {license.role}</option>
                      {/each}
                    </select>
                  </label>
                  <label class="k-field k-field-row">
                    <input type="checkbox" bind:checked={vm.syncLicenseName} />
                    <span class="k-field-label">Sync license name to employee</span>
                  </label>
                </FormGrid>

                <FormGrid columns={2}>
                  <label class="k-field">
                    <span class="k-field-label">License Display Name</span>
                    <input class="k-input" bind:value={vm.licenseDisplayNameDraft} placeholder="License display name" />
                  </label>
                </FormGrid>

                {#if vm.reassignError}
                  <CalloutWidget items={[{ label: 'Reassignment failed', text: vm.reassignError, tone: 'danger' }]} />
                {/if}
                {#if vm.licenseNameError}
                  <CalloutWidget items={[{ label: 'Save failed', text: vm.licenseNameError, tone: 'danger' }]} />
                {/if}

                <Row justify="end" gap="sm">
                  <Button onclick={() => vm.saveLicenseDisplayName()} disabled={vm.licenseNameBusy || !vm.selectedLicenseKey}>
                    {vm.licenseNameBusy ? 'Saving…' : 'Save License Name'}
                  </Button>
                  <Button variant="primary" onclick={() => vm.reassignLicense()} disabled={vm.reassignBusy || !vm.selectedLicenseKey}>
                    {vm.reassignBusy ? 'Updating…' : 'Reassign License'}
                  </Button>
                </Row>
              </Stack>
            </Card>
          </Stack>
        {/if}
      </Card>
    </Grid>
  </Stack>
{/snippet}

{#snippet checklistTab()}
  <Stack gap="lg">
    <Row justify="between" wrap>
      <span class="dh-section-label">Pilot Deployment Checklist</span>
      <span class="dh-meta">{vm.completedChecklistCount} / {vm.checklist.length} complete</span>
    </Row>

    {#if vm.checklistError}
      <CalloutWidget items={[{ label: 'Checklist update failed', text: vm.checklistError, tone: 'danger' }]} />
    {/if}

    <Grid min="340px" gap="md">
      {#each vm.checklist as item (item.id)}
        <Card>
          <Stack gap="sm">
            <Row justify="between" wrap align="start">
              <label class="k-field k-field-row">
                <input
                  type="checkbox"
                  checked={item.completed}
                  disabled={vm.checklistSavingId === item.id}
                  onchange={(e) => vm.toggleChecklistItem(item, e.currentTarget.checked)}
                />
                <Stack gap="xs">
                  <span class="dh-item-title">{item.title}</span>
                  <span class="dh-meta">{item.description}</span>
                </Stack>
              </label>
              {#if item.completedAt}
                <span class="dh-meta">{formatDate(item.completedAt)}</span>
              {/if}
            </Row>

            <label class="k-field">
              <span class="k-field-label">Notes</span>
              <textarea
                class="k-input k-input-area"
                rows="3"
                placeholder="Add rollout notes, blockers, or sign-off context…"
                value={checklistNotesValue(item)}
                oninput={(e) => vm.setChecklistNote(item.id, e.currentTarget.value)}
              ></textarea>
            </label>

            <Row justify="end">
              <Button onclick={() => vm.saveChecklistNotes(item)} disabled={vm.checklistSavingId === item.id}>
                {vm.checklistSavingId === item.id ? 'Saving…' : 'Save Notes'}
              </Button>
            </Row>
          </Stack>
        </Card>
      {/each}
    </Grid>
  </Stack>
{/snippet}

{#snippet supportTab()}
  <Stack gap="lg">
    <Card>
      <Stack gap="md">
        <span class="dh-section-label">Support Actions</span>
        <Row gap="sm" wrap>
          <Button variant="primary" onclick={() => vm.triggerSync()} disabled={vm.syncBusy}>
            {vm.syncBusy ? 'Working…' : 'Run Collaborative Sync'}
          </Button>
          <Button onclick={() => vm.requestBulkRetry('failed')} disabled={vm.bulkRetryBusy}>Retry Failed Ops</Button>
          <Button onclick={() => vm.requestBulkRetry('dead_letter')} disabled={vm.bulkRetryBusy}>Revive Dead Letters</Button>
          <Button onclick={() => vm.exportSupportBundle()} disabled={vm.exportingBundle}>
            {vm.exportingBundle ? 'Exporting…' : 'Export Support Bundle'}
          </Button>
          <Button onclick={() => vm.exportSignoffReport()} disabled={vm.exportingSignoff}>
            {vm.exportingSignoff ? 'Exporting…' : 'Export Sign-Off Report'}
          </Button>
        </Row>

        {#if vm.rollout}
          <StatTileGrid
            sections={[
              {
                items: [
                  { label: 'Legacy Follow-Ups', value: vm.rollout.legacyFollowupTasks, content: 'quantity' },
                  { label: 'Migrated Tasks', value: vm.rollout.migratedLegacyTasks, content: 'quantity' },
                  { label: 'Pending Ops', value: vm.rollout.pendingCollaborativeOps, content: 'quantity' },
                  { label: 'Awaiting Payroll Recon', value: vm.rollout.payrollPayoutsAwaitingRecon, content: 'quantity' },
                ],
              },
            ]}
          />
        {/if}

        {#if vm.syncError}
          <CalloutWidget items={[{ label: 'Sync failed', text: vm.syncError, tone: 'danger' }]} />
        {:else if vm.syncMessage}
          <CalloutWidget items={[{ label: 'Sync', text: vm.syncMessage, tone: 'success' }]} />
        {/if}
        {#if vm.bulkRetryError}
          <CalloutWidget items={[{ label: 'Bulk retry failed', text: vm.bulkRetryError, tone: 'danger' }]} />
        {:else if vm.bulkRetryMessage}
          <CalloutWidget items={[{ label: 'Bulk retry', text: vm.bulkRetryMessage, tone: 'success' }]} />
        {/if}
        {#if vm.exportBundleError}
          <CalloutWidget items={[{ label: 'Export failed', text: vm.exportBundleError, tone: 'danger' }]} />
        {:else if vm.exportBundleMessage}
          <CalloutWidget items={[{ label: 'Support bundle', text: vm.exportBundleMessage, tone: 'success' }]} />
        {/if}
        {#if vm.exportSignoffError}
          <CalloutWidget items={[{ label: 'Export failed', text: vm.exportSignoffError, tone: 'danger' }]} />
        {:else if vm.exportSignoffMessage}
          <CalloutWidget items={[{ label: 'Sign-off report', text: vm.exportSignoffMessage, tone: 'success' }]} />
        {/if}
      </Stack>
    </Card>

    {#if vm.queueDistribution.length > 0}
      <Card>
        <Stack gap="sm">
          <span class="dh-section-label">Queue by Status</span>
          <DistributionWidget segments={vm.queueDistribution} />
        </Stack>
      </Card>
    {/if}

    <Stack gap="md">
      <Card>
        <Toolbar>
          <label class="k-field">
            <span class="k-field-label">Collaborative Queue</span>
            <select class="k-input" value={vm.queueFilter} onchange={(e) => vm.setQueueFilter(e.currentTarget.value)}>
              {#each QUEUE_FILTER_OPTIONS as opt (opt.value)}
                <option value={opt.value}>{opt.label}</option>
              {/each}
            </select>
          </label>
          {#snippet trailing()}
            <Button onclick={() => vm.loadQueue()} disabled={vm.queueLoading}>
              {vm.queueLoading ? 'Refreshing…' : 'Refresh Queue'}
            </Button>
          {/snippet}
        </Toolbar>
      </Card>
      <Card padding="none">
        {#if vm.queueOps.length === 0}
          <EmptyState message="No collaborative queue items match the current filter." />
        {:else}
          <DataTable
            columns={queueColumns}
            rows={vm.queueOps}
            id={(o) => o.id}
            status={QUEUE_STATUS}
            selectedId={vm.selectedOpId}
            onSelect={(o) => vm.selectOp(o)}
          />
        {/if}
      </Card>
    </Stack>

    {#if vm.selectedOp}
      {@const op = vm.selectedOp}
      <Card>
        <Stack gap="sm">
          <Row justify="between" wrap>
            <span class="dh-section-label">Selected Operation — {op.entityType}:{op.entityId}</span>
            <Badge tone={queueStatusTone(op.status)} label={op.status} />
          </Row>
          {#if op.errorMessage}
            <CalloutWidget items={[{ label: 'Error', text: op.errorMessage, tone: 'danger' }]} />
          {/if}
          {#if vm.retryError}
            <CalloutWidget items={[{ label: 'Retry failed', text: vm.retryError, tone: 'danger' }]} />
          {/if}
          <Row justify="end">
            <Button
              variant="primary"
              onclick={() => vm.retrySelectedOp()}
              disabled={vm.retryBusy || !(op.status === 'failed' || op.status === 'dead_letter')}
            >
              {vm.retryBusy ? 'Retrying…' : 'Retry This Operation'}
            </Button>
          </Row>
        </Stack>
      </Card>
    {/if}
  </Stack>
{/snippet}

<PageShell title="Deployment" subtitle="Pilot rollout audit, employee-license-device readiness, and support controls.">
  {#if vm.loading}
    <EmptyState message="Loading deployment workspace…" />
  {:else if vm.error}
    <EmptyState message={`Could not load deployment workspace: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else}
    <TabShell
      activeKey={vm.activeTab}
      onSelect={(k) => vm.setTab(k as DeploymentTab)}
      tabs={[
        { key: 'audit', label: 'Audit', content: auditTab },
        { key: 'checklist', label: 'Checklist', badge: vm.checklist.length, content: checklistTab },
        { key: 'support', label: 'Support', content: supportTab },
      ]}
    >
      {#snippet header()}
        <Stack gap="sm">
          <StatTileGrid
            sections={[
              {
                title: 'Deployment Hub',
                items: [
                  { label: 'Deployment Audit', value: vm.audit?.blocking ? 'Blocked' : 'Clear', tone: auditToneOf(vm.audit?.blocking) },
                  { label: 'Pilot Readiness', value: `${vm.summary?.readyEmployees ?? 0} / ${vm.summary?.totalEmployees ?? 0}` },
                  {
                    label: 'Needs Attention',
                    value: vm.summary?.employeesWithIssues ?? 0,
                    content: 'quantity',
                    tone: (vm.summary?.employeesWithIssues ?? 0) > 0 ? 'warning' : 'success',
                  },
                  { label: 'Checklist Progress', value: `${vm.completedChecklistCount} / ${vm.checklist.length}` },
                  {
                    label: 'Queue Health',
                    value: `${vm.rollout?.pendingCollaborativeOps ?? 0} pending`,
                    tone: (vm.rollout?.failedCollaborativeOps ?? 0) + (vm.rollout?.deadLetterCollaborativeOps ?? 0) > 0 ? 'danger' : 'success',
                  },
                ],
              },
            ]}
          />
          <Row justify="end">
            <Button onclick={() => vm.refresh()} disabled={vm.loading}>Refresh</Button>
          </Row>
        </Stack>
      {/snippet}
    </TabShell>
  {/if}
</PageShell>

{#if vm.bulkRetryStatus}
  <ConfirmDialog
    title="Retry {bulkRetryLabel} operations?"
    message={`This re-queues up to 100 ${bulkRetryLabel} collaborative operations for another sync attempt — a bulk resync-storm against every affected device. This cannot be undone once triggered.`}
    confirmLabel="Retry All"
    danger
    onConfirm={() => vm.confirmBulkRetry()}
    onCancel={() => vm.cancelBulkRetry()}
  />
{/if}

<style>
  /* Typography only (L1) — layout/spacing lives in primitives; native form
   * controls use the kernel-owned .k-field/.k-field-label/.k-input classes
   * (single-source in styles/kernel.css), not per-screen CSS. */
  .dh-section-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
  .dh-subsection-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    color: var(--text-secondary);
  }
  .dh-employee-name {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
  .dh-item-title {
    font-weight: 600;
    color: var(--text-primary);
    overflow-wrap: break-word;
  }
  .dh-meta {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
</style>
