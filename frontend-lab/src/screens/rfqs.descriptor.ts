/* RFQScreen as a descriptor. Old frontend: RFQScreen.svelte — flat 100-row
 * load, no Load-More, client-side stage tabs + search. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import { deleteRFQ, fetchRFQs, updateRFQStage, type RFQRow } from '../bridge/rfqs'

/* The old screen's edit-stage control is a single dropdown restricted to
 * these four (Won/Lost are read-only, driven by the linked Offer). The
 * kernel's FormModal has no row-context binding today (ActionHost keeps the
 * clicked row in its own state but never passes it into the form spec) — so
 * a single "Edit stage" form action can't know which RFQ it's editing. That's
 * a genuine engine gap (see RFQs.parity.md #3), not something to fake here.
 * Building four gated, confirm-only "Set <stage>" row actions gets the same
 * job done using only what the engine has today (ActionSpec.confirm + run). */
const EDITABLE_STAGES = ['Pending', 'Qualified', 'Proposal', 'Negotiation']

function setStageAction(target: string) {
  return {
    key: `stage-${target.toLowerCase()}`,
    label: `Set ${target}`,
    kind: 'row' as const,
    visible: (r: RFQRow | null) => r != null && r.status !== target && EDITABLE_STAGES.includes(r.status),
    confirm: (r: RFQRow | null) => `Move ${r ? r.number : 'this RFQ'} to ${target}?`,
    run: async ({ row, reload }: { row: RFQRow | null; reload: () => Promise<void> }) => {
      if (!row) return
      await updateRFQStage(row.id, target)
      await reload()
    },
  }
}

export const rfqsDescriptor: LedgerDescriptor<RFQRow> = {
  entity: 'rfqs',
  title: 'RFQs',
  fetch: fetchRFQs,
  id: (r) => r.id,
  searchText: (r) => `${r.number} ${r.client} ${r.project} ${r.notes}`,

  columns: [
    { key: 'number', label: 'RFQ #', content: 'code', value: (r) => r.number, minWidth: 100 },
    { key: 'client', label: 'Customer', content: 'name', value: (r) => r.client, grow: true, minWidth: 200 },
    { key: 'productCount', label: 'Products', content: 'quantity', value: (r) => r.productCount, minWidth: 100 },
    { key: 'value', label: 'Total Value', content: 'money', value: (r) => r.value, minWidth: 150 },
    { key: 'createdAt', label: 'Created', content: 'date', value: (r) => r.createdAt, minWidth: 120 },
    // Phantom field (see RFQRow.dueDate) — shown per the old screen, always
    // blank from the real bridge until the backend gains a real column.
    { key: 'dueDate', label: 'Due Date', content: 'date', value: (r) => r.dueDate, minWidth: 120 },
    { key: 'status', label: 'Stage', content: 'status', value: (r) => r.status, minWidth: 140 },
  ],

  status: {
    value: (r) => r.status,
    tones: {
      Pending: 'neutral',
      Qualified: 'info',
      Proposal: 'info',
      Negotiation: 'warning',
      Won: 'success',
      Lost: 'danger',
      // Unknown stages render neutral by engine contract — never crash.
    },
    // Declared pipeline order (documentation-grade — see parity #2 for why
    // the actual edit actions gate on "not terminal", not this graph: the old
    // screen's dropdown lets you jump Pending→Negotiation directly).
    transitions: {
      Pending: ['Qualified'],
      Qualified: ['Proposal'],
      Proposal: ['Negotiation'],
      Negotiation: ['Won', 'Lost'],
      Won: [],
      Lost: [],
    },
  },

  summary: {
    metrics: [
      { label: 'RFQs', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Total Value (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.value, 0),
      },
      {
        label: 'Won',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.status === 'Won').length,
        tone: (rows) => (rows.some((r) => r.status === 'Won') ? 'success' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By stage',
      value: (r) => r.status,
      tones: {
        Pending: 'neutral',
        Qualified: 'info',
        Proposal: 'info',
        Negotiation: 'warning',
        Won: 'success',
        Lost: 'danger',
      },
    },
  },

  filters: [
    {
      key: 'status',
      label: 'Stage',
      options: 'derive',
      deriveValue: (r) => r.status,
      predicate: (r, v) => r.status === v,
    },
  ],

  actions: [
    ...EDITABLE_STAGES.map(setStageAction),
    {
      key: 'delete',
      label: 'Delete',
      kind: 'row',
      confirm: (r) => `Delete ${r ? (r as RFQRow).number : 'this RFQ'}? This cannot be undone.`,
      run: async ({ row, reload }) => {
        if (!row) return
        await deleteRFQ(row.id)
        await reload()
      },
    },
  ],

  emptyMessage: 'No RFQs yet. Log the first customer enquiry.',
}
