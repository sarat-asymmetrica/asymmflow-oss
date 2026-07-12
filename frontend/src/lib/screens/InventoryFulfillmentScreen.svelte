<script lang="ts">
  /**
   * InventoryFulfillmentScreen - Pending Fulfillment Report
   *
   * Read-only report answering the back-to-back trader's core question:
   * what's sold but not yet delivered/invoiced, and is there stock to cover it.
   * Backed by GetInventoryPendingFulfillmentReport, which previously had no caller.
   */

  import { onMount } from 'svelte';
  import { GetInventoryPendingFulfillmentReport } from '../../../wailsjs/go/main/App';
  import PageLayout from '$lib/components/layout/PageLayout.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import DataTable from '$lib/components/ui/DataTable.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import { toast } from '$lib/stores/toasts';
  import { escapeHtml } from '$lib/utils/escapeHtml';

  // Props
  export let embedded = false;

  // State
  let rows: any[] = [];
  let loading = false;
  let loadError = '';

  function ref(val: string | undefined, fallback = '—'): string {
    return val && val.trim() ? escapeHtml(val) : fallback;
  }

  function qty(val: number): string {
    return (val || 0).toLocaleString('en-US', { maximumFractionDigits: 2 });
  }

  // Order status → colour, same palette family as SerialTraceScreen's local map.
  function statusColor(status: string): string {
    const s = (status || '').toLowerCase();
    if (['delivered', 'invoiced', 'closed', 'complete'].some((k) => s.includes(k))) return 'var(--success,#10b981)';
    if (['pending', 'processing', 'open'].some((k) => s.includes(k))) return 'var(--warning,#d97706)';
    if (['cancelled', 'lost'].some((k) => s.includes(k))) return 'var(--danger,#b91c1c)';
    return 'var(--text-secondary,#64748b)';
  }

  const columns = [
    {
      key: 'order_number',
      label: 'Order #',
      sortable: true,
      width: '140px',
      render: (row: any) =>
        `<span style="font-family:var(--font-mono);font-weight:600;color:var(--accent,#6366f1);">${ref(row.order_number)}</span>`
    },
    {
      key: 'customer_name',
      label: 'Customer',
      sortable: true,
      render: (row: any) => `<span style="font-size:13px;">${ref(row.customer_name)}</span>`
    },
    {
      key: 'product_code',
      label: 'Product',
      sortable: true,
      width: '130px',
      render: (row: any) =>
        `<span style="font-family:var(--font-mono);font-size:12px;">${ref(row.product_code)}</span>`
    },
    {
      key: 'ordered_quantity',
      label: 'Ordered',
      sortable: true,
      align: 'right' as const,
      width: '90px',
      render: (row: any) => `<span style="font-size:12px;">${qty(row.ordered_quantity)}</span>`
    },
    {
      key: 'delivered_quantity',
      label: 'Delivered',
      sortable: true,
      align: 'right' as const,
      width: '90px',
      render: (row: any) => `<span style="font-size:12px;">${qty(row.delivered_quantity)}</span>`
    },
    {
      key: 'pending_quantity',
      label: 'Pending',
      sortable: true,
      align: 'right' as const,
      width: '90px',
      render: (row: any) =>
        `<span style="font-size:12px;font-weight:600;color:${row.pending_quantity > 0 ? 'var(--warning,#d97706)' : 'var(--text-secondary)'};">${qty(row.pending_quantity)}</span>`
    },
    {
      key: 'available_quantity',
      label: 'In Stock',
      sortable: true,
      align: 'right' as const,
      width: '90px',
      render: (row: any) => `<span style="font-size:12px;">${qty(row.available_quantity)}</span>`
    },
    {
      key: 'shortage_quantity',
      label: 'Shortage',
      sortable: true,
      align: 'right' as const,
      width: '90px',
      render: (row: any) =>
        `<span style="font-size:12px;font-weight:600;color:${row.shortage_quantity > 0 ? 'var(--danger,#b91c1c)' : 'var(--text-secondary)'};">${qty(row.shortage_quantity)}</span>`
    },
    {
      key: 'status',
      label: 'Order Status',
      sortable: true,
      width: '120px',
      render: (row: any) => {
        const s = row.status || 'Unknown';
        const c = statusColor(s);
        return `<span style="display:inline-flex;align-items:center;gap:5px;">
          <span style="width:7px;height:7px;border-radius:50%;background:${c};flex-shrink:0;"></span>
          <span style="font-size:12px;font-weight:600;color:${c};">${escapeHtml(s)}</span>
        </span>`;
      }
    }
  ];

  async function load() {
    loading = true;
    loadError = '';
    try {
      const data = await GetInventoryPendingFulfillmentReport(500);
      rows = data || [];
    } catch (err) {
      console.error('Failed to load pending fulfillment report:', err);
      loadError = 'Failed to load the fulfillment report — please try again.';
      toast.danger('Failed to load fulfillment report');
      rows = [];
    } finally {
      loading = false;
    }
  }

  // Deep-link to the order carrying the outstanding line — the only document
  // reference the report row carries (there is no per-line PO on a sales order).
  function handleRowClick(row: any) {
    window.dispatchEvent(new CustomEvent('navigateToScreen', {
      detail: { screen: 'orders', order_id: row.order_id, order_number: row.order_number }
    }));
  }

  onMount(load);
</script>

<PageLayout title="Pending Fulfillment" subtitle="Sold but not yet delivered or invoiced" {embedded}>
  <!-- Note: PageLayout hides its own header (and slot="header-actions") when embedded,
       which is how OperationsHub renders this tab — so the refresh action lives inline
       with the results header below, not in the slot. -->
  {#if loading}
    <div class="spinner-wrap">
      <WabiSpinner />
    </div>
  {:else if loadError}
    <Card>
      <div class="empty-state">
        <p>{loadError}</p>
        <Button variant="secondary" on:click={load}>Retry</Button>
      </div>
    </Card>
  {:else}
    <Card>
      <div class="results-header">
        <span class="results-count">
          {rows.length} outstanding line{rows.length !== 1 ? 's' : ''}
          <span class="results-hint">— click a row to open its order</span>
        </span>
        <Button variant="secondary" size="sm" on:click={load}>Refresh</Button>
      </div>

      <DataTable
        {columns}
        data={rows}
        onRowClick={handleRowClick}
        emptyMessage="Nothing outstanding — everything sold has been delivered and invoiced."
      />
    </Card>
  {/if}
</PageLayout>

<style>
  .spinner-wrap {
    display: flex;
    justify-content: center;
    padding: 60px 0;
  }

  .results-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 14px;
  }

  .results-count {
    font-weight: 600;
    font-size: 14px;
    color: var(--text-primary);
  }

  .results-hint {
    font-size: 12px;
    color: var(--text-muted);
    font-style: italic;
  }

  .empty-state {
    padding: 32px;
    text-align: center;
    color: var(--text-secondary);
    font-size: 14px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
  }
</style>
