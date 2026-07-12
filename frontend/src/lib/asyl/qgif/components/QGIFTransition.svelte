<script lang="ts">
  import { fade } from 'svelte/transition';
  import QuaternionScenePlayer from '../../components/QuaternionScenePlayer.svelte';
  import transitionData from '../presets/transition_morph.json';

  interface Props {
    visible?: boolean;
  }

  let { visible = false }: Props = $props();
</script>

{#if visible}
  <div class="qgif-transition-overlay" transition:fade>
    <QuaternionScenePlayer qgifData={transitionData} width={window.innerWidth} height={window.innerHeight} />
  </div>
{/if}

<style>
  .qgif-transition-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    z-index: 9999;
    pointer-events: none;
    display: flex;
    justify-content: center;
    align-items: center;
    background: rgba(253, 251, 247, 0.8); /* Rice Paper with opacity */
  }
</style>
