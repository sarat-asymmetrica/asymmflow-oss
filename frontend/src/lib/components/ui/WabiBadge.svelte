<script lang="ts">
  
  
  interface Props {
    /**
   * Wabi-Sabi Badge
   * Elegant status indicators with semantic meaning
   */
    variant?: 'default' | 'success' | 'warning' | 'danger' | 'info';
    size?: 'sm' | 'md' | 'lg';
    pulse?: boolean;
    dot?: boolean;
    children?: import('svelte').Snippet;
  }

  let {
    variant = 'default',
    size = 'md',
    pulse = false,
    dot = false,
    children
  }: Props = $props();
</script>

<span class="badge {variant} {size}" class:pulse class:dot>
  {#if dot}
    <span class="dot-indicator"></span>
  {/if}
  {@render children?.()}
</span>

<style>
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-family: 'Courier Prime', monospace;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border-radius: 4px;
    white-space: nowrap;
  }
  
  /* Sizes */
  .badge.sm {
    padding: 2px 6px;
    font-size: 9px;
  }
  
  .badge.md {
    padding: 4px 10px;
    font-size: 10px;
  }
  
  .badge.lg {
    padding: 6px 13px;
    font-size: 11px;
  }
  
  /* Variants */
  .badge.default {
    background: rgba(28, 28, 28, 0.08);
    color: #1c1c1c;
  }
  
  .badge.success {
    background: rgba(21, 128, 61, 0.1);
    color: #15803d;
  }
  
  .badge.warning {
    background: rgba(217, 119, 6, 0.1);
    color: #d97706;
  }
  
  .badge.danger {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
  }
  
  .badge.info {
    background: rgba(59, 130, 246, 0.1);
    color: #3b82f6;
  }
  
  /* Dot indicator */
  .dot-indicator {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: currentColor;
  }
  
  .badge.pulse .dot-indicator {
    animation: pulse 2s ease-in-out infinite;
  }
  
  @keyframes pulse {
    0%, 100% { opacity: 1; transform: scale(1); }
    50% { opacity: 0.6; transform: scale(0.9); }
  }
  
  /* Dot-only variant */
  .badge.dot:empty {
    padding: 0;
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }
  
  .badge.dot.success:empty { background: #15803d; }
  .badge.dot.warning:empty { background: #d97706; }
  .badge.dot.danger:empty { background: #ef4444; }
  .badge.dot.info:empty { background: #3b82f6; }
</style>
