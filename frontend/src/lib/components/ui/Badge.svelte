<script lang="ts">
  

  interface Props {
    /**
   * Enterprise Badge Component
   * Follows design system: Subtle, pill-shaped status indicators
   *
   * Features:
   * - 5 semantic variants (default, success, warning, danger, info)
   * - 2 sizes (sm, md)
   * - Automatic color mapping
   * - Fully accessible
   */
    variant?: 'default' | 'success' | 'warning' | 'danger' | 'info';
    size?: 'sm' | 'md';
    children?: import('svelte').Snippet;
  }

  let { variant = 'default', size = 'md', children }: Props = $props();
</script>

<span
  class="badge badge-{variant} badge-{size}"
  role="status"
  aria-label={children ? undefined : variant}
>
  {@render children?.()}
</span>

<style>
  .badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-weight: 500;
    border-radius: 12px;
    white-space: nowrap;
    transition: all var(--transition-fast);
    line-height: 1;
  }

  /* Sizes */
  .badge-sm {
    padding: 3px 8px;
    font-size: 10px;
    min-height: 18px;
  }

  .badge-md {
    padding: 4px 10px;
    font-size: 12px;
    min-height: 22px;
  }

  /* Variants - Using design system colors */
  .badge-default {
    background: var(--surface-elevated);
    color: var(--text-secondary);
    border: 1px solid var(--border);
  }

  .badge-success {
    background: rgba(34, 197, 94, 0.1);
    color: #16a34a;
    border: 1px solid rgba(34, 197, 94, 0.2);
  }

  .badge-warning {
    background: rgba(251, 191, 36, 0.1);
    color: #d97706;
    border: 1px solid rgba(251, 191, 36, 0.2);
  }

  .badge-danger {
    background: rgba(239, 68, 68, 0.1);
    color: #dc2626;
    border: 1px solid rgba(239, 68, 68, 0.2);
  }

  .badge-info {
    background: var(--indigo-contrast-surface);
    color: var(--brand-indigo);
    border: 1px solid rgba(47, 45, 255, 0.2);
  }

  /* Dark mode adjustments */
  :global([data-theme="dark"]) .badge-success {
    background: rgba(34, 197, 94, 0.15);
    color: #22c55e;
  }

  :global([data-theme="dark"]) .badge-warning {
    background: rgba(251, 191, 36, 0.15);
    color: #fbbf24;
  }

  :global([data-theme="dark"]) .badge-danger {
    background: rgba(239, 68, 68, 0.15);
    color: #f87171;
  }

  :global([data-theme="dark"]) .badge-info {
    background: rgba(47, 45, 255, 0.15);
    color: #818cf8;
  }
</style>
