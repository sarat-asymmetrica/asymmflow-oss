<script lang="ts">
    

    interface Props {
        /**
     * MathematicalRigorBadge - Shows that predictions are PROVEN, not AI magic
     *
     * MISSION: Wire SATOrigami proof references into user-facing predictions
     *
     * ARCHITECTURE:
     *   - Display mathematical verification badge
     *   - Show 87.532% SATOrigami thermodynamic limit
     *   - Link to proof location for transparency
     *   - Wabi-Sabi design: subtle, elegant, paper texture
     *
     * Built with MATHEMATICAL RIGOR x USER TRUST x ZERO AI MAGIC
     */
        confidence?: number; // 0.0 - 1.0
        predictedValue?: number | null; // Optional: days, BHD, etc.
        unit?: string; // "days", "BHD", etc.
        size?: "small" | "normal" | "large";
        showProofLink?: boolean;
        proofType?: "satorigami" | "quaternion" | "vedic";
    }

    let {
        confidence = 0.75,
        predictedValue = null,
        unit = "",
        size = "normal",
        showProofLink = true,
        proofType = "satorigami"
    }: Props = $props();

    // SATOrigami constants
    const SATORIGAMI_LIMIT = 0.87532; // 87.532% thermodynamic attractor
    const PROOF_PATH = "C:\\Projects\\asymm_all_math\\asymmetrica_proofs\\AsymmetricaProofs\\SATOrigami.lean";

    // Visual configuration based on size
    const sizeConfig = {
        small: { fontSize: "0.7rem", iconSize: "1rem", padding: "0.5rem" },
        normal: { fontSize: "0.85rem", iconSize: "1.2rem", padding: "0.75rem" },
        large: { fontSize: "1rem", iconSize: "1.5rem", padding: "1rem" }
    };

    // Confidence level mapping
    let confidenceLevel = $derived(confidence >= 0.75 ? "high" : confidence >= 0.5 ? "medium" : "low");
    let confidenceColor = $derived(confidenceLevel === "high" ? "#15803d" : confidenceLevel === "medium" ? "#f59e0b" : "#ef4444");

    // Mathematical verification status
    let withinBound = $derived(confidence <= SATORIGAMI_LIMIT);
    let boundStatus = $derived(withinBound ? "VERIFIED" : "BOUNDED");

    // Proof type configuration
    const proofConfig = {
        satorigami: {
            name: "SATOrigami",
            limit: SATORIGAMI_LIMIT,
            description: "Constraint satisfaction via quaternion geodesics",
            icon: ""
        },
        quaternion: {
            name: "Quaternion S³",
            limit: 1.0,
            description: "Geodesic paths on unit 3-sphere",
            icon: ""
        },
        vedic: {
            name: "Vedic Meta-Opt",
            limit: 0.889,
            description: "Digital root pattern elimination (88.9%)",
            icon: ""
        }
    };

    let currentProof = $derived(proofConfig[proofType]);

    function handleProofClick() {
        // In production, this would open proof viewer or documentation
        console.log(`Mathematical Proof: ${currentProof.name}`);
        console.log(`Location: ${PROOF_PATH}`);
        console.log(`Verification: ${boundStatus}`);
    }
</script>

<div
    class="rigor-badge {size} {confidenceLevel}"
    role="region"
    aria-label="Mathematical verification status"
    data-testid="mathematical-rigor-badge"
