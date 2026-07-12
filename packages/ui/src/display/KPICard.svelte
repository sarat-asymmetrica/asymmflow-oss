<script lang="ts">
  import type { Snippet } from 'svelte';

  export interface KPICardProps {
    /** Upper label — .af-label typography (xs, 600, uppercase, tracked) */
    label: string;
    /** The headline metric — .af-numeric, text-3xl, bold */
    value: string;
    /** Smaller secondary line below the value */
    meta?: string;
    /**
     * 'north-star': inverse-surface background, display-size value.
     * Use for the single hero KPI on a dashboard.
     */
    variant?: 'default' | 'north-star';
    /** Optional trend Snippet — renders inline after the value */
    trend?: Snippet;
    [key: string]: unknown;
  }

  let {
    label,
    value,
    meta,
    variant = 'default',
    trend,
    ...restProps
  }: KPICardProps = $props();
</script>

<div
  class="af-kpi af-kpi--{variant}"
  {...restProps}
>
  <span class="af-label af-kpi__label">{label}</span>

  <div class="af-kpi__value-row">
    <span class="af-numeric af-kpi__value">{value}</span>
    {#if trend}
      <span class="af-kpi__trend">
        {@render trend()}
      </span>
    {/if}
  </div>

  {#if meta}
    <span class="af-meta af-kpi__meta">{meta}</span>
  {/if}
</div>

<style>
  .af-kpi {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
    /* R2 micro-interaction on hover */
    transition:
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-kpi:hover {
    box-shadow: var(--af-shadow-lift);
  }

  /* Label */
  .af-kpi__label {
    /* Inherits .af-label: xs, 600, uppercase, tracked */
    display: block;
  }

  /* Value row */
  .af-kpi__value-row {
    display: flex;
    align-items: baseline;
    gap: var(--af-space-2);
    flex-wrap: wrap;
  }

  .af-kpi__value {
    font-size: var(--af-text-3xl);
    font-weight: var(--af-weight-bold);
    line-height: var(--af-leading-tight);
    color: var(--af-text);
    /* Numeric features inherited from .af-numeric (base.css) */
    /* Contain long unbroken figures inside the card rather than overflowing.
       overflow-wrap (not truncation) keeps every digit legible. */
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .af-kpi__trend {
    display: flex;
    align-items: center;
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-semibold);
    color: var(--af-text-secondary);
  }

  .af-kpi__meta {
    /* Inherits .af-meta: xs, regular, muted */
    display: block;
    margin-block-start: var(--af-space-1);
  }

  /* === North-star variant === */
  .af-kpi--north-star {
    background: var(--af-inverse-surface);
    border-color: transparent;
    color: var(--af-text-inverse);
  }

  .af-kpi--north-star .af-kpi__label {
    color: var(--af-text-inverse);
    opacity: 0.6;
  }

  .af-kpi--north-star .af-kpi__value {
    /* Display size for the hero north-star KPI */
    font-size: var(--af-text-display);
    color: var(--af-text-inverse);
  }

  .af-kpi--north-star .af-kpi__meta {
    color: var(--af-text-inverse);
    opacity: 0.5;
  }

  .af-kpi--north-star .af-kpi__trend {
    color: var(--af-text-inverse);
    opacity: 0.75;
  }

  .af-kpi--north-star:hover {
    box-shadow: none;
  }
</style>
