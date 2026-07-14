/* FXRevaluationScreen as a descriptor. K4 scope: the PRIMARY ledger = FX
 * Revaluations (one row per revaluation run, unrealized gain/loss, Post,
 * Reverse). The old screen's other two tabs (Exposure, Rates) are separate
 * fetches against different shapes (GetFXExposureReport;
 * GetLatestFXRate/CreateFXRate) — ledgered as a multi-panel ENGINE gap, see
 * screens/parity/FxRevaluation.parity.md. Financial hot-zone: Post/Reverse
 * are honestly mock-only here (real = INTEG-gap), same trade-off
 * cheque-register.descriptor.ts makes for its status-changing actions. */

import type { ActionSpec, LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import type { Tone } from '$kernel/tones'
import {
  fetchFxRevaluations,
  postFxRevaluation,
  reverseFxRevaluation,
  type FxRevaluationRow,
} from '../bridge/fx-revaluation'

// Two-state, not three (see bridge/fx-revaluation.ts's header note): reading
// pkg/finance/fx/fx.go directly, Reverse() never flips a row's own status to
// "Reversed" — it inserts a new Posted reversing row, or deletes an unposted
// one. Draft -> Posted is the only real in-place transition.
const FX_STATUS_TRANSITIONS: Record<string, string[]> = {
  Draft: ['Posted'],
  Posted: [],
}

const FX_STATUS_TONES: Record<string, Tone> = {
  Draft: 'neutral',
  Posted: 'success',
}

const reverseForm: FormSpec<{ reason: string }> = {
  title: 'Reverse FX Revaluation',
  submitLabel: 'Reverse',
  initial: () => ({ reason: '' }),
  fields: [
    {
      key: 'reason',
      label: 'Reason',
      kind: 'textarea',
      required: true,
      placeholder: 'Why is this posted revaluation being reversed?',
    },
  ],
  // Row-aware submit: the clicked revaluation flows through as `row`.
  submit: async (draft, row) => {
    const r = row as FxRevaluationRow
    await reverseFxRevaluation(r.id, draft.reason)
  },
}

const postAction: ActionSpec<FxRevaluationRow> = {
  key: 'post',
  label: 'Post',
  kind: 'row',
  visible: (r) => r != null && r.status === 'Draft',
  confirm: (r) => {
    if (!r) return 'Post this FX revaluation?'
    const sign = r.gainLossBhd >= 0 ? '+' : ''
    return `Post this FX revaluation of ${sign}${r.gainLossBhd.toFixed(3)} BHD for ${r.accountLabel} as of ${r.revaluationDate}? This marks it posted.`
  },
  run: async ({ row, reload }) => {
    if (!row) return
    await postFxRevaluation(row.id)
    await reload()
  },
}

const reverseAction: ActionSpec<FxRevaluationRow> = {
  key: 'reverse',
  label: 'Reverse',
  kind: 'row',
  visible: (r) => r != null && r.status === 'Posted',
  form: reverseForm,
  run: () => {
    /* form action submits via reverseForm; run is unused */
  },
}

export const fxRevaluationDescriptor: LedgerDescriptor<FxRevaluationRow> = {
  entity: 'fx-revaluation',
  title: 'FX Revaluation',
  fetch: fetchFxRevaluations,
  id: (r) => r.id,
  searchText: (r) => `${r.accountLabel} ${r.currency} ${r.status}`,

  columns: [
    { key: 'accountLabel', label: 'Account', content: 'name', value: (r) => r.accountLabel, grow: true, minWidth: 220 },
    { key: 'currency', label: 'Currency', content: 'code', value: (r) => r.currency, minWidth: 90 },
    { key: 'revaluationDate', label: 'As Of', content: 'date', value: (r) => r.revaluationDate, minWidth: 100 },
    {
      key: 'foreignBalance',
      label: 'Foreign Amount',
      content: 'money',
      value: (r) => r.foreignBalance,
      currency: (r) => r.currency,
      minWidth: 150,
    },
    { key: 'currentRate', label: 'Rate', content: 'quantity', value: (r) => r.currentRate, minWidth: 90 },
    { key: 'currentBhd', label: 'BHD Value', content: 'money', value: (r) => r.currentBhd, minWidth: 130 },
    {
      key: 'gainLossBhd',
      label: 'Unrealized G/L',
      content: 'money',
      value: (r) => r.gainLossBhd,
      tone: (r) => (r.gainLossBhd >= 0 ? 'success' : 'danger'),
      minWidth: 140,
    },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 100 },
  ],

  status: {
    value: (r) => r.status,
    tones: FX_STATUS_TONES,
    transitions: FX_STATUS_TRANSITIONS,
  },

  summary: {
    metrics: [
      {
        // "Current exposure" reduces to the LATEST revaluation per account
        // among the visible rows, not a sum across the whole run history —
        // summing every historical run would double-count. This is a
        // synthesized figure; the real one is GetFXExposureReport (ledgered,
        // see the parity doc), which this approximates honestly.
        label: 'Total Exposure (BHD)',
        content: 'money',
        value: (rows) => latestPerAccount(rows).reduce((s, r) => s + r.currentBhd, 0),
      },
      {
        label: 'Total Unrealized G/L (BHD)',
        content: 'money',
        value: (rows) => latestPerAccount(rows).reduce((s, r) => s + r.gainLossBhd, 0),
        tone: (rows) => (latestPerAccount(rows).reduce((s, r) => s + r.gainLossBhd, 0) >= 0 ? 'success' : 'danger'),
      },
      { label: 'Revaluation Runs', content: 'quantity', value: (rows) => rows.length },
    ],
    distribution: {
      label: 'By status',
      value: (r) => r.status,
      tones: FX_STATUS_TONES,
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
      key: 'currency',
      label: 'Currency',
      options: 'derive',
      deriveValue: (r) => r.currency,
      predicate: (r, v) => r.currency === v,
    },
  ],

  actions: [postAction, reverseAction],

  emptyMessage: 'No FX revaluations yet. Revalue All runs against every active foreign-currency account.',
}

function latestPerAccount(rows: FxRevaluationRow[]): FxRevaluationRow[] {
  const latest = new Map<string, FxRevaluationRow>()
  for (const r of rows) {
    const prev = latest.get(r.bankAccountId)
    if (!prev || r.revaluationDate > prev.revaluationDate) latest.set(r.bankAccountId, r)
  }
  return [...latest.values()]
}
