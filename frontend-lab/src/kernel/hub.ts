/* The Hub descriptor — pillar 2 for dashboards. A hub is KPI tiles + a mixed
 * widget grid, computed from ONE typed data payload (the dashboard binding).
 * Widgets are declared data (functions of the payload), rendered by the Hub
 * archetype. Charts are hand-rolled SVG/CSS on the kernel palette — no chart
 * library (recon K3a/K3b: zero chart-lib usage across all 14 dashboards read).
 *
 * L4 ejection: the `bespoke` widget type takes a component override for the
 * genuinely one-off panels (cash-conversion-cycle formula, onboarding copy). */

import type { Component } from 'svelte'
import type { ContentClass } from './descriptor'
import type { LedgerQuery } from './ledger-core'
import type { Tone } from './tones'

/** A drill-down target: a registry screen key + an optional query seed
 * (parity #4 — the same initialQuery LedgerViewModel already consumes). */
export interface NavIntent {
  key: string
  query?: Partial<LedgerQuery>
}

/** Provided to the Hub archetype so widgets/KPIs can drill into ledgers. */
export type Navigate = (intent: NavIntent) => void

/* ---- KPI tiles (the headline strip) ---- */

export interface HubKpiSpec<Data> {
  label: string
  content: ContentClass
  value: (d: Data) => unknown
  currency?: (d: Data) => string
  /** Small trend/delta line under the value (e.g. "+12% vs last month"). */
  delta?: (d: Data) => { text: string; tone: Tone } | null
  /** Tone for the value itself (threshold colouring). */
  tone?: (d: Data) => Tone
  /** Drill-down when the tile is clicked. */
  nav?: (d: Data) => NavIntent | null
}

/* ---- Widgets (the mixed grid) ---- */

export interface WidgetSegment {
  key: string
  label: string
  value: number
  /** 0–100; the archetype also derives this if omitted (value / Σ). */
  pct?: number | undefined
  tone: Tone
  nav?: NavIntent | undefined
}

export interface RankedRow {
  rank: number
  label: string
  value: number
  /** 0–100 bar fill. */
  pct: number
  sublabel?: string | undefined
  nav?: NavIntent | undefined
}

export interface StatItem {
  label: string
  value: unknown
  content?: ContentClass | undefined
  tone?: Tone | undefined
}

export interface ListRow {
  label: string
  value?: string | undefined
  detail?: string | undefined
  tone?: Tone | undefined
  nav?: NavIntent | undefined
}

export interface ActivityItem {
  title: string
  subtitle?: string | undefined
  timestamp?: string | undefined
  tone?: Tone | undefined
  nav?: NavIntent | undefined
}

export interface CalloutItem {
  label: string
  text: string
  tone: Tone
}

export interface ComparisonRow {
  label: string
  /** e.g. prior year. */
  base: number
  /** e.g. current year. */
  current: number
  currency?: string
}

/** Widget span in the responsive grid (1 = normal, 2 = wide/full-row). */
export type WidgetSpan = 1 | 2

interface WidgetBase {
  title: string
  span?: WidgetSpan
}

export type HubWidgetSpec<Data> =
  | (WidgetBase & {
      type: 'distribution'
      orientation?: 'horizontal' | 'vertical'
      segments: (d: Data) => WidgetSegment[]
    })
  | (WidgetBase & { type: 'ranked'; unit?: ContentClass; rows: (d: Data) => RankedRow[] })
  | (WidgetBase & {
      type: 'stat-grid'
      sections: (d: Data) => { title?: string; items: StatItem[] }[]
    })
  | (WidgetBase & { type: 'list'; rows: (d: Data) => ListRow[] })
  | (WidgetBase & { type: 'activity'; items: (d: Data) => ActivityItem[]; emptyMessage?: string })
  | (WidgetBase & { type: 'callout'; items: (d: Data) => CalloutItem[] })
  | (WidgetBase & { type: 'donut'; centerLabel?: string; segments: (d: Data) => WidgetSegment[] })
  | (WidgetBase & {
      type: 'comparison'
      baseLabel: string
      currentLabel: string
      rows: (d: Data) => ComparisonRow[]
    })
  | (WidgetBase & { type: 'bespoke'; component: Component<{ data: Data; navigate: Navigate }> })

/* ---- The descriptor ---- */

export interface HubDescriptor<Data> {
  entity: string
  title: string
  subtitle?: (d: Data) => string
  /** Loads the dashboard payload; receives the selected period when declared. */
  fetch: (period?: string) => Promise<Data>
  /** Optional period selector that drives a full refetch (FY / all-years). */
  period?: { label: string; options: { value: string; label: string }[]; default: string }
  kpis: HubKpiSpec<Data>[]
  widgets: HubWidgetSpec<Data>[]
}
