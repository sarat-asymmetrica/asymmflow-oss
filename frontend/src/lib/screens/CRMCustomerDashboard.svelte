<script lang="ts">
  import { createBubbler, stopPropagation } from 'svelte/legacy';

  const bubble = createBubbler();
  import { onMount, createEventDispatcher } from 'svelte';
  import { GetCRMCustomerDashboard } from '../../../wailsjs/go/main/App';
import { GetCRMCustomerDashboardByYear } from '../../../wailsjs/go/main/InfraService';
import { CreateCustomer } from '../../../wailsjs/go/main/CRMService';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import { toast } from '$lib/stores/toasts';
  import { formatNumber } from '$lib/utils/formatters';

  const dispatch = createEventDispatcher();

  let loading = $state(true);
  let dashboard: any = $state(null);
  let searchQuery = $state('');
  let gradeFilter = $state('All');
  const currentYear = new Date().getFullYear();
  const earliestDashboardYear = 2023;
  let selectedYear = $state(currentYear);
  const availableYears = [0, ...Array.from({ length: Math.max(currentYear - earliestDashboardYear + 1, 1) }, (_, index) => currentYear - index)];
  let showCreateModal = $state(false);

  function emptyCustomerForm() {
    return {
      business_name: '',
      customer_code: '',
      customer_type: '',
      address_line1: '',
      city: '',
      country: '',
      trn: '',
      industry: '',
      payment_grade: '',
      relation_years: 0
    };
  }

  let newCustomer = $state(emptyCustomerForm());

  async function loadDashboard() {
    loading = true;
    try {
      if (selectedYear === 0) {
        dashboard = await GetCRMCustomerDashboard();
      } else {
        dashboard = await GetCRMCustomerDashboardByYear(selectedYear);
      }
      console.log('Customer Dashboard loaded:', dashboard);
    } catch (err) {
      console.error('Failed to load customer dashboard:', err);
      toast.danger('Failed to load customer dashboard');
    } finally {
      loading = false;
    }
  }

  function handleYearChange() {
    loadDashboard();
  }

  function selectCustomer(customerId: string) {
    dispatch('select', { id: customerId });
  }

  function activateCustomer(customerId: string, event: KeyboardEvent) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      selectCustomer(customerId);
    }
  }

  async function saveCustomer() {
    if (!newCustomer.business_name.trim()) {
      toast.warning('Please enter business name');
      return;
    }
    try {
      await CreateCustomer(newCustomer as any);
      toast.success('Customer created successfully');
      showCreateModal = false;
      resetForm();
      await loadDashboard();
    } catch (err) {
      console.error('Failed to create customer:', err);
      toast.danger('Failed to create customer');
    }
  }

  function resetForm() {
    newCustomer = emptyCustomerForm();
  }

  function formatCurrency(value: number): string {
    return formatNumber(value || 0, 0);
  }

  let filteredCustomers = $derived(dashboard?.customers?.filter((c: any) => {
    const matchSearch = !searchQuery ||
      c.business_name?.toLowerCase().includes(searchQuery.toLowerCase());
    const matchGrade = gradeFilter === 'All' || c.payment_grade === gradeFilter;
    return matchSearch && matchGrade;
  }) || []);

  onMount(loadDashboard);
</script>

