<script lang="ts" generics="T">
  /**
   * DataShell — canonical list-screen composition.
   *
   * Wires DataToolbar + search Input + filter slot + DataTable + Pagination
   * so every ERP list screen stops repeating 200 lines of boilerplate.
   *
   * Constitution §2.1: no app coupling — typed props in, events out.
   * Constitution §4a: tabular-nums on all numeric cells enforced via Column.numeric.
   * Constitution §4e: R2 optimize for micro-interactions.
   *
   * Client-side: search filtering, sort, pagination windowing.
   * Server-side: caller passes data + controls onSortChange, totalCount for paging.
   */

  import type { Snippet } from 'svelte';
  import {
    DataTable,
    DataToolbar,
    Pagination,
    Input,
  } from '@asymmflow/ui';
  import type { Column, SortState, CellContext } from '@asymmflow/ui';

  // ─── Props ───────────────────────────────────────────────────────────────────

  export interface DataShellProps<T> {
    /** Screen / section title shown in the toolbar. */
    title: string;
    /** The full dataset (client-mode: DataShell filters/sorts/paginates). */
    data: T[];
    /** Column definitions — same type as DataTable. */
    columns: Column<T>[];
    /** Keys to search across (dot-notation supported). Default: all column keys. */
    searchableKeys?: string[];
    /** Page size. Default 25. */
    pageSize?: number;
    /**
     * Row identifier function. Defaults to (row) => (row as any).id.
     * Required when rows don't have an `id` property.
     */
    rowId?: (row: T) => string | number;
    /** Max height for the DataTable scroll region. */
    tableMaxHeight?: string;
    /**
     * Snippet: toolbar right-side action buttons (e.g. "New Invoice").
     * Receives no arguments — just render your buttons.
     */
    actions?: Snippet;
    /**
     * Snippet: additional filter controls rendered between the toolbar and table.
     * E.g. type filters, status toggles.
     */
    filters?: Snippet;
    /**
     * Snippet: custom row-actions overlay (appears on hover in the last cell).
     * Receives the row object.
     */
    rowActions?: Snippet<[T]>;
    /**
     * Snippet: custom cell renderer (forwarded to DataTable's `cell` prop).
     */
    cell?: Snippet<[CellContext<T>]>;
    /**
     * Snippet: custom empty state.
     */
    empty?: Snippet;
    /** Called when user clicks a row. */
    onRowClick?: (row: T) => void;
    /** Label for the DataTable (accessibility). */
    label?: string;
  }

  let {
    title,
    data,
    columns,
    searchableKeys,
    pageSize = 25,
    rowId = (row) => (row as Record<string, unknown>).id as string | number,
    tableMaxHeight = '560px',
    actions,
    filters,
    rowActions,
    cell,
    empty,
    onRowClick,
    label,
  }: DataShellProps<T> = $props();

  // ─── Effective search keys ────────────────────────────────────────────────────

  const effectiveKeys = $derived(
    searchableKeys ?? columns.map((c) => c.key),
  );

  // ─── Search ───────────────────────────────────────────────────────────────────

  let query = $state('');

  // ─── Sort (uncontrolled, client-side) ─────────────────────────────────────────

  let sort = $state<SortState | null>(null);

  function resolveValue(row: T, key: string): unknown {
    return key.split('.').reduce<unknown>((acc, k) => {
      if (acc == null) return undefined;
      return (acc as Record<string, unknown>)[k];
    }, row);
  }

  // ─── Derived: filtered + sorted + paginated ────────────────────────────────────

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) return data;
    return data.filter((row) =>
      effectiveKeys.some((key) => {
        const val = resolveValue(row, key);
        if (val == null) return false;
        return String(val).toLowerCase().includes(q);
      }),
    );
  });

  const sorted = $derived.by(() => {
    if (!sort) return filtered;
    return [...filtered].sort((a, b) => {
      const av = resolveValue(a, sort!.key);
      const bv = resolveValue(b, sort!.key);
      if (av === bv) return 0;
      const cmp = av! > bv! ? 1 : -1;
      return sort!.direction === 'asc' ? cmp : -cmp;
    });
  });

  // ─── Pagination ───────────────────────────────────────────────────────────────

  let currentPage = $state(1);

  // Reset to page 1 when filter/search changes
  $effect(() => {
    void query;
    currentPage = 1;
  });

  const pageData = $derived(
    sorted.slice((currentPage - 1) * pageSize, currentPage * pageSize),
  );
