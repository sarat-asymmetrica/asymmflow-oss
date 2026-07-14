<script lang="ts" generics="Data">
  /* The Hub archetype — KPI strip + a mixed widget grid, from one typed
   * dashboard payload. Fourth archetype after DocumentLedger/EntityMaster/
   * FormModal. Widgets are declared data; this shell computes each widget's
   * data from the payload and dispatches to a presentational widget component.
   * Honest states only: a failed load shows an error, never fabricated numbers. */
  import type { HubDescriptor, Navigate } from '../hub'
  import { HubViewModel } from '../hub.svelte'
  import PageShell from '../primitives/PageShell.svelte'
  import Card from '../primitives/Card.svelte'
  import Stack from '../primitives/Stack.svelte'
  import Row_ from '../primitives/Row.svelte'
  import EmptyState from '../controls/EmptyState.svelte'
  import Button from '../controls/Button.svelte'
  import { renderCell } from '../content'
  import DistributionWidget from '../widgets/DistributionWidget.svelte'
  import RankedBarList from '../widgets/RankedBarList.svelte'
  import StatTileGrid from '../widgets/StatTileGrid.svelte'
  import ListWidget from '../widgets/ListWidget.svelte'
  import ActivityFeed from '../widgets/ActivityFeed.svelte'
  import CalloutWidget from '../widgets/CalloutWidget.svelte'
  import ComparisonBars from '../widgets/ComparisonBars.svelte'
  import DonutWidget from '../widgets/DonutWidget.svelte'

  let {
    descriptor,
    navigate,
  }: {
    descriptor: HubDescriptor<Data>
    /** Drill-down handler — KPIs/widgets seed a ledger's initialQuery. */
    navigate?: Navigate
  } = $props()

  const vm = $derived(new HubViewModel(descriptor))
  $effect(() => {
    void vm.load()
  })

  const nav: Navigate = (intent) => navigate?.(intent)
</script>

<PageShell
  title={descriptor.title}
  subtitle={vm.loading ? 'Loading…' : vm.data && descriptor.subtitle ? descriptor.subtitle(vm.data) : ''}
