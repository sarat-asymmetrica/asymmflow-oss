import { describe, expect, it } from 'vitest'
import { formatDate, formatMoney, formatNumber } from '../src/kernel/format'

describe('formatMoney', () => {
  it('BHD carries 3 decimals (fils)', () => {
    expect(formatMoney(1234.5)).toBe('BHD 1,234.500')
    expect(formatMoney(0.001)).toBe('BHD 0.001')
  })
  it('other currencies carry 2 decimals', () => {
    expect(formatMoney(1500, 'USD')).toBe('USD 1,500.00')
  })
  it('null/NaN render the em-dash, never "NaN"', () => {
    expect(formatMoney(null)).toBe('—')
    expect(formatMoney(Number.NaN)).toBe('—')
  })
})

describe('formatDate', () => {
  it('renders dd MMM yyyy', () => {
    expect(formatDate('2026-07-14')).toBe('14 Jul 2026')
  })
  it('null/invalid render the em-dash', () => {
    expect(formatDate(null)).toBe('—')
    expect(formatDate('garbage')).toBe('—')
  })
})

describe('formatNumber', () => {
  it('groups thousands', () => {
    expect(formatNumber(1234567)).toBe('1,234,567')
  })
})
