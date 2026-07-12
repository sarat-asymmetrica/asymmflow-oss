<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  export interface CheckboxProps extends Omit<HTMLInputAttributes, 'checked' | 'type' | 'value'> {
    /** Bound checked state. */
    checked?: boolean;
    /**
     * When true renders the dash mark (mixed selection).
     * Also sets aria-checked="mixed" and overrides the visual tick.
     */
    indeterminate?: boolean;
    /** Visible label text rendered beside the custom box. */
    label?: string;
    /** Callback fired after the checked state changes. */
    onCheckedChange?: (checked: boolean) => void;
  }

  let {
    checked = $bindable(false),
    indeterminate = false,
    label,
    disabled = false,
    id,
    class: extraClass,
    onCheckedChange,
    ...restProps
  }: CheckboxProps = $props();

  const generatedId = $props.id();
  const uid = $derived(id ?? generatedId);

  let inputEl: HTMLInputElement | undefined = $state();

  $effect(() => {
    if (inputEl) {
      inputEl.indeterminate = indeterminate;
    }
  });

  function handleChange(e: Event) {
    const target = e.target as HTMLInputElement;
    checked = target.checked;
    onCheckedChange?.(checked);
  }
</script>

<label class="af-checkbox {extraClass ?? ''}" class:af-checkbox--disabled={disabled} for={uid}>
  <span class="af-checkbox__control">
    <input
      bind:this={inputEl}
      type="checkbox"
      id={uid}
      bind:checked
      {disabled}
      aria-checked={indeterminate ? 'mixed' : checked}
      class="af-checkbox__input"
      onchange={handleChange}
      {...restProps}
    />
    <span class="af-checkbox__box" aria-hidden="true">
      {#if indeterminate}
        <!-- Dash for indeterminate -->
        <svg class="af-checkbox__mark" viewBox="0 0 14 14" fill="none">
          <path d="M3 7h8" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        </svg>
      {:else if checked}
        <!-- Tick for checked -->
        <svg class="af-checkbox__mark" viewBox="0 0 14 14" fill="none">
          <path d="M2.5 7L5.5 10L11.5 4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      {/if}
    </span>
  </span>

  {#if label}
    <span class="af-checkbox__label">{label}</span>
  {/if}
</label>

<style>
  /* ── Layout ────────────────────────────────────────────────────────── */
  .af-checkbox {
    display: inline-flex;
    align-items: center;
    gap: var(--af-space-2);
    cursor: pointer;
    user-select: none;
    min-height: var(--af-tap-min); /* a11y touch target floor §2.6 */
  }

  .af-checkbox--disabled {
    opacity: 0.5;
    cursor: not-allowed;
    pointer-events: none;
  }

  /* ── Visually-hidden real input (keyboard & screen-reader) ─────────── */
  .af-checkbox__input {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
    border: 0;
  }

  /* ── Custom drawn box ──────────────────────────────────────────────── */
  .af-checkbox__control {
    position: relative;
    flex-shrink: 0;
  }

  .af-checkbox__box {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    border: 1.5px solid var(--af-border-strong);
    border-radius: calc(var(--af-radius-sm) / 2);
    background: var(--af-surface);
    color: var(--af-accent-contrast);
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  /* Checked / indeterminate — filled with accent */
  .af-checkbox__input:checked ~ .af-checkbox__box,
  .af-checkbox__input:indeterminate ~ .af-checkbox__box {
    background: var(--af-accent);
    border-color: var(--af-accent);
  }

  /* Hover when unchecked */
  .af-checkbox:hover:not(.af-checkbox--disabled) .af-checkbox__input:not(:checked):not(:indeterminate) ~ .af-checkbox__box {
    border-color: var(--af-accent);
  }

  /* Focus ring on box */
  .af-checkbox__input:focus-visible ~ .af-checkbox__box {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 2px;
    box-shadow: none;
  }

  /* ── Mark (tick / dash) ────────────────────────────────────────────── */
  .af-checkbox__mark {
    width: 10px;
    height: 10px;
    flex-shrink: 0;
  }

  /* ── Label ─────────────────────────────────────────────────────────── */
  .af-checkbox__label {
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    color: var(--af-text);
    line-height: var(--af-leading-tight);
  }
</style>
