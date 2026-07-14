<script lang="ts">
  import FormComponentsDemo from '$lib/components/ui/FormComponentsDemo.svelte';
  import { getDefaultDivisionKey } from '$lib/divisions.svelte';
</script>

<svelte:head>
  <title>Form Components Demo - {getDefaultDivisionKey()}</title>
</svelte:head>

<FormComponentsDemo />
