<script lang="ts">
    import ActionProposalCard from "../ui/ActionProposalCard.svelte";
    import EvidenceSourceList from "../ui/EvidenceSourceList.svelte";
    import KpiStatusStrip from "../ui/KpiStatusStrip.svelte";
    import type { ActionProposalItem, EvidenceSourceItem, KpiStatusItem } from "../ui";

    type InboxStats = {
        total_documents?: number;
        ready?: number;
        needs_review?: number;
        processed?: number;
        by_type?: Record<string, number>;
    };

    type IntakeReviewVM = {
        queueMetrics?: KpiStatusItem[];
        selected?: IntakeCandidateReviewVM | null;
    };

    type IntakeCandidateReviewVM = {
        sourceLabel?: string;
        sourceKind?: string;
        classification?: string;
        reviewStatus?: { label?: string };
        confidenceDisplay?: string;
        extractedFields?: Array<{ name?: string; label?: string; value?: string }>;
        sources?: EvidenceSourceItem[];
        sourceRegistry?: SourceRegistryItem[];
        actionProposals?: ActionProposalItem[];
        lastReview?: {
            decision?: string;
            reviewStatus?: string;
            actor?: string;
            reason?: string;
            createdAt?: string;
            proposedDeterministicService?: string;
        } | null;
        serviceTarget?: string;
    };

    type SourceRegistryItem = {
        sourceId?: string;
        kind?: string;
        label?: string;
        path?: string;
        privacyClass?: string;
        processingStatus?: string;
        candidateCount?: number;
        currentCandidate?: boolean;
        auditRefCount?: number;
        lastSeenAtDisplay?: string;
    };

    interface Props {
        document: any;
        stats: InboxStats;
        reviewQueue?: IntakeReviewVM | null;
        processing?: boolean;
        reviewBusy?: boolean;
        onProcess?: () => void;
        onArchive?: () => void;
        onReviewDecision?: (decision: string, proposal: ActionProposalItem) => void | Promise<void>;
        onContextPack?: () => void | Promise<void>;
    }

    let {
        document,
        stats,
        reviewQueue = null,
        processing = false,
        reviewBusy = false,
        onProcess,
        onArchive,
        onReviewDecision,
        onContextPack,
    }: Props = $props();

    let proposalReviews = $state<Record<string, string>>({});

    const reviewCandidate = $derived(reviewQueue?.selected ?? null);
    const reviewFields = $derived(reviewCandidate?.extractedFields || []);
    const extractedEntries = $derived((reviewFields.length > 0
        ? reviewFields.map((field) => [field.label || field.name || "Field", field.value || "Missing"])
        : Object.entries(document?.extracted_data || document?.entities || {}))
        .filter(([, value]) => String(value ?? "").trim() !== ""));

    const documentConfidence = $derived(Number(document?.confidence ?? document?.classification_confidence ?? 0));
    const documentType = $derived(reviewCandidate?.classification || document?.document_type || document?.detected_type || "Unclassified");
    const sourceLabel = $derived(reviewCandidate?.sourceLabel || document?.file_name || document?.filename || document?.document_id || "Inbox source");
    const reviewStatus = $derived(normalizeStatus(reviewCandidate?.reviewStatus?.label || document?.status || (documentConfidence < 0.8 ? "NeedsReview" : "Ready")));
    const proposalItems = $derived((reviewCandidate?.actionProposals?.length || 0) > 0 ? reviewCandidate?.actionProposals || [] : buildProposalItems(document));
    const sourceItems = $derived((reviewCandidate?.sources?.length || 0) > 0 ? reviewCandidate?.sources || [] : buildSourceItems(document, extractedEntries.length, documentConfidence));
    const sourceRegistryItems = $derived(reviewCandidate?.sourceRegistry || []);
    const lastReview = $derived(reviewCandidate?.lastReview ?? null);
    const serviceTarget = $derived(reviewCandidate?.serviceTarget || lastReview?.proposedDeterministicService || deterministicServiceFor(documentType));
    const fallbackQueueMetrics = $derived<KpiStatusItem[]>([
        {
            label: "Candidates",
            value: String(stats.total_documents ?? 0),
            meta: "in inbox",
            status: (stats.needs_review ?? 0) > 0 ? "review" : "ready",
        },
        {
            label: "Review",
            value: String(stats.needs_review ?? 0),
            meta: "operator queue",
            status: (stats.needs_review ?? 0) > 0 ? "review" : "ready",
        },
        {
            label: "Confidence",
            value: formatPercent(documentConfidence),
            meta: documentType,
            status: documentConfidence >= 0.8 ? "ready" : "review",
        },
        {
            label: "Fields",
            value: String(extractedEntries.length),
            meta: "extracted",
            status: extractedEntries.length > 0 ? "ready" : "review",
        },
    ]);
    const queueMetrics = $derived((reviewQueue?.queueMetrics?.length || 0) > 0 ? reviewQueue?.queueMetrics || [] : fallbackQueueMetrics);

    function normalizeStatus(value: string): string {
        return String(value || "")
            .trim()
            .toLowerCase()
            .replace(/[\s-]+/g, "_");
    }

    function formatPercent(value: number): string {
        if (!Number.isFinite(value) || value <= 0) return "not scored";
        return `${Math.round(Math.min(Math.max(value, 0), 1) * 100)}%`;
    }

    function sourceKind(doc: any): string {
        const path = String(doc?.file_path || doc?.path || doc?.file_name || "").toLowerCase();
        if (path.endsWith(".pdf")) return "pdf";
        if (path.endsWith(".xlsx") || path.endsWith(".xls") || path.endsWith(".csv")) return "excel";
        if (path.endsWith(".png") || path.endsWith(".jpg") || path.endsWith(".jpeg")) return "screenshot";
        if (path.endsWith(".msg") || path.endsWith(".eml")) return "email";
        return "inbox_record";
    }

    function buildSourceItems(doc: any, presentFields: number, confidence: number): EvidenceSourceItem[] {
        const required = Math.max(presentFields, 1);
        const status = presentFields > 0 && confidence >= 0.8 ? "ready" : "review";
        return [
            {
                source_type: sourceKind(doc),
                label: sourceLabel,
                required,
                present: presentFields,
                missing: Math.max(required - presentFields, presentFields > 0 ? 0 : 1),
                confidence,
                status,
                priority: status === "ready" ? "low" : "medium",
                last_updated: doc?.processed_at || doc?.created_at || "",
            },
        ];
    }

    function buildProposalItems(doc: any): ActionProposalItem[] {
        const actions = Array.isArray(doc?.suggested_actions) ? doc.suggested_actions : [];
        if (actions.length === 0) {
            return [{
                action: "review_candidate",
                source_type: sourceKind(doc),
                label: "Review candidate before linking",
                reason: "No deterministic link should be created until extracted fields and provenance are confirmed.",
                priority: reviewStatus === "needs_review" ? "high" : "medium",
                required_deterministic_service: deterministicServiceFor(documentType),
            }];
        }

        return actions.map((action: string, index: number) => ({
            action: `review_proposal_${index + 1}`,
            source_type: sourceKind(doc),
            label: action,
            reason: "Operator review records intent; deterministic services still own business changes.",
            priority: reviewStatus === "needs_review" ? "high" : "medium",
            required_deterministic_service: deterministicServiceFor(documentType),
        }));
    }

    function deterministicServiceFor(type: string): string {
        const normalized = normalizeStatus(type);
        if (normalized.includes("invoice") || normalized.includes("bank")) return "finance.review_link";
        if (normalized.includes("rfq") || normalized.includes("quotation")) return "crm.review_link";
        if (normalized.includes("purchase") || normalized.includes("delivery") || normalized.includes("order")) return "operations.review_link";
        return "documents.review_link";
    }

    function proposalKey(proposal: ActionProposalItem): string {
        return [
            proposal.action || "",
            proposal.source_type || "",
            proposal.required_deterministic_service || "",
            proposal.label,
        ].join("|").toLowerCase();
    }

    function reviewLabel(proposal: ActionProposalItem): string {
        if (lastReview?.decision) return lastReview.decision.replace(/_/g, " ");
        return proposalReviews[proposalKey(proposal)] || proposal.required_deterministic_service || "review required";
    }

    async function recordProposalChoice(proposal: ActionProposalItem, status: string) {
        proposalReviews = {
            ...proposalReviews,
            [proposalKey(proposal)]: status,
        };
        await onReviewDecision?.(status, proposal);
    }
