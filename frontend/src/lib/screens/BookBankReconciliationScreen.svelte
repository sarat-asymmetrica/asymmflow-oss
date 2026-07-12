<script lang="ts">
  /**
   * BookBankReconciliationScreen - Book vs Bank Balance Reconciliation
   *
   * Features:
   * - Side-by-side book and bank balance comparison
   * - Adjustment entries (bank charges, interest, NSF)
   * - Outstanding cheques and deposits in transit
   * - Variance analysis and explanation
   * - Finalization workflow
   *
   * Design System: Wabi-Sabi minimalism x Bloomberg data density
   */

  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  // Wails API imports
  import {
    GetActiveBankAccounts } from '../../../wailsjs/go/main/App';
import { GetBookBankReconciliations, CreateBookBankReconciliation, UpdateBookBankReconciliation, UpdateBookBankReconciliationAdjustments, FinalizeBookBankReconciliation, GetBookBankReconciliationReport, GetReconciliationVariances, GetDepositsInTransit } from '../../../wailsjs/go/main/InfraService';
import { GetOutstandingCheques } from '../../../wailsjs/go/main/FinanceService';

  // Design system components
  import PageLayout from '$lib/components/layout/PageLayout.svelte';
  import DataTable from '$lib/components/ui/DataTable.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Modal from '$lib/components/layout/Modal.svelte';
  import FormGroup from '$lib/components/ui/FormGroup.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import { toast } from '$lib/stores/toasts';
  import { currentUser } from '$lib/stores/authContext';
  import { formatNumber } from '$lib/utils/formatters';


  interface Props {
    // Props
    embedded?: boolean;
  }

  let { embedded = false }: Props = $props();

  // Types - aligned with Wails CompanyBankAccount (no balance field, it's calculated)
  interface BankAccount {
    id: string;
    bank_name: string;
    account_name?: string;
    account_number: string;
    currency: string;
    is_active?: boolean;
  }

  interface BookBankReconciliation {
    id: string;
    bank_account_id: string;
    reconciliation_date: any;
    bank_statement_balance: number;
    deposits_in_transit: number;
    outstanding_cheques: number;
    bank_errors: number;
    adjusted_bank_balance: number;
    book_balance: number;
    bank_charges_not_recorded: number;
    interest_not_recorded: number;
    nsf_cheques: number;
    book_errors: number;
    adjusted_book_balance: number;
    difference: number;
    is_reconciled: boolean;
    reconciled_by: string;
    reconciled_at: any;
    notes: string;
  }

  interface VarianceItem {
    category: string;
    description: string;
    amount: number;
    type: string;
  }

  // State - using any[] to accommodate Wails-generated classes
  let bankAccounts: any[] = $state([]);
  let reconciliations: any[] = $state([]);
  let selectedAccountId = $state('');
  let selectedRecon: any | null = $state(null);
  let variances: any[] = $state([]);
  let loading = $state(true);

  // Modal state
  let showNewModal = $state(false);
  let showEditModal = $state(false);
  let modalLoading = $state(false);

  // Form state
  let formAsOfDate = $state('');
  let formBankBalance = $state('');
  let formBookBalance = $state('');
  let formBankCharges = $state('0');
  let formInterest = $state('0');
  let formNSF = $state('0');
  let formBankErrors = $state('0');
  let formBookErrors = $state('0');
  let formNotes = $state('');

  // Wave 9.3 B1b: register-sourced deposits-in-transit / outstanding-cheque
  // line items and their editable totals, pre-populated on the New
  // Reconciliation modal so the user confirms (or adjusts) real register
  // data instead of trusting a silent server auto-compute.
  let depositsInTransitLines: any[] = $state([]);
  let outstandingChequeLines: any[] = $state([]);
  let formDepositsInTransit = $state('0');
  let formOutstandingCheques = $state('0');
  let loadingRegisterTotals = $state(false);

  // DataTable columns
  const reconColumns = [
    {
      key: 'reconciliation_date',
      label: 'Date',
      sortable: true,
      width: '110px',
      render: (row: BookBankReconciliation) => `<span style="font-family: var(--font-mono); font-size: 12px;">${formatDate(row.reconciliation_date)}</span>`
    },
    {
      key: 'bank_statement_balance',
      label: 'Bank Balance',
      sortable: true,
      width: '130px',
      align: 'right' as const,
      render: (row: BookBankReconciliation) => `<span style="font-family: var(--font-mono);">${formatBHD(row.bank_statement_balance)}</span>`
    },
    {
      key: 'book_balance',
      label: 'Book Balance',
      sortable: true,
      width: '130px',
      align: 'right' as const,
      render: (row: BookBankReconciliation) => `<span style="font-family: var(--font-mono);">${formatBHD(row.book_balance)}</span>`
    },
    {
      key: 'difference',
      label: 'Difference',
      sortable: true,
      width: '110px',
      align: 'right' as const,
      render: (row: BookBankReconciliation) => {
        const color = Math.abs(row.difference) < 0.001 ? '#10B981' : '#EF4444';
        return `<span style="font-family: var(--font-mono); color: ${color}; font-weight: 600;">${formatBHD(row.difference)}</span>`;
      }
    },
    {
      key: 'is_reconciled',
      label: 'Status',
      sortable: true,
      width: '100px',
      render: (row: BookBankReconciliation) => {
        if (row.is_reconciled) {
          return `<span style="color: #10B981; font-weight: 500; font-size: 12px;">Reconciled</span>`;
        }
        return `<span style="color: #F59E0B; font-weight: 500; font-size: 12px;">In Progress</span>`;
      }
    }
  ];

  // Formatters
  function formatBHD(value: number): string {
    return formatNumber(value || 0, 3);
  }

  function formatDate(dateValue: any): string {
    if (!dateValue) return '-';
    try {
      const date = typeof dateValue === 'string' ? new Date(dateValue) : new Date(dateValue);
      return date.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });
    } catch {
      return '-';
    }
  }

  // Data loading
  async function loadData() {
    loading = true;
    try {
      const accounts = await GetActiveBankAccounts();
      bankAccounts = accounts || [];

      if (bankAccounts.length > 0 && !selectedAccountId) {
        selectedAccountId = bankAccounts[0].id;
      }

      await loadReconciliations();
    } catch (err) {
      console.error('Failed to load data:', err);
      toast.danger('Failed to load reconciliation data');
    } finally {
      loading = false;
    }
  }

  async function loadReconciliations() {
    if (!selectedAccountId) return;

    try {
      const result = await GetBookBankReconciliations(selectedAccountId);
      reconciliations = result || [];
    } catch (err) {
      console.error('Failed to load reconciliations:', err);
    }
  }

  async function loadVariances(recon: BookBankReconciliation) {
    selectedRecon = recon;
    try {
      const result = await GetReconciliationVariances(recon.id);
      variances = result || [];
    } catch (err) {
      console.error('Failed to load variances:', err);
      variances = [];
    }
  }

  // Actions
  function todayForInput() {
    return new Date().toISOString().split('T')[0];
  }

  // Wave 9.3 B1b: pre-populate the DIT / outstanding-cheque line items (and
  // their editable totals) from the account's register, so the prove step
  // shows the user real data to confirm instead of a silent auto-compute.
  async function loadRegisterTotals() {
    if (!selectedAccountId) return;
    loadingRegisterTotals = true;
    try {
      const depositsResult: any = await GetDepositsInTransit(selectedAccountId).catch(() => null);
      const chequesResult: any = await GetOutstandingCheques(selectedAccountId).catch(() => null);
      depositsInTransitLines = depositsResult?.deposits || [];
      formDepositsInTransit = String(depositsResult?.total || 0);
      outstandingChequeLines = chequesResult?.cheques || [];
      formOutstandingCheques = String(chequesResult?.total || 0);
    } catch (err) {
      console.error('Failed to load register totals:', err);
    } finally {
      loadingRegisterTotals = false;
    }
  }

  function openNewModal() {
    formAsOfDate = todayForInput();
    formBankBalance = '';
    formBookBalance = '';
    formBankCharges = '0';
    formInterest = '0';
    formNSF = '0';
    formBankErrors = '0';
    formBookErrors = '0';
    formNotes = '';
    depositsInTransitLines = [];
    outstandingChequeLines = [];
    formDepositsInTransit = '0';
    formOutstandingCheques = '0';
    showNewModal = true;
    void loadRegisterTotals();
  }

  async function handleCreate() {
    if (!formBankBalance || !formBookBalance) {
      toast.warning('Please enter both balances');
      return;
    }
    if (!formAsOfDate) {
      toast.warning('Please choose an as-of date');
      return;
    }

    modalLoading = true;
    try {
      // Wave 9.3 B1b/B1c: single-write create carries the as-of date plus the
      // user-confirmed (register-pre-populated, editable) DIT / outstanding-cheque
      // totals. A negative value would fall back to the register auto-compute.
      const recon = await CreateBookBankReconciliation(
        selectedAccountId,
        new Date(formAsOfDate),
        parseFloat(formBankBalance),
        parseFloat(formBookBalance),
        parseFloat(formDepositsInTransit) || 0,
        parseFloat(formOutstandingCheques) || 0
      );

      if (recon) {
        await UpdateBookBankReconciliation(
          recon.id,
          parseFloat(formBankCharges),
          parseFloat(formInterest),
          parseFloat(formNSF),
          parseFloat(formBankErrors),
          parseFloat(formBookErrors),
          formNotes
        );
      }

      toast.success('Reconciliation created');
      showNewModal = false;
      await loadReconciliations();
      if (recon?.id) {
        const refreshed = reconciliations.find((item) => item.id === recon.id);
        if (refreshed) {
          await loadVariances(refreshed);
        }
      }
    } catch (err: any) {
      console.error('Failed to create reconciliation:', err);
      toast.danger(err?.message || 'Failed to create reconciliation');
    } finally {
      modalLoading = false;
    }
  }

  function openEditModal(recon: BookBankReconciliation) {
    selectedRecon = recon;
    formBankBalance = recon.bank_statement_balance.toString();
    formBookBalance = recon.book_balance.toString();
    formBankCharges = recon.bank_charges_not_recorded.toString();
    formInterest = recon.interest_not_recorded.toString();
    formNSF = recon.nsf_cheques.toString();
    formBankErrors = recon.bank_errors.toString();
    formBookErrors = recon.book_errors.toString();
    formNotes = recon.notes || '';
    // C3: deposits-in-transit / outstanding-cheque totals are editable here too
    // (server refuses the write once the reconciliation is finalized).
    formDepositsInTransit = (recon.deposits_in_transit ?? 0).toString();
    formOutstandingCheques = (recon.outstanding_cheques ?? 0).toString();
    showEditModal = true;
  }

  async function handleUpdate() {
    if (!selectedRecon) return;

    modalLoading = true;
    try {
      // C3: DIT / outstanding-cheque totals go through their own dedicated
      // method (UpdateBookBankReconciliationAdjustments) which recomputes
      // AdjustedBankBalance/Difference; the other five adjustment fields go
      // through UpdateBookBankReconciliation. Both refuse on a finalized recon.
      await UpdateBookBankReconciliationAdjustments(
        selectedRecon.id,
        parseFloat(formDepositsInTransit) || 0,
        parseFloat(formOutstandingCheques) || 0
      );
      await UpdateBookBankReconciliation(
        selectedRecon.id,
        parseFloat(formBankCharges),
        parseFloat(formInterest),
        parseFloat(formNSF),
        parseFloat(formBankErrors),
        parseFloat(formBookErrors),
        formNotes
      );

      toast.success('Reconciliation updated');
      showEditModal = false;
      await loadReconciliations();
      const refreshed = reconciliations.find((item) => item.id === selectedRecon.id);
      if (refreshed) {
        await loadVariances(refreshed);
      } else {
        selectedRecon = {
          ...selectedRecon,
          deposits_in_transit: parseFloat(formDepositsInTransit) || 0,
          outstanding_cheques: parseFloat(formOutstandingCheques) || 0,
          bank_charges_not_recorded: parseFloat(formBankCharges),
          interest_not_recorded: parseFloat(formInterest),
          nsf_cheques: parseFloat(formNSF),
          bank_errors: parseFloat(formBankErrors),
          book_errors: parseFloat(formBookErrors),
          notes: formNotes,
        };
      }
    } catch (err: any) {
      console.error('Failed to update:', err);
      toast.danger(err?.message || 'Failed to update');
    } finally {
      modalLoading = false;
    }
  }

  async function handleFinalize(recon: BookBankReconciliation) {
    if (Math.abs(recon.difference) > 0.001) {
      toast.warning('Cannot finalize with non-zero difference');
      return;
    }
    // Wave 9.3 B2: server re-resolves and ignores this value (Article III.4);
    // block here so the UI never sends a fake actor.
    if (!$currentUser?.id) {
      toast.danger('Cannot finalize: no authenticated user found. Please sign in again.');
      return;
    }

    try {
      await FinalizeBookBankReconciliation(recon.id, $currentUser.id);
      toast.success('Reconciliation finalized');
      await loadReconciliations();
    } catch (err: any) {
      console.error('Failed to finalize:', err);
      toast.danger(err?.message || 'Failed to finalize');
    }
  }

  function goToMatchTransactions() {
    window.dispatchEvent(new CustomEvent('finance:navigate', {
      detail: { tab: 'bank_recon' }
    }));
  }

  onMount(() => {
    loadData();
  });
