/* DataQualityScreen as a descriptor — an admin cleanup queue, structurally a
 * DocumentLedger (recon K4: "fits-an-archetype-after-all... StatTileGrid +
 * Toolbar + DataTable/ListWidget + small history table"). The review-history
 * table (`GetDataQualityReviewHistory`) is a genuine second panel the ledger
 * archetype has no slot for — ledgered, not built; see DataQuality.parity.md.
 *
 * Two independent status dimensions exist on each issue (severity + review
 * status), same shape as Expenses' status/payment_status split: `status`
 * drives the real badge column (review status — the workflow lifecycle),
 * severity renders as a toned text column instead of a second badge. */

import type { ActionSpec, LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import type { Tone } from '$kernel/tones'
import { fetchDataQualityIssues, reviewDataQualityIssue, type DataQualityIssueRow } from '../bridge/data-quality'

// Severities missing on a record read as "review" (old screen: `issue.severity
// || "review"`) — the same fallback value is used for the column, the
// summary distribution, and the filter chip, so tone lookups stay consistent.
const SEVERITY_TONES: Record<string, Tone> = {
  critical: 'danger',
  high: 'danger',
  medium: 'warning',
  low: 'neutral',
  review: 'info',
}
const severityOf = (r: DataQualityIssueRow): string => r.severity || 'review'

const REVIEW_STATUS_TONES: Record<string, Tone> = {
  unreviewed: 'warning',
  reviewed: 'info',
  resolved: 'success',
  dismissed: 'neutral',
}
const reviewStatusOf = (r: DataQualityIssueRow): string => r.reviewStatus || 'unreviewed'

/** Row-aware reason form, one shape shared by all three review actions — the
 * old screen's single `reviewNotes[issue.id]` textarea feeding whichever
 * button was clicked (Mark reviewed / Resolve / Dismiss), split into three
 * ROW-AWARE FORMS actions here (same mechanism as Expenses' Reject). Note is
 * optional, matching the old screen's `reviewNotes[issue.id] || ""`. */
function reviewForm(action: string, title: string): FormSpec<{ note: string }> {
  return {
    title,
    submitLabel: title,
    initial: () => ({ note: '' }),
    fields: [{ key: 'note', label: 'Review Note', kind: 'textarea', placeholder: 'Optional note for the audit trail' }],
    submit: async (draft, row) => {
      await reviewDataQualityIssue(row as DataQualityIssueRow, action, draft.note)
    },
  }
}

function reviewAction(key: string, label: string, action: string): ActionSpec<DataQualityIssueRow> {
  return {
    key,
    label,
    kind: 'row',
    // The old screen shows all three buttons unconditionally for every
    // issue; this hides only the action matching the row's CURRENT status
    // (no point re-marking an already-resolved issue resolved) — a mild
    // fix, not an invented state machine (all three remain reachable from
    // any other status, e.g. re-triaging a dismissed issue as resolved).
    visible: (r) => r != null && reviewStatusOf(r) !== action,
    form: reviewForm(action, label),
    run: () => {
      /* form action submits via reviewForm; run is unused */
    },
  }
}

export const dataQualityDescriptor: LedgerDescriptor<DataQualityIssueRow> = {
  entity: 'data-quality',
  title: 'Data Quality',
  fetch: fetchDataQualityIssues,
  id: (r) => r.id,
  searchText: (r) => `${r.summary} ${r.detail} ${r.primaryAction} ${r.entityType} ${r.entityId}`,

  columns: [
    { key: 'entityType', label: 'Entity Type', content: 'text', value: (r) => r.entityType, minWidth: 110 },
    { key: 'entityId', label: 'Entity', content: 'code', value: (r) => r.entityId, minWidth: 100 },
    { key: 'summary', label: 'Summary', content: 'name', value: (r) => r.summary, grow: true, minWidth: 260 },
    { key: 'issueType', label: 'Issue Kind', content: 'text', value: (r) => r.issueType, minWidth: 190 },
    {
      key: 'severity',
      label: 'Severity',
      content: 'text',
      value: (r) => severityOf(r),
      tone: (r) => SEVERITY_TONES[severityOf(r)] ?? 'info',
      minWidth: 90,
    },
    { key: 'reviewStatus', label: 'Status', content: 'status', value: (r) => reviewStatusOf(r), minWidth: 110 },
    { key: 'detail', label: 'Detail', content: 'text', value: (r) => r.detail, grow: true, minWidth: 260 },
  ],

  status: {
    value: (r) => reviewStatusOf(r),
    tones: REVIEW_STATUS_TONES,
    // unreviewed -> any of the three outcomes; a reviewed/resolved/dismissed
    // issue can still be re-triaged into either other outcome (matches the
    // old screen's unconditional button availability — see reviewAction).
    transitions: {
      unreviewed: ['reviewed', 'resolved', 'dismissed'],
      reviewed: ['resolved', 'dismissed'],
      resolved: ['reviewed', 'dismissed'],
      dismissed: ['reviewed', 'resolved'],
    },
  },

  summary: {
    metrics: [
      { label: 'Total Issues', content: 'quantity', value: (rows) => rows.length },
      { label: 'Duplicates', content: 'quantity', value: (rows) => rows.filter((r) => r.issueType === 'duplicate_customer').length },
      {
        label: 'Missing Links',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.issueType.includes('missing') || r.issueType.includes('orphan')).length,
      },
      { label: 'Blank Records', content: 'quantity', value: (rows) => rows.filter((r) => r.issueType.includes('blank')).length },
    ],
    distribution: {
      label: 'By severity',
      value: (r) => severityOf(r),
      tones: SEVERITY_TONES,
    },
  },

  filters: [
    { key: 'issueType', label: 'Issue Kind', options: 'derive', deriveValue: (r) => r.issueType, predicate: (r, v) => r.issueType === v },
    { key: 'severity', label: 'Severity', options: 'derive', deriveValue: (r) => severityOf(r), predicate: (r, v) => severityOf(r) === v },
  ],

  actions: [
    reviewAction('mark-reviewed', 'Mark Reviewed', 'reviewed'),
    reviewAction('resolve', 'Resolve', 'resolved'),
    reviewAction('dismiss', 'Dismiss', 'dismissed'),
  ],

  emptyMessage: 'No data quality issues found. The cleanup queue is clear.',
}
