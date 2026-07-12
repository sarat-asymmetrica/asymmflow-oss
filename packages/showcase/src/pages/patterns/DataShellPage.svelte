<script lang="ts">
  /**
   * DataShellPage — showcase for DataShell.
   *
   * Demonstrates a complete invoice list screen in ~30 lines of consumer code.
   * Displays the consumer code in a <pre> block to make the value proposition clear.
   */

  import { DataShell } from '@asymmflow/patterns';
  import { Button, StatusBadge } from '@asymmflow/ui';
  import type { Column, CellContext } from '@asymmflow/ui';
  import type { StatusKind } from '@asymmflow/ui';

  // ─── Types ──────────────────────────────────────────────────────────────────

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

  // ─── Mock data ───────────────────────────────────────────────────────────────

  const invoices: Invoice[] = [
    { id: '1',  number: 'INV-2026-0411', customer: 'Gulf Equipment Trading WLL',        issued: '2026-05-01', due: '2026-05-31', amount: 12_450.000, status: 'Outstanding', ref: 'PO-GE-0092' },
    { id: '2',  number: 'INV-2026-0410', customer: 'Al Moayyed Contracting Group',      issued: '2026-04-28', due: '2026-05-28', amount:  4_875.500, status: 'Paid',        ref: 'PO-AM-0341' },
    { id: '3',  number: 'INV-2026-0409', customer: 'National Petroleum Co.', issued: '2026-04-25', due: '2026-05-25', amount: 31_200.000, status: 'Paid',        ref: 'PO-NP-1108' },
    { id: '4',  number: 'INV-2026-0408', customer: 'National Motor Company BSC',        issued: '2026-04-20', due: '2026-05-20', amount:  2_340.750, status: 'Overdue',     ref: 'PO-NM-0079' },
    { id: '5',  number: 'INV-2026-0407', customer: 'Zain Bahrain BSC',                  issued: '2026-04-18', due: '2026-05-18', amount:  8_910.000, status: 'Paid',        ref: 'PO-ZB-0553' },
    { id: '6',  number: 'INV-2026-0406', customer: 'Gulf Air Group Holding Company',    issued: '2026-04-15', due: '2026-05-15', amount: 19_500.000, status: 'Outstanding', ref: 'PO-GA-0267' },
    { id: '7',  number: 'INV-2026-0405', customer: 'Ithmaar Bank BSC',                  issued: '2026-04-12', due: '2026-05-12', amount:  5_632.250, status: 'Overdue',     ref: 'PO-IB-0031' },
    { id: '8',  number: 'INV-2026-0404', customer: 'Gulf Smelting Co.',           issued: '2026-04-10', due: '2026-05-10', amount: 47_800.000, status: 'Paid',        ref: 'PO-AB-2204' },
    { id: '9',  number: 'INV-2026-0403', customer: 'Batelco Group',                     issued: '2026-04-08', due: '2026-05-08', amount:  3_221.000, status: 'Paid',        ref: 'PO-BT-0612' },
    { id: '10', number: 'INV-2026-0402', customer: 'Eskan Bank',                        issued: '2026-04-05', due: '2026-05-05', amount:  6_750.500, status: 'Outstanding', ref: 'PO-ES-0088' },
    { id: '11', number: 'INV-2026-0401', customer: 'Arab Banking Corporation (ABC)',    issued: '2026-04-01', due: '2026-05-01', amount: 22_100.000, status: 'Overdue',     ref: 'PO-AB-0174' },
    { id: '12', number: 'INV-2026-0400', customer: 'Seef Properties WLL',               issued: '2026-03-28', due: '2026-04-27', amount:  9_450.750, status: 'Draft',       ref: 'PO-SP-0033' },
  ];

  // ─── Columns ─────────────────────────────────────────────────────────────────

  function fmtDate(v: unknown): string {
    if (!v) return '—';
    return new Date(v as string).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });
  }

  function fmtBHD(v: unknown): string {
    if (v == null) return '—';
    return `BHD ${(v as number).toLocaleString('en-US', { minimumFractionDigits: 3, maximumFractionDigits: 3 })}`;
  }

  const columns: Column<Invoice>[] = [
    { key: 'number',   header: 'Invoice',        sortable: true, width: '148px' },
    { key: 'customer', header: 'Customer',        sortable: true },
    { key: 'ref',      header: 'PO Ref',          width: '112px' },
    { key: 'issued',   header: 'Issued',          sortable: true, width: '110px', format: fmtDate },
    { key: 'due',      header: 'Due',             sortable: true, width: '110px', format: fmtDate },
    { key: 'amount',   header: 'Amount (BHD)',    numeric: true,  sortable: true, width: '148px', format: fmtBHD },
    { key: 'status',   header: 'Status',          width: '120px' },
  ];

  // StatusBadge StatusKind = 'done' | 'pending' | 'attention' | 'failed'
  const statusMap: Record<string, StatusKind> = {
    Paid: 'done',
    Outstanding: 'pending',
    Overdue: 'attention',
    Draft: 'pending',
  };

  let lastClick = $state<string | null>(null);

  // ─── Consumer code (shown in <pre>) ──────────────────────────────────────────

  const consumerCode = `<script lang="ts">
  import { DataShell } from '@asymmflow/patterns';
  import type { Column } from '@asymmflow/ui';

  const columns: Column<Invoice>[] = [
    { key: 'number',   header: 'Invoice',     sortable: true },
    { key: 'customer', header: 'Customer',    sortable: true },
    { key: 'amount',   header: 'Amount (BHD)', numeric: true, sortable: true },
    { key: 'status',   header: 'Status' },
  ];
<\/script>

<DataShell
  title="Invoices"
  data={invoices}
  {columns}
  pageSize={25}
  onRowClick={(row) => navigate(row.id)}
>
  {#snippet actions()}
    <Button>New Invoice</Button>
  {/snippet}
</DataShell>`;
</script>

