<script lang="ts">
  import { run } from 'svelte/legacy';

  /**
   * BankReconciliationScreen - Bank Statement Import & Reconciliation
   *
   * Features:
   * - Import bank statements (PDF/CSV)
   * - Auto-match transactions to invoices/payments
   * - Manual matching for unmatched items
   * - Reconciliation workflow with verification
   * - Real-time cash position display
   * - BHD currency formatting (3 decimal places)
   *
   * Design System: Wabi-Sabi minimalism x Bloomberg data density
   */

  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  // Wails API imports - will be available after binding regeneration
  import {
    GetBankStatements } from '../../../wailsjs/go/main/App';
import { GetBankStatementLines, GetActiveBankAccounts, GetCashPosition, ImportBankStatementCSV, AutoMatchBankLines, ManualMatchLine, UnmatchLine, FinalizeReconciliation, GetAuditTrail, DeleteBankStatement, UpdateBankStatement, CreateBankStatementLine, UpdateBankStatementLine, DeleteBankStatementLine, ListCustomerInvoices, GetSupplierInvoices, GetAllSupplierPayments, ListExpenseEntries, ListUnreconciledPayrollPayouts, CreateSplitAllocation } from '../../../wailsjs/go/main/FinanceService';
import { ImportBankStatementWithDialog, PreviewBankStatementImportWithDialog, ConfirmBankStatementImport, DiscardBankStatementImportPreview } from '../../../wailsjs/go/main/DocumentsService';

  // Design system components
  import PageLayout from '$lib/components/layout/PageLayout.svelte';
  import DataTable from '$lib/components/ui/DataTable.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import StatusBadge from '$lib/components/ui/StatusBadge.svelte';
  import Modal from '$lib/components/layout/Modal.svelte';
  import FormGroup from '$lib/components/ui/FormGroup.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import { toast } from '$lib/stores/toasts';
  import { confirm } from '$lib/stores/confirm';
  import { currentUser } from '$lib/stores/authContext';
  import { escapeHtml } from '$lib/utils/escapeHtml';

  
  interface Props {
    // Props
    embedded?: boolean;
    company?: 'Acme Instrumentation' | 'Beacon Controls';
  }

  let { embedded = false, company = 'Acme Instrumentation' }: Props = $props();

  interface BankStatement {
    id: string;
    division?: string;
    bank_account_id: string;
    statement_number: string;
    period_start: any;
    period_end: any;
    opening_balance: number;
    closing_balance: number;
    total_debits: number;
    total_credits: number;
    status: string;
    notes?: string;
  }

  interface BankStatementLine {
    id: string;
    bank_statement_id: string;
    line_number: number;
    transaction_date: any;
    description: string;
    reference: string;
    debit: number;
    credit: number;
    balance: number;
    transaction_type: string;
    is_matched: boolean;
    match_type: string;
    match_confidence: number;
    extracted_customer: string;
    extracted_invoices: string;
  }

  interface CashPosition {
    total_bhd: number;
    by_account: Record<string, number>;
    as_of: any;
  }

  // State - using any types for Wails-generated compatibility
  let bankAccounts: any[] = $state([]);
  let statements: any[] = $state([]);
  let selectedAccountId = $state('');
  let selectedStatement: any | null = $state(null);
  let statementLines: any[] = $state([]);
  let cashPosition: any = $state(null);
  let loading = $state(true);
  let statementsLoading = $state(false);
  let linesLoading = $state(false);

  // Modal state
  let showImportModal = $state(false);
  let showImportPreviewModal = $state(false);
  let showMatchModal = $state(false);
  let showEditStatementModal = $state(false);
  let showEditLineModal = $state(false);
  let showAddLineModal = $state(false);
  let importLoading = $state(false);
  let importPreview: any = $state(null); // parsed-but-not-persisted statement (Wave 9.3 B1d)
  let confirmingImport = $state(false);
  let showHandoffBanner = $state(false); // "Next -> Step 2" CTA after a successful Finalize
  let matchingLine: BankStatementLine | null = $state(null);
  let matchingType = $state('CUSTOMER_INVOICE');
  let matchingCandidateId = $state('');
  let matchingSearch = $state('');
  let matchAllocations: MatchAllocationDraft[] = $state([]);
  let matchLoading = $state(false);
  let customerInvoiceCandidates: any[] = [];
  let supplierInvoiceCandidates: any[] = [];
  let supplierPaymentCandidates: any[] = [];
  let expenseCandidates: any[] = [];
  let payrollPayoutCandidates: any[] = [];

  type MatchAllocationDraft = {
    key: string;
    allocation_type: string;
    entity_id: string;
    label: string;
    allocated_amount: number;
    max_amount: number;
  };

  // Import form
  let importAccountId = $state('');

  // Statement edit form
  let editStatementData = $state({
    opening_balance: 0,
    closing_balance: 0,
    period_start: '',
    period_end: '',
    status: 'Imported',
    notes: ''
  });
  let savingStatement = $state(false);

  // Transaction line edit/add form
  let lineFormData = $state({
    transaction_date: '',
    description: '',
    reference: '',
    debit: 0,
    credit: 0
  });
  let savingLine = $state(false);
  let editingLine: BankStatementLine | null = $state(null);

  function matchesCompany(division?: string) {
    return (division || 'Acme Instrumentation') === company;
  }



  // DataTable columns for statements
  const statementColumns = [
    {
      key: 'statement_number',
      label: 'Statement #',
      sortable: true,
      width: '280px',
      render: (row: BankStatement) => `<span style="display:block; font-family: var(--font-mono); font-size: 12px; line-height: 1.35; white-space: normal; overflow-wrap: anywhere;">${escapeHtml(row.statement_number)}</span>`
    },
    {
      key: 'period_end',
      label: 'Period',
      sortable: true,
      width: '180px',
      render: (row: BankStatement) => `<span style="font-size: 13px;">${formatDate(row.period_start)} - ${formatDate(row.period_end)}</span>`
    },
    {
      key: 'closing_balance',
      label: 'Closing Balance',
      sortable: true,
      width: '140px',
      align: 'right' as const,
      render: (row: BankStatement) => `<span style="font-family: var(--font-mono); font-weight: 600;">${formatBHD(row.closing_balance)}</span>`
    },
    {
      key: 'status',
      label: 'Status',
      sortable: true,
      width: '120px',
      render: (row: BankStatement) => {
        const statusColors: Record<string, string> = {
          'Imported': '#F59E0B',
          'In Progress': '#3B82F6',
          'Reconciled': '#10B981',
          'Verified': '#059669'
        };
        const color = statusColors[row.status] || '#6B7280';
        return `<span style="color: ${color}; font-weight: 500; font-size: 12px;">${escapeHtml(row.status)}</span>`;
      }
    }
  ];

  // DataTable columns for statement lines
  const lineColumns = [
    {
      key: 'transaction_date',
      label: 'Date',
      sortable: true,
      width: '100px',
      render: (row: BankStatementLine) => `<span style="font-family: var(--font-mono); font-size: 12px;">${formatDate(row.transaction_date)}</span>`
    },
    {
      key: 'description',
      label: 'Description',
      sortable: true,
      render: (row: BankStatementLine) => {
        const maxLen = 50;
        const desc = row.description.length > maxLen ? row.description.substring(0, maxLen) + '...' : row.description;
        return `<span style="font-size: 13px;" title="${escapeHtml(row.description)}">${escapeHtml(desc)}</span>`;
      }
    },
    {
      key: 'reference',
      label: 'Reference',
      sortable: true,
      width: '120px',
      render: (row: BankStatementLine) => `<span style="font-family: var(--font-mono); font-size: 11px; color: var(--text-secondary);">${escapeHtml(row.reference || '-')}</span>`
    },
    {
      key: 'debit',
      label: 'Debit',
      sortable: true,
      width: '110px',
      align: 'right' as const,
      render: (row: BankStatementLine) => row.debit > 0 ? `<span style="font-family: var(--font-mono); color: #EF4444; font-size: 13px;">${formatBHD(row.debit)}</span>` : '-'
    },
    {
      key: 'credit',
      label: 'Credit',
      sortable: true,
      width: '110px',
      align: 'right' as const,
      render: (row: BankStatementLine) => row.credit > 0 ? `<span style="font-family: var(--font-mono); color: #10B981; font-size: 13px;">${formatBHD(row.credit)}</span>` : '-'
    },
    {
      key: 'transaction_type',
      label: 'Type',
      sortable: true,
      width: '130px',
      render: (row: BankStatementLine) => {
        const typeColors: Record<string, string> = {
          'CUSTOMER_PAYMENT': '#10B981',
          'SUPPLIER_PAYMENT': '#3B82F6',
          'BANK_FEE': '#F59E0B',
          'INTEREST': '#8B5CF6',
          'TRANSFER': '#6366F1',
          'UNKNOWN': '#6B7280'
        };
        const color = typeColors[row.transaction_type] || '#6B7280';
        const label = row.transaction_type?.replace(/_/g, ' ') || 'Unknown';
        return `<span style="color: ${color}; font-size: 11px; font-weight: 500;">${escapeHtml(label)}</span>`;
      }
    },
    {
      key: 'is_matched',
      label: 'Match',
      sortable: true,
      width: '100px',
      render: (row: BankStatementLine) => {
        if (row.is_matched) {
          const confidence = Math.round(row.match_confidence * 100);
          return `<span style="color: #10B981; font-size: 12px;">Matched (${confidence}%)</span>`;
        }
        return `<span style="color: #EF4444; font-size: 12px;">Unmatched</span>`;
      }
    }
  ];

  // Formatters
  function formatBHD(value: number): string {
    return new Intl.NumberFormat('en-US', {
      minimumFractionDigits: 3,
      maximumFractionDigits: 3
    }).format(Number(value) || 0);
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

  function formatBankAccountOption(account: any): string {
    const bank = account?.bank_name || 'Bank Account';
    const accountName = account?.account_name ? ` ${account.account_name}` : '';
    const accountNumber = account?.account_number ? ` - ${account.account_number}` : '';
    const currency = account?.currency ? ` (${account.currency})` : '';
    return `${bank}${accountName}${accountNumber}${currency}`;
  }

  function getAccountBalanceBHD(account: any): number {
    return Number(
      account?.current_balance_bhd ??
      account?.CurrentBalanceBHD ??
      account?.current_balance ??
      account?.CurrentBalance ??
      0
    ) || 0;
  }

  // escapeHtml now imported from centralized utility

  // Data loading - independent API calls to prevent one failure from blocking the other
  async function loadData() {
    loading = true;
    let accountsError = null;
    let positionError = null;

    try {
      const accountsResult = await GetActiveBankAccounts();
      bankAccounts = accountsResult || [];
      const scopedAccounts = (accountsResult || []).filter((account: any) => matchesCompany(account.division));
      if (scopedAccounts.length > 0 && !selectedAccountId) {
        selectedAccountId = scopedAccounts[0].id;
        await loadStatements();
      }
    } catch (err) {
      console.error('Failed to load bank accounts:', err);
      accountsError = err;
      toast.danger('Failed to load bank accounts');
    }

    try {
      const positionResult = await GetCashPosition();
      cashPosition = {
        ...(positionResult || {}),
        accounts: (positionResult?.accounts || []).filter((account: any) => matchesCompany(account.division)),
      };
    } catch (err) {
      console.error('Failed to load cash position:', err);
      positionError = err;
      // Don't show toast for cash position failure - accounts are more critical
    }

    loading = false;
  }

  async function loadStatements() {
    if (!selectedAccountId) return;
    statementsLoading = true;
    try {
      const result = await GetBankStatements(selectedAccountId);
      statements = result || [];
    } catch (err) {
      console.error('Failed to load statements:', err);
      toast.danger('Failed to load statements');
    } finally {
      statementsLoading = false;
    }
  }

  async function loadStatementLines(statement: BankStatement) {
    selectedStatement = statement;
    linesLoading = true;
    try {
      const result = await GetBankStatementLines(statement.id);
      statementLines = result || [];
    } catch (err) {
      console.error('Failed to load statement lines:', err);
      toast.danger('Failed to load transactions');
    } finally {
      linesLoading = false;
    }
  }

  // Actions
  async function handleAutoMatch() {
    if (!selectedStatement) return;
    try {
      const result = await AutoMatchBankLines(selectedStatement.id);
      toast.success(`Auto-matched ${result?.matched_count || 0} transactions`);
      await loadStatementLines(selectedStatement);
    } catch (err) {
      console.error('Auto-match failed:', err);
      toast.danger('Auto-match failed');
    }
  }

  async function handleFinalize() {
    if (!selectedStatement) return;
    if (totalUnmatched > 0) {
      toast.warning(`Cannot finalize with ${totalUnmatched} unmatched transactions`);
      return;
    }
    // Wave 9.3 B2: attribution comes from the authenticated session, not a
    // hardcoded 'admin' literal. The server re-resolves and ignores this
    // value anyway (Article III.4) — block here so the UI never pretends to
    // know who's acting when it doesn't.
    if (!$currentUser?.id) {
      toast.danger('Cannot finalize: no authenticated user found. Please sign in again.');
      return;
    }
    try {
      await FinalizeReconciliation(selectedStatement.id, $currentUser.id);
      // Wave 10 B6: toast removed — duplicated the persistent inline
      // .handoff-banner below, which already confirms this event with a CTA.
      showHandoffBanner = true;
      await loadStatements();
    } catch (err) {
      console.error('Finalize failed:', err);
      toast.danger('Finalization failed');
    }
  }

  function goToProveBalance() {
    window.dispatchEvent(new CustomEvent('finance:navigate', {
      detail: { tab: 'book_bank', company }
    }));
  }

  function openMatchModal(line: BankStatementLine) {
    matchingLine = line;
    matchingSearch = '';
    matchingCandidateId = '';
    matchAllocations = [];
    matchingType = line.credit > 0 ? 'CUSTOMER_INVOICE' : 'SUPPLIER_INVOICE';
    showMatchModal = true;
    void loadMatchCandidates(line);
  }

  function handleMatchingTypeChange() {
    matchingCandidateId = '';
    matchingSearch = '';
    matchAllocations = [];
  }

  async function loadMatchCandidates(line: BankStatementLine) {
    matchLoading = true;
    try {
      const [allCustomerInvoices, allSupplierInvoices, supplierPayments, expenses] = await Promise.all([
        ListCustomerInvoices(1000, 0).catch(() => []),
        GetSupplierInvoices().catch(() => []),
        GetAllSupplierPayments().catch(() => []),
        ListExpenseEntries('', false).catch(() => [])
      ]);

      customerInvoiceCandidates = (allCustomerInvoices || []).filter((invoice: any) => {
        const status = String(invoice?.status || '').toLowerCase();
        return matchesCompany(invoice?.division) && Number(invoice?.outstanding_bhd || 0) > 0 && !['cancelled', 'void', 'draft'].includes(status);
      });
      supplierInvoiceCandidates = (allSupplierInvoices || []).filter((invoice: any) => {
        const status = String(invoice?.status || '').toLowerCase();
        const paymentStatus = String(invoice?.payment_status || '').toLowerCase();
        return matchesCompany(invoice?.division) && paymentStatus !== 'paid' && !['cancelled', 'void', 'rejected'].includes(status);
      });
      supplierPaymentCandidates = (supplierPayments || []).filter((payment: any) => matchesCompany(payment?.division));
      expenseCandidates = (expenses || []).filter((expense: any) => {
        const status = String(expense?.status || '').toLowerCase();
        const paymentStatus = String(expense?.payment_status || '').toLowerCase();
        return matchesCompany(expense?.division)
          && paymentStatus !== 'paid'
          && !['cancelled', 'canceled', 'void', 'rejected'].includes(status)
          && getCandidateAmount(expense, 'EXPENSE') > 0;
      });
      payrollPayoutCandidates = (await ListUnreconciledPayrollPayouts().catch(() => [])).filter((payout: any) => matchesCompany(payout?.division));
    } finally {
      matchLoading = false;
    }
  }

  function formatMatchAmount(line: BankStatementLine) {
    return line.credit > 0 ? line.credit : line.debit;
  }

  function roundAmount(value: number) {
    return Math.round((Number(value) || 0) * 1000) / 1000;
  }

  function buildMatchCandidateLabel(candidate: any, type: string) {
    if (type === 'CUSTOMER_INVOICE') {
      const outstanding = Number(candidate.outstanding_bhd || candidate.grand_total_bhd || 0);
      return `${candidate.invoice_number || 'Invoice'} • ${candidate.customer_name || 'Customer'} • ${formatBHD(outstanding)} BHD`;
    }
    if (type === 'SUPPLIER_PAYMENT') {
      return `${candidate.reference || candidate.invoice_number || 'Payment'} • ${candidate.supplier_name || 'Supplier'} • ${formatBHD(candidate.amount_bhd || 0)} BHD`;
    }
    if (type === 'PAYROLL_PAYOUT') {
      return `${candidate.run_number || 'Payroll'} • ${candidate.employee_name || 'Employee'} • ${formatBHD(candidate.amount || 0)} BHD${candidate.payment_reference ? ` • ${candidate.payment_reference}` : ''}`;
    }
    if (type === 'EXPENSE') {
      const amount = getCandidateAmount(candidate, type);
      const name = candidate.vendor_name || candidate.category_name || candidate.cost_center || 'Expense';
      return `${candidate.entry_number || 'Expense'} • ${name} • ${formatBHD(amount)} BHD`;
    }
    return `${candidate.invoice_number || 'Invoice'} • ${candidate.supplier_name || 'Supplier'} • ${formatBHD(candidate.total_bhd || 0)} BHD`;
  }

  function getCandidateAmount(candidate: any, type: string) {
    if (type === 'CUSTOMER_INVOICE') return Number(candidate.outstanding_bhd || candidate.grand_total_bhd || 0);
    if (type === 'SUPPLIER_PAYMENT') return Number(candidate.amount_bhd || 0);
    if (type === 'PAYROLL_PAYOUT') return Number(candidate.amount || 0);
    if (type === 'EXPENSE') return Number(candidate.total_amount || ((Number(candidate.amount) || 0) + (Number(candidate.vat_amount) || 0)) || candidate.amount || 0);
    return Number(candidate.total_bhd || 0);
  }

  function allocationKey(type: string, entityId: string) {
    return `${type}:${entityId}`;
  }

  function isCandidateAllocated(candidate: any) {
    return matchAllocations.some((allocation) => allocation.key === allocationKey(matchingType, candidate.id));
  }

  function addMatchAllocation(candidate: any) {
    if (!matchingLine || !candidate?.id || isCandidateAllocated(candidate)) return;
    if (matchingType === 'SUPPLIER_PAYMENT' || matchingType === 'PAYROLL_PAYOUT') {
      toast.warning('Bulk allocation is available for invoices and expenses. Match payouts one at a time.');
      return;
    }

    const candidateAmount = roundAmount(getCandidateAmount(candidate, matchingType));
    const remaining = matchAllocations.length === 0 ? formatMatchAmount(matchingLine) : matchAllocationRemaining;
    const defaultAmount = roundAmount(Math.max(0, Math.min(candidateAmount || remaining, remaining > 0 ? remaining : candidateAmount)));
    matchAllocations = [
      ...matchAllocations,
      {
        key: allocationKey(matchingType, candidate.id),
        allocation_type: matchingType,
        entity_id: candidate.id,
        label: buildMatchCandidateLabel(candidate, matchingType),
        allocated_amount: defaultAmount,
        max_amount: candidateAmount
      }
    ];
    matchingCandidateId = '';
  }

  function updateMatchAllocationAmount(key: string, event: Event) {
    const target = event.currentTarget as HTMLInputElement;
    const value = roundAmount(Number(target.value) || 0);
    matchAllocations = matchAllocations.map((allocation) =>
      allocation.key === key ? { ...allocation, allocated_amount: value } : allocation
    );
  }

  function removeMatchAllocation(key: string) {
    matchAllocations = matchAllocations.filter((allocation) => allocation.key !== key);
  }

  function normalizeCandidateText(value: any) {
    return String(value || '').toLowerCase().replace(/[^a-z0-9.]+/g, ' ').trim();
  }

  function getFilteredMatchCandidates() {
    const search = normalizeCandidateText(matchingSearch);
    const source = matchingType === 'CUSTOMER_INVOICE'
      ? customerInvoiceCandidates
      : matchingType === 'SUPPLIER_PAYMENT'
        ? supplierPaymentCandidates
        : matchingType === 'PAYROLL_PAYOUT'
          ? payrollPayoutCandidates
          : matchingType === 'EXPENSE'
            ? expenseCandidates
            : supplierInvoiceCandidates;

    const lineAmount = matchingLine ? formatMatchAmount(matchingLine) : 0;
    return (source || []).filter((candidate) => {
      if (!search) return true;
      const haystack = [
        candidate.invoice_number,
        candidate.po_number,
        candidate.customer_name,
        candidate.supplier_name,
        candidate.reference,
        candidate.employee_name,
        candidate.run_number,
        candidate.payment_reference,
        candidate.order_number,
        candidate.entry_number,
        candidate.description,
        candidate.category_name,
        candidate.vendor_name,
        candidate.cost_center,
        getCandidateAmount(candidate, matchingType).toFixed(3)
      ].filter(Boolean).map(normalizeCandidateText).join(' ');
      return haystack.includes(search);
    }).sort((a, b) => {
      const aDiff = Math.abs(getCandidateAmount(a, matchingType) - lineAmount);
      const bDiff = Math.abs(getCandidateAmount(b, matchingType) - lineAmount);
      return aDiff - bDiff;
    }).slice(0, 200);
  }

  async function handleManualMatch() {
    if (!matchingLine) {
      toast.warning('Select a transaction to match');
      return;
    }
    if (matchAllocations.length === 0 && !matchingCandidateId) {
      toast.warning('Select a transaction target to match');
      return;
    }
    if (matchAllocations.length > 0) {
      const invalidAllocation = matchAllocations.find((allocation) => !allocation.allocated_amount || allocation.allocated_amount <= 0);
      if (invalidAllocation) {
        toast.warning('Every allocation needs a positive amount');
        return;
      }
      if (Math.abs(matchAllocationRemaining) > 0.001) {
        toast.warning(`Allocation total must equal the bank line amount. Remaining: ${formatBHD(matchAllocationRemaining)} BHD`);
        return;
      }
    }
    // Wave 9.3 B2: server re-resolves and ignores this value (Article III.4);
    // block here so the UI never sends a fake actor.
    if (!$currentUser?.id) {
      toast.danger('Cannot match: no authenticated user found. Please sign in again.');
      return;
    }

    matchLoading = true;
    try {
      if (matchAllocations.length > 0) {
        await CreateSplitAllocation(
          matchingLine.id,
          matchAllocations.map((allocation) => ({
            allocation_type: allocation.allocation_type,
            entity_id: allocation.entity_id,
            allocated_amount: roundAmount(allocation.allocated_amount)
          })),
          $currentUser.id
        );
        toast.success('Transaction allocated successfully');
      } else {
        await ManualMatchLine(matchingLine.id, matchingType, matchingCandidateId, $currentUser.id);
        toast.success('Transaction matched successfully');
      }
      showMatchModal = false;
      matchAllocations = [];
      if (selectedStatement) {
        await loadStatementLines(selectedStatement);
      }
    } catch (err) {
      console.error('Manual match failed:', err);
      toast.danger(`Manual match failed: ${err}`);
    } finally {
      matchLoading = false;
    }
  }

  async function handleUnmatch(line: BankStatementLine) {
    // Wave 9.3 B2: server re-resolves and ignores this value (Article III.4);
    // block here so the UI never sends a fake actor.
    if (!$currentUser?.id) {
      toast.danger('Cannot unmatch: no authenticated user found. Please sign in again.');
      return;
    }
    try {
      // UnmatchLine(lineID, user, reason)
      await UnmatchLine(line.id, $currentUser.id, 'Manual unmatch');
      toast.success('Transaction unmatched');
      if (selectedStatement) {
        await loadStatementLines(selectedStatement);
      }
    } catch (err) {
      console.error('Unmatch failed:', err);
      toast.danger('Failed to unmatch');
    }
  }

  // Import handler - Wave 9.3 B1d: parse the statement and show a preview
  // before anything is persisted. ConfirmImportPreview() does the actual write.
  async function handleImport() {
    if (!importAccountId) {
      toast.warning('Please select a bank account');
      return;
    }
    importLoading = true;
    try {
      const preview = await PreviewBankStatementImportWithDialog(importAccountId);
      if (preview) {
        importPreview = preview;
        showImportModal = false;
        showImportPreviewModal = true;
      } else {
        toast.info('Import cancelled');
      }
    } catch (err) {
      console.error('Preview failed:', err);
      toast.danger(`Import failed: ${err}`);
    } finally {
      importLoading = false;
    }
  }

  async function handleConfirmImportPreview() {
    if (!importPreview) return;
    confirmingImport = true;
    try {
      const importedForAccount = importAccountId;
      const result = await ConfirmBankStatementImport(importPreview.id);
      if (result) {
        toast.success(`Imported statement: ${result.statement_number}`);
        showImportPreviewModal = false;
        importPreview = null;
        importAccountId = '';
        // Switch to the imported account and refresh
        selectedAccountId = importedForAccount;
        await loadStatements();
        // Also refresh cash position
        try {
          const pos = await GetCashPosition();
          cashPosition = {
            ...(pos || {}),
            accounts: (pos?.accounts || []).filter((account: any) => matchesCompany(account.division)),
          };
        } catch (_) {}
      }
    } catch (err) {
      console.error('Import failed:', err);
      toast.danger(`Import failed: ${err}`);
    } finally {
      confirmingImport = false;
    }
  }

  async function handleCancelImportPreview() {
    if (importPreview) {
      try { await DiscardBankStatementImportPreview(importPreview.id); } catch (_) {}
    }
    showImportPreviewModal = false;
    importPreview = null;
  }

  // Account change handler
  async function handleAccountChange() {
    selectedStatement = null;
    statementLines = [];
    await loadStatements();
  }

  // Wave 9.4 C1: bank-account CRUD (add/edit/deactivate) relocated to
  // Settings -> Bank Accounts. This screen keeps only the read-only
  // GetActiveBankAccounts() picker below, used by the statement import and
  // matching dropdowns.
  function openManageAccountsSettings() {
    window.dispatchEvent(new CustomEvent('navigateToScreen', {
      detail: { screen: 'settings', section: 'accounts' }
    }));
  }

  // Statement CRUD handlers
  async function handleDeleteStatement() {
    if (!selectedStatement) return;
    if (!(await confirm.ask({
      title: 'Delete Statement',
      message: `Are you sure you want to delete statement ${selectedStatement.statement_number}?`,
      confirmLabel: 'Delete',
      variant: 'danger'
    }))) {
      return;
    }

    try {
      await DeleteBankStatement(selectedStatement.id);
      toast.success('Statement deleted successfully');
      selectedStatement = null;
      statementLines = [];
      await loadStatements();
    } catch (err) {
      console.error('Failed to delete statement:', err);
      toast.danger(`Failed to delete statement: ${err}`);
    }
  }

  function openEditStatementModal() {
    if (!selectedStatement) return;

    // Format dates for input type="date" (YYYY-MM-DD)
    const formatDateForInput = (dateValue: any) => {
      if (!dateValue) return '';
      const date = typeof dateValue === 'string' ? new Date(dateValue) : new Date(dateValue);
      return date.toISOString().split('T')[0];
    };

    editStatementData = {
      opening_balance: selectedStatement.opening_balance,
      closing_balance: selectedStatement.closing_balance,
      period_start: formatDateForInput(selectedStatement.period_start),
      period_end: formatDateForInput(selectedStatement.period_end),
      status: selectedStatement.status,
      notes: selectedStatement.notes || ''
    };
    showEditStatementModal = true;
  }

  async function handleSaveStatement() {
    if (savingStatement) return;
    if (!selectedStatement) return;

    // Validation
    if (!editStatementData.period_start || !editStatementData.period_end) {
      toast.warning('Period start and end dates are required');
      return;
    }

    savingStatement = true;
    try {
      const statementID = selectedStatement.id;
      await UpdateBankStatement(statementID, editStatementData);
      selectedStatement = {
        ...selectedStatement,
        ...editStatementData,
      };
      toast.success('Statement updated successfully');
      showEditStatementModal = false;
      await loadStatements();
      // Reload the selected statement to show updated data
      const updatedStatement = statements.find(s => s.id === statementID);
      if (updatedStatement) {
        selectedStatement = updatedStatement;
      }
    } catch (err) {
      console.error('Failed to update statement:', err);
      toast.danger(`Failed to update statement: ${err}`);
    } finally {
      savingStatement = false;
    }
  }

  // Transaction line CRUD handlers
  function openEditLineModal(line: BankStatementLine) {
    editingLine = line;

    // Format date for input
    const formatDateForInput = (dateValue: any) => {
      if (!dateValue) return '';
      const date = typeof dateValue === 'string' ? new Date(dateValue) : new Date(dateValue);
      return date.toISOString().split('T')[0];
    };

    lineFormData = {
      transaction_date: formatDateForInput(line.transaction_date),
      description: line.description,
      reference: line.reference || '',
      debit: line.debit,
      credit: line.credit
    };
    showEditLineModal = true;
  }

  function openAddLineModal() {
    if (!selectedStatement) return;
    editingLine = null;
    lineFormData = {
      transaction_date: '',
      description: '',
      reference: '',
      debit: 0,
      credit: 0
    };
    showAddLineModal = true;
  }

  function flipLineDebitCredit() {
    const nextDebit = Number(lineFormData.credit) || 0;
    const nextCredit = Number(lineFormData.debit) || 0;
    lineFormData = {
      ...lineFormData,
      debit: nextDebit,
      credit: nextCredit
    };
  }

  function openOCRReviewModal(line: BankStatementLine) {
    showMatchModal = false;
    openEditLineModal(line);
  }

  async function handleSaveLine() {
    if (savingLine) return;

    // Validation
    if (!lineFormData.transaction_date) {
      toast.warning('Transaction date is required');
      return;
    }
    if (!lineFormData.description) {
      toast.warning('Description is required');
      return;
    }
    if (lineFormData.debit === 0 && lineFormData.credit === 0) {
      toast.warning('Either debit or credit must be non-zero');
      return;
    }
    if (lineFormData.debit > 0 && lineFormData.credit > 0) {
      toast.warning('Cannot have both debit and credit in the same line');
      return;
    }

    savingLine = true;
    try {
      const editedMatchedLine = Boolean(editingLine?.is_matched);
      if (editingLine) {
        // Update existing line
        await UpdateBankStatementLine(editingLine.id, lineFormData);
        toast.success(editedMatchedLine
          ? 'Transaction updated and existing match cleared for re-review'
          : 'Transaction updated successfully');
        showEditLineModal = false;
      } else {
        // Create new line
        if (!selectedStatement) {
          toast.danger('No statement selected');
          return;
        }
        await CreateBankStatementLine(selectedStatement.id, lineFormData);
        toast.success('Transaction added successfully');
        showAddLineModal = false;
      }

      // Reload lines and statement (totals may have changed)
      if (selectedStatement) {
        await loadStatementLines(selectedStatement);
        await loadStatements();
      }
    } catch (err) {
      console.error('Failed to save transaction:', err);
      toast.danger(`Failed to save transaction: ${err}`);
    } finally {
      savingLine = false;
    }
  }

  async function handleDeleteLine() {
    if (!editingLine) return;
    if (!(await confirm.ask({
      title: 'Delete Transaction',
      message: 'Are you sure you want to delete this transaction?',
      confirmLabel: 'Delete',
      variant: 'danger'
    }))) {
      return;
    }

    savingLine = true;
    try {
      await DeleteBankStatementLine(editingLine.id);
      toast.success('Transaction deleted successfully');
      showEditLineModal = false;

      // Reload lines and statement
      if (selectedStatement) {
        await loadStatementLines(selectedStatement);
        await loadStatements();
      }
    } catch (err) {
      console.error('Failed to delete transaction:', err);
      toast.danger(`Failed to delete transaction: ${err}`);
    } finally {
      savingLine = false;
    }
  }

  onMount(() => {
    loadData();
  });

  let visibleBankAccounts = $derived(bankAccounts.filter((account) => matchesCompany(account.division)));
  let visibleCashAccounts = $derived((cashPosition?.accounts || []).filter((account: any) => matchesCompany(account.division)));
  let totalVisibleCashPosition = $derived(visibleCashAccounts.reduce((sum: number, account: any) => sum + getAccountBalanceBHD(account), 0));
  let cashPositionNotices = $derived(visibleCashAccounts.map((account: any) => account.notice).filter(Boolean));
  run(() => {
    if (company && visibleBankAccounts.length > 0 && !visibleBankAccounts.some((account: any) => account.id === selectedAccountId)) {
      selectedAccountId = visibleBankAccounts[0].id;
      loadStatements();
    }
  });
  run(() => {
    if (company && visibleBankAccounts.length === 0 && selectedAccountId) {
      selectedAccountId = '';
      selectedStatement = null;
      statementLines = [];
    }
  });
  // KPI calculations
  let totalUnmatched = $derived(statementLines.filter(l => !l.is_matched).length);
  let totalMatched = $derived(statementLines.filter(l => l.is_matched).length);
  let unmatchedDebit = $derived(statementLines.filter(l => !l.is_matched && l.debit > 0).reduce((sum, l) => sum + l.debit, 0));
  let unmatchedCredit = $derived(statementLines.filter(l => !l.is_matched && l.credit > 0).reduce((sum, l) => sum + l.credit, 0));
  let matchLineAmount = $derived(matchingLine ? formatMatchAmount(matchingLine) : 0);
  let matchAllocationTotal = $derived(roundAmount(matchAllocations.reduce((sum, allocation) => sum + (Number(allocation.allocated_amount) || 0), 0)));
  let matchAllocationRemaining = $derived(roundAmount(matchLineAmount - matchAllocationTotal));
  run(() => {
    if (company) {
      importAccountId = '';
      loadData();
    }
  });
</script>

<PageLayout title="Bank Reconciliation" subtitle="Import and reconcile bank statements" {embedded}>
    {#if loading}
      <div class="loading-container">
        <WabiSpinner size="lg" />
        <p>Loading bank data...</p>
      </div>
    {:else}
      <!-- Cash Position KPIs -->
      <div class="kpi-grid">
        <Card variant="elevated">
          <div class="kpi">
            <span class="kpi-label">Cash Balance</span>
            <span class="kpi-value primary">{formatBHD(totalVisibleCashPosition)} BHD</span>
            <span class="kpi-note">Latest statement closings</span>
          </div>
        </Card>
        <Card variant="elevated">
          <div class="kpi">
            <span class="kpi-label">Matched</span>
            <span class="kpi-value success">{totalMatched}</span>
          </div>
        </Card>
        <Card variant="elevated">
          <div class="kpi">
            <span class="kpi-label">Unmatched</span>
            <span class="kpi-value warning">{totalUnmatched}</span>
          </div>
        </Card>
        <Card variant="elevated">
          <div class="kpi">
            <span class="kpi-label">Unmatched Credits</span>
            <span class="kpi-value">{formatBHD(unmatchedCredit)} BHD</span>
          </div>
        </Card>
      </div>

      {#if cashPositionNotices.length > 0}
        <div class="cash-position-notice">
          <strong>Statement check</strong>
          <span>{cashPositionNotices.join(' ')}</span>
        </div>
      {/if}

      <!-- Wave 9.3 B1a: name the two-step month-end sequence so this screen
           reads as "step 1 of 2" rather than a standalone tool. -->
      <div class="close-month-header">
        <span class="close-month-step">Close the Month · Step 1: Match transactions</span>
        <button type="button" class="close-month-jump" onclick={goToProveBalance}>
          Step 2: Prove the balance →
        </button>
      </div>

      {#if showHandoffBanner}
        <div class="handoff-banner" in:fade={{ duration: 150 }}>
          <span>Statement reconciled. Ready for the next step.</span>
          <Button variant="primary" size="sm" on:click={goToProveBalance}>
            Next → Step 2: Prove the balance
          </Button>
        </div>
      {/if}

      <!-- Account Selector and Actions -->
      <div class="toolbar">
        <div class="left">
          <FormGroup label="Bank Account" inline>
            <select bind:value={selectedAccountId} onchange={handleAccountChange} class="account-select" disabled={statementsLoading}>
              {#each visibleBankAccounts as account}
                <option value={account.id}>{formatBankAccountOption(account)}</option>
              {/each}
            </select>
          </FormGroup>
          {#if statementsLoading}
            <div class="inline-loading">
              <WabiSpinner size="sm" tempo="calm" />
              <span>Refreshing statements...</span>
            </div>
          {/if}
        </div>
        <div class="right">
          <Button variant="secondary" on:click={() => showImportModal = true}>
            Import Statement
          </Button>
          {#if selectedStatement}
            <Button variant="secondary" on:click={openEditStatementModal}>
              Edit Statement
            </Button>
            <Button variant="danger" on:click={handleDeleteStatement}>
              Delete Statement
            </Button>
            <Button variant="primary" on:click={handleAutoMatch}>
              Auto-Match
            </Button>
            <Button variant="success" on:click={handleFinalize} disabled={totalUnmatched > 0}>
              Finalize
            </Button>
          {/if}
        </div>
      </div>

      <!-- Wave 9.4 C1: bank-account CRUD now lives in Settings -> Bank Accounts
           (one admin home). This screen keeps only the read-only account
           picker above for statement import/matching. -->
      <div class="manage-accounts-disclosure">
        <button type="button" class="manage-accounts-link" onclick={openManageAccountsSettings}>
          Manage bank accounts (Settings)…
        </button>
      </div>

      <!-- Split View: Statements + Lines -->
      <div class="split-view">
        <!-- Statements Panel -->
        <div class="panel statements-panel">
          <h3>Statements</h3>
          {#if statementsLoading}
            <div class="loading-container small">
              <WabiSpinner size="md" tempo="calm" />
            </div>
          {:else}
            <DataTable
              data={statements}
              columns={statementColumns}
              onRowClick={loadStatementLines}
              selectedId={selectedStatement?.id}
              emptyMessage="No statements imported"
            />
          {/if}
        </div>

        <!-- Lines Panel -->
        <div class="panel lines-panel">
          <div class="panel-header">
            <h3>
              {#if selectedStatement}
                Transactions - {selectedStatement.statement_number}
              {:else}
                Select a statement
              {/if}
            </h3>
            {#if selectedStatement}
              <Button variant="secondary" size="sm" on:click={openAddLineModal}>
                Add Transaction
              </Button>
            {/if}
          </div>
          {#if selectedStatement?.notes}
            <div class="statement-notes-card">
              <div class="statement-notes-head">
                <span>Statement Notes</span>
                <span>OCR / AI</span>
              </div>
              <p>{selectedStatement.notes}</p>
            </div>
          {/if}
          {#if linesLoading}
            <div class="loading-container small">
              <WabiSpinner size="md" />
            </div>
          {:else if selectedStatement}
            <DataTable
              data={statementLines}
              columns={lineColumns}
              onRowClick={openMatchModal}
              emptyMessage="No transactions"
              rowClass={(row) => row.is_matched ? '' : 'unmatched-row'}
            />
          {:else}
            <div class="empty-state">
              <p>Select a statement to view transactions</p>
            </div>
          {/if}
        </div>
      </div>
    {/if}
</PageLayout>

<!-- Import Modal -->
<Modal bind:open={showImportModal} title="Import Bank Statement" size="xl">
  <div class="import-form">
    <FormGroup label="Bank Account">
      <select bind:value={importAccountId} class="form-select">
        <option value="">Select account...</option>
        {#each visibleBankAccounts as account}
          <option value={account.id}>{formatBankAccountOption(account)}</option>
        {/each}
      </select>
    </FormGroup>
    <div class="import-note">
      <p>A file dialog will open when you click Preview. Nothing is saved until you confirm the parsed rows.</p>
      <p>Supported formats: CSV and PDF</p>
    </div>
  </div>
  {#snippet footer()}

      <Button variant="secondary" on:click={() => showImportModal = false}>Cancel</Button>
      <Button variant="primary" on:click={handleImport} loading={importLoading} disabled={!importAccountId}>
        Preview
      </Button>

  {/snippet}
</Modal>

<!-- Import Preview Modal (Wave 9.3 B1d): parsed rows, nothing persisted yet -->
<Modal bind:open={showImportPreviewModal} title="Review Parsed Statement" size="xl">
  {#if importPreview}
    <div class="import-preview">
      <div class="import-preview-summary">
        <div class="match-row">
          <span class="label">Statement #:</span>
          <span class="value">{importPreview.statement_number}</span>
        </div>
        <div class="match-row">
          <span class="label">Period:</span>
          <span class="value">{formatDate(importPreview.period_start)} - {formatDate(importPreview.period_end)}</span>
        </div>
        <div class="match-row">
          <span class="label">Opening / Closing:</span>
          <span class="value">{formatBHD(importPreview.opening_balance)} / {formatBHD(importPreview.closing_balance)} BHD</span>
        </div>
        <div class="match-row">
          <span class="label">Lines Parsed:</span>
          <span class="value">{(importPreview.Lines || importPreview.lines || []).length}</span>
        </div>
      </div>
      <div class="import-preview-note">
        <p>Nothing has been saved yet. Review the parsed rows below, then confirm to commit them to the ledger — or cancel to discard.</p>
      </div>
      <DataTable
        data={importPreview.Lines || importPreview.lines || []}
        columns={lineColumns}
        emptyMessage="No transactions parsed"
      />
    </div>
  {/if}
  {#snippet footer()}

      <Button variant="secondary" on:click={handleCancelImportPreview}>Cancel Import</Button>
      <Button variant="primary" on:click={handleConfirmImportPreview} loading={confirmingImport}>
        Confirm & Save
      </Button>

  {/snippet}
</Modal>

<!-- Match Modal -->
<Modal bind:open={showMatchModal} title="Match Transaction" size="xl">
  {#if matchingLine}
    <div class="match-details">
      <div class="match-row">
        <span class="label">Date:</span>
        <span class="value">{formatDate(matchingLine.transaction_date)}</span>
      </div>
      <div class="match-row">
        <span class="label">Description:</span>
        <span class="value">{matchingLine.description}</span>
      </div>
      <div class="match-row">
        <span class="label">Amount:</span>
        <span class="value {matchingLine.credit > 0 ? 'credit' : 'debit'}">
          {matchingLine.credit > 0 ? '+' : '-'}{formatBHD(matchingLine.credit || matchingLine.debit)} BHD
        </span>
      </div>
      {#if matchingLine.extracted_customer}
        <div class="match-row">
          <span class="label">Detected Customer:</span>
          <span class="value">{matchingLine.extracted_customer}</span>
        </div>
      {/if}
      {#if matchingLine.extracted_invoices}
        <div class="match-row">
          <span class="label">Detected Invoices:</span>
          <span class="value">{matchingLine.extracted_invoices}</span>
        </div>
      {/if}
    </div>
    <div class="match-actions">
      {#if matchingLine.is_matched}
        <Button variant="secondary" on:click={() => matchingLine && handleUnmatch(matchingLine)}>
          Unmatch
        </Button>
      {:else}
        <div class="manual-match-form">
          <div class="ocr-review-banner">
            <div>
              <strong>OCR review</strong>
              <p>If the parser flipped the polarity, correct the line before matching.</p>
            </div>
            <Button variant="secondary" size="sm" on:click={() => matchingLine && openOCRReviewModal(matchingLine)}>
              Fix Debit/Credit
            </Button>
          </div>
          <div class="form-row two-col">
            <FormGroup label="Match To">
              <select bind:value={matchingType} class="form-select" onchange={handleMatchingTypeChange}>
                {#if matchingLine.credit > 0}
                  <option value="CUSTOMER_INVOICE">Customer Invoice</option>
                {:else}
                  <option value="SUPPLIER_INVOICE">Supplier Invoice</option>
                  <option value="EXPENSE">Expense</option>
                  <option value="SUPPLIER_PAYMENT">Supplier Payment</option>
                  <option value="PAYROLL_PAYOUT">Payroll Payout</option>
                {/if}
              </select>
            </FormGroup>
            <FormGroup label="Search">
              <input
                type="search"
                class="form-input"
                bind:value={matchingSearch}
                placeholder="Search number, customer, supplier, amount..."
                autocomplete="off"
              />
            </FormGroup>
          </div>

          {#if matchLoading}
            <div class="loading-container small">
              <WabiSpinner size="sm" />
            </div>
          {:else}
            {#if matchAllocations.length > 0}
              <div class="allocation-panel">
                <div class="allocation-head">
                  <div>
                    <strong>Allocation plan</strong>
                    <span>{matchAllocations.length} target{matchAllocations.length === 1 ? '' : 's'}</span>
                  </div>
                  <div class="allocation-balance" class:balanced={Math.abs(matchAllocationRemaining) <= 0.001} class:over={matchAllocationRemaining < -0.001}>
                    <span>Total {formatBHD(matchAllocationTotal)} BHD</span>
                    <span>{matchAllocationRemaining >= 0 ? 'Remaining' : 'Over'} {formatBHD(Math.abs(matchAllocationRemaining))} BHD</span>
                  </div>
                </div>
                <div class="allocation-list">
                  {#each matchAllocations as allocation (allocation.key)}
                    <div class="allocation-row">
                      <div class="allocation-label">
                        <span>{allocation.label}</span>
                        <small>Open {formatBHD(allocation.max_amount)} BHD</small>
                      </div>
                      <input
                        type="number"
                        min="0.001"
                        step="0.001"
                        class="allocation-input"
                        value={allocation.allocated_amount}
                        oninput={(event) => updateMatchAllocationAmount(allocation.key, event)}
                      />
                      <button type="button" class="allocation-remove" onclick={() => removeMatchAllocation(allocation.key)}>
                        Remove
                      </button>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
            <div class="candidate-list">
              {#if getFilteredMatchCandidates().length > 0}
                {#each getFilteredMatchCandidates() as candidate}
                  <div class="candidate-option" class:in-plan={isCandidateAllocated(candidate)}>
                    <label>
                      <input type="radio" bind:group={matchingCandidateId} value={candidate.id} disabled={matchAllocations.length > 0} />
                      <span>{buildMatchCandidateLabel(candidate, matchingType)}</span>
                    </label>
                    {#if matchingType === 'CUSTOMER_INVOICE' || matchingType === 'SUPPLIER_INVOICE' || matchingType === 'EXPENSE'}
                      <button
                        type="button"
                        class="candidate-add"
                        onclick={() => addMatchAllocation(candidate)}
                        disabled={isCandidateAllocated(candidate)}
                      >
                        {isCandidateAllocated(candidate) ? 'Added' : 'Add'}
                      </button>
                    {/if}
                  </div>
                {/each}
              {:else}
                <p class="match-hint">No matching records found for this filter.</p>
              {/if}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  {/if}
  {#snippet footer()}
  
      {#if matchingLine && !matchingLine.is_matched}
        <Button
          variant="primary"
          on:click={handleManualMatch}
          loading={matchLoading}
          disabled={(matchAllocations.length === 0 && !matchingCandidateId) || (matchAllocations.length > 0 && Math.abs(matchAllocationRemaining) > 0.001)}
        >
          {matchAllocations.length > 0 ? 'Match Allocations' : 'Match Selected'}
        </Button>
      {/if}
      <Button variant="secondary" on:click={() => showMatchModal = false}>Close</Button>
    
  {/snippet}
</Modal>

<!-- Edit Statement Modal -->
<Modal bind:open={showEditStatementModal} title="Edit Statement" size="md">
  <div class="statement-edit-form">
    <div class="form-row two-col">
      <FormGroup label="Period Start">
        <input type="date" bind:value={editStatementData.period_start} class="form-input" />
      </FormGroup>
      <FormGroup label="Period End">
        <input type="date" bind:value={editStatementData.period_end} class="form-input" />
      </FormGroup>
    </div>
    <div class="form-row two-col">
      <FormGroup label="Opening Balance (BHD)">
        <input type="number" step="0.001" bind:value={editStatementData.opening_balance} class="form-input" />
      </FormGroup>
      <FormGroup label="Closing Balance (BHD)">
        <input type="number" step="0.001" bind:value={editStatementData.closing_balance} class="form-input" />
      </FormGroup>
    </div>
    <div class="form-row">
      <FormGroup label="Status">
        <select bind:value={editStatementData.status} class="form-select">
          <option value="Imported">Imported</option>
          <option value="In Progress">In Progress</option>
          <option value="Reconciled">Reconciled</option>
          <option value="Verified">Verified</option>
        </select>
      </FormGroup>
    </div>
    <div class="form-row">
      <FormGroup label="Notes">
        <textarea bind:value={editStatementData.notes} class="form-input form-textarea" rows="4" placeholder="OCR summary, import context, or reconciliation notes"></textarea>
      </FormGroup>
    </div>
  </div>
  {#snippet footer()}
  
      <Button variant="secondary" on:click={() => showEditStatementModal = false}>Cancel</Button>
      <Button variant="primary" on:click={handleSaveStatement} loading={savingStatement}>
        Save Changes
      </Button>
    
  {/snippet}
</Modal>

<!-- Edit Transaction Line Modal -->
<Modal bind:open={showEditLineModal} title="Edit Transaction" size="md">
  <div class="line-edit-form">
    <div class="form-row">
      <FormGroup label="Transaction Date">
        <input type="date" bind:value={lineFormData.transaction_date} class="form-input" />
      </FormGroup>
    </div>
    <div class="form-row">
      <FormGroup label="Description">
        <Input bind:value={lineFormData.description} placeholder="Transaction description" />
      </FormGroup>
    </div>
    <div class="form-row">
      <FormGroup label="Reference">
        <Input bind:value={lineFormData.reference} placeholder="Reference number (optional)" />
      </FormGroup>
    </div>
    <div class="form-row two-col">
      <FormGroup label="Debit (BHD)">
        <input type="number" step="0.001" bind:value={lineFormData.debit} class="form-input" />
      </FormGroup>
      <FormGroup label="Credit (BHD)">
        <input type="number" step="0.001" bind:value={lineFormData.credit} class="form-input" />
      </FormGroup>
    </div>
    <div class="line-fix-actions">
      <Button variant="secondary" size="sm" on:click={flipLineDebitCredit}>
        Flip Debit / Credit
      </Button>
      {#if editingLine?.is_matched}
        <span class="line-fix-hint">Saving this correction will clear the current match so it can be reviewed again.</span>
      {:else}
        <span class="line-fix-hint">Use this when OCR has placed the amount on the wrong side.</span>
      {/if}
    </div>
    <div class="form-note">
      <p>Note: Only one of Debit or Credit should be non-zero</p>
    </div>
  </div>
  {#snippet footer()}
  
      <Button variant="danger" on:click={handleDeleteLine} loading={savingLine}>
        Delete Transaction
      </Button>
      <div style="flex: 1;"></div>
      <Button variant="secondary" on:click={() => showEditLineModal = false}>Cancel</Button>
      <Button variant="primary" on:click={handleSaveLine} loading={savingLine}>
        Save Changes
      </Button>
    
  {/snippet}
</Modal>

<!-- Add Transaction Line Modal -->
<Modal bind:open={showAddLineModal} title="Add Transaction" size="md">
  <div class="line-edit-form">
    <div class="form-row">
      <FormGroup label="Transaction Date">
        <input type="date" bind:value={lineFormData.transaction_date} class="form-input" />
      </FormGroup>
    </div>
    <div class="form-row">
      <FormGroup label="Description">
        <Input bind:value={lineFormData.description} placeholder="Transaction description" />
      </FormGroup>
    </div>
    <div class="form-row">
      <FormGroup label="Reference">
        <Input bind:value={lineFormData.reference} placeholder="Reference number (optional)" />
      </FormGroup>
    </div>
    <div class="form-row two-col">
      <FormGroup label="Debit (BHD)">
        <input type="number" step="0.001" bind:value={lineFormData.debit} class="form-input" />
      </FormGroup>
      <FormGroup label="Credit (BHD)">
        <input type="number" step="0.001" bind:value={lineFormData.credit} class="form-input" />
      </FormGroup>
    </div>
    <div class="form-note">
      <p>Note: Only one of Debit or Credit should be non-zero</p>
    </div>
  </div>
  {#snippet footer()}
  
      <Button variant="secondary" on:click={() => showAddLineModal = false}>Cancel</Button>
      <Button variant="primary" on:click={handleSaveLine} loading={savingLine}>
        Add Transaction
      </Button>
    
  {/snippet}
</Modal>

<style>
  .loading-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 24px;
    gap: 12px;
  }

  .loading-container.small {
    padding: 16px;
  }

  .close-month-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 4px;
    margin-bottom: 8px;
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

  .handoff-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 12px 16px;
    margin-bottom: 12px;
    background: #D1FAE5;
    border: 1px solid rgba(16, 185, 129, 0.3);
    border-radius: 8px;
    color: #065F46;
    font-size: 13px;
  }

  .manage-accounts-disclosure {
    display: flex;
    justify-content: flex-end;
    margin: -6px 0 12px;
  }

  .manage-accounts-link {
    border: none;
    background: transparent;
    color: var(--text-secondary);
    font-size: 12px;
    cursor: pointer;
    padding: 2px 4px;
    text-decoration: underline;
    text-underline-offset: 2px;
  }

  .manage-accounts-link:hover {
    color: var(--text-primary);
  }

  .import-preview {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .import-preview-summary {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 6px 16px;
    padding: 12px;
    background: var(--surface-elevated, #F8FAFC);
    border-radius: 8px;
  }

  .import-preview-note {
    padding: 10px 12px;
    background: #FEF3C7;
    border: 1px solid #F59E0B;
    border-radius: 6px;
    color: #92400E;
    font-size: 12px;
  }

  .import-preview-note p {
    margin: 0;
  }

  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;
    margin-bottom: 16px;
  }

  .kpi {
    display: flex;
    flex-direction: column;
    gap: 4px;
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
  .kpi-value.warning { color: #F59E0B; }

  .kpi-note {
    color: var(--text-secondary);
    font-size: 11px;
  }

  .cash-position-notice {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    margin: -4px 0 14px;
    padding: 10px 12px;
    border: 1px solid rgba(217, 149, 34, 0.24);
    border-radius: var(--radius-md);
    background: rgba(255, 248, 232, 0.88);
    color: #7a4d0b;
    font-size: 12px;
    line-height: 1.45;
  }

  .cash-position-notice strong {
    white-space: nowrap;
  }

  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    padding: 8px 12px;
    background: var(--surface-elevated);
    border-radius: var(--radius-md);
  }

  .toolbar .left,
  .toolbar .right {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .account-select,
  .form-select {
    padding: 6px 12px;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: var(--surface-default);
    font-size: 14px;
    min-width: 250px;
  }

  .form-input {
    padding: 6px 12px;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: var(--surface-default);
    width: 100%;
  }

  .split-view {
    display: grid;
    grid-template-columns: minmax(520px, 0.95fr) minmax(0, 1.45fr);
    gap: 12px;
    height: calc(100vh - 300px);
    min-height: 400px;
  }

  .panel {
    background: var(--surface-default);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: 16px;
    overflow: auto;
  }

  .panel h3 {
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }

  .panel-header h3 {
    margin: 0;
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 200px;
    color: var(--text-secondary);
  }

  .import-form {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .inline-loading {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: var(--text-secondary);
    margin-left: 12px;
  }

  .import-note {
    padding: 12px;
    background: var(--surface-elevated);
    border-radius: var(--radius-sm);
    margin-top: 8px;
  }

  .import-note p {
    margin: 0;
    font-size: 13px;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .match-details {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px 18px;
    padding: 16px;
    background: var(--surface-elevated);
    border-radius: var(--radius-md);
  }

  .match-row {
    display: flex;
    gap: 12px;
  }

  .match-row .label {
    font-weight: 500;
    color: var(--text-secondary);
    min-width: 140px;
  }

  .match-row .value {
    color: var(--text-primary);
  }

  .match-row .value.credit { color: #10B981; }
  .match-row .value.debit { color: #EF4444; }

  .match-actions {
    margin-top: 12px;
    display: flex;
    gap: 12px;
  }

  .manual-match-form {
    display: flex;
    flex-direction: column;
    gap: 14px;
    width: 100%;
    min-width: 0;
  }

  .ocr-review-banner {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    padding: 12px 14px;
    border: 1px solid rgba(217, 119, 6, 0.24);
    border-radius: var(--radius-md);
    background: linear-gradient(180deg, rgba(255, 251, 235, 0.98) 0%, rgba(255, 247, 237, 0.95) 100%);
  }

  .ocr-review-banner strong {
    display: block;
    font-size: 12px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: #B45309;
    margin-bottom: 4px;
  }

  .ocr-review-banner p {
    margin: 0;
    font-size: 13px;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .candidate-list {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    max-height: 320px;
    overflow-y: auto;
    padding-right: 4px;
  }

  .candidate-option {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 10px;
    padding: 12px;
    border-radius: 14px;
    border: 1px solid var(--border-subtle);
    background: rgba(255, 255, 255, 0.92);
    font-size: 13px;
    line-height: 1.5;
    color: var(--text-primary);
  }

  .candidate-option.in-plan {
    border-color: rgba(16, 185, 129, 0.35);
    background: rgba(240, 253, 244, 0.88);
  }

  .candidate-option label {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    min-width: 0;
    flex: 1;
  }

  .candidate-option input {
    margin-top: 2px;
    flex-shrink: 0;
  }

  .candidate-add,
  .allocation-remove {
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: var(--surface-default);
    color: var(--text-primary);
    font-size: 12px;
    font-weight: 600;
    padding: 5px 9px;
    cursor: pointer;
    flex-shrink: 0;
  }

  .candidate-add:disabled {
    cursor: default;
    color: #059669;
    border-color: rgba(16, 185, 129, 0.35);
    background: rgba(240, 253, 244, 0.9);
  }

  .allocation-panel {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 12px;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    background: var(--surface-elevated);
  }

  .allocation-head,
  .allocation-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 10px;
    align-items: center;
  }

  .allocation-head strong {
    display: block;
    font-size: 13px;
    color: var(--text-primary);
  }

  .allocation-head span,
  .allocation-label small {
    color: var(--text-secondary);
    font-size: 12px;
  }

  .allocation-balance {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    font-family: var(--font-mono);
  }

  .allocation-balance.balanced span:last-child {
    color: #059669;
  }

  .allocation-balance.over span:last-child {
    color: #DC2626;
  }

  .allocation-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .allocation-row {
    grid-template-columns: minmax(0, 1fr) 130px auto;
    padding: 8px;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: rgba(255, 255, 255, 0.82);
  }

  .allocation-label {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .allocation-label span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
  }

  .allocation-input {
    width: 130px;
    padding: 6px 8px;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: var(--surface-default);
    font-family: var(--font-mono);
    text-align: right;
  }

  @media (max-width: 900px) {
    .match-details,
    .candidate-list,
    .allocation-row {
      grid-template-columns: 1fr;
    }

    .allocation-balance {
      align-items: flex-start;
    }
  }

  .match-hint {
    color: var(--text-secondary);
    font-size: 13px;
  }

  :global(.unmatched-row) {
    background: rgba(239, 68, 68, 0.05) !important;
  }

  .form-row {
    display: flex;
    flex-direction: column;
  }

  .form-row.two-col {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .statement-edit-form,
  .line-edit-form {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .statement-notes-card {
    margin-bottom: 12px;
    padding: 12px 14px;
    background: linear-gradient(180deg, rgba(255,255,255,0.96) 0%, rgba(247,247,244,0.98) 100%);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
  }

  .statement-notes-head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    margin-bottom: 8px;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-secondary);
  }

  .statement-notes-card p {
    margin: 0;
    font-size: 13px;
    line-height: 1.6;
    color: var(--text-primary);
    white-space: pre-wrap;
  }

  .form-textarea {
    min-height: 96px;
    resize: vertical;
  }

  .form-note {
    padding: 8px 12px;
    background: var(--surface-elevated);
    border-left: 3px solid var(--brand-indigo);
    border-radius: var(--radius-sm);
  }

  .line-fix-actions {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .line-fix-hint {
    font-size: 12px;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .form-note p {
    margin: 0;
    font-size: 12px;
    color: var(--text-secondary);
  }

  @media (max-width: 1200px) {
    .kpi-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .split-view {
      grid-template-columns: 1fr;
      height: auto;
    }

    .panel {
      max-height: 400px;
    }
  }
</style>
