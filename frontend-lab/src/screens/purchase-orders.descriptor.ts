/* PurchaseOrdersScreen as a descriptor. K1 scope: the ledger spine (list,
 * paging-free fetch, multi-currency columns, status machine) plus the
 * simple status-flip transitions; the deep features (multi-currency
 * create/edit, SoD-gated Approve) are ledgered — see
 * screens/parity/PurchaseOrders.parity.md. Receive Items (R5) is built as a
 * bespoke modal (ActionSpec.modal, PurchaseOrderReceiveModal.svelte). */

import type { ActionSpec, LedgerDescriptor } from '$kernel/descriptor'
import { nextStates } from '$kernel/ledger-core'
import { fetchPurchaseOrders, setPurchaseOrderStatus, type PurchaseOrderRow } from '../bridge/purchase-orders'
import PurchaseOrderReceiveModal from './PurchaseOrderReceiveModal.svelte'

// Mirrors purchase_order_service.go's UpdatePOStatus map exactly (source of
// truth for legal transitions — do not let this drift independently).
const PO_STATUS_TRANSITIONS: Record<string, string[]> = {
  Draft: ['Pending Approval', 'Approved', 'Sent', 'Cancelled'],
  'Pending Approval': ['Approved', 'Draft', 'Cancelled'],
  Approved: ['Sent', 'Cancelled'],
  Sent: ['Acknowledged', 'Partially Received', 'Received', 'Cancelled'],
  Acknowledged: ['Partially Received', 'Received', 'Cancelled'],
  'Partially Received': ['Received', 'Cancelled'],
  Received: [],
  Closed: [],
  Cancelled: [],
}

// Business-rule-as-data, duplicated from the backend (same drift risk the
// old screen already carried — see the parity doc's #4).
const PO_APPROVAL_THRESHOLD_BHD = 5000

// Receiving posts inventory and is routed exclusively through the Receive
// Items modal (receiveItemsAction below) — these targets never appear as a
// plain status-flip button, mirroring PurchaseOrdersScreen.svelte's
// RECEIVABLE_STATUSES gate.
const RECEIVABLE_STATUSES = ['Sent', 'Acknowledged', 'Partially Received']

const STATUS_BUTTON_LABELS: Record<string, string> = {
  Draft: 'Revert to Draft',
  'Pending Approval': 'Submit for Approval',
  Sent: 'Mark Sent',
  Acknowledged: 'Mark Acknowledged',
  Cancelled: 'Cancel PO',
}

function legalTargets(r: PurchaseOrderRow): string[] {
  let legal = nextStates(r.status, PO_STATUS_TRANSITIONS)
  if (r.status === 'Draft' && r.totalBhd > PO_APPROVAL_THRESHOLD_BHD) {
    legal = legal.filter((s) => s !== 'Sent' && s !== 'Approved')
  }
  if (RECEIVABLE_STATUSES.includes(r.status)) {
    legal = legal.filter((s) => s !== 'Partially Received' && s !== 'Received')
  }
  return legal
}

// Approve (Pending Approval → Approved) is deliberately excluded from every
// call below: it's a separate SoD-gated binding (ApprovePurchaseOrder), not
// a plain status flip — LEDGERED as a financial hot-zone (parity doc #4).
function statusAction(target: string): ActionSpec<PurchaseOrderRow> {
  const label = STATUS_BUTTON_LABELS[target] ?? `Mark ${target}`
  return {
    key: `status-${target}`,
    label,
    kind: 'row',
    visible: (r) => r != null && legalTargets(r).includes(target),
    // NOTE (Cancel specifically): the old screen captures a cancellation
    // reason via askForReason. The kernel's form archetype doesn't thread
    // the clicked row into FormSpec.submit yet — ActionHost captures
    // { action, row } but only forwards `spec` to FormModal, so a
    // row-scoped "confirm as form" isn't buildable without an engine
    // change. Downgraded to a plain confirm here; flagged as an ENGINE
    // gap in the parity doc rather than faked with a module-level hack.
    confirm: (r) => `${label} for ${r ? r.poNumber : 'this PO'}?`,
    run: async ({ row, reload }) => {
      if (!row) return
      await setPurchaseOrderStatus(row.id, target)
      await reload()
    },
  }
}

