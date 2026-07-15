import { describe, it, expect } from 'vitest'
import {
  allocationKey,
  totalAllocated,
  remainingToAllocate,
  isFullyAllocated,
  defaultAllocationAmount,
  type AllocationDraft,
} from '../src/kernel/allocation'

const draft = (amount: number, maxAmount = amount): AllocationDraft => ({
  key: `T:${amount}`,
  candidateId: String(amount),
  candidateType: 'T',
  label: 'x',
  amount,
  maxAmount,
})

describe('allocationKey', () => {
  it('composes type:id', () => {
    expect(allocationKey('CUSTOMER_INVOICE', 'inv-1')).toBe('CUSTOMER_INVOICE:inv-1')
  })
})

describe('totalAllocated / remainingToAllocate', () => {
  it('sums applied amounts', () => {
    expect(totalAllocated([draft(100), draft(50)])).toBe(150)
  })
  it('remainder is target minus allocated', () => {
    expect(remainingToAllocate(200, [draft(100), draft(50)])).toBe(50)
  })
  it('remainder goes negative when over-allocated', () => {
    expect(remainingToAllocate(100, [draft(80), draft(50)])).toBe(-30)
  })
})

describe('isFullyAllocated', () => {
  it('true within BHD 3dp tolerance', () => {
    expect(isFullyAllocated(100, [draft(99.9996)])).toBe(true)
  })
  it('false when under', () => {
    expect(isFullyAllocated(100, [draft(60)])).toBe(false)
  })
  it('false when over beyond tolerance', () => {
    expect(isFullyAllocated(100, [draft(100.01)])).toBe(false)
  })
})

describe('defaultAllocationAmount', () => {
  it('takes the full remainder when the candidate can absorb it', () => {
    expect(defaultAllocationAmount(70, 100)).toBe(70)
  })
  it('caps at the candidate ceiling', () => {
    expect(defaultAllocationAmount(200, 80)).toBe(80)
  })
  it('never goes negative when already over-allocated', () => {
    expect(defaultAllocationAmount(-30, 100)).toBe(0)
  })
})
