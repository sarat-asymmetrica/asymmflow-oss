<script lang="ts" generics="T">
  /**
   * DataTable — @asymmflow/ui
   *
   * The flagship data surface. Calm center of the ERP.
   * Design law: packages/DESIGN_CONSTITUTION.md
   *
   * Features:
   * - Sticky frosted-glass header (--af-glass-bg + blur)
   * - .af-label uppercase headers with sort chevrons (aria-sort)
   * - Row height: var(--af-row-height) — density-aware automatically
   * - Horizontal dividers only — zero vertical borders
   * - Row hover: --af-tint wash + row-actions Snippet reveal
   * - Numeric columns: .af-numeric, right-aligned, always (§4a)
   * - Client-side sort (uncontrolled) or controlled via onSortChange
   * - Selection: checkbox column, tri-state header, bindable selected Set
   * - Loading: inline shimmer skeleton rows
   * - Empty: calm default or custom Snippet
   * - Row click + keyboard navigation (Enter activates)
   * - Entrance: rows stagger in once on data arrival (first 12 rows)
   * - Full ARIA: aria-sort, aria-rowcount, aria-selected, th scope
   * - prefers-reduced-motion compliant
   */

  import type {
    Column,
    SortState,
    CellContext,
    DataTableProps,
    Alignment,
  } from './DataTable.types.js';

  type Props = DataTableProps<T>;

  let {
    columns,
    data = [],
    rowId = (row) => (row as Record<string, unknown>).id as string | number,
    loading = false,
    loadingRows = 5,
    empty,
    cell: globalCell,
    onRowClick,
    onSortChange,
    initialSort,
    selected = $bindable(undefined),
    rowActions,
    maxHeight = '520px',
    label,
    rowCount,
  }: Props = $props();

  // ─── Sort state ─────────────────────────────────────────────────────────────

  // svelte-ignore state_referenced_locally -- initialSort is intentionally read once to seed local sort state; subsequent changes are driven by user interaction, not the prop.
  let internalSort = $state<SortState | null>(initialSort ?? null);

  // In controlled mode, onSortChange is the authority. In uncontrolled, we sort locally.
  const isControlled = $derived(typeof onSortChange === 'function');
  const currentSort = $derived(isControlled ? null : internalSort);

  function handleSortClick(col: Column<T>) {
    if (!col.sortable) return;
    let next: SortState | null;

    if (currentSort?.key === col.key && !isControlled) {
      // Cycle: asc → desc → none
      if (internalSort!.direction === 'asc') {
        internalSort = { key: col.key, direction: 'desc' };
        next = internalSort;
      } else {
        internalSort = null;
        next = null;
      }
    } else {
      if (!isControlled) internalSort = { key: col.key, direction: 'asc' };
      next = { key: col.key, direction: 'asc' };
    }

    if (isControlled && onSortChange) onSortChange(next);
  }

  function ariaSortAttr(col: Column<T>): 'ascending' | 'descending' | 'none' | undefined {
    if (!col.sortable) return undefined;
    const s = isControlled ? null : internalSort;
    if (!s || s.key !== col.key) return 'none';
    return s.direction === 'asc' ? 'ascending' : 'descending';
  }

  // ─── Data processing ─────────────────────────────────────────────────────

  function resolveValue(row: T, key: string): unknown {
    return key.split('.').reduce<unknown>((acc, k) => {
      if (acc == null) return undefined;
      return (acc as Record<string, unknown>)[k];
    }, row);
  }

  function formatCell(col: Column<T>, row: T): string {
    const val = resolveValue(row, col.key);
    if (val == null) return '—';
    if (col.format) return col.format(val, row);
    return String(val);
  }

  function cellContext(col: Column<T>, row: T): CellContext<T> {
    const value = resolveValue(row, col.key);
    return {
      column: col,
      row,
      value,
      formatted: formatCell(col, row),
    };
  }

  function colAlign(col: Column<T>): Alignment {
    if (col.align) return col.align;
    return col.numeric ? 'right' : 'left';
  }

  // ─── Sorted data ──────────────────────────────────────────────────────────

  let sortedData = $derived.by(() => {
    const sort = isControlled ? null : internalSort;
    if (!sort || !data.length) return data;

    return [...data].sort((a, b) => {
      const av = resolveValue(a, sort.key);
      const bv = resolveValue(b, sort.key);
      if (av === bv) return 0;
      const cmp = av! > bv! ? 1 : -1;
      return sort.direction === 'asc' ? cmp : -cmp;
    });
  });

  // ─── Selection ────────────────────────────────────────────────────────────

  const hasSelection = $derived(selected !== undefined);

  const allIds = $derived(data.map((r) => rowId(r)));

  const selectionState = $derived.by((): 'none' | 'some' | 'all' => {
    if (!selected || selected.size === 0) return 'none';
    if (allIds.every((id) => selected!.has(id))) return 'all';
    return 'some';
  });

  function toggleAll() {
    if (!selected) return;
    if (selectionState === 'all') {
      allIds.forEach((id) => selected!.delete(id));
    } else {
      allIds.forEach((id) => selected!.add(id));
    }
    // Trigger Svelte reactivity by replacing the set
    selected = new Set(selected);
  }

  function toggleRow(row: T) {
    if (!selected) return;
    const id = rowId(row);
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selected = next;
  }

  function isSelected(row: T): boolean {
    return selected ? selected.has(rowId(row)) : false;
  }

  // ─── Keyboard navigation ─────────────────────────────────────────────────

  let tbodyEl = $state<HTMLTableSectionElement | undefined>(undefined);

  function handleRowKeydown(e: KeyboardEvent, row: T, idx: number) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      onRowClick?.(row);
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      const rows = tbodyEl?.querySelectorAll<HTMLElement>('tr[data-clickable]');
      rows?.[idx + 1]?.focus();
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      const rows = tbodyEl?.querySelectorAll<HTMLElement>('tr[data-clickable]');
      rows?.[idx - 1]?.focus();
    }
  }

  // ─── Stagger tracking ────────────────────────────────────────────────────

  // We only stagger on first mount of each unique data array reference.
  // After that (sort/filter changes) rows appear without stagger.
  let staggerGeneration = $state(0);
  let prevDataRef: T[] | null = null;

  $effect(() => {
    if (data !== prevDataRef && data.length > 0) {
      staggerGeneration++;
      prevDataRef = data;
    }
  });

  const STAGGER_MAX_ROWS = 12;

  // ─── Skeleton widths (random-ish but stable) ──────────────────────────────

  const SKEL_WIDTHS = ['55%', '72%', '43%', '88%', '61%', '79%', '50%', '67%'];
  function skelWidth(rowIdx: number, colIdx: number): string {
    return SKEL_WIDTHS[(rowIdx * 3 + colIdx) % SKEL_WIDTHS.length];
  }

  // ─── Effective column list (with optional checkbox prefix) ────────────────

  const effectiveCols = $derived<Column<T>[]>(
    hasSelection
      ? [{ key: '__check__', header: '', width: '44px' } as Column<T>, ...columns]
      : columns
  );

  // Unique table ID for accessibility
  const tableId = $props.id();

  // ─── Action: set indeterminate DOM property (not an HTML attribute) ───────

  function setIndeterminate(el: HTMLInputElement, indeterminate: boolean) {
    el.indeterminate = indeterminate;
    return {
      update(v: boolean) { el.indeterminate = v; },
    };
  }
