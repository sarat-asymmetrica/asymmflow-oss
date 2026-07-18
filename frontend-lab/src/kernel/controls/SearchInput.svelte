<script lang="ts">
  let {
    value = $bindable(''),
    placeholder = 'Search…',
  }: {
    value?: string
    placeholder?: string
  } = $props()
</script>

<input class="k-search" type="search" bind:value {placeholder} />

<style>
  .k-search {
    font-family: var(--font-ui);
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    padding: 7px 12px;
    width: 240px;
    max-width: 100%;
    outline: none;
    transition: border-color var(--motion-fast) var(--ease-standard);
  }
  .k-search:focus {
    border-color: var(--onyx);
  }
  .k-search::placeholder {
    color: var(--text-muted);
  }
</style>
