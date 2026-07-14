/* The second pilot: CustomersScreen as an EntityMaster descriptor.
 * Proves the archetype pattern generalizes past ledgers: same viewmodel,
 * same primitives, profile-centric rendering.
 *
 * K2 widen (see Customers.parity.md): the pilot's original profile (Contact +
 * Commercial, 3 KPIs) was a strict subset of CustomerFullProfile. Widened to
 * carry TRN/industry/relationship-years/payment-terms-days/credit-block,
 * AR aging, and RFQ performance — all profile-only fields the real bridge
 * blanks/zeroes until GetCustomerFullProfile is wired (K5). */

import type { EntityDescriptor } from '$kernel/descriptor'
import { fetchCustomers, setCustomerStatus, type CustomerRow } from '../bridge'

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

  // Visual-diversity strip (K2 widen): count/active/rate + total outstanding,
  // plus a status distribution bar — same shape as Invoices/GRNs/Suppliers.
  summary: {
    metrics: [
      { label: 'Customers', content: 'quantity', value: (rows) => rows.length },
      { label: 'Active', content: 'quantity', value: (rows) => rows.filter((r) => r.status === 'Active').length },
      {
        // Percentage points as a plain number (house convention — see GRNs'
        // Acceptance Rate); the % lives in the label, not the value.
        label: 'Active Rate %',
        content: 'quantity',
        value: (rows) =>
          rows.length === 0
            ? 0
            : Math.round((rows.filter((r) => r.status === 'Active').length / rows.length) * 1000) / 10,
      },
      {
        label: 'Total Outstanding (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.balance, 0),
      },
    ],
    distribution: {
      label: 'By status',
      value: (r) => r.status,
      tones: {
        Active: 'success',
        Dormant: 'neutral',
        'On Hold': 'warning',
        Blacklisted: 'danger',
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
    // A credit-blocked customer's Balance shows in danger red (ProfileKpiSpec.tone).
    kpis: [
      {
        label: 'Balance',
        content: 'money',
        value: (r) => r.balance,
        tone: (r) => (r.isCreditBlocked ? 'danger' : 'neutral'),
      },
      { label: 'Credit limit', content: 'money', value: (r) => r.creditLimit },
      { label: 'Open orders', content: 'quantity', value: (r) => r.openOrders },
      { label: 'RFQ Win Rate %', content: 'quantity', value: (r) => r.winRate },
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
          { label: 'TRN', content: 'code', value: (r) => r.trn },
          { label: 'Industry', content: 'text', value: (r) => r.industry },
          { label: 'Relationship (years)', content: 'quantity', value: (r) => r.relationYears },
          { label: 'Payment terms', content: 'text', value: (r) => r.paymentTerms },
          { label: 'Payment terms (days)', content: 'quantity', value: (r) => r.paymentTermsDays },
          { label: 'Credit limit', content: 'money', value: (r) => r.creditLimit },
          { label: 'Balance', content: 'money', value: (r) => r.balance },
          { label: 'Credit blocked', content: 'text', value: (r) => (r.isCreditBlocked ? 'Yes' : 'No') },
          { label: 'Last order', content: 'date', value: (r) => r.lastOrderDate },
        ],
      },
      {
        title: 'Receivables Aging',
        fields: [
          { label: 'Current', content: 'money', value: (r) => r.arCurrent },
          { label: '30 days', content: 'money', value: (r) => r.ar30 },
          { label: '60 days', content: 'money', value: (r) => r.ar60 },
          { label: '90+ days', content: 'money', value: (r) => r.ar90 },
        ],
      },
      {
        title: 'RFQ Performance',
        fields: [
          { label: 'RFQs floated', content: 'quantity', value: (r) => r.rfqsFloated },
          { label: 'RFQs won', content: 'quantity', value: (r) => r.rfqsWon },
          { label: 'Win rate %', content: 'quantity', value: (r) => r.winRate },
        ],
      },
    ],
  },

  emptyMessage: 'No customers yet. Import or add the first one.',
}
