<script lang="ts">
  /**
   * Components Showcase
   * Demonstrates all enterprise UI components with live examples
   */

  import Tabs from './Tabs.svelte';
  import Badge from './Badge.svelte';
  import StatusBadge from './StatusBadge.svelte';
  import Dropdown from './Dropdown.svelte';
  import type { Tab, DropdownOption } from '$lib/types/components';

  // Tabs demo
  let activeTab = $state('inbox');
  const tabItems: Tab[] = [
    { id: 'inbox', label: 'Inbox', count: 12 },
    { id: 'opportunities', label: 'Opportunities', count: 5 },
    { id: 'quotes', label: 'Quotes', count: 3 },
    { id: 'archived', label: 'Archived', disabled: true },
  ];

  // Dropdown demo
  const dropdownOptions: DropdownOption[] = [
    { value: 'edit', label: 'Edit', icon: '' },
    { value: 'duplicate', label: 'Duplicate', icon: '' },
    { value: 'delete', label: 'Delete', icon: '' },
    { value: 'disabled', label: 'Disabled Action', icon: '', disabled: true },
  ];

  function handleTabChange(event: CustomEvent<string>) {
    activeTab = event.detail;
    console.log('Tab changed to:', activeTab);
  }

  function handleDropdownSelect(event: CustomEvent<string>) {
    console.log('Selected:', event.detail);
  }

  // Status examples
  const statuses = [
    'Draft',
    'Sent',
    'Won',
    'Lost',
    'Pending',
    'Paid',
    'Overdue',
    'Processing',
    'Delivered',
    'Rejected',
  ];
</script>

<div class="showcase">
  <h1 class="page-title">UI Components Showcase</h1>

  <!-- Tabs Section -->
  <section class="section">
    <h2 class="section-title">Tabs</h2>

    <div class="demo-group">
      <h3 class="demo-label">Underline Variant (Default)</h3>
      <Tabs tabs={tabItems} {activeTab} variant="underline" on:change={handleTabChange} />
    </div>

    <div class="demo-group">
      <h3 class="demo-label">Pill Variant</h3>
      <Tabs tabs={tabItems} {activeTab} variant="pill" on:change={handleTabChange} />
    </div>
  </section>

  <!-- Badges Section -->
  <section class="section">
    <h2 class="section-title">Badges</h2>

    <div class="demo-group">
      <h3 class="demo-label">Small Size</h3>
      <div class="badge-row">
        <Badge variant="default" size="sm">Default</Badge>
        <Badge variant="success" size="sm">Success</Badge>
        <Badge variant="warning" size="sm">Warning</Badge>
        <Badge variant="danger" size="sm">Danger</Badge>
        <Badge variant="info" size="sm">Info</Badge>
      </div>
    </div>

    <div class="demo-group">
      <h3 class="demo-label">Medium Size</h3>
      <div class="badge-row">
        <Badge variant="default" size="md">Default</Badge>
        <Badge variant="success" size="md">Success</Badge>
        <Badge variant="warning" size="md">Warning</Badge>
        <Badge variant="danger" size="md">Danger</Badge>
        <Badge variant="info" size="md">Info</Badge>
      </div>
    </div>
  </section>

  <!-- Status Badges Section -->
  <section class="section">
    <h2 class="section-title">Status Badges (Smart Mapping)</h2>

    <div class="demo-group">
      <h3 class="demo-label">Automatic Color Assignment</h3>
      <div class="badge-grid">
        {#each statuses as status}
          <div class="status-example">
            <StatusBadge {status} />
            <span class="meta">"{status}"</span>
          </div>
        {/each}
      </div>
    </div>
  </section>

  <!-- Dropdown Section -->
  <section class="section">
    <h2 class="section-title">Dropdowns</h2>

    <div class="demo-group">
      <h3 class="demo-label">Click Trigger</h3>
      <div class="dropdown-row">
        <Dropdown options={dropdownOptions} trigger="click" align="left" on:select={handleDropdownSelect} />
        <Dropdown options={dropdownOptions} trigger="click" align="right" on:select={handleDropdownSelect}>
          <!-- @migration-task: migrate this slot by hand, `trigger` would shadow a prop on the parent component -->
  <button slot="trigger" class="btn btn-primary">Right Aligned</button>
        </Dropdown>
      </div>
    </div>

    <div class="demo-group">
      <h3 class="demo-label">Hover Trigger</h3>
      <Dropdown options={dropdownOptions} trigger="hover" align="left" on:select={handleDropdownSelect}>
        <!-- @migration-task: migrate this slot by hand, `trigger` would shadow a prop on the parent component -->
  <button slot="trigger" class="btn btn-secondary">Hover Me</button>
      </Dropdown>
    </div>

    <div class="demo-group">
      <h3 class="demo-label">Disabled State</h3>
      <Dropdown options={dropdownOptions} trigger="click" align="left" disabled />
    </div>
  </section>

  <!-- Usage Examples -->
  <section class="section">
    <h2 class="section-title">Real-World Usage</h2>

    <div class="card">
      <div class="card-header">
        <h3 class="card-title">Recent Invoices</h3>
        <Dropdown options={dropdownOptions} trigger="click" align="right" on:select={handleDropdownSelect}>
          <!-- @migration-task: migrate this slot by hand, `trigger` would shadow a prop on the parent component -->
  <button slot="trigger" class="btn btn-ghost">
            ⋯
          </button>
        </Dropdown>
      </div>

      <table class="data-table">
        <thead>
          <tr>
            <th>Invoice #</th>
            <th>Customer</th>
            <th>Amount</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>#INV-001</td>
            <td>Acme Corp</td>
            <td>$5,000</td>
            <td><StatusBadge status="Paid" size="sm" /></td>
          </tr>
          <tr>
            <td>#INV-002</td>
            <td>TechStart Inc</td>
            <td>$3,500</td>
            <td><StatusBadge status="Pending" size="sm" /></td>
          </tr>
          <tr>
            <td>#INV-003</td>
            <td>Global Traders</td>
            <td>$8,200</td>
            <td><StatusBadge status="Overdue" size="sm" /></td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</div>

<style>
  .showcase {
    padding: var(--page-padding);
    max-width: 1200px;
    margin: 0 auto;
  }

  .section {
    margin-bottom: 40px;
  }

  .demo-group {
    margin-bottom: 24px;
  }

  .demo-label {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-secondary);
    margin-bottom: 12px;
  }

  .badge-row {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
  }

  .badge-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 16px;
  }

  .status-example {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .dropdown-row {
    display: flex;
    gap: 16px;
    flex-wrap: wrap;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .card-title {
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    color: var(--text-primary);
  }
</style>
