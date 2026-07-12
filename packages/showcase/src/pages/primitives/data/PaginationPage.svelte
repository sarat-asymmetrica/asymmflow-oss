<script lang="ts">
  /**
   * Pagination showcase page.
   *
   * Demonstrates:
   * - Pagination composed with DataTable (paginated invoice slice)
   * - Standalone pagination in multiple configurations
   * - Edge cases: first page, last page, single page, many pages
   */

  import { DataTable, Pagination, type Column } from '@asymmflow/ui';

  // ─── Synthetic dataset — 47 transactions ─────────────────────────────────

  interface Transaction {
    id: string;
    ref: string;
    party: string;
    date: string;
    debit: number | null;
    credit: number | null;
  }

  const PARTIES = [
    'Gulf Equipment Trading WLL',
    'Al Moayyed Contracting Group',
    'National Petroleum Co.',
    'National Motor Company BSC',
    'Zain Bahrain BSC',
    'Gulf Air Group Holding',
    'Ithmaar Bank BSC',
    'Gulf Smelting Co.',
    'Batelco Group',
    'Eskan Bank',
  ];

  function fmtBHD(v: unknown): string {
    if (v == null) return '—';
    return (v as number).toLocaleString('en-US', { minimumFractionDigits: 3, maximumFractionDigits: 3 });
  }

  function fmtDate(v: unknown): string {
    if (!v) return '—';
    return new Date(v as string).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });
  }

  const allTransactions: Transaction[] = Array.from({ length: 47 }, (_, i) => {
    const isDebit = i % 3 !== 2;
    const amount = Math.round((1200 + i * 137.5 + (i % 5) * 890) * 1000) / 1000;
    const d = new Date(2026, 4, 1);
    d.setDate(d.getDate() - i);
    return {
      id: String(i + 1),
      ref: `TXN-2026-${String(i + 1).padStart(4, '0')}`,
      party: PARTIES[i % PARTIES.length],
      date: d.toISOString().slice(0, 10),
      debit: isDebit ? amount : null,
      credit: !isDebit ? amount : null,
    };
  });

  const txColumns: Column<Transaction>[] = [
    { key: 'ref',    header: 'Reference', width: '148px' },
    { key: 'party',  header: 'Party' },
    { key: 'date',   header: 'Date', width: '120px', format: fmtDate },
    { key: 'debit',  header: 'Debit (BHD)',  numeric: true, width: '140px', format: fmtBHD },
    { key: 'credit', header: 'Credit (BHD)', numeric: true, width: '140px', format: fmtBHD },
  ];

  // ─── Pagination state ─────────────────────────────────────────────────────

  let page = $state(1);
  const pageSize = 10;

  const pageSlice = $derived(
    allTransactions.slice((page - 1) * pageSize, page * pageSize)
  );

  // ─── Standalone demo states ───────────────────────────────────────────────

  let demoPage1 = $state(1);   // few pages
  let demoPage2 = $state(3);   // mid window
  let demoPage3 = $state(1);   // single page
</script>

<div class="sections">

  <!-- ===== SECTION 1: Table + Pagination composition ===== -->
  <section>
    <h2 class="af-section-title">Table + Pagination</h2>
    <p class="intro">
      The canonical composition: DataTable renders a page slice; Pagination
      controls the current page below. 47 total transactions, 10 per page.
      The "x–y of z" meta uses tabular numerals (§4a).
    </p>

    <div class="table-pagination-block">
      <DataTable
        data={pageSlice}
        columns={txColumns}
        label="Transaction ledger"
        rowCount={allTransactions.length}
        maxHeight="none"
      />
      <Pagination
        bind:page
        {pageSize}
        total={allTransactions.length}
      />
    </div>
  </section>

  <!-- ===== SECTION 2: Standalone configurations ===== -->
  <section>
    <h2 class="af-section-title">Standalone configurations</h2>
    <p class="intro">
      Pagination renders independently of the table — wire it to any data source.
      The numeric window contracts gracefully at the edges.
    </p>

    <div class="configs">
      <div class="config-card card">
        <span class="af-label config-label">5 pages · on page {demoPage1}</span>
        <Pagination bind:page={demoPage1} pageSize={10} total={50} />
      </div>

      <div class="config-card card">
        <span class="af-label config-label">12 pages · on page {demoPage2}</span>
        <Pagination bind:page={demoPage2} pageSize={10} total={120} window={5} />
      </div>

      <div class="config-card card">
        <span class="af-label config-label">1 page · all disabled</span>
        <Pagination bind:page={demoPage3} pageSize={20} total={14} />
      </div>

      <div class="config-card card">
        <span class="af-label config-label">0 results</span>
        <Pagination page={1} pageSize={20} total={0} />
      </div>
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

  /* Table + Pagination block: table provides the container, pagination docks below */
  .table-pagination-block {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    overflow: hidden;
  }

  /* Override table container border — the outer block provides it */
  .table-pagination-block > :global(.af-table-container) {
    border: none;
    border-radius: 0;
  }

  /* ── Standalone configs ─────────────────────────────────────────────── */
  .configs {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-3);
  }

  .card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    overflow: hidden;
  }

  .config-card {
    display: flex;
    flex-direction: column;
  }

  .config-label {
    padding: var(--af-space-2) var(--af-space-3);
    border-bottom: 1px solid var(--af-border);
    color: var(--af-text-muted);
  }

  /* Pagination inside config card: remove its top border (card provides it) */
  .config-card > :global(.af-pagination) {
    border-top: none;
  }
</style>
