<script lang="ts">
    import { run } from 'svelte/legacy';

    /**
     * CostingSheet - Intelligence-Enhanced
     * ABB competition warnings, margin alerts, approval workflow
     */
    import { createEventDispatcher } from "svelte";
    import { devLog } from "$lib/utils/devLog";
    interface Props {
        items?: any;
        currency?: string;
        showGenerateButton?: boolean;
        generating?: boolean;
        title?: string;
        warningThreshold?: number;
        abbCompeting?: boolean; // NEW: ABB competition flag
        paymentGrade?: string; // NEW: Customer payment grade
    }

    let {
        items = $bindable([]), currency = "BHD", showGenerateButton = true, generating = false, title = "Costing Sheet", warningThreshold = 15, abbCompeting = false, paymentGrade = 'B'
    }: Props = $props();
    
    // ABB Competition Rules from SSOT:
    // "IF ABB_is_competing: IF margin < 10%: DO_NOT_COMPETE"
    const ABB_MIN_MARGIN = 10;
    
    // Payment grade discount rules from SSOT
    const maxDiscountByGrade = {
        A: 7, // "MAX_DISCOUNT = 7%"
        B: 3, // "MAX_DISCOUNT = 3%"
        C: 0, // "MAX_DISCOUNT = 0%, REQUIRE 50% advance"
        D: 0, // "DECLINE_TO_QUOTE or 100% advance"
    };

    // TODO: These functions don't exist yet in the backend
    // import { CalculateCosting, GenerateQuote } from "../../../../wailsjs/go/main/App";

    const dispatch = createEventDispatcher();

    // Local state for the sheet
    let lineItems = $state(items.length > 0 ? items : [
        { description: "Item 1", quantity: 1, unitCost: 0, margin: 20 }
    ]);


    let subtotal = $state(0);
    let total = $state(0);
    let profit = $state(0);
    let marginPercent = $state(0);
    let quoteStatus = $state("");
    let lastSummary = null;


    async function updateTotals() {
        if (lineItems.length === 0) return;

        try {
            // Calculate locally since CalculateCosting doesn't exist yet
            subtotal = 0;
            total = 0;
            profit = 0;

            for (const item of lineItems) {
                const qty = parseFloat(item.quantity) || 0;
                const cost = parseFloat(item.unitCost) || 0;
                const margin = parseFloat(item.margin) || 0;

                const itemCost = qty * cost;
                const itemPrice = itemCost / (1 - margin / 100);
                const itemProfit = itemPrice - itemCost;

                subtotal += itemCost;
                total += itemPrice;
                profit += itemProfit;
            }

            marginPercent = subtotal > 0 ? (profit / total) * 100 : 0;

            const summary = { subtotal, total, profit, marginPercent };
            lastSummary = summary;
            dispatch("summary", { summary, items: lineItems });
        } catch (e) {
            devLog.error("Calculation failed:", e);
        }
    }

    function addItem() {
        lineItems = [...lineItems, { description: "New Item", quantity: 1, unitCost: 0, margin: 20 }];
    }

    function removeItem(index) {
        lineItems = lineItems.filter((_, i) => i !== index);
    }

    function updateItem(index, key, value) {
        const newItems = [...lineItems];
        newItems[index][key] = value;
        lineItems = newItems;
    }

    async function handleGenerateQuote() {
        quoteStatus = "Generating PDF...";
        const backendItems = lineItems.map(item => ({
            description: item.description,
            quantity: parseFloat(item.quantity) || 0,
            unitCost: parseFloat(item.unitCost) || 0,
            margin: parseFloat(item.margin) || 0
        }));
        dispatch("generate", { items: backendItems, summary: lastSummary });

        try {
            // TODO: Implement GenerateQuote when backend function is available
            devLog.log('Generate quote:', backendItems);
            quoteStatus = "GenerateQuote not yet implemented";
        } catch (e) {
            quoteStatus = "Error: " + e;
        }
    }
    run(() => {
        items = lineItems;
    });
    // Reactive update using backend
    run(() => {
        if (lineItems) {
            updateTotals();
        }
    });
</script>

