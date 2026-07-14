/* The pilot: InvoicesScreen as a descriptor.
 * Old frontend: 2,930 hand-written lines. This file: the whole screen. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import { fetchInvoices, markInvoicePaid, type InvoiceRow } from '../bridge/mock'

export const invoicesDescriptor: LedgerDescriptor<InvoiceRow> = {
  entity: 'invoices',
  title: 'Invoices',
  fetch: fetchInvoices,
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

  actions: [
    {
      key: 'new',
      label: '+ New Invoice',
      kind: 'screen',
      run: () => {
        /* form archetype arrives in a later wave */
      },
    },
    {
      key: 'markPaid',
      label: 'Mark Paid',
      kind: 'row',
      visible: (r) => r != null && (r.status === 'Sent' || r.status === 'Overdue'),
      run: async ({ row, reload }) => {
        if (!row) return
        await markInvoicePaid(row.id)
        await reload()
      },
    },
  ],

  emptyMessage: 'No invoices yet. Raise the first one from an order.',
}
