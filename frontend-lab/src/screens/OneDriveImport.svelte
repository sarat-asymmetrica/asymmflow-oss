<script lang="ts">
  /* OneDrive Import — bespoke-on-primitives, built on the Wizard primitive
   * (kernel/primitives/Wizard.svelte). Three local steps: configure paths ->
   * review scanned deals -> run import. All state/bridge-call logic lives in
   * onedrive-import-vm.svelte.ts (L5); this file only composes primitives and
   * switches the Wizard's `content` snippet on vm.currentIndex (L1) — no raw
   * layout CSS, no bridge calls of its own. */
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import DataTable from '$kernel/primitives/DataTable.svelte'
  import Wizard from '$kernel/primitives/Wizard.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import type { ColumnSpec, StatusSpec } from '$kernel/descriptor'
  import type { CalloutItem } from '$kernel/hub'
  import type { Tone } from '$kernel/tones'
  import type { OneDriveImportResult, ReviewDeal } from '../bridge/onedrive-import'
  import { OneDriveImportViewModel, WIZARD_STEPS } from './onedrive-import-vm.svelte'
  import OneDriveIncludeCell from './OneDriveIncludeCell.svelte'
  import OneDriveCustomerCell from './OneDriveCustomerCell.svelte'

  let { embedded = false }: { embedded?: boolean } = $props()

  const vm = new OneDriveImportViewModel()
  $effect(() => {
    void vm.detectInitialPath()
  })

  const dealColumns: ColumnSpec<ReviewDeal>[] = [
    {
      key: 'include',
      label: 'Include',
      content: 'text',
      value: (r) => (r.selected ? 'Included' : 'Skipped'), // unused by DataTable when `cell` is set; kept truthful
      cell: OneDriveIncludeCell,
      minWidth: 80,
    },
    {
      key: 'folderName',
      label: 'Deal Folder',
      content: 'name',
      value: (r) => r.folderName.trim() || '(unnamed folder)',
      grow: true,
      minWidth: 240,
    },
    { key: 'instrumentType', label: 'Instrument', content: 'text', value: (r) => r.instrumentType || 'Unknown', minWidth: 150 },
    { key: 'yearHint', label: 'Year', content: 'text', value: (r) => r.yearHint || '—', minWidth: 80 },
    { key: 'fileCount', label: 'Files', content: 'quantity', value: (r) => r.files.length, minWidth: 80 },
    {
      key: 'customer',
      label: 'Customer',
      content: 'text',
      value: (r) => r.customerMatches.find((m) => m.customerId === r.confirmedCustomerId)?.businessName ?? 'Unmatched',
      cell: OneDriveCustomerCell,
      minWidth: 280,
    },
  ]

  type ResultRow = OneDriveImportResult & { folderName: string }
  const RESULT_TONES: Record<string, Tone> = { success: 'success', failed: 'danger' }
  const resultStatus: StatusSpec<ResultRow> = {
    value: (r) => (r.success ? 'success' : 'failed'),
    tones: RESULT_TONES,
  }
  const resultColumns: ColumnSpec<ResultRow>[] = [
    { key: 'folderName', label: 'Deal', content: 'name', value: (r) => r.folderName, grow: true, minWidth: 220 },
    { key: 'status', label: 'Result', content: 'status', value: (r) => (r.success ? 'success' : 'failed'), minWidth: 100 },
    { key: 'offerId', label: 'Offer', content: 'code', value: (r) => r.offerId || '—', minWidth: 130 },
    { key: 'costingSheetsImported', label: 'Costing Sheets', content: 'quantity', value: (r) => r.costingSheetsImported, minWidth: 130 },
    { key: 'pdfsQueued', label: 'PDFs Queued', content: 'quantity', value: (r) => r.pdfsQueued, minWidth: 120 },
    { key: 'message', label: 'Message', content: 'text', value: (r) => r.message, grow: true, minWidth: 260 },
  ]

  const scanErrorItems = $derived(
    vm.scanErrors.map((text): CalloutItem => ({ label: 'Scan warning', text, tone: 'warning' })),
  )

  const dealId = (r: ReviewDeal) => r.localId
  const resultId = (r: ResultRow) => r.dealLocalId
</script>

<PageShell {embedded} title="OneDrive Import" subtitle="Import deal folders from locally-synced OneDrive">
  <Wizard
    steps={WIZARD_STEPS}
    currentIndex={vm.currentIndex}
    onBack={() => vm.goBack()}
    onNext={() => vm.goNext()}
    canAdvance={vm.canAdvance}
    nextLabel={vm.nextLabel}
    busy={vm.busy}
  >
    {#snippet content()}
      {#if vm.currentIndex === 0}
        <Stack gap="md">
          <Card padding="md">
            <Stack gap="md">
              <span class="k-field-label">Deal folder paths</span>
              {#each vm.paths as entry, i (i)}
                <Stack gap="xs">
                  <Row gap="sm" align="center">
                    <input
                      class="k-input k-grow"
                      value={entry.value}
                      oninput={(e) => vm.updatePathValue(i, e.currentTarget.value)}
                      placeholder="e.g. C:\Users\you\OneDrive - Company\Deals"
                    />
                    <Button onclick={() => vm.validatePath(i)} disabled={entry.validating || !entry.value.trim()}>
                      {entry.validating ? 'Validating…' : 'Validate'}
                    </Button>
                    {#if vm.paths.length > 1}
                      <Button variant="danger" onclick={() => vm.removePath(i)}>Remove</Button>
                    {/if}
                  </Row>
                  {#if entry.valid === true}
                    <Badge
                      tone="success"
                      label="~{entry.estimatedDeals} deal folder{entry.estimatedDeals === 1 ? '' : 's'} detected"
                    />
                  {:else if entry.valid === false}
                    <Badge tone="danger" label={entry.error ?? 'Invalid path'} />
                  {/if}
                </Stack>
              {/each}
              <Row gap="sm">
                <Button onclick={() => vm.addPath()}>Add another path</Button>
              </Row>
            </Stack>
          </Card>
        </Stack>
      {:else if vm.currentIndex === 1}
        <Stack gap="md">
          {#if vm.scanning}
            <EmptyState message="Scanning your OneDrive paths for deal folders…" />
          {:else if vm.error}
            <EmptyState message="Scan failed: {vm.error}" />
          {:else if vm.deals.length === 0}
            <EmptyState message="No deal folders found in the scanned paths." />
          {:else}
            {#if scanErrorItems.length > 0}
              <CalloutWidget items={scanErrorItems} />
            {/if}
            <p>
              {vm.deals.length} deal folder{vm.deals.length === 1 ? '' : 's'} scanned · {vm.includedDeals.length} ready
              to import.
            </p>
            <Card padding="none">
              <DataTable columns={dealColumns} rows={vm.deals} id={dealId} />
            </Card>
          {/if}
        </Stack>
      {:else}
        <Stack gap="md">
          {#if vm.importing}
            <EmptyState message="Importing confirmed deals…" />
          {:else if vm.error}
            <EmptyState message="Import failed: {vm.error}" />
          {:else if vm.results.length === 0}
            <EmptyState message="No deals were imported." />
          {:else}
            <p>{vm.importedCount} of {vm.results.length} deals imported successfully.</p>
            <Card padding="none">
              <DataTable columns={resultColumns} rows={vm.resultRows} id={resultId} status={resultStatus} />
            </Card>
          {/if}
          <Row gap="sm">
            <Button onclick={() => vm.reset()}>Start Over</Button>
          </Row>
        </Stack>
      {/if}
    {/snippet}
  </Wizard>
</PageShell>
