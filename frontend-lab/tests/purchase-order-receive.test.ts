/* Receive-Items-against-PO pure logic — validateRow mirrors the server guard
 * (purchase_order_service.go's ReceiveAgainstPO: non-negative quantities,
 * rejected <= received, and already_received + received <= ordered) and
 * buildReceiveItems assembles the GRNItem-shaped wire payload. Exercised
 * headless (L5) — no Svelte runtime needed. */
import { describe, expect, it } from 'vitest'
import { validateRow, buildReceiveItems, type ReceiveRow } from '../src/screens/purchase-order-receive-vm.svelte'

function row(overrides: Partial<ReceiveRow> = {}): ReceiveRow {
  return {
    poItemId: 'item-1',
    productCode: 'CODE-1',
    description: 'A line item',
    quantityOrdered: 10,
    quantityAlreadyReceived: 0,
    quantityReceiving: 0,
    quantityRejected: 0,
    rejectionReason: '',
    ...overrides,
  }
}

describe('validateRow', () => {
  it('accepts an untouched line (0/0)', () => {
    expect(validateRow(row())).toBeNull()
  })

  it('accepts a clean partial receive with a rejection', () => {
    expect(validateRow(row({ quantityReceiving: 4, quantityRejected: 1 }))).toBeNull()
  })

  it('rejects negative quantityReceiving', () => {
    expect(validateRow(row({ quantityReceiving: -1 }))).toMatch(/negative/i)
  })

  it('rejects negative quantityRejected', () => {
    expect(validateRow(row({ quantityReceiving: 2, quantityRejected: -1 }))).toMatch(/negative/i)
  })

  it('rejects quantityRejected exceeding quantityReceiving', () => {
    expect(validateRow(row({ quantityReceiving: 2, quantityRejected: 3 }))).toMatch(/exceed/i)
  })

  it('rejects an over-receipt against already-received + ordered', () => {
    const err = validateRow(row({ quantityAlreadyReceived: 8, quantityOrdered: 10, quantityReceiving: 5 }))
    expect(err).toMatch(/over-receipt/i)
  })

  it('allows receiving exactly the remaining quantity (boundary, not over)', () => {
    expect(validateRow(row({ quantityAlreadyReceived: 8, quantityOrdered: 10, quantityReceiving: 2 }))).toBeNull()
  })

  it('rejects a fully-received line receiving any further quantity', () => {
    const err = validateRow(row({ quantityAlreadyReceived: 20, quantityOrdered: 20, quantityReceiving: 1 }))
    expect(err).toMatch(/over-receipt/i)
  })
})

describe('buildReceiveItems', () => {
  it('includes only lines with quantityReceiving > 0', () => {
    const rows = [
      row({ poItemId: 'a', quantityReceiving: 5 }),
      row({ poItemId: 'b', quantityReceiving: 0 }),
      row({ poItemId: 'c', quantityReceiving: 0.001 }),
    ]
    const items = buildReceiveItems(rows)
    expect(items.map((i) => i.po_item_id)).toEqual(['a', 'c'])
  })

  it('shapes each item as a GRNItemInput (po_item_id/quantity_received/quantity_rejected)', () => {
    const items = buildReceiveItems([row({ poItemId: 'a', quantityReceiving: 5, quantityRejected: 2 })])
    expect(items).toEqual([{ po_item_id: 'a', quantity_received: 5, quantity_rejected: 2 }])
  })

  it('omits the rejection_reason key entirely when blank (not sent as \'\'/undefined)', () => {
    const items = buildReceiveItems([row({ poItemId: 'a', quantityReceiving: 5, rejectionReason: '' })])
    expect(items[0]).not.toHaveProperty('rejection_reason')
  })

  it('carries a provided rejection reason through verbatim', () => {
    const items = buildReceiveItems([row({ poItemId: 'a', quantityReceiving: 5, quantityRejected: 1, rejectionReason: 'Damaged crate on arrival' })])
    expect(items[0]!.rejection_reason).toBe('Damaged crate on arrival')
  })

  it('returns an empty array when no line is being received', () => {
    expect(buildReceiveItems([row(), row({ poItemId: 'b' })])).toEqual([])
  })
})
