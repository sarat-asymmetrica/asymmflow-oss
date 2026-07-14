/* OffersScreen as a descriptor. Old frontend: OffersScreen.svelte — every
 * mutating capability (create/edit via CostingSheet handoff, Won/Lost,
 * notes thread, PDF) is LEDGER per the K1 build brief; see Offers.parity.md.
 * This descriptor is the read/list/filter/summary surface at parity. */

import type { LedgerDescriptor } from '$kernel/descriptor'
import { fetchOffers, validityTone, type OfferRow } from '../bridge/offers'

export const offersDescriptor: LedgerDescriptor<OfferRow> = {
  entity: 'offers',
  title: 'Offers',
  fetch: fetchOffers,
  id: (r) => r.id,
  searchText: (r) => `${r.number} ${r.customer}`,

  columns: [
    { key: 'number', label: 'Offer #', content: 'code', value: (r) => r.number, minWidth: 120 },
    { key: 'customer', label: 'Customer', content: 'name', value: (r) => r.customer, grow: true, minWidth: 220 },
    { key: 'quotationDate', label: 'Date', content: 'date', value: (r) => r.quotationDate, minWidth: 120 },
    {
      key: 'validityDate',
      label: 'Valid Until',
      content: 'date',
      value: (r) => r.validityDate,
      minWidth: 120,
      tone: validityTone,
    },
    { key: 'value', label: 'Total Value', content: 'money', value: (r) => r.value, minWidth: 140 },
    { key: 'stage', label: 'Status', content: 'status', value: (r) => r.stage, minWidth: 110 },
  ],

  status: {
    value: (r) => r.stage,
    tones: {
      RFQ: 'neutral',
      Quoted: 'info',
      Expired: 'warning',
      Won: 'success',
      Lost: 'danger',
      // Unknown stages render neutral by engine contract — never crash.
    },
    transitions: {
      RFQ: ['Quoted'],
      Quoted: ['Won', 'Lost', 'Expired'],
      Expired: ['Lost'],
      Won: [],
      Lost: [],
    },
  },

  summary: {
    metrics: [
      { label: 'Offers', content: 'quantity', value: (rows) => rows.length },
      {
        label: 'Total Value (BHD)',
        content: 'money',
        value: (rows) => rows.reduce((s, r) => s + r.value, 0),
      },
      {
        label: 'Won',
        content: 'quantity',
        value: (rows) => rows.filter((r) => r.stage === 'Won').length,
        tone: (rows) => (rows.some((r) => r.stage === 'Won') ? 'success' : 'neutral'),
      },
    ],
    distribution: {
      label: 'By stage',
      value: (r) => r.stage,
      tones: {
        RFQ: 'neutral',
        Quoted: 'info',
        Expired: 'warning',
        Won: 'success',
        Lost: 'danger',
      },
    },
  },

  filters: [
    {
      // Already L7-clean in the old screen (sourced from the divisions
      // registry, not a hardcoded literal) — derived here the same way.
      key: 'division',
      label: 'Division',
      options: 'derive',
      deriveValue: (r) => r.division,
      predicate: (r, v) => r.division === v,
    },
    {
      key: 'stage',
      label: 'Status',
      options: 'derive',
      deriveValue: (r) => r.stage,
      predicate: (r, v) => r.stage === v,
    },
  ],

  // No actions: Create/Edit (CostingSheet handoff), Won/Lost (financial
  // hot-zone, document-creating/terminal), Notes thread, and PDF are all
  // LEDGER per the build brief — see Offers.parity.md #1, #4, #5, #6, #7.
  // "View" needs no action of its own: selecting a row already opens the
  // default detail panel.

  emptyMessage: 'No offers yet. Raise the first one from an RFQ.',
}
