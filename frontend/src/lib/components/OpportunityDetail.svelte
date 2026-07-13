<script lang="ts">
    import { run } from 'svelte/legacy';

    import { createEventDispatcher, onMount } from "svelte";
    import {
        AddOpportunityComment } from "../../../wailsjs/go/main/App";
import { AddRFQComment, DeleteOpportunityComment, GetOpportunityComments, GetOpportunityLineItems, GetPurchaseOrdersByOrder, GetRFQ, GetRFQComments, GetRFQTraceability, UpdateOpportunityStageWithVersion, UpdateRFQStage } from "../../../wailsjs/go/main/CRMService";
import { GetCurrentUserRole } from "../../../wailsjs/go/main/InfraService";
    import { devLog } from "$lib/utils/devLog";
    import { toast } from "$lib/stores/toasts";
    import { permissions } from "$lib/stores/authContext";

    interface LineItem {
        description?: string;
        quantity?: number;
        unit_price?: number;
        total_price?: number;
        part_number?: string;
        unit?: string;
        currency?: string;
    }

    type HistoryEntry = {
        status?: string;
        note?: string;
        createdAt?: string | number | Date | null;
    };

    interface PurchaseOrder {
        id: string;
        supplier_name: string;
        po_number: string;
        po_date: any;
        expected_delivery: any;
        total_bhd: number;
        status: string;
    }

    interface Props {
        opportunity?: any;
        history?: HistoryEntry[];
        quickCaptures?: any[];
        loading?: boolean;
        error?: string;
    }

    let {
        opportunity = $bindable(null),
        history = [],
        quickCaptures = [],
        loading = false,
        error = ""
    }: Props = $props();
    run(() => {
        quickCaptures;
    });

    const dispatch = createEventDispatcher();

    const rfqStageOptions = [
        "RFQ Received",
        "Offer Sent",
        "Follow-up/Eval",
        "PO/LOI Received",
        "Order Placed",
        "In Process",
        "Delivered",
        "Closed (Payment)",
        "Closed (Lost)",
    ];

    const pipelineStageOptions = ["Quoted", "Won", "Lost"];
    const deprecatedPipelineStageLabels = new Set(["New", "Qualified", "Proposal", "On Hold"]);
    const managementRoles = new Set(["admin", "administrator", "manager", "management", "developer"]);
    const adminRoles = new Set(["admin", "administrator", "developer"]);

    let activeTab: "details" | "purchase-orders" = $state("details");
    let purchaseOrders: PurchaseOrder[] = $state([]);
    let loadingPOs = $state(false);
    let poError = $state("");
    let orderId = $state("");

    let comments: any[] = $state([]);
    let commentsError = $state("");
    let newComment = $state("");
    let addingComment = $state(false);
    let deletingCommentId = $state("");
    let updatingStage = $state(false);
    let currentUserRole = $state("");
    let lastLoadedOpportunityKey = $state("");
    let lastLoadedTabKey = $state("");
    let offerLineItems: any[] = $state([]);

    function parseProductDetails(productDetails?: string): LineItem[] {
        if (!productDetails) return [];
        try {
            const parsed = JSON.parse(productDetails);
            if (Array.isArray(parsed)) return parsed;
            if (typeof parsed === "object" && parsed) return [parsed];
        } catch {
            return [];
        }
        return [];
    }

    let lineItems = $derived(offerLineItems.length > 0
        ? offerLineItems
        : parseProductDetails(opportunity?.product_details));
    let isPipelineOpportunity = $derived(opportunity?._source === "pipeline" || typeof opportunity?.id === "string");
    let currentStageOptions = $derived(isPipelineOpportunity ? pipelineStageOptions : rfqStageOptions);
    let currentStageValue = $derived(isPipelineOpportunity && deprecatedPipelineStageLabels.has(opportunity?.stage || opportunity?.status)
        ? "Pipeline"
        : (opportunity?.stage || opportunity?.status || currentStageOptions[0]));
    let canViewManagementComments = $derived(managementRoles.has((currentUserRole || "").toLowerCase()));
    let canDeleteManagementComments = $derived(adminRoles.has((currentUserRole || "").toLowerCase()));
    let activePermissions = $derived(Array.isArray($permissions) ? $permissions : []);
    function hasPermission(permission: string, roleValue = currentUserRole, permissionValues = activePermissions): boolean {
        const role = (roleValue || "").toLowerCase();
        if (adminRoles.has(role)) return true;
        if (managementRoles.has(role) && permission !== "comments:delete") return true;
        const permissionList = Array.isArray(permissionValues) ? permissionValues : [];
        if (permissionList.includes("*") || permissionList.includes(permission)) return true;
        const [resource] = permission.split(":");
        return permissionList.includes(`${resource}:*`);
    }
    let canEditOffers = $derived(hasPermission("offers:edit", currentUserRole, activePermissions));
    let canCreateOffers = $derived(hasPermission("offers:create", currentUserRole, activePermissions));

    const formatter = new Intl.NumberFormat("en-BH", {
        style: "currency",
        currency: "BHD",
        maximumFractionDigits: 0,
    });

    function formatLineMoney(item: LineItem, amount: number): string {
        const currency = isCurrencyCode(item.currency) ? String(item.currency).trim().toUpperCase() : "BHD";
        return new Intl.NumberFormat("en-BH", {
            style: "currency",
            currency,
            maximumFractionDigits: 0,
        }).format(amount || 0);
    }

    function isCurrencyCode(value?: string): boolean {
        return /^(BHD|USD|EUR|GBP|SAR|AED|CHF|KWD|OMR|QAR)$/i.test(String(value || "").trim());
    }

    function normalizeLineUnit(value?: string): string {
        const unit = String(value || "").trim();
        return isCurrencyCode(unit) ? "" : unit;
    }

    function formatQuantity(item: LineItem): string {
        const quantity = item.quantity || 1;
        const unit = normalizeLineUnit(item.unit);
        return unit ? `${quantity} ${unit}` : `${quantity}`;
    }

    async function loadUserRole() {
        try {
            currentUserRole = (await GetCurrentUserRole()) || "";
        } catch (err) {
            devLog.error("Failed to load current user role:", err);
            currentUserRole = "";
        }
    }

    async function loadPurchaseOrders() {
        if (!opportunity?.id || isPipelineOpportunity) {
            purchaseOrders = [];
            return;
        }

        loadingPOs = true;
        poError = "";

        try {
            const traceability = await GetRFQTraceability(opportunity.id.toString());
            if (traceability && traceability.order_id) {
                orderId = traceability.order_id;
                purchaseOrders = (await GetPurchaseOrdersByOrder(orderId)) || [];
            } else {
                orderId = "";
                purchaseOrders = [];
            }
        } catch (err) {
            devLog.error("Failed to load purchase orders:", err);
            poError = "Failed to load purchase orders";
            purchaseOrders = [];
        } finally {
            loadingPOs = false;
        }
    }

    async function refreshOpportunityData() {
        if (!opportunity?.id || isPipelineOpportunity) return;
        try {
            const fresh = await GetRFQ(Number(opportunity.id));
            if (fresh) {
                opportunity = { ...opportunity, ...fresh };
            }
        } catch (err) {
            devLog.error("Failed to refresh opportunity:", err);
        }
    }

    async function loadOpportunityLineItems() {
        offerLineItems = [];
        if (!opportunity?.id || !isPipelineOpportunity) return;
        try {
            const items = (await GetOpportunityLineItems(String(opportunity.id))) || [];
            offerLineItems = items
                .filter((item: any) => {
                    const raw = `${item.description || ""} ${item.equipment || ""} ${item.model || ""}`.toLowerCase();
                    return !raw.includes("total for order");
                })
                .map((item: any) => ({
                    description: item.description || item.equipment || item.model || item.product_code || "Line item",
                    quantity: item.quantity || 1,
                    unit_price: item.unit_price_bhd || item.unit_price || item.total_price || 0,
                    total_price: item.total_price || ((item.quantity || 1) * (item.unit_price_bhd || item.unit_price || 0)),
                    part_number: item.model || item.product_code || "",
                    unit: normalizeLineUnit(item.unit || item.unit_of_measure || ""),
                    currency: item.currency || "",
                    specification: item.specification || item.detailed_description || "",
                }));
        } catch (err) {
            devLog.error("Failed to load opportunity line items:", err);
            offerLineItems = [];
        }
    }

    async function loadComments() {
        comments = [];
        commentsError = "";
        if (!opportunity?.id) return;

        try {
            let loadedComments: any[] = [];
            if (isPipelineOpportunity) {
                if (!canViewManagementComments) return;
                loadedComments = (await GetOpportunityComments(String(opportunity.id))) || [];
                comments = mergeLegacyNotesIntoComments(loadedComments);
                return;
            }
            loadedComments = (await GetRFQComments(Number(opportunity.id))) || [];
            comments = mergeLegacyNotesIntoComments(loadedComments);
        } catch (err) {
            devLog.error("Failed to load comments:", err);
            comments = [];
            // Distinct from "no comments yet" - a load failure must be visibly
            // different so the user knows to retry rather than assume it's empty.
            commentsError = err?.message ? String(err.message) : "Couldn't load comments.";
        }
    }

    function mergeLegacyNotesIntoComments(loadedComments: any[]) {
        const merged = [...loadedComments];
        const legacyEntries = [
            opportunity?.notes || opportunity?.comment,
        ]
            .map((value) => String(value || "").trim())
            .filter(Boolean);

        for (const legacyComment of legacyEntries) {
            const exists = merged.some((comment) => String(comment.comment || comment.content || "").trim() === legacyComment);
            if (!exists) {
                merged.unshift({
                    id: `legacy-${merged.length}`,
                    comment: legacyComment,
                    created_by: "Legacy note",
                    created_at: opportunity?.updated_at || opportunity?.created_at || null,
                });
            }
        }

        return merged;
    }

    async function handleAddComment() {
        if (!newComment.trim() || !opportunity?.id) return;

        addingComment = true;
        try {
            if (isPipelineOpportunity) {
                await AddOpportunityComment(String(opportunity.id), newComment.trim());
            } else {
                await AddRFQComment(Number(opportunity.id), newComment.trim(), "");
            }
            newComment = "";
            await loadComments();
            toast.success("Comment added");
        } catch (err) {
            devLog.error("Failed to add comment:", err);
            toast.danger("Failed to add comment");
        } finally {
            addingComment = false;
        }
    }

    function canDeleteComment(comment: any) {
        const id = String(comment?.id || "");
        return isPipelineOpportunity && canDeleteManagementComments && id && !id.startsWith("legacy-");
    }

    async function handleDeleteComment(comment: any) {
        const commentId = String(comment?.id || "");
        if (!canDeleteComment(comment)) return;

        deletingCommentId = commentId;
        try {
            await DeleteOpportunityComment(commentId);
            await loadComments();
            toast.success("Comment deleted");
        } catch (err) {
            devLog.error("Failed to delete comment:", err);
            toast.danger(`Failed to delete comment: ${err?.message || err}`);
        } finally {
            deletingCommentId = "";
        }
    }

    async function handleStageChange(event: Event) {
        const newStage = (event.target as HTMLSelectElement).value;
        if (!opportunity?.id || !newStage) return;

        updatingStage = true;
        try {
            if (isPipelineOpportunity) {
                const updated = await UpdateOpportunityStageWithVersion(String(opportunity.id), newStage, Number(opportunity.version || 0));
                opportunity = { ...opportunity, ...updated, stage: updated?.stage || newStage, status: updated?.stage || newStage };
            } else {
                await UpdateRFQStage(Number(opportunity.id), newStage);
                opportunity = { ...opportunity, stage: newStage, status: newStage };
            }
            dispatch("updated");
            toast.success(`Stage updated to ${newStage}`);
        } catch (err) {
            devLog.error("Failed to update stage:", err);
            const message = String(err?.message || err || "Failed to update stage");
            if (message.toLowerCase().includes("conflict")) {
                // Wave 10 B6 (Article IV.4): this is the failure echo of the user's OWN
                // stage-change click (not an unbidden background announce), so it stays as
                // a toast — reworded to lead with the action that failed. The conflict was
                // flagged for admin review server-side; that logic is unchanged.
                toast.danger("Couldn't update stage — this opportunity changed on another device. It's been flagged for review.");
            } else {
                toast.danger("Failed to update stage");
            }
        } finally {
            updatingStage = false;
        }
    }

    function handleCreateCosting() {
        if (!opportunity?.id) return;

        const pendingPayload = {
            id: String(opportunity.id),
            customer_name: opportunity.client || opportunity.customer || "",
            folder_number: opportunity.folder_number || opportunity.rfq_number || "",
            title: opportunity.project || opportunity.project_name || opportunity.title || "",
        };
        sessionStorage.setItem("asymmflow.pendingCostingOpportunity", JSON.stringify(pendingPayload));

        dispatch("createCosting", { opportunity, pendingPayload });
        window.dispatchEvent(new CustomEvent("navigateToScreen", {
            detail: { screen: "opportunities", tab: "costing" }
        }));
    }

    function formatDate(value: any) {
        if (!value) return "Not set";
        const parsed = new Date(value);
        if (Number.isNaN(parsed.getTime())) return "Not set";
        return parsed.toLocaleDateString();
    }

    function formatDateTime(value: any) {
        if (!value) return "";
        const parsed = new Date(value);
        if (Number.isNaN(parsed.getTime())) return "";
        return parsed.toLocaleString();
    }

    function infoRows() {
        if (!opportunity) return [];
        return [
            { label: "Reference", value: opportunity.folder_number || opportunity.rfq_number || "Manual" },
            { label: "Year", value: opportunity.year || formatDate(opportunity.created_at).split("/").pop() },
            { label: "Source", value: opportunity._source === "rfq" ? "RFQ" : (opportunity.source || "Pipeline") },
            { label: "Expected Close", value: formatDate(opportunity.expected_date) },
            { label: "Payment Terms", value: opportunity.payment_terms || "Not captured" },
            { label: "Delivery Terms", value: opportunity.delivery_terms || "Not captured" },
        ];
    }

    let opportunityKey = $derived(opportunity?.id ? `${opportunity.id}:${opportunity._source || ""}` : "");
    run(() => {
        if (opportunityKey && opportunityKey !== lastLoadedOpportunityKey) {
            lastLoadedOpportunityKey = opportunityKey;
            lastLoadedTabKey = "";
            activeTab = "details";
            purchaseOrders = [];
            void loadComments();
            void loadOpportunityLineItems();
            void refreshOpportunityData();
        }
    });

    run(() => {
        if (activeTab === "purchase-orders" && opportunityKey && !isPipelineOpportunity) {
            const tabKey = `purchase-orders:${opportunityKey}`;
            if (tabKey !== lastLoadedTabKey) {
                lastLoadedTabKey = tabKey;
                void loadPurchaseOrders();
            }
        }
    });

    onMount(async () => {
        await loadUserRole();
        await loadComments();
    });
