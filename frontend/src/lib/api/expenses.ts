import { ApproveExpenseEntry, CreateExpenseCategory, CreateExpenseEntry, CreateExpenseFromBankCandidate, CreateExpenseVendor, DeleteExpenseCategory, DeleteExpenseEntry, DeleteExpenseVendor, DeleteRecurringExpense, CreateRecurringExpense, GenerateRecurringExpenses, ListBankExpenseCandidates, ListExpenseCategories, ListExpenseDashboardSummary, ListExpenseEntries, ListExpenseVendors, ListRecurringExpenses, MarkExpenseEntryPaid, PostExpenseEntry, RejectExpenseEntry, SubmitExpenseEntry } from "../../../wailsjs/go/main/FinanceService";
import { main, finance } from "../../../wailsjs/go/models";
import { buildWailsInput, normalizeWailsDateTime } from "$lib/utils/wailsInterop";

const isDesktop = () => Boolean((window as any)?.go?.main?.App);

function toExpenseEntry(entry: finance.ExpenseEntry): ExpenseEntry {
  return {
    id: entry.id,
    entry_number: entry.entry_number,
    division: entry.division,
    expense_date: normalizeWailsDateTime(entry.expense_date) || "",
    due_date: normalizeWailsDateTime(entry.due_date),
    description: entry.description,
    category_id: entry.category_id,
    category_name: entry.category_name,
    vendor_id: entry.vendor_id,
    vendor_name: entry.vendor_name,
    source_type: entry.source_type,
    source_ref_id: entry.source_ref_id,
    bank_expense_entry_id: entry.bank_expense_entry_id,
    project_id: entry.project_id,
    customer_id: entry.customer_id,
    opportunity_id: entry.opportunity_id,
    order_id: entry.order_id,
    cost_center: entry.cost_center,
    currency: entry.currency,
    amount: entry.amount,
    vat_amount: entry.vat_amount,
    total_amount: entry.total_amount,
    status: entry.status,
    payment_status: entry.payment_status,
    payment_method: entry.payment_method,
    payment_reference: entry.payment_reference,
    bank_account_id: entry.bank_account_id,
    notes: entry.notes,
    approved_at: normalizeWailsDateTime(entry.approved_at),
    approved_by: entry.approved_by,
    posted_at: normalizeWailsDateTime(entry.posted_at),
    paid_at: normalizeWailsDateTime(entry.paid_at),
  };
}

function toRecurringExpense(entry: finance.RecurringExpense): RecurringExpense {
  return {
    id: entry.id,
    name: entry.name,
    division: entry.division,
    description: entry.description,
    category_id: entry.category_id,
    category_name: entry.category_name,
    vendor_id: entry.vendor_id,
    vendor_name: entry.vendor_name,
    frequency: entry.frequency,
    interval_value: entry.interval_value,
    next_run_date: normalizeWailsDateTime(entry.next_run_date) || "",
    last_generated_at: normalizeWailsDateTime(entry.last_generated_at),
    default_amount: entry.default_amount,
    default_vat_amount: entry.default_vat_amount,
    currency: entry.currency,
    cost_center: entry.cost_center,
    project_id: entry.project_id,
    is_active: entry.is_active,
    auto_submit: entry.auto_submit,
  };
}

function toBankExpenseCandidate(entry: finance.BankExpenseEntry): BankExpenseCandidate {
  return {
    id: entry.id,
    bank_statement_line_id: entry.bank_statement_line_id,
    division: entry.division,
    expense_date: normalizeWailsDateTime(entry.expense_date) || "",
    description: entry.description,
    category: entry.category,
    amount: entry.amount,
    currency: entry.currency,
    vat_amount: entry.vat_amount,
    is_posted: entry.is_posted,
  };
}

export interface ExpenseCategory {
  id: string;
  name: string;
  code: string;
  description?: string;
  gl_account_id?: string;
  gl_account_name?: string;
  default_tax_rate?: number;
  is_active?: boolean;
}

export interface ExpenseVendor {
  id: string;
  name: string;
  contact_name?: string;
  email?: string;
  phone?: string;
  payment_terms?: string;
  tax_number?: string;
  notes?: string;
  is_active?: boolean;
}

export interface ExpenseEntry {
  id: string;
  entry_number: string;
  division?: string;
  expense_date: string;
  due_date?: string;
  description: string;
  category_id: string;
  category_name?: string;
  vendor_id?: string;
  vendor_name?: string;
  source_type?: string;
  source_ref_id?: string;
  bank_expense_entry_id?: string;
  project_id?: string;
  customer_id?: string;
  opportunity_id?: string;
  order_id?: string;
  cost_center?: string;
  currency?: string;
  amount: number;
  vat_amount: number;
  total_amount: number;
  status: string;
  payment_status?: string;
  payment_method?: string;
  payment_reference?: string;
  bank_account_id?: string;
  notes?: string;
  approved_at?: string;
  approved_by?: string;
  posted_at?: string;
  paid_at?: string;
}

export interface RecurringExpense {
  id: string;
  name: string;
  division?: string;
  description?: string;
  category_id: string;
  category_name?: string;
  vendor_id?: string;
  vendor_name?: string;
  frequency?: string;
  interval_value?: number;
  next_run_date: string;
  last_generated_at?: string;
  default_amount: number;
  default_vat_amount: number;
  currency?: string;
  cost_center?: string;
  project_id?: string;
  is_active?: boolean;
  auto_submit?: boolean;
}