// L4 ejection (R5): the bespoke Receive Items modal — per-line receiving/
// rejected quantities → GRNItem[] via ReceiveAgainstPO. `run` is unused for
// modal actions (ActionHost dispatches to the component instead).
const receiveItemsAction: ActionSpec<PurchaseOrderRow> = {
  key: 'receive',
  label: 'Receive Items',
  kind: 'row',
  visible: (r) => r != null && RECEIVABLE_STATUSES.includes(r.status),
  modal: PurchaseOrderReceiveModal,
  run: () => {},
}

const STATUS_TONES = {
  Draft: 'neutral',
  'Pending Approval': 'warning',
  Approved: 'info',
  Sent: 'info',
  Acknowledged: 'info',
  'Partially Received': 'warning',
  Received: 'success',
  Closed: 'neutral',
  Cancelled: 'danger',
} as const

export const purchaseOrdersDescriptor: LedgerDescriptor<PurchaseOrderRow> = {
  entity: 'purchase-orders',
  title: 'Purchase Orders',
  fetch: fetchPurchaseOrders,
  id: (r) => r.id,
  searchText: (r) => `${r.poNumber} ${r.supplierName} ${r.paymentTerms}`,

  columns: [
    { key: 'poNumber', label: 'PO #', content: 'code', value: (r) => r.poNumber, minWidth: 120 },
    { key: 'supplierName', label: 'Supplier', content: 'name', value: (r) => r.supplierName, grow: true, minWidth: 220 },
    { key: 'poDate', label: 'PO Date', content: 'date', value: (r) => r.poDate, minWidth: 120 },
    { key: 'currency', label: 'Currency', content: 'text', value: (r) => r.currency, minWidth: 90 },
    {
      key: 'subtotalForeign',
      label: 'Net Amount',
      content: 'money',
      value: (r) => r.subtotalForeign,
      currency: (r) => r.currency,
      minWidth: 150,
    },
    { key: 'totalBhd', label: 'Total incl. VAT', content: 'money', value: (r) => r.totalBhd, minWidth: 150 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 150 },
  ],

  status: {
    value: (r) => r.status,
    tones: STATUS_TONES,
    // Declared legal-transition graph (ENGINE candidate, K1-B synthesis #2)
    // — the same table both gates row-action visibility here (legalTargets)
    // and documents the machine for audit purposes.
    transitions: PO_STATUS_TRANSITIONS,
  },

  summary: {
    metrics: [
      { label: 'Purchase Orders', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Total Value (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.totalBhd, 0),
      },
      {
        label: 'Pending Receipts',
        content: 'quantity',
        value: (rows) => rows.filter((r) => RECEIVABLE_STATUSES.includes(r.status)).length,
        tone: (rows) => (rows.some((r) => RECEIVABLE_STATUSES.includes(r.status)) ? 'warning' : 'neutral'),
      },
      {
        label: 'Fully Received',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.status === 'Received' || r.status === 'Closed').length,
      },
    ],
    distribution: {
      label: 'By status',
      value: (r) => r.status,
      tones: STATUS_TONES,
    },
  },

  // 'derive' (not the old screen's static tab list) so every status actually
  // present in the data shows a chip — fixes the census-flagged gap where
  // the old tabs omitted Cancelled/Pending Approval/Approved.
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

  actions: [
    statusAction('Pending Approval'),
    statusAction('Sent'),
    statusAction('Acknowledged'),
    statusAction('Draft'),
    statusAction('Cancelled'),
    receiveItemsAction,
  ],

  emptyMessage: 'No purchase orders yet. Raise the first one from an order.',
}
