<script lang="ts" module>
  // Canonical tone type now lives in ../tones; re-exported here so existing
  // `import type { Tone } from '../controls/Badge.svelte'` sites keep working.
  export type { Tone } from '../tones'
</script>

<script lang="ts">
  import type { Tone } from '../tones'

  let {
    tone = 'neutral',
    label,
  }: {
    tone?: Tone
    label: string
  } = $props()
</script>

<span
  class="k-badge"
  style:background="var(--k-tone-{tone}-bg)"
  style:color="var(--k-tone-{tone}-fg)"
  title={label}>{label}</span>

<style>
  .k-badge {
    display: inline-block;
    font-size: calc(11px * var(--ui-font-scale));
    font-weight: 600;
    letter-spacing: 0.01em;
    padding: 2px 10px;
    border-radius: var(--border-radius-pill);
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    vertical-align: middle;
  }
</style>
