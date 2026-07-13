<script lang="ts">
    import { fade } from "svelte/transition";
    import { motionMs } from "$lib/motion";
    import CRMCustomerDashboard from "./CRMCustomerDashboard.svelte";
    import CRMSupplierDashboard from "./CRMSupplierDashboard.svelte";
    import CustomerDetailView from "./CustomerDetailView.svelte";
    import SupplierDetailView from "./SupplierDetailView.svelte";
    import DataQualityScreen from "./DataQualityScreen.svelte";
    import ErrorBoundary from "../components/ErrorBoundary.svelte";

    let activeTab = $state("customers");
    let selectedCustomer: string | null = $state(null);
    let selectedSupplier: string | null = $state(null);

    // Text-only tabs - Living Minimalist
    const tabs = [
        { id: "customers", label: "Customers" },
        { id: "suppliers", label: "Suppliers" },
        { id: "data_quality", label: "Data Quality" },
    ];

    function handleTabChange(tabId: string) {
        activeTab = tabId;
        selectedCustomer = null;
        selectedSupplier = null;
    }

    function handleCustomerSelect(e: CustomEvent<{id: string}>) {
        selectedCustomer = e.detail.id;
    }

    function handleSupplierSelect(e: CustomEvent<{id: string}>) {
        selectedSupplier = e.detail.id;
    }

    function handleCustomerBack() {
        selectedCustomer = null;
    }

    function handleSupplierBack() {
        selectedSupplier = null;
    }

    function handleDataQualityOpenIssue(event: CustomEvent<{ issue: any }>) {
        const issue = event.detail?.issue || {};
        const entityType = String(issue.entity_type || "").toLowerCase();
        const entityID = String(issue.entity_id || "");

        if (entityType.includes("customer") && entityID) {
            activeTab = "customers";
            selectedSupplier = null;
            selectedCustomer = entityID;
            return;
        }
        if (entityType.includes("supplier") && entityID) {
            activeTab = "suppliers";
            selectedCustomer = null;
            selectedSupplier = entityID;
            return;
        }
        if (entityType.includes("offer") || entityType.includes("opportun")) {
            window.dispatchEvent(new CustomEvent("navigateToScreen", {
                detail: { screen: "opportunities" },
            }));
        }
    }
</script>

<ErrorBoundary name="CRM Hub">
    <div class="hub" in:fade={{ duration: motionMs(200) }}>
        <header class="header">
            <div class="header-row">
                <h1>Customers &amp; Suppliers</h1>
                {#if selectedCustomer || selectedSupplier}
                    <button class="back-btn" onclick={() => { selectedCustomer = null; selectedSupplier = null; }}>
                        Back to Dashboard
                    </button>
                {/if}
            </div>

            {#if !selectedCustomer && !selectedSupplier}
                <nav class="tabs">
                    {#each tabs as tab}
                        <button
                            class="tab"
                            class:active={activeTab === tab.id}
                            onclick={() => handleTabChange(tab.id)}
                        >
                            {tab.label}
                        </button>
                    {/each}
                </nav>
            {/if}
        </header>

        <main class="content">
            {#if activeTab === "customers"}
                {#if selectedCustomer}
                    <CustomerDetailView customerId={selectedCustomer} on:back={handleCustomerBack} />
                {:else}
                    <CRMCustomerDashboard on:select={handleCustomerSelect} />
                {/if}
            {:else if activeTab === "suppliers"}
                {#if selectedSupplier}
                    <SupplierDetailView supplierId={selectedSupplier} on:back={handleSupplierBack} />
                {:else}
                    <CRMSupplierDashboard on:select={handleSupplierSelect} />
                {/if}
            {:else if activeTab === "data_quality"}
                <DataQualityScreen on:openIssue={handleDataQualityOpenIssue} />
            {/if}
        </main>
    </div>
</ErrorBoundary>

<style>
    .hub {
        min-height: 100vh;
        background: var(--bg-base);
    }

    .header {
        padding: var(--page-padding);
        padding-bottom: 0;
        border-bottom: 1px solid var(--border);
        background: var(--surface);
    }

    .header-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: var(--section-spacing);
    }

    .header h1 {
        font-family: var(--font-family);
        font-size: var(--page-title-size);
        font-weight: var(--page-title-weight);
        color: var(--text-primary);
        margin: 0;
        letter-spacing: var(--page-title-tracking);
    }

    .back-btn {
        background: transparent;
        border: 1px solid var(--border);
        padding: 8px 16px;
        border-radius: var(--border-radius-sm);
        font-size: 13px;
        font-family: var(--font-family);
        color: var(--text-secondary);
        cursor: pointer;
        transition: all var(--transition-fast);
    }

    .back-btn:hover {
        background: var(--surface-elevated);
        color: var(--text-primary);
        border-color: var(--onyx);
    }

    .tabs {
        display: flex;
        gap: 4px;
    }

    .tab {
        padding: 10px 20px;
        background: transparent;
        border: none;
        border-bottom: 2px solid transparent;
        font-family: var(--font-family);
        font-size: 13px;
        font-weight: 500;
        color: var(--text-muted);
        cursor: pointer;
        transition: all var(--transition-fast);
    }

    .tab:hover {
        color: var(--text-primary);
    }

    .tab.active {
        color: var(--onyx);
        font-weight: 600;
        border-bottom-color: var(--carbon);
    }

    .content {
        padding: 0;
    }

    @media (max-width: 768px) {
        .header {
            padding: 16px 16px 0;
        }

        .header h1 {
            font-size: 20px;
        }

        .header-row {
            flex-direction: column;
            align-items: flex-start;
            gap: 12px;
        }
    }
</style>
