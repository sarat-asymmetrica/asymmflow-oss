/* ChequeRegisterScreen as a descriptor. K1 scope: the PRIMARY ledger =
 * Outstanding cheques (list, status machine, Mark Stale, Cancel-with-reason).
 * The old screen's other two sub-views (cheque-book Registers, Stale-only
 * tab) are THREE separate bank-account-scoped fetches, not one dataset
 * filtered three ways — ledgered as a multi-panel ENGINE gap. Issue Cheque
 * and Mark Cleared's bank-statement-line picker are financial hot-zone
 * SLOTs, also ledgered — see screens/parity/ChequeRegister.parity.md. */

import type { ActionSpec, LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import type { Tone } from '$kernel/tones'
import { nextStates } from '$kernel/ledger-core'
import {
  cancelCheque,
  fetchChequeRegister,
  markChequeStale,
  type OutstandingChequeRow,
} from '../bridge/cheque-register'

// Mirrors pkg/finance/cheque/cheque.go's status-changing methods exactly:
// MarkCleared/MarkStale/MarkBounced all gate on status IN (ISSUED,
// PRESENTED); Cancel gates on ISSUED only — source of truth, do not let
// this drift independently (K1-B synthesis #2).
const CHEQUE_STATUS_TRANSITIONS: Record<string, string[]> = {
  ISSUED: ['PRESENTED', 'CLEARED', 'STALE', 'CANCELLED', 'BOUNCED'],
  PRESENTED: ['CLEARED', 'STALE', 'BOUNCED'],
  CLEARED: [],
  STALE: [],
  CANCELLED: [],
  BOUNCED: [],
}

const CHEQUE_STATUS_TONES: Record<string, Tone> = {
  ISSUED: 'info',
  PRESENTED: 'warning',
  CLEARED: 'success',
  STALE: 'danger',
  CANCELLED: 'neutral',
  BOUNCED: 'danger',
}

/** Days since issue. The backend's own StaleCheques query (cheque.go:315)
 * uses a 6-month window, so 150 days is the "approaching stale" amber zone
 * ahead of that boundary. */
function ageDays(r: OutstandingChequeRow): number {
  if (!r.issuedDate) return 0
  const issued = new Date(`${r.issuedDate}T00:00:00`)
  if (Number.isNaN(issued.getTime())) return 0
  return Math.max(0, Math.round((Date.now() - issued.getTime()) / 86_400_000))
}

function ageTone(r: OutstandingChequeRow): Tone {
  if (r.status === 'STALE' || r.isStale) return 'danger'
  if (ageDays(r) >= 150) return 'warning'
  return 'neutral'
}

const cancelChequeForm: FormSpec<{ reason: string }> = {
  title: 'Cancel Cheque',
  submitLabel: 'Cancel Cheque',
  initial: () => ({ reason: '' }),
  fields: [
    {
      key: 'reason',
      label: 'Reason',
      kind: 'textarea',
      required: true,
      placeholder: 'Why is this cheque being cancelled?',
    },
  ],
  // Row-aware submit (ROW-AWARE FORMS): the clicked cheque flows through as
  // `row`, so this reason-capture form knows exactly which cheque to cancel.
  submit: async (draft, row) => {
    const r = row as OutstandingChequeRow
    await cancelCheque(r.chequeNumber, draft.reason)
  },
}

const cancelAction: ActionSpec<OutstandingChequeRow> = {
  key: 'cancel',
  label: 'Cancel Cheque',
  kind: 'row',
  visible: (r) => r != null && r.status === 'ISSUED',
  form: cancelChequeForm,
  run: () => {
    /* form action submits via cancelChequeForm; run is unused */
  },
}

const markStaleAction: ActionSpec<OutstandingChequeRow> = {
  key: 'markStale',
  label: 'Mark Stale',
  kind: 'row',
  visible: (r) => r != null && nextStates(r.status, CHEQUE_STATUS_TRANSITIONS).includes('STALE'),
  confirm: (r) => `Mark cheque ${r ? r.chequeNumber : 'this cheque'} as stale?`,
  run: async ({ row, reload }) => {
    if (!row) return
    await markChequeStale(row.chequeNumber)
    await reload()
  },
}

export const chequeRegisterDescriptor: LedgerDescriptor<OutstandingChequeRow> = {
  entity: 'cheque-register',
  title: 'Cheque Register',
  fetch: fetchChequeRegister,
  id: (r) => r.id,
  searchText: (r) => `${r.chequeNumber} ${r.payeeName} ${r.purpose}`,

  columns: [
    { key: 'chequeNumber', label: 'Cheque #', content: 'code', value: (r) => r.chequeNumber, minWidth: 110 },
    { key: 'issuedDate', label: 'Issued', content: 'date', value: (r) => r.issuedDate, minWidth: 100 },
    { key: 'payeeName', label: 'Payee', content: 'name', value: (r) => r.payeeName, grow: true, minWidth: 220 },
    { key: 'amount', label: 'Amount', content: 'money', value: (r) => r.amount, currency: (r) => r.currency, minWidth: 130 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 110 },
    {
      key: 'age',
      label: 'Age (days)',
      content: 'quantity',
      value: (r) => ageDays(r),
      tone: ageTone,
      minWidth: 100,
    },
  ],

  status: {
    value: (r) => r.status,
    tones: CHEQUE_STATUS_TONES,
    // Declared legal-transition graph, consumed via the shared nextStates()
    // helper below (same pattern as purchase-orders.descriptor.ts).
    transitions: CHEQUE_STATUS_TRANSITIONS,
  },

  summary: {
    metrics: [
      { label: 'Outstanding Cheques', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Outstanding Total (BHD)',
        content: 'money',
        value: (rows) => rows.filter((r) => r.currency === 'BHD').reduce((s, r) => s + r.amount, 0),
      },
      {
        label: 'Stale',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.status === 'STALE' || r.isStale).length,
        tone: (rows) => (rows.some((r) => r.status === 'STALE' || r.isStale) ? 'danger' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By status',
      value: (r) => r.status,
      tones: CHEQUE_STATUS_TONES,
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
      key: 'payeeType',
      label: 'Payee Type',
      options: 'derive',
      deriveValue: (r) => r.payeeType,
      predicate: (r, v) => r.payeeType === v,
    },
  ],

  actions: [markStaleAction, cancelAction],

  emptyMessage: 'No outstanding cheques. Issue the first one from a bank account.',
}
