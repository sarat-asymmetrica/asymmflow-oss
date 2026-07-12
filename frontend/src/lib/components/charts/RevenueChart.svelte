<script lang="ts">
    import { fade } from "svelte/transition";
    import type { RevenueChartData } from "$lib/types";

    interface Props {
        data?: RevenueChartData[];
        loading?: boolean;
    }

    let { data = [], loading = false }: Props = $props();

    let hoveredBar: number | null = $state(null);

    // Reactive calculations
    let topCustomers = $derived(data.sort((a, b) => b.revenue - a.revenue).slice(0, 10));

    let maxRevenue = $derived(Math.max(...topCustomers.map((d) => d.revenue), 1));
    let maxInvoices = $derived(Math.max(...topCustomers.map((d) => d.invoice_count), 1));

    // Chart dimensions
    const width = 1000;
    const height = 350;
    const padding = { top: 20, right: 60, bottom: 100, left: 60 };
    const chartWidth = width - padding.left - padding.right;
    const chartHeight = height - padding.top - padding.bottom;

    function getBarHeight(revenue: number): number {
        return (revenue / maxRevenue) * chartHeight;
    }

    function getLineY(invoiceCount: number): number {
        return (
            height - padding.bottom - (invoiceCount / maxInvoices) * chartHeight
        );
    }

    function getBarX(index: number): number {
        const barSpacing = chartWidth / topCustomers.length;
        return padding.left + index * barSpacing + barSpacing / 2;
    }

    function getBarWidth(): number {
        const barSpacing = chartWidth / topCustomers.length;
        return Math.min(barSpacing * 0.7, 70);
    }

    function formatCurrency(val: number): string {
        if (val >= 1000000) return (val / 1000000).toFixed(1) + "M";
        if (val >= 1000) return (val / 1000).toFixed(0) + "k";
        return val.toLocaleString();
    }

    function formatNumber(val: number): string {
        return val.toFixed(0);
    }
</script>

