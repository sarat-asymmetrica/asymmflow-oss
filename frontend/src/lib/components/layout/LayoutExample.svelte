<script lang="ts">
  /**
   * Layout Components Example
   * Demonstrates all layout components from the design system
   * This file is for testing/reference only
   */
  import PageLayout from './PageLayout.svelte';
  import ModuleLayout from './ModuleLayout.svelte';
  import SplitView from './SplitView.svelte';
  import Modal from './Modal.svelte';
  import Sidebar from './Sidebar.svelte';
  import { Button, Card, KPICard } from '../ui';

  let activeExample: 'page' | 'module' | 'split' | 'modal' | 'sidebar' = $state('page');
  let activeTab = $state('inbox');
  let showModal = $state(false);
  let sidebarCollapsed = $state(false);
  let selectedCustomer: number | null = $state(null);

  const tabs = [
    { id: 'inbox', label: 'Inbox', count: 12 },
    { id: 'opportunities', label: 'Opportunities', count: 8 },
    { id: 'offers', label: 'Offers', count: 5 },
  ];

  const customers = [
    { id: 1, name: 'Acme Corp', lastContact: '2 days ago' },
    { id: 2, name: 'TechStart Inc', lastContact: '1 week ago' },
    { id: 3, name: 'Global Industries', lastContact: '3 days ago' },
  ];

  function handleTabChange(event: CustomEvent<string>) {
    activeTab = event.detail;
  }
</script>

