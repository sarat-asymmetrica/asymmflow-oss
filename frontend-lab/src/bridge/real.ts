/* The REAL bridge — generated Wails bindings adapted to kernel row shapes.
 * This file is the INTEG seam: ONE place where Go models meet descriptors
 * (parity #17/#20: Go-date parsing and instrumentation live here, never in
 * screens). Functions marked INTEG-GAP throw honestly — the archetypes
 * surface the error inline instead of pretending. */

import { DeleteCustomerInvoice, ListCustomerInvoices } from '$wails/go/main/FinanceService'
import { ListCustomers } from '$wails/go/main/CRMService'
import type { CustomerRow, InvoiceRow, NewInvoiceDraft } from './mock'
import { goDate, num, str } from './map'

/* ---- Invoices ---- */

function mapInvoice(inv: Record<string, unknown>): InvoiceRow {
  return {
    id: str(inv.id),
    number: str(inv.invoice_number),
    customer: str(inv.customer_name),
    division: str(inv.division),
    status: str(inv.status),
    issueDate: goDate(inv.invoice_date),
    dueDate: goDate(inv.due_date),
    amount: num(inv.grand_total_bhd),
    currency: 'BHD', // invoice grand totals are stored in BHD
  }
}

export async function fetchInvoicesPage(limit: number, offset: number): Promise<InvoiceRow[]> {
  const rows = await ListCustomerInvoices(limit, offset)
  return (rows ?? []).map((r) => mapInvoice(r as unknown as Record<string, unknown>))
}

export async function fetchInvoices(): Promise<InvoiceRow[]> {
  return fetchInvoicesPage(200, 0)
}

export async function deleteInvoice(id: string): Promise<void> {
  await DeleteCustomerInvoice(id)
}

export async function markInvoicePaid(_id: string): Promise<void> {
  throw new Error(
    'INTEG gap: settlement flows through customer receipts (ApplyCustomerReceiptToInvoice), not a status flip',
  )
}

export async function createInvoice(_draft: NewInvoiceDraft): Promise<void> {
  throw new Error('INTEG gap: real invoices are raised from an order (CreateInvoiceWithOptions)')
}

/* ---- Customers ---- */

function mapCustomer(c: Record<string, unknown>): CustomerRow {
  return {
    id: str(c.id),
    code: str(c.customer_code) || str(c.short_code) || str(c.customer_id),
    name: str(c.business_name) || str(c.trading_name),
    city: str(c.city),
    status: str(c.status) || 'Active',
    phone: str(c.primary_phone) || str(c.phone) || str(c.mobile_number),
    email: str(c.primary_email) || str(c.email),
    paymentTerms: str(c.payment_grade),
    creditLimit: num(c.credit_limit_bhd ?? c.credit_limit),
    balance: num(c.outstanding_bhd ?? c.balance_bhd),
    openOrders: num(c.open_orders),
    lastOrderDate: goDate(c.last_order_date),
  }
}

export async function fetchCustomers(): Promise<CustomerRow[]> {
  const rows = await ListCustomers(500, 0)
  return (rows ?? []).map((r) => mapCustomer(r as unknown as Record<string, unknown>))
}

export async function setCustomerStatus(_id: string, _status: string): Promise<void> {
  throw new Error('INTEG gap: customer status changes go through UpdateCustomer (full record)')
}

export async function customerOptions(): Promise<{ value: string; label: string }[]> {
  const rows = await fetchCustomers()
  return rows.map((c) => ({ value: c.name, label: c.name }))
}
