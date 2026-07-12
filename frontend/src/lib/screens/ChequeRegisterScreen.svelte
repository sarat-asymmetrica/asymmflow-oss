<script lang="ts">
  /**
   * ChequeRegisterScreen - Cheque Lifecycle Management
   *
   * Features:
   * - Cheque book registration
   * - Cheque issuance workflow
   * - Status tracking (Issued -> Presented -> Cleared/Stale/Bounced)
   * - Outstanding cheques report
   * - Stale cheque identification (>6 months)
   *
   * Design System: Wabi-Sabi minimalism x Bloomberg data density
   */

  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  // Wails API imports
  import {
    GetActiveBankAccounts } from '../../../wailsjs/go/main/App';
import { GetChequeRegisters, CreateChequeRegister, GetOutstandingCheques, GetStaleCheques, IssueCheque, MarkChequeCleared, MarkChequeStale, CancelCheque, GetNextChequeNumber, GetBankStatements, GetBankStatementLines } from '../../../wailsjs/go/main/FinanceService';

  // Design system components
  import PageLayout from '$lib/components/layout/PageLayout.svelte';
  import DataTable from '$lib/components/ui/DataTable.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import StatusBadge from '$lib/components/ui/StatusBadge.svelte';
  import Modal from '$lib/components/layout/Modal.svelte';
  import FormGroup from '$lib/components/ui/FormGroup.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Dropdown from '$lib/components/ui/Dropdown.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import { toast } from '$lib/stores/toasts';
  import { confirm } from '$lib/stores/confirm';
  import { escapeHtml } from '$lib/utils/escapeHtml';
  import { formatNumber } from '$lib/utils/formatters';
  import type { DropdownOption } from '$lib/types/components';

  
  interface Props {
    // Props
    embedded?: boolean;
  }

  let { embedded = false }: Props = $props();

  // Types
  interface BankAccount {
    id: string;
    bank_name: string;
    account_number: string;
    currency: string;
  }

  interface ChequeRegister {
    id: string;
    bank_account_id: string;
    cheque_book_no: string;
    start_number: number;
    end_number: number;
    current_number: number;
    status: string;
    issued_date: any;
  }

  interface OutstandingCheque {
    id: string;
    bank_account_id: string;
    cheque_number: string;
    amount: number;
    currency: string;
    issued_date: any;
    payee_name: string;
    payee_type: string;
    purpose: string;
    status: string;
    is_stale: boolean;
  }

  type ViewTab = 'outstanding' | 'registers' | 'stale';

  // State
  let bankAccounts: BankAccount[] = $state([]);
  let registers: ChequeRegister[] = $state([]);
  let cheques: OutstandingCheque[] = $state([]);
  let staleCheques: OutstandingCheque[] = $state([]);
  let selectedAccountId = $state('');
  let activeTab: ViewTab = $state('outstanding');
  let loading = $state(true);
  let totalOutstanding = $state(0);

  // Modal state
  let showRegisterModal = $state(false);
  let showIssueModal = $state(false);
  let registerLoading = $state(false);
  let issueLoading = $state(false);

  // Wave 9.3 B1e: cheque <-> bank-line linkage picker
  let showClearModal = $state(false);
  let clearingCheque: OutstandingCheque | null = $state(null);
  let clearedDateInput = $state('');
  let clearLineId = $state(''); // '' = clear without linking
  let candidateStatementLines: any[] = $state([]);
  let loadingCandidateLines = $state(false);
  let clearingLoading = $state(false);

  // Register form
  let newBookNo = $state('');
  let newStartNumber = $state('');
  let newEndNumber = $state('');

  // Issue form
  let issueAmount = $state('');
  let issuePayee = $state('');
  let issuePayeeType = $state('SUPPLIER');
  let issuePurpose = $state('');
  let nextChequeNumber = $state('');

  // DataTable columns for cheques
  const chequeColumns = [
    {
      key: 'cheque_number',
      label: 'Cheque #',
      sortable: true,
      width: '100px',
      render: (row: OutstandingCheque) => `<span style="font-family: var(--font-mono); font-weight: 600;">${escapeHtml(row.cheque_number)}</span>`
    },
    {
      key: 'issued_date',
      label: 'Issued',
      sortable: true,
      width: '100px',
      render: (row: OutstandingCheque) => `<span style="font-size: 12px;">${formatDate(row.issued_date)}</span>`
    },
    {
      key: 'payee_name',
      label: 'Payee',
      sortable: true,
      render: (row: OutstandingCheque) => `<span style="font-weight: 500;">${escapeHtml(row.payee_name)}</span>`
    },
    {
      key: 'amount',
      label: 'Amount',
      sortable: true,
      width: '130px',
      align: 'right' as const,
      render: (row: OutstandingCheque) => `<span style="font-family: var(--font-mono); font-weight: 600;">${formatBHD(row.amount)} ${escapeHtml(row.currency || '')}</span>`
    },
    {
      key: 'status',
      label: 'Status',
      sortable: true,
      width: '110px',
      render: (row: OutstandingCheque) => {
        const statusColors: Record<string, string> = {
          'ISSUED': '#3B82F6',
          'PRESENTED': '#F59E0B',
          'CLEARED': '#10B981',
          'STALE': '#EF4444',
          'CANCELLED': '#6B7280',
          'BOUNCED': '#DC2626'
        };
        const color = statusColors[row.status] || '#6B7280';
        return `<span style="color: ${color}; font-weight: 500; font-size: 12px;">${escapeHtml(row.status || '')}</span>`;
      }
    },
    {
      key: 'is_stale',
      label: 'Age',
      width: '80px',
      render: (row: OutstandingCheque) => {
        if (row.is_stale) {
          return `<span style="color: #EF4444; font-size: 11px;">STALE</span>`;
        }
        const days = getDaysOld(row.issued_date);
        return `<span style="color: var(--text-secondary); font-size: 11px;">${days}d</span>`;
      }
    },
    {
      key: 'actions',
      label: '',
      width: '60px',
      type: 'actions' as const
    }
  ];

  // Legal next transitions for a cheque, gated by its current status (Design Law #2 —
  // never a free-jump status grid). Mirrors pkg/finance/cheque.go: Clear/Stale accept
  // ISSUED or PRESENTED; Cancel only accepts ISSUED.
  function chequeActionsFor(cheque: OutstandingCheque): DropdownOption[] {
    const actions: DropdownOption[] = [];
    if (cheque.status === 'ISSUED' || cheque.status === 'PRESENTED') {
      actions.push({ value: 'clear', label: 'Mark Cleared' });
      actions.push({ value: 'stale', label: 'Mark Stale' });
    }
    if (cheque.status === 'ISSUED') {
      actions.push({ value: 'cancel', label: 'Cancel Cheque' });
    }
    return actions;
  }

  async function handleChequeAction(cheque: OutstandingCheque, action: string) {
    if (action === 'clear') return handleMarkCleared(cheque);
    if (action === 'stale') return handleMarkStale(cheque);
    if (action === 'cancel') return handleCancel(cheque);
  }

  // Register columns
  const registerColumns = [
    {
      key: 'cheque_book_no',
      label: 'Book #',
      sortable: true,
      width: '120px',
      render: (row: ChequeRegister) => `<span style="font-family: var(--font-mono); font-weight: 600;">${escapeHtml(String(row.cheque_book_no))}</span>`
    },
    {
      key: 'start_number',
      label: 'Range',
      width: '140px',
      render: (row: ChequeRegister) => `<span style="font-family: var(--font-mono); font-size: 12px;">${escapeHtml(String(row.start_number))} - ${escapeHtml(String(row.end_number))}</span>`
    },
    {
      key: 'current_number',
      label: 'Next',
      width: '100px',
      render: (row: ChequeRegister) => `<span style="font-family: var(--font-mono); color: var(--brand-indigo);">${escapeHtml(String(row.current_number))}</span>`
    },
    {
      key: 'status',
      label: 'Status',
      sortable: true,
      width: '110px',
      render: (row: ChequeRegister) => {
        const statusColors: Record<string, string> = {
          'ACTIVE': '#10B981',
          'EXHAUSTED': '#F59E0B',
          'CANCELLED': '#6B7280'
        };
        const color = statusColors[row.status] || '#6B7280';
        return `<span style="color: ${color}; font-weight: 500; font-size: 12px;">${escapeHtml(row.status || '')}</span>`;
      }
    },
    {
      key: 'usage',
      label: 'Used',
      width: '100px',
      render: (row: ChequeRegister) => {
        const total = row.end_number - row.start_number + 1;
        const used = row.current_number - row.start_number;
        const pct = Math.round((used / total) * 100);
        return `<span style="font-size: 12px;">${used}/${total} (${pct}%)</span>`;
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
      return date.toLocaleDateString('en-GB', { day: '2-digit', month: 'short' });
    } catch {
      return '-';
    }
  }

  function getDaysOld(dateValue: any): number {
    if (!dateValue) return 0;
    try {
      const date = typeof dateValue === 'string' ? new Date(dateValue) : new Date(dateValue);
      const diff = Date.now() - date.getTime();
      return Math.floor(diff / (1000 * 60 * 60 * 24));
    } catch {
      return 0;
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

      await loadAccountData();
    } catch (err) {
      console.error('Failed to load data:', err);
      toast.danger('Failed to load cheque data');
    } finally {
      loading = false;
    }
  }

  async function loadAccountData() {
    if (!selectedAccountId) return;

    try {
      const [registersResult, outstandingResult, staleResult] = await Promise.all([
        GetChequeRegisters(selectedAccountId),
        GetOutstandingCheques(selectedAccountId),
        GetStaleCheques(selectedAccountId)
      ]);

      registers = registersResult || [];
      cheques = outstandingResult?.cheques || [];
      totalOutstanding = outstandingResult?.total || 0;
      staleCheques = staleResult || [];

      // Get next cheque number
      try {
        nextChequeNumber = await GetNextChequeNumber(selectedAccountId);
      } catch {
        nextChequeNumber = 'N/A';
      }
    } catch (err) {
      console.error('Failed to load account data:', err);
    }
  }

  // Actions
  async function handleCreateRegister() {
    if (!newBookNo || !newStartNumber || !newEndNumber) {
      toast.warning('Please fill all fields');
      return;
    }

    registerLoading = true;
    try {
      await CreateChequeRegister(
        selectedAccountId,
        newBookNo,
        parseInt(newStartNumber),
        parseInt(newEndNumber)
      );
      toast.success('Cheque book registered');
      showRegisterModal = false;
      newBookNo = '';
      newStartNumber = '';
      newEndNumber = '';
      await loadAccountData();
    } catch (err: any) {
      console.error('Failed to create register:', err);
      toast.danger(err?.message || 'Failed to register cheque book');
    } finally {
      registerLoading = false;
    }
  }

  async function handleIssueCheque() {
    if (!issueAmount || !issuePayee || !issuePurpose) {
      toast.warning('Please fill all fields');
      return;
    }

    issueLoading = true;
    try {
      await IssueCheque(
        selectedAccountId,
        parseFloat(issueAmount),
        issuePayee,
        issuePayeeType,
        null,
        issuePurpose
      );
      toast.success('Cheque issued successfully');
      showIssueModal = false;
      issueAmount = '';
      issuePayee = '';
      issuePurpose = '';
      await loadAccountData();
    } catch (err: any) {
      console.error('Failed to issue cheque:', err);
      toast.danger(err?.message || 'Failed to issue cheque');
    } finally {
      issueLoading = false;
    }
  }

  // Wave 9.3 B1e: opens a picker of the account's recent bank statement lines
  // so a cleared cheque can carry the real MatchedLineID instead of always
  // going through empty. "Clear without linking" stays available as an
  // explicit fallback when the statement line isn't in yet.
  function todayForInput() {
    return new Date().toISOString().split('T')[0];
  }

  async function openClearModal(cheque: OutstandingCheque) {
    clearingCheque = cheque;
    clearedDateInput = todayForInput();
    clearLineId = '';
    candidateStatementLines = [];
    showClearModal = true;
    loadingCandidateLines = true;
    try {
      const statements = await GetBankStatements(cheque.bank_account_id || selectedAccountId);
      const latest = (statements || [])[0];
      if (latest) {
        const lines = await GetBankStatementLines(latest.id);
        candidateStatementLines = (lines || []).filter((l: any) => Number(l.debit) > 0);
      }
    } catch (err) {
      console.error('Failed to load candidate bank lines:', err);
    } finally {
      loadingCandidateLines = false;
    }
  }

  async function handleConfirmClear() {
    if (!clearingCheque) return;
    clearingLoading = true;
    try {
      await MarkChequeCleared(clearingCheque.cheque_number, clearLineId, new Date(clearedDateInput || todayForInput()));
      toast.success(`Cheque #${clearingCheque.cheque_number} marked as cleared`);
      showClearModal = false;
      clearingCheque = null;
      await loadAccountData();
    } catch (err) {
      console.error('Failed to mark cleared:', err);
      toast.danger('Failed to mark cheque as cleared');
    } finally {
      clearingLoading = false;
    }
  }

  async function handleMarkCleared(cheque: OutstandingCheque) {
    await openClearModal(cheque);
  }

  async function handleMarkStale(cheque: OutstandingCheque) {
    const confirmed = await confirm.ask({
      title: 'Mark Cheque Stale',
      message: `Mark cheque #${cheque.cheque_number} to ${cheque.payee_name} as stale? It will drop out of active outstanding tracking.`,
      confirmLabel: 'Mark Stale',
      variant: 'warning'
    });
    if (!confirmed) return;

    try {
      await MarkChequeStale(cheque.cheque_number);
      toast.success(`Cheque #${cheque.cheque_number} marked as stale`);
      await loadAccountData();
    } catch (err) {
      console.error('Failed to mark stale:', err);
      toast.danger('Failed to mark cheque as stale');
    }
  }

  async function handleCancel(cheque: OutstandingCheque) {
    const r = await confirm.askForReason({
      title: 'Cancel Cheque',
      message: `Cancel cheque #${cheque.cheque_number} (${formatBHD(cheque.amount)} ${cheque.currency || ''}) to ${cheque.payee_name}? This cannot be undone.`,
      confirmLabel: 'Cancel Cheque',
      variant: 'danger',
      reasonLabel: 'Cancellation reason',
      reasonRequired: true
    });
    if (!r.confirmed) return;

    try {
      await CancelCheque(cheque.cheque_number, r.reason);
      toast.success(`Cheque #${cheque.cheque_number} cancelled`);
      await loadAccountData();
    } catch (err) {
      console.error('Failed to cancel:', err);
      toast.danger('Failed to cancel cheque');
    }
  }

  onMount(() => {
    loadData();
  });
</script>

<PageLayout title="Cheque Register" subtitle="Manage cheque books and issuance" {embedded}>
  {#if loading}
    <div class="loading-container">
      <WabiSpinner size="lg" />
      <p>Loading cheque data...</p>
    </div>
  {:else}
    <!-- KPIs -->
    <div class="kpi-grid">
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">Outstanding Total</span>
          <span class="kpi-value primary">{formatBHD(totalOutstanding)} BHD</span>
        </div>
      </Card>
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">Outstanding Cheques</span>
          <span class="kpi-value">{cheques.length}</span>
        </div>
      </Card>
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">Stale Cheques</span>
          <span class="kpi-value warning">{staleCheques.length}</span>
        </div>
      </Card>
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">Next Cheque #</span>
          <span class="kpi-value mono">{nextChequeNumber}</span>
        </div>
      </Card>
    </div>

    <!-- Toolbar -->
    <div class="toolbar">
      <div class="left">
        <select bind:value={selectedAccountId} onchange={loadAccountData} class="account-select">
          {#each bankAccounts as account}
            <option value={account.id}>{account.bank_name} - {account.account_number}</option>
          {/each}
        </select>
        <div class="tabs">
          <button class:active={activeTab === 'outstanding'} onclick={() => activeTab = 'outstanding'}>
            Outstanding ({cheques.length})
          </button>
          <button class:active={activeTab === 'registers'} onclick={() => activeTab = 'registers'}>
            Registers ({registers.length})
          </button>
          <button class:active={activeTab === 'stale'} onclick={() => activeTab = 'stale'}>
            Stale ({staleCheques.length})
          </button>
        </div>
      </div>
      <div class="right">
        <Button variant="secondary" on:click={() => showRegisterModal = true}>
          New Cheque Book
        </Button>
        <Button variant="primary" on:click={() => showIssueModal = true} disabled={nextChequeNumber === 'N/A'}>
          Issue Cheque
        </Button>
      </div>
    </div>

    <!-- Content based on tab -->
    {#if activeTab === 'outstanding'}
      <DataTable
        data={cheques}
        columns={chequeColumns}
        emptyMessage="No outstanding cheques"
        cell={chequeCell}
      />
    {:else if activeTab === 'registers'}
      <DataTable
        data={registers}
        columns={registerColumns}
        emptyMessage="No cheque books registered"
      />
    {:else if activeTab === 'stale'}
      <DataTable
        data={staleCheques}
        columns={chequeColumns}
        emptyMessage="No stale cheques"
        cell={chequeCell}
      />
    {/if}

    {#snippet chequeCell({ column, row, value }: { column: any; row: OutstandingCheque; value: any })}
      {#if column.key === 'actions'}
        {@const actions = chequeActionsFor(row)}
        {#if actions.length > 0}
          <Dropdown options={actions} align="right" on:select={(e) => handleChequeAction(row, e.detail)}>
            <button slot="trigger" class="row-actions-trigger" aria-label={`Actions for cheque ${row.cheque_number}`}>&#8942;</button>
          </Dropdown>
        {:else}
          <span class="no-actions">—</span>
        {/if}
      {:else if column.render}
        {@html column.render(row)}
      {:else}
        {value}
      {/if}
    {/snippet}
  {/if}
</PageLayout>

<!-- Register Modal -->
<Modal bind:open={showRegisterModal} title="Register Cheque Book" size="md">
  <div class="form-grid">
    <FormGroup label="Cheque Book Number">
      <Input bind:value={newBookNo} placeholder="e.g., CB-2026-001" />
    </FormGroup>
    <FormGroup label="Start Number">
      <Input type="number" bind:value={newStartNumber} placeholder="e.g., 100001" />
    </FormGroup>
    <FormGroup label="End Number">
      <Input type="number" bind:value={newEndNumber} placeholder="e.g., 100050" />
    </FormGroup>
  </div>
  {#snippet footer()}
  
      <Button variant="secondary" on:click={() => showRegisterModal = false}>Cancel</Button>
      <Button variant="primary" on:click={handleCreateRegister} loading={registerLoading}>
        Register
      </Button>
    
  {/snippet}
</Modal>

<!-- Issue Modal -->
<Modal bind:open={showIssueModal} title="Issue Cheque" size="md">
  <div class="form-grid">
    <div class="cheque-preview">
      <span class="label">Next Cheque Number</span>
      <span class="number">{nextChequeNumber}</span>
    </div>
    <FormGroup label="Amount (BHD)">
      <Input type="number" step="0.001" bind:value={issueAmount} placeholder="0.000" />
    </FormGroup>
    <FormGroup label="Payee Name">
      <Input bind:value={issuePayee} placeholder="Enter payee name" />
    </FormGroup>
    <FormGroup label="Payee Type">
      <select bind:value={issuePayeeType} class="form-select">
        <option value="SUPPLIER">Supplier</option>
        <option value="EMPLOYEE">Employee</option>
        <option value="OTHER">Other</option>
      </select>
    </FormGroup>
    <FormGroup label="Purpose">
      <Input bind:value={issuePurpose} placeholder="Enter purpose" />
    </FormGroup>
  </div>
  {#snippet footer()}

      <Button variant="secondary" on:click={() => showIssueModal = false}>Cancel</Button>
      <Button variant="primary" on:click={handleIssueCheque} loading={issueLoading}>
        Issue Cheque
      </Button>

  {/snippet}
</Modal>

<!-- Clear Cheque Modal (Wave 9.3 B1e): pick the matched bank statement line, or clear without linking -->
<Modal bind:open={showClearModal} title="Mark Cheque Cleared" size="md">
  {#if clearingCheque}
    <div class="form-grid">
      <p class="clear-summary">
        Cheque #{clearingCheque.cheque_number} ({formatBHD(clearingCheque.amount)} {clearingCheque.currency || ''}) to {clearingCheque.payee_name}
      </p>
      <FormGroup label="Cleared Date">
        <input type="date" bind:value={clearedDateInput} class="form-select" />
      </FormGroup>
      <FormGroup label="Matched Bank Statement Line">
        {#if loadingCandidateLines}
          <div class="loading-container small"><WabiSpinner size="sm" /></div>
        {:else}
          <div class="line-picker">
            <label class="line-picker-option">
              <input type="radio" bind:group={clearLineId} value="" />
              <span>Clear without linking</span>
            </label>
            {#each candidateStatementLines as line}
              <label class="line-picker-option">
                <input type="radio" bind:group={clearLineId} value={line.id} />
                <span>{formatDate(line.transaction_date)} — {line.description} — {formatBHD(line.debit)}</span>
              </label>
            {/each}
            {#if candidateStatementLines.length === 0}
              <p class="line-picker-empty">No recent debit lines found for this account. You can still clear without linking.</p>
            {/if}
          </div>
        {/if}
      </FormGroup>
    </div>
  {/if}
  {#snippet footer()}

      <Button variant="secondary" on:click={() => showClearModal = false}>Cancel</Button>
      <Button variant="primary" on:click={handleConfirmClear} loading={clearingLoading}>
        Mark Cleared
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

  .clear-summary {
    margin: 0;
    font-size: 13px;
    color: var(--text-secondary);
  }

  .line-picker {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    max-height: 220px;
    overflow-y: auto;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    padding: var(--space-2);
  }

  .line-picker-option {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 12px;
    padding: var(--space-1);
    cursor: pointer;
  }

  .line-picker-empty {
    margin: 0;
    padding: var(--space-2);
    font-size: 12px;
    color: var(--text-secondary);
  }

  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--space-4);
    margin-bottom: var(--space-6);
  }

  .kpi {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .kpi-label {
    font-size: 12px;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .kpi-value {
    font-size: 24px;
    font-weight: 600;
  }

  .kpi-value.primary { color: var(--brand-indigo); }
  .kpi-value.warning { color: #F59E0B; }
  .kpi-value.mono { font-family: var(--font-mono); }

  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--space-4);
    padding: var(--space-3);
    background: var(--surface-elevated);
    border-radius: var(--radius-md);
    flex-wrap: wrap;
    gap: var(--space-3);
  }

  .toolbar .left,
  .toolbar .right {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .account-select,
  .form-select {
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: var(--surface-default);
    font-size: 14px;
    min-width: 200px;
  }

  .tabs {
    display: flex;
    gap: var(--space-1);
  }

  .tabs button {
    padding: var(--space-2) var(--space-3);
    border: none;
    background: transparent;
    font-size: 13px;
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all 0.2s;
  }

  .tabs button:hover {
    background: var(--surface-default);
  }

  .tabs button.active {
    background: var(--brand-indigo);
    color: white;
  }

  .form-grid {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .cheque-preview {
    padding: var(--space-3);
    background: var(--surface-elevated);
    border-radius: var(--radius-sm);
    text-align: center;
  }

  .cheque-preview .label {
    display: block;
    font-size: 11px;
    color: var(--text-secondary);
    margin-bottom: var(--space-1);
  }

  .cheque-preview .number {
    font-family: var(--font-mono);
    font-size: 24px;
    font-weight: 700;
    color: var(--brand-indigo);
  }

  @media (max-width: 900px) {
    .kpi-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  .row-actions-trigger {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    padding: 0;
    border: none;
    background: transparent;
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    font-size: 16px;
    line-height: 1;
    cursor: pointer;
    transition: background 0.15s;
  }

  .row-actions-trigger:hover {
    background: var(--surface-elevated);
    color: var(--text-primary);
  }

  .no-actions {
    color: var(--text-secondary);
    font-size: 12px;
  }
</style>
