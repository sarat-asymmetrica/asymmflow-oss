<script lang="ts">
  /* Two-column balance comparison + variance banner — the kernel's answer to
   * the "statement side vs book side, then a difference line" shape that
   * recurs across month-end reconciliation screens (bank, book-vs-bank,
   * intercompany). Generic over N columns (BookBankRecon uses 2); each
   * column is a title + a list of lines + a bold total. Primitive → owns
   * layout CSS and tone hex-via-var; screens never build this themselves
   * (L1). Hardened: huge/negative values format through formatMoney, long
   * labels truncate with a title tooltip, empty line lists render just the
   * total. */
  import { formatMoney } from '../format'

  const RECONCILED_TOLERANCE = 0.001

  let {
    columns,
    variance,
    currency = 'BHD',
  }: {
    columns: {
      title: string
      lines: { label: string; value: number; note?: string }[]
      total: { label: string; value: number }
    }[]
    variance: { label: string; value: number }
    currency?: string
  } = $props()

  const reconciled = $derived(Math.abs(variance.value) < RECONCILED_TOLERANCE)
</script>

<div class="k-balcmp">
  <div class="k-balcmp-columns">
    {#each columns as col (col.title)}
      <div class="k-balcmp-col">
        <span class="k-balcmp-title">{col.title}</span>
        <div class="k-balcmp-lines">
          {#each col.lines as line, i (i)}
            <div class="k-balcmp-line">
              <span class="k-balcmp-label-wrap">
                <span class="k-balcmp-label" title={line.label}>{line.label}</span>
                {#if line.note}<span class="k-balcmp-note" title={line.note}>{line.note}</span>{/if}
              </span>
              <span class="k-balcmp-value">{formatMoney(line.value, currency)}</span>
            </div>
          {/each}
        </div>
        <div class="k-balcmp-total">
          <span class="k-balcmp-total-label" title={col.total.label}>{col.total.label}</span>
          <span class="k-balcmp-total-value">{formatMoney(col.total.value, currency)}</span>
        </div>
      </div>
    {/each}
  </div>

  <div
    class="k-balcmp-variance"
    style:background={`var(--k-tone-${reconciled ? 'success' : 'danger'}-bg)`}
    style:color={`var(--k-tone-${reconciled ? 'success' : 'danger'}-fg)`}
  >
    {#if reconciled}
      <span class="k-balcmp-variance-label">Reconciled</span>
    {:else}
      <span class="k-balcmp-variance-label" title={variance.label}>{variance.label}</span>
      <span class="k-balcmp-variance-value">{formatMoney(variance.value, currency)}</span>
    {/if}
  </div>
</div>

<style>
  .k-balcmp {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    min-width: 0;
  }
  .k-balcmp-columns {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-md);
    min-width: 0;
  }
  .k-balcmp-col {
    flex: 1 1 260px;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-sm);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius);
    padding: var(--k-space-md);
  }
  .k-balcmp-title {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-balcmp-lines {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-xs);
    min-width: 0;
  }
  .k-balcmp-line {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-balcmp-label-wrap {
    display: flex;
    flex-direction: column;
    min-width: 0;
    flex: 1 1 auto;
  }
  .k-balcmp-label {
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }
  .k-balcmp-note {
    font-size: calc(11px * var(--ui-font-scale));
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }
  .k-balcmp-value {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-primary);
    flex-shrink: 0;
    white-space: nowrap;
  }
  .k-balcmp-total {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--k-space-sm);
    min-width: 0;
    padding-top: var(--k-space-sm);
    border-top: var(--border-width) solid var(--border);
  }
  .k-balcmp-total-label {
    font-size: calc(13px * var(--ui-font-scale));
    font-weight: 700;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }
  .k-balcmp-total-value {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(14px * var(--ui-font-scale));
    font-weight: 700;
    flex-shrink: 0;
    white-space: nowrap;
  }
  .k-balcmp-variance {
    display: flex;
    align-items: baseline;
    justify-content: center;
    gap: var(--k-space-sm);
    min-width: 0;
    padding: var(--k-space-sm) var(--k-space-md);
    border-radius: var(--border-radius);
    text-align: center;
  }
  .k-balcmp-variance-label {
    font-size: calc(13px * var(--ui-font-scale));
    font-weight: 700;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }
  .k-balcmp-variance-value {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(14px * var(--ui-font-scale));
    font-weight: 700;
    flex-shrink: 0;
    white-space: nowrap;
  }
</style>
