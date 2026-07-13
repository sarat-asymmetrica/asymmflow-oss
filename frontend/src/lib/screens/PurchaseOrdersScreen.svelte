<script lang="ts">
  import { run, preventDefault } from 'svelte/legacy';
  import { motionMs } from "$lib/motion";

  /**
   * PurchaseOrdersScreen - Production-Ready Purchase Orders Management
   *
   * Features:
   * - View all purchase orders with filtering by status
   * - Create new POs from orders or manually
   * - Multi-currency support (EUR, USD, BHD)
   * - Status tracking: Draft → Sent → Acknowledged → Partially Received → Received → Closed
   * - Link to parent orders and suppliers
   * - Payment tracking with due dates
   *
   * Design System: Wabi-Sabi minimalism × Bloomberg data density
   */

  import { onMount, onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';

  // Wails API imports
  import {
    GetPurchaseOrders, ReceiveAndCompletePO, ReceiveAndCompletePOWithSerials, RaiseGRNDiscrepancy } from '../../../wailsjs/go/main/App';
import { GetPurchaseOrderByID, CreatePurchaseOrder, UpdatePurchaseOrder, UpdatePOStatus, ApprovePurchaseOrder, DeletePurchaseOrder, ListOrders, ListSuppliers } from '../../../wailsjs/go/main/CRMService';
import { GeneratePurchaseOrderPDF } from '../../../wailsjs/go/main/DocumentsService';

  // Design system components
  import PageLayout from '$lib/components/layout/PageLayout.svelte';
  import DataTable from '$lib/components/ui/DataTable.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import StatusBadge from '$lib/components/ui/StatusBadge.svelte';
  import WabiModal from '$lib/components/ui/WabiModal.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import FormGroup from '$lib/components/ui/FormGroup.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import { toast } from '$lib/stores/toasts';
  import { confirm } from '$lib/stores/confirm';
  import { escapeHtml } from '$lib/utils/escapeHtml';
  import { formatBHD, formatNumber } from '$lib/utils/formatters';
  import { permissions, currentUser } from '$lib/stores/authContext';

  let permissionList = $derived(Array.isArray($permissions) ? $permissions : []);
  let canView = $derived(permissionList.includes('*') || permissionList.includes('po:view') || permissionList.includes('po:*'));

  
  interface Props {
    // Props
    embedded?: boolean;
  }

  let { embedded = false }: Props = $props();

  // Types
  type POStatus = 'all' | 'Draft' | 'Sent' | 'Acknowledged' | 'Partially Received' | 'Received' | 'Closed' | 'Cancelled';

  function getErrorMessage(err: unknown): string {
    if (!err) return 'Unknown error';
    if (err instanceof Error && err.message) return err.message;
    if (typeof err === 'string') return err;
    if (typeof err === 'object') {
      const candidate = (err as any).message || (err as any).error || (err as any).cause;
      if (candidate) return String(candidate);
    }
    return String(err);
  }

  function padDatePart(value: number): string {
    return String(value).padStart(2, '0');
  }

  function formatDateInput(value: string | Date | null | undefined): string {
    if (!value) return '';

    if (typeof value === 'string') {
      const trimmed = value.trim();
      if (!trimmed) return '';

      const isoDateMatch = trimmed.match(/^(\d{4})-(\d{2})-(\d{2})/);
      if (isoDateMatch) {
        return `${isoDateMatch[1]}-${isoDateMatch[2]}-${isoDateMatch[3]}`;
      }

      const localDateMatch = trimmed.match(/^(\d{2})-(\d{2})-(\d{4})$/);
      if (localDateMatch) {
        return `${localDateMatch[3]}-${localDateMatch[2]}-${localDateMatch[1]}`;
      }

      const slashDateMatch = trimmed.match(/^(\d{2})\/(\d{2})\/(\d{4})$/);
      if (slashDateMatch) {
        return `${slashDateMatch[3]}-${slashDateMatch[2]}-${slashDateMatch[1]}`;
      }

      const parsed = new Date(trimmed);
      if (!Number.isNaN(parsed.getTime())) {
        return `${parsed.getFullYear()}-${padDatePart(parsed.getMonth() + 1)}-${padDatePart(parsed.getDate())}`;
      }

      return '';
    }

    return `${value.getFullYear()}-${padDatePart(value.getMonth() + 1)}-${padDatePart(value.getDate())}`;
  }

  function parseLocalDate(value: string): Date | null {
    const trimmed = value.trim();
    if (!trimmed) return null;

    const isoDateMatch = trimmed.match(/^(\d{4})-(\d{2})-(\d{2})$/);
    if (isoDateMatch) {
      return new Date(Number(isoDateMatch[1]), Number(isoDateMatch[2]) - 1, Number(isoDateMatch[3]));
    }

    const localDateMatch = trimmed.match(/^(\d{2})-(\d{2})-(\d{4})$/);
    if (localDateMatch) {
      return new Date(Number(localDateMatch[3]), Number(localDateMatch[2]) - 1, Number(localDateMatch[1]));
    }

    const slashDateMatch = trimmed.match(/^(\d{2})\/(\d{2})\/(\d{4})$/);
    if (slashDateMatch) {
      return new Date(Number(slashDateMatch[3]), Number(slashDateMatch[2]) - 1, Number(slashDateMatch[1]));
    }

    const parsed = new Date(trimmed);
    if (Number.isNaN(parsed.getTime())) {
      return null;
    }

    return new Date(parsed.getFullYear(), parsed.getMonth(), parsed.getDate());
  }

  function addDaysLocal(baseDate: Date, days: number): Date {
    const nextDate = new Date(baseDate.getFullYear(), baseDate.getMonth(), baseDate.getDate());
    nextDate.setDate(nextDate.getDate() + days);
    return nextDate;
  }

  function toLocalDateISOString(value: string): string {
    const parsed = parseLocalDate(value);
    if (!parsed) {
      return '';
    }

    return new Date(
      parsed.getFullYear(),
      parsed.getMonth(),
      parsed.getDate(),
      0,
      0,
      0,
      0
    ).toISOString();
  }

  interface PODisplay {
    id: string;
    po_number: string;
    order_id?: string;
    rfq_id?: string;
    supplier_id: string;
    supplier_name?: string;
    po_date: string;
    expected_delivery: string;
    currency: string;
    exchange_rate: number;
    subtotal_foreign: number;
    subtotal_bhd: number;
    vat_amount: number;
    total_foreign: number;
    total_bhd: number;
    payment_terms: string;
    payment_due_date: string;
    status: string;
    items?: any[];
    created_at?: string;
    updated_at?: string;
  }

  // State
  let purchaseOrders: PODisplay[] = $state([]);
  let filteredPOs: PODisplay[] = $state([]);
  let loading = $state(true);
  let selectedStatus: POStatus = $state('all');

  // Modal state
  let showCreateModal = $state(false);
  let showEditModal = $state(false);
  let showViewModal = $state(false);
  let editingPO: PODisplay | null = null;
  let viewingPO: PODisplay | null = $state(null);

  // Receive Items panel state (Wave 9.7: routes receiving through the real
  // create-GRN -> CompleteGRN chain instead of the cosmetic status flip).
  interface ReceiveLine {
    po_item_id: string;
    product_id: string;
    product_code: string;
    description: string;
    ordered: number;
    alreadyReceived: number;
    remaining: number;
    receiveQty: number;
    rejectedQty: number;
    serialsText: string;
    // Wave 9.8 B1: threaded from PurchaseOrderItem.requires_serial_tracking
    // (query-time overlay populated server-side from ProductMaster — see
    // enrichPOItemsWithSerialTracking in purchase_order_service.go). When
    // true, serial capture is mandatory for this line, not optional.
    requiresSerial: boolean;
    // Wave 9.8 B1: required whenever rejectedQty > 0 — becomes the reason
    // passed to RaiseGRNDiscrepancy after the receive posts.
    rejectionReason: string;
  }
  let showReceiveModal = $state(false);
  let receiveLines: ReceiveLine[] = $state([]);
  let receiveLoading = $state(false);

  // Statuses from which a PO can still receive goods — mirrors the backend's
  // ReceiveAgainstPO/CompleteGRN chain, which only makes sense before the PO
  // is fully Received.
  const RECEIVABLE_STATUSES = ['Sent', 'Acknowledged', 'Partially Received'];

  // Reference data
  let orders: any[] = $state([]);
  let suppliers: any[] = $state([]);

  // Form state
  let formData = $state({
    order_id: '',
    supplier_id: '',
    po_date: formatDateInput(new Date()),
    expected_delivery: formatDateInput(addDaysLocal(new Date(), 14)), // +14 days
    currency: 'BHD',
    exchange_rate: 1.0,
    payment_terms: 'Net 30',
    items: [] as { product_id: string; description: string; quantity: number; unit_price_foreign: number; }[]
  });
  let formLoading = $state(false);
  let statusLoading = $state(false);
  let pdfLoading = $state(false);

  // Currency options
  const currencies = [
    { value: 'BHD', label: 'BHD - Bahraini Dinar' },
    { value: 'USD', label: 'USD - US Dollar' },
    { value: 'EUR', label: 'EUR - Euro' },
    { value: 'GBP', label: 'GBP - British Pound' },
    { value: 'CHF', label: 'CHF - Swiss Franc' },
    { value: 'AED', label: 'AED - UAE Dirham' },
    { value: 'SAR', label: 'SAR - Saudi Riyal' }
  ];

  // Payment terms options
  const paymentTerms = [
    'Net 15',
    'Net 30',
    'Net 45',
    'Net 60',
    'Net 90',
    'Advance Payment',
    'COD',
    'Due on Receipt'
  ];

  // DataTable columns configuration
  const columns = [
    {
      key: 'po_number',
      label: 'PO #',
      sortable: true,
      width: '120px'
    },
    {
      key: 'supplier_name',
      label: 'Supplier',
      sortable: true,
      render: (row) => escapeHtml(row.supplier_name)
    },
    {
      key: 'po_date',
      label: 'PO Date',
      type: 'date' as const,
      sortable: true,
      width: '120px'
    },
    {
      key: 'currency',
      label: 'Currency',
      sortable: true,
      width: '90px',
      render: (row: PODisplay) => {
        return `<span style="font-family: var(--font-mono); font-size: 12px;">${escapeHtml(row.currency || '')}</span>`;
      }
    },
    {
      key: 'subtotal_foreign',
      label: 'Net Amount',
      type: 'currency' as const,
      align: 'right' as const,
      sortable: true,
      width: '140px',
      render: (row: PODisplay) => {
        return `<span style="font-family: var(--font-mono); font-weight: 500;">${formatNumber(row.subtotal_foreign, 3)} ${escapeHtml(row.currency || '')}</span>`;
      }
    },
    {
      key: 'total_bhd',
      label: 'Total incl. VAT',
      type: 'currency' as const,
      align: 'right' as const,
      sortable: true,
      width: '130px'
    },
    {
      key: 'status',
      label: 'Status',
      type: 'status' as const,
      sortable: true,
      width: '140px'
    },
    {
      key: 'actions',
      label: 'Actions',
      type: 'actions' as const,
      width: '280px',
      render: (row: PODisplay) => {
        return `
          <div style="display: flex; gap: 6px; justify-content: flex-end; flex-wrap: wrap;">
            <button
              class="action-btn action-btn-view"
              data-action="view"
              data-id="${row.id}"
              aria-label="View PO details"
            >
              View
            </button>
            <button
              class="action-btn action-btn-edit"
              data-action="edit"
              data-id="${row.id}"
              aria-label="Edit PO"
            >
              Edit
            </button>
            <button
              class="action-btn action-btn-pdf"
              data-action="pdf"
              data-id="${row.id}"
              aria-label="Generate PDF"
              ${pdfLoading ? 'disabled' : ''}
            >
              PDF
            </button>
          </div>
        `;
      }
    }
  ];

  // Status filter tabs
  const statusTabs: { value: POStatus; label: string; count: number }[] = $state([
    { value: 'all', label: 'All POs', count: 0 },
    { value: 'Draft', label: 'Draft', count: 0 },
    { value: 'Sent', label: 'Sent', count: 0 },
    { value: 'Acknowledged', label: 'Acknowledged', count: 0 },
    { value: 'Partially Received', label: 'Partially Received', count: 0 },
    { value: 'Received', label: 'Fully Received', count: 0 },
    { value: 'Closed', label: 'Closed', count: 0 }
  ]);

  // Computed: Update tab counts
  run(() => {
    statusTabs[0].count = purchaseOrders.length;
    statusTabs[1].count = purchaseOrders.filter(p => p.status === 'Draft').length;
    statusTabs[2].count = purchaseOrders.filter(p => p.status === 'Sent').length;
    statusTabs[3].count = purchaseOrders.filter(p => p.status === 'Acknowledged').length;
    statusTabs[4].count = purchaseOrders.filter(p => p.status === 'Partially Received').length;
    statusTabs[5].count = purchaseOrders.filter(p => p.status === 'Received').length;
    statusTabs[6].count = purchaseOrders.filter(p => p.status === 'Closed').length;
  });

  // Computed: Filter POs by selected status
  run(() => {
    if (selectedStatus === 'all') {
      filteredPOs = purchaseOrders;
    } else {
      filteredPOs = purchaseOrders.filter(p => p.status === selectedStatus);
    }
  });

  // Load purchase orders and reference data
  async function loadPurchaseOrders() {
    loading = true;
    try {
      const [posData, ordersData, suppliersData] = await Promise.all([
        GetPurchaseOrders(),
        ListOrders(1000, 0),
        ListSuppliers(1000, 0)
      ]);

      suppliers = suppliersData || [];
      orders = ordersData || [];
      purchaseOrders = (posData || []).map(enrichPO);

      console.log(`Loaded ${purchaseOrders.length} POs, ${orders.length} orders, ${suppliers.length} suppliers`);
    } catch (err) {
      console.error('Failed to load purchase orders:', err);
      const errorMsg = getErrorMessage(err);
      toast.danger(`Failed to load purchase orders: ${errorMsg}`);
      purchaseOrders = [];
    } finally {
      loading = false;
    }
  }

  // Enrich PO with supplier name
  function enrichPO(po: any): PODisplay {
    const exchangeRate = Number(po.exchange_rate || 1) || 1;
    const subtotalForeign = Number(po.subtotal_foreign || 0) || 0;
    const vatAmount = Number(po.vat_amount || 0) || 0;
    const normalizedTotalForeign = Number(po.total_foreign || 0) > subtotalForeign
      ? Number(po.total_foreign || 0)
      : subtotalForeign + vatAmount;
    const normalizedTotalBHD = Number(po.total_bhd || 0) || (normalizedTotalForeign * exchangeRate);

    // If PO already has supplier_name from backend, use it
    if (po.supplier_name) {
      return {
        ...po,
        exchange_rate: exchangeRate,
        subtotal_foreign: subtotalForeign,
        vat_amount: vatAmount,
        total_foreign: normalizedTotalForeign,
        total_bhd: normalizedTotalBHD
      };
    }

    // Otherwise, lookup from suppliers array
    const supplier = suppliers.find(s => s.id === po.supplier_id);
    return {
      ...po,
      exchange_rate: exchangeRate,
      subtotal_foreign: subtotalForeign,
      vat_amount: vatAmount,
      total_foreign: normalizedTotalForeign,
      total_bhd: normalizedTotalBHD,
      supplier_name: supplier?.supplier_name || 'Unknown Supplier'
    };
  }

  // Open create modal
  function openCreateModal() {
    const today = new Date();
    formData = {
      order_id: '',
      supplier_id: '',
      po_date: formatDateInput(today),
      expected_delivery: formatDateInput(addDaysLocal(today, 14)),
      currency: 'BHD',
      exchange_rate: 1.0,
      payment_terms: 'Net 30',
      items: []
    };
    showCreateModal = true;
  }

  // Open edit modal
  async function openEditModal(poId: string) {
    try {
      const po = await GetPurchaseOrderByID(poId);
      editingPO = enrichPO(po);

      formData = {
        order_id: po.order_id || '',
        supplier_id: po.supplier_id || '',
        po_date: po.po_date ? formatDateInput(po.po_date as any) : formatDateInput(new Date()),
        expected_delivery: po.expected_delivery ? formatDateInput(po.expected_delivery as any) : '',
        currency: po.currency || 'BHD',
        exchange_rate: po.exchange_rate || 1.0,
        payment_terms: po.payment_terms || 'Net 30',
        items: (po.items || []).map((item: any) => ({
          product_id: item.product_id || '',
          description: item.description || '',
          quantity: item.quantity || 0,
          unit_price_foreign: item.unit_price_foreign || 0
        }))
      };

      showEditModal = true;
    } catch (err) {
      console.error('Failed to load PO for editing:', err);
      const errorMsg = getErrorMessage(err);
      toast.danger(`Failed to load purchase order: ${errorMsg}`);
    }
  }

  // Open view modal
  async function openViewModal(poId: string) {
    try {
      const po = await GetPurchaseOrderByID(poId);
      viewingPO = enrichPO(po);
      showViewModal = true;
    } catch (err) {
      console.error('Failed to load PO for viewing:', err);
      const errorMsg = getErrorMessage(err);
      toast.danger(`Failed to load purchase order: ${errorMsg}`);
    }
  }

  // Handle create PO
  async function handleCreatePO() {
    if (!formData.supplier_id || !formData.po_date) {
      toast.warning('Please fill all required fields');
      return;
    }

    // Validate dates
    const poDate = parseLocalDate(formData.po_date);
    const expectedDelivery = parseLocalDate(formData.expected_delivery);
    const today = new Date();
    const todayDate = new Date(today.getFullYear(), today.getMonth(), today.getDate());

    if (!poDate || !expectedDelivery) {
      toast.warning('Please provide valid PO and expected delivery dates');
      return;
    }

    if (poDate.getTime() > todayDate.getTime()) {
      toast.warning('PO date cannot be in the future');
      return;
    }

    if (expectedDelivery.getTime() <= poDate.getTime()) {
      toast.warning('Expected delivery must be after PO date');
      return;
    }

    // Validate at least one item
    if (formData.items.length === 0) {
      toast.warning('Please add at least one line item');
      return;
    }

    // Validate all items
    for (let i = 0; i < formData.items.length; i++) {
      const item = formData.items[i];
      if (item.quantity < 1) {
        toast.warning(`Item ${i + 1}: Quantity must be at least 1`);
        return;
      }
      if (item.unit_price_foreign <= 0) {
        toast.warning(`Item ${i + 1}: Unit price must be greater than 0`);
        return;
      }
      if (!item.description || !item.description.trim()) {
        toast.warning(`Item ${i + 1}: Description is required`);
        return;
      }
    }

    // Validate exchange rate
    if (formData.exchange_rate <= 0) {
      toast.warning('Exchange rate must be greater than 0');
      return;
    }

    formLoading = true;
    try {
      const selectedSupplier = suppliers.find((supplier) => supplier.id === formData.supplier_id);
      const poData: any = {
        order_id: formData.order_id || undefined,
        supplier_id: formData.supplier_id,
        supplier_name: selectedSupplier?.supplier_name || '',
        po_date: toLocalDateISOString(formData.po_date),
        expected_delivery: toLocalDateISOString(formData.expected_delivery),
        currency: formData.currency,
        exchange_rate: formData.exchange_rate,
        payment_terms: formData.payment_terms,
        status: 'Draft',
        items: formData.items.map(item => ({
          ...item,
          total_foreign: item.quantity * item.unit_price_foreign
        }))
      };

      // Calculate totals
      const subtotal = poData.items.reduce((sum: number, item: any) => sum + item.total_foreign, 0);
      poData.subtotal_foreign = subtotal;
      poData.subtotal_bhd = subtotal * formData.exchange_rate;
      poData.vat_amount = subtotal * 0.10; // 10% VAT
      poData.total_foreign = subtotal + poData.vat_amount;
      poData.total_bhd = poData.total_foreign * formData.exchange_rate;

      await CreatePurchaseOrder(poData);
      toast.success('Purchase order created successfully');
      showCreateModal = false;
      await loadPurchaseOrders();
    } catch (err) {
      console.error('Failed to create PO:', err);
      toast.danger('Failed to create purchase order: ' + getErrorMessage(err));
    } finally {
      formLoading = false;
    }
  }

  // Handle edit PO
  async function handleEditPO() {
    if (!editingPO) return;

    // Same validations as create
    if (!formData.supplier_id || !formData.po_date) {
      toast.warning('Please fill all required fields');
      return;
    }

    const poDate = parseLocalDate(formData.po_date);
    const expectedDelivery = parseLocalDate(formData.expected_delivery);
    const today = new Date();
    const todayDate = new Date(today.getFullYear(), today.getMonth(), today.getDate());

    if (!poDate || !expectedDelivery) {
      toast.warning('Please provide valid PO and expected delivery dates');
      return;
    }

    if (poDate.getTime() > todayDate.getTime()) {
      toast.warning('PO date cannot be in the future');
      return;
    }

    if (expectedDelivery.getTime() <= poDate.getTime()) {
      toast.warning('Expected delivery must be after PO date');
      return;
    }

    if (formData.items.length === 0) {
      toast.warning('Please add at least one line item');
      return;
    }

    for (let i = 0; i < formData.items.length; i++) {
      const item = formData.items[i];
      if (item.quantity < 1) {
        toast.warning(`Item ${i + 1}: Quantity must be at least 1`);
        return;
      }
      if (item.unit_price_foreign <= 0) {
        toast.warning(`Item ${i + 1}: Unit price must be greater than 0`);
        return;
      }
      if (!item.description || !item.description.trim()) {
        toast.warning(`Item ${i + 1}: Description is required`);
        return;
      }
    }

    if (formData.exchange_rate <= 0) {
      toast.warning('Exchange rate must be greater than 0');
      return;
    }

    formLoading = true;
    try {
      const selectedSupplier = suppliers.find((supplier) => supplier.id === formData.supplier_id);
      const poData: any = {
        ...editingPO,
        order_id: formData.order_id || undefined,
        supplier_id: formData.supplier_id,
        supplier_name: selectedSupplier?.supplier_name || editingPO?.supplier_name || '',
        po_date: toLocalDateISOString(formData.po_date),
        expected_delivery: toLocalDateISOString(formData.expected_delivery),
        currency: formData.currency,
        exchange_rate: formData.exchange_rate,
        payment_terms: formData.payment_terms,
        items: formData.items.map(item => ({
          ...item,
          total_foreign: item.quantity * item.unit_price_foreign
        }))
      };

      // Recalculate totals
      const subtotal = poData.items.reduce((sum: number, item: any) => sum + item.total_foreign, 0);
      poData.subtotal_foreign = subtotal;
      poData.subtotal_bhd = subtotal * formData.exchange_rate;
      poData.vat_amount = subtotal * 0.10;
      poData.total_foreign = subtotal + poData.vat_amount;
      poData.total_bhd = poData.total_foreign * formData.exchange_rate;

      await UpdatePurchaseOrder(poData);
      toast.success('Purchase order updated successfully');
      showEditModal = false;
      editingPO = null;
      await loadPurchaseOrders();
    } catch (err) {
      console.error('Failed to update PO:', err);
      toast.danger('Failed to update purchase order: ' + getErrorMessage(err));
    } finally {
      formLoading = false;
    }
  }

  // B5: status transition rules — mirrors purchase_order_service.go
  // UpdatePOStatus's validTransitions map EXACTLY (source of truth for legal
  // transitions; do not let this drift from it independently). Uses the
  // backend's canonical spaced strings, not the legacy unspaced POStatus
  // type above (which only drives the local filter tabs).
  const PO_APPROVAL_THRESHOLD_BHD = 5000;

  const PO_STATUS_TRANSITIONS: Record<string, string[]> = {
    'Draft': ['Pending Approval', 'Approved', 'Sent', 'Cancelled'],
    'Pending Approval': ['Approved', 'Draft', 'Cancelled'],
    'Approved': ['Sent', 'Cancelled'],
    'Sent': ['Acknowledged', 'Partially Received', 'Received', 'Cancelled'],
    'Acknowledged': ['Partially Received', 'Received', 'Cancelled'],
    'Partially Received': ['Received', 'Cancelled'],
    'Received': [],
    'Closed': [],
    'Cancelled': []
  };

  const PO_STATUS_BUTTON_LABELS: Record<string, string> = {
    'Draft': 'Revert to Draft',
    'Pending Approval': 'Submit for Approval',
    'Approved': 'Approve',
    'Sent': 'Mark Sent',
    'Acknowledged': 'Mark Acknowledged',
    'Partially Received': 'Mark Partially Received',
    'Received': 'Mark Received',
    'Cancelled': 'Cancel PO'
  };

  // Returns ONLY the transitions the backend will actually accept for this
  // PO's current status — so the UI can never offer a click that errors.
  function getAvailableTransitions(po: PODisplay): string[] {
    let legal = PO_STATUS_TRANSITIONS[po.status] || [];

    // Mirror the backend's >5000 BHD guard (purchase_order_service.go
    // UpdatePOStatus ~:663-678): Draft→Sent and Draft→Approved are blocked
    // server-side above the approval threshold — a Draft above threshold
    // may only be routed through Pending Approval (or Cancelled).
    if (po.status === 'Draft' && (po.total_bhd || 0) > PO_APPROVAL_THRESHOLD_BHD) {
      legal = legal.filter((s) => s !== 'Sent' && s !== 'Approved');
    }

    // Wave 9.7: receiving is no longer a raw status-setter click — it's
    // routed through the Receive Items panel (openReceivePanel), which
    // chains ReceiveAndCompletePO[WithSerials] so stock actually posts
    // (reconcileInventoryReceipt) before the PO status advances (server-
    // derived by updatePOStatus). Hide the 'Partially Received'/'Received'
    // status-setter buttons for POs that can still receive goods so the
    // cosmetic flip can't be triggered from here.
    if (RECEIVABLE_STATUSES.includes(po.status)) {
      legal = legal.filter((s) => s !== 'Partially Received' && s !== 'Received');
    }

    return legal;
  }

  // Handle status change
  async function handleStatusChange(poId: string, newStatus: string, currentStatus: string) {
    if (statusLoading) return;

    // Consequential/destructive transition: capture a reason via the
    // canonical confirm primitive (no native dialogs).
    if (newStatus === 'Cancelled') {
      const r = await confirm.askForReason({
        title: 'Cancel Purchase Order',
        message: 'Cancel this PO? This cannot be undone.',
        confirmLabel: 'Cancel PO',
        variant: 'danger',
        reasonLabel: 'Reason for cancellation',
        reasonRequired: true
      });
      if (!r.confirmed) return;
    }

    statusLoading = true;
    try {
      // Pending Approval → Approved routes through the dedicated approval
      // endpoint (segregation-of-duties check + approved_by/approved_at
      // tracking in purchase_order_service.go ApprovePurchaseOrder), not the
      // generic status setter.
      if (currentStatus === 'Pending Approval' && newStatus === 'Approved') {
        if (!$currentUser?.id) {
          toast.danger('Cannot approve PO: no authenticated user found. Please sign in again.');
          return;
        }
        await ApprovePurchaseOrder(poId, $currentUser.id);
      } else {
        await UpdatePOStatus(poId, newStatus);
      }
      toast.success(`PO status updated to ${newStatus}`);
      await loadPurchaseOrders();
      // Keep the open view modal in sync with the new status so the
      // transition buttons re-render for the PO's new state.
      if (viewingPO && viewingPO.id === poId) {
        const refreshed = purchaseOrders.find((p) => p.id === poId);
        if (refreshed) viewingPO = refreshed;
      }
    } catch (err) {
      console.error('Failed to update PO status:', err);
      const errorMsg = getErrorMessage(err);
      toast.danger(`Failed to update status: ${errorMsg}`);
    } finally {
      statusLoading = false;
    }
  }

  // ---------------------------------------------------------------------
  // Receive Items panel (Wave 9.7 tight-ship-2)
  // ---------------------------------------------------------------------
  // Opens a panel scoped to viewingPO's own line items — no PO picker. Each
  // line defaults its "receive now" quantity to the remaining unreceived
  // qty (ordered - already received), caps input there, and fully-received
  // lines (remaining === 0) are excluded so they can never be submitted.
  function openReceivePanel() {
    if (!viewingPO) return;

    receiveLines = (viewingPO.items || [])
      .map((item: any): ReceiveLine => {
        const ordered = Number(item.quantity || 0);
        const alreadyReceived = Number(item.quantity_received || 0);
        const remaining = Math.max(0, ordered - alreadyReceived);
        return {
          po_item_id: item.id || item.po_item_id || '',
          product_id: item.product_id || '',
          product_code: item.product_code || '',
          description: item.description || '',
          ordered,
          alreadyReceived,
          remaining,
          receiveQty: remaining,
          rejectedQty: 0,
          serialsText: '',
          requiresSerial: Boolean(item.requires_serial_tracking),
          rejectionReason: ''
        };
      })
      .filter((line) => line.remaining > 0);

    showReceiveModal = true;
  }

  // Caps a line's receive/rejected quantities in place after user input —
  // KEEP-LIST: receive qty can never exceed remaining, rejected can never
  // exceed the receive qty for that line.
  function clampReceiveLine(line: ReceiveLine) {
    if (!Number.isFinite(line.receiveQty) || line.receiveQty < 0) line.receiveQty = 0;
    if (line.receiveQty > line.remaining) line.receiveQty = line.remaining;
    if (!Number.isFinite(line.rejectedQty) || line.rejectedQty < 0) line.rejectedQty = 0;
    if (line.rejectedQty > line.receiveQty) line.rejectedQty = line.receiveQty;
  }

  // Serials are entered one-per-line (or comma separated). Wave 9.8 B1: the
  // PO item payload now carries requires_serial_tracking (query-time overlay
  // from ProductMaster — see openReceivePanel / enrichPOItemsWithSerialTracking
  // in purchase_order_service.go), so serial entry is mandatory on lines
  // where line.requiresSerial is true and the submitted count must exactly
  // match the receive quantity; on every other line it stays optional but
  // must still match if entered. The backend (ReceiveAgainstPOWithSerials)
  // is the enforcement of record either way — it re-validates
  // len(serials) === receivedQty and now also rejects an empty serials list
  // outright when the product requires serial tracking.
  function parseSerials(text: string): string[] {
    return text
      .split(/[\n,]/)
      .map((s) => s.trim())
      .filter((s) => s.length > 0);
  }

  async function handleSubmitReceive() {
    if (!viewingPO || receiveLoading) return;

    const submittable = receiveLines.filter((l) => l.receiveQty > 0);
    if (submittable.length === 0) {
      toast.warning('Enter a receive quantity for at least one line');
      return;
    }

    for (const line of submittable) {
      const serials = parseSerials(line.serialsText);
      if (line.requiresSerial && serials.length !== line.receiveQty) {
        toast.danger(
          `${line.product_code || line.description}: this product requires serial tracking — enter exactly ${line.receiveQty} serial(s) (got ${serials.length})`
        );
        return;
      }
      if (!line.requiresSerial && serials.length > 0 && serials.length !== line.receiveQty) {
        toast.danger(
          `${line.product_code || line.description}: entered ${serials.length} serial(s) but receiving ${line.receiveQty} — counts must match`
        );
        return;
      }
      if (line.rejectedQty > 0 && !line.rejectionReason.trim()) {
        toast.danger(
          `${line.product_code || line.description}: a reason is required to record ${line.rejectedQty} rejected unit(s) as a discrepancy`
        );
        return;
      }
    }

    // Wave 9.8 B1: route through the serial-aware endpoint whenever ANY line
    // requires serial tracking (not just when serials happen to be entered),
    // so ReceiveAgainstPOWithSerials's len(serials)===qty check becomes the
    // backend enforcement of record for that line.
    const anySerials = submittable.some((l) => l.requiresSerial || parseSerials(l.serialsText).length > 0);

    receiveLoading = true;
    try {
      let grnId = '';
      let grnItems: any[] = [];

      if (anySerials) {
        const items = submittable.map((l) => ({
          po_item_id: l.po_item_id,
          product_id: l.product_id,
          quantity_ordered: l.ordered,
          quantity_received: l.receiveQty,
          quantity_rejected: l.rejectedQty,
          serial_numbers: parseSerials(l.serialsText)
        }));
        const resp = await ReceiveAndCompletePOWithSerials(viewingPO.id, items as any);
        grnId = (resp as any)?.id || '';
        grnItems = (resp as any)?.items || [];
      } else {
        const items = submittable.map((l) => ({
          po_item_id: l.po_item_id,
          product_id: l.product_id,
          quantity_ordered: l.ordered,
          quantity_received: l.receiveQty,
          quantity_rejected: l.rejectedQty
        }));
        const resp = await ReceiveAndCompletePO(viewingPO.id, items as any);
        grnId = (resp as any)?.id || '';
        grnItems = (resp as any)?.items || [];
      }

      // Wave 9.8 B1: for every short-shipped/rejected line, raise a GRN
      // discrepancy against the just-created GRN — records through the
      // existing SupplierIssue side-effect of RaiseGRNDiscrepancy (surfaced
      // on the Supplier detail screen's Issues tab), no new table/screen.
      const rejectedLines = submittable.filter((l) => l.rejectedQty > 0);
      if (rejectedLines.length > 0) {
        if (!grnId) {
          console.error('Receive completed but no GRN id was returned — cannot raise discrepancies', { grnId, grnItems });
          toast.warning('Items received, but discrepancies could not be recorded (missing GRN reference)');
        } else {
          for (const line of rejectedLines) {
            const grnItem = grnItems.find((gi: any) => gi.po_item_id === line.po_item_id);
            if (!grnItem?.id) {
              console.error('No matching GRN item found for discrepancy', { line, grnItems });
              continue;
            }
            try {
              await RaiseGRNDiscrepancy(grnId, grnItem.id, line.rejectionReason.trim(), 'quantity_short', line.rejectedQty);
            } catch (discErr) {
              console.error('Failed to raise GRN discrepancy:', discErr);
              toast.warning(`Received, but failed to record discrepancy for ${line.product_code || line.description}`);
            }
          }
        }
      }

      toast.success('Items received and posted to stock');
      showReceiveModal = false;
      showViewModal = false;
      await loadPurchaseOrders();
    } catch (err) {
      console.error('Failed to receive PO items:', err);
      toast.danger(`Failed to receive items: ${getErrorMessage(err)}`);
    } finally {
      receiveLoading = false;
    }
  }

  // Handle PDF generation
  async function handleGeneratePDF(poId: string) {
    if (pdfLoading) return;

    pdfLoading = true;
    try {
      toast.info('Generating PDF...');
      const filePath = await GeneratePurchaseOrderPDF(poId);

      // Show success with file path
      toast.success(`PDF generated successfully! Saved to: ${filePath}`);

      // Optional: Open the exports folder in file explorer
      // This requires a backend function to open the folder
      console.log(`PO PDF saved: ${filePath}`);
    } catch (err) {
      console.error('Failed to generate PDF:', err);
      const errorMsg = getErrorMessage(err);
      toast.danger(`Failed to generate PDF: ${errorMsg}`);
    } finally {
      pdfLoading = false;
    }
  }

  // Handle action button clicks (delegated from DataTable)
  function handleRowClick(event: CustomEvent) {
    const target = event.detail.event?.target as HTMLElement;
    if (!target || !target.dataset.action) return;

    const action = target.dataset.action;
    const id = target.dataset.id;

    switch (action) {
      case 'view':
        openViewModal(id!);
        break;
      case 'edit':
        openEditModal(id!);
        break;
      case 'pdf':
        if (!pdfLoading) {
          handleGeneratePDF(id!);
        }
        break;
    }
  }

  // Supplier selection handler
  function handleSupplierSelect(e: Event) {
    const select = e.target as HTMLSelectElement;
    formData.supplier_id = select.value;
  }

  // Order selection handler
  function handleOrderSelect(e: Event) {
    const select = e.target as HTMLSelectElement;
    formData.order_id = select.value;
  }

  // Currency change handler - update exchange rate
  function handleCurrencyChange(e: Event) {
    const select = e.target as HTMLSelectElement;
    formData.currency = select.value;

    // BHD per 1 unit of foreign currency.
    const rates: Record<string, number> = {
      'BHD': 1.0,
      'USD': 0.376,
      'EUR': 0.45,
      'GBP': 0.52,
      'CHF': 0.425,
      'AED': 0.102,
      'SAR': 0.100
    };

    formData.exchange_rate = rates[formData.currency] || 1.0;
  }

  // Add line item
  function addLineItem() {
    formData.items = [...formData.items, {
      product_id: '',
      description: '',
      quantity: 1,
      unit_price_foreign: 0
    }];
  }

  // Remove line item
  function removeLineItem(index: number) {
    formData.items = formData.items.filter((_, i) => i !== index);
  }

  // Computed: Total value
  let totalForeign = $derived(formData.items.reduce((sum, item) => sum + (item.quantity * item.unit_price_foreign), 0));
  let vatAmount = $derived(totalForeign * 0.10);
  let totalWithVAT = $derived(totalForeign + vatAmount);
  let totalBHD = $derived(totalWithVAT * formData.exchange_rate);

  // Event handler for opening create modal from parent hub
  function handleOpenCreatePO() {
    openCreateModal();
  }

  onMount(() => {
    if (!canView) return;
    loadPurchaseOrders();

    // Listen for create PO events from OperationsHub header button
    window.addEventListener('openCreatePO', handleOpenCreatePO);

    // B7: land inside a PO created elsewhere in the app (Orders → Supplier Order handoff)
    const pendingOpenPO = sessionStorage.getItem('asymmflow.pendingOpenPO');
    if (pendingOpenPO) {
      try {
        const pending = JSON.parse(pendingOpenPO);
        if (pending?.id) {
          openViewModal(pending.id);
        }
      } catch (err) {
        // ignore malformed pending payload
      } finally {
        sessionStorage.removeItem('asymmflow.pendingOpenPO');
      }
    }

    // B3: open the create form preseeded with a supplier chosen elsewhere in the app
    const pendingSupplier = sessionStorage.getItem('asymmflow.pendingPOSupplier');
    if (pendingSupplier) {
      try {
        const pending = JSON.parse(pendingSupplier);
        if (pending?.id) {
          openCreateModal();
          formData.supplier_id = pending.id;
        }
      } catch (err) {
        // ignore malformed pending payload
      } finally {
        sessionStorage.removeItem('asymmflow.pendingPOSupplier');
      }
    }
  });

  onDestroy(() => {
    // Clean up event listener
    window.removeEventListener('openCreatePO', handleOpenCreatePO);
  });
</script>

<PageLayout title="Purchase Orders" subtitle="Procurement Management" {embedded}>
  <!-- @migration-task: migrate this slot by hand, `header-actions` is an invalid identifier -->
  <svelte:fragment slot="header-actions">
    <Button variant="primary" on:click={openCreateModal}>
      + New PO
    </Button>
  </svelte:fragment>

  <div class="po-container">
    <!-- Status Filter Tabs -->
    <Card padding="sm">
      <div class="status-tabs" role="tablist" aria-label="Filter POs by status">
        {#each statusTabs as tab}
          <button
            class="status-tab"
            class:active={selectedStatus === tab.value}
            role="tab"
            aria-selected={selectedStatus === tab.value}
            onclick={() => selectedStatus = tab.value}
          >
            {tab.label}
            <span class="tab-count">{tab.count}</span>
          </button>
        {/each}
      </div>
    </Card>

    <!-- Purchase Orders DataTable -->
    <Card padding="sm">
      <DataTable
        {columns}
        data={filteredPOs}
        {loading}
        emptyMessage="No purchase orders yet — raise one to order from a supplier."
        onRowClick={(row) => {}}
        on:rowClick={handleRowClick}
        stickyHeader={true}
        maxHeight="calc(100vh - 300px)"
        showBorder={false}
      />
    </Card>

    <!-- Summary Stats -->
    <div class="stats-grid">
      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Total POs</div>
          <div class="stat-value">{purchaseOrders.length}</div>
        </div>
      </Card>

      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Total Value (BHD)</div>
          <div class="stat-value">
            {formatBHD(purchaseOrders.reduce((sum, p) => sum + (p.total_bhd || 0), 0))}
          </div>
        </div>
      </Card>

      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Pending Receipts</div>
          <div class="stat-value stat-warning">
            {purchaseOrders.filter(p => ['Sent', 'Acknowledged', 'Partially Received'].includes(p.status)).length}
          </div>
        </div>
      </Card>

      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Fully Received</div>
          <div class="stat-value stat-success">
            {purchaseOrders.filter(p => p.status === 'Received').length}
          </div>
        </div>
      </Card>
    </div>
  </div>
</PageLayout>

<!-- Create PO Modal -->
<WabiModal bind:open={showCreateModal} title="Create Purchase Order" size="xl">
  <form onsubmit={preventDefault(handleCreatePO)} class="po-form">
    <!-- Supplier and Order Selection -->
    <div class="form-row">
      <FormGroup label="Supplier" required>
        <select
          class="select-input"
          bind:value={formData.supplier_id}
          onchange={handleSupplierSelect}
          required
        >
          <option value="">Select supplier...</option>
          {#each suppliers as supplier}
            <option value={supplier.id}>{supplier.supplier_name}</option>
          {/each}
        </select>
      </FormGroup>

      <FormGroup label="Link to Order (Optional)">
        <select
          class="select-input"
          bind:value={formData.order_id}
          onchange={handleOrderSelect}
        >
          <option value="">No order link...</option>
          {#each orders as order}
            <option value={order.id}>{order.order_number} - {order.customer_name}</option>
          {/each}
        </select>
      </FormGroup>
    </div>

    <!-- Dates -->
    <div class="form-row">
      <FormGroup label="PO Date" required>
        <Input
          type="date"
          bind:value={formData.po_date}
          max={formatDateInput(new Date())}
          required
        />
      </FormGroup>

      <FormGroup label="Expected Delivery" required>
        <Input
          type="date"
          bind:value={formData.expected_delivery}
          min={formData.po_date}
          required
        />
      </FormGroup>
    </div>

    <!-- Currency and Payment -->
    <div class="form-row">
      <FormGroup label="Currency" required>
        <select
          class="select-input"
          bind:value={formData.currency}
          onchange={handleCurrencyChange}
          required
        >
          {#each currencies as curr}
            <option value={curr.value}>{curr.label}</option>
          {/each}
        </select>
      </FormGroup>

      <FormGroup label="Exchange Rate (to BHD)" required>
        <Input
          type="number"
          bind:value={formData.exchange_rate}
          step="0.0001"
          min="0.0001"
          required
        />
        <div class="exchange-hint">1 {formData.currency} = {formData.exchange_rate.toFixed(4)} BHD</div>
      </FormGroup>
    </div>

    <FormGroup label="Payment Terms" required>
      <select
        class="select-input"
        bind:value={formData.payment_terms}
        required
      >
        {#each paymentTerms as term}
          <option value={term}>{term}</option>
        {/each}
      </select>
    </FormGroup>

    <!-- Line Items -->
    <div class="line-items-section">
      <div class="section-header">
        <h4>Line Items</h4>
        <Button variant="secondary" size="sm" on:click={addLineItem}>
          + Add Item
        </Button>
      </div>

      {#if formData.items.length === 0}
        <div class="empty-items">
          No items added yet. Click "Add Item" to start.
        </div>
      {:else}
        <div class="items-list">
          {#each formData.items as item, index}
            <div class="item-row" transition:fade={{ duration: motionMs(400) }}>
              <Input
                label="Description"
                bind:value={item.description}
                placeholder="Product or service description"
              />
              <Input
                label="Quantity"
                type="number"
                bind:value={item.quantity}
                min="1"
                step="1"
                required
              />
              <Input
                label="Unit Price ({formData.currency})"
                type="number"
                bind:value={item.unit_price_foreign}
                min="0.001"
                step="0.001"
                required
              />
              <div class="item-total">
                <div class="label">Total</div>
                <div class="value">{formatNumber(item.quantity * item.unit_price_foreign, 3)} {formData.currency}</div>
              </div>
              <button
                type="button"
                class="btn-remove"
                onclick={() => removeLineItem(index)}
                aria-label="Remove item"
              >
                ×
              </button>
            </div>
          {/each}
        </div>

        <!-- Totals Summary -->
        <div class="totals-section">
          <div class="total-row">
            <div class="total-label">Subtotal:</div>
            <div class="total-value">{formatNumber(totalForeign, 3)} {formData.currency}</div>
          </div>
          <div class="total-row">
            <div class="total-label">VAT (10%):</div>
            <div class="total-value">{formatNumber(vatAmount, 3)} {formData.currency}</div>
          </div>
          <div class="total-row total-grand">
            <div class="total-label">Total ({formData.currency}):</div>
            <div class="total-value">{formatNumber(totalWithVAT, 3)} {formData.currency}</div>
          </div>
          {#if formData.currency !== 'BHD'}
            <div class="total-row total-bhd">
              <div class="total-label">Total (BHD):</div>
              <div class="total-value">{formatBHD(totalBHD)}</div>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </form>

  {#snippet footer()}
  
      <Button variant="ghost" on:click={() => showCreateModal = false}>
        Cancel
      </Button>
      <Button
        variant="primary"
        loading={formLoading}
        on:click={handleCreatePO}
      >
        Create PO
      </Button>
    
  {/snippet}
</WabiModal>

<!-- Edit PO Modal (same as create) -->
<WabiModal bind:open={showEditModal} title="Edit Purchase Order" size="xl">
  <form onsubmit={preventDefault(handleEditPO)} class="po-form">
    <!-- Same form fields as create modal -->
    <div class="form-row">
      <FormGroup label="Supplier" required>
        <select
          class="select-input"
          bind:value={formData.supplier_id}
          onchange={handleSupplierSelect}
          required
        >
          <option value="">Select supplier...</option>
          {#each suppliers as supplier}
            <option value={supplier.id}>{supplier.supplier_name}</option>
          {/each}
        </select>
      </FormGroup>

      <FormGroup label="Link to Order (Optional)">
        <select
          class="select-input"
          bind:value={formData.order_id}
          onchange={handleOrderSelect}
        >
          <option value="">No order link...</option>
          {#each orders as order}
            <option value={order.id}>{order.order_number} - {order.customer_name}</option>
          {/each}
        </select>
      </FormGroup>
    </div>

    <div class="form-row">
      <FormGroup label="PO Date" required>
        <Input
          type="date"
          bind:value={formData.po_date}
          max={formatDateInput(new Date())}
          required
        />
      </FormGroup>

      <FormGroup label="Expected Delivery" required>
        <Input
          type="date"
          bind:value={formData.expected_delivery}
          min={formData.po_date}
          required
        />
      </FormGroup>
    </div>

    <div class="form-row">
      <FormGroup label="Currency" required>
        <select
          class="select-input"
          bind:value={formData.currency}
          onchange={handleCurrencyChange}
          required
        >
          {#each currencies as curr}
            <option value={curr.value}>{curr.label}</option>
          {/each}
        </select>
      </FormGroup>

      <FormGroup label="Exchange Rate (to BHD)" required>
        <Input
          type="number"
          bind:value={formData.exchange_rate}
          step="0.0001"
          min="0.0001"
          required
        />
        <div class="exchange-hint">1 {formData.currency} = {formData.exchange_rate.toFixed(4)} BHD</div>
      </FormGroup>
    </div>

    <FormGroup label="Payment Terms" required>
      <select
        class="select-input"
        bind:value={formData.payment_terms}
        required
      >
        {#each paymentTerms as term}
          <option value={term}>{term}</option>
        {/each}
      </select>
    </FormGroup>

    <div class="line-items-section">
      <div class="section-header">
        <h4>Line Items</h4>
        <Button variant="secondary" size="sm" on:click={addLineItem}>
          + Add Item
        </Button>
      </div>

      {#if formData.items.length === 0}
        <div class="empty-items">
          No items added yet. Click "Add Item" to start.
        </div>
      {:else}
        <div class="items-list">
          {#each formData.items as item, index}
            <div class="item-row" transition:fade={{ duration: motionMs(400) }}>
              <Input
                label="Description"
                bind:value={item.description}
                placeholder="Product or service description"
              />
              <Input
                label="Quantity"
                type="number"
                bind:value={item.quantity}
                min="1"
                step="1"
                required
              />
              <Input
                label="Unit Price ({formData.currency})"
                type="number"
                bind:value={item.unit_price_foreign}
                min="0.001"
                step="0.001"
                required
              />
              <div class="item-total">
                <div class="label">Total</div>
                <div class="value">{formatNumber(item.quantity * item.unit_price_foreign, 3)} {formData.currency}</div>
              </div>
              <button
                type="button"
                class="btn-remove"
                onclick={() => removeLineItem(index)}
                aria-label="Remove item"
              >
                ×
              </button>
            </div>
          {/each}
        </div>

        <div class="totals-section">
          <div class="total-row">
            <div class="total-label">Subtotal:</div>
            <div class="total-value">{formatNumber(totalForeign, 3)} {formData.currency}</div>
          </div>
          <div class="total-row">
            <div class="total-label">VAT (10%):</div>
            <div class="total-value">{formatNumber(vatAmount, 3)} {formData.currency}</div>
          </div>
          <div class="total-row total-grand">
            <div class="total-label">Total ({formData.currency}):</div>
            <div class="total-value">{formatNumber(totalWithVAT, 3)} {formData.currency}</div>
          </div>
          {#if formData.currency !== 'BHD'}
            <div class="total-row total-bhd">
              <div class="total-label">Total (BHD):</div>
              <div class="total-value">{formatBHD(totalBHD)}</div>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </form>

  {#snippet footer()}
  
      <Button variant="ghost" on:click={() => showEditModal = false}>
        Cancel
      </Button>
      <Button
        variant="primary"
        loading={formLoading}
        on:click={handleEditPO}
      >
        Save Changes
      </Button>
    
  {/snippet}
</WabiModal>

<!-- View PO Modal -->
{#if viewingPO}
  <WabiModal bind:open={showViewModal} title="Purchase Order Details" size="md">
    <div class="po-details">
      <!-- Header Info -->
      <div class="details-grid">
        <div class="detail-item">
          <div class="detail-label">PO Number</div>
          <div class="detail-value mono">{viewingPO.po_number}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Status</div>
          <StatusBadge status={viewingPO.status} />
        </div>
        <div class="detail-item">
          <div class="detail-label">Supplier</div>
          <div class="detail-value">{viewingPO.supplier_name}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">PO Date</div>
          <div class="detail-value">{new Date(viewingPO.po_date).toLocaleDateString()}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Expected Delivery</div>
          <div class="detail-value">{new Date(viewingPO.expected_delivery).toLocaleDateString()}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Payment Terms</div>
          <div class="detail-value">{viewingPO.payment_terms}</div>
        </div>
      </div>

      <!-- Items Table -->
      {#if viewingPO.items && viewingPO.items.length > 0}
        <div class="items-section">
          <h4>Items</h4>
          <table class="items-table">
            <thead>
              <tr>
                <th>Description</th>
                <th class="right">Quantity</th>
                <th class="right">Unit Price ({viewingPO.currency})</th>
                <th class="right">Total ({viewingPO.currency})</th>
              </tr>
            </thead>
            <tbody>
              {#each viewingPO.items as item}
                <tr>
                  <td>{item.description}</td>
                  <td class="right mono">{item.quantity}</td>
                  <td class="right mono">{formatNumber(item.unit_price_foreign, 3)}</td>
                  <td class="right mono">{formatNumber(item.total_foreign, 3)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      <!-- Totals -->
      <div class="view-totals">
        <div class="total-row">
          <div class="label">Net Amount:</div>
          <div class="value mono">{formatNumber(viewingPO.subtotal_foreign, 3)} {viewingPO.currency}</div>
        </div>
        <div class="total-row">
          <div class="label">VAT:</div>
          <div class="value mono">{formatNumber(viewingPO.vat_amount, 3)} {viewingPO.currency}</div>
        </div>
        <div class="total-row grand">
          <div class="label">Total incl. VAT ({viewingPO.currency}):</div>
          <div class="value mono">{formatNumber(viewingPO.total_foreign, 3)} {viewingPO.currency}</div>
        </div>
        {#if viewingPO.currency !== 'BHD'}
          <div class="total-row bhd">
            <div class="label">Total (BHD):</div>
            <div class="value mono">{formatBHD(viewingPO.total_bhd)}</div>
          </div>
        {/if}
      </div>

      <!-- Status Actions: only the legal next transition(s) for the PO's
           CURRENT status are ever rendered (mirrors validTransitions in
           purchase_order_service.go) — terminal statuses show none.
           Receiving (Wave 9.7) is its own action: it opens the Receive
           Items panel instead of firing the raw status setter, so goods
           actually post to stock before the PO status advances. -->
      {#if getAvailableTransitions(viewingPO).length > 0 || RECEIVABLE_STATUSES.includes(viewingPO.status)}
        <div class="status-actions">
          <h4>Update Status</h4>
          <div class="status-buttons">
            {#if RECEIVABLE_STATUSES.includes(viewingPO.status)}
              <Button
                variant="primary"
                size="sm"
                disabled={statusLoading}
                on:click={openReceivePanel}
              >
                Receive Items
              </Button>
            {/if}
            {#each getAvailableTransitions(viewingPO) as status}
              <Button
                variant={status === 'Cancelled' ? 'danger' : 'primary'}
                size="sm"
                disabled={statusLoading}
                on:click={() => handleStatusChange(viewingPO.id, status, viewingPO.status)}
              >
                {PO_STATUS_BUTTON_LABELS[status] || status}
              </Button>
            {/each}
          </div>
        </div>
      {/if}
    </div>

    {#snippet footer()}
      
        <Button
          variant="primary"
          loading={pdfLoading}
          on:click={() => handleGeneratePDF(viewingPO.id)}
        >
          Download PDF
        </Button>
        <Button variant="secondary" on:click={() => openEditModal(viewingPO.id)}>
          Edit PO
        </Button>
        <Button variant="ghost" on:click={() => showViewModal = false}>
          Close
        </Button>
      
      {/snippet}
  </WabiModal>
{/if}

{#if showReceiveModal && viewingPO}
  <WabiModal bind:open={showReceiveModal} title={`Receive Items — ${viewingPO.po_number}`} size="md">
    <div class="receive-panel">
      {#if receiveLines.length === 0}
        <p>All items on this PO have already been fully received.</p>
      {:else}
        <p class="receive-hint">
          Each line defaults to its remaining unreceived quantity. Serial numbers are required
          (one per unit, one per line or comma-separated) on lines whose product requires serial
          tracking — the count must exactly match the receive quantity. On other lines serials are
          optional but must still match if entered. Rejecting any quantity requires a reason and
          raises a supplier discrepancy once the receive posts.
        </p>
        <table class="items-table receive-table">
          <thead>
            <tr>
              <th>Product</th>
              <th class="right">Ordered</th>
              <th class="right">Received</th>
              <th class="right">Remaining</th>
              <th class="right">Receive now</th>
              <th class="right">Rejected</th>
              <th>Serials</th>
              <th>Rejection reason</th>
            </tr>
          </thead>
          <tbody>
            {#each receiveLines as line}
              <tr>
                <td>
                  <div class="mono">{line.product_code}</div>
                  <div class="muted">{line.description}</div>
                  {#if line.requiresSerial}
                    <div class="serial-required-badge">Serial required</div>
                  {/if}
                </td>
                <td class="right mono">{formatNumber(line.ordered, 3)}</td>
                <td class="right mono">{formatNumber(line.alreadyReceived, 3)}</td>
                <td class="right mono">{formatNumber(line.remaining, 3)}</td>
                <td class="right">
                  <input
                    class="receive-qty-input"
                    type="number"
                    min="0"
                    max={line.remaining}
                    step="any"
                    bind:value={line.receiveQty}
                    oninput={() => clampReceiveLine(line)}
                  />
                </td>
                <td class="right">
                  <input
                    class="receive-qty-input"
                    type="number"
                    min="0"
                    max={line.receiveQty}
                    step="any"
                    bind:value={line.rejectedQty}
                    oninput={() => clampReceiveLine(line)}
                  />
                </td>
                <td>
                  <textarea
                    class="serials-input"
                    rows="2"
                    placeholder={line.requiresSerial ? 'One serial per unit (required)' : 'One serial per unit (optional)'}
                    bind:value={line.serialsText}
                  ></textarea>
                </td>
                <td>
                  {#if line.rejectedQty > 0}
                    <textarea
                      class="serials-input"
                      rows="2"
                      placeholder="Reason for rejection (required)"
                      bind:value={line.rejectionReason}
                    ></textarea>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>

    {#snippet footer()}

        <Button
          variant="primary"
          loading={receiveLoading}
          disabled={receiveLoading || receiveLines.length === 0}
          on:click={handleSubmitReceive}
        >
          Receive
        </Button>
        <Button variant="ghost" disabled={receiveLoading} on:click={() => showReceiveModal = false}>
          Cancel
        </Button>

      {/snippet}
  </WabiModal>
{/if}

<style>
  .po-container {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  /* Status Tabs */
  .status-tabs {
    display: flex;
    gap: 8px;
    overflow-x: auto;
  }

  .status-tab {
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

  .status-tab:hover {
    background: var(--interactive-hover);
    color: var(--text-primary);
  }

  .status-tab.active {
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

  .status-tab.active .tab-count {
    background: rgba(255, 255, 255, 0.2);
  }

  /* Stats Grid */
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
  }

  .stat {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .stat-label {
    font-size: var(--label-size);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .stat-value {
    font-size: 24px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .stat-warning {
    color: #F59E0B;
  }

  .stat-success {
    color: #10B981;
  }

  /* Action Buttons in Table */
  :global(.action-btn) {
    padding: 4px 10px;
    font-size: 12px;
    font-weight: 500;
    border: none;
    border-radius: var(--border-radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  :global(.action-btn-view) {
    background: var(--surface-elevated);
    color: var(--text-primary);
  }

  :global(.action-btn-view:hover) {
    background: var(--interactive-hover);
  }

  :global(.action-btn-edit) {
    background: var(--brand-indigo-tint);
    color: var(--brand-indigo);
  }

  :global(.action-btn-edit:hover) {
    background: var(--brand-indigo);
    color: white;
  }

  :global(.action-btn-pdf) {
    background: rgba(245, 158, 11, 0.1);
    color: #F59E0B;
  }

  :global(.action-btn-pdf:hover:not([disabled])) {
    background: #F59E0B;
    color: white;
  }

  :global(.action-btn-pdf[disabled]) {
    opacity: 0.5;
    cursor: not-allowed;
    pointer-events: none;
  }

  /* Form Styles */
  .po-form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .form-row {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
  }

  .select-input {
    width: 100%;
    padding: 8px 12px;
    font-size: 14px;
    font-family: var(--font-family);
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
  }

  .select-input:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 3px var(--brand-indigo-tint);
  }

  /* Line Items Section */
  .line-items-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
    background: var(--surface-elevated);
    border-radius: var(--border-radius);
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .section-header h4 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .empty-items {
    padding: 24px;
    text-align: center;
    color: var(--text-muted);
    font-style: italic;
    font-size: 14px;
  }

  .items-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .item-row {
    display: grid;
    grid-template-columns: 2fr 1fr 1fr 1fr 32px;
    gap: 12px;
    align-items: end;
    padding: 12px;
    background: var(--surface);
    border-radius: var(--border-radius-sm);
    border: 1px solid var(--border);
  }

  .item-total {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .item-total .label {
    font-size: var(--label-size);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .item-total .value {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .btn-remove {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    border-radius: 50%;
    font-size: 24px;
    color: var(--text-muted);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .btn-remove:hover {
    background: rgba(220, 38, 38, 0.1);
    color: #DC2626;
  }

  /* Totals Section */
  .totals-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px 16px;
    background: var(--surface);
    border-radius: var(--border-radius-sm);
    border: 1px solid var(--border);
  }

  .total-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 4px 0;
  }

  .total-label {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .total-value {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .total-grand {
    padding-top: 8px;
    border-top: 2px solid var(--border);
  }

  .total-grand .total-label,
  .total-grand .total-value {
    font-size: 16px;
    font-weight: 700;
    color: var(--brand-indigo);
  }

  .total-bhd {
    padding-top: 8px;
    border-top: 1px dashed var(--border);
  }

  .total-bhd .total-value {
    color: #10B981;
  }

  /* View Modal Styles */
  .po-details {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }

  .details-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px;
  }

  .detail-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .detail-label {
    font-size: var(--label-size);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .detail-value {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary);
  }

  .mono {
    font-family: var(--font-mono);
  }

  .items-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-width: 0;
  }

  .items-section h4 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .items-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
    table-layout: fixed;
  }

  .items-table th {
    text-align: left;
    padding: 8px 12px;
    background: var(--surface-elevated);
    color: var(--text-secondary);
    font-weight: 600;
    border-bottom: 2px solid var(--border);
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .items-table td {
    padding: 8px 12px;
    border-bottom: 1px solid var(--border);
    color: var(--text-primary);
    overflow-wrap: anywhere;
  }

  .items-table .right {
    text-align: right;
  }

  .receive-panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-width: 0;
  }

  .receive-hint {
    margin: 0;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .receive-table th,
  .receive-table td {
    vertical-align: top;
  }

  .muted {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .receive-qty-input {
    width: 72px;
    padding: 4px 6px;
    font-family: var(--font-mono);
    font-size: 13px;
    text-align: right;
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
    background: var(--surface-base);
    color: var(--text-primary);
  }

  .serials-input {
    width: 100%;
    min-width: 140px;
    padding: 4px 6px;
    font-family: var(--font-mono);
    font-size: 12px;
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
    background: var(--surface-base);
    color: var(--text-primary);
    resize: vertical;
  }

  .serial-required-badge {
    margin-top: 4px;
    display: inline-block;
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.02em;
    padding: 2px 6px;
    border-radius: var(--border-radius-sm);
    background: var(--warning-bg, rgba(217, 119, 6, 0.15));
    color: var(--warning-text, #b45309);
  }

  .view-totals {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 14px;
    background: var(--surface-elevated);
    border-radius: var(--border-radius-sm);
  }

  .status-actions {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .status-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  @media (max-width: 900px) {
    .details-grid {
      grid-template-columns: 1fr;
    }
  }

  .view-totals .total-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .view-totals .label {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .view-totals .value {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .view-totals .grand {
    padding-top: 8px;
    border-top: 2px solid var(--border);
  }

  .view-totals .grand .label,
  .view-totals .grand .value {
    font-size: 16px;
    font-weight: 700;
    color: var(--brand-indigo);
  }

  .view-totals .bhd {
    padding-top: 8px;
    border-top: 1px dashed var(--border);
  }

  .view-totals .bhd .value {
    color: #10B981;
  }

  .status-actions {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .status-actions h4 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .status-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  /* Exchange hint */
  .exchange-hint {
    font-size: 11px;
    color: var(--text-secondary);
    margin-top: 4px;
    font-family: var(--font-mono);
  }

  /* Responsive */
  @media (max-width: 768px) {
    .form-row {
      grid-template-columns: 1fr;
    }

    .item-row {
      grid-template-columns: 1fr;
      gap: 8px;
    }

    .stats-grid {
      grid-template-columns: 1fr;
    }

    .details-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
