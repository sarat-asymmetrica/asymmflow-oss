/* Bank Accounts as a standalone descriptor — K4 SettingsScreen split
 * (see screens/parity/Settings.parity.md). FINANCIAL hot-zone: IBAN/SWIFT/
 * account-number fields, synthetic-only data. Create = screen action;
 * Edit = row-aware form pre-filled from the selected row; Delete = confirm. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import {
  createBankAccount,
  currencyOptions,
  deleteBankAccount,
  fetchBankAccounts,
  updateBankAccount,
  type BankAccountDraft,
  type BankAccountRow,
} from '../bridge/bank-accounts'

const STATUS_OPTIONS = [
  { value: 'Active', label: 'Active' },
  { value: 'Inactive', label: 'Inactive' },
]

const bankAccountFields: FormSpec<BankAccountDraft>['fields'] = [
  { key: 'name', label: 'Account Name', kind: 'text', required: true, placeholder: 'e.g. Main Operating Account' },
  { key: 'bankName', label: 'Bank', kind: 'text', required: true },
  { key: 'accountNumber', label: 'Account Number', kind: 'text' },
  { key: 'currency', label: 'Currency', kind: 'select', required: true, options: currencyOptions() },
  { key: 'iban', label: 'IBAN', kind: 'text', required: true },
  { key: 'swiftCode', label: 'SWIFT / BIC', kind: 'text' },
  { key: 'status', label: 'Status', kind: 'select', required: true, options: STATUS_OPTIONS },
]

const createBankAccountForm: FormSpec<BankAccountDraft> = {
  title: 'New Bank Account',
  submitLabel: 'Create Account',
  initial: () => ({
    name: '',
    bankName: '',
    accountNumber: '',
    currency: currencyOptions()[0]?.value ?? 'BHD',
    iban: '',
    swiftCode: '',
    status: 'Active',
  }),
  fields: bankAccountFields,
  submit: (draft) => createBankAccount(draft),
}

// Row-aware edit form: `row` is the clicked account, so `initial` seeds the
// draft from its current field values (KERNEL form.ts row-aware contract).
const editBankAccountForm: FormSpec<BankAccountDraft> = {
  title: 'Edit Bank Account',
  submitLabel: 'Save Changes',
  initial: (row) => {
    const r = row as BankAccountRow
    return {
      name: r.name,
      bankName: r.bankName,
      accountNumber: r.accountNumber,
      currency: r.currency,
      iban: r.iban,
      swiftCode: r.swiftCode,
      status: r.status,
    }
  },
  fields: bankAccountFields,
  submit: async (draft, row) => {
    const r = row as BankAccountRow
    await updateBankAccount(r.id, draft)
  },
}

export const bankAccountsDescriptor: LedgerDescriptor<BankAccountRow> = {
  entity: 'bank-accounts',
  title: 'Bank Accounts',
  fetch: fetchBankAccounts,
  id: (r) => r.id,
  searchText: (r) => `${r.name} ${r.bankName} ${r.iban}`,

  columns: [
    { key: 'name', label: 'Account', content: 'name', value: (r) => r.name, grow: true, minWidth: 220 },
    { key: 'bankName', label: 'Bank', content: 'text', value: (r) => r.bankName, minWidth: 170 },
    { key: 'currency', label: 'Currency', content: 'text', value: (r) => r.currency, minWidth: 90 },
    { key: 'iban', label: 'IBAN', content: 'code', value: (r) => r.iban, minWidth: 220 },
    { key: 'swiftCode', label: 'SWIFT', content: 'code', value: (r) => r.swiftCode, minWidth: 110 },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 100 },
  ],

  status: {
    value: (r) => r.status,
    tones: { Active: 'success', Inactive: 'neutral' },
  },

  summary: {
    metrics: [
      { label: 'Bank Accounts', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Active',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.status === 'Active').length,
      },
      {
        label: 'Currencies',
        content: 'quantity',
        value: (rows) => new Set(rows.map((r) => r.currency)).size,
      },
    ],
    distribution: {
      label: 'By currency',
      value: (r) => r.currency,
      tones: { BHD: 'info', USD: 'success', EUR: 'warning', GBP: 'neutral', SAR: 'neutral' },
    },
  },

  filters: [
    {
      key: 'currency',
      label: 'Currency',
      options: 'derive',
      deriveValue: (r) => r.currency,
      predicate: (r, v) => r.currency === v,
    },
    {
      key: 'status',
      label: 'Status',
      options: 'derive',
      deriveValue: (r) => r.status,
      predicate: (r, v) => r.status === v,
    },
  ],

  actions: [
    {
      key: 'new',
      label: '+ New Account',
      kind: 'screen',
      form: createBankAccountForm,
      run: () => {
        /* form actions submit via their FormSpec; run is unused */
      },
    },
    {
      key: 'edit',
      label: 'Edit',
      kind: 'row',
      form: editBankAccountForm,
      run: () => {
        /* form actions submit via their FormSpec; run is unused */
      },
    },
    {
      key: 'delete',
      label: 'Delete',
      kind: 'row',
      confirm: (r) => `Delete ${r ? (r as BankAccountRow).name : 'this account'}? This cannot be undone.`,
      run: async ({ row, reload }) => {
        if (!row) return
        await deleteBankAccount(row.id)
        await reload()
      },
    },
  ],

  emptyMessage: 'No bank accounts yet. Add the first one.',
}
