/* The pilot: InvoicesScreen as a descriptor.
 * Old frontend: 2,930 hand-written lines. This file: the whole screen. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import InvoiceReceiptModal from './InvoiceReceiptModal.svelte'
import {
  deleteInvoice,
  fetchInvoices,
  fetchInvoicesPage,
  sendInvoice,
  type InvoiceRow,
} from '../bridge'

export const invoicesDescriptor: LedgerDescriptor<InvoiceRow> = {
  entity: 'invoices',
  title: 'Invoices',
  fetch: fetchInvoices,
  fetchPage: fetchInvoicesPage,
  pageSize: 100,
  id: (r) => r.id,
  searchText: (r) => `${r.number} ${r.customer}`,

  columns: [
    { key: 'number', label: 'Invoice #', content: 'code', value: (r) => r.number, minWidth: 150 },
    { key: 'customer', label: 'Customer', content: 'name', value: (r) => r.customer, grow: true, minWidth: 220 },
    { key: 'division', label: 'Division', content: 'text', value: (r) => r.division, minWidth: 170 },
    { key: 'issueDate', label: 'Issued', content: 'date', value: (r) => r.issueDate, minWidth: 110 },
    { key: 'dueDate', label: 'Due', content: 'date', value: (r) => r.dueDate, minWidth: 110 },
    { key: 'amount', label: 'Amount', content: 'money', value: (r) => r.amount, currency: (r) => r.currency, minWidth: 150 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 120 },
  ],

  status: {
    value: (r) => r.status,
    tones: {
      Draft: 'neutral',
      Sent: 'info',
      Paid: 'success',
      Overdue: 'danger',
      Cancelled: 'neutral',
      // Unknown statuses render neutral by engine contract — never crash.
    },
  },

  // Visual-diversity strip: replaces the old screen's card grid with a dense
  // metric row + a status distribution bar, computed over the visible rows.
  summary: {
    metrics: [
      { label: 'Invoices', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Total (BHD)',
        content: 'money',
        value: (rows) => rows.filter((r) => r.currency === 'BHD').reduce((s, r) => s + r.amount, 0),
      },
      {
        label: 'Overdue',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.status === 'Overdue').length,
        tone: (rows) => (rows.some((r) => r.status === 'Overdue') ? 'danger' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By status',
      value: (r) => r.status,
      tones: {
        Draft: 'neutral',
        Sent: 'info',
        Paid: 'success',
        Overdue: 'danger',
        Cancelled: 'neutral',
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
    {
      // Derived from data in the lab; consumes the divisions registry store
      // at integration (L7 — no hardcoded division vocabulary in screens).
      key: 'division',
      label: 'Division',
      options: 'derive',
      deriveValue: (r) => r.division,
      predicate: (r, v) => r.division === v,
    },
  ],

  // Standalone invoice-create is RETIRED (owner ruling G1.3): invoices are
  // raised from an order (Orders → Create Invoice), never conjured standalone
  // on this ledger. No `new` action; the empty state points at Orders.
  actions: [
    {
      // R5: send a Draft invoice to the customer (Draft → Sent). Server rejects
      // a non-Draft status or an invoice with no line items.
      key: 'send',
      label: 'Send',
      kind: 'row',
      visible: (r) => r != null && r.status === 'Draft',
      confirm: (r) => `Send ${r ? (r as InvoiceRow).number : 'this invoice'} to the customer?`,
      run: async ({ row, reload }) => {
        if (!row) return
        await sendInvoice(row.id)
        await reload()
      },
    },
    {
      // G1.2 owner ruling: settlement is a receipt CAPTURE, never a status flip.
      // Opens a small receipt form (amount/date/method/reference) that records a
      // real customer receipt against the invoice (CreateCustomerReceipt with the
      // invoice bound → funds a Payment row + advances invoice state atomically).
      key: 'recordReceipt',
      label: 'Record Receipt',
      kind: 'row',
      visible: (r) => r != null && (r.status === 'Sent' || r.status === 'Overdue'),
      modal: InvoiceReceiptModal,
      run: () => {
        /* modal action: InvoiceReceiptModal owns its own submit; run is unused */
      },
    },
    {
      key: 'delete',
      label: 'Delete Draft',
      kind: 'row',
      visible: (r) => r != null && r.status === 'Draft',
      confirm: (r) => `Delete ${r ? (r as InvoiceRow).number : 'this draft'}? This cannot be undone.`,
      run: async ({ row, reload }) => {
        if (!row) return
        await deleteInvoice(row.id)
        await reload()
      },
    },
  ],

  emptyMessage: 'No invoices yet. Raise the first one from an order.',
}
