<script lang="ts">
  import { run } from 'svelte/legacy';

  /**
   * AuditTrailViewer - Bank Reconciliation Audit Trail
   *
   * Features:
   * - Complete audit history for all reconciliation actions
   * - Filter by statement, action type, date range
   * - Action reversal capability
   * - Export audit report
   * - Automatic vs Manual action distinction
   *
   * Design System: Wabi-Sabi minimalism x Bloomberg data density
   */

  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  // Wails API imports
  import {
    GetBankStatements } from '../../../wailsjs/go/main/App';
import { GetActiveBankAccounts, GetAuditTrail, GetAuditTrailByDateRange, ReverseAction } from '../../../wailsjs/go/main/FinanceService';

  // Design system components
  import PageLayout from '$lib/components/layout/PageLayout.svelte';
  import DataTable from '$lib/components/ui/DataTable.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Modal from '$lib/components/layout/Modal.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import { toast } from '$lib/stores/toasts';
  import { confirm } from '$lib/stores/confirm';
  import { currentUser } from '$lib/stores/authContext';
  import { escapeHtml } from '$lib/utils/escapeHtml';

  
  interface Props {
    // Props
    embedded?: boolean;
    statementId?: string; // Pre-filter by statement
  }

  let { embedded = false, statementId = '' }: Props = $props();

  // Types
  interface BankAccount {
    id: string;
    bank_name: string;
    account_number: string;
  }

  interface BankStatement {
    id: string;
    statement_number: string;
    period_end: any;
  }

  // AuditLogEntry - compatible with BankReconciliationAuditLog from Wails
  interface AuditLogEntry {
    id: string;
    created_at?: any;
    updated_at?: any;
    version?: number;
    created_by?: string;
    bank_statement_id: string;
    bank_statement_line_id?: string;
    action: string;
    action_detail: string;
    performed_by: string;
    performed_at: any;
    is_automatic: boolean;
    confidence_score: number;
    reason: string;
    is_reversed: boolean;
    reversed_by: string;
    reversed_at?: any;
    reversal_reason: string;
  }

  type ActionFilter = 'all' | 'IMPORT' | 'MATCH' | 'UNMATCH' | 'SPLIT' | 'CATEGORIZE' | 'RECONCILE' | 'VERIFY';

  // State
  let bankAccounts: BankAccount[] = $state([]);
  let statements: BankStatement[] = $state([]);
  let auditLogs: AuditLogEntry[] = $state([]);
  let filteredLogs: AuditLogEntry[] = $state([]);
  let selectedAccountId = $state('');
  let selectedStatementId = $state('');
  let actionFilter: ActionFilter = $state('all');
  let showReversed = $state(false);
  let loading = $state(true);

  // Modal state — Wave 9.3 B3(c): row-click opens a read-only details view;
  // "Reverse" is a distinct, explicitly-clicked guarded action inside it, not
  // something a stray row-click can trigger.
  let showDetailsModal = $state(false);
  let reverseLoading = $state(false);
  let selectedLog: AuditLogEntry | null = $state(null);

  // Stats
  let totalActions = $derived(auditLogs.length);
  let autoActions = $derived(auditLogs.filter(l => l.is_automatic).length);
  let manualActions = $derived(auditLogs.filter(l => !l.is_automatic).length);
  let reversedActions = $derived(auditLogs.filter(l => l.is_reversed).length);

  // Action types for filter
  const actionTypes: ActionFilter[] = ['all', 'IMPORT', 'MATCH', 'UNMATCH', 'SPLIT', 'CATEGORIZE', 'RECONCILE', 'VERIFY'];

  // DataTable columns
  const columns = [
    {
      key: 'performed_at',
      label: 'Timestamp',
      sortable: true,
      width: '150px',
      render: (row: AuditLogEntry) => `<span style="font-family: var(--font-mono); font-size: 11px;">${formatDateTime(row.performed_at)}</span>`
    },
    {
      key: 'action',
      label: 'Action',
      sortable: true,
      width: '110px',
      render: (row: AuditLogEntry) => {
        const actionColors: Record<string, string> = {
          'IMPORT': '#3B82F6',
          'MATCH': '#10B981',
          'UNMATCH': '#F59E0B',
          'SPLIT': '#8B5CF6',
          'CATEGORIZE': '#6366F1',
          'RECONCILE': '#059669',
          'VERIFY': '#0D9488'
        };
        const color = actionColors[row.action] || '#6B7280';
        return `<span style="color: ${color}; font-weight: 600; font-size: 12px;">${escapeHtml(row.action || '')}</span>`;
      }
    },
    {
      key: 'is_automatic',
      label: 'Type',
      width: '80px',
      render: (row: AuditLogEntry) => {
        if (row.is_automatic) {
          const confidence = Math.round(row.confidence_score * 100);
          return `<span style="color: #8B5CF6; font-size: 11px;">Auto (${confidence}%)</span>`;
        }
        return `<span style="color: #6B7280; font-size: 11px;">Manual</span>`;
      }
    },
    {
      key: 'performed_by',
      label: 'User',
      sortable: true,
      width: '100px',
      render: (row: AuditLogEntry) => `<span style="font-size: 12px;">${escapeHtml(row.performed_by || 'System')}</span>`
    },
    {
      key: 'reason',
      label: 'Details',
      render: (row: AuditLogEntry) => {
        let text = row.reason || '';
        if (row.action_detail) {
          try {
            const detail = JSON.parse(row.action_detail);
            if (detail.invoice_id) text = `Invoice: ${detail.invoice_id}`;
            else if (detail.matched_count) text = `Matched: ${detail.matched_count}`;
            else if (detail.reason) text = detail.reason;
          } catch {
            text = row.action_detail.substring(0, 50);
          }
        }
        const maxLen = 60;
        const display = text.length > maxLen ? text.substring(0, maxLen) + '...' : text;
        return `<span style="font-size: 12px; color: var(--text-secondary);" title="${escapeHtml(text)}">${escapeHtml(display) || '-'}</span>`;
      }
    },
    {
      key: 'is_reversed',
      label: 'Status',
      width: '90px',
      render: (row: AuditLogEntry) => {
        if (row.is_reversed) {
          return `<span style="color: #EF4444; font-size: 11px; text-decoration: line-through;">Reversed</span>`;
        }
        return `<span style="color: #10B981; font-size: 11px;">Active</span>`;
      }
    }
  ];

  // Formatters
  function formatDateTime(dateValue: any): string {
    if (!dateValue) return '-';
    try {
      const date = typeof dateValue === 'string' ? new Date(dateValue) : new Date(dateValue);
      return date.toLocaleString('en-GB', {
        day: '2-digit',
        month: 'short',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      });
    } catch {
      return '-';
    }
  }

  // escapeHtml now imported from centralized utility

  // Filtering
  run(() => {
    let logs = auditLogs;

    if (actionFilter !== 'all') {
      logs = logs.filter(l => l.action === actionFilter);
    }

    if (!showReversed) {
      logs = logs.filter(l => !l.is_reversed);
    }

    filteredLogs = logs;
  });

  // Data loading
  async function loadData() {
    loading = true;
    try {
      const accounts = await GetActiveBankAccounts();
      bankAccounts = accounts || [];

      if (bankAccounts.length > 0 && !selectedAccountId) {
        selectedAccountId = bankAccounts[0].id;
      }

      await loadStatements();

      // If pre-filtered, load that statement's audit trail
      if (statementId) {
        selectedStatementId = statementId;
        await loadAuditTrail();
      }
    } catch (err) {
      console.error('Failed to load data:', err);
      toast.danger('Failed to load audit data');
    } finally {
      loading = false;
    }
  }

  async function loadStatements() {
    if (!selectedAccountId) return;
    try {
      const result = await GetBankStatements(selectedAccountId);
      statements = result || [];
      if (statements.length > 0 && !selectedStatementId) {
        selectedStatementId = statements[0].id;
        await loadAuditTrail();
      }
    } catch (err) {
      console.error('Failed to load statements:', err);
    }
  }

  async function loadAuditTrail() {
    if (!selectedStatementId) return;
    try {
      const result = await GetAuditTrail(selectedStatementId);
      auditLogs = result || [];
    } catch (err) {
      console.error('Failed to load audit trail:', err);
      auditLogs = [];
    }
  }

  // Actions — Wave 9.3 B3(c): row-click ALWAYS opens the read-only details
  // view. Reverse is only reachable from an explicit button inside it.
  function openDetailsModal(log: AuditLogEntry) {
    selectedLog = log;
    showDetailsModal = true;
  }

  async function handleReverse() {
    if (!selectedLog) return;
    if (selectedLog.is_reversed) {
      toast.warning('This action has already been reversed');
      return;
    }
    // Wave 9.3 B2: server re-resolves and ignores this value (Article III.4);
    // block here so the UI never sends a fake actor.
    if (!$currentUser?.id) {
      toast.danger('Cannot reverse: no authenticated user found. Please sign in again.');
      return;
    }

    const r = await confirm.askForReason({
      title: 'Reverse Action',
      message: `Reverse "${selectedLog.action}" performed ${formatDateTime(selectedLog.performed_at)} by ${selectedLog.performed_by || 'System'}? This may affect related reconciliation data.`,
      confirmLabel: 'Reverse Action',
      variant: 'warning',
      reasonLabel: 'Reversal reason',
      reasonRequired: true
    });
    if (!r.confirmed) return;

    reverseLoading = true;
    try {
      await ReverseAction(selectedLog.id, $currentUser.id, r.reason);
      toast.success('Action reversed');
      showDetailsModal = false;
      await loadAuditTrail();
    } catch (err: any) {
      console.error('Failed to reverse:', err);
      toast.danger(err?.message || 'Failed to reverse action');
    } finally {
      reverseLoading = false;
    }
  }

  async function handleAccountChange() {
    selectedStatementId = '';
    auditLogs = [];
    await loadStatements();
  }

  onMount(() => {
    loadData();
  });
