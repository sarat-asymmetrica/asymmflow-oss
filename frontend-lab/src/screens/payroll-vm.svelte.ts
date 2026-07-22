/* Payroll viewmodel — L5's reactive half: mode switching, division scoping,
 * the Compensation upsert form, the Period-create form, the Runs
 * list/select/lifecycle (approve→post→pay, gated behind confirms), and the
 * client-side summary/distribution math. No rendering/layout — Payroll.svelte
 * binds an instance of this and composes primitives only (L1). Named
 * `payroll-vm` (not `payroll.svelte.ts`) so its stem never collides with
 * `Payroll.svelte` case-insensitively on Windows (same rule as pricing-vm).
 */

import type { Tone } from '$kernel/tones'
import type { WidgetSegment } from '$kernel/hub'
import {
  approvePayrollRun,
  createPayrollPeriod,
  fetchActiveBankAccounts,
  fetchCompensationProfiles,
  fetchPayrollEmployeeOptions,
  fetchPayrollPayouts,
  fetchPayrollPeriods,
  fetchPayrollRun,
  fetchPayrollRuns,
  generatePayrollRun,
  generatePayslipPdf,
  markPayrollRunPaid,
  payrollDivisionOptions,
  postPayrollRun,
  upsertCompensationProfile,
  type CompensationProfile,
  type CompensationProfileDraft,
  type PayrollBankAccount,
  type PayrollEmployeeOption,
  type PayrollPayout,
  type PayrollPeriod,
  type PayrollPeriodDraft,
  type PayrollRun,
  type PayrollRunItem,
  type PayrollRunSummary,
} from '../bridge/payroll'

export type PayrollMode = 'compensation' | 'runs' | 'payouts'

/** Client-side division scoping — replicates the old screen's
 * `matchesCompany` exactly. `''` (the FilterChips "All" chip) matches every
 * row; a specific division does plain equality, so a legacy-cased division
 * string never matches a canonical option (see bridge/payroll.ts's third
 * DIVISIONS entry) — that's the K5 gap this build surfaces rather than
 * silently normalizing away. */
export function matchesDivision(rowDivision: string, selected: string): boolean {
  return selected === '' || rowDivision === selected
}

const todayIso = (): string => new Date().toISOString().slice(0, 10)

function blankProfileDraft(): CompensationProfileDraft {
  return {
    employeeId: '',
    division: '',
    payFrequency: 'monthly',
    baseSalary: 0,
    housingAllowance: 0,
    transportAllowance: 0,
    otherAllowance: 0,
    standardDeduction: 0,
    taxDeduction: 0,
    employerCost: 0,
    effectiveFrom: '',
    effectiveTo: '',
    isActive: true,
    notes: '',
  }
}

function blankPeriodDraft(division: string): PayrollPeriodDraft {
  const now = new Date()
  return {
    name: '',
    division,
    periodStart: new Date(now.getFullYear(), now.getMonth(), 1).toISOString().slice(0, 10),
    periodEnd: todayIso(),
    paymentDate: todayIso(),
    notes: '',
  }
}

/** Total compensation package (matches the old screen's `totalCompensation`
 * exactly) — one definition (L2) shared by the Compensation table column and
 * anywhere else that needs "what does this profile cost gross." */
export function profileGrossPay(p: CompensationProfile): number {
  return p.baseSalary + p.housingAllowance + p.transportAllowance + p.otherAllowance
}
export function profileDeductions(p: CompensationProfile): number {
  return p.standardDeduction + p.taxDeduction
}

/** A run item's live `employeeName` can be blank (the employee record has no
 * current name on file); falls back to the run-time snapshot name rather
 * than rendering a raw blank cell — see bridge/payroll.ts's run-1/item-1
 * adversarial row. */
export function runItemDisplayName(item: PayrollRunItem): string {
  return item.employeeName || item.employeeNameSnapshot || 'Unknown employee'
}

export interface PayrollSummary {
  activeProfiles: number
  openPeriods: number
  approvedOrPosted: number
  upcomingLiability: number
}

/** Client-side over the currently SCOPED (division-filtered) rows — the old
 * screen's `getPayrollDashboardSummary()` server fetch was dead code (its
 * result was immediately overwritten by an identical client `.reduce()`);
 * this build never calls the server summary at all, matching what the old
 * screen actually did in practice, not what it appeared to do. */