</script>

<div class="af-shell">
  <!-- Toolbar: title + live record count + search + actions -->
  <DataToolbar>
    {#snippet left()}
      <div class="af-shell__title-block">
        <span class="af-section-title af-shell__title">{title}</span>
        <span class="af-shell__count af-meta af-numeric">{filtered.length} records</span>
      </div>
    {/snippet}
    {#snippet right()}
      <!-- Search -->
      <div class="af-shell__search-wrap">
        <svg class="af-shell__search-icon" width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <circle cx="6" cy="6" r="4" stroke="currentColor" stroke-width="1.2" />
          <path d="M9.5 9.5L12 12" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
        </svg>
        <Input
          placeholder="Search {title.toLowerCase()}…"
          bind:value={query}
          aria-label="Search {title}"
          class="af-shell__search-input"
        />
      </div>
      <!-- Caller-provided actions -->
      {#if actions}
        {@render actions()}
      {/if}
    {/snippet}
  </DataToolbar>

  <!-- Optional filter strip -->
  {#if filters}
    <div class="af-shell__filters">
      {@render filters()}
    </div>
  {/if}

  <!-- DataTable: toolbar radius flush at top -->
  <div class="af-shell__table-wrap">
    <DataTable
      data={pageData}
      {columns}
      {rowId}
      onSortChange={(s) => { sort = s; }}
      {cell}
      {rowActions}
      {empty}
      onRowClick={onRowClick}
      label={label ?? title}
      rowCount={filtered.length}
      maxHeight={tableMaxHeight}
    />
  </div>

  <!-- Pagination — only render when data exceeds one page -->
  {#if sorted.length > pageSize}
    <div class="af-shell__pagination">
      <Pagination
        bind:page={currentPage}
        total={sorted.length}
        pageSize={pageSize}
      />
    </div>
  {/if}
</div>

<style>
  .af-shell {
    display: flex;
    flex-direction: column;
  }

  /* Toolbar → table → no gap — they share the border visually */
  .af-shell__table-wrap > :global(.af-table-container) {
    border-radius: 0 0 var(--af-radius-md) var(--af-radius-md);
    border-top: none;
  }

  /* ── Toolbar internals ─────────────────────────────────────────────────── */
  .af-shell__title-block {
    display: flex;
    align-items: baseline;
    gap: var(--af-space-3);
  }

  .af-shell__title {
    line-height: 1;
  }

  .af-shell__count {
    color: var(--af-text-muted);
  }

  /* Search: icon-prefixed, styled to match DataTablePage exemplar */
  .af-shell__search-wrap {
    position: relative;
    display: flex;
    align-items: center;
  }

  .af-shell__search-icon {
    position: absolute;
    left: var(--af-space-2);
    color: var(--af-text-muted);
    pointer-events: none;
    z-index: 1;
  }

  /* Override Input padding for icon overlap */
  .af-shell__search-wrap :global(input) {
    padding-inline-start: calc(var(--af-space-2) + 14px + var(--af-space-1)) !important;
    width: 200px;
  }

  /* ── Filter strip ──────────────────────────────────────────────────────── */
  .af-shell__filters {
    padding: var(--af-space-2) var(--af-space-3);
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
    flex-wrap: wrap;
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-top: none;
  }

  /* ── Pagination strip ──────────────────────────────────────────────────── */
  .af-shell__pagination {
    display: flex;
    justify-content: flex-end;
    padding: var(--af-space-3) 0 0;
  }
</style>
