<script lang="ts">
  import '$lib/styles/wabi-sabi.css';
  interface Props {
    children?: import('svelte').Snippet;
  }

  let { children }: Props = $props();
</script>

{@render children?.()}
