<script lang="ts">
  /* Payroll — bespoke-on-primitives (K4/K5 tranche), PII hot-zone. Three
   * ViewSwitcher modes (Compensation / Runs / Payouts) over one division-
   * scoped dataset. Compensation = upsert form + DataTable. Runs = period-
   * create form, a periods/runs split, and the selected run's approve→
   * post→pay lifecycle on the new Stepper primitive (stats + per-employee
   * items + the payment sub-form live in its `detail` slot). Payouts = a
   * flat DataTable. All state/derivation/mutation-calls live in
   * payroll-vm.svelte.ts (L5); this file only composes primitives and
   * renders (L1). See screens/parity/Payroll.parity.md. */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Toolbar from '$kernel/primitives/Toolbar.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import FormGrid from '$kernel/primitives/FormGrid.svelte'
  import DataTable from '$kernel/primitives/DataTable.svelte'
  import ViewSwitcher from '$kernel/primitives/ViewSwitcher.svelte'
  import Stepper from '$kernel/primitives/Stepper.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import ConfirmDialog from '$kernel/controls/ConfirmDialog.svelte'
  import FilterChips from '$kernel/controls/FilterChips.svelte'
  import StatTileGrid from '$kernel/widgets/StatTileGrid.svelte'
  import DistributionWidget from '$kernel/widgets/DistributionWidget.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import type { ColumnSpec, StatusSpec } from '$kernel/descriptor'
  import type { Tone } from '$kernel/tones'
  import {
    PayrollViewModel,
    profileDeductions,
    profileGrossPay,
    runItemDisplayName,
    type PayrollMode,
  } from './payroll-vm.svelte'
  import type { CompensationProfile, PayrollPayout, PayrollPeriod, PayrollRunItem, PayrollRunSummary } from '../bridge/payroll'

  const vm = new PayrollViewModel()
  onMount(() => void vm.load())

  const VIEWS: { key: PayrollMode; label: string }[] = [
    { key: 'compensation', label: 'Compensation' },
    { key: 'runs', label: 'Runs' },
    { key: 'payouts', label: 'Payouts' },
  ]

  const PROFILE_STATUS: StatusSpec<CompensationProfile> = {
    value: (p) => p.status,
    tones: { Active: 'success', Inactive: 'neutral' },
  }
  const RUN_STATUS: StatusSpec<PayrollRunSummary> = {
    value: (r) => r.status,
    tones: { draft: 'neutral', approved: 'warning', posted: 'info', paid: 'success' },
  }
  const PAYOUT_STATUS: StatusSpec<PayrollPayout> = {
    value: (p) => p.status,
    tones: { scheduled: 'warning', paid: 'success' },
  }
  const RUN_BADGE_TONE: Record<string, Tone> = { draft: 'neutral', approved: 'warning', posted: 'info', paid: 'success' }

  // ---- PII masking helpers (field-masking is NET-NEW — see
  // canViewUnmasked's doc comment in payroll-vm.svelte.ts). Every money/name
  // column on a per-employee row routes through these so a future granular
  // permission needs no further plumbing; run-level AGGREGATE totals in the
  // StatTileGrid stay unmasked (not individually identifiable). ----
  function moneyCol<T>(
    key: string,
    label: string,
    get: (r: T) => number,
    opts: { minWidth?: number; currency?: (r: T) => string } = {},
  ): ColumnSpec<T> {
    if (!vm.canViewUnmasked) {
      return { key, label, content: 'text', value: () => '••••••', minWidth: opts.minWidth ?? 120 }
    }
    return {
      key,
      label,
      content: 'money',
      value: get,
      minWidth: opts.minWidth ?? 120,
      ...(opts.currency ? { currency: opts.currency } : {}),
    }
  }
  function nameCol<T>(key: string, label: string, get: (r: T) => string): ColumnSpec<T> {
    if (!vm.canViewUnmasked) {
      return { key, label, content: 'text', value: () => '•••••', grow: true, minWidth: 160 }
    }
    return { key, label, content: 'name', value: get, grow: true, minWidth: 160 }
  }

  const profileColumns: ColumnSpec<CompensationProfile>[] = $derived([
    nameCol('employeeName', 'Employee', (p) => p.employeeName),
    { key: 'jobTitle', label: 'Job Title', content: 'text', value: (p) => p.jobTitle, minWidth: 140 },
    { key: 'division', label: 'Division', content: 'text', value: (p) => p.division, minWidth: 150 },
    { key: 'payFrequency', label: 'Frequency', content: 'text', value: (p) => p.payFrequency, minWidth: 100 },
    moneyCol('gross', 'Gross', (p) => profileGrossPay(p), { currency: (p) => p.currency }),
    moneyCol('deductions', 'Deductions', (p) => profileDeductions(p), { currency: (p) => p.currency }),
    moneyCol('employerCost', 'Employer Cost', (p) => p.employerCost, { currency: (p) => p.currency }),
    { key: 'status', label: 'Status', content: 'status', value: (p) => p.status, minWidth: 100 },
  ])

  const periodColumns: ColumnSpec<PayrollPeriod>[] = [
    { key: 'name', label: 'Period', content: 'name', value: (p) => p.name, grow: true, minWidth: 160 },
    { key: 'division', label: 'Division', content: 'text', value: (p) => p.division, minWidth: 140 },
    { key: 'periodStart', label: 'Start', content: 'date', value: (p) => p.periodStart, minWidth: 100 },
    { key: 'periodEnd', label: 'End', content: 'date', value: (p) => p.periodEnd, minWidth: 100 },
    { key: 'status', label: 'Status', content: 'status', value: (p) => p.status, minWidth: 90 },
  ]
  const periodStatus: StatusSpec<PayrollPeriod> = {
    value: (p) => p.status,
    tones: { open: 'info', closed: 'neutral' },
  }

  const runColumns: ColumnSpec<PayrollRunSummary>[] = [
    { key: 'runNumber', label: 'Run', content: 'code', value: (r) => r.runNumber, minWidth: 130 },
    { key: 'division', label: 'Division', content: 'text', value: (r) => r.division, minWidth: 140 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 100 },
    { key: 'netTotal', label: 'Net', content: 'money', value: (r) => r.netTotal, currency: (r) => r.currency, minWidth: 130 },
  ]

  const itemColumns: ColumnSpec<PayrollRunItem>[] = $derived([
    nameCol('employee', 'Employee', (i) => runItemDisplayName(i)),
    { key: 'job', label: 'Job Title', content: 'text', value: (i) => i.jobTitleSnapshot, minWidth: 140 },
    moneyCol('gross', 'Gross', (i) => i.grossPay),
    moneyCol('deductions', 'Deductions', (i) => i.deductionsTotal),
    moneyCol('net', 'Net', (i) => i.netPay),
    { key: 'payoutStatus', label: 'Payout', content: 'text', value: (i) => i.payoutStatus || 'scheduled', minWidth: 110 },
  ])

  const payoutColumns: ColumnSpec<PayrollPayout>[] = $derived([
    { key: 'runNumber', label: 'Run', content: 'code', value: (p) => p.runNumber || 'Payroll run', minWidth: 130 },
    nameCol('employeeName', 'Employee', (p) => p.employeeName),
    { key: 'division', label: 'Division', content: 'text', value: (p) => p.division, minWidth: 140 },
    { key: 'scheduledAt', label: 'Scheduled', content: 'date', value: (p) => p.scheduledAt, minWidth: 100 },
    { key: 'paidAt', label: 'Paid', content: 'date', value: (p) => p.paidAt, minWidth: 100 },
    moneyCol('amount', 'Amount', (p) => p.amount, { currency: (p) => p.currency }),
    { key: 'status', label: 'Status', content: 'status', value: (p) => p.status, minWidth: 100 },
  ])

  const STEPS = [
    { key: 'draft', label: 'Draft', description: 'Generated from active profiles' },
    { key: 'approved', label: 'Approved', description: 'Reviewed and signed off' },
    { key: 'posted', label: 'Posted', description: 'Posted to the GL' },
    { key: 'paid', label: 'Paid', description: 'Salary transfer logged' },
  ]

  const stepperActions = $derived([
    { key: 'approve', label: 'Approve', enabledFrom: ['draft'], onAction: () => vm.requestApprove(), variant: 'primary' as const },
    { key: 'post', label: 'Post', enabledFrom: ['approved'], onAction: () => vm.requestPost(), variant: 'primary' as const },
    {
      key: 'pay',
      label: 'Mark Paid',
      // PRESERVE + FLAG: enabled from BOTH 'approved' and 'posted' — the old
      // screen did not strictly require Post before Pay. Kept, not fixed;
      // see Payroll.parity.md.
      enabledFrom: ['approved', 'posted'],
      onAction: () => vm.markPaid(),
      variant: 'ghost' as const,
    },
  ])

  const profileFormTitle = $derived(vm.editingProfileId ? 'Edit Compensation Profile' : 'New Compensation Profile')
