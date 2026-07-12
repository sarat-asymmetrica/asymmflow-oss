<!-- @migration-task Error while migrating Svelte code: $$props is used together with named props in a way that cannot be automatically migrated. -->
<script lang="ts">
  export let variant: 'default' | 'elevated' | 'accent' = 'default';
  export let padding: 'sm' | 'md' | 'lg' = 'md';
  export let hoverable: boolean = false;

  $: ariaLabel = $$props['aria-label'] || undefined;
</script>

{#if hoverable}
  <div
    class="card"
    class:card-elevated={variant === 'elevated'}
    class:card-accent={variant === 'accent'}
    class:card-hoverable={hoverable}
    class:padding-sm={padding === 'sm'}
    class:padding-md={padding === 'md'}
    class:padding-lg={padding === 'lg'}
    aria-label={ariaLabel}
    role="button"
    tabindex="0"
    on:click
    on:keydown
  >
    <slot />
  </div>
{:else}
  <div
    class="card"
    class:card-elevated={variant === 'elevated'}
    class:card-accent={variant === 'accent'}
    class:card-hoverable={hoverable}
    class:padding-sm={padding === 'sm'}
    class:padding-md={padding === 'md'}
    class:padding-lg={padding === 'lg'}
    aria-label={ariaLabel}
  >
    <slot />
  </div>
{/if}

<style>
  .card {
    background: var(--surface);
    border-radius: var(--border-radius);
    box-shadow: var(--shadow-sm);
    transition: box-shadow var(--transition-fast);
  }

  .card-elevated {
    background: var(--surface-elevated);
    box-shadow: var(--shadow-md);
  }

  .card-accent {
    border-left: 3px solid var(--brand-indigo);
  }

  .card-hoverable {
    cursor: pointer;
  }

  .card-hoverable:hover {
    box-shadow: var(--shadow-md);
  }

  .card-hoverable:focus-visible {
    outline: 2px solid var(--brand-indigo);
    outline-offset: 2px;
  }

  /* Padding variants */
  .padding-sm {
    padding: 8px;
  }

  .padding-md {
    padding: var(--card-padding);
  }

  .padding-lg {
    padding: var(--card-padding-lg);
  }
</style>