</script>

<div class="detail">
    {#if loading}
        <div class="skeleton title"></div>
        <div class="skeleton line"></div>
        <div class="skeleton line short"></div>
    {:else if error}
        <p class="error">{error}</p>
    {:else if opportunity}
        <header class="header">
            <div class="header-copy">
                <div class="header-meta">
                    <span class="reference-pill">{opportunity.folder_number || opportunity.rfq_number || "Opportunity"}</span>
                    <select
                        class="stage-select"
                        value={currentStageValue}
                        onchange={handleStageChange}
                        disabled={updatingStage || !canEditOffers}
                    >
                        {#if currentStageValue === "Pipeline"}
                            <option value="Pipeline" disabled>Pipeline</option>
                        {/if}
                        {#each currentStageOptions as stage}
                            <option value={stage}>{stage}</option>
                        {/each}
                    </select>
                </div>
                <h2>{opportunity.project || opportunity.title || "Untitled"}</h2>
                <p class="subtitle">{opportunity.client || opportunity.customer || "Unknown Customer"}</p>
            </div>
            <div class="header-actions">
                {#if canCreateOffers}
                    <button class="comment-btn" onclick={handleCreateCosting}>
                        Create Costing Sheet
                    </button>
                {/if}
                <div class="value">{formatter.format(opportunity.value || opportunity.revenue_bhd || 0)}</div>
            </div>
        </header>

        <section class="info-grid">
            {#each infoRows() as row}
                <div class="info-card">
                    <span class="mono">{row.label}</span>
                    <strong>{row.value}</strong>
                </div>
            {/each}
            {#if isPipelineOpportunity}
                <div class="info-card">
                    <span class="mono">Edit Version</span>
                    <strong>v{opportunity.version || 1}</strong>
                </div>
            {/if}
        </section>

        <section class="comments-section">
            <div class="comments-header">
                <div>
                    <p class="mono">Comments</p>
                    {#if isPipelineOpportunity}
                        <p class="comments-hint">Shared comment thread with timestamps.</p>
                    {/if}
                </div>
            </div>

            {#if isPipelineOpportunity && !canViewManagementComments}
                <p class="muted">Comments are visible to authorized roles.</p>
            {:else}
                <div class="comment-list">
                    {#if commentsError}
                        <div class="comments-error">
                            <p class="muted">Couldn't load comments — {commentsError}</p>
                            <button class="comment-btn" onclick={() => loadComments()}>Retry</button>
                        </div>
                    {:else if comments.length === 0}
                        <p class="muted">No comments yet.</p>
                    {:else}
                        {#each comments as comment}
                            <div class="comment-item">
                                <div class="comment-header">
                                    <span class="comment-author">{comment.created_by || "User"}</span>
                                    <div class="comment-meta">
                                        <span class="comment-time">{formatDateTime(comment.created_at)}</span>
                                        {#if canDeleteComment(comment)}
                                            <button
                                                class="comment-delete"
                                                onclick={() => handleDeleteComment(comment)}
                                                disabled={deletingCommentId === String(comment.id)}
                                                title="Delete comment"
                                            >
                                                {deletingCommentId === String(comment.id) ? "Deleting..." : "Delete"}
                                            </button>
                                        {/if}
                                    </div>
                                </div>
                                <p class="comment-text">{comment.comment || comment.content}</p>
                            </div>
                        {/each}
                    {/if}
                </div>

                {#if (!isPipelineOpportunity && canEditOffers) || (isPipelineOpportunity && canViewManagementComments)}
                    <div class="comment-input">
                        <textarea
                            bind:value={newComment}
                            placeholder="Add a comment..."
                            class="comment-textarea"
                            rows="3"
></textarea>
                        <button class="comment-btn" onclick={handleAddComment} disabled={!newComment.trim() || addingComment}>
                            {addingComment ? "Posting..." : "Post"}
                        </button>
                    </div>
                {/if}
            {/if}
        </section>

        <div class="tabs-nav">
            <button class="tab-btn" class:active={activeTab === "details"} onclick={() => (activeTab = "details")}>
                Details
            </button>
            {#if !isPipelineOpportunity}
                <button class="tab-btn" class:active={activeTab === "purchase-orders"} onclick={() => (activeTab = "purchase-orders")}>
                    Purchase Orders
                    {#if purchaseOrders.length > 0}
                        <span class="tab-count">{purchaseOrders.length}</span>
                    {/if}
                </button>
            {/if}
        </div>

        {#if activeTab === "details"}
            {#if lineItems.length > 0}
                <section class="line-items-section">
                    <p class="mono">Line Items ({lineItems.length})</p>
                    <div class="items-table">
                        <div class="items-header">
                            <span class="item-col-desc">Description</span>
                            <span class="item-col-qty">Qty</span>
                            <span class="item-col-price">Unit Price</span>
                            <span class="item-col-total">Total</span>
                        </div>
                        {#each lineItems as item, index}
                            <div class="items-row">
                                <span class="item-col-desc">
                                    {item.description || `Item ${index + 1}`}
                                    {#if item.part_number}
                                        <span class="part-number">{item.part_number}</span>
                                    {/if}
                                </span>
                                <span class="item-col-qty">{formatQuantity(item)}</span>
                                <span class="item-col-price">{formatLineMoney(item, item.unit_price || 0)}</span>
                                <span class="item-col-total">{formatLineMoney(item, item.total_price || 0)}</span>
                            </div>
                        {/each}
                    </div>
                </section>
            {:else}
                <p class="muted">No structured line items captured for this opportunity yet.</p>
            {/if}
        {:else}
            <div class="po-tab-content">
                {#if loadingPOs}
                    <div class="skeleton line"></div>
                    <div class="skeleton line"></div>
                    <div class="skeleton line short"></div>
                {:else if poError}
                    <p class="error">{poError}</p>
                {:else if purchaseOrders.length === 0}
                    <p class="muted">No purchase orders have been created for this opportunity yet.</p>
                {:else}
                    <div class="po-list">
                        {#each purchaseOrders as po}
                            <div class="po-card">
                                <div class="po-header">
                                    <div>
                                        <strong>{po.po_number}</strong>
                                        <p class="muted">{po.supplier_name || "Unknown Supplier"}</p>
                                    </div>
                                    <div class="po-amount">{formatter.format(po.total_bhd || 0)}</div>
                                </div>
                                <div class="po-meta">
                                    <span>{po.status}</span>
                                    <span>PO {formatDate(po.po_date)}</span>
                                    <span>Expected {formatDate(po.expected_delivery)}</span>
                                </div>
                            </div>
                        {/each}
                    </div>
                {/if}
                {#if orderId}
                    <p class="mono">Order ID: {orderId}</p>
                {/if}
            </div>
        {/if}

        {#if history.length > 0}
            <section class="timeline">
                <p class="mono">Timeline</p>
                {#each history as event}
                    <div class="event">
                        <div class="event-head">
                            <strong>{event.status || "Update"}</strong>
                            <span class="event-time">{formatDateTime(event.createdAt)}</span>
                        </div>
                        {#if event.note}
                            <p class="note">{event.note}</p>
                        {/if}
                    </div>
                {/each}
            </section>
        {/if}

    {:else}
        <p class="muted">Select an opportunity to view details.</p>
    {/if}
</div>

<style>
    .detail {
        background: rgba(255, 255, 255, 0.96);
        border: 1px solid rgba(15, 23, 42, 0.08);
        border-radius: 20px;
        padding: 1.5rem;
    }

    .header,
    .header-meta,
    .po-header,
    .comment-header,
    .comments-header,
    .comment-input,
    .event-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.75rem;
    }

    .header {
        align-items: flex-start;
        margin-bottom: 1rem;
        padding-bottom: 1rem;
        border-bottom: 1px solid rgba(15, 23, 42, 0.08);
    }

    .header-copy {
        flex: 1;
        min-width: 0;
    }

    .header-actions {
        display: flex;
        align-items: center;
        gap: 0.75rem;
    }

    .reference-pill,
    .value,
    .tab-count,
    .comment-btn {
        border-radius: 999px;
        padding: 0.45rem 0.8rem;
    }

    .reference-pill {
        background: rgba(15, 23, 42, 0.08);
        font-size: 0.75rem;
        text-transform: uppercase;
        letter-spacing: 0.06em;
    }

    .stage-select,
    .comment-textarea,
    .comment-btn {
        border: 1px solid rgba(15, 23, 42, 0.12);
        border-radius: 12px;
    }

    .stage-select,
    .comment-textarea {
        padding: 0.7rem 0.8rem;
        background: white;
    }

    h2 {
        margin: 0.45rem 0 0.15rem;
        font-size: 1.5rem;
    }

    .subtitle,
    .muted,
    .comments-hint,
    .event-time,
    .comment-time,
    .po-meta {
        color: var(--text-muted);
    }

    .comments-error {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        flex-wrap: wrap;
    }

    .comments-error .muted {
        margin: 0;
    }

    .value {
        background: var(--color-ink);
        color: var(--color-paper);
        font-weight: 800;
        white-space: nowrap;
    }

    .mono {
        margin: 0 0 0.25rem;
        font-size: 0.72rem;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--text-muted);
    }

    .info-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
        gap: 0.85rem;
        margin-bottom: 1rem;
    }

    .info-card,
    .po-card,
    .comment-item {
        padding: 1rem;
        border-radius: 16px;
        background: rgba(248, 250, 252, 0.9);
        border: 1px solid rgba(15, 23, 42, 0.06);
    }

    .tabs-nav {
        display: flex;
        gap: 0.6rem;
        margin: 1rem 0;
    }

    .tab-btn {
        border: 1px solid rgba(15, 23, 42, 0.12);
        background: white;
        color: var(--text-primary);
        border-radius: 999px;
        padding: 0.55rem 0.95rem;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 0.45rem;
    }

    .tab-btn.active {
        background: rgba(37, 99, 235, 0.1);
        border-color: rgba(37, 99, 235, 0.25);
        color: #1d4ed8;
    }

    .tab-count {
        background: rgba(37, 99, 235, 0.12);
        color: #1d4ed8;
        font-size: 0.72rem;
    }

    .items-table {
        display: flex;
        flex-direction: column;
        border-radius: 16px;
        overflow: hidden;
        border: 1px solid rgba(15, 23, 42, 0.08);
    }

    .items-header,
    .items-row,
    .po-meta {
        display: grid;
        grid-template-columns: minmax(0, 1.8fr) 0.6fr 0.8fr 0.8fr;
        gap: 0.75rem;
        align-items: center;
    }

    .items-header,
    .items-row {
        padding: 0.85rem 1rem;
    }

    .items-header {
        background: rgba(15, 23, 42, 0.06);
        font-size: 0.78rem;
        text-transform: uppercase;
        letter-spacing: 0.06em;
    }

    .items-row:nth-child(even) {
        background: rgba(248, 250, 252, 0.9);
    }

    .part-number {
        display: block;
        margin-top: 0.25rem;
        font-size: 0.78rem;
        color: var(--text-muted);
    }

    .po-list,
    .timeline,
    .comment-list {
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
    }

    .po-meta {
        grid-template-columns: repeat(3, minmax(0, 1fr));
        margin-top: 0.75rem;
        font-size: 0.85rem;
    }

    .po-amount {
        font-weight: 800;
    }

    .timeline,
    .comments-section {
        margin-top: 1rem;
    }

    .event {
        padding-bottom: 0.8rem;
        border-bottom: 1px solid rgba(15, 23, 42, 0.06);
    }

    .note {
        margin: 0.35rem 0 0;
    }

    .comment-input {
        align-items: stretch;
        margin-top: 0.9rem;
    }

    .comment-textarea {
        width: 100%;
        resize: vertical;
        min-height: 92px;
    }

    .comment-btn {
        background: rgba(37, 99, 235, 0.12);
        color: #1d4ed8;
        cursor: pointer;
        font-weight: 700;
        white-space: nowrap;
    }

    .comment-author {
        font-weight: 700;
    }

    .comment-meta {
        display: flex;
        align-items: center;
        gap: 0.65rem;
        flex-wrap: wrap;
        justify-content: flex-end;
    }

    .comment-delete {
        border: 1px solid rgba(220, 38, 38, 0.18);
        border-radius: 999px;
        background: rgba(220, 38, 38, 0.08);
        color: #b91c1c;
        cursor: pointer;
        font-size: 0.72rem;
        font-weight: 700;
        padding: 0.32rem 0.6rem;
    }

    .comment-delete:disabled {
        cursor: wait;
        opacity: 0.65;
    }

    .comment-text {
        margin: 0.45rem 0 0;
        white-space: pre-wrap;
        line-height: 1.55;
    }

    .skeleton {
        background: rgba(15, 23, 42, 0.06);
        border-radius: 8px;
        animation: pulse 1.6s ease-in-out infinite;
    }

    .skeleton.title {
        height: 24px;
        width: 70%;
    }

    .skeleton.line {
        height: 16px;
        width: 100%;
        margin-top: 0.7rem;
    }

    .skeleton.line.short {
        width: 55%;
    }

    .error {
        color: var(--color-danger);
    }

    @keyframes pulse {
        0% { opacity: 0.5; }
        50% { opacity: 0.95; }
        100% { opacity: 0.5; }
    }

    @media (max-width: 860px) {
        .header,
        .header-meta,
        .comment-input {
            flex-direction: column;
            align-items: stretch;
        }

        .items-header,
        .items-row,
        .po-meta {
            grid-template-columns: 1fr;
        }
    }
</style>
