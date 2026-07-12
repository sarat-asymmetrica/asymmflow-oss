<script lang="ts">
    import { createBubbler, stopPropagation } from 'svelte/legacy';

    const bubble = createBubbler();
    /**
     * OfferCard - Enhanced with Follow-up Timeline & Lost Reason
     *
     * Phase 2 Enhancement:
     * - Follow-up timeline display
     * - Lost reason capture modal
     * - Revision history indicator
     * - Email send action
    */
    import { createEventDispatcher } from "svelte";
    import { slide } from "svelte/transition";
    import { formatNumber } from "$lib/utils/formatters";

    let { offer } = $props();
    export const compact = false;

    const dispatch = createEventDispatcher();

    let showLostReasonModal = $state(false);
    let lostReason = $state("");
    let showTimeline = $state(false);

    const statusColors = {
        Pending: "#eab308",
        Sent: "#3b82f6",
        Accepted: "#15803d",
        Rejected: "#ef4444",
        Lost: "#9ca3af",
    };

    const lostReasons = [
        "Price too high",
        "Competitor won",
        "Customer budget constraints",
        "Project cancelled",
        "Timeline not met",
        "Technical requirements",
        "Other",
    ];

    function openPDF() {
        if (!offer?.pdfPath) return;
        const target = offer.pdfPath.startsWith("file://")
            ? offer.pdfPath
            : `file://${offer.pdfPath}`;
        window.open(target, "_blank");
    }

    function changeStatus(next) {
        if (next === "Lost" || next === "Rejected") {
            showLostReasonModal = true;
        } else {
            dispatch("status", { status: next });
        }
    }

    function handleLostSubmit() {
        if (!lostReason) return;
        dispatch("status", {
            status: "Lost",
            lostReason,
            lostDate: new Date().toISOString(),
        });
        showLostReasonModal = false;
        lostReason = "";
    }

    function sendEmail() {
        dispatch("email", { offerId: offer.id });
    }

    // Calculate days since created
    function getDaysSince(dateStr) {
        if (!dateStr) return 0;
        const now = new Date().getTime();
        const then = new Date(dateStr).getTime();
        const days = Math.floor((now - then) / (1000 * 60 * 60 * 24));
        return days;
    }

    let daysSinceCreated = $derived(getDaysSince(offer?.createdAt));
    let isStale = $derived(daysSinceCreated > 7 && offer?.status === "Pending");
    let hasRevisions = $derived(offer?.revisionNumber > 1);
</script>

