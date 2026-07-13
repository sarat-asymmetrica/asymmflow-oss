<script lang="ts">
    import { onMount } from "svelte";
    import { motionMs } from "$lib/motion";
    import { fade } from "svelte/transition";
    import { toast } from "../stores/toasts";
    import { confirm } from "../stores/confirm";
    import { devLog } from "$lib/utils/devLog";
    import { formatNumber } from "$lib/utils/formatters";
    import { RUNTIME_URL, API_ENDPOINTS } from "$lib/config";

    // State
    let selectedFile = $state("");
    let loading = $state(false);
    let previewHtml = $state("");
    let quotationData = $state(null);
    let pdfResult = $state(null);
    let recentQuotations = [];

    // Pricing
    let pricingRecommendation = $state(null);
    let loadingPricing = false;

    async function selectExcelFile() {
        if (window.go?.main?.App?.SelectExcelFile) {
            try {
                const path = await window.go.main.App.SelectExcelFile();
                if (path) {
                    selectedFile = path;
                    await previewQuotation();
                }
            } catch (e) {
                toast.danger("Dialog failed");
            }
        } else {
            const r = await confirm.askForReason({
                title: "Excel Costing File",
                message: "Enter the path to the Excel costing file to load.",
                reasonLabel: "Path to Excel costing file",
                reasonRequired: true
            });
            if (!r.confirmed) return;
            selectedFile = r.reason;
            await previewQuotation();
        }
    }

    async function previewQuotation() {
        if (!selectedFile) {
            toast.warning("Select file first");
            return;
        }
        loading = true;
        try {
            const response = await fetch(
                `${RUNTIME_URL}/api/quotation/from-excel`,
                {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ ExcelPath: selectedFile }),
                },
            );
            const result = await response.json();
            if (result.success) {
                previewHtml = result.html;
                quotationData = { ...result };
                toast.success("Loaded");
                fetchPricing(result.customer);
            } else {
                toast.danger(result.error);
            }
        } catch (e) {
            toast.danger("Connection failed");
        } finally {
            loading = false;
        }
    }

    async function fetchPricing(customer) {
        if (!customer) return;
        loadingPricing = true;
        try {
            const res = await fetch(`${RUNTIME_URL}/api/pricing/recommend`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    Customer: customer,
                    Items: [
                        {
                            ProductCode: "GEN",
                            UnitCost: quotationData?.subtotal || 1000,
                        },
                    ],
                }),
            });
            const data = await res.json();
            if (data.success) {
                pricingRecommendation = data;
                toast.info(`Analysis: ${data.customerRegime}`);
            }
        } catch (e) {
            console.error('Pricing recommendation failed:', e);
            toast.warning('Could not load pricing analysis');
        } finally {
            loadingPricing = false;
        }
    }

    async function generatePdf() {
        if (!selectedFile) return;
        loading = true;
        const fileName = selectedFile
            .split(/[/\\]/)
            .pop()
            .replace(".xlsx", ".pdf");
        const outputPath = `batch_output/quotation_pdfs/${fileName}`;

        try {
            const res = await fetch(`${RUNTIME_URL}/api/quotation/to-pdf`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    ExcelPath: selectedFile,
                    OutputPath: outputPath,
                }),
            });
            const result = await res.json();
            if (result.success) {
                pdfResult = result;
                recentQuotations = [
                    {
                        number: result.quotationNumber,
                        customer: result.customer,
                        total: result.total,
                        path: result.outputPath,
                    },
                    ...recentQuotations,
                ];
                toast.success("PDF Generated");
            } else {
                toast.danger(result.error);
            }
        } catch (e) {
            toast.danger("PDF Gen Failed");
        } finally {
            loading = false;
        }
    }

    async function openPdf() {
        if (pdfResult?.outputPath && window.go?.main?.App?.OpenFile) {
            await window.go.main.App.OpenFile(pdfResult.outputPath);
        }
    }

    function formatCurrency(val, curr = "BHD") {
        return `${curr} ${formatNumber(val || 0, 3)}`;
    }

    function getRegimeColor(regime) {
        return (
            {
                PriceSensitive: "#ef4444",
                ValueBalanced: "#f59e0b",
                Premium: "#15803d",
            }[regime] || "#6b7280"
        );
    }
