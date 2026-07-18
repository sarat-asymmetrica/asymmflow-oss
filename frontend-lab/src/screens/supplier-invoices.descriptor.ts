/* SupplierInvoicesScreen as a descriptor. K1 scope: read + summary only —
 * every mutating capability (New/Edit, 3-Way Match, Approve, Mark Paid) is a
 * financial hot-zone or a form-archetype SLOT, ledgered rather than rebuilt
 * loosely (see screens/parity/SupplierInvoices.parity.md).
 *
 * This row carries TWO independent status dimensions (match_status +
 * payment_status) — the kernel's `StatusSpec` is single-field, so
 * match_status drives the primary badge column/filter/summary distribution,
 * and payment_status renders as a plain toned text column instead of a
 * second badge (DataTable only badges one column per ledger — see
 * kernel/primitives/DataTable.svelte's `content === 'status' && status`
 * branch). Ledgered as an ENGINE gap (secondaryStatus) below. */

import type { ActionSpec, LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import type { Tone } from '$kernel/tones'
import {
  approveSupplierInvoice,
  fetchSupplierInvoices,
  markSupplierInvoicePaid,
  performThreeWayMatch,
  type SupplierInvoiceRow,
} from '../bridge/supplier-invoices'

// Settlement channels for the Mark-Paid capture form. A recorded reference +
// method is required (audit-trail discipline) — the payment lands as a real
// supplier-payment ledger entry server-side, not just a status flag.
const PAYMENT_METHODS = ['Bank Transfer', 'Cheque', 'Cash']

const markSupplierInvoicePaidForm: FormSpec<{ paymentMethod: string; paymentReference: string }> = {
  title: 'Record Supplier Invoice Payment',
  submitLabel: 'Record Payment',
  initial: () => ({ paymentMethod: 'Bank Transfer', paymentReference: '' }),
  fields: [
    {
      key: 'paymentMethod',
      label: 'Payment Method',
      kind: 'select',
      required: true,
      options: PAYMENT_METHODS.map((m) => ({ value: m, label: m })),
    },
    {
      key: 'paymentReference',
      label: 'Payment Reference',
      kind: 'text',
      required: true,
      placeholder: 'e.g. transfer confirmation / cheque no.',
    },
  ],
  submit: async (draft, row) => {
    const r = row as SupplierInvoiceRow
    await markSupplierInvoicePaid(r.id, draft.paymentReference, draft.paymentMethod)
  },
}

const supplierInvoiceActions: ActionSpec<SupplierInvoiceRow>[] = [
  {
    key: 'three-way-match',
    label: '3-Way Match',
    kind: 'row',
    // Re-runs PO ↔ GRN ↔ invoice verification and PERSISTS the resulting
    // match status server-side; reload then surfaces the new Match Status
    // badge + discrepancy reason (no announce toast — L7). Hidden once the
    // invoice is settled/rejected, where re-matching is meaningless.
    visible: (r) => r != null && r.status !== 'Paid' && r.status !== 'Rejected',
    run: async ({ row, reload }) => {
      if (!row) return
      await performThreeWayMatch(row.id)
      await reload()
    },
  },
  {
    key: 'approve',
    label: 'Approve',
    kind: 'row',
    // The server requires a clean 3-way match AND segregation of duties
    // (approver ≠ creator, approver derived from the session). Gate the button
    // on the same match precondition so it only shows when approval can succeed;
    // the SoD check stays server-side and surfaces its error honestly if hit.
    visible: (r) => r != null && r.status === 'Pending' && r.matchStatus === 'Matched',
    confirm: (r) =>
      `Approve ${r ? r.invoiceNumber : 'this invoice'} for payment? You are recorded as the approver (segregation of duties).`,
    run: async ({ row, reload }) => {
      if (!row) return
      await approveSupplierInvoice(row.id)
      await reload()
    },
  },
  {
    key: 'mark-paid',
    label: 'Mark Paid',
    kind: 'row',
    // Backend enforces the invoice is Approved before settling.
    visible: (r) => r != null && r.status === 'Approved',
    form: markSupplierInvoicePaidForm,
    run: () => {
      /* form action submits via markSupplierInvoicePaidForm; run is unused */
    },
  },
]

const MATCH_TONES: Record<string, Tone> = {
  Pending: 'warning',
  Matched: 'success',
  Discrepancy: 'danger',
  'Review Required': 'warning',
  Dispute: 'danger',
}

const PAYMENT_TONES: Record<string, Tone> = {
  Paid: 'success',
  Unpaid: 'neutral',
  Scheduled: 'info',
  Overdue: 'danger',
}

/** Days until due (negative = overdue); null when there's no due date. */
function daysUntilDue(r: SupplierInvoiceRow): number | null {
  if (!r.dueDate) return null
  const due = new Date(`${r.dueDate}T00:00:00`)
  if (Number.isNaN(due.getTime())) return null
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return Math.round((due.getTime() - today.getTime()) / 86_400_000)
}

function isOverdue(r: SupplierInvoiceRow): boolean {
  const d = daysUntilDue(r)
  return d != null && d < 0 && r.paymentStatus !== 'Paid'
}

/** payment_status, overridden to 'Overdue' the same way the old screen's
 * enrichInvoice() did — client-computed, never stored on the row. */
function effectivePaymentStatus(r: SupplierInvoiceRow): string {
  return isOverdue(r) ? 'Overdue' : r.paymentStatus
}

function dueDateTone(r: SupplierInvoiceRow): Tone {
  const d = daysUntilDue(r)
  if (d == null) return 'neutral'
  if (d < 0 && r.paymentStatus !== 'Paid') return 'danger'
  if (d <= 7 && r.paymentStatus !== 'Paid') return 'warning'
  return 'neutral'
}

export const supplierInvoicesDescriptor: LedgerDescriptor<SupplierInvoiceRow> = {
  entity: 'supplier-invoices',
  title: 'Supplier Invoices',
  fetch: fetchSupplierInvoices,
  id: (r) => r.id,
  searchText: (r) => `${r.invoiceNumber} ${r.supplierName}`,

  columns: [
    { key: 'invoiceNumber', label: 'Invoice #', content: 'code', value: (r) => r.invoiceNumber, minWidth: 130 },
    { key: 'supplierName', label: 'Supplier', content: 'name', value: (r) => r.supplierName, grow: true, minWidth: 220 },
    { key: 'purchaseOrderId', label: 'PO Reference', content: 'code', value: (r) => r.purchaseOrderId.slice(0, 8), minWidth: 110 },
    { key: 'grnId', label: 'GRN Reference', content: 'code', value: (r) => r.grnId.slice(0, 8), minWidth: 110 },
    { key: 'currency', label: 'Currency', content: 'text', value: (r) => r.currency, minWidth: 80 },
    { key: 'totalBhd', label: 'Amount (BHD)', content: 'money', value: (r) => r.totalBhd, minWidth: 130 },
    { key: 'matchStatus', label: 'Match Status', content: 'status', value: (r) => r.matchStatus, minWidth: 130 },
    {
      key: 'paymentStatus',
      label: 'Payment Status',
      // Regular toned text column, not a second badge — see file header.
      content: 'text',
      value: (r) => effectivePaymentStatus(r),
      tone: (r) => PAYMENT_TONES[effectivePaymentStatus(r)] ?? 'neutral',
      minWidth: 120,
    },
    {
      key: 'dueDate',
      label: 'Due Date',
      content: 'date',
      value: (r) => r.dueDate,
      tone: dueDateTone,
      minWidth: 110,
    },
  ],

  // Primary badge dimension — the 3-way-match verification gate.
  status: {
    value: (r) => r.matchStatus,
    tones: MATCH_TONES,
  },

  summary: {
    metrics: [
      { label: 'Supplier Invoices', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Total (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.totalBhd, 0),
      },
      {
        label: 'Match Rate',
        content: 'quantity',
        value: (rows) => {
          if (rows.length === 0) return 0
          const matched = rows.filter((r) => r.matchStatus === 'Matched').length
          return Math.round((matched / rows.length) * 1000) / 10
        },
        tone: (rows) => {
          if (rows.length === 0) return 'neutral'
          const rate = (rows.filter((r) => r.matchStatus === 'Matched').length / rows.length) * 100
          if (rate >= 80) return 'success'
          if (rate >= 50) return 'warning'
          return 'danger'
        },
      },
      {
        label: 'Overdue',
        content: 'quantity',
        value: (rows) => rows.filter(isOverdue).length,
        tone: (rows) => (rows.some(isOverdue) ? 'danger' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By match status',
      value: (r) => r.matchStatus,
      tones: MATCH_TONES,
    },
  },

  filters: [
    {
      key: 'matchStatus',
      label: 'Match Status',
      options: 'derive',
      deriveValue: (r) => r.matchStatus,
      predicate: (r, v) => r.matchStatus === v,
    },
    {
      key: 'paymentStatus',
      label: 'Payment Status',
      options: 'derive',
      deriveValue: (r) => effectivePaymentStatus(r),
      predicate: (r, v) => effectivePaymentStatus(r) === v,
    },
    // Simple derived-year filter — a stand-in for the old screen's
    // this_month/last_month/this_quarter/this_year dropdown, which needs a
    // real FilterSpec date-range primitive (ledgered as ENGINE below).
    {
      key: 'invoiceYear',
      label: 'Invoice Year',
      options: 'derive',
      deriveValue: (r) => r.invoiceDate.slice(0, 4),
      predicate: (r, v) => r.invoiceDate.slice(0, 4) === v,
    },
  ],

  // Per-status actions (R1.2): 3-Way Match (verify + persist), Approve (SoD),
  // Mark Paid (capture form). New/Edit + CreateSupplierInvoice remain ledgered
  // as struct-arg form SLOTs — see the parity doc. Row click still opens the
  // default column-list detail panel.
  actions: supplierInvoiceActions,

  emptyMessage: 'No supplier invoices yet. Match against a GRN once one is received.',
}
