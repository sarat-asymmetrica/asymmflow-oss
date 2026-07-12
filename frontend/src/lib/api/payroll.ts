import { ApprovePayrollRun, CreatePayrollPeriod, GeneratePayrollRun, GetPayrollRun, ListEmployeeCompensationProfiles, ListPayrollDashboardSummary, ListPayrollPayouts, ListPayrollPeriods, ListPayrollRuns, MarkPayrollRunPaid, PostPayrollRun, UpsertEmployeeCompensationProfile } from "../../../wailsjs/go/main/FinanceService";
import { payroll } from "../../../wailsjs/go/models";
import { buildWailsInput, normalizeWailsDateTime } from "$lib/utils/wailsInterop";

const isDesktop = () => Boolean((window as any)?.go?.main?.App);

function toPayrollComponent(component: payroll.Component): PayrollComponent {
  return {
    id: component.id,
    payroll_run_item_id: component.payroll_run_item_id,
    component_type: component.component_type,
    code: component.code,
    name: component.name,
    amount: component.amount,
  };
}

function toPayrollRunItem(item: payroll.RunItem): PayrollRunItem {
  return {
    id: item.id,
    payroll_run_id: item.payroll_run_id,
    employee_id: item.employee_id,
    employee_name: item.employee_name,
    employee_name_snapshot: item.employee_name_snapshot,
    job_title_snapshot: item.job_title_snapshot,
    base_salary: item.base_salary,
    allowances_total: item.allowances_total,
    deductions_total: item.deductions_total,
    employer_cost_total: item.employer_cost_total,
    gross_pay: item.gross_pay,
    net_pay: item.net_pay,
    status: item.status,
    payout_id: item.payout_id,
    payout_status: item.payout_status,
    payout_paid_at: normalizeWailsDateTime(item.payout_paid_at),
    components: (item.components || []).map(toPayrollComponent),
  };
}

function toEmployeeCompensationProfile(profile: payroll.CompensationProfile): EmployeeCompensationProfile {
  return {
    id: profile.id,
    employee_id: profile.employee_id,
    division: profile.division,
    employee_name: profile.employee_name,
    job_title: profile.job_title,
    pay_frequency: profile.pay_frequency,
    currency: profile.currency,
    base_salary: profile.base_salary,
    housing_allowance: profile.housing_allowance,
    transport_allowance: profile.transport_allowance,
    other_allowance: profile.other_allowance,
    standard_deduction: profile.standard_deduction,
    tax_deduction: profile.tax_deduction,
    employer_cost: profile.employer_cost,
    effective_from: normalizeWailsDateTime(profile.effective_from),
    effective_to: normalizeWailsDateTime(profile.effective_to),
    is_active: profile.is_active,
    notes: profile.notes,
  };
}

function toPayrollPeriod(period: payroll.Period): PayrollPeriod {
  return {
    id: period.id,
    name: period.name,
    division: period.division,
    period_start: normalizeWailsDateTime(period.period_start) || "",
    period_end: normalizeWailsDateTime(period.period_end) || "",
    payment_date: normalizeWailsDateTime(period.payment_date),
    status: period.status,
    notes: period.notes,
  };
}

function toPayrollPayout(payout: payroll.Payout): PayrollPayout {
  return {
    id: payout.id,
    payroll_run_id: payout.payroll_run_id,
    payroll_run_item_id: payout.payroll_run_item_id,
    employee_id: payout.employee_id,
    division: payout.division,
    employee_name: payout.employee_name,
    run_number: payout.run_number,
    scheduled_at: normalizeWailsDateTime(payout.scheduled_at),
    paid_at: normalizeWailsDateTime(payout.paid_at),
    amount: payout.amount,
    currency: payout.currency,
    status: payout.status,
    payment_reference: payout.payment_reference,
    bank_account_id: payout.bank_account_id,
    bank_statement_line_id: payout.bank_statement_line_id,
  };
}

function toPayrollRun(run: payroll.Run): PayrollRun {
  return {
    id: run.id,
    run_number: run.run_number,
    payroll_period_id: run.payroll_period_id,
    division: run.division,
    period_name: run.period_name,
    status: run.status,
    generated_at: normalizeWailsDateTime(run.generated_at),
    approved_at: normalizeWailsDateTime(run.approved_at),
    posted_at: normalizeWailsDateTime(run.posted_at),
    paid_at: normalizeWailsDateTime(run.paid_at),
    payment_reference: run.payment_reference,
    bank_account_id: run.bank_account_id,
    journal_entry_id: run.journal_entry_id,
    payout_journal_entry_id: run.payout_journal_entry_id,
    total_employees: run.total_employees,
    gross_total: run.gross_total,
    deductions_total: run.deductions_total,
    net_total: run.net_total,
    employer_cost_total: run.employer_cost_total,
    currency: run.currency,
    notes: run.notes,
    items: (run.items || []).map(toPayrollRunItem),
    payouts: (run.payouts || []).map(toPayrollPayout),
  };
}

export interface EmployeeCompensationProfile {
  id: string;
  employee_id: string;
  division?: string;
  employee_name?: string;
  job_title?: string;
  pay_frequency?: string;
  currency?: string;
  base_salary: number;
  housing_allowance: number;
  transport_allowance: number;
  other_allowance: number;
  standard_deduction: number;
  tax_deduction: number;
  employer_cost: number;
  effective_from?: string;
  effective_to?: string;
  is_active?: boolean;
  notes?: string;
}

