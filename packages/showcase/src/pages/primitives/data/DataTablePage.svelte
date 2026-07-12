<script lang="ts">
  /**
   * DataTable showcase page — the pitch artifact.
   *
   * Demonstrates:
   * - Realistic Gulf-trading invoice register (12 rows)
   * - Sortable columns, numeric BHD alignment, status via monochrome weight
   * - Row actions (View / Send) with hover reveal
   * - Selection enabled (checkbox column, tri-state header)
   * - DataToolbar with title + row count + search input + primary action
   * - Loading-state table
   * - Empty-state table
   *
   */

  import { DataTable, DataToolbar, type CellContext } from '@asymmflow/ui';

  // ─── Type ────────────────────────────────────────────────────────────────

  interface Invoice {
    id: string;
    number: string;
    customer: string;
    issued: string;
    due: string;
    amount: number;
    status: 'Paid' | 'Outstanding' | 'Overdue' | 'Draft';
    ref: string;
  }

  // ─── Data — plausible Gulf-trading invoice register ───────────────────────

  const invoices: Invoice[] = [
    { id: '1', number: 'INV-2026-0411', customer: 'Gulf Equipment Trading WLL',      issued: '2026-05-01', due: '2026-05-31', amount: 12_450.000, status: 'Outstanding', ref: 'PO-GE-0092' },
    { id: '2', number: 'INV-2026-0410', customer: 'Al Moayyed Contracting Group',    issued: '2026-04-28', due: '2026-05-28', amount:  4_875.500, status: 'Paid',        ref: 'PO-AM-0341' },
    { id: '3', number: 'INV-2026-0409', customer: 'National Petroleum Co.', issued: '2026-04-25', due: '2026-05-25', amount: 31_200.000, status: 'Paid',       ref: 'PO-NP-1108' },
    { id: '4', number: 'INV-2026-0408', customer: 'National Motor Company BSC',      issued: '2026-04-20', due: '2026-05-20', amount:  2_340.750, status: 'Overdue',     ref: 'PO-NM-0079' },
    { id: '5', number: 'INV-2026-0407', customer: 'Zain Bahrain BSC',                issued: '2026-04-18', due: '2026-05-18', amount:  8_910.000, status: 'Paid',        ref: 'PO-ZB-0553' },
    { id: '6', number: 'INV-2026-0406', customer: 'Gulf Air Group Holding Company',  issued: '2026-04-15', due: '2026-05-15', amount: 19_500.000, status: 'Outstanding', ref: 'PO-GA-0267' },
    { id: '7', number: 'INV-2026-0405', customer: 'Ithmaar Bank BSC',                issued: '2026-04-12', due: '2026-05-12', amount:  5_632.250, status: 'Overdue',     ref: 'PO-IB-0031' },
    { id: '8', number: 'INV-2026-0404', customer: 'Gulf Smelting Co.',         issued: '2026-04-10', due: '2026-05-10', amount: 47_800.000, status: 'Paid',        ref: 'PO-AB-2204' },
    { id: '9', number: 'INV-2026-0403', customer: 'Batelco Group',                   issued: '2026-04-08', due: '2026-05-08', amount:  3_221.000, status: 'Paid',        ref: 'PO-BT-0612' },
    { id:'10', number: 'INV-2026-0402', customer: 'Eskan Bank',                      issued: '2026-04-05', due: '2026-05-05', amount:  6_750.500, status: 'Outstanding', ref: 'PO-ES-0088' },
    { id:'11', number: 'INV-2026-0401', customer: 'Arab Banking Corporation (ABC)',   issued: '2026-04-01', due: '2026-05-01', amount: 22_100.000, status: 'Overdue',     ref: 'PO-AB-0174' },
    { id:'12', number: 'INV-2026-0400', customer: 'Seef Properties WLL',             issued: '2026-03-28', due: '2026-04-27', amount:  9_450.750, status: 'Draft',       ref: 'PO-SP-0033' },
  ];

  // ─── Column definitions ────────────────────────────────────────────────────

  import type { Column } from '@asymmflow/ui';

  function fmtDate(v: unknown): string {
    if (!v) return '—';
    const d = new Date(v as string);
    return d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });
  }

  function fmtBHD(v: unknown): string {
    if (v == null) return '—';
    return `BHD ${(v as number).toLocaleString('en-US', { minimumFractionDigits: 3, maximumFractionDigits: 3 })}`;
  }

  const columns: Column<Invoice>[] = [
    {
      key: 'number',
      header: 'Invoice',
      sortable: true,
      width: '148px',
    },
    {
      key: 'customer',
      header: 'Customer',
      sortable: true,
    },
    {
      key: 'ref',
      header: 'PO Ref',
      width: '112px',
    },
    {
      key: 'issued',
      header: 'Issued',
      sortable: true,
      width: '110px',
      format: fmtDate,
    },
    {
      key: 'due',
      header: 'Due',
      sortable: true,
      width: '110px',
      format: fmtDate,
    },
    {
      key: 'amount',
      header: 'Amount (BHD)',
      numeric: true,
      sortable: true,
      width: '148px',
      format: fmtBHD,
    },
    {
      key: 'status',
      header: 'Status',
      width: '110px',
      // Rendered via the global cell Snippet below (statusCellSnippet)
      // because per-column cell Snippets must live in the template, not script.
    },
  ];

  // ─── Search / filter ───────────────────────────────────────────────────────

  let searchQuery = $state('');
  let selected = $state(new Set<string | number>());

  const filteredData = $derived(
    searchQuery.trim().length < 2
      ? invoices
      : invoices.filter((inv) =>
          [inv.number, inv.customer, inv.ref, inv.status]
            .join(' ')
            .toLowerCase()
            .includes(searchQuery.trim().toLowerCase())
        )
  );

  // ─── Row click ────────────────────────────────────────────────────────────

  let lastClicked = $state<string | null>(null);

  function handleRowClick(row: Invoice) {
    lastClicked = row.number;
  }

  // ─── Demo states ─────────────────────────────────────────────────────────

  let replayKey = $state(0);

  // Outstanding amount sum for the toolbar count
  const outstandingTotal = $derived(
    invoices
      .filter((i) => i.status === 'Outstanding' || i.status === 'Overdue')
      .reduce((s, i) => s + i.amount, 0)
  );
