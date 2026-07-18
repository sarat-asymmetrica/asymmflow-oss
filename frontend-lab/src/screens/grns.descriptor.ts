/* GRNScreen as a descriptor. K1 scope: read + summary only — every mutating
 * capability (Receive from PO, QC Review, Complete) is a financial hot-zone
 * or auth-gated slot, ledgered rather than rebuilt loosely (see
 * screens/parity/GRNs.parity.md). */

import type { ActionSpec, LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import type { Tone } from '$kernel/tones'
import { completeGRN, fetchGRNs, updateGRNQCReview, type GRNRow } from '../bridge/grns'

/** Acceptance-rate threshold colouring (census: ≥95 green / ≥80 amber / <80 red). */
function acceptanceTone(pct: number): Tone {
  if (pct >= 95) return 'success'
  if (pct >= 80) return 'warning'
  return 'danger'
}

// R5 capture forms. Receive-from-PO (creates a GRN from PO line items) stays a
// ledgered SLOT — it needs the LineItemsEditor engine (per-line receive qty),
// not a flat FormModal. QC Review + Complete are flat and built here.
const qcReviewForm: FormSpec<{ status: string; notes: string }> = {
  title: 'QC Review',
  submitLabel: 'Record QC',
  initial: (row) => {
    const r = row as GRNRow | undefined
    return { status: r && r.qcStatus !== 'Pending' ? r.qcStatus : 'Passed', notes: r?.qcNotes ?? '' }
  },
  fields: [
    {
      key: 'status',
      label: 'QC Verdict',
      kind: 'select',
      required: true,
      options: [
        { value: 'Passed', label: 'Passed' },
        { value: 'Failed', label: 'Failed' },
        { value: 'Partial', label: 'Partial' },
      ],
    },
    { key: 'notes', label: 'QC Notes', kind: 'textarea', placeholder: 'Inspection findings (optional)' },
  ],
  submit: async (draft, row) => {
    const r = row as GRNRow
    await updateGRNQCReview(r.id, draft.status, draft.notes)
  },
}

const grnActions: ActionSpec<GRNRow>[] = [
  {
    key: 'qc-review',
    label: 'QC Review',
    kind: 'row',
    // Reviewable until the GRN is completed.
    visible: (r) => r != null && !r.isCompleted,
    form: qcReviewForm,
    run: () => {
      /* submitted via qcReviewForm */
    },
  },
  {
    key: 'complete',
    label: 'Complete',
    kind: 'row',
    // Server refuses a Failed-QC GRN and is idempotent; hide once completed.
    visible: (r) => r != null && !r.isCompleted && r.qcStatus !== 'Failed',
    confirm: (r) =>
      `Complete ${r ? r.grnNumber : 'this GRN'}? This closes the GRN and updates the linked purchase order.`,
    run: async ({ row, reload }) => {
      if (!row) return
      await completeGRN(row.id)
      await reload()
    },
  },
]

export const grnsDescriptor: LedgerDescriptor<GRNRow> = {
  entity: 'grns',
  title: 'Goods Received Notes',
  fetch: fetchGRNs,
  id: (r) => r.id,
  searchText: (r) => `${r.grnNumber} ${r.poNumber} ${r.supplierName}`,

  columns: [
    { key: 'grnNumber', label: 'GRN #', content: 'code', value: (r) => r.grnNumber, minWidth: 140 },
    { key: 'poNumber', label: 'PO Reference', content: 'code', value: (r) => r.poNumber, minWidth: 140 },
    { key: 'supplierName', label: 'Supplier', content: 'name', value: (r) => r.supplierName, grow: true, minWidth: 220 },
    { key: 'receivedDate', label: 'Received Date', content: 'date', value: (r) => r.receivedDate, minWidth: 130 },
    { key: 'qcStatus', label: 'QC Status', content: 'status', value: (r) => r.qcStatus, minWidth: 120 },
    { key: 'itemsCount', label: 'Items', content: 'quantity', value: (r) => r.itemsCount, minWidth: 80 },
    {
      key: 'acceptanceRate',
      label: 'Acceptance %',
      content: 'quantity',
      // acceptanceRate is a 0–1 fraction on the row — ×100 for display, same
      // convention the old screen used (row.acceptance_rate * 100).
      value: (r) => Math.round(r.acceptanceRate * 1000) / 10,
      tone: (r) => acceptanceTone(r.acceptanceRate * 100),
      minWidth: 110,
    },
  ],

  status: {
    value: (r) => r.qcStatus,
    tones: {
      Pending: 'warning',
      Passed: 'success',
      Failed: 'danger',
      Partial: 'warning',
      // Unknown statuses render neutral by engine contract — never crash.
    },
  },

  // Visual-diversity strip: counts + rate + a rejection callout in one row,
  // per the census's own note that GRN's stats are the richest of the batch.
  summary: {
    metrics: [
      { label: 'GRNs', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Items Accepted',
        content: 'quantity',
        value: (rows) => rows.reduce((s, r) => s + r.totalAccepted, 0),
      },
      {
        label: 'Items Rejected',
        content: 'quantity',
        value: (rows) => rows.reduce((s, r) => s + r.totalRejected, 0),
        tone: (rows) => (rows.some((r) => r.totalRejected > 0) ? 'warning' : 'neutral'),
      },
      {
        label: 'Acceptance Rate',
        content: 'quantity',
        // Weighted (sum accepted / sum received), not an average of averages.
        value: (rows) => {
          const received = rows.reduce((s, r) => s + r.totalReceived, 0)
          const accepted = rows.reduce((s, r) => s + r.totalAccepted, 0)
          return received > 0 ? Math.round((accepted / received) * 1000) / 10 : 0
        },
        tone: (rows) => {
          const received = rows.reduce((s, r) => s + r.totalReceived, 0)
          const accepted = rows.reduce((s, r) => s + r.totalAccepted, 0)
          return acceptanceTone(received > 0 ? (accepted / received) * 100 : 100)
        },
      },
    ],
    distribution: {
      label: 'By QC status',
      value: (r) => r.qcStatus,
      tones: {
        Pending: 'warning',
        Passed: 'success',
        Failed: 'danger',
        Partial: 'warning',
      },
    },
  },

  filters: [
    {
      key: 'qcStatus',
      label: 'QC Status',
      options: 'derive',
      deriveValue: (r) => r.qcStatus,
      predicate: (r, v) => r.qcStatus === v,
    },
  ],

  // R5: QC Review + Complete built as capture/confirm actions. Receive-from-PO
  // (creates the GRN from PO line items) stays a ledgered SLOT pending the
  // LineItemsEditor engine — see the parity doc. Row click still opens the
  // default column-list detail panel.
  actions: grnActions,

  emptyMessage: 'No GRNs yet. Receive against a purchase order to create the first one.',
}
