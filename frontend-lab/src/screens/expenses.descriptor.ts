/* ExpensesScreen as a descriptor. K1 scope: the PRIMARY panel = Entries,
 * rendered as a TABLE (the old screen's card/list layout reads as legacy —
 * the data is tabular). Recurring schedules, the Approvals queue, the
 * bank-candidate Workspace, and category/vendor master-data CRUD are all
 * separate panels on the old screen — ledgered as a multi-panel ENGINE gap.
 * See screens/parity/Expenses.parity.md.
 *
 * FIX, don't preserve (census-flagged gaps, owner's anti-jank mandate):
 * Approve/Post fired with NO confirmation on the old screen — added below.
 * Reject fired with a HARDCODED reason string ("Rejected from approvals
 * queue") — replaced with a row-aware reason form that actually captures
 * the operator's reason, closing a real audit-trail gap. */

import type { ActionSpec, LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import type { Tone } from '$kernel/tones'
import { nextStates } from '$kernel/ledger-core'
import {
  approveExpenseEntry,
  deleteExpenseEntry,
  fetchExpenseEntries,
  postExpenseEntry,
  rejectExpenseEntry,
  submitExpenseEntry,
  type ExpenseEntryRow,
} from '../bridge/expenses'

// draft -> submitted -> approved -> posted, with a submitted -> rejected
// side branch. Lowercase keys — this screen's vocabulary is lowercase,
// unlike every other ledger in this cluster (census gotcha).
const EXPENSE_STATUS_TRANSITIONS: Record<string, string[]> = {
  draft: ['submitted'],
  submitted: ['approved', 'rejected'],
  approved: ['posted'],
  rejected: [],
  posted: [],
}

const EXPENSE_STATUS_TONES: Record<string, Tone> = {
  draft: 'neutral',
  submitted: 'warning',
  approved: 'info',
  rejected: 'danger',
  posted: 'success',
}

const PAYMENT_STATUS_TONES: Record<string, Tone> = {
  paid: 'success',
  unpaid: 'neutral',
}

const rejectExpenseForm: FormSpec<{ reason: string }> = {
  title: 'Reject Expense',
  submitLabel: 'Reject',
  initial: () => ({ reason: '' }),
  fields: [
    {
      key: 'reason',
      label: 'Reason',
      kind: 'textarea',
      required: true,
      placeholder: 'Why is this expense being rejected?',
    },
  ],
  // Row-aware submit (ROW-AWARE FORMS): fixes the old screen's hardcoded
  // "Rejected from approvals queue" string with a real operator-supplied
  // reason, same audit-trail shape as Cancel PO / Reverse Receipt.
  submit: async (draft, row) => {
    const r = row as ExpenseEntryRow
    await rejectExpenseEntry(r.id, draft.reason)
  },
}

const rejectAction: ActionSpec<ExpenseEntryRow> = {
  key: 'reject',
  label: 'Reject',
  kind: 'row',
  visible: (r) => r != null && nextStates(r.status, EXPENSE_STATUS_TRANSITIONS).includes('rejected'),
  form: rejectExpenseForm,
  run: () => {
    /* form action submits via rejectExpenseForm; run is unused */
  },
}

export const expensesDescriptor: LedgerDescriptor<ExpenseEntryRow> = {
  entity: 'expenses',
  title: 'Expenses',
  fetch: fetchExpenseEntries,
  id: (r) => r.id,
  searchText: (r) => `${r.entryNumber} ${r.description} ${r.categoryName} ${r.vendorName}`,

  columns: [
    { key: 'entryNumber', label: 'Entry #', content: 'code', value: (r) => r.entryNumber, minWidth: 130 },
    { key: 'description', label: 'Description', content: 'name', value: (r) => r.description, grow: true, minWidth: 240 },
    { key: 'categoryName', label: 'Category', content: 'text', value: (r) => r.categoryName, minWidth: 150 },
    { key: 'vendorName', label: 'Vendor', content: 'text', value: (r) => r.vendorName, minWidth: 160 },
    { key: 'expenseDate', label: 'Date', content: 'date', value: (r) => r.expenseDate, minWidth: 110 },
    { key: 'totalAmount', label: 'Amount', content: 'money', value: (r) => r.totalAmount, currency: (r) => r.currency, minWidth: 130 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 110 },
    {
      key: 'paymentStatus',
      label: 'Payment',
      // Toned text, not a second badge — DataTable badges exactly one
      // column per ledger (see supplier-invoices.descriptor.ts's header
      // note; same ENGINE gap: dual-status rows need a `secondaryStatus`
      // concept StatusSpec doesn't have yet).
      content: 'text',
      value: (r) => r.paymentStatus,
      tone: (r) => PAYMENT_STATUS_TONES[r.paymentStatus] ?? 'neutral',
      minWidth: 100,
    },
  ],

  status: {
    value: (r) => r.status,
    tones: EXPENSE_STATUS_TONES,
    transitions: EXPENSE_STATUS_TRANSITIONS,
  },

  summary: {
    metrics: [
      {
        label: 'MTD Spend (BHD)',
        content: 'money',
        value: (rows) => {
          const now = new Date()
          return rows
            .filter((r) => {
              if (r.currency !== 'BHD' || !r.expenseDate) return false
              const d = new Date(`${r.expenseDate}T00:00:00`)
              return !Number.isNaN(d.getTime()) && d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth()
            })
            .reduce((s, r) => s + r.totalAmount, 0)
        },
      },
      { label: 'Submitted', content: 'quantity', value: (rows) => rows.filter((r) => r.status === 'submitted').length },
      { label: 'Approved', content: 'quantity', value: (rows) => rows.filter((r) => r.status === 'approved').length },
    ],
    distribution: {
      label: 'By status',
      value: (r) => r.status,
      tones: EXPENSE_STATUS_TONES,
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
      key: 'paymentStatus',
      label: 'Payment',
      options: 'derive',
      deriveValue: (r) => r.paymentStatus,
      predicate: (r, v) => r.paymentStatus === v,
    },
  ],

  actions: [
    {
      key: 'submit',
      label: 'Submit',
      kind: 'row',
      visible: (r) => r != null && nextStates(r.status, EXPENSE_STATUS_TRANSITIONS).includes('submitted'),
      confirm: (r) => `Submit ${r ? r.entryNumber : 'this entry'} for approval?`,
      run: async ({ row, reload }) => {
        if (!row) return
        await submitExpenseEntry(row.id)
        await reload()
      },
    },
    {
      key: 'approve',
      label: 'Approve',
      kind: 'row',
      visible: (r) => r != null && nextStates(r.status, EXPENSE_STATUS_TRANSITIONS).includes('approved'),
      // FIX, don't preserve: old screen fired this with NO confirmation.
      confirm: (r) => `Approve ${r ? r.entryNumber : 'this entry'}?`,
      run: async ({ row, reload }) => {
        if (!row) return
        await approveExpenseEntry(row.id)
        await reload()
      },
    },
    rejectAction,
    {
      key: 'post',
      label: 'Post',
      kind: 'row',
      visible: (r) => r != null && nextStates(r.status, EXPENSE_STATUS_TRANSITIONS).includes('posted'),
      // FIX, don't preserve: old screen fired this with NO confirmation,
      // despite posting to the GL.
      confirm: (r) => `Post ${r ? r.entryNumber : 'this entry'} to the ledger?`,
      run: async ({ row, reload }) => {
        if (!row) return
        await postExpenseEntry(row.id)
        await reload()
      },
    },
    {
      key: 'delete',
      label: 'Delete',
      kind: 'row',
      visible: (r) => r != null && r.status !== 'posted' && r.paymentStatus !== 'paid',
      confirm: (r) => `Delete ${r ? r.entryNumber : 'this entry'}? This cannot be undone.`,
      run: async ({ row, reload }) => {
        if (!row) return
        await deleteExpenseEntry(row.id)
        await reload()
      },
    },
  ],

  emptyMessage: 'No expenses yet. Create the first draft entry.',
}
