/* UserManagementScreen as an EntityMaster descriptor. RBAC/SECURITY HOT-ZONE
 * (recon-K2): `CreateUser`/`UpdateUser` are both server-gated (`users:create`/
 * `users:update`) and `role_id`/`is_active` are privilege-bearing fields —
 * K2 builds read + summary + profile ONLY, no mutation actions. Password/
 * password_hash never appears anywhere in this descriptor or its bridge
 * (see bridge/users.ts header). Status is derived from `is_active` — `User`
 * has no `status` field, same 2-state fix as Suppliers (Users.parity.md #1). */

import type { EntityDescriptor } from '$kernel/descriptor'
import { fetchUsers, type UserRow } from '../bridge/users'

export const usersDescriptor: EntityDescriptor<UserRow> = {
  entity: 'users',
  title: 'Users',
  fetch: fetchUsers,
  id: (r) => r.id,
  searchText: (r) => `${r.fullName} ${r.username} ${r.email} ${r.department} ${r.roleName}`,

  columns: [
    { key: 'user', label: 'User', content: 'name', value: (r) => r.fullName, grow: true, minWidth: 200 },
    { key: 'role', label: 'Role', content: 'text', value: (r) => r.roleName, minWidth: 160 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 100 },
  ],

  status: {
    value: (r) => r.status,
    tones: {
      Active: 'success',
      Inactive: 'neutral',
    },
  },

  // Pure count/distribution strip — Users has no money fields at all, a
  // deliberate visual-diversity contrast against the finance-heavy ledgers.
  summary: {
    metrics: [
      { label: 'Total Users', content: 'quantity', value: (rows) => rows.length },
      { label: 'Active', content: 'quantity', value: (rows) => rows.filter((r) => r.isActive).length },
      {
        label: 'Inactive',
        content: 'quantity',
        value: (rows) => rows.filter((r) => !r.isActive).length,
        tone: (rows) => (rows.some((r) => !r.isActive) ? 'warning' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By role',
      value: (r) => r.roleName,
      tones: {
        Administrator: 'warning', // highest-privilege role — worth a visual flag (RBAC hygiene)
        'Sales Manager': 'info',
        'Sales Representative': 'neutral',
        'Finance Officer': 'info',
        'Warehouse Supervisor': 'neutral',
        'Procurement Officer': 'neutral',
        Viewer: 'neutral',
        // Unknown/blank roles render neutral by engine contract — never crash.
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
      key: 'role',
      label: 'Role',
      options: 'derive',
      deriveValue: (r) => r.roleName,
      predicate: (r, v) => r.roleName === v,
    },
  ],

  // No actions: Add User / Edit (role_id + is_active) are RBAC hot-zone
  // mutations, deliberately not built here — see Users.parity.md #1/#2.

  profile: {
    heading: (r) => r.fullName || r.displayName || r.username,
    subheading: (r) => `@${r.username}${r.roleName ? ` · ${r.roleName}` : ''}`,
    badge: {
      value: (r) => r.status,
      tones: {
        Active: 'success',
        Inactive: 'neutral',
      },
    },
    kpis: [{ label: 'Last Login', content: 'date', value: (r) => r.lastLoginAt }],
    sections: [
      {
        title: 'Account',
        fields: [
          { label: 'Username', content: 'code', value: (r) => r.username },
          { label: 'Email', content: 'text', value: (r) => r.email },
          { label: 'Department', content: 'text', value: (r) => r.department },
          { label: 'Job title', content: 'text', value: (r) => r.jobTitle },
        ],
      },
      {
        title: 'Access',
        fields: [
          { label: 'Role', content: 'text', value: (r) => r.roleName },
          { label: 'Active', content: 'status', value: (r) => r.status },
          { label: 'Must change password', content: 'text', value: (r) => (r.mustChangePassword ? 'Yes' : 'No') },
          { label: 'Last login', content: 'date', value: (r) => r.lastLoginAt },
        ],
      },
    ],
  },

  emptyMessage: 'No users yet.',
}
