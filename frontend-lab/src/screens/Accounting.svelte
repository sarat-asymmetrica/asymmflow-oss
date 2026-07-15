<script lang="ts">
  /* Accounting — bespoke-on-primitives (K5), driven by a horizontal
   * ViewSwitcher (Overview / Chart of Accounts / Journal Entries / Reports),
   * mirroring the old screen's 4-tab console but on kernel primitives. All
   * state/derivation lives in accounting-vm.svelte.ts (L5); this file only
   * composes primitives and renders (L1). The voucher's debit/credit lines
   * use the shared LineItemsEditor widget (K5's new line-repeater primitive);
   * the old screen's client-side VAT heuristic is NOT ported — see
   * bridge/accounting.ts and screens/parity/Accounting.parity.md. */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import ViewSwitcher from '$kernel/primitives/ViewSwitcher.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Modal from '$kernel/primitives/Modal.svelte'
  import FormGrid from '$kernel/primitives/FormGrid.svelte'
  import DataTable from '$kernel/primitives/DataTable.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import FilterChips from '$kernel/controls/FilterChips.svelte'
  import StatTileGrid from '$kernel/widgets/StatTileGrid.svelte'
  import DistributionWidget from '$kernel/widgets/DistributionWidget.svelte'
  import DonutWidget from '$kernel/widgets/DonutWidget.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import ListWidget from '$kernel/widgets/ListWidget.svelte'
  import LineItemsEditor from '$kernel/widgets/LineItemsEditor.svelte'
  import type { ColumnSpec, StatusSpec } from '$kernel/descriptor'
  import type { Tone } from '$kernel/tones'
  import { sumField, isDebitCreditBalanced, type LineColumn, type LineFooterCell, type LineBalanceCheck } from '$kernel/line-items'
  import { formatMoney } from '$kernel/format'
  import type { ChartOfAccountRow, JournalEntryRow } from '../bridge/accounting'
  import {
    AccountingViewModel,
    KNOWN_ACCOUNT_TYPES,
    accountTypeTone,
    balanceTone,
    type ProposalReviewRow,
    type VoucherLine,
  } from './accounting-vm.svelte'

  const vm = new AccountingViewModel()
  onMount(() => void vm.load())

  const VIEWS = [
    { key: 'overview', label: 'Overview' },
    { key: 'coa', label: 'Chart of Accounts' },
    { key: 'journal', label: 'Journal Entries' },
    { key: 'reports', label: 'Reports' },
  ]

  const ACCOUNT_TYPE_TONES: Record<string, Tone> = Object.fromEntries(KNOWN_ACCOUNT_TYPES.map((t) => [t, accountTypeTone(t)]))
  const coaStatus: StatusSpec<ChartOfAccountRow> = { value: (r) => r.accountType, tones: ACCOUNT_TYPE_TONES }
  const coaColumns: ColumnSpec<ChartOfAccountRow>[] = [
    { key: 'accountCode', label: 'Code', content: 'code', value: (r) => r.accountCode, minWidth: 90 },
    { key: 'accountName', label: 'Name', content: 'name', value: (r) => r.accountName, grow: true, minWidth: 220 },
    { key: 'accountType', label: 'Type', content: 'status', value: (r) => r.accountType, minWidth: 120 },
    { key: 'balance', label: 'Balance', content: 'money', value: (r) => r.balance, tone: (r) => balanceTone(r.balance), minWidth: 160 },
    { key: 'isActive', label: 'Status', content: 'text', value: (r) => (r.isActive ? 'Active' : 'Inactive'), minWidth: 90 },
  ]

  const JOURNAL_STATUS_TONES: Record<string, Tone> = { Posted: 'success', Draft: 'neutral' }
  const journalStatus: StatusSpec<JournalEntryRow> = { value: (r) => (r.isPosted ? 'Posted' : 'Draft'), tones: JOURNAL_STATUS_TONES }
  const journalColumns: ColumnSpec<JournalEntryRow>[] = [
    { key: 'entryNumber', label: 'Entry #', content: 'code', value: (r) => r.entryNumber, minWidth: 150 },
    { key: 'entryDate', label: 'Date', content: 'date', value: (r) => r.entryDate, minWidth: 110 },
    { key: 'description', label: 'Description', content: 'text', value: (r) => r.description, grow: true, minWidth: 220 },
    { key: 'debitTotal', label: 'Debit', content: 'money', value: (r) => r.debitTotal, minWidth: 130 },
    {
      key: 'creditTotal',
      label: 'Credit',
      content: 'money',
      value: (r) => r.creditTotal,
      tone: (r) => (Math.abs(r.debitTotal - r.creditTotal) > 0.001 ? 'danger' : 'neutral'),
      minWidth: 130,
    },
    { key: 'isPosted', label: 'Status', content: 'status', value: (r) => (r.isPosted ? 'Posted' : 'Draft'), minWidth: 100 },
  ]

  const voucherColumns: LineColumn<VoucherLine>[] = [
    {
      key: 'account',
      label: 'Account',
      kind: 'select',
      minWidth: 240,
      grow: true,
      value: (l) => l.accountId,
      set: (l, v) => {
        l.accountId = String(v)
      },
      options: () => vm.activeAccounts.map((a) => ({ value: a.id, label: `${a.accountCode} — ${a.accountName || 'Unnamed account'}` })),
    },
    {
      key: 'description',
      label: 'Description',
      kind: 'text',
      minWidth: 200,
      value: (l) => l.description,
      set: (l, v) => {
        l.description = String(v)
      },
    },
    {
      key: 'debit',
      label: 'Debit',
      kind: 'money',
      step: '0.001',
      minWidth: 130,
      value: (l) => l.debit,
      set: (l, v) => {
        l.debit = Number(v) || 0
      },
    },
    {
      key: 'credit',
      label: 'Credit',
      kind: 'money',
      step: '0.001',
      minWidth: 130,
      value: (l) => l.credit,
      set: (l, v) => {
        l.credit = Number(v) || 0
      },
    },
  ]
  const voucherFooter: LineFooterCell<VoucherLine>[] = [
    { label: 'Total Debit', value: (rows) => sumField(rows, (r) => r.debit), currency: 'BHD' },
    { label: 'Total Credit', value: (rows) => sumField(rows, (r) => r.credit), currency: 'BHD' },
  ]
  const voucherBalance: LineBalanceCheck<VoucherLine> = {
    isBalanced: (rows) => isDebitCreditBalanced(rows, (r) => r.debit, (r) => r.credit),
  }

  const PRIORITY_TONE: Record<string, Tone> = { low: 'neutral', medium: 'warning', high: 'danger' }
  const priorityTone = (p: string): Tone => PRIORITY_TONE[p] ?? 'neutral'
  const REVIEW_STATUS_TONE: Record<string, Tone> = { approved: 'success', rejected: 'danger', needs_input: 'warning', pending: 'info' }
  const reviewStatusTone = (s: string): Tone => REVIEW_STATUS_TONE[s] ?? 'neutral'
  const proposalRowId = (r: ProposalReviewRow) => r.key
