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

  /* Profile secondary-fetch (INTEG): when an EntityMaster descriptor declares
   * `profile.enrich`, selecting a row triggers a second fetch (GetXFullProfile)
   * whose result is merged into the row. `enriching` gates a spinner; a failure
   * is NON-FATAL (the profile stays at list-depth with honest blanks). Enriched
   * ids are remembered so re-selecting doesn't refetch; cleared on reload. */
  enriching = $state(false)
  enrichError = $state<string | null>(null)
  private enrichedIds = new SvelteSet<string>()

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
      this.enrichedIds.clear() // fresh rows are at list-depth again
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

  /** Secondary-fetch the selected row's profile depth and merge it in. Idempotent
   * per id; safe to call on every selection change. Non-fatal on failure. */
  async enrichSelected(enrich: (row: Row) => Promise<Partial<Row>>): Promise<void> {
    const id = this.selectedId
    if (id == null || this.enrichedIds.has(id)) return
    const row = this.rows.find((r) => this.descriptor.id(r) === id)
    if (!row) return
    this.enriching = true
    this.enrichError = null
    try {
      const patch = await enrich(row)
      // Reassign the row (not mutate) so the `selected` derived recomputes.
      this.rows = this.rows.map((r) => (this.descriptor.id(r) === id ? { ...r, ...patch } : r))
      this.enrichedIds.add(id)
    } catch (e) {
      // Profile stays at list-depth (honest blanks) — never a page-level failure.
      this.enrichError = e instanceof Error ? e.message : String(e)
    } finally {
      this.enriching = false
    }
  }
}
