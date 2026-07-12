<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  export interface CurrencyInputProps extends Omit<HTMLInputAttributes, 'value' | 'type'> {
    /** Bound numeric value. Raw number — component handles display formatting. */
    value?: number;
    /** ISO currency code displayed as prefix label (e.g. 'BHD', 'USD', 'EUR'). */
    currency?: string;
    /**
     * Decimal places to format on blur.
     * Defaults to 3 for BHD — one of the few 3-decimal currencies in production.
     */
    decimals?: number;
    /** Numeric lower bound applied on blur. */
    min?: number;
    /** Numeric upper bound applied on blur. */
    max?: number;
    /** When true renders the danger border and sets aria-invalid. */
    invalid?: boolean;
    /** Callback with the parsed numeric value whenever it changes. */
    onValueChange?: (value: number) => void;
  }

  let {
    value = $bindable(0),
    currency = 'BHD',
    decimals = 3,
    min,
    max,
    disabled = false,
    readonly = false,
    invalid = false,
    id,
    class: extraClass,
    onValueChange,
    ...restProps
  }: CurrencyInputProps = $props();

  const generatedId = $props.id();
  const uid = $derived(id ?? generatedId);

  /** While focused, show the raw number the user is editing. */
  let editing = $state(false);

  function fmt(n: number): string {
    if (!isFinite(n)) return '';
    return n.toFixed(decimals);
  }

  function clamp(n: number): number {
    let v = n;
    if (min !== undefined && v < min) v = min;
    if (max !== undefined && v > max) v = max;
    return v;
  }

  function parse(s: string): number {
    const cleaned = s.replace(/[^\d.\-]/g, '');
    const n = parseFloat(cleaned);
    return isNaN(n) ? 0 : n;
  }

  // The string shown inside the input element
  let displayValue = $state(fmt(value));

  // Keep displayValue in sync when value is changed externally (not while editing)
  $effect(() => {
    if (!editing) {
      displayValue = fmt(value);
    }
  });

  function handleFocus(e: FocusEvent) {
    editing = true;
    displayValue = fmt(value);
    // Select-all so user can type straight away
    requestAnimationFrame(() => {
      (e.target as HTMLInputElement).select();
    });
  }

  function handleInput(e: Event) {
    const raw = (e.target as HTMLInputElement).value;
    displayValue = raw;
    const parsed = clamp(parse(raw));
    value = parsed;
    onValueChange?.(parsed);
  }

  function handleBlur() {
    editing = false;
    const clamped = clamp(parse(displayValue));
    value = clamped;
    displayValue = fmt(clamped);
    onValueChange?.(clamped);
  }
</script>

<div
  class="af-currency {extraClass ?? ''}"
  class:af-currency--invalid={invalid}
  class:af-currency--disabled={disabled}
  class:af-currency--readonly={readonly}
  class:af-currency--focused={editing}
>
  <!-- Currency code prefix — always present, always muted, non-interactive -->
  <span class="af-currency__code af-label" aria-hidden="true">{currency}</span>

  <input
    type="text"
    inputmode="decimal"
    id={uid}
    bind:value={displayValue}
    {disabled}
    {readonly}
    aria-invalid={invalid || undefined}
    class="af-currency__input af-numeric"
    oninput={handleInput}
    onfocus={handleFocus}
    onblur={handleBlur}
    {...restProps}
  />
</div>

<style>
  /* ── Wrapper ───────────────────────────────────────────────────────── */
  .af-currency {
    position: relative;
    display: flex;
    align-items: center;
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    transition:
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-currency:hover:not(.af-currency--disabled):not(.af-currency--readonly) {
    border-color: var(--af-border-strong);
  }

  .af-currency--focused {
    border-color: var(--af-accent);
    box-shadow: 0 0 0 3px var(--af-accent-tint);
  }

  .af-currency--invalid {
    border-color: var(--af-danger);
  }

  .af-currency--invalid.af-currency--focused {
    border-color: var(--af-danger);
    box-shadow: 0 0 0 3px var(--af-danger-tint);
  }

  .af-currency--disabled {
    opacity: 0.5;
  }

  .af-currency--readonly {
    background: var(--af-surface-raised);
  }

  /* ── Currency code label (left edge) ──────────────────────────────── */
  .af-currency__code {
    /* Inherits .af-label from base.css: 11px 600 uppercase 0.08em tracking */
    padding: 0 var(--af-space-2) 0 var(--af-space-3);
    color: var(--af-text-muted);
    pointer-events: none;
    user-select: none;
    white-space: nowrap;
    flex-shrink: 0;
    /* Slight vertical lift so the uppercase label sits on the number baseline */
    margin-top: 1px;
  }

  /* ── Input — right-aligned, tabular numerals always (.af-numeric) ── */
  .af-currency__input {
    /* .af-numeric from base.css: Space Grotesk, tabular-nums lining-nums */
    flex: 1 1 0;
    min-width: 0;
    height: var(--af-control-height);
    padding: 0 var(--af-space-3) 0 var(--af-space-1);
    background: transparent;
    border: none;
    outline: none;
    color: var(--af-text);
    font-size: var(--af-text-sm);
    text-align: end; /* right-align numerals, RTL-safe */
  }

  .af-currency__input::placeholder {
    color: var(--af-text-muted);
  }

  /* Touch devices get the full tap target (§2.6); refined desktop sizing stays. */
  @media (pointer: coarse) {
    .af-currency__input {
      height: var(--af-tap-min);
    }
  }

  .af-currency__input:disabled {
    cursor: not-allowed;
  }

  /* Hide browser spinners on number-like inputs */
  .af-currency__input::-webkit-inner-spin-button,
  .af-currency__input::-webkit-outer-spin-button {
    -webkit-appearance: none;
  }
</style>