export function buildSummary(profiles: CompensationProfile[], periods: PayrollPeriod[], runs: PayrollRunSummary[]): PayrollSummary {
  return {
    activeProfiles: profiles.filter((p) => p.isActive).length,
    openPeriods: periods.filter((p) => p.status === 'open').length,
    approvedOrPosted: runs.filter((r) => r.status === 'approved' || r.status === 'posted').length,
    upcomingLiability: runs
      .filter((r) => r.status === 'approved' || r.status === 'posted')
      .reduce((sum, r) => sum + r.netTotal + r.deductionsTotal + r.employerCostTotal, 0),
  }
}

const RUN_STATUS_TONE: Record<string, Tone> = {
  draft: 'neutral',
  approved: 'warning',
  posted: 'info',
  paid: 'success',
}

/** Run counts by lifecycle state, for the always-on DistributionWidget.
 * Unrecognized statuses (the adversarial `unknown_status` run) bucket under
 * their own segment rather than being dropped or crashing. */
export function runStateDistribution(runs: PayrollRunSummary[]): WidgetSegment[] {
  const counts = new Map<string, number>()
  for (const r of runs) counts.set(r.status, (counts.get(r.status) ?? 0) + 1)
  return [...counts.entries()].map(([status, value]) => ({
    key: status,
    label: status,
    value,
    tone: RUN_STATUS_TONE[status] ?? 'neutral',
  }))
}

/** Field-masking is NET-NEW (not old-screen parity — the old screen had
 * all-or-nothing screen-level RBAC, no per-field masking). `canViewUnmasked`
 * defaults true for byte-parity with today's behavior; a future granular
 * permission flips it, and every PII surface (salary/allowance/deduction
 * amounts + employee names) already routes through these two helpers so
 * that flip needs no further plumbing. */
export function maskedAmount(value: number, unmasked: boolean): string {
  return unmasked ? String(value) : '••••••'
}
export function maskedName(name: string, unmasked: boolean): string {
  if (unmasked) return name
  return name ? '•••••' : ''
}

export class PayrollViewModel {
  mode = $state<PayrollMode>('compensation')
  divisionFilter = $state('')
  canViewUnmasked = $state(true)

  loading = $state(true)
  error = $state<string | null>(null)

  employees = $state<PayrollEmployeeOption[]>([])
  profiles = $state<CompensationProfile[]>([])
  periods = $state<PayrollPeriod[]>([])
  runs = $state<PayrollRunSummary[]>([])
  payouts = $state<PayrollPayout[]>([])
  bankAccounts = $state<PayrollBankAccount[]>([])

  selectedPeriodId = $state('')
  selectedRunId = $state('')
  selectedRun = $state<PayrollRun | null>(null)
  runDetailError = $state<string | null>(null)

  // ---- Compensation form ----
  editingProfileId = $state('')
  profileDraft = $state<CompensationProfileDraft>(blankProfileDraft())
  savingProfile = $state(false)
  profileError = $state<string | null>(null)

  // ---- Period-create form ----
  periodDraft = $state<PayrollPeriodDraft>(blankPeriodDraft(''))
  creatingPeriod = $state(false)
  periodError = $state<string | null>(null)

  // ---- Run lifecycle ----
  generatingRun = $state(false)
  generateError = $state<string | null>(null)
  runActionBusy = $state(false)
  runActionError = $state<string | null>(null)

  approveReason = $state('')
  approveConfirmOpen = $state(false)
  postConfirmOpen = $state(false)

  paymentReference = $state('')
  paidAt = $state(todayIso())
  bankAccountId = $state('')

  // ---- Payslip PDF (per-employee export, read-only — no state transition) ----
  payslipEmployeeId = $state('')
  payslipBusy = $state(false)
  payslipError = $state<string | null>(null)
  payslipPath = $state<string | null>(null)

  divisions = $derived(payrollDivisionOptions())

  scopedProfiles = $derived(this.profiles.filter((p) => matchesDivision(p.division, this.divisionFilter)))
  scopedPeriods = $derived(this.periods.filter((p) => matchesDivision(p.division, this.divisionFilter)))
  scopedRuns = $derived(this.runs.filter((r) => matchesDivision(r.division, this.divisionFilter)))
  scopedPayouts = $derived(this.payouts.filter((p) => matchesDivision(p.division, this.divisionFilter)))

  displayedRuns = $derived(
    this.selectedPeriodId ? this.scopedRuns.filter((r) => r.payrollPeriodId === this.selectedPeriodId) : this.scopedRuns,
  )
  displayedPayouts = $derived(
    this.selectedRunId ? this.scopedPayouts.filter((p) => p.payrollRunId === this.selectedRunId) : this.scopedPayouts,
  )