<div class="example-container">
  <!-- Example Selector -->
  <div class="example-selector">
    <h1 class="page-title">Layout Components Gallery</h1>
    <div class="selector-buttons">
      <Button variant={activeExample === 'page' ? 'primary' : 'secondary'} on:click={() => activeExample = 'page'}>
        PageLayout
      </Button>
      <Button variant={activeExample === 'module' ? 'primary' : 'secondary'} on:click={() => activeExample = 'module'}>
        ModuleLayout
      </Button>
      <Button variant={activeExample === 'split' ? 'primary' : 'secondary'} on:click={() => activeExample = 'split'}>
        SplitView
      </Button>
      <Button variant={activeExample === 'modal' ? 'primary' : 'secondary'} on:click={() => showModal = true}>
        Modal
      </Button>
      <Button variant={activeExample === 'sidebar' ? 'primary' : 'secondary'} on:click={() => activeExample = 'sidebar'}>
        Sidebar
      </Button>
    </div>
  </div>

  <!-- Example Display Area -->
  <div class="example-display">
    {#if activeExample === 'page'}
      <PageLayout title="Dashboard" subtitle="Your business at a glance">
        <!-- @migration-task: migrate this slot by hand, `header-actions` is an invalid identifier -->
  <svelte:fragment slot="header-actions">
          <Button variant="primary">Create Report</Button>
        </svelte:fragment>

        <div class="grid grid-4">
          <KPICard label="Revenue" value="$2.4M" meta="YTD, +12% from target" trend="up" />
          <KPICard label="Orders" value="342" meta="This month" trend="up" />
          <KPICard label="Customers" value="128" meta="Active" />
          <KPICard label="Pending" value="23" meta="Requires attention" accent />
        </div>

        <Card title="Recent Activity">
          <p class="text-secondary">Latest transactions and updates will appear here.</p>
        </Card>
      </PageLayout>

    {:else if activeExample === 'module'}
      <ModuleLayout title="Sales Hub" {tabs} {activeTab} on:tabChange={handleTabChange}>
        <!-- @migration-task: migrate this slot by hand, `header-actions` is an invalid identifier -->
  <svelte:fragment slot="header-actions">
          <Button variant="secondary">Filter</Button>
          <Button variant="primary">New RFQ</Button>
        </svelte:fragment>

        <Card>
          <h3 class="section-title">{tabs.find(t => t.id === activeTab)?.label}</h3>
          <p class="text-secondary">Content for {activeTab} tab will appear here.</p>
          <p class="meta">This demonstrates the standard module layout with header, tabs, and content area.</p>
        </Card>
      </ModuleLayout>

    {:else if activeExample === 'split'}
      <div style="height: 600px;">
        <SplitView listWidth="350px">
          {#snippet list()}
                          
              <div class="customer-list">
                <div class="list-header">
                  <h3 class="section-title">Customers</h3>
                </div>
                {#each customers as customer}
                  <button
                    class="customer-item"
                    class:selected={selectedCustomer === customer.id}
                    onclick={() => selectedCustomer = customer.id}
                  >
                    <div class="customer-name">{customer.name}</div>
                    <div class="customer-meta">Last contact: {customer.lastContact}</div>
                  </button>
                {/each}
              </div>
            
                          {/snippet}

          {#snippet detail()}
                          
              {#if selectedCustomer}
                <div class="customer-detail">
                  <h2 class="page-title">
                    {customers.find(c => c.id === selectedCustomer)?.name}
                  </h2>
                  <Card title="Contact Information">
                    <p class="text-secondary">Detailed customer information will appear here.</p>
                  </Card>
                  <Card title="Recent Orders">
                    <p class="text-secondary">Order history and analytics.</p>
                  </Card>
                </div>
              {/if}
            
                          {/snippet}
        </SplitView>
      </div>

    {:else if activeExample === 'sidebar'}
      <div style="height: 600px; position: relative;">
        <Sidebar currentScreen="dashboard" collapsed={sidebarCollapsed} />
        <div style="margin-left: {sidebarCollapsed ? '60px' : '240px'}; padding: 16px; transition: margin-left 200ms;">
          <h2 class="page-title">Sidebar Navigation</h2>
          <p class="text-secondary">The sidebar supports nested navigation, icons, and collapsible state.</p>
          <Button variant="secondary" on:click={() => sidebarCollapsed = !sidebarCollapsed}>
            Toggle Collapse
          </Button>
        </div>
      </div>
    {/if}
  </div>

  <!-- Modal Example -->
  <Modal bind:open={showModal} title="Example Modal" size="md">
    <p>This is an example modal dialog. It demonstrates:</p>
    <ul>
      <li>Backdrop click to close</li>
      <li>Escape key support</li>
      <li>Focus trapping</li>
      <li>Multiple sizes (sm, md, lg, full)</li>
      <li>Accessible ARIA attributes</li>
    </ul>

    {#snippet footer()}
      
        <Button variant="secondary" on:click={() => showModal = false}>Cancel</Button>
        <Button variant="primary" on:click={() => showModal = false}>Confirm</Button>
      
      {/snippet}
  </Modal>
</div>

<style>
  .example-container {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-height: 100vh;
  }

  .example-selector {
    padding: 16px;
    background: var(--surface);
    border-radius: var(--border-radius);
    border: 1px solid var(--border);
  }

  .selector-buttons {
    display: flex;
    gap: 8px;
    margin-top: 16px;
    flex-wrap: wrap;
  }

  .example-display {
    flex: 1;
    background: var(--bg-base);
    border-radius: var(--border-radius);
    border: 1px solid var(--border);
    overflow: hidden;
  }

  /* Customer List Styles */
  .customer-list {
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  .list-header {
    padding: 16px;
    border-bottom: 1px solid var(--border);
  }

  .customer-item {
    width: 100%;
    padding: 16px;
    border: none;
    border-bottom: 1px solid var(--border);
    background: var(--surface);
    text-align: left;
    cursor: pointer;
    transition: background var(--transition-fast);
  }

  .customer-item:hover {
    background: var(--brand-indigo-tint);
  }

  .customer-item.selected {
    background: var(--indigo-contrast-surface);
    border-left: 3px solid var(--brand-indigo);
  }

  .customer-name {
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 4px;
  }

  .customer-meta {
    font-size: var(--meta-size);
    color: var(--text-muted);
  }

  .customer-detail {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
</style>
