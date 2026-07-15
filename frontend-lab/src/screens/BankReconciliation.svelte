<script lang="ts">
  /* Bank Reconciliation — bespoke-on-primitives (K4). Transaction-level
   * statement reconciliation: pick a bank account, import a statement
   * (two-phase preview -> confirm/discard — nothing persists until Confirm),
   * then match each statement LINE to an invoice/payment/expense/payroll
   * payout via the AllocationMatchPanel primitive. NOT BookBankRecon.svelte
   * (that screen compares two month-end running totals); this one works the
   * line level (old screen: BankReconciliationScreen.svelte, 2140 lines —
   * see screens/parity/BankReconciliation.parity.md). All state/derivation
   * lives in bank-reconciliation-vm.svelte.ts (L5); this file only composes
   * primitives and renders (L1). */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Toolbar from '$kernel/primitives/Toolbar.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import DataTable from '$kernel/primitives/DataTable.svelte'
  import Modal from '$kernel/primitives/Modal.svelte'
  import FormGrid from '$kernel/primitives/FormGrid.svelte'
  import AllocationMatchPanel from '$kernel/primitives/AllocationMatchPanel.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import ConfirmDialog from '$kernel/controls/ConfirmDialog.svelte'
  import StatTileGrid from '$kernel/widgets/StatTileGrid.svelte'
  import DonutWidget from '$kernel/widgets/DonutWidget.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import ActivityFeed from '$kernel/widgets/ActivityFeed.svelte'
  import { formatDate, formatMoney } from '$kernel/format'
  import type { ColumnSpec, StatusSpec } from '$kernel/descriptor'
  import type { Tone } from '$kernel/tones'
  import type { BankStatementRow, BankStatementLineRow } from '../bridge/bank-reconciliation'
  import {
    BankReconciliationViewModel,
    STATEMENT_STATUS_TONES,
    transactionTypeTone,
    lineAmount,
  } from './bank-reconciliation-vm.svelte'

  let { embedded = false }: { embedded?: boolean } = $props()

  const vm = new BankReconciliationViewModel()
  onMount(() => void vm.load())

  const statementStatus: StatusSpec<BankStatementRow> = {
    value: (r) => r.status,
    tones: STATEMENT_STATUS_TONES,
  }

  const lineMatchStatus: StatusSpec<BankStatementLineRow> = {
    value: (r) => (r.isMatched ? 'Matched' : 'Unmatched'),
    tones: { Matched: 'success', Unmatched: 'danger' },
  }

  const statementColumns: ColumnSpec<BankStatementRow>[] = [
    { key: 'statementNumber', label: 'Statement #', content: 'code', value: (r) => r.statementNumber, grow: true, minWidth: 170 },
    {
      key: 'period',
      label: 'Period',
      content: 'text',
      value: (r) => `${formatDate(r.periodStart)} – ${formatDate(r.periodEnd)}`,
      minWidth: 190,
    },
    { key: 'closingBalance', label: 'Closing', content: 'money', value: (r) => r.closingBalance, currency: (r) => r.currency, minWidth: 130 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 110 },
  ]

  const lineColumns: ColumnSpec<BankStatementLineRow>[] = [
    { key: 'transactionDate', label: 'Date', content: 'date', value: (r) => r.transactionDate, minWidth: 100 },
    { key: 'description', label: 'Description', content: 'text', value: (r) => r.description, grow: true, minWidth: 220 },
    { key: 'reference', label: 'Reference', content: 'code', value: (r) => r.reference, minWidth: 130 },
    { key: 'extractedCustomer', label: 'Detected', content: 'text', value: (r) => r.extractedCustomer, minWidth: 160 },
    { key: 'debit', label: 'Debit', content: 'money', value: (r) => r.debit, currency: () => vm.selectedStatement?.currency ?? 'BHD', minWidth: 120 },
    { key: 'credit', label: 'Credit', content: 'money', value: (r) => r.credit, currency: () => vm.selectedStatement?.currency ?? 'BHD', minWidth: 120 },
    {
      key: 'transactionType',
      label: 'Type',
      content: 'text',
      value: (r) => r.transactionType.replace(/_/g, ' '),
      tone: (r) => transactionTypeTone(r),
      minWidth: 130,
    },
    { key: 'isMatched', label: 'Match', content: 'status', value: (r) => (r.isMatched ? 'Matched' : 'Unmatched'), minWidth: 110 },
    {
      key: 'matchConfidence',
      label: 'Confidence',
      content: 'text',
      value: (r) => (r.isMatched ? `${Math.round(r.matchConfidence * 100)}%` : '—'),
      minWidth: 100,
    },
  ]

  const previewLineColumns: ColumnSpec<BankStatementLineRow>[] = [
    { key: 'transactionDate', label: 'Date', content: 'date', value: (r) => r.transactionDate, minWidth: 100 },
    { key: 'description', label: 'Description', content: 'text', value: (r) => r.description, grow: true, minWidth: 220 },
    { key: 'reference', label: 'Reference', content: 'code', value: (r) => r.reference, minWidth: 130 },
    { key: 'debit', label: 'Debit', content: 'money', value: (r) => r.debit, minWidth: 120 },
    { key: 'credit', label: 'Credit', content: 'money', value: (r) => r.credit, minWidth: 120 },
  ]

  const statementId = (r: BankStatementRow) => r.id
  const lineId = (r: BankStatementLineRow) => r.id
