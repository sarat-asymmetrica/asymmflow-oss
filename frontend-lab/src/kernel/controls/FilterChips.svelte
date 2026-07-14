<script lang="ts">
  let {
    label,
    options,
    selected = $bindable(''),
  }: {
    label: string
    options: { value: string; label: string; count?: number }[]
    /** '' = All */
    selected?: string
  } = $props()

  const totalCount = $derived(options.reduce((n, o) => n + (o.count ?? 0), 0))
</script>

<div class="k-chips" role="group" aria-label={label}>
  <button
    class="k-chip"
    class:active={selected === ''}
    onclick={() => (selected = '')}
  >
    All{#if options.some((o) => o.count != null)}<span class="k-chip-count">{totalCount}</span>{/if}
  </button>
  {#each options as opt (opt.value)}
    <button
      class="k-chip"
      class:active={selected === opt.value}
      onclick={() => (selected = opt.value)}
      title={opt.label}
    >
      {opt.label}{#if opt.count != null}<span class="k-chip-count">{opt.count}</span>{/if}
    </button>
  {/each}
</div>

<style>
  .k-chips {
    display: flex;
    align-items: center;
    gap: var(--k-space-xs);
    flex-wrap: wrap;
    min-width: 0;
  }
  .k-chip {
    font-family: var(--font-ui);
    font-size: calc(12px * var(--ui-font-scale));
    font-weight: 500;
    color: var(--text-secondary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-pill);
    padding: 4px 12px;
    cursor: pointer;
    transition: background var(--motion-fast) var(--ease-standard);
    max-width: 240px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-chip:hover {
    background: var(--onyx-tint);
  }
  .k-chip.active {
    background: var(--onyx);
    border-color: var(--onyx);
    color: var(--canvas);
  }
  .k-chip-count {
    display: inline-block;
    margin-left: 6px;
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: 0.9em;
    font-weight: 600;
    opacity: 0.7;
  }
</style>
