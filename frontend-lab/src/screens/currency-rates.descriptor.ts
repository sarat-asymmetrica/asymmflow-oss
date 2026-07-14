/* Currency Rates as a standalone descriptor — K4 SettingsScreen split
 * (see screens/parity/Settings.parity.md). One rate per currency against
 * BHD (see bridge/currency-rates.ts header) — "Create"/"Edit" are the same
 * upsert-by-currency form (SetExchangeRate), so a single screen action
 * covers both; there's no per-row edit because the currency itself is the
 * identity SetExchangeRate keys on. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import type { FormSpec } from '$kernel/form'
import {
  fetchCurrencyRates,
  setCurrencyRate,
  type CurrencyRateDraft,
  type CurrencyRateRow,
} from '../bridge/currency-rates'

const setRateForm: FormSpec<CurrencyRateDraft> = {
  title: 'Set Exchange Rate',
  submitLabel: 'Save Rate',
  initial: () => ({
    currency: '',
    rate: null,
    asOfDate: new Date().toISOString().slice(0, 10),
    source: 'Manual Entry',
  }),
  fields: [
    { key: 'currency', label: 'Currency', kind: 'text', required: true, placeholder: 'e.g. USD' },
    {
      key: 'rate',
      label: 'Rate (per BHD)',
      kind: 'number',
      required: true,
      step: '0.0001',
      validate: (v) => (typeof v === 'number' && v <= 0 ? 'Rate must be positive' : null),
    },
    { key: 'asOfDate', label: 'As-of Date', kind: 'date', required: true },
    { key: 'source', label: 'Source', kind: 'text', placeholder: 'e.g. CBB Reference' },
  ],
  submit: (draft) => setCurrencyRate(draft),
}

export const currencyRatesDescriptor: LedgerDescriptor<CurrencyRateRow> = {
  entity: 'currency-rates',
  title: 'Currency Rates',
  fetch: fetchCurrencyRates,
  id: (r) => r.id,
  searchText: (r) => `${r.currency} ${r.source}`,

  columns: [
    { key: 'currency', label: 'Currency', content: 'code', value: (r) => r.currency, minWidth: 100 },
    { key: 'rate', label: 'Rate (per BHD)', content: 'quantity', value: (r) => r.rate, minWidth: 130 },
    { key: 'asOfDate', label: 'As-of Date', content: 'date', value: (r) => r.asOfDate, minWidth: 110 },
    { key: 'source', label: 'Source', content: 'text', value: (r) => r.source, minWidth: 140 },
    { key: 'notes', label: 'Notes', content: 'text', value: (r) => r.notes, grow: true, minWidth: 180 },
  ],

  summary: {
    metrics: [{ label: 'Currencies Tracked', content: 'quantity', value: (rows) => rows.length }],
  },

  actions: [
    {
      key: 'set',
      label: '+ Set Rate',
      kind: 'screen',
      form: setRateForm,
      run: () => {
        /* form actions submit via their FormSpec; run is unused */
      },
    },
  ],

  emptyMessage: 'No exchange rates set yet.',
}
