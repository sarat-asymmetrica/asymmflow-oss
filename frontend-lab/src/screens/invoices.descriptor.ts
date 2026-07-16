/* The pilot: InvoicesScreen as a descriptor.
 * Old frontend: 2,930 hand-written lines. This file: the whole screen. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import {
  createInvoice,
  customerOptions,
  deleteInvoice,
  divisionOptions,
  fetchInvoices,
  fetchInvoicesPage,
  markInvoicePaid,
  sendInvoice,
  type InvoiceRow,
  type NewInvoiceDraft,
} from '../bridge'

const newInvoiceForm: FormSpec<NewInvoiceDraft> = {
  title: 'New Invoice',
  submitLabel: 'Create Draft',
  initial: () => ({
    customer: '',
    division: divisionOptions()[0]?.value ?? '',
    issueDate: '',
    dueDate: '',
    amount: null,
    currency: 'BHD',
    notes: '',
  }),
  fields: [
    { key: 'customer', label: 'Customer', kind: 'select', required: true, options: customerOptions },
    // Lazy (async fn), NOT a captured array: read at form-open so the real
    // GetDivisionRegistry (loaded during boot) wins over the synthetic fallback.
    { key: 'division', label: 'Division', kind: 'select', required: true, options: async () => divisionOptions() },
    { key: 'issueDate', label: 'Issue date', kind: 'date', required: true },
    {
      key: 'dueDate',
      label: 'Due date',
      kind: 'date',
      required: true,
      validate: (v, draft) =>
        draft.issueDate && typeof v === 'string' && v < draft.issueDate
          ? 'Due date cannot precede the issue date'
          : null,
    },
    {
      key: 'amount',
      label: 'Amount',
      kind: 'number',
      required: true,
      step: '0.001',
      validate: (v) => (typeof v === 'number' && v <= 0 ? 'Amount must be positive' : null),
    },
    {
      key: 'currency',
      label: 'Currency',
      kind: 'select',
      required: true,
      options: [
        { value: 'BHD', label: 'BHD' },
        { value: 'USD', label: 'USD' },
      ],
    },
    { key: 'notes', label: 'Notes', kind: 'textarea', placeholder: 'Optional' },
  ],
  submit: (draft) => createInvoice(draft),
}

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

  actions: [
    {
      key: 'new',
      label: '+ New Invoice',
      kind: 'screen',
      form: newInvoiceForm,
      run: () => {
        /* form actions submit via their FormSpec; run is unused */
      },
    },
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
