<script lang="ts">
  import { run } from 'svelte/legacy';

  /**
   * Enterprise Currency Input Component
   * Philosophy: Right-aligned numbers with proper decimal handling
   * Follows: Bloomberg-level precision for financial data
   */
  import { createEventDispatcher } from 'svelte';

  interface Props {
    value?: number;
    currency?: string;
    label?: string;
    error?: string;
    disabled?: boolean;
    readonly?: boolean;
    required?: boolean;
    decimals?: number; // BHD uses 3 decimals!
    min?: number | undefined;
    max?: number | undefined;
    showCurrency?: boolean;
    id?: string;
    name?: string;
    [key: string]: any
  }

  let {
    value = $bindable(0),
    currency = 'BHD',
    label = '',
    error = '',
    disabled = false,
    readonly = false,
    required = false,
    decimals = 3,
    min = undefined,
    max = undefined,
    showCurrency = true,
    id = `currency-${Math.random().toString(36).substr(2, 9)}`,
    name = '',
    ...rest
  }: Props = $props();

  const dispatch = createEventDispatcher();

  let displayValue: string = $state(formatNumber(value));
  let isFocused = $state(false);

  const currencySymbols: Record<string, string> = {
    BHD: 'BHD',
    USD: '$',
    EUR: '€',
    GBP: '£',
    SAR: 'SAR',
    AED: 'AED',
    KWD: 'KWD',
  };

  let currencySymbol = $derived(currencySymbols[currency] || currency);

  function formatNumber(num: number): string {
    if (isNaN(num) || num === null || num === undefined) return '';
    return num.toFixed(decimals);
  }

  function parseNumber(str: string): number {
    // Remove all non-numeric characters except decimal point and minus
    const cleaned = str.replace(/[^\d.-]/g, '');
    const parsed = parseFloat(cleaned);
    return isNaN(parsed) ? 0 : parsed;
  }

  function handleInput(e: Event) {
    const target = e.target as HTMLInputElement;
    const parsed = parseNumber(target.value);

    // Apply min/max constraints
    let constrained = parsed;
    if (min !== undefined && constrained < min) constrained = min;
    if (max !== undefined && constrained > max) constrained = max;

    value = constrained;
    dispatch('input', { value, event: e });
  }

  function handleFocus(e: FocusEvent) {
    isFocused = true;
    const target = e.target as HTMLInputElement;
    // Show raw number when focused
    displayValue = formatNumber(value);
    dispatch('focus', e);
    // Select all for easy replacement
    setTimeout(() => target.select(), 0);
  }

  function handleBlur(e: FocusEvent) {
    isFocused = false;
    const target = e.target as HTMLInputElement;

    // Parse and format on blur
    const parsed = parseNumber(target.value);
    let constrained = parsed;
    if (min !== undefined && constrained < min) constrained = min;
    if (max !== undefined && constrained > max) constrained = max;

    value = constrained;
    displayValue = formatNumber(value);

    dispatch('blur', { value, event: e });
    dispatch('change', { value, event: e });
  }

  function handleKeyDown(e: KeyboardEvent) {
    // Allow: backspace, delete, tab, escape, enter
    if ([8, 9, 27, 13, 46].indexOf(e.keyCode) !== -1 ||
      // Allow: Ctrl+A, Ctrl+C, Ctrl+V, Ctrl+X
      (e.keyCode === 65 && e.ctrlKey === true) ||
      (e.keyCode === 67 && e.ctrlKey === true) ||
      (e.keyCode === 86 && e.ctrlKey === true) ||
      (e.keyCode === 88 && e.ctrlKey === true) ||
      // Allow: home, end, left, right
      (e.keyCode >= 35 && e.keyCode <= 39)) {
      return;
    }

    // Ensure it's a number or decimal point or minus
    if ((e.shiftKey || (e.keyCode < 48 || e.keyCode > 57)) &&
      (e.keyCode < 96 || e.keyCode > 105) &&
      e.keyCode !== 190 && // period
      e.keyCode !== 110 && // decimal point (numpad)
      e.keyCode !== 189 && // minus
      e.keyCode !== 109) { // minus (numpad)
      e.preventDefault();
    }
  }

  run(() => {
    if (!isFocused) {
      displayValue = formatNumber(value);
    }
  });
</script>

<div class="currency-wrapper" class:has-error={!!error} class:disabled>
  {#if label}
    <label for={id} class="label">
      {label}
      {#if required}<span class="required" aria-label="required">*</span>{/if}
    </label>
  {/if}

  <div class="input-container">
    {#if showCurrency}
      <span class="currency-symbol">{currencySymbol}</span>
    {/if}
    <input
      {id}
      {name}
      type="text"
      inputmode="decimal"
      {disabled}
      {readonly}
      {required}
      bind:value={displayValue}
      class="currency-input"
      class:with-symbol={showCurrency}
      oninput={handleInput}
      onfocus={handleFocus}
      onblur={handleBlur}
      onkeydown={handleKeyDown}
      aria-invalid={!!error}
      aria-describedby={error ? `${id}-error` : undefined}
      {...rest}
    />
  </div>

  {#if error}
    <span id="{id}-error" class="error-message" role="alert">
      {error}
    </span>
  {/if}
</div>

<style>
  .currency-wrapper {
    display: flex;
    flex-direction: column;
    gap: 6px;
    width: 100%;
  }

  .label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    line-height: var(--line-height-tight);
  }

  .required {
    color: #DC2626;
    margin-left: 2px;
  }

  .input-container {
    position: relative;
    display: flex;
    align-items: center;
  }

  .currency-symbol {
    position: absolute;
    left: 12px;
    font-size: 14px;
    font-weight: 500;
    color: var(--text-secondary);
    pointer-events: none;
    user-select: none;
  }

  .currency-input {
    width: 100%;
    padding: 8px 12px;
    font-size: 14px;
    font-family: var(--font-family);
    font-variant-numeric: tabular-nums;
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
    text-align: right;
  }

  .currency-input.with-symbol {
    padding-left: 52px; /* Space for currency symbol */
  }

  .currency-input:hover:not(:disabled):not(:readonly) {
    border-color: var(--text-muted);
  }

  .currency-input:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 3px var(--brand-indigo-tint);
  }

  .currency-input:disabled,
  .currency-input:readonly {
    opacity: 0.5;
    cursor: not-allowed;
    background: var(--surface-elevated);
  }

  .has-error .currency-input {
    border-color: #DC2626;
  }

  .has-error .currency-input:focus {
    border-color: #DC2626;
    box-shadow: 0 0 0 3px rgba(220, 38, 38, 0.1);
  }

  .error-message {
    font-size: 12px;
    color: #DC2626;
    line-height: var(--line-height-tight);
  }

  .disabled {
    opacity: 0.6;
  }
</style>
