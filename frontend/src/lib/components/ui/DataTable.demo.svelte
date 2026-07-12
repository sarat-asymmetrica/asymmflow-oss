<script lang="ts">
  /**
   * DataTable Component Demo
   * Showcases all features: sorting, loading, empty state, column types, etc.
   */

  import DataTable from './DataTable.svelte';

  // Sample data for invoices
  const invoiceData = [
    { id: 1, invoiceNumber: 'INV-2026-001', customer: 'ACME Corp', amount: 15750.500, status: 'Approved', date: '2026-01-15', items: 12 },
    { id: 2, invoiceNumber: 'INV-2026-002', customer: 'TechStart Ltd', amount: 8900.250, status: 'Pending', date: '2026-01-16', items: 8 },
    { id: 3, invoiceNumber: 'INV-2026-003', customer: 'Global Industries', amount: 42500.750, status: 'Approved', date: '2026-01-17', items: 24 },
    { id: 4, invoiceNumber: 'INV-2026-004', customer: 'BuildCo', amount: 6200.000, status: 'Draft', date: '2026-01-18', items: 5 },
    { id: 5, invoiceNumber: 'INV-2026-005', customer: 'MediaWorks', amount: 19800.500, status: 'Rejected', date: '2026-01-19', items: 15 },
    { id: 6, invoiceNumber: 'INV-2026-006', customer: 'ACME Corp', amount: 11200.250, status: 'Approved', date: '2026-01-20', items: 9 },
    { id: 7, invoiceNumber: 'INV-2026-007', customer: 'StartupXYZ', amount: 3400.500, status: 'Pending', date: '2026-01-21', items: 4 },
    { id: 8, invoiceNumber: 'INV-2026-008', customer: 'Enterprise Inc', amount: 67500.750, status: 'Approved', date: '2026-01-22', items: 38 },
  ];

  // Column definitions showcasing all types
  const invoiceColumns = [
    {
      key: 'invoiceNumber',
      label: 'Invoice #',
      sortable: true,
      type: 'text',
      width: '140px'
    },
    {
      key: 'customer',
      label: 'Customer',
      sortable: true,
      type: 'text'
    },
    {
      key: 'date',
      label: 'Date',
      sortable: true,
      type: 'date',
      width: '120px'
    },
    {
      key: 'items',
      label: 'Items',
      sortable: true,
      type: 'number',
      align: 'right',
      width: '80px'
    },
    {
      key: 'amount',
      label: 'Amount',
      sortable: true,
      type: 'currency',
      align: 'right',
      width: '160px'
    },
    {
      key: 'status',
      label: 'Status',
      sortable: true,
      type: 'status',
      width: '120px'
    },
    {
      key: 'actions',
      label: 'Actions',
      type: 'actions',
      align: 'center',
      width: '100px',
      render: (row: any) => `
        <button class="action-btn" data-id="${row.id}" onclick="alert('View ${row.invoiceNumber}')">View</button>
      `
    }
  ];

  // State
  let selectedId: string | undefined = $state(undefined);
  let loading = $state(false);
  let showEmpty = $state(false);

  // Handlers
  function handleRowClick(row: any) {
    selectedId = row.id;
    console.log('Row clicked:', row);
  }

  function handleSort(event: CustomEvent) {
    console.log('Sort changed:', event.detail);
  }

  function simulateLoading() {
    loading = true;
    setTimeout(() => {
      loading = false;
    }, 2000);
  }

  function toggleEmpty() {
    showEmpty = !showEmpty;
  }
</script>

