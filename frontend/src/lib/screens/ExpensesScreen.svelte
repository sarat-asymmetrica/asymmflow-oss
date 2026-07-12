<script lang="ts">
  import { run, stopPropagation, self } from 'svelte/legacy';

  import { onDestroy, onMount } from "svelte";
  import { GetActiveBankAccounts } from "../../../wailsjs/go/main/App";
  import { EventsOff, EventsOn } from "../../../wailsjs/runtime/runtime";
  import { toast } from "../stores/toasts";
  import { formatBHD } from "$lib/utils/formatters";
  import FormGroup from "$lib/components/ui/FormGroup.svelte";
  import {
    approveExpenseEntry,
    createExpenseCategory,
    createExpenseEntry,
    createExpenseFromBankCandidate,
    createExpenseVendor,
    deleteExpenseCategory,
    deleteExpenseEntry,
    deleteExpenseVendor,
    createRecurringExpense,
    deleteRecurringExpense,
    generateRecurringExpenses,
    getExpenseDashboardSummary,
    listBankExpenseCandidates,
    listExpenseCategories,
    listExpenseEntries,
    listExpenseVendors,
    listRecurringExpenses,
    markExpenseEntryPaid,
    postExpenseEntry,
    rejectExpenseEntry,
    submitExpenseEntry,
    type BankExpenseCandidate,
    type ExpenseCategory,
    type ExpenseDashboardSummary,
    type ExpenseEntry,
    type ExpenseVendor,
    type RecurringExpense,
  } from "$lib/api/expenses";

  interface Props {
    embedded?: boolean;
    mode?: "entries" | "recurring" | "approvals" | "workspace";
    company?: "Acme Instrumentation" | "Beacon Controls";
  }

  let { embedded = false, mode = "entries", company = "Acme Instrumentation" }: Props = $props();

  let loading = $state(true);
  let saving = $state(false);
  let deletingCategoryID = $state("");
  let deletingVendorID = $state("");
  let deletingEntryID = $state("");
  let deletingRecurringID = $state("");
  let confirmDeleteCategoryID = $state("");
  let confirmDeleteVendorID = $state("");
  let confirmDeleteEntryID = $state("");
  let confirmDeleteRecurringID = $state("");

  let categories: ExpenseCategory[] = $state([]);
  let vendors: ExpenseVendor[] = $state([]);
  let entries: ExpenseEntry[] = $state([]);
  let recurringExpenses: RecurringExpense[] = $state([]);
  let bankCandidates: BankExpenseCandidate[] = $state([]);
  let bankAccounts: CompanyBankAccount[] = $state([]);
  let summary: ExpenseDashboardSummary = $state({
    total_drafts: 0,
    total_submitted: 0,
    total_approved_unpaid: 0,
    total_recurring: 0,
    month_to_date_spend: 0,
    upcoming_commitments: 0,
  });

  let categoryName = $state("");
  let vendorName = $state("");

  let description = $state("");
  let categoryID = $state("");
  let vendorID = $state("");
  let amount = $state("");
  let vatAmount = $state("");
  let expenseDate = $state(new Date().toISOString().slice(0, 10));
  let dueDate = $state(new Date().toISOString().slice(0, 10));
  let costCenter = $state("");
  let notes = $state("");

  let recurringName = $state("");
  let recurringCategoryID = $state("");
  let recurringVendorID = $state("");
  let recurringAmount = $state("");
  let recurringVATAmount = $state("");
  let recurringFrequency = $state("monthly");
  let recurringNextRunDate = $state(new Date().toISOString().slice(0, 10));
  let recurringAutoSubmit = $state(true);
  let showPaymentModal = $state(false);
  let paymentEntry: ExpenseEntry | null = $state(null);
  let payingExpense = $state(false);
  let expensePaymentForm = $state(createPaymentForm());

  type CompanyBankAccount = {
    id: string;
    division?: string;
    bank_name: string;
    account_name: string;
    account_number: string;
    currency?: string;
  };

  function matchesCompany(division?: string) {
    return (division || "Acme Instrumentation") === company;
  }

  function buildExpenseSummary(scopedEntries: ExpenseEntry[], scopedRecurring: RecurringExpense[]): ExpenseDashboardSummary {
    const now = new Date();
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);

    return {
      total_drafts: scopedEntries.filter((entry) => entry.status === "draft").length,
      total_submitted: scopedEntries.filter((entry) => entry.status === "submitted").length,
      total_approved_unpaid: scopedEntries.filter((entry) => ["approved", "posted"].includes(entry.status) && entry.payment_status !== "paid").length,
      total_recurring: scopedRecurring.filter((entry) => entry.is_active !== false).length,
      month_to_date_spend: scopedEntries
        .filter((entry) => {
          const expenseDate = entry.expense_date ? new Date(entry.expense_date) : null;
          return expenseDate && expenseDate >= monthStart && !["draft", "rejected"].includes(entry.status);
        })
        .reduce((sum, entry) => sum + (Number(entry.total_amount) || 0), 0),
      upcoming_commitments: scopedEntries
        .filter((entry) => ["approved", "posted"].includes(entry.status) && entry.payment_status !== "paid")
        .reduce((sum, entry) => sum + (Number(entry.total_amount) || 0), 0),
    };
  }

  function createPaymentForm() {
    return {
      paidAt: new Date().toISOString().slice(0, 10),
      paymentMethod: "NEFT",
      paymentReference: "",
      bankAccountID: "",
    };
  }

  const expensePaymentMethods = ["NEFT", "Bank Transfer", "Cheque", "Cash", "Card", "Wire Transfer"];

  function toISODate(value: string) {
    return value ? `${value}T00:00:00Z` : undefined;
  }

  function currency(value: number) {
    return formatBHD(value || 0);
  }

  async function load() {
    loading = true;
    try {
      const [categoryRows, vendorRows, summaryRow, entryRows, recurringRows, bankRows, bankAccountRows] = await Promise.all([
        listExpenseCategories(true),
        listExpenseVendors(true),
        getExpenseDashboardSummary(),
        listExpenseEntries("", true),
        listRecurringExpenses(true),
        listBankExpenseCandidates(false),
        GetActiveBankAccounts(),
      ]);
      categories = categoryRows;
      vendors = vendorRows;
      entries = (entryRows || []).filter((entry) => matchesCompany(entry.division));
      recurringExpenses = (recurringRows || []).filter((entry) => matchesCompany(entry.division));
      bankCandidates = (bankRows || []).filter((entry) => matchesCompany(entry.division));
      bankAccounts = ((bankAccountRows || []) as CompanyBankAccount[]).filter((account) => matchesCompany(account.division));
      summary = buildExpenseSummary(entries, recurringExpenses);

      if (!categoryID && categories.length > 0) {
        categoryID = categories[0].id;
      }
      if (!recurringCategoryID && categories.length > 0) {
        recurringCategoryID = categories[0].id;
      }
    } catch (err) {
      toast.danger(`Failed to load expenses: ${String(err)}`);
    } finally {
      loading = false;
    }
  }

  async function handleCreateCategory() {
    if (!categoryName.trim()) {
      toast.warning("Category name is required");
      return;
    }
    try {
      const created = await createExpenseCategory({ name: categoryName.trim() });
      categoryName = "";
      categoryID = created.id;
      recurringCategoryID = recurringCategoryID || created.id;
      await load();
      toast.success("Expense category created");
    } catch (err) {
      toast.danger(`Failed to create category: ${String(err)}`);
    }
  }

  async function handleDeleteCategory(category: ExpenseCategory) {
    if (!category?.id) {
      return;
    }

    if (confirmDeleteCategoryID !== category.id) {
      confirmDeleteCategoryID = category.id;
      confirmDeleteVendorID = "";
      toast.warning(`Press delete again to remove category "${category.name}". This only works if nothing is using it.`);
      return;
    }

    deletingCategoryID = category.id;
    try {
      await deleteExpenseCategory(category.id);
      if (categoryID === category.id) {
        categoryID = "";
      }
      if (recurringCategoryID === category.id) {
        recurringCategoryID = "";
      }
      await load();
      toast.success("Expense category deleted");
    } catch (err) {
      toast.danger(`Failed to delete category: ${String(err)}`);
    } finally {
      confirmDeleteCategoryID = "";
      deletingCategoryID = "";
    }
  }

  async function handleCreateVendor() {
    if (!vendorName.trim()) {
      toast.warning("Vendor name is required");
      return;
    }
    try {
      const created = await createExpenseVendor({ name: vendorName.trim() });
      vendorName = "";
      vendorID = created.id;
      recurringVendorID = recurringVendorID || created.id;
      await load();
      toast.success("Expense vendor created");
    } catch (err) {
      toast.danger(`Failed to create vendor: ${String(err)}`);
    }
  }

  async function handleDeleteVendor(vendor: ExpenseVendor) {
    if (!vendor?.id) {
      return;
    }

    if (confirmDeleteVendorID !== vendor.id) {
      confirmDeleteVendorID = vendor.id;
      confirmDeleteCategoryID = "";
      toast.warning(`Press delete again to remove vendor "${vendor.name}". This only works if nothing is using it.`);
      return;
    }

    deletingVendorID = vendor.id;
    try {
      await deleteExpenseVendor(vendor.id);
      if (vendorID === vendor.id) {
        vendorID = "";
      }
      if (recurringVendorID === vendor.id) {
        recurringVendorID = "";
      }
      await load();
      toast.success("Expense vendor deleted");
    } catch (err) {
      toast.danger(`Failed to delete vendor: ${String(err)}`);
    } finally {
      confirmDeleteVendorID = "";
      deletingVendorID = "";
    }
  }

  async function handleCreateExpense() {
    if (!description.trim()) {
      toast.warning("Expense description is required");
      return;
    }
    if (!categoryID) {
      toast.warning("Choose an expense category");
      return;
    }

    saving = true;
    try {
      await createExpenseEntry({
        description: description.trim(),
        category_id: categoryID,
        division: company,
        vendor_id: vendorID || undefined,
        amount: Number(amount || 0),
        vat_amount: Number(vatAmount || 0),
        expense_date: toISODate(expenseDate),
        due_date: toISODate(dueDate),
        cost_center: costCenter.trim(),
        notes: notes.trim(),
      });
      description = "";
      amount = "";
      vatAmount = "";
      costCenter = "";
      notes = "";
      vendorID = "";
      await load();
      toast.success("Expense draft created");
    } catch (err) {
      toast.danger(`Failed to create expense: ${String(err)}`);
    } finally {
      saving = false;
    }
  }

  function canDeleteEntry(entry: ExpenseEntry) {
    return entry.status !== "posted" && entry.status !== "paid" && entry.payment_status !== "paid";
  }

  async function handleDeleteEntry(entry: ExpenseEntry) {
    if (!entry?.id) {
      return;
    }

    if (!canDeleteEntry(entry)) {
      toast.warning("Posted or paid expenses are protected and cannot be deleted");
      return;
    }

    if (confirmDeleteEntryID !== entry.id) {
      confirmDeleteEntryID = entry.id;
      toast.warning(`Press delete again to remove expense "${entry.entry_number}".`);
      return;
    }

    deletingEntryID = entry.id;
    try {
      await deleteExpenseEntry(entry.id);
      await load();
      toast.success("Expense entry deleted");
    } catch (err) {
      toast.danger(`Failed to delete expense: ${String(err)}`);
    } finally {
      confirmDeleteEntryID = "";
      deletingEntryID = "";
    }
  }

  async function handleCreateRecurring() {
    if (!recurringName.trim()) {
      toast.warning("Recurring expense name is required");
      return;
    }
    if (!recurringCategoryID) {
      toast.warning("Choose a recurring expense category");
      return;
    }

    try {
      await createRecurringExpense({
        name: recurringName.trim(),
        category_id: recurringCategoryID,
        division: company,
        vendor_id: recurringVendorID || undefined,
        frequency: recurringFrequency,
        next_run_date: toISODate(recurringNextRunDate),
        default_amount: Number(recurringAmount || 0),
        default_vat_amount: Number(recurringVATAmount || 0),
        auto_submit: recurringAutoSubmit,
      });
      recurringName = "";
      recurringAmount = "";
      recurringVATAmount = "";
      recurringVendorID = "";
      await load();
      toast.success("Recurring expense created");
    } catch (err) {
      toast.danger(`Failed to create recurring expense: ${String(err)}`);
    }
  }

  async function handleGenerateRecurring() {
    try {
      const generated = await generateRecurringExpenses(toISODate(recurringNextRunDate) || "");
      await load();
      toast.success(`${generated.length} recurring expense entries generated`);
    } catch (err) {
      toast.danger(`Failed to generate recurring expenses: ${String(err)}`);
    }
  }

  async function handleDeleteRecurring(item: RecurringExpense) {
    if (!item?.id) {
      return;
    }

    if (confirmDeleteRecurringID !== item.id) {
      confirmDeleteRecurringID = item.id;
      toast.warning(`Press delete again to remove recurring schedule "${item.name}". Existing generated expenses will stay in the ledger.`);
      return;
    }

    deletingRecurringID = item.id;
    try {
      await deleteRecurringExpense(item.id);
      await load();
      toast.success("Recurring expense deleted");
    } catch (err) {
      toast.danger(`Failed to delete recurring expense: ${String(err)}`);
    } finally {
      confirmDeleteRecurringID = "";
      deletingRecurringID = "";
    }
  }

  async function runEntryAction(action: string, entryID: string) {
    try {
      if (action === "submit") {
        await submitExpenseEntry(entryID);
      } else if (action === "approve") {
        await approveExpenseEntry(entryID, "");
      } else if (action === "reject") {
        await rejectExpenseEntry(entryID, "Rejected from approvals queue");
      } else if (action === "post") {
        await postExpenseEntry(entryID);
      }
      await load();
      toast.success(`Expense ${action} complete`);
    } catch (err) {
      toast.danger(`Failed to ${action} expense: ${String(err)}`);
    }
  }

  async function importBankCandidate(candidate: BankExpenseCandidate) {
    try {
      await createExpenseFromBankCandidate(candidate.id, categoryID);
      await load();
      toast.success("Bank expense candidate imported");
    } catch (err) {
      toast.danger(`Failed to import bank expense candidate: ${String(err)}`);
    }
  }

  function openPaymentModal(entry: ExpenseEntry) {
    paymentEntry = entry;
    expensePaymentForm = {
      paidAt: new Date().toISOString().slice(0, 10),
      paymentMethod: entry.payment_method || "NEFT",
      paymentReference: entry.payment_reference || "",
      bankAccountID: entry.bank_account_id || bankAccounts[0]?.id || "",
    };
    showPaymentModal = true;
  }

  async function submitExpensePayment() {
    if (!paymentEntry) return;
    if (!expensePaymentForm.paidAt) {
      toast.warning("Payment date is required");
      return;
    }
    if (!expensePaymentForm.paymentMethod) {
      toast.warning("Payment method is required");
      return;
    }
    if (!expensePaymentForm.bankAccountID) {
      toast.warning("Choose the bank account used for this payment");
      return;
    }
    if (!expensePaymentForm.paymentReference.trim()) {
      toast.warning("Payment reference is required");
      return;
    }

    payingExpense = true;
    try {
      await markExpenseEntryPaid(
        paymentEntry.id,
        new Date(`${expensePaymentForm.paidAt}T09:00:00`).toISOString(),
        expensePaymentForm.paymentReference.trim(),
        expensePaymentForm.bankAccountID,
        expensePaymentForm.paymentMethod,
      );
      showPaymentModal = false;
      paymentEntry = null;
      expensePaymentForm = createPaymentForm();
      await load();
      toast.success("Expense marked paid");
    } catch (err) {
      toast.danger(`Failed to record expense payment: ${String(err)}`);
    } finally {
      payingExpense = false;
    }
  }

  let submittedEntries = $derived(entries.filter((entry) => entry.status === "submitted"));
  let approvalQueue = $derived(entries.filter((entry) => ["submitted", "approved", "posted"].includes(entry.status) && entry.payment_status !== "paid"));
  let sortedEntries = $derived([...entries].sort((left, right) => (right.expense_date || "").localeCompare(left.expense_date || "")));
  let showExpenseEntries = $derived(mode === "entries" || mode === "workspace");
  let showRecurring = $derived(mode === "recurring" || mode === "workspace");

  onMount(() => {
    load();
    EventsOn("expenses:updated", load);
  });

  onDestroy(() => {
    EventsOff("expenses:updated");
  });

  let lastLoadedCompany = $state("");
  run(() => {
    if (company && company !== lastLoadedCompany) {
      lastLoadedCompany = company;
      load();
    }
  });