  summary = $derived(buildSummary(this.scopedProfiles, this.scopedPeriods, this.scopedRuns))
  runDistribution = $derived(runStateDistribution(this.scopedRuns))

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      const [employees, profiles, periods, runs, payouts, bankAccounts] = await Promise.all([
        fetchPayrollEmployeeOptions().catch(() => []), // INTEG-gapped for real; mock always resolves
        fetchCompensationProfiles(false),
        fetchPayrollPeriods(false),
        fetchPayrollRuns(''),
        fetchPayrollPayouts(''),
        fetchActiveBankAccounts(),
      ])
      this.employees = employees
      this.profiles = profiles
      this.periods = periods
      this.runs = runs
      this.payouts = payouts
      this.bankAccounts = bankAccounts
      if (!this.bankAccountId && bankAccounts.length) this.bankAccountId = bankAccounts[0]!.id
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  setMode(mode: PayrollMode): void {
    this.mode = mode
  }

  /** Deep-link entry (PeopleHub → "Set up payroll" for one employee): jump to
   * the Compensation view and prefill that employee's profile if one exists.
   * Called after load() when the screen is embedded with a presetEmployeeID. */
  presetEmployee(employeeId: string): void {
    if (!employeeId) return
    this.mode = 'compensation'
    const profile = this.profiles.find((p) => p.employeeId === employeeId)
    if (profile) this.editProfile(profile)
    else {
      this.resetProfileForm()
      this.profileDraft = { ...this.profileDraft, employeeId }
    }
  }

  // ---- Compensation ----
  resetProfileForm(): void {
    this.editingProfileId = ''
    this.profileDraft = blankProfileDraft()
    this.profileError = null
  }

  editProfile(profile: CompensationProfile): void {
    this.editingProfileId = profile.id
    this.profileDraft = {
      id: profile.id,
      employeeId: profile.employeeId,
      division: profile.division,
      payFrequency: profile.payFrequency,
      baseSalary: profile.baseSalary,
      housingAllowance: profile.housingAllowance,
      transportAllowance: profile.transportAllowance,
      otherAllowance: profile.otherAllowance,
      standardDeduction: profile.standardDeduction,
      taxDeduction: profile.taxDeduction,
      employerCost: profile.employerCost,
      effectiveFrom: profile.effectiveFrom,
      effectiveTo: profile.effectiveTo,
      isActive: profile.isActive,
      notes: profile.notes,
    }
    this.profileError = null
  }

  async saveProfile(): Promise<void> {
    if (!this.profileDraft.employeeId) {
      this.profileError = 'Choose an employee for the compensation profile.'
      return
    }
    if (!this.profileDraft.division) {
      const emp = this.employees.find((e) => e.id === this.profileDraft.employeeId)
      this.profileDraft.division = emp?.division ?? this.divisions[0]?.value ?? ''
    }
    this.savingProfile = true
    this.profileError = null
    try {
      await upsertCompensationProfile(this.profileDraft)
      this.resetProfileForm()
      await this.load()
    } catch (e) {
      this.profileError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingProfile = false
    }
  }

  // ---- Periods & runs ----
  selectPeriod(periodId: string): void {
    this.selectedPeriodId = periodId
  }

  /** Dangling-FK guard: a payout can reference a run id absent from `runs`
   * (adversarial monster row). Selecting it clears the detail instead of
   * throwing, and surfaces an inline error rather than crashing the panel. */
  async selectRun(runId: string): Promise<void> {
    this.selectedRunId = runId
    this.runDetailError = null
    this.payslipEmployeeId = ''
    this.payslipError = null
    this.payslipPath = null
    if (!this.runs.some((r) => r.id === runId)) {
      this.selectedRun = null
      this.runDetailError = `Payroll run ${runId} could not be found (dangling reference).`
      return
    }
    try {
      const run = await fetchPayrollRun(runId)
      this.selectedRun = run
      this.paymentReference = run.paymentReference || ''
      this.paidAt = run.paidAt || todayIso()
      this.bankAccountId = run.bankAccountId || this.bankAccounts[0]?.id || ''
      this.approveReason = ''
    } catch (e) {
      this.selectedRun = null
      this.runDetailError = e instanceof Error ? e.message : String(e)
    }
  }

