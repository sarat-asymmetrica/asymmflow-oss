<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let title: string;
  export let tabs: { id: string; label: string; count?: number }[] = [];
  export let activeTab: string = '';

  const dispatch = createEventDispatcher();

  function handleTabChange(tabId: string) {
    activeTab = tabId;
    dispatch('tabChange', tabId);
  }
</script>

<div class="module-layout">
  <!-- Header -->
  <header class="module-header">
    <div class="header-content">
      <h1 class="module-title">{title}</h1>
      <div class="header-actions">
        <slot name="header-actions" />
      </div>
    </div>
  </header>

  <!-- Tabs -->
  {#if tabs.length > 0}
    <nav class="tabs">
      {#each tabs as tab}
        <button
          class="tab"
          class:active={activeTab === tab.id}
          onclick={() => handleTabChange(tab.id)}
          aria-current={activeTab === tab.id ? 'page' : undefined}
        >
          {tab.label}
          {#if tab.count !== undefined}
            <span class="tab-count">{tab.count}</span>
          {/if}
        </button>
      {/each}
    </nav>
  {/if}

  <!-- Content Area -->
  <main class="module-content">
    <slot />
  </main>
</div>

<style>
  .module-layout {
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: var(--bg-base);
  }

  .module-header {
    padding: var(--page-padding);
    height: var(--header-height);
    display: flex;
    align-items: center;
    border-bottom: 1px solid var(--border);
    background: var(--surface);
    flex-shrink: 0;
  }

  .header-content {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .module-title {
    font-size: var(--page-title-size);
    font-weight: var(--page-title-weight);
    color: var(--text-primary);
    line-height: var(--line-height-tight);
    margin: 0;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .tabs {
    display: flex;
    gap: 8px;
    border-bottom: 1px solid var(--border);
    height: var(--tab-height);
    background: var(--surface);
    padding: 0 var(--page-padding);
    flex-shrink: 0;
  }

  .tab {
    background: none;
    border: none;
    padding: 0 16px;
    font-size: 14px;
    font-weight: 500;
    color: var(--text-secondary);
    cursor: pointer;
    position: relative;
    transition: color var(--transition-fast);
    font-family: var(--font-family);
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .tab:hover {
    color: var(--text-primary);
  }

  .tab.active {
    color: var(--brand-indigo);
  }

  .tab.active::after {
    content: '';
    position: absolute;
    bottom: -1px;
    left: 0;
    right: 0;
    height: 2px;
    background: var(--brand-indigo);
    transition: all var(--transition-fast);
  }

  .tab-count {
    font-size: var(--label-size);
    font-weight: 500;
    background: var(--surface-elevated);
    color: var(--text-muted);
    padding: 2px 6px;
    border-radius: 4px;
    min-width: 20px;
    text-align: center;
  }

  .tab.active .tab-count {
    background: var(--indigo-contrast-surface);
    color: var(--brand-indigo);
  }

  .module-content {
    flex: 1;
    padding: var(--page-padding);
    overflow-y: auto;
    min-height: 0;
  }

  /* Focus styles for accessibility */
  .tab:focus-visible {
    outline: 2px solid var(--brand-indigo);
    outline-offset: -2px;
  }
</style>
