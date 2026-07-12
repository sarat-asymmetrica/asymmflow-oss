<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  interface Props {
    months?: number; // Total months to simulate
    currentMonth?: number; // Current position on slider
  }

  let { months = 12, currentMonth = $bindable(0) }: Props = $props();

  const dispatch = createEventDispatcher();

  function handleSlide(e: Event) {
    const target = e.target as HTMLInputElement;
    currentMonth = parseFloat(target.value);
    dispatch('timeChange', { month: currentMonth });
  }
</script>

<!-- No label - just the track, pure minimalism -->
<div class="time-machine">
  <input
    type="range"
    min="0"
    max={months}
    step="0.1"
    bind:value={currentMonth}
    oninput={handleSlide}
    class="time-slider"
    aria-label="Time machine - drag to see future"
  />
</div>

<style>
  .time-machine {
    width: 100%;
    padding: 21px 0; /* --space-md Fibonacci spacing */
  }

  .time-slider {
    width: 100%;
    height: 4px;
    background: linear-gradient(
      to right,
      var(--color-safe) 0%,       /* Green (safe zone) */
      var(--color-gold) 50%,      /* Gold (warning zone) */
      var(--color-danger) 100%    /* Red (danger zone) */
    );
    -webkit-appearance: none;
    appearance: none;
    border-radius: 2px;
    outline: none;
    cursor: grab;
    transition: all var(--duration-normal) var(--ease-wabi-sabi);
  }

  .time-slider:active {
    cursor: grabbing;
  }

  /* Webkit (Chrome, Safari) thumb */
  .time-slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 16px;
    height: 16px;
    background: var(--color-gold);
    border-radius: 50%;
    cursor: grab;
    box-shadow: 0 0 10px var(--color-gold);
    transition: all var(--duration-fast) var(--ease-wabi-sabi);
  }

  .time-slider::-webkit-slider-thumb:hover {
    width: 18px;
    height: 18px;
    box-shadow: 0 0 15px var(--color-gold);
  }

  .time-slider:active::-webkit-slider-thumb {
    cursor: grabbing;
    box-shadow: 0 0 20px var(--color-gold);
  }

  /* Firefox thumb */
  .time-slider::-moz-range-thumb {
    width: 16px;
    height: 16px;
    background: var(--color-gold);
    border-radius: 50%;
    border: none;
    cursor: grab;
    box-shadow: 0 0 10px var(--color-gold);
    transition: all var(--duration-fast) var(--ease-wabi-sabi);
  }

  .time-slider::-moz-range-thumb:hover {
    width: 18px;
    height: 18px;
    box-shadow: 0 0 15px var(--color-gold);
  }

  .time-slider:active::-moz-range-thumb {
    cursor: grabbing;
    box-shadow: 0 0 20px var(--color-gold);
  }
</style>
