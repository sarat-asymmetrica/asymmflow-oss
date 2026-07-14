<script lang="ts">
  import { run as run_1 } from 'svelte/legacy';

  import { onDestroy, onMount } from "svelte";
  import { GetActiveBankAccounts } from "../../../wailsjs/go/main/App";
  import { EventsOff, EventsOn } from "../../../wailsjs/runtime/runtime";
  import { toast } from "../stores/toasts";
  import { formatBHD } from "$lib/utils/formatters";
  import { getDefaultDivisionKey, normalizeDivision } from "$lib/divisions.svelte";
  import { listEmployeeProfiles, type EmployeeProfile } from "$lib/api/collaboration";
  import {
    approvePayrollRun,
    createPayrollPeriod,
    generatePayrollRun,
    getPayrollDashboardSummary,
    getPayrollRun,
    listEmployeeCompensationProfiles,
    listPayrollPayouts,
    listPayrollPeriods,
    listPayrollRuns,
    markPayrollRunPaid,
    postPayrollRun,
    upsertEmployeeCompensationProfile,
    type EmployeeCompensationProfile,
    type PayrollDashboardSummary,
    type PayrollPayout,
    type PayrollPeriod,
    type PayrollRun,
  } from "$lib/api/payroll";

  interface Props {
    embedded?: boolean;
    mode?: "compensation" | "runs" | "payouts" | "workspace";
    company?: string;
    // Wave 9.4 B2: "Set up payroll" deep-link from an employee record. Placement/nav
    // only — preselects (or opens a fresh) compensation profile for this employee;
    // no change to the generate/approve/post/pay state machine below.
    presetEmployeeID?: string;
  }

  let { embedded = false, mode = "compensation", company = getDefaultDivisionKey(), presetEmployeeID = "" }: Props = $props();
  let appliedPresetEmployeeID = $state("");

  let loading = $state(true);
  let saving = $state(false);

  let employees: EmployeeProfile[] = $state([]);
  let bankAccounts: CompanyBankAccount[] = $state([]);
  let profiles: EmployeeCompensationProfile[] = $state([]);
  let periods: PayrollPeriod[] = $state([]);
  let runs: PayrollRun[] = $state([]);
  let payouts: PayrollPayout[] = $state([]);
  let selectedRun: PayrollRun | null = $state(null);
  let selectedRunID = $state("");
  let selectedPeriodID = $state("");
  let paymentReference = $state("");
  let payrollPaidAt = $state(new Date().toISOString().slice(0, 10));
  let payrollBankAccountID = $state("");

  let summary: PayrollDashboardSummary = $state({
    active_profiles: 0,
    open_periods: 0,
    draft_runs: 0,
    approved_unpaid_runs: 0,
    month_to_date_net_payroll: 0,
    upcoming_payroll_liability: 0,
  });

  let editingProfileID = $state("");
  let selectedEmployeeID = $state("");
  let payFrequency = $state("monthly");
  let baseSalary = $state("");
  let housingAllowance = $state("");
  let transportAllowance = $state("");
  let otherAllowance = $state("");
  let standardDeduction = $state("");
  let taxDeduction = $state("");
  let employerCost = $state("");
  let effectiveFrom = $state("");
  let effectiveTo = $state("");
  let profileNotes = $state("");
  let profileIsActive = $state(true);

  let periodName = $state("");
  let periodStart = $state(new Date(new Date().getFullYear(), new Date().getMonth(), 1).toISOString().slice(0, 10));
  let periodEnd = $state(new Date().toISOString().slice(0, 10));
  let paymentDate = $state(new Date().toISOString().slice(0, 10));
  let periodNotes = $state("");

  type CompanyBankAccount = {
    id: string;
    division?: string;
    bank_name: string;
    account_name: string;
    account_number: string;
    currency?: string;
  };

  function matchesCompany(division?: string) {
    return normalizeDivision(division || getDefaultDivisionKey()) === normalizeDivision(company);
  }

  function buildPayrollSummary(
    scopedProfiles: EmployeeCompensationProfile[],
    scopedPeriods: PayrollPeriod[],
    scopedRuns: PayrollRun[],
  ): PayrollDashboardSummary {
    const currentMonth = new Date().getMonth();
    const currentYear = new Date().getFullYear();

    return {
      active_profiles: scopedProfiles.filter((profile) => profile.is_active !== false).length,
      open_periods: scopedPeriods.filter((period) => (period.status || "open") === "open").length,
      draft_runs: scopedRuns.filter((run) => run.status === "draft").length,
      approved_unpaid_runs: scopedRuns.filter((run) => ["approved", "posted"].includes(run.status)).length,
      month_to_date_net_payroll: scopedRuns
        .filter((run) => {
          if (run.status !== "paid" || !run.paid_at) return false;
          const paidAt = new Date(run.paid_at);
          return paidAt.getMonth() === currentMonth && paidAt.getFullYear() === currentYear;
        })
        .reduce((sum, run) => sum + (Number(run.net_total) || 0), 0),
      upcoming_payroll_liability: scopedRuns
        .filter((run) => ["approved", "posted"].includes(run.status))
        .reduce((sum, run) => sum + (Number(run.net_total) || 0) + (Number(run.deductions_total) || 0) + (Number(run.employer_cost_total) || 0), 0),
    };
  }

  function toISODate(value: string) {
    return value ? `${value}T00:00:00Z` : undefined;
  }

  function currency(value: number) {
    return formatBHD(value || 0);
  }

  function totalCompensation(profile: EmployeeCompensationProfile) {
    return (profile.base_salary || 0)
      + (profile.housing_allowance || 0)
      + (profile.transport_allowance || 0)
      + (profile.other_allowance || 0);
  }

  async function load() {
    loading = true;
    try {
      const [employeeRows, profileRows, periodRows, runRows, payoutRows, summaryRow, bankAccountRows] = await Promise.all([
        listEmployeeProfiles(true),
        listEmployeeCompensationProfiles(true),
        listPayrollPeriods(true),
        listPayrollRuns(""),
        listPayrollPayouts(""),
        getPayrollDashboardSummary(),
        GetActiveBankAccounts(),
      ]);

      employees = employeeRows;
      profiles = (profileRows || []).filter((profile) => matchesCompany(profile.division));
      periods = (periodRows || []).filter((period) => matchesCompany(period.division));
      runs = (runRows || []).filter((run) => matchesCompany(run.division));
      payouts = (payoutRows || []).filter((payout) => matchesCompany(payout.division));
      summary = buildPayrollSummary(profiles, periods, runs);
      bankAccounts = ((bankAccountRows || []) as CompanyBankAccount[]).filter((account) => matchesCompany(account.division));

      if (!selectedPeriodID && periods.length > 0) {
        selectedPeriodID = periods[0].id;
      }
      if (!selectedRunID && runs.length > 0) {
        selectedRunID = runs[0].id;
      }
      if (selectedRunID) {
        try {
          selectedRun = await getPayrollRun(selectedRunID);
          if (!matchesCompany(selectedRun.division)) {
            selectedRun = null;
            selectedRunID = "";
          } else {
            paymentReference = selectedRun.payment_reference || "";
            payrollPaidAt = selectedRun.paid_at?.slice(0, 10) || payrollPaidAt;
            payrollBankAccountID = selectedRun.bank_account_id || payrollBankAccountID;
          }
        } catch {
          selectedRun = null;
        }
      }
      if (!payrollBankAccountID && bankAccounts.length > 0) {
        payrollBankAccountID = bankAccounts[0].id;
      }
    } catch (err) {
      toast.danger(`Failed to load payroll workspace: ${String(err)}`);
    } finally {
      loading = false;
    }
  }

  function resetProfileForm() {
    editingProfileID = "";
    selectedEmployeeID = "";
    payFrequency = "monthly";
    baseSalary = "";
    housingAllowance = "";
    transportAllowance = "";
    otherAllowance = "";
    standardDeduction = "";
    taxDeduction = "";
    employerCost = "";
    effectiveFrom = "";
    effectiveTo = "";
    profileNotes = "";
    profileIsActive = true;
  }

  function editProfile(profile: EmployeeCompensationProfile) {
    editingProfileID = profile.id;
    selectedEmployeeID = profile.employee_id;
    payFrequency = profile.pay_frequency || "monthly";
    baseSalary = String(profile.base_salary || 0);
    housingAllowance = String(profile.housing_allowance || 0);
    transportAllowance = String(profile.transport_allowance || 0);
    otherAllowance = String(profile.other_allowance || 0);
    standardDeduction = String(profile.standard_deduction || 0);
    taxDeduction = String(profile.tax_deduction || 0);
    employerCost = String(profile.employer_cost || 0);
    effectiveFrom = profile.effective_from?.slice(0, 10) || "";
    effectiveTo = profile.effective_to?.slice(0, 10) || "";
    profileNotes = profile.notes || "";
    profileIsActive = profile.is_active ?? true;
  }

  async function handleSaveProfile() {
    if (!selectedEmployeeID) {
      toast.warning("Choose an employee for the compensation profile");
      return;
    }

    saving = true;
    try {
      await upsertEmployeeCompensationProfile({
        id: editingProfileID || undefined,
        employee_id: selectedEmployeeID,
        division: company,
        pay_frequency: payFrequency,
        base_salary: Number(baseSalary || 0),
        housing_allowance: Number(housingAllowance || 0),
        transport_allowance: Number(transportAllowance || 0),
        other_allowance: Number(otherAllowance || 0),
        standard_deduction: Number(standardDeduction || 0),
        tax_deduction: Number(taxDeduction || 0),
        employer_cost: Number(employerCost || 0),
        effective_from: toISODate(effectiveFrom),
        effective_to: toISODate(effectiveTo),
        notes: profileNotes.trim(),
        is_active: profileIsActive,
      });
      resetProfileForm();
      await load();
      toast.success("Compensation profile saved");
    } catch (err) {
      toast.danger(`Failed to save compensation profile: ${String(err)}`);
    } finally {
      saving = false;
    }
  }

  async function handleCreatePeriod() {
    if (!periodStart || !periodEnd) {
      toast.warning("Payroll period dates are required");
      return;
    }

    saving = true;
    try {
      const created = await createPayrollPeriod({
        name: periodName.trim() || undefined,
        division: company,
        period_start: toISODate(periodStart),
        period_end: toISODate(periodEnd),
        payment_date: toISODate(paymentDate),
        notes: periodNotes.trim(),
      });
      selectedPeriodID = created.id;
      periodName = "";
      periodNotes = "";
      await load();
      toast.success("Payroll period created");
    } catch (err) {
      toast.danger(`Failed to create payroll period: ${String(err)}`);
    } finally {
      saving = false;
    }
  }

  async function handleGenerateRun() {
    if (!selectedPeriodID) {
      toast.warning("Choose a payroll period first");
      return;
    }
    saving = true;
    try {
      const run = await generatePayrollRun(selectedPeriodID);
      selectedRunID = run.id;
      selectedRun = await getPayrollRun(run.id);
      if (!matchesCompany(selectedRun.division)) {
        throw new Error(`Generated payroll run belongs to ${selectedRun.division || getDefaultDivisionKey()}, not ${company}`);
      }
      await load();
      toast.success("Payroll run generated");
    } catch (err) {
      toast.danger(`Failed to generate payroll run: ${String(err)}`);
    } finally {
      saving = false;
    }
  }

  async function selectRun(runID: string) {
    selectedRunID = runID;
    try {
      selectedRun = await getPayrollRun(runID);
      if (!matchesCompany(selectedRun.division)) {
        selectedRun = null;
        selectedRunID = "";
        return;
      }
      paymentReference = selectedRun.payment_reference || "";
      payrollPaidAt = selectedRun.paid_at?.slice(0, 10) || new Date().toISOString().slice(0, 10);
      payrollBankAccountID = selectedRun.bank_account_id || bankAccounts[0]?.id || "";
    } catch (err) {
      toast.danger(`Failed to load payroll run: ${String(err)}`);
    }
  }

  async function handleRunAction(action: "approve" | "post" | "pay") {
    if (!selectedRunID) {
      toast.warning("Choose a payroll run first");
      return;
    }

    saving = true;
    try {
      if (action === "approve") {
        await approvePayrollRun(selectedRunID, "Approved from Finance Hub");
      } else if (action === "post") {
        await postPayrollRun(selectedRunID);
      } else {
        if (!paymentReference.trim()) {
          toast.warning("Add the salary payment reference before logging payroll as paid");
          return;
        }
        if (!payrollBankAccountID) {
          toast.warning("Choose the bank account used for salary payment");
          return;
        }
        await markPayrollRunPaid(
          selectedRunID,
          new Date(`${payrollPaidAt}T09:00:00`).toISOString(),
          paymentReference.trim(),
          payrollBankAccountID,
        );
      }
      selectedRun = await getPayrollRun(selectedRunID);
      await load();
      toast.success(`Payroll run ${action}d`);
    } catch (err) {
      toast.danger(`Failed to ${action} payroll run: ${String(err)}`);
    } finally {
      saving = false;
    }
  }

  onMount(() => {
    load();
    const unsubscribers = [
      EventsOn("payroll:updated", () => load()),
    ];
    return () => {
      for (const unsubscribe of unsubscribers) {
        try {
          unsubscribe();
        } catch {
          // ignore
        }
      }
    };
  });

  onDestroy(() => {
    EventsOff("payroll:updated");
  });

  let lastLoadedCompany = $state("");
  run_1(() => {
    if (company && company !== lastLoadedCompany) {
      lastLoadedCompany = company;
      selectedRun = null;
      selectedRunID = "";
      selectedPeriodID = "";
      load();
    }
  });

  // Deep-link handoff: once profiles have loaded, jump straight to this
  // employee's existing compensation profile (or a blank one scoped to them).
  run_1(() => {
    if (presetEmployeeID && presetEmployeeID !== appliedPresetEmployeeID && !loading) {
      appliedPresetEmployeeID = presetEmployeeID;
      const existingProfile = profiles.find((profile) => profile.employee_id === presetEmployeeID);
      if (existingProfile) {
        editProfile(existingProfile);
      } else {
        resetProfileForm();
        selectedEmployeeID = presetEmployeeID;
      }
    }
  });

  let selectedPeriod = $derived(periods.find((period) => period.id === selectedPeriodID) || null);
  let displayedRuns = $derived(selectedPeriodID
    ? runs.filter((run) => run.payroll_period_id === selectedPeriodID)
    : runs);
  let displayedPayouts = $derived(selectedRunID
    ? payouts.filter((payout) => payout.payroll_run_id === selectedRunID)
    : payouts);
  let showCompensation = $derived(mode === "compensation" || mode === "workspace");
  let showRuns = $derived(mode === "runs" || mode === "workspace");
  let showPayouts = $derived(mode === "payouts" || mode === "workspace");
