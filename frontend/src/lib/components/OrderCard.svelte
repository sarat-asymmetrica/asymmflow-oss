<script lang="ts">
    /**
     * OrderCard - Enhanced with Fulfillment Status
     *
     * Shows mini fulfillment progress bar and value
     */
    import { createEventDispatcher } from "svelte";

    interface Props {
        order?: any;
        loading?: boolean;
        selected?: boolean;
    }

    let { order = {}, loading = false, selected = false }: Props = $props();

    const dispatch = createEventDispatcher();

    const stageColors = {
        PO_Received: "#eab308",
        In_Production: "#f59e0b",
        Ready_To_Ship: "#22c55e",
        Shipped: "#16a34a",
        Delivered: "#15803d",
    };


    function calculateFulfillment(order) {
        const items = order.items || [];
        if (items.length === 0) return 0;

        let totalQty = 0;
        let shippedQty = 0;

        items.forEach((item) => {
            totalQty += item.quantity || 0;
            shippedQty += item.quantity_shipped || 0;
        });

        if (totalQty === 0) return 0;
        return (shippedQty / totalQty) * 100;
    }

    function handleClick() {
        if (loading) return;
        dispatch("select");
    }

    function handleKeydown(e) {
        if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            handleClick();
        }
    }

    // Format value
    function formatValue(val) {
        if (!val) return "—";
        return (
            new Intl.NumberFormat("en-BH", {
                minimumFractionDigits: 0,
                maximumFractionDigits: 0,
            }).format(val) + " BHD"
        );
    }

    // Calculate fulfillment from order items if available
    let fulfillmentPct = $derived(calculateFulfillment(order));
    // Get customer name from various sources
    let customerName = $derived(order.customer_name || order.customer || "Unknown");
    let poNumber = $derived(order.customer_po_number || order.poNumber || "Pending");
    let stage = $derived(order.status || order.stage || "PO_Received");
    let totalValue = $derived(order.total_value_bhd || order.totalValue || 0);
</script>

<div
    class={`card ${selected ? "selected" : ""}`}
    role="button"
    tabindex={loading ? -1 : 0}
    onclick={handleClick}
    onkeydown={handleKeydown}
    aria-label={loading ? "Loading order" : `Order for ${customerName}`}
>
    <div class="row">
        <div class="status">
            <span
                class={`dot ${loading ? "skeleton-dot" : ""}`}
                style={!loading
                    ? `background:${stageColors[stage] || "#6b7280"}`
                    : ""}
            ></span>
            <span class="mono"
                >{loading ? "Loading…" : stage.replaceAll("_", " ")}</span
            >
        </div>
        <span class="value">{loading ? "---" : formatValue(totalValue)}</span>
    </div>

    <h3>{loading ? "–––––––" : customerName}</h3>
    <p class="po-number">{loading ? "---" : `PO: ${poNumber}`}</p>

    <!-- Fulfillment Progress Bar -->
    {#if !loading && fulfillmentPct > 0}
        <div class="fulfillment-row">
            <div class="mini-progress">
                <div
                    class="mini-progress-fill"
                    style="width: {fulfillmentPct}%"
                ></div>
            </div>
            <span class="fulfillment-pct">{fulfillmentPct.toFixed(0)}%</span>
        </div>
    {:else if !loading}
        <div class="fulfillment-row">
            <span class="pending-label">Not shipped</span>
        </div>
    {/if}

    <div class="meta">
        <span class="mono">{loading ? "---" : `#${order.id || "—"}`}</span>
        <span class="mono"
            >{loading
                ? "---"
                : order.order_date
                  ? new Date(order.order_date).toLocaleDateString()
                  : ""}</span
        >
    </div>
</div>

<style>
    .card {
        background: rgba(255, 255, 255, 0.7);
        border: 1px solid rgba(0, 0, 0, 0.08);
        border-radius: 6px;
        padding: 0.75rem 1rem;
        display: flex;
        flex-direction: column;
        gap: 0.3rem;
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .card:hover {
        border-color: rgba(0, 0, 0, 0.2);
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
    }

    .selected {
        border-color: #1c1c1c;
        box-shadow: 0 6px 14px rgba(0, 0, 0, 0.08);
        background: rgba(255, 255, 255, 0.9);
    }

    .row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 0.5rem;
    }

    .status {
        display: flex;
        align-items: center;
        gap: 0.4rem;
        font-family: "Courier Prime", monospace;
        letter-spacing: 0.5px;
        text-transform: uppercase;
        font-size: 0.7rem;
        color: #57534e;
    }

    .dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
    }

    .value {
        font-family: Georgia, serif;
        font-size: 0.9rem;
        font-weight: 600;
        color: #1c1c1c;
    }

    h3 {
        margin: 0;
        font-family: Georgia, serif;
        font-size: 0.95rem;
        font-weight: 500;
    }

    .po-number {
        margin: 0;
        color: #57534e;
        font-size: 0.8rem;
        font-family: "Courier Prime", monospace;
    }

    /* Mini Fulfillment Progress */
    .fulfillment-row {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        margin-top: 0.25rem;
    }

    .mini-progress {
        flex: 1;
        height: 4px;
        background: rgba(0, 0, 0, 0.08);
        border-radius: 2px;
        overflow: hidden;
    }

    .mini-progress-fill {
        height: 100%;
        background: linear-gradient(90deg, #22c55e 0%, #16a34a 100%);
        transition: width 0.3s ease;
    }

    .fulfillment-pct {
        font-family: "Courier Prime", monospace;
        font-size: 0.65rem;
        color: #22c55e;
        font-weight: 600;
        min-width: 30px;
        text-align: right;
    }

    .pending-label {
        font-size: 0.7rem;
        color: #9ca3af;
        font-style: italic;
    }

    .meta {
        display: flex;
        gap: 0.8rem;
        flex-wrap: wrap;
        font-family: "Courier Prime", monospace;
        font-size: 0.7rem;
        color: #9ca3af;
        margin-top: 0.25rem;
    }

    .mono {
        font-family: "Courier Prime", monospace;
    }

    .skeleton-dot {
        background: rgba(0, 0, 0, 0.1);
    }

    .card.loading {
        pointer-events: none;
    }
</style>