</script>

<div class="page">
    <header class="header">
        <div class="header-content">
            <h1>Quotation.</h1>
            <p class="subtitle">Excel Processing & PDF Generation</p>
        </div>
        <div class="status-badge">Runtime Connected</div>
    </header>

    <div class="layout-split">
        <!-- Controls -->
        <aside class="sidebar">
            <div class="panel upload-panel">
                <h3>Source File (Excel)</h3>
                <div class="file-group">
                    <input
                        type="text"
                        value={selectedFile}
                        placeholder="No file selected..."
                        readonly
                    />
                    <button class="btn-ghost" onclick={selectExcelFile}
                        >Browse</button
                    >
                </div>
                <div class="actions">
                    <button
                        class="btn-ghost"
                        onclick={previewQuotation}
                        disabled={loading || !selectedFile}>Preview</button
                    >
                    <button
                        class="btn-primary"
                        onclick={generatePdf}
                        disabled={loading || !selectedFile}
                    >
                        {loading ? "Processing..." : "Generate PDF"}
                    </button>
                </div>
            </div>

            {#if quotationData}
                <div class="panel summary-panel" in:fade={{ duration: motionMs(400) }}>
                    <h3>Summary</h3>
                    <div class="row">
                        <span>Customer</span>
                        <strong>{quotationData.customer}</strong>
                    </div>
                    <div class="row">
                        <span>Reference</span>
                        <span class="mono">{quotationData.quotationNumber}</span
                        >
                    </div>
                    <div class="row">
                        <span>Items</span>
                        <span>{quotationData.lineItemCount}</span>
                    </div>
                    <div class="row total">
                        <span>Total</span>
                        <strong
                            >{formatCurrency(
                                quotationData.total,
                                quotationData.currency,
                            )}</strong
                        >
                    </div>
                </div>
            {/if}

            {#if pricingRecommendation}
                <div
                    class="panel pricing-panel"
                    in:fade={{ duration: motionMs(400) }}
                    style="border-color: {getRegimeColor(
                        pricingRecommendation.customerRegime,
                    )}"
                >
                    <div class="pricing-head">
                        <h3>Pricing Intelligence</h3>
                        <span
                            class="badge"
                            style="background: {getRegimeColor(
                                pricingRecommendation.customerRegime,
                            )}; color: white;"
                        >
                            {pricingRecommendation.customerRegime}
                        </span>
                    </div>
                    <div class="metrics">
                        <div class="metric">
                            <span class="lbl">Margin</span>
                            <span class="val"
                                >{(
                                    pricingRecommendation.suggestedMargin * 100
                                ).toFixed(1)}%</span
                            >
                        </div>
                        <div class="metric">
                            <span class="lbl">Win Rate</span>
                            <span class="val"
                                >{(
                                    pricingRecommendation.winProbability * 100
                                ).toFixed(0)}%</span
                            >
                        </div>
                    </div>
                </div>
            {/if}

            {#if pdfResult}
                <div class="panel success-panel" in:fade={{ duration: motionMs(400) }}>
                    <h3>PDF Ready</h3>
                    <p class="detail">
                        {(pdfResult.pdfSize / 1024).toFixed(1)} KB • {pdfResult.processingTimeMs}ms
                    </p>
                    <button class="btn-secondary" onclick={openPdf}
                        >Open PDF</button
                    >
                </div>
            {/if}
        </aside>

        <!-- Preview -->
        <main class="main-content">
            <div class="preview-container">
                {#if previewHtml}
                    <iframe
                        srcdoc={previewHtml}
                        title="Preview"
                        sandbox="allow-same-origin"
                    ></iframe>
                {:else}
                    <div class="empty-preview">
                        <span
                            >Select an Excel file to see quotation preview</span
                        >
                    </div>
                {/if}
            </div>
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

    .status-badge {
        font-family: var(--font-mono);
        font-size: 10px;
        background: #ecfdf5;
        color: #047857;
        padding: 4px 8px;
        border-radius: 4px;
    }

    .layout-split {
        display: grid;
        grid-template-columns: 350px 1fr;
        gap: var(--space-8);
        flex: 1;
        min-height: 0;
    }

    .sidebar {
        display: flex;
        flex-direction: column;
        gap: var(--space-6);
        overflow-y: auto;
    }

    .panel {
        background: var(--paper-subtle);
        padding: var(--space-5);
        border-radius: var(--radius-lg);
        border: 1px solid var(--border-subtle);
        display: flex;
        flex-direction: column;
        gap: var(--space-3);
    }
    .panel h3 {
        font-size: 11px;
        text-transform: uppercase;
        color: var(--ink-light);
        margin: 0 0 4px;
        letter-spacing: 1px;
    }

    .file-group {
        display: flex;
        gap: var(--space-2);
    }
    input {
        flex: 1;
        padding: 8px;
        border: 1px solid var(--border-medium);
        border-radius: var(--radius-md);
        font-size: 12px;
        background: var(--paper);
    }

    .actions {
        display: flex;
        gap: var(--space-2);
        margin-top: var(--space-2);
    }

    .btn-primary {
        background: var(--ink);
        color: var(--paper);
        border: none;
        padding: 8px 16px;
        border-radius: var(--radius-pill);
        cursor: pointer;
        flex: 1;
    }
    .btn-ghost {
        background: transparent;
        border: 1px solid var(--border-medium);
        padding: 8px 16px;
        border-radius: var(--radius-pill);
        cursor: pointer;
    }
    .btn-secondary {
        background: var(--paper);
        border: 1px solid var(--border-medium);
        padding: 8px;
        border-radius: var(--radius-md);
        cursor: pointer;
        width: 100%;
    }
    button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .summary-panel .row {
        display: flex;
        justify-content: space-between;
        font-size: 13px;
        padding: 2px 0;
    }
    .summary-panel .total {
        border-top: 1px solid var(--border-medium);
        margin-top: 8px;
        padding-top: 8px;
        font-weight: 600;
        font-size: 15px;
    }

    .pricing-head {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    .badge {
        padding: 2px 8px;
        border-radius: 4px;
        font-size: 10px;
        font-weight: 600;
        text-transform: uppercase;
    }
    .metrics {
        display: flex;
        gap: var(--space-4);
        margin-top: 8px;
    }
    .metric {
        display: flex;
        flex-direction: column;
        align-items: center;
        background: var(--paper);
        padding: 8px;
        border-radius: 4px;
        flex: 1;
    }
    .metric .lbl {
        font-size: 9px;
        text-transform: uppercase;
        color: var(--ink-light);
    }
    .metric .val {
        font-size: 16px;
        font-weight: 600;
        color: var(--ink);
    }

    .success-panel {
        background: #f0fdf4;
        border-color: #bbf7d0;
    }
    .success-panel h3 {
        color: #166534;
    }
    .detail {
        font-size: 11px;
        color: #166534;
        margin: 0;
    }

    .main-content {
        background: var(--paper-subtle);
        border-radius: var(--radius-xl);
        border: 1px solid var(--border-subtle);
        overflow: hidden;
        display: flex;
    }
    .preview-container {
        flex: 1;
        display: flex;
        flex-direction: column;
    }
    iframe {
        width: 100%;
        height: 100%;
        border: none;
        background: white;
    }

    .empty-preview {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--ink-light);
        font-style: italic;
    }
    .mono {
        font-family: var(--font-mono);
    }
</style>
