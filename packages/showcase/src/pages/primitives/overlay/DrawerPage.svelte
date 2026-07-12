<script lang="ts">
  import { Drawer } from '@asymmflow/ui';

  // ── Simulated order data ─────────────────────────────────────────────────
  const orders = [
    {
      id: 'PO-2026-0091',
      vendor: 'Al Jazeera Trading Co.',
      date: '2026-06-08',
      status: 'Pending approval',
      total: 'BHD 14,820.750',
      items: [
        { sku: 'STC-44-MX', description: 'Stainless conduit 44mm × 3m', qty: 240, unit: 'BHD 18.500', line: 'BHD 4,440.000' },
        { sku: 'FIT-ELB-90', description: '90° elbow fitting SS316', qty: 480, unit: 'BHD 7.250', line: 'BHD 3,480.000' },
        { sku: 'GAS-NBR-25', description: 'NBR gasket sheet 25mm', qty: 120, unit: 'BHD 12.000', line: 'BHD 1,440.000' },
        { sku: 'STC-25-MX', description: 'Stainless conduit 25mm × 3m', qty: 300, unit: 'BHD 11.480', line: 'BHD 3,444.000' },
        { sku: 'BOL-M12-SS', description: 'M12 SS316 hex bolt set', qty: 1000, unit: 'BHD 2.016', line: 'BHD 2,016.750' },
      ],
    },
    {
      id: 'PO-2026-0087',
      vendor: 'Gulf Supplies & Equipment',
      date: '2026-06-05',
      status: 'Approved',
      total: 'BHD 7,203.000',
      items: [
        { sku: 'PUMP-CEN-4', description: 'Centrifugal pump 4" SS', qty: 3, unit: 'BHD 1,840.000', line: 'BHD 5,520.000' },
        { sku: 'SEAL-MEC-4', description: 'Mechanical seal 4"', qty: 9, unit: 'BHD 187.000', line: 'BHD 1,683.000' },
      ],
    },
  ];

  let activeOrder = $state<typeof orders[0] | null>(null);
  let drawerOpen = $state(false);
  let sideDemo = $state(false);

  function openOrder(order: typeof orders[0]) {
    activeOrder = order;
    drawerOpen = true;
  }

  // ── Size demo ────────────────────────────────────────────────────────────
  let smOpen = $state(false);
  let lgOpen = $state(false);
  let leftOpen = $state(false);
</script>

