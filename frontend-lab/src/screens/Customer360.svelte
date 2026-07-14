<script lang="ts">
  /* Customer 360 — bespoke-on-primitives (K4). A rich single-customer detail
   * view: no master list behind it, so a small synthetic picker (FilterChips)
   * stands in for "which customer am I looking at." Left: info panel with a
   * payment-regime badge + headline financial stats. Right: a tabbed panel
   * (Overview / Predictions / Connections) built as a plain button row, not
   * an ejected tab primitive — three fixed, mutually-exclusive tabs, no
   * dynamic tab set. State/load/tab logic lives in customer-360.svelte.ts
   * (L5); this file only composes primitives and renders (L1) — no raw
   * layout CSS, no fetch/mutation calls of its own. See
   * screens/parity/Customer360.parity.md. */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Toolbar from '$kernel/primitives/Toolbar.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import FilterChips from '$kernel/controls/FilterChips.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import StatTileGrid from '$kernel/widgets/StatTileGrid.svelte'
  import ListWidget from '$kernel/widgets/ListWidget.svelte'
  import type { ListRow } from '$kernel/hub'
  import type { Tone } from '$kernel/tones'
  import { formatDate } from '$kernel/format'
  import { Customer360ViewModel, type Customer360Tab } from './customer-360.svelte'
  import type { GradePrediction } from '../bridge/customer-360'

  const vm = new Customer360ViewModel()
  onMount(() => void vm.loadDirectory())
  // Re-loads whenever the picker changes (including back to '' via the
  // FilterChips "All" chip, which the viewmodel treats as "deselected").
  $effect(() => {
    vm.selectedId
    void vm.loadSelected()
  })

  const REGIME_TONE: Record<string, Tone> = {
    Prompt: 'success',
    Standard: 'info',
    Slow: 'warning',
    AtRisk: 'danger',
  }
  const GRADE_TONE: Record<string, Tone> = {
    A: 'success',
    B: 'info',
    C: 'warning',
    D: 'danger',
  }

  function disputeTone(n: number): Tone {
    return n === 0 ? 'success' : n <= 2 ? 'warning' : 'danger'
  }
  function paymentDaysTone(n: number): Tone {
    return n <= 30 ? 'success' : n <= 60 ? 'warning' : 'danger'
  }

  function toListRow(p: GradePrediction): ListRow {
    const pct = Math.round(p.confidence * 100)
    return {
      label: `Grade ${p.grade || 'Unknown'} — ${formatDate(p.date)}`,
      detail: `Confidence ${pct}% · Predicted ${p.predictedDays} days`,
      value: `${pct}%`,
      tone: GRADE_TONE[p.grade] ?? 'neutral',
    }
  }

  const TABS: { key: Customer360Tab; label: string }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'predictions', label: 'Predictions' },
    { key: 'connections', label: 'Connections' },
  ]

  const subtitle = $derived(
    vm.data ? `${vm.data.code} · ${vm.data.regime || 'Unknown regime'}` : 'Pick a customer to view their 360 profile.',
  )
</script>

