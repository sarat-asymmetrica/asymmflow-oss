<script lang="ts">
    /**
     * Wave 9.5 B4 — canonical line-item editor (Article VI.3: one shared
     * component, screens compose).
     *
     * This component is PRESENTATION ONLY. All calculation (the suggested-vs-
     * override pricing and the landed-cost waterfall) stays owned by the
     * consuming screen — this component reads already-computed fields off each
     * item and calls back into the parent (via the callback props below) to
     * trigger recalculation. It never computes cost/markup/pricing itself.
     *
     * `mode="costing"` is the full FOB -> landed-cost -> markup -> suggested-
     * price waterfall editor, extracted verbatim from CostingSheetScreen (its
     * first, and currently only, consumer).
     *
     * `mode="order"` (Wave 9.6 B2) is the simpler code/description/qty/
     * unit-price/line-total editor, extracted verbatim from OrdersScreen's
     * inline formItems editor (its first, and currently only, consumer).
     *
     * `items` is expected to be the SAME reactive array reference the parent
     * owns (e.g. a $state array) — this component mutates fields on it via
     * bind:value the same way the original inline markup did, so parent-owned
     * reactivity (including draft-persistence / dirty-tracking that listens on
     * the DOM via bubbling oninput/onchange on an ancestor element) keeps
     * working unchanged as long as this component is rendered inside that
     * ancestor.
     */
    import Button from './Button.svelte';
    import Card from './Card.svelte';

    interface Props {
        mode?: 'costing' | 'order';
        items: any[];
        currencyOptions?: { code: string; rate: number }[];
        maxItems?: number;
        cardTitle?: string;

        // Formatting (pure, no math) — owned by the parent, just referenced here.
        formatBHD: (val: unknown) => string;
        formatPercent?: (val: unknown) => string;
        formatNumber?: (val: unknown, decimals?: number) => string;

        // Callbacks — the parent owns and performs all calculation; this
        // component only asks the parent to recalculate/act.
        onRecalculate: () => void;
        onCurrencyChange?: (index: number) => void;
        onUserPriceInput?: (item: any) => void;
        onFreightPercentInput?: () => void;
        onRemoveItem: (index: number) => void;
        onCopyFirstToAll?: () => void;
        onAddItem: () => void;
    }

    let {
        mode = 'costing',
        items,
        currencyOptions = [],
        maxItems = 100,
        cardTitle = 'Line Items',
        formatBHD,
        formatPercent = (val: unknown) => `${Number(val) || 0}%`,
        formatNumber = (val: unknown, decimals = 0) =>
            (Number(val) || 0).toLocaleString('en-US', { minimumFractionDigits: decimals, maximumFractionDigits: decimals }),
        onRecalculate,
        onCurrencyChange,
        onUserPriceInput,
        onFreightPercentInput,
        onRemoveItem,
        onCopyFirstToAll,
        onAddItem,
    }: Props = $props();
</script>

