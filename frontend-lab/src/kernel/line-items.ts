/* Line-items editor schema — the declarative contract for the LineItemsEditor
 * widget (kernel/widgets/LineItemsEditor.svelte). Same philosophy as
 * descriptor.ts's ColumnSpec, but EDIT-capable: each column declares how to
 * read a value off a row AND how to write one back. The widget is
 * presentation + edit-event plumbing ONLY — it never computes. All derivation
 * (line totals, sheet subtotals, journal balance) lives in the consuming
 * viewmodel (L5), so a costing waterfall and a journal voucher share one
 * repeater without the widget knowing either domain.
 *
 * Two independent recon passes (Accounting voucher + CostingSheet pricing)
 * converged on this column-config shape over the old hardcoded `mode=` switch,
 * so a third consumer (invoice/PO create lines) needs no widget change.
 */

import type { Component } from 'svelte'
import type { ContentClass } from './descriptor'
import type { Tone } from './tones'

/** Which input renders in an editable cell (or how a readonly cell formats). */
export type LineFieldKind =
  | 'text'
  | 'textarea'
  | 'number'
  | 'money' // number input, right-aligned, formatMoney in readonly/footer
  | 'percent'
  | 'select'
  | 'readonly' // computed/derived — displayed via renderCell, never editable

export interface LineColumn<Row> {
  key: string
  label: string
  kind: LineFieldKind
  /** Read the current value off the row (controlled input / readonly display). */
  value: (row: Row) => unknown
  /** Write a new value back onto the row. The widget calls this; it never
   * mutates Row itself. Omitted for kind:'readonly'. The raw input value
   * (string for text/select, number for numeric kinds) is passed — the column
   * decides coercion (e.g. clamp, default-on-blank), keeping domain rules in
   * the descriptor, not the widget. */
  set?: (row: Row, value: string | number) => void
  /** kind:'select' options, row-dependent (e.g. chart-of-accounts, currencies). */
  options?: (row: Row) => { value: string; label: string }[]
  /** Numeric <input step> — '0.001' for BHD money. */
  step?: string
  minWidth?: number
  /** Absorbs remaining width (at most one grower). */
  grow?: boolean
  align?: 'start' | 'end'
  /** Readonly/footer formatting class (defaults inferred from kind). */
  content?: ContentClass
  /** Per-row currency for money cells (default BHD). */
  currency?: (row: Row) => string
  /** Semantic colour for the cell — e.g. amber when a price was user-overridden,
   * red when a JV line has both debit and credit. Never a raw hex (L1). */
  tone?: (row: Row) => Tone
  /** Placeholder text (e.g. the computed suggested price on a manual-override cell). */
  placeholder?: (row: Row) => string
  /** Render as a full-width row BENEATH the main cell grid (CostingSheet's
   * long product code / detailed description). Wide columns are excluded from
   * the aligned grid + header. */
  wide?: boolean
  /** Fire set() on every keystroke ('input') vs on blur/change ('change').
   * Costing wants 'input' on percent fields for live recompute; qty is fine on
   * 'change'. Defaults: 'change' for number/money/percent/select, 'input' for
   * text/textarea. */
  eventGranularity?: 'input' | 'change'
  /** L4 ejection at cell granularity — same contract as ColumnSpec.cell, plus
   * an onInput to write back through the widget's event path. */
  cell?: Component<{ row: Row; onInput: (value: string | number) => void }>
}

/** A running-total cell in the footer, reduced over the live rows. */
export interface LineFooterCell<Row> {
  /** Align under this column key (optional — unaligned cells render inline). */
  colKey?: string
  label: string
  value: (rows: Row[]) => unknown
  content?: ContentClass
  currency?: string
  tone?: (rows: Row[]) => Tone
}

/** Optional balanced-check badge (journal voucher: Σdebit === Σcredit > 0).
 * This is a DISPLAY aid only — the real posting validates server-side and is
 * INTEG-gapped; the badge never authorizes anything (financial semantics stay
 * with the deterministic backend). */
export interface LineBalanceCheck<Row> {
  isBalanced: (rows: Row[]) => boolean
  balancedLabel?: string
  unbalancedLabel?: string
}

/* ---- pure helpers (unit-tested; reused by consumers' VMs) ---- */

/** Sum a numeric field over rows, coercing non-numbers to 0. */
export function sumField<Row>(rows: Row[], read: (row: Row) => unknown): number {
  return rows.reduce<number>((s, r) => s + (Number(read(r)) || 0), 0)
}

/** Debit/credit balance predicate with BHD 3dp tolerance. Both totals must be
 * positive AND equal within tolerance — an all-zero voucher is NOT balanced. */
export function isDebitCreditBalanced<Row>(
  rows: Row[],
  debit: (row: Row) => unknown,
  credit: (row: Row) => unknown,
  tolerance = 0.001,
): boolean {
  const dr = sumField(rows, debit)
  const cr = sumField(rows, credit)
  return dr > 0 && cr > 0 && Math.abs(dr - cr) < tolerance
}
