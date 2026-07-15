import { describe, it, expect } from 'vitest'
import { sumField, isDebitCreditBalanced } from '../src/kernel/line-items'

interface JV {
  debit: number
  credit: number
}

describe('sumField', () => {
  it('sums a numeric field', () => {
    expect(sumField([{ debit: 10 }, { debit: 5 }, { debit: 0 }], (r) => r.debit)).toBe(15)
  })
  it('coerces non-numbers to 0', () => {
    expect(sumField([{ v: '3' }, { v: null }, { v: undefined }, { v: 'x' }], (r) => r.v)).toBe(3)
  })
  it('empty rows sum to 0', () => {
    expect(sumField<JV>([], (r) => r.debit)).toBe(0)
  })
})

describe('isDebitCreditBalanced', () => {
  const dr = (r: JV) => r.debit
  const cr = (r: JV) => r.credit
  it('balances equal positive totals', () => {
    expect(isDebitCreditBalanced([{ debit: 100, credit: 0 }, { debit: 0, credit: 100 }], dr, cr)).toBe(true)
  })
  it('rejects an all-zero voucher (both sides zero is NOT balanced)', () => {
    expect(isDebitCreditBalanced([{ debit: 0, credit: 0 }], dr, cr)).toBe(false)
  })
  it('rejects unequal totals', () => {
    expect(isDebitCreditBalanced([{ debit: 100, credit: 0 }, { debit: 0, credit: 90 }], dr, cr)).toBe(false)
  })
  it('rejects a one-sided voucher (debit only)', () => {
    expect(isDebitCreditBalanced([{ debit: 100, credit: 0 }], dr, cr)).toBe(false)
  })
  it('honours the BHD 3dp tolerance', () => {
    expect(isDebitCreditBalanced([{ debit: 100.0004, credit: 0 }, { debit: 0, credit: 100 }], dr, cr)).toBe(true)
    expect(isDebitCreditBalanced([{ debit: 100.01, credit: 0 }, { debit: 0, credit: 100 }], dr, cr)).toBe(false)
  })
})