<div class="sections">

  <!-- ===== SECTION 1: Live DataShell ===== -->
  <section>
    <h2 class="af-section-title">DataShell — invoice register</h2>
    <p class="intro">
      Full list screen: toolbar, search, sort, pagination, and row click —
      wired in a single component. Try searching "Overdue" or sorting by Amount.
    </p>

    <DataShell
      title="Invoices"
      data={invoices}
      {columns}
      searchableKeys={['number', 'customer', 'ref', 'status']}
      pageSize={8}
      onRowClick={(row) => (lastClick = `${row.number} — ${row.customer}`)}
      label="Invoice register"
      tableMaxHeight="400px"
    >
      {#snippet actions()}
        <Button variant="primary">
          {#snippet icon()}
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
              <path d="M7 3v8M3 7h8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
            </svg>
          {/snippet}
          New Invoice
        </Button>
      {/snippet}

      {#snippet cell(ctx)}
        {#if ctx.column.key === 'status'}
          {@const kind = statusMap[ctx.value as string] ?? 'neutral'}
          <StatusBadge status={kind} label={ctx.value as string} />
        {:else}
          {ctx.formatted}
        {/if}
      {/snippet}
    </DataShell>

    {#if lastClick}
      <p class="feedback af-meta">Row clicked: <strong>{lastClick}</strong></p>
    {/if}
  </section>

  <!-- ===== SECTION 2: Consumer code ===== -->
  <section>
    <h2 class="af-section-title">Consumer code — 30 lines vs. ~200 of boilerplate</h2>
    <p class="intro">
      This is all the code you write to get a full list screen with search,
      sort, and pagination. DataShell handles the rest.
    </p>

    <div class="code-block-wrap">
      <div class="code-block-header">
        <span class="af-label">InvoiceScreen.svelte</span>
      </div>
      <pre class="code-block"><code>{consumerCode}</code></pre>
    </div>
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

  /* ── Feedback ────────────────────────────────────────────────────────────── */
  .feedback {
    margin-top: var(--af-space-2);
    padding: var(--af-space-2) var(--af-space-3);
    background: var(--af-surface-raised);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    font-size: var(--af-text-xs);
    color: var(--af-text-muted);
  }

  .feedback strong {
    color: var(--af-text);
    font-weight: var(--af-weight-semibold);
  }

  /* ── Code block ──────────────────────────────────────────────────────────── */
  .code-block-wrap {
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    overflow: hidden;
  }

  .code-block-header {
    padding: var(--af-space-2) var(--af-space-3);
    background: var(--af-surface-raised);
    border-bottom: 1px solid var(--af-border);
  }

  .code-block {
    margin: 0;
    padding: var(--af-space-4);
    overflow-x: auto;
    background: var(--af-surface-sunken);
    font-family: 'Fira Code', 'Cascadia Code', 'Courier New', monospace;
    font-size: var(--af-text-xs);
    line-height: 1.7;
    color: var(--af-text-secondary);
    white-space: pre;
  }

  .code-block code {
    background: none;
    padding: 0;
    border-radius: 0;
    color: inherit;
    font-size: inherit;
  }
</style>