</script>

<div class:embedded class="page">
  <section class="summary-grid">
    <article class="summary-card">
      <span>Month To Date Spend</span>
      <strong>{currency(summary.month_to_date_spend)}</strong>
    </article>
    <article class="summary-card">
      <span>Upcoming Commitments</span>
      <strong>{currency(summary.upcoming_commitments)}</strong>
    </article>
    <article class="summary-card">
      <span>Submitted / Approved</span>
      <strong>{summary.total_submitted} / {summary.total_approved_unpaid}</strong>
    </article>
    <article class="summary-card">
      <span>Recurring Schedules</span>
      <strong>{summary.total_recurring}</strong>
    </article>
  </section>

  {#if showExpenseEntries}
    <section class="layout">
	      <article class="panel">
        <div class="panel-head">
          <h2>Quick Entry</h2>
          <span>Draft-first expense capture</span>
        </div>

        <div class="entry-form">
          <fieldset class="field-group">
            <legend>Classification</legend>
            <div class="field-grid">
              <FormGroup label="Description">
                <input bind:value={description} placeholder="What was this expense for?" />
              </FormGroup>
              <FormGroup label="Category">
                <select bind:value={categoryID}>
                  <option value="">Select category</option>
                  {#each categories as category}
                    <option value={category.id}>{category.name}</option>
                  {/each}
                </select>
              </FormGroup>
              <FormGroup label="Vendor">
                <select bind:value={vendorID}>
                  <option value="">No vendor</option>
                  {#each vendors as vendor}
                    <option value={vendor.id}>{vendor.name}</option>
                  {/each}
                </select>
              </FormGroup>
              <FormGroup label="Cost Center / Memo">
                <input bind:value={costCenter} placeholder="Cost center or memo" />
              </FormGroup>
            </div>
          </fieldset>

          <fieldset class="field-group">
            <legend>Money &amp; Dates</legend>
            <div class="field-grid">
              <FormGroup label="Amount">
                <input bind:value={amount} min="0" step="0.001" type="number" placeholder="0.000" />
              </FormGroup>
              <FormGroup label="VAT Amount">
                <input bind:value={vatAmount} min="0" step="0.001" type="number" placeholder="0.000" />
              </FormGroup>
              <FormGroup label="Expense Date">
                <input bind:value={expenseDate} type="date" />
              </FormGroup>
              <FormGroup label="Due Date">
                <input bind:value={dueDate} type="date" />
              </FormGroup>
            </div>
          </fieldset>
        </div>

        <textarea bind:value={notes} placeholder="Notes"></textarea>
        <div class="actions">
          <button disabled={saving} onclick={handleCreateExpense}>Create Draft</button>
        </div>

        <details class="setup-disclosure">
          <summary>Setup: Categories &amp; Vendors</summary>

          <div class="mini-tools">
            <div class="tool-row">
              <input bind:value={categoryName} placeholder="New category" />
              <button class="secondary" onclick={handleCreateCategory}>Add Category</button>
            </div>
            <div class="tool-row">
              <input bind:value={vendorName} placeholder="New vendor" />
              <button class="secondary" onclick={handleCreateVendor}>Add Vendor</button>
            </div>
          </div>

          <div class="category-manager">
            <div class="category-manager-head">
              <strong>Manage Categories</strong>
              <span>{categories.length} active</span>
            </div>
            {#if categories.length === 0}
              <div class="empty compact">No expense categories available.</div>
            {:else}
              <div class="category-list">
                {#each categories as category}
                  <div class="category-chip" class:selected={category.id === categoryID || category.id === recurringCategoryID}>
                    <div class="category-copy">
                      <strong>{category.name}</strong>
                      <span>{category.code || "No code"}</span>
                    </div>
                    <button
                      type="button"
                      class="danger ghost"
                      class:confirming={confirmDeleteCategoryID === category.id}
                      disabled={deletingCategoryID === category.id}
                      onclick={stopPropagation(() => handleDeleteCategory(category))}
                    >
                      {deletingCategoryID === category.id ? "Deleting..." : confirmDeleteCategoryID === category.id ? "Confirm Delete" : "Delete"}
                    </button>
                  </div>
                {/each}
              </div>
            {/if}
          </div>

          <div class="category-manager">
            <div class="category-manager-head">
              <strong>Manage Vendors</strong>
              <span>{vendors.length} active</span>
            </div>
            {#if vendors.length === 0}
              <div class="empty compact">No expense vendors available.</div>
            {:else}
              <div class="category-list">
                {#each vendors as vendor}
                  <div class="category-chip" class:selected={vendor.id === vendorID || vendor.id === recurringVendorID}>
                    <div class="category-copy">
                      <strong>{vendor.name}</strong>
                      <span>{vendor.payment_terms || vendor.email || vendor.phone || "No extra details"}</span>
                    </div>
                    <button
                      type="button"
                      class="danger ghost"
                      class:confirming={confirmDeleteVendorID === vendor.id}
                      disabled={deletingVendorID === vendor.id}
                      onclick={stopPropagation(() => handleDeleteVendor(vendor))}
                    >
                      {deletingVendorID === vendor.id ? "Deleting..." : confirmDeleteVendorID === vendor.id ? "Confirm Delete" : "Delete"}
                    </button>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </details>
      </article>

      <article class="panel">
        <div class="panel-head">
          <h2>Bank Candidates</h2>
          <span>{bankCandidates.length} available</span>
        </div>
        {#if loading}
          <div class="empty">Loading bank candidates...</div>
        {:else if bankCandidates.length === 0}
          <div class="empty">No bank-derived expense candidates waiting to import.</div>
        {:else}
          <div class="list">
            {#each bankCandidates as candidate}
              <div class="list-row">
                <div>
                  <strong>{candidate.description}</strong>
                  <div class="meta">{candidate.category} • {candidate.expense_date?.slice(0, 10)}</div>
                </div>
                <div class="right">
                  <div>{currency(candidate.amount + (candidate.vat_amount || 0))}</div>
                  <button class="secondary" onclick={() => importBankCandidate(candidate)}>Import</button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </article>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h2>Expense Ledger</h2>
        <span>{sortedEntries.length} entries</span>
      </div>
      {#if loading}
        <div class="empty">Loading expense ledger...</div>
      {:else if sortedEntries.length === 0}
        <div class="empty">No expense entries yet.</div>
      {:else}
        <div class="list">
          {#each sortedEntries as entry}
            <div class="list-row">
              <div>
                <strong>{entry.entry_number} • {entry.description}</strong>
                <div class="meta">{entry.category_name || "Uncategorized"} {entry.vendor_name ? `• ${entry.vendor_name}` : ""}</div>
                <div class="meta">{entry.expense_date?.slice(0, 10)} • {entry.status} • {entry.payment_status || "unpaid"}</div>
                {#if entry.notes}
                  <div class="note-preview">{entry.notes}</div>
                {/if}
                {#if entry.payment_status === "paid"}
                  <div class="meta">
                    {entry.payment_method || "Payment"} • {entry.payment_reference || "No ref"} {entry.paid_at ? `• ${entry.paid_at.slice(0, 10)}` : ""}
                  </div>
                {/if}
              </div>
              <div class="right actions tight">
                <div>{currency(entry.total_amount)}</div>
                <div class="inline-actions">
                  {#if entry.status === "draft"}
                    <button class="secondary" onclick={() => runEntryAction("submit", entry.id)}>Submit</button>
                  {:else if entry.status === "approved"}
                    <button class="secondary" onclick={() => runEntryAction("post", entry.id)}>Post</button>
                  {:else if entry.status === "posted"}
                    <button class="secondary" onclick={() => openPaymentModal(entry)}>Record Payment</button>
                  {/if}
                  {#if canDeleteEntry(entry)}
                    <button
                      type="button"
                      class="danger ghost"
                      class:confirming={confirmDeleteEntryID === entry.id}
                      disabled={deletingEntryID === entry.id}
                      onclick={() => handleDeleteEntry(entry)}
                    >
                      {deletingEntryID === entry.id ? "Deleting..." : confirmDeleteEntryID === entry.id ? "Confirm Delete" : "Delete"}
                    </button>
                  {/if}
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </section>
  {/if}

  {#if showRecurring}
    <section class="layout">
      <article class="panel">
        <div class="panel-head">
          <h2>Recurring Schedule</h2>
          <span>Generate future overheads automatically</span>
        </div>
        <div class="form-grid">
          <input bind:value={recurringName} placeholder="Recurring expense name" />
          <select bind:value={recurringCategoryID}>
            <option value="">Select category</option>
            {#each categories as category}
              <option value={category.id}>{category.name}</option>
            {/each}
          </select>
          <select bind:value={recurringVendorID}>
            <option value="">No vendor</option>
            {#each vendors as vendor}
              <option value={vendor.id}>{vendor.name}</option>
            {/each}
          </select>
          <input bind:value={recurringAmount} min="0" step="0.001" type="number" placeholder="Amount" />
          <input bind:value={recurringVATAmount} min="0" step="0.001" type="number" placeholder="VAT amount" />
          <select bind:value={recurringFrequency}>
            <option value="monthly">Monthly</option>
            <option value="quarterly">Quarterly</option>
            <option value="yearly">Yearly</option>
            <option value="weekly">Weekly</option>
          </select>
          <label>
            <span>Next Run Date</span>
            <input bind:value={recurringNextRunDate} type="date" />
          </label>
          <label class="toggle">
            <span>Auto Submit</span>
            <input bind:checked={recurringAutoSubmit} type="checkbox" />
          </label>
        </div>
        <div class="actions">
          <button onclick={handleCreateRecurring}>Create Schedule</button>
          <button class="secondary" onclick={handleGenerateRecurring}>Generate Due Items</button>
        </div>
      </article>

      <article class="panel">
        <div class="panel-head">
          <h2>Active Schedules</h2>
          <span>{recurringExpenses.length} active</span>
        </div>
        {#if loading}
          <div class="empty">Loading recurring schedules...</div>
        {:else if recurringExpenses.length === 0}
          <div class="empty">No recurring expense schedules yet.</div>
	        {:else}
	          <div class="list">
            {#each recurringExpenses as item}
              <div class="list-row">
                <div>
                  <strong>{item.name}</strong>
                  <div class="meta">{item.category_name || "Uncategorized"} {item.vendor_name ? `• ${item.vendor_name}` : ""}</div>
                  <div class="meta">{item.frequency} • next {item.next_run_date?.slice(0, 10)}</div>
                </div>
                <div class="right actions tight recurring-actions">
                  <div class="recurring-summary">
                    <div>{currency((item.default_amount || 0) + (item.default_vat_amount || 0))}</div>
                    <div class="meta">{item.auto_submit ? "Auto-submit" : "Draft"}</div>
                  </div>
                  <button
                    type="button"
                    class="danger ghost recurring-delete"
                    class:confirming={confirmDeleteRecurringID === item.id}
                    disabled={deletingRecurringID === item.id}
                    onclick={() => handleDeleteRecurring(item)}
                  >
                    {deletingRecurringID === item.id ? "Deleting..." : confirmDeleteRecurringID === item.id ? "Confirm Delete" : "Delete Schedule"}
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </article>
    </section>
  {/if}

  {#if mode === "approvals"}
    <section class="panel">
      <div class="panel-head">
        <h2>Approvals Queue</h2>
        <span>{approvalQueue.length} needing action</span>
      </div>
      {#if loading}
        <div class="empty">Loading approvals...</div>
      {:else if approvalQueue.length === 0}
        <div class="empty">No submitted or approved expenses waiting for action.</div>
      {:else}
        <div class="list">
          {#each approvalQueue as entry}
            <div class="list-row">
              <div>
                <strong>{entry.entry_number} • {entry.description}</strong>
                <div class="meta">{entry.category_name || "Uncategorized"} • {entry.status} • {entry.payment_status || "unpaid"}</div>
                <div class="meta">Due {entry.due_date?.slice(0, 10) || entry.expense_date?.slice(0, 10)}</div>
                {#if entry.notes}
                  <div class="note-preview">{entry.notes}</div>
                {/if}
              </div>
              <div class="right actions tight">
                <div>{currency(entry.total_amount)}</div>
                {#if entry.status === "submitted"}
                  <button class="secondary" onclick={() => runEntryAction("approve", entry.id)}>Approve</button>
                  <button class="danger" onclick={() => runEntryAction("reject", entry.id)}>Reject</button>
                {:else if entry.status === "approved"}
                  <button class="secondary" onclick={() => runEntryAction("post", entry.id)}>Post</button>
                {:else if entry.status === "posted"}
                  <button class="secondary" onclick={() => openPaymentModal(entry)}>Record Payment</button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </section>
  {/if}
</div>

{#if showPaymentModal && paymentEntry}
  <div class="modal-overlay" role="button" tabindex="0" onclick={self(() => { showPaymentModal = false; paymentEntry = null; })} onkeydown={(event) => (event.key === "Enter" || event.key === " ") && (showPaymentModal = false, paymentEntry = null)}>
    <div class="modal">
      <div class="panel-head">
        <div>
          <h2>Record Expense Payment</h2>
          <span>{paymentEntry.entry_number} • {paymentEntry.description}</span>
        </div>
      </div>
      <div class="form-grid">
        <label>
          <span>Payment Date</span>
          <input bind:value={expensePaymentForm.paidAt} type="date" />
        </label>
        <label>
          <span>Method</span>
          <select bind:value={expensePaymentForm.paymentMethod}>
            {#each expensePaymentMethods as method}
              <option value={method}>{method}</option>
            {/each}
          </select>
        </label>
        <label>
          <span>Bank Account</span>
          <select bind:value={expensePaymentForm.bankAccountID}>
            <option value="">Select bank account</option>
            {#each bankAccounts as bank}
              <option value={bank.id}>{bank.bank_name} • {bank.account_number}</option>
            {/each}
          </select>
        </label>
        <label>
          <span>Reference</span>
          <input bind:value={expensePaymentForm.paymentReference} placeholder="NEFT / cheque / transfer reference" />
        </label>
      </div>
      <div class="actions modal-actions">
        <button class="secondary" onclick={() => { showPaymentModal = false; paymentEntry = null; }}>Cancel</button>
        <button disabled={payingExpense} onclick={submitExpensePayment}>
          {payingExpense ? "Saving..." : "Mark Paid"}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .page {
    display: grid;
    gap: 18px;
  }

  .page.embedded {
    padding: 0;
  }

  .summary-grid,
  .layout,
  .form-grid,
  .list {
    display: grid;
    gap: 12px;
  }

  .summary-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .layout {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .form-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .summary-card,
  .panel,
  .modal {
    border: 1px solid var(--border, #e5e7eb);
    border-radius: 16px;
    background: var(--surface, #fff);
    padding: 16px;
  }

  .summary-card span,
  .meta,
  .note-preview,
  label span {
    color: var(--text-secondary, #6b7280);
  }

  .summary-card strong {
    display: block;
    margin-top: 8px;
    font-size: 1.3rem;
  }

  .panel-head,
  .list-row,
  .actions,
  .tool-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
  }

  .entry-form {
    display: grid;
    gap: 16px;
  }

  .field-group {
    border: 1px solid var(--border, #e5e7eb);
    border-radius: 14px;
    padding: 14px;
    margin: 0;
  }

  .field-group legend {
    padding: 0 6px;
    font-size: 0.78rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary, #6b7280);
  }

  .field-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }

  .setup-disclosure {
    margin-top: 16px;
    border: 1px dashed var(--border, #d1d5db);
    border-radius: 14px;
    padding: 12px 14px;
  }

  .setup-disclosure summary {
    cursor: pointer;
    font-weight: 600;
    color: var(--text-secondary, #6b7280);
  }

  .setup-disclosure[open] summary {
    margin-bottom: 10px;
  }

  .mini-tools {
    display: grid;
    gap: 10px;
    margin: 14px 0;
  }

  .category-manager {
    display: grid;
    gap: 10px;
    margin-bottom: 14px;
  }

  .category-manager-head,
  .category-chip,
  .category-copy {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
  }

  .category-list {
    display: grid;
    gap: 8px;
    max-height: 180px;
    overflow: auto;
    padding-right: 4px;
  }

  .inline-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    justify-content: flex-end;
  }

  .note-preview {
    margin-top: 6px;
    padding: 8px 10px;
    border-radius: 10px;
    background: color-mix(in srgb, var(--surface, #fff) 88%, var(--accent-primary, #2563eb) 12%);
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
  }

  .category-chip {
    border: 1px solid var(--border, #e5e7eb);
    border-radius: 12px;
    padding: 10px 12px;
    background: color-mix(in srgb, var(--surface, #fff) 94%, var(--accent-primary, #2563eb) 6%);
  }

  .category-chip.selected {
    border-color: color-mix(in srgb, var(--accent-primary, #2563eb) 52%, #ffffff 48%);
    background: color-mix(in srgb, var(--surface, #fff) 80%, var(--accent-primary, #2563eb) 20%);
  }

  .category-copy {
    flex: 1;
    min-width: 0;
  }

  .category-copy strong,
  .category-copy span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .category-copy span {
    color: var(--text-secondary, #6b7280);
    font-size: 0.86rem;
  }

  .list {
    margin-top: 14px;
  }

  .list-row {
    border: 1px solid var(--border, #e5e7eb);
    border-radius: 14px;
    padding: 12px;
    background: color-mix(in srgb, var(--surface, #fff) 92%, var(--accent-primary, #2563eb) 8%);
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
    width: 100%;
    border: 1px solid var(--border, #d1d5db);
    border-radius: 12px;
    padding: 11px 12px;
    background: var(--surface, #fff);
    color: var(--text-primary, #111827);
    box-sizing: border-box;
  }

  textarea {
    min-height: 96px;
    margin-top: 12px;
    resize: vertical;
  }

  label {
    display: grid;
    gap: 6px;
  }

  button {
    border: 1px solid var(--border, #d1d5db);
    border-radius: 12px;
    padding: 10px 14px;
    background: color-mix(in srgb, var(--surface, #fff) 84%, var(--accent-primary, #2563eb) 16%);
    color: var(--text-primary, #111827);
    cursor: pointer;
  }

  button.secondary {
    background: color-mix(in srgb, var(--surface, #fff) 90%, #d97706 10%);
  }

  button.danger {
    background: color-mix(in srgb, var(--surface, #fff) 86%, #dc2626 14%);
  }

  button.ghost {
    padding: 8px 12px;
    white-space: nowrap;
  }

  button.confirming {
    background: color-mix(in srgb, var(--surface, #fff) 78%, #dc2626 22%);
    border-color: color-mix(in srgb, #dc2626 55%, var(--border, #d1d5db) 45%);
    color: #7f1d1d;
  }

  .empty {
    border: 1px dashed var(--border, #d1d5db);
    border-radius: 14px;
    padding: 18px;
    color: var(--text-secondary, #6b7280);
    margin-top: 14px;
  }

  .empty.compact {
    margin-top: 0;
    padding: 12px;
  }

  .right {
    text-align: right;
  }

  .tight {
    flex-wrap: wrap;
    justify-content: flex-end;
  }

  .recurring-actions {
    min-width: 180px;
    align-items: flex-end;
  }

  .recurring-summary {
    display: grid;
    gap: 4px;
    justify-items: end;
  }

  .recurring-delete {
    min-width: 140px;
    justify-self: end;
  }

  .toggle {
    grid-template-columns: 1fr auto;
    align-items: center;
  }

  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(15, 23, 42, 0.38);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
    z-index: 1000;
  }

  .modal {
    width: min(620px, 100%);
    display: grid;
    gap: 14px;
  }

  .modal-actions {
    margin-top: 0;
  }

  @media (max-width: 1100px) {
    .summary-grid,
    .layout,
    .form-grid,
    .field-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