export interface PayrollPeriod {
  id: string;
  name: string;
  division?: string;
  period_start: string;
  period_end: string;
  payment_date?: string;
  status?: string;
  notes?: string;
}

export interface PayrollComponent {
  id: string;
  payroll_run_item_id: string;
  component_type: string;
  code: string;
  name: string;
  amount: number;
}

export interface PayrollRunItem {
  id: string;
  payroll_run_id: string;
  employee_id: string;
  employee_name?: string;
  employee_name_snapshot?: string;
  job_title_snapshot?: string;
  base_salary: number;
  allowances_total: number;
  deductions_total: number;
  employer_cost_total: number;
  gross_pay: number;
  net_pay: number;
  status?: string;
  payout_id?: string;
  payout_status?: string;
  payout_paid_at?: string;
  components?: PayrollComponent[];
}

export interface PayrollPayout {
  id: string;
  payroll_run_id: string;
  payroll_run_item_id: string;
  employee_id: string;
  division?: string;
  employee_name?: string;
  run_number?: string;
  scheduled_at?: string;
  paid_at?: string;
  amount: number;
  currency?: string;
  status?: string;
  payment_reference?: string;
  bank_account_id?: string;
  bank_statement_line_id?: string;
}

export interface PayrollRun {
  id: string;
  run_number: string;
  payroll_period_id: string;
  division?: string;
  period_name?: string;
  status: string;
  generated_at?: string;
  approved_at?: string;
  posted_at?: string;
  paid_at?: string;
  payment_reference?: string;
  bank_account_id?: string;
  journal_entry_id?: string;
  payout_journal_entry_id?: string;
  total_employees: number;
  gross_total: number;
  deductions_total: number;
  net_total: number;
  employer_cost_total: number;
  currency?: string;
  notes?: string;
  items?: PayrollRunItem[];
  payouts?: PayrollPayout[];
}

export interface PayrollDashboardSummary {
  active_profiles: number;
  open_periods: number;
  draft_runs: number;
  approved_unpaid_runs: number;
  month_to_date_net_payroll: number;
  upcoming_payroll_liability: number;
}

export async function listEmployeeCompensationProfiles(activeOnly = true): Promise<EmployeeCompensationProfile[]> {
  if (!isDesktop()) return [];
  return (await ListEmployeeCompensationProfiles(activeOnly)).map(toEmployeeCompensationProfile);
}

export async function upsertEmployeeCompensationProfile(
  profile: Partial<EmployeeCompensationProfile>,
): Promise<EmployeeCompensationProfile> {
  return toEmployeeCompensationProfile(
    await UpsertEmployeeCompensationProfile(buildWailsInput(payroll.CompensationProfile, profile as Record<string, any>)),
  );
}

export async function listPayrollPeriods(includeClosed = true): Promise<PayrollPeriod[]> {
  if (!isDesktop()) return [];
  return (await ListPayrollPeriods(includeClosed)).map(toPayrollPeriod);
}

export async function createPayrollPeriod(period: Partial<PayrollPeriod>): Promise<PayrollPeriod> {
  return toPayrollPeriod(await CreatePayrollPeriod(buildWailsInput(payroll.Period, period as Record<string, any>)));
}

export async function listPayrollRuns(payrollPeriodID = ""): Promise<PayrollRun[]> {
  if (!isDesktop()) return [];
  return (await ListPayrollRuns(payrollPeriodID)).map(toPayrollRun);
}

export async function getPayrollRun(runID: string): Promise<PayrollRun> {
  return toPayrollRun(await GetPayrollRun(runID));
}

export async function generatePayrollRun(payrollPeriodID: string): Promise<PayrollRun> {
  return toPayrollRun(await GeneratePayrollRun(payrollPeriodID));
}

export async function approvePayrollRun(runID: string, notes = ""): Promise<PayrollRun> {
  return toPayrollRun(await ApprovePayrollRun(runID, notes));
}

export async function postPayrollRun(runID: string): Promise<PayrollRun> {
  return toPayrollRun(await PostPayrollRun(runID));
}

export async function markPayrollRunPaid(
  runID: string,
  paidAtISO = "",
  paymentReference = "",
  bankAccountID = "",
): Promise<PayrollRun> {
  return toPayrollRun(await MarkPayrollRunPaid(runID, paidAtISO, paymentReference, bankAccountID));
}

export async function listPayrollPayouts(payrollRunID = ""): Promise<PayrollPayout[]> {
  if (!isDesktop()) return [];
  return (await ListPayrollPayouts(payrollRunID)).map(toPayrollPayout);
}

export async function getPayrollDashboardSummary(): Promise<PayrollDashboardSummary> {
  if (!isDesktop()) {
    return {
      active_profiles: 0,
      open_periods: 0,
      draft_runs: 0,
      approved_unpaid_runs: 0,
      month_to_date_net_payroll: 0,
      upcoming_payroll_liability: 0,
    };
  }
  return await ListPayrollDashboardSummary() as PayrollDashboardSummary;
}
