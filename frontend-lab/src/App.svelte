<script lang="ts">
  import Showcase from './screens/Showcase.svelte'
  import DocumentLedger from '$kernel/archetypes/DocumentLedger.svelte'
  import { invoicesDescriptor } from './screens/invoices.descriptor'

  const views = ['Invoices', 'Showcase'] as const
  type View = (typeof views)[number]
  let view = $state<View>('Invoices')
</script>

<div class="lab-shell">
  <nav class="lab-nav">
    <span class="lab-brand">Kernel Lab</span>
    {#each views as v (v)}
      <button class="lab-tab" class:active={view === v} onclick={() => (view = v)}>{v}</button>
    {/each}
  </nav>
  <main class="lab-main">
    {#if view === 'Invoices'}
      <DocumentLedger descriptor={invoicesDescriptor} />
    {:else}
      <Showcase />
    {/if}
  </main>
</div>

<style>
  /* Lab shell chrome — dev harness territory, not a product screen. */
  .lab-shell {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }
  .lab-nav {
    display: flex;
    align-items: center;
    gap: var(--k-space-sm);
    padding: 8px var(--page-padding);
    border-bottom: var(--border-width) solid var(--border);
    background: var(--surface);
    flex-shrink: 0;
  }
  .lab-brand {
    font-family: var(--font-display);
    font-weight: 700;
    font-size: calc(13px * var(--ui-font-scale));
    margin-right: var(--k-space-md);
  }
  .lab-tab {
    font: inherit;
    font-size: calc(13px * var(--ui-font-scale));
    padding: 4px 12px;
    border: none;
    border-radius: var(--border-radius-sm);
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
  }
  .lab-tab.active {
    background: var(--onyx-tint);
    color: var(--text-primary);
    font-weight: 600;
  }
  .lab-main {
    flex: 1;
    min-height: 0;
    min-width: 0;
  }
</style>
