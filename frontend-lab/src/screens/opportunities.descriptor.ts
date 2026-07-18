/* OpportunitiesScreen as a descriptor. Old frontend: 1303 lines merging RFQs +
 * pipeline Opportunities into one sales-pipeline ledger, with a detail drawer,
 * create form, two-tier delete (plain vs. typed-reason cascade), and a "Start
 * Project" handoff into WorkHub. K4 scope: the ledger spine (list, summary,
 * create, two-tier delete). Detail drawer and the WorkHub handoff are
 * ledgered — see screens/parity/Opportunities.parity.md. */

import type { ActionSpec, LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import type { Tone } from '$kernel/tones'
import {
  cascadeDeleteOpportunity,
  createOpportunity,
  deleteOpportunity,
  fetchOpportunities,
  opportunityCustomerOptions,
  type NewOpportunityDraft,
  type OpportunityRow,
} from '../bridge/opportunities'

const STAGE_TONES: Record<string, Tone> = {
  Pending: 'neutral',
  Qualified: 'info',
  Proposal: 'info',
  Negotiation: 'warning',
  Won: 'success',
  Lost: 'danger',
  // Unknown stages render neutral by engine contract — never crash.
}

const newOpportunityForm: FormSpec<NewOpportunityDraft> = {
  title: 'New Opportunity',
  submitLabel: 'Create',
  initial: () => ({ customer: '', project: '', value: null, notes: '' }),
  fields: [
    { key: 'customer', label: 'Customer', kind: 'select', required: true, options: opportunityCustomerOptions },
    { key: 'project', label: 'Project', kind: 'text', required: true },
    {
      key: 'value',
      label: 'Value (BHD)',
      kind: 'number',
      required: true,
      step: '0.001',
      validate: (v) => (typeof v === 'number' && v <= 0 ? 'Value must be positive' : null),
    },
    { key: 'notes', label: 'Notes', kind: 'textarea', placeholder: 'Optional' },
  ],
  submit: (draft) => createOpportunity(draft),
}

// Row-aware reason form (ROW-AWARE FORMS pattern, see cheque-register's
// cancelChequeForm): cascade delete destroys linked costing sheets/offers, so
// it's escalated behind a mandatory typed reason, not a plain confirm — same
// two-tier shape the old screen's cascade-delete dialog enforced.
const cascadeDeleteForm: FormSpec<{ reason: string }> = {
  title: 'Delete with Cascade',
  submitLabel: 'Delete & Cascade',
  initial: () => ({ reason: '' }),
  fields: [
    {
      key: 'reason',
      label: 'Reason',
      kind: 'textarea',
      required: true,
      placeholder: 'Why is this opportunity — and its linked costing sheets/offers — being destroyed?',
    },
  ],
  submit: async (draft, row) => {
    const r = row as OpportunityRow
    await cascadeDeleteOpportunity(r, draft.reason)
  },
}

const deleteAction: ActionSpec<OpportunityRow> = {
  key: 'delete',
  label: 'Delete',
  kind: 'row',
  confirm: (r) => `Delete ${r ? (r as OpportunityRow).ref : 'this opportunity'}? This cannot be undone.`,
  run: async ({ row, reload }) => {
    if (!row) return
    await deleteOpportunity(row)
    await reload()
  },
}

// Cascade delete is only offered for RFQ-sourced rows — the real
// DeleteRFQWithCascade binding has no pipeline-Opportunity equivalent (see
// bridge/opportunities.ts realCascadeDelete). Ledgering the gap honestly
// rather than offering a button the real side can't back.
const cascadeDeleteAction: ActionSpec<OpportunityRow> = {
  key: 'cascadeDelete',
  label: 'Delete with Cascade',
  kind: 'row',
  visible: (r) => r != null && r.source === 'rfq',
  form: cascadeDeleteForm,
  run: () => {
    /* form action submits via cascadeDeleteForm; run is unused */
  },
}

export const opportunitiesDescriptor: LedgerDescriptor<OpportunityRow> = {
  entity: 'opportunities',
  title: 'Opportunities',
  fetch: fetchOpportunities,
  id: (r) => r.id,
  searchText: (r) => `${r.ref} ${r.customer} ${r.project}`,

  columns: [
    { key: 'ref', label: 'Ref', content: 'code', value: (r) => r.ref, minWidth: 110 },
    {
      key: 'source',
      label: 'Type',
      content: 'text',
      value: (r) => (r.source === 'rfq' ? 'RFQ' : 'Pipeline'),
      minWidth: 90,
    },
    { key: 'customer', label: 'Customer', content: 'name', value: (r) => r.customer, grow: true, minWidth: 200 },
    { key: 'project', label: 'Project', content: 'text', value: (r) => r.project, minWidth: 200 },
    { key: 'value', label: 'Value', content: 'money', value: (r) => r.value, minWidth: 140 },
    { key: 'stage', label: 'Stage', content: 'status', value: (r) => r.stage, minWidth: 130 },
    { key: 'createdAt', label: 'Created', content: 'date', value: (r) => r.createdAt, minWidth: 120 },
  ],

  status: {
    value: (r) => r.stage,
    tones: STAGE_TONES,
    // Documentation-grade (no row action edits stage on this screen today —
    // stage changes originate from RFQ/Offer workflows, not here).
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
      { label: 'Opportunities', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Pipeline Value (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.value, 0),
      },
      {
        label: 'Win Rate %',
        content: 'quantity',
        value: (rows) => {
          const won = rows.filter((r) => r.stage === 'Won').length
          const decided = won + rows.filter((r) => r.stage === 'Lost').length
          return decided === 0 ? 0 : Math.round((won / decided) * 1000) / 10
        },
        tone: (rows) => (rows.some((r) => r.stage === 'Won') ? 'success' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By stage',
      value: (r) => r.stage,
      tones: STAGE_TONES,
    },
  },

  filters: [
    {
      key: 'stage',
      label: 'Stage',
      options: 'derive',
      deriveValue: (r) => r.stage,
      predicate: (r, v) => r.stage === v,
    },
    {
      key: 'source',
      label: 'Type',
      options: [
        { value: 'rfq', label: 'RFQ' },
        { value: 'pipeline', label: 'Pipeline' },
      ],
      predicate: (r, v) => r.source === v,
    },
  ],

  actions: [
    {
      key: 'new',
      label: '+ New Opportunity',
      kind: 'screen',
      form: newOpportunityForm,
      run: () => {
        /* form actions submit via their FormSpec; run is unused */
      },
    },
    deleteAction,
    cascadeDeleteAction,
  ],

  emptyMessage: 'No opportunities yet. Log the first customer enquiry.',
}
