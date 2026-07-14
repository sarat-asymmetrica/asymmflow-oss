/* DeliveryNotesScreen as a descriptor. Old frontend's Create DN (fulfillment
 * sub-form), Dispatch-with-capture, and Confirm-POD are all ledgered here
 * (see parity doc) — K1 builds the list spine + a simplified status advance. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import {
  advanceDeliveryNoteStatus,
  deleteDeliveryNote,
  deliveryNoteTransitions,
  fetchDeliveryNotes,
  type DeliveryNoteRow,
} from '../bridge/delivery-notes'
import { nextStates } from '$kernel/ledger-core'

export const deliveryNotesDescriptor: LedgerDescriptor<DeliveryNoteRow> = {
  entity: 'delivery-notes',
  title: 'Delivery Notes',
  fetch: fetchDeliveryNotes,
  id: (r) => r.id,
  searchText: (r) => `${r.dnNumber} ${r.customerName} ${r.orderReference} ${r.driverName} ${r.vehicleNumber}`,

  columns: [
    { key: 'dnNumber', label: 'DN Number', content: 'code', value: (r) => r.dnNumber, minWidth: 140 },
    { key: 'orderReference', label: 'Order Reference', content: 'code', value: (r) => r.orderReference, minWidth: 140 },
    { key: 'customerName', label: 'Customer', content: 'name', value: (r) => r.customerName, grow: true, minWidth: 220 },
    { key: 'deliveryDate', label: 'Delivery Date', content: 'date', value: (r) => r.deliveryDate, minWidth: 130 },
    {
      key: 'deliverySeq',
      label: 'Delivery #',
      content: 'text',
      value: (r) => (r.totalDeliveries <= 1 ? 'Full' : `${r.deliverySeq} of ${r.totalDeliveries}`),
      minWidth: 100,
    },
    { key: 'transportMethod', label: 'Transport', content: 'text', value: (r) => r.transportMethod, minWidth: 140 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 130 },
  ],

  status: {
    value: (r) => r.status,
    tones: {
      Prepared: 'neutral',
      Dispatched: 'info',
      InTransit: 'info',
      Delivered: 'success',
      // Unknown statuses render neutral by engine contract — never crash.
    },
    transitions: deliveryNoteTransitions,
  },

  summary: {
    metrics: [
      { label: 'Delivery Notes', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'In Transit',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.status === 'InTransit').length,
        tone: (rows) => (rows.some((r) => r.status === 'InTransit') ? 'info' : 'neutral'),
      },
      {
        label: 'Delivered',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.status === 'Delivered').length,
      },
    ],
    distribution: {
      label: 'By status',
      value: (r) => r.status,
      tones: {
        Prepared: 'neutral',
        Dispatched: 'info',
        InTransit: 'info',
        Delivered: 'success',
      },
    },
  },

  filters: [
    {
      key: 'status',
      label: 'Status',
      options: 'derive',
      deriveValue: (r) => r.status,
      predicate: (r, v) => r.status === v,
    },
  ],

  actions: [
    {
      // Simplified status advance (Prepared→Dispatched→InTransit→Delivered).
      // Real Dispatch (driver/vehicle capture) and Confirm Delivery (POD
      // signature capture) are richer forms — ledgered per the brief, not
      // built in K1. This is a plain confirm-gated flip along the same chain.
      key: 'advance',
      label: 'Advance Status',
      kind: 'row',
      visible: (r) => r != null && nextStates(r.status, deliveryNoteTransitions).length > 0,
      confirm: (r) => {
        const row = r as DeliveryNoteRow | null
        const next = row ? nextStates(row.status, deliveryNoteTransitions)[0] : undefined
        return `Advance ${row?.dnNumber ?? 'this delivery note'} to ${next ?? 'the next status'}?`
      },
      run: async ({ row, reload }) => {
        if (!row) return
        await advanceDeliveryNoteStatus(row.id)
        await reload()
      },
    },
    {
      // Old screen allows Delete from any status; K1 restricts it to
      // Prepared (pre-dispatch) — an intentional safety improvement, not a
      // preserved-as-is behavior. See parity doc.
      key: 'delete',
      label: 'Delete',
      kind: 'row',
      visible: (r) => r != null && r.status === 'Prepared',
      confirm: (r) => `Delete ${r ? (r as DeliveryNoteRow).dnNumber : 'this delivery note'}? This cannot be undone.`,
      run: async ({ row, reload }) => {
        if (!row) return
        await deleteDeliveryNote(row.id)
        await reload()
      },
    },
  ],

  emptyMessage: 'No delivery notes yet. Create one from a confirmed order.',
}
