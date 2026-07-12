<script lang="ts">
  /**
   * Spinner — THE system spinner, replaces WabiSpinner (43 call sites).
   *
   * Design: a quiet, sophisticated mark.
   * - Ring with a φ-proportioned arc: 137.5° (the golden angle) rotating continuously.
   * - Continuous linear rotation — the ONE allowed linear easing (constitution §4e).
   * - currentColor — inherits from parent, no hardcoded palette.
   * - role="status" + visually hidden label for a11y.
   * - prefers-reduced-motion: pulsing opacity instead of rotation.
   */

  export interface SpinnerProps {
    size?: 'sm' | 'md' | 'lg';
    /** Accessible label announced to screen readers */
    label?: string;
    [key: string]: unknown;
  }

  let { size = 'md', label = 'Loading', ...restProps }: SpinnerProps = $props();

  // Sizes in px
  const px = { sm: 16, md: 24, lg: 40 } as const;
  const stroke = { sm: 1.5, md: 2, lg: 2.5 } as const;
</script>

<span
  class="af-spinner af-spinner--{size}"
  role="status"
  aria-label={label}
  {...restProps}
>
  <!--
    SVG ring: the arc is 137.5° (golden angle ≈ φ × 85°).
    stroke-dasharray encodes arc length on the circumference.
    circumference = 2π × (r) where r = (viewBox/2 - strokeWidth/2).

    For a 24-unit viewBox: r = 10, circumference ≈ 62.83
    Golden arc = 137.5/360 × 62.83 ≈ 24.0 — neat.
    We use a proportional approach via CSS custom props so all sizes work.
  -->
  <svg
    class="af-spinner__svg"
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
    width={px[size]}
    height={px[size]}
  >
    <!-- Background track: faint full ring -->
    <circle
      class="af-spinner__track"
      cx="12"
      cy="12"
      r="10"
      stroke="currentColor"
      stroke-width={stroke[size]}
    />
    <!-- Active arc: 137.5° / 360° × 2π×10 ≈ 24.0 of 62.83 -->
    <circle
      class="af-spinner__arc"
      cx="12"
      cy="12"
      r="10"
      stroke="currentColor"
      stroke-width={stroke[size]}
      stroke-linecap="round"
      stroke-dasharray="24 38.83"
      stroke-dashoffset="0"
    />
  </svg>
  <span class="af-sr-only">{label}</span>
</span>

<style>
  .af-spinner {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    color: currentColor;
  }

  .af-spinner__svg {
    display: block;
    overflow: visible;
  }

  /* Track: very faint — the arc is the mark */
  .af-spinner__track {
    opacity: 0.12;
  }

  /* Arc: continuous linear rotation — the ONE allowed linear easing */
  .af-spinner__arc {
    transform-origin: 12px 12px;
    animation: af-spin var(--af-motion-spin) linear infinite;
  }

  @keyframes af-spin {
    to {
      transform: rotate(360deg);
    }
  }

  /* prefers-reduced-motion: replace rotation with quiet opacity pulse */
  @media (prefers-reduced-motion: reduce) {
    .af-spinner__arc {
      animation: af-spin-pulse var(--af-motion-pulse) ease-in-out infinite;
      transform: none;
    }

    @keyframes af-spin-pulse {
      0%, 100% { opacity: 0.2; }
      50% { opacity: 1; }
    }
  }
</style>
