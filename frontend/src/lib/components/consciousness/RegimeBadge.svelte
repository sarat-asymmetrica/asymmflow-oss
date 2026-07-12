<script lang="ts">
    

    interface Props {
        /**
     * RegimeBadge - Three-Regime Dynamics Visualizer
     *
     * Displays R1/R2/R3 breakdown with color coding:
     * - R1 (Exploration): Red - Risky, high variance
     * - R2 (Optimization): Amber - Working, moderate
     * - R3 (Stabilization): Green - Stable, safe
     *
     * Used for:
     * - Customer payment regime display
     * - System health regime indicator
     * - Any three-regime classification
     */
        r1?: number; // Exploration regime (0-1)
        r2?: number; // Optimization regime (0-1)
        r3?: number; // Stabilization regime (0-1)
        size?: string; // "small" | "normal" | "large"
        showPercentages?: boolean;
        showLabels?: boolean;
        dominant?: string; // Auto-calculate if not provided
    }

    let {
        r1 = 0.1,
        r2 = 0.2,
        r3 = 0.7,
        size = "normal",
        showPercentages = true,
        showLabels = false,
        dominant = ""
    }: Props = $props();


    // Calculate dominant regime
    function calculateDominant(r1Val, r2Val, r3Val) {
        if (r1Val > r2Val && r1Val > r3Val) return "R1";
        if (r2Val > r1Val && r2Val > r3Val) return "R2";
        return "R3";
    }

    // Get color for regime
    function getRegimeColor(regime) {
        const colors = {
            R1: "#ef4444", // red-500 - Risky
            R2: "#f59e0b", // amber-500 - Moderate
            R3: "#15803d"  // green-700 - Stable
        };
        return colors[regime] || colors.R3;
    }

    // Get background color (lighter version)
    function getRegimeBgColor(regime) {
        const colors = {
            R1: "rgba(239, 68, 68, 0.1)",   // red with 10% opacity
            R2: "rgba(245, 158, 11, 0.1)",  // amber with 10% opacity
            R3: "rgba(21, 128, 61, 0.1)"    // green with 10% opacity
        };
        return colors[regime] || colors.R3;
    }

    // Get icon for regime
    function getRegimeIcon(regime) {
        const icons = {
            R1: "",  // Warning - risky
            R2: "",  // Lightning - working
            R3: ""   // Check - stable
        };
        return icons[regime] || icons.R3;
    }

    // Get state name
    function getStateName(regime) {
        const names = {
            R1: "Exploration",
            R2: "Optimization",
            R3: "Stabilization"
        };
        return names[regime] || names.R3;
    }

    // Auto-calculate dominant regime if not provided
    let dominantRegime = $derived(dominant || calculateDominant(r1, r2, r3));
    // Size classes
    const sizeClassMap = {
        small: "text-xs px-2 py-1",
        normal: "text-sm px-3 py-1.5",
        large: "text-base px-4 py-2"
    };
    let sizeClasses = $derived(sizeClassMap[size] || sizeClassMap.normal);
</script>

<div
    class="regime-badge {sizeClasses}"
    style="background-color: {getRegimeBgColor(dominantRegime)}; border-color: {getRegimeColor(dominantRegime)}"
    title="R1: {(r1*100).toFixed(0)}% | R2: {(r2*100).toFixed(0)}% | R3: {(r3*100).toFixed(0)}%"
>
    <span class="regime-icon">{getRegimeIcon(dominantRegime)}</span>
    <span class="regime-label" style="color: {getRegimeColor(dominantRegime)}">
        {dominantRegime}
        {#if showLabels}
            <span class="regime-state">({getStateName(dominantRegime)})</span>
        {/if}
    </span>

    {#if showPercentages}
        <div class="regime-breakdown">
            <div class="regime-bar">
                <div
                    class="regime-segment r1"
                    style="width: {r1 * 100}%"
                    title="R1 (Exploration): {(r1*100).toFixed(0)}%"
                ></div>
                <div
                    class="regime-segment r2"
                    style="width: {r2 * 100}%"
                    title="R2 (Optimization): {(r2*100).toFixed(0)}%"
                ></div>
                <div
                    class="regime-segment r3"
                    style="width: {r3 * 100}%"
                    title="R3 (Stabilization): {(r3*100).toFixed(0)}%"
                ></div>
            </div>
            <div class="regime-values">
                <span class="regime-value r1-text">{(r1*100).toFixed(0)}%</span>
                <span class="regime-value r2-text">{(r2*100).toFixed(0)}%</span>
                <span class="regime-value r3-text">{(r3*100).toFixed(0)}%</span>
            </div>
        </div>
    {/if}
</div>

<style>
    .regime-badge {
        display: inline-flex;
        flex-direction: column;
        gap: 0.5rem;
        border: 1px solid;
        border-radius: 6px;
        font-family: var(--font-mono, 'Courier Prime', monospace);
        transition: all 0.2s ease;
    }

    .regime-badge:hover {
        transform: translateY(-1px);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }

    .regime-icon {
        font-size: 1rem;
        display: inline-block;
        margin-right: 0.25rem;
    }

    .regime-label {
        font-weight: 600;
        letter-spacing: 0.5px;
        display: flex;
        align-items: center;
        gap: 0.25rem;
    }

    .regime-state {
        font-size: 0.75em;
        opacity: 0.8;
        font-weight: normal;
    }

    /* Regime breakdown visualization */
    .regime-breakdown {
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
    }

    .regime-bar {
        display: flex;
        height: 6px;
        width: 100%;
        border-radius: 3px;
        overflow: hidden;
        background: rgba(0, 0, 0, 0.05);
    }

    .regime-segment {
        height: 100%;
        transition: width 0.6s cubic-bezier(0.4, 0, 0.2, 1);
    }

    .regime-segment.r1 {
        background: linear-gradient(90deg, #ef4444, #dc2626);
    }

    .regime-segment.r2 {
        background: linear-gradient(90deg, #f59e0b, #d97706);
    }

    .regime-segment.r3 {
        background: linear-gradient(90deg, #15803d, #166534);
    }

    .regime-values {
        display: flex;
        justify-content: space-between;
        font-size: 0.65rem;
        letter-spacing: 0.3px;
    }

    .regime-value {
        opacity: 0.7;
    }

    .r1-text { color: #ef4444; }
    .r2-text { color: #f59e0b; }
    .r3-text { color: #15803d; }

    /* Responsive */
    @media (max-width: 640px) {
        .regime-badge {
            font-size: 0.75rem;
        }

        .regime-breakdown {
            display: none;
        }
    }
</style>
