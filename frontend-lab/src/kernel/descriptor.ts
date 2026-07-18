/* The descriptor schema — pillar 2 of KERNEL.md.
 *
 * A screen in the archetype system is a typed, compiled-in descriptor:
 * declared data that an archetype engine (DocumentLedger, EntityMaster, …)
 * renders. Descriptors carry NO layout and NO imperative rendering — and
 * because they are data, they are verifiable (pretext arithmetic can prove
 * a declared column fits its declared content class at its declared width).
 *
 * L4 — graceful ejection is load-bearing: every level of the descriptor
 * accepts a component override (cell → panel → whole screen bespoke).
 * A screen never fights the engine; it steps outside it explicitly.
 */

import type { Component } from 'svelte'
import type { FormSpec } from './form'
import type { Tone } from './tones'

/** Drives formatting, alignment, fonts AND dev-time layout verification.
 * Each class has a known worst-case content shape (see verify/ later). */
export type ContentClass =
  | 'code' // document numbers, unbroken tokens (INV-2026-…)
  | 'name' // free-text names — the 200-char adversary lives here
  | 'money' // right-aligned, numeric font, BHD 3-decimals
  | 'quantity'
  | 'date'
  | 'status' // badge-rendered, finite vocabulary
  | 'text' // general prose, wraps freely

export interface ColumnSpec<Row> {
  key: string
  label: string
  content: ContentClass
  value: (row: Row) => unknown
  /** px at default zoom; the pretext harness verifies worst-case content
   * of `content` class fits (or truncates by declared policy) at this width. */
  minWidth?: number
  /** Column absorbs remaining space (at most one grower per ledger). */
  grow?: boolean
  /** For money columns whose currency varies per row (default BHD). */
  currency?: (row: Row) => string
  /** Threshold/semantic colour for the cell text — e.g. GRN acceptance rate
   * (green ≥95, amber ≥80, red <80), overdue dates, unapplied balances.
   * Tints the text via the shared tone palette; never a raw hex in a screen. */
  tone?: (row: Row) => Tone
  /** L4 ejection, cell granularity. */
  cell?: Component<{ row: Row }>
}

export interface StatusSpec<Row> {
  value: (row: Row) => string
  /** Finite vocabulary → badge tone. Unknown statuses render neutral, never crash. */
  tones: Record<string, Tone>
  /** Optional legal state machine: current status → allowed next statuses.
   * Recurs across POs, cheques, supplier-invoices, expenses, credit-notes
   * (recon K1-B synthesis #2). Actions gate on it via `nextStates()`; it is
   * declared data, so a status graph is auditable rather than buried in
   * per-screen `visible` predicates. */
  transitions?: Record<string, string[]>
}

export interface FilterSpec<Row> {
  key: string
  label: string
  /** Static options, or derived from loaded rows (e.g. divisions come from
   * the divisions store — L7: never a hardcoded division literal). */
  options: 'derive' | { value: string; label: string }[]
  predicate: (row: Row, selected: string) => boolean
  deriveValue?: (row: Row) => string
}

export interface ActionSpec<Row> {
  key: string
  label: string
  kind: 'screen' | 'row'
  /** Visibility gate — e.g. only Draft invoices can be edited. */
  visible?: (row: Row | null) => boolean
  /** Declared escalation: a form action opens FormModal (its submit IS the
   * action); a confirm action gates run() behind ConfirmDialog; a modal action
   * ejects (L4) to a bespoke component for flows a flat FormSpec can't express
   * (e.g. per-line receive quantities via LineItemsEditor). One path for all
   * archetypes via ActionHost. */
  form?: FormSpec<any>
  /** L4 ejection at action granularity: a bespoke modal component. It receives
   * the clicked `row`, `reload` (to refresh the ledger after a mutation), and
   * `close` (to dismiss). The component owns its own submit — `run` is unused. */
  modal?: Component<{ row: Row | null; reload: () => Promise<void>; close: () => void }>
  confirm?: (row: Row | null) => string
  run: (ctx: { row: Row | null; reload: () => Promise<void> }) => void | Promise<void>
}

