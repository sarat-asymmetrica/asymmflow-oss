<script lang="ts">
  /* Pricing — bespoke-on-primitives (K4). Margin simulator: pick a customer
   * from the sidebar list, drag the target-margin slider, read the
   * projected win rate + guidance. Only `SimulateMargin` is real on the old
   * screen — the customer list + win rates here are synthetic mock, not a
   * port of the old screen's hardcoded array (see bridge/pricing.ts and
   * screens/parity/Pricing.parity.md). */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import RangeSlider from '$kernel/controls/RangeSlider.svelte'
  import ListWidget from '$kernel/widgets/ListWidget.svelte'
  import StatTileGrid from '$kernel/widgets/StatTileGrid.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import type { ListRow, NavIntent } from '$kernel/hub'
  import type { Tone } from '$kernel/tones'
  import { PricingViewModel } from './pricing-vm.svelte'
  import type { PricingCustomerRow, Regime } from '../bridge/pricing'

  const vm = new PricingViewModel()
  onMount(() => void vm.load())

  const REGIME_TONE: Record<Regime, Tone> = {
    Premium: 'success',
    PriceSensitive: 'danger',
    ValueBalanced: 'warning',
  }

  const REGIME_GUIDANCE: Record<Regime, string> = {
    Premium: 'Prioritizes quality and reliability — higher margins are accepted if service levels are high.',
    PriceSensitive: 'Price is the primary driver — competitive margins (10–15%) are critical for winning deals.',
    ValueBalanced: 'Balances cost and value — mid-range margins (18–22%) with value-adds work best.',
  }

  function toListRow(c: PricingCustomerRow): ListRow {
    const wr = Math.round(c.currentWinRate * 100)
    return {
      label: c.name,
      detail: `${c.regime} · WR ${wr}%`,
      value: `${wr}%`,
      tone: REGIME_TONE[c.regime],
      nav: { key: `pricing-select:${c.id}` },
    }
  }

  function onSelect(intent: NavIntent) {
    const id = intent.key.slice('pricing-select:'.length)
    const row = vm.customers.find((c) => c.id === id)
    if (row) vm.select(row)
  }
</script>

<PageShell
  title="Pricing"
  subtitle="Margin simulator — pick a customer, drag the target margin, read the projected win rate."
>
  {#if vm.loading}
    <EmptyState message="Loading customers…" />
  {:else if vm.error}
    <EmptyState message={`Could not load customers: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else}
    <Grid min="320px" gap="lg">
      <Card>
        <Stack gap="sm">
          <span class="p-section-label">Customers</span>
          <ListWidget rows={vm.customers.map(toListRow)} navigate={onSelect} />
        </Stack>
      </Card>

      <Card>
        {#if !vm.selected}
          <EmptyState message="Select a customer to analyze pricing strategy." />
        {:else}
          {@const c = vm.selected}
          <Stack gap="lg">
            <Row justify="between" wrap>
              <span class="p-customer-name">{c.name}</span>
              <Badge tone={REGIME_TONE[c.regime]} label={`${c.regime} strategy`} />
            </Row>

            <Stack gap="sm">
              <RangeSlider
                label="Target margin"
                bind:value={vm.targetMargin}
                min={0.05}
                max={0.5}
                step={0.01}
                formatValue={(v) => `${Math.round(v * 100)}%`}
              />
              <Row justify="end">
                <Button variant="primary" onclick={() => vm.simulate()} disabled={vm.simulating}>
                  {vm.simulating ? 'Simulating…' : 'Run Simulation'}
                </Button>
              </Row>
            </Stack>

            {#if vm.simError}
              <CalloutWidget items={[{ label: 'Simulation failed', text: vm.simError, tone: 'danger' }]} />
            {:else if vm.result}
              {@const r = vm.result}
              <Stack gap="sm">
                <StatTileGrid
                  sections={[
                    {
                      title: 'Projected outcome',
                      items: [
                        { label: 'Current win rate', value: `${Math.round(r.currentWinRate * 100)}%` },
                        {
                          label: 'Projected win rate',
                          value: `${Math.round(r.projectedWinRate * 100)}%`,
                          tone: r.projectedWinRate >= r.currentWinRate ? 'success' : 'danger',
                        },
                        { label: 'Confidence', value: `${Math.round(r.confidence * 100)}%` },
                      ],
                    },
                  ]}
                />
                <CalloutWidget
                  items={[
                    { label: 'Guidance', text: r.recommendedAction, tone: 'info' },
                    ...(r.warning ? [{ label: 'Warning', text: r.warning, tone: 'warning' as Tone }] : []),
                  ]}
                />
              </Stack>
            {/if}

            <CalloutWidget
              items={[{ label: `${c.regime} regime`, text: REGIME_GUIDANCE[c.regime], tone: 'neutral' }]}
            />
          </Stack>
        {/if}
      </Card>
    </Grid>
  {/if}
</PageShell>

<style>
  /* Typography only (L1) — no layout/spacing rules here, that's Grid/Row/
   * Stack's job. Mirrors StatTileGrid's k-stat-title / EntityMaster's
   * k-profile-heading section-label treatment. */
  .p-section-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
  .p-customer-name {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
</style>
