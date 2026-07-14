/* The screen registry — the single source of "what screens exist" for the
 * kernel app. Each entry maps a nav key to an archetype + its descriptor (or a
 * bespoke component). App.svelte renders from this list; it grows wave by wave
 * and becomes the real sidebar nav at K5. Orchestrator-owned merge point so
 * parallel build agents never contend on it.
 *
 * `descriptor` is intentionally loosely typed here: each archetype re-narrows
 * its own descriptor at the render site (the `bind:this` generic-erasure seam,
 * KERNEL lesson 2). Descriptor files stay fully typed. */

import type { Component } from 'svelte'
import { invoicesDescriptor } from './invoices.descriptor'
import { customersDescriptor } from './customers.descriptor'
import { ordersDescriptor } from './orders.descriptor'
import { deliveryNotesDescriptor } from './delivery-notes.descriptor'
import { rfqsDescriptor } from './rfqs.descriptor'
import { offersDescriptor } from './offers.descriptor'
import { grnsDescriptor } from './grns.descriptor'
import { purchaseOrdersDescriptor } from './purchase-orders.descriptor'
import { paymentsDescriptor } from './payments.descriptor'
import { creditNotesDescriptor } from './credit-notes.descriptor'
import { supplierInvoicesDescriptor } from './supplier-invoices.descriptor'
import { supplierPaymentsDescriptor } from './supplier-payments.descriptor'
import { chequeRegisterDescriptor } from './cheque-register.descriptor'
import { expensesDescriptor } from './expenses.descriptor'
import { suppliersDescriptor } from './suppliers.descriptor'
import { usersDescriptor } from './users.descriptor'
import { inventoryFulfillmentDescriptor } from './inventory-fulfillment.descriptor'
import { mainDashboardDescriptor } from './dashboards/main-dashboard.hub'
import { crmCustomerHubDescriptor } from './dashboards/crm-customer.hub'
import { crmSupplierHubDescriptor } from './dashboards/crm-supplier.hub'
import { financeOverviewDescriptor } from './dashboards/finance-overview.hub'
import Showcase from './Showcase.svelte'

export type ArchetypeKind = 'ledger' | 'entity' | 'hub' | 'bespoke'

export interface ScreenEntry {
  key: string
  label: string
  /** Nav grouping — Sales / Finance / Operations / People / System. */
  group: string
  archetype: ArchetypeKind
  /** For ledger/entity/hub archetypes. */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  descriptor?: any
  /** For bespoke/hub screens rendered by a hand-written component. */
  component?: Component
}

export const screens: ScreenEntry[] = [
  // K3 — Hub archetype + dashboards
  { key: 'dashboard', label: 'Dashboard', group: 'Home', archetype: 'hub', descriptor: mainDashboardDescriptor },
  { key: 'finance-overview', label: 'Finance Overview', group: 'Finance', archetype: 'hub', descriptor: financeOverviewDescriptor },
  { key: 'crm-customer', label: 'CRM Customer Overview', group: 'Sales', archetype: 'hub', descriptor: crmCustomerHubDescriptor },
  { key: 'crm-supplier', label: 'CRM Supplier Overview', group: 'Operations', archetype: 'hub', descriptor: crmSupplierHubDescriptor },
  // Pilots
  { key: 'invoices', label: 'Invoices', group: 'Finance', archetype: 'ledger', descriptor: invoicesDescriptor },
  { key: 'customers', label: 'Customers', group: 'Sales', archetype: 'entity', descriptor: customersDescriptor },
  // K1 — Ledger blitz (batch 1)
  { key: 'orders', label: 'Orders', group: 'Sales', archetype: 'ledger', descriptor: ordersDescriptor },
  { key: 'rfqs', label: 'RFQs', group: 'Sales', archetype: 'ledger', descriptor: rfqsDescriptor },
  { key: 'offers', label: 'Offers', group: 'Sales', archetype: 'ledger', descriptor: offersDescriptor },
  { key: 'purchase-orders', label: 'Purchase Orders', group: 'Operations', archetype: 'ledger', descriptor: purchaseOrdersDescriptor },
  { key: 'delivery-notes', label: 'Delivery Notes', group: 'Operations', archetype: 'ledger', descriptor: deliveryNotesDescriptor },
  { key: 'grns', label: 'Goods Received', group: 'Operations', archetype: 'ledger', descriptor: grnsDescriptor },
  // K1 — Ledger blitz (batch 2, finance)
  { key: 'payments', label: 'Payments', group: 'Finance', archetype: 'ledger', descriptor: paymentsDescriptor },
  { key: 'credit-notes', label: 'Credit Notes', group: 'Finance', archetype: 'ledger', descriptor: creditNotesDescriptor },
  { key: 'supplier-invoices', label: 'Supplier Invoices', group: 'Finance', archetype: 'ledger', descriptor: supplierInvoicesDescriptor },
  { key: 'supplier-payments', label: 'Supplier Payments', group: 'Finance', archetype: 'ledger', descriptor: supplierPaymentsDescriptor },
  { key: 'cheque-register', label: 'Cheque Register', group: 'Finance', archetype: 'ledger', descriptor: chequeRegisterDescriptor },
  { key: 'expenses', label: 'Expenses', group: 'Finance', archetype: 'ledger', descriptor: expensesDescriptor },
  // K2 — Entity blitz
  { key: 'suppliers', label: 'Suppliers', group: 'Operations', archetype: 'entity', descriptor: suppliersDescriptor },
  { key: 'inventory-fulfillment', label: 'Inventory Fulfillment', group: 'Operations', archetype: 'ledger', descriptor: inventoryFulfillmentDescriptor },
  { key: 'users', label: 'Users', group: 'System', archetype: 'entity', descriptor: usersDescriptor },
  // Lab
  { key: 'showcase', label: 'Showcase', group: 'Lab', archetype: 'bespoke', component: Showcase },
]

/** Stable group order for the nav. Unknown groups append alphabetically. */
export const GROUP_ORDER = ['Home', 'Sales', 'Finance', 'Operations', 'People', 'System', 'Lab']

export function screensByGroup(): { group: string; items: ScreenEntry[] }[] {
  const groups = new Map<string, ScreenEntry[]>()
  for (const s of screens) {
    if (!groups.has(s.group)) groups.set(s.group, [])
    groups.get(s.group)!.push(s)
  }
  const order = (g: string) => {
    const i = GROUP_ORDER.indexOf(g)
    return i === -1 ? GROUP_ORDER.length : i
  }
  return [...groups.entries()]
    .sort((a, b) => order(a[0]) - order(b[0]) || a[0].localeCompare(b[0]))
    .map(([group, items]) => ({ group, items }))
}
