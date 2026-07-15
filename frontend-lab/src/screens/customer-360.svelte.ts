/* Customer 360 viewmodel — L5's reactive half: picker directory, selected-
 * customer load (info + connections in parallel), and tab state. No
 * rendering/layout; Customer360.svelte binds an instance and renders on
 * primitives only (L1), same split as pricing-vm.svelte.ts /
 * serial-trace.svelte.ts.
 *
 * Named `customer-360.svelte.ts` (not `Customer360.svelte.ts`) so its stem
 * never differs from `Customer360.svelte` by case only — that collides under
 * TypeScript's case-insensitive file resolution on Windows (same reasoning
 * pricing-vm.svelte.ts documents). */

import {
  fetchCustomer360,
  fetchCustomer360Connections,
  fetchCustomer360Directory,
  type Customer360Info,
  type CustomerConnections,
  type CustomerDirectoryEntry,
} from '../bridge/customer-360'

export type Customer360Tab = 'overview' | 'predictions' | 'connections'

export class Customer360ViewModel {
  directory = $state<CustomerDirectoryEntry[]>([])
  loadingDirectory = $state(true)
  directoryError = $state<string | null>(null)

  /** '' = nothing picked yet (or the picker's "All" chip was clicked to
   * deselect) — renders the "pick a customer" empty state. */
  selectedId = $state('')

  data = $state<Customer360Info | null>(null)
  connections = $state<CustomerConnections | null>(null)
  loading = $state(false)
  error = $state<string | null>(null)

  tab = $state<Customer360Tab>('overview')

  async loadDirectory(): Promise<void> {
    this.loadingDirectory = true
    this.directoryError = null
    try {
      this.directory = await fetchCustomer360Directory()
      if (!this.selectedId && this.directory.length > 0) this.selectedId = this.directory[0]!.id
    } catch (e) {
      this.directoryError = e instanceof Error ? e.message : String(e)
    } finally {
      this.loadingDirectory = false
    }
  }

  /** Loads whatever `selectedId` currently is — called from the screen's
   * `$effect` so it re-runs on every picker change (same shape as
   * SerialTrace's mount-effect calling into the viewmodel). */
  async loadSelected(): Promise<void> {
    const id = this.selectedId
    if (!id) {
      this.data = null
      this.connections = null
      return
    }
    this.loading = true
    this.error = null
    try {
      const [data, connections] = await Promise.all([fetchCustomer360(id), fetchCustomer360Connections(id)])
      this.data = data
      this.connections = connections
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
      this.data = null
      this.connections = null
    } finally {
      this.loading = false
    }
  }
}
