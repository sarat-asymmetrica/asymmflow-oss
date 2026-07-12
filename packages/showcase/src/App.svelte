<script lang="ts">
  import type { Component } from 'svelte';
  import Foundations from './pages/Foundations.svelte';
  import Motion from './pages/Motion.svelte';
  import MotionEngine from './pages/MotionEngine.svelte';
  import { formPages } from './pages/primitives/form/registry';
  import { displayPages } from './pages/primitives/display/registry';
  import { dataPages } from './pages/primitives/data/registry';
  import { overlayPages } from './pages/primitives/overlay/registry';
  import { patternPages } from './pages/patterns/registry';
  import { scenePages } from './pages/scenes/registry';
  import { timeseriesChartPages } from './pages/charts/timeseries/registry';
  import { categoricalChartPages } from './pages/charts/categorical/registry';
  import { erpChartPages } from './pages/charts/erp/registry';

  type Page = {
    id: string;
    title: string;
    group: 'Foundations' | 'Primitives' | 'Charts' | 'Patterns' | 'Scenes';
    component: Component;
  };

  // Each kit registers its pages in its own registry file; new waves spread here.
  const pages: Page[] = [
    { id: 'foundations', title: 'Tokens', group: 'Foundations', component: Foundations },
    { id: 'motion', title: 'Motion', group: 'Foundations', component: Motion },
    { id: 'motion-engine', title: 'Motion Engine', group: 'Foundations', component: MotionEngine },
    ...formPages,
    ...displayPages,
    ...dataPages,
    ...overlayPages,
    ...timeseriesChartPages,
    ...categoricalChartPages,
    ...erpChartPages,
    ...patternPages,
    ...scenePages,
  ];

  const groups = ['Foundations', 'Primitives', 'Charts', 'Patterns', 'Scenes'] as const;

  let activeId = $state('foundations');
  let density = $state<'comfortable' | 'compact'>('comfortable');
  let textScale = $state(1);

  const active = $derived(pages.find((p) => p.id === activeId) ?? pages[0]);

  $effect(() => {
    if (density === 'compact') {
      document.documentElement.dataset.afDensity = 'compact';
    } else {
      delete document.documentElement.dataset.afDensity;
    }
  });

  $effect(() => {
    document.documentElement.style.setProperty('--af-scale', String(textScale));
  });
</script>

<a href="#main" class="af-skip-link">Skip to content</a>

<div class="shell">
  <aside class="sidebar">
    <div class="brand">
      <span class="brand-mark" aria-hidden="true"></span>
      <div>
        <div class="brand-name">AsymmFlow</div>
        <div class="af-meta">Design System</div>
      </div>
    </div>

    <nav aria-label="Showcase sections">
      {#each groups as group}
        {@const groupPages = pages.filter((p) => p.group === group)}
        <div class="nav-group">
          <div class="af-label nav-group-label">{group}</div>
          {#if groupPages.length === 0}
            <div class="af-meta nav-empty">coming in a later wave</div>
          {:else}
            {#each groupPages as page}
              <button
                class="nav-item"
                class:active={page.id === activeId}
                onclick={() => (activeId = page.id)}
                aria-current={page.id === activeId ? 'page' : undefined}
              >
                {page.title}
              </button>
            {/each}
          {/if}
        </div>
      {/each}
    </nav>
  </aside>

  <div class="content-column">
    <header class="topbar">
      <h1 class="af-section-title">{active.title}</h1>
      <div class="controls">
        <div class="control" role="group" aria-label="Density">
          <button
            class="seg"
            class:on={density === 'comfortable'}
            onclick={() => (density = 'comfortable')}>Comfortable</button
          >
          <button
            class="seg"
            class:on={density === 'compact'}
            onclick={() => (density = 'compact')}>Compact</button
          >
        </div>
        <label class="control scale-control">
          <span class="af-label">Text</span>
          <input
            type="range"
            min="0.85"
            max="1.3"
            step="0.05"
            bind:value={textScale}
            aria-label="Text scale"
          />
          <span class="af-meta af-numeric">{Math.round(textScale * 100)}%</span>
        </label>
      </div>
    </header>

    <main id="main" class="page">
      <active.component />
    </main>
  </div>
</div>

<style>
  .shell {
    display: grid;
    grid-template-columns: 230px 1fr;
    min-height: 100vh;
  }

  .sidebar {
    border-right: 1px solid var(--af-border);
    background: var(--af-surface);
    padding: var(--af-space-4) var(--af-space-3);
    display: flex;
    flex-direction: column;
    gap: var(--af-space-5);
    position: sticky;
    top: 0;
    height: 100vh;
    overflow-y: auto;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    padding: 0 var(--af-space-2);
  }

  .brand-mark {
    width: 28px;
    height: 28px;
    border-radius: var(--af-radius-sm);
    background:
      conic-gradient(from 210deg, var(--af-accent), var(--af-inverse-surface) 62%, var(--af-accent));
  }

  .brand-name {
    font-family: var(--af-font-display);
    font-weight: var(--af-weight-bold);
    font-size: var(--af-text-lg);
    letter-spacing: var(--af-title-tracking);
  }

  .nav-group {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin-bottom: var(--af-space-4);
  }

  .nav-group-label {
    padding: 0 var(--af-space-2);
    margin-bottom: var(--af-space-2);
  }

  .nav-empty {
    padding: 0 var(--af-space-2);
    font-style: italic;
  }

  .nav-item {
    text-align: left;
    border: none;
    background: none;
    padding: var(--af-space-2) var(--af-space-2);
    border-radius: var(--af-radius-sm);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    color: var(--af-text-secondary);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .nav-item:hover {
    background: var(--af-tint);
    color: var(--af-text);
  }

  .nav-item.active {
    background: var(--af-accent-tint);
    color: var(--af-accent-pressed);
    font-weight: var(--af-weight-semibold);
  }

  .content-column {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .topbar {
    height: var(--af-header-height);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 var(--af-page-padding);
    border-bottom: 1px solid var(--af-border);
    background: var(--af-glass-bg);
    backdrop-filter: var(--af-glass-blur);
    -webkit-backdrop-filter: var(--af-glass-blur);
    position: sticky;
    top: 0;
    z-index: var(--af-z-sticky);
  }

  .controls {
    display: flex;
    align-items: center;
    gap: var(--af-space-4);
  }

  .control {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
  }

  .seg {
    border: 1px solid var(--af-border);
    background: var(--af-surface);
    color: var(--af-text-secondary);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-medium);
    padding: 6px 12px;
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .seg:first-child {
    border-radius: var(--af-radius-pill) 0 0 var(--af-radius-pill);
  }

  .seg:last-child {
    border-radius: 0 var(--af-radius-pill) var(--af-radius-pill) 0;
    margin-left: -1px;
  }

  .seg.on {
    background: var(--af-inverse-surface);
    border-color: var(--af-inverse-surface);
    color: var(--af-text-inverse);
  }

  .scale-control input {
    accent-color: var(--af-accent);
    width: 110px;
  }

  .page {
    padding: var(--af-page-padding);
    max-width: 1080px;
    width: 100%;
  }
</style>
