/* Pricing viewmodel — L5's reactive half. State + derivations for the
 * margin simulator: customer list, selection, target-margin slider, and the
 * async simulation result. No rendering/layout.
 *
 * Named `pricing-vm` (not `pricing.svelte.ts`) so its stem never differs
 * from `Pricing.svelte` by case only — that collides under TypeScript's
 * case-insensitive file resolution on Windows. */

import {
  fetchPricingCustomers,
  simulateMargin,
  type MarginSimulationResult,
  type PricingCustomerRow,
} from '../bridge/pricing'

export class PricingViewModel {
  customers = $state<PricingCustomerRow[]>([])
  loading = $state(true)
  error = $state<string | null>(null)

  selectedId = $state<string | null>(null)
  /** 20% — same default the old screen's slider opened on. */
  targetMargin = $state(0.2)

  result = $state<MarginSimulationResult | null>(null)
  simulating = $state(false)
  simError = $state<string | null>(null)

  selected = $derived.by(() => this.customers.find((c) => c.id === this.selectedId) ?? null)

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      this.customers = await fetchPricingCustomers()
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  select(row: PricingCustomerRow): void {
    this.selectedId = row.id
    this.result = null
    this.simError = null
  }

  async simulate(): Promise<void> {
    const c = this.selected
    if (!c) return
    this.simulating = true
    this.simError = null
    try {
      this.result = await simulateMargin(c.name, this.targetMargin)
    } catch (e) {
      this.simError = e instanceof Error ? e.message : String(e)
      this.result = null
    } finally {
      this.simulating = false
    }
  }
}