<div class="card" class:stale={isStale}>
    <div class="header">
        <div class="title-block">
            <div class="status-row">
                <span
                    class="dot"
                    style={`background:${statusColors[offer?.status] || "#9ca3af"}`}
                ></span>
                <span class="mono">{offer?.status || "Pending"}</span>
                {#if hasRevisions}
                    <span class="revision-badge"
                        >Rev {offer.revisionNumber}</span
                    >
                {/if}
                {#if isStale}
                    <span class="stale-badge">{daysSinceCreated}d old</span>
                {/if}
            </div>
            <h3>{offer?.title || "Untitled Offer"}</h3>
            <p class="customer">{offer?.customer}</p>
        </div>
        <div class="actions">
            <button
                class="ghost"
                onclick={openPDF}
                disabled={!offer?.pdfPath}
                title="View PDF"
            >
                PDF
            </button>
            {#if offer?.status === "Pending" || offer?.status === "Sent"}
                <button
                    class="ghost"
                    onclick={sendEmail}
                    title="Send via email"
                >
                    Send
                </button>
                <button
                    class="ghost success"
                    onclick={() => changeStatus("Accepted")}
                >
                    Accept
                </button>
                <button
                    class="ghost danger"
                    onclick={() => changeStatus("Lost")}
                >
                    &times; Lost
                </button>
            {/if}
        </div>
    </div>

    <div class="meta">
        <span class="mono"
            >Total {formatNumber(offer?.totalValue || 0, 2)} BHD</span
        >
        <span class="mono"
            >Margin {offer?.marginPercent?.toFixed(1) || "0.0"}%</span
        >
        <span class="mono"
            >{offer?.createdAt
                ? new Date(offer.createdAt).toLocaleDateString()
                : "—"}</span
        >
    </div>

    {#if offer?.warning}
        <div class="warning">Warning: {offer.warning}</div>
    {/if}

    {#if offer?.lostReason}
        <div class="lost-reason">
            <strong>Lost Reason:</strong>
            {offer.lostReason}
            {#if offer?.lostDate}
                <span class="lost-date"
                    >({new Date(offer.lostDate).toLocaleDateString()})</span
                >
            {/if}
        </div>
    {/if}

    <!-- Follow-up Timeline Toggle -->
    {#if offer?.followUps && offer.followUps.length > 0}
        <button
            class="timeline-toggle"
            onclick={() => (showTimeline = !showTimeline)}
        >
            {showTimeline ? "Hide" : "Show"} Follow-ups ({offer.followUps.length})
        </button>

        {#if showTimeline}
            <div class="timeline" transition:slide={{ duration: 200 }}>
                {#each offer.followUps as followUp}
                    <div class="timeline-item">
                        <span class="timeline-dot"></span>
                        <div class="timeline-content">
                            <span class="timeline-date"
                                >{new Date(
                                    followUp.date,
                                ).toLocaleDateString()}</span
                            >
                            <span class="timeline-text">{followUp.note}</span>
                            {#if followUp.user}
                                <span class="timeline-user"
                                    >by {followUp.user}</span
                                >
                            {/if}
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    {/if}
</div>

<!-- Lost Reason Modal -->
{#if showLostReasonModal}
    <div
        class="modal-overlay"
        role="button"
        tabindex="0"
        onclick={() => (showLostReasonModal = false)}
        onkeydown={(event) =>
            (event.key === "Enter" || event.key === " ") &&
            (showLostReasonModal = false)}
    >
        <div class="modal" role="presentation" tabindex="-1" onclick={stopPropagation(bubble('click'))} onkeydown={stopPropagation(bubble('keydown'))}>
            <div class="modal-header">
                <h3>Why was this offer lost?</h3>
                <button
                    class="close-btn"
                    onclick={() => (showLostReasonModal = false)}>&times;</button
                >
            </div>
            <div class="modal-body">
                <div class="reason-options">
                    {#each lostReasons as reason}
                        <button
                            class="reason-btn"
                            class:selected={lostReason === reason}
                            onclick={() => (lostReason = reason)}
                        >
                            {reason}
                        </button>
                    {/each}
                </div>
                {#if lostReason === "Other"}
                    <textarea
                        bind:value={lostReason}
                        placeholder="Please specify the reason..."
                        rows="3"
                    ></textarea>
                {/if}
            </div>
            <div class="modal-footer">
                <button
                    class="cancel-btn"
                    onclick={() => (showLostReasonModal = false)}
                >
                    Cancel
                </button>
                <button
                    class="submit-btn"
                    onclick={handleLostSubmit}
                    disabled={!lostReason}
                >
                    Mark as Lost
                </button>
            </div>
        </div>
    </div>
{/if}

<style>
    .card {
        background: rgba(255, 255, 255, 0.7);
        border: 1px solid rgba(0, 0, 0, 0.08);
        border-radius: 8px;
        padding: 1rem 1.25rem;
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
        transition: all 0.2s ease;
    }

    .card:hover {
        background: rgba(255, 255, 255, 0.9);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
    }

    .card.stale {
        border-left: 3px solid #f59e0b;
        background: rgba(245, 158, 11, 0.05);
    }

    .header {
        display: flex;
        justify-content: space-between;
        gap: 1rem;
        align-items: flex-start;
    }

    .title-block h3 {
        margin: 0.25rem 0 0;
        font-family: Georgia, serif;
        font-size: 16px;
        font-weight: 500;
    }

    .customer {
        margin: 0.25rem 0 0;
        color: #57534e;
        font-size: 13px;
    }

    .status-row {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        font-family: "Courier Prime", monospace;
        letter-spacing: 1px;
        text-transform: uppercase;
        font-size: 10px;
        flex-wrap: wrap;
    }

    .dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
    }

    .revision-badge {
        background: rgba(59, 130, 246, 0.1);
        color: #3b82f6;
        padding: 2px 6px;
        border-radius: 3px;
        font-size: 9px;
    }

    .stale-badge {
        background: rgba(245, 158, 11, 0.1);
        color: #f59e0b;
        padding: 2px 6px;
        border-radius: 3px;
        font-size: 9px;
    }

    .actions {
        display: flex;
        gap: 0.5rem;
        flex-wrap: wrap;
    }

    .ghost {
        background: transparent;
        border: 1px solid rgba(0, 0, 0, 0.1);
        padding: 0.35rem 0.7rem;
        font-family: "Courier Prime", monospace;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        font-size: 10px;
        cursor: pointer;
        border-radius: 4px;
        transition: all 0.2s ease;
    }

    .ghost:hover:not(:disabled) {
        background: rgba(0, 0, 0, 0.05);
    }

    .ghost:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .ghost.success {
        border-color: #15803d;
        color: #15803d;
    }

    .ghost.success:hover {
        background: rgba(21, 128, 61, 0.1);
    }

    .ghost.danger {
        border-color: #ef4444;
        color: #ef4444;
    }

    .ghost.danger:hover {
        background: rgba(239, 68, 68, 0.1);
    }

    .meta {
        display: flex;
        gap: 1rem;
        font-family: "Courier Prime", monospace;
        color: #57534e;
        font-size: 11px;
        flex-wrap: wrap;
    }

    .warning {
        font-family: "Courier Prime", monospace;
        color: #ef4444;
        font-size: 12px;
        background: rgba(239, 68, 68, 0.08);
        padding: 0.5rem;
        border-radius: 4px;
    }

    .lost-reason {
        font-size: 12px;
        color: #9ca3af;
        background: rgba(0, 0, 0, 0.03);
        padding: 0.5rem;
        border-radius: 4px;
    }

    .lost-date {
        font-size: 10px;
        color: #9ca3af;
    }

    /* Timeline */
    .timeline-toggle {
        background: transparent;
        border: none;
        font-family: "Courier Prime", monospace;
        font-size: 11px;
        text-transform: uppercase;
        color: #3b82f6;
        cursor: pointer;
        padding: 0.25rem 0;
        text-align: left;
    }

    .timeline {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
        padding-left: 1rem;
        border-left: 2px solid rgba(0, 0, 0, 0.1);
    }

    .timeline-item {
        display: flex;
        gap: 0.75rem;
        align-items: flex-start;
    }

    .timeline-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: #3b82f6;
        margin-top: 0.25rem;
        flex-shrink: 0;
    }

    .timeline-content {
        display: flex;
        flex-direction: column;
        gap: 0.125rem;
    }

    .timeline-date {
        font-family: "Courier Prime", monospace;
        font-size: 10px;
        color: #9ca3af;
    }

    .timeline-text {
        font-size: 12px;
        color: #1c1c1c;
    }

    .timeline-user {
        font-size: 10px;
        color: #9ca3af;
        font-style: italic;
    }

    /* Modal */
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }

    .modal {
        background: #fdfbf7;
        border-radius: 8px;
        max-width: 500px;
        width: 90%;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
    }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 1.25rem;
        border-bottom: 1px solid rgba(0, 0, 0, 0.08);
    }

    .modal-header h3 {
        margin: 0;
        font-family: Georgia, serif;
        font-size: 18px;
        font-weight: normal;
    }

    .close-btn {
        background: none;
        border: none;
        font-size: 20px;
        cursor: pointer;
        color: #9ca3af;
    }

    .modal-body {
        padding: 1.25rem;
    }

    .reason-options {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 0.5rem;
        margin-bottom: 1rem;
    }

    .reason-btn {
        padding: 0.75rem;
        background: rgba(255, 255, 255, 0.6);
        border: 1px solid rgba(0, 0, 0, 0.1);
        border-radius: 6px;
        font-family: Georgia, serif;
        font-size: 13px;
        cursor: pointer;
        transition: all 0.2s ease;
    }

    .reason-btn:hover {
        background: rgba(255, 255, 255, 0.9);
    }

    .reason-btn.selected {
        background: #3b82f6;
        color: white;
        border-color: #3b82f6;
    }

    textarea {
        width: 100%;
        padding: 0.75rem;
        border: 1px solid rgba(0, 0, 0, 0.12);
        border-radius: 6px;
        font-family: Georgia, serif;
        font-size: 14px;
        resize: vertical;
    }

    .modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 0.75rem;
        padding: 1.25rem;
        border-top: 1px solid rgba(0, 0, 0, 0.08);
    }

    .cancel-btn {
        padding: 0.6rem 1.25rem;
        background: transparent;
        border: 1px solid rgba(0, 0, 0, 0.1);
        border-radius: 6px;
        font-family: "Courier Prime", monospace;
        font-size: 11px;
        text-transform: uppercase;
        cursor: pointer;
    }

    .submit-btn {
        padding: 0.6rem 1.25rem;
        background: #ef4444;
        color: white;
        border: none;
        border-radius: 6px;
        font-family: "Courier Prime", monospace;
        font-size: 11px;
        text-transform: uppercase;
        cursor: pointer;
    }

    .submit-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
</style>
