<script lang="ts">
  /**
   * Enterprise Textarea Component
   * Philosophy: Clean, accessible, with optional auto-resize
   * Follows: Apple-level polish × Bloomberg-level data density
   */
  import { createEventDispatcher, onMount } from 'svelte';

  interface Props {
    value?: string;
    placeholder?: string;
    label?: string;
    rows?: number;
    maxLength?: number | undefined;
    error?: string;
    disabled?: boolean;
    readonly?: boolean;
    required?: boolean;
    autoResize?: boolean;
    id?: string;
    name?: string;
    [key: string]: any
  }

  let {
    value = $bindable(''),
    placeholder = '',
    label = '',
    rows = 3,
    maxLength = undefined,
    error = '',
    disabled = false,
    readonly = false,
    required = false,
    autoResize = false,
    id = `textarea-${Math.random().toString(36).substr(2, 9)}`,
    name = '',
    ...rest
  }: Props = $props();

  const dispatch = createEventDispatcher();

  let textareaElement: HTMLTextAreaElement = $state();

  let characterCount = $derived(value.length);
  let remainingChars = $derived(maxLength ? maxLength - characterCount : null);

  function handleInput(e: Event) {
    const target = e.target as HTMLTextAreaElement;
    value = target.value;
    dispatch('input', { value, event: e });

    if (autoResize) {
      resizeTextarea();
    }
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

  function resizeTextarea() {
    if (!textareaElement || !autoResize) return;
    textareaElement.style.height = 'auto';
    textareaElement.style.height = `${textareaElement.scrollHeight}px`;
  }

  onMount(() => {
    if (autoResize && value) {
      resizeTextarea();
    }
  });
</script>

<div class="textarea-wrapper" class:has-error={!!error} class:disabled>
  {#if label}
    <div class="label-row">
      <label for={id} class="label">
        {label}
        {#if required}<span class="required" aria-label="required">*</span>{/if}
      </label>
      {#if maxLength}
        <span class="char-count" class:warning={remainingChars !== null && remainingChars < 20}>
          {characterCount}{maxLength ? `/${maxLength}` : ''}
        </span>
      {/if}
    </div>
  {/if}

  <textarea
    bind:this={textareaElement}
    {id}
    {name}
    {placeholder}
    {disabled}
    {readonly}
    {required}
    {rows}
    maxlength={maxLength}
    bind:value
    class="textarea"
    class:auto-resize={autoResize}
    oninput={handleInput}
    onchange={handleChange}
    onfocus={handleFocus}
    onblur={handleBlur}
    aria-invalid={!!error}
    aria-describedby={error ? `${id}-error` : undefined}
    {...rest}
></textarea>

  {#if error}
    <span id="{id}-error" class="error-message" role="alert">
      {error}
    </span>
  {/if}
</div>

<style>
  .textarea-wrapper {
    display: flex;
    flex-direction: column;
    gap: 6px;
    width: 100%;
  }

  .label-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
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

  .char-count {
    font-size: var(--meta-size);
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
  }

  .char-count.warning {
    color: #F59E0B;
    font-weight: 500;
  }

  .textarea {
    width: 100%;
    padding: 8px 12px;
    font-size: 14px;
    font-family: var(--font-family);
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
    line-height: var(--line-height-base);
    resize: vertical;
    min-height: 44px;
  }

  .textarea.auto-resize {
    resize: none;
    overflow: hidden;
  }

  .textarea::placeholder {
    color: var(--text-muted);
  }

  .textarea:hover:not(:disabled):not(:readonly) {
    border-color: var(--text-muted);
  }

  .textarea:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 3px var(--brand-indigo-tint);
  }

  .textarea:disabled,
  .textarea:readonly {
    opacity: 0.5;
    cursor: not-allowed;
    background: var(--surface-elevated);
    resize: none;
  }

  .has-error .textarea {
    border-color: #DC2626;
  }

  .has-error .textarea:focus {
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
