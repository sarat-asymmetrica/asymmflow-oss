/* The second pilot: CustomersScreen as an EntityMaster descriptor.
 * Proves the archetype pattern generalizes past ledgers: same viewmodel,
 * same primitives, profile-centric rendering. */

import type { EntityDescriptor } from '$kernel/descriptor'
import { fetchCustomers, setCustomerStatus, type CustomerRow } from '../bridge/mock'

export const customersDescriptor: EntityDescriptor<CustomerRow> = {
  entity: 'customers',
  title: 'Customers',
  fetch: fetchCustomers,
  id: (r) => r.id,
  searchText: (r) => `${r.code} ${r.name} ${r.city} ${r.email}`,

  columns: [
    { key: 'code', label: 'Code', content: 'code', value: (r) => r.code, minWidth: 90 },
    { key: 'name', label: 'Customer', content: 'name', value: (r) => r.name, grow: true, minWidth: 200 },
    { key: 'city', label: 'City', content: 'text', value: (r) => r.city, minWidth: 110 },
    { key: 'balance', label: 'Balance', content: 'money', value: (r) => r.balance, minWidth: 140 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 110 },
  ],

  status: {
    value: (r) => r.status,
    tones: {
      Active: 'success',
      Dormant: 'neutral',
      'On Hold': 'warning',
      Blacklisted: 'danger',
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
      key: 'terms',
      label: 'Payment terms',
      options: 'derive',
      deriveValue: (r) => r.paymentTerms,
      predicate: (r, v) => r.paymentTerms === v,
    },
  ],

  actions: [
    {
      key: 'new',
      label: '+ New Customer',
      kind: 'screen',
      run: () => {
        /* form archetype arrives in a later wave */
      },
    },
    {
      key: 'hold',
      label: 'Put On Hold',
      kind: 'row',
      visible: (r) => r != null && r.status === 'Active',
      run: async ({ row, reload }) => {
        if (!row) return
        await setCustomerStatus(row.id, 'On Hold')
        await reload()
      },
    },
    {
      key: 'reactivate',
      label: 'Reactivate',
      kind: 'row',
      visible: (r) => r != null && (r.status === 'On Hold' || r.status === 'Dormant'),
      run: async ({ row, reload }) => {
        if (!row) return
        await setCustomerStatus(row.id, 'Active')
        await reload()
      },
    },
  ],

  profile: {
    heading: (r) => r.name,
    subheading: (r) => `${r.code}${r.city ? ` · ${r.city}` : ''}`,
    badge: {
      value: (r) => r.status,
      tones: {
        Active: 'success',
        Dormant: 'neutral',
        'On Hold': 'warning',
        Blacklisted: 'danger',
      },
    },
    kpis: [
      { label: 'Balance', content: 'money', value: (r) => r.balance },
      { label: 'Credit limit', content: 'money', value: (r) => r.creditLimit },
      { label: 'Open orders', content: 'quantity', value: (r) => r.openOrders },
    ],
    sections: [
      {
        title: 'Contact',
        fields: [
          { label: 'Phone', content: 'text', value: (r) => r.phone },
          { label: 'Email', content: 'text', value: (r) => r.email },
          { label: 'City', content: 'text', value: (r) => r.city },
        ],
      },
      {
        title: 'Commercial',
        fields: [
          { label: 'Payment terms', content: 'text', value: (r) => r.paymentTerms },
          { label: 'Credit limit', content: 'money', value: (r) => r.creditLimit },
          { label: 'Balance', content: 'money', value: (r) => r.balance },
          { label: 'Last order', content: 'date', value: (r) => r.lastOrderDate },
        ],
      },
    ],
  },

  emptyMessage: 'No customers yet. Import or add the first one.',
}
