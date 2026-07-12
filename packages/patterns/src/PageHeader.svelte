<script lang="ts">
  /**
   * PageHeader — the top row of every screen.
   *
   * Renders: title (af-page-title) + optional breadcrumb + meta slot + actions Snippet.
   * Constitution §4a: page title uses display face (Space Grotesk).
   * Constitution §4b: spacing from the φ ladder only.
   */

  import type { Snippet } from 'svelte';

  export interface BreadcrumbItem {
    label: string;
    href?: string;
  }

  export interface PageHeaderProps {
    /** Primary page title. */
    title: string;
    /** Optional breadcrumb trail rendered above the title. */
    breadcrumb?: BreadcrumbItem[];
    /** Optional meta Snippet — rendered beside the title (e.g. status badge, record count). */
    meta?: Snippet;
    /** Optional actions Snippet — rendered at the trailing edge (buttons, menus). */
    actions?: Snippet;
  }

  let {
    title,
    breadcrumb = [],
    meta,
    actions,
  }: PageHeaderProps = $props();
</script>

<header class="af-ph">
  <!-- Breadcrumb -->
  {#if breadcrumb.length > 0}
    <nav class="af-ph__breadcrumb" aria-label="Breadcrumb">
      <ol class="af-ph__crumb-list">
        {#each breadcrumb as crumb, i}
          <li class="af-ph__crumb-item">
            {#if crumb.href}
              <a class="af-ph__crumb-link af-meta" href={crumb.href}>{crumb.label}</a>
            {:else}
              <span class="af-ph__crumb-current af-meta">{crumb.label}</span>
            {/if}
            {#if i < breadcrumb.length - 1}
              <span class="af-ph__crumb-sep" aria-hidden="true">
                <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                  <path d="M3.5 2L6.5 5L3.5 8" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
              </span>
            {/if}
          </li>
        {/each}
      </ol>
    </nav>
  {/if}

  <!-- Title row -->
  <div class="af-ph__row">
    <div class="af-ph__title-group">
      <h1 class="af-page-title">{title}</h1>
      {#if meta}
        <div class="af-ph__meta">
          {@render meta()}
        </div>
      {/if}
    </div>

    {#if actions}
      <div class="af-ph__actions" role="toolbar" aria-label="{title} actions">
        {@render actions()}
      </div>
    {/if}
  </div>
</header>

<style>
  .af-ph {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
    padding-block-end: var(--af-space-4);
    border-bottom: 1px solid var(--af-border);
    margin-block-end: var(--af-space-4);
  }

  /* ── Breadcrumb ─────────────────────────────────────────────────────────── */
  .af-ph__crumb-list {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0;
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .af-ph__crumb-item {
    display: flex;
    align-items: center;
  }

  .af-ph__crumb-link {
    color: var(--af-text-secondary);
    text-decoration: none;
    font-size: var(--af-text-xs);
    transition: color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-ph__crumb-link:hover {
    color: var(--af-text);
  }

  .af-ph__crumb-link:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 2px;
    border-radius: 2px;
  }

  .af-ph__crumb-current {
    color: var(--af-text-muted);
    font-size: var(--af-text-xs);
  }

  .af-ph__crumb-sep {
    display: inline-flex;
    align-items: center;
    color: var(--af-text-muted);
    margin: 0 var(--af-space-1);
  }

  /* ── Title row ──────────────────────────────────────────────────────────── */
  .af-ph__row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--af-space-4);
    flex-wrap: wrap;
  }

  .af-ph__title-group {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    flex-wrap: wrap;
    min-width: 0;
  }

  /* .af-page-title from base.css: Space Grotesk, 24px, 700, -0.02em */
  .af-ph__title-group :global(.af-page-title) {
    flex-shrink: 0;
  }

  .af-ph__meta {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
  }

  /* ── Actions ────────────────────────────────────────────────────────────── */
  .af-ph__actions {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
    flex-shrink: 0;
  }
</style>
