<script lang="ts">
  import { run, stopPropagation } from 'svelte/legacy';

  /**
   * InvoicesScreen - Real Customer Invoice Management
   *
   * Features:
   * - List all customer invoices from database
   * - Create invoices from unfulfilled orders
   * - Send invoices (mark as Sent)
   * - Delete invoices with confirmation
   * - Filter by status: Draft, Sent, Paid, Overdue, PartiallyPaid
   * - Search by invoice number or customer name
   * - KPI dashboard: Total Revenue, Outstanding, Overdue count
   * - BHD currency with 3 decimal places
   *
   * Design System: Wabi-Sabi minimalism × Bloomberg data density
   */

  import { createEventDispatcher, onMount } from 'svelte';
  import {
    ListCustomerInvoices, ListCustomers, CreateProformaInvoiceManual, ConvertProformaToInvoice } from '../../../wailsjs/go/main/App';
import { CreateInvoiceWithOptions, CreateInvoiceWithCreditOverride, SendCustomerInvoice, DeleteCustomerInvoice, UpdateCustomerInvoice, GetCustomerInvoiceByID, GetAvailableDeliveryNotesForOrder, ListCreditNotes, CreateCreditNote, ApplyCreditNote, GenerateCreditNotePDF } from '../../../wailsjs/go/main/FinanceService';
import { ListOrders, GetOrder } from '../../../wailsjs/go/main/CRMService';
import { GenerateInvoicePDF } from '../../../wailsjs/go/main/DocumentsService';
  import { pendingInvoiceCreate } from '$lib/stores/navigation';
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
  import { t } from '$lib/i18n';
  import { permissions } from '$lib/stores/authContext';
  import { devLog } from '$lib/utils/devLog';
  import { escapeHtml } from '$lib/utils/escapeHtml';
  import { debounce } from '$lib/utils/debounce';
  import { getDefaultDivisionKey, normalizeDivision } from '$lib/divisions.svelte';

  const dispatch = createEventDispatcher();



  interface Props {
    // Props
    embedded?: boolean;
    company?: string;
    // Wave 9 B1.4: lets FinanceHub thread a status filter in from a dashboard/360 drill-through.
    invoiceFilter?: string;
    // C5: lets FinanceHub thread an AR-aging bucket in from the dashboard drill
    // (days_30/days_60/days_90/days_120_plus — matches ARAgingReport's json keys).
    agingBucket?: string;
  }

  let { embedded = false, company = getDefaultDivisionKey(), invoiceFilter = undefined, agingBucket = undefined }: Props = $props();

  // Types
  type StatusFilter = 'All' | 'Draft' | 'Sent' | 'Paid' | 'Overdue' | 'PartiallyPaid' | 'Proforma';
  type AgingBucketFilter = '' | 'days_30' | 'days_60' | 'days_90' | 'days_120_plus';

  interface InvoiceItem {
    id: string;
    description: string;
    quantity: number;
    rate: number;
    total_bhd: number;
    line_number?: number;
    product_code?: string;
    equipment?: string;
    model?: string;
    specification?: string;
    detailed_description?: string;
  }

  interface Invoice {
    id: string;
    invoice_number: string;
    invoice_date: any; // time.Time from Go
    customer_id: string;
    customer_name: string;
    order_id?: string;
    customer_po_number: string;
    grand_total_bhd: number;
    subtotal_bhd: number;
    outstanding_bhd: number;
    status: string;
    due_date: any; // time.Time from Go
    delivery_note_id?: string;
    delivery_note_number?: string;
    items?: InvoiceItem[];  // Line items from Preload
    rfq_id?: string;
    offer_id?: string;
    offer_number?: string;
    customer_reference?: string;
    attention_person?: string;
    attention_company?: string;
    delivery_weeks?: string;
    country_of_origin?: string;
    payment_terms?: string;
    delivery_terms?: string;
    division?: string;
  }

  interface Order {
    id: string;
    order_number: string;
    customer_name: string;
    customer_po_number: string;
    order_date: any;
    grand_total_bhd: number;
    status: string;
    division?: string;
  }

  interface DeliveryNote {
    id: string;
    dn_number: string;
    delivery_date: any;
    status: string;
    order_id: string;
  }

  // State
  let invoices: Invoice[] = $state([]);
  let filteredInvoices: Invoice[] = $state([]);
  let orders: Order[] = $state([]);
  let loading = $state(true);
  let selectedFilter: StatusFilter = $state('All');
  // C5: day-range aging-bucket filter, usable directly on this screen and
  // pre-selectable from the dashboard drill via the agingBucket prop.
  let selectedAgingBucket: AgingBucketFilter = $state('');
  let searchQuery = $state('');

  // Pagination state
  const PAGE_SIZE = 50;
  let currentPage = 0;
  let hasMore = $state(true);
  let loadingMore = $state(false);
  let totalLoaded = $state(0);

  // Debounced search state
  let debouncedQuery = $state('');

  // Debounced search update (300ms delay to prevent excessive filtering)
  const updateDebouncedQuery = debounce((value: string) => {
    debouncedQuery = value;
  }, 300);


  // Modal state
  let showCreateModal = $state(false);
  let showDeleteModal = $state(false);
  let createLoading = $state(false);
  let deleteLoading = $state(false);
  let selectedOrderId = $state('');
  let invoiceToDelete: Invoice | null = $state(null);

  // C4: orderless proforma creation state
  interface CustomerOption {
    id: string;
    business_name: string;
  }
  let showProformaModal = $state(false);
  let proformaCreating = $state(false);
  let proformaCustomers: CustomerOption[] = $state([]);
  let proformaCustomerId = $state('');
  let proformaItems: { description: string; quantity: number; rate: number }[] = $state([
    { description: '', quantity: 1, rate: 0 },
  ]);
  let proformaNotes = $state('');
  let convertingProformaId: string | null = $state(null);

  // Delivery Note linking state
  let availableDeliveryNotes: DeliveryNote[] = $state([]);
  let selectedDeliveryNoteId = $state('');
  let loadingDeliveryNotes = $state(false);
  let deliveryNotesLoadedForOrderId = $state('');

  // Phase 23: Credit Notes state
  type MainView = 'invoices' | 'credit-notes';
  let mainView: MainView = 'invoices';
  let creditNotes: any[] = $state([]);
  let loadingCreditNotes = $state(false);
  let showCNModal = $state(false);
  let cnInvoiceId = $state('');
  let cnReason = $state('');
  let cnItems: { description: string; quantity: number; rate: number }[] = $state([{ description: '', quantity: 1, rate: 0 }]);
  let cnSaving = $state(false);

  async function loadCreditNotes() {
    loadingCreditNotes = true;
    try {
      const data = await ListCreditNotes(100, 0);
      creditNotes = (data || []).filter((note: any) => matchesCompany(note.division));
    } catch (err) {
      console.error('Failed to load credit notes:', err);
      creditNotes = [];
    } finally {
      loadingCreditNotes = false;
    }
  }

  function openCNModal(invoiceId?: string) {
    cnInvoiceId = invoiceId || '';
    cnReason = '';
    cnItems = [{ description: '', quantity: 1, rate: 0 }];
    showCNModal = true;
  }

  function addCNItem() {
    cnItems = [...cnItems, { description: '', quantity: 1, rate: 0 }];
  }

  function removeCNItem(index: number) {
    if (cnItems.length <= 1) return;
    cnItems = cnItems.filter((_, i) => i !== index);
  }

  async function handleCreateCN() {
    if (!cnInvoiceId) { toast.warning('Select an invoice'); return; }
    if (!cnReason.trim()) { toast.warning('Reason is required'); return; }
    const validItems = cnItems.filter(i => i.description.trim() && i.quantity > 0 && i.rate > 0);
    if (validItems.length === 0) { toast.warning('At least one valid item required'); return; }

    cnSaving = true;
    try {
      await CreateCreditNote(cnInvoiceId, cnReason, validItems);
      toast.success('Credit Note created');
      showCNModal = false;
      await loadCreditNotes();
    } catch (err) {
      toast.danger('Failed: ' + (err as Error).message);
    } finally {
      cnSaving = false;
    }
  }

  async function handleApplyCN(cnId: string) {
    try {
      await ApplyCreditNote(cnId);
      toast.success('Credit Note applied — invoice outstanding updated');
      await Promise.all([loadCreditNotes(), loadInvoices()]);
    } catch (err) {
      toast.danger('Failed: ' + (err as Error).message);
    }
  }

  async function handleCNPDF(cnId: string) {
    try {
      const path = await GenerateCreditNotePDF(cnId);
      toast.success('Credit Note PDF saved');
    } catch (err) {
      toast.danger('Failed: ' + (err as Error).message);
    }
  }

  // Field visibility options (user can select what appears on the invoice)
  const defaultFieldVisibility = {
    show_equipment: true,        // Equipment/Product name
    show_specification: true,    // Short specification
    show_detailed_desc: true,    // Full detailed description
    show_fob: false,             // FOB cost
    show_freight: false,         // Freight cost
    show_cost: false,            // Total cost
    show_margin: false,          // Margin %
    show_contact: true,          // Contact person/company
    show_rfq: true,              // RFQ reference
    show_currency: false,        // Source currency
    show_country_origin: true,   // Country of origin
    show_delivery_weeks: true,   // Delivery time
  };
  let fieldVisibility = $state({ ...defaultFieldVisibility });
  // Wave 9.2 B6 / Article I.5: the 12-checkbox PDF field panel is not part of
  // the primary create flow — it lives behind a collapsed disclosure.
  let showFieldVisibility = $state(false);

  // Reset field visibility to defaults (prevents data leak between modal sessions)
  // SPOC #9: Admin/Manager credit-limit override. When the initial create is
  // blocked by the credit limit, capture the in-flight selection so submit can
  // replay the create with a recorded reason. The backend chokepoint enforces
  // the management gate + audits the reason.
  let creditOverrideOpen = $state(false);
  let creditOverrideReason = $state('');
  let creditOverrideSubmitting = $state(false);
  let creditOverrideOrderId = $state('');
  let creditOverrideOrderLabel = $state('');
  let creditOverrideDeliveryNoteId = $state('');
  let creditOverrideVisibilityJSON = $state('');

  function resetFieldVisibility() {
    fieldVisibility = { ...defaultFieldVisibility };
    showFieldVisibility = false;
  }

  // Edit state
  let showEditModal = $state(false);
  let editLoading = $state(false);
  let editInvoice: any = $state(null);
  // Wave 9.6 AR2: snapshot of the status the invoice ACTUALLY had when the
  // modal opened. editInvoice.status is bound to the select below and
  // mutates as the user picks — this stays fixed so the dropdown's legal
  // option set is computed from what the invoice is, not what the user
  // is mid-way through changing it to.
  let editInvoiceOriginalStatus: string | null = $state(null);
  let pdfLoadingMap: Record<string, boolean> = $state({});

  function matchesCompany(division?: string) {
    return normalizeDivision(division || getDefaultDivisionKey()) === normalizeDivision(company);
  }

  // P1-1 FIX: Track which invoice is being sent to prevent double-clicks
  let sendingInvoiceId: string | null = $state(null);

  // Detail modal state
  let showDetailModal = $state(false);
  let selectedInvoice: Invoice | null = $state(null);

  async function openDetailModal(invoice: Invoice) {
    console.log('[InvoicesScreen] Opening detail modal, invoice items:', invoice.items);
    selectedInvoice = invoice;
    showDetailModal = true;
    try {
      selectedInvoice = await GetCustomerInvoiceByID(invoice.id) as Invoice;
    } catch {
      selectedInvoice = invoice;
    }
  }

  // Status color mapping
  const statusColors: Record<string, string> = {
    Draft: '#6B7280',
    Sent: '#3B82F6',
    Paid: '#10B981',
    Overdue: '#EF4444',
    PartiallyPaid: '#F59E0B',
    Proforma: '#8B5CF6',
  };

  // C5: mirrors buildARAgingReport's bucket boundaries exactly (customer_invoice
  // aging engine, app_accounting_inventory.go) so the on-screen filter and the
  // dashboard drill agree on what "31-60 days" means.
  function daysOverdue(invoice: Invoice): number {
    const due = parseGoDate(invoice.due_date);
    if (due.getTime() === 0) return 0;
    const days = Math.floor((Date.now() - due.getTime()) / (1000 * 60 * 60 * 24));
    return days > 0 ? days : 0;
  }

  function agingBucketOf(invoice: Invoice): Exclude<AgingBucketFilter, ''> | 'current' {
    const d = daysOverdue(invoice);
    if (d === 0) return 'current';
    if (d <= 30) return 'days_30';
    if (d <= 60) return 'days_60';
    if (d <= 90) return 'days_90';
    return 'days_120_plus';
  }

  // Same open/collectible gate the backend's customerInvoiceClosedWorkflowStatuses
  // applies (customer_invoice_payment_policy.go) — Draft/Cancelled/Void/Proforma
  // carry no real outstanding, so they never belong in an aging bucket.
  const agingExcludedStatuses = new Set(['Draft', 'Cancelled', 'Void', 'Proforma']);
  function isAgingCollectible(invoice: Invoice): boolean {
    return invoice.outstanding_bhd > 0 && !agingExcludedStatuses.has(invoice.status);
  }

  // Round to BHD's 3-decimal precision.
  function roundMoney(value: number): number {
    return Math.round((Number(value) || 0) * 1000) / 1000;
  }

  function openEditModal(invoice: Invoice) {
    // Deep-copy line items so the editor never mutates the live list row in place.
    // Normalize numeric fields and guarantee an array even for hollow Draft invoices.
    const items = Array.isArray((invoice as any).items)
      ? (invoice as any).items.map((it: any, i: number) => ({
          ...it,
          line_number: Number(it.line_number) || i + 1,
          quantity: Number(it.quantity) || 0,
          rate: Number(it.rate) || 0,
          total_bhd: Number(it.total_bhd) || 0,
        }))
      : [];
    editInvoice = { ...invoice, items };
    editInvoiceOriginalStatus = invoice.status;
    showEditModal = true;
  }

  // Append a blank editable line (Draft only). Use max(line_number)+1 to avoid
  // collisions after mid-list removals; the payload is re-sequenced on save.
  function addInvoiceLine() {
    if (!editInvoice) return;
    const items = Array.isArray(editInvoice.items) ? editInvoice.items : [];
    const nextLine = items.length > 0
      ? Math.max(...items.map((it: any) => Number(it.line_number) || 0)) + 1
      : 1;
    editInvoice = {
      ...editInvoice,
      items: [...items, { line_number: nextLine, description: '', quantity: 1, rate: 0, total_bhd: 0 }],
    };
  }

  function removeInvoiceLine(index: number) {
    if (!editInvoice || !Array.isArray(editInvoice.items)) return;
    editInvoice = {
      ...editInvoice,
      items: editInvoice.items.filter((_: any, i: number) => i !== index),
    };
  }

  // Preserves an explicit 0% (zero-rated/export) VAT rate; only falls back to
  // the 10% default when the value is truly absent (null/undefined/''/NaN).
  function vatOrDefault(v: unknown): number {
    const n = Number(v);
    return (v === null || v === undefined || v === '' || Number.isNaN(n)) ? 10 : n;
  }

  // Live-derived totals for Draft invoices (recompute on nested item edits).
  const editDraftVatPct = $derived(vatOrDefault(editInvoice?.vat_percent));
  const editDraftSubtotal = $derived(
    editInvoice?.status === 'Draft' && Array.isArray(editInvoice?.items)
      ? editInvoice.items.reduce((sum: number, it: any) => sum + (Number(it.quantity) || 0) * (Number(it.rate) || 0), 0)
      : 0
  );
  const editDraftGrandTotal = $derived(roundMoney(editDraftSubtotal * (1 + editDraftVatPct / 100)));

  async function saveInvoiceEdit() {
    if (!editInvoice) return;

    // Draft invoices are edited at the LINE-ITEM level: rebuild items with the
    // backend contract shape (line_number/description/quantity/rate/total_bhd)
    // and derive the totals; the backend recomputes Subtotal/VAT/Grand/
    // Outstanding from these Draft items. Non-Draft keeps header-only edits.
    if (editInvoice.status === 'Draft') {
      const items = (Array.isArray(editInvoice.items) ? editInvoice.items : []).map((it: any, i: number) => {
        const quantity = Number(it.quantity) || 0;
        const rate = Number(it.rate) || 0;
        return { ...it, line_number: i + 1, description: it.description || '', quantity, rate, total_bhd: roundMoney(quantity * rate) };
      });
      const subtotal = items.reduce((sum: number, it: any) => sum + (it.total_bhd || 0), 0);
      const vatPct = vatOrDefault(editInvoice.vat_percent);
      const grand = roundMoney(subtotal * (1 + vatPct / 100));
      // Draft = fully outstanding; the backend derives the same on save.
      editInvoice = { ...editInvoice, items, grand_total_bhd: grand, outstanding_bhd: grand };
    }

    // Validate amounts > 0
    if (editInvoice.grand_total_bhd <= 0) {
      toast.warning('Invoice amount must be greater than 0');
      return;
    }
    if (editInvoice.outstanding_bhd < 0) {
      toast.warning('Outstanding amount cannot be negative');
      return;
    }
    if (editInvoice.outstanding_bhd > editInvoice.grand_total_bhd) {
      toast.warning('Outstanding amount cannot exceed invoice total');
      return;
    }

    editLoading = true;
    try {
      await UpdateCustomerInvoice(editInvoice);
      toast.success('Invoice updated successfully');
      showEditModal = false;
      editInvoice = null;
      editInvoiceOriginalStatus = null;
      await loadInvoices();
    } catch (e) {
      const errorMsg = e?.message || String(e);
      toast.danger(`Failed to update invoice: ${errorMsg}`);
    } finally {
      editLoading = false;
    }
  }

  async function downloadPDF(invoice: Invoice) {
    if (pdfLoadingMap[invoice.id]) return;
    pdfLoadingMap[invoice.id] = true;
    pdfLoadingMap = { ...pdfLoadingMap };
    try {
      const path = await GenerateInvoicePDF(invoice.id);
      toast.success(`PDF saved: ${path}`);
    } catch (e) {
      const errorMsg = e?.message || String(e);
      toast.danger(`Failed to generate PDF: ${errorMsg}`);
    } finally {
      pdfLoadingMap[invoice.id] = false;
      pdfLoadingMap = { ...pdfLoadingMap };
    }
  }

  // DataTable columns configuration
  const columns = [
    {
      key: 'invoice_number',
      label: 'Invoice #',
      sortable: true,
      width: '140px',
      render: (row: Invoice) => {
        return `<span style="font-family: var(--font-mono); font-size: 13px; color: var(--brand-indigo); font-weight: 600;">${escapeHtml(row.invoice_number)}</span>`;
      },
    },
    {
      key: 'customer_name',
      label: 'Customer',
      sortable: true,
      render: (row: Invoice) => {
        return `<span style="font-weight: 500;">${escapeHtml(row.customer_name)}</span>`;
      },
    },
    {
      key: 'invoice_date',
      label: 'Date',
      type: 'date' as const,
      sortable: true,
      width: '110px',
      render: (row: Invoice) => {
        return `<span style="font-family: var(--font-mono); font-size: 12px;">${formatDate(row.invoice_date)}</span>`;
      },
    },
    {
      key: 'due_date',
      label: 'Due Date',
      type: 'date' as const,
      sortable: true,
      width: '110px',
      render: (row: Invoice) => {
        const isOverdue = isDateOverdue(row.due_date) && row.outstanding_bhd > 0;
        const color = isOverdue ? '#EF4444' : 'var(--text-primary)';
        return `<span style="font-family: var(--font-mono); font-size: 12px; color: ${color}; font-weight: ${isOverdue ? '600' : '400'};">${formatDate(row.due_date)}</span>`;
      },
    },
    {
      key: 'grand_total_bhd',
      label: 'Amount (BHD)',
      sortable: true,
      width: '140px',
      align: 'right' as const,
      render: (row: Invoice) => {
        return `<span style="font-family: var(--font-mono); font-weight: 600; font-size: 14px;">${formatBHD(row.grand_total_bhd)}</span>`;
      },
    },
    {
      key: 'outstanding_bhd',
      label: 'Outstanding (BHD)',
      sortable: true,
      width: '160px',
      align: 'right' as const,
      render: (row: Invoice) => {
        const color = row.outstanding_bhd > 0 ? '#F59E0B' : '#10B981';
        return `<span style="font-family: var(--font-mono); font-weight: 600; font-size: 14px; color: ${color};">${formatBHD(row.outstanding_bhd)}</span>`;
      },
    },
    {
      key: 'delivery_note_number',
      label: 'DN #',
      sortable: true,
      width: '100px',
      render: (row: Invoice) => {
        if (!row.delivery_note_number) {
          return `<span style="color: var(--text-secondary); font-size: 12px;">—</span>`;
        }
        return `<span style="font-family: var(--font-mono); font-size: 12px; color: #059669; font-weight: 500;">${escapeHtml(row.delivery_note_number)}</span>`;
      },
    },
    {
      key: 'status',
      label: 'Status',
      sortable: true,
      width: '110px',
      render: (row: Invoice) => {
        const color = statusColors[row.status] || '#6B7280';
        return `<span style="display: inline-block; padding: 4px 10px; border-radius: 12px; font-size: 11px; font-weight: 600; text-transform: uppercase; background: ${color}15; color: ${color};">${escapeHtml(row.status || '')}</span>`;
      },
    },
    {
      key: 'actions',
      label: 'Actions',
      width: '140px',
      align: 'center' as const,
      render: (row: Invoice) => {
        return ''; // Handled by slot
      },
    },
  ];

  // Filter tabs
  const filterTabs: { value: StatusFilter; label: string; count: number }[] = $state([
    { value: 'All', label: 'All', count: 0 },
    { value: 'Draft', label: 'Draft', count: 0 },
    { value: 'Sent', label: 'Sent', count: 0 },
    { value: 'Paid', label: 'Paid', count: 0 },
    { value: 'Overdue', label: 'Overdue', count: 0 },
    { value: 'PartiallyPaid', label: 'Partial', count: 0 },
    { value: 'Proforma', label: 'Proforma', count: 0 },
  ]);

  // C5: aging-bucket filter options, boundary semantics identical to the
  // dashboard's ARAgingReport buckets (agingBucketOf above).
  const agingBucketOptions: { value: AgingBucketFilter; label: string }[] = [
    { value: '', label: 'All Ages' },
    { value: 'days_30', label: '1-30 days' },
    { value: 'days_60', label: '31-60 days' },
    { value: 'days_90', label: '61-90 days' },
    { value: 'days_120_plus', label: '120+ days' },
  ];




  // Currency formatter - BHD with 3 decimal places
  function formatBHD(amount: number): string {
    return new Intl.NumberFormat('en-US', {
      minimumFractionDigits: 3,
      maximumFractionDigits: 3
    }).format(Number(amount) || 0);
  }

  function openBankReconciliation() {
    dispatch('navigate', { tab: 'bank_recon', source: 'customer_invoices' });
  }

  // Parse Go time.Time to JS Date
  function parseGoDate(dateValue: any): Date {
    if (!dateValue) return new Date(0);

    // If it's already a Date object
    if (dateValue instanceof Date) return dateValue;

    // If it's a string (ISO format)
    if (typeof dateValue === 'string') {
      return new Date(dateValue);
    }

    // If it's a time.Time object (has properties)
    if (typeof dateValue === 'object') {
      // Try to convert to string first
      const str = dateValue.toString?.() || String(dateValue);
      if (str && str !== '[object Object]') {
        return new Date(str);
      }
    }

    return new Date(0);
  }

  // Format date for display
  function formatDate(dateValue: any): string {
    const date = parseGoDate(dateValue);
    if (date.getTime() === 0) return '—';

    return date.toLocaleDateString('en-GB', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    });
  }

  // Check if date is overdue
  function isDateOverdue(dateValue: any): boolean {
    const date = parseGoDate(dateValue);
    if (date.getTime() === 0) return false;
    return date < new Date();
  }

  // Load invoices from database with pagination
  async function loadInvoices() {
    loading = true;
    currentPage = 0;
    hasMore = true;
    try {
      const data = await ListCustomerInvoices(PAGE_SIZE, 0);
      invoices = (data || []).filter((invoice) => matchesCompany(invoice.division));
      currentPage = 1;
      totalLoaded = invoices.length;
      hasMore = invoices.length === PAGE_SIZE;
      // DEBUG: Check if items are loaded
      const withItems = invoices.filter(inv => inv.items && inv.items.length > 0);
      console.log(`[InvoicesScreen] Loaded ${invoices.length} invoices, ${withItems.length} have items`);
      if (invoices[0]) console.log('[InvoicesScreen] First invoice items:', invoices[0].items);
      devLog.info(`Loaded ${invoices.length} invoices (page 1, hasMore: ${hasMore})`);
    } catch (err) {
      toast.danger('Failed to load invoices');
      invoices = [];
      hasMore = false;
    } finally {
      loading = false;
    }
  }

  // Load more invoices (pagination)
  async function loadMore() {
    if (loadingMore || !hasMore) return;

    loadingMore = true;
    try {
      const offset = currentPage * PAGE_SIZE;
      const data = await ListCustomerInvoices(PAGE_SIZE, offset);

      if (data && data.length > 0) {
        invoices = [...invoices, ...data.filter((invoice) => matchesCompany(invoice.division))];
        currentPage++;
        totalLoaded = invoices.length;
        hasMore = data.length === PAGE_SIZE;
        devLog.info(`Loaded ${data.length} more invoices (total: ${totalLoaded}, hasMore: ${hasMore})`);
      } else {
        hasMore = false;
      }
    } catch (err) {
      console.error('Failed to load more:', err);
      toast.danger('Failed to load more invoices');
    } finally {
      loadingMore = false;
    }
  }

  // Load unfulfilled orders for invoice creation
  async function loadUnfulfilledOrders() {
    try {
      const allOrders = await ListOrders(500, 0);
      orders = (allOrders || []).filter(
        (order) => matchesCompany(order.division) && order.status !== 'Complete' && order.status !== 'Invoiced'
      );
      devLog.info(`Loaded ${orders.length} unfulfilled orders`);
    } catch (err) {
      toast.danger('Failed to load orders');
      orders = [];
    }
  }

  // Open create invoice modal
  async function openCreateModal() {
    await loadUnfulfilledOrders();

    selectedOrderId = '';
    selectedDeliveryNoteId = '';
    availableDeliveryNotes = [];
    deliveryNotesLoadedForOrderId = '';
    resetFieldVisibility();
    // No bare toast dead-end (Article II #4): if there's nothing to invoice,
    // the modal itself explains why and links out to Orders — see the
    // empty-hint block in the template below.
    showCreateModal = true;
  }

  // B6: recoverable dead-end — when there are no unfulfilled orders, send the
  // user to where they can fix that instead of leaving them stuck.
  function goToOrders() {
    showCreateModal = false;
    resetFieldVisibility();
    window.dispatchEvent(new CustomEvent('navigateToScreen', { detail: { screen: 'opportunities', tab: 'orders' } }));
  }

  // C4: proforma invoices don't need an order — they're a reference/quotation
  // document a customer can be handed before Ops confirms anything. Loads the
  // customer list (unfiltered by company: CustomerMaster carries no division).
  async function openProformaModal() {
    proformaCustomerId = '';
    proformaItems = [{ description: '', quantity: 1, rate: 0 }];
    proformaNotes = '';
    try {
      const data = await ListCustomers(500, 0);
      proformaCustomers = (data || []) as CustomerOption[];
    } catch (err) {
      proformaCustomers = [];
      toast.danger('Failed to load customers');
    }
    showProformaModal = true;
  }

  function addProformaItem() {
    proformaItems = [...proformaItems, { description: '', quantity: 1, rate: 0 }];
  }

  function removeProformaItem(index: number) {
    if (proformaItems.length <= 1) return;
    proformaItems = proformaItems.filter((_, i) => i !== index);
  }

  async function handleCreateProforma() {
    if (!canCreateInvoice) {
      toast.danger('Your role does not have permission to create invoices.');
      return;
    }
    if (!proformaCustomerId) {
      toast.warning('Select a customer');
      return;
    }
    const validItems = proformaItems.filter((i) => i.description.trim() && i.quantity > 0 && i.rate > 0);
    if (validItems.length === 0) {
      toast.warning('At least one valid line item is required');
      return;
    }

    proformaCreating = true;
    try {
      const customerName = proformaCustomers.find((c) => c.id === proformaCustomerId)?.business_name || '';
      const created = await CreateProformaInvoiceManual(proformaCustomerId, customerName, validItems, proformaNotes);
      toast.success(`Proforma ${created.invoice_number} created — it posts nothing until converted`);
      showProformaModal = false;
      await loadInvoices();
    } catch (err) {
      toast.danger(`Failed to create proforma: ${(err as Error).message}`);
    } finally {
      proformaCreating = false;
    }
  }

  // Guarded conversion (mirrors handleSendInvoice's double-click guard): once
  // converted, the proforma becomes a real, fiscally-numbered invoice and
  // enters AR aging/VAT — make the consequence explicit before committing.
  async function handleConvertProforma(invoice: Invoice) {
    if (convertingProformaId === invoice.id) return;
    const confirmed = await confirm.ask({
      title: 'Convert Proforma to Invoice',
      message: `Proforma ${invoice.invoice_number} will become a real invoice with a new INV- number and will enter accounts receivable. This cannot be undone.`,
      confirmLabel: 'Convert',
      variant: 'warning',
    });
    if (!confirmed) return;

    convertingProformaId = invoice.id;
    try {
      const converted = await ConvertProformaToInvoice(invoice.id, '');
      toast.success(`Proforma converted to invoice ${converted.invoice_number}`);
      await loadInvoices();
    } catch (err) {
      toast.danger(`Failed to convert proforma: ${(err as Error).message}`);
    } finally {
      convertingProformaId = null;
    }
  }

  // Load delivery notes when order is selected
  async function loadDeliveryNotesForOrder(orderId: string) {
    if (!orderId) {
      availableDeliveryNotes = [];
      selectedDeliveryNoteId = '';
      deliveryNotesLoadedForOrderId = '';
      return;
    }

    loadingDeliveryNotes = true;
    try {
      const dns = await GetAvailableDeliveryNotesForOrder(orderId);
      availableDeliveryNotes = dns || [];
      selectedDeliveryNoteId = '';
      deliveryNotesLoadedForOrderId = orderId;
      devLog.info(`Loaded ${availableDeliveryNotes.length} available DNs for order ${orderId}`);
    } catch (err) {
      availableDeliveryNotes = [];
      deliveryNotesLoadedForOrderId = orderId;
    } finally {
      loadingDeliveryNotes = false;
    }
  }



  // Create invoice from order (with optional DN linkage and field visibility)
  async function handleCreateInvoice() {
    if (!selectedOrderId) {
      toast.warning('Please select an order');
      return;
    }
    if (!canCreateInvoice) {
      toast.danger('Your role does not have permission to create invoices.');
      return;
    }

    createLoading = true;
    try {
      const visibilityJSON = JSON.stringify(fieldVisibility);

      const newInvoice = await CreateInvoiceWithOptions(
        selectedOrderId,
        selectedDeliveryNoteId || '',
        visibilityJSON
      );

      let successMsg = `Invoice ${newInvoice.invoice_number} created`;
      if (selectedDeliveryNoteId) {
        const dnNumber = availableDeliveryNotes.find(dn => dn.id === selectedDeliveryNoteId)?.dn_number || '';
        successMsg += ` and linked to DN ${dnNumber}`;
      }
      toast.success(successMsg);

      showCreateModal = false;
      resetFieldVisibility();
      await loadInvoices();
    } catch (err) {
      const errorMsg = err?.message || String(err);
      // SPOC #9: a credit-limit block is recoverable — offer a management
      // override (with a reason) instead of a dead-end toast. Any other
      // failure keeps the existing behaviour.
      if (/credit limit exceeded/i.test(errorMsg)) {
        openCreditOverride();
      } else {
        toast.danger(`Failed to create invoice: ${errorMsg}`);
      }
    } finally {
      createLoading = false;
    }
  }

  // Capture the in-flight selection and swap the create modal for the override modal.
  function openCreditOverride() {
    creditOverrideOrderId = selectedOrderId;
    creditOverrideDeliveryNoteId = selectedDeliveryNoteId || '';
    creditOverrideVisibilityJSON = JSON.stringify(fieldVisibility);
    creditOverrideOrderLabel = orders.find(o => o.id === selectedOrderId)?.order_number || '';
    creditOverrideReason = '';
    creditOverrideSubmitting = false;
    showCreateModal = false; // close the create modal so the dialogs don't stack
    creditOverrideOpen = true;
  }

  function cancelCreditOverride() {
    creditOverrideOpen = false;
    creditOverrideReason = '';
    creditOverrideOrderId = '';
    resetFieldVisibility();
  }

  async function submitCreditOverride() {
    const reason = creditOverrideReason.trim();
    if (!creditOverrideOrderId || !reason || creditOverrideSubmitting) {
      return;
    }
    creditOverrideSubmitting = true;
    try {
      const newInvoice = await CreateInvoiceWithCreditOverride(
        creditOverrideOrderId,
        creditOverrideDeliveryNoteId,
        creditOverrideVisibilityJSON,
        reason
      );
      toast.success(`Invoice ${newInvoice.invoice_number} created (credit limit overridden)`);
      creditOverrideOpen = false;
      creditOverrideReason = '';
      creditOverrideOrderId = '';
      resetFieldVisibility();
      await loadInvoices();
    } catch (err) {
      // The backend rejects non-management users — surface it; the modal stays
      // open so a privileged user can adjust the reason and retry.
      const errorMsg = err?.message || String(err);
      toast.danger(`Failed to override credit limit: ${errorMsg}`);
    } finally {
      creditOverrideSubmitting = false;
    }
  }

  // Send invoice (mark as Sent)
  async function handleSendInvoice(invoice: Invoice) {
    if (invoice.status === 'Sent') {
      toast.info('Invoice already sent');
      return;
    }

    // P1-1 FIX: Prevent double-clicks
    if (sendingInvoiceId === invoice.id) return;
    sendingInvoiceId = invoice.id;

    try {
      await SendCustomerInvoice(invoice.id);
      toast.success(`Invoice ${invoice.invoice_number} marked as sent`);
      await loadInvoices();
    } catch (err) {
      const errorMsg = err?.message || String(err);
      toast.danger(`Failed to send invoice: ${errorMsg}`);
    } finally {
      sendingInvoiceId = null;
    }
  }

  // Open delete confirmation
  function openDeleteModal(invoice: Invoice) {
    invoiceToDelete = invoice;
    showDeleteModal = true;
  }

  // Delete invoice
  async function handleDeleteInvoice() {
    if (!invoiceToDelete) return;

    deleteLoading = true;
    try {
      await DeleteCustomerInvoice(invoiceToDelete.id);
      toast.success(`Invoice ${invoiceToDelete.invoice_number} deleted`);
      showDeleteModal = false;
      invoiceToDelete = null;
      await loadInvoices();
    } catch (err) {
      const errorMsg = err?.message || String(err);
      toast.danger(`Failed to delete invoice: ${errorMsg}`);
    } finally {
      deleteLoading = false;
    }
  }

  // B7b: DeliveryNotesScreen requests this when confirming a delivery brings an
  // order's remaining-to-deliver to zero — the sales loop's last handoff.
  // Mirrors how DeliveryNotesScreen consumes pendingDNCreate.
  async function checkPendingInvoiceCreate() {
    const pending = $pendingInvoiceCreate;
    if (!pending) return;
    pendingInvoiceCreate.clear();

    await openCreateModal();

    if (!orders.find((o) => o.id === pending.orderId)) {
      try {
        const order = await GetOrder(pending.orderId);
        if (order) orders = [order, ...orders];
      } catch (err) {
        console.warn('Failed to load pending order for invoice creation:', err);
      }
    }
    selectedOrderId = pending.orderId;
  }

  onMount(() => {
    loadInvoices().then(() => {
      // B3 360-continuity: CustomerDetailView drills into a specific invoice via
      // this pending-store handoff (mirrors CostingSheetScreen's launch payloads).
      const pendingInvoiceFocus = sessionStorage.getItem('asymmflow.pendingInvoiceFocus');
      if (pendingInvoiceFocus) {
        try {
          const pending = JSON.parse(pendingInvoiceFocus);
          const match = invoices.find(
            (inv) => (pending.id && inv.id === pending.id) || (pending.invoice_number && inv.invoice_number === pending.invoice_number)
          );
          if (match) {
            openDetailModal(match);
          } else {
            toast.warning('Could not find that invoice in this view.');
          }
        } finally {
          sessionStorage.removeItem('asymmflow.pendingInvoiceFocus');
        }
      }
    });
    void checkPendingInvoiceCreate();
  });

  // Wave 9 B1.4: apply an incoming status filter from a dashboard/360 drill-through.
  let lastAppliedInvoiceFilter = '';
  const invoiceStatusFilters: StatusFilter[] = ['All', 'Draft', 'Sent', 'Paid', 'Overdue', 'PartiallyPaid', 'Proforma'];
  run(() => {
    if (invoiceFilter && invoiceFilter !== lastAppliedInvoiceFilter) {
      if (invoiceStatusFilters.includes(invoiceFilter as StatusFilter)) {
        selectedFilter = invoiceFilter as StatusFilter;
      }
      lastAppliedInvoiceFilter = invoiceFilter;
    }
  });

  // C5: apply an incoming aging-bucket filter from the dashboard AR-aging drill.
  let lastAppliedAgingBucket = '';
  const validAgingBuckets: AgingBucketFilter[] = ['days_30', 'days_60', 'days_90', 'days_120_plus'];
  run(() => {
    if (agingBucket && agingBucket !== lastAppliedAgingBucket) {
      if (validAgingBuckets.includes(agingBucket as AgingBucketFilter)) {
        selectedAgingBucket = agingBucket as AgingBucketFilter;
      }
      lastAppliedAgingBucket = agingBucket;
    }
  });

  let permissionList = $derived(Array.isArray($permissions) ? $permissions : []);
  let canCreateInvoice =
    $derived(permissionList.includes('*') ||
    permissionList.includes('invoices:create') ||
    permissionList.includes('invoices:*') ||
    permissionList.includes('finance:*'));
  // Watch searchQuery changes and debounce
  run(() => {
    updateDebouncedQuery(searchQuery);
  });
  // Computed: Update tab counts
  run(() => {
    filterTabs[0].count = invoices.length;
    filterTabs[1].count = invoices.filter((i) => i.status === 'Draft').length;
    filterTabs[2].count = invoices.filter((i) => i.status === 'Sent').length;
    filterTabs[3].count = invoices.filter((i) => i.status === 'Paid').length;
    filterTabs[4].count = invoices.filter((i) => i.status === 'Overdue').length;
    filterTabs[5].count = invoices.filter((i) => i.status === 'PartiallyPaid').length;
    filterTabs[6].count = invoices.filter((i) => i.status === 'Proforma').length;
  });
  // Computed: Filter and search (uses debouncedQuery for performance)
  run(() => {
    let result = [...invoices];

    // Status filter
    if (selectedFilter !== 'All') {
      result = result.filter((i) => i.status === selectedFilter);
    }

    // C5: aging-bucket filter — same collectible gate + bucket boundaries the
    // dashboard's AR-aging drill uses (agingBucketOf/isAgingCollectible above).
    if (selectedAgingBucket) {
      result = result.filter((i) => isAgingCollectible(i) && agingBucketOf(i) === selectedAgingBucket);
    }

    // Search filter (debounced to prevent excessive re-renders)
    if (debouncedQuery) {
      const q = debouncedQuery.toLowerCase();
      result = result.filter(
        (i) =>
          i.invoice_number.toLowerCase().includes(q) ||
          i.customer_name.toLowerCase().includes(q)
      );
    }

    filteredInvoices = result.sort((a, b) => {
      const dateA = parseGoDate(a.invoice_date);
      const dateB = parseGoDate(b.invoice_date);
      return dateB.getTime() - dateA.getTime();
    });
  });
  // Computed: KPI stats
  let stats = $derived({
    // C4: a Proforma posts nothing (see CreateProformaInvoiceManual) — it must
    // not inflate Total Revenue until it is converted into a real invoice.
    totalRevenueYTD: invoices
      .filter((invoice) => invoice.status !== 'Proforma' && parseGoDate(invoice.invoice_date).getFullYear() === new Date().getFullYear())
      .reduce((sum, i) => sum + i.grand_total_bhd, 0),
    outstanding: invoices.reduce((sum, i) => sum + i.outstanding_bhd, 0),
    overdueCount: invoices.filter(
      (i) => i.status === 'Overdue' && i.outstanding_bhd > 0
    ).length,
  });
  run(() => {
    if (!showCreateModal && deliveryNotesLoadedForOrderId) {
      deliveryNotesLoadedForOrderId = '';
    }
  });
  // React to order selection changes
  run(() => {
    if (showCreateModal && selectedOrderId && selectedOrderId !== deliveryNotesLoadedForOrderId && !loadingDeliveryNotes) {
      loadDeliveryNotesForOrder(selectedOrderId);
    }
  });
  run(() => {
    if (company) {
      loadInvoices();
    }
  });
