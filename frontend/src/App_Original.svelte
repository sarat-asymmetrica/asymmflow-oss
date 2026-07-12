<script lang="ts">
  import "./app.css";
  import AlchemyRenderer from "./lib/components/AlchemyRenderer.svelte";
  import OrigamiNav from "./lib/components/ui/OrigamiNav.svelte";
  import SystemFooter from "./lib/components/ui/SystemFooter.svelte";
  import { onMount } from "svelte";
  import Lenis from "lenis";

  let currentScreen = $state("dashboard");
  let isNavCollapsed = $state(false);
  let mainContainer: HTMLElement = $state();
  let lenis: Lenis;

  function handleNavigate(event) {
    currentScreen = event.detail;
  }

  function handleCollapse(event) {
    isNavCollapsed = event.detail;
  }

  onMount(() => {
    // Initialize Smooth Scroll
    lenis = new Lenis({
      wrapper: mainContainer,
      content: mainContainer.querySelector('div'), // The inner div
      duration: 1.2,
      easing: (t) => Math.min(1, 1.001 - Math.pow(2, -10 * t)), // Exponential ease
      orientation: 'vertical',
      gestureOrientation: 'vertical',
      smoothWheel: true,
      wheelMultiplier: 1,
    });

    function raf(time) {
      lenis.raf(time);
      requestAnimationFrame(raf);
    }
    requestAnimationFrame(raf);

    return () => {
      lenis.destroy();
    };
  });
</script>

<div class="flex h-screen w-screen overflow-hidden bg-[var(--color-paper)] text-[var(--color-ink)]">
  <!-- Origami Navigation - 3D Folding Paper Segments -->
  <OrigamiNav
    activeRoute={currentScreen}
    collapsed={isNavCollapsed}
    on:navigate={handleNavigate}
    on:collapse={handleCollapse}
  />

  <!-- Main Content Area (Scrollable) -->
  <main
    bind:this={mainContainer}
    class="flex-1 h-full overflow-y-auto overflow-x-hidden relative transition-all duration-300 ease-wabi-sabi pb-8 scroll-smooth-container"
  >
    <div class="min-h-full w-full">
      {#key currentScreen}
        <AlchemyRenderer screenID={currentScreen} />
      {/key}
    </div>
  </main>

  <SystemFooter {isNavCollapsed} />
</div>
