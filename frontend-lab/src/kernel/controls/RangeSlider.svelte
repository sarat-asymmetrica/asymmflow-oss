<script lang="ts">
  /* Range slider — a styled <input type="range"> wrapper. First user is the
   * Pricing simulator's target-margin drag; kept generic (no domain
   * knowledge) so any future numeric-range control reuses it. */
  let {
    value = $bindable(0),
    min = 0,
    max = 100,
    step = 1,
    label,
    formatValue,
  }: {
    value?: number
    min?: number
    max?: number
    step?: number
    label?: string
    /** Formats the live value shown beside the label (defaults to the raw number). */
    formatValue?: (v: number) => string
  } = $props()

  const display = $derived(formatValue ? formatValue(value) : String(value))
</script>

<div class="k-range">
  {#if label}
    <div class="k-range-head">
      <span class="k-range-label">{label}</span>
      <span class="k-range-value">{display}</span>
    </div>
  {/if}
  <input class="k-range-input" type="range" {min} {max} {step} bind:value />
</div>

<style>
  .k-range {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-xs);
    min-width: 0;
  }
  .k-range-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-range-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }
  .k-range-value {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-weight: 700;
    color: var(--text-primary);
    flex-shrink: 0;
  }
  .k-range-input {
    -webkit-appearance: none;
    appearance: none;
    width: 100%;
    height: 4px;
    border-radius: var(--border-radius-pill);
    background: var(--onyx-tint);
    outline: none;
    cursor: pointer;
  }
  .k-range-input::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: var(--brand-indigo);
    border: 2px solid var(--surface);
    box-shadow: var(--shadow-sm);
    cursor: pointer;
  }
  .k-range-input::-moz-range-thumb {
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: var(--brand-indigo);
    border: 2px solid var(--surface);
    box-shadow: var(--shadow-sm);
    cursor: pointer;
  }
</style>