<div class="bg-[var(--bg-color)] border border-[var(--border-color)] p-6 rounded-lg shadow-sm h-full flex flex-col">
    <div class="flex justify-between items-center mb-6 pb-2 border-b border-[var(--border-color)]">
        <h3 class="text-xl font-serif opacity-80">{title}</h3>
        <div class="flex gap-4 text-sm font-mono">
            <div>
                <span class="opacity-50">CURRENCY</span>
                <span class="font-bold ml-1">{currency}</span>
            </div>
            <div>
                <span class="opacity-50">MARGIN</span>
                <span class="font-bold ml-1" style="color: {marginPercent < warningThreshold ? 'var(--danger-color)' : 'var(--safe-color)'}">{marginPercent.toFixed(1)}%</span>
            </div>
        </div>
    </div>

    <div class="flex-1 overflow-auto">
        <table class="w-full text-left text-sm">
            <thead>
                <tr class="opacity-50 font-mono text-xs uppercase tracking-wider border-b border-[var(--border-color)]">
                    <th class="pb-2 w-1/2">Description</th>
                    <th class="pb-2 text-right">Qty</th>
                    <th class="pb-2 text-right">Unit Cost</th>
                    <th class="pb-2 text-right">Margin %</th>
                    <th class="pb-2 text-right">Total Price</th>
                    <th class="pb-2 w-8"></th>
                </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border-color)]">
                {#each lineItems as item, i}
                    <tr class="group">
                        <td class="py-2 pr-2">
                            <input
                                type="text"
                                value={item.description}
                                oninput={(e) => updateItem(i, 'description', e.currentTarget.value)}
                                class="w-full bg-transparent focus:outline-none focus:text-[var(--accent-color)] placeholder-opacity-30"
                                placeholder="Item description..."
                            />
                        </td>
                        <td class="py-2 text-right">
                            <input
                                type="number"
                                value={item.quantity}
                                oninput={(e) => updateItem(i, 'quantity', parseFloat(e.currentTarget.value))}
                                class="w-16 bg-transparent text-right focus:outline-none focus:text-[var(--accent-color)]"
                            />
                        </td>
                        <td class="py-2 text-right">
                             <input
                                type="number"
                                value={item.unitCost}
                                oninput={(e) => updateItem(i, 'unitCost', parseFloat(e.currentTarget.value))}
                                class="w-24 bg-transparent text-right focus:outline-none focus:text-[var(--accent-color)]"
                            />
                        </td>
                        <td class="py-2 text-right">
                            <input
                                type="number"
                                value={item.margin}
                                oninput={(e) => updateItem(i, 'margin', parseFloat(e.currentTarget.value))}
                                class="w-16 bg-transparent text-right focus:outline-none focus:text-[var(--accent-color)]"
                            />
                        </td>
                        <td class="py-2 text-right font-mono opacity-80">
                            {((item.quantity * item.unitCost) * (1 + item.margin/100)).toFixed(2)}
                        </td>
                        <td class="py-2 text-right">
                            <button
                                onclick={() => removeItem(i)}
                                class="opacity-0 group-hover:opacity-50 hover:!opacity-100 text-[var(--danger-color)] transition-opacity"
                            >
                                ×
                            </button>
                        </td>
                    </tr>
                {/each}
            </tbody>
        </table>

        <button
            onclick={addItem}
            class="mt-4 text-xs font-mono uppercase tracking-widest opacity-50 hover:opacity-100 hover:text-[var(--accent-color)] transition-colors flex items-center gap-2"
        >
            <span>+ Add Line Item</span>
        </button>
    </div>

    <div class="mt-6 pt-4 border-t border-[var(--border-color)] bg-[var(--bg-color)]">
        <div class="flex justify-end gap-12">
            <div class="text-right">
                <div class="text-xs font-mono opacity-50 uppercase">Total Cost</div>
                <div class="text-lg font-serif">{subtotal.toFixed(2)} <span class="text-sm opacity-50">{currency}</span></div>
            </div>
            <div class="text-right">
                <div class="text-xs font-mono opacity-50 uppercase">Profit</div>
                <div class="text-lg font-serif text-[var(--safe-color)]">{profit.toFixed(2)} <span class="text-sm opacity-50">{currency}</span></div>
            </div>
            <div class="text-right">
                <div class="text-xs font-mono opacity-50 uppercase">Final Price</div>
                <div class="text-2xl font-serif font-bold">{total.toFixed(2)} <span class="text-sm opacity-50">{currency}</span></div>
            </div>
        </div>

        <!-- Intelligence Warnings -->
        {#if abbCompeting && marginPercent < ABB_MIN_MARGIN}
            <div class="mt-3 p-3 bg-red-50 border border-red-200 rounded text-sm">
                <div class="flex items-center gap-2 text-red-700 font-bold">
                    <span>STOP - DO NOT COMPETE</span>
                </div>
                <p class="text-red-600 text-xs mt-1 font-mono">
                    ABB is competing and margin is below 10%. Per SSOT rules: focus on service value or walk away.
                </p>
            </div>
        {:else if abbCompeting}
            <div class="mt-3 p-3 bg-amber-50 border border-amber-200 rounded text-sm">
                <div class="flex items-center gap-2 text-amber-700 font-bold">
                    <span>WARNING - ABB COMPETITION DETECTED</span>
                </div>
                <p class="text-amber-600 text-xs mt-1 font-mono">
                    Focus on service value, emergency delivery, and technical support advantages.
                </p>
            </div>
        {/if}

        {#if paymentGrade === 'D'}
            <div class="mt-3 p-3 bg-red-50 border border-red-200 rounded text-sm">
                <div class="flex items-center gap-2 text-red-700 font-bold">
                    <span>STOP - D-GRADE CUSTOMER</span>
                </div>
                <p class="text-red-600 text-xs mt-1 font-mono">
                    REQUIRE 100% ADVANCE PAYMENT or decline to quote. Chase history: 6+ months.
                </p>
            </div>
        {:else if paymentGrade === 'C'}
            <div class="mt-3 p-3 bg-amber-50 border border-amber-200 rounded text-sm">
                <div class="flex items-center gap-2 text-amber-700 font-bold">
                    <span>WARNING - C-GRADE CUSTOMER</span>
                </div>
                <p class="text-amber-600 text-xs mt-1 font-mono">
                    No discount allowed. Require 50% advance payment.
                </p>
            </div>
        {/if}

        {#if marginPercent < warningThreshold}
            <div class="mt-3 text-xs font-mono text-[var(--danger-color)] text-right">
                Low margin detected — approval will be required before sending.
            </div>
        {/if}

        {#if quoteStatus}
            <div class="mt-4 text-right text-xs font-mono {quoteStatus.includes('Error') ? 'text-[var(--danger-color)]' : 'text-[var(--safe-color)]'}">
                {quoteStatus}
            </div>
        {/if}

        {#if showGenerateButton}
            <div class="mt-6 flex justify-end gap-4">
                <button
                    onclick={handleGenerateQuote}
                    class="px-6 py-2 rounded bg-[var(--text-color)] text-[var(--bg-color)] text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-50"
                    disabled={generating}
                >
                    {generating ? 'Working…' : 'Generate Quote'}
                </button>
            </div>
        {/if}
    </div>
</div>
