<script lang="ts">
    import { run } from 'svelte/legacy';

    /**
     * OrderDetail - Enhanced with Partial Fulfillment Tracking
     *
     * Phase 2 Enhancement:
     * - Fulfillment progress bar per item
     * - Overall order fulfillment percentage
     * - Partial shipment support
     * - Color-coded status indicators
     */
    import { createEventDispatcher, onMount } from "svelte";
    import ShipmentCard from "./ShipmentCard.svelte";
    import MathematicalRigorBadge from "./consciousness/MathematicalRigorBadge.svelte";
    import { GetOrderFulfillmentStatus } from "../../../wailsjs/go/main/App";

    type LineItem = {
        id?: number;
        description?: string;
        quantity?: number;
        quantity_shipped?: number;
        quantity_invoiced?: number;
        unitCost?: number;
        unit_price_bhd?: number;
        margin?: number;
        actual_margin?: number;
    };

    type OrderInfo = {
        id?: number;
        stage?: keyof typeof stageColors | string;
        status?: string;
        customer?: string;
        customer_name?: string;
        customer_id?: number;
        poNumber?: string;
        customer_po_number?: string;
        offerId?: string | number;
        offer_id?: number;
        opportunityId?: string | number;
        lineItems?: LineItem[];
        items?: any[];
        total_value_bhd?: number;
        payment_grade?: string;
        predicted_payment_days?: number;
    };

    type Shipment = {
        trackingNumber?: string;
        carrier?: string;
        status?: string;
        deliveryDate?: string | Date;
        items?: Array<number | string>;
    };

    type HistoryEntry = {
        stage?: string;
        note?: string;
        createdAt?: string | number | Date;
    };

    type OrderDetail = {
        order: OrderInfo;
        nextStages: string[];
        shipments?: Shipment[];
        history?: HistoryEntry[];
    } | null;

    interface Props {
        detail?: OrderDetail;
        loading?: boolean;
        error?: string;
    }

    let { detail = null, loading = false, error = "" }: Props = $props();

    // Fulfillment status
    let fulfillmentStatus: any = $state(null);
    let loadingFulfillment = false;



    async function loadFulfillmentStatus(orderId: number) {
        if (!orderId) return;
        loadingFulfillment = true;
        try {
            fulfillmentStatus = await GetOrderFulfillmentStatus(String(orderId));
        } catch (e) {
            // Silently fail - fulfillment data might not exist yet
            fulfillmentStatus = null;
        } finally {
            loadingFulfillment = false;
        }
    }




    // Fulfillment status color
    function getFulfillmentColor(status: string): string {
        switch (status) {
            case "Fully Invoiced":
                return "#15803d";
            case "Fully Shipped":
                return "#22c55e";
            case "Partially Shipped":
                return "#f59e0b";
            default:
                return "#9ca3af";
        }
    }

    const dispatch = createEventDispatcher();

    const stageColors = {
        PO_Received: "var(--color-gold)",
        In_Production: "var(--color-gold)",
        Ready_To_Ship: "var(--color-gold)",
        Shipped: "var(--color-safe)",
        Delivered: "var(--color-safe)",
    };

    let shipmentForm: {
        trackingNumber: string;
        carrier: string;
        status: string;
        deliveryDate: string;
        items: Set<number>;
        quantities: Map<number, number>;
    } = $state({
        trackingNumber: "",
        carrier: "",
        status: "Preparing",
        deliveryDate: "",
        items: new Set(),
        quantities: new Map(),
    });

    function handleQuantityInput(e: Event, idx: number) {
        const input = e.target as HTMLInputElement;
        updateShipmentQty(idx, parseFloat(input.value));
    }

    function toggleItem(idx: number) {
        const next = new Set(shipmentForm.items);
        if (next.has(idx)) {
            next.delete(idx);
            shipmentForm.quantities.delete(idx);
        } else {
            next.add(idx);
            // Default to remaining quantity
            const item = lineItems[idx];
            const remaining =
                (item.quantity || 0) - (item.quantity_shipped || 0);
            shipmentForm.quantities.set(idx, remaining);
        }
        shipmentForm = { ...shipmentForm, items: next };
    }

    function updateShipmentQty(idx: number, qty: number) {
        shipmentForm.quantities.set(idx, qty);
        shipmentForm = { ...shipmentForm };
    }

    function submitShipment() {
        const items = Array.from(shipmentForm.items);
        const itemQuantities = Object.fromEntries(shipmentForm.quantities);
        dispatch("shipment", {
            trackingNumber: shipmentForm.trackingNumber,
            carrier: shipmentForm.carrier,
            status: shipmentForm.status,
            deliveryDate: shipmentForm.deliveryDate,
            items,
            quantities: itemQuantities,
        });
        shipmentForm = {
            trackingNumber: "",
            carrier: "",
            status: "Preparing",
            deliveryDate: "",
            items: new Set(),
            quantities: new Map(),
        };
    }

    function changeStage(next) {
        dispatch("stage", next);
    }

    // Calculate item fulfillment percentage
    function getItemFulfillmentPct(item: LineItem): number {
        if (!item.quantity || item.quantity === 0) return 0;
        return ((item.quantity_shipped || 0) / item.quantity) * 100;
    }

    // Get remaining quantity for item
    function getRemainingQty(item: LineItem): number {
        return (item.quantity || 0) - (item.quantity_shipped || 0);
    }
    // Reactive: Normalize the order data structure
    let normalizedOrder = $derived(detail?.order || (detail as any) || {});
    let customerName = $derived(normalizedOrder.customer_name || normalizedOrder.customer);
    let customerId = $derived(normalizedOrder.customer_id);
    let poNumber =
        $derived(normalizedOrder.customer_po_number || normalizedOrder.poNumber);
    let stage = $derived(normalizedOrder.status || normalizedOrder.stage);
    let offerId = $derived(normalizedOrder.offer_id || normalizedOrder.offerId);
    let opportunityId = $derived(normalizedOrder.opportunityId);
    let lineItems = $derived(normalizedOrder.items || normalizedOrder.lineItems || []);
    let totalValue = $derived(normalizedOrder.total_value_bhd);
    let paymentGrade = $derived(normalizedOrder.payment_grade);
    let predictedDays = $derived(normalizedOrder.predicted_payment_days);
    // Load fulfillment status when order changes
    run(() => {
        if (normalizedOrder.id) {
            loadFulfillmentStatus(normalizedOrder.id);
        }
    });
    // Payment prediction severity
    let paymentSeverity = $derived(!predictedDays
        ? "unknown"
        : predictedDays < 15
          ? "quick"
          : predictedDays <= 30
            ? "standard"
            : predictedDays <= 60
              ? "slow"
              : "very-slow");
    let paymentLabel = $derived(!predictedDays
        ? "Not calculated"
        : predictedDays < 15
          ? "Quick payment expected"
          : predictedDays <= 30
            ? "Standard payment"
            : predictedDays <= 60
              ? "Slow payment expected"
              : "Very slow payment");
    let paymentColor = $derived({
        quick: "var(--color-safe)",
        standard: "#3b82f6",
        slow: "#eab308",
        "very-slow": "var(--color-danger)",
        unknown: "var(--color-ink-light)",
    }[paymentSeverity]);
