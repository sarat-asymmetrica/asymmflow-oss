/* InventoryFulfillmentScreen as a descriptor. Read-only report — recon-K2
 * calls this "the closest-to-kernel-shape screen in the whole batch": no
 * create/edit/delete, no status transitions, every row is an outstanding
 * order line, not an entity with a profile. See InventoryFulfillment.parity.md. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import type { Tone } from '$kernel/tones'
import { fetchInventoryFulfillment, type InventoryFulfillmentRow } from '../bridge/inventory-fulfillment'

const shortageBucket = (r: InventoryFulfillmentRow): string => (r.shortageQuantity > 0 ? 'Shortage' : 'OK')

export const inventoryFulfillmentDescriptor: LedgerDescriptor<InventoryFulfillmentRow> = {
  entity: 'inventory-fulfillment',
  title: 'Inventory Fulfillment',
  fetch: fetchInventoryFulfillment,
  id: (r) => r.orderId,
  searchText: (r) => `${r.orderNumber} ${r.customerName} ${r.productCode} ${r.description}`,

  columns: [
    { key: 'orderNumber', label: 'Order #', content: 'code', value: (r) => r.orderNumber, minWidth: 140 },
    { key: 'customerName', label: 'Customer', content: 'name', value: (r) => r.customerName, grow: true, minWidth: 220 },
    { key: 'productCode', label: 'Product', content: 'code', value: (r) => r.productCode, minWidth: 110 },
    { key: 'description', label: 'Description', content: 'text', value: (r) => r.description, minWidth: 260 },
    { key: 'orderedQuantity', label: 'Ordered', content: 'quantity', value: (r) => r.orderedQuantity, minWidth: 90 },
    { key: 'deliveredQuantity', label: 'Delivered', content: 'quantity', value: (r) => r.deliveredQuantity, minWidth: 90 },
    {
      key: 'pendingQuantity',
      label: 'Pending',
      content: 'quantity',
      value: (r) => r.pendingQuantity,
      tone: (r) => (r.pendingQuantity > 0 ? 'warning' : 'neutral'),
      minWidth: 90,
    },
    { key: 'availableQuantity', label: 'In Stock', content: 'quantity', value: (r) => r.availableQuantity, minWidth: 90 },
    {
      key: 'shortageQuantity',
      label: 'Shortage',
      content: 'quantity',
      value: (r) => r.shortageQuantity,
      tone: (r) => (r.shortageQuantity > 0 ? 'danger' : 'neutral'),
      minWidth: 90,
    },
    { key: 'status', label: 'Order Status', content: 'status', value: (r) => r.status, minWidth: 130 },
  ],

  status: {
    value: (r) => r.status,
    tones: {
      Delivered: 'success',
      Invoiced: 'success',
      Closed: 'success',
      Complete: 'success',
      Pending: 'info',
      Processing: 'info',
      Open: 'info',
      Cancelled: 'danger',
      Lost: 'danger',
      // Unknown statuses render neutral by engine contract — never crash.
    },
  },

  summary: {
    metrics: [
      { label: 'Total Lines', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Total Shortage Qty',
        content: 'quantity',
        value: (rows) => rows.reduce((s, r) => s + r.shortageQuantity, 0),
        tone: (rows) => (rows.some((r) => r.shortageQuantity > 0) ? 'danger' : 'neutral'),
      },
      {
        label: 'Lines With Shortage',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.shortageQuantity > 0).length,
        tone: (rows) => (rows.some((r) => r.shortageQuantity > 0) ? 'danger' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By shortage',
      value: shortageBucket,
      tones: { Shortage: 'danger' as Tone, OK: 'success' as Tone },
    },
  },

  filters: [
    {
      key: 'status',
      label: 'Order Status',
      options: 'derive',
      deriveValue: (r) => r.status,
      predicate: (r, v) => r.status === v,
    },
    {
      key: 'shortage',
      label: 'Shortage',
      options: 'derive',
      deriveValue: shortageBucket,
      predicate: (r, v) => shortageBucket(r) === v,
    },
  ],

  // No actions: row-click → Open Order is a cross-screen nav (INTEG, needs
  // the app-shell router — see InventoryFulfillment.parity.md #2). No
  // create/edit/delete exists for this report in the old screen either.
  emptyMessage: 'No outstanding fulfillment lines. Every order is fully delivered.',
}
