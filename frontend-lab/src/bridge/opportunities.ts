/* Opportunities bridge module — self-contained: types + mock + real + switch.
 * Old screen: OpportunitiesScreen.svelte merges two real record kinds client-
 * side — RFQs (early-stage enquiries, `GetRFQs`) and pipeline Opportunities
 * (post-offer, further down the funnel, `GetPipelineOpportunities`) — into one
 * sales-pipeline feed. This bridge reproduces that merge.
 *
 * Per the K4 orchestrator brief, BOTH fetch and mutations are INTEG-gapped for
 * this screen (two-source merge + cascade-delete blast radius) — unlike the K1
 * ledgers (rfqs/purchase-orders/cheque-register), where fetch is wired real.
 * Mock stands in entirely until K5; the real side throws, naming the exact
 * Go bindings it will call. */

import { pick } from './runtime'
import { goDate, num, str } from './map'
import { GetRFQs, GetPipelineOpportunities, ListCustomers } from '$wails/go/main/App'

export interface OpportunityRow {
  id: string
  /** Which real table this row came from — drives which delete binding
   * applies (only RFQs have a cascade-delete binding, see below). */
  source: 'rfq' | 'pipeline'
  ref: string
  customer: string
  project: string
  value: number
  stage: string
  createdAt: string
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts pattern) ---- */

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number): string => String(n).padStart(w, '0')

const CUSTOMERS = [
  'Gulf Fabrication W.L.L.',
  'Manama Process Systems',
  'Al Dana Engineering Co.',
  'Interntional Establishment for Industrial & Petrochemical Instrumentation Services and General Trading (formerly Gulf Technical Calibration & Measurement Systems Company) W.L.L.',
  'المؤسسة الدولية لخدمات الأجهزة الصناعية والبتروكيماوية والتجارة العامة ذ.م.م',
  'Sitra Contracting',
  'X',
  'Bahrain Water Authority — Directorate of Operations & Maintenance, Section 7',
]
const PROJECTS = [
  'Flow metering skid replacement',
  'DCS migration — Phase 2',
  'Tank farm level instrumentation',
  '',
  'Turbine control retrofit',
  'Analyzer shelter upgrade',
]
// Same six-stage vocabulary rfqs.descriptor.ts declares. Pipeline Opportunity's
// real `stage` field is confirmed (app_sales_pipeline.go) to include "Won"/
// "Lost" but its full enumeration wasn't verified beyond that in this recon —
// mock reuses the RFQ vocabulary rather than inventing an unverified one.
const STAGES = ['Pending', 'Qualified', 'Proposal', 'Negotiation', 'Won', 'Lost']

let cache: OpportunityRow[] | null = null

function generate(): OpportunityRow[] {
  const rand = lcg(20260714 + 3)
  const rows: OpportunityRow[] = []
  const n = 150
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 18)
    const year = 2025 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const createdAt = `${year}-${pad(month, 2)}-${pad(day, 2)}`

    // ~2/3 RFQ-sourced, ~1/3 pipeline-sourced — plausible funnel shape (more
    // enquiries than opportunities that progressed past the offer stage).
    const source: OpportunityRow['source'] = i % 3 === 0 ? 'pipeline' : 'rfq'
    const stage = i % 97 === 0 ? 'UNKNOWN_STAGE' : STAGES[Math.floor(r * STAGES.length)]!
    const value =
      i % 89 === 0 ? 87654321098.765 : i % 53 === 0 ? 0.001 : Math.round(rand() * 2_000_000) / 100

    rows.push({
      id: `opp-${source}-${i}`,
      source,
      ref: source === 'rfq' ? `RFQ-${pad(i, 4)}` : `OPP-${year}-${pad(i, 4)}`,
      customer: CUSTOMERS[i % CUSTOMERS.length]!,
      project: PROJECTS[i % PROJECTS.length]!,
      value,
      stage,
      createdAt,
    })
  }
  return rows
}

async function mockFetch(): Promise<OpportunityRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

export interface NewOpportunityDraft {
  customer: string
  project: string
  value: number | null
  notes: string
}

let createdCount = 0

async function mockCreate(draft: NewOpportunityDraft): Promise<void> {
  cache ??= generate()
  createdCount++
  cache.unshift({
    id: `opp-rfq-new-${createdCount}`,
    source: 'rfq',
    ref: `RFQ-N${pad(createdCount, 3)}`,
    customer: draft.customer,
    project: draft.project,
    value: draft.value ?? 0,
    stage: 'Pending',
    createdAt: new Date().toISOString().slice(0, 10),
  })
  await sleep(150)
}

