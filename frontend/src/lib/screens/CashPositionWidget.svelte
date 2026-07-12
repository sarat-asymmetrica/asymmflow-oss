<script lang="ts">
  /**
   * CashPositionWidget - Real-time Cash Position Display
   *
   * Features:
   * - Total cash across all bank accounts
   * - Breakdown by account with currency conversion
   * - Outstanding cheques deduction
   * - Deposits in transit addition
   * - Mini sparkline for trend (if data available)
   *
   * Design System: Wabi-Sabi minimalism x Bloomberg compactness
   */

  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  // Wails API imports
  import {
    GetCashPosition } from '../../../wailsjs/go/main/App';
import { GetCashPositionByAccount, GetActiveBankAccounts, GetOutstandingCheques } from '../../../wailsjs/go/main/FinanceService';
import { GetDepositsInTransit } from '../../../wailsjs/go/main/InfraService';

  import Card from '$lib/components/ui/Card.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import { formatNumber } from '$lib/utils/formatters';

  
  interface Props {
    // Props
    compact?: boolean;
    showBreakdown?: boolean;
  }

  let { compact = false, showBreakdown = true }: Props = $props();

  // Types - aligned with CompanyBankAccount from Wails (no balance field, calculated from cash position)
  interface BankAccount {
    id: string;
    bank_name: string;
    account_name?: string;
    account_number: string;
    currency: string;
    is_active?: boolean;
  }

  interface AccountPosition {
    account: BankAccount;
    balance: number;
    outstanding_cheques: number;
    deposits_in_transit: number;
    adjusted_balance: number;
  }

  // State
  let loading = $state(true);
  let totalCash = $state(0);
  let positions: AccountPosition[] = $state([]);
  let lastUpdated: Date | null = $state(null);

  // Formatters
  function formatBHD(value: number): string {
    return formatNumber(value || 0, 3);
  }

  function formatCurrency(value: number, currency: string): string {
    const decimals = currency === 'BHD' ? 3 : 2;
    return formatNumber(value || 0, decimals);
  }

  function formatTime(date: Date): string {
    return date.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
  }

  // Data loading
  async function loadData() {
    loading = true;
    try {
      const [accounts, cashResult] = await Promise.all([
        GetActiveBankAccounts(),
        GetCashPosition()
      ]);

      totalCash = cashResult?.total_bhd || 0;
      positions = [];

      // Get by_account mapping from cash position (keyed by account ID or name)
      const byAccount = cashResult?.by_account || {};

      for (const account of accounts || []) {
        if (!account.is_active) continue;

        let outstandingCheques = 0;
        let depositsInTransit = 0;

        try {
          const [chequesResult, depositsResult] = await Promise.all([
            GetOutstandingCheques(account.id),
            GetDepositsInTransit(account.id)
          ]);
          outstandingCheques = chequesResult?.total || 0;
          depositsInTransit = depositsResult?.total || 0;
        } catch (e) {
          // Ignore errors for individual accounts
        }

        // Balance from cash position, keyed by account ID or account_number
        const accountBalance = byAccount[account.id] || byAccount[account.account_number] || 0;
        const adjustedBalance = accountBalance - outstandingCheques + depositsInTransit;

        positions.push({
          account,
          balance: accountBalance,
          outstanding_cheques: outstandingCheques,
          deposits_in_transit: depositsInTransit,
          adjusted_balance: adjustedBalance
        });
      }

      lastUpdated = new Date();
    } catch (err) {
      console.error('Failed to load cash position:', err);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadData();
    // Refresh every 5 minutes
    const interval = setInterval(loadData, 5 * 60 * 1000);
    return () => clearInterval(interval);
  });
</script>

