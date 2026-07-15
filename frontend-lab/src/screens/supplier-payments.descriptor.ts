/* SupplierPaymentsScreen as a descriptor. K1 scope: the merged ledger
 * (Supplier Invoice settlements + Expense settlements, tagged by `source`)
 * + Delete. Record Payment (FX-aware) and Edit (locked linkage) are
 * financial-hot-zone SLOTs, ledgered rather than rebuilt loosely — see
 * screens/parity/SupplierPayments.parity.md.
 *
 * No formal status field on this ledger (it's a completed-transaction
 * record, like Invoices' Payment History sub-ledger) — `source` doubles as
 * the descriptor's `status` badge, which also gives it filter/summary
 * distribution support for free. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import type { Tone } from '$kernel/tones'
import { deleteSupplierPayment, fetchSupplierPayments, type SupplierPaymentRow } from '../bridge/supplier-payments'

const SOURCE_TONES: Record<string, Tone> = {
  'Supplier Invoice': 'info',
  Expense: 'warning',
}

export const supplierPaymentsDescriptor: LedgerDescriptor<SupplierPaymentRow> = {
  entity: 'supplier-payments',
  title: 'Supplier Payments',
  fetch: fetchSupplierPayments,
  id: (r) => r.id,
  searchText: (r) => `${r.invoiceNumber} ${r.supplierName} ${r.reference}`,

  columns: [
    { key: 'source', label: 'Source', content: 'status', value: (r) => r.source, minWidth: 130 },
    { key: 'paymentDate', label: 'Date', content: 'date', value: (r) => r.paymentDate, minWidth: 110 },
    { key: 'supplierName', label: 'Supplier', content: 'name', value: (r) => r.supplierName, grow: true, minWidth: 220 },
    { key: 'invoiceNumber', label: 'Invoice #', content: 'code', value: (r) => r.invoiceNumber, minWidth: 140 },
    { key: 'amountBhd', label: 'Amount (BHD)', content: 'money', value: (r) => r.amountBhd, minWidth: 140 },
    { key: 'currency', label: 'Currency', content: 'text', value: (r) => r.currency, minWidth: 80 },
    { key: 'paymentMethod', label: 'Method', content: 'text', value: (r) => r.paymentMethod, minWidth: 130 },
    { key: 'reference', label: 'Reference', content: 'code', value: (r) => r.reference, minWidth: 140 },
  ],

  status: {
    value: (r) => r.source,
    tones: SOURCE_TONES,
  },

  summary: {
    metrics: [
      { label: 'Payments', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Total Paid (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.amountBhd, 0),
      },
      {
        label: 'Expense Settlements',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.source === 'Expense').length,
        tone: (rows) => (rows.some((r) => r.source === 'Expense') ? 'warning' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By source',
      value: (r) => r.source,
      tones: SOURCE_TONES,
    },
  },

  filters: [
    {
      key: 'source',
      label: 'Source',
      options: 'derive',
      deriveValue: (r) => r.source,
      predicate: (r, v) => r.source === v,
    },
  ],

  actions: [
    {
      key: 'delete',
      label: 'Delete',
      kind: 'row',
      // Expense-settlement rows are "managed from Expenses" in the old
      // screen (isExpenseSettlement guard) — Delete stays scoped to real
      // SupplierPayment rows only.
      visible: (r) => r != null && r.source === 'Supplier Invoice',
      confirm: (r) => `Delete payment ${r ? r.reference || r.invoiceNumber : 'this payment'}? This cannot be undone.`,
      run: async ({ row, reload }) => {
        if (!row) return
        await deleteSupplierPayment(row.id)
        await reload()
      },
    },
  ],

  emptyMessage: 'No supplier payments yet. Record one against an approved invoice.',
}