async function mockDelete(row: OpportunityRow): Promise<void> {
  cache ??= generate()
  cache = cache.filter((x) => x.id !== row.id)
  await sleep(120)
}

async function mockCascadeDelete(row: OpportunityRow, reason: string): Promise<void> {
  void reason // reason is captured for the (mocked) audit trail, not replayed here
  await mockDelete(row)
}

async function mockCustomerOptions(): Promise<{ value: string; label: string }[]> {
  await sleep(100)
  return CUSTOMERS.filter((c) => c).map((c) => ({ value: c, label: c }))
}

/* ---- real: INTEG-gapped entirely (fetch merges two sources; mutations touch
 * a financial/cascade-delete hot zone) — naming the exact bindings for K5 ---- */

/* Mirrors costing-sheet.ts's proven RFQ+pipeline merge mapping (the same two
 * bindings, already wired real there). Pipeline rows whose folder already has an
 * RFQ row are dropped (they're the same deal further down the funnel), matching
 * the old OpportunitiesScreen's dedup. */
function mapRfq(r: Record<string, unknown>): OpportunityRow {
  const id = str(num(r.id))
  return {
    id,
    source: 'rfq',
    ref: str(r.rfq_ref) || str(r.rfq_number) || `RFQ-${id}`,
    customer: str(r.client),
    project: str(r.project),
    value: num(r.value),
    stage: str(r.status) || str(r.stage),
    createdAt: goDate(r.created_at),
  }
}
function mapPipeline(o: Record<string, unknown>): OpportunityRow {
  return {
    id: str(o.id),
    source: 'pipeline',
    ref: str(o.eh_ref) || str(o.folder_number),
    customer: str(o.customer_name),
    project: str(o.title) || str(o.folder_name),
    value: num(o.revenue_bhd),
    stage: str(o.stage),
    createdAt: goDate(o.updated_at) || goDate(o.offer_date),
  }
}

async function realFetch(): Promise<OpportunityRow[]> {
  const [rfqs, pipeline] = await Promise.all([GetRFQs(500, 0), GetPipelineOpportunities(500, 0)])
  const rfqRecords = (rfqs ?? [])
    .map((r) => r as unknown as Record<string, unknown>)
    .filter((r) => str(r.rfq_number).trim())
  const rfqFolders = new Set(rfqRecords.map((r) => str(r.rfq_number)).filter(Boolean))
  const rfqRows = rfqRecords.map(mapRfq)
  const pipelineRows = (pipeline ?? [])
    .map((o) => o as unknown as Record<string, unknown>)
    .filter((o) => !rfqFolders.has(str(o.folder_number)))
    .map(mapPipeline)
  return [...pipelineRows, ...rfqRows].sort((a, b) => b.createdAt.localeCompare(a.createdAt))
}

async function realCreate(_draft: NewOpportunityDraft): Promise<void> {
  void _draft
  throw new Error('INTEG gap: CreateRFQWithReference — wires at K5')
}

async function realDelete(_row: OpportunityRow): Promise<void> {
  void _row
  throw new Error('INTEG gap: DeleteRFQ (rfq rows) / DeleteOpportunity (pipeline rows) — wires at K5')
}

async function realCascadeDelete(_row: OpportunityRow, _reason: string): Promise<void> {
  void _row
  void _reason
  throw new Error(
    'INTEG gap: DeleteRFQWithCascade — wires at K5. (RFQ-sourced rows only; pipeline Opportunities ' +
      'have no cascade-delete binding — see Opportunities.parity.md.)',
  )
}

async function realCustomerOptions(): Promise<{ value: string; label: string }[]> {
  const rows = await ListCustomers(500, 0)
  return (rows ?? [])
    .map((c) => str((c as unknown as Record<string, unknown>).business_name))
    .filter(Boolean)
    .map((name) => ({ value: name, label: name }))
}

/* ---- public switched API (descriptor imports THESE) ---- */

export const fetchOpportunities = (): Promise<OpportunityRow[]> => pick(realFetch, mockFetch)()
export const createOpportunity = (draft: NewOpportunityDraft): Promise<void> =>
  pick(realCreate, mockCreate)(draft)
export const deleteOpportunity = (row: OpportunityRow): Promise<void> =>
  pick(realDelete, mockDelete)(row)
export const cascadeDeleteOpportunity = (row: OpportunityRow, reason: string): Promise<void> =>
  pick(realCascadeDelete, mockCascadeDelete)(row, reason)
export const opportunityCustomerOptions = (): Promise<{ value: string; label: string }[]> =>
  pick(realCustomerOptions, mockCustomerOptions)()
