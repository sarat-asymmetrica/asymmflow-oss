<script lang="ts">
  /* Serial Trace — bespoke-on-primitives (K4). Read-only: search a serial
   * number and trace PO -> GRN -> Delivery Note -> Invoice -> Customer,
   * with warranty-date coloring. State/search logic lives in
   * serial-trace.svelte.ts (L5); this file only composes primitives and
   * renders (L1) — no raw layout CSS, no fetch/mutation calls of its own. */
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Toolbar from '$kernel/primitives/Toolbar.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import DataTable from '$kernel/primitives/DataTable.svelte'
  import SearchInput from '$kernel/controls/SearchInput.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import type { ColumnSpec, StatusSpec } from '$kernel/descriptor'
  import type { Tone } from '$kernel/tones'
  import type { SerialTraceRow } from '../bridge/serial-trace'
  import { SerialTraceViewModel, warrantyLabel } from './serial-trace.svelte'
  import SerialWarrantyBadge from './SerialWarrantyBadge.svelte'

  const vm = new SerialTraceViewModel()
  $effect(() => {
    void vm.loadRecent()
  })

  // Lifecycle stage badge — a finite vocabulary, unknown stages render
  // neutral rather than crash (same contract every StatusSpec makes).
  const STAGE_TONES: Record<string, Tone> = {
    Available: 'success',
    Delivered: 'success',
    Reserved: 'warning',
    Shipped: 'info',
    Returned: 'danger',
  }
  const stageStatus: StatusSpec<SerialTraceRow> = {
    value: (r) => r.status,
    tones: STAGE_TONES,
  }

  const columns: ColumnSpec<SerialTraceRow>[] = [
    { key: 'serialNo', label: 'Serial #', content: 'code', value: (r) => r.serialNo, minWidth: 170 },
    { key: 'productCode', label: 'Product', content: 'code', value: (r) => r.productCode, minWidth: 130 },
    { key: 'status', label: 'Stage', content: 'status', value: (r) => r.status, minWidth: 100 },
    { key: 'customerName', label: 'Customer', content: 'name', value: (r) => r.customerName, grow: true, minWidth: 220 },
    {
      key: 'warranty',
      label: 'Warranty',
      content: 'text',
      value: (r) => warrantyLabel(r), // unused by DataTable when `cell` is set; kept for a truthful ColumnSpec
      cell: SerialWarrantyBadge,
      minWidth: 170,
    },
    { key: 'poNumber', label: 'PO', content: 'code', value: (r) => r.poNumber, minWidth: 120 },
    { key: 'grnNumber', label: 'GRN', content: 'code', value: (r) => r.grnNumber, minWidth: 120 },
    { key: 'dnNumber', label: 'Delivery Note', content: 'code', value: (r) => r.dnNumber, minWidth: 130 },
    { key: 'invoiceNumber', label: 'Invoice', content: 'code', value: (r) => r.invoiceNumber, minWidth: 120 },
  ]

  const id = (r: SerialTraceRow) => r.id

  const subtitle = $derived(
    vm.hasSearched
      ? `${vm.rows.length} serial${vm.rows.length === 1 ? '' : 's'} found`
      : 'Recently delivered · PO → GRN → Delivery Note → Invoice → Customer',
  )
</script>

<PageShell title="Serial Trace" {subtitle}>
  {#snippet toolbar()}
    <Toolbar>
      <SearchInput
        bind:value={vm.query}
        placeholder="Search by serial number, product code, GRN, PO, invoice or customer…"
      />
      <Button variant="primary" onclick={() => vm.search()} disabled={vm.searching}>
        {vm.searching ? 'Searching…' : 'Search'}
      </Button>
      {#if vm.hasSearched}
        <Button onclick={() => vm.reset()}>Clear</Button>
      {/if}
    </Toolbar>
  {/snippet}

  {#if vm.error}
    <EmptyState message="Could not load serials: {vm.error}" />
  {:else if vm.hasSearched && vm.searching}
    <EmptyState message="Searching…" />
  {:else if !vm.hasSearched && vm.loadingRecent}
    <EmptyState message="Loading recently delivered serials…" />
  {:else if vm.rows.length === 0}
    <EmptyState
      message={vm.hasSearched
        ? 'No serials matched your search — try a different term.'
        : 'No deliveries recorded yet.'}
    />
  {:else}
    <Card padding="none">
      <DataTable {columns} rows={vm.rows} {id} status={stageStatus} />
    </Card>
  {/if}
</PageShell>
