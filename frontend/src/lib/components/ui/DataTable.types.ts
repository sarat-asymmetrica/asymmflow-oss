/**
 * DataTable Component Type Definitions
 * Enterprise-grade table with Bloomberg density + Apple polish
 */

export type Alignment = 'left' | 'center' | 'right';

export type ColumnType = 'text' | 'number' | 'currency' | 'date' | 'status' | 'actions';

export type SortDirection = 'asc' | 'desc';

export interface Column<T = any> {
  /**
   * The key/path to access the value in the data object
   * Supports nested paths like 'customer.name'
   */
  key: string;

  /**
   * Display label for the column header
   */
  label: string;

  /**
   * Text alignment for the column
   * @default 'left' (auto 'right' for number/currency types)
   */
  align?: Alignment;

  /**
   * Fixed width for the column (e.g., '120px', '15%')
   */
  width?: string;

  /**
   * Enable sorting for this column
   * @default false
   */
  sortable?: boolean;

  /**
   * Column type for automatic formatting
   * @default 'text'
   */
  type?: ColumnType;

  /**
   * Custom render function for complex cell content
   * Returns HTML string (will be rendered with @html)
   */
  render?: (row: T) => string | number;

  /**
   * Custom format function for the value
   * Takes precedence over type-based formatting
   */
  format?: (value: any) => string;
}

export interface DataTableProps<T = any> {
  /**
   * Array of column definitions
   */
  columns: Column<T>[];

  /**
   * Array of data objects to display
   */
  data: T[];

  /**
   * Show loading skeleton
   * @default false
   */
  loading?: boolean;

  /**
   * Message to display when data is empty
   * @default 'No data available'
   */
  emptyMessage?: string;

  /**
   * Callback when a row is clicked
   */
  onRowClick?: (row: T) => void;

  /**
   * ID of the currently selected row (for highlighting)
   */
  selectedId?: string | number;

  /**
   * Enable sticky header that stays visible while scrolling
   * @default true
   */
  stickyHeader?: boolean;

  /**
   * Use compact mode with reduced padding and row height
   * @default false
   */
  compact?: boolean;

  /**
   * Maximum height of the table before scrolling
   * @default '600px'
   */
  maxHeight?: string;

  /**
   * Show border around the table
   * @default true
   */
  showBorder?: boolean;

  /**
   * The field name to use as the unique key for rows
   * Used for selection matching
   * @default 'id'
   */
  keyField?: string;
}

export interface SortEvent {
  key: string;
  direction: SortDirection;
}

export interface RowClickEvent<T = any> {
  row: T;
  index: number;
}

/**
 * Example usage:
 *
 * ```typescript
 * import type { Column } from '$lib/components/ui/DataTable.types';
 *
 * interface Invoice {
 *   id: string;
 *   number: string;
 *   amount: number;
 *   status: 'approved' | 'pending' | 'rejected';
 * }
 *
 * const columns: Column<Invoice>[] = [
 *   { key: 'number', label: 'Invoice #', sortable: true },
 *   { key: 'amount', label: 'Amount', type: 'currency', sortable: true },
 *   { key: 'status', label: 'Status', type: 'status' }
 * ];
 * ```
 */
