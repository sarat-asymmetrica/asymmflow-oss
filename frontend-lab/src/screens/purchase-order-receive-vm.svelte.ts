/* Receive-Items-against-PO viewmodel — L5's reactive half for the bespoke
 * modal ejected off Purchase Orders via the kernel's ActionSpec.modal seam.
 * Loads a PO's line items, holds an editable rows array, validates each line
 * against the same guard ReceiveAgainstPO enforces server-side (so a bad line
 * is caught before the round trip, not just echoed back as a failed submit),
 * and assembles the GRNItem[] payload. The pure validate/build helpers are
 * exported standalone — no DOM, no Svelte runtime needed to exercise them (L5,
 * same split as book-bank-recon.svelte.ts). */

import {
  fetchPOReceiveLines,
  receiveAgainstPO,
  type GRNItemInput,
  type POReceiveLine,
} from '../bridge/purchase-orders'

export interface ReceiveRow {
  poItemId: string
  productCode: string
  description: string
  quantityOrdered: number
  quantityAlreadyReceived: number
  /** Editable — defaults to 0 (not receiving this line unless touched). */
  quantityReceiving: number
  quantityRejected: number
  rejectionReason: string
}

export function blankReceiveRow(): ReceiveRow {
  return {
    poItemId: '',
    productCode: '',
    description: '',
    quantityOrdered: 0,
    quantityAlreadyReceived: 0,
    quantityReceiving: 0,
    quantityRejected: 0,
    rejectionReason: '',
  }
}

function toRow(line: POReceiveLine): ReceiveRow {
  return { ...line, quantityReceiving: 0, quantityRejected: 0, rejectionReason: '' }
}

/** Per-line guard, mirroring purchase_order_service.go's ReceiveAgainstPO
 * validation: non-negative quantities, rejected never exceeds received, and
 * the line can never carry the PO's total received past what was ordered.
 * Returns null when the line is clean. */
export function validateRow(row: ReceiveRow): string | null {
  if (row.quantityReceiving < 0) return 'Quantity received cannot be negative.'
  if (row.quantityRejected < 0) return 'Quantity rejected cannot be negative.'
  if (row.quantityRejected > row.quantityReceiving) return 'Rejected quantity cannot exceed quantity received.'
  if (row.quantityAlreadyReceived + row.quantityReceiving > row.quantityOrdered) {
    return `Over-receipt: ${row.quantityAlreadyReceived} already received + ${row.quantityReceiving} now exceeds the ordered ${row.quantityOrdered}.`
  }
  return null
}

/** Assemble the GRNItem[] payload — only lines actually being received
 * (quantityReceiving > 0) are sent; an untouched zero-qty row is silently
 * dropped rather than posted as a no-op receipt. rejection_reason is omitted
 * entirely (not sent as ''/undefined) when blank — same conditional-spread
 * idiom as book-bank-recon.ts's optional `note` field. */
export function buildReceiveItems(rows: ReceiveRow[]): GRNItemInput[] {
  return rows
    .filter((r) => r.quantityReceiving > 0)
    .map((r) => ({
      po_item_id: r.poItemId,
      quantity_received: r.quantityReceiving,
      quantity_rejected: r.quantityRejected,
      ...(r.rejectionReason ? { rejection_reason: r.rejectionReason } : {}),
    }))
}

export class PurchaseOrderReceiveViewModel {
  poId: string
  rows = $state<ReceiveRow[]>([])
  loading = $state(true)
  error = $state<string | null>(null)
  submitting = $state(false)
  submitError = $state<string | null>(null)

  constructor(poId: string) {
    this.poId = poId
  }

  rowErrors = $derived(this.rows.map((r) => validateRow(r)))
  hasAnyReceiving = $derived(this.rows.some((r) => r.quantityReceiving > 0))
  hasErrors = $derived(this.rowErrors.some((e) => e != null))
  canSubmit = $derived(!this.loading && !this.submitting && this.hasAnyReceiving && !this.hasErrors)

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      const lines = await fetchPOReceiveLines(this.poId)
      this.rows = lines.map(toRow)
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
      this.rows = []
    } finally {
      this.loading = false
    }
  }

  /** Returns true on success (caller reloads the ledger + closes the modal). */
  async submit(): Promise<boolean> {
    if (!this.canSubmit) return false
    this.submitting = true
    this.submitError = null
    try {
      await receiveAgainstPO(this.poId, buildReceiveItems(this.rows))
      return true
    } catch (e) {
      this.submitError = e instanceof Error ? e.message : String(e)
      return false
    } finally {
      this.submitting = false
    }
  }
}
