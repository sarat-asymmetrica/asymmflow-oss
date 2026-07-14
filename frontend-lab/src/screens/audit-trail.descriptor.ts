/* AuditTrailViewer as a descriptor. Old frontend: 570 lines — bank-
 * reconciliation audit trail (IMPORT/MATCH/UNMATCH/SPLIT/CATEGORIZE/
 * RECONCILE/VERIFY), read-only detail, reverse a non-reversed action with a
 * mandatory reason. K4 scope: the ledger spine + Reverse.
 *
 * CRITICAL (preserved, not just ported): Article V.4/B3(c) — row-click is a
 * READ-ONLY detail view; Reverse is a SEPARATE explicit row action, never
 * click-to-reverse. The DocumentLedger archetype already keeps these paths
 * distinct (row selection only ever opens the default/slotted detail panel;
 * mutating actions live in the actions column), so this descriptor doesn't
 * need to do anything special to hold the line — it just must never wire
 * Reverse to row selection. See screens/parity/AuditTrail.parity.md. */

import type { ActionSpec, LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import type { Tone } from '$kernel/tones'
import { fetchAuditTrail, reverseAuditAction, type AuditTrailRow } from '../bridge/audit-trail'

const ACTION_TONES: Record<string, Tone> = {
  IMPORT: 'info',
  MATCH: 'success',
  UNMATCH: 'warning',
  SPLIT: 'info',
  CATEGORIZE: 'neutral',
  RECONCILE: 'success',
  VERIFY: 'info',
  // Unknown actions render neutral by engine contract — never crash.
}

const STATE_TONES: Record<string, Tone> = {
  Active: 'neutral',
  Reversed: 'danger',
}

function stateOf(r: AuditTrailRow): string {
  return r.reversed ? 'Reversed' : 'Active'
}

// Row-aware reason form (ROW-AWARE FORMS pattern) — the mandatory reason IS
// the reversal_reason on the real ReverseAction(logId, user, reason) binding.
const reverseForm: FormSpec<{ reason: string }> = {
  title: 'Reverse Action',
  submitLabel: 'Reverse',
  initial: () => ({ reason: '' }),
  fields: [
    {
      key: 'reason',
      label: 'Reason',
      kind: 'textarea',
      required: true,
      placeholder: 'Why is this action being reversed?',
    },
  ],
  submit: async (draft, row) => {
    const r = row as AuditTrailRow
    await reverseAuditAction(r, draft.reason)
  },
}

const reverseAction: ActionSpec<AuditTrailRow> = {
  key: 'reverse',
  label: 'Reverse',
  kind: 'row',
  visible: (r) => r != null && !r.reversed,
  form: reverseForm,
  run: () => {
    /* form action submits via reverseForm; run is unused */
  },
}

export const auditTrailDescriptor: LedgerDescriptor<AuditTrailRow> = {
  entity: 'audit-trail',
  title: 'Audit Trail',
  fetch: fetchAuditTrail,
  id: (r) => r.id,
  searchText: (r) => `${r.action} ${r.statementRef} ${r.actor}`,

  columns: [
    { key: 'timestamp', label: 'Timestamp', content: 'date', value: (r) => r.timestamp, minWidth: 110 },
    {
      key: 'action',
      label: 'Action',
      content: 'status',
      value: (r) => r.action,
      tone: (r) => ACTION_TONES[r.action] ?? 'neutral',
      minWidth: 120,
    },
    { key: 'statementRef', label: 'Statement', content: 'code', value: (r) => r.statementRef, minWidth: 130 },
    { key: 'actor', label: 'Actor', content: 'name', value: (r) => r.actor, grow: true, minWidth: 180 },
    { key: 'amount', label: 'Amount', content: 'money', value: (r) => r.amount, minWidth: 140 },
    {
      key: 'reversed',
      label: 'State',
      content: 'status',
      value: (r) => stateOf(r),
      tone: (r) => STATE_TONES[stateOf(r)] ?? 'neutral',
      minWidth: 100,
    },
  ],

  // The reversed-state, not the action type, is the ledger's canonical
  // status — it's what gates the Reverse row action and is the field with a
  // real legal transition (Active → Reversed, one-way, never back).
  status: {
    value: stateOf,
    tones: STATE_TONES,
    transitions: {
      Active: ['Reversed'],
      Reversed: [],
    },
  },

  summary: {
    metrics: [
      { label: 'Actions', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Reversed',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.reversed).length,
        tone: (rows) => (rows.some((r) => r.reversed) ? 'danger' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By action',
      value: (r) => r.action,
      tones: ACTION_TONES,
    },
  },

  filters: [
    {
      key: 'action',
      label: 'Action',
      options: 'derive',
      deriveValue: (r) => r.action,
      predicate: (r, v) => r.action === v,
    },
    {
      key: 'state',
      label: 'State',
      options: [
        { value: 'Active', label: 'Active' },
        { value: 'Reversed', label: 'Reversed' },
      ],
      predicate: (r, v) => stateOf(r) === v,
    },
  ],

  actions: [reverseAction],

  emptyMessage: 'No audit trail entries yet. Import a bank statement to begin.',
}
