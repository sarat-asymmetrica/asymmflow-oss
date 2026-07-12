<script lang="ts">
  
  
  interface Props {
    /**
   * Wabi-Sabi Divider
   * Elegant separators with optional labels
   * Like the space between verses in a haiku
   */
    label?: string;
    orientation?: 'horizontal' | 'vertical';
    variant?: 'solid' | 'dashed' | 'dotted' | 'gradient';
    spacing?: 'sm' | 'md' | 'lg';
  }

  let {
    label = '',
    orientation = 'horizontal',
    variant = 'solid',
    spacing = 'md'
  }: Props = $props();
</script>

{#if orientation === 'horizontal'}
  <div class="divider horizontal {variant} spacing-{spacing}">
    {#if label}
      <span class="divider-label">{label}</span>
    {/if}
  </div>
{:else}
  <div class="divider vertical {variant} spacing-{spacing}"></div>
{/if}

<style>
  .divider {
    position: relative;
  }
  
  /* Horizontal */
  .divider.horizontal {
    width: 100%;
    display: flex;
    align-items: center;
  }
  
  .divider.horizontal::before,
  .divider.horizontal::after {
    content: '';
    flex: 1;
    height: 1px;
    background: rgba(28, 28, 28, 0.1);
  }
  
  .divider.horizontal.dashed::before,
  .divider.horizontal.dashed::after {
    background: repeating-linear-gradient(
      90deg,
      rgba(28, 28, 28, 0.15) 0px,
      rgba(28, 28, 28, 0.15) 4px,
      transparent 4px,
      transparent 8px
    );
  }
  
  .divider.horizontal.dotted::before,
  .divider.horizontal.dotted::after {
    background: repeating-linear-gradient(
      90deg,
      rgba(28, 28, 28, 0.2) 0px,
      rgba(28, 28, 28, 0.2) 2px,
      transparent 2px,
      transparent 6px
    );
  }
  
  .divider.horizontal.gradient::before {
    background: linear-gradient(90deg, transparent, rgba(28, 28, 28, 0.15));
  }
  
  .divider.horizontal.gradient::after {
    background: linear-gradient(90deg, rgba(28, 28, 28, 0.15), transparent);
  }
  
  /* Vertical */
  .divider.vertical {
    width: 1px;
    height: 100%;
    background: rgba(28, 28, 28, 0.1);
  }
  
  .divider.vertical.dashed {
    background: repeating-linear-gradient(
      180deg,
      rgba(28, 28, 28, 0.15) 0px,
      rgba(28, 28, 28, 0.15) 4px,
      transparent 4px,
      transparent 8px
    );
  }
  
  .divider.vertical.gradient {
    background: linear-gradient(180deg, transparent, rgba(28, 28, 28, 0.15), transparent);
  }
  
  /* Label */
  .divider-label {
    padding: 0 13px;
    font-family: 'Courier Prime', monospace;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 2px;
    color: #57534e;
    white-space: nowrap;
  }
  
  /* Spacing */
  .divider.spacing-sm { margin: 8px 0; }
  .divider.spacing-md { margin: 21px 0; }
  .divider.spacing-lg { margin: 34px 0; }
  
  .divider.vertical.spacing-sm { margin: 0 8px; }
  .divider.vertical.spacing-md { margin: 0 21px; }
  .divider.vertical.spacing-lg { margin: 0 34px; }
</style>
