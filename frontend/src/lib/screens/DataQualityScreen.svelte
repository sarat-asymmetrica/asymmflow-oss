<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import WabiSpinner from "$lib/components/ui/WabiSpinner.svelte";
  import {
    GetDataQualityReviewHistory,
    PreviewCustomerDataQuality,
    ReviewDataQualityIssue,
  } from "../../../wailsjs/go/main/App";
  import { toast } from "$lib/stores/toasts";

  type DataQualityIssue = {
    id: string;
    issue_type: string;
    severity: string;
    entity_type: string;
    entity_id: string;
    summary: string;
    detail: string;
    primary_action: string;
    review_status?: string;
    review_note?: string;
    reviewed_by?: string;
    reviewed_at?: string;
  };

  type DataQualityReview = {
    id: string;
    issue_id: string;
    issue_type: string;
    entity_type: string;
    entity_id: string;
    summary: string;
    status: string;
    review_note: string;
    reviewed_by: string;
    reviewed_at?: string;
    updated_at?: string;
  };

  let loading = true;
  let actionBusy = "";
  let issues: DataQualityIssue[] = [];
  let reviewHistory: DataQualityReview[] = [];
  let filter = "all";
  let search = "";
  let reviewNotes: Record<string, string> = {};
  const dispatch = createEventDispatcher();

  async function loadIssues() {
    loading = true;
    try {
      const [nextIssues, history] = await Promise.all([
        PreviewCustomerDataQuality(300),
        GetDataQualityReviewHistory(50),
      ]);
      issues = (nextIssues || []) as any;
      reviewHistory = (history || []) as any;
    } catch (err) {
      toast.danger(`Failed to load data quality queue: ${String(err)}`);
      issues = [];
    } finally {
      loading = false;
    }
  }

  async function reviewIssue(issue: DataQualityIssue, action: "reviewed" | "resolved" | "dismissed") {
    actionBusy = `${issue.id}:${action}`;
    try {
      await ReviewDataQualityIssue(issue as any, action, reviewNotes[issue.id] || "");
      reviewNotes = { ...reviewNotes, [issue.id]: "" };
      toast.success(action === "reviewed" ? "Issue marked reviewed" : `Issue ${action}`);
      await loadIssues();
    } catch (err) {
      toast.danger(`Data quality review failed: ${String(err)}`);
    } finally {
      actionBusy = "";
    }
  }

  function setReviewNote(issueID: string, event: Event) {
    const target = event.currentTarget as HTMLTextAreaElement;
    reviewNotes = {
      ...reviewNotes,
      [issueID]: target.value,
    };
  }

  function openIssue(issue: DataQualityIssue) {
    dispatch("openIssue", { issue });
  }

  function matchesSearch(issue: DataQualityIssue) {
    const term = search.trim().toLowerCase();
    if (!term) return true;
    return [issue.summary, issue.detail, issue.primary_action, issue.entity_type, issue.entity_id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(term));
  }

  $: filteredIssues = issues.filter((issue) => (filter === "all" || issue.issue_type === filter) && matchesSearch(issue));
  $: duplicateCount = issues.filter((issue) => issue.issue_type === "duplicate_customer").length;
  $: orphanCount = issues.filter((issue) => issue.issue_type.includes("missing") || issue.issue_type.includes("orphan")).length;
  $: blankCount = issues.filter((issue) => issue.issue_type.includes("blank")).length;

  onMount(loadIssues);
</script>

