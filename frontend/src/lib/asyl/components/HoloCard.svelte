<script lang="ts">
  import { spring } from "svelte/motion";
  interface Props {
    children?: import('svelte').Snippet;
  }

  let { children }: Props = $props();

  let card = $state<HTMLDivElement | null>(null);
  let width, height;
  let mouseX = 0;
  let mouseY = 0;

  // Springs for smooth tilt
  const rotateX = spring(0, { stiffness: 0.05, damping: 0.3 });
  const rotateY = spring(0, { stiffness: 0.05, damping: 0.3 });
  const glareX = spring(50, { stiffness: 0.05, damping: 0.3 });
  const glareY = spring(50, { stiffness: 0.05, damping: 0.3 });

  function handleMouseMove(e: MouseEvent) {
    if (!card) return;
    const rect = card.getBoundingClientRect();
    width = rect.width;
    height = rect.height;

    // Calculate mouse position relative to card center (-1 to 1)
    const x = (e.clientX - rect.left) / width - 0.5;
    const y = (e.clientY - rect.top) / height - 0.5;

    // Tilt (inverted for natural feel)
    rotateX.set(-y * 20); // Max 20 deg tilt
    rotateY.set(x * 20);

    // Glare position (0% to 100%)
    glareX.set(((e.clientX - rect.left) / width) * 100);
    glareY.set(((e.clientY - rect.top) / height) * 100);
  }

  function handleMouseLeave() {
    rotateX.set(0);
    rotateY.set(0);
    glareX.set(50);
    glareY.set(50);
  }
</script>

<div
  class="perspective-1000"
  onmousemove={handleMouseMove}
  onmouseleave={handleMouseLeave}
  role="group"
>
  <div
    bind:this={card}
    class="relative bg-[var(--color-paper)]/90 border border-[var(--color-ink)]/10 backdrop-blur-sm rounded-xl overflow-hidden transition-shadow duration-300 hover:shadow-2xl hover:shadow-[var(--color-gold)]/20 group"
    style="transform: rotateX({$rotateX}deg) rotateY({$rotateY}deg); transform-style: preserve-3d;"
  >
    <!-- Holographic Glare -->
    <div
      class="absolute inset-0 pointer-events-none z-50 mix-blend-overlay opacity-0 group-hover:opacity-60 transition-opacity duration-500"
      style="background: radial-gradient(circle at {$glareX}% {$glareY}%, rgba(197, 160, 89, 0.8) 0%, transparent 60%);"
    ></div>

    <!-- Subtle Scanline Texture (Wabi-Sabi Imperfection) -->
    <div
      class="absolute inset-0 pointer-events-none z-40 opacity-5 bg-[url('data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI0IiBoZWlnaHQ9IjQiPgo8cmVjdCB3aWR0aD0iNCIgaGVpZ2h0PSIxIiBmaWxsPSIjMWMxYzFjIiAvPgo8L3N2Zz4=')]"
    ></div>

    <!-- Content Slot with 3D Transform -->
    <div class="relative z-10 transform translate-z-10">
      {@render children?.()}
    </div>
  </div>
</div>

<style>
  .perspective-1000 {
    perspective: 1000px;
  }
  .translate-z-10 {
    transform: translateZ(20px);
  }
</style>
