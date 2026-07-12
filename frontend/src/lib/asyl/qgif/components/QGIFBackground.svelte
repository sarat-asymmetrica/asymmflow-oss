<script lang="ts">
  import QuaternionScenePlayer from '../../components/QuaternionScenePlayer.svelte';
  import breathingData from '../presets/breathing_pulse.json';
</script>

<div class="qgif-background">
  <QuaternionScenePlayer qgifData={breathingData} width={800} height={800} loop={true} />
</div>

<style>
  .qgif-background {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    opacity: 0.1;
    z-index: 0;
    pointer-events: none;
  }
</style>
