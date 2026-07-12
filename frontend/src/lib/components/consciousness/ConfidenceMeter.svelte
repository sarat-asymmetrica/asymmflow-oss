<script lang="ts">
    interface ConfidenceBreakdown {
        completeness: number;
        reliability: number;
        relevance: number;
        consistency: number;
        historical: number;
    }

    interface ConfidenceData {
        score: number;
        breakdown: ConfidenceBreakdown;
    }

    interface Props {
        confidence?: ConfidenceData;
    }

    let { confidence = {
        score: 0.85,
        breakdown: {
            completeness: 0.90,
            reliability: 0.85,
            relevance: 0.88,
            consistency: 0.80,
            historical: 0.82
        }
    } }: Props = $props();

    let totalScore = $derived(confidence.score || 0);
    let breakdown = $derived(confidence.breakdown || {} as ConfidenceBreakdown);

    // Color gradient based on score
    function getColor(score: number): string {
        if (score < 0.3) return '#ef4444'; // red
        if (score < 0.7) return '#fbbf24'; // yellow/gold
        return '#15803d'; // green
    }

    function formatPercentage(value: number): string {
        return (value * 100).toFixed(1);
    }
</script>

<div class="confidence-meter">
    <div class="meter-header">
        <span class="meter-label">Confidence</span>
        <span class="meter-score" style="color: {getColor(totalScore)}">
            {formatPercentage(totalScore)}%
        </span>
    </div>

    <!-- Visual meter bar -->
    <div class="meter-bar-container">
        <div
            class="meter-bar-fill"
            style="width: {totalScore * 100}%; background-color: {getColor(totalScore)}"
        ></div>
    </div>

    <!-- Breakdown (expandable) -->
    <details class="breakdown-details">
        <summary class="breakdown-toggle">View breakdown</summary>
        <div class="breakdown-grid">
            {#if breakdown.completeness !== undefined}
                <div class="breakdown-item">
                    <span class="breakdown-label">Completeness</span>
                    <span class="breakdown-value">{formatPercentage(breakdown.completeness)}%</span>
                </div>
            {/if}
            {#if breakdown.reliability !== undefined}
                <div class="breakdown-item">
                    <span class="breakdown-label">Reliability</span>
                    <span class="breakdown-value">{formatPercentage(breakdown.reliability)}%</span>
                </div>
            {/if}
            {#if breakdown.relevance !== undefined}
                <div class="breakdown-item">
                    <span class="breakdown-label">Relevance</span>
                    <span class="breakdown-value">{formatPercentage(breakdown.relevance)}%</span>
                </div>
            {/if}
            {#if breakdown.consistency !== undefined}
                <div class="breakdown-item">
                    <span class="breakdown-label">Consistency</span>
                    <span class="breakdown-value">{formatPercentage(breakdown.consistency)}%</span>
                </div>
            {/if}
            {#if breakdown.historical !== undefined}
                <div class="breakdown-item">
                    <span class="breakdown-label">Historical</span>
                    <span class="breakdown-value">{formatPercentage(breakdown.historical)}%</span>
                </div>
            {/if}
        </div>
    </details>
</div>

<style>
    .confidence-meter {
        background: rgba(255, 255, 255, 0.5);
        border: 1px solid rgba(0, 0, 0, 0.1);
        border-radius: 8px;
        padding: 0.75rem;
        margin-top: 0.5rem;
        font-size: 0.9rem;
    }

    .meter-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 0.5rem;
    }

    .meter-label {
        font-family: var(--font-mono), 'Courier New', monospace;
        font-size: 0.7rem;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: var(--color-ink-light, #57534e);
    }

    .meter-score {
        font-family: var(--font-serif), Georgia, serif;
        font-size: 1.1rem;
        font-weight: 600;
    }

    .meter-bar-container {
        width: 100%;
        height: 8px;
        background: rgba(0, 0, 0, 0.08);
        border-radius: 4px;
        overflow: hidden;
        margin-bottom: 0.75rem;
    }

    .meter-bar-fill {
        height: 100%;
        transition: width 0.6s cubic-bezier(0.4, 0, 0.2, 1), background-color 0.3s ease;
        border-radius: 4px;
    }

    .breakdown-details {
        margin-top: 0.5rem;
        border-top: 1px solid rgba(0, 0, 0, 0.08);
        padding-top: 0.5rem;
    }

    .breakdown-toggle {
        cursor: pointer;
        font-family: var(--font-mono), 'Courier New', monospace;
        font-size: 0.65rem;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--color-ink-light, #57534e);
        user-select: none;
    }

    .breakdown-toggle:hover {
        color: var(--color-ink, #1c1c1c);
    }

    .breakdown-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 0.5rem;
        margin-top: 0.5rem;
    }

    .breakdown-item {
        display: flex;
        justify-content: space-between;
        padding: 0.25rem 0;
        font-size: 0.8rem;
    }

    .breakdown-label {
        color: var(--color-ink-light, #57534e);
        font-family: var(--font-serif), Georgia, serif;
    }

    .breakdown-value {
        font-family: var(--font-mono), 'Courier New', monospace;
        color: var(--color-ink, #1c1c1c);
        font-weight: 500;
    }

    /* Compact on mobile */
    @media (max-width: 640px) {
        .breakdown-grid {
            grid-template-columns: 1fr;
        }
    }
</style>
