/* Pure ledger logic — L5's testability half. No runes, no DOM, no bridge:
 * these functions are the viewmodel's brain and vitest exercises them
 * directly in node. The reactive shell (ledger.svelte.ts) stays thin. */

import type { FilterSpec, LedgerDescriptor } from './descriptor'

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

/** Options for a 'derive' filter: distinct values present in the data,
 * sorted, so a deployment's real vocabulary shows up with zero config. */
export function deriveFilterOptions<Row>(
  filter: FilterSpec<Row>,
  rows: Row[],
): { value: string; label: string }[] {
  if (filter.options !== 'derive') return filter.options
  const derive = filter.deriveValue
  if (!derive) return []
  const seen = new Set<string>()
  for (const row of rows) {
    const v = derive(row)
    if (v) seen.add(v)
  }
  return [...seen].sort().map((v) => ({ value: v, label: v }))
}
