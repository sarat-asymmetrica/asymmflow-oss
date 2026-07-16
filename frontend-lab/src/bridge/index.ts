/* The bridge switch: real Wails bindings when the runtime is present,
 * deterministic adversarial mock otherwise. Descriptors import from HERE —
 * they never know which side they're talking to. That indifference is the
 * INTEG thesis being tested. */

import * as mock from './mock'
import * as real from './real'
import { pick, usingWails } from './runtime'

export type { CustomerRow, InvoiceRow, NewInvoiceDraft } from './mock'
export { usingWails }

export const fetchInvoices = (): ReturnType<typeof mock.fetchInvoices> =>
  pick(real.fetchInvoices, mock.fetchInvoices)()
export const fetchInvoicesPage = (l: number, o: number): ReturnType<typeof mock.fetchInvoicesPage> =>
  pick(real.fetchInvoicesPage, mock.fetchInvoicesPage)(l, o)
export const markInvoicePaid = (id: string): Promise<void> =>
  pick(real.markInvoicePaid, mock.markInvoicePaid)(id)
export const sendInvoice = (id: string): Promise<void> =>
  pick(real.sendInvoice, mock.sendInvoice)(id)
export const createInvoice = (d: mock.NewInvoiceDraft): Promise<void> =>
  pick(real.createInvoice, mock.createInvoice)(d)
export const deleteInvoice = (id: string): Promise<void> =>
  pick(real.deleteInvoice, mock.deleteInvoice)(id)
export const customerOptions = (): ReturnType<typeof mock.customerOptions> =>
  pick(real.customerOptions, mock.customerOptions)()
export const fetchCustomers = (): ReturnType<typeof mock.fetchCustomers> =>
  pick(real.fetchCustomers, mock.fetchCustomers)()
export const setCustomerStatus = (id: string, s: string): Promise<void> =>
  pick(real.setCustomerStatus, mock.setCustomerStatus)(id, s)
export const fetchCustomerProfile = (id: string): ReturnType<typeof mock.fetchCustomerProfile> =>
  pick(real.fetchCustomerProfile, mock.fetchCustomerProfile)(id)

/** Division vocabulary now comes from the divisions store (L7): the real
 * `GetDivisionRegistry` under Wails, the BUILTIN synthetic fallback under mock.
 * ONE source for every division dropdown (I1). Read lazily — see the store. */
export { getDivisionOptions as divisionOptions } from '../stores/divisions.svelte'
