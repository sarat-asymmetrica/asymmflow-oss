<script lang="ts">
  import HoloCard from "../asyl/components/HoloCard.svelte";
  import TextBloom from "./ui/TextBloom.svelte";

  let { label, value } = $props();
</script>

<HoloCard>
  <div class="component-card flex flex-col justify-center items-center text-center" style="padding: var(--space-md)">
    <span class="metric-big text-3xl font-serif font-bold text-[var(--accent-color)]">
      {#key value}
        <TextBloom text={value} duration={800} className="metric-big" />
      {/key}
    </span>
    <span class="metric-label text-xs font-mono uppercase tracking-widest opacity-60 mt-2">{label}</span>
  </div>
</HoloCard>
