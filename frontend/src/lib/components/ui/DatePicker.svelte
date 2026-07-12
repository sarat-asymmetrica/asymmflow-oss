<script lang="ts">
  /**
   * Enterprise Date Picker Component
   * Philosophy: Native date input with consistent styling
   * Follows: Apple-level polish with browser-native functionality
   */
  import { createEventDispatcher } from 'svelte';

  interface Props {
    value?: string; // ISO date string (YYYY-MM-DD)
    label?: string;
    error?: string;
    disabled?: boolean;
    readonly?: boolean;
    required?: boolean;
    min?: string;
    max?: string;
    id?: string;
    name?: string;
    [key: string]: any
  }

  let {
    value = $bindable(''),
    label = '',
    error = '',
    disabled = false,
    readonly = false,
    required = false,
    min = '',
    max = '',
    id = `date-${Math.random().toString(36).substr(2, 9)}`,
    name = '',
    ...rest
  }: Props = $props();

  const dispatch = createEventDispatcher();

  function handleInput(e: Event) {
    const target = e.target as HTMLInputElement;
    value = target.value;
    dispatch('input', { value, event: e });
  }

  function handleChange(e: Event) {
    dispatch('change', { value, event: e });
  }

  function handleFocus(e: FocusEvent) {
    dispatch('focus', e);
  }

  function handleBlur(e: FocusEvent) {
    dispatch('blur', e);
  }

  // Format date for display (optional, native input handles this)
  function formatDateForDisplay(isoDate: string): string {
    if (!isoDate) return '';
    try {
      const date = new Date(isoDate);
      return new Intl.DateTimeFormat('en-GB', {
        day: '2-digit',
        month: 'short',
        year: 'numeric'
      }).format(date);
    } catch {
      return isoDate;
    }
  }
</script>

<div class="date-wrapper" class:has-error={!!error} class:disabled>
  {#if label}
    <label for={id} class="label">
      {label}
      {#if required}<span class="required" aria-label="required">*</span>{/if}
    </label>
  {/if}

  <div class="input-container">
    <input
      {id}
      {name}
      type="date"
      {disabled}
      {readonly}
      {required}
      {min}
      {max}
      bind:value
      class="date-input"
      oninput={handleInput}
      onchange={handleChange}
      onfocus={handleFocus}
      onblur={handleBlur}
      aria-invalid={!!error}
      aria-describedby={error ? `${id}-error` : undefined}
      {...rest}
    />
    <svg class="calendar-icon" width="16" height="16" viewBox="0 0 16 16" fill="none">
      <path d="M12 2H13C13.5304 2 14.0391 2.21071 14.4142 2.58579C14.7893 2.96086 15 3.46957 15 4V13C15 13.5304 14.7893 14.0391 14.4142 14.4142C14.0391 14.7893 13.5304 15 13 15H3C2.46957 15 1.96086 14.7893 1.58579 14.4142C1.21071 14.0391 1 13.5304 1 13V4C1 3.46957 1.21071 2.96086 1.58579 2.58579C1.96086 2.21071 2.46957 2 3 2H4M5 1V3M11 1V3M1 6H15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
    </svg>
  </div>

  {#if error}
    <span id="{id}-error" class="error-message" role="alert">
      {error}
    </span>
  {/if}
</div>

<style>
  .date-wrapper {
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

  .date-input {
    width: 100%;
    padding: 8px 12px;
    padding-right: 36px; /* Space for calendar icon */
    font-size: 14px;
    font-family: var(--font-family);
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
    line-height: var(--line-height-base);
  }

  /* Hide default calendar icon in Chrome/Safari */
  .date-input::-webkit-calendar-picker-indicator {
    opacity: 0;
    position: absolute;
    right: 0;
    width: 100%;
    height: 100%;
    cursor: pointer;
  }

  .calendar-icon {
    position: absolute;
    right: 12px;
    color: var(--text-muted);
    pointer-events: none;
  }

  .date-input:hover:not(:disabled):not(:readonly) {
    border-color: var(--text-muted);
  }

  .date-input:hover:not(:disabled):not(:readonly) + .calendar-icon {
    color: var(--text-primary);
  }

  .date-input:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 3px var(--brand-indigo-tint);
  }

  .date-input:focus + .calendar-icon {
    color: var(--brand-indigo);
  }

  .date-input:disabled,
  .date-input:readonly {
    opacity: 0.5;
    cursor: not-allowed;
    background: var(--surface-elevated);
  }

  .has-error .date-input {
    border-color: #DC2626;
  }

  .has-error .date-input:focus {
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
