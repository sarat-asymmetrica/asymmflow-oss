<script lang="ts">
  /**
   * Wabi-Sabi Page Transition
   * Smooth, zen-like transitions between screens
   * Like turning a page in a beautiful book
   */
  import { fade, fly } from 'svelte/transition';
  import { cubicOut, quintOut } from 'svelte/easing';
  import { motionMs } from '../../motion';
  
  interface Props {
    key: string | number;
    direction?: 'up' | 'down' | 'left' | 'right' | 'fade';
    duration?: number;
    children?: import('svelte').Snippet;
  }

  let {
    key,
    direction = 'fade',
    duration = 300,
    children
  }: Props = $props();
  
  const transitions = {
    up: { y: 20, x: 0 },
    down: { y: -20, x: 0 },
    left: { y: 0, x: 20 },
    right: { y: 0, x: -20 },
    fade: { y: 0, x: 0 },
  };
  
  let trans = $derived(transitions[direction]);
</script>

{#key key}
  <div
    class="page-transition"
    in:fly={{ y: trans.y, x: trans.x, duration: motionMs(duration), easing: quintOut, delay: 50 }}
    out:fade={{ duration: motionMs(duration) / 2 }}
  >
    {@render children?.()}
  </div>
{/key}

<style>
  .page-transition {
    width: 100%;
    height: 100%;
  }
</style>
