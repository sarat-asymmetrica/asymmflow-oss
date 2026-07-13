<script lang="ts">
  import { run } from 'svelte/legacy';

  /**
   * RFQ (Request for Quotation) Screen
   *
   * Complete RFQ management with multi-product line items
   * Win rate target: 56.8%
   * Stages: Pending → Qualified → Proposal → Negotiation → Won/Lost
   *
   * Features:
   * - Stage-based filtering tabs
   * - Multi-product line item support
   * - Real-time win rate tracking
   * - Full CRUD with backend integration
   */

  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  // Layout & UI Components
  import PageLayout from '$lib/components/layout/PageLayout.svelte';
  import DataTable from '$lib/components/ui/DataTable.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import WabiModal from '$lib/components/ui/WabiModal.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Select, { type SelectOption } from '$lib/components/ui/Select.svelte';
  import Textarea from '$lib/components/ui/Textarea.svelte';
  import DatePicker from '$lib/components/ui/DatePicker.svelte';
  import CurrencyInput from '$lib/components/ui/CurrencyInput.svelte';
  import StatusBadge from '$lib/components/ui/StatusBadge.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';

  // Backend APIs
  import { GetRFQs } from '../../../wailsjs/go/main/App';
import { CreateRFQ, UpdateRFQStatus, UpdateRFQStage, DeleteRFQ, ListCustomers } from '../../../wailsjs/go/main/CRMService';

  // Toast notifications
  import { toast } from '$lib/stores/toasts';
  import { confirm } from '$lib/stores/confirm';
  import { escapeHtml } from '$lib/utils/escapeHtml';

  // ========================================
  // STATE MANAGEMENT
  // ========================================

  // RFQ list
  let rfqs: any[] = $state([]);
  let loading = $state(true);
  let error = $state('');

  // Filters
  let activeStage = $state('All');
  let searchQuery = $state('');

  // Modal state
  let showCreateModal = $state(false);
  let showEditModal = $state(false);
  let editingRFQ: any = $state(null);

  // Form state
  let formData = $state({
    customer: '',
    products: [] as Array<{ description: string; quantity: number; target_price: number }>,
    due_date: '',
    notes: ''
  });

  let formErrors = $state({
    customer: '',
    products: '',
    due_date: ''
  });

  // Lookup data
  let customers: any[] = $state([]);

  // Stats
  let stats = $state({
    total: 0,
    pending: 0,
    qualified: 0,
    proposal: 0,
    negotiation: 0,
    won: 0,
    lost: 0,
    winRate: 0,
    totalValue: 0,
    avgDealSize: 0
  });

  // ========================================
  // STAGE CONFIGURATION
  // ========================================

  const stages = $state([
    { id: 'All', label: 'All RFQs', count: 0 },
    { id: 'Pending', label: 'Pending', count: 0 },
    { id: 'Qualified', label: 'Qualified', count: 0 },
    { id: 'Proposal', label: 'Proposal', count: 0 },
    { id: 'Negotiation', label: 'Negotiation', count: 0 },
    { id: 'Won', label: 'Won', count: 0 },
    { id: 'Lost', label: 'Lost', count: 0 }
  ]);

  const stageColors: Record<string, any> = {
    'Pending': { variant: 'neutral', color: '#6B7280' },
    'Qualified': { variant: 'info', color: '#3B82F6' },
    'Proposal': { variant: 'info', color: '#6366F1' },
    'Negotiation': { variant: 'warning', color: '#F59E0B' },
    'Won': { variant: 'success', color: '#10B981' },
    'Lost': { variant: 'danger', color: '#EF4444' }
  };

  // ========================================
  // DATA TABLE CONFIGURATION
  // ========================================

  const columns = [
    {
      key: 'id',
      label: 'RFQ #',
      sortable: true,
      width: '100px',
      render: (row: any) => `<span class="mono">RFQ-${escapeHtml(String(row.id).padStart(4, '0'))}</span>`
    },
    {
      key: 'client',
      label: 'Customer',
      sortable: true,
      width: '200px',
      render: (row: any) => {
        const name = row.client || '';
        // Detect UUID-like values (8 hex chars followed by dash)
        if (/^[0-9a-f]{8}-/i.test(name)) {
          return `<span class="unknown-company">Unknown Company</span>`;
        }
        return escapeHtml(name);
      }
    },
    {
      key: 'product_count',
      label: 'Products',
      align: 'center' as const,
      width: '100px',
      render: (row: any) => {
        const count = Number(row.product_count) || 1;
        return `<span class="product-count">${count} item${count > 1 ? 's' : ''}</span>`;
      }
    },
    {
      key: 'value',
      label: 'Total Value',
      type: 'currency' as const,
      align: 'right' as const,
      sortable: true,
      width: '150px'
    },
    {
      key: 'created_at',
      label: 'Created',
      type: 'date' as const,
      sortable: true,
      width: '120px'
    },
    {
      key: 'notes',
      label: 'Due Date',
      width: '120px',
      render: (row: any) => {
        // Extract due date from notes or use a placeholder
        const dueDate = row.due_date || 'Not set';
        if (dueDate === 'Not set') {
          return '<span style="color: var(--text-muted); font-style: italic;">Not set</span>';
        }
        try {
          return new Date(dueDate).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
        } catch {
          return escapeHtml(dueDate);
        }
      }
    },
    {
      key: 'status',
      label: 'Stage',
      type: 'status' as const,
      width: '140px',
      render: (row: any) => {
        const stage = row.status || 'Pending';
        const config = stageColors[stage] || stageColors['Pending'];
        return `<span class="status-badge status-${config.variant}">${escapeHtml(stage)}</span>`;
      }
    },
    {
      key: 'actions',
      label: '',
      type: 'actions' as const,
      width: '120px',
      align: 'right' as const,
      render: (row: any) => {
        return `
          <div class="action-buttons">
            <button class="btn-action" data-action="edit" data-id="${row.id}" title="Edit RFQ">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path d="M11.333 2.00004C11.5081 1.82494 11.716 1.68605 11.9447 1.59129C12.1735 1.49653 12.4187 1.44775 12.6663 1.44775C12.914 1.44775 13.1592 1.49653 13.3879 1.59129C13.6167 1.68605 13.8246 1.82494 13.9997 2.00004C14.1748 2.17513 14.3137 2.383 14.4084 2.61178C14.5032 2.84055 14.552 3.08575 14.552 3.33337C14.552 3.58099 14.5032 3.82619 14.4084 4.05497C14.3137 4.28374 14.1748 4.49161 13.9997 4.66671L5.33301 13.3334L1.99967 14.3334L2.99967 11L11.333 2.00004Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
            <button class="btn-action" data-action="delete" data-id="${row.id}" title="Delete RFQ">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path d="M2 4H3.33333M3.33333 4H14M3.33333 4V13.3333C3.33333 13.687 3.47381 14.0261 3.72386 14.2761C3.97391 14.5262 4.31304 14.6667 4.66667 14.6667H11.3333C11.687 14.6667 12.0261 14.5262 12.2761 14.2761C12.5262 14.0261 12.6667 13.687 12.6667 13.3333V4H3.33333ZM5.33333 4V2.66667C5.33333 2.31304 5.47381 1.97391 5.72386 1.72386C5.97391 1.47381 6.31304 1.33333 6.66667 1.33333H9.33333C9.68696 1.33333 10.0261 1.47381 10.2761 1.72386C10.5262 1.97391 10.6667 2.31304 10.6667 2.66667V4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
          </div>
        `;
      }
    }
  ];

  // ========================================
  // COMPUTED VALUES
  // ========================================

  let filteredRFQs = $derived(rfqs.filter(rfq => {
    // Stage filter
    const stageMatch = activeStage === 'All' || rfq.status === activeStage;

    // Search filter
    const searchLower = searchQuery.toLowerCase();
    const searchMatch = !searchQuery ||
      rfq.client?.toLowerCase().includes(searchLower) ||
      rfq.project?.toLowerCase().includes(searchLower) ||
      rfq.notes?.toLowerCase().includes(searchLower) ||
      `RFQ-${String(rfq.id).padStart(4, '0')}`.toLowerCase().includes(searchLower);

    return stageMatch && searchMatch;
  }));

  // Update stage counts
  run(() => {
    stages[0].count = rfqs.length;
    stages[1].count = rfqs.filter(r => r.status === 'Pending').length;
    stages[2].count = rfqs.filter(r => r.status === 'Qualified').length;
    stages[3].count = rfqs.filter(r => r.status === 'Proposal').length;
    stages[4].count = rfqs.filter(r => r.status === 'Negotiation').length;
    stages[5].count = rfqs.filter(r => r.status === 'Won').length;
    stages[6].count = rfqs.filter(r => r.status === 'Lost').length;
  });

  // Update stats
  run(() => {
    stats.total = rfqs.length;
    stats.pending = rfqs.filter(r => r.status === 'Pending').length;
    stats.qualified = rfqs.filter(r => r.status === 'Qualified').length;
    stats.proposal = rfqs.filter(r => r.status === 'Proposal').length;
    stats.negotiation = rfqs.filter(r => r.status === 'Negotiation').length;
    stats.won = rfqs.filter(r => r.status === 'Won').length;
    stats.lost = rfqs.filter(r => r.status === 'Lost').length;

    const closed = stats.won + stats.lost;
    stats.winRate = closed > 0 ? (stats.won / closed) * 100 : 0;

    stats.totalValue = rfqs.reduce((sum, r) => sum + (r.value || 0), 0);
    stats.avgDealSize = stats.total > 0 ? stats.totalValue / stats.total : 0;
  });

  // Customer options for select - CustomerMaster uses business_name field
  let customerOptions = $derived(customers.map(c => ({
    value: String(c.id),
    label: c.business_name || c.name || 'Unknown'
  })) as SelectOption[]);

  // ========================================
  // LIFECYCLE & DATA LOADING
  // ========================================

  onMount(async () => {
    await loadData();
  });

  async function loadData() {
    loading = true;
    error = '';

    try {
      // Load RFQs and customers in parallel
      const [rfqsData, customersData] = await Promise.all([
        GetRFQs(100, 0),
        ListCustomers(500, 0).catch(() => [])
      ]);

      rfqs = rfqsData || [];
      customers = customersData || [];

      console.log('Loaded RFQs:', rfqs.length);
      console.log('Loaded Customers:', customers.length);

    } catch (err) {
      console.error('Failed to load RFQ data:', err);
      error = err instanceof Error ? err.message : 'Failed to load data';
      toast.danger(`Failed to load RFQs: ${error}`);
    } finally {
      loading = false;
    }
  }

  // ========================================
  // FORM MANAGEMENT
  // ========================================

  function openCreateModal() {
    formData = {
      customer: '',
      products: [],
      due_date: '',
      notes: ''
    };
    formErrors = {
      customer: '',
      products: '',
      due_date: ''
    };
    showCreateModal = true;
  }

  function closeCreateModal() {
    showCreateModal = false;
  }

  function openEditModal(rfq: any) {
    editingRFQ = rfq;
    formData = {
      customer: String(rfq.customer_id || ''),
      products: [], // TODO: Parse from RFQ data
      due_date: rfq.due_date || '',
      notes: rfq.notes || ''
    };
    showEditModal = true;
  }

  function closeEditModal() {
    showEditModal = false;
    editingRFQ = null;
  }

  function addLineItem() {
    formData.products = [...formData.products, {
      description: '',
      quantity: 1,
      target_price: 0
    }];
  }

  function removeLineItem(index: number) {
    formData.products = formData.products.filter((_, i) => i !== index);
  }

  function validateForm(): boolean {
    formErrors = {
      customer: '',
      products: '',
      due_date: ''
    };

    let isValid = true;

    if (!formData.customer) {
      formErrors.customer = 'Customer is required';
      isValid = false;
    }

    if (formData.products.length === 0) {
      formErrors.products = 'At least one product is required';
      isValid = false;
    } else {
      // Validate each line item
      for (const item of formData.products) {
        if (!item.description || !item.description.trim() || item.quantity <= 0) {
          formErrors.products = 'All products must have descriptions with valid quantities';
          isValid = false;
          break;
        }
      }
    }

    if (!formData.due_date) {
      formErrors.due_date = 'Due date is required';
      isValid = false;
    }

    return isValid;
  }

  // ========================================
  // CRUD OPERATIONS
  // ========================================

  let rfqSaving = false;
  let rfqDeleting = false;
  let stageUpdating = false;

  async function handleCreateRFQ() {
    if (rfqSaving) return;
    if (!validateForm()) {
      toast.warning('Please fill in all required fields');
      return;
    }

    rfqSaving = true;
    try {
      // Get customer name - CustomerMaster uses business_name field
      const customer = customers.find(c => String(c.id) === formData.customer);
      const customerName = customer?.business_name || customer?.name || 'Unknown';

      // Calculate total value
      const totalValue = formData.products.reduce((sum, item) => {
        return sum + (item.quantity * item.target_price);
      }, 0);

      // Create project description (list of products)
      const project = formData.products.map(item => {
        return `${item.description} (${item.quantity})`;
      }).join(', ');

      // Serialize the per-line products so CostingSheetScreen can seed real
      // line items later (parseOpportunitySeedItems / mapSeedItemsToCosting
      // shape: description, quantity, unit_price, total_price, currency).
      // Target Price is entered in BHD, so currency is explicitly tagged.
      const productDetails = JSON.stringify(formData.products.map(item => ({
        description: item.description,
        quantity: item.quantity,
        unit_price: item.target_price,
        total_price: item.quantity * item.target_price,
        currency: 'BHD'
      })));

      // Create RFQ via backend
      const newRFQ = await CreateRFQ(
        customerName,
        project,
        totalValue,
        formData.notes,
        productDetails
      );

      toast.success(`RFQ created successfully! RFQ-${String(newRFQ.id).padStart(4, '0')}`);

      // Reload data
      await loadData();

      closeCreateModal();

    } catch (err) {
      console.error('Failed to create RFQ:', err);
      toast.danger(`Failed to create RFQ: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      rfqSaving = false;
    }
  }

  async function handleUpdateStage(rfqId: number, newStage: string) {
    if (stageUpdating) return;
    stageUpdating = true;
    try {
      await UpdateRFQStage(rfqId, newStage);
      toast.success(`RFQ stage updated to ${newStage}`);
      await loadData();
    } catch (err) {
      console.error('Failed to update RFQ stage:', err);
      toast.danger(`Failed to update stage: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      stageUpdating = false;
    }
  }

  async function handleDeleteRFQ(rfqId: number) {
    if (rfqDeleting) return;
    if (!(await confirm.ask({
      title: 'Delete RFQ',
      message: 'Are you sure you want to delete this RFQ? This cannot be undone.',
      confirmLabel: 'Delete',
      variant: 'danger'
    }))) {
      return;
    }

    rfqDeleting = true;
    try {
      await DeleteRFQ(rfqId);
      toast.success('RFQ deleted successfully');
      await loadData();
    } catch (err) {
      console.error('Failed to delete RFQ:', err);
      toast.danger(`Failed to delete RFQ: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      rfqDeleting = false;
    }
  }

  // ========================================
  // EVENT HANDLERS
  // ========================================

  function handleRowClick(event: CustomEvent) {
    const rfq = event.detail.row;
    console.log('Selected RFQ:', rfq);
    // Could open a detail modal here
  }

  function handleTableAction(event: MouseEvent) {
    const target = event.target as HTMLElement;
    const button = target.closest('[data-action]') as HTMLButtonElement;

    if (!button) return;

    const action = button.dataset.action;
    const id = parseInt(button.dataset.id || '0', 10);

    const rfq = rfqs.find(r => r.id === id);
    if (!rfq) return;

    event.stopPropagation();

    switch (action) {
      case 'edit':
        openEditModal(rfq);
        break;
      case 'delete':
        handleDeleteRFQ(id);
        break;
    }
  }

  function handleStageChange(stageId: string) {
    activeStage = stageId;
  }
</script>

<PageLayout title="RFQs" subtitle="Request for Quotation Management">
  <!-- @migration-task: migrate this slot by hand, `header-actions` is an invalid identifier -->
  <div slot="header-actions" class="header-actions">
    <Button variant="primary" on:click={openCreateModal}>
      + New RFQ
    </Button>
  </div>

  <div class="rfq-screen">
    <!-- Stats Cards -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">Total RFQs</div>
        <div class="stat-value">{stats.total}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Win Rate</div>
        <div class="stat-value" class:success={stats.winRate >= 56.8} class:warning={stats.winRate < 56.8}>
          {stats.winRate.toFixed(1)}%
        </div>
        <div class="stat-meta">Target: 56.8%</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Total Value</div>
        <div class="stat-value">{stats.totalValue.toLocaleString('en-BH', { minimumFractionDigits: 3 })} BHD</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Avg Deal Size</div>
        <div class="stat-value">{stats.avgDealSize.toLocaleString('en-BH', { minimumFractionDigits: 3 })} BHD</div>
      </div>
    </div>

    <!-- Stage Tabs -->
    <div class="stage-tabs">
      {#each stages as stage}
        <button
          class="stage-tab"
          class:active={activeStage === stage.id}
          onclick={() => handleStageChange(stage.id)}
        >
          <span class="stage-label">{stage.label}</span>
          <span class="stage-count">{stage.count}</span>
        </button>
      {/each}
    </div>

    <!-- Search & Filters -->
    <div class="toolbar">
      <Input
        type="search"
        placeholder="Search RFQs..."
        bind:value={searchQuery}
      />
    </div>

    <!-- RFQ Table -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="table-wrapper" onclick={handleTableAction}>
      {#if loading}
        <div class="loading-state">
          <WabiSpinner size="lg" />
          <p>Loading RFQs...</p>
        </div>
      {:else if error}
        <div class="error-state">
          <p class="error-message">{error}</p>
          <Button variant="secondary" on:click={loadData}>Retry</Button>
        </div>
      {:else}
        <DataTable
          {columns}
          data={filteredRFQs}
          loading={false}
          emptyMessage="No RFQs yet — enquiries you log land here."
          onRowClick={handleRowClick}
          stickyHeader={true}
          maxHeight="600px"
        />
      {/if}
    </div>
  </div>
</PageLayout>

<!-- Create RFQ Modal -->
<WabiModal
  bind:open={showCreateModal}
  title="Create New RFQ"
  size="lg"
  on:close={closeCreateModal}
>
  <div class="modal-form">
    <div class="form-row">
      <Select
        label="Customer"
        options={customerOptions}
        bind:value={formData.customer}
        error={formErrors.customer}
        required={true}
        searchable={true}
        placeholder="Select customer..."
      />
    </div>

    <div class="form-section">
      <div class="section-header">
        <h3>Products</h3>
        <Button variant="ghost" size="sm" on:click={addLineItem}>
          + Add Product
        </Button>
      </div>

      {#if formErrors.products}
        <p class="error-text">{formErrors.products}</p>
      {/if}

      {#each formData.products as item, index}
        <div class="line-item" transition:fade>
          <div class="line-item-fields">
            <Input
              label="Product Description"
              bind:value={item.description}
              required={true}
              placeholder="Enter product or service description..."
            />
            <Input
              type="number"
              label="Quantity"
              bind:value={item.quantity}
              required={true}
              min="1"
            />
            <CurrencyInput
              label="Target Price (BHD)"
              bind:value={item.target_price}
              required={true}
            />
          </div>
          <button class="btn-remove" onclick={() => removeLineItem(index)} title="Remove product">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M12 4L4 12M4 4L12 12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
            </svg>
          </button>
        </div>
      {/each}

      {#if formData.products.length === 0}
        <p class="empty-hint">Click "Add Product" to add line items</p>
      {/if}
    </div>

    <div class="form-row">
      <DatePicker
        label="Due Date"
        bind:value={formData.due_date}
        error={formErrors.due_date}
        required={true}
      />
    </div>

    <div class="form-row">
      <Textarea
        label="Notes"
        bind:value={formData.notes}
        placeholder="Additional notes or requirements..."
        rows={3}
      />
    </div>
  </div>

  {#snippet footer()}
    <div  class="modal-actions">
      <Button variant="ghost" on:click={closeCreateModal}>Cancel</Button>
      <Button variant="primary" on:click={handleCreateRFQ}>Create RFQ</Button>
    </div>
  {/snippet}
</WabiModal>

<!-- Edit RFQ Modal (simplified for now) -->
<WabiModal
  bind:open={showEditModal}
  title="Edit RFQ"
  size="lg"
  on:close={closeEditModal}
>
  {#if editingRFQ}
    <div class="modal-form">
      <p class="info-text">Editing RFQ-{String(editingRFQ.id).padStart(4, '0')}</p>

      <div class="form-row">
        <Select
          label="Update Stage"
          options={[
            { value: 'Pending', label: 'Pending' },
            { value: 'Qualified', label: 'Qualified' },
            { value: 'Proposal', label: 'Proposal' },
            { value: 'Negotiation', label: 'Negotiation' },
          ]}
          value={editingRFQ.status}
          on:change={(e) => handleUpdateStage(editingRFQ.id, e.detail.value)}
        />
      </div>

      <p class="info-hint">Won/Lost status is set automatically when the linked offer is marked as Won or Lost in the Offers screen.</p>
    </div>
  {/if}

  {#snippet footer()}
    <div  class="modal-actions">
      <Button variant="ghost" on:click={closeEditModal}>Close</Button>
    </div>
  {/snippet}
</WabiModal>

<style>
  :global(.unknown-company) {
    color: var(--steel, #86868B);
    font-style: italic;
    font-size: 13px;
  }

  .rfq-screen {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  /* ========================================
     STATS GRID
     ======================================== */
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
  }

  .stat-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius);
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .stat-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .stat-value {
    font-size: 28px;
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
  }

  .stat-value.success {
    color: #10B981;
  }

  .stat-value.warning {
    color: #F59E0B;
  }

  .stat-meta {
    font-size: 12px;
    color: var(--text-muted);
  }

  /* ========================================
     STAGE TABS
     ======================================== */
  .stage-tabs {
    display: flex;
    gap: 4px;
    background: var(--surface-elevated);
    padding: 4px;
    border-radius: var(--border-radius);
    overflow-x: auto;
  }

  .stage-tab {
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

  .stage-tab:hover {
    background: var(--interactive-hover);
    color: var(--text-primary);
  }

  .stage-tab.active {
    background: var(--brand-indigo);
    color: white;
  }

  .stage-label {
    font-weight: 500;
  }

  .stage-count {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 20px;
    height: 20px;
    padding: 0 6px;
    font-size: 11px;
    font-weight: 600;
    background: rgba(0, 0, 0, 0.1);
    border-radius: 10px;
  }

  .stage-tab.active .stage-count {
    background: rgba(255, 255, 255, 0.2);
  }

  /* ========================================
     TOOLBAR
     ======================================== */
  .toolbar {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  /* ========================================
     TABLE WRAPPER
     ======================================== */
  .table-wrapper {
    background: var(--surface);
    border-radius: var(--border-radius);
    overflow: hidden;
  }

  .loading-state,
  .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 48px 24px;
    gap: 16px;
  }

  .loading-state p,
  .error-state p {
    color: var(--text-secondary);
  }

  /* ========================================
     MODAL FORM
     ======================================== */
  .modal-form {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .form-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .section-header h3 {
    font-size: 14px;
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin: 0;
  }

  .line-item {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    padding: 12px;
    background: var(--surface-elevated);
    border-radius: var(--border-radius-sm);
  }

  .line-item-fields {
    flex: 1;
    display: grid;
    grid-template-columns: 2fr 1fr 1fr;
    gap: 12px;
  }

  .btn-remove {
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    margin-top: 22px; /* Align with input fields */
    border: none;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all var(--transition-fast);
  }

  .btn-remove:hover {
    background: rgba(220, 38, 38, 0.1);
    color: #DC2626;
  }

  .empty-hint {
    text-align: center;
    color: var(--text-muted);
    font-style: italic;
    padding: 24px;
  }

  .error-text {
    color: #DC2626;
    font-size: 12px;
    margin: 0;
  }

  .info-text {
    color: var(--text-secondary);
    font-size: 14px;
    margin: 0 0 16px 0;
  }

  .info-hint {
    text-align: center;
    color: var(--text-muted);
    font-style: italic;
    margin-top: 16px;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  /* ========================================
     ACTION BUTTONS (in table)
     ======================================== */
  :global(.action-buttons) {
    display: flex;
    gap: 4px;
    justify-content: flex-end;
  }

  :global(.btn-action) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border: none;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    border-radius: 4px;
    transition: all var(--transition-fast);
  }

  :global(.btn-action:hover) {
    background: var(--interactive-hover);
    color: var(--text-primary);
  }

  :global(.btn-action svg) {
    width: 16px;
    height: 16px;
  }

  /* ========================================
     RESPONSIVE
     ======================================== */
  @media (max-width: 768px) {
    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .line-item-fields {
      grid-template-columns: 1fr;
    }

    .stage-tabs {
      overflow-x: auto;
      -webkit-overflow-scrolling: touch;
    }
  }
</style>
