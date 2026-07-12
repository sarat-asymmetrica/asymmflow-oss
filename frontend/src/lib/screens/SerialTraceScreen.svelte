<script lang="ts">
  /**
   * SerialTraceScreen - Serial Number Lifecycle Traceability
   *
   * Read-only search & view screen. Shows each serial's full lifecycle:
   * PO → GRN → Delivery Note → Invoice → Customer, with warranty dates.
   */

  import { onMount } from 'svelte';
  import { SearchSerials, GetRecentlyDeliveredSerials } from '../../../wailsjs/go/main/App';
  import PageLayout from '$lib/components/layout/PageLayout.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import DataTable from '$lib/components/ui/DataTable.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import { toast } from '$lib/stores/toasts';
  import { escapeHtml } from '$lib/utils/escapeHtml';

  // Props
  export let embedded = false;

  // State
  let searchQuery = '';
  let results: any[] = [];
  let loading = false;
  let hasSearched = false;

  // B10-4: greet with recently-delivered serials instead of a blank page.
  let recentSerials: any[] = [];
  let loadingRecent = false;

  // Serial lifecycle status → colour (local map; the PH statusColor util and its
  // theme tokens are not on this substrate).
  function statusColor(status: string): string {
    const s = (status || '').toLowerCase();
    if (['available', 'delivered', 'received'].some((k) => s.includes(k))) return '#10b981';
    if (['reserved', 'allocated', 'pending'].some((k) => s.includes(k))) return '#d97706';
    if (['shipped', 'dispatched', 'in transit', 'invoiced'].some((k) => s.includes(k))) return '#0e7490';
    if (['returned', 'defective', 'scrapped', 'failed'].some((k) => s.includes(k))) return '#b91c1c';
    return '#64748b';
  }

  function formatDate(val: any): string {
    if (!val) return '—';
    const d = new Date(val);
    if (isNaN(d.getTime())) return '—';
    return d.toLocaleDateString('en-BH', { year: 'numeric', month: 'short', day: '2-digit' });
  }

  function ref(val: string | undefined, fallback = '—'): string {
    return val && val.trim() ? escapeHtml(val) : fallback;
  }

  // Reference columns that deep-link to their document. GRN has no target — the
  // GRN screen is deprecated (see OperationsHub) and there is no surface to send it to.
  function navigateToDoc(kind: 'po' | 'dn' | 'invoice', row: any) {
    const targets: Record<string, Record<string, any>> = {
      po: { screen: 'operations', tab: 'pos', po_id: row.po_id, po_number: row.po_number },
      dn: { screen: 'operations', tab: 'delivery-notes', dn_number: row.dn_number },
      invoice: { screen: 'finance', tab: 'invoices', invoice_id: row.invoice_id, invoice_number: row.invoice_number },
    };
    window.dispatchEvent(new CustomEvent('navigateToScreen', { detail: targets[kind] }));
  }

  // DataTable column definitions
  const columns = [
    {
      key: 'serial_no',
      label: 'Serial #',
      sortable: true,
      width: '160px',
      render: (row: any) =>
        `<span style="font-family:var(--font-mono);font-weight:600;color:var(--accent,#6366f1);">${ref(row.serial_no)}</span>`
    },
    {
      key: 'product_code',
      label: 'Product',
      sortable: true,
      width: '140px',
      render: (row: any) =>
        `<span style="font-family:var(--font-mono);font-size:12px;">${ref(row.product_code)}</span>`
    },
    {
      key: 'status',
      label: 'Status',
      sortable: true,
      width: '110px',
      render: (row: any) => {
        const s = row.status || 'Unknown';
        const c = statusColor(s);
        return `<span style="display:inline-flex;align-items:center;gap:5px;">
          <span style="width:7px;height:7px;border-radius:50%;background:${c};flex-shrink:0;"></span>
          <span style="font-size:12px;font-weight:600;color:${c};">${escapeHtml(s)}</span>
        </span>`;
      }
    },
    {
      key: 'po_number',
      label: 'PO',
      sortable: true,
      width: '130px',
      render: (row: any) =>
        `<span style="font-family:var(--font-mono);font-size:12px;">${ref(row.po_number)}</span>`
    },
    {
      key: 'grn_number',
      label: 'GRN',
      sortable: true,
      width: '130px',
      render: (row: any) =>
        `<span style="font-family:var(--font-mono);font-size:12px;">${ref(row.grn_number)}</span>`
    },
    {
      key: 'dn_number',
      label: 'Delivery Note',
      sortable: true,
      width: '130px',
      render: (row: any) =>
        `<span style="font-family:var(--font-mono);font-size:12px;">${ref(row.dn_number)}</span>`
    },
    {
      key: 'invoice_number',
      label: 'Invoice',
      sortable: true,
      width: '130px',
      render: (row: any) =>
        `<span style="font-family:var(--font-mono);font-size:12px;">${ref(row.invoice_number)}</span>`
    },
    {
      key: 'customer_name',
      label: 'Customer',
      sortable: true,
      render: (row: any) =>
        `<span style="font-size:13px;">${ref(row.customer_name)}</span>`
    },
    {
      key: 'warranty_start_date',
      label: 'Warranty Start',
      sortable: true,
      width: '130px',
      render: (row: any) =>
        `<span style="font-size:12px;color:var(--text-secondary);">${formatDate(row.warranty_start_date)}</span>`
    },
    {
      key: 'warranty_end_date',
      label: 'Warranty End',
      sortable: true,
      width: '130px',
      render: (row: any) => {
        const d = row.warranty_end_date ? new Date(row.warranty_end_date) : null;
        const expired = d && d < new Date();
        const label = formatDate(row.warranty_end_date);
        const color = expired
          ? 'var(--text-muted,#64748b)'
          : 'var(--success,#10b981)';
        return `<span style="font-size:12px;font-weight:600;color:${color};">${label}</span>`;
      }
    }
  ];

  async function doSearch() {
    const q = searchQuery.trim();
    if (!q) {
      toast.warning('Enter a serial number or keyword to search');
      return;
    }
    loading = true;
    hasSearched = true;
    try {
      // SearchSerials(query: string, limit: number): Promise<SerialNumber[]>
      const data = await SearchSerials(q, 200);
      results = data || [];
      if (results.length === 0) {
        toast.info('No serials matched your search');
      }
    } catch (err) {
      console.error('Serial search failed:', err);
      toast.danger('Search failed — please try again');
      results = [];
    } finally {
      loading = false;
    }
  }

  function handleKeydown(e: CustomEvent<KeyboardEvent>) {
    // Input.svelte dispatches a CustomEvent wrapping the native KeyboardEvent in .detail
    if (e.detail?.key === 'Enter') doSearch();
  }

  // B10-4: default greeting — most recently delivered serials, shown until a
  // search is performed. Keeps the read-only search + empty/loading + warranty
  // coloring keep-list behaviors intact (same columns, same cell renderer).
  async function loadRecentlyDelivered() {
    loadingRecent = true;
    try {
      const data = await GetRecentlyDeliveredSerials(25);
      recentSerials = data || [];
    } catch (err) {
      console.error('Failed to load recently delivered serials:', err);
      recentSerials = [];
    } finally {
      loadingRecent = false;
    }
  }

  onMount(() => {
    loadRecentlyDelivered();
  });