</script>

<div class="panel">
    {#if loading}
        <p class="mono">Loading order…</p>
    {:else if error}
        <p class="note danger">{error}</p>
    {:else if !detail}
        <p class="muted">Select an order to view details.</p>
    {:else}
        <header class="header">
            <div>
                <div class="status">
                    <span
                        class="dot"
                        style={`background:${stageColors[stage] || "var(--color-ink)"}`}
                    ></span>
                    <span class="mono">{stage}</span>
                </div>
                <h2>{customerName}</h2>
                <p class="muted">PO: {poNumber || "Awaiting"}</p>
                <p class="muted">
                    Offer #{offerId} - Opportunity #{opportunityId}
                </p>

                {#if predictedDays}
                    <div class="payment-prediction-container">
                        <MathematicalRigorBadge
                            confidence={0.75}
                            predictedValue={predictedDays}
                            unit="days"
                            size="normal"
                            showProofLink={true}
                            proofType="satorigami"
                        />
                        {#if paymentGrade}
                            <div class="grade-badge">
                                <span class="mono"
                                    >Payment Grade: {paymentGrade}</span
                                >
                            </div>
                        {/if}
                    </div>
                {/if}
            </div>
            <div class="actions">
                {#if detail?.nextStages}
                    {#each detail.nextStages as next}
                        <button class="ghost" onclick={() => changeStage(next)}
                            >{next.replaceAll("_", " ")}</button
                        >
                    {/each}
                {/if}
            </div>
        </header>

        <!-- Fulfillment Status Banner -->
        {#if fulfillmentStatus}
            <section
                class="fulfillment-banner"
                style="--fill-color: {getFulfillmentColor(
                    fulfillmentStatus.status,
                )}"
            >
                <div class="fulfillment-header">
                    <span class="fulfillment-status"
                        >{fulfillmentStatus.status}</span
                    >
                    <span class="fulfillment-pct"
                        >{(fulfillmentStatus.fulfillment_pct * 100).toFixed(0)}%
                        Shipped</span
                    >
                </div>
                <div class="fulfillment-bar">
                    <div
                        class="fulfillment-fill"
                        style="width: {fulfillmentStatus.fulfillment_pct *
                            100}%"
                    ></div>
                    <div
                        class="invoiced-fill"
                        style="width: {fulfillmentStatus.invoicing_pct * 100}%"
                    ></div>
                </div>
                <div class="fulfillment-legend">
                    <span
                        ><span class="legend-dot shipped"></span> Shipped: {fulfillmentStatus.shipped_quantity?.toFixed(
                            0,
                        ) || 0}</span
                    >
                    <span
                        ><span class="legend-dot invoiced"></span> Invoiced: {fulfillmentStatus.invoiced_quantity?.toFixed(
                            0,
                        ) || 0}</span
                    >
                    <span
                        >Total: {fulfillmentStatus.total_quantity?.toFixed(0) ||
                            0}</span
                    >
                </div>
            </section>
        {/if}

        <section class="grid">
            <div class="card">
                <p class="mono">Line Items</p>
                <table class="items">
                    <thead>
                        <tr>
                            <th>#</th>
                            <th>Description</th>
                            <th>Qty</th>
                            <th>Shipped</th>
                            <th>Fulfillment</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each lineItems as item, idx}
                            {@const fulfillmentPct =
                                getItemFulfillmentPct(item)}
                            {@const remaining = getRemainingQty(item)}
                            <tr class:fully-shipped={fulfillmentPct >= 100}>
                                <td>{idx + 1}</td>
                                <td>{item.description}</td>
                                <td>{item.quantity}</td>
                                <td>
                                    <span class="shipped-qty"
                                        >{item.quantity_shipped || 0}</span
                                    >
                                    {#if remaining > 0}
                                        <span class="remaining-qty"
                                            >({remaining} left)</span
                                        >
                                    {/if}
                                </td>
                                <td>
                                    <div class="item-progress">
                                        <div
                                            class="item-progress-fill"
                                            style="width: {fulfillmentPct}%"
                                        ></div>
                                    </div>
                                    <span class="item-pct"
                                        >{fulfillmentPct.toFixed(0)}%</span
                                    >
                                </td>
                                <td>
                                    {#if remaining > 0}
                                        <input
                                            type="checkbox"
                                            checked={shipmentForm.items.has(
                                                idx,
                                            )}
                                            onchange={() => toggleItem(idx)}
                                        />
                                        {#if shipmentForm.items.has(idx)}
                                            <input
                                                type="number"
                                                min="1"
                                                max={remaining}
                                                value={shipmentForm.quantities.get(
                                                    idx,
                                                ) || remaining}
                                                oninput={(e) =>
                                                    handleQuantityInput(e, idx)}
                                                class="qty-input"
                                            />
                                        {/if}
                                    {:else}
                                        <span class="check-icon">Done</span>
                                    {/if}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>

            <div class="card">
                <p class="mono">Shipments</p>
                {#if detail.shipments?.length === 0}
                    <p class="muted">No shipments yet.</p>
                {:else}
                    <div class="stack">
                        {#each detail.shipments as shipment}
                            <ShipmentCard {shipment} {stageColors} />
                        {/each}
                    </div>
                {/if}

                <div class="shipment-form">
                    <p class="mono">Create Partial Shipment</p>
                    <div class="form-grid">
                        <label
                            >Tracking
                            <input
                                bind:value={shipmentForm.trackingNumber}
                                placeholder="Tracking number"
                            />
                        </label>
                        <label
                            >Carrier
                            <input
                                bind:value={shipmentForm.carrier}
                                placeholder="e.g. DHL, Aramex"
                            />
                        </label>
                        <label
                            >Status
                            <select bind:value={shipmentForm.status}>
                                <option>Preparing</option>
                                <option>Picked Up</option>
                                <option>In Transit</option>
                                <option>Delivered</option>
                            </select>
                        </label>
                        <label
                            >Delivery Date
                            <input
                                type="date"
                                bind:value={shipmentForm.deliveryDate}
                            />
                        </label>
                    </div>
                    <button
                        class="primary"
                        onclick={submitShipment}
                        disabled={shipmentForm.items.size === 0}
                    >
                        Ship {shipmentForm.items.size} Item{shipmentForm.items
                            .size !== 1
                            ? "s"
                            : ""}
                    </button>
                </div>
            </div>
        </section>

        <section class="timeline">
            <p class="mono">Stage Timeline</p>
            <div class="history">
                {#if detail.history?.length === 0}
                    <p class="muted">No history yet.</p>
                {:else}
                    {#each detail.history as entry}
                        <div class="history-row">
                            <span
                                class="dot"
                                style={`background:${stageColors[entry.stage] || "var(--color-gold)"}`}
                            ></span>
                            <div>
                                <p class="mono">{entry.stage}</p>
                                <p class="muted">{entry.note}</p>
                            </div>
                            <span class="mono"
                                >{new Date(
                                    entry.createdAt,
                                ).toLocaleString()}</span
                            >
                        </div>
                    {/each}
                {/if}
            </div>
        </section>
    {/if}
</div>

<style>
    .panel {
        background: rgba(255, 255, 255, 0.7);
        border: 1px solid rgba(0, 0, 0, 0.08);
        padding: 1rem;
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }
    .header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 1rem;
    }
    .status {
        display: flex;
        gap: 0.4rem;
        align-items: center;
        font-family: var(--font-mono);
        letter-spacing: 1px;
        text-transform: uppercase;
        font-size: 0.8rem;
    }
    .dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
        border: 1px solid rgba(0, 0, 0, 0.1);
    }
    h2 {
        margin: 0.1rem 0;
        font-family: var(--font-serif);
    }
    .muted {
        margin: 0;
        color: var(--color-ink-light);
    }
    .actions {
        display: flex;
        gap: 0.5rem;
        flex-wrap: wrap;
    }
    .ghost {
        background: transparent;
        border: 1px solid rgba(0, 0, 0, 0.12);
        padding: 0.45rem 0.7rem;
        font-family: var(--font-mono);
        letter-spacing: 1px;
        text-transform: uppercase;
        cursor: pointer;
    }

    /* Fulfillment Banner */
    .fulfillment-banner {
        background: rgba(0, 0, 0, 0.02);
        border: 1px solid rgba(0, 0, 0, 0.06);
        border-radius: 8px;
        padding: 1rem;
    }

    .fulfillment-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 0.5rem;
    }

    .fulfillment-status {
        font-family: var(--font-mono);
        font-size: 0.75rem;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: var(--fill-color);
        font-weight: 600;
    }

    .fulfillment-pct {
        font-family: var(--font-serif);
        font-size: 1.25rem;
        font-weight: 600;
        color: var(--fill-color);
    }

    .fulfillment-bar {
        position: relative;
        height: 8px;
        background: rgba(0, 0, 0, 0.08);
        border-radius: 4px;
        overflow: hidden;
        margin-bottom: 0.5rem;
    }

    .fulfillment-fill {
        position: absolute;
        left: 0;
        top: 0;
        height: 100%;
        background: #22c55e;
        transition: width 0.3s ease;
    }

    .invoiced-fill {
        position: absolute;
        left: 0;
        top: 0;
        height: 100%;
        background: #15803d;
        transition: width 0.3s ease;
    }

    .fulfillment-legend {
        display: flex;
        gap: 1rem;
        font-size: 0.75rem;
        color: var(--color-ink-light);
    }

    .legend-dot {
        display: inline-block;
        width: 8px;
        height: 8px;
        border-radius: 50%;
        margin-right: 4px;
    }

    .legend-dot.shipped {
        background: #22c55e;
    }
    .legend-dot.invoiced {
        background: #15803d;
    }

    /* Item Progress */
    .item-progress {
        width: 60px;
        height: 6px;
        background: rgba(0, 0, 0, 0.1);
        border-radius: 3px;
        overflow: hidden;
        display: inline-block;
        vertical-align: middle;
        margin-right: 0.5rem;
    }

    .item-progress-fill {
        height: 100%;
        background: #22c55e;
        transition: width 0.3s ease;
    }

    .item-pct {
        font-size: 0.7rem;
        color: var(--color-ink-light);
    }

    .shipped-qty {
        font-weight: 600;
    }

    .remaining-qty {
        font-size: 0.7rem;
        color: var(--color-ink-light);
    }

    .fully-shipped {
        opacity: 0.6;
        background: rgba(21, 128, 61, 0.05);
    }

    .check-icon {
        color: #15803d;
        font-weight: bold;
    }

    .qty-input {
        width: 60px !important;
        padding: 0.25rem !important;
        font-size: 0.8rem;
        margin-left: 0.25rem;
    }

    .grid {
        display: grid;
        grid-template-columns: 1.2fr 1fr;
        gap: 0.85rem;
    }
    .card {
        background: rgba(255, 255, 255, 0.6);
        border: 1px solid rgba(0, 0, 0, 0.08);
        padding: 0.75rem;
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }
    .mono {
        font-family: var(--font-mono);
        letter-spacing: 1px;
        text-transform: uppercase;
    }
    .items {
        width: 100%;
        border-collapse: collapse;
    }
    .items th,
    .items td {
        border-bottom: 1px solid rgba(0, 0, 0, 0.08);
        text-align: left;
        padding: 0.35rem;
        font-family: var(--font-serif);
        font-size: 0.85rem;
    }

    .stack {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }
    .shipment-form {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
        border-top: 1px dashed rgba(0, 0, 0, 0.1);
        padding-top: 0.5rem;
        margin-top: auto;
    }
    .form-grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 0.5rem;
    }
    input,
    select {
        width: 100%;
        padding: 0.5rem;
        border: 1px solid rgba(0, 0, 0, 0.15);
        background: rgba(255, 255, 255, 0.7);
        font-family: var(--font-serif);
    }
    .primary {
        background: var(--color-ink);
        color: var(--color-paper);
        border: none;
        padding: 0.55rem 0.8rem;
        font-family: var(--font-mono);
        letter-spacing: 1px;
        text-transform: uppercase;
        cursor: pointer;
        align-self: flex-start;
    }
    .primary:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .timeline {
        background: rgba(255, 255, 255, 0.6);
        border: 1px solid rgba(0, 0, 0, 0.08);
        padding: 0.75rem;
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }
    .history {
        display: flex;
        flex-direction: column;
        gap: 0.4rem;
    }
    .history-row {
        display: flex;
        align-items: center;
        gap: 0.6rem;
        justify-content: space-between;
        border-bottom: 1px dashed rgba(0, 0, 0, 0.08);
        padding: 0.35rem 0;
    }
    .history-row div {
        flex: 1;
    }

    /* Payment Prediction Container */
    .payment-prediction-container {
        margin-top: 0.75rem;
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }

    .grade-badge {
        padding: 0.5rem;
        background: rgba(197, 160, 89, 0.05);
        border: 1px solid rgba(197, 160, 89, 0.2);
        border-radius: 4px;
        text-align: center;
    }

    .grade-badge .mono {
        font-size: 0.75rem;
        color: var(--color-ink-light);
    }

    @media (max-width: 900px) {
        .grid {
            grid-template-columns: 1fr;
        }
    }
</style>
