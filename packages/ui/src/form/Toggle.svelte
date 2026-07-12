<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  export interface ToggleProps extends Omit<HTMLInputAttributes, 'checked' | 'type'> {
    /** Bound on/off state. */
    checked?: boolean;
    /** Visible label text. */
    label?: string;
    /** Secondary descriptive line rendered below the label. */
    description?: string;
    /**
     * 'end' (default): label is to the right of the switch.
     * 'start': label is to the left (useful in settings lists).
     */
    labelPosition?: 'start' | 'end';
    /** Callback fired after the checked state changes. */
    onCheckedChange?: (checked: boolean) => void;
  }

  let {
    checked = $bindable(false),
    label,
    description,
    labelPosition = 'end',
    disabled = false,
    id,
    name,
    class: extraClass,
    onCheckedChange,
    ...restProps
  }: ToggleProps = $props();

  const generatedId = $props.id();
  const uid = $derived(id ?? generatedId);

  function handleChange(e: Event) {
    const target = e.target as HTMLInputElement;
    checked = target.checked;
    onCheckedChange?.(checked);
  }
</script>

<label
  class="af-toggle {extraClass ?? ''}"
  class:af-toggle--disabled={disabled}
  class:af-toggle--label-start={labelPosition === 'start'}
  for={uid}
>
  {#if (label || description) && labelPosition === 'start'}
    <span class="af-toggle__content">
      {#if label}<span class="af-toggle__label">{label}</span>{/if}
      {#if description}<span class="af-toggle__desc">{description}</span>{/if}
    </span>
  {/if}

  <span class="af-toggle__control">
    <input
      type="checkbox"
      role="switch"
      id={uid}
      {name}
      bind:checked
      {disabled}
      aria-checked={checked}
      class="af-toggle__input"
      onchange={handleChange}
      {...restProps}
    />
    <span class="af-toggle__track" aria-hidden="true">
      <span class="af-toggle__thumb"></span>
    </span>
  </span>

  {#if (label || description) && labelPosition === 'end'}
    <span class="af-toggle__content">
      {#if label}<span class="af-toggle__label">{label}</span>{/if}
      {#if description}<span class="af-toggle__desc">{description}</span>{/if}
    </span>
  {/if}
</label>

<style>
  /* ── Wrapper ───────────────────────────────────────────────────────── */
  .af-toggle {
    display: inline-flex;
    align-items: flex-start;
    gap: var(--af-space-3);
    cursor: pointer;
    user-select: none;
    min-height: var(--af-tap-min); /* a11y touch target */
  }

  .af-toggle--label-start {
    justify-content: space-between;
    width: 100%;
  }

  .af-toggle--disabled {
    opacity: 0.5;
    cursor: not-allowed;
    pointer-events: none;
  }

  /* ── Hidden real checkbox ──────────────────────────────────────────── */
  .af-toggle__input {
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

  /* ── Switch chrome ─────────────────────────────────────────────────── */
  .af-toggle__control {
    position: relative;
    flex-shrink: 0;
    margin-top: 2px; /* optical align with first text baseline */
  }

  .af-toggle__track {
    display: block;
    width: 44px;
    height: 24px;
    border-radius: var(--af-radius-pill);
    background: var(--af-border-strong);
    transition: background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-toggle__thumb {
    position: absolute;
    top: 3px;
    inset-inline-start: 3px;
    width: 18px;
    height: 18px;
    border-radius: var(--af-radius-pill);
    background: var(--af-surface);
    box-shadow: var(--af-shadow-sm);
    transition: transform var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  /* Checked state — fill track with accent, slide thumb */
  .af-toggle__input:checked ~ .af-toggle__track {
    background: var(--af-accent);
  }

  .af-toggle__input:checked ~ .af-toggle__track .af-toggle__thumb {
    transform: translateX(20px);
  }

  /* Hover — darken unchecked track slightly */
  .af-toggle:hover:not(.af-toggle--disabled) .af-toggle__input:not(:checked) ~ .af-toggle__track {
    background: var(--af-text-muted);
  }

  /* Focus ring */
  .af-toggle__input:focus-visible ~ .af-toggle__track {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  /* ── Text content ──────────────────────────────────────────────────── */
  .af-toggle__content {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
    padding-top: 2px;
  }

  .af-toggle__label {
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    color: var(--af-text);
    line-height: var(--af-leading-tight);
  }

  .af-toggle__desc {
    font-size: var(--af-text-xs);
    color: var(--af-text-secondary);
    line-height: var(--af-leading-base);
  }

  /* ── Reduced motion ────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .af-toggle__thumb {
      transition: none;
    }
  }
</style>
