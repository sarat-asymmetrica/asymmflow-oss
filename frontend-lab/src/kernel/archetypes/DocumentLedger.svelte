<script lang="ts" generics="Row">
  import type { LedgerDescriptor } from '../descriptor'
  import { LedgerViewModel } from '../ledger.svelte'
  import PageShell from '../primitives/PageShell.svelte'
  import Toolbar from '../primitives/Toolbar.svelte'
  import Card from '../primitives/Card.svelte'
  import Stack from '../primitives/Stack.svelte'
  import Row_ from '../primitives/Row.svelte'
  import DataTable from '../primitives/DataTable.svelte'
  import SearchInput from '../controls/SearchInput.svelte'
  import FilterChips from '../controls/FilterChips.svelte'
  import Button from '../controls/Button.svelte'
  import EmptyState from '../controls/EmptyState.svelte'
  import { renderCell } from '../content'

  let { descriptor }: { descriptor: LedgerDescriptor<Row> } = $props()

  // VM rebuilds if (and only if) the descriptor prop changes; the effect
  // fetches once per VM instance (Wave-10 lesson: no double-fetch paths).
  const vm = $derived(new LedgerViewModel(descriptor))
  $effect(() => {
    void vm.load()
  })

  const screenActions = $derived((descriptor.actions ?? []).filter((a) => a.kind === 'screen'))
  const rowActions = $derived((descriptor.actions ?? []).filter((a) => a.kind === 'row'))

  const reload = () => vm.load()
</script>

<PageShell
  title={descriptor.title}
  subtitle={vm.loading ? 'Loading…' : `${vm.visible.length} of ${vm.rows.length}`}
>
  {#snippet actions()}
    <Row_ gap="sm">
      {#each screenActions as action (action.key)}
        <Button variant="primary" onclick={() => action.run({ row: null, reload })}>
          {action.label}
        </Button>
      {/each}
    </Row_>
  {/snippet}

  {#snippet toolbar()}
    <Toolbar>
      <SearchInput bind:value={vm.search} placeholder="Search {descriptor.entity}…" />
      {#each vm.filterOptions as f (f.spec.key)}
        <FilterChips
          label={f.spec.label}
          options={f.options}
          bind:selected={
            () => vm.filters[f.spec.key] ?? '',
            (v) => (vm.filters = { ...vm.filters, [f.spec.key]: v })
          }
        />
      {/each}
    </Toolbar>
  {/snippet}

  {#if vm.error}
    <EmptyState message="Could not load {descriptor.entity}: {vm.error}">
      {#snippet actions()}
        <Button onclick={reload}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if vm.loading}
    <Card padding="lg">
      <div class="k-ledger-skeleton" aria-hidden="true">
        {#each Array(8) as _unused, i (i)}
          <div class="k-skeleton-row"></div>
        {/each}
      </div>
    </Card>
  {:else if vm.visible.length === 0}
    {#if descriptor.slots?.empty}
      {@const Empty = descriptor.slots.empty}
      <Empty />
    {:else}
      <EmptyState
        message={vm.rows.length === 0
          ? (descriptor.emptyMessage ?? `No ${descriptor.entity} yet.`)
          : 'Nothing matches the current search and filters.'}
      />
    {/if}
  {:else}
    <div class="k-ledger-body">
      <Card padding="none">
        <DataTable
          columns={descriptor.columns}
          rows={vm.visible}
          id={descriptor.id}
          status={descriptor.status}
          selectedId={vm.selectedId}
          onSelect={(row) => vm.select(vm.selectedId === descriptor.id(row) ? null : row)}
        />
      </Card>

      {#if vm.selected}
        {@const row = vm.selected}
        <div class="k-ledger-detail">
          <Card padding="lg">
            {#if descriptor.slots?.detail}
              {@const Detail = descriptor.slots.detail}
              <Detail {row} {reload} />
            {:else}
              <Stack gap="sm">
                {#each descriptor.columns as col (col.key)}
                  <div class="k-detail-field">
                    <span class="k-detail-label">{col.label}</span>
                    <span class="k-detail-value"
                      >{renderCell(col.content, col.value(row), col.currency?.(row))}</span
                    >
                  </div>
                {/each}
                {#if rowActions.some((a) => !a.visible || a.visible(row))}
                  <Row_ gap="sm" wrap>
                    {#each rowActions.filter((a) => !a.visible || a.visible(row)) as action (action.key)}
                      <Button onclick={() => action.run({ row, reload })}>{action.label}</Button>
                    {/each}
                  </Row_>
                {/if}
              </Stack>
            {/if}
          </Card>
        </div>
      {/if}
    </div>
  {/if}
</PageShell>

<style>
  /* Archetype-owned layout (archetypes are kernel; L1 binds screens). */
  .k-ledger-body {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-md);
    flex-wrap: wrap;
    min-width: 0;
  }
  .k-ledger-body > :global(:first-child) {
    flex: 1 1 480px;
    min-width: 0;
  }
  .k-ledger-detail {
    flex: 0 1 340px;
    min-width: 260px;
  }
  .k-detail-field {
    display: flex;
    justify-content: space-between;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-detail-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.02em;
    color: var(--text-secondary);
    flex-shrink: 0;
  }
  .k-detail-value {
    text-align: right;
    overflow-wrap: anywhere;
    min-width: 0;
  }
  .k-ledger-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-sm);
  }
  .k-skeleton-row {
    height: var(--table-row-height);
    border-radius: var(--border-radius-sm);
    background: var(--onyx-tint);
  }
</style>
