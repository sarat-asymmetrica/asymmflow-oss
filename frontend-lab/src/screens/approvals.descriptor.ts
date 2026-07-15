/* ApprovalsQueueScreen as a descriptor. Old frontend: 359 lines, a durable
 * admin queue for pending delete/employee-archive requests, mounted standalone
 * or embedded (WorkHub hosts it via an `embedded` prop — out of scope here,
 * the registry entry is the standalone mount). Admin-privileged: real fetch
 * server-side already returns an empty list to non-admin sessions, and the
 * whole real side is INTEG-gapped for K4 regardless — see
 * screens/parity/Approvals.parity.md. */

import type { ActionSpec, LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import type { Tone } from '$kernel/tones'
import { approveApproval, fetchApprovals, rejectApproval, type ApprovalRow } from '../bridge/approvals'

const STATUS_TONES: Record<string, Tone> = {
  pending: 'warning',
  approved: 'success',
  rejected: 'danger',
  // Unknown statuses render neutral by engine contract — never crash.
}

const KIND_TONES: Record<string, Tone> = {
  delete: 'info',
  archive: 'warning',
}

// Row-aware reason form (ROW-AWARE FORMS pattern, see cheque-register's
// cancelChequeForm) — a rejection needs a reviewer note, an approval doesn't.
const rejectForm: FormSpec<{ reason: string }> = {
  title: 'Reject Request',
  submitLabel: 'Reject',
  initial: () => ({ reason: '' }),
  fields: [
    {
      key: 'reason',
      label: 'Reason',
      kind: 'textarea',
      required: true,
      placeholder: 'Why is this request being rejected?',
    },
  ],
  submit: async (draft, row) => {
    const r = row as ApprovalRow
    await rejectApproval(r, draft.reason)
  },
}

const approveAction: ActionSpec<ApprovalRow> = {
  key: 'approve',
  label: 'Approve',
  kind: 'row',
  visible: (r) => r != null && r.status === 'pending',
  confirm: (r) => `Approve this ${r ? (r as ApprovalRow).kind : ''} request for ${r ? (r as ApprovalRow).target : 'this record'}?`,
  run: async ({ row, reload }) => {
    if (!row) return
    await approveApproval(row)
    await reload()
  },
}

const rejectAction: ActionSpec<ApprovalRow> = {
  key: 'reject',
  label: 'Reject',
  kind: 'row',
  visible: (r) => r != null && r.status === 'pending',
  form: rejectForm,
  run: () => {
    /* form action submits via rejectForm; run is unused */
  },
}

export const approvalsDescriptor: LedgerDescriptor<ApprovalRow> = {
  entity: 'approvals',
  title: 'Approvals Queue',
  fetch: fetchApprovals,
  id: (r) => r.id,
  searchText: (r) => `${r.target} ${r.requestedBy} ${r.reason}`,

  columns: [
    { key: 'kind', label: 'Kind', content: 'status', value: (r) => r.kind, tone: (r) => KIND_TONES[r.kind] ?? 'neutral', minWidth: 90 },
    { key: 'target', label: 'Target', content: 'name', value: (r) => r.target, grow: true, minWidth: 220 },
    { key: 'requestedBy', label: 'Requested By', content: 'name', value: (r) => r.requestedBy, minWidth: 170 },
    { key: 'requestedAt', label: 'Requested', content: 'date', value: (r) => r.requestedAt, minWidth: 110 },
    { key: 'reason', label: 'Reason', content: 'text', value: (r) => r.reason, minWidth: 240 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 110 },
  ],

  status: {
    value: (r) => r.status,
    tones: STATUS_TONES,
    transitions: {
      pending: ['approved', 'rejected'],
      approved: [],
      rejected: [],
    },
  },

  summary: {
    metrics: [
      { label: 'Requests', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Pending',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.status === 'pending').length,
        tone: (rows) => (rows.some((r) => r.status === 'pending') ? 'warning' : 'neutral'),
      },
      { label: 'Approved', content: 'quantity', value: (rows) => rows.filter((r) => r.status === 'approved').length },
      { label: 'Rejected', content: 'quantity', value: (rows) => rows.filter((r) => r.status === 'rejected').length },
    ],
    distribution: {
      label: 'By kind',
      value: (r) => r.kind,
      tones: KIND_TONES,
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
      key: 'kind',
      label: 'Kind',
      options: [
        { value: 'delete', label: 'Delete' },
        { value: 'archive', label: 'Archive' },
      ],
      predicate: (r, v) => r.kind === v,
    },
  ],

  actions: [approveAction, rejectAction],

  emptyMessage: 'No pending approvals. The queue is clear.',
}
