<script lang="ts">
  /**
   * FXRevaluationScreen - Foreign Exchange Revaluation
   *
   * Features:
   * - FX rate management and history
   * - Unrealized gain/loss calculation
   * - Multi-currency exposure report
   * - Batch revaluation for all foreign currency accounts
   * - Post/reverse revaluation entries
   *
   * Design System: Wabi-Sabi minimalism x Bloomberg data density
   */

  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  // Wails API imports
  import {
    GetActiveBankAccounts } from '../../../wailsjs/go/main/App';
import { GetLatestFXRate, CreateFXRate, GetFXRevaluations, CalculateFXRevaluation, RevalueAllForeignAccounts, PostFXRevaluation, GetFXExposureReport, GetFXGainLossSummary } from '../../../wailsjs/go/main/FinanceService';
  import { escapeHtml } from '$lib/utils/escapeHtml';
  import { formatNumber } from '$lib/utils/formatters';

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
  import { confirm } from '$lib/stores/confirm';
  import { currentUser } from '$lib/stores/authContext';

  
  interface Props {
    // Props
    embedded?: boolean;
  }

  let { embedded = false }: Props = $props();

  // Types - aligned with Wails CompanyBankAccount
  interface BankAccount {
    id: string;
    bank_name: string;
    account_name?: string;
    account_number: string;
    currency: string;
    is_active?: boolean;
  }

  interface FXRate {
    id: string;
    from_currency: string;
    to_currency: string;
    rate_date: any;
    rate: number;
    source: string;
  }

  interface FXRevaluation {
    id: string;
    bank_account_id: string;
    revaluation_date: any;
    foreign_currency: string;
    foreign_balance: number;
    previous_rate: number;
    previous_bhd: number;
    current_rate: number;
    current_bhd: number;
    gain_loss_bhd: number;
    is_posted: boolean;
    posted_by: string;
    posted_at: any;
  }

  interface ExposureReport {
    currency: string;
    account_count: number;
    total_foreign: number;
    current_rate: number;
    total_bhd: number;
    unrealized_gain: number;
    percent_exposure: number;
  }

  type ViewTab = 'exposure' | 'revaluations' | 'rates';

  // State - using any[] to accommodate Wails-generated classes
  let bankAccounts: any[] = [];
  let foreignAccounts: any[] = $state([]);
  let revaluations: any[] = $state([]);
  let exposureReport: any[] = $state([]);
  let totalExposure = $state(0);
  let selectedAccountId = $state('');
  let activeTab: ViewTab = $state('exposure');
  let loading = $state(true);

  // Summary
  let ytdGain = $state(0);
  let ytdLoss = $state(0);
  let netGainLoss = $state(0);
  let unpostedNet = $state(0);

  // Modal state
  let showRateModal = $state(false);
  let showRevalModal = false;
  let modalLoading = $state(false);

  // Rate form
  let rateCurrency = $state('USD');
  let rateValue = $state('');
  let rateSource = $state('CBB');
  let rateModalCurrentRate = $state<number | null>(null);
  let rateModalLoadingCurrent = $state(false);

  // As-of date for revaluation runs (B3b) — defaults to today, user-adjustable.
  let asOfDate = $state(new Date().toISOString().slice(0, 10));

  // Details view for a single revaluation (B3a) — row-click opens this
  // read-only panel instead of posting directly.
  let showDetailsModal = $state(false);
  let selectedReval: FXRevaluation | null = $state(null);
  let posting = $state(false);

  // Common currencies
  const currencies = ['USD', 'EUR', 'GBP', 'SAR', 'AED', 'KWD', 'QAR', 'OMR'];

  // DataTable columns for revaluations
  const revalColumns = [
    {
      key: 'revaluation_date',
      label: 'Date',
      sortable: true,
      width: '100px',
      render: (row: FXRevaluation) => `<span style="font-family: var(--font-mono); font-size: 12px;">${formatDate(row.revaluation_date)}</span>`
    },
    {
      key: 'foreign_currency',
      label: 'Currency',
      width: '80px',
      render: (row: FXRevaluation) => `<span style="font-weight: 600;">${escapeHtml(row.foreign_currency || '')}</span>`
    },
    {
      key: 'foreign_balance',
      label: 'Foreign Bal',
      width: '120px',
      align: 'right' as const,
      render: (row: FXRevaluation) => `<span style="font-family: var(--font-mono);">${formatAmount(row.foreign_balance, row.foreign_currency)}</span>`
    },
    {
      key: 'current_rate',
      label: 'Rate',
      width: '90px',
      align: 'right' as const,
      render: (row: FXRevaluation) => `<span style="font-family: var(--font-mono); font-size: 12px;">${row.current_rate.toFixed(6)}</span>`
    },
    {
      key: 'current_bhd',
      label: 'BHD Value',
      width: '120px',
      align: 'right' as const,
      render: (row: FXRevaluation) => `<span style="font-family: var(--font-mono);">${formatBHD(row.current_bhd)}</span>`
    },
    {
      key: 'gain_loss_bhd',
      label: 'Gain/Loss',
      sortable: true,
      width: '110px',
      align: 'right' as const,
      render: (row: FXRevaluation) => {
        const color = row.gain_loss_bhd >= 0 ? '#10B981' : '#EF4444';
        const sign = row.gain_loss_bhd >= 0 ? '+' : '';
        return `<span style="font-family: var(--font-mono); color: ${color}; font-weight: 600;">${sign}${formatBHD(row.gain_loss_bhd)}</span>`;
      }
    },
    {
      key: 'is_posted',
      label: 'Status',
      width: '90px',
      render: (row: FXRevaluation) => {
        if (row.is_posted) {
          return `<span style="color: #10B981; font-size: 12px;">Posted</span>`;
        }
        return `<span style="color: #F59E0B; font-size: 12px;">Pending</span>`;
      }
    }
  ];

  // Exposure columns
  const exposureColumns = [
    {
      key: 'currency',
      label: 'Currency',
      width: '80px',
      render: (row: ExposureReport) => `<span style="font-weight: 600;">${escapeHtml(row.currency || '')}</span>`
    },
    {
      key: 'account_count',
      label: 'Accounts',
      width: '80px',
      align: 'center' as const
    },
    {
      key: 'total_foreign',
      label: 'Foreign Balance',
      width: '130px',
      align: 'right' as const,
      render: (row: ExposureReport) => `<span style="font-family: var(--font-mono);">${formatAmount(row.total_foreign, row.currency)}</span>`
    },
    {
      key: 'current_rate',
      label: 'Rate',
      width: '90px',
      align: 'right' as const,
      render: (row: ExposureReport) => `<span style="font-family: var(--font-mono); font-size: 12px;">${row.current_rate.toFixed(6)}</span>`
    },
    {
      key: 'total_bhd',
      label: 'BHD Value',
      width: '130px',
      align: 'right' as const,
      render: (row: ExposureReport) => `<span style="font-family: var(--font-mono); font-weight: 600;">${formatBHD(row.total_bhd)}</span>`
    },
    {
      key: 'percent_exposure',
      label: 'Exposure %',
      width: '100px',
      align: 'right' as const,
      render: (row: ExposureReport) => `<span style="font-size: 13px;">${row.percent_exposure.toFixed(1)}%</span>`
    },
    {
      key: 'unrealized_gain',
      label: 'Unrealized',
      width: '110px',
      align: 'right' as const,
      render: (row: ExposureReport) => {
        const color = row.unrealized_gain >= 0 ? '#10B981' : '#EF4444';
        return `<span style="font-family: var(--font-mono); color: ${color};">${formatBHD(row.unrealized_gain)}</span>`;
      }
    }
  ];

  // Formatters
  function formatBHD(value: number): string {
    return formatNumber(value || 0, 3);
  }

  function formatAmount(value: number, currency: string): string {
    const decimals = currency === 'BHD' || currency === 'KWD' || currency === 'OMR' ? 3 : 2;
    return formatNumber(value || 0, decimals);
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

  // Data loading
  async function loadData() {
    loading = true;
    try {
      const accounts = await GetActiveBankAccounts();
      bankAccounts = accounts || [];
      foreignAccounts = bankAccounts.filter((a: BankAccount) => a.currency !== 'BHD');

      if (foreignAccounts.length > 0 && !selectedAccountId) {
        selectedAccountId = foreignAccounts[0].id;
      }

      await Promise.all([
        loadExposure(),
        loadRevaluations(),
        loadSummary()
      ]);
    } catch (err) {
      console.error('Failed to load data:', err);
      toast.danger('Failed to load FX data');
    } finally {
      loading = false;
    }
  }

  async function loadExposure() {
    try {
      const result = await GetFXExposureReport();
      exposureReport = result?.reports || [];
      totalExposure = result?.total || 0;
    } catch (err) {
      console.error('Failed to load exposure:', err);
    }
  }

  async function loadRevaluations() {
    if (!selectedAccountId) return;
    try {
      const result = await GetFXRevaluations(selectedAccountId);
      revaluations = result || [];
    } catch (err) {
      console.error('Failed to load revaluations:', err);
    }
  }

  async function loadSummary() {
    try {
      const year = new Date().getFullYear();
      const summary = await GetFXGainLossSummary(year);
      ytdGain = summary?.total_gain || 0;
      ytdLoss = summary?.total_loss || 0;
      netGainLoss = summary?.net_gain_loss || 0;
      unpostedNet = summary?.unposted_net || 0;
    } catch (err) {
      console.error('Failed to load summary:', err);
    }
  }

  // Actions
  function asOfDateObj(): Date {
    // Parsed as local midnight, not UTC, so the picked calendar day survives.
    const [y, m, d] = asOfDate.split('-').map(Number);
    return new Date(y, (m || 1) - 1, d || 1);
  }

  async function loadRateModalCurrentRate(currency: string) {
    rateModalLoadingCurrent = true;
    rateModalCurrentRate = null;
    try {
      const rate = await GetLatestFXRate(currency, 'BHD');
      rateModalCurrentRate = rate?.rate ?? null;
    } catch (err) {
      // No rate on file yet for this pair — leave as null, not an error to the user.
      rateModalCurrentRate = null;
    } finally {
      rateModalLoadingCurrent = false;
    }
  }

  // Re-fetch the current rate whenever the modal is open and the currency changes.
  $effect(() => {
    if (showRateModal) {
      loadRateModalCurrentRate(rateCurrency);
    }
  });

  async function handleAddRate() {
    if (!rateValue) {
      toast.warning('Please enter a rate');
      return;
    }

    modalLoading = true;
    try {
      await CreateFXRate(rateCurrency, 'BHD', parseFloat(rateValue), new Date(), rateSource);
      toast.success(`${rateCurrency}/BHD rate updated`);
      showRateModal = false;
      rateValue = '';
      await loadExposure();
    } catch (err: any) {
      console.error('Failed to add rate:', err);
      toast.danger(err?.message || 'Failed to add rate');
    } finally {
      modalLoading = false;
    }
  }

  async function handleRevalue() {
    if (!selectedAccountId) return;

    modalLoading = true;
    try {
      await CalculateFXRevaluation(selectedAccountId, asOfDateObj());
      toast.success('Revaluation calculated');
      await loadRevaluations();
      await loadSummary();
    } catch (err: any) {
      console.error('Failed to revalue:', err);
      toast.danger(err?.message || 'Failed to calculate revaluation');
    } finally {
      modalLoading = false;
    }
  }

  async function handleBatchRevalue() {
    const count = foreignAccounts.length;
    if (count === 0) {
      toast.warning('No foreign currency accounts to revalue');
      return;
    }
    if (!(await confirm.ask({
      title: 'Revalue All Foreign Accounts',
      message: `Revalue ${count} foreign currency account${count === 1 ? '' : 's'} as of ${asOfDate}? This calculates unrealized gain/loss for each — it does not post them.`,
      confirmLabel: 'Revalue All',
      variant: 'primary'
    }))) return;

    modalLoading = true;
    try {
      const result = await RevalueAllForeignAccounts(asOfDateObj());
      const count2 = (result?.revaluations || []).length;
      const total = result?.total || 0;
      toast.success(`Revalued ${count2} accounts. Net: ${formatBHD(total)} BHD`);
      await Promise.all([loadExposure(), loadRevaluations(), loadSummary()]);
    } catch (err: any) {
      console.error('Batch revalue failed:', err);
      toast.danger(err?.message || 'Batch revaluation failed');
    } finally {
      modalLoading = false;
    }
  }

  // Row-click opens a read-only details view (B3a) — it must never post directly.
  function openDetails(row: FXRevaluation) {
    selectedReval = row;
    showDetailsModal = true;
  }

  function closeDetails() {
    showDetailsModal = false;
    selectedReval = null;
  }

  function accountLabel(bankAccountId: string): string {
    const acc = bankAccounts.find((a: BankAccount) => a.id === bankAccountId);
    return acc ? `${acc.bank_name} (${acc.currency})` : bankAccountId;
  }

  async function confirmAndPost(reval: FXRevaluation | null) {
    if (!reval) return;

    // Identity is resolved server-side (B2); block here only if we don't even
    // know who is operating the app, mirroring SupplierInvoicesScreen (Article III.4).
    const operatorId = $currentUser?.id;
    const operatorLabel = $currentUser?.full_name || $currentUser?.username || operatorId;
    if (!operatorId) {
      toast.danger('Cannot post: no authenticated user found. Please sign in again.');
      return;
    }

    const sign = reval.gain_loss_bhd >= 0 ? '+' : '';
    if (!(await confirm.ask({
      title: 'Post FX Revaluation',
      message: `Post this FX revaluation of ${sign}${formatBHD(reval.gain_loss_bhd)} BHD for ${accountLabel(reval.bank_account_id)} as of ${formatDate(reval.revaluation_date)} as ${operatorLabel}? This marks it posted.`,
      confirmLabel: 'Post',
      variant: 'primary'
    }))) return;

    posting = true;
    try {
      // Client-supplied id is advisory only — the server re-resolves identity.
      await PostFXRevaluation(reval.id, operatorId);
      toast.success('Revaluation posted');
      closeDetails();
      await loadRevaluations();
      await loadSummary();
    } catch (err: any) {
      console.error('Failed to post:', err);
      toast.danger(err?.message || 'Failed to post');
    } finally {
      posting = false;
    }
  }

  onMount(() => {
    loadData();
  });
</script>

<PageLayout title="FX Revaluation" subtitle="Foreign exchange exposure and revaluation" {embedded}>
  {#if loading}
    <div class="loading-container">
      <WabiSpinner size="lg" />
      <p>Loading FX data...</p>
    </div>
  {:else}
    <!-- KPIs -->
    <div class="kpi-grid">
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">FX Exposure</span>
          <span class="kpi-value primary">{formatBHD(totalExposure)} BHD</span>
        </div>
      </Card>
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">YTD Gain</span>
          <span class="kpi-value success">+{formatBHD(ytdGain)}</span>
        </div>
      </Card>
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">YTD Loss</span>
          <span class="kpi-value danger-text">{formatBHD(ytdLoss)}</span>
        </div>
      </Card>
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">Net (Unposted)</span>
          <span class="kpi-value" class:success={netGainLoss >= 0} class:danger-text={netGainLoss < 0}>
            {formatBHD(netGainLoss)} ({formatBHD(unpostedNet)})
          </span>
        </div>
      </Card>
    </div>

    <!-- Toolbar -->
    <div class="toolbar">
      <div class="left">
        <div class="tabs">
          <button class:active={activeTab === 'exposure'} onclick={() => activeTab = 'exposure'}>
            Exposure
          </button>
          <button class:active={activeTab === 'revaluations'} onclick={() => activeTab = 'revaluations'}>
            Revaluations
          </button>
        </div>
        {#if activeTab === 'revaluations'}
          <select bind:value={selectedAccountId} onchange={loadRevaluations} class="account-select">
            {#each foreignAccounts as account}
              <option value={account.id}>{account.bank_name} ({account.currency})</option>
            {/each}
          </select>
        {/if}
      </div>
      <div class="right">
        <label class="as-of-group">
          <span class="as-of-label">As of</span>
          <input type="date" bind:value={asOfDate} class="date-input" />
        </label>
        <Button variant="secondary" on:click={() => showRateModal = true}>
          Update Rate
        </Button>
        <Button variant="primary" on:click={handleBatchRevalue} loading={modalLoading}>
          Revalue All
        </Button>
      </div>
    </div>

    <!-- Content -->
    {#if activeTab === 'exposure'}
      <DataTable
        data={exposureReport}
        columns={exposureColumns}
        emptyMessage="No foreign currency exposure"
      />
    {:else if activeTab === 'revaluations'}
      <div class="reval-actions">
        <Button variant="secondary" size="sm" on:click={handleRevalue} disabled={!selectedAccountId}>
          Calculate for Selected Account (as of {asOfDate})
        </Button>
      </div>
      <DataTable
        data={revaluations}
        columns={revalColumns}
        emptyMessage="No revaluations for this account"
        onRowClick={(row) => openDetails(row)}
      />
    {/if}
  {/if}
</PageLayout>

<!-- Rate Modal -->
<Modal bind:open={showRateModal} title="Update FX Rate" size="sm">
  <div class="form-grid">
    <FormGroup label="Currency">
      <select bind:value={rateCurrency} class="form-select">
        {#each currencies as c}
          <option value={c}>{c}/BHD</option>
        {/each}
      </select>
    </FormGroup>
    <div class="rate-compare">
      <span class="rate-compare-label">Current rate</span>
      <span class="rate-compare-value">
        {#if rateModalLoadingCurrent}
          loading&hellip;
        {:else if rateModalCurrentRate !== null}
          {rateModalCurrentRate.toFixed(6)}
        {:else}
          no rate on file
        {/if}
      </span>
    </div>
    <FormGroup label="New Rate">
      <Input type="number" step="0.000001" bind:value={rateValue} placeholder="0.000000" />
    </FormGroup>
    {#if rateValue && rateModalCurrentRate !== null}
      <p class="rate-compare-preview">
        {rateModalCurrentRate.toFixed(6)} &rarr; {parseFloat(rateValue).toFixed(6)}
      </p>
    {/if}
    <FormGroup label="Source">
      <select bind:value={rateSource} class="form-select">
        <option value="CBB">CBB (Central Bank)</option>
        <option value="MANUAL">Manual Entry</option>
        <option value="API">API Feed</option>
      </select>
    </FormGroup>
  </div>
  {#snippet footer()}

      <Button variant="secondary" on:click={() => showRateModal = false}>Cancel</Button>
      <Button variant="primary" on:click={handleAddRate} loading={modalLoading}>
        Save Rate
      </Button>

  {/snippet}
</Modal>

<!-- Revaluation Details Modal (read-only; posting is an explicit action here, B3a) -->
<Modal bind:open={showDetailsModal} title="Revaluation Detail" size="sm" on:close={closeDetails}>
  {#if selectedReval}
    <div class="detail-grid">
      <div class="detail-row">
        <span class="detail-label">Account</span>
        <span class="detail-value">{accountLabel(selectedReval.bank_account_id)}</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">As of</span>
        <span class="detail-value">{formatDate(selectedReval.revaluation_date)}</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">Foreign balance</span>
        <span class="detail-value">{formatAmount(selectedReval.foreign_balance, selectedReval.foreign_currency)} {selectedReval.foreign_currency}</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">Previous rate &rarr; current rate</span>
        <span class="detail-value">{selectedReval.previous_rate.toFixed(6)} &rarr; {selectedReval.current_rate.toFixed(6)}</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">BHD value</span>
        <span class="detail-value">{formatBHD(selectedReval.current_bhd)} BHD</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">Gain / loss</span>
        <span class="detail-value" class:success={selectedReval.gain_loss_bhd >= 0} class:danger-text={selectedReval.gain_loss_bhd < 0}>
          {selectedReval.gain_loss_bhd >= 0 ? '+' : ''}{formatBHD(selectedReval.gain_loss_bhd)} BHD
        </span>
      </div>
      <div class="detail-row">
        <span class="detail-label">Status</span>
        <span class="detail-value">
          {#if selectedReval.is_posted}
            Posted by {selectedReval.posted_by} on {formatDate(selectedReval.posted_at)}
          {:else}
            Pending — not yet posted
          {/if}
        </span>
      </div>
      {#if !selectedReval.is_posted}
        <div class="detail-row">
          <span class="detail-label">Operator</span>
          <span class="detail-value">{$currentUser?.full_name || $currentUser?.username || 'Unknown — sign in required'}</span>
        </div>
      {/if}
    </div>
  {/if}
  {#snippet footer()}
    <Button variant="secondary" on:click={closeDetails}>Close</Button>
    {#if selectedReval && !selectedReval.is_posted}
      <Button variant="primary" on:click={() => confirmAndPost(selectedReval)} loading={posting}>
        Post Revaluation
      </Button>
    {/if}
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
    font-family: var(--font-mono);
  }

  .kpi-value.primary { color: var(--brand-indigo); }
  .kpi-value.success { color: #10B981; }
  .kpi-value.danger-text { color: #EF4444; }

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

  .account-select,
  .form-select {
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: var(--surface-default);
    font-size: 14px;
    min-width: 200px;
  }

  .reval-actions {
    margin-bottom: var(--space-3);
  }

  .form-grid {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .as-of-group {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .as-of-label {
    font-size: 12px;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .date-input {
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: var(--surface-default);
    font-size: 14px;
    font-family: var(--font-mono);
  }

  .rate-compare {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    padding: var(--space-2) var(--space-3);
    background: var(--surface-elevated);
    border-radius: var(--radius-sm);
  }

  .rate-compare-label {
    font-size: 12px;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .rate-compare-value {
    font-family: var(--font-mono);
    font-size: 14px;
    font-weight: 600;
  }

  .rate-compare-preview {
    margin: 0;
    font-family: var(--font-mono);
    font-size: 13px;
    color: var(--text-secondary);
  }

  .detail-grid {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .detail-row {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: var(--space-3);
    padding-bottom: var(--space-2);
    border-bottom: 1px solid var(--border-subtle);
  }

  .detail-label {
    font-size: 12px;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    white-space: nowrap;
  }

  .detail-value {
    font-family: var(--font-mono);
    font-size: 14px;
    text-align: right;
  }

  .detail-value.success { color: #10B981; }
  .detail-value.danger-text { color: #EF4444; }

  @media (max-width: 900px) {
    .kpi-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
</style>
