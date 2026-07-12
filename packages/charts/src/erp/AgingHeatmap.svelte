<script lang="ts">
  /**
   * AgingHeatmap — receivables aging matrix, the CFO's morning read.
   *
   * Real HTML table (not SVG): ERP users want to read / tab / copy numbers.
   * Cell intensity = value / global max, mapped to alpha 8–55%.
   * Severity hue per bucket: 0 → success, 1–2 → warning, 3+ → danger.
   * Zero-value cells: plain surface, value rendered as '—'.
   * TOTAL row + TOTAL column computed. Rows/cells keyboard-accessible.
   * Entrance: rows fade-rise staggered.
   *
   * Constitution: all --af-* tokens, color-mix() for alpha ramps, no raw hex.
   */

  interface Row {
    label: string;
    values: number[];
  }

  interface Props {
    rows: Row[];
    buckets?: string[];
    height?: number;
    valueFormat?: (n: number) => string;
    title: string;
    description?: string;
    onCellClick?: (row: string, bucketIndex: number) => void;
  }

  import { formatCompact } from '../format.js';

  let {
    rows,
    buckets = ['Current', '1–30', '31–60', '61–90', '90+'],
    height,
    valueFormat,
    title,
    description,
    onCellClick,
  }: Props = $props();

  const fmt = $derived(valueFormat ?? formatCompact);

  // ─── Totals ────────────────────────────────────────────────────────────────

  const colTotals = $derived(
    buckets.map((_, ci) => rows.reduce((s, r) => s + (r.values[ci] ?? 0), 0)),
  );

  const rowTotals = $derived(
    rows.map((r) => r.values.reduce((s, v) => s + v, 0)),
  );

  const grandTotal = $derived(rowTotals.reduce((s, v) => s + v, 0));

  // ─── Global max for honest intensity ─────────────────────────────────────

  const globalMax = $derived(
    Math.max(1, ...rows.flatMap((r) => r.values.filter((v) => v > 0))),
  );

  // ─── Color token per bucket index ─────────────────────────────────────────
  // 0 → success, 1–2 → warning, 3+ → danger

  function bucketColorToken(ci: number): string {
    if (ci === 0) return '--af-success';
    if (ci <= 2) return '--af-warning';
    return '--af-danger';
  }

  // ─── Cell background via color-mix ────────────────────────────────────────
  // intensity maps 0.0 → 8%, 1.0 → 55%

  function cellBg(value: number, ci: number): string {
    if (value <= 0) return '';
    const intensity = Math.min(1, value / globalMax);
    const pct = Math.round(8 + intensity * 47);
    const token = bucketColorToken(ci);
    return `background: color-mix(in srgb, var(${token}) ${pct}%, transparent)`;
  }

  // ─── Entrance ─────────────────────────────────────────────────────────────

  let entered = $state(false);

  $effect(() => {
    const id = setTimeout(() => { entered = true; }, 30);
    return () => clearTimeout(id);
  });
</script>

<div
  class="af-aging-heatmap"
  role="region"
  aria-label={description ? `${title}. ${description}` : title}