/* ---- Summary strip: the declarative KPI/stats header (recon: ~10 of 13
 * ledgers hand-roll a stats grid). Metrics are computed over the currently
 * VISIBLE (filtered) rows, so the strip responds live to search/filters. This
 * is the visual-diversity vehicle: a compact metric row + one status
 * distribution bar, replacing card-heavy headers. Server-backed summaries
 * (e.g. Expenses' GetExpenseDashboardSummary) are a later INTEG enhancement. */

export interface SummaryMetricSpec<Row> {
  label: string
  content: ContentClass
  /** Reduced over the visible rows. */
  value: (rows: Row[]) => unknown
  currency?: (rows: Row[]) => string
  /** Optional emphasis tone (e.g. amber when total unapplied > 0). */
  tone?: (rows: Row[]) => Tone
}

export interface SummaryDistributionSpec<Row> {
  label?: string
  /** Bucket key per row (usually the status). */
  value: (row: Row) => string
  /** Bucket key → tone; unknown buckets render neutral. */
  tones: Record<string, Tone>
}

export interface SummarySpec<Row> {
  metrics: SummaryMetricSpec<Row>[]
  distribution?: SummaryDistributionSpec<Row>
}

/** The DocumentLedger archetype's contract: the pattern hand-written
 * ~15× in the old frontend (Invoices, Orders, POs, Cheques, GRNs, …). */
export interface LedgerDescriptor<Row> {
  /** Entity key — will link to the backend UI schema (GetUISchema) later. */
  entity: string
  title: string
  /** Bridge call that loads the rows (mock bridge in the lab). */
  fetch: () => Promise<Row[]>
  /** Paged loading (parity #1/#19): when present, the VM pages with
   * fetchPage(limit, offset) + Load More instead of load-all fetch(). */
  fetchPage?: (limit: number, offset: number) => Promise<Row[]>
  pageSize?: number
  id: (row: Row) => string
  /** Fields swept by the search box — ONE search implementation (L2). */
  searchText: (row: Row) => string
  columns: ColumnSpec<Row>[]
  status?: StatusSpec<Row>
  /** Declarative KPI/stats strip over the visible rows (visual-diversity). */
  summary?: SummarySpec<Row>
  filters?: FilterSpec<Row>[]
  actions?: ActionSpec<Row>[]
  /** L4 ejection, panel granularity. */
  slots?: {
    /** Custom detail panel when a row is selected. */
    detail?: Component<{ row: Row; reload: () => Promise<void> }>
    /** Custom empty state. */
    empty?: Component
  }
  emptyMessage?: string
}

/* ---- EntityMaster: the second archetype ---- */

export interface ProfileFieldSpec<Row> {
  label: string
  content: ContentClass
  value: (row: Row) => unknown
  currency?: (row: Row) => string
}

export interface ProfileSectionSpec<Row> {
  title: string
  fields: ProfileFieldSpec<Row>[]
}

export interface ProfileKpiSpec<Row> {
  label: string
  content: ContentClass
  value: (row: Row) => unknown
  currency?: (row: Row) => string
  /** Semantic colour for the KPI value — e.g. a credit-blocked customer's
   * balance in danger red. Mirrors ColumnSpec.tone / SummaryMetricSpec.tone. */
  tone?: (row: Row) => Tone
}

/** The EntityMaster archetype's contract: master list + rich profile.
 * Pattern hand-written in the old frontend for Customers, Suppliers, Users,
 * plus the CRM detail views. Structurally a LedgerDescriptor with a profile,
 * so the same LedgerViewModel drives both archetypes (one path, L2). */
export interface EntityDescriptor<Row> extends LedgerDescriptor<Row> {
  profile: {
    heading: (row: Row) => string
    subheading?: (row: Row) => string
    badge?: StatusSpec<Row>
    kpis?: ProfileKpiSpec<Row>[]
    sections: ProfileSectionSpec<Row>[]
    /** Secondary profile fetch (INTEG): the list `fetch` returns list-depth rows;
     * when a row is SELECTED, the archetype calls this with the row and merges the
     * returned partial in (e.g. GetCustomerFullProfile → TRN/aging/win-rate that
     * ListCustomers doesn't carry). Idempotent per row; failure is non-fatal (the
     * profile stays at list-depth). Omit when the list fetch is already complete. */
    enrich?: (row: Row) => Promise<Partial<Row>>
  }
}