>
  {#snippet actions()}
    {#if descriptor.period}
      <div class="k-hub-period" role="group" aria-label={descriptor.period.label}>
        {#each descriptor.period.options as opt (opt.value)}
          <button
            class="k-hub-period-btn"
            class:active={vm.period === opt.value}
            onclick={() => vm.setPeriod(opt.value)}
          >
            {opt.label}
          </button>
        {/each}
      </div>
    {/if}
  {/snippet}

  {#if vm.error}
    <EmptyState message="Could not load {descriptor.entity}: {vm.error}">
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if vm.loading}
    <Card padding="lg">
      <div class="k-hub-skeleton" aria-hidden="true">
        {#each Array(4) as _u, i (i)}<div class="k-skeleton-tile"></div>{/each}
      </div>
    </Card>
  {:else if vm.data}
    {@const data = vm.data}
    <Stack gap="md">
      <!-- KPI strip -->
      {#if descriptor.kpis.length}
        <Card padding="lg">
          <div class="k-hub-kpis">
            {#each descriptor.kpis as kpi (kpi.label)}
              {@const target = kpi.nav?.(data) ?? null}
              {@const tone = kpi.tone?.(data)}
              {@const delta = kpi.delta?.(data) ?? null}
              <svelte:element
                this={target ? 'button' : 'div'}
                role={target ? 'button' : undefined}
                tabindex={target ? 0 : undefined}
                class="k-hub-kpi"
                class:clickable={!!target}
                onclick={target ? () => nav(target) : undefined}
              >
                <span class="k-hub-kpi-label">{kpi.label}</span>
                <span
                  class="k-hub-kpi-value"
                  style:color={tone ? `var(--k-tone-${tone}-fg)` : undefined}
                >
                  {renderCell(kpi.content, kpi.value(data), kpi.currency?.(data))}
                </span>
                {#if delta}
                  <span class="k-hub-kpi-delta" style:color={`var(--k-tone-${delta.tone}-fg)`}>
                    {delta.text}
                  </span>
                {/if}
              </svelte:element>
            {/each}
          </div>
        </Card>
      {/if}

      <!-- Widget grid -->
      <div class="k-hub-grid">
        {#each descriptor.widgets as widget, i (widget.title + i)}
          <div class="k-hub-cell" class:wide={widget.span === 2}>
            <Card padding="lg">
              <Stack gap="sm">
                <h3 class="k-hub-widget-title">{widget.title}</h3>
                {#if widget.type === 'distribution'}
                  <DistributionWidget
                    segments={widget.segments(data)}
                    orientation={widget.orientation ?? 'horizontal'}
                    navigate={nav}
                  />
                {:else if widget.type === 'ranked'}
                  <RankedBarList rows={widget.rows(data)} unit={widget.unit ?? 'quantity'} navigate={nav} />
                {:else if widget.type === 'stat-grid'}
                  <StatTileGrid sections={widget.sections(data)} navigate={nav} />
                {:else if widget.type === 'list'}
                  <ListWidget rows={widget.rows(data)} navigate={nav} />
                {:else if widget.type === 'activity'}
                  <ActivityFeed
                    items={widget.items(data)}
                    emptyMessage={widget.emptyMessage ?? 'Nothing here yet.'}
                    navigate={nav}
                  />
                {:else if widget.type === 'callout'}
                  <CalloutWidget items={widget.items(data)} />
                {:else if widget.type === 'donut'}
                  <DonutWidget segments={widget.segments(data)} centerLabel={widget.centerLabel} />
                {:else if widget.type === 'comparison'}
                  <ComparisonBars
                    rows={widget.rows(data)}
                    baseLabel={widget.baseLabel}
                    currentLabel={widget.currentLabel}
                  />
                {:else if widget.type === 'bespoke'}
                  {@const Bespoke = widget.component}
                  <Bespoke {data} navigate={nav} />
                {/if}
              </Stack>
            </Card>
          </div>
        {/each}
      </div>
    </Stack>
  {/if}
</PageShell>

<style>
  .k-hub-period {
    display: flex;
    gap: 2px;
    background: var(--onyx-tint);
    border-radius: var(--border-radius-pill);
    padding: 2px;
  }
  .k-hub-period-btn {
    font: inherit;
    font-size: calc(12px * var(--ui-font-scale));
    padding: 4px 12px;
    border: none;
    border-radius: var(--border-radius-pill);
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    white-space: nowrap;
  }
  .k-hub-period-btn.active {
    background: var(--surface);
    color: var(--text-primary);
    font-weight: 600;
  }
  .k-hub-kpis {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(180px, 100%), 1fr));
    gap: var(--k-space-lg);
  }
  .k-hub-kpi {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
    text-align: left;
    border: none;
    background: transparent;
    padding: 0;
    font: inherit;
  }
  .k-hub-kpi.clickable {
    cursor: pointer;
  }
  .k-hub-kpi.clickable:hover .k-hub-kpi-value {
    text-decoration: underline;
  }
  .k-hub-kpi-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .k-hub-kpi-value {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(24px * var(--ui-font-scale));
    font-weight: 700;
    line-height: 1.1;
    overflow-wrap: anywhere;
  }
  .k-hub-kpi-delta {
    font-size: calc(11px * var(--ui-font-scale));
    font-weight: 600;
  }
  .k-hub-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(320px, 100%), 1fr));
    gap: var(--k-space-md);
    align-items: start;
    min-width: 0;
  }
  .k-hub-cell {
    min-width: 0;
  }
  .k-hub-cell.wide {
    grid-column: 1 / -1;
  }
  .k-hub-widget-title {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
  .k-hub-skeleton {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
    gap: var(--k-space-md);
  }
  .k-skeleton-tile {
    height: 64px;
    border-radius: var(--border-radius-sm);
    background: var(--onyx-tint);
  }
</style>
