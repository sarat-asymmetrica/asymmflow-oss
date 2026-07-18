import { describe, expect, it } from 'vitest'
import { isBlank, validateForm, visibleFields } from '../src/kernel/form-core'
import type { FormSpec } from '../src/kernel/form'

interface Draft {
  customer: string
  issueDate: string
  dueDate: string
  amount: number | null
  notes: string
}

const spec: FormSpec<Draft> = {
  title: 'Test',
  initial: () => ({ customer: '', issueDate: '', dueDate: '', amount: null, notes: '' }),
  fields: [
    { key: 'customer', label: 'Customer', kind: 'select', required: true },
    { key: 'issueDate', label: 'Issue date', kind: 'date', required: true },
    {
      key: 'dueDate',
      label: 'Due date',
      kind: 'date',
      required: true,
      validate: (v, d) =>
        d.issueDate && typeof v === 'string' && v < d.issueDate ? 'Due before issue' : null,
    },
    {
      key: 'amount',
      label: 'Amount',
      kind: 'number',
      required: true,
      validate: (v) => (typeof v === 'number' && v <= 0 ? 'Must be positive' : null),
    },
    // Conditional: only visible when customer chosen — must not validate before.
    { key: 'notes', label: 'Notes', kind: 'text', required: true, visible: (d) => d.customer !== '' },
  ],
  submit: async () => {},
}

describe('isBlank', () => {
  it('treats empty string, whitespace, null, undefined, NaN as blank', () => {
    expect(isBlank('')).toBe(true)
    expect(isBlank('   ')).toBe(true)
    expect(isBlank(null)).toBe(true)
    expect(isBlank(undefined)).toBe(true)
    expect(isBlank(Number.NaN)).toBe(true)
    expect(isBlank(0)).toBe(false)
    expect(isBlank('x')).toBe(false)
  })
})

describe('validateForm', () => {
  it('flags all required blanks — but never hidden fields', () => {
    const errors = validateForm(spec, spec.initial())
    expect(errors).toHaveProperty('customer')
    expect(errors).toHaveProperty('issueDate')
    expect(errors).toHaveProperty('amount')
    expect(errors).not.toHaveProperty('notes') // hidden while customer is blank
  })

  it('hidden field becomes required once its condition reveals it', () => {
    const draft: Draft = {
      customer: 'Gulf',
      issueDate: '2026-07-01',
      dueDate: '2026-07-31',
      amount: 100,
      notes: '',
    }
    expect(validateForm(spec, draft)).toHaveProperty('notes')
  })

  it('cross-field validation: due date before issue date', () => {
    const draft: Draft = {
      customer: 'Gulf',
      issueDate: '2026-07-14',
      dueDate: '2026-07-01',
      amount: 100,
      notes: 'n',
    }
    expect(validateForm(spec, draft).dueDate).toBe('Due before issue')
  })

  it('value validation: non-positive amount rejected', () => {
    const draft: Draft = {
      customer: 'Gulf',
      issueDate: '2026-07-01',
      dueDate: '2026-07-31',
      amount: 0,
      notes: 'n',
    }
    expect(validateForm(spec, draft).amount).toBe('Must be positive')
  })

  it('a fully valid draft returns zero errors', () => {
    const draft: Draft = {
      customer: 'Gulf',
      issueDate: '2026-07-01',
      dueDate: '2026-07-31',
      amount: 12.345,
      notes: 'ok',
    }
    expect(validateForm(spec, draft)).toEqual({})
  })
})

describe('visibleFields', () => {
  it('filters by the live draft', () => {
    expect(visibleFields(spec, spec.initial()).map((f) => f.key)).not.toContain('notes')
    expect(
      visibleFields(spec, { ...spec.initial(), customer: 'Gulf' }).map((f) => f.key),
    ).toContain('notes')
  })
})