<div class="sections">
  <section>
    <h2 class="af-section-title">Drawer</h2>
    <p class="intro">
      A side panel for detail views — the natural home for OpportunityDetail,
      OrderDetail, and other context-rich records. Same a11y / focus-trap / scrim
      contract as Modal. Slides in on the R1 explore curve; exits on R3 stabilize.
    </p>
  </section>

  <!-- ── Realistic order detail ────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">Order detail — realistic demo</h2>
    <p class="intro">
      Click any row to open its order detail drawer. The drawer body uses
      .af-label / .af-numeric field pairs from the design grammar.
    </p>

    <div class="order-table card">
      <div class="order-table__head">
        <span class="af-label">PO number</span>
        <span class="af-label">Vendor</span>
        <span class="af-label">Date</span>
        <span class="af-label">Status</span>
        <span class="af-label" style="text-align: end;">Total</span>
      </div>
      {#each orders as order}
        <button class="order-table__row" onclick={() => openOrder(order)}>
          <span class="af-numeric row-id">{order.id}</span>
          <span class="row-vendor">{order.vendor}</span>
          <span class="af-meta">{order.date}</span>
          <span class="row-status" class:row-status--approved={order.status === 'Approved'}>
            {order.status}
          </span>
          <span class="af-numeric row-total" style="text-align: end;">{order.total}</span>
        </button>
      {/each}
    </div>
  </section>

  <!-- ── The detail drawer ─────────────────────────────────────────────── -->
  <Drawer bind:open={drawerOpen} title={activeOrder?.id ?? ''} size="md">
    {#snippet children()}
      {#if activeOrder}
        <!-- Meta fields -->
        <div class="detail-grid">
          <div class="detail-field">
            <div class="af-label">Vendor</div>
            <div class="detail-value">{activeOrder.vendor}</div>
          </div>
          <div class="detail-field">
            <div class="af-label">Order date</div>
            <div class="detail-value af-numeric">{activeOrder.date}</div>
          </div>
          <div class="detail-field">
            <div class="af-label">Status</div>
            <div class="detail-value">{activeOrder.status}</div>
          </div>
          <div class="detail-field">
            <div class="af-label">Total</div>
            <div class="detail-value af-numeric total-value">{activeOrder.total}</div>
          </div>
        </div>

        <!-- Line items -->
        <div class="detail-section">
          <div class="af-section-title" style="font-size: var(--af-text-md); margin-bottom: var(--af-space-3);">
            Line items
          </div>
          <div class="line-table">
            <div class="line-table__head">
              <span class="af-label">SKU</span>
              <span class="af-label">Description</span>
              <span class="af-label" style="text-align: end;">Qty</span>
              <span class="af-label" style="text-align: end;">Unit</span>
              <span class="af-label" style="text-align: end;">Line total</span>
            </div>
            {#each activeOrder.items as item}
              <div class="line-table__row">
                <span class="af-numeric line-sku">{item.sku}</span>
                <span class="line-desc">{item.description}</span>
                <span class="af-numeric" style="text-align: end;">{item.qty}</span>
                <span class="af-numeric" style="text-align: end;">{item.unit}</span>
                <span class="af-numeric" style="text-align: end;">{item.line}</span>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    {/snippet}

    {#snippet footer()}
      <button class="action-btn action-btn--secondary" onclick={() => (drawerOpen = false)}>
        Close
      </button>
      <button class="action-btn action-btn--primary">
        Approve order
      </button>
    {/snippet}
  </Drawer>

  <!-- ── Size / side variants ──────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">Sizes &amp; sides</h2>
    <div class="row">
      <button class="trigger-btn" onclick={() => (smOpen = true)}>sm — 360px</button>
      <button class="trigger-btn" onclick={() => (lgOpen = true)}>lg — 640px</button>
      <button class="trigger-btn" onclick={() => (leftOpen = true)}>left side</button>
    </div>

    <Drawer bind:open={smOpen} title="Compact panel" size="sm">
      {#snippet children()}
        <p class="body-copy">
          360px — ideal for supplementary context, property panels, or filter
          configurations that don't need the full detail treatment.
        </p>
      {/snippet}
      {#snippet footer()}
        <button class="action-btn action-btn--secondary" onclick={() => (smOpen = false)}>
          Close
        </button>
      {/snippet}
    </Drawer>

    <Drawer bind:open={lgOpen} title="Wide panel" size="lg">
      {#snippet children()}
        <p class="body-copy">
          640px — use when the detail view has a secondary column, a preview pane,
          or a rich form that would feel cramped at 480px.
        </p>
      {/snippet}
      {#snippet footer()}
        <button class="action-btn action-btn--secondary" onclick={() => (lgOpen = false)}>
          Close
        </button>
      {/snippet}
    </Drawer>

    <Drawer bind:open={leftOpen} title="Left-side panel" side="left" size="md">
      {#snippet children()}
        <p class="body-copy">
          Slides in from the left edge. Use when the content logically originates
          from the left navigation (e.g., a settings panel hanging off the sidebar).
        </p>
      {/snippet}
      {#snippet footer()}
        <button class="action-btn action-btn--secondary" onclick={() => (leftOpen = false)}>
          Close
        </button>
      {/snippet}
    </Drawer>
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
    max-width: 64ch;
    margin-top: var(--af-space-2);
    margin-bottom: var(--af-space-4);
  }

  .body-copy {
    font-size: var(--af-text-md);
    line-height: var(--af-leading-base);
    color: var(--af-text-secondary);
  }

  .row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--af-space-3);
  }

  /* ── Order table ────────────────────────────────────────────────────── */
  .card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    overflow: hidden;
  }

  .order-table {
    display: flex;
    flex-direction: column;
    /* Below the table's natural width, scroll within the card instead of
       spilling the page — the honest responsive pattern for dense tables. */
    overflow-x: auto;
  }

  .order-table__head,
  .order-table__row {
    display: grid;
    grid-template-columns: 140px minmax(120px, 1fr) 100px 140px 120px;
    gap: var(--af-space-3);
    padding: var(--af-space-3) var(--af-space-4);
    align-items: center;
    /* Keep the column rhythm intact; the card scrolls X when cramped. */
    min-width: 600px;
  }

  .order-table__head {
    background: var(--af-surface-raised);
    border-bottom: 1px solid var(--af-border);
  }

  .order-table__row {
    border: none;
    background: transparent;
    cursor: pointer;
    text-align: start;
    border-bottom: 1px solid var(--af-border);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    color: var(--af-text);
    transition: background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .order-table__row:last-child { border-bottom: none; }

  .order-table__row:hover {
    background: var(--af-tint);
  }

  .row-id { color: var(--af-accent); font-weight: var(--af-weight-medium); }
  .row-vendor { color: var(--af-text); }
  .row-status {
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    color: var(--af-text-secondary);
  }
  .row-status--approved { color: var(--af-success); }
  .row-total { font-weight: var(--af-weight-medium); }

  /* ── Detail drawer content ───────────────────────────────────────────── */
  .detail-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--af-space-4);
    margin-bottom: var(--af-space-5);
  }

  .detail-field {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
  }

  .detail-value {
    font-size: var(--af-text-md);
    font-weight: var(--af-weight-medium);
    color: var(--af-text);
  }

  .total-value {
    font-size: var(--af-text-xl);
    font-weight: var(--af-weight-bold);
    font-family: var(--af-font-numeric);
  }

  .detail-section {
    padding-top: var(--af-space-4);
    border-top: 1px solid var(--af-border);
  }

  /* ── Line items table ─────────────────────────────────────────────────── */
  .line-table {
    display: flex;
    flex-direction: column;
  }

  .line-table__head,
  .line-table__row {
    display: grid;
    grid-template-columns: 100px 1fr 60px 90px 100px;
    gap: var(--af-space-2);
    padding: var(--af-space-2) 0;
    align-items: baseline;
  }

  .line-table__head {
    border-bottom: 1px solid var(--af-border);
    margin-bottom: var(--af-space-1);
  }

  .line-table__row {
    border-bottom: 1px solid var(--af-border);
    font-size: var(--af-text-sm);
    color: var(--af-text);
  }

  .line-table__row:last-child { border-bottom: none; }

  .line-sku { color: var(--af-accent); font-weight: var(--af-weight-medium); }
  .line-desc { color: var(--af-text-secondary); }

  /* ── Action buttons ───────────────────────────────────────────────────── */
  .trigger-btn {
    display: inline-flex;
    align-items: center;
    min-height: var(--af-control-height);
    padding: 0 var(--af-space-4);
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    border: 1px solid var(--af-inverse-surface);
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-semibold);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .trigger-btn:hover {
    background: color-mix(in srgb, var(--af-inverse-surface) 88%, transparent);
    box-shadow: var(--af-shadow-lift);
  }

  .trigger-btn:active { transform: scale(0.985); }

  .action-btn {
    display: inline-flex;
    align-items: center;
    min-height: var(--af-control-height);
    padding: 0 var(--af-space-4);
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-semibold);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .action-btn:active { transform: scale(0.985); }

  .action-btn--primary {
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    border: 1px solid var(--af-inverse-surface);
  }

  .action-btn--primary:hover {
    background: color-mix(in srgb, var(--af-inverse-surface) 88%, transparent);
    box-shadow: var(--af-shadow-lift);
  }

  .action-btn--secondary {
    background: var(--af-surface);
    color: var(--af-text);
    border: 1px solid var(--af-border-strong);
  }

  .action-btn--secondary:hover {
    background: var(--af-surface-raised);
    box-shadow: var(--af-shadow-sm);
  }
</style>
