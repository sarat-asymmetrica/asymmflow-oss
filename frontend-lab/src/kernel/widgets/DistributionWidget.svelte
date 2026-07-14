<script lang="ts">
  /* Distribution — share-of-whole as a stacked bar (horizontal, the
   * LedgerSummary distribution look) or a row of vertical bars (AR-aging
   * buckets and similar ranged breakdowns). Segments carry their own Tone
   * (not a categorical series slot) because a distribution is usually a
   * status/threshold mix, not free categories — that's DonutWidget's job. */
  import type { Navigate, WidgetSegment } from '../hub'

  let {
    segments,
    orientation = 'horizontal',
    navigate,
  }: {
    segments: WidgetSegment[]
    orientation?: 'horizontal' | 'vertical'
    navigate?: Navigate
  } = $props()

  const total = $derived(segments.reduce((s, x) => s + x.value, 0))
  const max = $derived(segments.reduce((m, x) => Math.max(m, x.value), 0))
  const pctOf = (seg: WidgetSegment) =>
    seg.pct ?? (total > 0 ? (seg.value / total) * 100 : 0)

  function click(seg: WidgetSegment) {
    if (seg.nav) navigate?.(seg.nav)
  }
</script>

{#if orientation === 'vertical'}
  <div class="k-dist-v">
    {#each segments as seg (seg.key)}
      <svelte:element
        this={seg.nav ? 'button' : 'div'}
        role={seg.nav ? 'button' : undefined}
        tabindex={seg.nav ? 0 : undefined}
        class="k-dist-v-col"
        class:clickable={!!seg.nav}
        onclick={seg.nav ? () => click(seg) : undefined}>
        <span class="k-dist-v-val">{seg.value.toLocaleString('en-US')}</span>
        <span
          class="k-dist-v-bar"
          style:height="{max > 0 ? (seg.value / max) * 100 : 0}%"
          style:background="var(--k-tone-{seg.tone}-fg)"
        ></span>
        <span class="k-dist-v-label">{seg.label}</span>
      </svelte:element>
    {/each}
  </div>
{:else}
  <div class="k-dist-h">
    <div class="k-dist-h-bar" role="img" aria-label={segments.map((s) => `${s.label}: ${s.value}`).join(', ')}>
      {#each segments as seg (seg.key)}
        <svelte:element
          this={seg.nav ? 'button' : 'span'}
          role={seg.nav ? 'button' : undefined}
          tabindex={seg.nav ? 0 : undefined}
          class="k-dist-h-seg"
          class:clickable={!!seg.nav}
          style:width="{pctOf(seg)}%"
          style:background="var(--k-tone-{seg.tone}-fg)"
          title="{seg.label}: {seg.value} ({Math.round(pctOf(seg))}%)"
          onclick={seg.nav ? () => click(seg) : undefined}
        ></svelte:element>
      {/each}
    </div>
    <ul class="k-dist-h-legend">
      {#each segments as seg (seg.key)}
        <li>
          <svelte:element
            this={seg.nav ? 'button' : 'div'}
        role={seg.nav ? 'button' : undefined}
        tabindex={seg.nav ? 0 : undefined}
            class="k-dist-h-key"
            class:clickable={!!seg.nav}
            onclick={seg.nav ? () => click(seg) : undefined}>
            <span class="k-dist-h-dot" style:background="var(--k-tone-{seg.tone}-fg)"></span>
            <span class="k-dist-h-name">{seg.label}</span>
            <span class="k-dist-h-val">{seg.value.toLocaleString('en-US')}</span>
            <span class="k-dist-h-pct">{Math.round(pctOf(seg))}%</span>
          </svelte:element>
        </li>
      {/each}
    </ul>
  </div>
{/if}

<style>
  /* Horizontal (stacked-bar + legend). */
  .k-dist-h {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-dist-h-bar {
    display: flex;
    width: 100%;
    height: 10px;
    border-radius: var(--border-radius-pill);
    overflow: hidden;
    background: var(--onyx-tint);
    gap: 2px;
  }
  .k-dist-h-seg {
    height: 100%;
    min-width: 2px;
    border: none;
    padding: 0;
    margin: 0;
  }
  .k-dist-h-seg.clickable {
    cursor: pointer;
  }
  .k-dist-h-legend {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-sm) var(--k-space-md);
    min-width: 0;
  }
  .k-dist-h-key {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-xs);
    font-size: calc(12px * var(--ui-font-scale));
    color: var(--text-secondary);
    min-width: 0;
    max-width: 100%;
    background: none;
    border: none;
    padding: 0;
    font-family: inherit;
  }
  .k-dist-h-key.clickable {
    cursor: pointer;
  }
  .k-dist-h-key.clickable:hover .k-dist-h-name {
    color: var(--text-primary);
  }
  .k-dist-h-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .k-dist-h-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }
  .k-dist-h-val {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-weight: 600;
    color: var(--text-primary);
    flex-shrink: 0;
  }
  .k-dist-h-pct {
    color: var(--text-muted);
    flex-shrink: 0;
  }

  /* Vertical (bucketed bars — AR aging etc). */
  .k-dist-v {
    display: flex;
    align-items: flex-end;
    gap: var(--k-space-md);
    height: 120px;
    min-width: 0;
  }
  .k-dist-v-col {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: flex-end;
    gap: var(--k-space-xs);
    flex: 1 1 0;
    min-width: 0;
    height: 100%;
    background: none;
    border: none;
    padding: 0;
    font-family: inherit;
  }
  .k-dist-v-col.clickable {
    cursor: pointer;
  }
  .k-dist-v-val {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(12px * var(--ui-font-scale));
    font-weight: 600;
    color: var(--text-primary);
  }
  .k-dist-v-bar {
    width: 100%;
    max-width: 32px;
    min-height: 2px;
    border-radius: 3px 3px 0 0;
  }
  .k-dist-v-label {
    font-size: calc(11px * var(--ui-font-scale));
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 100%;
    min-width: 0;
  }
</style>
