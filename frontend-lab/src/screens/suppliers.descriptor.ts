/* SuppliersScreen as an EntityMaster descriptor. Direct sibling of the
 * Customers pilot. List columns are exactly what the real `ListSuppliers`
 * SELECT provides (recon-K2 #2) — tax_id/brands/address/bank are PROFILE-only.
 * Status is derived from `is_active` (SupplierMaster has no `status` field;
 * the old 3-state Active/Inactive/Pending vocabulary is a UI fiction — see
 * Suppliers.parity.md #1). */

import type { EntityDescriptor } from '$kernel/descriptor'
import { deleteSupplier, fetchSupplierProfile, fetchSuppliers, type SupplierRow } from '../bridge/suppliers'

export const suppliersDescriptor: EntityDescriptor<SupplierRow> = {
  entity: 'suppliers',
  title: 'Suppliers',
  fetch: fetchSuppliers,
  id: (r) => r.id,
  searchText: (r) => `${r.code} ${r.name} ${r.primaryContact} ${r.email}`,

  columns: [
    { key: 'code', label: 'Code', content: 'code', value: (r) => r.code, minWidth: 100 },
    { key: 'name', label: 'Supplier Name', content: 'name', value: (r) => r.name, grow: true, minWidth: 220 },
    { key: 'contact', label: 'Contact Person', content: 'text', value: (r) => r.primaryContact, minWidth: 170 },
    { key: 'phone', label: 'Phone', content: 'text', value: (r) => r.phone, minWidth: 140 },
    { key: 'email', label: 'Email', content: 'text', value: (r) => r.email, minWidth: 200 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 110 },
  ],

  status: {
    value: (r) => r.status,
    tones: {
      Active: 'success',
      Inactive: 'neutral',
    },
  },

  summary: {
    metrics: [
      { label: 'Suppliers', content: 'quantity', value: (rows) => rows.length },
      { label: 'Active', content: 'quantity', value: (rows) => rows.filter((r) => r.isActive).length },
      {
        // Percentage points as a plain number (house convention — see
        // GRNs' Acceptance Rate); the % lives in the label, not the value.
        label: 'Active Rate %',
        content: 'quantity',
        value: (rows) =>
          rows.length === 0 ? 0 : Math.round((rows.filter((r) => r.isActive).length / rows.length) * 1000) / 10,
      },
    ],
    distribution: {
      label: 'By supplier type',
      value: (r) => r.supplierType,
      tones: {
        Manufacturer: 'info',
        Distributor: 'success',
        Agent: 'warning',
        'Service Provider': 'neutral',
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
      key: 'supplierType',
      label: 'Supplier type',
      options: 'derive',
      deriveValue: (r) => r.supplierType,
      predicate: (r, v) => r.supplierType === v,
    },
  ],

  actions: [
    {
      key: 'delete',
      label: 'Delete',
      kind: 'row',
      confirm: (r) => `Delete ${r ? (r as SupplierRow).name : 'this supplier'}? This cannot be undone.`,
      run: async ({ row, reload }) => {
        if (!row) return
        await deleteSupplier(row.id)
        await reload()
      },
    },
  ],

  profile: {
    heading: (r) => r.name,
    subheading: (r) => `${r.code}${r.country ? ` · ${r.country}` : ''}`,
    // Second fetch on select: GetSupplierFullProfile fills TRN/bank/KPIs.
    enrich: (r) => fetchSupplierProfile(r.id),
    badge: {
      value: (r) => r.status,
      tones: {
        Active: 'success',
        Inactive: 'neutral',
      },
    },
    kpis: [
      { label: 'Total Purchases', content: 'money', value: (r) => r.totalPurchases },
      { label: 'Total POs', content: 'quantity', value: (r) => r.totalPOs },
      { label: 'Avg PO Value', content: 'money', value: (r) => r.avgPOValue },
      { label: 'Open Issues', content: 'quantity', value: (r) => r.openIssues },
    ],
    sections: [
      {
        title: 'Contact',
        fields: [
          { label: 'Primary contact', content: 'text', value: (r) => r.primaryContact },
          { label: 'Email', content: 'text', value: (r) => r.email },
          { label: 'Phone', content: 'text', value: (r) => r.phone },
          { label: 'Address', content: 'text', value: (r) => r.address },
        ],
      },
      {
        title: 'Commercial',
        fields: [
          { label: 'Payment terms', content: 'text', value: (r) => r.paymentTerms },
          { label: 'Lead time (days)', content: 'quantity', value: (r) => r.leadTimeDays },
          { label: 'Rating (/5)', content: 'quantity', value: (r) => r.rating },
          { label: 'Tax ID', content: 'code', value: (r) => r.taxId },
        ],
      },
      {
        title: 'Bank Details',
        fields: [
          { label: 'Bank', content: 'text', value: (r) => r.bankName },
          { label: 'Account', content: 'code', value: (r) => r.accountNumber },
          { label: 'IBAN', content: 'code', value: (r) => r.iban },
          { label: 'SWIFT', content: 'code', value: (r) => r.swiftCode },
        ],
      },
    ],
  },

  emptyMessage: 'No suppliers yet. Add the first one.',
}
