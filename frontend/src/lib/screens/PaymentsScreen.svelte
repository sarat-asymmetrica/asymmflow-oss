<script lang="ts">
  import { run } from 'svelte/legacy';
  import { motionMs } from "$lib/motion";

  /**
   * PaymentsScreen - AR Money-In Workspace (Receipts + Payment History)
   *
   * Features:
   * - Record receipts: applied to an invoice immediately, or held on-account
   * - Apply an on-account/unapplied receipt balance to an invoice later
   * - Reverse a fully-unapplied (zero-application) receipt
   * - View payment history (invoice payments, created automatically when a
   *   receipt is applied, or edited directly here for legacy records)
   * - Real-time KPI dashboard (Total Collected, This Month, Avg Days, Unapplied)
   * - BHD currency formatting (3 decimal places)
   * - Filter tabs: All, This Month, This Quarter
   * - Embedded mode support for FinanceHub
   *
   * Article III.1: receipts are the ONE AR money-in path. Standalone payment
   * creation was removed — every new payment now originates from a receipt
   * (CreateCustomerReceipt auto-creates the Payment when applied to an invoice).
   * Editing/deleting legacy Payment rows remains supported for backward compat.
   *
   * Design System: Wabi-Sabi minimalism × Bloomberg data density
   */

  import { createEventDispatcher, onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  // Wails API imports
  import {
    GetAllPayments,
    ListCustomers,
    CreateCustomerReceipt,
    ApplyCustomerReceiptToInvoice,
    ListCustomerReceipts,
    ReverseCustomerReceipt } from '../../../wailsjs/go/main/App';
import { GetPayment, UpdatePayment, DeletePayment, ListCustomerInvoices } from '../../../wailsjs/go/main/FinanceService';
  import { main, finance } from '../../../wailsjs/go/models';

  // Security utilities
  import { escapeHtml } from '$lib/utils/escapeHtml';
  import { debounce } from '$lib/utils/debounce';

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
  import { permissions } from '$lib/stores/authContext';
  import { buildWailsInput, toWailsDate } from '$lib/utils/wailsInterop';
  import { playPaidSettle } from '$lib/sound';

  // Wave 10 / B4: same tolerance the backend uses for BHD float comparisons
  // (FloatingPointTolerance, supplier_payment_service.go) — a receipt that
  // brings outstanding to (approximately) zero is a full settlement.
  const PAID_TOLERANCE_BHD = 0.001;

  const dispatch = createEventDispatcher();

  
  interface Props {
    // Props
    embedded?: boolean;
    company?: 'Acme Instrumentation' | 'Beacon Controls';
  }

  let { embedded = false, company = 'Acme Instrumentation' }: Props = $props();

  // Types
  type TimeFilter = 'All' | 'This Month' | 'This Quarter';

  interface Payment {
    id: string;
    invoice_id: string;
    invoice_number: string;
    customer_name?: string;
    amount_bhd: number;
    payment_date: any; // Go Time type serializes as object, not string
    payment_method: string;
    days_to_payment: number;
    reference?: string;
    division?: string;
  }

  interface Invoice {
    id: string;
    invoice_number: string;
    customer_name: string;
    grand_total_bhd: number;
    outstanding_bhd: number;
    status: string;
    invoice_date?: any; // Go Time type serializes as object, not string
    division?: string;
  }

  interface CustomerReceipt {
    id: string;
    receipt_number: string;
    customer_id: string;
    customer_name: string;
    division?: string;
    receipt_date: any; // Go Time type serializes as object, not string
    amount_bhd: number;
    applied_amount_bhd: number;
    unapplied_amount_bhd: number;
    payment_method: string;
    reference?: string;
    status: string; // OnAccount, PartiallyApplied, Applied, Reversed
    notes?: string;
  }

  interface CustomerOption {
    id: string;
    business_name: string;
    customer_code?: string;
  }

  // State
  let payments: Payment[] = $state([]);
  let filteredPayments: Payment[] = $state([]);
  let invoices: Invoice[] = $state([]);
  let receipts: CustomerReceipt[] = $state([]);
  let customers: CustomerOption[] = $state([]);
  let loading = $state(true);
  let selectedFilter: TimeFilter = $state('All');

  // Pagination state
  const PAGE_SIZE = 50;
  let currentPage = 0;
  let hasMore = $state(true);
  let loadingMore = $state(false);
  let totalLoaded = $state(0);

  // Receipts pagination state — receipts are fetched in the same PAGE_SIZE
  // windows as payments (previously a flat ListCustomerReceipts(200, 0) with
  // no follow-up, which silently dropped any receipt past row 200).
  const RECEIPT_PAGE_SIZE = 50;
  let receiptsCurrentPage = 0;
  let receiptsHasMore = $state(true);
  let receiptsLoadingMore = $state(false);
  let receiptsTotalLoaded = $state(0);

  // Modal state — shared by "Record Receipt" (create) and "Edit Payment" (edit).
  // create: records a CustomerReceipt (applied-to-invoice or on-account).
  // edit: updates a legacy/existing Payment row directly.
  let showRecordModal = $state(false);
  let recordLoading = $state(false);
  let paymentModalMode: 'create' | 'edit' = $state('create');
  let editingPaymentId = $state('');
  let selectedInvoiceId = $state('');
  let selectedInvoice: Invoice | null = $state(null);
  let paymentAmount = $state('');
  let paymentMethod = $state('Bank Transfer');
  let paymentDate = $state(new Date().toISOString().split('T')[0]);
  let paymentReference = $state('');
  let invoiceSearchQuery = $state('');
  let debouncedInvoiceSearch = $state('');
  let selectedPaymentId = $state('');
  let selectedPayment: Payment | null = $state(null);
  let deleteConfirmPaymentId = $state('');

  // Receipt-only fields (paymentModalMode === 'create')
  let receiptApplyNow = $state(true); // true = apply to selected invoice now, false = on-account
  let receiptCustomerId = $state('');
  let receiptCustomerName = $state('');
  let receiptNotes = $state('');
  let receiptCustomerSearchQuery = $state(''); // typeahead filter for the on-account customer picker

  // Receipts table selection + row-action state
  let selectedReceiptId = $state('');
  let selectedReceipt: CustomerReceipt | null = $state(null);
  let receiptActionLoading = $state(false);

  // "Apply Unapplied Balance" modal — pre-scoped to the selected receipt
  let showApplyModal = $state(false);
  let applySaving = $state(false);
  let applyReceipt: CustomerReceipt | null = $state(null);
  let applyInvoiceId = $state('');
  let applySelectedInvoice: Invoice | null = $state(null);
  let applyAmount = $state('');

  function matchesCompany(division?: string) {
    return (division || 'Acme Instrumentation') === company;
  }

  // Debounced invoice search (300ms delay)
  const updateDebouncedInvoiceSearch = debounce((value: string) => {
    debouncedInvoiceSearch = value;
  }, 300);


  // Payment method options
  const paymentMethods = [
    'Bank Transfer',
    'Cheque',
    'Cash',
    'Credit Card',
    'LC',
    'PDC',
    'Other',
    'Wire Transfer',
    'Online'
  ];

  // DataTable columns configuration
  const columns = [
    {
      key: 'payment_date',
      label: 'Date',
      type: 'date' as const,
      sortable: true,
      width: '120px',
      render: (row: Payment) => {
        return `<span style="font-family: var(--font-mono); font-size: 13px;">${formatDate(row.payment_date)}</span>`;
      }
    },
    {
      key: 'invoice_number',
      label: 'Invoice #',
      sortable: true,
      width: '140px',
      render: (row: Payment) => {
        return `<span style="font-family: var(--font-mono); font-size: 12px; color: var(--brand-indigo);">${escapeHtml(row.invoice_number || '')}</span>`;
      }
    },
    {
      key: 'customer_name',
      label: 'Customer',
      sortable: true,
      render: (row: Payment) => {
        return `<span style="font-weight: 500;">${escapeHtml(row.customer_name || '—')}</span>`;
      }
    },
    {
      key: 'amount_bhd',
      label: 'Amount (BHD)',
      sortable: true,
      width: '140px',
      align: 'right' as const,
      render: (row: Payment) => {
        return `<span style="font-family: var(--font-mono); font-weight: 600; font-size: 14px;">${formatBHD(row.amount_bhd)}</span>`;
      }
    },
    {
      key: 'payment_method',
      label: 'Method',
      sortable: true,
      width: '140px',
      render: (row: Payment) => {
        const colorMap: Record<string, string> = {
          'Bank Transfer': '#10B981',
          'Wire Transfer': '#3B82F6',
          'Cheque': '#F59E0B',
          'Cash': '#EF4444',
          'Online': '#8B5CF6'
        };
        const color = colorMap[row.payment_method] || '#6B7280';
        return `<span style="color: ${color}; font-size: 12px; font-weight: 500;">${escapeHtml(row.payment_method || 'N/A')}</span>`;
      }
    },
    {
      key: 'days_to_payment',
      label: 'Days to Payment',
      sortable: true,
      width: '140px',
      align: 'center' as const,
      render: (row: Payment) => {
        const days = row.days_to_payment;
        const color = days <= 30 ? '#10B981' : days <= 60 ? '#F59E0B' : '#EF4444';
        return `<span style="font-family: var(--font-mono); font-weight: 600; color: ${color};">${days}d</span>`;
      }
    },
    {
      key: 'reference',
      label: 'Reference',
      sortable: true,
      width: '140px',
      render: (row: Payment) => {
        return `<span style="font-family: var(--font-mono); font-size: 12px; color: var(--text-secondary);">${escapeHtml(row.reference || '—')}</span>`;
      }
    }
  ];

  // Receipts DataTable columns configuration
  const receiptColumns = [
    {
      key: 'receipt_date',
      label: 'Date',
      type: 'date' as const,
      sortable: true,
      width: '110px',
      render: (row: CustomerReceipt) => {
        return `<span style="font-family: var(--font-mono); font-size: 13px;">${formatDate(row.receipt_date)}</span>`;
      }
    },
    {
      key: 'receipt_number',
      label: 'Receipt #',
      sortable: true,
      width: '130px',
      render: (row: CustomerReceipt) => {
        return `<span style="font-family: var(--font-mono); font-size: 12px; color: var(--brand-indigo);">${escapeHtml(row.receipt_number || '')}</span>`;
      }
    },
    {
      key: 'customer_name',
      label: 'Customer',
      sortable: true,
      render: (row: CustomerReceipt) => {
        return `<span style="font-weight: 500;">${escapeHtml(row.customer_name || '—')}</span>`;
      }
    },
    {
      key: 'amount_bhd',
      label: 'Amount (BHD)',
      sortable: true,
      width: '120px',
      align: 'right' as const,
      render: (row: CustomerReceipt) => {
        return `<span style="font-family: var(--font-mono); font-weight: 600; font-size: 13px;">${formatBHD(row.amount_bhd)}</span>`;
      }
    },
    {
      key: 'applied_amount_bhd',
      label: 'Applied (BHD)',
      sortable: true,
      width: '120px',
      align: 'right' as const,
      render: (row: CustomerReceipt) => {
        return `<span style="font-family: var(--font-mono); font-size: 13px; color: var(--text-secondary);">${formatBHD(row.applied_amount_bhd)}</span>`;
      }
    },
    {
      key: 'unapplied_amount_bhd',
      label: 'Unapplied (BHD)',
      sortable: true,
      width: '130px',
      align: 'right' as const,
      render: (row: CustomerReceipt) => {
        const hasBalance = row.unapplied_amount_bhd > 0.001;
        return `<span style="font-family: var(--font-mono); font-weight: ${hasBalance ? 600 : 400}; font-size: 13px; color: ${hasBalance ? '#F59E0B' : 'var(--text-secondary)'};">${formatBHD(row.unapplied_amount_bhd)}</span>`;
      }
    },
    {
      key: 'status',
      label: 'Status',
      sortable: true,
      width: '130px',
      render: (row: CustomerReceipt) => {
        const colorMap: Record<string, string> = {
          'OnAccount': '#3B82F6',
          'PartiallyApplied': '#F59E0B',
          'Applied': '#10B981',
          'Reversed': '#EF4444'
        };
        const color = colorMap[row.status] || '#6B7280';
        return `<span style="color: ${color}; font-size: 12px; font-weight: 600;">${escapeHtml(row.status || '—')}</span>`;
      }
    },
    {
      key: 'payment_method',
      label: 'Method',
      sortable: true,
      width: '120px',
      render: (row: CustomerReceipt) => {
        return `<span style="font-size: 12px; color: var(--text-secondary);">${escapeHtml(row.payment_method || '—')}</span>`;
      }
    },
    {
      key: 'reference',
      label: 'Reference',
      sortable: true,
      width: '130px',
      render: (row: CustomerReceipt) => {
        return `<span style="font-family: var(--font-mono); font-size: 12px; color: var(--text-secondary);">${escapeHtml(row.reference || '—')}</span>`;
      }
    }
  ];

  // Filter tabs
  const filterTabs: { value: TimeFilter; label: string; count: number }[] = $state([
    { value: 'All', label: 'All Payments', count: 0 },
    { value: 'This Month', label: 'This Month', count: 0 },
    { value: 'This Quarter', label: 'This Quarter', count: 0 }
  ]);






  // Currency formatter - BHD with 3 decimal places
  function formatBHD(amount: number): string {
    return new Intl.NumberFormat('en-US', {
      minimumFractionDigits: 3,
      maximumFractionDigits: 3
    }).format(Number(amount) || 0);
  }

  function toInputDate(dateStr: any): string {
    if (!dateStr) return new Date().toISOString().split('T')[0];
    const dateVal = typeof dateStr === 'string' ? dateStr : String(dateStr);
    const date = new Date(dateVal);
    if (isNaN(date.getTime())) return new Date().toISOString().split('T')[0];
    return date.toISOString().split('T')[0];
  }

  function openBankReconciliation() {
    dispatch('navigate', { tab: 'bank_recon', source: 'payments' });
  }

  // Format date - handles both string and Go Time object
  function formatDate(dateStr: any): string {
    if (!dateStr) return '—';
    // Convert to string if it's an object (Go Time type)
    const dateVal = typeof dateStr === 'string' ? dateStr : String(dateStr);
    const date = new Date(dateVal);
    if (isNaN(date.getTime())) return '—';
    return date.toLocaleDateString('en-GB', {
      day: '2-digit',
      month: 'short',
      year: 'numeric'
    });
  }

  // Check if date is this month - handles both string and Go Time object
  function isThisMonth(dateStr: any): boolean {
    if (!dateStr) return false;
    const dateVal = typeof dateStr === 'string' ? dateStr : String(dateStr);
    const date = new Date(dateVal);
    if (isNaN(date.getTime())) return false;
    const now = new Date();
    return date.getMonth() === now.getMonth() && date.getFullYear() === now.getFullYear();
  }

  // Check if date is this quarter - handles both string and Go Time object
  function isThisQuarter(dateStr: any): boolean {
    if (!dateStr) return false;
    const dateVal = typeof dateStr === 'string' ? dateStr : String(dateStr);
    const date = new Date(dateVal);
    if (isNaN(date.getTime())) return false;
    const now = new Date();
    const currentQuarter = Math.floor(now.getMonth() / 3);
    const dateQuarter = Math.floor(date.getMonth() / 3);
    return dateQuarter === currentQuarter && date.getFullYear() === now.getFullYear();
  }

  function isCurrentYear(dateStr: any): boolean {
    if (!dateStr) return false;
    const dateVal = typeof dateStr === 'string' ? dateStr : String(dateStr);
    const date = new Date(dateVal);
    if (isNaN(date.getTime())) return false;
    return date.getFullYear() === new Date().getFullYear();
  }

  // Load payments and unpaid invoices with pagination
  async function loadData() {
    loading = true;
    currentPage = 0;
    hasMore = true;
    receiptsCurrentPage = 0;
    receiptsHasMore = true;
    try {
      const [paymentsData, invoicesData, receiptsData, customersData] = await Promise.all([
        GetAllPayments(PAGE_SIZE, 0),
        ListCustomerInvoices(500, 0), // Load more invoices for dropdown
        ListCustomerReceipts(RECEIPT_PAGE_SIZE, 0),
        ListCustomers(500, 0)
      ]);

      payments = (paymentsData || []).filter((payment) => matchesCompany(payment.division));
      if (selectedPaymentId) {
        selectedPayment = payments.find((payment) => payment.id === selectedPaymentId) || null;
        if (!selectedPayment) {
          selectedPaymentId = '';
          deleteConfirmPaymentId = '';
        }
      }
      currentPage = 1;
      totalLoaded = payments.length;
      hasMore = payments.length === PAGE_SIZE;

      // Filter to only unpaid/partially paid invoices for dropdown
      invoices = (invoicesData || []).filter(
        inv => matchesCompany(inv.division) && inv.outstanding_bhd > 0
      );

      receipts = (receiptsData || []).filter((r) => matchesCompany(r.division));
      receiptsCurrentPage = 1;
      receiptsTotalLoaded = receipts.length;
      receiptsHasMore = receipts.length === RECEIPT_PAGE_SIZE;
      if (selectedReceiptId) {
        selectedReceipt = receipts.find((r) => r.id === selectedReceiptId) || null;
        if (!selectedReceipt) {
          selectedReceiptId = '';
        }
      }

      // Customers aren't division-scoped in the data model — offered as-is
      // for on-account receipt attribution.
      customers = (customersData || [])
        .map((c) => ({ id: c.id, business_name: c.business_name, customer_code: c.customer_code }))
        .filter((c) => c.business_name);

      console.log(`Loaded ${payments.length} payments (page 1), ${invoices.length} unpaid invoices, ${receipts.length} receipts`);
    } catch (err) {
      console.error('Failed to load payment data:', err);
      toast.danger('Failed to load payment data');
      payments = [];
      invoices = [];
      receipts = [];
      hasMore = false;
      receiptsHasMore = false;
    } finally {
      loading = false;
    }
  }

  // Load more payments (pagination)
  async function loadMore() {
    if (loadingMore || !hasMore) return;

    loadingMore = true;
    try {
      const offset = currentPage * PAGE_SIZE;
      const data = await GetAllPayments(PAGE_SIZE, offset);

      if (data && data.length > 0) {
        payments = [...payments, ...data.filter((payment) => matchesCompany(payment.division))];
        currentPage++;
        totalLoaded = payments.length;
        hasMore = data.length === PAGE_SIZE;
        console.log(`Loaded ${data.length} more payments (total: ${totalLoaded})`);
      } else {
        hasMore = false;
      }
    } catch (err) {
      console.error('Failed to load more payments:', err);
      toast.danger('Failed to load more payments');
    } finally {
      loadingMore = false;
    }
  }

  // Load more receipts (pagination) — mirrors loadMore() above so the
  // receipts list can scale past RECEIPT_PAGE_SIZE without a silent cap.
  async function loadMoreReceipts() {
    if (receiptsLoadingMore || !receiptsHasMore) return;

    receiptsLoadingMore = true;
    try {
      const offset = receiptsCurrentPage * RECEIPT_PAGE_SIZE;
      const data = await ListCustomerReceipts(RECEIPT_PAGE_SIZE, offset);

      if (data && data.length > 0) {
        receipts = [...receipts, ...data.filter((r) => matchesCompany(r.division))];
        receiptsCurrentPage++;
        receiptsTotalLoaded = receipts.length;
        receiptsHasMore = data.length === RECEIPT_PAGE_SIZE;
        console.log(`Loaded ${data.length} more receipts (total: ${receiptsTotalLoaded})`);
      } else {
        receiptsHasMore = false;
      }
    } catch (err) {
      console.error('Failed to load more receipts:', err);
      toast.danger('Failed to load more receipts');
    } finally {
      receiptsLoadingMore = false;
    }
  }

  // Open the "Record Receipt" modal — the one AR money-in creation path.
  function openReceiptModal() {
    if (!canCreate) {
      toast.warning('You do not have permission to record customer receipts');
      return;
    }

    paymentModalMode = 'create';
    editingPaymentId = '';
    // Reset shared form fields
    selectedInvoiceId = '';
    selectedInvoice = null;
    paymentAmount = '';
    paymentMethod = 'Bank Transfer';
    paymentDate = new Date().toISOString().split('T')[0];
    paymentReference = '';
    invoiceSearchQuery = '';
    // Reset receipt-only fields
    receiptApplyNow = true;
    receiptCustomerId = '';
    receiptCustomerName = '';
    receiptNotes = '';
    receiptCustomerSearchQuery = '';

    showRecordModal = true;
  }

  // Switching between "apply to invoice" and "on-account" clears the fields
  // that only apply to the other branch, so a stale selection can't leak in.
  function setReceiptApplyNow(applyNow: boolean) {
    receiptApplyNow = applyNow;
    if (applyNow) {
      receiptCustomerId = '';
      receiptCustomerSearchQuery = '';
    } else {
      selectedInvoiceId = '';
      selectedInvoice = null;
      invoiceSearchQuery = '';
      paymentAmount = '';
    }
  }

  function handlePaymentRowClick(row: Payment) {
    if (selectedPaymentId === row.id) {
      selectedPaymentId = '';
      selectedPayment = null;
      deleteConfirmPaymentId = '';
      return;
    }

    selectedPaymentId = row.id;
    selectedPayment = row;
    deleteConfirmPaymentId = '';
  }

  function handleReceiptRowClick(row: CustomerReceipt) {
    if (selectedReceiptId === row.id) {
      selectedReceiptId = '';
      selectedReceipt = null;
      return;
    }

    selectedReceiptId = row.id;
    selectedReceipt = row;
  }

  // Invoices open to a given receipt's customer/division — used both to
  // suggest an invoice when opening the apply modal and to populate its
  // dropdown. Best-effort match on customer name (the frontend Invoice type
  // doesn't carry customer_id); the backend re-validates the true match.
  function eligibleInvoicesForReceipt(receipt: CustomerReceipt): Invoice[] {
    return invoices.filter(
      (inv) =>
        inv.customer_name === receipt.customer_name &&
        (inv.division || 'Acme Instrumentation') === (receipt.division || 'Acme Instrumentation') &&
        inv.outstanding_bhd > 0
    );
  }

  // Open the "Apply Unapplied Balance" modal, pre-scoped to the selected
  // receipt with a suggested invoice + amount already filled in.
  function openApplyModal() {
    if (!selectedReceipt || !canCreate) return;
    const receipt = selectedReceipt;
    const eligible = eligibleInvoicesForReceipt(receipt);
    const suggested = eligible[0] || null;

    applyReceipt = receipt;
    applyInvoiceId = suggested?.id || '';
    applySelectedInvoice = suggested;
    applyAmount = suggested
      ? Math.min(receipt.unapplied_amount_bhd, suggested.outstanding_bhd).toFixed(3)
      : receipt.unapplied_amount_bhd.toFixed(3);
    showApplyModal = true;
  }

  function selectFullApplyAmount() {
    if (applySelectedInvoice && applyReceipt) {
      applyAmount = Math.min(applyReceipt.unapplied_amount_bhd, applySelectedInvoice.outstanding_bhd).toFixed(3);
    }
  }

  async function openEditModal() {
    if (!selectedPayment || !canUpdate) {
      return;
    }

    recordLoading = true;
    try {
      const payment = await GetPayment(selectedPayment.id);
      paymentModalMode = 'edit';
      editingPaymentId = payment.id;
      selectedInvoiceId = payment.invoice_id || selectedPayment.invoice_id || '';
      selectedInvoice = {
        id: selectedInvoiceId,
        invoice_number: payment.invoice_number || selectedPayment.invoice_number,
        customer_name: selectedPayment.customer_name || '—',
        grand_total_bhd: 0,
        outstanding_bhd: 0,
        status: '',
        invoice_date: null
      };
      paymentAmount = Number(payment.amount_bhd || 0).toFixed(3);
      paymentMethod = payment.payment_method || 'Bank Transfer';
      paymentDate = toInputDate(payment.payment_date);
      paymentReference = payment.reference || '';
      showRecordModal = true;
    } catch (err) {
      console.error('Failed to load payment for editing:', err);
      toast.danger('Failed to load payment details');
    } finally {
      recordLoading = false;
    }
  }

  async function handleDeleteSelectedPayment() {
    if (!selectedPayment || !canDelete) {
      return;
    }

    if (deleteConfirmPaymentId !== selectedPayment.id) {
      deleteConfirmPaymentId = selectedPayment.id;
      return;
    }

    recordLoading = true;
    try {
      await DeletePayment(selectedPayment.id);
      toast.success(`Deleted payment ${selectedPayment.invoice_number || ''}`.trim());
      selectedPaymentId = '';
      selectedPayment = null;
      deleteConfirmPaymentId = '';
      await loadData();
    } catch (err) {
      console.error('Failed to delete payment:', err);
      toast.danger(`Failed to delete payment: ${String(err)}`);
    } finally {
      recordLoading = false;
    }
  }

  // Record a receipt (create mode) or save an edited legacy payment (edit mode).
  // This is the single submit handler for the shared modal.
  async function handleModalSubmit() {
    if (paymentModalMode === 'create') {
      await handleRecordReceipt();
      return;
    }
    await handleUpdatePayment();
  }

  async function handleRecordReceipt() {
    const amount = parseFloat(paymentAmount);
    if (isNaN(amount) || amount <= 0) {
      toast.warning('Please enter a valid receipt amount');
      return;
    }

    if (receiptApplyNow) {
      if (!selectedInvoiceId || !selectedInvoice) {
        toast.warning('Please select an invoice to apply the receipt to');
        return;
      }
      if (amount > selectedInvoice.outstanding_bhd) {
        toast.warning(`Receipt amount (${formatBHD(amount)} BHD) exceeds outstanding balance (${formatBHD(selectedInvoice.outstanding_bhd)} BHD)`);
        return;
      }
    } else if (!receiptCustomerId) {
      toast.warning('Please select a customer for the on-account receipt');
      return;
    }

    if (!paymentDate) {
      toast.warning('Please select a receipt date');
      return;
    }

    if ((paymentMethod === 'Bank Transfer' || paymentMethod === 'Wire Transfer') && !paymentReference.trim()) {
      toast.warning('Reference is required for bank transfers');
      return;
    }

    // Wave 10 / B4: decide up front — using only the client-known
    // outstanding balance and the amount the user is about to post — whether
    // this receipt will fully settle the invoice. This mirrors the same
    // condition the backend's settlement policy uses (outstanding <=
    // FloatingPointTolerance after the payment is applied). Computed BEFORE
    // the async post so it can never be affected by loadData() resetting
    // selectedInvoice afterwards, and only acted on after the post succeeds
    // (never on a partial payment, never on an error).
    const willFullySettle =
      receiptApplyNow &&
      !!selectedInvoice &&
      (selectedInvoice.outstanding_bhd - amount) <= PAID_TOLERANCE_BHD;

    recordLoading = true;
    try {
      await CreateCustomerReceipt(buildWailsInput(main.CustomerReceiptInput, {
        customer_id: receiptApplyNow ? '' : receiptCustomerId,
        customer_name: receiptApplyNow ? '' : receiptCustomerName,
        invoice_id: receiptApplyNow ? selectedInvoiceId : '',
        amount_bhd: amount,
        receipt_date: paymentDate,
        payment_method: paymentMethod,
        reference: paymentReference.trim(),
        division: company,
        notes: receiptNotes.trim(),
      }));
      toast.success(
        receiptApplyNow
          ? `Receipt of ${formatBHD(amount)} BHD recorded and applied`
          : `Receipt of ${formatBHD(amount)} BHD recorded on-account`
      );

      // Wave 10 / B4 — the one sound. Fires only when this posting click
      // just fully applied the invoice (never on partial payments, never
      // on-account, never on error — this line is unreachable unless the
      // await above resolved without throwing).
      if (willFullySettle) {
        playPaidSettle();
      }

      showRecordModal = false;
      await loadData();
    } catch (err) {
      console.error('Failed to record receipt:', err);
      toast.danger(`Failed to record receipt: ${String(err)}`);
    } finally {
      recordLoading = false;
    }
  }

  async function handleUpdatePayment() {
    const amount = parseFloat(paymentAmount);
    if (isNaN(amount) || amount <= 0) {
      toast.warning('Please enter a valid payment amount');
      return;
    }

    if (!paymentDate) {
      toast.warning('Please select a payment date');
      return;
    }

    if ((paymentMethod === 'Bank Transfer' || paymentMethod === 'Wire Transfer') && !paymentReference.trim()) {
      toast.warning('Reference is required for bank transfers');
      return;
    }

    recordLoading = true;
    try {
      const existingPayment = await GetPayment(editingPaymentId);
      const originalDate = toWailsDate(existingPayment.payment_date) || new Date();
      const nextDate = new Date(`${paymentDate}T00:00:00`);
      const inferredInvoiceDate = new Date(originalDate.getTime() - ((existingPayment.days_to_payment || 0) * 24 * 60 * 60 * 1000));
      const recalculatedDaysToPayment = Math.max(
        0,
        Math.round((nextDate.getTime() - inferredInvoiceDate.getTime()) / (24 * 60 * 60 * 1000))
      );

      await UpdatePayment(editingPaymentId, buildWailsInput(finance.Payment, {
        ...existingPayment,
        amount_bhd: amount,
        payment_method: paymentMethod,
        payment_date: nextDate.toISOString(),
        reference: paymentReference.trim(),
        days_to_payment: recalculatedDaysToPayment,
      }));
      toast.success(`Payment updated to ${formatBHD(amount)} BHD`);

      showRecordModal = false;
      deleteConfirmPaymentId = '';

      // Reload data to reflect changes
      await loadData();
    } catch (err) {
      console.error('Failed to update payment:', err);
      toast.danger('Failed to update payment. Please try again.');
    } finally {
      recordLoading = false;
    }
  }

  // Handle quick amount selection (full outstanding) — Edit Payment modal
  function selectFullOutstanding() {
    if (selectedInvoice) {
      paymentAmount = selectedInvoice.outstanding_bhd.toFixed(3);
    }
  }

  // Apply an unapplied receipt balance to the pre-scoped invoice
  async function handleApplyReceipt() {
    if (!applyReceipt || !applyInvoiceId) {
      toast.warning('Please select an invoice');
      return;
    }

    const amount = parseFloat(applyAmount);
    if (isNaN(amount) || amount <= 0) {
      toast.warning('Please enter a valid amount');
      return;
    }

    const cap = applySelectedInvoice
      ? Math.min(applyReceipt.unapplied_amount_bhd, applySelectedInvoice.outstanding_bhd)
      : applyReceipt.unapplied_amount_bhd;
    if (amount > cap + 0.001) {
      toast.warning(`Amount exceeds available balance (${formatBHD(cap)} BHD)`);
      return;
    }

    applySaving = true;
    try {
      await ApplyCustomerReceiptToInvoice(applyReceipt.id, applyInvoiceId, amount);
      toast.success(`Applied ${formatBHD(amount)} BHD from receipt ${applyReceipt.receipt_number}`);
      showApplyModal = false;
      applyReceipt = null;
      selectedReceiptId = '';
      selectedReceipt = null;
      await loadData();
    } catch (err) {
      console.error('Failed to apply receipt:', err);
      toast.danger(`Failed to apply receipt: ${String(err)}`);
    } finally {
      applySaving = false;
    }
  }

  // Reverse the selected receipt. Only fully-unapplied (zero-application)
  // receipts can be reversed here — applied/posted receipt reversal is
  // stop-and-report and is intentionally NOT implemented.
  async function handleReverseReceipt() {
    if (!selectedReceipt || !canCreate) return;
    const receipt = selectedReceipt;

    if (receipt.applied_amount_bhd > 0.001) {
      toast.warning('Only fully unapplied receipts can be reversed here');
      return;
    }
    if (receipt.status === 'Reversed') {
      toast.warning('Receipt is already reversed');
      return;
    }

    const result = await confirm.askForReason({
      title: 'Reverse Receipt',
      message: `Reverse receipt ${receipt.receipt_number} for ${formatBHD(receipt.amount_bhd)} BHD? This cannot be undone.`,
      confirmLabel: 'Reverse Receipt',
      variant: 'danger',
      reasonLabel: 'Reason for reversal',
      reasonRequired: true,
    });
    if (!result.confirmed) return;

    receiptActionLoading = true;
    try {
      await ReverseCustomerReceipt(receipt.id, result.reason);
      toast.success(`Receipt ${receipt.receipt_number} reversed`);
      selectedReceiptId = '';
      selectedReceipt = null;
      await loadData();
    } catch (err) {
      console.error('Failed to reverse receipt:', err);
      toast.danger(`Failed to reverse receipt: ${String(err)}`);
    } finally {
      receiptActionLoading = false;
    }
  }

  onMount(() => {
    if (!canView) return;
    loadData();
  });

  let permissionList = $derived(Array.isArray($permissions) ? $permissions : []);
  let canView = $derived(permissionList.includes('*') || permissionList.includes('finance:view') || permissionList.includes('finance:*'));
  let canCreate = $derived(permissionList.includes('*') || permissionList.includes('payments:create') || permissionList.includes('payments:*') || permissionList.includes('finance:*'));
  let canUpdate = $derived(permissionList.includes('*') || permissionList.includes('payments:update') || permissionList.includes('payments:*') || permissionList.includes('finance:*'));
  let canDelete = $derived(permissionList.includes('*') || permissionList.includes('payments:delete') || permissionList.includes('payments:*') || permissionList.includes('finance:*'));
  // Watch invoiceSearchQuery changes and debounce
  run(() => {
    updateDebouncedInvoiceSearch(invoiceSearchQuery);
  });
  // Computed: Update tab counts
  run(() => {
    filterTabs[0].count = payments.length;
    filterTabs[1].count = payments.filter(p => isThisMonth(p.payment_date)).length;
    filterTabs[2].count = payments.filter(p => isThisQuarter(p.payment_date)).length;
  });
  // Computed: Filter payments by time
  run(() => {
    let result = [...payments];

    if (selectedFilter === 'This Month') {
      result = result.filter(p => isThisMonth(p.payment_date));
    } else if (selectedFilter === 'This Quarter') {
      result = result.filter(p => isThisQuarter(p.payment_date));
    }

    filteredPayments = result.sort((a, b) =>
      new Date(b.payment_date).getTime() - new Date(a.payment_date).getTime()
    );
  });
  // Computed: KPI stats
  let stats = $derived({
    totalCollectedYTD: payments.filter((p) => isCurrentYear(p.payment_date)).reduce((sum, p) => sum + p.amount_bhd, 0),
    paymentsThisMonth: payments.filter(p => isThisMonth(p.payment_date)).length,
    avgDaysToPayment: filteredPayments.length > 0
      ? filteredPayments.reduce((sum, p) => sum + p.days_to_payment, 0) / filteredPayments.length
      : 0,
    count: filteredPayments.length,
    totalUnapplied: receipts
      .filter((r) => r.status !== 'Reversed')
      .reduce((sum, r) => sum + r.unapplied_amount_bhd, 0)
  });
  // Computed: Update selected invoice when dropdown changes (Record Receipt,
  // apply-now branch only — on-account and Edit Payment don't drive this).
  run(() => {
    if (paymentModalMode === 'create' && receiptApplyNow) {
      if (selectedInvoiceId) {
        selectedInvoice = invoices.find(inv => inv.id === selectedInvoiceId) || null;
        if (selectedInvoice) {
          // Pre-fill receipt amount with outstanding balance
          paymentAmount = selectedInvoice.outstanding_bhd.toFixed(3);
        }
      } else {
        selectedInvoice = null;
        paymentAmount = '';
      }
    }
  });
  // Keep receiptCustomerName in sync with the on-account customer dropdown
  run(() => {
    receiptCustomerName = customers.find(c => c.id === receiptCustomerId)?.business_name || '';
  });
  // Filter invoices for dropdown search (debounced for performance)
  let filteredInvoicesForDropdown = $derived(debouncedInvoiceSearch
    ? invoices.filter(inv => {
        const query = debouncedInvoiceSearch.toLowerCase();
        return (
          inv.invoice_number?.toLowerCase().includes(query) ||
          inv.customer_name?.toLowerCase().includes(query)
        );
      })
    : invoices);
  // Filter customers for the on-account receipt picker (typeahead by name or
  // customer code). Not company-scoped — CustomerMaster has no division/company
  // field in the data model, so the picker offers all customers as-is.
  let filteredCustomersForDropdown = $derived(receiptCustomerSearchQuery
    ? customers.filter(c => {
        const query = receiptCustomerSearchQuery.toLowerCase();
        return (
          c.business_name?.toLowerCase().includes(query) ||
          c.customer_code?.toLowerCase().includes(query)
        );
      })
    : customers);
  // Invoices eligible for the "Apply Unapplied Balance" modal's selected receipt
  let applyEligibleInvoices = $derived(applyReceipt ? eligibleInvoicesForReceipt(applyReceipt) : []);
  // Keep the apply modal's selected invoice + suggested amount in sync with the dropdown
  run(() => {
    if (showApplyModal && applyReceipt) {
      if (applyInvoiceId) {
        applySelectedInvoice = applyEligibleInvoices.find(inv => inv.id === applyInvoiceId) || null;
        if (applySelectedInvoice) {
          applyAmount = Math.min(applyReceipt.unapplied_amount_bhd, applySelectedInvoice.outstanding_bhd).toFixed(3);
        }
      } else {
        applySelectedInvoice = null;
      }
    }
  });
  run(() => {
    if (company && canView) {
      loadData();
    }
  });
</script>

<PageLayout
  title="Payments"
  subtitle="Payment recording and history management"
  {embedded}
>
  <!-- @migration-task: migrate this slot by hand, `header-actions` is an invalid identifier -->
  <svelte:fragment slot="header-actions">
    <Button variant="secondary" on:click={loadData}>
      Refresh
    </Button>
    <Button variant="secondary" on:click={openBankReconciliation}>
      Open Bank Recon
    </Button>
    <Button variant="primary" on:click={openReceiptModal} disabled={!canCreate}>
      + Record Receipt
    </Button>
  </svelte:fragment>

  <div class="payments-container">
    <!-- KPI Stats -->
    <div class="kpi-grid">
      <Card padding="md" variant="elevated">
        <div class="kpi-card">
          <div class="kpi-label">Total Collected (YTD)</div>
          <div class="kpi-value kpi-success">{formatBHD(stats.totalCollectedYTD)} <span class="kpi-currency">BHD</span></div>
          <div class="kpi-sublabel">{new Date().getFullYear()} collections</div>
        </div>
      </Card>

      <Card padding="md" variant="elevated">
        <div class="kpi-card">
          <div class="kpi-label">Payments This Month</div>
          <div class="kpi-value">{stats.paymentsThisMonth}</div>
          <div class="kpi-sublabel">Current month activity</div>
        </div>
      </Card>

      <Card padding="md" variant="elevated">
        <div class="kpi-card">
          <div class="kpi-label">Avg Days to Payment</div>
          <div class="kpi-value" class:kpi-success={stats.avgDaysToPayment <= 30} class:kpi-warning={stats.avgDaysToPayment > 30 && stats.avgDaysToPayment <= 60} class:kpi-danger={stats.avgDaysToPayment > 60}>
            {stats.avgDaysToPayment.toFixed(1)}<span class="kpi-unit">d</span>
          </div>
          <div class="kpi-sublabel">Payment velocity</div>
        </div>
      </Card>

      <Card padding="md" variant="elevated">
        <div class="kpi-card">
          <div class="kpi-label">Unapplied / On-Account</div>
          <div class="kpi-value" class:kpi-warning={stats.totalUnapplied > 0}>{formatBHD(stats.totalUnapplied)} <span class="kpi-currency">BHD</span></div>
          <div class="kpi-sublabel">Held receipt balances</div>
        </div>
      </Card>
    </div>

    <!-- Receipts — the AR money-in surface -->
    <div class="section-header">
      <h2 class="section-title">Receipts</h2>
      <p class="section-subtitle">Record money in, applied to an invoice or held on-account, and apply held balances later.</p>
    </div>
    <Card padding="sm">
      {#if selectedReceipt}
        <div class="selection-bar">
          <div class="selection-copy">
            <span class="selection-label">Selected Receipt</span>
            <span class="selection-title">{selectedReceipt.receipt_number} · {selectedReceipt.customer_name}</span>
            <span class="selection-meta">{formatBHD(selectedReceipt.amount_bhd)} BHD · {selectedReceipt.status} · Unapplied {formatBHD(selectedReceipt.unapplied_amount_bhd)} BHD</span>
          </div>
          <div class="selection-actions">
            {#if canCreate && selectedReceipt.unapplied_amount_bhd > 0.001 && selectedReceipt.status !== 'Reversed'}
              <Button variant="secondary" size="sm" on:click={openApplyModal} disabled={receiptActionLoading}>
                Apply Unapplied Balance
              </Button>
            {/if}
            {#if canCreate && selectedReceipt.applied_amount_bhd <= 0.001 && selectedReceipt.status !== 'Reversed'}
              <Button variant="ghost" size="sm" on:click={handleReverseReceipt} disabled={receiptActionLoading}>
                Reverse
              </Button>
            {/if}
          </div>
        </div>
      {/if}

      {#if loading}
        <div class="loading-container">
          <WabiSpinner size="lg" />
          <p>Loading receipts...</p>
        </div>
      {:else}
        <DataTable
          columns={receiptColumns}
          data={receipts}
          {loading}
          emptyMessage="No receipts recorded yet. Click 'Record Receipt' to add one."
          selectedId={selectedReceiptId}
          onRowClick={handleReceiptRowClick}
          stickyHeader={!embedded}
          maxHeight={embedded ? "320px" : "360px"}
          showBorder={false}
        />
      {/if}
    </Card>

    <!-- Receipts Pagination Controls -->
    {#if receiptsHasMore && !loading}
      <div class="pagination-controls">
        <button
          class="load-more-btn"
          onclick={loadMoreReceipts}
          disabled={receiptsLoadingMore}
          aria-label={receiptsLoadingMore ? 'Loading more receipts' : `Load more receipts, ${receiptsTotalLoaded} currently loaded`}
        >
          {#if receiptsLoadingMore}
            Loading more...
          {:else}
            Load More Receipts ({receiptsTotalLoaded} loaded)
          {/if}
        </button>
      </div>
    {/if}

    {#if !receiptsHasMore && receiptsTotalLoaded > 0 && !loading}
      <p class="all-loaded">All {receiptsTotalLoaded} receipts loaded</p>
    {/if}

    <!-- Filter Tabs -->
    <Card padding="sm">
      <div class="filter-tabs" role="tablist" aria-label="Filter payments by time">
        {#each filterTabs as tab}
          <button
            class="filter-tab"
            class:active={selectedFilter === tab.value}
            role="tab"
            aria-selected={selectedFilter === tab.value}
            onclick={() => selectedFilter = tab.value}
          >
            {tab.label}
            <span class="tab-count">{tab.count}</span>
          </button>
        {/each}
      </div>
    </Card>

    <!-- Payment History -->
    <div class="section-header">
      <h2 class="section-title">Payment History</h2>
      <p class="section-subtitle">Invoice payments — created automatically when a receipt is applied, or edited here for legacy records.</p>
    </div>
    <Card padding="sm">
      {#if selectedPayment}
        <div class="selection-bar">
          <div class="selection-copy">
            <span class="selection-label">Selected Payment</span>
            <span class="selection-title">{selectedPayment.invoice_number} · {selectedPayment.customer_name || 'Customer payment'}</span>
            <span class="selection-meta">{formatBHD(selectedPayment.amount_bhd)} BHD on {formatDate(selectedPayment.payment_date)}</span>
          </div>
          <div class="selection-actions">
            {#if canUpdate}
              <Button variant="secondary" size="sm" on:click={openEditModal} disabled={recordLoading}>
                Edit
              </Button>
            {/if}
            {#if canDelete}
              <Button variant={deleteConfirmPaymentId === selectedPayment.id ? 'primary' : 'ghost'} size="sm" on:click={handleDeleteSelectedPayment} disabled={recordLoading}>
                {deleteConfirmPaymentId === selectedPayment.id ? 'Confirm Delete' : 'Delete'}
              </Button>
            {/if}
          </div>
        </div>
      {/if}

      {#if loading}
        <div class="loading-container">
          <WabiSpinner size="lg" />
          <p>Loading payments...</p>
        </div>
      {:else}
        <DataTable
          {columns}
          data={filteredPayments}
          {loading}
          emptyMessage="No payments recorded yet. Apply a receipt to an invoice to create one."
          selectedId={selectedPaymentId}
          onRowClick={handlePaymentRowClick}
          stickyHeader={!embedded}
          maxHeight={embedded ? "400px" : "calc(100vh - 480px)"}
          showBorder={false}
        />
      {/if}
    </Card>

    <!-- Pagination Controls -->
    {#if hasMore && !loading}
      <div class="pagination-controls">
        <button
          class="load-more-btn"
          onclick={loadMore}
          disabled={loadingMore}
          aria-label={loadingMore ? 'Loading more payments' : `Load more payments, ${totalLoaded} currently loaded`}
        >
          {#if loadingMore}
            Loading more...
          {:else}
            Load More ({totalLoaded} loaded)
          {/if}
        </button>
      </div>
    {/if}

    {#if !hasMore && totalLoaded > 0 && !loading}
      <p class="all-loaded">All {totalLoaded} payments loaded</p>
    {/if}
  </div>
</PageLayout>

<!-- Record Receipt / Edit Payment Modal -->
{#if showRecordModal}
  <Modal
    title={paymentModalMode === 'create' ? 'Record Receipt' : 'Edit Payment'}
    open={showRecordModal}
    on:close={() => {
      showRecordModal = false;
      paymentModalMode = 'create';
      editingPaymentId = '';
    }}
    size="md"
  >
    <div class="record-form">
      {#if paymentModalMode === 'create'}
        <!-- Receipt type: apply to an invoice now, or hold on-account -->
        <FormGroup label="Receipt Type" required>
          <div class="toggle-group" role="tablist" aria-label="Receipt type">
            <button type="button" class="toggle-btn" class:active={receiptApplyNow} onclick={() => setReceiptApplyNow(true)} disabled={recordLoading}>
              Apply to Invoice
            </button>
            <button type="button" class="toggle-btn" class:active={!receiptApplyNow} onclick={() => setReceiptApplyNow(false)} disabled={recordLoading}>
              On Account
            </button>
          </div>
        </FormGroup>
      {/if}

      <!-- Invoice Selection with Search (create + apply-now) or Customer (create + on-account) or locked display (edit) -->
      <FormGroup label={paymentModalMode === 'create' && !receiptApplyNow ? 'Customer' : 'Select Invoice'} required>
        {#if paymentModalMode === 'create' && receiptApplyNow}
          <!-- Search input -->
          <Input
            type="text"
            bind:value={invoiceSearchQuery}
            placeholder="Search by invoice # or customer..."
            disabled={recordLoading}
          />

          <!-- Filtered dropdown -->
          <select
            class="select-input"
            bind:value={selectedInvoiceId}
            disabled={recordLoading}
            style="margin-top: 8px;"
          >
            <option value="">-- Select an unpaid invoice ({filteredInvoicesForDropdown.length} shown) --</option>
            {#each filteredInvoicesForDropdown as invoice}
              <option value={invoice.id}>
                {invoice.invoice_number} - {invoice.customer_name} ({formatBHD(invoice.outstanding_bhd)} BHD)
              </option>
            {/each}
          </select>

          {#if invoiceSearchQuery && filteredInvoicesForDropdown.length === 0}
            <p class="no-results" style="color: var(--text-muted); margin-top: 4px; font-size: 0.85rem;">
              No invoices match "{invoiceSearchQuery}"
            </p>
          {/if}
        {:else if paymentModalMode === 'create'}
          <!-- Search input (typeahead) -->
          <Input
            type="text"
            bind:value={receiptCustomerSearchQuery}
            placeholder="Search by customer name or code..."
            disabled={recordLoading}
          />

          <!-- Filtered dropdown -->
          <select
            class="select-input"
            bind:value={receiptCustomerId}
            disabled={recordLoading}
            style="margin-top: 8px;"
          >
            <option value="">-- Select a customer ({filteredCustomersForDropdown.length} shown) --</option>
            {#each filteredCustomersForDropdown as c}
              <option value={c.id}>{c.business_name}{c.customer_code ? ` (${c.customer_code})` : ''}</option>
            {/each}
          </select>

          {#if receiptCustomerSearchQuery && filteredCustomersForDropdown.length === 0}
            <p class="no-results" style="color: var(--text-muted); margin-top: 4px; font-size: 0.85rem;">
              No customers match "{receiptCustomerSearchQuery}"
            </p>
          {/if}
        {:else if selectedPayment}
          <div class="locked-field">
            <div class="locked-title">{selectedPayment.invoice_number}</div>
            <div class="locked-subtitle">{selectedPayment.customer_name || 'Customer payment'}</div>
          </div>
        {/if}
      </FormGroup>

      <!-- Invoice Details Display -->
      {#if selectedInvoice}
        <div class="invoice-details" transition:fade={{ duration: motionMs(200) }}>
          <div class="detail-row">
            <span class="detail-label">Customer:</span>
            <span class="detail-value">{selectedInvoice.customer_name}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Invoice Total:</span>
            <span class="detail-value detail-mono">{formatBHD(selectedInvoice.grand_total_bhd)} BHD</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Outstanding:</span>
            <span class="detail-value detail-mono detail-highlight">{formatBHD(selectedInvoice.outstanding_bhd)} BHD</span>
          </div>
          {#if selectedInvoice.invoice_date}
            <div class="detail-row">
              <span class="detail-label">Invoice Date:</span>
              <span class="detail-value">{formatDate(selectedInvoice.invoice_date)}</span>
            </div>
          {/if}
          <button class="quick-fill-btn" type="button" onclick={selectFullOutstanding} disabled={recordLoading}>
            Pay Full Outstanding
          </button>
        </div>
      {/if}

      <!-- Amount Input -->
      <FormGroup label={paymentModalMode === 'create' ? 'Receipt Amount (BHD)' : 'Payment Amount (BHD)'} required>
        <Input
          type="number"
          step="0.001"
          min="0.001"
          placeholder="0.000"
          bind:value={paymentAmount}
          disabled={recordLoading || (paymentModalMode === 'create' && receiptApplyNow && !selectedInvoiceId)}
        />
        {#if selectedInvoice && paymentAmount && (paymentModalMode === 'edit' || receiptApplyNow)}
          <div class="amount-helper">
            {#if parseFloat(paymentAmount) > selectedInvoice.outstanding_bhd}
              <span class="helper-warning">Amount exceeds outstanding balance</span>
            {:else if parseFloat(paymentAmount) === selectedInvoice.outstanding_bhd}
              <span class="helper-success">Full payment (invoice will be marked as paid)</span>
            {:else}
              <span class="helper-info">Partial payment (remaining: {formatBHD(selectedInvoice.outstanding_bhd - parseFloat(paymentAmount))} BHD)</span>
            {/if}
          </div>
        {:else if paymentModalMode === 'create' && !receiptApplyNow}
          <p class="no-results" style="color: var(--text-muted); margin-top: 4px; font-size: 0.85rem;">
            Held on-account until applied to an invoice later.
          </p>
        {/if}
      </FormGroup>

      <!-- Payment Method -->
      <FormGroup label="Payment Method" required>
        <select
          class="select-input"
          bind:value={paymentMethod}
          disabled={recordLoading}
        >
          {#each paymentMethods as method}
            <option value={method}>{method}</option>
          {/each}
        </select>
      </FormGroup>

      <!-- Payment Date -->
      <FormGroup label={paymentModalMode === 'create' ? 'Receipt Date' : 'Payment Date'} required>
        <Input
          type="date"
          bind:value={paymentDate}
          disabled={recordLoading}
          max={new Date().toISOString().split('T')[0]}
        />
      </FormGroup>

      <!-- Reference -->
      <FormGroup label="Reference" hint="Bank ref, cheque number, etc. (required for bank/wire transfers)">
        <Input
          type="text"
          placeholder="e.g., TXN123456, CHQ001"
          bind:value={paymentReference}
          disabled={recordLoading}
        />
      </FormGroup>

      {#if paymentModalMode === 'create'}
        <FormGroup label="Notes" hint="Optional">
          <Input
            type="text"
            placeholder="Internal notes"
            bind:value={receiptNotes}
            disabled={recordLoading}
          />
        </FormGroup>
      {/if}
    </div>

    {#snippet footer()}

        <Button variant="ghost" on:click={() => {
          showRecordModal = false;
          paymentModalMode = 'create';
          editingPaymentId = '';
        }} disabled={recordLoading}>
          Cancel
        </Button>
        <Button
          variant="primary"
          on:click={handleModalSubmit}
          disabled={recordLoading || !paymentAmount || (
            paymentModalMode === 'create'
              ? (receiptApplyNow
                  ? (!selectedInvoiceId || (selectedInvoice && parseFloat(paymentAmount) > selectedInvoice.outstanding_bhd))
                  : !receiptCustomerId)
              : !selectedInvoiceId
          )}
        >
          {#if recordLoading}
            {paymentModalMode === 'create' ? 'Recording...' : 'Saving...'}
          {:else}
            {paymentModalMode === 'create' ? 'Record Receipt' : 'Save Changes'}
          {/if}
        </Button>

      {/snippet}
  </Modal>
{/if}

<!-- Apply Unapplied Balance Modal — pre-scoped to the selected receipt -->
{#if showApplyModal && applyReceipt}
  <Modal
    title="Apply Unapplied Balance"
    open={showApplyModal}
    on:close={() => {
      showApplyModal = false;
      applyReceipt = null;
    }}
    size="md"
  >
    <div class="record-form">
      <div class="locked-field">
        <div class="locked-title">{applyReceipt.receipt_number} · {applyReceipt.customer_name}</div>
        <div class="locked-subtitle">Unapplied balance: {formatBHD(applyReceipt.unapplied_amount_bhd)} BHD</div>
      </div>

      <FormGroup label="Select Invoice" required>
        <select class="select-input" bind:value={applyInvoiceId} disabled={applySaving}>
          <option value="">-- Select an invoice ({applyEligibleInvoices.length} eligible) --</option>
          {#each applyEligibleInvoices as invoice}
            <option value={invoice.id}>
              {invoice.invoice_number} ({formatBHD(invoice.outstanding_bhd)} BHD outstanding)
            </option>
          {/each}
        </select>
        {#if applyEligibleInvoices.length === 0}
          <p class="no-results" style="color: var(--text-muted); margin-top: 4px; font-size: 0.85rem;">
            No open invoices found for this customer.
          </p>
        {/if}
      </FormGroup>

      {#if applySelectedInvoice}
        <div class="invoice-details" transition:fade={{ duration: motionMs(200) }}>
          <div class="detail-row">
            <span class="detail-label">Outstanding:</span>
            <span class="detail-value detail-mono detail-highlight">{formatBHD(applySelectedInvoice.outstanding_bhd)} BHD</span>
          </div>
          <button class="quick-fill-btn" type="button" onclick={selectFullApplyAmount} disabled={applySaving}>
            Apply Max Available
          </button>
        </div>
      {/if}

      <FormGroup label="Apply Amount (BHD)" required>
        <Input
          type="number"
          step="0.001"
          min="0.001"
          placeholder="0.000"
          bind:value={applyAmount}
          disabled={applySaving || !applyInvoiceId}
        />
        {#if applySelectedInvoice && applyAmount}
          {@const cap = Math.min(applyReceipt.unapplied_amount_bhd, applySelectedInvoice.outstanding_bhd)}
          <div class="amount-helper">
            {#if parseFloat(applyAmount) > cap}
              <span class="helper-warning">Amount exceeds available balance ({formatBHD(cap)} BHD)</span>
            {:else if parseFloat(applyAmount) === cap}
              <span class="helper-success">Full available balance applied</span>
            {:else}
              <span class="helper-info">Partial application (remaining unapplied: {formatBHD(applyReceipt.unapplied_amount_bhd - parseFloat(applyAmount))} BHD)</span>
            {/if}
          </div>
        {/if}
      </FormGroup>
    </div>

    {#snippet footer()}

        <Button variant="ghost" on:click={() => {
          showApplyModal = false;
          applyReceipt = null;
        }} disabled={applySaving}>
          Cancel
        </Button>
        <Button
          variant="primary"
          on:click={handleApplyReceipt}
          disabled={applySaving || !applyInvoiceId || !applyAmount || (applySelectedInvoice && applyReceipt ? parseFloat(applyAmount) > Math.min(applyReceipt.unapplied_amount_bhd, applySelectedInvoice.outstanding_bhd) : false)}
        >
          {applySaving ? 'Applying...' : 'Apply Balance'}
        </Button>

      {/snippet}
  </Modal>
{/if}

<style>
  .payments-container {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section-header {
    margin: 4px 0 -8px;
  }

  .section-title {
    font-size: 16px;
    font-weight: 700;
    color: var(--text-primary);
    margin: 0 0 2px;
  }

  .section-subtitle {
    font-size: 12px;
    color: var(--text-secondary);
    margin: 0;
  }

  .toggle-group {
    display: flex;
    gap: 8px;
    margin-bottom: 4px;
  }

  .toggle-btn {
    flex: 1;
    padding: 8px 12px;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
    font-size: 13px;
    font-weight: 500;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .toggle-btn:hover:not(:disabled) {
    background: var(--interactive-hover);
  }

  .toggle-btn.active {
    background: var(--brand-indigo);
    border-color: var(--brand-indigo);
    color: white;
  }

  .toggle-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .selection-bar {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    align-items: center;
    padding: 10px 12px 14px;
    border-bottom: 1px solid var(--border);
    margin-bottom: 8px;
  }

  .selection-copy {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .selection-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-secondary);
    font-weight: 600;
  }

  .selection-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .selection-meta {
    font-size: 12px;
    color: var(--text-secondary);
    font-family: var(--font-mono);
  }

  .selection-actions {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .locked-field {
    border: 1px solid var(--border);
    background: var(--surface-subtle, #f8f7f4);
    border-radius: 10px;
    padding: 12px 14px;
  }

  .locked-title {
    font-weight: 600;
    color: var(--text-primary);
    font-family: var(--font-mono);
  }

  .locked-subtitle {
    margin-top: 4px;
    font-size: 13px;
    color: var(--text-secondary);
  }

  /* KPI Grid */
  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: 16px;
  }

  .kpi-card {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .kpi-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .kpi-value {
    font-size: 28px;
    font-weight: 700;
    font-family: var(--font-mono);
    color: var(--text-primary);
    line-height: 1.2;
  }

  .kpi-currency,
  .kpi-unit {
    font-size: 16px;
    font-weight: 500;
    color: var(--text-secondary);
    margin-left: 4px;
  }

  .kpi-success {
    color: #10B981;
  }

  .kpi-warning {
    color: #F59E0B;
  }

  .kpi-danger {
    color: #EF4444;
  }

  .kpi-sublabel {
    font-size: 12px;
    color: var(--text-secondary);
  }

  /* Filter Tabs */
  .filter-tabs {
    display: flex;
    gap: 8px;
    overflow-x: auto;
  }

  .filter-tab {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: transparent;
    border: none;
    border-radius: var(--border-radius-sm);
    font-size: 14px;
    font-weight: 500;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all var(--transition-fast);
    white-space: nowrap;
  }

  .filter-tab:hover {
    background: var(--interactive-hover);
    color: var(--text-primary);
  }

  .filter-tab.active {
    background: var(--brand-indigo);
    color: white;
  }

  .tab-count {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 24px;
    height: 20px;
    padding: 0 6px;
    background: rgba(0, 0, 0, 0.1);
    border-radius: 10px;
    font-size: 12px;
    font-weight: 600;
  }

  .filter-tab.active .tab-count {
    background: rgba(255, 255, 255, 0.2);
  }

  /* Loading Container */
  .loading-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    padding: 64px 24px;
  }

  .loading-container p {
    font-size: 14px;
    color: var(--text-secondary);
  }

  /* Record Form */
  .record-form {
    display: flex;
    flex-direction: column;
    gap: 20px;
    padding: 4px 0;
  }

  .select-input {
    width: 100%;
    padding: 10px 12px;
    font-size: 14px;
    font-family: var(--font-family);
    color: var(--text-primary);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
  }

  .select-input:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 3px var(--brand-indigo-tint);
  }

  .select-input:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Invoice Details */
  .invoice-details {
    padding: 16px;
    background: var(--surface-elevated);
    border-radius: var(--border-radius-sm);
    border-left: 3px solid var(--brand-indigo);
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .detail-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 13px;
  }

  .detail-label {
    font-weight: 500;
    color: var(--text-secondary);
  }

  .detail-value {
    font-weight: 600;
    color: var(--text-primary);
  }

  .detail-mono {
    font-family: var(--font-mono);
    font-size: 14px;
  }

  .detail-highlight {
    color: var(--brand-indigo);
    font-size: 16px;
  }

  .quick-fill-btn {
    margin-top: 8px;
    padding: 8px 14px;
    background: var(--brand-indigo);
    color: white;
    border: none;
    border-radius: var(--border-radius-sm);
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .quick-fill-btn:hover:not(:disabled) {
    background: var(--brand-indigo-hover);
    transform: translateY(-1px);
  }

  .quick-fill-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Amount Helper */
  .amount-helper {
    margin-top: 6px;
    font-size: 12px;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .helper-warning {
    color: #F59E0B;
    font-weight: 500;
  }

  .helper-success {
    color: #10B981;
    font-weight: 500;
  }

  .helper-info {
    color: var(--text-secondary);
  }

  /* Pagination Controls */
  .pagination-controls {
    display: flex;
    justify-content: center;
    padding: 1rem;
  }

  .load-more-btn {
    padding: 0.75rem 2rem;
    background: var(--brand-indigo, #6366f1);
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    transition: all var(--transition-fast);
  }

  .load-more-btn:hover:not(:disabled) {
    background: var(--brand-indigo-hover, #4f46e5);
    transform: translateY(-1px);
  }

  .load-more-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .all-loaded {
    text-align: center;
    color: var(--text-secondary, #a0a0a0);
    padding: 1rem;
    font-size: 13px;
  }

  /* Responsive */
  @media (max-width: 768px) {
    .kpi-grid {
      grid-template-columns: 1fr;
    }

    .filter-tabs {
      flex-wrap: wrap;
    }
  }
</style>
