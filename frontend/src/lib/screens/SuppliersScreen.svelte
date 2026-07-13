<script lang="ts">
  import { run } from 'svelte/legacy';

  /**
   * SuppliersScreen - Production-Ready Supplier Management
   * Features:
   * - View all suppliers with DataTable
   * - Search and filter by name, status
   * - Supplier detail view modal (with POs and invoices summary)
   * - Stats: Total suppliers, Active, Pending invoices
   * - Status filter tabs
   */

  import { onMount, createEventDispatcher } from 'svelte';
  import { fade } from 'svelte/transition';

  // Wails API imports
	  import {
	    ListSuppliers } from '../../../wailsjs/go/main/App';
import { CreateSupplier, DeleteSupplier, GetPurchaseOrdersBySupplier } from '../../../wailsjs/go/main/CRMService';
import { GetSupplierInvoicesBySupplier } from '../../../wailsjs/go/main/FinanceService';

  // Design system components
  import PageLayout from '$lib/components/layout/PageLayout.svelte';
  import DataTable from '$lib/components/ui/DataTable.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import StatusBadge from '$lib/components/ui/StatusBadge.svelte';
  import WabiModal from '$lib/components/ui/WabiModal.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import { toast } from '$lib/stores/toasts';
  import { confirm } from '$lib/stores/confirm';
  import { formatBHD } from '$lib/utils/formatters';
  import { escapeHtml } from '$lib/utils/escapeHtml';

  
  interface Props {
    // Props
    embedded?: boolean;
  }

  let { embedded = false }: Props = $props();

  const dispatch = createEventDispatcher();

  // Types
  type SupplierStatus = 'all' | 'Active' | 'Inactive' | 'Pending';

  // State
  let suppliers: any[] = $state([]);
  let filteredSuppliers: any[] = $state([]);
  let loading = $state(true);
  let searchQuery = $state('');
  let selectedStatus: SupplierStatus = $state('all');

	  // Create supplier state
	  let showCreateModal = $state(false);
	  let creating = $state(false);
	  let deletingSupplierId = $state('');

	  function emptySupplierForm() {
	    return {
	      supplier_name: '', supplier_code: '', supplier_type: '',
	      primary_contact: '', email: '', phone: '', country: '',
	      address: '', tax_id: '', brands_handled: '', lead_time_days: '',
	    };
	  }

	  let newSupplier: any = $state(emptySupplierForm());

  // Wave 9.6 Sh1: SupplierMaster.BrandsHandled is a JSON-encoded STRING column
  // (see GetSupplierFullProfile's json.Unmarshal), but this form lets the user
  // type free-text/CSV. Stored verbatim, that CSV fails the profile's
  // json.Unmarshal silently and the brands never display. Convert the typed
  // CSV into a JSON-encoded string[] so it round-trips through the profile view.
  function encodeBrandsHandled(raw: string): string {
    const trimmed = (raw || '').trim();
    if (!trimmed) return '';
    const list = trimmed.split(',').map((s) => s.trim()).filter(Boolean);
    return list.length > 0 ? JSON.stringify(list) : '';
  }

  async function handleCreateSupplier() {
    if (!newSupplier.supplier_name?.trim()) {
      toast.danger('Supplier name is required');
      return;
    }
    creating = true;
    try {
	      const prefix = newSupplier.supplier_name.substring(0, 4).toUpperCase().replace(/[^A-Z]/g, '');
	      const suffix = Date.now().toString().slice(-4);
	      const payload = {
	        ...newSupplier,
	        supplier_code: newSupplier.supplier_code || `SUP-${prefix}${suffix}`,
	        lead_time_days: Number(newSupplier.lead_time_days) || 0,
	        brands_handled: encodeBrandsHandled(newSupplier.brands_handled),
	      };
	      await CreateSupplier(payload);
	      toast.success(`Created supplier: ${newSupplier.supplier_name}`);
	      showCreateModal = false;
	      newSupplier = emptySupplierForm();
	      await loadSuppliers();
    } catch (e) {
      toast.danger(`Create failed: ${String(e)}`);
    } finally {
      creating = false;
    }
  }

  // Detail modal state
  let showDetailModal = $state(false);
  let selectedSupplier: any = $state(null);
  let supplierPOs: any[] = $state([]);
  let supplierInvoices: any[] = $state([]);
  let detailLoading = $state(false);

  // DataTable columns configuration
  const columns = [
    {
      key: 'supplier_code',
      label: 'Code',
      sortable: true,
      width: '120px',
      render: (row: any) => {
        const code = row.supplier_code || row.supplier_id || 'N/A';
        return `<span style="font-family: 'JetBrains Mono', monospace; color: var(--brand-indigo); font-weight: 500;">${escapeHtml(code)}</span>`;
      }
    },
    {
      key: 'supplier_name',
      label: 'Supplier Name',
      sortable: true
    },
    {
      key: 'primary_contact',
      label: 'Contact Person',
      sortable: true,
      width: '180px'
    },
    {
      key: 'phone',
      label: 'Phone',
      width: '140px'
    },
    {
      key: 'email',
      label: 'Email',
      sortable: true,
      width: '200px'
    },
    {
      key: 'tax_id',
      label: 'VAT/TRN',
      width: '140px',
      render: (row: any) => {
        const vat = row.tax_id || '-';
        return `<span style="font-family: 'JetBrains Mono', monospace; font-size: 12px;">${escapeHtml(vat)}</span>`;
      }
    },
    {
      key: 'status',
      label: 'Status',
      type: 'status' as const,
      sortable: true,
      width: '110px',
      render: (row: any) => {
        const status = row.status || 'Active';
        const colorMap: Record<string, string> = {
          'Active': 'success',
          'Inactive': 'neutral',
          'Pending': 'warning'
        };
        const color = colorMap[status] || 'neutral';
        return `<span class="status-badge status-${color}">${escapeHtml(status)}</span>`;
      }
    },
    {
      key: 'actions',
      label: 'Actions',
      type: 'actions' as const,
      width: '100px',
      align: 'center' as const,
      render: (row: any) => {
        return `
          <div style="display: flex; gap: 8px; justify-content: center;">
	            <button
	              class="action-btn action-btn-view"
              data-action="view"
              data-id="${row.id}"
              aria-label="View supplier details"
	            >
	              View
	            </button>
	            <button
	              class="action-btn action-btn-delete"
	              data-action="delete"
	              data-id="${row.id}"
	              aria-label="Delete supplier"
	            >
	              Delete
	            </button>
	          </div>
	        `;
      }
    }
  ];

  // Status filter tabs
  const statusTabs: { value: SupplierStatus; label: string; count: number }[] = $state([
    { value: 'all', label: 'All Suppliers', count: 0 },
    { value: 'Active', label: 'Active', count: 0 },
    { value: 'Inactive', label: 'Inactive', count: 0 },
    { value: 'Pending', label: 'Pending', count: 0 }
  ]);

  // Computed: Update tab counts
  run(() => {
    statusTabs[0].count = suppliers.length;
    statusTabs[1].count = suppliers.filter(s => (s.status || 'Active') === 'Active').length;
    statusTabs[2].count = suppliers.filter(s => s.status === 'Inactive').length;
    statusTabs[3].count = suppliers.filter(s => s.status === 'Pending').length;
  });

  // Computed: Filter suppliers by search and status
  run(() => {
    filteredSuppliers = suppliers.filter((s) => {
      // Search filter
      const matchSearch =
        !searchQuery ||
        s.supplier_name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        s.supplier_code?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        s.primary_contact?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        s.email?.toLowerCase().includes(searchQuery.toLowerCase());

      // Status filter
      const supplierStatus = s.status || 'Active';
      const matchStatus = selectedStatus === 'all' || supplierStatus === selectedStatus;

      return matchSearch && matchStatus;
    });
  });

  // Load suppliers
  async function loadSuppliers() {
    loading = true;
    try {
      const res = await ListSuppliers(500, 0); // Get up to 500 suppliers
      suppliers = res || [];
      console.log(`Loaded ${suppliers.length} suppliers`);
    } catch (err) {
      console.error('Failed to load suppliers:', err);
      toast.danger('Failed to load suppliers');
      suppliers = [];
    } finally {
      loading = false;
    }
  }

  // Open supplier detail modal
  async function openSupplierDetail(supplier: any) {
    selectedSupplier = supplier;
    showDetailModal = true;
    detailLoading = true;
    supplierPOs = [];
    supplierInvoices = [];

    try {
      // Load supplier's POs and invoices in parallel
      const [pos, invoices] = await Promise.all([
        GetPurchaseOrdersBySupplier(supplier.id),
        GetSupplierInvoicesBySupplier(supplier.id)
      ]);

      supplierPOs = pos || [];
      supplierInvoices = invoices || [];
      console.log(`Loaded ${supplierPOs.length} POs, ${supplierInvoices.length} invoices for ${supplier.business_name}`);
    } catch (err) {
      console.error('Failed to load supplier details:', err);
      toast.danger('Failed to load supplier details');
    } finally {
      detailLoading = false;
    }
  }

  // Handle action button clicks (delegated from DataTable)
  function handleRowClick(event: CustomEvent) {
    const target = event.detail.event?.target as HTMLElement;
    if (!target || !target.dataset.action) return;

    const action = target.dataset.action;
    const id = target.dataset.id;
    const supplier = suppliers.find(s => s.id === id);

    if (!supplier) return;

	    if (action === 'view') {
	      if (embedded) {
	        dispatch('select', { id: supplier.id || supplier.supplier_id });
	      } else {
	        openSupplierDetail(supplier);
	      }
	    } else if (action === 'delete') {
	      void handleDeleteSupplier(supplier);
	    }
	  }

	  async function handleDeleteSupplier(supplier: any) {
	    const supplierId = supplier?.id || supplier?.supplier_id;
	    const supplierName = supplier?.supplier_name || 'this supplier';
	    if (!supplierId || !(await confirm.ask({
	      title: 'Delete Supplier',
	      message: `Delete ${supplierName}? This cannot be undone.`,
	      confirmLabel: 'Delete',
	      variant: 'danger'
	    }))) {
	      return;
	    }
	    deletingSupplierId = supplierId;
	    try {
	      await DeleteSupplier(supplierId);
	      toast.success(`Deleted supplier: ${supplierName}`);
	      showDetailModal = false;
	      selectedSupplier = null;
	      await loadSuppliers();
	    } catch (err) {
	      toast.danger(`Delete failed: ${String(err)}`);
	    } finally {
	      deletingSupplierId = '';
	    }
	  }

  // Computed: Summary stats
  let totalSuppliers = $derived(suppliers.length);
  let activeSuppliers = $derived(suppliers.filter(s => (s.status || 'Active') === 'Active').length);
  let totalPendingInvoices = $derived(suppliers.reduce((sum, s) => {
    // Assuming suppliers might have a pending_invoices_count field
    // If not, we'll show 0 for now
    return sum + (s.pending_invoices_count || 0);
  }, 0));

  onMount(() => {
    loadSuppliers();
  });
