<script lang="ts">
  /* Customer 360 — bespoke-on-primitives (K4). A rich single-customer detail
   * view: no master list behind it, so a small synthetic picker (FilterChips)
   * stands in for "which customer am I looking at." Left: identity panel with
   * a grade badge, risk-flag badges + headline financial stats. Right: a
   * tabbed panel (Overview / Activity / Predictions / Connections) built as a
   * plain button row, not an ejected tab primitive — four fixed,
   * mutually-exclusive tabs, no dynamic tab set. State/load/tab logic lives in
   * customer-360.svelte.ts (L5); this file only composes primitives and
   * renders (L1) — no raw layout CSS, no fetch/mutation calls of its own.
   *
   * Shape follows the backend (main.Customer360Data + main.Customer360Graph)
   * verbatim: no invented contact/TRN/credit fields. Fields the backend can't
   * provide are honest-blanked, never fabricated. See
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
  import { formatDate, formatMoney } from '$kernel/format'
  import { Customer360ViewModel, type Customer360Tab } from './customer-360.svelte'
  import type {
    Customer360Info,
    CustomerOpportunity,
    CustomerOrder,
    CustomerPayment,
    CustomerPrediction,
  } from '../bridge/customer-360'

  let { embedded = false }: { embedded?: boolean } = $props()

  const vm = new Customer360ViewModel()
  onMount(() => void vm.loadDirectory())
  // Re-loads whenever the picker changes (including back to '' via the
  // FilterChips "All" chip, which the viewmodel treats as "deselected").
  $effect(() => {
    vm.selectedId
    void vm.loadSelected()
  })

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

  /** Risk/behaviour flags shown only when true — never a fabricated "no". */
  function activeFlags(c: Customer360Info): { label: string; tone: Tone }[] {
    const f: { label: string; tone: Tone }[] = []
    if (c.isCreditBlocked) f.push({ label: 'Credit Blocked', tone: 'danger' })
    if (c.requiresPrepayment) f.push({ label: 'Prepayment Required', tone: 'warning' })
    if (c.hasAbbCompetition) f.push({ label: 'ABB Competition', tone: 'info' })
    if (c.isEmergencyOnly) f.push({ label: 'Emergency Only', tone: 'warning' })
    return f
  }

  const pct = (v: number): string => `${Math.round(v * 100)}%`

  function predictionRow(p: CustomerPrediction): ListRow {
    return {
      label: `Grade ${p.grade || 'Unknown'}`,
      detail: `Predicted ${p.predictedDays} days · ${formatDate(p.createdAt)}`,
      value: pct(p.confidence),
      tone: GRADE_TONE[p.grade] ?? 'neutral',
    }
  }
  function orderRow(o: CustomerOrder): ListRow {
    return {
      label: o.orderNumber || '(no number)',
      detail: `${formatDate(o.orderDate)} · ${o.status || '—'}`,
      value: formatMoney(o.totalValueBhd),
    }
  }
  function opportunityRow(o: CustomerOpportunity): ListRow {
    return {
      label: o.project || '(untitled opportunity)',
      detail: `${o.status || '—'} · ${formatDate(o.createdAt)}`,
      value: formatMoney(o.value),
    }
  }
  function paymentRow(p: CustomerPayment): ListRow {
    return {
      label: p.invoiceNumber || '(no invoice)',
      detail: `${formatDate(p.paymentDate)} · ${p.paymentMethod || '—'} · ${p.daysToPayment} days to pay`,
      value: formatMoney(p.amountBhd),
    }
  }

  const TABS: { key: Customer360Tab; label: string }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'activity', label: 'Activity' },
    { key: 'predictions', label: 'Predictions' },
    { key: 'connections', label: 'Connections' },
  ]

  const subtitle = $derived(
    vm.data
      ? `${vm.data.customerType || 'Customer'} · Grade ${vm.data.grade || 'Unknown'}`
      : 'Pick a customer to view their 360 profile.',
  )
</script>

