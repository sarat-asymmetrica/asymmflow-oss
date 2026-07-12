<script lang="ts">
  import type { HTMLSelectAttributes } from 'svelte/elements';

  export interface SelectOption {
    value: string;
    label: string;
    disabled?: boolean;
  }

  export interface SelectProps extends Omit<HTMLSelectAttributes, 'value'> {
    /** Bound value — mirrors the selected option's value. */
    value?: string;
    /** Array of options. Can also pass children directly as <option> elements. */
    options?: SelectOption[];
    /** Placeholder text rendered as the first disabled option. */
    placeholder?: string;
    /** When true renders the danger border and sets aria-invalid. */
    invalid?: boolean;
    /** Children Snippet — rendered inside <select> (alternative to options prop). */
    children?: import('svelte').Snippet;
  }

  let {
    value = $bindable(''),
    options,
    placeholder,
    disabled = false,
    invalid = false,
    id,
    class: extraClass,
    children,
    ...restProps
  }: SelectProps = $props();

  const generatedId = $props.id();
  const uid = $derived(id ?? generatedId);
</script>

<div
  class="af-select-wrap {extraClass ?? ''}"
  class:af-select-wrap--invalid={invalid}
  class:af-select-wrap--disabled={disabled}
>
  <select
    id={uid}
    bind:value
    {disabled}
    aria-invalid={invalid || undefined}
    class="af-select"
    {...restProps}
  >
    {#if placeholder}
      <option value="" disabled selected={!value}>{placeholder}</option>
    {/if}
    {#if options}
      {#each options as opt}
        <option value={opt.value} disabled={opt.disabled ?? false}>{opt.label}</option>
      {/each}
    {:else}
      {@render children?.()}
    {/if}
  </select>

  <!-- Custom chevron — token-styled, replaces UA arrow -->
  <span class="af-select__chevron" aria-hidden="true">
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M2 4.5L6 8L10 4.5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
    </svg>
  </span>
</div>

<style>
  /* ── Wrapper positions the chevron ─────────────────────────────────── */
  .af-select-wrap {
    position: relative;
    display: flex;
    align-items: center;
    width: 100%;
  }

  /* ── The native select ─────────────────────────────────────────────── */
  .af-select {
    appearance: none;
    -webkit-appearance: none;
    width: 100%;
    height: var(--af-control-height);
    padding: 0 calc(var(--af-space-5) + var(--af-space-1)) 0 var(--af-space-3);
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    color: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    cursor: pointer;
    outline: none;
    transition:
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-select:hover:not(:disabled) {
    border-color: var(--af-border-strong);
  }

  /* Touch devices get the full tap target (§2.6); refined desktop sizing stays. */
  @media (pointer: coarse) {
    .af-select {
      height: var(--af-tap-min);
    }
  }

  .af-select:focus-visible {
    border-color: var(--af-accent);
    box-shadow: 0 0 0 3px var(--af-accent-tint);
    outline: none;
  }

  .af-select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background: var(--af-surface-raised);
  }

  /* ── Invalid state ─────────────────────────────────────────────────── */
  .af-select-wrap--invalid .af-select {
    border-color: var(--af-danger);
  }

  .af-select-wrap--invalid .af-select:focus-visible {
    border-color: var(--af-danger);
    box-shadow: 0 0 0 3px var(--af-danger-tint);
  }

  .af-select-wrap--disabled {
    opacity: 0.5;
  }

  /* ── Chevron (absolute right edge) ─────────────────────────────────── */
  .af-select__chevron {
    position: absolute;
    inset-inline-end: var(--af-space-3);
    pointer-events: none;
    color: var(--af-text-muted);
    display: inline-flex;
    align-items: center;
    transition: color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-select:focus-visible ~ .af-select__chevron {
    color: var(--af-accent);
  }
</style>