</script>

<PageLayout title="Suppliers" subtitle="Vendor & Partner Directory" {embedded}>
  <!-- @migration-task: migrate this slot by hand, `header-actions` is an invalid identifier -->
  <svelte:fragment slot="header-actions">
    <Button variant="primary" on:click={() => { newSupplier = emptySupplierForm(); showCreateModal = true; }}>
      + New Supplier
    </Button>
    <Button variant="secondary" on:click={loadSuppliers}>
      Refresh
    </Button>
  </svelte:fragment>

  <div class="suppliers-container">
    <!-- Search and Status Tabs -->
    <Card padding="sm">
      <div class="filters-row">
        <!-- Search -->
        <div class="search-box">
          <input
            type="text"
            placeholder="Search suppliers..."
            bind:value={searchQuery}
            class="search-input"
          />
        </div>

        <!-- Status Tabs -->
        <div class="status-tabs" role="tablist" aria-label="Filter suppliers by status">
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
      </div>
    </Card>

    <!-- Stats Cards -->
    <div class="stats-grid">
      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Total Suppliers</div>
          <div class="stat-value">{totalSuppliers}</div>
        </div>
      </Card>

      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Active</div>
          <div class="stat-value stat-success">{activeSuppliers}</div>
        </div>
      </Card>

      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Pending Invoices</div>
          <div class="stat-value stat-warning">{totalPendingInvoices}</div>
        </div>
      </Card>

      <Card padding="md">
        <div class="stat">
          <div class="stat-label">Active Rate</div>
          <div class="stat-value">
            {totalSuppliers > 0 ? ((activeSuppliers / totalSuppliers) * 100).toFixed(1) : 0}%
          </div>
        </div>
      </Card>
    </div>

    <!-- Suppliers DataTable -->
    <Card padding="sm">
      <DataTable
        {columns}
        data={filteredSuppliers}
        {loading}
        emptyMessage="No suppliers yet — add one to begin sourcing."
        onRowClick={(row) => {}}
        on:rowClick={handleRowClick}
        stickyHeader={true}
        maxHeight="calc(100vh - 380px)"
        showBorder={false}
      />
    </Card>
  </div>