  async createPeriod(): Promise<void> {
    if (!this.periodDraft.periodStart || !this.periodDraft.periodEnd) {
      this.periodError = 'Period start and end dates are required.'
      return
    }
    this.creatingPeriod = true
    this.periodError = null
    try {
      const draft = { ...this.periodDraft, division: this.periodDraft.division || this.divisions[0]?.value || '' }
      const created = await createPayrollPeriod(draft)
      this.selectedPeriodId = created.id
      this.periodDraft = blankPeriodDraft(draft.division)
      await this.load()
    } catch (e) {
      this.periodError = e instanceof Error ? e.message : String(e)
    } finally {
      this.creatingPeriod = false
    }
  }

  async generateRun(): Promise<void> {
    if (!this.selectedPeriodId) {
      this.generateError = 'Choose a payroll period first.'
      return
    }
    this.generatingRun = true
    this.generateError = null
    try {
      const run = await generatePayrollRun(this.selectedPeriodId)
      await this.load()
      await this.selectRun(run.id)
    } catch (e) {
      this.generateError = e instanceof Error ? e.message : String(e)
    } finally {
      this.generatingRun = false
    }
  }

  // ---- Run lifecycle: Approve (FIX — was a bare click, hardcoded note) ----
  requestApprove(): void {
    if (!this.selectedRun || this.selectedRun.status !== 'draft') return
    this.approveConfirmOpen = true
  }
  cancelApprove(): void {
    this.approveConfirmOpen = false
  }
  async confirmApprove(): Promise<void> {
    if (!this.selectedRun) return
    this.approveConfirmOpen = false
    this.runActionBusy = true
    this.runActionError = null
    try {
      await approvePayrollRun(this.selectedRun.id, this.approveReason.trim())
      await this.load()
      await this.selectRun(this.selectedRun.id)
    } catch (e) {
      this.runActionError = e instanceof Error ? e.message : String(e)
    } finally {
      this.runActionBusy = false
    }
  }

  // ---- Run lifecycle: Post (FIX — was a bare click, no confirm) ----
  requestPost(): void {
    if (!this.selectedRun || this.selectedRun.status !== 'approved') return
    this.postConfirmOpen = true
  }
  cancelPost(): void {
    this.postConfirmOpen = false
  }
  async confirmPost(): Promise<void> {
    if (!this.selectedRun) return
    this.postConfirmOpen = false
    this.runActionBusy = true
    this.runActionError = null
    try {
      await postPayrollRun(this.selectedRun.id)
      await this.load()
      await this.selectRun(this.selectedRun.id)
    } catch (e) {
      this.runActionError = e instanceof Error ? e.message : String(e)
    } finally {
      this.runActionBusy = false
    }
  }

  /** PRESERVE + FLAG: enabledFrom is ['approved','posted'] on purpose — the
   * old screen let Mark Paid fire from either state (post-before-pay isn't
   * strictly enforced). Kept as-is; see Payroll.parity.md "owner question". */
  async markPaid(): Promise<void> {
    if (!this.selectedRun) return
    if (!this.paymentReference.trim()) {
      this.runActionError = 'Add the salary payment reference before marking payroll as paid.'
      return
    }
    if (!this.bankAccountId) {
      this.runActionError = 'Choose the bank account used for the salary payment.'
      return
    }
    this.runActionBusy = true
    this.runActionError = null
    try {
      await markPayrollRunPaid(
        this.selectedRun.id,
        new Date(`${this.paidAt}T09:00:00`).toISOString(),
        this.paymentReference.trim(),
        this.bankAccountId,
      )
      await this.load()
      await this.selectRun(this.selectedRun.id)
    } catch (e) {
      this.runActionError = e instanceof Error ? e.message : String(e)
    } finally {
      this.runActionBusy = false
    }
  }

  /** Renders and saves a single employee's payslip PDF from the selected
   * run's already-committed data — a read-only export, not a run-lifecycle
   * mutation, so it has its own busy/error/result state rather than reusing
   * runActionBusy/runActionError. */
  async generatePayslip(employeeId: string): Promise<void> {
    if (!this.selectedRun || !employeeId) return
    this.payslipBusy = true
    this.payslipError = null
    this.payslipPath = null
    try {
      this.payslipPath = await generatePayslipPdf(employeeId, this.selectedRun.payrollPeriodId)
    } catch (e) {
      this.payslipError = e instanceof Error ? e.message : String(e)
    } finally {
      this.payslipBusy = false
    }
  }
}
