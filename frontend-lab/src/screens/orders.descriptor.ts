/* OrdersScreen as a descriptor. Old frontend: ~1,800 lines incl. the
 * traceability chain, delivery-status batch fetch, and cross-screen handoffs
 * to DN/PO/Invoice/Proforma/Project — all ledgered here (see parity doc). */

import type { LedgerDescriptor } from '$kernel/descriptor'
import { fetchOrdersPage, markOrderDelivered, type OrderRow } from '../bridge/orders'

export const ordersDescriptor: LedgerDescriptor<OrderRow> = {
  entity: 'orders',
  title: 'Orders',
  fetch: () => fetchOrdersPage(200, 0),
  fetchPage: fetchOrdersPage,
  pageSize: 100,
  id: (r) => r.id,
  searchText: (r) => `${r.orderNumber} ${r.customerName} ${r.customerPoNumber}`,

  columns: [
    { key: 'orderNumber', label: 'Order Number', content: 'code', value: (r) => r.orderNumber, minWidth: 140 },
    { key: 'customerName', label: 'Customer', content: 'name', value: (r) => r.customerName, grow: true, minWidth: 220 },
    { key: 'customerPoNumber', label: 'RFQ/Offer Ref', content: 'text', value: (r) => r.customerPoNumber, minWidth: 140 },
    { key: 'totalValueBhd', label: 'Total (BHD)', content: 'money', value: (r) => r.totalValueBhd, minWidth: 140 },
    { key: 'orderDate', label: 'Order Date', content: 'date', value: (r) => r.orderDate, minWidth: 120 },
    {
      key: 'delivery',
      label: 'Delivery',
      content: 'text',
      value: (r) => `${r.deliveryPercent}%`,
      minWidth: 100,
      tone: (r) => (r.deliveryPercent >= 100 ? 'success' : r.deliveryPercent > 0 ? 'info' : 'neutral'),
    },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 160 },
  ],

  status: {
    value: (r) => r.status,
    tones: {
      Confirmed: 'info',
      InProgress: 'info',
      PartiallyDelivered: 'warning',
      FullyDelivered: 'success',
      Invoiced: 'success',
      Cancelled: 'neutral',
      // Unknown/legacy status strings render neutral by engine contract —
      // the old screen's normalizeOrderStatusKey() is not reproduced here.
    },
  },

  summary: {
    metrics: [
      { label: 'Orders', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Total Value (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.totalValueBhd, 0),
      },
      {
        label: 'Cancelled',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.status === 'Cancelled').length,
        tone: (rows) => (rows.some((r) => r.status === 'Cancelled') ? 'warning' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By status',
      value: (r) => r.status,
      tones: {
        Confirmed: 'info',
        InProgress: 'info',
        PartiallyDelivered: 'warning',
        FullyDelivered: 'success',
        Invoiced: 'success',
        Cancelled: 'neutral',
      },
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
      key: 'year',
      label: 'Year',
      options: 'derive',
      deriveValue: (r) => r.orderDate.slice(0, 4),
      predicate: (r, v) => r.orderDate.slice(0, 4) === v,
    },
    {
      key: 'customer',
      label: 'Customer',
      options: 'derive',
      deriveValue: (r) => r.customerName,
      predicate: (r, v) => r.customerName === v,
    },
  ],

  actions: [
    {
      // Fulfillment-only status flip (QuickMarkOrderDelivered) — the only
      // Orders row action that isn't a financial hot-zone or line-item form.
      // Create DN/PO/Invoice/Proforma, Start Project, cascade-delete are all
      // ledgered (see parity #3/#4) — not built in K1.
      key: 'markDelivered',
      label: 'Mark as Delivered',
      kind: 'row',
      visible: (r) => r != null && r.status !== 'Cancelled' && r.status !== 'FullyDelivered',
      confirm: (r) => `Mark ${r ? (r as OrderRow).orderNumber : 'this order'} as fully delivered?`,
      run: async ({ row, reload }) => {
        if (!row) return
        await markOrderDelivered(row.id)
        await reload()
      },
    },
  ],

  emptyMessage: 'No orders yet. Orders are raised from a Won offer.',
}
