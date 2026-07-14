/* The screen registry — the single source of "what screens exist" for the
 * kernel app. Each entry maps a nav key to an archetype + its descriptor (or a
 * bespoke component). App.svelte renders from this list; it grows wave by wave
 * and becomes the real sidebar nav at K5. Orchestrator-owned merge point so
 * parallel build agents never contend on it.
 *
 * `descriptor` is intentionally loosely typed here: each archetype re-narrows
 * its own descriptor at the render site (the `bind:this` generic-erasure seam,
 * KERNEL lesson 2). Descriptor files stay fully typed. */

import type { Component } from 'svelte'
import { invoicesDescriptor } from './invoices.descriptor'
import { customersDescriptor } from './customers.descriptor'
import Showcase from './Showcase.svelte'

export type ArchetypeKind = 'ledger' | 'entity' | 'hub' | 'bespoke'

export interface ScreenEntry {
  key: string
  label: string
  /** Nav grouping — Sales / Finance / Operations / People / System. */
  group: string
  archetype: ArchetypeKind
  /** For ledger/entity/hub archetypes. */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  descriptor?: any
  /** For bespoke/hub screens rendered by a hand-written component. */
  component?: Component
}

export const screens: ScreenEntry[] = [
  // Pilots
  { key: 'invoices', label: 'Invoices', group: 'Finance', archetype: 'ledger', descriptor: invoicesDescriptor },
  { key: 'customers', label: 'Customers', group: 'Sales', archetype: 'entity', descriptor: customersDescriptor },
  { key: 'showcase', label: 'Showcase', group: 'Lab', archetype: 'bespoke', component: Showcase },
]

/** Stable group order for the nav. Unknown groups append alphabetically. */
export const GROUP_ORDER = ['Sales', 'Finance', 'Operations', 'People', 'System', 'Lab']

export function screensByGroup(): { group: string; items: ScreenEntry[] }[] {
  const groups = new Map<string, ScreenEntry[]>()
  for (const s of screens) {
    if (!groups.has(s.group)) groups.set(s.group, [])
    groups.get(s.group)!.push(s)
  }
  const order = (g: string) => {
    const i = GROUP_ORDER.indexOf(g)
    return i === -1 ? GROUP_ORDER.length : i
  }
  return [...groups.entries()]
    .sort((a, b) => order(a[0]) - order(b[0]) || a[0].localeCompare(b[0]))
    .map(([group, items]) => ({ group, items }))
}
