<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { fade } from "svelte/transition";
    import { toast } from "../stores/toasts";
    import {
        GetPricingRecommendation } from "../../../wailsjs/go/main/App";
import { SimulateMargin } from "../../../wailsjs/go/main/InfraService";
    import WabiSpinner from "../components/ui/WabiSpinner.svelte";

    let loading = false;
    let selectedCustomer = $state(null);
    let simMargin = $state(0.2);
    let simResult = $state(null);
    let simulating = $state(false);

    // Mock Overall Data
    let overallStats = {
        winRate: 0.35,
        averageMargin: 0.22,
        revenue: 1250000,
        customers: [
            {
                id: 1,
                name: "Gulf Smelting Co.",
                regime: "Premium",
                winRate: 0.45,
                rev: 450000,
            },
            {
                id: 2,
                name: "National Petroleum Co.",
                regime: "PriceSensitive",
                winRate: 0.28,
                rev: 320000,
            },
            {
                id: 3,
                name: "Delta Petrochemicals",
                regime: "ValueBalanced",
                winRate: 0.35,
                rev: 280000,
            },
            {
                id: 4,
                name: "Highland Energy",
                regime: "Premium",
                winRate: 0.55,
                rev: 150000,
            },
        ],
    };

    function selectCustomer(c) {
        selectedCustomer = c;
        simResult = null;
    }

    async function runSimulation() {
        if (!selectedCustomer) {
            toast.warning("Please select a customer first");
            return;
        }
        simulating = true;
        simResult = null;
        try {
            // Real API call - SimulateMargin expects (customerName, margin)
            const res = await SimulateMargin(
                selectedCustomer.name || selectedCustomer.id,
                simMargin,
            );

            if (res) {
                // Map backend response to UI format
                simResult = {
                    projectedWinRate: res.estimated_win_rate || 0,
                    currentWinRate: res.current_win_rate || 0,
                    confidence: res.confidence || 0,
                    impact: res.recommended_action || "No recommendation",
                    warning: res.warning || "",
                };
            } else {
                toast.warning("No simulation result received");
            }
        } catch (e) {
            console.error("Simulation error:", e);
            toast.danger("Simulation failed: " + (e.message || "Unknown error"));
        } finally {
            simulating = false;
        }
    }
</script>

