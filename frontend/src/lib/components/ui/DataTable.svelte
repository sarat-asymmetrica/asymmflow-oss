<script lang="ts">
  import { run } from 'svelte/legacy';

  /**
   * Enterprise DataTable Component
   * Philosophy: Bloomberg-style density, Apple-level polish
   *
   * Features:
   * - Sticky header for long scrolling
   * - Sortable columns with visual indicators
   * - Row selection with keyboard navigation
   * - Multiple column types (text, number, currency, date, status, actions)
   * - Loading states with skeleton UI
   * - Empty state messaging
   * - Full accessibility (ARIA, keyboard)
   * - 100+ row performance optimized
   */

  import { createEventDispatcher, onMount } from 'svelte';
  import type { Snippet } from 'svelte';

  const dispatch = createEventDispatcher();

  // Type definitions for columns
  type Alignment = 'left' | 'center' | 'right';

  type ColumnType = 'text' | 'number' | 'currency' | 'date' | 'status' | 'actions';

  interface Column<T = any> {
    key: string;
    label: string;
    align?: Alignment;
    width?: string;
    sortable?: boolean;
    type?: ColumnType;
    render?: (row: T) => string | number;
    format?: (value: any) => string;
  }

  
  interface Props {
    // Props
    columns?: Column[];
    data?: any[];
    loading?: boolean;
    emptyMessage?: string;
    onRowClick?: ((row: any) => void) | undefined;
    selectedId?: string | undefined;
    stickyHeader?: boolean;
    compact?: boolean;
    maxHeight?: string;
    showBorder?: boolean;
    striped?: boolean; // DEPRECATED - not used per design system
    keyField?: string;
    alternateRows?: boolean; // Optional alternating row colors for readability
    rowClass?: ((row: any) => string) | undefined; // Custom row class function
    cell?: Snippet<[{ column: Column; row: any; value: any }]>;
  }

  let {
    columns = [],
    data = [],
    loading = false,
    emptyMessage = 'No data available',
    onRowClick = undefined,
    selectedId = undefined,
    stickyHeader = true,
    compact = false,
    maxHeight = '600px',
    showBorder = true,
    striped = false,
    keyField = 'id',
    alternateRows = false,
    rowClass = undefined,
    cell
  }: Props = $props();

  // Internal state
  let sortKey: string | null = $state(null);
  let sortDirection: 'asc' | 'desc' = $state('asc');
  let focusedRowIndex: number = -1;
  let tableElement: HTMLTableElement = $state();

  // Generate unique ID for accessibility
  const tableId = `table-${Math.random().toString(36).substr(2, 9)}`;


  // Helper: Get nested object value
  function getNestedValue(obj: any, path: string): any {
    return path.split('.').reduce((curr, key) => curr?.[key], obj);
  }

  // Helper: Format value based on column type
  function formatValue(value: any, column: Column): string {
    if (value === null || value === undefined) return '—';

    // Custom formatter has priority
    if (column.format) return column.format(value);

    switch (column.type) {
      case 'currency':
        return formatCurrency(value);
      case 'number':
        return formatNumber(value);
      case 'date':
        return formatDate(value);
      default:
        return String(value);
    }
  }

  // Format: Currency (BHD with 3 decimals)
  function formatCurrency(value: number): string {
    return `${value.toLocaleString('en-BH', {
      minimumFractionDigits: 3,
      maximumFractionDigits: 3
    })} BHD`;
  }

  // Format: Number with thousands separator
  function formatNumber(value: number): string {
    return value.toLocaleString('en-US');
  }

  // Format: Date
  function formatDate(value: string | Date): string {
    const date = typeof value === 'string' ? new Date(value) : value;
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    });
  }

  // Handle: Column header click (sorting)
  function handleSort(column: Column) {
    if (!column.sortable) return;

    if (sortKey === column.key) {
      sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
    } else {
      sortKey = column.key;
      sortDirection = 'asc';
    }

    dispatch('sort', { key: sortKey, direction: sortDirection });
  }

  // Handle: Row click
  function handleRowClick(row: any, index: number, event?: MouseEvent) {
    focusedRowIndex = index;

    if (onRowClick) {
      onRowClick(row);
    }

    dispatch('rowClick', { row, index, event });
  }

  // Handle: Keyboard navigation
  function handleKeyDown(event: KeyboardEvent, row: any, index: number) {
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault();
        if (index < sortedData.length - 1) {
          focusedRowIndex = index + 1;
          focusRow(focusedRowIndex);
        }
        break;
      case 'ArrowUp':
        event.preventDefault();
        if (index > 0) {
          focusedRowIndex = index - 1;
          focusRow(focusedRowIndex);
        }
        break;
      case 'Enter':
      case ' ':
        event.preventDefault();
        handleRowClick(row, index);
        break;
    }
  }

  // Helper: Focus a specific row
  function focusRow(index: number) {
    const rows = tableElement?.querySelectorAll('tbody tr');
    if (rows && rows[index]) {
      (rows[index] as HTMLElement).focus();
    }
  }

  // Computed: Check if row is selected
  function isRowSelected(row: any): boolean {
    if (!selectedId) return false;
    return getNestedValue(row, keyField) === selectedId;
  }

  // Get cell alignment class
  function getAlignClass(column: Column): string {
    const align = column.align || (column.type === 'number' || column.type === 'currency' ? 'right' : 'left');
    return `text-${align}`;
  }

  onMount(() => {
    // Any initialization if needed
  });
  run(() => {
    striped;
  });
  // Computed: Sorted data
  let sortedData = $derived((() => {
    if (!sortKey || !data.length) return data;

    return [...data].sort((a, b) => {
      const aVal = getNestedValue(a, sortKey);
      const bVal = getNestedValue(b, sortKey);

      if (aVal === bVal) return 0;

      const comparison = aVal > bVal ? 1 : -1;
      return sortDirection === 'asc' ? comparison : -comparison;
    });
  })());
