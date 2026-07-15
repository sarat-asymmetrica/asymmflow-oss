/* GRNScreen as a descriptor. K1 scope: read + summary only — every mutating
 * capability (Receive from PO, QC Review, Complete) is a financial hot-zone
 * or auth-gated slot, ledgered rather than rebuilt loosely (see
 * screens/parity/GRNs.parity.md). */

import type { LedgerDescriptor } from '$kernel/descriptor'
import type { Tone } from '$kernel/tones'
import { fetchGRNs, type GRNRow } from '../bridge/grns'

/** Acceptance-rate threshold colouring (census: ≥95 green / ≥80 amber / <80 red). */
function acceptanceTone(pct: number): Tone {
  if (pct >= 95) return 'success'
  if (pct >= 80) return 'warning'
  return 'danger'
}

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

  // No actions: Receive-from-PO (creates the GRN), QC Review (auth-gated),
  // and Complete (posts inventory, idempotent) are all ledgered — see the
  // parity doc. Row click still opens the default column-list detail panel.
  emptyMessage: 'No GRNs yet. Receive against a purchase order to create the first one.',
}
