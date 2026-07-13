<script lang="ts">
  import { run, preventDefault } from 'svelte/legacy';

  import { onMount } from 'svelte';
  import {
    ListOrders, PreviewOrderDeleteCascade } from '../../../wailsjs/go/main/App';
import { GetOrder, CreateOrderWithItems, UpdateOrder, DeleteOrder, UpdateOrderStage, ListCustomers, GetOrderDeliveryStatusBatch, GetOrderFulfillmentStatus, CreatePOFromOrder, QuickMarkOrderDelivered, GetOrdersWithNoItems } from '../../../wailsjs/go/main/CRMService';
import { CreateInvoiceFromOrder, CreateProformaInvoice } from '../../../wailsjs/go/main/FinanceService';
  import { toast } from '$lib/stores/toasts';
  import { pendingDNCreate, pendingProjectHandoff, pendingOrderView } from '$lib/stores/navigation';
  import { main, crm } from '../../../wailsjs/go/models';
  import { escapeHtml } from '$lib/utils/escapeHtml';
  import PageLayout from '$lib/components/layout/PageLayout.svelte';
  import DataTable from '$lib/components/ui/DataTable.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import LineItemsEditor from '$lib/components/ui/LineItemsEditor.svelte';
  import StatusBadge from '$lib/components/ui/StatusBadge.svelte';
  import { GetInvoicesByOrder } from '../../../wailsjs/go/main/FinanceService';
  import Modal from '$lib/components/layout/Modal.svelte';
  import FormGroup from '$lib/components/ui/FormGroup.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import ContextTaskModal from '$lib/components/ContextTaskModal.svelte';

  
  interface Props {
    // Props
    embedded?: boolean;
  }

  let { embedded = false }: Props = $props();

  // Status definitions matching backend statuses
  const ORDER_STATUSES = [
    { key: 'Confirmed', label: 'Confirmed', color: 'blue' },
    { key: 'InProgress', label: 'In Progress', color: 'indigo' },
    { key: 'PartiallyDelivered', label: 'Partially Delivered', color: 'amber' },
    { key: 'FullyDelivered', label: 'Fully Delivered', color: 'green' },
    { key: 'Invoiced', label: 'Invoiced', color: 'emerald' },
    { key: 'Cancelled', label: 'Cancelled', color: 'red' }
  ];

  function errorMessage(err: unknown) {
    if (err instanceof Error && err.message) return err.message;
    if (typeof err === 'string') return err;
    if (err && typeof err === 'object' && 'message' in err) {
      return String((err as { message?: unknown }).message || err);
    }
    return String(err || 'Unknown error');
  }

  // State
  let orders: crm.Order[] = $state([]);
  let filteredOrders: crm.Order[] = $state([]);
  let loading = $state(true);
  let activeFilter = $state('All');
  let searchQuery = $state('');
  let selectedYear = $state('All');
  let selectedCustomer = $state('All');
  let availableYears: string[] = $state([]);
  let availableCustomers: string[] = $state([]);
  let ordersWithNoItems: any[] = $state([]);

  // B10-2: pagination — replaces the old flat ListOrders(10000, 0). Client-
  // side filters/search below still operate on the loaded set only; year and
  // customer dropdown options are similarly built from what's loaded so far.
  const PAGE_SIZE = 100;
  let currentPage = 0;
  let hasMoreOrders = $state(true);
  let loadingMoreOrders = $state(false);
  let totalOrdersLoaded = $state(0);

  // Create/Edit modal
  let showModal = $state(false);
  let editingOrder: crm.Order | null = null;
  let modalMode: 'create' | 'edit' = $state('create');
  let saving = $state(false);
  let isDeleting = $state(false);
  let isProcessingAction = $state(false);

  // Form state
  let formData = $state({
    orderNumber: '',
    customerPONumber: '',
    customerId: '',
    customerName: '',
    orderDate: new Date().toISOString().split('T')[0],
    requiredDate: '',
    totalValue: 0,
    status: 'Confirmed',
    paymentTerms: 'Net 30',
    deliveryTerms: 'Ex-Works'
  });
  interface EditableOrderItem {
    id?: string;
    product_id?: string;
    product_code: string;
    description: string;
    quantity: number;
    unit_price_bhd: number;
    total_price: number;
    quantity_shipped?: number;
    quantity_invoiced?: number;
    equipment?: string;
    model?: string;
    specification?: string;
    detailed_description?: string;
    currency?: string;
    fob?: number;
    freight?: number;
    total_cost?: number;
    margin_percent?: number;
    line_number?: number;
  }
  let formItems: EditableOrderItem[] = $state([]);

  // Helper data
  let customers: any[] = $state([]);

  function normalizeCustomerName(value: string) {
    return (value || '').trim().toLowerCase().replace(/\s+/g, ' ');
  }

  function resolveCustomerId(customerId: string, customerName: string) {
    if (customerId && customers.some((customer) => customer.id === customerId)) {
      return customerId;
    }
    const normalizedName = normalizeCustomerName(customerName);
    if (!normalizedName) return '';
    const match = customers.find((customer) => normalizeCustomerName(customer.business_name || '') === normalizedName);
    return match?.id || '';
  }

  // Detail modal
  let showDetailModal = $state(false);
  let selectedOrder: crm.Order | null = $state(null);
  let showTaskModal = $state(false);
  let orderItems: EditableOrderItem[] = $state([]);
  let deliveryProgress = $state({ delivered: 0, total: 0, percentage: 0 });
  let fulfillmentStatus: any = $state(null);
  // B3: invoices generated from this order — loaded lazily with the detail modal
  let linkedInvoices: any[] = $state([]);
  let loadingInvoices = $state(false);
  type ConfirmVariant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'success' | 'warning';
  let confirmDialogOpen = $state(false);
  let confirmDialogTitle = $state('');
  let confirmDialogMessage = $state('');
  let confirmDialogConfirmLabel = $state('Confirm');
  let confirmDialogVariant: ConfirmVariant = $state('primary');
  let confirmDialogResolve: ((confirmed: boolean) => void) | null = null;
  // C3: cascade-preview lines shown under the confirm message, and a blocked
  // mode (payments exist) that hides the proceed button — Cancel/Close only.
  let confirmDialogLines: string[] = $state([]);
  let confirmDialogBlocked = $state(false);

  function getOrderDateValue(value: any): Date | null {
    if (!value) return null;
    if (value instanceof Date) return Number.isNaN(value.getTime()) ? null : value;
    if (typeof value === 'string' || typeof value === 'number') {
      const parsed = new Date(value);
      return Number.isNaN(parsed.getTime()) ? null : parsed;
    }
    if (typeof value === 'object' && typeof value.toString === 'function') {
      const parsed = new Date(value.toString());
      return Number.isNaN(parsed.getTime()) ? null : parsed;
    }
    return null;
  }

  function createEmptyFormItem(lineNumber: number): EditableOrderItem {
    return {
      line_number: lineNumber,
      product_code: '',
      description: '',
      quantity: 1,
      unit_price_bhd: 0,
      total_price: 0,
      quantity_shipped: 0,
      quantity_invoiced: 0,
      equipment: '',
      model: '',
      specification: '',
      detailed_description: '',
      currency: 'BHD',
      fob: 0,
      freight: 0,
      total_cost: 0,
      margin_percent: 0
    };
  }

  function roundMoney(value: number): number {
    const numeric = Number(value) || 0;
    return Math.round(numeric * 1000) / 1000;
  }

  function safeTrim(value: unknown): string {
    return typeof value === 'string' ? value.trim() : '';
  }

  function normalizeOrderStatusKey(value: unknown): string {
    const raw = safeTrim(value);
    const compact = raw.toLowerCase().replace(/[\s_-]+/g, '');

    if (!compact || compact === 'all') return '';
    if (compact.includes('invoice')) return 'Invoiced';
    if (compact.includes('partial') && compact.includes('deliver')) return 'PartiallyDelivered';
    if (compact.includes('deliver') || compact.includes('complete') || compact.includes('closed')) return 'FullyDelivered';
    if (compact.includes('progress') || compact.includes('process')) return 'InProgress';
    if (compact.includes('cancel') || compact.includes('void')) return 'Cancelled';
    if (compact.includes('confirm') || compact.includes('open') || compact.includes('pending')) return 'Confirmed';

    return raw;
  }

  function orderMatchesStatus(order: crm.Order, statusKey: string): boolean {
    if (statusKey === 'All') return true;
    return normalizeOrderStatusKey(order.status) === statusKey;
  }

  function isSummaryLineItem(item: Partial<EditableOrderItem>): boolean {
    const description = safeTrim(item.description).toLowerCase().replace(/\s+/g, ' ');
    const productCode = safeTrim(item.product_code).toLowerCase().replace(/\s+/g, ' ');
    return description.startsWith('total for order') || productCode === 'total for order';
  }

  function sanitizeFormItems(items: EditableOrderItem[], targetTotal = 0): EditableOrderItem[] {
    const cleaned = items
      .filter((item) => !isSummaryLineItem(item))
      .map((item) => {
        const quantity = Number(item.quantity) || 0;
        const incomingTotal = Number(item.total_price) || 0;
        let unitPrice = Number(item.unit_price_bhd) || 0;
        if (unitPrice <= 0 && quantity > 0 && incomingTotal > 0) {
          unitPrice = roundMoney(incomingTotal / quantity);
        }
        const totalPrice = incomingTotal > 0 ? roundMoney(incomingTotal) : roundMoney(quantity * unitPrice);
        return {
          ...item,
          quantity,
          unit_price_bhd: roundMoney(unitPrice),
          total_price: totalPrice
        };
      })
      .filter((item) => {
        const hasIdentity = safeTrim(item.description) || safeTrim(item.product_code) || safeTrim(item.equipment) || safeTrim(item.model);
        const hasValue = (item.quantity || 0) > 0 && ((item.unit_price_bhd || 0) > 0 || (item.total_price || 0) > 0);
        return Boolean(hasIdentity || hasValue);
      });

    const deduped: EditableOrderItem[] = [];
    const seen = new Set<string>();
    for (const item of cleaned) {
      const signature = [
        safeTrim(item.product_code).toLowerCase(),
        safeTrim(item.description).toLowerCase(),
        roundMoney(item.quantity || 0),
        roundMoney(item.unit_price_bhd || 0),
        roundMoney(item.total_price || 0),
      ].join("|");
      if (seen.has(signature)) continue;
      seen.add(signature);
      deduped.push(item);
    }

    const normalizedTarget = roundMoney(targetTotal || 0);
    if (normalizedTarget > 0) {
      const exactMatch = deduped.find((item) => Math.abs(roundMoney(item.total_price || 0) - normalizedTarget) <= 0.01);
      if (exactMatch) {
        return [{ ...exactMatch, line_number: 1 }];
      }
    }

    return deduped.map((item, index) => ({
      ...item,
      line_number: index + 1,
    }));
  }

  function recalculateFormItems() {
    const normalizedItems = sanitizeFormItems(formItems);
    formItems = normalizedItems.length > 0 ? normalizedItems : [createEmptyFormItem(1)];
    formData.totalValue = roundMoney(normalizedItems.reduce((sum, item) => sum + (item.total_price || 0), 0));
  }

  function addFormItem() {
    const normalizedItems = sanitizeFormItems(formItems);
    formItems = [...normalizedItems, createEmptyFormItem(normalizedItems.length + 1)];
    formData.totalValue = roundMoney(normalizedItems.reduce((sum, item) => sum + (item.total_price || 0), 0));
  }

  function removeFormItem(index: number) {
    formItems = formItems.filter((_, itemIndex) => itemIndex !== index);
    if (formItems.length === 0) {
      formItems = [createEmptyFormItem(1)];
    }
    recalculateFormItems();
  }

  function getLineItemIssues(items: Array<{ quantity?: number; unit_price_bhd?: number; total_price?: number; description?: string; product_code?: string }>) {
    const normalized = items || [];
    return {
      missingItems: normalized.length === 0,
      zeroPriceItems: normalized.filter(item =>
        ((item.quantity || 0) > 0 || (item.description || '').trim() || (item.product_code || '').trim()) &&
        ((item.unit_price_bhd || 0) <= 0 || (item.total_price || 0) <= 0)
      ).length
    };
  }

  function askActionConfirmation(options: {
    title: string;
    message: string;
    confirmLabel?: string;
    variant?: ConfirmVariant;
    lines?: string[];
    blocked?: boolean;
  }): Promise<boolean> {
    return new Promise((resolve) => {
      if (confirmDialogResolve) {
        confirmDialogResolve(false);
      }
      confirmDialogTitle = options.title;
      confirmDialogMessage = options.message;
      confirmDialogConfirmLabel = options.confirmLabel || 'Confirm';
      confirmDialogVariant = options.variant || 'primary';
      confirmDialogLines = options.lines || [];
      confirmDialogBlocked = options.blocked || false;
      confirmDialogResolve = resolve;
      confirmDialogOpen = true;
    });
  }

  function resolveActionConfirmation(confirmed: boolean) {
    const resolver = confirmDialogResolve;
    confirmDialogOpen = false;
    confirmDialogResolve = null;
    if (resolver) {
      resolver(confirmed);
    }
  }

  // DataTable columns
  const columns = [
    {
      key: 'order_number',
      label: 'Order Number',
      sortable: true,
      type: 'text' as const,
      width: '140px'
    },
    {
      key: 'customer_name',
      label: 'Customer',
      sortable: true,
      type: 'text' as const,
      render: (row) => escapeHtml(row.customer_name)
    },
    {
      key: 'customer_po_number',
      label: 'RFQ/Offer Ref',
      sortable: false,
      type: 'text' as const,
      width: '140px'
    },
    {
      key: 'total_value_bhd',
      label: 'Total (BHD)',
      sortable: true,
      type: 'currency' as const,
      align: 'right' as const,
      width: '140px'
    },
    {
      key: 'order_date',
      label: 'Order Date',
      sortable: true,
      type: 'date' as const,
      width: '120px'
    },
    {
      key: 'delivery_status',
      label: 'Delivery',
      sortable: false,
      type: 'text' as const,
      width: '140px'
    },
    {
      key: 'status',
      label: 'Status',
      sortable: true,
      type: 'status' as const,
      width: '160px'
    }
  ];

  // B10-1: batch the per-order delivery status fetch into ONE round-trip
  // instead of an N+1 loop of GetOrderDeliveryStatus calls.
  async function loadOrderDeliveryStatusBatch(orderList: crm.Order[]) {
    if (orderList.length === 0) return;
    try {
      const batch = await GetOrderDeliveryStatusBatch(orderList.map((o) => o.id));
      orderList.forEach((order) => {
        (order as any).deliveryStatus = (batch as any)?.[order.id] || null;
      });
    } catch (err) {
      orderList.forEach((order) => {
        (order as any).deliveryStatus = null;
      });
    }
  }

  function updateYearsAndCustomers(orderList: crm.Order[]) {
    const yearsSet = new Set<string>(availableYears);
    const customersSet = new Set<string>(availableCustomers);
    orderList.forEach(o => {
      const orderDate = getOrderDateValue(o.order_date);
      if (orderDate) {
        yearsSet.add(orderDate.getFullYear().toString());
      }
      if (o.customer_name) {
        customersSet.add(o.customer_name);
      }
    });
    availableYears = Array.from(yearsSet).sort().reverse();
    availableCustomers = Array.from(customersSet).sort();
  }

  // Load data (first page)
  async function loadOrders() {
    loading = true;
    currentPage = 0;
    hasMoreOrders = true;
    try {
      const [ordersData, customersData] = await Promise.all([
        ListOrders(PAGE_SIZE, 0),
        ListCustomers(500, 0)
      ]);

      orders = ordersData || [];
      customers = customersData || [];
      currentPage = 1;
      totalOrdersLoaded = orders.length;
      hasMoreOrders = orders.length === PAGE_SIZE;

      await loadOrderDeliveryStatusBatch(orders);

      availableYears = [];
      availableCustomers = [];
      updateYearsAndCustomers(orders);

      applyFilters();
    } catch (err) {
      const errorMsg = err?.message || String(err);
      toast.danger(`Failed to load orders: ${errorMsg}`);
    } finally {
      loading = false;
    }
  }

  // Load more orders (pagination) — mirrors the Load-more pattern used in
  // PaymentsScreen/InvoicesScreen.
  async function loadMoreOrders() {
    if (loadingMoreOrders || !hasMoreOrders) return;

    loadingMoreOrders = true;
    try {
      const offset = currentPage * PAGE_SIZE;
      const data = await ListOrders(PAGE_SIZE, offset);

      if (data && data.length > 0) {
        await loadOrderDeliveryStatusBatch(data);
        orders = [...orders, ...data];
        currentPage++;
        totalOrdersLoaded = orders.length;
        hasMoreOrders = data.length === PAGE_SIZE;
        updateYearsAndCustomers(data);
        applyFilters();
      } else {
        hasMoreOrders = false;
      }
    } catch (err) {
      console.error('Failed to load more orders:', err);
      toast.danger('Failed to load more orders');
    } finally {
      loadingMoreOrders = false;
    }
  }

  // Apply filters
  function applyFilters() {
    let result = [...orders];

    // Status filter
    if (activeFilter !== 'All') {
      result = result.filter(o => orderMatchesStatus(o, activeFilter));
    }

    // Year filter
    if (selectedYear !== 'All') {
      result = result.filter(o => {
        const orderDate = getOrderDateValue(o.order_date);
        if (!orderDate) return false;
        return orderDate.getFullYear().toString() === selectedYear;
      });
    }

    // Customer filter
    if (selectedCustomer !== 'All') {
      result = result.filter(o => o.customer_name === selectedCustomer);
    }

    // Search filter
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter(o =>
        o.order_number?.toLowerCase().includes(query) ||
        o.customer_name?.toLowerCase().includes(query) ||
        o.customer_po_number?.toLowerCase().includes(query)
      );
    }

    filteredOrders = result;
  }

  run(() => {
    searchQuery;
    activeFilter;
    selectedYear;
    selectedCustomer;
    applyFilters();
  });

  // Stats
  let stats = $derived({
    total: orders.length,
    confirmed: orders.filter(o => orderMatchesStatus(o, 'Confirmed')).length,
    inProgress: orders.filter(o => orderMatchesStatus(o, 'InProgress')).length,
    delivered: orders.filter(o => orderMatchesStatus(o, 'FullyDelivered')).length,
    totalValue: orders.reduce((sum, o) => sum + (o.total_value_bhd || 0), 0)
  });

  let zeroValueOrders = $derived(orders.filter((order) => (order.total_value_bhd || 0) <= 0));
  // B7: drives disabled+explained state for the DN/Supplier-Order CTAs on the open order's detail modal
  let hasNoItemsSelected = $derived(orderItems.length === 0);

  // Open create modal
  function openCreateModal() {
    modalMode = 'create';
    editingOrder = null;
    formData = {
      orderNumber: '',
      customerPONumber: '',
      customerId: '',
      customerName: '',
      orderDate: new Date().toISOString().split('T')[0],
      requiredDate: '',
      totalValue: 0,
      status: 'Confirmed',
      paymentTerms: 'Net 30',
      deliveryTerms: 'Ex-Works'
    };
    formItems = [createEmptyFormItem(1)];
    showModal = true;
  }

  // Open edit modal
  async function openEditModal(order: crm.Order) {
    modalMode = 'edit';
    const fullOrder = (order.items && order.items.length > 0) ? order : await GetOrder(order.id);
    editingOrder = fullOrder;
    const resolvedCustomerId = resolveCustomerId(fullOrder.customer_id || '', fullOrder.customer_name || '');
    formData = {
      orderNumber: fullOrder.order_number || '',
      customerPONumber: fullOrder.customer_po_number || '',
      customerId: resolvedCustomerId,
      customerName: fullOrder.customer_name || '',
      orderDate: fullOrder.order_date ? new Date(fullOrder.order_date as any).toISOString().split('T')[0] : '',
      requiredDate: fullOrder.required_date ? new Date(fullOrder.required_date as any).toISOString().split('T')[0] : '',
      totalValue: fullOrder.total_value_bhd || 0,
      status: normalizeOrderStatusKey(fullOrder.status) || 'Confirmed',
      paymentTerms: fullOrder.payment_terms || 'Net 30',
      deliveryTerms: fullOrder.delivery_terms || 'Ex-Works'
    };
    formItems = fullOrder.items && fullOrder.items.length > 0
      ? sanitizeFormItems(fullOrder.items.map((item) => ({ ...item } as EditableOrderItem)), fullOrder.total_value_bhd || fullOrder.grand_total_bhd || 0)
      : [createEmptyFormItem(1)];
    showDetailModal = false;
    showModal = true;
  }

  // Handle form submit
  async function handleSubmit() {
    // Validation
    if (!formData.orderNumber.trim()) {
      toast.warning('Order Number is required');
      return;
    }

    if (!formData.customerName.trim() && !formData.customerId.trim()) {
      toast.warning('Customer is required');
      return;
    }

    // Validate total value is non-negative
    if (formData.totalValue < 0) {
      toast.warning('Total Value cannot be negative');
      return;
    }

    const normalizedItems = sanitizeFormItems(formItems).filter(item =>
      safeTrim(item.description) || safeTrim(item.product_code) || item.quantity > 0 || item.unit_price_bhd > 0
    );

    if (normalizedItems.length === 0) {
      toast.warning('Add at least one line item with quantity and price before saving this order');
      return;
    }

    for (const item of normalizedItems) {
      if (item.quantity <= 0) {
        toast.warning('Each line item must have a quantity greater than zero');
        return;
      }
      if (item.unit_price_bhd <= 0) {
        toast.warning('Each line item must have a unit price greater than zero');
        return;
      }
    }

    // Validate dates
    const orderDate = new Date(formData.orderDate);
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    if (orderDate > today) {
      toast.warning('Order date cannot be in the future');
      return;
    }

    if (formData.requiredDate) {
      const requiredDate = new Date(formData.requiredDate);
      if (requiredDate < orderDate) {
        toast.warning('Required date must be after order date');
        return;
      }
    }

    saving = true;
    try {
      const resolvedCustomerId = resolveCustomerId(formData.customerId, formData.customerName);
      const computedTotalValue = roundMoney(normalizedItems.reduce((sum, item) => {
        const lineTotal = roundMoney((Number(item.quantity) || 0) * (Number(item.unit_price_bhd) || 0));
        return sum + lineTotal;
      }, 0));
      formData.customerId = resolvedCustomerId;
      formData.totalValue = computedTotalValue;

      if (modalMode === 'create') {
        // Wave 9.7 tight-ship fix: header + items now go through a single
        // atomic backend call (CreateOrderWithItems) instead of a separate
        // CreateOrder + UpdateOrder pair — the old two-call sequence could
        // leave a header-only "ghost" order behind if the second call failed.
        const orderHeader = {
          order_number: formData.orderNumber,
          customer_po_number: formData.customerPONumber,
          customer_id: resolvedCustomerId,
          customer_name: formData.customerName,
          order_date: formData.orderDate,
          required_date: formData.requiredDate,
          total_value_bhd: computedTotalValue,
          grand_total_bhd: computedTotalValue,
          status: formData.status,
          payment_terms: formData.paymentTerms,
          delivery_terms: formData.deliveryTerms
        };

        const orderItems = normalizedItems.map((item, index) => ({
          ...item,
          line_number: index + 1,
          total_price: roundMoney(item.quantity * item.unit_price_bhd)
        }));

        await CreateOrderWithItems(orderHeader as any, orderItems as any);
        toast.success('Order created successfully');
      } else if (modalMode === 'edit' && editingOrder) {
        // Send full order update, not just status
        const updatedOrder = {
          ...editingOrder,
          order_number: formData.orderNumber,
          customer_po_number: formData.customerPONumber,
          customer_id: resolvedCustomerId,
          customer_name: formData.customerName,
          order_date: formData.orderDate,
          required_date: formData.requiredDate,
          total_value_bhd: computedTotalValue,
          status: formData.status,
          payment_terms: formData.paymentTerms,
          delivery_terms: formData.deliveryTerms,
          items: normalizedItems.map((item, index) => ({
            ...item,
            line_number: index + 1,
            total_price: roundMoney(item.quantity * item.unit_price_bhd)
          }))
        };
        await UpdateOrder(editingOrder.id, updatedOrder as any);
        toast.success('Order updated successfully');
      }

      showModal = false;
      await Promise.all([loadOrders(), loadOrdersWithNoItems()]);
    } catch (err) {
      const errorMsg = err?.message || String(err);
      toast.danger('Failed to save order: ' + errorMsg);
    } finally {
      saving = false;
    }
  }

  // Handle row click - open detail modal
  async function handleRowClick(row: crm.Order) {
    selectedOrder = row;
    linkedInvoices = [];
    loadingInvoices = true;

    try {
      const [fullOrder, fulfillment, invoices] = await Promise.all([
        GetOrder(row.id),
        GetOrderFulfillmentStatus(row.id).catch(() => null),
        GetInvoicesByOrder(row.id).catch(() => [])
      ]);

      selectedOrder = fullOrder;
      linkedInvoices = invoices || [];
      orderItems = sanitizeFormItems(
        (fullOrder.items || []).map((item) => ({ ...item } as EditableOrderItem)),
        fullOrder.total_value_bhd || fullOrder.grand_total_bhd || 0,
      );
      fulfillmentStatus = fulfillment;

      if (orderItems.length > 0) {
        const totalQty = orderItems.reduce((sum, item) => sum + item.quantity, 0);
        const shippedQty = orderItems.reduce((sum, item) => sum + (item.quantity_shipped || 0), 0);
        const percentage = totalQty > 0 ? Math.round((shippedQty / totalQty) * 100) : 0;
        deliveryProgress = { delivered: shippedQty, total: totalQty, percentage };
      } else {
        deliveryProgress = { delivered: 0, total: 0, percentage: 0 };
      }
    } catch (err) {
      const errorMsg = err?.message || String(err);
      toast.warning(`Could not load full order details: ${errorMsg}`);
    }

    loadingInvoices = false;
    showDetailModal = true;
  }

  // Quick actions
  async function handleCreateDeliveryNote(order: crm.Order) {
    if (isProcessingAction) return;

    if (!orderItems || orderItems.length === 0) {
      toast.warning('This order has no saved line items. Delivery notes need the actual items to ship; edit or re-import the order before creating a DN.');
      return;
    }

    // Set pending DN creation in store (DeliveryNotesScreen will consume on mount)
    pendingDNCreate.request(order.id, order.order_number || '', order.customer_name || '');
    showDetailModal = false;
    // Navigate to operations screen with delivery-notes tab
    window.dispatchEvent(new CustomEvent('navigateToScreen', {
      detail: { screen: 'operations', tab: 'delivery-notes' }
    }));
  }

  // Wave 9.4 B4.1: "Start project" handoff — a live order becomes a WorkHub
  // project in one action, with lineage (order id, customer, attention
  // person/phone as POC where available) preseeded into the create composer.
  function handleStartProject(order: crm.Order) {
    if (!order) return;
    pendingProjectHandoff.request({
      source: 'order',
      sourceId: order.id,
      orderId: order.id,
      customerId: order.customer_id || undefined,
      customerName: order.customer_name || '',
      pocName: order.attention_person || '',
      pocPhone: order.attention_phone || '',
      suggestedName: order.customer_name
        ? `${order.customer_name} — ${order.order_number || order.id}`
        : (order.order_number || 'Order'),
    });
    showDetailModal = false;
    window.dispatchEvent(new CustomEvent('navigateToScreen', {
      detail: { screen: 'work' }
    }));
  }

  async function handleCreateInvoice(order: crm.Order) {
    if (isProcessingAction) return;

    const confirmed = await askActionConfirmation({
      title: 'Create Invoice',
      message: `Create invoice for order ${order.order_number}? This will generate a billing record for ${order.customer_name}.`,
      confirmLabel: 'Create Invoice',
      variant: 'primary'
    });
    if (!confirmed) {
      return;
    }

    isProcessingAction = true;
    try {
      const invoice = await CreateInvoiceFromOrder(order.id);
      toast.success(`Invoice ${invoice.invoice_number} created`);
      showDetailModal = false;
      window.dispatchEvent(new CustomEvent('navigateToScreen', {
        detail: { screen: 'finance', tab: 'invoices', company: order.division || 'Acme Instrumentation' }
      }));
      window.dispatchEvent(new CustomEvent('finance:navigate', {
        detail: { tab: 'invoices', company: order.division || 'Acme Instrumentation' }
      }));
      await loadOrders();
    } catch (err) {
      toast.danger('Failed to create invoice: ' + (err as Error).message);
    } finally {
      isProcessingAction = false;
    }
  }

  async function handleCreatePurchaseOrder(order: crm.Order) {
    if (isProcessingAction) return;

    if (!orderItems || orderItems.length === 0) {
      toast.warning('This order has no saved line items. Supplier orders need item detail; edit or re-import the order before creating a PO.');
      return;
    }

    const confirmed = await askActionConfirmation({
      title: 'Create Supplier Order',
      message: `Create a draft supplier order from ${order.order_number}? If the order contains items from multiple suppliers, create supplier orders separately.`,
      confirmLabel: 'Create Supplier Order',
      variant: 'secondary'
    });
    if (!confirmed) {
      return;
    }

    isProcessingAction = true;
    try {
      // B7: pass the order's own line-item IDs (not []) so PO supplier inference and
      // item selection are scoped to exactly what this order carries.
      const itemIds = (order.items || [])
        .map((item) => item.id)
        .filter((id): id is string => Boolean(id));
      const po = await CreatePOFromOrder(order.id, '', itemIds);
      // B7: land inside the new PO draft instead of the PO list (context-preserving handoff)
      sessionStorage.setItem('asymmflow.pendingOpenPO', JSON.stringify({ id: po.id, number: po.po_number }));
      toast.success(`Supplier order ${po.po_number} created — approve to send`);
      showDetailModal = false;
      window.dispatchEvent(new CustomEvent('navigateToScreen', {
        detail: { screen: 'operations', tab: 'pos' }
      }));
    } catch (err) {
      toast.danger('Failed to create supplier order: ' + errorMessage(err));
    } finally {
      isProcessingAction = false;
    }
  }

  async function handleCreateProforma(order: crm.Order) {
    if (isProcessingAction) return;

    const confirmed = await askActionConfirmation({
      title: 'Create Proforma',
      message: `Create proforma invoice for order ${order.order_number}? This is a reference document, not a tax invoice.`,
      confirmLabel: 'Create Proforma',
      variant: 'secondary'
    });
    if (!confirmed) {
      return;
    }

    isProcessingAction = true;
    try {
      const invoice = await CreateProformaInvoice(order.id);
      toast.success(`Proforma Invoice ${invoice.invoice_number} created`);
      showDetailModal = false;
      window.dispatchEvent(new CustomEvent('navigateToScreen', {
        detail: { screen: 'finance', tab: 'invoices', company: order.division || 'Acme Instrumentation' }
      }));
      window.dispatchEvent(new CustomEvent('finance:navigate', {
        detail: { tab: 'invoices', company: order.division || 'Acme Instrumentation' }
      }));
      await loadOrders();
    } catch (err) {
      toast.danger('Failed to create proforma: ' + errorMessage(err));
    } finally {
      isProcessingAction = false;
    }
  }

  async function handleQuickMarkDelivered() {
    if (isProcessingAction) return;
    if (!selectedOrder) return;

    const confirmed = await askActionConfirmation({
      title: 'Mark Delivered',
      message: `Mark order ${selectedOrder.order_number} as fully delivered? This updates the order status without creating a GRN record.`,
      confirmLabel: 'Mark Delivered',
      variant: 'primary'
    });
    if (!confirmed) {
      return;
    }

    isProcessingAction = true;
    try {
      const message = await QuickMarkOrderDelivered(selectedOrder.id);
      toast.success(message);
      showDetailModal = false;
      await loadOrders();
    } catch (err) {
      toast.danger('Failed to mark as delivered: ' + (err as Error).message);
    } finally {
      isProcessingAction = false;
    }
  }

  async function handleDeleteOrder(order: crm.Order) {
    if (isDeleting) return;

    // C3: preview the cascade before the destructive confirm (Article III.2 +
    // pattern #7) and pre-empt the server's PAYMENT_EXISTS block rather than
    // letting DeleteOrder throw.
    let cascade: Record<string, any> | null = null;
    try {
      cascade = await PreviewOrderDeleteCascade(order.id);
    } catch (err) {
      toast.danger('Failed to check delete impact: ' + (err as Error).message);
      return;
    }

    if (cascade?.blocked) {
      await askActionConfirmation({
        title: 'Cannot Delete Order',
        message: cascade.block_reason || `Order ${order.order_number} cannot be deleted while dependent records exist.`,
        lines: cascade.summary,
        blocked: true,
      });
      return;
    }

    const zeroDependents =
      (cascade?.order_item_count || 0) === 0 &&
      (cascade?.invoice_count || 0) === 0 &&
      (cascade?.invoice_item_count || 0) === 0 &&
      (cascade?.purchase_order_count || 0) === 0 &&
      (cascade?.purchase_order_item_count || 0) === 0 &&
      (cascade?.delivery_note_count || 0) === 0 &&
      (cascade?.delivery_note_item_count || 0) === 0;

    const confirmed = await askActionConfirmation({
      title: 'Delete Order',
      message: `Delete order ${order.order_number}? This cannot be undone.`,
      lines: zeroDependents
        ? ['No dependent records — this order can be safely deleted.']
        : (cascade?.summary || []),
      confirmLabel: 'Delete Order',
      variant: 'danger'
    });
    if (!confirmed) {
      return;
    }

    isDeleting = true;
    try {
      await DeleteOrder(order.id);
      toast.success(`Order ${order.order_number} deleted`);
      showDetailModal = false;
      selectedOrder = null;
      await loadOrders();
    } catch (err) {
      toast.danger('Delete failed: ' + (err as Error).message);
    } finally {
      isDeleting = false;
    }
  }

  // Format currency
  function formatCurrency(value: number): string {
    return new Intl.NumberFormat('en-BH', {
      style: 'currency',
      currency: 'BHD',
      minimumFractionDigits: 3,
      maximumFractionDigits: 3
    }).format(value);
  }

  // Format date
  function formatDate(date: any): string {
    if (!date) return '—';
    return new Date(date).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    });
  }

  // Format delivery status
  function formatDeliveryStatus(order: crm.Order): string {
    const deliveryStatus = (order as any).deliveryStatus;
    if (!deliveryStatus) return '—';

    const totalItems = Object.keys(deliveryStatus).length;
    const deliveredItems = Object.values(deliveryStatus).filter((qty: any) => qty > 0).length;

    if (deliveredItems === 0) return 'Not Started';
    if (deliveredItems === totalItems) return '100% Complete';
    return `${Math.round((deliveredItems / totalItems) * 100)}% (${deliveredItems}/${totalItems} items)`;
  }

  // B1b: OffersScreen sets pendingOrderView after MarkOfferWon so "View Order →"
  // lands on the created order's detail instead of a toast-only dead end.
  // handleRowClick fetches the full order via GetOrder regardless of whether
  // it's already present in the (now paginated) `orders` list.
  async function checkPendingOrderView() {
    const pending = $pendingOrderView;
    if (!pending) return;
    pendingOrderView.clear();
    await handleRowClick({ id: pending.orderId, order_number: pending.orderNumber } as crm.Order);
  }

  onMount(() => {
    loadOrders();
    loadOrdersWithNoItems();
    void checkPendingOrderView();
  });

  async function loadOrdersWithNoItems() {
    try {
      const data = await GetOrdersWithNoItems();
      ordersWithNoItems = data || [];
    } catch (err) {
      // Non-critical — silently ignore
      ordersWithNoItems = [];
    }
  }