<div class="page">
    <header class="header">
        <div class="header-content">
            <h1>Pricing.</h1>
            <p class="subtitle">Strategy & Simulation</p>
        </div>
        <div class="meta-badges">
            <span class="badge"
                >Avg Margin: {(overallStats.averageMargin * 100).toFixed(
                    1,
                )}%</span
            >
            <span class="badge"
                >Win Rate: {(overallStats.winRate * 100).toFixed(1)}%</span
            >
        </div>
    </header>

    <div class="layout-split">
        <!-- Sidebar: Customers -->
        <aside class="sidebar">
            <h3>Customers</h3>
            <div class="cust-list">
                {#each overallStats.customers as c}
                    <button
                        class="cust-item"
                        class:active={selectedCustomer?.id === c.id}
                        onclick={() => selectCustomer(c)}
                    >
                        <div class="cust-head">
                            <span class="name">{c.name}</span>
                            <span class="regime-dot {c.regime}"></span>
                        </div>
                        <div class="cust-meta">
                            WR: {(c.winRate * 100).toFixed(0)}% • {c.regime}
                        </div>
                    </button>
                {/each}
            </div>
        </aside>

        <!-- Main: Simulation -->
        <main class="main-content">
            {#if !selectedCustomer}
                <div class="empty-state">
                    <p>Select a customer to analyze pricing strategy.</p>
                </div>
            {:else}
                <div class="analysis-panel" in:fade>
                    <div class="panel-header">
                        <h2>{selectedCustomer.name}</h2>
                        <span class="tag"
                            >{selectedCustomer.regime} Strategy</span
                        >
                    </div>

                    <div class="sim-controls">
                        <label for="pricing-target-margin"
                            >Target Margin: {(simMargin * 100).toFixed(
                                0,
                            )}%</label
                        >
                        <input
                            id="pricing-target-margin"
                            type="range"
                            min="0.05"
                            max="0.50"
                            step="0.01"
                            bind:value={simMargin}
                            class="slider"
                        />
                        <div class="range-labels">
                            <span>5%</span>
                            <span>25%</span>
                            <span>50%</span>
                        </div>
                        <button
                            class="btn-primary"
                            onclick={runSimulation}
                            disabled={simulating}
                        >
                            {simulating ? "Simulating..." : "Run Simulation"}
                        </button>
                    </div>

                    {#if simResult}
                        <div class="result-box" in:fade>
                            <h3>Projected Outcome</h3>
                            <div class="res-grid">
                                <div class="res-item">
                                    <span class="lbl">Win Probability</span>
                                    <span class="val"
                                        >{(
                                            simResult.projectedWinRate * 100
                                        ).toFixed(1)}%</span
                                    >
                                </div>
                                <div class="res-item">
                                    <span class="lbl">Strategic Impact</span>
                                    <span class="val text-sm"
                                        >{simResult.impact}</span
                                    >
                                </div>
                            </div>
                        </div>
                    {/if}

                    <div class="insight-box">
                        <h3>Customer Profile</h3>
                        <p>
                            {selectedCustomer.name} operates under a
                            <strong>{selectedCustomer.regime}</strong>
                            regime.
                            {#if selectedCustomer.regime === "Premium"}
                                They prioritize quality and reliability. Higher
                                margins are accepted if service levels are high.
                            {:else if selectedCustomer.regime === "PriceSensitive"}
                                Price is the primary driver. Competitive margins
                                (10-15%) are critical for winning deals.
                            {:else}
                                They balance cost and value. Mid-range margins
                                (18-22%) with value-adds work best.
                            {/if}
                        </p>
                    </div>
                </div>
            {/if}
        </main>
    </div>
</div>

<style>
    .page {
        padding: var(--page-padding);
        height: 100vh;
        background: var(--paper);
        color: var(--ink);
        display: flex;
        flex-direction: column;
        box-sizing: border-box;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: flex-end;
        margin-bottom: var(--space-6);
        flex-shrink: 0;
    }
    h1 {
        font-size: var(--text-5xl);
        font-weight: var(--font-weight-light);
        margin: 0;
        letter-spacing: -0.02em;
    }
    .subtitle {
        color: var(--ink-faint);
        margin-top: var(--space-2);
    }

    .meta-badges {
        display: flex;
        gap: 8px;
    }
    .badge {
        padding: 4px 10px;
        background: var(--paper-subtle);
        border: 1px solid var(--border-medium);
        border-radius: 20px;
        font-size: 11px;
    }

    .layout-split {
        display: grid;
        grid-template-columns: 260px 1fr;
        gap: var(--space-8);
        flex: 1;
        min-height: 0;
    }

    .sidebar {
        border-right: 1px solid var(--border-subtle);
        padding-right: 16px;
        display: flex;
        flex-direction: column;
        gap: 16px;
    }
    .sidebar h3 {
        font-size: 11px;
        text-transform: uppercase;
        color: var(--ink-light);
        margin: 0;
    }

    .cust-item {
        display: block;
        width: 100%;
        text-align: left;
        padding: 12px;
        margin-bottom: 8px;
        background: var(--paper-subtle);
        border: 1px solid transparent;
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.2s;
    }
    .cust-item:hover {
        transform: translateY(-1px);
        box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
    }
    .cust-item.active {
        background: var(--ink);
        color: var(--paper);
        border-color: var(--ink);
    }
    .cust-item.active .cust-meta {
        color: rgba(255, 255, 255, 0.7);
    }

    .cust-head {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 4px;
    }
    .name {
        font-weight: 500;
        font-size: 14px;
    }
    .cust-meta {
        font-size: 11px;
        color: var(--ink-light);
    }

    .regime-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        display: inline-block;
    }
    .regime-dot.Premium {
        background: #166534;
    }
    .regime-dot.PriceSensitive {
        background: #dc2626;
    }
    .regime-dot.ValueBalanced {
        background: #d97706;
    }

    .main-content {
        overflow-y: auto;
        display: flex;
        flex-direction: column;
    }
    .empty-state {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: var(--ink-light);
        font-style: italic;
    }

    .analysis-panel {
        padding: 24px;
        background: var(--paper-subtle);
        border-radius: 16px;
        border: 1px solid var(--border-subtle);
        max-width: 600px;
    }
    .panel-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 32px;
    }
    .panel-header h2 {
        margin: 0;
        font-weight: 400;
        font-size: 24px;
    }
    .tag {
        font-size: 11px;
        text-transform: uppercase;
        background: var(--paper);
        padding: 4px 8px;
        border-radius: 4px;
        border: 1px solid var(--border-medium);
    }

    .sim-controls {
        margin-bottom: 32px;
        background: var(--paper);
        padding: 24px;
        border-radius: 12px;
        border: 1px solid var(--border-medium);
    }
    .slider {
        width: 100%;
        margin: 16px 0;
    }
    .range-labels {
        display: flex;
        justify-content: space-between;
        font-size: 10px;
        color: var(--ink-light);
        margin-bottom: 16px;
    }

    .btn-primary {
        width: 100%;
        background: var(--ink);
        color: var(--paper);
        border: none;
        padding: 12px;
        border-radius: 8px;
        cursor: pointer;
    }

    .result-box {
        background: var(--ink);
        color: var(--paper);
        padding: 24px;
        border-radius: 12px;
        margin-bottom: 24px;
    }
    .result-box h3 {
        margin: 0 0 16px;
        font-size: 11px;
        text-transform: uppercase;
        color: rgba(255, 255, 255, 0.7);
    }
    .res-grid {
        display: flex;
        gap: 32px;
    }
    .res-item {
        display: flex;
        flex-direction: column;
    }
    .res-item .lbl {
        font-size: 10px;
        text-transform: uppercase;
        opacity: 0.7;
        margin-bottom: 4px;
    }
    .res-item .val {
        font-size: 24px;
        font-weight: 300;
    }
    .text-sm {
        font-size: 16px;
    }

    .insight-box {
        font-size: 14px;
        line-height: 1.6;
        color: var(--ink-light);
    }
    .insight-box h3 {
        font-size: 11px;
        text-transform: uppercase;
        color: var(--ink);
        margin-bottom: 8px;
    }
</style>
