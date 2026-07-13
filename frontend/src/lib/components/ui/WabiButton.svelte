<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import WabiSpinner from './WabiSpinner.svelte';

  interface Props {
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
    size?: 'sm' | 'md' | 'lg';
    disabled?: boolean;
    loading?: boolean;
    fullWidth?: boolean;
    type?: 'button' | 'submit' | 'reset';
    children?: import('svelte').Snippet;
    [key: string]: any
  }

  let {
    variant = 'primary',
    size = 'md',
    disabled = false,
    loading = false,
    fullWidth = false,
    type = 'button',
    children,
    ...rest
  }: Props = $props();

  const dispatch = createEventDispatcher();

  function handleClick(e: MouseEvent) {
    if (!disabled && !loading) {
      dispatch('click', e);
    }
  }
</script>

<button
  {type}
  class="wabi-button {variant} {size}"
  class:full-width={fullWidth}
  class:loading
  {disabled}
  onclick={handleClick}
  {...rest}
>
  {#if loading}
    <span class="spinner-wrapper">
      <WabiSpinner size="sm" color={variant === 'primary' ? '#fdfbf7' : '#1c1c1c'} tempo="alert" />
    </span>
  {/if}
  <span class="button-content" class:hidden={loading}>
    {@render children?.()}
  </span>
</button>

<style>
  .wabi-button {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--fib-2);
    font-family: var(--font-prose, Georgia, serif);
    border-radius: var(--fib-1);
    cursor: pointer;
    transition: transform var(--transition-fast, 120ms cubic-bezier(0.25, 0.1, 0.25, 1)), background var(--transition-fast, 120ms), border-color var(--transition-fast, 120ms);
    white-space: nowrap;
  }

  /* Press state — scale 0.97 per Constitution IV.2, shared by every variant. */
  .wabi-button:active:not(:disabled):not(.loading) {
    transform: scale(0.97);
  }

  /* Sizes */
  .wabi-button.sm {
    padding: 6px var(--fib-2);
    font-size: var(--text-sm, 12px);
  }

  .wabi-button.md {
    padding: 10px var(--fib-3);
    font-size: var(--text-base, 16px);
  }

  .wabi-button.lg {
    padding: var(--fib-2) var(--fib-4);
    font-size: var(--text-lg, 20px);
  }

  /* Variants */
  .wabi-button.primary {
    background: var(--color-ink, #1c1c1c);
    color: var(--color-paper, #fdfbf7);
    border: 1px solid var(--color-ink, #1c1c1c);
  }

  .wabi-button.primary:hover:not(:disabled) {
    background: #2d2d2d;
    border-color: #2d2d2d;
  }

  .wabi-button.primary:active:not(:disabled) {
    background: #000000;
    border-color: #000000;
  }

  .wabi-button.secondary {
    background: transparent;
    color: var(--color-ink, #1c1c1c);
    border: 1px solid var(--color-ink, #1c1c1c);
  }

  .wabi-button.secondary:hover:not(:disabled) {
    background: rgba(0, 0, 0, 0.05);
  }

  .wabi-button.secondary:active:not(:disabled) {
    background: rgba(0, 0, 0, 0.1);
  }

  .wabi-button.ghost {
    background: transparent;
    color: var(--color-ink, #1c1c1c);
    border: 1px solid transparent;
  }

  .wabi-button.ghost:hover:not(:disabled) {
    background: rgba(0, 0, 0, 0.05);
  }

  .wabi-button.ghost:active:not(:disabled) {
    background: rgba(0, 0, 0, 0.1);
  }

  .wabi-button.danger {
    background: var(--color-danger, #ef4444);
    color: white;
    border: 1px solid var(--color-danger, #ef4444);
  }

  .wabi-button.danger:hover:not(:disabled) {
    background: #dc2626;
    border-color: #dc2626;
  }

  .wabi-button.danger:active:not(:disabled) {
    background: #b91c1c;
    border-color: #b91c1c;
  }

  /* States */
  .wabi-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .wabi-button.loading {
    cursor: wait;
  }

  .wabi-button.full-width {
    width: 100%;
  }

  /* Focus ring — points at the single app-wide focus token (Batch 1). */
  .wabi-button:focus-visible {
    outline: 2px solid var(--focus-ring-color, var(--color-ink, #1c1c1c));
    outline-offset: 2px;
  }

  /* Content */
  .button-content {
    display: inline-flex;
    align-items: center;
    gap: var(--fib-2);
  }

  .button-content.hidden {
    visibility: hidden;
  }

  .spinner-wrapper {
    position: absolute;
    display: flex;
    align-items: center;
    justify-content: center;
  }
</style>
