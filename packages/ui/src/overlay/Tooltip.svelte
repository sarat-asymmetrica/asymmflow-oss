<script lang="ts">
  /**
   * Tooltip — hover / focus tooltip wrapping a trigger Snippet.
   *
   * Design rules:
   *   - Inverse surface background, text-inverse text (§4c monochrome).
   *   - text-xs, max-width 36ch.
   *   - ~500ms show delay (R2 optimize × 3.57 ≈ computed from token).
   *   - Instant out (0ms delay).
   *   - role=tooltip + aria-describedby wired by this component.
   *   - NEVER traps pointer.
   *   - Opacity-only animation (tiny surface, no 16px rise needed).
   *   - Positions: top (default) | bottom | left | right.
   *   - No raw hex, no raw ms — delay computed from CSS var fallback.
   *
   * Constitution §2.5: reduced motion — opacity stays, no transform.
   * Constitution §3: no glassmorphism, no indigo, no blur.
   */
  import type { Snippet } from 'svelte';
  import { TOOLTIP_DELAY_MS } from '@asymmflow/tokens';

  export interface TooltipProps {
    /** Text content of the tooltip bubble. */
    text: string;
    /** Preferred placement relative to trigger. Tooltip stays within viewport. */
    position?: 'top' | 'bottom' | 'left' | 'right';
    /** The interactive element the tooltip annotates. */
    children: Snippet;
  }

  let { text, position = 'top', children }: TooltipProps = $props();

  let visible = $state(false);
  let timer: ReturnType<typeof setTimeout> | null = null;
  const tooltipId = $props.id();

  // Hover-intent delay sourced from the single token contract — no raw ms here.
  function show() {
    timer = setTimeout(() => {
      visible = true;
    }, TOOLTIP_DELAY_MS);
  }

  function hide() {
    if (timer !== null) {
      clearTimeout(timer);
      timer = null;
    }
    visible = false;
  }
</script>

<!--
  Wrapper must be inline-flex so it doesn't disrupt flow.
  tabindex=-1 ensures keyboard users who focus the child trigger
  the onfocus event via event bubbling.
  a11y: this span is a non-interactive positioning wrapper — the child
  trigger (passed via the children snippet) carries the interactive role,
  and the tooltip itself has role="tooltip". The hover/focus handlers only
  toggle a supplementary tooltip, so no ARIA role applies to the wrapper.
-->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<span
  class="af-tooltip-wrap"
  onmouseenter={show}
  onmouseleave={hide}
  onfocusin={show}
  onfocusout={hide}
>
  <!-- aria-describedby wires the trigger to the tooltip text for AT. -->
  <span aria-describedby={visible ? tooltipId : undefined}>
    {@render children()}
  </span>

  {#if visible && text}
    <div
      id={tooltipId}
      class="af-tooltip af-tooltip--{position}"
      role="tooltip"
    >
      {text}
    </div>
  {/if}
</span>

<style>
  /* ── Wrapper ──────────────────────────────────────────────────────────── */
  .af-tooltip-wrap {
    position: relative;
    display: inline-flex;
    align-items: center;
  }

  /* ── Bubble ───────────────────────────────────────────────────────────── */
  .af-tooltip {
    position: absolute;
    /* Never traps pointer — pointer-events: none is mandatory. */
    pointer-events: none;
    z-index: var(--af-z-tooltip);
    /* Inverse surface (§4c: monochrome, not colored) */
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-regular);
    line-height: var(--af-leading-base);
    /* 36ch max-width keeps lines scannable */
    max-width: 36ch;
    padding: var(--af-space-2) var(--af-space-3);
    border-radius: var(--af-radius-sm);
    /* Instant fade in (appearance is driven by JS show delay, not CSS) */
    animation: af-tooltip-in var(--af-motion-optimize-duration) var(--af-motion-optimize-ease) both;
    white-space: normal;
    word-break: break-word;
  }

  @keyframes af-tooltip-in {
    from { opacity: 0; }
    to   { opacity: 1; }
  }

  /* ── Positions ────────────────────────────────────────────────────────── */
  /* top (default) — centers above trigger */
  .af-tooltip--top {
    bottom: calc(100% + var(--af-space-2));
    left: 50%;
    transform: translateX(-50%);
  }

  /* bottom — centers below trigger */
  .af-tooltip--bottom {
    top: calc(100% + var(--af-space-2));
    left: 50%;
    transform: translateX(-50%);
  }

  /* left — centers to the left */
  .af-tooltip--left {
    right: calc(100% + var(--af-space-2));
    top: 50%;
    transform: translateY(-50%);
  }

  /* right — centers to the right */
  .af-tooltip--right {
    left: calc(100% + var(--af-space-2));
    top: 50%;
    transform: translateY(-50%);
  }

  /* ── Reduced motion ── opacity already handles this; no transform to drop */
  @media (prefers-reduced-motion: reduce) {
    .af-tooltip {
      animation: none;
    }
  }
</style>