<PageShell {embedded} title="Customer 360" {subtitle}>
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
    {@const flags = activeFlags(c)}
    <Grid min="340px" gap="lg">
      <Card>
        <Stack gap="lg">
          <Row justify="between" wrap gap="sm">
            <Stack gap="xs">
              <span class="c-name">{c.name || '—'}</span>
              <span class="c-code">
                {[c.customerType, [c.city, c.country].filter(Boolean).join(', ')].filter(Boolean).join(' · ') || '—'}
              </span>
            </Stack>
            <Badge tone={GRADE_TONE[c.grade] ?? 'neutral'} label={`Grade ${c.grade || 'Unknown'}`} />
          </Row>

          {#if flags.length > 0}
            <Row gap="xs" wrap>
              {#each flags as f (f.label)}
                <Badge tone={f.tone} label={f.label} />
              {/each}
            </Row>
          {/if}

          <StatTileGrid
            sections={[
              {
                title: 'Financial',
                items: [
                  { label: 'Lifetime Value', value: c.lifetimeValue, content: 'money' },
                  { label: 'Total Orders Value', value: c.totalOrdersValue, content: 'money' },
                  { label: 'Total Orders', value: c.totalOrdersCount, content: 'quantity' },
                  { label: 'Avg Order Value', value: c.avgOrderValue, content: 'money' },
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
          <Row gap="sm" wrap>
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
                    title: 'Profile',
                    items: [
                      { label: 'Type', value: c.customerType || '—' },
                      { label: 'Industry', value: c.industry || '—' },
                      { label: 'City', value: c.city || '—' },
                      { label: 'Country', value: c.country || '—' },
                      { label: 'Relationship', value: `${c.relationYears} yrs` },
                      { label: 'Payment Terms', value: `${c.paymentTermsDays} days` },
                    ],
                  },
                ]}
              />
              <StatTileGrid
                sections={[
                  {
                    title: 'Three-Regime Dynamics',
                    items: [
                      { label: 'R1', value: pct(c.r1) },
                      { label: 'R2', value: pct(c.r2) },
                      { label: 'R3', value: pct(c.r3) },
                    ],
                  },
                ]}
              />
              <StatTileGrid
                sections={[
                  {
                    title: 'Receivables Aging',
                    items: [
                      { label: 'Current', value: c.receivablesAging.current, content: 'money' },
                      { label: '30–60 days', value: c.receivablesAging.days30_60, content: 'money' },
                      { label: '60–90 days', value: c.receivablesAging.days60_90, content: 'money' },
                      { label: '90–120 days', value: c.receivablesAging.days90_120, content: 'money' },
                      { label: '120+ days', value: c.receivablesAging.days120plus, content: 'money' },
                      { label: 'Total Outstanding', value: c.receivablesAging.totalOutstanding, content: 'money' },
                    ],
                  },
                ]}
              />
            </Stack>
          {:else if vm.tab === 'activity'}
            <Stack gap="lg">
              <Stack gap="sm">
                <span class="c-section-label">Recent Orders</span>
                {#if c.recentOrders.length === 0}
                  <p class="c-note">No recent orders recorded.</p>
                {:else}
                  <ListWidget rows={c.recentOrders.map(orderRow)} />
                {/if}
              </Stack>
              <Stack gap="sm">
                <span class="c-section-label">Open Opportunities</span>
                {#if c.openOpportunities.length === 0}
                  <p class="c-note">No open opportunities recorded.</p>
                {:else}
                  <ListWidget rows={c.openOpportunities.map(opportunityRow)} />
                {/if}
              </Stack>
              <Stack gap="sm">
                <span class="c-section-label">Payment History</span>
                {#if c.paymentHistory.length === 0}
                  <p class="c-note">No payment history recorded.</p>
                {:else}
                  <ListWidget rows={c.paymentHistory.map(paymentRow)} />
                {/if}
              </Stack>
            </Stack>
          {:else if vm.tab === 'predictions'}
            {#if c.recentPredictions.length === 0}
              <p class="c-note">No grade predictions recorded for this customer.</p>
            {:else}
              <ListWidget rows={c.recentPredictions.map(predictionRow)} />
            {/if}
          {:else}
            <Stack gap="lg">
              <StatTileGrid
                sections={[
                  {
                    title: 'Network',
                    items: [{ label: 'Total Connections', value: conn.totalConnections, content: 'quantity' }],
                  },
                ]}
              />
              <Stack gap="sm">
                <span class="c-section-label">Related Products</span>
                {#if conn.relatedProducts.length === 0}
                  <p class="c-note">No related products recorded.</p>
                {:else}
                  <Row gap="xs" wrap>
                    {#each conn.relatedProducts as p, idx (idx)}
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
                    {#each conn.relatedSuppliers as s, idx (idx)}
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
