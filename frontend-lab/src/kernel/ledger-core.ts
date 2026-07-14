/* Pure ledger logic — L5's testability half. No runes, no DOM, no bridge:
 * these functions are the viewmodel's brain and vitest exercises them
 * directly in node. The reactive shell (ledger.svelte.ts) stays thin. */

import type { ContentClass, FilterSpec, LedgerDescriptor, SummarySpec } from './descriptor'
import type { Tone } from './tones'

export interface LedgerQuery {
  search: string
  /** filter key → selected value; '' or absent = All. */
  filters: Record<string, string>
}

export function applyLedgerQuery<Row>(
  descriptor: LedgerDescriptor<Row>,
  rows: Row[],
  query: LedgerQuery,
): Row[] {
  const needle = query.search.trim().toLowerCase()
  const active = (descriptor.filters ?? []).filter((f) => {
    const sel = query.filters[f.key]
    return sel != null && sel !== ''
  })
  return rows.filter((row) => {
    if (needle && !descriptor.searchText(row).toLowerCase().includes(needle)) return false
    for (const f of active) {
      if (!f.predicate(row, query.filters[f.key]!)) return false
    }
    return true
  })
}

export interface FilterOption {
  value: string
  label: string
  /** How many currently-loaded rows fall in this option (chip count badge). */
  count: number
}

/** Options for a filter, each with a live count. 'derive' filters surface the
 * deployment's real vocabulary (distinct values in the data, sorted); static
 * filters keep their declared order. Counts run the filter's own predicate so
 * derive and static count identically (recon: ≥8 ledgers want count-in-chip). */
export function deriveFilterOptions<Row>(filter: FilterSpec<Row>, rows: Row[]): FilterOption[] {
  let base: { value: string; label: string }[]
  if (filter.options === 'derive') {
    const derive = filter.deriveValue
    if (!derive) return []
    const seen = new Set<string>()
    for (const row of rows) {
      const v = derive(row)
      if (v) seen.add(v)
    }
    base = [...seen].sort().map((v) => ({ value: v, label: v }))
  } else {
    base = filter.options
  }
  return base.map((o) => ({
    ...o,
    count: rows.reduce((n, row) => (filter.predicate(row, o.value) ? n + 1 : n), 0),
  }))
}

/* ---- Status transitions ---- */

/** Legal next statuses from `status` per a declared transition table; [] when
 * the status is terminal or unknown. Actions gate row visibility on this. */
export function nextStates(
  status: string,
  transitions: Record<string, string[]> | undefined,
): string[] {
  return transitions?.[status] ?? []
}

/* ---- Summary computation (visual-diversity strip) ---- */

export interface ComputedMetric {
  label: string
  content: ContentClass
  value: unknown
  currency?: string | undefined
  tone?: Tone | undefined
}

export interface ComputedDistributionSegment {
  key: string
  count: number
  tone: Tone
  /** Percentage of the total (0–100), for the bar segment width. */
  pct: number
}

export interface ComputedSummary {
  metrics: ComputedMetric[]
  distribution?:
    | {
        label?: string | undefined
        total: number
        segments: ComputedDistributionSegment[]
      }
    | undefined
}

/** Reduce a summary spec over the given (visible) rows into render-ready data.
 * Pure — the component only formats via renderCell and draws the bar. */
export function computeSummary<Row>(
  spec: SummarySpec<Row> | undefined,
  rows: Row[],
): ComputedSummary | null {
  if (!spec) return null
  const metrics: ComputedMetric[] = spec.metrics.map((m) => ({
    label: m.label,
    content: m.content,
    value: m.value(rows),
    currency: m.currency?.(rows),
    tone: m.tone?.(rows),
  }))

  let distribution: ComputedSummary['distribution']
  if (spec.distribution) {
    const dist = spec.distribution
    const counts = new Map<string, number>()
    for (const row of rows) {
      const key = dist.value(row) || '—'
      counts.set(key, (counts.get(key) ?? 0) + 1)
    }
    const total = rows.length
    const segments: ComputedDistributionSegment[] = [...counts.entries()]
      .sort((a, b) => b[1] - a[1])
      .map(([key, count]) => ({
        key,
        count,
        tone: dist.tones[key] ?? 'neutral',
        pct: total > 0 ? (count / total) * 100 : 0,
      }))
    distribution = { label: dist.label, total, segments }
  }

  return { metrics, distribution }
}
