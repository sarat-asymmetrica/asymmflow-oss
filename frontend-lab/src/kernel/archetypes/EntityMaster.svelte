<script lang="ts" generics="Row">
  import type { EntityDescriptor } from '../descriptor'
  import { LedgerViewModel } from '../ledger.svelte'
  import PageShell from '../primitives/PageShell.svelte'
  import Toolbar from '../primitives/Toolbar.svelte'
  import Card from '../primitives/Card.svelte'
  import Stack from '../primitives/Stack.svelte'
  import Row_ from '../primitives/Row.svelte'
  import FormGrid from '../primitives/FormGrid.svelte'
  import DataTable from '../primitives/DataTable.svelte'
  import SearchInput from '../controls/SearchInput.svelte'
  import FilterChips from '../controls/FilterChips.svelte'
  import Button from '../controls/Button.svelte'
  import Badge from '../controls/Badge.svelte'
  import EmptyState from '../controls/EmptyState.svelte'
  import { renderCell } from '../content'

  let { descriptor }: { descriptor: EntityDescriptor<Row> } = $props()

  // Same viewmodel as DocumentLedger — one query path (L2); the archetypes
  // differ only in how they RENDER the selection.
  const vm = $derived(new LedgerViewModel(descriptor))
  $effect(() => {
    void vm.load()
  })

  const screenActions = $derived((descriptor.actions ?? []).filter((a) => a.kind === 'screen'))
  const rowActions = $derived((descriptor.actions ?? []).filter((a) => a.kind === 'row'))
  const reload = () => vm.load()

  function badgeToneFor(row: Row): 'neutral' | 'info' | 'success' | 'warning' | 'danger' {
    const b = descriptor.profile.badge
    if (!b) return 'neutral'
    return b.tones[b.value(row)] ?? 'neutral'
  }
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
      <div class="k-master-skeleton" aria-hidden="true">
        {#each Array(8) as _unused, i (i)}
          <div class="k-skeleton-row"></div>
        {/each}
      </div>
    </Card>
  {:else if vm.visible.length === 0}
    <EmptyState
      message={vm.rows.length === 0
        ? (descriptor.emptyMessage ?? `No ${descriptor.entity} yet.`)
        : 'Nothing matches the current search and filters.'}
    />
  {:else}
    <div class="k-master-body">
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
        <div class="k-master-profile">
          <Card padding="lg">
            <Stack gap="md">
              <div class="k-profile-head">
                <div class="k-profile-titles">
                  <h2 class="k-profile-heading">{descriptor.profile.heading(row)}</h2>
                  {#if descriptor.profile.subheading}
                    <p class="k-profile-subheading">{descriptor.profile.subheading(row)}</p>
                  {/if}
                </div>
                {#if descriptor.profile.badge}
                  <Badge
                    tone={badgeToneFor(row)}
                    label={String(descriptor.profile.badge.value(row) ?? '—')}
                  />
                {/if}
              </div>

              {#if descriptor.profile.kpis?.length}
                <div class="k-profile-kpis">
                  {#each descriptor.profile.kpis as kpi (kpi.label)}
                    <div class="k-kpi">
                      <span class="k-kpi-label">{kpi.label}</span>
                      <span class="k-kpi-value"
                        >{renderCell(kpi.content, kpi.value(row), kpi.currency?.(row))}</span
                      >
                    </div>
                  {/each}
                </div>
              {/if}

              {#each descriptor.profile.sections as section (section.title)}
                <Stack gap="sm">
                  <h3 class="k-profile-section-title">{section.title}</h3>
                  <FormGrid columns={2}>
                    {#each section.fields as field (field.label)}
                      <div class="k-profile-field">
                        <span class="k-profile-label">{field.label}</span>
                        <span class="k-profile-value"
                          >{renderCell(field.content, field.value(row), field.currency?.(row))}</span
                        >
                      </div>
                    {/each}
                  </FormGrid>
                </Stack>
              {/each}

              {#if rowActions.some((a) => !a.visible || a.visible(row))}
                <Row_ gap="sm" wrap>
                  {#each rowActions.filter((a) => !a.visible || a.visible(row)) as action (action.key)}
                    <Button onclick={() => action.run({ row, reload })}>{action.label}</Button>
                  {/each}
                </Row_>
              {/if}
            </Stack>
          </Card>
        </div>
      {/if}
    </div>
  {/if}
</PageShell>

<style>
  /* Archetype-owned layout. Entity work is profile-centric: the profile
   * panel gets the wider share (inverse of DocumentLedger's split). */
  .k-master-body {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-md);
    flex-wrap: wrap;
    min-width: 0;
  }
  .k-master-body > :global(:first-child) {
    flex: 1 1 380px;
    min-width: 0;
  }
  .k-master-profile {
    flex: 1 1 420px;
    min-width: 300px;
  }
  .k-profile-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--k-space-md);
    flex-wrap: wrap; /* anti-collapse */
    min-width: 0;
  }
  .k-profile-titles {
    min-width: 0;
    flex: 1 1 200px;
  }
  .k-profile-heading {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: 700;
    overflow-wrap: anywhere;
  }
  .k-profile-subheading {
    color: var(--text-secondary);
    font-size: var(--meta-size);
    margin-top: var(--k-space-xs);
    overflow-wrap: anywhere;
  }
  .k-profile-kpis {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(120px, 100%), 1fr));
    gap: var(--k-space-sm);
  }
  .k-kpi {
    background: var(--surface-elevated);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    padding: var(--k-space-sm) var(--k-space-md);
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .k-kpi-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.02em;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .k-kpi-value {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(15px * var(--ui-font-scale));
    font-weight: 600;
    overflow-wrap: anywhere;
  }
  .k-profile-section-title {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
    border-bottom: var(--border-width) solid var(--border);
    padding-bottom: var(--k-space-xs);
  }
  .k-profile-field {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .k-profile-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.02em;
    color: var(--text-secondary);
  }
  .k-profile-value {
    font-size: calc(13px * var(--ui-font-scale));
    overflow-wrap: anywhere;
  }
  .k-master-skeleton {
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
