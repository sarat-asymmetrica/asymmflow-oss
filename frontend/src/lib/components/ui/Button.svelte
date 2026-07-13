<!-- @migration-task Error while migrating Svelte code: $$props is used together with named props in a way that cannot be automatically migrated. -->
<script lang="ts">
  export let variant: 'primary' | 'secondary' | 'ghost' | 'danger' | 'success' | 'warning' = 'primary';
  export let size: 'sm' | 'md' | 'lg' = 'md';
  export let type: 'button' | 'submit' | 'reset' = 'button';
  export let disabled: boolean = false;
  export let loading: boolean = false;
  export let fullWidth: boolean = false;

  $: ariaLabel = $$props['aria-label'] || undefined;
</script>

<button
  class="btn btn-{variant} btn-{size}"
  class:w-full={fullWidth}
  class:loading={loading}
  {type}
  disabled={disabled || loading}
  aria-label={ariaLabel}
  aria-disabled={disabled || loading}
  on:click
>
  {#if loading}
    <span class="spinner" aria-hidden="true"></span>
  {/if}
  <slot />
</button>

<style>
  .btn {
    padding: 8px 16px;
    border-radius: var(--border-radius-sm);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: transform var(--transition-fast), background var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), color var(--transition-fast);
    border: none;
    font-family: var(--font-family);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    position: relative;
  }

  /* Press state — defined ONCE, inherited by every variant (Article IV.2: press ≈ scale 0.97). */
  .btn:active:not(:disabled):not(.loading) {
    transform: scale(0.97);
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    pointer-events: none;
  }

  .btn.loading {
    cursor: wait;
  }

  /* Variants */
  .btn-primary {
    background: var(--brand-indigo);
    color: white;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--brand-indigo-hover);
    box-shadow: var(--shadow-indigo);
  }

  .btn-primary:active:not(:disabled) {
    background: var(--brand-indigo-pressed);
  }

  .btn-secondary {
    background: transparent;
    border: var(--border-width) solid var(--border);
    color: var(--text-primary);
  }

  .btn-secondary:hover:not(:disabled) {
    background: var(--surface-elevated);
    border-color: var(--text-muted);
    box-shadow: var(--shadow-sm);
  }

  .btn-secondary:active:not(:disabled) {
    background: var(--brand-indigo-tint);
    border-color: var(--text-muted);
    box-shadow: none;
  }

  .btn-ghost {
    background: transparent;
    color: var(--text-secondary);
  }

  .btn-ghost:hover:not(:disabled) {
    background: var(--brand-indigo-tint);
    color: var(--text-primary);
  }

  .btn-ghost:active:not(:disabled) {
    background: var(--brand-indigo-tint-medium);
    color: var(--text-primary);
  }

  .btn-danger {
    background: #DC2626;
    color: white;
  }

  .btn-danger:hover:not(:disabled) {
    background: #B91C1C;
    box-shadow: 0 4px 12px rgba(220, 38, 38, 0.24);
  }

  .btn-danger:active:not(:disabled) {
    background: #991B1B;
  }

  .btn-success {
    background: #10B981;
    color: white;
  }

  .btn-success:hover:not(:disabled) {
    background: #059669;
    box-shadow: 0 4px 12px rgba(16, 185, 129, 0.24);
  }

  .btn-success:active:not(:disabled) {
    background: #047857;
  }

  .btn-warning {
    background: #F59E0B;
    color: white;
  }

  .btn-warning:hover:not(:disabled) {
    background: #D97706;
    box-shadow: 0 4px 12px rgba(245, 158, 11, 0.24);
  }

  .btn-warning:active:not(:disabled) {
    background: #B45309;
  }

  /* Sizes */
  .btn-sm {
    padding: 6px 12px;
    font-size: 13px;
    gap: 6px;
  }

  .btn-md {
    padding: 8px 16px;
    font-size: 14px;
    gap: 8px;
  }

  .btn-lg {
    padding: 10px 20px;
    font-size: 15px;
    gap: 10px;
  }

  .w-full {
    width: 100%;
  }

  /* Loading spinner */
  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid currentColor;
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  /* Focus visible for accessibility */
  .btn:focus-visible {
    outline: 2px solid var(--focus-ring-color);
    outline-offset: 2px;
  }
</style>