<PageShell title="Customer 360" {subtitle}>
  {#snippet toolbar()}
    <Toolbar>
      <FilterChips
        label="Customer"
        options={vm.directory.map((c) => ({ value: c.id, label: c.name || '(no name)' }))}
        bind:selected={vm.selectedId}
      />
    </Toolbar>
  {/snippet}

  {#if vm.loadingDirectory}
    <EmptyState message="Loading customers…" />
  {:else if vm.directoryError}
    <EmptyState message={`Could not load customers: ${vm.directoryError}`}>
      {#snippet actions()}
        <Button onclick={() => vm.loadDirectory()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if !vm.selectedId}
    <EmptyState message="Select a customer above to view their 360 profile." />
  {:else if vm.loading}
    <EmptyState message="Loading customer 360…" />
  {:else if vm.error}
    <EmptyState message={`Could not load this customer: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.loadSelected()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if vm.data && vm.connections}
    {@const c = vm.data}
    {@const conn = vm.connections}
    <Grid min="340px" gap="lg">
      <Card>
        <Stack gap="lg">
          <Row justify="between" wrap gap="sm">
            <Stack gap="xs">
              <span class="c-name">{c.name || '—'}</span>
              <span class="c-code">{c.code}</span>
            </Stack>
            <Badge tone={REGIME_TONE[c.regime] ?? 'neutral'} label={c.regime || 'Unknown regime'} />
          </Row>
          <StatTileGrid
            sections={[
              {
                title: 'Financial',
                items: [
                  { label: 'Lifetime Value', value: c.lifetimeValue, content: 'money' },
                  { label: 'Avg Payment Days', value: `${c.avgPaymentDays} days`, tone: paymentDaysTone(c.avgPaymentDays) },
                  { label: 'Disputes', value: c.disputeCount, tone: disputeTone(c.disputeCount) },
                ],
              },
            ]}
          />
        </Stack>
      </Card>

      <Card>
        <Stack gap="lg">
          <Row gap="sm">
            {#each TABS as t (t.key)}
              <Button variant={vm.tab === t.key ? 'primary' : 'ghost'} onclick={() => (vm.tab = t.key)}>
                {t.label}
              </Button>
            {/each}
          </Row>

          {#if vm.tab === 'overview'}
            <Stack gap="lg">
              <StatTileGrid
                sections={[
                  {
                    title: 'Contact',
                    items: [
                      { label: 'Contact Person', value: c.contact.contactPerson || '—' },
                      { label: 'Phone', value: c.contact.phone || '—' },
                      { label: 'Email', value: c.contact.email || '—' },
                      { label: 'Address', value: c.contact.address || '—' },
                    ],
                  },
                ]}
              />
              <StatTileGrid
                sections={[
                  {
                    title: 'Commercial',
                    items: [
                      { label: 'Payment Terms', value: c.commercial.paymentTerms },
                      { label: 'Credit Limit', value: c.commercial.creditLimit, content: 'money' },
                      { label: 'TRN', value: c.commercial.trn || '—' },
                      { label: 'Industry', value: c.commercial.industry || '—' },
                      { label: 'Relationship', value: `${c.commercial.relationYears} yrs` },
                    ],
                  },
                ]}
              />
            </Stack>
          {:else if vm.tab === 'predictions'}
            {#if c.predictions.length === 0}
              <p class="c-note">No grade predictions recorded for this customer.</p>
            {:else}
              <ListWidget rows={c.predictions.map(toListRow)} />
            {/if}
          {:else}
            <Stack gap="lg">
              <StatTileGrid
                sections={[
                  {
                    title: 'Network',
                    items: [
                      { label: 'Total Connections', value: conn.totalConnections },
                      { label: 'Centrality Score', value: `${Math.round(conn.centralityScore * 100)}%` },
                    ],
                  },
                ]}
              />
              <Stack gap="sm">
                <span class="c-section-label">Related Products</span>
                {#if conn.relatedProducts.length === 0}
                  <p class="c-note">No related products recorded.</p>
                {:else}
                  <Row gap="xs" wrap>
                    {#each conn.relatedProducts as p (p)}
                      <Badge tone="neutral" label={p || '—'} />
                    {/each}
                  </Row>
                {/if}
              </Stack>
              <Stack gap="sm">
                <span class="c-section-label">Related Suppliers</span>
                {#if conn.relatedSuppliers.length === 0}
                  <p class="c-note">No related suppliers recorded.</p>
                {:else}
                  <Row gap="xs" wrap>
                    {#each conn.relatedSuppliers as s (s)}
                      <Badge tone="neutral" label={s || '—'} />
                    {/each}
                  </Row>
                {/if}
              </Stack>
            </Stack>
          {/if}
        </Stack>
      </Card>
    </Grid>
  {/if}
</PageShell>

<style>
  /* Typography only (L1) — no layout/spacing rules here, that's Grid/Row/
   * Stack's job. Mirrors Pricing.svelte's p-customer-name / p-section-label
   * treatment. */
  .c-name {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
  .c-code {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: var(--meta-size);
    color: var(--text-secondary);
  }
  .c-section-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
  .c-note {
    color: var(--text-secondary);
    font-size: var(--table-text-size);
  }
</style>