</script>

<div
  class="table-container"
  class:compact
  class:bordered={showBorder}
>
  <div class="table-scroll-wrapper" style="max-height: {maxHeight};">
    <table
      bind:this={tableElement}
      class="data-table"
      class:sticky-header={stickyHeader}
      aria-label="Data table"
      id={tableId}
    >
      <thead>
        <tr>
          {#each columns as column}
            <th
              role="columnheader"
              scope="col"
              class={getAlignClass(column)}
              class:sortable={column.sortable}
              class:sorted={sortKey === column.key}
              style={column.width ? `width: ${column.width};` : ''}
              onclick={() => handleSort(column)}
              onkeydown={(e) => {
                if (column.sortable && (e.key === 'Enter' || e.key === ' ')) {
                  e.preventDefault();
                  handleSort(column);
                }
              }}
              tabindex={column.sortable ? 0 : -1}
            >
              <div class="th-content">
                <span>{column.label}</span>
                {#if column.sortable}
                  <span class="sort-indicator" aria-hidden="true">
                    {#if sortKey === column.key}
                      {sortDirection === 'asc' ? '\u2191' : '\u2193'}
                    {:else}
                      <span class="sort-icon-neutral">{'\u2195'}</span>
                    {/if}
                  </span>
                {/if}
              </div>
            </th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#if loading}
          <!-- Loading skeleton -->
          {#each Array(5) as _, i}
            <tr class="skeleton-row">
              {#each columns as column}
                <td class={getAlignClass(column)}>
                  <div class="skeleton-bar"></div>
                </td>
              {/each}
            </tr>
          {/each}
        {:else if sortedData.length === 0}
          <!-- Empty state -->
          <tr>
            <td colspan={columns.length} class="empty-state">
              <div class="empty-content">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                </svg>
                <p>{emptyMessage}</p>
              </div>
            </td>
          </tr>
        {:else}
          <!-- Data rows -->
          {#each sortedData as row, index}
            <tr
              class="table-row {rowClass ? rowClass(row) : ''}"
              class:selected={isRowSelected(row)}
              class:clickable={!!onRowClick}
              class:alternate={alternateRows && index % 2 === 1}
              tabindex={onRowClick ? 0 : -1}
              onclick={(e) => handleRowClick(row, index, e)}
              onkeydown={(e) => handleKeyDown(e, row, index)}
              aria-selected={isRowSelected(row)}
            >
              {#each columns as column}
                {@const value = getNestedValue(row, column.key)}
                <td
                  role="cell"
                  class={getAlignClass(column)}
                  class:cell-status={column.type === 'status'}
                  class:cell-actions={column.type === 'actions'}
                >
                  {#if cell}
                    {@render cell({ column, row, value })}
                  {:else if column.render}
                    {@html column.render(row)}
                  {:else if column.type === 'status'}
                    <span class="status-badge status-{value?.toLowerCase()}">
                      {value}
                    </span>
                  {:else}
                    {formatValue(value, column)}
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
  /* ========================================
     TABLE CONTAINER
     ======================================== */
  .table-container {
    background: var(--surface);
    border-radius: var(--border-radius);
    overflow: hidden;
    position: relative;
  }

  .table-container.bordered {
    border: 1px solid var(--border);
  }

  .table-scroll-wrapper {
    overflow: auto;
    width: 100%;
    height: 100%;
  }

  /* ========================================
     TABLE STRUCTURE
     ======================================== */
  .data-table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--table-text-size);
    table-layout: auto;
  }

  /* ========================================
     TABLE HEADER
     ======================================== */
  .data-table thead {
    background: var(--surface);
    z-index: var(--z-sticky);
  }

  .data-table.sticky-header thead {
    position: sticky;
    top: 0;
    background: var(--surface);
  }

  .data-table.sticky-header thead::after {
    content: '';
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 1px;
    background: var(--border);
  }

  .data-table th {
    padding: 12px;
    text-align: left;
    font-weight: var(--font-weight-semibold);
    color: var(--text-secondary);
    font-size: var(--label-size);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid var(--border);
    background: var(--surface);
    white-space: nowrap;
    user-select: none;
  }

  .data-table th.sortable {
    cursor: pointer;
    transition: color var(--transition-fast);
  }

  .data-table th.sortable:hover {
    color: var(--text-primary);
  }

  .data-table th.sortable:focus {
    outline: 2px solid var(--brand-indigo);
    outline-offset: -2px;
  }

  .data-table th.sorted {
    color: var(--brand-indigo);
  }

  .th-content {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .sort-indicator {
    font-size: 14px;
    opacity: 1;
    color: var(--brand-indigo);
  }

  .sort-icon-neutral {
    opacity: 0.3;
    font-size: 12px;
  }

  /* Alignment classes */
  .text-left {
    text-align: left;
    justify-content: flex-start;
  }

  .text-center {
    text-align: center;
    justify-content: center;
  }

  .text-right {
    text-align: right;
    justify-content: flex-end;
  }

  /* ========================================
     TABLE BODY
     ======================================== */
  .data-table tbody {
    background: var(--surface);
  }

  .data-table td {
    padding: 12px;
    color: var(--text-primary);
    border-bottom: 1px solid var(--border);
    vertical-align: middle;
  }

  /* ========================================
     TABLE ROWS
     ======================================== */
  .table-row {
    height: var(--table-row-height);
    transition: background var(--transition-fast);
  }

  /* Hover state - 4% indigo tint (Bloomberg style) */
  .table-row:hover {
    background: var(--interactive-hover);
  }

  /* Clickable rows */
  .table-row.clickable {
    cursor: pointer;
  }

  /* Selected row */
  .table-row.selected {
    background: var(--interactive-pressed);
  }

  .table-row.selected:hover {
    background: rgba(47, 45, 255, 0.10);
  }

  /* Focus state for keyboard navigation */
  .table-row:focus {
    outline: 2px solid var(--brand-indigo);
    outline-offset: -2px;
    position: relative;
  }

  /* Optional alternating rows for readability */
  .table-row.alternate {
    background: rgba(0, 0, 0, 0.015);
  }

  .table-row.alternate:hover {
    background: var(--interactive-hover);
  }

  /* ========================================
     COMPACT MODE
     ======================================== */
  .table-container.compact .data-table th,
  .table-container.compact .data-table td {
    padding: 8px;
  }

  .table-container.compact .table-row {
    height: 36px;
  }

  /* ========================================
     CELL TYPES
     ======================================== */

  /* Status badges */
  .cell-status {
    padding-top: 8px;
    padding-bottom: 8px;
  }

  .status-badge {
    display: inline-block;
    padding: 4px 10px;
    border-radius: var(--border-radius-sm);
    font-size: 11px;
    font-weight: var(--font-weight-medium);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    white-space: nowrap;
  }

  .status-badge.status-active,
  .status-badge.status-open,
  .status-badge.status-approved {
    background: rgba(16, 185, 129, 0.1);
    color: #10B981;
  }

  .status-badge.status-pending,
  .status-badge.status-draft {
    background: rgba(245, 158, 11, 0.1);
    color: #F59E0B;
  }

  .status-badge.status-closed,
  .status-badge.status-rejected,
  .status-badge.status-cancelled {
    background: rgba(239, 68, 68, 0.1);
    color: #EF4444;
  }

  .status-badge.status-inactive {
    background: rgba(140, 144, 168, 0.1);
    color: var(--text-muted);
  }

  /* Actions cell */
  .cell-actions {
    white-space: nowrap;
  }

  /* ========================================
     LOADING STATE
     ======================================== */
  .skeleton-row {
    height: var(--table-row-height);
  }

  .skeleton-bar {
    height: 12px;
    background: linear-gradient(
      90deg,
      var(--border) 0%,
      var(--surface-elevated) 50%,
      var(--border) 100%
    );
    background-size: 200% 100%;
    border-radius: 4px;
    animation: shimmer 1.5s infinite;
    max-width: 200px;
  }

  @keyframes shimmer {
    0% {
      background-position: -200% 0;
    }
    100% {
      background-position: 200% 0;
    }
  }

  /* ========================================
     EMPTY STATE
     ======================================== */
  .empty-state {
    padding: 48px 24px !important;
    text-align: center;
    border-bottom: none !important;
  }

  .empty-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: var(--text-muted);
  }

  .empty-content svg {
    opacity: 0.5;
  }

  .empty-content p {
    margin: 0;
    font-size: 14px;
    font-style: italic;
  }

  /* ========================================
     RESPONSIVE BEHAVIOR
     ======================================== */
  @media (max-width: 768px) {
    .table-scroll-wrapper {
      overflow-x: auto;
      -webkit-overflow-scrolling: touch;
    }

    .data-table {
      min-width: 600px;
    }

    .data-table th,
    .data-table td {
      padding: 8px;
    }
  }

  /* ========================================
     ACCESSIBILITY
     ======================================== */

  /* High contrast mode support */
  @media (prefers-contrast: high) {
    .table-row:hover {
      background: var(--interactive-pressed);
    }

    .table-row.selected {
      outline: 2px solid var(--brand-indigo);
      outline-offset: -2px;
    }
  }

  /* Reduced motion support */
  @media (prefers-reduced-motion: reduce) {
    .table-row,
    .data-table th.sortable,
    .skeleton-bar {
      transition: none;
      animation: none;
    }
  }
</style>
