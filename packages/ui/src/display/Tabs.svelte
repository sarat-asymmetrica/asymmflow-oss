<script lang="ts">
  import type { Snippet } from 'svelte';

  export interface TabItem {
    id: string;
    label: string;
    disabled?: boolean;
  }

  export interface TabsProps {
    tabs: TabItem[];
    /** Currently active tab id — $bindable */
    active: string;
    /** Renders panel content receiving the active tab id */
    children?: Snippet<[string]>;
    /** Accessible label for the tablist */
    label?: string;
    [key: string]: unknown;
  }

  let {
    tabs,
    active = $bindable(),
    children,
    label = 'Content sections',
    ...restProps
  }: TabsProps = $props();

  // Roving tabindex — only the active tab is in the tab order
  let tabEls: HTMLButtonElement[] = $state([]);

  function select(id: string) {
    active = id;
  }

  function onKeyDown(e: KeyboardEvent, idx: number) {
    const enabledIndices = tabs
      .map((t, i) => (!t.disabled ? i : -1))
      .filter((i) => i !== -1);

    const pos = enabledIndices.indexOf(idx);
    let next = -1;

    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
      e.preventDefault();
      next = enabledIndices[(pos + 1) % enabledIndices.length];
    } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
      e.preventDefault();
      next = enabledIndices[(pos - 1 + enabledIndices.length) % enabledIndices.length];
    } else if (e.key === 'Home') {
      e.preventDefault();
      next = enabledIndices[0];
    } else if (e.key === 'End') {
      e.preventDefault();
      next = enabledIndices[enabledIndices.length - 1];
    }

    if (next !== -1 && tabEls[next]) {
      tabEls[next].focus();
      select(tabs[next].id);
    }
  }
</script>

<div class="af-tabs" {...restProps}>
  <!-- tablist -->
  <div class="af-tabs__list" role="tablist" aria-label={label}>
    {#each tabs as tab, i (tab.id)}
      <button
        bind:this={tabEls[i]}
        class="af-tabs__tab"
        class:af-tabs__tab--active={active === tab.id}
        role="tab"
        id="tab-{tab.id}"
        aria-selected={active === tab.id}
        aria-controls="tabpanel-{tab.id}"
        aria-disabled={tab.disabled || undefined}
        tabindex={active === tab.id ? 0 : -1}
        disabled={tab.disabled}
        onclick={() => select(tab.id)}
        onkeydown={(e) => onKeyDown(e, i)}
      >
        {tab.label}
      </button>
    {/each}
  </div>

  <!-- panel -->
  {#each tabs as tab (tab.id)}
    <div
      id="tabpanel-{tab.id}"
      role="tabpanel"
      aria-labelledby="tab-{tab.id}"
      hidden={active !== tab.id}
      class="af-tabs__panel"
    >
      {#if active === tab.id}
        {@render children?.(tab.id)}
      {/if}
    </div>
  {/each}
</div>

<style>
  .af-tabs {
    display: flex;
    flex-direction: column;
  }

  /* Tablist — underline style */
  .af-tabs__list {
    display: flex;
    align-items: stretch;
    gap: 0;
    border-block-end: 1px solid var(--af-border);
    overflow-x: auto;
    scrollbar-width: none;
  }

  .af-tabs__list::-webkit-scrollbar {
    display: none;
  }

  /* Individual tab */
  .af-tabs__tab {
    position: relative;
    flex-shrink: 0;
    border: none;
    background: transparent;
    padding: var(--af-space-2) var(--af-space-4);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    color: var(--af-text-secondary);
    cursor: pointer;
    /* Min 44px touch target (§2.6) */
    min-height: var(--af-tap-min);
    transition:
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-tabs__tab::after {
    content: '';
    position: absolute;
    inset-inline: 0;
    bottom: -1px; /* sits on the list border */
    height: 2px;
    background: transparent;
    transition: background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
    border-radius: 1px 1px 0 0;
  }

  .af-tabs__tab:hover:not(:disabled) {
    color: var(--af-text);
    background: var(--af-tint);
  }

  .af-tabs__tab--active {
    color: var(--af-text);
    font-weight: var(--af-weight-semibold);
  }

  .af-tabs__tab--active::after {
    background: var(--af-inverse-surface);
  }

  .af-tabs__tab:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  /* Focus visible — overrides global to be inset for tab aesthetics */
  .af-tabs__tab:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: -3px;
    border-radius: var(--af-radius-sm);
  }

  /* Panel */
  .af-tabs__panel {
    padding-block-start: var(--af-space-4);
  }

  /* hidden attr handles display:none natively */
</style>
