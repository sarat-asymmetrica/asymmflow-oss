<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { SearchProducts } from "../../../wailsjs/go/main/App";
    import { fade } from "svelte/transition";
    import { motionMs } from "../motion";
    import { formatNumber } from "$lib/utils/formatters";

    interface Props {
        value?: string;
        placeholder?: string;
    }

    let { value = $bindable(""), placeholder = "Search products..." }: Props = $props();

    const dispatch = createEventDispatcher();

    let results = $state([]);
    let showResults = $state(false);
    let loading = $state(false);
    let searchTimeout;

    async function handleInput(e) {
        const query = e.target.value;
        console.log("ProductSearch input:", query);
        value = query;
        
        // Debounce search
        clearTimeout(searchTimeout);
        if (query.length < 2) {
            console.log("Query too short, waiting...");
            results = [];
            showResults = false;
            return;
        }

        searchTimeout = setTimeout(async () => {
            loading = true;
            console.log("Triggering SearchProducts for:", query);
            try {
                // Call VQC-ready search engine
                results = await SearchProducts(query);
                console.log("Search results:", results);
                showResults = results && results.length > 0;
            } catch (err) {
                console.error("Search failed", err);
            } finally {
                loading = false;
            }
        }, 300);
    }

    function selectProduct(product) {
        value = product.product_code || product.ProductCode; // Display code in input
        showResults = false;
        dispatch("select", product);
    }

    function handleBlur() {
        // Delay hide to allow click registration
        setTimeout(() => {
            showResults = false;
        }, 200);
    }
</script>

<div class="product-search">
    <div class="input-wrapper">
        <input
            type="text"
            bind:value
            {placeholder}
            oninput={handleInput}
            onfocus={() => value.length >= 2 && (showResults = true)}
            onblur={handleBlur}
            class:has-results={showResults}
        />
        {#if loading}
            <div class="spinner"></div>
        {/if}
    </div>

    {#if showResults}
        <div class="results-dropdown" transition:fade={{ duration: motionMs(100) }}>
            {#each results as product}
                <button class="result-item" onclick={() => selectProduct(product)}>
                    <div class="code">{product.product_code || product.ProductCode}</div>
                    <div class="name">{product.product_name || product.ProductName}</div>
                    <div class="meta">
                        <span class="price">BHD {formatNumber(product.standard_cost_bhd || product.StandardCostBHD || 0, 2)}</span>
                        <span class="supplier">{product.supplier_code || product.SupplierCode}</span>
                    </div>
                </button>
            {/each}
        </div>
    {/if}
</div>

<style>
    .product-search {
        position: relative;
        width: 100%;
    }

    input {
        width: 100%;
        padding: 0.5rem;
        border: 1px solid rgba(0, 0, 0, 0.2);
        border-radius: 4px;
        font-size: 0.9rem;
        font-family: var(--font-sans);
    }

    input:focus {
        outline: none;
        border-color: var(--color-ink);
        box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.05);
    }

    .spinner {
        position: absolute;
        right: 10px;
        top: 50%;
        transform: translateY(-50%);
        width: 12px;
        height: 12px;
        border: 2px solid rgba(0, 0, 0, 0.1);
        border-top-color: var(--color-ink);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    .results-dropdown {
        position: absolute;
        top: 100%;
        left: 0;
        right: 0;
        background: #ffffff;
        border: 1px solid rgba(0, 0, 0, 0.15);
        border-radius: 4px;
        box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
        max-height: 250px;
        overflow-y: auto;
        z-index: 9999;
        margin-top: 2px;
    }

    .result-item {
        display: block;
        width: 100%;
        padding: 8px 12px;
        text-align: left;
        border: none;
        background: none;
        border-bottom: 1px solid rgba(0, 0, 0, 0.05);
        cursor: pointer;
        transition: background 0.1s;
    }

    .result-item:last-child {
        border-bottom: none;
    }

    .result-item:hover {
        background: rgba(0, 0, 0, 0.03);
    }

    .code {
        font-family: var(--font-mono);
        font-size: 0.75rem;
        color: var(--color-ink);
        font-weight: bold;
    }

    .name {
        font-size: 0.85rem;
        margin: 2px 0;
        color: var(--color-ink);
    }

    .meta {
        display: flex;
        justify-content: space-between;
        font-size: 0.7rem;
        color: var(--color-ink-light);
    }

    @keyframes spin {
        to { transform: translateY(-50%) rotate(360deg); }
    }
</style>