</script>

{#if embedded}
  <!-- Embedded mode - ALL orders grouped by year -->
  <div class="orders-embedded">
    <!-- Summary bar -->
    <div class="embedded-header">
      <h3>All Orders ({orders.length})</h3>
      <div class="embedded-stats">
        <span class="embedded-stat">{formatCurrency(stats.totalValue)} total</span>
        <span class="embedded-stat">{stats.confirmed} confirmed</span>
        <span class="embedded-stat">{stats.delivered} delivered</span>
      </div>
    </div>

    <!-- Filters -->
    <div class="embedded-filters">
      <input
        type="text"
        placeholder="Search orders..."
        bind:value={searchQuery}
        class="embedded-search"
      />
      <select class="embedded-select" bind:value={activeFilter}>
        <option value="All">All Status</option>
        {#each ORDER_STATUSES as status}
          <option value={status.key}>{status.label}</option>
        {/each}
      </select>
    </div>

    {#if loading}
      <div class="loading-container">
        <WabiSpinner size="md" />
      </div>
    {:else if filteredOrders.length === 0}
      <p class="empty-message">No orders yet — won offers become orders here.</p>
    {:else}
      <!-- Group orders by year -->
      {#each availableYears as year}
        {@const yearOrders = filteredOrders.filter(o => {
          const orderDate = getOrderDateValue(o.order_date);
          return orderDate ? orderDate.getFullYear().toString() === year : false;
        })}
        {#if yearOrders.length > 0}
          <div class="year-group">
            <div class="year-header">
              <span class="year-label">{year}</span>
              <span class="year-count">{yearOrders.length} orders</span>
              <span class="year-value">{formatCurrency(yearOrders.reduce((sum, o) => sum + (o.total_value_bhd || 0), 0))}</span>
            </div>
            <div class="orders-list">
              {#each yearOrders as order}
                <div class="order-item" role="button" tabindex="0" onclick={() => handleRowClick(order)} onkeydown={(event) => (event.key === "Enter" || event.key === " ") && handleRowClick(order)}>
                  <div class="order-left">
                    <span class="order-number">{order.order_number}</span>
                    <span class="order-customer">{order.customer_name}</span>
                    {#if order.customer_po_number}
                      <span class="order-ref">Ref: {order.customer_po_number}</span>
                    {/if}
                  </div>
                  <div class="order-right">
                    <StatusBadge status={order.status} size="sm" />
                    <span class="order-value">{formatCurrency(order.total_value_bhd || 0)}</span>
                    <span class="order-date">{formatDate(order.order_date)}</span>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      {/each}

      <!-- Orders without dates -->
      {@const undatedOrders = filteredOrders.filter(o => !o.order_date)}
      {#if undatedOrders.length > 0}
        <div class="year-group">
          <div class="year-header">
            <span class="year-label">No Date</span>
            <span class="year-count">{undatedOrders.length} orders</span>
          </div>
          <div class="orders-list">
            {#each undatedOrders as order}
              <div class="order-item" role="button" tabindex="0" onclick={() => handleRowClick(order)} onkeydown={(event) => (event.key === "Enter" || event.key === " ") && handleRowClick(order)}>
                <div class="order-left">
                  <span class="order-number">{order.order_number}</span>
                  <span class="order-customer">{order.customer_name}</span>
                </div>
                <div class="order-right">
                  <StatusBadge status={order.status} size="sm" />
                  <span class="order-value">{formatCurrency(order.total_value_bhd || 0)}</span>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    {/if}

    {#if hasMoreOrders && !loading}
      <div class="pagination-controls">
        <button
          class="load-more-btn"
          onclick={loadMoreOrders}
          disabled={loadingMoreOrders}
          aria-label={loadingMoreOrders ? 'Loading more orders' : `Load more orders, ${totalOrdersLoaded} currently loaded`}
        >
          {#if loadingMoreOrders}
            Loading more...
          {:else}
            Load More ({totalOrdersLoaded} loaded)
          {/if}
        </button>
      </div>
    {/if}
    {#if !hasMoreOrders && totalOrdersLoaded > 0 && !loading}
      <p class="all-loaded">All {totalOrdersLoaded} orders loaded</p>
    {/if}
  </div>
{:else}
  <!-- Full screen mode with DataTable -->
  <PageLayout title="Orders" subtitle="Sales Order Management">
    <!-- @migration-task: migrate this slot by hand, `header-actions` is an invalid identifier -->
  <div slot="header-actions" class="header-actions">
      <div class="stats-row">
        <div class="stat-item">
          <span class="stat-value">{stats.total}</span>
          <span class="stat-label">Total Orders</span>
        </div>
        <div class="stat-item">
          <span class="stat-value">{formatCurrency(stats.totalValue)}</span>
          <span class="stat-label">Total Value</span>
        </div>
      </div>
      <Button variant="primary" on:click={openCreateModal}>
        + New Order
      </Button>
    </div>

    <!-- Warning: Orders with no line items -->
    {#if ordersWithNoItems.length > 0}
      <div class="no-items-warning">
        <span class="warning-icon">&#9888;</span>
        <div class="warning-content">
          <strong>{ordersWithNoItems.length} order{ordersWithNoItems.length === 1 ? '' : 's'} ha{ordersWithNoItems.length === 1 ? 's' : 've'} no line items.</strong>
          Edit these orders to add items before invoicing.
          <div class="warning-list">
            {#each ordersWithNoItems as item}
              <span class="warning-order-number">{escapeHtml(item.order_number || item.id)}</span>
            {/each}
          </div>
        </div>
      </div>
    {/if}

    {#if zeroValueOrders.length > 0}
      <div class="no-items-warning no-items-warning-critical">
        <span class="warning-icon">&#9888;</span>
        <div class="warning-content">
          <strong>{zeroValueOrders.length} order{zeroValueOrders.length === 1 ? '' : 's'} currently have zero total value.</strong>
          Review the line items and unit prices before using them for delivery, invoicing, or procurement.
          <div class="warning-list">
            {#each zeroValueOrders.slice(0, 10) as item}
              <span class="warning-order-number">{escapeHtml(item.order_number || item.id)}</span>
            {/each}
          </div>
        </div>
      </div>
    {/if}

    <!-- Filters -->
    <div class="controls-bar">
      <div class="filter-tabs">
        <button
          class="filter-tab"
          class:active={activeFilter === 'All'}
          onclick={() => activeFilter = 'All'}
        >
          All ({orders.length})
        </button>
        {#each ORDER_STATUSES as status}
          <button
            class="filter-tab"
            class:active={activeFilter === status.key}
            onclick={() => activeFilter = status.key}
          >
            {status.label} ({orders.filter(o => orderMatchesStatus(o, status.key)).length})
          </button>
        {/each}
      </div>

      <div class="dropdown-filters">
        <select class="filter-select" bind:value={selectedYear}>
          <option value="All">All Years</option>
          {#each availableYears as year}
            <option value={year}>{year}</option>
          {/each}
        </select>

        <select class="filter-select" bind:value={selectedCustomer}>
          <option value="All">All Customers</option>
          {#each availableCustomers as customer}
            <option value={customer}>{customer}</option>
          {/each}
        </select>
      </div>

      <div class="search-box">
        <input
          type="text"
          placeholder="Search orders..."
          bind:value={searchQuery}
          class="search-input"
        />
      </div>
    </div>

    <!-- DataTable -->
    <Card>
      {#if loading}
        <div class="loading-container">
          <WabiSpinner size="lg" />
        </div>
      {:else}
        <DataTable
          {columns}
          data={filteredOrders}
          {loading}
          emptyMessage="No orders yet — won offers become orders here."
          onRowClick={handleRowClick}
          keyField="id"
          stickyHeader={true}
          maxHeight="calc(100vh - 280px)"
        >
          {#snippet cell({ column, row, value })}
                    <div    >
              {#if column.key === 'status'}
                <StatusBadge status={value} />
              {:else if column.key === 'delivery_status'}
                <div class="delivery-progress">
                  <span class="progress-text">{formatDeliveryStatus(row)}</span>
                </div>
              {:else if column.key === 'customer_po_number'}
                <span class="ref-value">{value || '—'}</span>
              {:else}
                {value}
              {/if}
            </div>
                  {/snippet}
        </DataTable>
      {/if}
    </Card>

    {#if hasMoreOrders && !loading}
      <div class="pagination-controls">
        <button
          class="load-more-btn"
          onclick={loadMoreOrders}
          disabled={loadingMoreOrders}
          aria-label={loadingMoreOrders ? 'Loading more orders' : `Load more orders, ${totalOrdersLoaded} currently loaded`}
        >
          {#if loadingMoreOrders}
            Loading more...
          {:else}
            Load More ({totalOrdersLoaded} loaded)
          {/if}
        </button>
      </div>
    {/if}
    {#if !hasMoreOrders && totalOrdersLoaded > 0 && !loading}
      <p class="all-loaded">All {totalOrdersLoaded} orders loaded</p>
    {/if}
  </PageLayout>
{/if}

<!-- Create/Edit Modal -->
{#if showModal}
  <Modal
    title={modalMode === 'create' ? 'New Order' : 'Edit Order'}
    open={showModal}
    on:close={() => showModal = false}
    size="xl"
  >
    <form onsubmit={preventDefault(handleSubmit)}>
      <div class="form-row">
        <FormGroup label="Order Number" required>
          <Input
            bind:value={formData.orderNumber}
            placeholder="ORD-2025-001"
            disabled={modalMode === 'edit'}
          />
        </FormGroup>

        <FormGroup label="Customer PO Number">
          <Input
            bind:value={formData.customerPONumber}
            placeholder="Customer's PO reference"
          />
        </FormGroup>
      </div>

      <FormGroup label="Customer" required>
        <input
          list="customers-list"
          bind:value={formData.customerName}
          placeholder="Search customer..."
          class="form-input"
        />
        <datalist id="customers-list">
          {#each customers as customer}
            <option value={customer.business_name}>{customer.business_name}</option>
          {/each}
        </datalist>
      </FormGroup>

      <div class="form-row">
        <FormGroup label="Order Date" required>
          <Input type="date" bind:value={formData.orderDate} />
        </FormGroup>

        <FormGroup label="Required Date">
          <Input type="date" bind:value={formData.requiredDate} />
        </FormGroup>
      </div>

      <div class="form-row">
        <FormGroup label="Total Value (BHD)">
          <Input
            type="number"
            step="0.001"
            min="0"
            bind:value={formData.totalValue}
            placeholder="0.000"
            readonly
          />
        </FormGroup>

        <FormGroup label="Status" required>
          <select bind:value={formData.status} class="form-select">
            {#each ORDER_STATUSES as status}
              <option value={status.key}>{status.label}</option>
            {/each}
          </select>
        </FormGroup>
      </div>

      <div class="form-row">
        <FormGroup label="Payment Terms">
          <select bind:value={formData.paymentTerms} class="form-select">
            <option value="Net 7">Net 7</option>
            <option value="Net 15">Net 15</option>
            <option value="Net 30">Net 30</option>
            <option value="Net 45">Net 45</option>
            <option value="Net 60">Net 60</option>
            <option value="Cash on Delivery">Cash on Delivery</option>
            <option value="Advance Payment">Advance Payment</option>
          </select>
        </FormGroup>

        <FormGroup label="Delivery Terms">
          <select bind:value={formData.deliveryTerms} class="form-select">
            <option value="Ex-Works">Ex-Works</option>
            <option value="FOB">FOB</option>
            <option value="CIF">CIF</option>
            <option value="DDP">DDP</option>
          </select>
        </FormGroup>
      </div>

      <div class="line-items-panel">
        <div class="line-items-panel-header">
          <div>
            <h4>Order Line Items</h4>
            <p>Capture each line separately so delivery notes, invoices, and purchase orders stay accurate.</p>
          </div>
        </div>

        {#if getLineItemIssues(formItems).missingItems}
          <div class="line-item-alert">
            This order has no line items yet. Add them now so downstream documents carry the right quantities and values.
          </div>
        {:else if getLineItemIssues(formItems).zeroPriceItems > 0}
          <div class="line-item-alert">
            {getLineItemIssues(formItems).zeroPriceItems} line item{getLineItemIssues(formItems).zeroPriceItems === 1 ? '' : 's'} still have zero pricing. Please complete them before saving.
          </div>
        {/if}

        <!-- Wave 9.6 B2: extracted to the canonical LineItemsEditor — this
             screen keeps ALL calculation (sanitizeFormItems /
             recalculateFormItems / roundMoney / handleSubmit's authoritative
             total); the component is presentation only. -->
        <LineItemsEditor
          mode="order"
          items={formItems}
          maxItems={999}
          formatBHD={formatCurrency}
          onRecalculate={recalculateFormItems}
          onRemoveItem={removeFormItem}
          onAddItem={addFormItem}
        />
      </div>

      <div class="modal-actions">
        <Button variant="ghost" on:click={() => showModal = false} disabled={saving}>
          Cancel
        </Button>
        <Button variant="primary" type="submit" disabled={saving}>
          {saving ? 'Saving...' : modalMode === 'edit' ? 'Update Order' : 'Create Order'}
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- Order Detail Modal -->
{#if showDetailModal && selectedOrder}
  <Modal
    title={`Order ${selectedOrder.order_number}`}
    open={showDetailModal}
    on:close={() => { showDetailModal = false; showTaskModal = false; }}
    size="lg"
  >
    <div class="order-detail">
      {#if getLineItemIssues(orderItems).missingItems || getLineItemIssues(orderItems).zeroPriceItems > 0 || (selectedOrder.total_value_bhd || 0) <= 0}
        <div class="line-item-alert detail-alert">
          {#if getLineItemIssues(orderItems).missingItems}
            This order does not have any saved line items yet.
          {:else if getLineItemIssues(orderItems).zeroPriceItems > 0}
            {getLineItemIssues(orderItems).zeroPriceItems} line item{getLineItemIssues(orderItems).zeroPriceItems === 1 ? '' : 's'} still have zero pricing.
          {:else}
            This order total is zero and should be reviewed.
          {/if}
          Use <strong>Edit Order</strong> to correct the commercial data before creating downstream documents.
        </div>
      {/if}

      <!-- Header Info -->
      <div class="detail-header">
        <div class="detail-section">
          <h4>Customer</h4>
          <p>{selectedOrder.customer_name}</p>
        </div>
        <div class="detail-section">
          <h4>Order Date</h4>
          <p>{formatDate(selectedOrder.order_date)}</p>
        </div>
        <div class="detail-section">
          <h4>Status</h4>
          <StatusBadge status={selectedOrder.status} />
        </div>
        <div class="detail-section">
          <h4>Total Value</h4>
          <p class="value-highlight">{formatCurrency(selectedOrder.total_value_bhd || 0)}</p>
        </div>
      </div>

      <!-- Traceability Chain -->
      <div class="traceability-section">
        <h4>Traceability Chain</h4>
        <div class="chain">
          <div class="chain-link">
            <span class="chain-label">RFQ / Enquiry</span>
            <span class="chain-value">{selectedOrder.customer_reference || selectedOrder.rfq_id || 'N/A'}</span>
          </div>
          <span class="chain-arrow">></span>
          <div class="chain-link">
            <span class="chain-label">Offer</span>
            <span class="chain-value">{selectedOrder.offer_number || 'N/A'}</span>
          </div>
          <span class="chain-arrow">></span>
          <div class="chain-link active">
            <span class="chain-label">Order</span>
            <span class="chain-value">{selectedOrder.order_number}</span>
          </div>
        </div>
      </div>

      <!-- Delivery Progress -->
      {#if orderItems.length > 0}
        <div class="delivery-section">
          <h4>Delivery Progress</h4>
          <div class="progress-bar-container">
            <div class="progress-bar">
              <div
                class="progress-fill"
                style="width: {deliveryProgress.percentage}%"
></div>
            </div>
            <span class="progress-label">
              {deliveryProgress.delivered} / {deliveryProgress.total} items delivered ({deliveryProgress.percentage}%)
            </span>
          </div>
        </div>
      {/if}

      <!-- Order Items -->
      {#if orderItems.length > 0}
        <div class="items-section">
          <h4>Order Items</h4>
          <table class="items-table">
            <thead>
              <tr>
                <th>Line</th>
                <th>Product Code</th>
                <th>Description</th>
                <th class="number-col">Quantity</th>
                <th class="number-col">Shipped</th>
                <th class="number-col">Invoiced</th>
                <th class="number-col">Unit Price</th>
                <th class="number-col">Total</th>
              </tr>
            </thead>
            <tbody>
              {#each orderItems as item}
                <tr>
                  <td>{item.line_number}</td>
                  <td>{item.product_code}</td>
                  <td>{item.description}</td>
                  <td class="number-col">{item.quantity}</td>
                  <td class="number-col">{item.quantity_shipped || 0}</td>
                  <td class="number-col">{item.quantity_invoiced || 0}</td>
                  <td class="number-col">{formatCurrency(item.unit_price_bhd)}</td>
                  <td class="number-col">{formatCurrency(item.total_price || (item.quantity * item.unit_price_bhd))}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      <!-- B3: Linked Invoices (invoices generated from this order) -->
      <div class="invoices-section">
        <h4>Linked Invoices</h4>
        {#if loadingInvoices}
          <p class="invoices-empty">Loading invoices…</p>
        {:else if linkedInvoices.length > 0}
          <table class="items-table">
            <thead>
              <tr>
                <th>Invoice #</th>
                <th>Date</th>
                <th>Status</th>
                <th class="number-col">Total</th>
              </tr>
            </thead>
            <tbody>
              {#each linkedInvoices as invoice}
                <tr>
                  <td>{invoice.invoice_number}</td>
                  <td>{formatDate(invoice.invoice_date)}</td>
                  <td><StatusBadge status={invoice.status} /></td>
                  <td class="number-col">{formatCurrency(invoice.grand_total_bhd || 0)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <p class="invoices-empty">No invoices yet</p>
        {/if}
      </div>

      <!-- Quick Actions (conditional on order state) -->
      <div class="quick-actions">
        {#if selectedOrder.status !== 'Cancelled' && (!fulfillmentStatus || fulfillmentStatus?.fulfillment_pct < 1.0)}
          <span title={hasNoItemsSelected ? 'This order has no saved line items. Edit the order to add items before creating a delivery note.' : undefined}>
            <Button variant="secondary" on:click={() => handleCreateDeliveryNote(selectedOrder)} disabled={isProcessingAction || hasNoItemsSelected}>
              Create Delivery Note
            </Button>
          </span>
          <span title={hasNoItemsSelected ? 'This order has no saved line items. Edit the order to add items before creating a supplier order.' : undefined}>
            <Button variant="secondary" on:click={() => handleCreatePurchaseOrder(selectedOrder)} disabled={isProcessingAction || hasNoItemsSelected}>
              Create Supplier Order
            </Button>
          </span>
        {/if}
        {#if selectedOrder.status !== 'Cancelled' && selectedOrder.status !== 'FullyDelivered'}
          <Button variant="primary" on:click={handleQuickMarkDelivered} disabled={isProcessingAction}>
            Mark as Delivered
          </Button>
        {/if}
        {#if selectedOrder.status !== 'Cancelled' && (!fulfillmentStatus || fulfillmentStatus?.invoicing_pct < 1.0)}
          <Button variant="secondary" on:click={() => handleCreateInvoice(selectedOrder)} disabled={isProcessingAction}>
            Create Invoice
          </Button>
          <Button variant="ghost" on:click={() => handleCreateProforma(selectedOrder)} disabled={isProcessingAction}>
            Proforma
          </Button>
        {/if}
        <Button variant="secondary" on:click={() => showTaskModal = true} disabled={isProcessingAction}>
          Create Task
        </Button>
        <Button variant="ghost" on:click={() => handleStartProject(selectedOrder)} disabled={isProcessingAction}>
          Start Project
        </Button>
        <Button variant="ghost" on:click={() => void openEditModal(selectedOrder)} disabled={isProcessingAction || isDeleting}>
          Edit Order
        </Button>
        <Button variant="ghost" on:click={() => handleDeleteOrder(selectedOrder)} disabled={isDeleting || isProcessingAction} style="color: #EF4444;">
          Delete Order
        </Button>
      </div>
    </div>
  </Modal>
{/if}

{#if confirmDialogOpen}
  <Modal
    title={confirmDialogTitle}
    open={confirmDialogOpen}
    on:close={() => resolveActionConfirmation(false)}
    size="sm"
  >
    <div class="confirm-dialog">
      <p>{confirmDialogMessage}</p>
      {#if confirmDialogLines.length > 0}
        <ul class="confirm-dialog-lines">
          {#each confirmDialogLines as line}
            <li>{line}</li>
          {/each}
        </ul>
      {/if}
      <div class="confirm-actions">
        <Button variant="secondary" on:click={() => resolveActionConfirmation(false)}>
          {confirmDialogBlocked ? 'Close' : 'Cancel'}
        </Button>
        {#if !confirmDialogBlocked}
          <Button variant={confirmDialogVariant} on:click={() => resolveActionConfirmation(true)}>
            {confirmDialogConfirmLabel}
          </Button>
        {/if}
      </div>
    </div>
  </Modal>
{/if}

{#if selectedOrder}
  <ContextTaskModal
    open={showTaskModal}
    title="Create Order Task"
    subtitle={`Link work to order ${selectedOrder.order_number}`}
    defaults={{
      customer_id: selectedOrder.customer_id,
      order_id: selectedOrder.id,
      seed_title: `Order task: ${selectedOrder.order_number}`,
    }}
    on:close={() => showTaskModal = false}
    on:created={() => showTaskModal = false}
  />
{/if}

<style>
  /* Embedded Mode */
  .orders-embedded {
    padding: 16px;
    max-height: calc(100vh - 200px);
    overflow-y: auto;
  }

  .embedded-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }

  .orders-embedded h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .embedded-stats {
    display: flex;
    gap: 16px;
  }

  .embedded-stat {
    font-size: 11px;
    color: var(--text-secondary);
    font-family: var(--font-mono, monospace);
  }

  .embedded-filters {
    display: flex;
    gap: 8px;
    margin-bottom: 16px;
  }

  .embedded-search {
    flex: 1;
    padding: 6px 10px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md, 6px);
    font-size: 12px;
    outline: none;
  }

  .embedded-search:focus {
    border-color: var(--primary, #4F46E5);
  }

  .embedded-select {
    padding: 6px 10px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md, 6px);
    font-size: 12px;
    background: transparent;
    outline: none;
    cursor: pointer;
    min-width: 120px;
  }

  /* Year Groups */
  .year-group {
    margin-bottom: 20px;
  }

  .year-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 0;
    border-bottom: 2px solid var(--text-primary, #1a1a1a);
    margin-bottom: 8px;
  }

  .year-label {
    font-size: 18px;
    font-weight: 700;
    color: var(--text-primary);
    font-family: var(--font-mono, monospace);
  }

  .year-count {
    font-size: 11px;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .year-value {
    margin-left: auto;
    font-size: 13px;
    font-weight: 600;
    font-family: var(--font-mono, monospace);
    color: var(--text-primary);
  }

  .orders-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .order-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 12px;
    background: var(--surface, #fafafa);
    border: 1px solid var(--border, #e5e5e5);
    border-radius: var(--radius-md, 6px);
    cursor: pointer;
    transition: all 0.15s;
  }

  .order-item:hover {
    background: var(--bg-hover, #f0f0f0);
    border-color: var(--text-muted, #999);
  }

  .order-left {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
  }

  .order-number {
    font-size: 12px;
    font-weight: 600;
    font-family: var(--font-mono, monospace);
    color: var(--text-primary);
    white-space: nowrap;
  }

  .order-customer {
    font-size: 12px;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 200px;
  }

  .order-ref {
    font-size: 10px;
    color: var(--text-muted, #999);
    font-family: var(--font-mono, monospace);
    white-space: nowrap;
  }

  .order-right {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-shrink: 0;
  }

  .order-value {
    font-size: 12px;
    font-family: var(--font-mono, monospace);
    font-weight: 600;
    color: var(--text-primary);
    min-width: 100px;
    text-align: right;
  }

  .order-date {
    font-size: 11px;
    color: var(--text-muted, #999);
    min-width: 80px;
    text-align: right;
  }

  /* Full Screen Mode */
  .header-actions {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .stats-row {
    display: flex;
    gap: 24px;
    margin-right: auto;
  }

  .stat-item {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
  }

  .stat-value {
    font-size: 20px;
    font-weight: 600;
    color: var(--text-primary);
    line-height: 1.2;
  }

  .stat-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    margin-top: 2px;
  }

  /* Controls Bar */
  .controls-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    gap: 16px;
    flex-wrap: wrap;
  }

  .filter-tabs {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .filter-tab {
    padding: 8px 16px;
    border: 1px solid var(--border);
    background: transparent;
    border-radius: var(--radius-md);
    font-size: 13px;
    cursor: pointer;
    transition: all 0.2s;
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .filter-tab:hover {
    background: var(--bg-subtle);
    color: var(--text-primary);
  }

  .filter-tab.active {
    background: var(--primary);
    color: white;
    border-color: var(--primary);
  }

  .dropdown-filters {
    display: flex;
    gap: 8px;
  }

  .filter-select {
    padding: 8px 12px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    font-size: 13px;
    background: transparent;
    color: var(--text-primary);
    outline: none;
    cursor: pointer;
    min-width: 140px;
    transition: border-color 0.2s;
  }

  .filter-select:focus {
    border-color: var(--primary);
  }

  .search-box {
    flex-shrink: 0;
  }

  .search-input {
    padding: 8px 12px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    font-size: 13px;
    width: 280px;
    outline: none;
    transition: border-color 0.2s;
  }

  .search-input:focus {
    border-color: var(--primary);
  }

  /* Loading */
  .loading-container {
    display: flex;
    justify-content: center;
    padding: 48px;
  }

  .empty-message {
    text-align: center;
    padding: 24px;
    color: var(--text-secondary);
    font-size: 13px;
  }

  /* Custom cells */
  .delivery-progress {
    font-size: 12px;
    font-family: var(--font-mono);
    color: var(--text-secondary);
  }

  .ref-value {
    font-size: 12px;
    font-family: var(--font-mono);
    color: var(--text-secondary);
  }

  /* Form */
  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .form-input,
  .form-select {
    width: 100%;
    padding: 8px 12px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    font-size: 13px;
    outline: none;
    transition: border-color 0.2s;
    font-family: var(--font-sans);
  }

  .form-input:focus,
  .form-select:focus {
    border-color: var(--primary);
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 24px;
    padding-top: 16px;
    border-top: 1px solid var(--border);
  }

  .line-items-panel {
    margin-top: 20px;
    padding: 16px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
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

  /* Wave 9.6 B2: .line-items-list / .line-item-editor(+head) /
     .line-item-top-row / .line-item-bottom-row / .line-field(+span) /
     .compact / .number-input / .line-total-cell(+strong) /
     .line-remove-btn(+disabled) now live in LineItemsEditor.svelte
     alongside the markup that used them. */

  @media (max-width: 840px) {
    .form-row {
      grid-template-columns: 1fr;
    }
  }

  .line-item-alert {
    padding: 10px 12px;
    background: rgba(245, 158, 11, 0.1);
    border: 1px solid rgba(245, 158, 11, 0.28);
    border-radius: var(--radius-sm);
    color: #92400e;
    font-size: 12px;
    line-height: 1.5;
  }

  /* Order Detail */
  .order-detail {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .detail-header {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border);
  }

  .detail-section h4 {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    margin: 0 0 4px 0;
  }

  .detail-section p {
    font-size: 14px;
    color: var(--text-primary);
    margin: 0;
  }

  .value-highlight {
    font-size: 16px !important;
    font-weight: 600;
    color: var(--primary);
  }

  /* Traceability Chain */
  .traceability-section h4 {
    font-size: 13px;
    font-weight: 600;
    margin: 0 0 12px 0;
    color: var(--text-primary);
  }

  .chain {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: var(--bg-subtle);
    border-radius: var(--radius-md);
  }

  .chain-link {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 8px 16px;
    background: white;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    flex: 1;
  }

  .chain-link.active {
    border-color: var(--primary);
    background: var(--primary-subtle);
  }

  .chain-label {
    font-size: 9px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .chain-value {
    font-size: 12px;
    font-family: var(--font-mono);
    font-weight: 600;
    color: var(--text-primary);
  }

  .chain-arrow {
    color: var(--text-secondary);
    font-size: 16px;
  }

  /* Delivery Progress */
  .delivery-section h4 {
    font-size: 13px;
    font-weight: 600;
    margin: 0 0 12px 0;
    color: var(--text-primary);
  }

  .progress-bar-container {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .progress-bar {
    height: 8px;
    background: var(--bg-subtle);
    border-radius: 4px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--primary), var(--primary-light));
    transition: width 0.3s ease;
  }

  .progress-label {
    font-size: 12px;
    color: var(--text-secondary);
  }

  /* Items Table */
  .items-section h4 {
    font-size: 13px;
    font-weight: 600;
    margin: 0 0 12px 0;
    color: var(--text-primary);
  }

  /* Linked Invoices */
  .invoices-section h4 {
    font-size: 13px;
    font-weight: 600;
    margin: 0 0 12px 0;
    color: var(--text-primary);
  }

  .invoices-empty {
    font-size: 13px;
    color: var(--text-secondary);
    margin: 0;
  }

  .items-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }

  .items-table thead {
    background: var(--bg-subtle);
  }

  .items-table th {
    padding: 8px;
    text-align: left;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    border-bottom: 1px solid var(--border);
  }

  .items-table td {
    padding: 8px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .items-table .number-col {
    text-align: right;
    font-family: var(--font-mono);
  }

  .items-table tbody tr:hover {
    background: var(--bg-subtle);
  }

  /* Quick Actions */
  .quick-actions {
    display: flex;
    gap: 8px;
    padding-top: 16px;
    border-top: 1px solid var(--border);
    flex-wrap: wrap;
  }

  .confirm-dialog {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }

  .confirm-dialog p {
    margin: 0;
    color: var(--text-primary);
    font-size: 14px;
    line-height: 1.5;
  }

  .confirm-dialog-lines {
    margin: 0;
    padding-left: 18px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 13px;
    color: var(--text-secondary);
  }

  .confirm-dialog-lines li {
    line-height: 1.4;
  }

  .confirm-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  /* Warning banner for orders with no line items */
  .no-items-warning {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 12px 16px;
    background: #fef9e7;
    border: 1px solid #f0d050;
    border-radius: 8px;
    margin-bottom: 12px;
    font-size: 13px;
    color: #7a6200;
  }

  .no-items-warning-critical {
    background: rgba(239, 68, 68, 0.08);
    border-color: rgba(239, 68, 68, 0.24);
    color: #991b1b;
  }

  .no-items-warning .warning-icon {
    font-size: 18px;
    flex-shrink: 0;
    line-height: 1.4;
  }

  .no-items-warning .warning-content {
    flex: 1;
    line-height: 1.5;
  }

  .no-items-warning .warning-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 6px;
  }

  .no-items-warning .warning-order-number {
    display: inline-block;
    padding: 2px 8px;
    background: rgba(240, 208, 80, 0.3);
    border-radius: 4px;
    font-weight: 600;
    font-size: 12px;
    cursor: default;
  }

  .detail-alert {
    margin-top: -4px;
  }

  /* B10-2: Load-more pagination controls (mirrors PaymentsScreen/InvoicesScreen) */
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

</style>
