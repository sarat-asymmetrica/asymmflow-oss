<script lang="ts">
    import { run } from 'svelte/legacy';

    import { onMount, onDestroy } from "svelte";
    import { devLog } from "$lib/utils/devLog";
    import {
        GetCustomer360 } from "../../../wailsjs/go/main/App";
import { GetCustomer360Graph } from "../../../wailsjs/go/main/CRMService";
    import ContactCard from "../components/ContactCard.svelte";
    import RegimeBadge from "../components/consciousness/RegimeBadge.svelte";
    import MathematicalRigorBadge from "../components/consciousness/MathematicalRigorBadge.svelte";
    import WabiSpinner from "../components/ui/WabiSpinner.svelte";
    import { isAbortError } from "../utils/abortable";
    import { createEventDispatcher } from "svelte";

    const dispatch = createEventDispatcher();
    let { params = {} } = $props();

    function goBack() {
        dispatch("navigate", { screen: "relationships" });
    }

    let data = $state(null);
    let graphData = $state(null);
    let loading = $state(true);
    let error = $state("");
    let activeTab = $state("predictions"); // predictions | graph
    let loadController = null;

    async function loadData() {
        if (!params.id) return;
        loadController?.abort();
        loadController = new AbortController();
        loading = true;
        error = "";

        try {
            if (!window.go) {
                // Mock
                data = {
                    business_name: "Acme Corp",
                    customer_id: "CUST_001",
                    customer_type: "EC",
                    current_grade: "A",
                    total_orders_count: 12,
                    total_orders_value: 45000,
                    r1: 0.15,
                    r2: 0.25,
                    r3: 0.6,
                    industry: "Manufacturing",
                    city: "Manama",
                    country: "Bahrain",
                    relation_years: 5,
                    avg_payment_days: 42,
                    dispute_count: 0,
                    recent_predictions: [
                        {
                            grade: "A",
                            confidence: 0.85,
                            predicted_days: 40,
                            created_at: new Date().toISOString(),
                        },
                    ],
                };
            } else {
                const customerData = await GetCustomer360(params.id.toString());
                if (!loadController.signal.aborted) data = customerData;

                try {
                    const gData = await GetCustomer360Graph(
                        params.id.toString(),
                    );
                    if (!loadController.signal.aborted) graphData = gData;
                } catch (e) {
                    console.error('Failed to load customer graph:', e);
                    // Graph is optional - main data already loaded
                }
            }
        } catch (err) {
            if (!isAbortError(err)) error = "Failed to load profile";
        } finally {
            if (!loadController?.signal.aborted) loading = false;
        }
    }

    run(() => {
        if (params.id) loadData();
    });
    onDestroy(() => loadController?.abort());
</script>