</script>

<PageShell
  {embedded}
  title="Bank Reconciliation"
  subtitle="Import bank statements and match each transaction line to an invoice, payment, expense, or payroll payout."
>
  {#snippet actions()}
    <Button variant="primary" onclick={() => vm.openImport()}>Import Statement</Button>
  {/snippet}

  {#if !vm.loading && !vm.error && vm.bankAccounts.length > 0}
    {#snippet toolbar()}
      <Toolbar>
        <select
          class="k-input"
          value={vm.selectedAccountId ?? ''}
          onchange={(e) => vm.selectAccount(e.currentTarget.value)}
        >
          {#each vm.bankAccounts as a (a.id)}
            <option value={a.id}>
              {a.bankName ? `${a.bankName} — ` : ''}{a.accountName} ({a.currency}){a.isActive ? '' : ' · inactive'}
            </option>
          {/each}
        </select>
        {#snippet trailing()}
          {#if vm.selectedStatement}
            <Button onclick={() => vm.openEditStatement()}>Edit Statement</Button>
            <Button variant="danger" onclick={() => vm.requestDelete()}>Delete Statement</Button>
            <Button onclick={() => vm.openAddLine()}>Add Transaction</Button>
            <Button onclick={() => vm.autoMatch()} disabled={vm.autoMatching}>
              {vm.autoMatching ? 'Matching…' : 'Auto-Match'}
            </Button>
            <Button
              variant="primary"
              onclick={() => vm.requestFinalize()}
              disabled={vm.totalUnmatched > 0 || vm.finalizing}
            >
              {vm.finalizing ? 'Finalizing…' : 'Finalize'}
            </Button>
            <Button onclick={() => vm.openAuditTrail()}>Audit Trail</Button>
          {/if}
        {/snippet}
      </Toolbar>
    {/snippet}
  {/if}

  {#if vm.loading}
    <EmptyState message="Loading bank accounts…" />
  {:else if vm.error}
    <EmptyState message={`Could not load bank accounts: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if vm.bankAccounts.length === 0}
    <EmptyState message="No bank accounts are set up yet. Add one in Settings → Bank Accounts to begin reconciling statements." />
  {:else}
    <Stack gap="lg">
      <Grid min="220px" gap="md">
        <Card>
          <StatTileGrid
            sections={[
              {
                items: [
                  { label: 'Cash Balance', value: vm.cashPosition?.totalBhd ?? 0, content: 'money' },
                  { label: 'Matched', value: vm.totalMatched, content: 'quantity', tone: 'success' },
                  {
                    label: 'Unmatched',
                    value: vm.totalUnmatched,
                    content: 'quantity',
                    tone: vm.totalUnmatched > 0 ? 'danger' : 'success',
                  },
                  {
                    label: 'Unmatched Credits',
                    value: vm.unmatchedCredit,
                    content: 'money',
                    ...(vm.unmatchedCredit > 0 ? { tone: 'warning' as Tone } : {}),
                  },
                ],
              },
            ]}
          />
        </Card>
        <Card>
          <DonutWidget
            centerLabel="Lines"
            segments={[
              { key: 'matched', label: 'Matched', value: vm.totalMatched, tone: 'success' },
              { key: 'unmatched', label: 'Unmatched', value: vm.totalUnmatched, tone: 'danger' },
            ]}
          />
        </Card>
      </Grid>

      {#if vm.cashPosition && vm.cashPosition.notices.length > 0}
        <CalloutWidget items={vm.cashPosition.notices.map((n) => ({ label: 'Statement check', text: n, tone: 'warning' as Tone }))} />
      {/if}

      {#if vm.selectedStatement && vm.selectedStatement.discrepancyAmount !== 0}
        <CalloutWidget
          items={[
            {
              label: 'Discrepancy',
              text: `${vm.selectedStatement.statementNumber} shows a discrepancy of ${formatMoney(vm.selectedStatement.discrepancyAmount, vm.selectedStatement.currency)} against its declared closing balance.`,
              tone: 'danger',
            },
          ]}
        />
      {/if}

      {#if vm.autoMatchResult}
        <CalloutWidget
          items={[
            {
              label: 'Auto-match complete',
              text: `${vm.autoMatchResult.matchedCount} transaction(s) matched automatically. ${vm.autoMatchResult.unmatchedCount} remain unmatched.`,
              tone: 'info',
            },
          ]}
        />
      {/if}

      {#if vm.actionError}
        <CalloutWidget items={[{ label: 'Action failed', text: vm.actionError, tone: 'danger' }]} />
      {/if}

      {#if vm.deleteError}
        <CalloutWidget items={[{ label: 'Delete failed', text: vm.deleteError, tone: 'danger' }]} />
      {/if}

      {#if vm.finalizeError}
        <CalloutWidget items={[{ label: 'Finalize failed', text: vm.finalizeError, tone: 'danger' }]} />
      {/if}

      <Grid min="380px" gap="lg">
        <Card>
          <Stack gap="sm">
            <span class="br-panel-title">Statements</span>
            {#if vm.statementsLoading}
              <EmptyState message="Loading statements…" />
            {:else if vm.statements.length === 0}
              <EmptyState message="No statements imported for this account yet." />
            {:else}
              <DataTable
                columns={statementColumns}
                rows={vm.statements}
                id={statementId}
                status={statementStatus}
                selectedId={vm.selectedStatementId}
                onSelect={(r) => vm.selectStatement(r)}
              />
            {/if}
          </Stack>
        </Card>

        <Card>
          <Stack gap="sm">
            <span class="br-panel-title">
              {vm.selectedStatement ? `Transactions — ${vm.selectedStatement.statementNumber}` : 'Transactions'}
            </span>
            {#if vm.selectedStatement?.notes}
              <Stack gap="xs">
                <span class="br-notes-label">Statement notes</span>
                <p class="br-notes-body">{vm.selectedStatement.notes}</p>
              </Stack>
            {/if}
            {#if !vm.selectedStatement}
              <EmptyState message="Select a statement to view its transactions." />
            {:else if vm.linesLoading}
              <EmptyState message="Loading transactions…" />
            {:else if vm.lines.length === 0}
              <EmptyState message="No transactions on this statement." />
            {:else}
              <DataTable
                columns={lineColumns}
                rows={vm.lines}
                id={lineId}
                status={lineMatchStatus}
                onSelect={(r) => vm.openMatch(r)}
              />
            {/if}
          </Stack>
        </Card>
      </Grid>
    </Stack>
  {/if}
</PageShell>

<!-- Import Statement modal -->
{#if vm.importOpen}
  <Modal title="Import Bank Statement" onClose={() => vm.closeImport()}>
    <Stack gap="md">
      <label class="k-field">
        <span class="k-field-label">Bank Account</span>
        <select class="k-input" bind:value={vm.importAccountId}>
          <option value="">Select account…</option>
          {#each vm.bankAccounts as a (a.id)}
            <option value={a.id}>{a.bankName ? `${a.bankName} — ` : ''}{a.accountName} ({a.currency})</option>
          {/each}
        </select>
      </label>
      <CalloutWidget
        items={[
          { label: 'Two-step import', text: 'A file dialog opens on Preview. Nothing is saved until you confirm the parsed rows.', tone: 'info' },
        ]}
      />
      {#if vm.importError}
        <CalloutWidget items={[{ label: 'Import failed', text: vm.importError, tone: 'danger' }]} />
      {/if}
    </Stack>
    {#snippet footer()}
      <Button onclick={() => vm.closeImport()}>Cancel</Button>
      <Button variant="primary" onclick={() => vm.runPreview()} disabled={!vm.importAccountId || vm.importLoading}>
        {vm.importLoading ? 'Loading…' : 'Preview'}
      </Button>
    {/snippet}
  </Modal>
{/if}

<!-- Import preview modal (two-phase: nothing persisted until Confirm) -->
{#if vm.importPreview}
  {@const preview = vm.importPreview}
  <Modal title="Review Parsed Statement" onClose={() => vm.cancelImportPreview()}>
    <Stack gap="md">
      <StatTileGrid
        sections={[
          {
            items: [
              { label: 'Statement #', value: preview.statementNumber, content: 'code' },
              { label: 'Period', value: `${formatDate(preview.periodStart)} – ${formatDate(preview.periodEnd)}` },
              { label: 'Opening', value: preview.openingBalance, content: 'money' },
              { label: 'Closing', value: preview.closingBalance, content: 'money' },
              { label: 'Lines Parsed', value: preview.lines.length, content: 'quantity' },
            ],
          },
        ]}
      />
      <CalloutWidget
        items={[
          { label: 'Nothing saved yet', text: 'Review the parsed rows below, then confirm to commit them to the ledger — or cancel to discard.', tone: 'warning' },
        ]}
      />
      <Card padding="none">
        <DataTable columns={previewLineColumns} rows={preview.lines} id={lineId} />
      </Card>
      {#if vm.importError}
        <CalloutWidget items={[{ label: 'Import failed', text: vm.importError, tone: 'danger' }]} />
      {/if}
    </Stack>
    {#snippet footer()}
      <Button onclick={() => vm.cancelImportPreview()}>Cancel Import</Button>
      <Button variant="primary" onclick={() => vm.confirmImport()} disabled={vm.confirmingImport}>
        {vm.confirmingImport ? 'Saving…' : 'Confirm & Save'}
      </Button>
    {/snippet}
  </Modal>
{/if}

<!-- Match modal: AllocationMatchPanel embedded -->
{#if vm.matchOpen && vm.matchingLine}
  {@const line = vm.matchingLine}
  <Modal title="Match Transaction" onClose={() => vm.closeMatch()}>
    <Stack gap="md">
      <Row justify="between" wrap>
        <Stack gap="xs">
          <span class="br-line-desc">{line.description || '(no description)'}</span>
          <span class="br-line-meta">{formatDate(line.transactionDate)} · {line.reference || 'No reference'}</span>
        </Stack>
        <span class="br-line-amount">{formatMoney(lineAmount(line), vm.selectedStatement?.currency ?? 'BHD')}</span>
      </Row>

      {#if line.extractedCustomer}
        <CalloutWidget items={[{ label: 'Detected customer', text: line.extractedCustomer, tone: 'info' }]} />
      {/if}

      {#if line.isMatched}
        <CalloutWidget
          items={[
            {
              label: 'Already matched',
              text: `Matched at ${Math.round(line.matchConfidence * 100)}% confidence (${line.matchedInvoiceIds || 'no reference on file'}). Unmatch first to re-match.`,
              tone: 'neutral',
            },
          ]}
        />
        <Row justify="end">
          <Button variant="danger" onclick={() => vm.unmatch(line)} disabled={vm.unmatching === line.id}>
            {vm.unmatching === line.id ? 'Unmatching…' : 'Unmatch'}
          </Button>
        </Row>
      {:else}
        <AllocationMatchPanel
          target={{ amount: lineAmount(line), currency: vm.selectedStatement?.currency ?? 'BHD', label: 'Bank line amount' }}
          allocations={vm.allocations}
          candidates={vm.matchCandidates}
          candidateTypeOptions={vm.matchCandidateTypeOptions}
          singleSelectTypes={vm.singleSelectTypes}
          loading={vm.candidatesLoading}
          bind:balanced={vm.matchBalanced}
          onAdd={(c, amount) => vm.addAllocation(c, amount)}
          onAmountChange={(key, amount) => vm.changeAllocationAmount(key, amount)}
          onRemove={(key) => vm.removeAllocation(key)}
        />
        {#if vm.matchError}
          <CalloutWidget items={[{ label: 'Match failed', text: vm.matchError, tone: 'danger' }]} />
        {/if}
      {/if}
    </Stack>
    {#snippet footer()}
      <Button onclick={() => vm.closeMatch()}>Close</Button>
      {#if !line.isMatched}
        <Button variant="primary" onclick={() => vm.confirmMatch()} disabled={!vm.matchBalanced || vm.matchSaving}>
          {vm.matchSaving ? 'Matching…' : 'Confirm Match'}
        </Button>
      {/if}
    {/snippet}
  </Modal>
{/if}

<!-- Edit Statement modal -->
{#if vm.editStatementOpen}
  <Modal title="Edit Statement" onClose={() => vm.closeEditStatement()}>
    <Stack gap="md">
      <FormGrid columns={2}>
        <label class="k-field">
          <span class="k-field-label">Period Start</span>
          <input class="k-input" type="date" bind:value={vm.statementDraft.periodStart} />
        </label>
        <label class="k-field">
          <span class="k-field-label">Period End</span>
          <input class="k-input" type="date" bind:value={vm.statementDraft.periodEnd} />
        </label>
        <label class="k-field">
          <span class="k-field-label">Opening Balance</span>
          <input class="k-input" type="number" step="0.001" bind:value={vm.statementDraft.openingBalance} />
        </label>
        <label class="k-field">
          <span class="k-field-label">Closing Balance</span>
          <input class="k-input" type="number" step="0.001" bind:value={vm.statementDraft.closingBalance} />
        </label>
        <label class="k-field">
          <span class="k-field-label">Status</span>
          <select class="k-input" bind:value={vm.statementDraft.status}>
            <option value="Imported">Imported</option>
            <option value="In Progress">In Progress</option>
            <option value="Reconciled">Reconciled</option>
            <option value="Verified">Verified</option>
          </select>
        </label>
        <label class="k-field k-field-wide">
          <span class="k-field-label">Notes</span>
          <textarea class="br-input br-input-area" rows="3" bind:value={vm.statementDraft.notes}></textarea>
        </label>
      </FormGrid>
      {#if vm.statementError}
        <CalloutWidget items={[{ label: 'Save failed', text: vm.statementError, tone: 'danger' }]} />
      {/if}
    </Stack>
    {#snippet footer()}
      <Button onclick={() => vm.closeEditStatement()}>Cancel</Button>
      <Button variant="primary" onclick={() => vm.saveStatement()} disabled={vm.savingStatement}>
        {vm.savingStatement ? 'Saving…' : 'Save Changes'}
      </Button>
    {/snippet}
  </Modal>
{/if}

<!-- Add/Edit Transaction Line modal -->
{#if vm.lineModalOpen}
  <Modal title={vm.editingLineId ? 'Edit Transaction' : 'Add Transaction'} onClose={() => vm.closeLineModal()}>
    <Stack gap="md">
      <FormGrid columns={2}>
        <label class="k-field k-field-wide">
          <span class="k-field-label">Transaction Date</span>
          <input class="k-input" type="date" bind:value={vm.lineDraft.transactionDate} />
        </label>
        <label class="k-field k-field-wide">
          <span class="k-field-label">Description</span>
          <input class="k-input" type="text" placeholder="Transaction description" bind:value={vm.lineDraft.description} />
        </label>
        <label class="k-field k-field-wide">
          <span class="k-field-label">Reference</span>
          <input class="k-input" type="text" placeholder="Reference number (optional)" bind:value={vm.lineDraft.reference} />
        </label>
        <label class="k-field">
          <span class="k-field-label">Debit</span>
          <input class="k-input" type="number" step="0.001" bind:value={vm.lineDraft.debit} />
        </label>
        <label class="k-field">
          <span class="k-field-label">Credit</span>
          <input class="k-input" type="number" step="0.001" bind:value={vm.lineDraft.credit} />
        </label>
      </FormGrid>
      {#if vm.editingLineWasMatched}
        <CalloutWidget
          items={[
            {
              label: 'Match will be cleared',
              text: 'This transaction is currently matched. Saving a correction clears the match so it re-enters the review queue.',
              tone: 'warning',
            },
          ]}
        />
      {/if}
      {#if vm.lineError}
        <CalloutWidget items={[{ label: 'Save failed', text: vm.lineError, tone: 'danger' }]} />
      {/if}
    </Stack>
    {#snippet footer()}
      {#if vm.editingLineId}
        <Button variant="danger" onclick={() => vm.deleteLine()} disabled={vm.savingLine}>Delete Transaction</Button>
      {/if}
      <Button onclick={() => vm.closeLineModal()}>Cancel</Button>
      <Button variant="primary" onclick={() => vm.saveLine()} disabled={vm.savingLine}>
        {vm.savingLine ? 'Saving…' : vm.editingLineId ? 'Save Changes' : 'Add Transaction'}
      </Button>
    {/snippet}
  </Modal>
{/if}

<!-- Audit trail drawer (real GetAuditTrail binding) -->
{#if vm.auditOpen}
  <Modal title="Audit Trail" onClose={() => vm.closeAuditTrail()}>
    {#if vm.auditLoading}
      <EmptyState message="Loading audit trail…" />
    {:else if vm.auditError}
      <CalloutWidget items={[{ label: 'Could not load audit trail', text: vm.auditError, tone: 'danger' }]} />
    {:else}
      <ActivityFeed
        items={vm.auditEntries.map((e) => ({
          title: `${e.action}${e.isReversed ? ' (reversed)' : ''}`,
          subtitle: e.detail || e.actor || undefined,
          timestamp: e.timestamp,
          tone: (e.isReversed ? 'warning' : e.action === 'FINALIZE' ? 'success' : 'neutral') as Tone,
        }))}
        emptyMessage="No audit entries for this statement."
      />
    {/if}
    {#snippet footer()}
      <Button onclick={() => vm.closeAuditTrail()}>Close</Button>
    {/snippet}
  </Modal>
{/if}

{#if vm.finalizeConfirmOpen && vm.selectedStatement}
  <ConfirmDialog
    title="Finalize reconciliation?"
    message={`This locks ${vm.selectedStatement.statementNumber} as reconciled. It cannot be edited afterward.`}
    confirmLabel="Finalize"
    danger={false}
    onConfirm={() => vm.confirmFinalize()}
    onCancel={() => vm.cancelFinalize()}
  />
{/if}

{#if vm.deleteConfirmOpen && vm.selectedStatement}
  <ConfirmDialog
    title="Delete statement?"
    message={`This permanently deletes ${vm.selectedStatement.statementNumber} and its ${vm.lines.length} transaction line(s). This cannot be undone.`}
    confirmLabel="Delete"
    onConfirm={() => vm.confirmDelete()}
    onCancel={() => vm.cancelDelete()}
  />
{/if}

<style>
  /* Typography only (L1) — native form controls use the kernel-owned
   * k-field/k-field-label/k-input classes (single-source in styles/kernel.css);
   * no form-control layout CSS lives here. */
  .br-panel-title {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
  .br-notes-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
  .br-notes-body {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
  .br-line-desc {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
  .br-line-meta {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
  .br-line-amount {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(18px * var(--ui-font-scale));
    font-weight: 700;
    white-space: nowrap;
  }
</style>
