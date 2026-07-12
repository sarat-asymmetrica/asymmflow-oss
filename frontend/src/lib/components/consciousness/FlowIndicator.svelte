<script lang="ts">
    let { flowState = {
        state: "In Flow", // "In Flow" | "Building" | "Distracted"
        challenge: 0.75,
        skill: 0.80,
        recommendations: []
    } } = $props();

    let state = $derived(flowState.state || "Building");
    let inFlow = $derived(state === "In Flow");
    let building = $derived(state === "Building");
    let distracted = $derived(state === "Distracted");

    let challenge = $derived(flowState.challenge || 0.5);
    let skill = $derived(flowState.skill || 0.5);
    let balance = $derived(Math.abs(challenge - skill));
    let recommendations = $derived(flowState.recommendations || []);

    // State colors
    function getStateColor() {
        if (inFlow) return '#15803d'; // green
        if (building) return '#fbbf24'; // yellow
        return '#ef4444'; // red
    }

    // State icon
    function getStateIcon() {
        return ''; // Removed emojis
    }
</script>

<div class="flow-indicator" class:in-flow={inFlow}>
    <div class="flow-header">
        <span class="flow-icon">{getStateIcon()}</span>
        <span class="flow-state" style="color: {getStateColor()}">{state}</span>
    </div>

    <!-- Pulsing circle when in flow -->
    {#if inFlow}
        <div class="pulse-container">
            <div class="pulse-ring"></div>
            <div class="pulse-core" style="background-color: {getStateColor()}"></div>
        </div>
    {/if}

    <!-- Challenge/Skill balance -->
    <div class="balance-section">
        <div class="balance-row">
            <span class="balance-label">Challenge</span>
            <div class="balance-bar">
                <div class="balance-fill" style="width: {challenge * 100}%; background-color: #fbbf24"></div>
            </div>
        </div>
        <div class="balance-row">
            <span class="balance-label">Skill</span>
            <div class="balance-bar">
                <div class="balance-fill" style="width: {skill * 100}%; background-color: #15803d"></div>
            </div>
        </div>
        {#if balance > 0.2}
            <div class="balance-warning">
                <span class="warning-icon"></span>
                <span class="warning-text">Balance: {(balance * 100).toFixed(0)}% off</span>
            </div>
        {/if}
    </div>

    <!-- Recommendations when not in flow -->
    {#if !inFlow && recommendations.length > 0}
        <div class="recommendations">
            <div class="rec-header">Suggestions:</div>
            <ul class="rec-list">
                {#each recommendations as rec}
                    <li class="rec-item">{rec}</li>
                {/each}
            </ul>
        </div>
    {/if}
</div>

<style>
    .flow-indicator {
        background: rgba(255, 255, 255, 0.5);
        border: 1px solid rgba(0, 0, 0, 0.1);
        border-radius: 8px;
        padding: 0.75rem;
        font-size: 0.85rem;
    }

    .flow-indicator.in-flow {
        background: linear-gradient(135deg, rgba(21, 128, 61, 0.05), rgba(255, 255, 255, 0.5));
        border-color: rgba(21, 128, 61, 0.2);
    }

    .flow-header {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        margin-bottom: 0.75rem;
    }

    .flow-icon {
        font-size: 1.2rem;
    }

    .flow-state {
        font-family: var(--font-serif), Georgia, serif;
        font-weight: 600;
        font-size: 0.95rem;
    }

    /* Pulsing animation */
    .pulse-container {
        position: relative;
        width: 32px;
        height: 32px;
        margin: 0.5rem auto;
    }

    .pulse-ring {
        position: absolute;
        width: 32px;
        height: 32px;
        border-radius: 50%;
        border: 2px solid #15803d;
        opacity: 0;
        animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
    }

    .pulse-core {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        width: 12px;
        height: 12px;
        border-radius: 50%;
    }

    @keyframes pulse {
        0% {
            transform: scale(0.5);
            opacity: 1;
        }
        100% {
            transform: scale(1.5);
            opacity: 0;
        }
    }

    /* Balance section */
    .balance-section {
        margin-top: 0.75rem;
        padding-top: 0.75rem;
        border-top: 1px solid rgba(0, 0, 0, 0.08);
    }

    .balance-row {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        margin-bottom: 0.5rem;
    }

    .balance-label {
        font-family: var(--font-mono), 'Courier New', monospace;
        font-size: 0.65rem;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--color-ink-light, #57534e);
        width: 60px;
        flex-shrink: 0;
    }

    .balance-bar {
        flex: 1;
        height: 6px;
        background: rgba(0, 0, 0, 0.08);
        border-radius: 3px;
        overflow: hidden;
    }

    .balance-fill {
        height: 100%;
        transition: width 0.6s cubic-bezier(0.4, 0, 0.2, 1);
        border-radius: 3px;
    }

    .balance-warning {
        display: flex;
        align-items: center;
        gap: 0.4rem;
        margin-top: 0.5rem;
        padding: 0.4rem;
        background: rgba(251, 191, 36, 0.1);
        border-radius: 4px;
        font-size: 0.75rem;
    }

    .warning-icon {
        font-size: 0.9rem;
    }

    .warning-text {
        font-family: var(--font-mono), 'Courier New', monospace;
        color: var(--color-ink-light, #57534e);
    }

    /* Recommendations */
    .recommendations {
        margin-top: 0.75rem;
        padding-top: 0.75rem;
        border-top: 1px solid rgba(0, 0, 0, 0.08);
    }

    .rec-header {
        font-family: var(--font-mono), 'Courier New', monospace;
        font-size: 0.65rem;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--color-ink-light, #57534e);
        margin-bottom: 0.5rem;
    }

    .rec-list {
        list-style: none;
        padding: 0;
        margin: 0;
    }

    .rec-item {
        font-family: var(--font-serif), Georgia, serif;
        font-size: 0.75rem;
        color: var(--color-ink, #1c1c1c);
        padding: 0.25rem 0;
        padding-left: 1rem;
        position: relative;
    }

    .rec-item::before {
        content: '→';
        position: absolute;
        left: 0;
        color: var(--color-ink-light, #57534e);
    }
</style>
