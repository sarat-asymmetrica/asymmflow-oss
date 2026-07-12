<script lang="ts">
  export type ActionProposalReviewStatus = "approved" | "needs_input" | "rejected";

  export type ActionProposalItem = {
    action?: string;
    source_type?: string;
    label: string;
    reason: string;
    priority?: string;
    required_deterministic_service?: string;
  };

  interface Props {
    proposal: ActionProposalItem;
    reviewLabel?: string;
    hasReview?: boolean;
    reviewing?: boolean;
    onApprove?: () => void;
    onNeedsInput?: () => void;
    onReject?: () => void;
  }

  let {
    proposal,
    reviewLabel = "",
    hasReview = false,
    reviewing = false,
    onApprove,
    onNeedsInput,
    onReject,
  }: Props = $props();
</script>

<article class="action-proposal-card" data-priority={proposal.priority || ""}>
  <div class="proposal-copy">
    <span>{proposal.source_type || "cashflow evidence"}</span>
    <strong>{proposal.label}</strong>
    <small>{proposal.reason}</small>
  </div>
  <div class="proposal-review">
    <em>{reviewLabel || proposal.required_deterministic_service}</em>
    {#if hasReview}
      <div>
        <button type="button" disabled={reviewing} onclick={onApprove}>Approve</button>
        <button type="button" disabled={reviewing} onclick={onNeedsInput}>Needs input</button>
        <button type="button" disabled={reviewing} onclick={onReject}>Reject</button>
      </div>
    {/if}
  </div>
</article>

<style>
  .action-proposal-card {
    min-width: 0;
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(220px, 0.42fr);
    align-items: start;
    gap: 12px;
    padding: 12px 16px;
    background: var(--paper, #fff);
    box-shadow: inset 3px 0 0 transparent;
  }

  .action-proposal-card[data-priority="high"],
  .action-proposal-card[data-priority="urgent"],
  .action-proposal-card[data-priority="critical"] {
    box-shadow: inset 3px 0 0 #b45309;
  }

  .proposal-copy {
    min-width: 0;
    display: grid;
    gap: 3px;
  }

  span {
    color: var(--ink-light, #666);
    font-size: 11px;
    text-transform: uppercase;
  }

  strong {
    min-width: 0;
    color: var(--ink, #1c1c1c);
    font-size: 13px;
    overflow-wrap: anywhere;
  }

  small,
  em {
    min-width: 0;
    color: var(--ink-light, #666);
    font-size: 11px;
    font-style: normal;
    overflow-wrap: anywhere;
  }

  .proposal-review {
    min-width: 0;
    display: grid;
    justify-items: end;
    gap: 8px;
  }

  em {
    justify-self: end;
    font-family: var(--font-mono, ui-monospace, SFMono-Regular, Consolas, monospace);
    text-align: right;
  }

  .proposal-review > div {
    display: flex;
    justify-content: flex-end;
    gap: 6px;
    flex-wrap: wrap;
  }

  button {
    border: 1px solid var(--border-subtle, #e5e1d8);
    background: var(--paper-soft, #fafafa);
    color: var(--ink, #1c1c1c);
    font-size: 11px;
    padding: 6px 8px;
  }

  button:hover:not(:disabled) {
    border-color: var(--ink-light, #666);
  }

  button:disabled {
    cursor: progress;
    opacity: 0.6;
  }

  @media (max-width: 860px) {
    .action-proposal-card {
      grid-template-columns: 1fr;
    }

    .proposal-review {
      justify-items: start;
    }

    em {
      justify-self: start;
      text-align: left;
    }

    .proposal-review > div {
      justify-content: flex-start;
    }
  }
</style>