</script>

<PageLayout title="Audit Trail" subtitle="Bank reconciliation action history" {embedded}>
  {#if loading}
    <div class="loading-container">
      <WabiSpinner size="lg" />
      <p>Loading audit trail...</p>
    </div>
  {:else}
    <!-- KPIs -->
    <div class="kpi-grid">
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">Total Actions</span>
          <span class="kpi-value">{totalActions}</span>
        </div>
      </Card>
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">Automatic</span>
          <span class="kpi-value auto">{autoActions}</span>
        </div>
      </Card>
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">Manual</span>
          <span class="kpi-value manual">{manualActions}</span>
        </div>
      </Card>
      <Card variant="elevated">
        <div class="kpi">
          <span class="kpi-label">Reversed</span>
          <span class="kpi-value reversed">{reversedActions}</span>
        </div>
      </Card>
    </div>

    <!-- Toolbar -->
    <div class="toolbar">
      <div class="left">
        <select bind:value={selectedAccountId} onchange={handleAccountChange} class="select">
          {#each bankAccounts as account}
            <option value={account.id}>{account.bank_name} - {account.account_number}</option>
          {/each}
        </select>
        <select bind:value={selectedStatementId} onchange={loadAuditTrail} class="select">
          {#each statements as stmt}
            <option value={stmt.id}>{stmt.statement_number}</option>
          {/each}
        </select>
      </div>
      <div class="right">
        <select bind:value={actionFilter} class="select filter">
          {#each actionTypes as type}
            <option value={type}>{type === 'all' ? 'All Actions' : type}</option>
          {/each}
        </select>
        <label class="checkbox-label">
          <input type="checkbox" bind:checked={showReversed} />
          Show Reversed
        </label>
      </div>
    </div>

    <!-- Audit Log Table — row-click ALWAYS opens read-only details (B3(c));
         Reverse only lives behind the explicit button inside that modal. -->
    <DataTable
      data={filteredLogs}
      columns={columns}
      onRowClick={openDetailsModal}
      emptyMessage="No audit entries found"
      rowClass={(row) => row.is_reversed ? 'reversed-row' : ''}
    />
  {/if}
</PageLayout>

<!-- Details Modal (read-only) — Wave 9.3 B3(c) -->
<Modal bind:open={showDetailsModal} title="Audit Log Entry" size="md">
  {#if selectedLog}
    <div class="reverse-details">
      <div class="detail-row">
        <span class="label">Action:</span>
        <span class="value">{selectedLog.action}</span>
      </div>
      <div class="detail-row">
        <span class="label">Performed:</span>
        <span class="value">{formatDateTime(selectedLog.performed_at)}</span>
      </div>
      <div class="detail-row">
        <span class="label">By:</span>
        <span class="value">{selectedLog.performed_by || 'System'}</span>
      </div>
      <div class="detail-row">
        <span class="label">Type:</span>
        <span class="value">{selectedLog.is_automatic ? `Automatic (${Math.round(selectedLog.confidence_score * 100)}%)` : 'Manual'}</span>
      </div>
      {#if selectedLog.reason}
        <div class="detail-row">
          <span class="label">Reason:</span>
          <span class="value">{selectedLog.reason}</span>
        </div>
      {/if}
      {#if selectedLog.is_reversed}
        <div class="detail-row">
          <span class="label">Reversed:</span>
          <span class="value">by {selectedLog.reversed_by || 'System'} on {formatDateTime(selectedLog.reversed_at)} — {selectedLog.reversal_reason}</span>
        </div>
      {/if}
    </div>
  {/if}
  {#snippet footer()}

      <Button variant="secondary" on:click={() => showDetailsModal = false}>Close</Button>
      {#if selectedLog && !selectedLog.is_reversed}
        <Button variant="warning" on:click={handleReverse} loading={reverseLoading}>
          Reverse Action
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
    color: var(--text-primary);
  }

  .kpi-value.auto { color: #8B5CF6; }
  .kpi-value.manual { color: #6B7280; }
  .kpi-value.reversed { color: #EF4444; }

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

  .select {
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: var(--surface-default);
    font-size: 14px;
    min-width: 180px;
  }

  .select.filter {
    min-width: 140px;
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 13px;
    color: var(--text-secondary);
    cursor: pointer;
  }

  .checkbox-label input {
    cursor: pointer;
  }

  .reverse-details {
    padding: var(--space-3);
    background: var(--surface-elevated);
    border-radius: var(--radius-sm);
    margin-bottom: var(--space-4);
  }

  .detail-row {
    display: flex;
    gap: var(--space-3);
    padding: var(--space-1) 0;
  }

  .detail-row .label {
    color: var(--text-secondary);
    min-width: 120px;
    font-size: 13px;
  }

  .detail-row .value {
    color: var(--text-primary);
    font-size: 13px;
  }

  :global(.reversed-row) {
    opacity: 0.6;
    background: rgba(239, 68, 68, 0.05) !important;
  }

  @media (max-width: 900px) {
    .kpi-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
</style>
