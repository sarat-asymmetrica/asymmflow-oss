<script lang="ts">
  import { run, preventDefault } from 'svelte/legacy';

  /**
   * GRNScreen - Production-Ready Goods Received Notes Management
   *
   * Features:
   * - View all GRNs with filtering by QC status
   * - Create new GRNs by receiving against a Purchase Order (there is no
   *   PO-less/manual creation path — backend createGRN hard-requires a
   *   valid PO, so "Receive Against PO" is the only creation UI)
   * - QC Status tracking: Pending → Passed / Failed / Partial
   * - Records quantity received, accepted, rejected
   * - Links to parent Purchase Orders
   * - Quality control workflow with notes
   * - Acceptance rate tracking
   *
   * Design System: Wabi-Sabi minimalism × Bloomberg data density
   */

  import { onMount, onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';

  // Wails API imports
  import {
    ListGRNs } from '../../../wailsjs/go/main/App';
import { GetGRN, UpdateGRN, DeleteGRN, UpdateGRNQCStatus, CompleteGRN, GetPurchaseOrders, GetPurchaseOrderByID, ReceiveAgainstPO, ReceiveAgainstPOWithSerials } from '../../../wailsjs/go/main/CRMService';

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
  import { currentUser } from '$lib/stores/authContext';
  import { escapeHtml } from '$lib/utils/escapeHtml';

  
  interface Props {
    // Props
    embedded?: boolean;
  }

  let { embedded = false }: Props = $props();

  // Types
  type QCStatus = 'All' | 'Pending' | 'Passed' | 'Failed' | 'Partial';

  interface GRNDisplay {
    id: string;
    grn_number: string;
    purchase_order_id: string;
    po_number: string;
    supplier_name: string;
    received_date: string;
    received_by: string;
    qc_status: string;
    qc_date?: string;
    qc_by?: string;
    qc_notes?: string;
    items_count: number;
    total_received: number;
    total_accepted: number;
    total_rejected: number;
    acceptance_rate: number;
    // B6: server-resolved "already posted" signal (grn_service.go
    // grnHasPostedMovement) — the true completion state, not derivable from
    // qc_status alone. Drives the Complete action gate below.
    is_completed?: boolean;
    items?: GRNItem[];
    created_at?: string;
    updated_at?: string;
  }

  interface GRNItem {
    id?: string;
    grn_id?: string;
    po_item_id: string;
    product_id?: string;
    product_code?: string;
    description?: string;
    quantity_ordered: number;
    quantity_received: number;
    quantity_rejected: number;
    previously_received: number; // Already received in prior GRNs
    remaining: number; // What's left to receive
    fully_received: boolean; // True if nothing left to receive
    rejection_reason?: string;
    notes?: string;
    // Phase 23: Serial tracking
    requires_serial?: boolean;
    serial_numbers?: string[];
    serial_input?: string; // Raw textarea input
  }

  // State - using 'any' for flexibility with backend responses
  let grns: any[] = $state([]);
  let filteredGRNs: any[] = $state([]);
  let loading = $state(true);
  let selectedQCStatus: QCStatus = $state('All');

  // Modal state
  let showCreateModal = false;
  let showReceiveModal = $state(false);
  let showViewModal = $state(false);
  let showQCModal = $state(false);
  let viewingGRN: any = $state(null);
  let qcGRN: any = $state(null);

  // Reference data
  let purchaseOrders: any[] = [];
  let availablePOs: any[] = $state([]); // POs that can have GRNs created

  // Form state - Receive against PO
  let receiveFormData = $state({
    po_id: '',
    items: [] as GRNItem[]
  });

  // QC Form state
  // Wave 9.3 B2: qc_by is server-resolved to the authenticated operator
  // (Article III.4) — never a hardcoded "System User" ghost actor.
  let qcFormData = $state({
    status: 'Pending' as QCStatus,
    notes: '',
    qc_by: ''
  });

  let formLoading = $state(false);

  // B6: Complete is legal exactly when QC has been resolved (not 'Pending'),
  // QC did not fail (backend CompleteGRN hard-blocks 'Failed' — QC_FAILED),
  // and the GRN hasn't already been applied (is_completed, server-resolved
  // from the posted-movement ledger — see grn_service.go
  // grnHasPostedMovement). Once true, the button disappears rather than
  // re-offering a no-op click.
  function canCompleteGRN(row: GRNDisplay): boolean {
    return row.qc_status !== 'Pending' && row.qc_status !== 'Failed' && !row.is_completed;
  }

  // DataTable columns configuration
  const columns = [
    {
      key: 'grn_number',
      label: 'GRN #',
      sortable: true,
      width: '140px',
      render: (row: GRNDisplay) => {
        return `<span style="font-family: var(--font-mono); font-weight: 600; color: var(--brand-indigo);">${escapeHtml(row.grn_number || '')}</span>`;
      }
    },
    {
      key: 'po_number',
      label: 'PO Reference',
      sortable: true,
      width: '140px',
      render: (row: GRNDisplay) => {
        return `<span style="font-family: var(--font-mono); font-size: 12px;">${escapeHtml(row.po_number || '')}</span>`;
      }
    },
    {
      key: 'supplier_name',
      label: 'Supplier',
      sortable: true,
      render: (row: GRNDisplay) => escapeHtml(row.supplier_name)
    },
    {
      key: 'received_date',
      label: 'Received Date',
      type: 'date' as const,
      sortable: true,
      width: '130px'
    },
    {
      key: 'qc_status',
      label: 'QC Status',
      type: 'status' as const,
      sortable: true,
      width: '120px'
    },
    {
      key: 'items_count',
      label: 'Items',
      sortable: true,
      width: '80px',
      align: 'center' as const,
      render: (row: GRNDisplay) => {
        return `<span style="font-weight: 600;">${Number(row.items_count) || 0}</span>`;
      }
    },
    {
      key: 'acceptance_rate',
      label: 'Acceptance',
      sortable: true,
      width: '110px',
      align: 'right' as const,
      render: (row: GRNDisplay) => {
        const rate = row.acceptance_rate * 100;
        const color = rate >= 95 ? '#10B981' : rate >= 80 ? '#F59E0B' : '#EF4444';
        return `<span style="font-weight: 600; color: ${color};">${rate.toFixed(1)}%</span>`;
      }
    },
    {
      key: 'actions',
      label: 'Actions',
      type: 'actions' as const,
      width: '220px',
      render: (row: GRNDisplay) => {
        return `
          <div style="display: flex; gap: 8px; justify-content: flex-end;">
            <button
              class="action-btn action-btn-view"
              data-action="view"
              data-id="${row.id}"
              aria-label="View GRN details"
            >
              View
            </button>
            ${row.qc_status === 'Pending' ? `
              <button
                class="action-btn action-btn-qc"
                data-action="qc"
                data-id="${row.id}"
                aria-label="QC Review"
              >
                QC Review
              </button>
            ` : ''}
            ${canCompleteGRN(row) ? `
              <button
                class="action-btn action-btn-complete"
                data-action="complete"
                data-id="${row.id}"
                aria-label="Complete GRN"
              >
                Complete
              </button>
            ` : ''}
          </div>
        `;
      }
    }
  ];

  // Status filter tabs
  const statusTabs: { value: QCStatus; label: string; count: number }[] = $state([
    { value: 'All', label: 'All GRNs', count: 0 },
    { value: 'Pending', label: 'Pending QC', count: 0 },
    { value: 'Passed', label: 'Passed', count: 0 },
    { value: 'Failed', label: 'Failed', count: 0 },
    { value: 'Partial', label: 'Partial', count: 0 }
  ]);

  // Computed: Update tab counts
  run(() => {
    statusTabs[0].count = grns.length;
    statusTabs[1].count = grns.filter(g => g.qc_status === 'Pending').length;
    statusTabs[2].count = grns.filter(g => g.qc_status === 'Passed').length;
    statusTabs[3].count = grns.filter(g => g.qc_status === 'Failed').length;
    statusTabs[4].count = grns.filter(g => g.qc_status === 'Partial').length;
  });

  // Computed: Filter GRNs by selected QC status
  run(() => {
    if (selectedQCStatus === 'All') {
      filteredGRNs = grns;
    } else {
      filteredGRNs = grns.filter(g => g.qc_status === selectedQCStatus);
    }
  });

  // Load GRNs and reference data
  async function loadGRNs() {
    loading = true;
    try {
      const [grnsData, posData] = await Promise.all([
        ListGRNs(1000, 0, selectedQCStatus === 'All' ? '' : selectedQCStatus),
        GetPurchaseOrders()
      ]);

      grns = grnsData || [];
      purchaseOrders = posData || [];

      // Filter POs that can have GRNs created (Draft, Sent, Acknowledged, PartiallyReceived)
      availablePOs = purchaseOrders.filter(po =>
        ['Draft', 'Sent', 'Acknowledged', 'PartiallyReceived'].includes(po.status)
      );

      console.log(`Loaded ${grns.length} GRNs, ${availablePOs.length} available POs`);
    } catch (err) {
      console.error('Failed to load GRNs:', err);
      toast.danger('Failed to load GRNs');
      grns = [];
    } finally {
      loading = false;
    }
  }

  // Open receive against PO modal
  function openReceiveModal() {
    if (availablePOs.length === 0) {
      toast.warning('No purchase orders available for receiving');
      return;
    }

    receiveFormData = {
      po_id: '',
      items: []
    };
    showReceiveModal = true;
  }

  // Map PO items to GRN receive form items with partial receiving context
  function mapPOItemsToReceiveItems(poItems: any[]): GRNItem[] {
    return poItems.map((item: any) => {
      const ordered = item.quantity || 0;
      const prevReceived = item.quantity_received || 0;
      const remaining = Math.max(0, ordered - prevReceived);
      return {
        po_item_id: item.id,
        product_id: item.product_id,
        product_code: item.product_code,
        description: item.description,
        quantity_ordered: ordered,
        previously_received: prevReceived,
        remaining: remaining,
        fully_received: remaining === 0,
        quantity_received: remaining, // Default to remaining qty
        quantity_rejected: 0,
        rejection_reason: '',
        notes: '',
        requires_serial: item.requires_serial_tracking || false,
        serial_numbers: [],
        serial_input: ''
      };
    });
  }

  // Handle PO selection for receiving
  async function handlePOSelect(e: Event) {
    const select = e.target as HTMLSelectElement;
    const poId = select.value;
    receiveFormData.po_id = poId;

    if (!poId) {
      receiveFormData.items = [];
      return;
    }

    try {
      const po = await GetPurchaseOrderByID(poId);
      receiveFormData.items = mapPOItemsToReceiveItems(po.items || []);
      console.log(`Loaded ${receiveFormData.items.length} items from PO ${po.po_number}`);
    } catch (err) {
      console.error('Failed to load PO items:', err);
      toast.danger('Failed to load PO items');
    }
  }

  // Handle receive against PO
  async function handleReceiveAgainstPO() {
    if (!receiveFormData.po_id) {
      toast.warning('Please select a purchase order');
      return;
    }

    // Wave 9.3 B2: block instead of letting the backend fall back to a
    // ghost "System" identity when we don't know who is receiving.
    if (!$currentUser?.id) {
      toast.danger('Cannot receive goods: no authenticated user found. Please sign in again.');
      return;
    }

    // Filter to items that are actually receiving something
    const itemsToReceive = receiveFormData.items.filter(
      item => !item.fully_received && item.quantity_received > 0
    );

    if (itemsToReceive.length === 0) {
      toast.warning('No quantities entered. Please enter receiving quantities for at least one item.');
      return;
    }

    // Phase 23: Parse serial numbers from textarea inputs
    const hasSerials = itemsToReceive.some(i => i.requires_serial && i.serial_input?.trim());
    if (hasSerials) {
      for (const item of itemsToReceive) {
        if (item.requires_serial && item.serial_input?.trim()) {
          item.serial_numbers = item.serial_input.trim().split('\n').map(s => s.trim()).filter(s => s.length > 0);
          if (item.serial_numbers.length !== item.quantity_received) {
            toast.warning(`Serial count (${item.serial_numbers.length}) must match receiving qty (${item.quantity_received}) for ${item.product_code}`);
            return;
          }
        } else {
          item.serial_numbers = [];
        }
      }
    }

    formLoading = true;
    try {
      if (hasSerials) {
        await ReceiveAgainstPOWithSerials(receiveFormData.po_id, itemsToReceive as any);
      } else {
        await ReceiveAgainstPO(receiveFormData.po_id, itemsToReceive as any);
      }
      const totalQty = itemsToReceive.reduce((sum, i) => sum + i.quantity_received, 0);
      toast.success(`GRN created: ${totalQty} units received across ${itemsToReceive.length} items`);
      showReceiveModal = false;
      await loadGRNs();
    } catch (err) {
      console.error('Failed to create GRN:', err);
      toast.danger('Failed to create GRN: ' + (err as Error).message);
    } finally {
      formLoading = false;
    }
  }

  // Open view modal
  async function openViewModal(grnId: string) {
    try {
      const grn = await GetGRN(grnId);
      viewingGRN = grn;
      showViewModal = true;
    } catch (err) {
      console.error('Failed to load GRN for viewing:', err);
      toast.danger('Failed to load GRN');
    }
  }

  // Open QC modal
  async function openQCModal(grnId: string) {
    try {
      const grn = await GetGRN(grnId);
      qcGRN = grn;
      qcFormData = {
        status: grn.qc_status as QCStatus,
        notes: grn.qc_notes || '',
        qc_by: $currentUser?.id || ''
      };
      showQCModal = true;
    } catch (err) {
      console.error('Failed to load GRN for QC:', err);
      toast.danger('Failed to load GRN');
    }
  }

  // Handle QC status update
  // Wave 9.3 B2: attribution to the authenticated user (Article III.4). No
  // fallback to a hardcoded "System User" string: if we don't know who is
  // reviewing, block the action instead of sending a fake identity. The
  // backend re-resolves qc_by server-side regardless (defense in depth).
  async function handleQCUpdate() {
    if (!qcGRN) return;

    const qcById = $currentUser?.id;
    if (!qcById) {
      toast.danger('Cannot record QC review: no authenticated user found. Please sign in again.');
      return;
    }

    formLoading = true;
    try {
      await UpdateGRNQCStatus(
        qcGRN.id,
        qcFormData.status,
        qcFormData.notes,
        qcById
      );
      toast.success(`GRN QC status updated to ${qcFormData.status}`);
      showQCModal = false;
      qcGRN = null;
      await loadGRNs();
    } catch (err) {
      console.error('Failed to update QC status:', err);
      toast.danger('Failed to update QC status: ' + (err as Error).message);
    } finally {
      formLoading = false;
    }
  }

  // Handle complete GRN
  let completingGRN = false;
  async function handleCompleteGRN(grnId: string) {
    if (completingGRN) return;
    if (!(await confirm.ask({
      title: 'Complete GRN',
      message: 'Complete this GRN? This will update PO quantities.',
      confirmLabel: 'Complete',
      variant: 'warning'
    }))) {
      return;
    }

    completingGRN = true;
    try {
      // B6: CompleteGRN resolves successfully even when it no-ops (the
      // backend idempotency guard silently skips an already-applied GRN —
      // see grn_service.go CompleteGRN / grnHasPostedMovement). Capture the
      // pre-call completion state so we can tell a real completion apart
      // from a no-op and avoid a false "completed successfully" toast.
      const before = grns.find((g) => g.id === grnId);
      const wasCompleted = !!before?.is_completed;

      await CompleteGRN(grnId);
      await loadGRNs();

      const after = grns.find((g) => g.id === grnId);
      if (!wasCompleted && after?.is_completed) {
        toast.success('GRN completed successfully');
      } else if (wasCompleted) {
        // Should be unreachable — the Complete action is hidden once a GRN
        // is already completed (canCompleteGRN gate) — but guard defensively
        // against a race with another session completing it first.
        toast.warning('This GRN was already completed. No changes were made.');
      } else {
        toast.warning('GRN completion did not apply any changes. Please refresh and try again.');
      }
    } catch (err) {
      console.error('Failed to complete GRN:', err);
      toast.danger('Failed to complete GRN: ' + (err as Error).message);
    } finally {
      completingGRN = false;
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
      case 'qc':
        openQCModal(id!);
        break;
      case 'complete':
        handleCompleteGRN(id!);
        break;
    }
  }

  // Update item quantity received
  // (c) FIX: reject-qty/rejection-reason capture was removed from this
  // retired screen's receive form (see below) because the values were never
  // persisted by handleReceiveAgainstPO — a stub that lied. This helper now
  // only handles quantity_received. Discrepancies are captured via
  // RaiseGRNDiscrepancy, which records a SupplierIssue (the discrepancy
  // record) surfaced on the Supplier detail "Issues" tab.
  function updateItemQuantity(index: number, value: number) {
    const item = receiveFormData.items[index];
    // Cap at remaining (not ordered - prevents exceeding what's left)
    item.quantity_received = Math.max(0, Math.min(value, item.remaining));
    receiveFormData.items = [...receiveFormData.items]; // Trigger reactivity
  }

  // Computed: Total statistics
  let totalReceived = $derived(grns.reduce((sum, g) => sum + g.total_received, 0));
  let totalAccepted = $derived(grns.reduce((sum, g) => sum + g.total_accepted, 0));
  let totalRejected = $derived(grns.reduce((sum, g) => sum + g.total_rejected, 0));
  let overallAcceptanceRate = $derived(totalReceived > 0 ? (totalAccepted / totalReceived) * 100 : 0);

  // Event handler for opening create/receive modal from parent hub
  async function handleOpenCreateGRN(e: any) {
    openReceiveModal();
    // If a PO ID was passed, pre-select it
    if (e?.detail?.poId) {
      receiveFormData.po_id = e.detail.poId;
      // Trigger the PO selection to load items
      await handlePOSelectById(e.detail.poId);
    }
  }

  // Helper to programmatically select a PO by ID
  async function handlePOSelectById(poId: string) {
    receiveFormData.po_id = poId;

    if (!poId) {
      receiveFormData.items = [];
      return;
    }

    try {
      const po = await GetPurchaseOrderByID(poId);
      receiveFormData.items = mapPOItemsToReceiveItems(po.items || []);
      console.log(`Pre-selected PO ${po.po_number} with ${receiveFormData.items.length} items`);
    } catch (err) {
      console.error('Failed to load PO items:', err);
      toast.danger('Failed to load PO items');
    }
  }

  onMount(() => {
    loadGRNs();

    // Listen for create GRN events from OperationsHub header button
    window.addEventListener('openCreateGRN', handleOpenCreateGRN);
  });

  onDestroy(() => {
    // Clean up event listener
    window.removeEventListener('openCreateGRN', handleOpenCreateGRN);
  });
</script>

<PageLayout title="Goods Received Notes" subtitle="Warehouse Receiving & QC" {embedded}>
  <!-- @migration-task: migrate this slot by hand, `header-actions` is an invalid identifier -->
  <svelte:fragment slot="header-actions">
    <Button variant="secondary" on:click={openReceiveModal}>
      + Receive from PO
    </Button>
  </svelte:fragment>

  <div class="grn-container">
    <!-- QC Status Filter Tabs -->
    <Card padding="sm">
      <div class="status-tabs" role="tablist" aria-label="Filter GRNs by QC status">
        {#each statusTabs as tab}
          <button
            class="status-tab"
            class:active={selectedQCStatus === tab.value}
            role="tab"
            aria-selected={selectedQCStatus === tab.value}
            onclick={() => selectedQCStatus = tab.value}
          >
            {tab.label}
            <span class="tab-count">{tab.count}</span>
          </button>
        {/each}
      </div>
    </Card>

    <!-- GRNs DataTable -->
    <Card padding="sm">
      <DataTable
        {columns}
        data={filteredGRNs}
        {loading}
        emptyMessage="No goods received notes found"
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
          <div class="stat-label">Total GRNs</div>
          <div class="stat-value">{grns.length}</div>
        </div>
      </Card>

      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Items Received</div>
          <div class="stat-value">{totalReceived.toFixed(0)}</div>
        </div>
      </Card>

      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Items Accepted</div>
          <div class="stat-value stat-success">{totalAccepted.toFixed(0)}</div>
        </div>
      </Card>

      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Items Rejected</div>
          <div class="stat-value stat-danger">{totalRejected.toFixed(0)}</div>
        </div>
      </Card>

      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Acceptance Rate</div>
          <div class="stat-value" class:stat-success={overallAcceptanceRate >= 95} class:stat-warning={overallAcceptanceRate < 95 && overallAcceptanceRate >= 80} class:stat-danger={overallAcceptanceRate < 80}>
            {overallAcceptanceRate.toFixed(1)}%
          </div>
        </div>
      </Card>

      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Pending QC</div>
          <div class="stat-value stat-warning">
            {grns.filter(g => g.qc_status === 'Pending').length}
          </div>
        </div>
      </Card>
    </div>
  </div>
</PageLayout>

<!-- Receive Against PO Modal -->
<WabiModal bind:open={showReceiveModal} title="Receive Goods from PO" size="xl">
  <form onsubmit={preventDefault(handleReceiveAgainstPO)} class="receive-form">
    <!-- PO Selection -->
    <FormGroup label="Purchase Order" required>
      <select
        class="select-input"
        bind:value={receiveFormData.po_id}
        onchange={handlePOSelect}
        required
      >
        <option value="">Select purchase order...</option>
        {#each availablePOs as po}
          <option value={po.id}>{po.po_number} - {po.supplier_name || 'Unknown Supplier'}{po.status === 'PartiallyReceived' ? ' (Partial)' : ''}</option>
        {/each}
      </select>
    </FormGroup>

    <!-- Wave 9.3 B2: received_by is server-resolved to the authenticated
         operator (Article III.4), not a hardcoded "System User" ghost actor. -->
    <div class="receiving-as">
      Receiving as: <strong>{$currentUser?.full_name || $currentUser?.username || 'Unknown — please sign in again'}</strong>
    </div>

    <!-- Items to Receive -->
    {#if receiveFormData.items.length > 0}
      <div class="items-section">
        <h4>Items to Receive</h4>

        <!-- Partial receiving info banner -->
        {#if receiveFormData.items.some(i => i.previously_received > 0)}
          <div class="partial-info-banner">
            This PO has previous deliveries. Previously received quantities are shown below.
          </div>
        {/if}

        <div class="items-table-wrapper">
          <table class="items-table">
            <thead>
              <tr>
                <th>Product</th>
                <th class="right">Ordered</th>
                <th class="right">Prev Rcvd</th>
                <th class="right">Remaining</th>
                <th class="right">Receiving Now</th>
              </tr>
            </thead>
            <tbody>
              {#each receiveFormData.items as item, index}
                <tr class:fully-received={item.fully_received}>
                  <td>
                    <div class="product-cell">
                      <div class="product-code">{item.product_code}</div>
                      <div class="product-desc">{item.description}</div>
                    </div>
                  </td>
                  <td class="right mono">{item.quantity_ordered.toFixed(0)}</td>
                  <td class="right mono prev-rcvd">
                    {#if item.previously_received > 0}
                      {item.previously_received.toFixed(0)}
                    {:else}
                      <span class="muted">-</span>
                    {/if}
                  </td>
                  <td class="right mono">
                    {#if item.fully_received}
                      <span class="fully-done">Done</span>
                    {:else}
                      {item.remaining.toFixed(0)}
                    {/if}
                  </td>
                  <td class="right">
                    {#if item.fully_received}
                      <span class="muted">-</span>
                    {:else}
                      <input
                        type="number"
                        class="qty-input"
                        bind:value={item.quantity_received}
                        oninput={(e) => updateItemQuantity(index, parseFloat((e.currentTarget).value) || 0)}
                        min="0"
                        max={item.remaining}
                        step="1"
                      />
                    {/if}
                  </td>
                </tr>
                {#if item.requires_serial && !item.fully_received}
                  <tr class="serial-row">
                    <td colspan="5" style="padding: 4px 8px 8px 24px;">
                      <div class="serial-input-section">
                        <div class="serial-label">
                          <span class="serial-badge">Serialized</span>
                          Enter serial numbers (one per line) — {item.quantity_received} required
                        </div>
                        <textarea
                          class="serial-textarea"
                          bind:value={item.serial_input}
                          placeholder={"SN-001\nSN-002\nSN-003"}
                          rows={Math.min(Math.max(item.quantity_received, 2), 6)}
                        ></textarea>
                        {#if item.serial_input?.trim()}
                          {@const count = item.serial_input.trim().split('\n').filter(s => s.trim()).length}
                          <div class="serial-count" class:serial-count-ok={count === item.quantity_received} class:serial-count-err={count !== item.quantity_received}>
                            {count} / {item.quantity_received} entered
                          </div>
                        {/if}
                      </div>
                    </td>
                  </tr>
                {/if}
              {/each}
            </tbody>
          </table>
        </div>

        <!-- Summary -->
        <div class="receive-summary">
          <div class="summary-row">
            <span>Total Ordered:</span>
            <span class="summary-value">{receiveFormData.items.reduce((sum, item) => sum + item.quantity_ordered, 0).toFixed(0)}</span>
          </div>
          {#if receiveFormData.items.some(i => i.previously_received > 0)}
            <div class="summary-row">
              <span>Previously Received:</span>
              <span class="summary-value prev-rcvd">{receiveFormData.items.reduce((sum, item) => sum + item.previously_received, 0).toFixed(0)}</span>
            </div>
            <div class="summary-row">
              <span>Remaining to Receive:</span>
              <span class="summary-value">{receiveFormData.items.reduce((sum, item) => sum + item.remaining, 0).toFixed(0)}</span>
            </div>
          {/if}
          <div class="summary-row summary-highlight">
            <span>Receiving Now:</span>
            <span class="summary-value">{receiveFormData.items.reduce((sum, item) => sum + item.quantity_received, 0).toFixed(0)}</span>
          </div>
        </div>
      </div>
    {/if}
  </form>

  {#snippet footer()}
  
      <Button variant="ghost" on:click={() => showReceiveModal = false}>
        Cancel
      </Button>
      <Button
        variant="primary"
        loading={formLoading}
        on:click={handleReceiveAgainstPO}
        disabled={!receiveFormData.po_id || receiveFormData.items.length === 0}
      >
        Create GRN
      </Button>
    
  {/snippet}
</WabiModal>

<!-- View GRN Modal -->
{#if viewingGRN}
  <WabiModal bind:open={showViewModal} title="GRN Details" size="lg">
    <div class="grn-details">
      <!-- Header Info -->
      <div class="details-grid">
        <div class="detail-item">
          <div class="detail-label">GRN Number</div>
          <div class="detail-value mono">{viewingGRN.grn_number}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">QC Status</div>
          <StatusBadge status={viewingGRN.qc_status} />
        </div>
        <div class="detail-item">
          <div class="detail-label">PO Reference</div>
          <div class="detail-value mono">{viewingGRN.po_number}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Supplier</div>
          <div class="detail-value">{viewingGRN.supplier_name}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Received Date</div>
          <div class="detail-value">{new Date(viewingGRN.received_date).toLocaleDateString()}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Received By</div>
          <div class="detail-value">{viewingGRN.received_by}</div>
        </div>
        {#if viewingGRN.qc_date}
          <div class="detail-item">
            <div class="detail-label">QC Date</div>
            <div class="detail-value">{new Date(viewingGRN.qc_date).toLocaleDateString()}</div>
          </div>
        {/if}
        {#if viewingGRN.qc_by}
          <div class="detail-item">
            <div class="detail-label">QC By</div>
            <div class="detail-value">{viewingGRN.qc_by}</div>
          </div>
        {/if}
      </div>

      <!-- QC Notes -->
      {#if viewingGRN.qc_notes}
        <div class="notes-section">
          <h4>QC Notes</h4>
          <div class="notes-content">{viewingGRN.qc_notes}</div>
        </div>
      {/if}

      <!-- Items Table -->
      {#if viewingGRN.items && viewingGRN.items.length > 0}
        <div class="items-section">
          <h4>Items ({viewingGRN.items.length})</h4>
          <table class="items-table">
            <thead>
              <tr>
                <th>Product</th>
                <th class="right">Ordered</th>
                <th class="right">Received</th>
                <th class="right">Rejected</th>
                <th class="right">Accepted</th>
              </tr>
            </thead>
            <tbody>
              {#each viewingGRN.items as item}
                <tr>
                  <td>
                    <div class="product-cell">
                      <div class="product-code">{item.product_code || 'N/A'}</div>
                      <div class="product-desc">{item.description || 'No description'}</div>
                    </div>
                  </td>
                  <td class="right mono">{item.quantity_ordered.toFixed(0)}</td>
                  <td class="right mono">{item.quantity_received.toFixed(0)}</td>
                  <td class="right mono stat-danger">{item.quantity_rejected.toFixed(0)}</td>
                  <td class="right mono stat-success">{(item.quantity_received - item.quantity_rejected).toFixed(0)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      <!-- Totals -->
      <div class="view-totals">
        <div class="total-row">
          <div class="label">Total Received:</div>
          <div class="value mono">{viewingGRN.total_received.toFixed(0)}</div>
        </div>
        <div class="total-row">
          <div class="label">Total Accepted:</div>
          <div class="value mono stat-success">{viewingGRN.total_accepted.toFixed(0)}</div>
        </div>
        <div class="total-row">
          <div class="label">Total Rejected:</div>
          <div class="value mono stat-danger">{viewingGRN.total_rejected.toFixed(0)}</div>
        </div>
        <div class="total-row grand">
          <div class="label">Acceptance Rate:</div>
          <div class="value mono">{(viewingGRN.acceptance_rate * 100).toFixed(1)}%</div>
        </div>
      </div>
    </div>

    {#snippet footer()}
      
        <Button variant="ghost" on:click={() => showViewModal = false}>
          Close
        </Button>
      
      {/snippet}
  </WabiModal>
{/if}

<!-- QC Review Modal -->
{#if qcGRN}
  <WabiModal bind:open={showQCModal} title="Quality Control Review" size="md">
    <form onsubmit={preventDefault(handleQCUpdate)} class="qc-form">
      <div class="qc-header">
        <div class="qc-info">
          <span class="label">GRN:</span>
          <span class="value mono">{qcGRN.grn_number}</span>
        </div>
        <div class="qc-info">
          <span class="label">Supplier:</span>
          <span class="value">{qcGRN.supplier_name}</span>
        </div>
      </div>

      <FormGroup label="QC Status" required>
        <select class="select-input" bind:value={qcFormData.status} required>
          <option value="Pending">Pending</option>
          <option value="Passed">Passed</option>
          <option value="Failed">Failed</option>
          <option value="Partial">Partial</option>
        </select>
      </FormGroup>

      <FormGroup label="QC Notes" required>
        <textarea
          class="textarea-input"
          bind:value={qcFormData.notes}
          rows="4"
          placeholder="Enter QC inspection notes..."
          required
        ></textarea>
      </FormGroup>

      <FormGroup label="QC By">
        <Input
          type="text"
          value={$currentUser?.full_name || $currentUser?.username || 'Unknown — please sign in again'}
          readonly
          disabled
        />
      </FormGroup>
    </form>

    {#snippet footer()}
      
        <Button variant="ghost" on:click={() => showQCModal = false}>
          Cancel
        </Button>
        <Button
          variant="primary"
          loading={formLoading}
          on:click={handleQCUpdate}
        >
          Update QC Status
        </Button>
      
      {/snippet}
  </WabiModal>
{/if}

<style>
  .grn-container {
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

  .stat-danger {
    color: #EF4444;
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

  :global(.action-btn-qc) {
    background: rgba(245, 158, 11, 0.1);
    color: #F59E0B;
  }

  :global(.action-btn-qc:hover) {
    background: #F59E0B;
    color: white;
  }

  :global(.action-btn-complete) {
    background: rgba(16, 185, 129, 0.1);
    color: #10B981;
  }

  :global(.action-btn-complete:hover) {
    background: #10B981;
    color: white;
  }

  /* Form Styles */
  .receive-form,
  .qc-form {
    display: flex;
    flex-direction: column;
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

  .receiving-as {
    font-size: 13px;
    color: var(--text-secondary);
  }

  .receiving-as strong {
    color: var(--text-primary);
  }

  .textarea-input {
    width: 100%;
    padding: 8px 12px;
    font-size: 14px;
    font-family: var(--font-family);
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
    resize: vertical;
  }

  .textarea-input:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 3px var(--brand-indigo-tint);
  }

  /* Items Section */
  .items-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .items-section h4 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .items-table-wrapper {
    overflow-x: auto;
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
  }

  .items-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
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
  }

  .items-table .right {
    text-align: right;
  }

  .product-cell {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .product-code {
    font-family: var(--font-mono);
    font-size: 12px;
    font-weight: 600;
    color: var(--brand-indigo);
  }

  .product-desc {
    font-size: 12px;
    color: var(--text-secondary);
  }

  .qty-input {
    width: 100%;
    padding: 4px 8px;
    font-size: 13px;
    font-family: var(--font-mono);
    color: var(--text-primary);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
    text-align: right;
    max-width: 80px;
  }

  .qty-input:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 2px var(--brand-indigo-tint);
  }

  .mono {
    font-family: var(--font-mono);
  }

  /* Receive Summary */
  .receive-summary {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
    background: var(--surface-elevated);
    border-radius: var(--border-radius-sm);
    border: 1px solid var(--border);
  }

  .summary-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 14px;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .summary-value {
    font-family: var(--font-mono);
    font-weight: 600;
    color: var(--text-primary);
  }

  .summary-highlight {
    padding-top: 8px;
    border-top: 1px solid var(--border);
    font-weight: 600;
  }

  .summary-highlight .summary-value {
    color: var(--onyx, #1d1d1f);
    font-size: 15px;
  }

  /* Partial receiving styles */
  .partial-info-banner {
    padding: 10px 14px;
    background: #f0f4ff;
    border: 1px solid #d0dcf0;
    border-radius: 6px;
    font-size: 13px;
    color: #3b5998;
    margin-bottom: 12px;
  }

  .prev-rcvd {
    color: var(--steel, #86868b);
  }

  .fully-received {
    opacity: 0.5;
    background: #f9f9f9;
  }

  .fully-received td {
    color: var(--steel, #86868b);
  }

  .fully-done {
    display: inline-block;
    padding: 2px 8px;
    background: #e8f5e9;
    color: #2e7d32;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
  }

  .muted {
    color: var(--steel, #86868b);
  }

  /* GRN Details */
  .grn-details {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .details-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
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

  .notes-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .notes-section h4 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .notes-content {
    padding: 12px;
    background: var(--surface-elevated);
    border-radius: var(--border-radius-sm);
    border: 1px solid var(--border);
    font-size: 14px;
    line-height: 1.5;
    color: var(--text-primary);
  }

  .view-totals {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 16px;
    background: var(--surface-elevated);
    border-radius: var(--border-radius-sm);
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

  /* QC Form */
  .qc-header {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
    background: var(--surface-elevated);
    border-radius: var(--border-radius-sm);
    border: 1px solid var(--border);
  }

  .qc-info {
    display: flex;
    gap: 8px;
    font-size: 14px;
  }

  .qc-info .label {
    font-weight: 500;
    color: var(--text-secondary);
  }

  .qc-info .value {
    font-weight: 600;
    color: var(--text-primary);
  }

  /* Responsive */
  @media (max-width: 768px) {
    .stats-grid {
      grid-template-columns: 1fr;
    }

    .details-grid {
      grid-template-columns: 1fr;
    }

    .items-table-wrapper {
      overflow-x: scroll;
    }
  }

  /* Phase 23: Serial number input */
  .serial-row td {
    background: var(--bg-tertiary, #1a1a2e) !important;
    border-top: none !important;
  }
  .serial-input-section {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .serial-label {
    font-size: 11px;
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .serial-badge {
    background: var(--brand-indigo, #6366f1);
    color: white;
    font-size: 10px;
    padding: 1px 6px;
    border-radius: 4px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .serial-textarea {
    width: 100%;
    max-width: 400px;
    padding: 6px 8px;
    font-family: var(--font-mono);
    font-size: 12px;
    line-height: 1.5;
    background: var(--bg-primary, #0f0f1a);
    border: 1px solid var(--border-primary, #2a2a4a);
    border-radius: 6px;
    color: var(--text-primary);
    resize: vertical;
  }
  .serial-textarea:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 2px var(--brand-indigo-tint);
  }
  .serial-count {
    font-size: 11px;
    font-family: var(--font-mono);
  }
  .serial-count-ok {
    color: var(--stat-success, #10b981);
  }
  .serial-count-err {
    color: var(--stat-danger, #ef4444);
  }
</style>