<Card title={cardTitle}>
    {#if mode === 'costing'}
        {#each items as item, i}
            <div class="line-item-group" class:alt-row={i % 2 === 1}>
            <!-- Line 1: Product identification -->
            <div class="line-row-top">
                <span class="sl-no">{i + 1}</span>
                <div class="field-inline">
                    <span class="field-label">Equipment</span>
                    <input type="text" bind:value={item.equipment} class="input-sm" placeholder="Product name" />
                </div>
                <div class="field-inline">
                    <span class="field-label">Model</span>
                    <input type="text" bind:value={item.model} class="input-sm" placeholder="Model #" />
                </div>
                <div class="field-inline compact">
                    <span class="field-label">Qty</span>
                    <input type="number" bind:value={item.quantity} class="input-sm qty" min="1" onchange={onRecalculate} />
                </div>
                <div class="field-inline compact">
                    <span class="field-label">Currency</span>
                    <select bind:value={item.currency} class="input-sm" onchange={() => onCurrencyChange?.(i)}>
                        {#each currencyOptions as c}
                            <option value={c.code}>{c.code}</option>
                        {/each}
                    </select>
                </div>
                <div class="field-inline exchange-rate">
                    <span class="field-label">Exch. Rate</span>
                    <input type="number" bind:value={item.exchangeRate} class="input-sm money" step="0.0001" min="0.0001" onchange={onRecalculate} />
                </div>
                {#if items.length > 1}
                    <button class="btn-remove" onclick={() => onRemoveItem(i)} title="Remove item" aria-label={`Remove line item ${i + 1}`}>×</button>
                {:else}
                    <span class="remove-slot" aria-hidden="true"></span>
                {/if}
            </div>

            <!-- Line 2: Pricing -->
            <div class="line-row-pricing">
                <div class="field-inline">
                    <span class="field-label">Unit Price (FOB)</span>
                    <input type="number" bind:value={item.fobForeign} class="input-sm money" step="0.001" onchange={onRecalculate} placeholder="0.000" />
                </div>
                <div class="field-inline">
                    <span class="field-label">Freight %</span>
                    <input type="number" bind:value={item.freightPercent} class="input-sm money" step="0.01" min="0" oninput={() => onFreightPercentInput?.()} onchange={onRecalculate} placeholder="0.00" />
                </div>
                <div class="field-inline">
                    <span class="field-label">Extra Cost</span>
                    <input type="number" bind:value={item.otherCosts} class="input-sm money" step="0.001" min="0" onchange={onRecalculate} placeholder="0.000" title="Internal extra cost added before margin" />
                </div>
                <div class="field-inline">
                    <span class="field-label">Manual Unit Price</span>
                    <input
                        type="number"
                        class="input-sm money sell-price"
                        class:user-overridden={item.userPriceSet && item.userPrice > 0}
                        step="0.001"
                        placeholder={`Auto ${formatNumber(item.suggestedPriceUnit, 0)}`}
                        bind:value={item.userPrice}
                        oninput={() => onUserPriceInput?.(item)}
                        title={`Optional unit selling price override. Leave blank or 0 to use the suggested price: ${formatBHD(item.suggestedPriceUnit)}`}
                    />
                </div>
                <div class="line-total">
                    <span class="field-label">Total (BHD)</span>
                    <span class="calc-value highlight">{formatBHD(item.totalSuggestedPrice)}</span>
                </div>
            </div>

            <!-- Order Code - full width input for long supplier order/configuration codes -->
            <div class="long-code-row">
                <span class="long-code-label">Long Code:</span>
                <input
                    type="text"
                    bind:value={item.longCode}
                    class="long-code-input"
                    placeholder="e.g. FMU90-R11CA131AA3A  /  71452626+Z1C5+Z1E3+Z1F1+Z1G1..."
                />
            </div>

            <!-- Wide Detailed Description Field - for instrumentation specifications -->
            <div class="detailed-description-row">
                <textarea
                    bind:value={item.detailedDescription}
                    class="detailed-description-input"
                    rows="2"
                    placeholder="Detailed specs, approvals, HS codes, country of origin... (prints under line item in PDF)"
                ></textarea>
            </div>

            <!-- Expanded cost breakdown -->
            <div class="cost-breakdown">
                <div class="cost-row">
                    <span class="cost-label">Rate: {item.exchangeRate}</span>
                    <span class="cost-label">FOB: {formatBHD(item.fobBHD)}</span>
                    <span class="cost-label">Freight %: {formatPercent(item.freightPercent)}</span>
                    <span class="cost-label">Freight: {formatBHD(item.freightBHD)}</span>
                    <span class="cost-label">C&F: {formatBHD(item.cf)}</span>
                </div>
                <div class="cost-row">
                    <div class="cost-input">
                        <span class="field-label">Customs %</span>
                        <input type="number" bind:value={item.customsPercent} class="input-xs" onchange={onRecalculate} />
                    </div>
                    <div class="cost-input">
                        <span class="field-label">Handling %</span>
                        <input type="number" bind:value={item.handlingPercent} class="input-xs" onchange={onRecalculate} />
                    </div>
                    <div class="cost-input">
                        <span class="field-label">Finance %</span>
                        <input type="number" bind:value={item.financePercent} class="input-xs" onchange={onRecalculate} />
                    </div>
                    <div class="cost-input">
                        <span class="field-label">Markup %</span>
                        <input type="number" bind:value={item.marginPercent} class="input-xs" min="0" max="99" onchange={onRecalculate} />
                    </div>
                    <div class="cost-input">
                        <span class="field-label">Total Cost</span>
                        <span class="cost-value">{formatBHD(item.totalCost)}</span>
                    </div>
                    <!-- Issue #7: Copy costs button - only on first item -->
                    {#if i === 0 && items.length > 1}
                        <button
                            class="btn-copy-costs"
                            onclick={() => onCopyFirstToAll?.()}
                            title="Copy customs, handling, finance, and margin to all items below"
                        >
                            Copy to All
                        </button>
                    {/if}
                </div>
                <div class="cost-row">
                    <span class="cost-label">Customs: {formatBHD(item.customsBHD)}</span>
                    <span class="cost-label">Landed: {formatBHD(item.landedCost)}</span>
                    <span class="cost-label">Handling: {formatBHD(item.handlingBHD)}</span>
                    <span class="cost-label">Markup: {formatBHD(item.marginBHD)}</span>
                    <span class="cost-label suggested-highlight">Suggested: {formatBHD(item.suggestedPriceUnit)}</span>
                </div>
            </div>
            </div>
        {/each}

        <div class="add-item-row">
            <Button variant="secondary" size="sm" on:click={onAddItem}>+ Add Line Item</Button>
            <span class="item-count">{items.length}/{maxItems} items</span>
        </div>
    {:else}
        <!-- Wave 9.6 B2: mode="order" — wired to OrdersScreen's inline
             editor. Fields/markup extracted verbatim from OrdersScreen's
             original {#each formItems} block (product_code/description/
             quantity/unit_price_bhd + line total). ALL calculation
             (sanitizeFormItems / recalculateFormItems / roundMoney /
             handleSubmit's authoritative total) stays owned by OrdersScreen
             — this only renders the already-computed item.total_price and
             calls back via onRecalculate on blur (code/description) or
             oninput (qty/unit price), matching the original event
             granularity so the live per-keystroke total is preserved. -->
        <div class="line-items-list">
            {#each items as item, i}
                <div class="line-item-editor">
                    <div class="line-item-editor-head">
                        <span>Line {i + 1}</span>
                        <button
                            type="button"
                            class="line-remove-btn"
                            onclick={() => onRemoveItem(i)}
                            disabled={items.length <= 1}
                            aria-label="Remove order line"
                        >
                            Remove
                        </button>
                    </div>

                    <div class="line-item-top-row">
                        <label class="line-field">
                            <span>Code</span>
                            <input
                                class="form-input compact"
                                bind:value={item.product_code}
                                placeholder="Code"
                                onblur={onRecalculate}
                            />
                        </label>
                        <label class="line-field description-field">
                            <span>Description</span>
                            <input
                                class="form-input compact"
                                bind:value={item.description}
                                placeholder="Description"
                                onblur={onRecalculate}
                            />
                        </label>
                    </div>

                    <div class="line-item-bottom-row">
                        <label class="line-field">
                            <span>Quantity</span>
                            <input
                                class="form-input compact number-input"
                                type="number"
                                min="0"
                                step="0.001"
                                bind:value={item.quantity}
                                oninput={onRecalculate}
                            />
                        </label>
                        <label class="line-field">
                            <span>Unit Price</span>
                            <input
                                class="form-input compact number-input"
                                type="number"
                                min="0"
                                step="0.001"
                                bind:value={item.unit_price_bhd}
                                oninput={onRecalculate}
                            />
                        </label>
                        <div class="line-total-cell">
                            <span>Line Total</span>
                            <strong>{formatBHD(item.total_price)}</strong>
                        </div>
                    </div>
                </div>
            {/each}
        </div>

        <div class="add-item-row">
            <Button variant="secondary" size="sm" on:click={onAddItem}>+ Add Line Item</Button>
            <span class="item-count">{items.length}/{maxItems} items</span>
        </div>
    {/if}
</Card>

<style>
    /* Line item inputs — mirrors CostingSheetScreen's .input-sm / .input-xs so
       this component renders pixel-identical to the pre-extraction markup.
       (Svelte scoped styles don't cascade into child components, so these are
       intentionally duplicated rather than shared with the parent screen.) */
    .input-sm {
        padding: 6px 8px;
        border: 1px solid var(--border);
        border-radius: var(--border-radius-sm);
        font-size: 12px;
        background: var(--bg-base);
        width: 100%;
        min-width: 0;
        box-sizing: border-box;
        min-height: 32px;
        height: 32px;
        line-height: 1.2;
    }

    .input-xs {
        padding: 4px 6px;
        border: 1px solid var(--border);
        border-radius: var(--border-radius-sm);
        font-size: 11px;
        width: 60px;
        background: var(--bg-base);
    }

    .input-sm.qty {
        width: 100%;
        text-align: center;
    }

    .input-sm.money {
        width: 100%;
        text-align: right;
        font-family: var(--font-mono, 'JetBrains Mono', monospace);
    }

    .input-sm.sell-price {
        background: #f0f4ff;
        border-color: var(--brand-indigo);
        font-weight: 500;
    }

    .input-sm.sell-price:focus {
        background: #fff;
    }

    .line-row-top {
        display: grid;
        grid-template-columns: 35px minmax(220px, 2.2fr) minmax(140px, 1.2fr) 68px 88px 132px 28px;
        gap: 8px;
        padding: 8px 10px 2px;
        align-items: end;
        background: var(--surface-elevated);
        border-radius: var(--border-radius-sm) var(--border-radius-sm) 0 0;
        border-left: 3px solid transparent;
    }

    .line-row-pricing {
        display: grid;
        grid-template-columns: 1fr 1fr 1fr 1fr 130px;
        gap: 8px;
        padding: 2px 10px 8px;
        margin-left: 43px;
        align-items: end;
        background: var(--surface-elevated);
        border-radius: 0 0 var(--border-radius-sm) var(--border-radius-sm);
    }

    .field-inline {
        display: flex;
        flex-direction: column;
        gap: 1px;
    }

    .field-inline.compact {
        max-width: 88px;
    }

    .field-inline.exchange-rate {
        min-width: 132px;
    }

    .field-inline .field-label {
        font-size: 9px;
        font-weight: 600;
        color: var(--text-tertiary, #9ca3af);
        text-transform: uppercase;
        letter-spacing: 0.3px;
    }

    .line-total {
        display: flex;
        flex-direction: column;
        align-items: flex-end;
        gap: 1px;
    }

    .line-total .field-label {
        font-size: 9px;
        font-weight: 600;
        color: var(--text-tertiary, #9ca3af);
        text-transform: uppercase;
    }

    .line-item-group.alt-row .line-row-top,
    .line-item-group.alt-row .line-row-pricing {
        background: rgba(99, 102, 241, 0.04);
    }

    .line-item-group {
        margin-bottom: 4px;
    }

    .sl-no {
        font-size: 12px;
        font-weight: 600;
        color: var(--text-secondary);
        text-align: center;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    /* Order code row - full width for long supplier configuration codes */
    .long-code-row {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 4px 10px 4px 10px;
    }

    .long-code-label {
        font-size: 10px;
        font-weight: 600;
        color: var(--text-secondary);
        text-transform: uppercase;
        letter-spacing: 0.03em;
        white-space: nowrap;
        min-width: 80px;
    }

    .long-code-input {
        flex: 1;
        padding: 5px 10px;
        border: 1px solid var(--border);
        border-radius: var(--border-radius-sm);
        font-size: 12px;
        font-family: 'SF Mono', 'Fira Code', monospace;
        background: var(--bg-base);
        letter-spacing: 0.02em;
    }

    .long-code-input:focus {
        outline: none;
        border-color: var(--brand-indigo);
        box-shadow: 0 0 0 2px var(--brand-indigo-tint);
    }

    .long-code-input::placeholder {
        font-family: inherit;
        opacity: 0.4;
    }

    /* Wide detailed description field for instrumentation specs */
    .detailed-description-row {
        padding: 0 10px 8px 10px;
        margin-bottom: 8px;
        border-bottom: 1px solid var(--border);
    }

    .detailed-description-input {
        width: 100%;
        padding: 10px 12px;
        font-size: 12px;
        font-family: var(--font-mono, 'JetBrains Mono', monospace);
        line-height: 1.5;
        border: 1px solid var(--border);
        border-radius: var(--border-radius-sm);
        background: var(--surface);
        color: var(--text-primary);
        resize: vertical;
        min-height: 60px;
    }

    .detailed-description-input:focus {
        outline: none;
        border-color: var(--brand-indigo, #4F46E5);
        box-shadow: 0 0 0 2px rgba(79, 70, 229, 0.1);
    }

    .detailed-description-input::placeholder {
        color: var(--text-muted);
        font-style: italic;
    }

    /* User override highlight */
    .sell-price.user-overridden {
        background: rgba(79, 70, 229, 0.1);
        border-color: var(--brand-indigo, #4F46E5);
    }

    .calc-value {
        font-size: 12px;
        font-family: var(--font-mono, 'JetBrains Mono', monospace);
        color: var(--text-secondary);
        text-align: right;
    }

    .calc-value.highlight {
        color: var(--brand-indigo);
        font-weight: 600;
    }

    .btn-remove {
        background: transparent;
        border: none;
        color: #ef4444;
        cursor: pointer;
        font-size: 18px;
        padding: 0;
        width: 24px;
        height: 24px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 4px;
    }

    .btn-remove:hover {
        background: rgba(239, 68, 68, 0.1);
    }

    .remove-slot {
        width: 24px;
        height: 24px;
        display: block;
    }

    .cost-breakdown {
        padding: 8px 12px 12px;
        margin-bottom: 12px;
        background: var(--bg-base);
        border: 1px dashed var(--border);
        border-radius: var(--border-radius-sm);
        font-size: 11px;
    }

    .cost-row {
        display: flex;
        gap: 16px;
        margin-bottom: 8px;
        flex-wrap: wrap;
    }

    .cost-label {
        color: var(--text-secondary);
    }

    .cost-label.suggested-highlight {
        color: var(--brand-indigo, #4F46E5);
        font-weight: 600;
        background: rgba(79, 70, 229, 0.08);
        padding: 2px 8px;
        border-radius: 4px;
    }

    .cost-input {
        display: flex;
        align-items: center;
        gap: 4px;
    }

    .cost-input .field-label {
        font-size: 10px;
        color: var(--text-muted);
    }

    /* Issue #7: Copy costs button styling */
    .btn-copy-costs {
        background: var(--brand-indigo, #6366f1);
        color: white;
        border: none;
        padding: 4px 10px;
        border-radius: 4px;
        font-size: 0.75rem;
        cursor: pointer;
        white-space: nowrap;
        transition: background 0.15s;
        margin-left: auto;
    }

    .btn-copy-costs:hover {
        background: var(--brand-indigo-hover, #4f46e5);
    }

    .add-item-row {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px;
        border-top: 1px solid var(--border);
        margin-top: 8px;
    }

    .item-count {
        font-size: 12px;
        color: var(--text-secondary);
    }

    /* FIX #23: Disable hover animation "dance" on costing inputs — mirrors
       the parent screen's rule for the fields this component owns. */
    .cost-breakdown input:hover,
    .cost-breakdown input:focus,
    .input-sm:hover,
    .input-sm:focus,
    .input-xs:hover,
    .input-xs:focus {
        transition: none !important;
        transform: none !important;
    }

    .cost-breakdown:hover {
        transform: none !important;
        transition: none !important;
    }

    /* Wave 9.6 B2 — order-mode fields, duplicated verbatim from
       OrdersScreen's original inline editor styles (Svelte scoped styles
       don't cascade into child components — same reasoning as the
       costing-mode .input-sm/.input-xs duplication above). */
    .line-items-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .line-item-editor {
        background: var(--surface);
        border: 1px solid var(--border-subtle);
        border-radius: var(--radius-sm);
        padding: 12px;
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .line-item-editor-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        font-size: 11px;
        font-weight: 700;
        text-transform: uppercase;
        color: var(--text-secondary);
    }

    .line-item-top-row {
        display: grid;
        grid-template-columns: minmax(140px, 0.45fr) minmax(320px, 1fr);
        gap: 12px;
    }

    .line-item-bottom-row {
        display: grid;
        grid-template-columns: minmax(120px, 0.35fr) minmax(150px, 0.45fr) minmax(160px, 0.5fr);
        gap: 12px;
        align-items: end;
    }

    .line-field {
        display: flex;
        flex-direction: column;
        gap: 5px;
        min-width: 0;
    }

    .line-field span,
    .line-total-cell span {
        font-size: 10px;
        font-weight: 700;
        text-transform: uppercase;
        color: var(--text-secondary);
    }

    .compact {
        min-width: 0;
    }

    .number-input {
        text-align: right;
        font-family: var(--font-mono);
    }

    .line-total-cell {
        min-height: 36px;
        padding: 8px 10px;
        border: 1px solid var(--border-subtle);
        border-radius: var(--radius-sm);
        background: var(--bg-subtle);
        display: flex;
        flex-direction: column;
        justify-content: center;
        gap: 2px;
        text-align: right;
        font-size: 12px;
        font-family: var(--font-mono);
        color: var(--text-primary);
    }

    .line-total-cell strong {
        font-size: 13px;
        font-weight: 700;
    }

    .line-remove-btn {
        border: 1px solid var(--border);
        background: transparent;
        color: var(--text-secondary);
        border-radius: var(--radius-sm);
        padding: 8px 10px;
        font-size: 12px;
        cursor: pointer;
    }

    .line-remove-btn:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }

    .form-input {
        width: 100%;
        padding: 8px 12px;
        border: 1px solid var(--border);
        border-radius: var(--radius-md);
        font-size: 13px;
        outline: none;
        transition: border-color 0.2s;
        font-family: var(--font-sans);
    }

    .form-input:focus {
        border-color: var(--primary);
    }

    @media (max-width: 840px) {
        .line-item-top-row,
        .line-item-bottom-row {
            grid-template-columns: 1fr;
        }

        .line-total-cell {
            text-align: left;
        }
    }
</style>
