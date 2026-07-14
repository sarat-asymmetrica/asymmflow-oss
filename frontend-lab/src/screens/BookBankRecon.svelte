<script lang="ts">
  /* Book vs Bank Reconciliation — bespoke-on-primitives (K4). Split view:
   * a list of month-end reconciliations (left) + the selected one's balance
   * comparison (right), via the new BalanceComparisonPanel kernel primitive.
   * Not transaction-level matching — that's the separate
   * BankReconciliationScreen (see screens/parity/BookBankRecon.parity.md).
   * All state/derivation lives in book-bank-recon.svelte.ts (L5); this file
   * only composes primitives and renders (L1). */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import DataTable from '$kernel/primitives/DataTable.svelte'
  import BalanceComparisonPanel from '$kernel/primitives/BalanceComparisonPanel.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import ConfirmDialog from '$kernel/controls/ConfirmDialog.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import type { ColumnSpec, StatusSpec } from '$kernel/descriptor'
  import type { Tone } from '$kernel/tones'
  import type { BookBankReconciliationRow } from '../bridge/book-bank-recon'
  import {
    BookBankReconViewModel,
    comparisonColumns,
    comparisonVariance,
    varianceTone,
  } from './book-bank-recon.svelte'

  const vm = new BookBankReconViewModel()
  onMount(() => void vm.load())

  const STATUS_TONES: Record<string, Tone> = {
    Draft: 'neutral',
    Reconciled: 'success',
    Finalized: 'info',
  }
  const status: StatusSpec<BookBankReconciliationRow> = {
    value: (r) => r.status,
    tones: STATUS_TONES,
  }

  const columns: ColumnSpec<BookBankReconciliationRow>[] = [
    { key: 'period', label: 'Period', content: 'code', value: (r) => r.period, minWidth: 90 },
    {
      key: 'bankAccountName',
      label: 'Account',
      content: 'name',
      value: (r) => r.bankAccountName,
      grow: true,
      minWidth: 180,
    },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 100 },
    {
      key: 'variance',
      label: 'Variance',
      content: 'money',
      value: (r) => comparisonVariance(r).value,
      currency: (r) => r.currency,
      tone: (r) => varianceTone(r),
      minWidth: 140,
    },
  ]

  const id = (r: BookBankReconciliationRow) => r.id
</script>

<PageShell
  title="Book vs Bank Reconciliation"
  subtitle="Month-end statement reconciliation — bank side vs book side, then finalize."
>
  {#if vm.error}
    <EmptyState message="Could not load reconciliations: {vm.error}">
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if vm.loading}
    <EmptyState message="Loading reconciliations…" />
  {:else if vm.rows.length === 0}
    <EmptyState message="No reconciliations recorded yet." />
  {:else}
    <Grid min="360px" gap="lg">
      <Card padding="none">
        <DataTable {columns} rows={vm.rows} {id} {status} selectedId={vm.selectedId} onSelect={(r) => vm.select(r)} />
      </Card>

      <Card>
        {#if !vm.selected}
          <EmptyState message="Select a reconciliation to compare bank and book balances." />
        {:else}
          {@const r = vm.selected}
          <Stack gap="lg">
            <Row justify="between" wrap>
              <Stack gap="xs">
                <span class="bbr-account">{r.bankAccountName || '—'}</span>
                <span class="bbr-meta">{r.period} · {r.bankAccountNumber || 'No account number on file'}</span>
              </Stack>
              <Badge tone={STATUS_TONES[r.status] ?? 'neutral'} label={r.status} />
            </Row>

            <BalanceComparisonPanel
              columns={comparisonColumns(r)}
              variance={comparisonVariance(r)}
              currency={r.currency}
            />

            {#if vm.finalizeError}
              <CalloutWidget items={[{ label: 'Finalize failed', text: vm.finalizeError, tone: 'danger' }]} />
            {/if}

            <Row justify="end">
              {#if r.status === 'Finalized'}
                <span class="bbr-finalized">Finalized {r.finalizedAt} by {r.finalizedBy}</span>
              {:else}
                <Button variant="primary" onclick={() => vm.requestFinalize()} disabled={vm.finalizing}>
                  {vm.finalizing ? 'Finalizing…' : 'Finalize'}
                </Button>
              {/if}
            </Row>
          </Stack>
        {/if}
      </Card>
    </Grid>
  {/if}
</PageShell>

{#if vm.confirmOpen && vm.selected}
  <ConfirmDialog
    title="Finalize reconciliation?"
    message={`This locks ${vm.selected.period} (${vm.selected.bankAccountName || 'this account'}) as reconciled. It cannot be edited afterward.`}
    confirmLabel="Finalize"
    danger={false}
    onConfirm={() => vm.confirmFinalize()}
    onCancel={() => vm.cancelFinalize()}
  />
{/if}

<style>
  /* Typography only (L1) — no layout/spacing rules here, that's Grid/Row/
   * Stack's job. Mirrors Pricing.svelte's p-customer-name / p-section-label. */
  .bbr-account {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
  .bbr-meta {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
  .bbr-finalized {
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
</style>
