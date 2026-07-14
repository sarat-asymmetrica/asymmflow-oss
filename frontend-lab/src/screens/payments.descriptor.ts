/* PaymentsScreen as a descriptor — the Receipts sub-ledger only.
 * Old frontend: Receipts + Payment History co-located on one screen (recon
 * K1-B synthesis #1). Payment History stays ledgered (see Payments.parity.md)
 * until the kernel gets a multi-panel/composed-ledger archetype — this
 * descriptor renders the PRIMARY ledger, the one AR money-in sub-ledger. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import { fetchReceiptsPage, reverseReceipt, type ReceiptRow } from '../bridge/payments'

interface ReverseReceiptDraft {
  reason: string
}

const reverseReceiptForm: FormSpec<ReverseReceiptDraft> = {
  title: 'Reverse Receipt',
  submitLabel: 'Reverse',
  initial: () => ({ reason: '' }),
  fields: [
    {
      key: 'reason',
      label: 'Reason',
      kind: 'textarea',
      required: true,
      placeholder: 'Why is this receipt being reversed?',
    },
  ],
  // Row-aware submit (recon K1-B #114): reverseReceipt needs to know WHICH
  // receipt to mutate, not just the reason.
  submit: async (draft, row) => {
    const r = row as ReceiptRow
    await reverseReceipt(r.id, draft.reason)
  },
}

export const paymentsDescriptor: LedgerDescriptor<ReceiptRow> = {
  entity: 'payments',
  title: 'Payments — Receipts',
  fetch: () => fetchReceiptsPage(200, 0),
  fetchPage: fetchReceiptsPage,
  pageSize: 50,
  id: (r) => r.id,
  searchText: (r) => `${r.receiptNumber} ${r.customerName} ${r.reference}`,

  columns: [
    { key: 'receiptDate', label: 'Date', content: 'date', value: (r) => r.receiptDate, minWidth: 110 },
    { key: 'receiptNumber', label: 'Receipt #', content: 'code', value: (r) => r.receiptNumber, minWidth: 130 },
    { key: 'customerName', label: 'Customer', content: 'name', value: (r) => r.customerName, grow: true, minWidth: 220 },
    { key: 'amountBhd', label: 'Amount (BHD)', content: 'money', value: (r) => r.amountBhd, minWidth: 120 },
    { key: 'appliedAmountBhd', label: 'Applied (BHD)', content: 'money', value: (r) => r.appliedAmountBhd, minWidth: 120 },
    {
      key: 'unappliedAmountBhd',
      label: 'Unapplied (BHD)',
      content: 'money',
      value: (r) => r.unappliedAmountBhd,
      minWidth: 130,
      tone: (r) => (r.unappliedAmountBhd > 0.001 ? 'warning' : 'neutral'),
    },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 130 },
    { key: 'paymentMethod', label: 'Method', content: 'text', value: (r) => r.paymentMethod, minWidth: 120 },
    { key: 'reference', label: 'Reference', content: 'code', value: (r) => r.reference, minWidth: 130 },
  ],

  status: {
    value: (r) => r.status,
    tones: {
      OnAccount: 'info',
      PartiallyApplied: 'warning',
      Applied: 'success',
      Reversed: 'danger',
      // Unknown statuses render neutral by engine contract — never crash.
    },
  },

  summary: {
    metrics: [
      { label: 'Receipts', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Total Received (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.amountBhd, 0),
      },
      {
        label: 'Applied (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.appliedAmountBhd, 0),
      },
      {
        label: 'Unapplied (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.unappliedAmountBhd, 0),
        tone: (rows) => (rows.reduce((s, r) => s + r.unappliedAmountBhd, 0) > 0.001 ? 'warning' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By status',
      value: (r) => r.status,
      tones: {
        OnAccount: 'info',
        PartiallyApplied: 'warning',
        Applied: 'success',
        Reversed: 'danger',
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
      // Reverse Receipt — zero-application-only (recon K1-B #114). Record
      // Receipt (the one AR money-in creation path) and Apply Unapplied
      // Balance (invoice picker) stay ledgered — see Payments.parity.md.
      key: 'reverse',
      label: 'Reverse',
      kind: 'row',
      visible: (r) => r != null && r.appliedAmountBhd <= 0.001 && r.status !== 'Reversed',
      form: reverseReceiptForm,
      run: () => {
        /* form actions submit via their FormSpec; run is unused */
      },
    },
  ],

  emptyMessage: 'No receipts yet. Receipts are recorded against a customer invoice or on account.',
}