</script>

<PageLayout title="Book-Bank Reconciliation" subtitle="Reconcile book balance with bank statement" {embedded}>
  {#if loading}
    <div class="loading-container">
      <WabiSpinner size="lg" />
      <p>Loading reconciliation data...</p>
    </div>
  {:else}
    <!-- Wave 9.3 B1a: name the two-step month-end sequence and link back to step 1. -->
    <div class="close-month-header">
      <span class="close-month-step">Close the Month · Step 2: Prove the balance</span>
      <button type="button" class="close-month-jump" onclick={goToMatchTransactions}>
        ← Step 1: Match transactions
      </button>
    </div>

    <!-- Toolbar -->
    <div class="toolbar">
      <div class="left">
        <select bind:value={selectedAccountId} onchange={loadReconciliations} class="account-select">
          {#each bankAccounts as account}
            <option value={account.id}>{account.bank_name} - {account.account_number}</option>
          {/each}
        </select>
      </div>
      <div class="right">
        <Button variant="primary" on:click={openNewModal}>
          New Reconciliation
        </Button>
      </div>
    </div>

    <!-- Split View -->
    <div class="split-view">
      <!-- Reconciliations List -->
      <div class="panel list-panel">
        <h3>Reconciliations</h3>
        <DataTable
          data={reconciliations}
          columns={reconColumns}
          onRowClick={loadVariances}
          selectedId={selectedRecon?.id}
          emptyMessage="No reconciliations yet"
        />
      </div>

      <!-- Detail Panel -->
      <div class="panel detail-panel">
        {#if selectedRecon}
          <h3>Reconciliation Detail - {formatDate(selectedRecon.reconciliation_date)}</h3>

          <!-- Balance Comparison -->
          <div class="balance-comparison">
            <div class="balance-side bank">
              <h4>Bank Side</h4>
              <div class="balance-row">
                <span>Statement Balance</span>
                <span class="value">{formatBHD(selectedRecon.bank_statement_balance)}</span>
              </div>
              <div class="balance-row adjustment positive">
                <span>+ Deposits in Transit</span>
                <span class="value">{formatBHD(selectedRecon.deposits_in_transit)}</span>
              </div>
              <div class="balance-row adjustment negative">
                <span>- Outstanding Cheques</span>
                <span class="value">{formatBHD(selectedRecon.outstanding_cheques)}</span>
              </div>
              <div class="balance-row adjustment">
                <span>+/- Bank Errors</span>
                <span class="value">{formatBHD(selectedRecon.bank_errors)}</span>
              </div>
              <div class="balance-row total">
                <span>Adjusted Bank Balance</span>
                <span class="value">{formatBHD(selectedRecon.adjusted_bank_balance)}</span>
              </div>
            </div>

            <div class="balance-side book">
              <h4>Book Side</h4>
              <div class="balance-row">
                <span>Book Balance</span>
                <span class="value">{formatBHD(selectedRecon.book_balance)}</span>
              </div>
              <div class="balance-row adjustment negative">
                <span>- Bank Charges</span>
                <span class="value">{formatBHD(selectedRecon.bank_charges_not_recorded)}</span>
              </div>
              <div class="balance-row adjustment positive">
                <span>+ Interest Income</span>
                <span class="value">{formatBHD(selectedRecon.interest_not_recorded)}</span>
              </div>
              <div class="balance-row adjustment negative">
                <span>- NSF Cheques</span>
                <span class="value">{formatBHD(selectedRecon.nsf_cheques)}</span>
              </div>
              <div class="balance-row adjustment">
                <span>+/- Book Errors</span>
                <span class="value">{formatBHD(selectedRecon.book_errors)}</span>
              </div>
              <div class="balance-row total">
                <span>Adjusted Book Balance</span>
                <span class="value">{formatBHD(selectedRecon.adjusted_book_balance)}</span>
              </div>
            </div>
          </div>

          <!-- Difference -->
          <div class="difference-box" class:reconciled={Math.abs(selectedRecon.difference) < 0.001}>
            <span class="label">Difference</span>
            <span class="value">{formatBHD(selectedRecon.difference)} BHD</span>
          </div>

          {#if selectedRecon.notes}
            <div class="notes-card">
              <div class="notes-head">
                <span>Reconciliation Notes</span>
                <span>Context</span>
              </div>
              <p>{selectedRecon.notes}</p>
            </div>
          {/if}

          <!-- Variances -->
          {#if variances.length > 0}
            <div class="variances">
              <h4>Variance Analysis</h4>
              {#each variances as v}
                <div class="variance-row" class:bank={v.type === 'BANK_ADJUSTMENT'} class:book={v.type === 'BOOK_ADJUSTMENT'}>
                  <span class="category">{v.category}</span>
                  <span class="description">{v.description}</span>
                  <span class="amount" class:positive={v.amount > 0} class:negative={v.amount < 0}>
                    {v.amount > 0 ? '+' : ''}{formatBHD(v.amount)}
                  </span>
                </div>
              {/each}
            </div>
          {/if}

          <!-- Actions -->
          <div class="detail-actions">
            {#if !selectedRecon.is_reconciled}
              <Button variant="secondary" on:click={() => selectedRecon && openEditModal(selectedRecon)}>
                Edit Adjustments
              </Button>
              <Button variant="primary" on:click={() => selectedRecon && handleFinalize(selectedRecon)} disabled={Math.abs(selectedRecon.difference) > 0.001}>
                Finalize
              </Button>
            {:else}
              <div class="reconciled-badge">
                Reconciled by {selectedRecon.reconciled_by} on {formatDate(selectedRecon.reconciled_at)}
              </div>
            {/if}
          </div>
        {:else}
          <div class="empty-state">
            <p>Select a reconciliation to view details</p>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</PageLayout>

<!-- New Reconciliation Modal -->
<Modal bind:open={showNewModal} title="New Book-Bank Reconciliation" size="md">
  <div class="form-grid">
    <FormGroup label="As-of Date">
      <input type="date" bind:value={formAsOfDate} class="form-textarea" />
    </FormGroup>
    <FormGroup label="Bank Statement Balance (BHD)">
      <Input type="number" step="0.001" bind:value={formBankBalance} placeholder="0.000" />
    </FormGroup>
    <FormGroup label="Book Balance (BHD)">
      <Input type="number" step="0.001" bind:value={formBookBalance} placeholder="0.000" />
    </FormGroup>
    <hr />
    <h4>Deposits in Transit &amp; Outstanding Cheques</h4>
    <p class="register-hint">Pre-populated from the register for this account. Review and adjust the totals if needed.</p>
    {#if loadingRegisterTotals}
      <div class="loading-container small"><WabiSpinner size="sm" /></div>
    {:else}
      {#if depositsInTransitLines.length > 0}
        <div class="register-lines">
          <span class="register-lines-label">Deposits in Transit ({depositsInTransitLines.length})</span>
          {#each depositsInTransitLines as d}
            <div class="register-line-row">
              <span>{d.description || d.deposit_slip_no || 'Deposit'}</span>
              <span class="value">{formatBHD(d.amount)}</span>
            </div>
          {/each}
        </div>
      {/if}
      {#if outstandingChequeLines.length > 0}
        <div class="register-lines">
          <span class="register-lines-label">Outstanding Cheques ({outstandingChequeLines.length})</span>
          {#each outstandingChequeLines as c}
            <div class="register-line-row">
              <span>#{c.cheque_number} — {c.payee_name}</span>
              <span class="value">{formatBHD(c.amount)}</span>
            </div>
          {/each}
        </div>
      {/if}
    {/if}
    <div class="adjustment-grid">
      <FormGroup label="Deposits in Transit Total">
        <Input type="number" step="0.001" bind:value={formDepositsInTransit} />
      </FormGroup>
      <FormGroup label="Outstanding Cheques Total">
        <Input type="number" step="0.001" bind:value={formOutstandingCheques} />
      </FormGroup>
    </div>
    <hr />
    <h4>Adjustments</h4>
    <div class="adjustment-grid">
      <FormGroup label="Bank Charges">
        <Input type="number" step="0.001" bind:value={formBankCharges} />
      </FormGroup>
      <FormGroup label="Interest Income">
        <Input type="number" step="0.001" bind:value={formInterest} />
      </FormGroup>
      <FormGroup label="NSF Cheques">
        <Input type="number" step="0.001" bind:value={formNSF} />
      </FormGroup>
      <FormGroup label="Bank Errors">
        <Input type="number" step="0.001" bind:value={formBankErrors} />
      </FormGroup>
      <FormGroup label="Book Errors">
        <Input type="number" step="0.001" bind:value={formBookErrors} />
      </FormGroup>
    </div>
    <FormGroup label="Notes">
      <textarea bind:value={formNotes} rows="3" class="form-textarea" placeholder="Add notes..."></textarea>
    </FormGroup>
  </div>
  {#snippet footer()}
  
      <Button variant="secondary" on:click={() => showNewModal = false}>Cancel</Button>
      <Button variant="primary" on:click={handleCreate} loading={modalLoading}>
        Create
      </Button>
    
  {/snippet}
</Modal>

<!-- Edit Modal -->
<Modal bind:open={showEditModal} title="Edit Adjustments" size="md">
  <div class="form-grid">
    <h4>Deposits in Transit &amp; Outstanding Cheques</h4>
    <div class="adjustment-grid">
      <FormGroup label="Deposits in Transit Total">
        <Input type="number" step="0.001" bind:value={formDepositsInTransit} />
      </FormGroup>
      <FormGroup label="Outstanding Cheques Total">
        <Input type="number" step="0.001" bind:value={formOutstandingCheques} />
      </FormGroup>
    </div>
    <hr />
    <h4>Adjustments</h4>
    <div class="adjustment-grid">
      <FormGroup label="Bank Charges">
        <Input type="number" step="0.001" bind:value={formBankCharges} />
      </FormGroup>
      <FormGroup label="Interest Income">
        <Input type="number" step="0.001" bind:value={formInterest} />
      </FormGroup>
      <FormGroup label="NSF Cheques">
        <Input type="number" step="0.001" bind:value={formNSF} />
      </FormGroup>
      <FormGroup label="Bank Errors">
        <Input type="number" step="0.001" bind:value={formBankErrors} />
      </FormGroup>
      <FormGroup label="Book Errors">
        <Input type="number" step="0.001" bind:value={formBookErrors} />
      </FormGroup>
    </div>
    <FormGroup label="Notes">
      <textarea bind:value={formNotes} rows="3" class="form-textarea" placeholder="Add notes..."></textarea>
    </FormGroup>
  </div>
  {#snippet footer()}
  
      <Button variant="secondary" on:click={() => showEditModal = false}>Cancel</Button>
      <Button variant="primary" on:click={handleUpdate} loading={modalLoading}>
        Update
      </Button>
    
  {/snippet}
</Modal>

<style>
  .loading-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: var(--space-8);
    gap: var(--space-4);
  }

  .loading-container.small {
    padding: var(--space-3);
  }

  .close-month-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-2) var(--space-1);
    margin-bottom: var(--space-2);
  }

  .close-month-step {
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.02em;
    color: var(--text-secondary);
  }

  .close-month-jump {
    border: none;
    background: transparent;
    color: var(--brand-indigo);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    padding: 0;
  }

  .close-month-jump:hover {
    text-decoration: underline;
  }

  .register-hint {
    margin: 0;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .register-lines {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    padding: var(--space-2) var(--space-3);
    background: var(--surface-elevated);
    border-radius: var(--radius-sm);
  }

  .register-lines-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .register-line-row {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
  }

  .register-line-row .value {
    font-family: var(--font-mono);
  }

  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--space-4);
    padding: var(--space-3);
    background: var(--surface-elevated);
    border-radius: var(--radius-md);
  }

  .toolbar .left,
  .toolbar .right {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .account-select {
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: var(--surface-default);
    font-size: 14px;
    min-width: 250px;
  }

  .split-view {
    display: grid;
    grid-template-columns: 450px 1fr;
    gap: var(--space-4);
    min-height: 500px;
  }

  .panel {
    background: var(--surface-default);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: var(--space-4);
    overflow: auto;
  }

  .panel h3 {
    margin: 0 0 var(--space-3) 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .balance-comparison {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-4);
    margin-bottom: var(--space-4);
  }

  .balance-side {
    padding: var(--space-3);
    background: var(--surface-elevated);
    border-radius: var(--radius-sm);
  }

  .balance-side h4 {
    margin: 0 0 var(--space-3) 0;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-secondary);
  }

  .balance-row {
    display: flex;
    justify-content: space-between;
    padding: var(--space-1) 0;
    font-size: 13px;
  }

  .balance-row.adjustment {
    color: var(--text-secondary);
    font-size: 12px;
  }

  .balance-row.adjustment.positive .value { color: #10B981; }
  .balance-row.adjustment.negative .value { color: #EF4444; }

  .balance-row.total {
    margin-top: var(--space-2);
    padding-top: var(--space-2);
    border-top: 1px solid var(--border-subtle);
    font-weight: 600;
  }

  .balance-row .value {
    font-family: var(--font-mono);
  }

  .difference-box {
    padding: var(--space-4);
    background: #FEE2E2;
    border-radius: var(--radius-md);
    text-align: center;
    margin-bottom: var(--space-4);
  }

  .difference-box.reconciled {
    background: #D1FAE5;
  }

  .difference-box .label {
    display: block;
    font-size: 12px;
    color: var(--text-secondary);
    margin-bottom: var(--space-1);
  }

  .difference-box .value {
    font-family: var(--font-mono);
    font-size: 24px;
    font-weight: 700;
    color: #EF4444;
  }

  .difference-box.reconciled .value {
    color: #10B981;
  }

  .notes-card {
    margin-bottom: var(--space-4);
    padding: var(--space-3);
    border-radius: var(--radius-md);
    border: 1px solid rgba(59, 130, 246, 0.16);
    background: linear-gradient(135deg, rgba(239, 246, 255, 0.95), rgba(248, 250, 252, 0.9));
    display: grid;
    gap: var(--space-2);
  }

  .notes-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-secondary);
  }

  .notes-card p {
    margin: 0;
    color: var(--text-primary);
    font-size: 13px;
    line-height: 1.65;
    white-space: pre-wrap;
  }

  .variances {
    margin-bottom: var(--space-4);
  }

  .variances h4 {
    margin: 0 0 var(--space-2) 0;
    font-size: 13px;
    font-weight: 600;
  }

  .variance-row {
    display: grid;
    grid-template-columns: 150px 1fr 100px;
    gap: var(--space-2);
    padding: var(--space-2);
    font-size: 12px;
    border-radius: var(--radius-sm);
    margin-bottom: var(--space-1);
  }

  .variance-row.bank { background: rgba(59, 130, 246, 0.1); }
  .variance-row.book { background: rgba(139, 92, 246, 0.1); }

  .variance-row .category { font-weight: 500; }
  .variance-row .description { color: var(--text-secondary); }
  .variance-row .amount {
    text-align: right;
    font-family: var(--font-mono);
  }
  .variance-row .amount.positive { color: #10B981; }
  .variance-row .amount.negative { color: #EF4444; }

  .detail-actions {
    display: flex;
    gap: var(--space-3);
    padding-top: var(--space-3);
    border-top: 1px solid var(--border-subtle);
  }

  .reconciled-badge {
    padding: var(--space-2) var(--space-3);
    background: #D1FAE5;
    color: #059669;
    border-radius: var(--radius-sm);
    font-size: 12px;
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 200px;
    color: var(--text-secondary);
  }

  .form-grid {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .form-grid h4 {
    margin: 0;
    font-size: 13px;
    color: var(--text-secondary);
  }

  .form-grid hr {
    border: none;
    border-top: 1px solid var(--border-subtle);
    margin: var(--space-2) 0;
  }

  .adjustment-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-3);
  }

  .form-textarea {
    width: 100%;
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    font-size: 14px;
    font-family: inherit;
    resize: vertical;
  }

  @media (max-width: 1000px) {
    .split-view {
      grid-template-columns: 1fr;
    }

    .balance-comparison {
      grid-template-columns: 1fr;
    }
  }
</style>
