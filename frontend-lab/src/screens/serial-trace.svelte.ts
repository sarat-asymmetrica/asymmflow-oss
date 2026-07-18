/* Serial Trace viewmodel — L5's reactive half: state + search, no layout,
 * same split as kernel/ledger.svelte.ts. SerialTrace.svelte binds an
 * instance of this and renders on primitives only (L1). Also exports the
 * two pure warranty helpers (label + tone) — ONE definition (L2) shared by
 * the column's declared `value` and the ejected SerialWarrantyBadge cell,
 * so the badge a user reads and the text a future search/export would see
 * never drift apart. */

import { formatDate } from '$kernel/format'
import type { Tone } from '$kernel/tones'
import { recentlyDeliveredSerials, searchSerials, type SerialTraceRow } from '../bridge/serial-trace'

const RECENT_COUNT = 20
const SEARCH_LIMIT = 200

/** Warranty end date compares as a plain 'YYYY-MM-DD' string against
 * today's, same lexicographic-safe pattern the bridge layer's goDate()
 * output guarantees. */
function isExpired(row: SerialTraceRow): boolean {
  return !!row.warrantyEndDate && row.warrantyEndDate < new Date().toISOString().slice(0, 10)
}

export function warrantyLabel(row: SerialTraceRow): string {
  if (!row.warrantyEndDate) return 'No warranty'
  return `${isExpired(row) ? 'Expired' : 'Valid until'} ${formatDate(row.warrantyEndDate)}`
}

/** Expired reads neutral/muted, valid reads success — mirrors the old
 * screen's grey-vs-green warranty coloring exactly (SerialTraceScreen.svelte
 * warranty_end_date column). */
export function warrantyTone(row: SerialTraceRow): Tone {
  if (!row.warrantyEndDate) return 'neutral'
  return isExpired(row) ? 'neutral' : 'success'
}

export class SerialTraceViewModel {
  query = $state('')
  hasSearched = $state(false)
  results = $state<SerialTraceRow[]>([])
  recent = $state<SerialTraceRow[]>([])
  searching = $state(false)
  loadingRecent = $state(true)
  error = $state<string | null>(null)

  /** Rows to render: search results once a search has run, otherwise the
   * "recently delivered" greeting (old screen's B10-4 behavior). */
  rows = $derived.by(() => (this.hasSearched ? this.results : this.recent))

  async loadRecent(): Promise<void> {
    this.loadingRecent = true
    try {
      this.recent = await recentlyDeliveredSerials(RECENT_COUNT)
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
      this.recent = []
    } finally {
      this.loadingRecent = false
    }
  }

  async search(): Promise<void> {
    const q = this.query.trim()
    if (!q) return
    this.searching = true
    this.hasSearched = true
    this.error = null
    try {
      this.results = await searchSerials(q, SEARCH_LIMIT)
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
      this.results = []
    } finally {
      this.searching = false
    }
  }

  /** Back to the recently-delivered greeting (e.g. the search box is cleared). */
  reset(): void {
    this.query = ''
    this.hasSearched = false
    this.results = []
  }
}