</script>

<PageLayout title={t("finance.invoice.title")} subtitle="Customer billing and receivables management" {embedded}>
  <!-- @migration-task: migrate this slot by hand, `header-actions` is an invalid identifier -->
  <svelte:fragment slot="header-actions">
    <Button variant="secondary" on:click={loadInvoices}>{t("common.refresh")}</Button>
    <Button variant="secondary" on:click={openBankReconciliation}>Open Bank Recon</Button>
    {#if canCreateInvoice}
      <Button variant="secondary" on:click={openProformaModal}>+ Create Proforma</Button>
      <Button variant="primary" on:click={openCreateModal}>+ {t("finance.invoice.create")}</Button>
    {/if}
  </svelte:fragment>

  <div class="invoices-container">
  {#if mainView === 'invoices'}
    <!-- KPI Stats -->
    <div class="kpi-grid">
      <Card padding="md" variant="elevated">
        <div class="kpi-card">
          <div class="kpi-label">Total Revenue (YTD)</div>
          <div class="kpi-value kpi-primary">
            {formatBHD(stats.totalRevenueYTD)} <span class="kpi-currency">BHD</span>
          </div>
          <div class="kpi-sublabel">{new Date().getFullYear()} billings</div>
        </div>
      </Card>

      <Card padding="md" variant="elevated">
        <div class="kpi-card">
          <div class="kpi-label">Outstanding</div>
          <div class="kpi-value" class:kpi-warning={stats.outstanding > 0}>
            {formatBHD(stats.outstanding)} <span class="kpi-currency">BHD</span>
          </div>
          <div class="kpi-sublabel">Unpaid receivables</div>
        </div>
      </Card>

      <Card padding="md" variant="elevated">
        <div class="kpi-card">
          <div class="kpi-label">Overdue</div>
          <div class="kpi-value" class:kpi-danger={stats.overdueCount > 0}>
            {stats.overdueCount}
          </div>
          <div class="kpi-sublabel">
            {stats.overdueCount === 1 ? 'Invoice' : 'Invoices'} past due
          </div>
        </div>
      </Card>
    </div>

    <!-- Filter Tabs and Search -->
    <div class="controls-row">
      <Card padding="sm" style="flex: 1;">
        <div class="filter-tabs" role="tablist" aria-label="Filter invoices by status">
          {#each filterTabs as tab}
            <button
              class="filter-tab"
              class:active={selectedFilter === tab.value}
              role="tab"
              aria-selected={selectedFilter === tab.value}
              onclick={() => (selectedFilter = tab.value)}
            >
              {tab.label}
              <span class="tab-count">{tab.count}</span>
            </button>
          {/each}
        </div>
      </Card>

      <select
        class="select-input"
        style="width: 160px;"
        bind:value={selectedAgingBucket}
        aria-label="Filter invoices by AR aging bucket"
      >
        {#each agingBucketOptions as opt}
          <option value={opt.value}>{opt.label}</option>
        {/each}
      </select>

      <Input
        type="text"
        placeholder={t("common.search")}
        bind:value={searchQuery}
        style="width: 300px;"
        aria-label="Search invoices by number or customer"
      />
    </div>

    <!-- Invoices DataTable -->
    <Card padding="sm">
      {#if loading}
        <div class="loading-container">
          <WabiSpinner size="lg" />
          <p>{t("common.loading")}</p>
        </div>
      {:else}
        <DataTable
          {columns}
          data={filteredInvoices}
          {loading}
          emptyMessage="No invoices yet — bill a completed order to start."
          onRowClick={(row) => openDetailModal(row)}
          stickyHeader={!embedded}
          maxHeight={embedded ? '400px' : 'calc(100vh - 520px)'}
          showBorder={false}
        >
          {#snippet cell({ row, column })}
                      
              {#if column.key === 'actions'}
                <div class="action-buttons">
                  <button
                    class="action-btn action-edit"
                    onclick={stopPropagation(() => openEditModal(row))}
                    title="Edit Invoice"
                  >
                    Edit
                  </button>
                  <button
                    class="action-btn action-pdf"
                    onclick={stopPropagation(() => downloadPDF(row))}
                    title="Download PDF"
                    disabled={pdfLoadingMap[row.id]}
                  >
                    {pdfLoadingMap[row.id] ? '...' : 'PDF'}
                  </button>
                  {#if row.status === 'Draft'}
                    <button
                      class="action-btn action-send"
                      onclick={stopPropagation(() => handleSendInvoice(row))}
                      title="Send Invoice"
                      disabled={sendingInvoiceId === row.id}
                    >
                      {sendingInvoiceId === row.id ? '...' : 'Send'}
                    </button>
                  {/if}
                  {#if row.status === 'Proforma'}
                    <button
                      class="action-btn action-send"
                      onclick={stopPropagation(() => handleConvertProforma(row))}
                      title="Convert to Invoice"
                      disabled={convertingProformaId === row.id}
                    >
                      {convertingProformaId === row.id ? '...' : 'Convert'}
                    </button>
                  {/if}
                  {#if row.status !== 'Paid'}
                    <button
                      class="action-btn action-delete"
                      onclick={stopPropagation(() => openDeleteModal(row))}
                      title="Delete Invoice"
                    >
                      Delete
                    </button>
                  {/if}
                </div>
              {/if}
            
                      {/snippet}
        </DataTable>
      {/if}
    </Card>

    <!-- Pagination Controls -->
    {#if hasMore && !loading}
      <div class="pagination-controls">
        <button
          class="load-more-btn"
          onclick={loadMore}
          disabled={loadingMore}
          aria-label={loadingMore ? 'Loading more invoices' : `Load more invoices, ${totalLoaded} currently loaded`}
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
      <p class="all-loaded">All {totalLoaded} invoices loaded</p>
    {/if}

  {:else}
    <!-- Phase 23: Credit Notes View -->
    <Card padding="md">
      {#if loadingCreditNotes}
        <div style="text-align: center; padding: 32px;">
          <WabiSpinner size="md" />
          <p style="margin-top: 8px; color: var(--text-secondary);">Loading credit notes...</p>
        </div>
      {:else if creditNotes.length === 0}
        <div style="text-align: center; padding: 40px; color: var(--text-secondary);">
          <p style="font-size: 14px; font-weight: 500;">No credit notes yet</p>
          <p style="font-size: 12px;">Issue a credit note against an invoice for returns or adjustments.</p>
          <Button variant="primary" on:click={() => openCNModal()}>+ Issue Credit Note</Button>
        </div>
      {:else}
        <table class="cn-table">
          <thead>
            <tr>
              <th>CN Number</th>
              <th>Date</th>
              <th>Invoice Ref</th>
              <th>Customer</th>
              <th class="right">Amount (BHD)</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each creditNotes as cn}
              <tr>
                <td class="mono">{cn.credit_note_number}</td>
                <td>{formatDate(cn.credit_note_date)}</td>
                <td class="mono">{cn.invoice_number}</td>
                <td>{cn.customer_name}</td>
                <td class="right mono">{formatBHD(cn.grand_total_bhd || 0)}</td>
                <td>
                  <span class="cn-status" class:cn-draft={cn.status === 'Draft'} class:cn-issued={cn.status === 'Issued'} class:cn-applied={cn.status === 'Applied'}>
                    {cn.status}
                  </span>
                </td>
                <td>
                  <div style="display: flex; gap: 6px;">
                    <button class="action-btn action-secondary" onclick={() => handleCNPDF(cn.id)}>PDF</button>
                    {#if cn.status !== 'Applied'}
                      <button class="action-btn action-primary" onclick={() => handleApplyCN(cn.id)}>Apply</button>
                    {/if}
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </Card>
  {/if}
  </div>
</PageLayout>

<!-- Create Invoice Modal -->
{#if showCreateModal}
  <Modal
    title="Create Invoice from Order"
    open={showCreateModal}
    on:close={() => { showCreateModal = false; resetFieldVisibility(); }}
    size="md"
  >
    <div class="create-form">
      {#if orders.length === 0}
        <!-- B6: recoverable dead-end (Article II #4) — explain why, and link
             out to where the user can fix it, instead of a bare toast. -->
        <div class="empty-hint">
          <p><strong>No unfulfilled orders available.</strong> Every order is already Complete or Invoiced, so there's nothing eligible to bill right now.</p>
          <p>Confirm a new order first, then come back here to invoice it.</p>
          <Button variant="secondary" on:click={goToOrders}>Open Orders</Button>
        </div>
      {:else}
        <FormGroup label="Select Unfulfilled Order" required>
          <select
            class="select-input"
            bind:value={selectedOrderId}
            disabled={createLoading}
          >
            <option value="">-- Select an order --</option>
            {#each orders as order}
              <option value={order.id}>
                {order.order_number} - {order.customer_name} ({formatBHD(order.grand_total_bhd)} BHD) - {order.status}
              </option>
            {/each}
          </select>
        </FormGroup>

        <!-- Delivery Note Selector (appears after order is selected) -->
        {#if selectedOrderId}
          <FormGroup label="Link Delivery Note (Optional)">
            {#if loadingDeliveryNotes}
              <div class="dn-loading">Loading delivery notes...</div>
            {:else if availableDeliveryNotes.length === 0}
              <div class="dn-empty-hint">
                <p>No delivery notes available for this order. You can create the invoice without a DN link.</p>
              </div>
            {:else}
              <select
                class="select-input"
                bind:value={selectedDeliveryNoteId}
                disabled={createLoading}
              >
                <option value="">-- No DN (create invoice without DN) --</option>
                {#each availableDeliveryNotes as dn}
                  <option value={dn.id}>
                    {dn.dn_number} - {formatDate(dn.delivery_date)} - {dn.status}
                  </option>
                {/each}
              </select>
              <p class="dn-hint">Select a delivery note to link with this invoice, or leave blank to create invoice without DN.</p>
            {/if}
          </FormGroup>

          <!-- Field Visibility Options — collapsed by default (Article I.5: creation
               leads with the job; PDF field customization is a disclosure, not a
               12-checkbox wall in the primary create flow). -->
          <div class="field-visibility-section">
            <button
              type="button"
              class="expand-toggle"
              onclick={() => (showFieldVisibility = !showFieldVisibility)}
              aria-expanded={showFieldVisibility}
            >
              {showFieldVisibility ? '▾' : '▸'} Customize fields shown on PDF
            </button>

            {#if showFieldVisibility}
              <p class="visibility-hint">Select which fields to show on the invoice PDF.</p>

              <div class="visibility-grid">
                <!-- Line Item Fields -->
                <div class="visibility-group">
                  <span class="group-label">Line Items</span>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_equipment} />
                    Equipment Name
                  </label>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_specification} />
                    Specification
                  </label>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_detailed_desc} />
                    Detailed Description
                  </label>
                </div>

                <!-- Cost Fields -->
                <div class="visibility-group">
                  <span class="group-label">Cost Data (Internal)</span>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_fob} />
                    FOB Cost
                  </label>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_freight} />
                    Freight Cost
                  </label>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_cost} />
                    Total Cost
                  </label>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_margin} />
                    Margin %
                  </label>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_currency} />
                    Source Currency
                  </label>
                </div>

                <!-- Contact/RFQ Fields -->
                <div class="visibility-group">
                  <span class="group-label">Header Details</span>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_contact} />
                    Contact Person
                  </label>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_rfq} />
                    RFQ Reference
                  </label>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_country_origin} />
                    Country of Origin
                  </label>
                  <label class="checkbox-label">
                    <input type="checkbox" bind:checked={fieldVisibility.show_delivery_weeks} />
                    Delivery Time
                  </label>
                </div>
              </div>
            {/if}
          </div>
        {/if}
      {/if}
    </div>

    {#snippet footer()}
      
        <Button variant="ghost" on:click={() => { showCreateModal = false; resetFieldVisibility(); }} disabled={createLoading}>
          Cancel
        </Button>
        <Button
          variant="primary"
          on:click={handleCreateInvoice}
          disabled={createLoading || !selectedOrderId}
        >
          {createLoading ? 'Creating...' : selectedDeliveryNoteId ? 'Create Invoice + Link DN' : 'Create Invoice'}
        </Button>
      
      {/snippet}
  </Modal>
{/if}

<!-- C4: Create Proforma Modal (orderless — customer + manual line items) -->
{#if showProformaModal}
  <Modal
    title="Create Proforma Invoice"
    open={showProformaModal}
    on:close={() => (showProformaModal = false)}
    size="md"
  >
    <div class="create-form">
      <p class="dn-hint">A proforma is a reference/quotation document — it posts nothing (no VAT, no AR aging) until it's converted into a real invoice.</p>

      <FormGroup label="Customer" required>
        <select class="select-input" bind:value={proformaCustomerId} disabled={proformaCreating}>
          <option value="">-- Select a customer --</option>
          {#each proformaCustomers as c}
            <option value={c.id}>{c.business_name}</option>
          {/each}
        </select>
      </FormGroup>

      <div class="cn-items-section">
        <div class="cn-items-header">
          <h4>Line Items</h4>
          <button class="cn-add-item-btn" onclick={addProformaItem} disabled={proformaCreating}>+ Add Item</button>
        </div>

        {#each proformaItems as item, i}
          <div class="cn-item-row">
            <div class="cn-item-desc">
              <input
                type="text"
                class="input-field"
                placeholder="Description"
                bind:value={item.description}
                disabled={proformaCreating}
              />
            </div>
            <div class="cn-item-qty">
              <input
                type="number"
                class="input-field"
                placeholder="Qty"
                min="0"
                step="0.001"
                bind:value={item.quantity}
                disabled={proformaCreating}
              />
            </div>
            <div class="cn-item-rate">
              <input
                type="number"
                class="input-field"
                placeholder="Rate (BHD)"
                min="0"
                step="0.001"
                bind:value={item.rate}
                disabled={proformaCreating}
              />
            </div>
            <div class="cn-item-total">
              {formatBHD(item.quantity * item.rate)}
            </div>
            <button
              class="cn-remove-btn"
              onclick={() => removeProformaItem(i)}
              disabled={proformaCreating || proformaItems.length <= 1}
              title="Remove item"
            >
              &times;
            </button>
          </div>
        {/each}

        <div class="cn-total-row">
          <span>Subtotal:</span>
          <span class="cn-total-value">
            {formatBHD(proformaItems.reduce((s, it) => s + (it.quantity * it.rate), 0))} BHD
          </span>
        </div>
      </div>

      <FormGroup label="Notes">
        <textarea class="cn-textarea" rows="2" bind:value={proformaNotes} disabled={proformaCreating}></textarea>
      </FormGroup>
    </div>

    {#snippet footer()}

        <Button variant="ghost" on:click={() => (showProformaModal = false)} disabled={proformaCreating}>
          Cancel
        </Button>
        <Button variant="primary" on:click={handleCreateProforma} disabled={proformaCreating}>
          {proformaCreating ? 'Creating...' : 'Create Proforma'}
        </Button>

      {/snippet}
  </Modal>
{/if}

<!-- Delete Confirmation Modal -->
{#if showDeleteModal && invoiceToDelete}
  <Modal
    title="Delete Invoice"
    open={showDeleteModal}
    on:close={() => (showDeleteModal = false)}
    size="sm"
  >
    <div class="delete-confirmation">
      <p>Are you sure you want to delete invoice <strong>{invoiceToDelete.invoice_number}</strong>?</p>
      <p class="warning-text">This action cannot be undone.</p>
    </div>

    {#snippet footer()}
      
        <Button variant="ghost" on:click={() => (showDeleteModal = false)} disabled={deleteLoading}>
          Cancel
        </Button>
        <Button variant="danger" on:click={handleDeleteInvoice} disabled={deleteLoading}>
          {deleteLoading ? 'Deleting...' : 'Delete Invoice'}
        </Button>
      
      {/snippet}
  </Modal>
{/if}

<!-- SPOC #9: Credit-Limit Override Modal (management) -->
{#if creditOverrideOpen}
  <Modal
    title="Credit Limit Exceeded"
    open={creditOverrideOpen}
    on:close={cancelCreditOverride}
    size="sm"
  >
    <div class="credit-override-dialog">
      <p>
        This customer is over their credit limit, so the invoice for order
        {creditOverrideOrderLabel || 'this order'} was blocked. An Admin or Manager can
        override this. Enter a reason to proceed — it will be recorded with the override.
      </p>
      <FormGroup label="Override Reason" required>
        <textarea
          class="override-reason"
          bind:value={creditOverrideReason}
          rows="3"
          placeholder="e.g. Approved by management; payment plan agreed"
          disabled={creditOverrideSubmitting}
        ></textarea>
      </FormGroup>
    </div>

    {#snippet footer()}
      <Button variant="ghost" on:click={cancelCreditOverride} disabled={creditOverrideSubmitting}>
        Cancel
      </Button>
      <Button
        variant="danger"
        on:click={submitCreditOverride}
        disabled={creditOverrideSubmitting || !creditOverrideReason.trim()}
      >
        {creditOverrideSubmitting ? 'Overriding...' : 'Override & Create Invoice'}
      </Button>
    {/snippet}
  </Modal>
{/if}

{#if showEditModal && editInvoice}
  <Modal
    title="Edit Invoice"
    open={showEditModal}
    on:close={() => { showEditModal = false; editInvoice = null; editInvoiceOriginalStatus = null; }}
    size="md"
  >
    <div class="edit-form">
      <div class="form-row">
        <div class="form-field">
          <label for="edit-invoice-number">Invoice Number</label>
          <input id="edit-invoice-number" type="text" value={editInvoice.invoice_number} class="input-field" disabled title="Invoice numbers are system-generated" />
        </div>
        <div class="form-field">
          <label for="edit-invoice-status">Status</label>
          <!-- Wave 9.6 AR2: only offer legal hand-set transitions. Settlement
               statuses (Paid/PartiallyPaid/Overdue) are payment-derived and
               must never be options here (mirrors the backend settlement-status
               guard). Once an invoice is Sent+ it is posted and the backend
               (Wave 9.6 AR1) rejects reverting it to Draft, so the dropdown
               only opens up Draft->Sent while still Draft. -->
          {#if editInvoiceOriginalStatus === 'Draft'}
            <select id="edit-invoice-status" bind:value={editInvoice.status} class="input-field">
              <option value="Draft">Draft</option>
              <option value="Sent">Sent</option>
            </select>
          {:else}
            <input
              id="edit-invoice-status"
              type="text"
              value={editInvoice.status}
              class="input-field"
              disabled
              title="Settlement status is derived from payments and cannot be hand-set; a posted invoice cannot be reverted to Draft"
            />
          {/if}
        </div>
      </div>
      <div class="form-row">
        <div class="form-field">
          <label for="edit-invoice-amount">Amount (BHD) <span class="required">*</span></label>
          {#if editInvoice.status === 'Draft'}
            <input
              id="edit-invoice-amount"
              type="text"
              value={formatBHD(editDraftGrandTotal)}
              class="input-field"
              disabled
              title="Derived from the line items below (subtotal + VAT)"
            />
          {:else}
            <input
              id="edit-invoice-amount"
              type="number"
              step="0.001"
              min="0.001"
              bind:value={editInvoice.grand_total_bhd}
              class="input-field"
              required
            />
          {/if}
        </div>
        <div class="form-field">
          <label for="edit-invoice-outstanding">Outstanding (BHD) <span class="required">*</span></label>
          <input
            id="edit-invoice-outstanding"
            type="number"
            step="0.001"
            min="0"
            bind:value={editInvoice.outstanding_bhd}
            class="input-field"
            required
          />
        </div>
      </div>
      <div class="form-field">
        <label for="edit-invoice-customer-po">Customer PO Number</label>
        <input id="edit-invoice-customer-po" type="text" bind:value={editInvoice.customer_po_number} class="input-field" />
      </div>

      <!-- Line-item editor (Draft only). The Amount above is derived from these
           lines + VAT; the backend recomputes Subtotal/VAT/Grand on save. -->
      {#if editInvoice.status === 'Draft'}
        <div class="line-items-panel">
          <div class="line-items-panel-header">
            <div>
              <h4>Invoice Line Items</h4>
              <p>Edit each line separately. The Amount above recalculates from these lines plus {editDraftVatPct}% VAT.</p>
            </div>
            <Button variant="ghost" type="button" on:click={addInvoiceLine}>
              + Add Line
            </Button>
          </div>

          <div class="line-items-list">
            {#each editInvoice.items as item, index}
              <div class="line-item-editor">
                <div class="line-item-editor-head">
                  <span>Line {index + 1}</span>
                  <button
                    type="button"
                    class="line-remove-btn"
                    onclick={() => removeInvoiceLine(index)}
                    aria-label="Remove invoice line"
                  >
                    Remove
                  </button>
                </div>

                <div class="line-item-top-row">
                  <label class="line-field">
                    <span>Description</span>
                    <input
                      class="input-field compact"
                      bind:value={editInvoice.items[index].description}
                      placeholder="Description"
                    />
                  </label>
                </div>

                <div class="line-item-bottom-row">
                  <label class="line-field">
                    <span>Quantity</span>
                    <input
                      class="input-field compact number-input"
                      type="number"
                      min="0"
                      step="0.001"
                      bind:value={editInvoice.items[index].quantity}
                    />
                  </label>
                  <label class="line-field">
                    <span>Rate (BHD)</span>
                    <input
                      class="input-field compact number-input"
                      type="number"
                      min="0"
                      step="0.001"
                      bind:value={editInvoice.items[index].rate}
                    />
                  </label>
                  <div class="line-total-cell">
                    <span>Line Total</span>
                    <strong>{formatBHD(roundMoney((item.quantity || 0) * (item.rate || 0)))}</strong>
                  </div>
                </div>
              </div>
            {/each}

            {#if editInvoice.items.length === 0}
              <div class="line-item-alert">
                No line items yet. Use <strong>+ Add Line</strong> to build this invoice.
              </div>
            {/if}
          </div>

          <div class="line-items-totals">
            <div class="totals-row">
              <span>Subtotal</span>
              <span class="mono">{formatBHD(editDraftSubtotal)} BHD</span>
            </div>
            <div class="totals-row">
              <span>VAT ({editDraftVatPct}%)</span>
              <span class="mono">{formatBHD(roundMoney(editDraftSubtotal * editDraftVatPct / 100))} BHD</span>
            </div>
            <div class="totals-row grand">
              <span>Amount</span>
              <span class="mono">{formatBHD(editDraftGrandTotal)} BHD</span>
            </div>
          </div>
        </div>
      {/if}
    </div>

    {#snippet footer()}
      
        <Button variant="ghost" on:click={() => { showEditModal = false; editInvoice = null; editInvoiceOriginalStatus = null; }} disabled={editLoading}>
          Cancel
        </Button>
        <Button variant="primary" on:click={saveInvoiceEdit} disabled={editLoading}>
          {editLoading ? 'Saving...' : 'Save Changes'}
        </Button>
      
      {/snippet}
  </Modal>
{/if}

<!-- Phase 23: Credit Note Creation Modal -->
{#if showCNModal}
  <Modal
    title="Issue Credit Note"
    open={showCNModal}
    on:close={() => showCNModal = false}
    size="md"
  >
    <div class="cn-form">
      <FormGroup label="Against Invoice" required>
        <select class="select-input" bind:value={cnInvoiceId} disabled={cnSaving}>
          <option value="">-- Select an invoice --</option>
          {#each invoices.filter(i => i.status !== 'Draft') as inv}
            <option value={inv.id}>
              {inv.invoice_number} - {inv.customer_name} ({formatBHD(inv.grand_total_bhd)} BHD)
            </option>
          {/each}
        </select>
      </FormGroup>

      <FormGroup label="Reason" required>
        <textarea
          class="cn-textarea"
          bind:value={cnReason}
          placeholder="e.g. Goods returned, pricing correction, quantity adjustment..."
          rows="3"
          disabled={cnSaving}
        ></textarea>
      </FormGroup>

      <div class="cn-items-section">
        <div class="cn-items-header">
          <h4>Credit Note Items</h4>
          <button class="cn-add-item-btn" onclick={addCNItem} disabled={cnSaving}>+ Add Item</button>
        </div>

        {#each cnItems as item, i}
          <div class="cn-item-row">
            <div class="cn-item-desc">
              <input
                type="text"
                class="input-field"
                placeholder="Description"
                bind:value={item.description}
                disabled={cnSaving}
              />
            </div>
            <div class="cn-item-qty">
              <input
                type="number"
                class="input-field"
                placeholder="Qty"
                min="1"
                step="1"
                bind:value={item.quantity}
                disabled={cnSaving}
              />
            </div>
            <div class="cn-item-rate">
              <input
                type="number"
                class="input-field"
                placeholder="Rate (BHD)"
                min="0"
                step="0.001"
                bind:value={item.rate}
                disabled={cnSaving}
              />
            </div>
            <div class="cn-item-total">
              {formatBHD(item.quantity * item.rate)}
            </div>
            <button
              class="cn-remove-btn"
              onclick={() => removeCNItem(i)}
              disabled={cnSaving || cnItems.length <= 1}
              title="Remove item"
            >
              &times;
            </button>
          </div>
        {/each}

        <div class="cn-total-row">
          <span>Subtotal:</span>
          <span class="cn-total-value">
            {formatBHD(cnItems.reduce((s, it) => s + (it.quantity * it.rate), 0))} BHD
          </span>
        </div>
      </div>
    </div>

    {#snippet footer()}
      
        <Button variant="ghost" on:click={() => showCNModal = false} disabled={cnSaving}>
          Cancel
        </Button>
        <Button variant="primary" on:click={handleCreateCN} disabled={cnSaving || !cnInvoiceId}>
          {cnSaving ? 'Creating...' : 'Issue Credit Note'}
        </Button>
      
      {/snippet}
  </Modal>
{/if}

<!-- Invoice Detail Modal -->
{#if showDetailModal && selectedInvoice}
  <Modal
    title="Invoice Details"
    open={showDetailModal}
    on:close={() => { showDetailModal = false; selectedInvoice = null; }}
    size="lg"
  >
    <div class="invoice-detail">
      <!-- Header Info -->
      <div class="detail-header">
        <div class="detail-main">
          <h2 class="invoice-number">{selectedInvoice.invoice_number}</h2>
          <span class="status-badge" style="background: {statusColors[selectedInvoice.status] || '#6B7280'}15; color: {statusColors[selectedInvoice.status] || '#6B7280'};">
            {selectedInvoice.status}
          </span>
        </div>
        <div class="detail-customer">{selectedInvoice.customer_name}</div>
      </div>

      <!-- Key Info Grid -->
      <div class="detail-grid">
        <div class="detail-item">
          <span class="detail-label">Invoice Date</span>
          <span class="detail-value">{formatDate(selectedInvoice.invoice_date)}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">Due Date</span>
          <span class="detail-value" class:overdue={isDateOverdue(selectedInvoice.due_date) && selectedInvoice.outstanding_bhd > 0}>
            {formatDate(selectedInvoice.due_date)}
          </span>
        </div>
        <div class="detail-item">
          <span class="detail-label">Amount (BHD)</span>
          <span class="detail-value mono">{formatBHD(selectedInvoice.grand_total_bhd)}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">Outstanding (BHD)</span>
          <span class="detail-value mono" class:warning={selectedInvoice.outstanding_bhd > 0}>
            {formatBHD(selectedInvoice.outstanding_bhd)}
          </span>
        </div>
        {#if selectedInvoice.customer_po_number}
          <div class="detail-item">
            <span class="detail-label">Customer PO</span>
            <span class="detail-value">{selectedInvoice.customer_po_number}</span>
          </div>
        {/if}
        {#if selectedInvoice.delivery_note_number}
          <div class="detail-item">
            <span class="detail-label">Delivery Note</span>
            <span class="detail-value dn-link">{selectedInvoice.delivery_note_number}</span>
          </div>
        {/if}
        {#if selectedInvoice.offer_number}
          <div class="detail-item">
            <span class="detail-label">Offer</span>
            <span class="detail-value">{selectedInvoice.offer_number}</span>
          </div>
        {/if}
        {#if selectedInvoice.customer_reference}
          <div class="detail-item">
            <span class="detail-label">RFQ / Reference</span>
            <span class="detail-value">{selectedInvoice.customer_reference}</span>
          </div>
        {/if}
        {#if selectedInvoice.attention_person || selectedInvoice.attention_company}
          <div class="detail-item">
            <span class="detail-label">Attention</span>
            <span class="detail-value">
              {selectedInvoice.attention_person || '—'}{selectedInvoice.attention_company ? ` · ${selectedInvoice.attention_company}` : ''}
            </span>
          </div>
        {/if}
        {#if selectedInvoice.delivery_weeks}
          <div class="detail-item">
            <span class="detail-label">Delivery Time</span>
            <span class="detail-value">{selectedInvoice.delivery_weeks}</span>
          </div>
        {/if}
        {#if selectedInvoice.country_of_origin}
          <div class="detail-item">
            <span class="detail-label">Country of Origin</span>
            <span class="detail-value">{selectedInvoice.country_of_origin}</span>
          </div>
        {/if}
        {#if selectedInvoice.payment_terms}
          <div class="detail-item">
            <span class="detail-label">Payment Terms</span>
            <span class="detail-value">{selectedInvoice.payment_terms}</span>
          </div>
        {/if}
        {#if selectedInvoice.delivery_terms}
          <div class="detail-item">
            <span class="detail-label">Delivery Terms</span>
            <span class="detail-value">{selectedInvoice.delivery_terms}</span>
          </div>
        {/if}
      </div>

      <!-- Line Items Section -->
      {#if selectedInvoice.items && selectedInvoice.items.length > 0}
        <div class="line-items-section">
          <h4>Line Items ({selectedInvoice.items.length})</h4>
          <div class="items-table">
            <div class="items-header">
              <span class="item-col-num">#</span>
              <span class="item-col-desc">Description</span>
              <span class="item-col-qty">Qty</span>
              <span class="item-col-rate">Rate</span>
              <span class="item-col-total">Total (BHD)</span>
            </div>
            {#each selectedInvoice.items as item, i}
              <div class="items-row">
                <span class="item-col-num">{item.line_number || i + 1}</span>
                <span class="item-col-desc">
                  <strong>{item.equipment || item.description || `Item ${i + 1}`}</strong>
                  {#if item.model || item.product_code}
                    <small>{item.model || item.product_code}</small>
                  {/if}
                  {#if item.specification}
                    <small>{item.specification}</small>
                  {/if}
                  {#if item.detailed_description}
                    <small>{item.detailed_description}</small>
                  {/if}
                </span>
                <span class="item-col-qty">{item.quantity || 0}</span>
                <span class="item-col-rate">{formatBHD(item.rate || 0)}</span>
                <span class="item-col-total">{formatBHD(item.total_bhd || 0)}</span>
              </div>
            {/each}
            <div class="items-footer">
              <span class="footer-label">Grand Total</span>
              <span class="footer-value">{formatBHD(selectedInvoice.grand_total_bhd)} BHD</span>
            </div>
          </div>
        </div>
      {:else}
        <div class="no-items-hint">
          <p>No line items found for this invoice.</p>
        </div>
      {/if}
    </div>

    {#snippet footer()}
      
        <Button variant="ghost" on:click={() => { showDetailModal = false; selectedInvoice = null; }}>
          Close
        </Button>
        <Button variant="secondary" on:click={() => downloadPDF(selectedInvoice)} disabled={pdfLoadingMap[selectedInvoice?.id || '']}>
          {pdfLoadingMap[selectedInvoice?.id || ''] ? 'Generating...' : 'Download PDF'}
        </Button>
        <Button variant="primary" on:click={() => { showDetailModal = false; openEditModal(selectedInvoice); }}>
          Edit Invoice
        </Button>
      
      {/snippet}
  </Modal>
{/if}

<style>
  .invoices-container {
    display: flex;
    flex-direction: column;
    gap: 16px;
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

  .kpi-currency {
    font-size: 16px;
    font-weight: 500;
    color: var(--text-secondary);
    margin-left: 4px;
  }

  .kpi-primary {
    color: var(--brand-indigo);
  }

  .kpi-warning {
    color: #f59e0b;
  }

  .kpi-danger {
    color: #ef4444;
  }

  .kpi-sublabel {
    font-size: 12px;
    color: var(--text-secondary);
  }

  /* Controls Row */
  .controls-row {
    display: flex;
    gap: 16px;
    align-items: center;
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

  /* Action Buttons in Table */
  .action-buttons {
    display: flex;
    gap: 8px;
    justify-content: center;
  }

  .action-btn {
    padding: 6px 12px;
    font-size: 12px;
    font-weight: 600;
    border: none;
    border-radius: var(--border-radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .action-send {
    background: var(--brand-indigo);
    color: white;
  }

  .action-send:hover {
    background: var(--brand-indigo-hover);
    transform: translateY(-1px);
  }

  .action-delete {
    background: #fee2e2;
    color: #991b1b;
  }

  .action-delete:hover {
    background: #fecaca;
    transform: translateY(-1px);
  }

  .action-edit {
    background: var(--bg-secondary, #f3f4f6);
    color: var(--ink, #1d1d1f);
  }
  .action-edit:hover {
    background: var(--border-medium, #e5e5e5);
    transform: translateY(-1px);
  }
  .action-pdf {
    background: #ede9fe;
    color: #5b21b6;
  }
  .action-pdf:hover {
    background: #ddd6fe;
    transform: translateY(-1px);
  }

  .edit-form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .form-row {
    display: flex;
    gap: 12px;
  }
  .form-field {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .form-field label {
    font-size: 12px;
    font-weight: 500;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .input-field {
    padding: 8px 12px;
    border: 1px solid var(--border-medium, #e5e5e5);
    border-radius: 6px;
    font-size: 14px;
    font-family: inherit;
  }
  .input-field:focus {
    border-color: var(--ink, #1d1d1f);
    outline: none;
  }

  /* SPOC #9: credit-limit override modal */
  .credit-override-dialog {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .credit-override-dialog p {
    margin: 0;
    font-size: 13px;
    line-height: 1.6;
    color: var(--text-secondary);
  }

  .override-reason {
    width: 100%;
    box-sizing: border-box;
    min-height: 72px;
    padding: 8px 12px;
    font-size: 14px;
    font-family: var(--font-family);
    color: var(--text-primary);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
    resize: vertical;
  }

  /* Draft invoice line-item editor */
  .line-items-panel {
    margin-top: 4px;
    padding: 16px;
    border: 1px solid var(--border);
    border-radius: var(--border-radius);
    background: var(--bg-subtle);
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .line-items-panel-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }

  .line-items-panel-header h4 {
    margin: 0;
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .line-items-panel-header p {
    margin: 4px 0 0;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .line-items-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .line-item-editor {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .line-item-editor-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    color: var(--text-secondary);
  }

  .line-item-top-row {
    display: grid;
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .line-item-bottom-row {
    display: grid;
    grid-template-columns: minmax(120px, 0.4fr) minmax(150px, 0.5fr) minmax(160px, 0.5fr);
    gap: 12px;
    align-items: end;
  }

  .line-field {
    display: flex;
    flex-direction: column;
    gap: 5px;
    min-width: 0;
  }

  .line-field span,
  .line-total-cell span {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    color: var(--text-secondary);
  }

  .compact {
    min-width: 0;
  }

  .number-input {
    text-align: right;
    font-family: var(--font-mono);
  }

  .line-total-cell {
    min-height: 36px;
    padding: 8px 10px;
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
    background: var(--bg-subtle);
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 2px;
    text-align: right;
    font-size: 12px;
    font-family: var(--font-mono);
    color: var(--text-primary);
  }

  .line-total-cell strong {
    font-size: 13px;
    font-weight: 700;
  }

  .line-remove-btn {
    border: 1px solid var(--border);
    background: transparent;
    color: var(--text-secondary);
    border-radius: var(--border-radius-sm);
    padding: 8px 10px;
    font-size: 12px;
    cursor: pointer;
  }

  .line-item-alert {
    padding: 10px 12px;
    background: rgba(245, 158, 11, 0.1);
    border: 1px solid rgba(245, 158, 11, 0.28);
    border-radius: var(--border-radius-sm);
    color: #92400e;
    font-size: 12px;
    line-height: 1.5;
  }

  .line-items-totals {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding-top: 8px;
    border-top: 1px solid var(--border);
  }

  .line-items-totals .totals-row {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .line-items-totals .totals-row .mono {
    font-family: var(--font-mono);
  }

  .line-items-totals .totals-row.grand {
    font-size: 14px;
    font-weight: 700;
    color: var(--text-primary);
    margin-top: 2px;
  }

  @media (max-width: 840px) {
    .line-item-bottom-row {
      grid-template-columns: 1fr;
    }

    .line-total-cell {
      text-align: left;
    }
  }

  /* Modal Forms */
  .create-form {
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

  .empty-hint {
    padding: 16px;
    background: var(--surface-elevated);
    border-radius: var(--border-radius-sm);
    border-left: 3px solid #f59e0b;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .empty-hint p {
    margin: 0;
    font-size: 13px;
    color: var(--text-secondary);
  }

  /* Delivery Note selector styles */
  .dn-loading {
    padding: 12px;
    font-size: 13px;
    color: var(--text-secondary);
    font-style: italic;
  }

  .dn-empty-hint {
    padding: 12px;
    background: var(--surface-elevated, #f9fafb);
    border-radius: var(--border-radius-sm);
    border-left: 3px solid var(--steel, #86868b);
  }

  .dn-empty-hint p {
    margin: 0;
    font-size: 13px;
    color: var(--text-secondary);
  }

  .dn-hint {
    margin: 8px 0 0 0;
    font-size: 12px;
    color: var(--text-secondary);
  }

  /* Field Visibility Section */
  .field-visibility-section {
    margin-top: 20px;
    padding: 12px 16px;
    background: var(--surface-elevated, #f9fafb);
    border-radius: var(--border-radius-sm, 6px);
    border: 1px solid var(--border, #e5e5e5);
  }

  /* B6: disclosure toggle for the PDF field-visibility panel — collapsed by
     default so it doesn't compete with the primary create-invoice flow. */
  .expand-toggle {
    background: none;
    border: none;
    color: var(--text-secondary);
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    padding: 4px 0;
    text-align: left;
    width: 100%;
  }

  .expand-toggle:hover {
    color: var(--brand-indigo);
  }

  .visibility-hint {
    margin: 12px 0 0 0;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .visibility-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
    margin-top: 12px;
  }

  .visibility-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .group-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    padding-bottom: 4px;
    border-bottom: 1px solid var(--border, #e5e5e5);
    margin-bottom: 4px;
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: var(--text-primary);
    cursor: pointer;
  }

  .checkbox-label input[type="checkbox"] {
    width: 16px;
    height: 16px;
    accent-color: var(--carbon, #000);
    cursor: pointer;
  }

  @media (max-width: 768px) {
    .visibility-grid {
      grid-template-columns: 1fr;
    }
  }

  .delete-confirmation {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .delete-confirmation p {
    margin: 0;
    font-size: 14px;
    color: var(--text-primary);
  }

  .warning-text {
    color: #ef4444;
    font-weight: 500;
    font-size: 13px;
  }

  /* Required field indicator */
  .required {
    color: #ef4444;
    margin-left: 2px;
    font-weight: 600;
  }

  /* Invoice Detail Modal */
  .invoice-detail {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .detail-header {
    border-bottom: 2px solid var(--text-primary);
    padding-bottom: 16px;
  }

  .detail-main {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 4px;
  }

  .invoice-number {
    margin: 0;
    font-size: 24px;
    font-weight: 600;
    font-family: var(--font-mono);
    color: var(--brand-indigo);
  }

  .status-badge {
    display: inline-block;
    padding: 4px 12px;
    border-radius: 12px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
  }

  .detail-customer {
    font-size: 16px;
    color: var(--text-secondary);
  }

  .detail-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 16px;
    padding: 16px;
    background: var(--surface-elevated, #f9fafb);
    border-radius: 8px;
  }

  .detail-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .detail-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .detail-value {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary);
  }

  .detail-value.mono {
    font-family: var(--font-mono);
  }

  .detail-value.overdue {
    color: #ef4444;
    font-weight: 600;
  }

  .detail-value.warning {
    color: #f59e0b;
  }

  .detail-value.dn-link {
    color: #059669;
    font-family: var(--font-mono);
  }

  /* Line Items in Detail Modal */
  .line-items-section {
    border: 1px solid var(--border, #e5e5e5);
    border-radius: 8px;
    overflow: hidden;
  }

  .line-items-section h4 {
    margin: 0;
    padding: 12px 16px;
    font-size: 14px;
    font-weight: 600;
    background: var(--surface-elevated, #f9fafb);
    border-bottom: 1px solid var(--border, #e5e5e5);
  }

  .items-table {
    width: 100%;
  }

  .items-header {
    display: grid;
    grid-template-columns: 40px 2fr 80px 100px 120px;
    gap: 8px;
    padding: 10px 16px;
    background: rgba(0, 0, 0, 0.02);
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .items-row {
    display: grid;
    grid-template-columns: 40px 2fr 80px 100px 120px;
    gap: 8px;
    padding: 10px 16px;
    border-bottom: 1px solid rgba(0, 0, 0, 0.05);
    font-size: 13px;
  }

  .items-row:last-child {
    border-bottom: none;
  }

  .item-col-num {
    color: var(--text-secondary);
    font-family: var(--font-mono);
  }

  .item-col-desc {
    display: grid;
    gap: 4px;
    min-width: 0;
  }

  .item-col-desc strong {
    font-weight: 600;
    color: var(--text-primary);
  }

  .item-col-desc small {
    color: var(--text-secondary);
    line-height: 1.45;
  }

  .item-col-qty,
  .item-col-rate,
  .item-col-total {
    text-align: right;
    font-family: var(--font-mono);
  }

  .items-footer {
    display: flex;
    justify-content: flex-end;
    gap: 24px;
    padding: 12px 16px;
    background: var(--surface-elevated, #f9fafb);
    border-top: 2px solid var(--text-primary);
  }

  .footer-label {
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    color: var(--text-secondary);
  }

  .footer-value {
    font-size: 16px;
    font-weight: 700;
    font-family: var(--font-mono);
    color: var(--text-primary);
  }

  .no-items-hint {
    padding: 24px;
    text-align: center;
    background: var(--surface-elevated, #f9fafb);
    border-radius: 8px;
  }

  .no-items-hint p {
    margin: 0;
    font-size: 13px;
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

  /* Phase 23: Credit Notes Table */
  .cn-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }

  .cn-table th {
    text-align: left;
    padding: 10px 12px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    border-bottom: 2px solid var(--border, #e5e5e5);
  }

  .cn-table th.right {
    text-align: right;
  }

  .cn-table td {
    padding: 10px 12px;
    border-bottom: 1px solid var(--border, #e5e5e5);
    vertical-align: middle;
  }

  .cn-table td.right {
    text-align: right;
  }

  .cn-table td.mono {
    font-family: var(--font-mono);
    font-size: 12px;
  }

  .cn-table tr:hover {
    background: var(--interactive-hover, rgba(0,0,0,0.02));
  }

  .cn-status {
    display: inline-block;
    padding: 3px 10px;
    border-radius: 10px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
  }

  .cn-draft {
    background: #6b728015;
    color: #6b7280;
  }

  .cn-issued {
    background: #3b82f615;
    color: #3b82f6;
  }

  .cn-applied {
    background: #10b98115;
    color: #10b981;
  }

  .action-secondary {
    background: #ede9fe;
    color: #5b21b6;
  }

  .action-secondary:hover {
    background: #ddd6fe;
  }

  .action-primary {
    background: var(--brand-indigo, #6366f1);
    color: white;
  }

  .action-primary:hover {
    background: var(--brand-indigo-hover, #4f46e5);
  }

  /* Phase 23: Credit Note Form */
  .cn-form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .cn-textarea {
    width: 100%;
    padding: 10px 12px;
    font-size: 14px;
    font-family: inherit;
    color: var(--text-primary);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
    resize: vertical;
  }

  .cn-textarea:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 3px var(--brand-indigo-tint);
  }

  .cn-items-section {
    border: 1px solid var(--border, #e5e5e5);
    border-radius: 8px;
    padding: 12px;
  }

  .cn-items-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }

  .cn-items-header h4 {
    margin: 0;
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .cn-add-item-btn {
    padding: 4px 12px;
    font-size: 12px;
    font-weight: 600;
    background: var(--brand-indigo, #6366f1);
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  .cn-add-item-btn:hover {
    background: var(--brand-indigo-hover, #4f46e5);
  }

  .cn-item-row {
    display: grid;
    grid-template-columns: 2fr 80px 100px 80px 32px;
    gap: 8px;
    align-items: center;
    margin-bottom: 8px;
  }

  .cn-item-row .input-field {
    padding: 6px 8px;
    font-size: 13px;
  }

  .cn-item-total {
    text-align: right;
    font-family: var(--font-mono);
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary);
  }

  .cn-remove-btn {
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #fee2e2;
    color: #991b1b;
    border: none;
    border-radius: 4px;
    font-size: 16px;
    cursor: pointer;
  }

  .cn-remove-btn:hover:not(:disabled) {
    background: #fecaca;
  }

  .cn-remove-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .cn-total-row {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding-top: 10px;
    border-top: 1px solid var(--border, #e5e5e5);
    margin-top: 4px;
  }

  .cn-total-row span {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .cn-total-value {
    font-family: var(--font-mono);
    font-weight: 600 !important;
    color: var(--text-primary) !important;
  }

  /* Responsive */
  @media (max-width: 768px) {
    .kpi-grid {
      grid-template-columns: 1fr;
    }

    .controls-row {
      flex-direction: column;
      align-items: stretch;
    }

    .filter-tabs {
      flex-wrap: wrap;
    }

    .items-header,
    .items-row {
      grid-template-columns: 30px 1fr 60px 80px 90px;
      font-size: 11px;
    }
  }
</style>
