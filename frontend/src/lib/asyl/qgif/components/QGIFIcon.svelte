<script lang="ts">
  import { run } from 'svelte/legacy';

  import QuaternionScenePlayer from '../../components/QuaternionScenePlayer.svelte';

  import successData from '../presets/success_burst.json';
  import errorData from '../presets/error_shake.json';
  interface Props {
    icon?: string; // default
    size?: number;
  }

  let { icon = 'success_burst', size = 24 }: Props = $props();

  let data = $state();
  run(() => {
    if (icon === 'success_burst') data = successData;
    else if (icon === 'error_shake') data = errorData;
    else data = successData;
  });
</script>

<div class="qgif-icon" style="width: {size}px; height: {size}px;">
  <QuaternionScenePlayer qgifData={data} width={size} height={size} />
</div>
