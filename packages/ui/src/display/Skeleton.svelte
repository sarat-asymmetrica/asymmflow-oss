<script lang="ts">
  /**
   * Skeleton — shimmer loading placeholder.
   *
   * Composable: nest multiple Skeleton blocks to represent complex layouts.
   * Variants: text/line (inline-width, short height), rect (block), circle.
   * Shimmer uses tokens only — no raw hex.
   * prefers-reduced-motion: collapses animation (global backstop in base.css covers it,
   * but we also provide a no-animation path explicitly).
   */

  export interface SkeletonProps {
    variant?: 'text' | 'rect' | 'circle';
    /** Width — any CSS value. Defaults vary by variant */
    width?: string;
    /** Height — any CSS value. Defaults vary by variant */
    height?: string;
    [key: string]: unknown;
  }

  let { variant = 'rect', width, height, ...restProps }: SkeletonProps = $props();

  // Sensible defaults per variant
  const defaultW: Record<string, string> = {
    text: '80%',
    rect: '100%',
    circle: '40px',
  };
  const defaultH: Record<string, string> = {
    text: '1em',
    rect: '80px',
    circle: '40px',
  };

  const w = $derived(width ?? defaultW[variant]);
  const h = $derived(height ?? defaultH[variant]);
</script>

<span
  class="af-skeleton af-skeleton--{variant}"
  style:width={w}
  style:height={h}
  aria-hidden="true"
  {...restProps}
></span>

<style>
  .af-skeleton {
    display: block;
    position: relative;
    overflow: hidden;
    border-radius: var(--af-radius-sm);
    background: var(--af-surface-sunken);
  }

  /*
   * Shimmer = a highlight band swept across with transform: translateX
   * (GPU-composited, §4e "opacity + transform only"). The earlier
   * background-position approach forced a continuous paint on every skeleton.
   */
  .af-skeleton::after {
    content: '';
    position: absolute;
    inset: 0;
    background-image: linear-gradient(
      90deg,
      transparent 0%,
      var(--af-surface-raised) 50%,
      transparent 100%
    );
    transform: translateX(-100%);
    animation: af-shimmer var(--af-motion-shimmer) linear infinite;
    /* Continuous loop — linear is the ONE allowed linear easing */
  }

  .af-skeleton--text {
    border-radius: var(--af-radius-pill);
    max-width: 100%;
  }

  .af-skeleton--circle {
    border-radius: var(--af-radius-pill);
    flex-shrink: 0;
  }

  .af-skeleton--rect {
    border-radius: var(--af-radius-md);
  }

  @keyframes af-shimmer {
    to {
      transform: translateX(100%);
    }
  }

  /* prefers-reduced-motion: drop the sweep entirely — the sunken fill remains. */
  @media (prefers-reduced-motion: reduce) {
    .af-skeleton::after {
      animation: none;
      opacity: 0;
    }
  }
</style>