</script>

{#if loading}
  <div class="state">Loading payroll workspace...</div>
{:else}
  <div class:page={!embedded} class="workspace">
    <section class="summary-grid">
      <article class="summary-card">
        <span>Active Profiles</span>
        <strong>{summary.active_profiles}</strong>
      </article>
      <article class="summary-card">
        <span>Open Periods</span>
        <strong>{summary.open_periods}</strong>
      </article>
      <article class="summary-card">
        <span>Approved / Posted</span>
        <strong>{summary.approved_unpaid_runs}</strong>
      </article>
      <article class="summary-card">
        <span>Upcoming Liability</span>
        <strong>{currency(summary.upcoming_payroll_liability)}</strong>
      </article>
    </section>

    {#if showCompensation}
      <section class="panel">
        <div class="panel-header">
          <div>
            <h2>Compensation Profiles</h2>
            <p>Configure base salary, allowances, deductions, and employer costs per employee.</p>
          </div>
          <button class="ghost" onclick={resetProfileForm}>New Profile</button>
        </div>

        <div class="info-strip">
          <div>
            <strong>Employer Cost</strong>
            <p>The extra company-side payroll burden beyond gross pay, such as visa, insurance, leave accruals, or other employer-funded obligations.</p>
          </div>
          <div>
            <strong>How Salary Payment Is Logged</strong>
            <p>Generate and post the payroll run first, then record the paid date, bank account, and transfer reference in the Payroll run card below.</p>
          </div>
        </div>

        <div class="form-grid compensation-grid primary-grid">
          <label>
            <span>Employee</span>
            <select bind:value={selectedEmployeeID}>
              <option value="">Select employee</option>
              {#each employees as employee}
                <option value={employee.id}>{employee.full_name}</option>
              {/each}
            </select>
          </label>
          <label>
            <span>Pay Frequency</span>
            <select bind:value={payFrequency}>
              <option value="monthly">Monthly</option>
              <option value="biweekly">Biweekly</option>
              <option value="weekly">Weekly</option>
            </select>
          </label>
          <label>
            <span>Base Salary</span>
            <input bind:value={baseSalary} type="number" min="0" step="0.001" />
          </label>
          <label>
            <span>Employer Cost</span>
            <input bind:value={employerCost} type="number" min="0" step="0.001" />
          </label>
        </div>

        <div class="form-grid compensation-grid">
          <label>
            <span>Housing Allowance</span>
            <input bind:value={housingAllowance} type="number" min="0" step="0.001" />
          </label>
          <label>
            <span>Transport Allowance</span>
            <input bind:value={transportAllowance} type="number" min="0" step="0.001" />
          </label>
          <label>
            <span>Other Allowance</span>
            <input bind:value={otherAllowance} type="number" min="0" step="0.001" />
          </label>
          <label>
            <span>Standard Deduction</span>
            <input bind:value={standardDeduction} type="number" min="0" step="0.001" />
          </label>
          <label>
            <span>Tax Deduction</span>
            <input bind:value={taxDeduction} type="number" min="0" step="0.001" />
          </label>
        </div>

        <div class="form-grid compensation-grid effective-grid">
          <label>
            <span>Effective From</span>
            <input bind:value={effectiveFrom} type="date" />
          </label>
          <label>
            <span>Effective To</span>
            <input bind:value={effectiveTo} type="date" />
          </label>
          <label class="checkbox">
            <input bind:checked={profileIsActive} type="checkbox" />
            <span>Active profile</span>
          </label>
        </div>

        <label class="full-width">
          <span>Notes</span>
          <textarea bind:value={profileNotes} rows="3" placeholder="Add compensation notes or assumptions"></textarea>
        </label>

        <div class="actions">
          <button class="primary" disabled={saving} onclick={handleSaveProfile}>
            {editingProfileID ? "Update Profile" : "Save Profile"}
          </button>
        </div>

        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Employee</th>
                <th>Gross</th>
                <th>Deductions</th>
                <th>Employer Cost</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {#each profiles as profile}
                <tr class:selected={editingProfileID === profile.id} onclick={() => editProfile(profile)}>
                  <td>
                    <strong>{profile.employee_name || "Unknown employee"}</strong>
                    <small>{profile.job_title || profile.pay_frequency || "monthly"}</small>
                    {#if profile.notes}
                      <div class="inline-note">{profile.notes}</div>
                    {/if}
                  </td>
                  <td>{currency(totalCompensation(profile))}</td>
                  <td>{currency((profile.standard_deduction || 0) + (profile.tax_deduction || 0))}</td>
                  <td>{currency(profile.employer_cost || 0)}</td>
                  <td>{profile.is_active ? "Active" : "Inactive"}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </section>
    {/if}

    {#if showRuns}
      <section class="panel">
        <div class="panel-header">
          <div>
            <h2>Payroll Periods & Runs</h2>
            <p>Open periods, generate runs from compensation profiles, then approve and post them.</p>
          </div>
        </div>

        <div class="form-grid">
          <label>
            <span>Period Name</span>
            <input bind:value={periodName} placeholder="Apr 2026 Payroll" />
          </label>
          <label>
            <span>Period Start</span>
            <input bind:value={periodStart} type="date" />
          </label>
          <label>
            <span>Period End</span>
            <input bind:value={periodEnd} type="date" />
          </label>
          <label>
            <span>Payment Date</span>
            <input bind:value={paymentDate} type="date" />
          </label>
        </div>
        <label class="full-width">
          <span>Notes</span>
          <textarea bind:value={periodNotes} rows="2" placeholder="Optional payroll period notes"></textarea>
        </label>
        <div class="actions">
          <button class="secondary" disabled={saving} onclick={handleCreatePeriod}>Create Period</button>
        </div>

        <div class="split">
          <article class="subpanel">
            <div class="subpanel-header">
              <h3>Periods</h3>
              <button class="primary" disabled={saving || !selectedPeriodID} onclick={handleGenerateRun}>Generate Run</button>
            </div>
            <div class="stack">
              {#each periods as period}
                <button class:selected={selectedPeriodID === period.id} class="list-item" onclick={() => selectedPeriodID = period.id}>
                  <strong>{period.name}</strong>
                  <span>{period.period_start?.slice(0, 10)} to {period.period_end?.slice(0, 10)}</span>
                  {#if period.notes}
                    <small class="list-note">{period.notes}</small>
                  {/if}
                </button>
              {/each}
            </div>
          </article>

          <article class="subpanel">
            <div class="subpanel-header">
              <h3>Runs</h3>
            </div>
            <div class="stack">
              {#each displayedRuns as run}
                <button class:selected={selectedRunID === run.id} class="list-item" onclick={() => selectRun(run.id)}>
                  <strong>{run.run_number}</strong>
                  <span>{run.status} · {currency(run.net_total)} net</span>
                </button>
              {/each}
            </div>
          </article>
        </div>

        {#if selectedRun}
          <article class="detail-card">
            <div class="panel-header">
              <div>
                <h3>{selectedRun.run_number}</h3>
                <p>{selectedRun.period_name || selectedPeriod?.name || "Payroll run"} · {selectedRun.status}</p>
              </div>
              <div class="actions compact">
                <button class="secondary" disabled={saving || selectedRun.status !== "draft"} onclick={() => handleRunAction("approve")}>Approve</button>
                <button class="secondary" disabled={saving || selectedRun.status !== "approved"} onclick={() => handleRunAction("post")}>Post</button>
                <button class="primary" disabled={saving || !["posted", "approved"].includes(selectedRun.status)} onclick={() => handleRunAction("pay")}>Mark Paid</button>
              </div>
            </div>

            <div class="stats-row">
              <div><span>Employees</span><strong>{selectedRun.total_employees}</strong></div>
              <div><span>Gross</span><strong>{currency(selectedRun.gross_total)}</strong></div>
              <div><span>Deductions</span><strong>{currency(selectedRun.deductions_total)}</strong></div>
              <div><span>Net</span><strong>{currency(selectedRun.net_total)}</strong></div>
              <div><span>Employer Cost</span><strong>{currency(selectedRun.employer_cost_total)}</strong></div>
            </div>

            {#if selectedRun.notes}
              <div class="note-card">
                <div class="note-card-head">
                  <span>Run Notes</span>
                  <span>Payroll Context</span>
                </div>
                <p>{selectedRun.notes}</p>
              </div>
            {/if}

            <div class="form-grid payment-grid">
              <label>
                <span>Paid Date</span>
                <input bind:value={payrollPaidAt} type="date" />
              </label>
              <label>
                <span>Bank Account</span>
                <select bind:value={payrollBankAccountID}>
                  <option value="">Select bank account</option>
                  {#each bankAccounts as account}
                    <option value={account.id}>
                      {account.bank_name} - {account.account_name} ({account.account_number})
                    </option>
                  {/each}
                </select>
              </label>
              <label class="payment-reference">
                <span>Payment Reference</span>
                <input bind:value={paymentReference} placeholder="Bank batch / transfer reference" />
              </label>
            </div>
            <p class="helper-copy">Use <strong>Mark Paid</strong> only after the salary transfer actually goes out. The paid date, bank account, and reference become the payroll payout trail.</p>

            <div class="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Employee</th>
                    <th>Gross</th>
                    <th>Deductions</th>
                    <th>Net</th>
                    <th>Payout</th>
                  </tr>
                </thead>
                <tbody>
                  {#each selectedRun.items || [] as item}
                    <tr>
                      <td>
                        <strong>{item.employee_name || item.employee_name_snapshot}</strong>
                        <small>{item.job_title_snapshot || ""}</small>
                        <div class="chips">
                          {#each item.components || [] as component}
                            <span class="chip {component.component_type}">{component.name}: {currency(component.amount)}</span>
                          {/each}
                        </div>
                      </td>
                      <td>{currency(item.gross_pay)}</td>
                      <td>{currency(item.deductions_total)}</td>
                      <td>{currency(item.net_pay)}</td>
                      <td>{item.payout_status || "scheduled"}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </article>
        {/if}
      </section>
    {/if}

    {#if showPayouts}
      <section class="panel">
        <div class="panel-header">
          <div>
            <h2>Payout Tracking</h2>
            <p>Track scheduled and paid payroll disbursements across runs. This is now part of Payroll instead of a separate finance tab.</p>
          </div>
        </div>

        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Run</th>
                <th>Employee</th>
                <th>Scheduled</th>
                <th>Paid</th>
                <th>Amount</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {#each displayedPayouts as payout}
                <tr class:selected={selectedRunID === payout.payroll_run_id} onclick={() => selectRun(payout.payroll_run_id)}>
                  <td>{payout.run_number || "Payroll run"}</td>
                  <td>{payout.employee_name || "Unknown employee"}</td>
                  <td>{payout.scheduled_at?.slice(0, 10) || "-"}</td>
                  <td>{payout.paid_at?.slice(0, 10) || "-"}</td>
                  <td>{currency(payout.amount)}</td>
                  <td>{payout.status || "scheduled"}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </section>
    {/if}
  </div>
{/if}

<style>
  .workspace {
    display: grid;
    gap: 20px;
  }

  .page {
    padding: 8px 0 24px;
  }

  .summary-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 14px;
  }

  .summary-card,
  .panel,
  .subpanel,
  .detail-card {
    background: #ffffff;
    border: 1px solid #e7e5df;
    border-radius: 18px;
    box-shadow: 0 12px 32px rgba(42, 49, 64, 0.06);
  }

  .summary-card {
    padding: 18px;
    display: grid;
    gap: 6px;
  }

  .summary-card span,
  .panel-header p,
  .list-item span,
  .list-note,
  td small,
  .stats-row span {
    color: #6d6a63;
    font-size: 12px;
  }

  .summary-card strong,
  .stats-row strong {
    font-size: 22px;
    color: #1f2a37;
  }

  .info-strip {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 12px;
    padding: 14px 16px;
    background: #f8f6f1;
    border: 1px solid #e7e5df;
    border-radius: 14px;
  }

  .info-strip strong {
    display: block;
    margin-bottom: 4px;
    color: #1f2a37;
  }

  .info-strip p,
  .helper-copy {
    color: #6d6a63;
    font-size: 12px;
    line-height: 1.55;
  }

  .panel,
  .detail-card {
    padding: 20px;
    display: grid;
    gap: 16px;
  }

  .panel-header,
  .subpanel-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 12px;
  }

  h2,
  h3,
  p {
    margin: 0;
  }

  .form-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 12px;
  }

  .compensation-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .primary-grid {
    grid-template-columns: minmax(0, 1.3fr) minmax(0, 0.9fr) repeat(2, minmax(0, 1fr));
  }

  .effective-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    align-items: end;
  }

  .payment-grid {
    grid-template-columns: 160px minmax(220px, 1fr) minmax(240px, 1.2fr);
    align-items: end;
  }

  label {
    display: grid;
    gap: 6px;
    color: #1f2a37;
    font-size: 13px;
    font-weight: 600;
  }

  .full-width {
    width: 100%;
  }

  input,
  select,
  textarea,
  button {
    font: inherit;
  }

  input,
  select,
  textarea {
    border: 1px solid #d8d3c9;
    border-radius: 12px;
    padding: 11px 12px;
    background: #fcfbf7;
    color: #1f2a37;
  }

  input,
  select {
    min-height: 42px;
  }

  .checkbox {
    display: flex;
    align-items: center;
    gap: 10px;
    padding-top: 28px;
    align-self: end;
  }

  .checkbox input {
    width: 18px;
    height: 18px;
  }

  .actions,
  .compact {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
  }

  button {
    border: none;
    border-radius: 999px;
    padding: 10px 16px;
    cursor: pointer;
    transition: transform 0.15s ease, opacity 0.15s ease;
  }

  button:hover {
    transform: translateY(-1px);
  }

  button:disabled {
    opacity: 0.55;
    cursor: not-allowed;
    transform: none;
  }

  .primary {
    background: #1f5eff;
    color: #ffffff;
  }

  .secondary {
    background: #eef2ff;
    color: #2743b8;
  }

  .ghost {
    background: #f4f1ea;
    color: #5f584d;
  }

  .split {
    display: grid;
    grid-template-columns: minmax(0, 280px) minmax(0, 1fr);
    gap: 16px;
  }

  .subpanel {
    padding: 16px;
    display: grid;
    gap: 12px;
  }

  .stack {
    display: grid;
    gap: 10px;
  }

  .list-item {
    padding: 12px;
    border-radius: 14px;
    text-align: left;
    background: #f8f5ef;
    color: #1f2a37;
    display: grid;
    gap: 4px;
  }

  .list-note {
    white-space: pre-wrap;
    line-height: 1.5;
  }

  .selected {
    outline: 2px solid #1f5eff;
    outline-offset: -2px;
  }

  .stats-row {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
    gap: 12px;
    padding: 14px;
    border-radius: 16px;
    background: #f8f5ef;
  }

  .stats-row > div {
    display: grid;
    gap: 4px;
  }

  .inline-note,
  .note-card p {
    color: #5f584d;
    font-size: 12px;
    line-height: 1.55;
    white-space: pre-wrap;
  }

  .inline-note {
    margin-top: 8px;
    padding: 8px 10px;
    border-radius: 10px;
    background: #f8f5ef;
  }

  .note-card {
    display: grid;
    gap: 8px;
    padding: 14px;
    border-radius: 16px;
    background: #f8f5ef;
    border: 1px solid #ece7db;
  }

  .note-card-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    color: #6d6a63;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .note-card p {
    margin: 0;
  }

  .table-wrap {
    overflow: auto;
    border: 1px solid #ece7db;
    border-radius: 16px;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    min-width: 720px;
  }

  th,
  td {
    padding: 14px 16px;
    border-bottom: 1px solid #f0ece3;
    text-align: left;
    vertical-align: top;
  }

  th {
    background: #faf7f1;
    font-size: 12px;
    color: #6d6a63;
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  td strong {
    display: block;
    color: #1f2a37;
  }

  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 8px;
  }

  .chip {
    padding: 4px 8px;
    border-radius: 999px;
    font-size: 11px;
    background: #ebe6da;
    color: #4f463b;
  }

  .chip.earning {
    background: #e6f6eb;
    color: #1b6a39;
  }

  .chip.deduction {
    background: #fff0d8;
    color: #9a5c00;
  }

  .chip.employer_cost {
    background: #eef2ff;
    color: #2743b8;
  }

  .state {
    padding: 40px;
    text-align: center;
    color: #6d6a63;
  }

  @media (max-width: 960px) {
    .split {
      grid-template-columns: 1fr;
    }

    .compensation-grid,
    .primary-grid,
    .effective-grid,
    .payment-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 720px) {
    .compensation-grid,
    .primary-grid,
    .effective-grid,
    .payment-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
