<script lang="ts">
  import type { HTMLButtonAttributes } from 'svelte/elements';
  import type { Snippet } from 'svelte';

  export interface ButtonProps extends Omit<HTMLButtonAttributes, 'disabled'> {
    /** Visual hierarchy: primary uses inverse-surface (executive monochrome). */
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
    /** Control height tier. Both honour var(--af-control-height). */
    size?: 'sm' | 'md';
    /** HTML button type. */
    type?: 'button' | 'submit' | 'reset';
    /** Renders inline spinner; preserves button width; sets aria-disabled. */
    loading?: boolean;
    /** Disables the element; sets aria-disabled. */
    disabled?: boolean;
    /** Optional leading icon or decorative content rendered before the label. */
    icon?: Snippet;
    /** Label / default content. */
    children?: Snippet;
  }

  let {
    variant = 'primary',
    size = 'md',
    type = 'button',
    loading = false,
    disabled = false,
    icon,
    children,
    class: extraClass,
    ...restProps
  }: ButtonProps = $props();

  const isDisabled = $derived(disabled || loading);
</script>

<button
  {type}
  class="af-btn af-btn--{variant} af-btn--{size} {extraClass ?? ''}"
  class:af-btn--loading={loading}
  disabled={isDisabled}
  aria-disabled={isDisabled}
  aria-busy={loading}
  {...restProps}
>
  {#if loading}
    <span class="af-btn__spinner" aria-hidden="true"></span>
  {:else if icon}
    <span class="af-btn__icon" aria-hidden="true">
      {@render icon()}
    </span>
  {/if}
  {@render children?.()}
</button>

<style>
  /* ── Base ──────────────────────────────────────────────────────────── */
  .af-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--af-space-2);
    min-height: var(--af-control-height);
    padding: 0 var(--af-space-4);
    border: 1px solid transparent;
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-weight: var(--af-weight-semibold);
    font-size: var(--af-text-sm);
    line-height: 1;
    cursor: pointer;
    white-space: nowrap;
    user-select: none;
    text-decoration: none;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      transform var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      opacity var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-btn:active:not(:disabled):not(.af-btn--loading) {
    transform: scale(0.985);
  }

  .af-btn:disabled,
  .af-btn--loading {
    cursor: not-allowed;
    opacity: 0.46;
    pointer-events: none;
  }

  /* Restore pointer-events for loading so aria-busy stays accessible */
  .af-btn--loading {
    pointer-events: none;
    opacity: 1; /* spinner overrides — keep full opacity so width is stable */
  }

  /* ── Sizes ─────────────────────────────────────────────────────────── */
  .af-btn--sm {
    min-height: calc(var(--af-control-height) - 6px);
    padding: 0 var(--af-space-3);
    font-size: var(--af-text-xs);
    gap: var(--af-space-1);
  }

  /* Touch devices get the full tap target (§2.6); refined desktop sizing stays. */
  @media (pointer: coarse) {
    .af-btn,
    .af-btn--sm {
      min-height: var(--af-tap-min);
    }
  }

  /* ── Primary — executive monochrome (inverse-surface, NOT accent) ─── */
  .af-btn--primary {
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    border-color: var(--af-inverse-surface);
  }

  .af-btn--primary:hover:not(:disabled):not(.af-btn--loading) {
    background: color-mix(in srgb, var(--af-inverse-surface) 88%, transparent);
    box-shadow: var(--af-shadow-lift);
  }

  .af-btn--primary:active:not(:disabled):not(.af-btn--loading) {
    background: var(--af-inverse-surface);
    box-shadow: none;
  }

  /* ── Secondary — border + surface ─────────────────────────────────── */
  .af-btn--secondary {
    background: var(--af-surface);
    color: var(--af-text);
    border-color: var(--af-border-strong);
  }

  .af-btn--secondary:hover:not(:disabled):not(.af-btn--loading) {
    background: var(--af-surface-raised);
    border-color: var(--af-text-muted);
    box-shadow: var(--af-shadow-sm);
  }

  .af-btn--secondary:active:not(:disabled):not(.af-btn--loading) {
    background: var(--af-tint);
  }

  /* ── Ghost — no chrome, text-secondary, tint on hover ─────────────── */
  .af-btn--ghost {
    background: transparent;
    color: var(--af-text-secondary);
    border-color: transparent;
  }

  .af-btn--ghost:hover:not(:disabled):not(.af-btn--loading) {
    background: var(--af-tint);
    color: var(--af-text);
  }

  .af-btn--ghost:active:not(:disabled):not(.af-btn--loading) {
    background: var(--af-tint-medium);
  }

  /* ── Danger — ghost style, danger text, danger tint on hover ──────── */
  .af-btn--danger {
    background: transparent;
    color: var(--af-danger);
    border-color: transparent;
  }

  .af-btn--danger:hover:not(:disabled):not(.af-btn--loading) {
    background: var(--af-danger-tint);
    color: var(--af-danger);
  }

  .af-btn--danger:active:not(:disabled):not(.af-btn--loading) {
    background: var(--af-danger-tint);
    opacity: 0.8;
  }

  /* ── Loading spinner ───────────────────────────────────────────────── */
  .af-btn__spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid currentColor;
    border-block-start-color: transparent;
    border-radius: var(--af-radius-pill);
    animation: af-btn-spin var(--af-motion-spin) linear infinite;
    flex-shrink: 0;
  }

  @keyframes af-btn-spin {
    to { transform: rotate(360deg); }
  }

  /* ── Icon slot ─────────────────────────────────────────────────────── */
  .af-btn__icon {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
  }

  /* ── Reduced motion backstop ────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .af-btn__spinner {
      animation: none;
      opacity: 0.5;
    }
  }
</style>
