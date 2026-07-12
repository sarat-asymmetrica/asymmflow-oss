<script lang="ts">
  export type EvidenceSourceItem = {
    source_type?: string;
    label: string;
    required?: number;
    present?: number;
    missing?: number;
    confidence?: number;
    status?: string;
    priority?: string;
    last_updated?: string;
  };

  interface Props {
    sources: EvidenceSourceItem[];
  }

  let { sources }: Props = $props();

  function confidencePercent(source: EvidenceSourceItem): number {
    return Math.round((source.confidence || 0) * 100);
  }
</script>

<div class="evidence-source-list">
  {#each sources as source}
    <div class="evidence-source" data-priority={source.priority || ""} data-status={source.status || ""}>
      <span>{source.label}</span>
      <strong>{source.present ?? 0}/{source.required ?? 0}</strong>
      <small>{source.missing ?? 0} missing / {confidencePercent(source)}%</small>
    </div>
  {/each}
</div>

<style>
  .evidence-source-list {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1px;
    background: var(--border-subtle, #e5e1d8);
    border-top: 1px solid var(--border-subtle, #e5e1d8);
  }

  .evidence-source {
    min-width: 0;
    display: grid;
    gap: 4px;
    padding: 12px;
    background: var(--paper, #fff);
    box-shadow: inset 3px 0 0 transparent;
  }

  .evidence-source[data-priority="high"],
  .evidence-source[data-priority="critical"],
  .evidence-source[data-priority="urgent"],
  .evidence-source[data-status="blocked"],
  .evidence-source[data-status="critical"] {
    box-shadow: inset 3px 0 0 #b45309;
  }

  .evidence-source[data-status="ready"] {
    box-shadow: inset 3px 0 0 #166534;
  }

  span {
    color: var(--ink-light, #666);
    font-size: 11px;
    text-transform: uppercase;
  }

  strong {
    min-width: 0;
    color: var(--ink, #1c1c1c);
    font-family: var(--font-mono, ui-monospace, SFMono-Regular, Consolas, monospace);
    font-size: 16px;
    overflow-wrap: anywhere;
  }

  small {
    min-width: 0;
    color: var(--ink-light, #666);
    font-size: 11px;
    overflow-wrap: anywhere;
  }

  @media (max-width: 860px) {
    .evidence-source-list {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 560px) {
    .evidence-source-list {
      grid-template-columns: 1fr;
    }
  }
</style>