<div class="page">
    {#if loading}
        <div class="loading-state"><WabiSpinner size="lg" tempo="calm" /></div>
    {:else if error}
        <div class="error-state">
            <p>{error}</p>
            <button onclick={loadData}>Retry</button>
        </div>
    {:else if data}
        <header class="header">
            <button class="back-btn" onclick={goBack}> Back to CRM </button>
            <div class="header-content">
                <p class="subtitle">Customer 360</p>
                <h1>{data.business_name || "Unknown"}</h1>
                <div class="badges">
                    <span class="badge">{data.customer_id}</span>
                    <span class="badge">{data.customer_type}</span>
                    <span class="badge grade-{data.current_grade}"
                        >Grade {data.current_grade}</span
                    >
                </div>
            </div>
            {#if data.r1}
                <div class="regime-box">
                    <span class="lbl">Payment Regime</span>
                    <RegimeBadge
                        r1={data.r1}
                        r2={data.r2}
                        r3={data.r3}
                        size="normal"
                        showPercentages={true}
                        showLabels={false}
                    />
                </div>
            {/if}
        </header>

        <div class="layout-split">
            <!-- Sidebar: Info -->
            <aside class="sidebar">
                <div class="panel info-panel">
                    <h3>At a Glance</h3>
                    <div class="info-row">
                        <span class="lbl">Industry</span>
                        <span class="val">{data.industry || "-"}</span>
                    </div>
                    <div class="info-row">
                        <span class="lbl">Location</span>
                        <span class="val">{data.city}, {data.country}</span>
                    </div>
                    <div class="info-row">
                        <span class="lbl">Relationship</span>
                        <span class="val">{data.relation_years} years</span>
                    </div>
                    <div class="info-row">
                        <span class="lbl">Avg Payment</span>
                        <span class="val"
                            >{data.avg_payment_days?.toFixed(0)} days</span
                        >
                    </div>
                    <div class="info-row">
                        <span class="lbl">Lifetime Value</span>
                        <span class="val"
                            >{(data.total_orders_value || 0).toLocaleString()} BHD</span
                        >
                    </div>
                </div>
            </aside>

            <!-- Main: Tabs & Predictions -->
            <main class="main-content">
                <div class="tabs">
                    <button
                        class:active={activeTab === "predictions"}
                        onclick={() => (activeTab = "predictions")}
                    >
                        Predictions ({data.recent_predictions?.length || 0})
                    </button>
                    <button
                        class:active={activeTab === "graph"}
                        onclick={() => (activeTab = "graph")}
                    >
                        Relationships {graphData
                            ? `(${graphData.graph_metrics?.total_connections})`
                            : ""}
                    </button>
                </div>

                <div class="tab-content">
                    {#if activeTab === "predictions"}
                        <div class="grid-cards">
                            {#each data.recent_predictions || [] as pred}
                                <div class="pred-card">
                                    <div class="pred-header">
                                        <span
                                            class="pred-grade grade-{pred.grade}"
                                            >Grade {pred.grade}</span
                                        >
                                        <span class="pred-date"
                                            >{new Date(
                                                pred.created_at,
                                            ).toLocaleDateString()}</span
                                        >
                                    </div>
                                    <div class="rigor-box">
                                        <MathematicalRigorBadge
                                            confidence={pred.confidence}
                                            predictedValue={pred.predicted_days}
                                            unit="days"
                                            size="small"
                                            showProofLink={false}
                                        />
                                    </div>
                                    <div class="regime-mini">
                                        <RegimeBadge
                                            r1={pred.r1}
                                            r2={pred.r2}
                                            r3={pred.r3}
                                            size="small"
                                            showPercentages={true}
                                        />
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {:else if activeTab === "graph"}
                        {#if !graphData}
                            <div class="empty">No graph data.</div>
                        {:else}
                            <div class="graph-section">
                                <h3>Connections</h3>
                                <div class="graph-stats">
                                    <div class="stat">
                                        <span>Connections</span>
                                        <strong
                                            >{graphData.graph_metrics
                                                .total_connections}</strong
                                        >
                                    </div>
                                    <div class="stat">
                                        <span>Centrality</span>
                                        <strong
                                            >{(
                                                graphData.graph_metrics
                                                    .centrality_score * 100
                                            ).toFixed(1)}%</strong
                                        >
                                    </div>
                                </div>
                                <!-- Simple lists for related entities -->
                                <div class="entity-groups">
                                    <div class="group">
                                        <h4>
                                            Products ({graphData
                                                .related_products.length})
                                        </h4>
                                        {#each graphData.related_products as p}
                                            <div class="chip">{p.name}</div>
                                        {/each}
                                    </div>
                                    <div class="group">
                                        <h4>
                                            Suppliers ({graphData
                                                .related_suppliers.length})
                                        </h4>
                                        {#each graphData.related_suppliers as s}
                                            <div class="chip">{s.name}</div>
                                        {/each}
                                    </div>
                                </div>
                            </div>
                        {/if}
                    {/if}
                </div>
            </main>
        </div>
    {/if}
</div>

<style>
    .page {
        padding: var(--page-padding);
        min-height: 100vh;
        background: #fafafa;
        color: #1d1d1f;
        display: flex;
        flex-direction: column;
        box-sizing: border-box;
    }

    .back-btn {
        background: none;
        border: none;
        color: #6e6e73;
        font-size: 14px;
        padding: 0;
        margin-bottom: 16px;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 4px;
    }
    .back-btn:hover {
        color: #1d1d1f;
    }

    .header {
        display: flex;
        flex-direction: column;
        margin-bottom: 24px;
        flex-shrink: 0;
    }
    .header-content {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
    }
    h1 {
        font-family: "Arvo", "Georgia", serif;
        font-size: 28px;
        font-weight: 400;
        margin: 4px 0 12px;
        letter-spacing: -0.02em;
    }
    .subtitle {
        color: var(--ink-faint);
        margin: 0;
        text-transform: uppercase;
        font-size: 11px;
        letter-spacing: 1px;
    }

    .badges {
        display: flex;
        gap: 8px;
    }
    .badge {
        padding: 4px 10px;
        border-radius: 4px;
        border: 1px solid var(--border-medium);
        font-size: 11px;
        font-family: var(--font-mono);
        color: var(--ink-light);
    }
    .badge.grade-A {
        background: #dcfce7;
        border-color: #86efac;
        color: #166534;
    }
    .badge.grade-B {
        background: #dbeafe;
        border-color: #93c5fd;
        color: #1e40af;
    }
    .badge.grade-C {
        background: #fef9c3;
        border-color: #fde047;
        color: #854d0e;
    }
    .badge.grade-D {
        background: #fee2e2;
        border-color: #fca5a5;
        color: #991b1b;
    }

    .regime-box {
        display: flex;
        flex-direction: column;
        align-items: flex-end;
        gap: 4px;
    }
    .lbl {
        font-size: 10px;
        text-transform: uppercase;
        color: var(--ink-light);
    }

    .layout-split {
        display: grid;
        grid-template-columns: 300px 1fr;
        gap: var(--space-8);
        flex: 1;
        min-height: 0;
    }

    .sidebar {
        border-right: 1px solid var(--border-subtle);
        padding-right: var(--space-4);
    }

    .info-panel {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }
    .info-panel h3 {
        font-size: 14px;
        text-transform: uppercase;
        color: var(--ink-light);
        border-bottom: 1px solid var(--border-medium);
        padding-bottom: 8px;
    }
    .info-row {
        display: flex;
        justify-content: space-between;
        font-size: 13px;
    }
    .info-row .lbl {
        color: var(--ink-light);
    }
    .info-row .val {
        font-weight: 500;
    }

    .main-content {
        display: flex;
        flex-direction: column;
    }

    .tabs {
        display: flex;
        gap: 24px;
        border-bottom: 1px solid var(--border-subtle);
        margin-bottom: 24px;
    }
    .tabs button {
        background: none;
        border: none;
        padding: 0 0 12px;
        cursor: pointer;
        color: var(--ink-light);
        font-size: 14px;
        border-bottom: 2px solid transparent;
    }
    .tabs button.active {
        color: var(--ink);
        border-bottom-color: var(--ink);
        font-weight: 500;
    }

    .tab-content {
        overflow-y: auto;
        flex: 1;
    }

    .grid-cards {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
        gap: 16px;
    }

    .pred-card {
        background: var(--paper-subtle);
        border: 1px solid var(--border-subtle);
        border-radius: var(--radius-lg);
        padding: 16px;
        display: flex;
        flex-direction: column;
        gap: 12px;
    }
    .pred-header {
        display: flex;
        justify-content: space-between;
        align-items: baseline;
    }
    .pred-grade {
        font-weight: 600;
        font-size: 18px;
    }
    .pred-grade.grade-A {
        color: #166534;
    }
    .pred-grade.grade-B {
        color: #1e40af;
    }
    .pred-grade.grade-D {
        color: #991b1b;
    }
    .pred-date {
        font-size: 11px;
        color: var(--ink-light);
        font-family: var(--font-mono);
    }

    .rigor-box {
        display: flex;
        justify-content: center;
    }

    .loading-state,
    .error-state {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        flex-direction: column;
        gap: 16px;
    }

    .chip {
        display: inline-block;
        background: var(--paper);
        border: 1px solid var(--border-medium);
        padding: 4px 8px;
        border-radius: 4px;
        font-size: 12px;
        margin: 0 4px 4px 0;
    }

    .graph-stats {
        display: flex;
        gap: 24px;
        margin-bottom: 24px;
    }
    .stat {
        display: flex;
        flex-direction: column;
    }
    .stat span {
        font-size: 10px;
        text-transform: uppercase;
        color: var(--ink-light);
    }
    .stat strong {
        font-size: 20px;
        font-weight: 400;
    }
</style>
