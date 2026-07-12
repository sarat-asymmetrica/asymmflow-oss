<script lang="ts">
  /**
   * Drawer — side-panel layered surface for detail views (OpportunityDetail, OrderDetail).
   *
   * Constitution §2.6: focus trap mandatory.
   * Constitution §2.7: interruptible motion — R1 slide-in explore, R3 slide-out stabilize.
   * Constitution §4d: --af-shadow-overlay only on layered surfaces.
   * Constitution §4e: translate only (GPU-composited); no width animation.
   */
  import type { Snippet } from 'svelte';
  import { focusTrap } from '../actions/focusTrap.js';
  import { portal } from './portal.js';

  export interface DrawerProps {
    /** Bindable open state. */
    open?: boolean;
    /** Which edge the drawer slides in from. Default right. */
    side?: 'right' | 'left';
    /**
     * Width tier:
     *   sm  = 360px   — compact detail
     *   md  = 480px   — default detail view
     *   lg  = 640px   — wide detail / split-panel
     *   full = 100vw  — full-width (mobile or immersive)
     */
    size?: 'sm' | 'md' | 'lg' | 'full';
    /** Click scrim to close. Default true. */
    closeOnScrim?: boolean;
    /** String title rendered in the header. Omit when providing header Snippet. */
    title?: string;
    /** Custom header Snippet. */
    header?: Snippet;
    /** Body content Snippet. */
    children?: Snippet;
    /** Footer Snippet — action buttons, save/cancel, etc. */
    footer?: Snippet;
  }

  let {
    open = $bindable(false),
    side = 'right',
    size = 'md',
    closeOnScrim = true,
    title = '',
    header,
    children,
    footer,
  }: DrawerProps = $props();

  const titleId = $props.id();

  $effect(() => {
    if (open) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  });

  function close() {
    open = false;
  }

  function handleScrimPointer(e: PointerEvent) {
    if (closeOnScrim && e.target === e.currentTarget) close();
  }
</script>

{#if open}
  <div use:portal>
    <div
      class="af-drawer-scrim"
      role="presentation"
      onpointerdown={handleScrimPointer}
    >
      <div
        class="af-drawer af-drawer--{size} af-drawer--{side}"
        role="dialog"
        aria-modal="true"
        aria-labelledby={title || !header ? titleId : undefined}
        use:focusTrap={{ active: open, onEscape: close }}
      >
        <!-- Header -->
        <header class="af-drawer__header">
          {#if header}
            {@render header()}
          {:else}
            <h2 id={titleId} class="af-drawer__title">{title}</h2>
          {/if}
          <button
            class="af-drawer__close"
            onclick={close}
            aria-label="Close panel"
            type="button"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <path
                d="M3 3L13 13M13 3L3 13"
                stroke="currentColor"
                stroke-width="1.75"
                stroke-linecap="round"
              />
            </svg>
          </button>
        </header>

        <!-- Body -->
        <div class="af-drawer__body">
          {@render children?.()}
        </div>

        <!-- Footer -->
        {#if footer}
          <footer class="af-drawer__footer">
            {@render footer()}
          </footer>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  /* ── Scrim ──────────────────────────────────────────────────────────────── */
  .af-drawer-scrim {
    position: fixed;
    inset: 0;
    background: var(--af-scrim);
    display: flex;
    z-index: var(--af-z-modal);
    animation: af-drawer-scrim-in var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
  }

  @keyframes af-drawer-scrim-in {
    from { opacity: 0; }
    to   { opacity: 1; }
  }

  /* ── Panel ────────────────────────────────────────────────────────────── */
  .af-drawer {
    position: absolute;
    top: 0;
    bottom: 0;
    background: var(--af-surface);
    box-shadow: var(--af-shadow-overlay);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    /* R1 explore — slides in on the explore curve */
    animation: af-drawer-in-right var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
  }

  /* Side positioning */
  .af-drawer--right { right: 0; border-radius: var(--af-radius-lg) 0 0 var(--af-radius-lg); }
  .af-drawer--left  { left: 0;  border-radius: 0 var(--af-radius-lg) var(--af-radius-lg) 0; }

  /* Sizes */
  .af-drawer--sm   { width: 360px; }
  .af-drawer--md   { width: 480px; }
  .af-drawer--lg   { width: 640px; }
  .af-drawer--full { width: 100vw; border-radius: 0; }

  /* Slide animations — right (default) */
  .af-drawer--right {
    animation-name: af-drawer-in-right;
  }

  @keyframes af-drawer-in-right {
    from {
      opacity: 0;
      transform: translateX(32px);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }

  /* Slide animations — left */
  .af-drawer--left {
    animation-name: af-drawer-in-left;
  }

  @keyframes af-drawer-in-left {
    from {
      opacity: 0;
      transform: translateX(-32px);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }

  /* ── Header ───────────────────────────────────────────────────────────── */
  .af-drawer__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--af-space-3);
    padding: var(--af-space-4);
    border-bottom: 1px solid var(--af-border);
    flex-shrink: 0;
  }

  .af-drawer__title {
    font-family: var(--af-font-display);
    font-size: var(--af-text-xl);
    font-weight: var(--af-weight-semibold);
    letter-spacing: var(--af-title-tracking);
    line-height: var(--af-leading-tight);
    color: var(--af-text);
    margin: 0;
  }

  .af-drawer__close {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    /* Tap-target-sized close (§2.6 accessibility floor) */
    width: var(--af-tap-min);
    height: var(--af-tap-min);
    border-radius: var(--af-radius-sm);
    border: none;
    background: transparent;
    color: var(--af-text-secondary);
    cursor: pointer;
    flex-shrink: 0;
    margin-inline-end: calc(var(--af-space-2) * -1);
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-drawer__close:hover {
    background: var(--af-tint);
    color: var(--af-text);
  }

  .af-drawer__close:active {
    background: var(--af-tint-medium);
  }

  /* ── Body ─────────────────────────────────────────────────────────────── */
  .af-drawer__body {
    padding: var(--af-space-4);
    overflow-y: auto;
    flex: 1;
    min-height: 0;
    font-size: var(--af-text-md);
    line-height: var(--af-leading-base);
    color: var(--af-text);
  }

  /* ── Footer ───────────────────────────────────────────────────────────── */
  .af-drawer__footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: var(--af-space-3);
    padding: var(--af-space-3) var(--af-space-4);
    border-top: 1px solid var(--af-border);
    flex-shrink: 0;
  }

  /* ── Reduced motion ───────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .af-drawer-scrim,
    .af-drawer,
    .af-drawer--right,
    .af-drawer--left {
      animation: af-drawer-opacity-only var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
    }

    @keyframes af-drawer-opacity-only {
      from { opacity: 0; }
      to   { opacity: 1; }
    }
  }
</style>