</script>

{#snippet invoiceCellSnippet(ctx: CellContext<Invoice>)}
  {#if ctx.column.key === 'status'}
    {@const s = ctx.value as string}
    <span
      class="inv-status"
      class:inv-status--paid={s === 'Paid'}
      class:inv-status--outstanding={s === 'Outstanding'}
      class:inv-status--overdue={s === 'Overdue'}
      class:inv-status--draft={s === 'Draft'}
    >
      {s}
    </span>
  {:else}
    {ctx.formatted}
  {/if}
{/snippet}

{#snippet rowActionsSnippet(row: Invoice)}
  <button class="act-btn" title="View invoice" aria-label="View {row.number}" onclick={(e) => { e.stopPropagation(); lastClicked = `View: ${row.number}`; }}>
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <path d="M7 3C4.2 3 2 7 2 7s2.2 4 5 4 5-4 5-4-2.2-4-5-4Z" stroke="currentColor" stroke-width="1.2"/>
      <circle cx="7" cy="7" r="1.5" fill="currentColor"/>
    </svg>
    View
  </button>
  {#if row.status === 'Outstanding' || row.status === 'Overdue'}
    <button class="act-btn act-btn--accent" title="Send reminder" aria-label="Send reminder for {row.number}" onclick={(e) => { e.stopPropagation(); lastClicked = `Sent: ${row.number}`; }}>
      <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
        <path d="M2 7l9.5-4.5-4.5 9.5-.5-5-4.5-.5Z" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/>
      </svg>
      Send
    </button>
  {/if}
{/snippet}

{#snippet emptyState()}
  <div class="custom-empty">
    <svg width="36" height="36" viewBox="0 0 36 36" fill="none" aria-hidden="true">
      <circle cx="18" cy="18" r="14" stroke="currentColor" stroke-width="1.2" opacity="0.3"/>
      <path d="M12 18h7M16 14l4 4-4 4" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" opacity="0.5"/>
    </svg>
    <p>No invoices match your search.</p>
    <button class="clear-btn" onclick={() => (searchQuery = '')}>Clear search</button>
  </div>
{/snippet}

<div class="sections">

  <!-- ===== SECTION 1: Main invoice register ===== -->
  <section>
    <h2 class="af-section-title">Invoice register</h2>
    <p class="intro">
      The pitch artifact. Realistic Gulf-trading data — sortable columns,
      BHD 3-decimal numerals (tabular, always), monochrome status weight,
      hover-reveal row actions, selection, and a glass toolbar.
      Click a row or sort a column header.
    </p>

    {#key replayKey}
      <!-- Toolbar: glass, flush with table top -->
      <DataToolbar>
        {#snippet left()}
          <div class="toolbar-title-block">
            <span class="af-section-title toolbar-title">Invoices</span>
            <span class="af-meta af-numeric toolbar-count">{filteredData.length} records</span>
          </div>
          <div class="toolbar-outstanding">
            <span class="af-label">Outstanding</span>
            <span class="af-numeric toolbar-outstanding-val">
              BHD {outstandingTotal.toLocaleString('en-US', { minimumFractionDigits: 3, maximumFractionDigits: 3 })}
            </span>
          </div>
        {/snippet}
        {#snippet right()}
          <div class="search-wrap">
            <svg class="search-icon" width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
              <circle cx="6" cy="6" r="4" stroke="currentColor" stroke-width="1.2"/>
              <path d="M9.5 9.5L12 12" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
            </svg>
            <input
              type="search"
              class="search-input af-text-sm"
              placeholder="Search invoices…"
              aria-label="Search invoices"
              bind:value={searchQuery}
            />
          </div>
          <button class="primary-action" onclick={() => (replayKey++)}>
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
              <path d="M7 3v8M3 7h8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
            </svg>
            New Invoice
          </button>
        {/snippet}
      </DataToolbar>

      <!-- Table -->
      <div class="table-with-toolbar">
        <DataTable
          data={filteredData}
          {columns}
          rowId={(r) => r.id}
          bind:selected
          cell={invoiceCellSnippet}
          onRowClick={handleRowClick}
          rowActions={rowActionsSnippet}
          empty={emptyState}
          label="Invoice register"
          rowCount={invoices.length}
          maxHeight="480px"
        />
      </div>
    {/key}

    {#if lastClicked}
      <p class="feedback af-meta">Last action: <span class="af-numeric">{lastClicked}</span></p>
    {/if}

    {#if selected.size > 0}
      <p class="feedback af-meta">
        <span class="af-numeric">{selected.size}</span> row{selected.size > 1 ? 's' : ''} selected.
      </p>
    {/if}
  </section>

  <!-- ===== SECTION 2: Loading state ===== -->
  <section>
    <h2 class="af-section-title">Loading state</h2>
    <p class="intro">
      Skeleton rows shimmer at the correct row height. The header is present and sticky —
      the user sees structure before data arrives.
    </p>
    <DataTable
      data={[]}
      {columns}
      loading={true}
      loadingRows={6}
      label="Loading invoice register"
      maxHeight="320px"
    />
  </section>

  <!-- ===== SECTION 3: Empty state ===== -->
  <section>
    <h2 class="af-section-title">Empty state</h2>
    <p class="intro">
      Calm default when no data is present. The header stays visible so the user
      understands the shape of the table before any records exist.
    </p>
    <DataTable
      data={[]}
      columns={columns.slice(0, 5)}
      loading={false}
      label="Empty invoice register"
      maxHeight="280px"
    />
  </section>

  <!-- ===== SECTION 4: Custom empty Snippet ===== -->
  <section>
    <h2 class="af-section-title">Custom empty Snippet</h2>
    <p class="intro">
      Callers provide their own empty state via the <code>empty</code> Snippet prop.
      Below: search for "xyz" to trigger it in the live table above, or this one is
      always empty to demonstrate the custom message.
    </p>
    <DataTable
      data={[]}
      columns={columns.slice(0, 4)}
      loading={false}
      empty={emptyState}
      label="Custom empty state demo"
      maxHeight="240px"
    />
  </section>

</div>

<style>
  .sections {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-6);
  }

  .intro {
    color: var(--af-text-secondary);
    font-size: var(--af-text-md);
    max-width: 72ch;
    margin-top: var(--af-space-2);
    margin-bottom: var(--af-space-4);
  }

  /* Table sits directly below toolbar — radii handled by components */
  .table-with-toolbar > :global(.af-table-container) {
    border-radius: 0 0 var(--af-radius-md) var(--af-radius-md);
  }

  /* ── Toolbar internals ───────────────────────────────────────────────── */
  .toolbar-title-block {
    display: flex;
    align-items: baseline;
    gap: var(--af-space-3);
  }

  .toolbar-title {
    line-height: 1;
  }

  .toolbar-count {
    color: var(--af-text-muted);
    font-variant-numeric: tabular-nums lining-nums;
  }

  .toolbar-outstanding {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding-left: var(--af-space-4);
    border-left: 1px solid var(--af-border);
    margin-left: var(--af-space-1);
  }

  .toolbar-outstanding-val {
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-md);
    font-weight: var(--af-weight-semibold);
    font-variant-numeric: tabular-nums lining-nums;
    color: var(--af-text);
  }

  /* Search */
  .search-wrap {
    position: relative;
    display: flex;
    align-items: center;
  }

  .search-icon {
    position: absolute;
    left: var(--af-space-2);
    color: var(--af-text-muted);
    pointer-events: none;
  }

  .search-input {
    padding: 0 var(--af-space-3) 0 calc(var(--af-space-2) + 14px + var(--af-space-1));
    height: var(--af-control-height);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    background: var(--af-surface);
    color: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    width: 200px;
    transition:
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .search-input::placeholder {
    color: var(--af-text-muted);
  }

  .search-input:focus {
    outline: none;
    border-color: var(--af-accent);
    box-shadow: 0 0 0 3px var(--af-accent-tint);
  }

  /* Primary action button */
  .primary-action {
    display: inline-flex;
    align-items: center;
    gap: var(--af-space-2);
    padding: 0 var(--af-space-4);
    height: var(--af-control-height);
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    border: 1px solid var(--af-inverse-surface);
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-semibold);
    cursor: pointer;
    white-space: nowrap;
    transition:
      opacity var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .primary-action:hover {
    opacity: 0.9;
    box-shadow: var(--af-shadow-lift);
  }

  .primary-action:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  /* ── Row action buttons ──────────────────────────────────────────────── */
  .act-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--af-space-1);
    padding: 0 var(--af-space-2);
    height: 28px;
    border: 1px solid var(--af-border-strong);
    border-radius: var(--af-radius-sm);
    background: var(--af-surface);
    color: var(--af-text-secondary);
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-medium);
    cursor: pointer;
    white-space: nowrap;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .act-btn:hover {
    background: var(--af-surface-raised);
    color: var(--af-text);
  }

  .act-btn--accent {
    border-color: var(--af-accent-tint-strong);
    color: var(--af-accent-pressed);
  }

  .act-btn--accent:hover {
    background: var(--af-accent-tint);
    color: var(--af-accent-pressed);
  }

  .act-btn:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  /* ── Status — monochrome weight (§4c — no colored pills) ────────────── */
  .inv-status {
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    font-family: var(--af-font-body);
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }

  .inv-status--paid {
    color: var(--af-success);
  }

  .inv-status--outstanding {
    color: var(--af-text-secondary);
  }

  .inv-status--overdue {
    color: var(--af-danger);
  }

  .inv-status--draft {
    color: var(--af-text-muted);
  }

  /* ── Feedback ────────────────────────────────────────────────────────── */
  .feedback {
    margin-top: var(--af-space-2);
    padding: var(--af-space-2) var(--af-space-3);
    background: var(--af-surface-raised);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    font-size: var(--af-text-xs);
    color: var(--af-text-muted);
  }

  /* ── Custom empty ────────────────────────────────────────────────────── */
  .custom-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--af-space-3);
    color: var(--af-text-muted);
  }

  .custom-empty p {
    font-size: var(--af-text-sm);
    margin: 0;
  }

  .clear-btn {
    padding: var(--af-space-2) var(--af-space-3);
    border: 1px solid var(--af-border-strong);
    border-radius: var(--af-radius-sm);
    background: var(--af-surface);
    color: var(--af-text-secondary);
    font-size: var(--af-text-sm);
    cursor: pointer;
    font-family: var(--af-font-body);
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .clear-btn:hover {
    background: var(--af-surface-raised);
    color: var(--af-text);
  }

  code {
    font-family: 'Fira Code', 'Cascadia Code', 'Courier New', monospace;
    font-size: 0.88em;
    background: var(--af-surface-sunken);
    padding: 1px 5px;
    border-radius: 3px;
    color: var(--af-accent-pressed);
  }

  @media (prefers-reduced-motion: reduce) {
    .search-input,
    .primary-action,
    .act-btn,
    .clear-btn {
      transition: none;
    }
  }
</style>