export interface BankExpenseCandidate {
  id: string;
  bank_statement_line_id: string;
  division?: string;
  expense_date: string;
  description: string;
  category: string;
  amount: number;
  currency?: string;
  vat_amount?: number;
  is_posted?: boolean;
}

export interface ExpenseDashboardSummary {
  total_drafts: number;
  total_submitted: number;
  total_approved_unpaid: number;
  total_recurring: number;
  month_to_date_spend: number;
  upcoming_commitments: number;
}

export async function listExpenseCategories(activeOnly = true): Promise<ExpenseCategory[]> {
  if (!isDesktop()) return [];
  return await ListExpenseCategories(activeOnly) as ExpenseCategory[];
}

export async function createExpenseCategory(category: Partial<ExpenseCategory>): Promise<ExpenseCategory> {
  return await CreateExpenseCategory(buildWailsInput(finance.ExpenseCategory, category as Record<string, any>)) as ExpenseCategory;
}

export async function deleteExpenseCategory(categoryID: string): Promise<void> {
  return await DeleteExpenseCategory(categoryID);
}

export async function listExpenseVendors(activeOnly = true): Promise<ExpenseVendor[]> {
  if (!isDesktop()) return [];
  return await ListExpenseVendors(activeOnly) as ExpenseVendor[];
}

export async function createExpenseVendor(vendor: Partial<ExpenseVendor>): Promise<ExpenseVendor> {
  return await CreateExpenseVendor(buildWailsInput(finance.ExpenseVendor, vendor as Record<string, any>)) as ExpenseVendor;
}

export async function deleteExpenseVendor(vendorID: string): Promise<void> {
  return await DeleteExpenseVendor(vendorID);
}

export async function listExpenseEntries(status = "", includePaid = true): Promise<ExpenseEntry[]> {
  if (!isDesktop()) return [];
  return (await ListExpenseEntries(status, includePaid)).map(toExpenseEntry);
}

export async function createExpenseEntry(entry: Partial<ExpenseEntry>): Promise<ExpenseEntry> {
  return toExpenseEntry(await CreateExpenseEntry(buildWailsInput(finance.ExpenseEntry, entry as Record<string, any>)));
}

export async function deleteExpenseEntry(entryID: string): Promise<void> {
  return await DeleteExpenseEntry(entryID);
}

export async function submitExpenseEntry(entryID: string): Promise<ExpenseEntry> {
  return toExpenseEntry(await SubmitExpenseEntry(entryID));
}

export async function approveExpenseEntry(entryID: string, notes = ""): Promise<ExpenseEntry> {
  return toExpenseEntry(await ApproveExpenseEntry(entryID, notes));
}

export async function rejectExpenseEntry(entryID: string, reason: string): Promise<ExpenseEntry> {
  return toExpenseEntry(await RejectExpenseEntry(entryID, reason));
}

export async function postExpenseEntry(entryID: string): Promise<ExpenseEntry> {
  return toExpenseEntry(await PostExpenseEntry(entryID));
}

export async function markExpenseEntryPaid(
  entryID: string,
  paidAtISO = "",
  paymentReference = "",
  bankAccountID = "",
  paymentMethod = "",
): Promise<ExpenseEntry> {
  return toExpenseEntry(await MarkExpenseEntryPaid(entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod));
}

export async function listRecurringExpenses(activeOnly = true): Promise<RecurringExpense[]> {
  if (!isDesktop()) return [];
  return (await ListRecurringExpenses(activeOnly)).map(toRecurringExpense);
}

export async function createRecurringExpense(recurring: Partial<RecurringExpense>): Promise<RecurringExpense> {
  return toRecurringExpense(await CreateRecurringExpense(buildWailsInput(finance.RecurringExpense, recurring as Record<string, any>)));
}

export async function deleteRecurringExpense(recurringID: string): Promise<void> {
  return await DeleteRecurringExpense(recurringID);
}

export async function generateRecurringExpenses(cutoffISO = ""): Promise<ExpenseEntry[]> {
  return (await GenerateRecurringExpenses(cutoffISO)).map(toExpenseEntry);
}

export async function listBankExpenseCandidates(includeLinked = false): Promise<BankExpenseCandidate[]> {
  if (!isDesktop()) return [];
  return (await ListBankExpenseCandidates(includeLinked)).map(toBankExpenseCandidate);
}

export async function createExpenseFromBankCandidate(bankExpenseID: string, categoryID = ""): Promise<ExpenseEntry> {
  return toExpenseEntry(await CreateExpenseFromBankCandidate(bankExpenseID, categoryID));
}

export async function getExpenseDashboardSummary(): Promise<ExpenseDashboardSummary> {
  if (!isDesktop()) {
    return {
      total_drafts: 0,
      total_submitted: 0,
      total_approved_unpaid: 0,
      total_recurring: 0,
      month_to_date_spend: 0,
      upcoming_commitments: 0,
    };
  }
  return await ListExpenseDashboardSummary() as ExpenseDashboardSummary;
}
