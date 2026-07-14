<script lang="ts">
  export type Tone = 'neutral' | 'info' | 'success' | 'warning' | 'danger'

  let {
    tone = 'neutral',
    label,
  }: {
    tone?: Tone
    label: string
  } = $props()
</script>

<span class="k-badge k-badge-{tone}" title={label}>{label}</span>

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
  .k-badge-neutral {
    background: var(--onyx-tint);
    color: var(--text-secondary);
  }
  .k-badge-info {
    background: rgba(10, 90, 180, 0.1);
    color: #0a5ab4;
  }
  .k-badge-success {
    background: rgba(30, 130, 76, 0.12);
    color: #1e824c;
  }
  .k-badge-warning {
    background: rgba(180, 120, 10, 0.12);
    color: #9a6700;
  }
  .k-badge-danger {
    background: rgba(179, 38, 30, 0.1);
    color: #b3261e;
  }
</style>
