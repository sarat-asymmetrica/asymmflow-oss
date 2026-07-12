<script lang="ts">
  export let title: string = '';
  export let subtitle: string = '';
  export let embedded: boolean = false;
</script>

<div class="page-layout" class:embedded>
  {#if title || subtitle}
    <div class="page-header">
      {#if title}
        <h1 class="page-title">{title}</h1>
      {/if}
      {#if subtitle}
        <p class="page-subtitle">{subtitle}</p>
      {/if}
      <slot name="header-actions" />
    </div>
  {/if}

  <div class="page-content">
    <slot />
  </div>
</div>

<style>
  .page-layout {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }

  .page-header {
    padding: var(--page-padding);
    background: var(--bg-base);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    gap: 16px;
    min-height: var(--header-height);
  }

  .page-title {
    font-size: var(--page-title-size);
    font-weight: var(--page-title-weight);
    color: var(--text-primary);
    line-height: var(--line-height-tight);
    margin: 0;
  }

  .page-subtitle {
    font-size: var(--label-size);
    color: var(--text-secondary);
    margin: 0;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .page-content {
    flex: 1;
    padding: var(--page-padding);
    overflow-y: auto;
    min-height: 0;
  }

  /* Embedded mode - no header, compact padding */
  .page-layout.embedded .page-header {
    display: none;
  }

  .page-layout.embedded .page-content {
    padding: 0;
  }
</style>