</PageLayout>

<!-- Supplier Detail Modal -->
<WabiModal bind:open={showDetailModal} title="Supplier Details" size="lg">
  {#if selectedSupplier}
    <div class="detail-content">
      <!-- Supplier Info -->
      <div class="info-section">
        <h3 class="section-title">Supplier Information</h3>
        <div class="info-grid">
          <div class="info-item">
            <div class="info-label">Business Name</div>
            <div class="info-value">{selectedSupplier.supplier_name || '-'}</div>
          </div>
          <div class="info-item">
            <div class="info-label">Supplier Code</div>
            <div class="info-value code">{selectedSupplier.supplier_code || selectedSupplier.supplier_id || '-'}</div>
          </div>
          <div class="info-item">
            <div class="info-label">Contact Person</div>
            <div class="info-value">{selectedSupplier.primary_contact || '-'}</div>
          </div>
          <div class="info-item">
            <div class="info-label">Phone</div>
            <div class="info-value">{selectedSupplier.phone || '-'}</div>
          </div>
          <div class="info-item">
            <div class="info-label">Email</div>
            <div class="info-value">{selectedSupplier.email || '-'}</div>
          </div>
          <div class="info-item">
            <div class="info-label">VAT Number</div>
            <div class="info-value code">{selectedSupplier.tax_id || '-'}</div>
          </div>
          <div class="info-item">
            <div class="info-label">Address</div>
            <div class="info-value">{selectedSupplier.address || '-'}</div>
          </div>
          <div class="info-item">
            <div class="info-label">Status</div>
            <div class="info-value">
              <StatusBadge status={selectedSupplier.status || 'Active'} />
            </div>
          </div>
        </div>
      </div>

      <!-- Purchase Orders Summary -->
      <div class="summary-section">
        <h3 class="section-title">
          Purchase Orders
          {#if !detailLoading}
            <span class="count-badge">{supplierPOs.length}</span>
          {/if}
        </h3>
        {#if detailLoading}
          <div class="loading-state">
            <WabiSpinner size="sm" />
          </div>
        {:else if supplierPOs.length === 0}
          <div class="empty-state">No purchase orders found</div>
        {:else}
          <div class="summary-list">
            {#each supplierPOs.slice(0, 5) as po}
              <div class="summary-item" transition:fade>
                <div class="summary-item-main">
                  <span class="summary-code">{po.po_number || po.id}</span>
                  <span class="summary-date">
                    {po.order_date ? new Date(po.order_date).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }) : 'N/A'}
                  </span>
                </div>
                <div class="summary-item-meta">
                  <span class="summary-status">
                    <StatusBadge status={po.status || 'Pending'} size="sm" />
                  </span>
                  <span class="summary-amount">{formatBHD(po.total_amount || 0)}</span>
                </div>
              </div>
            {/each}
            {#if supplierPOs.length > 5}
              <div class="more-items">+ {supplierPOs.length - 5} more...</div>
            {/if}
          </div>
        {/if}
      </div>

      <!-- Supplier Invoices Summary -->
      <div class="summary-section">
        <h3 class="section-title">
          Invoices
          {#if !detailLoading}
            <span class="count-badge">{supplierInvoices.length}</span>
          {/if}
        </h3>
        {#if detailLoading}
          <div class="loading-state">
            <WabiSpinner size="sm" />
          </div>
        {:else if supplierInvoices.length === 0}
          <div class="empty-state">No invoices found</div>
        {:else}
          <div class="summary-list">
            {#each supplierInvoices.slice(0, 5) as invoice}
              <div class="summary-item" transition:fade>
                <div class="summary-item-main">
                  <span class="summary-code">{invoice.invoice_number || invoice.id}</span>
                  <span class="summary-date">
                    {invoice.invoice_date ? new Date(invoice.invoice_date).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }) : 'N/A'}
                  </span>
                </div>
                <div class="summary-item-meta">
                  <span class="summary-status">
                    <StatusBadge status={invoice.status || 'Pending'} size="sm" />
                  </span>
                  <span class="summary-amount">{formatBHD(invoice.total_amount || invoice.amount || 0)}</span>
                </div>
              </div>
            {/each}
            {#if supplierInvoices.length > 5}
              <div class="more-items">+ {supplierInvoices.length - 5} more...</div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  {/if}

	  {#snippet footer()}
  
  	    <Button
  	      variant="danger"
  	      on:click={() => handleDeleteSupplier(selectedSupplier)}
  	      disabled={deletingSupplierId === (selectedSupplier?.id || selectedSupplier?.supplier_id)}
  	    >
  	      {deletingSupplierId === (selectedSupplier?.id || selectedSupplier?.supplier_id) ? 'Deleting...' : 'Delete Supplier'}
  	    </Button>
  	    <Button variant="ghost" on:click={() => showDetailModal = false}>
  	      Close
  	    </Button>
    
  {/snippet}
</WabiModal>

<!-- Create Supplier Modal -->
<WabiModal bind:open={showCreateModal} title="New Supplier" size="lg">
  <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
    <div style="grid-column: 1 / -1;">
      <label for="supplier-name" style="font-size: 12px; color: var(--steel); display: block; margin-bottom: 4px;">Supplier Name *</label>
      <input id="supplier-name" type="text" bind:value={newSupplier.supplier_name} placeholder="Company name" style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-secondary); border-radius: 6px; background: var(--bg-primary); color: var(--text-primary); font-size: 14px;" />
    </div>
    <div>
      <label for="supplier-code" style="font-size: 12px; color: var(--steel); display: block; margin-bottom: 4px;">Supplier Code</label>
      <input id="supplier-code" type="text" bind:value={newSupplier.supplier_code} placeholder="Auto-generated if empty" style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-secondary); border-radius: 6px; background: var(--bg-primary); color: var(--text-primary); font-size: 14px;" />
    </div>
    <div>
      <label for="supplier-type" style="font-size: 12px; color: var(--steel); display: block; margin-bottom: 4px;">Supplier Type</label>
	      <select id="supplier-type" bind:value={newSupplier.supplier_type} style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-secondary); border-radius: 6px; background: var(--bg-primary); color: var(--text-primary); font-size: 14px;">
	        <option value="" disabled>Select type</option>
	        <option>Manufacturer</option>
	        <option>Distributor</option>
        <option>Agent</option>
        <option>Service Provider</option>
      </select>
    </div>
    <div>
      <label for="supplier-contact" style="font-size: 12px; color: var(--steel); display: block; margin-bottom: 4px;">Contact Person</label>
      <input id="supplier-contact" type="text" bind:value={newSupplier.primary_contact} placeholder="Primary contact name" style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-secondary); border-radius: 6px; background: var(--bg-primary); color: var(--text-primary); font-size: 14px;" />
    </div>
    <div>
      <label for="supplier-email" style="font-size: 12px; color: var(--steel); display: block; margin-bottom: 4px;">Email</label>
      <input id="supplier-email" type="email" bind:value={newSupplier.email} placeholder="supplier@example.com" style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-secondary); border-radius: 6px; background: var(--bg-primary); color: var(--text-primary); font-size: 14px;" />
    </div>
    <div>
      <label for="supplier-phone" style="font-size: 12px; color: var(--steel); display: block; margin-bottom: 4px;">Phone</label>
      <input id="supplier-phone" type="text" bind:value={newSupplier.phone} placeholder="+973 1234 5678" style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-secondary); border-radius: 6px; background: var(--bg-primary); color: var(--text-primary); font-size: 14px;" />
    </div>
    <div>
      <label for="supplier-country" style="font-size: 12px; color: var(--steel); display: block; margin-bottom: 4px;">Country</label>
      <input id="supplier-country" type="text" bind:value={newSupplier.country} placeholder="Bahrain" style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-secondary); border-radius: 6px; background: var(--bg-primary); color: var(--text-primary); font-size: 14px;" />
    </div>
    <div>
      <label for="supplier-tax-id" style="font-size: 12px; color: var(--steel); display: block; margin-bottom: 4px;">Tax ID / TRN</label>
      <input id="supplier-tax-id" type="text" bind:value={newSupplier.tax_id} placeholder="VAT registration number" style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-secondary); border-radius: 6px; background: var(--bg-primary); color: var(--text-primary); font-size: 14px;" />
    </div>
    <div>
      <label for="supplier-lead-time" style="font-size: 12px; color: var(--steel); display: block; margin-bottom: 4px;">Lead Time (days)</label>
      <input id="supplier-lead-time" type="number" bind:value={newSupplier.lead_time_days} min="0" max="365" style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-secondary); border-radius: 6px; background: var(--bg-primary); color: var(--text-primary); font-size: 14px;" />
    </div>
    <div style="grid-column: 1 / -1;">
      <label for="supplier-address" style="font-size: 12px; color: var(--steel); display: block; margin-bottom: 4px;">Address</label>
      <input id="supplier-address" type="text" bind:value={newSupplier.address} placeholder="Full address" style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-secondary); border-radius: 6px; background: var(--bg-primary); color: var(--text-primary); font-size: 14px;" />
    </div>
    <div style="grid-column: 1 / -1;">
      <label for="supplier-brands" style="font-size: 12px; color: var(--steel); display: block; margin-bottom: 4px;">Brands Handled</label>
      <input id="supplier-brands" type="text" bind:value={newSupplier.brands_handled} placeholder="Rhine Instruments, Oxan Analytics, etc." style="width: 100%; padding: 8px 12px; border: 1px solid var(--border-secondary); border-radius: 6px; background: var(--bg-primary); color: var(--text-primary); font-size: 14px;" />
    </div>
  </div>
  {#snippet footer()}
  
      <Button variant="ghost" on:click={() => showCreateModal = false}>Cancel</Button>
      <Button variant="primary" on:click={handleCreateSupplier} disabled={creating || !newSupplier.supplier_name?.trim()}>
        {creating ? 'Creating...' : 'Create Supplier'}
      </Button>
    
  {/snippet}
</WabiModal>

<style>
  .suppliers-container {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  /* Filters Row */
  .filters-row {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .search-box {
    width: 100%;
  }

  .search-input {
    width: 100%;
    padding: 10px 16px;
    font-size: 14px;
    font-family: var(--font-family);
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius);
    transition: all var(--transition-fast);
  }

  .search-input:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 3px var(--brand-indigo-tint);
  }

  .search-input::placeholder {
    color: var(--text-muted);
  }

  /* Status Tabs */
  .status-tabs {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    padding-bottom: 4px;
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
    font-size: 28px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .stat-success {
    color: #10B981;
  }

  .stat-warning {
    color: #F59E0B;
  }

  /* Action Buttons in Table */
  :global(.action-btn) {
    padding: 6px 12px;
    font-size: 12px;
    font-weight: 500;
    border: none;
    border-radius: var(--border-radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  :global(.action-btn-view) {
    background: var(--brand-indigo-tint);
    color: var(--brand-indigo);
  }

	  :global(.action-btn-view:hover) {
	    background: var(--brand-indigo);
	    color: white;
	  }

	  :global(.action-btn-delete) {
	    background: #fee2e2;
	    color: #991b1b;
	    border: 1px solid rgba(185, 28, 28, 0.24);
	  }

	  :global(.action-btn-delete:hover) {
	    background: #fecaca;
	    border-color: rgba(185, 28, 28, 0.42);
	  }

	  :global(.status-badge) {
    display: inline-block;
    padding: 4px 8px;
    border-radius: 12px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }

  :global(.status-success) {
    background: rgba(16, 185, 129, 0.1);
    color: #10B981;
  }

  :global(.status-neutral) {
    background: rgba(107, 114, 128, 0.1);
    color: #6B7280;
  }

  :global(.status-warning) {
    background: rgba(245, 158, 11, 0.1);
    color: #F59E0B;
  }

  /* Detail Modal */
  .detail-content {
    display: flex;
    flex-direction: column;
    gap: 24px;
    max-height: 60vh;
    overflow-y: auto;
  }

  .section-title {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .count-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 24px;
    height: 20px;
    padding: 0 6px;
    background: var(--brand-indigo-tint);
    color: var(--brand-indigo);
    border-radius: 10px;
    font-size: 12px;
    font-weight: 700;
  }

  .info-section {
    padding: 16px;
    background: var(--surface-elevated);
    border-radius: var(--border-radius);
  }

  .info-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
  }

  .info-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .info-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .info-value {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary);
  }

  .info-value.code {
    font-family: 'JetBrains Mono', monospace;
    color: var(--brand-indigo);
  }

  .summary-section {
    padding: 16px;
    background: var(--surface-elevated);
    border-radius: var(--border-radius);
  }

  .summary-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .summary-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
  }

  .summary-item:hover {
    border-color: var(--brand-indigo);
    box-shadow: 0 2px 8px rgba(99, 102, 241, 0.1);
  }

  .summary-item-main {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .summary-code {
    font-family: 'JetBrains Mono', monospace;
    font-size: 13px;
    font-weight: 600;
    color: var(--brand-indigo);
  }

  .summary-date {
    font-size: 12px;
    color: var(--text-muted);
  }

  .summary-item-meta {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .summary-amount {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
    font-family: 'JetBrains Mono', monospace;
  }

  .loading-state {
    display: flex;
    justify-content: center;
    padding: 24px;
  }

  .empty-state {
    padding: 24px;
    text-align: center;
    color: var(--text-muted);
    font-style: italic;
    font-size: 14px;
  }

  .more-items {
    padding: 8px;
    text-align: center;
    font-size: 12px;
    color: var(--text-muted);
    font-style: italic;
  }

  /* Responsive */
  @media (max-width: 768px) {
    .info-grid {
      grid-template-columns: 1fr;
    }

    .stats-grid {
      grid-template-columns: 1fr;
    }

    .summary-item {
      flex-direction: column;
      align-items: flex-start;
      gap: 8px;
    }

    .summary-item-meta {
      width: 100%;
      justify-content: space-between;
    }
  }
</style>
