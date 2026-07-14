import { describe, expect, it } from 'vitest'
import { applyLedgerQuery, deriveFilterOptions } from '../src/kernel/ledger-core'
import type { LedgerDescriptor } from '../src/kernel/descriptor'

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
  it('derives distinct sorted values from data', () => {
    expect(deriveFilterOptions(descriptor.filters![0]!, rows)).toEqual([
      { value: 'Draft', label: 'Draft' },
      { value: 'Paid', label: 'Paid' },
    ])
  })
})
