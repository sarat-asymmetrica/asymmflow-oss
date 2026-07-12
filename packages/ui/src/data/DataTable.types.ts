/**
 * DataTable type contract — @asymmflow/ui
 *
 * Generic over T (the row shape). Typed column model with Snippet-based
 * custom cell rendering, sort state, and selection helpers.
 *
 * Constitution: packages/DESIGN_CONSTITUTION.md §4a (tabular numerals always)
 */

import type { Snippet } from 'svelte';

// ─── Alignment ──────────────────────────────────────────────────────────────

export type Alignment = 'left' | 'center' | 'right';

// ─── Sort ───────────────────────────────────────────────────────────────────

export type SortDirection = 'asc' | 'desc';

export interface SortState {
  key: string;
  direction: SortDirection;
}

// ─── Column ─────────────────────────────────────────────────────────────────

/**
 * Column definition, generic over the row type T.
 *
 * - `numeric: true` applies `.af-numeric` (tabular-nums lining-nums, §4a)
 *   and forces right-alignment unless `align` is explicitly overridden.
 * - `format` takes precedence over the default string coercion.
 * - `cell` is a Snippet<[CellContext<T>]>; when provided, it fully owns
 *   rendering for this column — format is ignored.
 */
export interface Column<T = unknown> {
  /** Key path into T; supports dot-notation for nested values, e.g. "customer.name". */
  key: string;

  /** Column header text. */
  header: string;

  /**
   * Text alignment.
   * Defaults to 'right' when `numeric` is true, 'left' otherwise.
   */
  align?: Alignment;

  /**
   * When true: applies .af-numeric (tabular numerals, §4a) and defaults
   * alignment to 'right'. Use for ALL financial figures.
   */
  numeric?: boolean;

  /** Fixed column width, e.g. "120px", "15%". */
  width?: string;

  /** Enables the sort chevron in the header and client-side sort. */
  sortable?: boolean;

  /**
   * Value formatter. Called with (rawValue, row).
   * Return a display string. Ignored when `cell` Snippet is provided.
   */
  format?: (value: unknown, row: T) => string;

  /**
   * Snippet-based custom cell renderer.
   * Receives a CellContext<T>. When present, `format` is not called.
   */
  cell?: Snippet<[CellContext<T>]>;
}

// ─── Cell context (passed into the cell Snippet) ─────────────────────────────

export interface CellContext<T = unknown> {
  column: Column<T>;
  row: T;
  /** The raw resolved value at column.key. */
  value: unknown;
  /** Display string via format() if defined, else String(value). */
  formatted: string;
}

// ─── DataTable props ─────────────────────────────────────────────────────────

export interface DataTableProps<T = unknown> {
  /** Column definitions. */
  columns: Column<T>[];

  /** Row data array. */
  data: T[];

  /**
   * Row identity function. Used for selection matching.
   * Defaults to row => (row as any).id
   */
  rowId?: (row: T) => string | number;

  /** Show loading skeleton rows instead of data. */
  loading?: boolean;

  /** Number of skeleton rows to show when loading. Default: 5. */
  loadingRows?: number;

  /** Snippet rendered when data is empty and not loading. */
  empty?: Snippet;

  /** Row click handler. When provided rows become focusable + keyboard-activatable. */
  onRowClick?: (row: T) => void;

  /**
   * Sort change callback (controlled sort mode).
   * When provided, sort state is controlled externally — internal sort is disabled.
   */
  onSortChange?: (sort: SortState | null) => void;

  /**
   * Initial sort state (uncontrolled mode).
   * In uncontrolled mode, client-side sort is applied automatically.
   */
  initialSort?: SortState;

  /**
   * Selection — bindable Set of row ids (as returned by rowId).
   * Renders a leading checkbox column when provided.
   */
  selected?: Set<string | number>;

  /**
   * Snippet: row-level action buttons revealed on row hover.
   * Receives the row T. Rendered in an absolutely-positioned reveal layer.
   */
  rowActions?: Snippet<[T]>;

  /** Max height before the table scrolls. Default: '520px'. */
  maxHeight?: string;

  /**
   * aria-label for the <table> element.
   * Required when the table has no visible caption.
   */
  label: string;

  /** Total row count for aria-rowcount (may differ from data.length in paginated tables). */
  rowCount?: number;

  /**
   * Global cell renderer Snippet — a table-level fallback.
   * Called for any column that does NOT have its own `column.cell` Snippet.
   * Useful when you want to override rendering for one or two columns without
   * putting Snippets inside the column definition (which requires them to be
   * defined in the template, not script).
   *
   * @example
   * {#snippet cell(ctx)}
   *   {#if ctx.column.key === 'status'}
   *     <StatusBadge value={ctx.value} />
   *   {:else}
   *     {ctx.formatted}
   *   {/if}
   * {/snippet}
   * <DataTable {cell} ... />
   */
  cell?: Snippet<[CellContext<T>]>;
}
