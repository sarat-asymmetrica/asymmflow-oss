<script lang="ts">
  /**
   * Pagination — @asymmflow/ui
   *
   * Page controls: prev/next + numeric window + "x–y of z" meta.
   * Fully controlled: bind:page for two-way binding.
   * All interactive targets ≥ 44px (§2.6).
   * Numeric meta uses .af-numeric (tabular-nums, §4a).
   *
   * Constitution: packages/DESIGN_CONSTITUTION.md
   */

  interface Props {
    /** Current page (1-indexed). Bindable. */
    page?: number;
    /** Rows per page. */
    pageSize?: number;
    /** Total row count across all pages. */
    total: number;
    /** Max page buttons to show in the window (excluding prev/next). Default: 5. */
    window?: number;
    /** Callback fired when page changes (supplemental to bind:page). */
    onPageChange?: (page: number) => void;
  }

  let {
    page = $bindable(1),
    pageSize = 20,
    total,
    window: windowSize = 5,
    onPageChange,
  }: Props = $props();

  const totalPages = $derived(Math.max(1, Math.ceil(total / pageSize)));

  // Page must stay in range
  $effect(() => {
    if (page < 1) page = 1;
    if (page > totalPages) page = totalPages;
  });

  // x–y of z
  const firstItem = $derived(total === 0 ? 0 : (page - 1) * pageSize + 1);
  const lastItem = $derived(Math.min(page * pageSize, total));

  // Numeric page window
  const pages = $derived.by(() => {
    const half = Math.floor(windowSize / 2);
    let start = Math.max(1, page - half);
    let end = start + windowSize - 1;
    if (end > totalPages) {
      end = totalPages;
      start = Math.max(1, end - windowSize + 1);
    }
    const arr: (number | null)[] = [];
    if (start > 1) {
      arr.push(1);
      if (start > 2) arr.push(null); // ellipsis
    }
    for (let i = start; i <= end; i++) arr.push(i);
    if (end < totalPages) {
      if (end < totalPages - 1) arr.push(null);
      arr.push(totalPages);
    }
    return arr;
  });

  function goTo(p: number) {
    if (p < 1 || p > totalPages || p === page) return;
    page = p;
    onPageChange?.(p);
  }
</script>

<nav class="af-pagination" aria-label="Table pagination">
  <!-- x–y of z -->
  <span class="af-pagination__meta af-meta af-numeric" aria-live="polite" aria-atomic="true">
    {#if total === 0}
      0 results
    {:else}
      {firstItem}–{lastItem} of {total.toLocaleString('en-US')}
    {/if}
  </span>

  <div class="af-pagination__controls" role="group" aria-label="Page navigation">
    <!-- Prev -->
    <button
      class="af-pg-btn af-pg-btn--nav"
      aria-label="Previous page"
      disabled={page <= 1}
      onclick={() => goTo(page - 1)}
    >
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
        <path d="M9.5 11L6.5 8L9.5 5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    </button>

    <!-- Page number window -->
    {#each pages as p}
      {#if p === null}
        <span class="af-pg-ellipsis af-meta" aria-hidden="true">…</span>
      {:else}
        <button
          class="af-pg-btn"
          class:af-pg-btn--active={p === page}
          aria-label="Page {p}"
          aria-current={p === page ? 'page' : undefined}
          onclick={() => goTo(p)}
        >
          <span class="af-numeric">{p}</span>
        </button>
      {/if}
    {/each}

    <!-- Next -->
    <button
      class="af-pg-btn af-pg-btn--nav"
      aria-label="Next page"
      disabled={page >= totalPages}
      onclick={() => goTo(page + 1)}
    >
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
        <path d="M6.5 5L9.5 8L6.5 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    </button>
  </div>
</nav>

<style>
  .af-pagination {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--af-space-4);
    padding: var(--af-space-2) var(--af-space-3);
    border-top: 1px solid var(--af-border);
  }

  .af-pagination__meta {
    color: var(--af-text-muted);
    font-size: var(--af-text-xs);
    font-family: var(--af-font-numeric);
    font-variant-numeric: tabular-nums lining-nums;
    font-feature-settings: var(--af-numeric-features);
  }

  .af-pagination__controls {
    display: flex;
    align-items: center;
    gap: var(--af-space-1);
  }

  /* ── Page button ────────────────────────────────────────────────────────── */
  .af-pg-btn {
    /* ≥44px touch target (§2.6) */
    min-width: var(--af-tap-min);
    min-height: var(--af-tap-min);
    padding: 0 var(--af-space-2);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 1px solid transparent;
    border-radius: var(--af-radius-sm);
    background: transparent;
    color: var(--af-text-secondary);
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-sm);
    font-variant-numeric: tabular-nums lining-nums;
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-pg-btn:hover:not(:disabled) {
    background: var(--af-tint);
    color: var(--af-text);
  }

  .af-pg-btn:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  .af-pg-btn:active:not(:disabled) {
    background: var(--af-tint-medium);
  }

  .af-pg-btn:disabled {
    opacity: 0.36;
    cursor: not-allowed;
    pointer-events: none;
  }

  /* Active page */
  .af-pg-btn--active {
    background: var(--af-inverse-surface);
    border-color: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    font-weight: var(--af-weight-semibold);
  }

  .af-pg-btn--active:hover {
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
  }

  /* Nav buttons (prev/next): slightly narrower, icon only */
  .af-pg-btn--nav {
    min-width: var(--af-tap-min);
    color: var(--af-text-muted);
  }

  .af-pg-btn--nav:hover:not(:disabled) {
    color: var(--af-text);
  }

  .af-pg-ellipsis {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 32px;
    min-height: var(--af-tap-min);
    color: var(--af-text-muted);
    font-size: var(--af-text-sm);
    user-select: none;
  }

  /* ── Reduced motion ─────────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .af-pg-btn {
      transition: none;
    }
  }
</style>
