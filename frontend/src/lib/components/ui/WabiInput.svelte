<script lang="ts">
  /**
   * Wabi-Sabi Input Component
   * Clean, paper-like input with ink focus
   */
  import { createEventDispatcher } from 'svelte';

  interface Props {
    value?: string;
    type?: 'text' | 'password' | 'email' | 'number' | 'search' | 'tel' | 'url';
    placeholder?: string;
    label?: string;
    hint?: string;
    error?: string;
    disabled?: boolean;
    required?: boolean;
    id?: string;
    [key: string]: any
  }

  let {
    value = $bindable(''),
    type = 'text',
    placeholder = '',
    label = '',
    hint = '',
    error = '',
    disabled = false,
    required = false,
    id = `input-${Math.random().toString(36).substr(2, 9)}`,
    ...rest
  }: Props = $props();

  const dispatch = createEventDispatcher();

  function handleInput(e: Event) {
    const target = e.target as HTMLInputElement;
    value = target.value;
    dispatch('input', value);
  }

  function handleChange(e: Event) {
    dispatch('change', value);
  }

  function handleFocus(e: FocusEvent) {
    dispatch('focus', e);
  }

  function handleBlur(e: FocusEvent) {
    dispatch('blur', e);
  }
</script>

<div class="wabi-input-wrapper" class:has-error={!!error} class:disabled>
  {#if label}
    <label for={id} class="input-label">
      {label}
      {#if required}<span class="required">*</span>{/if}
    </label>
  {/if}

  <input
    {id}
    {type}
    {placeholder}
    {disabled}
    {required}
    {value}
    class="wabi-input"
    oninput={handleInput}
    onchange={handleChange}
    onfocus={handleFocus}
    onblur={handleBlur}
    {...rest}
  />

  {#if hint && !error}
    <span class="input-hint">{hint}</span>
  {/if}

  {#if error}
    <span class="input-error">{error}</span>
  {/if}
</div>

<style>
  .wabi-input-wrapper {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .input-label {
    font-family: var(--font-prose, Georgia, serif);
    font-size: var(--text-sm, 12px);
    color: var(--color-ink, #1c1c1c);
  }

  .required {
    color: var(--color-danger, #ef4444);
    margin-left: 2px;
  }

  .wabi-input {
    padding: var(--space-1, 13px) var(--space-2, 21px);
    background: rgba(255, 255, 255, 0.6);
    border: 1px solid rgba(0, 0, 0, 0.1);
    border-radius: var(--space-0, 8px);
    font-family: var(--font-prose, Georgia, serif);
    font-size: var(--text-base, 16px);
    color: var(--color-ink, #1c1c1c);
    transition: all 0.2s var(--ease-wabi);
    width: 100%;
    box-sizing: border-box;
  }

  .wabi-input::placeholder {
    color: var(--color-ink-light, #57534e);
    opacity: 0.6;
  }

  .wabi-input:hover:not(:disabled) {
    border-color: rgba(0, 0, 0, 0.2);
  }

  .wabi-input:focus {
    outline: none;
    border-color: var(--color-ink, #1c1c1c);
    background: rgba(255, 255, 255, 0.8);
  }

  .wabi-input:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background: rgba(0, 0, 0, 0.03);
  }

  .has-error .wabi-input {
    border-color: var(--color-danger, #ef4444);
  }

  .has-error .wabi-input:focus {
    border-color: var(--color-danger, #ef4444);
  }

  .input-hint {
    font-family: var(--font-data, monospace);
    font-size: 11px;
    color: var(--color-ink-light, #57534e);
  }

  .input-error {
    font-family: var(--font-data, monospace);
    font-size: 11px;
    color: var(--color-danger, #ef4444);
  }

  .disabled {
    opacity: 0.6;
  }
</style>
