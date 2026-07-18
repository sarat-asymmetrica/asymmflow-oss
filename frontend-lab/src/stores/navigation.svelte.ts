/* Navigation store — the kernel app's routing state + cross-screen handoffs.
 * Replaces the old frontend's three tangled nav mechanisms (window custom
 * events, component-event bubbling, pending-* writable stores) with ONE
 * rune-based surface (L2). The shell renders `currentRoute()`; anything can
 * `navigate()`. Cross-screen "open X with this payload" handoffs (the old
 * pendingDNCreate / pendingProjectHandoff / pendingInvoiceCreate / openTask
 * flows the parity docs deferred to K5) go through set/consumeHandoff. */

import type { LedgerQuery } from '$kernel/ledger-core'

export interface Route {
  /** Registry key of the active screen. */
  key: string
  /** Seeds a ledger drill-down (search/filters) on arrival — parity #4. */
  query?: Partial<LedgerQuery>
  /** For hub screens: which tab to open on arrival. */
  tab?: string
}

let route = $state<Route>({ key: '' })

export function currentRoute(): Route {
  return route
}

/** Navigate to a screen, optionally seeding a drill-down query or hub tab. */
export function navigate(key: string, opts?: { query?: Partial<LedgerQuery>; tab?: string }): void {
  route = { key, ...(opts?.query ? { query: opts.query } : {}), ...(opts?.tab ? { tab: opts.tab } : {}) }
}

/** Set the initial route (shell boot) without a drill-down. */
export function setInitialRoute(key: string): void {
  if (!route.key) route = { key }
}

/** Resolve a tab-navigator hub's active tab from the current route: returns
 * `route.tab` when it names one of the hub's known tabs, else `fallback`. Lets
 * `navigate(hubKey, { tab })` deep-link straight to a hub tab (the Route.tab
 * contract, previously defined but unwired). Read inside a hub's init AND an
 * `$effect(() => …)` so an in-place re-navigation switches tabs too. */
export function routeTabOr(validKeys: readonly string[], fallback: string): string {
  const t = route.tab
  return t && validKeys.includes(t) ? t : fallback
}

/* ---- cross-screen handoffs: a source screen stashes a payload under a key;
 * the target screen consumes it once on mount (self-clearing, one-shot). ---- */
const handoffs = $state<Record<string, unknown>>({})

export function setHandoff(key: string, payload: unknown): void {
  handoffs[key] = payload
}

/** Read + clear a handoff (returns undefined if none). */
export function consumeHandoff(key: string): unknown {
  if (!(key in handoffs)) return undefined
  const payload = handoffs[key]
  delete handoffs[key]
  return payload
}
