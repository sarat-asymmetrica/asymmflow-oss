<script lang="ts">
  /* The summary strip — a compact KPI row + one status-distribution bar,
   * computed over the visible rows. This is the kernel's answer to the
   * card-heavy stat grids the old screens hand-rolled ~10 times: same data,
   * one consistent, denser, more legible treatment. Primitive → owns layout
   * CSS and tone hex-via-var; screens never build this themselves (L1). */
  import type { ComputedSummary } from '../ledger-core'
  import { renderCell } from '../content'

  let { summary }: { summary: ComputedSummary } = $props()
</script>

<div class="k-summary">
  {#if summary.metrics.length}
    <div class="k-summary-metrics">
      {#each summary.metrics as m (m.label)}
        <div class="k-summary-metric">
          <span class="k-summary-label">{m.label}</span>
          <span
            class="k-summary-value"
            class:numeric={m.content === 'money' || m.content === 'quantity' || m.content === 'code'}
            style:color={m.tone ? `var(--k-tone-${m.tone}-fg)` : undefined}
          >
            {renderCell(m.content, m.value, m.currency)}
          </span>
        </div>
      {/each}
    </div>
  {/if}

  {#if summary.distribution && summary.distribution.total > 0}
    {@const dist = summary.distribution}
    <div class="k-summary-dist">
      {#if dist.label}<span class="k-summary-label">{dist.label}</span>{/if}
      <div
        class="k-dist-bar"
        role="img"
        aria-label={dist.segments.map((s) => `${s.key}: ${s.count}`).join(', ')}
      >
        {#each dist.segments as seg (seg.key)}
          <span
            class="k-dist-seg"
            style:width="{seg.pct}%"
            style:background={`var(--k-tone-${seg.tone}-fg)`}
            title="{seg.key}: {seg.count} ({Math.round(seg.pct)}%)"
          ></span>
        {/each}
      </div>
      <div class="k-dist-legend">
        {#each dist.segments as seg (seg.key)}
          <span class="k-dist-key">
            <span class="k-dist-dot" style:background={`var(--k-tone-${seg.tone}-fg)`}></span>
            {seg.key}
            <span class="k-dist-count">{seg.count}</span>
          </span>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .k-summary {
    display: flex;
    flex-wrap: wrap;
    align-items: stretch;
    gap: var(--k-space-lg);
    min-width: 0;
  }
  .k-summary-metrics {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(140px, 100%), 1fr));
    gap: var(--k-space-md);
    flex: 2 1 340px;
    min-width: 0;
  }
  .k-summary-metric {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .k-summary-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .k-summary-value {
    font-size: calc(20px * var(--ui-font-scale));
    font-weight: 700;
    line-height: 1.1;
    overflow-wrap: anywhere;
  }
  .k-summary-value.numeric {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
  }
  .k-summary-dist {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: var(--k-space-sm);
    flex: 1 1 260px;
    min-width: 0;
  }
  .k-dist-bar {
    display: flex;
    width: 100%;
    height: 10px;
    border-radius: var(--border-radius-pill);
    overflow: hidden;
    background: var(--onyx-tint);
  }
  .k-dist-seg {
    height: 100%;
    min-width: 2px;
  }
  .k-dist-legend {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-sm) var(--k-space-md);
    min-width: 0;
  }
  .k-dist-key {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-xs);
    font-size: calc(11px * var(--ui-font-scale));
    color: var(--text-secondary);
    white-space: nowrap;
  }
  .k-dist-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .k-dist-count {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-weight: 600;
    color: var(--text-primary);
  }
</style>