</script>

<PageShell title="Accounting" subtitle="General ledger, chart of accounts, journal entries and reports.">
  {#snippet toolbar()}
    <ViewSwitcher views={VIEWS} activeKey={vm.activeView} onSelect={(k) => vm.selectView(k)} />
  {/snippet}

  {#if vm.error}
    <EmptyState message={`Could not load accounting data: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if vm.loading}
    <EmptyState message="Loading accounting data…" />
  {:else if vm.activeView === 'overview'}
    <Stack gap="lg">
      <Card>
        <Stack gap="md">
          <span class="acc-section-label">Balance sheet position</span>
          <StatTileGrid sections={vm.overviewStatSections} />
        </Stack>
      </Card>

      <Grid min="360px" gap="lg">
        <Card>
          <Stack gap="md">
            <span class="acc-section-label">Asset / Liability / Equity split</span>
            <DonutWidget segments={vm.compositionSegments} centerLabel="Position" />
          </Stack>
        </Card>
        <Card>
          <Stack gap="md">
            <span class="acc-section-label">Posting coverage by source</span>
            {#if vm.coverage}
              <DistributionWidget segments={vm.coverageSegments} />
              <span class="acc-meta"
                >{vm.coverage.linked}/{vm.coverage.total} linked · {vm.coverage.missing} missing · {vm.coverage.isComplete
                  ? 'Complete'
                  : 'Incomplete'}</span
              >
            {/if}
          </Stack>
        </Card>
      </Grid>

      <Card>
        <Stack gap="sm">
          <span class="acc-section-label">Trial balance gate</span>
          <CalloutWidget items={vm.trialBalanceCallout} />
        </Stack>
      </Card>

      <Card>
        <Stack gap="md">
          <Row justify="between" wrap>
            <span class="acc-section-label">Cashflow evidence — command center</span>
            <Button onclick={() => vm.runExport('evidence-pack')} disabled={vm.exportLoading === 'evidence-pack'}>
              {vm.exportLoading === 'evidence-pack' ? 'Exporting…' : 'Export Evidence Pack'}
            </Button>
          </Row>
          <StatTileGrid sections={vm.cashflowKpiSections} />

          <Grid min="320px" gap="md">
            <Stack gap="sm">
              <span class="acc-subsection-label">Evidence sources</span>
              <ListWidget rows={vm.evidenceSourceRows} />
            </Stack>
            <Stack gap="sm">
              <span class="acc-subsection-label">Action proposals — human review log</span>
              {#if vm.reviewError}
                <CalloutWidget items={[{ label: 'Review failed', text: vm.reviewError, tone: 'danger' }]} />
              {/if}
              <Stack gap="sm">
                {#each vm.actionProposalRows as row (proposalRowId(row))}
                  <Card padding="md">
                    <Stack gap="sm">
                      <Row justify="between" wrap>
                        <Stack gap="xs">
                          <span class="acc-proposal-label">{row.proposal.label}</span>
                          <span class="acc-proposal-reason">{row.proposal.reason}</span>
                        </Stack>
                        <Row gap="xs" wrap>
                          <Badge tone={priorityTone(row.proposal.priority)} label={row.proposal.priority} />
                          {#if row.reviewStatus}
                            <Badge tone={reviewStatusTone(row.reviewStatus)} label={row.reviewStatus.replace(/_/g, ' ')} />
                          {/if}
                        </Row>
                      </Row>
                      <Row gap="xs" wrap>
                        <Button
                          onclick={() => vm.reviewProposal(row.proposal, 'approved')}
                          disabled={vm.reviewingKey === row.key}>Approve</Button
                        >
                        <Button
                          onclick={() => vm.reviewProposal(row.proposal, 'needs_input')}
                          disabled={vm.reviewingKey === row.key}>Needs Input</Button
                        >
                        <Button
                          variant="danger"
                          onclick={() => vm.reviewProposal(row.proposal, 'rejected')}
                          disabled={vm.reviewingKey === row.key}>Reject</Button
                        >
                      </Row>
                    </Stack>
                  </Card>
                {/each}
              </Stack>
            </Stack>
          </Grid>
        </Stack>
      </Card>
    </Stack>
  {:else if vm.activeView === 'coa'}
    <Stack gap="md">
      <Row justify="between" wrap gap="md">
        <FilterChips label="Account type" options={vm.accountTypeOptions} bind:selected={vm.coaTypeFilter} />
        <Button variant="primary" onclick={() => vm.openCreateAccount()}>+ New Account</Button>
      </Row>
      {#if vm.filteredAccounts.length === 0}
        <EmptyState message="No accounts match this filter." />
      {:else}
        <Card padding="none">
          <DataTable columns={coaColumns} rows={vm.filteredAccounts} id={(r) => r.id} status={coaStatus} onSelect={(r) => vm.openEditAccount(r)} />
        </Card>
      {/if}
    </Stack>
  {:else if vm.activeView === 'journal'}
    <Stack gap="md">
      <Row justify="end" wrap gap="sm">
        <Button onclick={() => vm.runExport('journal-csv')} disabled={vm.exportLoading === 'journal-csv'}>
          {vm.exportLoading === 'journal-csv' ? 'Exporting…' : 'Export CSV'}
        </Button>
        <Button variant="primary" onclick={() => vm.openVoucher()}>+ New Voucher</Button>
      </Row>
      {#if vm.exportError}
        <CalloutWidget items={[{ label: 'Export failed', text: vm.exportError, tone: 'danger' }]} />
      {:else if vm.exportMessage}
        <CalloutWidget items={[{ label: 'Export ready', text: vm.exportMessage, tone: 'success' }]} />
      {/if}
      {#if vm.journalEntries.length === 0}
        <EmptyState message="No journal entries for this fiscal year yet." />
      {:else}
        <Card padding="none">
          <DataTable columns={journalColumns} rows={vm.journalEntries} id={(r) => r.id} status={journalStatus} />
        </Card>
      {/if}
    </Stack>
  {:else}
    <Stack gap="lg">
      <Card>
        <Row gap="md" align="end" wrap>
          <label class="k-field">
            <span class="k-field-label">Fiscal Year</span>
            <input class="k-input" type="number" min="2000" max="2100" bind:value={vm.reportYear} />
          </label>
          {#if vm.reportError}
            <span class="acc-error">{vm.reportError}</span>
          {/if}
        </Row>
      </Card>

      <Grid min="300px" gap="lg">
        <Card>
          <Stack gap="sm">
            <span class="acc-section-label">Profit &amp; Loss</span>
            <Button variant="primary" onclick={() => vm.generatePLReport()} disabled={vm.reportLoading === 'pl'}>
              {vm.reportLoading === 'pl' ? 'Generating…' : 'Generate'}
            </Button>
            {#if vm.plReport}
              {@const pl = vm.plReport}
              <Stack gap="xs">
                <Row justify="between"><span class="acc-row-label">Sales Revenue</span><span class="acc-num">{formatMoney(pl.salesRevenue, pl.currency)}</span></Row>
                <Row justify="between"><span class="acc-row-label">Other Income</span><span class="acc-num">{formatMoney(pl.otherIncome, pl.currency)}</span></Row>
                <Row justify="between"
                  ><span class="acc-row-label acc-total">Total Revenue</span><span class="acc-num acc-total"
                    >{formatMoney(pl.totalRevenue, pl.currency)}</span
                  ></Row
                >
                <Row justify="between"><span class="acc-row-label">Cost of Goods Sold</span><span class="acc-num">{formatMoney(pl.costOfGoodsSold, pl.currency)}</span></Row>
                <Row justify="between"
                  ><span class="acc-row-label acc-total">Gross Profit</span><span class="acc-num acc-total"
                    >{formatMoney(pl.grossProfit, pl.currency)} ({Math.round(pl.grossProfitMargin * 100)}%)</span
                  ></Row
                >
                <Row justify="between"><span class="acc-row-label">Operating Expenses</span><span class="acc-num">{formatMoney(pl.operatingExpenses, pl.currency)}</span></Row>
                <Row justify="between">
                  <span class="acc-row-label acc-total">Net Profit</span>
                  <span class="acc-num acc-total" style:color={pl.netProfit < 0 ? 'var(--k-tone-danger-fg)' : undefined}>
                    {formatMoney(pl.netProfit, pl.currency)} ({Math.round(pl.netProfitMargin * 100)}%)
                  </span>
                </Row>
              </Stack>
            {/if}
          </Stack>
        </Card>

        <Card>
          <Stack gap="sm">
            <span class="acc-section-label">Balance Sheet</span>
            <Row gap="xs" wrap>
              <Button variant="primary" onclick={() => vm.generateBSReport()} disabled={vm.reportLoading === 'balance'}>
                {vm.reportLoading === 'balance' ? 'Generating…' : 'Generate'}
              </Button>
              <Button onclick={() => vm.runExport('balance-csv')} disabled={vm.exportLoading === 'balance-csv'}>
                {vm.exportLoading === 'balance-csv' ? 'Exporting…' : 'Export CSV'}
              </Button>
            </Row>
            {#if vm.bsReport}
              {@const bs = vm.bsReport}
              <Stack gap="xs">
                <Row justify="between"><span class="acc-row-label">Cash</span><span class="acc-num">{formatMoney(bs.cash, bs.currency)}</span></Row>
                <Row justify="between"><span class="acc-row-label">Accounts Receivable</span><span class="acc-num">{formatMoney(bs.accountsReceivable, bs.currency)}</span></Row>
                <Row justify="between"><span class="acc-row-label">Inventory</span><span class="acc-num">{formatMoney(bs.inventory, bs.currency)}</span></Row>
                <Row justify="between"
                  ><span class="acc-row-label acc-total">Total Assets</span><span class="acc-num acc-total">{formatMoney(bs.totalAssets, bs.currency)}</span></Row
                >
                <Row justify="between"><span class="acc-row-label">Accounts Payable</span><span class="acc-num">{formatMoney(bs.accountsPayable, bs.currency)}</span></Row>
                <Row justify="between"
                  ><span class="acc-row-label acc-total">Total Liabilities</span><span class="acc-num acc-total"
                    >{formatMoney(bs.totalLiabilities, bs.currency)}</span
                  ></Row
                >
                <Row justify="between"><span class="acc-row-label">Retained Earnings</span><span class="acc-num">{formatMoney(bs.retainedEarnings, bs.currency)}</span></Row>
                <Row justify="between"
                  ><span class="acc-row-label acc-total">Total Equity</span><span class="acc-num acc-total">{formatMoney(bs.totalEquity, bs.currency)}</span></Row
                >
              </Stack>
            {/if}
          </Stack>
        </Card>

        <Card>
          <Stack gap="sm">
            <span class="acc-section-label">General Ledger</span>
            <Button onclick={() => vm.runExport('gl-csv')} disabled={vm.exportLoading === 'gl-csv'}>
              {vm.exportLoading === 'gl-csv' ? 'Exporting…' : 'Export CSV'}
            </Button>
          </Stack>
        </Card>

        <Card>
          <Stack gap="sm">
            <span class="acc-section-label">VAT Return</span>
            <Button onclick={() => vm.runExport('vat')} disabled={vm.exportLoading === 'vat'}>
              {vm.exportLoading === 'vat' ? 'Exporting…' : 'Generate'}
            </Button>
          </Stack>
        </Card>
      </Grid>

      {#if vm.exportError}
        <CalloutWidget items={[{ label: 'Export failed', text: vm.exportError, tone: 'danger' }]} />
      {:else if vm.exportMessage}
        <CalloutWidget items={[{ label: 'Export ready', text: vm.exportMessage, tone: 'success' }]} />
      {/if}
    </Stack>
  {/if}
</PageShell>

{#if vm.voucherOpen}
  <Modal title="New Journal Voucher" onClose={() => vm.closeVoucher()}>
    <Stack gap="md">
      <FormGrid columns={2}>
        <label class="k-field">
          <span class="k-field-label">Date</span>
          <input class="k-input" type="date" bind:value={vm.voucherDate} />
        </label>
        <label class="k-field">
          <span class="k-field-label">Description</span>
          <input class="k-input" type="text" placeholder="Voucher narration" bind:value={vm.voucherDescription} />
        </label>
      </FormGrid>

      <LineItemsEditor
        columns={voucherColumns}
        rows={vm.voucherLines}
        createRow={() => vm.blankVoucherLine()}
        onAdd={() => vm.voucherLines.push(vm.blankVoucherLine())}
        onRemove={(i) => vm.voucherLines.splice(i, 1)}
        minRows={2}
        maxRows={100}
        footer={voucherFooter}
        balance={voucherBalance}
        disabled={vm.voucherSaving}
      />

      {#if vm.voucherError}
        <p class="acc-error">Could not save: {vm.voucherError}</p>
      {/if}
    </Stack>

    {#snippet footer()}
      <Button onclick={() => vm.closeVoucher()} disabled={vm.voucherSaving}>Cancel</Button>
      <Button variant="primary" onclick={() => vm.submitVoucher()} disabled={vm.voucherSaving}>
        {vm.voucherSaving ? 'Saving…' : 'Post Voucher'}
      </Button>
    {/snippet}
  </Modal>
{/if}

{#if vm.accountFormOpen && vm.accountDraft}
  {@const draft = vm.accountDraft}
  <Modal title={vm.accountFormMode === 'create' ? 'New Account' : 'Edit Account'} onClose={() => vm.closeAccountForm()}>
    <FormGrid columns={2}>
      <label class="k-field">
        <span class="k-field-label">Account Code</span>
        <input class="k-input" type="text" bind:value={draft.accountCode} />
      </label>
      <label class="k-field">
        <span class="k-field-label">Account Name</span>
        <input class="k-input" type="text" bind:value={draft.accountName} />
      </label>
      <label class="k-field">
        <span class="k-field-label">Type</span>
        <select class="k-input" bind:value={draft.accountType}>
          {#each KNOWN_ACCOUNT_TYPES as t (t)}
            <option value={t}>{t}</option>
          {/each}
        </select>
      </label>
      <label class="k-field">
        <span class="k-field-label">Status</span>
        <select class="k-input" bind:value={draft.isActive}>
          <option value={true}>Active</option>
          <option value={false}>Inactive</option>
        </select>
      </label>
      <label class="k-field k-field-row">
        <input type="checkbox" checked={draft.isVatAccount} onchange={(e) => vm.toggleAccountDraftVat(e.currentTarget.checked)} />
        <span>VAT account</span>
      </label>
      {#if draft.isVatAccount}
        <label class="k-field">
          <span class="k-field-label">VAT Direction</span>
          <select class="k-input" bind:value={draft.vatDirection}>
            <option value="input">Input</option>
            <option value="output">Output</option>
          </select>
        </label>
      {/if}
    </FormGrid>

    {#if vm.accountError}
      <p class="acc-error">Could not save: {vm.accountError}</p>
    {/if}

    {#snippet footer()}
      <Button onclick={() => vm.closeAccountForm()} disabled={vm.accountSaving}>Cancel</Button>
      <Button variant="primary" onclick={() => vm.saveAccount()} disabled={vm.accountSaving}>
        {vm.accountSaving ? 'Saving…' : 'Save'}
      </Button>
    {/snippet}
  </Modal>
{/if}

<style>
  /* Typography/control skin only (L1) — the acc-field/acc-label/acc-input
   * trio mirrors FormModal's k-field/k-input and BusinessSettings' bs-field
   * look (no dedicated kernel TextInput/NumberInput primitive exists yet). */
  .acc-section-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
  .acc-subsection-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.03em;
    color: var(--text-muted);
  }
  .acc-meta {
    font-size: var(--meta-size);
    color: var(--text-secondary);
  }
  .acc-proposal-label {
    font-family: var(--font-display);
    font-size: calc(14px * var(--ui-font-scale));
    font-weight: 600;
    color: var(--text-primary);
    overflow-wrap: break-word;
  }
  .acc-proposal-reason {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
  .acc-row-label {
    font-size: var(--modal-body-size);
    color: var(--text-secondary);
  }
  .acc-num {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: var(--modal-body-size);
    color: var(--text-primary);
  }
  .acc-total {
    font-weight: 700;
    color: var(--text-primary);
  }
  .acc-error {
    font-size: var(--modal-body-size);
    color: var(--k-tone-danger-fg);
    overflow-wrap: break-word;
  }
</style>
