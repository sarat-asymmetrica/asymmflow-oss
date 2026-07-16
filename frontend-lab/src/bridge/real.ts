/* The REAL bridge — generated Wails bindings adapted to kernel row shapes.
 * This file is the INTEG seam: ONE place where Go models meet descriptors
 * (parity #17/#20: Go-date parsing and instrumentation live here, never in
 * screens). Functions marked INTEG-GAP throw honestly — the archetypes
 * surface the error inline instead of pretending. */

import { DeleteCustomerInvoice, ListCustomerInvoices, SendCustomerInvoice } from '$wails/go/main/FinanceService'
import { ListCustomers } from '$wails/go/main/CRMService'
import { CreateCustomerReceipt, GetCustomerFullProfile } from '$wails/go/main/App'
import type { main } from '$wails/go/models'
import type { CustomerProfilePatch, CustomerRow, InvoiceReceiptInput, InvoiceRow } from './mock'
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

/** Settlement = receipt capture (owner ruling G1.2), NOT a status flip. We call
 * CreateCustomerReceipt with the invoice bound: the server derives customer +
 * division from the invoice, creates the receipt, and applies it in ONE
 * transaction — funding a Payment row and advancing invoice payment state
 * (receipt_service.go). Deviation-of-record from the ruling's literal
 * `ApplyCustomerReceiptToInvoice`: that binding needs a pre-existing receipt id,
 * which a capture modal does not have; CreateCustomerReceipt(invoice-bound) IS
 * the create-and-apply path (it calls the same applyCustomerReceiptToInvoiceTx).
 * customer_id/name/division are left blank on purpose — the server fills them
 * from the invoice; sending client-side guesses would risk a cross-customer
 * mis-post. The receipt_date string is parsed server-side (no time.Time bridge). */
export async function recordCustomerReceipt(input: InvoiceReceiptInput): Promise<void> {
  const payload = {
    customer_id: '',
    customer_name: '',
    invoice_id: input.invoiceId,
    amount_bhd: input.amount,
    receipt_date: input.date,
    payment_method: input.method,
    reference: input.reference,
    division: '',
    notes: input.notes,
  }
  await CreateCustomerReceipt(payload as unknown as main.CustomerReceiptInput)
}

export async function sendInvoice(id: string): Promise<void> {
  // SendCustomerInvoice(id) — Draft → Sent. Server rejects any non-Draft status
  // and an invoice with no line items; the descriptor surfaces that honestly.
  await SendCustomerInvoice(id)
}

// Standalone invoice-create is RETIRED (owner ruling G1.3): invoices are raised
// from an order (Orders → Create Invoice via CreateInvoiceWithOptions), never
// conjured on this ledger. No createInvoice bridge fn exists any more.

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
    // CustomerFullProfile fields — blank/zero here (INTEG gap: ListCustomers
    // does not return them; GetCustomerFullProfile is a second fetch this
    // bridge does not wire). Mock generates full values. See Customers.parity.md.
    trn: '',
    industry: '',
    relationYears: 0,
    paymentTermsDays: 0,
    isCreditBlocked: false,
    arCurrent: 0,
    ar30: 0,
    ar60: 0,
    ar90: 0,
    rfqsFloated: 0,
    rfqsWon: 0,
    winRate: 0,
  }
}

export async function fetchCustomers(): Promise<CustomerRow[]> {
  const rows = await ListCustomers(500, 0)
  return (rows ?? []).map((r) => mapCustomer(r as unknown as Record<string, unknown>))
}

export async function setCustomerStatus(_id: string, _status: string): Promise<void> {
  throw new Error('INTEG gap: customer status changes go through UpdateCustomer (full record)')
}

/** Secondary profile fetch: GetCustomerFullProfile fills the depth ListCustomers
 * omits (TRN/industry/credit/AR-aging/RFQ win-rate). AR buckets: the summary's
 * two oldest (90-120 + 120+) collapse into this view's single "90+". */
export async function fetchCustomerProfile(id: string): Promise<CustomerProfilePatch> {
  const p = (await GetCustomerFullProfile(id)) as unknown as Record<string, unknown>
  const aging = (p.ar_aging_buckets ?? {}) as Record<string, unknown>
  return {
    trn: str(p.trn),
    industry: str(p.industry),
    relationYears: num(p.relation_years),
    paymentTermsDays: num(p.payment_terms_days),
    isCreditBlocked: Boolean(p.is_credit_blocked),
    arCurrent: num(aging.current),
    ar30: num(aging.days_30_60),
    ar60: num(aging.days_60_90),
    ar90: num(aging.days_90_120) + num(aging.days_120_plus),
    rfqsFloated: num(p.rfqs_floated),
    rfqsWon: num(p.rfqs_won),
    winRate: num(p.win_rate),
  }
}

export async function customerOptions(): Promise<{ value: string; label: string }[]> {
  const rows = await fetchCustomers()
  return rows.map((c) => ({ value: c.name, label: c.name }))
}