<section class="data-quality">
  <header class="workspace-head">
    <div>
      <h2>Data Quality</h2>
      <p>Admin-reviewed cleanup queue for duplicates, blank commercial records, and missing links.</p>
    </div>
    <Button variant="secondary" size="sm" on:click={loadIssues} disabled={loading}>Refresh</Button>
  </header>

  <div class="kpi-strip">
    <div class="kpi">
      <span>Total Issues</span>
      <strong>{issues.length}</strong>
    </div>
    <div class="kpi">
      <span>Duplicates</span>
      <strong>{duplicateCount}</strong>
    </div>
    <div class="kpi">
      <span>Missing Links</span>
      <strong>{orphanCount}</strong>
    </div>
    <div class="kpi">
      <span>Blank Records</span>
      <strong>{blankCount}</strong>
    </div>
  </div>

  <div class="toolbar">
    <select bind:value={filter}>
      <option value="all">All issue types</option>
      <option value="duplicate_customer">Duplicate customers</option>
      <option value="blank_customer_name">Blank customers</option>
      <option value="blank_opportunity_name">Blank opportunities</option>
      <option value="missing_customer_link">Missing opportunity customer</option>
      <option value="offer_missing_customer">Missing offer customer</option>
    </select>
    <input bind:value={search} placeholder="Search customer, opportunity, offer, detail" />
  </div>

  {#if loading}
    <div class="state">
      <WabiSpinner size="md" tempo="calm" />
      <span>Scanning records...</span>
    </div>
  {:else}
    <div class="issue-list">
      {#each filteredIssues as issue}
        <article class="issue-row">
          <div>
            <div class="issue-topline">
              <span class="severity {issue.severity || 'review'}">{issue.severity || "review"}</span>
              {#if issue.review_status}
                <span class="review-status">{issue.review_status}</span>
              {/if}
              <span class="entity">{issue.entity_type || "record"} · {issue.entity_id || "-"}</span>
            </div>
            <h3>{issue.summary}</h3>
            <p>{issue.detail}</p>
            {#if issue.review_note}
              <p class="review-note">Reviewed by {issue.reviewed_by || "admin"}: {issue.review_note}</p>
            {/if}
          </div>
          <div class="action-note">
            <span>Suggested Action</span>
            <strong>{issue.primary_action || "Review manually"}</strong>
            <textarea
              value={reviewNotes[issue.id] || ""}
              on:input={(event) => setReviewNote(issue.id, event)}
              placeholder="Admin review note"
              rows="2"
            ></textarea>
            <div class="review-actions">
              <Button
                variant="secondary"
                size="sm"
                on:click={() => openIssue(issue)}
                disabled={!!actionBusy}
              >
                Open record
              </Button>
              <Button
                variant="secondary"
                size="sm"
                on:click={() => reviewIssue(issue, "reviewed")}
                disabled={!!actionBusy}
              >
                {actionBusy === `${issue.id}:reviewed` ? "Saving..." : "Mark reviewed"}
              </Button>
              <Button
                variant="secondary"
                size="sm"
                on:click={() => reviewIssue(issue, "resolved")}
                disabled={!!actionBusy}
              >
                Resolve
              </Button>
              <Button
                variant="ghost"
                size="sm"
                on:click={() => reviewIssue(issue, "dismissed")}
                disabled={!!actionBusy}
              >
                Dismiss
              </Button>
            </div>
          </div>
        </article>
      {:else}
        <div class="state">No data quality issues match the current filters.</div>
      {/each}
    </div>
  {/if}

  {#if reviewHistory.length}
    <section class="review-history" aria-label="Data quality review history">
      <div class="section-title">
        <h3>Review History</h3>
        <span>{reviewHistory.length} recent decisions</span>
      </div>
      <div class="history-table">
        {#each reviewHistory.slice(0, 8) as review}
          <div class="history-row">
            <strong>{review.status}</strong>
            <span>{review.summary || review.issue_id}</span>
            <span>{review.reviewed_by || "Admin"}</span>
            <span>{review.review_note || "-"}</span>
          </div>
        {/each}
      </div>
    </section>
  {/if}
</section>

<style>
  .data-quality {
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .workspace-head,
  .toolbar,
  .issue-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  h2,
  h3,
  p {
    margin: 0;
  }

  h2 {
    font-size: var(--page-title-size-scaled, 24px);
  }

  p {
    color: var(--text-muted, #6b7280);
  }

  .kpi-strip {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
  }

  .kpi,
  .issue-row {
    border: 1px solid var(--border, #e5e7eb);
    border-radius: 8px;
    background: var(--surface, #fff);
  }

  .kpi {
    padding: 14px;
  }

  .kpi span,
  .action-note span {
    color: var(--text-muted, #6b7280);
    font-size: var(--modal-label-size, 12px);
    font-weight: 700;
    text-transform: uppercase;
  }

  .kpi strong {
    display: block;
    margin-top: 8px;
    font-size: 22px;
  }

  .toolbar select,
  .toolbar input {
    padding: 10px 12px;
  }

  .toolbar input {
    min-width: 360px;
  }

  .issue-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .issue-row {
    align-items: flex-start;
    padding: 14px;
  }

  .issue-topline {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .review-status {
    padding: 2px 8px;
    border-radius: 999px;
    background: #ecfdf5;
    color: #047857;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
  }

  .severity {
    padding: 2px 8px;
    border-radius: 999px;
    background: #eef2ff;
    color: #3730a3;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
  }

  .severity.high,
  .severity.critical {
    background: #fef2f2;
    color: #b91c1c;
  }

  .entity {
    color: var(--text-muted, #6b7280);
    font-size: 12px;
  }

  .action-note {
    min-width: 240px;
    text-align: right;
  }

  .action-note strong {
    display: block;
    margin-top: 6px;
  }

  .action-note textarea {
    margin-top: 10px;
    width: 100%;
    min-width: 260px;
    resize: vertical;
    border: 1px solid var(--border, #e5e7eb);
    border-radius: 8px;
    padding: 8px 10px;
    font: inherit;
  }

  .review-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 10px;
    flex-wrap: wrap;
  }

  .review-note {
    margin-top: 8px;
    color: #047857;
  }

  .review-history {
    border: 1px solid var(--border, #e5e7eb);
    border-radius: 8px;
    background: var(--surface, #fff);
    padding: 14px;
  }

  .section-title,
  .history-row {
    display: grid;
    grid-template-columns: 120px 1.4fr 160px 1fr;
    gap: 12px;
    align-items: center;
  }

  .section-title {
    display: flex;
    justify-content: space-between;
    margin-bottom: 10px;
  }

  .section-title span,
  .history-row span {
    color: var(--text-muted, #6b7280);
  }

  .history-table {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .history-row {
    padding: 10px 0;
    border-top: 1px solid var(--border, #e5e7eb);
  }

  .history-row strong {
    text-transform: uppercase;
    font-size: 12px;
  }

  .state {
    padding: 32px;
    border: 1px solid var(--border, #e5e7eb);
    border-radius: 8px;
    display: flex;
    justify-content: center;
    gap: 12px;
    color: var(--text-muted, #6b7280);
  }

  @media (max-width: 900px) {
    .data-quality {
      padding: 16px;
    }

    .workspace-head,
    .toolbar,
    .issue-row {
      align-items: stretch;
      flex-direction: column;
    }

    .kpi-strip {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .toolbar input,
    .action-note {
      min-width: 0;
      width: 100%;
      text-align: left;
    }

    .review-actions {
      justify-content: flex-start;
    }

    .section-title,
    .history-row {
      display: flex;
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