</script>

<!-- Shared cell renderer for both the search-results table and the
     "Recently delivered" default table below (read-only search + warranty
     coloring keep-list behaviors). Declared OUTSIDE <PageLayout> — a
     {#snippet} declared as a direct child of a component is treated by
     Svelte as an implicit named-slot prop for that component, which would
     otherwise fail typecheck since PageLayout declares no `serialCell` prop. -->
{#snippet serialCell({ column, row }: { column: any; row: any })}
  {#if column.key === 'po_number' && row.po_number?.trim()}
    <button type="button" class="ref-link" onclick={() => navigateToDoc('po', row)}>
      {row.po_number}
    </button>
  {:else if column.key === 'dn_number' && row.dn_number?.trim()}
    <button type="button" class="ref-link" onclick={() => navigateToDoc('dn', row)}>
      {row.dn_number}
    </button>
  {:else if column.key === 'invoice_number' && row.invoice_number?.trim()}
    <button type="button" class="ref-link" onclick={() => navigateToDoc('invoice', row)}>
      {row.invoice_number}
    </button>
  {:else if column.render}
    {@html column.render(row)}
  {/if}
{/snippet}

<PageLayout title="Serial Traceability" subtitle="Search serial numbers and trace their full lifecycle" {embedded}>
  <!-- Search bar -->
  <div class="search-bar">
    <div class="search-input-wrap">
      <Input
        placeholder="Search by serial number, product code, GRN, PO, invoice or customer…"
        bind:value={searchQuery}
        on:keydown={handleKeydown}
      />
    </div>
    <Button variant="primary" on:click={doSearch} disabled={loading}>
      {loading ? 'Searching…' : 'Search'}
    </Button>
  </div>

  <!-- Results -->
  {#if loading}
    <div class="spinner-wrap">
      <WabiSpinner />
    </div>
  {:else if hasSearched}
    <Card>
      <div class="results-header">
        <span class="results-count">
          {results.length} serial{results.length !== 1 ? 's' : ''} found
        </span>
        <span class="results-hint">Lifecycle: PO → GRN → Delivery Note → Invoice → Customer</span>
      </div>

      {#if results.length > 0}
        <DataTable
          {columns}
          data={results}
          emptyMessage="No serials found"
        >
          {#snippet cell({ column, row })}{@render serialCell({ column, row })}{/snippet}
        </DataTable>
      {:else}
        <div class="empty-state">
          <p>No matching serials — try a different search term.</p>
        </div>
      {/if}
    </Card>
  {:else}
    <Card>
      <div class="empty-state empty-state--initial">
        <div class="empty-icon">⬡</div>
        <p class="empty-title">Serial Lifecycle Traceability</p>
        <p class="empty-sub">
          Enter a serial number, product code, GRN reference, or customer name above to trace
          the full chain — from goods receipt through dispatch to the end customer.
          Warranty status is computed in real time.
        </p>
      </div>
    </Card>

    <!-- B10-4: greet with recently-delivered serials instead of a blank page -->
    <Card>
      <div class="results-header">
        <span class="results-count">Recently delivered</span>
        <span class="results-hint">Lifecycle: PO → GRN → Delivery Note → Invoice → Customer</span>
      </div>

      {#if loadingRecent}
        <div class="spinner-wrap">
          <WabiSpinner />
        </div>
      {:else if recentSerials.length > 0}
        <DataTable
          {columns}
          data={recentSerials}
          emptyMessage="No recently delivered serials"
        >
          {#snippet cell({ column, row })}{@render serialCell({ column, row })}{/snippet}
        </DataTable>
      {:else}
        <div class="empty-state">
          <p>No deliveries recorded yet.</p>
        </div>
      {/if}
    </Card>
  {/if}
</PageLayout>

<style>
  .search-bar {
    display: flex;
    align-items: flex-end;
    gap: 12px;
    margin-bottom: 20px;
  }

  .search-input-wrap {
    flex: 1;
  }

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
  }

  .ref-link {
    font-family: var(--font-mono);
    font-size: 12px;
    background: none;
    border: none;
    padding: 0;
    color: var(--accent, #6366f1);
    cursor: pointer;
    text-decoration: underline;
    text-decoration-color: transparent;
    transition: text-decoration-color var(--transition-fast, 0.15s);
  }

  .ref-link:hover,
  .ref-link:focus-visible {
    text-decoration-color: currentColor;
  }

  .ref-link:focus-visible {
    outline: 2px solid var(--brand-indigo, var(--accent, #6366f1));
    outline-offset: 2px;
  }

  .empty-state--initial {
    padding: 60px 32px;
  }

  .empty-icon {
    font-size: 36px;
    margin-bottom: 16px;
    opacity: 0.3;
  }

  .empty-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 8px;
  }

  .empty-sub {
    max-width: 540px;
    margin: 0 auto;
    line-height: 1.6;
    color: var(--text-muted);
  }
</style>
