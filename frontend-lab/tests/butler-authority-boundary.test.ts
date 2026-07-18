/* AI-authority boundary tripwire (owner ruling G1.1 — the butler SPLIT).
 *
 * The mechanical guarantee, mirroring the mesh gate's agent-rejection check:
 * the 4 approve-class bindings (ApprovePurchaseOrder, ApproveStockAdjustment,
 * ApproveSupplierInvoice, ApproveCostingSheet) can NEVER resolve to an
 * executable action from butler — every path that used to reach them now
 * redirects the human to the Approvals Queue and executes nothing. The 19
 * draft/update-class bindings, by contrast, DO resolve and execute (against the
 * mock here, since vitest has no Wails runtime — usingWails() is false, so the
 * bridge wrappers take their simulate-success path).
 *
 * A "Done — …" message means an action executed; an "Approvals Queue" message
 * means it was redirected and nothing ran. The two are mutually exclusive, so
 * the message alone proves whether the boundary held. */
import { describe, it, expect } from 'vitest'
import {
  executeButlerAction,
  RETIRED_APPROVE_BINDINGS,
  type ResolvedButlerAction,
} from '../src/screens/butler-actions'

function action(type: string, target: string, data: Record<string, unknown>): ResolvedButlerAction {
  return { type, target, label: '', data, requiresApproval: false, storedStatus: '', missingFields: [], invalidReason: '' }
}

describe('AI-authority boundary — approve-class bindings are retired', () => {
  const retiredCases: { name: string; action: ResolvedButlerAction }[] = [
    { name: 'approve purchase order', action: action('approve', 'purchase_order', { id: '123' }) },
    { name: 'approve stock adjustment', action: action('approve', 'stock_adjustment', { id: '5' }) },
    { name: 'approve supplier invoice', action: action('approve', 'supplier_invoice', { id: '9' }) },
    { name: 'approve costing sheet', action: action('approve', 'costing_sheet', { id: '7' }) },
    // The two "update to approved-status" paths that used to reach an approve
    // binding also redirect (stock-adjustment approve + costing status change).
    { name: 'update stock adjustment → approved', action: action('update', 'stock_adjustment', { id: '5', status: 'approved' }) },
    { name: 'update costing sheet status', action: action('update', 'costing_sheet', { id: '7', status: 'approved' }) },
  ]

  for (const { name, action: a } of retiredCases) {
    it(`redirects "${name}" to the Approvals Queue and executes nothing`, async () => {
      const { message } = await executeButlerAction(a)
      expect(message).toMatch(/Approvals Queue/i)
      // Never a completion — nothing was executed on the human's behalf.
      expect(message).not.toMatch(/^Done —/)
    })
  }

  it('exposes exactly the 4 retired approve-class binding names', () => {
    expect([...RETIRED_APPROVE_BINDINGS].sort()).toEqual(
      ['ApproveCostingSheet', 'ApprovePurchaseOrder', 'ApproveStockAdjustment', 'ApproveSupplierInvoice'],
    )
  })
})

describe('AI-authority boundary — draft/update-class bindings execute', () => {
  const wiredCases: { name: string; action: ResolvedButlerAction; verb: string }[] = [
    {
      name: 'create offer draft',
      action: action('create', 'offer', { customer_name: 'Alpha Controls', line_items: [{ equipment: 'Flow meter', quantity: 2, unit_price_bhd: 10 }] }),
      verb: 'create the offer draft',
    },
    {
      name: 'create follow-up',
      action: action('create', 'follow_up', { customer_name: 'Alpha Controls', title: 'Chase quote' }),
      verb: 'create the follow-up',
    },
    {
      name: 'mark offer won',
      action: action('approve', 'offer', { id: '3', customer_po: 'PO-778' }),
      verb: 'mark that offer as won',
    },
    {
      name: 'mark offer lost',
      action: action('reject', 'offer', { id: '3', reason: 'price' }),
      verb: 'mark that offer as lost',
    },
    {
      name: 'dispute supplier invoice',
      action: action('reject', 'supplier_invoice', { id: '9', reason: 'quantity mismatch' }),
      verb: 'dispute that supplier invoice',
    },
    {
      name: 'reject costing sheet',
      action: action('reject', 'costing_sheet', { id: '7', reason: 'margin too low' }),
      verb: 'reject that costing sheet',
    },
    {
      name: 'update PO status',
      action: action('update', 'purchase_order', { id: '2', status: 'Cancelled' }),
      verb: 'update that purchase order',
    },
    {
      name: 'update offer status',
      action: action('update', 'offer', { id: '4', status: 'sent' }),
      verb: 'update that offer',
    },
  ]

  for (const { name, action: a, verb } of wiredCases) {
    it(`executes "${name}" (mock success), never a redirect`, async () => {
      const { message } = await executeButlerAction(a)
      expect(message).toBe(`Done — ${verb}.`)
      expect(message).not.toMatch(/Approvals Queue/i)
    })
  }
})
