import { describe, expect, it } from 'vitest'
import {
  applyLedgerQuery,
  computeSummary,
  deriveFilterOptions,
  nextStates,
} from '../src/kernel/ledger-core'
import type { LedgerDescriptor, SummarySpec } from '../src/kernel/descriptor'

interface Doc {
  id: string
  number: string
  customer: string
  status: string
}

const rows: Doc[] = [
  { id: '1', number: 'INV-001', customer: 'Gulf Fabrication', status: 'Draft' },
  { id: '2', number: 'INV-002', customer: 'Manama Process', status: 'Paid' },
  { id: '3', number: 'INV-003', customer: 'مؤسسة الخليج', status: 'Paid' },
]

const descriptor: LedgerDescriptor<Doc> = {
  entity: 'docs',
  title: 'Docs',
  fetch: async () => rows,
  id: (r) => r.id,
  searchText: (r) => `${r.number} ${r.customer}`,
  columns: [],
  filters: [
    {
      key: 'status',
      label: 'Status',
      options: 'derive',
      deriveValue: (r) => r.status,
      predicate: (r, v) => r.status === v,
    },
  ],
}

describe('applyLedgerQuery', () => {
  it('returns all rows for the empty query', () => {
    expect(applyLedgerQuery(descriptor, rows, { search: '', filters: {} })).toHaveLength(3)
  })

  it('search is case-insensitive and sweeps declared fields', () => {
    expect(applyLedgerQuery(descriptor, rows, { search: 'gulf', filters: {} })).toHaveLength(1)
    expect(applyLedgerQuery(descriptor, rows, { search: 'inv-00', filters: {} })).toHaveLength(3)
  })

  it('search matches RTL text', () => {
    expect(applyLedgerQuery(descriptor, rows, { search: 'مؤسسة', filters: {} })).toHaveLength(1)
  })

  it('filters AND with search; empty-string filter means All', () => {
    expect(
      applyLedgerQuery(descriptor, rows, { search: 'inv', filters: { status: 'Paid' } }),
    ).toHaveLength(2)
    expect(
      applyLedgerQuery(descriptor, rows, { search: '', filters: { status: '' } }),
    ).toHaveLength(3)
  })

  it('unknown filter value matches nothing (never throws)', () => {
    expect(
      applyLedgerQuery(descriptor, rows, { search: '', filters: { status: 'Nope' } }),
    ).toHaveLength(0)
  })
})

describe('deriveFilterOptions', () => {
  it('derives distinct sorted values from data with live counts', () => {
    expect(deriveFilterOptions(descriptor.filters![0]!, rows)).toEqual([
      { value: 'Draft', label: 'Draft', count: 1 },
      { value: 'Paid', label: 'Paid', count: 2 },
    ])
  })

  it('counts static options via the filter predicate', () => {
    const staticFilter = {
      key: 'status',
      label: 'Status',
      options: [
        { value: 'Draft', label: 'Draft' },
        { value: 'Void', label: 'Void' },
      ],
      predicate: (r: Doc, v: string) => r.status === v,
    }
    expect(deriveFilterOptions(staticFilter, rows)).toEqual([
      { value: 'Draft', label: 'Draft', count: 1 },
      { value: 'Void', label: 'Void', count: 0 },
    ])
  })
})

describe('nextStates', () => {
  const transitions = {
    Draft: ['Sent', 'Cancelled'],
    Sent: ['Paid'],
    Paid: [],
  }
  it('returns declared next statuses', () => {
    expect(nextStates('Draft', transitions)).toEqual(['Sent', 'Cancelled'])
  })
  it('returns [] for terminal or unknown status, and when no table', () => {
    expect(nextStates('Paid', transitions)).toEqual([])
    expect(nextStates('Ghost', transitions)).toEqual([])
    expect(nextStates('Draft', undefined)).toEqual([])
  })
})

describe('computeSummary', () => {
  const spec: SummarySpec<Doc> = {
    metrics: [
      { label: 'Count', content: 'quantity', value: (rs) => rs.length },
      {
        label: 'Paid share',
        content: 'text',
        value: (rs) => `${rs.filter((r) => r.status === 'Paid').length}/${rs.length}`,
        tone: (rs) => (rs.some((r) => r.status === 'Draft') ? 'warning' : 'success'),
      },
    ],
    distribution: {
      label: 'By status',
      value: (r) => r.status,
      tones: { Draft: 'neutral', Paid: 'success' },
    },
  }

  it('returns null when no spec', () => {
    expect(computeSummary(undefined, rows)).toBeNull()
  })

  it('reduces metrics and tones over the rows', () => {
    const s = computeSummary(spec, rows)!
    expect(s.metrics[0]!.value).toBe(3)
    expect(s.metrics[1]!.value).toBe('2/3')
    expect(s.metrics[1]!.tone).toBe('warning')
  })

  it('builds a distribution sorted by count with percentages and tones', () => {
    const s = computeSummary(spec, rows)!
    expect(s.distribution!.total).toBe(3)
    expect(s.distribution!.segments).toEqual([
      { key: 'Paid', count: 2, tone: 'success', pct: (2 / 3) * 100 },
      { key: 'Draft', count: 1, tone: 'neutral', pct: (1 / 3) * 100 },
    ])
  })

  it('unknown distribution buckets fall back to neutral tone', () => {
    const oddRows: Doc[] = [{ id: '9', number: 'X', customer: 'Y', status: 'Weird' }]
    const s = computeSummary(spec, oddRows)!
    expect(s.distribution!.segments[0]).toMatchObject({ key: 'Weird', tone: 'neutral' })
  })
})
