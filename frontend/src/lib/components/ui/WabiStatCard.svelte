<script lang="ts">
  import { run } from 'svelte/legacy';

  /**
   * Wabi-Sabi Stat Card
   * Beautiful metric display with optional trend indicator
   */
  import { onMount } from 'svelte';
  import { tweened } from 'svelte/motion';
  import { cubicOut } from 'svelte/easing';
  
  interface Props {
    label: string;
    value: number | string;
    unit?: string;
    trend?: 'up' | 'down' | 'neutral' | null;
    trendValue?: string;
    format?: 'number' | 'currency' | 'percent' | 'none';
    decimals?: number;
    animate?: boolean;
  }

  let {
    label,
    value,
    unit = '',
    trend = null,
    trendValue = '',
    format = 'number',
    decimals = 0,
    animate = true
  }: Props = $props();
  
  const animatedValue = tweened(0, { duration: 800, easing: cubicOut });
  
  run(() => {
    if (typeof value === 'number' && animate) {
      animatedValue.set(value);
    }
  });
  
  function formatValue(v: number | string): string {
    if (typeof v === 'string') return v;
    
    switch (format) {
      case 'currency':
        return new Intl.NumberFormat('en-US', { 
          minimumFractionDigits: decimals,
          maximumFractionDigits: decimals 
        }).format(v);
      case 'percent':
        return `${v.toFixed(decimals)}%`;
      case 'number':
        return new Intl.NumberFormat('en-US', {
          minimumFractionDigits: decimals,
          maximumFractionDigits: decimals
        }).format(v);
      default:
        return String(v);
    }
  }
  
  let displayValue = $derived(typeof value === 'number' && animate 
    ? formatValue($animatedValue) 
    : formatValue(value));
</script>

<div class="stat-card">
  <span class="stat-label">{label}</span>
  
  <div class="stat-value-row">
    <span class="stat-value">{displayValue}</span>
    {#if unit}
      <span class="stat-unit">{unit}</span>
    {/if}
  </div>
  
  {#if trend}
    <div class="stat-trend {trend}">
      {#if trend === 'up'}
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="18 15 12 9 6 15"></polyline>
        </svg>
      {:else if trend === 'down'}
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9"></polyline>
        </svg>
      {:else}
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
      {/if}
      {#if trendValue}
        <span>{trendValue}</span>
      {/if}
    </div>
  {/if}
</div>

<style>
  .stat-card {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .stat-label {
    font-family: 'Courier Prime', monospace;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: #57534e;
  }
  
  .stat-value-row {
    display: flex;
    align-items: baseline;
    gap: 4px;
  }
  
  .stat-value {
    font-family: Georgia, serif;
    font-size: 32px;
    color: #1c1c1c;
    line-height: 1;
  }
  
  .stat-unit {
    font-family: 'Courier Prime', monospace;
    font-size: 12px;
    color: #57534e;
  }
  
  .stat-trend {
    display: flex;
    align-items: center;
    gap: 4px;
    font-family: 'Courier Prime', monospace;
    font-size: 11px;
    margin-top: 4px;
  }
  
  .stat-trend svg {
    width: 14px;
    height: 14px;
  }
  
  .stat-trend.up {
    color: #15803d;
  }
  
  .stat-trend.down {
    color: #ef4444;
  }
  
  .stat-trend.neutral {
    color: #57534e;
  }
</style>