</script>

<section class="review-panel">
    <div class="review-head">
        <div>
            <span>{reviewCandidate?.sourceKind || sourceKind(document)}</span>
            <h2>{sourceLabel}</h2>
        </div>
        <strong data-status={reviewStatus}>{lastReview?.reviewStatus || reviewCandidate?.reviewStatus?.label || document?.status || "New"}</strong>
    </div>

    <KpiStatusStrip items={queueMetrics} />

    <EvidenceSourceList sources={sourceItems} />

    {#if sourceRegistryItems.length > 0}
        <div class="source-registry">
            {#each sourceRegistryItems as source}
                <div class="source-registry-row">
                    <div>
                        <span>{source.kind || "source"}</span>
                        <strong>{source.label || source.sourceId || "Tracked source"}</strong>
                        {#if source.path}
                            <small>{source.path}</small>
                        {/if}
                    </div>
                    <dl>
                        <div>
                            <dt>Source</dt>
                            <dd>{source.sourceId || "untracked"}</dd>
                        </div>
                        <div>
                            <dt>Privacy</dt>
                            <dd>{source.privacyClass || "internal"}</dd>
                        </div>
                        <div>
                            <dt>Status</dt>
                            <dd>{source.processingStatus || "discovered"}</dd>
                        </div>
                        <div>
                            <dt>Candidates</dt>
                            <dd>{source.candidateCount ?? (source.currentCandidate ? 1 : 0)}</dd>
                        </div>
                        <div>
                            <dt>Audit refs</dt>
                            <dd>{source.auditRefCount ?? 0}</dd>
                        </div>
                    </dl>
                </div>
            {/each}
        </div>
    {/if}

    <div class="field-section">
        <div class="section-head">
            <span>{documentType}</span>
            <strong>{formatPercent(documentConfidence)}</strong>
        </div>

        {#if extractedEntries.length === 0}
            <div class="empty-fields">
                <span>No extracted fields</span>
                <small>{document?.extracted_text ? document.extracted_text.slice(0, 260) : "Awaiting analysis"}</small>
            </div>
        {:else}
            <div class="field-grid">
                {#each extractedEntries as [name, value]}
                    <div class="field-row">
                        <span>{name}</span>
                        <strong>{String(value)}</strong>
                    </div>
                {/each}
            </div>
        {/if}
    </div>

    {#if lastReview}
        <div class="last-review">
            <span>{lastReview.actor || "operator"} / {lastReview.createdAt || "recorded"}</span>
            <strong>{lastReview.decision?.replace(/_/g, " ") || "review recorded"}</strong>
            {#if lastReview.reason}
                <small>{lastReview.reason}</small>
            {/if}
            <small>{serviceTarget}</small>
        </div>
    {/if}

    <div class="proposal-list">
        {#each proposalItems as proposal}
            <ActionProposalCard
                {proposal}
                reviewLabel={reviewLabel(proposal)}
                hasReview
                onApprove={() => recordProposalChoice(proposal, "accept_proposal")}
                onNeedsInput={() => recordProposalChoice(proposal, "needs_input")}
                onReject={() => recordProposalChoice(proposal, "reject_candidate")}
            />
        {/each}
    </div>

    <div class="review-actions">
        <button type="button" class="primary" disabled={processing} onclick={() => onProcess?.()}>
            {processing ? "Analyzing" : "Analyze"}
        </button>
        <button type="button" disabled={reviewBusy || proposalItems.length === 0} onclick={() => recordProposalChoice(proposalItems[0], "needs_input")}>
            Needs input
        </button>
        <button type="button" disabled={reviewBusy || proposalItems.length === 0} onclick={() => recordProposalChoice(proposalItems[0], "correct_field")}>
            Correct field
        </button>
        <button type="button" disabled={reviewBusy || proposalItems.length === 0} onclick={() => recordProposalChoice(proposalItems[0], "archive")}>
            Archive review
        </button>
        <button type="button" disabled={reviewBusy} onclick={() => onContextPack?.()}>
            Context pack
        </button>
        <button type="button" onclick={() => onArchive?.()}>
            Archive document
        </button>
    </div>
</section>

<style>
    .review-panel {
        min-width: 0;
        display: grid;
        gap: 1px;
        background: var(--border-subtle, #e5e1d8);
        border: 1px solid var(--border-subtle, #e5e1d8);
    }

    .review-head,
    .field-section,
    .last-review,
    .review-actions {
        min-width: 0;
        background: var(--paper, #fffefa);
    }

    .review-head {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;
        padding: 16px;
    }

    .review-head div {
        min-width: 0;
        display: grid;
        gap: 4px;
    }

    .review-head span,
    .section-head span,
    .field-row span,
    .empty-fields span {
        color: var(--ink-light, #666);
        font-size: 11px;
        text-transform: uppercase;
    }

    h2 {
        min-width: 0;
        margin: 0;
        color: var(--ink, #1c1c1c);
        font-size: 18px;
        font-weight: 600;
        overflow-wrap: anywhere;
    }

    .review-head > strong {
        flex: 0 0 auto;
        border: 1px solid var(--border-subtle, #e5e1d8);
        padding: 5px 8px;
        color: var(--ink, #1c1c1c);
        font-size: 11px;
        text-transform: uppercase;
    }

    .review-head > strong[data-status="needsreview"],
    .review-head > strong[data-status="needs_review"] {
        border-color: #b45309;
        color: #92400e;
    }

    .field-section {
        display: grid;
        gap: 12px;
        padding: 16px;
    }

    .source-registry {
        display: grid;
        gap: 1px;
        background: var(--border-subtle, #e5e1d8);
    }

    .source-registry-row {
        min-width: 0;
        display: grid;
        grid-template-columns: minmax(180px, 0.7fr) minmax(0, 1.3fr);
        gap: 12px;
        background: var(--paper, #fffefa);
        padding: 12px 16px;
    }

    .source-registry-row > div {
        min-width: 0;
        display: grid;
        gap: 4px;
    }

    .source-registry-row span,
    .source-registry-row dt {
        color: var(--ink-light, #666);
        font-size: 10px;
        text-transform: uppercase;
    }

    .source-registry-row strong,
    .source-registry-row small,
    .source-registry-row dd {
        min-width: 0;
        margin: 0;
        overflow-wrap: anywhere;
    }

    .source-registry-row strong {
        color: var(--ink, #1c1c1c);
        font-size: 13px;
    }

    .source-registry-row small {
        color: var(--ink-light, #666);
        font-size: 11px;
    }

    .source-registry-row dl {
        min-width: 0;
        display: grid;
        grid-template-columns: repeat(5, minmax(0, 1fr));
        gap: 8px;
        margin: 0;
    }

    .source-registry-row dl div {
        min-width: 0;
        display: grid;
        gap: 3px;
    }

    .source-registry-row dd {
        color: var(--ink, #1c1c1c);
        font-size: 11px;
    }

    .section-head {
        min-width: 0;
        display: flex;
        justify-content: space-between;
        gap: 12px;
    }

    .section-head strong {
        font-family: var(--font-mono, ui-monospace, SFMono-Regular, Consolas, monospace);
        font-size: 13px;
    }

    .field-grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 1px;
        background: var(--border-subtle, #e5e1d8);
    }

    .field-row,
    .empty-fields {
        min-width: 0;
        display: grid;
        gap: 4px;
        background: var(--paper-soft, #fafafa);
        padding: 10px 12px;
    }

    .field-row strong,
    .empty-fields small {
        min-width: 0;
        color: var(--ink, #1c1c1c);
        font-size: 12px;
        overflow-wrap: anywhere;
    }

    .empty-fields small {
        color: var(--ink-light, #666);
        line-height: 1.5;
    }

    .proposal-list {
        display: grid;
        gap: 1px;
        background: var(--border-subtle, #e5e1d8);
    }

    .last-review {
        display: grid;
        gap: 4px;
        padding: 12px 16px;
        border-left: 3px solid #2f6f5e;
    }

    .last-review span,
    .last-review small {
        min-width: 0;
        color: var(--ink-light, #666);
        font-size: 11px;
        overflow-wrap: anywhere;
    }

    .last-review strong {
        min-width: 0;
        color: var(--ink, #1c1c1c);
        font-size: 13px;
        text-transform: capitalize;
        overflow-wrap: anywhere;
    }

    .review-actions {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
        flex-wrap: wrap;
        padding: 12px 16px;
    }

    button {
        min-height: 34px;
        border: 1px solid var(--border-subtle, #e5e1d8);
        background: var(--paper-soft, #fafafa);
        color: var(--ink, #1c1c1c);
        padding: 7px 12px;
        font-size: 12px;
        cursor: pointer;
    }

    button.primary {
        border-color: var(--ink, #1c1c1c);
        background: var(--ink, #1c1c1c);
        color: var(--paper, #fffefa);
    }

    button:disabled {
        cursor: progress;
        opacity: 0.65;
    }

    @media (max-width: 760px) {
        .review-head,
        .section-head {
            display: grid;
        }

        .field-grid {
            grid-template-columns: 1fr;
        }

        .source-registry-row,
        .source-registry-row dl {
            grid-template-columns: 1fr;
        }

        .review-actions {
            justify-content: flex-start;
        }
    }
</style>
