<script lang="ts">
  /**
   * Modal — THE layered dialog surface.
   *
   * Constitution §2.6: focus containment mandatory.
   * Constitution §2.7: interruptible motion (R1 explore entrance, R3 stabilize exit).
   * Constitution §4d: --af-shadow-overlay is the ONLY place true elevation lives.
   * Constitution §4e: opacity + transform only, no width/height/top/left animation.
   * Constitution §2.5: prefers-reduced-motion collapses entrances to opacity-only.
   *
   * Uses existing actions: focusTrap (ui/src/actions/focusTrap.ts),
   *                        portal (overlay/portal.ts — owned directory).
   */
  import type { Snippet } from 'svelte';
  import { focusTrap } from '../actions/focusTrap.js';
  import { portal } from './portal.js';

  export interface ModalProps {
    /** Bindable open state. */
    open?: boolean;
    /** String title — rendered in the header with af-section-title styling.
     *  Omit when providing a custom header Snippet. */
    title?: string;
    /** Width tier. */
    size?: 'sm' | 'md' | 'lg' | 'full';
    /** Whether clicking the scrim closes the modal. Default true. */
    closeOnScrim?: boolean;
    /** Custom header Snippet replaces the default title + close button row. */
    header?: Snippet;
    /** Body content. */
    children?: Snippet;
    /** Footer Snippet — typically action buttons. */
    footer?: Snippet;
  }

  let {
    open = $bindable(false),
    title = '',
    size = 'md',
    closeOnScrim = true,
    header,
    children,
    footer,
  }: ModalProps = $props();

  // Unique id for aria-labelledby linkage.
  const titleId = $props.id();

  // Body scroll lock — pairs with portal so the lock is always on <body>.
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

  // Expose close to focusTrap's onEscape.
  function handleEscape() {
    close();
  }
</script>

{#if open}
  <!-- Portal hoists the entire overlay to document.body, escaping stacking contexts. -->
  <div use:portal>
    <!--
      Scrim layer — receives click-to-close, provides --af-scrim backdrop.
      role="presentation" (not dialog) so screenreaders address the inner dialog only.
    -->
    <div
      class="af-modal-scrim"
      role="presentation"
      onpointerdown={handleScrimPointer}
    >
      <!--
        Dialog surface — focus trap lives here, not on the scrim,
        so Tab cycling stays inside the dialog panel.
      -->
      <div
        class="af-modal af-modal--{size}"
        role="dialog"
        aria-modal="true"
        aria-labelledby={title || !header ? titleId : undefined}
        use:focusTrap={{ active: open, onEscape: handleEscape }}
      >
        <!-- Header -->
        <header class="af-modal__header">
          {#if header}
            {@render header()}
          {:else}
            <h2 id={titleId} class="af-modal__title">{title}</h2>
          {/if}
          <button
            class="af-modal__close"
            onclick={close}
            aria-label="Close dialog"
            type="button"
          >
            <!-- 20×20 X, stroke only, no fill. No emoji (constitution §3). -->
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

        <!-- Body — scrolls independently; header + footer stay sticky -->
        <div class="af-modal__body">
          {@render children?.()}
        </div>

        <!-- Footer — optional; floats action buttons -->
        {#if footer}
          <footer class="af-modal__footer">
            {@render footer()}
          </footer>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  /* ── Scrim ──────────────────────────────────────────────────────────────── */
  .af-modal-scrim {
    position: fixed;
    inset: 0;
    background: var(--af-scrim);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--af-space-4);
    z-index: var(--af-z-modal);

    /* R1 explore — scrim fades in (opacity only, no transform) */
    animation: af-modal-scrim-in var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
  }

  @keyframes af-modal-scrim-in {
    from { opacity: 0; }
    to   { opacity: 1; }
  }

  /* ── Dialog panel ─────────────────────────────────────────────────────── */
  .af-modal {
    background: var(--af-surface);
    border-radius: var(--af-radius-lg);
    /* §4d: only layered surfaces get true elevation */
    box-shadow: var(--af-shadow-overlay);
    display: flex;
    flex-direction: column;
    /* Prevent modal from taller than viewport with padding for visual breathing room */
    max-height: calc(100dvh - var(--af-space-6) * 2);
    width: 100%;
    overflow: hidden;

    /* R1 explore — panel rises 16px, synced with scrim */
    animation: af-modal-panel-in var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
  }

  @keyframes af-modal-panel-in {
    from {
      opacity: 0;
      transform: translateY(16px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  /* ── Sizes ────────────────────────────────────────────────────────────── */
  .af-modal--sm   { max-width: 400px; }
  .af-modal--md   { max-width: 560px; }
  .af-modal--lg   { max-width: 840px; }
  .af-modal--full {
    max-width: calc(100dvw - var(--af-space-5));
    max-height: calc(100dvh - var(--af-space-5));
  }

  /* ── Header ───────────────────────────────────────────────────────────── */
  .af-modal__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--af-space-3);
    padding: var(--af-space-4) var(--af-space-4) var(--af-space-3);
    border-bottom: 1px solid var(--af-border);
    flex-shrink: 0;
  }

  .af-modal__title {
    font-family: var(--af-font-display);
    font-size: var(--af-text-xl);
    font-weight: var(--af-weight-semibold);
    letter-spacing: var(--af-title-tracking);
    line-height: var(--af-leading-tight);
    color: var(--af-text);
    margin: 0;
  }

  .af-modal__close {
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
    /* Negative margin so visual padding aligns with header edge */
    margin-inline-end: calc(var(--af-space-2) * -1);
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-modal__close:hover {
    background: var(--af-tint);
    color: var(--af-text);
  }

  .af-modal__close:active {
    background: var(--af-tint-medium);
  }

  /* ── Body ─────────────────────────────────────────────────────────────── */
  .af-modal__body {
    padding: var(--af-space-4);
    overflow-y: auto;
    flex: 1;
    min-height: 0;
    font-size: var(--af-text-md);
    line-height: var(--af-leading-base);
    color: var(--af-text);
  }

  /* ── Footer ───────────────────────────────────────────────────────────── */
  .af-modal__footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: var(--af-space-3);
    padding: var(--af-space-3) var(--af-space-4);
    border-top: 1px solid var(--af-border);
    flex-shrink: 0;
  }

  /* ── Reduced motion — entrances collapse to opacity only ─────────────── */
  @media (prefers-reduced-motion: reduce) {
    .af-modal-scrim,
    .af-modal {
      animation: af-modal-opacity-only var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
    }

    @keyframes af-modal-opacity-only {
      from { opacity: 0; }
      to   { opacity: 1; }
    }
  }
</style>