<div class="dashboard">
  {#if loading}
    <div class="loading-state">
      <WabiSpinner size="lg" />
    </div>
  {:else if dashboard}
    <!-- Year Selector -->
    <div class="year-selector-row">
      <div class="year-selector">
        <label class="year-label" for="customer-dashboard-year">FINANCIAL YEAR</label>
        <select id="customer-dashboard-year" bind:value={selectedYear} onchange={handleYearChange} class="year-select">
          {#each availableYears as yr}
            <option value={yr}>{yr === 0 ? 'All Years' : yr}</option>
          {/each}
        </select>
      </div>
    </div>

    <!-- KPI Row -->
    <div class="kpi-row">
      <Card padding="md">
        <div class="kpi">
          <div class="kpi-label">CUSTOMERS</div>
          <div class="kpi-value">{dashboard.total_customers}</div>
          <div class="kpi-subtext">{dashboard.active_customers} active</div>
        </div>
      </Card>
      <Card padding="md">
        <div class="kpi">
          <div class="kpi-label">YTD BUSINESS</div>
          <div class="kpi-value">{formatCurrency(dashboard.total_revenue)} BHD</div>
          <div class="kpi-subtext" class:negative={dashboard.revenue_yoy < 0}>
            {dashboard.revenue_yoy > 0 ? '+' : ''}{dashboard.revenue_yoy?.toFixed(1) || 0}% YoY
          </div>
        </div>
      </Card>
      <Card padding="md">
        <div class="kpi">
          <div class="kpi-label">OPEN EXPOSURE</div>
          <div class="kpi-value">{formatCurrency(dashboard.total_outstanding)} BHD</div>
          <div class="kpi-subtext">AR + uninvoiced orders</div>
        </div>
      </Card>
      <Card padding="md">
        <div class="kpi">
          <div class="kpi-label">OVERDUE</div>
          <div class="kpi-value">{formatCurrency(dashboard.overdue_amount)} BHD</div>
          <div class="kpi-subtext" class:warning={dashboard.overdue_pct > 20}>
            {dashboard.overdue_pct?.toFixed(1) || 0}%
          </div>
        </div>
      </Card>
    </div>

    <!-- Analytics Row -->
    <div class="analytics-row">
      <!-- Top 10 Customers -->
      <Card padding="md">
        <h3 class="section-title">TOP 10 BY BUSINESS</h3>
        <div class="top-list">
          {#each dashboard.top_customers || [] as customer, i}
            <div class="top-item" role="button" tabindex="0" onclick={() => selectCustomer(customer.id)} onkeydown={(event) => activateCustomer(customer.id, event)}>
              <span class="rank">{i + 1}</span>
              <span class="name">{customer.business_name}</span>
              <div class="bar-container">
                <div
                  class="bar"
                  style="width: {dashboard.total_revenue > 0 ? (customer.total_revenue / dashboard.total_revenue * 100) : 0}%"
                ></div>
              </div>
              <span class="amount">{formatCurrency(customer.total_revenue)}</span>
            </div>
          {/each}
        </div>
      </Card>

      <!-- Concentration Risk -->
      <Card padding="md">
        <h3 class="section-title">CONCENTRATION RISK</h3>
        <div class="risk-metrics">
          <div class="risk-item">
            <span class="risk-label">Top 3 Customers</span>
            <span class="risk-value" class:warning={dashboard.top3_revenue_pct > 50}>
              {dashboard.top3_revenue_pct?.toFixed(0) || 0}%
            </span>
          </div>
          <div class="risk-item">
            <span class="risk-label">Top 5 Customers</span>
            <span class="risk-value" class:warning={dashboard.top5_revenue_pct > 70}>
              {dashboard.top5_revenue_pct?.toFixed(0) || 0}%
            </span>
          </div>
          <div class="risk-item">
            <span class="risk-label">Top 10 Customers</span>
            <span class="risk-value" class:danger={dashboard.top10_revenue_pct > 90}>
              {dashboard.top10_revenue_pct?.toFixed(0) || 0}%
            </span>
          </div>
        </div>
      </Card>

      <!-- Grade Distribution -->
      <Card padding="md">
        <h3 class="section-title">GRADE DISTRIBUTION</h3>
        <div class="grade-grid">
          <div class="grade-item grade-a">
            <span class="grade-letter">A</span>
            <span class="grade-count">{dashboard.grade_a_count}</span>
            <span class="grade-revenue">{formatCurrency(dashboard.grade_a_revenue)}</span>
          </div>
          <div class="grade-item grade-b">
            <span class="grade-letter">B</span>
            <span class="grade-count">{dashboard.grade_b_count}</span>
            <span class="grade-revenue">{formatCurrency(dashboard.grade_b_revenue)}</span>
          </div>
          <div class="grade-item grade-c">
            <span class="grade-letter">C</span>
            <span class="grade-count">{dashboard.grade_c_count}</span>
            <span class="grade-revenue">{formatCurrency(dashboard.grade_c_revenue)}</span>
          </div>
          <div class="grade-item grade-d">
            <span class="grade-letter">D</span>
            <span class="grade-count">{dashboard.grade_d_count}</span>
            <span class="grade-revenue">{formatCurrency(dashboard.grade_d_revenue)}</span>
          </div>
        </div>
      </Card>
    </div>

    <!-- Customer Cards Section -->
    <Card padding="sm">
      <div class="customers-header">
        <h3 class="section-title">ALL CUSTOMERS</h3>
        <div class="filters">
          <input
            type="text"
            placeholder="Search customers..."
            bind:value={searchQuery}
            class="search-input"
          />
          <select bind:value={gradeFilter} class="grade-select">
            <option value="All">All Grades</option>
            <option value="A">Grade A</option>
            <option value="B">Grade B</option>
            <option value="C">Grade C</option>
            <option value="D">Grade D</option>
          </select>
          <Button variant="primary" on:click={() => { resetForm(); showCreateModal = true; }}>+ New Customer</Button>
        </div>
      </div>

      <div class="customer-grid">
        {#each filteredCustomers as customer}
          <div class="customer-card" role="button" tabindex="0" onclick={() => selectCustomer(customer.id)} onkeydown={(event) => activateCustomer(customer.id, event)}>
            <div class="card-header">
              <span class="customer-name">{customer.business_name}</span>
              <span class="grade-badge grade-{customer.payment_grade?.toLowerCase()}">{customer.payment_grade}</span>
            </div>
            <div class="card-type">{customer.customer_type}</div>
            <div class="card-metrics">
              <div class="metric">
                <span class="metric-value">{customer.active_invoices || 0}</span>
                <span class="metric-label">invoices</span>
              </div>
              <div class="metric">
                <span class="metric-value">{formatCurrency(customer.outstanding_bhd || 0)}</span>
                <span class="metric-label">exposure</span>
              </div>
            </div>
            {#if customer.city}
              <div class="card-city">{customer.city}</div>
            {/if}
          </div>
        {/each}
      </div>
    </Card>
  {/if}
</div>

{#if showCreateModal}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-backdrop" onclick={() => showCreateModal = false}>
    <div class="modal" onclick={stopPropagation(bubble('click'))}>
      <h3>Create New Customer</h3>
      <div class="form-grid">
        <div class="form-group">
          <label for="crm-customer-business-name">Business Name <span class="required">*</span></label>
          <input id="crm-customer-business-name" type="text" bind:value={newCustomer.business_name} placeholder="Company name" />
        </div>
        <div class="form-group">
          <label for="crm-customer-code">Customer Code</label>
          <input id="crm-customer-code" type="text" bind:value={newCustomer.customer_code} placeholder="Auto-generated if empty" />
        </div>
        <div class="form-group">
          <label for="crm-customer-type">Customer Type</label>
          <select id="crm-customer-type" bind:value={newCustomer.customer_type}>
            <option value="" disabled>Select type</option>
            <option value="Corporate">Corporate</option>
            <option value="Government">Government</option>
            <option value="SME">SME</option>
            <option value="Individual">Individual</option>
          </select>
        </div>
        <div class="form-group">
          <label for="crm-customer-grade">Payment Grade</label>
          <select id="crm-customer-grade" bind:value={newCustomer.payment_grade}>
            <option value="" disabled>Select grade</option>
            <option value="A">Grade A</option>
            <option value="B">Grade B</option>
            <option value="C">Grade C</option>
            <option value="D">Grade D</option>
          </select>
        </div>
        <div class="form-group full-width">
          <label for="crm-customer-address">Address</label>
          <input id="crm-customer-address" type="text" bind:value={newCustomer.address_line1} placeholder="Street address" />
        </div>
        <div class="form-group">
          <label for="crm-customer-city">City</label>
          <input id="crm-customer-city" type="text" bind:value={newCustomer.city} placeholder="City" />
        </div>
        <div class="form-group">
          <label for="crm-customer-country">Country</label>
          <input id="crm-customer-country" type="text" bind:value={newCustomer.country} placeholder="Country" />
        </div>
        <div class="form-group">
          <label for="crm-customer-trn">TRN</label>
          <input id="crm-customer-trn" type="text" bind:value={newCustomer.trn} placeholder="Tax registration number" />
        </div>
        <div class="form-group">
          <label for="crm-customer-industry">Industry</label>
          <input id="crm-customer-industry" type="text" bind:value={newCustomer.industry} placeholder="Industry sector" />
        </div>
      </div>
      <div class="modal-actions">
        <Button variant="ghost" on:click={() => showCreateModal = false}>Cancel</Button>
        <Button variant="primary" on:click={saveCustomer}>Create Customer</Button>
      </div>
    </div>
  </div>
{/if}

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 16px;
  }

  .loading-state {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 400px;
  }

  /* Year Selector */
  .year-selector-row {
    display: flex;
    justify-content: flex-end;
    margin-bottom: -8px;
  }

  .year-selector {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .year-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    font-weight: 600;
  }

  .year-select {
    padding: 6px 12px;
    border: 1px solid var(--border);
    border-radius: 6px;
    font-size: 14px;
    font-weight: 600;
    background: var(--surface);
    color: var(--text-primary);
    cursor: pointer;
    min-width: 120px;
  }

  .year-select:focus {
    border-color: var(--brand-indigo);
    outline: none;
  }

  /* KPI Row */
  .kpi-row {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
  }

  .kpi {
    text-align: center;
  }

  .kpi-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }

  .kpi-value {
    font-size: 28px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .kpi-subtext {
    font-size: 12px;
    color: var(--text-muted);
    margin-top: 4px;
  }

  .kpi-subtext.negative {
    color: #EF4444;
  }

  .kpi-subtext.warning {
    color: #F59E0B;
  }

  /* Analytics Row */
  .analytics-row {
    display: grid;
    grid-template-columns: 2fr 1fr 1fr;
    gap: 16px;
  }

  .section-title {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    margin: 0 0 16px 0;
    font-weight: 600;
  }

  /* Top 10 List */
  .top-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .top-item {
    display: grid;
    grid-template-columns: 24px 1fr 100px 60px;
    align-items: center;
    gap: 8px;
    padding: 4px 0;
    cursor: pointer;
    border-radius: 4px;
  }

  .top-item:hover {
    background: var(--interactive-hover);
  }

  .rank {
    font-size: 12px;
    color: var(--text-muted);
    text-align: center;
  }

  .name {
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .bar-container {
    height: 8px;
    background: var(--border);
    border-radius: 4px;
    overflow: hidden;
  }

  .bar {
    height: 100%;
    background: var(--brand-indigo);
    border-radius: 4px;
  }

  .amount {
    font-size: 12px;
    font-weight: 600;
    text-align: right;
    font-family: 'JetBrains Mono', monospace;
  }

  /* Concentration Risk */
  .risk-metrics {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .risk-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .risk-label {
    font-size: 13px;
    color: var(--text-secondary);
  }

  .risk-value {
    font-size: 16px;
    font-weight: 600;
  }

  .risk-value.warning {
    color: #F59E0B;
  }

  .risk-value.danger {
    color: #EF4444;
  }

  /* Grade Distribution */
  .grade-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }

  .grade-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 12px;
    border-radius: 8px;
    background: var(--surface-elevated);
  }

  .grade-letter {
    font-size: 18px;
    font-weight: 700;
  }

  .grade-count {
    font-size: 14px;
    color: var(--text-secondary);
  }

  .grade-revenue {
    font-size: 12px;
    color: var(--text-muted);
    font-family: 'JetBrains Mono', monospace;
  }

  .grade-a .grade-letter { color: #10B981; }
  .grade-b .grade-letter { color: #3B82F6; }
  .grade-c .grade-letter { color: #F59E0B; }
  .grade-d .grade-letter { color: #EF4444; }

  /* Customers Header */
  .customers-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .filters {
    display: flex;
    gap: 12px;
  }

  .search-input, .grade-select {
    padding: 8px 12px;
    border: 1px solid var(--border);
    border-radius: 6px;
    font-size: 14px;
    background: var(--surface);
    color: var(--text-primary);
  }

  .search-input {
    width: 200px;
  }

  /* Customer Grid */
  .customer-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    gap: 16px;
    max-height: calc(100vh - 600px);
    overflow-y: auto;
    padding: 4px;
  }

  .customer-card {
    background: var(--surface-elevated);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 16px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .customer-card:hover {
    border-color: var(--brand-indigo);
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.1);
    transform: translateY(-2px);
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 8px;
  }

  .customer-name {
    font-weight: 600;
    font-size: 14px;
    line-height: 1.3;
  }

  .grade-badge {
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
  }

  .grade-badge.grade-a { background: #DCFCE7; color: #166534; }
  .grade-badge.grade-b { background: #DBEAFE; color: #1E40AF; }
  .grade-badge.grade-c { background: #FEF9C3; color: #854D0E; }
  .grade-badge.grade-d { background: #FEE2E2; color: #991B1B; }

  .card-type {
    font-size: 12px;
    color: var(--text-muted);
    margin-bottom: 12px;
  }

  .card-metrics {
    display: flex;
    gap: 16px;
    margin-bottom: 8px;
  }

  .metric {
    display: flex;
    flex-direction: column;
  }

  .metric-value {
    font-size: 16px;
    font-weight: 600;
    font-family: 'JetBrains Mono', monospace;
  }

  .metric-label {
    font-size: 10px;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .card-city {
    font-size: 11px;
    color: var(--text-muted);
  }

  /* Responsive */
  @media (max-width: 1200px) {
    .analytics-row {
      grid-template-columns: 1fr;
    }

    .kpi-row {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  /* Modal styles */
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: var(--surface);
    padding: 24px;
    border-radius: 12px;
    width: 600px;
    max-height: 80vh;
    overflow-y: auto;
  }

  .modal h3 {
    margin: 0 0 20px 0;
    font-size: 18px;
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
    margin-bottom: 20px;
  }

  .form-group {
    display: flex;
    flex-direction: column;
  }

  .form-group.full-width {
    grid-column: 1 / -1;
  }

  .form-group label {
    font-size: 12px;
    text-transform: uppercase;
    color: var(--text-muted);
    margin-bottom: 4px;
  }

  .required {
    color: #EF4444;
  }

  .form-group input,
  .form-group select {
    padding: 8px 12px;
    border: 1px solid var(--border);
    border-radius: 6px;
    font-size: 14px;
    background: var(--surface);
    color: var(--text-primary);
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 20px;
  }
</style>