>
  <table class="heatmap-table" aria-label={title}>
    <thead>
      <tr>
        <th class="col-label af-label" scope="col">Account</th>
        {#each buckets as bucket, ci}
          <th
            class="col-bucket af-label"
            class:col-bucket--current={ci === 0}
            class:col-bucket--warning={ci >= 1 && ci <= 2}
            class:col-bucket--danger={ci >= 3}
            scope="col"
          >
            {bucket}
          </th>
        {/each}
        <th class="col-total af-label" scope="col">Total</th>
      </tr>
    </thead>

    <tbody>
      {#if rows.length === 0}
        <tr>
          <td class="empty-label" colspan={buckets.length + 2}>No data</td>
        </tr>
      {:else}
      {#each rows as row, ri}
        <tr
          class="heatmap-row"
          class:heatmap-row--entered={entered}
          style="--row-index: {Math.min(ri, 12)}"
        >
          <th class="row-label" scope="row">{row.label}</th>

          {#each buckets as _bucket, ci}
            {@const value = row.values[ci] ?? 0}
            {@const isEmpty = value <= 0}

            {#if onCellClick}
              <td class="heatmap-cell heatmap-cell--clickable" style={isEmpty ? '' : cellBg(value, ci)}>
                <button
                  class="cell-btn af-numeric"
                  onclick={() => onCellClick(row.label, ci)}
                  aria-label="{row.label}, {buckets[ci]}: {isEmpty ? 'zero' : fmt(value)}"
                >
                  {isEmpty ? '—' : fmt(value)}
                </button>
              </td>
            {:else}
              <td
                class="heatmap-cell af-numeric"
                class:heatmap-cell--empty={isEmpty}
                style={isEmpty ? '' : cellBg(value, ci)}
              >
                {isEmpty ? '—' : fmt(value)}
              </td>
            {/if}
          {/each}

          <td class="heatmap-cell heatmap-cell--row-total af-numeric">
            {rowTotals[ri] > 0 ? fmt(rowTotals[ri]) : '—'}
          </td>
        </tr>
      {/each}
      {/if}
    </tbody>

    <tfoot>
      <tr class="total-row">
        <th class="row-label row-label--total" scope="row">Total</th>
        {#each colTotals as total, ci}
          <td
            class="heatmap-cell heatmap-cell--col-total af-numeric"
            style={total > 0 ? cellBg(total, ci) : ''}
          >
            {total > 0 ? fmt(total) : '—'}
          </td>
        {/each}
        <td class="heatmap-cell heatmap-cell--grand-total af-numeric">
          {grandTotal > 0 ? fmt(grandTotal) : '—'}
        </td>
      </tr>
    </tfoot>
  </table>
</div>

<style>
  /* ── Wrapper ──────────────────────────────────────────────────────────────── */
  .af-aging-heatmap {
    width: 100%;
    overflow-x: auto;
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    background: var(--af-surface);
  }

  /* ── Empty state ──────────────────────────────────────────────────────────── */
  .empty-label {
    padding: var(--af-space-4) var(--af-space-3);
    text-align: center;
    color: var(--af-text-muted);
    font-size: var(--af-text-sm);
    font-family: var(--af-font-body);
  }

  /* ── Table ────────────────────────────────────────────────────────────────── */
  .heatmap-table {
    width: 100%;
    border-collapse: collapse;
    font-family: var(--af-font-body);
  }

  /* ── Header ───────────────────────────────────────────────────────────────── */
  thead th {
    padding: var(--af-space-2) var(--af-space-3);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
    color: var(--af-text-muted);
    background: var(--af-surface-raised);
    border-bottom: 1px solid var(--af-border-strong);
    text-align: right;
    white-space: nowrap;
  }

  .col-label {
    text-align: left;
    min-width: 160px;
  }

  .col-bucket--current { color: var(--af-success); }
  .col-bucket--warning { color: var(--af-warning); }
  .col-bucket--danger  { color: var(--af-danger); }

  /* ── Row labels ───────────────────────────────────────────────────────────── */
  .row-label {
    padding: var(--af-space-2) var(--af-space-3);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    color: var(--af-text);
    text-align: left;
    white-space: nowrap;
    border-bottom: 1px solid var(--af-border);
  }

  .row-label--total {
    font-weight: var(--af-weight-semibold);
    color: var(--af-text);
  }

  /* ── Data cells ───────────────────────────────────────────────────────────── */
  .heatmap-cell {
    padding: var(--af-space-2) var(--af-space-3);
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-sm);
    font-variant-numeric: tabular-nums lining-nums;
    color: var(--af-text);
    text-align: right;
    border-bottom: 1px solid var(--af-border);
    transition: box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .heatmap-cell--empty {
    color: var(--af-text-muted);
  }

  /* Row hover wash */
  .heatmap-row:hover .heatmap-cell,
  .heatmap-row:hover .row-label {
    background-color: var(--af-tint);
  }

  /* Cell hover inset ring */
  .heatmap-cell:hover {
    box-shadow: inset 0 0 0 1px var(--af-border-strong);
    position: relative;
    z-index: 1;
  }

  /* ── Clickable cells ──────────────────────────────────────────────────────── */
  .heatmap-cell--clickable {
    padding: 0;
  }

  .cell-btn {
    width: 100%;
    height: 100%;
    min-height: 44px;
    padding: var(--af-space-2) var(--af-space-3);
    background: transparent;
    border: none;
    cursor: pointer;
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-sm);
    font-variant-numeric: tabular-nums lining-nums;
    color: var(--af-text);
    text-align: right;
    display: block;
    width: 100%;
    transition: background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .cell-btn:hover {
    background: var(--af-tint-medium);
  }

  .cell-btn:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: -2px;
    border-radius: 2px;
  }

  /* ── Total row ────────────────────────────────────────────────────────────── */
  .total-row .heatmap-cell,
  .total-row .row-label {
    border-top: 2px solid var(--af-border-strong);
    border-bottom: none;
    font-weight: var(--af-weight-semibold);
  }

  .heatmap-cell--row-total,
  .heatmap-cell--col-total,
  .heatmap-cell--grand-total {
    font-weight: var(--af-weight-semibold);
    border-inline-start: 1px solid var(--af-border-strong);
  }

  /* ── Entrance: rows fade-rise staggered ─────────────────────────────────── */
  @media (prefers-reduced-motion: no-preference) {
    .heatmap-row {
      opacity: 0;
      transform: translateY(6px);
    }

    .heatmap-row--entered {
      animation: af-row-rise var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
      animation-delay: calc(var(--row-index, 0) * var(--af-motion-stagger));
    }

    @keyframes af-row-rise {
      from {
        opacity: 0;
        transform: translateY(6px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .heatmap-row {
      opacity: 1;
      transform: none;
    }
  }
</style>
