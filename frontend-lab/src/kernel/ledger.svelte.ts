/* The ledger viewmodel — L5's reactive half. A thin rune shell over
 * ledger-core: state + derivations, no rendering, no layout. Views bind
 * to an instance of this; the DocumentLedger archetype owns its lifecycle. */

import type { LedgerDescriptor } from './descriptor'
import { applyLedgerQuery, deriveFilterOptions } from './ledger-core'

export class LedgerViewModel<Row> {
  rows = $state<Row[]>([])
  loading = $state(true)
  error = $state<string | null>(null)
  search = $state('')
  filters = $state<Record<string, string>>({})
  selectedId = $state<string | null>(null)

  constructor(readonly descriptor: LedgerDescriptor<Row>) {}

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

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      this.rows = await this.descriptor.fetch()
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  select(row: Row | null): void {
    this.selectedId = row == null ? null : this.descriptor.id(row)
  }
}
