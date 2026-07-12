<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';
  import type { Snippet } from 'svelte';

  export interface InputProps extends Omit<HTMLInputAttributes, 'value' | 'prefix'> {
    /** Input type — kept to the text-family subset (no number; use CurrencyInput). */
    type?: 'text' | 'email' | 'password' | 'tel' | 'search' | 'url' | 'date';
    /** Bound value. */
    value?: string;
    /** Renders inside the control start edge (currency symbol, icon, etc.). */
    prefix?: Snippet;
    /** Renders inside the control end edge (unit label, clear icon, etc.). */
    suffix?: Snippet;
    /** When true renders the danger border and sets aria-invalid. */
    invalid?: boolean;
  }

  let {
    type = 'text',
    value = $bindable(''),
    prefix,
    suffix,
    disabled = false,
    readonly = false,
    invalid = false,
    id,
    class: extraClass,
    ...restProps
  }: InputProps = $props();

  const generatedId = $props.id();
  const uid = $derived(id ?? generatedId);
</script>

<div
  class="af-input-wrap {extraClass ?? ''}"
  class:af-input-wrap--invalid={invalid}
  class:af-input-wrap--disabled={disabled}
  class:af-input-wrap--readonly={readonly}
  class:af-input-wrap--has-prefix={!!prefix}
  class:af-input-wrap--has-suffix={!!suffix}
>
  {#if prefix}
    <span class="af-input__adorn af-input__adorn--prefix" aria-hidden="true">
      {@render prefix()}
    </span>
  {/if}

  <input
    {type}
    id={uid}
    bind:value
    {disabled}
    {readonly}
    aria-invalid={invalid || undefined}
    class="af-input"
    {...restProps}
  />

  {#if suffix}
    <span class="af-input__adorn af-input__adorn--suffix" aria-hidden="true">
      {@render suffix()}
    </span>
  {/if}
</div>

<style>
  /* ── Wrapper ───────────────────────────────────────────────────────── */
  .af-input-wrap {
    position: relative;
    display: flex;
    align-items: center;
    width: 100%;
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    transition:
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-input-wrap:has(.af-input:hover):not(.af-input-wrap--disabled):not(.af-input-wrap--readonly) {
    border-color: var(--af-border-strong);
  }

  .af-input-wrap:has(.af-input:focus-visible) {
    border-color: var(--af-accent);
    box-shadow: 0 0 0 3px var(--af-accent-tint);
    outline: none;
  }

  .af-input-wrap--invalid {
    border-color: var(--af-danger);
  }

  .af-input-wrap--invalid:has(.af-input:focus-visible) {
    border-color: var(--af-danger);
    box-shadow: 0 0 0 3px var(--af-danger-tint);
  }

  .af-input-wrap--disabled {
    opacity: 0.5;
  }

  .af-input-wrap--readonly {
    background: var(--af-surface-raised);
    border-color: var(--af-border);
  }

  /* ── The actual input — no own chrome ─────────────────────────────── */
  .af-input {
    flex: 1 1 0;
    min-width: 0;
    height: var(--af-control-height);
    padding: 0 var(--af-space-3);
    background: transparent;
    border: none;
    color: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    line-height: var(--af-leading-base);
    /* Suppress the browser's own focus ring — the wrapper draws ours */
    outline: none;
  }

  .af-input::placeholder {
    color: var(--af-text-muted);
  }

  /* Touch devices get the full tap target (§2.6); refined desktop sizing stays. */
  @media (pointer: coarse) {
    .af-input {
      height: var(--af-tap-min);
    }
  }

  .af-input:disabled {
    cursor: not-allowed;
  }

  .af-input:read-only {
    cursor: default;
  }

  /* Compress padding when adorns are present */
  .af-input-wrap--has-prefix .af-input {
    padding-inline-start: var(--af-space-2);
  }

  .af-input-wrap--has-suffix .af-input {
    padding-inline-end: var(--af-space-2);
  }

  /* ── Adornments ────────────────────────────────────────────────────── */
  .af-input__adorn {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    color: var(--af-text-muted);
    font-size: var(--af-text-sm);
    pointer-events: none;
    user-select: none;
  }

  .af-input__adorn--prefix {
    padding-inline-start: var(--af-space-3);
  }

  .af-input__adorn--suffix {
    padding-inline-end: var(--af-space-3);
  }

  /* ── Spinners hidden for numeric inputs ────────────────────────────── */
  .af-input[type='number']::-webkit-inner-spin-button,
  .af-input[type='number']::-webkit-outer-spin-button {
    -webkit-appearance: none;
  }

  .af-input[type='number'] {
    -moz-appearance: textfield;
    appearance: textfield;
  }
</style>