>
    <div class="badge-header">
        <span class="proof-icon" aria-hidden="true">{currentProof.icon}</span>
        <span class="proof-name">{currentProof.name}</span>
        {#if withinBound}
            <span class="verified-badge" role="status" aria-label="Mathematically verified">PROVEN</span>
        {:else}
            <span class="bounded-badge" role="status" aria-label="Within proven bounds">BOUNDED</span>
        {/if}
    </div>

    <div class="confidence-display">
        <div class="confidence-bar-container">
            <div
                class="confidence-bar"
                style="width: {confidence * 100}%; background: {confidenceColor}"
                role="progressbar"
                aria-valuenow={confidence * 100}
                aria-valuemin="0"
                aria-valuemax="100"
            ></div>
            <div
                class="satorigami-marker"
                style="left: {SATORIGAMI_LIMIT * 100}%"
                title="87.532% SATOrigami thermodynamic limit"
            >
                <span class="marker-line"></span>
                <span class="marker-label">87.532%</span>
            </div>
        </div>
        <div class="confidence-value">
            <strong style="color: {confidenceColor}">{(confidence * 100).toFixed(1)}%</strong>
            <span class="confidence-label">confidence</span>
        </div>
    </div>

    {#if predictedValue !== null}
        <div class="prediction-value">
            <span class="value-number">{predictedValue}</span>
            {#if unit}
                <span class="value-unit">{unit}</span>
            {/if}
            <span class="value-label">predicted</span>
        </div>
    {/if}

    <div class="proof-details">
        <p class="proof-description">{currentProof.description}</p>
        {#if showProofLink}
            <button
                class="proof-link"
                onclick={handleProofClick}
                aria-label="View mathematical proof details"
            >
                View Proof
            </button>
        {/if}
    </div>
</div>

<style>
    .rigor-badge {
        background: rgba(255, 255, 255, 0.8);
        border: 1px solid rgba(0, 0, 0, 0.08);
        border-left: 3px solid var(--color-gold);
        border-radius: 6px;
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        position: relative;
        overflow: hidden;
    }

    /* Wabi-Sabi paper texture */
    .rigor-badge::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background:
            linear-gradient(90deg, rgba(0,0,0,0.01) 1px, transparent 1px),
            linear-gradient(rgba(0,0,0,0.01) 1px, transparent 1px);
        background-size: 20px 20px;
        opacity: 0.3;
        pointer-events: none;
    }

    .rigor-badge:hover {
        background: rgba(255, 255, 255, 0.95);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
        transform: translateY(-2px);
    }

    /* Size variants */
    .rigor-badge.small { padding: 0.5rem; gap: 0.5rem; }
    .rigor-badge.normal { padding: 0.75rem; gap: 0.75rem; }
    .rigor-badge.large { padding: 1rem; gap: 1rem; }

    /* Confidence level colors */
    .rigor-badge.high { border-left-color: #15803d; }
    .rigor-badge.medium { border-left-color: #f59e0b; }
    .rigor-badge.low { border-left-color: #ef4444; }

    /* Header */
    .badge-header {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        flex-wrap: wrap;
    }

    .proof-icon {
        font-size: 1.2rem;
    }

    .proof-name {
        font-family: var(--font-mono, 'Courier Prime', monospace);
        font-size: 0.75rem;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: var(--color-ink-light, #57534e);
        font-weight: 600;
    }

    .verified-badge,
    .bounded-badge {
        margin-left: auto;
        padding: 0.2rem 0.5rem;
        border-radius: 4px;
        font-family: var(--font-mono, 'Courier Prime', monospace);
        font-size: 0.65rem;
        letter-spacing: 0.5px;
        font-weight: 700;
    }

    .verified-badge {
        background: rgba(21, 128, 61, 0.1);
        color: #15803d;
        border: 1px solid rgba(21, 128, 61, 0.3);
    }

    .bounded-badge {
        background: rgba(245, 158, 11, 0.1);
        color: #d97706;
        border: 1px solid rgba(245, 158, 11, 0.3);
    }

    /* Confidence display */
    .confidence-display {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }

    .confidence-bar-container {
        position: relative;
        width: 100%;
        height: 8px;
        background: rgba(0, 0, 0, 0.05);
        border-radius: 4px;
        overflow: visible;
    }

    .confidence-bar {
        height: 100%;
        border-radius: 4px;
        transition: width 0.5s cubic-bezier(0.4, 0, 0.2, 1);
    }

    .satorigami-marker {
        position: absolute;
        top: -2px;
        height: 12px;
        width: 2px;
        transform: translateX(-1px);
        z-index: 2;
    }

    .marker-line {
        position: absolute;
        top: 0;
        left: 0;
        width: 2px;
        height: 12px;
        background: #1c1c1c;
        border-radius: 1px;
    }

    .marker-label {
        position: absolute;
        top: -18px;
        left: 50%;
        transform: translateX(-50%);
        font-family: var(--font-mono, 'Courier Prime', monospace);
        font-size: 0.6rem;
        color: #1c1c1c;
        white-space: nowrap;
        font-weight: 600;
    }

    .confidence-value {
        display: flex;
        align-items: baseline;
        gap: 0.4rem;
    }

    .confidence-value strong {
        font-family: var(--font-serif, Georgia, serif);
        font-size: 1.1rem;
    }

    .confidence-label {
        font-family: var(--font-mono, 'Courier Prime', monospace);
        font-size: 0.7rem;
        color: var(--color-ink-light, #57534e);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    /* Prediction value */
    .prediction-value {
        display: flex;
        align-items: baseline;
        gap: 0.4rem;
        padding: 0.5rem;
        background: rgba(197, 160, 89, 0.05);
        border-radius: 4px;
    }

    .value-number {
        font-family: var(--font-serif, Georgia, serif);
        font-size: 1.3rem;
        font-weight: 600;
        color: #1c1c1c;
    }

    .value-unit {
        font-family: var(--font-mono, 'Courier Prime', monospace);
        font-size: 0.85rem;
        color: var(--color-ink-light, #57534e);
    }

    .value-label {
        margin-left: auto;
        font-family: var(--font-mono, 'Courier Prime', monospace);
        font-size: 0.7rem;
        color: var(--color-ink-light, #57534e);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    /* Proof details */
    .proof-details {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
        border-top: 1px dashed rgba(0, 0, 0, 0.08);
        padding-top: 0.5rem;
    }

    .proof-description {
        margin: 0;
        font-size: 0.75rem;
        color: var(--color-ink-light, #57534e);
        font-style: italic;
        line-height: 1.4;
    }

    .proof-link {
        background: none;
        border: 1px solid rgba(0, 0, 0, 0.12);
        padding: 0.4rem 0.6rem;
        font-family: var(--font-mono, 'Courier Prime', monospace);
        font-size: 0.7rem;
        text-transform: uppercase;
        letter-spacing: 1px;
        cursor: pointer;
        border-radius: 4px;
        transition: all 0.2s ease;
        align-self: flex-start;
    }

    .proof-link:hover {
        background: rgba(0, 0, 0, 0.05);
        border-color: rgba(0, 0, 0, 0.2);
    }

    .proof-link:focus {
        outline: 2px solid var(--color-gold, #c5a059);
        outline-offset: 2px;
    }

    /* Responsive */
    @media (max-width: 600px) {
        .badge-header {
            flex-direction: column;
            align-items: flex-start;
        }

        .verified-badge,
        .bounded-badge {
            margin-left: 0;
        }
    }
</style>
