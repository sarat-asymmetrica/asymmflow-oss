/* The ledger viewmodel — L5's reactive half. A thin rune shell over
 * ledger-core: state + derivations, no rendering, no layout. Views bind
 * to an instance of this; the archetypes own its lifecycle. */

import { SvelteSet } from 'svelte/reactivity'
import type { LedgerDescriptor } from './descriptor'
import {
  applyLedgerQuery,
  computeSummary,
  deriveFilterOptions,
  type LedgerQuery,
} from './ledger-core'

const DEFAULT_PAGE_SIZE = 100

export class LedgerViewModel<Row> {
  rows = $state<Row[]>([])
  loading = $state(true)
  error = $state<string | null>(null)
  search = $state('')
  filters = $state<Record<string, string>>({})
  selectedId = $state<string | null>(null)

  /* Paged loading (parity #1/#19) — engaged when descriptor.fetchPage exists. */
  hasMore = $state(false)
  loadingMore = $state(false)

  /* Column visibility (parity #15) — hidden column keys, per VM instance. */
  hiddenColumns = new SvelteSet<string>()

  constructor(
    readonly descriptor: LedgerDescriptor<Row>,
    /** Initial-query seeding (parity #4): dashboard drills pre-filter here. */
    initialQuery?: Partial<LedgerQuery>,
  ) {
    if (initialQuery?.search) this.search = initialQuery.search
    if (initialQuery?.filters) this.filters = { ...initialQuery.filters }
  }

  private get pageSize(): number {
    return this.descriptor.pageSize ?? DEFAULT_PAGE_SIZE
  }

  visible = $derived.by(() =>
    applyLedgerQuery(this.descriptor, this.rows, { search: this.search, filters: this.filters }),
  )

  selected = $derived.by(() => {
    if (this.selectedId == null) return null
    return this.rows.find((r) => this.descriptor.id(r) === this.selectedId) ?? null
  })

  filterOptions = $derived.by(() =>
    (this.descriptor.filters ?? []).map((f) => ({
      spec: f,
      options: deriveFilterOptions(f, this.rows),
    })),
  )

  /** The summary strip, reduced over the VISIBLE (filtered) rows so it responds
   * live to search/filters. null when the descriptor declares no summary. */
  summary = $derived.by(() => computeSummary(this.descriptor.summary, this.visible))

  visibleColumns = $derived.by(() =>
    this.descriptor.columns.filter((c) => !this.hiddenColumns.has(c.key)),
  )

  toggleColumn(key: string): void {
    if (this.hiddenColumns.has(key)) this.hiddenColumns.delete(key)
    else this.hiddenColumns.add(key)
  }

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      if (this.descriptor.fetchPage) {
        const page = await this.descriptor.fetchPage(this.pageSize, 0)
        this.rows = page
        this.hasMore = page.length === this.pageSize
      } else {
        this.rows = await this.descriptor.fetch()
        this.hasMore = false
      }
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  async loadMore(): Promise<void> {
    if (!this.descriptor.fetchPage || this.loadingMore || !this.hasMore) return
    this.loadingMore = true
    try {
      const page = await this.descriptor.fetchPage(this.pageSize, this.rows.length)
      this.rows = [...this.rows, ...page]
      this.hasMore = page.length === this.pageSize
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loadingMore = false
    }
  }

  select(row: Row | null): void {
    this.selectedId = row == null ? null : this.descriptor.id(row)
  }
}