<div class="chart-wrapper">
    {#if loading}
        <div class="chart-loading">
            <div class="spinner"></div>
            <p>Loading chart data...</p>
        </div>
    {:else if topCustomers.length === 0}
        <div class="chart-empty">
            <p>No revenue data available</p>
        </div>
    {:else}
        <svg
            class="revenue-chart"
            viewBox="0 0 {width} {height}"
            preserveAspectRatio="xMidYMid meet"
            in:fade
        >
            <!-- Grid lines -->
            <g class="grid">
                {#each [0, 0.25, 0.5, 0.75, 1] as tick}
                    {@const y = height - padding.bottom - tick * chartHeight}
                    <line
                        x1={padding.left}
                        y1={y}
                        x2={width - padding.right}
                        y2={y}
                        stroke="rgba(0,0,0,0.08)"
                        stroke-width="1"
                    />
                    <text
                        x={padding.left - 10}
                        y={y + 4}
                        text-anchor="end"
                        class="axis-label"
                    >
                        {formatCurrency(maxRevenue * tick)}
                    </text>
                {/each}
            </g>

            <!-- Hover Tooltip Container (Unified) -->
            {#if hoveredBar !== null}
                {@const i = hoveredBar}
                {@const customer = topCustomers[i]}
                {@const barX = getBarX(i)}
                {@const barHeight = getBarHeight(customer.revenue)}
                {@const pointY = getLineY(customer.invoice_count)}
                {@const barTop = height - padding.bottom - barHeight}
                
                <!-- Calculate tooltip position (highest point of bar or line) -->
                {@const tipY = Math.min(barTop, pointY) - 10}
                
                <g class="tooltip" transform="translate({barX}, {tipY})">
                    <!-- Tooltip Background -->
                    <rect x="-60" y="-55" width="120" height="50" rx="6" fill="rgba(255,255,255,0.95)" stroke="#e5e5e5" filter="drop-shadow(0 4px 6px rgba(0,0,0,0.1))" />
                    
                    <!-- Revenue Text -->
                    <text x="0" y="-35" text-anchor="middle" font-size="12" font-weight="600" fill="#2A7FFF">
                        {formatCurrency(customer.revenue)}
                    </text>
                    
                    <!-- Invoice Count Text -->
                    <text x="0" y="-15" text-anchor="middle" font-size="11" fill="#F59E0B">
                        {customer.invoice_count} Invoices
                    </text>
                    
                    <!-- Connector Triangle -->
                    <path d="M -6 -5 L 0 0 L 6 -5 Z" fill="rgba(255,255,255,0.95)" />
                </g>
            {/if}

            <!-- Bars -->
            <g class="bars">
                {#each topCustomers as customer, i}
                    {@const barHeight = getBarHeight(customer.revenue)}
                    {@const barX = getBarX(i)}
                    {@const barWidth = getBarWidth()}
                    <rect
                        x={barX - barWidth / 2}
                        y={height - padding.bottom - barHeight}
                        width={barWidth}
                        height={barHeight}
                        role="presentation"
                        class="bar"
                        class:hovered={hoveredBar === i}
                        onmouseenter={() => (hoveredBar = i)}
                        onmouseleave={() => (hoveredBar = null)}
                    />
                {/each}
            </g>

            <!-- Line chart for invoice count -->
            <g class="line-chart">
                <polyline
                    points={topCustomers
                        .map((c, i) => {
                            const x = getBarX(i);
                            const y = getLineY(c.invoice_count);
                            return `${x},${y}`;
                        })
                        .join(" ")}
                    fill="none"
                    stroke="#F59E0B"
                    stroke-width="3"
                    class="line"
                    style="pointer-events: none;" 
                />

                <!-- Line points -->
                {#each topCustomers as customer, i}
                    {@const pointX = getBarX(i)}
                    {@const pointY = getLineY(customer.invoice_count)}
                    <circle
                        cx={pointX}
                        cy={pointY}
                        r="5"
                        role="presentation"
                        class="line-point"
                        class:hovered={hoveredBar === i}
                        onmouseenter={() => (hoveredBar = i)}
                    />
                {/each}
            </g>

            <!-- X-axis labels -->
            <g class="x-axis">
                {#each topCustomers as customer, i}
                    {@const x = getBarX(i)}
                    <text
                        {x}
                        y={height - padding.bottom + 20}
                        text-anchor="end"
                        transform="rotate(-45, {x}, {height -
                            padding.bottom +
                            20})"
                        class="customer-label"
                        class:hovered={hoveredBar === i}
                    >
                        {customer.customer_name.length > 25
                            ? customer.customer_name.substring(0, 25) + "..."
                            : customer.customer_name}
                    </text>
                {/each}
            </g>

            <!-- Axis titles -->
            <text x={padding.left} y={padding.top - 5} class="axis-title" fill="#2A7FFF">
                Revenue (BHD)
            </text>

            <text
                x={width - padding.right}
                y={padding.top - 5}
                text-anchor="end"
                class="axis-title"
                fill="#F59E0B"
            >
                Invoice Count
            </text>
        </svg>

        <!-- Legend -->
        <div class="legend">
            <div class="legend-item">
                <div class="legend-color bar-color"></div>
                <span>Revenue (BHD)</span>
            </div>
            <div class="legend-item">
                <div class="legend-color line-color"></div>
                <span>Invoice Count</span>
            </div>
        </div>
    {/if}
</div>

<style>
    .chart-wrapper {
        position: relative;
        width: 100%;
        min-height: 350px;
        background: transparent;
    }

    .chart-loading,
    .chart-empty {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 300px;
        color: var(--text-secondary, #4B5563);
        font-size: 13px;
    }

    .spinner {
        width: 32px;
        height: 32px;
        border: 3px solid rgba(0, 0, 0, 0.1);
        border-top-color: var(--primary, #2A7FFF);
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin-bottom: 12px;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    .revenue-chart {
        width: 100%;
        height: auto;
    }

    .bar {
        fill: var(--primary, #2A7FFF);
        opacity: 0.8;
        transition: all 0.2s ease;
        cursor: pointer;
    }

    .bar:hover,
    .bar.hovered {
        opacity: 1;
        filter: drop-shadow(0 4px 6px rgba(42, 127, 255, 0.3));
    }

    .line {
        transition: stroke-width 0.2s ease;
    }

    .line-point {
        fill: var(--warning, #F59E0B);
        stroke: white;
        stroke-width: 2;
        transition: all 0.2s ease;
        cursor: pointer;
    }

    .line-point:hover,
    .line-point.hovered {
        r: 7;
        filter: brightness(1.2);
    }

    .axis-label,
    .customer-label,
    .axis-title {
        font-family: var(--font-sans, system-ui, sans-serif);
        font-size: 11px;
        fill: var(--text-secondary, #4B5563);
    }

    .customer-label {
        font-size: 12px;
        transition: all 0.2s ease;
    }

    .customer-label:hover,
    .customer-label.hovered {
        fill: var(--text-primary, #111827);
        font-weight: 600;
    }

    .axis-title {
        font-size: 13px;
        font-weight: 600;
    }

    .legend {
        display: flex;
        gap: 24px;
        justify-content: center;
        margin-top: 16px;
        padding-top: 16px;
    }

    .legend-item {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 12px;
        color: var(--text-secondary, #4B5563);
    }

    .legend-color {
        width: 20px;
        height: 4px;
        border-radius: 2px;
    }

    .bar-color {
        background: var(--primary, #2A7FFF);
    }

    .line-color {
        background: var(--warning, #F59E0B);
    }
</style>
