<script lang="ts">
  /**
   * GlyphMark — THE generative identity mark.
   *
   * A seed string → deterministic quaternion walk on S³ → unique SVG sigil.
   * Same seed always produces the same mark. Looks like a signature, not a blob.
   * Constitution §5: "seeds → deterministic quaternion walks on S³ → unique-but-coherent marks."
   *
   * Props:
   *   seed         - The identity seed (e.g. company name). REQUIRED.
   *   size         - Tile dimension in px (default 64).
   *   strokeColor  - Override stroke color (default currentColor — inherits from CSS).
   *   bgToken      - CSS custom property name for tile background (default --af-surface-raised).
   *   animate      - If true, draw-on animation via stroke-dasharray reveal (default true).
   *   steps        - Number of walk vertices (default 18). More = more complex marks.
   *   stepAngle    - Max rotation per step in radians (default 0.65).
   */

  import { walkPoints, projectTo2D, smoothPath } from './quaternionWalk.js';

  interface Props {
    seed: string;
    size?: number;
    strokeColor?: string;
    bgToken?: string;
    animate?: boolean;
    steps?: number;
    stepAngle?: number;
  }

  let {
    seed,
    size = 64,
    strokeColor,
    bgToken = '--af-surface-raised',
    animate = true,
    steps = 18,
    stepAngle = 0.65,
  }: Props = $props();

  // Compute the path from the seed deterministically
  const pathData = $derived.by(() => {
    const raw = walkPoints(seed, steps, stepAngle);
    const pts = projectTo2D(raw, size, size, 0.12);
    return smoothPath(pts);
  });

  // Unique ID for the clipPath (scoped to seed+size to avoid collisions)
  const clipId = $derived(`glyph-clip-${seed.replace(/\W/g, '-')}-${size}`);

  // Stroke-dasharray animation: we use a sufficiently large value that covers
  // the path length without needing to measure it (SVGPathElement.getTotalLength
  // is not available SSR-side). 1000 is adequate for our size range 32–200px.
  const DASH_LEN = 1000;

  // Reduced motion: check at component init, re-check if preference changes
  let prefersReducedMotion = $state(
    typeof window !== 'undefined'
      ? window.matchMedia('(prefers-reduced-motion: reduce)').matches
      : false,
  );

  $effect(() => {
    if (typeof window === 'undefined') return;
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    const handler = (e: MediaQueryListEvent) => {
      prefersReducedMotion = e.matches;
    };
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  });

  const shouldAnimate = $derived(animate && !prefersReducedMotion);

  // Stroke width scales slightly with size (constitutional restraint)
  const strokeWidth = $derived(Math.max(1.5, size / 44));

  // Corner radius for the rounded-square tile
  const tileRadius = $derived(Math.round(size * 0.22)); // ≈ --af-radius-md feel
</script>

<!--
  Rounded-square tile containing the sigil.
  Stroke color: explicit prop or currentColor (CSS cascade handles theming).
  Background: reads from a CSS custom property token via inline style.
-->
<svg
  width={size}
  height={size}
  viewBox="0 0 {size} {size}"
  role="img"
  aria-label="Identity mark for {seed}"
  class="glyph-mark"
  style:--glyph-bg="var({bgToken})"
  style:--glyph-stroke={strokeColor ?? 'currentColor'}
>
  <defs>
    <clipPath id={clipId}>
      <rect width={size} height={size} rx={tileRadius} ry={tileRadius} />
    </clipPath>
  </defs>

  <!-- Tile background (token-driven) -->
  <rect
    width={size}
    height={size}
    rx={tileRadius}
    ry={tileRadius}
    fill="var(--glyph-bg)"
  />

  <!-- The sigil path, clipped to the tile -->
  <g clip-path="url(#{clipId})">
    {#key seed}
      <path
        d={pathData}
        fill="none"
        stroke="var(--glyph-stroke)"
        stroke-width={strokeWidth}
        stroke-linecap="round"
        stroke-linejoin="round"
        class:draw-on={shouldAnimate}
        style:--dash-len={DASH_LEN}
      />
    {/key}
  </g>
</svg>

<style>
  .glyph-mark {
    display: block;
    flex-shrink: 0;
  }

  /*
   * Draw-on animation: stroke-dasharray reveal at R1·Explore speed (400ms).
   * prefers-reduced-motion: the .draw-on class is NOT applied, so path renders
   * instantly at full opacity. No separate query needed — the $derived handles it.
   */
  .draw-on {
    stroke-dasharray: var(--dash-len);
    stroke-dashoffset: var(--dash-len);
    animation: glyph-draw var(--af-motion-explore-duration)
      var(--af-motion-explore-ease) forwards;
  }

  @keyframes glyph-draw {
    to {
      stroke-dashoffset: 0;
    }
  }
</style>