</script>

<div class="af-table-container" style:--max-h={maxHeight}>
  <div class="af-table-scroll">
    <table
      id={tableId}
      class="af-table"
      aria-label={label}
      aria-rowcount={rowCount ?? (loading ? undefined : data.length)}
    >
      <!-- ── Header ──────────────────────────────────────────────────────── -->
      <thead class="af-table-head">
        <tr>
          {#each effectiveCols as col}
            {#if col.key === '__check__'}
              <!-- Tri-state checkbox header -->
              <th scope="col" class="af-th af-th--check" style:width={col.width}>
                <label class="af-check-wrap" aria-label="Select all rows">
                  <input
                    type="checkbox"
                    class="af-check"
                    checked={selectionState === 'all'}
                    onchange={toggleAll}
                    aria-label="Select all"
                    use:setIndeterminate={selectionState === 'some'}
                  />
                </label>
              </th>
            {:else}
              <th
                scope="col"
                class="af-th af-label"
                class:af-th--sortable={col.sortable}
                class:af-th--sorted={!isControlled && internalSort?.key === col.key}
                class:af-th--right={colAlign(col) === 'right'}
                class:af-th--center={colAlign(col) === 'center'}
                style:width={col.width}
                aria-sort={ariaSortAttr(col)}
                tabindex={col.sortable ? 0 : undefined}
                role={col.sortable ? 'columnheader' : undefined}
                onclick={() => handleSortClick(col)}
                onkeydown={(e) => {
                  if (col.sortable && (e.key === 'Enter' || e.key === ' ')) {
                    e.preventDefault();
                    handleSortClick(col);
                  }
                }}
              >
                <span class="af-th-inner">
                  <span class="af-th-label">{col.header}</span>
                  {#if col.sortable}
                    <span class="af-sort-chevron" aria-hidden="true">
                      {#if !isControlled && internalSort?.key === col.key && internalSort.direction === 'asc'}
                        <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                          <path d="M5 2L8.5 7H1.5L5 2Z" fill="currentColor"/>
                        </svg>
                      {:else if !isControlled && internalSort?.key === col.key && internalSort.direction === 'desc'}
                        <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                          <path d="M5 8L1.5 3H8.5L5 8Z" fill="currentColor"/>
                        </svg>
                      {:else}
                        <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                          <path d="M5 1.5L7.5 4.5H2.5L5 1.5Z" fill="currentColor" opacity="0.35"/>
                          <path d="M5 8.5L2.5 5.5H7.5L5 8.5Z" fill="currentColor" opacity="0.35"/>
                        </svg>
                      {/if}
                    </span>
                  {/if}
                </span>
              </th>
            {/if}
          {/each}
        </tr>
      </thead>

      <!-- ── Body ───────────────────────────────────────────────────────── -->
      <tbody bind:this={tbodyEl}>
        {#if loading}
          <!-- Skeleton rows -->
          {#each Array(loadingRows) as _, ri}
            <tr class="af-tr af-tr--skeleton">
              {#if hasSelection}
                <td class="af-td af-td--check">
                  <div class="af-skel af-skel--check"></div>
                </td>
              {/if}
              {#each columns as col, ci}
                <td class="af-td" class:af-td--right={colAlign(col) === 'right'} class:af-td--center={colAlign(col) === 'center'}>
                  <div class="af-skel" style:width={skelWidth(ri, ci)}></div>
                </td>
              {/each}
            </tr>
          {/each}

        {:else if sortedData.length === 0}
          <!-- Empty state -->
          <tr class="af-tr af-tr--empty">
            <td colspan={effectiveCols.length} class="af-td--empty-cell">
              {#if empty}
                {@render empty()}
              {:else}
                <div class="af-empty-default">
                  <svg class="af-empty-icon" width="32" height="32" viewBox="0 0 32 32" fill="none" aria-hidden="true">
                    <rect x="6" y="8" width="20" height="18" rx="2" stroke="currentColor" stroke-width="1.5"/>
                    <path d="M11 13h10M11 17h6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                  </svg>
                  <span class="af-empty-label">No records found</span>
                </div>
              {/if}
            </td>
          </tr>

        {:else}
          <!-- Data rows -->
          {#each sortedData as row, ri}
            {@const rowIdx = ri}
            {@const staggerIndex = staggerGeneration > 0 && ri < STAGGER_MAX_ROWS
              ? ri
              : 0}
            {@const isClickable = !!onRowClick}
            {@const rowSelected = isSelected(row)}

            <tr
              class="af-tr"
              class:af-tr--clickable={isClickable}
              class:af-tr--selected={rowSelected}
              class:af-tr--has-actions={!!rowActions}
              aria-selected={hasSelection ? rowSelected : undefined}
              tabindex={isClickable ? 0 : undefined}
              data-clickable={isClickable ? '' : undefined}
              style:--stagger-index={staggerIndex}
              onclick={() => isClickable && onRowClick?.(row)}
              onkeydown={(e) => isClickable && handleRowKeydown(e, row, rowIdx)}
            >
              {#if hasSelection}
                <td class="af-td af-td--check">
                  <label class="af-check-wrap" aria-label="Select row">
                    <input
                      type="checkbox"
                      class="af-check"
                      checked={rowSelected}
                      onchange={() => toggleRow(row)}
                      onclick={(e) => e.stopPropagation()}
                    />
                  </label>
                </td>
              {/if}

              {#each columns as col, ci}
                <td
                  class="af-td"
                  class:af-numeric={col.numeric}
                  class:af-td--right={colAlign(col) === 'right'}
                  class:af-td--center={colAlign(col) === 'center'}
                  class:af-td--last={ci === columns.length - 1}
                >
                  {#if col.cell}
                    {@render col.cell(cellContext(col, row))}
                  {:else if globalCell}
                    {@render globalCell(cellContext(col, row))}
                  {:else}
                    {formatCell(col, row)}
                  {/if}
                  <!-- Row actions: overlay in last cell, revealed on hover -->
                  {#if ci === columns.length - 1 && rowActions}
                    <div class="af-row-actions" aria-hidden="true">
                      {@render rowActions(row)}
                    </div>
                  {/if}
                </td>
              {/each}
            </tr>
          {/each}
        {/if}
      </tbody>
    </table>
  </div>
</div>

<style>
  /* ── Container ─────────────────────────────────────────────────────────── */
  .af-table-container {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    overflow: hidden;
    position: relative;
  }

  .af-table-scroll {
    overflow: auto;
    max-height: var(--max-h, 520px);
    width: 100%;
    /* Custom scrollbar — light and unobtrusive */
    scrollbar-width: thin;
    scrollbar-color: var(--af-border-strong) transparent;
  }

  .af-table-scroll::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }

  .af-table-scroll::-webkit-scrollbar-track {
    background: transparent;
  }

  .af-table-scroll::-webkit-scrollbar-thumb {
    background: var(--af-border-strong);
    border-radius: var(--af-radius-pill);
  }

  /* ── Table ─────────────────────────────────────────────────────────────── */
  .af-table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--af-text-sm);
    font-family: var(--af-font-body);
    table-layout: auto;
  }

  /* ── Header ────────────────────────────────────────────────────────────── */
  .af-table-head {
    position: sticky;
    top: 0;
    z-index: var(--af-z-sticky);
    background: var(--af-glass-bg);
    backdrop-filter: var(--af-glass-blur);
    -webkit-backdrop-filter: var(--af-glass-blur);
  }

  .af-th {
    padding: var(--af-space-2) var(--af-space-3);
    /* .af-label styles (from base.css) — 11px, 600, uppercase, 0.08em tracking */
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
    color: var(--af-text-secondary);
    white-space: nowrap;
    user-select: none;
    border-bottom: 1px solid var(--af-border);
    background: transparent;
    /* Logical alignment — follows reading direction (start = left in LTR, right in RTL). */
    text-align: start;
    vertical-align: middle;
  }

  .af-th--right { text-align: right; }
  .af-th--center { text-align: center; }

  .af-th--check {
    padding: var(--af-space-2);
    width: 44px;
  }

  .af-th--sortable {
    cursor: pointer;
    transition: color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-th--sortable:hover {
    color: var(--af-text);
  }

  .af-th--sorted {
    color: var(--af-text);
  }

  .af-th--sortable:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: -2px;
  }

  .af-th-inner {
    display: inline-flex;
    align-items: center;
    gap: var(--af-space-1);
    /* Stay within the cell so the label can ellipsize instead of spilling
       over the sort chevron / neighbouring column when the column narrows. */
    max-width: 100%;
    min-width: 0;
  }

  /* Label truncates; the sort chevron is never clipped (flex-shrink: 0). */
  .af-th-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }

  /* Right-align: flip inner so chevron leads number columns */
  .af-th--right .af-th-inner {
    flex-direction: row-reverse;
  }

  .af-sort-chevron {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    color: var(--af-text-muted);
    transition: color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-th--sorted .af-sort-chevron {
    color: var(--af-accent);
  }

  /* ── Row ───────────────────────────────────────────────────────────────── */
  .af-tr {
    height: var(--af-row-height);
    position: relative;
  }

  /* Stagger entrance animation — first mount only */
  @keyframes af-row-in {
    from {
      opacity: 0;
      transform: translateY(6px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .af-tr:not(.af-tr--skeleton):not(.af-tr--empty) {
    animation: af-row-in var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
    /* Delay derives from the stagger token — no hardcoded ms in JS (§4b). */
    animation-delay: calc(var(--stagger-index, 0) * var(--af-motion-stagger));
  }

  .af-tr--clickable {
    cursor: pointer;
  }

  .af-tr--clickable:hover {
    background: var(--af-tint);
  }

  .af-tr--clickable:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: -2px;
    z-index: 1;
  }

  .af-tr--selected {
    background: var(--af-accent-tint) !important;
  }

  /* ── Row actions reveal ─────────────────────────────────────────────────── */
  /* Actions overlay sits inside the last <td> (position: relative) */
  .af-td--last {
    position: relative;
    /* Ensure text content doesn't overlap with actions area */
  }

  .af-row-actions {
    position: absolute;
    right: 0;
    top: 50%;
    transform: translateY(-50%) translateX(4px);
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
    padding-inline: var(--af-space-3);
    /* Gradient fades from the hover-tinted surface to transparent */
    background: linear-gradient(to left, var(--af-surface) 40%, transparent);
    opacity: 0;
    transition:
      opacity var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      transform var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
    pointer-events: none;
  }

  .af-tr:hover .af-row-actions,
  .af-tr:focus-within .af-row-actions {
    opacity: 1;
    transform: translateY(-50%) translateX(0);
    pointer-events: auto;
  }

  /* ── Cell ──────────────────────────────────────────────────────────────── */
  .af-td {
    padding: 0 var(--af-space-3);
    color: var(--af-text);
    border-bottom: 1px solid var(--af-border);
    vertical-align: middle;
    font-size: var(--af-text-sm);
    /* Ellipsis for overflow — tables are dense */
    max-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Last row: no bottom divider (container border is enough) */
  .af-tr:last-child .af-td {
    border-bottom: none;
  }

  .af-td--right { text-align: right; }
  .af-td--center { text-align: center; }
  .af-td--check {
    padding: 0 var(--af-space-2);
    width: 44px;
    text-align: center;
    overflow: visible;
    max-width: none;
  }


  /* ── Checkbox ──────────────────────────────────────────────────────────── */
  .af-check-wrap {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: var(--af-tap-min);
    min-height: var(--af-tap-min);
    cursor: pointer;
  }

  .af-check {
    width: 14px;
    height: 14px;
    accent-color: var(--af-accent);
    cursor: pointer;
    border-radius: var(--af-radius-sm);
  }

  /* ── Skeleton shimmer ──────────────────────────────────────────────────── */
  /* Highlight band swept with transform: translateX (GPU-composited, §4e),
     replacing the prior background-position paint loop. */
  @keyframes af-shimmer {
    to { transform: translateX(100%); }
  }

  .af-skel {
    position: relative;
    overflow: hidden;
    height: 11px;
    border-radius: var(--af-radius-sm);
    background: var(--af-surface-sunken);
  }

  .af-skel::after {
    content: '';
    position: absolute;
    inset: 0;
    background-image: linear-gradient(
      90deg,
      transparent 0%,
      var(--af-surface-raised) 50%,
      transparent 100%
    );
    transform: translateX(-100%);
    animation: af-shimmer var(--af-motion-shimmer) linear infinite;
  }

  .af-skel--check {
    width: 14px;
    height: 14px;
    border-radius: var(--af-radius-sm);
    margin: auto;
  }

  /* ── Empty state ───────────────────────────────────────────────────────── */
  .af-td--empty-cell {
    padding: var(--af-space-7) var(--af-space-4);
    text-align: center;
    border-bottom: none;
    max-width: none;
    overflow: visible;
    white-space: normal;
  }

  .af-empty-default {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--af-space-3);
    color: var(--af-text-muted);
  }

  .af-empty-icon {
    opacity: 0.45;
  }

  .af-empty-label {
    font-size: var(--af-text-sm);
    color: var(--af-text-muted);
  }

  /* ── Skeleton row ──────────────────────────────────────────────────────── */
  .af-tr--skeleton {
    animation: none;
  }

  /* ── Reduced motion ────────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .af-tr {
      animation: none;
    }
    .af-skel::after {
      animation: none;
      opacity: 0;
    }
    .af-row-actions {
      transition: none;
    }
  }
</style>