</script>

<PageShell title="Payroll" subtitle="Compensation profiles, payroll runs, and payout tracking.">
  {#snippet toolbar()}
    <Toolbar>
      <FilterChips label="Division" options={vm.divisions} bind:selected={vm.divisionFilter} />
      {#snippet trailing()}
        <Button onclick={() => (vm.canViewUnmasked = !vm.canViewUnmasked)}>
          {vm.canViewUnmasked ? 'Mask Salaries' : 'Unmask Salaries'}
        </Button>
      {/snippet}
    </Toolbar>
  {/snippet}

  {#if vm.loading}
    <EmptyState message="Loading payroll workspace…" />
  {:else if vm.error}
    <EmptyState message={`Could not load payroll: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else}
    <Stack gap="lg">
      <StatTileGrid
        sections={[
          {
            title: 'Payroll',
            items: [
              { label: 'Active Profiles', value: vm.summary.activeProfiles },
              { label: 'Open Periods', value: vm.summary.openPeriods },
              { label: 'Approved / Posted', value: vm.summary.approvedOrPosted },
              { label: 'Upcoming Liability', value: vm.summary.upcomingLiability, content: 'money' },
            ],
          },
        ]}
      />

      {#if vm.runDistribution.length}
        <Card>
          <Stack gap="sm">
            <span class="pr-section-label">Runs by State</span>
            <DistributionWidget segments={vm.runDistribution} />
          </Stack>
        </Card>
      {/if}

      <ViewSwitcher views={VIEWS} activeKey={vm.mode} onSelect={(k) => vm.setMode(k as PayrollMode)} />

      {#if vm.mode === 'compensation'}
        <Stack gap="lg">
          <Card>
            <Stack gap="md">
              <Row justify="between" wrap>
                <span class="pr-section-label">{profileFormTitle}</span>
                <Button onclick={() => vm.resetProfileForm()}>New Profile</Button>
              </Row>

              <FormGrid columns={3}>
                <label class="k-field">
                  <span class="k-field-label">Employee</span>
                  <select class="k-input" bind:value={vm.profileDraft.employeeId} disabled={!!vm.editingProfileId}>
                    <option value="">Select employee</option>
                    {#each vm.employees as employee (employee.id)}
                      <option value={employee.id}>{employee.name || '(no name on file)'}</option>
                    {/each}
                  </select>
                </label>
                <label class="k-field">
                  <span class="k-field-label">Pay Frequency</span>
                  <select class="k-input" bind:value={vm.profileDraft.payFrequency}>
                    <option value="monthly">Monthly</option>
                    <option value="biweekly">Biweekly</option>
                    <option value="weekly">Weekly</option>
                  </select>
                </label>
                <label class="k-field">
                  <span class="k-field-label">Division</span>
                  <select class="k-input" bind:value={vm.profileDraft.division}>
                    <option value="">Use employee's division</option>
                    {#each vm.divisions as d (d.value)}
                      <option value={d.value}>{d.label}</option>
                    {/each}
                  </select>
                </label>

                <label class="k-field">
                  <span class="k-field-label">Base Salary</span>
                  <input class="k-input" type="number" step="0.001" bind:value={vm.profileDraft.baseSalary} />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Employer Cost</span>
                  <input class="k-input" type="number" step="0.001" bind:value={vm.profileDraft.employerCost} />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Housing Allowance</span>
                  <input class="k-input" type="number" step="0.001" bind:value={vm.profileDraft.housingAllowance} />
                </label>

                <label class="k-field">
                  <span class="k-field-label">Transport Allowance</span>
                  <input class="k-input" type="number" step="0.001" bind:value={vm.profileDraft.transportAllowance} />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Other Allowance</span>
                  <input class="k-input" type="number" step="0.001" bind:value={vm.profileDraft.otherAllowance} />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Standard Deduction</span>
                  <input class="k-input" type="number" step="0.001" bind:value={vm.profileDraft.standardDeduction} />
                </label>

                <label class="k-field">
                  <span class="k-field-label">Tax Deduction</span>
                  <input class="k-input" type="number" step="0.001" bind:value={vm.profileDraft.taxDeduction} />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Effective From</span>
                  <input class="k-input" type="date" bind:value={vm.profileDraft.effectiveFrom} />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Effective To</span>
                  <input class="k-input" type="date" bind:value={vm.profileDraft.effectiveTo} />
                </label>
              </FormGrid>

              <label class="k-field k-field-row">
                <input type="checkbox" bind:checked={vm.profileDraft.isActive} />
                <span class="k-field-label">Active profile</span>
              </label>

              <label class="k-field">
                <span class="k-field-label">Notes</span>
                <textarea class="k-input" rows="2" bind:value={vm.profileDraft.notes}></textarea>
              </label>

              {#if vm.profileError}
                <CalloutWidget items={[{ label: 'Save failed', text: vm.profileError, tone: 'danger' }]} />
              {/if}

              <Row justify="end">
                <Button variant="primary" onclick={() => vm.saveProfile()} disabled={vm.savingProfile}>
                  {vm.savingProfile ? 'Saving…' : vm.editingProfileId ? 'Update Profile' : 'Save Profile'}
                </Button>
              </Row>
            </Stack>
          </Card>

          <Card padding="none">
            {#if vm.scopedProfiles.length === 0}
              <EmptyState message="No compensation profiles for this division yet." />
            {:else}
              <DataTable
                columns={profileColumns}
                rows={vm.scopedProfiles}
                id={(p) => p.id}
                status={PROFILE_STATUS}
                selectedId={vm.editingProfileId || null}
                onSelect={(p) => vm.editProfile(p)}
              />
            {/if}
          </Card>
        </Stack>
      {:else if vm.mode === 'runs'}
        <Stack gap="lg">
          <Card>
            <Stack gap="md">
              <span class="pr-section-label">New Payroll Period</span>
              <FormGrid columns={3}>
                <label class="k-field">
                  <span class="k-field-label">Period Name</span>
                  <input class="k-input" bind:value={vm.periodDraft.name} placeholder="Jul 2026 Payroll" />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Division</span>
                  <select class="k-input" bind:value={vm.periodDraft.division}>
                    {#each vm.divisions as d (d.value)}
                      <option value={d.value}>{d.label}</option>
                    {/each}
                  </select>
                </label>
                <label class="k-field">
                  <span class="k-field-label">Payment Date</span>
                  <input class="k-input" type="date" bind:value={vm.periodDraft.paymentDate} />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Period Start</span>
                  <input class="k-input" type="date" bind:value={vm.periodDraft.periodStart} />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Period End</span>
                  <input class="k-input" type="date" bind:value={vm.periodDraft.periodEnd} />
                </label>
              </FormGrid>
              {#if vm.periodError}
                <CalloutWidget items={[{ label: 'Create failed', text: vm.periodError, tone: 'danger' }]} />
              {/if}
              <Row justify="end">
                <Button onclick={() => vm.createPeriod()} disabled={vm.creatingPeriod}>
                  {vm.creatingPeriod ? 'Creating…' : 'Create Period'}
                </Button>
              </Row>
            </Stack>
          </Card>

          <Grid min="300px" gap="lg">
            <Card>
              <Stack gap="sm">
                <Row justify="between" wrap>
                  <span class="pr-section-label">Periods</span>
                  <Button variant="primary" disabled={!vm.selectedPeriodId || vm.generatingRun} onclick={() => vm.generateRun()}>
                    {vm.generatingRun ? 'Generating…' : 'Generate Run'}
                  </Button>
                </Row>
                {#if vm.scopedPeriods.length === 0}
                  <EmptyState message="No payroll periods for this division yet." />
                {:else}
                  <DataTable
                    columns={periodColumns}
                    rows={vm.scopedPeriods}
                    id={(p) => p.id}
                    status={periodStatus}
                    selectedId={vm.selectedPeriodId || null}
                    onSelect={(p) => vm.selectPeriod(p.id)}
                  />
                {/if}
                {#if vm.generateError}
                  <CalloutWidget items={[{ label: 'Generate failed', text: vm.generateError, tone: 'danger' }]} />
                {/if}
              </Stack>
            </Card>

            <Card>
              <Stack gap="sm">
                <span class="pr-section-label">Runs</span>
                {#if vm.displayedRuns.length === 0}
                  <EmptyState message="No payroll runs yet — generate one from a period." />
                {:else}
                  <DataTable
                    columns={runColumns}
                    rows={vm.displayedRuns}
                    id={(r) => r.id}
                    status={RUN_STATUS}
                    selectedId={vm.selectedRunId || null}
                    onSelect={(r) => vm.selectRun(r.id)}
                  />
                {/if}
              </Stack>
            </Card>
          </Grid>

          {#if vm.runDetailError}
            <CalloutWidget items={[{ label: 'Run unavailable', text: vm.runDetailError, tone: 'danger' }]} />
          {/if}

          {#if vm.selectedRun}
            {@const run = vm.selectedRun}
            <Card>
              <Stack gap="lg">
                <Row justify="between" wrap>
                  <Stack gap="xs">
                    <span class="pr-run-title">{run.runNumber}</span>
                    <span class="pr-meta">{run.periodName || 'Payroll run'} · {run.division}</span>
                  </Stack>
                  <Badge tone={RUN_BADGE_TONE[run.status] ?? 'neutral'} label={run.status} />
                </Row>

                <Stepper steps={STEPS} currentKey={run.status} actions={stepperActions} busy={vm.runActionBusy}>
                  {#snippet detail()}
                    <Stack gap="lg">
                      <StatTileGrid
                        sections={[
                          {
                            title: 'Run Totals',
                            items: [
                              { label: 'Employees', value: run.totalEmployees },
                              { label: 'Gross', value: run.grossTotal, content: 'money' },
                              { label: 'Deductions', value: run.deductionsTotal, content: 'money' },
                              { label: 'Net', value: run.netTotal, content: 'money' },
                              { label: 'Employer Cost', value: run.employerCostTotal, content: 'money' },
                            ],
                          },
                        ]}
                      />

                      {#if run.notes}
                        <CalloutWidget items={[{ label: 'Run Notes', text: run.notes, tone: 'neutral' }]} />
                      {/if}

                      <Card padding="none">
                        {#if run.items.length === 0}
                          <EmptyState message="This run has no line items." />
                        {:else}
                          <DataTable columns={itemColumns} rows={run.items} id={(i) => i.id} />
                        {/if}
                      </Card>

                      <Stack gap="sm">
                        <span class="pr-section-label">Approval Reason</span>
                        <input class="k-input" bind:value={vm.approveReason} placeholder="Why is this run being approved?" />
                      </Stack>

                      <Stack gap="sm">
                        <span class="pr-section-label">Payment Details (Mark Paid)</span>
                        <FormGrid columns={3}>
                          <label class="k-field">
                            <span class="k-field-label">Paid Date</span>
                            <input class="k-input" type="date" bind:value={vm.paidAt} />
                          </label>
                          <label class="k-field">
                            <span class="k-field-label">Bank Account</span>
                            <select class="k-input" bind:value={vm.bankAccountId}>
                              <option value="">Select bank account</option>
                              {#each vm.bankAccounts as account (account.id)}
                                <option value={account.id}>{account.bankName} - {account.accountName} ({account.accountNumber || 'no number on file'})</option>
                              {/each}
                            </select>
                          </label>
                          <label class="k-field">
                            <span class="k-field-label">Payment Reference</span>
                            <input class="k-input" bind:value={vm.paymentReference} placeholder="Bank batch / transfer reference" />
                          </label>
                        </FormGrid>
                      </Stack>

                      {#if vm.runActionError}
                        <CalloutWidget items={[{ label: 'Action failed', text: vm.runActionError, tone: 'danger' }]} />
                      {/if}
                    </Stack>
                  {/snippet}
                </Stepper>
              </Stack>
            </Card>
          {:else if !vm.runDetailError}
            <EmptyState message="Select a payroll run to view its approve → post → pay lifecycle." />
          {/if}
        </Stack>
      {:else}
        <Card padding="none">
          {#if vm.displayedPayouts.length === 0}
            <EmptyState message="No payroll payouts recorded for this division yet." />
          {:else}
            <DataTable
              columns={payoutColumns}
              rows={vm.displayedPayouts}
              id={(p) => p.id}
              status={PAYOUT_STATUS}
              selectedId={vm.selectedRunId || null}
              onSelect={(p) => vm.selectRun(p.payrollRunId)}
            />
          {/if}
        </Card>
      {/if}
    </Stack>
  {/if}
</PageShell>

{#if vm.approveConfirmOpen && vm.selectedRun}
  <ConfirmDialog
    title="Approve payroll run?"
    message={`This approves ${vm.selectedRun.runNumber}${vm.approveReason.trim() ? ` — reason: "${vm.approveReason.trim()}"` : ' with no reason entered'}.`}
    confirmLabel="Approve"
    danger={false}
    onConfirm={() => vm.confirmApprove()}
    onCancel={() => vm.cancelApprove()}
  />
{/if}

{#if vm.postConfirmOpen && vm.selectedRun}
  <ConfirmDialog
    title="Post payroll run?"
    message={`This posts ${vm.selectedRun.runNumber} to the general ledger. This cannot be easily undone.`}
    confirmLabel="Post"
    danger={false}
    onConfirm={() => vm.confirmPost()}
    onCancel={() => vm.cancelPost()}
  />
{/if}

<style>
  /* Typography only (L1) — layout/spacing lives in primitives; native form
   * controls use the kernel-owned .k-field/.k-field-label/.k-input classes
   * (single-source in styles/kernel.css), not per-screen CSS. */
  .pr-section-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
  .pr-run-title {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
  .pr-meta {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
</style>