<div class="cash-widget" class:compact transition:fade={{ duration: 200 }}>
  {#if loading}
    <div class="loading">
      <WabiSpinner size="sm" />
    </div>
  {:else}
    <!-- Total Cash Header -->
    <div class="total-section">
      <div class="total-label">
        <span class="label">Total Cash Position</span>
        {#if lastUpdated}
          <span class="updated">Updated {formatTime(lastUpdated)}</span>
        {/if}
      </div>
      <div class="total-value">
        <span class="currency">BHD</span>
        <span class="amount">{formatBHD(totalCash)}</span>
      </div>
    </div>

    <!-- Account Breakdown -->
    {#if showBreakdown && positions.length > 0}
      <div class="breakdown">
        {#each positions as pos}
          <div class="account-row">
            <div class="account-info">
              <span class="bank-name">{pos.account.bank_name}</span>
              <span class="account-number">{pos.account.account_number}</span>
            </div>
            <div class="account-values">
              <div class="balance-line">
                <span class="label">Book Balance</span>
                <span class="value">{formatCurrency(pos.balance, pos.account.currency)} {pos.account.currency}</span>
              </div>
              {#if pos.outstanding_cheques > 0}
                <div class="adjustment-line negative">
                  <span class="label">Outstanding Cheques</span>
                  <span class="value">-{formatCurrency(pos.outstanding_cheques, pos.account.currency)}</span>
                </div>
              {/if}
              {#if pos.deposits_in_transit > 0}
                <div class="adjustment-line positive">
                  <span class="label">Deposits in Transit</span>
                  <span class="value">+{formatCurrency(pos.deposits_in_transit, pos.account.currency)}</span>
                </div>
              {/if}
              <div class="adjusted-line">
                <span class="label">Available</span>
                <span class="value">{formatCurrency(pos.adjusted_balance, pos.account.currency)}</span>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Quick Stats -->
    {#if !compact}
      <div class="quick-stats">
        <div class="stat">
          <span class="stat-label">Accounts</span>
          <span class="stat-value">{positions.length}</span>
        </div>
        <div class="stat">
          <span class="stat-label">O/S Cheques</span>
          <span class="stat-value negative">
            {formatBHD(positions.reduce((sum, p) => sum + p.outstanding_cheques, 0))}
          </span>
        </div>
        <div class="stat">
          <span class="stat-label">In Transit</span>
          <span class="stat-value positive">
            {formatBHD(positions.reduce((sum, p) => sum + p.deposits_in_transit, 0))}
          </span>
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .cash-widget {
    background: var(--surface-default);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: var(--space-4);
  }

  .cash-widget.compact {
    padding: var(--space-3);
  }

  .loading {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4);
  }

  .total-section {
    margin-bottom: var(--space-4);
    padding-bottom: var(--space-3);
    border-bottom: 1px solid var(--border-subtle);
  }

  .total-label {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--space-2);
  }

  .total-label .label {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-secondary);
  }

  .total-label .updated {
    font-size: 11px;
    color: var(--text-tertiary);
  }

  .total-value {
    display: flex;
    align-items: baseline;
    gap: var(--space-2);
  }

  .total-value .currency {
    font-size: 14px;
    color: var(--text-secondary);
    font-weight: 500;
  }

  .total-value .amount {
    font-size: 32px;
    font-weight: 700;
    font-family: var(--font-mono);
    color: var(--brand-indigo);
    letter-spacing: -0.5px;
  }

  .compact .total-value .amount {
    font-size: 24px;
  }

  .breakdown {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    margin-bottom: var(--space-4);
  }

  .account-row {
    padding: var(--space-3);
    background: var(--surface-elevated);
    border-radius: var(--radius-sm);
  }

  .account-info {
    display: flex;
    justify-content: space-between;
    margin-bottom: var(--space-2);
  }

  .bank-name {
    font-weight: 600;
    font-size: 13px;
    color: var(--text-primary);
  }

  .account-number {
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--text-secondary);
  }

  .account-values {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .balance-line,
  .adjustment-line,
  .adjusted-line {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
  }

  .balance-line .label,
  .adjustment-line .label {
    color: var(--text-secondary);
  }

  .balance-line .value {
    font-family: var(--font-mono);
    color: var(--text-primary);
  }

  .adjustment-line.negative .value {
    color: #EF4444;
    font-family: var(--font-mono);
  }

  .adjustment-line.positive .value {
    color: #10B981;
    font-family: var(--font-mono);
  }

  .adjusted-line {
    margin-top: var(--space-1);
    padding-top: var(--space-1);
    border-top: 1px dashed var(--border-subtle);
    font-weight: 600;
  }

  .adjusted-line .label {
    color: var(--text-primary);
  }

  .adjusted-line .value {
    font-family: var(--font-mono);
    color: var(--brand-indigo);
  }

  .quick-stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--space-3);
    padding-top: var(--space-3);
    border-top: 1px solid var(--border-subtle);
  }

  .stat {
    text-align: center;
  }

  .stat-label {
    display: block;
    font-size: 11px;
    color: var(--text-tertiary);
    margin-bottom: var(--space-1);
  }

  .stat-value {
    font-family: var(--font-mono);
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .stat-value.negative {
    color: #EF4444;
  }

  .stat-value.positive {
    color: #10B981;
  }
</style>
