/* Credit Notes as a standalone descriptor. Old frontend: a sub-ledger
 * squatting inside InvoicesScreen (PARITY_INVOICES #14) — this is that
 * ~80-line standalone screen, linked from Invoices by action at INTEG. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import { applyCreditNote, fetchCreditNotesPage, type CreditNoteRow } from '../bridge/credit-notes'

export const creditNotesDescriptor: LedgerDescriptor<CreditNoteRow> = {
  entity: 'credit-notes',
  title: 'Credit Notes',
  fetch: () => fetchCreditNotesPage(200, 0),
  fetchPage: fetchCreditNotesPage,
  pageSize: 50,
  id: (r) => r.id,
  searchText: (r) => `${r.creditNoteNumber} ${r.invoiceNumber} ${r.customerName}`,

  columns: [
    { key: 'creditNoteNumber', label: 'CN Number', content: 'code', value: (r) => r.creditNoteNumber, minWidth: 130 },
    { key: 'creditNoteDate', label: 'Date', content: 'date', value: (r) => r.creditNoteDate, minWidth: 110 },
    { key: 'invoiceNumber', label: 'Invoice Ref', content: 'code', value: (r) => r.invoiceNumber, minWidth: 130 },
    { key: 'customerName', label: 'Customer', content: 'name', value: (r) => r.customerName, grow: true, minWidth: 220 },
    { key: 'grandTotalBhd', label: 'Amount (BHD)', content: 'money', value: (r) => r.grandTotalBhd, minWidth: 130 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 120 },
  ],

  status: {
    value: (r) => r.status,
    tones: {
      Draft: 'neutral',
      Issued: 'info',
      Applied: 'success',
      // Unknown statuses render neutral by engine contract — never crash.
    },
  },

  summary: {
    metrics: [
      { label: 'Credit Notes', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Total (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.grandTotalBhd, 0),
      },
    ],
    distribution: {
      label: 'By status',
      value: (r) => r.status,
      tones: {
        Draft: 'neutral',
        Issued: 'info',
        Applied: 'success',
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
      // Apply reduces AR — the old screen fired this with NO confirm
      // (recon K1-B #391/#397). Adding a confirm here is a "fix, don't
      // preserve" call, the same class as Reverse Receipt requiring one.
      // Issue Credit Note (invoice picker + line items) and PDF stay
      // ledgered — see CreditNotes.parity.md.
      key: 'apply',
      label: 'Apply',
      kind: 'row',
      visible: (r) => r != null && r.status !== 'Applied',
      confirm: (r) =>
        `Apply ${r ? (r as CreditNoteRow).creditNoteNumber : 'this credit note'} against ${r ? (r as CreditNoteRow).invoiceNumber || 'its invoice' : 'the invoice'}? This reduces the customer's outstanding balance.`,
      run: async ({ row, reload }) => {
        if (!row) return
        await applyCreditNote(row.id)
        await reload()
      },
    },
  ],

  emptyMessage: 'No credit notes yet. Issue one from an invoice.',
}
