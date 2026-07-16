/* DeliveryNotesScreen as a descriptor. Old frontend's Create DN (fulfillment
 * sub-form), Dispatch-with-capture, and Confirm-POD are all ledgered here
 * (see parity doc) — K1 builds the list spine + a simplified status advance. */

import type { ActionSpec, LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import {
  confirmDeliveryNote,
  deleteDeliveryNote,
  deliveryNoteTransitions,
  dispatchDeliveryNote,
  fetchDeliveryNotes,
  type DeliveryNoteRow,
} from '../bridge/delivery-notes'

// R5 capture forms — the real backend flow is Prepared → Dispatched → Delivered.
const dispatchForm: FormSpec<{ driverName: string; vehicleNumber: string }> = {
  title: 'Dispatch Delivery Note',
  submitLabel: 'Dispatch',
  initial: (row) => {
    const r = row as DeliveryNoteRow | undefined
    return { driverName: r?.driverName ?? '', vehicleNumber: r?.vehicleNumber ?? '' }
  },
  fields: [
    { key: 'driverName', label: 'Driver Name', kind: 'text', required: true, placeholder: 'Who is driving the delivery?' },
    { key: 'vehicleNumber', label: 'Vehicle Number', kind: 'text', required: true, placeholder: 'e.g. plate / fleet no.' },
  ],
  submit: async (draft, row) => {
    const r = row as DeliveryNoteRow
    await dispatchDeliveryNote(r.id, draft.driverName, draft.vehicleNumber)
  },
}

const confirmDeliveryForm: FormSpec<{ signedBy: string }> = {
  title: 'Confirm Delivery',
  submitLabel: 'Confirm Delivery',
  initial: () => ({ signedBy: '' }),
  fields: [
    {
      key: 'signedBy',
      label: 'Received / Signed By',
      kind: 'text',
      required: true,
      placeholder: 'Name of the person who received the goods',
    },
  ],
  submit: async (draft, row) => {
    const r = row as DeliveryNoteRow
    await confirmDeliveryNote(r.id, draft.signedBy)
  },
}

const dispatchAction: ActionSpec<DeliveryNoteRow> = {
  key: 'dispatch',
  label: 'Dispatch',
  kind: 'row',
  // Server accepts dispatch only from Prepared.
  visible: (r) => r != null && r.status === 'Prepared',
  form: dispatchForm,
  run: () => {
    /* submitted via dispatchForm */
  },
}

const confirmDeliveryAction: ActionSpec<DeliveryNoteRow> = {
  key: 'confirm-delivery',
  label: 'Confirm Delivery',
  kind: 'row',
  // Server accepts confirmation only from Dispatched.
  visible: (r) => r != null && r.status === 'Dispatched',
  form: confirmDeliveryForm,
  run: () => {
    /* submitted via confirmDeliveryForm */
  },
}

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
    // R5: the real two-step fulfillment flow, each with its capture form.
    dispatchAction,
    confirmDeliveryAction,
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
