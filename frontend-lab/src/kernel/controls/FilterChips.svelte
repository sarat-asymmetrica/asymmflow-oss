<script lang="ts">
  let {
    label,
    options,
    selected = $bindable(''),
  }: {
    label: string
    options: { value: string; label: string }[]
    /** '' = All */
    selected?: string
  } = $props()
</script>

<div class="k-chips" role="group" aria-label={label}>
  <button
    class="k-chip"
    class:active={selected === ''}
    onclick={() => (selected = '')}
  >
    All
  </button>
  {#each options as opt (opt.value)}
    <button
      class="k-chip"
      class:active={selected === opt.value}
      onclick={() => (selected = opt.value)}
      title={opt.label}
    >
      {opt.label}
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
</style>
