<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    title,
    subtitle,
    actions,
    toolbar,
    children,
  }: {
    title: string
    subtitle?: string
    actions?: Snippet
    toolbar?: Snippet
    children: Snippet
  } = $props()
</script>

<div class="k-page">
  <header class="k-page-header">
    <div class="k-page-titles">
      <h1 class="k-page-title">{title}</h1>
      {#if subtitle}<p class="k-page-subtitle">{subtitle}</p>{/if}
    </div>
    {#if actions}
      <div class="k-page-actions">{@render actions()}</div>
    {/if}
  </header>

  {#if toolbar}
    <div class="k-page-toolbar">{@render toolbar()}</div>
  {/if}

  <!-- Layout doctrine: THE one default scroll region per screen. -->
  <div class="k-page-content">
    {@render children()}
  </div>
</div>

<style>
  .k-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    min-width: 0;
    padding: var(--page-padding);
    gap: var(--k-space-md);
    container-type: inline-size;
  }
  .k-page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--k-space-md);
    flex-shrink: 0;
    min-width: 0;
    /* Anti-collapse doctrine: when actions would squeeze the title below a
     * readable width, they wrap BELOW it instead. min-width:0 alone prevents
     * overflow but permits collapse-to-zero (one-letter-per-line titles). */
    flex-wrap: wrap;
  }
  .k-page-titles {
    min-width: 0;
    flex: 1 1 260px;
  }
  .k-page-title {
    font-family: var(--font-display);
    font-size: var(--page-title-size);
    font-weight: var(--page-title-weight);
    letter-spacing: var(--page-title-tracking);
    line-height: var(--line-height-tight);
    overflow-wrap: break-word;
  }
  .k-page-subtitle {
    color: var(--text-secondary);
    font-size: var(--meta-size);
    margin-top: var(--k-space-xs);
  }
  .k-page-actions {
    flex-shrink: 0;
    /* Never exceed the header width — a wide action group (e.g. a Hub's
     * period selector with many options) wraps within, rather than pushing
     * the page. Narrow button rows are unaffected. */
    max-width: 100%;
    min-width: 0;
  }
  .k-page-toolbar {
    flex-shrink: 0;
    min-width: 0;
  }
  .k-page-content {
    flex: 1;
    min-height: 0;
    min-width: 0;
    overflow-y: auto;
  }
</style>
