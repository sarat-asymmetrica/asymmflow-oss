<script lang="ts">
  import HoloCard from "./HoloCard.svelte";

  interface Props {
    message?: string;
    sentiment?: string; // calm, urgent, happy
  }

  let { message = "", sentiment = "neutral" }: Props = $props();

  const colors = {
    calm: "#2d4a6f",   // Blue
    urgent: "#ef4444", // Red
    happy: "#15803d",  // Green
    neutral: "#475569" // Gray
  };

  const icons = {
    calm: "Calm",
    urgent: "!",
    happy: "OK",
    neutral: "AI"
  };
</script>

<HoloCard>
  <div class="component-card flex items-start gap-4 p-6 bg-white/80 border-l-4"
       style="border-left-color: {colors[sentiment]}; min-height: auto;">
    <div class="text-2xl select-none">{icons[sentiment]}</div>
    <div>
      <div class="text-xs font-mono uppercase tracking-widest text-gray-400 mb-1">Butler Insight</div>
      <div class="font-serif text-lg text-[#1c1c1c] leading-snug">
        {message}
      </div>
    </div>
  </div>
</HoloCard>
