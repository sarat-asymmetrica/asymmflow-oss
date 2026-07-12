<script lang="ts">
  import { run } from 'svelte/legacy';

  import { onMount } from "svelte";

  interface Props {
    text?: string;
    duration?: number;
    delay?: number;
    className?: string;
  }

  let {
    text = "",
    duration = 600,
    delay = 0,
    className = ""
  }: Props = $props();

  let displayChars = $state([]);
  let mounted = $state(false);


  onMount(() => {
    mounted = true;
    animateText();
  });

  function animateText() {
    const chars = text.toString().split('');
    displayChars = chars.map((char, i) => ({
      char,
      delay: delay + (i * duration) / chars.length
    }));
  }
  run(() => {
    if (text && mounted) {
      animateText();
    }
  });
</script>

<span class="text-bloom {className}">
  {#each displayChars as { char, delay: charDelay }, i (i + text)}
    <span class="bloom-char inline-block" style="animation-delay: {charDelay}ms">
      {char}
    </span>
  {/each}
</span>

<style>
  .text-bloom {
    display: inline-flex;
  }

  .bloom-char {
    opacity: 0;
    transform: scale(0.5) translateY(10px);
    animation: bloom 0.4s cubic-bezier(0.25, 0.46, 0.45, 0.94) forwards;
  }

  @keyframes bloom {
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }
</style>
