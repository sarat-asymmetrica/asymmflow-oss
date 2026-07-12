<script lang="ts">
  import { onMount } from 'svelte';

  let stars: Array<{x: number, y: number, size: number, opacity: number, twinkle: number}> = $state([]);
  const STAR_COUNT = 50; // Very sparse - subtle stars only

  onMount(() => {
    // Generate star positions
    stars = Array.from({ length: STAR_COUNT }, () => ({
      x: Math.random() * 100,
      y: Math.random() * 100,
      size: Math.random() * 1.5 + 0.5, // 0.5px to 2px
      opacity: Math.random() * 0.2 + 0.1, // 0.1 to 0.3 opacity (VERY subtle!)
      twinkle: Math.random() * 4 + 2 // 2-6 second twinkle duration
    }));
  });
</script>

<div class="celestial-stars">
  {#each stars as star}
    <div
      class="star"
      style="
        left: {star.x}%;
        top: {star.y}%;
        width: {star.size}px;
        height: {star.size}px;
        opacity: {star.opacity};
        animation-duration: {star.twinkle}s;
      "
></div>
  {/each}
</div>

<style>
  .celestial-stars {
    position: fixed;
    inset: 0;
    pointer-events: none;
    z-index: 0;
    background: #050505; /* Void black background */
  }

  .star {
    position: absolute;
    background: white;
    border-radius: 50%;
    animation: twinkle ease-in-out infinite;
  }

  @keyframes twinkle {
    0%, 100% {
      opacity: inherit;
    }
    50% {
      opacity: 0.05; /* Fade almost to invisible */
    }
  }
</style>
