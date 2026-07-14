/* Hub viewmodel — L5's reactive half for the Hub archetype. Loads the one
 * dashboard payload, tracks the selected period, and exposes loading/error so
 * the archetype renders an honest state and never fabricates numbers on a
 * failed load. */

import type { HubDescriptor } from './hub'

export class HubViewModel<Data> {
  data = $state<Data | null>(null)
  loading = $state(true)
  error = $state<string | null>(null)
  period = $state<string>('')

  constructor(readonly descriptor: HubDescriptor<Data>) {
    this.period = descriptor.period?.default ?? ''
  }

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      this.data = await this.descriptor.fetch(this.period || undefined)
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
      this.data = null
    } finally {
      this.loading = false
    }
  }

  setPeriod(value: string): void {
    if (value === this.period) return
    this.period = value
    void this.load()
  }
}