<div class="demo-container">
  <div class="demo-header">
    <h2>DataTable Component Demo</h2>
    <p class="demo-description">
      Showcasing Bloomberg-style data density with Apple-level polish
    </p>
  </div>

  <div class="demo-controls">
    <button class="demo-btn" onclick={simulateLoading}>
      Simulate Loading
    </button>
    <button class="demo-btn" onclick={toggleEmpty}>
      Toggle Empty State
    </button>
    <button class="demo-btn" onclick={() => selectedId = undefined}>
      Clear Selection
    </button>
  </div>

  <div class="demo-section">
    <h3>Standard Table</h3>
    <DataTable
      columns={invoiceColumns}
      data={showEmpty ? [] : invoiceData}
      {loading}
      {selectedId}
      onRowClick={handleRowClick}
      on:sort={handleSort}
      emptyMessage="No invoices found. Create your first invoice to get started."
      maxHeight="500px"
    />
  </div>

  <div class="demo-section">
    <h3>Compact Mode</h3>
    <DataTable
      columns={invoiceColumns}
      data={invoiceData.slice(0, 4)}
      compact={true}
      maxHeight="300px"
    />
  </div>

  <div class="demo-section">
    <h3>Without Border</h3>
    <DataTable
      columns={invoiceColumns}
      data={invoiceData.slice(0, 3)}
      showBorder={false}
      maxHeight="300px"
    />
  </div>

  <div class="demo-section">
    <h3>Custom Format Example</h3>
    <DataTable
      columns={[
        { key: 'invoiceNumber', label: 'Invoice', sortable: true },
        {
          key: 'amount',
          label: 'Amount (Custom)',
          sortable: true,
          align: 'right',
          format: (val) => `BHD ${(val / 1000).toFixed(1)}K`
        },
        { key: 'status', label: 'Status', type: 'status' }
      ]}
      data={invoiceData.slice(0, 5)}
      maxHeight="300px"
    />
  </div>

  <div class="demo-info">
    <h4>Features Demonstrated:</h4>
    <ul>
      <li>Sticky header (scrolls with data)</li>
      <li>Sortable columns (click headers)</li>
      <li>Row selection (click rows)</li>
      <li>Keyboard navigation (arrow keys, Enter, Space)</li>
      <li>Column types: text, number, currency, date, status, actions</li>
      <li>Loading state with skeleton UI</li>
      <li>Empty state message</li>
      <li>Compact mode</li>
      <li>Custom formatters</li>
      <li>Row height: 42px (Bloomberg density)</li>
      <li>Hover: 4% indigo tint (no zebra stripes!)</li>
      <li>Full accessibility (ARIA, keyboard)</li>
      <li>Responsive (horizontal scroll on mobile)</li>
    </ul>
  </div>

  <div class="code-sample">
    <h4>Usage Example:</h4>
    <pre><code>{`<script>
  import DataTable from '$lib/components/ui/DataTable.svelte';

  const columns = [
    {
      key: 'invoiceNumber',
      label: 'Invoice #',
      sortable: true
    },
    {
      key: 'amount',
      label: 'Amount',
      type: 'currency',
      sortable: true,
      align: 'right'
    },
    {
      key: 'status',
      label: 'Status',
      type: 'status'
    }
  ];

  const data = [
    {
      id: 1,
      invoiceNumber: 'INV-001',
      amount: 15750.500,
      status: 'Approved'
    }
  ];

  function handleRowClick(row) {
    console.log('Selected:', row);
  }
</script>

<DataTable
  {columns}
  {data}
  onRowClick={handleRowClick}
  selectedId="1"
  maxHeight="600px"
/>
`}</code></pre>
  </div>
</div>

<style>
  .demo-container {
    padding: var(--page-padding);
    max-width: 1400px;
    margin: 0 auto;
  }

  .demo-header {
    margin-bottom: 32px;
  }

  .demo-header h2 {
    font-size: 28px;
    font-weight: 700;
    color: var(--text-primary);
    margin-bottom: 8px;
  }

  .demo-description {
    font-size: 16px;
    color: var(--text-secondary);
    margin: 0;
  }

  .demo-controls {
    display: flex;
    gap: 12px;
    margin-bottom: 24px;
    flex-wrap: wrap;
  }

  .demo-btn {
    padding: 8px 16px;
    background: var(--brand-indigo);
    color: white;
    border: none;
    border-radius: var(--border-radius-sm);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .demo-btn:hover {
    background: var(--brand-indigo-hover);
    box-shadow: var(--shadow-indigo);
  }

  .demo-btn:active {
    background: var(--brand-indigo-pressed);
  }

  .demo-section {
    margin-bottom: 48px;
  }

  .demo-section h3 {
    font-size: 18px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 16px;
  }

  .demo-info {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius);
    padding: 24px;
    margin-bottom: 32px;
  }

  .demo-info h4 {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 12px;
  }

  .demo-info ul {
    margin: 0;
    padding-left: 20px;
    color: var(--text-secondary);
  }

  .demo-info li {
    margin-bottom: 6px;
    font-size: 14px;
  }

  .code-sample {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius);
    padding: 24px;
  }

  .code-sample h4 {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 12px;
  }

  .code-sample pre {
    margin: 0;
    padding: 16px;
    background: var(--bg-base);
    border-radius: var(--border-radius-sm);
    overflow-x: auto;
  }

  .code-sample code {
    font-family: 'Fira Code', 'Monaco', 'Courier New', monospace;
    font-size: 13px;
    line-height: 1.6;
    color: var(--text-primary);
  }

  /* Action button styles for the demo */
  :global(.action-btn) {
    padding: 4px 12px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--border-radius-sm);
    font-size: 12px;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  :global(.action-btn:hover) {
    background: var(--surface-elevated);
    border-color: var(--brand-indigo);
    color: var(--brand-indigo);
  }
</style>
